package hsmc

import (
	"context"
	"strings"
	"testing"
)

const tsGroupedSnapshotSource = `import * as hsm from "@stateforward/hsm.ts";

export const SnapshotNow = (ctx: hsm.Context, machine: hsm.Instance) => hsm.TakeSnapshot(ctx, machine),
  DoorModel = hsm.Define(
    "Door",
    hsm.Initial(hsm.Target("closed")),
    hsm.State("closed"),
  );
`

const tsCamelCaseSnapshotSource = `import * as hsm from "@stateforward/hsm.ts";

export const SnapshotNow = (ctx: hsm.Context, machine: hsm.Instance) => hsm.takeSnapshot(ctx, machine),
  DoorModel = hsm.define(
    "Door",
    hsm.initial(hsm.target("closed")),
    hsm.state("closed"),
  );
`

const jsCamelCaseSnapshotSource = `import * as hsm from "@stateforward/hsm";

export const SnapshotNow = (ctx, machine) => hsm.takeSnapshot(ctx, machine),
  DoorModel = hsm.define(
    "Door",
    hsm.initial(hsm.target("closed")),
    hsm.state("closed"),
  );
`

const tsRenamedImportSnapshotSource = `import {
  TakeSnapshot as HTakeSnapshot,
  Define as HDefine,
  Initial as HInitial,
  Target as HTarget,
  State as HState,
} from "@stateforward/hsm.ts";

export const SnapshotNow = (ctx, machine) => HTakeSnapshot(ctx, machine),
  DoorModel = HDefine(
    "Door",
    HInitial(HTarget("closed")),
    HState("closed"),
  );
`

const jsRenamedImportSnapshotSource = `import {
  TakeSnapshot as HTakeSnapshot,
  Define as HDefine,
  Initial as HInitial,
  Target as HTarget,
  State as HState,
} from "@stateforward/hsm";

export const SnapshotNow = (ctx, machine) => HTakeSnapshot(ctx, machine),
  DoorModel = HDefine(
    "Door",
    HInitial(HTarget("closed")),
    HState("closed"),
  );
`

const pythonGroupedSnapshotSource = `import hsm

snapshot_now, door_model = hsm.TakeSnapshot(ctx, machine), hsm.Define(
    "Door",
    hsm.Initial(hsm.Target("closed")),
    hsm.State("closed"),
)
`

const pythonSnakeCaseSnapshotSource = `import hsm

snapshot_now, door_model = hsm.take_snapshot(ctx, machine), hsm.define(
    "Door",
    hsm.initial(hsm.target("closed")),
    hsm.state("closed"),
)
`

const pythonRenamedImportSnapshotSource = `from hsm import TakeSnapshot as HTakeSnapshot, Define as HDefine, Initial as HInitial, Target as HTarget, State as HState

snapshot_now, door_model = HTakeSnapshot(ctx, machine), HDefine(
    "Door",
    HInitial(HTarget("closed")),
    HState("closed"),
)
`

const goGroupedSnapshotSource = `package sample

import hsm "github.com/stateforward/hsm.go"

var (
	SnapshotNow = func(ctx hsm.Context, machine hsm.Instance) hsm.Snapshot {
		return hsm.TakeSnapshot(ctx, machine)
	}
	DoorModel = hsm.Define(
		"Door",
		hsm.Initial(hsm.Target("closed")),
		hsm.State("closed"),
	)
)
`

const goAliasedSnapshotSource = `package sample

import sfhsm "github.com/stateforward/hsm.go"

var (
	SnapshotNow = func(ctx sfhsm.Context, machine sfhsm.Instance) sfhsm.Snapshot {
		return sfhsm.TakeSnapshot(ctx, machine)
	}
	DoorModel = sfhsm.Define(
		"Door",
		sfhsm.Initial(sfhsm.Target("closed")),
		sfhsm.State("closed"),
	)
)
`

const csharpGroupedSnapshotSource = `using Stateforward.Hsm;

public static class Sample
{
    public static readonly object SnapshotNow = Hsm.TakeSnapshot(Context.Empty, DoorMachine.Instance),
        DoorModel = Hsm.Define(
            "Door",
            Hsm.Initial(Hsm.Target("closed")),
            Hsm.State("closed")
        );
}
`

const csharpAliasedSnapshotSource = `using H = Stateforward.Hsm.Hsm;
using Stateforward.Hsm;

public static class Sample
{
    public static readonly object SnapshotNow = H.TakeSnapshot(Context.Empty, DoorMachine.Instance),
        DoorModel = H.Define(
            "Door",
            H.Initial(H.Target("closed")),
            H.State("closed")
        );
}
`

