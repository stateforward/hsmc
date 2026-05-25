package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestForeignTriggerExpressionFallbacksAreNonThrowing(t *testing.T) {
	cases := []struct {
		name   string
		target Language
		emit   func(*Program) ([]byte, error)
		want   []string
		forbid []string
	}{
		{
			name:   "go",
			target: LanguageGo,
			emit: func(program *Program) ([]byte, error) {
				program.PackageName = "sample"
				return NewGoBackend().Emit(context.Background(), program)
			},
			want: []string{
				"// Original typescript after trigger expression preserved for manual porting:",
				"// Date.now() + 1000",
				"return 0",
			},
			forbid: []string{"panic(", "requires translation"},
		},
		{
			name:   "csharp",
			target: LanguageCSharp,
			emit: func(program *Program) ([]byte, error) {
				return NewCSharpBackend().Emit(context.Background(), program)
			},
			want: []string{
				"// Original typescript after trigger expression preserved for manual porting:",
				"// Date.now() + 1000",
				"return System.TimeSpan.Zero;",
			},
			forbid: []string{"throw new", "requires translation"},
		},
		{
			name:   "cpp",
			target: LanguageCPP,
			emit: func(program *Program) ([]byte, error) {
				return NewCPPBackend().Emit(context.Background(), program)
			},
			want: []string{
				"// Original typescript after trigger expression preserved for manual porting:",
				"// Date.now() + 1000",
				"return std::chrono::milliseconds{0};",
			},
			forbid: []string{"throw ", "requires translation"},
		},
		{
			name:   "java",
			target: LanguageJava,
			emit: func(program *Program) ([]byte, error) {
				return NewJavaBackend().Emit(context.Background(), program)
			},
			want: []string{
				"java.time.Duration.ZERO /* Original typescript after trigger preserved for manual porting. */",
			},
			forbid: []string{"throw new", "requires translation"},
		},
		{
			name:   "javascript",
			target: LanguageJS,
			emit: func(program *Program) ([]byte, error) {
				return NewJavaScriptBackend().Emit(context.Background(), program)
			},
			want: []string{
				"// Original typescript after trigger expression preserved for manual porting:",
				"// Date.now() + 1000",
				"return 0;",
			},
			forbid: []string{"throw new Error", "requires translation"},
		},
		{
			name:   "typescript",
			target: LanguageTS,
			emit: func(program *Program) ([]byte, error) {
				return NewTypeScriptBackend().Emit(context.Background(), program)
			},
			want: []string{
				"// Original python after trigger expression preserved for manual porting:",
				"// deadline()",
				"return 0;",
			},
			forbid: []string{"throw new Error", "requires translation"},
		},
		{
			name:   "python",
			target: LanguagePython,
			emit: func(program *Program) ([]byte, error) {
				return NewPythonBackend().Emit(context.Background(), program)
			},
			want: []string{
				"def hsmc_trigger_fallback_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> float:",
				"# Original typescript after trigger expression preserved for manual porting:",
				"# Date.now() + 1000",
				"return 0.0",
				"hsm.After(hsmc_trigger_fallback_1)",
			},
			forbid: []string{"RuntimeError", "requires translation"},
		},
		{
			name:   "rust",
			target: LanguageRust,
			emit: func(program *Program) ([]byte, error) {
				return NewRustBackend().Emit(context.Background(), program)
			},
			want: []string{
				"/* Original typescript after trigger expression preserved for manual porting:",
				" * Date.now() + 1000",
				"*/ hsmc_zero_duration",
			},
			forbid: []string{"panic!", "requires translation", "|| {"},
		},
		{
			name:   "dart",
			target: LanguageDart,
			emit: func(program *Program) ([]byte, error) {
				return NewDartBackend().Emit(context.Background(), program)
			},
			want: []string{
				"// Original typescript after trigger expression preserved for manual porting:",
				"// Date.now() + 1000",
				"return Duration.zero;",
			},
			forbid: []string{"throw Exception", "requires translation"},
		},
		{
			name:   "zig",
			target: LanguageZig,
			emit: func(program *Program) ([]byte, error) {
				return NewZigBackend().Emit(context.Background(), program)
			},
			want: []string{
				"// Original typescript after trigger expression preserved for manual porting:",
				"// Date.now() + 1000",
				"hsm.after(hsmc_trigger_zero),",
			},
			forbid: []string{"@panic", "requires translation"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sourceLanguage := LanguageTS
			value := "Date.now() + 1000"
			if tc.target == LanguageTS {
				sourceLanguage = LanguagePython
				value = "deadline()"
			}
			output, err := tc.emit(programWithForeignTriggerExpression(sourceLanguage, value))
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

func TestJavaForeignAtTriggerExpressionFallbackUsesInstant(t *testing.T) {
	program := programWithForeignTriggerExpression(LanguageTS, "new Date()")
	program.Models[0].States[0].Transitions[0].Trigger.Kind = TriggerAt

	output, err := NewJavaBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"Hsm.At(java.time.Instant.EPOCH /* Original typescript at trigger preserved for manual porting. */)",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Java output missing %q:\n%s", needle, text)
		}
	}
	for _, needle := range []string{
		"Hsm.At(java.time.Duration.ZERO",
		"throw new",
		"requires translation",
	} {
		if strings.Contains(text, needle) {
			t.Fatalf("Java output contains forbidden %q:\n%s", needle, text)
		}
	}
}

func TestJavaScriptForeignAtAndWhenTriggerExpressionFallbacks(t *testing.T) {
	cases := []struct {
		name  string
		kind  TriggerKind
		value string
		want  []string
	}{
		{
			name:  "at",
			kind:  TriggerAt,
			value: "deadline()",
			want: []string{
				"hsm.At(() => {",
				"// Original python at trigger expression preserved for manual porting:",
				"// deadline()",
				"return new Date(0);",
			},
		},
		{
			name:  "when",
			kind:  TriggerWhen,
			value: "ready()",
			want: []string{
				"hsm.When(() => {",
				"// Original python when trigger expression preserved for manual porting:",
				"// ready()",
				"return undefined;",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := programWithForeignTriggerExpression(LanguagePython, tc.value)
			program.Models[0].States[0].Transitions[0].Trigger.Kind = tc.kind

			output, err := NewJavaScriptBackend().Emit(context.Background(), program)
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			for _, needle := range tc.want {
				if !strings.Contains(text, needle) {
					t.Fatalf("JavaScript output missing %q:\n%s", needle, text)
				}
			}
			for _, needle := range []string{"throw new Error", "requires translation"} {
				if strings.Contains(text, needle) {
					t.Fatalf("JavaScript output contains forbidden %q:\n%s", needle, text)
				}
			}
		})
	}
}

func TestZigForeignWhenTriggerExpressionIsPreservedWithoutUndefinedFallback(t *testing.T) {
	program := programWithForeignTriggerExpression(LanguagePython, "ready()")
	program.Models[0].States[0].Transitions[0].Trigger.Kind = TriggerWhen

	output, err := NewZigBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Unsupported when trigger preserved for manual porting:",
		"// ready()",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Zig output missing %q:\n%s", needle, text)
		}
	}
	for _, needle := range []string{
		"hsmc_trigger_false",
		"hsmc_trigger_zero",
		"hsm.when(",
		"hsm.guard(hsmc_trigger",
	} {
		if strings.Contains(text, needle) {
			t.Fatalf("Zig output contains forbidden %q:\n%s", needle, text)
		}
	}
}

