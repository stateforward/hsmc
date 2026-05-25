package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestAssignMutableImportsAttachesGoImportsToRegions(t *testing.T) {
	program, err := NewGoFrontend().Parse(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)})
	if err != nil {
		t.Fatal(err)
	}
	AssignMutableImports(program)
	if len(program.Imports) != 0 {
		t.Fatalf("expected all Go imports to be attributed to mutable regions, remaining = %#v", program.Imports)
	}
	var fmtOnGlobal bool
	for _, global := range program.Globals {
		for _, imp := range global.Imports {
			if imp.Path == "fmt" {
				fmtOnGlobal = true
			}
			assertNoHSMImport(t, imp)
		}
	}
	if !fmtOnGlobal {
		t.Fatalf("globals did not receive fmt import: %#v", program.Globals)
	}
	var contextOnBehavior bool
	for _, behavior := range program.Behaviors {
		for _, imp := range behavior.Imports {
			if imp.Path == "context" {
				contextOnBehavior = true
			}
			assertNoHSMImport(t, imp)
		}
	}
	if !contextOnBehavior {
		t.Fatalf("behaviors did not receive context import: %#v", program.Behaviors)
	}
}

func TestAssignMutableImportsAttachesGoDotImportsToMutableRegions(t *testing.T) {
	source := `package sample

import (
	"context"
	. "example.com/helpers"
	hsm "github.com/stateforward/hsm.go"
)

var DoorModel = hsm.Define(
	"Door",
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed",
		hsm.Entry(func(ctx context.Context, sm hsm.Instance, event hsm.Event) {
			Record("closed")
		}),
	),
)
`
	program, err := NewGoFrontend().Parse(context.Background(), SourceInput{Path: "door.go", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}

	AssignMutableImports(program)

	if len(program.Imports) != 0 {
		t.Fatalf("expected Go dot import to be attributed to mutable regions, remaining = %#v", program.Imports)
	}
	if len(program.Behaviors) != 1 {
		t.Fatalf("behaviors = %d, want 1", len(program.Behaviors))
	}
	var dotImport, contextImport bool
	for _, imp := range program.Behaviors[0].Imports {
		if imp.Path == "example.com/helpers" && imp.Alias == "." && strings.Join(imp.LocalNames, ",") == "." {
			dotImport = true
		}
		if imp.Path == "context" {
			contextImport = true
		}
	}
	if !dotImport || !contextImport {
		t.Fatalf("behavior imports = %#v, want dot helper and context imports", program.Behaviors[0].Imports)
	}
}

func TestAssignMutableImportsAttachesTypeScriptNamedImports(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";
import { record as rec } from "./format";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => { rec("closed"); })),
);
`
	program, err := NewTypeScriptFrontend().Parse(context.Background(), SourceInput{Path: "door.ts", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	AssignMutableImports(program)
	if len(program.Imports) != 0 {
		t.Fatalf("expected named import to be attributed to behavior, remaining = %#v", program.Imports)
	}
	if len(program.Behaviors) != 1 {
		t.Fatalf("behaviors = %d, want 1", len(program.Behaviors))
	}
	if len(program.Behaviors[0].Imports) != 1 || program.Behaviors[0].Imports[0].Path != "./format" {
		t.Fatalf("behavior imports = %#v", program.Behaviors[0].Imports)
	}
	if got := strings.Join(program.Behaviors[0].Imports[0].LocalNames, ","); got != "rec" {
		t.Fatalf("local names = %q, want rec", got)
	}
}

const dartPlainImportSource = `import 'package:hsm/hsm.dart';
import 'package:sample/format.dart';

