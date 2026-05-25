package hsmc

import (
	"bytes"
	"context"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
)

type ZigBackend struct{}

func NewZigBackend() ZigBackend {
	return ZigBackend{}
}

func (ZigBackend) Language() Language { return LanguageZig }

func (backend ZigBackend) Emit(_ context.Context, program *Program) ([]byte, error) {
	var out bytes.Buffer
	backend.emitImports(&out, program)
	if zigNeedsTriggerZeroFallback(program) {
		backend.emitTriggerZeroFallback(&out)
	}
	backend.emitOperationReferenceFallbacks(&out, program)
	for _, event := range program.Events {
		backend.emitEvent(&out, event)
	}
	for _, global := range program.Globals {
		if global.Language != "" && global.Language != LanguageZig {
			writeZigCommentedGlobalSource(&out, "", global, LanguageZig)
			continue
		}
		code := strings.TrimSpace(global.Code)
		if code == "" {
			continue
		}
		out.WriteString(code)
		if !strings.HasSuffix(code, "\n") {
			out.WriteString("\n")
		}
		out.WriteString("\n")
	}
	behaviorNames := map[string]string{}
	for _, behavior := range program.Behaviors {
		name := snakeName(behavior.ID)
		behaviorNames[behavior.ID] = name
		if err := backend.emitBehavior(&out, name, behavior); err != nil {
			return nil, err
		}
	}
	for _, model := range program.Models {
		backend.emitModel(&out, model, behaviorNames)
	}
	return out.Bytes(), nil
}

func (backend ZigBackend) emitTriggerZeroFallback(out *bytes.Buffer) {
	out.WriteString(`fn hsmc_trigger_zero(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) u64 {
    _ = ctx;
    _ = inst;
    _ = event;
    return 0;
}

`)
}

type zigOperationReferenceFallback struct {
	Name      string
	Kind      BehaviorKind
	Operation string
}

func (backend ZigBackend) emitOperationReferenceFallbacks(out *bytes.Buffer, program *Program) {
	for _, fallback := range collectZigOperationReferenceFallbacks(program) {
		returnType := "void"
		if fallback.Kind == BehaviorGuard {
			returnType = "bool"
		}
		fmt.Fprintf(out, "fn %s%s %s {\n", fallback.Name, zigParameterList(), returnType)
		out.WriteString("    _ = ctx;\n")
		out.WriteString("    _ = inst;\n")
		out.WriteString("    _ = event;\n")
		fmt.Fprintf(out, "    // Operation reference %q preserved for manual porting; hsm.zig behavior callbacks cannot call model operations without a state machine handle.\n", fallback.Operation)
		if fallback.Kind == BehaviorGuard {
			out.WriteString("    return false;\n")
		}
		out.WriteString("}\n\n")
	}
}

func (backend ZigBackend) emitImports(out *bytes.Buffer, program *Program) {
	out.WriteString("const std = @import(\"std\");\n")
	out.WriteString("const hsm = @import(\"hsm\");\n")
	for _, imp := range collectRenderableImports(program, LanguageZig, compilerOwnedImports(LanguageZig)...) {
		fmt.Fprintln(out, formatZigImport(imp))
	}
	out.WriteString("\n")
}

func (backend ZigBackend) emitEvent(out *bytes.Buffer, event EventDecl) {
	fmt.Fprintf(out, "pub const %s = %s;\n\n", eventSymbolFor(LanguageZig, event.Symbol), strconv.Quote(event.Name))
}

