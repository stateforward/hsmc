package hsmc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type JSONIRBackend struct{}

func NewJSONIRBackend() JSONIRBackend {
	return JSONIRBackend{}
}

func (JSONIRBackend) Language() Language { return LanguageJSONIR }

func (JSONIRBackend) Emit(_ context.Context, program *Program) ([]byte, error) {
	transport := cloneProgram(program)
	for index := range transport.Behaviors {
		retargetJSONIRBehaviorSourceContext(&transport.Behaviors[index])
		transport.Behaviors[index].TargetLanguage = ""
		transport.Behaviors[index].TargetABI = nil
	}
	return json.MarshalIndent(transport, "", "  ")
}

func retargetJSONIRBehaviorSourceContext(behavior *Behavior) {
	if behavior == nil || behavior.TargetLanguage == "" || behavior.TargetLanguage == behavior.SourceLanguage {
		return
	}
	if strings.TrimSpace(behavior.Body) == "" && strings.TrimSpace(behavior.Code) == "" {
		return
	}
	behavior.SourceLanguage = behavior.TargetLanguage
	behavior.Inline = false
	behavior.Signature = ""
}

type JSONIRFrontend struct{}

func NewJSONIRFrontend() JSONIRFrontend {
	return JSONIRFrontend{}
}

func (JSONIRFrontend) Language() Language { return LanguageJSONIR }

func (JSONIRFrontend) Parse(_ context.Context, input SourceInput) (*Program, error) {
	var program Program
	if err := rejectDuplicateJSONFields(input.Data, "json-ir"); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(input.Data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&program); err != nil {
		return nil, err
	}
	if decoder.More() {
		return nil, fmt.Errorf("json-ir contains multiple JSON values")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return nil, fmt.Errorf("json-ir contains multiple JSON values")
	} else if err != io.EOF {
		return nil, err
	}
	if program.SourceLanguage == "" {
		return nil, fmt.Errorf("json-ir source_language is required")
	}
	sourceLanguage, ok := ParseLanguage(string(program.SourceLanguage))
	if !ok {
		return nil, fmt.Errorf("json-ir source_language %q is not supported", program.SourceLanguage)
	}
	if sourceLanguage == LanguageJSONIR {
		return nil, fmt.Errorf("json-ir source_language must be an implementation language, got %q", program.SourceLanguage)
	}
	program.SourceLanguage = sourceLanguage
	clearJSONIRTargetBehaviorMetadata(&program)
	if err := normalizeJSONIRLanguages(&program); err != nil {
		return nil, err
	}
	return &program, nil
}

func clearJSONIRTargetBehaviorMetadata(program *Program) {
	for index := range program.Behaviors {
		if program.Behaviors[index].TargetLanguage != "" || program.Behaviors[index].TargetABI != nil {
			program.Behaviors[index].Inline = false
			program.Behaviors[index].Signature = ""
		}
		program.Behaviors[index].TargetLanguage = ""
		program.Behaviors[index].TargetABI = nil
	}
}

func normalizeJSONIRLanguages(program *Program) error {
	for index := range program.Imports {
		if err := normalizeJSONIRImplementationLanguage(&program.Imports[index].Language, fmt.Sprintf("imports[%d].language", index)); err != nil {
			return err
		}
	}
	for index := range program.Globals {
		if err := normalizeJSONIRImplementationLanguage(&program.Globals[index].Language, fmt.Sprintf("globals[%d].language", index)); err != nil {
			return err
		}
		for importIndex := range program.Globals[index].Imports {
			if err := normalizeJSONIRImplementationLanguage(&program.Globals[index].Imports[importIndex].Language, fmt.Sprintf("globals[%d].imports[%d].language", index, importIndex)); err != nil {
				return err
			}
		}
	}
	for index := range program.Events {
		if err := normalizeJSONIRImplementationLanguage(&program.Events[index].Language, fmt.Sprintf("events[%d].language", index)); err != nil {
			return err
		}
	}
	for index := range program.Models {
		if err := normalizeJSONIRModelLanguages(&program.Models[index], fmt.Sprintf("models[%d]", index)); err != nil {
			return err
		}
	}
	for index := range program.Behaviors {
		if err := normalizeJSONIRImplementationLanguage(&program.Behaviors[index].SourceLanguage, fmt.Sprintf("behaviors[%d].source_language", index)); err != nil {
			return err
		}
		for importIndex := range program.Behaviors[index].Imports {
			if err := normalizeJSONIRImplementationLanguage(&program.Behaviors[index].Imports[importIndex].Language, fmt.Sprintf("behaviors[%d].imports[%d].language", index, importIndex)); err != nil {
				return err
			}
		}
	}
	return nil
}

func normalizeJSONIRModelLanguages(model *Model, owner string) error {
	for index := range model.Attributes {
		if err := normalizeJSONIRImplementationLanguage(&model.Attributes[index].Language, fmt.Sprintf("%s.attributes[%d].language", owner, index)); err != nil {
			return err
		}
	}
	for index := range model.Transitions {
		if err := normalizeJSONIRTransitionLanguages(&model.Transitions[index], fmt.Sprintf("%s.transitions[%d]", owner, index)); err != nil {
			return err
		}
	}
	for index := range model.States {
		if err := normalizeJSONIRStateLanguages(&model.States[index], fmt.Sprintf("%s.states[%d]", owner, index)); err != nil {
			return err
		}
	}
	return nil
}

func normalizeJSONIRStateLanguages(state *State, owner string) error {
	for index := range state.Transitions {
		if err := normalizeJSONIRTransitionLanguages(&state.Transitions[index], fmt.Sprintf("%s.transitions[%d]", owner, index)); err != nil {
			return err
		}
	}
	for index := range state.States {
		if err := normalizeJSONIRStateLanguages(&state.States[index], fmt.Sprintf("%s.states[%d]", owner, index)); err != nil {
			return err
		}
	}
	return nil
}

func normalizeJSONIRTransitionLanguages(transition *Transition, owner string) error {
	if transition.Trigger == nil {
		return nil
	}
	if err := normalizeJSONIRImplementationLanguage(&transition.Trigger.Language, owner+".trigger.language"); err != nil {
		return err
	}
	for index := range transition.Trigger.Values {
		if err := normalizeJSONIRImplementationLanguage(&transition.Trigger.Values[index].Language, fmt.Sprintf("%s.trigger.values[%d].language", owner, index)); err != nil {
			return err
		}
	}
	return nil
}

func normalizeJSONIRImplementationLanguage(value *Language, field string) error {
	if err := normalizeJSONIRSupportedLanguage(value, field); err != nil {
		return err
	}
	if *value == LanguageJSONIR {
		return fmt.Errorf("json-ir %s must be an implementation language, got %q", field, *value)
	}
	return nil
}

func normalizeJSONIRSupportedLanguage(value *Language, field string) error {
	if *value == "" {
		return nil
	}
	language, ok := ParseLanguage(string(*value))
	if !ok {
		return fmt.Errorf("json-ir %s %q is not supported", field, *value)
	}
	*value = language
	return nil
}
