package hsmc

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type RustFrontend struct{}

func NewRustFrontend() RustFrontend {
	return RustFrontend{}
}

func (RustFrontend) Language() Language { return LanguageRust }

func (frontend RustFrontend) Parse(ctx context.Context, input SourceInput) (*Program, error) {
	tree, err := parseTree(ctx, input, parserGrammarRust)
	if err != nil {
		return nil, err
	}
	defer tree.Close()
	lowerer := rustLowerer{
		input: SourceInput{Path: input.Path, Data: input.Data},
		program: &Program{
			SourceLanguage: LanguageRust,
			PackageName:    "main",
		},
	}
	root := tree.RootNode()
	lowerer.collectTopLevel(root)
	lowerer.collectModels(root)
	lowerer.filterReferencedFunctionGlobals()
	lowerer.filterGeneratedScaffoldGlobals()
	if len(lowerer.diagnostics) > 0 {
		return nil, lowerer.diagnostics
	}
	if len(lowerer.program.Models) == 0 {
		return nil, fmt.Errorf("no define! model found in %s", input.Path)
	}
	return lowerer.program, nil
}

type rustLowerer struct {
	input         SourceInput
	program       *Program
	counters      map[string]int
	functions     map[string]rustFunctionDecl
	usedFunctions map[string]bool
	hsmGlob       bool
	hsmMacroNames map[string]bool
	diagnostics   Diagnostics
}

func (lowerer *rustLowerer) collectTopLevel(root *sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		child := root.NamedChild(i)
		switch child.Kind() {
		case "use_declaration":
			if imp, ok := lowerer.parseUse(child); ok {
				if isHSMImportPath(imp.Path) {
					lowerer.recordHSMUse(imp)
					continue
				}
				lowerer.program.Imports = append(lowerer.program.Imports, imp)
			}
		case "function_item":
			if rustNodeContainsMacro(child, lowerer.input.Data, "define") {
				continue
			}
			block := lowerer.globalBlock(child)
			lowerer.registerFunctionDecl(child, block.Range)
			lowerer.program.Globals = append(lowerer.program.Globals, block)
		case "struct_item", "enum_item", "impl_item", "type_item":
			block := lowerer.globalBlock(child)
			lowerer.program.Globals = append(lowerer.program.Globals, block)
		case "const_item", "let_declaration":
			if !rustNodeContainsMacro(child, lowerer.input.Data, "define") {
				if event, ok := lowerer.parseEventDecl(child); ok {
					lowerer.program.Events = append(lowerer.program.Events, event)
					continue
				}
				lowerer.program.Globals = append(lowerer.program.Globals, lowerer.globalBlock(child))
			}
		}
	}
}

func (lowerer *rustLowerer) collectModels(root *sitter.Node) {
	walk(root, func(node *sitter.Node) {
		if node.Kind() != "macro_invocation" || rustCanonicalMacroName(node.Utf8Text(lowerer.input.Data)) != "Define" {
			return
		}
		if model, ok := lowerer.parseModel(node); ok {
			lowerer.program.Models = append(lowerer.program.Models, model)
		}
	})
}

func (lowerer *rustLowerer) parseModel(macro *sitter.Node) (Model, bool) {
	args := rustMacroArgs(macro.Utf8Text(lowerer.input.Data))
	if len(args) == 0 {
		return Model{}, false
	}
	name, ok := rustStringArg(args[0])
	if !ok {
		return Model{}, false
	}
	model := Model{ID: lowerer.nextID("model"), Name: name, Range: lowerer.sourceRange(macro)}
	for _, arg := range args[1:] {
		lowerer.applyModelPartial(&model, arg)
	}
	return model, true
}

