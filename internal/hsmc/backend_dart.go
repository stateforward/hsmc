package hsmc

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
)

type DartBackend struct{}

func NewDartBackend() DartBackend {
	return DartBackend{}
}

func (DartBackend) Language() Language { return LanguageDart }

func (backend DartBackend) Emit(_ context.Context, program *Program) ([]byte, error) {
	var out bytes.Buffer
	backend.emitImports(&out, program)
	for _, event := range program.Events {
		backend.emitEvent(&out, event)
	}
	for _, global := range program.Globals {
		if global.Language != "" && global.Language != LanguageDart {
			writeCommentedGlobalSource(&out, "", "//", global, LanguageDart)
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

func (backend DartBackend) emitImports(out *bytes.Buffer, program *Program) {
	out.WriteString(`import "package:hsm/hsm.dart";` + "\n")
	if dartNeedsAsyncImport(program) {
		out.WriteString(`import "dart:async";` + "\n")
	}
	for _, imp := range collectRenderableImports(program, LanguageDart, compilerOwnedImports(LanguageDart)...) {
		fmt.Fprintln(out, formatDartImport(imp))
	}
	out.WriteString("\n")
}

func (backend DartBackend) emitEvent(out *bytes.Buffer, event EventDecl) {
	fmt.Fprintf(out, "final %s = Event(name: %s);\n\n", eventSymbolFor(LanguageDart, event.Symbol), quoteDart(event.Name))
}

func (backend DartBackend) emitBehavior(out *bytes.Buffer, name string, behavior Behavior) error {
	if (behavior.SourceLanguage == LanguageDart || behavior.TargetLanguage == LanguageDart) && strings.TrimSpace(behavior.Body) == "" {
		code := strings.TrimSpace(behavior.Code)
		if code != "" {
			rendered, fullDeclaration, err := dartFullCodeBehavior(name, behavior, code)
			if err != nil {
				return err
			}
			if fullDeclaration {
				out.WriteString(rendered)
				if !strings.HasSuffix(rendered, "\n") {
					out.WriteString("\n")
				}
				out.WriteString("\n")
				return nil
			}
			code = strings.TrimSuffix(code, ";")
			fmt.Fprintf(out, "final %s = %s;\n\n", name, code)
			return nil
		}
	}
	body := ""
	if behavior.TargetLanguage == LanguageDart || behavior.SourceLanguage == LanguageDart {
		body = strings.TrimSpace(behavior.Body)
	}
	fmt.Fprintf(out, "%s %s(Context ctx, Instance instance, Event event) {\n", dartReturnType(behavior), name)
	if body == "" {
		writeCommentedBehaviorSource(out, "\t", "//", behavior)
		if ret := dartUntranslatedReturn(behavior); ret != "" {
			fmt.Fprintf(out, "\t%s\n", ret)
		}
	} else {
		out.WriteString(indentDart(body))
		out.WriteString("\n")
		if behavior.Kind == BehaviorGuard && !strings.Contains(body, "return") {
			out.WriteString("\treturn false;\n")
		}
	}
	out.WriteString("}\n\n")
	return nil
}

func dartFullCodeBehavior(expectedName string, behavior Behavior, code string) (string, bool, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", false, nil
	}
	if declaredName, ok := dartTopLevelVariableName(code); ok {
		if declaredName != expectedName {
			return "", false, fmt.Errorf("dart behavior full code declares %q, want compiler-owned name %q", declaredName, expectedName)
		}
		if parameters, ok := dartFunctionExpressionParameters(code); ok {
			if err := validateDartFullCodeParameters(behavior, parameters); err != nil {
				return "", false, err
			}
		} else {
			return "", false, fmt.Errorf("dart behavior full code for %q must be a compiler-owned callable declaration", expectedName)
		}
		return ensureTrailingSemicolon(code), true, nil
	}
	if declaredName, ok := dartTypeDeclarationName(code); ok {
		if declaredName != expectedName {
			return "", false, fmt.Errorf("dart behavior full code declares %q, want compiler-owned name %q", declaredName, expectedName)
		}
		return "", false, fmt.Errorf("dart behavior full code for %q must be a compiler-owned callable declaration", expectedName)
	}
	if declaredName, ok := dartExternalFunctionDeclarationName(code); ok {
		if declaredName != expectedName {
			return "", false, fmt.Errorf("dart behavior full code declares %q, want compiler-owned name %q", declaredName, expectedName)
		}
		return "", false, fmt.Errorf("dart behavior full code for %q must be a compiler-owned callable declaration", expectedName)
	}
	declaredName, ok := dartFunctionDeclarationName(code)
	if !ok {
		parameters, ok := dartFunctionExpressionParameters(code)
		if !ok {
			if behavior.TargetLanguage == LanguageDart {
				return "", false, fmt.Errorf("dart behavior full code for %q must be a compiler-owned callable expression", expectedName)
			}
			return "", false, nil
		}
		if err := validateDartFullCodeParameters(behavior, parameters); err != nil {
			return "", false, err
		}
		return "", false, nil
	}
	if declaredName != expectedName {
		return "", false, fmt.Errorf("dart behavior full code declares %q, want compiler-owned name %q", declaredName, expectedName)
	}
	if returnType, _, parameters, ok := dartFunctionDeclarationShape(code); ok {
		if wantReturnType := dartReturnType(behavior); returnType != wantReturnType {
			return "", false, fmt.Errorf("dart behavior full code declares return type %q, want compiler-owned return type %q", returnType, wantReturnType)
		}
		if err := validateDartFullCodeParameters(behavior, parameters); err != nil {
			return "", false, err
		}
	}
	return code, true, nil
}

func validateDartFullCodeParameters(_ Behavior, parameters string) error {
	if wantParameters := dartParameterList(); parameters != wantParameters {
		return fmt.Errorf("dart behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
	}
	return nil
}

func dartTopLevelVariableName(code string) (string, bool) {
	trimmed := strings.TrimSpace(code)
	for _, keyword := range []string{"final", "const", "var"} {
		prefix := keyword + " "
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(trimmed, keyword))
		eq := strings.Index(rest, "=")
		if eq < 0 {
			return "", false
		}
		name := strings.TrimSpace(rest[:eq])
		fields := strings.Fields(name)
		if len(fields) == 0 {
			return "", false
		}
		name = fields[len(fields)-1]
		return name, isDartIdentifier(name)
	}
	firstLine := strings.TrimSpace(strings.SplitN(trimmed, "\n", 2)[0])
	eq := firstTopLevelByte(firstLine, '=')
	if eq < 0 {
		return "", false
	}
	name := strings.TrimSpace(firstLine[:eq])
	if len(splitTopLevelCommaSegments(name)) > 1 {
		return "", false
	}
	fields := strings.Fields(name)
	if len(fields) < 2 {
		return "", false
	}
	name = trimDeclarationName(fields[len(fields)-1])
	return name, isDartIdentifier(name)
}

func dartImportDeclarationNames(code string) []string {
	trimmed := strings.TrimSpace(strings.TrimSuffix(code, ";"))
	if !strings.HasPrefix(trimmed, "import ") {
		return nil
	}
	rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "import "))
	if rest == "" || (rest[0] != '\'' && rest[0] != '"') {
		return nil
	}
	quote := rest[0]
	close := -1
	for index := 1; index < len(rest); index++ {
		if rest[index] == quote && rest[index-1] != '\\' {
			close = index
			break
		}
	}
	if close < 0 {
		return nil
	}
	suffix := strings.TrimSpace(rest[close+1:])
	if suffix == "" {
		return nil
	}
	fields := strings.Fields(strings.ReplaceAll(suffix, ",", " , "))
	for index, field := range fields {
		if field == "as" && index+1 < len(fields) {
			name := trimDeclarationName(fields[index+1])
			if isDartIdentifier(name) {
				return []string{name}
			}
			return nil
		}
	}
	for index, field := range fields {
		if field != "show" {
			continue
		}
		var names []string
		for _, token := range fields[index+1:] {
			if token == "," {
				continue
			}
			if token == "as" || token == "hide" || token == "show" || token == "deferred" {
				break
			}
			name := trimDeclarationName(token)
			if isDartIdentifier(name) {
				names = append(names, name)
			}
		}
		return names
	}
	return nil
}