func TestTypeScriptForeignAtAndWhenTriggerExpressionFallbacks(t *testing.T) {
	cases := []struct {
		name  string
		kind  TriggerKind
		value string
		want  []string
	}{
		{
			name:  "at",
			kind:  TriggerAt,
			value: "deadline()",
			want: []string{
				"hsm.At(() => {",
				"// Original python at trigger expression preserved for manual porting:",
				"// deadline()",
				"return new Date(0);",
			},
		},
		{
			name:  "when",
			kind:  TriggerWhen,
			value: "ready()",
			want: []string{
				"(hsm.When as any)(() => {",
				"// Original python when trigger expression preserved for manual porting:",
				"// ready()",
				"return undefined;",
				") as any,",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := programWithForeignTriggerExpression(LanguagePython, tc.value)
			program.Models[0].States[0].Transitions[0].Trigger.Kind = tc.kind

			output, err := NewTypeScriptBackend().Emit(context.Background(), program)
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			for _, needle := range tc.want {
				if !strings.Contains(text, needle) {
					t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
				}
			}
			for _, needle := range []string{"throw new Error", "requires translation"} {
				if strings.Contains(text, needle) {
					t.Fatalf("TypeScript output contains forbidden %q:\n%s", needle, text)
				}
			}
		})
	}
}

func programWithForeignTriggerExpression(language Language, value string) *Program {
	return &Program{
		SourceLanguage: language,
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
							Kind:     TriggerAfter,
							Value:    value,
							Language: language,
						},
					}},
				},
				{ID: "state_2", Name: "done", Kind: StateKindState},
			},
		}},
	}
}