func (backend ZigBackend) emitBehavior(out *bytes.Buffer, name string, behavior Behavior) error {
	if (behavior.SourceLanguage == LanguageZig || behavior.TargetLanguage == LanguageZig) && strings.TrimSpace(behavior.Body) == "" {
		code := strings.TrimSpace(behavior.Code)
		if code != "" {
			declaredName, ok := zigFunctionDeclarationName(code)
			if !ok {
				return fmt.Errorf("zig behavior code for %q must be a compiler-owned function declaration; use body for function statements", name)
			}
			if declaredName != name {
				return fmt.Errorf("zig behavior full code declares %q, want compiler-owned name %q", declaredName, name)
			}
			if _, parameters, returnType, ok := zigFunctionDeclarationShape(code); ok {
				if wantParameters := zigParameterList(); parameters != wantParameters {
					return fmt.Errorf("zig behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
				}
				if wantReturnType := zigReturnType(behavior); returnType != wantReturnType {
					return fmt.Errorf("zig behavior full code declares return type %q, want compiler-owned return type %q", returnType, wantReturnType)
				}
			}
			out.WriteString(code)
			if !strings.HasSuffix(code, "\n") {
				out.WriteString("\n")
			}
			out.WriteString("\n")
			return nil
		}
	}
	fmt.Fprintf(out, "fn %s%s %s {\n", name, zigParameterList(), zigReturnType(behavior))
	body := ""
	if behavior.TargetLanguage == LanguageZig || behavior.SourceLanguage == LanguageZig {
		body = strings.TrimSpace(behavior.Body)
	}
	if body == "" {
		out.WriteString("    _ = ctx;\n")
		out.WriteString("    _ = inst;\n")
		out.WriteString("    _ = event;\n")
		writeZigCommentedBehaviorSource(out, "    ", behavior)
		if ret := zigUntranslatedReturn(behavior); ret != "" {
			fmt.Fprintf(out, "    %s\n", ret)
		}
	} else {
		out.WriteString(indentZig(body))
		out.WriteString("\n")
		if behavior.Kind == BehaviorGuard && !strings.Contains(body, "return") {
			out.WriteString("    return false;\n")
		}
	}
	out.WriteString("}\n\n")
	return nil
}

func zigReturnType(behavior Behavior) string {
	if behavior.TargetABI != nil && behavior.TargetABI.ReturnType != "" {
		return behavior.TargetABI.ReturnType
	}
	if behavior.Kind == BehaviorGuard {
		return "bool"
	}
	if behavior.Kind == BehaviorTrigger {
		return "u64"
	}
	return "void"
}

func zigParameterList() string {
	return "(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event)"
}

func (backend ZigBackend) emitModel(out *bytes.Buffer, model Model, behaviorNames map[string]string) {
	fmt.Fprintf(out, "pub const %s_model = hsm.define(%s, .{\n", snakeName(model.Name), strconv.Quote(model.Name))
	if model.Initializer != nil {
		backend.emitInitial(out, "    ", *model.Initializer, behaviorNames)
	}
	for _, attribute := range model.Attributes {
		backend.emitAttribute(out, "    ", attribute)
	}
	for _, operation := range model.Operations {
		backend.emitOperation(out, "    ", operation, behaviorNames)
	}
	for _, state := range model.States {
		backend.emitState(out, "    ", state, behaviorNames)
	}
	for _, transition := range model.Transitions {
		backend.emitTransition(out, "    ", transition, behaviorNames)
	}
	out.WriteString("});\n\n")
}

func (backend ZigBackend) emitAttribute(out *bytes.Buffer, indent string, attribute Attribute) {
	if defaultValue, ok := zigAttributeTargetDefault(attribute); ok {
		fmt.Fprintf(out, "%shsm.attribute(%s, %s),\n", indent, strconv.Quote(attribute.Name), defaultValue)
		if attribute.Type != "" && attribute.Language != LanguageZig {
			fmt.Fprintf(out, "%s// Type %s preserved as inferred Zig attribute default.\n", indent, attribute.Type)
		}
		return
	}
	if typeName, ok := zigAttributeTargetType(attribute); ok {
		fmt.Fprintf(out, "%shsm.attribute(%s, %s),\n", indent, strconv.Quote(attribute.Name), typeName)
		return
	}
	if attribute.HasDefault {
		fmt.Fprintf(out, "%s// Default: %s\n", indent, attribute.Default)
	}
	fmt.Fprintf(out, "%s// Attribute %q preserved for manual porting; hsm.zig attributes require a Zig type or default value.\n", indent, attribute.Name)
	if attribute.Type != "" {
		fmt.Fprintf(out, "%s// Type: %s\n", indent, attribute.Type)
	}
}

func zigAttributeTargetDefault(attribute Attribute) (string, bool) {
	if !attribute.HasDefault {
		return "", false
	}
	value := strings.TrimSpace(attribute.Default)
	if value == "" {
		return "", false
	}
	if attribute.Language == "" || attribute.Language == LanguageZig {
		return value, true
	}
	if converted, ok := zigPortableAttributeLiteral(value); ok {
		return converted, true
	}
	return "", false
}

func zigPortableAttributeLiteral(value string) (string, bool) {
	switch value {
	case "True":
		return "true", true
	case "False":
		return "false", true
	}
	if value == "true" || value == "false" {
		return value, true
	}
	if _, err := strconv.ParseInt(value, 10, 64); err == nil {
		return value, true
	}
	if _, err := strconv.Unquote(value); err == nil {
		if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
			return value, true
		}
	}
	return "", false
}

