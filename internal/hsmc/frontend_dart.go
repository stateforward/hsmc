package hsmc

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type DartFrontend struct{}

func NewDartFrontend() DartFrontend {
	return DartFrontend{}
}

func (DartFrontend) Language() Language { return LanguageDart }

func (frontend DartFrontend) Parse(ctx context.Context, input SourceInput) (*Program, error) {
	tree, err := parseTree(ctx, input, parserGrammarDart)
	if err != nil {
		return nil, err
	}
	defer tree.Close()
	lowerer := dartLowerer{
		input: SourceInput{Path: input.Path, Data: input.Data},
		program: &Program{
			SourceLanguage: LanguageDart,
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

type dartLowerer struct {
	input          SourceInput
	program        *Program
	counters       map[string]int
	functions      map[string]dartFunctionDecl
	usedFunctions  map[string]bool
	hsmAliases     map[string]bool
	hsmUnqualified bool
	diagnostics    Diagnostics
}

type dartCall struct {
	name      string
	qualifier string
	selector  *sitter.Node
	start     uint
	end       uint
}

func (lowerer *dartLowerer) collectTopLevel(root *sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		child := root.NamedChild(i)
		switch child.Kind() {
		case "import_or_export":
			if imp, ok := lowerer.parseImport(child); ok {
				if isHSMImportPath(imp.Path) {
					lowerer.recordHSMImport(imp)
					continue
				}
				lowerer.program.Imports = append(lowerer.program.Imports, imp)
			}
		case "static_final_declaration_list", "initialized_variable_definition":
			if lowerer.collectVariableDeclaration(child) {
				continue
			}
		case "function_signature":
			end := child.EndByte()
			var body *sitter.Node
			if i+1 < root.NamedChildCount() && root.NamedChild(i+1).Kind() == "function_body" {
				body = root.NamedChild(i + 1)
				end = body.EndByte()
				i++
			}
			block := lowerer.globalRange(child.StartByte(), end, child)
			lowerer.registerFunctionSignature(child, body, block.Range)
			lowerer.program.Globals = append(lowerer.program.Globals, block)
		case "class_definition", "mixin_declaration", "enum_declaration", "extension_declaration", "extension_type_declaration":
			lowerer.program.Globals = append(lowerer.program.Globals, lowerer.globalBlock(child))
		}
	}
}

func (lowerer *dartLowerer) collectVariableDeclaration(node *sitter.Node) bool {
	if !lowerer.containsDefine(node) {
		if event, ok := lowerer.parseEventDecl(node); ok {
			lowerer.program.Events = append(lowerer.program.Events, event)
			return true
		}
		declarations := directDartDeclarations(node)
		if lowerer.hasGeneratedFunctionExpressionDecl(declarations) {
			prefix := strings.TrimSpace(string(lowerer.input.Data[node.StartByte():declarations[0].StartByte()]))
			if prefix == "" && declarations[0].Kind() == "static_final_declaration" {
				prefix = "final"
			}
			for _, declaration := range declarations {
				if event, ok := lowerer.parseEventDeclaration(declaration); ok {
					lowerer.program.Events = append(lowerer.program.Events, event)
					continue
				}
				block := lowerer.globalDeclarationBlock(prefix, declaration)
				lowerer.registerFunctionExpressionDecl(declaration, block.Range)
				lowerer.program.Globals = append(lowerer.program.Globals, block)
			}
			return true
		}
		block := lowerer.globalBlock(node)
		for _, declaration := range directDartDeclarations(node) {
			lowerer.registerFunctionExpressionDecl(declaration, block.Range)
		}
		lowerer.program.Globals = append(lowerer.program.Globals, block)
		return true
	}

	declarations := directDartDeclarations(node)
	if len(declarations) == 0 {
		return true
	}
	prefix := strings.TrimSpace(string(lowerer.input.Data[node.StartByte():declarations[0].StartByte()]))
	if prefix == "" && declarations[0].Kind() == "static_final_declaration" {
		prefix = "final"
	}
	for _, declaration := range declarations {
		if lowerer.containsDefine(declaration) {
			continue
		}
		if event, ok := lowerer.parseEventDeclaration(declaration); ok {
			lowerer.program.Events = append(lowerer.program.Events, event)
			continue
		}
		block := lowerer.globalDeclarationBlock(prefix, declaration)
		lowerer.registerFunctionExpressionDecl(declaration, block.Range)
		lowerer.program.Globals = append(lowerer.program.Globals, block)
	}
	return true
}

func (lowerer *dartLowerer) collectModels(root *sitter.Node) {
	walk(root, func(node *sitter.Node) {
		if node.Kind() != "static_final_declaration" && node.Kind() != "initialized_identifier" {
			return
		}
		call, ok := lowerer.callFromDeclaration(node)
		if !ok || !dartNameIs(call.name, "define") {
			return
		}
		if model, ok := lowerer.parseModel(call); ok {
			lowerer.program.Models = append(lowerer.program.Models, model)
		}
	})
}

func (lowerer *dartLowerer) parseModel(call dartCall) (Model, bool) {
	args := lowerer.callArgs(call)
	if len(args) == 0 {
		return Model{}, false
	}
	name, ok := dartStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Model{}, false
	}
	model := Model{ID: lowerer.nextID("model"), Name: name, Range: lowerer.sourceRangeBytes(call.start, call.end)}
	for _, partial := range lowerer.partialCalls(args[1:]...) {
		lowerer.applyModelPartial(&model, partial)
	}
	return model, true
}

func (lowerer *dartLowerer) applyModelPartial(model *Model, partial dartCall) {
	name := dartCanonicalName(partial.name)
	switch name {
	case "Initial":
		initial := lowerer.parseInitial(model.ID, partial)
		model.Initializer = &initial
	case "State", "Final", "Choice", "ShallowHistory", "DeepHistory":
		if state, ok := lowerer.parseState(model.ID, partial); ok {
			model.States = append(model.States, state)
		}
	case "Transition":
		model.Transitions = append(model.Transitions, lowerer.parseTransition(model.ID, partial))
	case "Attribute":
		if attribute, ok := lowerer.parseAttribute(partial); ok {
			model.Attributes = append(model.Attributes, attribute)
		}
	case "Operation":
		if operation, ok := lowerer.parseOperation(model.ID, partial); ok {
			model.Operations = append(model.Operations, operation)
		}
	default:
		lowerer.addUnknownHSMPartialDiagnostic("model", partial, name)
	}
}

func (lowerer *dartLowerer) parseAttribute(call dartCall) (Attribute, bool) {
	args := lowerer.callArgs(call)
	if len(args) == 0 {
		return Attribute{}, false
	}
	name, ok := dartStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Attribute{}, false
	}
	attribute := Attribute{ID: lowerer.nextID("attribute"), Name: name, Language: LanguageDart, Range: lowerer.sourceRangeBytes(call.start, call.end)}
	if len(args) == 2 {
		field, value := classifyAttributeSecondArg(LanguageDart, args[1].Utf8Text(lowerer.input.Data))
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

func (lowerer *dartLowerer) parseOperation(ownerID string, call dartCall) (Operation, bool) {
	args := lowerer.callArgs(call)
	if len(args) < 2 {
		return Operation{}, false
	}
	name, ok := dartStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Operation{}, false
	}
	operation := Operation{ID: lowerer.nextID("operation_decl"), Name: name, Range: lowerer.sourceRangeBytes(call.start, call.end)}
	behavior := lowerer.parseBehavior(ownerID, BehaviorOperation, args[1])
	lowerer.program.Behaviors = append(lowerer.program.Behaviors, behavior)
	operation.Behavior = &BehaviorRef{ID: behavior.ID, Kind: BehaviorOperation}
	return operation, true
}

func (lowerer *dartLowerer) parseState(ownerID string, call dartCall) (State, bool) {
	args := lowerer.callArgs(call)
	if len(args) == 0 {
		return State{}, false
	}
	name, ok := dartStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return State{}, false
	}
	state := State{ID: lowerer.nextID("state"), Name: name, Kind: stateKind(dartCanonicalName(call.name)), Range: lowerer.sourceRangeBytes(call.start, call.end)}
	for _, partial := range lowerer.partialCalls(args[1:]...) {
		name := dartCanonicalName(partial.name)
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
			state.Behaviors = append(state.Behaviors, lowerer.parseBehaviors(state.ID, behaviorKind(dartCanonicalName(partial.name)), partial)...)
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

func (lowerer *dartLowerer) applyHistoryDefaultPartial(state *State, partial dartCall, name string) bool {
	if state.Kind != StateKindShallowHistory && state.Kind != StateKindDeepHistory {
		return false
	}
	if len(state.Transitions) == 0 {
		state.Transitions = append(state.Transitions, Transition{OwnerID: state.ID, ID: lowerer.nextID("transition"), Range: lowerer.sourceRangeBytes(partial.start, partial.end)})
	}
	transition := &state.Transitions[0]
	switch name {
	case "Target":
		transition.Target = lowerer.firstStringArg(partial)
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

func (lowerer *dartLowerer) parseInitial(ownerID string, call dartCall) Initial {
	initial := Initial{OwnerID: ownerID, ID: lowerer.nextID("initial"), Range: lowerer.sourceRangeBytes(call.start, call.end)}
	for _, partial := range lowerer.partialCalls(lowerer.callArgs(call)...) {
		name := dartCanonicalName(partial.name)
		switch name {
		case "Target":
			initial.Target = lowerer.firstStringArg(partial)
		case "Effect":
			initial.Effects = append(initial.Effects, lowerer.parseBehaviors(ownerID, BehaviorEffect, partial)...)
		default:
			lowerer.addUnknownHSMPartialDiagnostic("initial", partial, name)
		}
	}
	return initial
}

func (lowerer *dartLowerer) parseTransition(ownerID string, call dartCall) Transition {
	transition := Transition{OwnerID: ownerID, ID: lowerer.nextID("transition"), Range: lowerer.sourceRangeBytes(call.start, call.end)}
	for _, partial := range lowerer.partialCalls(lowerer.callArgs(call)...) {
		name := dartCanonicalName(partial.name)
		switch name {
		case "Source":
			transition.Source = lowerer.firstStringArg(partial)
		case "Target":
			transition.Target = lowerer.firstStringArg(partial)
		case "On":
			transition.Trigger = lowerer.parseOnTrigger(partial)
		case "OnSet":
			transition.Trigger = &Trigger{Kind: TriggerOnSet, Value: lowerer.firstStringArg(partial), IsString: true, Language: LanguageDart}
		case "OnCall":
			transition.Trigger = &Trigger{Kind: TriggerOnCall, Value: lowerer.firstStringArg(partial), IsString: true, Language: LanguageDart}
		case "After":
			value, isString := lowerer.firstArgValue(partial)
			transition.Trigger = lowerer.parseExpressionTrigger(ownerID, TriggerAfter, value, isString, partial)
		case "Every":
			value, isString := lowerer.firstArgValue(partial)
			transition.Trigger = lowerer.parseExpressionTrigger(ownerID, TriggerEvery, value, isString, partial)
		case "At":
			value, isString := lowerer.firstArgValue(partial)
			transition.Trigger = lowerer.parseExpressionTrigger(ownerID, TriggerAt, value, isString, partial)
		case "When":
			value, isString := lowerer.firstArgValue(partial)
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

func (lowerer *dartLowerer) parseOnTrigger(call dartCall) *Trigger {
	trigger := &Trigger{Kind: TriggerOn, Language: LanguageDart}
	for _, arg := range lowerer.callArgs(call) {
		value, isString := lowerer.nodeArgValue(arg)
		eventSymbol := ""
		if !isString && lowerer.eventBySymbol(value) != nil {
			eventSymbol = value
		} else if !isString && strings.HasSuffix(value, ".name") && lowerer.eventBySymbol(strings.TrimSuffix(value, ".name")) != nil {
			eventSymbol = strings.TrimSuffix(value, ".name")
		}
		if trigger.Value == "" {
			trigger.Value = value
			trigger.IsString = isString
		}
		trigger.Values = append(trigger.Values, TriggerValue{Value: value, IsString: isString, EventSymbol: eventSymbol, Language: LanguageDart})
	}
	if len(trigger.Values) <= 1 && (len(trigger.Values) == 0 || trigger.Values[0].EventSymbol == "") {
		trigger.Values = nil
	}
	return trigger
}

func (lowerer *dartLowerer) parseExpressionTrigger(ownerID string, kind TriggerKind, value string, isString bool, call dartCall) *Trigger {
	trigger := &Trigger{Kind: kind, Value: value, IsString: isString, Language: LanguageDart}
	args := lowerer.callArgs(call)
	if isString || len(args) == 0 {
		return trigger
	}
	behavior := lowerer.parseBehavior(ownerID, BehaviorTrigger, args[0])
	behavior.TriggerKind = kind
	lowerer.program.Behaviors = append(lowerer.program.Behaviors, behavior)
	trigger.Expr = &BehaviorRef{ID: behavior.ID, Kind: BehaviorTrigger}
	return trigger
}

func (lowerer *dartLowerer) parseBehaviors(ownerID string, kind BehaviorKind, call dartCall) []BehaviorRef {
	var refs []BehaviorRef
	for _, arg := range lowerer.callArgs(call) {
		if operation, ok := dartStringLiteral(arg, lowerer.input.Data); ok {
			refs = append(refs, BehaviorRef{Kind: kind, Operation: operation})
			continue
		}
		behavior := lowerer.parseBehavior(ownerID, kind, arg)
		lowerer.program.Behaviors = append(lowerer.program.Behaviors, behavior)
		refs = append(refs, BehaviorRef{ID: behavior.ID, Kind: kind})
	}
	return refs
}

func (lowerer *dartLowerer) parseBehavior(ownerID string, kind BehaviorKind, node *sitter.Node) Behavior {
	inline := node.Kind() == "function_expression"
	if resolved := lowerer.referencedBehaviorFunction(node, kind); resolved != nil {
		return lowerer.parseRegisteredBehavior(ownerID, kind, *resolved)
	}
	code := strings.TrimSpace(node.Utf8Text(lowerer.input.Data))
	behavior := Behavior{
		ID:             lowerer.nextID(string(kind)),
		OwnerID:        ownerID,
		Kind:           kind,
		Inline:         inline,
		SourceLanguage: LanguageDart,
		Code:           code,
		Range:          lowerer.sourceRange(node),
	}
	if node.Kind() == "function_expression" {
		body := firstNamedChildOfKind(node, "function_expression_body")
		if body != nil {
			behavior.Body = lowerer.behaviorBody(kind, body)
			behavior.Signature = strings.TrimSpace(string(lowerer.input.Data[node.StartByte():body.StartByte()]))
		}
	}
	return behavior
}

type dartFunctionDecl struct {
	name        string
	node        *sitter.Node
	body        *sitter.Node
	globalRange SourceRange
}

func (lowerer *dartLowerer) parseRegisteredBehavior(ownerID string, kind BehaviorKind, decl dartFunctionDecl) Behavior {
	node := decl.node
	code := strings.TrimSpace(node.Utf8Text(lowerer.input.Data))
	if decl.body != nil && decl.body.EndByte() > node.EndByte() {
		code = strings.TrimSpace(string(lowerer.input.Data[node.StartByte():decl.body.EndByte()]))
	}
	behavior := Behavior{
		ID:             lowerer.nextID(string(kind)),
		OwnerID:        ownerID,
		Kind:           kind,
		SourceLanguage: LanguageDart,
		Code:           code,
		Range:          lowerer.sourceRange(node),
	}
	switch node.Kind() {
	case "function_signature":
		if decl.body != nil {
			behavior.Body = lowerer.behaviorBody(kind, decl.body)
			behavior.Signature = strings.TrimSpace(string(lowerer.input.Data[node.StartByte():decl.body.StartByte()]))
		}
	case "function_expression":
		if body := firstNamedChildOfKind(node, "function_expression_body"); body != nil {
			behavior.Body = lowerer.behaviorBody(kind, body)
			behavior.Signature = strings.TrimSpace(string(lowerer.input.Data[node.StartByte():body.StartByte()]))
		}
	}
	return behavior
}

func (lowerer *dartLowerer) registerFunctionSignature(signature *sitter.Node, body *sitter.Node, globalRange SourceRange) {
	if signature == nil {
		return
	}
	nameNode := lowerer.dartFunctionSignatureName(signature)
	if nameNode == nil {
		return
	}
	name := nameNode.Utf8Text(lowerer.input.Data)
	lowerer.registerFunctionDecl(dartFunctionDecl{name: name, node: signature, body: body, globalRange: globalRange})
}

func (lowerer *dartLowerer) registerFunctionExpressionDecl(declaration *sitter.Node, globalRange SourceRange) {
	nameNode := lowerer.dartDeclarationName(declaration)
	if nameNode == nil {
		return
	}
	name := nameNode.Utf8Text(lowerer.input.Data)
	function := firstNamedChildOfKind(declaration, "function_expression")
	if function == nil {
		return
	}
	lowerer.registerFunctionDecl(dartFunctionDecl{name: name, node: function, globalRange: globalRange})
}

func (lowerer *dartLowerer) hasGeneratedFunctionExpressionDecl(declarations []*sitter.Node) bool {
	for _, declaration := range declarations {
		nameNode := lowerer.dartDeclarationName(declaration)
		if nameNode == nil {
			continue
		}
		name := nameNode.Utf8Text(lowerer.input.Data)
		if isGeneratedBehaviorFunctionNameForAnyKind(name) && firstNamedChildOfKind(declaration, "function_expression") != nil {
			return true
		}
	}
	return false
}

func (lowerer *dartLowerer) registerFunctionDecl(decl dartFunctionDecl) {
	if lowerer.functions == nil {
		lowerer.functions = map[string]dartFunctionDecl{}
	}
	lowerer.functions[decl.name] = decl
}

func (lowerer *dartLowerer) referencedBehaviorFunction(node *sitter.Node, kind BehaviorKind) *dartFunctionDecl {
	if node == nil || node.Kind() != "identifier" {
		return nil
	}
	name := node.Utf8Text(lowerer.input.Data)
	if isGeneratedBehaviorFunctionNameForAnyKind(name) && !isGeneratedBehaviorFunctionName(name, kind) {
		return nil
	}
	decl, ok := lowerer.functions[name]
	if !ok {
		return nil
	}
	if lowerer.usedFunctions == nil {
		lowerer.usedFunctions = map[string]bool{}
	}
	lowerer.usedFunctions[name] = true
	return &decl
}

func (lowerer *dartLowerer) filterReferencedFunctionGlobals() {
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

func (lowerer *dartLowerer) dartFunctionSignatureName(signature *sitter.Node) *sitter.Node {
	for i := uint(0); i < signature.NamedChildCount(); i++ {
		child := signature.NamedChild(i)
		if child.Kind() == "identifier" {
			return child
		}
	}
	return nil
}

func (lowerer *dartLowerer) dartDeclarationName(declaration *sitter.Node) *sitter.Node {
	for i := uint(0); i < declaration.NamedChildCount(); i++ {
		child := declaration.NamedChild(i)
		if child.Kind() == "identifier" {
			return child
		}
	}
	return nil
}

func isGeneratedBehaviorFunctionNameForAnyKind(name string) bool {
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

func (lowerer *dartLowerer) behaviorBody(kind BehaviorKind, body *sitter.Node) string {
	if block := firstNamedChildOfKind(body, "block"); block != nil {
		return trimBlock(block.Utf8Text(lowerer.input.Data))
	}
	expression := strings.TrimSpace(body.Utf8Text(lowerer.input.Data))
	expression = strings.TrimSpace(strings.TrimPrefix(expression, "=>"))
	if kind == BehaviorGuard || kind == BehaviorTrigger {
		return "return " + strings.TrimSuffix(expression, ";") + ";"
	}
	return strings.TrimSuffix(expression, ";") + ";"
}

func (lowerer *dartLowerer) parseDefers(call dartCall) []string {
	var defers []string
	for _, arg := range lowerer.callArgs(call) {
		if value, ok := dartStringLiteral(arg, lowerer.input.Data); ok {
			defers = append(defers, value)
			continue
		}
		value := strings.TrimSpace(arg.Utf8Text(lowerer.input.Data))
		if event := lowerer.eventBySymbol(value); event != nil {
			defers = append(defers, event.Name)
			continue
		}
		if strings.HasSuffix(value, ".name") {
			if event := lowerer.eventBySymbol(strings.TrimSuffix(value, ".name")); event != nil {
				defers = append(defers, event.Name)
				continue
			}
		}
		defers = append(defers, value)
	}
	return defers
}

func (lowerer *dartLowerer) parseEventDecl(node *sitter.Node) (EventDecl, bool) {
	var event EventDecl
	walk(node, func(child *sitter.Node) {
		if event.Symbol != "" || child.Kind() != "static_final_declaration" {
			return
		}
		if parsed, ok := lowerer.parseEventDeclaration(child); ok {
			event = parsed
		}
	})
	return event, event.Symbol != ""
}

func (lowerer *dartLowerer) parseEventDeclaration(node *sitter.Node) (EventDecl, bool) {
	if node.Kind() != "static_final_declaration" || node.NamedChildCount() < 3 {
		return EventDecl{}, false
	}
	symbol := node.NamedChild(0)
	call, ok := lowerer.callFromDeclaration(node)
	if !ok || !dartNameIs(call.name, "Event") {
		return EventDecl{}, false
	}
	name := lowerer.eventNameArg(call)
	if name == "" {
		return EventDecl{}, false
	}
	return EventDecl{
		ID:       lowerer.nextID("event"),
		Symbol:   strings.TrimSpace(symbol.Utf8Text(lowerer.input.Data)),
		Name:     name,
		Language: LanguageDart,
		Range:    lowerer.sourceRange(node),
	}, true
}

func (lowerer *dartLowerer) eventNameArg(call dartCall) string {
	for _, arg := range lowerer.callArgs(call) {
		if value, ok := dartStringLiteral(arg, lowerer.input.Data); ok {
			return value
		}
		if arg.NamedChildCount() == 2 && arg.NamedChild(0).Kind() == "label" {
			label := strings.TrimSuffix(strings.TrimSpace(arg.NamedChild(0).Utf8Text(lowerer.input.Data)), ":")
			if strings.EqualFold(label, "name") {
				if value, ok := dartStringLiteral(arg.NamedChild(1), lowerer.input.Data); ok {
					return value
				}
			}
		}
	}
	return ""
}

func (lowerer *dartLowerer) parseImport(node *sitter.Node) (Import, bool) {
	if !strings.HasPrefix(strings.TrimSpace(node.Utf8Text(lowerer.input.Data)), "import ") {
		return Import{}, false
	}
	var imp Import
	var pathEnd uint
	walk(node, func(child *sitter.Node) {
		if imp.Path == "" && child.Kind() == "string_literal" {
			if value, ok := dartStringLiteral(child, lowerer.input.Data); ok {
				imp.Path = value
				imp.Language = LanguageDart
				imp.Range = lowerer.sourceRange(node)
				pathEnd = child.EndByte()
			}
			return
		}
	})
	if imp.Path != "" && pathEnd <= node.EndByte() {
		suffix := string(lowerer.input.Data[pathEnd:node.EndByte()])
		imp.Alias, imp.Deferred, imp.Specifiers, imp.Hidden, imp.LocalNames = parseDartImportCombinators(suffix)
	}
	return imp, imp.Path != ""
}

func (lowerer *dartLowerer) recordHSMImport(imp Import) {
	if imp.Alias == "" {
		lowerer.hsmUnqualified = true
		return
	}
	if lowerer.hsmAliases == nil {
		lowerer.hsmAliases = map[string]bool{}
	}
	lowerer.hsmAliases[imp.Alias] = true
}

func (lowerer *dartLowerer) addUnknownHSMPartialDiagnostic(context string, call dartCall, name string) {
	if !lowerer.isHSMRuntimeCall(call) {
		return
	}
	lowerer.diagnostics = append(lowerer.diagnostics, Diagnostic{
		Message: fmt.Sprintf("unknown or unsupported hsm.%s partial in %s context", name, context),
		Range:   lowerer.sourceRangeBytes(call.start, call.end),
	})
}

func (lowerer *dartLowerer) isHSMRuntimeCall(call dartCall) bool {
	qualifier := strings.TrimSpace(call.qualifier)
	if qualifier != "" {
		return lowerer.hsmAliases[qualifier]
	}
	return lowerer.hsmUnqualified
}

func parseDartImportCombinators(suffix string) (string, bool, []ImportSpecifier, []string, []string) {
	words := dartImportWords(suffix)
	var alias string
	var deferred bool
	var specifiers []ImportSpecifier
	var hidden []string
	var localNames []string
	for index := 0; index < len(words); index++ {
		switch words[index] {
		case "deferred":
			deferred = true
		case "as":
			if index+1 < len(words) {
				alias = words[index+1]
				localNames = append(localNames, alias)
				index++
			}
		case "show":
			for index+1 < len(words) {
				next := words[index+1]
				if next == "as" || next == "show" || next == "hide" {
					break
				}
				specifiers = append(specifiers, ImportSpecifier{Name: next})
				localNames = append(localNames, next)
				index++
			}
		case "hide":
			for index+1 < len(words) {
				next := words[index+1]
				if next == "as" || next == "show" || next == "hide" {
					break
				}
				hidden = append(hidden, next)
				index++
			}
		}
	}
	if alias != "" {
		localNames = []string{alias}
	}
	return alias, deferred, specifiers, hidden, localNames
}

func dartImportWords(suffix string) []string {
	return strings.FieldsFunc(suffix, func(r rune) bool {
		return r == ',' || r == ';' || r == '(' || r == ')' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
}

func (lowerer *dartLowerer) callFromDeclaration(node *sitter.Node) (dartCall, bool) {
	for i := uint(0); i+2 < node.NamedChildCount(); i++ {
		base := node.NamedChild(i)
		member := node.NamedChild(i + 1)
		args := node.NamedChild(i + 2)
		if base.Kind() == "identifier" && member.Kind() == "selector" && args.Kind() == "selector" {
			name := dartSelectorMemberName(member.Utf8Text(lowerer.input.Data))
			if name != "" {
				return dartCall{name: name, qualifier: base.Utf8Text(lowerer.input.Data), selector: args, start: base.StartByte(), end: args.EndByte()}, true
			}
		}
	}
	for i := uint(0); i+1 < node.NamedChildCount(); i++ {
		name := node.NamedChild(i)
		selector := node.NamedChild(i + 1)
		if name.Kind() == "identifier" && selector.Kind() == "selector" {
			return dartCall{name: lowerer.dartSelectedCallName(name, selector), selector: selector, start: name.StartByte(), end: selector.EndByte()}, true
		}
	}
	var call dartCall
	walk(node, func(child *sitter.Node) {
		if call.name != "" || child.Kind() != "selector" {
			return
		}
		name := dartSelectorCallName(child.Utf8Text(lowerer.input.Data))
		if name == "" {
			return
		}
		call = dartCall{name: name, selector: child, start: child.StartByte(), end: child.EndByte()}
	})
	if call.name != "" {
		return call, true
	}
	return dartCall{}, false
}

func (lowerer *dartLowerer) dartSelectedCallName(base *sitter.Node, selector *sitter.Node) string {
	name := base.Utf8Text(lowerer.input.Data)
	var argsStart uint
	walk(selector, func(node *sitter.Node) {
		if argsStart == 0 && node.Kind() == "arguments" {
			argsStart = node.StartByte()
		}
	})
	if argsStart == 0 {
		return name
	}
	walk(selector, func(node *sitter.Node) {
		if node.Kind() == "identifier" && node.EndByte() <= argsStart {
			name = node.Utf8Text(lowerer.input.Data)
		}
	})
	return name
}

func dartSelectorCallName(selectorText string) string {
	beforeArgs, _, ok := strings.Cut(selectorText, "(")
	if !ok {
		return ""
	}
	beforeArgs = strings.TrimSpace(beforeArgs)
	if beforeArgs == "" {
		return ""
	}
	if index := strings.LastIndex(beforeArgs, "."); index >= 0 {
		beforeArgs = beforeArgs[index+1:]
	}
	return strings.TrimSpace(beforeArgs)
}

func dartSelectorMemberName(selectorText string) string {
	text := strings.TrimSpace(selectorText)
	if strings.Contains(text, "(") {
		return ""
	}
	if index := strings.LastIndex(text, "."); index >= 0 {
		text = text[index+1:]
	}
	return strings.TrimSpace(text)
}

func (lowerer *dartLowerer) callArgs(call dartCall) []*sitter.Node {
	var argsNode *sitter.Node
	walk(call.selector, func(node *sitter.Node) {
		if argsNode == nil && node.Kind() == "arguments" {
			argsNode = node
		}
	})
	if argsNode == nil {
		return nil
	}
	var args []*sitter.Node
	for i := uint(0); i < argsNode.NamedChildCount(); i++ {
		args = append(args, dartUnwrapArg(argsNode.NamedChild(i)))
	}
	return args
}

func (lowerer *dartLowerer) partialCalls(nodes ...*sitter.Node) []dartCall {
	var calls []dartCall
	for _, node := range nodes {
		if node.Kind() == "list_literal" {
			for i := uint(0); i+1 < node.NamedChildCount(); i++ {
				if i+2 < node.NamedChildCount() {
					base := node.NamedChild(i)
					member := node.NamedChild(i + 1)
					args := node.NamedChild(i + 2)
					if base.Kind() == "identifier" && member.Kind() == "selector" && args.Kind() == "selector" {
						name := dartSelectorMemberName(member.Utf8Text(lowerer.input.Data))
						if name != "" {
							calls = append(calls, dartCall{name: name, qualifier: base.Utf8Text(lowerer.input.Data), selector: args, start: base.StartByte(), end: args.EndByte()})
							i += 2
							continue
						}
					}
				}
				name := node.NamedChild(i)
				selector := node.NamedChild(i + 1)
				if name.Kind() == "identifier" && selector.Kind() == "selector" {
					calls = append(calls, dartCall{name: lowerer.dartSelectedCallName(name, selector), selector: selector, start: name.StartByte(), end: selector.EndByte()})
					i++
				}
			}
			continue
		}
		if node.Kind() == "argument" {
			node = dartUnwrapArg(node)
		}
		if node.NamedChildCount() == 3 && node.NamedChild(0).Kind() == "identifier" && node.NamedChild(1).Kind() == "selector" && node.NamedChild(2).Kind() == "selector" {
			name := dartSelectorMemberName(node.NamedChild(1).Utf8Text(lowerer.input.Data))
			if name != "" {
				calls = append(calls, dartCall{name: name, qualifier: node.NamedChild(0).Utf8Text(lowerer.input.Data), selector: node.NamedChild(2), start: node.NamedChild(0).StartByte(), end: node.NamedChild(2).EndByte()})
			}
			continue
		}
		if node.NamedChildCount() == 2 && node.NamedChild(0).Kind() == "identifier" && node.NamedChild(1).Kind() == "selector" {
			calls = append(calls, dartCall{name: lowerer.dartSelectedCallName(node.NamedChild(0), node.NamedChild(1)), selector: node.NamedChild(1), start: node.NamedChild(0).StartByte(), end: node.NamedChild(1).EndByte()})
		}
	}
	return calls
}

func (lowerer *dartLowerer) firstStringArg(call dartCall) string {
	args := lowerer.callArgs(call)
	if len(args) == 0 {
		return ""
	}
	if value, ok := dartStringLiteral(args[0], lowerer.input.Data); ok {
		return value
	}
	return strings.TrimSpace(args[0].Utf8Text(lowerer.input.Data))
}

func (lowerer *dartLowerer) firstArgValue(call dartCall) (string, bool) {
	args := lowerer.callArgs(call)
	if len(args) == 0 {
		return "", false
	}
	return lowerer.nodeArgValue(args[0])
}

func (lowerer *dartLowerer) nodeArgValue(node *sitter.Node) (string, bool) {
	if value, ok := dartStringLiteral(node, lowerer.input.Data); ok {
		return value, true
	}
	return strings.TrimSpace(node.Utf8Text(lowerer.input.Data)), false
}

func (lowerer *dartLowerer) eventBySymbol(symbol string) *EventDecl {
	for index := range lowerer.program.Events {
		if lowerer.program.Events[index].Symbol == symbol {
			return &lowerer.program.Events[index]
		}
	}
	return nil
}

func (lowerer *dartLowerer) containsDefine(node *sitter.Node) bool {
	found := false
	walk(node, func(child *sitter.Node) {
		if child.Kind() == "static_final_declaration" {
			call, ok := lowerer.callFromDeclaration(child)
			if ok && dartNameIs(call.name, "define") {
				found = true
			}
		}
	})
	return found
}

func (lowerer *dartLowerer) globalBlock(node *sitter.Node) CodeBlock {
	return lowerer.globalRange(node.StartByte(), node.EndByte(), node)
}

func (lowerer *dartLowerer) globalRange(startByte uint, endByte uint, node *sitter.Node) CodeBlock {
	return CodeBlock{
		ID:       lowerer.nextID("global"),
		Language: LanguageDart,
		Code:     string(lowerer.input.Data[startByte:endByte]),
		Range:    lowerer.sourceRange(node),
	}
}

func (lowerer *dartLowerer) globalDeclarationBlock(prefix string, declaration *sitter.Node) CodeBlock {
	code := strings.TrimSpace(prefix)
	if code != "" {
		code += " "
	}
	code += strings.TrimSpace(declaration.Utf8Text(lowerer.input.Data))
	if !strings.HasSuffix(code, ";") {
		code += ";"
	}
	return CodeBlock{
		ID:       lowerer.nextID("global"),
		Language: LanguageDart,
		Code:     code,
		Range:    lowerer.sourceRange(declaration),
	}
}

func directDartDeclarations(node *sitter.Node) []*sitter.Node {
	var declarations []*sitter.Node
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child.Kind() == "static_final_declaration" || child.Kind() == "initialized_identifier" {
			declarations = append(declarations, child)
		}
	}
	return declarations
}

func (lowerer *dartLowerer) sourceRange(node *sitter.Node) SourceRange {
	return lowerer.sourceRangeBytes(node.StartByte(), node.EndByte())
}

func (lowerer *dartLowerer) sourceRangeBytes(startByte uint, endByte uint) SourceRange {
	startLine := bytes.Count(lowerer.input.Data[:startByte], []byte("\n")) + 1
	endLine := bytes.Count(lowerer.input.Data[:endByte], []byte("\n")) + 1
	return SourceRange{File: lowerer.input.Path, StartByte: startByte, EndByte: endByte, StartLine: uint(startLine), EndLine: uint(endLine)}
}

func (lowerer *dartLowerer) nextID(prefix string) string {
	if lowerer.counters == nil {
		lowerer.counters = map[string]int{}
	}
	lowerer.counters[prefix]++
	return fmt.Sprintf("%s_%d", prefix, lowerer.counters[prefix])
}

func dartUnwrapArg(node *sitter.Node) *sitter.Node {
	if node.Kind() == "argument" && node.NamedChildCount() == 1 {
		return node.NamedChild(0)
	}
	return node
}

func dartStringLiteral(node *sitter.Node, source []byte) (string, bool) {
	node = dartUnwrapArg(node)
	if node == nil || node.Kind() != "string_literal" {
		return "", false
	}
	text := strings.TrimSpace(node.Utf8Text(source))
	if value, err := strconv.Unquote(text); err == nil {
		return value, true
	}
	if len(text) >= 2 && ((text[0] == '\'' && text[len(text)-1] == '\'') || (text[0] == '"' && text[len(text)-1] == '"')) {
		return text[1 : len(text)-1], true
	}
	return "", false
}

func dartNameIs(value string, want string) bool {
	return strings.EqualFold(value, want)
}

func dartCanonicalName(value string) string {
	switch strings.ToLower(value) {
	case "define":
		return "Define"
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
