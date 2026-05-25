package hsmc

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
)

type CSharpBackend struct{}

func NewCSharpBackend() CSharpBackend {
	return CSharpBackend{}
}

func (CSharpBackend) Language() Language { return LanguageCSharp }

func (backend CSharpBackend) Emit(_ context.Context, program *Program) ([]byte, error) {
	var out bytes.Buffer
	backend.emitImports(&out, program)
	out.WriteString("public static class GeneratedHsm\n{\n")
	for _, event := range program.Events {
		backend.emitEvent(&out, event)
	}
	for _, global := range program.Globals {
		if global.Language != "" && global.Language != LanguageCSharp {
			writeCommentedGlobalSource(&out, "\t", "//", global, LanguageCSharp)
			continue
		}
		code := strings.TrimSpace(global.Code)
		if code == "" {
			continue
		}
		out.WriteString(indentCSharp(code))
		out.WriteString("\n\n")
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
	return []byte(removeCSharpTrailingCommas(out.String())), nil
}

func (backend CSharpBackend) emitImports(out *bytes.Buffer, program *Program) {
	out.WriteString("using Stateforward.Hsm;\n")
	for _, imp := range collectRenderableImports(program, LanguageCSharp, compilerOwnedImports(LanguageCSharp)...) {
		fmt.Fprintln(out, formatCSharpImport(imp))
	}
	out.WriteString("\n")
}

func (backend CSharpBackend) emitEvent(out *bytes.Buffer, event EventDecl) {
	fmt.Fprintf(out, "\tprivate static readonly Event %s = new Event(%s);\n\n", eventSymbolFor(LanguageCSharp, event.Symbol), strconv.Quote(event.Name))
}

func (backend CSharpBackend) emitBehavior(out *bytes.Buffer, name string, behavior Behavior) error {
	if (behavior.TargetLanguage == LanguageCSharp || behavior.SourceLanguage == LanguageCSharp) && strings.TrimSpace(behavior.Body) == "" {
		code := strings.TrimSpace(behavior.Code)
		if code != "" {
			if declaredName, ok := csharpMethodDeclarationName(code); ok {
				if declaredName != name {
					return fmt.Errorf("csharp behavior full code declares %q, want compiler-owned name %q", declaredName, name)
				}
				if returnType, _, parameters, ok := cLikeMethodDeclarationShape(code); ok {
					if wantReturnType := csharpReturnType(behavior); returnType != wantReturnType {
						return fmt.Errorf("csharp behavior full code declares return type %q, want compiler-owned return type %q", returnType, wantReturnType)
					}
					if wantParameters := csharpParameterList(behavior); parameters != wantParameters {
						return fmt.Errorf("csharp behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
					}
				}
				out.WriteString(indentCSharp(code))
				out.WriteString("\n\n")
				return nil
			}
			if declaredName, ok := cLikeFieldDeclarationName(code); ok {
				if declaredName != name {
					return fmt.Errorf("csharp behavior full code declares %q, want compiler-owned name %q", declaredName, name)
				}
				if parameters, ok := csharpLambdaParameterNames(csharpFieldInitializer(code)); ok {
					if err := validateCSharpLambdaParameters(behavior, parameters); err != nil {
						return err
					}
				} else {
					return fmt.Errorf("csharp behavior full code for %q must be a compiler-owned callable declaration", name)
				}
				out.WriteString(indentCSharp(ensureTrailingSemicolon(code)))
				out.WriteString("\n\n")
				return nil
			}
			if declaredName, ok := csharpNonCallableDeclarationName(code); ok {
				if declaredName != name {
					return fmt.Errorf("csharp behavior full code declares %q, want compiler-owned name %q", declaredName, name)
				}
				return fmt.Errorf("csharp behavior full code for %q must be a compiler-owned callable declaration", name)
			}
			parameters, ok := csharpLambdaParameterNames(code)
			if !ok {
				if behavior.TargetLanguage == LanguageCSharp {
					return fmt.Errorf("csharp behavior full code for %q must be a compiler-owned callable expression", name)
				}
				code = strings.TrimSuffix(code, ";")
				fmt.Fprintf(out, "\tprivate static readonly %s %s = %s;\n\n", csharpDelegateType(behavior), name, code)
				return nil
			}
			if err := validateCSharpLambdaParameters(behavior, parameters); err != nil {
				return err
			}
			code = strings.TrimSuffix(code, ";")
			fmt.Fprintf(out, "\tprivate static readonly %s %s = %s;\n\n", csharpDelegateType(behavior), name, code)
			return nil
		}
	}
	body := ""
	if behavior.TargetLanguage == LanguageCSharp || behavior.SourceLanguage == LanguageCSharp {
		body = strings.TrimSpace(behavior.Body)
	}
	fmt.Fprintf(out, "\tprivate static %s %s%s\n\t{\n", csharpReturnType(behavior), name, csharpParameterList(behavior))
	if body == "" {
		writeCommentedBehaviorSource(out, "\t\t", "//", behavior)
		if ret := csharpUntranslatedReturn(behavior); ret != "" {
			fmt.Fprintf(out, "\t\t%s\n", ret)
		}
	} else {
		out.WriteString(indentCSharp(body, "\t\t"))
		out.WriteString("\n")
		if behavior.Kind == BehaviorGuard && !strings.Contains(body, "return") {
			out.WriteString("\t\treturn false;\n")
		}
	}
	out.WriteString("\t}\n\n")
	return nil
}

func csharpNonCallableDeclarationName(code string) (string, bool) {
	if name, ok := csharpDelegateDeclarationName(code); ok {
		return name, true
	}
	if name, ok := cLikeTypeDeclarationName(code); ok {
		return name, true
	}
	return "", false
}

func csharpDelegateType(behavior Behavior) string {
	if behavior.Kind == BehaviorGuard {
		return "Expression<Instance>"
	}
	if behavior.Kind == BehaviorTrigger {
		switch behavior.TriggerKind {
		case TriggerAfter, TriggerEvery:
			return "DurationProvider<Instance>"
		case TriggerAt:
			return "TimeProvider<Instance>"
		case TriggerWhen:
			return "ConditionChannel<Instance>"
		}
	}
	return "Operation<Instance>"
}

func csharpReturnType(behavior Behavior) string {
	if behavior.TargetABI != nil && behavior.TargetABI.ReturnType != "" {
		return behavior.TargetABI.ReturnType
	}
	if behavior.Kind == BehaviorGuard {
		return "bool"
	}
	return "void"
}

func csharpParameterList(behavior Behavior) string {
	if behavior.Kind == BehaviorTrigger && behavior.TriggerKind == TriggerWhen {
		return "(Context ctx, Instance instance, Event @event, System.Threading.CancellationToken cancellationToken)"
	}
	return "(Context ctx, Instance instance, Event @event)"
}

func csharpCallableParameterNames(behavior Behavior) []string {
	if behavior.Kind == BehaviorTrigger && behavior.TriggerKind == TriggerWhen {
		return []string{"ctx", "instance", "@event", "cancellationToken"}
	}
	return []string{"ctx", "instance", "@event"}
}

func validateCSharpLambdaParameters(behavior Behavior, parameters []string) error {
	wantParameters := csharpCallableParameterNames(behavior)
	if strings.Join(parameters, ", ") != strings.Join(wantParameters, ", ") {
		return fmt.Errorf("csharp behavior full code declares lambda parameters %q, want compiler-owned parameters %q", "("+strings.Join(parameters, ", ")+")", "("+strings.Join(wantParameters, ", ")+")")
	}
	return nil
}

func csharpFieldInitializer(code string) string {
	if eq := firstTopLevelByte(code, '='); eq >= 0 {
		return strings.TrimSpace(code[eq+1:])
	}
	return code
}

func csharpLambdaParameterNames(code string) ([]string, bool) {
	code = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(code), ";"))
	if strings.HasPrefix(code, "async ") {
		code = strings.TrimSpace(strings.TrimPrefix(code, "async "))
	}
	arrow := strings.Index(code, "=>")
	if arrow < 0 {
		return nil, false
	}
	prefix := strings.TrimSpace(code[:arrow])
	if prefix == "" {
		return nil, false
	}
	if strings.HasPrefix(prefix, "(") {
		closeParen := matchingCloseParen(prefix)
		if closeParen < 0 || strings.TrimSpace(prefix[closeParen+1:]) != "" {
			return nil, false
		}
		inside := strings.TrimSpace(prefix[1:closeParen])
		if inside == "" {
			return []string{}, true
		}
		parts := strings.Split(inside, ",")
		names := make([]string, 0, len(parts))
		for _, part := range parts {
			name := csharpParameterName(part)
			if name == "" {
				return nil, false
			}
			names = append(names, name)
		}
		return names, true
	}
	name := csharpParameterName(prefix)
	if name == "" {
		return nil, false
	}
	return []string{name}, true
}

