package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stateforward/hsmc/internal/hsmc"
)

func TestParseLanguagePythonAliases(t *testing.T) {
	for _, alias := range []string{"py", "python"} {
		if got := parseLanguage(alias); got != hsmc.LanguagePython {
			t.Fatalf("parseLanguage(%q) = %q, want python", alias, got)
		}
	}
}

func TestReadInputSupportsStdin(t *testing.T) {
	data, sourcePath, err := readInput("-", strings.NewReader("package sample\n"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "package sample\n" || sourcePath != "stdin" {
		t.Fatalf("readInput stdin = %q, %q", data, sourcePath)
	}
}

func TestWriteOutputCreatesParentDirectories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "generated", "python", "door.py")
	if err := writeOutput(path, []byte("import hsm\n")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "import hsm\n" {
		t.Fatalf("output = %q", data)
	}
}

func TestParseLanguageJavaScriptAliases(t *testing.T) {
	for _, alias := range []string{"js", "javascript"} {
		if got := parseLanguage(alias); got != hsmc.LanguageJS {
			t.Fatalf("parseLanguage(%q) = %q, want javascript", alias, got)
		}
	}
}

func TestParseLanguageOtherAliases(t *testing.T) {
	cases := map[string]hsmc.Language{
		"c#":         hsmc.LanguageCSharp,
		"c-sharp":    hsmc.LanguageCSharp,
		"cs":         hsmc.LanguageCSharp,
		"csharp":     hsmc.LanguageCSharp,
		"c++":        hsmc.LanguageCPP,
		"cc":         hsmc.LanguageCPP,
		"cpp":        hsmc.LanguageCPP,
		"cxx":        hsmc.LanguageCPP,
		"dart":       hsmc.LanguageDart,
		"go":         hsmc.LanguageGo,
		"golang":     hsmc.LanguageGo,
		"java":       hsmc.LanguageJava,
		"cjs":        hsmc.LanguageJS,
		"js":         hsmc.LanguageJS,
		"javascript": hsmc.LanguageJS,
		"jsx":        hsmc.LanguageJS,
		"mjs":        hsmc.LanguageJS,
		"rs":         hsmc.LanguageRust,
		"rust":       hsmc.LanguageRust,
		"cts":        hsmc.LanguageTS,
		"mts":        hsmc.LanguageTS,
		"ts":         hsmc.LanguageTS,
		"tsx":        hsmc.LanguageTS,
		"typescript": hsmc.LanguageTS,
		"x-state":    hsmc.LanguageXState,
		"xstate":     hsmc.LanguageXState,
		"zig":        hsmc.LanguageZig,
		"ir":         hsmc.LanguageJSONIR,
		"json":       hsmc.LanguageJSONIR,
		"json-ir":    hsmc.LanguageJSONIR,
		"mermaid":    hsmc.LanguageMermaid,
		"mmd":        hsmc.LanguageMermaid,
		"plant":      hsmc.LanguagePlantUML,
		"plantuml":   hsmc.LanguagePlantUML,
		"puml":       hsmc.LanguagePlantUML,
	}
	for alias, want := range cases {
		if got := parseLanguage(alias); got != want {
			t.Fatalf("parseLanguage(%q) = %q, want %q", alias, got, want)
		}
	}
}

func TestInferLanguageFromPath(t *testing.T) {
	cases := map[string]hsmc.Language{
		"door.cs":       hsmc.LanguageCSharp,
		"door.csx":      hsmc.LanguageCSharp,
		"door.c++":      hsmc.LanguageCPP,
		"door.cc":       hsmc.LanguageCPP,
		"door.cpp":      hsmc.LanguageCPP,
		"door.cppm":     hsmc.LanguageCPP,
		"door.dart":     hsmc.LanguageDart,
		"door.h":        hsmc.LanguageCPP,
		"door.h++":      hsmc.LanguageCPP,
		"door.hh":       hsmc.LanguageCPP,
		"door.hpp":      hsmc.LanguageCPP,
		"door.hxx":      hsmc.LanguageCPP,
		"door.inl":      hsmc.LanguageCPP,
		"door.ipp":      hsmc.LanguageCPP,
		"door.ixx":      hsmc.LanguageCPP,
		"door.mpp":      hsmc.LanguageCPP,
		"door.tpp":      hsmc.LanguageCPP,
		"door.go":       hsmc.LanguageGo,
		"Door.java":     hsmc.LanguageJava,
		"door.cjs":      hsmc.LanguageJS,
		"door.js":       hsmc.LanguageJS,
		"door.jsx":      hsmc.LanguageJS,
		"door.mjs":      hsmc.LanguageJS,
		"door.hsm.json": hsmc.LanguageJSONIR,
		"door.mermaid":  hsmc.LanguageMermaid,
		"door.mmd":      hsmc.LanguageMermaid,
		"door.plantuml": hsmc.LanguagePlantUML,
		"door.puml":     hsmc.LanguagePlantUML,
		"door.py":       hsmc.LanguagePython,
		"door.rs":       hsmc.LanguageRust,
		"door.cts":      hsmc.LanguageTS,
		"door.mts":      hsmc.LanguageTS,
		"door.ts":       hsmc.LanguageTS,
		"door.tsx":      hsmc.LanguageTS,
		"door.zig":      hsmc.LanguageZig,
	}
	for path, want := range cases {
		got, ok := inferLanguageFromPath(path)
		if !ok || got != want {
			t.Fatalf("inferLanguageFromPath(%q) = %q, %v; want %q, true", path, got, ok, want)
		}
	}
	if got, ok := inferLanguageFromPath("-"); ok || got != "" {
		t.Fatalf("inferLanguageFromPath stdin = %q, %v; want no inference", got, ok)
	}
}

func TestDirectoryCompileFileDiscoveryMirrorsRelativePaths(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "src")
	output := filepath.Join(dir, "out")
	for _, path := range []string{
		filepath.Join(input, "door.go"),
		filepath.Join(input, "nested", "timer.ts"),
		filepath.Join(input, "transport.hsm.json"),
		filepath.Join(input, "README.md"),
		filepath.Join(output, "old.go"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := discoverDirectoryCompileFiles(input, output, "", hsmc.LanguageMermaid)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]hsmc.Language{}
	for _, file := range files {
		relative, err := filepath.Rel(output, file.outputPath)
		if err != nil {
			t.Fatal(err)
		}
		got[filepath.ToSlash(relative)] = file.from
	}
	want := map[string]hsmc.Language{
		"door.mmd":         hsmc.LanguageGo,
		"nested/timer.mmd": hsmc.LanguageTS,
		"transport.mmd":    hsmc.LanguageJSONIR,
	}
	if len(got) != len(want) {
		t.Fatalf("discovered files = %#v, want %#v", got, want)
	}
	for path, language := range want {
		if got[path] != language {
			t.Fatalf("discovered %s language = %q, want %q; all files %#v", path, got[path], language, got)
		}
	}

	filtered, err := discoverDirectoryCompileFiles(input, output, "go", hsmc.LanguagePython)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 1 || filepath.Base(filtered[0].outputPath) != "door.py" || filtered[0].from != hsmc.LanguageGo {
		t.Fatalf("filtered files = %#v, want only Go door.py", filtered)
	}
}

func TestMainCompilesDirectoryRecursivelyToDiagramTarget(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "src")
	output := filepath.Join(dir, "diagrams")
	writeTestFile(t, filepath.Join(input, "door.go"), directoryGoSource)
	writeTestFile(t, filepath.Join(input, "nested", "door.ts"), directoryTypeScriptSource)
	writeTestFile(t, filepath.Join(input, "README.md"), "# ignored\n")

	command := exec.Command("go", "run", ".", "-in", input, "-out", output, "-to", "mermaid")
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc directory to Mermaid failed: %v\n%s", err, data)
	}

	goDiagram := readTestFile(t, filepath.Join(output, "door.mmd"))
	for _, needle := range []string{
		"stateDiagram-v2",
		`state "Door" as Door {`,
		"Original go behavior entry_1 preserved for manual porting:",
		`instance.Log = append(instance.Log, "entered")`,
	} {
		if !strings.Contains(goDiagram, needle) {
			t.Fatalf("Go Mermaid diagram missing %q:\n%s", needle, goDiagram)
		}
	}
	tsDiagram := readTestFile(t, filepath.Join(output, "nested", "door.mmd"))
	for _, needle := range []string{
		"stateDiagram-v2",
		"Original typescript behavior entry_1 preserved for manual porting:",
		`instance.log.push("entered");`,
	} {
		if !strings.Contains(tsDiagram, needle) {
			t.Fatalf("TypeScript Mermaid diagram missing %q:\n%s", needle, tsDiagram)
		}
	}
	if _, err := os.Stat(filepath.Join(output, "README.mmd")); !os.IsNotExist(err) {
		t.Fatalf("directory compile should not output unsupported README.mmd, stat err = %v", err)
	}
}

func TestMainCompilesDirectoryRecursivelyToImplementationTarget(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "src")
	output := filepath.Join(dir, "python")
	writeTestFile(t, filepath.Join(input, "door.go"), directoryGoSource)
	writeTestFile(t, filepath.Join(input, "nested", "timer.go"), strings.ReplaceAll(directoryGoSource, `"Door"`, `"Timer"`))

	command := exec.Command("go", "run", ".", "-from", "go", "-to", "python", "-in", input, "-out", output)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc directory to Python failed: %v\n%s", err, data)
	}
	for _, path := range []string{
		filepath.Join(output, "door.py"),
		filepath.Join(output, "nested", "timer.py"),
	} {
		text := readTestFile(t, path)
		for _, needle := range []string{
			"import hsm",
			"# Original go behavior entry_1 preserved for manual porting:",
			`# instance.Log = append(instance.Log, "entered")`,
		} {
			if !strings.Contains(text, needle) {
				t.Fatalf("Python output %s missing %q:\n%s", path, needle, text)
			}
		}
	}
}

func TestMainCompilesDirectoryRecursivelyToJSONIR(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "src")
	output := filepath.Join(dir, "ir")
	writeTestFile(t, filepath.Join(input, "door.ts"), directoryTypeScriptSource)

	command := exec.Command("go", "run", ".", "-to", "json-ir", "-in", input, "-out", output)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc directory to JSON IR failed: %v\n%s", err, data)
	}
	generated := readTestFile(t, filepath.Join(output, "door.hsm.json"))
	var program hsmc.Program
	if err := json.Unmarshal([]byte(generated), &program); err != nil {
		t.Fatalf("directory JSON IR is invalid: %v\n%s", err, generated)
	}
	if program.SourceLanguage != hsmc.LanguageTS || len(program.Models) != 1 {
		t.Fatalf("directory JSON IR program = %#v", program)
	}
}

func TestMainRejectsDirectoryInputWithoutOutputDirectory(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "src")
	writeTestFile(t, filepath.Join(input, "door.go"), directoryGoSource)

	command := exec.Command("go", "run", ".", "-in", input, "-to", "mermaid")
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted directory input without output:\n%s", data)
	}
	if !strings.Contains(string(data), "-out is required when -in is a directory") {
		t.Fatalf("directory input missing output rejection:\n%s", data)
	}
}

func TestCompileOptionsInferLanguages(t *testing.T) {
	options := compileOptions("", "", "door.ts", "door.py", " none ")
	if options.From != hsmc.LanguageTS || options.To != hsmc.LanguagePython || options.Adapter != "none" {
		t.Fatalf("compileOptions inferred %#v, want ts -> python", options)
	}
	options = compileOptions("py", "rs", "door.ts", "door.go", "codex")
	if options.From != hsmc.LanguagePython || options.To != hsmc.LanguageRust || options.Adapter != "codex" {
		t.Fatalf("compileOptions explicit %#v, want python -> rust", options)
	}
	options = compileOptions("", "", "door.go", "Door.java", "none")
	if options.From != hsmc.LanguageGo || options.To != hsmc.LanguageJava {
		t.Fatalf("compileOptions Java target %#v, want go -> java", options)
	}
	options = compileOptions("", "", "stdin", "", "none")
	if options.From != hsmc.LanguageGo || options.To != hsmc.LanguageGo {
		t.Fatalf("compileOptions default %#v, want go -> go", options)
	}
}

func TestAdapterProtocolOptionsDoNotInferJSONIRFromProtocolOutput(t *testing.T) {
	options := adapterProtocolOptions("", "", "door.ts", "protocol.json")
	if options.From != hsmc.LanguageTS || options.To != hsmc.LanguageGo {
		t.Fatalf("adapterProtocolOptions inferred %#v, want typescript -> go", options)
	}
	options = adapterProtocolOptions("", "", "door.ts", "protocol.py")
	if options.From != hsmc.LanguageTS || options.To != hsmc.LanguagePython {
		t.Fatalf("adapterProtocolOptions inferred %#v, want typescript -> python", options)
	}
	options = adapterProtocolOptions("", "json-ir", "door.ts", "protocol.json")
	if options.To != hsmc.LanguageJSONIR {
		t.Fatalf("adapterProtocolOptions explicit target = %q, want json-ir", options.To)
	}
}

func TestMainTrimsAdapterNameBeforeRegistration(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte("#!/bin/sh\ncat >/dev/null\nprintf '{}\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-in", input, "-out", output, "-adapter", " command ", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), `door_model = hsm.Define(`) {
		t.Fatalf("trimmed adapter CLI output missing model:\n%s", generated)
	}
}

func TestMainCommandAdapterRequiresCommand(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command")
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted command adapter without command:\n%s", data)
	}
	if !strings.Contains(string(data), "command adapter command is required") {
		t.Fatalf("go run hsmc error missing command-required rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("command adapter missing-command rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected command output: %v", readErr)
	}
}

func TestMainCommandAdapterRejectsNullPatch(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte("#!/bin/sh\ncat >/dev/null\nprintf 'null\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted null adapter patch:\n%s", data)
	}
	if !strings.Contains(string(data), `adapter patch must be a JSON object, got null`) {
		t.Fatalf("go run hsmc error missing null-patch rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("adapter null-patch rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected command output: %v", readErr)
	}
}

func TestMainRejectsUnknownAdapterName(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "missing-agent")
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted unknown adapter:\n%s", data)
	}
	if !strings.Contains(string(data), `unsupported behavior adapter "missing-agent"`) {
		t.Fatalf("go run hsmc error missing unsupported-adapter rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("unknown adapter rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected adapter output: %v", readErr)
	}
}

func TestMainCommandAdapterRejectsModelEdits(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte("#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' '{\"models\":[{\"id\":\"model_1\",\"name\":\"Changed\"}]}'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted adapter model edits:\n%s", data)
	}
	if !strings.Contains(string(data), `unsupported top-level field "models"`) {
		t.Fatalf("go run hsmc error missing model-edit rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("adapter model-edit rejection should not write output, got:\n%s", generated)
	}
}

func TestMainCommandAdapterRejectsStructuralTopLevelEdits(t *testing.T) {
	cases := []struct {
		name  string
		field string
		patch string
	}{
		{
			name:  "models",
			field: "models",
			patch: `{"models":[{"id":"model_1","name":"Changed"}]}`,
		},
		{
			name:  "events",
			field: "events",
			patch: `{"events":[{"id":"event_1","name":"changed"}]}`,
		},
		{
			name:  "attributes",
			field: "attributes",
			patch: `{"attributes":[{"id":"attribute_1","name":"changed"}]}`,
		},
		{
			name:  "operations",
			field: "operations",
			patch: `{"operations":[{"id":"operation_1","name":"changed"}]}`,
		},
		{
			name:  "states",
			field: "states",
			patch: `{"states":[{"id":"state_1","name":"changed"}]}`,
		},
		{
			name:  "initial",
			field: "initial",
			patch: `{"initial":{"target":"changed"}}`,
		},
		{
			name:  "transitions",
			field: "transitions",
			patch: `{"transitions":[{"id":"transition_1","target":"changed"}]}`,
		},
		{
			name:  "triggers",
			field: "triggers",
			patch: `{"triggers":[{"kind":"on","value":"changed"}]}`,
		},
		{
			name:  "defers",
			field: "defers",
			patch: `{"defers":["changed"]}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "door.ts")
			output := filepath.Join(dir, "door.py")
			adapter := filepath.Join(dir, "adapter.sh")
			source := `import * as hsm from "@stateforward/hsm.ts";

export const opened = hsm.Event("opened");

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Transition(hsm.On(opened), hsm.Target("../open"))),
  hsm.State("open"),
);
`
			if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(adapter, []byte("#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' '"+tc.patch+"'\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
			command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
			data, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("go run hsmc unexpectedly accepted adapter %s edits:\n%s", tc.field, data)
			}
			want := `unsupported top-level field "` + tc.field + `"`
			if !strings.Contains(string(data), want) {
				t.Fatalf("go run hsmc error missing %s rejection %q:\n%s", tc.field, want, data)
			}
			if generated, readErr := os.ReadFile(output); readErr == nil {
				t.Fatalf("adapter %s rejection should not write output, got:\n%s", tc.field, generated)
			}
		})
	}
}

func TestMainCommandAdapterRejectsBehaviorIdentityEdits(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte("#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' '{\"behaviors\":[{\"id\":\"entry_1\",\"owner_id\":\"state_2\",\"body\":\"pass\"}]}'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted adapter behavior identity edits:\n%s", data)
	}
	if !strings.Contains(string(data), `behaviors[0] contains unsupported field "owner_id"`) {
		t.Fatalf("go run hsmc error missing behavior identity rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("adapter behavior identity rejection should not write output, got:\n%s", generated)
	}
}

func TestMainCommandAdapterRejectsBehaviorMetadataEdits(t *testing.T) {
	cases := []struct {
		name  string
		field string
		patch string
	}{
		{
			name:  "kind",
			field: "kind",
			patch: `{"behaviors":[{"id":"entry_1","kind":"guard","body":"return True"}]}`,
		},
		{
			name:  "trigger kind",
			field: "trigger_kind",
			patch: `{"behaviors":[{"id":"entry_1","trigger_kind":"when","body":"return True"}]}`,
		},
		{
			name:  "source language",
			field: "source_language",
			patch: `{"behaviors":[{"id":"entry_1","source_language":"python","body":"pass"}]}`,
		},
		{
			name:  "inline",
			field: "inline",
			patch: `{"behaviors":[{"id":"entry_1","inline":true,"body":"pass"}]}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "door.ts")
			output := filepath.Join(dir, "door.py")
			adapter := filepath.Join(dir, "adapter.sh")
			source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
			if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(adapter, []byte("#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' '"+tc.patch+"'\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
			command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
			data, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("go run hsmc unexpectedly accepted adapter behavior %s edits:\n%s", tc.field, data)
			}
			want := `behaviors[0] contains unsupported field "` + tc.field + `"`
			if !strings.Contains(string(data), want) {
				t.Fatalf("go run hsmc error missing behavior %s rejection %q:\n%s", tc.field, want, data)
			}
			if generated, readErr := os.ReadFile(output); readErr == nil {
				t.Fatalf("adapter behavior %s rejection should not write output, got:\n%s", tc.field, generated)
			}
		})
	}
}

