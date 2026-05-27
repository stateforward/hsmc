package hsmc

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestSupportedLanguageMatrixCompilesAnyToAny(t *testing.T) {
	ctx := context.Background()
	compiler := NewCompiler()
	sources := implementationMatrixSources()
	ir, _, err := compiler.Compile(ctx, SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageJSONIR,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	sources = append(sources, matrixSource{name: "json-ir", language: LanguageJSONIR, path: "door.hsm.json", data: ir})

	var targets []struct {
		name     string
		language Language
	}
	for _, language := range compiler.TargetLanguages() {
		targets = append(targets, struct {
			name     string
			language Language
		}{name: string(language), language: language})
	}

	for _, source := range sources {
		for _, target := range targets {
			t.Run(source.name+"_to_"+target.name, func(t *testing.T) {
				output, program, err := compiler.Compile(ctx, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
					From:    source.language,
					To:      target.language,
					Adapter: "none",
				})
				if err != nil {
					t.Fatal(err)
				}
				if len(output) == 0 {
					t.Fatal("empty output")
				}
				if program == nil || len(program.Models) == 0 {
					t.Fatalf("missing compiled program/model: %#v", program)
				}
				for _, behavior := range program.Behaviors {
					if behavior.TargetABI == nil || behavior.TargetABI.Language != target.language {
						t.Fatalf("behavior %s target ABI = %#v, want %s", behavior.ID, behavior.TargetABI, target.language)
					}
				}
				assertNoAdapterPreservesForeignBehaviorsAsComments(t, string(output), program, target.language)
				assertNoAdapterPreservesForeignGlobalsAsComments(t, string(output), program, target.language)
			})
		}
	}
}

func TestSupportedLanguageMatrixRoundTripsEverySourceThroughJSONIR(t *testing.T) {
	ctx := context.Background()
	compiler := NewCompiler()
	targets := []Language{LanguageCSharp, LanguageCPP, LanguageDart, LanguageElixir, LanguageGo, LanguageJava, LanguageJS, LanguagePython, LanguageTS, LanguageRust, LanguageZig}
	for _, source := range implementationMatrixSources() {
		source := source
		t.Run(source.name, func(t *testing.T) {
			ir, irProgram, err := compiler.Compile(ctx, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
				From:    source.language,
				To:      LanguageJSONIR,
				Adapter: "none",
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(ir) == 0 || irProgram == nil || irProgram.SourceLanguage != source.language {
				t.Fatalf("json-ir output/program = %d bytes, %#v; want source %s", len(ir), irProgram, source.language)
			}
			transportProgram, err := NewJSONIRFrontend().Parse(ctx, SourceInput{Path: source.name + ".hsm.json", Data: ir})
			if err != nil {
				t.Fatal(err)
			}
			if transportProgram.SourceLanguage != source.language {
				t.Fatalf("json-ir transport source = %s, want %s", transportProgram.SourceLanguage, source.language)
			}
			for _, behavior := range transportProgram.Behaviors {
				if behavior.TargetLanguage != "" || behavior.TargetABI != nil {
					t.Fatalf("json-ir output leaked target behavior metadata: %#v", behavior)
				}
			}

			for _, target := range targets {
				target := target
				t.Run(string(target), func(t *testing.T) {
					output, program, err := compiler.Compile(ctx, SourceInput{Path: source.name + ".hsm.json", Data: ir}, CompileOptions{
						From:    LanguageJSONIR,
						To:      target,
						Adapter: "none",
					})
					if err != nil {
						t.Fatal(err)
					}
					if len(output) == 0 {
						t.Fatal("empty output")
					}
					if program == nil || program.SourceLanguage != source.language || len(program.Models) == 0 {
						t.Fatalf("json-ir target program = %#v, want source %s with model", program, source.language)
					}
					for _, behavior := range program.Behaviors {
						if behavior.TargetABI == nil || behavior.TargetABI.Language != target {
							t.Fatalf("behavior %s target ABI = %#v, want %s", behavior.ID, behavior.TargetABI, target)
						}
					}
					assertNoAdapterPreservesForeignBehaviorsAsComments(t, string(output), program, target)
					assertNoAdapterPreservesForeignGlobalsAsComments(t, string(output), program, target)
				})
			}
		})
	}
}

func TestSupportedLanguageMatrixCompilesAnyToAnyWithAdapterPatch(t *testing.T) {
	ctx := context.Background()
	compiler := NewCompiler()
	adapter := &matrixAdapter{}
	compiler.RegisterAdapter(adapter)

	sources := matrixSources(t, compiler)
	targets := []Language{LanguageCSharp, LanguageCPP, LanguageDart, LanguageGo, LanguageJava, LanguageJS, LanguagePython, LanguageTS, LanguageRust, LanguageZig}
	for _, source := range sources {
		for _, target := range targets {
			t.Run(source.name+"_to_"+string(target), func(t *testing.T) {
				_, baselineProgram, err := compiler.Compile(ctx, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
					From:    source.language,
					To:      target,
					Adapter: "none",
				})
				if err != nil {
					t.Fatal(err)
				}
				baselineModels := BuildModelViews(baselineProgram)
				output, program, err := compiler.Compile(ctx, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
					From:    source.language,
					To:      target,
					Adapter: adapter.Name(),
				})
				if err != nil {
					t.Fatal(err)
				}
				assertMatrixAdapterOutput(t, target, string(output))
				if program == nil || len(program.Behaviors) == 0 {
					t.Fatalf("missing program behaviors: %#v", program)
				}
				if got := BuildModelViews(program); !reflect.DeepEqual(got, baselineModels) {
					t.Fatalf("adapter changed compiler-owned model view:\ngot:  %#v\nwant: %#v", got, baselineModels)
				}
				for _, behavior := range program.Behaviors {
					if behavior.TargetLanguage != target {
						t.Fatalf("behavior %s target language = %q, want %q", behavior.ID, behavior.TargetLanguage, target)
					}
				}
			})
		}
	}
}