final doorModel = define('Door', [
  initial(target('closed')),
  state('closed', [
    entry((ctx, instance, event) {
      record('closed');
    }),
  ]),
]);
`

func TestAssignMutableImportsAttachesDartPlainImportsConservatively(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartPlainImportSource)})
	if err != nil {
		t.Fatal(err)
	}

	AssignMutableImports(program)

	if len(program.Imports) != 0 {
		t.Fatalf("expected Dart plain import to be attributed to behavior, remaining = %#v", program.Imports)
	}
	if len(program.Behaviors) != 1 {
		t.Fatalf("behaviors = %d, want 1", len(program.Behaviors))
	}
	behaviorImports := program.Behaviors[0].Imports
	if len(behaviorImports) != 1 || behaviorImports[0].Path != "package:sample/format.dart" {
		t.Fatalf("behavior imports = %#v", behaviorImports)
	}
	if behaviorImports[0].Alias != "" || len(behaviorImports[0].Specifiers) != 0 {
		t.Fatalf("plain Dart import should stay renderable as plain import, got %#v", behaviorImports[0])
	}
}

func TestAssignMutableImportsAttachesDartShowImports(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartShowImportSource)})
	if err != nil {
		t.Fatal(err)
	}

	AssignMutableImports(program)

	if len(program.Imports) != 0 {
		t.Fatalf("expected Dart show import to be attributed to behavior, remaining = %#v", program.Imports)
	}
	if len(program.Behaviors) != 1 {
		t.Fatalf("behaviors = %d, want 1", len(program.Behaviors))
	}
	behaviorImports := program.Behaviors[0].Imports
	if len(behaviorImports) != 1 || behaviorImports[0].Path != "package:sample/format.dart" {
		t.Fatalf("behavior imports = %#v", behaviorImports)
	}
	if got := strings.Join(behaviorImports[0].LocalNames, ","); got != "record,normalize" {
		t.Fatalf("local names = %q, want record,normalize", got)
	}
	if len(behaviorImports[0].Specifiers) != 2 {
		t.Fatalf("specifier context = %#v, want two shown names", behaviorImports[0].Specifiers)
	}
}

func TestAssignMutableImportsAttachesDartPrefixedShowImports(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartPrefixedShowImportSource)})
	if err != nil {
		t.Fatal(err)
	}

	AssignMutableImports(program)

	if len(program.Imports) != 0 {
		t.Fatalf("expected Dart prefixed show import to be attributed to behavior, remaining = %#v", program.Imports)
	}
	if len(program.Behaviors) != 1 {
		t.Fatalf("behaviors = %d, want 1", len(program.Behaviors))
	}
	behaviorImports := program.Behaviors[0].Imports
	if len(behaviorImports) != 1 || behaviorImports[0].Path != "package:sample/format.dart" {
		t.Fatalf("behavior imports = %#v", behaviorImports)
	}
	if behaviorImports[0].Alias != "fmt" || len(behaviorImports[0].Specifiers) != 2 {
		t.Fatalf("prefixed show import context = %#v", behaviorImports[0])
	}
	if got := strings.Join(behaviorImports[0].LocalNames, ","); got != "fmt" {
		t.Fatalf("local names = %q, want fmt", got)
	}
}

func TestAssignMutableImportsAttachesDartHideImports(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartHideImportSource)})
	if err != nil {
		t.Fatal(err)
	}

	AssignMutableImports(program)

	if len(program.Imports) != 0 {
		t.Fatalf("expected Dart hide import to be attributed to behavior, remaining = %#v", program.Imports)
	}
	if len(program.Behaviors) != 1 {
		t.Fatalf("behaviors = %d, want 1", len(program.Behaviors))
	}
	behaviorImports := program.Behaviors[0].Imports
	if len(behaviorImports) != 1 || behaviorImports[0].Path != "package:sample/format.dart" {
		t.Fatalf("behavior imports = %#v", behaviorImports)
	}
	if strings.Join(behaviorImports[0].Hidden, ",") != "debugRecord,privateNormalize" {
		t.Fatalf("hidden context = %#v", behaviorImports[0].Hidden)
	}
	if len(behaviorImports[0].Specifiers) != 0 {
		t.Fatalf("Dart hide import should not become show specifiers: %#v", behaviorImports[0].Specifiers)
	}
}

func TestAssignMutableImportsAttachesCSharpAliasUsing(t *testing.T) {
	program, err := NewCSharpFrontend().Parse(context.Background(), SourceInput{Path: "door.cs", Data: []byte(csharpAliasUsingSource)})
	if err != nil {
		t.Fatal(err)
	}

	AssignMutableImports(program)

	if len(program.Imports) != 0 {
		t.Fatalf("expected C# alias using to be attributed to behavior, remaining = %#v", program.Imports)
	}
	if len(program.Behaviors) != 1 {
		t.Fatalf("behaviors = %d, want 1", len(program.Behaviors))
	}
	behaviorImports := program.Behaviors[0].Imports
	if len(behaviorImports) != 1 || behaviorImports[0].Path != "Sample.Formatting.Helper" {
		t.Fatalf("behavior imports = %#v", behaviorImports)
	}
	if behaviorImports[0].Alias != "Format" || strings.Join(behaviorImports[0].LocalNames, ",") != "Format" {
		t.Fatalf("alias import context = %#v", behaviorImports[0])
	}
}

func TestAssignMutableImportsAttachesDartDeferredImports(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartDeferredImportSource)})
	if err != nil {
		t.Fatal(err)
	}

	AssignMutableImports(program)

	if len(program.Imports) != 0 {
		t.Fatalf("expected Dart deferred import to be attributed to behavior, remaining = %#v", program.Imports)
	}
	if len(program.Behaviors) != 1 {
		t.Fatalf("behaviors = %d, want 1", len(program.Behaviors))
	}
	behaviorImports := program.Behaviors[0].Imports
	if len(behaviorImports) != 1 || behaviorImports[0].Path != "package:sample/lazy_format.dart" {
		t.Fatalf("behavior imports = %#v", behaviorImports)
	}
	if !behaviorImports[0].Deferred || behaviorImports[0].Alias != "lazyFormat" {
		t.Fatalf("deferred import context = %#v", behaviorImports[0])
	}
}

const csharpStaticHelperUsingSource = `using Stateforward.Hsm;
using static Sample.Formatting.Helper;

