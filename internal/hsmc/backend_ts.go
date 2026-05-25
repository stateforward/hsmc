package hsmc

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
)

type TypeScriptBackend struct{}

func NewTypeScriptBackend() TypeScriptBackend {
	return TypeScriptBackend{}
}

func (TypeScriptBackend) Language() Language { return LanguageTS }

func (backend TypeScriptBackend) Emit(_ context.Context, program *Program) ([]byte, error) {
	var out bytes.Buffer
	backend.emitImports(&out, program)
	for _, event := range program.Events {
		backend.emitEvent(&out, event)
	}
	for _, global := range program.Globals {
		if global.Language != "" && global.Language != LanguageTS {
			writeCommentedGlobalSource(&out, "", "//", global, LanguageTS)
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
		name := lowerCamel(exportName(behavior.ID))
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

func (backend TypeScriptBackend) emitImports(out *bytes.Buffer, program *Program) {
	out.WriteString(`import * as hsm from "@stateforward/hsm.ts";` + "\n")
	for _, imp := range collectRenderableImports(program, LanguageTS, compilerOwnedImports(LanguageTS)...) {
		fmt.Fprintln(out, formatESImport(imp))
	}
	out.WriteString("\n")
}

func (backend TypeScriptBackend) emitEvent(out *bytes.Buffer, event EventDecl) {
	fmt.Fprintf(out, "const %s = hsm.Event(%s", eventSymbolFor(LanguageTS, event.Symbol), quoteTS(event.Name))
	if event.Schema != "" && event.Language == LanguageTS {
		fmt.Fprintf(out, ", %s", event.Schema)
	}
	out.WriteString(");\n\n")
}

func (backend TypeScriptBackend) emitBehavior(out *bytes.Buffer, name string, behavior Behavior) error {
	if (behavior.SourceLanguage == LanguageTS || behavior.TargetLanguage == LanguageTS) && strings.TrimSpace(behavior.Body) == "" {
		code := strings.TrimSpace(behavior.Code)
		if code != "" {
			if declaredName, ok := esTopLevelDeclarationName(code); ok {
				if declaredName != name {
					return fmt.Errorf("typescript behavior full code declares %q, want compiler-owned name %q", declaredName, name)
				}
				callable := false
				if _, parameters, returnType, ok := esFunctionDeclarationShape(code); ok {
					callable = true
					if wantParameters := tsParameterList(); parameters != wantParameters {
						return fmt.Errorf("typescript behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
					}
					if wantReturnType := tsReturnType(behavior); returnType != wantReturnType {
						return fmt.Errorf("typescript behavior full code declares return type %q, want compiler-owned return type %q", returnType, wantReturnType)
					}
				}
				if parameters, returnType, ok := esFunctionExpressionShape(code); ok {
					callable = true
					if wantParameters := tsParameterList(); parameters != wantParameters {
						return fmt.Errorf("typescript behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
					}
					if returnType != "" {
						if wantReturnType := tsReturnType(behavior); returnType != wantReturnType {
							return fmt.Errorf("typescript behavior full code declares return type %q, want compiler-owned return type %q", returnType, wantReturnType)
						}
					}
				}
				if !callable {
					return fmt.Errorf("typescript behavior full code for %q must be a compiler-owned callable declaration", name)
				}
				out.WriteString(ensureTrailingSemicolonForESDeclaration(code))
				out.WriteString("\n\n")
				return nil
			}
			parameters, returnType, ok := esFunctionExpressionShape(code)
			if !ok {
				if behavior.TargetLanguage == LanguageTS {
					return fmt.Errorf("typescript behavior full code for %q must be a compiler-owned callable expression", name)
				}
				fmt.Fprintf(out, "const %s = %s;\n\n", name, code)
				return nil
			}
			if wantParameters := tsParameterList(); parameters != wantParameters {
				return fmt.Errorf("typescript behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
			}
			if returnType != "" {
				if wantReturnType := tsReturnType(behavior); returnType != wantReturnType {
					return fmt.Errorf("typescript behavior full code declares return type %q, want compiler-owned return type %q", returnType, wantReturnType)
				}
			}
			fmt.Fprintf(out, "const %s = %s;\n\n", name, code)
			return nil
		}
	}
	instanceName := "instance"
	if behavior.SourceLanguage == LanguageTS && behavior.TargetLanguage != LanguageTS && strings.TrimSpace(behavior.Body) != "" {
		instanceName = "_instance"
	}
	fmt.Fprintf(out, "function %s(ctx: hsm.Context, %s: InstanceType<typeof hsm.Instance>, event: hsm.Event): %s {\n", name, instanceName, tsReturnType(behavior))
	var body string
	if behavior.TargetLanguage == LanguageTS || behavior.SourceLanguage == LanguageTS {
		body = strings.TrimSpace(behavior.Body)
	}
	if body == "" {
		writeCommentedBehaviorSource(out, "\t", "//", behavior)
		if ret := tsUntranslatedReturn(behavior); ret != "" {
			fmt.Fprintf(out, "\t%s\n", ret)
		}
	} else {
		if instanceName != "instance" {
			out.WriteString("\tconst instance = _instance as any;\n")
		}
		out.WriteString(indentTS(body))
		out.WriteString("\n")
		if behavior.Kind == BehaviorGuard && !strings.Contains(body, "return") {
			out.WriteString("\treturn false;\n")
		}
	}
	out.WriteString("}\n\n")
	return nil
}

func tsParameterList() string {
	return "(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event)"
}

func tsReturnType(behavior Behavior) string {
	if behavior.TargetABI != nil && behavior.TargetABI.ReturnType != "" {
		return behavior.TargetABI.ReturnType
	}
	if behavior.Kind == BehaviorGuard {
		return "boolean"
	}
	return "void"
}

func (backend TypeScriptBackend) emitModel(out *bytes.Buffer, model Model, behaviorNames map[string]string) {
	define := "hsm.Define"
	if tsModelNeedsAnyCast(model) {
		define = "(hsm.Define as any)"
	}
	fmt.Fprintf(out, "export const %sModel = %s(\n", exportName(model.Name), define)
	fmt.Fprintf(out, "\t%s,\n", quoteTS(model.Name))
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
		backend.emitState(out, "\t", state, behaviorNames)
	}
	for _, transition := range model.Transitions {
		backend.emitTransition(out, "\t", transition, behaviorNames)
	}
	out.WriteString(");\n")
}

func tsModelNeedsAnyCast(model Model) bool {
	for _, transition := range model.Transitions {
		if tsTransitionNeedsAnyCast(transition) {
			return true
		}
	}
	for _, state := range model.States {
		if tsStateNeedsAnyCast(state) {
			return true
		}
	}
	return false
}

func (backend TypeScriptBackend) emitAttribute(out *bytes.Buffer, indent string, attribute Attribute) {
	if !attributeTypeIsTargetExpression(attribute, LanguageTS) {
		if attribute.Type != "" {
			fmt.Fprintf(out, "%s// Type: %s\n", indent, attribute.Type)
		}
		if defaultValue, ok := esAttributeTargetDefault(attribute, LanguageTS); ok {
			fmt.Fprintf(out, "%shsm.Attribute(%s, %s),\n", indent, quoteTS(attribute.Name), defaultValue)
			return
		}
		if attribute.HasDefault {
			fmt.Fprintf(out, "%s// Default: %s\n", indent, attribute.Default)
		}
		fmt.Fprintf(out, "%shsm.Attribute(%s),\n", indent, quoteTS(attribute.Name))
		return
	}
	if defaultValue, ok := esAttributeTargetDefault(attribute, LanguageTS); ok {
		fmt.Fprintf(out, "%shsm.Attribute(%s, %s, %s),\n", indent, quoteTS(attribute.Name), attribute.Type, defaultValue)
		return
	}
	if attribute.HasDefault {
		fmt.Fprintf(out, "%s// Default: %s\n", indent, attribute.Default)
	}
	fmt.Fprintf(out, "%shsm.Attribute(%s, %s),\n", indent, quoteTS(attribute.Name), attribute.Type)
}

func esAttributeTargetDefault(attribute Attribute, target Language) (string, bool) {
	if !attribute.HasDefault {
		return "", false
	}
	value := strings.TrimSpace(attribute.Default)
	if value == "" {
		return "", false
	}
	if attribute.Language == "" || attribute.Language == target {
		return value, true
	}
	switch value {
	case "True":
		return "true", true
	case "False":
		return "false", true
	case "None", "nil":
		return "null", true
	}
	if value == "true" || value == "false" || value == "null" {
		return value, true
	}
	if esNumericLiteral(value) {
		return value, true
	}
	if _, err := strconv.Unquote(value); err == nil {
		if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
			return value, true
		}
	}
	return "", false
}

func esNumericLiteral(value string) bool {
	lower := strings.ToLower(value)
	if strings.Contains(lower, "nan") || strings.Contains(lower, "inf") {
		return false
	}
	_, err := strconv.ParseFloat(value, 64)
	return err == nil
}

func (backend TypeScriptBackend) emitOperation(out *bytes.Buffer, indent string, operation Operation, behaviorNames map[string]string) {
	if operation.Behavior == nil {
		return
	}
	name := behaviorNames[operation.Behavior.ID]
	if name == "" {
		name = operation.Behavior.ID
	}
	fmt.Fprintf(out, "%shsm.Operation(%s, %s),\n", indent, quoteTS(operation.Name), name)
}

func (backend TypeScriptBackend) emitState(out *bytes.Buffer, indent string, state State, behaviorNames map[string]string) {
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
	fmt.Fprintf(out, "%s\t%s,\n", indent, quoteTS(state.Name))
	if state.Initializer != nil {
		backend.emitInitial(out, indent+"\t", *state.Initializer, behaviorNames)
	}
	for _, behavior := range state.Behaviors {
		backend.emitBehaviorPartial(out, indent+"\t", behavior, behaviorNames)
	}
	for _, eventName := range state.Defers {
		fmt.Fprintf(out, "%shsm.Defer(%s),\n", indent+"\t", quoteTS(eventName))
	}
	for _, child := range state.States {
		backend.emitState(out, indent+"\t", child, behaviorNames)
	}
	for _, transition := range state.Transitions {
		backend.emitTransition(out, indent+"\t", transition, behaviorNames)
	}
	if tsStateNeedsAnyCast(state) {
		fmt.Fprintf(out, "%s) as any,\n", indent)
		return
	}
	fmt.Fprintf(out, "%s),\n", indent)
}

func tsStateNeedsAnyCast(state State) bool {
	for _, transition := range state.Transitions {
		if tsTransitionNeedsAnyCast(transition) {
			return true
		}
	}
	for _, child := range state.States {
		if tsStateNeedsAnyCast(child) {
			return true
		}
	}
	return false
}

func (backend TypeScriptBackend) emitInitial(out *bytes.Buffer, indent string, initial Initial, behaviorNames map[string]string) {
	fmt.Fprintf(out, "%shsm.Initial(\n", indent)
	if initial.Target != "" {
		fmt.Fprintf(out, "%s\thsm.Target(%s),\n", indent, quoteTS(initial.Target))
	}
	for _, effect := range initial.Effects {
		backend.emitBehaviorPartial(out, indent+"\t", effect, behaviorNames)
	}
	fmt.Fprintf(out, "%s),\n", indent)
}

func (backend TypeScriptBackend) emitTransition(out *bytes.Buffer, indent string, transition Transition, behaviorNames map[string]string) {
	fmt.Fprintf(out, "%shsm.Transition(\n", indent)
	if transition.Source != "" {
		fmt.Fprintf(out, "%s\thsm.Source(%s),\n", indent, quoteTS(transition.Source))
	}
	if transition.Trigger != nil {
		backend.emitTrigger(out, indent+"\t", *transition.Trigger, behaviorNames)
	}
	if transition.Guard != nil {
		backend.emitBehaviorPartial(out, indent+"\t", *transition.Guard, behaviorNames)
	}
	if transition.Target != "" {
		fmt.Fprintf(out, "%s\thsm.Target(%s),\n", indent, quoteTS(transition.Target))
	}
	for _, effect := range transition.Effects {
		backend.emitBehaviorPartial(out, indent+"\t", effect, behaviorNames)
	}
	if tsTransitionNeedsAnyCast(transition) {
		fmt.Fprintf(out, "%s) as any,\n", indent)
		return
	}
	fmt.Fprintf(out, "%s),\n", indent)
}

func tsTransitionNeedsAnyCast(transition Transition) bool {
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerWhen {
		return false
	}
	return transition.Trigger.Expr != nil || !transition.Trigger.IsString
}

func (backend TypeScriptBackend) emitTrigger(out *bytes.Buffer, indent string, trigger Trigger, behaviorNames map[string]string) {
	switch trigger.Kind {
	case TriggerOn:
		fmt.Fprintf(out, "%shsm.On(%s),\n", indent, tsTriggerValues(trigger))
	case TriggerOnSet:
		fmt.Fprintf(out, "%shsm.OnSet(%s),\n", indent, quoteTS(trigger.Value))
	case TriggerOnCall:
		fmt.Fprintf(out, "%shsm.OnCall(%s),\n", indent, quoteTS(trigger.Value))
	case TriggerAfter:
		fmt.Fprintf(out, "%shsm.After(%s),\n", indent, tsExpressionValue(trigger, behaviorNames))
	case TriggerEvery:
		fmt.Fprintf(out, "%shsm.Every(%s),\n", indent, tsExpressionValue(trigger, behaviorNames))
	case TriggerAt:
		fmt.Fprintf(out, "%shsm.At(%s),\n", indent, tsExpressionValue(trigger, behaviorNames))
	case TriggerWhen:
		if trigger.IsString && trigger.Value != "" && trigger.Expr == nil {
			fmt.Fprintf(out, "%shsm.OnSet(%s),\n", indent, quoteTS(trigger.Value))
			return
		}
		if trigger.Expr != nil || !trigger.IsString {
			fmt.Fprintf(out, "%s(hsm.When as any)(%s),\n", indent, tsExpressionValue(trigger, behaviorNames))
			return
		}
		fmt.Fprintf(out, "%shsm.When(%s),\n", indent, tsExpressionValue(trigger, behaviorNames))
	}
}

func tsTriggerValues(trigger Trigger) string {
	values := trigger.Values
	if len(values) == 0 {
		values = []TriggerValue{{Value: trigger.Value, IsString: trigger.IsString, Language: trigger.Language}}
	}
	rendered := make([]string, 0, len(values))
	for _, value := range values {
		text := value.Value
		if value.EventSymbol != "" {
			text = eventSymbolFor(LanguageTS, value.EventSymbol)
		} else if value.IsString {
			text = quoteTS(text)
		}
		rendered = append(rendered, text)
	}
	return strings.Join(rendered, ", ")
}

func (backend TypeScriptBackend) emitBehaviorPartial(out *bytes.Buffer, indent string, ref BehaviorRef, behaviorNames map[string]string) {
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
		name = quoteTS(ref.Operation)
	}
	if name == "" {
		name = ref.ID
	}
	fmt.Fprintf(out, "%shsm.%s(%s),\n", indent, constructor, name)
}

func quoteTS(value string) string {
	return strconv.Quote(value)
}

func indentTS(body string) string {
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

func lowerCamel(value string) string {
	if value == "" {
		return "generated"
	}
	return strings.ToLower(value[:1]) + value[1:]
}

func tsExpressionValue(trigger Trigger, behaviorNames map[string]string) string {
	if trigger.IsString {
		return quoteTS(trigger.Value)
	}
	if trigger.Expr != nil {
		if name := behaviorNames[trigger.Expr.ID]; name != "" {
			return name
		}
	}
	if trigger.Language == "" || trigger.Language == LanguageTS {
		return trigger.Value
	}
	return fmt.Sprintf("() => {\n\t// Original %s %s trigger expression preserved for manual porting:\n%s\n\t%s\n}", trigger.Language, trigger.Kind, commentLines("\t", "//", trigger.Value), tsTriggerDefaultReturn(trigger.Kind))
}

func tsTriggerDefaultReturn(kind TriggerKind) string {
	switch kind {
	case TriggerAt:
		return "return new Date(0);"
	case TriggerAfter, TriggerEvery:
		return "return 0;"
	default:
		return "return undefined;"
	}
}
