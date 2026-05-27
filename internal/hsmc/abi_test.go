package hsmc

import (
	"context"
	"reflect"
	"testing"
)

func TestAssignTargetABIPopulatesBehaviorContracts(t *testing.T) {
	program, err := NewGoFrontend().Parse(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)})
	if err != nil {
		t.Fatal(err)
	}
	AssignTargetABI(program, LanguageTS)
	for _, behavior := range program.Behaviors {
		if behavior.TargetLanguage != "" {
			t.Fatalf("target language should remain adapter-owned, got %q", behavior.TargetLanguage)
		}
		if behavior.TargetABI == nil {
			t.Fatalf("missing target ABI for %#v", behavior)
		}
		if behavior.TargetABI.Language != LanguageTS {
			t.Fatalf("ABI language = %q", behavior.TargetABI.Language)
		}
		if behavior.Kind == BehaviorGuard && behavior.TargetABI.ReturnType != "boolean" {
			t.Fatalf("guard return type = %q", behavior.TargetABI.ReturnType)
		}
		if behavior.Kind != BehaviorGuard && behavior.TargetABI.ReturnType != "void" {
			t.Fatalf("behavior return type = %q", behavior.TargetABI.ReturnType)
		}
	}
}

func TestBehaviorABIsExposeStableUniqueParameters(t *testing.T) {
	languages := []Language{LanguageCSharp, LanguageCPP, LanguageDart, LanguageElixir, LanguageGo, LanguageJava, LanguageJS, LanguagePython, LanguageRust, LanguageTS, LanguageZig}
	kinds := []BehaviorKind{BehaviorEntry, BehaviorExit, BehaviorActivity, BehaviorGuard, BehaviorEffect, BehaviorOperation}
	triggerKinds := []TriggerKind{"", TriggerAfter, TriggerEvery, TriggerAt, TriggerWhen}

	for _, language := range languages {
		for _, kind := range kinds {
			assertBehaviorABIStable(t, BehaviorABIFor(language, kind))
		}
		for _, triggerKind := range triggerKinds {
			assertBehaviorABIStable(t, BehaviorABIForTrigger(language, BehaviorTrigger, triggerKind))
		}
	}
}

func TestSupportedImplementationTargetABIsMatchTargetLanguage(t *testing.T) {
	kinds := []BehaviorKind{BehaviorEntry, BehaviorExit, BehaviorActivity, BehaviorGuard, BehaviorEffect, BehaviorOperation}
	triggerKinds := []TriggerKind{"", TriggerAfter, TriggerEvery, TriggerAt, TriggerWhen}

	for _, language := range SupportedTargetLanguages() {
		if !isImplementationLanguage(language) {
			continue
		}
		t.Run(string(language), func(t *testing.T) {
			for _, kind := range kinds {
				abi := BehaviorABIFor(language, kind)
				if abi == nil || abi.Language != language {
					t.Fatalf("%s %s ABI = %#v, want language %s", language, kind, abi, language)
				}
			}
			for _, triggerKind := range triggerKinds {
				abi := BehaviorABIForTrigger(language, BehaviorTrigger, triggerKind)
				if abi == nil || abi.Language != language {
					t.Fatalf("%s trigger %s ABI = %#v, want language %s", language, triggerKind, abi, language)
				}
			}
		})
	}
}

func assertBehaviorABIStable(t *testing.T, abi *BehaviorABI) {
	t.Helper()
	if abi == nil {
		t.Fatal("nil behavior ABI")
	}
	if abi.Language == "" {
		t.Fatalf("ABI missing language: %#v", abi)
	}
	if abi.Signature == "" {
		t.Fatalf("%s ABI missing signature: %#v", abi.Language, abi)
	}
	seen := map[string]bool{}
	for _, parameter := range abi.Parameters {
		if parameter.Name == "" || parameter.Type == "" {
			t.Fatalf("%s ABI contains incomplete parameter: %#v", abi.Language, abi)
		}
		if seen[parameter.Name] {
			t.Fatalf("%s ABI contains duplicate parameter %q: %#v", abi.Language, parameter.Name, abi)
		}
		seen[parameter.Name] = true
	}
}

func TestAdapterRequestCarriesTargetABI(t *testing.T) {
	adapter := &capturingAdapter{}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)
	_, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(adapter.request.Behaviors) == 0 {
		t.Fatal("adapter received no behaviors")
	}
	for _, behavior := range adapter.request.Behaviors {
		if behavior.TargetABI == nil {
			t.Fatalf("adapter behavior missing ABI: %#v", behavior)
		}
		if behavior.TargetABI.Signature == "" || len(behavior.TargetABI.Parameters) != 3 {
			t.Fatalf("adapter behavior ABI incomplete: %#v", behavior.TargetABI)
		}
	}
}

func TestAdapterRequestCarriesTriggerSpecificTargetABI(t *testing.T) {
	adapter := &capturingAdapter{}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)
	_, _, err := compiler.Compile(context.Background(), SourceInput{Path: "timer.go", Data: []byte(timerSmokeGoSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageDart,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, behavior := range adapter.request.Behaviors {
		if behavior.Kind != BehaviorTrigger || behavior.TriggerKind != TriggerAt {
			continue
		}
		if behavior.TargetABI == nil || behavior.TargetABI.Language != LanguageDart || behavior.TargetABI.ReturnType != "FutureOr<DateTime>" {
			t.Fatalf("adapter trigger behavior ABI = %#v, want Dart At trigger ABI", behavior.TargetABI)
		}
		if behavior.TargetABI.Signature == "" || len(behavior.TargetABI.Parameters) != 3 {
			t.Fatalf("adapter trigger behavior ABI incomplete: %#v", behavior.TargetABI)
		}
		return
	}
	t.Fatalf("adapter request missing At trigger behavior: %#v", adapter.request.Behaviors)
}

func TestApplyBehaviorPatchPreservesTargetABI(t *testing.T) {
	program, err := NewGoFrontend().Parse(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)})
	if err != nil {
		t.Fatal(err)
	}
	AssignTargetABI(program, LanguageTS)
	originalABI := program.Behaviors[0].TargetABI
	err = applyBehaviorPatch(program, AdapterRequest{TargetLanguage: LanguageTS, Behaviors: program.Behaviors}, &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:   program.Behaviors[0].ID,
			Body: "translated();",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(program.Behaviors[0].TargetABI, originalABI) {
		t.Fatalf("target ABI was not preserved: %#v", program.Behaviors[0].TargetABI)
	}
	if program.Behaviors[0].TargetABI == originalABI {
		t.Fatalf("target ABI should be value-preserved without sharing mutable ABI pointer")
	}
}