func dartTypeDeclarationName(code string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(code))
	for index, field := range fields {
		if field != "class" && field != "enum" && field != "mixin" && field != "extension" && field != "typedef" {
			continue
		}
		nameIndex := index + 1
		if field == "extension" && nameIndex < len(fields) && fields[nameIndex] == "type" {
			nameIndex++
		}
		if nameIndex >= len(fields) {
			return "", false
		}
		if !dartTypeDeclarationPrefix(fields[:index]) {
			return "", false
		}
		name := trimDeclarationName(fields[nameIndex])
		if !isDartIdentifier(name) {
			return "", false
		}
		return name, true
	}
	return "", false
}

func dartTypeDeclarationPrefix(fields []string) bool {
	for _, field := range fields {
		if field == "" {
			continue
		}
		if field == "abstract" || field == "base" || field == "final" || field == "interface" ||
			field == "sealed" || field == "mixin" || field == "extension" {
			continue
		}
		return false
	}
	return true
}

func dartFunctionExpressionParameters(code string) (string, bool) {
	expression := strings.TrimSpace(code)
	if _, ok := dartTopLevelVariableName(expression); ok {
		eq := firstTopLevelByte(expression, '=')
		if eq < 0 {
			return "", false
		}
		expression = strings.TrimSpace(expression[eq+1:])
	}
	expression = strings.TrimSuffix(strings.TrimSpace(expression), ";")
	if !strings.HasPrefix(expression, "(") {
		return "", false
	}
	closeParen := matchingCloseParen(expression)
	if closeParen < 0 {
		return "", false
	}
	rest := strings.TrimSpace(expression[closeParen+1:])
	if strings.HasPrefix(rest, "async") {
		rest = strings.TrimSpace(strings.TrimPrefix(rest, "async"))
	}
	if !strings.HasPrefix(rest, "{") && !strings.HasPrefix(rest, "=>") {
		return "", false
	}
	return strings.TrimSpace(expression[:closeParen+1]), true
}