func zigAttributeTargetType(attribute Attribute) (string, bool) {
	value := strings.TrimSpace(attribute.Type)
	if value == "" || (attribute.Language != "" && attribute.Language != LanguageZig) {
		return "", false
	}
	return value, true
}

func (backend ZigBackend) emitOperation(out *bytes.Buffer, indent string, operation Operation, behaviorNames map[string]string) {
	if operation.Behavior == nil {
		return
	}
	name := behaviorNames[operation.Behavior.ID]
	if name == "" {
		name = snakeName(operation.Behavior.ID)
	}
	fmt.Fprintf(out, "%shsm.operation(%s, %s),\n", indent, strconv.Quote(operation.Name), name)
}

func (backend ZigBackend) emitState(out *bytes.Buffer, indent string, state State, behaviorNames map[string]string) {
	switch state.Kind {
	case StateKindFinal:
		fmt.Fprintf(out, "%shsm.final(%s),\n", indent, strconv.Quote(state.Name))
		return
	case StateKindShallowHistory:
		target := zigHistoryTarget(state)
		if target == "." {
			fmt.Fprintf(out, "%s// History state %s had no default target; using self target for syntactic preservation.\n", indent, state.Name)
		}
		backend.emitHistoryDefaultUnsupportedPartials(out, indent, state, behaviorNames)
		fmt.Fprintf(out, "%shsm.history(%s, hsm.target(%s)),\n", indent, strconv.Quote(state.Name), strconv.Quote(target))
		return
	case StateKindDeepHistory:
		target := zigHistoryTarget(state)
		if target == "." {
			fmt.Fprintf(out, "%s// Deep history state %s had no default target; using self target for syntactic preservation.\n", indent, state.Name)
		}
		backend.emitHistoryDefaultUnsupportedPartials(out, indent, state, behaviorNames)
		fmt.Fprintf(out, "%shsm.deepHistory(%s, hsm.target(%s)),\n", indent, strconv.Quote(state.Name), strconv.Quote(target))
		return
	}
	constructor := "state"
	if state.Kind == StateKindChoice {
		constructor = "choice"
	}
	fmt.Fprintf(out, "%shsm.%s(%s, .{\n", indent, constructor, strconv.Quote(state.Name))
	if state.Initializer != nil {
		backend.emitInitial(out, indent+"    ", *state.Initializer, behaviorNames)
	}
	for _, behavior := range state.Behaviors {
		backend.emitBehaviorPartial(out, indent+"    ", behavior, behaviorNames)
	}
	if len(state.Defers) > 0 {
		backend.emitDefers(out, indent+"    ", state.Defers)
	}
	for _, child := range state.States {
		backend.emitState(out, indent+"    ", child, behaviorNames)
	}
	for _, transition := range state.Transitions {
		backend.emitTransition(out, indent+"    ", transition, behaviorNames)
	}
	fmt.Fprintf(out, "%s}),\n", indent)
}

func (backend ZigBackend) emitHistoryDefaultUnsupportedPartials(out *bytes.Buffer, indent string, state State, behaviorNames map[string]string) {
	for _, transition := range state.Transitions {
		if transition.Guard == nil && len(transition.Effects) == 0 {
			continue
		}
		fmt.Fprintf(out, "%s// History default transition guard/effects preserved for manual porting; hsm.zig history helpers accept only a target.\n", indent)
		if transition.Guard != nil {
			backend.emitBehaviorPartial(out, indent+"// ", *transition.Guard, behaviorNames)
		}
		for _, effect := range transition.Effects {
			backend.emitBehaviorPartial(out, indent+"// ", effect, behaviorNames)
		}
		return
	}
}

func zigHistoryTarget(state State) string {
	if state.Initializer != nil && state.Initializer.Target != "" {
		return state.Initializer.Target
	}
	for _, transition := range state.Transitions {
		if transition.Target != "" {
			return transition.Target
		}
	}
	return "."
}

func (backend ZigBackend) emitInitial(out *bytes.Buffer, indent string, initial Initial, behaviorNames map[string]string) {
	if len(initial.Effects) > 0 {
		fmt.Fprintf(out, "%s// Initial transition effects preserved for manual porting; hsm.zig initial() accepts only a target.\n", indent)
		for _, effect := range initial.Effects {
			backend.emitBehaviorPartial(out, indent+"// ", effect, behaviorNames)
		}
	}
	fmt.Fprintf(out, "%shsm.initial(hsm.target(%s)),\n", indent, strconv.Quote(initial.Target))
}

