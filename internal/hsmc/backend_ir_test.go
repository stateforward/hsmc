package hsmc

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONIRBackendStripsTargetBehaviorMetadataWithoutMutatingProgram(t *testing.T) {
	program := &Program{
		SourceLanguage: LanguageGo,
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
		}},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			Inline:         true,
			TargetLanguage: LanguageTS,
			TargetABI: &BehaviorABI{
				Language:   LanguageTS,
				Signature:  "function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void",
				ReturnType: "void",
				Parameters: []ABIParameter{
					{Name: "ctx", Type: "hsm.Context"},
					{Name: "instance", Type: "InstanceType<typeof hsm.Instance>"},
					{Name: "event", Type: "hsm.Event"},
				},
			},
			Body: `instance["log"] = ["translated"];`,
		}},
	}

	output, err := NewJSONIRBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	var transport Program
	if err := json.Unmarshal(output, &transport); err != nil {
		t.Fatal(err)
	}
	if len(transport.Behaviors) != 1 {
		t.Fatalf("transport behaviors = %#v", transport.Behaviors)
	}
	if transport.Behaviors[0].TargetLanguage != "" || transport.Behaviors[0].TargetABI != nil {
		t.Fatalf("json-ir transport preserved target behavior metadata: %#v", transport.Behaviors[0])
	}
	if transport.Behaviors[0].Inline {
		t.Fatalf("json-ir transport preserved stale inline metadata: %#v", transport.Behaviors[0])
	}
	if program.Behaviors[0].TargetLanguage != LanguageTS {
		t.Fatalf("backend mutated original target language: %#v", program.Behaviors[0])
	}
	if !program.Behaviors[0].Inline {
		t.Fatalf("backend mutated original inline metadata: %#v", program.Behaviors[0])
	}
	if program.Behaviors[0].TargetABI == nil || program.Behaviors[0].TargetABI.Language != LanguageTS || len(program.Behaviors[0].TargetABI.Parameters) != 3 {
		t.Fatalf("backend mutated original target ABI: %#v", program.Behaviors[0].TargetABI)
	}
}

func TestJSONIRBackendRetagsAdapterTranslatedBehaviorSourceContext(t *testing.T) {
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:   "entry_1",
			Body: `instance["log"] = ["translated"];`,
			Imports: []Import{{
				Path:       "./behavior-format",
				Specifiers: []ImportSpecifier{{Name: "track"}},
			}},
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	_, program, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	var originalEntry *Behavior
	for index := range program.Behaviors {
		if program.Behaviors[index].ID == "entry_1" {
			originalEntry = &program.Behaviors[index]
			break
		}
	}
	if originalEntry == nil {
		t.Fatalf("compiled program missing entry_1: %#v", program.Behaviors)
	}
	if originalEntry.SourceLanguage != LanguageGo || originalEntry.TargetLanguage != LanguageTS || originalEntry.TargetABI == nil {
		t.Fatalf("compiled behavior was not target-translated before JSON IR emit: %#v", originalEntry)
	}

	output, err := NewJSONIRBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	var transport Program
	if err := json.Unmarshal(output, &transport); err != nil {
		t.Fatal(err)
	}
	var entry *Behavior
	for index := range transport.Behaviors {
		if transport.Behaviors[index].ID == "entry_1" {
			entry = &transport.Behaviors[index]
			break
		}
	}
	if entry == nil {
		t.Fatalf("json-ir transport missing entry_1: %#v", transport.Behaviors)
	}
	if entry.SourceLanguage != LanguageTS {
		t.Fatalf("json-ir translated behavior source_language = %q, want typescript: %#v", entry.SourceLanguage, entry)
	}
	if entry.TargetLanguage != "" || entry.TargetABI != nil {
		t.Fatalf("json-ir transport preserved target metadata: %#v", entry)
	}
	if entry.Signature != "" {
		t.Fatalf("json-ir transport preserved stale source signature: %#v", entry)
	}
	if entry.Inline {
		t.Fatalf("json-ir transport preserved stale source inline marker: %#v", entry)
	}
	if entry.Body != `instance["log"] = ["translated"];` {
		t.Fatalf("json-ir transport body = %q", entry.Body)
	}
	if len(entry.Imports) != 1 || entry.Imports[0].Language != LanguageTS {
		t.Fatalf("json-ir transport behavior imports = %#v, want target-language import", entry.Imports)
	}
	if originalEntry.SourceLanguage != LanguageGo || originalEntry.TargetLanguage != LanguageTS || originalEntry.TargetABI == nil {
		t.Fatalf("json-ir backend mutated compiled program behavior: %#v", originalEntry)
	}
}

