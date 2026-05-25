package hsmc

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type JavaScriptFrontend struct{}

func NewJavaScriptFrontend() JavaScriptFrontend {
	return JavaScriptFrontend{}
}

func (JavaScriptFrontend) Language() Language { return LanguageJS }

func (frontend JavaScriptFrontend) Parse(ctx context.Context, input SourceInput) (*Program, error) {
	tree, err := parseTree(ctx, input, parserGrammarJavaScript)
	if err != nil {
		return nil, err
	}
	defer tree.Close()
	root := tree.RootNode()
	lowerer := tsLowerer{
		input:    SourceInput{Path: input.Path, Data: input.Data},
		language: LanguageJS,
		program: &Program{
			SourceLanguage: LanguageJS,
			PackageName:    "main",
		},
	}
	lowerer.collectTopLevel(root)
	lowerer.collectModels(root)
	lowerer.filterReferencedFunctionGlobals()
	if len(lowerer.diagnostics) > 0 {
		return nil, lowerer.diagnostics
	}
	if len(lowerer.program.Models) == 0 {
		return nil, fmt.Errorf("no hsm.Define model found in %s", input.Path)
	}
	return lowerer.program, nil
}

type TypeScriptFrontend struct{}

func NewTypeScriptFrontend() TypeScriptFrontend {
	return TypeScriptFrontend{}
}

func (TypeScriptFrontend) Language() Language { return LanguageTS }

func (frontend TypeScriptFrontend) Parse(ctx context.Context, input SourceInput) (*Program, error) {
	grammar := parserGrammarTypeScript
	if strings.EqualFold(filepath.Ext(input.Path), ".tsx") {
		grammar = parserGrammarTSX
	}
	tree, err := parseTree(ctx, input, grammar)
	if err != nil {
		return nil, err
	}
	defer tree.Close()
	root := tree.RootNode()
	lowerer := tsLowerer{
		input:    SourceInput{Path: input.Path, Data: input.Data},
		language: LanguageTS,
		program: &Program{
			SourceLanguage: LanguageTS,
			PackageName:    "main",
		},
	}
	lowerer.collectTopLevel(root)
	lowerer.collectModels(root)
	lowerer.filterReferencedFunctionGlobals()
	if len(lowerer.diagnostics) > 0 {
		return nil, lowerer.diagnostics
	}
	if len(lowerer.program.Models) == 0 {
		return nil, fmt.Errorf("no hsm.Define model found in %s", input.Path)
	}
	return lowerer.program, nil
}

type tsLowerer struct {
	input                   SourceInput
	language                Language
	program                 *Program
	diagnostics             Diagnostics
	counters                map[string]int
	functions               map[string]tsFunctionDecl
	usedFunctions           map[string]bool
	hsmNamespaceAliases     map[string]bool
	hsmNamedBindings        map[string]string
	hsmDefaultClockBindings map[string]bool
}

func (lowerer *tsLowerer) collectTopLevel(root *sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		child := root.NamedChild(i)
		switch child.Kind() {
		case "import_statement":
			if imp, ok := lowerer.parseImport(child); ok {
				if isHSMImportPath(imp.Path) {
					lowerer.recordHSMImportBindings(imp)
					continue
				}
				lowerer.program.Imports = append(lowerer.program.Imports, imp)
			}
		case "class_declaration", "function_declaration", "interface_declaration", "type_alias_declaration":
			block := lowerer.globalBlock(child)
			if child.Kind() == "function_declaration" {
				lowerer.registerFunctionDecl(child, block.Range)
			}
			lowerer.program.Globals = append(lowerer.program.Globals, block)
		case "lexical_declaration", "variable_declaration":
			if lowerer.collectVariableDeclaration(child, nil) {
				continue
			}
		case "expression_statement":
			if lowerer.isRuntimeDefaultClockStatement(child) {
				lowerer.program.Globals = append(lowerer.program.Globals, lowerer.globalBlock(child))
			}
		case "export_statement":
			declaration := child.ChildByFieldName("declaration")
			if declaration == nil {
				continue
			}
			switch declaration.Kind() {
			case "class_declaration", "function_declaration", "interface_declaration", "type_alias_declaration":
				block := lowerer.globalBlock(child)
				if declaration.Kind() == "function_declaration" {
					lowerer.registerFunctionDecl(declaration, block.Range)
				}
				lowerer.program.Globals = append(lowerer.program.Globals, block)
			case "lexical_declaration", "variable_declaration":
				if lowerer.collectVariableDeclaration(declaration, child) {
					continue
				}
			}
		}
	}
}