public static class Sample
{
    public static readonly Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed",
            Hsm.Entry<Instance>((ctx, instance, @event) => Record("closed"))
        )
    );
}
`

func TestAssignMutableImportsAttachesCSharpStaticHelperUsing(t *testing.T) {
	program, err := NewCSharpFrontend().Parse(context.Background(), SourceInput{Path: "door.cs", Data: []byte(csharpStaticHelperUsingSource)})
	if err != nil {
		t.Fatal(err)
	}

	AssignMutableImports(program)

	if len(program.Imports) != 0 {
		t.Fatalf("expected C# static using to be attributed to behavior, remaining = %#v", program.Imports)
	}
	if len(program.Behaviors) != 1 {
		t.Fatalf("behaviors = %d, want 1", len(program.Behaviors))
	}
	behaviorImports := program.Behaviors[0].Imports
	if len(behaviorImports) != 1 || behaviorImports[0].Path != "Sample.Formatting.Helper" {
		t.Fatalf("behavior imports = %#v", behaviorImports)
	}
	if behaviorImports[0].Alias != "" || strings.Join(behaviorImports[0].LocalNames, ",") != "*" {
		t.Fatalf("static using import context = %#v", behaviorImports[0])
	}
}

func TestAssignMutableImportsAttachesRustGroupedUse(t *testing.T) {
	program, err := NewRustFrontend().Parse(context.Background(), SourceInput{Path: "door.rs", Data: []byte(rustGroupedUseSource)})
	if err != nil {
		t.Fatal(err)
	}

	AssignMutableImports(program)

	if len(program.Imports) != 0 {
		t.Fatalf("expected Rust grouped use to be attributed to behavior, remaining = %#v", program.Imports)
	}
	if len(program.Behaviors) != 1 {
		t.Fatalf("behaviors = %d, want 1", len(program.Behaviors))
	}
	behaviorImports := program.Behaviors[0].Imports
	if len(behaviorImports) != 1 || behaviorImports[0].Path != "crate::helpers" {
		t.Fatalf("behavior imports = %#v", behaviorImports)
	}
	if got := strings.Join(behaviorImports[0].LocalNames, ","); got != "record,norm" {
		t.Fatalf("local names = %q, want record,norm", got)
	}
	if len(behaviorImports[0].Specifiers) != 2 || behaviorImports[0].Specifiers[1].Alias != "norm" {
		t.Fatalf("specifier context = %#v", behaviorImports[0].Specifiers)
	}
}

func TestAssignMutableImportsUsesStructuredESBindings(t *testing.T) {
	program := &Program{
		Imports: []Import{
			{Path: "./default-helper", Default: "format", Language: LanguageTS},
			{Path: "./named-helper", Specifiers: []ImportSpecifier{{Name: "record", Alias: "track"}}, Language: LanguageTS},
			{Path: "./unused-helper", Default: "unused", Language: LanguageTS},
		},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageTS,
			Body:           "format(track('closed'));",
		}},
	}

	AssignMutableImports(program)

	if len(program.Imports) != 1 || program.Imports[0].Path != "./unused-helper" {
		t.Fatalf("remaining structural imports = %#v", program.Imports)
	}
	behaviorImports := program.Behaviors[0].Imports
	if len(behaviorImports) != 2 {
		t.Fatalf("behavior imports = %#v, want default and named imports", behaviorImports)
	}
	if behaviorImports[0].Path != "./default-helper" || strings.Join(behaviorImports[0].LocalNames, ",") != "format" {
		t.Fatalf("default import attribution = %#v", behaviorImports[0])
	}
	if behaviorImports[1].Path != "./named-helper" || strings.Join(behaviorImports[1].LocalNames, ",") != "track" {
		t.Fatalf("named import attribution = %#v", behaviorImports[1])
	}
}

func TestAssignMutableImportsPreservesTypeScriptTypeOnlyImports(t *testing.T) {
	program := &Program{
		Imports: []Import{
			{Path: "./types", TypeOnly: true, Specifiers: []ImportSpecifier{{Name: "DoorSnapshot"}}, Language: LanguageTS},
			{Path: "./metrics", Specifiers: []ImportSpecifier{{Name: "DoorMetrics", TypeOnly: true}, {Name: "recordMetric"}}, Language: LanguageTS},
		},
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageTS,
			Code:     "type SnapshotFormatter = (snapshot: DoorSnapshot, metrics: DoorMetrics) => string;\nconst formatMetric = recordMetric;",
		}},
	}

	AssignMutableImports(program)

	if len(program.Imports) != 0 {
		t.Fatalf("expected type-only import to be attributed to global, remaining = %#v", program.Imports)
	}
	if len(program.Globals[0].Imports) != 2 {
		t.Fatalf("global imports = %#v", program.Globals[0].Imports)
	}
	imp := program.Globals[0].Imports[0]
	if imp.Path != "./types" || !imp.TypeOnly || strings.Join(imp.LocalNames, ",") != "DoorSnapshot" {
		t.Fatalf("type-only global import context = %#v", imp)
	}
	mixed := program.Globals[0].Imports[1]
	if mixed.Path != "./metrics" || len(mixed.Specifiers) != 2 || !mixed.Specifiers[0].TypeOnly || strings.Join(mixed.LocalNames, ",") != "DoorMetrics,recordMetric" {
		t.Fatalf("mixed type-only global import context = %#v", mixed)
	}
}

func TestAssignMutableImportsUsesStructuredPythonBindings(t *testing.T) {
	program := &Program{
		Imports: []Import{
			{Path: "formatting", Specifiers: []ImportSpecifier{{Name: "record", Alias: "rec"}}, Language: LanguagePython},
			{Path: "datetime", Language: LanguagePython},
			{Path: "unused", Language: LanguagePython},
		},
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguagePython,
			Code:     "def entered():\n    return rec(datetime.datetime.now())",
		}},
	}

	AssignMutableImports(program)

	if len(program.Imports) != 1 || program.Imports[0].Path != "unused" {
		t.Fatalf("remaining structural imports = %#v", program.Imports)
	}
	globalImports := program.Globals[0].Imports
	if len(globalImports) != 2 {
		t.Fatalf("global imports = %#v, want named and module imports", globalImports)
	}
	if globalImports[0].Path != "formatting" || strings.Join(globalImports[0].LocalNames, ",") != "rec" {
		t.Fatalf("python from-import attribution = %#v", globalImports[0])
	}
	if globalImports[1].Path != "datetime" || strings.Join(globalImports[1].LocalNames, ",") != "datetime" {
		t.Fatalf("python module import attribution = %#v", globalImports[1])
	}
}

func TestAssignMutableImportsAttachesPythonWildcardImports(t *testing.T) {
	program := &Program{
		Imports: []Import{
			{Path: "helpers", Specifiers: []ImportSpecifier{{Name: "*"}}, Language: LanguagePython},
		},
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguagePython,
			Code:     "def entered():\n    return record('closed')",
		}},
	}

	AssignMutableImports(program)

	if len(program.Imports) != 0 {
		t.Fatalf("expected Python wildcard import to be attributed to mutable region, remaining = %#v", program.Imports)
	}
	globalImports := program.Globals[0].Imports
	if len(globalImports) != 1 || globalImports[0].Path != "helpers" {
		t.Fatalf("global imports = %#v", globalImports)
	}
	if len(globalImports[0].Specifiers) != 1 || globalImports[0].Specifiers[0].Name != "*" {
		t.Fatalf("wildcard specifier context = %#v", globalImports[0].Specifiers)
	}
}

func TestPythonFrontendPreservesNonRuntimeHSMNamedImports(t *testing.T) {
	source := `import hsm, hsm_helpers

