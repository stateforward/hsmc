package hsmc

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
)

type JavaBackend struct{}

func NewJavaBackend() JavaBackend {
	return JavaBackend{}
}

func (JavaBackend) Language() Language { return LanguageJava }

func (backend JavaBackend) Emit(_ context.Context, program *Program) ([]byte, error) {
	var out bytes.Buffer
	backend.emitImports(&out, program)
	out.WriteString("public final class GeneratedHsm {\n")
	out.WriteString("\tprivate GeneratedHsm() {}\n\n")
	for _, event := range program.Events {
		fmt.Fprintf(&out, "\tprivate static final Event %s = new Event(%s);\n\n", eventSymbolFor(LanguageJava, event.Symbol), strconv.Quote(event.Name))
	}
	for _, global := range program.Globals {
		if global.Language != "" && global.Language != LanguageJava {
			writeCommentedGlobalSource(&out, "\t", "//", global, LanguageJava)
			continue
		}
		if code := strings.TrimSpace(global.Code); code != "" {
			out.WriteString(indentJava(code))
			out.WriteString("\n\n")
		}
	}
	behaviorNames := map[string]string{}
	for _, behavior := range program.Behaviors {
		name := exportName(behavior.ID)
		behaviorNames[behavior.ID] = name
		if err := backend.emitBehavior(&out, name, behavior); err != nil {
			return nil, err
		}
	}
	for _, model := range program.Models {
		backend.emitModel(&out, model, behaviorNames)
	}
	out.WriteString("}\n")
	return out.Bytes(), nil
}

func (backend JavaBackend) emitImports(out *bytes.Buffer, program *Program) {
	out.WriteString("import com.stateforward.hsm.*;\n")
	for _, imp := range collectRenderableImports(program, LanguageJava, compilerOwnedImports(LanguageJava)...) {
		fmt.Fprintln(out, formatJavaImport(imp))
	}
	out.WriteString("\n")
}