func (lowerer *tsLowerer) isRuntimeDefaultClockStatement(node *sitter.Node) bool {
	text := strings.TrimSpace(node.Utf8Text(lowerer.input.Data))
	left, _, ok := strings.Cut(text, "=")
	if !ok || strings.Contains(left, "==") || strings.Contains(left, "!=") {
		return false
	}
	left = strings.TrimSpace(left)
	if lowerer.hsmDefaultClockBindings[left] {
		return true
	}
	if qualifier, name, ok := strings.Cut(left, "."); ok && tsCanonicalName(name) == "DefaultClock" {
		return lowerer.hsmNamespaceAliases[strings.TrimSpace(qualifier)]
	}
	return false
}

func (lowerer *tsLowerer) recordHSMImportBindings(imp Import) {
	if lowerer.hsmNamespaceAliases == nil {
		lowerer.hsmNamespaceAliases = map[string]bool{}
	}
	if lowerer.hsmDefaultClockBindings == nil {
		lowerer.hsmDefaultClockBindings = map[string]bool{}
	}
	if lowerer.hsmNamedBindings == nil {
		lowerer.hsmNamedBindings = map[string]string{}
	}
	if imp.Alias != "" {
		lowerer.hsmNamespaceAliases[imp.Alias] = true
	}
	if imp.Default != "" {
		lowerer.hsmNamespaceAliases[imp.Default] = true
	}
	for _, specifier := range imp.Specifiers {
		name := specifier.Name
		if specifier.Alias != "" {
			name = specifier.Alias
		}
		canonical := tsCanonicalName(specifier.Name)
		if tsKnownDSLName(canonical) {
			lowerer.hsmNamedBindings[name] = canonical
		}
		if canonical == "DefaultClock" {
			lowerer.hsmDefaultClockBindings[name] = true
		}
	}
}

func (lowerer *tsLowerer) collectVariableDeclaration(declaration *sitter.Node, wrapper *sitter.Node) bool {
	if !lowerer.containsHSMDefine(declaration) {
		if event, ok := lowerer.parseEventDecl(declaration); ok {
			lowerer.program.Events = append(lowerer.program.Events, event)
			return true
		}
		if wrapper != nil {
			lowerer.program.Globals = append(lowerer.program.Globals, lowerer.globalBlock(wrapper))
		} else {
			lowerer.program.Globals = append(lowerer.program.Globals, lowerer.globalBlock(declaration))
		}
		return true
	}

	declarators := directVariableDeclarators(declaration)
	if len(declarators) == 0 {
		return true
	}
	prefixStart := declaration.StartByte()
	if wrapper != nil {
		prefixStart = wrapper.StartByte()
	}
	prefix := strings.TrimSpace(string(lowerer.input.Data[prefixStart:declarators[0].StartByte()]))
	for _, declarator := range declarators {
		if lowerer.containsHSMDefine(declarator) {
			continue
		}
		if event, ok := lowerer.parseEventDeclarator(declarator); ok {
			lowerer.program.Events = append(lowerer.program.Events, event)
			continue
		}
		lowerer.program.Globals = append(lowerer.program.Globals, lowerer.globalDeclaratorBlock(prefix, declarator))
	}
	return true
}

func (lowerer *tsLowerer) collectModels(root *sitter.Node) {
	walk(root, func(node *sitter.Node) {
		if node.Kind() != "call_expression" || lowerer.selectorName(node) != "Define" {
			return
		}
		model, ok := lowerer.parseModel(node)
		if ok {
			lowerer.program.Models = append(lowerer.program.Models, model)
		}
	})
}

func (lowerer *tsLowerer) parseModel(call *sitter.Node) (Model, bool) {
	args := callArgs(call)
	if len(args) == 0 {
		return Model{}, false
	}
	name, ok := stringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Model{}, false
	}
	model := Model{ID: lowerer.nextID("model"), Name: name, Range: lowerer.sourceRange(call)}
	for _, arg := range args[1:] {
		lowerer.applyModelPartial(&model, arg)
	}
	return model, true
}

