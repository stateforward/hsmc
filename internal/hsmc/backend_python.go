package hsmc

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
)

type PythonBackend struct{}

func NewPythonBackend() PythonBackend {
	return PythonBackend{}
}

func (PythonBackend) Language() Language { return LanguagePython }

func (backend PythonBackend) Emit(_ context.Context, program *Program) ([]byte, error) {
	var out bytes.Buffer
	backend.emitImports(&out, program)
	for _, event := range program.Events {
		backend.emitEvent(&out, event)
	}
	for _, global := range program.Globals {
		if global.Language != "" && global.Language != LanguagePython {
			writeCommentedGlobalSource(&out, "", "#", global, LanguagePython)
			continue
		}
		code := strings.TrimSpace(global.Code)
		if code == "" {
			continue
		}
		out.WriteString(code)
		out.WriteString("\n\n")
	}
	behaviorNames := map[string]string{}
	for _, behavior := range program.Behaviors {
		name := snakeName(behavior.ID)
		behaviorNames[behavior.ID] = name
		if err := backend.emitBehavior(&out, name, behavior); err != nil {
			return nil, err
		}
	}
	triggerFallbacks := collectPythonTriggerFallbacks(program)
	for _, fallback := range triggerFallbacks {
		backend.emitTriggerFallback(&out, fallback)
	}
	triggerFallbackNames := pythonTriggerFallbackNameMap(triggerFallbacks)
	for _, model := range program.Models {
		backend.emitModel(&out, model, behaviorNames, triggerFallbackNames)
	}
	return out.Bytes(), nil
}

func (backend PythonBackend) emitImports(out *bytes.Buffer, program *Program) {
	out.WriteString("import datetime\n")
	out.WriteString("import hsm\n")
	out.WriteString("import typing\n")
	for _, imp := range collectRenderableImports(program, LanguagePython, compilerOwnedImports(LanguagePython)...) {
		fmt.Fprintln(out, formatPythonImport(imp))
	}
	out.WriteString("\n\n")
}

func (backend PythonBackend) emitEvent(out *bytes.Buffer, event EventDecl) {
	fmt.Fprintf(out, "%s = hsm.Event(name=%s)\n\n", eventSymbolFor(LanguagePython, event.Symbol), quotePython(event.Name))
}

