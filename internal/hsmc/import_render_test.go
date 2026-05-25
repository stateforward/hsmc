package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestJavaScriptAndTypeScriptBackendsEmitStructuredImports(t *testing.T) {
	cases := []struct {
		name     string
		language Language
		backend  Backend
	}{
		{name: "javascript", language: LanguageJS, backend: NewJavaScriptBackend()},
		{name: "typescript", language: LanguageTS, backend: NewTypeScriptBackend()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := &Program{
				Imports: []Import{
					{Path: "./default", Default: "defaultHelper", Language: tc.language},
					{Path: "./named", Specifiers: []ImportSpecifier{{Name: "record", Alias: "rec"}, {Name: "format"}}, Language: tc.language},
					{Path: "./namespace", Alias: "helpers", Language: tc.language},
					{Path: "./types", TypeOnly: true, Specifiers: []ImportSpecifier{{Name: "DoorSnapshot"}}, Language: LanguageTS},
					{Path: "./adapter-types", TypeOnly: true, Specifiers: []ImportSpecifier{{Name: "AdapterContext", TypeOnly: true}}, Language: LanguageTS},
					{Path: "./metrics", Specifiers: []ImportSpecifier{{Name: "DoorMetrics", TypeOnly: true}, {Name: "recordMetric"}}, Language: LanguageTS},
				},
				Behaviors: []Behavior{{
					ID:             "entry_1",
					OwnerID:        "state_1",
					Kind:           BehaviorEntry,
					SourceLanguage: LanguageGo,
					TargetLanguage: tc.language,
					Body:           "rec('closed');",
					Imports: []Import{{
						Path:       "./behavior",
						Specifiers: []ImportSpecifier{{Name: "track"}},
						Language:   tc.language,
					}},
				}},
			}

			output, err := tc.backend.Emit(context.Background(), program)
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			for _, needle := range []string{
				`import defaultHelper from "./default";`,
				`import { record as rec, format } from "./named";`,
				`import * as helpers from "./namespace";`,
				`import { track } from "./behavior";`,
			} {
				if tc.language == LanguageTS && needle == `import { track } from "./behavior";` && !strings.Contains(text, `import type { DoorSnapshot } from "./types";`) {
					t.Fatalf("TypeScript output missing type-only import:\n%s", text)
				}
				if tc.language == LanguageTS && needle == `import { track } from "./behavior";` && !strings.Contains(text, `import type { AdapterContext } from "./adapter-types";`) {
					t.Fatalf("TypeScript output did not canonicalize nested type-only specifier in type-only import:\n%s", text)
				}
				if tc.language == LanguageTS && strings.Contains(text, `import type { type AdapterContext } from "./adapter-types";`) {
					t.Fatalf("TypeScript output rendered duplicate type-only marker:\n%s", text)
				}
				if tc.language == LanguageTS && needle == `import { track } from "./behavior";` && !strings.Contains(text, `import { type DoorMetrics, recordMetric } from "./metrics";`) {
					t.Fatalf("TypeScript output missing mixed type-only specifier import:\n%s", text)
				}
				if !strings.Contains(text, needle) {
					t.Fatalf("%s output missing %q:\n%s", tc.name, needle, text)
				}
			}
		})
	}
}

func TestZigBackendEmitsStructuredImports(t *testing.T) {
	program := &Program{
		Imports: []Import{
			{Path: "helpers.zig", Alias: "helpers", Language: LanguageZig},
			{Path: "tools/format.zig", Language: LanguageZig},
		},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageZig,
			Body:           `_ = helpers; _ = format;`,
			Imports: []Import{{
				Path:     "behavior_format",
				Alias:    "behavior_format",
				Language: LanguageZig,
			}},
		}},
	}

	output, err := NewZigBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`const helpers = @import("helpers.zig");`,
		`const format = @import("tools/format.zig");`,
		`const behavior_format = @import("behavior_format");`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Zig output missing %q:\n%s", needle, text)
		}
	}
}