func dartFunctionDeclarationName(code string) (string, bool) {
	_, name, _, ok := dartFunctionDeclarationShape(code)
	return name, ok
}

func dartExternalFunctionDeclarationName(code string) (string, bool) {
	trimmed := strings.TrimSpace(code)
	if !strings.HasPrefix(trimmed, "external ") {
		return "", false
	}
	trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "external "))
	paren := strings.Index(trimmed, "(")
	if paren < 0 {
		return "", false
	}
	closeParen := matchingCloseParen(trimmed[paren:])
	if closeParen < 0 {
		return "", false
	}
	closeParen += paren
	prefix := strings.TrimSpace(trimmed[:paren])
	if prefix == "" || containsDeclarationBoundary(prefix) {
		return "", false
	}
	fields := strings.Fields(prefix)
	if len(fields) < 2 {
		return "", false
	}
	name := fields[len(fields)-1]
	if !isDartIdentifier(name) {
		return "", false
	}
	afterParams := strings.TrimSpace(trimmed[closeParen+1:])
	if afterParams != ";" {
		return "", false
	}
	return name, true
}

func dartFunctionDeclarationShape(code string) (returnType string, name string, parameters string, ok bool) {
	paren := strings.Index(code, "(")
	if paren < 0 {
		return "", "", "", false
	}
	closeParen := matchingCloseParen(code[paren:])
	if closeParen < 0 {
		return "", "", "", false
	}
	closeParen += paren
	prefix := strings.TrimSpace(code[:paren])
	if prefix == "" || strings.HasPrefix(prefix, "(") {
		return "", "", "", false
	}
	if containsDeclarationBoundary(prefix) {
		return "", "", "", false
	}
	fields := strings.Fields(prefix)
	if len(fields) < 2 {
		return "", "", "", false
	}
	name = fields[len(fields)-1]
	if !isDartIdentifier(name) {
		return "", "", "", false
	}
	returnType = strings.TrimSpace(strings.TrimSuffix(prefix, name))
	parameters = strings.TrimSpace(code[paren : closeParen+1])
	afterParams := strings.TrimSpace(code[closeParen+1:])
	if strings.HasPrefix(afterParams, "async") {
		afterParams = strings.TrimSpace(strings.TrimPrefix(afterParams, "async"))
	}
	if !strings.HasPrefix(afterParams, "{") && !strings.HasPrefix(afterParams, "=>") {
		return "", "", "", false
	}
	if returnType == "" || parameters == "" || strings.ContainsAny(parameters, "{};=") {
		return "", "", "", false
	}
	return returnType, name, parameters, true
}

func isDartIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for index, r := range value {
		if index == 0 {
			if r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
				continue
			}
			return false
		}
		if r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

func ensureTrailingSemicolon(code string) string {
	code = strings.TrimSpace(code)
	if strings.HasSuffix(code, ";") {
		return code
	}
	return code + ";"
}

func dartReturnType(behavior Behavior) string {
	if behavior.TargetABI != nil && behavior.TargetABI.ReturnType != "" {
		return behavior.TargetABI.ReturnType
	}
	if behavior.Kind == BehaviorGuard {
		return "bool"
	}
	return "void"
}

func dartParameterList() string {
	return "(Context ctx, Instance instance, Event event)"
}

func (backend DartBackend) emitModel(out *bytes.Buffer, model Model, behaviorNames map[string]string) {
	fmt.Fprintf(out, "final %sModel = define(\n", lowerCamel(exportName(model.Name)))
	fmt.Fprintf(out, "\t%s,\n", quoteDart(model.Name))
	out.WriteString("\t[\n")
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
	out.WriteString("\t],\n")
	out.WriteString(");\n\n")
}

func (backend DartBackend) emitAttribute(out *bytes.Buffer, indent string, attribute Attribute) {
	typeName, hasTargetType := dartAttributeTargetType(attribute)
	defaultValue, hasTargetDefault := dartAttributeTargetDefault(attribute)
	if attribute.Type != "" && !hasTargetType {
		fmt.Fprintf(out, "%s// Type: %s\n", indent, attribute.Type)
	}
	if attribute.HasDefault && !hasTargetDefault {
		fmt.Fprintf(out, "%s// Default: %s\n", indent, attribute.Default)
	}
	switch {
	case hasTargetType && hasTargetDefault:
		fmt.Fprintf(out, "%sattribute(%s, %s, %s),\n", indent, quoteDart(attribute.Name), typeName, defaultValue)
	case hasTargetType:
		fmt.Fprintf(out, "%sattribute(%s, %s),\n", indent, quoteDart(attribute.Name), typeName)
	case hasTargetDefault:
		fmt.Fprintf(out, "%sattribute(%s, %s),\n", indent, quoteDart(attribute.Name), defaultValue)
	default:
		fmt.Fprintf(out, "%sattribute(%s),\n", indent, quoteDart(attribute.Name))
	}
}

func dartAttributeTargetType(attribute Attribute) (string, bool) {
	if !attributeTypeIsTargetExpression(attribute, LanguageDart) {
		return "", false
	}
	return strings.TrimSpace(attribute.Type), true
}

func dartAttributeTargetDefault(attribute Attribute) (string, bool) {
	if !attribute.HasDefault {
		return "", false
	}
	value := strings.TrimSpace(attribute.Default)
	if value == "" {
		return "", false
	}
	if attribute.Language == "" || attribute.Language == LanguageDart {
		return value, true
	}
	if converted, ok := dartPortableLiteral(value); ok {
		return converted, true
	}
	return "", false
}