func (lowerer *tsLowerer) applyModelPartial(model *Model, arg *sitter.Node) {
	call := asCall(arg)
	if call == nil {
		return
	}
	name := lowerer.selectorName(call)
	switch name {
	case "Initial":
		initial := lowerer.parseInitial(model.ID, call)
		model.Initializer = &initial
	case "State", "Final", "Choice", "ShallowHistory", "DeepHistory":
		if state, ok := lowerer.parseState(model.ID, call); ok {
			model.States = append(model.States, state)
		}
	case "Transition":
		model.Transitions = append(model.Transitions, lowerer.parseTransition(model.ID, call))
	case "Attribute":
		if attribute, ok := lowerer.parseAttribute(call); ok {
			model.Attributes = append(model.Attributes, attribute)
		}
	case "Operation":
		if operation, ok := lowerer.parseOperation(model.ID, call); ok {
			model.Operations = append(model.Operations, operation)
		}
	default:
		lowerer.addUnknownHSMPartialDiagnostic("model", call, name)
	}
}

func (lowerer *tsLowerer) parseAttribute(call *sitter.Node) (Attribute, bool) {
	args := callArgs(call)
	if len(args) == 0 {
		return Attribute{}, false
	}
	name, ok := stringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Attribute{}, false
	}
	attribute := Attribute{ID: lowerer.nextID("attribute"), Name: name, Language: lowerer.language, Range: lowerer.sourceRange(call)}
	if len(args) == 2 {
		field, value := classifyAttributeSecondArg(lowerer.language, args[1].Utf8Text(lowerer.input.Data))
		if field == "type" {
			attribute.Type = value
		} else {
			attribute.HasDefault = true
			attribute.Default = value
		}
	} else if len(args) > 2 {
		attribute.Type = strings.TrimSpace(args[1].Utf8Text(lowerer.input.Data))
		attribute.HasDefault = true
		attribute.Default = strings.TrimSpace(args[2].Utf8Text(lowerer.input.Data))
	}
	return attribute, true
}

func (lowerer *tsLowerer) parseOperation(ownerID string, call *sitter.Node) (Operation, bool) {
	args := callArgs(call)
	if len(args) < 2 {
		return Operation{}, false
	}
	name, ok := stringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Operation{}, false
	}
	operation := Operation{ID: lowerer.nextID("operation_decl"), Name: name, Range: lowerer.sourceRange(call)}
	behavior := lowerer.parseBehavior(ownerID, BehaviorOperation, args[1])
	lowerer.program.Behaviors = append(lowerer.program.Behaviors, behavior)
	operation.Behavior = &BehaviorRef{ID: behavior.ID, Kind: BehaviorOperation}
	return operation, true
}

func (lowerer *tsLowerer) parseState(ownerID string, call *sitter.Node) (State, bool) {
	args := callArgs(call)
	if len(args) == 0 {
		return State{}, false
	}
	name, ok := stringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return State{}, false
	}
	state := State{ID: lowerer.nextID("state"), Name: name, Kind: stateKind(lowerer.selectorName(call)), Range: lowerer.sourceRange(call)}
	for _, arg := range args[1:] {
		partial := asCall(arg)
		if partial == nil {
			continue
		}
		name := lowerer.selectorName(partial)
		switch name {
		case "Initial":
			initial := lowerer.parseInitial(state.ID, partial)
			state.Initializer = &initial
		case "State", "Final", "Choice", "ShallowHistory", "DeepHistory":
			if child, ok := lowerer.parseState(state.ID, partial); ok {
				state.States = append(state.States, child)
			}
		case "Transition":
			state.Transitions = append(state.Transitions, lowerer.parseTransition(state.ID, partial))
		case "Entry", "Exit", "Activity":
			state.Behaviors = append(state.Behaviors, lowerer.parseBehaviors(state.ID, behaviorKind(lowerer.selectorName(partial)), partial)...)
		case "Defer":
			state.Defers = append(state.Defers, lowerer.parseDefers(partial)...)
		case "Target", "Guard", "Effect":
			if !lowerer.applyHistoryDefaultPartial(&state, partial, name) {
				lowerer.addUnknownHSMPartialDiagnostic("state", partial, name)
			}
		default:
			lowerer.addUnknownHSMPartialDiagnostic("state", partial, name)
		}
	}
	_ = ownerID
	return state, true
}

