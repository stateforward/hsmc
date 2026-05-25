package hsmc

import (
	"fmt"
	"path"
	"strings"
)

type Diagnostic struct {
	Message string      `json:"message"`
	Range   SourceRange `json:"range,omitempty"`
}

type Diagnostics []Diagnostic

func (diagnostics Diagnostics) Error() string {
	if len(diagnostics) == 0 {
		return ""
	}
	if len(diagnostics) == 1 {
		return diagnostics[0].Message
	}
	return fmt.Sprintf("%s and %d more diagnostics", diagnostics[0].Message, len(diagnostics)-1)
}

func Validate(program *Program) error {
	if program == nil {
		return Diagnostics{{Message: "program is nil"}}
	}
	validator := programValidator{
		globalIDs:        map[string]SourceRange{},
		behaviorInlines:  map[string]bool{},
		behaviorKinds:    map[string]BehaviorKind{},
		behaviorOwners:   map[string]string{},
		behaviorRanges:   map[string]SourceRange{},
		behaviorTriggers: map[string]TriggerKind{},
		eventIDs:         map[string]SourceRange{},
		eventNames:       map[string]SourceRange{},
		eventSymbols:     map[string]SourceRange{},
		modelNames:       map[string]SourceRange{},
		nodeIDs:          map[string]SourceRange{},
		ownerPaths:       map[string]string{},
		stateKinds:       map[string]StateKind{},
		statePaths:       map[string]SourceRange{},
	}
	validator.validateProgramLanguages(program)
	for index, imp := range program.Imports {
		validator.validateImport(imp, fmt.Sprintf("imports[%d]", index))
	}
	for index, global := range program.Globals {
		validator.validateGlobal(global, index)
	}
	for index, behavior := range program.Behaviors {
		validator.validateBehaviorDefinition(behavior, index)
	}
	for index, event := range program.Events {
		validator.validateEventDecl(event, index)
	}
	for modelIndex, model := range program.Models {
		validator.collectModel(model)
		validator.validateModelLanguages(model, fmt.Sprintf("models[%d]", modelIndex))
	}
	for _, behavior := range program.Behaviors {
		validator.validateBehaviorOwner(behavior)
	}
	for _, model := range program.Models {
		validator.validateModel(model)
	}
	if len(validator.diagnostics) > 0 {
		return validator.diagnostics
	}
	return nil
}

type programValidator struct {
	diagnostics      Diagnostics
	globalIDs        map[string]SourceRange
	behaviorInlines  map[string]bool
	behaviorKinds    map[string]BehaviorKind
	behaviorOwners   map[string]string
	behaviorRanges   map[string]SourceRange
	behaviorTriggers map[string]TriggerKind
	eventIDs         map[string]SourceRange
	eventNames       map[string]SourceRange
	eventSymbols     map[string]SourceRange
	modelNames       map[string]SourceRange
	nodeIDs          map[string]SourceRange
	ownerPaths       map[string]string
	stateKinds       map[string]StateKind
	statePaths       map[string]SourceRange
}

func (validator *programValidator) validateProgramLanguages(program *Program) {
	validator.validateImplementationLanguage(program.SourceLanguage, "source_language", SourceRange{})
}

func (validator *programValidator) validateGlobal(global CodeBlock, index int) {
	validator.validateImplementationLanguage(global.Language, fmt.Sprintf("globals[%d].language", index), global.Range)
	for importIndex, imp := range global.Imports {
		validator.validateImport(imp, fmt.Sprintf("globals[%d].imports[%d]", index, importIndex))
		validator.validateRegionImportLanguage(imp, global.Language, fmt.Sprintf("global %q", global.ID), global.Range)
	}
	if strings.TrimSpace(global.ID) == "" {
		validator.add("global id is empty", global.Range)
		return
	}
	if global.ID != strings.TrimSpace(global.ID) {
		validator.add(fmt.Sprintf("global id %q has leading or trailing whitespace", global.ID), global.Range)
		return
	}
	if previous, exists := validator.globalIDs[global.ID]; exists {
		validator.add(fmt.Sprintf("duplicate global id %q", global.ID), previous)
		validator.add(fmt.Sprintf("duplicate global id %q", global.ID), global.Range)
		return
	}
	validator.globalIDs[global.ID] = global.Range
}

