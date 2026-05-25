package hsmc

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

type JavaScriptBackend struct{}

func NewJavaScriptBackend() JavaScriptBackend {
	return JavaScriptBackend{}
}

func (JavaScriptBackend) Language() Language { return LanguageJS }

func (backend JavaScriptBackend) Emit(_ context.Context, program *Program) ([]byte, error) {
	var out bytes.Buffer
	backend.emitImports(&out, program)
	for _, event := range program.Events {
		backend.emitEvent(&out, event)
	}
	for _, global := range program.Globals {
		if global.Language != "" && global.Language != LanguageJS {
			writeCommentedGlobalSource(&out, "", "//", global, LanguageJS)
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

func (backend JavaScriptBackend) emitImports(out *bytes.Buffer, program *Program) {
	out.WriteString(`import * as hsm from "@stateforward/hsm";` + "\n")
	for _, imp := range collectRenderableImports(program, LanguageJS, compilerOwnedImports(LanguageJS)...) {
		fmt.Fprintln(out, formatESImport(imp))
	}
	out.WriteString("\n")
}

func (backend JavaScriptBackend) emitEvent(out *bytes.Buffer, event EventDecl) {
	fmt.Fprintf(out, "const %s = hsm.Event(%s", eventSymbolFor(LanguageJS, event.Symbol), quoteTS(event.Name))
	if event.Schema != "" && event.Language == LanguageJS {
		fmt.Fprintf(out, ", %s", event.Schema)
	}
	out.WriteString(");\n\n")
}

func (backend JavaScriptBackend) emitBehavior(out *bytes.Buffer, name string, behavior Behavior) error {
	if (behavior.SourceLanguage == LanguageJS || behavior.TargetLanguage == LanguageJS) && strings.TrimSpace(behavior.Body) == "" {
		code := strings.TrimSpace(behavior.Code)
		if code != "" {
			if declaredName, ok := esTopLevelDeclarationName(code); ok {
				if declaredName != name {
					return fmt.Errorf("javascript behavior full code declares %q, want compiler-owned name %q", declaredName, name)
				}
				callable := false
				if _, parameters, _, ok := esFunctionDeclarationShape(code); ok {
					callable = true
					if wantParameters := jsParameterList(); parameters != wantParameters {
						return fmt.Errorf("javascript behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
					}
				}
				if parameters, _, ok := esFunctionExpressionShape(code); ok {
					callable = true
					if wantParameters := jsParameterList(); parameters != wantParameters {
						return fmt.Errorf("javascript behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
					}
				}
				if !callable {
					return fmt.Errorf("javascript behavior full code for %q must be a compiler-owned callable declaration", name)
				}
				out.WriteString(ensureTrailingSemicolonForESDeclaration(code))
				out.WriteString("\n\n")
				return nil
			}
			parameters, _, ok := esFunctionExpressionShape(code)
			if !ok {
				if behavior.TargetLanguage == LanguageJS {
					return fmt.Errorf("javascript behavior full code for %q must be a compiler-owned callable expression", name)
				}
				fmt.Fprintf(out, "const %s = %s;\n\n", name, code)
				return nil
			}
			if wantParameters := jsParameterList(); parameters != wantParameters {
				return fmt.Errorf("javascript behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
			}
			fmt.Fprintf(out, "const %s = %s;\n\n", name, code)
			return nil
		}
	}
	body := ""
	if behavior.TargetLanguage == LanguageJS || behavior.SourceLanguage == LanguageJS {
		body = strings.TrimSpace(behavior.Body)
	}
	fmt.Fprintf(out, "function %s%s {\n", name, jsParameterList())
	if body == "" {
		writeCommentedBehaviorSource(out, "\t", "//", behavior)
		if ret := jsUntranslatedReturn(behavior); ret != "" {
			fmt.Fprintf(out, "\t%s\n", ret)
		}
	} else {
		out.WriteString(indentTS(body))
		out.WriteString("\n")
		if behavior.Kind == BehaviorGuard && !strings.Contains(body, "return") {
			out.WriteString("\treturn false;\n")
		}
	}
	out.WriteString("}\n\n")
	return nil
}

func jsParameterList() string {
	return "(ctx, instance, event)"
}

func (backend JavaScriptBackend) emitModel(out *bytes.Buffer, model Model, behaviorNames map[string]string) {
	fmt.Fprintf(out, "export const %sModel = hsm.Define(\n", exportName(model.Name))
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

func (backend JavaScriptBackend) emitAttribute(out *bytes.Buffer, indent string, attribute Attribute) {
	if attribute.Type != "" {
		fmt.Fprintf(out, "%s// Type: %s\n", indent, attribute.Type)
	}
	if defaultValue, ok := esAttributeTargetDefault(attribute, LanguageJS); ok {
		fmt.Fprintf(out, "%shsm.Attribute(%s, %s),\n", indent, quoteTS(attribute.Name), defaultValue)
		return
	}
	if attribute.HasDefault {
		fmt.Fprintf(out, "%s// Default: %s\n", indent, attribute.Default)
	}
	fmt.Fprintf(out, "%shsm.Attribute(%s),\n", indent, quoteTS(attribute.Name))
}

func (backend JavaScriptBackend) emitOperation(out *bytes.Buffer, indent string, operation Operation, behaviorNames map[string]string) {
	if operation.Behavior == nil {
		return
	}
	name := behaviorNames[operation.Behavior.ID]
	if name == "" {
		name = operation.Behavior.ID
	}
	fmt.Fprintf(out, "%shsm.Operation(%s, %s),\n", indent, quoteTS(operation.Name), name)
}

func (backend JavaScriptBackend) emitState(out *bytes.Buffer, indent string, state State, behaviorNames map[string]string) {
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
	fmt.Fprintf(out, "%s),\n", indent)
}

func (backend JavaScriptBackend) emitInitial(out *bytes.Buffer, indent string, initial Initial, behaviorNames map[string]string) {
	fmt.Fprintf(out, "%shsm.Initial(\n", indent)
	if initial.Target != "" {
		fmt.Fprintf(out, "%s\thsm.Target(%s),\n", indent, quoteTS(initial.Target))
	}
	for _, effect := range initial.Effects {
		backend.emitBehaviorPartial(out, indent+"\t", effect, behaviorNames)
	}
	fmt.Fprintf(out, "%s),\n", indent)
}

func (backend JavaScriptBackend) emitTransition(out *bytes.Buffer, indent string, transition Transition, behaviorNames map[string]string) {
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
	fmt.Fprintf(out, "%s),\n", indent)
}

func (backend JavaScriptBackend) emitTrigger(out *bytes.Buffer, indent string, trigger Trigger, behaviorNames map[string]string) {
	switch trigger.Kind {
	case TriggerOn:
		fmt.Fprintf(out, "%shsm.On(%s),\n", indent, jsTriggerValues(trigger))
	case TriggerOnSet:
		fmt.Fprintf(out, "%shsm.OnSet(%s),\n", indent, quoteTS(trigger.Value))
	case TriggerOnCall:
		fmt.Fprintf(out, "%shsm.OnCall(%s),\n", indent, quoteTS(trigger.Value))
	case TriggerAfter:
		fmt.Fprintf(out, "%shsm.After(%s),\n", indent, jsExpressionValue(trigger, behaviorNames))
	case TriggerEvery:
		fmt.Fprintf(out, "%shsm.Every(%s),\n", indent, jsExpressionValue(trigger, behaviorNames))
	case TriggerAt:
		fmt.Fprintf(out, "%shsm.At(%s),\n", indent, jsExpressionValue(trigger, behaviorNames))
	case TriggerWhen:
		if trigger.IsString && trigger.Value != "" && trigger.Expr == nil {
			fmt.Fprintf(out, "%shsm.OnSet(%s),\n", indent, quoteTS(trigger.Value))
			return
		}
		fmt.Fprintf(out, "%shsm.When(%s),\n", indent, jsExpressionValue(trigger, behaviorNames))
	}
}

func jsTriggerValues(trigger Trigger) string {
	values := trigger.Values
	if len(values) == 0 {
		values = []TriggerValue{{Value: trigger.Value, IsString: trigger.IsString, Language: trigger.Language}}
	}
	rendered := make([]string, 0, len(values))
	for _, value := range values {
		text := value.Value
		if value.EventSymbol != "" {
			text = eventSymbolFor(LanguageJS, value.EventSymbol)
		} else if value.IsString {
			text = quoteTS(text)
		}
		rendered = append(rendered, text)
	}
	return strings.Join(rendered, ", ")
}

func (backend JavaScriptBackend) emitBehaviorPartial(out *bytes.Buffer, indent string, ref BehaviorRef, behaviorNames map[string]string) {
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

func jsExpressionValue(trigger Trigger, behaviorNames map[string]string) string {
	if trigger.IsString {
		return quoteTS(trigger.Value)
	}
	if trigger.Expr != nil {
		if name := behaviorNames[trigger.Expr.ID]; name != "" {
			return name
		}
	}
	if trigger.Language == "" || trigger.Language == LanguageJS {
		return trigger.Value
	}
	return fmt.Sprintf("() => {\n\t// Original %s %s trigger expression preserved for manual porting:\n%s\n\t%s\n}", trigger.Language, trigger.Kind, commentLines("\t", "//", trigger.Value), jsTriggerDefaultReturn(trigger.Kind))
}

func jsTriggerDefaultReturn(kind TriggerKind) string {
	switch kind {
	case TriggerAt:
		return "return new Date(0);"
	case TriggerAfter, TriggerEvery:
		return "return 0;"
	default:
		return "return undefined;"
	}
}