func (lowerer *tsLowerer) applyHistoryDefaultPartial(state *State, partial *sitter.Node, name string) bool {
	if state.Kind != StateKindShallowHistory && state.Kind != StateKindDeepHistory {
		return false
	}
	if len(state.Transitions) == 0 {
		state.Transitions = append(state.Transitions, Transition{OwnerID: state.ID, ID: lowerer.nextID("transition"), Range: lowerer.sourceRange(partial)})
	}
	transition := &state.Transitions[0]
	switch name {
	case "Target":
		transition.Target = firstStringArg(partial, lowerer.input.Data)
	case "Guard":
		refs := lowerer.parseBehaviors(state.ID, BehaviorGuard, partial)
		if len(refs) > 0 {
			transition.Guard = &refs[0]
		}
	case "Effect":
		transition.Effects = append(transition.Effects, lowerer.parseBehaviors(state.ID, BehaviorEffect, partial)...)
	}
	return true
}

func (lowerer *tsLowerer) parseDefers(call *sitter.Node) []string {
	var defers []string
	for _, arg := range callArgs(call) {
		if value, ok := stringLiteral(arg, lowerer.input.Data); ok {
			defers = append(defers, value)
			continue
		}
		value := strings.TrimSpace(arg.Utf8Text(lowerer.input.Data))
		if event := lowerer.eventBySymbol(value); event != nil {
			defers = append(defers, event.Name)
			continue
		}
		if symbol, ok := eventMemberSymbol(value, ".name", ".Name"); ok {
			if event := lowerer.eventBySymbol(symbol); event != nil {
				defers = append(defers, event.Name)
				continue
			}
		}
		defers = append(defers, value)
	}
	return defers
}

func (lowerer *tsLowerer) parseInitial(ownerID string, call *sitter.Node) Initial {
	initial := Initial{OwnerID: ownerID, ID: lowerer.nextID("initial"), Range: lowerer.sourceRange(call)}
	for _, arg := range callArgs(call) {
		partial := asCall(arg)
		if partial == nil {
			continue
		}
		name := lowerer.selectorName(partial)
		switch name {
		case "Target":
			initial.Target = firstStringArg(partial, lowerer.input.Data)
		case "Effect":
			initial.Effects = append(initial.Effects, lowerer.parseBehaviors(ownerID, BehaviorEffect, partial)...)
		default:
			lowerer.addUnknownHSMPartialDiagnostic("initial", partial, name)
		}
	}
	return initial
}

func (lowerer *tsLowerer) parseTransition(ownerID string, call *sitter.Node) Transition {
	transition := Transition{OwnerID: ownerID, ID: lowerer.nextID("transition"), Range: lowerer.sourceRange(call)}
	for _, arg := range callArgs(call) {
		partial := asCall(arg)
		if partial == nil {
			continue
		}
		name := lowerer.selectorName(partial)
		switch name {
		case "Source":
			transition.Source = firstStringArg(partial, lowerer.input.Data)
		case "Target":
			transition.Target = firstStringArg(partial, lowerer.input.Data)
		case "On":
			transition.Trigger = lowerer.parseOnTrigger(partial)
		case "OnSet":
			transition.Trigger = &Trigger{Kind: TriggerOnSet, Value: firstStringArg(partial, lowerer.input.Data), IsString: true, Language: lowerer.language}
		case "OnCall":
			transition.Trigger = &Trigger{Kind: TriggerOnCall, Value: firstStringArg(partial, lowerer.input.Data), IsString: true, Language: lowerer.language}
		case "After":
			value, isString := firstArgValue(partial, lowerer.input.Data)
			transition.Trigger = lowerer.parseExpressionTrigger(ownerID, TriggerAfter, value, isString, partial)
		case "Every":
			value, isString := firstArgValue(partial, lowerer.input.Data)
			transition.Trigger = lowerer.parseExpressionTrigger(ownerID, TriggerEvery, value, isString, partial)
		case "At":
			value, isString := firstArgValue(partial, lowerer.input.Data)
			transition.Trigger = lowerer.parseExpressionTrigger(ownerID, TriggerAt, value, isString, partial)
		case "When":
			value, isString := firstArgValue(partial, lowerer.input.Data)
			transition.Trigger = lowerer.parseExpressionTrigger(ownerID, TriggerWhen, value, isString, partial)
		case "Guard":
			refs := lowerer.parseBehaviors(ownerID, BehaviorGuard, partial)
			if len(refs) > 0 {
				transition.Guard = &refs[0]
			}
		case "Effect":
			transition.Effects = append(transition.Effects, lowerer.parseBehaviors(ownerID, BehaviorEffect, partial)...)
		default:
			lowerer.addUnknownHSMPartialDiagnostic("transition", partial, name)
		}
	}
	return transition
}

