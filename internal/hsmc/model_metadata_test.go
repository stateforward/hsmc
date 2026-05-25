package hsmc

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestGoFrontendLowersAttributesAndOperations(t *testing.T) {
	source := `package sample

import hsm "github.com/stateforward/hsm.go"

type DoorHSM struct{ hsm.HSM }

func approve(sm *DoorHSM) bool { return true }

var DoorModel = hsm.Define(
	"Door",
	hsm.Attribute("count", 1),
	hsm.Operation("approve", approve),
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed", hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open"))),
	hsm.State("open"),
)
`
	program, err := NewGoFrontend().Parse(context.Background(), SourceInput{Path: "door.go", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	model := program.Models[0]
	if len(model.Attributes) != 1 || model.Attributes[0].Name != "count" || model.Attributes[0].Default != "1" {
		t.Fatalf("attributes = %#v", model.Attributes)
	}
	if len(model.Operations) != 1 || model.Operations[0].Name != "approve" || model.Operations[0].Behavior == nil {
		t.Fatalf("operations = %#v", model.Operations)
	}
	if len(program.Behaviors) != 1 || program.Behaviors[0].Kind != BehaviorOperation {
		t.Fatalf("operation behavior not captured: %#v", program.Behaviors)
	}
}

func TestBackendsEmitAttributesAndOperations(t *testing.T) {
	source := `package sample

import hsm "github.com/stateforward/hsm.go"

type DoorHSM struct{ hsm.HSM }

func approve(sm *DoorHSM) bool { return true }

var DoorModel = hsm.Define(
	"Door",
	hsm.Attribute("count", 1),
	hsm.Operation("approve", approve),
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed", hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open"))),
	hsm.State("open"),
)
`
	cases := []struct {
		name   string
		target Language
		want   []string
		forbid []string
	}{
		{
			name:   "csharp",
			target: LanguageCSharp,
			want:   []string{`Hsm.Attribute<object>("count", 1)`, `Hsm.Operation("approve", new Operation<Instance>(Operation1))`, `Hsm.OnCall("approve")`},
		},
		{
			name:   "cpp",
			target: LanguageCPP,
			want:   []string{`hsm::attribute("count", 1)`, `hsm::operation("approve", &operation_1)`, `hsm::on_call("approve")`},
		},
		{
			name:   "dart",
			target: LanguageDart,
			want: []string{
				`attribute("count", 1)`,
				`operation("approve", operation1)`,
				`onCall("approve")`,
			},
			forbid: []string{`Attribute "count" preserved for manual porting`, `Operation "approve" -> operation1 preserved for manual porting`, `on("approve")`},
		},
		{
			name:   "go",
			target: LanguageGo,
			want:   []string{`hsm.Attribute("count", 1)`, `hsm.Operation("approve", Operation1)`, `hsm.OnCall("approve")`},
			forbid: []string{`"context"`},
		},
		{
			name:   "java",
			target: LanguageJava,
			want:   []string{`Hsm.Attribute("count", 1)`, `Hsm.Operation("approve", GeneratedHsm::Operation1)`, `Hsm.OnCall("approve")`},
		},
		{
			name:   "javascript",
			target: LanguageJS,
			want:   []string{`hsm.Attribute("count", 1)`, `hsm.Operation("approve", operation1)`, `hsm.OnCall("approve")`},
		},
		{
			name:   "python",
			target: LanguagePython,
			want:   []string{`hsm.Attribute("count", 1)`, `hsm.Operation("approve", operation_1)`, `hsm.OnCall("approve")`},
		},
		{
			name:   "rust",
			target: LanguageRust,
			want:   []string{`hsm::attribute("count", 1)`, `hsm::operation("approve", operation_1)`, `hsm::on_call("approve")`},
			forbid: []string{`Attribute "count" preserved for manual porting`, `Operation "approve" -> operation_1 preserved for manual porting`, `hsm::on!("approve")`, `// default: 1`},
		},
		{
			name:   "typescript",
			target: LanguageTS,
			want:   []string{`hsm.Attribute("count", 1)`, `hsm.Operation("approve", operation1)`, `hsm.OnCall("approve")`},
		},
		{
			name:   "zig",
			target: LanguageZig,
			want:   []string{`hsm.attribute("count", 1)`, `hsm.operation("approve", operation_1)`, `hsm.onCall("approve")`},
			forbid: []string{`Original attribute count preserved`, `Original operation approve preserved`, `hsm.on("approve")`},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(source)}, CompileOptions{
				From:    LanguageGo,
				To:      tc.target,
				Adapter: "none",
			})
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
					t.Fatalf("%s output contained forbidden %q:\n%s", tc.name, needle, text)
				}
			}
		})
	}
}

