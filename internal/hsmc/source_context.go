package hsmc

// NormalizeSourceContext fills source-language metadata that older or
// hand-authored IR may omit. Import attribution and target backends rely on
// language tags to distinguish source context from target adapter output.
func NormalizeSourceContext(program *Program, fallback Language) {
	if program == nil {
		return
	}
	if program.SourceLanguage == "" {
		program.SourceLanguage = fallback
	}
	if program.SourceLanguage == "" {
		return
	}
	for index := range program.Imports {
		if program.Imports[index].Language == "" {
			program.Imports[index].Language = program.SourceLanguage
		}
	}
	for index := range program.Events {
		if program.Events[index].Language == "" {
			program.Events[index].Language = program.SourceLanguage
		}
	}
	for index := range program.Globals {
		if program.Globals[index].Language == "" {
			program.Globals[index].Language = program.SourceLanguage
		}
		normalizeImportLanguages(program.Globals[index].Imports, program.Globals[index].Language)
	}
	for index := range program.Models {
		normalizeModelSourceContext(&program.Models[index], program.SourceLanguage)
	}
	for index := range program.Behaviors {
		if program.Behaviors[index].SourceLanguage == "" {
			program.Behaviors[index].SourceLanguage = program.SourceLanguage
		}
		normalizeImportLanguages(program.Behaviors[index].Imports, program.Behaviors[index].SourceLanguage)
	}
}

func normalizeModelSourceContext(model *Model, language Language) {
	for index := range model.Attributes {
		if model.Attributes[index].Language == "" {
			model.Attributes[index].Language = language
		}
	}
	for index := range model.Transitions {
		normalizeTransitionSourceContext(&model.Transitions[index], language)
	}
	for index := range model.States {
		normalizeStateSourceContext(&model.States[index], language)
	}
}

func normalizeStateSourceContext(state *State, language Language) {
	for index := range state.Transitions {
		normalizeTransitionSourceContext(&state.Transitions[index], language)
	}
	for index := range state.States {
		normalizeStateSourceContext(&state.States[index], language)
	}
}

func normalizeTransitionSourceContext(transition *Transition, language Language) {
	if transition.Trigger != nil {
		normalizeTriggerSourceContext(transition.Trigger, language)
	}
}

func normalizeTriggerSourceContext(trigger *Trigger, language Language) {
	if trigger.Language == "" {
		trigger.Language = language
	}
	for index := range trigger.Values {
		if trigger.Values[index].Language == "" {
			trigger.Values[index].Language = language
		}
	}
}

func normalizeImportLanguages(imports []Import, language Language) {
	for index := range imports {
		if imports[index].Language == "" {
			imports[index].Language = language
		}
	}
}
