package hsmc

import (
	"context"
	"strings"
	"testing"
)

const tsGroupedTypeUtilitySource = `import * as hsm from "@stateforward/hsm.ts";

export const DoorKind = hsm.MakeKind(),
  ActiveDoorKind = hsm.MakeKind(DoorKind),
  IsActiveDoor = hsm.IsKind(ActiveDoorKind, DoorKind),
  DoorModel = hsm.Define(
    "Door",
    hsm.Initial(hsm.Target("closed")),
    hsm.State("closed"),
  );
`

const tsCamelCaseTypeUtilitySource = `import * as hsm from "@stateforward/hsm.ts";

export const DoorKind = hsm.makeKind(),
  ActiveDoorKind = hsm.makeKind(DoorKind),
  IsActiveDoor = hsm.isKind(ActiveDoorKind, DoorKind),
  DoorModel = hsm.define(
    "Door",
    hsm.initial(hsm.target("closed")),
    hsm.state("closed"),
  );
`

const jsCamelCaseTypeUtilitySource = `import * as hsm from "@stateforward/hsm";

export const DoorKind = hsm.makeKind(),
  ActiveDoorKind = hsm.makeKind(DoorKind),
  IsActiveDoor = hsm.isKind(ActiveDoorKind, DoorKind),
  DoorModel = hsm.define(
    "Door",
    hsm.initial(hsm.target("closed")),
    hsm.state("closed"),
  );
`

const tsRenamedImportTypeUtilitySource = `import {
  MakeKind as HMakeKind,
  IsKind as HIsKind,
  Define as HDefine,
  Initial as HInitial,
  Target as HTarget,
  State as HState,
} from "@stateforward/hsm.ts";

export const DoorKind = HMakeKind(),
  ActiveDoorKind = HMakeKind(DoorKind),
  IsActiveDoor = HIsKind(ActiveDoorKind, DoorKind),
  DoorModel = HDefine(
    "Door",
    HInitial(HTarget("closed")),
    HState("closed"),
  );
`

const jsRenamedImportTypeUtilitySource = `import {
  MakeKind as HMakeKind,
  IsKind as HIsKind,
  Define as HDefine,
  Initial as HInitial,
  Target as HTarget,
  State as HState,
} from "@stateforward/hsm";

export const DoorKind = HMakeKind(),
  ActiveDoorKind = HMakeKind(DoorKind),
  IsActiveDoor = HIsKind(ActiveDoorKind, DoorKind),
  DoorModel = HDefine(
    "Door",
    HInitial(HTarget("closed")),
    HState("closed"),
  );
`

const pythonGroupedTypeUtilitySource = `import hsm

door_kind, active_door_kind, is_active_door, door_model = hsm.make_kind(), hsm.make_kind(door_kind), hsm.is_kind(active_door_kind, door_kind), hsm.Define(
    "Door",
    hsm.Initial(hsm.Target("closed")),
    hsm.State("closed"),
)
`

const pythonRenamedImportTypeUtilitySource = `from hsm import MakeKind as HMakeKind, IsKind as HIsKind, Define as HDefine, Initial as HInitial, Target as HTarget, State as HState

door_kind, active_door_kind, is_active_door, door_model = HMakeKind(), HMakeKind(door_kind), HIsKind(active_door_kind, door_kind), HDefine(
    "Door",
    HInitial(HTarget("closed")),
    HState("closed"),
)
`

const goTypeUtilitySource = `package sample

import hsm "github.com/stateforward/hsm.go"

var (
	DoorKind = hsm.MakeKind()
	ActiveDoorKind = hsm.MakeKind(DoorKind)
	IsActiveDoor = hsm.IsKind(ActiveDoorKind, DoorKind)
	DoorModel = hsm.Define(
		"Door",
		hsm.Initial(hsm.Target("closed")),
		hsm.State("closed"),
	)
)
`