func (validator *programValidator) validateBehaviorDefinition(behavior Behavior, index int) {
	validator.validateImplementationLanguage(behavior.SourceLanguage, fmt.Sprintf("behaviors[%d].source_language", index), behavior.Range)
	validator.validateSupportedLanguage(behavior.TargetLanguage, fmt.Sprintf("behaviors[%d].target_language", index), behavior.Range)
	if behavior.TargetABI != nil {
		if behavior.TargetABI.Language == "" {
			validator.add(fmt.Sprintf("behavior %q target_abi.language is empty", behavior.ID), behavior.Range)
		}
		if strings.TrimSpace(behavior.TargetABI.Signature) == "" {
			validator.add(fmt.Sprintf("behavior %q target_abi.signature is empty", behavior.ID), behavior.Range)
		} else if behavior.TargetABI.Signature != strings.TrimSpace(behavior.TargetABI.Signature) {
			validator.add(fmt.Sprintf("behavior %q target_abi.signature has leading or trailing whitespace", behavior.ID), behavior.Range)
		}
		if behavior.TargetABI.ReturnType != "" {
			if strings.TrimSpace(behavior.TargetABI.ReturnType) == "" {
				validator.add(fmt.Sprintf("behavior %q target_abi.return_type is empty", behavior.ID), behavior.Range)
			} else if behavior.TargetABI.ReturnType != strings.TrimSpace(behavior.TargetABI.ReturnType) {
				validator.add(fmt.Sprintf("behavior %q target_abi.return_type %q has leading or trailing whitespace", behavior.ID, behavior.TargetABI.ReturnType), behavior.Range)
			}
		}
		validator.validateBehaviorABIParameters(behavior)
		validator.validateSupportedLanguage(behavior.TargetABI.Language, fmt.Sprintf("behaviors[%d].target_abi.language", index), behavior.Range)
		if behavior.TargetLanguage != "" && behavior.TargetABI.Language != "" && behavior.TargetABI.Language != behavior.TargetLanguage {
			validator.add(fmt.Sprintf("behavior %q target_abi.language %q does not match target_language %q", behavior.ID, behavior.TargetABI.Language, behavior.TargetLanguage), behavior.Range)
		}
	}
	if behavior.TargetLanguage != "" && behavior.TargetLanguage != behavior.SourceLanguage && strings.TrimSpace(behavior.Body) == "" && strings.TrimSpace(behavior.Code) == "" {
		validator.add(fmt.Sprintf("behavior %q target_language %q requires target body or code", behavior.ID, behavior.TargetLanguage), behavior.Range)
	}
	for importIndex, imp := range behavior.Imports {
		validator.validateImport(imp, fmt.Sprintf("behaviors[%d].imports[%d]", index, importIndex))
		validator.validateRegionImportLanguage(imp, behaviorImportOwnerLanguage(behavior), fmt.Sprintf("behavior %q", behavior.ID), behavior.Range)
	}
	if strings.TrimSpace(behavior.ID) == "" {
		validator.add("behavior id is empty", behavior.Range)
		return
	}
	if behavior.ID != strings.TrimSpace(behavior.ID) {
		validator.add(fmt.Sprintf("behavior id %q has leading or trailing whitespace", behavior.ID), behavior.Range)
		return
	}
	if previous, exists := validator.behaviorRanges[behavior.ID]; exists {
		validator.add(fmt.Sprintf("duplicate behavior id %q", behavior.ID), previous)
		validator.add(fmt.Sprintf("duplicate behavior id %q", behavior.ID), behavior.Range)
		return
	}
	if !isValidBehaviorKind(behavior.Kind) {
		validator.add(fmt.Sprintf("behavior %q has invalid kind %q", behavior.ID, behavior.Kind), behavior.Range)
	}
	if behavior.TriggerKind != "" && !isValidTriggerKind(behavior.TriggerKind) {
		validator.add(fmt.Sprintf("behavior %q has invalid trigger_kind %q", behavior.ID, behavior.TriggerKind), behavior.Range)
	}
	if behavior.TriggerKind != "" && behavior.Kind != BehaviorTrigger {
		validator.add(fmt.Sprintf("behavior %q has trigger_kind %q but kind %q", behavior.ID, behavior.TriggerKind, behavior.Kind), behavior.Range)
	}
	validator.behaviorKinds[behavior.ID] = behavior.Kind
	validator.behaviorInlines[behavior.ID] = behavior.Inline
	validator.behaviorOwners[behavior.ID] = behavior.OwnerID
	validator.behaviorTriggers[behavior.ID] = behavior.TriggerKind
	validator.behaviorRanges[behavior.ID] = behavior.Range
}

func behaviorImportOwnerLanguage(behavior Behavior) Language {
	if behavior.TargetLanguage != "" {
		return behavior.TargetLanguage
	}
	return behavior.SourceLanguage
}

func (validator *programValidator) validateRegionImportLanguage(imp Import, ownerLanguage Language, owner string, sourceRange SourceRange) {
	if ownerLanguage == "" || imp.Language == "" || imp.Language == ownerLanguage {
		return
	}
	if isCompilerOwnedImportPath(imp.Path, imp.Language) {
		return
	}
	validator.add(fmt.Sprintf("%s import %q language %q does not match region language %q", owner, imp.Path, imp.Language, ownerLanguage), sourceRange)
}

func (validator *programValidator) validateBehaviorABIParameters(behavior Behavior) {
	if behavior.TargetABI == nil {
		return
	}
	seen := map[string]bool{}
	for index, parameter := range behavior.TargetABI.Parameters {
		field := fmt.Sprintf("behavior %q target_abi.parameters[%d]", behavior.ID, index)
		name := strings.TrimSpace(parameter.Name)
		if name == "" {
			validator.add(field+" name is empty", behavior.Range)
		} else if parameter.Name != name {
			validator.add(fmt.Sprintf("%s name %q has leading or trailing whitespace", field, parameter.Name), behavior.Range)
		} else if seen[parameter.Name] {
			validator.add(fmt.Sprintf("behavior %q target_abi contains duplicate parameter %q", behavior.ID, parameter.Name), behavior.Range)
		} else {
			seen[parameter.Name] = true
		}
		if strings.TrimSpace(parameter.Type) == "" {
			validator.add(field+" type is empty", behavior.Range)
		} else if parameter.Type != strings.TrimSpace(parameter.Type) {
			validator.add(fmt.Sprintf("%s type %q has leading or trailing whitespace", field, parameter.Type), behavior.Range)
		}
	}
}

