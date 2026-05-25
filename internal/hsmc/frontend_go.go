package hsmc

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type GoFrontend struct{}

func NewGoFrontend() GoFrontend {
	return GoFrontend{}
}

func (GoFrontend) Language() Language { return LanguageGo }

func (frontend GoFrontend) Parse(ctx context.Context, input SourceInput) (*Program, error) {
	tree, err := parseTree(ctx, input, parserGrammarGo)
	if err != nil {
		return nil, err
	}
	defer tree.Close()
	root := tree.RootNode()
	lowerer := goLowerer{
		input: SourceInput{Path: input.Path, Data: input.Data},
		program: &Program{
			SourceLanguage: LanguageGo,
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

type goLowerer struct {
	input            SourceInput
	program          *Program
	counters         map[string]int
	functions        map[string]goFunctionDecl
	usedFunctions    map[string]bool
	hsmImportAliases map[string]bool
	diagnostics      Diagnostics
}

func (lowerer *goLowerer) collectTopLevel(root *sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		child := root.NamedChild(i)
		switch child.Kind() {
		case "package_clause":
			if ident := firstNamedChildOfKind(child, "package_identifier"); ident != nil {
				lowerer.program.PackageName = ident.Utf8Text(lowerer.input.Data)
			}
		case "import_declaration":
			for _, imp := range lowerer.parseImports(child) {
				if isHSMImportPath(imp.Path) {
					lowerer.recordHSMImportAlias(imp)
					continue
				}
				lowerer.program.Imports = append(lowerer.program.Imports, imp)
			}
		case "function_declaration", "method_declaration", "type_declaration", "const_declaration":
			block := CodeBlock{
				ID:       lowerer.nextID("global"),
				Language: LanguageGo,
				Code:     lowerer.normalizeHSMRuntimeAliases(child.Utf8Text(lowerer.input.Data)),
				Range:    lowerer.sourceRange(child),
			}
			if child.Kind() == "function_declaration" {
				lowerer.registerFunctionDecl(child, block.Range)
			}
			lowerer.program.Globals = append(lowerer.program.Globals, block)
		case "var_declaration":
			lowerer.collectVarDeclaration(child)
		}
	}
}

func (lowerer *goLowerer) collectVarDeclaration(node *sitter.Node) {
	if !containsHSMDefine(node, lowerer.input.Data) {
		if event, ok := lowerer.parseEventDecl(node); ok {
			lowerer.program.Events = append(lowerer.program.Events, event)
			return
		}
		lowerer.program.Globals = append(lowerer.program.Globals, CodeBlock{
			ID:       lowerer.nextID("global"),
			Language: LanguageGo,
			Code:     lowerer.normalizeHSMRuntimeAliases(node.Utf8Text(lowerer.input.Data)),
			Range:    lowerer.sourceRange(node),
		})
		return
	}
	walk(node, func(child *sitter.Node) {
		if child.Kind() != "var_spec" || containsHSMDefine(child, lowerer.input.Data) {
			return
		}
		code := "var " + strings.TrimSpace(child.Utf8Text(lowerer.input.Data))
		if event, ok := lowerer.parseEventDeclText(code, child); ok {
			lowerer.program.Events = append(lowerer.program.Events, event)
			return
		}
		lowerer.program.Globals = append(lowerer.program.Globals, CodeBlock{
			ID:       lowerer.nextID("global"),
			Language: LanguageGo,
			Code:     lowerer.normalizeHSMRuntimeAliases(code),
			Range:    lowerer.sourceRange(child),
		})
	})
}

func (lowerer *goLowerer) collectModels(root *sitter.Node) {
	walk(root, func(node *sitter.Node) {
		if node.Kind() != "call_expression" || selectorName(node, lowerer.input.Data) != "Define" {
			return
		}
		model, ok := lowerer.parseModel(node)
		if ok {
			lowerer.program.Models = append(lowerer.program.Models, model)
		}
	})
}

func (lowerer *goLowerer) parseModel(call *sitter.Node) (Model, bool) {
	args := callArgs(call)
	if len(args) == 0 {
		return Model{}, false
	}
	name, ok := stringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Model{}, false
	}
	model := Model{
		ID:    lowerer.nextID("model"),
		Name:  name,
		Range: lowerer.sourceRange(call),
	}
	for _, arg := range args[1:] {
		lowerer.applyModelPartial(&model, arg)
	}
	return model, true
}

func (lowerer *goLowerer) applyModelPartial(model *Model, arg *sitter.Node) {
	call := asCall(arg)
	if call == nil {
		return
	}
	name := selectorName(call, lowerer.input.Data)
	switch name {
	case "Initial":
		initial := lowerer.parseInitial(model.ID, call)
		model.Initializer = &initial
	case "State", "Final", "Choice", "ShallowHistory", "DeepHistory":
		if state, ok := lowerer.parseState(model.ID, call); ok {
			model.States = append(model.States, state)
		}
	case "Transition":
		transition := lowerer.parseTransition(model.ID, call)
		model.Transitions = append(model.Transitions, transition)
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

func (lowerer *goLowerer) parseAttribute(call *sitter.Node) (Attribute, bool) {
	args := callArgs(call)
	if len(args) == 0 {
		return Attribute{}, false
	}
	name, ok := stringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Attribute{}, false
	}
	attribute := Attribute{ID: lowerer.nextID("attribute"), Name: name, Language: LanguageGo, Range: lowerer.sourceRange(call)}
	if len(args) == 2 {
		field, value := classifyAttributeSecondArg(LanguageGo, args[1].Utf8Text(lowerer.input.Data))
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

func (lowerer *goLowerer) parseOperation(ownerID string, call *sitter.Node) (Operation, bool) {
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

func (lowerer *goLowerer) parseState(ownerID string, call *sitter.Node) (State, bool) {
	args := callArgs(call)
	if len(args) == 0 {
		return State{}, false
	}
	name, ok := stringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return State{}, false
	}
	state := State{
		ID:    lowerer.nextID("state"),
		Name:  name,
		Kind:  stateKind(selectorName(call, lowerer.input.Data)),
		Range: lowerer.sourceRange(call),
	}
	for _, arg := range args[1:] {
		partial := asCall(arg)
		if partial == nil {
			continue
		}
		name := selectorName(partial, lowerer.input.Data)
		switch name {
		case "Initial":
			initial := lowerer.parseInitial(state.ID, partial)
			state.Initializer = &initial
		case "State", "Final", "Choice", "ShallowHistory", "DeepHistory":
			if child, ok := lowerer.parseState(state.ID, partial); ok {
				state.States = append(state.States, child)
			}
		case "Transition":
			transition := lowerer.parseTransition(state.ID, partial)
			state.Transitions = append(state.Transitions, transition)
		case "Entry", "Exit", "Activity":
			state.Behaviors = append(state.Behaviors, lowerer.parseBehaviors(state.ID, behaviorKind(selectorName(partial, lowerer.input.Data)), partial)...)
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

func (lowerer *goLowerer) applyHistoryDefaultPartial(state *State, partial *sitter.Node, name string) bool {
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

func (lowerer *goLowerer) parseDefers(call *sitter.Node) []string {
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
		if symbol, ok := eventMemberSymbol(value, ".Name", ".name"); ok {
			if event := lowerer.eventBySymbol(symbol); event != nil {
				defers = append(defers, event.Name)
				continue
			}
		}
		defers = append(defers, value)
	}
	return defers
}

func (lowerer *goLowerer) parseInitial(ownerID string, call *sitter.Node) Initial {
	initial := Initial{OwnerID: ownerID, ID: lowerer.nextID("initial"), Range: lowerer.sourceRange(call)}
	for _, arg := range callArgs(call) {
		partial := asCall(arg)
		if partial == nil {
			continue
		}
		name := selectorName(partial, lowerer.input.Data)
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

func (lowerer *goLowerer) parseTransition(ownerID string, call *sitter.Node) Transition {
	transition := Transition{OwnerID: ownerID, ID: lowerer.nextID("transition"), Range: lowerer.sourceRange(call)}
	for _, arg := range callArgs(call) {
		partial := asCall(arg)
		if partial == nil {
			continue
		}
		name := selectorName(partial, lowerer.input.Data)
		switch name {
		case "Source":
			transition.Source = firstStringArg(partial, lowerer.input.Data)
		case "Target":
			transition.Target = firstStringArg(partial, lowerer.input.Data)
		case "On":
			transition.Trigger = lowerer.parseOnTrigger(partial)
		case "OnSet":
			transition.Trigger = &Trigger{Kind: TriggerOnSet, Value: firstStringArg(partial, lowerer.input.Data), IsString: true, Language: LanguageGo}
		case "OnCall":
			transition.Trigger = &Trigger{Kind: TriggerOnCall, Value: firstStringArg(partial, lowerer.input.Data), IsString: true, Language: LanguageGo}
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

func (lowerer *goLowerer) parseOnTrigger(call *sitter.Node) *Trigger {
	trigger := &Trigger{Kind: TriggerOn, Language: LanguageGo}
	for _, arg := range callArgs(call) {
		value, isString := nodeArgValue(arg, lowerer.input.Data)
		eventSymbol := ""
		if !isString {
			if lowerer.eventBySymbol(value) != nil {
				eventSymbol = value
			} else if symbol, ok := eventMemberSymbol(value, ".Name", ".name"); ok && lowerer.eventBySymbol(symbol) != nil {
				eventSymbol = symbol
			}
		}
		if trigger.Value == "" {
			trigger.Value = value
			trigger.IsString = isString
		}
		trigger.Values = append(trigger.Values, TriggerValue{Value: value, IsString: isString, EventSymbol: eventSymbol, Language: LanguageGo})
	}
	if len(trigger.Values) <= 1 && (len(trigger.Values) == 0 || trigger.Values[0].EventSymbol == "") {
		trigger.Values = nil
	}
	return trigger
}

func (lowerer *goLowerer) parseEventDecl(node *sitter.Node) (EventDecl, bool) {
	text := strings.TrimSpace(node.Utf8Text(lowerer.input.Data))
	return lowerer.parseEventDeclText(text, node)
}

func (lowerer *goLowerer) parseEventDeclText(text string, node *sitter.Node) (EventDecl, bool) {
	if !strings.HasPrefix(text, "var ") || !strings.Contains(text, "Event{") {
		return EventDecl{}, false
	}
	before, after, ok := strings.Cut(strings.TrimPrefix(text, "var "), "=")
	if !ok {
		return EventDecl{}, false
	}
	fields := strings.Fields(strings.TrimSpace(before))
	if len(fields) == 0 {
		return EventDecl{}, false
	}
	name, ok := goEventNameFromComposite(after)
	if !ok {
		return EventDecl{}, false
	}
	return EventDecl{
		ID:       lowerer.nextID("event"),
		Symbol:   fields[0],
		Name:     name,
		Language: LanguageGo,
		Range:    lowerer.sourceRange(node),
	}, true
}

func (lowerer *goLowerer) eventBySymbol(symbol string) *EventDecl {
	for index := range lowerer.program.Events {
		if lowerer.program.Events[index].Symbol == symbol {
			return &lowerer.program.Events[index]
		}
	}
	return nil
}

func goEventNameFromComposite(text string) (string, bool) {
	index := strings.Index(text, "Name:")
	if index < 0 {
		return "", false
	}
	rest := strings.TrimSpace(text[index+len("Name:"):])
	end := strings.IndexAny(rest, ",}")
	if end >= 0 {
		rest = rest[:end]
	}
	value, err := strconv.Unquote(strings.TrimSpace(rest))
	return value, err == nil
}

func (lowerer *goLowerer) parseExpressionTrigger(ownerID string, kind TriggerKind, value string, isString bool, call *sitter.Node) *Trigger {
	trigger := &Trigger{Kind: kind, Value: value, IsString: isString, Language: LanguageGo}
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

func (lowerer *goLowerer) parseBehaviors(ownerID string, kind BehaviorKind, call *sitter.Node) []BehaviorRef {
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

func (lowerer *goLowerer) parseBehavior(ownerID string, kind BehaviorKind, node *sitter.Node) Behavior {
	inline := node.Kind() == "func_literal"
	if resolved := lowerer.referencedBehaviorFunction(node, kind); resolved != nil {
		node = resolved
	}
	code := strings.TrimSpace(lowerer.normalizeHSMRuntimeAliases(node.Utf8Text(lowerer.input.Data)))
	behavior := Behavior{
		ID:             lowerer.nextID(string(kind)),
		OwnerID:        ownerID,
		Kind:           kind,
		Inline:         inline,
		SourceLanguage: LanguageGo,
		Code:           code,
		Range:          lowerer.sourceRange(node),
	}
	if node.Kind() == "func_literal" || node.Kind() == "function_declaration" {
		if body := node.ChildByFieldName("body"); body != nil {
			behavior.Body = trimBlock(lowerer.normalizeHSMRuntimeAliases(body.Utf8Text(lowerer.input.Data)))
			signature := strings.TrimSpace(lowerer.normalizeHSMRuntimeAliases(string(lowerer.input.Data[node.StartByte():body.StartByte()])))
			behavior.Signature = signature
		}
	}
	return behavior
}

type goFunctionDecl struct {
	node        *sitter.Node
	globalRange SourceRange
}

func (lowerer *goLowerer) registerFunctionDecl(node *sitter.Node, globalRange SourceRange) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		nameNode = firstNamedChildOfKind(node, "identifier")
	}
	if nameNode == nil {
		return
	}
	if lowerer.functions == nil {
		lowerer.functions = map[string]goFunctionDecl{}
	}
	lowerer.functions[nameNode.Utf8Text(lowerer.input.Data)] = goFunctionDecl{node: node, globalRange: globalRange}
}

func (lowerer *goLowerer) referencedBehaviorFunction(node *sitter.Node, kind BehaviorKind) *sitter.Node {
	if node == nil || node.Kind() != "identifier" {
		return nil
	}
	name := node.Utf8Text(lowerer.input.Data)
	if isGeneratedGoBehaviorFunctionNameForAnyKind(name) && !isGeneratedGoBehaviorFunctionName(name, kind) {
		return nil
	}
	decl, ok := lowerer.functions[name]
	if !ok {
		return nil
	}
	if !isGeneratedGoBehaviorFunctionName(name, kind) && !lowerer.functionDeclLooksLikeBehavior(decl.node) {
		return nil
	}
	if lowerer.usedFunctions == nil {
		lowerer.usedFunctions = map[string]bool{}
	}
	lowerer.usedFunctions[name] = true
	return decl.node
}

func (lowerer *goLowerer) functionDeclLooksLikeBehavior(node *sitter.Node) bool {
	if node == nil || node.Kind() != "function_declaration" {
		return false
	}
	body := node.ChildByFieldName("body")
	if body == nil {
		return false
	}
	signature := lowerer.normalizeHSMRuntimeAliases(string(lowerer.input.Data[node.StartByte():body.StartByte()]))
	return strings.Contains(signature, "hsm.Event")
}

func (lowerer *goLowerer) filterReferencedFunctionGlobals() {
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

func isGeneratedGoBehaviorFunctionName(name string, kind BehaviorKind) bool {
	prefix := exportName(string(kind))
	if kind == BehaviorOperation {
		prefix = "Operation"
	}
	if kind == BehaviorTrigger {
		prefix = "Trigger"
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

func isGeneratedGoBehaviorFunctionNameForAnyKind(name string) bool {
	for _, kind := range []BehaviorKind{
		BehaviorEntry,
		BehaviorExit,
		BehaviorActivity,
		BehaviorGuard,
		BehaviorEffect,
		BehaviorTrigger,
		BehaviorOperation,
	} {
		if isGeneratedGoBehaviorFunctionName(name, kind) {
			return true
		}
	}
	return false
}

func (lowerer *goLowerer) recordHSMImportAlias(imp Import) {
	if imp.Alias == "" || imp.Alias == "hsm" || imp.Alias == "." || imp.Alias == "_" {
		return
	}
	if lowerer.hsmImportAliases == nil {
		lowerer.hsmImportAliases = map[string]bool{}
	}
	lowerer.hsmImportAliases[imp.Alias] = true
}

func (lowerer *goLowerer) normalizeHSMRuntimeAliases(code string) string {
	for alias := range lowerer.hsmImportAliases {
		pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(alias) + `\.`)
		code = pattern.ReplaceAllString(code, "hsm.")
	}
	return code
}

func (lowerer *goLowerer) addUnknownHSMPartialDiagnostic(context string, call *sitter.Node, name string) {
	if !lowerer.isHSMNamespaceCall(call) {
		return
	}
	lowerer.diagnostics = append(lowerer.diagnostics, Diagnostic{
		Message: fmt.Sprintf("unknown or unsupported hsm.%s partial in %s context", name, context),
		Range:   lowerer.sourceRange(call),
	})
}

func (lowerer *goLowerer) isHSMNamespaceCall(call *sitter.Node) bool {
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
	return qualifier == "hsm" || lowerer.hsmImportAliases[qualifier]
}

func (lowerer *goLowerer) parseImports(node *sitter.Node) []Import {
	var imports []Import
	walk(node, func(child *sitter.Node) {
		if child.Kind() != "import_spec" {
			return
		}
		var alias string
		var pathValue string
		for i := uint(0); i < child.NamedChildCount(); i++ {
			part := child.NamedChild(i)
			switch part.Kind() {
			case "package_identifier", "dot", "blank_identifier":
				alias = part.Utf8Text(lowerer.input.Data)
			case "interpreted_string_literal", "raw_string_literal":
				pathValue, _ = stringLiteral(part, lowerer.input.Data)
			}
		}
		if pathValue != "" {
			imports = append(imports, Import{Path: pathValue, Alias: alias, LocalNames: goImportLocalNames(pathValue, alias), Language: LanguageGo, Range: lowerer.sourceRange(child)})
		}
	})
	return imports
}

func goImportLocalNames(importPath string, alias string) []string {
	if alias != "" {
		return []string{alias}
	}
	base := path.Base(importPath)
	base = strings.TrimSuffix(base, ".go")
	return []string{base}
}

func (lowerer *goLowerer) sourceRange(node *sitter.Node) SourceRange {
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

func (lowerer *goLowerer) nextID(prefix string) string {
	if lowerer.counters == nil {
		lowerer.counters = map[string]int{}
	}
	lowerer.counters[prefix]++
	return fmt.Sprintf("%s_%d", prefix, lowerer.counters[prefix])
}

func stateKind(name string) StateKind {
	switch name {
	case "Final":
		return StateKindFinal
	case "Choice":
		return StateKindChoice
	case "ShallowHistory":
		return StateKindShallowHistory
	case "DeepHistory":
		return StateKindDeepHistory
	default:
		return StateKindState
	}
}

func behaviorKind(name string) BehaviorKind {
	switch name {
	case "Exit":
		return BehaviorExit
	case "Activity":
		return BehaviorActivity
	default:
		return BehaviorEntry
	}
}

func trimBlock(code string) string {
	code = strings.TrimSpace(code)
	if strings.HasPrefix(code, "{") && strings.HasSuffix(code, "}") {
		code = strings.TrimPrefix(code, "{")
		code = strings.TrimSuffix(code, "}")
	}
	return strings.TrimSpace(code)
}

func containsHSMDefine(node *sitter.Node, source []byte) bool {
	found := false
	walk(node, func(child *sitter.Node) {
		if (child.Kind() == "call_expression" || child.Kind() == "call") && selectorName(child, source) == "Define" {
			found = true
		}
	})
	return found
}

func asCall(node *sitter.Node) *sitter.Node {
	if node == nil || (node.Kind() != "call_expression" && node.Kind() != "call") {
		return nil
	}
	return node
}

func callArgs(call *sitter.Node) []*sitter.Node {
	args := call.ChildByFieldName("arguments")
	if args == nil {
		for i := uint(0); i < call.NamedChildCount(); i++ {
			child := call.NamedChild(i)
			if child.Kind() == "argument_list" {
				args = child
				break
			}
		}
	}
	if args == nil {
		return nil
	}
	var result []*sitter.Node
	for i := uint(0); i < args.NamedChildCount(); i++ {
		result = append(result, args.NamedChild(i))
	}
	return result
}

func selectorName(call *sitter.Node, source []byte) string {
	function := call.ChildByFieldName("function")
	if function == nil && call.NamedChildCount() > 0 {
		function = call.NamedChild(0)
	}
	if function == nil {
		return ""
	}
	text := function.Utf8Text(source)
	if index := strings.LastIndex(text, "."); index >= 0 {
		return text[index+1:]
	}
	return text
}

func firstStringArg(call *sitter.Node, source []byte) string {
	args := callArgs(call)
	if len(args) == 0 {
		return ""
	}
	if value, ok := stringLiteral(args[0], source); ok {
		return value
	}
	return strings.TrimSpace(args[0].Utf8Text(source))
}

func firstArgText(call *sitter.Node, source []byte) string {
	value, _ := firstArgValue(call, source)
	return value
}

func firstArgValue(call *sitter.Node, source []byte) (string, bool) {
	args := callArgs(call)
	if len(args) == 0 {
		return "", false
	}
	return nodeArgValue(args[0], source)
}

func nodeArgValue(node *sitter.Node, source []byte) (string, bool) {
	if value, ok := stringLiteral(node, source); ok {
		return value, true
	}
	return strings.TrimSpace(node.Utf8Text(source)), false
}

func stringLiteral(node *sitter.Node, source []byte) (string, bool) {
	if node == nil {
		return "", false
	}
	switch node.Kind() {
	case "interpreted_string_literal", "raw_string_literal", "string":
		value, err := strconv.Unquote(node.Utf8Text(source))
		return value, err == nil
	default:
		return "", false
	}
}

func firstNamedChildOfKind(node *sitter.Node, kind string) *sitter.Node {
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child.Kind() == kind {
			return child
		}
	}
	return nil
}

func walk(node *sitter.Node, visit func(*sitter.Node)) {
	if node == nil {
		return
	}
	visit(node)
	for i := uint(0); i < node.NamedChildCount(); i++ {
		walk(node.NamedChild(i), visit)
	}
}
