package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestHistoryPseudostatesRequireDefaultTransitions(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:   "model_1",
			Name: "HistoryDoor",
			Initializer: &Initial{
				OwnerID: "model_1",
				ID:      "initial_1",
				Target:  "closed",
			},
			States: []State{
				{
					ID:   "state_1",
					Name: "closed",
					Kind: StateKindState,
					States: []State{
						{ID: "state_2", Name: "a", Kind: StateKindState},
						{ID: "state_3", Name: "history", Kind: StateKindShallowHistory},
					},
					Initializer: &Initial{
						OwnerID: "state_1",
						ID:      "initial_2",
						Target:  "a",
					},
				},
				{ID: "state_4", Name: "open", Kind: StateKindState},
			},
		}},
	}
	err := Validate(program)
	if err == nil || !strings.Contains(err.Error(), `shallow_history "history" must have a default transition`) {
		t.Fatalf("Validate() = %v, want history default transition error", err)
	}
}

func TestHistoryPseudostatesRequireExactlyOneTargetedDefaultTransition(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:          "model_1",
			Name:        "HistoryDoor",
			Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
			States: []State{{
				ID:   "state_1",
				Name: "closed",
				Kind: StateKindState,
				States: []State{
					{ID: "state_2", Name: "a", Kind: StateKindState},
					{
						ID:   "state_3",
						Name: "history",
						Kind: StateKindDeepHistory,
						Transitions: []Transition{
							{OwnerID: "state_3", ID: "transition_1"},
							{OwnerID: "state_3", ID: "transition_2", Target: "../a"},
						},
					},
				},
				Initializer: &Initial{OwnerID: "state_1", ID: "initial_2", Target: "a"},
			}},
		}},
	}

	err := Validate(program)
	if err == nil {
		t.Fatal("expected malformed history default transition diagnostics")
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
		`deep_history "history" must have exactly one default transition`,
		`deep_history "history" default transition "transition_1" is missing target`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected %q in diagnostics, got %v", needle, err)
		}
	}
}

func TestHistoryDefaultTransitionRejectsSourceAndTrigger(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:          "model_1",
			Name:        "HistoryDoor",
			Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
			States: []State{{
				ID:   "state_1",
				Name: "closed",
				Kind: StateKindState,
				States: []State{
					{ID: "state_2", Name: "a", Kind: StateKindState},
					{
						ID:   "state_3",
						Name: "history",
						Kind: StateKindShallowHistory,
						Transitions: []Transition{{
							OwnerID: "state_3",
							ID:      "transition_1",
							Source:  "../a",
							Target:  "../a",
							Trigger: &Trigger{Kind: TriggerOn, Value: "resume", IsString: true},
						}},
					},
				},
				Initializer: &Initial{OwnerID: "state_1", ID: "initial_2", Target: "a"},
			}},
		}},
	}

	err := Validate(program)
	if err == nil {
		t.Fatal("expected malformed history default transition diagnostics")
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
		`shallow_history "history" default transition "transition_1" must not set source`,
		`shallow_history "history" default transition "transition_1" must not set trigger`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected %q in diagnostics, got %v", needle, err)
		}
	}
}

func TestCSharpFrontendLowersHistoryDefaultTransition(t *testing.T) {
	source := `using Stateforward.Hsm;

public static class Sample
{
    public static readonly Model HistoryModel = Hsm.Define(
        "HistoryDoor",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed",
            Hsm.Initial(Hsm.Target("a")),
            Hsm.State("a"),
            Hsm.ShallowHistory("history",
                Hsm.Transition(Hsm.Target("../a"))
            )
        ),
        Hsm.State("open",
            Hsm.Transition(Hsm.On("resume"), Hsm.Target("../closed/history"))
        )
    );
}`
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "history.cs", Data: []byte(source)}, CompileOptions{
		From:    LanguageCSharp,
		To:      LanguageCSharp,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	history := program.Models[0].States[0].States[1]
	if history.Kind != StateKindShallowHistory || len(history.Transitions) != 1 || history.Transitions[0].Target != "../a" {
		t.Fatalf("history state = %#v", history)
	}
	if !strings.Contains(string(output), `Hsm.ShallowHistory(`) || !strings.Contains(string(output), `"history"`) {
		t.Fatalf("C# output missing history:\n%s", output)
	}
	if !strings.Contains(string(output), `Hsm.Transition(`) || !strings.Contains(string(output), `Hsm.Target("../a")`) {
		t.Fatalf("C# output missing history default transition:\n%s", output)
	}
}

