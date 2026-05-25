package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestValidateGeneratedTopLevelSymbolUniquenessRejectsCompilerOwnedCollisions(t *testing.T) {
	zigEntryOperationFallback := zigOperationReferenceFallbackName(BehaviorEntry, "approve")
	tests := []struct {
		name    string
		target  Language
		program Program
		want    string
	}{
		{
			name:   "go event behavior",
			target: LanguageGo,
			program: Program{
				Events:    []EventDecl{{ID: "event_1", Symbol: "entry_1", Name: "entry"}},
				Behaviors: []Behavior{{ID: "entry_1", Kind: BehaviorEntry}},
			},
			want: `compiler-owned go symbol "Entry1" for behavior "entry_1" collides with event "event_1"`,
		},
		{
			name:   "csharp behavior scaffold",
			target: LanguageCSharp,
			program: Program{
				Behaviors: []Behavior{{ID: "generated_hsm", Kind: BehaviorEntry}},
			},
			want: `compiler-owned csharp symbol "GeneratedHsm" for compiler scaffold collides with behavior "generated_hsm"`,
		},
		{
			name:   "cpp event deferred event type",
			target: LanguageCPP,
			program: Program{
				Events: []EventDecl{{ID: "event_1", Symbol: "open", Name: "Timeout"}},
				Models: []Model{{
					ID:   "model_1",
					Name: "Door",
					States: []State{{
						ID:     "state_1",
						Name:   "closed",
						Defers: []string{"Timeout"},
					}},
				}},
			},
			want: `compiler-owned cpp symbol "TimeoutEvent" for deferred event type "Timeout" collides with event type for event "event_1"`,
		},
		{
			name:   "python behavior trigger fallback scaffold",
			target: LanguagePython,
			program: Program{
				Behaviors: []Behavior{{ID: "hsmc_trigger_fallback_1", Kind: BehaviorTrigger}},
				Models: []Model{{
					ID:   "model_1",
					Name: "Door",
					States: []State{{
						ID:   "state_1",
						Name: "closed",
						Transitions: []Transition{{
							ID:      "transition_1",
							OwnerID: "state_1",
							Trigger: &Trigger{Kind: TriggerAt, Language: LanguageTS, Value: "deadline()"},
						}},
					}},
				}},
			},
			want: `compiler-owned python symbol "hsmc_trigger_fallback_1" for compiler scaffold collides with behavior "hsmc_trigger_fallback_1"`,
		},
		{
			name:   "zig behavior operation fallback scaffold",
			target: LanguageZig,
			program: Program{
				Behaviors: []Behavior{{ID: zigEntryOperationFallback, Kind: BehaviorEntry}},
				Models: []Model{{
					ID:         "model_1",
					Name:       "Door",
					Operations: []Operation{{ID: "operation_1", Name: "approve"}},
					States: []State{{
						ID:        "state_1",
						Name:      "closed",
						Behaviors: []BehaviorRef{{Kind: BehaviorEntry, Operation: "approve"}},
					}},
				}},
			},
			want: `compiler-owned zig symbol "` + zigEntryOperationFallback + `" for compiler scaffold collides with behavior "` + zigEntryOperationFallback + `"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGeneratedTopLevelSymbolUniqueness(&tt.program, tt.target)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestValidateGeneratedTopLevelSymbolUniquenessRejectsInvalidTargetIdentifiers(t *testing.T) {
	tests := []struct {
		name    string
		target  Language
		program Program
		want    string
	}{
		{
			name:   "python event keyword",
			target: LanguagePython,
			program: Program{
				Events: []EventDecl{{ID: "event_1", Symbol: "class", Name: "class"}},
			},
			want: `compiler-owned python symbol "class" for event "event_1" is not a valid target identifier`,
		},
		{
			name:   "typescript behavior keyword",
			target: LanguageTS,
			program: Program{
				Behaviors: []Behavior{{ID: "delete", Kind: BehaviorEntry}},
			},
			want: `compiler-owned typescript symbol "delete" for behavior "delete" is not a valid target identifier`,
		},
		{
			name:   "cpp behavior keyword",
			target: LanguageCPP,
			program: Program{
				Behaviors: []Behavior{{ID: "class", Kind: BehaviorEntry}},
			},
			want: `compiler-owned cpp symbol "class" for behavior "class" is not a valid target identifier`,
		},
		{
			name:   "zig event keyword",
			target: LanguageZig,
			program: Program{
				Events: []EventDecl{{ID: "event_1", Symbol: "const", Name: "const"}},
			},
			want: `compiler-owned zig symbol "const" for event "event_1" is not a valid target identifier`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGeneratedTopLevelSymbolUniqueness(&tt.program, tt.target)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestValidateTargetPackageNameRejectsInvalidGoPackageNames(t *testing.T) {
	tests := []struct {
		name        string
		packageName string
		want        string
	}{
		{name: "whitespace", packageName: "sample ", want: `target go package_name "sample " has leading or trailing whitespace`},
		{name: "keyword", packageName: "type", want: `target go package_name "type" is not a valid Go package identifier`},
		{name: "path", packageName: "example/sample", want: `target go package_name "example/sample" is not a valid Go package identifier`},
		{name: "blank identifier", packageName: "_", want: `target go package_name "_" is not a valid Go package identifier`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTargetPackageName(&Program{PackageName: tt.packageName}, LanguageGo)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestCompileRejectsInvalidJSONIRPackageNameForGoTarget(t *testing.T) {
	source := `{
  "source_language": "typescript",
  "package_name": "sample bad",
  "models": []
}`
	_, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err == nil || !strings.Contains(err.Error(), `target go package_name "sample bad" is not a valid Go package identifier`) {
		t.Fatalf("error = %v, want invalid Go package name", err)
	}
}

func TestValidateTargetGlobalSymbolCollisionsRejectsCompilerOwnedCollisions(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageTS,
			Code:     "function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {}",
		}},
		Behaviors: []Behavior{{ID: "entry_1", Kind: BehaviorEntry}},
	}

	err := validateTargetGlobalSymbolCollisions(&program, LanguageTS)
	want := `target-language global "global_1" declares "entry1", which collides with compiler-owned typescript symbol`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestValidateTargetGlobalSymbolCollisionsRejectsLaterCompilerOwnedDeclarations(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageTS,
			Code: `const helper = (value: string): string => value;
function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {}`,
		}},
		Behaviors: []Behavior{{ID: "entry_1", Kind: BehaviorEntry}},
	}

	err := validateTargetGlobalSymbolCollisions(&program, LanguageTS)
	want := `target-language global "global_1" declares "entry1", which collides with compiler-owned typescript symbol`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestValidateTargetGlobalSymbolCollisionsRejectsSameStatementCompilerOwnedDeclarations(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageTS,
			Code:     `const helper = (value: string): string => value, entry1 = (ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void => {};`,
		}},
		Behaviors: []Behavior{{ID: "entry_1", Kind: BehaviorEntry}},
	}

	err := validateTargetGlobalSymbolCollisions(&program, LanguageTS)
	want := `target-language global "global_1" declares "entry1", which collides with compiler-owned typescript symbol`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestValidateTargetGlobalSymbolCollisionsRejectsGoGroupedCompilerOwnedDeclarations(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageGo,
			Code: `var (
	Helper = func(value string) string { return value }
	Entry1 = func(ctx context.Context, instance hsm.Instance, event hsm.Event) {}
)`,
		}},
		Behaviors: []Behavior{{ID: "entry_1", Kind: BehaviorEntry}},
	}

	err := validateTargetGlobalSymbolCollisions(&program, LanguageGo)
	want := `target-language global "global_1" declares "Entry1", which collides with compiler-owned go symbol`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestValidateTargetGlobalSymbolCollisionsRejectsGoSameStatementCompilerOwnedDeclarations(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageGo,
			Code:     `var Helper, Entry1 func(ctx context.Context, instance hsm.Instance, event hsm.Event) = func(ctx context.Context, instance hsm.Instance, event hsm.Event) {}, func(ctx context.Context, instance hsm.Instance, event hsm.Event) {}`,
		}},
		Behaviors: []Behavior{{ID: "entry_1", Kind: BehaviorEntry}},
	}

	err := validateTargetGlobalSymbolCollisions(&program, LanguageGo)
	want := `target-language global "global_1" declares "Entry1", which collides with compiler-owned go symbol`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestValidateTargetGlobalSymbolCollisionsRejectsGoSameStatementCompilerOwnedDeclarationWithoutInitializer(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageGo,
			Code:     `var Helper, Entry1 func(ctx context.Context, instance hsm.Instance, event hsm.Event)`,
		}},
		Behaviors: []Behavior{{ID: "entry_1", Kind: BehaviorEntry}},
	}

	err := validateTargetGlobalSymbolCollisions(&program, LanguageGo)
	want := `target-language global "global_1" declares "Entry1", which collides with compiler-owned go symbol`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestValidateTargetGlobalSymbolCollisionsRejectsLaterGoGroupedCompilerOwnedDeclarations(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageGo,
			Code: `func Helper(value string) string { return value }

var (
	Entry1 = func(ctx context.Context, instance hsm.Instance, event hsm.Event) {}
)`,
		}},
		Behaviors: []Behavior{{ID: "entry_1", Kind: BehaviorEntry}},
	}

	err := validateTargetGlobalSymbolCollisions(&program, LanguageGo)
	want := `target-language global "global_1" declares "Entry1", which collides with compiler-owned go symbol`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestValidateTargetGlobalSymbolCollisionsRejectsCPPSameStatementCompilerOwnedDeclarations(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageCPP,
			Code:     `static constexpr auto helper = 0, entry_1 = []() {};`,
		}},
		Behaviors: []Behavior{{ID: "entry_1", Kind: BehaviorEntry}},
	}

	err := validateTargetGlobalSymbolCollisions(&program, LanguageCPP)
	want := `target-language global "global_1" declares "entry_1", which collides with compiler-owned cpp symbol`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestValidateTargetGlobalSymbolCollisionsRejectsCPPFunctionPointerDeclarations(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageCPP,
			Code:     `void (*entry_1)(hsm::Context&, HsmcInstance&, const hsm::Event&) = nullptr;`,
		}},
		Behaviors: []Behavior{{ID: "entry_1", Kind: BehaviorEntry}},
	}

	err := validateTargetGlobalSymbolCollisions(&program, LanguageCPP)
	want := `target-language global "global_1" declares "entry_1", which collides with compiler-owned cpp symbol`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestValidateTargetGlobalSymbolCollisionsRejectsCPPFunctionPrototypes(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageCPP,
			Code:     `void entry_1(auto& signal, auto& instance, const auto& event);`,
		}},
		Behaviors: []Behavior{{ID: "entry_1", Kind: BehaviorEntry}},
	}

	err := validateTargetGlobalSymbolCollisions(&program, LanguageCPP)
	want := `target-language global "global_1" declares "entry_1", which collides with compiler-owned cpp symbol`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestValidateTargetGlobalSymbolCollisionsRejectsJavaArraySuffixDeclarations(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageJava,
			Code:     `private static Event Helper, Entry1[];`,
		}},
		Behaviors: []Behavior{{ID: "entry_1", Kind: BehaviorEntry}},
	}

	err := validateTargetGlobalSymbolCollisions(&program, LanguageJava)
	want := `target-language global "global_1" declares "Entry1", which collides with compiler-owned java symbol`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestValidateTargetGlobalSymbolCollisionsRejectsZigExternAndThreadlocalDeclarations(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{
			name: "extern",
			code: `pub extern var entry_1: ?*const fn (*hsm.Context, *hsm.Instance, hsm.Event) void;`,
		},
		{
			name: "threadlocal",
			code: `pub threadlocal var entry_1: ?*const fn (*hsm.Context, *hsm.Instance, hsm.Event) void = null;`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := Program{
				Globals: []CodeBlock{{
					ID:       "global_1",
					Language: LanguageZig,
					Code:     tt.code,
				}},
				Behaviors: []Behavior{{ID: "entry_1", Kind: BehaviorEntry}},
			}

			err := validateTargetGlobalSymbolCollisions(&program, LanguageZig)
			want := `target-language global "global_1" declares "entry_1", which collides with compiler-owned zig symbol`
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want %q", err, want)
			}
		})
	}
}

func TestValidateTargetGlobalSymbolCollisionsRejectsCSharpPropertyDeclarations(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageCSharp,
			Code:     `private static Model DoorModel { get; } = Hsm.Define("Door");`,
		}},
		Models: []Model{{ID: "model:Door", Name: "Door"}},
	}

	err := validateTargetGlobalSymbolCollisions(&program, LanguageCSharp)
	want := `target-language global "global_1" declares "DoorModel", which collides with compiler-owned csharp symbol`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestValidateTargetGlobalSymbolCollisionsRejectsCSharpAliasUsingDeclarations(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageCSharp,
			Code:     `using Entry1 = Sample.Helpers.Entry1;`,
		}},
		Behaviors: []Behavior{{ID: "entry_1", Kind: BehaviorEntry}},
	}

	err := validateTargetGlobalSymbolCollisions(&program, LanguageCSharp)
	want := `target-language global "global_1" declares "Entry1", which collides with compiler-owned csharp symbol`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestValidateTargetGlobalSymbolCollisionsRejectsCSharpExternAliasDeclarations(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageCSharp,
			Code:     `extern alias Entry1;`,
		}},
		Behaviors: []Behavior{{ID: "entry_1", Kind: BehaviorEntry}},
	}

	err := validateTargetGlobalSymbolCollisions(&program, LanguageCSharp)
	want := `target-language global "global_1" declares "Entry1", which collides with compiler-owned csharp symbol`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestValidateTargetGlobalSymbolCollisionsRejectsCSharpMethodPrototypes(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageCSharp,
			Code:     `private static extern void Entry1(Context ctx, Instance instance, Event @event);`,
		}},
		Behaviors: []Behavior{{ID: "entry_1", Kind: BehaviorEntry}},
	}

	err := validateTargetGlobalSymbolCollisions(&program, LanguageCSharp)
	want := `target-language global "global_1" declares "Entry1", which collides with compiler-owned csharp symbol`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestValidateTargetGlobalSymbolCollisionsRejectsJavaMethodPrototypes(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageJava,
			Code:     `private static abstract void Entry1(Context ctx, Instance instance, Event event);`,
		}},
		Behaviors: []Behavior{{ID: "entry_1", Kind: BehaviorEntry}},
	}

	err := validateTargetGlobalSymbolCollisions(&program, LanguageJava)
	want := `target-language global "global_1" declares "Entry1", which collides with compiler-owned java symbol`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestValidateTargetGlobalSymbolCollisionsRejectsDartExtensionTypeDeclarations(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageDart,
			Code: `extension type entry1(int value) {
	int call() => value;
}`,
		}},
		Behaviors: []Behavior{{ID: "entry_1", Kind: BehaviorEntry}},
	}

	err := validateTargetGlobalSymbolCollisions(&program, LanguageDart)
	want := `target-language global "global_1" declares "entry1", which collides with compiler-owned dart symbol`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestValidateTargetGlobalSymbolCollisionsIgnoresForeignGlobals(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageGo,
			Code:     "func entry1() {}",
		}},
		Behaviors: []Behavior{{ID: "entry_1", Kind: BehaviorEntry}},
	}

	if err := validateTargetGlobalSymbolCollisions(&program, LanguageTS); err != nil {
		t.Fatalf("foreign global collision should be ignored: %v", err)
	}
}

func TestValidateTargetGlobalRuntimeBindingCollisionsRejectsRuntimeBindingDeclarations(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageTS,
			Code:     `const hsm = createHelperRuntime();`,
		}},
	}

	err := validateTargetGlobalRuntimeBindingCollisions(&program, LanguageTS)
	want := `target-language global "global_1" declares "hsm", which collides with compiler-owned typescript runtime binding`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestValidateTargetGlobalRuntimeBindingCollisionsIgnoresForeignGlobals(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageGo,
			Code:     `var hsm = createHelperRuntime()`,
		}},
	}

	if err := validateTargetGlobalRuntimeBindingCollisions(&program, LanguageTS); err != nil {
		t.Fatalf("foreign global runtime binding collision should be ignored: %v", err)
	}
}

func TestValidateTargetGlobalDeclarationUniquenessRejectsDuplicateTargetGlobals(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{
			{ID: "global_1", Language: LanguageTS, Code: `const helper = (value: string) => value;`},
			{ID: "global_2", Language: LanguageTS, Code: `function helper(value: string): string { return value; }`},
		},
	}

	err := validateTargetGlobalDeclarationUniqueness(&program, LanguageTS)
	want := `target-language global "global_2" declares "helper", which collides with target-language global "global_1"`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestValidateTargetGlobalDeclarationUniquenessRejectsDuplicateDeclarationsInSameGlobal(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageTS,
			Code: `const helper = (value: string) => value;
function helper(value: string): string { return value; }`,
		}},
	}

	err := validateTargetGlobalDeclarationUniqueness(&program, LanguageTS)
	want := `target-language global "global_1" declares "helper" more than once`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestValidateTargetImportGlobalDeclarationCollisionsRejectsImportBindingGlobalCollision(t *testing.T) {
	program := Program{
		Imports: []Import{{
			Path:       "./format",
			Language:   LanguageTS,
			Specifiers: []ImportSpecifier{{Name: "format"}},
		}},
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageTS,
			Code:     `const format = (value: string) => value;`,
		}},
	}

	err := validateTargetImportGlobalDeclarationCollisions(&program, LanguageTS)
	want := `target imports for program import "./format" bind local name "format", which collides with target-language global "global_1"`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestValidateTargetImportGlobalDeclarationCollisionsAllowsCSharpNamespaceUsingMatchingGlobal(t *testing.T) {
	program := Program{
		Imports: []Import{{
			Path:     "Sample.Helpers",
			Language: LanguageCSharp,
		}},
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageCSharp,
			Code:     `private static class Helpers {}`,
		}},
	}

	if err := validateTargetImportGlobalDeclarationCollisions(&program, LanguageCSharp); err != nil {
		t.Fatalf("ordinary C# namespace using should not collide with helper global: %v", err)
	}
}

func TestValidateTargetImportGlobalDeclarationCollisionsIgnoresForeignImportsAndGlobals(t *testing.T) {
	program := Program{
		Imports: []Import{{Path: "./format", Language: LanguageTS, Specifiers: []ImportSpecifier{{Name: "format"}}}},
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageGo,
			Code:     `var format = func(value string) string { return value }`,
		}},
	}

	if err := validateTargetImportGlobalDeclarationCollisions(&program, LanguageTS); err != nil {
		t.Fatalf("foreign global import/declaration collision should be ignored: %v", err)
	}
}

func TestValidateTargetGlobalDeclarationUniquenessIgnoresForeignGlobals(t *testing.T) {
	program := Program{
		Globals: []CodeBlock{
			{ID: "global_1", Language: LanguageGo, Code: `func helper(value string) string { return value }`},
			{ID: "global_2", Language: LanguageTS, Code: `function helper(value: string): string { return value; }`},
		},
	}

	if err := validateTargetGlobalDeclarationUniqueness(&program, LanguageTS); err != nil {
		t.Fatalf("foreign global duplicate should be ignored: %v", err)
	}
}

func TestCompileRejectsDuplicateTargetGlobalDeclarationsFromJSONIR(t *testing.T) {
	source := `{
  "source_language": "typescript",
  "globals": [
    {
      "id": "global_1",
      "language": "typescript",
      "code": "const helper = (value: string) => value;"
    },
    {
      "id": "global_2",
      "language": "typescript",
      "code": "function helper(value: string): string { return value; }"
    }
  ],
  "models": [
    {
      "id": "model_1",
      "name": "Door",
      "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
      "states": [
        { "id": "state_1", "name": "closed", "kind": "state" }
      ]
    }
  ]
}`

	_, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageTS,
		Adapter: "none",
	})
	want := `target-language global "global_2" declares "helper", which collides with target-language global "global_1"`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}
