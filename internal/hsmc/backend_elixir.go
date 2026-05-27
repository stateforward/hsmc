package hsmc

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
)

type ElixirBackend struct{}

func NewElixirBackend() ElixirBackend {
	return ElixirBackend{}
}

func (ElixirBackend) Language() Language { return LanguageElixir }

func (backend ElixirBackend) Emit(_ context.Context, program *Program) ([]byte, error) {
	var out bytes.Buffer
	module := "GeneratedHSM"
	if len(program.Models) > 0 {
		module = exportName(program.Models[0].Name) + "HSM"
	}
	fmt.Fprintf(&out, "defmodule %s do\n", module)
	backend.emitImports(&out, program)
	for _, event := range program.Events {
		backend.emitEvent(&out, event)
	}
	for _, global := range program.Globals {
		backend.emitGlobal(&out, global)
	}
	behaviorNames := map[string]string{}
	for _, behavior := range program.Behaviors {
		name := snakeName(behavior.ID)
		behaviorNames[behavior.ID] = name
		if err := backend.emitBehavior(&out, name, behavior); err != nil {
			return nil, err
		}
	}
	triggerFallbacks := collectElixirTriggerFallbacks(program)
	for _, fallback := range triggerFallbacks {
		backend.emitTriggerFallback(&out, fallback)
	}
	triggerFallbackNames := elixirTriggerFallbackNameMap(triggerFallbacks)
	for _, model := range program.Models {
		backend.emitModel(&out, model, behaviorNames, triggerFallbackNames)
	}
	out.WriteString("end\n")
	return out.Bytes(), nil
}

func (backend ElixirBackend) emitImports(out *bytes.Buffer, program *Program) {
	for _, imp := range collectRenderableImports(program, LanguageElixir, compilerOwnedImports(LanguageElixir)...) {
		fmt.Fprintf(out, "  %s\n", formatElixirImport(imp))
	}
	if len(collectRenderableImports(program, LanguageElixir, compilerOwnedImports(LanguageElixir)...)) > 0 {
		out.WriteString("\n")
	}
}

func formatElixirImport(imp Import) string {
	if imp.Alias != "" {
		return fmt.Sprintf("alias %s, as: %s", imp.Path, imp.Alias)
	}
	return fmt.Sprintf("alias %s", imp.Path)
}

func (backend ElixirBackend) emitEvent(out *bytes.Buffer, event EventDecl) {
	fmt.Fprintf(out, "  def %s do\n", eventSymbolFor(LanguageElixir, event.Symbol))
	fmt.Fprintf(out, "    HSM.event(%s)\n", quoteElixir(event.Name))
	out.WriteString("  end\n\n")
}

func (backend ElixirBackend) emitGlobal(out *bytes.Buffer, global CodeBlock) {
	code := strings.TrimSpace(global.Code)
	if code == "" {
		return
	}
	if global.Language != "" && global.Language != LanguageElixir {
		writeCommentedGlobalSource(out, "  ", "#", global, LanguageElixir)
		out.WriteString("\n")
		return
	}
	out.WriteString(indentElixir(code))
	out.WriteString("\n\n")
}

func (backend ElixirBackend) emitBehavior(out *bytes.Buffer, name string, behavior Behavior) error {
	body := ""
	if behavior.TargetLanguage == LanguageElixir || behavior.SourceLanguage == LanguageElixir {
		body = strings.TrimSpace(behavior.Body)
	}
	fmt.Fprintf(out, "  def %s(_ctx, _instance, _event) do\n", name)
	if body == "" {
		writeCommentedBehaviorSource(out, "    ", "#", behavior)
		fmt.Fprintf(out, "    %s\n", elixirUntranslatedReturn(behavior))
	} else {
		out.WriteString(indentElixirBody(body))
		out.WriteString("\n")
	}
	out.WriteString("  end\n\n")
	return nil
}

func (backend ElixirBackend) emitTriggerFallback(out *bytes.Buffer, fallback elixirTriggerFallback) {
	fmt.Fprintf(out, "  def %s(_ctx, _instance, _event) do\n", fallback.Name)
	fmt.Fprintf(out, "    # Original %s %s trigger expression preserved for manual porting:\n", fallback.Language, fallback.Kind)
	source := strings.TrimSpace(fallback.Value)
	if source == "" {
		out.WriteString("    # No source trigger expression was captured.\n")
	} else {
		for _, line := range strings.Split(source, "\n") {
			if strings.TrimSpace(line) == "" {
				out.WriteString("    #\n")
				continue
			}
			fmt.Fprintf(out, "    # %s\n", strings.ReplaceAll(line, "\t", "    "))
		}
	}
	fmt.Fprintf(out, "    %s\n", elixirTriggerFallbackReturn(fallback.Kind))
	out.WriteString("  end\n\n")
}

