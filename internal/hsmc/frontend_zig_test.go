package hsmc

import (
	"context"
	"strings"
	"testing"
)

const zigDoorSource = `const std = @import("std");
const hsm = @import("hsm");
const helper = @import("helper.zig");

pub const opened = "open";

fn record(value: []const u8) []const u8 {
    return value;
}

fn enterClosed(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void {
    _ = ctx;
    _ = inst;
    _ = event;
    _ = record("closed");
    _ = helper;
}

fn canOpen(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) bool {
    _ = ctx;
    _ = inst;
    _ = event;
    return true;
}

fn openedEffect(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void {
    _ = ctx;
    _ = inst;
    _ = event;
}

pub const door_model = hsm.define("Door", .{
    hsm.initial(hsm.target("closed")),
    hsm.state("closed", .{
        hsm.entry(enterClosed),
        hsm.transition(.{
            hsm.on(opened),
            hsm.guard(canOpen),
            hsm.target("../open"),
            hsm.effect(openedEffect),
        }),
    }),
    hsm.state("open", .{}),
});
`

func TestZigFrontendLowersModelAndBehaviorBoundaries(t *testing.T) {
	program, err := NewZigFrontend().Parse(context.Background(), SourceInput{Path: "door.zig", Data: []byte(zigDoorSource)})
	if err != nil {
		t.Fatal(err)
	}
	if program.SourceLanguage != LanguageZig {
		t.Fatalf("source language = %q, want zig", program.SourceLanguage)
	}
	if len(program.Imports) != 1 || program.Imports[0].Path != "helper.zig" || program.Imports[0].Alias != "helper" {
		t.Fatalf("imports = %#v", program.Imports)
	}
	for _, imp := range program.Imports {
		if imp.Path == "hsm" || imp.Path == "std" {
			t.Fatalf("runtime import leaked into mutable imports: %#v", imp)
		}
	}
	if len(program.Imports[0].LocalNames) != 1 || program.Imports[0].LocalNames[0] != "helper" {
		t.Fatalf("import local names = %#v", program.Imports[0].LocalNames)
	}
	if len(program.Globals) != 1 || !strings.Contains(program.Globals[0].Code, "fn record") {
		t.Fatalf("globals = %#v, want only non-behavior helper", program.Globals)
	}
	for _, global := range program.Globals {
		code := strings.TrimSpace(global.Code)
		if code == `const hsm = @import("hsm");` || code == `const std = @import("std");` {
			t.Fatalf("runtime import declaration leaked into globals: %#v", global)
		}
	}
	if len(program.Events) != 1 || program.Events[0].Symbol != "opened" || program.Events[0].Name != "open" {
		t.Fatalf("events = %#v", program.Events)
	}
	if len(program.Models) != 1 {
		t.Fatalf("models = %d, want 1", len(program.Models))
	}
	model := program.Models[0]
	if model.Name != "Door" || model.Initializer == nil || model.Initializer.Target != "closed" {
		t.Fatalf("model = %#v", model)
	}
	if len(model.States) != 2 {
		t.Fatalf("states = %d, want 2", len(model.States))
	}
	closed := model.States[0]
	if closed.Name != "closed" || len(closed.Behaviors) != 1 || len(closed.Transitions) != 1 {
		t.Fatalf("closed state = %#v", closed)
	}
	transition := closed.Transitions[0]
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerOn || transition.Trigger.Values[0].EventSymbol != "opened" {
		t.Fatalf("trigger = %#v", transition.Trigger)
	}
	if transition.Guard == nil || len(transition.Effects) != 1 || transition.Target != "../open" {
		t.Fatalf("transition = %#v", transition)
	}
	if len(program.Behaviors) != 3 {
		t.Fatalf("behaviors = %d, want 3", len(program.Behaviors))
	}
	for _, behavior := range program.Behaviors {
		if behavior.SourceLanguage != LanguageZig || behavior.OwnerID == "" || behavior.Kind == "" || behavior.Body == "" {
			t.Fatalf("incomplete behavior = %#v", behavior)
		}
	}
	if !strings.Contains(program.Behaviors[0].Body, `record("closed")`) {
		t.Fatalf("entry body = %q", program.Behaviors[0].Body)
	}
	if !strings.Contains(program.Behaviors[1].Body, "return true;") {
		t.Fatalf("guard body = %q", program.Behaviors[1].Body)
	}
	if strings.Contains(program.Behaviors[2].Body, "openedEffect(ctx, inst, event)") {
		t.Fatalf("effect body kept only wrapper call: %q", program.Behaviors[2].Body)
	}
}