func (backend ZigBackend) emitTransition(out *bytes.Buffer, indent string, transition Transition, behaviorNames map[string]string) {
	fmt.Fprintf(out, "%shsm.transition(.{\n", indent)
	if transition.Source != "" {
		fmt.Fprintf(out, "%s    hsm.source(%s),\n", indent, strconv.Quote(transition.Source))
	}
	if transition.Trigger != nil {
		backend.emitTrigger(out, indent+"    ", *transition.Trigger, behaviorNames)
	}
	if transition.Guard != nil {
		backend.emitBehaviorPartial(out, indent+"    ", *transition.Guard, behaviorNames)
	}
	if transition.Target != "" {
		fmt.Fprintf(out, "%s    hsm.target(%s),\n", indent, strconv.Quote(transition.Target))
	}
	for _, effect := range transition.Effects {
		backend.emitBehaviorPartial(out, indent+"    ", effect, behaviorNames)
	}
	fmt.Fprintf(out, "%s}),\n", indent)
}

func (backend ZigBackend) emitTrigger(out *bytes.Buffer, indent string, trigger Trigger, behaviorNames map[string]string) {
	switch trigger.Kind {
	case TriggerOn:
		for _, value := range zigTriggerValues(trigger) {
			fmt.Fprintf(out, "%shsm.on(%s),\n", indent, value)
		}
	case TriggerOnSet:
		fmt.Fprintf(out, "%shsm.onSet(%s),\n", indent, strconv.Quote(trigger.Value))
	case TriggerOnCall:
		fmt.Fprintf(out, "%shsm.onCall(%s),\n", indent, strconv.Quote(trigger.Value))
	case TriggerAfter:
		writeZigForeignTriggerExpressionComment(out, indent, trigger)
		fmt.Fprintf(out, "%shsm.after(%s),\n", indent, zigExpressionValue(trigger, behaviorNames))
	case TriggerEvery:
		writeZigForeignTriggerExpressionComment(out, indent, trigger)
		fmt.Fprintf(out, "%shsm.every(%s),\n", indent, zigExpressionValue(trigger, behaviorNames))
	case TriggerAt:
		writeZigForeignTriggerExpressionComment(out, indent, trigger)
		fmt.Fprintf(out, "%shsm.at(%s),\n", indent, zigExpressionValue(trigger, behaviorNames))
	case TriggerWhen:
		if trigger.IsString && trigger.Value != "" && trigger.Expr == nil {
			fmt.Fprintf(out, "%shsm.onSet(%s),\n", indent, strconv.Quote(trigger.Value))
			return
		}
		if trigger.Expr != nil {
			if value := behaviorNames[trigger.Expr.ID]; value != "" {
				fmt.Fprintf(out, "%s// when trigger lowered to guard for hsm.zig\n", indent)
				fmt.Fprintf(out, "%shsm.guard(%s),\n", indent, value)
				return
			}
		}
		writeZigUnsupportedTrigger(out, indent, trigger)
	default:
		writeZigUnsupportedTrigger(out, indent, trigger)
	}
}

func writeZigUnsupportedTrigger(out *bytes.Buffer, indent string, trigger Trigger) {
	fmt.Fprintf(out, "%s// Unsupported %s trigger preserved for manual porting:\n", indent, trigger.Kind)
	source := strings.TrimSpace(trigger.Value)
	if source == "" {
		fmt.Fprintf(out, "%s// No source trigger expression was captured.\n", indent)
		return
	}
	for _, line := range strings.Split(source, "\n") {
		line = strings.ReplaceAll(line, "\t", "    ")
		if strings.TrimSpace(line) == "" {
			fmt.Fprintf(out, "%s//\n", indent)
			continue
		}
		fmt.Fprintf(out, "%s// %s\n", indent, line)
	}
}

func writeZigForeignTriggerExpressionComment(out *bytes.Buffer, indent string, trigger Trigger) {
	if trigger.Expr != nil || trigger.IsString || trigger.Language == "" || trigger.Language == LanguageZig {
		return
	}
	fmt.Fprintf(out, "%s// Original %s %s trigger expression preserved for manual porting:\n", indent, trigger.Language, trigger.Kind)
	source := strings.TrimSpace(trigger.Value)
	if source == "" {
		fmt.Fprintf(out, "%s// No source trigger expression was captured.\n", indent)
		return
	}
	for _, line := range strings.Split(source, "\n") {
		line = strings.ReplaceAll(line, "\t", "    ")
		if strings.TrimSpace(line) == "" {
			fmt.Fprintf(out, "%s//\n", indent)
			continue
		}
		fmt.Fprintf(out, "%s// %s\n", indent, line)
	}
}