func (validator *programValidator) validateImport(imp Import, field string) {
	validator.validateImplementationLanguage(imp.Language, field+".language", imp.Range)
	if strings.TrimSpace(imp.Path) == "" {
		validator.add(fmt.Sprintf("%s path is empty", field), imp.Range)
		return
	}
	language := imp.Language
	if language != "" {
		canonical, ok := ParseLanguage(string(language))
		if !ok || canonical != language || language == LanguageJSONIR {
			return
		}
	}
	if err := validateImportShape(imp, language, field); err != nil {
		validator.add(err.Error(), imp.Range)
	}
}

func (validator *programValidator) validateBehaviorOwner(behavior Behavior) {
	if behavior.ID == "" {
		return
	}
	if strings.TrimSpace(behavior.OwnerID) == "" {
		validator.add(fmt.Sprintf("behavior %q has empty owner_id", behavior.ID), behavior.Range)
		return
	}
	if behavior.OwnerID != strings.TrimSpace(behavior.OwnerID) {
		validator.add(fmt.Sprintf("behavior %q owner_id %q has leading or trailing whitespace", behavior.ID, behavior.OwnerID), behavior.Range)
		return
	}
	if _, ok := validator.ownerPaths[behavior.OwnerID]; !ok {
		validator.add(fmt.Sprintf("behavior %q has unknown owner_id %q", behavior.ID, behavior.OwnerID), behavior.Range)
	}
}

type modelTriggerSymbols struct {
	attributes map[string]SourceRange
	operations map[string]SourceRange
}

func (validator *programValidator) add(message string, sourceRange SourceRange) {
	validator.diagnostics = append(validator.diagnostics, Diagnostic{Message: message, Range: sourceRange})
}

func (validator *programValidator) registerNodeID(kind string, id string, sourceRange SourceRange) bool {
	if strings.TrimSpace(id) == "" {
		validator.add(fmt.Sprintf("%s id is empty", kind), sourceRange)
		return false
	}
	if id != strings.TrimSpace(id) {
		validator.add(fmt.Sprintf("%s id %q has leading or trailing whitespace", kind, id), sourceRange)
		return false
	}
	if previous, exists := validator.nodeIDs[id]; exists {
		validator.add(fmt.Sprintf("duplicate %s id %q", kind, id), previous)
		validator.add(fmt.Sprintf("duplicate %s id %q", kind, id), sourceRange)
		return false
	}
	validator.nodeIDs[id] = sourceRange
	return true
}

func (validator *programValidator) validateEventDecl(event EventDecl, index int) {
	validator.validateImplementationLanguage(event.Language, fmt.Sprintf("events[%d].language", index), event.Range)
	if strings.TrimSpace(event.ID) == "" {
		validator.add("event id is empty", event.Range)
	} else if event.ID != strings.TrimSpace(event.ID) {
		validator.add(fmt.Sprintf("event id %q has leading or trailing whitespace", event.ID), event.Range)
	} else if previous, exists := validator.eventIDs[event.ID]; exists {
		validator.add(fmt.Sprintf("duplicate event id %q", event.ID), previous)
		validator.add(fmt.Sprintf("duplicate event id %q", event.ID), event.Range)
	} else {
		validator.eventIDs[event.ID] = event.Range
	}
	if strings.TrimSpace(event.Symbol) == "" {
		validator.add("event symbol is empty", event.Range)
	} else if event.Symbol != strings.TrimSpace(event.Symbol) {
		validator.add(fmt.Sprintf("event symbol %q has leading or trailing whitespace", event.Symbol), event.Range)
	} else if previous, exists := validator.eventSymbols[event.Symbol]; exists {
		validator.add(fmt.Sprintf("duplicate event symbol %q", event.Symbol), previous)
		validator.add(fmt.Sprintf("duplicate event symbol %q", event.Symbol), event.Range)
	} else {
		validator.eventSymbols[event.Symbol] = event.Range
	}
	if strings.TrimSpace(event.Name) == "" {
		validator.add(fmt.Sprintf("event %q has empty name", event.Symbol), event.Range)
		return
	}
	if event.Name != strings.TrimSpace(event.Name) {
		validator.add(fmt.Sprintf("event name %q has leading or trailing whitespace", event.Name), event.Range)
		return
	}
	if previous, exists := validator.eventNames[event.Name]; exists {
		validator.add(fmt.Sprintf("duplicate event name %q", event.Name), previous)
		validator.add(fmt.Sprintf("duplicate event name %q", event.Name), event.Range)
		return
	}
	validator.eventNames[event.Name] = event.Range
}

