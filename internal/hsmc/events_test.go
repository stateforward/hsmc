package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestEventDeclarationsAreCompilerOwned(t *testing.T) {
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
    static readonly Event GoEvent = new Event("go");
    public static readonly Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed", Hsm.Transition(Hsm.On(GoEvent), Hsm.Target("../open"))),
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

static constexpr auto go_event = "go";

static constexpr auto DoorModel = hsm::define(
  "Door",
  hsm::initial(hsm::target("closed")),
  hsm::state("closed", hsm::transition(hsm::on(go_event), hsm::target("../open"))),
  hsm::state("open")
);
`,
		},
		{
			name:     "dart",
			language: LanguageDart,
			path:     "door.dart",
			source: `import 'package:hsm/hsm.dart';

final goEvent = Event(name: 'go');

final doorModel = define('Door', [
  initial(target('closed')),
  state('closed', [
    transition([
      on(goEvent.name),
      target('../open'),
    ]),
  ]),
  state('open'),
]);
`,
		},
		{
			name:     "go",
			language: LanguageGo,
			path:     "door.go",
			source: `package sample

import hsm "github.com/stateforward/hsm.go"

var GoEvent = hsm.Event{Name: "go"}

var DoorModel = hsm.Define(
	"Door",
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed", hsm.Transition(hsm.On(GoEvent), hsm.Target("../open"))),
	hsm.State("open"),
)
`,
		},
		{
			name:     "java",
			language: LanguageJava,
			path:     "Door.java",
			source: `import com.stateforward.hsm.*;

public final class Sample {
    static final Event GoEvent = new Event("go");
    static final Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed", Hsm.Transition(Hsm.On(GoEvent), Hsm.Target("../open"))),
        Hsm.State("open")
    );
}
`,
		},
		{
			name:     "javascript",
			language: LanguageJS,
			path:     "door.js",
			source: `import * as hsm from "@stateforward/hsm";

const goEvent = hsm.Event("go");

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Transition(hsm.On(goEvent), hsm.Target("../open"))),
  hsm.State("open"),
);
`,
		},
		{
			name:     "python",
			language: LanguagePython,
			path:     "door.py",
			source: `import hsm

go_event = hsm.Event(name="go")

model = hsm.Define(
    "Door",
    hsm.Initial(hsm.Target("closed")),
    hsm.State("closed", hsm.Transition(hsm.On(go_event), hsm.Target("../open"))),
    hsm.State("open"),
)
`,
		},
		{
			name:     "rust",
			language: LanguageRust,
			path:     "door.rs",
			source: `use hsm::*;

const goEvent: &str = "go";

const DOOR: hsm::Model = define!("Door",
    initial!(target!("closed")),
    state!("closed", transition!(on!(goEvent), target!("../open"))),
    state!("open")
);
`,
		},
		{
			name:     "typescript",
			language: LanguageTS,
			path:     "door.ts",
			source: `import * as hsm from "@stateforward/hsm.ts";

const goEvent = hsm.Event("go");

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Transition(hsm.On(goEvent), hsm.Target("../open"))),
  hsm.State("open"),
);
`,
		},
		{
			name:     "zig",
			language: LanguageZig,
			path:     "door.zig",
			source: `const hsm = @import("hsm");

pub const go_event = "go";

