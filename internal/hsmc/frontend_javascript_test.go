package hsmc

import (
	"context"
	"strings"
	"testing"
)

const jsDoorSource = `import * as hsm from "@stateforward/hsm";
import { record as rec } from "./helpers";

class Door {}

function entered(ctx, instance, event) {
  rec("entered");
}

const approve = (ctx, instance, event) => true;
const deadline = (ctx, instance, event) => new Date();

export const model = hsm.Define(
  "Door",
  hsm.Attribute("count", Number, 1),
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
);
`

func TestJavaScriptFrontendLowersModel(t *testing.T) {
	program, err := NewJavaScriptFrontend().Parse(context.Background(), SourceInput{Path: "door.js", Data: []byte(jsDoorSource)})
	if err != nil {
		t.Fatal(err)
	}
	if program.SourceLanguage != LanguageJS {
		t.Fatalf("source language = %q", program.SourceLanguage)
	}
	if len(program.Imports) != 1 || program.Imports[0].Path != "./helpers" {
		t.Fatalf("imports = %#v", program.Imports)
	}
	if len(program.Globals) != 3 {
		t.Fatalf("globals = %#v", program.Globals)
	}
	if strings.Contains(program.Globals[0].Code+program.Globals[1].Code+program.Globals[2].Code, "function entered") {
		t.Fatalf("referenced behavior function leaked as global: %#v", program.Globals)
	}
	model := program.Models[0]
	if model.Name != "Door" || model.Initializer == nil || model.Initializer.Target != "closed" {
		t.Fatalf("model = %#v", model)
	}
	if len(model.Attributes) != 1 || model.Attributes[0].Type != "Number" || model.Attributes[0].Default != "1" {
		t.Fatalf("attributes = %#v", model.Attributes)
	}
	state := model.States[0]
	if len(state.Behaviors) != 1 || len(state.Defers) != 1 || len(state.Transitions) != 2 {
		t.Fatalf("state = %#v", state)
	}
	if state.Transitions[1].Trigger == nil || state.Transitions[1].Trigger.Kind != TriggerAt || state.Transitions[1].Trigger.Expr == nil {
		t.Fatalf("At trigger = %#v", state.Transitions[1].Trigger)
	}
}