const goAliasedTypeUtilitySource = `package sample

import sfhsm "github.com/stateforward/hsm.go"

var (
	DoorKind = sfhsm.MakeKind()
	ActiveDoorKind = sfhsm.MakeKind(DoorKind)
	IsActiveDoor = sfhsm.IsKind(ActiveDoorKind, DoorKind)
	DoorModel = sfhsm.Define(
		"Door",
		sfhsm.Initial(sfhsm.Target("closed")),
		sfhsm.State("closed"),
	)
)
`

const csharpGroupedTypeUtilitySource = `using Stateforward.Hsm;

public static class Sample
{
    public static readonly object DoorKind = Hsm.MakeKind(),
        ActiveDoorKind = Hsm.MakeKind(DoorKind),
        IsActiveDoor = Hsm.IsKind(ActiveDoorKind, DoorKind),
        DoorModel = Hsm.Define(
            "Door",
            Hsm.Initial(Hsm.Target("closed")),
            Hsm.State("closed")
        );
}
`

const csharpAliasedTypeUtilitySource = `using H = Stateforward.Hsm.Hsm;
using Stateforward.Hsm;

public static class Sample
{
    public static readonly object DoorKind = H.MakeKind(),
        ActiveDoorKind = H.MakeKind(DoorKind),
        IsActiveDoor = H.IsKind(ActiveDoorKind, DoorKind),
        DoorModel = H.Define(
            "Door",
            H.Initial(H.Target("closed")),
            H.State("closed")
        );
}
`

const javaGroupedTypeUtilitySource = `import com.stateforward.hsm.*;

public final class Sample {
    static final Object DoorKind = Hsm.MakeKind(),
        ActiveDoorKind = Hsm.MakeKind(DoorKind),
        IsActiveDoor = Hsm.IsKind(ActiveDoorKind, DoorKind),
        DoorModel = Hsm.Define(
            "Door",
            Hsm.Initial(Hsm.Target("closed")),
            Hsm.State("closed")
        );
}
`

const cppGroupedTypeUtilitySource = `#include "hsm/hsm.hpp"

using namespace hsm;

static auto DoorKind = make_kind(),
  ActiveDoorKind = make_kind(DoorKind),
  IsActiveDoor = is_kind(ActiveDoorKind, DoorKind),
  DoorModel = define(
    "Door",
    initial(target("closed")),
    state("closed")
  );
`

const cppQualifiedTypeUtilitySource = `#include "hsm/hsm.hpp"

static auto DoorKind = hsm::make_kind(),
  ActiveDoorKind = hsm::make_kind(DoorKind),
  IsActiveDoor = hsm::is_kind(ActiveDoorKind, DoorKind),
  DoorModel = hsm::define(
    "Door",
    hsm::initial(hsm::target("closed")),
    hsm::state("closed")
  );
`

const dartGroupedTypeUtilitySource = `import 'package:hsm/hsm.dart';

final doorKind = makeKind(),
  activeDoorKind = makeKind(doorKind),
  isActiveDoor = isKind(activeDoorKind, doorKind),
  doorModel = define('Door', [
    initial(target('closed')),
    state('closed'),
  ]);
`

const dartAliasedTypeUtilitySource = `import 'package:hsm/hsm.dart' as hsm;

final doorKind = hsm.makeKind(),
  activeDoorKind = hsm.makeKind(doorKind),
  isActiveDoor = hsm.isKind(activeDoorKind, doorKind),
  doorModel = hsm.define('Door', [
    hsm.initial(hsm.target('closed')),
    hsm.state('closed'),
  ]);
`

const rustTypeUtilitySource = `use hsm::*;

const DOOR_KIND: Kind = make_kind!();
const ACTIVE_DOOR_KIND: Kind = make_kind!(DOOR_KIND);
const IS_ACTIVE_DOOR: bool = is_kind!(ACTIVE_DOOR_KIND, DOOR_KIND);
const DOOR: Model = define!("Door",
    initial!(target!("closed")),
    state!("closed")
);
`