pub const door_model = hsm.define("Door", .{
    hsm.initial(hsm.target("closed")),
    hsm.state("closed", .{
        hsm.transition(.{
            hsm.on(go_event),
            hsm.target("../open"),
        }),
    }),
    hsm.state("open", .{}),
});
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program, err := NewCompiler().frontends[tc.language].Parse(context.Background(), SourceInput{Path: tc.path, Data: []byte(tc.source)})
			if err != nil {
				t.Fatal(err)
			}
			if len(program.Events) != 1 || program.Events[0].Name != "go" {
				t.Fatalf("events = %#v", program.Events)
			}
			sourceSymbol := program.Events[0].Symbol
			if len(program.Globals) != 0 {
				t.Fatalf("event declaration should not remain mutable global: %#v", program.Globals)
			}
			trigger := program.Models[0].States[0].Transitions[0].Trigger
			if trigger == nil || len(trigger.Values) != 1 || trigger.Values[0].EventSymbol != sourceSymbol {
				t.Fatalf("trigger = %#v", trigger)
			}

			goOutput, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: tc.path, Data: []byte(tc.source)}, CompileOptions{
				From:    tc.language,
				To:      LanguageGo,
				Adapter: "none",
			})
			if err != nil {
				t.Fatal(err)
			}
			goText := string(goOutput)
			for _, needle := range []string{`var GoEvent = hsm.Event{Name: "go"}`, `hsm.On(GoEvent)`} {
				if !strings.Contains(goText, needle) {
					t.Fatalf("Go output missing %q:\n%s", needle, goText)
				}
			}

			jsOutput, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: tc.path, Data: []byte(tc.source)}, CompileOptions{
				From:    tc.language,
				To:      LanguageJS,
				Adapter: "none",
			})
			if err != nil {
				t.Fatal(err)
			}
			jsText := string(jsOutput)
			for _, needle := range []string{`const goEvent = hsm.Event("go");`, `hsm.On(goEvent)`} {
				if !strings.Contains(jsText, needle) {
					t.Fatalf("JS output missing %q:\n%s", needle, jsText)
				}
			}

			pythonOutput, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: tc.path, Data: []byte(tc.source)}, CompileOptions{
				From:    tc.language,
				To:      LanguagePython,
				Adapter: "none",
			})
			if err != nil {
				t.Fatal(err)
			}
			pythonText := string(pythonOutput)
			for _, needle := range []string{`go_event = hsm.Event(name="go")`, `hsm.On(go_event)`} {
				if !strings.Contains(pythonText, needle) {
					t.Fatalf("Python output missing %q:\n%s", needle, pythonText)
				}
			}
		})
	}
}

func TestBackendsReferenceCompilerOwnedEventSymbolsInTriggers(t *testing.T) {
	program := &Program{
		Events: []EventDecl{{
			ID:     "event_1",
			Symbol: "GoEvent",
			Name:   "go",
		}},
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			States: []State{{
				ID:   "state_1",
				Name: "closed",
				Kind: StateKindState,
				Transitions: []Transition{{
					ID: "transition_1",
					Trigger: &Trigger{
						Kind: TriggerOn,
						Values: []TriggerValue{{
							Value:       "go",
							EventSymbol: "GoEvent",
						}},
					},
					Target: "../open",
				}},
			}},
		}},
	}

	cases := []struct {
		name    string
		backend Backend
		want    []string
	}{
		{name: "csharp", backend: NewCSharpBackend(), want: []string{`private static readonly Event GoEvent = new Event("go");`, `Hsm.On(GoEvent)`}},
		{name: "cpp", backend: NewCPPBackend(), want: []string{`static constexpr auto go_event = GoEvent::name;`, `hsm::on(go_event)`}},
		{name: "dart", backend: NewDartBackend(), want: []string{`final goEvent = Event(name: "go");`, `on(goEvent.name)`}},
		{name: "go", backend: NewGoBackend(), want: []string{`var GoEvent = hsm.Event{Name: "go"}`, `hsm.On(GoEvent)`}},
		{name: "java", backend: NewJavaBackend(), want: []string{`private static final Event GoEvent = new Event("go");`, `Hsm.On(GoEvent)`}},
		{name: "javascript", backend: NewJavaScriptBackend(), want: []string{`const goEvent = hsm.Event("go");`, `hsm.On(goEvent)`}},
		{name: "python", backend: NewPythonBackend(), want: []string{`go_event = hsm.Event(name="go")`, `hsm.On(go_event)`}},
		{name: "rust", backend: NewRustBackend(), want: []string{`const go_event: &str = "go";`, `hsm::on!(go_event)`}},
		{name: "typescript", backend: NewTypeScriptBackend(), want: []string{`const goEvent = hsm.Event("go");`, `hsm.On(goEvent)`}},
		{name: "zig", backend: NewZigBackend(), want: []string{`pub const go_event = "go";`, `hsm.on(go_event)`}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := tc.backend.Emit(context.Background(), program)
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			for _, needle := range tc.want {
				if !strings.Contains(text, needle) {
					t.Fatalf("%s output missing %q:\n%s", tc.name, needle, text)
				}
			}
		})
	}
}