func TestCompilerPreservesModelMetadataAcrossTargets(t *testing.T) {
	source := `package sample

import hsm "github.com/stateforward/hsm.go"

type DoorHSM struct{ hsm.HSM }

func approve(sm *DoorHSM) bool { return true }

var DoorModel = hsm.Define(
	"Door",
	hsm.Attribute("count", int, 1),
	hsm.Operation("approve", approve),
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed", hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open"))),
	hsm.State("open"),
)
`
	ctx := context.Background()
	compiler := NewCompiler()
	parsed, err := NewGoFrontend().Parse(ctx, SourceInput{Path: "door.go", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	want := BuildModelViews(parsed)
	if len(want) != 1 || len(want[0].Attributes) != 1 || len(want[0].Operations) != 1 {
		t.Fatalf("baseline model metadata = %#v", want)
	}

	for _, target := range compiler.TargetLanguages() {
		t.Run(string(target), func(t *testing.T) {
			output, program, err := compiler.Compile(ctx, SourceInput{Path: "door.go", Data: []byte(source)}, CompileOptions{
				From:    LanguageGo,
				To:      target,
				Adapter: "none",
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(output) == 0 {
				t.Fatal("empty output")
			}
			if got := BuildModelViews(program); !reflect.DeepEqual(got, want) {
				t.Fatalf("%s target changed compiler-owned model metadata:\ngot:  %#v\nwant: %#v", target, got, want)
			}
		})
	}
}

func TestFrontendsAndBackendsPreserveTypedAttributes(t *testing.T) {
	goSource := `package sample

import hsm "github.com/stateforward/hsm.go"

var DoorModel = hsm.Define(
	"Door",
	hsm.Attribute("count", int, 1),
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed"),
)
`
	program, err := NewGoFrontend().Parse(context.Background(), SourceInput{Path: "door.go", Data: []byte(goSource)})
	if err != nil {
		t.Fatal(err)
	}
	attribute := program.Models[0].Attributes[0]
	if attribute.Type != "int" || !attribute.HasDefault || attribute.Default != "1" {
		t.Fatalf("Go attribute = %#v", attribute)
	}
	goOutput, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(goOutput), `// Type: int`) || !strings.Contains(string(goOutput), `hsm.Attribute("count", 1)`) {
		t.Fatalf("Go output missing commented type and default attribute:\n%s", string(goOutput))
	}
	cppOutput, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageCPP,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cppOutput), `hsm::attribute<int>("count", 1)`) {
		t.Fatalf("C++ output missing typed attribute:\n%s", string(cppOutput))
	}

	tsSource := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Attribute("count", Number, 1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`
	program, err = NewTypeScriptFrontend().Parse(context.Background(), SourceInput{Path: "door.ts", Data: []byte(tsSource)})
	if err != nil {
		t.Fatal(err)
	}
	attribute = program.Models[0].Attributes[0]
	if attribute.Type != "Number" || !attribute.HasDefault || attribute.Default != "1" {
		t.Fatalf("TypeScript attribute = %#v", attribute)
	}
	tsOutput, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.ts", Data: []byte(tsSource)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(tsOutput), `hsm.Attribute("count", Number, 1)`) {
		t.Fatalf("TS output missing typed attribute:\n%s", string(tsOutput))
	}
}

func TestFrontendsAndBackendsPreserveTypeOnlyAttributes(t *testing.T) {
	cases := []struct {
		name       string
		language   Language
		path       string
		source     string
		wantType   string
		wantOutput string
	}{
		{
			name:     "go",
			language: LanguageGo,
			path:     "door.go",
			source: `package sample

import hsm "github.com/stateforward/hsm.go"

var DoorModel = hsm.Define(
	"Door",
	hsm.Attribute("count", int),
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed"),
)
`,
			wantType:   "int",
			wantOutput: `// Type: int`,
		},
		{
			name:     "typescript",
			language: LanguageTS,
			path:     "door.ts",
			source: `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Attribute("count", Number),
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`,
			wantType:   "Number",
			wantOutput: `hsm.Attribute("count", Number)`,
		},
		{
			name:     "python",
			language: LanguagePython,
			path:     "door.py",
			source: `import hsm

model = hsm.Define(
    "Door",
    hsm.Attribute("count", int),
    hsm.Initial(hsm.Target("closed")),
    hsm.State("closed"),
)
`,
			wantType:   "int",
			wantOutput: `# Type: int`,
		},
		{
			name:     "csharp",
			language: LanguageCSharp,
			path:     "door.cs",
			source: `using Stateforward.Hsm;

public static class Sample
{
    public static readonly Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Attribute("count", typeof(int)),
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed")
    );
}
`,
			wantType:   "typeof(int)",
			wantOutput: `Hsm.Attribute<int>("count")`,
		},
		{
			name:     "java",
			language: LanguageJava,
			path:     "Door.java",
			source: `import com.stateforward.hsm.*;

public final class Door {
  static final Model DoorModel = Hsm.Define(
    "Door",
    Hsm.Attribute("count", Integer.class),
    Hsm.Initial(Hsm.Target("closed")),
    Hsm.State("closed")
  );
}
`,
			wantType:   "Integer.class",
			wantOutput: `Hsm.Attribute("count", Integer.class)`,
		},
		{
			name:     "dart",
			language: LanguageDart,
			path:     "door.dart",
			source: `import 'package:hsm/hsm.dart';

final doorModel = define('Door', [
  attribute('count', int),
  initial(target('closed')),
  state('closed'),
]);
`,
			wantType:   "int",
			wantOutput: `attribute("count", int)`,
		},
		{
			name:     "cpp",
			language: LanguageCPP,
			path:     "door.cpp",
			source: `#include "hsm/hsm.hpp"

using namespace hsm;

static constexpr auto DoorModel = define(
  "Door",
  attribute("count", std::type_identity<int>{}),
  initial(target("closed")),
  state("closed")
);
`,
			wantType:   "int",
			wantOutput: `hsm::attribute<int>("count")`,
		},
		{
			name:     "rust",
			language: LanguageRust,
			path:     "door.rs",
			source: `use hsm::*;

const DOOR: hsm::Model = define!("Door",
    attribute!("count", i32),
    initial!(target!("closed")),
    state!("closed")
);
`,
			wantType:   "i32",
			wantOutput: `// type: i32`,
		},
		{
			name:     "zig",
			language: LanguageZig,
			path:     "door.zig",
			source: `const hsm = @import("hsm");

pub const door_model = hsm.define("Door", .{
    hsm.attribute("count", i32),
    hsm.initial(hsm.target("closed")),
    hsm.state("closed", .{}),
});
`,
			wantType:   "i32",
			wantOutput: `hsm.attribute("count", i32),`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: tc.path, Data: []byte(tc.source)}, CompileOptions{
				From:    tc.language,
				To:      tc.language,
				Adapter: "none",
			})
			if err != nil {
				t.Fatal(err)
			}
			attribute := program.Models[0].Attributes[0]
			if attribute.Type != tc.wantType || attribute.HasDefault || attribute.Default != "" {
				t.Fatalf("attribute = %#v, want type %q and no default", attribute, tc.wantType)
			}
			if !strings.Contains(string(output), tc.wantOutput) {
				t.Fatalf("output missing %q:\n%s", tc.wantOutput, string(output))
			}
		})
	}
}

