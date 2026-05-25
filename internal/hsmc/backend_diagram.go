package hsmc

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type MermaidBackend struct{}

func NewMermaidBackend() MermaidBackend {
	return MermaidBackend{}
}

func (MermaidBackend) Language() Language { return LanguageMermaid }

func (MermaidBackend) Emit(_ context.Context, program *Program) ([]byte, error) {
	diagram := newDiagramModel(program)
	var out bytes.Buffer
	out.WriteString("stateDiagram-v2\n")
	for _, model := range diagram.models {
		diagram.writeMermaidModel(&out, model)
	}
	return out.Bytes(), nil
}

type PlantUMLBackend struct{}

func NewPlantUMLBackend() PlantUMLBackend {
	return PlantUMLBackend{}
}

func (PlantUMLBackend) Language() Language { return LanguagePlantUML }

func (PlantUMLBackend) Emit(_ context.Context, program *Program) ([]byte, error) {
	diagram := newDiagramModel(program)
	var out bytes.Buffer
	out.WriteString("@startuml\n")
	for _, model := range diagram.models {
		diagram.writePlantUMLModel(&out, model)
	}
	out.WriteString("@enduml\n")
	return out.Bytes(), nil
}

type diagramModel struct {
	behaviorByID map[string]Behavior
	globals      []CodeBlock
	stateByID    map[string]diagramStateRef
	models       []Model
}

type diagramStateRef struct {
	alias string
	path  string
}

func newDiagramModel(program *Program) diagramModel {
	diagram := diagramModel{
		behaviorByID: map[string]Behavior{},
		stateByID:    map[string]diagramStateRef{},
	}
	if program == nil {
		return diagram
	}
	diagram.globals = append([]CodeBlock(nil), program.Globals...)
	for _, behavior := range program.Behaviors {
		diagram.behaviorByID[behavior.ID] = behavior
	}
	diagram.models = append([]Model(nil), program.Models...)
	for _, model := range program.Models {
		modelPath := cleanModelPath(model.Name)
		for _, state := range model.States {
			diagram.indexState(state, modelPath)
		}
	}
	return diagram
}

func (diagram *diagramModel) indexState(state State, ownerPath string) {
	statePath := cleanJoinedPath(ownerPath, state.Name)
	diagram.stateByID[state.ID] = diagramStateRef{alias: diagramAlias(statePath), path: statePath}
	for _, child := range state.States {
		diagram.indexState(child, statePath)
	}
}

func (diagram diagramModel) writeMermaidModel(out *bytes.Buffer, model Model) {
	modelPath := cleanModelPath(model.Name)
	modelAlias := diagramAlias(modelPath)
	fmt.Fprintf(out, "state %s as %s {\n", strconv.Quote(model.Name), modelAlias)
	if model.Initializer != nil && model.Initializer.Target != "" {
		fmt.Fprintf(out, "  [*] --> %s\n", diagram.resolveTargetAlias(modelPath, model.Initializer.Target))
	}
	for _, transition := range model.Transitions {
		diagram.writeMermaidTransition(out, "  ", modelPath, modelAlias, transition)
	}
	for _, state := range model.States {
		diagram.writeMermaidState(out, "  ", modelPath, state)
	}
	diagram.writeMermaidModelNotes(out, "  ", modelAlias, model)
	out.WriteString("}\n")
}

func (diagram diagramModel) writeMermaidState(out *bytes.Buffer, indent string, ownerPath string, state State) {
	statePath := cleanJoinedPath(ownerPath, state.Name)
	alias := diagramAlias(statePath)
	if len(state.States) == 0 && state.Initializer == nil && len(state.Transitions) == 0 {
		fmt.Fprintf(out, "%sstate %s as %s\n", indent, strconv.Quote(state.Name), alias)
	} else {
		fmt.Fprintf(out, "%sstate %s as %s {\n", indent, strconv.Quote(state.Name), alias)
		if state.Initializer != nil && state.Initializer.Target != "" {
			fmt.Fprintf(out, "%s  [*] --> %s\n", indent, diagram.resolveTargetAlias(statePath, state.Initializer.Target))
		}
		for _, transition := range state.Transitions {
			diagram.writeMermaidTransition(out, indent+"  ", statePath, alias, transition)
		}
		for _, child := range state.States {
			diagram.writeMermaidState(out, indent+"  ", statePath, child)
		}
		fmt.Fprintf(out, "%s}\n", indent)
	}
	diagram.writeMermaidStateNotes(out, indent, alias, state)
}