func TestOperationReferenceMatrixCompilesSourceToTargets(t *testing.T) {
	ctx := context.Background()
	compiler := NewCompiler()
	targets := []struct {
		language Language
		want     []string
		forbid   []string
	}{
		{
			language: LanguageCSharp,
			want: []string{
				`Hsm.Effect<Instance>((ctx, instance, @event) => Hsm.Call(ctx, instance, "approve"))`,
				`Hsm.Entry<Instance>((ctx, instance, @event) => Hsm.Call(ctx, instance, "approve"))`,
				`Hsm.Exit<Instance>((ctx, instance, @event) => Hsm.Call(ctx, instance, "approve"))`,
				`Hsm.Activity<Instance>((ctx, instance, @event) => Hsm.Call(ctx, instance, "approve"))`,
				`Hsm.Guard<Instance>((ctx, instance, @event) => Hsm.Call(ctx, instance, "approve") is true)`,
			},
			forbid: []string{`Hsm.Effect<Instance>(Approve)`, `Hsm.Entry<Instance>(Approve)`, `Hsm.Exit<Instance>(Approve)`, `Hsm.Activity<Instance>(Approve)`, `Hsm.Guard<Instance>(Approve)`},
		},
		{
			language: LanguageCPP,
			want:     []string{`hsm::effect("approve")`, `hsm::entry("approve")`, `hsm::exit("approve")`, `hsm::activity("approve")`, `hsm::guard("approve")`},
			forbid:   []string{`hsm::effect(approve)`, `hsm::entry(approve)`, `hsm::exit(approve)`, `hsm::activity(approve)`, `hsm::guard(approve)`},
		},
		{
			language: LanguageDart,
			want:     []string{`effect("approve")`, `entry("approve")`, `exit("approve")`, `activity("approve")`, `guard("approve")`},
			forbid:   []string{`effect(approve)`, `entry(approve)`, `exit(approve)`, `activity(approve)`, `guard(approve)`},
		},
		{
			language: LanguageGo,
			want:     []string{`hsm.Effect("approve")`, `hsm.Entry("approve")`, `hsm.Exit("approve")`, `hsm.Activity("approve")`, `hsm.Guard("approve")`},
			forbid:   []string{`hsm.Effect(approve)`, `hsm.Entry(approve)`, `hsm.Exit(approve)`, `hsm.Activity(approve)`, `hsm.Guard(approve)`},
		},
		{
			language: LanguageJava,
			want: []string{
				`Hsm.Effect((ctx, instance, event) -> Hsm.Call(ctx, instance, "approve"))`,
				`Hsm.Entry((ctx, instance, event) -> Hsm.Call(ctx, instance, "approve"))`,
				`Hsm.Exit((ctx, instance, event) -> Hsm.Call(ctx, instance, "approve"))`,
				`Hsm.Activity((ctx, instance, event) -> Hsm.Call(ctx, instance, "approve"))`,
				`Hsm.Guard((ctx, instance, event) -> Boolean.TRUE.equals(Hsm.Call(ctx, instance, "approve")))`,
			},
			forbid: []string{`Hsm.Effect(GeneratedHsm::Approve)`, `Hsm.Entry(GeneratedHsm::Approve)`, `Hsm.Exit(GeneratedHsm::Approve)`, `Hsm.Activity(GeneratedHsm::Approve)`, `Hsm.Guard(GeneratedHsm::Approve)`},
		},
		{
			language: LanguageJS,
			want:     []string{`hsm.Effect("approve")`, `hsm.Entry("approve")`, `hsm.Exit("approve")`, `hsm.Activity("approve")`, `hsm.Guard("approve")`},
			forbid:   []string{`hsm.Effect(approve)`, `hsm.Entry(approve)`, `hsm.Exit(approve)`, `hsm.Activity(approve)`, `hsm.Guard(approve)`},
		},
		{
			language: LanguagePython,
			want:     []string{`hsm.Effect("approve")`, `hsm.Entry("approve")`, `hsm.Exit("approve")`, `hsm.Activity("approve")`, `hsm.Guard("approve")`},
			forbid:   []string{`hsm.Effect(approve)`, `hsm.Entry(approve)`, `hsm.Exit(approve)`, `hsm.Activity(approve)`, `hsm.Guard(approve)`},
		},
		{
			language: LanguageRust,
			want: []string{
				`hsm::effect_operation("approve")`,
				`hsm::entry_operation("approve")`,
				`hsm::exit_operation("approve")`,
				`hsm::activity_operation("approve")`,
				`hsm::guard_operation_ref("approve")`,
			},
			forbid: []string{`hsm::effect(approve)`, `hsm::entry(approve)`, `hsm::exit(approve)`, `hsm::activity(approve)`, `hsm::guard(approve)`},
		},
		{
			language: LanguageTS,
			want:     []string{`hsm.Effect("approve")`, `hsm.Entry("approve")`, `hsm.Exit("approve")`, `hsm.Activity("approve")`, `hsm.Guard("approve")`},
			forbid:   []string{`hsm.Effect(approve)`, `hsm.Entry(approve)`, `hsm.Exit(approve)`, `hsm.Activity(approve)`, `hsm.Guard(approve)`},
		},
		{
			language: LanguageZig,
			want: []string{
				`// Operation reference "approve" preserved for manual porting; hsm.zig behavior callbacks cannot call model operations without a state machine handle.`,
				zigOperationReferenceFallbackName(BehaviorEffect, "approve"),
				zigOperationReferenceFallbackName(BehaviorEntry, "approve"),
				zigOperationReferenceFallbackName(BehaviorExit, "approve"),
				zigOperationReferenceFallbackName(BehaviorActivity, "approve"),
				zigOperationReferenceFallbackName(BehaviorGuard, "approve"),
			},
			forbid: []string{`hsm.effect(approve)`, `hsm.entry(approve)`, `hsm.exit(approve)`, `hsm.activity(approve)`, `hsm.guard(approve)`},
		},
	}

	for _, target := range targets {
		target := target
		t.Run(string(target.language), func(t *testing.T) {
			output, program, err := compiler.Compile(ctx, SourceInput{Path: "door.go", Data: []byte(goOperationReferenceMatrixSource)}, CompileOptions{
				From:    LanguageGo,
				To:      target.language,
				Adapter: "none",
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(program.Models) != 1 || len(program.Models[0].Operations) != 1 {
				t.Fatalf("compiled operations = %#v", program.Models)
			}
			text := string(output)
			for _, needle := range target.want {
				if !strings.Contains(text, needle) {
					t.Fatalf("%s output missing operation reference %q:\n%s", target.language, needle, text)
				}
			}
			for _, forbidden := range target.forbid {
				if strings.Contains(text, forbidden) {
					t.Fatalf("%s output emitted operation reference as target identifier %q:\n%s", target.language, forbidden, text)
				}
			}
		})
	}
}

func TestNamedBehaviorBodyExtractionMatrix(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name     string
		frontend Frontend
		path     string
		data     []byte
		want     map[BehaviorKind]string
	}{
		{
			name:     "csharp",
			frontend: NewCSharpFrontend(),
			path:     "named.cs",
			data:     []byte(namedCSharpBehaviorMatrixSource),
			want: map[BehaviorKind]string{
				BehaviorEntry: `instance.Set("entered", true);`,
				BehaviorGuard: `return @event.Name == "open";`,
			},
		},
		{
			name:     "cpp",
			frontend: NewCPPFrontend(),
			path:     "door.cpp",
			data:     []byte(cppDoorSource),
			want: map[BehaviorKind]string{
				BehaviorEntry: `format::record("closed");`,
				BehaviorGuard: `return door.count >= 0;`,
			},
		},
		{
			name:     "dart",
			frontend: NewDartFrontend(),
			path:     "named.dart",
			data:     []byte(dartNamedBehaviorSource),
			want: map[BehaviorKind]string{
				BehaviorEntry: `instance.set('entered', true);`,
				BehaviorGuard: `return event.name == 'open';`,
			},
		},
		{
			name:     "go",
			frontend: NewGoFrontend(),
			path:     "named.go",
			data:     []byte(namedGoBehaviorMatrixSource),
			want: map[BehaviorKind]string{
				BehaviorEntry: `instance.Set("entered", true)`,
				BehaviorGuard: `return event.Name == "open"`,
			},
		},
		{
			name:     "java",
			frontend: NewJavaFrontend(),
			path:     "DoorHsm.java",
			data:     []byte(namedJavaBehaviorMatrixSource),
			want: map[BehaviorKind]string{
				BehaviorEntry: `instance.set("entered", true);`,
				BehaviorGuard: `return event.name().equals("open");`,
			},
		},
		{
			name:     "javascript",
			frontend: NewJavaScriptFrontend(),
			path:     "door.js",
			data:     []byte(jsDoorSource),
			want: map[BehaviorKind]string{
				BehaviorEntry:  `rec("entered");`,
				BehaviorEffect: `rec("entered");`,
			},
		},
		{
			name:     "python",
			frontend: NewPythonFrontend(),
			path:     "door.py",
			data:     []byte(pythonDoorSource),
			want: map[BehaviorKind]string{
				BehaviorEntry:  `rec("entered")`,
				BehaviorEffect: `rec("entered")`,
			},
		},
		{
			name:     "rust",
			frontend: NewRustFrontend(),
			path:     "door.rs",
			data:     []byte(rustDoorSource),
			want: map[BehaviorKind]string{
				BehaviorEntry: `record("entered");`,
				BehaviorGuard: `true`,
			},
		},
		{
			name:     "typescript",
			frontend: NewTypeScriptFrontend(),
			path:     "named.ts",
			data:     []byte(namedTypeScriptBehaviorMatrixSource),
			want: map[BehaviorKind]string{
				BehaviorEntry: `instance.set("entered", true);`,
				BehaviorGuard: `return event.name === "open";`,
			},
		},
		{
			name:     "zig",
			frontend: NewZigFrontend(),
			path:     "door.zig",
			data:     []byte(zigDoorSource),
			want: map[BehaviorKind]string{
				BehaviorEntry: `record("closed");`,
				BehaviorGuard: `return true;`,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program, err := tc.frontend.Parse(ctx, SourceInput{Path: tc.path, Data: tc.data})
			if err != nil {
				t.Fatal(err)
			}
			if len(program.Behaviors) == 0 {
				t.Fatalf("%s produced no behaviors", tc.name)
			}
			for kind, want := range tc.want {
				body := firstBehaviorBody(program, kind)
				if !strings.Contains(body, want) {
					t.Fatalf("%s %s body = %q, want resolved callback body containing %q", tc.name, kind, body, want)
				}
				trimmed := strings.TrimSpace(body)
				if trimmed == "entered" || trimmed == "allowed" || trimmed == "Entered" || trimmed == "Allowed" ||
					strings.Contains(trimmed, "DoorHsm::entered") || strings.Contains(trimmed, "DoorHsm::allowed") {
					t.Fatalf("%s %s kept callback reference instead of resolved body: %q", tc.name, kind, body)
				}
			}
		})
	}
}

func TestBehaviorAdapterCannotTargetJSONIR(t *testing.T) {
	compiler := NewCompiler()
	adapter := &matrixAdapter{}
	compiler.RegisterAdapter(adapter)

	_, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageJSONIR,
		Adapter: adapter.Name(),
	})
	if err == nil || !strings.Contains(err.Error(), `cannot target transport language "json-ir"`) {
		t.Fatalf("error = %v, want transport target rejection", err)
	}
}