const goPseudostateSource = `package sample

import (
	"context"

	hsm "github.com/stateforward/hsm.go"
)

func approve(ctx context.Context, instance hsm.Instance, event hsm.Event) bool {
	return true
}

var PseudoModel = hsm.Define(
	"Pseudo",
	hsm.Operation("canResume", approve),
	hsm.Operation("restore", approve),
	hsm.Initial(hsm.Target("choice")),
	hsm.Choice("choice", hsm.Transition(hsm.Target("../closed"))),
	hsm.State("closed",
		hsm.Initial(hsm.Target("a")),
		hsm.State("a"),
		hsm.ShallowHistory("history", hsm.Target("../a"), hsm.Guard("canResume"), hsm.Effect("restore")),
		hsm.DeepHistory("deep", hsm.Target("../a")),
		hsm.Transition(hsm.On("finish"), hsm.Target("../done")),
	),
	hsm.Final("done"),
)
`

const tsPseudostateSource = `import * as hsm from "@stateforward/hsm.ts";

const approve = () => true;

export const PseudoModel = hsm.Define(
  "Pseudo",
  hsm.Operation("canResume", approve),
  hsm.Operation("restore", approve),
  hsm.Initial(hsm.Target("choice")),
  hsm.Choice("choice", hsm.Transition(hsm.Target("../closed"))),
  hsm.State(
    "closed",
    hsm.Initial(hsm.Target("a")),
    hsm.State("a"),
    hsm.ShallowHistory("history", hsm.Target("../a"), hsm.Guard("canResume"), hsm.Effect("restore")),
    hsm.DeepHistory("deep", hsm.Target("../a")),
    hsm.Transition(hsm.On("finish"), hsm.Target("../done")),
  ),
  hsm.Final("done"),
);
`

const jsPseudostateSource = `import * as hsm from "@stateforward/hsm";

const approve = () => true;

export const PseudoModel = hsm.Define(
  "Pseudo",
  hsm.Operation("canResume", approve),
  hsm.Operation("restore", approve),
  hsm.Initial(hsm.Target("choice")),
  hsm.Choice("choice", hsm.Transition(hsm.Target("../closed"))),
  hsm.State(
    "closed",
    hsm.Initial(hsm.Target("a")),
    hsm.State("a"),
    hsm.ShallowHistory("history", hsm.Target("../a"), hsm.Guard("canResume"), hsm.Effect("restore")),
    hsm.DeepHistory("deep", hsm.Target("../a")),
    hsm.Transition(hsm.On("finish"), hsm.Target("../done")),
  ),
  hsm.Final("done"),
);
`

const pythonPseudostateSource = `import hsm

def approve(ctx, instance, event):
    return True

pseudo_model = hsm.Define(
    "Pseudo",
    hsm.Operation("canResume", approve),
    hsm.Operation("restore", approve),
    hsm.Initial(hsm.Target("choice")),
    hsm.Choice("choice", hsm.Transition(hsm.Target("../closed"))),
    hsm.State(
        "closed",
        hsm.Initial(hsm.Target("a")),
        hsm.State("a"),
        hsm.ShallowHistory("history", hsm.Target("../a"), hsm.Guard("canResume"), hsm.Effect("restore")),
        hsm.DeepHistory("deep", hsm.Target("../a")),
        hsm.Transition(hsm.On("finish"), hsm.Target("../done")),
    ),
    hsm.Final("done"),
)
`

const csharpPseudostateSource = `using Stateforward.Hsm;

public static class Sample
{
    private static bool Approve(Context ctx, Instance instance, Event @event) { return true; }

    public static readonly Model PseudoModel = Hsm.Define(
        "Pseudo",
        Hsm.Operation("canResume", Approve),
        Hsm.Operation("restore", Approve),
        Hsm.Initial(Hsm.Target("choice")),
        Hsm.Choice("choice", Hsm.Transition(Hsm.Target("../closed"))),
        Hsm.State("closed",
            Hsm.Initial(Hsm.Target("a")),
            Hsm.State("a"),
            Hsm.ShallowHistory("history", Hsm.Target("../a"), Hsm.Guard("canResume"), Hsm.Effect("restore")),
            Hsm.DeepHistory("deep", Hsm.Target("../a")),
            Hsm.Transition(Hsm.On("finish"), Hsm.Target("../done"))
        ),
        Hsm.Final("done")
    );
}
`