func (lowerer *tsLowerer) parseOnTrigger(call *sitter.Node) *Trigger {
	trigger := &Trigger{Kind: TriggerOn, Language: lowerer.language}
	for _, arg := range callArgs(call) {
		value, isString := nodeArgValue(arg, lowerer.input.Data)
		eventSymbol := ""
		if !isString {
			if lowerer.eventBySymbol(value) != nil {
				eventSymbol = value
			} else if symbol, ok := eventMemberSymbol(value, ".name", ".Name"); ok && lowerer.eventBySymbol(symbol) != nil {
				eventSymbol = symbol
			}
		}
		if trigger.Value == "" {
			trigger.Value = value
			trigger.IsString = isString
		}
		trigger.Values = append(trigger.Values, TriggerValue{Value: value, IsString: isString, EventSymbol: eventSymbol, Language: lowerer.language})
	}
	if len(trigger.Values) <= 1 && (len(trigger.Values) == 0 || trigger.Values[0].EventSymbol == "") {
		trigger.Values = nil
	}
	return trigger
}

func (lowerer *tsLowerer) parseEventDecl(node *sitter.Node) (EventDecl, bool) {
	var event EventDecl
	walk(node, func(child *sitter.Node) {
		if event.Symbol != "" || child.Kind() != "variable_declarator" {
			return
		}
		if parsed, ok := lowerer.parseEventDeclarator(child); ok {
			event = parsed
		}
	})
	return event, event.Symbol != ""
}

func (lowerer *tsLowerer) parseEventDeclarator(declarator *sitter.Node) (EventDecl, bool) {
	nameNode := declarator.ChildByFieldName("name")
	valueNode := declarator.ChildByFieldName("value")
	if nameNode == nil || valueNode == nil || asCall(valueNode) == nil || lowerer.selectorName(valueNode) != "Event" {
		return EventDecl{}, false
	}
	args := callArgs(valueNode)
	if len(args) == 0 {
		return EventDecl{}, false
	}
	name, ok := stringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return EventDecl{}, false
	}
	event := EventDecl{
		ID:       lowerer.nextID("event"),
		Symbol:   strings.TrimSpace(nameNode.Utf8Text(lowerer.input.Data)),
		Name:     name,
		Language: lowerer.language,
		Range:    lowerer.sourceRange(declarator),
	}
	if len(args) > 1 {
		event.Schema = strings.TrimSpace(args[1].Utf8Text(lowerer.input.Data))
	}
	return event, true
}

func (lowerer *tsLowerer) eventBySymbol(symbol string) *EventDecl {
	for index := range lowerer.program.Events {
		if lowerer.program.Events[index].Symbol == symbol {
			return &lowerer.program.Events[index]
		}
	}
	return nil
}

func (lowerer *tsLowerer) parseExpressionTrigger(ownerID string, kind TriggerKind, value string, isString bool, call *sitter.Node) *Trigger {
	trigger := &Trigger{Kind: kind, Value: value, IsString: isString, Language: lowerer.language}
	args := callArgs(call)
	if isString || len(args) == 0 {
		return trigger
	}
	behavior := lowerer.parseBehavior(ownerID, BehaviorTrigger, args[0])
	behavior.TriggerKind = kind
	lowerer.program.Behaviors = append(lowerer.program.Behaviors, behavior)
	trigger.Expr = &BehaviorRef{ID: behavior.ID, Kind: BehaviorTrigger}
	return trigger
}

func (lowerer *tsLowerer) parseBehaviors(ownerID string, kind BehaviorKind, call *sitter.Node) []BehaviorRef {
	var refs []BehaviorRef
	for _, arg := range callArgs(call) {
		if operation, ok := stringLiteral(arg, lowerer.input.Data); ok {
			refs = append(refs, BehaviorRef{Kind: kind, Operation: operation})
			continue
		}
		behavior := lowerer.parseBehavior(ownerID, kind, arg)
		lowerer.program.Behaviors = append(lowerer.program.Behaviors, behavior)
		refs = append(refs, BehaviorRef{ID: behavior.ID, Kind: kind})
	}
	return refs
}

