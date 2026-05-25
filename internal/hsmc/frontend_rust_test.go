package hsmc

import (
	"context"
	"strings"
	"testing"
)

const rustDoorSource = `use hsm::*;
use crate::helpers::record;

fn entered(ctx: &mut hsm::Context, instance: &mut hsm::Instance, event: &hsm::Event) {
    record("entered");
}

fn allowed(ctx: &mut hsm::Context, instance: &mut hsm::Instance, event: &hsm::Event) -> bool {
    true
}

const DOOR: hsm::Model = define!("Door",
    initial!(target!("closed")),
    state!("closed",
        entry!(entered),
        transition!(on!("open"), guard!(allowed), target!("../open"))
    ),
    state!("open")
);
`

const rustGeneratedDoorSource = `use hsm::*;

pub struct HsmcInstance;

impl hsm::Instance for HsmcInstance {
    fn as_any(&self) -> &dyn std::any::Any { self }
    fn as_any_mut(&mut self) -> &mut dyn std::any::Any { self }
}

fn entry_1(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) -> std::pin::Pin<Box<dyn std::future::Future<Output = ()> + Send>> {
    let _ = (ctx, instance, event);
    Box::pin(async {})
}

fn guard_1(ctx: &hsm::Context, instance: &HsmcInstance, event: &hsm::Event) -> bool {
    let _ = (ctx, instance, event);
    true
}

pub fn door_model() -> hsm::Model<HsmcInstance> {
    hsm::define!("Door",
        hsm::initial!(hsm::target!("closed")),
        hsm::state!("closed",
            hsm::entry!(entry_1),
            hsm::transition!(hsm::on!("open"), hsm::guard!(guard_1), hsm::target!("../open"))
        ),
        hsm::state!("open")
    )
}
`

func TestRustFrontendLowersMacroModel(t *testing.T) {
	program, err := NewRustFrontend().Parse(context.Background(), SourceInput{Path: "door.rs", Data: []byte(rustDoorSource)})
	if err != nil {
		t.Fatal(err)
	}
	if program.SourceLanguage != LanguageRust {
		t.Fatalf("source language = %q, want rust", program.SourceLanguage)
	}
	if len(program.Imports) != 1 || program.Imports[0].Path != "crate::helpers::record" {
		t.Fatalf("imports = %#v", program.Imports)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("globals = %#v, want referenced behavior functions removed", program.Globals)
	}
	if len(program.Models) != 1 {
		t.Fatalf("models = %d, want 1", len(program.Models))
	}
	model := program.Models[0]
	if model.Name != "Door" || model.Initializer == nil || model.Initializer.Target != "closed" {
		t.Fatalf("model = %#v", model)
	}
	if len(model.States) != 2 || model.States[0].Name != "closed" || model.States[1].Name != "open" {
		t.Fatalf("states = %#v", model.States)
	}
	closed := model.States[0]
	if len(closed.Behaviors) != 1 || closed.Behaviors[0].ID == "" || closed.Behaviors[0].Kind != BehaviorEntry {
		t.Fatalf("closed behaviors = %#v", closed.Behaviors)
	}
	if len(closed.Transitions) != 1 {
		t.Fatalf("closed transitions = %#v", closed.Transitions)
	}
	transition := closed.Transitions[0]
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerOn || transition.Trigger.Value != "open" || !transition.Trigger.IsString {
		t.Fatalf("trigger = %#v", transition.Trigger)
	}
	if transition.Target != "../open" {
		t.Fatalf("target = %q, want ../open", transition.Target)
	}
	if transition.Guard == nil || transition.Guard.ID == "" || transition.Guard.Kind != BehaviorGuard {
		t.Fatalf("guard = %#v", transition.Guard)
	}
	if len(program.Behaviors) != 2 {
		t.Fatalf("behaviors = %#v, want entry and guard", program.Behaviors)
	}
	entry := program.Behaviors[0]
	guard := program.Behaviors[1]
	if entry.SourceLanguage != LanguageRust || !strings.Contains(entry.Body, `record("entered");`) {
		t.Fatalf("entry behavior = %#v", entry)
	}
	if guard.SourceLanguage != LanguageRust || !strings.Contains(guard.Body, "true") {
		t.Fatalf("guard behavior = %#v", guard)
	}
}

