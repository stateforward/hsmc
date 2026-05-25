package hsmc

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type JavaFrontend struct{}

func NewJavaFrontend() JavaFrontend {
	return JavaFrontend{}
}

func (JavaFrontend) Language() Language { return LanguageJava }

func (frontend JavaFrontend) Parse(ctx context.Context, input SourceInput) (*Program, error) {
	tree, err := parseTree(ctx, input, parserGrammarJava)
	if err != nil {
		return nil, err
	}
	defer tree.Close()
	lowerer := javaLowerer{
		input: SourceInput{Path: input.Path, Data: input.Data},
		program: &Program{
			SourceLanguage: LanguageJava,
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

type javaLowerer struct {
	input         SourceInput
	program       *Program
	counters      map[string]int
	functions     map[string]javaFunctionDecl
	usedFunctions map[string]bool
	hsmStatic     bool
	diagnostics   Diagnostics
}

func (lowerer *javaLowerer) collectTopLevel(node *sitter.Node) {
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		switch child.Kind() {
		case "import_declaration":
			if imp, ok := lowerer.parseImport(child); ok {
				if isHSMImportPath(imp.Path) {
					lowerer.recordHSMImport(imp)
					continue
				}
				lowerer.program.Imports = append(lowerer.program.Imports, imp)
			}
		case "class_declaration", "interface_declaration", "enum_declaration", "class_body":
			lowerer.collectTopLevel(child)
		case "field_declaration":
			lowerer.collectField(child)
		case "static_initializer":
			lowerer.program.Globals = append(lowerer.program.Globals, lowerer.globalBlock(child))
		case "method_declaration", "constructor_declaration":
			if child.Kind() == "constructor_declaration" && lowerer.isGeneratedEmptyConstructor(child) {
				continue
			}
			block := lowerer.globalBlock(child)
			if child.Kind() == "method_declaration" {
				lowerer.registerFunctionDecl(child, block.Range)
			}
			lowerer.program.Globals = append(lowerer.program.Globals, block)
		}
	}
}

func (lowerer *javaLowerer) collectField(node *sitter.Node) {
	declarators := javaVariableDeclarators(node)
	if !javaNodeContainsCall(node, lowerer.input.Data, "Define") {
		if event, ok := lowerer.parseEventDecl(node); ok {
			lowerer.program.Events = append(lowerer.program.Events, event)
			return
		}
		lowerer.program.Globals = append(lowerer.program.Globals, lowerer.globalBlock(node))
		return
	}
	if len(declarators) == 0 {
		return
	}
	prefix := javaFieldPrefix(node, declarators[0], lowerer.input.Data)
	for _, declarator := range declarators {
		if javaNodeContainsCall(declarator, lowerer.input.Data, "Define") {
			continue
		}
		if event, ok := lowerer.parseEventDeclarator(declarator); ok {
			lowerer.program.Events = append(lowerer.program.Events, event)
			continue
		}
		lowerer.program.Globals = append(lowerer.program.Globals, lowerer.globalDeclaratorBlock(prefix, declarator))
	}
}

func (lowerer *javaLowerer) collectModels(root *sitter.Node) {
	walk(root, func(node *sitter.Node) {
		if node.Kind() != "method_invocation" || javaCallName(node, lowerer.input.Data) != "Define" {
			return
		}
		if model, ok := lowerer.parseModel(node); ok {
			lowerer.program.Models = append(lowerer.program.Models, model)
		}
	})
}

func (lowerer *javaLowerer) parseModel(call *sitter.Node) (Model, bool) {
	args := javaCallArgs(call)
	if len(args) == 0 {
		return Model{}, false
	}
	name, ok := javaStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Model{}, false
	}
	model := Model{ID: lowerer.nextID("model"), Name: name, Range: lowerer.sourceRange(call)}
	for _, arg := range args[1:] {
		if partial := javaAsCall(arg); partial != nil {
			lowerer.applyModelPartial(&model, partial)
		}
	}
	return model, true
}

func (lowerer *javaLowerer) applyModelPartial(model *Model, call *sitter.Node) {
	name := javaCallName(call, lowerer.input.Data)
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

func (lowerer *javaLowerer) parseAttribute(call *sitter.Node) (Attribute, bool) {
	args := javaCallArgs(call)
	if len(args) == 0 {
		return Attribute{}, false
	}
	name, ok := javaStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Attribute{}, false
	}
	attribute := Attribute{ID: lowerer.nextID("attribute"), Name: name, Language: LanguageJava, Range: lowerer.sourceRange(call)}
	if len(args) > 1 {
		field, value := classifyAttributeSecondArg(LanguageJava, args[1].Utf8Text(lowerer.input.Data))
		if field == "type" {
			attribute.Type = value
		} else {
			attribute.HasDefault = true
			attribute.Default = value
		}
	}
	if len(args) > 2 {
		attribute.Type = strings.TrimSpace(args[1].Utf8Text(lowerer.input.Data))
		attribute.HasDefault = true
		attribute.Default = strings.TrimSpace(args[2].Utf8Text(lowerer.input.Data))
	}
	return attribute, true
}

func (lowerer *javaLowerer) parseOperation(ownerID string, call *sitter.Node) (Operation, bool) {
	args := javaCallArgs(call)
	if len(args) < 2 {
		return Operation{}, false
	}
	name, ok := javaStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Operation{}, false
	}
	behavior := lowerer.parseBehavior(ownerID, BehaviorOperation, args[1], "")
	lowerer.program.Behaviors = append(lowerer.program.Behaviors, behavior)
	return Operation{ID: lowerer.nextID("operation_decl"), Name: name, Behavior: &BehaviorRef{ID: behavior.ID, Kind: BehaviorOperation}, Range: lowerer.sourceRange(call)}, true
}