func TestFrontendsResolveEventNameMemberReferences(t *testing.T) {
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
    static readonly Event GoEvent = new Event("go");
    public static readonly Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed",
            Hsm.Defer(GoEvent.Name),
            Hsm.Transition(Hsm.On(GoEvent.Name), Hsm.Target("../open"))
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

struct GoEvent {
  static constexpr auto name = "go";
};

static constexpr auto DoorModel = hsm::define(
  "Door",
  hsm::initial(hsm::target("closed")),
  hsm::state("closed",
    hsm::defer(GoEvent::name),
    hsm::transition(hsm::on(GoEvent::name), hsm::target("../open"))
  ),
  hsm::state("open")
);
`,
		},
		{
			name:     "dart",
			language: LanguageDart,
			path:     "door.dart",
			source: `import 'package:hsm/hsm.dart';

final goEvent = Event(name: 'go');

final doorModel = define('Door', [
  initial(target('closed')),
  state('closed', [
    defer(goEvent.name),
    transition([
      on(goEvent.name),
      target('../open'),
    ]),
  ]),
  state('open'),
]);
`,
		},
		{
			name:     "go",
			language: LanguageGo,
			path:     "door.go",
			source: `package sample

import hsm "github.com/stateforward/hsm.go"

var GoEvent = hsm.Event{Name: "go"}

var DoorModel = hsm.Define(
	"Door",
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed",
		hsm.Defer(GoEvent.Name),
		hsm.Transition(hsm.On(GoEvent.Name), hsm.Target("../open")),
	),
	hsm.State("open"),
)
`,
		},
		{
			name:     "java",
			language: LanguageJava,
			path:     "Door.java",
			source: `import com.stateforward.hsm.*;

public final class Sample {
    static final Event GoEvent = new Event("go");
    static final Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed",
            Hsm.Defer(GoEvent.Name),
            Hsm.Transition(Hsm.On(GoEvent.Name), Hsm.Target("../open"))
        ),
        Hsm.State("open")
    );
}
`,
		},
		{
			name:     "javascript",
			language: LanguageJS,
			path:     "door.js",
			source: `import * as hsm from "@stateforward/hsm";