func TestBuildAdapterProtocolRejectsJSONIRTarget(t *testing.T) {
	_, _, err := NewCompiler().BuildAdapterProtocol(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From: LanguageGo,
		To:   LanguageJSONIR,
	})
	if err == nil || !strings.Contains(err.Error(), `adapter protocol cannot target transport language "json-ir"`) {
		t.Fatalf("error = %v, want transport target rejection", err)
	}
}

func TestGoBackendUsesAdapterCodeForForeignSourceTargetGoBehavior(t *testing.T) {
	program := &Program{
		PackageName: "sample",
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageTS,
			TargetLanguage: LanguageGo,
			Signature:      "(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void",
			Code:           `func(ctx context.Context, instance hsm.Instance, event hsm.Event) { _ = adapterFormat("entry") }`,
		}},
	}

	output, err := NewGoBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if strings.Contains(text, "ctx: hsm.Context") {
		t.Fatalf("Go backend reused foreign signature:\n%s", text)
	}
	if !strings.Contains(text, "var Entry1 = func(ctx context.Context, instance hsm.Instance, event hsm.Event)") {
		t.Fatalf("Go backend did not emit adapter code as function literal:\n%s", text)
	}
}

func TestGoBackendWrapsAdapterBodyForForeignSourceTargetGoBehavior(t *testing.T) {
	program := &Program{
		PackageName: "sample",
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageTS,
			TargetLanguage: LanguageGo,
			Body:           `_ = adapterFormat("entry")`,
		}},
	}
	AssignTargetABI(program, LanguageGo)

	output, err := NewGoBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if strings.Contains(text, "var Entry1 =") {
		t.Fatalf("Go backend emitted body-only adapter patch as a variable:\n%s", text)
	}
	if !strings.Contains(text, "func Entry1(ctx context.Context, instance hsm.Instance, event hsm.Event)") {
		t.Fatalf("Go backend did not wrap adapter body in ABI function:\n%s", text)
	}
	if !strings.Contains(text, `_ = adapterFormat("entry")`) {
		t.Fatalf("Go backend output missing adapter body:\n%s", text)
	}
}