func (lowerer *javaLowerer) parseState(ownerID string, call *sitter.Node) (State, bool) {
	args := javaCallArgs(call)
	if len(args) == 0 {
		return State{}, false
	}
	name, ok := javaStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return State{}, false
	}
	state := State{ID: lowerer.nextID("state"), Name: name, Kind: stateKind(javaCallName(call, lowerer.input.Data)), Range: lowerer.sourceRange(call)}
	for _, arg := range args[1:] {
		partial := javaAsCall(arg)
		if partial == nil {
			continue
		}
		name := javaCallName(partial, lowerer.input.Data)
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
			state.Behaviors = append(state.Behaviors, lowerer.parseBehaviorRefs(state.ID, behaviorKind(javaCallName(partial, lowerer.input.Data)), partial, "")...)
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

func (lowerer *javaLowerer) applyHistoryDefaultPartial(state *State, partial *sitter.Node, name string) bool {
	if state.Kind != StateKindShallowHistory && state.Kind != StateKindDeepHistory {
		return false
	}
	if len(state.Transitions) == 0 {
		state.Transitions = append(state.Transitions, Transition{OwnerID: state.ID, ID: lowerer.nextID("transition"), Range: lowerer.sourceRange(partial)})
	}
	transition := &state.Transitions[0]
	switch name {
	case "Target":
		transition.Target = javaFirstStringOrRaw(partial, lowerer.input.Data)
	case "Guard":
		refs := lowerer.parseBehaviorRefs(state.ID, BehaviorGuard, partial, "")
		if len(refs) > 0 {
			transition.Guard = &refs[0]
		}
	case "Effect":
		transition.Effects = append(transition.Effects, lowerer.parseBehaviorRefs(state.ID, BehaviorEffect, partial, "")...)
	}
	return true
}

func (lowerer *javaLowerer) parseInitial(ownerID string, call *sitter.Node) Initial {
	initial := Initial{OwnerID: ownerID, ID: lowerer.nextID("initial"), Range: lowerer.sourceRange(call)}
	for _, arg := range javaCallArgs(call) {
		partial := javaAsCall(arg)
		if partial == nil {
			continue
		}
		name := javaCallName(partial, lowerer.input.Data)
		switch name {
		case "Target":
			initial.Target = javaFirstStringOrRaw(partial, lowerer.input.Data)
		case "Effect":
			initial.Effects = append(initial.Effects, lowerer.parseBehaviorRefs(ownerID, BehaviorEffect, partial, "")...)
		default:
			lowerer.addUnknownHSMPartialDiagnostic("initial", partial, name)
		}
	}
	return initial
}

func (lowerer *javaLowerer) parseTransition(ownerID string, call *sitter.Node) Transition {
	transition := Transition{OwnerID: ownerID, ID: lowerer.nextID("transition"), Range: lowerer.sourceRange(call)}
	for _, arg := range javaCallArgs(call) {
		partial := javaAsCall(arg)
		if partial == nil {
			continue
		}
		name := javaCallName(partial, lowerer.input.Data)
		switch name {
		case "Source":
			transition.Source = javaFirstStringOrRaw(partial, lowerer.input.Data)
		case "Target":
			transition.Target = javaFirstStringOrRaw(partial, lowerer.input.Data)
		case "On":
			transition.Trigger = lowerer.parseOnTrigger(partial)
		case "OnSet":
			transition.Trigger = &Trigger{Kind: TriggerOnSet, Value: javaFirstStringOrRaw(partial, lowerer.input.Data), IsString: true, Language: LanguageJava}
		case "OnCall":
			transition.Trigger = &Trigger{Kind: TriggerOnCall, Value: javaFirstStringOrRaw(partial, lowerer.input.Data), IsString: true, Language: LanguageJava}
		case "After":
			transition.Trigger = lowerer.parseValueTrigger(TriggerAfter, partial)
		case "Every":
			transition.Trigger = lowerer.parseValueTrigger(TriggerEvery, partial)
		case "At":
			transition.Trigger = lowerer.parseValueTrigger(TriggerAt, partial)
		case "When":
			behaviors := lowerer.parseBehaviors(ownerID, BehaviorTrigger, partial, TriggerWhen)
			if len(behaviors) > 0 {
				lowerer.program.Behaviors = append(lowerer.program.Behaviors, behaviors...)
				transition.Trigger = &Trigger{Kind: TriggerWhen, Language: LanguageJava, Expr: &BehaviorRef{ID: behaviors[0].ID, Kind: BehaviorTrigger}}
			}
		case "Guard":
			refs := lowerer.parseBehaviorRefs(ownerID, BehaviorGuard, partial, "")
			if len(refs) > 0 {
				transition.Guard = &refs[0]
			}
		case "Effect":
			transition.Effects = append(transition.Effects, lowerer.parseBehaviorRefs(ownerID, BehaviorEffect, partial, "")...)
		default:
			lowerer.addUnknownHSMPartialDiagnostic("transition", partial, name)
		}
	}
	return transition
}

func (lowerer *javaLowerer) parseOnTrigger(call *sitter.Node) *Trigger {
	trigger := &Trigger{Kind: TriggerOn, Language: LanguageJava}
	for _, arg := range javaCallArgs(call) {
		value, isString := javaNodeArgValue(arg, lowerer.input.Data)
		eventSymbol := ""
		if !isString {
			if lowerer.eventBySymbol(value) != nil {
				eventSymbol = value
			} else if symbol, ok := eventMemberSymbol(value, ".Name", ".name"); ok && lowerer.eventBySymbol(symbol) != nil {
				eventSymbol = symbol
			}
		}
		trigger.Values = append(trigger.Values, TriggerValue{Value: value, IsString: isString, EventSymbol: eventSymbol, Language: LanguageJava})
		if trigger.Value == "" {
			trigger.Value = value
			trigger.IsString = isString
		}
	}
	if len(trigger.Values) <= 1 && (len(trigger.Values) == 0 || trigger.Values[0].EventSymbol == "") {
		trigger.Values = nil
	}
	return trigger
}

func (lowerer *javaLowerer) parseValueTrigger(kind TriggerKind, call *sitter.Node) *Trigger {
	value, isString := javaFirstArgValue(call, lowerer.input.Data)
	return &Trigger{Kind: kind, Value: value, IsString: isString, Language: LanguageJava}
}

func (lowerer *javaLowerer) parseBehaviors(ownerID string, kind BehaviorKind, call *sitter.Node, triggerKind TriggerKind) []Behavior {
	var behaviors []Behavior
	for _, arg := range javaCallArgs(call) {
		if arg.Kind() == "lambda_expression" || arg.Kind() == "method_reference" || arg.Kind() == "identifier" {
			behaviors = append(behaviors, lowerer.parseBehavior(ownerID, kind, arg, triggerKind))
		}
	}
	return behaviors
}

func (lowerer *javaLowerer) parseBehaviorRefs(ownerID string, kind BehaviorKind, call *sitter.Node, triggerKind TriggerKind) []BehaviorRef {
	var refs []BehaviorRef
	for _, arg := range javaCallArgs(call) {
		if operation, ok := javaStringLiteral(arg, lowerer.input.Data); ok {
			refs = append(refs, BehaviorRef{Kind: kind, Operation: operation})
			continue
		}
		if arg.Kind() == "lambda_expression" || arg.Kind() == "method_reference" || arg.Kind() == "identifier" {
			behavior := lowerer.parseBehavior(ownerID, kind, arg, triggerKind)
			lowerer.program.Behaviors = append(lowerer.program.Behaviors, behavior)
			refs = append(refs, BehaviorRef{ID: behavior.ID, Kind: behavior.Kind})
		}
	}
	return refs
}

func (lowerer *javaLowerer) parseBehavior(ownerID string, kind BehaviorKind, node *sitter.Node, triggerKind TriggerKind) Behavior {
	inline := node.Kind() == "lambda_expression"
	if resolved := lowerer.referencedBehaviorFunction(node, kind); resolved != nil {
		return lowerer.parseRegisteredBehavior(ownerID, kind, triggerKind, *resolved)
	}
	idPrefix := string(kind)
	body := strings.TrimSpace(node.Utf8Text(lowerer.input.Data))
	if block := firstNamedChildOfKind(node, "block"); block != nil {
		body = trimBlock(block.Utf8Text(lowerer.input.Data))
	}
	return Behavior{
		ID:             lowerer.nextID(idPrefix),
		OwnerID:        ownerID,
		Kind:           kind,
		TriggerKind:    triggerKind,
		Inline:         inline,
		SourceLanguage: LanguageJava,
		Signature:      "lambda",
		Body:           body,
		Code:           strings.TrimSpace(node.Utf8Text(lowerer.input.Data)),
		Range:          lowerer.sourceRange(node),
	}
}

type javaFunctionDecl struct {
	name        string
	node        *sitter.Node
	body        *sitter.Node
	globalRange SourceRange
}

func (lowerer *javaLowerer) parseRegisteredBehavior(ownerID string, kind BehaviorKind, triggerKind TriggerKind, decl javaFunctionDecl) Behavior {
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
		TriggerKind:    triggerKind,
		SourceLanguage: LanguageJava,
		Body:           body,
		Code:           strings.TrimSpace(decl.node.Utf8Text(lowerer.input.Data)),
		Signature:      signature,
		Range:          lowerer.sourceRange(decl.node),
	}
}