func (validator *programValidator) validateModelLanguages(model Model, owner string) {
	for index, attribute := range model.Attributes {
		validator.validateImplementationLanguage(attribute.Language, fmt.Sprintf("%s.attributes[%d].language", owner, index), attribute.Range)
	}
	for index, transition := range model.Transitions {
		validator.validateTransitionLanguages(transition, fmt.Sprintf("%s.transitions[%d]", owner, index))
	}
	for index, state := range model.States {
		validator.validateStateLanguages(state, fmt.Sprintf("%s.states[%d]", owner, index))
	}
}

func (validator *programValidator) validateStateLanguages(state State, owner string) {
	for index, transition := range state.Transitions {
		validator.validateTransitionLanguages(transition, fmt.Sprintf("%s.transitions[%d]", owner, index))
	}
	for index, child := range state.States {
		validator.validateStateLanguages(child, fmt.Sprintf("%s.states[%d]", owner, index))
	}
}

func (validator *programValidator) validateTransitionLanguages(transition Transition, owner string) {
	if transition.Trigger == nil {
		return
	}
	validator.validateImplementationLanguage(transition.Trigger.Language, owner+".trigger.language", transition.Range)
	for index, value := range transition.Trigger.Values {
		validator.validateImplementationLanguage(value.Language, fmt.Sprintf("%s.trigger.values[%d].language", owner, index), transition.Range)
	}
}

func (validator *programValidator) validateImplementationLanguage(language Language, field string, sourceRange SourceRange) {
	validator.validateLanguage(language, field, sourceRange, false)
}

func (validator *programValidator) validateSupportedLanguage(language Language, field string, sourceRange SourceRange) {
	validator.validateLanguage(language, field, sourceRange, true)
}

func (validator *programValidator) validateLanguage(language Language, field string, sourceRange SourceRange, allowTransport bool) {
	if language == "" {
		return
	}
	canonical, ok := ParseLanguage(string(language))
	if !ok {
		validator.add(fmt.Sprintf("%s %q is not supported", field, language), sourceRange)
		return
	}
	if canonical != language {
		validator.add(fmt.Sprintf("%s %q must be canonical %q", field, language, canonical), sourceRange)
		return
	}
	if !allowTransport && language == LanguageJSONIR {
		validator.add(fmt.Sprintf("%s must be an implementation language, got %q", field, language), sourceRange)
	}
}

func (validator *programValidator) collectModel(model Model) {
	modelPath := "/" + model.Name
	if validator.registerNodeID("model", model.ID, model.Range) {
		validator.ownerPaths[model.ID] = modelPath
	}
	if model.Name != "" {
		if previous, exists := validator.modelNames[model.Name]; exists {
			validator.add(fmt.Sprintf("duplicate model name %q", model.Name), previous)
			validator.add(fmt.Sprintf("duplicate model name %q", model.Name), model.Range)
		} else {
			validator.modelNames[model.Name] = model.Range
		}
	}
	seen := map[string]SourceRange{}
	for _, state := range model.States {
		validator.collectState(state, modelPath, seen)
	}
}

func (validator *programValidator) collectState(state State, ownerPath string, siblingNames map[string]SourceRange) {
	if previous, exists := siblingNames[state.Name]; exists {
		validator.add(fmt.Sprintf("duplicate state %q under %s", state.Name, ownerPath), previous)
		validator.add(fmt.Sprintf("duplicate state %q under %s", state.Name, ownerPath), state.Range)
	}
	siblingNames[state.Name] = state.Range
	statePath := path.Join(ownerPath, state.Name)
	if !strings.HasPrefix(statePath, "/") {
		statePath = "/" + statePath
	}
	if validator.registerNodeID("state", state.ID, state.Range) {
		validator.ownerPaths[state.ID] = statePath
	}
	validator.stateKinds[statePath] = state.Kind
	validator.statePaths[statePath] = state.Range
	childNames := map[string]SourceRange{}
	for _, child := range state.States {
		validator.collectState(child, statePath, childNames)
	}
}

