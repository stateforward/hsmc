package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestXStateBackendCompilesGoModelToCreateMachine(t *testing.T) {
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageXState,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`import { fromCallback, setup } from "xstate";`,
		`type XStateContext = Record<string, unknown>;`,
		`type XStateBehaviorArgs = {`,
		`function entry1(args: XStateBehaviorArgs): void`,
		`function guard1(args: XStateBehaviorArgs): boolean`,
		`export const DoorModel = setup({`,
		`actions: {`,
		`entry1,`,
		`effect1,`,
		`guards: {`,
		`guard1,`,
		`}).createMachine({`,
		`id: "Door"`,
		`initial: "closed"`,
		`"closed": {`,
		`entry: [{ type: "entry1" }]`,
		`"open": { target: "#Door.open", guard: { type: "guard1" }, actions: [{ type: "effect1" }] }`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("XState output missing %q:\n%s", needle, text)
		}
	}
	if program.Behaviors[0].TargetABI == nil || program.Behaviors[0].TargetABI.Language != LanguageXState {
		t.Fatalf("behavior ABI = %#v, want xstate", program.Behaviors[0].TargetABI)
	}
}

func TestXStateBackendEmitsTimersActivitiesAndFinalStates(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:   "model_1",
			Name: "TimerDoor",
			Initializer: &Initial{
				OwnerID: "model_1",
				ID:      "initial_1",
				Target:  "closed",
			},
			Attributes: []Attribute{{ID: "attribute_1", Name: "delayMs", HasDefault: true, Default: "1000", Language: LanguageTS}},
			States: []State{{
				ID:   "state_1",
				Name: "closed",
				Kind: StateKindState,
				Behaviors: []BehaviorRef{{
					ID:   "activity_1",
					Kind: BehaviorActivity,
				}},
				Transitions: []Transition{
					{
						ID:      "transition_after",
						OwnerID: "state_1",
						Target:  "../open",
						Trigger: &Trigger{Kind: TriggerAfter, Value: "delayMs", IsString: true},
					},
					{
						ID:      "transition_every",
						OwnerID: "state_1",
						Target:  ".",
						Trigger: &Trigger{Kind: TriggerEvery, Expr: &BehaviorRef{ID: "trigger_every", Kind: BehaviorTrigger}},
						Effects: []BehaviorRef{{ID: "effect_1", Kind: BehaviorEffect}},
					},
				},
			}, {
				ID:   "state_2",
				Name: "open",
				Kind: StateKindFinal,
			}},
		}},
		Behaviors: []Behavior{
			{ID: "activity_1", OwnerID: "state_1", Kind: BehaviorActivity, SourceLanguage: LanguageGo},
			{ID: "trigger_every", OwnerID: "transition_every", Kind: BehaviorTrigger, TriggerKind: TriggerEvery, SourceLanguage: LanguageGo},
			{ID: "effect_1", OwnerID: "transition_every", Kind: BehaviorEffect, SourceLanguage: LanguageGo},
		},
	}
	AssignTargetABI(program, LanguageXState)

	output, err := NewXStateBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`context: {`,
		`"delayMs": 1000`,
		`actors: {`,
		`activity1: fromCallback<XStateEvent, XStateActorInput>`,
		`transitionEveryEvery: fromCallback<XStateEvent, XStateActorInput>`,
		`delays: {`,
		`transitionAfterDelay: ({ context }: XStateBehaviorArgs) => Number(context?.["delayMs"] ?? 0)`,
		`after: {`,
		`"transitionAfterDelay": { target: "#TimerDoor.open" }`,
		`"__hsmc.every.transition_every": { target: "#TimerDoor.closed", actions: [{ type: "effect1" }] }`,
		`type: "final"`,
		`let cleanup: (() => void) | undefined;`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("XState timer output missing %q:\n%s", needle, text)
		}
	}
}

func TestXStateTargetABIUsesXStateBehaviorArgs(t *testing.T) {
	abi := BehaviorABIForTrigger(LanguageXState, BehaviorTrigger, TriggerEvery)
	if abi.Language != LanguageXState || abi.Signature != "function(args: XStateBehaviorArgs): number | Promise<number>" {
		t.Fatalf("xstate ABI = %#v", abi)
	}
	if len(abi.Parameters) != 1 || abi.Parameters[0].Type != "XStateBehaviorArgs" {
		t.Fatalf("xstate ABI parameters = %#v", abi.Parameters)
	}
}