func (lowerer *javaLowerer) registerFunctionDecl(node *sitter.Node, globalRange SourceRange) {
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
		lowerer.functions = map[string]javaFunctionDecl{}
	}
	lowerer.functions[name] = javaFunctionDecl{name: name, node: node, body: body, globalRange: globalRange}
}

func (lowerer *javaLowerer) referencedBehaviorFunction(node *sitter.Node, kind BehaviorKind) *javaFunctionDecl {
	if node == nil || (node.Kind() != "identifier" && node.Kind() != "method_reference") {
		return nil
	}
	name := strings.TrimSpace(node.Utf8Text(lowerer.input.Data))
	if index := strings.LastIndex(name, "::"); index >= 0 {
		name = name[index+2:]
	}
	if isJavaGeneratedBehaviorFunctionNameForAnyKind(name) && !isJavaGeneratedBehaviorFunctionName(name, kind) {
		return nil
	}
	decl, ok := lowerer.functions[name]
	if !ok {
		return nil
	}
	if !isJavaGeneratedBehaviorFunctionName(name, kind) && !lowerer.functionDeclLooksLikeBehavior(decl) {
		return nil
	}
	if lowerer.usedFunctions == nil {
		lowerer.usedFunctions = map[string]bool{}
	}
	lowerer.usedFunctions[name] = true
	return &decl
}