func (validator *programValidator) validateModel(model Model) {
	if invalidSymbolicName(model.Name) {
		validator.add(fmt.Sprintf("invalid model name %q", model.Name), model.Range)
	}
	if model.Initializer == nil {
		validator.add(fmt.Sprintf("model %q is missing hsm.Initial", model.Name), model.Range)
	}
	symbols := modelTriggerSymbols{
		attributes: map[string]SourceRange{},
		operations: map[string]SourceRange{},
	}
	for _, attribute := range model.Attributes {
		validator.registerNodeID("attribute", attribute.ID, attribute.Range)
		if invalidSymbolicName(attribute.Name) {
			validator.add(fmt.Sprintf("invalid attribute name %q", attribute.Name), attribute.Range)
		}
		validator.validateAttributeValueFields(attribute)
		if previous, exists := symbols.attributes[attribute.Name]; exists {
			validator.add(fmt.Sprintf("duplicate attribute %q", attribute.Name), previous)
			validator.add(fmt.Sprintf("duplicate attribute %q", attribute.Name), attribute.Range)
		}
		symbols.attributes[attribute.Name] = attribute.Range
	}
	for _, operation := range model.Operations {
		validator.registerNodeID("operation", operation.ID, operation.Range)
		if invalidSymbolicName(operation.Name) {
			validator.add(fmt.Sprintf("invalid operation name %q", operation.Name), operation.Range)
		}
		if previous, exists := symbols.operations[operation.Name]; exists {
			validator.add(fmt.Sprintf("duplicate operation %q", operation.Name), previous)
			validator.add(fmt.Sprintf("duplicate operation %q", operation.Name), operation.Range)
		}
		symbols.operations[operation.Name] = operation.Range
		if operation.Behavior != nil {
			validator.validateBehaviorRef(*operation.Behavior, symbols, model.ID, operation.Range, BehaviorOperation)
			if operation.Behavior.ID != "" && validator.behaviorInlines[operation.Behavior.ID] {
				validator.add(fmt.Sprintf("operation %q implementation must be a callable reference, not an inline anonymous callable", operation.Name), operation.Range)
			}
		}
	}
	if model.Initializer != nil {
		validator.validateInitial(*model.Initializer, model.ID, symbols)
	}
	for _, transition := range model.Transitions {
		validator.validateTransition(transition, model.ID, symbols)
	}
	for _, state := range model.States {
		validator.validateState(state, symbols)
	}
}

func (validator *programValidator) validateAttributeValueFields(attribute Attribute) {
	if attribute.Type != "" {
		if strings.TrimSpace(attribute.Type) == "" {
			validator.add(fmt.Sprintf("attribute %q type is empty", attribute.Name), attribute.Range)
		} else if attribute.Type != strings.TrimSpace(attribute.Type) {
			validator.add(fmt.Sprintf("attribute %q type %q has leading or trailing whitespace", attribute.Name, attribute.Type), attribute.Range)
		}
	}
	if !attribute.HasDefault {
		return
	}
	if strings.TrimSpace(attribute.Default) == "" {
		validator.add(fmt.Sprintf("attribute %q default is empty", attribute.Name), attribute.Range)
		return
	}
	if attribute.Default != strings.TrimSpace(attribute.Default) {
		validator.add(fmt.Sprintf("attribute %q default %q has leading or trailing whitespace", attribute.Name, attribute.Default), attribute.Range)
	}
}

func (validator *programValidator) validateState(state State, symbols modelTriggerSymbols) {
	if invalidSymbolicName(state.Name) {
		validator.add(fmt.Sprintf("invalid state name %q", state.Name), state.Range)
	}
	if !isValidStateKind(state.Kind) {
		validator.add(fmt.Sprintf("state %q has invalid kind %q", state.Name, state.Kind), state.Range)
	}
	if state.Kind == StateKindFinal {
		if state.Initializer != nil || len(state.Transitions) > 0 || len(state.Behaviors) > 0 || len(state.States) > 0 || len(state.Defers) > 0 {
			validator.add(fmt.Sprintf("final state %q must not own behaviors, transitions, defers, child states, or initializers", state.Name), state.Range)
		}
	}
	if state.Kind == StateKindChoice || state.Kind == StateKindShallowHistory || state.Kind == StateKindDeepHistory {
		if state.Initializer != nil || len(state.Behaviors) > 0 || len(state.States) > 0 || len(state.Defers) > 0 {
			validator.add(fmt.Sprintf("%s %q must not own behaviors, defers, child states, or initializers", state.Kind, state.Name), state.Range)
		}
	}
	if state.Kind == StateKindChoice && len(state.Transitions) == 0 {
		validator.add(fmt.Sprintf("%s %q must have at least one transition", state.Kind, state.Name), state.Range)
	}
	if state.Kind == StateKindShallowHistory || state.Kind == StateKindDeepHistory {
		if len(state.Transitions) == 0 {
			validator.add(fmt.Sprintf("%s %q must have a default transition", state.Kind, state.Name), state.Range)
		} else if len(state.Transitions) > 1 {
			validator.add(fmt.Sprintf("%s %q must have exactly one default transition", state.Kind, state.Name), state.Range)
		}
		for _, transition := range state.Transitions {
			if strings.TrimSpace(transition.Target) == "" {
				validator.add(fmt.Sprintf("%s %q default transition %q is missing target", state.Kind, state.Name, transition.ID), transition.Range)
			}
			if transition.Source != "" {
				validator.add(fmt.Sprintf("%s %q default transition %q must not set source", state.Kind, state.Name, transition.ID), transition.Range)
			}
			if transition.Trigger != nil {
				validator.add(fmt.Sprintf("%s %q default transition %q must not set trigger", state.Kind, state.Name, transition.ID), transition.Range)
			}
		}
	}
	if state.Kind == StateKindChoice {
		last := len(state.Transitions) - 1
		if last >= 0 && state.Transitions[last].Guard != nil {
			validator.add(fmt.Sprintf("choice %q should end with an unguarded fallback transition", state.Name), state.Transitions[last].Range)
		}
	}
	for _, behavior := range state.Behaviors {
		validator.validateBehaviorRef(behavior, symbols, state.ID, state.Range, BehaviorEntry, BehaviorExit, BehaviorActivity)
	}
	for _, eventName := range state.Defers {
		if strings.TrimSpace(eventName) == "" {
			validator.add(fmt.Sprintf("state %q has empty deferred event", state.Name), state.Range)
		} else if eventName != strings.TrimSpace(eventName) {
			validator.add(fmt.Sprintf("state %q deferred event %q has leading or trailing whitespace", state.Name, eventName), state.Range)
		}
	}
	if state.Initializer != nil {
		validator.validateInitial(*state.Initializer, state.ID, symbols)
	}
	for _, transition := range state.Transitions {
		validator.validateTransition(transition, state.ID, symbols)
	}
	for _, child := range state.States {
		validator.validateState(child, symbols)
	}
}

