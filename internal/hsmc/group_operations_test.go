package hsmc

import (
	"context"
	"strings"
	"testing"
)

const tsGroupedMakeGroupSource = `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
), DoorGroup = hsm.MakeGroup("main", DoorModel);
`

const tsCamelCaseMakeGroupSource = `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.define(
  "Door",
  hsm.initial(hsm.target("closed")),
  hsm.state("closed"),
), DoorGroup = hsm.makeGroup("main", DoorModel);
`

const jsCamelCaseMakeGroupSource = `import * as hsm from "@stateforward/hsm";

export const DoorModel = hsm.define(
  "Door",
  hsm.initial(hsm.target("closed")),
  hsm.state("closed"),
), DoorGroup = hsm.makeGroup("main", DoorModel);
`

const tsRenamedImportMakeGroupSource = `import {
  Define as HDefine,
  Initial as HInitial,
  Target as HTarget,
  State as HState,
  MakeGroup as HMakeGroup,
} from "@stateforward/hsm.ts";

export const DoorModel = HDefine(
  "Door",
  HInitial(HTarget("closed")),
  HState("closed"),
), DoorGroup = HMakeGroup("main", DoorModel);
`

const jsRenamedImportMakeGroupSource = `import {
  Define as HDefine,
  Initial as HInitial,
  Target as HTarget,
  State as HState,
  MakeGroup as HMakeGroup,
} from "@stateforward/hsm";

export const DoorModel = HDefine(
  "Door",
  HInitial(HTarget("closed")),
  HState("closed"),
), DoorGroup = HMakeGroup("main", DoorModel);
`

const pythonGroupedMakeGroupSource = `import hsm

door_model, door_group = hsm.Define(
    "Door",
    hsm.Initial(hsm.Target("closed")),
    hsm.State("closed"),
), hsm.make_group("main", door_model)
`

const pythonRenamedImportMakeGroupSource = `from hsm import Define as HDefine, Initial as HInitial, Target as HTarget, State as HState, MakeGroup as HMakeGroup

door_model, door_group = HDefine(
    "Door",
    HInitial(HTarget("closed")),
    HState("closed"),
), HMakeGroup("main", door_model)
`

const goGroupedMakeGroupSource = `package sample

import hsm "github.com/stateforward/hsm.go"

var (
	DoorModel = hsm.Define(
		"Door",
		hsm.Initial(hsm.Target("closed")),
		hsm.State("closed"),
	)
	DoorGroup = hsm.MakeGroup("main", DoorModel)
)
`

const goAliasedMakeGroupSource = `package sample

import sfhsm "github.com/stateforward/hsm.go"

var (
	DoorModel = sfhsm.Define(
		"Door",
		sfhsm.Initial(sfhsm.Target("closed")),
		sfhsm.State("closed"),
	)
	DoorGroup = sfhsm.MakeGroup("main", DoorModel)
)
`

const csharpGroupedMakeGroupSource = `using Stateforward.Hsm;

public static class Sample
{
    public static readonly object DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed")
    ), DoorGroup = Hsm.MakeGroup("main", DoorModel);
}
`

const csharpAliasedMakeGroupSource = `using H = Stateforward.Hsm.Hsm;
using Stateforward.Hsm;

public static class Sample
{
    public static readonly object DoorModel = H.Define(
        "Door",
        H.Initial(H.Target("closed")),
        H.State("closed")
    ), DoorGroup = H.MakeGroup("main", DoorModel);
}
`

const javaGroupedMakeGroupSource = `import com.stateforward.hsm.*;

public final class Sample {
    static final Object DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed")
    ), DoorGroup = Hsm.MakeGroup("main", DoorModel);
}
`

const cppGroupedMakeGroupSource = `#include "hsm/hsm.hpp"

using namespace hsm;

static auto DoorModel = define(
  "Door",
  initial(target("closed")),
  state("closed")
), DoorGroup = make_group("main", DoorModel);
`

const cppQualifiedMakeGroupSource = `#include "hsm/hsm.hpp"

static auto DoorModel = hsm::define(
  "Door",
  hsm::initial(hsm::target("closed")),
  hsm::state("closed")
), DoorGroup = hsm::make_group("main", DoorModel);
`

const dartGroupedMakeGroupSource = `import 'package:hsm/hsm.dart';

final doorModel = define('Door', [
  initial(target('closed')),
  state('closed'),
]), doorGroup = makeGroup('main', doorModel);
`