func dartPortableLiteral(value string) (string, bool) {
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

func (backend DartBackend) emitOperation(out *bytes.Buffer, indent string, operation Operation, behaviorNames map[string]string) {
	if operation.Behavior == nil {
		return
	}
	name := behaviorNames[operation.Behavior.ID]
	if name == "" {
		name = lowerCamel(exportName(operation.Behavior.ID))
	}
	fmt.Fprintf(out, "%soperation(%s, %s),\n", indent, quoteDart(operation.Name), name)
}

func (backend DartBackend) emitState(out *bytes.Buffer, indent string, state State, behaviorNames map[string]string) {
	constructor := "state"
	switch state.Kind {
	case StateKindFinal:
		constructor = "finalState"
	case StateKindChoice:
		constructor = "choice"
	case StateKindShallowHistory:
		constructor = "shallowHistory"
	case StateKindDeepHistory:
		constructor = "deepHistory"
	}
	if state.Kind == StateKindFinal {
		fmt.Fprintf(out, "%s%s(%s),\n", indent, constructor, quoteDart(state.Name))
		return
	}
	fmt.Fprintf(out, "%s%s(\n", indent, constructor)
	fmt.Fprintf(out, "%s\t%s,\n", indent, quoteDart(state.Name))
	out.WriteString(indent + "\t[\n")
	if state.Initializer != nil {
		backend.emitInitial(out, indent+"\t\t", *state.Initializer, behaviorNames)
	}
	for _, behavior := range state.Behaviors {
		backend.emitBehaviorPartial(out, indent+"\t\t", behavior, behaviorNames)
	}
	for _, eventName := range state.Defers {
		fmt.Fprintf(out, "%sdefer(%s),\n", indent+"\t\t", quoteDart(eventName))
	}
	for _, child := range state.States {
		backend.emitState(out, indent+"\t\t", child, behaviorNames)
	}
	for _, transition := range state.Transitions {
		backend.emitTransition(out, indent+"\t\t", transition, behaviorNames)
	}
	out.WriteString(indent + "\t],\n")
	fmt.Fprintf(out, "%s),\n", indent)
}

func (backend DartBackend) emitInitial(out *bytes.Buffer, indent string, initial Initial, behaviorNames map[string]string) {
	if len(initial.Effects) == 0 {
		fmt.Fprintf(out, "%sinitial(target(%s)),\n", indent, quoteDart(initial.Target))
		return
	}
	fmt.Fprintf(out, "%sinitial(\n", indent)
	fmt.Fprintf(out, "%s\ttarget(%s),\n", indent, quoteDart(initial.Target))
	out.WriteString(indent + "\t[\n")
	for _, effect := range initial.Effects {
		backend.emitBehaviorPartial(out, indent+"\t\t", effect, behaviorNames)
	}
	out.WriteString(indent + "\t],\n")
	fmt.Fprintf(out, "%s),\n", indent)
}

func (backend DartBackend) emitTransition(out *bytes.Buffer, indent string, transition Transition, behaviorNames map[string]string) {
	fmt.Fprintf(out, "%stransition([\n", indent)
	if transition.Source != "" {
		fmt.Fprintf(out, "%s\tsource(%s),\n", indent, quoteDart(transition.Source))
	}
	if transition.Trigger != nil {
		backend.emitTrigger(out, indent+"\t", *transition.Trigger, behaviorNames)
	}
	if transition.Guard != nil {
		backend.emitBehaviorPartial(out, indent+"\t", *transition.Guard, behaviorNames)
	}
	if transition.Target != "" {
		fmt.Fprintf(out, "%s\ttarget(%s),\n", indent, quoteDart(transition.Target))
	}
	for _, effect := range transition.Effects {
		backend.emitBehaviorPartial(out, indent+"\t", effect, behaviorNames)
	}
	fmt.Fprintf(out, "%s]),\n", indent)
}

func (backend DartBackend) emitTrigger(out *bytes.Buffer, indent string, trigger Trigger, behaviorNames map[string]string) {
	switch trigger.Kind {
	case TriggerOn:
		for _, value := range dartTriggerValues(trigger) {
			fmt.Fprintf(out, "%son(%s),\n", indent, value)
		}
	case TriggerOnSet:
		fmt.Fprintf(out, "%sonSet(%s),\n", indent, quoteDart(trigger.Value))
	case TriggerOnCall:
		fmt.Fprintf(out, "%sonCall(%s),\n", indent, quoteDart(trigger.Value))
	case TriggerAfter:
		fmt.Fprintf(out, "%safter(%s),\n", indent, dartExpressionValue(trigger, behaviorNames))
	case TriggerEvery:
		fmt.Fprintf(out, "%severy(%s),\n", indent, dartExpressionValue(trigger, behaviorNames))
	case TriggerAt:
		fmt.Fprintf(out, "%sat(%s),\n", indent, dartExpressionValue(trigger, behaviorNames))
	case TriggerWhen:
		if trigger.IsString && trigger.Value != "" && trigger.Expr == nil {
			fmt.Fprintf(out, "%swhen(%s),\n", indent, quoteDart(trigger.Value))
			return
		}
		fmt.Fprintf(out, "%swhen(%s),\n", indent, dartExpressionValue(trigger, behaviorNames))
	default:
		fmt.Fprintf(out, "%s// Unsupported %s trigger preserved for manual porting: %s\n", indent, trigger.Kind, trigger.Value)
	}
}

func dartTriggerValues(trigger Trigger) []string {
	values := trigger.Values
	if len(values) == 0 {
		values = []TriggerValue{{Value: trigger.Value, IsString: trigger.IsString, Language: trigger.Language}}
	}
	rendered := make([]string, 0, len(values))
	for _, value := range values {
		text := value.Value
		if value.EventSymbol != "" {
			text = eventSymbolFor(LanguageDart, value.EventSymbol) + ".name"
		} else if value.IsString {
			text = quoteDart(text)
		}
		rendered = append(rendered, text)
	}
	return rendered
}

func (backend DartBackend) emitBehaviorPartial(out *bytes.Buffer, indent string, ref BehaviorRef, behaviorNames map[string]string) {
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
		name = quoteDart(ref.Operation)
	}
	if name == "" {
		name = lowerCamel(exportName(ref.ID))
	}
	fmt.Fprintf(out, "%s%s(%s),\n", indent, constructor, name)
}