const javaPseudostateSource = `import com.stateforward.hsm.*;

public final class Sample {
    private static boolean approve(Context ctx, Instance instance, Event event) { return true; }

    static final Model PseudoModel = Hsm.Define(
        "Pseudo",
        Hsm.Operation("canResume", Sample::approve),
        Hsm.Operation("restore", Sample::approve),
        Hsm.Initial(Hsm.Target("choice")),
        Hsm.Choice("choice", Hsm.Transition(Hsm.Target("../closed"))),
        Hsm.State("closed",
            Hsm.Initial(Hsm.Target("a")),
            Hsm.State("a"),
            Hsm.ShallowHistory("history", Hsm.Target("../a"), Hsm.Guard("canResume"), Hsm.Effect("restore")),
            Hsm.DeepHistory("deep", Hsm.Target("../a")),
            Hsm.Transition(Hsm.On("finish"), Hsm.Target("../done"))
        ),
        Hsm.Final("done")
    );
}
`

const cppPseudostateSource = `#include "hsm/hsm.hpp"

using namespace hsm;

static auto approve = [](auto& ctx, auto& instance, const auto& event) { return true; };

static auto PseudoModel = define(
  "Pseudo",
  operation("canResume", approve),
  operation("restore", approve),
  initial(target("choice")),
  choice("choice", transition(target("../closed"))),
  state("closed",
    initial(target("a")),
    state("a"),
    shallow_history("history", target("../a"), guard("canResume"), effect("restore")),
    deep_history("deep", target("../a")),
    transition(on("finish"), target("../done"))
  ),
  final("done")
);
`

const dartPseudostateSource = `import 'package:hsm/hsm.dart';

bool approve(Context ctx, Instance instance, Event event) => true;

final pseudoModel = define('Pseudo', [
  operation('canResume', approve),
  operation('restore', approve),
  initial(target('choice')),
  choice('choice', [
    transition([target('../closed')]),
  ]),
  state('closed', [
    initial(target('a')),
    state('a'),
    shallowHistory('history', [
      target('../a'),
      guard('canResume'),
      effect('restore'),
    ]),
    deepHistory('deep', [
      target('../a'),
    ]),
    transition([on('finish'), target('../done')]),
  ]),
  finalState('done'),
]);
`

const rustPseudostateSource = `use hsm::*;

fn approve(ctx: &hsm::Context, instance: &hsm::Instance, event: &hsm::Event) -> bool {
    true
}

const PSEUDO: Model = define!("Pseudo",
    operation!("canResume", approve),
    operation!("restore", approve),
    initial!(target!("choice")),
    choice!("choice", transition!(target!("../closed"))),
    state!("closed",
        initial!(target!("a")),
        state!("a"),
        shallow_history!("history", target!("../a"), guard!("canResume"), effect!("restore")),
        deep_history!("deep", target!("../a")),
        transition!(on!("finish"), target!("../done"))
    ),
    final_state!("done")
);
`

const zigPseudostateSource = `const hsm = @import("hsm");

fn approve(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) bool {
    _ = ctx;
    _ = inst;
    _ = event;
    return true;
}

pub const pseudo_model = hsm.define("Pseudo", .{
    hsm.operation("canResume", approve),
    hsm.operation("restore", approve),
    hsm.initial(hsm.target("choice")),
    hsm.choice("choice", .{
        hsm.transition(.{ hsm.target("../closed") }),
    }),
    hsm.state("closed", .{
        hsm.initial(hsm.target("a")),
        hsm.state("a", .{}),
        hsm.history("history", hsm.target("../a"), hsm.guard("canResume"), hsm.effect("restore")),
        hsm.deepHistory("deep", hsm.target("../a")),
        hsm.transition(.{ hsm.on("finish"), hsm.target("../done") }),
    }),
    hsm.final("done"),
});
`