func (validator *programValidator) validateInitial(initial Initial, expectedOwnerID string, symbols modelTriggerSymbols) {
	validator.registerNodeID("initial", initial.ID, initial.Range)
	validator.validateStructuralOwner("initial", initial.ID, initial.OwnerID, expectedOwnerID, initial.Range)
	if strings.TrimSpace(initial.Target) == "" {
		validator.add("initial transition is missing target", initial.Range)
	} else if initial.Target != strings.TrimSpace(initial.Target) {
		validator.add(fmt.Sprintf("initial transition target %q has leading or trailing whitespace", initial.Target), initial.Range)
	} else {
		validator.validateTarget(initial.OwnerID, initial.Target, initial.Range)
	}
	for _, effect := range initial.Effects {
		validator.validateBehaviorRef(effect, symbols, initial.OwnerID, initial.Range, BehaviorEffect)
	}
}

func (validator *programValidator) validateTransition(transition Transition, expectedOwnerID string, symbols modelTriggerSymbols) {
	validator.registerNodeID("transition", transition.ID, transition.Range)
	validator.validateStructuralOwner("transition", transition.ID, transition.OwnerID, expectedOwnerID, transition.Range)
	if transition.Source != "" {
		if strings.TrimSpace(transition.Source) == "" {
			validator.add(fmt.Sprintf("transition %q source is empty", transition.ID), transition.Range)
		} else if transition.Source != strings.TrimSpace(transition.Source) {
			validator.add(fmt.Sprintf("transition %q source %q has leading or trailing whitespace", transition.ID, transition.Source), transition.Range)
		} else {
			validator.validateSource(transition.OwnerID, transition.Source, transition.Range)
		}
	}
	if transition.Target != "" {
		if strings.TrimSpace(transition.Target) == "" {
			validator.add(fmt.Sprintf("transition %q target is empty", transition.ID), transition.Range)
		} else if transition.Target != strings.TrimSpace(transition.Target) {
			validator.add(fmt.Sprintf("transition %q target %q has leading or trailing whitespace", transition.ID, transition.Target), transition.Range)
		} else {
			validator.validateTarget(transition.OwnerID, transition.Target, transition.Range)
		}
	}
	if transition.Trigger != nil {
		validator.validateTrigger(*transition.Trigger, symbols, transition.Range)
	}
	if transition.Guard != nil {
		validator.validateBehaviorRef(*transition.Guard, symbols, transition.OwnerID, transition.Range, BehaviorGuard)
	}
	if transition.Trigger != nil && transition.Trigger.Expr != nil {
		validator.validateBehaviorRef(*transition.Trigger.Expr, symbols, transition.OwnerID, transition.Range, BehaviorTrigger)
		validator.validateTriggerBehaviorKind(*transition.Trigger.Expr, transition.Trigger.Kind, transition.Range)
	}
	for _, effect := range transition.Effects {
		validator.validateBehaviorRef(effect, symbols, transition.OwnerID, transition.Range, BehaviorEffect)
	}
}

func (validator *programValidator) validateTriggerBehaviorKind(ref BehaviorRef, triggerKind TriggerKind, sourceRange SourceRange) {
	if strings.TrimSpace(ref.ID) == "" || ref.ID != strings.TrimSpace(ref.ID) {
		return
	}
	behaviorTriggerKind, ok := validator.behaviorTriggers[ref.ID]
	if !ok || behaviorTriggerKind == "" {
		return
	}
	if behaviorTriggerKind != triggerKind {
		validator.add(fmt.Sprintf("trigger behavior %q has trigger_kind %q but reference is owned by %q trigger", ref.ID, behaviorTriggerKind, triggerKind), sourceRange)
	}
}

func (validator *programValidator) validateStructuralOwner(kind string, id string, ownerID string, expectedOwnerID string, sourceRange SourceRange) {
	if strings.TrimSpace(ownerID) == "" {
		validator.add(fmt.Sprintf("%s %q has empty owner_id", kind, id), sourceRange)
		return
	}
	if ownerID != strings.TrimSpace(ownerID) {
		validator.add(fmt.Sprintf("%s %q owner_id %q has leading or trailing whitespace", kind, id, ownerID), sourceRange)
		return
	}
	if expectedOwnerID != "" && ownerID != expectedOwnerID {
		validator.add(fmt.Sprintf("%s %q has owner_id %q but is contained by %q", kind, id, ownerID, expectedOwnerID), sourceRange)
	}
	if _, ok := validator.ownerPaths[ownerID]; !ok {
		validator.add(fmt.Sprintf("%s %q has unknown owner_id %q", kind, id, ownerID), sourceRange)
	}
}

