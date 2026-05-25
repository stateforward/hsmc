package hsmc

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestCompilerAppliesAdapterPatchEndToEnd(t *testing.T) {
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Globals: []CodeBlock{{
			ID:      "adapter_global_format",
			Code:    "const format = (value: string): string => value;",
			Imports: []Import{{Path: "./format"}},
		}},
		Behaviors: []Behavior{{
			ID:   "entry_1",
			Body: `instance["log"] = [format("closed")];`,
			Imports: []Import{{
				Path:       "./behavior-format",
				Specifiers: []ImportSpecifier{{Name: "track"}},
			}},
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	output, program, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`import "./format";`,
		`import { track } from "./behavior-format";`,
		`const format = (value: string): string => value;`,
		`instance["log"] = [format("closed")];`,
		"hsm.Entry(entry1)",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("output missing %q:\n%s", needle, text)
		}
	}
	var helperGlobal bool
	for _, global := range program.Globals {
		if global.ID == "adapter_global_format" && global.Language == LanguageTS {
			helperGlobal = true
		}
	}
	if !helperGlobal {
		t.Fatalf("program missing adapter helper global: %#v", program.Globals)
	}
	if len(program.Models) != 1 {
		t.Fatalf("models = %#v", program.Models)
	}
	model := program.Models[0]
	if model.Name != "Door" || model.Initializer == nil || model.Initializer.Target != "closed" {
		t.Fatalf("adapter patch changed model identity or initial target: %#v", model)
	}
	if len(model.States) != 2 || model.States[0].Name != "closed" || model.States[1].Name != "open" {
		t.Fatalf("adapter patch changed state topology: %#v", model.States)
	}
	closed := model.States[0]
	if len(closed.Behaviors) != 1 || closed.Behaviors[0].ID != "entry_1" || closed.Behaviors[0].Kind != BehaviorEntry {
		t.Fatalf("adapter patch changed state behavior refs: %#v", closed.Behaviors)
	}
	if len(closed.Transitions) != 1 {
		t.Fatalf("adapter patch changed transition topology: %#v", closed.Transitions)
	}
	transition := closed.Transitions[0]
	if transition.Target != "../open" || transition.Trigger == nil || transition.Trigger.Value != "open" || transition.Guard == nil || len(transition.Effects) != 1 {
		t.Fatalf("adapter patch changed transition structure: %#v", transition)
	}
}

func TestCompilerRejectsAdapterFullCodeBehaviorNameChangesAcrossTargets(t *testing.T) {
	cases := []struct {
		target Language
		code   string
		want   string
	}{
		{
			target: LanguageCSharp,
			code: `private static void RenamedEntry(Context ctx, Instance instance, Event @event)
{
    System.GC.KeepAlive("entry");
}`,
			want: `declares "RenamedEntry", want compiler-owned name "Entry1"`,
		},
		{
			target: LanguageCPP,
			code:   `static constexpr auto renamed_entry = [](auto& signal, auto& instance, const auto& event) -> void { adapter_format("entry"); };`,
			want:   `declares "renamed_entry", want compiler-owned name "entry_1"`,
		},
		{
			target: LanguageDart,
			code: `FutureOr<void> renamedEntry(Context ctx, Instance instance, Event event) {
  adapterFormat("entry");
}`,
			want: `declares "renamedEntry", want compiler-owned name "entry1"`,
		},
		{
			target: LanguageGo,
			code: `func RenamedEntry(ctx context.Context, instance hsm.Instance, event hsm.Event) {
	instance.Set("ready", true)
}`,
			want: `declares "RenamedEntry", want compiler-owned name "Entry1"`,
		},
		{
			target: LanguageJava,
			code: `private static void RenamedEntry(Context ctx, Instance instance, Event event) {
    AdapterFormat.record("entry");
}`,
			want: `declares "RenamedEntry", want compiler-owned name "Entry1"`,
		},
		{
			target: LanguageJS,
			code: `const renamedEntry = (ctx, instance, event) => {
	instance.ready = true;
};`,
			want: `declares "renamedEntry", want compiler-owned name "entry1"`,
		},
		{
			target: LanguagePython,
			code: `def renamed_entry(ctx, instance, event):
	instance.ready = True`,
			want: `declares "renamed_entry", want compiler-owned name "entry_1"`,
		},
		{
			target: LanguageRust,
			code: `fn renamed_entry(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) -> Pin<Box<dyn Future<Output = ()> + Send>> {
	let _ = (ctx, instance, event);
	Box::pin(async {})
}`,
			want: `declares "renamed_entry", want compiler-owned name "entry_1"`,
		},
		{
			target: LanguageTS,
			code: `const renamedEntry: hsm.Operation<InstanceType<typeof hsm.Instance>> = (ctx, instance, event) => {
	instance.ready = true;
};`,
			want: `declares "renamedEntry", want compiler-owned name "entry1"`,
		},
		{
			target: LanguageZig,
			code: `fn renamed_entry(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void {
    _ = ctx;
    _ = inst;
    _ = event;
}`,
			want: `declares "renamed_entry", want compiler-owned name "entry_1"`,
		},
	}
	for _, tc := range cases {
		t.Run(string(tc.target), func(t *testing.T) {
			adapter := &patchingAdapter{patch: &BehaviorPatch{
				Behaviors: []Behavior{{
					ID:   "entry_1",
					Code: tc.code,
				}},
			}}
			compiler := NewCompiler()
			compiler.RegisterAdapter(adapter)

			_, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
				From:    LanguageGo,
				To:      tc.target,
				Adapter: adapter.Name(),
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestCompilerAcceptsAdapterFullCodeBehaviorDeclarationsAcrossTargets(t *testing.T) {
	cases := []struct {
		target Language
		code   string
		want   string
	}{
		{
			target: LanguageCSharp,
			code: `private static void Entry1(Context ctx, Instance instance, Event @event)
{
    System.GC.KeepAlive("entry");
}`,
			want: `private static void Entry1(Context ctx, Instance instance, Event @event)`,
		},
		{
			target: LanguageCPP,
			code:   `static constexpr auto entry_1 = [](auto& signal, auto& instance, const auto& event) -> void { adapter_format("entry"); };`,
			want:   `static constexpr auto entry_1 = [](auto& signal, auto& instance, const auto& event) -> void { adapter_format("entry"); };`,
		},
		{
			target: LanguageDart,
			code: `FutureOr<void> entry1(Context ctx, Instance instance, Event event) {
  adapterFormat("entry");
}`,
			want: `FutureOr<void> entry1(Context ctx, Instance instance, Event event)`,
		},
		{
			target: LanguageGo,
			code: `func Entry1(ctx context.Context, instance hsm.Instance, event hsm.Event) {
	instance.Set("ready", true)
}`,
			want: `func Entry1(ctx context.Context, instance hsm.Instance, event hsm.Event)`,
		},
		{
			target: LanguageJava,
			code: `private static void Entry1(Context ctx, Instance instance, Event event) {
    AdapterFormat.record("entry");
}`,
			want: `private static void Entry1(Context ctx, Instance instance, Event event)`,
		},
		{
			target: LanguageJS,
			code: `function entry1(ctx, instance, event) {
	instance.ready = true;
}`,
			want: `function entry1(ctx, instance, event)`,
		},
		{
			target: LanguagePython,
			code: `async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:
	instance.ready = True`,
			want: `async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:`,
		},
		{
			target: LanguageRust,
			code: `fn entry_1(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) -> Pin<Box<dyn Future<Output = ()> + Send>> {
	let _ = (ctx, instance, event);
	Box::pin(async {})
}`,
			want: `fn entry_1(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) -> Pin<Box<dyn Future<Output = ()> + Send>>`,
		},
		{
			target: LanguageTS,
			code: `function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
	instance.ready = true;
}`,
			want: `function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void`,
		},
		{
			target: LanguageZig,
			code: `fn entry_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void {
    _ = ctx;
    _ = inst;
    _ = event;
}`,
			want: `fn entry_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void`,
		},
	}
	for _, tc := range cases {
		t.Run(string(tc.target), func(t *testing.T) {
			compiler := NewCompiler()
			_, baseline, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
				From:    LanguageGo,
				To:      tc.target,
				Adapter: "none",
			})
			if err != nil {
				t.Fatal(err)
			}
			baselineModels := BuildModelViews(baseline)

			adapter := &patchingAdapter{patch: &BehaviorPatch{
				Behaviors: []Behavior{{
					ID:   "entry_1",
					Code: tc.code,
				}},
			}}
			compiler.RegisterAdapter(adapter)

			output, program, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
				From:    LanguageGo,
				To:      tc.target,
				Adapter: adapter.Name(),
			})
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			if !strings.Contains(text, tc.want) {
				t.Fatalf("%s output missing accepted full-code declaration %q:\n%s", tc.target, tc.want, text)
			}
			if strings.Contains(text, "Original go behavior entry_1 preserved for manual porting") {
				t.Fatalf("%s output commented accepted adapter full-code declaration:\n%s", tc.target, text)
			}
			if got := BuildModelViews(program); !reflect.DeepEqual(got, baselineModels) {
				t.Fatalf("%s adapter full-code declaration changed compiler-owned model:\ngot:  %#v\nwant: %#v", tc.target, got, baselineModels)
			}
		})
	}
}