func TestZigFrontendRejectsUnknownHSMDslPartials(t *testing.T) {
	source := `const runtime = @import("hsm");

pub const door_model = runtime.define("Door", .{
    runtime.unknownModelPartial(),
    runtime.initial(runtime.unknownInitialPartial(), runtime.target("closed")),
    runtime.state("closed", .{
        runtime.unknownStatePartial(),
        runtime.transition(.{
            runtime.unknownTransitionPartial(),
            runtime.target("../open"),
        }),
    }),
});
`
	_, err := NewZigFrontend().Parse(context.Background(), SourceInput{Path: "unknown.zig", Data: []byte(source)})
	if err == nil {
		t.Fatal("expected unknown hsm partial diagnostics")
	}
	for _, want := range []string{
		`unknown or unsupported hsm.unknownModelPartial partial in model context`,
		`unknown or unsupported hsm.unknownInitialPartial partial in initial context`,
		`unknown or unsupported hsm.unknownStatePartial partial in state context`,
		`unknown or unsupported hsm.unknownTransitionPartial partial in transition context`,
	} {
		if !diagnosticsContain(err, want) {
			t.Fatalf("diagnostics = %v, want %q", err, want)
		}
	}
}

func TestCompilerCompilesZigToZig(t *testing.T) {
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.zig", Data: []byte(zigDoorSource)}, CompileOptions{
		From:    LanguageZig,
		To:      LanguageZig,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if program.SourceLanguage != LanguageZig {
		t.Fatalf("source language = %q, want zig", program.SourceLanguage)
	}
	text := string(output)
	for _, needle := range []string{
		`const hsm = @import("hsm");`,
		`const helper = @import("helper.zig");`,
		`pub const opened = "open";`,
		`pub const door_model = hsm.define("Door", .{`,
		`hsm.on(opened),`,
		`hsm.guard(guard_1),`,
		`_ = record("closed");`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Zig output missing %q:\n%s", needle, text)
		}
	}
}

const zigRichSource = `const hsm = @import("hsm");

fn approve(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) bool {
    _ = ctx;
    _ = inst;
    _ = event;
    return true;
}

fn deadline(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) i64 {
    _ = ctx;
    _ = inst;
    _ = event;
    return 0;
}

pub const rich_door_model = hsm.define("RichDoor", .{
    hsm.attribute("count", i32, 1),
    hsm.operation("approve", approve),
    hsm.initial(hsm.target("closed")),
    hsm.state("closed", .{
        hsm.transition(.{
            hsm.onCall("approve"),
            hsm.target("../open"),
        }),
        hsm.transition(.{
            hsm.onSet("count"),
            hsm.target("../open"),
        }),
        hsm.transition(.{
            hsm.at(deadline),
            hsm.target("../open"),
        }),
        hsm.transition(.{
            hsm.when(approve),
            hsm.target("../open"),
        }),
    }),
    hsm.state("open", .{}),
});
`