func TestJSONIRAdapterTranslatedBehaviorRecompilesAsTargetSource(t *testing.T) {
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:   "entry_1",
			Body: `instance["log"] = ["translated"];`,
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	_, translated, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	ir, err := NewJSONIRBackend().Emit(context.Background(), translated)
	if err != nil {
		t.Fatal(err)
	}
	output, program, err := compiler.Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: ir}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`instance["log"] = ["translated"];`,
		"hsm.Entry(entry1)",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("recompiled TypeScript output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `// instance["log"] = ["translated"];`) {
		t.Fatalf("recompiled target behavior was commented instead of emitted as target source:\n%s", text)
	}
	var entry *Behavior
	for index := range program.Behaviors {
		if program.Behaviors[index].ID == "entry_1" {
			entry = &program.Behaviors[index]
			break
		}
	}
	if entry == nil {
		t.Fatalf("recompiled program missing entry_1: %#v", program.Behaviors)
	}
	if entry.SourceLanguage != LanguageTS || entry.TargetLanguage != "" || entry.TargetABI == nil || entry.TargetABI.Language != LanguageTS {
		t.Fatalf("recompiled behavior metadata = %#v, want TypeScript source with compiler-owned ABI only", entry)
	}
}

func TestJSONIRAdapterTranslatedFullCodeBehaviorRecompilesAsTargetSource(t *testing.T) {
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:   "entry_1",
			Code: `(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void => { instance["log"] = ["translated"]; }`,
			Imports: []Import{{
				Path:       "./behavior-format",
				Specifiers: []ImportSpecifier{{Name: "track"}},
			}},
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	_, translated, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	ir, err := NewJSONIRBackend().Emit(context.Background(), translated)
	if err != nil {
		t.Fatal(err)
	}
	var transport Program
	if err := json.Unmarshal(ir, &transport); err != nil {
		t.Fatal(err)
	}
	if len(transport.Behaviors) == 0 || transport.Behaviors[0].SourceLanguage != LanguageTS || transport.Behaviors[0].Signature != "" || transport.Behaviors[0].Inline {
		t.Fatalf("json-ir transport behavior source context = %#v, want TypeScript source without stale signature", transport.Behaviors)
	}
	if transport.Behaviors[0].TargetLanguage != "" || transport.Behaviors[0].TargetABI != nil {
		t.Fatalf("json-ir transport preserved target metadata for full-code behavior: %#v", transport.Behaviors[0])
	}
	if len(transport.Behaviors[0].Imports) != 1 || transport.Behaviors[0].Imports[0].Language != LanguageTS {
		t.Fatalf("json-ir transport behavior imports = %#v, want target-language import", transport.Behaviors[0].Imports)
	}

	output, program, err := compiler.Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: ir}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`import { track } from "./behavior-format";`,
		`const entry1 = (ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void => { instance["log"] = ["translated"]; };`,
		"hsm.Entry(entry1)",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("recompiled TypeScript full-code output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original typescript behavior entry_1 preserved for manual porting") ||
		strings.Contains(text, `// (ctx: hsm.Context`) {
		t.Fatalf("recompiled target full-code behavior was commented instead of emitted as target source:\n%s", text)
	}
	var entry *Behavior
	for index := range program.Behaviors {
		if program.Behaviors[index].ID == "entry_1" {
			entry = &program.Behaviors[index]
			break
		}
	}
	if entry == nil {
		t.Fatalf("recompiled program missing entry_1: %#v", program.Behaviors)
	}
	if entry.SourceLanguage != LanguageTS || entry.TargetLanguage != "" || entry.TargetABI == nil || entry.TargetABI.Language != LanguageTS {
		t.Fatalf("recompiled full-code behavior metadata = %#v, want TypeScript source with compiler-owned ABI only", entry)
	}
}