func TestCompilerRejectsAdapterFullCodeBehaviorExtraTopLevelCodeAcrossTargets(t *testing.T) {
	cases := []struct {
		target Language
		code   string
	}{
		{
			target: LanguageCSharp,
			code: `private static void Entry1(Context ctx, Instance instance, Event @event)
{
    System.GC.KeepAlive("entry");
}
private static void DoorModel() {}`,
		},
		{
			target: LanguageCPP,
			code:   `static constexpr auto entry_1 = [](auto& signal, auto& instance, const auto& event) -> void { adapter_format("entry"); }; static constexpr auto door_model = 0;`,
		},
		{
			target: LanguageDart,
			code: `FutureOr<void> entry1(Context ctx, Instance instance, Event event) {
  adapterFormat("entry");
}
final doorModel = null;`,
		},
		{
			target: LanguageGo,
			code: `func Entry1(ctx context.Context, instance hsm.Instance, event hsm.Event) {
	instance.Set("ready", true)
}
var DoorModel = nil`,
		},
		{
			target: LanguageJava,
			code: `private static void Entry1(Context ctx, Instance instance, Event event) {
    AdapterFormat.record("entry");
}
private static void DoorModel() {}`,
		},
		{
			target: LanguageJS,
			code: `function entry1(ctx, instance, event) {
	instance.ready = true;
}
const DoorModel = null;`,
		},
		{
			target: LanguagePython,
			code: `async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:
	instance.ready = True
def door_model():
	return None`,
		},
		{
			target: LanguageRust,
			code: `fn entry_1(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) -> Pin<Box<dyn Future<Output = ()> + Send>> {
	let _ = (ctx, instance, event);
	Box::pin(async {})
}
fn door_model() {}`,
		},
		{
			target: LanguageTS,
			code: `function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
	instance.ready = true;
}
const DoorModel = null;`,
		},
		{
			target: LanguageZig,
			code: `fn entry_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void {
    _ = ctx;
    _ = inst;
    _ = event;
}
const door_model = 0;`,
		},
	}
	for _, tc := range cases {
		t.Run(string(tc.target), func(t *testing.T) {
			adapter := &patchingAdapter{patch: &BehaviorPatch{
				Behaviors: []Behavior{{
					ID:   "entry_1",
					Code: tc.code,
				}},
			}}
			compiler := NewCompiler()
			compiler.RegisterAdapter(adapter)

			_, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
				From:    LanguageGo,
				To:      tc.target,
				Adapter: adapter.Name(),
			})
			if err == nil || !strings.Contains(err.Error(), "contains extra top-level code after the compiler-owned callable") {
				t.Fatalf("error = %v, want extra top-level code rejection", err)
			}
		})
	}
}

func TestCompilerRejectsAdapterPythonExpressionFullCodeExtraTopLevelCode(t *testing.T) {
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:   "entry_1",
			Code: `lambda ctx, instance, event: setattr(instance, "ready", True); door_model = None`,
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	_, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguagePython,
		Adapter: adapter.Name(),
	})
	if err == nil || !strings.Contains(err.Error(), "contains extra top-level code after the compiler-owned callable") {
		t.Fatalf("error = %v, want Python expression extra top-level code rejection", err)
	}
}

func TestCompilerRejectsAdapterPythonUnsupportedOneLineFunctionFullCode(t *testing.T) {
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:   "entry_1",
			Code: `async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None: instance.ready = True`,
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	_, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguagePython,
		Adapter: adapter.Name(),
	})
	if err == nil || !strings.Contains(err.Error(), "must be a supported compiler-owned function declaration") {
		t.Fatalf("error = %v, want unsupported Python one-line full-code declaration rejection", err)
	}
}

func TestCompilerRejectsAdapterESExpressionFullCodeExtraTopLevelCode(t *testing.T) {
	cases := []struct {
		name   string
		target Language
		code   string
	}{
		{
			name:   "javascript arrow expression",
			target: LanguageJS,
			code: `const entry1 = (ctx, instance, event) => instance.ready = true
const DoorModel = null`,
		},
		{
			name:   "typescript arrow expression",
			target: LanguageTS,
			code: `(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void => instance.ready = true
const DoorModel = null`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &patchingAdapter{patch: &BehaviorPatch{
				Behaviors: []Behavior{{
					ID:   "entry_1",
					Code: tc.code,
				}},
			}}
			compiler := NewCompiler()
			compiler.RegisterAdapter(adapter)

			_, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
				From:    LanguageGo,
				To:      tc.target,
				Adapter: adapter.Name(),
			})
			if err == nil || !strings.Contains(err.Error(), "contains extra top-level code after the compiler-owned callable") {
				t.Fatalf("error = %v, want ES expression extra top-level code rejection", err)
			}
		})
	}
}

func TestCompilerRejectsAdapterCallableExpressionFullCodeExtraTopLevelCode(t *testing.T) {
	cases := []struct {
		name   string
		target Language
		code   string
	}{
		{
			name:   "csharp lambda expression",
			target: LanguageCSharp,
			code: `(ctx, instance, @event) => System.GC.KeepAlive("entry")
private static void DoorModel() {}`,
		},
		{
			name:   "dart arrow expression",
			target: LanguageDart,
			code: `(Context ctx, Instance instance, Event event) => adapterFormat("entry")
final doorModel = null`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &patchingAdapter{patch: &BehaviorPatch{
				Behaviors: []Behavior{{
					ID:   "entry_1",
					Code: tc.code,
				}},
			}}
			compiler := NewCompiler()
			compiler.RegisterAdapter(adapter)

			_, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
				From:    LanguageGo,
				To:      tc.target,
				Adapter: adapter.Name(),
			})
			if err == nil || !strings.Contains(err.Error(), "contains extra top-level code after the compiler-owned callable") {
				t.Fatalf("error = %v, want callable expression extra top-level code rejection", err)
			}
		})
	}
}

func TestCompilerRejectsAdapterDartExternalFunctionBehaviorFullCode(t *testing.T) {
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:   "entry_1",
			Code: `external FutureOr<void> entry1(Context ctx, Instance instance, Event event);`,
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	_, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageDart,
		Adapter: adapter.Name(),
	})
	if err == nil || !strings.Contains(err.Error(), `dart behavior full code for "entry1" must be a compiler-owned callable declaration`) {
		t.Fatalf("error = %v, want Dart external function full-code rejection", err)
	}
}

func TestCompilerRejectsAdapterCPPPrototypeBehaviorFullCode(t *testing.T) {
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:   "entry_1",
			Code: `void entry_1(auto& signal, auto& instance, const auto& event);`,
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	_, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageCPP,
		Adapter: adapter.Name(),
	})
	if err == nil || !strings.Contains(err.Error(), `cpp behavior full code for "entry_1" must be a compiler-owned function definition, lambda declaration, or lambda expression`) {
		t.Fatalf("error = %v, want C++ prototype full-code rejection", err)
	}
}