func zigTriggerValues(trigger Trigger) []string {
	values := trigger.Values
	if len(values) == 0 {
		values = []TriggerValue{{Value: trigger.Value, IsString: trigger.IsString, Language: trigger.Language}}
	}
	rendered := make([]string, 0, len(values))
	for _, value := range values {
		text := value.Value
		if value.EventSymbol != "" {
			text = eventSymbolFor(LanguageZig, value.EventSymbol)
		} else if value.IsString {
			text = strconv.Quote(text)
		}
		rendered = append(rendered, text)
	}
	return rendered
}

func (backend ZigBackend) emitBehaviorPartial(out *bytes.Buffer, indent string, ref BehaviorRef, behaviorNames map[string]string) {
	constructor := "entry"
	switch ref.Kind {
	case BehaviorExit:
		constructor = "exit"
	case BehaviorActivity:
		constructor = "activity"
	case BehaviorGuard:
		constructor = "guard"
	case BehaviorEffect:
		constructor = "effect"
	}
	name := behaviorNames[ref.ID]
	if ref.Operation != "" {
		name = zigOperationReferenceFallbackName(ref.Kind, ref.Operation)
	}
	if name == "" {
		name = snakeName(ref.ID)
	}
	fmt.Fprintf(out, "%shsm.%s(%s),\n", indent, constructor, name)
}

func collectZigOperationReferenceFallbacks(program *Program) []zigOperationReferenceFallback {
	if program == nil {
		return nil
	}
	seen := map[string]bool{}
	var fallbacks []zigOperationReferenceFallback
	add := func(ref BehaviorRef) {
		if ref.Operation == "" {
			return
		}
		switch ref.Kind {
		case BehaviorEntry, BehaviorExit, BehaviorActivity, BehaviorGuard, BehaviorEffect:
		default:
			return
		}
		name := zigOperationReferenceFallbackName(ref.Kind, ref.Operation)
		if seen[name] {
			return
		}
		seen[name] = true
		fallbacks = append(fallbacks, zigOperationReferenceFallback{Name: name, Kind: ref.Kind, Operation: ref.Operation})
	}
	var visitState func(State)
	var visitTransition func(Transition)
	visitTransition = func(transition Transition) {
		if transition.Guard != nil {
			add(*transition.Guard)
		}
		for _, effect := range transition.Effects {
			add(effect)
		}
	}
	visitState = func(state State) {
		for _, behavior := range state.Behaviors {
			add(behavior)
		}
		if state.Initializer != nil {
			for _, effect := range state.Initializer.Effects {
				add(effect)
			}
		}
		for _, transition := range state.Transitions {
			visitTransition(transition)
		}
		for _, child := range state.States {
			visitState(child)
		}
	}
	for _, model := range program.Models {
		if model.Initializer != nil {
			for _, effect := range model.Initializer.Effects {
				add(effect)
			}
		}
		for _, transition := range model.Transitions {
			visitTransition(transition)
		}
		for _, state := range model.States {
			visitState(state)
		}
	}
	return fallbacks
}

func zigOperationReferenceFallbackName(kind BehaviorKind, operation string) string {
	role := strings.TrimPrefix(string(kind), "behavior_")
	if role == "" {
		role = "behavior"
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(string(kind)))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(operation))
	return fmt.Sprintf("hsmc_%s_operation_%s_%08x", snakeName(role), snakeName(operation), hash.Sum32())
}

func (backend ZigBackend) emitDefers(out *bytes.Buffer, indent string, events []string) {
	out.WriteString(indent + "hsm.deferEvents(.{")
	for index, eventName := range events {
		if index > 0 {
			out.WriteString(", ")
		}
		out.WriteString(strconv.Quote(eventName))
	}
	out.WriteString("}),\n")
}

func zigExpressionValue(trigger Trigger, behaviorNames map[string]string) string {
	if trigger.Expr != nil {
		if name := behaviorNames[trigger.Expr.ID]; name != "" {
			return name
		}
	}
	if trigger.IsString {
		return strconv.Quote(trigger.Value)
	}
	if trigger.Language == "" || trigger.Language == LanguageZig {
		return trigger.Value
	}
	return zigDefaultTriggerFunctionName()
}