func csharpParameterName(parameter string) string {
	fields := strings.Fields(strings.TrimSpace(parameter))
	if len(fields) == 0 {
		return ""
	}
	name := fields[len(fields)-1]
	name = strings.TrimPrefix(name, "this ")
	if !isSimpleIdentifier(strings.TrimPrefix(name, "@")) {
		return ""
	}
	return name
}

func (backend CSharpBackend) emitModel(out *bytes.Buffer, model Model, behaviorNames map[string]string) {
	fmt.Fprintf(out, "\tpublic static readonly Model %sModel = Hsm.Define(\n", exportName(model.Name))
	fmt.Fprintf(out, "\t\t%s", strconv.Quote(model.Name))
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
	out.WriteString("\t);\n\n")
}

func (backend CSharpBackend) emitAttribute(out *bytes.Buffer, indent string, attribute Attribute) {
	typeName := "object"
	if attributeTypeIsTargetExpression(attribute, LanguageCSharp) {
		typeName = csharpAttributeType(attribute.Type)
	} else if attribute.Type != "" {
		fmt.Fprintf(out, "%s// Type: %s\n", indent, attribute.Type)
	}
	if defaultValue, ok := csharpAttributeTargetDefault(attribute); ok {
		fmt.Fprintf(out, "%sHsm.Attribute<%s>(%s, %s),\n", indent, typeName, strconv.Quote(attribute.Name), defaultValue)
		return
	}
	if attribute.HasDefault {
		fmt.Fprintf(out, "%s// Default: %s\n", indent, attribute.Default)
	}
	fmt.Fprintf(out, "%sHsm.Attribute<%s>(%s),\n", indent, typeName, strconv.Quote(attribute.Name))
}

