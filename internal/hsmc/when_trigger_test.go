package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestBackendsEmitAttributeWhenAsTargetAttributeTrigger(t *testing.T) {
	cases := []struct {
		name    string
		target  Language
		prepare func(*Program)
		emit    func(*Program) ([]byte, error)
		want    []string
		forbid  []string
	}{
		{
			name:   "csharp",
			target: LanguageCSharp,
			emit: func(program *Program) ([]byte, error) {
				return NewCSharpBackend().Emit(context.Background(), program)
			},
			want:   []string{`Hsm.OnSet("count")`},
			forbid: []string{`Hsm.When<Instance>("count")`},
		},
		{
			name:   "cpp",
			target: LanguageCPP,
			emit: func(program *Program) ([]byte, error) {
				return NewCPPBackend().Emit(context.Background(), program)
			},
			want:   []string{`hsm::on_set("count")`},
			forbid: []string{`hsm::when("count")`},
		},
		{
			name:   "dart",
			target: LanguageDart,
			emit: func(program *Program) ([]byte, error) {
				return NewDartBackend().Emit(context.Background(), program)
			},
			want:   []string{`when("count")`},
			forbid: []string{"when attribute trigger lowered", `onSet("count")`},
		},
		{
			name:   "go",
			target: LanguageGo,
			prepare: func(program *Program) {
				program.PackageName = "sample"
			},
			emit: func(program *Program) ([]byte, error) {
				return NewGoBackend().Emit(context.Background(), program)
			},
			want:   []string{`hsm.OnSet("count")`},
			forbid: []string{`hsm.When("count")`},
		},
		{
			name:   "javascript",
			target: LanguageJS,
			emit: func(program *Program) ([]byte, error) {
				return NewJavaScriptBackend().Emit(context.Background(), program)
			},
			want:   []string{`hsm.OnSet("count")`},
			forbid: []string{`hsm.When("count")`},
		},
		{
			name:   "java",
			target: LanguageJava,
			emit: func(program *Program) ([]byte, error) {
				return NewJavaBackend().Emit(context.Background(), program)
			},
			want:   []string{`Hsm.OnSet("count")`},
			forbid: []string{`Hsm.When("count")`, `Hsm.When(() -> false)`},
		},
		{
			name:   "python",
			target: LanguagePython,
			emit: func(program *Program) ([]byte, error) {
				return NewPythonBackend().Emit(context.Background(), program)
			},
			want:   []string{`hsm.OnSet("count")`},
			forbid: []string{`hsm.When("count")`},
		},
		{
			name:   "rust",
			target: LanguageRust,
			emit: func(program *Program) ([]byte, error) {
				return NewRustBackend().Emit(context.Background(), program)
			},
			want:   []string{`hsm::when!("count")`},
			forbid: []string{`hsm::guard!`, `when attribute trigger lowered`, `hsm::on_set("count")`},
		},
		{
			name:   "typescript",
			target: LanguageTS,
			emit: func(program *Program) ([]byte, error) {
				return NewTypeScriptBackend().Emit(context.Background(), program)
			},
			want:   []string{`hsm.OnSet("count")`},
			forbid: []string{`hsm.When("count")`},
		},
		{
			name:   "zig",
			target: LanguageZig,
			emit: func(program *Program) ([]byte, error) {
				return NewZigBackend().Emit(context.Background(), program)
			},
			want:   []string{`hsm.onSet("count")`},
			forbid: []string{"Unsupported when trigger", "when attribute trigger lowered to event trigger", `hsm.on("count")`},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := programWithAttributeWhenTrigger()
			if tc.prepare != nil {
				tc.prepare(program)
			}
			AssignTargetABI(program, tc.target)
			output, err := tc.emit(program)
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			for _, needle := range tc.want {
				if !strings.Contains(text, needle) {
					t.Fatalf("%s output missing %q:\n%s", tc.name, needle, text)
				}
			}
			for _, needle := range tc.forbid {
				if strings.Contains(text, needle) {
					t.Fatalf("%s output contains forbidden %q:\n%s", tc.name, needle, text)
				}
			}
		})
	}
}