func TestTypeScriptFrontendLowersAttributesAndOperations(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";

function approve(instance: hsm.Instance): boolean { return true; }

export const DoorModel = hsm.Define(
  "Door",
  hsm.Attribute("count", 1),
  hsm.Operation("approve", approve),
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open"))),
  hsm.State("open"),
);
`
	program, err := NewTypeScriptFrontend().Parse(context.Background(), SourceInput{Path: "door.ts", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	model := program.Models[0]
	if len(model.Attributes) != 1 || model.Attributes[0].Name != "count" || model.Attributes[0].Default != "1" {
		t.Fatalf("attributes = %#v", model.Attributes)
	}
	if len(model.Operations) != 1 || model.Operations[0].Name != "approve" || model.Operations[0].Behavior == nil {
		t.Fatalf("operations = %#v", model.Operations)
	}
}

func TestFrontendsLowerStringBehaviorsAsOperationReferences(t *testing.T) {
	cases := []struct {
		name     string
		language Language
		path     string
		source   string
	}{
		{
			name:     "go",
			language: LanguageGo,
			path:     "door.go",
			source: `package sample

import hsm "github.com/stateforward/hsm.go"

func approve() bool { return true }

var DoorModel = hsm.Define(
	"Door",
	hsm.Operation("approve", approve),
	hsm.Initial(hsm.Target("closed"), hsm.Effect("approve")),
	hsm.State("closed",
		hsm.Entry("approve"),
		hsm.Transition(hsm.On("open"), hsm.Guard("approve"), hsm.Target("../open"), hsm.Effect("approve")),
	),
	hsm.State("open"),
)
`,
		},
		{
			name:     "csharp",
			language: LanguageCSharp,
			path:     "door.cs",
			source: `using Stateforward.Hsm;

public static class Sample
{
    static bool Approve(Context ctx, Instance instance, Event @event) { return true; }

    public static readonly Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Operation("approve", Approve),
        Hsm.Initial(Hsm.Target("closed"), Hsm.Effect("approve")),
        Hsm.State("closed",
            Hsm.Entry("approve"),
            Hsm.Transition(Hsm.On("open"), Hsm.Guard("approve"), Hsm.Target("../open"), Hsm.Effect("approve"))
        ),
        Hsm.State("open")
    );
}
`,
		},
		{
			name:     "cpp",
			language: LanguageCPP,
			path:     "door.cpp",
			source: `#include "hsm/hsm.hpp"

using namespace hsm;

static auto approve = [](auto& ctx, auto& instance, const auto& event) { return true; };

static auto DoorModel = define(
  "Door",
  operation("approve", approve),
  initial(target("closed"), effect("approve")),
  state("closed",
    entry("approve"),
    transition(on("open"), guard("approve"), target("../open"), effect("approve"))
  ),
  state("open")
);
`,
		},
		{
			name:     "dart",
			language: LanguageDart,
			path:     "door.dart",
			source: `import 'package:hsm/hsm.dart';

bool approve(ctx, instance, event) => true;

final doorModel = define('Door', [
  operation('approve', approve),
  initial(target('closed'), [effect('approve')]),
  state('closed', [
    entry('approve'),
    transition([
      on('open'),
      guard('approve'),
      target('../open'),
      effect('approve'),
    ]),
  ]),
  state('open'),
]);
`,
		},
		{
			name:     "javascript",
			language: LanguageJS,
			path:     "door.js",
			source: `import * as hsm from "@stateforward/hsm";

const approve = () => true;

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", approve),
  hsm.Initial(hsm.Target("closed"), hsm.Effect("approve")),
  hsm.State("closed",
    hsm.Entry("approve"),
    hsm.Transition(hsm.On("open"), hsm.Guard("approve"), hsm.Target("../open"), hsm.Effect("approve")),
  ),
  hsm.State("open"),
);
`,
		},
		{
			name:     "java",
			language: LanguageJava,
			path:     "Door.java",
			source: `import com.stateforward.hsm.*;

public final class Sample {
    static boolean approve(Context ctx, Instance instance, Event event) { return true; }

    static final Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Operation("approve", Sample::approve),
        Hsm.Initial(Hsm.Target("closed"), Hsm.Effect("approve")),
        Hsm.State("closed",
            Hsm.Entry("approve"),
            Hsm.Transition(Hsm.On("open"), Hsm.Guard("approve"), Hsm.Target("../open"), Hsm.Effect("approve"))
        ),
        Hsm.State("open")
    );
}
`,
		},
		{
			name:     "python",
			language: LanguagePython,
			path:     "door.py",
			source: `import hsm

async def approve(ctx, instance, event):
    return True

model = hsm.Define(
    "Door",
    hsm.Operation("approve", approve),
    hsm.Initial(hsm.Target("closed"), hsm.Effect("approve")),
    hsm.State("closed",
        hsm.Entry("approve"),
        hsm.Transition(hsm.On("open"), hsm.Guard("approve"), hsm.Target("../open"), hsm.Effect("approve")),
    ),
    hsm.State("open"),
)
`,
		},
		{
			name:     "rust",
			language: LanguageRust,
			path:     "door.rs",
			source: `use hsm::*;

fn approve(ctx: &hsm::Context, instance: &HsmcInstance, event: &hsm::Event) -> bool {
    true
}

const DOOR: hsm::Model = define!("Door",
    operation!("approve", approve),
    initial!(target!("closed"), effect!("approve")),
    state!("closed",
        entry!("approve"),
        transition!(on!("open"), guard!("approve"), target!("../open"), effect!("approve"))
    ),
    state!("open")
);
`,
		},
		{
			name:     "typescript",
			language: LanguageTS,
			path:     "door.ts",
			source: `import * as hsm from "@stateforward/hsm.ts";

const approve = () => true;

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", approve),
  hsm.Initial(hsm.Target("closed"), hsm.Effect("approve")),
  hsm.State("closed",
    hsm.Entry("approve"),
    hsm.Transition(hsm.On("open"), hsm.Guard("approve"), hsm.Target("../open"), hsm.Effect("approve")),
  ),
  hsm.State("open"),
);
`,
		},
		{
			name:     "zig",
			language: LanguageZig,
			path:     "door.zig",
			source: `const hsm = @import("hsm");

fn approve(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) bool {
    _ = ctx;
    _ = inst;
    _ = event;
    return true;
}

pub const door_model = hsm.define("Door", .{
    hsm.operation("approve", approve),
    hsm.initial(hsm.target("closed"), hsm.effect("approve")),
    hsm.state("closed", .{
        hsm.entry("approve"),
        hsm.transition(.{
            hsm.on("open"),
            hsm.guard("approve"),
            hsm.target("../open"),
            hsm.effect("approve"),
        }),
    }),
    hsm.state("open", .{}),
});
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: tc.path, Data: []byte(tc.source)}, CompileOptions{
				From:    tc.language,
				To:      LanguageGo,
				Adapter: "none",
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(program.Behaviors) != 1 || program.Behaviors[0].Kind != BehaviorOperation {
				t.Fatalf("behaviors = %#v, want only operation implementation", program.Behaviors)
			}
			model := program.Models[0]
			if model.Initializer == nil || len(model.Initializer.Effects) != 1 || model.Initializer.Effects[0].Operation != "approve" {
				t.Fatalf("initial effects = %#v", model.Initializer)
			}
			state := model.States[0]
			if len(state.Behaviors) != 1 || state.Behaviors[0].Operation != "approve" {
				t.Fatalf("state behaviors = %#v", state.Behaviors)
			}
			transition := state.Transitions[0]
			if transition.Guard == nil || transition.Guard.Operation != "approve" || len(transition.Effects) != 1 || transition.Effects[0].Operation != "approve" {
				t.Fatalf("transition operation refs = guard %#v effects %#v", transition.Guard, transition.Effects)
			}
			text := string(output)
			for _, needle := range []string{`hsm.Entry("approve")`, `hsm.Guard("approve")`, `hsm.Effect("approve")`} {
				if !strings.Contains(text, needle) {
					t.Fatalf("Go output missing %q:\n%s", needle, text)
				}
			}
		})
	}
}