func csharpAttributeTargetDefault(attribute Attribute) (string, bool) {
	if !attribute.HasDefault {
		return "", false
	}
	value := strings.TrimSpace(attribute.Default)
	if value == "" {
		return "", false
	}
	if attribute.Language == "" || attribute.Language == LanguageCSharp {
		return value, true
	}
	switch value {
	case "True":
		return "true", true
	case "False":
		return "false", true
	case "None", "nil", "null":
		return "null", true
	}
	if value == "true" || value == "false" {
		return value, true
	}
	if csharpNumericLiteral(value) {
		return value, true
	}
	if _, err := strconv.Unquote(value); err == nil {
		if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
			return value, true
		}
	}
	return "", false
}

func csharpNumericLiteral(value string) bool {
	lower := strings.ToLower(value)
	if strings.Contains(lower, "nan") || strings.Contains(lower, "inf") {
		return false
	}
	_, err := strconv.ParseFloat(value, 64)
	return err == nil
}

func csharpAttributeType(typeName string) string {
	typeName = strings.TrimSpace(typeName)
	if strings.HasPrefix(typeName, "typeof(") && strings.HasSuffix(typeName, ")") {
		typeName = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(typeName, "typeof("), ")"))
	}
	if typeName == "" {
		return "object"
	}
	return typeName
}

func (backend CSharpBackend) emitOperation(out *bytes.Buffer, indent string, operation Operation, behaviorNames map[string]string) {
	if operation.Behavior == nil {
		return
	}
	name := behaviorNames[operation.Behavior.ID]
	if name == "" {
		name = exportName(operation.Behavior.ID)
	}
	fmt.Fprintf(out, "%sHsm.Operation(%s, new Operation<Instance>(%s)),\n", indent, strconv.Quote(operation.Name), name)
}

