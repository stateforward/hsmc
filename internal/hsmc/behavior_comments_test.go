package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestUntranslatedBehaviorCommentIncludesContextWhenBodyMissing(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "guard_1",
			OwnerID:        "state_1",
			Kind:           BehaviorGuard,
			SourceLanguage: LanguagePython,
			TargetLanguage: LanguageTS,
		}},
	}
	AssignTargetABI(program, LanguageTS)

	output, err := NewTypeScriptBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"function guard1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): boolean {",
		"// Original python behavior guard_1 preserved for manual porting:",
		"// Target language: typescript",
		"// Behavior kind: guard",
		"// No source behavior body was captured.",
		"return false;",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestUntranslatedBehaviorCommentUsesTargetABIWithoutMutatingTargetLanguage(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			Body:           `sm.log = append(sm.log, "entered")`,
		}},
	}
	AssignTargetABI(program, LanguageTS)

	output, err := NewTypeScriptBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	if program.Behaviors[0].TargetLanguage != "" {
		t.Fatalf("comment fallback mutated adapter-owned target_language: %#v", program.Behaviors[0])
	}
	text := string(output)
	for _, needle := range []string{
		"// Original go behavior entry_1 preserved for manual porting:",
		"// Target language: typescript",
		`// sm.log = append(sm.log, "entered")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestUntranslatedBehaviorCommentIncludesSourceImportsInsideBody(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageTS,
			Body:           `rec("closed");`,
			Imports: []Import{{
				Path:       "./format",
				Language:   LanguageTS,
				Specifiers: []ImportSpecifier{{Name: "record", Alias: "rec"}},
			}},
		}},
	}
	AssignTargetABI(program, LanguagePython)

	output, err := NewPythonBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:",
		"# Source imports:",
		`# import { record as rec } from "./format";`,
		`# rec("closed");`,
		"pass",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "\nimport { record as rec } from \"./format\";") {
		t.Fatalf("Python output emitted foreign TypeScript import as executable code:\n%s", text)
	}
}

func TestUntranslatedBehaviorCommentUsesExplicitImportLanguage(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			Body:           `rec("closed")`,
			Imports: []Import{{
				Path:       "./format",
				Language:   LanguageTS,
				Specifiers: []ImportSpecifier{{Name: "record", Alias: "rec"}},
			}},
		}},
	}
	AssignTargetABI(program, LanguagePython)

	output, err := NewPythonBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, `# import { record as rec } from "./format";`) {
		t.Fatalf("Python output did not format explicit TypeScript import language:\n%s", text)
	}
	if strings.Contains(text, `# import "./format"`) {
		t.Fatalf("Python output formatted explicit TypeScript import as Go:\n%s", text)
	}
}

func TestUntranslatedGlobalCommentUsesExplicitImportLanguage(t *testing.T) {
	program := &Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageGo,
			Code:     `var _ = rec("closed")`,
			Imports: []Import{{
				Path:       "./format",
				Language:   LanguageTS,
				Specifiers: []ImportSpecifier{{Name: "record", Alias: "rec"}},
			}},
		}},
	}

	output, err := NewPythonBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, `# import { record as rec } from "./format";`) {
		t.Fatalf("Python output did not format explicit TypeScript global import language:\n%s", text)
	}
	if strings.Contains(text, `# import "./format"`) {
		t.Fatalf("Python output formatted explicit TypeScript global import as Go:\n%s", text)
	}
}

