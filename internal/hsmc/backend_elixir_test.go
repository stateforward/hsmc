package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestElixirBackendEmitsHsmExModel(t *testing.T) {
	program := &Program{
		Imports: []Import{{Path: "MyApp.Helpers", Alias: "Helpers", Language: LanguageElixir}},
		Events:  []EventDecl{{ID: "event_1", Symbol: "OpenEvent", Name: "open", Language: LanguageGo}},
		Behaviors: []Behavior{
			{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo, Body: `instance.Log = append(instance.Log, "closed")`},
			{ID: "guard_1", OwnerID: "transition_1", Kind: BehaviorGuard, SourceLanguage: LanguageGo, Body: `return instance.Ready`},
			{ID: "operation_1", OwnerID: "operation_1", Kind: BehaviorOperation, SourceLanguage: LanguageGo},
		},
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			Initializer: &Initial{
				ID:     "initial_1",
				Target: "closed",
			},
			Attributes: []Attribute{
				{ID: "attribute_1", Name: "count", Type: "int", HasDefault: true, Default: "1", Language: LanguageGo},
				{ID: "attribute_2", Name: "label", HasDefault: true, Default: `"closed"`, Language: LanguageTS},
			},
			Operations: []Operation{{ID: "operation_1", Name: "approve", Behavior: &BehaviorRef{ID: "operation_1", Kind: BehaviorOperation}}},
			States: []State{{
				ID:        "state_1",
				Name:      "closed",
				Kind:      StateKindState,
				Behaviors: []BehaviorRef{{ID: "entry_1", Kind: BehaviorEntry}},
				Transitions: []Transition{{
					ID:      "transition_1",
					Target:  "open",
					Trigger: &Trigger{Kind: TriggerOn, Values: []TriggerValue{{EventSymbol: "OpenEvent", Value: "open"}}},
					Guard:   &BehaviorRef{ID: "guard_1", Kind: BehaviorGuard},
				}},
			}, {
				ID:   "state_2",
				Name: "open",
				Kind: StateKindFinal,
			}},
		}},
	}
	AssignTargetABI(program, LanguageElixir)

	output, err := NewElixirBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`defmodule DoorHSM do`,
		`alias MyApp.Helpers, as: Helpers`,
		`def open_event do`,
		`HSM.event("open")`,
		`def entry_1(_ctx, _instance, _event) do`,
		`# Original go behavior entry_1 preserved for manual porting:`,
		`HSM.define("Door", [`,
		`HSM.initial([`,
		`HSM.target("closed")`,
		`HSM.attribute("count", :integer, 1)`,
		`HSM.attribute("label", "closed")`,
		`HSM.operation("approve", &__MODULE__.operation_1/3)`,
		`HSM.state("closed", [`,
		`HSM.entry(&__MODULE__.entry_1/3)`,
		`HSM.on(open_event())`,
		`HSM.guard(&__MODULE__.guard_1/3)`,
		`HSM.final("open")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Elixir output missing %q:\n%s", needle, text)
		}
	}
}

func TestElixirBehaviorABI(t *testing.T) {
	guard := BehaviorABIFor(LanguageElixir, BehaviorGuard)
	if guard.Language != LanguageElixir || guard.ReturnType != "boolean" || !strings.Contains(guard.Signature, "fun(ctx, instance, event)") {
		t.Fatalf("guard ABI = %#v, want Elixir boolean fun ABI", guard)
	}
	timer := BehaviorABIForTrigger(LanguageElixir, BehaviorTrigger, TriggerAfter)
	if timer.ReturnType != "non_neg_integer" {
		t.Fatalf("timer ABI return = %q, want non_neg_integer", timer.ReturnType)
	}
	when := BehaviorABIForTrigger(LanguageElixir, BehaviorTrigger, TriggerWhen)
	if when.ReturnType != "boolean" {
		t.Fatalf("when ABI return = %q, want boolean", when.ReturnType)
	}
}