func (diagram diagramModel) writeMermaidTransition(out *bytes.Buffer, indent string, ownerPath string, ownerAlias string, transition Transition) {
	source := ownerAlias
	if transition.Source != "" {
		source = diagram.resolveTargetAlias(ownerPath, transition.Source)
	}
	target := "[*]"
	if transition.Target != "" {
		target = diagram.resolveTargetAlias(ownerPath, transition.Target)
	}
	label := diagram.transitionLabel(transition)
	if label != "" {
		fmt.Fprintf(out, "%s%s --> %s : %s\n", indent, source, target, mermaidLine(label))
	} else {
		fmt.Fprintf(out, "%s%s --> %s\n", indent, source, target)
	}
}

func (diagram diagramModel) writeMermaidStateNotes(out *bytes.Buffer, indent string, alias string, state State) {
	lines := diagram.behaviorRefNoteLines(state.Behaviors)
	if len(state.Defers) > 0 {
		lines = append(lines, "defers: "+strings.Join(state.Defers, ", "))
	}
	diagram.writeMermaidNote(out, indent, alias, lines)
}

func (diagram diagramModel) writeMermaidModelNotes(out *bytes.Buffer, indent string, alias string, model Model) {
	var lines []string
	for _, attribute := range model.Attributes {
		lines = append(lines, diagramAttributeNoteLines(attribute)...)
	}
	for _, operation := range model.Operations {
		if operation.Behavior != nil {
			lines = append(lines, "operation "+operation.Name+":")
			lines = append(lines, indentNoteLines(diagram.behaviorRefNoteLines([]BehaviorRef{*operation.Behavior}))...)
		}
	}
	lines = append(lines, diagram.allBehaviorNoteLines()...)
	lines = append(lines, diagram.globalNoteLines()...)
	diagram.writeMermaidNote(out, indent, alias, lines)
}

func (diagram diagramModel) writeMermaidNote(out *bytes.Buffer, indent string, alias string, lines []string) {
	lines = compactDiagramNoteLines(lines)
	if len(lines) == 0 {
		return
	}
	fmt.Fprintf(out, "%snote right of %s\n", indent, alias)
	for _, line := range lines {
		fmt.Fprintf(out, "%s  %s\n", indent, mermaidLine(line))
	}
	fmt.Fprintf(out, "%send note\n", indent)
}

func (diagram diagramModel) writePlantUMLModel(out *bytes.Buffer, model Model) {
	modelPath := cleanModelPath(model.Name)
	modelAlias := diagramAlias(modelPath)
	fmt.Fprintf(out, "state %s as %s {\n", strconv.Quote(model.Name), modelAlias)
	if model.Initializer != nil && model.Initializer.Target != "" {
		fmt.Fprintf(out, "  [*] --> %s\n", diagram.resolveTargetAlias(modelPath, model.Initializer.Target))
	}
	for _, transition := range model.Transitions {
		diagram.writePlantUMLTransition(out, "  ", modelPath, modelAlias, transition)
	}
	for _, state := range model.States {
		diagram.writePlantUMLState(out, "  ", modelPath, state)
	}
	diagram.writePlantUMLModelNotes(out, "  ", modelAlias, model)
	out.WriteString("}\n")
}

func (diagram diagramModel) writePlantUMLState(out *bytes.Buffer, indent string, ownerPath string, state State) {
	statePath := cleanJoinedPath(ownerPath, state.Name)
	alias := diagramAlias(statePath)
	if len(state.States) == 0 && state.Initializer == nil && len(state.Transitions) == 0 {
		fmt.Fprintf(out, "%sstate %s as %s\n", indent, strconv.Quote(state.Name), alias)
	} else {
		fmt.Fprintf(out, "%sstate %s as %s {\n", indent, strconv.Quote(state.Name), alias)
		if state.Initializer != nil && state.Initializer.Target != "" {
			fmt.Fprintf(out, "%s  [*] --> %s\n", indent, diagram.resolveTargetAlias(statePath, state.Initializer.Target))
		}
		for _, transition := range state.Transitions {
			diagram.writePlantUMLTransition(out, indent+"  ", statePath, alias, transition)
		}
		for _, child := range state.States {
			diagram.writePlantUMLState(out, indent+"  ", statePath, child)
		}
		fmt.Fprintf(out, "%s}\n", indent)
	}
	diagram.writePlantUMLStateNotes(out, indent, alias, state)
}