func (backend CSharpBackend) emitState(out *bytes.Buffer, indent string, state State, behaviorNames map[string]string) {
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
	if state.Initializer == nil && len(state.Behaviors) == 0 && len(state.Defers) == 0 && len(state.States) == 0 && len(state.Transitions) == 0 {
		fmt.Fprintf(out, "%sHsm.%s(%s),\n", indent, constructor, strconv.Quote(state.Name))
		return
	}
	fmt.Fprintf(out, "%sHsm.%s(\n", indent, constructor)
	fmt.Fprintf(out, "%s\t%s", indent, strconv.Quote(state.Name))
	if state.Initializer != nil || len(state.Behaviors) > 0 || len(state.Defers) > 0 || len(state.States) > 0 || len(state.Transitions) > 0 {
		out.WriteString(",\n")
	}
	if state.Initializer != nil {
		backend.emitInitial(out, indent+"\t", *state.Initializer, behaviorNames)
	}
	for _, behavior := range state.Behaviors {
		backend.emitBehaviorPartial(out, indent+"\t", behavior, behaviorNames)
	}
	for _, eventName := range state.Defers {
		fmt.Fprintf(out, "%sHsm.Defer(%s),\n", indent+"\t", strconv.Quote(eventName))
	}
	for _, child := range state.States {
		backend.emitState(out, indent+"\t", child, behaviorNames)
	}
	for _, transition := range state.Transitions {
		backend.emitTransition(out, indent+"\t", transition, behaviorNames)
	}
	fmt.Fprintf(out, "%s),\n", indent)
}

func (backend CSharpBackend) emitInitial(out *bytes.Buffer, indent string, initial Initial, behaviorNames map[string]string) {
	fmt.Fprintf(out, "%sHsm.Initial(\n", indent)
	if initial.Target != "" {
		fmt.Fprintf(out, "%s\tHsm.Target(%s),\n", indent, strconv.Quote(initial.Target))
	}
	for _, effect := range initial.Effects {
		backend.emitBehaviorPartial(out, indent+"\t", effect, behaviorNames)
	}
	fmt.Fprintf(out, "%s),\n", indent)
}

func (backend CSharpBackend) emitTransition(out *bytes.Buffer, indent string, transition Transition, behaviorNames map[string]string) {
	fmt.Fprintf(out, "%sHsm.Transition(\n", indent)
	if transition.Source != "" {
		fmt.Fprintf(out, "%s\tHsm.Source(%s),\n", indent, strconv.Quote(transition.Source))
	}
	if transition.Trigger != nil {
		backend.emitTrigger(out, indent+"\t", *transition.Trigger, behaviorNames)
	}
	if transition.Guard != nil {
		backend.emitBehaviorPartial(out, indent+"\t", *transition.Guard, behaviorNames)
	}
	if transition.Target != "" {
		fmt.Fprintf(out, "%s\tHsm.Target(%s),\n", indent, strconv.Quote(transition.Target))
	}
	for _, effect := range transition.Effects {
		backend.emitBehaviorPartial(out, indent+"\t", effect, behaviorNames)
	}
	fmt.Fprintf(out, "%s),\n", indent)
}

func (backend CSharpBackend) emitTrigger(out *bytes.Buffer, indent string, trigger Trigger, behaviorNames map[string]string) {
	switch trigger.Kind {
	case TriggerOn:
		for _, value := range csharpTriggerValues(trigger) {
			fmt.Fprintf(out, "%sHsm.On(%s),\n", indent, value)
		}
	case TriggerOnSet:
		fmt.Fprintf(out, "%sHsm.OnSet(%s),\n", indent, strconv.Quote(trigger.Value))
	case TriggerOnCall:
		fmt.Fprintf(out, "%sHsm.OnCall(%s),\n", indent, strconv.Quote(trigger.Value))
	case TriggerAfter:
		fmt.Fprintf(out, "%sHsm.After<Instance>(%s),\n", indent, csharpExpressionValue(trigger, behaviorNames))
	case TriggerEvery:
		fmt.Fprintf(out, "%sHsm.Every<Instance>(%s),\n", indent, csharpExpressionValue(trigger, behaviorNames))
	case TriggerAt:
		fmt.Fprintf(out, "%sHsm.At<Instance>(%s),\n", indent, csharpExpressionValue(trigger, behaviorNames))
	case TriggerWhen:
		if trigger.IsString && trigger.Value != "" && trigger.Expr == nil {
			fmt.Fprintf(out, "%sHsm.OnSet(%s),\n", indent, strconv.Quote(trigger.Value))
			return
		}
		fmt.Fprintf(out, "%sHsm.When<Instance>(%s),\n", indent, csharpExpressionValue(trigger, behaviorNames))
	}
}