func TestJSONIRAdapterTranslatedGlobalRecompilesAsTargetSource(t *testing.T) {
	adapter := &patchingAdapter{patch: &BehaviorPatch{
		Globals: []CodeBlock{{
			ID:   "global_1",
			Code: "const record = (value: string): string => `event:${value}`;",
			Imports: []Import{{
				Path:       "./format",
				Specifiers: []ImportSpecifier{{Name: "formatRecord"}},
			}},
		}},
	}}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)

	_, translated, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(translated.Globals) == 0 || translated.Globals[0].Language != LanguageTS {
		t.Fatalf("compiled translated globals = %#v, want target-language global", translated.Globals)
	}

	ir, err := NewJSONIRBackend().Emit(context.Background(), translated)
	if err != nil {
		t.Fatal(err)
	}
	var transport Program
	if err := json.Unmarshal(ir, &transport); err != nil {
		t.Fatal(err)
	}
	if len(transport.Globals) == 0 || transport.Globals[0].Language != LanguageTS {
		t.Fatalf("json-ir transport globals = %#v, want target-language global", transport.Globals)
	}
	if len(transport.Globals[0].Imports) != 1 || transport.Globals[0].Imports[0].Language != LanguageTS {
		t.Fatalf("json-ir transport global imports = %#v, want target-language import", transport.Globals[0].Imports)
	}

	output, program, err := compiler.Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: ir}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`import { formatRecord } from "./format";`,
		"const record = (value: string): string => `event:${value}`;",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("recompiled TypeScript output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original go global global_1 preserved for manual porting") ||
		strings.Contains(text, "// const record =") {
		t.Fatalf("recompiled target global was commented instead of emitted as target source:\n%s", text)
	}
	if len(program.Globals) == 0 || program.Globals[0].Language != LanguageTS {
		t.Fatalf("recompiled program globals = %#v, want TypeScript source global", program.Globals)
	}
}

