package hsmc

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type CPPFrontend struct{}

func NewCPPFrontend() CPPFrontend {
	return CPPFrontend{}
}

func (CPPFrontend) Language() Language { return LanguageCPP }

func (frontend CPPFrontend) Parse(ctx context.Context, input SourceInput) (*Program, error) {
	tree, err := parseTree(ctx, input, parserGrammarCPP)
	if err != nil {
		return nil, err
	}
	defer tree.Close()
	lowerer := cppLowerer{
		input: SourceInput{Path: input.Path, Data: input.Data},
		program: &Program{
			SourceLanguage: LanguageCPP,
			PackageName:    "main",
		},
	}
	root := tree.RootNode()
	lowerer.collectTopLevel(root)
	lowerer.collectModels(root)
	lowerer.filterReferencedFunctionGlobals()
	if len(lowerer.diagnostics) > 0 {
		return nil, lowerer.diagnostics
	}
	if len(lowerer.program.Models) == 0 {
		return nil, fmt.Errorf("no define model found in %s", input.Path)
	}
	return lowerer.program, nil
}

type cppLowerer struct {
	input         SourceInput
	program       *Program
	counters      map[string]int
	functions     map[string]cppFunctionDecl
	usedFunctions map[string]bool
	hsmUsing      bool
	diagnostics   Diagnostics
}

func (lowerer *cppLowerer) collectTopLevel(root *sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		child := root.NamedChild(i)
		switch child.Kind() {
		case "preproc_include":
			if imp, ok := lowerer.parseInclude(child); ok && !isHSMImportPath(imp.Path) {
				lowerer.program.Imports = append(lowerer.program.Imports, imp)
			}
		case "namespace_definition", "declaration_list":
			lowerer.collectTopLevel(child)
		case "struct_specifier":
			if event, ok := lowerer.parseEventDecl(child); ok {
				lowerer.program.Events = append(lowerer.program.Events, event)
				continue
			}
			lowerer.program.Globals = append(lowerer.program.Globals, lowerer.globalBlock(child))
		case "function_definition":
			block := lowerer.globalBlock(child)
			lowerer.registerFunctionDefinition(child, block.Range)
			lowerer.program.Globals = append(lowerer.program.Globals, block)
		case "class_specifier", "enum_specifier", "alias_declaration":
			lowerer.program.Globals = append(lowerer.program.Globals, lowerer.globalBlock(child))
		case "using_declaration":
			if cppIsHSMRuntimeUsing(child.Utf8Text(lowerer.input.Data)) {
				lowerer.hsmUsing = true
				continue
			}
			lowerer.program.Globals = append(lowerer.program.Globals, lowerer.globalBlock(child))
		case "declaration":
			if lowerer.collectDeclaration(child) {
				continue
			}
		}
	}
}

func (lowerer *cppLowerer) collectDeclaration(node *sitter.Node) bool {
	if !cppNodeContainsCall(node, lowerer.input.Data, "define") {
		if event, ok := lowerer.parseEventDecl(node); ok {
			lowerer.program.Events = append(lowerer.program.Events, event)
			return true
		}
		if lowerer.collectGeneratedFunctionDeclaration(node) {
			return true
		}
		lowerer.program.Globals = append(lowerer.program.Globals, lowerer.globalBlock(node))
		return true
	}
	declarators := directCPPInitDeclarators(node)
	if len(declarators) == 0 {
		return true
	}
	prefix := strings.TrimSpace(string(lowerer.input.Data[node.StartByte():declarators[0].StartByte()]))
	for _, declarator := range declarators {
		if cppNodeContainsCall(declarator, lowerer.input.Data, "define") {
			continue
		}
		block := lowerer.globalDeclaratorBlock(prefix, declarator)
		lowerer.registerFunctionDeclarator(declarator, block.Range)
		lowerer.program.Globals = append(lowerer.program.Globals, block)
	}
	return true
}