const javaGroupedSnapshotSource = `import com.stateforward.hsm.*;

public final class Sample {
    static final Object SnapshotNow = Hsm.TakeSnapshot(ctx, machine),
        DoorModel = Hsm.Define(
            "Door",
            Hsm.Initial(Hsm.Target("closed")),
            Hsm.State("closed")
        );
}
`

const cppGroupedSnapshotSource = `#include "hsm/hsm.hpp"

using namespace hsm;

static auto SnapshotNow = take_snapshot(ctx, machine),
  DoorModel = define(
    "Door",
    initial(target("closed")),
    state("closed")
  );
`

const cppQualifiedSnapshotSource = `#include "hsm/hsm.hpp"

static auto SnapshotNow = hsm::take_snapshot(ctx, machine),
  DoorModel = hsm::define(
    "Door",
    hsm::initial(hsm::target("closed")),
    hsm::state("closed")
  );
`

const dartGroupedSnapshotSource = `import 'package:hsm/hsm.dart';

final snapshotNow = takeSnapshot(ctx, machine),
  doorModel = define('Door', [
    initial(target('closed')),
    state('closed'),
  ]);
`

const dartAliasedSnapshotSource = `import 'package:hsm/hsm.dart' as hsm;

final snapshotNow = hsm.takeSnapshot(ctx, machine),
  doorModel = hsm.define('Door', [
    hsm.initial(hsm.target('closed')),
    hsm.state('closed'),
  ]);
`

const rustSnapshotSource = `use hsm::*;

fn snapshot_now(ctx: &hsm::Context, machine: &hsm::Instance) -> hsm::Snapshot {
    hsm::take_snapshot(ctx, machine)
}

const DOOR: Model = define!("Door",
    initial!(target!("closed")),
    state!("closed")
);
`

const zigSnapshotSource = `const hsm = @import("hsm");

pub const snapshot_now = hsm.takeSnapshot(ctx, machine);
pub const door_model = hsm.define("Door", .{
    hsm.initial(hsm.target("closed")),
    hsm.state("closed", .{}),
});
`