const zigTypeUtilitySource = `const hsm = @import("hsm");

pub const door_kind = hsm.makeKind();
pub const active_door_kind = hsm.makeKind(door_kind);
pub const is_active_door = hsm.isKind(active_door_kind, door_kind);
pub const door_model = hsm.define("Door", .{
    hsm.initial(hsm.target("closed")),
    hsm.state("closed", .{}),
});
`

func TestTypeUtilitiesArePreservedAsGlobalContext(t *testing.T) {
	cases := []struct {
		name        string
		language    Language
		path        string
		source      string
		wantGlobal  []string
		forbidModel string
	}{
		{
			name:        "typescript grouped",
			language:    LanguageTS,
			path:        "door.ts",
			source:      tsGroupedTypeUtilitySource,
			forbidModel: "Define(",
			wantGlobal: []string{
				`DoorKind = hsm.MakeKind();`,
				`ActiveDoorKind = hsm.MakeKind(DoorKind);`,
				`IsActiveDoor = hsm.IsKind(ActiveDoorKind, DoorKind);`,
			},
		},
		{
			name:        "typescript camelCase grouped",
			language:    LanguageTS,
			path:        "door.ts",
			source:      tsCamelCaseTypeUtilitySource,
			forbidModel: "define(",
			wantGlobal: []string{
				`DoorKind = hsm.makeKind();`,
				`ActiveDoorKind = hsm.makeKind(DoorKind);`,
				`IsActiveDoor = hsm.isKind(ActiveDoorKind, DoorKind);`,
			},
		},
		{
			name:        "javascript camelCase grouped",
			language:    LanguageJS,
			path:        "door.js",
			source:      jsCamelCaseTypeUtilitySource,
			forbidModel: "define(",
			wantGlobal: []string{
				`DoorKind = hsm.makeKind();`,
				`ActiveDoorKind = hsm.makeKind(DoorKind);`,
				`IsActiveDoor = hsm.isKind(ActiveDoorKind, DoorKind);`,
			},
		},
		{
			name:        "typescript renamed imports",
			language:    LanguageTS,
			path:        "door.ts",
			source:      tsRenamedImportTypeUtilitySource,
			forbidModel: "HDefine(",
			wantGlobal: []string{
				`DoorKind = HMakeKind();`,
				`ActiveDoorKind = HMakeKind(DoorKind);`,
				`IsActiveDoor = HIsKind(ActiveDoorKind, DoorKind);`,
			},
		},
		{
			name:        "javascript renamed imports",
			language:    LanguageJS,
			path:        "door.js",
			source:      jsRenamedImportTypeUtilitySource,
			forbidModel: "HDefine(",
			wantGlobal: []string{
				`DoorKind = HMakeKind();`,
				`ActiveDoorKind = HMakeKind(DoorKind);`,
				`IsActiveDoor = HIsKind(ActiveDoorKind, DoorKind);`,
			},
		},
		{
			name:        "python grouped aliases",
			language:    LanguagePython,
			path:        "door.py",
			source:      pythonGroupedTypeUtilitySource,
			forbidModel: "Define(",
			wantGlobal: []string{
				`door_kind = hsm.make_kind()`,
				`active_door_kind = hsm.make_kind(door_kind)`,
				`is_active_door = hsm.is_kind(active_door_kind, door_kind)`,
			},
		},
		{
			name:        "python renamed imports",
			language:    LanguagePython,
			path:        "door.py",
			source:      pythonRenamedImportTypeUtilitySource,
			forbidModel: "HDefine(",
			wantGlobal: []string{
				`door_kind = HMakeKind()`,
				`active_door_kind = HMakeKind(door_kind)`,
				`is_active_door = HIsKind(active_door_kind, door_kind)`,
			},
		},
		{
			name:        "go grouped var",
			language:    LanguageGo,
			path:        "door.go",
			source:      goTypeUtilitySource,
			forbidModel: "Define(",
			wantGlobal: []string{
				`var DoorKind = hsm.MakeKind()`,
				`var ActiveDoorKind = hsm.MakeKind(DoorKind)`,
				`var IsActiveDoor = hsm.IsKind(ActiveDoorKind, DoorKind)`,
			},
		},
		{
			name:        "go aliased runtime import",
			language:    LanguageGo,
			path:        "door.go",
			source:      goAliasedTypeUtilitySource,
			forbidModel: "Define(",
			wantGlobal: []string{
				`var DoorKind = hsm.MakeKind()`,
				`var ActiveDoorKind = hsm.MakeKind(DoorKind)`,
				`var IsActiveDoor = hsm.IsKind(ActiveDoorKind, DoorKind)`,
			},
		},
		{
			name:        "csharp grouped field",
			language:    LanguageCSharp,
			path:        "Door.cs",
			source:      csharpGroupedTypeUtilitySource,
			forbidModel: "Define(",
			wantGlobal: []string{
				`DoorKind = Hsm.MakeKind();`,
				`ActiveDoorKind = Hsm.MakeKind(DoorKind);`,
				`IsActiveDoor = Hsm.IsKind(ActiveDoorKind, DoorKind);`,
			},
		},
		{
			name:        "csharp aliased runtime field",
			language:    LanguageCSharp,
			path:        "Door.cs",
			source:      csharpAliasedTypeUtilitySource,
			forbidModel: "Define(",
			wantGlobal: []string{
				`DoorKind = Hsm.MakeKind();`,
				`ActiveDoorKind = Hsm.MakeKind(DoorKind);`,
				`IsActiveDoor = Hsm.IsKind(ActiveDoorKind, DoorKind);`,
			},
		},
		{
			name:        "java grouped field",
			language:    LanguageJava,
			path:        "Door.java",
			source:      javaGroupedTypeUtilitySource,
			forbidModel: "Define(",
			wantGlobal: []string{
				`DoorKind = Hsm.MakeKind();`,
				`ActiveDoorKind = Hsm.MakeKind(DoorKind);`,
				`IsActiveDoor = Hsm.IsKind(ActiveDoorKind, DoorKind);`,
			},
		},
		{
			name:        "cpp grouped declarator",
			language:    LanguageCPP,
			path:        "door.cpp",
			source:      cppGroupedTypeUtilitySource,
			forbidModel: "define(",
			wantGlobal: []string{
				`DoorKind = make_kind();`,
				`ActiveDoorKind = make_kind(DoorKind);`,
				`IsActiveDoor = is_kind(ActiveDoorKind, DoorKind);`,
			},
		},
		{
			name:        "cpp qualified declarator",
			language:    LanguageCPP,
			path:        "door.cpp",
			source:      cppQualifiedTypeUtilitySource,
			forbidModel: "define(",
			wantGlobal: []string{
				`DoorKind = hsm::make_kind();`,
				`ActiveDoorKind = hsm::make_kind(DoorKind);`,
				`IsActiveDoor = hsm::is_kind(ActiveDoorKind, DoorKind);`,
			},
		},
		{
			name:        "dart grouped declaration",
			language:    LanguageDart,
			path:        "door.dart",
			source:      dartGroupedTypeUtilitySource,
			forbidModel: "define(",
			wantGlobal: []string{
				`doorKind = makeKind();`,
				`activeDoorKind = makeKind(doorKind);`,
				`isActiveDoor = isKind(activeDoorKind, doorKind);`,
			},
		},
		{
			name:        "dart aliased runtime declaration",
			language:    LanguageDart,
			path:        "door.dart",
			source:      dartAliasedTypeUtilitySource,
			forbidModel: "define(",
			wantGlobal: []string{
				`doorKind = hsm.makeKind();`,
				`activeDoorKind = hsm.makeKind(doorKind);`,
				`isActiveDoor = hsm.isKind(activeDoorKind, doorKind);`,
			},
		},
		{
			name:        "rust separate const",
			language:    LanguageRust,
			path:        "door.rs",
			source:      rustTypeUtilitySource,
			forbidModel: "define!",
			wantGlobal: []string{
				`const DOOR_KIND: Kind = make_kind!();`,
				`const ACTIVE_DOOR_KIND: Kind = make_kind!(DOOR_KIND);`,
				`const IS_ACTIVE_DOOR: bool = is_kind!(ACTIVE_DOOR_KIND, DOOR_KIND);`,
			},
		},
		{
			name:        "zig separate const",
			language:    LanguageZig,
			path:        "door.zig",
			source:      zigTypeUtilitySource,
			forbidModel: "define(",
			wantGlobal: []string{
				`pub const door_kind = hsm.makeKind();`,
				`pub const active_door_kind = hsm.makeKind(door_kind);`,
				`pub const is_active_door = hsm.isKind(active_door_kind, door_kind);`,
			},
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
			for _, want := range tc.wantGlobal {
				var matched string
				for _, global := range program.Globals {
					if strings.Contains(global.Code, want) {
						matched = global.Code
						break
					}
				}
				if matched == "" {
					t.Fatalf("globals = %#v, want type utility global containing %q", program.Globals, want)
				}
				if tc.forbidModel != "" && strings.Contains(matched, tc.forbidModel) {
					t.Fatalf("model definition leaked into type utility global:\n%s", matched)
				}
			}
		})
	}
}