func (lowerer *rustLowerer) applyModelPartial(model *Model, arg string) {
	name := rustCanonicalMacroName(arg)
	switch name {
	case "Initial":
		initial := lowerer.parseInitial(model.ID, arg)
		model.Initializer = &initial
	case "State", "Final", "Choice", "ShallowHistory", "DeepHistory":
		if state, ok := lowerer.parseState(model.ID, arg); ok {
			model.States = append(model.States, state)
		}
	case "Transition":
		model.Transitions = append(model.Transitions, lowerer.parseTransition(model.ID, arg))
	case "Attribute":
		if attribute, ok := lowerer.parseAttribute(arg); ok {
			model.Attributes = append(model.Attributes, attribute)
		}
	case "Operation":
		if operation, ok := lowerer.parseOperation(model.ID, arg); ok {
			model.Operations = append(model.Operations, operation)
		}
	default:
		lowerer.addUnknownHSMPartialDiagnostic("model", arg, name)
	}
}

func (lowerer *rustLowerer) parseAttribute(call string) (Attribute, bool) {
	args := rustMacroArgs(call)
	if len(args) == 0 {
		return Attribute{}, false
	}
	name, ok := rustStringArg(args[0])
	if !ok {
		return Attribute{}, false
	}
	attribute := Attribute{ID: lowerer.nextID("attribute"), Name: name, Language: LanguageRust}
	if len(args) == 2 {
		field, value := classifyAttributeSecondArg(LanguageRust, args[1])
		if field == "type" {
			attribute.Type = value
		} else {
			attribute.HasDefault = true
			attribute.Default = value
		}
	} else if len(args) > 2 {
		attribute.Type = strings.TrimSpace(args[1])
		attribute.HasDefault = true
		attribute.Default = strings.TrimSpace(args[2])
	}
	return attribute, true
}

func (lowerer *rustLowerer) parseOperation(ownerID string, call string) (Operation, bool) {
	args := rustMacroArgs(call)
	if len(args) < 2 {
		return Operation{}, false
	}
	name, ok := rustStringArg(args[0])
	if !ok {
		return Operation{}, false
	}
	operation := Operation{ID: lowerer.nextID("operation_decl"), Name: name}
	behavior := lowerer.parseBehavior(ownerID, BehaviorOperation, args[1])
	lowerer.program.Behaviors = append(lowerer.program.Behaviors, behavior)
	operation.Behavior = &BehaviorRef{ID: behavior.ID, Kind: BehaviorOperation}
	return operation, true
}

func (lowerer *rustLowerer) parseState(ownerID string, call string) (State, bool) {
	args := rustMacroArgs(call)
	if len(args) == 0 {
		return State{}, false
	}
	name, ok := rustStringArg(args[0])
	if !ok {
		return State{}, false
	}
	state := State{ID: lowerer.nextID("state"), Name: name, Kind: rustStateKind(rustCanonicalMacroName(call))}
	for _, arg := range args[1:] {
		name := rustCanonicalMacroName(arg)
		switch name {
		case "Initial":
			initial := lowerer.parseInitial(state.ID, arg)
			state.Initializer = &initial
		case "State", "Final", "Choice", "ShallowHistory", "DeepHistory":
			if child, ok := lowerer.parseState(state.ID, arg); ok {
				state.States = append(state.States, child)
			}
		case "Transition":
			state.Transitions = append(state.Transitions, lowerer.parseTransition(state.ID, arg))
		case "Entry", "Exit", "Activity":
			state.Behaviors = append(state.Behaviors, lowerer.parseBehaviors(state.ID, rustBehaviorKind(name), arg)...)
		case "Defer":
			state.Defers = append(state.Defers, lowerer.parseDefers(arg)...)
		case "Target", "Guard", "Effect":
			if !lowerer.applyHistoryDefaultPartial(&state, arg, name) {
				lowerer.addUnknownHSMPartialDiagnostic("state", arg, name)
			}
		default:
			lowerer.addUnknownHSMPartialDiagnostic("state", arg, name)
		}
	}
	_ = ownerID
	return state, true
}