func TestSnapshotHelpersArePreservedAsGlobalContext(t *testing.T) {
	cases := []struct {
		name        string
		language    Language
		path        string
		source      string
		wantGlobal  string
		forbidModel string
	}{
		{
			name:        "typescript grouped",
			language:    LanguageTS,
			path:        "door.ts",
			source:      tsGroupedSnapshotSource,
			wantGlobal:  `SnapshotNow = (ctx: hsm.Context, machine: hsm.Instance) => hsm.TakeSnapshot(ctx, machine);`,
			forbidModel: "Define(",
		},
		{
			name:        "typescript camelCase grouped",
			language:    LanguageTS,
			path:        "door.ts",
			source:      tsCamelCaseSnapshotSource,
			wantGlobal:  `SnapshotNow = (ctx: hsm.Context, machine: hsm.Instance) => hsm.takeSnapshot(ctx, machine);`,
			forbidModel: "define(",
		},
		{
			name:        "javascript camelCase grouped",
			language:    LanguageJS,
			path:        "door.js",
			source:      jsCamelCaseSnapshotSource,
			wantGlobal:  `SnapshotNow = (ctx, machine) => hsm.takeSnapshot(ctx, machine);`,
			forbidModel: "define(",
		},
		{
			name:        "typescript renamed imports",
			language:    LanguageTS,
			path:        "door.ts",
			source:      tsRenamedImportSnapshotSource,
			wantGlobal:  `SnapshotNow = (ctx, machine) => HTakeSnapshot(ctx, machine);`,
			forbidModel: "HDefine(",
		},
		{
			name:        "javascript renamed imports",
			language:    LanguageJS,
			path:        "door.js",
			source:      jsRenamedImportSnapshotSource,
			wantGlobal:  `SnapshotNow = (ctx, machine) => HTakeSnapshot(ctx, machine);`,
			forbidModel: "HDefine(",
		},
		{
			name:        "python grouped",
			language:    LanguagePython,
			path:        "door.py",
			source:      pythonGroupedSnapshotSource,
			wantGlobal:  `snapshot_now = hsm.TakeSnapshot(ctx, machine)`,
			forbidModel: "Define(",
		},
		{
			name:        "python snake_case grouped",
			language:    LanguagePython,
			path:        "door.py",
			source:      pythonSnakeCaseSnapshotSource,
			wantGlobal:  `snapshot_now = hsm.take_snapshot(ctx, machine)`,
			forbidModel: "define(",
		},
		{
			name:        "python renamed imports",
			language:    LanguagePython,
			path:        "door.py",
			source:      pythonRenamedImportSnapshotSource,
			wantGlobal:  `snapshot_now = HTakeSnapshot(ctx, machine)`,
			forbidModel: "HDefine(",
		},
		{
			name:        "go grouped var",
			language:    LanguageGo,
			path:        "door.go",
			source:      goGroupedSnapshotSource,
			wantGlobal:  `var SnapshotNow = func(ctx hsm.Context, machine hsm.Instance) hsm.Snapshot {`,
			forbidModel: "Define(",
		},
		{
			name:        "go aliased runtime import",
			language:    LanguageGo,
			path:        "door.go",
			source:      goAliasedSnapshotSource,
			wantGlobal:  `var SnapshotNow = func(ctx hsm.Context, machine hsm.Instance) hsm.Snapshot {`,
			forbidModel: "Define(",
		},
		{
			name:        "csharp grouped field",
			language:    LanguageCSharp,
			path:        "Door.cs",
			source:      csharpGroupedSnapshotSource,
			wantGlobal:  `SnapshotNow = Hsm.TakeSnapshot(Context.Empty, DoorMachine.Instance);`,
			forbidModel: "Define(",
		},
		{
			name:        "csharp aliased runtime field",
			language:    LanguageCSharp,
			path:        "Door.cs",
			source:      csharpAliasedSnapshotSource,
			wantGlobal:  `SnapshotNow = Hsm.TakeSnapshot(Context.Empty, DoorMachine.Instance);`,
			forbidModel: "Define(",
		},
		{
			name:        "java grouped field",
			language:    LanguageJava,
			path:        "Door.java",
			source:      javaGroupedSnapshotSource,
			wantGlobal:  `SnapshotNow = Hsm.TakeSnapshot(ctx, machine);`,
			forbidModel: "Define(",
		},
		{
			name:        "cpp grouped declarator",
			language:    LanguageCPP,
			path:        "door.cpp",
			source:      cppGroupedSnapshotSource,
			wantGlobal:  `SnapshotNow = take_snapshot(ctx, machine);`,
			forbidModel: "define(",
		},
		{
			name:        "cpp qualified declarator",
			language:    LanguageCPP,
			path:        "door.cpp",
			source:      cppQualifiedSnapshotSource,
			wantGlobal:  `SnapshotNow = hsm::take_snapshot(ctx, machine);`,
			forbidModel: "define(",
		},
		{
			name:        "dart grouped declaration",
			language:    LanguageDart,
			path:        "door.dart",
			source:      dartGroupedSnapshotSource,
			wantGlobal:  `snapshotNow = takeSnapshot(ctx, machine);`,
			forbidModel: "define(",
		},
		{
			name:        "dart aliased runtime declaration",
			language:    LanguageDart,
			path:        "door.dart",
			source:      dartAliasedSnapshotSource,
			wantGlobal:  `snapshotNow = hsm.takeSnapshot(ctx, machine);`,
			forbidModel: "define(",
		},
		{
			name:        "rust function",
			language:    LanguageRust,
			path:        "door.rs",
			source:      rustSnapshotSource,
			wantGlobal:  `hsm::take_snapshot(ctx, machine)`,
			forbidModel: "define!",
		},
		{
			name:        "zig separate const",
			language:    LanguageZig,
			path:        "door.zig",
			source:      zigSnapshotSource,
			wantGlobal:  `pub const snapshot_now = hsm.takeSnapshot(ctx, machine);`,
			forbidModel: "define(",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			frontend := NewCompiler().frontends[tc.language]
			program, err := frontend.Parse(context.Background(), SourceInput{Path: tc.path, Data: []byte(tc.source)})
			if err != nil {
				t.Fatal(err)
			}
			if len(program.Models) != 1 || program.Models[0].Name != "Door" {
				t.Fatalf("models = %#v", program.Models)
			}
			var matched string
			for _, global := range program.Globals {
				if strings.Contains(global.Code, tc.wantGlobal) {
					matched = global.Code
					break
				}
			}
			if matched == "" {
				t.Fatalf("globals = %#v, want snapshot global containing %q", program.Globals, tc.wantGlobal)
			}
			if tc.forbidModel != "" && strings.Contains(matched, tc.forbidModel) {
				t.Fatalf("model definition leaked into snapshot global:\n%s", matched)
			}
		})
	}
}