const dartAliasedMakeGroupSource = `import 'package:hsm/hsm.dart' as hsm;

final doorModel = hsm.define('Door', [
  hsm.initial(hsm.target('closed')),
  hsm.state('closed'),
]), doorGroup = hsm.makeGroup('main', doorModel);
`

const rustMakeGroupSource = `use hsm::*;

const DOOR: Model = define!("Door",
    initial!(target!("closed")),
    state!("closed")
);
const DOOR_GROUP: Group = make_group!("main", DOOR);
`

const zigMakeGroupSource = `const hsm = @import("hsm");

pub const door_model = hsm.define("Door", .{
    hsm.initial(hsm.target("closed")),
    hsm.state("closed", .{}),
});
pub const door_group = hsm.makeGroup("main", door_model);
`

func TestMakeGroupIsPreservedAsGlobalContext(t *testing.T) {
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
			source:      tsGroupedMakeGroupSource,
			wantGlobal:  `DoorGroup = hsm.MakeGroup("main", DoorModel);`,
			forbidModel: "Define(",
		},
		{
			name:        "typescript camelCase grouped",
			language:    LanguageTS,
			path:        "door.ts",
			source:      tsCamelCaseMakeGroupSource,
			wantGlobal:  `DoorGroup = hsm.makeGroup("main", DoorModel);`,
			forbidModel: "define(",
		},
		{
			name:        "javascript camelCase grouped",
			language:    LanguageJS,
			path:        "door.js",
			source:      jsCamelCaseMakeGroupSource,
			wantGlobal:  `DoorGroup = hsm.makeGroup("main", DoorModel);`,
			forbidModel: "define(",
		},
		{
			name:        "typescript renamed imports",
			language:    LanguageTS,
			path:        "door.ts",
			source:      tsRenamedImportMakeGroupSource,
			wantGlobal:  `DoorGroup = HMakeGroup("main", DoorModel);`,
			forbidModel: "HDefine(",
		},
		{
			name:        "javascript renamed imports",
			language:    LanguageJS,
			path:        "door.js",
			source:      jsRenamedImportMakeGroupSource,
			wantGlobal:  `DoorGroup = HMakeGroup("main", DoorModel);`,
			forbidModel: "HDefine(",
		},
		{
			name:        "python grouped alias",
			language:    LanguagePython,
			path:        "door.py",
			source:      pythonGroupedMakeGroupSource,
			wantGlobal:  `door_group = hsm.make_group("main", door_model)`,
			forbidModel: "Define(",
		},
		{
			name:        "python renamed imports",
			language:    LanguagePython,
			path:        "door.py",
			source:      pythonRenamedImportMakeGroupSource,
			wantGlobal:  `door_group = HMakeGroup("main", door_model)`,
			forbidModel: "HDefine(",
		},
		{
			name:        "go grouped var",
			language:    LanguageGo,
			path:        "door.go",
			source:      goGroupedMakeGroupSource,
			wantGlobal:  `var DoorGroup = hsm.MakeGroup("main", DoorModel)`,
			forbidModel: "Define(",
		},
		{
			name:        "go aliased runtime import",
			language:    LanguageGo,
			path:        "door.go",
			source:      goAliasedMakeGroupSource,
			wantGlobal:  `var DoorGroup = hsm.MakeGroup("main", DoorModel)`,
			forbidModel: "Define(",
		},
		{
			name:        "csharp grouped field",
			language:    LanguageCSharp,
			path:        "Door.cs",
			source:      csharpGroupedMakeGroupSource,
			wantGlobal:  `DoorGroup = Hsm.MakeGroup("main", DoorModel);`,
			forbidModel: "Define(",
		},
		{
			name:        "csharp aliased runtime field",
			language:    LanguageCSharp,
			path:        "Door.cs",
			source:      csharpAliasedMakeGroupSource,
			wantGlobal:  `DoorGroup = Hsm.MakeGroup("main", DoorModel);`,
			forbidModel: "Define(",
		},
		{
			name:        "java grouped field",
			language:    LanguageJava,
			path:        "Door.java",
			source:      javaGroupedMakeGroupSource,
			wantGlobal:  `DoorGroup = Hsm.MakeGroup("main", DoorModel);`,
			forbidModel: "Define(",
		},
		{
			name:        "cpp grouped declarator",
			language:    LanguageCPP,
			path:        "door.cpp",
			source:      cppGroupedMakeGroupSource,
			wantGlobal:  `DoorGroup = make_group("main", DoorModel);`,
			forbidModel: "define(",
		},
		{
			name:        "cpp qualified declarator",
			language:    LanguageCPP,
			path:        "door.cpp",
			source:      cppQualifiedMakeGroupSource,
			wantGlobal:  `DoorGroup = hsm::make_group("main", DoorModel);`,
			forbidModel: "define(",
		},
		{
			name:        "dart grouped declaration",
			language:    LanguageDart,
			path:        "door.dart",
			source:      dartGroupedMakeGroupSource,
			wantGlobal:  `doorGroup = makeGroup('main', doorModel);`,
			forbidModel: "define(",
		},
		{
			name:        "dart aliased runtime declaration",
			language:    LanguageDart,
			path:        "door.dart",
			source:      dartAliasedMakeGroupSource,
			wantGlobal:  `doorGroup = hsm.makeGroup('main', doorModel);`,
			forbidModel: "define(",
		},
		{
			name:        "rust separate const",
			language:    LanguageRust,
			path:        "door.rs",
			source:      rustMakeGroupSource,
			wantGlobal:  `const DOOR_GROUP: Group = make_group!("main", DOOR);`,
			forbidModel: "define!",
		},
		{
			name:        "zig separate const",
			language:    LanguageZig,
			path:        "door.zig",
			source:      zigMakeGroupSource,
			wantGlobal:  `pub const door_group = hsm.makeGroup("main", door_model);`,
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
			var makeGroupGlobal string
			for _, global := range program.Globals {
				if strings.Contains(global.Code, tc.wantGlobal) {
					makeGroupGlobal = global.Code
					break
				}
			}
			if makeGroupGlobal == "" {
				t.Fatalf("globals = %#v, want MakeGroup global containing %q", program.Globals, tc.wantGlobal)
			}
			if tc.forbidModel != "" && strings.Contains(makeGroupGlobal, tc.forbidModel) {
				t.Fatalf("model definition leaked into MakeGroup global:\n%s", makeGroupGlobal)
			}
		})
	}
}