func (backend JavaBackend) emitBehavior(out *bytes.Buffer, name string, behavior Behavior) error {
	if (behavior.TargetLanguage == LanguageJava || behavior.SourceLanguage == LanguageJava) && strings.TrimSpace(behavior.Body) == "" {
		if code := strings.TrimSpace(behavior.Code); code != "" {
			declaredName, ok := javaMethodDeclarationName(code)
			if !ok {
				return fmt.Errorf("java behavior code for %q must be a compiler-owned method declaration; use body for method statements", name)
			}
			if declaredName != name {
				return fmt.Errorf("java behavior full code declares %q, want compiler-owned name %q", declaredName, name)
			}
			if returnType, _, parameters, ok := cLikeMethodDeclarationShape(code); ok {
				if wantReturnType := javaReturnType(behavior); returnType != wantReturnType {
					return fmt.Errorf("java behavior full code declares return type %q, want compiler-owned return type %q", returnType, wantReturnType)
				}
				const wantParameters = "(Context ctx, Instance instance, Event event)"
				if parameters != wantParameters {
					return fmt.Errorf("java behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
				}
			}
			out.WriteString(indentJava(code))
			out.WriteString("\n\n")
			return nil
		}
	}
	body := ""
	if behavior.TargetLanguage == LanguageJava || behavior.SourceLanguage == LanguageJava {
		body = strings.TrimSpace(behavior.Body)
	}
	fmt.Fprintf(out, "\tprivate static %s %s(Context ctx, Instance instance, Event event) {\n", javaReturnType(behavior), name)
	if body == "" {
		writeCommentedBehaviorSource(out, "\t\t", "//", behavior)
		if ret := javaUntranslatedReturn(behavior); ret != "" {
			fmt.Fprintf(out, "\t\t%s\n", ret)
		}
	} else {
		out.WriteString(indentJava(body, "\t\t"))
		out.WriteString("\n")
		if behavior.Kind == BehaviorGuard && !strings.Contains(body, "return") {
			out.WriteString("\t\treturn false;\n")
		}
	}
	out.WriteString("\t}\n\n")
	return nil
}

func (backend JavaBackend) emitModel(out *bytes.Buffer, model Model, behaviorNames map[string]string) {
	fmt.Fprintf(out, "\tpublic static final Model %sModel = %s;\n\n", exportName(model.Name), backend.modelExpr(model, behaviorNames))
}

func (backend JavaBackend) modelExpr(model Model, behaviorNames map[string]string) string {
	args := []string{strconv.Quote(model.Name)}
	if model.Initializer != nil {
		args = append(args, backend.initialExpr(*model.Initializer, behaviorNames))
	}
	for _, attribute := range model.Attributes {
		args = append(args, backend.attributeExpr(attribute))
	}
	for _, operation := range model.Operations {
		if operation.Behavior != nil {
			args = append(args, fmt.Sprintf("Hsm.Operation(%s, GeneratedHsm::%s)", strconv.Quote(operation.Name), backend.behaviorName(*operation.Behavior, behaviorNames)))
		}
	}
	for _, state := range model.States {
		args = append(args, backend.stateExpr(state, behaviorNames))
	}
	for _, transition := range model.Transitions {
		args = append(args, backend.transitionExpr(transition, behaviorNames))
	}
	return "Hsm.Define(" + strings.Join(args, ", ") + ")"
}

func (backend JavaBackend) stateExpr(state State, behaviorNames map[string]string) string {
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
	args := []string{strconv.Quote(state.Name)}
	if state.Initializer != nil {
		args = append(args, backend.initialExpr(*state.Initializer, behaviorNames))
	}
	for _, behavior := range state.Behaviors {
		args = append(args, backend.behaviorPartialExpr(behavior, behaviorNames))
	}
	for _, eventName := range state.Defers {
		args = append(args, fmt.Sprintf("Hsm.Defer(%s)", strconv.Quote(eventName)))
	}
	for _, child := range state.States {
		args = append(args, backend.stateExpr(child, behaviorNames))
	}
	for _, transition := range state.Transitions {
		args = append(args, backend.transitionExpr(transition, behaviorNames))
	}
	return "Hsm." + constructor + "(" + strings.Join(args, ", ") + ")"
}

func (backend JavaBackend) initialExpr(initial Initial, behaviorNames map[string]string) string {
	var args []string
	if initial.Target != "" {
		args = append(args, fmt.Sprintf("Hsm.Target(%s)", strconv.Quote(initial.Target)))
	}
	for _, effect := range initial.Effects {
		args = append(args, backend.behaviorPartialExpr(effect, behaviorNames))
	}
	return "Hsm.Initial(" + strings.Join(args, ", ") + ")"
}

func (backend JavaBackend) transitionExpr(transition Transition, behaviorNames map[string]string) string {
	var args []string
	if transition.Source != "" {
		args = append(args, fmt.Sprintf("Hsm.Source(%s)", strconv.Quote(transition.Source)))
	}
	if transition.Trigger != nil {
		args = append(args, backend.triggerExpr(*transition.Trigger, behaviorNames))
	}
	if transition.Guard != nil {
		args = append(args, backend.behaviorPartialExpr(*transition.Guard, behaviorNames))
	}
	if transition.Target != "" {
		args = append(args, fmt.Sprintf("Hsm.Target(%s)", strconv.Quote(transition.Target)))
	}
	for _, effect := range transition.Effects {
		args = append(args, backend.behaviorPartialExpr(effect, behaviorNames))
	}
	return "Hsm.Transition(" + strings.Join(args, ", ") + ")"
}

func (backend JavaBackend) attributeExpr(attribute Attribute) string {
	prefix := ""
	if attribute.Type != "" && !attributeTypeIsTargetExpression(attribute, LanguageJava) {
		prefix = "/* Type: " + attribute.Type + " */ "
	}
	if attributeTypeIsTargetExpression(attribute, LanguageJava) && attribute.HasDefault {
		return fmt.Sprintf("Hsm.Attribute(%s, %s, %s)", strconv.Quote(attribute.Name), attribute.Type, javaAttributeDefault(attribute))
	}
	if attributeTypeIsTargetExpression(attribute, LanguageJava) {
		return fmt.Sprintf("Hsm.Attribute(%s, %s)", strconv.Quote(attribute.Name), attribute.Type)
	}
	if attribute.HasDefault {
		return prefix + fmt.Sprintf("Hsm.Attribute(%s, %s)", strconv.Quote(attribute.Name), javaAttributeDefault(attribute))
	}
	return prefix + fmt.Sprintf("Hsm.Attribute(%s)", strconv.Quote(attribute.Name))
}

func (backend JavaBackend) triggerExpr(trigger Trigger, behaviorNames map[string]string) string {
	switch trigger.Kind {
	case TriggerOn:
		values := trigger.Values
		if len(values) == 0 && trigger.Value != "" {
			values = []TriggerValue{{Value: trigger.Value, IsString: trigger.IsString, Language: trigger.Language}}
		}
		parts := make([]string, 0, len(values))
		for _, value := range values {
			text := strconv.Quote(value.Value)
			if value.EventSymbol != "" {
				text = eventSymbolFor(LanguageJava, value.EventSymbol)
			}
			parts = append(parts, text)
		}
		return "Hsm.On(" + strings.Join(parts, ", ") + ")"
	case TriggerOnSet:
		return fmt.Sprintf("Hsm.OnSet(%s)", strconv.Quote(trigger.Value))
	case TriggerOnCall:
		return fmt.Sprintf("Hsm.OnCall(%s)", strconv.Quote(trigger.Value))
	case TriggerAfter:
		return "Hsm.After(" + backend.triggerValueExpr(trigger, behaviorNames) + ")"
	case TriggerEvery:
		return "Hsm.Every(" + backend.triggerValueExpr(trigger, behaviorNames) + ")"
	case TriggerAt:
		return "Hsm.At(" + backend.triggerValueExpr(trigger, behaviorNames) + ")"
	case TriggerWhen:
		if trigger.Expr != nil {
			return "Hsm.When(GeneratedHsm::" + backend.behaviorName(*trigger.Expr, behaviorNames) + ")"
		}
		if trigger.Value != "" {
			return fmt.Sprintf("Hsm.OnSet(%s)", strconv.Quote(trigger.Value))
		}
		return "Hsm.When(() -> false)"
	default:
		return "Hsm.On(\"<unknown>\")"
	}
}

func (backend JavaBackend) triggerValueExpr(trigger Trigger, behaviorNames map[string]string) string {
	if trigger.Expr != nil {
		return "GeneratedHsm::" + backend.behaviorName(*trigger.Expr, behaviorNames)
	}
	if trigger.Language == "" || trigger.Language == LanguageJava {
		return strings.TrimSpace(trigger.Value)
	}
	fallback := "java.time.Duration.ZERO"
	if trigger.Kind == TriggerAt {
		fallback = "java.time.Instant.EPOCH"
	}
	return fallback + " /* Original " + string(trigger.Language) + " " + string(trigger.Kind) + " trigger preserved for manual porting. */"
}

func (backend JavaBackend) behaviorPartialExpr(ref BehaviorRef, behaviorNames map[string]string) string {
	if ref.Operation != "" {
		operation := strconv.Quote(ref.Operation)
		switch ref.Kind {
		case BehaviorEntry:
			return "Hsm.Entry((ctx, instance, event) -> Hsm.Call(ctx, instance, " + operation + "))"
		case BehaviorExit:
			return "Hsm.Exit((ctx, instance, event) -> Hsm.Call(ctx, instance, " + operation + "))"
		case BehaviorActivity:
			return "Hsm.Activity((ctx, instance, event) -> Hsm.Call(ctx, instance, " + operation + "))"
		case BehaviorGuard:
			return "Hsm.Guard((ctx, instance, event) -> Boolean.TRUE.equals(Hsm.Call(ctx, instance, " + operation + ")))"
		case BehaviorEffect:
			return "Hsm.Effect((ctx, instance, event) -> Hsm.Call(ctx, instance, " + operation + "))"
		}
	}
	name := backend.behaviorName(ref, behaviorNames)
	switch ref.Kind {
	case BehaviorEntry:
		return "Hsm.Entry(GeneratedHsm::" + name + ")"
	case BehaviorExit:
		return "Hsm.Exit(GeneratedHsm::" + name + ")"
	case BehaviorActivity:
		return "Hsm.Activity(GeneratedHsm::" + name + ")"
	case BehaviorGuard:
		return "Hsm.Guard(GeneratedHsm::" + name + ")"
	case BehaviorEffect:
		return "Hsm.Effect(GeneratedHsm::" + name + ")"
	case BehaviorOperation:
		return "GeneratedHsm::" + name
	default:
		return "GeneratedHsm::" + name
	}
}

func (backend JavaBackend) behaviorName(ref BehaviorRef, behaviorNames map[string]string) string {
	if name := behaviorNames[ref.ID]; name != "" {
		return name
	}
	return exportName(ref.ID)
}

func javaReturnType(behavior Behavior) string {
	if behavior.TargetABI != nil && behavior.TargetABI.ReturnType != "" {
		return behavior.TargetABI.ReturnType
	}
	if behavior.Kind == BehaviorGuard {
		return "boolean"
	}
	return "void"
}

func javaUntranslatedReturn(behavior Behavior) string {
	returnType := javaReturnType(behavior)
	switch returnType {
	case "", "void":
		return ""
	case "boolean":
		return "return false;"
	case "java.time.Duration":
		return "return java.time.Duration.ZERO;"
	case "java.time.Instant":
		return "return java.time.Instant.EPOCH;"
	default:
		return "return null;"
	}
}

func javaAttributeDefault(attribute Attribute) string {
	value := strings.TrimSpace(attribute.Default)
	if attribute.Language == "" || attribute.Language == LanguageJava {
		return value
	}
	if converted, ok := javaPortableAttributeLiteral(value); ok {
		return converted
	}
	return "null /* Original " + string(attribute.Language) + " default preserved for manual porting: " + value + " */"
}

func javaPortableAttributeLiteral(value string) (string, bool) {
	if value == "" {
		return "", false
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
	if javaNumericLiteral(value) {
		return value, true
	}
	if _, err := strconv.Unquote(value); err == nil {
		if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
			return value, true
		}
	}
	return "", false
}

func javaNumericLiteral(value string) bool {
	lower := strings.ToLower(value)
	if strings.Contains(lower, "nan") || strings.Contains(lower, "inf") {
		return false
	}
	_, err := strconv.ParseFloat(value, 64)
	return err == nil
}

func indentJava(code string, prefixes ...string) string {
	prefix := "\t"
	if len(prefixes) > 0 {
		prefix = prefixes[0]
	}
	lines := strings.Split(strings.TrimRight(code, "\n"), "\n")
	for index, line := range lines {
		lines[index] = prefix + line
	}
	return strings.Join(lines, "\n")
}