func TestCPPBackendFiltersCompilerOwnedIncludeSpellings(t *testing.T) {
	program := &Program{
		Imports: []Import{
			{Path: `"hsm/hsm.hpp"`, Language: LanguageCPP},
			{Path: "<chrono>", Language: LanguageCPP},
			{Path: "<cstdint>", Language: LanguageCPP},
			{Path: `"string"`, Language: LanguageCPP},
			{Path: "helpers.hpp", Language: LanguageCPP},
		},
	}

	output, err := NewCPPBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`#include "hsm/hsm.hpp"`,
		`#include <chrono>`,
		`#include <cstdint>`,
		`#include <string>`,
		`#include "helpers.hpp"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C++ output missing %q:\n%s", needle, text)
		}
		if strings.Count(text, needle) != 1 {
			t.Fatalf("C++ output emitted duplicate %q:\n%s", needle, text)
		}
	}
}

func TestCollectRenderableImportsFiltersCompilerOwnedImports(t *testing.T) {
	for _, language := range SupportedTargetLanguages() {
		if language == LanguageJSONIR {
			continue
		}
		t.Run(string(language), func(t *testing.T) {
			var imports []Import
			for _, path := range compilerOwnedImports(language) {
				imports = append(imports, Import{Path: path, Language: language})
			}
			if language == LanguageCPP {
				imports = append(imports,
					Import{Path: `<chrono>`, Language: language},
					Import{Path: `"hsm/hsm.hpp"`, Language: language},
				)
			}
			program := &Program{
				Imports: imports,
				Globals: []CodeBlock{{
					ID:       "global_1",
					Language: language,
					Imports:  imports,
				}},
				Behaviors: []Behavior{{
					ID:             "entry_1",
					OwnerID:        "state_1",
					Kind:           BehaviorEntry,
					SourceLanguage: LanguageGo,
					TargetLanguage: language,
					Imports:        imports,
				}},
			}

			renderable := collectRenderableImports(program, language, compilerOwnedImports(language)...)
			if len(renderable) != 0 {
				t.Fatalf("renderable compiler-owned imports for %s = %#v, want none", language, renderable)
			}
		})
	}
}

func TestJavaBackendEmitsStructuredImports(t *testing.T) {
	program := &Program{
		Imports: []Import{
			{Path: "com.stateforward.hsm.*", Language: LanguageJava},
			{Path: "com.example.ProgramHelper", Language: LanguageJava},
			{Path: "com.example.Format.record", Static: true, Language: LanguageJava},
		},
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageJava,
			Code:     "private static String adapterFormat(String value) { return GlobalHelper.format(value); }",
			Imports:  []Import{{Path: "com.example.GlobalHelper", Language: LanguageJava}},
		}},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageJava,
			Body:           `BehaviorHelper.record(adapterFormat("closed"));`,
			Imports:        []Import{{Path: "com.example.BehaviorHelper", Language: LanguageJava}},
		}},
	}

	output, err := NewJavaBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"import com.stateforward.hsm.*;",
		"import com.example.ProgramHelper;",
		"import static com.example.Format.record;",
		"import com.example.GlobalHelper;",
		"import com.example.BehaviorHelper;",
		"private static String adapterFormat",
		`BehaviorHelper.record(adapterFormat("closed"));`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Java output missing %q:\n%s", needle, text)
		}
	}
	if strings.Count(text, "import com.stateforward.hsm.*;") != 1 {
		t.Fatalf("Java output emitted duplicate compiler-owned runtime import:\n%s", text)
	}
}

func TestCSharpBackendEmitsStructuredImports(t *testing.T) {
	program := &Program{
		Imports: []Import{
			{Path: "System.Text", Language: LanguageCSharp},
			{Path: "Sample.Formatting.Helper", Alias: "Format", Language: LanguageCSharp},
			{Path: "Sample.Static.Helper", LocalNames: []string{"*"}, Language: LanguageCSharp},
		},
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageCSharp,
			Code:     "private static string AdapterFormat(string value) => Format.Record(value);",
			Imports:  []Import{{Path: "Sample.Global.Helper", Alias: "GlobalFormat", Language: LanguageCSharp}},
		}},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageCSharp,
			Body:           `BehaviorFormat.Record(AdapterFormat("closed"));`,
			Imports:        []Import{{Path: "Sample.Behavior.Helper", Alias: "BehaviorFormat", Language: LanguageCSharp}},
		}},
	}

	output, err := NewCSharpBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"using System.Text;",
		"using Format = Sample.Formatting.Helper;",
		"using static Sample.Static.Helper;",
		"using GlobalFormat = Sample.Global.Helper;",
		"using BehaviorFormat = Sample.Behavior.Helper;",
		"BehaviorFormat.Record(AdapterFormat(\"closed\"));",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C# output missing %q:\n%s", needle, text)
		}
	}
}

func TestCSharpBackendEmitsStaticUsingImports(t *testing.T) {
	program := &Program{
		Imports: []Import{{
			Path:     "Sample.Formatting.Helper",
			Static:   true,
			Language: LanguageCSharp,
		}},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageCSharp,
			Body:           `Record("closed");`,
		}},
	}

	output, err := NewCSharpBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, "using static Sample.Formatting.Helper;") {
		t.Fatalf("C# output missing static using:\n%s", text)
	}
	if strings.Contains(text, "using Sample.Formatting.Helper;") {
		t.Fatalf("C# output rendered static using as ordinary using:\n%s", text)
	}
}

func TestDartBackendEmitsStructuredImports(t *testing.T) {
	program := &Program{
		Imports: []Import{
			{Path: "package:sample/format.dart", Alias: "format", Language: LanguageDart},
			{Path: "package:sample/lazy_format.dart", Alias: "lazyFormat", Deferred: true, Language: LanguageDart},
			{Path: "package:sample/records.dart", Specifiers: []ImportSpecifier{{Name: "record"}, {Name: "normalize"}}, Language: LanguageDart},
			{Path: "package:sample/tools.dart", Hidden: []string{"debugRecord", "privateNormalize"}, Language: LanguageDart},
		},
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageDart,
			Code:     "String adapterFormat(String value) => globalFormat(value);",
			Imports: []Import{{
				Path:       "package:sample/global.dart",
				Specifiers: []ImportSpecifier{{Name: "globalFormat"}},
				Language:   LanguageDart,
			}},
		}},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageDart,
			Body:           `record(adapterFormat("closed"));`,
			Imports: []Import{{
				Path:       "package:sample/behavior.dart",
				Specifiers: []ImportSpecifier{{Name: "behaviorRecord"}},
				Language:   LanguageDart,
			}},
		}},
	}

	output, err := NewDartBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`import "package:sample/format.dart" as format;`,
		`import "package:sample/lazy_format.dart" deferred as lazyFormat;`,
		`import "package:sample/records.dart" show record, normalize;`,
		`import "package:sample/tools.dart" hide debugRecord, privateNormalize;`,
		`import "package:sample/global.dart" show globalFormat;`,
		`import "package:sample/behavior.dart" show behaviorRecord;`,
		`record(adapterFormat("closed"));`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Dart output missing %q:\n%s", needle, text)
		}
	}
}

func TestRustBackendEmitsStructuredImports(t *testing.T) {
	program := &Program{
		Imports: []Import{
			{Path: "crate::helpers", Specifiers: []ImportSpecifier{{Name: "record"}, {Name: "normalize", Alias: "norm"}}, Language: LanguageRust},
			{Path: "crate::format", Alias: "format", Language: LanguageRust},
		},
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageRust,
			Code:     "fn adapter_format(value: &str) -> String { global_format(value) }",
			Imports: []Import{{
				Path:       "crate::global",
				Specifiers: []ImportSpecifier{{Name: "global_format"}},
				Language:   LanguageRust,
			}},
		}},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageRust,
			Body:           `let _ = record(&adapter_format("closed"));`,
			Imports: []Import{{
				Path:       "crate::behavior",
				Specifiers: []ImportSpecifier{{Name: "behavior_record", Alias: "behaviorRecord"}},
				Language:   LanguageRust,
			}},
		}},
	}

	output, err := NewRustBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"use crate::helpers::{record, normalize as norm};",
		"use crate::format as format;",
		"use crate::global::{global_format};",
		"use crate::behavior::{behavior_record as behaviorRecord};",
		`let _ = record(&adapter_format("closed"));`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Rust output missing %q:\n%s", needle, text)
		}
	}
}

func TestJavaScriptAndTypeScriptBackendsMergeNamedImportsByPath(t *testing.T) {
	cases := []struct {
		name     string
		language Language
		backend  Backend
	}{
		{name: "javascript", language: LanguageJS, backend: NewJavaScriptBackend()},
		{name: "typescript", language: LanguageTS, backend: NewTypeScriptBackend()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := &Program{
				Imports: []Import{{
					Path:       "./shared",
					Specifiers: []ImportSpecifier{{Name: "record"}},
					Language:   tc.language,
				}},
				Behaviors: []Behavior{{
					ID:             "entry_1",
					OwnerID:        "state_1",
					Kind:           BehaviorEntry,
					SourceLanguage: LanguageGo,
					TargetLanguage: tc.language,
					Body:           "record(); track();",
					Imports: []Import{{
						Path:       "./shared",
						Specifiers: []ImportSpecifier{{Name: "track"}},
						Language:   tc.language,
					}},
				}},
			}

			output, err := tc.backend.Emit(context.Background(), program)
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			if !strings.Contains(text, `import { record, track } from "./shared";`) {
				t.Fatalf("%s output did not merge named imports:\n%s", tc.name, text)
			}
			if strings.Count(text, `from "./shared"`) != 1 {
				t.Fatalf("%s output emitted duplicate shared imports:\n%s", tc.name, text)
			}
		})
	}
}

func TestJavaScriptAndTypeScriptBackendsMergeDefaultAndNamedImportsByPath(t *testing.T) {
	cases := []struct {
		name     string
		language Language
		backend  Backend
	}{
		{name: "javascript", language: LanguageJS, backend: NewJavaScriptBackend()},
		{name: "typescript", language: LanguageTS, backend: NewTypeScriptBackend()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := &Program{
				Imports: []Import{{
					Path:     "./shared",
					Default:  "shared",
					Language: tc.language,
				}},
				Behaviors: []Behavior{{
					ID:             "entry_1",
					OwnerID:        "state_1",
					Kind:           BehaviorEntry,
					SourceLanguage: LanguageGo,
					TargetLanguage: tc.language,
					Body:           "shared.record(); track();",
					Imports: []Import{{
						Path:       "./shared",
						Specifiers: []ImportSpecifier{{Name: "track"}},
						Language:   tc.language,
					}},
				}},
			}

			output, err := tc.backend.Emit(context.Background(), program)
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			if !strings.Contains(text, `import shared, { track } from "./shared";`) {
				t.Fatalf("%s output did not merge default and named imports:\n%s", tc.name, text)
			}
			if strings.Count(text, `from "./shared"`) != 1 {
				t.Fatalf("%s output emitted duplicate shared imports:\n%s", tc.name, text)
			}
		})
	}
}

func TestJavaScriptAndTypeScriptBackendsDoNotDropConflictingDefaultImports(t *testing.T) {
	cases := []struct {
		name     string
		language Language
		backend  Backend
	}{
		{name: "javascript", language: LanguageJS, backend: NewJavaScriptBackend()},
		{name: "typescript", language: LanguageTS, backend: NewTypeScriptBackend()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := &Program{
				Imports: []Import{{
					Path:     "./shared",
					Default:  "shared",
					Language: tc.language,
				}},
				Behaviors: []Behavior{{
					ID:             "entry_1",
					OwnerID:        "state_1",
					Kind:           BehaviorEntry,
					SourceLanguage: LanguageGo,
					TargetLanguage: tc.language,
					Body:           "other();",
					Imports: []Import{{
						Path:     "./shared",
						Default:  "otherShared",
						Language: tc.language,
					}},
				}},
			}

			output, err := tc.backend.Emit(context.Background(), program)
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			for _, needle := range []string{
				`import shared from "./shared";`,
				`import otherShared from "./shared";`,
			} {
				if !strings.Contains(text, needle) {
					t.Fatalf("%s output dropped conflicting default import %q:\n%s", tc.name, needle, text)
				}
			}
		})
	}
}

func TestPythonBackendEmitsStructuredImports(t *testing.T) {
	program := &Program{
		Imports: []Import{
			{Path: "helpers", Alias: "h", Language: LanguagePython},
			{Path: "formatting", Specifiers: []ImportSpecifier{{Name: "record", Alias: "rec"}, {Name: "format"}}, Language: LanguagePython},
		},
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguagePython,
			Code:     "def helper():\n\tpass",
			Imports: []Import{{
				Path:       "global_helpers",
				Specifiers: []ImportSpecifier{{Name: "prepare"}},
				Language:   LanguagePython,
			}},
		}},
	}

	output, err := NewPythonBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"import helpers as h",
		"from formatting import record as rec, format",
		"from global_helpers import prepare",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python output missing %q:\n%s", needle, text)
		}
	}
}

func TestPythonBackendMergesFromImportsByPath(t *testing.T) {
	program := &Program{
		Imports: []Import{{
			Path:       "formatting",
			Specifiers: []ImportSpecifier{{Name: "record"}},
			Language:   LanguagePython,
		}},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguagePython,
			Body:           "record(); track()",
			Imports: []Import{{
				Path:       "formatting",
				Specifiers: []ImportSpecifier{{Name: "track"}},
				Language:   LanguagePython,
			}},
		}},
	}

	output, err := NewPythonBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, "from formatting import record, track") {
		t.Fatalf("Python output did not merge from-imports:\n%s", text)
	}
	if strings.Count(text, "from formatting import") != 1 {
		t.Fatalf("Python output emitted duplicate formatting imports:\n%s", text)
	}
}

func TestGoBackendDoesNotOverwriteSamePathDifferentAliasImports(t *testing.T) {
	program := &Program{
		SourceLanguage: LanguageGo,
		PackageName:    "sample",
		Imports: []Import{
			{Path: "fmt", Language: LanguageGo},
			{Path: "fmt", Alias: "format", Language: LanguageGo},
		},
	}

	output, err := NewGoBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`"fmt"`,
		`format "fmt"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing import %q:\n%s", needle, text)
		}
	}
}

func TestGoBackendRuntimeImportsReplaceAliasedSourceRuntimeImports(t *testing.T) {
	program := &Program{
		SourceLanguage: LanguageGo,
		PackageName:    "sample",
		Imports: []Import{
			{Path: "context", Alias: "ctxpkg", Language: LanguageGo},
			{Path: "github.com/stateforward/hsm.go", Alias: "runtime", Language: LanguageGo},
			{Path: "time", Alias: "clock", Language: LanguageGo},
		},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageTS,
			Body:           "instance.ready = true;",
		}, {
			ID:             "trigger_1",
			OwnerID:        "state_1",
			Kind:           BehaviorTrigger,
			TriggerKind:    TriggerAfter,
			SourceLanguage: LanguageTS,
			Body:           "return 1;",
		}},
	}
	AssignTargetABI(program, LanguageGo)

	output, err := NewGoBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`"context"`,
		`hsm "github.com/stateforward/hsm.go"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing compiler-owned runtime import %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{
		`ctxpkg "context"`,
		`runtime "github.com/stateforward/hsm.go"`,
		`clock "time"`,
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Go output preserved aliased compiler-owned runtime import %q:\n%s", forbidden, text)
		}
	}
}
