package hsmc

import (
	"context"
	"strings"
	"testing"
)

const pythonDoorSource = `import datetime
import hsm
from helpers import record as rec

class Door(hsm.Instance):
    pass

async def entered(ctx: hsm.Context, instance: Door, event: hsm.Event) -> None:
    rec("entered")

async def approve(ctx: hsm.Context, instance: Door, event: hsm.Event) -> bool:
    return True

async def deadline(ctx: hsm.Context, instance: Door, event: hsm.Event) -> datetime.datetime:
    return datetime.datetime.now()

model = hsm.Define(
    "Door",
    hsm.Attribute("count", int, 1),
    hsm.Operation("approve", approve),
    hsm.Initial(hsm.Target("closed")),
    hsm.State(
        "closed",
        hsm.Entry(entered),
        hsm.Defer("knock"),
        hsm.Transition(
            hsm.OnCall("approve"),
            hsm.Target("../open"),
            hsm.Effect(entered),
        ),
        hsm.Transition(
            hsm.At(deadline),
            hsm.Target("../open"),
        ),
    ),
    hsm.State("open"),
)
`

func TestPythonFrontendLowersModel(t *testing.T) {
	program, err := NewPythonFrontend().Parse(context.Background(), SourceInput{Path: "door.py", Data: []byte(pythonDoorSource)})
	if err != nil {
		t.Fatal(err)
	}
	if program.SourceLanguage != LanguagePython {
		t.Fatalf("source language = %q", program.SourceLanguage)
	}
	if len(program.Imports) != 2 {
		t.Fatalf("imports = %#v", program.Imports)
	}
	if len(program.Globals) != 1 {
		t.Fatalf("globals = %#v", program.Globals)
	}
	if !strings.Contains(program.Globals[0].Code, "class Door") {
		t.Fatalf("globals = %#v, want only class global", program.Globals)
	}
	model := program.Models[0]
	if model.Name != "Door" || model.Initializer == nil || model.Initializer.Target != "closed" {
		t.Fatalf("model = %#v", model)
	}
	if len(model.Attributes) != 1 || model.Attributes[0].Type != "int" || model.Attributes[0].Default != "1" {
		t.Fatalf("attributes = %#v", model.Attributes)
	}
	if len(model.Operations) != 1 || model.Operations[0].Behavior == nil {
		t.Fatalf("operations = %#v", model.Operations)
	}
	state := model.States[0]
	if len(state.Behaviors) != 1 || len(state.Defers) != 1 || state.Defers[0] != "knock" {
		t.Fatalf("state = %#v", state)
	}
	if len(state.Transitions) != 2 {
		t.Fatalf("transitions = %#v", state.Transitions)
	}
	if state.Transitions[1].Trigger == nil || state.Transitions[1].Trigger.Kind != TriggerAt || state.Transitions[1].Trigger.Expr == nil {
		t.Fatalf("At trigger = %#v", state.Transitions[1].Trigger)
	}
}

const pythonSnakeCaseAliasSource = `import hsm

go_event = hsm.event("go")

model = hsm.define(
    "Door",
    hsm.initial(hsm.target("closed")),
    hsm.state(
        "closed",
        hsm.entry(lambda ctx, instance, event: instance.set("entered", True)),
        hsm.transition(
            hsm.on(go_event.name),
            hsm.guard(lambda ctx, instance, event: True),
            hsm.target("../open"),
        ),
        hsm.transition(
            hsm.on_set("count"),
            hsm.target("../open"),
        ),
    ),
    hsm.state("open"),
)
`