func (lowerer *javaLowerer) functionDeclLooksLikeBehavior(decl javaFunctionDecl) bool {
	if decl.node == nil || decl.body == nil {
		return false
	}
	signature := string(lowerer.input.Data[decl.node.StartByte():decl.body.StartByte()])
	return strings.Contains(signature, "Event")
}

func (lowerer *javaLowerer) filterReferencedFunctionGlobals() {
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

func (lowerer *javaLowerer) isGeneratedEmptyConstructor(node *sitter.Node) bool {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		nameNode = firstNamedChildOfKind(node, "identifier")
	}
	if nameNode == nil || nameNode.Utf8Text(lowerer.input.Data) != "GeneratedHsm" {
		return false
	}
	if !strings.HasPrefix(strings.TrimSpace(node.Utf8Text(lowerer.input.Data)), "private ") {
		return false
	}
	body := node.ChildByFieldName("body")
	if body == nil {
		body = firstNamedChildOfKind(node, "block")
	}
	return body != nil && strings.TrimSpace(trimBlock(body.Utf8Text(lowerer.input.Data))) == ""
}

func isJavaGeneratedBehaviorFunctionNameForAnyKind(name string) bool {
	for _, kind := range []BehaviorKind{
		BehaviorEntry,
		BehaviorExit,
		BehaviorActivity,
		BehaviorGuard,
		BehaviorEffect,
		BehaviorTrigger,
		BehaviorOperation,
	} {
		if isJavaGeneratedBehaviorFunctionName(name, kind) {
			return true
		}
	}
	return false
}