func TestCompilerUsesAdapterFullCodeForSourceGoTargetGoBehavior(t *testing.T) {
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Behaviors: []Behavior{{
			ID: "entry_1",
			Code: `func(ctx context.Context, instance hsm.Instance, event hsm.Event) {
	instance.Set("adapter", true)
}`,
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	output, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageGo,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, "var Entry1 = func(ctx context.Context, instance hsm.Instance, event hsm.Event)") {
		t.Fatalf("Go output did not emit adapter full-code function literal:\n%s", text)
	}
	if !strings.Contains(text, `instance.Set("adapter", true)`) {
		t.Fatalf("Go output missing adapter code body:\n%s", text)
	}
	if strings.Contains(text, `sm.log = append(sm.log, record("closed"))`) {
		t.Fatalf("Go output kept original source behavior body instead of adapter code:\n%s", text)
	}
}

func TestCompilerWrapsAdapterBodyWithTargetABIForSourceGoTargetGoBehavior(t *testing.T) {
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:   "entry_1",
			Body: `instance.Set("adapter", true)`,
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	output, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageGo,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, "func Entry1(ctx context.Context, instance hsm.Instance, event hsm.Event)") {
		t.Fatalf("Go output did not wrap adapter body with target ABI:\n%s", text)
	}
	if strings.Contains(text, "func Entry1(ctx context.Context, sm *DoorHSM") {
		t.Fatalf("Go output reused source behavior signature for adapter body:\n%s", text)
	}
	if !strings.Contains(text, `instance.Set("adapter", true)`) {
		t.Fatalf("Go output missing adapter body:\n%s", text)
	}
	if strings.Contains(text, `sm.log = append(sm.log, record("closed"))`) {
		t.Fatalf("Go output kept original source behavior body instead of adapter body:\n%s", text)
	}
}