func (validator *programValidator) validateTrigger(trigger Trigger, symbols modelTriggerSymbols, sourceRange SourceRange) {
	if !isValidTriggerKind(trigger.Kind) {
		validator.add(fmt.Sprintf("trigger has invalid kind %q", trigger.Kind), sourceRange)
		return
	}
	validator.validateTriggerValues(trigger, sourceRange)
	if trigger.Expr != nil {
		switch trigger.Kind {
		case TriggerOn, TriggerOnSet, TriggerOnCall:
			validator.add(fmt.Sprintf("%s trigger must not use a behavior expression", trigger.Kind), sourceRange)
		}
		if trigger.IsString || len(trigger.Values) > 0 {
			validator.add(fmt.Sprintf("%s trigger must not combine a behavior expression with trigger value metadata", trigger.Kind), sourceRange)
		}
	}
	validator.validateTriggerExpressionPresence(trigger, sourceRange)
	if trigger.Kind == TriggerOn {
		validator.validateOnTriggerValues(trigger, sourceRange)
	}
	switch trigger.Kind {
	case TriggerOnCall:
		validator.validateNamedTrigger("operation", trigger.Value, symbols.operations, sourceRange)
	case TriggerOnSet:
		validator.validateNamedTrigger("attribute", trigger.Value, symbols.attributes, sourceRange)
	case TriggerWhen, TriggerAfter, TriggerEvery, TriggerAt:
		if trigger.IsString {
			validator.validateNamedTrigger("attribute", trigger.Value, symbols.attributes, sourceRange)
		}
	}
}

func (validator *programValidator) validateTriggerExpressionPresence(trigger Trigger, sourceRange SourceRange) {
	if trigger.Expr != nil || len(trigger.Values) > 0 || strings.TrimSpace(trigger.Value) != "" {
		return
	}
	switch trigger.Kind {
	case TriggerAfter, TriggerEvery, TriggerAt, TriggerWhen:
		validator.add(fmt.Sprintf("%s trigger is missing expression", trigger.Kind), sourceRange)
	}
}

func (validator *programValidator) validateTriggerValues(trigger Trigger, sourceRange SourceRange) {
	if trigger.Kind != TriggerOn && len(trigger.Values) > 0 {
		validator.add(fmt.Sprintf("%s trigger must not use an event values list", trigger.Kind), sourceRange)
	}
	if trigger.Kind == TriggerOn && len(trigger.Values) > 0 && trigger.Value != "" && trigger.Value != trigger.Values[0].Value {
		validator.add(fmt.Sprintf("on trigger value %q does not match first event value %q", trigger.Value, trigger.Values[0].Value), sourceRange)
	}
	values := trigger.Values
	if len(values) == 0 {
		values = []TriggerValue{{Value: trigger.Value, IsString: trigger.IsString, EventSymbol: "", Language: trigger.Language}}
	}
	for _, value := range values {
		if value.EventSymbol != "" {
			if trigger.Kind != TriggerOn {
				validator.add(fmt.Sprintf("%s trigger must not use event symbol metadata", trigger.Kind), sourceRange)
				continue
			}
			if value.IsString {
				validator.add(fmt.Sprintf("event reference %q must not also be marked as a string literal", value.EventSymbol), sourceRange)
				continue
			}
			if strings.TrimSpace(value.EventSymbol) == "" {
				validator.add("event reference is empty", sourceRange)
				continue
			}
			if value.EventSymbol != strings.TrimSpace(value.EventSymbol) {
				validator.add(fmt.Sprintf("event reference %q has leading or trailing whitespace", value.EventSymbol), sourceRange)
				continue
			}
			if _, ok := validator.eventSymbols[value.EventSymbol]; !ok {
				validator.add(fmt.Sprintf("event reference %q does not reference a declared event", value.EventSymbol), sourceRange)
			}
		}
	}
}

func (validator *programValidator) validateOnTriggerValues(trigger Trigger, sourceRange SourceRange) {
	values := trigger.Values
	if len(values) == 0 {
		values = []TriggerValue{{Value: trigger.Value, IsString: trigger.IsString, EventSymbol: "", Language: trigger.Language}}
	}
	for _, value := range values {
		if strings.TrimSpace(value.Value) == "" && strings.TrimSpace(value.EventSymbol) == "" {
			validator.add("on trigger has empty event", sourceRange)
			continue
		}
		if value.Value != "" && value.Value != strings.TrimSpace(value.Value) {
			validator.add(fmt.Sprintf("on trigger event %q has leading or trailing whitespace", value.Value), sourceRange)
		}
	}
}

func (validator *programValidator) validateNamedTrigger(kind string, name string, known map[string]SourceRange, sourceRange SourceRange) {
	if invalidSymbolicName(name) {
		validator.add(fmt.Sprintf("invalid %s trigger name %q", kind, name), sourceRange)
		return
	}
	if _, ok := known[name]; !ok {
		validator.add(fmt.Sprintf("%s trigger %q does not reference a model %s", kind, name, kind), sourceRange)
	}
}