func TestNoAdapterSnapshotHelperIsCommentedForForeignTarget(t *testing.T) {
	cases := []struct {
		name     string
		source   string
		from     Language
		path     string
		target   Language
		snapshot string
		model    string
	}{
		{
			name:     "typescript PascalCase",
			source:   tsGroupedSnapshotSource,
			from:     LanguageTS,
			path:     "door.ts",
			target:   LanguagePython,
			snapshot: `# export const SnapshotNow = (ctx: hsm.Context, machine: hsm.Instance) => hsm.TakeSnapshot(ctx, machine);`,
			model:    `door_model = hsm.Define(`,
		},
		{
			name:     "typescript camelCase",
			source:   tsCamelCaseSnapshotSource,
			from:     LanguageTS,
			path:     "door.ts",
			target:   LanguagePython,
			snapshot: `# export const SnapshotNow = (ctx: hsm.Context, machine: hsm.Instance) => hsm.takeSnapshot(ctx, machine);`,
			model:    `door_model = hsm.Define(`,
		},
		{
			name:     "javascript camelCase",
			source:   jsCamelCaseSnapshotSource,
			from:     LanguageJS,
			path:     "door.js",
			target:   LanguagePython,
			snapshot: `# export const SnapshotNow = (ctx, machine) => hsm.takeSnapshot(ctx, machine);`,
			model:    `door_model = hsm.Define(`,
		},
		{
			name:     "javascript renamed imports",
			source:   jsRenamedImportSnapshotSource,
			from:     LanguageJS,
			path:     "door.js",
			target:   LanguagePython,
			snapshot: `# export const SnapshotNow = (ctx, machine) => HTakeSnapshot(ctx, machine);`,
			model:    `door_model = hsm.Define(`,
		},
		{
			name:     "dart aliased runtime",
			source:   dartAliasedSnapshotSource,
			from:     LanguageDart,
			path:     "door.dart",
			target:   LanguagePython,
			snapshot: `# final snapshotNow = hsm.takeSnapshot(ctx, machine);`,
			model:    `door_model = hsm.Define(`,
		},
		{
			name:     "csharp aliased runtime",
			source:   csharpAliasedSnapshotSource,
			from:     LanguageCSharp,
			path:     "Door.cs",
			target:   LanguagePython,
			snapshot: `# public static readonly object SnapshotNow = Hsm.TakeSnapshot(Context.Empty, DoorMachine.Instance);`,
			model:    `door_model = hsm.Define(`,
		},
		{
			name:     "go aliased runtime",
			source:   goAliasedSnapshotSource,
			from:     LanguageGo,
			path:     "door.go",
			target:   LanguagePython,
			snapshot: `# var SnapshotNow = func(ctx hsm.Context, machine hsm.Instance) hsm.Snapshot {`,
			model:    `door_model = hsm.Define(`,
		},
		{
			name:     "python snake_case",
			source:   pythonSnakeCaseSnapshotSource,
			from:     LanguagePython,
			path:     "door.py",
			target:   LanguageTS,
			snapshot: `// snapshot_now = hsm.take_snapshot(ctx, machine)`,
			model:    `export const DoorModel = hsm.Define(`,
		},
		{
			name:     "cpp qualified declarator",
			source:   cppQualifiedSnapshotSource,
			from:     LanguageCPP,
			path:     "door.cpp",
			target:   LanguageTS,
			snapshot: `// static auto SnapshotNow = hsm::take_snapshot(ctx, machine);`,
			model:    `export const DoorModel = hsm.Define(`,
		},
		{
			name:     "rust function",
			source:   rustSnapshotSource,
			from:     LanguageRust,
			path:     "door.rs",
			target:   LanguageTS,
			snapshot: `//     hsm::take_snapshot(ctx, machine)`,
			model:    `export const DoorModel = hsm.Define(`,
		},
		{
			name:     "zig separate const",
			source:   zigSnapshotSource,
			from:     LanguageZig,
			path:     "door.zig",
			target:   LanguageTS,
			snapshot: `// pub const snapshot_now = hsm.takeSnapshot(ctx, machine);`,
			model:    `export const DoorModel = hsm.Define(`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: tc.path, Data: []byte(tc.source)}, CompileOptions{
				From:    tc.from,
				To:      tc.target,
				Adapter: "none",
			})
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			commentPrefix := "#"
			if tc.target != LanguagePython {
				commentPrefix = "//"
			}
			for _, needle := range []string{
				commentPrefix + ` Original ` + string(tc.from) + ` global global_1 preserved for manual porting:`,
				tc.snapshot,
				tc.model,
			} {
				if !strings.Contains(text, needle) {
					t.Fatalf("Python output missing %q:\n%s", needle, text)
				}
			}
		})
	}
}