def entered(ctx, instance, event):
    hsm_helpers.record("closed")

model = hsm.Define(
    "Door",
    hsm.Initial(hsm.Target("closed")),
    hsm.State("closed", hsm.Entry(entered)),
)
`
	program, err := NewPythonFrontend().Parse(context.Background(), SourceInput{Path: "door.py", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 1 || program.Imports[0].Path != "hsm_helpers" {
		t.Fatalf("imports = %#v, want only non-runtime hsm_helpers import", program.Imports)
	}

	AssignMutableImports(program)

	if len(program.Imports) != 0 {
		t.Fatalf("expected helper import to be attributed, remaining = %#v", program.Imports)
	}
	if len(program.Behaviors) != 1 || len(program.Behaviors[0].Imports) != 1 || program.Behaviors[0].Imports[0].Path != "hsm_helpers" {
		t.Fatalf("behavior imports = %#v", program.Behaviors)
	}
}

func TestAssignMutableImportsAttachesJavaImportsToRegions(t *testing.T) {
	source := `import com.stateforward.hsm.*;
import com.example.Format;
import com.example.Track;

public final class DoorHsm {
  static String record(String value) {
    return Format.record(value);
  }

  static final Model DoorModel = Hsm.Define(
    "Door",
    Hsm.Initial(Hsm.Target("closed")),
    Hsm.State("closed", Hsm.Entry((ctx, instance, event) -> {
      Track.record(record("closed"));
    }))
  );
}`
	program, err := NewJavaFrontend().Parse(context.Background(), SourceInput{Path: "DoorHsm.java", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	AssignMutableImports(program)
	if len(program.Imports) != 0 {
		t.Fatalf("expected all Java helper imports to be attributed, remaining = %#v", program.Imports)
	}
	if len(program.Globals) != 1 || len(program.Globals[0].Imports) != 1 || program.Globals[0].Imports[0].Path != "com.example.Format" {
		t.Fatalf("global imports = %#v", program.Globals)
	}
	if len(program.Behaviors) != 1 || len(program.Behaviors[0].Imports) != 1 {
		t.Fatalf("behavior imports = %#v", program.Behaviors)
	}
	if program.Behaviors[0].Imports[0].Path != "com.example.Track" {
		t.Fatalf("behavior imports = %#v, want Track context", program.Behaviors[0].Imports)
	}
	for _, imp := range program.Globals[0].Imports {
		assertNoHSMImport(t, imp)
	}
	for _, imp := range program.Behaviors[0].Imports {
		assertNoHSMImport(t, imp)
	}
}

func TestAssignMutableImportsAttachesJavaWildcardImports(t *testing.T) {
	program, err := NewJavaFrontend().Parse(context.Background(), SourceInput{Path: "DoorHsm.java", Data: []byte(javaWildcardImportSource)})
	if err != nil {
		t.Fatal(err)
	}

	AssignMutableImports(program)

	if len(program.Imports) != 0 {
		t.Fatalf("expected Java wildcard import to be attributed to behavior, remaining = %#v", program.Imports)
	}
	if len(program.Behaviors) != 1 {
		t.Fatalf("behaviors = %d, want 1", len(program.Behaviors))
	}
	behaviorImports := program.Behaviors[0].Imports
	if len(behaviorImports) != 1 || behaviorImports[0].Path != "com.example.helpers" {
		t.Fatalf("behavior imports = %#v", behaviorImports)
	}
	if len(behaviorImports[0].Specifiers) != 1 || behaviorImports[0].Specifiers[0].Name != "*" {
		t.Fatalf("wildcard specifier context = %#v", behaviorImports[0].Specifiers)
	}
}

func TestAssignMutableImportsAttachesJavaStaticHelperImports(t *testing.T) {
	program, err := NewJavaFrontend().Parse(context.Background(), SourceInput{Path: "DoorHsm.java", Data: []byte(javaStaticHelperImportSource)})
	if err != nil {
		t.Fatal(err)
	}

	AssignMutableImports(program)

	if len(program.Imports) != 0 {
		t.Fatalf("expected Java static helper import to be attributed to behavior, remaining = %#v", program.Imports)
	}
	if len(program.Behaviors) != 1 {
		t.Fatalf("behaviors = %d, want 1", len(program.Behaviors))
	}
	behaviorImports := program.Behaviors[0].Imports
	if len(behaviorImports) != 1 || behaviorImports[0].Path != "com.example.Format.record" {
		t.Fatalf("behavior imports = %#v", behaviorImports)
	}
	if got := strings.Join(behaviorImports[0].LocalNames, ","); got != "record" {
		t.Fatalf("local names = %q, want record", got)
	}
	if !behaviorImports[0].Static {
		t.Fatalf("static import context lost: %#v", behaviorImports[0])
	}
}

func TestAssignMutableImportsAttachesCPPHeaderBasenameImports(t *testing.T) {
	source := `#include "hsm/hsm.hpp"