func dartExpressionValue(trigger Trigger, behaviorNames map[string]string) string {
	if trigger.Expr != nil {
		if name := behaviorNames[trigger.Expr.ID]; name != "" {
			return name
		}
	}
	if trigger.IsString {
		return quoteDart(trigger.Value)
	}
	if trigger.Language == "" || trigger.Language == LanguageDart {
		return trigger.Value
	}
	return fmt.Sprintf("(Context ctx, Instance instance, Event event) {\n\t// Original %s %s trigger expression preserved for manual porting:\n%s\n\t%s\n}", trigger.Language, trigger.Kind, commentLines("\t", "//", trigger.Value), dartTriggerDefaultReturn(trigger.Kind))
}

func dartTriggerDefaultReturn(kind TriggerKind) string {
	switch kind {
	case TriggerAt:
		return "return DateTime.fromMillisecondsSinceEpoch(0);"
	case TriggerWhen:
		return "return false;"
	default:
		return "return Duration.zero;"
	}
}

func quoteDart(value string) string {
	return strconv.Quote(value)
}

func indentDart(body string) string {
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

func dartUsesTriggerKind(program *Program, kind TriggerKind) bool {
	for _, model := range program.Models {
		if dartModelUsesTriggerKind(model, kind) {
			return true
		}
	}
	return false
}

func dartUsesStateKind(program *Program, kind StateKind) bool {
	for _, model := range program.Models {
		for _, state := range model.States {
			if dartStateUsesStateKind(state, kind) {
				return true
			}
		}
	}
	return false
}

func dartNeedsAsyncImport(program *Program) bool {
	if dartUsesTriggerKind(program, TriggerAt) || dartUsesTriggerKind(program, TriggerEvery) {
		return true
	}
	for _, behavior := range program.Behaviors {
		if behavior.TargetABI != nil && strings.Contains(behavior.TargetABI.ReturnType, "FutureOr<") {
			return true
		}
	}
	return false
}

func dartStateUsesStateKind(state State, kind StateKind) bool {
	if state.Kind == kind {
		return true
	}
	for _, child := range state.States {
		if dartStateUsesStateKind(child, kind) {
			return true
		}
	}
	return false
}

func dartModelUsesTriggerKind(model Model, kind TriggerKind) bool {
	for _, transition := range model.Transitions {
		if transition.Trigger != nil && transition.Trigger.Kind == kind {
			return true
		}
	}
	for _, state := range model.States {
		if dartStateUsesTriggerKind(state, kind) {
			return true
		}
	}
	return false
}

func dartStateUsesTriggerKind(state State, kind TriggerKind) bool {
	for _, transition := range state.Transitions {
		if transition.Trigger != nil && transition.Trigger.Kind == kind {
			return true
		}
	}
	for _, child := range state.States {
		if dartStateUsesTriggerKind(child, kind) {
			return true
		}
	}
	return false
}