func TestZigFrontendLowersAttributesOperationsAndExtendedTriggers(t *testing.T) {
	program, err := NewZigFrontend().Parse(context.Background(), SourceInput{Path: "rich.zig", Data: []byte(zigRichSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 {
		t.Fatalf("models = %d, want 1", len(program.Models))
	}
	model := program.Models[0]
	if len(model.Attributes) != 1 || model.Attributes[0].Name != "count" || model.Attributes[0].Type != "i32" || model.Attributes[0].Default != "1" {
		t.Fatalf("attributes = %#v", model.Attributes)
	}
	if len(model.Operations) != 1 || model.Operations[0].Name != "approve" || model.Operations[0].Behavior == nil {
		t.Fatalf("operations = %#v", model.Operations)
	}
	if len(model.States) == 0 || len(model.States[0].Transitions) != 4 {
		t.Fatalf("state transitions = %#v", model.States)
	}
	transitions := model.States[0].Transitions
	if transitions[0].Trigger == nil || transitions[0].Trigger.Kind != TriggerOnCall || transitions[0].Trigger.Value != "approve" {
		t.Fatalf("OnCall trigger = %#v", transitions[0].Trigger)
	}
	if transitions[1].Trigger == nil || transitions[1].Trigger.Kind != TriggerOnSet || transitions[1].Trigger.Value != "count" {
		t.Fatalf("OnSet trigger = %#v", transitions[1].Trigger)
	}
	if transitions[2].Trigger == nil || transitions[2].Trigger.Kind != TriggerAt || transitions[2].Trigger.Expr == nil {
		t.Fatalf("At trigger = %#v", transitions[2].Trigger)
	}
	if transitions[3].Trigger == nil || transitions[3].Trigger.Kind != TriggerWhen || transitions[3].Trigger.Expr == nil {
		t.Fatalf("When trigger = %#v", transitions[3].Trigger)
	}
	if len(program.Behaviors) != 3 {
		t.Fatalf("behaviors = %d, want operation plus At and When triggers", len(program.Behaviors))
	}
}

func TestCompilerConvertsZigExtendedSourceToGo(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "rich.zig", Data: []byte(zigRichSource)}, CompileOptions{
		From:    LanguageZig,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`// Type: i32`,
		`hsm.Attribute("count", 1)`,
		`hsm.Operation("approve", Operation1)`,
		`hsm.OnCall("approve")`,
		`hsm.OnSet("count")`,
		`hsm.At(Trigger1)`,
		`hsm.When(Trigger2)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
}

func TestCompilerConvertsZigExtendedSourceToZig(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "rich.zig", Data: []byte(zigRichSource)}, CompileOptions{
		From:    LanguageZig,
		To:      LanguageZig,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`hsm.onCall("approve"),`,
		`hsm.onSet("count"),`,
		"hsm.at(trigger_1),",
		"// when trigger lowered to guard for hsm.zig",
		"hsm.guard(trigger_2),",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Zig output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{
		"Unsupported when trigger",
		"on_call trigger lowered to event trigger",
		"on_set trigger lowered to event trigger",
		"at trigger lowered to after trigger",
		`hsm.on("approve")`,
		`hsm.on("count")`,
		`hsm.after(trigger_1)`,
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Zig output used stale trigger fallback %q:\n%s", forbidden, text)
		}
	}
}

func TestZigFrontendReingestsGeneratedBehaviorFunctions(t *testing.T) {
	source := `const hsm = @import("hsm");

fn entry_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void {
    _ = ctx;
    _ = event;
    inst.set("entered", true);
}

fn guard_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) bool {
    _ = ctx;
    _ = inst;
    _ = event;
    return true;
}

pub const door_model = hsm.define("Door", .{
    hsm.initial(hsm.target("closed")),
    hsm.state("closed", .{
        hsm.entry(entry_1),
        hsm.transition(.{
            hsm.on("open"),
            hsm.guard(guard_1),
            hsm.target("../open"),
        }),
    }),
    hsm.state("open", .{}),
});
`
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "generated.zig", Data: []byte(source)}, CompileOptions{
		From:    LanguageZig,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("generated behavior functions should not remain globals: %#v", program.Globals)
	}
	if len(program.Behaviors) != 2 {
		t.Fatalf("behaviors = %d, want 2", len(program.Behaviors))
	}
	var entryBody, guardBody string
	for _, behavior := range program.Behaviors {
		switch behavior.Kind {
		case BehaviorEntry:
			entryBody = behavior.Body
		case BehaviorGuard:
			guardBody = behavior.Body
		}
	}
	if !strings.Contains(entryBody, `inst.set("entered", true);`) {
		t.Fatalf("entry body = %q", entryBody)
	}
	if !strings.Contains(guardBody, "return true;") {
		t.Fatalf("guard body = %q", guardBody)
	}
	text := string(output)
	for _, needle := range []string{
		`inst.set("entered", true);`,
		`return true;`,
		`hsm.Entry(Entry1)`,
		`hsm.Guard(Guard1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original zig global") || strings.Contains(text, "entry_1(ctx, inst, event)") || strings.Contains(text, "guard_1(ctx, inst, event)") {
		t.Fatalf("Go output leaked generated Zig functions as globals or callable wrappers:\n%s", text)
	}
}

func TestZigFrontendReingestsGeneratedOperationFunction(t *testing.T) {
	source := `const hsm = @import("hsm");

fn operation_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void {
    _ = ctx;
    _ = inst;
    _ = event;
    hsm.keepAlive("approve");
}

pub const door_model = hsm.define("Door", .{
    hsm.operation("approve", operation_1),
    hsm.initial(hsm.target("closed")),
    hsm.state("closed", .{
        hsm.transition(.{
            hsm.onCall("approve"),
            hsm.target("../open"),
        }),
    }),
    hsm.state("open", .{}),
});
`
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "generated.zig", Data: []byte(source)}, CompileOptions{
		From:    LanguageZig,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("generated operation function should not remain global: %#v", program.Globals)
	}
	if len(program.Behaviors) != 1 || program.Behaviors[0].Kind != BehaviorOperation || !strings.Contains(program.Behaviors[0].Body, `hsm.keepAlive("approve");`) {
		t.Fatalf("operation behavior = %#v", program.Behaviors)
	}
	text := string(output)
	for _, needle := range []string{
		`hsm.Operation("approve", Operation1)`,
		`hsm.keepAlive("approve");`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original zig global") || strings.Contains(text, "operation_1(ctx, inst, event)") {
		t.Fatalf("Go output leaked generated Zig operation function:\n%s", text)
	}
}