func TestValidateTreatsStringWhenAsAttributeTrigger(t *testing.T) {
	program := programWithAttributeWhenTrigger()
	program.Models[0].Attributes[0].Name = "count/current"
	program.Models[0].States[0].Transitions[0].Trigger.Value = "count/current"

	err := Validate(program)
	if err == nil {
		t.Fatal("Validate returned nil")
	}
	if !diagnosticsContain(err, `invalid attribute trigger name "count/current"`) {
		t.Fatalf("diagnostics = %v", err)
	}
}

func TestCPPBackendLowersPredicateWhenBehaviorToGuard(t *testing.T) {
	program := &Program{
		SourceLanguage: LanguageGo,
		Behaviors: []Behavior{{
			ID:             "trigger_1",
			OwnerID:        "transition_1",
			Kind:           BehaviorTrigger,
			TriggerKind:    TriggerWhen,
			SourceLanguage: LanguageGo,
			Body:           "return ready",
		}},
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			Initializer: &Initial{
				OwnerID: "model_1",
				ID:      "initial_1",
				Target:  "closed",
			},
			States: []State{
				{
					ID:   "state_1",
					Name: "closed",
					Kind: StateKindState,
					Transitions: []Transition{{
						OwnerID: "state_1",
						ID:      "transition_1",
						Target:  "../open",
						Trigger: &Trigger{
							Kind: TriggerWhen,
							Expr: &BehaviorRef{ID: "trigger_1", Kind: BehaviorTrigger},
						},
					}},
				},
				{ID: "state_2", Name: "open", Kind: StateKindState},
			},
		}},
	}
	AssignTargetABI(program, LanguageCPP)

	output, err := NewCPPBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"static constexpr auto trigger_1 = [](auto& signal, auto& instance, const auto& event) -> bool",
		"// Original go behavior trigger_1 preserved for manual porting:",
		"return false;",
		`hsm::transition(hsm::guard(trigger_1), hsm::target("/Door/open"))`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C++ output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Unsupported when trigger") || strings.Contains(text, "hsm::when(trigger_1)") {
		t.Fatalf("C++ output did not lower predicate when behavior to guard:\n%s", text)
	}
}

func TestTypeScriptBackendEmitsPredicateWhenAsRuntimeCall(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "predicate_when.go", Data: []byte(predicateWhenSmokeGoSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"function trigger1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): unknown",
		"// Original go behavior trigger_1 preserved for manual porting:",
		"return undefined;",
		"export const PredicateWhenDoorModel = (hsm.Define as any)(",
		"(hsm.When as any)(trigger1)",
		") as any,",
		"\t) as any,",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript predicate When output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "hsm.When(\"func(") || strings.Contains(text, "hsm.When(trigger1)") {
		t.Fatalf("TypeScript predicate When output used typed attribute overload:\n%s", text)
	}
}

func programWithAttributeWhenTrigger() *Program {
	return &Program{
		SourceLanguage: LanguageGo,
		Models: []Model{{
			ID:         "model_1",
			Name:       "Door",
			Attributes: []Attribute{{ID: "attribute_1", Name: "count", HasDefault: true, Default: "0"}},
			Initializer: &Initial{
				OwnerID: "model_1",
				ID:      "initial_1",
				Target:  "closed",
			},
			States: []State{
				{
					ID:   "state_1",
					Name: "closed",
					Kind: StateKindState,
					Transitions: []Transition{{
						OwnerID: "state_1",
						ID:      "transition_1",
						Target:  "../open",
						Trigger: &Trigger{
							Kind:     TriggerWhen,
							Value:    "count",
							IsString: true,
							Language: LanguageGo,
						},
					}},
				},
				{ID: "state_2", Name: "open", Kind: StateKindState},
			},
		}},
	}
}
