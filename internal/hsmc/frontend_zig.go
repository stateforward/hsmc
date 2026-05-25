package hsmc

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type ZigFrontend struct{}

func NewZigFrontend() ZigFrontend {
	return ZigFrontend{}
}

func (ZigFrontend) Language() Language { return LanguageZig }

func (frontend ZigFrontend) Parse(ctx context.Context, input SourceInput) (*Program, error) {
	tree, err := parseTree(ctx, input, parserGrammarZig)
	if err != nil {
		return nil, err
	}
	defer tree.Close()
	lowerer := zigLowerer{
		input: SourceInput{Path: input.Path, Data: input.Data},
		program: &Program{
			SourceLanguage: LanguageZig,
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

type zigLowerer struct {
	input         SourceInput
	program       *Program
	counters      map[string]int
	functions     map[string]zigFunctionDecl
	usedFunctions map[string]bool
	hsmAliases    map[string]bool
	diagnostics   Diagnostics
}

func (lowerer *zigLowerer) collectTopLevel(root *sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		child := root.NamedChild(i)
		switch child.Kind() {
		case "variable_declaration":
			if imp, ok := lowerer.parseImport(child); ok {
				if isHSMImportPath(imp.Path) {
					lowerer.recordHSMImport(imp)
				} else if imp.Path != "std" {
					lowerer.program.Imports = append(lowerer.program.Imports, imp)
				}
				continue
			}
			if zigNodeContainsCall(child, lowerer.input.Data, "define") {
				continue
			}
			if event, ok := lowerer.parseEventDecl(child); ok {
				lowerer.program.Events = append(lowerer.program.Events, event)
				continue
			}
			lowerer.program.Globals = append(lowerer.program.Globals, lowerer.globalBlock(child))
		case "function_declaration", "test_declaration", "using_namespace_declaration":
			block := lowerer.globalBlock(child)
			if child.Kind() == "function_declaration" {
				lowerer.registerFunctionDecl(child, block.Range)
			}
			lowerer.program.Globals = append(lowerer.program.Globals, block)
		}
	}
}

func (lowerer *zigLowerer) collectModels(root *sitter.Node) {
	walk(root, func(node *sitter.Node) {
		if node.Kind() != "call_expression" || zigCanonicalCallName(node, lowerer.input.Data) != "Define" {
			return
		}
		if model, ok := lowerer.parseModel(node); ok {
			lowerer.program.Models = append(lowerer.program.Models, model)
		}
	})
}

func (lowerer *zigLowerer) parseModel(call *sitter.Node) (Model, bool) {
	args := zigCallArgs(call)
	if len(args) == 0 {
		return Model{}, false
	}
	name, ok := zigStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Model{}, false
	}
	model := Model{ID: lowerer.nextID("model"), Name: name, Range: lowerer.sourceRange(call)}
	for _, partial := range zigPartialCalls(args[1:]...) {
		lowerer.applyModelPartial(&model, partial)
	}
	return model, true
}

func (lowerer *zigLowerer) applyModelPartial(model *Model, call *sitter.Node) {
	name := zigCanonicalCallName(call, lowerer.input.Data)
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

func (lowerer *zigLowerer) parseAttribute(call *sitter.Node) (Attribute, bool) {
	args := zigCallArgs(call)
	if len(args) == 0 {
		return Attribute{}, false
	}
	name, ok := zigStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Attribute{}, false
	}
	attribute := Attribute{ID: lowerer.nextID("attribute"), Name: name, Language: LanguageZig, Range: lowerer.sourceRange(call)}
	if len(args) == 2 {
		field, value := classifyAttributeSecondArg(LanguageZig, args[1].Utf8Text(lowerer.input.Data))
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

func (lowerer *zigLowerer) parseOperation(ownerID string, call *sitter.Node) (Operation, bool) {
	args := zigCallArgs(call)
	if len(args) < 2 {
		return Operation{}, false
	}
	name, ok := zigStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Operation{}, false
	}
	operation := Operation{ID: lowerer.nextID("operation_decl"), Name: name, Range: lowerer.sourceRange(call)}
	behavior := lowerer.parseBehavior(ownerID, BehaviorOperation, args[1])
	lowerer.program.Behaviors = append(lowerer.program.Behaviors, behavior)
	operation.Behavior = &BehaviorRef{ID: behavior.ID, Kind: BehaviorOperation}
	return operation, true
}

func (lowerer *zigLowerer) parseState(ownerID string, call *sitter.Node) (State, bool) {
	args := zigCallArgs(call)
	if len(args) == 0 {
		return State{}, false
	}
	name, ok := zigStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return State{}, false
	}
	state := State{ID: lowerer.nextID("state"), Name: name, Kind: zigStateKind(zigCanonicalCallName(call, lowerer.input.Data)), Range: lowerer.sourceRange(call)}
	if state.Kind == StateKindShallowHistory || state.Kind == StateKindDeepHistory {
		for _, partial := range zigPartialCalls(args[1:]...) {
			name := zigCanonicalCallName(partial, lowerer.input.Data)
			switch name {
			case "Transition":
				state.Transitions = append(state.Transitions, lowerer.parseTransition(state.ID, partial))
			case "Target", "Guard", "Effect":
				lowerer.applyHistoryDefaultPartial(&state, partial, name)
			default:
				lowerer.addUnknownHSMPartialDiagnostic("state", partial, name)
			}
		}
		_ = ownerID
		return state, true
	}
	for _, partial := range zigPartialCalls(args[1:]...) {
		name := zigCanonicalCallName(partial, lowerer.input.Data)
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
			state.Behaviors = append(state.Behaviors, lowerer.parseBehaviors(state.ID, behaviorKind(zigCanonicalCallName(partial, lowerer.input.Data)), partial)...)
		case "Defer":
			state.Defers = append(state.Defers, lowerer.parseDefers(partial)...)
		default:
			lowerer.addUnknownHSMPartialDiagnostic("state", partial, name)
		}
	}
	_ = ownerID
	return state, true
}