func TestJavaScriptFrontendRejectsUnknownHSMDslPartials(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm";

export const DoorModel = hsm.Define(
  "Door",
  hsm.UnknownModelPartial("lost"),
  hsm.Initial(hsm.UnknownInitialPartial("lost"), hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.UnknownStatePartial("lost"),
    hsm.Transition(
      hsm.On("open"),
      hsm.UnknownTransitionPartial("lost"),
      hsm.Target("../open"),
    ),
  ),
  hsm.State("open"),
);
`
	_, err := NewJavaScriptFrontend().Parse(context.Background(), SourceInput{Path: "door.js", Data: []byte(source)})
	if err == nil {
		t.Fatal("expected unknown hsm partial diagnostics")
	}
	for _, want := range []string{
		`unknown or unsupported hsm.UnknownModelPartial partial in model context`,
		`unknown or unsupported hsm.UnknownInitialPartial partial in initial context`,
		`unknown or unsupported hsm.UnknownStatePartial partial in state context`,
		`unknown or unsupported hsm.UnknownTransitionPartial partial in transition context`,
	} {
		if !diagnosticsContain(err, want) {
			t.Fatalf("diagnostics = %v, want %q", err, want)
		}
	}
}

const jsRenamedImportAliasSource = `import {
  Define as HDefine,
  Initial as HInitial,
  Target as HTarget,
  State as HState,
  Transition as HTransition,
  On as HOn,
  Event as HEvent,
} from "@stateforward/hsm";

const opened = HEvent("open");

export const DoorModel = HDefine(
  "Door",
  HInitial(HTarget("closed")),
  HState(
    "closed",
    HTransition(
      HOn(opened.name),
      HTarget("../open"),
    ),
  ),
  HState("open"),
);
`

func TestJavaScriptFrontendLowersRenamedHSMImportAliases(t *testing.T) {
	program, err := NewJavaScriptFrontend().Parse(context.Background(), SourceInput{Path: "door.js", Data: []byte(jsRenamedImportAliasSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 0 {
		t.Fatalf("hsm named import aliases should be compiler-owned, got %#v", program.Imports)
	}
	if len(program.Events) != 1 || program.Events[0].Symbol != "opened" || program.Events[0].Name != "open" {
		t.Fatalf("events = %#v", program.Events)
	}
	if len(program.Models) != 1 {
		t.Fatalf("models = %#v", program.Models)
	}
	model := program.Models[0]
	if model.Name != "Door" || model.Initializer == nil || model.Initializer.Target != "closed" {
		t.Fatalf("model = %#v", model)
	}
	if len(model.States) != 2 || len(model.States[0].Transitions) != 1 {
		t.Fatalf("states = %#v", model.States)
	}
	transition := model.States[0].Transitions[0]
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerOn || len(transition.Trigger.Values) != 1 || transition.Trigger.Values[0].EventSymbol != "opened" {
		t.Fatalf("trigger = %#v", transition.Trigger)
	}
}

const jsCamelCaseAliasSource = `import * as hsm from "@stateforward/hsm";

const opened = hsm.event("open");

export const DoorModel = hsm.define(
  "Door",
  hsm.initial(hsm.target("closed")),
  hsm.state(
    "closed",
    hsm.entry((ctx, instance, event) => {
      instance.set("entered", true);
    }),
    hsm.transition(
      hsm.on(opened.name),
      hsm.guard((ctx, instance, event) => true),
      hsm.target("../open"),
    ),
  ),
  hsm.state("open"),
);
`

func TestJavaScriptFrontendLowersCamelCaseAliases(t *testing.T) {
	program, err := NewJavaScriptFrontend().Parse(context.Background(), SourceInput{Path: "door.js", Data: []byte(jsCamelCaseAliasSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Events) != 1 || program.Events[0].Symbol != "opened" || program.Events[0].Name != "open" {
		t.Fatalf("events = %#v", program.Events)
	}
	if len(program.Models) != 1 {
		t.Fatalf("models = %#v", program.Models)
	}
	model := program.Models[0]
	if model.Name != "Door" || model.Initializer == nil || model.Initializer.Target != "closed" {
		t.Fatalf("model = %#v", model)
	}
	if len(model.States) != 2 || len(model.States[0].Behaviors) != 1 || len(model.States[0].Transitions) != 1 {
		t.Fatalf("states = %#v", model.States)
	}
	transition := model.States[0].Transitions[0]
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerOn || len(transition.Trigger.Values) != 1 || transition.Trigger.Values[0].EventSymbol != "opened" {
		t.Fatalf("trigger = %#v", transition.Trigger)
	}
	if transition.Guard == nil {
		t.Fatalf("guard not lowered: %#v", transition)
	}
	if len(program.Behaviors) != 2 {
		t.Fatalf("behaviors = %#v", program.Behaviors)
	}
}

func TestJavaScriptSourceCompilesToJavaScriptGoAndTypeScript(t *testing.T) {
	jsOutput, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.js", Data: []byte(jsDoorSource)}, CompileOptions{
		From:    LanguageJS,
		To:      LanguageJS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	jsText := string(jsOutput)
	for _, needle := range []string{
		`import * as hsm from "@stateforward/hsm";`,
		`function entry1(ctx, instance, event)`,
		`rec("entered");`,
		`const operation1 = approve;`,
		`const trigger1 = deadline;`,
		`// Type: Number`,
		`hsm.Attribute("count", 1)`,
		`hsm.Operation("approve", operation1)`,
		`hsm.At(trigger1)`,
	} {
		if !strings.Contains(jsText, needle) {
			t.Fatalf("JS output missing %q:\n%s", needle, jsText)
		}
	}

	goOutput, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.js", Data: []byte(jsDoorSource)}, CompileOptions{
		From:    LanguageJS,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	goText := string(goOutput)
	for _, needle := range []string{
		`func Entry1(ctx context.Context, instance hsm.Instance, event hsm.Event)`,
		`// Original javascript behavior entry_1 preserved for manual porting:`,
		`// rec("entered");`,
		`hsm.Operation("approve", Operation1)`,
		`hsm.At(Trigger1)`,
	} {
		if !strings.Contains(goText, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, goText)
		}
	}

	tsOutput, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.js", Data: []byte(jsDoorSource)}, CompileOptions{
		From:    LanguageJS,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	tsText := string(tsOutput)
	for _, needle := range []string{
		`function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void`,
		`// Original javascript behavior entry_1 preserved for manual porting:`,
		`// rec("entered");`,
		`hsm.Operation("approve", operation1)`,
		`hsm.At(trigger1)`,
	} {
		if !strings.Contains(tsText, needle) {
			t.Fatalf("TS output missing %q:\n%s", needle, tsText)
		}
	}
}