func TestValidateRejectsDuplicateAttributesAndOperations(t *testing.T) {
	program := &Program{Models: []Model{{
		ID:          "model_1",
		Name:        "Door",
		Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
		Attributes:  []Attribute{{ID: "attribute_1", Name: "count"}, {ID: "attribute_2", Name: "count"}},
		Operations:  []Operation{{ID: "operation_1", Name: "approve"}, {ID: "operation_2", Name: "approve"}},
		States:      []State{{ID: "state_1", Name: "closed", Kind: StateKindState}},
	}}}
	err := Validate(program)
	if err == nil {
		t.Fatal("expected duplicate diagnostics")
	}
	if !strings.Contains(err.Error(), "duplicate attribute") {
		t.Fatalf("expected duplicate attribute diagnostic, got %v", err)
	}
}

func TestValidateRejectsDuplicateStructuralIDsAndModelNames(t *testing.T) {
	program := &Program{Models: []Model{
		{
			ID:          "model_1",
			Name:        "Door",
			Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
			Attributes:  []Attribute{{ID: "attribute_1", Name: "count"}, {ID: "attribute_1", Name: "other"}},
			Operations:  []Operation{{ID: "operation_decl_1", Name: "approve"}, {ID: "operation_decl_1", Name: "reject"}},
			States: []State{{
				ID:   "state_1",
				Name: "closed",
				Kind: StateKindState,
				Transitions: []Transition{
					{OwnerID: "state_1", ID: "transition_1", Target: "../closed"},
					{OwnerID: "state_1", ID: "transition_1", Target: "../closed"},
				},
			}},
		},
		{
			ID:          "model_2",
			Name:        "Door",
			Initializer: &Initial{OwnerID: "model_2", ID: "initial_2", Target: "closed"},
			States:      []State{{ID: "state_1", Name: "closed", Kind: StateKindState}},
		},
	}}

	err := Validate(program)
	if err == nil {
		t.Fatal("expected structural duplicate diagnostics")
	}
	text := err.Error()
	if diagnostics, ok := err.(Diagnostics); ok {
		messages := make([]string, 0, len(diagnostics))
		for _, diagnostic := range diagnostics {
			messages = append(messages, diagnostic.Message)
		}
		text = strings.Join(messages, "\n")
	}
	for _, needle := range []string{
		`duplicate model name "Door"`,
		`duplicate attribute id "attribute_1"`,
		`duplicate operation id "operation_decl_1"`,
		`duplicate transition id "transition_1"`,
		`duplicate state id "state_1"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected %q in diagnostics, got %v", needle, err)
		}
	}
}

func TestValidateRejectsEmptyStructuralIDs(t *testing.T) {
	program := &Program{Models: []Model{{
		ID:          "",
		Name:        "Door",
		Initializer: &Initial{OwnerID: "", ID: "", Target: "closed"},
		Attributes:  []Attribute{{ID: "", Name: "count"}},
		Operations:  []Operation{{ID: "", Name: "approve"}},
		States: []State{{
			ID:   "",
			Name: "closed",
			Kind: StateKindState,
			Transitions: []Transition{{
				OwnerID: "",
				ID:      "",
				Target:  "../closed",
			}},
		}},
	}}}

	err := Validate(program)
	if err == nil {
		t.Fatal("expected empty structural id diagnostics")
	}
	text := err.Error()
	if diagnostics, ok := err.(Diagnostics); ok {
		messages := make([]string, 0, len(diagnostics))
		for _, diagnostic := range diagnostics {
			messages = append(messages, diagnostic.Message)
		}
		text = strings.Join(messages, "\n")
	}
	for _, needle := range []string{
		"model id is empty",
		"attribute id is empty",
		"operation id is empty",
		"state id is empty",
		"initial id is empty",
		"transition id is empty",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected %q in diagnostics, got %v", needle, err)
		}
	}
}

func TestValidateRejectsPaddedIDsAndOwnerIDs(t *testing.T) {
	program := &Program{
		Globals: []CodeBlock{{ID: " global_1"}},
		Events:  []EventDecl{{ID: " event_1", Symbol: "goEvent", Name: "go"}},
		Behaviors: []Behavior{
			{ID: " entry_bad", OwnerID: "state_1", Kind: BehaviorEntry},
			{ID: "entry_1", OwnerID: " state_1", Kind: BehaviorEntry},
		},
		Models: []Model{{
			ID:          " model_1",
			Name:        "Door",
			Initializer: &Initial{OwnerID: " model_1", ID: " initial_1", Target: "closed"},
			Attributes:  []Attribute{{ID: " attribute_1", Name: "count"}},
			Operations:  []Operation{{ID: " operation_1", Name: "approve"}},
			States: []State{{
				ID:        " state_1",
				Name:      "closed",
				Kind:      StateKindState,
				Behaviors: []BehaviorRef{{ID: " entry_1", Kind: BehaviorEntry}},
				Transitions: []Transition{{
					OwnerID: " state_1",
					ID:      " transition_1",
					Target:  "../closed",
				}},
			}},
		}},
	}

	err := Validate(program)
	if err == nil {
		t.Fatal("expected padded id diagnostics")
	}
	text := err.Error()
	if diagnostics, ok := err.(Diagnostics); ok {
		var messages []string
		for _, diagnostic := range diagnostics {
			messages = append(messages, diagnostic.Message)
		}
		text = strings.Join(messages, "\n")
	}
	for _, needle := range []string{
		`global id " global_1" has leading or trailing whitespace`,
		`event id " event_1" has leading or trailing whitespace`,
		`behavior id " entry_bad" has leading or trailing whitespace`,
		`behavior "entry_1" owner_id " state_1" has leading or trailing whitespace`,
		`model id " model_1" has leading or trailing whitespace`,
		`initial id " initial_1" has leading or trailing whitespace`,
		`initial " initial_1" owner_id " model_1" has leading or trailing whitespace`,
		`attribute id " attribute_1" has leading or trailing whitespace`,
		`operation id " operation_1" has leading or trailing whitespace`,
		`state id " state_1" has leading or trailing whitespace`,
		`transition id " transition_1" has leading or trailing whitespace`,
		`transition " transition_1" owner_id " state_1" has leading or trailing whitespace`,
		`behavior reference id " entry_1" has leading or trailing whitespace`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected %q in diagnostics, got %v", needle, err)
		}
	}
}

func TestValidateRejectsMismatchedStructuralOwners(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{
			{ID: "entry_1", OwnerID: "state_2", Kind: BehaviorEntry},
			{ID: "effect_1", OwnerID: "missing_owner", Kind: BehaviorEffect},
		},
		Models: []Model{{
			ID:          "model_1",
			Name:        "Door",
			Initializer: &Initial{OwnerID: "state_1", ID: "initial_1", Target: "../open"},
			States: []State{
				{
					ID:        "state_1",
					Name:      "closed",
					Kind:      StateKindState,
					Behaviors: []BehaviorRef{{ID: "entry_1", Kind: BehaviorEntry}},
					Transitions: []Transition{{
						OwnerID: "model_1",
						ID:      "transition_1",
						Target:  "open",
						Effects: []BehaviorRef{{ID: "effect_1", Kind: BehaviorEffect}},
					}},
				},
				{ID: "state_2", Name: "open", Kind: StateKindState},
			},
		}},
	}

	err := Validate(program)
	if err == nil {
		t.Fatal("expected owner diagnostics")
	}
	text := err.Error()
	if diagnostics, ok := err.(Diagnostics); ok {
		messages := make([]string, 0, len(diagnostics))
		for _, diagnostic := range diagnostics {
			messages = append(messages, diagnostic.Message)
		}
		text = strings.Join(messages, "\n")
	}
	for _, needle := range []string{
		`behavior "effect_1" has unknown owner_id "missing_owner"`,
		`initial "initial_1" has owner_id "state_1" but is contained by "model_1"`,
		`transition "transition_1" has owner_id "model_1" but is contained by "state_1"`,
		`behavior "entry_1" has owner_id "state_2" but reference is owned by "state_1"`,
		`behavior "effect_1" has owner_id "missing_owner" but reference is owned by "model_1"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected %q in diagnostics, got %v", needle, err)
		}
	}
}