func TestBackendsEmitAdapterCodeForForeignSourceTargetBehaviors(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name       string
		target     Language
		behaviorID string
		code       string
		emit       func(*Program) ([]byte, error)
		want       string
		forbid     string
	}{
		{
			name:       "csharp",
			target:     LanguageCSharp,
			behaviorID: "entry_1",
			code:       `(ctx, instance, @event) => System.GC.KeepAlive(instance)`,
			emit: func(program *Program) ([]byte, error) {
				return NewCSharpBackend().Emit(ctx, program)
			},
			want:   `private static readonly Operation<Instance> Entry1 = (ctx, instance, @event) => System.GC.KeepAlive(instance);`,
			forbid: `private static void Entry1(Context ctx, Instance instance, Event @event)`,
		},
		{
			name:       "cpp",
			target:     LanguageCPP,
			behaviorID: "entry_1",
			code:       `[](auto& signal, auto& instance, const auto& event) { instance.ready = true; }`,
			emit: func(program *Program) ([]byte, error) {
				return NewCPPBackend().Emit(ctx, program)
			},
			want:   `static constexpr auto entry_1 = [](auto& signal, auto& instance, const auto& event) { instance.ready = true; };`,
			forbid: `static constexpr auto entry_1 = [](auto& signal, auto& instance, const auto& event) -> void {`,
		},
		{
			name:       "javascript",
			target:     LanguageJS,
			behaviorID: "entry_1",
			code:       `(ctx, instance, event) => { instance.ready = true; }`,
			emit: func(program *Program) ([]byte, error) {
				return NewJavaScriptBackend().Emit(ctx, program)
			},
			want:   `const entry1 = (ctx, instance, event) => { instance.ready = true; };`,
			forbid: `throw new Error("behavior entry_1 requires translation from typescript to JavaScript")`,
		},
		{
			name:       "dart",
			target:     LanguageDart,
			behaviorID: "entry_1",
			code:       `(Context ctx, Instance instance, Event event) { instance.ready = true; }`,
			emit: func(program *Program) ([]byte, error) {
				return NewDartBackend().Emit(ctx, program)
			},
			want:   `final entry1 = (Context ctx, Instance instance, Event event) { instance.ready = true; };`,
			forbid: `void entry1(Context ctx, Instance instance, Event event)`,
		},
		{
			name:       "java",
			target:     LanguageJava,
			behaviorID: "entry_1",
			code: `private static void Entry1(Context ctx, Instance instance, Event event) {
	instance.ready = true;
}`,
			emit: func(program *Program) ([]byte, error) {
				return NewJavaBackend().Emit(ctx, program)
			},
			want:   `private static void Entry1(Context ctx, Instance instance, Event event)`,
			forbid: `Original typescript behavior entry_1 preserved for manual porting`,
		},
		{
			name:       "typescript",
			target:     LanguageTS,
			behaviorID: "entry_1",
			code:       `(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void => { instance.ready = true; }`,
			emit: func(program *Program) ([]byte, error) {
				return NewTypeScriptBackend().Emit(ctx, program)
			},
			want:   `const entry1 = (ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void => { instance.ready = true; };`,
			forbid: `function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event)`,
		},
		{
			name:       "python",
			target:     LanguagePython,
			behaviorID: "entry_1",
			code:       `lambda ctx, instance, event: setattr(instance, "ready", True)`,
			emit: func(program *Program) ([]byte, error) {
				return NewPythonBackend().Emit(ctx, program)
			},
			want:   `entry_1 = lambda ctx, instance, event: setattr(instance, "ready", True)`,
			forbid: `raise RuntimeError("behavior entry_1 requires translation from typescript to Python")`,
		},
		{
			name:       "rust",
			target:     LanguageRust,
			behaviorID: "entry_1",
			code: `fn entry_1(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) -> Pin<Box<dyn Future<Output = ()> + Send>> {
	let _ = (ctx, instance, event);
	Box::pin(async {})
}`,
			emit: func(program *Program) ([]byte, error) {
				return NewRustBackend().Emit(ctx, program)
			},
			want:   `fn entry_1(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) -> Pin<Box<dyn Future<Output = ()> + Send>>`,
			forbid: `Original typescript behavior entry_1 preserved for manual porting`,
		},
		{
			name:       "zig",
			target:     LanguageZig,
			behaviorID: "entry_1",
			code: `fn entry_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void {
    _ = ctx;
    inst.ready = true;
    _ = event;
}`,
			emit: func(program *Program) ([]byte, error) {
				return NewZigBackend().Emit(ctx, program)
			},
			want:   `fn entry_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void`,
			forbid: `Original typescript behavior entry_1 preserved for manual porting`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := &Program{
				Behaviors: []Behavior{{
					ID:             tc.behaviorID,
					OwnerID:        "state_1",
					Kind:           BehaviorEntry,
					SourceLanguage: LanguageTS,
					TargetLanguage: tc.target,
					Code:           tc.code,
				}},
			}
			AssignTargetABI(program, tc.target)

			output, err := tc.emit(program)
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			if !strings.Contains(text, tc.want) {
				t.Fatalf("%s backend missing adapter code %q:\n%s", tc.target, tc.want, text)
			}
			if strings.Contains(text, tc.forbid) {
				t.Fatalf("%s backend ignored adapter code:\n%s", tc.target, text)
			}
		})
	}
}

type matrixSource struct {
	name     string
	language Language
	path     string
	data     []byte
}

func firstBehaviorBody(program *Program, kind BehaviorKind) string {
	for _, behavior := range program.Behaviors {
		if behavior.Kind == kind {
			return behavior.Body
		}
	}
	return ""
}

