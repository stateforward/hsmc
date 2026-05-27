package hsmc

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestSupportedLanguagesAreRegisteredAsSourcesOrTargets(t *testing.T) {
	compiler := NewCompiler()
	wantAll := []Language{LanguageCSharp, LanguageCPP, LanguageDart, LanguageElixir, LanguageGo, LanguageJava, LanguageJS, LanguagePython, LanguageTS, LanguageXState, LanguageRust, LanguageZig, LanguageJSONIR, LanguageMermaid, LanguagePlantUML}
	wantSources := []Language{LanguageCSharp, LanguageCPP, LanguageDart, LanguageGo, LanguageJava, LanguageJS, LanguagePython, LanguageTS, LanguageRust, LanguageZig, LanguageJSONIR}
	wantTargets := []Language{LanguageCSharp, LanguageCPP, LanguageDart, LanguageElixir, LanguageGo, LanguageJava, LanguageJS, LanguagePython, LanguageTS, LanguageXState, LanguageRust, LanguageZig, LanguageJSONIR, LanguageMermaid, LanguagePlantUML}

	if got := SupportedLanguages(); !reflect.DeepEqual(got, wantAll) {
		t.Fatalf("SupportedLanguages() = %#v, want %#v", got, wantAll)
	}
	if got := SupportedSourceLanguages(); !reflect.DeepEqual(got, wantSources) {
		t.Fatalf("SupportedSourceLanguages() = %#v, want %#v", got, wantSources)
	}
	if got := SupportedTargetLanguages(); !reflect.DeepEqual(got, wantTargets) {
		t.Fatalf("SupportedTargetLanguages() = %#v, want %#v", got, wantTargets)
	}
	if got := compiler.SourceLanguages(); !reflect.DeepEqual(got, wantSources) {
		t.Fatalf("SourceLanguages() = %#v, want %#v", got, wantSources)
	}
	if got := compiler.TargetLanguages(); !reflect.DeepEqual(got, wantTargets) {
		t.Fatalf("TargetLanguages() = %#v, want %#v", got, wantTargets)
	}
}

func TestSupportedAdapters(t *testing.T) {
	want := []string{"none", "codex", "command"}
	if got := SupportedAdapters(); !reflect.DeepEqual(got, want) {
		t.Fatalf("SupportedAdapters() = %#v, want %#v", got, want)
	}
}

func TestBuiltInAdaptersAreRegistered(t *testing.T) {
	want := []string{"codex", "command", "none"}
	if got := NewCompiler().AdapterNames(); !reflect.DeepEqual(got, want) {
		t.Fatalf("AdapterNames() = %#v, want %#v", got, want)
	}
}

func TestParseLanguageAliases(t *testing.T) {
	cases := map[string]Language{
		"c#":         LanguageCSharp,
		"c-sharp":    LanguageCSharp,
		"cs":         LanguageCSharp,
		"csharp":     LanguageCSharp,
		"c++":        LanguageCPP,
		"cc":         LanguageCPP,
		"cpp":        LanguageCPP,
		"cxx":        LanguageCPP,
		"dart":       LanguageDart,
		"elixir":     LanguageElixir,
		"ex":         LanguageElixir,
		"exs":        LanguageElixir,
		"hsm.ex":     LanguageElixir,
		"go":         LanguageGo,
		"golang":     LanguageGo,
		"java":       LanguageJava,
		"cjs":        LanguageJS,
		"js":         LanguageJS,
		"javascript": LanguageJS,
		"jsx":        LanguageJS,
		"mjs":        LanguageJS,
		"py":         LanguagePython,
		"python":     LanguagePython,
		"rs":         LanguageRust,
		"rust":       LanguageRust,
		"cts":        LanguageTS,
		"mts":        LanguageTS,
		"ts":         LanguageTS,
		"tsx":        LanguageTS,
		"typescript": LanguageTS,
		"x-state":    LanguageXState,
		"xstate":     LanguageXState,
		"zig":        LanguageZig,
		"ir":         LanguageJSONIR,
		"json":       LanguageJSONIR,
		"json-ir":    LanguageJSONIR,
		"mermaid":    LanguageMermaid,
		"mmd":        LanguageMermaid,
		"plant":      LanguagePlantUML,
		"plantuml":   LanguagePlantUML,
		"puml":       LanguagePlantUML,
	}
	for input, want := range cases {
		got, ok := ParseLanguage(input)
		if !ok || got != want {
			t.Fatalf("ParseLanguage(%q) = %q, %v; want %q, true", input, got, ok, want)
		}
	}
	if got, ok := ParseLanguage("ruby"); ok || got != "" {
		t.Fatalf("ParseLanguage(ruby) = %q, %v; want empty, false", got, ok)
	}
}

func TestCompilerCompileCanonicalizesLanguageAliases(t *testing.T) {
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    Language(" golang "),
		To:      Language(" ts "),
		Adapter: " none ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), `import * as hsm from "@stateforward/hsm.ts";`) {
		t.Fatalf("output does not look like TypeScript:\n%s", output)
	}
	if program.SourceLanguage != LanguageGo {
		t.Fatalf("source language = %q, want %q", program.SourceLanguage, LanguageGo)
	}
	for _, behavior := range program.Behaviors {
		if behavior.TargetABI == nil || behavior.TargetABI.Language != LanguageTS {
			t.Fatalf("behavior %s target ABI = %#v, want TypeScript", behavior.ID, behavior.TargetABI)
		}
	}
}

func TestCompilerBuildAdapterProtocolCanonicalizesLanguageAliases(t *testing.T) {
	protocol, program, err := NewCompiler().BuildAdapterProtocol(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From: Language("golang"),
		To:   Language("py"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if program.SourceLanguage != LanguageGo {
		t.Fatalf("source language = %q, want %q", program.SourceLanguage, LanguageGo)
	}
	if protocol.Request.TargetLanguage != LanguagePython || protocol.Guidance.TargetLanguage != LanguagePython {
		t.Fatalf("protocol target language = request %q guidance %q, want Python", protocol.Request.TargetLanguage, protocol.Guidance.TargetLanguage)
	}
	if len(protocol.Request.Behaviors) == 0 || protocol.Request.Behaviors[0].TargetABI == nil || protocol.Request.Behaviors[0].TargetABI.Language != LanguagePython {
		t.Fatalf("protocol behavior ABI = %#v, want Python", protocol.Request.Behaviors)
	}
}
