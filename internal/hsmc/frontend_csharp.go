package hsmc

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type CSharpFrontend struct{}

func NewCSharpFrontend() CSharpFrontend {
	return CSharpFrontend{}
}

func (CSharpFrontend) Language() Language { return LanguageCSharp }

func (frontend CSharpFrontend) Parse(ctx context.Context, input SourceInput) (*Program, error) {
	tree, err := parseTree(ctx, input, parserGrammarCSharp)
	if err != nil {
		return nil, err
	}
	defer tree.Close()
	lowerer := csharpLowerer{
		input: SourceInput{Path: input.Path, Data: input.Data},
		program: &Program{
			SourceLanguage: LanguageCSharp,
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

type csharpLowerer struct {
	input             SourceInput
	program           *Program
	counters          map[string]int
	functions         map[string]csharpFunctionDecl
	usedFunctions     map[string]bool
	hsmRuntimeAliases map[string]bool
	hsmStaticRuntime  bool
	diagnostics       Diagnostics
}

func (lowerer *csharpLowerer) collectTopLevel(root *sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		lowerer.collectTopLevelNode(root.NamedChild(i))
	}
}

func (lowerer *csharpLowerer) collectTopLevelNode(node *sitter.Node) {
	switch node.Kind() {
	case "using_directive":
		if imp, ok := lowerer.parseUsing(node); ok {
			if isHSMImportPath(imp.Path) {
				lowerer.recordHSMRuntimeAlias(imp)
				return
			}
			if imp.Path != "System" {
				lowerer.program.Imports = append(lowerer.program.Imports, imp)
			}
		}
	case "class_declaration", "record_declaration", "struct_declaration":
		for i := uint(0); i < node.NamedChildCount(); i++ {
			child := node.NamedChild(i)
			if child.Kind() != "declaration_list" {
				continue
			}
			for j := uint(0); j < child.NamedChildCount(); j++ {
				lowerer.collectTopLevelNode(child.NamedChild(j))
			}
		}
	case "field_declaration", "local_declaration_statement", "global_statement":
		if lowerer.collectVariableDeclaration(node) {
			return
		}
	case "method_declaration", "property_declaration", "constructor_declaration":
		block := lowerer.globalBlock(node)
		if node.Kind() == "method_declaration" {
			lowerer.registerFunctionDecl(node, block.Range)
		}
		lowerer.program.Globals = append(lowerer.program.Globals, block)
	}
}

func (lowerer *csharpLowerer) collectVariableDeclaration(node *sitter.Node) bool {
	if !csharpNodeContainsCall(node, lowerer.input.Data, "Define") {
		if event, ok := lowerer.parseEventDecl(node); ok {
			lowerer.program.Events = append(lowerer.program.Events, event)
			return true
		}
		lowerer.program.Globals = append(lowerer.program.Globals, lowerer.globalBlock(node))
		return true
	}
	declarators := csharpVariableDeclarators(node)
	if len(declarators) == 0 {
		return true
	}
	prefix := lowerer.csharpDeclarationPrefix(node, declarators[0])
	for _, declarator := range declarators {
		if csharpNodeContainsCall(declarator, lowerer.input.Data, "Define") {
			continue
		}
		if event, ok := lowerer.parseEventDecl(declarator); ok {
			lowerer.program.Events = append(lowerer.program.Events, event)
			continue
		}
		lowerer.program.Globals = append(lowerer.program.Globals, lowerer.globalDeclaratorBlock(prefix, declarator))
	}
	return true
}

func (lowerer *csharpLowerer) collectModels(root *sitter.Node) {
	walk(root, func(node *sitter.Node) {
		if node.Kind() != "invocation_expression" || csharpCanonicalCallName(node, lowerer.input.Data) != "Define" {
			return
		}
		if model, ok := lowerer.parseModel(node); ok {
			lowerer.program.Models = append(lowerer.program.Models, model)
		}
	})
}

func (lowerer *csharpLowerer) parseModel(call *sitter.Node) (Model, bool) {
	args := csharpCallArgs(call)
	if len(args) == 0 {
		return Model{}, false
	}
	name, ok := csharpStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Model{}, false
	}
	model := Model{ID: lowerer.nextID("model"), Name: name, Range: lowerer.sourceRange(call)}
	for _, arg := range args[1:] {
		partial := csharpAsCall(arg)
		if partial != nil {
			lowerer.applyModelPartial(&model, partial)
		}
	}
	return model, true
}

func (lowerer *csharpLowerer) applyModelPartial(model *Model, call *sitter.Node) {
	name := csharpCanonicalCallName(call, lowerer.input.Data)
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

func (lowerer *csharpLowerer) parseAttribute(call *sitter.Node) (Attribute, bool) {
	args := csharpCallArgs(call)
	if len(args) == 0 {
		return Attribute{}, false
	}
	name, ok := csharpStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Attribute{}, false
	}
	attribute := Attribute{ID: lowerer.nextID("attribute"), Name: name, Language: LanguageCSharp, Range: lowerer.sourceRange(call)}
	if len(args) == 2 {
		field, value := classifyAttributeSecondArg(LanguageCSharp, args[1].Utf8Text(lowerer.input.Data))
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

func (lowerer *csharpLowerer) parseOperation(ownerID string, call *sitter.Node) (Operation, bool) {
	args := csharpCallArgs(call)
	if len(args) < 2 {
		return Operation{}, false
	}
	name, ok := csharpStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Operation{}, false
	}
	behavior := lowerer.parseBehavior(ownerID, BehaviorOperation, csharpOperationImplementationArg(args[1]))
	lowerer.program.Behaviors = append(lowerer.program.Behaviors, behavior)
	return Operation{ID: lowerer.nextID("operation_decl"), Name: name, Behavior: &BehaviorRef{ID: behavior.ID, Kind: BehaviorOperation}, Range: lowerer.sourceRange(call)}, true
}

func (lowerer *csharpLowerer) parseState(ownerID string, call *sitter.Node) (State, bool) {
	args := csharpCallArgs(call)
	if len(args) == 0 {
		return State{}, false
	}
	name, ok := csharpStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return State{}, false
	}
	state := State{ID: lowerer.nextID("state"), Name: name, Kind: csharpStateKind(csharpCanonicalCallName(call, lowerer.input.Data)), Range: lowerer.sourceRange(call)}
	for _, arg := range args[1:] {
		partial := csharpAsCall(arg)
		if partial == nil {
			continue
		}
		name := csharpCanonicalCallName(partial, lowerer.input.Data)
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
			state.Behaviors = append(state.Behaviors, lowerer.parseBehaviors(state.ID, behaviorKind(csharpCanonicalCallName(partial, lowerer.input.Data)), partial)...)
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

func (lowerer *csharpLowerer) applyHistoryDefaultPartial(state *State, partial *sitter.Node, name string) bool {
	if state.Kind != StateKindShallowHistory && state.Kind != StateKindDeepHistory {
		return false
	}
	if len(state.Transitions) == 0 {
		state.Transitions = append(state.Transitions, Transition{OwnerID: state.ID, ID: lowerer.nextID("transition"), Range: lowerer.sourceRange(partial)})
	}
	transition := &state.Transitions[0]
	switch name {
	case "Target":
		transition.Target = csharpFirstStringOrRaw(partial, lowerer.input.Data)
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

func (lowerer *csharpLowerer) parseInitial(ownerID string, call *sitter.Node) Initial {
	initial := Initial{OwnerID: ownerID, ID: lowerer.nextID("initial"), Range: lowerer.sourceRange(call)}
	for _, arg := range csharpCallArgs(call) {
		partial := csharpAsCall(arg)
		if partial == nil {
			continue
		}
		name := csharpCanonicalCallName(partial, lowerer.input.Data)
		switch name {
		case "Target":
			initial.Target = csharpFirstStringOrRaw(partial, lowerer.input.Data)
		case "Effect":
			initial.Effects = append(initial.Effects, lowerer.parseBehaviors(ownerID, BehaviorEffect, partial)...)
		default:
			lowerer.addUnknownHSMPartialDiagnostic("initial", partial, name)
		}
	}
	return initial
}

func (lowerer *csharpLowerer) parseTransition(ownerID string, call *sitter.Node) Transition {
	transition := Transition{OwnerID: ownerID, ID: lowerer.nextID("transition"), Range: lowerer.sourceRange(call)}
	for _, arg := range csharpCallArgs(call) {
		partial := csharpAsCall(arg)
		if partial == nil {
			continue
		}
		name := csharpCanonicalCallName(partial, lowerer.input.Data)
		switch name {
		case "Source":
			transition.Source = csharpFirstStringOrRaw(partial, lowerer.input.Data)
		case "Target":
			transition.Target = csharpFirstStringOrRaw(partial, lowerer.input.Data)
		case "On":
			transition.Trigger = lowerer.parseOnTrigger(partial)
		case "OnSet":
			transition.Trigger = &Trigger{Kind: TriggerOnSet, Value: csharpFirstStringOrRaw(partial, lowerer.input.Data), IsString: true, Language: LanguageCSharp}
		case "OnCall":
			transition.Trigger = &Trigger{Kind: TriggerOnCall, Value: csharpFirstStringOrRaw(partial, lowerer.input.Data), IsString: true, Language: LanguageCSharp}
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

func (lowerer *csharpLowerer) parseOnTrigger(call *sitter.Node) *Trigger {
	trigger := &Trigger{Kind: TriggerOn, Language: LanguageCSharp}
	for _, arg := range csharpCallArgs(call) {
		value, isString := csharpNodeArgValue(arg, lowerer.input.Data)
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
		trigger.Values = append(trigger.Values, TriggerValue{Value: value, IsString: isString, EventSymbol: eventSymbol, Language: LanguageCSharp})
	}
	if len(trigger.Values) <= 1 && (len(trigger.Values) == 0 || trigger.Values[0].EventSymbol == "") {
		trigger.Values = nil
	}
	return trigger
}

func (lowerer *csharpLowerer) parseExpressionTrigger(ownerID string, kind TriggerKind, call *sitter.Node) *Trigger {
	value, isString := csharpFirstArgValue(call, lowerer.input.Data)
	trigger := &Trigger{Kind: kind, Value: value, IsString: isString, Language: LanguageCSharp}
	args := csharpCallArgs(call)
	if isString || len(args) == 0 {
		return trigger
	}
	behavior := lowerer.parseBehavior(ownerID, BehaviorTrigger, args[0])
	behavior.TriggerKind = kind
	lowerer.program.Behaviors = append(lowerer.program.Behaviors, behavior)
	trigger.Expr = &BehaviorRef{ID: behavior.ID, Kind: BehaviorTrigger}
	return trigger
}

func (lowerer *csharpLowerer) parseBehaviors(ownerID string, kind BehaviorKind, call *sitter.Node) []BehaviorRef {
	var refs []BehaviorRef
	for _, arg := range csharpCallArgs(call) {
		if operation, ok := csharpStringLiteral(arg, lowerer.input.Data); ok {
			refs = append(refs, BehaviorRef{Kind: kind, Operation: operation})
			continue
		}
		behavior := lowerer.parseBehavior(ownerID, kind, arg)
		lowerer.program.Behaviors = append(lowerer.program.Behaviors, behavior)
		refs = append(refs, BehaviorRef{ID: behavior.ID, Kind: kind})
	}
	return refs
}

func (lowerer *csharpLowerer) parseBehavior(ownerID string, kind BehaviorKind, node *sitter.Node) Behavior {
	inline := node.Kind() == "lambda_expression" || node.Kind() == "anonymous_method_expression"
	if resolved := lowerer.referencedBehaviorFunction(node, kind); resolved != nil {
		return lowerer.parseRegisteredBehavior(ownerID, kind, *resolved)
	}
	code := strings.TrimSpace(lowerer.normalizeHSMRuntimeAliases(node.Utf8Text(lowerer.input.Data)))
	behavior := Behavior{
		ID:             lowerer.nextID(string(kind)),
		OwnerID:        ownerID,
		Kind:           kind,
		Inline:         inline,
		SourceLanguage: LanguageCSharp,
		Code:           code,
		Range:          lowerer.sourceRange(node),
	}
	if node.Kind() == "lambda_expression" {
		behavior.Signature, behavior.Body = csharpLambdaSignatureAndBody(kind, code)
	} else if node.Kind() == "identifier" || node.Kind() == "member_access_expression" {
		behavior.Body = csharpCallableReferenceBody(kind, code)
	}
	return behavior
}

type csharpFunctionDecl struct {
	name        string
	node        *sitter.Node
	body        *sitter.Node
	globalRange SourceRange
}

func (lowerer *csharpLowerer) parseRegisteredBehavior(ownerID string, kind BehaviorKind, decl csharpFunctionDecl) Behavior {
	body := ""
	signature := ""
	if decl.body != nil {
		body = trimBlock(lowerer.normalizeHSMRuntimeAliases(decl.body.Utf8Text(lowerer.input.Data)))
		signature = strings.TrimSpace(lowerer.normalizeHSMRuntimeAliases(string(lowerer.input.Data[decl.node.StartByte():decl.body.StartByte()])))
	}
	return Behavior{
		ID:             lowerer.nextID(string(kind)),
		OwnerID:        ownerID,
		Kind:           kind,
		SourceLanguage: LanguageCSharp,
		Body:           body,
		Code:           strings.TrimSpace(lowerer.normalizeHSMRuntimeAliases(decl.node.Utf8Text(lowerer.input.Data))),
		Signature:      signature,
		Range:          lowerer.sourceRange(decl.node),
	}
}

func (lowerer *csharpLowerer) registerFunctionDecl(node *sitter.Node, globalRange SourceRange) {
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
		lowerer.functions = map[string]csharpFunctionDecl{}
	}
	lowerer.functions[name] = csharpFunctionDecl{name: name, node: node, body: body, globalRange: globalRange}
}

func (lowerer *csharpLowerer) referencedBehaviorFunction(node *sitter.Node, kind BehaviorKind) *csharpFunctionDecl {
	if node == nil || (node.Kind() != "identifier" && node.Kind() != "member_access_expression") {
		return nil
	}
	name := strings.TrimSpace(node.Utf8Text(lowerer.input.Data))
	if index := strings.LastIndex(name, "."); index >= 0 {
		name = name[index+1:]
	}
	if isCSharpGeneratedBehaviorFunctionNameForAnyKind(name) && !isCSharpGeneratedBehaviorFunctionName(name, kind) {
		return nil
	}
	decl, ok := lowerer.functions[name]
	if !ok {
		return nil
	}
	if !isCSharpGeneratedBehaviorFunctionName(name, kind) && !lowerer.functionDeclLooksLikeBehavior(decl) {
		return nil
	}
	if lowerer.usedFunctions == nil {
		lowerer.usedFunctions = map[string]bool{}
	}
	lowerer.usedFunctions[name] = true
	return &decl
}

func (lowerer *csharpLowerer) functionDeclLooksLikeBehavior(decl csharpFunctionDecl) bool {
	if decl.node == nil || decl.body == nil {
		return false
	}
	signature := lowerer.normalizeHSMRuntimeAliases(string(lowerer.input.Data[decl.node.StartByte():decl.body.StartByte()]))
	return strings.Contains(signature, "Event")
}

func (lowerer *csharpLowerer) filterReferencedFunctionGlobals() {
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

func isCSharpGeneratedBehaviorFunctionNameForAnyKind(name string) bool {
	for _, kind := range []BehaviorKind{
		BehaviorEntry,
		BehaviorExit,
		BehaviorActivity,
		BehaviorGuard,
		BehaviorEffect,
		BehaviorTrigger,
		BehaviorOperation,
	} {
		if isCSharpGeneratedBehaviorFunctionName(name, kind) {
			return true
		}
	}
	return false
}

func isCSharpGeneratedBehaviorFunctionName(name string, kind BehaviorKind) bool {
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

func csharpLambdaSignatureAndBody(kind BehaviorKind, code string) (string, string) {
	left, right, ok := strings.Cut(code, "=>")
	if !ok {
		return "", ""
	}
	signature := strings.TrimSpace(left)
	expr := strings.TrimSpace(right)
	if strings.HasPrefix(expr, "{") {
		return signature, trimBlock(expr)
	}
	if kind == BehaviorGuard || kind == BehaviorTrigger {
		return signature, "return " + strings.TrimSuffix(expr, ";") + ";"
	}
	return signature, strings.TrimSuffix(expr, ";") + ";"
}

func csharpCallableReferenceBody(kind BehaviorKind, name string) string {
	if name == "" {
		if kind == BehaviorGuard {
			return "return false;"
		}
		if kind == BehaviorTrigger {
			return "return default!;"
		}
		return ""
	}
	call := fmt.Sprintf("%s(ctx, instance, @event)", name)
	if kind == BehaviorGuard || kind == BehaviorTrigger {
		return "return " + call + ";"
	}
	return call + ";"
}

func (lowerer *csharpLowerer) parseDefers(call *sitter.Node) []string {
	var defers []string
	for _, arg := range csharpCallArgs(call) {
		value, isString := csharpNodeArgValue(arg, lowerer.input.Data)
		if !isString {
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
		}
		defers = append(defers, value)
	}
	return defers
}

func (lowerer *csharpLowerer) parseEventDecl(node *sitter.Node) (EventDecl, bool) {
	nameNode := csharpDeclarationName(node)
	valueNode := csharpDeclarationValue(node)
	if nameNode == nil || valueNode == nil {
		return EventDecl{}, false
	}
	var eventName string
	walk(valueNode, func(child *sitter.Node) {
		if eventName != "" || child.Kind() != "string_literal" {
			return
		}
		if value, ok := csharpStringLiteral(child, lowerer.input.Data); ok {
			eventName = value
		}
	})
	if eventName == "" || !strings.Contains(valueNode.Utf8Text(lowerer.input.Data), "Event") {
		return EventDecl{}, false
	}
	symbol := nameNode.Utf8Text(lowerer.input.Data)
	return EventDecl{ID: lowerer.nextID("event"), Symbol: symbol, Name: eventName, Language: LanguageCSharp, Range: lowerer.sourceRange(node)}, true
}

func (lowerer *csharpLowerer) parseUsing(node *sitter.Node) (Import, bool) {
	if node == nil {
		return Import{}, false
	}
	text := strings.TrimSpace(node.Utf8Text(lowerer.input.Data))
	text = strings.TrimSuffix(text, ";")
	text = strings.TrimSpace(strings.TrimPrefix(text, "global "))
	text = strings.TrimSpace(strings.TrimPrefix(text, "using "))
	isStatic := strings.HasPrefix(text, "static ")
	text = strings.TrimSpace(strings.TrimPrefix(text, "static "))
	alias := ""
	if left, right, ok := strings.Cut(text, "="); ok {
		alias = strings.TrimSpace(left)
		text = strings.TrimSpace(right)
	}
	path := strings.TrimSpace(text)
	if path == "" {
		return Import{}, false
	}
	imp := Import{Path: path, Alias: alias, Static: isStatic, Language: LanguageCSharp, Range: lowerer.sourceRange(node)}
	if alias != "" {
		imp.LocalNames = []string{alias}
	} else if isStatic {
		imp.LocalNames = []string{"*"}
	}
	return imp, true
}

func (lowerer *csharpLowerer) recordHSMRuntimeAlias(imp Import) {
	if imp.Path != "Stateforward.Hsm.Hsm" {
		return
	}
	if len(imp.LocalNames) == 1 && imp.LocalNames[0] == "*" {
		lowerer.hsmStaticRuntime = true
		return
	}
	if imp.Alias == "" || imp.Alias == "Hsm" {
		return
	}
	if lowerer.hsmRuntimeAliases == nil {
		lowerer.hsmRuntimeAliases = map[string]bool{}
	}
	lowerer.hsmRuntimeAliases[imp.Alias] = true
}

func (lowerer *csharpLowerer) normalizeHSMRuntimeAliases(code string) string {
	for alias := range lowerer.hsmRuntimeAliases {
		pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(alias) + `\.`)
		code = pattern.ReplaceAllString(code, "Hsm.")
	}
	return code
}

func (lowerer *csharpLowerer) addUnknownHSMPartialDiagnostic(context string, call *sitter.Node, name string) {
	if !lowerer.isHSMRuntimeCall(call) {
		return
	}
	lowerer.diagnostics = append(lowerer.diagnostics, Diagnostic{
		Message: fmt.Sprintf("unknown or unsupported Hsm.%s partial in %s context", name, context),
		Range:   lowerer.sourceRange(call),
	})
}

func (lowerer *csharpLowerer) isHSMRuntimeCall(call *sitter.Node) bool {
	function := call.ChildByFieldName("function")
	if function == nil && call.NamedChildCount() > 0 {
		function = call.NamedChild(0)
	}
	if function == nil {
		return false
	}
	text := strings.TrimSpace(function.Utf8Text(lowerer.input.Data))
	if index := strings.Index(text, "<"); index >= 0 {
		text = strings.TrimSpace(text[:index])
	}
	if index := strings.LastIndex(text, "."); index >= 0 {
		qualifier := strings.TrimSpace(text[:index])
		return qualifier == "Hsm" || lowerer.hsmRuntimeAliases[qualifier]
	}
	return lowerer.hsmStaticRuntime
}

func (lowerer *csharpLowerer) globalBlock(node *sitter.Node) CodeBlock {
	code := lowerer.normalizeHSMRuntimeAliases(node.Utf8Text(lowerer.input.Data))
	switch node.Kind() {
	case "field_declaration", "local_declaration_statement":
		code = strings.TrimSpace(code)
		if strings.HasPrefix(code, "static readonly ") {
			code = "private " + code
		}
	case "method_declaration":
		code = strings.TrimSpace(code)
		if strings.HasPrefix(code, "static ") {
			code = "private " + code
		}
	}
	return CodeBlock{ID: lowerer.nextID("global"), Language: LanguageCSharp, Code: code, Range: lowerer.sourceRange(node)}
}

func (lowerer *csharpLowerer) globalDeclaratorBlock(prefix string, declarator *sitter.Node) CodeBlock {
	code := strings.TrimSpace(prefix)
	if code != "" {
		code += " "
	}
	code += strings.TrimSpace(declarator.Utf8Text(lowerer.input.Data))
	code = lowerer.normalizeHSMRuntimeAliases(code)
	if !strings.HasSuffix(code, ";") {
		code += ";"
	}
	return CodeBlock{ID: lowerer.nextID("global"), Language: LanguageCSharp, Code: code, Range: lowerer.sourceRange(declarator)}
}

func (lowerer *csharpLowerer) csharpDeclarationPrefix(node *sitter.Node, firstDeclarator *sitter.Node) string {
	prefix := strings.TrimSpace(string(lowerer.input.Data[node.StartByte():firstDeclarator.StartByte()]))
	if strings.HasPrefix(prefix, "static readonly ") {
		prefix = "private " + prefix
	}
	return prefix
}

func (lowerer *csharpLowerer) eventBySymbol(symbol string) *EventDecl {
	for index := range lowerer.program.Events {
		if lowerer.program.Events[index].Symbol == symbol {
			return &lowerer.program.Events[index]
		}
	}
	return nil
}

func (lowerer *csharpLowerer) sourceRange(node *sitter.Node) SourceRange {
	start := node.StartPosition()
	end := node.EndPosition()
	return SourceRange{File: lowerer.input.Path, StartByte: node.StartByte(), EndByte: node.EndByte(), StartLine: uint(start.Row) + 1, EndLine: uint(end.Row) + 1}
}

func (lowerer *csharpLowerer) nextID(prefix string) string {
	if lowerer.counters == nil {
		lowerer.counters = map[string]int{}
	}
	lowerer.counters[prefix]++
	return fmt.Sprintf("%s_%d", prefix, lowerer.counters[prefix])
}

func csharpAsCall(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Kind() == "argument" && node.NamedChildCount() == 1 {
		node = node.NamedChild(0)
	}
	if node.Kind() == "invocation_expression" {
		return node
	}
	return nil
}

func csharpCallArgs(call *sitter.Node) []*sitter.Node {
	args := firstNamedChildOfKind(call, "argument_list")
	if args == nil {
		return nil
	}
	var result []*sitter.Node
	for i := uint(0); i < args.NamedChildCount(); i++ {
		arg := args.NamedChild(i)
		if arg.Kind() == "argument" && arg.NamedChildCount() == 1 {
			arg = arg.NamedChild(0)
		}
		result = append(result, arg)
	}
	return result
}

func csharpOperationImplementationArg(node *sitter.Node) *sitter.Node {
	if node == nil {
		return node
	}
	switch node.Kind() {
	case "object_creation_expression", "implicit_object_creation_expression":
	default:
		return node
	}
	args := firstNamedChildOfKind(node, "argument_list")
	if args == nil || args.NamedChildCount() == 0 {
		return node
	}
	arg := args.NamedChild(0)
	if arg.Kind() == "argument" && arg.NamedChildCount() == 1 {
		arg = arg.NamedChild(0)
	}
	return arg
}

func csharpCallName(call *sitter.Node, source []byte) string {
	if call == nil || call.Kind() != "invocation_expression" || call.NamedChildCount() == 0 {
		return ""
	}
	function := call.NamedChild(0)
	text := function.Utf8Text(source)
	if index := strings.LastIndex(text, "."); index >= 0 {
		text = text[index+1:]
	}
	if index := strings.Index(text, "<"); index >= 0 {
		text = text[:index]
	}
	return strings.TrimSpace(text)
}

func csharpCanonicalCallName(call *sitter.Node, source []byte) string {
	switch strings.ToLower(csharpCallName(call, source)) {
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
	case "shallowhistory", "history":
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
		return csharpCallName(call, source)
	}
}

func csharpStateKind(name string) StateKind {
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

func csharpFirstStringOrRaw(call *sitter.Node, source []byte) string {
	value, _ := csharpFirstArgValue(call, source)
	return value
}

func csharpFirstArgValue(call *sitter.Node, source []byte) (string, bool) {
	args := csharpCallArgs(call)
	if len(args) == 0 {
		return "", false
	}
	return csharpNodeArgValue(args[0], source)
}

func csharpNodeArgValue(node *sitter.Node, source []byte) (string, bool) {
	if value, ok := csharpStringLiteral(node, source); ok {
		return value, true
	}
	return strings.TrimSpace(node.Utf8Text(source)), false
}

func csharpStringLiteral(node *sitter.Node, source []byte) (string, bool) {
	if node == nil || node.Kind() != "string_literal" {
		return "", false
	}
	text := strings.TrimSpace(node.Utf8Text(source))
	if strings.HasPrefix(text, "$") {
		return "", false
	}
	text = strings.TrimPrefix(text, "@")
	if value, err := strconv.Unquote(text); err == nil {
		return value, true
	}
	if node.NamedChildCount() > 0 {
		return strings.Trim(node.NamedChild(0).Utf8Text(source), "\""), true
	}
	return "", false
}

func csharpVariableDeclarators(node *sitter.Node) []*sitter.Node {
	var declarators []*sitter.Node
	walk(node, func(child *sitter.Node) {
		if child.Kind() == "variable_declarator" {
			declarators = append(declarators, child)
		}
	})
	return declarators
}

func csharpDeclarationName(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	var declarator *sitter.Node
	walk(node, func(child *sitter.Node) {
		if declarator == nil && child.Kind() == "variable_declarator" {
			declarator = child
		}
	})
	if declarator == nil || declarator.NamedChildCount() == 0 {
		return nil
	}
	return declarator.NamedChild(0)
}

func csharpDeclarationValue(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	var declarator *sitter.Node
	walk(node, func(child *sitter.Node) {
		if declarator == nil && child.Kind() == "variable_declarator" {
			declarator = child
		}
	})
	if declarator == nil || declarator.NamedChildCount() < 2 {
		return nil
	}
	return declarator.NamedChild(1)
}

func csharpNodeContainsCall(node *sitter.Node, source []byte, name string) bool {
	var found bool
	walk(node, func(child *sitter.Node) {
		if found || child.Kind() != "invocation_expression" {
			return
		}
		found = strings.EqualFold(csharpCallName(child, source), name)
	})
	return found
}