func TestFrontendsPreservePseudostateKindsAndDefaultTransitions(t *testing.T) {
	cases := []struct {
		name     string
		language Language
		path     string
		source   string
	}{
		{name: "csharp", language: LanguageCSharp, path: "Pseudo.cs", source: csharpPseudostateSource},
		{name: "cpp", language: LanguageCPP, path: "pseudo.cpp", source: cppPseudostateSource},
		{name: "dart", language: LanguageDart, path: "pseudo.dart", source: dartPseudostateSource},
		{name: "go", language: LanguageGo, path: "pseudo.go", source: goPseudostateSource},
		{name: "java", language: LanguageJava, path: "Pseudo.java", source: javaPseudostateSource},
		{name: "javascript", language: LanguageJS, path: "pseudo.js", source: jsPseudostateSource},
		{name: "python", language: LanguagePython, path: "pseudo.py", source: pythonPseudostateSource},
		{name: "rust", language: LanguageRust, path: "pseudo.rs", source: rustPseudostateSource},
		{name: "typescript", language: LanguageTS, path: "pseudo.ts", source: tsPseudostateSource},
		{name: "zig", language: LanguageZig, path: "pseudo.zig", source: zigPseudostateSource},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			frontend := NewCompiler().frontends[tc.language]
			program, err := frontend.Parse(context.Background(), SourceInput{Path: tc.path, Data: []byte(tc.source)})
			if err != nil {
				t.Fatal(err)
			}
			if err := Validate(program); err != nil {
				t.Fatal(err)
			}
			assertPseudostateModel(t, program)
		})
	}
}

func TestBackendsPreserveHistoryDefaultGuardAndEffects(t *testing.T) {
	cases := []struct {
		name     string
		language Language
		want     []string
	}{
		{
			name:     "csharp",
			language: LanguageCSharp,
			want: []string{
				`Hsm.ShallowHistory(`,
				`Hsm.Call(ctx, instance, "canResume") is true`,
				`Hsm.Call(ctx, instance, "restore")`,
			},
		},
		{
			name:     "cpp",
			language: LanguageCPP,
			want: []string{
				`hsm::shallow_history("history"`,
				`hsm::guard("canResume")`,
				`hsm::effect("restore")`,
			},
		},
		{
			name:     "dart",
			language: LanguageDart,
			want: []string{
				`shallowHistory(`,
				`guard("canResume")`,
				`effect("restore")`,
			},
		},
		{
			name:     "go",
			language: LanguageGo,
			want: []string{
				`hsm.ShallowHistory(`,
				`hsm.Guard("canResume")`,
				`hsm.Effect("restore")`,
			},
		},
		{
			name:     "java",
			language: LanguageJava,
			want: []string{
				`Hsm.ShallowHistory("history"`,
				`Hsm.Call(ctx, instance, "canResume")`,
				`Hsm.Call(ctx, instance, "restore")`,
			},
		},
		{
			name:     "javascript",
			language: LanguageJS,
			want: []string{
				`hsm.ShallowHistory(`,
				`hsm.Guard("canResume")`,
				`hsm.Effect("restore")`,
			},
		},
		{
			name:     "python",
			language: LanguagePython,
			want: []string{
				`hsm.ShallowHistory(`,
				`hsm.Guard("canResume")`,
				`hsm.Effect("restore")`,
			},
		},
		{
			name:     "rust",
			language: LanguageRust,
			want: []string{
				`hsm::shallow_history!("history"`,
				`hsm::guard_operation_ref("canResume")`,
				`hsm::effect_operation("restore")`,
			},
		},
		{
			name:     "typescript",
			language: LanguageTS,
			want: []string{
				`hsm.ShallowHistory(`,
				`hsm.Guard("canResume")`,
				`hsm.Effect("restore")`,
			},
		},
		{
			name:     "zig",
			language: LanguageZig,
			want: []string{
				`// History default transition guard/effects preserved for manual porting; hsm.zig history helpers accept only a target.`,
				`// hsm.guard(` + zigOperationReferenceFallbackName(BehaviorGuard, "canResume") + `),`,
				`// hsm.effect(` + zigOperationReferenceFallbackName(BehaviorEffect, "restore") + `),`,
				`hsm.history("history", hsm.target("../a")),`,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "pseudo.go", Data: []byte(goPseudostateSource)}, CompileOptions{
				From:    LanguageGo,
				To:      tc.language,
				Adapter: "none",
			})
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			for _, needle := range tc.want {
				if !strings.Contains(text, needle) {
					t.Fatalf("%s history default output missing %q:\n%s", tc.name, needle, text)
				}
			}
		})
	}
}