func TestValidateRejectsBehaviorRefsInWrongStructuralSlots(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{
			{ID: "operation_1", OwnerID: "model_1", Kind: BehaviorOperation},
			{ID: "guard_1", OwnerID: "state_1", Kind: BehaviorGuard},
			{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry},
			{ID: "effect_1", OwnerID: "state_1", Kind: BehaviorEffect},
		},
		Models: []Model{{
			ID:          "model_1",
			Name:        "Door",
			Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed", Effects: []BehaviorRef{{ID: "entry_1", Kind: BehaviorEntry}}},
			Operations:  []Operation{{ID: "operation_decl_1", Name: "approve", Behavior: &BehaviorRef{ID: "guard_1", Kind: BehaviorGuard}}},
			States: []State{{
				ID:        "state_1",
				Name:      "closed",
				Kind:      StateKindState,
				Behaviors: []BehaviorRef{{ID: "effect_1", Kind: BehaviorEffect}},
				Transitions: []Transition{{
					OwnerID: "state_1",
					ID:      "transition_1",
					Target:  "../closed",
					Guard:   &BehaviorRef{ID: "entry_1", Kind: BehaviorEntry},
					Trigger: &Trigger{Kind: TriggerWhen, Expr: &BehaviorRef{ID: "effect_1", Kind: BehaviorEffect}},
					Effects: []BehaviorRef{{ID: "guard_1", Kind: BehaviorGuard}},
				}},
			}},
		}},
	}

	err := Validate(program)
	if err == nil {
		t.Fatal("expected behavior slot diagnostics")
	}
	text := err.Error()
	if diagnostics, ok := err.(Diagnostics); ok {
		messages := make([]string, 0, len(diagnostics))
		for _, diagnostic := range diagnostics {
			messages = append(messages, diagnostic.Message)
		}
		text = strings.Join(messages, "\n")
	}
	for _, needle := range []string{
		`behavior reference kind "guard" is not valid in this position`,
		`behavior reference kind "entry" is not valid in this position`,
		`behavior reference kind "effect" is not valid in this position`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected %q in diagnostics, got %v", needle, err)
		}
	}
}

func TestValidateRejectsMismatchedTriggerBehaviorKind(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "trigger_1",
			OwnerID:        "state_1",
			Kind:           BehaviorTrigger,
			TriggerKind:    TriggerWhen,
			SourceLanguage: LanguageGo,
		}},
		Models: []Model{{
			ID:          "model_1",
			Name:        "Door",
			Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
			States: []State{
				{
					ID:   "state_1",
					Name: "closed",
					Kind: StateKindState,
					Transitions: []Transition{{
						OwnerID: "state_1",
						ID:      "transition_1",
						Target:  "../open",
						Trigger: &Trigger{Kind: TriggerAfter, Expr: &BehaviorRef{ID: "trigger_1", Kind: BehaviorTrigger}},
					}},
				},
				{ID: "state_2", Name: "open", Kind: StateKindState},
			},
		}},
	}

	err := Validate(program)
	if err == nil || !strings.Contains(err.Error(), `trigger behavior "trigger_1" has trigger_kind "when" but reference is owned by "after" trigger`) {
		t.Fatalf("expected mismatched trigger behavior diagnostic, got %v", err)
	}
}

func TestValidateRejectsExpressionTriggerWithValueMetadata(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "trigger_1",
			OwnerID:        "state_1",
			Kind:           BehaviorTrigger,
			TriggerKind:    TriggerWhen,
			SourceLanguage: LanguageGo,
		}},
		Models: []Model{{
			ID:          "model_1",
			Name:        "Door",
			Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
			Attributes:  []Attribute{{ID: "attribute_1", Name: "count"}},
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
							Expr:     &BehaviorRef{ID: "trigger_1", Kind: BehaviorTrigger},
						},
					}},
				},
				{ID: "state_2", Name: "open", Kind: StateKindState},
			},
		}},
	}

	err := Validate(program)
	if err == nil || !diagnosticsContain(err, `when trigger must not combine a behavior expression with trigger value metadata`) {
		t.Fatalf("expected expression/value trigger diagnostic, got %v", err)
	}
}