func (diagram diagramModel) writePlantUMLTransition(out *bytes.Buffer, indent string, ownerPath string, ownerAlias string, transition Transition) {
	source := ownerAlias
	if transition.Source != "" {
		source = diagram.resolveTargetAlias(ownerPath, transition.Source)
	}
	target := "[*]"
	if transition.Target != "" {
		target = diagram.resolveTargetAlias(ownerPath, transition.Target)
	}
	label := diagram.transitionLabel(transition)
	if label != "" {
		fmt.Fprintf(out, "%s%s --> %s : %s\n", indent, source, target, plantUMLLine(label))
	} else {
		fmt.Fprintf(out, "%s%s --> %s\n", indent, source, target)
	}
}

func (diagram diagramModel) writePlantUMLStateNotes(out *bytes.Buffer, indent string, alias string, state State) {
	lines := diagram.behaviorRefNoteLines(state.Behaviors)
	if len(state.Defers) > 0 {
		lines = append(lines, "defers: "+strings.Join(state.Defers, ", "))
	}
	diagram.writePlantUMLNote(out, indent, alias, lines)
}

func (diagram diagramModel) writePlantUMLModelNotes(out *bytes.Buffer, indent string, alias string, model Model) {
	var lines []string
	for _, attribute := range model.Attributes {
		lines = append(lines, diagramAttributeNoteLines(attribute)...)
	}
	for _, operation := range model.Operations {
		if operation.Behavior != nil {
			lines = append(lines, "operation "+operation.Name+":")
			lines = append(lines, indentNoteLines(diagram.behaviorRefNoteLines([]BehaviorRef{*operation.Behavior}))...)
		}
	}
	lines = append(lines, diagram.allBehaviorNoteLines()...)
	lines = append(lines, diagram.globalNoteLines()...)
	diagram.writePlantUMLNote(out, indent, alias, lines)
}

func (diagram diagramModel) writePlantUMLNote(out *bytes.Buffer, indent string, alias string, lines []string) {
	lines = compactDiagramNoteLines(lines)
	if len(lines) == 0 {
		return
	}
	fmt.Fprintf(out, "%snote right of %s\n", indent, alias)
	for _, line := range lines {
		fmt.Fprintf(out, "%s  %s\n", indent, plantUMLLine(line))
	}
	fmt.Fprintf(out, "%send note\n", indent)
}

func (diagram diagramModel) transitionLabel(transition Transition) string {
	var parts []string
	if transition.Trigger != nil {
		parts = append(parts, diagramTriggerLabel(*transition.Trigger))
	}
	if transition.Guard != nil {
		parts = append(parts, "guard "+diagramBehaviorRefName(*transition.Guard))
	}
	if len(transition.Effects) > 0 {
		var effects []string
		for _, effect := range transition.Effects {
			effects = append(effects, diagramBehaviorRefName(effect))
		}
		parts = append(parts, "effect "+strings.Join(effects, ", "))
	}
	return strings.Join(parts, " / ")
}

func (diagram diagramModel) behaviorRefNoteLines(refs []BehaviorRef) []string {
	var lines []string
	for _, ref := range refs {
		behavior, ok := diagram.behaviorByID[ref.ID]
		if !ok {
			continue
		}
		lines = append(lines, diagramBehaviorNoteLines(behavior)...)
	}
	return lines
}

func (diagram diagramModel) allBehaviorNoteLines() []string {
	ids := make([]string, 0, len(diagram.behaviorByID))
	for id := range diagram.behaviorByID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var lines []string
	for _, id := range ids {
		lines = append(lines, diagramBehaviorNoteLines(diagram.behaviorByID[id])...)
	}
	return lines
}