func zigDefaultTriggerFunctionName() string {
	return "hsmc_trigger_zero"
}

func zigNeedsTriggerZeroFallback(program *Program) bool {
	if program == nil {
		return false
	}
	for _, model := range program.Models {
		if zigModelNeedsTriggerZeroFallback(model) {
			return true
		}
	}
	return false
}

func zigModelNeedsTriggerZeroFallback(model Model) bool {
	for _, transition := range model.Transitions {
		if zigTransitionNeedsTriggerZeroFallback(transition) {
			return true
		}
	}
	for _, state := range model.States {
		if zigStateNeedsTriggerZeroFallback(state) {
			return true
		}
	}
	return false
}

func zigStateNeedsTriggerZeroFallback(state State) bool {
	for _, transition := range state.Transitions {
		if zigTransitionNeedsTriggerZeroFallback(transition) {
			return true
		}
	}
	for _, child := range state.States {
		if zigStateNeedsTriggerZeroFallback(child) {
			return true
		}
	}
	return false
}

func zigTransitionNeedsTriggerZeroFallback(transition Transition) bool {
	if transition.Trigger == nil {
		return false
	}
	trigger := *transition.Trigger
	if trigger.Kind != TriggerAfter && trigger.Kind != TriggerEvery && trigger.Kind != TriggerAt {
		return false
	}
	return trigger.Expr == nil && !trigger.IsString && trigger.Language != "" && trigger.Language != LanguageZig
}

func indentZig(body string) string {
	lines := strings.Split(body, "\n")
	for index, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[index] = ""
			continue
		}
		lines[index] = "    " + strings.ReplaceAll(line, "\t", "    ")
	}
	return strings.Join(lines, "\n")
}

func writeZigCommentedBehaviorSource(out *bytes.Buffer, indent string, behavior Behavior) {
	fmt.Fprintf(out, "%s// Original %s behavior %s preserved for manual porting:\n", indent, behavior.SourceLanguage, behavior.ID)
	if target := behaviorCommentTargetLanguage(behavior); target != "" && target != behavior.SourceLanguage {
		fmt.Fprintf(out, "%s// Target language: %s\n", indent, target)
	}
	if behavior.Kind != "" {
		fmt.Fprintf(out, "%s// Behavior kind: %s\n", indent, behavior.Kind)
	}
	if len(behavior.Imports) > 0 {
		fmt.Fprintf(out, "%s// Source imports:\n", indent)
		for _, imp := range behavior.Imports {
			if line := sourceImportText(importCommentLanguage(behavior.SourceLanguage, imp), imp); line != "" {
				line = strings.ReplaceAll(line, "\t", "    ")
				fmt.Fprintf(out, "%s// %s\n", indent, line)
			}
		}
	}
	source := behaviorSourceText(behavior)
	if source == "" {
		fmt.Fprintf(out, "%s// No source behavior body was captured.\n", indent)
		return
	}
	for _, line := range strings.Split(source, "\n") {
		line = strings.ReplaceAll(line, "\t", "    ")
		if strings.TrimSpace(line) == "" {
			fmt.Fprintf(out, "%s//\n", indent)
			continue
		}
		fmt.Fprintf(out, "%s// %s\n", indent, line)
	}
}

func writeZigCommentedGlobalSource(out *bytes.Buffer, indent string, global CodeBlock, target Language) {
	source := strings.TrimSpace(global.Code)
	if source == "" {
		return
	}
	fmt.Fprintf(out, "%s// Original %s global %s preserved for manual porting:\n", indent, global.Language, global.ID)
	if target != "" && target != global.Language {
		fmt.Fprintf(out, "%s// Target language: %s\n", indent, target)
	}
	if len(global.Imports) > 0 {
		fmt.Fprintf(out, "%s// Source imports:\n", indent)
		for _, imp := range global.Imports {
			if line := sourceImportText(importCommentLanguage(global.Language, imp), imp); line != "" {
				line = strings.ReplaceAll(line, "\t", "    ")
				fmt.Fprintf(out, "%s// %s\n", indent, line)
			}
		}
	}
	for _, line := range strings.Split(source, "\n") {
		line = strings.ReplaceAll(line, "\t", "    ")
		if strings.TrimSpace(line) == "" {
			fmt.Fprintf(out, "%s//\n", indent)
			continue
		}
		fmt.Fprintf(out, "%s// %s\n", indent, line)
	}
	out.WriteString("\n")
}