func TestUntranslatedBehaviorCommentIncludesSourceImportsInZigBackend(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			Body:           `fmt.Println("closed")`,
			Imports:        []Import{{Path: "fmt", Language: LanguageGo}},
		}},
	}
	AssignTargetABI(program, LanguageZig)

	output, err := NewZigBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Source imports:",
		`// import "fmt"`,
		`// fmt.Println("closed")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Zig output missing %q:\n%s", needle, text)
		}
	}
}

func TestGoUntranslatedAtTriggerBehaviorImportsTime(t *testing.T) {
	program := &Program{
		PackageName: "sample",
		Behaviors: []Behavior{{
			ID:             "trigger_1",
			OwnerID:        "state_1",
			Kind:           BehaviorTrigger,
			TriggerKind:    TriggerAt,
			SourceLanguage: LanguageTS,
			Body:           "return new Date()",
		}},
	}
	AssignTargetABI(program, LanguageGo)

	output, err := NewGoBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`"time"`,
		"func Trigger1(ctx context.Context, instance hsm.Instance, event hsm.Event) time.Time {",
		"// Original typescript behavior trigger_1 preserved for manual porting:",
		"// return new Date()",
		"return time.Time{}",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
}

func TestJavaScriptBackendCommentsUnadaptedTypeScriptBehavior(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageTS,
			Body:           "const count: number = instance.count;",
		}},
	}
	AssignTargetABI(program, LanguageJS)

	output, err := NewJavaScriptBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"function entry1(ctx, instance, event) {",
		"// Original typescript behavior entry_1 preserved for manual porting:",
		"// const count: number = instance.count;",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("JavaScript output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "\tconst count: number = instance.count;") {
		t.Fatalf("JavaScript output emitted unadapted TypeScript body as executable code:\n%s", text)
	}
}

func TestBackendsPreserveUntranslatedBehaviorAsTargetCommentInsideBody(t *testing.T) {
	cases := []struct {
		name      string
		behavior  Behavior
		prepare   func(*Program)
		assignABI Language
		emit      func(*Program) ([]byte, error)
		before    string
		comment   string
		after     string
	}{
		{
			name: "csharp",
			behavior: Behavior{
				ID:             "guard_1",
				OwnerID:        "state_1",
				Kind:           BehaviorGuard,
				SourceLanguage: LanguageTS,
				Body:           "return instance.allowed;",
			},
			assignABI: LanguageCSharp,
			emit: func(program *Program) ([]byte, error) {
				return NewCSharpBackend().Emit(context.Background(), program)
			},
			before:  "private static bool Guard1(Context ctx, Instance instance, Event @event)",
			comment: "// return instance.allowed;",
			after:   "return false;",
		},
		{
			name: "go",
			behavior: Behavior{
				ID:             "entry_1",
				OwnerID:        "state_1",
				Kind:           BehaviorEntry,
				SourceLanguage: LanguageTS,
				Body:           "instance.log.push('entered');",
			},
			prepare:   func(program *Program) { program.PackageName = "sample" },
			assignABI: LanguageGo,
			emit: func(program *Program) ([]byte, error) {
				return NewGoBackend().Emit(context.Background(), program)
			},
			before:  "func Entry1(ctx context.Context, instance hsm.Instance, event hsm.Event) {",
			comment: "// instance.log.push('entered');",
			after:   "}",
		},
		{
			name: "cpp",
			behavior: Behavior{
				ID:             "guard_1",
				OwnerID:        "state_1",
				Kind:           BehaviorGuard,
				SourceLanguage: LanguageTS,
				Body:           "return instance.allowed;",
			},
			assignABI: LanguageCPP,
			emit: func(program *Program) ([]byte, error) {
				return NewCPPBackend().Emit(context.Background(), program)
			},
			before:  "static constexpr auto guard_1 = [](auto& signal, auto& instance, const auto& event) -> bool {",
			comment: "// return instance.allowed;",
			after:   "return false;",
		},
		{
			name: "dart",
			behavior: Behavior{
				ID:             "guard_1",
				OwnerID:        "state_1",
				Kind:           BehaviorGuard,
				SourceLanguage: LanguageGo,
				Body:           "return instance.Allowed",
			},
			assignABI: LanguageDart,
			emit: func(program *Program) ([]byte, error) {
				return NewDartBackend().Emit(context.Background(), program)
			},
			before:  "bool guard1(Context ctx, Instance instance, Event event) {",
			comment: "// return instance.Allowed",
			after:   "return false;",
		},
		{
			name: "java",
			behavior: Behavior{
				ID:             "guard_1",
				OwnerID:        "state_1",
				Kind:           BehaviorGuard,
				SourceLanguage: LanguageTS,
				Body:           "return instance.allowed;",
			},
			assignABI: LanguageJava,
			emit: func(program *Program) ([]byte, error) {
				return NewJavaBackend().Emit(context.Background(), program)
			},
			before:  "private static boolean Guard1(Context ctx, Instance instance, Event event) {",
			comment: "// return instance.allowed;",
			after:   "return false;",
		},
		{
			name: "javascript",
			behavior: Behavior{
				ID:             "entry_1",
				OwnerID:        "state_1",
				Kind:           BehaviorEntry,
				SourceLanguage: LanguagePython,
				Body:           "instance.log.append('entered')",
			},
			assignABI: LanguageJS,
			emit: func(program *Program) ([]byte, error) {
				return NewJavaScriptBackend().Emit(context.Background(), program)
			},
			before:  "function entry1(ctx, instance, event) {",
			comment: "// instance.log.append('entered')",
			after:   "}",
		},
		{
			name: "typescript",
			behavior: Behavior{
				ID:             "guard_1",
				OwnerID:        "state_1",
				Kind:           BehaviorGuard,
				SourceLanguage: LanguagePython,
				Body:           "return instance.allowed",
			},
			assignABI: LanguageTS,
			emit: func(program *Program) ([]byte, error) {
				return NewTypeScriptBackend().Emit(context.Background(), program)
			},
			before:  "function guard1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): boolean {",
			comment: "// return instance.allowed",
			after:   "return false;",
		},
		{
			name: "python",
			behavior: Behavior{
				ID:             "entry_1",
				OwnerID:        "state_1",
				Kind:           BehaviorEntry,
				SourceLanguage: LanguageTS,
				Body:           "instance.log.push('entered');",
			},
			assignABI: LanguagePython,
			emit: func(program *Program) ([]byte, error) {
				return NewPythonBackend().Emit(context.Background(), program)
			},
			before:  "async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:",
			comment: "# instance.log.push('entered');",
			after:   "pass",
		},
		{
			name: "rust",
			behavior: Behavior{
				ID:             "guard_1",
				OwnerID:        "state_1",
				Kind:           BehaviorGuard,
				SourceLanguage: LanguageGo,
				Body:           "return instance.Allowed",
			},
			assignABI: LanguageRust,
			emit: func(program *Program) ([]byte, error) {
				return NewRustBackend().Emit(context.Background(), program)
			},
			before:  "fn guard_1(ctx: &hsm::Context, instance: &HsmcInstance, event: &hsm::Event) -> bool {",
			comment: "// return instance.Allowed",
			after:   "false",
		},
		{
			name: "zig",
			behavior: Behavior{
				ID:             "guard_1",
				OwnerID:        "state_1",
				Kind:           BehaviorGuard,
				SourceLanguage: LanguageGo,
				Body:           "return instance.Allowed",
			},
			assignABI: LanguageZig,
			emit: func(program *Program) ([]byte, error) {
				return NewZigBackend().Emit(context.Background(), program)
			},
			before:  "fn guard_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) bool {",
			comment: "// return instance.Allowed",
			after:   "return false;",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := &Program{Behaviors: []Behavior{tc.behavior}}
			if tc.prepare != nil {
				tc.prepare(program)
			}
			AssignTargetABI(program, tc.assignABI)

			output, err := tc.emit(program)
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			beforeIndex := strings.Index(text, tc.before)
			commentIndex := strings.Index(text, tc.comment)
			afterIndex := -1
			if commentIndex >= 0 {
				afterIndex = strings.Index(text[commentIndex+len(tc.comment):], tc.after)
			}
			if beforeIndex < 0 || commentIndex < 0 || afterIndex < 0 || beforeIndex >= commentIndex {
				t.Fatalf("%s output did not preserve untranslated behavior inside body:\n%s", tc.name, text)
			}
			for _, needle := range []string{
				"Original " + string(tc.behavior.SourceLanguage) + " behavior " + tc.behavior.ID + " preserved for manual porting:",
				"Target language: " + string(tc.assignABI),
				"Behavior kind: " + string(tc.behavior.Kind),
			} {
				if !strings.Contains(text, needle) {
					t.Fatalf("%s output missing behavior preservation context %q:\n%s", tc.name, needle, text)
				}
			}
			header := "Original " + string(tc.behavior.SourceLanguage) + " behavior " + tc.behavior.ID + " preserved for manual porting:"
			if count := strings.Count(text, header); count != 1 {
				t.Fatalf("%s output emitted behavior preservation header %d times, want 1:\n%s", tc.name, count, text)
			}
		})
	}
}

func TestBackendsPreserveUntranslatedBehaviorCodeAsTargetCommentInsideBody(t *testing.T) {
	cases := []struct {
		name      string
		behavior  Behavior
		prepare   func(*Program)
		assignABI Language
		emit      func(*Program) ([]byte, error)
		before    string
		comment   string
		after     string
		forbid    string
	}{
		{
			name: "csharp",
			behavior: Behavior{
				ID:             "entry_1",
				OwnerID:        "state_1",
				Kind:           BehaviorEntry,
				SourceLanguage: LanguageTS,
				Code:           "(ctx, instance, event) => { instance.log.push('entered'); }",
			},
			assignABI: LanguageCSharp,
			emit: func(program *Program) ([]byte, error) {
				return NewCSharpBackend().Emit(context.Background(), program)
			},
			before:  "private static void Entry1(Context ctx, Instance instance, Event @event)",
			comment: "// (ctx, instance, event) => { instance.log.push('entered'); }",
			after:   "}",
			forbid:  "\t\t(ctx, instance, event) => { instance.log.push('entered'); }",
		},
		{
			name: "go",
			behavior: Behavior{
				ID:             "entry_1",
				OwnerID:        "state_1",
				Kind:           BehaviorEntry,
				SourceLanguage: LanguageTS,
				Code:           "(ctx, instance, event) => { instance.log.push('entered'); }",
			},
			prepare:   func(program *Program) { program.PackageName = "sample" },
			assignABI: LanguageGo,
			emit: func(program *Program) ([]byte, error) {
				return NewGoBackend().Emit(context.Background(), program)
			},
			before:  "func Entry1(ctx context.Context, instance hsm.Instance, event hsm.Event) {",
			comment: "// (ctx, instance, event) => { instance.log.push('entered'); }",
			after:   "}",
			forbid:  "\t(ctx, instance, event) => { instance.log.push('entered'); }",
		},
		{
			name: "cpp",
			behavior: Behavior{
				ID:             "entry_1",
				OwnerID:        "state_1",
				Kind:           BehaviorEntry,
				SourceLanguage: LanguageGo,
				Code:           "func(ctx context.Context, instance hsm.Instance, event hsm.Event) { instance.Set(\"entered\", true) }",
			},
			assignABI: LanguageCPP,
			emit: func(program *Program) ([]byte, error) {
				return NewCPPBackend().Emit(context.Background(), program)
			},
			before:  "static constexpr auto entry_1 = [](auto& signal, auto& instance, const auto& event) -> void {",
			comment: "// func(ctx context.Context, instance hsm.Instance, event hsm.Event) { instance.Set(\"entered\", true) }",
			after:   "};",
			forbid:  "\tfunc(ctx context.Context",
		},
		{
			name: "dart",
			behavior: Behavior{
				ID:             "entry_1",
				OwnerID:        "state_1",
				Kind:           BehaviorEntry,
				SourceLanguage: LanguageGo,
				Code:           "func(ctx context.Context, instance hsm.Instance, event hsm.Event) { instance.Set(\"entered\", true) }",
			},
			assignABI: LanguageDart,
			emit: func(program *Program) ([]byte, error) {
				return NewDartBackend().Emit(context.Background(), program)
			},
			before:  "FutureOr<void> entry1(Context ctx, Instance instance, Event event) {",
			comment: "// func(ctx context.Context, instance hsm.Instance, event hsm.Event) { instance.Set(\"entered\", true) }",
			after:   "}",
			forbid:  "\tfunc(ctx context.Context",
		},
		{
			name: "java",
			behavior: Behavior{
				ID:             "entry_1",
				OwnerID:        "state_1",
				Kind:           BehaviorEntry,
				SourceLanguage: LanguageTS,
				Code:           "(ctx, instance, event) => { instance.log.push('entered'); }",
			},
			assignABI: LanguageJava,
			emit: func(program *Program) ([]byte, error) {
				return NewJavaBackend().Emit(context.Background(), program)
			},
			before:  "private static void Entry1(Context ctx, Instance instance, Event event) {",
			comment: "// (ctx, instance, event) => { instance.log.push('entered'); }",
			after:   "}",
			forbid:  "\t\t(ctx, instance, event) => { instance.log.push('entered'); }",
		},
		{
			name: "javascript",
			behavior: Behavior{
				ID:             "entry_1",
				OwnerID:        "state_1",
				Kind:           BehaviorEntry,
				SourceLanguage: LanguagePython,
				Code:           "lambda ctx, instance, event: instance.log.append('entered')",
			},
			assignABI: LanguageJS,
			emit: func(program *Program) ([]byte, error) {
				return NewJavaScriptBackend().Emit(context.Background(), program)
			},
			before:  "function entry1(ctx, instance, event) {",
			comment: "// lambda ctx, instance, event: instance.log.append('entered')",
			after:   "}",
			forbid:  "\tlambda ctx, instance",
		},
		{
			name: "typescript",
			behavior: Behavior{
				ID:             "entry_1",
				OwnerID:        "state_1",
				Kind:           BehaviorEntry,
				SourceLanguage: LanguagePython,
				Code:           "lambda ctx, instance, event: instance.log.append('entered')",
			},
			assignABI: LanguageTS,
			emit: func(program *Program) ([]byte, error) {
				return NewTypeScriptBackend().Emit(context.Background(), program)
			},
			before:  "function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {",
			comment: "// lambda ctx, instance, event: instance.log.append('entered')",
			after:   "}",
			forbid:  "\tlambda ctx, instance",
		},
		{
			name: "python",
			behavior: Behavior{
				ID:             "entry_1",
				OwnerID:        "state_1",
				Kind:           BehaviorEntry,
				SourceLanguage: LanguageTS,
				Code:           "(ctx, instance, event) => { instance.log.push('entered'); }",
			},
			assignABI: LanguagePython,
			emit: func(program *Program) ([]byte, error) {
				return NewPythonBackend().Emit(context.Background(), program)
			},
			before:  "async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:",
			comment: "# (ctx, instance, event) => { instance.log.push('entered'); }",
			after:   "pass",
			forbid:  "\t(ctx, instance, event) => { instance.log.push('entered'); }",
		},
		{
			name: "rust",
			behavior: Behavior{
				ID:             "entry_1",
				OwnerID:        "state_1",
				Kind:           BehaviorEntry,
				SourceLanguage: LanguageTS,
				Code:           "(ctx, instance, event) => { instance.log.push('entered'); }",
			},
			assignABI: LanguageRust,
			emit: func(program *Program) ([]byte, error) {
				return NewRustBackend().Emit(context.Background(), program)
			},
			before:  "fn entry_1(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event)",
			comment: "// (ctx, instance, event) => { instance.log.push('entered'); }",
			after:   "Box::pin(async {})",
			forbid:  "\t(ctx, instance, event) => { instance.log.push('entered'); }",
		},
		{
			name: "zig",
			behavior: Behavior{
				ID:             "entry_1",
				OwnerID:        "state_1",
				Kind:           BehaviorEntry,
				SourceLanguage: LanguageTS,
				Code:           "(ctx, instance, event) => { instance.log.push('entered'); }",
			},
			assignABI: LanguageZig,
			emit: func(program *Program) ([]byte, error) {
				return NewZigBackend().Emit(context.Background(), program)
			},
			before:  "fn entry_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void {",
			comment: "// (ctx, instance, event) => { instance.log.push('entered'); }",
			after:   "}",
			forbid:  "    (ctx, instance, event) => { instance.log.push('entered'); }",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := &Program{Behaviors: []Behavior{tc.behavior}}
			if tc.prepare != nil {
				tc.prepare(program)
			}
			AssignTargetABI(program, tc.assignABI)

			output, err := tc.emit(program)
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			beforeIndex := strings.Index(text, tc.before)
			commentIndex := strings.Index(text, tc.comment)
			afterIndex := -1
			if commentIndex >= 0 {
				afterIndex = strings.Index(text[commentIndex+len(tc.comment):], tc.after)
			}
			if beforeIndex < 0 || commentIndex < 0 || afterIndex < 0 || beforeIndex >= commentIndex {
				t.Fatalf("%s output did not preserve untranslated behavior code inside body:\n%s", tc.name, text)
			}
			if strings.Contains(text, tc.forbid) {
				t.Fatalf("%s output emitted untranslated behavior code as executable target code:\n%s", tc.name, text)
			}
			header := "Original " + string(tc.behavior.SourceLanguage) + " behavior " + tc.behavior.ID + " preserved for manual porting:"
			if count := strings.Count(text, header); count != 1 {
				t.Fatalf("%s output emitted behavior code preservation header %d times, want 1:\n%s", tc.name, count, text)
			}
		})
	}
}

func TestBackendsPreserveUntranslatedBehaviorBodyAndCodeAsLabeledTargetComments(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			Body:           "instance.Set(\"body\", true)",
			Code:           "func(ctx context.Context, instance hsm.Instance, event hsm.Event) { instance.Set(\"code\", true) }",
		}},
	}
	AssignTargetABI(program, LanguagePython)

	output, err := NewPythonBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:",
		"# Original go behavior entry_1 preserved for manual porting:",
		"# Target language: python",
		"# body:",
		"# instance.Set(\"body\", true)",
		"# code:",
		"# func(ctx context.Context, instance hsm.Instance, event hsm.Event) { instance.Set(\"code\", true) }",
		"pass",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{
		"\tinstance.Set(\"body\", true)",
		"\tfunc(ctx context.Context",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Python output emitted untranslated source as executable code:\n%s", text)
		}
	}
}

func TestBackendsPreserveUntranslatedOperationBehaviorCodeAsTargetCommentInsideBody(t *testing.T) {
	cases := []struct {
		name      string
		behavior  Behavior
		prepare   func(*Program)
		assignABI Language
		emit      func(*Program) ([]byte, error)
		before    string
		comment   string
		after     string
		forbid    string
	}{
		{
			name: "csharp",
			behavior: Behavior{
				ID:             "operation_1",
				OwnerID:        "model_1",
				Kind:           BehaviorOperation,
				SourceLanguage: LanguageTS,
				Code:           "(ctx, instance, event) => { instance.log.push('approved'); }",
			},
			assignABI: LanguageCSharp,
			emit: func(program *Program) ([]byte, error) {
				return NewCSharpBackend().Emit(context.Background(), program)
			},
			before:  "private static void Operation1(Context ctx, Instance instance, Event @event)",
			comment: "// (ctx, instance, event) => { instance.log.push('approved'); }",
			after:   "}",
			forbid:  "\t\t(ctx, instance, event) => { instance.log.push('approved'); }",
		},
		{
			name: "go",
			behavior: Behavior{
				ID:             "operation_1",
				OwnerID:        "model_1",
				Kind:           BehaviorOperation,
				SourceLanguage: LanguageTS,
				Code:           "(ctx, instance, event) => { instance.log.push('approved'); }",
			},
			prepare:   func(program *Program) { program.PackageName = "sample" },
			assignABI: LanguageGo,
			emit: func(program *Program) ([]byte, error) {
				return NewGoBackend().Emit(context.Background(), program)
			},
			before:  "func Operation1(ctx context.Context, instance hsm.Instance, event hsm.Event) {",
			comment: "// (ctx, instance, event) => { instance.log.push('approved'); }",
			after:   "}",
			forbid:  "\t(ctx, instance, event) => { instance.log.push('approved'); }",
		},
		{
			name: "cpp",
			behavior: Behavior{
				ID:             "operation_1",
				OwnerID:        "model_1",
				Kind:           BehaviorOperation,
				SourceLanguage: LanguageTS,
				Code:           "(ctx, instance, event) => { instance.log.push('approved'); }",
			},
			assignABI: LanguageCPP,
			emit: func(program *Program) ([]byte, error) {
				return NewCPPBackend().Emit(context.Background(), program)
			},
			before:  "static void operation_1() {",
			comment: "// (ctx, instance, event) => { instance.log.push('approved'); }",
			after:   "}",
			forbid:  "\t(ctx, instance, event) => { instance.log.push('approved'); }",
		},
		{
			name: "dart",
			behavior: Behavior{
				ID:             "operation_1",
				OwnerID:        "model_1",
				Kind:           BehaviorOperation,
				SourceLanguage: LanguageGo,
				Code:           "func(ctx context.Context, instance hsm.Instance, event hsm.Event) { instance.Set(\"approved\", true) }",
			},
			assignABI: LanguageDart,
			emit: func(program *Program) ([]byte, error) {
				return NewDartBackend().Emit(context.Background(), program)
			},
			before:  "FutureOr<void> operation1(Context ctx, Instance instance, Event event) {",
			comment: "// func(ctx context.Context, instance hsm.Instance, event hsm.Event) { instance.Set(\"approved\", true) }",
			after:   "}",
			forbid:  "\tfunc(ctx context.Context",
		},
		{
			name: "java",
			behavior: Behavior{
				ID:             "operation_1",
				OwnerID:        "model_1",
				Kind:           BehaviorOperation,
				SourceLanguage: LanguageTS,
				Code:           "(ctx, instance, event) => { instance.log.push('approved'); }",
			},
			assignABI: LanguageJava,
			emit: func(program *Program) ([]byte, error) {
				return NewJavaBackend().Emit(context.Background(), program)
			},
			before:  "private static void Operation1(Context ctx, Instance instance, Event event) {",
			comment: "// (ctx, instance, event) => { instance.log.push('approved'); }",
			after:   "}",
			forbid:  "\t\t(ctx, instance, event) => { instance.log.push('approved'); }",
		},
		{
			name: "javascript",
			behavior: Behavior{
				ID:             "operation_1",
				OwnerID:        "model_1",
				Kind:           BehaviorOperation,
				SourceLanguage: LanguagePython,
				Code:           "lambda ctx, instance, event: instance.log.append('approved')",
			},
			assignABI: LanguageJS,
			emit: func(program *Program) ([]byte, error) {
				return NewJavaScriptBackend().Emit(context.Background(), program)
			},
			before:  "function operation1(ctx, instance, event) {",
			comment: "// lambda ctx, instance, event: instance.log.append('approved')",
			after:   "}",
			forbid:  "\tlambda ctx, instance",
		},
		{
			name: "typescript",
			behavior: Behavior{
				ID:             "operation_1",
				OwnerID:        "model_1",
				Kind:           BehaviorOperation,
				SourceLanguage: LanguagePython,
				Code:           "lambda ctx, instance, event: instance.log.append('approved')",
			},
			assignABI: LanguageTS,
			emit: func(program *Program) ([]byte, error) {
				return NewTypeScriptBackend().Emit(context.Background(), program)
			},
			before:  "function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {",
			comment: "// lambda ctx, instance, event: instance.log.append('approved')",
			after:   "}",
			forbid:  "\tlambda ctx, instance",
		},
		{
			name: "python",
			behavior: Behavior{
				ID:             "operation_1",
				OwnerID:        "model_1",
				Kind:           BehaviorOperation,
				SourceLanguage: LanguageTS,
				Code:           "(ctx, instance, event) => { instance.log.push('approved'); }",
			},
			assignABI: LanguagePython,
			emit: func(program *Program) ([]byte, error) {
				return NewPythonBackend().Emit(context.Background(), program)
			},
			before:  "async def operation_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:",
			comment: "# (ctx, instance, event) => { instance.log.push('approved'); }",
			after:   "pass",
			forbid:  "\t(ctx, instance, event) => { instance.log.push('approved'); }",
		},
		{
			name: "rust",
			behavior: Behavior{
				ID:             "operation_1",
				OwnerID:        "model_1",
				Kind:           BehaviorOperation,
				SourceLanguage: LanguageTS,
				Code:           "(ctx, instance, event) => { instance.log.push('approved'); }",
			},
			assignABI: LanguageRust,
			emit: func(program *Program) ([]byte, error) {
				return NewRustBackend().Emit(context.Background(), program)
			},
			before:  "fn operation_1(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event)",
			comment: "// (ctx, instance, event) => { instance.log.push('approved'); }",
			after:   "Box::pin(async {})",
			forbid:  "\t(ctx, instance, event) => { instance.log.push('approved'); }",
		},
		{
			name: "zig",
			behavior: Behavior{
				ID:             "operation_1",
				OwnerID:        "model_1",
				Kind:           BehaviorOperation,
				SourceLanguage: LanguageTS,
				Code:           "(ctx, instance, event) => { instance.log.push('approved'); }",
			},
			assignABI: LanguageZig,
			emit: func(program *Program) ([]byte, error) {
				return NewZigBackend().Emit(context.Background(), program)
			},
			before:  "fn operation_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void {",
			comment: "// (ctx, instance, event) => { instance.log.push('approved'); }",
			after:   "}",
			forbid:  "    (ctx, instance, event) => { instance.log.push('approved'); }",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := &Program{Behaviors: []Behavior{tc.behavior}}
			if tc.prepare != nil {
				tc.prepare(program)
			}
			AssignTargetABI(program, tc.assignABI)

			output, err := tc.emit(program)
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			beforeIndex := strings.Index(text, tc.before)
			commentIndex := strings.Index(text, tc.comment)
			afterIndex := -1
			if commentIndex >= 0 {
				afterIndex = strings.Index(text[commentIndex+len(tc.comment):], tc.after)
			}
			if beforeIndex < 0 || commentIndex < 0 || afterIndex < 0 || beforeIndex >= commentIndex {
				t.Fatalf("%s output did not preserve untranslated operation behavior code inside body:\n%s", tc.name, text)
			}
			if strings.Contains(text, tc.forbid) {
				t.Fatalf("%s output emitted untranslated operation behavior code as executable target code:\n%s", tc.name, text)
			}
			for _, needle := range []string{
				"Original " + string(tc.behavior.SourceLanguage) + " behavior " + tc.behavior.ID + " preserved for manual porting:",
				"Target language: " + string(tc.assignABI),
				"Behavior kind: " + string(tc.behavior.Kind),
			} {
				if !strings.Contains(text, needle) {
					t.Fatalf("%s output missing operation preservation context %q:\n%s", tc.name, needle, text)
				}
			}
			header := "Original " + string(tc.behavior.SourceLanguage) + " behavior " + tc.behavior.ID + " preserved for manual porting:"
			if count := strings.Count(text, header); count != 1 {
				t.Fatalf("%s output emitted operation behavior code preservation header %d times, want 1:\n%s", tc.name, count, text)
			}
		})
	}
}

func TestGoTargetBodyBehaviorImportsContextFromTargetABI(t *testing.T) {
	program := &Program{
		PackageName: "sample",
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageTS,
			TargetLanguage: LanguageGo,
			Body:           `_ = instance`,
		}},
	}
	AssignTargetABI(program, LanguageGo)

	output, err := NewGoBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`"context"`,
		"func Entry1(ctx context.Context, instance hsm.Instance, event hsm.Event)",
		"_ = instance",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
}

func TestBackendsPreserveForeignGlobalsAsTargetComments(t *testing.T) {
	cases := []struct {
		name    string
		global  CodeBlock
		emit    func(*Program) ([]byte, error)
		needles []string
	}{
		{
			name: "go",
			global: CodeBlock{
				ID:       "global_1",
				Language: LanguageTS,
				Code:     "const helper = () => true;",
				Imports: []Import{{
					Path:       "./format",
					Language:   LanguageTS,
					Specifiers: []ImportSpecifier{{Name: "format"}},
				}},
			},
			emit: func(program *Program) ([]byte, error) {
				program.PackageName = "sample"
				return NewGoBackend().Emit(context.Background(), program)
			},
			needles: []string{
				"// Original typescript global global_1 preserved for manual porting:",
				"// Target language: go",
				"// Source imports:",
				`// import { format } from "./format";`,
				"// const helper = () => true;",
			},
		},
		{
			name:   "csharp",
			global: CodeBlock{ID: "global_1", Language: LanguagePython, Code: "def helper():\n\treturn True"},
			emit: func(program *Program) ([]byte, error) {
				return NewCSharpBackend().Emit(context.Background(), program)
			},
			needles: []string{
				"// Original python global global_1 preserved for manual porting:",
				"// Target language: csharp",
				"// def helper():",
				"// \treturn True",
			},
		},
		{
			name:   "cpp",
			global: CodeBlock{ID: "global_1", Language: LanguagePython, Code: "def helper():\n\treturn True"},
			emit: func(program *Program) ([]byte, error) {
				return NewCPPBackend().Emit(context.Background(), program)
			},
			needles: []string{
				"// Original python global global_1 preserved for manual porting:",
				"// Target language: cpp",
				"// def helper():",
				"// \treturn True",
			},
		},
		{
			name:   "dart",
			global: CodeBlock{ID: "global_1", Language: LanguageGo, Code: "func helper() bool { return true }"},
			emit: func(program *Program) ([]byte, error) {
				return NewDartBackend().Emit(context.Background(), program)
			},
			needles: []string{
				"// Original go global global_1 preserved for manual porting:",
				"// Target language: dart",
				"// func helper() bool { return true }",
			},
		},
		{
			name:   "java",
			global: CodeBlock{ID: "global_1", Language: LanguagePython, Code: "def helper():\n\treturn True"},
			emit: func(program *Program) ([]byte, error) {
				return NewJavaBackend().Emit(context.Background(), program)
			},
			needles: []string{
				"// Original python global global_1 preserved for manual porting:",
				"// Target language: java",
				"// def helper():",
				"// \treturn True",
			},
		},
		{
			name:   "javascript",
			global: CodeBlock{ID: "global_1", Language: LanguagePython, Code: "def helper():\n\treturn True"},
			emit: func(program *Program) ([]byte, error) {
				return NewJavaScriptBackend().Emit(context.Background(), program)
			},
			needles: []string{
				"// Original python global global_1 preserved for manual porting:",
				"// Target language: javascript",
				"// def helper():",
				"// \treturn True",
			},
		},
		{
			name:   "typescript",
			global: CodeBlock{ID: "global_1", Language: LanguageGo, Code: "func helper() bool { return true }"},
			emit: func(program *Program) ([]byte, error) {
				return NewTypeScriptBackend().Emit(context.Background(), program)
			},
			needles: []string{
				"// Original go global global_1 preserved for manual porting:",
				"// Target language: typescript",
				"// func helper() bool { return true }",
			},
		},
		{
			name:   "python",
			global: CodeBlock{ID: "global_1", Language: LanguageJS, Code: "const helper = () => true;"},
			emit: func(program *Program) ([]byte, error) {
				return NewPythonBackend().Emit(context.Background(), program)
			},
			needles: []string{
				"# Original javascript global global_1 preserved for manual porting:",
				"# Target language: python",
				"# const helper = () => true;",
			},
		},
		{
			name:   "rust",
			global: CodeBlock{ID: "global_1", Language: LanguagePython, Code: "def helper():\n\treturn True"},
			emit: func(program *Program) ([]byte, error) {
				return NewRustBackend().Emit(context.Background(), program)
			},
			needles: []string{
				"// Original python global global_1 preserved for manual porting:",
				"// Target language: rust",
				"// def helper():",
				"// \treturn True",
			},
		},
		{
			name: "zig",
			global: CodeBlock{
				ID:       "global_1",
				Language: LanguagePython,
				Code:     "def helper():\n\treturn True",
				Imports:  []Import{{Path: "helpers", Language: LanguagePython, Specifiers: []ImportSpecifier{{Name: "helper"}}}},
			},
			emit: func(program *Program) ([]byte, error) {
				return NewZigBackend().Emit(context.Background(), program)
			},
			needles: []string{
				"// Original python global global_1 preserved for manual porting:",
				"// Target language: zig",
				"// Source imports:",
				"// from helpers import helper",
				"// def helper():",
				"//     return True",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := tc.emit(&Program{Globals: []CodeBlock{tc.global}})
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			for _, needle := range tc.needles {
				if !strings.Contains(text, needle) {
					t.Fatalf("%s output missing %q:\n%s", tc.name, needle, text)
				}
			}
			header := "Original " + string(tc.global.Language) + " global " + tc.global.ID + " preserved for manual porting:"
			if count := strings.Count(text, header); count != 1 {
				t.Fatalf("%s output emitted global preservation header %d times, want 1:\n%s", tc.name, count, text)
			}
		})
	}
}
