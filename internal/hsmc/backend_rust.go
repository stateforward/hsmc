package hsmc

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
)

type RustBackend struct{}

func NewRustBackend() RustBackend {
	return RustBackend{}
}

func (RustBackend) Language() Language { return LanguageRust }

func (backend RustBackend) Emit(_ context.Context, program *Program) ([]byte, error) {
	var out bytes.Buffer
	backend.emitImports(&out, program)
	if len(program.Models) > 0 {
		backend.emitGeneratedInstance(&out)
	}
	for _, event := range program.Events {
		backend.emitEvent(&out, event)
	}
	for _, global := range program.Globals {
		if global.Language != "" && global.Language != LanguageRust {
			writeCommentedGlobalSource(&out, "", "//", global, LanguageRust)
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

func (backend RustBackend) emitImports(out *bytes.Buffer, program *Program) {
	out.WriteString("use std::any::Any;\n")
	out.WriteString("use std::future::Future;\n")
	out.WriteString("use std::pin::Pin;\n")
	out.WriteString("use std::time::Duration;\n")
	for _, imp := range collectRenderableImports(program, LanguageRust, compilerOwnedImports(LanguageRust)...) {
		fmt.Fprintln(out, formatRustImport(imp))
	}
	out.WriteString("\n")
}

func (backend RustBackend) emitGeneratedInstance(out *bytes.Buffer) {
	out.WriteString("pub struct HsmcInstance;\n\n")
	out.WriteString("impl hsm::Instance for HsmcInstance {\n")
	out.WriteString("\tfn as_any(&self) -> &dyn Any { self }\n")
	out.WriteString("\tfn as_any_mut(&mut self) -> &mut dyn Any { self }\n")
	out.WriteString("}\n\n")
	out.WriteString("fn hsmc_true_guard(ctx: &hsm::Context, instance: &HsmcInstance, event: &hsm::Event) -> bool {\n")
	out.WriteString("\tlet _ = (ctx, instance, event);\n")
	out.WriteString("\ttrue\n")
	out.WriteString("}\n\n")
	out.WriteString("fn hsmc_zero_duration(ctx: &hsm::Context, instance: &HsmcInstance, event: &hsm::Event) -> Duration {\n")
	out.WriteString("\tlet _ = (ctx, instance, event);\n")
	out.WriteString("\tDuration::from_secs(0)\n")
	out.WriteString("}\n\n")
	out.WriteString("fn hsmc_unix_epoch(ctx: &hsm::Context, instance: &HsmcInstance, event: &hsm::Event) -> std::time::SystemTime {\n")
	out.WriteString("\tlet _ = (ctx, instance, event);\n")
	out.WriteString("\tstd::time::UNIX_EPOCH\n")
	out.WriteString("}\n\n")
}

func (backend RustBackend) emitEvent(out *bytes.Buffer, event EventDecl) {
	fmt.Fprintf(out, "const %s: &str = %s;\n\n", eventSymbolFor(LanguageRust, event.Symbol), strconv.Quote(event.Name))
}

func (backend RustBackend) emitBehavior(out *bytes.Buffer, name string, behavior Behavior) error {
	if (behavior.SourceLanguage == LanguageRust || behavior.TargetLanguage == LanguageRust) && strings.TrimSpace(behavior.Body) == "" {
		code := strings.TrimSpace(behavior.Code)
		if code != "" {
			declaredName, ok := rustFunctionDeclarationName(code)
			if !ok {
				return fmt.Errorf("rust behavior code for %q must be a compiler-owned function declaration; use body for function statements", name)
			}
			if declaredName != name {
				return fmt.Errorf("rust behavior full code declares %q, want compiler-owned name %q", declaredName, name)
			}
			if _, parameters, returnType, ok := rustFunctionDeclarationShape(code); ok {
				if wantParameters := rustParameterList(behavior); parameters != wantParameters {
					return fmt.Errorf("rust behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
				}
				if wantReturnType := rustFullCodeReturnType(behavior); returnType != wantReturnType {
					return fmt.Errorf("rust behavior full code declares return type %q, want compiler-owned return type %q", returnType, wantReturnType)
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
	fmt.Fprintf(out, "fn %s%s", name, rustParameterList(behavior))
	if ret := rustReturnType(behavior); ret != "" {
		fmt.Fprintf(out, " -> %s", ret)
	}
	out.WriteString(" {\n")
	body := ""
	if behavior.TargetLanguage == LanguageRust || behavior.SourceLanguage == LanguageRust {
		body = strings.TrimSpace(behavior.Body)
	}
	if body == "" {
		writeCommentedBehaviorSource(out, "\t", "//", behavior)
		if ret := rustUntranslatedReturn(behavior); ret != "" {
			fmt.Fprintf(out, "\t%s\n", ret)
		}
	} else {
		out.WriteString(indentRust(body))
		out.WriteString("\n")
		if behavior.Kind == BehaviorGuard && !rustBodyReturnsValue(body) {
			out.WriteString("\tfalse\n")
		}
		if rustBehaviorReturnsFuture(behavior) && !rustBodyReturnsValue(body) {
			out.WriteString("\tBox::pin(async {})\n")
		}
	}
	out.WriteString("}\n\n")
	return nil
}

func rustBodyReturnsValue(body string) bool {
	if strings.Contains(body, "return ") {
		return true
	}
	lines := strings.Split(body, "\n")
	for index := len(lines) - 1; index >= 0; index-- {
		line := lines[index]
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		return !strings.HasSuffix(line, ";")
	}
	return false
}

func rustInstanceParam(behavior Behavior) string {
	switch behavior.Kind {
	case BehaviorGuard, BehaviorTrigger:
		return "&HsmcInstance"
	default:
		return "&mut HsmcInstance"
	}
}

func rustParameterList(behavior Behavior) string {
	return fmt.Sprintf("(ctx: &hsm::Context, instance: %s, event: &hsm::Event)", rustInstanceParam(behavior))
}

func rustFullCodeReturnType(behavior Behavior) string {
	if ret := rustReturnType(behavior); ret != "" {
		return ret
	}
	return "()"
}

func rustReturnType(behavior Behavior) string {
	if behavior.TargetABI != nil && behavior.TargetABI.ReturnType != "" && behavior.TargetABI.ReturnType != "()" {
		return behavior.TargetABI.ReturnType
	}
	if behavior.Kind == BehaviorGuard {
		return "bool"
	}
	if rustBehaviorReturnsFuture(behavior) {
		return "Pin<Box<dyn Future<Output = ()> + Send>>"
	}
	return ""
}

func rustUntranslatedReturn(behavior Behavior) string {
	returnType := "()"
	if behavior.TargetABI != nil && behavior.TargetABI.ReturnType != "" {
		returnType = behavior.TargetABI.ReturnType
	}
	if returnType == "()" && rustBehaviorReturnsFuture(behavior) {
		returnType = "Pin<Box<dyn Future<Output = ()> + Send>>"
	}
	switch returnType {
	case "bool":
		return "false"
	case "Pin<Box<dyn Future<Output = ()> + Send>>":
		return "Box::pin(async {})"
	case "Duration", "std::time::Duration":
		return "Duration::from_secs(0)"
	case "std::time::SystemTime":
		return "std::time::UNIX_EPOCH"
	default:
		return ""
	}
}

func rustBehaviorReturnsFuture(behavior Behavior) bool {
	switch behavior.Kind {
	case BehaviorEntry, BehaviorExit, BehaviorActivity, BehaviorEffect, BehaviorOperation:
		return true
	default:
		return false
	}
}

func indentRust(body string) string {
	var out strings.Builder
	for _, line := range strings.Split(body, "\n") {
		if strings.TrimSpace(line) == "" {
			out.WriteString("\n")
			continue
		}
		out.WriteString("\t")
		out.WriteString(line)
		out.WriteString("\n")
	}
	return strings.TrimRight(out.String(), "\n")
}

func (backend RustBackend) emitModel(out *bytes.Buffer, model Model, behaviorNames map[string]string) {
	fmt.Fprintf(out, "pub fn %s_model() -> hsm::Model<HsmcInstance> {\n", snakeName(model.Name))
	fmt.Fprintf(out, "\thsm::define!(%s", strconv.Quote(model.Name))
	if model.Initializer != nil || len(model.Attributes) > 0 || len(model.Operations) > 0 || len(model.States) > 0 || len(model.Transitions) > 0 {
		out.WriteString(",\n")
	}
	if model.Initializer != nil {
		backend.emitInitial(out, "\t\t", *model.Initializer, behaviorNames)
	}
	for _, attribute := range model.Attributes {
		backend.emitAttribute(out, "\t\t", attribute)
	}
	for _, operation := range model.Operations {
		backend.emitOperation(out, "\t\t", operation, behaviorNames)
	}
	for _, state := range model.States {
		backend.emitState(out, "\t\t", state, behaviorNames)
	}
	for _, transition := range model.Transitions {
		backend.emitTransition(out, "\t\t", transition, behaviorNames)
	}
	out.WriteString("\t)\n")
	out.WriteString("}\n")
}

func (backend RustBackend) emitAttribute(out *bytes.Buffer, indent string, attribute Attribute) {
	if attribute.Type != "" {
		fmt.Fprintf(out, "%s// type: %s\n", indent, attribute.Type)
	}
	defaultValue, hasTargetDefault := rustAttributeTargetDefault(attribute)
	if attribute.HasDefault && !hasTargetDefault {
		fmt.Fprintf(out, "%s// default: %s\n", indent, attribute.Default)
	}
	if hasTargetDefault {
		fmt.Fprintf(out, "%shsm::attribute(%s, %s),\n", indent, strconv.Quote(attribute.Name), defaultValue)
		return
	}
	fmt.Fprintf(out, "%s// Attribute %q preserved for manual porting; hsm.rs attributes require a default value.\n", indent, attribute.Name)
}

func rustAttributeTargetDefault(attribute Attribute) (string, bool) {
	if !attribute.HasDefault {
		return "", false
	}
	value := strings.TrimSpace(attribute.Default)
	if value == "" {
		return "", false
	}
	if attribute.Language == "" || attribute.Language == LanguageRust {
		return value, true
	}
	if converted, ok := rustPortableAttributeLiteral(value); ok {
		return converted, true
	}
	return "", false
}

func rustPortableAttributeLiteral(value string) (string, bool) {
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

func (backend RustBackend) emitOperation(out *bytes.Buffer, indent string, operation Operation, behaviorNames map[string]string) {
	if operation.Behavior == nil {
		return
	}
	name := behaviorNames[operation.Behavior.ID]
	if name == "" {
		name = snakeName(operation.Behavior.ID)
	}
	fmt.Fprintf(out, "%shsm::operation(%s, %s),\n", indent, strconv.Quote(operation.Name), name)
}

func (backend RustBackend) emitState(out *bytes.Buffer, indent string, state State, behaviorNames map[string]string) {
	constructor := "state"
	switch state.Kind {
	case StateKindFinal:
		constructor = "final_state"
	case StateKindChoice:
		constructor = "choice"
	case StateKindShallowHistory:
		constructor = "shallow_history"
	case StateKindDeepHistory:
		constructor = "deep_history"
	}
	hasParts := state.Initializer != nil || len(state.Behaviors) > 0 || len(state.Defers) > 0 || len(state.States) > 0 || len(state.Transitions) > 0
	fmt.Fprintf(out, "%shsm::%s!(%s", indent, constructor, strconv.Quote(state.Name))
	if hasParts {
		out.WriteString(",\n")
	}
	if state.Initializer != nil {
		backend.emitInitial(out, indent+"\t", *state.Initializer, behaviorNames)
	}
	for _, behavior := range state.Behaviors {
		backend.emitBehaviorPartial(out, indent+"\t", behavior, behaviorNames)
	}
	for _, eventName := range state.Defers {
		fmt.Fprintf(out, "%shsm::defer!(%s),\n", indent+"\t", strconv.Quote(eventName))
	}
	for _, child := range state.States {
		backend.emitState(out, indent+"\t", child, behaviorNames)
	}
	for _, transition := range state.Transitions {
		backend.emitTransition(out, indent+"\t", transition, behaviorNames)
	}
	if hasParts {
		fmt.Fprintf(out, "%s),\n", indent)
		return
	}
	out.WriteString("),\n")
}

func (backend RustBackend) emitInitial(out *bytes.Buffer, indent string, initial Initial, behaviorNames map[string]string) {
	out.WriteString(indent + "hsm::initial!(")
	if initial.Target != "" {
		fmt.Fprintf(out, "hsm::target!(%s)", strconv.Quote(initial.Target))
	}
	for index, effect := range initial.Effects {
		if initial.Target != "" || index > 0 {
			out.WriteString(", ")
		}
		backend.emitBehaviorPartialInline(out, effect, behaviorNames)
	}
	out.WriteString("),\n")
}

func (backend RustBackend) emitTransition(out *bytes.Buffer, indent string, transition Transition, behaviorNames map[string]string) {
	out.WriteString(indent + "hsm::transition!(")
	var parts []string
	if transition.Source != "" {
		parts = append(parts, fmt.Sprintf("hsm::source!(%s)", strconv.Quote(transition.Source)))
	}
	if transition.Trigger != nil {
		parts = append(parts, rustTriggerValue(*transition.Trigger, behaviorNames))
	}
	if transition.Guard != nil {
		parts = append(parts, rustBehaviorPartialValue(*transition.Guard, behaviorNames))
	}
	if transition.Target != "" {
		parts = append(parts, fmt.Sprintf("hsm::target!(%s)", strconv.Quote(transition.Target)))
	}
	for _, effect := range transition.Effects {
		parts = append(parts, rustBehaviorPartialValue(effect, behaviorNames))
	}
	out.WriteString(strings.Join(parts, ", "))
	out.WriteString("),\n")
}

func rustTriggerValue(trigger Trigger, behaviorNames map[string]string) string {
	switch trigger.Kind {
	case TriggerOn:
		return "hsm::on!(" + rustTriggerValues(trigger) + ")"
	case TriggerOnSet:
		return fmt.Sprintf("hsm::on_set(%s)", strconv.Quote(trigger.Value))
	case TriggerOnCall:
		return fmt.Sprintf("hsm::on_call(%s)", strconv.Quote(trigger.Value))
	case TriggerAfter:
		return "hsm::after!(" + rustExpressionValue(trigger, behaviorNames) + ")"
	case TriggerEvery:
		return "hsm::every!(" + rustExpressionValue(trigger, behaviorNames) + ")"
	case TriggerAt:
		return "hsm::at!(" + rustExpressionValue(trigger, behaviorNames) + ")"
	case TriggerWhen:
		if trigger.IsString && trigger.Value != "" && trigger.Expr == nil {
			return fmt.Sprintf("hsm::when!(%s)", strconv.Quote(trigger.Value))
		}
		value := "hsmc_true_guard"
		if trigger.Expr != nil {
			if name := behaviorNames[trigger.Expr.ID]; name != "" {
				value = name
			}
		}
		return "/* when trigger lowered to guard for hsm.rs */ hsm::guard!(" + value + ")"
	default:
		return ""
	}
}

func rustTriggerValues(trigger Trigger) string {
	values := trigger.Values
	if len(values) == 0 {
		values = []TriggerValue{{Value: trigger.Value, IsString: trigger.IsString, Language: trigger.Language}}
	}
	rendered := make([]string, 0, len(values))
	for _, value := range values {
		text := value.Value
		if value.EventSymbol != "" {
			text = eventSymbolFor(LanguageRust, value.EventSymbol)
		} else if value.IsString {
			text = strconv.Quote(text)
		}
		rendered = append(rendered, text)
	}
	return strings.Join(rendered, ", ")
}

func (backend RustBackend) emitBehaviorPartial(out *bytes.Buffer, indent string, ref BehaviorRef, behaviorNames map[string]string) {
	fmt.Fprintf(out, "%s%s,\n", indent, rustBehaviorPartialValue(ref, behaviorNames))
}

func (backend RustBackend) emitBehaviorPartialInline(out *bytes.Buffer, ref BehaviorRef, behaviorNames map[string]string) {
	out.WriteString(rustBehaviorPartialValue(ref, behaviorNames))
}

func rustBehaviorPartialValue(ref BehaviorRef, behaviorNames map[string]string) string {
	if ref.Operation != "" {
		operation := strconv.Quote(ref.Operation)
		switch ref.Kind {
		case BehaviorEntry:
			return fmt.Sprintf("hsm::entry_operation(%s)", operation)
		case BehaviorExit:
			return fmt.Sprintf("hsm::exit_operation(%s)", operation)
		case BehaviorActivity:
			return fmt.Sprintf("hsm::activity_operation(%s)", operation)
		case BehaviorGuard:
			return fmt.Sprintf("hsm::guard_operation_ref(%s)", operation)
		case BehaviorEffect:
			return fmt.Sprintf("hsm::effect_operation(%s)", operation)
		}
	}
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
	return fmt.Sprintf("hsm::%s!(%s)", constructor, name)
}

func rustExpressionValue(trigger Trigger, behaviorNames map[string]string) string {
	if trigger.Expr != nil {
		if name := behaviorNames[trigger.Expr.ID]; name != "" {
			return name
		}
	}
	if trigger.IsString {
		return strconv.Quote(trigger.Value)
	}
	if trigger.Language == "" || trigger.Language == LanguageRust {
		return trigger.Value
	}
	if trigger.Kind == TriggerAt {
		return fmt.Sprintf("/* Original %s %s trigger expression preserved for manual porting:\n%s\n*/ hsmc_unix_epoch", trigger.Language, trigger.Kind, rustBlockCommentLines(trigger.Value))
	}
	return fmt.Sprintf("/* Original %s %s trigger expression preserved for manual porting:\n%s\n*/ hsmc_zero_duration", trigger.Language, trigger.Kind, rustBlockCommentLines(trigger.Value))
}

func rustBlockCommentLines(text string) string {
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		line = strings.ReplaceAll(line, "*/", "* /")
		if strings.TrimSpace(line) == "" {
			lines = append(lines, " *")
			continue
		}
		lines = append(lines, " * "+line)
	}
	if len(lines) == 0 {
		return " * No source trigger expression was captured."
	}
	return strings.Join(lines, "\n")
}