func (lowerer *tsLowerer) parseBehavior(ownerID string, kind BehaviorKind, node *sitter.Node) Behavior {
	inline := node.Kind() == "arrow_function" || node.Kind() == "function"
	if resolved := lowerer.referencedBehaviorFunction(node, kind); resolved != nil {
		node = resolved
	}
	code := strings.TrimSpace(node.Utf8Text(lowerer.input.Data))
	behavior := Behavior{
		ID:             lowerer.nextID(string(kind)),
		OwnerID:        ownerID,
		Kind:           kind,
		Inline:         inline,
		SourceLanguage: lowerer.language,
		Code:           code,
		Range:          lowerer.sourceRange(node),
	}
	if node.Kind() == "arrow_function" || node.Kind() == "function" || node.Kind() == "function_declaration" {
		if body := node.ChildByFieldName("body"); body != nil {
			behavior.Body = lowerer.behaviorBody(kind, body)
			behavior.Signature = strings.TrimSpace(string(lowerer.input.Data[node.StartByte():body.StartByte()]))
		}
	}
	return behavior
}

type tsFunctionDecl struct {
	node        *sitter.Node
	globalRange SourceRange
}

func (lowerer *tsLowerer) registerFunctionDecl(node *sitter.Node, globalRange SourceRange) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		nameNode = firstNamedChildOfKind(node, "identifier")
	}
	if nameNode == nil {
		return
	}
	if lowerer.functions == nil {
		lowerer.functions = map[string]tsFunctionDecl{}
	}
	lowerer.functions[nameNode.Utf8Text(lowerer.input.Data)] = tsFunctionDecl{node: node, globalRange: globalRange}
}

func (lowerer *tsLowerer) referencedBehaviorFunction(node *sitter.Node, kind BehaviorKind) *sitter.Node {
	if node == nil || node.Kind() != "identifier" {
		return nil
	}
	name := node.Utf8Text(lowerer.input.Data)
	if isGeneratedTSBehaviorFunctionNameForAnyKind(name) && !isGeneratedBehaviorFunctionName(name, kind) {
		return nil
	}
	decl, ok := lowerer.functions[name]
	if !ok {
		return nil
	}
	if !isGeneratedBehaviorFunctionName(name, kind) && !lowerer.functionDeclLooksLikeBehavior(decl.node) {
		return nil
	}
	if lowerer.usedFunctions == nil {
		lowerer.usedFunctions = map[string]bool{}
	}
	lowerer.usedFunctions[name] = true
	return decl.node
}

func (lowerer *tsLowerer) functionDeclLooksLikeBehavior(node *sitter.Node) bool {
	if node == nil || node.Kind() != "function_declaration" {
		return false
	}
	body := node.ChildByFieldName("body")
	if body == nil {
		return false
	}
	signature := string(lowerer.input.Data[node.StartByte():body.StartByte()])
	if strings.Contains(signature, "hsm.Event") {
		return true
	}
	parameters := node.ChildByFieldName("parameters")
	return countCommaSeparatedParameters(parameters, lowerer.input.Data) >= 3
}

func (lowerer *tsLowerer) filterReferencedFunctionGlobals() {
	if len(lowerer.usedFunctions) == 0 || len(lowerer.functions) == 0 {
		return
	}
	usedRanges := map[SourceRange]bool{}
	for name := range lowerer.usedFunctions {
		if decl, ok := lowerer.functions[name]; ok {
			usedRanges[decl.globalRange] = true
		}
	}
	globals := lowerer.program.Globals[:0]
	for _, global := range lowerer.program.Globals {
		if !usedRanges[global.Range] {
			globals = append(globals, global)
		}
	}
	lowerer.program.Globals = globals
}