#include "format.hpp"

using namespace hsm;

static void entered(Signal &, Door &, const EventBase &) {
  format::record("closed");
}

static constexpr auto DoorModel = define(
  "Door",
  initial(target("/Door/closed")),
  state("closed", entry(entered))
);
`
	program, err := NewCPPFrontend().Parse(context.Background(), SourceInput{Path: "door.cpp", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}

	AssignMutableImports(program)

	if len(program.Imports) != 0 {
		t.Fatalf("expected C++ helper header to be attributed, remaining = %#v", program.Imports)
	}
	var behaviorImports []Import
	for _, behavior := range program.Behaviors {
		if strings.Contains(behavior.Body, "format::record") {
			behaviorImports = behavior.Imports
		}
	}
	if len(behaviorImports) != 1 || behaviorImports[0].Path != "format.hpp" {
		t.Fatalf("behavior imports = %#v", behaviorImports)
	}
	if got := strings.Join(behaviorImports[0].LocalNames, ","); got != "format" {
		t.Fatalf("local names = %q, want format", got)
	}
}

func TestAssignMutableImportsAttachesZigAliasImports(t *testing.T) {
	program, err := NewZigFrontend().Parse(context.Background(), SourceInput{Path: "door.zig", Data: []byte(zigDoorSource)})
	if err != nil {
		t.Fatal(err)
	}

	AssignMutableImports(program)

	if len(program.Imports) != 0 {
		t.Fatalf("expected Zig helper import to be attributed, remaining = %#v", program.Imports)
	}
	var helperOnBehavior bool
	for _, behavior := range program.Behaviors {
		if !strings.Contains(behavior.Body, "_ = helper") {
			continue
		}
		for _, imp := range behavior.Imports {
			if imp.Path == "helper.zig" && imp.Alias == "helper" && strings.Join(imp.LocalNames, ",") == "helper" {
				helperOnBehavior = true
			}
		}
	}
	if !helperOnBehavior {
		t.Fatalf("behaviors missing Zig helper import context: %#v", program.Behaviors)
	}
}

func TestCompilerAdapterRequestCarriesRegionImports(t *testing.T) {
	adapter := &capturingAdapter{}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)
	_, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(adapter.request.Imports) != 0 {
		t.Fatalf("top-level request imports should exclude behavior/global imports: %#v", adapter.request.Imports)
	}
	var behaviorHasContext bool
	for _, behavior := range adapter.request.Behaviors {
		for _, imp := range behavior.Imports {
			if imp.Path == "context" {
				behaviorHasContext = true
			}
		}
	}
	if !behaviorHasContext {
		t.Fatalf("adapter request behaviors missing attributed imports: %#v", adapter.request.Behaviors)
	}
}

func TestCompilerAdapterRequestDoesNotCrossAttributeMixedLanguageImports(t *testing.T) {
	adapter := &capturingAdapter{}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)
	source := `{
  "source_language": "go",
  "imports": [
    { "path": "fmt", "language": "go" },
    { "path": "./format", "language": "typescript", "specifiers": [{ "name": "record", "alias": "rec" }] },
    { "path": "unused_helpers", "language": "python" }
  ],
  "globals": [{
    "id": "global_1",
    "language": "typescript",
    "code": "export const helper = () => { fmt('shadow'); rec('closed'); };"
  }],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "states": [{
      "id": "state_1",
      "name": "closed",
      "kind": "state",
      "behaviors": [{ "id": "entry_1", "kind": "entry" }]
    }]
  }],
  "behaviors": [{
    "id": "entry_1",
    "owner_id": "state_1",
    "kind": "entry",
    "source_language": "go",
    "body": "fmt.Println(rec(\"shadow\"))"
  }]
}`

	_, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageDart,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(adapter.request.Imports) != 1 || adapter.request.Imports[0].Path != "unused_helpers" || adapter.request.Imports[0].Language != LanguagePython {
		t.Fatalf("top-level request imports = %#v, want only unused Python structural import", adapter.request.Imports)
	}
	if len(adapter.request.Globals) != 1 {
		t.Fatalf("adapter request globals = %#v, want one global", adapter.request.Globals)
	}
	globalImports := adapter.request.Globals[0].Imports
	if len(globalImports) != 1 || globalImports[0].Path != "./format" || globalImports[0].Language != LanguageTS {
		t.Fatalf("global imports = %#v, want only TypeScript format import", globalImports)
	}
	if len(adapter.request.Behaviors) != 1 {
		t.Fatalf("adapter request behaviors = %#v, want one behavior", adapter.request.Behaviors)
	}
	behaviorImports := adapter.request.Behaviors[0].Imports
	if len(behaviorImports) != 1 || behaviorImports[0].Path != "fmt" || behaviorImports[0].Language != LanguageGo {
		t.Fatalf("behavior imports = %#v, want only Go fmt import", behaviorImports)
	}
}

func TestCompilerAdapterRequestCarriesStructuredRegionImportContext(t *testing.T) {
	cases := []struct {
		name   string
		source SourceInput
		from   Language
		to     Language
		check  func(*testing.T, AdapterRequest)
	}{
		{
			name:   "dart plain import",
			source: SourceInput{Path: "door.dart", Data: []byte(dartPlainImportSource)},
			from:   LanguageDart,
			to:     LanguagePython,
			check: func(t *testing.T, request AdapterRequest) {
				t.Helper()
				imp := requireBehaviorImport(t, request, "package:sample/format.dart")
				if imp.Alias != "" || len(imp.Specifiers) != 0 {
					t.Fatalf("Dart plain import context should remain plain import, got %#v", imp)
				}
			},
		},
		{
			name:   "dart show import",
			source: SourceInput{Path: "door.dart", Data: []byte(dartShowImportSource)},
			from:   LanguageDart,
			to:     LanguagePython,
			check: func(t *testing.T, request AdapterRequest) {
				t.Helper()
				imp := requireBehaviorImport(t, request, "package:sample/format.dart")
				if len(imp.Specifiers) != 2 || imp.Specifiers[0].Name != "record" || imp.Specifiers[1].Name != "normalize" {
					t.Fatalf("Dart behavior import specifiers = %#v", imp.Specifiers)
				}
				if strings.Join(imp.LocalNames, ",") != "record,normalize" {
					t.Fatalf("Dart behavior import local names = %#v", imp.LocalNames)
				}
			},
		},
		{
			name:   "dart prefixed show import",
			source: SourceInput{Path: "door.dart", Data: []byte(dartPrefixedShowImportSource)},
			from:   LanguageDart,
			to:     LanguagePython,
			check: func(t *testing.T, request AdapterRequest) {
				t.Helper()
				imp := requireBehaviorImport(t, request, "package:sample/format.dart")
				if imp.Alias != "fmt" || len(imp.Specifiers) != 2 || imp.Specifiers[0].Name != "record" || imp.Specifiers[1].Name != "normalize" {
					t.Fatalf("Dart prefixed show import context = %#v", imp)
				}
				if strings.Join(imp.LocalNames, ",") != "fmt" {
					t.Fatalf("Dart prefixed show import local names = %#v", imp.LocalNames)
				}
			},
		},
		{
			name:   "dart hide import",
			source: SourceInput{Path: "door.dart", Data: []byte(dartHideImportSource)},
			from:   LanguageDart,
			to:     LanguagePython,
			check: func(t *testing.T, request AdapterRequest) {
				t.Helper()
				imp := requireBehaviorImport(t, request, "package:sample/format.dart")
				if strings.Join(imp.Hidden, ",") != "debugRecord,privateNormalize" {
					t.Fatalf("Dart behavior import hidden context = %#v", imp.Hidden)
				}
				if len(imp.Specifiers) != 0 {
					t.Fatalf("Dart hide import should not become show specifiers: %#v", imp.Specifiers)
				}
			},
		},
		{
			name:   "csharp alias using",
			source: SourceInput{Path: "door.cs", Data: []byte(csharpAliasUsingSource)},
			from:   LanguageCSharp,
			to:     LanguageTS,
			check: func(t *testing.T, request AdapterRequest) {
				t.Helper()
				imp := requireBehaviorImport(t, request, "Sample.Formatting.Helper")
				if imp.Alias != "Format" || strings.Join(imp.LocalNames, ",") != "Format" {
					t.Fatalf("C# behavior import context = %#v", imp)
				}
			},
		},
		{
			name:   "csharp static using",
			source: SourceInput{Path: "door.cs", Data: []byte(csharpStaticHelperUsingSource)},
			from:   LanguageCSharp,
			to:     LanguageTS,
			check: func(t *testing.T, request AdapterRequest) {
				t.Helper()
				imp := requireBehaviorImport(t, request, "Sample.Formatting.Helper")
				if imp.Alias != "" || strings.Join(imp.LocalNames, ",") != "*" {
					t.Fatalf("C# static using import context = %#v", imp)
				}
			},
		},
		{
			name:   "rust grouped use",
			source: SourceInput{Path: "door.rs", Data: []byte(rustGroupedUseSource)},
			from:   LanguageRust,
			to:     LanguageTS,
			check: func(t *testing.T, request AdapterRequest) {
				t.Helper()
				imp := requireBehaviorImport(t, request, "crate::helpers")
				if len(imp.Specifiers) != 2 || imp.Specifiers[1].Name != "normalize" || imp.Specifiers[1].Alias != "norm" {
					t.Fatalf("Rust behavior import specifiers = %#v", imp.Specifiers)
				}
				if strings.Join(imp.LocalNames, ",") != "record,norm" {
					t.Fatalf("Rust behavior import local names = %#v", imp.LocalNames)
				}
			},
		},
		{
			name:   "java wildcard import",
			source: SourceInput{Path: "DoorHsm.java", Data: []byte(javaWildcardImportSource)},
			from:   LanguageJava,
			to:     LanguageTS,
			check: func(t *testing.T, request AdapterRequest) {
				t.Helper()
				imp := requireBehaviorImport(t, request, "com.example.helpers")
				if len(imp.Specifiers) != 1 || imp.Specifiers[0].Name != "*" {
					t.Fatalf("Java behavior wildcard import = %#v", imp)
				}
			},
		},
		{
			name:   "zig alias import",
			source: SourceInput{Path: "door.zig", Data: []byte(zigDoorSource)},
			from:   LanguageZig,
			to:     LanguageTS,
			check: func(t *testing.T, request AdapterRequest) {
				t.Helper()
				for _, imp := range request.Imports {
					if imp.Path == "hsm" || imp.Path == "std" {
						t.Fatalf("Zig runtime import leaked into adapter request imports: %#v", imp)
					}
				}
				for _, global := range request.Globals {
					code := strings.TrimSpace(global.Code)
					if code == `const hsm = @import("hsm");` || code == `const std = @import("std");` {
						t.Fatalf("Zig runtime import declaration leaked into adapter globals: %#v", global)
					}
				}
				imp := requireBehaviorImport(t, request, "helper.zig")
				if imp.Alias != "helper" || strings.Join(imp.LocalNames, ",") != "helper" {
					t.Fatalf("Zig behavior import context = %#v", imp)
				}
			},
		},
		{
			name: "cpp header basename import",
			source: SourceInput{Path: "door.cpp", Data: []byte(`#include "hsm/hsm.hpp"