func (backend ElixirBackend) emitModel(out *bytes.Buffer, model Model, behaviorNames map[string]string, triggerFallbackNames map[string]string) {
	fmt.Fprintf(out, "  def %s_model do\n", snakeName(model.Name))
	fmt.Fprintf(out, "    HSM.define(%s, [\n", quoteElixir(model.Name))
	if model.Initializer != nil {
		backend.emitInitial(out, "      ", *model.Initializer, behaviorNames)
	}
	for _, attribute := range model.Attributes {
		backend.emitAttribute(out, "      ", attribute)
	}
	for _, operation := range model.Operations {
		backend.emitOperation(out, "      ", operation, behaviorNames)
	}
	for _, state := range model.States {
		backend.emitState(out, "      ", state, behaviorNames, triggerFallbackNames)
	}
	for _, transition := range model.Transitions {
		backend.emitTransition(out, "      ", transition, behaviorNames, triggerFallbackNames)
	}
	out.WriteString("    ])\n")
	out.WriteString("  end\n\n")
}

func (backend ElixirBackend) emitAttribute(out *bytes.Buffer, indent string, attribute Attribute) {
	typeValue, hasType := elixirAttributeType(attribute.Type)
	defaultValue, hasDefault := elixirAttributeDefault(attribute)
	switch {
	case hasType && hasDefault:
		fmt.Fprintf(out, "%sHSM.attribute(%s, %s, %s),\n", indent, quoteElixir(attribute.Name), typeValue, defaultValue)
	case hasDefault:
		fmt.Fprintf(out, "%sHSM.attribute(%s, %s),\n", indent, quoteElixir(attribute.Name), defaultValue)
	default:
		fmt.Fprintf(out, "%sHSM.attribute(%s),\n", indent, quoteElixir(attribute.Name))
	}
}

func (backend ElixirBackend) emitOperation(out *bytes.Buffer, indent string, operation Operation, behaviorNames map[string]string) {
	if operation.Behavior == nil {
		return
	}
	name := behaviorNames[operation.Behavior.ID]
	if name == "" {
		name = snakeName(operation.Behavior.ID)
	}
	fmt.Fprintf(out, "%sHSM.operation(%s, &__MODULE__.%s/3),\n", indent, quoteElixir(operation.Name), name)
}

func (backend ElixirBackend) emitState(out *bytes.Buffer, indent string, state State, behaviorNames map[string]string, triggerFallbackNames map[string]string) {
	constructor := "state"
	switch state.Kind {
	case StateKindFinal:
		fmt.Fprintf(out, "%sHSM.final(%s),\n", indent, quoteElixir(state.Name))
		return
	case StateKindChoice:
		constructor = "choice"
	case StateKindShallowHistory:
		constructor = "shallow_history"
	case StateKindDeepHistory:
		constructor = "deep_history"
	}
	fmt.Fprintf(out, "%sHSM.%s(%s, [\n", indent, constructor, quoteElixir(state.Name))
	if state.Initializer != nil {
		backend.emitInitial(out, indent+"  ", *state.Initializer, behaviorNames)
	}
	for _, behavior := range state.Behaviors {
		backend.emitBehaviorPartial(out, indent+"  ", behavior, behaviorNames)
	}
	for _, eventName := range state.Defers {
		fmt.Fprintf(out, "%s  HSM.defer(%s),\n", indent, quoteElixir(eventName))
	}
	for _, child := range state.States {
		backend.emitState(out, indent+"  ", child, behaviorNames, triggerFallbackNames)
	}
	for _, transition := range state.Transitions {
		backend.emitTransition(out, indent+"  ", transition, behaviorNames, triggerFallbackNames)
	}
	fmt.Fprintf(out, "%s]),\n", indent)
}

func (backend ElixirBackend) emitInitial(out *bytes.Buffer, indent string, initial Initial, behaviorNames map[string]string) {
	fmt.Fprintf(out, "%sHSM.initial([\n", indent)
	if initial.Target != "" {
		fmt.Fprintf(out, "%s  HSM.target(%s),\n", indent, quoteElixir(initial.Target))
	}
	for _, effect := range initial.Effects {
		backend.emitBehaviorPartial(out, indent+"  ", effect, behaviorNames)
	}
	fmt.Fprintf(out, "%s]),\n", indent)
}