func (lowerer *zigLowerer) applyHistoryDefaultPartial(state *State, partial *sitter.Node, name string) bool {
	if state.Kind != StateKindShallowHistory && state.Kind != StateKindDeepHistory {
		return false
	}
	if len(state.Transitions) == 0 {
		state.Transitions = append(state.Transitions, Transition{OwnerID: state.ID, ID: lowerer.nextID("transition"), Range: lowerer.sourceRange(partial)})
	}
	transition := &state.Transitions[0]
	switch name {
	case "Target":
		transition.Target = zigFirstStringOrRaw(partial, lowerer.input.Data)
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

func (lowerer *zigLowerer) parseInitial(ownerID string, call *sitter.Node) Initial {
	initial := Initial{OwnerID: ownerID, ID: lowerer.nextID("initial"), Range: lowerer.sourceRange(call)}
	for _, arg := range zigCallArgs(call) {
		partial := zigAsCall(arg)
		if partial == nil {
			continue
		}
		name := zigCanonicalCallName(partial, lowerer.input.Data)
		switch name {
		case "Target":
			initial.Target = zigFirstStringOrRaw(partial, lowerer.input.Data)
		case "Effect":
			initial.Effects = append(initial.Effects, lowerer.parseBehaviors(ownerID, BehaviorEffect, partial)...)
		default:
			lowerer.addUnknownHSMPartialDiagnostic("initial", partial, name)
		}
	}
	return initial
}

func (lowerer *zigLowerer) parseTransition(ownerID string, call *sitter.Node) Transition {
	transition := Transition{OwnerID: ownerID, ID: lowerer.nextID("transition"), Range: lowerer.sourceRange(call)}
	for _, partial := range zigPartialCalls(zigCallArgs(call)...) {
		name := zigCanonicalCallName(partial, lowerer.input.Data)
		switch name {
		case "Source":
			transition.Source = zigFirstStringOrRaw(partial, lowerer.input.Data)
		case "Target":
			transition.Target = zigFirstStringOrRaw(partial, lowerer.input.Data)
		case "On":
			transition.Trigger = lowerer.parseOnTrigger(partial)
		case "OnSet":
			transition.Trigger = &Trigger{Kind: TriggerOnSet, Value: zigFirstStringOrRaw(partial, lowerer.input.Data), IsString: true, Language: LanguageZig}
		case "OnCall":
			transition.Trigger = &Trigger{Kind: TriggerOnCall, Value: zigFirstStringOrRaw(partial, lowerer.input.Data), IsString: true, Language: LanguageZig}
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

func (lowerer *zigLowerer) parseOnTrigger(call *sitter.Node) *Trigger {
	trigger := &Trigger{Kind: TriggerOn, Language: LanguageZig}
	for _, arg := range zigCallArgs(call) {
		value, isString := zigNodeArgValue(arg, lowerer.input.Data)
		eventSymbol := ""
		if !isString && lowerer.eventBySymbol(value) != nil {
			eventSymbol = value
		}
		if trigger.Value == "" {
			trigger.Value = value
			trigger.IsString = isString
		}
		trigger.Values = append(trigger.Values, TriggerValue{Value: value, IsString: isString, EventSymbol: eventSymbol, Language: LanguageZig})
	}
	if len(trigger.Values) <= 1 && (len(trigger.Values) == 0 || trigger.Values[0].EventSymbol == "") {
		trigger.Values = nil
	}
	return trigger
}

func (lowerer *zigLowerer) parseExpressionTrigger(ownerID string, kind TriggerKind, call *sitter.Node) *Trigger {
	value, isString := zigFirstArgValue(call, lowerer.input.Data)
	trigger := &Trigger{Kind: kind, Value: value, IsString: isString, Language: LanguageZig}
	args := zigCallArgs(call)
	if isString || len(args) == 0 {
		return trigger
	}
	behavior := lowerer.parseBehavior(ownerID, BehaviorTrigger, args[0])
	behavior.TriggerKind = kind
	lowerer.program.Behaviors = append(lowerer.program.Behaviors, behavior)
	trigger.Expr = &BehaviorRef{ID: behavior.ID, Kind: BehaviorTrigger}
	return trigger
}

func (lowerer *zigLowerer) parseBehaviors(ownerID string, kind BehaviorKind, call *sitter.Node) []BehaviorRef {
	var refs []BehaviorRef
	for _, arg := range zigCallArgs(call) {
		if operation, ok := zigStringLiteral(arg, lowerer.input.Data); ok {
			refs = append(refs, BehaviorRef{Kind: kind, Operation: operation})
			continue
		}
		for _, behaviorArg := range zigBehaviorArgs(arg) {
			behavior := lowerer.parseBehavior(ownerID, kind, behaviorArg)
			lowerer.program.Behaviors = append(lowerer.program.Behaviors, behavior)
			refs = append(refs, BehaviorRef{ID: behavior.ID, Kind: kind})
		}
	}
	return refs
}

func (lowerer *zigLowerer) parseBehavior(ownerID string, kind BehaviorKind, node *sitter.Node) Behavior {
	if resolved := lowerer.referencedBehaviorFunction(node, kind); resolved != nil {
		return lowerer.parseRegisteredBehavior(ownerID, kind, *resolved)
	}
	code := strings.TrimSpace(node.Utf8Text(lowerer.input.Data))
	behavior := Behavior{
		ID:             lowerer.nextID(string(kind)),
		OwnerID:        ownerID,
		Kind:           kind,
		Inline:         node.Kind() == "function_declaration",
		SourceLanguage: LanguageZig,
		Code:           code,
		Range:          lowerer.sourceRange(node),
	}
	if node.Kind() == "function_declaration" {
		if body := node.ChildByFieldName("body"); body != nil {
			behavior.Body = trimBlock(body.Utf8Text(lowerer.input.Data))
			behavior.Signature = strings.TrimSpace(string(lowerer.input.Data[node.StartByte():body.StartByte()]))
		}
	} else if node.Kind() == "identifier" || node.Kind() == "field_expression" {
		behavior.Body = zigCallableReferenceBody(kind, strings.TrimSpace(node.Utf8Text(lowerer.input.Data)))
	}
	return behavior
}

type zigFunctionDecl struct {
	name        string
	node        *sitter.Node
	body        *sitter.Node
	globalRange SourceRange
}

func (lowerer *zigLowerer) parseRegisteredBehavior(ownerID string, kind BehaviorKind, decl zigFunctionDecl) Behavior {
	body := ""
	signature := ""
	if decl.body != nil {
		body = trimBlock(decl.body.Utf8Text(lowerer.input.Data))
		signature = strings.TrimSpace(string(lowerer.input.Data[decl.node.StartByte():decl.body.StartByte()]))
	}
	return Behavior{
		ID:             lowerer.nextID(string(kind)),
		OwnerID:        ownerID,
		Kind:           kind,
		SourceLanguage: LanguageZig,
		Body:           body,
		Code:           strings.TrimSpace(decl.node.Utf8Text(lowerer.input.Data)),
		Signature:      signature,
		Range:          lowerer.sourceRange(decl.node),
	}
}

func (lowerer *zigLowerer) registerFunctionDecl(node *sitter.Node, globalRange SourceRange) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		nameNode = firstNamedChildOfKind(node, "identifier")
	}
	if nameNode == nil {
		return
	}
	name := nameNode.Utf8Text(lowerer.input.Data)
	body := node.ChildByFieldName("body")
	if body == nil {
		body = firstNamedChildOfKind(node, "block")
	}
	if lowerer.functions == nil {
		lowerer.functions = map[string]zigFunctionDecl{}
	}
	lowerer.functions[name] = zigFunctionDecl{name: name, node: node, body: body, globalRange: globalRange}
}

func (lowerer *zigLowerer) referencedBehaviorFunction(node *sitter.Node, kind BehaviorKind) *zigFunctionDecl {
	if node == nil || (node.Kind() != "identifier" && node.Kind() != "field_expression") {
		return nil
	}
	name := strings.TrimSpace(node.Utf8Text(lowerer.input.Data))
	if index := strings.LastIndex(name, "."); index >= 0 {
		name = name[index+1:]
	}
	if isZigGeneratedBehaviorFunctionNameForAnyKind(name) && !isZigGeneratedBehaviorFunctionName(name, kind) {
		return nil
	}
	decl, ok := lowerer.functions[name]
	if !ok {
		return nil
	}
	if !isZigGeneratedBehaviorFunctionName(name, kind) && !lowerer.functionDeclLooksLikeBehavior(decl) {
		return nil
	}
	if lowerer.usedFunctions == nil {
		lowerer.usedFunctions = map[string]bool{}
	}
	lowerer.usedFunctions[name] = true
	return &decl
}

func (lowerer *zigLowerer) functionDeclLooksLikeBehavior(decl zigFunctionDecl) bool {
	if decl.node == nil || decl.body == nil {
		return false
	}
	signature := string(lowerer.input.Data[decl.node.StartByte():decl.body.StartByte()])
	return strings.Contains(signature, "hsm.Event")
}

func (lowerer *zigLowerer) filterReferencedFunctionGlobals() {
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

func isZigGeneratedBehaviorFunctionNameForAnyKind(name string) bool {
	for _, kind := range []BehaviorKind{
		BehaviorEntry,
		BehaviorExit,
		BehaviorActivity,
		BehaviorGuard,
		BehaviorEffect,
		BehaviorTrigger,
		BehaviorOperation,
	} {
		if isZigGeneratedBehaviorFunctionName(name, kind) {
			return true
		}
	}
	return false
}

func isZigGeneratedBehaviorFunctionName(name string, kind BehaviorKind) bool {
	prefix := string(kind)
	if kind == BehaviorOperation {
		prefix = "operation"
	}
	suffix := strings.TrimPrefix(name, prefix+"_")
	if suffix == name || suffix == "" {
		return false
	}
	for _, r := range suffix {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func zigCallableReferenceBody(kind BehaviorKind, name string) string {
	if name == "" {
		if kind == BehaviorGuard || kind == BehaviorTrigger {
			return "return false;"
		}
		return ""
	}
	call := fmt.Sprintf("%s(ctx, inst, event)", name)
	if kind == BehaviorGuard || kind == BehaviorTrigger {
		return "return " + call + ";"
	}
	return call + ";"
}

func zigBehaviorArgs(node *sitter.Node) []*sitter.Node {
	if node == nil {
		return nil
	}
	if node.Kind() == "anonymous_struct_initializer" || node.Kind() == "initializer_list" {
		var result []*sitter.Node
		for _, call := range zigPartialCalls(node) {
			result = append(result, call)
		}
		if len(result) > 0 {
			return result
		}
	}
	return []*sitter.Node{node}
}

func (lowerer *zigLowerer) parseDefers(call *sitter.Node) []string {
	var defers []string
	for _, arg := range zigCallArgs(call) {
		for _, eventNode := range zigInitializerElements(arg) {
			value, isString := zigNodeArgValue(eventNode, lowerer.input.Data)
			if value != "" {
				if !isString {
					if event := lowerer.eventBySymbol(value); event != nil {
						defers = append(defers, event.Name)
						continue
					}
				}
				defers = append(defers, value)
			}
		}
	}
	return defers
}

func (lowerer *zigLowerer) parseEventDecl(node *sitter.Node) (EventDecl, bool) {
	nameNode := zigDeclarationName(node)
	valueNode := zigDeclarationValue(node)
	if nameNode == nil || valueNode == nil || valueNode.Kind() != "string" {
		return EventDecl{}, false
	}
	name, ok := zigStringLiteral(valueNode, lowerer.input.Data)
	if !ok {
		return EventDecl{}, false
	}
	symbol := nameNode.Utf8Text(lowerer.input.Data)
	return EventDecl{ID: lowerer.nextID("event"), Symbol: symbol, Name: name, Language: LanguageZig, Range: lowerer.sourceRange(node)}, true
}

func (lowerer *zigLowerer) parseImport(node *sitter.Node) (Import, bool) {
	nameNode := zigDeclarationName(node)
	valueNode := zigDeclarationValue(node)
	if nameNode == nil || valueNode == nil || valueNode.Kind() != "builtin_function" {
		return Import{}, false
	}
	if valueNode.NamedChildCount() == 0 || valueNode.NamedChild(0).Utf8Text(lowerer.input.Data) != "@import" {
		return Import{}, false
	}
	var path string
	walk(valueNode, func(child *sitter.Node) {
		if path != "" || child.Kind() != "string" {
			return
		}
		if value, ok := zigStringLiteral(child, lowerer.input.Data); ok {
			path = value
		}
	})
	if path == "" {
		return Import{}, false
	}
	alias := nameNode.Utf8Text(lowerer.input.Data)
	return Import{Path: path, Alias: alias, LocalNames: []string{alias}, Language: LanguageZig, Range: lowerer.sourceRange(node)}, true
}

func (lowerer *zigLowerer) recordHSMImport(imp Import) {
	if !isHSMImportPath(imp.Path) || imp.Alias == "" {
		return
	}
	if lowerer.hsmAliases == nil {
		lowerer.hsmAliases = map[string]bool{}
	}
	lowerer.hsmAliases[imp.Alias] = true
}

func (lowerer *zigLowerer) addUnknownHSMPartialDiagnostic(context string, call *sitter.Node, name string) {
	if !lowerer.isHSMRuntimeCall(call) {
		return
	}
	if name == "" {
		name = zigCallName(call, lowerer.input.Data)
	}
	lowerer.diagnostics = append(lowerer.diagnostics, Diagnostic{
		Message: fmt.Sprintf("unknown or unsupported hsm.%s partial in %s context", name, context),
		Range:   lowerer.sourceRange(call),
	})
}

func (lowerer *zigLowerer) isHSMRuntimeCall(call *sitter.Node) bool {
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
	return qualifier == "hsm" || lowerer.hsmAliases[qualifier]
}

func (lowerer *zigLowerer) globalBlock(node *sitter.Node) CodeBlock {
	return CodeBlock{ID: lowerer.nextID("global"), Language: LanguageZig, Code: node.Utf8Text(lowerer.input.Data), Range: lowerer.sourceRange(node)}
}

func (lowerer *zigLowerer) eventBySymbol(symbol string) *EventDecl {
	for index := range lowerer.program.Events {
		if lowerer.program.Events[index].Symbol == symbol {
			return &lowerer.program.Events[index]
		}
	}
	return nil
}

func (lowerer *zigLowerer) sourceRange(node *sitter.Node) SourceRange {
	start := node.StartPosition()
	end := node.EndPosition()
	return SourceRange{File: lowerer.input.Path, StartByte: node.StartByte(), EndByte: node.EndByte(), StartLine: uint(start.Row) + 1, EndLine: uint(end.Row) + 1}
}

func (lowerer *zigLowerer) nextID(prefix string) string {
	if lowerer.counters == nil {
		lowerer.counters = map[string]int{}
	}
	lowerer.counters[prefix]++
	return fmt.Sprintf("%s_%d", prefix, lowerer.counters[prefix])
}

func zigAsCall(node *sitter.Node) *sitter.Node {
	if node != nil && node.Kind() == "call_expression" {
		return node
	}
	return nil
}

func zigCallArgs(call *sitter.Node) []*sitter.Node {
	if call == nil || call.Kind() != "call_expression" {
		return nil
	}
	function := call.ChildByFieldName("function")
	var result []*sitter.Node
	for i := uint(0); i < call.NamedChildCount(); i++ {
		child := call.NamedChild(i)
		if function != nil && child.StartByte() == function.StartByte() && child.EndByte() == function.EndByte() {
			continue
		}
		result = append(result, child)
	}
	return result
}

func zigPartialCalls(nodes ...*sitter.Node) []*sitter.Node {
	var result []*sitter.Node
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if node.Kind() == "call_expression" {
			result = append(result, node)
			continue
		}
		if node.Kind() == "anonymous_struct_initializer" || node.Kind() == "initializer_list" {
			result = append(result, zigPartialCalls(zigInitializerElements(node)...)...)
		}
	}
	return result
}

func zigInitializerElements(node *sitter.Node) []*sitter.Node {
	if node == nil {
		return nil
	}
	if node.Kind() == "anonymous_struct_initializer" && node.NamedChildCount() > 0 {
		node = node.NamedChild(0)
	}
	if node.Kind() != "initializer_list" {
		return []*sitter.Node{node}
	}
	var result []*sitter.Node
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child.Kind() == "field_initializer" {
			continue
		}
		result = append(result, child)
	}
	return result
}

func zigCallName(call *sitter.Node, source []byte) string {
	if call == nil {
		return ""
	}
	function := call.ChildByFieldName("function")
	if function == nil && call.NamedChildCount() > 0 {
		function = call.NamedChild(0)
	}
	if function == nil {
		return ""
	}
	if function.Kind() == "field_expression" {
		if member := function.ChildByFieldName("member"); member != nil {
			return member.Utf8Text(source)
		}
	}
	text := function.Utf8Text(source)
	if index := strings.LastIndex(text, "."); index >= 0 {
		return text[index+1:]
	}
	return text
}

func zigCanonicalCallName(call *sitter.Node, source []byte) string {
	switch strings.ToLower(zigCallName(call, source)) {
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
	case "history":
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
	case "deferevents":
		return "Defer"
	case "attribute":
		return "Attribute"
	case "operation":
		return "Operation"
	default:
		return zigCallName(call, source)
	}
}

func zigStateKind(name string) StateKind {
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

func zigFirstStringOrRaw(call *sitter.Node, source []byte) string {
	value, _ := zigFirstArgValue(call, source)
	return value
}

func zigFirstArgValue(call *sitter.Node, source []byte) (string, bool) {
	args := zigCallArgs(call)
	if len(args) == 0 {
		return "", false
	}
	return zigNodeArgValue(args[0], source)
}

func zigNodeArgValue(node *sitter.Node, source []byte) (string, bool) {
	if value, ok := zigStringLiteral(node, source); ok {
		return value, true
	}
	return strings.TrimSpace(node.Utf8Text(source)), false
}

func zigStringLiteral(node *sitter.Node, source []byte) (string, bool) {
	if node == nil || node.Kind() != "string" {
		return "", false
	}
	text := strings.TrimSpace(node.Utf8Text(source))
	if value, err := strconv.Unquote(text); err == nil {
		return value, true
	}
	if node.NamedChildCount() > 0 && node.NamedChild(0).Kind() == "string_content" {
		return node.NamedChild(0).Utf8Text(source), true
	}
	return "", false
}

func zigDeclarationName(node *sitter.Node) *sitter.Node {
	if node == nil || node.Kind() != "variable_declaration" {
		return nil
	}
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child.Kind() == "identifier" {
			return child
		}
	}
	return nil
}

func zigDeclarationValue(node *sitter.Node) *sitter.Node {
	if node == nil || node.Kind() != "variable_declaration" {
		return nil
	}
	name := zigDeclarationName(node)
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if name != nil && child.StartByte() == name.StartByte() && child.EndByte() == name.EndByte() {
			continue
		}
		switch child.Kind() {
		case "builtin_function", "call_expression", "string", "anonymous_struct_initializer", "struct_initializer", "identifier", "integer", "float", "boolean":
			return child
		}
	}
	return nil
}

func zigNodeContainsCall(node *sitter.Node, source []byte, name string) bool {
	var found bool
	walk(node, func(child *sitter.Node) {
		if found || child.Kind() != "call_expression" {
			return
		}
		found = strings.EqualFold(zigCallName(child, source), name)
	})
	return found
}