const goEvent = hsm.Event("go");

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed",
    hsm.Defer(goEvent.name),
    hsm.Transition(hsm.On(goEvent.name), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`,
		},
		{
			name:     "python",
			language: LanguagePython,
			path:     "door.py",
			source: `import hsm

go_event = hsm.Event(name="go")

model = hsm.Define(
    "Door",
    hsm.Initial(hsm.Target("closed")),
    hsm.State("closed",
        hsm.Defer(go_event.name),
        hsm.Transition(hsm.On(go_event.name), hsm.Target("../open")),
    ),
    hsm.State("open"),
)
`,
		},
		{
			name:     "typescript",
			language: LanguageTS,
			path:     "door.ts",
			source: `import * as hsm from "@stateforward/hsm.ts";

const goEvent = hsm.Event("go");

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed",
    hsm.Defer(goEvent.name),
    hsm.Transition(hsm.On(goEvent.name), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`,
		},
	}

	compiler := NewCompiler()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, program, err := compiler.Compile(context.Background(), SourceInput{Path: tc.path, Data: []byte(tc.source)}, CompileOptions{
				From:    tc.language,
				To:      LanguageGo,
				Adapter: "none",
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(program.Events) != 1 || program.Events[0].Name != "go" {
				t.Fatalf("events = %#v", program.Events)
			}
			sourceSymbol := program.Events[0].Symbol
			state := program.Models[0].States[0]
			if got := strings.Join(state.Defers, ","); got != "go" {
				t.Fatalf("defers = %q, want go", got)
			}
			trigger := state.Transitions[0].Trigger
			if trigger == nil || len(trigger.Values) != 1 || trigger.Values[0].EventSymbol != sourceSymbol {
				t.Fatalf("trigger = %#v, want event symbol %q", trigger, sourceSymbol)
			}
			text := string(output)
			for _, needle := range []string{`var GoEvent = hsm.Event{Name: "go"}`, `hsm.Defer("go")`, `hsm.On(GoEvent)`} {
				if !strings.Contains(text, needle) {
					t.Fatalf("Go output missing %q:\n%s", needle, text)
				}
			}
		})
	}
}

func TestValidateRejectsDuplicateEventSymbols(t *testing.T) {
	program := validEventProgram()
	program.Events = append(program.Events, EventDecl{ID: "event_2", Symbol: "goEvent", Name: "again"})

	err := Validate(program)
	if err == nil {
		t.Fatal("Validate returned nil")
	}
	if !diagnosticsContain(err, `duplicate event symbol "goEvent"`) {
		t.Fatalf("diagnostics = %v", err)
	}
}

func TestValidateRejectsDuplicateEventNames(t *testing.T) {
	program := validEventProgram()
	program.Events = append(program.Events, EventDecl{ID: "event_2", Symbol: "againEvent", Name: "go"})

	err := Validate(program)
	if err == nil {
		t.Fatal("Validate returned nil")
	}
	if !diagnosticsContain(err, `duplicate event name "go"`) {
		t.Fatalf("diagnostics = %v", err)
	}
}

func TestValidateRejectsEmptyAndDuplicateEventIDs(t *testing.T) {
	cases := []struct {
		name   string
		events []EventDecl
		want   string
	}{
		{
			name:   "empty",
			events: []EventDecl{{ID: "", Symbol: "goEvent", Name: "go"}},
			want:   "event id is empty",
		},
		{
			name: "duplicate",
			events: []EventDecl{
				{ID: "event_1", Symbol: "goEvent", Name: "go"},
				{ID: "event_1", Symbol: "againEvent", Name: "again"},
			},
			want: `duplicate event id "event_1"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := validEventProgram()
			program.Events = tc.events

			err := Validate(program)
			if err == nil {
				t.Fatal("Validate returned nil")
			}
			if !diagnosticsContain(err, tc.want) {
				t.Fatalf("diagnostics = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestValidateRejectsMissingEventSymbolReferences(t *testing.T) {
	program := validEventProgram()
	trigger := program.Models[0].States[0].Transitions[0].Trigger
	trigger.Value = "missingEvent"
	trigger.Values = []TriggerValue{{Value: "missingEvent", EventSymbol: "missingEvent"}}

	err := Validate(program)
	if err == nil {
		t.Fatal("Validate returned nil")
	}
	if !diagnosticsContain(err, `event reference "missingEvent" does not reference a declared event`) {
		t.Fatalf("diagnostics = %v", err)
	}
}

func TestValidateRejectsPaddedEventReferences(t *testing.T) {
	cases := []struct {
		name    string
		trigger Trigger
		want    string
	}{
		{
			name:    "event symbol",
			trigger: Trigger{Kind: TriggerOn, Values: []TriggerValue{{Value: "goEvent", EventSymbol: " goEvent"}}},
			want:    `event reference " goEvent" has leading or trailing whitespace`,
		},
		{
			name:    "event value",
			trigger: Trigger{Kind: TriggerOn, Values: []TriggerValue{{Value: " go"}}},
			want:    `on trigger event " go" has leading or trailing whitespace`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := validEventProgram()
			program.Models[0].States[0].Transitions[0].Trigger = &tc.trigger

			err := Validate(program)
			if err == nil {
				t.Fatal("Validate returned nil")
			}
			if !diagnosticsContain(err, tc.want) {
				t.Fatalf("diagnostics = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestValidateRejectsEmptyOnTriggerEvents(t *testing.T) {
	cases := []struct {
		name    string
		trigger Trigger
	}{
		{name: "empty value", trigger: Trigger{Kind: TriggerOn}},
		{name: "empty values item", trigger: Trigger{Kind: TriggerOn, Values: []TriggerValue{{Value: ""}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := validEventProgram()
			program.Models[0].States[0].Transitions[0].Trigger = &tc.trigger

			err := Validate(program)
			if err == nil {
				t.Fatal("Validate returned nil")
			}
			if !diagnosticsContain(err, "on trigger has empty event") {
				t.Fatalf("diagnostics = %v", err)
			}
		})
	}
}

func TestValidateRejectsEventMetadataOnNonOnTriggers(t *testing.T) {
	cases := []struct {
		name    string
		trigger Trigger
		want    string
	}{
		{
			name:    "timer values list",
			trigger: Trigger{Kind: TriggerAfter, Values: []TriggerValue{{Value: "1000"}}},
			want:    `after trigger must not use an event values list`,
		},
		{
			name:    "attribute event symbol",
			trigger: Trigger{Kind: TriggerOnSet, Values: []TriggerValue{{Value: "count", EventSymbol: "goEvent"}}},
			want:    `on_set trigger must not use event symbol metadata`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := validEventProgram()
			program.Models[0].Attributes = []Attribute{{ID: "attribute_1", Name: "count"}}
			program.Models[0].States[0].Transitions[0].Trigger = &tc.trigger

			err := Validate(program)
			if err == nil {
				t.Fatal("Validate returned nil")
			}
			if !diagnosticsContain(err, tc.want) {
				t.Fatalf("diagnostics = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestValidateRejectsMissingExpressionTriggers(t *testing.T) {
	cases := []struct {
		name    string
		trigger Trigger
		want    string
	}{
		{name: "after", trigger: Trigger{Kind: TriggerAfter}, want: "after trigger is missing expression"},
		{name: "every", trigger: Trigger{Kind: TriggerEvery}, want: "every trigger is missing expression"},
		{name: "at", trigger: Trigger{Kind: TriggerAt}, want: "at trigger is missing expression"},
		{name: "when", trigger: Trigger{Kind: TriggerWhen}, want: "when trigger is missing expression"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := validEventProgram()
			program.Models[0].States[0].Transitions[0].Trigger = &tc.trigger

			err := Validate(program)
			if err == nil {
				t.Fatal("Validate returned nil")
			}
			if !diagnosticsContain(err, tc.want) {
				t.Fatalf("diagnostics = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestValidateRejectsInconsistentOnTriggerValueMetadata(t *testing.T) {
	cases := []struct {
		name    string
		trigger Trigger
		want    string
	}{
		{
			name: "stale top-level value",
			trigger: Trigger{
				Kind:   TriggerOn,
				Value:  "stale",
				Values: []TriggerValue{{Value: "goEvent", EventSymbol: "goEvent"}},
			},
			want: `on trigger value "stale" does not match first event value "goEvent"`,
		},
		{
			name: "event symbol marked as string literal",
			trigger: Trigger{
				Kind:   TriggerOn,
				Value:  "goEvent",
				Values: []TriggerValue{{Value: "goEvent", EventSymbol: "goEvent", IsString: true}},
			},
			want: `event reference "goEvent" must not also be marked as a string literal`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := validEventProgram()
			program.Models[0].States[0].Transitions[0].Trigger = &tc.trigger

			err := Validate(program)
			if err == nil {
				t.Fatal("Validate returned nil")
			}
			if !diagnosticsContain(err, tc.want) {
				t.Fatalf("diagnostics = %v, want %q", err, tc.want)
			}
		})
	}
}

func validEventProgram() *Program {
	return &Program{
		Events: []EventDecl{{ID: "event_1", Symbol: "goEvent", Name: "go"}},
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
						Trigger: &Trigger{
							Kind:   TriggerOn,
							Value:  "goEvent",
							Values: []TriggerValue{{Value: "goEvent", EventSymbol: "goEvent"}},
						},
					}},
				},
				{ID: "state_2", Name: "open", Kind: StateKindState},
			},
		}},
	}
}

func diagnosticsContain(err error, needle string) bool {
	if diagnostics, ok := err.(Diagnostics); ok {
		for _, diagnostic := range diagnostics {
			if strings.Contains(diagnostic.Message, needle) {
				return true
			}
		}
	}
	return strings.Contains(err.Error(), needle)
}
