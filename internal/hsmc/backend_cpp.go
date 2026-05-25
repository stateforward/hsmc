package hsmc

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
)

type CPPBackend struct{}

func NewCPPBackend() CPPBackend {
	return CPPBackend{}
}

func (CPPBackend) Language() Language { return LanguageCPP }

func (backend CPPBackend) Emit(_ context.Context, program *Program) ([]byte, error) {
	var out bytes.Buffer
	backend.emitImports(&out, program)
	out.WriteString("namespace generated_hsm {\n\n")
	out.WriteString("using namespace hsm;\n\n")
	emittedEventTypes := map[string]bool{}
	for _, event := range program.Events {
		backend.emitEvent(&out, event)
		emittedEventTypes[cppEventTypeName(event.Name)] = true
	}
	for _, eventName := range cppDeferredEventNames(program.Models) {
		typeName := cppEventTypeName(eventName)
		if emittedEventTypes[typeName] {
			continue
		}
		backend.emitSyntheticEvent(&out, eventName)
		emittedEventTypes[typeName] = true
	}
	for _, global := range program.Globals {
		if global.Language != "" && global.Language != LanguageCPP {
			writeCommentedGlobalSource(&out, "", "//", global, LanguageCPP)
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
	behaviorsByID := map[string]Behavior{}
	for _, behavior := range program.Behaviors {
		name := snakeName(behavior.ID)
		behaviorNames[behavior.ID] = name
		behaviorsByID[behavior.ID] = behavior
		if err := backend.emitBehavior(&out, name, behavior); err != nil {
			return nil, err
		}
	}
	for _, model := range program.Models {
		backend.emitModel(&out, model, behaviorNames, behaviorsByID)
	}
	out.WriteString("}  // namespace generated_hsm\n")
	return []byte(removeCSharpTrailingCommas(out.String())), nil
}

func (backend CPPBackend) emitImports(out *bytes.Buffer, program *Program) {
	out.WriteString("#include \"hsm/hsm.hpp\"\n")
	out.WriteString("#include <chrono>\n")
	out.WriteString("#include <cstdint>\n")
	out.WriteString("#include <string>\n")
	for _, imp := range collectRenderableImports(program, LanguageCPP, compilerOwnedImports(LanguageCPP)...) {
		fmt.Fprintln(out, formatCPPImport(imp))
	}
	out.WriteString("\n")
}

func (backend CPPBackend) emitEvent(out *bytes.Buffer, event EventDecl) {
	typeName := cppEventTypeName(event.Name)
	fmt.Fprintf(out, "struct %s {\n", typeName)
	fmt.Fprintf(out, "\tstatic constexpr auto name = hsm::detail::make_fixed_string(%s);\n", strconv.Quote(event.Name))
	out.WriteString("};\n\n")
	fmt.Fprintf(out, "static constexpr auto %s = %s::name;\n\n", eventSymbolFor(LanguageCPP, event.Symbol), typeName)
}

func (backend CPPBackend) emitSyntheticEvent(out *bytes.Buffer, name string) {
	fmt.Fprintf(out, "struct %s {\n", cppEventTypeName(name))
	fmt.Fprintf(out, "\tstatic constexpr auto name = hsm::detail::make_fixed_string(%s);\n", strconv.Quote(name))
	out.WriteString("};\n\n")
}

func (backend CPPBackend) emitBehavior(out *bytes.Buffer, name string, behavior Behavior) error {
	code := strings.TrimSpace(behavior.Code)
	bodyIsEmpty := strings.TrimSpace(behavior.Body) == ""
	preserveSourceCPPExpression := behavior.SourceLanguage == LanguageCPP && !bodyIsEmpty && !cppCodeIsFunctionDefinition(code)
	if (behavior.TargetLanguage == LanguageCPP || behavior.SourceLanguage == LanguageCPP) && (bodyIsEmpty || preserveSourceCPPExpression) {
		if declaredName, ok := cLikeMethodDeclarationName(code); ok {
			if declaredName != name {
				return fmt.Errorf("cpp behavior full code declares %q, want compiler-owned name %q", declaredName, name)
			}
			if err := validateCPPFunctionDeclarationShape(code, behavior); err != nil {
				return err
			}
			out.WriteString(code)
			if !strings.HasSuffix(code, "\n") {
				out.WriteString("\n")
			}
			out.WriteString("\n")
			return nil
		}
		if declaredName, ok := cppNonCallableDeclarationName(code); ok {
			if declaredName != name {
				return fmt.Errorf("cpp behavior full code declares %q, want compiler-owned name %q", declaredName, name)
			}
			return fmt.Errorf("cpp behavior full code for %q must be a compiler-owned callable declaration, not a type or alias declaration", behavior.ID)
		}
		if declaredName, ok := cppTopLevelDeclarationName(code); ok {
			if declaredName != name {
				return fmt.Errorf("cpp behavior full code declares %q, want compiler-owned name %q", declaredName, name)
			}
			if err := validateCPPLambdaShape(code, behavior); err != nil {
				return err
			}
			out.WriteString(ensureTrailingSemicolonForCPPDeclaration(code))
			out.WriteString("\n\n")
			return nil
		}
		if strings.HasPrefix(code, "[") {
			if err := validateCPPLambdaShape(code, behavior); err != nil {
				return err
			}
			fmt.Fprintf(out, "static constexpr auto %s = %s;\n\n", name, code)
			return nil
		}
	}
	body := ""
	if behavior.TargetLanguage == LanguageCPP || behavior.SourceLanguage == LanguageCPP {
		body = strings.TrimSpace(behavior.Body)
	}
	if behavior.Kind == BehaviorOperation {
		fmt.Fprintf(out, "static %s %s() {\n", cppReturnType(behavior), name)
		if body == "" {
			writeCommentedBehaviorSource(out, "\t", "//", behavior)
			if ret := cppUntranslatedReturn(behavior); ret != "" {
				fmt.Fprintf(out, "\t%s\n", ret)
			}
		} else {
			out.WriteString(indentCPP(body))
			out.WriteString("\n")
		}
		out.WriteString("}\n\n")
		return nil
	}
	fmt.Fprintf(out, "static constexpr auto %s = []%s -> %s {\n", name, cppParameterList(behavior), cppReturnType(behavior))
	if body == "" {
		writeCommentedBehaviorSource(out, "\t", "//", behavior)
		if ret := cppUntranslatedReturn(behavior); ret != "" {
			fmt.Fprintf(out, "\t%s\n", ret)
		}
	} else {
		out.WriteString(indentCPP(body))
		out.WriteString("\n")
		if behavior.Kind == BehaviorGuard && !strings.Contains(body, "return") {
			out.WriteString("\treturn false;\n")
		}
	}
	out.WriteString("};\n\n")
	return nil
}

func cppCodeIsFunctionDefinition(code string) bool {
	_, ok := cLikeMethodDeclarationName(code)
	return ok
}

func validateCPPLambdaShape(code string, behavior Behavior) error {
	if behavior.SourceLanguage == LanguageCPP && strings.TrimSpace(behavior.Body) != "" {
		return nil
	}
	parameters, returnType, ok := cppLambdaShape(code)
	if !ok {
		return fmt.Errorf("cpp behavior full code for %q must be a compiler-owned lambda declaration or lambda expression; use body for statements", behavior.ID)
	}
	if wantParameters := cppParameterList(behavior); parameters != wantParameters {
		return fmt.Errorf("cpp behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
	}
	wantReturnType := cppReturnType(behavior)
	if returnType == "" {
		if wantReturnType != "void" {
			return fmt.Errorf("cpp behavior full code omits return type, want compiler-owned return type %q", wantReturnType)
		}
		return nil
	}
	if returnType != wantReturnType {
		return fmt.Errorf("cpp behavior full code declares return type %q, want compiler-owned return type %q", returnType, wantReturnType)
	}
	return nil
}

func validateCPPFunctionDeclarationShape(code string, behavior Behavior) error {
	returnType, _, parameters, ok := cLikeMethodDeclarationShape(code)
	if !ok {
		return nil
	}
	if behavior.Kind != BehaviorOperation {
		return fmt.Errorf("cpp behavior full code declares function parameters %q, want compiler-owned lambda parameters %q", parameters, cppParameterList(behavior))
	}
	if wantParameters := cppParameterList(behavior); parameters != wantParameters {
		return fmt.Errorf("cpp behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
	}
	if wantReturnType := cppReturnType(behavior); returnType != wantReturnType {
		return fmt.Errorf("cpp behavior full code declares return type %q, want compiler-owned return type %q", returnType, wantReturnType)
	}
	return nil
}

func cppNonCallableDeclarationName(code string) (string, bool) {
	if name, ok := cppTypeDeclarationName(code); ok {
		return name, true
	}
	if name, ok := cppUsingDeclarationName(code); ok {
		return name, true
	}
	if name, ok := cppTypedefDeclarationName(code); ok {
		return name, true
	}
	return "", false
}

func cppParameterList(behavior Behavior) string {
	if behavior.Kind == BehaviorOperation {
		return "()"
	}
	return "(auto& signal, auto& instance, const auto& event)"
}

func cppReturnType(behavior Behavior) string {
	if behavior.TargetABI != nil && behavior.TargetABI.ReturnType != "" {
		return behavior.TargetABI.ReturnType
	}
	if behavior.Kind == BehaviorGuard {
		return "bool"
	}
	return "void"
}

func (backend CPPBackend) emitModel(out *bytes.Buffer, model Model, behaviorNames map[string]string, behaviorsByID map[string]Behavior) {
	fmt.Fprintf(out, "static constexpr auto %s_model = hsm::define(\n", snakeName(model.Name))
	fmt.Fprintf(out, "\t%s", strconv.Quote(model.Name))
	if model.Initializer != nil || len(model.Attributes) > 0 || len(model.Operations) > 0 || len(model.States) > 0 || len(model.Transitions) > 0 {
		out.WriteString(",\n")
	}
	if model.Initializer != nil {
		backend.emitInitial(out, "\t", "/"+model.Name, *model.Initializer, behaviorNames)
	}
	for _, attribute := range model.Attributes {
		backend.emitAttribute(out, "\t", attribute)
	}
	for _, operation := range model.Operations {
		backend.emitOperation(out, "\t", operation, behaviorNames, behaviorsByID)
	}
	for _, state := range model.States {
		backend.emitState(out, "\t", "/"+model.Name, state, behaviorNames)
	}
	for _, transition := range model.Transitions {
		backend.emitTransition(out, "\t", "/"+model.Name, transition, behaviorNames)
	}
	out.WriteString(");\n\n")
}

func (backend CPPBackend) emitAttribute(out *bytes.Buffer, indent string, attribute Attribute) {
	if typeName, ok := cppAttributeTargetType(attribute); ok {
		if defaultValue, ok := cppAttributeTargetDefault(attribute); ok {
			fmt.Fprintf(out, "%shsm::attribute<%s>(%s, %s),\n", indent, typeName, strconv.Quote(attribute.Name), defaultValue)
			return
		}
		if attribute.HasDefault {
			fmt.Fprintf(out, "%s// Default: %s\n", indent, attribute.Default)
		}
		fmt.Fprintf(out, "%shsm::attribute<%s>(%s),\n", indent, typeName, strconv.Quote(attribute.Name))
		return
	}
	if defaultValue, ok := cppAttributeTargetDefault(attribute); ok {
		fmt.Fprintf(out, "%shsm::attribute(%s, %s),\n", indent, strconv.Quote(attribute.Name), defaultValue)
		if attribute.Type != "" {
			fmt.Fprintf(out, "%s// Type %s preserved as deduced C++ attribute default.\n", indent, attribute.Type)
		}
		return
	}
	if attribute.HasDefault {
		fmt.Fprintf(out, "%s// Default: %s\n", indent, attribute.Default)
	}
	fmt.Fprintf(out, "%s// Original attribute %s preserved for manual porting.\n", indent, attribute.Name)
	if attribute.Type != "" {
		fmt.Fprintf(out, "%s// Type: %s\n", indent, attribute.Type)
	}
}

func cppAttributeTargetDefault(attribute Attribute) (string, bool) {
	if !attribute.HasDefault {
		return "", false
	}
	value := strings.TrimSpace(attribute.Default)
	if value == "" {
		return "", false
	}
	if attribute.Language == "" || attribute.Language == LanguageCPP {
		return value, true
	}
	if converted, ok := cppPortableAttributeLiteral(value); ok {
		return converted, true
	}
	return "", false
}

func cppPortableAttributeLiteral(value string) (string, bool) {
	switch value {
	case "True":
		return "true", true
	case "False":
		return "false", true
	case "None", "null", "nil":
		return "nullptr", true
	}
	if value == "true" || value == "false" || value == "nullptr" {
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

func cppAttributeTargetType(attribute Attribute) (string, bool) {
	typeName := strings.TrimSpace(attribute.Type)
	if typeName == "" {
		return "", false
	}
	typeName = cppAttributeTypeArg(typeName)
	if strings.HasPrefix(typeName, "typeof(") && strings.HasSuffix(typeName, ")") {
		typeName = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(typeName, "typeof("), ")"))
	}
	if strings.HasSuffix(typeName, ".class") {
		typeName = strings.TrimSuffix(typeName, ".class")
	}
	switch typeName {
	case "Boolean":
		return "bool", true
	case "Byte":
		return "std::int8_t", true
	case "Character":
		return "char", true
	case "Double", "Number", "num":
		return "double", true
	case "Float":
		return "float", true
	case "Integer":
		return "int", true
	case "Long":
		return "long", true
	case "Short":
		return "short", true
	case "String", "str", "string":
		return "std::string", true
	case "boolean":
		return "bool", true
	case "bool", "byte", "char", "double", "float", "int", "long", "short", "std::string":
		return typeName, true
	case "i8":
		return "std::int8_t", true
	case "i16":
		return "std::int16_t", true
	case "i32":
		return "std::int32_t", true
	case "i64":
		return "std::int64_t", true
	case "u8":
		return "std::uint8_t", true
	case "u16":
		return "std::uint16_t", true
	case "u32":
		return "std::uint32_t", true
	case "u64":
		return "std::uint64_t", true
	default:
		return "", false
	}
}

func (backend CPPBackend) emitOperation(out *bytes.Buffer, indent string, operation Operation, behaviorNames map[string]string, behaviorsByID map[string]Behavior) {
	if operation.Behavior == nil {
		return
	}
	name := behaviorNames[operation.Behavior.ID]
	if name == "" {
		name = snakeName(operation.Behavior.ID)
	}
	if _, ok := behaviorsByID[operation.Behavior.ID]; !ok {
		fmt.Fprintf(out, "%s// Operation %q -> %s preserved for manual porting; no behavior function was generated.\n", indent, operation.Name, name)
		return
	}
	fmt.Fprintf(out, "%shsm::operation(%s, &%s),\n", indent, strconv.Quote(operation.Name), name)
}

func (backend CPPBackend) emitState(out *bytes.Buffer, indent string, parentPath string, state State, behaviorNames map[string]string) {
	constructor := "state"
	switch state.Kind {
	case StateKindFinal:
		constructor = "final"
	case StateKindChoice:
		constructor = "choice"
	case StateKindShallowHistory:
		constructor = "shallow_history"
	case StateKindDeepHistory:
		constructor = "deep_history"
	}
	hasParts := state.Initializer != nil || len(state.Behaviors) > 0 || len(state.Defers) > 0 || len(state.States) > 0 || len(state.Transitions) > 0
	fmt.Fprintf(out, "%shsm::%s(%s", indent, constructor, strconv.Quote(state.Name))
	if hasParts {
		out.WriteString(",\n")
	}
	if state.Initializer != nil {
		backend.emitInitial(out, indent+"\t", cppJoinPath(parentPath, state.Name), *state.Initializer, behaviorNames)
	}
	for _, behavior := range state.Behaviors {
		backend.emitBehaviorPartial(out, indent+"\t", behavior, behaviorNames)
	}
	for _, eventName := range state.Defers {
		fmt.Fprintf(out, "%shsm::defer<%s>(),\n", indent+"\t", cppEventTypeName(eventName))
	}
	for _, child := range state.States {
		backend.emitState(out, indent+"\t", cppJoinPath(parentPath, state.Name), child, behaviorNames)
	}
	for _, transition := range state.Transitions {
		backend.emitTransition(out, indent+"\t", cppJoinPath(parentPath, state.Name), transition, behaviorNames)
	}
	if hasParts {
		fmt.Fprintf(out, "%s),\n", indent)
		return
	}
	out.WriteString("),\n")
}

func (backend CPPBackend) emitInitial(out *bytes.Buffer, indent string, ownerPath string, initial Initial, behaviorNames map[string]string) {
	out.WriteString(indent + "hsm::initial(")
	var parts []string
	if initial.Target != "" {
		parts = append(parts, fmt.Sprintf("hsm::target(%s)", strconv.Quote(cppResolvePath(ownerPath, initial.Target))))
	}
	for _, effect := range initial.Effects {
		parts = append(parts, cppBehaviorPartialValue(effect, behaviorNames))
	}
	out.WriteString(strings.Join(parts, ", "))
	out.WriteString("),\n")
}

func (backend CPPBackend) emitTransition(out *bytes.Buffer, indent string, ownerPath string, transition Transition, behaviorNames map[string]string) {
	var parts []string
	var comments []string
	if transition.Source != "" {
		parts = append(parts, fmt.Sprintf("hsm::source(%s)", strconv.Quote(cppResolvePath(ownerPath, transition.Source))))
	}
	if transition.Trigger != nil {
		value := cppTriggerValue(*transition.Trigger, behaviorNames)
		if value == "" {
			comments = append(comments, cppUnsupportedTriggerComments(*transition.Trigger, behaviorNames)...)
		} else {
			parts = append(parts, value)
		}
	}
	if transition.Guard != nil {
		parts = append(parts, cppBehaviorPartialValue(*transition.Guard, behaviorNames))
	}
	if transition.Target != "" {
		parts = append(parts, fmt.Sprintf("hsm::target(%s)", strconv.Quote(cppResolvePath(ownerPath, transition.Target))))
	}
	for _, effect := range transition.Effects {
		parts = append(parts, cppBehaviorPartialValue(effect, behaviorNames))
	}
	for _, comment := range comments {
		fmt.Fprintf(out, "%s// %s\n", indent, comment)
	}
	out.WriteString(indent + "hsm::transition(")
	out.WriteString(strings.Join(parts, ", "))
	out.WriteString("),\n")
}

func cppTriggerValue(trigger Trigger, behaviorNames map[string]string) string {
	switch trigger.Kind {
	case TriggerOn:
		return "hsm::on(" + cppTriggerValues(trigger) + ")"
	case TriggerOnSet:
		return fmt.Sprintf("hsm::on_set(%s)", strconv.Quote(trigger.Value))
	case TriggerOnCall:
		return fmt.Sprintf("hsm::on_call(%s)", strconv.Quote(trigger.Value))
	case TriggerAfter:
		return "hsm::after(" + cppExpressionValue(trigger, behaviorNames) + ")"
	case TriggerEvery:
		return "hsm::every(" + cppExpressionValue(trigger, behaviorNames) + ")"
	case TriggerAt:
		return "hsm::at(" + cppExpressionValue(trigger, behaviorNames) + ")"
	case TriggerWhen:
		if trigger.Expr != nil {
			if name := behaviorNames[trigger.Expr.ID]; name != "" {
				return "hsm::guard(" + name + ")"
			}
			return ""
		}
		if !trigger.IsString {
			return ""
		}
		return "hsm::on_set(" + cppExpressionValue(trigger, behaviorNames) + ")"
	default:
		return ""
	}
}

func cppUnsupportedTriggerComments(trigger Trigger, behaviorNames map[string]string) []string {
	source := strings.TrimSpace(trigger.Value)
	if source == "" && trigger.Expr != nil {
		source = behaviorNames[trigger.Expr.ID]
		if source == "" {
			source = trigger.Expr.ID
		}
	}
	comments := []string{fmt.Sprintf("Unsupported %s trigger preserved for manual porting.", trigger.Kind)}
	if source == "" {
		return append(comments, "No source trigger expression was captured.")
	}
	for _, line := range strings.Split(source, "\n") {
		line = strings.ReplaceAll(line, "\t", "    ")
		if strings.TrimSpace(line) == "" {
			comments = append(comments, "")
			continue
		}
		comments = append(comments, line)
	}
	return comments
}

func cppTriggerValues(trigger Trigger) string {
	values := trigger.Values
	if len(values) == 0 {
		values = []TriggerValue{{Value: trigger.Value, IsString: trigger.IsString, Language: trigger.Language}}
	}
	rendered := make([]string, 0, len(values))
	for _, value := range values {
		text := value.Value
		if value.EventSymbol != "" {
			text = eventSymbolFor(LanguageCPP, value.EventSymbol)
		} else if value.IsString {
			text = strconv.Quote(text)
		}
		rendered = append(rendered, text)
	}
	return strings.Join(rendered, ", ")
}

func (backend CPPBackend) emitBehaviorPartial(out *bytes.Buffer, indent string, ref BehaviorRef, behaviorNames map[string]string) {
	fmt.Fprintf(out, "%s%s,\n", indent, cppBehaviorPartialValue(ref, behaviorNames))
}

func cppBehaviorPartialValue(ref BehaviorRef, behaviorNames map[string]string) string {
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
		name = strconv.Quote(ref.Operation)
	}
	if name == "" {
		name = snakeName(ref.ID)
	}
	return fmt.Sprintf("hsm::%s(%s)", constructor, name)
}

func cppExpressionValue(trigger Trigger, behaviorNames map[string]string) string {
	if trigger.Expr != nil {
		if name := behaviorNames[trigger.Expr.ID]; name != "" {
			return name
		}
	}
	if trigger.IsString {
		return strconv.Quote(trigger.Value)
	}
	if trigger.Language == "" || trigger.Language == LanguageCPP {
		return trigger.Value
	}
	return fmt.Sprintf("[](auto&, auto&, const auto&) {\n\t// Original %s %s trigger expression preserved for manual porting:\n%s\n\t%s\n}", trigger.Language, trigger.Kind, commentLines("\t", "//", trigger.Value), cppTriggerDefaultReturn(trigger.Kind))
}

func cppTriggerDefaultReturn(kind TriggerKind) string {
	switch kind {
	case TriggerAt:
		return "return std::chrono::system_clock::time_point{};"
	case TriggerWhen:
		return "return false;"
	default:
		return "return std::chrono::milliseconds{0};"
	}
}

func cppDeferredEventNames(models []Model) []string {
	seen := map[string]bool{}
	var names []string
	var visitState func(State)
	visitState = func(state State) {
		for _, name := range state.Defers {
			name = strings.TrimSpace(name)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			names = append(names, name)
		}
		for _, child := range state.States {
			visitState(child)
		}
	}
	for _, model := range models {
		for _, state := range model.States {
			visitState(state)
		}
	}
	return names
}

func cppEventTypeName(name string) string {
	return exportName(name) + "Event"
}

func cppJoinPath(parentPath string, name string) string {
	parentPath = strings.TrimRight(parentPath, "/")
	if parentPath == "" {
		return "/" + strings.Trim(name, "/")
	}
	return parentPath + "/" + strings.Trim(name, "/")
}

func cppResolvePath(ownerPath string, target string) string {
	target = strings.TrimSpace(target)
	if target == "" || strings.HasPrefix(target, "/") {
		return target
	}
	parts := strings.Split(strings.Trim(ownerPath, "/"), "/")
	for _, part := range strings.Split(target, "/") {
		switch part {
		case "", ".":
			continue
		case "..":
			if len(parts) > 0 {
				parts = parts[:len(parts)-1]
			}
		default:
			parts = append(parts, part)
		}
	}
	return "/" + strings.Join(parts, "/")
}

func indentCPP(body string) string {
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
