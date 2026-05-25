package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestDartBackendEmitsModelStructureAndCommentedForeignBehaviors(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageDart,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`import "package:hsm/hsm.dart";`,
		"FutureOr<void> entry1(Context ctx, Instance instance, Event event)",
		"// Original go behavior entry_1 preserved for manual porting:",
		`// sm.log = append(sm.log, record("closed"))`,
		"bool guard1(Context ctx, Instance instance, Event event)",
		"return false;",
		"final doorModel = define(",
		`initial(target("closed"))`,
		`state(`,
		`entry(entry1)`,
		`transition([`,
		`on("open")`,
		`guard(guard1)`,
		`effect(effect1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Dart output missing %q:\n%s", needle, text)
		}
	}
}

func TestDartTargetABI(t *testing.T) {
	entry := BehaviorABIFor(LanguageDart, BehaviorEntry)
	if entry == nil || entry.ReturnType != "FutureOr<void>" || !strings.Contains(entry.Signature, "FutureOr<void>") {
		t.Fatalf("Dart entry ABI = %#v", entry)
	}
	guard := BehaviorABIFor(LanguageDart, BehaviorGuard)
	if guard == nil || guard.ReturnType != "bool" || !strings.Contains(guard.Signature, "Context ctx") {
		t.Fatalf("Dart guard ABI = %#v", guard)
	}
	timer := BehaviorABIForTrigger(LanguageDart, BehaviorTrigger, TriggerAfter)
	if timer == nil || timer.ReturnType != "FutureOr<Duration>" {
		t.Fatalf("Dart after ABI = %#v", timer)
	}
	at := BehaviorABIForTrigger(LanguageDart, BehaviorTrigger, TriggerAt)
	if at == nil || at.ReturnType != "FutureOr<DateTime>" {
		t.Fatalf("Dart at ABI = %#v", at)
	}
	when := BehaviorABIForTrigger(LanguageDart, BehaviorTrigger, TriggerWhen)
	if when == nil || when.ReturnType != "bool" {
		t.Fatalf("Dart when ABI = %#v", when)
	}
}

func TestDartBackendEmitsAttributesWithPortableFallbacks(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			Attributes: []Attribute{
				{ID: "attribute_1", Name: "count", Type: "int", HasDefault: true, Default: "1", Language: LanguageDart},
				{ID: "attribute_2", Name: "label", HasDefault: true, Default: `"closed"`, Language: LanguageGo},
				{ID: "attribute_3", Name: "flag", Type: "bool", HasDefault: true, Default: "True", Language: LanguagePython},
				{ID: "attribute_4", Name: "maybe", HasDefault: true, Default: "None", Language: LanguagePython},
			},
		}},
	}

	output, err := NewDartBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`attribute("count", int, 1)`,
		`attribute("label", "closed")`,
		`// Type: bool`,
		`attribute("flag", true)`,
		`attribute("maybe", null)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Dart attribute output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, `attribute("flag", True)`) ||
		strings.Contains(text, `attribute("maybe", None)`) ||
		strings.Contains(text, `Attribute "count" preserved for manual porting`) {
		t.Fatalf("Dart attribute output used stale or invalid emission:\n%s", text)
	}
}

func TestDartBackendEmitsOperationPartial(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "operation_1",
			OwnerID:        "model_1",
			Kind:           BehaviorOperation,
			SourceLanguage: LanguageDart,
			Body:           `instance.set("approved", true);`,
		}},
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			Operations: []Operation{{
				ID:       "operation_decl_1",
				Name:     "approve",
				Behavior: &BehaviorRef{ID: "operation_1", Kind: BehaviorOperation},
			}},
		}},
	}
	AssignTargetABI(program, LanguageDart)

	output, err := NewDartBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`FutureOr<void> operation1(Context ctx, Instance instance, Event event)`,
		`operation("approve", operation1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Dart operation output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "preserved for manual porting") {
		t.Fatalf("Dart operation output preserved operation as comment:\n%s", text)
	}
}

func TestDartBackendQuotesOperationReferencesInBehaviorPartials(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			Initializer: &Initial{
				ID:      "initial_1",
				Target:  "closed",
				Effects: []BehaviorRef{{Kind: BehaviorEffect, Operation: "approve"}},
			},
			Operations: []Operation{{
				ID:   "operation_decl_1",
				Name: "approve",
			}},
			States: []State{{ID: "state_1", Name: "closed", Kind: StateKindState}},
		}},
	}

	output, err := NewDartBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, `effect("approve")`) {
		t.Fatalf("Dart operation reference effect should be quoted:\n%s", text)
	}
	if strings.Contains(text, `effect(approve)`) {
		t.Fatalf("Dart operation reference effect emitted as identifier:\n%s", text)
	}
}

