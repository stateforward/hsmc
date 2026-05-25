package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestFrontendsLowerStringDefers(t *testing.T) {
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
        Hsm.State("closed",
            Hsm.Defer("knock", "timeout"),
            Hsm.Transition(Hsm.On("open"), Hsm.Target("../open"))
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

using namespace hsm;

static constexpr auto DoorModel = define(
  "Door",
  initial(target("closed")),
  state("closed",
    defer("knock", "timeout"),
    transition(on("open"), target("../open"))
  ),
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
    defer('knock', 'timeout'),
    transition([
      on('open'),
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
	hsm.State("closed",
		hsm.Defer("knock", "timeout"),
		hsm.Transition(hsm.On("open"), hsm.Target("../open")),
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
    static final Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed",
            Hsm.Defer("knock", "timeout"),
            Hsm.Transition(Hsm.On("open"), Hsm.Target("../open"))
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

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed",
    hsm.Defer("knock", "timeout"),
    hsm.Transition(hsm.On("open"), hsm.Target("../open")),
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

model = hsm.Define(
    "Door",
    hsm.Initial(hsm.Target("closed")),
    hsm.State("closed",
        hsm.Defer("knock", "timeout"),
        hsm.Transition(hsm.On("open"), hsm.Target("../open")),
    ),
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
    state!("closed",
        defer!("knock", "timeout"),
        transition!(on!("open"), target!("../open"))
    ),
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
  hsm.State("closed",
    hsm.Defer("knock", "timeout"),
    hsm.Transition(hsm.On("open"), hsm.Target("../open")),
  ),
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
        hsm.deferEvents(.{"knock", "timeout"}),
        hsm.transition(.{
            hsm.on("open"),
            hsm.target("../open"),
        }),
    }),
    hsm.state("open", .{}),
});
`,
		},
	}
	compiler := NewCompiler()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program, err := compiler.frontends[tc.language].Parse(context.Background(), SourceInput{Path: tc.path, Data: []byte(tc.source)})
			if err != nil {
				t.Fatal(err)
			}
			defers := program.Models[0].States[0].Defers
			if strings.Join(defers, ",") != "knock,timeout" {
				t.Fatalf("defers = %#v", defers)
			}
		})
	}
}

func TestFrontendsLowerTypedEventDefersToEventNames(t *testing.T) {
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
            Hsm.Defer(GoEvent),
            Hsm.Transition(Hsm.On("open"), Hsm.Target("../open"))
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

using namespace hsm;

struct GoEvent {
  static constexpr auto name = "go";
};

static constexpr auto DoorModel = define(
  "Door",
  initial(target("closed")),
  state("closed",
    defer<GoEvent>(),
    transition(on("open"), target("../open"))
  ),
  state("open")
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
      on('open'),
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
		hsm.Defer(GoEvent),
		hsm.Transition(hsm.On("open"), hsm.Target("../open")),
	),
	hsm.State("open"),
)
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
    hsm.Defer(goEvent),
    hsm.Transition(hsm.On("open"), hsm.Target("../open")),
  ),
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
    static final Event GoEvent = new Event("go");
    static final Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed",
            Hsm.Defer(GoEvent),
            Hsm.Transition(Hsm.On("open"), Hsm.Target("../open"))
        ),
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

go_event = hsm.Event(name="go")

model = hsm.Define(
    "Door",
    hsm.Initial(hsm.Target("closed")),
    hsm.State("closed",
        hsm.Defer(go_event),
        hsm.Transition(hsm.On("open"), hsm.Target("../open")),
    ),
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
    state!("closed",
        defer!(goEvent),
        transition!(on!("open"), target!("../open"))
    ),
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
  hsm.State("closed",
    hsm.Defer(goEvent),
    hsm.Transition(hsm.On("open"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`,
		},
		{
			name:     "zig",
			language: LanguageZig,
			path:     "door.zig",
			source: `const hsm = @import("hsm");

pub const goEvent = "go";

pub const door_model = hsm.define("Door", .{
    hsm.initial(hsm.target("closed")),
    hsm.state("closed", .{
        hsm.deferEvents(.{goEvent}),
        hsm.transition(.{
            hsm.on("open"),
            hsm.target("../open"),
        }),
    }),
    hsm.state("open", .{}),
});
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
			if got := strings.Join(program.Models[0].States[0].Defers, ","); got != "go" {
				t.Fatalf("defers = %q, want go", got)
			}
			if !strings.Contains(string(output), `hsm.Defer("go")`) {
				t.Fatalf("Go output missing typed event defer by event name:\n%s", output)
			}
		})
	}
}

func TestBackendsEmitDefers(t *testing.T) {
	source := `package sample

import hsm "github.com/stateforward/hsm.go"

var DoorModel = hsm.Define(
	"Door",
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed",
		hsm.Defer("knock"),
		hsm.Transition(hsm.On("open"), hsm.Target("../open")),
	),
	hsm.State("open"),
)
`
	goOutput, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(source)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(goOutput), `hsm.Defer("knock")`) {
		t.Fatalf("Go output missing defer:\n%s", goOutput)
	}
	tsOutput, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(source)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(tsOutput), `hsm.Defer("knock")`) {
		t.Fatalf("TS output missing defer:\n%s", tsOutput)
	}
}

func TestAllBackendsEmitDefersFromIR(t *testing.T) {
	program := &Program{
		SourceLanguage: LanguageGo,
		PackageName:    "sample",
		Models: []Model{{
			ID:          "model_1",
			Name:        "Door",
			Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
			States: []State{{
				ID:     "state_1",
				Name:   "closed",
				Kind:   StateKindState,
				Defers: []string{"knock", "timeout"},
			}},
		}},
	}
	cases := []struct {
		name string
		emit func(*Program) ([]byte, error)
		want []string
	}{
		{
			name: "csharp",
			emit: func(program *Program) ([]byte, error) {
				return NewCSharpBackend().Emit(context.Background(), program)
			},
			want: []string{`Hsm.Defer("knock")`, `Hsm.Defer("timeout")`},
		},
		{
			name: "cpp",
			emit: func(program *Program) ([]byte, error) {
				return NewCPPBackend().Emit(context.Background(), program)
			},
			want: []string{`hsm::defer<KnockEvent>()`, `hsm::defer<TimeoutEvent>()`},
		},
		{
			name: "dart",
			emit: func(program *Program) ([]byte, error) {
				return NewDartBackend().Emit(context.Background(), program)
			},
			want: []string{`defer("knock")`, `defer("timeout")`},
		},
		{
			name: "go",
			emit: func(program *Program) ([]byte, error) {
				return NewGoBackend().Emit(context.Background(), program)
			},
			want: []string{`hsm.Defer("knock")`, `hsm.Defer("timeout")`},
		},
		{
			name: "javascript",
			emit: func(program *Program) ([]byte, error) {
				return NewJavaScriptBackend().Emit(context.Background(), program)
			},
			want: []string{`hsm.Defer("knock")`, `hsm.Defer("timeout")`},
		},
		{
			name: "java",
			emit: func(program *Program) ([]byte, error) {
				return NewJavaBackend().Emit(context.Background(), program)
			},
			want: []string{`Hsm.Defer("knock")`, `Hsm.Defer("timeout")`},
		},
		{
			name: "json-ir",
			emit: func(program *Program) ([]byte, error) {
				return NewJSONIRBackend().Emit(context.Background(), program)
			},
			want: []string{`"defers": [`, `"knock"`, `"timeout"`},
		},
		{
			name: "python",
			emit: func(program *Program) ([]byte, error) {
				return NewPythonBackend().Emit(context.Background(), program)
			},
			want: []string{`hsm.Defer(hsm.Event(name="knock"))`, `hsm.Defer(hsm.Event(name="timeout"))`},
		},
		{
			name: "rust",
			emit: func(program *Program) ([]byte, error) {
				return NewRustBackend().Emit(context.Background(), program)
			},
			want: []string{`hsm::defer!("knock")`, `hsm::defer!("timeout")`},
		},
		{
			name: "typescript",
			emit: func(program *Program) ([]byte, error) {
				return NewTypeScriptBackend().Emit(context.Background(), program)
			},
			want: []string{`hsm.Defer("knock")`, `hsm.Defer("timeout")`},
		},
		{
			name: "zig",
			emit: func(program *Program) ([]byte, error) {
				return NewZigBackend().Emit(context.Background(), program)
			},
			want: []string{`hsm.deferEvents(.{"knock", "timeout"})`},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := tc.emit(cloneProgram(program))
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			for _, needle := range tc.want {
				if !strings.Contains(text, needle) {
					t.Fatalf("%s output missing defer %q:\n%s", tc.name, needle, text)
				}
			}
		})
	}
}

func TestValidateRejectsEmptyDefers(t *testing.T) {
	program := &Program{Models: []Model{{
		ID:          "model_1",
		Name:        "Door",
		Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
		States:      []State{{ID: "state_1", Name: "closed", Kind: StateKindState, Defers: []string{""}}},
	}}}
	err := Validate(program)
	if err == nil || !strings.Contains(err.Error(), "empty deferred event") {
		t.Fatalf("expected empty deferred event diagnostic, got %v", err)
	}
}

func TestValidateRejectsPaddedDefers(t *testing.T) {
	program := &Program{Models: []Model{{
		ID:          "model_1",
		Name:        "Door",
		Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
		States:      []State{{ID: "state_1", Name: "closed", Kind: StateKindState, Defers: []string{" knock"}}},
	}}}
	err := Validate(program)
	if err == nil || !strings.Contains(err.Error(), `deferred event " knock" has leading or trailing whitespace`) {
		t.Fatalf("expected padded deferred event diagnostic, got %v", err)
	}
}