func TestJavaScriptFrontendParsesJSXSource(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm";

const Preview = () => <section data-state="closed">Closed</section>;

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.jsx", Data: []byte(source)}, CompileOptions{
		From:    LanguageJS,
		To:      LanguagePython,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if program.SourceLanguage != LanguageJS || len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("program = %#v", program)
	}
	text := string(output)
	for _, needle := range []string{
		`door_model = hsm.Define(`,
		`hsm.Target("closed")`,
		`# const Preview = () => <section data-state="closed">Closed</section>;`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("JSX compile output missing %q:\n%s", needle, text)
		}
	}
}

func TestJavaScriptTargetABI(t *testing.T) {
	abi := BehaviorABIForTrigger(LanguageJS, BehaviorTrigger, TriggerAt)
	if abi == nil || abi.ReturnType != "Date" || !strings.Contains(abi.Signature, "function") {
		t.Fatalf("javascript trigger ABI = %#v", abi)
	}
}

func TestJavaScriptFrontendReingestsGeneratedBehaviorFunctions(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm";

function entry1(ctx, instance, event) {
  instance.set("entered", true);
}

function guard1(ctx, instance, event) {
  return true;
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry(entry1),
    hsm.Transition(hsm.On("open"), hsm.Guard(guard1), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "generated.js", Data: []byte(source)}, CompileOptions{
		From:    LanguageJS,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("generated behavior functions should not remain globals: %#v", program.Globals)
	}
	if len(program.Behaviors) != 2 || !strings.Contains(program.Behaviors[0].Body, `instance.set("entered", true);`) || !strings.Contains(program.Behaviors[1].Body, "return true;") {
		t.Fatalf("behaviors = %#v", program.Behaviors)
	}
	text := string(output)
	for _, needle := range []string{
		`// instance.set("entered", true);`,
		`// return true;`,
		`hsm.Entry(Entry1)`,
		`hsm.Guard(Guard1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original javascript global") {
		t.Fatalf("Go output preserved generated behavior functions as globals:\n%s", text)
	}
}

func TestJavaScriptFrontendLowersNamedBehaviorFunctions(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm";

function entered(ctx, instance, event) {
  instance.set("entered", true);
}

function allowed(ctx, instance, event) {
  return event.name === "open";
}

function approve(instance) {
  return true;
}

function helper(value) {
  return value;
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", approve),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry(entered),
    hsm.Transition(hsm.On("open"), hsm.Guard(allowed), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	program, err := NewJavaScriptFrontend().Parse(context.Background(), SourceInput{Path: "named.js", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want operation helper and ordinary helper", program.Globals)
	}
	if !strings.Contains(program.Globals[0].Code+program.Globals[1].Code, "function approve") ||
		!strings.Contains(program.Globals[0].Code+program.Globals[1].Code, "function helper") {
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
	if !strings.Contains(entryBody, `instance.set("entered", true);`) {
		t.Fatalf("entry body = %q", entryBody)
	}
	if !strings.Contains(guardBody, `return event.name === "open";`) {
		t.Fatalf("guard body = %q", guardBody)
	}
	if operationCode != "approve" {
		t.Fatalf("operation code = %q, want unresolved operation reference", operationCode)
	}
}

func TestJavaScriptFrontendReingestsGeneratedOperationFunction(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm";

function operation1(ctx, instance, event) {
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
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "generated.js", Data: []byte(source)}, CompileOptions{
		From:    LanguageJS,
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
	if !strings.Contains(program.Behaviors[0].Body, `instance.set("approved", true);`) {
		t.Fatalf("operation body = %q", program.Behaviors[0].Body)
	}
	text := string(output)
	for _, needle := range []string{
		`// instance.set("approved", true);`,
		`hsm.Operation("approve", Operation1)`,
		`hsm.OnCall("approve")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original javascript global") {
		t.Fatalf("Go output preserved generated operation function as global:\n%s", text)
	}
}