func (backend ElixirBackend) emitTransition(out *bytes.Buffer, indent string, transition Transition, behaviorNames map[string]string, triggerFallbackNames map[string]string) {
	fmt.Fprintf(out, "%sHSM.transition([\n", indent)
	if transition.Source != "" {
		fmt.Fprintf(out, "%s  HSM.source(%s),\n", indent, quoteElixir(transition.Source))
	}
	if transition.Trigger != nil {
		backend.emitTrigger(out, indent+"  ", *transition.Trigger, behaviorNames, triggerFallbackNames)
	}
	if transition.Guard != nil {
		backend.emitBehaviorPartial(out, indent+"  ", *transition.Guard, behaviorNames)
	}
	if transition.Target != "" {
		fmt.Fprintf(out, "%s  HSM.target(%s),\n", indent, quoteElixir(transition.Target))
	}
	for _, effect := range transition.Effects {
		backend.emitBehaviorPartial(out, indent+"  ", effect, behaviorNames)
	}
	fmt.Fprintf(out, "%s]),\n", indent)
}

func (backend ElixirBackend) emitTrigger(out *bytes.Buffer, indent string, trigger Trigger, behaviorNames map[string]string, triggerFallbackNames map[string]string) {
	switch trigger.Kind {
	case TriggerOn:
		fmt.Fprintf(out, "%sHSM.on(%s),\n", indent, elixirTriggerValues(trigger))
	case TriggerOnSet:
		fmt.Fprintf(out, "%sHSM.on_set(%s),\n", indent, quoteElixir(trigger.Value))
	case TriggerOnCall:
		fmt.Fprintf(out, "%sHSM.on_call(%s),\n", indent, quoteElixir(trigger.Value))
	case TriggerAfter:
		fmt.Fprintf(out, "%sHSM.after_ms(%s),\n", indent, elixirExpressionValue(trigger, behaviorNames, triggerFallbackNames))
	case TriggerEvery:
		fmt.Fprintf(out, "%sHSM.every_ms(%s),\n", indent, elixirExpressionValue(trigger, behaviorNames, triggerFallbackNames))
	case TriggerAt:
		fmt.Fprintf(out, "%sHSM.at_ms(%s),\n", indent, elixirExpressionValue(trigger, behaviorNames, triggerFallbackNames))
	case TriggerWhen:
		fmt.Fprintf(out, "%sHSM.when_expr(%s),\n", indent, elixirExpressionValue(trigger, behaviorNames, triggerFallbackNames))
	}
}

func (backend ElixirBackend) emitBehaviorPartial(out *bytes.Buffer, indent string, ref BehaviorRef, behaviorNames map[string]string) {
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
	if name == "" {
		name = snakeName(ref.ID)
	}
	fmt.Fprintf(out, "%sHSM.%s(&__MODULE__.%s/3),\n", indent, constructor, name)
}

func quoteElixir(value string) string {
	return strconv.Quote(value)
}

func indentElixir(body string) string {
	lines := strings.Split(body, "\n")
	for index, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[index] = ""
			continue
		}
		lines[index] = "  " + line
	}
	return strings.Join(lines, "\n")
}

func indentElixirBody(body string) string {
	lines := strings.Split(body, "\n")
	for index, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[index] = ""
			continue
		}
		lines[index] = "    " + line
	}
	return strings.Join(lines, "\n")
}

func elixirUntranslatedReturn(behavior Behavior) string {
	if behavior.Kind == BehaviorGuard || (behavior.Kind == BehaviorTrigger && behavior.TriggerKind == TriggerWhen) {
		return "false"
	}
	if behavior.Kind == BehaviorTrigger {
		return "0"
	}
	return "nil"
}

func elixirAttributeType(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "int", "integer", "number", "i32", "i64", "u32", "u64":
		return ":integer", true
	case "float", "double", "f32", "f64":
		return ":float", true
	case "bool", "boolean":
		return ":boolean", true
	case "binary":
		return ":binary", true
	case "str", "string":
		return ":string", true
	case "atom":
		return ":atom", true
	case "list", "array":
		return ":list", true
	case "map", "object":
		return ":map", true
	default:
		return "", false
	}
}