func TestJSONIRPreservesJavaStaticImportsAcrossRecompile(t *testing.T) {
	compiler := NewCompiler()
	ir, transport, err := compiler.Compile(context.Background(), SourceInput{Path: "DoorHsm.java", Data: []byte(javaStaticHelperImportSource)}, CompileOptions{
		From:    LanguageJava,
		To:      LanguageJSONIR,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	var staticImport *Import
	for index := range transport.Behaviors {
		for importIndex := range transport.Behaviors[index].Imports {
			imp := &transport.Behaviors[index].Imports[importIndex]
			if imp.Path == "com.example.Format.record" {
				staticImport = imp
			}
		}
	}
	if staticImport == nil || !staticImport.Static || staticImport.Language != LanguageJava {
		t.Fatalf("compiled json-ir program static import = %#v, behaviors = %#v", staticImport, transport.Behaviors)
	}

	var decoded Program
	if err := json.Unmarshal(ir, &decoded); err != nil {
		t.Fatal(err)
	}
	staticImport = nil
	for index := range decoded.Behaviors {
		for importIndex := range decoded.Behaviors[index].Imports {
			imp := &decoded.Behaviors[index].Imports[importIndex]
			if imp.Path == "com.example.Format.record" {
				staticImport = imp
			}
		}
	}
	if staticImport == nil || !staticImport.Static || staticImport.Language != LanguageJava {
		t.Fatalf("serialized json-ir static import = %#v, behaviors = %#v", staticImport, decoded.Behaviors)
	}

	output, _, err := compiler.Compile(context.Background(), SourceInput{Path: "DoorHsm.hsm.json", Data: ir}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageJava,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, "import static com.example.Format.record;") {
		t.Fatalf("recompiled Java output missing static import:\n%s", text)
	}
	if strings.Contains(text, "import com.example.Format.record;") {
		t.Fatalf("recompiled Java output rendered static import as ordinary import:\n%s", text)
	}
}

func TestJSONIRPreservesDartDeferredImportsAcrossRecompile(t *testing.T) {
	compiler := NewCompiler()
	ir, transport, err := compiler.Compile(context.Background(), SourceInput{Path: "DoorHsm.dart", Data: []byte(dartDeferredImportSource)}, CompileOptions{
		From:    LanguageDart,
		To:      LanguageJSONIR,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	var deferredImport *Import
	for index := range transport.Behaviors {
		for importIndex := range transport.Behaviors[index].Imports {
			imp := &transport.Behaviors[index].Imports[importIndex]
			if imp.Path == "package:sample/lazy_format.dart" {
				deferredImport = imp
			}
		}
	}
	if deferredImport == nil || !deferredImport.Deferred || deferredImport.Alias != "lazyFormat" || deferredImport.Language != LanguageDart {
		t.Fatalf("compiled json-ir program deferred import = %#v, behaviors = %#v", deferredImport, transport.Behaviors)
	}

	var decoded Program
	if err := json.Unmarshal(ir, &decoded); err != nil {
		t.Fatal(err)
	}
	deferredImport = nil
	for index := range decoded.Behaviors {
		for importIndex := range decoded.Behaviors[index].Imports {
			imp := &decoded.Behaviors[index].Imports[importIndex]
			if imp.Path == "package:sample/lazy_format.dart" {
				deferredImport = imp
			}
		}
	}
	if deferredImport == nil || !deferredImport.Deferred || deferredImport.Alias != "lazyFormat" || deferredImport.Language != LanguageDart {
		t.Fatalf("serialized json-ir deferred import = %#v, behaviors = %#v", deferredImport, decoded.Behaviors)
	}

	output, _, err := compiler.Compile(context.Background(), SourceInput{Path: "DoorHsm.hsm.json", Data: ir}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageDart,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, `import "package:sample/lazy_format.dart" deferred as lazyFormat;`) {
		t.Fatalf("recompiled Dart output missing deferred import:\n%s", text)
	}
	if strings.Contains(text, `import "package:sample/lazy_format.dart" as lazyFormat;`) {
		t.Fatalf("recompiled Dart output rendered deferred import as eager prefix import:\n%s", text)
	}
}

func TestJSONIRPreservesTypeScriptTypeOnlyImportsAcrossRecompile(t *testing.T) {
	compiler := NewCompiler()
	ir, transport, err := compiler.Compile(context.Background(), SourceInput{Path: "DoorHsm.ts", Data: []byte(tsTypeOnlyImportSource)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguageJSONIR,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	var typeImport *Import
	for index := range transport.Globals {
		for importIndex := range transport.Globals[index].Imports {
			imp := &transport.Globals[index].Imports[importIndex]
			if imp.Path == "./types" {
				typeImport = imp
			}
		}
	}
	if typeImport == nil || !typeImport.TypeOnly || typeImport.Language != LanguageTS {
		t.Fatalf("compiled json-ir program type-only import = %#v, globals = %#v", typeImport, transport.Globals)
	}

	var decoded Program
	if err := json.Unmarshal(ir, &decoded); err != nil {
		t.Fatal(err)
	}
	typeImport = nil
	for index := range decoded.Globals {
		for importIndex := range decoded.Globals[index].Imports {
			imp := &decoded.Globals[index].Imports[importIndex]
			if imp.Path == "./types" {
				typeImport = imp
			}
		}
	}
	if typeImport == nil || !typeImport.TypeOnly || typeImport.Language != LanguageTS {
		t.Fatalf("serialized json-ir type-only import = %#v, globals = %#v", typeImport, decoded.Globals)
	}

	output, _, err := compiler.Compile(context.Background(), SourceInput{Path: "DoorHsm.hsm.json", Data: ir}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, `import type { DoorSnapshot, DoorStatus as Status } from "./types";`) {
		t.Fatalf("recompiled TypeScript output missing type-only import:\n%s", text)
	}
	if !strings.Contains(text, `import { type DoorMetrics, recordMetric } from "./metrics";`) {
		t.Fatalf("recompiled TypeScript output missing mixed type-only specifier import:\n%s", text)
	}
	if strings.Contains(text, `import { DoorSnapshot, DoorStatus as Status } from "./types";`) {
		t.Fatalf("recompiled TypeScript output rendered type-only import as value import:\n%s", text)
	}
}