func (lowerer *cppLowerer) collectGeneratedFunctionDeclaration(node *sitter.Node) bool {
	declarators := directCPPInitDeclarators(node)
	if len(declarators) == 0 {
		return false
	}
	prefix := strings.TrimSpace(string(lowerer.input.Data[node.StartByte():declarators[0].StartByte()]))
	collected := false
	for _, declarator := range declarators {
		block := lowerer.globalDeclaratorBlock(prefix, declarator)
		before := len(lowerer.functions)
		lowerer.registerFunctionDeclarator(declarator, block.Range)
		if len(lowerer.functions) != before {
			lowerer.program.Globals = append(lowerer.program.Globals, block)
			collected = true
		}
	}
	return collected
}

func (lowerer *cppLowerer) collectModels(root *sitter.Node) {
	walk(root, func(node *sitter.Node) {
		if node.Kind() != "call_expression" || cppCanonicalCallName(node, lowerer.input.Data) != "Define" {
			return
		}
		if model, ok := lowerer.parseModel(node); ok {
			lowerer.program.Models = append(lowerer.program.Models, model)
		}
	})
}

func (lowerer *cppLowerer) parseModel(call *sitter.Node) (Model, bool) {
	args := cppCallArgs(call)
	if len(args) == 0 {
		return Model{}, false
	}
	name, ok := cppStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Model{}, false
	}
	model := Model{ID: lowerer.nextID("model"), Name: name, Range: lowerer.sourceRange(call)}
	for _, arg := range args[1:] {
		lowerer.applyModelPartial(&model, arg)
	}
	return model, true
}

func (lowerer *cppLowerer) applyModelPartial(model *Model, arg *sitter.Node) {
	call := cppAsCall(arg)
	if call == nil {
		return
	}
	name := cppCanonicalCallName(call, lowerer.input.Data)
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

func (lowerer *cppLowerer) parseAttribute(call *sitter.Node) (Attribute, bool) {
	args := cppCallArgs(call)
	if len(args) == 0 {
		return Attribute{}, false
	}
	name, ok := cppStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Attribute{}, false
	}
	attribute := Attribute{ID: lowerer.nextID("attribute"), Name: name, Language: LanguageCPP, Range: lowerer.sourceRange(call)}
	if typeArg := cppTemplateArgText(call, lowerer.input.Data); typeArg != "" {
		attribute.Type = typeArg
	}
	if len(args) == 2 && attribute.Type == "" {
		field, value := classifyAttributeSecondArg(LanguageCPP, args[1].Utf8Text(lowerer.input.Data))
		if field == "type" {
			attribute.Type = cppAttributeTypeArg(value)
		} else {
			attribute.HasDefault = true
			attribute.Default = value
		}
	} else if len(args) > 1 {
		attribute.HasDefault = true
		attribute.Default = strings.TrimSpace(args[1].Utf8Text(lowerer.input.Data))
	}
	return attribute, true
}

func cppAttributeTypeArg(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "std::type_identity<") && strings.HasSuffix(value, ">{}") {
		return strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "std::type_identity<"), ">{}"))
	}
	return value
}

func (lowerer *cppLowerer) parseOperation(ownerID string, call *sitter.Node) (Operation, bool) {
	args := cppCallArgs(call)
	if len(args) < 2 {
		return Operation{}, false
	}
	name, ok := cppStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Operation{}, false
	}
	operation := Operation{ID: lowerer.nextID("operation_decl"), Name: name, Range: lowerer.sourceRange(call)}
	behavior := lowerer.parseBehavior(ownerID, BehaviorOperation, args[1])
	lowerer.program.Behaviors = append(lowerer.program.Behaviors, behavior)
	operation.Behavior = &BehaviorRef{ID: behavior.ID, Kind: BehaviorOperation}
	return operation, true
}