func elixirAttributeDefault(attribute Attribute) (string, bool) {
	if !attribute.HasDefault {
		return "", false
	}
	value := strings.TrimSpace(attribute.Default)
	if value == "" {
		return "", false
	}
	if attribute.Language == "" || attribute.Language == LanguageElixir {
		return value, true
	}
	switch value {
	case "true", "True":
		return "true", true
	case "false", "False":
		return "false", true
	case "null", "nil", "None":
		return "nil", true
	}
	if elixirNumericLiteral(value) {
		return value, true
	}
	if _, err := strconv.Unquote(value); err == nil && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		return value, true
	}
	return "", false
}

func elixirNumericLiteral(value string) bool {
	lower := strings.ToLower(value)
	if strings.Contains(lower, "nan") || strings.Contains(lower, "inf") {
		return false
	}
	_, err := strconv.ParseFloat(value, 64)
	return err == nil
}

func elixirTriggerValues(trigger Trigger) string {
	values := trigger.Values
	if len(values) == 0 {
		values = []TriggerValue{{Value: trigger.Value, IsString: trigger.IsString, Language: trigger.Language}}
	}
	rendered := make([]string, 0, len(values))
	for _, value := range values {
		text := value.Value
		if value.EventSymbol != "" {
			text = eventSymbolFor(LanguageElixir, value.EventSymbol) + "()"
		} else if value.IsString {
			text = quoteElixir(text)
		}
		rendered = append(rendered, text)
	}
	if len(rendered) == 1 {
		return rendered[0]
	}
	return "[" + strings.Join(rendered, ", ") + "]"
}

func elixirExpressionValue(trigger Trigger, behaviorNames map[string]string, triggerFallbackNames map[string]string) string {
	if trigger.IsString {
		return quoteElixir(trigger.Value)
	}
	if trigger.Expr != nil {
		if name := behaviorNames[trigger.Expr.ID]; name != "" {
			return "&__MODULE__." + name + "/3"
		}
	}
	if trigger.Language == "" || trigger.Language == LanguageElixir {
		return trigger.Value
	}
	if name := triggerFallbackNames[elixirTriggerFallbackKey(trigger)]; name != "" {
		return "&__MODULE__." + name + "/3"
	}
	return "0"
}

func elixirTriggerFallbackReturn(kind TriggerKind) string {
	if kind == TriggerWhen {
		return "false"
	}
	return "0"
}

type elixirTriggerFallback struct {
	Name     string
	Kind     TriggerKind
	Language Language
	Value    string
}

func collectElixirTriggerFallbacks(program *Program) []elixirTriggerFallback {
	if program == nil {
		return nil
	}
	seen := map[string]bool{}
	var fallbacks []elixirTriggerFallback
	var visitTransition func(Transition)
	var visitState func(State)
	visitTransition = func(transition Transition) {
		if transition.Trigger == nil || !elixirNeedsTriggerFallback(*transition.Trigger) {
			return
		}
		key := elixirTriggerFallbackKey(*transition.Trigger)
		if seen[key] {
			return
		}
		seen[key] = true
		fallbacks = append(fallbacks, elixirTriggerFallback{
			Name:     fmt.Sprintf("hsmc_trigger_fallback_%d", len(fallbacks)+1),
			Kind:     transition.Trigger.Kind,
			Language: transition.Trigger.Language,
			Value:    transition.Trigger.Value,
		})
	}
	visitState = func(state State) {
		for _, transition := range state.Transitions {
			visitTransition(transition)
		}
		for _, child := range state.States {
			visitState(child)
		}
	}
	for _, model := range program.Models {
		for _, transition := range model.Transitions {
			visitTransition(transition)
		}
		for _, state := range model.States {
			visitState(state)
		}
	}
	return fallbacks
}

func elixirTriggerFallbackNameMap(fallbacks []elixirTriggerFallback) map[string]string {
	names := map[string]string{}
	for _, fallback := range fallbacks {
		names[elixirTriggerFallbackKey(Trigger{
			Kind:     fallback.Kind,
			Language: fallback.Language,
			Value:    fallback.Value,
		})] = fallback.Name
	}
	return names
}

func elixirNeedsTriggerFallback(trigger Trigger) bool {
	if trigger.Expr != nil || trigger.IsString {
		return false
	}
	if trigger.Language == "" || trigger.Language == LanguageElixir {
		return false
	}
	switch trigger.Kind {
	case TriggerAfter, TriggerEvery, TriggerAt, TriggerWhen:
		return true
	default:
		return false
	}
}

func elixirTriggerFallbackKey(trigger Trigger) string {
	return strings.Join([]string{string(trigger.Kind), string(trigger.Language), trigger.Value}, "\x00")
}