func isGeneratedBehaviorFunctionName(name string, kind BehaviorKind) bool {
	prefix := string(kind)
	if kind == BehaviorOperation {
		prefix = "operation"
	}
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	suffix := strings.TrimPrefix(name, prefix)
	if suffix == "" {
		return false
	}
	for _, r := range suffix {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isGeneratedTSBehaviorFunctionNameForAnyKind(name string) bool {
	for _, kind := range []BehaviorKind{
		BehaviorEntry,
		BehaviorExit,
		BehaviorActivity,
		BehaviorGuard,
		BehaviorEffect,
		BehaviorTrigger,
		BehaviorOperation,
	} {
		if isGeneratedBehaviorFunctionName(name, kind) {
			return true
		}
	}
	return false
}

func countCommaSeparatedParameters(node *sitter.Node, input []byte) int {
	if node == nil {
		return 0
	}
	text := strings.TrimSpace(node.Utf8Text(input))
	text = strings.TrimPrefix(text, "(")
	text = strings.TrimSuffix(text, ")")
	if strings.TrimSpace(text) == "" {
		return 0
	}
	count := 1
	depth := 0
	for _, r := range text {
		switch r {
		case '(', '[', '{', '<':
			depth++
		case ')', ']', '}', '>':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				count++
			}
		}
	}
	return count
}

func (lowerer *tsLowerer) behaviorBody(kind BehaviorKind, body *sitter.Node) string {
	if body.Kind() == "statement_block" {
		return trimBlock(body.Utf8Text(lowerer.input.Data))
	}
	expression := strings.TrimSpace(body.Utf8Text(lowerer.input.Data))
	if kind == BehaviorGuard {
		return "return " + expression + ";"
	}
	return expression + ";"
}

func (lowerer *tsLowerer) parseImport(node *sitter.Node) (Import, bool) {
	source := node.ChildByFieldName("source")
	if source == nil {
		return Import{}, false
	}
	pathValue, ok := stringLiteral(source, lowerer.input.Data)
	if !ok {
		return Import{}, false
	}
	text := strings.TrimSpace(node.Utf8Text(lowerer.input.Data))
	imp := Import{Path: pathValue, TypeOnly: strings.HasPrefix(text, "import type "), Language: lowerer.language, Range: lowerer.sourceRange(node)}
	walk(node, func(child *sitter.Node) {
		if child.Kind() == "namespace_import" && child.NamedChildCount() > 0 {
			imp.Alias = child.NamedChild(0).Utf8Text(lowerer.input.Data)
			imp.LocalNames = append(imp.LocalNames, imp.Alias)
			return
		}
		if child.Kind() == "import_specifier" {
			specifier := ImportSpecifier{TypeOnly: strings.HasPrefix(strings.TrimSpace(child.Utf8Text(lowerer.input.Data)), "type ")}
			if aliasNode := child.ChildByFieldName("alias"); aliasNode != nil {
				specifier.Alias = aliasNode.Utf8Text(lowerer.input.Data)
			}
			if nameNode := child.ChildByFieldName("name"); nameNode != nil {
				specifier.Name = nameNode.Utf8Text(lowerer.input.Data)
			}
			if specifier.Name != "" {
				imp.Specifiers = append(imp.Specifiers, specifier)
				if specifier.Alias != "" {
					imp.LocalNames = append(imp.LocalNames, specifier.Alias)
				} else {
					imp.LocalNames = append(imp.LocalNames, specifier.Name)
				}
			}
			return
		}
		if child.Kind() == "identifier" && child.Parent() != nil && child.Parent().Kind() == "import_clause" {
			imp.Default = child.Utf8Text(lowerer.input.Data)
			imp.LocalNames = append(imp.LocalNames, imp.Default)
		}
	})
	return imp, true
}

func (lowerer *tsLowerer) globalBlock(node *sitter.Node) CodeBlock {
	return CodeBlock{
		ID:       lowerer.nextID("global"),
		Language: lowerer.language,
		Code:     node.Utf8Text(lowerer.input.Data),
		Range:    lowerer.sourceRange(node),
	}
}

func (lowerer *tsLowerer) globalDeclaratorBlock(prefix string, declarator *sitter.Node) CodeBlock {
	code := strings.TrimSpace(prefix)
	if code != "" {
		code += " "
	}
	code += strings.TrimSpace(declarator.Utf8Text(lowerer.input.Data))
	if !strings.HasSuffix(code, ";") {
		code += ";"
	}
	return CodeBlock{
		ID:       lowerer.nextID("global"),
		Language: lowerer.language,
		Code:     code,
		Range:    lowerer.sourceRange(declarator),
	}
}

func directVariableDeclarators(node *sitter.Node) []*sitter.Node {
	var declarators []*sitter.Node
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child.Kind() == "variable_declarator" {
			declarators = append(declarators, child)
		}
	}
	return declarators
}