func (lowerer *rustLowerer) applyHistoryDefaultPartial(state *State, arg string, name string) bool {
	if state.Kind != StateKindShallowHistory && state.Kind != StateKindDeepHistory {
		return false
	}
	if len(state.Transitions) == 0 {
		state.Transitions = append(state.Transitions, Transition{OwnerID: state.ID, ID: lowerer.nextID("transition")})
	}
	transition := &state.Transitions[0]
	switch name {
	case "Target":
		transition.Target = rustFirstStringOrRaw(arg)
	case "Guard":
		refs := lowerer.parseBehaviors(state.ID, BehaviorGuard, arg)
		if len(refs) > 0 {
			transition.Guard = &refs[0]
		}
	case "Effect":
		transition.Effects = append(transition.Effects, lowerer.parseBehaviors(state.ID, BehaviorEffect, arg)...)
	}
	return true
}

func (lowerer *rustLowerer) parseDefers(call string) []string {
	raw := rustMacroArgs(call)
	values := make([]string, 0, len(raw))
	for _, arg := range raw {
		if value, ok := rustStringArg(arg); ok {
			values = append(values, value)
			continue
		}
		value := strings.TrimSpace(arg)
		if event := lowerer.eventBySymbol(value); event != nil {
			values = append(values, event.Name)
			continue
		}
		values = append(values, value)
	}
	return values
}

func (lowerer *rustLowerer) parseInitial(ownerID string, call string) Initial {
	initial := Initial{OwnerID: ownerID, ID: lowerer.nextID("initial")}
	for _, arg := range rustMacroArgs(call) {
		name := rustCanonicalMacroName(arg)
		switch name {
		case "Target":
			initial.Target = rustFirstStringOrRaw(arg)
		case "Effect":
			initial.Effects = append(initial.Effects, lowerer.parseBehaviors(ownerID, BehaviorEffect, arg)...)
		default:
			lowerer.addUnknownHSMPartialDiagnostic("initial", arg, name)
		}
	}
	return initial
}

func (lowerer *rustLowerer) parseTransition(ownerID string, call string) Transition {
	transition := Transition{OwnerID: ownerID, ID: lowerer.nextID("transition")}
	for _, arg := range rustMacroArgs(call) {
		name := rustCanonicalMacroName(arg)
		switch name {
		case "Source":
			transition.Source = rustFirstStringOrRaw(arg)
		case "Target":
			transition.Target = rustFirstStringOrRaw(arg)
		case "On":
			transition.Trigger = lowerer.parseOnTrigger(arg)
		case "OnSet":
			transition.Trigger = &Trigger{Kind: TriggerOnSet, Value: rustFirstStringOrRaw(arg), IsString: true, Language: LanguageRust}
		case "OnCall":
			transition.Trigger = &Trigger{Kind: TriggerOnCall, Value: rustFirstStringOrRaw(arg), IsString: true, Language: LanguageRust}
		case "After":
			transition.Trigger = lowerer.parseExpressionTrigger(ownerID, TriggerAfter, arg)
		case "Every":
			transition.Trigger = lowerer.parseExpressionTrigger(ownerID, TriggerEvery, arg)
		case "At":
			transition.Trigger = lowerer.parseExpressionTrigger(ownerID, TriggerAt, arg)
		case "When":
			transition.Trigger = lowerer.parseExpressionTrigger(ownerID, TriggerWhen, arg)
		case "Guard":
			refs := lowerer.parseBehaviors(ownerID, BehaviorGuard, arg)
			if len(refs) > 0 {
				transition.Guard = &refs[0]
			}
		case "Effect":
			transition.Effects = append(transition.Effects, lowerer.parseBehaviors(ownerID, BehaviorEffect, arg)...)
		default:
			lowerer.addUnknownHSMPartialDiagnostic("transition", arg, name)
		}
	}
	return transition
}