func (backend PythonBackend) emitBehavior(out *bytes.Buffer, name string, behavior Behavior) error {
	if (behavior.SourceLanguage == LanguagePython || behavior.TargetLanguage == LanguagePython) && strings.TrimSpace(behavior.Body) == "" {
		code := strings.TrimSpace(behavior.Code)
		if code != "" {
			if declaredName, ok := pythonFunctionDeclarationName(code); ok {
				if declaredName != name {
					return fmt.Errorf("python behavior full code declares %q, want compiler-owned name %q", declaredName, name)
				}
				if async, _, parameters, returnType, ok := pythonFunctionDeclarationShape(code); ok {
					if !async {
						return fmt.Errorf("python behavior full code declares synchronous function %q, want compiler-owned async function", name)
					}
					if wantParameters := pythonParameterList(); parameters != wantParameters {
						return fmt.Errorf("python behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
					}
					if wantReturnType := pythonReturnType(behavior); returnType != wantReturnType {
						return fmt.Errorf("python behavior full code declares return type %q, want compiler-owned return type %q", returnType, wantReturnType)
					}
				}
				out.WriteString(code)
				if !strings.HasSuffix(code, "\n") {
					out.WriteString("\n")
				}
				out.WriteString("\n")
				return nil
			}
			parameters, isLambda, ok := pythonLambdaParameterList(code)
			if !isLambda {
				if behavior.TargetLanguage == LanguagePython {
					return fmt.Errorf("python behavior full code for %q must be a compiler-owned callable expression", name)
				}
				fmt.Fprintf(out, "%s = %s\n\n", name, code)
				return nil
			}
			if !ok {
				return fmt.Errorf("python behavior full code declares unsupported lambda signature, want compiler-owned parameters %q", pythonExpressionParameterList())
			}
			if wantParameters := pythonExpressionParameterList(); parameters != wantParameters {
				return fmt.Errorf("python behavior full code declares lambda parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
			}
			fmt.Fprintf(out, "%s = %s\n\n", name, code)
			return nil
		}
	}
	body := ""
	if behavior.TargetLanguage == LanguagePython || behavior.SourceLanguage == LanguagePython {
		body = strings.TrimSpace(behavior.Body)
	}
	fmt.Fprintf(out, "async def %s%s -> %s:\n", name, pythonParameterList(), pythonReturnType(behavior))
	if body == "" {
		writeCommentedBehaviorSource(out, "\t", "#", behavior)
		fmt.Fprintf(out, "\t%s\n", pythonUntranslatedReturn(behavior))
	} else {
		out.WriteString(indentPython(body))
		out.WriteString("\n")
		if behavior.Kind == BehaviorGuard && !strings.Contains(body, "return") {
			out.WriteString("\treturn False\n")
		}
	}
	out.WriteString("\n")
	return nil
}

func pythonParameterList() string {
	return "(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event)"
}

func pythonExpressionParameterList() string {
	return "(ctx, instance, event)"
}

func pythonReturnType(behavior Behavior) string {
	if behavior.TargetABI != nil && behavior.TargetABI.ReturnType != "" {
		return behavior.TargetABI.ReturnType
	}
	if behavior.Kind == BehaviorGuard {
		return "bool"
	}
	return "None"
}

func (backend PythonBackend) emitTriggerFallback(out *bytes.Buffer, fallback pythonTriggerFallback) {
	returnType := "typing.Any"
	switch fallback.Kind {
	case TriggerAt:
		returnType = "datetime.datetime"
	case TriggerAfter, TriggerEvery:
		returnType = "float"
	case TriggerWhen:
		returnType = "bool"
	}
	fmt.Fprintf(out, "def %s(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> %s:\n", fallback.Name, returnType)
	fmt.Fprintf(out, "\t# Original %s %s trigger expression preserved for manual porting:\n", fallback.Language, fallback.Kind)
	source := strings.TrimSpace(fallback.Value)
	if source == "" {
		out.WriteString("\t# No source trigger expression was captured.\n")
	} else {
		for _, line := range strings.Split(source, "\n") {
			if strings.TrimSpace(line) == "" {
				out.WriteString("\t#\n")
				continue
			}
			fmt.Fprintf(out, "\t# %s\n", strings.ReplaceAll(line, "\t", "    "))
		}
	}
	fmt.Fprintf(out, "\t%s\n\n", pythonTriggerFallbackReturn(fallback.Kind))
}

func pythonTriggerFallbackReturn(kind TriggerKind) string {
	switch kind {
	case TriggerAt:
		return "return datetime.datetime.fromtimestamp(0)"
	case TriggerAfter, TriggerEvery:
		return "return 0.0"
	case TriggerWhen:
		return "return False"
	default:
		return "return None"
	}
}

func (backend PythonBackend) emitModel(out *bytes.Buffer, model Model, behaviorNames map[string]string, triggerFallbackNames map[string]string) {
	fmt.Fprintf(out, "%s_model = hsm.Define(\n", snakeName(model.Name))
	fmt.Fprintf(out, "\t%s,\n", quotePython(model.Name))
	if model.Initializer != nil {
		backend.emitInitial(out, "\t", *model.Initializer, behaviorNames)
	}
	for _, attribute := range model.Attributes {
		backend.emitAttribute(out, "\t", attribute)
	}
	for _, operation := range model.Operations {
		backend.emitOperation(out, "\t", operation, behaviorNames)
	}
	for _, state := range model.States {
		backend.emitState(out, "\t", state, behaviorNames, triggerFallbackNames)
	}
	for _, transition := range model.Transitions {
		backend.emitTransition(out, "\t", transition, behaviorNames, triggerFallbackNames)
	}
	out.WriteString(")\n\n")
}

func (backend PythonBackend) emitAttribute(out *bytes.Buffer, indent string, attribute Attribute) {
	if attribute.Type != "" {
		fmt.Fprintf(out, "%s# Type: %s\n", indent, attribute.Type)
	}
	if defaultValue, ok := pythonAttributeTargetDefault(attribute); ok {
		fmt.Fprintf(out, "%shsm.Attribute(%s, %s),\n", indent, quotePython(attribute.Name), defaultValue)
		return
	}
	if attribute.HasDefault {
		fmt.Fprintf(out, "%s# Default: %s\n", indent, attribute.Default)
	}
	fmt.Fprintf(out, "%shsm.Attribute(%s),\n", indent, quotePython(attribute.Name))
}

func pythonAttributeTargetDefault(attribute Attribute) (string, bool) {
	if !attribute.HasDefault {
		return "", false
	}
	value := strings.TrimSpace(attribute.Default)
	if value == "" {
		return "", false
	}
	if attribute.Language == "" || attribute.Language == LanguagePython {
		return value, true
	}
	switch value {
	case "true":
		return "True", true
	case "false":
		return "False", true
	case "null", "nil":
		return "None", true
	}
	if value == "True" || value == "False" || value == "None" {
		return value, true
	}
	if pythonNumericLiteral(value) {
		return value, true
	}
	if _, err := strconv.Unquote(value); err == nil {
		if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
			return value, true
		}
	}
	return "", false
}

func pythonNumericLiteral(value string) bool {
	lower := strings.ToLower(value)
	if strings.Contains(lower, "nan") || strings.Contains(lower, "inf") {
		return false
	}
	_, err := strconv.ParseFloat(value, 64)
	return err == nil
}

func (backend PythonBackend) emitOperation(out *bytes.Buffer, indent string, operation Operation, behaviorNames map[string]string) {
	if operation.Behavior == nil {
		return
	}
	name := behaviorNames[operation.Behavior.ID]
	if name == "" {
		name = snakeName(operation.Behavior.ID)
	}
	fmt.Fprintf(out, "%shsm.Operation(%s, %s),\n", indent, quotePython(operation.Name), name)
}

func (backend PythonBackend) emitState(out *bytes.Buffer, indent string, state State, behaviorNames map[string]string, triggerFallbackNames map[string]string) {
	constructor := "State"
	switch state.Kind {
	case StateKindFinal:
		constructor = "Final"
	case StateKindChoice:
		constructor = "Choice"
	case StateKindShallowHistory:
		constructor = "ShallowHistory"
	case StateKindDeepHistory:
		constructor = "DeepHistory"
	}
	fmt.Fprintf(out, "%shsm.%s(\n", indent, constructor)
	fmt.Fprintf(out, "%s\t%s,\n", indent, quotePython(state.Name))
	if state.Initializer != nil {
		backend.emitInitial(out, indent+"\t", *state.Initializer, behaviorNames)
	}
	for _, behavior := range state.Behaviors {
		backend.emitBehaviorPartial(out, indent+"\t", behavior, behaviorNames)
	}
	for _, eventName := range state.Defers {
		fmt.Fprintf(out, "%shsm.Defer(hsm.Event(name=%s)),\n", indent+"\t", quotePython(eventName))
	}
	for _, child := range state.States {
		backend.emitState(out, indent+"\t", child, behaviorNames, triggerFallbackNames)
	}
	for _, transition := range state.Transitions {
		backend.emitTransition(out, indent+"\t", transition, behaviorNames, triggerFallbackNames)
	}
	fmt.Fprintf(out, "%s),\n", indent)
}

func (backend PythonBackend) emitInitial(out *bytes.Buffer, indent string, initial Initial, behaviorNames map[string]string) {
	fmt.Fprintf(out, "%shsm.Initial(\n", indent)
	if initial.Target != "" {
		fmt.Fprintf(out, "%s\thsm.Target(%s),\n", indent, quotePython(initial.Target))
	}
	for _, effect := range initial.Effects {
		backend.emitBehaviorPartial(out, indent+"\t", effect, behaviorNames)
	}
	fmt.Fprintf(out, "%s),\n", indent)
}

func (backend PythonBackend) emitTransition(out *bytes.Buffer, indent string, transition Transition, behaviorNames map[string]string, triggerFallbackNames map[string]string) {
	fmt.Fprintf(out, "%shsm.Transition(\n", indent)
	if transition.Source != "" {
		fmt.Fprintf(out, "%s\thsm.Source(%s),\n", indent, quotePython(transition.Source))
	}
	if transition.Trigger != nil {
		backend.emitTrigger(out, indent+"\t", *transition.Trigger, behaviorNames, triggerFallbackNames)
	}
	if transition.Guard != nil {
		backend.emitBehaviorPartial(out, indent+"\t", *transition.Guard, behaviorNames)
	}
	if transition.Target != "" {
		fmt.Fprintf(out, "%s\thsm.Target(%s),\n", indent, quotePython(transition.Target))
	}
	for _, effect := range transition.Effects {
		backend.emitBehaviorPartial(out, indent+"\t", effect, behaviorNames)
	}
	fmt.Fprintf(out, "%s),\n", indent)
}

func (backend PythonBackend) emitTrigger(out *bytes.Buffer, indent string, trigger Trigger, behaviorNames map[string]string, triggerFallbackNames map[string]string) {
	switch trigger.Kind {
	case TriggerOn:
		fmt.Fprintf(out, "%shsm.On(%s),\n", indent, pythonTriggerValues(trigger))
	case TriggerOnSet:
		fmt.Fprintf(out, "%shsm.OnSet(%s),\n", indent, quotePython(trigger.Value))
	case TriggerOnCall:
		fmt.Fprintf(out, "%shsm.OnCall(%s),\n", indent, quotePython(trigger.Value))
	case TriggerAfter:
		fmt.Fprintf(out, "%shsm.After(%s),\n", indent, pythonExpressionValue(trigger, behaviorNames, triggerFallbackNames))
	case TriggerEvery:
		fmt.Fprintf(out, "%shsm.Every(%s),\n", indent, pythonExpressionValue(trigger, behaviorNames, triggerFallbackNames))
	case TriggerAt:
		fmt.Fprintf(out, "%shsm.At(%s),\n", indent, pythonExpressionValue(trigger, behaviorNames, triggerFallbackNames))
	case TriggerWhen:
		if trigger.IsString && trigger.Value != "" && trigger.Expr == nil {
			fmt.Fprintf(out, "%shsm.OnSet(%s),\n", indent, quotePython(trigger.Value))
			return
		}
		fmt.Fprintf(out, "%shsm.When(%s),\n", indent, pythonExpressionValue(trigger, behaviorNames, triggerFallbackNames))
	}
}

func pythonTriggerValues(trigger Trigger) string {
	values := trigger.Values
	if len(values) == 0 {
		values = []TriggerValue{{Value: trigger.Value, IsString: trigger.IsString, Language: trigger.Language}}
	}
	rendered := make([]string, 0, len(values))
	for _, value := range values {
		text := value.Value
		if value.EventSymbol != "" {
			text = eventSymbolFor(LanguagePython, value.EventSymbol)
		} else if value.IsString {
			text = quotePython(text)
		}
		rendered = append(rendered, text)
	}
	return strings.Join(rendered, ", ")
}

func (backend PythonBackend) emitBehaviorPartial(out *bytes.Buffer, indent string, ref BehaviorRef, behaviorNames map[string]string) {
	constructor := "Entry"
	switch ref.Kind {
	case BehaviorExit:
		constructor = "Exit"
	case BehaviorActivity:
		constructor = "Activity"
	case BehaviorGuard:
		constructor = "Guard"
	case BehaviorEffect:
		constructor = "Effect"
	}
	name := behaviorNames[ref.ID]
	if ref.Operation != "" {
		name = quotePython(ref.Operation)
	}
	if name == "" {
		name = snakeName(ref.ID)
	}
	fmt.Fprintf(out, "%shsm.%s(%s),\n", indent, constructor, name)
}

func quotePython(value string) string {
	return strconv.Quote(value)
}

func indentPython(body string) string {
	lines := strings.Split(body, "\n")
	for index, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[index] = ""
			continue
		}
		lines[index] = "\t" + line
	}
	return strings.Join(lines, "\n")
}