func (lowerer *tsLowerer) sourceRange(node *sitter.Node) SourceRange {
	start := node.StartPosition()
	end := node.EndPosition()
	return SourceRange{
		File:      lowerer.input.Path,
		StartByte: node.StartByte(),
		EndByte:   node.EndByte(),
		StartLine: uint(start.Row) + 1,
		EndLine:   uint(end.Row) + 1,
	}
}

func (lowerer *tsLowerer) nextID(prefix string) string {
	if lowerer.counters == nil {
		lowerer.counters = map[string]int{}
	}
	lowerer.counters[prefix]++
	return fmt.Sprintf("%s_%d", prefix, lowerer.counters[prefix])
}

func (lowerer *tsLowerer) containsHSMDefine(node *sitter.Node) bool {
	found := false
	walk(node, func(child *sitter.Node) {
		if (child.Kind() == "call_expression" || child.Kind() == "call") && lowerer.selectorName(child) == "Define" {
			found = true
		}
	})
	return found
}

func (lowerer *tsLowerer) selectorName(call *sitter.Node) string {
	name := selectorName(call, lowerer.input.Data)
	if canonical, ok := lowerer.hsmNamedBindings[name]; ok {
		return canonical
	}
	return tsCanonicalName(name)
}

func (lowerer *tsLowerer) addUnknownHSMPartialDiagnostic(context string, call *sitter.Node, name string) {
	if !lowerer.isHSMNamespaceCall(call) {
		return
	}
	if name == "" {
		name = selectorName(call, lowerer.input.Data)
	}
	lowerer.diagnostics = append(lowerer.diagnostics, Diagnostic{
		Message: fmt.Sprintf("unknown or unsupported hsm.%s partial in %s context", name, context),
		Range:   lowerer.sourceRange(call),
	})
}

func (lowerer *tsLowerer) isHSMNamespaceCall(call *sitter.Node) bool {
	function := call.ChildByFieldName("function")
	if function == nil && call.NamedChildCount() > 0 {
		function = call.NamedChild(0)
	}
	if function == nil {
		return false
	}
	text := strings.TrimSpace(function.Utf8Text(lowerer.input.Data))
	index := strings.LastIndex(text, ".")
	if index < 0 {
		return false
	}
	qualifier := strings.TrimSpace(text[:index])
	qualifier = strings.Trim(qualifier, "()")
	qualifier = strings.TrimSpace(qualifier)
	return qualifier == "hsm" || lowerer.hsmNamespaceAliases[qualifier]
}

func tsCanonicalName(value string) string {
	switch strings.ToLower(value) {
	case "define":
		return "Define"
	case "event":
		return "Event"
	case "defaultclock":
		return "DefaultClock"
	case "config":
		return "Config"
	case "queue":
		return "Queue"
	case "clock":
		return "Clock"
	case "makegroup":
		return "MakeGroup"
	case "makekind":
		return "MakeKind"
	case "iskind":
		return "IsKind"
	case "takesnapshot":
		return "TakeSnapshot"
	case "initial":
		return "Initial"
	case "state":
		return "State"
	case "finalstate", "final":
		return "Final"
	case "choice":
		return "Choice"
	case "shallowhistory":
		return "ShallowHistory"
	case "deephistory":
		return "DeepHistory"
	case "transition":
		return "Transition"
	case "source":
		return "Source"
	case "target":
		return "Target"
	case "on":
		return "On"
	case "onset":
		return "OnSet"
	case "oncall":
		return "OnCall"
	case "after":
		return "After"
	case "every":
		return "Every"
	case "at":
		return "At"
	case "when":
		return "When"
	case "guard":
		return "Guard"
	case "effect":
		return "Effect"
	case "entry":
		return "Entry"
	case "exit":
		return "Exit"
	case "activity":
		return "Activity"
	case "defer":
		return "Defer"
	case "attribute":
		return "Attribute"
	case "operation":
		return "Operation"
	default:
		return value
	}
}

func tsKnownDSLName(value string) bool {
	switch value {
	case "Define", "Event", "DefaultClock", "Config", "Queue", "Clock", "MakeGroup", "MakeKind", "IsKind", "TakeSnapshot", "Initial", "State", "Final", "Choice", "ShallowHistory", "DeepHistory", "Transition", "Source", "Target", "On", "OnSet", "OnCall", "After", "Every", "At", "When", "Guard", "Effect", "Entry", "Exit", "Activity", "Defer", "Attribute", "Operation":
		return true
	default:
		return false
	}
}