func (lowerer *cppLowerer) parseState(ownerID string, call *sitter.Node) (State, bool) {
	args := cppCallArgs(call)
	if len(args) == 0 {
		return State{}, false
	}
	name, ok := cppStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return State{}, false
	}
	state := State{ID: lowerer.nextID("state"), Name: name, Kind: cppStateKind(cppCanonicalCallName(call, lowerer.input.Data)), Range: lowerer.sourceRange(call)}
	for _, arg := range args[1:] {
		partial := cppAsCall(arg)
		if partial == nil {
			continue
		}
		name := cppCanonicalCallName(partial, lowerer.input.Data)
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
			state.Behaviors = append(state.Behaviors, lowerer.parseBehaviors(state.ID, behaviorKind(cppCanonicalCallName(partial, lowerer.input.Data)), partial)...)
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

func (lowerer *cppLowerer) applyHistoryDefaultPartial(state *State, partial *sitter.Node, name string) bool {
	if state.Kind != StateKindShallowHistory && state.Kind != StateKindDeepHistory {
		return false
	}
	if len(state.Transitions) == 0 {
		state.Transitions = append(state.Transitions, Transition{OwnerID: state.ID, ID: lowerer.nextID("transition"), Range: lowerer.sourceRange(partial)})
	}
	transition := &state.Transitions[0]
	switch name {
	case "Target":
		transition.Target = cppFirstStringOrRaw(partial, lowerer.input.Data)
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

func (lowerer *cppLowerer) parseInitial(ownerID string, call *sitter.Node) Initial {
	initial := Initial{OwnerID: ownerID, ID: lowerer.nextID("initial"), Range: lowerer.sourceRange(call)}
	for _, arg := range cppCallArgs(call) {
		partial := cppAsCall(arg)
		if partial == nil {
			continue
		}
		name := cppCanonicalCallName(partial, lowerer.input.Data)
		switch name {
		case "Target":
			initial.Target = cppFirstStringOrRaw(partial, lowerer.input.Data)
		case "Effect":
			initial.Effects = append(initial.Effects, lowerer.parseBehaviors(ownerID, BehaviorEffect, partial)...)
		default:
			lowerer.addUnknownHSMPartialDiagnostic("initial", partial, name)
		}
	}
	return initial
}

func (lowerer *cppLowerer) parseTransition(ownerID string, call *sitter.Node) Transition {
	transition := Transition{OwnerID: ownerID, ID: lowerer.nextID("transition"), Range: lowerer.sourceRange(call)}
	for _, arg := range cppCallArgs(call) {
		partial := cppAsCall(arg)
		if partial == nil {
			continue
		}
		name := cppCanonicalCallName(partial, lowerer.input.Data)
		switch name {
		case "Source":
			transition.Source = cppFirstStringOrRaw(partial, lowerer.input.Data)
		case "Target":
			transition.Target = cppFirstStringOrRaw(partial, lowerer.input.Data)
		case "On":
			transition.Trigger = lowerer.parseOnTrigger(partial)
		case "OnSet":
			transition.Trigger = &Trigger{Kind: TriggerOnSet, Value: cppFirstStringOrRaw(partial, lowerer.input.Data), IsString: true, Language: LanguageCPP}
		case "OnCall":
			transition.Trigger = &Trigger{Kind: TriggerOnCall, Value: cppFirstStringOrRaw(partial, lowerer.input.Data), IsString: true, Language: LanguageCPP}
		case "After":
			transition.Trigger = lowerer.parseExpressionTrigger(ownerID, TriggerAfter, partial)
		case "Every":
			transition.Trigger = lowerer.parseExpressionTrigger(ownerID, TriggerEvery, partial)
		case "At":
			transition.Trigger = lowerer.parseExpressionTrigger(ownerID, TriggerAt, partial)
		case "When":
			transition.Trigger = lowerer.parseExpressionTrigger(ownerID, TriggerWhen, partial)
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

func (lowerer *cppLowerer) parseOnTrigger(call *sitter.Node) *Trigger {
	trigger := &Trigger{Kind: TriggerOn, Language: LanguageCPP}
	if eventType := cppTemplateArgText(call, lowerer.input.Data); eventType != "" {
		value := eventType
		if event := lowerer.eventBySymbol(eventType); event != nil {
			value = event.Name
		}
		trigger.Value = value
		trigger.Values = []TriggerValue{{Value: value, EventSymbol: eventType, Language: LanguageCPP}}
		return trigger
	}
	for _, arg := range cppCallArgs(call) {
		value, isString := cppNodeArgValue(arg, lowerer.input.Data)
		eventSymbol := ""
		if !isString {
			if event := lowerer.eventBySymbol(value); event != nil {
				eventSymbol = value
				value = event.Name
			} else if symbol, ok := eventMemberSymbol(value, "::name", ".name", ".Name"); ok {
				if event := lowerer.eventBySymbol(symbol); event != nil {
					eventSymbol = symbol
					value = event.Name
				}
			}
		}
		if trigger.Value == "" {
			trigger.Value = value
			trigger.IsString = isString
		}
		trigger.Values = append(trigger.Values, TriggerValue{Value: value, IsString: isString, EventSymbol: eventSymbol, Language: LanguageCPP})
	}
	if len(trigger.Values) <= 1 {
		if len(trigger.Values) == 1 && trigger.Values[0].EventSymbol == "" {
			trigger.Values = nil
		}
	}
	return trigger
}

func (lowerer *cppLowerer) parseExpressionTrigger(ownerID string, kind TriggerKind, call *sitter.Node) *Trigger {
	value, isString := cppFirstArgValue(call, lowerer.input.Data)
	trigger := &Trigger{Kind: kind, Value: value, IsString: isString, Language: LanguageCPP}
	args := cppCallArgs(call)
	if isString || len(args) == 0 {
		return trigger
	}
	behavior := lowerer.parseBehavior(ownerID, BehaviorTrigger, args[0])
	behavior.TriggerKind = kind
	lowerer.program.Behaviors = append(lowerer.program.Behaviors, behavior)
	trigger.Expr = &BehaviorRef{ID: behavior.ID, Kind: BehaviorTrigger}
	return trigger
}

func (lowerer *cppLowerer) parseBehaviors(ownerID string, kind BehaviorKind, call *sitter.Node) []BehaviorRef {
	var refs []BehaviorRef
	for _, arg := range cppCallArgs(call) {
		if operation, ok := cppStringLiteral(arg, lowerer.input.Data); ok {
			refs = append(refs, BehaviorRef{Kind: kind, Operation: operation})
			continue
		}
		behavior := lowerer.parseBehavior(ownerID, kind, arg)
		lowerer.program.Behaviors = append(lowerer.program.Behaviors, behavior)
		refs = append(refs, BehaviorRef{ID: behavior.ID, Kind: kind})
	}
	return refs
}

func (lowerer *cppLowerer) parseBehavior(ownerID string, kind BehaviorKind, node *sitter.Node) Behavior {
	inline := node.Kind() == "lambda_expression"
	if resolved := lowerer.referencedBehaviorFunction(node, kind); resolved != nil {
		node = resolved
	}
	code := strings.TrimSpace(node.Utf8Text(lowerer.input.Data))
	behavior := Behavior{
		ID:             lowerer.nextID(string(kind)),
		OwnerID:        ownerID,
		Kind:           kind,
		Inline:         inline,
		SourceLanguage: LanguageCPP,
		Code:           code,
		Range:          lowerer.sourceRange(node),
	}
	if node.Kind() == "lambda_expression" || node.Kind() == "function_definition" {
		if body := firstNamedChildOfKind(node, "compound_statement"); body != nil {
			behavior.Body = trimBlock(body.Utf8Text(lowerer.input.Data))
			behavior.Signature = strings.TrimSpace(string(lowerer.input.Data[node.StartByte():body.StartByte()]))
		}
	} else if node.Kind() == "identifier" || node.Kind() == "qualified_identifier" {
		behavior.Body = cppCallableReferenceBody(kind, strings.TrimSpace(node.Utf8Text(lowerer.input.Data)))
	}
	return behavior
}

type cppFunctionDecl struct {
	node        *sitter.Node
	globalRange SourceRange
}

func (lowerer *cppLowerer) registerFunctionDeclarator(node *sitter.Node, globalRange SourceRange) {
	name := cppDeclaratorName(node, lowerer.input.Data)
	if name == "" || !isGeneratedCPPBehaviorFunctionName(name) {
		return
	}
	value := cppDeclaratorValue(node)
	if value == nil || value.Kind() != "lambda_expression" {
		return
	}
	if lowerer.functions == nil {
		lowerer.functions = map[string]cppFunctionDecl{}
	}
	lowerer.functions[name] = cppFunctionDecl{node: value, globalRange: globalRange}
}

func (lowerer *cppLowerer) registerFunctionDefinition(node *sitter.Node, globalRange SourceRange) {
	name := cppFunctionDefinitionName(node, lowerer.input.Data)
	if name == "" {
		return
	}
	if lowerer.functions == nil {
		lowerer.functions = map[string]cppFunctionDecl{}
	}
	lowerer.functions[name] = cppFunctionDecl{node: node, globalRange: globalRange}
}

func (lowerer *cppLowerer) referencedBehaviorFunction(node *sitter.Node, kind BehaviorKind) *sitter.Node {
	if node == nil || (node.Kind() != "identifier" && node.Kind() != "qualified_identifier") {
		return nil
	}
	name := strings.TrimSpace(node.Utf8Text(lowerer.input.Data))
	if index := strings.LastIndex(name, "::"); index >= 0 {
		name = name[index+2:]
	}
	if isGeneratedCPPBehaviorFunctionName(name) && !isGeneratedCPPBehaviorFunctionNameForKind(name, kind) {
		return nil
	}
	decl, ok := lowerer.functions[name]
	if !ok {
		return nil
	}
	if !isGeneratedCPPBehaviorFunctionNameForKind(name, kind) && !lowerer.functionDeclLooksLikeBehavior(decl) {
		return nil
	}
	if lowerer.usedFunctions == nil {
		lowerer.usedFunctions = map[string]bool{}
	}
	lowerer.usedFunctions[name] = true
	return decl.node
}

func (lowerer *cppLowerer) functionDeclLooksLikeBehavior(decl cppFunctionDecl) bool {
	if decl.node == nil {
		return false
	}
	body := firstNamedChildOfKind(decl.node, "compound_statement")
	if body == nil {
		return false
	}
	signature := string(lowerer.input.Data[decl.node.StartByte():body.StartByte()])
	return strings.Contains(signature, "Event")
}

func (lowerer *cppLowerer) filterReferencedFunctionGlobals() {
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

func isGeneratedCPPBehaviorFunctionName(name string) bool {
	for _, prefix := range []string{"entry_", "exit_", "activity_", "guard_", "effect_", "trigger_", "operation_"} {
		if hasNumericSuffix(name, prefix) {
			return true
		}
	}
	return false
}

func isGeneratedCPPBehaviorFunctionNameForKind(name string, kind BehaviorKind) bool {
	prefix := string(kind) + "_"
	if kind == BehaviorOperation {
		prefix = "operation_"
	}
	return hasNumericSuffix(name, prefix)
}

func hasNumericSuffix(name string, prefix string) bool {
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

func cppCallableReferenceBody(kind BehaviorKind, name string) string {
	if name == "" {
		if kind == BehaviorGuard {
			return "return false;"
		}
		return ""
	}
	call := fmt.Sprintf("%s(signal, instance, event)", name)
	if kind == BehaviorGuard || kind == BehaviorTrigger {
		return "return " + call + ";"
	}
	return call + ";"
}

func (lowerer *cppLowerer) parseDefers(call *sitter.Node) []string {
	var defers []string
	if eventType := cppTemplateArgText(call, lowerer.input.Data); eventType != "" {
		if event := lowerer.eventBySymbol(eventType); event != nil {
			defers = append(defers, event.Name)
			return defers
		}
		defers = append(defers, eventType)
	}
	for _, arg := range cppCallArgs(call) {
		value, isString := cppNodeArgValue(arg, lowerer.input.Data)
		if !isString {
			if event := lowerer.eventBySymbol(value); event != nil {
				defers = append(defers, event.Name)
				continue
			}
			if symbol, ok := eventMemberSymbol(value, "::name", ".name", ".Name"); ok {
				if event := lowerer.eventBySymbol(symbol); event != nil {
					defers = append(defers, event.Name)
					continue
				}
			}
		}
		defers = append(defers, value)
	}
	return defers
}

func (lowerer *cppLowerer) parseEventDecl(node *sitter.Node) (EventDecl, bool) {
	if event, ok := lowerer.parseConstEventDecl(node); ok {
		return event, true
	}
	var event EventDecl
	walk(node, func(child *sitter.Node) {
		if event.Symbol != "" || child.Kind() != "struct_specifier" {
			return
		}
		if child.NamedChildCount() == 0 {
			return
		}
		nameNode := child.NamedChild(0)
		if nameNode.Kind() != "type_identifier" {
			return
		}
		symbol := nameNode.Utf8Text(lowerer.input.Data)
		name := ""
		walk(child, func(member *sitter.Node) {
			if member.Kind() == "string_literal" && name == "" {
				if value, ok := cppStringLiteral(member, lowerer.input.Data); ok {
					name = value
				}
			}
		})
		if name == "" {
			return
		}
		event = EventDecl{ID: lowerer.nextID("event"), Symbol: symbol, Name: name, Language: LanguageCPP, Range: lowerer.sourceRange(child)}
	})
	return event, event.Symbol != ""
}

func (lowerer *cppLowerer) parseConstEventDecl(node *sitter.Node) (EventDecl, bool) {
	text := strings.TrimSpace(node.Utf8Text(lowerer.input.Data))
	if !strings.Contains(text, "=") || !strings.Contains(text, "constexpr") {
		return EventDecl{}, false
	}
	parts := strings.SplitN(text, "=", 2)
	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(strings.TrimSuffix(parts[1], ";"))
	name, ok := cppStringLiteralText(right)
	if !ok {
		return EventDecl{}, false
	}
	fields := strings.Fields(left)
	if len(fields) == 0 {
		return EventDecl{}, false
	}
	symbol := fields[len(fields)-1]
	if symbol == "" || strings.ContainsAny(symbol, "*&") {
		return EventDecl{}, false
	}
	return EventDecl{ID: lowerer.nextID("event"), Symbol: symbol, Name: name, Language: LanguageCPP, Range: lowerer.sourceRange(node)}, true
}

func (lowerer *cppLowerer) eventBySymbol(symbol string) *EventDecl {
	for index := range lowerer.program.Events {
		if lowerer.program.Events[index].Symbol == symbol {
			return &lowerer.program.Events[index]
		}
	}
	return nil
}

func (lowerer *cppLowerer) parseInclude(node *sitter.Node) (Import, bool) {
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child.Kind() == "string_literal" {
			if value, ok := cppStringLiteral(child, lowerer.input.Data); ok {
				return Import{Path: value, Language: LanguageCPP, Range: lowerer.sourceRange(node)}, true
			}
		}
		if child.Kind() == "system_lib_string" {
			value := strings.Trim(child.Utf8Text(lowerer.input.Data), "<>")
			return Import{Path: value, Language: LanguageCPP, Range: lowerer.sourceRange(node)}, true
		}
	}
	return Import{}, false
}

func cppIsHSMRuntimeUsing(code string) bool {
	code = strings.TrimSpace(code)
	code = strings.TrimSuffix(code, ";")
	code = strings.TrimSpace(code)
	return code == "using namespace hsm" || code == "using hsm"
}

func (lowerer *cppLowerer) globalBlock(node *sitter.Node) CodeBlock {
	code := node.Utf8Text(lowerer.input.Data)
	switch node.Kind() {
	case "struct_specifier", "class_specifier", "enum_specifier":
		code = strings.TrimRight(code, " \t\r\n")
		if !strings.HasSuffix(code, ";") {
			code += ";"
		}
	}
	return CodeBlock{ID: lowerer.nextID("global"), Language: LanguageCPP, Code: code, Range: lowerer.sourceRange(node)}
}

func (lowerer *cppLowerer) globalDeclaratorBlock(prefix string, declarator *sitter.Node) CodeBlock {
	code := strings.TrimSpace(prefix)
	if code != "" {
		code += " "
	}
	code += strings.TrimSpace(declarator.Utf8Text(lowerer.input.Data))
	if !strings.HasSuffix(code, ";") {
		code += ";"
	}
	return CodeBlock{ID: lowerer.nextID("global"), Language: LanguageCPP, Code: code, Range: lowerer.sourceRange(declarator)}
}

func (lowerer *cppLowerer) sourceRange(node *sitter.Node) SourceRange {
	start := node.StartPosition()
	end := node.EndPosition()
	return SourceRange{File: lowerer.input.Path, StartByte: node.StartByte(), EndByte: node.EndByte(), StartLine: uint(start.Row) + 1, EndLine: uint(end.Row) + 1}
}

func (lowerer *cppLowerer) addUnknownHSMPartialDiagnostic(context string, call *sitter.Node, name string) {
	if !lowerer.isHSMRuntimeCall(call) {
		return
	}
	lowerer.diagnostics = append(lowerer.diagnostics, Diagnostic{
		Message: fmt.Sprintf("unknown or unsupported hsm::%s partial in %s context", name, context),
		Range:   lowerer.sourceRange(call),
	})
}

func (lowerer *cppLowerer) isHSMRuntimeCall(call *sitter.Node) bool {
	if call == nil || call.NamedChildCount() == 0 {
		return false
	}
	function := call.NamedChild(0)
	if function.Kind() == "template_function" && function.NamedChildCount() > 0 {
		function = function.NamedChild(0)
	}
	text := strings.TrimSpace(function.Utf8Text(lowerer.input.Data))
	if index := strings.LastIndex(text, "::"); index >= 0 {
		return strings.TrimSpace(text[:index]) == "hsm"
	}
	return lowerer.hsmUsing
}

func (lowerer *cppLowerer) nextID(prefix string) string {
	if lowerer.counters == nil {
		lowerer.counters = map[string]int{}
	}
	lowerer.counters[prefix]++
	return fmt.Sprintf("%s_%d", prefix, lowerer.counters[prefix])
}

func directCPPInitDeclarators(node *sitter.Node) []*sitter.Node {
	var declarators []*sitter.Node
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child.Kind() == "init_declarator" {
			declarators = append(declarators, child)
		}
	}
	return declarators
}

func cppDeclaratorName(node *sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}
	text := node.Utf8Text(source)
	left, _, ok := strings.Cut(text, "=")
	if !ok {
		return ""
	}
	left = strings.TrimSpace(left)
	if left == "" {
		return ""
	}
	fields := strings.FieldsFunc(left, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '*' || r == '&'
	})
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimSpace(fields[len(fields)-1])
}

func cppFunctionDefinitionName(node *sitter.Node, source []byte) string {
	body := firstNamedChildOfKind(node, "compound_statement")
	if body == nil {
		return ""
	}
	signature := strings.TrimSpace(string(source[node.StartByte():body.StartByte()]))
	left, _, ok := strings.Cut(signature, "(")
	if !ok {
		return ""
	}
	left = strings.TrimSpace(left)
	fields := strings.FieldsFunc(left, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '*' || r == '&'
	})
	if len(fields) == 0 {
		return ""
	}
	name := strings.TrimSpace(fields[len(fields)-1])
	if index := strings.LastIndex(name, "::"); index >= 0 {
		name = name[index+2:]
	}
	return name
}

func cppDeclaratorValue(node *sitter.Node) *sitter.Node {
	var value *sitter.Node
	walk(node, func(child *sitter.Node) {
		if value == nil && child.Kind() == "lambda_expression" {
			value = child
		}
	})
	return value
}

func cppAsCall(node *sitter.Node) *sitter.Node {
	if node != nil && node.Kind() == "call_expression" {
		return node
	}
	return nil
}

func cppCallArgs(call *sitter.Node) []*sitter.Node {
	args := firstNamedChildOfKind(call, "argument_list")
	if args == nil {
		return nil
	}
	var result []*sitter.Node
	for i := uint(0); i < args.NamedChildCount(); i++ {
		result = append(result, args.NamedChild(i))
	}
	return result
}

func cppCallName(call *sitter.Node, source []byte) string {
	if call == nil || call.NamedChildCount() == 0 {
		return ""
	}
	function := call.NamedChild(0)
	if function.Kind() == "template_function" && function.NamedChildCount() > 0 {
		return function.NamedChild(0).Utf8Text(source)
	}
	text := function.Utf8Text(source)
	if index := strings.LastIndex(text, "::"); index >= 0 {
		return text[index+2:]
	}
	return text
}

func cppCanonicalCallName(call *sitter.Node, source []byte) string {
	switch strings.ToLower(cppCallName(call, source)) {
	case "define":
		return "Define"
	case "initial":
		return "Initial"
	case "state":
		return "State"
	case "final":
		return "Final"
	case "choice":
		return "Choice"
	case "shallow_history", "shallowhistory":
		return "ShallowHistory"
	case "deep_history", "deephistory":
		return "DeepHistory"
	case "transition":
		return "Transition"
	case "source":
		return "Source"
	case "target":
		return "Target"
	case "on":
		return "On"
	case "on_set", "onset":
		return "OnSet"
	case "on_call", "oncall":
		return "OnCall"
	case "after", "on_timeout":
		return "After"
	case "every", "on_interval":
		return "Every"
	case "at", "on_timepoint":
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
		return cppCallName(call, source)
	}
}

func cppTemplateArgText(call *sitter.Node, source []byte) string {
	if call == nil || call.NamedChildCount() == 0 {
		return ""
	}
	function := call.NamedChild(0)
	if function.Kind() != "template_function" {
		return ""
	}
	args := firstNamedChildOfKind(function, "template_argument_list")
	if args == nil || args.NamedChildCount() == 0 {
		return ""
	}
	return strings.TrimSpace(args.NamedChild(0).Utf8Text(source))
}

func cppFirstStringOrRaw(call *sitter.Node, source []byte) string {
	value, _ := cppFirstArgValue(call, source)
	return value
}

func cppFirstArgValue(call *sitter.Node, source []byte) (string, bool) {
	args := cppCallArgs(call)
	if len(args) == 0 {
		return "", false
	}
	return cppNodeArgValue(args[0], source)
}

func cppNodeArgValue(node *sitter.Node, source []byte) (string, bool) {
	if value, ok := cppStringLiteral(node, source); ok {
		return value, true
	}
	return strings.TrimSpace(node.Utf8Text(source)), false
}

func cppStringLiteral(node *sitter.Node, source []byte) (string, bool) {
	if node == nil || node.Kind() != "string_literal" {
		return "", false
	}
	text := strings.TrimSpace(node.Utf8Text(source))
	if value, ok := cppStringLiteralText(text); ok {
		return value, true
	}
	if node.NamedChildCount() > 0 && node.NamedChild(0).Kind() == "string_content" {
		return node.NamedChild(0).Utf8Text(source), true
	}
	return "", false
}

func cppStringLiteralText(text string) (string, bool) {
	text = strings.TrimSpace(text)
	if value, err := strconv.Unquote(text); err == nil {
		return value, true
	}
	return strings.Trim(text, `"`), strings.HasPrefix(text, `"`) && strings.HasSuffix(text, `"`)
}

func cppNodeContainsCall(node *sitter.Node, source []byte, name string) bool {
	found := false
	walk(node, func(child *sitter.Node) {
		if child.Kind() == "call_expression" && strings.EqualFold(cppCallName(child, source), name) {
			found = true
		}
	})
	return found
}

func cppStateKind(name string) StateKind {
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