func csharpTriggerValues(trigger Trigger) []string {
	values := trigger.Values
	if len(values) == 0 {
		values = []TriggerValue{{Value: trigger.Value, IsString: trigger.IsString, Language: trigger.Language}}
	}
	rendered := make([]string, 0, len(values))
	for _, value := range values {
		text := value.Value
		if value.EventSymbol != "" {
			text = eventSymbolFor(LanguageCSharp, value.EventSymbol)
		} else if value.IsString {
			text = strconv.Quote(text)
		}
		rendered = append(rendered, text)
	}
	return rendered
}

func (backend CSharpBackend) emitBehaviorPartial(out *bytes.Buffer, indent string, ref BehaviorRef, behaviorNames map[string]string) {
	fmt.Fprintf(out, "%s%s,\n", indent, csharpBehaviorPartialValue(ref, behaviorNames))
}

func csharpBehaviorPartialValue(ref BehaviorRef, behaviorNames map[string]string) string {
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
		operation := strconv.Quote(ref.Operation)
		if ref.Kind == BehaviorGuard {
			return fmt.Sprintf("Hsm.Guard<Instance>((ctx, instance, @event) => Hsm.Call(ctx, instance, %s) is true)", operation)
		}
		return fmt.Sprintf("Hsm.%s<Instance>((ctx, instance, @event) => Hsm.Call(ctx, instance, %s))", constructor, operation)
	}
	if name == "" {
		name = exportName(ref.ID)
	}
	return fmt.Sprintf("Hsm.%s<Instance>(%s)", constructor, name)
}

func csharpExpressionValue(trigger Trigger, behaviorNames map[string]string) string {
	if trigger.Expr != nil {
		if name := behaviorNames[trigger.Expr.ID]; name != "" {
			return name
		}
	}
	if trigger.IsString {
		return strconv.Quote(trigger.Value)
	}
	if trigger.Language == "" || trigger.Language == LanguageCSharp {
		return trigger.Value
	}
	parameters := "(_, _, _)"
	if trigger.Kind == TriggerWhen {
		parameters = "(_, _, _, _)"
	}
	return fmt.Sprintf("%s => {\n\t// Original %s %s trigger expression preserved for manual porting:\n%s\n\t%s\n}", parameters, trigger.Language, trigger.Kind, commentLines("\t", "//", trigger.Value), csharpTriggerDefaultReturn(trigger.Kind))
}

func csharpTriggerDefaultReturn(kind TriggerKind) string {
	switch kind {
	case TriggerAt:
		return "return System.DateTimeOffset.UnixEpoch;"
	case TriggerWhen:
		return "return System.Threading.Tasks.Task.CompletedTask;"
	default:
		return "return System.TimeSpan.Zero;"
	}
}

func indentCSharp(body string, indents ...string) string {
	indent := "\t"
	if len(indents) > 0 {
		indent = indents[0]
	}
	lines := strings.Split(body, "\n")
	for index, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[index] = ""
			continue
		}
		lines[index] = indent + line
	}
	return strings.Join(lines, "\n")
}

func removeCSharpTrailingCommas(source string) string {
	lines := strings.Split(source, "\n")
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasSuffix(trimmed, ",") {
			continue
		}
		for next := index + 1; next < len(lines); next++ {
			nextTrimmed := strings.TrimSpace(lines[next])
			if nextTrimmed == "" {
				continue
			}
			if nextTrimmed == ")" || nextTrimmed == ");" || nextTrimmed == ")," {
				lines[index] = strings.TrimSuffix(line, ",")
			}
			break
		}
	}
	return strings.Join(lines, "\n")
}