func TestRustFrontendLowersCanonicalPascalCaseDSL(t *testing.T) {
	source := `use hsm::*;

const DOOR: hsm::Model = hsm::Define!("Door",
    hsm::Initial!(hsm::Target!("closed")),
    hsm::State!("closed",
        hsm::Transition!(hsm::OnSet!("open"), hsm::Target!("../open"))
    ),
    hsm::State!("open"),
    hsm::ShallowHistory!("recent", hsm::Target!("closed"))
);
`
	program, err := NewRustFrontend().Parse(context.Background(), SourceInput{Path: "door.rs", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("PascalCase model declaration leaked into globals: %#v", program.Globals)
	}
	if len(program.Models) != 1 {
		t.Fatalf("models = %#v, want one model", program.Models)
	}
	model := program.Models[0]
	if model.Name != "Door" || model.Initializer == nil || model.Initializer.Target != "closed" {
		t.Fatalf("model = %#v", model)
	}
	if len(model.States) != 3 {
		t.Fatalf("states = %#v, want closed, open, and history", model.States)
	}
	if model.States[2].Kind != StateKindShallowHistory {
		t.Fatalf("history state = %#v, want shallow history", model.States[2])
	}
	if len(model.States[0].Transitions) != 1 || model.States[0].Transitions[0].Trigger == nil ||
		model.States[0].Transitions[0].Trigger.Kind != TriggerOnSet {
		t.Fatalf("PascalCase OnSet transition not lowered: %#v", model.States[0].Transitions)
	}
}

const rustGroupedUseSource = `use hsm::*;
use crate::helpers::{record, normalize as norm};

fn entry_1(ctx: &mut hsm::Context, instance: &mut hsm::Instance, event: &hsm::Event) {
    record(norm("entered"));
}

const DOOR: hsm::Model = define!("Door",
    initial!(target!("closed")),
    state!("closed",
        entry!(entry_1)
    )
);
`

func TestRustFrontendPreservesGroupedUseSpecifiers(t *testing.T) {
	program, err := NewRustFrontend().Parse(context.Background(), SourceInput{Path: "door.rs", Data: []byte(rustGroupedUseSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 1 {
		t.Fatalf("imports = %#v", program.Imports)
	}
	imp := program.Imports[0]
	if imp.Path != "crate::helpers" || imp.Language != LanguageRust {
		t.Fatalf("import metadata = %#v", imp)
	}
	if len(imp.Specifiers) != 2 || imp.Specifiers[0].Name != "record" || imp.Specifiers[1].Name != "normalize" || imp.Specifiers[1].Alias != "norm" {
		t.Fatalf("import specifiers = %#v", imp.Specifiers)
	}
	if got := strings.Join(imp.LocalNames, ","); got != "record,norm" {
		t.Fatalf("local names = %q, want record,norm", got)
	}
}

func TestRustFrontendRejectsUnknownHSMDslPartials(t *testing.T) {
	source := `use hsm::*;
use hsm::{unknown_transition_partial};

const DOOR: hsm::Model = hsm::define!("Door",
    hsm::unknown_model_partial!(),
    hsm::initial!(hsm::unknown_initial_partial!(), hsm::target!("closed")),
    hsm::state!("closed",
        hsm::unknown_state_partial!(),
        hsm::transition!(
            unknown_transition_partial!(),
            hsm::target!("../open")
        )
    )
);
`
	_, err := NewRustFrontend().Parse(context.Background(), SourceInput{Path: "door.rs", Data: []byte(source)})
	if err == nil {
		t.Fatal("expected unknown hsm partial diagnostics")
	}
	for _, want := range []string{
		`unknown or unsupported hsm::unknown_model_partial! partial in model context`,
		`unknown or unsupported hsm::unknown_initial_partial! partial in initial context`,
		`unknown or unsupported hsm::unknown_state_partial! partial in state context`,
		`unknown or unsupported hsm::unknown_transition_partial! partial in transition context`,
	} {
		if !diagnosticsContain(err, want) {
			t.Fatalf("diagnostics = %v, want %q", err, want)
		}
	}
}

func TestRustSourceCompilesToRustAndTypeScript(t *testing.T) {
	rustOutput, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.rs", Data: []byte(rustDoorSource)}, CompileOptions{
		From:    LanguageRust,
		To:      LanguageRust,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	rustText := string(rustOutput)
	for _, needle := range []string{
		"use crate::helpers::record;",
		"fn entry_1(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) -> Pin<Box<dyn Future<Output = ()> + Send>>",
		`record("entered");`,
		"Box::pin(async {})",
		`hsm::transition!(hsm::on!("open"), hsm::guard!(guard_1), hsm::target!("../open")),`,
	} {
		if !strings.Contains(rustText, needle) {
			t.Fatalf("Rust output missing %q:\n%s", needle, rustText)
		}
	}

	tsOutput, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.rs", Data: []byte(rustDoorSource)}, CompileOptions{
		From:    LanguageRust,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	tsText := string(tsOutput)
	for _, needle := range []string{
		"function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void",
		"// Original rust behavior entry_1 preserved for manual porting:",
		`// record("entered");`,
		"function guard1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): boolean",
		"// true",
	} {
		if !strings.Contains(tsText, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, tsText)
		}
	}
}

func TestRustFrontendReingestsGeneratedBehaviorFunctions(t *testing.T) {
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "generated.rs", Data: []byte(rustGeneratedDoorSource)}, CompileOptions{
		From:    LanguageRust,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("generated compiler scaffold should not remain mutable globals: %#v", program.Globals)
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
	if !strings.Contains(entryBody, `Box::pin(async {})`) {
		t.Fatalf("entry body = %q", entryBody)
	}
	if !strings.Contains(guardBody, "true") {
		t.Fatalf("guard body = %q", guardBody)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original rust behavior entry_1 preserved for manual porting:",
		`Box::pin(async {})`,
		"// Original rust behavior guard_1 preserved for manual porting:",
		"true",
		"hsm.Entry(Entry1)",
		"hsm.Guard(Guard1)",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original rust global") || strings.Contains(text, "door_model") {
		t.Fatalf("generated Rust functions leaked as globals:\n%s", text)
	}
}

func TestCompilerReemitsGeneratedRustWithoutDuplicateScaffoldOrDefaultReturns(t *testing.T) {
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "generated.rs", Data: []byte(rustGeneratedDoorSource)}, CompileOptions{
		From:    LanguageRust,
		To:      LanguageRust,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("generated compiler scaffold should not remain mutable globals: %#v", program.Globals)
	}
	text := string(output)
	if strings.Count(text, "pub struct HsmcInstance;") != 1 || strings.Count(text, "impl hsm::Instance for HsmcInstance") != 1 {
		t.Fatalf("Rust output duplicated generated instance scaffold:\n%s", text)
	}
	if strings.Count(text, "Box::pin(async {})") != 1 {
		t.Fatalf("Rust output should preserve the source future expression without adding a default:\n%s", text)
	}
	if strings.Contains(text, "\ttrue\n\tfalse") {
		t.Fatalf("Rust output appended a default guard return after a preserved value expression:\n%s", text)
	}
}

func TestRustFrontendReingestsGeneratedOperationFunction(t *testing.T) {
	source := `use hsm::*;

struct HsmcInstance;

fn operation_1(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) {
    instance.set("approved", true);
}

pub fn door_model() -> hsm::Model<HsmcInstance> {
    hsm::define!("Door",
        hsm::operation!("approve", operation_1),
        hsm::initial!(hsm::target!("closed")),
        hsm::state!("closed",
            hsm::transition!(hsm::on_call!("approve"), hsm::target!("../open"))
        ),
        hsm::state!("open")
    )
}
`
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "generated_operation.rs", Data: []byte(source)}, CompileOptions{
		From:    LanguageRust,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("generated compiler scaffold should not remain mutable globals: %#v", program.Globals)
	}
	if len(program.Behaviors) != 1 || program.Behaviors[0].Kind != BehaviorOperation {
		t.Fatalf("behaviors = %#v, want one operation behavior", program.Behaviors)
	}
	if !strings.Contains(program.Behaviors[0].Body, `instance.set("approved", true);`) {
		t.Fatalf("operation body = %q", program.Behaviors[0].Body)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original rust behavior operation_1 preserved for manual porting:",
		`// instance.set("approved", true);`,
		`hsm.Operation("approve", Operation1)`,
		`hsm.OnCall("approve")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original rust global") || strings.Contains(text, "door_model") {
		t.Fatalf("generated Rust operation function leaked as global:\n%s", text)
	}
}

const rustRichSource = `use hsm::*;

fn approve(ctx: &hsm::Context, instance: &hsm::Instance, event: &hsm::Event) -> bool {
    true
}

fn deadline(ctx: &hsm::Context, instance: &hsm::Instance, event: &hsm::Event) -> std::time::Duration {
    std::time::Duration::from_secs(1)
}

const RICH_DOOR: hsm::Model = define!("RichDoor",
    attribute!("count", i32, 1),
    operation!("approve", approve),
    initial!(target!("closed")),
    state!("closed",
        transition!(on_call!("approve"), target!("../open")),
        transition!(on_set!("count"), target!("../open")),
        transition!(at!(deadline), target!("../open")),
        transition!(when!(approve), target!("../open"))
    ),
    state!("open")
);
`

func TestRustFrontendLowersAttributesOperationsAndExtendedTriggers(t *testing.T) {
	program, err := NewRustFrontend().Parse(context.Background(), SourceInput{Path: "rich.rs", Data: []byte(rustRichSource)})
	if err != nil {
		t.Fatal(err)
	}
	model := program.Models[0]
	if len(model.Attributes) != 1 || model.Attributes[0].Name != "count" || model.Attributes[0].Type != "i32" || model.Attributes[0].Default != "1" {
		t.Fatalf("attributes = %#v", model.Attributes)
	}
	if len(model.Operations) != 1 || model.Operations[0].Name != "approve" || model.Operations[0].Behavior == nil {
		t.Fatalf("operations = %#v", model.Operations)
	}
	transitions := model.States[0].Transitions
	if len(transitions) != 4 {
		t.Fatalf("transitions = %#v", transitions)
	}
	if transitions[0].Trigger == nil || transitions[0].Trigger.Kind != TriggerOnCall || transitions[0].Trigger.Value != "approve" {
		t.Fatalf("on-call trigger = %#v", transitions[0].Trigger)
	}
	if transitions[1].Trigger == nil || transitions[1].Trigger.Kind != TriggerOnSet || transitions[1].Trigger.Value != "count" {
		t.Fatalf("on-set trigger = %#v", transitions[1].Trigger)
	}
	if transitions[2].Trigger == nil || transitions[2].Trigger.Kind != TriggerAt || transitions[2].Trigger.Expr == nil {
		t.Fatalf("at trigger = %#v", transitions[2].Trigger)
	}
	if transitions[3].Trigger == nil || transitions[3].Trigger.Kind != TriggerWhen || transitions[3].Trigger.Expr == nil {
		t.Fatalf("when trigger = %#v", transitions[3].Trigger)
	}
	if len(program.Behaviors) != 3 {
		t.Fatalf("behaviors = %d, want operation plus At and When triggers", len(program.Behaviors))
	}
}

func TestCompilerConvertsRustExtendedSourceToGo(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "rich.rs", Data: []byte(rustRichSource)}, CompileOptions{
		From:    LanguageRust,
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