func TestAdapterRequestCarriesSnapshotHelperGlobal(t *testing.T) {
	cases := []struct {
		name     string
		language Language
		path     string
		source   string
		want     string
		target   Language
	}{
		{name: "typescript namespace import", language: LanguageTS, path: "door.ts", source: tsGroupedSnapshotSource, want: `hsm.TakeSnapshot(ctx, machine)`, target: LanguagePython},
		{name: "typescript camelCase namespace import", language: LanguageTS, path: "door.ts", source: tsCamelCaseSnapshotSource, want: `hsm.takeSnapshot(ctx, machine)`, target: LanguagePython},
		{name: "javascript camelCase namespace import", language: LanguageJS, path: "door.js", source: jsCamelCaseSnapshotSource, want: `hsm.takeSnapshot(ctx, machine)`, target: LanguagePython},
		{name: "javascript renamed imports", language: LanguageJS, path: "door.js", source: jsRenamedImportSnapshotSource, want: `HTakeSnapshot(ctx, machine)`, target: LanguagePython},
		{name: "dart aliased runtime import", language: LanguageDart, path: "door.dart", source: dartAliasedSnapshotSource, want: `hsm.takeSnapshot(ctx, machine)`, target: LanguagePython},
		{name: "csharp aliased runtime using", language: LanguageCSharp, path: "Door.cs", source: csharpAliasedSnapshotSource, want: `Hsm.TakeSnapshot(Context.Empty, DoorMachine.Instance)`, target: LanguagePython},
		{name: "go aliased runtime import", language: LanguageGo, path: "door.go", source: goAliasedSnapshotSource, want: `hsm.TakeSnapshot(ctx, machine)`, target: LanguagePython},
		{name: "cpp qualified declarator", language: LanguageCPP, path: "door.cpp", source: cppQualifiedSnapshotSource, want: `hsm::take_snapshot(ctx, machine)`, target: LanguageTS},
		{name: "python snake_case", language: LanguagePython, path: "door.py", source: pythonSnakeCaseSnapshotSource, want: `hsm.take_snapshot(ctx, machine)`, target: LanguageTS},
		{name: "python renamed imports", language: LanguagePython, path: "door.py", source: pythonRenamedImportSnapshotSource, want: `HTakeSnapshot(ctx, machine)`, target: LanguageTS},
		{name: "rust function", language: LanguageRust, path: "door.rs", source: rustSnapshotSource, want: `hsm::take_snapshot(ctx, machine)`, target: LanguageTS},
		{name: "zig separate const", language: LanguageZig, path: "door.zig", source: zigSnapshotSource, want: `hsm.takeSnapshot(ctx, machine)`, target: LanguageTS},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &capturingAdapter{}
			compiler := NewCompiler()
			compiler.RegisterAdapter(adapter)

			_, _, err := compiler.Compile(context.Background(), SourceInput{Path: tc.path, Data: []byte(tc.source)}, CompileOptions{
				From:    tc.language,
				To:      tc.target,
				Adapter: adapter.Name(),
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(adapter.request.Globals) != 1 {
				t.Fatalf("adapter globals = %#v, want snapshot helper global", adapter.request.Globals)
			}
			if !strings.Contains(adapter.request.Globals[0].Code, tc.want) {
				t.Fatalf("adapter request missing snapshot helper %q:\n%s", tc.want, adapter.request.Globals[0].Code)
			}
			if strings.Contains(adapter.request.Globals[0].Code, "Define(") || strings.Contains(adapter.request.Globals[0].Code, "HDefine(") || strings.Contains(adapter.request.Globals[0].Code, "define(") {
				t.Fatalf("model definition leaked into snapshot helper global:\n%s", adapter.request.Globals[0].Code)
			}
			if len(adapter.request.Models) != 1 || adapter.request.Models[0].Name != "Door" {
				t.Fatalf("adapter model context changed by snapshot helper: %#v", adapter.request.Models)
			}
		})
	}
}