func TestMainCommandAdapterRejectsBehaviorABIEdits(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := `{"behaviors":[{"id":"entry_1","target_abi":{"language":"python","signature":"def changed(ctx):"},"body":"pass"}]}`
	if err := os.WriteFile(adapter, []byte("#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' '"+patch+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted adapter behavior ABI edits:\n%s", data)
	}
	if !strings.Contains(string(data), `behaviors[0] contains unsupported field "target_abi"`) {
		t.Fatalf("go run hsmc error missing behavior ABI rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("adapter behavior ABI rejection should not write output, got:\n%s", generated)
	}
}

func TestMainCommandAdapterRejectsBehaviorSignatureEdits(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := `{"behaviors":[{"id":"entry_1","signature":"def changed(ctx):","body":"pass"}]}`
	if err := os.WriteFile(adapter, []byte("#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' '"+patch+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted adapter behavior signature edits:\n%s", data)
	}
	if !strings.Contains(string(data), `behaviors[0] contains unsupported field "signature"`) {
		t.Fatalf("go run hsmc error missing behavior signature rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("adapter behavior signature rejection should not write output, got:\n%s", generated)
	}
}

func TestMainCommandAdapterRejectsBehaviorBodyAndCode(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := `{"behaviors":[{"id":"entry_1","body":"instance.log = \"closed\"","code":"def entry_1(ctx, instance, event):\n\tinstance.log = \"closed\""}]}`
	if err := os.WriteFile(adapter, []byte("#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' '"+patch+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted adapter behavior body+code patch:\n%s", data)
	}
	if !strings.Contains(string(data), `behaviors[0] must set either body or code, not both`) {
		t.Fatalf("go run hsmc error missing behavior body+code rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("adapter behavior body+code rejection should not write output, got:\n%s", generated)
	}
}

func TestMainCommandAdapterRejectsBehaviorTargetLanguageEdits(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := `{"behaviors":[{"id":"entry_1","target_language":"python","body":"pass"}]}`
	if err := os.WriteFile(adapter, []byte("#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' '"+patch+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted adapter behavior target_language edits:\n%s", data)
	}
	if !strings.Contains(string(data), `behaviors[0] contains unsupported field "target_language"`) {
		t.Fatalf("go run hsmc error missing behavior target_language rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("adapter behavior target_language rejection should not write output, got:\n%s", generated)
	}
}

func TestMainCommandAdapterRejectsGlobalLanguageEdits(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function record(value: string): string {
  return value;
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"globals":[{"id":"global_1","language":"python","code":"def record(value: str) -> str:\n\treturn value"}]}
JSON
`
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted adapter global language edits:\n%s", data)
	}
	if !strings.Contains(string(data), `globals[0] contains unsupported field "language"`) {
		t.Fatalf("go run hsmc error missing global language rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("adapter global language rejection should not write output, got:\n%s", generated)
	}
}

func TestMainCommandAdapterRejectsAdapterOnlyMetadataFields(t *testing.T) {
	cases := []struct {
		name  string
		patch string
		want  string
	}{
		{
			name:  "global range",
			patch: `{"globals":[{"id":"global_1","code":"def record(value: str) -> str:\n\treturn value","range":{"start_byte":1}}]}`,
			want:  `globals[0] contains unsupported field "range"`,
		},
		{
			name:  "import local names",
			patch: `{"imports":[{"path":"helpers","local_names":["helpers"]}]}`,
			want:  `imports[0] contains unsupported field "local_names"`,
		},
		{
			name:  "specifier range",
			patch: `{"imports":[{"path":"helpers","specifiers":[{"name":"record","range":{"start_byte":1}}]}]}`,
			want:  `imports[0].specifiers[0] contains unsupported field "range"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "door.ts")
			output := filepath.Join(dir, "door.py")
			adapter := filepath.Join(dir, "adapter.sh")
			source := `import * as hsm from "@stateforward/hsm.ts";

function record(value: string): string {
  return value;
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`
			if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(adapter, []byte("#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' '"+tc.patch+"'\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
			command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
			data, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("go run hsmc unexpectedly accepted adapter-only metadata field in %s:\n%s", tc.name, data)
			}
			if !strings.Contains(string(data), tc.want) {
				t.Fatalf("go run hsmc error missing adapter-only metadata rejection %q:\n%s", tc.want, data)
			}
			if generated, readErr := os.ReadFile(output); readErr == nil {
				t.Fatalf("adapter-only metadata rejection should not write output, got:\n%s", generated)
			}
		})
	}
}

func TestMainCommandAdapterRejectsMalformedPatchFields(t *testing.T) {
	cases := []struct {
		name  string
		patch string
		want  string
	}{
		{
			name:  "top-level imports null",
			patch: `{"imports":null}`,
			want:  `field "imports" must be an array, got null`,
		},
		{
			name:  "behavior body number",
			patch: `{"behaviors":[{"id":"entry_1","body":42}]}`,
			want:  `behaviors[0].body must be a string`,
		},
		{
			name:  "behavior whitespace-only body",
			patch: `{"behaviors":[{"id":"entry_1","body":" \t\n"}]}`,
			want:  `behaviors[0].body contains whitespace-only code`,
		},
		{
			name:  "global whitespace-only code",
			patch: `{"globals":[{"id":"global_1","code":" \t\n"}]}`,
			want:  `globals[0].code contains whitespace-only code`,
		},
		{
			name:  "behavior whitespace-only code",
			patch: `{"behaviors":[{"id":"entry_1","code":" \t\n"}]}`,
			want:  `behaviors[0].code contains whitespace-only code`,
		},
		{
			name:  "behavior body and code",
			patch: `{"behaviors":[{"id":"entry_1","body":"instance.log = 'closed'","code":"def entry_1(ctx, instance, event):\n\tinstance.log = 'closed'"}]}`,
			want:  `behaviors[0] must set either body or code, not both`,
		},
		{
			name:  "behavior missing id",
			patch: `{"behaviors":[{"body":"pass"}]}`,
			want:  `behaviors[0].id is required`,
		},
		{
			name:  "global padded id",
			patch: `{"globals":[{"id":" global_1 ","code":"def record(value: str) -> str:\n\treturn value"}]}`,
			want:  `globals[0].id " global_1 " has leading or trailing whitespace`,
		},
		{
			name:  "behavior padded id",
			patch: `{"behaviors":[{"id":" entry_1 ","body":"instance.log = 'closed'"}]}`,
			want:  `behaviors[0].id " entry_1 " has leading or trailing whitespace`,
		},
		{
			name:  "program import padded path",
			patch: `{"imports":[{"path":" helpers "}]}`,
			want:  `imports[0].path " helpers " has leading or trailing whitespace`,
		},
		{
			name:  "duplicate top-level field",
			patch: `{"behaviors":[],"behaviors":[{"id":"entry_1","body":"translated()"}]}`,
			want:  `adapter patch contains duplicate field "behaviors"`,
		},
		{
			name:  "duplicate behavior field",
			patch: `{"behaviors":[{"id":"entry_1","body":"one()","body":"two()"}]}`,
			want:  `adapter patch behaviors[0] contains duplicate field "body"`,
		},
		{
			name:  "duplicate nested import field",
			patch: `{"globals":[{"id":"global_1","code":"def record(value: str) -> str:\n\treturn value","imports":[{"path":"one","path":"two"}]}]}`,
			want:  `adapter patch globals[0].imports[0] contains duplicate field "path"`,
		},
		{
			name:  "duplicate global id",
			patch: `{"globals":[{"id":"global_1","code":"def one() -> None:\n\tpass"},{"id":"global_1","code":"def two() -> None:\n\tpass"}]}`,
			want:  `adapter patch contains duplicate global "global_1"`,
		},
		{
			name:  "duplicate behavior id",
			patch: `{"behaviors":[{"id":"entry_1","body":"one()"},{"id":"entry_1","body":"two()"}]}`,
			want:  `adapter patch contains duplicate behavior "entry_1"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "door.ts")
			output := filepath.Join(dir, "door.py")
			adapter := filepath.Join(dir, "adapter.sh")
			source := `import * as hsm from "@stateforward/hsm.ts";

function record(value: string): string {
  return value;
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = record("closed");
  })),
);
`
			if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(adapter, []byte("#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' '"+tc.patch+"'\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
			command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
			data, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("go run hsmc unexpectedly accepted malformed adapter patch %s:\n%s", tc.name, data)
			}
			if !strings.Contains(string(data), tc.want) {
				t.Fatalf("go run hsmc error missing malformed patch rejection %q:\n%s", tc.want, data)
			}
			if generated, readErr := os.ReadFile(output); readErr == nil {
				t.Fatalf("malformed adapter patch rejection should not write output, got:\n%s", generated)
			}
		})
	}
}

func TestMainCommandAdapterRejectsUnknownEditableIDs(t *testing.T) {
	cases := []struct {
		name  string
		patch string
		want  string
	}{
		{
			name:  "unknown behavior",
			patch: `{"behaviors":[{"id":"entry_missing","body":"pass"}]}`,
			want:  `adapter patch references unknown behavior "entry_missing"`,
		},
		{
			name:  "unknown global",
			patch: `{"globals":[{"id":"global_missing","code":"def helper() -> None:\n\tpass"}]}`,
			want:  `adapter patch references unknown global "global_missing"`,
		},
		{
			name:  "helper global missing code",
			patch: `{"globals":[{"id":"adapter_global_helper"}]}`,
			want:  `adapter-created global "adapter_global_helper" is missing target code`,
		},
		{
			name:  "helper global empty suffix",
			patch: `{"globals":[{"id":"adapter_global_","code":"def helper() -> None:\n\tpass"}]}`,
			want:  `adapter-created global id "adapter_global_" is missing a helper name`,
		},
		{
			name:  "helper global punctuated suffix",
			patch: `{"globals":[{"id":"adapter_global_helper.bad","code":"def helper() -> None:\n\tpass"}]}`,
			want:  `adapter-created global id "adapter_global_helper.bad" must contain only letters, digits, and underscores after adapter_global_`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "door.ts")
			output := filepath.Join(dir, "door.py")
			adapter := filepath.Join(dir, "adapter.sh")
			source := `import * as hsm from "@stateforward/hsm.ts";

function record(value: string): string {
  return value;
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = record("closed");
  })),
);
`
			if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(adapter, []byte("#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' '"+tc.patch+"'\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
			command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
			data, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("go run hsmc unexpectedly accepted unknown editable ID patch %s:\n%s", tc.name, data)
			}
			if !strings.Contains(string(data), tc.want) {
				t.Fatalf("go run hsmc error missing unknown editable ID rejection %q:\n%s", tc.want, data)
			}
			if generated, readErr := os.ReadFile(output); readErr == nil {
				t.Fatalf("unknown editable ID rejection should not write output, got:\n%s", generated)
			}
		})
	}
}

func TestMainCommandAdapterRejectsCompilerOwnedRuntimeImports(t *testing.T) {
	cases := []struct {
		name  string
		patch string
		want  string
	}{
		{
			name:  "program import",
			patch: `{"imports":[{"path":"datetime","alias":"dt"}]}`,
			want:  `program import "datetime" is compiler-owned`,
		},
		{
			name:  "global import",
			patch: `{"globals":[{"id":"global_1","code":"def record(value: str) -> str:\n\treturn value","imports":[{"path":"datetime","alias":"dt"}]}]}`,
			want:  `global "global_1" import "datetime" is compiler-owned`,
		},
		{
			name:  "helper global import",
			patch: `{"globals":[{"id":"adapter_global_time","code":"def adapter_time() -> None:\n\treturn None","imports":[{"path":"datetime","alias":"dt"}]}]}`,
			want:  `global "adapter_global_time" import "datetime" is compiler-owned`,
		},
		{
			name:  "behavior import",
			patch: `{"behaviors":[{"id":"entry_1","body":"instance.log = 'closed'","imports":[{"path":"datetime","alias":"dt"}]}]}`,
			want:  `behavior "entry_1" import "datetime" is compiler-owned`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "door.ts")
			output := filepath.Join(dir, "door.py")
			adapter := filepath.Join(dir, "adapter.sh")
			source := `import * as hsm from "@stateforward/hsm.ts";

function record(value: string): string {
  return value;
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = record("closed");
  })),
);
`
			if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(adapter, []byte("#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' '"+tc.patch+"'\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
			command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
			data, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("go run hsmc unexpectedly accepted adapter compiler-owned runtime import %s:\n%s", tc.name, data)
			}
			if !strings.Contains(string(data), tc.want) {
				t.Fatalf("go run hsmc error missing compiler-owned import rejection %q:\n%s", tc.want, data)
			}
			if generated, readErr := os.ReadFile(output); readErr == nil {
				t.Fatalf("adapter compiler-owned import rejection should not write output, got:\n%s", generated)
			}
		})
	}
}

func TestMainCommandAdapterRejectsImportLanguageBoundaryViolations(t *testing.T) {
	cases := []struct {
		name  string
		patch string
		want  string
	}{
		{
			name:  "program import wrong language",
			patch: `{"imports":[{"path":"helpers","language":"go"}]}`,
			want:  `adapter patch program import "helpers" has language "go", want target language "python"`,
		},
		{
			name:  "translated global import wrong language",
			patch: `{"globals":[{"id":"global_1","code":"def record(value: str) -> str:\n\treturn value","imports":[{"path":"helpers","language":"go"}]}]}`,
			want:  `adapter patch global "global_1" import "helpers" has language "go", want target language "python"`,
		},
		{
			name:  "helper global import wrong language",
			patch: `{"globals":[{"id":"adapter_global_record","code":"def adapter_record(value: str) -> str:\n\treturn value","imports":[{"path":"helpers","language":"go"}]}]}`,
			want:  `adapter patch global "adapter_global_record" import "helpers" has language "go", want target language "python"`,
		},
		{
			name:  "behavior import wrong language",
			patch: `{"behaviors":[{"id":"entry_1","body":"instance.log = record('closed')","imports":[{"path":"helpers","language":"go"}]}]}`,
			want:  `adapter patch behavior "entry_1" import "helpers" has language "go", want target language "python"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "door.ts")
			output := filepath.Join(dir, "door.py")
			adapter := filepath.Join(dir, "adapter.sh")
			source := `import * as hsm from "@stateforward/hsm.ts";

function record(value: string): string {
  return value;
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = record("closed");
  })),
);
`
			if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(adapter, []byte("#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' '"+tc.patch+"'\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
			command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
			data, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("go run hsmc unexpectedly accepted adapter import language boundary violation %s:\n%s", tc.name, data)
			}
			if !strings.Contains(string(data), tc.want) {
				t.Fatalf("go run hsmc error missing import language boundary rejection %q:\n%s", tc.want, data)
			}
			if generated, readErr := os.ReadFile(output); readErr == nil {
				t.Fatalf("adapter import language boundary rejection should not write output, got:\n%s", generated)
			}
		})
	}
}

func TestMainCommandAdapterRejectsForeignImportOnlyPatches(t *testing.T) {
	cases := []struct {
		name  string
		patch string
		want  string
	}{
		{
			name:  "foreign global import only",
			patch: `{"globals":[{"id":"global_1","imports":[{"path":"helpers","specifiers":[{"name":"record"}]}]}]}`,
			want:  `adapter patch changes global "global_1" imports without target code`,
		},
		{
			name:  "foreign behavior import only",
			patch: `{"behaviors":[{"id":"entry_1","imports":[{"path":"helpers","specifiers":[{"name":"record"}]}]}]}`,
			want:  `adapter patch changes behavior "entry_1" imports without target body or code`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "door.ts")
			output := filepath.Join(dir, "door.py")
			adapter := filepath.Join(dir, "adapter.sh")
			source := `import * as hsm from "@stateforward/hsm.ts";

function record(value: string): string {
  return value;
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = record("closed");
  })),
);
`
			if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(adapter, []byte("#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' '"+tc.patch+"'\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
			command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
			data, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("go run hsmc unexpectedly accepted foreign import-only adapter patch %s:\n%s", tc.name, data)
			}
			if !strings.Contains(string(data), tc.want) {
				t.Fatalf("go run hsmc error missing foreign import-only rejection %q:\n%s", tc.want, data)
			}
			if generated, readErr := os.ReadFile(output); readErr == nil {
				t.Fatalf("foreign import-only rejection should not write output, got:\n%s", generated)
			}
		})
	}
}