func TestNoAdapterTypeUtilitiesAreCommentedForForeignTarget(t *testing.T) {
	cases := []struct {
		name    string
		source  string
		needles []string
	}{
		{
			name:   "typescript PascalCase",
			source: tsGroupedTypeUtilitySource,
			needles: []string{
				`# export const DoorKind = hsm.MakeKind();`,
				`# export const ActiveDoorKind = hsm.MakeKind(DoorKind);`,
				`# export const IsActiveDoor = hsm.IsKind(ActiveDoorKind, DoorKind);`,
			},
		},
		{
			name:   "typescript camelCase",
			source: tsCamelCaseTypeUtilitySource,
			needles: []string{
				`# export const DoorKind = hsm.makeKind();`,
				`# export const ActiveDoorKind = hsm.makeKind(DoorKind);`,
				`# export const IsActiveDoor = hsm.isKind(ActiveDoorKind, DoorKind);`,
			},
		},
		{
			name:   "javascript camelCase",
			source: jsCamelCaseTypeUtilitySource,
			needles: []string{
				`# export const DoorKind = hsm.makeKind();`,
				`# export const ActiveDoorKind = hsm.makeKind(DoorKind);`,
				`# export const IsActiveDoor = hsm.isKind(ActiveDoorKind, DoorKind);`,
			},
		},
		{
			name:   "javascript renamed imports",
			source: jsRenamedImportTypeUtilitySource,
			needles: []string{
				`# export const DoorKind = HMakeKind();`,
				`# export const ActiveDoorKind = HMakeKind(DoorKind);`,
				`# export const IsActiveDoor = HIsKind(ActiveDoorKind, DoorKind);`,
			},
		},
		{
			name:   "dart aliased runtime",
			source: dartAliasedTypeUtilitySource,
			needles: []string{
				`# final doorKind = hsm.makeKind();`,
				`# final activeDoorKind = hsm.makeKind(doorKind);`,
				`# final isActiveDoor = hsm.isKind(activeDoorKind, doorKind);`,
			},
		},
		{
			name:   "csharp aliased runtime",
			source: csharpAliasedTypeUtilitySource,
			needles: []string{
				`# public static readonly object DoorKind = Hsm.MakeKind();`,
				`# public static readonly object ActiveDoorKind = Hsm.MakeKind(DoorKind);`,
				`# public static readonly object IsActiveDoor = Hsm.IsKind(ActiveDoorKind, DoorKind);`,
			},
		},
		{
			name:   "go aliased runtime",
			source: goAliasedTypeUtilitySource,
			needles: []string{
				`# var DoorKind = hsm.MakeKind()`,
				`# var ActiveDoorKind = hsm.MakeKind(DoorKind)`,
				`# var IsActiveDoor = hsm.IsKind(ActiveDoorKind, DoorKind)`,
			},
		},
		{
			name:   "cpp qualified declarator",
			source: cppQualifiedTypeUtilitySource,
			needles: []string{
				`# static auto DoorKind = hsm::make_kind();`,
				`# static auto ActiveDoorKind = hsm::make_kind(DoorKind);`,
				`# static auto IsActiveDoor = hsm::is_kind(ActiveDoorKind, DoorKind);`,
			},
		},
		{
			name:   "rust separate const",
			source: rustTypeUtilitySource,
			needles: []string{
				`# const DOOR_KIND: Kind = make_kind!();`,
				`# const ACTIVE_DOOR_KIND: Kind = make_kind!(DOOR_KIND);`,
				`# const IS_ACTIVE_DOOR: bool = is_kind!(ACTIVE_DOOR_KIND, DOOR_KIND);`,
			},
		},
		{
			name:   "zig separate const",
			source: zigTypeUtilitySource,
			needles: []string{
				`# pub const door_kind = hsm.makeKind();`,
				`# pub const active_door_kind = hsm.makeKind(door_kind);`,
				`# pub const is_active_door = hsm.isKind(active_door_kind, door_kind);`,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			from := LanguageTS
			path := "door.ts"
			if strings.Contains(tc.name, "javascript") {
				from = LanguageJS
				path = "door.js"
			} else if strings.Contains(tc.name, "dart") {
				from = LanguageDart
				path = "door.dart"
			} else if strings.Contains(tc.name, "csharp") {
				from = LanguageCSharp
				path = "Door.cs"
			} else if strings.Contains(tc.name, "go") {
				from = LanguageGo
				path = "door.go"
			} else if strings.Contains(tc.name, "cpp") {
				from = LanguageCPP
				path = "door.cpp"
			} else if strings.Contains(tc.name, "rust") {
				from = LanguageRust
				path = "door.rs"
			} else if strings.Contains(tc.name, "zig") {
				from = LanguageZig
				path = "door.zig"
			}
			output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: path, Data: []byte(tc.source)}, CompileOptions{
				From:    from,
				To:      LanguagePython,
				Adapter: "none",
			})
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			commentLanguage := "typescript"
			if from == LanguageJS {
				commentLanguage = "javascript"
			} else if from == LanguageDart {
				commentLanguage = "dart"
			} else if from == LanguageCSharp {
				commentLanguage = "csharp"
			} else if from == LanguageGo {
				commentLanguage = "go"
			} else if from == LanguageCPP {
				commentLanguage = "cpp"
			} else if from == LanguageRust {
				commentLanguage = "rust"
			} else if from == LanguageZig {
				commentLanguage = "zig"
			}
			commonNeedles := []string{
				`# Original ` + commentLanguage + ` global global_1 preserved for manual porting:`,
				`# Original ` + commentLanguage + ` global global_2 preserved for manual porting:`,
				`# Original ` + commentLanguage + ` global global_3 preserved for manual porting:`,
				`door_model = hsm.Define(`,
			}
			for _, needle := range append(commonNeedles, tc.needles...) {
				if !strings.Contains(text, needle) {
					t.Fatalf("Python output missing %q:\n%s", needle, text)
				}
			}
		})
	}
}