func TestNoAdapterMakeGroupIsCommentedForForeignTarget(t *testing.T) {
	cases := []struct {
		name        string
		source      string
		groupLine   string
		modelNeedle string
	}{
		{
			name:        "typescript PascalCase",
			source:      tsGroupedMakeGroupSource,
			groupLine:   `# export const DoorGroup = hsm.MakeGroup("main", DoorModel);`,
			modelNeedle: `door_model = hsm.Define(`,
		},
		{
			name:        "typescript camelCase",
			source:      tsCamelCaseMakeGroupSource,
			groupLine:   `# export const DoorGroup = hsm.makeGroup("main", DoorModel);`,
			modelNeedle: `door_model = hsm.Define(`,
		},
		{
			name:        "javascript camelCase",
			source:      jsCamelCaseMakeGroupSource,
			groupLine:   `# export const DoorGroup = hsm.makeGroup("main", DoorModel);`,
			modelNeedle: `door_model = hsm.Define(`,
		},
		{
			name:        "javascript renamed imports",
			source:      jsRenamedImportMakeGroupSource,
			groupLine:   `# export const DoorGroup = HMakeGroup("main", DoorModel);`,
			modelNeedle: `door_model = hsm.Define(`,
		},
		{
			name:        "dart aliased runtime",
			source:      dartAliasedMakeGroupSource,
			groupLine:   `# final doorGroup = hsm.makeGroup('main', doorModel);`,
			modelNeedle: `door_model = hsm.Define(`,
		},
		{
			name:        "csharp aliased runtime",
			source:      csharpAliasedMakeGroupSource,
			groupLine:   `# public static readonly object DoorGroup = Hsm.MakeGroup("main", DoorModel);`,
			modelNeedle: `door_model = hsm.Define(`,
		},
		{
			name:        "go aliased runtime",
			source:      goAliasedMakeGroupSource,
			groupLine:   `# var DoorGroup = hsm.MakeGroup("main", DoorModel)`,
			modelNeedle: `door_model = hsm.Define(`,
		},
		{
			name:        "cpp qualified declarator",
			source:      cppQualifiedMakeGroupSource,
			groupLine:   `# static auto DoorGroup = hsm::make_group("main", DoorModel);`,
			modelNeedle: `door_model = hsm.Define(`,
		},
		{
			name:        "rust separate const",
			source:      rustMakeGroupSource,
			groupLine:   `# const DOOR_GROUP: Group = make_group!("main", DOOR);`,
			modelNeedle: `door_model = hsm.Define(`,
		},
		{
			name:        "zig separate const",
			source:      zigMakeGroupSource,
			groupLine:   `# pub const door_group = hsm.makeGroup("main", door_model);`,
			modelNeedle: `door_model = hsm.Define(`,
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
			for _, needle := range []string{
				`# Original ` + commentLanguage + ` global global_1 preserved for manual porting:`,
				tc.groupLine,
				tc.modelNeedle,
			} {
				if !strings.Contains(text, needle) {
					t.Fatalf("Python output missing %q:\n%s", needle, text)
				}
			}
		})
	}
}