func assertNoAdapterPreservesForeignBehaviorsAsComments(t *testing.T, output string, program *Program, target Language) {
	t.Helper()
	if target == LanguageJSONIR || program == nil || len(program.Behaviors) == 0 {
		return
	}
	for _, behavior := range program.Behaviors {
		if behavior.SourceLanguage == target || behavior.TargetLanguage == target {
			continue
		}
		needle := "Original " + string(behavior.SourceLanguage) + " behavior " + behavior.ID + " preserved for manual porting:"
		if !strings.Contains(output, needle) {
			t.Fatalf("no-adapter %s target did not preserve foreign behavior %s as a comment, missing %q:\n%s", target, behavior.ID, needle, output)
		}
		assertNoAdapterPreservesRegionImportContext(t, output, behavior.SourceLanguage, behavior.Imports, "behavior", behavior.ID, target)
	}
}

func assertNoAdapterPreservesForeignGlobalsAsComments(t *testing.T, output string, program *Program, target Language) {
	t.Helper()
	if target == LanguageJSONIR || program == nil || len(program.Globals) == 0 {
		return
	}
	for _, global := range program.Globals {
		if global.Language == "" || global.Language == target || strings.TrimSpace(global.Code) == "" {
			continue
		}
		needle := "Original " + string(global.Language) + " global " + global.ID + " preserved for manual porting:"
		if !strings.Contains(output, needle) {
			t.Fatalf("no-adapter %s target did not preserve foreign global %s as a comment, missing %q:\n%s", target, global.ID, needle, output)
		}
		assertNoAdapterPreservesRegionImportContext(t, output, global.Language, global.Imports, "global", global.ID, target)
	}
}

func assertNoAdapterPreservesRegionImportContext(t *testing.T, output string, source Language, imports []Import, kind string, id string, target Language) {
	t.Helper()
	if len(imports) == 0 {
		return
	}
	if !strings.Contains(output, "Source imports:") {
		t.Fatalf("no-adapter %s target did not preserve %s %s source import context header:\n%s", target, kind, id, output)
	}
	for _, imp := range imports {
		line := sourceImportText(source, imp)
		if line == "" {
			continue
		}
		if !strings.Contains(output, line) {
			t.Fatalf("no-adapter %s target did not preserve %s %s source import %q:\n%s", target, kind, id, line, output)
		}
	}
}

func implementationMatrixSources() []matrixSource {
	return []matrixSource{
		{name: "csharp", language: LanguageCSharp, path: "door.cs", data: []byte(csharpDoorSource)},
		{name: "cpp", language: LanguageCPP, path: "door.cpp", data: []byte(cppDoorSource)},
		{name: "dart", language: LanguageDart, path: "door.dart", data: []byte(dartDoorSource)},
		{name: "go", language: LanguageGo, path: "door.go", data: []byte(goDoorSource)},
		{name: "java", language: LanguageJava, path: "DoorHsm.java", data: []byte(javaDoorSource)},
		{name: "javascript", language: LanguageJS, path: "door.js", data: []byte(jsDoorSource)},
		{name: "python", language: LanguagePython, path: "door.py", data: []byte(pythonDoorSource)},
		{name: "rust", language: LanguageRust, path: "door.rs", data: []byte(rustDoorSource)},
		{name: "typescript", language: LanguageTS, path: "door.ts", data: []byte(tsDoorSource)},
		{name: "zig", language: LanguageZig, path: "door.zig", data: []byte(zigDoorSource)},
	}
}

func matrixSources(t *testing.T, compiler *Compiler) []matrixSource {
	t.Helper()
	sources := implementationMatrixSources()
	ir, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageJSONIR,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	return append(sources, matrixSource{name: "json-ir", language: LanguageJSONIR, path: "door.hsm.json", data: ir})
}

const namedCSharpBehaviorMatrixSource = `using Stateforward.Hsm;

public static class Sample
{
    private static void Entered(Context ctx, Instance instance, Event @event)
    {
        instance.Set("entered", true);
    }

    private static bool Allowed(Context ctx, Instance instance, Event @event)
    {
        return @event.Name == "open";
    }

    public static readonly Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed",
            Hsm.Entry<Instance>(Entered),
            Hsm.Transition(Hsm.On("open"), Hsm.Guard<Instance>(Allowed), Hsm.Target("../open"))
        ),
        Hsm.State("open")
    );
}
`

const namedGoBehaviorMatrixSource = `package sample

import (
	"context"

	hsm "github.com/stateforward/hsm.go"
)

func entered(ctx context.Context, instance hsm.Instance, event hsm.Event) {
	instance.Set("entered", true)
}

func allowed(ctx context.Context, instance hsm.Instance, event hsm.Event) bool {
	return event.Name == "open"
}

var DoorModel = hsm.Define(
	"Door",
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed",
		hsm.Entry(entered),
		hsm.Transition(hsm.On("open"), hsm.Guard(allowed), hsm.Target("../open")),
	),
	hsm.State("open"),
)
`

const goOperationReferenceMatrixSource = `package sample

import hsm "github.com/stateforward/hsm.go"

type DoorHSM struct{ hsm.HSM }

func approve(sm *DoorHSM) bool {
	return true
}

var DoorModel = hsm.Define(
	"Door",
	hsm.Operation("approve", approve),
	hsm.Initial(hsm.Target("closed"), hsm.Effect("approve")),
	hsm.State("closed",
		hsm.Entry("approve"),
		hsm.Exit("approve"),
		hsm.Activity("approve"),
		hsm.Transition(hsm.On("open"), hsm.Guard("approve"), hsm.Target("../open"), hsm.Effect("approve")),
	),
	hsm.State("open"),
)
`