func TestDartBackendEmitsNativeEveryTrigger(t *testing.T) {
	program := &Program{
		SourceLanguage: LanguageGo,
		Behaviors: []Behavior{{
			ID:          "trigger_1",
			Kind:        BehaviorTrigger,
			TriggerKind: TriggerEvery,
		}},
		Models: []Model{{
			ID:   "model_1",
			Name: "Timer",
			Initializer: &Initial{
				ID:     "initial_1",
				Target: "idle",
			},
			States: []State{
				{
					ID:   "state_1",
					Name: "idle",
					Kind: StateKindState,
					Transitions: []Transition{{
						ID:     "transition_1",
						Target: "../done",
						Trigger: &Trigger{
							Kind: TriggerEvery,
							Expr: &BehaviorRef{ID: "trigger_1"},
						},
					}},
				},
				{ID: "state_2", Name: "done", Kind: StateKindState},
			},
		}},
	}
	AssignTargetABI(program, LanguageDart)

	output, err := NewDartBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`import "dart:async";`,
		"FutureOr<Duration> trigger1(Context ctx, Instance instance, Event event)",
		`every(trigger1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Dart every output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "_hsmcEveryTransition") {
		t.Fatalf("Dart every trigger should use native runtime DSL, got:\n%s", text)
	}
}

func TestDartBackendCompilesGoEveryTriggerToNativeRuntime(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "every.go", Data: []byte(everySmokeGoSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageDart,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`import "dart:async";`,
		"FutureOr<Duration> trigger1(Context ctx, Instance instance, Event event)",
		"// Original go behavior trigger_1 preserved for manual porting:",
		"// Target language: dart",
		"// return time.Second",
		"return Duration.zero;",
		`every(trigger1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Dart Every output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{"Unsupported every trigger", "_hsmcEveryTransition", `"func(ctx context.Context`} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Dart Every output emitted unsupported or unstable trigger %q:\n%s", forbidden, text)
		}
	}
}

func TestDartBackendEmitsExtendedTriggers(t *testing.T) {
	program := &Program{
		SourceLanguage: LanguageGo,
		Behaviors: []Behavior{{
			ID:             "trigger_1",
			OwnerID:        "transition_3",
			Kind:           BehaviorTrigger,
			TriggerKind:    TriggerAt,
			SourceLanguage: LanguageGo,
			Body:           "return time.Now()",
		}},
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			States: []State{{
				ID:   "state_1",
				Name: "closed",
				Kind: StateKindState,
				Transitions: []Transition{
					{
						ID:     "transition_1",
						Target: "../open",
						Trigger: &Trigger{
							Kind:     TriggerOnSet,
							Value:    "count",
							IsString: true,
							Language: LanguageGo,
						},
					},
					{
						ID:     "transition_2",
						Target: "../open",
						Trigger: &Trigger{
							Kind:     TriggerOnCall,
							Value:    "approve",
							IsString: true,
							Language: LanguageGo,
						},
					},
					{
						ID:     "transition_3",
						Target: "../open",
						Trigger: &Trigger{
							Kind: TriggerAt,
							Expr: &BehaviorRef{ID: "trigger_1"},
						},
					},
				},
			}},
		}},
	}
	AssignTargetABI(program, LanguageDart)

	output, err := NewDartBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`import "dart:async";`,
		"FutureOr<DateTime> trigger1(Context ctx, Instance instance, Event event)",
		`onSet("count")`,
		`onCall("approve")`,
		`at(trigger1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Dart extended trigger output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{"Unsupported at trigger", "on_call trigger lowered to event trigger", "_hsmcAtTransition", "on(\"count\")", "on(\"approve\")"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Dart backend emitted unsupported runtime trigger %q:\n%s", forbidden, text)
		}
	}
}

func TestDartBackendEmitsPredicateWhenBehavior(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "predicate_when.go", Data: []byte(predicateWhenSmokeGoSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageDart,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"bool trigger1(Context ctx, Instance instance, Event event)",
		"// Original go behavior trigger_1 preserved for manual porting:",
		"// Target language: dart",
		"// return make(chan struct{})",
		"return false;",
		"when(trigger1)",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Dart predicate When output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "when(false)") || strings.Contains(text, "Unsupported when trigger") {
		t.Fatalf("Dart predicate When output used fallback trigger instead of behavior reference:\n%s", text)
	}
}

func TestDartBackendEmitsHistoryHelpersForHistoryPseudostates(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "pseudo.go", Data: []byte(goPseudostateSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageDart,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`shallowHistory(`,
		`deepHistory(`,
		`finalState("done"),`,
		`target("../a")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Dart history output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{
		"_hsmcShallowHistory",
		"_hsmcDeepHistory",
		"_hsmcHistory",
		"Kind.pseudostate",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Dart history output used custom history helper %q:\n%s", forbidden, text)
		}
	}
}