func TestValidateRejectsInvalidSymbolicKinds(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{
			{ID: "entry_1", Kind: BehaviorKind("startup")},
			{ID: "trigger_1", Kind: BehaviorTrigger, TriggerKind: TriggerKind("soon")},
			{ID: "effect_1", Kind: BehaviorEffect, TriggerKind: TriggerAfter},
		},
		Models: []Model{{
			ID:          "model_1",
			Name:        "Door",
			Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
			States: []State{{
				ID:        "state_1",
				Name:      "closed",
				Kind:      StateKind("compound"),
				Behaviors: []BehaviorRef{{ID: "entry_1", Kind: BehaviorKind("startup")}},
				Transitions: []Transition{{
					OwnerID: "state_1",
					ID:      "transition_1",
					Target:  "../closed",
					Trigger: &Trigger{Kind: TriggerKind("sometimes"), Value: "open", IsString: true},
				}},
			}},
		}},
	}

	err := Validate(program)
	if err == nil {
		t.Fatal("expected invalid symbolic kind diagnostics")
	}
	text := err.Error()
	if diagnostics, ok := err.(Diagnostics); ok {
		messages := make([]string, 0, len(diagnostics))
		for _, diagnostic := range diagnostics {
			messages = append(messages, diagnostic.Message)
		}
		text = strings.Join(messages, "\n")
	}
	for _, needle := range []string{
		`behavior "entry_1" has invalid kind "startup"`,
		`behavior "trigger_1" has invalid trigger_kind "soon"`,
		`behavior "effect_1" has trigger_kind "after" but kind "effect"`,
		`state "closed" has invalid kind "compound"`,
		`behavior reference has invalid kind "startup"`,
		`trigger has invalid kind "sometimes"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected %q in diagnostics, got %v", needle, err)
		}
	}
}

func TestValidateRejectsUnknownOperationBehaviorReferences(t *testing.T) {
	program := &Program{Models: []Model{{
		ID:          "model_1",
		Name:        "Door",
		Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed", Effects: []BehaviorRef{{Kind: BehaviorEffect, Operation: "missing"}}},
		States: []State{
			{ID: "state_1", Name: "closed", Kind: StateKindState, Behaviors: []BehaviorRef{{Kind: BehaviorEntry, Operation: "missing"}}, Transitions: []Transition{
				{OwnerID: "state_1", ID: "transition_1", Target: "../open", Guard: &BehaviorRef{Kind: BehaviorGuard, Operation: "missing"}, Effects: []BehaviorRef{{Kind: BehaviorEffect, Operation: "missing"}}},
			}},
			{ID: "state_2", Name: "open", Kind: StateKindState},
		},
	}}}
	err := Validate(program)
	if err == nil {
		t.Fatal("expected unknown operation reference diagnostics")
	}
	var messages []string
	if diagnostics, ok := err.(Diagnostics); ok {
		for _, diagnostic := range diagnostics {
			messages = append(messages, diagnostic.Message)
		}
	} else {
		messages = append(messages, err.Error())
	}
	if text := strings.Join(messages, "\n"); !strings.Contains(text, `operation reference "missing" does not reference a model operation`) {
		t.Fatalf("expected missing operation diagnostics, got %v", err)
	}
}

func TestValidateRejectsInlineOperationImplementation(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "operation_1",
			OwnerID:        "model_1",
			Kind:           BehaviorOperation,
			Inline:         true,
			SourceLanguage: LanguageTS,
			Body:           "return true",
		}},
		Models: []Model{{
			ID:          "model_1",
			Name:        "Door",
			Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
			Operations:  []Operation{{ID: "operation_decl_1", Name: "approve", Behavior: &BehaviorRef{ID: "operation_1", Kind: BehaviorOperation}}},
			States:      []State{{ID: "state_1", Name: "closed", Kind: StateKindState}},
		}},
	}

	err := Validate(program)
	if err == nil || !strings.Contains(err.Error(), `operation "approve" implementation must be a callable reference, not an inline anonymous callable`) {
		t.Fatalf("expected inline operation diagnostic, got %v", err)
	}
}

func TestCompileRejectsInlineOperationImplementation(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", () => true),
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`
	_, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.ts", Data: []byte(source)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err == nil || !strings.Contains(err.Error(), `operation "approve" implementation must be a callable reference, not an inline anonymous callable`) {
		t.Fatalf("expected inline operation diagnostic, got %v", err)
	}
}