const namedJavaBehaviorMatrixSource = `import com.stateforward.hsm.*;

public final class DoorHsm {
  private static void entered(Context ctx, Instance instance, Event event) {
    instance.set("entered", true);
  }

  private static boolean allowed(Context ctx, Instance instance, Event event) {
    return event.name().equals("open");
  }

  public static final Model DoorModel = Hsm.Define(
    "Door",
    Hsm.Initial(Hsm.Target("closed")),
    Hsm.State(
      "closed",
      Hsm.Entry(DoorHsm::entered),
      Hsm.Transition(Hsm.On("open"), Hsm.Guard(DoorHsm::allowed), Hsm.Target("../open"))
    ),
    Hsm.State("open")
  );
}
`

const namedTypeScriptBehaviorMatrixSource = `import * as hsm from "@stateforward/hsm.ts";

function entered(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("entered", true);
}

function allowed(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): boolean {
  return event.name === "open";
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry(entered),
    hsm.Transition(hsm.On("open"), hsm.Guard(allowed), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`

type matrixAdapter struct{}

func (adapter *matrixAdapter) Name() string { return "matrix" }

func (adapter *matrixAdapter) Translate(_ context.Context, request AdapterRequest) (*BehaviorPatch, error) {
	patch := &BehaviorPatch{Globals: matrixAdapterGlobals(request.TargetLanguage)}
	for _, behavior := range request.Behaviors {
		patch.Behaviors = append(patch.Behaviors, matrixAdapterBehavior(request.TargetLanguage, request.SourceLanguage, behavior))
	}
	return patch, nil
}

func matrixAdapterGlobals(target Language) []CodeBlock {
	switch target {
	case LanguageCSharp:
		return []CodeBlock{{
			ID:      "adapter_global_format",
			Code:    `private static string AdapterFormat(string value) => AdapterGlobalFormat(value);`,
			Imports: []Import{{Path: "System.Text"}},
		}}
	case LanguageCPP:
		return []CodeBlock{{
			ID:      "adapter_global_format",
			Code:    `static std::string adapter_format(std::string value) { return adapter_global_format(value); }`,
			Imports: []Import{{Path: "global_format.hpp"}},
		}}
	case LanguageDart:
		return []CodeBlock{{
			ID:   "adapter_global_format",
			Code: `String adapterFormat(String value) => adapterGlobalFormat(value);`,
			Imports: []Import{{
				Path:       "global_format.dart",
				Specifiers: []ImportSpecifier{{Name: "adapterGlobalFormat"}},
			}},
		}}
	case LanguageGo:
		return []CodeBlock{{
			ID:      "adapter_global_format",
			Code:    `func adapterFormat(value string) string { return strings.ToUpper(value) }`,
			Imports: []Import{{Path: "strings"}},
		}}
	case LanguageJava:
		return []CodeBlock{{
			ID:      "adapter_global_format",
			Code:    `private static String AdapterFormat(String value) { return AdapterGlobalFormat.format(value); }`,
			Imports: []Import{{Path: "com.example.AdapterGlobalFormat"}},
		}}
	case LanguageJS:
		return []CodeBlock{{
			ID:   "adapter_global_format",
			Code: `const adapterFormat = (value) => adapterGlobalFormat(value);`,
			Imports: []Import{{
				Path:       "./global-format",
				Specifiers: []ImportSpecifier{{Name: "adapterGlobalFormat"}},
			}},
		}}
	case LanguagePython:
		return []CodeBlock{{
			ID:   "adapter_global_format",
			Code: "def adapter_format(value: str) -> str:\n\treturn adapter_global_format(value)",
			Imports: []Import{{
				Path:       "global_format",
				Specifiers: []ImportSpecifier{{Name: "adapter_global_format"}},
			}},
		}}
	case LanguageRust:
		return []CodeBlock{{
			ID:   "adapter_global_format",
			Code: `fn adapter_format(value: &str) -> String { adapter_global_format(value) }`,
			Imports: []Import{{
				Path:       "crate::global_format",
				Specifiers: []ImportSpecifier{{Name: "adapter_global_format"}},
			}},
		}}
	case LanguageZig:
		return []CodeBlock{{
			ID:   "adapter_global_format",
			Code: `fn adapter_format(value: []const u8) []const u8 { return adapter_global_format.format(value); }`,
			Imports: []Import{{
				Path:  "global_format",
				Alias: "adapter_global_format",
			}},
		}}
	case LanguageTS:
		return []CodeBlock{{
			ID:   "adapter_global_format",
			Code: `const adapterFormat = (value: string): string => adapterGlobalFormat(value);`,
			Imports: []Import{{
				Path:       "./global-format",
				Specifiers: []ImportSpecifier{{Name: "adapterGlobalFormat"}},
			}},
		}}
	default:
		return nil
	}
}