#include "format.hpp"

using namespace hsm;

static void entered(Signal &, Door &, const EventBase &) {
  format::record("closed");
}

static constexpr auto DoorModel = define(
  "Door",
  initial(target("/Door/closed")),
  state("closed", entry(entered))
);
`)},
			from: LanguageCPP,
			to:   LanguageTS,
			check: func(t *testing.T, request AdapterRequest) {
				t.Helper()
				for _, global := range request.Globals {
					if strings.TrimSpace(global.Code) == "using namespace hsm;" {
						t.Fatalf("C++ runtime namespace using leaked into adapter globals: %#v", global)
					}
				}
				imp := requireBehaviorImport(t, request, "format.hpp")
				if strings.Join(imp.LocalNames, ",") != "format" {
					t.Fatalf("C++ behavior import local names = %#v", imp.LocalNames)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &capturingAdapter{}
			compiler := NewCompiler()
			compiler.RegisterAdapter(adapter)
			_, _, err := compiler.Compile(context.Background(), tc.source, CompileOptions{
				From:    tc.from,
				To:      tc.to,
				Adapter: adapter.Name(),
			})
			if err != nil {
				t.Fatal(err)
			}
			tc.check(t, adapter.request)
		})
	}
}

func requireBehaviorImport(t *testing.T, request AdapterRequest, path string) Import {
	t.Helper()
	for _, behavior := range request.Behaviors {
		for _, imp := range behavior.Imports {
			if imp.Path == path {
				return imp
			}
		}
	}
	t.Fatalf("adapter request behaviors missing import %q: %#v", path, request.Behaviors)
	return Import{}
}

func requireGlobalImport(t *testing.T, request AdapterRequest, path string) Import {
	t.Helper()
	for _, global := range request.Globals {
		for _, imp := range global.Imports {
			if imp.Path == path {
				return imp
			}
		}
	}
	t.Fatalf("adapter request globals missing import %q: %#v", path, request.Globals)
	return Import{}
}

type capturingAdapter struct {
	request AdapterRequest
}

func (adapter *capturingAdapter) Name() string { return "capture" }

func (adapter *capturingAdapter) Translate(_ context.Context, request AdapterRequest) (*BehaviorPatch, error) {
	adapter.request = request
	return nil, nil
}

func assertNoHSMImport(t *testing.T, imp Import) {
	t.Helper()
	if isHSMImportPath(imp.Path) {
		t.Fatalf("HSM runtime import should stay compiler-owned, got %#v", imp)
	}
}