func TestDartBackendUsesAdapterFullCodeBehavior(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageDart,
			Code:           `(Context ctx, Instance instance, Event event) { adapterFormat("entry"); }`,
		}},
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			States: []State{{
				ID:        "state_1",
				Name:      "closed",
				Kind:      StateKindState,
				Behaviors: []BehaviorRef{{ID: "entry_1", Kind: BehaviorEntry}},
			}},
		}},
	}
	AssignTargetABI(program, LanguageDart)

	output, err := NewDartBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`final entry1 = (Context ctx, Instance instance, Event event) { adapterFormat("entry"); };`,
		`entry(entry1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Dart output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original go behavior") {
		t.Fatalf("Dart output commented adapter full code instead of using it:\n%s", text)
	}
}

func TestDartBackendUsesAdapterFullFunctionDeclaration(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageDart,
			Code: `FutureOr<void> entry1(Context ctx, Instance instance, Event event) {
  adapterFormat("entry");
}`,
		}},
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			States: []State{{
				ID:        "state_1",
				Name:      "closed",
				Kind:      StateKindState,
				Behaviors: []BehaviorRef{{ID: "entry_1", Kind: BehaviorEntry}},
			}},
		}},
	}
	AssignTargetABI(program, LanguageDart)

	output, err := NewDartBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`import "dart:async";`,
		`FutureOr<void> entry1(Context ctx, Instance instance, Event event)`,
		`adapterFormat("entry");`,
		`entry(entry1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Dart output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "final entry1 = FutureOr<void> entry1") {
		t.Fatalf("Dart output wrapped a full function declaration as an expression:\n%s", text)
	}
}

func TestDartBackendRejectsAdapterFullCodeBehaviorNameChanges(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageDart,
			Code: `FutureOr<void> renamedEntry(Context ctx, Instance instance, Event event) {
  adapterFormat("entry");
}`,
		}},
	}
	AssignTargetABI(program, LanguageDart)

	_, err := NewDartBackend().Emit(context.Background(), program)
	if err == nil || !strings.Contains(err.Error(), `declares "renamedEntry", want compiler-owned name "entry1"`) {
		t.Fatalf("error = %v, want compiler-owned Dart behavior name rejection", err)
	}
}

func TestDartBackendRejectsAdapterFullCodeBehaviorSignatureChanges(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "return type",
			code: `bool entry1(Context ctx, Instance instance, Event event) {
  return true;
}`,
			want: `declares return type "bool", want compiler-owned return type "FutureOr<void>"`,
		},
		{
			name: "parameters",
			code: `FutureOr<void> entry1(Context ctx) {
}`,
			want: `declares parameters "(Context ctx)", want compiler-owned parameters "(Context ctx, Instance instance, Event event)"`,
		},
		{
			name: "arrow declaration parameters",
			code: `FutureOr<void> entry1(Context ctx) => adapterFormat("entry");`,
			want: `declares parameters "(Context ctx)", want compiler-owned parameters "(Context ctx, Instance instance, Event event)"`,
		},
		{
			name: "arrow declaration return type",
			code: `bool entry1(Context ctx, Instance instance, Event event) => true;`,
			want: `declares return type "bool", want compiler-owned return type "FutureOr<void>"`,
		},
		{
			name: "function expression parameters",
			code: `(Context ctx) {
  adapterFormat("entry");
}`,
			want: `declares parameters "(Context ctx)", want compiler-owned parameters "(Context ctx, Instance instance, Event event)"`,
		},
		{
			name: "top level variable function expression parameters",
			code: `final entry1 = (Context ctx) {
  adapterFormat("entry");
};`,
			want: `declares parameters "(Context ctx)", want compiler-owned parameters "(Context ctx, Instance instance, Event event)"`,
		},
		{
			name: "async expression parameters",
			code: `(Context ctx) async {
  adapterFormat("entry");
}`,
			want: `declares parameters "(Context ctx)", want compiler-owned parameters "(Context ctx, Instance instance, Event event)"`,
		},
		{
			name: "non-callable variable declaration",
			code: `final entry1 = 1;`,
			want: `must be a compiler-owned callable declaration`,
		},
		{
			name: "same-name class declaration",
			code: `class entry1 {}`,
			want: `must be a compiler-owned callable declaration`,
		},
		{
			name: "same-name typedef declaration",
			code: `typedef entry1 = void Function(Context ctx, Instance instance, Event event);`,
			want: `must be a compiler-owned callable declaration`,
		},
		{
			name: "same-name external function declaration",
			code: `external FutureOr<void> entry1(Context ctx, Instance instance, Event event);`,
			want: `must be a compiler-owned callable declaration`,
		},
		{
			name: "non-callable expression",
			code: `1`,
			want: `must be a compiler-owned callable expression`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := &Program{
				Behaviors: []Behavior{{
					ID:             "entry_1",
					OwnerID:        "state_1",
					Kind:           BehaviorEntry,
					SourceLanguage: LanguageGo,
					TargetLanguage: LanguageDart,
					Code:           tt.code,
				}},
			}
			AssignTargetABI(program, LanguageDart)

			_, err := NewDartBackend().Emit(context.Background(), program)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}