func TestCompilerRejectsAdapterCSharpAndJavaPrototypeBehaviorFullCode(t *testing.T) {
	cases := []struct {
		name   string
		target Language
		code   string
		want   string
	}{
		{
			name:   "csharp extern method prototype",
			target: LanguageCSharp,
			code:   `private static extern void Entry1(Context ctx, Instance instance, Event @event);`,
			want:   `csharp behavior full code for "Entry1" must be a compiler-owned callable declaration with a body`,
		},
		{
			name:   "java abstract method prototype",
			target: LanguageJava,
			code:   `private static abstract void Entry1(Context ctx, Instance instance, Event event);`,
			want:   `java behavior code for "Entry1" must be a compiler-owned method declaration with a body`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &patchingAdapter{patch: &BehaviorPatch{
				Behaviors: []Behavior{{
					ID:   "entry_1",
					Code: tc.code,
				}},
			}}
			compiler := NewCompiler()
			compiler.RegisterAdapter(adapter)

			_, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
				From:    LanguageGo,
				To:      tc.target,
				Adapter: adapter.Name(),
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestCompilerRejectsAdapterESPrototypeBehaviorFullCode(t *testing.T) {
	cases := []struct {
		name   string
		target Language
		code   string
		want   string
	}{
		{
			name:   "javascript function prototype",
			target: LanguageJS,
			code:   `function entry1(ctx, instance, event);`,
			want:   `javascript behavior full code for "entry1" must be a compiler-owned callable declaration with a body`,
		},
		{
			name:   "typescript declare function",
			target: LanguageTS,
			code:   `declare function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void;`,
			want:   `typescript behavior full code for "entry1" must be a compiler-owned callable declaration with a body`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &patchingAdapter{patch: &BehaviorPatch{
				Behaviors: []Behavior{{
					ID:   "entry_1",
					Code: tc.code,
				}},
			}}
			compiler := NewCompiler()
			compiler.RegisterAdapter(adapter)

			_, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
				From:    LanguageGo,
				To:      tc.target,
				Adapter: adapter.Name(),
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestBackendsRejectAlreadyTargetAdapterFullCodeNonCallableExpressions(t *testing.T) {
	cases := []struct {
		name    string
		target  Language
		backend Backend
		want    string
	}{
		{
			name:    "csharp",
			target:  LanguageCSharp,
			backend: NewCSharpBackend(),
			want:    `csharp behavior full code for "Entry1" must be a compiler-owned callable expression`,
		},
		{
			name:    "dart",
			target:  LanguageDart,
			backend: NewDartBackend(),
			want:    `dart behavior full code for "entry1" must be a compiler-owned callable expression`,
		},
		{
			name:    "javascript",
			target:  LanguageJS,
			backend: NewJavaScriptBackend(),
			want:    `javascript behavior full code for "entry1" must be a compiler-owned callable expression`,
		},
		{
			name:    "python",
			target:  LanguagePython,
			backend: NewPythonBackend(),
			want:    `python behavior full code for "entry_1" must be a compiler-owned callable expression`,
		},
		{
			name:    "typescript",
			target:  LanguageTS,
			backend: NewTypeScriptBackend(),
			want:    `typescript behavior full code for "entry1" must be a compiler-owned callable expression`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := &Program{
				Behaviors: []Behavior{{
					ID:             "entry_1",
					OwnerID:        "state_1",
					Kind:           BehaviorEntry,
					SourceLanguage: tc.target,
					TargetLanguage: tc.target,
					Code:           "1",
				}},
			}
			AssignTargetABI(program, tc.target)

			_, err := tc.backend.Emit(context.Background(), program)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestCompilerRejectsAdapterNonDeclarationCodeForDeclarationOnlyTargets(t *testing.T) {
	cases := []struct {
		target Language
		code   string
		want   string
	}{
		{
			target: LanguageJava,
			code:   `(ctx, instance, event) -> AdapterFormat.record("entry")`,
			want:   `java behavior code for "Entry1" must be a compiler-owned method declaration`,
		},
		{
			target: LanguageJava,
			code:   `private static Event Entry1 = null;`,
			want:   `java behavior code for "Entry1" must be a compiler-owned method declaration`,
		},
		{
			target: LanguageRust,
			code: `|ctx, instance, event| {
	let _ = (ctx, instance, event);
}`,
			want: `rust behavior code for "entry_1" must be a compiler-owned function declaration`,
		},
		{
			target: LanguageRust,
			code:   `pub static entry_1: Option<fn(&hsm::Context, &mut HsmcInstance, &hsm::Event)> = None;`,
			want:   `rust behavior code for "entry_1" must be a compiler-owned function declaration`,
		},
		{
			target: LanguageZig,
			code: `struct {
	fn call(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void {
		_ = ctx;
		_ = inst;
		_ = event;
	}
}.call`,
			want: `zig behavior code for "entry_1" must be a compiler-owned function declaration`,
		},
		{
			target: LanguageZig,
			code:   `pub var entry_1: ?*const fn (*hsm.Context, *hsm.Instance, hsm.Event) void = null;`,
			want:   `zig behavior code for "entry_1" must be a compiler-owned function declaration`,
		},
	}
	for _, tc := range cases {
		t.Run(string(tc.target), func(t *testing.T) {
			adapter := &patchingAdapter{patch: &BehaviorPatch{
				Behaviors: []Behavior{{
					ID:   "entry_1",
					Code: tc.code,
				}},
			}}
			compiler := NewCompiler()
			compiler.RegisterAdapter(adapter)

			_, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
				From:    LanguageGo,
				To:      tc.target,
				Adapter: adapter.Name(),
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestCompilerAppliesGoAtTriggerAdapterPatchWithoutChangingModel(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:   "trigger_1",
			Body: "return deadline.Now()",
			Imports: []Import{{
				Path: "example.com/deadline",
			}},
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	output, program, err := compiler.Compile(context.Background(), SourceInput{Path: "timer.ts", Data: []byte(source)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguageGo,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`"context"`,
		`"example.com/deadline"`,
		`"time"`,
		"func Trigger1(ctx context.Context, instance hsm.Instance, event hsm.Event) time.Time {",
		"return deadline.Now()",
		"var TimerModel = hsm.Define(",
		`hsm.Initial(`,
		`hsm.Target("idle")`,
		`hsm.State(`,
		`"idle"`,
		`hsm.Transition(`,
		`hsm.At(Trigger1)`,
		`hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go at trigger adapter output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return new Date();") {
		t.Fatalf("Go at trigger adapter did not replace source trigger body cleanly:\n%s", text)
	}
	if len(program.Models) != 1 || len(program.Models[0].States) != 2 {
		t.Fatalf("adapter patch changed model topology: %#v", program.Models)
	}
	transition := program.Models[0].States[0].Transitions[0]
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerAt || transition.Trigger.Expr == nil || transition.Trigger.Expr.ID != "trigger_1" {
		t.Fatalf("adapter patch changed trigger structure: %#v", transition.Trigger)
	}
	if len(program.Behaviors) != 1 || program.Behaviors[0].ID != "trigger_1" || program.Behaviors[0].Kind != BehaviorTrigger || program.Behaviors[0].TriggerKind != TriggerAt || program.Behaviors[0].TargetLanguage != LanguageGo {
		t.Fatalf("adapter patch changed trigger behavior identity: %#v", program.Behaviors)
	}
	if program.Behaviors[0].TargetABI == nil || program.Behaviors[0].TargetABI.ReturnType != "time.Time" {
		t.Fatalf("adapter patch changed trigger ABI: %#v", program.Behaviors[0].TargetABI)
	}
}

func TestCompilerAppliesDartAtTriggerAdapterPatchWithoutChangingModel(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:   "trigger_1",
			Body: "return deadlineNow();",
			Imports: []Import{{
				Path:       "clock.dart",
				Specifiers: []ImportSpecifier{{Name: "deadlineNow"}},
			}},
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	output, program, err := compiler.Compile(context.Background(), SourceInput{Path: "timer.ts", Data: []byte(source)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguageDart,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`import "dart:async";`,
		`import "clock.dart" show deadlineNow;`,
		"FutureOr<DateTime> trigger1(Context ctx, Instance instance, Event event)",
		"return deadlineNow();",
		`final timerModel = define(`,
		`initial(target("idle"))`,
		`state(`,
		`transition([`,
		`at(trigger1)`,
		`target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Dart at trigger adapter output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return new Date();") ||
		strings.Contains(text, "Unsupported at trigger") ||
		strings.Contains(text, "_hsmcAtTransition") {
		t.Fatalf("Dart at trigger adapter did not preserve the target trigger boundary cleanly:\n%s", text)
	}
	if len(program.Models) != 1 || len(program.Models[0].States) != 2 {
		t.Fatalf("adapter patch changed model topology: %#v", program.Models)
	}
	transition := program.Models[0].States[0].Transitions[0]
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerAt || transition.Trigger.Expr == nil || transition.Trigger.Expr.ID != "trigger_1" {
		t.Fatalf("adapter patch changed trigger structure: %#v", transition.Trigger)
	}
	if len(program.Behaviors) != 1 || program.Behaviors[0].ID != "trigger_1" || program.Behaviors[0].Kind != BehaviorTrigger || program.Behaviors[0].TriggerKind != TriggerAt || program.Behaviors[0].TargetLanguage != LanguageDart {
		t.Fatalf("adapter patch changed trigger behavior identity: %#v", program.Behaviors)
	}
	if program.Behaviors[0].TargetABI == nil || program.Behaviors[0].TargetABI.ReturnType != "FutureOr<DateTime>" {
		t.Fatalf("adapter patch changed trigger ABI: %#v", program.Behaviors[0].TargetABI)
	}
}

func TestCompilerAppliesCSharpAtTriggerAdapterPatchWithoutChangingModel(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:   "trigger_1",
			Body: "return Clock.Deadline();",
			Imports: []Import{{
				Path:  "Sample.Clock",
				Alias: "Clock",
			}},
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	output, program, err := compiler.Compile(context.Background(), SourceInput{Path: "timer.ts", Data: []byte(source)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguageCSharp,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"using Stateforward.Hsm;",
		"using Clock = Sample.Clock;",
		"private static System.DateTimeOffset Trigger1(Context ctx, Instance instance, Event @event)",
		"return Clock.Deadline();",
		`public static readonly Model TimerModel = Hsm.Define(`,
		`Hsm.Initial(`,
		`Hsm.Target("idle")`,
		`Hsm.State(`,
		`"idle"`,
		`Hsm.Transition(`,
		`Hsm.At<Instance>(Trigger1)`,
		`Hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C# at trigger adapter output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return new Date();") {
		t.Fatalf("C# at trigger adapter did not replace source trigger body cleanly:\n%s", text)
	}
	if len(program.Models) != 1 || len(program.Models[0].States) != 2 {
		t.Fatalf("adapter patch changed model topology: %#v", program.Models)
	}
	transition := program.Models[0].States[0].Transitions[0]
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerAt || transition.Trigger.Expr == nil || transition.Trigger.Expr.ID != "trigger_1" {
		t.Fatalf("adapter patch changed trigger structure: %#v", transition.Trigger)
	}
	if len(program.Behaviors) != 1 || program.Behaviors[0].ID != "trigger_1" || program.Behaviors[0].Kind != BehaviorTrigger || program.Behaviors[0].TriggerKind != TriggerAt || program.Behaviors[0].TargetLanguage != LanguageCSharp {
		t.Fatalf("adapter patch changed trigger behavior identity: %#v", program.Behaviors)
	}
	if program.Behaviors[0].TargetABI == nil || program.Behaviors[0].TargetABI.ReturnType != "System.DateTimeOffset" {
		t.Fatalf("adapter patch changed trigger ABI: %#v", program.Behaviors[0].TargetABI)
	}
}

func TestCompilerAppliesJavaAtTriggerAdapterPatchWithoutChangingModel(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:      "trigger_1",
			Body:    "return Clock.deadline();",
			Imports: []Import{{Path: "com.example.Clock"}},
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	output, program, err := compiler.Compile(context.Background(), SourceInput{Path: "timer.ts", Data: []byte(source)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguageJava,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"import com.stateforward.hsm.*;",
		"import com.example.Clock;",
		"private static java.time.Instant Trigger1(Context ctx, Instance instance, Event event)",
		"return Clock.deadline();",
		`public static final Model TimerModel = Hsm.Define("Timer"`,
		`Hsm.Initial(Hsm.Target("idle"))`,
		`Hsm.State("idle"`,
		`Hsm.Transition(Hsm.At(GeneratedHsm::Trigger1), Hsm.Target("../done"))`,
		`Hsm.State("done")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Java at trigger adapter output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return new Date();") {
		t.Fatalf("Java at trigger adapter did not replace source trigger body cleanly:\n%s", text)
	}
	if len(program.Models) != 1 || len(program.Models[0].States) != 2 {
		t.Fatalf("adapter patch changed model topology: %#v", program.Models)
	}
	transition := program.Models[0].States[0].Transitions[0]
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerAt || transition.Trigger.Expr == nil || transition.Trigger.Expr.ID != "trigger_1" {
		t.Fatalf("adapter patch changed trigger structure: %#v", transition.Trigger)
	}
	if len(program.Behaviors) != 1 || program.Behaviors[0].ID != "trigger_1" || program.Behaviors[0].Kind != BehaviorTrigger || program.Behaviors[0].TriggerKind != TriggerAt || program.Behaviors[0].TargetLanguage != LanguageJava {
		t.Fatalf("adapter patch changed trigger behavior identity: %#v", program.Behaviors)
	}
	if program.Behaviors[0].TargetABI == nil || program.Behaviors[0].TargetABI.ReturnType != "java.time.Instant" {
		t.Fatalf("adapter patch changed trigger ABI: %#v", program.Behaviors[0].TargetABI)
	}
}

func TestCompilerAppliesCPPAtTriggerAdapterPatchWithoutChangingModel(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:      "trigger_1",
			Body:    "return clock::deadline();",
			Imports: []Import{{Path: "clock.hpp"}},
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	output, program, err := compiler.Compile(context.Background(), SourceInput{Path: "timer.ts", Data: []byte(source)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguageCPP,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`#include "hsm/hsm.hpp"`,
		`#include "clock.hpp"`,
		"#include <chrono>",
		"static constexpr auto trigger_1 = [](auto& signal, auto& instance, const auto& event) -> std::chrono::system_clock::time_point",
		"return clock::deadline();",
		"static constexpr auto timer_model = hsm::define(",
		`"Timer"`,
		`hsm::initial(hsm::target("/Timer/idle"))`,
		`hsm::state("idle"`,
		`hsm::transition(hsm::at(trigger_1), hsm::target("/Timer/done"))`,
		`hsm::state("done")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C++ at trigger adapter output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return new Date();") {
		t.Fatalf("C++ at trigger adapter did not replace source trigger body cleanly:\n%s", text)
	}
	if len(program.Models) != 1 || len(program.Models[0].States) != 2 {
		t.Fatalf("adapter patch changed model topology: %#v", program.Models)
	}
	transition := program.Models[0].States[0].Transitions[0]
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerAt || transition.Trigger.Expr == nil || transition.Trigger.Expr.ID != "trigger_1" {
		t.Fatalf("adapter patch changed trigger structure: %#v", transition.Trigger)
	}
	if len(program.Behaviors) != 1 || program.Behaviors[0].ID != "trigger_1" || program.Behaviors[0].Kind != BehaviorTrigger || program.Behaviors[0].TriggerKind != TriggerAt || program.Behaviors[0].TargetLanguage != LanguageCPP {
		t.Fatalf("adapter patch changed trigger behavior identity: %#v", program.Behaviors)
	}
	if program.Behaviors[0].TargetABI == nil || program.Behaviors[0].TargetABI.ReturnType != "std::chrono::system_clock::time_point" {
		t.Fatalf("adapter patch changed trigger ABI: %#v", program.Behaviors[0].TargetABI)
	}
}

func TestCompilerAppliesAtTriggerAdapterPatchForScriptAndPythonTargetsWithoutChangingModel(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	cases := []struct {
		name           string
		target         Language
		body           string
		imports        []Import
		needles        []string
		forbidden      []string
		wantReturnType string
	}{
		{
			name:   "javascript",
			target: LanguageJS,
			body:   "return deadlineNow();",
			imports: []Import{{
				Path:       "./clock",
				Specifiers: []ImportSpecifier{{Name: "deadlineNow"}},
			}},
			needles: []string{
				`import { deadlineNow } from "./clock";`,
				`function trigger1(ctx, instance, event)`,
				`return deadlineNow();`,
				`export const TimerModel = hsm.Define(`,
				`hsm.At(trigger1)`,
			},
			wantReturnType: "Date",
		},
		{
			name:   "typescript",
			target: LanguageTS,
			body:   "return deadlineNow();",
			imports: []Import{{
				Path:       "./clock",
				Specifiers: []ImportSpecifier{{Name: "deadlineNow"}},
			}},
			needles: []string{
				`import { deadlineNow } from "./clock";`,
				`function trigger1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): Date | Promise<Date>`,
				`return deadlineNow();`,
				`export const TimerModel = hsm.Define(`,
				`hsm.At(trigger1)`,
			},
			wantReturnType: "Date | Promise<Date>",
		},
		{
			name:   "python",
			target: LanguagePython,
			body:   "return deadline_now()",
			imports: []Import{{
				Path:       "clock",
				Specifiers: []ImportSpecifier{{Name: "deadline_now"}},
			}},
			needles: []string{
				`from clock import deadline_now`,
				`async def trigger_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> datetime.datetime:`,
				`return deadline_now()`,
				`timer_model = hsm.Define(`,
				`hsm.At(trigger_1)`,
			},
			wantReturnType: "datetime.datetime",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &patchingAdapter{patch: &BehaviorPatch{
				Behaviors: []Behavior{{
					ID:      "trigger_1",
					Body:    tc.body,
					Imports: tc.imports,
				}},
			}}
			compiler := NewCompiler()
			compiler.RegisterAdapter(adapter)

			output, program, err := compiler.Compile(context.Background(), SourceInput{Path: "timer.ts", Data: []byte(source)}, CompileOptions{
				From:    LanguageTS,
				To:      tc.target,
				Adapter: adapter.Name(),
			})
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			for _, needle := range tc.needles {
				if !strings.Contains(text, needle) {
					t.Fatalf("%s at trigger adapter output missing %q:\n%s", tc.name, needle, text)
				}
			}
			for _, forbidden := range append(tc.forbidden, "Original typescript behavior trigger_1 preserved for manual porting", "return new Date();") {
				if strings.Contains(text, forbidden) {
					t.Fatalf("%s at trigger adapter output contains forbidden %q:\n%s", tc.name, forbidden, text)
				}
			}
			if len(program.Models) != 1 || len(program.Models[0].States) != 2 {
				t.Fatalf("adapter patch changed model topology: %#v", program.Models)
			}
			transition := program.Models[0].States[0].Transitions[0]
			if transition.Trigger == nil || transition.Trigger.Kind != TriggerAt || transition.Trigger.Expr == nil || transition.Trigger.Expr.ID != "trigger_1" {
				t.Fatalf("adapter patch changed trigger structure: %#v", transition.Trigger)
			}
			if len(program.Behaviors) != 1 || program.Behaviors[0].ID != "trigger_1" || program.Behaviors[0].Kind != BehaviorTrigger || program.Behaviors[0].TriggerKind != TriggerAt || program.Behaviors[0].TargetLanguage != tc.target {
				t.Fatalf("adapter patch changed trigger behavior identity: %#v", program.Behaviors)
			}
			if program.Behaviors[0].TargetABI == nil || program.Behaviors[0].TargetABI.ReturnType != tc.wantReturnType {
				t.Fatalf("adapter patch changed trigger ABI: %#v", program.Behaviors[0].TargetABI)
			}
		})
	}
}

func TestCompilerAppliesAtTriggerAdapterPatchForLoweredRustAndZigTargetsWithoutChangingModel(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	cases := []struct {
		name           string
		target         Language
		body           string
		imports        []Import
		needles        []string
		forbidden      []string
		wantReturnType string
	}{
		{
			name:   "rust",
			target: LanguageRust,
			body:   "deadline()",
			imports: []Import{{
				Path:       "crate::clock",
				Specifiers: []ImportSpecifier{{Name: "deadline"}},
			}},
			needles: []string{
				`use crate::clock::{deadline};`,
				`fn trigger_1(ctx: &hsm::Context, instance: &HsmcInstance, event: &hsm::Event) -> std::time::SystemTime`,
				`deadline()`,
				`pub fn timer_model() -> hsm::Model<HsmcInstance>`,
				`hsm::at!(trigger_1)`,
			},
			forbidden:      []string{`at trigger lowered to after trigger`, `hsm::after!(trigger_1)`, `hsm::after!(hsmc_zero_duration)`},
			wantReturnType: "std::time::SystemTime",
		},
		{
			name:   "zig",
			target: LanguageZig,
			body:   "return clock.deadline();",
			imports: []Import{{
				Path:  "clock",
				Alias: "clock",
			}},
			needles: []string{
				`const clock = @import("clock");`,
				`fn trigger_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) u64`,
				`return clock.deadline();`,
				`pub const timer_model = hsm.define("Timer", .{`,
				`hsm.at(trigger_1),`,
			},
			forbidden:      []string{`Unsupported at trigger`, `at trigger lowered to after trigger`, `hsm.after(trigger_1)`, `hsm.after(hsmc_trigger_zero)`},
			wantReturnType: "u64",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &patchingAdapter{patch: &BehaviorPatch{
				Behaviors: []Behavior{{
					ID:      "trigger_1",
					Body:    tc.body,
					Imports: tc.imports,
				}},
			}}
			compiler := NewCompiler()
			compiler.RegisterAdapter(adapter)

			output, program, err := compiler.Compile(context.Background(), SourceInput{Path: "timer.ts", Data: []byte(source)}, CompileOptions{
				From:    LanguageTS,
				To:      tc.target,
				Adapter: adapter.Name(),
			})
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			for _, needle := range tc.needles {
				if !strings.Contains(text, needle) {
					t.Fatalf("%s at trigger adapter output missing %q:\n%s", tc.name, needle, text)
				}
			}
			for _, forbidden := range append(tc.forbidden, "Original typescript behavior trigger_1 preserved for manual porting", "return new Date();") {
				if strings.Contains(text, forbidden) {
					t.Fatalf("%s at trigger adapter output contains forbidden %q:\n%s", tc.name, forbidden, text)
				}
			}
			if len(program.Models) != 1 || len(program.Models[0].States) != 2 {
				t.Fatalf("adapter patch changed model topology: %#v", program.Models)
			}
			transition := program.Models[0].States[0].Transitions[0]
			if transition.Trigger == nil || transition.Trigger.Kind != TriggerAt || transition.Trigger.Expr == nil || transition.Trigger.Expr.ID != "trigger_1" {
				t.Fatalf("adapter patch changed trigger structure: %#v", transition.Trigger)
			}
			if len(program.Behaviors) != 1 || program.Behaviors[0].ID != "trigger_1" || program.Behaviors[0].Kind != BehaviorTrigger || program.Behaviors[0].TriggerKind != TriggerAt || program.Behaviors[0].TargetLanguage != tc.target {
				t.Fatalf("adapter patch changed trigger behavior identity: %#v", program.Behaviors)
			}
			if program.Behaviors[0].TargetABI == nil || program.Behaviors[0].TargetABI.ReturnType != tc.wantReturnType {
				t.Fatalf("adapter patch changed trigger ABI: %#v", program.Behaviors[0].TargetABI)
			}
		})
	}
}

func TestCompilerPreservesOmittedForeignBehaviorAfterPartialAdapterPatch(t *testing.T) {
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:   "entry_1",
			Body: `instance["log"] = ["translated"];`,
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	output, program, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`instance["log"] = ["translated"];`,
		"function guard1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): boolean {",
		"// Original go behavior guard_1 preserved for manual porting:",
		"// return len(sm.log) >= 0",
		"return false;",
		"hsm.Guard(guard1)",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "\n\treturn len(sm.log) >= 0") {
		t.Fatalf("omitted Go guard was emitted as executable TypeScript:\n%s", text)
	}
	for _, behavior := range program.Behaviors {
		if behavior.ID == "guard_1" {
			if behavior.SourceLanguage != LanguageGo || behavior.TargetLanguage != "" {
				t.Fatalf("omitted guard should retain source context, got %#v", behavior)
			}
			return
		}
	}
	t.Fatalf("compiled program missing guard_1 behavior: %#v", program.Behaviors)
}

func TestCompilerPreservesOmittedForeignGlobalAfterPartialAdapterPatch(t *testing.T) {
	source := `{
  "source_language": "go",
  "package_name": "sample",
  "globals": [
    {
      "id": "global_1",
      "language": "go",
      "code": "func translatedHelper() string { return \"translated\" }"
    },
    {
      "id": "global_2",
      "language": "go",
      "code": "func manualHelper() string { return \"manual\" }"
    }
  ],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "id": "initial_1", "owner_id": "model_1", "target": "closed" },
    "states": [{ "id": "state_1", "name": "closed", "kind": "state" }]
  }]
}`
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Globals: []CodeBlock{{
			ID:   "global_1",
			Code: `const translatedHelper = (): string => "translated";`,
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	output, program, err := compiler.Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageTS,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`const translatedHelper = (): string => "translated";`,
		"// Original go global global_2 preserved for manual porting:",
		"// Target language: typescript",
		`// func manualHelper() string { return "manual" }`,
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "\nfunc manualHelper()") {
		t.Fatalf("omitted Go global was emitted as executable TypeScript:\n%s", text)
	}
	if program.Globals[0].Language != LanguageTS || program.Globals[1].Language != LanguageGo {
		t.Fatalf("partial global patch languages = %#v", program.Globals)
	}
}

func TestCompilerAppliesCSharpOperationAdapterPatchWithoutChangingModel(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:   "operation_1",
			Body: `Approvals.Record(instance, "approve");`,
			Imports: []Import{{
				Path:  "Sample.Approvals",
				Alias: "Approvals",
			}},
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	output, program, err := compiler.Compile(context.Background(), SourceInput{Path: "door.ts", Data: []byte(source)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguageCSharp,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"using Stateforward.Hsm;",
		"using Approvals = Sample.Approvals;",
		"private static void Operation1(Context ctx, Instance instance, Event @event)",
		`Approvals.Record(instance, "approve");`,
		"public static readonly Model DoorModel = Hsm.Define(",
		`Hsm.Operation("approve", new Operation<Instance>(Operation1))`,
		`Hsm.Initial(`,
		`Hsm.Target("closed")`,
		`Hsm.State(`,
		`"closed"`,
		`Hsm.Transition(`,
		`Hsm.OnCall("approve")`,
		`Hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C# operation adapter output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) ||
		strings.Contains(text, "function operation1(") {
		t.Fatalf("C# operation adapter did not replace source operation body cleanly:\n%s", text)
	}
	if len(program.Models) != 1 || len(program.Models[0].Operations) != 1 {
		t.Fatalf("adapter patch changed operation model structure: %#v", program.Models)
	}
	operation := program.Models[0].Operations[0]
	if operation.Name != "approve" || operation.Behavior == nil || operation.Behavior.ID != "operation_1" || operation.Behavior.Kind != BehaviorOperation {
		t.Fatalf("adapter patch changed operation binding: %#v", operation)
	}
	if len(program.Behaviors) != 1 || program.Behaviors[0].ID != "operation_1" || program.Behaviors[0].Kind != BehaviorOperation || program.Behaviors[0].TargetLanguage != LanguageCSharp {
		t.Fatalf("adapter patch changed operation behavior identity: %#v", program.Behaviors)
	}
}

func TestCompilerAppliesJavaOperationAdapterPatchWithoutChangingModel(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:   "operation_1",
			Body: `Approvals.record(instance, "approve");`,
			Imports: []Import{{
				Path: "com.example.Approvals",
			}},
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	output, program, err := compiler.Compile(context.Background(), SourceInput{Path: "door.ts", Data: []byte(source)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguageJava,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"import com.stateforward.hsm.*;",
		"import com.example.Approvals;",
		"private static void Operation1(Context ctx, Instance instance, Event event)",
		`Approvals.record(instance, "approve");`,
		"public static final Model DoorModel = Hsm.Define(",
		`Hsm.Operation("approve", GeneratedHsm::Operation1)`,
		`Hsm.Initial(`,
		`Hsm.Target("closed")`,
		`Hsm.State(`,
		`"closed"`,
		`Hsm.Transition(`,
		`Hsm.OnCall("approve")`,
		`Hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Java operation adapter output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) ||
		strings.Contains(text, "function operation1(") {
		t.Fatalf("Java operation adapter did not replace source operation body cleanly:\n%s", text)
	}
	if len(program.Models) != 1 || len(program.Models[0].Operations) != 1 {
		t.Fatalf("adapter patch changed operation model structure: %#v", program.Models)
	}
	operation := program.Models[0].Operations[0]
	if operation.Name != "approve" || operation.Behavior == nil || operation.Behavior.ID != "operation_1" || operation.Behavior.Kind != BehaviorOperation {
		t.Fatalf("adapter patch changed operation binding: %#v", operation)
	}
	if len(program.Behaviors) != 1 || program.Behaviors[0].ID != "operation_1" || program.Behaviors[0].Kind != BehaviorOperation || program.Behaviors[0].TargetLanguage != LanguageJava {
		t.Fatalf("adapter patch changed operation behavior identity: %#v", program.Behaviors)
	}
}

func TestCompilerAppliesOperationAdapterPatchForScriptGoAndDartTargetsWithoutChangingModel(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	cases := []struct {
		name    string
		target  Language
		body    string
		imports []Import
		want    []string
	}{
		{
			name:   "go",
			target: LanguageGo,
			body:   `approvals.Record(instance, "approve")`,
			imports: []Import{{
				Path: "example.com/approvals",
			}},
			want: []string{
				`"example.com/approvals"`,
				"func Operation1(ctx context.Context, instance hsm.Instance, event hsm.Event)",
				`approvals.Record(instance, "approve")`,
				"var DoorModel = hsm.Define(",
				`hsm.Operation("approve", Operation1)`,
				`hsm.Initial(`,
				`hsm.Target("closed")`,
				`hsm.State(`,
				`"closed"`,
				`hsm.Transition(`,
				`hsm.OnCall("approve")`,
				`hsm.Target("../open")`,
				`"open"`,
			},
		},
		{
			name:   "dart",
			target: LanguageDart,
			body:   `adapterApprove(instance, "approve");`,
			imports: []Import{{
				Path:       "approvals.dart",
				Specifiers: []ImportSpecifier{{Name: "adapterApprove"}},
			}},
			want: []string{
				`import "approvals.dart" show adapterApprove;`,
				"FutureOr<void> operation1(Context ctx, Instance instance, Event event)",
				`adapterApprove(instance, "approve");`,
				"final doorModel = define(",
				`operation("approve", operation1)`,
				`initial(target("closed"))`,
				`state(`,
				`"closed"`,
				`transition([`,
				`onCall("approve")`,
				`target("../open")`,
				`"open"`,
			},
		},
		{
			name:   "javascript",
			target: LanguageJS,
			body:   `record(instance, "approve");`,
			imports: []Import{{
				Path:       "./approvals",
				Specifiers: []ImportSpecifier{{Name: "record"}},
			}},
			want: []string{
				`import { record } from "./approvals";`,
				"function operation1(ctx, instance, event)",
				`record(instance, "approve");`,
				"export const DoorModel = hsm.Define(",
				`hsm.Operation("approve", operation1)`,
				`hsm.Initial(`,
				`hsm.Target("closed")`,
				`hsm.State(`,
				`"closed"`,
				`hsm.Transition(`,
				`hsm.OnCall("approve")`,
				`hsm.Target("../open")`,
				`"open"`,
			},
		},
		{
			name:   "typescript",
			target: LanguageTS,
			body:   `record(instance, "approve");`,
			imports: []Import{{
				Path:       "./approvals",
				Specifiers: []ImportSpecifier{{Name: "record"}},
			}},
			want: []string{
				`import { record } from "./approvals";`,
				"function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void",
				`record(instance, "approve");`,
				"export const DoorModel = hsm.Define(",
				`hsm.Operation("approve", operation1)`,
				`hsm.Initial(`,
				`hsm.Target("closed")`,
				`hsm.State(`,
				`"closed"`,
				`hsm.Transition(`,
				`hsm.OnCall("approve")`,
				`hsm.Target("../open")`,
				`"open"`,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &patchingAdapter{patch: &BehaviorPatch{
				Behaviors: []Behavior{{
					ID:      "operation_1",
					Body:    tc.body,
					Imports: tc.imports,
				}},
			}}
			compiler := NewCompiler()
			compiler.RegisterAdapter(adapter)

			output, program, err := compiler.Compile(context.Background(), SourceInput{Path: "door.ts", Data: []byte(source)}, CompileOptions{
				From:    LanguageTS,
				To:      tc.target,
				Adapter: adapter.Name(),
			})
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			for _, needle := range tc.want {
				if !strings.Contains(text, needle) {
					t.Fatalf("%s operation adapter output missing %q:\n%s", tc.name, needle, text)
				}
			}
			if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
				strings.Contains(text, `instance.set("approved", true);`) {
				t.Fatalf("%s operation adapter did not replace source operation body cleanly:\n%s", tc.name, text)
			}
			if len(program.Models) != 1 || len(program.Models[0].Operations) != 1 {
				t.Fatalf("%s adapter patch changed operation model structure: %#v", tc.name, program.Models)
			}
			operation := program.Models[0].Operations[0]
			if operation.Name != "approve" || operation.Behavior == nil || operation.Behavior.ID != "operation_1" || operation.Behavior.Kind != BehaviorOperation {
				t.Fatalf("%s adapter patch changed operation binding: %#v", tc.name, operation)
			}
			if len(program.Behaviors) != 1 || program.Behaviors[0].ID != "operation_1" || program.Behaviors[0].Kind != BehaviorOperation || program.Behaviors[0].TargetLanguage != tc.target {
				t.Fatalf("%s adapter patch changed operation behavior identity: %#v", tc.name, program.Behaviors)
			}
		})
	}
}

func TestCompilerAppliesZigOperationAdapterPatchWithoutChangingModel(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:   "operation_1",
			Body: "_ = ctx;\n_ = event;\napprovals.record(inst, \"approve\");",
			Imports: []Import{{
				Path:  "approvals.zig",
				Alias: "approvals",
			}},
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	output, program, err := compiler.Compile(context.Background(), SourceInput{Path: "door.ts", Data: []byte(source)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguageZig,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`const approvals = @import("approvals.zig");`,
		"fn operation_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void",
		`approvals.record(inst, "approve");`,
		`pub const door_model = hsm.define("Door", .{`,
		`hsm.operation("approve", operation_1),`,
		`hsm.initial(hsm.target("closed"))`,
		`hsm.state("closed", .{`,
		`hsm.onCall("approve")`,
		`hsm.target("../open")`,
		`hsm.state("open", .{`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Zig operation adapter output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) ||
		strings.Contains(text, "Original operation approve preserved") ||
		strings.Contains(text, "on_call trigger lowered to event trigger") ||
		strings.Contains(text, `hsm.on("approve")`) {
		t.Fatalf("Zig operation adapter did not preserve the target operation boundary cleanly:\n%s", text)
	}
	if len(program.Models) != 1 || len(program.Models[0].Operations) != 1 {
		t.Fatalf("adapter patch changed operation model structure: %#v", program.Models)
	}
	operation := program.Models[0].Operations[0]
	if operation.Name != "approve" || operation.Behavior == nil || operation.Behavior.ID != "operation_1" || operation.Behavior.Kind != BehaviorOperation {
		t.Fatalf("adapter patch changed operation binding: %#v", operation)
	}
	if len(program.Behaviors) != 1 || program.Behaviors[0].ID != "operation_1" || program.Behaviors[0].Kind != BehaviorOperation || program.Behaviors[0].TargetLanguage != LanguageZig {
		t.Fatalf("adapter patch changed operation behavior identity: %#v", program.Behaviors)
	}
}

func TestCompilerAppliesRustWhenTriggerAdapterPatchWithoutChangingModel(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.When((ctx, instance, event) => {
        return instance.ready;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:   "trigger_1",
			Body: "ready(instance)",
			Imports: []Import{{
				Path:       "crate::signals",
				Specifiers: []ImportSpecifier{{Name: "ready"}},
			}},
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	output, program, err := compiler.Compile(context.Background(), SourceInput{Path: "timer.ts", Data: []byte(source)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguageRust,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"use crate::signals::{ready};",
		"fn trigger_1(ctx: &hsm::Context, instance: &HsmcInstance, event: &hsm::Event) -> bool",
		"ready(instance)",
		"pub fn timer_model() -> hsm::Model<HsmcInstance>",
		`hsm::define!("Timer"`,
		`hsm::initial!(hsm::target!("idle"))`,
		`hsm::state!("idle"`,
		`/* when trigger lowered to guard for hsm.rs */ hsm::guard!(trigger_1)`,
		`hsm::target!("../done")`,
		`hsm::state!("done")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Rust when trigger adapter output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return instance.ready;") ||
		strings.Contains(text, "hsm::when") {
		t.Fatalf("Rust when trigger adapter did not preserve the target trigger boundary cleanly:\n%s", text)
	}
	if len(program.Models) != 1 || len(program.Models[0].States) != 2 {
		t.Fatalf("adapter patch changed model topology: %#v", program.Models)
	}
	transition := program.Models[0].States[0].Transitions[0]
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerWhen || transition.Trigger.Expr == nil || transition.Trigger.Expr.ID != "trigger_1" {
		t.Fatalf("adapter patch changed trigger structure: %#v", transition.Trigger)
	}
	if len(program.Behaviors) != 1 || program.Behaviors[0].ID != "trigger_1" || program.Behaviors[0].Kind != BehaviorTrigger || program.Behaviors[0].TriggerKind != TriggerWhen || program.Behaviors[0].TargetLanguage != LanguageRust {
		t.Fatalf("adapter patch changed trigger behavior identity: %#v", program.Behaviors)
	}
}

func TestCompilerAppliesZigWhenTriggerAdapterPatchWithoutChangingModel(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.When((ctx, instance, event) => {
        return instance.ready;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:   "trigger_1",
			Body: "return signals.ready(inst);",
			Imports: []Import{{
				Path:  "signals.zig",
				Alias: "signals",
			}},
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	output, program, err := compiler.Compile(context.Background(), SourceInput{Path: "timer.ts", Data: []byte(source)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguageZig,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`const signals = @import("signals.zig");`,
		"fn trigger_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) bool",
		"return signals.ready(inst);",
		`pub const timer_model = hsm.define("Timer", .{`,
		`hsm.initial(hsm.target("idle")),`,
		`hsm.state("idle", .{`,
		`// when trigger lowered to guard for hsm.zig`,
		`hsm.guard(trigger_1),`,
		`hsm.target("../done"),`,
		`hsm.state("done", .{`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Zig when trigger adapter output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return instance.ready;") ||
		strings.Contains(text, "Unsupported when trigger") {
		t.Fatalf("Zig when trigger adapter did not preserve the target trigger boundary cleanly:\n%s", text)
	}
	if len(program.Models) != 1 || len(program.Models[0].States) != 2 {
		t.Fatalf("adapter patch changed model topology: %#v", program.Models)
	}
	transition := program.Models[0].States[0].Transitions[0]
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerWhen || transition.Trigger.Expr == nil || transition.Trigger.Expr.ID != "trigger_1" {
		t.Fatalf("adapter patch changed trigger structure: %#v", transition.Trigger)
	}
	if len(program.Behaviors) != 1 || program.Behaviors[0].ID != "trigger_1" || program.Behaviors[0].Kind != BehaviorTrigger || program.Behaviors[0].TriggerKind != TriggerWhen || program.Behaviors[0].TargetLanguage != LanguageZig {
		t.Fatalf("adapter patch changed trigger behavior identity: %#v", program.Behaviors)
	}
}

func TestCompilerAppliesCPPWhenTriggerAdapterPatchWithoutChangingModel(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.When((ctx, instance, event) => {
        return instance.ready;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:      "trigger_1",
			Body:    "return ready(instance);",
			Imports: []Import{{Path: "signals.hpp"}},
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	output, program, err := compiler.Compile(context.Background(), SourceInput{Path: "timer.ts", Data: []byte(source)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguageCPP,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`#include "signals.hpp"`,
		"static constexpr auto trigger_1 = [](auto& signal, auto& instance, const auto& event) -> bool",
		"return ready(instance);",
		`static constexpr auto timer_model = hsm::define(`,
		`"Timer"`,
		`hsm::initial(hsm::target("/Timer/idle"))`,
		`hsm::state("idle"`,
		`hsm::transition(hsm::guard(trigger_1), hsm::target("/Timer/done"))`,
		`hsm::state("done")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C++ when trigger adapter output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return instance.ready;") ||
		strings.Contains(text, "Unsupported when trigger") ||
		strings.Contains(text, "hsm::when(trigger_1)") {
		t.Fatalf("C++ when trigger adapter did not preserve the target trigger boundary cleanly:\n%s", text)
	}
	if len(program.Models) != 1 || len(program.Models[0].States) != 2 {
		t.Fatalf("adapter patch changed model topology: %#v", program.Models)
	}
	transition := program.Models[0].States[0].Transitions[0]
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerWhen || transition.Trigger.Expr == nil || transition.Trigger.Expr.ID != "trigger_1" {
		t.Fatalf("adapter patch changed trigger structure: %#v", transition.Trigger)
	}
	if len(program.Behaviors) != 1 || program.Behaviors[0].ID != "trigger_1" || program.Behaviors[0].Kind != BehaviorTrigger || program.Behaviors[0].TriggerKind != TriggerWhen || program.Behaviors[0].TargetLanguage != LanguageCPP {
		t.Fatalf("adapter patch changed trigger behavior identity: %#v", program.Behaviors)
	}
}

func TestCompilerIgnoresAdapterRequestModelMutations(t *testing.T) {
	adapter := &mutatingRequestAdapter{}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	_, program, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 {
		t.Fatalf("models = %#v", program.Models)
	}
	model := program.Models[0]
	if model.Name != "Door" || len(model.States) == 0 || model.States[0].Name != "closed" {
		t.Fatalf("adapter request mutation changed compiler-owned model: %#v", model)
	}
	if model.Initializer == nil || model.Initializer.Target != "closed" {
		t.Fatalf("adapter request mutation changed initial target: %#v", model.Initializer)
	}
	if len(model.States[0].Behaviors) == 0 || model.States[0].Behaviors[0].ID != "entry_1" {
		t.Fatalf("adapter request mutation changed state behavior refs: %#v", model.States[0].Behaviors)
	}
	for _, behavior := range program.Behaviors {
		if behavior.ID == "entry_1" {
			if behavior.Body != `instance["log"] = ["translated"];` || behavior.OwnerID != "state_1" || behavior.Kind != BehaviorEntry {
				t.Fatalf("translated behavior lost compiler-owned identity: %#v", behavior)
			}
			return
		}
	}
	t.Fatalf("compiled program missing entry_1 behavior: %#v", program.Behaviors)
}

func TestCompilerIgnoresAdapterRequestImportEventAndABIMutations(t *testing.T) {
	source := `{
  "source_language": "go",
  "package_name": "sample",
  "imports": [
    { "path": "sample/structural", "language": "go" }
  ],
  "globals": [{
    "id": "global_1",
    "language": "go",
    "code": "func helper() string { return \"closed\" }",
    "imports": [{ "path": "strings", "language": "go" }]
  }],
  "events": [
    { "id": "event_1", "symbol": "openedEvent", "name": "opened", "language": "go" }
  ],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "id": "initial_1", "owner_id": "model_1", "target": "closed" },
    "states": [{
      "id": "state_1",
      "name": "closed",
      "kind": "state",
      "behaviors": [{ "id": "entry_1", "kind": "entry" }]
    }]
  }],
  "behaviors": [{
    "id": "entry_1",
    "owner_id": "state_1",
    "kind": "entry",
    "source_language": "go",
    "body": "sm.log = append(sm.log, \"closed\")"
  }]
}`
	adapter := &mutatingRequestAdapter{}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	_, program, err := compiler.Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageTS,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 1 || program.Imports[0].Path != "sample/structural" || program.Imports[0].Language != LanguageGo {
		t.Fatalf("adapter request mutation changed compiler-owned imports: %#v", program.Imports)
	}
	if len(program.Events) != 1 || program.Events[0].ID != "event_1" || program.Events[0].Symbol != "openedEvent" || program.Events[0].Name != "opened" {
		t.Fatalf("adapter request mutation changed compiler-owned events: %#v", program.Events)
	}
	if len(program.Globals) != 1 || program.Globals[0].ID != "global_1" || program.Globals[0].Language != LanguageGo || program.Globals[0].Code != `func helper() string { return "closed" }` {
		t.Fatalf("adapter request mutation changed compiler-owned globals: %#v", program.Globals)
	}
	if len(program.Globals[0].Imports) != 1 || program.Globals[0].Imports[0].Path != "strings" || program.Globals[0].Imports[0].Language != LanguageGo {
		t.Fatalf("adapter request mutation changed global import context: %#v", program.Globals[0].Imports)
	}
	if len(program.Behaviors) != 1 {
		t.Fatalf("behaviors = %#v", program.Behaviors)
	}
	behavior := program.Behaviors[0]
	if behavior.TargetABI == nil || behavior.TargetABI.Language != LanguageTS || behavior.TargetABI.ReturnType != "void" {
		t.Fatalf("adapter request mutation changed compiler-owned ABI: %#v", behavior.TargetABI)
	}
	if behavior.TargetABI.Parameters[0].Name != "ctx" || behavior.TargetABI.Parameters[0].Type != "hsm.Context" {
		t.Fatalf("adapter request mutation changed ABI parameters: %#v", behavior.TargetABI.Parameters)
	}
	if behavior.TargetLanguage != LanguageTS || behavior.Body != `instance["log"] = ["translated"];` {
		t.Fatalf("adapter patch was not applied within behavior body boundary: %#v", behavior)
	}
}

type patchingAdapter struct {
	patch   *BehaviorPatch
	request AdapterRequest
}

func (adapter *patchingAdapter) Name() string { return "patching" }

func (adapter *patchingAdapter) Translate(_ context.Context, request AdapterRequest) (*BehaviorPatch, error) {
	adapter.request = request
	return adapter.patch, nil
}

type mutatingRequestAdapter struct{}

func (adapter *mutatingRequestAdapter) Name() string { return "mutating-request" }

func (adapter *mutatingRequestAdapter) Translate(_ context.Context, request AdapterRequest) (*BehaviorPatch, error) {
	if len(request.Imports) > 0 {
		request.Imports[0].Path = "mutated/import"
		request.Imports[0].Language = LanguagePython
	}
	if len(request.Events) > 0 {
		request.Events[0].ID = "mutated_event"
		request.Events[0].Symbol = "mutatedEvent"
		request.Events[0].Name = "mutated"
		request.Events[0].Language = LanguagePython
	}
	if len(request.Globals) > 0 {
		request.Globals[0].ID = "mutated_global"
		request.Globals[0].Language = LanguagePython
		request.Globals[0].Code = `def helper(): return "mutated"`
		if len(request.Globals[0].Imports) > 0 {
			request.Globals[0].Imports[0].Path = "mutated_global_import"
			request.Globals[0].Imports[0].Language = LanguagePython
		}
	}
	if len(request.Models) > 0 {
		request.Models[0].Name = "Mutated"
		request.Models[0].InitialTarget = "mutated"
		if len(request.Models[0].States) > 0 {
			request.Models[0].States[0].Name = "mutated"
			request.Models[0].States[0].Behaviors = []BehaviorRef{{ID: "mutated", Kind: BehaviorExit}}
		}
	}
	if len(request.Behaviors) > 0 {
		request.Behaviors[0].OwnerID = "mutated"
		request.Behaviors[0].Kind = BehaviorExit
		if request.Behaviors[0].TargetABI != nil {
			request.Behaviors[0].TargetABI.Language = LanguagePython
			request.Behaviors[0].TargetABI.ReturnType = "bool"
			if len(request.Behaviors[0].TargetABI.Parameters) > 0 {
				request.Behaviors[0].TargetABI.Parameters[0].Name = "mutated"
				request.Behaviors[0].TargetABI.Parameters[0].Type = "mutated.Type"
			}
		}
	}
	return &BehaviorPatch{Behaviors: []Behavior{{
		ID:   "entry_1",
		Body: `instance["log"] = ["translated"];`,
	}}}, nil
}