func TestAdapterRequestCarriesTypeUtilityGlobals(t *testing.T) {
	cases := []struct {
		name     string
		language Language
		path     string
		source   string
		want     []string
		target   Language
	}{
		{
			name:     "typescript namespace import",
			language: LanguageTS,
			path:     "door.ts",
			source:   tsGroupedTypeUtilitySource,
			target:   LanguagePython,
			want:     []string{`hsm.MakeKind()`, `hsm.MakeKind(DoorKind)`, `hsm.IsKind(ActiveDoorKind, DoorKind)`},
		},
		{
			name:     "typescript camelCase namespace import",
			language: LanguageTS,
			path:     "door.ts",
			source:   tsCamelCaseTypeUtilitySource,
			target:   LanguagePython,
			want:     []string{`hsm.makeKind()`, `hsm.makeKind(DoorKind)`, `hsm.isKind(ActiveDoorKind, DoorKind)`},
		},
		{
			name:     "javascript camelCase namespace import",
			language: LanguageJS,
			path:     "door.js",
			source:   jsCamelCaseTypeUtilitySource,
			target:   LanguagePython,
			want:     []string{`hsm.makeKind()`, `hsm.makeKind(DoorKind)`, `hsm.isKind(ActiveDoorKind, DoorKind)`},
		},
		{
			name:     "javascript renamed imports",
			language: LanguageJS,
			path:     "door.js",
			source:   jsRenamedImportTypeUtilitySource,
			target:   LanguagePython,
			want:     []string{`HMakeKind()`, `HMakeKind(DoorKind)`, `HIsKind(ActiveDoorKind, DoorKind)`},
		},
		{
			name:     "dart aliased runtime import",
			language: LanguageDart,
			path:     "door.dart",
			source:   dartAliasedTypeUtilitySource,
			target:   LanguagePython,
			want:     []string{`hsm.makeKind()`, `hsm.makeKind(doorKind)`, `hsm.isKind(activeDoorKind, doorKind)`},
		},
		{
			name:     "csharp aliased runtime using",
			language: LanguageCSharp,
			path:     "Door.cs",
			source:   csharpAliasedTypeUtilitySource,
			target:   LanguagePython,
			want:     []string{`Hsm.MakeKind()`, `Hsm.MakeKind(DoorKind)`, `Hsm.IsKind(ActiveDoorKind, DoorKind)`},
		},
		{
			name:     "go aliased runtime import",
			language: LanguageGo,
			path:     "door.go",
			source:   goAliasedTypeUtilitySource,
			target:   LanguagePython,
			want:     []string{`hsm.MakeKind()`, `hsm.MakeKind(DoorKind)`, `hsm.IsKind(ActiveDoorKind, DoorKind)`},
		},
		{
			name:     "python renamed imports",
			language: LanguagePython,
			path:     "door.py",
			source:   pythonRenamedImportTypeUtilitySource,
			target:   LanguageTS,
			want:     []string{`HMakeKind()`, `HMakeKind(door_kind)`, `HIsKind(active_door_kind, door_kind)`},
		},
		{
			name:     "cpp qualified declarator",
			language: LanguageCPP,
			path:     "door.cpp",
			source:   cppQualifiedTypeUtilitySource,
			target:   LanguageTS,
			want:     []string{`hsm::make_kind()`, `hsm::make_kind(DoorKind)`, `hsm::is_kind(ActiveDoorKind, DoorKind)`},
		},
		{
			name:     "rust separate const",
			language: LanguageRust,
			path:     "door.rs",
			source:   rustTypeUtilitySource,
			target:   LanguageTS,
			want:     []string{`make_kind!()`, `make_kind!(DOOR_KIND)`, `is_kind!(ACTIVE_DOOR_KIND, DOOR_KIND)`},
		},
		{
			name:     "zig separate const",
			language: LanguageZig,
			path:     "door.zig",
			source:   zigTypeUtilitySource,
			target:   LanguageTS,
			want:     []string{`hsm.makeKind()`, `hsm.makeKind(door_kind)`, `hsm.isKind(active_door_kind, door_kind)`},
		},
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
			if len(adapter.request.Globals) != 3 {
				t.Fatalf("adapter globals = %#v, want type utility globals", adapter.request.Globals)
			}
			text := adapter.request.Globals[0].Code + "\n" + adapter.request.Globals[1].Code + "\n" + adapter.request.Globals[2].Code
			for _, needle := range tc.want {
				if !strings.Contains(text, needle) {
					t.Fatalf("adapter request missing type utility %q:\n%s", needle, text)
				}
			}
			if strings.Contains(text, "Define(") || strings.Contains(text, "HDefine(") || strings.Contains(text, "define(") {
				t.Fatalf("model definition leaked into adapter type utility globals:\n%s", text)
			}
			if len(adapter.request.Models) != 1 || adapter.request.Models[0].Name != "Door" {
				t.Fatalf("adapter model context changed by type utilities: %#v", adapter.request.Models)
			}
		})
	}
}