func assertPseudostateModel(t *testing.T, program *Program) {
	t.Helper()
	if len(program.Models) != 1 {
		t.Fatalf("models = %#v, want one pseudostate model", program.Models)
	}
	model := program.Models[0]
	if model.Name != "Pseudo" || model.Initializer == nil || model.Initializer.Target != "choice" {
		t.Fatalf("model = %#v", model)
	}
	if len(model.States) != 3 {
		t.Fatalf("root states = %#v, want choice, closed, done", model.States)
	}
	choice := model.States[0]
	if choice.Name != "choice" || choice.Kind != StateKindChoice || len(choice.Transitions) != 1 || choice.Transitions[0].Target != "../closed" {
		t.Fatalf("choice state = %#v", choice)
	}
	closed := model.States[1]
	if closed.Name != "closed" || closed.Kind != StateKindState || closed.Initializer == nil || closed.Initializer.Target != "a" {
		t.Fatalf("closed state = %#v", closed)
	}
	if len(closed.States) != 3 {
		t.Fatalf("closed child states = %#v, want state plus histories", closed.States)
	}
	if closed.States[0].Name != "a" || closed.States[0].Kind != StateKindState {
		t.Fatalf("nested state = %#v", closed.States[0])
	}
	history := closed.States[1]
	if history.Name != "history" || history.Kind != StateKindShallowHistory || len(history.Transitions) != 1 || history.Transitions[0].Target != "../a" {
		t.Fatalf("shallow history = %#v", history)
	}
	if history.Transitions[0].Guard == nil || history.Transitions[0].Guard.Operation != "canResume" {
		t.Fatalf("shallow history guard = %#v", history.Transitions[0].Guard)
	}
	if len(history.Transitions[0].Effects) != 1 || history.Transitions[0].Effects[0].Operation != "restore" {
		t.Fatalf("shallow history effects = %#v", history.Transitions[0].Effects)
	}
	deep := closed.States[2]
	if deep.Name != "deep" || deep.Kind != StateKindDeepHistory || len(deep.Transitions) != 1 || deep.Transitions[0].Target != "../a" {
		t.Fatalf("deep history = %#v", deep)
	}
	done := model.States[2]
	if done.Name != "done" || done.Kind != StateKindFinal {
		t.Fatalf("final state = %#v", done)
	}
}

func TestChoiceStillRequiresOwnedTransition(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:   "model_1",
			Name: "BadChoice",
			Initializer: &Initial{
				OwnerID: "model_1",
				ID:      "initial_1",
				Target:  "choice",
			},
			States: []State{{ID: "state_1", Name: "choice", Kind: StateKindChoice}},
		}},
	}
	err := Validate(program)
	if err == nil || !strings.Contains(err.Error(), `choice "choice" must have at least one transition`) {
		t.Fatalf("Validate() = %v, want choice transition error", err)
	}
}

func TestValidateRejectsNonTransitionPseudostatePartials(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry}},
		Models: []Model{{
			ID:          "model_1",
			Name:        "Pseudostates",
			Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "choice"},
			States: []State{{
				ID:        "state_1",
				Name:      "choice",
				Kind:      StateKindChoice,
				Behaviors: []BehaviorRef{{ID: "entry_1", Kind: BehaviorEntry}},
				Defers:    []string{"later"},
				Initializer: &Initial{
					OwnerID: "state_1",
					ID:      "initial_2",
					Target:  "../done",
				},
				States: []State{{ID: "state_2", Name: "nested", Kind: StateKindState}},
				Transitions: []Transition{{
					OwnerID: "state_1",
					ID:      "transition_1",
					Target:  "../done",
				}},
			}, {
				ID:   "state_3",
				Name: "done",
				Kind: StateKindState,
			}},
		}},
	}

	err := Validate(program)
	if err == nil || !strings.Contains(err.Error(), `choice "choice" must not own behaviors, defers, child states, or initializers`) {
		t.Fatalf("Validate() = %v, want pseudostate partial rejection", err)
	}
}

func TestValidateRejectsFinalStateOwnedParts(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:          "model_1",
			Name:        "Finals",
			Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "done"},
			States: []State{{
				ID:     "state_1",
				Name:   "done",
				Kind:   StateKindFinal,
				Defers: []string{"later"},
				States: []State{{ID: "state_2", Name: "nested", Kind: StateKindState}},
				Range:  SourceRange{StartLine: 1},
			}},
		}},
	}

	err := Validate(program)
	if err == nil || !strings.Contains(err.Error(), `final state "done" must not own behaviors, transitions, defers, child states, or initializers`) {
		t.Fatalf("Validate() = %v, want final state owned part rejection", err)
	}
}