func TestCompileRejectsInlineOperationImplementationAcrossSources(t *testing.T) {
	cases := []struct {
		name     string
		language Language
		path     string
		source   string
	}{
		{
			name:     "csharp",
			language: LanguageCSharp,
			path:     "door.cs",
			source: `using Stateforward.Hsm;

public static class Sample
{
    public static readonly Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Operation("approve", (ctx, instance, @event) => true),
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed")
    );
}
`,
		},
		{
			name:     "cpp",
			language: LanguageCPP,
			path:     "door.cpp",
			source: `#include "hsm/hsm.hpp"

using namespace hsm;

static auto DoorModel = define(
  "Door",
  operation("approve", [](auto& ctx, auto& instance, const auto& event) { return true; }),
  initial(target("closed")),
  state("closed")
);
`,
		},
		{
			name:     "dart",
			language: LanguageDart,
			path:     "door.dart",
			source: `import 'package:hsm/hsm.dart';

final doorModel = define('Door', [
  operation('approve', (ctx, instance, event) => true),
  initial(target('closed')),
  state('closed'),
]);
`,
		},
		{
			name:     "go",
			language: LanguageGo,
			path:     "door.go",
			source: `package sample

import (
	"context"

	hsm "github.com/stateforward/hsm.go"
)

var DoorModel = hsm.Define(
	"Door",
	hsm.Operation("approve", func(ctx context.Context, instance hsm.Instance, event hsm.Event) bool { return true }),
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed"),
)
`,
		},
		{
			name:     "java",
			language: LanguageJava,
			path:     "Door.java",
			source: `import com.stateforward.hsm.*;

public final class Sample {
    static final Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Operation("approve", (ctx, instance, event) -> true),
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed")
    );
}
`,
		},
		{
			name:     "javascript",
			language: LanguageJS,
			path:     "door.js",
			source: `import * as hsm from "@stateforward/hsm";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", () => true),
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`,
		},
		{
			name:     "python",
			language: LanguagePython,
			path:     "door.py",
			source: `import hsm

model = hsm.Define(
    "Door",
    hsm.Operation("approve", lambda ctx, instance, event: True),
    hsm.Initial(hsm.Target("closed")),
    hsm.State("closed"),
)
`,
		},
		{
			name:     "rust",
			language: LanguageRust,
			path:     "door.rs",
			source: `use hsm::*;

const DOOR: hsm::Model = define!("Door",
    operation!("approve", |ctx, instance, event| true),
    initial!(target!("closed")),
    state!("closed")
);
`,
		},
		{
			name:     "typescript",
			language: LanguageTS,
			path:     "door.ts",
			source: `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", () => true),
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: tc.path, Data: []byte(tc.source)}, CompileOptions{
				From:    tc.language,
				To:      LanguageGo,
				Adapter: "none",
			})
			if err == nil || !strings.Contains(err.Error(), `operation "approve" implementation must be a callable reference, not an inline anonymous callable`) {
				t.Fatalf("expected inline operation diagnostic, got %v", err)
			}
		})
	}
}

func TestValidateRejectsAmbiguousBehaviorReferences(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry}},
		Models: []Model{{
			ID:          "model_1",
			Name:        "Door",
			Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
			Operations:  []Operation{{ID: "operation_1", Name: "approve"}},
			States: []State{
				{
					ID:        "state_1",
					Name:      "closed",
					Kind:      StateKindState,
					Behaviors: []BehaviorRef{{ID: "entry_1", Kind: BehaviorEntry, Operation: "approve"}},
					Transitions: []Transition{{
						OwnerID: "state_1",
						ID:      "transition_1",
						Target:  "../open",
					}},
				},
				{ID: "state_2", Name: "open", Kind: StateKindState},
			},
		}},
	}

	err := Validate(program)
	if err == nil {
		t.Fatal("expected ambiguous behavior reference diagnostic")
	}
	text := err.Error()
	if diagnostics, ok := err.(Diagnostics); ok {
		var messages []string
		for _, diagnostic := range diagnostics {
			messages = append(messages, diagnostic.Message)
		}
		text = strings.Join(messages, "\n")
	}
	if !strings.Contains(text, `behavior reference sets both id "entry_1" and operation "approve"`) {
		t.Fatalf("expected ambiguous behavior reference diagnostic, got %v", err)
	}
}

func TestValidateRejectsEmptyBehaviorReferences(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:          "model_1",
			Name:        "Door",
			Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
			States: []State{{
				ID:        "state_1",
				Name:      "closed",
				Kind:      StateKindState,
				Behaviors: []BehaviorRef{{Kind: BehaviorEntry}},
			}},
		}},
	}

	err := Validate(program)
	if err == nil {
		t.Fatal("expected empty behavior reference diagnostic")
	}
	text := err.Error()
	if diagnostics, ok := err.(Diagnostics); ok {
		var messages []string
		for _, diagnostic := range diagnostics {
			messages = append(messages, diagnostic.Message)
		}
		text = strings.Join(messages, "\n")
	}
	if !strings.Contains(text, `behavior reference has empty id`) {
		t.Fatalf("expected empty behavior reference diagnostic, got %v", err)
	}
}

func TestValidateRejectsBlankAndPaddedStatePaths(t *testing.T) {
	program := &Program{Models: []Model{{
		ID:          "model_1",
		Name:        "Door",
		Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "   "},
		States: []State{{
			ID:   "state_1",
			Name: "closed",
			Kind: StateKindState,
			Transitions: []Transition{
				{OwnerID: "state_1", ID: "transition_1", Source: "  ", Target: "../open"},
				{OwnerID: "state_1", ID: "transition_2", Source: "../closed", Target: " ../open"},
			},
		}, {
			ID:   "state_2",
			Name: "open",
			Kind: StateKindState,
		}},
	}}}

	err := Validate(program)
	if err == nil {
		t.Fatal("expected blank and padded state path diagnostics")
	}
	text := err.Error()
	if diagnostics, ok := err.(Diagnostics); ok {
		var messages []string
		for _, diagnostic := range diagnostics {
			messages = append(messages, diagnostic.Message)
		}
		text = strings.Join(messages, "\n")
	}
	for _, needle := range []string{
		`initial transition is missing target`,
		`transition "transition_1" source is empty`,
		`transition "transition_2" target " ../open" has leading or trailing whitespace`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected %q in diagnostics, got %v", needle, err)
		}
	}
	if strings.Contains(text, `resolves to missing state "/Door/ "`) {
		t.Fatalf("blank path should be rejected before path resolution, got %v", err)
	}
}

func TestValidateRejectsTransitionSourcePseudostates(t *testing.T) {
	program := &Program{Models: []Model{{
		ID:          "model_1",
		Name:        "Door",
		Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "parent/done"},
		States: []State{{
			ID:   "state_1",
			Name: "parent",
			Kind: StateKindState,
			States: []State{{
				ID:   "state_2",
				Name: "choice",
				Kind: StateKindChoice,
				Transitions: []Transition{{
					OwnerID: "state_2",
					ID:      "transition_1",
					Target:  "../done",
				}},
			}, {
				ID:   "state_3",
				Name: "done",
				Kind: StateKindState,
			}},
			Transitions: []Transition{{
				OwnerID: "state_1",
				ID:      "transition_2",
				Source:  "choice",
				Target:  "done",
			}},
		}},
	}}}

	err := Validate(program)
	if err == nil {
		t.Fatal("expected pseudostate source diagnostic")
	}
	text := err.Error()
	if diagnostics, ok := err.(Diagnostics); ok {
		var messages []string
		for _, diagnostic := range diagnostics {
			messages = append(messages, diagnostic.Message)
		}
		text = strings.Join(messages, "\n")
	}
	if !strings.Contains(text, `source "choice" resolves to pseudostate "/Door/parent/choice"`) {
		t.Fatalf("expected pseudostate source diagnostic, got %v", err)
	}
}

func TestValidateRejectsBlankAndPaddedSymbolicNames(t *testing.T) {
	program := &Program{
		Events: []EventDecl{
			{ID: "event_1", Symbol: " open", Name: "open"},
			{ID: "event_2", Symbol: "close", Name: "close "},
		},
		Models: []Model{{
			ID:          "model_1",
			Name:        " Door",
			Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
			Attributes:  []Attribute{{ID: "attribute_1", Name: " count"}},
			Operations:  []Operation{{ID: "operation_1", Name: "approve "}},
			States: []State{{
				ID:   "state_1",
				Name: "closed",
				Kind: StateKindState,
				Transitions: []Transition{{
					OwnerID: "state_1",
					ID:      "transition_1",
					Target:  "../closed",
					Trigger: &Trigger{Kind: TriggerOnCall, Value: " approve", IsString: true},
					Effects: []BehaviorRef{{Kind: BehaviorEffect, Operation: " approve"}},
				}},
			}, {
				ID:   "state_2",
				Name: " ",
				Kind: StateKindState,
			}},
		}},
	}

	err := Validate(program)
	if err == nil {
		t.Fatal("expected symbolic name diagnostics")
	}
	text := err.Error()
	if diagnostics, ok := err.(Diagnostics); ok {
		var messages []string
		for _, diagnostic := range diagnostics {
			messages = append(messages, diagnostic.Message)
		}
		text = strings.Join(messages, "\n")
	}
	for _, needle := range []string{
		`event symbol " open" has leading or trailing whitespace`,
		`event name "close " has leading or trailing whitespace`,
		`invalid model name " Door"`,
		`invalid attribute name " count"`,
		`invalid operation name "approve "`,
		`invalid state name " "`,
		`invalid operation trigger name " approve"`,
		`invalid operation reference " approve"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected %q in diagnostics, got %v", needle, err)
		}
	}
}