func (lowerer *rustLowerer) parseOnTrigger(call string) *Trigger {
	trigger := &Trigger{Kind: TriggerOn, Language: LanguageRust}
	for _, arg := range rustMacroArgs(call) {
		if value, ok := rustStringArg(arg); ok {
			trigger.Values = append(trigger.Values, TriggerValue{Value: value, IsString: true, Language: LanguageRust})
			if trigger.Value == "" {
				trigger.Value = value
				trigger.IsString = true
			}
			continue
		}
		value := strings.TrimSpace(arg)
		if event := lowerer.eventBySymbol(value); event != nil {
			trigger.Values = append(trigger.Values, TriggerValue{Value: event.Name, EventSymbol: value, Language: LanguageRust})
			if trigger.Value == "" {
				trigger.Value = event.Name
			}
			continue
		}
		trigger.Values = append(trigger.Values, TriggerValue{Value: value, EventSymbol: value, Language: LanguageRust})
		if trigger.Value == "" {
			trigger.Value = value
		}
	}
	return trigger
}

func (lowerer *rustLowerer) parseExpressionTrigger(ownerID string, kind TriggerKind, call string) *Trigger {
	args := rustMacroArgs(call)
	if len(args) == 0 {
		return &Trigger{Kind: kind, Language: LanguageRust}
	}
	value := strings.TrimSpace(args[0])
	if stringValue, ok := rustStringArg(value); ok {
		return &Trigger{Kind: kind, Value: stringValue, IsString: true, Language: LanguageRust}
	}
	behavior := lowerer.parseBehavior(ownerID, BehaviorTrigger, value)
	behavior.TriggerKind = kind
	lowerer.program.Behaviors = append(lowerer.program.Behaviors, behavior)
	return &Trigger{Kind: kind, Value: value, Language: LanguageRust, Expr: &BehaviorRef{ID: behavior.ID, Kind: BehaviorTrigger}}
}

func (lowerer *rustLowerer) parseBehaviors(ownerID string, kind BehaviorKind, call string) []BehaviorRef {
	var refs []BehaviorRef
	for _, arg := range rustMacroArgs(call) {
		if stringValue, ok := rustStringArg(arg); ok {
			refs = append(refs, BehaviorRef{Kind: kind, Operation: stringValue})
			continue
		}
		behavior := lowerer.parseBehavior(ownerID, kind, arg)
		lowerer.program.Behaviors = append(lowerer.program.Behaviors, behavior)
		refs = append(refs, BehaviorRef{ID: behavior.ID, Kind: kind})
	}
	return refs
}

func (lowerer *rustLowerer) parseBehavior(ownerID string, kind BehaviorKind, raw string) Behavior {
	if resolved := lowerer.referencedBehaviorFunction(raw, kind); resolved != nil {
		return lowerer.parseRegisteredBehavior(ownerID, kind, *resolved)
	}
	code := strings.TrimSpace(raw)
	body := rustBehaviorCallBody(kind, code)
	inline := strings.HasPrefix(code, "|") || strings.HasPrefix(code, "move |")
	return Behavior{
		ID:             lowerer.nextID(string(kind)),
		OwnerID:        ownerID,
		Kind:           kind,
		Inline:         inline,
		SourceLanguage: LanguageRust,
		Body:           body,
		Code:           code,
	}
}

type rustFunctionDecl struct {
	name        string
	node        *sitter.Node
	body        *sitter.Node
	globalRange SourceRange
}

func (lowerer *rustLowerer) parseRegisteredBehavior(ownerID string, kind BehaviorKind, decl rustFunctionDecl) Behavior {
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
		SourceLanguage: LanguageRust,
		Body:           body,
		Code:           strings.TrimSpace(decl.node.Utf8Text(lowerer.input.Data)),
		Signature:      signature,
		Range:          lowerer.sourceRange(decl.node),
	}
}

func (lowerer *rustLowerer) registerFunctionDecl(node *sitter.Node, globalRange SourceRange) {
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
		lowerer.functions = map[string]rustFunctionDecl{}
	}
	lowerer.functions[name] = rustFunctionDecl{name: name, node: node, body: body, globalRange: globalRange}
}

