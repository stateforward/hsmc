package hsmc

import (
	"bytes"
	"context"
	"fmt"
	"go/format"
	"strconv"
	"strings"
)

type GoBackend struct{}

func NewGoBackend() GoBackend {
	return GoBackend{}
}

func (GoBackend) Language() Language { return LanguageGo }

func (backend GoBackend) Emit(_ context.Context, program *Program) ([]byte, error) {
	var out bytes.Buffer
	packageName := program.PackageName
	if packageName == "" {
		packageName = "main"
	}
	fmt.Fprintf(&out, "package %s\n\n", packageName)
	backend.emitImports(&out, program)
	for _, event := range program.Events {
		backend.emitEvent(&out, event)
	}
	for _, global := range program.Globals {
		if global.Language != "" && global.Language != LanguageGo {
			writeCommentedGlobalSource(&out, "", "//", global, LanguageGo)
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
		name := exportName(behavior.ID)
		behaviorNames[behavior.ID] = name
		if err := backend.emitBehavior(&out, name, behavior); err != nil {
			return nil, err
		}
	}
	for _, model := range program.Models {
		backend.emitModel(&out, model, behaviorNames)
	}
	formatted, err := format.Source(out.Bytes())
	if err != nil {
		return out.Bytes(), err
	}
	return formatted, nil
}

func (backend GoBackend) emitImports(out *bytes.Buffer, program *Program) {
	imports := map[string]Import{}
	addImport := func(imp Import) {
		imports[goImportKey(imp)] = imp
	}
	addRuntimeImport := func(imp Import) {
		for key, existing := range imports {
			if existing.Path == imp.Path {
				delete(imports, key)
			}
		}
		addImport(imp)
	}
	for _, imp := range program.Imports {
		addImport(imp)
	}
	for _, global := range program.Globals {
		for _, imp := range global.Imports {
			addImport(imp)
		}
	}
	for _, behavior := range program.Behaviors {
		for _, imp := range behavior.Imports {
			addImport(imp)
		}
	}
	addRuntimeImport(Import{Path: "github.com/stateforward/hsm.go", Alias: "hsm"})
	if behaviorNeedsContext(program) {
		addRuntimeImport(Import{Path: "context"})
	}
	if needsTime(program) {
		addRuntimeImport(Import{Path: "time"})
	}
	if len(imports) == 0 {
		return
	}
	out.WriteString("import (\n")
	keys := sortedGoImportKeys(imports)
	for _, key := range keys {
		imp := imports[key]
		if imp.Language != "" && imp.Language != LanguageGo {
			continue
		}
		if imp.Alias != "" {
			fmt.Fprintf(out, "\t%s %s\n", imp.Alias, strconv.Quote(imp.Path))
		} else {
			fmt.Fprintf(out, "\t%s\n", strconv.Quote(imp.Path))
		}
	}
	out.WriteString(")\n\n")
}

func (backend GoBackend) emitEvent(out *bytes.Buffer, event EventDecl) {
	fmt.Fprintf(out, "var %s = hsm.Event{Name: %s}\n\n", eventSymbolFor(LanguageGo, event.Symbol), strconv.Quote(event.Name))
}

func behaviorNeedsContext(program *Program) bool {
	for _, behavior := range program.Behaviors {
		if strings.Contains(behavior.Signature, "context.Context") || strings.Contains(behavior.Code, "context.Context") {
			return true
		}
		if behavior.TargetLanguage == LanguageGo && strings.TrimSpace(behavior.Body) != "" {
			return true
		}
		if behavior.SourceLanguage == LanguageGo || behavior.TargetLanguage == LanguageGo {
			if strings.TrimSpace(behavior.Code) == "" {
				return true
			}
			continue
		}
		if behavior.SourceLanguage != LanguageGo {
			return true
		}
	}
	for _, model := range program.Models {
		if modelNeedsForeignTriggerContext(model) {
			return true
		}
	}
	return false
}

func needsTime(program *Program) bool {
	for _, behavior := range program.Behaviors {
		if behaviorNeedsTime(behavior) {
			return true
		}
	}
	for _, model := range program.Models {
		if modelNeedsForeignTimeTrigger(model) {
			return true
		}
	}
	return false
}

func behaviorNeedsTime(behavior Behavior) bool {
	if behavior.TargetABI != nil && strings.Contains(behavior.TargetABI.ReturnType, "time.") {
		return true
	}
	if strings.Contains(behavior.Signature, "time.") || strings.Contains(behavior.Body, "time.") || strings.Contains(behavior.Code, "time.") {
		return true
	}
	return behavior.TargetLanguage == LanguageGo && behavior.Kind == BehaviorTrigger && behavior.TriggerKind == TriggerAt
}

func modelNeedsForeignTriggerContext(model Model) bool {
	for _, transition := range model.Transitions {
		if transitionHasForeignExpression(transition) {
			return true
		}
	}
	for _, state := range model.States {
		if stateNeedsForeignTriggerContext(state) {
			return true
		}
	}
	return false
}

func stateNeedsForeignTriggerContext(state State) bool {
	for _, transition := range state.Transitions {
		if transitionHasForeignExpression(transition) {
			return true
		}
	}
	for _, child := range state.States {
		if stateNeedsForeignTriggerContext(child) {
			return true
		}
	}
	return false
}

func modelNeedsForeignTimeTrigger(model Model) bool {
	for _, transition := range model.Transitions {
		if transitionHasForeignTimeExpression(transition) {
			return true
		}
	}
	for _, state := range model.States {
		if stateNeedsForeignTimeTrigger(state) {
			return true
		}
	}
	return false
}

func stateNeedsForeignTimeTrigger(state State) bool {
	for _, transition := range state.Transitions {
		if transitionHasForeignTimeExpression(transition) {
			return true
		}
	}
	for _, child := range state.States {
		if stateNeedsForeignTimeTrigger(child) {
			return true
		}
	}
	return false
}

func transitionHasForeignExpression(transition Transition) bool {
	return transition.Trigger != nil && !transition.Trigger.IsString && transition.Trigger.Language != "" && transition.Trigger.Language != LanguageGo
}

func transitionHasForeignTimeExpression(transition Transition) bool {
	if transition.Trigger == nil || transition.Trigger.IsString || transition.Trigger.Language == "" || transition.Trigger.Language == LanguageGo {
		return false
	}
	return transition.Trigger.Kind == TriggerAfter || transition.Trigger.Kind == TriggerEvery || transition.Trigger.Kind == TriggerAt
}

func goImportKey(imp Import) string {
	return imp.Path + "\x00" + imp.Alias
}

func sortedGoImportKeys(imports map[string]Import) []string {
	keys := make([]string, 0, len(imports))
	for key := range imports {
		keys = append(keys, key)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			left := imports[keys[i]]
			right := imports[keys[j]]
			if right.Path < left.Path || (right.Path == left.Path && right.Alias < left.Alias) {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

func (backend GoBackend) emitBehavior(out *bytes.Buffer, name string, behavior Behavior) error {
	if behavior.TargetLanguage == LanguageGo && strings.TrimSpace(behavior.Body) == "" {
		code := strings.TrimSpace(behavior.Code)
		if code != "" {
			if declaredName, ok := goTopLevelDeclarationName(code); ok {
				if declaredName != name {
					return fmt.Errorf("go behavior full code declares %q, want compiler-owned name %q", declaredName, name)
				}
				declarationSignature, hasDeclarationSignature := goFunctionDeclarationSignature(code, name)
				literalSignature, hasLiteralSignature := goFunctionLiteralSignature(code)
				if !hasDeclarationSignature && !hasLiteralSignature {
					return fmt.Errorf("go behavior full code for %q must be a compiler-owned callable declaration", name)
				}
				if hasDeclarationSignature && declarationSignature != goABISuffix(behavior) {
					return fmt.Errorf("go behavior full code declares signature %q, want compiler-owned signature %q", declarationSignature, goABISuffix(behavior))
				}
				if hasLiteralSignature && literalSignature != goABISuffix(behavior) {
					return fmt.Errorf("go behavior full code declares function literal signature %q, want compiler-owned signature %q", literalSignature, goABISuffix(behavior))
				}
				out.WriteString(code)
				if !strings.HasSuffix(code, "\n") {
					out.WriteString("\n")
				}
				out.WriteString("\n")
				return nil
			}
			if signature, ok := goFunctionLiteralSignature(code); ok && signature != goABISuffix(behavior) {
				return fmt.Errorf("go behavior full code declares function literal signature %q, want compiler-owned signature %q", signature, goABISuffix(behavior))
			}
			fmt.Fprintf(out, "var %s = %s\n\n", name, code)
			return nil
		}
	}
	if behavior.TargetLanguage == LanguageGo && strings.TrimSpace(behavior.Body) != "" {
		fmt.Fprintf(out, "func %s%s {\n", name, goABISuffix(behavior))
		out.WriteString(behavior.Body)
		out.WriteString("\n}\n\n")
		return nil
	}
	if behavior.Signature != "" && behavior.SourceLanguage == LanguageGo {
		signature := goBehaviorSignatureSuffix(behavior.Signature)
		fmt.Fprintf(out, "func %s%s {\n", name, signature)
		if strings.TrimSpace(behavior.Body) != "" {
			out.WriteString(behavior.Body)
			out.WriteString("\n")
		}
		out.WriteString("}\n\n")
		return nil
	}
	if behavior.SourceLanguage == LanguageGo || behavior.TargetLanguage == LanguageGo {
		code := strings.TrimSpace(behavior.Code)
		if code != "" {
			if declaredName, ok := goTopLevelDeclarationName(code); ok {
				if declaredName != name {
					return fmt.Errorf("go behavior full code declares %q, want compiler-owned name %q", declaredName, name)
				}
				declarationSignature, hasDeclarationSignature := goFunctionDeclarationSignature(code, name)
				literalSignature, hasLiteralSignature := goFunctionLiteralSignature(code)
				if !hasDeclarationSignature && !hasLiteralSignature {
					return fmt.Errorf("go behavior full code for %q must be a compiler-owned callable declaration", name)
				}
				if hasDeclarationSignature && declarationSignature != goABISuffix(behavior) {
					return fmt.Errorf("go behavior full code declares signature %q, want compiler-owned signature %q", declarationSignature, goABISuffix(behavior))
				}
				if hasLiteralSignature && literalSignature != goABISuffix(behavior) {
					return fmt.Errorf("go behavior full code declares function literal signature %q, want compiler-owned signature %q", literalSignature, goABISuffix(behavior))
				}
				out.WriteString(code)
				if !strings.HasSuffix(code, "\n") {
					out.WriteString("\n")
				}
				out.WriteString("\n")
				return nil
			}
			if signature, ok := goFunctionLiteralSignature(code); ok && signature != goABISuffix(behavior) {
				return fmt.Errorf("go behavior full code declares function literal signature %q, want compiler-owned signature %q", signature, goABISuffix(behavior))
			}
			fmt.Fprintf(out, "var %s = %s\n\n", name, code)
			return nil
		}
		fmt.Fprintf(out, "func %s%s {\n", name, goABISuffix(behavior))
		writeCommentedBehaviorSource(out, "\t", "//", behavior)
		if ret := goUntranslatedReturn(behavior); ret != "" {
			fmt.Fprintf(out, "\t%s\n", ret)
		}
		out.WriteString("}\n\n")
		return nil
	}
	if behavior.Kind == BehaviorGuard {
		fmt.Fprintf(out, "func %s%s {\n", name, goABISuffix(behavior))
		writeCommentedBehaviorSource(out, "\t", "//", behavior)
		if ret := goUntranslatedReturn(behavior); ret != "" {
			fmt.Fprintf(out, "\t%s\n", ret)
		}
		out.WriteString("}\n\n")
		return nil
	}
	fmt.Fprintf(out, "func %s%s {\n", name, goABISuffix(behavior))
	writeCommentedBehaviorSource(out, "\t", "//", behavior)
	if ret := goUntranslatedReturn(behavior); ret != "" {
		fmt.Fprintf(out, "\t%s\n", ret)
	}
	out.WriteString("}\n\n")
	return nil
}

func goBehaviorSignatureSuffix(signature string) string {
	signature = strings.TrimSpace(signature)
	if strings.HasPrefix(signature, "func ") {
		if start := strings.Index(signature, "("); start >= 0 {
			return strings.TrimSpace(signature[start:])
		}
	}
	return strings.TrimSpace(strings.TrimPrefix(signature, "func"))
}

func goABISuffix(behavior Behavior) string {
	if behavior.TargetABI != nil && behavior.TargetABI.Language == LanguageGo && strings.HasPrefix(behavior.TargetABI.Signature, "func") {
		return strings.TrimPrefix(behavior.TargetABI.Signature, "func")
	}
	return strings.TrimPrefix(goBehaviorABI(behavior.Kind, behavior.TriggerKind).Signature, "func")
}

func (backend GoBackend) emitModel(out *bytes.Buffer, model Model, behaviorNames map[string]string) {
	fmt.Fprintf(out, "var %sModel = hsm.Define(\n", exportName(model.Name))
	fmt.Fprintf(out, "\t%s,\n", strconv.Quote(model.Name))
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
	out.WriteString(")\n\n")
}

func (backend GoBackend) emitAttribute(out *bytes.Buffer, indent string, attribute Attribute) {
	if attribute.Type != "" {
		fmt.Fprintf(out, "%s// Type: %s\n", indent, attribute.Type)
	}
	if defaultValue, ok := goAttributeTargetDefault(attribute); ok {
		fmt.Fprintf(out, "%shsm.Attribute(%s, %s),\n", indent, strconv.Quote(attribute.Name), defaultValue)
		return
	}
	if attribute.HasDefault {
		fmt.Fprintf(out, "%s// Default: %s\n", indent, attribute.Default)
	}
	fmt.Fprintf(out, "%shsm.Attribute(%s),\n", indent, strconv.Quote(attribute.Name))
}

func goAttributeTargetDefault(attribute Attribute) (string, bool) {
	if !attribute.HasDefault {
		return "", false
	}
	value := strings.TrimSpace(attribute.Default)
	if value == "" {
		return "", false
	}
	if attribute.Language == "" || attribute.Language == LanguageGo {
		return value, true
	}
	if converted, ok := goPortableAttributeLiteral(value); ok {
		return converted, true
	}
	return "", false
}

func goPortableAttributeLiteral(value string) (string, bool) {
	switch value {
	case "True":
		return "true", true
	case "False":
		return "false", true
	case "None", "null":
		return "nil", true
	}
	if value == "true" || value == "false" || value == "nil" {
		return value, true
	}
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return value, true
	}
	if _, err := strconv.Unquote(value); err == nil {
		if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
			return value, true
		}
	}
	return "", false
}

func (backend GoBackend) emitOperation(out *bytes.Buffer, indent string, operation Operation, behaviorNames map[string]string) {
	if operation.Behavior == nil {
		return
	}
	name := behaviorNames[operation.Behavior.ID]
	if name == "" {
		name = operation.Behavior.ID
	}
	fmt.Fprintf(out, "%shsm.Operation(%s, %s),\n", indent, strconv.Quote(operation.Name), name)
}

func (backend GoBackend) emitState(out *bytes.Buffer, indent string, state State, behaviorNames map[string]string) {
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
	fmt.Fprintf(out, "%s\t%s,\n", indent, strconv.Quote(state.Name))
	if state.Initializer != nil {
		backend.emitInitial(out, indent+"\t", *state.Initializer, behaviorNames)
	}
	for _, behavior := range state.Behaviors {
		backend.emitBehaviorPartial(out, indent+"\t", behavior, behaviorNames)
	}
	for _, eventName := range state.Defers {
		fmt.Fprintf(out, "%shsm.Defer(%s),\n", indent+"\t", strconv.Quote(eventName))
	}
	for _, child := range state.States {
		backend.emitState(out, indent+"\t", child, behaviorNames)
	}
	for _, transition := range state.Transitions {
		backend.emitTransition(out, indent+"\t", transition, behaviorNames)
	}
	fmt.Fprintf(out, "%s),\n", indent)
}

func (backend GoBackend) emitInitial(out *bytes.Buffer, indent string, initial Initial, behaviorNames map[string]string) {
	fmt.Fprintf(out, "%shsm.Initial(\n", indent)
	if initial.Target != "" {
		fmt.Fprintf(out, "%s\thsm.Target(%s),\n", indent, strconv.Quote(initial.Target))
	}
	for _, effect := range initial.Effects {
		backend.emitBehaviorPartial(out, indent+"\t", effect, behaviorNames)
	}
	fmt.Fprintf(out, "%s),\n", indent)
}

func (backend GoBackend) emitTransition(out *bytes.Buffer, indent string, transition Transition, behaviorNames map[string]string) {
	fmt.Fprintf(out, "%shsm.Transition(\n", indent)
	if transition.Source != "" {
		fmt.Fprintf(out, "%s\thsm.Source(%s),\n", indent, strconv.Quote(transition.Source))
	}
	if transition.Trigger != nil {
		backend.emitTrigger(out, indent+"\t", *transition.Trigger, behaviorNames)
	}
	if transition.Guard != nil {
		backend.emitBehaviorPartial(out, indent+"\t", *transition.Guard, behaviorNames)
	}
	if transition.Target != "" {
		fmt.Fprintf(out, "%s\thsm.Target(%s),\n", indent, strconv.Quote(transition.Target))
	}
	for _, effect := range transition.Effects {
		backend.emitBehaviorPartial(out, indent+"\t", effect, behaviorNames)
	}
	fmt.Fprintf(out, "%s),\n", indent)
}

func (backend GoBackend) emitTrigger(out *bytes.Buffer, indent string, trigger Trigger, behaviorNames map[string]string) {
	switch trigger.Kind {
	case TriggerOn:
		fmt.Fprintf(out, "%shsm.On(%s),\n", indent, goTriggerValues(trigger))
	case TriggerOnSet:
		fmt.Fprintf(out, "%shsm.OnSet(%s),\n", indent, strconv.Quote(trigger.Value))
	case TriggerOnCall:
		fmt.Fprintf(out, "%shsm.OnCall(%s),\n", indent, strconv.Quote(trigger.Value))
	case TriggerAfter:
		fmt.Fprintf(out, "%shsm.After(%s),\n", indent, goExpressionValue(trigger, behaviorNames))
	case TriggerEvery:
		fmt.Fprintf(out, "%shsm.Every(%s),\n", indent, goExpressionValue(trigger, behaviorNames))
	case TriggerAt:
		fmt.Fprintf(out, "%shsm.At(%s),\n", indent, goExpressionValue(trigger, behaviorNames))
	case TriggerWhen:
		if trigger.IsString && trigger.Value != "" && trigger.Expr == nil {
			fmt.Fprintf(out, "%shsm.OnSet(%s),\n", indent, strconv.Quote(trigger.Value))
			return
		}
		fmt.Fprintf(out, "%shsm.When(%s),\n", indent, goExpressionValue(trigger, behaviorNames))
	}
}

func goTriggerValues(trigger Trigger) string {
	values := trigger.Values
	if len(values) == 0 {
		values = []TriggerValue{{Value: trigger.Value, IsString: trigger.IsString, Language: trigger.Language}}
	}
	rendered := make([]string, 0, len(values))
	for _, value := range values {
		text := value.Value
		if value.EventSymbol != "" {
			text = eventSymbolFor(LanguageGo, value.EventSymbol)
		} else if value.IsString {
			text = "hsm.Event{Name: " + strconv.Quote(text) + "}"
		}
		rendered = append(rendered, text)
	}
	return strings.Join(rendered, ", ")
}

func (backend GoBackend) emitBehaviorPartial(out *bytes.Buffer, indent string, ref BehaviorRef, behaviorNames map[string]string) {
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
		name = strconv.Quote(ref.Operation)
	}
	if name == "" {
		name = ref.ID
	}
	fmt.Fprintf(out, "%shsm.%s(%s),\n", indent, constructor, name)
}

func exportName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Generated"
	}
	var out strings.Builder
	upperNext := true
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			if upperNext && r >= 'a' && r <= 'z' {
				r -= 'a' - 'A'
			}
			out.WriteRune(r)
			upperNext = false
			continue
		}
		upperNext = true
	}
	result := out.String()
	if result == "" {
		return "Generated"
	}
	if result[0] >= '0' && result[0] <= '9' {
		return "Generated" + result
	}
	return result
}

func goExpressionValue(trigger Trigger, behaviorNames map[string]string) string {
	if trigger.IsString {
		return strconv.Quote(trigger.Value)
	}
	if trigger.Expr != nil {
		if name := behaviorNames[trigger.Expr.ID]; name != "" {
			return name
		}
	}
	if trigger.Language == "" || trigger.Language == LanguageGo {
		return trigger.Value
	}
	returnType := goTriggerReturnType(trigger.Kind)
	return fmt.Sprintf("func(ctx context.Context, instance hsm.Instance, event hsm.Event) %s {\n\t// Original %s %s trigger expression preserved for manual porting:\n%s\n\t%s\n}", returnType, trigger.Language, trigger.Kind, commentLines("\t", "//", trigger.Value), goTriggerDefaultReturn(trigger.Kind))
}

func goTriggerReturnType(kind TriggerKind) string {
	switch kind {
	case TriggerAt:
		return "time.Time"
	case TriggerWhen:
		return "<-chan struct{}"
	default:
		return "time.Duration"
	}
}

func goTriggerDefaultReturn(kind TriggerKind) string {
	switch kind {
	case TriggerAt:
		return "return time.Time{}"
	case TriggerWhen:
		return "return nil"
	default:
		return "return 0"
	}
}