func TestValidateRejectsBlankAndPaddedAttributeValueFields(t *testing.T) {
	program := &Program{Models: []Model{{
		ID:          "model_1",
		Name:        "Door",
		Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
		Attributes: []Attribute{
			{ID: "attribute_1", Name: "typed", Type: " int"},
			{ID: "attribute_2", Name: "empty_type", Type: " "},
			{ID: "attribute_3", Name: "defaulted", HasDefault: true, Default: " 1"},
			{ID: "attribute_4", Name: "empty_default", HasDefault: true, Default: "\t"},
		},
		States: []State{{ID: "state_1", Name: "closed", Kind: StateKindState}},
	}}}

	err := Validate(program)
	if err == nil {
		t.Fatal("expected attribute value field diagnostics")
	}
	text := err.Error()
	if diagnostics, ok := err.(Diagnostics); ok {
		var messages []string
		for _, diagnostic := range diagnostics {
			messages = append(messages, diagnostic.Message)
		}
		text = strings.Join(messages, "\n")
	}
	for _, needle := range []string{
		`attribute "typed" type " int" has leading or trailing whitespace`,
		`attribute "empty_type" type is empty`,
		`attribute "defaulted" default " 1" has leading or trailing whitespace`,
		`attribute "empty_default" default is empty`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected %q in diagnostics, got %v", needle, err)
		}
	}
}

func TestValidateRejectsSlashInAttributeOperationAndTriggerNames(t *testing.T) {
	program := &Program{Models: []Model{{
		ID:          "model_1",
		Name:        "Door",
		Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
		Attributes:  []Attribute{{ID: "attribute_1", Name: "count/current"}},
		Operations:  []Operation{{ID: "operation_1", Name: "approve/path"}},
		States: []State{{
			ID:   "state_1",
			Name: "closed",
			Kind: StateKindState,
			Transitions: []Transition{
				{
					OwnerID: "state_1",
					ID:      "transition_1",
					Target:  "../closed",
					Trigger: &Trigger{Kind: TriggerOnSet, Value: "count/current", IsString: true},
				},
				{
					OwnerID: "state_1",
					ID:      "transition_2",
					Target:  "../closed",
					Trigger: &Trigger{Kind: TriggerOnCall, Value: "approve/path", IsString: true},
					Effects: []BehaviorRef{{Kind: BehaviorEffect, Operation: "approve/path"}},
				},
			},
		}},
	}}}

	err := Validate(program)
	if err == nil {
		t.Fatal("expected slash-name diagnostics")
	}
	text := err.Error()
	if diagnostics, ok := err.(Diagnostics); ok {
		var messages []string
		for _, diagnostic := range diagnostics {
			messages = append(messages, diagnostic.Message)
		}
		text = strings.Join(messages, "\n")
	}
	for _, needle := range []string{
		`invalid attribute name "count/current"`,
		`invalid operation name "approve/path"`,
		`invalid attribute trigger name "count/current"`,
		`invalid operation trigger name "approve/path"`,
		`invalid operation reference "approve/path"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected %q in diagnostics, got %v", needle, err)
		}
	}
}

func TestValidateRejectsUnknownAttributeAndOperationTriggers(t *testing.T) {
	program := &Program{Models: []Model{{
		ID:          "model_1",
		Name:        "Door",
		Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
		Attributes:  []Attribute{{ID: "attribute_1", Name: "count"}},
		Operations:  []Operation{{ID: "operation_1", Name: "approve"}},
		States: []State{
			{ID: "state_1", Name: "closed", Kind: StateKindState, Transitions: []Transition{
				{OwnerID: "state_1", ID: "transition_1", Target: "../open", Trigger: &Trigger{Kind: TriggerOnSet, Value: "missing", IsString: true}},
				{OwnerID: "state_1", ID: "transition_2", Target: "../open", Trigger: &Trigger{Kind: TriggerWhen, Value: "missing", IsString: true}},
				{OwnerID: "state_1", ID: "transition_3", Target: "../open", Trigger: &Trigger{Kind: TriggerOnCall, Value: "missing", IsString: true}},
			}},
			{ID: "state_2", Name: "open", Kind: StateKindState},
		},
	}}}
	err := Validate(program)
	if err == nil {
		t.Fatal("expected unknown trigger diagnostics")
	}
	var messages []string
	if diagnostics, ok := err.(Diagnostics); ok {
		for _, diagnostic := range diagnostics {
			messages = append(messages, diagnostic.Message)
		}
	} else {
		messages = append(messages, err.Error())
	}
	text := strings.Join(messages, "\n")
	for _, needle := range []string{
		`attribute trigger "missing" does not reference a model attribute`,
		`operation trigger "missing" does not reference a model operation`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected %q in diagnostics, got %v", needle, err)
		}
	}
}