func TestMainCommandAdapterAppliesBehaviorBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx, instance, event) => {
      instance.log = "closed";
    }),
    hsm.Transition(
      hsm.On("open"),
      hsm.Target("../open"),
    ),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"entry_1","body":"instance.log = [record(\"closed\")]","imports":[{"path":"helpers","specifiers":[{"name":"record"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		"from helpers import record",
		"async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:",
		`instance.log = [record("closed")]`,
		`door_model = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("closed")`,
		`hsm.State(`,
		`"closed"`,
		`hsm.Entry(entry_1)`,
		`hsm.Transition(`,
		`hsm.On("open")`,
		`hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("adapter CLI did not replace source behavior body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesCSharpBodiesAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "GeneratedHsm.cs")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx, instance, event) => {
      instance.log = "closed";
    }),
    hsm.Transition(
      hsm.On("open"),
      hsm.Target("../open"),
    ),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"globals":[{"id":"adapter_global_format","code":"private static string AdapterFormat(string value) => GlobalFormat.Record(value);","imports":[{"path":"Sample.Global.Helper","alias":"GlobalFormat"}]}],"behaviors":[{"id":"entry_1","body":"BehaviorFormat.Record(AdapterFormat(\"closed\"));","imports":[{"path":"Sample.Behavior.Helper","alias":"BehaviorFormat"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "csharp", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		"using Stateforward.Hsm;",
		"using GlobalFormat = Sample.Global.Helper;",
		"using BehaviorFormat = Sample.Behavior.Helper;",
		"public static class GeneratedHsm",
		`private static string AdapterFormat(string value) => GlobalFormat.Record(value);`,
		"private static void Entry1(Context ctx, Instance instance, Event @event)",
		`BehaviorFormat.Record(AdapterFormat("closed"));`,
		`public static readonly Model DoorModel = Hsm.Define(`,
		`Hsm.Initial(`,
		`Hsm.Target("closed")`,
		`Hsm.State(`,
		`"closed"`,
		`Hsm.Entry<Instance>(Entry1)`,
		`Hsm.Transition(`,
		`Hsm.On("open")`,
		`Hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command C# adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("command C# adapter CLI did not replace source behavior body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesCSharpOperationBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "GeneratedHsm.cs")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"operation_1","body":"Approvals.Record(instance, \"approve\");","imports":[{"path":"Sample.Approvals","alias":"Approvals"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "csharp", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command C# operation adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		"using Stateforward.Hsm;",
		"using Approvals = Sample.Approvals;",
		"private static void Operation1(Context ctx, Instance instance, Event @event)",
		`Approvals.Record(instance, "approve");`,
		`public static readonly Model DoorModel = Hsm.Define(`,
		`Hsm.Operation("approve", new Operation<Instance>(Operation1))`,
		`Hsm.Initial(`,
		`Hsm.Target("closed")`,
		`Hsm.State(`,
		`"closed"`,
		`Hsm.Transition(`,
		`Hsm.OnCall("approve")`,
		`Hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command C# operation adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) ||
		strings.Contains(text, "function operation1(") {
		t.Fatalf("command C# operation adapter CLI did not replace source operation body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterRejectsCSharpFieldDeclarationBehaviorFullCodeSignatureChange(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "GeneratedHsm.cs")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"entry_1","code":"private static readonly Operation<Instance> Entry1 = (ctx, instance) => System.GC.KeepAlive(\"entry\");"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "csharp", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted C# field declaration behavior full-code signature change:\n%s", data)
	}
	if !strings.Contains(string(data), `declares lambda parameters "(ctx, instance)", want compiler-owned parameters "(ctx, instance, @event)"`) {
		t.Fatalf("go run hsmc error missing C# field declaration signature rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("C# field declaration behavior full-code signature rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected command output: %v", readErr)
	}
}

func TestMainCommandAdapterAppliesCPPBodiesAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.cpp")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx, instance, event) => {
      instance.log = "closed";
    }),
    hsm.Transition(
      hsm.On("open"),
      hsm.Target("../open"),
    ),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"globals":[{"id":"adapter_global_format","code":"static std::string adapter_format(std::string value) { return adapter_global_format(value); }","imports":[{"path":"global_format.hpp"}]}],"behaviors":[{"id":"entry_1","body":"(void)adapter_behavior_format(adapter_format(\"closed\"));","imports":[{"path":"behavior_format.hpp"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "cpp", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`#include "hsm/hsm.hpp"`,
		`#include "global_format.hpp"`,
		`#include "behavior_format.hpp"`,
		"namespace generated_hsm",
		"static std::string adapter_format(std::string value) { return adapter_global_format(value); }",
		"static constexpr auto entry_1 = [](auto& signal, auto& instance, const auto& event) -> void",
		`(void)adapter_behavior_format(adapter_format("closed"));`,
		`static constexpr auto door_model = hsm::define(`,
		`hsm::initial(hsm::target("/Door/closed"))`,
		`hsm::state("closed"`,
		`hsm::entry(entry_1)`,
		`hsm::transition(`,
		`hsm::on("open")`,
		`hsm::target("/Door/open")`,
		`hsm::state("open")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command C++ adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("command C++ adapter CLI did not replace source behavior body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesCPPOperationBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.cpp")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"operation_1","body":"(void)adapter_approve(\"approve\");","imports":[{"path":"approval.hpp"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "cpp", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`#include "hsm/hsm.hpp"`,
		`#include "approval.hpp"`,
		`namespace generated_hsm`,
		`static void operation_1()`,
		`(void)adapter_approve("approve");`,
		`static constexpr auto door_model = hsm::define(`,
		`hsm::operation("approve", &operation_1)`,
		`hsm::initial(hsm::target("/Door/closed"))`,
		`hsm::state("closed"`,
		`hsm::transition(hsm::on_call("approve"), hsm::target("/Door/open"))`,
		`hsm::state("open")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command C++ operation adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) ||
		strings.Contains(text, "function operation1(") ||
		strings.Contains(text, "hsm.cpp requires a function pointer or member pointer") {
		t.Fatalf("command C++ operation adapter CLI did not replace source operation body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesCPPAtTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.cpp")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return clock::deadline();","imports":[{"path":"clock.hpp"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "cpp", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`#include "hsm/hsm.hpp"`,
		`#include "clock.hpp"`,
		`#include <chrono>`,
		`static constexpr auto trigger_1 = [](auto& signal, auto& instance, const auto& event) -> std::chrono::system_clock::time_point`,
		`return clock::deadline();`,
		`static constexpr auto timer_model = hsm::define(`,
		`"Timer"`,
		`hsm::initial(hsm::target("/Timer/idle"))`,
		`hsm::state("idle"`,
		`hsm::transition(hsm::at(trigger_1), hsm::target("/Timer/done"))`,
		`hsm::state("done")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command C++ at trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return new Date();") {
		t.Fatalf("command C++ at trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesCPPWhenTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.cpp")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.When((ctx, instance, event) => {
        return instance.ready;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return ready(instance);","imports":[{"path":"signals.hpp"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "cpp", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command C++ when trigger adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`#include "hsm/hsm.hpp"`,
		`#include "signals.hpp"`,
		`static constexpr auto trigger_1 = [](auto& signal, auto& instance, const auto& event) -> bool`,
		`return ready(instance);`,
		`static constexpr auto timer_model = hsm::define(`,
		`"Timer"`,
		`hsm::initial(hsm::target("/Timer/idle"))`,
		`hsm::state("idle"`,
		`hsm::transition(hsm::guard(trigger_1), hsm::target("/Timer/done"))`,
		`hsm::state("done")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command C++ when trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return instance.ready;") ||
		strings.Contains(text, "Unsupported when trigger") ||
		strings.Contains(text, "hsm::when(trigger_1)") {
		t.Fatalf("command C++ when trigger adapter CLI did not preserve the target trigger boundary cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterRejectsCPPOperationFullCodeSignatureChange(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.cpp")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"operation_1","code":"static void operation_1(auto& signal) { adapter_format(\"approve\"); }"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "cpp", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted C++ operation full-code signature change:\n%s", data)
	}
	if !strings.Contains(string(data), `declares parameters "(auto& signal)", want compiler-owned parameters "()"`) {
		t.Fatalf("go run hsmc error missing C++ operation signature rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("C++ operation full-code signature rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected command output: %v", readErr)
	}
}

func TestMainCommandAdapterAppliesGoBodiesAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.go")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx, instance, event) => {
      instance.log = "closed";
    }),
    hsm.Transition(
      hsm.On("open"),
      hsm.Target("../open"),
    ),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"globals":[{"id":"adapter_global_format","code":"func adapterFormat(value string) string { return strings.ToUpper(value) }","imports":[{"path":"strings"}]}],"behaviors":[{"id":"entry_1","body":"var buffer bytes.Buffer\n_ = buffer\n_ = adapterFormat(\"closed\")","imports":[{"path":"bytes"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "go", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`"bytes"`,
		`hsm "github.com/stateforward/hsm.go"`,
		`"context"`,
		`"strings"`,
		`func adapterFormat(value string) string { return strings.ToUpper(value) }`,
		`func Entry1(ctx context.Context, instance hsm.Instance, event hsm.Event)`,
		`var buffer bytes.Buffer`,
		`_ = adapterFormat("closed")`,
		`var DoorModel = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("closed")`,
		`hsm.State(`,
		`"closed"`,
		`hsm.Entry(Entry1)`,
		`hsm.Transition(`,
		`hsm.On(hsm.Event{Name: "open"})`,
		`hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command Go adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("command Go adapter CLI did not replace source behavior body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesGoOperationBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.go")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"operation_1","body":"approvals.Record(instance, \"approve\")","imports":[{"path":"example.com/approvals"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "go", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command operation adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`"context"`,
		`hsm "github.com/stateforward/hsm.go"`,
		`"example.com/approvals"`,
		`func Operation1(ctx context.Context, instance hsm.Instance, event hsm.Event)`,
		`approvals.Record(instance, "approve")`,
		`var DoorModel = hsm.Define(`,
		`hsm.Operation("approve", Operation1)`,
		`hsm.Initial(`,
		`hsm.Target("closed")`,
		`hsm.State(`,
		`"closed"`,
		`hsm.Transition(`,
		`hsm.OnCall("approve")`,
		`hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command Go operation adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) ||
		strings.Contains(text, "function operation1(") {
		t.Fatalf("command Go operation adapter CLI did not replace source operation body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesRustBodiesAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "lib.rs")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx, instance, event) => {
      instance.log = "closed";
    }),
    hsm.Transition(
      hsm.On("open"),
      hsm.Target("../open"),
    ),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"globals":[{"id":"adapter_global_format","code":"fn adapter_format(value: &str) -> String { adapter_global_format(value) }","imports":[{"path":"crate::global_format","specifiers":[{"name":"adapter_global_format"}]}]}],"behaviors":[{"id":"entry_1","body":"adapter_behavior_format(&adapter_format(\"closed\"));","imports":[{"path":"crate::behavior_format","specifiers":[{"name":"adapter_behavior_format"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "rust", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		"use std::future::Future;",
		"use std::pin::Pin;",
		"use crate::global_format::{adapter_global_format};",
		"use crate::behavior_format::{adapter_behavior_format};",
		"pub struct HsmcInstance;",
		"fn adapter_format(value: &str) -> String { adapter_global_format(value) }",
		"fn entry_1(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) -> Pin<Box<dyn Future<Output = ()> + Send>>",
		`adapter_behavior_format(&adapter_format("closed"));`,
		"Box::pin(async {})",
		`pub fn door_model() -> hsm::Model<HsmcInstance> {`,
		`hsm::define!("Door",`,
		`hsm::initial!(hsm::target!("closed")),`,
		`hsm::state!("closed",`,
		`hsm::entry!(entry_1)`,
		`hsm::transition!(hsm::on!("open"), hsm::target!("../open"))`,
		`hsm::state!("open")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command Rust adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("command Rust adapter CLI did not replace source behavior body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesJavaBodiesAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "GeneratedHsm.java")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx, instance, event) => {
      instance.log = "closed";
    }),
    hsm.Transition(
      hsm.On("open"),
      hsm.Target("../open"),
    ),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"globals":[{"id":"adapter_global_format","code":"private static String AdapterFormat(String value) { return AdapterGlobalFormat.format(value); }","imports":[{"path":"com.example.AdapterGlobalFormat"}]}],"behaviors":[{"id":"entry_1","body":"AdapterBehaviorFormat.record(AdapterFormat(\"closed\"));","imports":[{"path":"com.example.AdapterBehaviorFormat"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "java", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		"import com.stateforward.hsm.*;",
		"import com.example.AdapterGlobalFormat;",
		"import com.example.AdapterBehaviorFormat;",
		"public final class GeneratedHsm",
		"private static String AdapterFormat(String value) { return AdapterGlobalFormat.format(value); }",
		"private static void Entry1(Context ctx, Instance instance, Event event)",
		`AdapterBehaviorFormat.record(AdapterFormat("closed"));`,
		`public static final Model DoorModel = Hsm.Define(`,
		`Hsm.Initial(Hsm.Target("closed"))`,
		`Hsm.State(`,
		`"closed"`,
		`Hsm.Entry(GeneratedHsm::Entry1)`,
		`Hsm.Transition(`,
		`Hsm.On("open")`,
		`Hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command Java adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("command Java adapter CLI did not replace source behavior body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesJavaOperationBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "GeneratedHsm.java")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"operation_1","body":"Approvals.record(instance, \"approve\");","imports":[{"path":"com.example.Approvals"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "java", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import com.stateforward.hsm.*;`,
		`import com.example.Approvals;`,
		`private static void Operation1(Context ctx, Instance instance, Event event)`,
		`Approvals.record(instance, "approve");`,
		`public static final Model DoorModel = Hsm.Define(`,
		`Hsm.Operation("approve", GeneratedHsm::Operation1)`,
		`Hsm.Initial(`,
		`Hsm.Target("closed")`,
		`Hsm.State(`,
		`"closed"`,
		`Hsm.Transition(`,
		`Hsm.OnCall("approve")`,
		`Hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command Java operation adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) ||
		strings.Contains(text, "function operation1(") {
		t.Fatalf("command Java operation adapter CLI did not replace source operation body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesZigBodiesAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.zig")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx, instance, event) => {
      instance.log = "closed";
    }),
    hsm.Transition(
      hsm.On("open"),
      hsm.Target("../open"),
    ),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"globals":[{"id":"adapter_global_format","code":"fn adapter_format(value: []const u8) []const u8 { return adapter_global_format.format(value); }","imports":[{"path":"global_format","alias":"adapter_global_format"}]}],"behaviors":[{"id":"entry_1","body":"_ = adapter_behavior_format.format(adapter_format(\"closed\"));","imports":[{"path":"behavior_format","alias":"adapter_behavior_format"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "zig", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`const std = @import("std");`,
		`const hsm = @import("hsm");`,
		`const adapter_global_format = @import("global_format");`,
		`const adapter_behavior_format = @import("behavior_format");`,
		"fn adapter_format(value: []const u8) []const u8 { return adapter_global_format.format(value); }",
		"fn entry_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void",
		`_ = adapter_behavior_format.format(adapter_format("closed"));`,
		`pub const door_model = hsm.define("Door", .{`,
		`hsm.initial(hsm.target("closed")),`,
		`hsm.state("closed", .{`,
		`hsm.entry(entry_1),`,
		`hsm.transition(.{`,
		`hsm.on("open"),`,
		`hsm.target("../open"),`,
		`hsm.state("open", .{`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command Zig adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("command Zig adapter CLI did not replace source behavior body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesZigOperationBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.zig")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"operation_1","body":"_ = ctx;\n_ = event;\napprovals.record(inst, \"approve\");","imports":[{"path":"approvals.zig","alias":"approvals"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "zig", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`const approvals = @import("approvals.zig");`,
		"fn operation_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void",
		`approvals.record(inst, "approve");`,
		`pub const door_model = hsm.define("Door", .{`,
		`hsm.operation("approve", operation_1),`,
		`hsm.initial(hsm.target("closed")),`,
		`hsm.state("closed", .{`,
		`hsm.transition(.{`,
		`hsm.onCall("approve"),`,
		`hsm.target("../open"),`,
		`hsm.state("open", .{`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command Zig operation adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) ||
		strings.Contains(text, "Original operation approve preserved") ||
		strings.Contains(text, "on_call trigger lowered to event trigger") ||
		strings.Contains(text, `hsm.on("approve")`) {
		t.Fatalf("command Zig operation adapter CLI did not preserve the target operation boundary cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterRejectsZigOperationFullCodeSignatureChange(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.zig")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"operation_1","code":"fn operation_1(ctx: *hsm.Context, instance: *hsm.Instance, event: hsm.Event) void { _ = ctx; _ = instance; _ = event; }"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "zig", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted Zig operation full-code signature change:\n%s", data)
	}
	if !strings.Contains(string(data), `declares parameters "(ctx: *hsm.Context, instance: *hsm.Instance, event: hsm.Event)", want compiler-owned parameters "(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event)"`) {
		t.Fatalf("go run hsmc error missing Zig operation signature rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("Zig operation full-code signature rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected command output: %v", readErr)
	}
}

func TestMainCommandAdapterAppliesJavaScriptBodiesAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.js")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx, instance, event) => {
      instance.log = "closed";
    }),
    hsm.Transition(
      hsm.On("open"),
      hsm.Target("../open"),
    ),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"globals":[{"id":"adapter_global_format","code":"const adapterFormat = (value) => adapterGlobalFormat(value);","imports":[{"path":"./global-format","specifiers":[{"name":"adapterGlobalFormat"}]}]}],"behaviors":[{"id":"entry_1","body":"void adapterBehaviorFormat(adapterFormat(\"closed\"));","imports":[{"path":"./behavior-format","specifiers":[{"name":"adapterBehaviorFormat"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "javascript", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import * as hsm from "@stateforward/hsm";`,
		`import { adapterGlobalFormat } from "./global-format";`,
		`import { adapterBehaviorFormat } from "./behavior-format";`,
		`const adapterFormat = (value) => adapterGlobalFormat(value);`,
		"function entry1(ctx, instance, event)",
		`void adapterBehaviorFormat(adapterFormat("closed"));`,
		`export const DoorModel = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("closed")`,
		`hsm.State(`,
		`"closed"`,
		`hsm.Entry(entry1)`,
		`hsm.Transition(`,
		`hsm.On("open")`,
		`hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command JavaScript adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("command JavaScript adapter CLI did not replace source behavior body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesTypeScriptBodiesAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.ts.out")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx, instance, event) => {
      instance.log = "closed";
    }),
    hsm.Transition(
      hsm.On("open"),
      hsm.Target("../open"),
    ),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"globals":[{"id":"adapter_global_format","code":"const adapterFormat = (value: string): string => adapterGlobalFormat(value);","imports":[{"path":"./global-format","specifiers":[{"name":"adapterGlobalFormat"}]}]}],"behaviors":[{"id":"entry_1","body":"void adapterBehaviorFormat(adapterFormat(\"closed\"));","imports":[{"path":"./behavior-format","specifiers":[{"name":"adapterBehaviorFormat"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "typescript", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import * as hsm from "@stateforward/hsm.ts";`,
		`import { adapterGlobalFormat } from "./global-format";`,
		`import { adapterBehaviorFormat } from "./behavior-format";`,
		`const adapterFormat = (value: string): string => adapterGlobalFormat(value);`,
		"function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void",
		`void adapterBehaviorFormat(adapterFormat("closed"));`,
		`export const DoorModel = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("closed")`,
		`hsm.State(`,
		`"closed"`,
		`hsm.Entry(entry1)`,
		`hsm.Transition(`,
		`hsm.On("open")`,
		`hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command TypeScript adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("command TypeScript adapter CLI did not replace source behavior body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesJavaScriptOperationBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.js")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"operation_1","body":"record(instance, \"approve\");","imports":[{"path":"./approvals","specifiers":[{"name":"record"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "javascript", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import * as hsm from "@stateforward/hsm";`,
		`import { record } from "./approvals";`,
		`function operation1(ctx, instance, event)`,
		`record(instance, "approve");`,
		`export const DoorModel = hsm.Define(`,
		`hsm.Operation("approve", operation1)`,
		`hsm.Initial(`,
		`hsm.Target("closed")`,
		`hsm.State(`,
		`"closed"`,
		`hsm.Transition(`,
		`hsm.OnCall("approve")`,
		`hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command JavaScript operation adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) {
		t.Fatalf("command JavaScript operation adapter CLI did not replace source operation body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesTypeScriptOperationBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.ts.out")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"operation_1","body":"record(instance, \"approve\");","imports":[{"path":"./approvals","specifiers":[{"name":"record"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "typescript", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import * as hsm from "@stateforward/hsm.ts";`,
		`import { record } from "./approvals";`,
		`function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void`,
		`record(instance, "approve");`,
		`export const DoorModel = hsm.Define(`,
		`hsm.Operation("approve", operation1)`,
		`hsm.Initial(`,
		`hsm.Target("closed")`,
		`hsm.State(`,
		`"closed"`,
		`hsm.Transition(`,
		`hsm.OnCall("approve")`,
		`hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command TypeScript operation adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) {
		t.Fatalf("command TypeScript operation adapter CLI did not replace source operation body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesOperationBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"operation_1","body":"instance.approved = approval_value(\"approve\")","imports":[{"path":"approvals","specifiers":[{"name":"approval_value"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		"from approvals import approval_value",
		"async def operation_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:",
		`instance.approved = approval_value("approve")`,
		`door_model = hsm.Define(`,
		`hsm.Operation("approve", operation_1)`,
		`hsm.Initial(`,
		`hsm.Target("closed")`,
		`hsm.State(`,
		`"closed"`,
		`hsm.Transition(`,
		`hsm.OnCall("approve")`,
		`hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command operation adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) ||
		strings.Contains(text, "Original typescript global") ||
		strings.Contains(text, "function operation1(") {
		t.Fatalf("command operation adapter CLI did not replace source operation body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesGlobalCodeAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function record(value: string): string {
  return value.toUpperCase();
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"globals":[{"id":"global_1","code":"def record(value: str) -> str:\n\treturn normalize(value)","imports":[{"path":"helpers","specifiers":[{"name":"normalize"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		"from helpers import normalize",
		"def record(value: str) -> str:",
		"return normalize(value)",
		`door_model = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("closed")`,
		`hsm.State(`,
		`"closed"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript global global_1 preserved for manual porting") ||
		strings.Contains(text, "function record(value: string): string") {
		t.Fatalf("adapter CLI did not replace source global cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterRejectsTranslatedGlobalGeneratedSymbolCollisions(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function record(value: string): string {
  return value.toUpperCase();
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = record("closed");
  })),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"globals":[{"id":"global_1","code":"def helper(value: str) -> str:\n\treturn value\ndef entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:\n\tpass"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted translated global compiler-owned symbol collision:\n%s", data)
	}
	want := `target-language global "global_1" declares "entry_1", which collides with compiler-owned python symbol`
	if !strings.Contains(string(data), want) {
		t.Fatalf("go run hsmc error missing translated global symbol collision rejection %q:\n%s", want, data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("translated global symbol collision rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected translated global symbol collision output: %v", readErr)
	}
}

func TestMainCommandAdapterRejectsHelperGlobalGeneratedFallbackCollision(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.hsm.json")
	output := filepath.Join(dir, "timer.py")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `{
  "source_language": "typescript",
  "models": [{
    "id": "model_1",
    "name": "Timer",
    "initial": {"id": "initial_1", "owner_id": "model_1", "target": "idle"},
    "states": [
      {
        "id": "state_1",
        "name": "idle",
        "kind": "state",
        "transitions": [{
          "id": "transition_1",
          "owner_id": "state_1",
          "target": "../done",
          "trigger": {"kind": "after", "value": "Date.now() + 1000", "language": "typescript"}
        }]
      },
      {"id": "state_2", "name": "done", "kind": "state"}
    ]
  }]
}`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"globals":[{"id":"adapter_global_collision","code":"def hsmc_trigger_fallback_1(ctx, instance, event):\n\treturn 1.0"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "json-ir", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted helper global fallback collision:\n%s", data)
	}
	want := `adapter-created global "adapter_global_collision" declares "hsmc_trigger_fallback_1", which collides with compiler-owned python symbol`
	if !strings.Contains(string(data), want) {
		t.Fatalf("go run hsmc error missing helper global fallback collision rejection %q:\n%s", want, data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("helper global fallback collision rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected helper global fallback collision output: %v", readErr)
	}
}

func TestMainCommandAdapterCreatesHelperGlobalWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"globals":[{"id":"adapter_global_format","code":"def adapter_format(value: str) -> str:\n\treturn value.upper()"}],"behaviors":[{"id":"entry_1","body":"instance.log = adapter_format(\"closed\")"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		"def adapter_format(value: str) -> str:",
		"return value.upper()",
		"async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:",
		`instance.log = adapter_format("closed")`,
		`door_model = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("closed")`,
		`hsm.State(`,
		`"closed"`,
		`hsm.Entry(entry_1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("adapter helper-global CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("adapter helper-global CLI did not replace source behavior cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterRejectsBehaviorFullCodeExtraTopLevelDeclarations(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"entry_1","code":"async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:\n\tinstance.log = [\"closed\"]\ndef door_model():\n\treturn None"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted behavior full-code extra top-level declaration:\n%s", data)
	}
	if !strings.Contains(string(data), "contains extra top-level code after the compiler-owned callable") {
		t.Fatalf("go run hsmc error missing extra top-level behavior code rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("behavior full-code extra top-level rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected command output: %v", readErr)
	}
}

func TestMainCommandAdapterRejectsPythonExpressionBehaviorFullCodeExtraTopLevelDeclarations(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"entry_1","code":"lambda ctx, instance, event: setattr(instance, \"log\", \"closed\"); door_model = None"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted Python expression behavior full-code extra top-level declaration:\n%s", data)
	}
	if !strings.Contains(string(data), "contains extra top-level code after the compiler-owned callable") {
		t.Fatalf("go run hsmc error missing Python expression extra top-level behavior code rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("Python expression behavior full-code extra top-level rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected command output: %v", readErr)
	}
}

func TestMainCommandAdapterRejectsTypeScriptExpressionBehaviorFullCodeExtraTopLevelDeclarations(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.go")
	output := filepath.Join(dir, "door.ts")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `package door

import hsm "github.com/stateforward/hsm"

var DoorModel = hsm.Define("Door",
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed", hsm.Entry(func(ctx hsm.Context, instance hsm.Instance, event hsm.Event) {
		instance.Set("log", "closed")
	})),
)
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"entry_1","code":"(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void => instance.ready = true\nconst DoorModel = null"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "go", "-to", "typescript", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted TypeScript expression behavior full-code extra top-level declaration:\n%s", data)
	}
	if !strings.Contains(string(data), "contains extra top-level code after the compiler-owned callable") {
		t.Fatalf("go run hsmc error missing TypeScript expression extra top-level behavior code rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("TypeScript expression behavior full-code extra top-level rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected command output: %v", readErr)
	}
}

func TestMainCommandAdapterRejectsDartArrowDeclarationBehaviorFullCodeSignatureChange(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.dart")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"entry_1","code":"FutureOr<void> entry1(Context ctx) => adapterFormat(\"entry\");"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "dart", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted Dart arrow declaration behavior full-code signature change:\n%s", data)
	}
	if !strings.Contains(string(data), `declares parameters "(Context ctx)", want compiler-owned parameters "(Context ctx, Instance instance, Event event)"`) {
		t.Fatalf("go run hsmc error missing Dart arrow declaration signature rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("Dart arrow declaration behavior full-code signature rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected command output: %v", readErr)
	}
}

func TestMainCommandAdapterAppliesDartBodiesAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.dart")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx, instance, event) => {
      instance.log = "closed";
    }),
    hsm.Transition(
      hsm.On("open"),
      hsm.Target("../open"),
    ),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"globals":[{"id":"adapter_global_format","code":"String adapterFormat(String value) => adapterGlobalFormat(value);","imports":[{"path":"global_format.dart","specifiers":[{"name":"adapterGlobalFormat"}]}]}],"behaviors":[{"id":"entry_1","body":"adapterBehaviorFormat(adapterFormat(\"closed\"));","imports":[{"path":"behavior_format.dart","specifiers":[{"name":"adapterBehaviorFormat"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "dart", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import "global_format.dart" show adapterGlobalFormat;`,
		`import "behavior_format.dart" show adapterBehaviorFormat;`,
		`String adapterFormat(String value) => adapterGlobalFormat(value);`,
		"FutureOr<void> entry1(Context ctx, Instance instance, Event event)",
		`adapterBehaviorFormat(adapterFormat("closed"));`,
		`final doorModel = define(`,
		`initial(target("closed"))`,
		`state(`,
		`entry(entry1)`,
		`transition([`,
		`on("open")`,
		`target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command Dart adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("command Dart adapter CLI did not replace source behavior body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesDartOperationBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.dart")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"operation_1","body":"adapterApprove(instance, \"approve\");","imports":[{"path":"approvals.dart","specifiers":[{"name":"adapterApprove"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "dart", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import "approvals.dart" show adapterApprove;`,
		"FutureOr<void> operation1(Context ctx, Instance instance, Event event)",
		`adapterApprove(instance, "approve");`,
		`final doorModel = define(`,
		`operation("approve", operation1)`,
		`initial(target("closed"))`,
		`state(`,
		`transition([`,
		`onCall("approve")`,
		`target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command Dart operation adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) ||
		strings.Contains(text, "Original typescript global") ||
		strings.Contains(text, "function operation1(") {
		t.Fatalf("command Dart operation adapter CLI did not replace source operation body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesDartWhenTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.dart")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.When((ctx, instance, event) => {
        return instance.ready;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return ready(instance);","imports":[{"path":"signals.dart","specifiers":[{"name":"ready"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "dart", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import "signals.dart" show ready;`,
		`bool trigger1(Context ctx, Instance instance, Event event)`,
		`return ready(instance);`,
		`final timerModel = define(`,
		`initial(target("idle"))`,
		`state(`,
		`"idle"`,
		`transition([`,
		`when(trigger1)`,
		`target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command Dart when trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return instance.ready;") {
		t.Fatalf("command Dart when trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesDartAtTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.dart")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return deadlineNow();","imports":[{"path":"clock.dart","specifiers":[{"name":"deadlineNow"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "dart", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import "dart:async";`,
		`import "clock.dart" show deadlineNow;`,
		`FutureOr<DateTime> trigger1(Context ctx, Instance instance, Event event)`,
		`return deadlineNow();`,
		`final timerModel = define(`,
		`initial(target("idle"))`,
		`state(`,
		`transition([`,
		`at(trigger1)`,
		`target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command Dart at trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return new Date();") ||
		strings.Contains(text, "Unsupported at trigger") ||
		strings.Contains(text, "_hsmcAtTransition") {
		t.Fatalf("command Dart at trigger adapter CLI did not preserve the target trigger boundary cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesRustOperationBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "lib.rs")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"operation_1","body":"let _ = approval_value(\"approve\");\nBox::pin(async {})","imports":[{"path":"crate::approvals","specifiers":[{"name":"approval_value"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "rust", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`use crate::approvals::{approval_value};`,
		`fn operation_1(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) -> Pin<Box<dyn Future<Output = ()> + Send>>`,
		`let _ = approval_value("approve");`,
		`Box::pin(async {})`,
		`pub fn door_model() -> hsm::Model<HsmcInstance>`,
		`hsm::operation("approve", operation_1)`,
		`hsm::initial!(hsm::target!("closed"))`,
		`hsm::state!("closed"`,
		`hsm::transition!(hsm::on_call("approve"), hsm::target!("../open"))`,
		`hsm::state!("open")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command Rust operation adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) ||
		strings.Contains(text, "Original typescript global") ||
		strings.Contains(text, "function operation1(") ||
		strings.Contains(text, "hsm.rs does not expose an operation partial") ||
		strings.Contains(text, "on_call trigger lowered to event trigger") {
		t.Fatalf("command Rust operation adapter CLI did not replace source operation body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesRustAtTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "lib.rs")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"deadline()","imports":[{"path":"crate::clock","specifiers":[{"name":"deadline"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "rust", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`use crate::clock::{deadline};`,
		`fn trigger_1(ctx: &hsm::Context, instance: &HsmcInstance, event: &hsm::Event) -> std::time::SystemTime`,
		`deadline()`,
		`pub fn timer_model() -> hsm::Model<HsmcInstance>`,
		`hsm::initial!(hsm::target!("idle"))`,
		`hsm::state!("idle"`,
		`hsm::transition!(hsm::at!(trigger_1), hsm::target!("../done"))`,
		`hsm::state!("done")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command Rust at trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{
		"Original typescript behavior trigger_1 preserved for manual porting",
		"return new Date();",
		"at trigger lowered to after trigger",
		"hsm::after!(trigger_1)",
		"hsm::after!(hsmc_zero_duration)",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("command Rust at trigger adapter CLI output contains forbidden %q:\n%s", forbidden, text)
		}
	}
}

func TestMainCommandAdapterAppliesRustWhenTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "lib.rs")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.When((ctx, instance, event) => {
        return instance.ready;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"ready(instance)","imports":[{"path":"crate::signals","specifiers":[{"name":"ready"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "rust", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command Rust when trigger adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`use crate::signals::{ready};`,
		`fn trigger_1(ctx: &hsm::Context, instance: &HsmcInstance, event: &hsm::Event) -> bool`,
		`ready(instance)`,
		`pub fn timer_model() -> hsm::Model<HsmcInstance>`,
		`hsm::define!("Timer"`,
		`hsm::initial!(hsm::target!("idle"))`,
		`hsm::state!("idle"`,
		`/* when trigger lowered to guard for hsm.rs */ hsm::guard!(trigger_1)`,
		`hsm::target!("../done")`,
		`hsm::state!("done")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command Rust when trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return instance.ready;") ||
		strings.Contains(text, "hsm::when") ||
		strings.Contains(text, "Unsupported when trigger") {
		t.Fatalf("command Rust when trigger adapter CLI did not preserve the target trigger boundary cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterRejectsRustOperationFullCodeSignatureChange(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "lib.rs")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"operation_1","code":"fn operation_1(ctx: &hsm::Context, instance: &HsmcInstance, event: &hsm::Event) -> Pin<Box<dyn Future<Output = ()> + Send>> { Box::pin(async {}) }"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "rust", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted Rust operation full-code signature change:\n%s", data)
	}
	if !strings.Contains(string(data), `declares parameters "(ctx: &hsm::Context, instance: &HsmcInstance, event: &hsm::Event)", want compiler-owned parameters "(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event)"`) {
		t.Fatalf("go run hsmc error missing Rust operation signature rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("Rust operation full-code signature rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected command output: %v", readErr)
	}
}

func TestMainCommandAdapterAppliesTriggerBodyWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.go")
	output := filepath.Join(dir, "timer.ts")
	adapter := filepath.Join(dir, "adapter.sh")
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
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return delay(250);","imports":[{"path":"./timers","specifiers":[{"name":"delay"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "go", "-to", "typescript", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import { delay } from "./timers";`,
		"function trigger1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): number | Date | Promise<number> | Promise<Date> {",
		"return delay(250);",
		"hsm.After(trigger1)",
		`export const TimerModel = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("idle")`,
		`hsm.State(`,
		`"idle"`,
		`hsm.Transition(`,
		`hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original go behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return time.Second") {
		t.Fatalf("command trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesPythonWhenTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.py")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.When((ctx, instance, event) => {
        return instance.ready;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return ready(instance)","imports":[{"path":"signals","specifiers":[{"name":"ready"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import datetime`,
		`import hsm`,
		`import typing`,
		`from signals import ready`,
		`async def trigger_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> typing.Any:`,
		`return ready(instance)`,
		`timer_model = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("idle")`,
		`hsm.State(`,
		`"idle"`,
		`hsm.Transition(`,
		`hsm.When(trigger_1)`,
		`hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command Python when trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return instance.ready;") {
		t.Fatalf("command Python when trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesPythonAtTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.py")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return deadline_now()","imports":[{"path":"clock","specifiers":[{"name":"deadline_now"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import datetime`,
		`import hsm`,
		`from clock import deadline_now`,
		`async def trigger_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> datetime.datetime:`,
		`return deadline_now()`,
		`timer_model = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("idle")`,
		`hsm.State(`,
		`"idle"`,
		`hsm.Transition(`,
		`hsm.At(trigger_1)`,
		`hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command Python at trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return new Date();") {
		t.Fatalf("command Python at trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesJavaScriptWhenTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.js")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.When((ctx, instance, event) => {
        return instance.ready;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return ready(instance);","imports":[{"path":"./signals","specifiers":[{"name":"ready"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "javascript", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import * as hsm from "@stateforward/hsm";`,
		`import { ready } from "./signals";`,
		`function trigger1(ctx, instance, event)`,
		`return ready(instance);`,
		`export const TimerModel = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("idle")`,
		`hsm.State(`,
		`"idle"`,
		`hsm.Transition(`,
		`hsm.When(trigger1)`,
		`hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command JavaScript when trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return instance.ready;") {
		t.Fatalf("command JavaScript when trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesJavaScriptAtTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.js")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return deadlineNow();","imports":[{"path":"./clock","specifiers":[{"name":"deadlineNow"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "javascript", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import * as hsm from "@stateforward/hsm";`,
		`import { deadlineNow } from "./clock";`,
		`function trigger1(ctx, instance, event)`,
		`return deadlineNow();`,
		`export const TimerModel = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("idle")`,
		`hsm.State(`,
		`"idle"`,
		`hsm.Transition(`,
		`hsm.At(trigger1)`,
		`hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command JavaScript at trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return new Date();") {
		t.Fatalf("command JavaScript at trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesTypeScriptWhenTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.ts.out")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.When((ctx, instance, event) => {
        return instance.ready;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return ready(instance);","imports":[{"path":"./signals","specifiers":[{"name":"ready"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "typescript", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import * as hsm from "@stateforward/hsm.ts";`,
		`import { ready } from "./signals";`,
		`function trigger1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): unknown`,
		`return ready(instance);`,
		`export const TimerModel = (hsm.Define as any)(`,
		`hsm.Initial(`,
		`hsm.Target("idle")`,
		`hsm.State(`,
		`"idle"`,
		`hsm.Transition(`,
		`(hsm.When as any)(trigger1)`,
		`) as any,`,
		`) as any,`,
		`hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command TypeScript when trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return instance.ready;") {
		t.Fatalf("command TypeScript when trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesTypeScriptAtTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.ts.out")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return deadlineNow();","imports":[{"path":"./clock","specifiers":[{"name":"deadlineNow"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "typescript", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import * as hsm from "@stateforward/hsm.ts";`,
		`import { deadlineNow } from "./clock";`,
		`function trigger1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): Date | Promise<Date>`,
		`return deadlineNow();`,
		`export const TimerModel = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("idle")`,
		`hsm.State(`,
		`"idle"`,
		`hsm.Transition(`,
		`hsm.At(trigger1)`,
		`hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command TypeScript at trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return new Date();") {
		t.Fatalf("command TypeScript at trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesZigWhenTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.zig")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.When((ctx, instance, event) => {
        return instance.ready;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return signals.ready(inst);","imports":[{"path":"./signals.zig","alias":"signals"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "zig", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`const hsm = @import("hsm");`,
		`const signals = @import("./signals.zig");`,
		`fn trigger_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) bool`,
		`return signals.ready(inst);`,
		`pub const timer_model = hsm.define("Timer", .{`,
		`hsm.initial(hsm.target("idle"))`,
		`hsm.state("idle", .{`,
		`hsm.transition(.{`,
		`// when trigger lowered to guard for hsm.zig`,
		`hsm.guard(trigger_1),`,
		`hsm.target("../done")`,
		`hsm.state("done", .{`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command Zig when trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return instance.ready;") ||
		strings.Contains(text, "Unsupported when trigger") ||
		strings.Contains(text, "hsm.when") {
		t.Fatalf("command Zig when trigger adapter CLI did not preserve the intended boundary:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesZigAtTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.zig")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return clock.deadline();","imports":[{"path":"clock","alias":"clock"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "zig", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`const hsm = @import("hsm");`,
		`const clock = @import("clock");`,
		`fn trigger_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) u64`,
		`return clock.deadline();`,
		`pub const timer_model = hsm.define("Timer", .{`,
		`hsm.initial(hsm.target("idle"))`,
		`hsm.state("idle", .{`,
		`hsm.transition(.{`,
		`hsm.at(trigger_1),`,
		`hsm.target("../done")`,
		`hsm.state("done", .{`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command Zig at trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{
		"Original typescript behavior trigger_1 preserved for manual porting",
		"return new Date();",
		"Unsupported at trigger",
		"at trigger lowered to after trigger",
		"hsm.after(trigger_1)",
		"hsm.after(hsmc_trigger_zero)",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("command Zig at trigger adapter CLI output contains forbidden %q:\n%s", forbidden, text)
		}
	}
}

func TestMainCommandAdapterAppliesGoTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.go")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.After((ctx, instance, event) => {
        return 1000;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return timers.Delay(250) * time.Millisecond","imports":[{"path":"example.com/timers","alias":"timers"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "go", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`"context"`,
		`timers "example.com/timers"`,
		`hsm "github.com/stateforward/hsm.go"`,
		`"time"`,
		`func Trigger1(ctx context.Context, instance hsm.Instance, event hsm.Event) time.Duration`,
		`return timers.Delay(250) * time.Millisecond`,
		`var TimerModel = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("idle")`,
		`hsm.State(`,
		`"idle"`,
		`hsm.Transition(`,
		`hsm.After(Trigger1)`,
		`hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command Go trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return 1000;") {
		t.Fatalf("command Go trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesGoAtTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.go")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return deadline.Now()","imports":[{"path":"example.com/deadline"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "go", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command Go at trigger adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`"context"`,
		`"example.com/deadline"`,
		`hsm "github.com/stateforward/hsm.go"`,
		`"time"`,
		`func Trigger1(ctx context.Context, instance hsm.Instance, event hsm.Event) time.Time`,
		`return deadline.Now()`,
		`var TimerModel = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("idle")`,
		`hsm.State(`,
		`"idle"`,
		`hsm.Transition(`,
		`hsm.At(Trigger1)`,
		`hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command Go at trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return new Date();") {
		t.Fatalf("command Go at trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesCSharpWhenTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "GeneratedHsm.cs")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.When((ctx, instance, event) => {
        return instance.ready;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return Ready.WaitAsync(cancellationToken);","imports":[{"path":"Sample.Ready.Helper","alias":"Ready"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "csharp", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`using Stateforward.Hsm;`,
		`using Ready = Sample.Ready.Helper;`,
		`private static System.Threading.Tasks.Task Trigger1(Context ctx, Instance instance, Event @event, System.Threading.CancellationToken cancellationToken)`,
		`return Ready.WaitAsync(cancellationToken);`,
		`public static readonly Model TimerModel = Hsm.Define(`,
		`Hsm.Initial(`,
		`Hsm.Target("idle")`,
		`Hsm.State(`,
		`"idle"`,
		`Hsm.Transition(`,
		`Hsm.When<Instance>(Trigger1)`,
		`Hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command C# when trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return instance.ready;") {
		t.Fatalf("command C# when trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesCSharpAtTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "GeneratedHsm.cs")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return Clock.Deadline();","imports":[{"path":"Sample.Clock","alias":"Clock"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "csharp", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`using Stateforward.Hsm;`,
		`using Clock = Sample.Clock;`,
		`private static System.DateTimeOffset Trigger1(Context ctx, Instance instance, Event @event)`,
		`return Clock.Deadline();`,
		`public static readonly Model TimerModel = Hsm.Define(`,
		`Hsm.Initial(`,
		`Hsm.Target("idle")`,
		`Hsm.State(`,
		`"idle"`,
		`Hsm.Transition(`,
		`Hsm.At<Instance>(Trigger1)`,
		`Hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command C# at trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return new Date();") {
		t.Fatalf("command C# at trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesJavaWhenTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "GeneratedHsm.java")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.When((ctx, instance, event) => {
        return instance.ready;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return Ready.isReady(instance);","imports":[{"path":"com.example.Ready"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "java", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import com.stateforward.hsm.*;`,
		`import com.example.Ready;`,
		`private static boolean Trigger1(Context ctx, Instance instance, Event event)`,
		`return Ready.isReady(instance);`,
		`public static final Model TimerModel = Hsm.Define(`,
		`Hsm.Initial(Hsm.Target("idle"))`,
		`Hsm.State(`,
		`"idle"`,
		`Hsm.Transition(`,
		`Hsm.When(GeneratedHsm::Trigger1)`,
		`Hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command Java when trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return instance.ready;") {
		t.Fatalf("command Java when trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCommandAdapterAppliesJavaAtTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "GeneratedHsm.java")
	adapter := filepath.Join(dir, "adapter.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return Clock.deadline();","imports":[{"path":"com.example.Clock"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "java", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc command adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import com.stateforward.hsm.*;`,
		`import com.example.Clock;`,
		`private static java.time.Instant Trigger1(Context ctx, Instance instance, Event event)`,
		`return Clock.deadline();`,
		`public static final Model TimerModel = Hsm.Define(`,
		`Hsm.Initial(Hsm.Target("idle"))`,
		`Hsm.State(`,
		`"idle"`,
		`Hsm.Transition(`,
		`Hsm.At(GeneratedHsm::Trigger1)`,
		`Hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("command Java at trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return new Date();") {
		t.Fatalf("command Java at trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesOutputLastMessagePatchWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx, instance, event) => {
      instance.log = "closed";
    }),
    hsm.Transition(
      hsm.On("open"),
      hsm.Target("../open"),
    ),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"entry_1","body":"instance.log = [\"translated by codex\"]"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		"async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:",
		`instance.log = ["translated by codex"]`,
		`door_model = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("closed")`,
		`hsm.State(`,
		`"closed"`,
		`hsm.Entry(entry_1)`,
		`hsm.Transition(`,
		`hsm.On("open")`,
		`hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("codex adapter CLI did not replace source behavior body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesPythonBodiesAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx, instance, event) => {
      instance.log = "closed";
    }),
    hsm.Transition(
      hsm.On("open"),
      hsm.Target("../open"),
    ),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"entry_1","body":"instance.log = [record(\"closed\")]","imports":[{"path":"helpers","specifiers":[{"name":"record"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		"from helpers import record",
		"async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:",
		`instance.log = [record("closed")]`,
		`door_model = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("closed")`,
		`hsm.State(`,
		`"closed"`,
		`hsm.Entry(entry_1)`,
		`hsm.Transition(`,
		`hsm.On("open")`,
		`hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Python adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("codex Python adapter CLI did not replace source behavior body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesCSharpBodiesAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "GeneratedHsm.cs")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx, instance, event) => {
      instance.log = "closed";
    }),
    hsm.Transition(
      hsm.On("open"),
      hsm.Target("../open"),
    ),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"globals":[{"id":"adapter_global_format","code":"private static string AdapterFormat(string value) => GlobalFormat.Record(value);","imports":[{"path":"Sample.Global.Helper","alias":"GlobalFormat"}]}],"behaviors":[{"id":"entry_1","body":"BehaviorFormat.Record(AdapterFormat(\"closed\"));","imports":[{"path":"Sample.Behavior.Helper","alias":"BehaviorFormat"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "csharp", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		"using Stateforward.Hsm;",
		"using GlobalFormat = Sample.Global.Helper;",
		"using BehaviorFormat = Sample.Behavior.Helper;",
		"public static class GeneratedHsm",
		`private static string AdapterFormat(string value) => GlobalFormat.Record(value);`,
		"private static void Entry1(Context ctx, Instance instance, Event @event)",
		`BehaviorFormat.Record(AdapterFormat("closed"));`,
		`public static readonly Model DoorModel = Hsm.Define(`,
		`Hsm.Initial(`,
		`Hsm.Target("closed")`,
		`Hsm.State(`,
		`"closed"`,
		`Hsm.Entry<Instance>(Entry1)`,
		`Hsm.Transition(`,
		`Hsm.On("open")`,
		`Hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex C# adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("codex C# adapter CLI did not replace source behavior body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesCSharpOperationBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "GeneratedHsm.cs")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"operation_1","body":"Approvals.Record(instance, \"approve\");","imports":[{"path":"Sample.Approvals","alias":"Approvals"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "csharp", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex C# operation adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		"using Stateforward.Hsm;",
		"using Approvals = Sample.Approvals;",
		"private static void Operation1(Context ctx, Instance instance, Event @event)",
		`Approvals.Record(instance, "approve");`,
		`public static readonly Model DoorModel = Hsm.Define(`,
		`Hsm.Operation("approve", new Operation<Instance>(Operation1))`,
		`Hsm.Initial(`,
		`Hsm.Target("closed")`,
		`Hsm.State(`,
		`"closed"`,
		`Hsm.Transition(`,
		`Hsm.OnCall("approve")`,
		`Hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex C# operation adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) ||
		strings.Contains(text, "function operation1(") {
		t.Fatalf("codex C# operation adapter CLI did not replace source operation body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterRejectsCSharpFieldDeclarationBehaviorFullCodeSignatureChange(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "GeneratedHsm.cs")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"entry_1","code":"private static readonly Operation<Instance> Entry1 = (ctx, instance) => System.GC.KeepAlive(\"entry\");"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "csharp", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc codex unexpectedly accepted C# field declaration behavior full-code signature change:\n%s", data)
	}
	if !strings.Contains(string(data), `declares lambda parameters "(ctx, instance)", want compiler-owned parameters "(ctx, instance, @event)"`) {
		t.Fatalf("go run hsmc codex error missing C# field declaration signature rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("codex C# field declaration behavior full-code signature rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected codex output: %v", readErr)
	}
}

func TestMainCodexAdapterAppliesCPPBodiesAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.cpp")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx, instance, event) => {
      instance.log = "closed";
    }),
    hsm.Transition(
      hsm.On("open"),
      hsm.Target("../open"),
    ),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"globals":[{"id":"adapter_global_format","code":"static std::string adapter_format(std::string value) { return adapter_global_format(value); }","imports":[{"path":"global_format.hpp"}]}],"behaviors":[{"id":"entry_1","body":"(void)adapter_behavior_format(adapter_format(\"closed\"));","imports":[{"path":"behavior_format.hpp"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "cpp", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`#include "hsm/hsm.hpp"`,
		`#include "global_format.hpp"`,
		`#include "behavior_format.hpp"`,
		"namespace generated_hsm",
		"static std::string adapter_format(std::string value) { return adapter_global_format(value); }",
		"static constexpr auto entry_1 = [](auto& signal, auto& instance, const auto& event) -> void",
		`(void)adapter_behavior_format(adapter_format("closed"));`,
		`static constexpr auto door_model = hsm::define(`,
		`hsm::initial(hsm::target("/Door/closed"))`,
		`hsm::state("closed"`,
		`hsm::entry(entry_1)`,
		`hsm::transition(`,
		`hsm::on("open")`,
		`hsm::target("/Door/open")`,
		`hsm::state("open")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex C++ adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("codex C++ adapter CLI did not replace source behavior body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesCPPOperationBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.cpp")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"operation_1","body":"(void)adapter_approve(\"approve\");","imports":[{"path":"approval.hpp"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "cpp", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`#include "hsm/hsm.hpp"`,
		`#include "approval.hpp"`,
		`namespace generated_hsm`,
		`static void operation_1()`,
		`(void)adapter_approve("approve");`,
		`static constexpr auto door_model = hsm::define(`,
		`hsm::operation("approve", &operation_1)`,
		`hsm::initial(hsm::target("/Door/closed"))`,
		`hsm::state("closed"`,
		`hsm::transition(hsm::on_call("approve"), hsm::target("/Door/open"))`,
		`hsm::state("open")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex C++ operation adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) ||
		strings.Contains(text, "function operation1(") ||
		strings.Contains(text, "hsm.cpp requires a function pointer or member pointer") {
		t.Fatalf("codex C++ operation adapter CLI did not replace source operation body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesCPPAtTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.cpp")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return clock::deadline();","imports":[{"path":"clock.hpp"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "cpp", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`#include "hsm/hsm.hpp"`,
		`#include "clock.hpp"`,
		`#include <chrono>`,
		`static constexpr auto trigger_1 = [](auto& signal, auto& instance, const auto& event) -> std::chrono::system_clock::time_point`,
		`return clock::deadline();`,
		`static constexpr auto timer_model = hsm::define(`,
		`"Timer"`,
		`hsm::initial(hsm::target("/Timer/idle"))`,
		`hsm::state("idle"`,
		`hsm::transition(hsm::at(trigger_1), hsm::target("/Timer/done"))`,
		`hsm::state("done")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex C++ at trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return new Date();") {
		t.Fatalf("codex C++ at trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesCPPWhenTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.cpp")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.When((ctx, instance, event) => {
        return instance.ready;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return ready(instance);","imports":[{"path":"signals.hpp"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "cpp", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex C++ when trigger adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`#include "hsm/hsm.hpp"`,
		`#include "signals.hpp"`,
		`static constexpr auto trigger_1 = [](auto& signal, auto& instance, const auto& event) -> bool`,
		`return ready(instance);`,
		`static constexpr auto timer_model = hsm::define(`,
		`"Timer"`,
		`hsm::initial(hsm::target("/Timer/idle"))`,
		`hsm::state("idle"`,
		`hsm::transition(hsm::guard(trigger_1), hsm::target("/Timer/done"))`,
		`hsm::state("done")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex C++ when trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return instance.ready;") ||
		strings.Contains(text, "Unsupported when trigger") ||
		strings.Contains(text, "hsm::when(trigger_1)") {
		t.Fatalf("codex C++ when trigger adapter CLI did not preserve the target trigger boundary cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterRejectsCPPOperationFullCodeSignatureChange(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.cpp")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"operation_1","code":"static void operation_1(auto& signal) { adapter_format(\"approve\"); }"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "cpp", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc codex unexpectedly accepted C++ operation full-code signature change:\n%s", data)
	}
	if !strings.Contains(string(data), `declares parameters "(auto& signal)", want compiler-owned parameters "()"`) {
		t.Fatalf("go run hsmc codex error missing C++ operation signature rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("codex C++ operation full-code signature rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected codex output: %v", readErr)
	}
}

func TestMainCodexAdapterAppliesGoBodiesAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.go")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx, instance, event) => {
      instance.log = "closed";
    }),
    hsm.Transition(
      hsm.On("open"),
      hsm.Target("../open"),
    ),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"globals":[{"id":"adapter_global_format","code":"func adapterFormat(value string) string { return strings.ToUpper(value) }","imports":[{"path":"strings"}]}],"behaviors":[{"id":"entry_1","body":"var buffer bytes.Buffer\n_ = buffer\n_ = adapterFormat(\"closed\")","imports":[{"path":"bytes"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "go", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`"bytes"`,
		`hsm "github.com/stateforward/hsm.go"`,
		`"context"`,
		`"strings"`,
		`func adapterFormat(value string) string { return strings.ToUpper(value) }`,
		`func Entry1(ctx context.Context, instance hsm.Instance, event hsm.Event)`,
		`var buffer bytes.Buffer`,
		`_ = adapterFormat("closed")`,
		`var DoorModel = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("closed")`,
		`hsm.State(`,
		`"closed"`,
		`hsm.Entry(Entry1)`,
		`hsm.Transition(`,
		`hsm.On(hsm.Event{Name: "open"})`,
		`hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Go adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("codex Go adapter CLI did not replace source behavior body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesGoOperationBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.go")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"operation_1","body":"approvals.Record(instance, \"approve\")","imports":[{"path":"example.com/approvals"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "go", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex operation adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`"context"`,
		`hsm "github.com/stateforward/hsm.go"`,
		`"example.com/approvals"`,
		`func Operation1(ctx context.Context, instance hsm.Instance, event hsm.Event)`,
		`approvals.Record(instance, "approve")`,
		`var DoorModel = hsm.Define(`,
		`hsm.Operation("approve", Operation1)`,
		`hsm.Initial(`,
		`hsm.Target("closed")`,
		`hsm.State(`,
		`"closed"`,
		`hsm.Transition(`,
		`hsm.OnCall("approve")`,
		`hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Go operation adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) ||
		strings.Contains(text, "function operation1(") {
		t.Fatalf("codex Go operation adapter CLI did not replace source operation body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesRustBodiesAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "lib.rs")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx, instance, event) => {
      instance.log = "closed";
    }),
    hsm.Transition(
      hsm.On("open"),
      hsm.Target("../open"),
    ),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"globals":[{"id":"adapter_global_format","code":"fn adapter_format(value: &str) -> String { adapter_global_format(value) }","imports":[{"path":"crate::global_format","specifiers":[{"name":"adapter_global_format"}]}]}],"behaviors":[{"id":"entry_1","body":"adapter_behavior_format(&adapter_format(\"closed\"));","imports":[{"path":"crate::behavior_format","specifiers":[{"name":"adapter_behavior_format"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "rust", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		"use std::future::Future;",
		"use std::pin::Pin;",
		"use crate::global_format::{adapter_global_format};",
		"use crate::behavior_format::{adapter_behavior_format};",
		"pub struct HsmcInstance;",
		"fn adapter_format(value: &str) -> String { adapter_global_format(value) }",
		"fn entry_1(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) -> Pin<Box<dyn Future<Output = ()> + Send>>",
		`adapter_behavior_format(&adapter_format("closed"));`,
		"Box::pin(async {})",
		`pub fn door_model() -> hsm::Model<HsmcInstance> {`,
		`hsm::define!("Door",`,
		`hsm::initial!(hsm::target!("closed")),`,
		`hsm::state!("closed",`,
		`hsm::entry!(entry_1),`,
		`hsm::transition!(hsm::on!("open"), hsm::target!("../open")),`,
		`hsm::state!("open"),`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Rust adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("codex Rust adapter CLI did not replace source behavior body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesJavaBodiesAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "GeneratedHsm.java")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx, instance, event) => {
      instance.log = "closed";
    }),
    hsm.Transition(
      hsm.On("open"),
      hsm.Target("../open"),
    ),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"globals":[{"id":"adapter_global_format","code":"private static String AdapterFormat(String value) { return AdapterGlobalFormat.format(value); }","imports":[{"path":"com.example.AdapterGlobalFormat"}]}],"behaviors":[{"id":"entry_1","body":"AdapterBehaviorFormat.record(AdapterFormat(\"closed\"));","imports":[{"path":"com.example.AdapterBehaviorFormat"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "java", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		"import com.stateforward.hsm.*;",
		"import com.example.AdapterGlobalFormat;",
		"import com.example.AdapterBehaviorFormat;",
		"public final class GeneratedHsm",
		"private static String AdapterFormat(String value) { return AdapterGlobalFormat.format(value); }",
		"private static void Entry1(Context ctx, Instance instance, Event event)",
		`AdapterBehaviorFormat.record(AdapterFormat("closed"));`,
		`public static final Model DoorModel = Hsm.Define(`,
		`Hsm.Initial(Hsm.Target("closed"))`,
		`Hsm.State(`,
		`"closed"`,
		`Hsm.Entry(GeneratedHsm::Entry1)`,
		`Hsm.Transition(`,
		`Hsm.On("open")`,
		`Hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Java adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("codex Java adapter CLI did not replace source behavior body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesJavaOperationBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "GeneratedHsm.java")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"operation_1","body":"Approvals.record(instance, \"approve\");","imports":[{"path":"com.example.Approvals"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "java", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import com.stateforward.hsm.*;`,
		`import com.example.Approvals;`,
		`private static void Operation1(Context ctx, Instance instance, Event event)`,
		`Approvals.record(instance, "approve");`,
		`public static final Model DoorModel = Hsm.Define(`,
		`Hsm.Operation("approve", GeneratedHsm::Operation1)`,
		`Hsm.Initial(`,
		`Hsm.Target("closed")`,
		`Hsm.State(`,
		`"closed"`,
		`Hsm.Transition(`,
		`Hsm.OnCall("approve")`,
		`Hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Java operation adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) ||
		strings.Contains(text, "function operation1(") {
		t.Fatalf("codex Java operation adapter CLI did not replace source operation body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesZigBodiesAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.zig")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx, instance, event) => {
      instance.log = "closed";
    }),
    hsm.Transition(
      hsm.On("open"),
      hsm.Target("../open"),
    ),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"globals":[{"id":"adapter_global_format","code":"fn adapter_format(value: []const u8) []const u8 { return adapter_global_format.format(value); }","imports":[{"path":"global_format","alias":"adapter_global_format"}]}],"behaviors":[{"id":"entry_1","body":"_ = adapter_behavior_format.format(adapter_format(\"closed\"));","imports":[{"path":"behavior_format","alias":"adapter_behavior_format"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "zig", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`const std = @import("std");`,
		`const hsm = @import("hsm");`,
		`const adapter_global_format = @import("global_format");`,
		`const adapter_behavior_format = @import("behavior_format");`,
		"fn adapter_format(value: []const u8) []const u8 { return adapter_global_format.format(value); }",
		"fn entry_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void",
		`_ = adapter_behavior_format.format(adapter_format("closed"));`,
		`pub const door_model = hsm.define("Door", .{`,
		`hsm.initial(hsm.target("closed")),`,
		`hsm.state("closed", .{`,
		`hsm.entry(entry_1),`,
		`hsm.transition(.{`,
		`hsm.on("open"),`,
		`hsm.target("../open"),`,
		`hsm.state("open", .{`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Zig adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("codex Zig adapter CLI did not replace source behavior body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesZigOperationBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.zig")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"operation_1","body":"_ = ctx;\n_ = event;\napprovals.record(inst, \"approve\");","imports":[{"path":"approvals.zig","alias":"approvals"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "zig", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`const approvals = @import("approvals.zig");`,
		"fn operation_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void",
		`approvals.record(inst, "approve");`,
		`pub const door_model = hsm.define("Door", .{`,
		`hsm.operation("approve", operation_1),`,
		`hsm.initial(hsm.target("closed")),`,
		`hsm.state("closed", .{`,
		`hsm.transition(.{`,
		`hsm.onCall("approve"),`,
		`hsm.target("../open"),`,
		`hsm.state("open", .{`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Zig operation adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) ||
		strings.Contains(text, "Original operation approve preserved") ||
		strings.Contains(text, "on_call trigger lowered to event trigger") ||
		strings.Contains(text, `hsm.on("approve")`) {
		t.Fatalf("codex Zig operation adapter CLI did not preserve the target operation boundary cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterRejectsZigOperationFullCodeSignatureChange(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.zig")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"operation_1","code":"fn operation_1(ctx: *hsm.Context, instance: *hsm.Instance, event: hsm.Event) void { _ = ctx; _ = instance; _ = event; }"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "zig", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc codex unexpectedly accepted Zig operation full-code signature change:\n%s", data)
	}
	if !strings.Contains(string(data), `declares parameters "(ctx: *hsm.Context, instance: *hsm.Instance, event: hsm.Event)", want compiler-owned parameters "(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event)"`) {
		t.Fatalf("go run hsmc codex error missing Zig operation signature rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("codex Zig operation full-code signature rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected codex output: %v", readErr)
	}
}

func TestMainCodexAdapterAppliesJavaScriptBodiesAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.js")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx, instance, event) => {
      instance.log = "closed";
    }),
    hsm.Transition(
      hsm.On("open"),
      hsm.Target("../open"),
    ),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"globals":[{"id":"adapter_global_format","code":"const adapterFormat = (value) => adapterGlobalFormat(value);","imports":[{"path":"./global-format","specifiers":[{"name":"adapterGlobalFormat"}]}]}],"behaviors":[{"id":"entry_1","body":"void adapterBehaviorFormat(adapterFormat(\"closed\"));","imports":[{"path":"./behavior-format","specifiers":[{"name":"adapterBehaviorFormat"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "javascript", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import * as hsm from "@stateforward/hsm";`,
		`import { adapterGlobalFormat } from "./global-format";`,
		`import { adapterBehaviorFormat } from "./behavior-format";`,
		`const adapterFormat = (value) => adapterGlobalFormat(value);`,
		"function entry1(ctx, instance, event)",
		`void adapterBehaviorFormat(adapterFormat("closed"));`,
		`export const DoorModel = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("closed")`,
		`hsm.State(`,
		`"closed"`,
		`hsm.Entry(entry1)`,
		`hsm.Transition(`,
		`hsm.On("open")`,
		`hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex JavaScript adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("codex JavaScript adapter CLI did not replace source behavior body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesTypeScriptBodiesAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.ts.out")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx, instance, event) => {
      instance.log = "closed";
    }),
    hsm.Transition(
      hsm.On("open"),
      hsm.Target("../open"),
    ),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"globals":[{"id":"adapter_global_format","code":"const adapterFormat = (value: string): string => adapterGlobalFormat(value);","imports":[{"path":"./global-format","specifiers":[{"name":"adapterGlobalFormat"}]}]}],"behaviors":[{"id":"entry_1","body":"void adapterBehaviorFormat(adapterFormat(\"closed\"));","imports":[{"path":"./behavior-format","specifiers":[{"name":"adapterBehaviorFormat"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "typescript", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import * as hsm from "@stateforward/hsm.ts";`,
		`import { adapterGlobalFormat } from "./global-format";`,
		`import { adapterBehaviorFormat } from "./behavior-format";`,
		`const adapterFormat = (value: string): string => adapterGlobalFormat(value);`,
		"function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void",
		`void adapterBehaviorFormat(adapterFormat("closed"));`,
		`export const DoorModel = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("closed")`,
		`hsm.State(`,
		`"closed"`,
		`hsm.Entry(entry1)`,
		`hsm.Transition(`,
		`hsm.On("open")`,
		`hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex TypeScript adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("codex TypeScript adapter CLI did not replace source behavior body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesJavaScriptOperationBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.js")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"operation_1","body":"record(instance, \"approve\");","imports":[{"path":"./approvals","specifiers":[{"name":"record"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "javascript", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import * as hsm from "@stateforward/hsm";`,
		`import { record } from "./approvals";`,
		`function operation1(ctx, instance, event)`,
		`record(instance, "approve");`,
		`export const DoorModel = hsm.Define(`,
		`hsm.Operation("approve", operation1)`,
		`hsm.Initial(`,
		`hsm.Target("closed")`,
		`hsm.State(`,
		`"closed"`,
		`hsm.Transition(`,
		`hsm.OnCall("approve")`,
		`hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex JavaScript operation adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) {
		t.Fatalf("codex JavaScript operation adapter CLI did not replace source operation body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesTypeScriptOperationBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.ts.out")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"operation_1","body":"record(instance, \"approve\");","imports":[{"path":"./approvals","specifiers":[{"name":"record"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "typescript", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import * as hsm from "@stateforward/hsm.ts";`,
		`import { record } from "./approvals";`,
		`function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void`,
		`record(instance, "approve");`,
		`export const DoorModel = hsm.Define(`,
		`hsm.Operation("approve", operation1)`,
		`hsm.Initial(`,
		`hsm.Target("closed")`,
		`hsm.State(`,
		`"closed"`,
		`hsm.Transition(`,
		`hsm.OnCall("approve")`,
		`hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex TypeScript operation adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) {
		t.Fatalf("codex TypeScript operation adapter CLI did not replace source operation body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesOperationBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"operation_1","body":"instance.approved = approval_value(\"approve\")","imports":[{"path":"approvals","specifiers":[{"name":"approval_value"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		"from approvals import approval_value",
		"async def operation_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:",
		`instance.approved = approval_value("approve")`,
		`door_model = hsm.Define(`,
		`hsm.Operation("approve", operation_1)`,
		`hsm.Initial(`,
		`hsm.Target("closed")`,
		`hsm.State(`,
		`"closed"`,
		`hsm.Transition(`,
		`hsm.OnCall("approve")`,
		`hsm.Target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex operation adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) ||
		strings.Contains(text, "Original typescript global") ||
		strings.Contains(text, "function operation1(") {
		t.Fatalf("codex operation adapter CLI did not replace source operation body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesGlobalCodeAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function record(value: string): string {
  return value.toUpperCase();
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"globals":[{"id":"global_1","code":"def record(value: str) -> str:\n\treturn normalize(value)","imports":[{"path":"helpers","specifiers":[{"name":"normalize"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		"from helpers import normalize",
		"def record(value: str) -> str:",
		"return normalize(value)",
		`door_model = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("closed")`,
		`hsm.State(`,
		`"closed"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript global global_1 preserved for manual porting") ||
		strings.Contains(text, "function record(value: string): string") {
		t.Fatalf("codex adapter CLI did not replace source global cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterRejectsTranslatedGlobalGeneratedSymbolCollisions(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function record(value: string): string {
  return value.toUpperCase();
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = record("closed");
  })),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"globals":[{"id":"global_1","code":"def helper(value: str) -> str:\n\treturn value\ndef entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:\n\tpass"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc codex unexpectedly accepted translated global compiler-owned symbol collision:\n%s", data)
	}
	want := `target-language global "global_1" declares "entry_1", which collides with compiler-owned python symbol`
	if !strings.Contains(string(data), want) {
		t.Fatalf("go run hsmc codex error missing translated global symbol collision rejection %q:\n%s", want, data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("codex translated global symbol collision rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected codex translated global symbol collision output: %v", readErr)
	}
}

func TestMainCodexAdapterRejectsHelperGlobalGeneratedFallbackCollision(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.hsm.json")
	output := filepath.Join(dir, "timer.py")
	codex := filepath.Join(dir, "codex.sh")
	source := `{
  "source_language": "typescript",
  "models": [{
    "id": "model_1",
    "name": "Timer",
    "initial": {"id": "initial_1", "owner_id": "model_1", "target": "idle"},
    "states": [
      {
        "id": "state_1",
        "name": "idle",
        "kind": "state",
        "transitions": [{
          "id": "transition_1",
          "owner_id": "state_1",
          "target": "../done",
          "trigger": {"kind": "after", "value": "Date.now() + 1000", "language": "typescript"}
        }]
      },
      {"id": "state_2", "name": "done", "kind": "state"}
    ]
  }]
}`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"globals":[{"id":"adapter_global_collision","code":"def hsmc_trigger_fallback_1(ctx, instance, event):\n\treturn 1.0"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "json-ir", "-to", "python", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc codex unexpectedly accepted helper global fallback collision:\n%s", data)
	}
	want := `adapter-created global "adapter_global_collision" declares "hsmc_trigger_fallback_1", which collides with compiler-owned python symbol`
	if !strings.Contains(string(data), want) {
		t.Fatalf("go run hsmc codex error missing helper global fallback collision rejection %q:\n%s", want, data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("codex helper global fallback collision rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected codex helper global fallback collision output: %v", readErr)
	}
}

func TestMainCodexAdapterCreatesHelperGlobalWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"globals":[{"id":"adapter_global_format","code":"def adapter_format(value: str) -> str:\n\treturn value.upper()"}],"behaviors":[{"id":"entry_1","body":"instance.log = adapter_format(\"closed\")"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		"def adapter_format(value: str) -> str:",
		"return value.upper()",
		"async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:",
		`instance.log = adapter_format("closed")`,
		`door_model = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("closed")`,
		`hsm.State(`,
		`"closed"`,
		`hsm.Entry(entry_1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex adapter helper-global CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("codex adapter helper-global CLI did not replace source behavior cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesDartBodiesAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.dart")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx, instance, event) => {
      instance.log = "closed";
    }),
    hsm.Transition(
      hsm.On("open"),
      hsm.Target("../open"),
    ),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"globals":[{"id":"adapter_global_format","code":"String adapterFormat(String value) => adapterGlobalFormat(value);","imports":[{"path":"global_format.dart","specifiers":[{"name":"adapterGlobalFormat"}]}]}],"behaviors":[{"id":"entry_1","body":"adapterBehaviorFormat(adapterFormat(\"closed\"));","imports":[{"path":"behavior_format.dart","specifiers":[{"name":"adapterBehaviorFormat"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "dart", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import "global_format.dart" show adapterGlobalFormat;`,
		`import "behavior_format.dart" show adapterBehaviorFormat;`,
		`String adapterFormat(String value) => adapterGlobalFormat(value);`,
		"FutureOr<void> entry1(Context ctx, Instance instance, Event event)",
		`adapterBehaviorFormat(adapterFormat("closed"));`,
		`final doorModel = define(`,
		`initial(target("closed"))`,
		`state(`,
		`entry(entry1)`,
		`transition([`,
		`on("open")`,
		`target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Dart adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `instance.log = "closed";`) {
		t.Fatalf("codex Dart adapter CLI did not replace source behavior body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesDartOperationBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.dart")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"operation_1","body":"adapterApprove(instance, \"approve\");","imports":[{"path":"approvals.dart","specifiers":[{"name":"adapterApprove"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "dart", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import "approvals.dart" show adapterApprove;`,
		"FutureOr<void> operation1(Context ctx, Instance instance, Event event)",
		`adapterApprove(instance, "approve");`,
		`final doorModel = define(`,
		`operation("approve", operation1)`,
		`initial(target("closed"))`,
		`state(`,
		`transition([`,
		`onCall("approve")`,
		`target("../open")`,
		`"open"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Dart operation adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) ||
		strings.Contains(text, "Original typescript global") ||
		strings.Contains(text, "function operation1(") {
		t.Fatalf("codex Dart operation adapter CLI did not replace source operation body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesDartWhenTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.dart")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.When((ctx, instance, event) => {
        return instance.ready;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return ready(instance);","imports":[{"path":"signals.dart","specifiers":[{"name":"ready"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "dart", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import "signals.dart" show ready;`,
		`bool trigger1(Context ctx, Instance instance, Event event)`,
		`return ready(instance);`,
		`final timerModel = define(`,
		`initial(target("idle"))`,
		`state(`,
		`"idle"`,
		`transition([`,
		`when(trigger1)`,
		`target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Dart when trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return instance.ready;") {
		t.Fatalf("codex Dart when trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesDartAtTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.dart")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return deadlineNow();","imports":[{"path":"clock.dart","specifiers":[{"name":"deadlineNow"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "dart", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import "dart:async";`,
		`import "clock.dart" show deadlineNow;`,
		`FutureOr<DateTime> trigger1(Context ctx, Instance instance, Event event)`,
		`return deadlineNow();`,
		`final timerModel = define(`,
		`initial(target("idle"))`,
		`state(`,
		`transition([`,
		`at(trigger1)`,
		`target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Dart at trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return new Date();") ||
		strings.Contains(text, "Unsupported at trigger") ||
		strings.Contains(text, "_hsmcAtTransition") {
		t.Fatalf("codex Dart at trigger adapter CLI did not preserve the target trigger boundary cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesRustOperationBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "lib.rs")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"operation_1","body":"let _ = approval_value(\"approve\");\nBox::pin(async {})","imports":[{"path":"crate::approvals","specifiers":[{"name":"approval_value"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "rust", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`use crate::approvals::{approval_value};`,
		`fn operation_1(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) -> Pin<Box<dyn Future<Output = ()> + Send>>`,
		`let _ = approval_value("approve");`,
		`Box::pin(async {})`,
		`pub fn door_model() -> hsm::Model<HsmcInstance>`,
		`hsm::operation("approve", operation_1)`,
		`hsm::initial!(hsm::target!("closed"))`,
		`hsm::state!("closed"`,
		`hsm::transition!(hsm::on_call("approve"), hsm::target!("../open"))`,
		`hsm::state!("open")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Rust operation adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior operation_1 preserved for manual porting") ||
		strings.Contains(text, `instance.set("approved", true);`) ||
		strings.Contains(text, "Original typescript global") ||
		strings.Contains(text, "function operation1(") ||
		strings.Contains(text, "hsm.rs does not expose an operation partial") ||
		strings.Contains(text, "on_call trigger lowered to event trigger") {
		t.Fatalf("codex Rust operation adapter CLI did not replace source operation body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesRustAtTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "lib.rs")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"deadline()","imports":[{"path":"crate::clock","specifiers":[{"name":"deadline"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "rust", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`use crate::clock::{deadline};`,
		`fn trigger_1(ctx: &hsm::Context, instance: &HsmcInstance, event: &hsm::Event) -> std::time::SystemTime`,
		`deadline()`,
		`pub fn timer_model() -> hsm::Model<HsmcInstance>`,
		`hsm::initial!(hsm::target!("idle"))`,
		`hsm::state!("idle"`,
		`hsm::transition!(hsm::at!(trigger_1), hsm::target!("../done"))`,
		`hsm::state!("done")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Rust at trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{
		"Original typescript behavior trigger_1 preserved for manual porting",
		"return new Date();",
		"at trigger lowered to after trigger",
		"hsm::after!(trigger_1)",
		"hsm::after!(hsmc_zero_duration)",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("codex Rust at trigger adapter CLI output contains forbidden %q:\n%s", forbidden, text)
		}
	}
}

func TestMainCodexAdapterAppliesRustWhenTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "lib.rs")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.When((ctx, instance, event) => {
        return instance.ready;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"ready(instance)","imports":[{"path":"crate::signals","specifiers":[{"name":"ready"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "rust", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex Rust when trigger adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`use crate::signals::{ready};`,
		`fn trigger_1(ctx: &hsm::Context, instance: &HsmcInstance, event: &hsm::Event) -> bool`,
		`ready(instance)`,
		`pub fn timer_model() -> hsm::Model<HsmcInstance>`,
		`hsm::define!("Timer"`,
		`hsm::initial!(hsm::target!("idle"))`,
		`hsm::state!("idle"`,
		`/* when trigger lowered to guard for hsm.rs */ hsm::guard!(trigger_1)`,
		`hsm::target!("../done")`,
		`hsm::state!("done")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Rust when trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return instance.ready;") ||
		strings.Contains(text, "hsm::when") ||
		strings.Contains(text, "Unsupported when trigger") {
		t.Fatalf("codex Rust when trigger adapter CLI did not preserve the target trigger boundary cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterRejectsRustOperationFullCodeSignatureChange(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "lib.rs")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"operation_1","code":"fn operation_1(ctx: &hsm::Context, instance: &HsmcInstance, event: &hsm::Event) -> Pin<Box<dyn Future<Output = ()> + Send>> { Box::pin(async {}) }"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "rust", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc codex unexpectedly accepted Rust operation full-code signature change:\n%s", data)
	}
	if !strings.Contains(string(data), `declares parameters "(ctx: &hsm::Context, instance: &HsmcInstance, event: &hsm::Event)", want compiler-owned parameters "(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event)"`) {
		t.Fatalf("go run hsmc codex error missing Rust operation signature rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("codex Rust operation full-code signature rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected codex output: %v", readErr)
	}
}

func TestMainCodexAdapterAppliesTriggerBodyWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.go")
	output := filepath.Join(dir, "timer.ts")
	codex := filepath.Join(dir, "codex.sh")
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
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return delay(250);","imports":[{"path":"./timers","specifiers":[{"name":"delay"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "go", "-to", "typescript", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import { delay } from "./timers";`,
		"function trigger1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): number | Date | Promise<number> | Promise<Date> {",
		"return delay(250);",
		"hsm.After(trigger1)",
		`export const TimerModel = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("idle")`,
		`hsm.State(`,
		`"idle"`,
		`hsm.Transition(`,
		`hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original go behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return time.Second") {
		t.Fatalf("codex trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesPythonWhenTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.py")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.When((ctx, instance, event) => {
        return instance.ready;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return ready(instance)","imports":[{"path":"signals","specifiers":[{"name":"ready"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import datetime`,
		`import hsm`,
		`import typing`,
		`from signals import ready`,
		`async def trigger_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> typing.Any:`,
		`return ready(instance)`,
		`timer_model = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("idle")`,
		`hsm.State(`,
		`"idle"`,
		`hsm.Transition(`,
		`hsm.When(trigger_1)`,
		`hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Python when trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return instance.ready;") {
		t.Fatalf("codex Python when trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesPythonAtTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.py")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return deadline_now()","imports":[{"path":"clock","specifiers":[{"name":"deadline_now"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import datetime`,
		`import hsm`,
		`from clock import deadline_now`,
		`async def trigger_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> datetime.datetime:`,
		`return deadline_now()`,
		`timer_model = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("idle")`,
		`hsm.State(`,
		`"idle"`,
		`hsm.Transition(`,
		`hsm.At(trigger_1)`,
		`hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Python at trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return new Date();") {
		t.Fatalf("codex Python at trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesJavaScriptWhenTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.js")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.When((ctx, instance, event) => {
        return instance.ready;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return ready(instance);","imports":[{"path":"./signals","specifiers":[{"name":"ready"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "javascript", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import * as hsm from "@stateforward/hsm";`,
		`import { ready } from "./signals";`,
		`function trigger1(ctx, instance, event)`,
		`return ready(instance);`,
		`export const TimerModel = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("idle")`,
		`hsm.State(`,
		`"idle"`,
		`hsm.Transition(`,
		`hsm.When(trigger1)`,
		`hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex JavaScript when trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return instance.ready;") {
		t.Fatalf("codex JavaScript when trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesJavaScriptAtTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.js")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return deadlineNow();","imports":[{"path":"./clock","specifiers":[{"name":"deadlineNow"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "javascript", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import * as hsm from "@stateforward/hsm";`,
		`import { deadlineNow } from "./clock";`,
		`function trigger1(ctx, instance, event)`,
		`return deadlineNow();`,
		`export const TimerModel = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("idle")`,
		`hsm.State(`,
		`"idle"`,
		`hsm.Transition(`,
		`hsm.At(trigger1)`,
		`hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex JavaScript at trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return new Date();") {
		t.Fatalf("codex JavaScript at trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesTypeScriptWhenTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.ts.out")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.When((ctx, instance, event) => {
        return instance.ready;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return ready(instance);","imports":[{"path":"./signals","specifiers":[{"name":"ready"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "typescript", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import * as hsm from "@stateforward/hsm.ts";`,
		`import { ready } from "./signals";`,
		`function trigger1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): unknown`,
		`return ready(instance);`,
		`export const TimerModel = (hsm.Define as any)(`,
		`hsm.Initial(`,
		`hsm.Target("idle")`,
		`hsm.State(`,
		`"idle"`,
		`hsm.Transition(`,
		`(hsm.When as any)(trigger1)`,
		`) as any,`,
		`) as any,`,
		`hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex TypeScript when trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return instance.ready;") {
		t.Fatalf("codex TypeScript when trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesTypeScriptAtTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.ts.out")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return deadlineNow();","imports":[{"path":"./clock","specifiers":[{"name":"deadlineNow"}]}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "typescript", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import * as hsm from "@stateforward/hsm.ts";`,
		`import { deadlineNow } from "./clock";`,
		`function trigger1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): Date | Promise<Date>`,
		`return deadlineNow();`,
		`export const TimerModel = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("idle")`,
		`hsm.State(`,
		`"idle"`,
		`hsm.Transition(`,
		`hsm.At(trigger1)`,
		`hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex TypeScript at trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return new Date();") {
		t.Fatalf("codex TypeScript at trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesZigWhenTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.zig")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.When((ctx, instance, event) => {
        return instance.ready;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return signals.ready(inst);","imports":[{"path":"./signals.zig","alias":"signals"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "zig", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`const hsm = @import("hsm");`,
		`const signals = @import("./signals.zig");`,
		`fn trigger_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) bool`,
		`return signals.ready(inst);`,
		`pub const timer_model = hsm.define("Timer", .{`,
		`hsm.initial(hsm.target("idle"))`,
		`hsm.state("idle", .{`,
		`hsm.transition(.{`,
		`// when trigger lowered to guard for hsm.zig`,
		`hsm.guard(trigger_1),`,
		`hsm.target("../done")`,
		`hsm.state("done", .{`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Zig when trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return instance.ready;") ||
		strings.Contains(text, "Unsupported when trigger") ||
		strings.Contains(text, "hsm.when") {
		t.Fatalf("codex Zig when trigger adapter CLI did not preserve the intended boundary:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesZigAtTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.zig")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return clock.deadline();","imports":[{"path":"clock","alias":"clock"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "zig", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`const hsm = @import("hsm");`,
		`const clock = @import("clock");`,
		`fn trigger_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) u64`,
		`return clock.deadline();`,
		`pub const timer_model = hsm.define("Timer", .{`,
		`hsm.initial(hsm.target("idle"))`,
		`hsm.state("idle", .{`,
		`hsm.transition(.{`,
		`hsm.at(trigger_1),`,
		`hsm.target("../done")`,
		`hsm.state("done", .{`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Zig at trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{
		"Original typescript behavior trigger_1 preserved for manual porting",
		"return new Date();",
		"Unsupported at trigger",
		"at trigger lowered to after trigger",
		"hsm.after(trigger_1)",
		"hsm.after(hsmc_trigger_zero)",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("codex Zig at trigger adapter CLI output contains forbidden %q:\n%s", forbidden, text)
		}
	}
}

func TestMainCodexAdapterAppliesGoTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.go")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.After((ctx, instance, event) => {
        return 1000;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return timers.Delay(250) * time.Millisecond","imports":[{"path":"example.com/timers","alias":"timers"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "go", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`"context"`,
		`timers "example.com/timers"`,
		`hsm "github.com/stateforward/hsm.go"`,
		`"time"`,
		`func Trigger1(ctx context.Context, instance hsm.Instance, event hsm.Event) time.Duration`,
		`return timers.Delay(250) * time.Millisecond`,
		`var TimerModel = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("idle")`,
		`hsm.State(`,
		`"idle"`,
		`hsm.Transition(`,
		`hsm.After(Trigger1)`,
		`hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Go trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return 1000;") {
		t.Fatalf("codex Go trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesGoAtTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "timer.go")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return deadline.Now()","imports":[{"path":"example.com/deadline"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "go", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex Go at trigger adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`"context"`,
		`"example.com/deadline"`,
		`hsm "github.com/stateforward/hsm.go"`,
		`"time"`,
		`func Trigger1(ctx context.Context, instance hsm.Instance, event hsm.Event) time.Time`,
		`return deadline.Now()`,
		`var TimerModel = hsm.Define(`,
		`hsm.Initial(`,
		`hsm.Target("idle")`,
		`hsm.State(`,
		`"idle"`,
		`hsm.Transition(`,
		`hsm.At(Trigger1)`,
		`hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Go at trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return new Date();") {
		t.Fatalf("codex Go at trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesCSharpWhenTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "GeneratedHsm.cs")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.When((ctx, instance, event) => {
        return instance.ready;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return Ready.WaitAsync(cancellationToken);","imports":[{"path":"Sample.Ready.Helper","alias":"Ready"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "csharp", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`using Stateforward.Hsm;`,
		`using Ready = Sample.Ready.Helper;`,
		`private static System.Threading.Tasks.Task Trigger1(Context ctx, Instance instance, Event @event, System.Threading.CancellationToken cancellationToken)`,
		`return Ready.WaitAsync(cancellationToken);`,
		`public static readonly Model TimerModel = Hsm.Define(`,
		`Hsm.Initial(`,
		`Hsm.Target("idle")`,
		`Hsm.State(`,
		`"idle"`,
		`Hsm.Transition(`,
		`Hsm.When<Instance>(Trigger1)`,
		`Hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex C# when trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return instance.ready;") {
		t.Fatalf("codex C# when trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesCSharpAtTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "GeneratedHsm.cs")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return Clock.Deadline();","imports":[{"path":"Sample.Clock","alias":"Clock"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "csharp", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`using Stateforward.Hsm;`,
		`using Clock = Sample.Clock;`,
		`private static System.DateTimeOffset Trigger1(Context ctx, Instance instance, Event @event)`,
		`return Clock.Deadline();`,
		`public static readonly Model TimerModel = Hsm.Define(`,
		`Hsm.Initial(`,
		`Hsm.Target("idle")`,
		`Hsm.State(`,
		`"idle"`,
		`Hsm.Transition(`,
		`Hsm.At<Instance>(Trigger1)`,
		`Hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex C# at trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return new Date();") {
		t.Fatalf("codex C# at trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesJavaWhenTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "GeneratedHsm.java")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.When((ctx, instance, event) => {
        return instance.ready;
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return Ready.isReady(instance);","imports":[{"path":"com.example.Ready"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "java", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import com.stateforward.hsm.*;`,
		`import com.example.Ready;`,
		`private static boolean Trigger1(Context ctx, Instance instance, Event event)`,
		`return Ready.isReady(instance);`,
		`public static final Model TimerModel = Hsm.Define(`,
		`Hsm.Initial(Hsm.Target("idle"))`,
		`Hsm.State(`,
		`"idle"`,
		`Hsm.Transition(`,
		`Hsm.When(GeneratedHsm::Trigger1)`,
		`Hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Java when trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return instance.ready;") {
		t.Fatalf("codex Java when trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterAppliesJavaAtTriggerBodyAndImportsWithoutChangingModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.ts")
	output := filepath.Join(dir, "GeneratedHsm.java")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const TimerModel = hsm.Define(
  "Timer",
  hsm.Initial(hsm.Target("idle")),
  hsm.State(
    "idle",
    hsm.Transition(
      hsm.At((ctx, instance, event) => {
        return new Date();
      }),
      hsm.Target("../done"),
    ),
  ),
  hsm.State("done"),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"trigger_1","body":"return Clock.deadline();","imports":[{"path":"com.example.Clock"}]}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "java", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc codex adapter failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		`import com.stateforward.hsm.*;`,
		`import com.example.Clock;`,
		`private static java.time.Instant Trigger1(Context ctx, Instance instance, Event event)`,
		`return Clock.deadline();`,
		`public static final Model TimerModel = Hsm.Define(`,
		`Hsm.Initial(Hsm.Target("idle"))`,
		`Hsm.State(`,
		`"idle"`,
		`Hsm.Transition(`,
		`Hsm.At(GeneratedHsm::Trigger1)`,
		`Hsm.Target("../done")`,
		`"done"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("codex Java at trigger adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior trigger_1 preserved for manual porting") ||
		strings.Contains(text, "return new Date();") {
		t.Fatalf("codex Java at trigger adapter CLI did not replace source trigger body cleanly:\n%s", text)
	}
}

func TestMainCodexAdapterRejectsBehaviorFullCodeExtraTopLevelDeclarations(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"entry_1","code":"async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:\n\tinstance.log = [\"closed\"]\ndef door_model():\n\treturn None"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc codex unexpectedly accepted behavior full-code extra top-level declaration:\n%s", data)
	}
	if !strings.Contains(string(data), "contains extra top-level code after the compiler-owned callable") {
		t.Fatalf("go run hsmc codex error missing extra top-level behavior code rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("codex behavior full-code extra top-level rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected codex output: %v", readErr)
	}
}

func TestMainCodexAdapterRejectsPythonExpressionBehaviorFullCodeExtraTopLevelDeclarations(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"entry_1","code":"lambda ctx, instance, event: setattr(instance, \"log\", \"closed\"); door_model = None"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc codex unexpectedly accepted Python expression behavior full-code extra top-level declaration:\n%s", data)
	}
	if !strings.Contains(string(data), "contains extra top-level code after the compiler-owned callable") {
		t.Fatalf("go run hsmc codex error missing Python expression extra top-level behavior code rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("codex Python expression behavior full-code extra top-level rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected codex output: %v", readErr)
	}
}

func TestMainCodexAdapterRejectsTypeScriptExpressionBehaviorFullCodeExtraTopLevelDeclarations(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.go")
	output := filepath.Join(dir, "door.ts")
	codex := filepath.Join(dir, "codex.sh")
	source := `package door

import hsm "github.com/stateforward/hsm"

var DoorModel = hsm.Define("Door",
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed", hsm.Entry(func(ctx hsm.Context, instance hsm.Instance, event hsm.Event) {
		instance.Set("log", "closed")
	})),
)
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"entry_1","code":"(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void => instance.ready = true\nconst DoorModel = null"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "go", "-to", "typescript", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc codex unexpectedly accepted TypeScript expression behavior full-code extra top-level declaration:\n%s", data)
	}
	if !strings.Contains(string(data), "contains extra top-level code after the compiler-owned callable") {
		t.Fatalf("go run hsmc codex error missing TypeScript expression extra top-level behavior code rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("codex TypeScript expression behavior full-code extra top-level rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected codex output: %v", readErr)
	}
}

func TestMainCodexAdapterRejectsDartArrowDeclarationBehaviorFullCodeSignatureChange(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.dart")
	codex := filepath.Join(dir, "codex.sh")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
{"behaviors":[{"id":"entry_1","code":"FutureOr<void> entry1(Context ctx) => adapterFormat(\"entry\");"}]}
JSON
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "dart", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc codex unexpectedly accepted Dart arrow declaration behavior full-code signature change:\n%s", data)
	}
	if !strings.Contains(string(data), `declares parameters "(Context ctx)", want compiler-owned parameters "(Context ctx, Instance instance, Event event)"`) {
		t.Fatalf("go run hsmc codex error missing Dart arrow declaration signature rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("codex Dart arrow declaration behavior full-code signature rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected codex output: %v", readErr)
	}
}

func TestMainCodexAdapterRejectsUnknownEditableIDs(t *testing.T) {
	cases := []struct {
		name  string
		patch string
		want  string
	}{
		{
			name:  "unknown behavior",
			patch: `{"behaviors":[{"id":"entry_missing","body":"pass"}]}`,
			want:  `adapter patch references unknown behavior "entry_missing"`,
		},
		{
			name:  "unknown global",
			patch: `{"globals":[{"id":"global_missing","code":"def helper() -> None:\n\tpass"}]}`,
			want:  `adapter patch references unknown global "global_missing"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "door.ts")
			output := filepath.Join(dir, "door.py")
			codex := filepath.Join(dir, "codex.sh")
			source := `import * as hsm from "@stateforward/hsm.ts";

function record(value: string): string {
  return value;
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = record("closed");
  })),
);
`
			script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
` + tc.patch + `
JSON
`
			if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
			command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
			data, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("go run hsmc codex unexpectedly accepted unknown editable ID patch %s:\n%s", tc.name, data)
			}
			if !strings.Contains(string(data), tc.want) {
				t.Fatalf("go run hsmc codex error missing unknown editable ID rejection %q:\n%s", tc.want, data)
			}
			if generated, readErr := os.ReadFile(output); readErr == nil {
				t.Fatalf("codex unknown editable ID rejection should not write output, got:\n%s", generated)
			}
		})
	}
}

func TestMainCodexAdapterRejectsStructuralTopLevelEdits(t *testing.T) {
	cases := []struct {
		name  string
		field string
		patch string
	}{
		{
			name:  "models",
			field: "models",
			patch: `{"models":[{"id":"model_1","name":"Changed"}]}`,
		},
		{
			name:  "events",
			field: "events",
			patch: `{"events":[{"id":"event_1","name":"changed"}]}`,
		},
		{
			name:  "attributes",
			field: "attributes",
			patch: `{"attributes":[{"id":"attribute_1","name":"changed"}]}`,
		},
		{
			name:  "operations",
			field: "operations",
			patch: `{"operations":[{"id":"operation_1","name":"changed"}]}`,
		},
		{
			name:  "states",
			field: "states",
			patch: `{"states":[{"id":"state_1","name":"changed"}]}`,
		},
		{
			name:  "initial",
			field: "initial",
			patch: `{"initial":{"target":"changed"}}`,
		},
		{
			name:  "transitions",
			field: "transitions",
			patch: `{"transitions":[{"id":"transition_1","target":"changed"}]}`,
		},
		{
			name:  "triggers",
			field: "triggers",
			patch: `{"triggers":[{"kind":"on","value":"changed"}]}`,
		},
		{
			name:  "defers",
			field: "defers",
			patch: `{"defers":["changed"]}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "door.ts")
			output := filepath.Join(dir, "door.py")
			codex := filepath.Join(dir, "codex.sh")
			source := `import * as hsm from "@stateforward/hsm.ts";

export const opened = hsm.Event("opened");

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Transition(hsm.On(opened), hsm.Target("../open"))),
  hsm.State("open"),
);
`
			script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
` + tc.patch + `
JSON
`
			if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
			command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
			data, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("go run hsmc codex unexpectedly accepted adapter %s edits:\n%s", tc.field, data)
			}
			want := `unsupported top-level field "` + tc.field + `"`
			if !strings.Contains(string(data), want) {
				t.Fatalf("go run hsmc codex error missing %s rejection %q:\n%s", tc.field, want, data)
			}
			if generated, readErr := os.ReadFile(output); readErr == nil {
				t.Fatalf("codex adapter %s rejection should not write output, got:\n%s", tc.field, generated)
			}
		})
	}
}

func TestMainCodexAdapterRejectsCompilerOwnedMetadataFields(t *testing.T) {
	cases := []struct {
		name  string
		patch string
		want  string
	}{
		{
			name:  "behavior owner",
			patch: `{"behaviors":[{"id":"entry_1","owner_id":"state_2","body":"pass"}]}`,
			want:  `behaviors[0] contains unsupported field "owner_id"`,
		},
		{
			name:  "behavior kind",
			patch: `{"behaviors":[{"id":"entry_1","kind":"guard","body":"return True"}]}`,
			want:  `behaviors[0] contains unsupported field "kind"`,
		},
		{
			name:  "behavior trigger kind",
			patch: `{"behaviors":[{"id":"entry_1","trigger_kind":"when","body":"return True"}]}`,
			want:  `behaviors[0] contains unsupported field "trigger_kind"`,
		},
		{
			name:  "behavior source language",
			patch: `{"behaviors":[{"id":"entry_1","source_language":"python","body":"pass"}]}`,
			want:  `behaviors[0] contains unsupported field "source_language"`,
		},
		{
			name:  "behavior inline",
			patch: `{"behaviors":[{"id":"entry_1","inline":true,"body":"pass"}]}`,
			want:  `behaviors[0] contains unsupported field "inline"`,
		},
		{
			name:  "behavior target language",
			patch: `{"behaviors":[{"id":"entry_1","target_language":"python","body":"pass"}]}`,
			want:  `behaviors[0] contains unsupported field "target_language"`,
		},
		{
			name:  "behavior signature",
			patch: `{"behaviors":[{"id":"entry_1","signature":"def changed(ctx):","body":"pass"}]}`,
			want:  `behaviors[0] contains unsupported field "signature"`,
		},
		{
			name:  "behavior abi",
			patch: `{"behaviors":[{"id":"entry_1","target_abi":{"language":"python","signature":"def changed(ctx):"},"body":"pass"}]}`,
			want:  `behaviors[0] contains unsupported field "target_abi"`,
		},
		{
			name:  "global language",
			patch: `{"globals":[{"id":"global_1","language":"python","code":"def record(value: str) -> str:\n\treturn value"}]}`,
			want:  `globals[0] contains unsupported field "language"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "door.ts")
			output := filepath.Join(dir, "door.py")
			codex := filepath.Join(dir, "codex.sh")
			source := `import * as hsm from "@stateforward/hsm.ts";

function record(value: string): string {
  return value;
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = record("closed");
  })),
);
`
			script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
` + tc.patch + `
JSON
`
			if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
			command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
			data, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("go run hsmc codex unexpectedly accepted compiler-owned metadata field %s:\n%s", tc.name, data)
			}
			if !strings.Contains(string(data), tc.want) {
				t.Fatalf("go run hsmc codex error missing compiler-owned metadata rejection %q:\n%s", tc.want, data)
			}
			if generated, readErr := os.ReadFile(output); readErr == nil {
				t.Fatalf("codex compiler-owned metadata rejection should not write output, got:\n%s", generated)
			}
		})
	}
}

func TestMainCodexAdapterRejectsMalformedPatchFields(t *testing.T) {
	cases := []struct {
		name  string
		patch string
		want  string
	}{
		{
			name:  "top-level imports null",
			patch: `{"imports":null}`,
			want:  `field "imports" must be an array, got null`,
		},
		{
			name:  "behavior body number",
			patch: `{"behaviors":[{"id":"entry_1","body":42}]}`,
			want:  `behaviors[0].body must be a string`,
		},
		{
			name:  "behavior whitespace-only body",
			patch: `{"behaviors":[{"id":"entry_1","body":" \t\n"}]}`,
			want:  `behaviors[0].body contains whitespace-only code`,
		},
		{
			name:  "global whitespace-only code",
			patch: `{"globals":[{"id":"global_1","code":" \t\n"}]}`,
			want:  `globals[0].code contains whitespace-only code`,
		},
		{
			name:  "behavior whitespace-only code",
			patch: `{"behaviors":[{"id":"entry_1","code":" \t\n"}]}`,
			want:  `behaviors[0].code contains whitespace-only code`,
		},
		{
			name:  "behavior body and code",
			patch: `{"behaviors":[{"id":"entry_1","body":"instance.log = 'closed'","code":"def entry_1(ctx, instance, event):\n\tinstance.log = 'closed'"}]}`,
			want:  `behaviors[0] must set either body or code, not both`,
		},
		{
			name:  "behavior missing id",
			patch: `{"behaviors":[{"body":"pass"}]}`,
			want:  `behaviors[0].id is required`,
		},
		{
			name:  "global padded id",
			patch: `{"globals":[{"id":" global_1 ","code":"def record(value: str) -> str:\n\treturn value"}]}`,
			want:  `globals[0].id " global_1 " has leading or trailing whitespace`,
		},
		{
			name:  "behavior padded id",
			patch: `{"behaviors":[{"id":" entry_1 ","body":"instance.log = 'closed'"}]}`,
			want:  `behaviors[0].id " entry_1 " has leading or trailing whitespace`,
		},
		{
			name:  "program import padded path",
			patch: `{"imports":[{"path":" helpers "}]}`,
			want:  `imports[0].path " helpers " has leading or trailing whitespace`,
		},
		{
			name:  "duplicate top-level field",
			patch: `{"behaviors":[],"behaviors":[{"id":"entry_1","body":"translated()"}]}`,
			want:  `adapter patch contains duplicate field "behaviors"`,
		},
		{
			name:  "duplicate behavior field",
			patch: `{"behaviors":[{"id":"entry_1","body":"one()","body":"two()"}]}`,
			want:  `adapter patch behaviors[0] contains duplicate field "body"`,
		},
		{
			name:  "duplicate nested import field",
			patch: `{"globals":[{"id":"global_1","code":"def record(value: str) -> str:\n\treturn value","imports":[{"path":"one","path":"two"}]}]}`,
			want:  `adapter patch globals[0].imports[0] contains duplicate field "path"`,
		},
		{
			name:  "duplicate global id",
			patch: `{"globals":[{"id":"global_1","code":"def one() -> None:\n\tpass"},{"id":"global_1","code":"def two() -> None:\n\tpass"}]}`,
			want:  `adapter patch contains duplicate global "global_1"`,
		},
		{
			name:  "duplicate behavior id",
			patch: `{"behaviors":[{"id":"entry_1","body":"one()"},{"id":"entry_1","body":"two()"}]}`,
			want:  `adapter patch contains duplicate behavior "entry_1"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "door.ts")
			output := filepath.Join(dir, "door.py")
			codex := filepath.Join(dir, "codex.sh")
			source := `import * as hsm from "@stateforward/hsm.ts";

function record(value: string): string {
  return value;
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = record("closed");
  })),
);
`
			script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
` + tc.patch + `
JSON
`
			if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
			command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
			data, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("go run hsmc codex unexpectedly accepted malformed adapter patch %s:\n%s", tc.name, data)
			}
			if !strings.Contains(string(data), tc.want) {
				t.Fatalf("go run hsmc codex error missing malformed patch rejection %q:\n%s", tc.want, data)
			}
			if generated, readErr := os.ReadFile(output); readErr == nil {
				t.Fatalf("codex malformed patch rejection should not write output, got:\n%s", generated)
			}
		})
	}
}

func TestMainCodexAdapterRejectsImportBoundaryViolations(t *testing.T) {
	cases := []struct {
		name  string
		patch string
		want  string
	}{
		{
			name:  "program compiler-owned runtime import",
			patch: `{"imports":[{"path":"datetime","alias":"dt"}]}`,
			want:  `program import "datetime" is compiler-owned`,
		},
		{
			name:  "helper global compiler-owned runtime import",
			patch: `{"globals":[{"id":"adapter_global_time","code":"def adapter_time() -> None:\n\treturn None","imports":[{"path":"datetime","alias":"dt"}]}]}`,
			want:  `global "adapter_global_time" import "datetime" is compiler-owned`,
		},
		{
			name:  "behavior compiler-owned runtime import",
			patch: `{"behaviors":[{"id":"entry_1","body":"instance.log = 'closed'","imports":[{"path":"datetime","alias":"dt"}]}]}`,
			want:  `behavior "entry_1" import "datetime" is compiler-owned`,
		},
		{
			name:  "program import wrong language",
			patch: `{"imports":[{"path":"helpers","language":"go"}]}`,
			want:  `adapter patch program import "helpers" has language "go", want target language "python"`,
		},
		{
			name:  "helper global import wrong language",
			patch: `{"globals":[{"id":"adapter_global_record","code":"def adapter_record(value: str) -> str:\n\treturn value","imports":[{"path":"helpers","language":"go"}]}]}`,
			want:  `adapter patch global "adapter_global_record" import "helpers" has language "go", want target language "python"`,
		},
		{
			name:  "behavior import wrong language",
			patch: `{"behaviors":[{"id":"entry_1","body":"instance.log = record('closed')","imports":[{"path":"helpers","language":"go"}]}]}`,
			want:  `adapter patch behavior "entry_1" import "helpers" has language "go", want target language "python"`,
		},
		{
			name:  "foreign global import only",
			patch: `{"globals":[{"id":"global_1","imports":[{"path":"helpers","specifiers":[{"name":"record"}]}]}]}`,
			want:  `adapter patch changes global "global_1" imports without target code`,
		},
		{
			name:  "foreign behavior import only",
			patch: `{"behaviors":[{"id":"entry_1","imports":[{"path":"helpers","specifiers":[{"name":"record"}]}]}]}`,
			want:  `adapter patch changes behavior "entry_1" imports without target body or code`,
		},
		{
			name:  "import local names",
			patch: `{"imports":[{"path":"helpers","local_names":["helpers"]}]}`,
			want:  `imports[0] contains unsupported field "local_names"`,
		},
		{
			name:  "specifier range",
			patch: `{"imports":[{"path":"helpers","specifiers":[{"name":"record","range":{"start_byte":1}}]}]}`,
			want:  `imports[0].specifiers[0] contains unsupported field "range"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "door.ts")
			output := filepath.Join(dir, "door.py")
			codex := filepath.Join(dir, "codex.sh")
			source := `import * as hsm from "@stateforward/hsm.ts";

function record(value: string): string {
  return value;
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = record("closed");
  })),
);
`
			script := `#!/bin/sh
if [ "$1" != "exec" ]; then
  echo "expected codex exec" >&2
  exit 2
fi
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    output=$1
  fi
  shift
done
cat >/dev/null
cat > "$output" <<'JSON'
` + tc.patch + `
JSON
`
			if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(codex, []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-adapter", "codex", "-adapter-command", codex)
			command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
			data, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("go run hsmc codex unexpectedly accepted import-boundary violation %s:\n%s", tc.name, data)
			}
			if !strings.Contains(string(data), tc.want) {
				t.Fatalf("go run hsmc codex error missing import-boundary rejection %q:\n%s", tc.want, data)
			}
			if generated, readErr := os.ReadFile(output); readErr == nil {
				t.Fatalf("codex import-boundary rejection should not write output, got:\n%s", generated)
			}
		})
	}
}

func TestMainInfersLanguagesFromInputAndOutputPaths(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-in", input, "-out", output)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		"import hsm",
		`door_model = hsm.Define(`,
		`hsm.Target("closed")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("inferred CLI output missing %q:\n%s", needle, text)
		}
	}
}

func TestMainReadsStdinAndWritesStdout(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", "-")
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	command.Stdin = strings.NewReader(source)
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc stdin/stdout failed: %v\n%s", err, data)
	}
	text := string(data)
	for _, needle := range []string{
		"import hsm",
		"async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:",
		"# Original typescript behavior entry_1 preserved for manual porting:",
		`# instance.log = "closed";`,
		`door_model = hsm.Define(`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("stdin/stdout CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "\n    instance.log = \"closed\";") {
		t.Fatalf("stdin/stdout CLI emitted TypeScript behavior as executable Python:\n%s", text)
	}
}

func TestMainDefaultNoAdapterPreservesForeignBehaviorAsComment(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	signature := "async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:"
	comment := "# Original typescript behavior entry_1 preserved for manual porting:"
	body := `# instance.log = "closed";`
	noop := "\n\tpass\n"
	model := `hsm.Entry(entry_1)`
	for _, needle := range []string{
		signature,
		comment,
		"# Target language: python",
		"# Behavior kind: entry",
		body,
		noop,
		model,
		`door_model = hsm.Define(`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("default no-adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	signatureIndex := strings.Index(text, signature)
	commentIndex := strings.Index(text, comment)
	bodyIndex := strings.Index(text, body)
	noopIndex := strings.Index(text, noop)
	modelIndex := strings.Index(text, model)
	if !(signatureIndex >= 0 && signatureIndex < commentIndex && commentIndex < bodyIndex && bodyIndex < noopIndex && noopIndex < modelIndex) {
		t.Fatalf("default no-adapter CLI entry comment was not inside the generated target behavior body:\n%s", text)
	}
	if strings.Contains(text, "\n    instance.log = \"closed\";") {
		t.Fatalf("default no-adapter CLI emitted TypeScript behavior as executable Python:\n%s", text)
	}
}

func TestMainDefaultNoAdapterPreservesForeignOperationBehaviorInsideTargetBody(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	source := `import * as hsm from "@stateforward/hsm.ts";

function operation1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
  instance.set("approved", true);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Operation("approve", operation1),
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	signature := "async def operation_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:"
	comment := "# Original typescript behavior operation_1 preserved for manual porting:"
	body := `# instance.set("approved", true);`
	noop := "\n\tpass\n"
	model := `hsm.Operation("approve", operation_1)`
	for _, needle := range []string{
		signature,
		comment,
		"# Target language: python",
		"# Behavior kind: operation",
		body,
		model,
		`hsm.Transition(`,
		`hsm.OnCall("approve")`,
		`hsm.Target("../open")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("default no-adapter CLI operation output missing %q:\n%s", needle, text)
		}
	}
	signatureIndex := strings.Index(text, signature)
	commentIndex := strings.Index(text, comment)
	bodyIndex := strings.Index(text, body)
	noopIndex := strings.Index(text, noop)
	modelIndex := strings.Index(text, model)
	if !(signatureIndex >= 0 && signatureIndex < commentIndex && commentIndex < bodyIndex && bodyIndex < noopIndex && noopIndex < modelIndex) {
		t.Fatalf("default no-adapter CLI operation comment was not inside the generated target operation body:\n%s", text)
	}
	if strings.Contains(text, "\n    instance.set(\"approved\", true);") ||
		strings.Contains(text, "\nfunction operation1(") {
		t.Fatalf("default no-adapter CLI emitted TypeScript operation as executable Python:\n%s", text)
	}
}

func TestMainDefaultNoAdapterPreservesForeignTriggerBehaviorAsComment(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.go")
	output := filepath.Join(dir, "timer.ts")
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
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "go", "-to", "typescript", "-in", input, "-out", output)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	signature := "function trigger1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): number | Date | Promise<number> | Promise<Date> {"
	comment := "// Original go behavior trigger_1 preserved for manual porting:"
	body := "// return time.Second"
	noop := "\n\treturn 0;\n}"
	model := "hsm.After(trigger1)"
	for _, needle := range []string{
		signature,
		comment,
		"// Target language: typescript",
		"// Behavior kind: trigger",
		body,
		noop,
		model,
		`export const TimerModel = hsm.Define(`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("default no-adapter CLI trigger output missing %q:\n%s", needle, text)
		}
	}
	signatureIndex := strings.Index(text, signature)
	commentIndex := strings.Index(text, comment)
	bodyIndex := strings.Index(text, body)
	noopIndex := strings.Index(text, noop)
	modelIndex := strings.Index(text, model)
	if !(signatureIndex >= 0 && signatureIndex < commentIndex && commentIndex < bodyIndex && bodyIndex < noopIndex && noopIndex < modelIndex) {
		t.Fatalf("default no-adapter CLI trigger comment was not inside the generated target behavior body:\n%s", text)
	}
	if strings.Contains(text, "\n\treturn time.Second\n") {
		t.Fatalf("default no-adapter CLI emitted Go trigger body as executable TypeScript:\n%s", text)
	}
}

func TestMainDefaultNoAdapterPreservesForeignGlobalAsComment(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "door.py")
	source := `import * as hsm from "@stateforward/hsm.ts";

function record(value: string): string {
  return value.toUpperCase();
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		"# Original typescript global global_1 preserved for manual porting:",
		"# Target language: python",
		"# function record(value: string): string {",
		"#   return value.toUpperCase();",
		`door_model = hsm.Define(`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("default no-adapter CLI output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "\nfunction record(value: string): string") {
		t.Fatalf("default no-adapter CLI emitted TypeScript global as executable Python:\n%s", text)
	}
}

func TestMainRoundTripsThroughJSONIRTransport(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	irOutput := filepath.Join(dir, "door.hsm.json")
	pythonOutput := filepath.Join(dir, "door.py")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	toIR := exec.Command("go", "run", ".", "-in", input, "-out", irOutput)
	toIR.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	if data, err := toIR.CombinedOutput(); err != nil {
		t.Fatalf("go run hsmc to JSON IR failed: %v\n%s", err, data)
	}
	irData, err := os.ReadFile(irOutput)
	if err != nil {
		t.Fatal(err)
	}
	var transport hsmc.Program
	if err := json.Unmarshal(irData, &transport); err != nil {
		t.Fatalf("json-ir output = %v\n%s", err, irData)
	}
	if transport.SourceLanguage != hsmc.LanguageTS || len(transport.Models) != 1 || len(transport.Behaviors) != 1 {
		t.Fatalf("json-ir transport program = %#v", transport)
	}
	if transport.Behaviors[0].TargetLanguage != "" || transport.Behaviors[0].TargetABI != nil {
		t.Fatalf("json-ir transport leaked target behavior metadata: %#v", transport.Behaviors[0])
	}

	fromIR := exec.Command("go", "run", ".", "-in", irOutput, "-out", pythonOutput)
	fromIR.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	if data, err := fromIR.CombinedOutput(); err != nil {
		t.Fatalf("go run hsmc from JSON IR failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(pythonOutput)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, needle := range []string{
		"import hsm",
		`door_model = hsm.Define(`,
		`hsm.Entry(entry_1)`,
		"# Original typescript behavior entry_1 preserved for manual porting:",
		`# instance.log = "closed";`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("json-ir CLI round trip output missing %q:\n%s", needle, text)
		}
	}
}

func TestMainInfersLanguagesForAdapterProtocol(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "protocol.py")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-in", input, "-out", output, "-print-adapter-protocol")
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var protocol hsmc.AdapterProtocol
	if err := json.Unmarshal(generated, &protocol); err != nil {
		t.Fatalf("protocol JSON = %v\n%s", err, generated)
	}
	if protocol.Request.SourceLanguage != hsmc.LanguageTS || protocol.Request.TargetLanguage != hsmc.LanguagePython {
		t.Fatalf("protocol request languages = %q -> %q, want typescript -> python", protocol.Request.SourceLanguage, protocol.Request.TargetLanguage)
	}
	if protocol.Guidance.TargetLanguage != hsmc.LanguagePython {
		t.Fatalf("protocol guidance target = %q, want python", protocol.Guidance.TargetLanguage)
	}
	if len(protocol.Request.Behaviors) != 1 || protocol.Request.Behaviors[0].TargetABI == nil || protocol.Request.Behaviors[0].TargetABI.Language != hsmc.LanguagePython {
		t.Fatalf("protocol behavior ABI = %#v, want python target ABI", protocol.Request.Behaviors)
	}
}

func TestMainPrintAdapterProtocolListsZigOperationReferenceFallbackSymbols(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.go")
	output := filepath.Join(dir, "protocol.zig")
	source := `package sample

import hsm "github.com/stateforward/hsm.go"

type DoorHSM struct{ hsm.HSM }

func approve(sm *DoorHSM) bool {
	return true
}

var DoorModel = hsm.Define(
	"Door",
	hsm.Operation("approve", approve),
	hsm.Initial(hsm.Target("closed"), hsm.Effect("approve")),
	hsm.State("closed",
		hsm.Entry("approve"),
		hsm.Exit("approve"),
		hsm.Activity("approve"),
		hsm.Transition(hsm.On("open"), hsm.Guard("approve"), hsm.Target("../open"), hsm.Effect("approve")),
	),
	hsm.State("open"),
)
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "go", "-to", "zig", "-in", input, "-out", output, "-print-adapter-protocol")
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc Zig operation protocol failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var protocol hsmc.AdapterProtocol
	if err := json.Unmarshal(generated, &protocol); err != nil {
		t.Fatalf("protocol JSON = %v\n%s", err, generated)
	}
	if protocol.Request.SourceLanguage != hsmc.LanguageGo || protocol.Request.TargetLanguage != hsmc.LanguageZig {
		t.Fatalf("protocol request languages = %q -> %q, want go -> zig", protocol.Request.SourceLanguage, protocol.Request.TargetLanguage)
	}
	for _, prefix := range []string{
		"hsmc_effect_operation_approve_",
		"hsmc_entry_operation_approve_",
		"hsmc_exit_operation_approve_",
		"hsmc_activity_operation_approve_",
		"hsmc_guard_operation_approve_",
	} {
		if !containsStringWithPrefix(protocol.Guidance.CompilerOwnedSymbols, prefix) {
			t.Fatalf("protocol guidance missing Zig operation fallback symbol prefix %q: %#v", prefix, protocol.Guidance.CompilerOwnedSymbols)
		}
	}
}

func TestMainPrintAdapterProtocolJSONOutputDoesNotInferTransportTarget(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "protocol.json")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-in", input, "-out", output, "-print-adapter-protocol")
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var protocol hsmc.AdapterProtocol
	if err := json.Unmarshal(generated, &protocol); err != nil {
		t.Fatalf("protocol JSON = %v\n%s", err, generated)
	}
	if protocol.Request.SourceLanguage != hsmc.LanguageTS || protocol.Request.TargetLanguage != hsmc.LanguageGo {
		t.Fatalf("protocol request languages = %q -> %q, want typescript -> go", protocol.Request.SourceLanguage, protocol.Request.TargetLanguage)
	}
}

func TestMainPrintAdapterProtocolReadsStdinAndWritesStdout(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    instance.log = "closed";
  })),
);
`
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", "-", "-print-adapter-protocol")
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	command.Stdin = strings.NewReader(source)
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc stdin adapter protocol failed: %v\n%s", err, data)
	}
	var protocol hsmc.AdapterProtocol
	if err := json.Unmarshal(data, &protocol); err != nil {
		t.Fatalf("protocol JSON = %v\n%s", err, data)
	}
	if protocol.Request.SourceLanguage != hsmc.LanguageTS || protocol.Request.TargetLanguage != hsmc.LanguagePython {
		t.Fatalf("protocol request languages = %q -> %q, want typescript -> python", protocol.Request.SourceLanguage, protocol.Request.TargetLanguage)
	}
	if len(protocol.Request.Behaviors) != 1 || protocol.Request.Behaviors[0].TargetABI == nil || protocol.Request.Behaviors[0].TargetABI.Language != hsmc.LanguagePython {
		t.Fatalf("protocol behavior ABI = %#v, want python target ABI", protocol.Request.Behaviors)
	}
}

func TestMainPrintAdapterProtocolCarriesStructuredRegionImports(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	output := filepath.Join(dir, "protocol.py")
	source := `import * as hsm from "@stateforward/hsm.ts";
import { record as rec } from "./format";
import { normalize } from "./normalize";

function localFormat(value: string): string {
  return normalize(value);
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed", hsm.Entry((ctx, instance, event) => {
    rec(localFormat("closed"));
  })),
);
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".", "-from", "typescript", "-to", "python", "-in", input, "-out", output, "-print-adapter-protocol")
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var protocol hsmc.AdapterProtocol
	if err := json.Unmarshal(generated, &protocol); err != nil {
		t.Fatalf("protocol JSON = %v\n%s", err, generated)
	}
	if len(protocol.Request.Globals) != 1 || len(protocol.Request.Globals[0].Imports) != 1 {
		t.Fatalf("protocol request global imports = %#v", protocol.Request.Globals)
	}
	globalImport := protocol.Request.Globals[0].Imports[0]
	if globalImport.Path != "./normalize" || len(globalImport.Specifiers) != 1 || globalImport.Specifiers[0].Name != "normalize" {
		t.Fatalf("protocol request global import = %#v", globalImport)
	}
	if strings.Join(globalImport.LocalNames, ",") != "normalize" {
		t.Fatalf("protocol request global import local names = %#v", globalImport.LocalNames)
	}
	if len(protocol.Request.Behaviors) != 1 || len(protocol.Request.Behaviors[0].Imports) != 1 {
		t.Fatalf("protocol request behavior imports = %#v", protocol.Request.Behaviors)
	}
	behaviorImport := protocol.Request.Behaviors[0].Imports[0]
	if behaviorImport.Path != "./format" || len(behaviorImport.Specifiers) != 1 || behaviorImport.Specifiers[0].Name != "record" || behaviorImport.Specifiers[0].Alias != "rec" {
		t.Fatalf("protocol request behavior import = %#v", behaviorImport)
	}
	if strings.Join(behaviorImport.LocalNames, ",") != "rec" {
		t.Fatalf("protocol request behavior import local names = %#v", behaviorImport.LocalNames)
	}
	if len(protocol.AllowedEdits.GlobalImports) != 1 || len(protocol.AllowedEdits.GlobalImports[0].Imports) != 1 {
		t.Fatalf("protocol allowed global import regions = %#v", protocol.AllowedEdits.GlobalImports)
	}
	allowedGlobalImport := protocol.AllowedEdits.GlobalImports[0].Imports[0]
	if allowedGlobalImport.Path != "./normalize" || len(allowedGlobalImport.Specifiers) != 1 || allowedGlobalImport.Specifiers[0].Name != "normalize" {
		t.Fatalf("protocol allowed global import = %#v", allowedGlobalImport)
	}
	if strings.Join(protocol.AllowedEdits.GlobalImports[0].ImportPaths, ",") != "./normalize" {
		t.Fatalf("protocol allowed global import paths = %#v", protocol.AllowedEdits.GlobalImports[0].ImportPaths)
	}
	if len(protocol.AllowedEdits.BehaviorImports) != 1 || len(protocol.AllowedEdits.BehaviorImports[0].Imports) != 1 {
		t.Fatalf("protocol allowed behavior import regions = %#v", protocol.AllowedEdits.BehaviorImports)
	}
	allowedImport := protocol.AllowedEdits.BehaviorImports[0].Imports[0]
	if allowedImport.Path != "./format" || len(allowedImport.Specifiers) != 1 || allowedImport.Specifiers[0].Alias != "rec" {
		t.Fatalf("protocol allowed behavior import = %#v", allowedImport)
	}
	if strings.Join(protocol.AllowedEdits.BehaviorImports[0].ImportPaths, ",") != "./format" {
		t.Fatalf("protocol allowed behavior import paths = %#v", protocol.AllowedEdits.BehaviorImports[0].ImportPaths)
	}
}

func TestMainPrintAdapterProtocolFromJSONIRRetargetsMutableContext(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.hsm.json")
	output := filepath.Join(dir, "protocol.dart")
	source := `{
  "source_language": "typescript",
  "globals": [{
    "id": "global_1",
    "language": "typescript",
    "code": "function localFormat(value: string): string { return normalize(value); }",
    "imports": [{
      "path": "./normalize",
      "specifiers": [{ "name": "normalize" }],
      "local_names": ["normalize"],
      "language": "typescript"
    }]
  }],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "id": "initial_1", "owner_id": "model_1", "target": "closed" },
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
    "source_language": "typescript",
    "target_language": "python",
    "target_abi": {
      "language": "python",
      "signature": "def entry_1(ctx, instance, event) -> None:"
    },
    "body": "rec(localFormat(\"closed\"));",
    "imports": [{
      "path": "./format",
      "specifiers": [{ "name": "record", "alias": "rec" }],
      "local_names": ["rec"],
      "language": "typescript"
    }]
  }]
}`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	command := exec.Command("go", "run", ".", "-from", "json-ir", "-to", "dart", "-in", input, "-out", output, "-print-adapter-protocol")
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc JSON IR adapter protocol failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var protocol hsmc.AdapterProtocol
	if err := json.Unmarshal(generated, &protocol); err != nil {
		t.Fatalf("protocol JSON = %v\n%s", err, generated)
	}
	if protocol.Request.SourceLanguage != hsmc.LanguageTS || protocol.Request.TargetLanguage != hsmc.LanguageDart {
		t.Fatalf("protocol request languages = %q -> %q, want typescript -> dart", protocol.Request.SourceLanguage, protocol.Request.TargetLanguage)
	}
	if !containsString(protocol.Response.ImportFields, "hidden") {
		t.Fatalf("JSON IR protocol response import fields missing hidden: %#v", protocol.Response.ImportFields)
	}
	if protocol.Guidance.TargetImportShape.SupportsHiddenSpecifiers {
		t.Fatalf("JSON IR protocol Dart import shape should reject adapter hidden specifiers: %#v", protocol.Guidance.TargetImportShape)
	}
	if !containsString(protocol.Guidance.TargetImportShape.Notes, "Native Dart source context may preserve hide combinators, but translated target imports cannot use them because they implicitly bind unlisted names.") {
		t.Fatalf("JSON IR protocol Dart import shape missing hidden specifier boundary note: %#v", protocol.Guidance.TargetImportShape)
	}
	if !containsString(protocol.Guidance.TargetImportShape.Notes, "Do not combine a Dart prefix alias with named specifiers in translated target imports; use either a prefix import or a show import.") {
		t.Fatalf("JSON IR protocol Dart import shape missing prefix/show boundary note: %#v", protocol.Guidance.TargetImportShape)
	}
	if len(protocol.Request.Globals) != 1 || len(protocol.Request.Globals[0].Imports) != 1 {
		t.Fatalf("JSON IR protocol global imports = %#v", protocol.Request.Globals)
	}
	if got := protocol.Request.Globals[0].Imports[0]; got.Path != "./normalize" || got.Language != hsmc.LanguageTS || got.Specifiers[0].Name != "normalize" {
		t.Fatalf("JSON IR protocol global import = %#v", got)
	}
	if len(protocol.Request.Behaviors) != 1 || len(protocol.Request.Behaviors[0].Imports) != 1 {
		t.Fatalf("JSON IR protocol behavior imports = %#v", protocol.Request.Behaviors)
	}
	behavior := protocol.Request.Behaviors[0]
	if behavior.TargetLanguage != "" {
		t.Fatalf("stale JSON IR behavior target language reached protocol request: %#v", behavior)
	}
	if behavior.TargetABI == nil || behavior.TargetABI.Language != hsmc.LanguageDart || !strings.Contains(behavior.TargetABI.Signature, "Context ctx") {
		t.Fatalf("JSON IR protocol behavior ABI = %#v, want Dart ABI", behavior.TargetABI)
	}
	if strings.Contains(behavior.TargetABI.Signature, "def entry_1") {
		t.Fatalf("stale JSON IR Python ABI reached protocol request: %#v", behavior.TargetABI)
	}
	behaviorImport := behavior.Imports[0]
	if behaviorImport.Path != "./format" || behaviorImport.Language != hsmc.LanguageTS || behaviorImport.Specifiers[0].Alias != "rec" {
		t.Fatalf("JSON IR protocol behavior import = %#v", behaviorImport)
	}
	if len(protocol.AllowedEdits.GlobalImports) != 1 || protocol.AllowedEdits.GlobalImports[0].Imports[0].Path != "./normalize" {
		t.Fatalf("JSON IR protocol allowed global imports = %#v", protocol.AllowedEdits.GlobalImports)
	}
	if len(protocol.AllowedEdits.BehaviorImports) != 1 || protocol.AllowedEdits.BehaviorImports[0].Imports[0].Path != "./format" {
		t.Fatalf("JSON IR protocol allowed behavior imports = %#v", protocol.AllowedEdits.BehaviorImports)
	}
}

func TestMainPrintAdapterProtocolCarriesTriggerSpecificTargetABI(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "timer.go")
	output := filepath.Join(dir, "protocol.dart")
	source := `package sample

import (
	"context"
	"time"

	hsm "github.com/stateforward/hsm.go"
)

type Timer struct{}

var TimerModel = hsm.Define(
	"Timer",
	hsm.Initial(hsm.Target("waiting")),
	hsm.State("waiting",
		hsm.Transition(
			hsm.At(func(ctx context.Context, instance *Timer, event hsm.Event) time.Time {
				return time.Now()
			}),
			hsm.Target("../done"),
		),
	),
	hsm.State("done"),
)
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	command := exec.Command("go", "run", ".", "-from", "go", "-to", "dart", "-in", input, "-out", output, "-print-adapter-protocol")
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc trigger adapter protocol failed: %v\n%s", err, data)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var protocol hsmc.AdapterProtocol
	if err := json.Unmarshal(generated, &protocol); err != nil {
		t.Fatalf("protocol JSON = %v\n%s", err, generated)
	}
	if protocol.Request.SourceLanguage != hsmc.LanguageGo || protocol.Request.TargetLanguage != hsmc.LanguageDart {
		t.Fatalf("protocol request languages = %q -> %q, want go -> dart", protocol.Request.SourceLanguage, protocol.Request.TargetLanguage)
	}
	for _, binding := range []string{"at", "DateTime", "Duration", "every", "Future", "onCall", "when"} {
		if !containsString(protocol.Guidance.CompilerOwnedBindings, binding) {
			t.Fatalf("protocol guidance missing Dart compiler-owned binding %q: %#v", binding, protocol.Guidance.CompilerOwnedBindings)
		}
	}
	for _, behavior := range protocol.Request.Behaviors {
		if behavior.Kind != hsmc.BehaviorTrigger || behavior.TriggerKind != hsmc.TriggerAt {
			continue
		}
		if behavior.TargetLanguage != "" {
			t.Fatalf("protocol trigger behavior leaked target_language: %#v", behavior)
		}
		if behavior.TargetABI == nil || behavior.TargetABI.Language != hsmc.LanguageDart || behavior.TargetABI.ReturnType != "FutureOr<DateTime>" {
			t.Fatalf("protocol trigger ABI = %#v, want Dart At trigger ABI", behavior.TargetABI)
		}
		if !strings.Contains(behavior.TargetABI.Signature, "Context ctx") || strings.Contains(behavior.TargetABI.Signature, "time.Time") {
			t.Fatalf("protocol trigger ABI signature = %q, want Dart signature without stale Go metadata", behavior.TargetABI.Signature)
		}
		return
	}
	t.Fatalf("protocol request missing At trigger behavior: %#v", protocol.Request.Behaviors)
}

func TestMainPrintAdapterProtocolDoesNotInvokeConfiguredAdapter(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.go")
	output := filepath.Join(dir, "protocol.ts")
	adapter := filepath.Join(dir, "adapter.sh")
	if err := os.WriteFile(input, []byte(goCLIProtocolSource), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte("#!/bin/sh\nprintf 'adapter should not run\\n' >&2\nexit 99\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	command := exec.Command("go", "run", ".", "-from", "go", "-to", "ts", "-in", input, "-out", output, "-adapter", "command", "-adapter-command", adapter, "-print-adapter-protocol")
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run hsmc failed: %v\n%s", err, data)
	}

	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var protocol hsmc.AdapterProtocol
	if err := json.Unmarshal(generated, &protocol); err != nil {
		t.Fatalf("protocol JSON = %v\n%s", err, generated)
	}
	if protocol.Request.TargetLanguage != hsmc.LanguageTS {
		t.Fatalf("protocol target language = %q, want TypeScript", protocol.Request.TargetLanguage)
	}
	if !containsString(protocol.Guidance.CompilerOwnedBindings, "hsm") {
		t.Fatalf("protocol guidance missing TypeScript compiler-owned hsm binding: %#v", protocol.Guidance.CompilerOwnedBindings)
	}
	if !containsString(protocol.Guidance.CompilerOwnedSymbols, "entry1") {
		t.Fatalf("protocol guidance missing generated behavior symbol: %#v", protocol.Guidance.CompilerOwnedSymbols)
	}
	if !containsString(protocol.Guidance.ForbiddenFields, "imports that bind compiler-owned symbols") {
		t.Fatalf("protocol guidance does not forbid imports that bind compiler-owned symbols: %#v", protocol.Guidance.ForbiddenFields)
	}
	if !strings.Contains(protocol.Instruction, "bind names listed in guidance.compiler_owned_symbols") {
		t.Fatalf("protocol instruction missing compiler-owned import binding rule: %s", protocol.Instruction)
	}
	if containsString(protocol.Response.TopLevelFields, "models") ||
		containsString(protocol.Response.TopLevelFields, "events") ||
		containsString(protocol.Response.TopLevelFields, "states") ||
		containsString(protocol.Response.TopLevelFields, "transitions") ||
		containsString(protocol.Response.TopLevelFields, "source_language") ||
		containsString(protocol.Response.TopLevelFields, "target_language") ||
		containsString(protocol.Response.TopLevelFields, "package_name") {
		t.Fatalf("protocol response made compiler-owned structure editable: %#v", protocol.Response.TopLevelFields)
	}
	for _, field := range []string{"source_language", "target_language", "package_name", "models", "events", "states", "transitions"} {
		if !containsString(protocol.Guidance.ForbiddenFields, field) {
			t.Fatalf("protocol guidance does not forbid compiler-owned response field %q: %#v", field, protocol.Guidance.ForbiddenFields)
		}
	}
	for _, field := range []string{
		"globals[].language",
		"imports[].local_names",
		"imports[].range",
		"imports[].specifiers[].range",
		"globals[].imports[].local_names",
		"globals[].imports[].range",
		"globals[].imports[].specifiers[].range",
		"behaviors[].owner_id",
		"behaviors[].kind",
		"behaviors[].source_language",
		"behaviors[].target_language",
		"behaviors[].signature",
		"behaviors[].target_abi",
		"behaviors[].imports[].local_names",
		"behaviors[].imports[].range",
		"behaviors[].imports[].specifiers[].range",
	} {
		if !containsString(protocol.Guidance.ForbiddenFields, field) {
			t.Fatalf("protocol guidance does not forbid compiler-owned nested response field %q: %#v", field, protocol.Guidance.ForbiddenFields)
		}
	}
	if !strings.Contains(protocol.Response.UnknownFieldPolicy, "models") ||
		!strings.Contains(protocol.Response.UnknownFieldPolicy, "transitions") {
		t.Fatalf("protocol unknown-field policy missing model restrictions: %s", protocol.Response.UnknownFieldPolicy)
	}
}

func TestMainPrintAdapterProtocolRejectsJSONIRTarget(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.go")
	output := filepath.Join(dir, "protocol.hsm.json")
	if err := os.WriteFile(input, []byte(goCLIProtocolSource), 0o644); err != nil {
		t.Fatal(err)
	}

	command := exec.Command("go", "run", ".", "-from", "go", "-to", "json-ir", "-in", input, "-out", output, "-print-adapter-protocol")
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly printed adapter protocol for JSON IR target:\n%s", data)
	}
	if !strings.Contains(string(data), `adapter protocol cannot target transport language "json-ir"`) {
		t.Fatalf("go run hsmc error missing JSON IR protocol rejection:\n%s", data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("JSON IR adapter protocol rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected protocol output: %v", readErr)
	}
}

func TestMainPrintAdapterProtocolRejectsTargetGlobalSymbolCollisionsWithoutWritingOutput(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.hsm.json")
	output := filepath.Join(dir, "protocol.ts")
	source := `{
  "source_language": "typescript",
  "globals": [{
    "id": "global_1",
    "language": "typescript",
    "code": "function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {}"
  }],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "id": "initial_1", "owner_id": "model_1", "target": "closed" },
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
    "source_language": "typescript",
    "body": "instance.log = \"closed\";"
  }]
}`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	command := exec.Command("go", "run", ".", "-from", "json-ir", "-to", "typescript", "-in", input, "-out", output, "-print-adapter-protocol")
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly printed adapter protocol for target global symbol collision:\n%s", data)
	}
	want := `target-language global "global_1" declares "entry1", which collides with compiler-owned typescript symbol`
	if !strings.Contains(string(data), want) {
		t.Fatalf("go run hsmc error missing adapter protocol target global collision rejection %q:\n%s", want, data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("target global collision adapter protocol rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected target global collision protocol output: %v", readErr)
	}
}

func TestMainRejectsTargetImportGeneratedSymbolCollisionsWithoutWritingOutput(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.hsm.json")
	output := filepath.Join(dir, "door.ts")
	source := `{
  "source_language": "typescript",
  "imports": [{
    "path": "./helpers",
    "language": "typescript",
    "specifiers": [{ "name": "entry1" }]
  }],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "id": "initial_1", "owner_id": "model_1", "target": "closed" },
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
    "source_language": "typescript",
    "body": "instance.log = \"closed\";"
  }]
}`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	command := exec.Command("go", "run", ".", "-from", "json-ir", "-to", "typescript", "-in", input, "-out", output)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted target import generated-symbol collision:\n%s", data)
	}
	want := `target imports for program import "./helpers" bind local name "entry1", which collides with compiler-owned typescript symbol`
	if !strings.Contains(string(data), want) {
		t.Fatalf("go run hsmc error missing target import generated-symbol collision rejection %q:\n%s", want, data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("target import generated-symbol collision rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected target import collision output: %v", readErr)
	}
}

func TestSplitArgsTrimsEmptyItems(t *testing.T) {
	args := splitArgs(" --model , gpt-5 ,, --ask-for-approval=never ")
	want := []string{"--model", "gpt-5", "--ask-for-approval=never"}
	if len(args) != len(want) {
		t.Fatalf("splitArgs len = %d, want %d: %#v", len(args), len(want), args)
	}
	for index := range want {
		if args[index] != want[index] {
			t.Fatalf("splitArgs[%d] = %q, want %q", index, args[index], want[index])
		}
	}
}

func TestFormatLanguages(t *testing.T) {
	got := formatLanguages([]hsmc.Language{hsmc.LanguageGo, hsmc.LanguageTS, hsmc.LanguageJSONIR})
	want := "go\ntypescript\njson-ir"
	if got != want {
		t.Fatalf("formatLanguages() = %q, want %q", got, want)
	}
}

func TestMainListCommandsDoNotRequireInput(t *testing.T) {
	cases := []struct {
		name string
		flag string
		want string
	}{
		{name: "all languages", flag: "-list-languages", want: formatLanguages(hsmc.SupportedLanguages())},
		{name: "source languages", flag: "-list-source-languages", want: formatLanguages(hsmc.SupportedSourceLanguages())},
		{name: "target languages", flag: "-list-target-languages", want: formatLanguages(hsmc.SupportedTargetLanguages())},
		{name: "adapters", flag: "-list-adapters", want: formatAdapters(hsmc.NewCompiler().AdapterNames())},
		{name: "version", flag: "-version", want: hsmc.Version},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			command := exec.Command("go", "run", ".", tc.flag)
			command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
			data, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("go run hsmc %s failed: %v\n%s", tc.flag, err, data)
			}
			text := string(data)
			if strings.Contains(text, "-in is required") {
				t.Fatalf("%s unexpectedly required input:\n%s", tc.flag, text)
			}
			if strings.TrimSpace(text) != tc.want {
				t.Fatalf("%s output = %q, want %q", tc.flag, strings.TrimSpace(text), tc.want)
			}
		})
	}
}

func TestMainRejectsUnsupportedLanguagesWithoutWritingOutput(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.ts")
	source := `import * as hsm from "@stateforward/hsm.ts";

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State("closed"),
);
`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "source",
			args: []string{"-from", "ruby", "-to", "python"},
			want: `unsupported source language "ruby"`,
		},
		{
			name: "target",
			args: []string{"-from", "typescript", "-to", "ruby"},
			want: `unsupported target language "ruby"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output := filepath.Join(dir, tc.name+".out")
			args := append([]string{"run", "."}, tc.args...)
			args = append(args, "-in", input, "-out", output)
			command := exec.Command("go", args...)
			command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
			data, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("go run hsmc unexpectedly accepted unsupported %s language:\n%s", tc.name, data)
			}
			if !strings.Contains(string(data), tc.want) {
				t.Fatalf("go run hsmc error missing %q:\n%s", tc.want, data)
			}
			if generated, readErr := os.ReadFile(output); readErr == nil {
				t.Fatalf("unsupported %s language rejection should not write output, got:\n%s", tc.name, generated)
			} else if !os.IsNotExist(readErr) {
				t.Fatalf("read rejected %s language output: %v", tc.name, readErr)
			}
		})
	}
}

func TestMainRejectsGeneratedTargetSymbolErrorsWithoutWritingOutput(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.hsm.json")
	output := filepath.Join(dir, "door.py")
	source := `{
  "source_language": "typescript",
  "events": [{
    "id": "event_1",
    "symbol": "class",
    "name": "class"
  }],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "id": "initial_1", "owner_id": "model_1", "target": "closed" },
    "states": [{
      "id": "state_1",
      "name": "closed",
      "kind": "state"
    }]
  }]
}`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	command := exec.Command("go", "run", ".", "-from", "json-ir", "-to", "python", "-in", input, "-out", output)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted invalid generated target symbol:\n%s", data)
	}
	want := `compiler-owned python symbol "class" for event "event_1" is not a valid target identifier`
	if !strings.Contains(string(data), want) {
		t.Fatalf("go run hsmc error missing generated target symbol rejection %q:\n%s", want, data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("generated target symbol rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected generated target symbol output: %v", readErr)
	}
}

func TestMainRejectsTargetGlobalSymbolCollisionsWithoutWritingOutput(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "door.hsm.json")
	output := filepath.Join(dir, "door.ts")
	source := `{
  "source_language": "typescript",
  "globals": [{
    "id": "global_1",
    "language": "typescript",
    "code": "function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {}"
  }],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "id": "initial_1", "owner_id": "model_1", "target": "closed" },
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
    "source_language": "typescript",
    "body": "instance.log = \"closed\";"
  }]
}`
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	command := exec.Command("go", "run", ".", "-from", "json-ir", "-to", "typescript", "-in", input, "-out", output)
	command.Env = append(os.Environ(), "ASDF_GOLANG_VERSION=1.25.1")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("go run hsmc unexpectedly accepted target global symbol collision:\n%s", data)
	}
	want := `target-language global "global_1" declares "entry1", which collides with compiler-owned typescript symbol`
	if !strings.Contains(string(data), want) {
		t.Fatalf("go run hsmc error missing target global collision rejection %q:\n%s", want, data)
	}
	if generated, readErr := os.ReadFile(output); readErr == nil {
		t.Fatalf("target global collision rejection should not write output, got:\n%s", generated)
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read rejected target global collision output: %v", readErr)
	}
}

func TestCompilerLanguageListsExposeSourceTargetSplit(t *testing.T) {
	want := strings.Join([]string{
		"csharp",
		"cpp",
		"dart",
		"go",
		"java",
		"javascript",
		"python",
		"typescript",
		"rust",
		"zig",
		"json-ir",
	}, "\n")
	wantTargets := strings.Join([]string{
		"csharp",
		"cpp",
		"dart",
		"go",
		"java",
		"javascript",
		"python",
		"typescript",
		"xstate",
		"rust",
		"zig",
		"json-ir",
		"mermaid",
		"plantuml",
	}, "\n")
	if got := formatLanguages(hsmc.SupportedSourceLanguages()); got != want {
		t.Fatalf("source language list = %q, want %q", got, want)
	}
	if got := formatLanguages(hsmc.SupportedTargetLanguages()); got != wantTargets {
		t.Fatalf("target language list = %q, want %q", got, wantTargets)
	}
	if got := formatLanguages(hsmc.SupportedLanguages()); got != wantTargets {
		t.Fatalf("combined language list = %q, want %q", got, wantTargets)
	}
}

func TestFormatAdapters(t *testing.T) {
	got := formatAdapters([]string{"none", "codex", "command"})
	want := "none\ncodex\ncommand"
	if got != want {
		t.Fatalf("formatAdapters() = %q, want %q", got, want)
	}
}

func TestCompilerAdapterNamesAreUsedForListAdapters(t *testing.T) {
	got := formatAdapters(hsmc.NewCompiler().AdapterNames())
	want := "codex\ncommand\nnone"
	if got != want {
		t.Fatalf("registered adapters = %q, want %q", got, want)
	}
}

func TestAdapterProtocolJSONBuildsEnvelopeWithoutAdapter(t *testing.T) {
	output, err := adapterProtocolJSON(context.Background(), hsmc.NewCompiler(), hsmc.SourceInput{Path: "door.go", Data: []byte(goCLIProtocolSource)}, hsmc.CompileOptions{
		From: hsmc.LanguageGo,
		To:   hsmc.LanguageTS,
	})
	if err != nil {
		t.Fatal(err)
	}
	var protocol hsmc.AdapterProtocol
	if err := json.Unmarshal(output, &protocol); err != nil {
		t.Fatal(err)
	}
	if protocol.Version != "hsmc.adapter.v1" || protocol.Request.TargetLanguage != hsmc.LanguageTS {
		t.Fatalf("protocol metadata = %#v", protocol)
	}
	if len(protocol.Request.Models) != 1 || protocol.Request.Models[0].Name != "Door" {
		t.Fatalf("protocol model context = %#v", protocol.Request.Models)
	}
	if len(protocol.Request.Behaviors) != 1 || protocol.Request.Behaviors[0].TargetABI == nil || protocol.Request.Behaviors[0].TargetABI.Language != hsmc.LanguageTS {
		t.Fatalf("protocol behavior ABI = %#v", protocol.Request.Behaviors)
	}
	if !strings.Contains(string(output), "\"target_import_shape\"") {
		t.Fatalf("protocol output missing guidance:\n%s", output)
	}
}

func TestAdapterProtocolJSONRejectsTransportTarget(t *testing.T) {
	_, err := adapterProtocolJSON(context.Background(), hsmc.NewCompiler(), hsmc.SourceInput{Path: "door.go", Data: []byte(goCLIProtocolSource)}, hsmc.CompileOptions{
		From: hsmc.LanguageGo,
		To:   hsmc.LanguageJSONIR,
	})
	if err == nil || !strings.Contains(err.Error(), `adapter protocol cannot target transport language "json-ir"`) {
		t.Fatalf("error = %v, want JSON IR target rejection", err)
	}
}

const goCLIProtocolSource = `package sample

import hsm "github.com/stateforward/hsm.go"

type Door struct{}

func entered(ctx context.Context, instance hsm.Instance, event hsm.Event) {
}

var DoorModel = hsm.Define(
	"Door",
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed", hsm.Entry(entered)),
)
`

func writeTestFile(t *testing.T, path string, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

const directoryGoSource = `package sample

import hsm "github.com/stateforward/hsm.go"

type Instance struct {
	Log []string
}

var DoorModel = hsm.Define("Door",
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed",
		hsm.Entry(func(ctx hsm.Context, instance *Instance, event hsm.Event) {
			instance.Log = append(instance.Log, "entered")
		}),
		hsm.Transition(hsm.On("open"), hsm.Target("../open")),
	),
	hsm.State("open"),
)
`

const directoryTypeScriptSource = `import * as hsm from "@stateforward/hsm.ts";

class Door extends hsm.Instance {
  log: string[] = [];
}

export const DoorModel = hsm.Define(
  "Door",
  hsm.Initial(hsm.Target("closed")),
  hsm.State(
    "closed",
    hsm.Entry((ctx, instance: Door, event) => {
      instance.log.push("entered");
    }),
    hsm.Transition(hsm.On("open"), hsm.Target("../open")),
  ),
  hsm.State("open"),
);
`

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsStringWithPrefix(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}