func pythonExpressionValue(trigger Trigger, behaviorNames map[string]string, triggerFallbackNames map[string]string) string {
	if trigger.IsString {
		return quotePython(trigger.Value)
	}
	if trigger.Expr != nil {
		if name := behaviorNames[trigger.Expr.ID]; name != "" {
			return name
		}
	}
	if trigger.Language == "" || trigger.Language == LanguagePython {
		return trigger.Value
	}
	if name := triggerFallbackNames[pythonTriggerFallbackKey(trigger)]; name != "" {
		return name
	}
	return pythonTriggerDefaultExpression(trigger.Kind)
}

func pythonTriggerDefaultExpression(kind TriggerKind) string {
	switch kind {
	case TriggerAt:
		return "lambda ctx, instance, event: datetime.datetime.fromtimestamp(0)"
	case TriggerAfter, TriggerEvery:
		return "lambda ctx, instance, event: 0.0"
	default:
		return "lambda ctx, instance, event: None"
	}
}

type pythonTriggerFallback struct {
	Name     string
	Kind     TriggerKind
	Language Language
	Value    string
}

func collectPythonTriggerFallbacks(program *Program) []pythonTriggerFallback {
	if program == nil {
		return nil
	}
	seen := map[string]string{}
	var fallbacks []pythonTriggerFallback
	var visitTransition func(Transition)
	var visitState func(State)
	visitTransition = func(transition Transition) {
		if transition.Trigger == nil || !pythonNeedsTriggerFallback(*transition.Trigger) {
			return
		}
		key := pythonTriggerFallbackKey(*transition.Trigger)
		if seen[key] != "" {
			return
		}
		name := fmt.Sprintf("hsmc_trigger_fallback_%d", len(fallbacks)+1)
		seen[key] = name
		fallbacks = append(fallbacks, pythonTriggerFallback{
			Name:     name,
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

func pythonTriggerFallbackNameMap(fallbacks []pythonTriggerFallback) map[string]string {
	names := map[string]string{}
	for _, fallback := range fallbacks {
		names[pythonTriggerFallbackKey(Trigger{
			Kind:     fallback.Kind,
			Language: fallback.Language,
			Value:    fallback.Value,
		})] = fallback.Name
	}
	return names
}

func pythonNeedsTriggerFallback(trigger Trigger) bool {
	if trigger.Expr != nil || trigger.IsString {
		return false
	}
	if trigger.Language == "" || trigger.Language == LanguagePython {
		return false
	}
	switch trigger.Kind {
	case TriggerAfter, TriggerEvery, TriggerAt, TriggerWhen:
		return true
	default:
		return false
	}
}

func pythonTriggerFallbackKey(trigger Trigger) string {
	return strings.Join([]string{string(trigger.Kind), string(trigger.Language), trigger.Value}, "\x00")
}

func snakeName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "generated"
	}
	var out strings.Builder
	previousLowerOrDigit := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			out.WriteRune(r)
			previousLowerOrDigit = true
		case r >= 'A' && r <= 'Z':
			if out.Len() > 0 && previousLowerOrDigit {
				out.WriteByte('_')
			}
			out.WriteRune(r + ('a' - 'A'))
			previousLowerOrDigit = true
		case r >= '0' && r <= '9':
			out.WriteRune(r)
			previousLowerOrDigit = true
		default:
			if out.Len() > 0 && !strings.HasSuffix(out.String(), "_") {
				out.WriteByte('_')
			}
			previousLowerOrDigit = false
		}
	}
	result := strings.Trim(out.String(), "_")
	if result == "" {
		return "generated"
	}
	if result[0] >= '0' && result[0] <= '9' {
		return "generated_" + result
	}
	return result
}