func TestPythonFrontendLowersSnakeCaseAliases(t *testing.T) {
	program, err := NewPythonFrontend().Parse(context.Background(), SourceInput{Path: "door.py", Data: []byte(pythonSnakeCaseAliasSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Events) != 1 || program.Events[0].Symbol != "go_event" || program.Events[0].Name != "go" {
		t.Fatalf("events = %#v", program.Events)
	}
	if len(program.Models) != 1 {
		t.Fatalf("models = %#v", program.Models)
	}
	model := program.Models[0]
	if model.Name != "Door" || model.Initializer == nil || model.Initializer.Target != "closed" {
		t.Fatalf("model = %#v", model)
	}
	if len(model.States) != 2 {
		t.Fatalf("states = %#v", model.States)
	}
	closed := model.States[0]
	if len(closed.Behaviors) != 1 || len(closed.Transitions) != 2 {
		t.Fatalf("closed state = %#v", closed)
	}
	if closed.Transitions[0].Trigger == nil || closed.Transitions[0].Trigger.Kind != TriggerOn || len(closed.Transitions[0].Trigger.Values) != 1 || closed.Transitions[0].Trigger.Values[0].EventSymbol != "go_event" {
		t.Fatalf("on trigger = %#v", closed.Transitions[0].Trigger)
	}
	if closed.Transitions[0].Guard == nil {
		t.Fatalf("guard not lowered: %#v", closed.Transitions[0])
	}
	if closed.Transitions[1].Trigger == nil || closed.Transitions[1].Trigger.Kind != TriggerOnSet || closed.Transitions[1].Trigger.Value != "count" {
		t.Fatalf("on_set trigger = %#v", closed.Transitions[1].Trigger)
	}
	if len(program.Behaviors) != 2 {
		t.Fatalf("behaviors = %#v", program.Behaviors)
	}
}

func TestPythonFrontendRejectsUnknownHSMDslPartials(t *testing.T) {
	source := `import hsm as sf
from hsm import UnknownTransitionPartial as unknown_transition_partial

model = sf.Define(
    "Door",
    sf.UnknownModelPartial(),
    sf.Initial(sf.UnknownInitialPartial(), sf.Target("closed")),
    sf.State(
        "closed",
        sf.UnknownStatePartial(),
        sf.Transition(
            unknown_transition_partial(),
            sf.Target("../open"),
        ),
    ),
)
`
	_, err := NewPythonFrontend().Parse(context.Background(), SourceInput{Path: "door.py", Data: []byte(source)})
	if err == nil {
		t.Fatal("expected unknown hsm partial diagnostics")
	}
	for _, want := range []string{
		`unknown or unsupported hsm.UnknownModelPartial partial in model context`,
		`unknown or unsupported hsm.UnknownInitialPartial partial in initial context`,
		`unknown or unsupported hsm.UnknownStatePartial partial in state context`,
		`unknown or unsupported hsm.unknown_transition_partial partial in transition context`,
	} {
		if !diagnosticsContain(err, want) {
			t.Fatalf("diagnostics = %v, want %q", err, want)
		}
	}
}

const pythonRenamedImportAliasSource = `from hsm import (
    Define as HDefine,
    Initial as HInitial,
    Target as HTarget,
    State as HState,
    Entry as HEntry,
    Transition as HTransition,
    On as HOn,
    Guard as HGuard,
    Event as HEvent,
)

go_event = HEvent("go")

model = HDefine(
    "Door",
    HInitial(HTarget("closed")),
    HState(
        "closed",
        HEntry(lambda ctx, instance, event: instance.set("entered", True)),
        HTransition(
            HOn(go_event.name),
            HGuard(lambda ctx, instance, event: True),
            HTarget("../open"),
        ),
    ),
    HState("open"),
)
`

func TestPythonFrontendLowersRenamedHSMImportAliases(t *testing.T) {
	program, err := NewPythonFrontend().Parse(context.Background(), SourceInput{Path: "door.py", Data: []byte(pythonRenamedImportAliasSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 0 {
		t.Fatalf("hsm named import aliases should be compiler-owned, got %#v", program.Imports)
	}
	if len(program.Events) != 1 || program.Events[0].Symbol != "go_event" || program.Events[0].Name != "go" {
		t.Fatalf("events = %#v", program.Events)
	}
	if len(program.Models) != 1 {
		t.Fatalf("models = %#v", program.Models)
	}
	model := program.Models[0]
	if model.Name != "Door" || model.Initializer == nil || model.Initializer.Target != "closed" {
		t.Fatalf("model = %#v", model)
	}
	if len(model.States) != 2 {
		t.Fatalf("states = %#v", model.States)
	}
	closed := model.States[0]
	if len(closed.Behaviors) != 1 || len(closed.Transitions) != 1 {
		t.Fatalf("closed state = %#v", closed)
	}
	transition := closed.Transitions[0]
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerOn || len(transition.Trigger.Values) != 1 || transition.Trigger.Values[0].EventSymbol != "go_event" {
		t.Fatalf("trigger = %#v", transition.Trigger)
	}
	if transition.Guard == nil {
		t.Fatalf("guard not lowered: %#v", transition)
	}
	if len(program.Behaviors) != 2 {
		t.Fatalf("behaviors = %#v", program.Behaviors)
	}
}

func TestPythonSourceCompilesToPythonAndGo(t *testing.T) {
	pythonOutput, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.py", Data: []byte(pythonDoorSource)}, CompileOptions{
		From:    LanguagePython,
		To:      LanguagePython,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	pythonText := string(pythonOutput)
	for _, needle := range []string{
		`async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:`,
		`rec("entered")`,
		`async def operation_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:`,
		`return True`,
		`async def trigger_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> datetime.datetime:`,
		`return datetime.datetime.now()`,
		`# Type: int`,
		`hsm.Attribute("count", 1)`,
		`door_model = hsm.Define(`,
		`hsm.At(trigger_1)`,
	} {
		if !strings.Contains(pythonText, needle) {
			t.Fatalf("Python output missing %q:\n%s", needle, pythonText)
		}
	}

	goOutput, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.py", Data: []byte(pythonDoorSource)}, CompileOptions{
		From:    LanguagePython,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	goText := string(goOutput)
	for _, needle := range []string{
		`func Entry1(ctx context.Context, instance hsm.Instance, event hsm.Event)`,
		`// Original python behavior entry_1 preserved for manual porting:`,
		`// rec("entered")`,
		`hsm.Operation("approve", Operation1)`,
		`hsm.At(Trigger1)`,
	} {
		if !strings.Contains(goText, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, goText)
		}
	}

	tsOutput, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.py", Data: []byte(pythonDoorSource)}, CompileOptions{
		From:    LanguagePython,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	tsText := string(tsOutput)
	for _, needle := range []string{
		`function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void`,
		`// Original python behavior entry_1 preserved for manual porting:`,
		`// rec("entered")`,
		`hsm.Operation("approve", operation1)`,
		`hsm.At(trigger1)`,
	} {
		if !strings.Contains(tsText, needle) {
			t.Fatalf("TS output missing %q:\n%s", needle, tsText)
		}
	}
}

func TestPythonFrontendLowersNamedBehaviorFunctions(t *testing.T) {
	source := `import hsm

async def entered(ctx, instance, event):
    instance.set("entered", True)

async def allowed(ctx, instance, event):
    return event.name == "open"

def approve(instance):
    return True

def helper(value):
    return value

door_model = hsm.Define(
    "Door",
    hsm.Operation("approve", approve),
    hsm.Initial(hsm.Target("closed")),
    hsm.State(
        "closed",
        hsm.Entry(entered),
        hsm.Transition(hsm.On("open"), hsm.Guard(allowed), hsm.Target("../open")),
    ),
    hsm.State("open"),
)
`
	program, err := NewPythonFrontend().Parse(context.Background(), SourceInput{Path: "named.py", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want operation helper and ordinary helper", program.Globals)
	}
	if !strings.Contains(program.Globals[0].Code+program.Globals[1].Code, "def approve") ||
		!strings.Contains(program.Globals[0].Code+program.Globals[1].Code, "def helper") {
		t.Fatalf("globals = %#v, want non-HSM-callback functions preserved", program.Globals)
	}
	if len(program.Behaviors) != 3 {
		t.Fatalf("behaviors = %d, want operation reference plus entry and guard", len(program.Behaviors))
	}
	var entryBody, guardBody, operationCode string
	for _, behavior := range program.Behaviors {
		switch behavior.Kind {
		case BehaviorEntry:
			entryBody = behavior.Body
		case BehaviorGuard:
			guardBody = behavior.Body
		case BehaviorOperation:
			operationCode = behavior.Code
		}
		if behavior.Code == "entered" || behavior.Code == "allowed" {
			t.Fatalf("behavior kept only identifier instead of resolved function body: %#v", behavior)
		}
	}
	if !strings.Contains(entryBody, `instance.set("entered", True)`) {
		t.Fatalf("entry body = %q", entryBody)
	}
	if !strings.Contains(guardBody, `return event.name == "open"`) {
		t.Fatalf("guard body = %q", guardBody)
	}
	if operationCode != "approve" {
		t.Fatalf("operation code = %q, want unresolved operation reference", operationCode)
	}
}

func TestPythonFrontendReingestsGeneratedBehaviorFunctions(t *testing.T) {
	source := `import hsm

async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:
    instance.set("entered", True)

async def guard_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> bool:
    return True

door_model = hsm.Define(
    "Door",
    hsm.Initial(hsm.Target("closed")),
    hsm.State(
        "closed",
        hsm.Entry(entry_1),
        hsm.Transition(hsm.On("open"), hsm.Guard(guard_1), hsm.Target("../open")),
    ),
    hsm.State("open"),
)
`
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "generated.py", Data: []byte(source)}, CompileOptions{
		From:    LanguagePython,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("generated behavior functions should not remain globals: %#v", program.Globals)
	}
	if len(program.Behaviors) != 2 || !strings.Contains(program.Behaviors[0].Body, `instance.set("entered", True)`) || !strings.Contains(program.Behaviors[1].Body, "return True") {
		t.Fatalf("behaviors = %#v", program.Behaviors)
	}
	text := string(output)
	for _, needle := range []string{
		`// instance.set("entered", True)`,
		`// return True`,
		`hsm.Entry(Entry1)`,
		`hsm.Guard(Guard1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original python global") {
		t.Fatalf("Go output preserved generated behavior functions as globals:\n%s", text)
	}
}

func TestPythonFrontendReingestsGeneratedOperationFunction(t *testing.T) {
	source := `import hsm

async def operation_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:
    instance.set("approved", True)

door_model = hsm.Define(
    "Door",
    hsm.Operation("approve", operation_1),
    hsm.Initial(hsm.Target("closed")),
    hsm.State(
        "closed",
        hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
    ),
    hsm.State("open"),
)
`
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "generated.py", Data: []byte(source)}, CompileOptions{
		From:    LanguagePython,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("generated operation function should not remain global: %#v", program.Globals)
	}
	if len(program.Behaviors) != 1 || program.Behaviors[0].Kind != BehaviorOperation {
		t.Fatalf("behaviors = %#v, want one operation behavior", program.Behaviors)
	}
	if !strings.Contains(program.Behaviors[0].Body, `instance.set("approved", True)`) {
		t.Fatalf("operation body = %q", program.Behaviors[0].Body)
	}
	text := string(output)
	for _, needle := range []string{
		`// instance.set("approved", True)`,
		`hsm.Operation("approve", Operation1)`,
		`hsm.OnCall("approve")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original python global") {
		t.Fatalf("Go output preserved generated operation function as global:\n%s", text)
	}
}