func diagramBehaviorNoteLines(behavior Behavior) []string {
	lines := []string{"Original " + string(behavior.SourceLanguage) + " behavior " + behavior.ID + " preserved for manual porting:"}
	if behavior.Kind != "" {
		lines = append(lines, "Behavior kind: "+string(behavior.Kind))
	}
	lines = append(lines, diagramImportNoteLines(behavior.SourceLanguage, behavior.Imports)...)
	lines = append(lines, indentNoteLines(diagramBehaviorSourceLines(behavior))...)
	return lines
}

func (diagram diagramModel) globalNoteLines() []string {
	var lines []string
	for _, global := range diagram.globals {
		code := strings.TrimSpace(global.Code)
		if global.Language == "" || code == "" {
			continue
		}
		lines = append(lines, "Original "+string(global.Language)+" global "+global.ID+" preserved for manual porting:")
		lines = append(lines, diagramImportNoteLines(global.Language, global.Imports)...)
		lines = append(lines, indentNoteLines(strings.Split(code, "\n"))...)
	}
	return lines
}

func diagramImportNoteLines(language Language, imports []Import) []string {
	if len(imports) == 0 {
		return nil
	}
	lines := []string{"Source imports:"}
	for _, imp := range imports {
		if line := sourceImportText(importCommentLanguage(language, imp), imp); line != "" {
			lines = append(lines, "  "+line)
		}
	}
	return lines
}

func (diagram diagramModel) resolveTargetAlias(ownerPath string, target string) string {
	if target == "" {
		return ""
	}
	targetPath := target
	if !strings.HasPrefix(targetPath, "/") {
		targetPath = path.Clean(path.Join(ownerPath, targetPath))
	}
	if !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}
	return diagramAlias(targetPath)
}

func diagramTriggerLabel(trigger Trigger) string {
	if len(trigger.Values) > 0 {
		var values []string
		for _, value := range trigger.Values {
			if value.EventSymbol != "" {
				values = append(values, value.EventSymbol)
			} else {
				values = append(values, value.Value)
			}
		}
		return string(trigger.Kind) + " " + strings.Join(values, ", ")
	}
	if trigger.Expr != nil {
		return string(trigger.Kind) + " " + diagramBehaviorRefName(*trigger.Expr)
	}
	if trigger.Value != "" {
		return string(trigger.Kind) + " " + trigger.Value
	}
	return string(trigger.Kind)
}

func diagramBehaviorRefName(ref BehaviorRef) string {
	if ref.Operation != "" {
		return ref.Operation
	}
	if ref.ID != "" {
		return ref.ID
	}
	if ref.Kind != "" {
		return string(ref.Kind)
	}
	return "behavior"
}

func diagramBehaviorSourceLines(behavior Behavior) []string {
	source := behaviorSourceText(behavior)
	if strings.TrimSpace(source) == "" {
		return []string{"No source behavior body was captured."}
	}
	return strings.Split(strings.TrimSpace(source), "\n")
}

func diagramAttributeNoteLines(attribute Attribute) []string {
	var lines []string
	if attribute.Type != "" {
		lines = append(lines, "attribute "+attribute.Name+": "+attribute.Type)
	}
	if attribute.HasDefault && strings.TrimSpace(attribute.Default) != "" {
		lines = append(lines, "default "+attribute.Name+" ("+string(attribute.Language)+"): "+strings.TrimSpace(attribute.Default))
	}
	return lines
}

func indentNoteLines(lines []string) []string {
	indented := make([]string, 0, len(lines))
	for _, line := range lines {
		indented = append(indented, "  "+line)
	}
	return indented
}

func compactDiagramNoteLines(lines []string) []string {
	compacted := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == "" {
			continue
		}
		compacted = append(compacted, line)
	}
	return compacted
}

var diagramAliasPattern = regexp.MustCompile(`[^A-Za-z0-9_]`)

func diagramAlias(value string) string {
	trimmed := strings.Trim(value, "/")
	if trimmed == "" {
		trimmed = "model"
	}
	alias := diagramAliasPattern.ReplaceAllString(trimmed, "_")
	alias = strings.Trim(alias, "_")
	if alias == "" {
		alias = "state"
	}
	if alias[0] >= '0' && alias[0] <= '9' {
		alias = "s_" + alias
	}
	return alias
}

func mermaidLine(value string) string {
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "`", "'")
	return strings.TrimSpace(value)
}

func plantUMLLine(value string) string {
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "@enduml", "@ enduml")
	return strings.TrimSpace(value)
}