func TestAdapterRequestCarriesMakeGroupGlobal(t *testing.T) {
	cases := []struct {
		name     string
		language Language
		path     string
		source   string
		want     string
		target   Language
	}{
		{name: "typescript namespace import", language: LanguageTS, path: "door.ts", source: tsGroupedMakeGroupSource, want: `hsm.MakeGroup("main", DoorModel)`, target: LanguagePython},
		{name: "typescript camelCase namespace import", language: LanguageTS, path: "door.ts", source: tsCamelCaseMakeGroupSource, want: `hsm.makeGroup("main", DoorModel)`, target: LanguagePython},
		{name: "javascript camelCase namespace import", language: LanguageJS, path: "door.js", source: jsCamelCaseMakeGroupSource, want: `hsm.makeGroup("main", DoorModel)`, target: LanguagePython},
		{name: "typescript renamed import", language: LanguageTS, path: "door.ts", source: tsRenamedImportMakeGroupSource, want: `HMakeGroup("main", DoorModel)`, target: LanguagePython},
		{name: "javascript renamed import", language: LanguageJS, path: "door.js", source: jsRenamedImportMakeGroupSource, want: `HMakeGroup("main", DoorModel)`, target: LanguagePython},
		{name: "dart aliased runtime import", language: LanguageDart, path: "door.dart", source: dartAliasedMakeGroupSource, want: `hsm.makeGroup('main', doorModel)`, target: LanguagePython},
		{name: "csharp aliased runtime using", language: LanguageCSharp, path: "Door.cs", source: csharpAliasedMakeGroupSource, want: `Hsm.MakeGroup("main", DoorModel)`, target: LanguagePython},
		{name: "go aliased runtime import", language: LanguageGo, path: "door.go", source: goAliasedMakeGroupSource, want: `hsm.MakeGroup("main", DoorModel)`, target: LanguagePython},
		{name: "python renamed imports", language: LanguagePython, path: "door.py", source: pythonRenamedImportMakeGroupSource, want: `HMakeGroup("main", door_model)`, target: LanguageTS},
		{name: "cpp qualified declarator", language: LanguageCPP, path: "door.cpp", source: cppQualifiedMakeGroupSource, want: `hsm::make_group("main", DoorModel)`, target: LanguageTS},
		{name: "rust separate const", language: LanguageRust, path: "door.rs", source: rustMakeGroupSource, want: `make_group!("main", DOOR)`, target: LanguageTS},
		{name: "zig separate const", language: LanguageZig, path: "door.zig", source: zigMakeGroupSource, want: `hsm.makeGroup("main", door_model)`, target: LanguageTS},
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
				t.Fatalf("adapter globals = %#v, want MakeGroup global", adapter.request.Globals)
			}
			if !strings.Contains(adapter.request.Globals[0].Code, tc.want) {
				t.Fatalf("adapter request missing MakeGroup global %q:\n%s", tc.want, adapter.request.Globals[0].Code)
			}
			if strings.Contains(adapter.request.Globals[0].Code, "Define(") || strings.Contains(adapter.request.Globals[0].Code, "HDefine(") || strings.Contains(adapter.request.Globals[0].Code, "define(") {
				t.Fatalf("model definition leaked into MakeGroup global:\n%s", adapter.request.Globals[0].Code)
			}
			if len(adapter.request.Models) != 1 || adapter.request.Models[0].Name != "Door" {
				t.Fatalf("adapter model context changed by MakeGroup: %#v", adapter.request.Models)
			}
		})
	}
}