func isJavaGeneratedBehaviorFunctionName(name string, kind BehaviorKind) bool {
	prefix := exportName(string(kind))
	if kind == BehaviorOperation {
		prefix = "Operation"
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

func (lowerer *javaLowerer) parseDefers(call *sitter.Node) []string {
	var values []string
	for _, arg := range javaCallArgs(call) {
		value := javaNodeTextOrString(arg, lowerer.input.Data)
		if event := lowerer.eventBySymbol(value); event != nil {
			values = append(values, event.Name)
			continue
		}
		if symbol, ok := eventMemberSymbol(value, ".Name", ".name"); ok {
			if event := lowerer.eventBySymbol(symbol); event != nil {
				values = append(values, event.Name)
				continue
			}
		}
		values = append(values, value)
	}
	return values
}

func (lowerer *javaLowerer) parseEventDecl(node *sitter.Node) (EventDecl, bool) {
	for _, declarator := range javaVariableDeclarators(node) {
		if event, ok := lowerer.parseEventDeclarator(declarator); ok {
			return event, true
		}
	}
	return EventDecl{}, false
}

func (lowerer *javaLowerer) parseEventDeclarator(declarator *sitter.Node) (EventDecl, bool) {
	if declarator.NamedChildCount() < 2 {
		return EventDecl{}, false
	}
	symbol := strings.TrimSpace(declarator.NamedChild(0).Utf8Text(lowerer.input.Data))
	initializer := declarator.NamedChild(1)
	if initializer.Kind() != "object_creation_expression" || !strings.Contains(initializer.Utf8Text(lowerer.input.Data), "Event") {
		return EventDecl{}, false
	}
	args := javaObjectArgs(initializer)
	if len(args) == 0 {
		return EventDecl{}, false
	}
	name, ok := javaStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		name = strings.TrimSpace(args[0].Utf8Text(lowerer.input.Data))
	}
	return EventDecl{ID: lowerer.nextID("event"), Symbol: symbol, Name: name, Language: LanguageJava, Range: lowerer.sourceRange(declarator)}, true
}

func (lowerer *javaLowerer) eventBySymbol(symbol string) *EventDecl {
	for index := range lowerer.program.Events {
		if lowerer.program.Events[index].Symbol == symbol {
			return &lowerer.program.Events[index]
		}
	}
	return nil
}

func (lowerer *javaLowerer) parseImport(node *sitter.Node) (Import, bool) {
	text := strings.TrimSpace(node.Utf8Text(lowerer.input.Data))
	text = strings.TrimPrefix(text, "import ")
	isStatic := strings.HasPrefix(text, "static ")
	text = strings.TrimPrefix(text, "static ")
	text = strings.TrimSuffix(text, ";")
	text = strings.TrimSpace(text)
	wildcard := strings.HasSuffix(text, ".*")
	text = strings.TrimSuffix(text, ".*")
	if text == "" {
		return Import{}, false
	}
	imp := Import{Path: text, Static: isStatic, LocalNames: javaImportLocalNames(text), Language: LanguageJava, Range: lowerer.sourceRange(node)}
	if wildcard {
		imp.Specifiers = []ImportSpecifier{{Name: "*"}}
		imp.LocalNames = []string{"*"}
	}
	return imp, true
}

func (lowerer *javaLowerer) recordHSMImport(imp Import) {
	if imp.Path == "com.stateforward.hsm.Hsm" && len(imp.LocalNames) == 1 && imp.LocalNames[0] == "*" {
		lowerer.hsmStatic = true
	}
}

func (lowerer *javaLowerer) addUnknownHSMPartialDiagnostic(context string, call *sitter.Node, name string) {
	if !lowerer.isHSMRuntimeCall(call) {
		return
	}
	lowerer.diagnostics = append(lowerer.diagnostics, Diagnostic{
		Message: fmt.Sprintf("unknown or unsupported Hsm.%s partial in %s context", name, context),
		Range:   lowerer.sourceRange(call),
	})
}

func (lowerer *javaLowerer) isHSMRuntimeCall(call *sitter.Node) bool {
	if call == nil || call.Kind() != "method_invocation" {
		return false
	}
	var qualifier string
	var name string
	for i := uint(0); i < call.NamedChildCount(); i++ {
		child := call.NamedChild(i)
		if child.Kind() != "identifier" {
			continue
		}
		if name == "" {
			name = child.Utf8Text(lowerer.input.Data)
			continue
		}
		qualifier = name
		name = child.Utf8Text(lowerer.input.Data)
	}
	if qualifier != "" {
		return qualifier == "Hsm"
	}
	return lowerer.hsmStatic
}

func javaImportLocalNames(path string) []string {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return nil
	}
	return []string{parts[len(parts)-1]}
}

