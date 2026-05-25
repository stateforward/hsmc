package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestGoFrontendLowersTimeExpressionAsTriggerBehavior(t *testing.T) {
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
	program, err := NewGoFrontend().Parse(context.Background(), SourceInput{Path: "timer.go", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	transition := program.Models[0].States[0].Transitions[0]
	if transition.Trigger == nil || transition.Trigger.Expr == nil {
		t.Fatalf("trigger expression was not lowered: %#v", transition.Trigger)
	}
	if len(program.Behaviors) != 1 || program.Behaviors[0].Kind != BehaviorTrigger || program.Behaviors[0].TriggerKind != TriggerAfter {
		t.Fatalf("trigger behavior not captured: %#v", program.Behaviors)
	}
	AssignTargetABI(program, LanguageTS)
	if program.Behaviors[0].TargetABI == nil || !strings.Contains(program.Behaviors[0].TargetABI.ReturnType, "number") {
		t.Fatalf("trigger ABI not assigned: %#v", program.Behaviors[0].TargetABI)
	}
}

func TestGoBackendUsesTriggerBehaviorReference(t *testing.T) {
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
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, "func Trigger1(ctx context.Context, sm *TimerHSM, event hsm.Event) time.Duration") {
		t.Fatalf("Go output missing trigger function:\n%s", text)
	}
	if !strings.Contains(text, "hsm.After(Trigger1)") {
		t.Fatalf("Go output did not reference trigger function:\n%s", text)
	}
}

func TestAtTriggerRoundTripsThroughBackends(t *testing.T) {
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
			hsm.At(func(ctx context.Context, sm *TimerHSM, event hsm.Event) time.Time {
				return time.Now()
			}),
			hsm.Target("../done"),
		),
	),
	hsm.State("done"),
)
`
	program, err := NewGoFrontend().Parse(context.Background(), SourceInput{Path: "timer.go", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	transition := program.Models[0].States[0].Transitions[0]
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerAt || transition.Trigger.Expr == nil {
		t.Fatalf("At trigger was not lowered: %#v", transition.Trigger)
	}
	AssignTargetABI(program, LanguageGo)
	if program.Behaviors[0].TargetABI == nil || program.Behaviors[0].TargetABI.ReturnType != "time.Time" {
		t.Fatalf("At trigger ABI = %#v, want time.Time", program.Behaviors[0].TargetABI)
	}

	goOutput, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "timer.go", Data: []byte(source)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(goOutput), "hsm.At(Trigger1)") {
		t.Fatalf("Go output missing At behavior reference:\n%s", string(goOutput))
	}

	tsOutput, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "timer.go", Data: []byte(source)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(tsOutput), "hsm.At(trigger1)") {
		t.Fatalf("TS output missing At behavior reference:\n%s", string(tsOutput))
	}
}

func TestFrontendsAndBackendsPreserveMultipleOnEvents(t *testing.T) {
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
    public static readonly Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed", Hsm.Transition(Hsm.On("open", "unlock"), Hsm.Target("../open"))),
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

using namespace hsm;

static auto DoorModel = define(
  "Door",
  initial(target("closed")),
  state("closed", transition(on("open", "unlock"), target("../open"))),
  state("open")
);
`,
		},
		{
			name:     "dart",
			language: LanguageDart,
			path:     "door.dart",
			source: `import 'package:hsm/hsm.dart';

final doorModel = define('Door', [
  initial(target('closed')),
  state('closed', [
    transition([
      on('open', 'unlock'),
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

var DoorModel = hsm.Define(
	"Door",
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed", hsm.Transition(hsm.On("open", "unlock"), hsm.Target("../open"))),
	hsm.State("open"),
)
`,
		},
		{
			name:     "javascript",
			language: LanguageJS,
			path:     "door.js",
			source: `import * as hsm from "@stateforward/hsm";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Transition(hsm.On("open", "unlock"), hsm.Target("../open"))),
  hsm.State("open"),
);
`,
		},
		{
			name:     "java",
			language: LanguageJava,
			path:     "Door.java",
			source: `import com.stateforward.hsm.*;

public final class Sample {
    static final Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed", Hsm.Transition(Hsm.On("open", "unlock"), Hsm.Target("../open"))),
        Hsm.State("open")
    );
}
`,
		},
		{
			name:     "python",
			language: LanguagePython,
			path:     "door.py",
			source: `import hsm

model = hsm.Define(
    "Door",
    hsm.Initial(hsm.Target("closed")),
    hsm.State("closed", hsm.Transition(hsm.On("open", "unlock"), hsm.Target("../open"))),
    hsm.State("open"),
)
`,
		},
		{
			name:     "rust",
			language: LanguageRust,
			path:     "door.rs",
			source: `use hsm::*;

const DOOR: hsm::Model = define!("Door",
    initial!(target!("closed")),
    state!("closed", transition!(on!("open", "unlock"), target!("../open"))),
    state!("open")
);
`,
		},
		{
			name:     "typescript",
			language: LanguageTS,
			path:     "door.ts",
			source: `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Transition(hsm.On("open", "unlock"), hsm.Target("../open"))),
  hsm.State("open"),
);
`,
		},
		{
			name:     "zig",
			language: LanguageZig,
			path:     "door.zig",
			source: `const hsm = @import("hsm");

pub const door_model = hsm.define("Door", .{
    hsm.initial(hsm.target("closed")),
    hsm.state("closed", .{
        hsm.transition(.{
            hsm.on("open", "unlock"),
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
			output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: tc.path, Data: []byte(tc.source)}, CompileOptions{
				From:    tc.language,
				To:      LanguageGo,
				Adapter: "none",
			})
			if err != nil {
				t.Fatal(err)
			}
			trigger := program.Models[0].States[0].Transitions[0].Trigger
			if trigger == nil || len(trigger.Values) != 2 || trigger.Values[0].Value != "open" || trigger.Values[1].Value != "unlock" {
				t.Fatalf("trigger = %#v", trigger)
			}
			if !strings.Contains(string(output), `hsm.On(hsm.Event{Name: "open"}, hsm.Event{Name: "unlock"})`) {
				t.Fatalf("Go output lost multiple On events:\n%s", string(output))
			}
		})
	}
}

func TestFrontendsAndBackendsPreserveExplicitTransitionSource(t *testing.T) {
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
    public static readonly Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed"),
        Hsm.State("open"),
        Hsm.Transition(Hsm.Source("closed"), Hsm.On("open"), Hsm.Target("open"))
    );
}
`,
		},
		{
			name:     "cpp",
			language: LanguageCPP,
			path:     "door.cpp",
			source: `#include "hsm/hsm.hpp"

using namespace hsm;

static auto DoorModel = define(
  "Door",
  initial(target("closed")),
  state("closed"),
  state("open"),
  transition(source("closed"), on("open"), target("open"))
);
`,
		},
		{
			name:     "dart",
			language: LanguageDart,
			path:     "door.dart",
			source: `import 'package:hsm/hsm.dart';

final doorModel = define('Door', [
  initial(target('closed')),
  state('closed'),
  state('open'),
  transition([
    source('closed'),
    on('open'),
    target('open'),
  ]),
]);
`,
		},
		{
			name:     "go",
			language: LanguageGo,
			path:     "door.go",
			source: `package sample

import hsm "github.com/stateforward/hsm.go"

var DoorModel = hsm.Define(
	"Door",
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed"),
	hsm.State("open"),
	hsm.Transition(hsm.Source("closed"), hsm.On("open"), hsm.Target("open")),
)
`,
		},
		{
			name:     "javascript",
			language: LanguageJS,
			path:     "door.js",
			source: `import * as hsm from "@stateforward/hsm";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
  hsm.State("open"),
  hsm.Transition(hsm.Source("closed"), hsm.On("open"), hsm.Target("open")),
);
`,
		},
		{
			name:     "java",
			language: LanguageJava,
			path:     "Door.java",
			source: `import com.stateforward.hsm.*;

public final class Sample {
    static final Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed"),
        Hsm.State("open"),
        Hsm.Transition(Hsm.Source("closed"), Hsm.On("open"), Hsm.Target("open"))
    );
}
`,
		},
		{
			name:     "python",
			language: LanguagePython,
			path:     "door.py",
			source: `import hsm

model = hsm.Define(
    "Door",
    hsm.Initial(hsm.Target("closed")),
    hsm.State("closed"),
    hsm.State("open"),
    hsm.Transition(hsm.Source("closed"), hsm.On("open"), hsm.Target("open")),
)
`,
		},
		{
			name:     "rust",
			language: LanguageRust,
			path:     "door.rs",
			source: `use hsm::*;

const DOOR: hsm::Model = define!("Door",
    initial!(target!("closed")),
    state!("closed"),
    state!("open"),
    transition!(source!("closed"), on!("open"), target!("open"))
);
`,
		},
		{
			name:     "typescript",
			language: LanguageTS,
			path:     "door.ts",
			source: `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
  hsm.State("open"),
  hsm.Transition(hsm.Source("closed"), hsm.On("open"), hsm.Target("open")),
);
`,
		},
		{
			name:     "zig",
			language: LanguageZig,
			path:     "door.zig",
			source: `const hsm = @import("hsm");

pub const door_model = hsm.define("Door", .{
    hsm.initial(hsm.target("closed")),
    hsm.state("closed", .{}),
    hsm.state("open", .{}),
    hsm.transition(.{
        hsm.source("closed"),
        hsm.on("open"),
        hsm.target("open"),
    }),
});
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: tc.path, Data: []byte(tc.source)}, CompileOptions{
				From:    tc.language,
				To:      LanguageGo,
				Adapter: "none",
			})
			if err != nil {
				t.Fatal(err)
			}
			transitions := program.Models[0].Transitions
			if len(transitions) != 1 {
				t.Fatalf("root transitions = %#v", transitions)
			}
			transition := transitions[0]
			if transition.Source != "closed" || transition.Target != "open" {
				t.Fatalf("transition route = source %q target %q, want closed -> open", transition.Source, transition.Target)
			}
			text := string(output)
			if !strings.Contains(text, `hsm.Source("closed")`) || !strings.Contains(text, `hsm.Target("open")`) {
				t.Fatalf("Go output lost explicit transition route:\n%s", text)
			}
		})
	}
}
