package hsmc

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type PythonFrontend struct{}

func NewPythonFrontend() PythonFrontend {
	return PythonFrontend{}
}

func (PythonFrontend) Language() Language { return LanguagePython }

func (frontend PythonFrontend) Parse(ctx context.Context, input SourceInput) (*Program, error) {
	tree, err := parseTree(ctx, input, parserGrammarPython)
	if err != nil {
		return nil, err
	}
	defer tree.Close()
	root := tree.RootNode()
	lowerer := pyLowerer{
		input: SourceInput{Path: input.Path, Data: input.Data},
		program: &Program{
			SourceLanguage: LanguagePython,
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

type pyLowerer struct {
	input            SourceInput
	program          *Program
	counters         map[string]int
	functions        map[string]pyFunctionDecl
	usedFunctions    map[string]bool
	hsmNamedBindings map[string]string
	diagnostics      Diagnostics
}

func (lowerer *pyLowerer) collectTopLevel(root *sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		child := root.NamedChild(i)
		switch child.Kind() {
		case "import_statement", "import_from_statement":
			imports := lowerer.parseImports(child)
			for _, imp := range imports {
				if isHSMImportPath(imp.Path) {
					lowerer.recordHSMImportBindings(imp)
				}
			}
			lowerer.program.Imports = append(lowerer.program.Imports, filterCompilerOwnedImports(imports)...)
		case "function_definition", "class_definition", "decorated_definition":
			block := lowerer.globalBlock(child)
			if child.Kind() == "function_definition" {
				lowerer.registerFunctionDecl(child, block.Range)
			}
			lowerer.program.Globals = append(lowerer.program.Globals, block)
		case "expression_statement":
			if lowerer.collectExpressionStatement(child) {
				continue
			}
		}
	}
}

func (lowerer *pyLowerer) collectExpressionStatement(node *sitter.Node) bool {
	if !lowerer.containsHSMDefine(node) {
		if event, ok := lowerer.parseEventDecl(node); ok {
			lowerer.program.Events = append(lowerer.program.Events, event)
			return true
		}
		lowerer.program.Globals = append(lowerer.program.Globals, lowerer.globalBlock(node))
		return true
	}
	assignments, ok := splitPythonGroupedAssignment(strings.TrimSpace(node.Utf8Text(lowerer.input.Data)))
	if !ok {
		return true
	}
	for _, assignment := range assignments {
		if lowerer.expressionContainsDefine(assignment.value) {
			continue
		}
		code := assignment.name + " = " + assignment.value
		if event, ok := lowerer.parseEventDeclText(code, node); ok {
			lowerer.program.Events = append(lowerer.program.Events, event)
			continue
		}
		lowerer.program.Globals = append(lowerer.program.Globals, lowerer.globalCodeBlock(code, node))
	}
	return true
}

func (lowerer *pyLowerer) collectModels(root *sitter.Node) {
	walk(root, func(node *sitter.Node) {
		if node.Kind() != "call" || lowerer.selectorName(node) != "Define" {
			return
		}
		model, ok := lowerer.parseModel(node)
		if ok {
			lowerer.program.Models = append(lowerer.program.Models, model)
		}
	})
}

func (lowerer *pyLowerer) parseModel(call *sitter.Node) (Model, bool) {
	args := callArgs(call)
	if len(args) == 0 {
		return Model{}, false
	}
	name, ok := pythonStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Model{}, false
	}
	model := Model{ID: lowerer.nextID("model"), Name: name, Range: lowerer.sourceRange(call)}
	for _, arg := range args[1:] {
		lowerer.applyModelPartial(&model, arg)
	}
	return model, true
}

func (lowerer *pyLowerer) applyModelPartial(model *Model, arg *sitter.Node) {
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

func (lowerer *pyLowerer) parseAttribute(call *sitter.Node) (Attribute, bool) {
	args := callArgs(call)
	if len(args) == 0 {
		return Attribute{}, false
	}
	name, ok := pythonStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Attribute{}, false
	}
	attribute := Attribute{ID: lowerer.nextID("attribute"), Name: name, Language: LanguagePython, Range: lowerer.sourceRange(call)}
	if len(args) == 2 {
		field, value := classifyAttributeSecondArg(LanguagePython, args[1].Utf8Text(lowerer.input.Data))
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

func (lowerer *pyLowerer) parseOperation(ownerID string, call *sitter.Node) (Operation, bool) {
	args := callArgs(call)
	if len(args) < 2 {
		return Operation{}, false
	}
	name, ok := pythonStringLiteral(args[0], lowerer.input.Data)
	if !ok {
		return Operation{}, false
	}
	operation := Operation{ID: lowerer.nextID("operation_decl"), Name: name, Range: lowerer.sourceRange(call)}
	behavior := lowerer.parseBehavior(ownerID, BehaviorOperation, args[1])
	lowerer.program.Behaviors = append(lowerer.program.Behaviors, behavior)
	operation.Behavior = &BehaviorRef{ID: behavior.ID, Kind: BehaviorOperation}
	return operation, true
}

func (lowerer *pyLowerer) parseState(ownerID string, call *sitter.Node) (State, bool) {
	args := callArgs(call)
	if len(args) == 0 {
		return State{}, false
	}
	name, ok := pythonStringLiteral(args[0], lowerer.input.Data)
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

func (lowerer *pyLowerer) applyHistoryDefaultPartial(state *State, partial *sitter.Node, name string) bool {
	if state.Kind != StateKindShallowHistory && state.Kind != StateKindDeepHistory {
		return false
	}
	if len(state.Transitions) == 0 {
		state.Transitions = append(state.Transitions, Transition{OwnerID: state.ID, ID: lowerer.nextID("transition"), Range: lowerer.sourceRange(partial)})
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

func (lowerer *pyLowerer) parseDefers(call *sitter.Node) []string {
	var defers []string
	for _, arg := range callArgs(call) {
		if value, ok := pythonStringLiteral(arg, lowerer.input.Data); ok {
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

func (lowerer *pyLowerer) parseInitial(ownerID string, call *sitter.Node) Initial {
	initial := Initial{OwnerID: ownerID, ID: lowerer.nextID("initial"), Range: lowerer.sourceRange(call)}
	for _, arg := range callArgs(call) {
		partial := asCall(arg)
		if partial == nil {
			continue
		}
		name := lowerer.selectorName(partial)
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

func (lowerer *pyLowerer) parseTransition(ownerID string, call *sitter.Node) Transition {
	transition := Transition{OwnerID: ownerID, ID: lowerer.nextID("transition"), Range: lowerer.sourceRange(call)}
	for _, arg := range callArgs(call) {
		partial := asCall(arg)
		if partial == nil {
			continue
		}
		name := lowerer.selectorName(partial)
		switch name {
		case "Source":
			transition.Source = lowerer.firstStringArg(partial)
		case "Target":
			transition.Target = lowerer.firstStringArg(partial)
		case "On":
			transition.Trigger = lowerer.parseOnTrigger(partial)
		case "OnSet":
			transition.Trigger = &Trigger{Kind: TriggerOnSet, Value: lowerer.firstStringArg(partial), IsString: true, Language: LanguagePython}
		case "OnCall":
			transition.Trigger = &Trigger{Kind: TriggerOnCall, Value: lowerer.firstStringArg(partial), IsString: true, Language: LanguagePython}
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

func (lowerer *pyLowerer) parseOnTrigger(call *sitter.Node) *Trigger {
	trigger := &Trigger{Kind: TriggerOn, Language: LanguagePython}
	for _, arg := range callArgs(call) {
		value, isString := lowerer.nodeArgValue(arg)
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
		trigger.Values = append(trigger.Values, TriggerValue{Value: value, IsString: isString, EventSymbol: eventSymbol, Language: LanguagePython})
	}
	if len(trigger.Values) <= 1 && (len(trigger.Values) == 0 || trigger.Values[0].EventSymbol == "") {
		trigger.Values = nil
	}
	return trigger
}

func (lowerer *pyLowerer) parseEventDecl(node *sitter.Node) (EventDecl, bool) {
	return lowerer.parseEventDeclText(strings.TrimSpace(node.Utf8Text(lowerer.input.Data)), node)
}

func (lowerer *pyLowerer) parseEventDeclText(text string, node *sitter.Node) (EventDecl, bool) {
	if !strings.Contains(text, "=") || !lowerer.expressionContainsEvent(text) {
		return EventDecl{}, false
	}
	before, after, ok := strings.Cut(text, "=")
	if !ok {
		return EventDecl{}, false
	}
	symbol := strings.TrimSpace(before)
	if symbol == "" || strings.ContainsAny(symbol, " \t()[]{}") {
		return EventDecl{}, false
	}
	name, ok := lowerer.eventNameFromCall(after)
	if !ok {
		return EventDecl{}, false
	}
	return EventDecl{
		ID:       lowerer.nextID("event"),
		Symbol:   symbol,
		Name:     name,
		Language: LanguagePython,
		Range:    lowerer.sourceRange(node),
	}, true
}

func (lowerer *pyLowerer) eventBySymbol(symbol string) *EventDecl {
	for index := range lowerer.program.Events {
		if lowerer.program.Events[index].Symbol == symbol {
			return &lowerer.program.Events[index]
		}
	}
	return nil
}

func (lowerer *pyLowerer) eventNameFromCall(text string) (string, bool) {
	start := lowerer.eventCallIndex(text)
	if start < 0 {
		return "", false
	}
	args := text[start:]
	if paren := strings.Index(args, "("); paren >= 0 {
		args = args[paren+1:]
	}
	if end := strings.LastIndex(args, ")"); end >= 0 {
		args = args[:end]
	}
	if named := strings.Index(args, "name="); named >= 0 {
		args = args[named+len("name="):]
	}
	args = strings.TrimSpace(args)
	if comma := strings.Index(args, ","); comma >= 0 {
		args = args[:comma]
	}
	return pythonQuotedStringValue(strings.TrimSpace(args))
}

func (lowerer *pyLowerer) expressionContainsEvent(value string) bool {
	return lowerer.eventCallIndex(value) >= 0
}

func (lowerer *pyLowerer) eventCallIndex(value string) int {
	index := -1
	for _, token := range []string{"Event(", "event("} {
		if found := strings.Index(value, token); found >= 0 && (index < 0 || found < index) {
			index = found
		}
	}
	for name, canonical := range lowerer.hsmNamedBindings {
		if canonical == "Event" {
			if found := strings.Index(value, name+"("); found >= 0 && (index < 0 || found < index) {
				index = found
			}
		}
	}
	return index
}

func (lowerer *pyLowerer) parseExpressionTrigger(ownerID string, kind TriggerKind, value string, isString bool, call *sitter.Node) *Trigger {
	trigger := &Trigger{Kind: kind, Value: value, IsString: isString, Language: LanguagePython}
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

func (lowerer *pyLowerer) parseBehaviors(ownerID string, kind BehaviorKind, call *sitter.Node) []BehaviorRef {
	var refs []BehaviorRef
	for _, arg := range callArgs(call) {
		if operation, ok := pythonStringLiteral(arg, lowerer.input.Data); ok {
			refs = append(refs, BehaviorRef{Kind: kind, Operation: operation})
			continue
		}
		behavior := lowerer.parseBehavior(ownerID, kind, arg)
		lowerer.program.Behaviors = append(lowerer.program.Behaviors, behavior)
		refs = append(refs, BehaviorRef{ID: behavior.ID, Kind: kind})
	}
	return refs
}

func (lowerer *pyLowerer) parseBehavior(ownerID string, kind BehaviorKind, node *sitter.Node) Behavior {
	inline := node.Kind() == "lambda"
	if resolved := lowerer.referencedBehaviorFunction(node, kind); resolved != nil {
		node = resolved
	}
	code := strings.TrimSpace(node.Utf8Text(lowerer.input.Data))
	behavior := Behavior{
		ID:             lowerer.nextID(string(kind)),
		OwnerID:        ownerID,
		Kind:           kind,
		Inline:         inline,
		SourceLanguage: LanguagePython,
		Code:           code,
		Range:          lowerer.sourceRange(node),
	}
	if node.Kind() == "lambda" {
		behavior.Body = code
	} else if node.Kind() == "function_definition" {
		if body := node.ChildByFieldName("body"); body != nil {
			behavior.Signature = strings.TrimSpace(string(lowerer.input.Data[node.StartByte():body.StartByte()]))
			behavior.Body = pythonBlockBody(body.Utf8Text(lowerer.input.Data))
		}
	}
	return behavior
}

type pyFunctionDecl struct {
	node        *sitter.Node
	globalRange SourceRange
}

func (lowerer *pyLowerer) registerFunctionDecl(node *sitter.Node, globalRange SourceRange) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		nameNode = firstNamedChildOfKind(node, "identifier")
	}
	if nameNode == nil {
		return
	}
	if lowerer.functions == nil {
		lowerer.functions = map[string]pyFunctionDecl{}
	}
	lowerer.functions[nameNode.Utf8Text(lowerer.input.Data)] = pyFunctionDecl{node: node, globalRange: globalRange}
}

func (lowerer *pyLowerer) referencedBehaviorFunction(node *sitter.Node, kind BehaviorKind) *sitter.Node {
	if node == nil || node.Kind() != "identifier" {
		return nil
	}
	name := node.Utf8Text(lowerer.input.Data)
	if isGeneratedPythonBehaviorFunctionNameForAnyKind(name) && !isGeneratedPythonBehaviorFunctionName(name, kind) {
		return nil
	}
	decl, ok := lowerer.functions[name]
	if !ok {
		return nil
	}
	if !isGeneratedPythonBehaviorFunctionName(name, kind) && !lowerer.functionDeclLooksLikeBehavior(decl.node) {
		return nil
	}
	if lowerer.usedFunctions == nil {
		lowerer.usedFunctions = map[string]bool{}
	}
	lowerer.usedFunctions[name] = true
	return decl.node
}

func (lowerer *pyLowerer) functionDeclLooksLikeBehavior(node *sitter.Node) bool {
	if node == nil || node.Kind() != "function_definition" {
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

func (lowerer *pyLowerer) filterReferencedFunctionGlobals() {
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

func isGeneratedPythonBehaviorFunctionName(name string, kind BehaviorKind) bool {
	prefix := string(kind)
	if kind == BehaviorOperation {
		prefix = "operation"
	}
	prefix += "_"
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

func isGeneratedPythonBehaviorFunctionNameForAnyKind(name string) bool {
	for _, kind := range []BehaviorKind{
		BehaviorEntry,
		BehaviorExit,
		BehaviorActivity,
		BehaviorGuard,
		BehaviorEffect,
		BehaviorTrigger,
		BehaviorOperation,
	} {
		if isGeneratedPythonBehaviorFunctionName(name, kind) {
			return true
		}
	}
	return false
}

func pythonBlockBody(text string) string {
	lines := strings.Split(strings.TrimRight(text, "\r\n"), "\n")
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent <= 0 {
		return strings.TrimSpace(text)
	}
	for index, line := range lines {
		if len(line) >= minIndent {
			lines[index] = line[minIndent:]
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func (lowerer *pyLowerer) parseImports(node *sitter.Node) []Import {
	text := strings.TrimSpace(node.Utf8Text(lowerer.input.Data))
	var imports []Import
	if strings.HasPrefix(text, "import ") {
		imports = lowerer.parseImportStatement(node)
	} else if strings.HasPrefix(text, "from ") {
		imports = lowerer.parseFromImport(text, node)
	}
	return imports
}

func filterCompilerOwnedImports(imports []Import) []Import {
	var filtered []Import
	for _, imp := range imports {
		if isHSMImportPath(imp.Path) {
			continue
		}
		filtered = append(filtered, imp)
	}
	return filtered
}

func (lowerer *pyLowerer) parseImportStatement(node *sitter.Node) []Import {
	var imports []Import
	for _, item := range strings.Split(strings.TrimPrefix(strings.TrimSpace(node.Utf8Text(lowerer.input.Data)), "import "), ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		parts := strings.Fields(item)
		imp := Import{Path: parts[0], Language: LanguagePython, Range: lowerer.sourceRange(node)}
		if len(parts) == 3 && parts[1] == "as" {
			imp.Alias = parts[2]
		}
		imp.LocalNames = pythonImportLocalNames(imp.Path, imp.Alias)
		imports = append(imports, imp)
	}
	return imports
}

func (lowerer *pyLowerer) parseFromImport(text string, node *sitter.Node) []Import {
	before, after, ok := strings.Cut(strings.TrimPrefix(text, "from "), " import ")
	if !ok {
		return nil
	}
	pathValue := strings.TrimSpace(before)
	after = strings.TrimSpace(after)
	after = strings.TrimPrefix(after, "(")
	after = strings.TrimSuffix(after, ")")
	var localNames []string
	var specifiers []ImportSpecifier
	for _, item := range strings.Split(after, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		parts := strings.Fields(item)
		if len(parts) == 3 && parts[1] == "as" {
			localNames = append(localNames, parts[2])
			specifiers = append(specifiers, ImportSpecifier{Name: parts[0], Alias: parts[2]})
		} else if len(parts) > 0 {
			localNames = append(localNames, parts[0])
			specifiers = append(specifiers, ImportSpecifier{Name: parts[0]})
		}
	}
	return []Import{{Path: pathValue, Specifiers: specifiers, LocalNames: localNames, Language: LanguagePython, Range: lowerer.sourceRange(node)}}
}

func (lowerer *pyLowerer) recordHSMImportBindings(imp Import) {
	if lowerer.hsmNamedBindings == nil {
		lowerer.hsmNamedBindings = map[string]string{}
	}
	if imp.Alias != "" {
		lowerer.hsmNamedBindings[imp.Alias] = ""
	}
	for _, specifier := range imp.Specifiers {
		name := specifier.Name
		if specifier.Alias != "" {
			name = specifier.Alias
		}
		canonical := pythonCanonicalName(specifier.Name)
		if pythonKnownDSLName(canonical) {
			lowerer.hsmNamedBindings[name] = canonical
		} else {
			lowerer.hsmNamedBindings[name] = ""
		}
	}
}

func pythonImportLocalNames(importPath string, alias string) []string {
	if alias != "" {
		return []string{alias}
	}
	name := importPath
	if index := strings.Index(name, "."); index >= 0 {
		name = name[:index]
	}
	return []string{name}
}

func (lowerer *pyLowerer) globalBlock(node *sitter.Node) CodeBlock {
	return CodeBlock{
		ID:       lowerer.nextID("global"),
		Language: LanguagePython,
		Code:     node.Utf8Text(lowerer.input.Data),
		Range:    lowerer.sourceRange(node),
	}
}

func (lowerer *pyLowerer) globalCodeBlock(code string, node *sitter.Node) CodeBlock {
	return CodeBlock{
		ID:       lowerer.nextID("global"),
		Language: LanguagePython,
		Code:     code,
		Range:    lowerer.sourceRange(node),
	}
}

func (lowerer *pyLowerer) firstStringArg(call *sitter.Node) string {
	args := callArgs(call)
	if len(args) == 0 {
		return ""
	}
	if value, ok := pythonStringLiteral(args[0], lowerer.input.Data); ok {
		return value
	}
	return strings.TrimSpace(args[0].Utf8Text(lowerer.input.Data))
}

type pythonAssignmentPart struct {
	name  string
	value string
}

func splitPythonGroupedAssignment(text string) ([]pythonAssignmentPart, bool) {
	index := topLevelPythonRuneIndex(text, '=')
	if index < 0 {
		return nil, false
	}
	left := strings.TrimSpace(text[:index])
	right := strings.TrimSpace(text[index+1:])
	names := splitTopLevelPythonList(left)
	values := splitTopLevelPythonList(right)
	if len(names) <= 1 || len(names) != len(values) {
		return nil, false
	}
	parts := make([]pythonAssignmentPart, 0, len(names))
	for index := range names {
		name := strings.TrimSpace(names[index])
		value := strings.TrimSpace(values[index])
		if name == "" || value == "" || strings.ContainsAny(name, " \t()[]{}") {
			return nil, false
		}
		parts = append(parts, pythonAssignmentPart{name: name, value: value})
	}
	return parts, true
}

func topLevelPythonRuneIndex(text string, target rune) int {
	depth := 0
	quote := rune(0)
	escaped := false
	for index, char := range text {
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == quote {
				quote = 0
			}
			continue
		}
		switch char {
		case '\'', '"':
			quote = char
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 && char == target {
				return index
			}
		}
	}
	return -1
}

func splitTopLevelPythonList(text string) []string {
	var parts []string
	start := 0
	depth := 0
	quote := rune(0)
	escaped := false
	for index, char := range text {
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == quote {
				quote = 0
			}
			continue
		}
		switch char {
		case '\'', '"':
			quote = char
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(text[start:index]))
				start = index + len(string(char))
			}
		}
	}
	parts = append(parts, strings.TrimSpace(text[start:]))
	return parts
}

func (lowerer *pyLowerer) firstArgValue(call *sitter.Node) (string, bool) {
	args := callArgs(call)
	if len(args) == 0 {
		return "", false
	}
	return lowerer.nodeArgValue(args[0])
}

func (lowerer *pyLowerer) nodeArgValue(node *sitter.Node) (string, bool) {
	if value, ok := pythonStringLiteral(node, lowerer.input.Data); ok {
		return value, true
	}
	return strings.TrimSpace(node.Utf8Text(lowerer.input.Data)), false
}

func (lowerer *pyLowerer) sourceRange(node *sitter.Node) SourceRange {
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

func (lowerer *pyLowerer) nextID(prefix string) string {
	if lowerer.counters == nil {
		lowerer.counters = map[string]int{}
	}
	lowerer.counters[prefix]++
	return fmt.Sprintf("%s_%d", prefix, lowerer.counters[prefix])
}

func (lowerer *pyLowerer) containsHSMDefine(node *sitter.Node) bool {
	found := false
	walk(node, func(child *sitter.Node) {
		if child.Kind() == "call" && lowerer.selectorName(child) == "Define" {
			found = true
		}
	})
	return found
}

func (lowerer *pyLowerer) selectorName(call *sitter.Node) string {
	name := selectorName(call, lowerer.input.Data)
	if canonical, ok := lowerer.hsmNamedBindings[name]; ok && canonical != "" {
		return canonical
	}
	return pythonCanonicalName(name)
}

func (lowerer *pyLowerer) addUnknownHSMPartialDiagnostic(context string, call *sitter.Node, name string) {
	if !lowerer.isHSMCall(call) {
		return
	}
	lowerer.diagnostics = append(lowerer.diagnostics, Diagnostic{
		Message: fmt.Sprintf("unknown or unsupported hsm.%s partial in %s context", name, context),
		Range:   lowerer.sourceRange(call),
	})
}

func (lowerer *pyLowerer) isHSMCall(call *sitter.Node) bool {
	function := call.ChildByFieldName("function")
	if function == nil && call.NamedChildCount() > 0 {
		function = call.NamedChild(0)
	}
	if function == nil {
		return false
	}
	text := strings.TrimSpace(function.Utf8Text(lowerer.input.Data))
	if index := strings.LastIndex(text, "."); index >= 0 {
		qualifier := strings.TrimSpace(text[:index])
		_, ok := lowerer.hsmNamedBindings[qualifier]
		return qualifier == "hsm" || ok
	}
	_, ok := lowerer.hsmNamedBindings[text]
	return ok
}

func (lowerer *pyLowerer) expressionContainsDefine(value string) bool {
	for _, token := range []string{"Define(", "define("} {
		if strings.Contains(value, token) {
			return true
		}
	}
	for name, canonical := range lowerer.hsmNamedBindings {
		if canonical == "Define" && strings.Contains(value, name+"(") {
			return true
		}
	}
	return false
}

func pythonCanonicalName(value string) string {
	switch strings.ToLower(value) {
	case "define":
		return "Define"
	case "event":
		return "Event"
	case "default_clock", "defaultclock":
		return "DefaultClock"
	case "config":
		return "Config"
	case "queue":
		return "Queue"
	case "clock":
		return "Clock"
	case "make_group", "makegroup":
		return "MakeGroup"
	case "make_kind", "makekind":
		return "MakeKind"
	case "is_kind", "iskind":
		return "IsKind"
	case "take_snapshot", "takesnapshot":
		return "TakeSnapshot"
	case "initial":
		return "Initial"
	case "state":
		return "State"
	case "final_state", "finalstate", "final":
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
		return value
	}
}

func pythonKnownDSLName(value string) bool {
	switch value {
	case "Define", "Event", "DefaultClock", "Config", "Queue", "Clock", "MakeGroup", "MakeKind", "IsKind", "TakeSnapshot", "Initial", "State", "Final", "Choice", "ShallowHistory", "DeepHistory", "Transition", "Source", "Target", "On", "OnSet", "OnCall", "After", "Every", "At", "When", "Guard", "Effect", "Entry", "Exit", "Activity", "Defer", "Attribute", "Operation":
		return true
	default:
		return false
	}
}

func pythonStringLiteral(node *sitter.Node, source []byte) (string, bool) {
	if node == nil || node.Kind() != "string" {
		return "", false
	}
	return pythonQuotedStringValue(strings.TrimSpace(node.Utf8Text(source)))
}

func pythonQuotedStringValue(text string) (string, bool) {
	if strings.HasPrefix(text, "r") || strings.HasPrefix(text, "R") {
		text = text[1:]
	}
	if strings.HasPrefix(text, "'") && strings.HasSuffix(text, "'") {
		text = `"` + strings.ReplaceAll(strings.Trim(text, "'"), `"`, `\"`) + `"`
	}
	value, err := strconv.Unquote(text)
	return value, err == nil
}
