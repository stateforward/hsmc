package hsmc

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

const goDoorSource = `package sample

import (
	"context"
	"fmt"

	hsm "github.com/stateforward/hsm.go"
)

type DoorHSM struct {
	hsm.HSM
	log []string
}

func record(value string) string {
	return fmt.Sprintf("event:%s", value)
}

var DoorModel = hsm.Define(
	"Door",
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed",
		hsm.Entry(func(ctx context.Context, sm *DoorHSM, event hsm.Event) {
			sm.log = append(sm.log, record("closed"))
		}),
		hsm.Transition(
			hsm.On("open"),
			hsm.Guard(func(ctx context.Context, sm *DoorHSM, event hsm.Event) bool {
				return len(sm.log) >= 0
			}),
			hsm.Target("../open"),
			hsm.Effect(func(ctx context.Context, sm *DoorHSM, event hsm.Event) {
				sm.log = append(sm.log, event.Name)
			}),
		),
	),
	hsm.State("open"),
)
`

func TestGoFrontendLowersModelAndBehaviorBoundaries(t *testing.T) {
	program, err := NewGoFrontend().Parse(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)})
	if err != nil {
		t.Fatal(err)
	}
	if program.PackageName != "sample" {
		t.Fatalf("package = %q, want sample", program.PackageName)
	}
	if len(program.Models) != 1 {
		t.Fatalf("models = %d, want 1", len(program.Models))
	}
	model := program.Models[0]
	if model.Name != "Door" {
		t.Fatalf("model name = %q, want Door", model.Name)
	}
	if model.Initializer == nil || model.Initializer.Target != "closed" {
		t.Fatalf("initial = %#v, want target closed", model.Initializer)
	}
	if len(model.States) != 2 {
		t.Fatalf("states = %d, want 2", len(model.States))
	}
	closed := model.States[0]
	if closed.Name != "closed" || len(closed.Behaviors) != 1 || len(closed.Transitions) != 1 {
		t.Fatalf("closed state = %#v", closed)
	}
	transition := closed.Transitions[0]
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerOn || transition.Trigger.Value != "open" || !transition.Trigger.IsString {
		t.Fatalf("trigger = %#v", transition.Trigger)
	}
	if transition.Guard == nil {
		t.Fatal("guard was not lowered")
	}
	if len(transition.Effects) != 1 {
		t.Fatalf("effects = %d, want 1", len(transition.Effects))
	}
	if len(program.Behaviors) != 3 {
		t.Fatalf("behaviors = %d, want 3", len(program.Behaviors))
	}
	for _, behavior := range program.Behaviors {
		if behavior.OwnerID == "" || behavior.Kind == "" || behavior.Code == "" {
			t.Fatalf("incomplete behavior = %#v", behavior)
		}
	}
	if len(program.Globals) == 0 {
		t.Fatal("expected helper globals to be available for adapter edits")
	}
}

const goAliasedRuntimeSource = `package sample

import sf "github.com/stateforward/hsm.go"

type DoorHSM struct {
	sf.HSM
}

var DoorModel = sf.Define(
	"Door",
	sf.Initial(sf.Target("closed")),
	sf.State("closed",
		sf.Entry(func(ctx sf.Context, sm *DoorHSM, event sf.Event) {
			sm.Set("entered", true)
		}),
	),
)
`