func matrixAdapterBehavior(target Language, source Language, original Behavior) Behavior {
	behavior := Behavior{ID: original.ID}
	switch target {
	case LanguageCSharp:
		behavior.Body = `System.GC.KeepAlive(AdapterFormat("entry"));`
		behavior.Imports = []Import{{Path: "System.Collections.Generic"}}
	case LanguageCPP:
		behavior.Body = `(void)adapter_behavior_format(adapter_format("entry"));`
		behavior.Imports = []Import{{Path: "behavior_format.hpp"}}
	case LanguageDart:
		behavior.Body = `adapterBehaviorFormat(adapterFormat("entry"));`
		behavior.Imports = []Import{{
			Path:       "behavior_format.dart",
			Specifiers: []ImportSpecifier{{Name: "adapterBehaviorFormat"}},
		}}
	case LanguageGo:
		behavior.Imports = []Import{{Path: "bytes"}}
		body := matrixGoBehaviorBody(original)
		if source == LanguageGo {
			behavior.Body = body
		} else {
			behavior.Code = fmt.Sprintf("func%s {\n%s\n}", goABISuffix(original), indentTS(body))
		}
	case LanguageJava:
		behavior.Body = `AdapterBehaviorFormat.record(AdapterFormat("entry"));`
		behavior.Imports = []Import{{Path: "com.example.AdapterBehaviorFormat"}}
	case LanguageJS:
		behavior.Body = `void adapterBehaviorFormat(adapterFormat("entry"));`
		behavior.Imports = []Import{{
			Path:       "./behavior-format",
			Specifiers: []ImportSpecifier{{Name: "adapterBehaviorFormat"}},
		}}
	case LanguagePython:
		behavior.Body = `adapter_behavior_format(adapter_format("entry"))`
		behavior.Imports = []Import{{
			Path:       "behavior_format",
			Specifiers: []ImportSpecifier{{Name: "adapter_behavior_format"}},
		}}
	case LanguageRust:
		behavior.Body = `adapter_behavior_format(&adapter_format("entry"));`
		behavior.Imports = []Import{{
			Path:       "crate::behavior_format",
			Specifiers: []ImportSpecifier{{Name: "adapter_behavior_format"}},
		}}
	case LanguageZig:
		behavior.Body = `_ = adapter_behavior_format.format(adapter_format("entry"));`
		behavior.Imports = []Import{{
			Path:  "behavior_format",
			Alias: "adapter_behavior_format",
		}}
	case LanguageTS:
		behavior.Body = `void adapterBehaviorFormat(adapterFormat("entry"));`
		behavior.Imports = []Import{{
			Path:       "./behavior-format",
			Specifiers: []ImportSpecifier{{Name: "adapterBehaviorFormat"}},
		}}
	}
	return behavior
}

func matrixGoBehaviorBody(behavior Behavior) string {
	body := `var _ = bytes.Buffer{}
_ = adapterFormat("entry")`
	switch {
	case behavior.Kind == BehaviorGuard:
		return body + "\nreturn true"
	case behavior.Kind == BehaviorTrigger && behavior.TriggerKind == TriggerAt:
		return body + "\nreturn time.Time{}"
	case behavior.Kind == BehaviorTrigger && behavior.TriggerKind == TriggerWhen:
		return body + "\nreturn nil"
	case behavior.Kind == BehaviorTrigger:
		return body + "\nreturn 0"
	default:
		return body
	}
}

func assertMatrixAdapterOutput(t *testing.T, target Language, output string) {
	t.Helper()
	var needles []string
	switch target {
	case LanguageCSharp:
		needles = []string{"using System.Text;", "using System.Collections.Generic;", "private static string AdapterFormat", `System.GC.KeepAlive(AdapterFormat("entry"))`}
	case LanguageCPP:
		needles = []string{`#include "global_format.hpp"`, `#include "behavior_format.hpp"`, "static std::string adapter_format", `adapter_behavior_format(adapter_format("entry"))`}
	case LanguageDart:
		needles = []string{`import "global_format.dart" show adapterGlobalFormat;`, `import "behavior_format.dart" show adapterBehaviorFormat;`, "String adapterFormat", `adapterBehaviorFormat(adapterFormat("entry"))`}
	case LanguageGo:
		needles = []string{`"bytes"`, `"strings"`, "func adapterFormat", "bytes.Buffer", `adapterFormat("entry")`}
	case LanguageJava:
		needles = []string{"import com.example.AdapterGlobalFormat;", "import com.example.AdapterBehaviorFormat;", "private static String AdapterFormat", `AdapterBehaviorFormat.record(AdapterFormat("entry"))`}
	case LanguageJS:
		needles = []string{`import { adapterGlobalFormat } from "./global-format";`, `import { adapterBehaviorFormat } from "./behavior-format";`, "const adapterFormat", `adapterBehaviorFormat(adapterFormat("entry"))`}
	case LanguagePython:
		needles = []string{"from global_format import adapter_global_format", "from behavior_format import adapter_behavior_format", "def adapter_format", `adapter_behavior_format(adapter_format("entry"))`}
	case LanguageRust:
		needles = []string{"use crate::global_format::{adapter_global_format};", "use crate::behavior_format::{adapter_behavior_format};", "fn adapter_format", `adapter_behavior_format(&adapter_format("entry"))`}
	case LanguageZig:
		needles = []string{`const adapter_global_format = @import("global_format");`, `const adapter_behavior_format = @import("behavior_format");`, "fn adapter_format", `adapter_behavior_format.format(adapter_format("entry"))`}
	case LanguageTS:
		needles = []string{`import { adapterGlobalFormat } from "./global-format";`, `import { adapterBehaviorFormat } from "./behavior-format";`, "const adapterFormat", `adapterBehaviorFormat(adapterFormat("entry"))`}
	default:
		t.Fatalf("unexpected target %q", target)
	}
	for _, needle := range needles {
		if !strings.Contains(output, needle) {
			t.Fatalf("%s output missing %q:\n%s", target, needle, output)
		}
	}
}