func (validator *programValidator) validateBehaviorRef(ref BehaviorRef, symbols modelTriggerSymbols, expectedOwnerID string, sourceRange SourceRange, allowedKinds ...BehaviorKind) {
	validKind := isValidBehaviorKind(ref.Kind)
	if !validKind {
		validator.add(fmt.Sprintf("behavior reference has invalid kind %q", ref.Kind), sourceRange)
	}
	if validKind && len(allowedKinds) > 0 && !behaviorKindAllowed(ref.Kind, allowedKinds) {
		validator.add(fmt.Sprintf("behavior reference kind %q is not valid in this position", ref.Kind), sourceRange)
	}
	if ref.Operation != "" {
		if ref.ID != "" {
			validator.add(fmt.Sprintf("behavior reference sets both id %q and operation %q", ref.ID, ref.Operation), sourceRange)
		}
		validator.validateOperationReference(ref.Operation, symbols.operations, sourceRange)
		return
	}
	if strings.TrimSpace(ref.ID) == "" {
		validator.add("behavior reference has empty id", sourceRange)
		return
	}
	if ref.ID != strings.TrimSpace(ref.ID) {
		validator.add(fmt.Sprintf("behavior reference id %q has leading or trailing whitespace", ref.ID), sourceRange)
		return
	}
	kind, ok := validator.behaviorKinds[ref.ID]
	if !ok {
		validator.add(fmt.Sprintf("unknown behavior %q", ref.ID), sourceRange)
		return
	}
	if kind != ref.Kind {
		validator.add(fmt.Sprintf("behavior %q has kind %q but reference expects %q", ref.ID, kind, ref.Kind), sourceRange)
	}
	if expectedOwnerID != "" {
		if ownerID := validator.behaviorOwners[ref.ID]; ownerID != "" && ownerID != expectedOwnerID {
			validator.add(fmt.Sprintf("behavior %q has owner_id %q but reference is owned by %q", ref.ID, ownerID, expectedOwnerID), sourceRange)
		}
	}
}

func behaviorKindAllowed(kind BehaviorKind, allowed []BehaviorKind) bool {
	for _, allowedKind := range allowed {
		if kind == allowedKind {
			return true
		}
	}
	return false
}

func (validator *programValidator) validateOperationReference(name string, known map[string]SourceRange, sourceRange SourceRange) {
	if invalidSymbolicName(name) {
		validator.add(fmt.Sprintf("invalid operation reference %q", name), sourceRange)
		return
	}
	if _, ok := known[name]; !ok {
		validator.add(fmt.Sprintf("operation reference %q does not reference a model operation", name), sourceRange)
	}
}

func (validator *programValidator) validateTarget(ownerID string, target string, sourceRange SourceRange) {
	if target == "" {
		return
	}
	resolved := validator.resolvePath(ownerID, target)
	if _, ok := validator.statePaths[resolved]; !ok {
		validator.add(fmt.Sprintf("target %q resolves to missing state %q", target, resolved), sourceRange)
	}
}

func (validator *programValidator) validateSource(ownerID string, source string, sourceRange SourceRange) {
	if source == "" {
		return
	}
	resolved := validator.resolvePath(ownerID, source)
	kind, ok := validator.stateKinds[resolved]
	if !ok {
		validator.add(fmt.Sprintf("source %q resolves to missing state %q", source, resolved), sourceRange)
		return
	}
	if kind == StateKindChoice || kind == StateKindShallowHistory || kind == StateKindDeepHistory {
		validator.add(fmt.Sprintf("source %q resolves to pseudostate %q", source, resolved), sourceRange)
	}
}

func (validator *programValidator) resolvePath(ownerID string, value string) string {
	if strings.HasPrefix(value, "/") {
		return path.Clean(value)
	}
	ownerPath := validator.ownerPaths[ownerID]
	if ownerPath == "" {
		ownerPath = "/"
	}
	resolved := path.Clean(path.Join(ownerPath, value))
	if !strings.HasPrefix(resolved, "/") {
		resolved = "/" + resolved
	}
	return resolved
}

func invalidSymbolicName(name string) bool {
	return strings.TrimSpace(name) == "" || name != strings.TrimSpace(name) || strings.Contains(name, "/")
}

func isValidStateKind(kind StateKind) bool {
	switch kind {
	case StateKindState, StateKindFinal, StateKindChoice, StateKindShallowHistory, StateKindDeepHistory:
		return true
	default:
		return false
	}
}

func isValidBehaviorKind(kind BehaviorKind) bool {
	switch kind {
	case BehaviorEntry, BehaviorExit, BehaviorActivity, BehaviorGuard, BehaviorEffect, BehaviorTrigger, BehaviorOperation:
		return true
	default:
		return false
	}
}

func isValidTriggerKind(kind TriggerKind) bool {
	switch kind {
	case TriggerOn, TriggerOnSet, TriggerOnCall, TriggerAfter, TriggerEvery, TriggerAt, TriggerWhen:
		return true
	default:
		return false
	}
}