func TestGoFrontendNormalizesAliasedRuntimeImport(t *testing.T) {
	program, err := NewGoFrontend().Parse(context.Background(), SourceInput{Path: "door.go", Data: []byte(goAliasedRuntimeSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 0 {
		t.Fatalf("hsm runtime alias import should be compiler-owned, got %#v", program.Imports)
	}
	if len(program.Globals) != 1 || !strings.Contains(program.Globals[0].Code, "hsm.HSM") || strings.Contains(program.Globals[0].Code, "sf.HSM") {
		t.Fatalf("globals did not normalize hsm alias: %#v", program.Globals)
	}
	if len(program.Behaviors) != 1 || !strings.Contains(program.Behaviors[0].Signature, "event hsm.Event") || strings.Contains(program.Behaviors[0].Code, "sf.") {
		t.Fatalf("behavior did not normalize hsm alias: %#v", program.Behaviors)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
}

func TestGoFrontendRejectsUnknownHSMDslPartials(t *testing.T) {
	source := `package sample

import sf "github.com/stateforward/hsm.go"

var DoorModel = sf.Define(
	"Door",
	sf.UnknownModelPartial(),
	sf.Initial(sf.UnknownInitialPartial(), sf.Target("closed")),
	sf.State("closed",
		sf.UnknownStatePartial(),
		sf.Transition(
			sf.UnknownTransitionPartial(),
			sf.Target("../open"),
		),
	),
)
`
	_, err := NewGoFrontend().Parse(context.Background(), SourceInput{Path: "door.go", Data: []byte(source)})
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

func TestCompilerConvertsAliasedGoRuntimeSourceToGo(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goAliasedRuntimeSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`hsm "github.com/stateforward/hsm.go"`,
		"hsm.HSM",
		"event hsm.Event",
		"var DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "sf.") || strings.Contains(text, `sf "github.com/stateforward/hsm.go"`) {
		t.Fatalf("Go output retained source runtime alias:\n%s", text)
	}
}

func TestGoBackendKeepsModelGeneratedAndBehaviorsSeparate(t *testing.T) {
	compiler := NewCompiler()
	output, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	required := []string{
		"var DoorModel = hsm.Define(",
		"func Entry1(ctx context.Context, sm *DoorHSM, event hsm.Event)",
		"hsm.Entry(Entry1)",
		"hsm.Guard(Guard1)",
		"hsm.Effect(Effect1)",
		`hsm.On(hsm.Event{Name: "open"})`,
	}
	for _, needle := range required {
		if !strings.Contains(text, needle) {
			t.Fatalf("generated output missing %q:\n%s", needle, text)
		}
	}
}

func TestGoFrontendReingestsGeneratedBehaviorFunctions(t *testing.T) {
	source := `package sample

import (
	"context"

	hsm "github.com/stateforward/hsm.go"
)

func Entry1(ctx context.Context, instance hsm.Instance, event hsm.Event) {
	instance.Set("entered", true)
}

func Guard1(ctx context.Context, instance hsm.Instance, event hsm.Event) bool {
	return true
}

var DoorModel = hsm.Define(
	"Door",
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed",
		hsm.Entry(Entry1),
		hsm.Transition(hsm.On("open"), hsm.Guard(Guard1), hsm.Target("../open")),
	),
	hsm.State("open"),
)
`
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "generated.go", Data: []byte(source)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("generated behavior functions should not remain globals: %#v", program.Globals)
	}
	if len(program.Behaviors) != 2 || !strings.Contains(program.Behaviors[0].Body, `instance.Set("entered", true)`) || !strings.Contains(program.Behaviors[1].Body, "return true") {
		t.Fatalf("behaviors = %#v", program.Behaviors)
	}
	text := string(output)
	for _, needle := range []string{
		`// instance.Set("entered", true)`,
		`// return true`,
		`hsm.Entry(entry1)`,
		`hsm.Guard(guard1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original go global") {
		t.Fatalf("TypeScript output preserved generated behavior functions as globals:\n%s", text)
	}
}

func TestGoFrontendLowersNamedBehaviorFunctions(t *testing.T) {
	source := `package sample

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

func helper(value string) string {
	return value
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
	program, err := NewGoFrontend().Parse(context.Background(), SourceInput{Path: "named.go", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 1 || !strings.Contains(program.Globals[0].Code, "func helper") {
		t.Fatalf("globals = %#v, want only non-behavior helper", program.Globals)
	}
	if len(program.Behaviors) != 2 {
		t.Fatalf("behaviors = %d, want entry and guard", len(program.Behaviors))
	}
	var entryBody, guardBody string
	for _, behavior := range program.Behaviors {
		switch behavior.Kind {
		case BehaviorEntry:
			entryBody = behavior.Body
		case BehaviorGuard:
			guardBody = behavior.Body
		}
		if behavior.Code == "entered" || behavior.Code == "allowed" {
			t.Fatalf("behavior kept only identifier instead of resolved function body: %#v", behavior)
		}
	}
	if !strings.Contains(entryBody, `instance.Set("entered", true)`) {
		t.Fatalf("entry body = %q", entryBody)
	}
	if !strings.Contains(guardBody, `return event.Name == "open"`) {
		t.Fatalf("guard body = %q", guardBody)
	}
}

func TestGoFrontendReingestsGeneratedOperationFunction(t *testing.T) {
	source := `package sample

import (
	"context"

	hsm "github.com/stateforward/hsm.go"
)

func Operation1(ctx context.Context, instance hsm.Instance, event hsm.Event) {
	instance.Set("approved", true)
}

var DoorModel = hsm.Define(
	"Door",
	hsm.Operation("approve", Operation1),
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed",
		hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
	),
	hsm.State("open"),
)
`
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "generated.go", Data: []byte(source)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
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
	if !strings.Contains(program.Behaviors[0].Body, `instance.Set("approved", true)`) {
		t.Fatalf("operation body = %q", program.Behaviors[0].Body)
	}
	text := string(output)
	for _, needle := range []string{
		`// instance.Set("approved", true)`,
		`hsm.Operation("approve", operation1)`,
		`hsm.OnCall("approve")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original go global") {
		t.Fatalf("TypeScript output preserved generated operation function as global:\n%s", text)
	}
}

func TestCompilerRejectsInvalidModelStructure(t *testing.T) {
	source := `package sample

import hsm "github.com/stateforward/hsm.go"

var BrokenModel = hsm.Define(
	"Broken",
	hsm.State("idle", hsm.Transition(hsm.On("go"), hsm.Target("../missing"))),
)
`
	_, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "broken.go", Data: []byte(source)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	text := err.Error()
	if !strings.Contains(text, `model "Broken" is missing hsm.Initial`) {
		t.Fatalf("validation error did not mention missing initial: %v", err)
	}
}

func TestBehaviorPatchCanChangeImportsAndBehaviorButNotModel(t *testing.T) {
	program, err := NewGoFrontend().Parse(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)})
	if err != nil {
		t.Fatal(err)
	}
	originalModel := program.Models[0]
	patch := &BehaviorPatch{
		Imports: []Import{{Path: "strings"}},
		Behaviors: []Behavior{{
			ID:      program.Behaviors[0].ID,
			Body:    `sm.log = append(sm.log, strings.TrimSpace(" patched "))`,
			Imports: []Import{{Path: "strings"}},
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguageGo, Imports: program.Imports, Globals: program.Globals, Behaviors: program.Behaviors}
	if err := applyBehaviorPatch(program, request, patch); err != nil {
		t.Fatal(err)
	}
	if program.Imports[len(program.Imports)-1].Path != "strings" {
		t.Fatalf("imports were not patched: %#v", program.Imports)
	}
	patched := program.Behaviors[0]
	if patched.OwnerID != originalModel.States[0].ID {
		t.Fatalf("adapter changed behavior owner to %q", patched.OwnerID)
	}
	if patched.Kind != BehaviorEntry {
		t.Fatalf("adapter changed behavior kind to %q", patched.Kind)
	}
	if patched.SourceLanguage != LanguageGo {
		t.Fatalf("adapter changed source language to %q", patched.SourceLanguage)
	}
	if program.Models[0].Name != originalModel.Name || program.Models[0].States[0].Name != originalModel.States[0].Name {
		t.Fatalf("model structure changed: %#v", program.Models[0])
	}
}

func TestBehaviorPatchRejectsUnknownBehavior(t *testing.T) {
	program, err := NewGoFrontend().Parse(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)})
	if err != nil {
		t.Fatal(err)
	}
	err = applyBehaviorPatch(program, AdapterRequest{TargetLanguage: LanguageGo, Behaviors: program.Behaviors}, &BehaviorPatch{Behaviors: []Behavior{{ID: "missing"}}})
	if err == nil || !strings.Contains(err.Error(), `unknown behavior "missing"`) {
		t.Fatalf("expected unknown behavior rejection, got %v", err)
	}
}

func TestJSONIRRoundTripBackendAndFrontend(t *testing.T) {
	compiler := NewCompiler()
	irOutput, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageJSONIR,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	var decoded Program
	if err := json.Unmarshal(irOutput, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Models) != 1 || decoded.Models[0].Name != "Door" {
		t.Fatalf("unexpected IR output: %s", irOutput)
	}
	for _, behavior := range decoded.Behaviors {
		if behavior.TargetLanguage != "" || behavior.TargetABI != nil {
			t.Fatalf("json-ir output leaked target behavior metadata: %#v", behavior)
		}
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
		t.Fatalf("go output from IR missing model:\n%s", goOutput)
	}
}

func TestTypeScriptBackendEmitsModelAndCommentedForeignBehaviors(t *testing.T) {
	compiler := NewCompiler()
	output, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	required := []string{
		`import * as hsm from "@stateforward/hsm.ts";`,
		"export const DoorModel = hsm.Define(",
		"hsm.Initial(",
		"hsm.State(",
		`hsm.On("open")`,
		"hsm.Guard(guard1)",
		"// Original go behavior entry_1 preserved for manual porting:",
		"// sm.log = append(sm.log, record(\"closed\"))",
	}
	for _, needle := range required {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
	forbidden := []string{
		`from "context"`,
		`from "fmt"`,
		`from "github.com/stateforward/hsm.go"`,
	}
	for _, needle := range forbidden {
		if strings.Contains(text, needle) {
			t.Fatalf("TypeScript output leaked Go import %q:\n%s", needle, text)
		}
	}
}

func TestTypeScriptBackendCommentsForeignTriggerBehaviors(t *testing.T) {
	source := `package sample

import (
	"context"
	"time"

	hsm "github.com/stateforward/hsm.go"
)

type TimerHSM struct{ hsm.HSM }

var TimerModel = hsm.Define(
	"Timer",
	hsm.Initial(hsm.Target("idle")),
	hsm.State("idle",
		hsm.Transition(
			hsm.After(func(ctx context.Context, sm *TimerHSM, event hsm.Event) time.Duration {
				return time.Second
			}),
			hsm.Target("../done"),
		),
	),
	hsm.State("done"),
)
`
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "timer.go", Data: []byte(source)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original go behavior trigger_1 preserved for manual porting:",
		"// return time.Second",
		"return 0;",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing commented trigger behavior %q:\n%s", needle, text)
		}
	}
	if !strings.Contains(text, "hsm.After(trigger1)") {
		t.Fatalf("TypeScript output did not reference trigger behavior:\n%s", text)
	}
}

func TestTypeScriptBackendUsesAdapterBehaviorImportsAndBodies(t *testing.T) {
	program := &Program{
		SourceLanguage: LanguageJSONIR,
		PackageName:    "sample",
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			Initializer: &Initial{
				OwnerID: "model_1",
				ID:      "initial_1",
				Target:  "closed",
			},
			States: []State{{
				ID:   "state_1",
				Name: "closed",
				Kind: StateKindState,
				Behaviors: []BehaviorRef{{
					ID:   "entry_1",
					Kind: BehaviorEntry,
				}},
			}},
		}},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageTS,
			Body:           `instance["log"] = format.record("closed");`,
			Imports: []Import{{
				Path:     "./format",
				Alias:    "format",
				Language: LanguageTS,
			}},
		}},
	}
	output, err := NewTypeScriptBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	required := []string{
		`import * as format from "./format";`,
		`instance["log"] = format.record("closed");`,
		"hsm.Entry(entry1)",
	}
	for _, needle := range required {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}