func (lowerer *javaLowerer) globalBlock(node *sitter.Node) CodeBlock {
	return CodeBlock{ID: lowerer.nextID("global"), Language: LanguageJava, Code: strings.TrimSpace(node.Utf8Text(lowerer.input.Data)), Range: lowerer.sourceRange(node)}
}

func (lowerer *javaLowerer) globalDeclaratorBlock(prefix string, declarator *sitter.Node) CodeBlock {
	code := strings.TrimSpace(prefix)
	if code != "" {
		code += " "
	}
	code += strings.TrimSpace(declarator.Utf8Text(lowerer.input.Data))
	if !strings.HasSuffix(code, ";") {
		code += ";"
	}
	return CodeBlock{ID: lowerer.nextID("global"), Language: LanguageJava, Code: code, Range: lowerer.sourceRange(declarator)}
}

func javaFieldPrefix(node *sitter.Node, firstDeclarator *sitter.Node, source []byte) string {
	if firstDeclarator == nil {
		return ""
	}
	return strings.TrimSpace(string(source[node.StartByte():firstDeclarator.StartByte()]))
}

func javaVariableDeclarators(node *sitter.Node) []*sitter.Node {
	var declarators []*sitter.Node
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child.Kind() == "variable_declarator" {
			declarators = append(declarators, child)
		}
	}
	return declarators
}

func javaObjectArgs(node *sitter.Node) []*sitter.Node {
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child.Kind() == "argument_list" {
			return javaArgumentListChildren(child)
		}
	}
	return nil
}

func javaAsCall(node *sitter.Node) *sitter.Node {
	if node == nil || node.Kind() != "method_invocation" {
		return nil
	}
	return node
}

func javaCallArgs(call *sitter.Node) []*sitter.Node {
	args := firstNamedChildOfKind(call, "argument_list")
	if args == nil {
		return nil
	}
	return javaArgumentListChildren(args)
}

func javaArgumentListChildren(args *sitter.Node) []*sitter.Node {
	var result []*sitter.Node
	for i := uint(0); i < args.NamedChildCount(); i++ {
		result = append(result, args.NamedChild(i))
	}
	return result
}

func javaCallName(call *sitter.Node, source []byte) string {
	var last string
	for i := uint(0); i < call.NamedChildCount(); i++ {
		child := call.NamedChild(i)
		if child.Kind() == "identifier" {
			last = child.Utf8Text(source)
		}
	}
	return last
}

func javaFirstStringOrRaw(call *sitter.Node, source []byte) string {
	value, _ := javaFirstArgValue(call, source)
	return value
}

func javaFirstArgValue(call *sitter.Node, source []byte) (string, bool) {
	args := javaCallArgs(call)
	if len(args) == 0 {
		return "", false
	}
	return javaNodeArgValue(args[0], source)
}

func javaNodeArgValue(node *sitter.Node, source []byte) (string, bool) {
	if value, ok := javaStringLiteral(node, source); ok {
		return value, true
	}
	return strings.TrimSpace(node.Utf8Text(source)), false
}

func javaNodeTextOrString(node *sitter.Node, source []byte) string {
	value, _ := javaNodeArgValue(node, source)
	return value
}

func javaStringLiteral(node *sitter.Node, source []byte) (string, bool) {
	if node == nil || node.Kind() != "string_literal" {
		return "", false
	}
	value, err := strconv.Unquote(node.Utf8Text(source))
	return value, err == nil
}

func javaNodeContainsCall(node *sitter.Node, source []byte, name string) bool {
	found := false
	walk(node, func(child *sitter.Node) {
		if child.Kind() == "method_invocation" && javaCallName(child, source) == name {
			found = true
		}
	})
	return found
}

func (lowerer *javaLowerer) sourceRange(node *sitter.Node) SourceRange {
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

func (lowerer *javaLowerer) nextID(prefix string) string {
	if lowerer.counters == nil {
		lowerer.counters = map[string]int{}
	}
	lowerer.counters[prefix]++
	return fmt.Sprintf("%s_%d", prefix, lowerer.counters[prefix])
}
