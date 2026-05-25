package hsmc

import (
	"context"
	"strings"
	"testing"
)

const tsDoorSource = `import * as hsm from "@stateforward/hsm.ts";
import * as format from "./format";

class Door extends hsm.Instance {
  log: string[] = [];
}

function record(value: string): string {
  return format.record(value);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx: hsm.Context, instance: Door, event: hsm.EventRecord) => {
      instance.log.push(record("closed"));
    }),
    hsm.Transition(
      hsm.On("open"),
      hsm.Guard((ctx, instance, event) => instance.log.length >= 0),
      hsm.Target("../open"),
      hsm.Effect((ctx, instance, event) => {
        instance.log.push(event.name);
      }),
    ),
  ),
  hsm.State("open"),
);
`

func TestTypeScriptFrontendLowersModelAndBehaviors(t *testing.T) {
	program, err := NewTypeScriptFrontend().Parse(context.Background(), SourceInput{Path: "door.ts", Data: []byte(tsDoorSource)})
	if err != nil {
		t.Fatal(err)
	}
	if program.SourceLanguage != LanguageTS {
		t.Fatalf("source language = %q, want TypeScript", program.SourceLanguage)
	}
	if len(program.Imports) != 1 || program.Imports[0].Path != "./format" || program.Imports[0].Alias != "format" {
		t.Fatalf("imports = %#v", program.Imports)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %d, want class and helper function", len(program.Globals))
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
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerOn || transition.Trigger.Value != "open" || transition.Trigger.Language != LanguageTS {
		t.Fatalf("trigger = %#v", transition.Trigger)
	}
	if transition.Guard == nil || len(transition.Effects) != 1 {
		t.Fatalf("transition behaviors not lowered: %#v", transition)
	}
	if len(program.Behaviors) != 3 {
		t.Fatalf("behaviors = %d, want 3", len(program.Behaviors))
	}
	var guardBody string
	for _, behavior := range program.Behaviors {
		if behavior.SourceLanguage != LanguageTS {
			t.Fatalf("behavior source language = %q", behavior.SourceLanguage)
		}
		if behavior.Kind == BehaviorGuard {
			guardBody = behavior.Body
		}
	}
	if !strings.Contains(guardBody, "return instance.log.length >= 0;") {
		t.Fatalf("guard expression body not normalized: %q", guardBody)
	}
}

const tsTypeOnlyImportSource = `import * as hsm from "@stateforward/hsm.ts";
import type { DoorSnapshot, DoorStatus as Status } from "./types";
import { type DoorMetrics, recordMetric } from "./metrics";

class Door extends hsm.Instance {
  snapshot?: DoorSnapshot;
  status?: Status;
  metrics?: DoorMetrics;
}

function record(value: string): string {
  return recordMetric(value);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`

func TestTypeScriptFrontendPreservesTypeOnlyImports(t *testing.T) {
	program, err := NewTypeScriptFrontend().Parse(context.Background(), SourceInput{Path: "door.ts", Data: []byte(tsTypeOnlyImportSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 2 {
		t.Fatalf("imports = %#v", program.Imports)
	}
	imp := program.Imports[0]
	if imp.Path != "./types" || !imp.TypeOnly || imp.Language != LanguageTS {
		t.Fatalf("type-only import metadata = %#v", imp)
	}
	if len(imp.Specifiers) != 2 || imp.Specifiers[0].Name != "DoorSnapshot" || imp.Specifiers[1].Name != "DoorStatus" || imp.Specifiers[1].Alias != "Status" {
		t.Fatalf("type-only import specifiers = %#v", imp.Specifiers)
	}
	if strings.Join(imp.LocalNames, ",") != "DoorSnapshot,Status" {
		t.Fatalf("type-only import local names = %#v", imp.LocalNames)
	}
	mixed := program.Imports[1]
	if mixed.Path != "./metrics" || mixed.TypeOnly {
		t.Fatalf("mixed import metadata = %#v", mixed)
	}
	if len(mixed.Specifiers) != 2 || !mixed.Specifiers[0].TypeOnly || mixed.Specifiers[0].Name != "DoorMetrics" || mixed.Specifiers[1].TypeOnly || mixed.Specifiers[1].Name != "recordMetric" {
		t.Fatalf("mixed import specifiers = %#v", mixed.Specifiers)
	}
}

func TestCompilerPreservesTypeScriptTypeOnlyImportsInTypeScriptTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.ts", Data: []byte(tsTypeOnlyImportSource)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, `import type { DoorSnapshot, DoorStatus as Status } from "./types";`) {
		t.Fatalf("TypeScript output missing type-only import:\n%s", text)
	}
	if !strings.Contains(text, `import { type DoorMetrics, recordMetric } from "./metrics";`) {
		t.Fatalf("TypeScript output missing mixed type-only specifier import:\n%s", text)
	}
	if strings.Contains(text, `import { DoorSnapshot, DoorStatus as Status } from "./types";`) {
		t.Fatalf("TypeScript output rendered type-only import as value import:\n%s", text)
	}
}

func TestTypeScriptFrontendRejectsUnknownHSMDslPartials(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";

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
	_, err := NewTypeScriptFrontend().Parse(context.Background(), SourceInput{Path: "door.ts", Data: []byte(source)})
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

const tsCamelCaseAliasSource = `import * as hsm from "@stateforward/hsm.ts";

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

func TestTypeScriptFrontendLowersCamelCaseAliases(t *testing.T) {
	program, err := NewTypeScriptFrontend().Parse(context.Background(), SourceInput{Path: "door.ts", Data: []byte(tsCamelCaseAliasSource)})
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

const tsRenamedImportAliasSource = `import {
  Define as HDefine,
  Initial as HInitial,
  Target as HTarget,
  State as HState,
  Entry as HEntry,
  Transition as HTransition,
  On as HOn,
  Guard as HGuard,
  Event as HEvent,
} from "@stateforward/hsm.ts";

const opened = HEvent("open");

export const DoorModel = HDefine(
  "Door",
  HInitial(HTarget("closed")),
  HState(
    "closed",
    HEntry((ctx, instance, event) => {
      instance.set("entered", true);
    }),
    HTransition(
      HOn(opened.name),
      HGuard((ctx, instance, event) => true),
      HTarget("../open"),
    ),
  ),
  HState("open"),
);
`

func TestTypeScriptFrontendLowersRenamedHSMImportAliases(t *testing.T) {
	program, err := NewTypeScriptFrontend().Parse(context.Background(), SourceInput{Path: "door.ts", Data: []byte(tsRenamedImportAliasSource)})
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

func TestTypeScriptFrontendParsesTSXSource(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";

const Preview = () => <section data-state="closed">Closed</section>;

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.tsx", Data: []byte(source)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguagePython,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if program.SourceLanguage != LanguageTS || len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("program = %#v", program)
	}
	text := string(output)
	for _, needle := range []string{
		`door_model = hsm.Define(`,
		`hsm.Target("closed")`,
		`# const Preview = () => <section data-state="closed">Closed</section>;`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TSX compile output missing %q:\n%s", needle, text)
		}
	}
}

func TestTypeScriptSourceToTypeScriptUsesStableABIWithEditableInstanceAlias(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.ts", Data: []byte(tsDoorSource)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`function entry1(ctx: hsm.Context, _instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void`,
		`const instance = _instance as any;`,
		`instance.log.push(record("closed"));`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestTypeScriptFrontendToGoCommentsForeignBehaviorsAndKeepsModel(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.ts", Data: []byte(tsDoorSource)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	required := []string{
		"var DoorModel = hsm.Define(",
		"hsm.Initial(",
		"hsm.State(",
		`hsm.On(hsm.Event{Name: "open"})`,
		"hsm.Guard(Guard1)",
		"// Original typescript global global_1 preserved for manual porting:",
		"// class Door extends hsm.Instance",
		"// Source imports:",
		`// import * as format from "./format";`,
		"// Original typescript behavior entry_1 preserved for manual porting:",
		"// instance.log.push(record(\"closed\"));",
	}
	for _, needle := range required {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
	forbidden := []string{
		"\nclass Door extends hsm.Instance",
		"\nimport * as format from \"./format\";",
	}
	for _, needle := range forbidden {
		if strings.Contains(text, needle) {
			t.Fatalf("Go output leaked TypeScript %q:\n%s", needle, text)
		}
	}
}

func TestTypeScriptFrontendToJSONIRToGo(t *testing.T) {
	compiler := NewCompiler()
	irOutput, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.ts", Data: []byte(tsDoorSource)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguageJSONIR,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	goOutput, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: irOutput}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(goOutput), "var DoorModel = hsm.Define(") {
		t.Fatalf("Go output from TS IR missing model:\n%s", goOutput)
	}
}

func TestTypeScriptFrontendReingestsGeneratedBehaviorFunctions(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";

function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("entered", true);
}

function guard1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): boolean {
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
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "generated.ts", Data: []byte(source)}, CompileOptions{
		From:    LanguageTS,
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
	if strings.Contains(text, "Original typescript global") {
		t.Fatalf("Go output preserved generated behavior functions as globals:\n%s", text)
	}
}

func TestTypeScriptFrontendLowersNamedBehaviorFunctions(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";

function entered(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("entered", true);
}

function allowed(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): boolean {
  return event.name === "open";
}

function approve(instance: InstanceType<typeof hsm.Instance>): boolean {
  return true;
}

function helper(value: string): string {
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
	program, err := NewTypeScriptFrontend().Parse(context.Background(), SourceInput{Path: "named.ts", Data: []byte(source)})
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

func TestTypeScriptFrontendReingestsGeneratedOperationFunction(t *testing.T) {
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
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "generated.ts", Data: []byte(source)}, CompileOptions{
		From:    LanguageTS,
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
	if strings.Contains(text, "Original typescript global") {
		t.Fatalf("Go output preserved generated operation function as global:\n%s", text)
	}
}