func (lowerer *rustLowerer) referencedBehaviorFunction(raw string, kind BehaviorKind) *rustFunctionDecl {
	name := strings.TrimSpace(raw)
	if index := strings.LastIndex(name, "::"); index >= 0 {
		name = name[index+2:]
	}
	if isRustGeneratedBehaviorFunctionNameForAnyKind(name) && !isRustGeneratedBehaviorFunctionName(name, kind) {
		return nil
	}
	decl, ok := lowerer.functions[name]
	if !ok {
		return nil
	}
	if !isRustGeneratedBehaviorFunctionName(name, kind) && !lowerer.functionDeclLooksLikeBehavior(decl) {
		return nil
	}
	if lowerer.usedFunctions == nil {
		lowerer.usedFunctions = map[string]bool{}
	}
	lowerer.usedFunctions[name] = true
	return &decl
}

func (lowerer *rustLowerer) functionDeclLooksLikeBehavior(decl rustFunctionDecl) bool {
	if decl.node == nil || decl.body == nil {
		return false
	}
	signature := string(lowerer.input.Data[decl.node.StartByte():decl.body.StartByte()])
	return strings.Contains(signature, "hsm::Event")
}

func (lowerer *rustLowerer) filterReferencedFunctionGlobals() {
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

func (lowerer *rustLowerer) filterGeneratedScaffoldGlobals() {
	globals := lowerer.program.Globals[:0]
	for _, global := range lowerer.program.Globals {
		if isGeneratedRustScaffoldGlobal(global.Code) {
			continue
		}
		globals = append(globals, global)
	}
	lowerer.program.Globals = globals
}

func isGeneratedRustScaffoldGlobal(code string) bool {
	code = strings.TrimSpace(code)
	if code == "pub struct HsmcInstance;" || code == "struct HsmcInstance;" {
		return true
	}
	return strings.Contains(code, "impl hsm::Instance for HsmcInstance") &&
		strings.Contains(code, "fn as_any(&self)") &&
		strings.Contains(code, "fn as_any_mut(&mut self)")
}

func isRustGeneratedBehaviorFunctionNameForAnyKind(name string) bool {
	for _, kind := range []BehaviorKind{
		BehaviorEntry,
		BehaviorExit,
		BehaviorActivity,
		BehaviorGuard,
		BehaviorEffect,
		BehaviorTrigger,
		BehaviorOperation,
	} {
		if isRustGeneratedBehaviorFunctionName(name, kind) {
			return true
		}
	}
	return false
}

func isRustGeneratedBehaviorFunctionName(name string, kind BehaviorKind) bool {
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

func rustBehaviorCallBody(kind BehaviorKind, code string) string {
	if strings.Contains(code, "|") || strings.Contains(code, "{") {
		return code
	}
	switch kind {
	case BehaviorGuard:
		return fmt.Sprintf("return %s(ctx, instance, event);", code)
	case BehaviorTrigger:
		return fmt.Sprintf("return %s(ctx, instance, event);", code)
	default:
		return fmt.Sprintf("%s(ctx, instance, event);", code)
	}
}

func (lowerer *rustLowerer) parseUse(node *sitter.Node) (Import, bool) {
	text := strings.TrimSpace(node.Utf8Text(lowerer.input.Data))
	text = strings.TrimPrefix(text, "use ")
	text = strings.TrimSuffix(text, ";")
	text = strings.TrimSpace(text)
	if text == "" {
		return Import{}, false
	}
	imp := Import{Path: text, Language: LanguageRust, Range: lowerer.sourceRange(node)}
	if path, specifiers, ok := parseRustGroupedUse(text); ok {
		imp.Path = path
		imp.Specifiers = specifiers
		imp.LocalNames = normalizeLocalNames(imp)
		return imp, true
	}
	if strings.Contains(text, " as ") {
		parts := strings.SplitN(text, " as ", 2)
		imp.Path = strings.TrimSpace(parts[0])
		imp.Alias = strings.TrimSpace(parts[1])
	}
	imp.LocalNames = normalizeLocalNames(imp)
	return imp, true
}

func parseRustGroupedUse(text string) (string, []ImportSpecifier, bool) {
	open := strings.LastIndex(text, "::{")
	if open < 0 || !strings.HasSuffix(text, "}") {
		return "", nil, false
	}
	path := strings.TrimSpace(text[:open])
	members := strings.TrimSpace(text[open+3 : len(text)-1])
	if path == "" || members == "" {
		return "", nil, false
	}
	var specifiers []ImportSpecifier
	for _, member := range splitRustArgs(members) {
		member = strings.TrimSpace(member)
		if member == "" {
			continue
		}
		specifier := ImportSpecifier{Name: member}
		if left, right, ok := strings.Cut(member, " as "); ok {
			specifier.Name = strings.TrimSpace(left)
			specifier.Alias = strings.TrimSpace(right)
		}
		if specifier.Name != "" {
			specifiers = append(specifiers, specifier)
		}
	}
	return path, specifiers, len(specifiers) > 0
}

func (lowerer *rustLowerer) recordHSMUse(imp Import) {
	if imp.Path == "hsm::*" {
		lowerer.hsmGlob = true
		return
	}
	if imp.Path != "hsm" {
		return
	}
	for _, specifier := range imp.Specifiers {
		if specifier.Name == "*" {
			lowerer.hsmGlob = true
			continue
		}
		name := specifier.Name
		if specifier.Alias != "" {
			name = specifier.Alias
		}
		if name == "" {
			continue
		}
		if lowerer.hsmMacroNames == nil {
			lowerer.hsmMacroNames = map[string]bool{}
		}
		lowerer.hsmMacroNames[name] = true
	}
}

func (lowerer *rustLowerer) addUnknownHSMPartialDiagnostic(context string, call string, name string) {
	if name == "" || !lowerer.isHSMRuntimeMacro(call) {
		return
	}
	lowerer.diagnostics = append(lowerer.diagnostics, Diagnostic{
		Message: fmt.Sprintf("unknown or unsupported hsm::%s! partial in %s context", name, context),
	})
}

func (lowerer *rustLowerer) isHSMRuntimeMacro(call string) bool {
	rawName := rustRawMacroCallName(call)
	if rawName == "" {
		return false
	}
	if index := strings.LastIndex(rawName, "::"); index >= 0 {
		return strings.TrimSpace(rawName[:index]) == "hsm"
	}
	return lowerer.hsmGlob || lowerer.hsmMacroNames[rawName]
}

func (lowerer *rustLowerer) parseEventDecl(node *sitter.Node) (EventDecl, bool) {
	text := strings.TrimSpace(node.Utf8Text(lowerer.input.Data))
	if !strings.HasPrefix(text, "const ") && !strings.HasPrefix(text, "pub const ") {
		return EventDecl{}, false
	}
	decl := strings.TrimPrefix(text, "pub ")
	decl = strings.TrimPrefix(decl, "const ")
	parts := strings.SplitN(decl, "=", 2)
	if len(parts) != 2 {
		return EventDecl{}, false
	}
	left := strings.TrimSpace(parts[0])
	if !strings.Contains(left, "&str") {
		return EventDecl{}, false
	}
	symbol := strings.TrimSpace(strings.SplitN(left, ":", 2)[0])
	if symbol == "" {
		return EventDecl{}, false
	}
	right := strings.TrimSpace(strings.TrimSuffix(parts[1], ";"))
	name, ok := rustStringArg(right)
	if !ok {
		return EventDecl{}, false
	}
	return EventDecl{ID: lowerer.nextID("event"), Symbol: symbol, Name: name, Language: LanguageRust, Range: lowerer.sourceRange(node)}, true
}

func (lowerer *rustLowerer) eventBySymbol(symbol string) *EventDecl {
	for index := range lowerer.program.Events {
		if lowerer.program.Events[index].Symbol == symbol {
			return &lowerer.program.Events[index]
		}
	}
	return nil
}

func (lowerer *rustLowerer) globalBlock(node *sitter.Node) CodeBlock {
	return CodeBlock{
		ID:       lowerer.nextID("global"),
		Language: LanguageRust,
		Code:     strings.TrimSpace(node.Utf8Text(lowerer.input.Data)),
		Range:    lowerer.sourceRange(node),
	}
}

func (lowerer *rustLowerer) sourceRange(node *sitter.Node) SourceRange {
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

func (lowerer *rustLowerer) nextID(prefix string) string {
	if lowerer.counters == nil {
		lowerer.counters = map[string]int{}
	}
	lowerer.counters[prefix]++
	return fmt.Sprintf("%s_%d", prefix, lowerer.counters[prefix])
}

func rustStateKind(name string) StateKind {
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

func rustBehaviorKind(name string) BehaviorKind {
	switch name {
	case "Exit":
		return BehaviorExit
	case "Activity":
		return BehaviorActivity
	default:
		return BehaviorEntry
	}
}

func rustNodeContainsMacro(node *sitter.Node, source []byte, name string) bool {
	found := false
	canonical := rustCanonicalMacroName(name + "!")
	walk(node, func(child *sitter.Node) {
		if child.Kind() == "macro_invocation" && rustCanonicalMacroName(child.Utf8Text(source)) == canonical {
			found = true
		}
	})
	return found
}

func rustMacroName(node *sitter.Node, source []byte) string {
	return rustMacroCallName(node.Utf8Text(source))
}

func rustRawMacroCallName(text string) string {
	text = strings.TrimSpace(text)
	bang := strings.Index(text, "!")
	if bang < 0 {
		return ""
	}
	return strings.TrimSpace(text[:bang])
}

func rustMacroCallName(text string) string {
	name := rustRawMacroCallName(text)
	if index := strings.LastIndex(name, "::"); index >= 0 {
		name = name[index+2:]
	}
	return name
}

func rustCanonicalMacroName(text string) string {
	switch strings.ToLower(rustMacroCallName(text)) {
	case "define":
		return "Define"
	case "initial":
		return "Initial"
	case "state":
		return "State"
	case "final", "final_state":
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
		return rustMacroCallName(text)
	}
}

func rustMacroArgs(text string) []string {
	text = strings.TrimSpace(text)
	bang := strings.Index(text, "!")
	if bang < 0 {
		return nil
	}
	rest := strings.TrimSpace(text[bang+1:])
	start := strings.Index(rest, "(")
	if start < 0 {
		return nil
	}
	content, ok := balancedContent(rest[start:])
	if !ok {
		return nil
	}
	return splitRustArgs(content)
}

func balancedContent(text string) (string, bool) {
	if !strings.HasPrefix(text, "(") {
		return "", false
	}
	depth := 0
	inString := false
	escaped := false
	for index, r := range text {
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == '"' {
				inString = false
			}
			continue
		}
		switch r {
		case '"':
			inString = true
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
			if depth == 0 {
				return text[1:index], true
			}
		}
	}
	return "", false
}

func splitRustArgs(text string) []string {
	var args []string
	start := 0
	depth := 0
	inString := false
	escaped := false
	for index, r := range text {
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == '"' {
				inString = false
			}
			continue
		}
		switch r {
		case '"':
			inString = true
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				if arg := strings.TrimSpace(text[start:index]); arg != "" {
					args = append(args, arg)
				}
				start = index + len(string(r))
			}
		}
	}
	if arg := strings.TrimSpace(text[start:]); arg != "" {
		args = append(args, arg)
	}
	return args
}

func rustStringArg(text string) (string, bool) {
	text = strings.TrimSpace(text)
	value, err := strconv.Unquote(text)
	return value, err == nil
}

func rustStringOrRawArgs(call string) []string {
	raw := rustMacroArgs(call)
	values := make([]string, 0, len(raw))
	for _, arg := range raw {
		if value, ok := rustStringArg(arg); ok {
			values = append(values, value)
		} else {
			values = append(values, strings.TrimSpace(arg))
		}
	}
	return values
}

func rustFirstStringOrRaw(call string) string {
	args := rustStringOrRawArgs(call)
	if len(args) == 0 {
		return ""
	}
	return args[0]
}
