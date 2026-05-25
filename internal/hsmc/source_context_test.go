package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestNormalizeSourceContextDefaultsMissingMutableLanguages(t *testing.T) {
	program := &Program{
		SourceLanguage: LanguageGo,
		Imports:        []Import{{Path: "fmt"}},
		Globals: []CodeBlock{{
			ID:      "global_1",
			Code:    "func helper() {}",
			Imports: []Import{{Path: "strings"}},
		}},
		Behaviors: []Behavior{{
			ID:      "entry_1",
			OwnerID: "state_1",
			Kind:    BehaviorEntry,
			Body:    "fmt.Println(\"closed\")",
			Imports: []Import{{Path: "context"}},
		}},
	}

	NormalizeSourceContext(program, "")

	if program.Imports[0].Language != LanguageGo {
		t.Fatalf("program import language = %q, want go", program.Imports[0].Language)
	}
	if program.Globals[0].Language != LanguageGo || program.Globals[0].Imports[0].Language != LanguageGo {
		t.Fatalf("global source context not normalized: %#v", program.Globals[0])
	}
	if program.Behaviors[0].SourceLanguage != LanguageGo || program.Behaviors[0].Imports[0].Language != LanguageGo {
		t.Fatalf("behavior source context not normalized: %#v", program.Behaviors[0])
	}
}

func TestNormalizeSourceContextDefaultsMutableImportsToRegionLanguage(t *testing.T) {
	program := &Program{
		SourceLanguage: LanguageGo,
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageTS,
			Code:     "export const helper = normalize('closed');",
			Imports:  []Import{{Path: "./normalize"}},
		}},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguagePython,
			Body:           "record('closed')",
			Imports:        []Import{{Path: "formatting"}},
		}},
	}

	NormalizeSourceContext(program, "")

	if program.Globals[0].Imports[0].Language != LanguageTS {
		t.Fatalf("global import language = %q, want region language typescript: %#v", program.Globals[0].Imports[0].Language, program.Globals[0])
	}
	if program.Behaviors[0].Imports[0].Language != LanguagePython {
		t.Fatalf("behavior import language = %q, want region language python: %#v", program.Behaviors[0].Imports[0].Language, program.Behaviors[0])
	}
}

func TestNormalizeSourceContextDefaultsCompilerOwnedModelLanguages(t *testing.T) {
	program := &Program{
		SourceLanguage: LanguageTS,
		Events:         []EventDecl{{ID: "event_1", Symbol: "goEvent", Name: "go", Schema: "{ kind: string }"}},
		Models: []Model{{
			ID:         "model_1",
			Name:       "Door",
			Attributes: []Attribute{{ID: "attribute_1", Name: "count", Default: "0", HasDefault: true}},
			Transitions: []Transition{{
				ID: "transition_1",
				Trigger: &Trigger{
					Kind:   TriggerOn,
					Value:  "goEvent",
					Values: []TriggerValue{{Value: "goEvent", EventSymbol: "goEvent"}},
				},
			}},
			States: []State{{
				ID:   "state_1",
				Name: "closed",
				Kind: StateKindState,
				Transitions: []Transition{{
					ID:      "transition_2",
					Trigger: &Trigger{Kind: TriggerAfter, Value: "1000"},
				}},
			}},
		}},
	}

	NormalizeSourceContext(program, "")

	if program.Events[0].Language != LanguageTS {
		t.Fatalf("event language = %q, want typescript", program.Events[0].Language)
	}
	if program.Models[0].Attributes[0].Language != LanguageTS {
		t.Fatalf("attribute language = %q, want typescript", program.Models[0].Attributes[0].Language)
	}
	if program.Models[0].Transitions[0].Trigger.Language != LanguageTS || program.Models[0].Transitions[0].Trigger.Values[0].Language != LanguageTS {
		t.Fatalf("transition trigger not normalized: %#v", program.Models[0].Transitions[0].Trigger)
	}
	if program.Models[0].States[0].Transitions[0].Trigger.Language != LanguageTS {
		t.Fatalf("state transition trigger not normalized: %#v", program.Models[0].States[0].Transitions[0].Trigger)
	}
}

func TestNormalizeSourceContextUsesFallbackSourceLanguage(t *testing.T) {
	program := &Program{
		Imports: []Import{{Path: "fmt"}},
		Behaviors: []Behavior{{
			ID:      "entry_1",
			OwnerID: "state_1",
			Kind:    BehaviorEntry,
		}},
	}

	NormalizeSourceContext(program, LanguageGo)

	if program.SourceLanguage != LanguageGo {
		t.Fatalf("source language = %q, want fallback go", program.SourceLanguage)
	}
	if program.Imports[0].Language != LanguageGo || program.Behaviors[0].SourceLanguage != LanguageGo {
		t.Fatalf("fallback source context not applied: %#v / %#v", program.Imports, program.Behaviors)
	}
}

func TestCompileNormalizesLanguageLessJSONIRMutableContextBeforeAdapterRequest(t *testing.T) {
	adapter := &capturingAdapter{}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)
	source := `{
  "source_language": "go",
  "imports": [{ "path": "os" }],
  "globals": [{
    "id": "global_1",
    "code": "func helper() string { return strings.TrimSpace(\"closed\") }",
    "imports": [{ "path": "strings" }]
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
    "body": "fmt.Println(helper())",
    "imports": [{ "path": "fmt" }]
  }]
}`

	_, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageTS,
		Adapter: adapter.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(adapter.request.Imports) != 1 || adapter.request.Imports[0].Language != LanguageGo {
		t.Fatalf("adapter imports = %#v, want source language normalized", adapter.request.Imports)
	}
	if len(adapter.request.Globals) != 1 || adapter.request.Globals[0].Language != LanguageGo {
		t.Fatalf("adapter globals = %#v, want source language normalized", adapter.request.Globals)
	}
	if len(adapter.request.Globals[0].Imports) != 1 || adapter.request.Globals[0].Imports[0].Language != LanguageGo {
		t.Fatalf("adapter global imports = %#v, want source language normalized", adapter.request.Globals[0].Imports)
	}
	if len(adapter.request.Behaviors) != 1 || adapter.request.Behaviors[0].SourceLanguage != LanguageGo {
		t.Fatalf("adapter behaviors = %#v, want source language normalized", adapter.request.Behaviors)
	}
	if len(adapter.request.Behaviors[0].Imports) != 1 || adapter.request.Behaviors[0].Imports[0].Language != LanguageGo {
		t.Fatalf("adapter behavior imports = %#v, want source language normalized", adapter.request.Behaviors[0].Imports)
	}
}

func TestCompileNormalizesLanguageLessJSONIRRegionImportsToRegionLanguage(t *testing.T) {
	adapter := &capturingAdapter{}
	compiler := NewCompiler()
	compiler.RegisterAdapter(adapter)
	source := `{
  "source_language": "go",
  "globals": [{
    "id": "global_1",
    "language": "typescript",
    "code": "export const helper = normalize('closed');",
    "imports": [{ "path": "./normalize" }]
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
    "source_language": "python",
    "body": "record('closed')",
    "imports": [{ "path": "formatting" }]
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
	if len(adapter.request.Globals) != 1 || len(adapter.request.Globals[0].Imports) != 1 || adapter.request.Globals[0].Imports[0].Language != LanguageTS {
		t.Fatalf("adapter global imports = %#v, want region language typescript", adapter.request.Globals)
	}
	if len(adapter.request.Behaviors) != 1 || len(adapter.request.Behaviors[0].Imports) != 1 || adapter.request.Behaviors[0].Imports[0].Language != LanguagePython {
		t.Fatalf("adapter behavior imports = %#v, want region language python", adapter.request.Behaviors)
	}
}

func TestCompileRejectsJSONIRGlobalImportLanguageMismatch(t *testing.T) {
	compiler := NewCompiler()
	source := `{
  "source_language": "go",
  "globals": [{
    "id": "global_1",
    "language": "go",
    "code": "func helper() string { return format(\"closed\") }",
    "imports": [{ "path": "./format", "language": "typescript", "specifiers": [{ "name": "format" }] }]
  }],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "states": [{ "id": "state_1", "name": "closed", "kind": "state" }]
  }]
}`

	_, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err == nil || !strings.Contains(err.Error(), `global "global_1" import "./format" language "typescript" does not match region language "go"`) {
		t.Fatalf("Compile() error = %v, want global region import language mismatch", err)
	}
}

func TestCompileRejectsJSONIRBehaviorImportLanguageMismatch(t *testing.T) {
	compiler := NewCompiler()
	source := `{
  "source_language": "go",
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
    "body": "format(\"closed\")",
    "imports": [{ "path": "./format", "language": "typescript", "specifiers": [{ "name": "format" }] }]
  }]
}`

	_, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err == nil || !strings.Contains(err.Error(), `behavior "entry_1" import "./format" language "typescript" does not match region language "go"`) {
		t.Fatalf("Compile() error = %v, want behavior region import language mismatch", err)
	}
}

func TestCompileRejectsImplementationFrontendSourceLanguageMismatch(t *testing.T) {
	compiler := NewCompiler()
	compiler.RegisterFrontend(fakeFrontend{
		language: LanguageGo,
		program: &Program{
			SourceLanguage: LanguagePython,
			Models: []Model{{
				ID:          "model_1",
				Name:        "Door",
				Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
				States:      []State{{ID: "state_1", Name: "closed", Kind: StateKindState}},
			}},
		},
	})

	_, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte("package sample")}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err == nil || !strings.Contains(err.Error(), `frontend "go" returned source_language "python"`) {
		t.Fatalf("Compile() error = %v, want frontend source language mismatch", err)
	}
}

func TestJSONIRFrontendRequiresImplementationSourceLanguage(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "missing",
			source: `{"models":[]}`,
			want:   "source_language is required",
		},
		{
			name:   "transport language",
			source: `{"source_language":"json-ir","models":[]}`,
			want:   "must be an implementation language",
		},
		{
			name:   "unsupported",
			source: `{"source_language":"ruby","models":[]}`,
			want:   `source_language "ruby" is not supported`,
		},
	}
	frontend := NewJSONIRFrontend()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := frontend.Parse(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(tc.source)})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestJSONIRFrontendRejectsUnknownFields(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "top level",
			source: `{"source_language":"go","models":[],"modelz":[]}`,
			want:   `unknown field "modelz"`,
		},
		{
			name:   "nested behavior",
			source: `{"source_language":"go","models":[],"behaviors":[{"id":"entry_1","owner_id":"state_1","kind":"entry","unexpected":"value"}]}`,
			want:   `unknown field "unexpected"`,
		},
		{
			name:   "nested model",
			source: `{"source_language":"go","models":[{"id":"model_1","name":"Door","states":[{"id":"state_1","name":"closed","kind":"state","transitons":[]}]}]}`,
			want:   `unknown field "transitons"`,
		},
	}
	frontend := NewJSONIRFrontend()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := frontend.Parse(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(tc.source)})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestJSONIRFrontendRejectsDuplicateFields(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "top level",
			source: `{"source_language":"go","source_language":"typescript","models":[]}`,
			want:   `json-ir contains duplicate field "source_language"`,
		},
		{
			name:   "nested behavior",
			source: `{"source_language":"go","models":[],"behaviors":[{"id":"entry_1","id":"entry_2","owner_id":"state_1","kind":"entry"}]}`,
			want:   `json-ir.behaviors[0] contains duplicate field "id"`,
		},
		{
			name:   "nested trigger value",
			source: `{"source_language":"go","models":[{"id":"model_1","name":"Door","states":[{"id":"state_1","name":"closed","kind":"state","transitions":[{"id":"transition_1","owner_id":"state_1","trigger":{"kind":"on","value":"go","values":[{"value":"go","value":"again"}]}}]}]}]}`,
			want:   `json-ir.models[0].states[0].transitions[0].trigger.values[0] contains duplicate field "value"`,
		},
	}
	frontend := NewJSONIRFrontend()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := frontend.Parse(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(tc.source)})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestJSONIRFrontendRejectsTrailingJSONValues(t *testing.T) {
	_, err := NewJSONIRFrontend().Parse(context.Background(), SourceInput{
		Path: "door.hsm.json",
		Data: []byte(`{"source_language":"go","models":[]} {"source_language":"go","models":[]}`),
	})
	if err == nil || !strings.Contains(err.Error(), "multiple JSON values") {
		t.Fatalf("error = %v, want multiple JSON values rejection", err)
	}
}

func TestCompileRejectsInvalidJSONIRBeforeBackend(t *testing.T) {
	source := `{
  "source_language": "go",
  "events": [
    { "id": "event_1", "symbol": "goEvent", "name": "go" },
    { "id": "event_1", "symbol": "againEvent", "name": "again" }
  ],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "states": [
      { "id": "state_1", "name": "closed", "kind": "state" }
    ]
  }]
}`
	_, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err == nil || !strings.Contains(err.Error(), `duplicate event id "event_1"`) {
		t.Fatalf("Compile() error = %v, want duplicate event id validation", err)
	}
}

func TestJSONIRFrontendCanonicalizesSourceLanguageAliases(t *testing.T) {
	program, err := NewJSONIRFrontend().Parse(context.Background(), SourceInput{
		Path: "door.hsm.json",
		Data: []byte(`{"source_language":"ts","models":[]}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if program.SourceLanguage != LanguageTS {
		t.Fatalf("source language = %q, want typescript", program.SourceLanguage)
	}
}

func TestJSONIRFrontendCanonicalizesNestedLanguageAliases(t *testing.T) {
	source := `{
  "source_language": "ts",
  "imports": [{ "path": "fmt", "language": "golang" }],
  "globals": [{
    "id": "global_1",
    "language": "py",
    "code": "def helper(): pass",
    "imports": [{ "path": "sys", "language": "python" }]
  }],
  "events": [{ "id": "event_1", "symbol": "goEvent", "name": "go", "language": "js" }],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "attributes": [{ "id": "attribute_1", "name": "count", "language": "c++" }],
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "transitions": [{
      "owner_id": "model_1",
      "id": "transition_1",
      "trigger": {
        "kind": "on",
        "value": "goEvent",
        "language": "cs",
        "values": [{ "value": "goEvent", "event_symbol": "goEvent", "language": "rs" }]
      }
    }],
    "states": [{ "id": "state_1", "name": "closed", "kind": "state" }]
  }],
  "behaviors": [{
    "id": "entry_1",
    "owner_id": "state_1",
    "kind": "entry",
    "source_language": "zig",
    "target_language": "ts",
    "target_abi": { "language": "py", "signature": "def entry(ctx, instance, event):" },
    "imports": [{ "path": "std", "language": "zig" }]
  }]
}`

	program, err := NewJSONIRFrontend().Parse(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	if program.SourceLanguage != LanguageTS ||
		program.Imports[0].Language != LanguageGo ||
		program.Globals[0].Language != LanguagePython ||
		program.Globals[0].Imports[0].Language != LanguagePython ||
		program.Events[0].Language != LanguageJS ||
		program.Models[0].Attributes[0].Language != LanguageCPP ||
		program.Models[0].Transitions[0].Trigger.Language != LanguageCSharp ||
		program.Models[0].Transitions[0].Trigger.Values[0].Language != LanguageRust ||
		program.Behaviors[0].SourceLanguage != LanguageZig ||
		program.Behaviors[0].Imports[0].Language != LanguageZig {
		t.Fatalf("nested language aliases not canonicalized: %#v", program)
	}
	if program.Behaviors[0].TargetLanguage != "" || program.Behaviors[0].TargetABI != nil {
		t.Fatalf("json-ir frontend preserved stale target behavior metadata: %#v", program.Behaviors[0])
	}
}

func TestCompileRetargetsJSONIRBehaviorMetadata(t *testing.T) {
	source := `{
  "source_language": "go",
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
    "target_language": "ruby",
    "target_abi": {
      "language": "ruby",
      "signature": "async def(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None"
    },
    "body": "fmt.Println(\"closed\")"
  }]
}`

	_, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Behaviors) != 1 {
		t.Fatalf("behaviors = %#v", program.Behaviors)
	}
	behavior := program.Behaviors[0]
	if behavior.TargetLanguage != "" {
		t.Fatalf("json-ir stale target language survived retargeting: %#v", behavior)
	}
	if behavior.TargetABI == nil || behavior.TargetABI.Language != LanguageTS {
		t.Fatalf("behavior target ABI = %#v, want TypeScript", behavior.TargetABI)
	}
}

func TestJSONIRCompileRejectsInlineOperationImplementation(t *testing.T) {
	source := `{
  "source_language": "typescript",
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "operations": [{
      "id": "operation_decl_1",
      "name": "approve",
      "behavior": { "id": "operation_1", "kind": "operation" }
    }],
    "states": [{ "id": "state_1", "name": "closed", "kind": "state" }]
  }],
  "behaviors": [{
    "id": "operation_1",
    "owner_id": "model_1",
    "kind": "operation",
    "inline": true,
    "source_language": "typescript",
    "body": "return true"
  }]
}`

	_, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err == nil || !strings.Contains(err.Error(), `operation "approve" implementation must be a callable reference, not an inline anonymous callable`) {
		t.Fatalf("expected inline operation diagnostic, got %v", err)
	}
}

func TestJSONIRFrontendRejectsInvalidNestedLanguages(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "unsupported source context",
			source: `{"source_language":"go","imports":[{"path":"fmt","language":"ruby"}],"models":[]}`,
			want:   `imports[0].language "ruby" is not supported`,
		},
		{
			name:   "transport source context",
			source: `{"source_language":"go","globals":[{"id":"global_1","language":"json-ir","code":"{}"}],"models":[]}`,
			want:   `globals[0].language must be an implementation language`,
		},
		{
			name:   "unsupported behavior source",
			source: `{"source_language":"go","models":[],"behaviors":[{"id":"entry_1","owner_id":"state_1","kind":"entry","source_language":"ruby"}]}`,
			want:   `behaviors[0].source_language "ruby" is not supported`,
		},
		{
			name:   "unsupported behavior import source",
			source: `{"source_language":"go","models":[],"behaviors":[{"id":"entry_1","owner_id":"state_1","kind":"entry","source_language":"go","imports":[{"path":"helper","language":"ruby"}]}]}`,
			want:   `behaviors[0].imports[0].language "ruby" is not supported`,
		},
	}
	frontend := NewJSONIRFrontend()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := frontend.Parse(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(tc.source)})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestJSONIRFrontendClearsUnsupportedStaleTargetBehaviorMetadata(t *testing.T) {
	source := `{
  "source_language": "go",
  "models": [],
  "behaviors": [{
    "id": "entry_1",
    "owner_id": "state_1",
    "kind": "entry",
    "source_language": "go",
    "inline": true,
    "signature": "def stale(ctx, instance, event):",
    "target_language": "ruby",
    "target_abi": { "language": "ruby", "signature": "def stale(): pass" }
  }]
}`

	program, err := NewJSONIRFrontend().Parse(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Behaviors) != 1 {
		t.Fatalf("behaviors = %#v", program.Behaviors)
	}
	if program.Behaviors[0].TargetLanguage != "" || program.Behaviors[0].TargetABI != nil {
		t.Fatalf("json-ir frontend preserved stale target behavior metadata: %#v", program.Behaviors[0])
	}
	if program.Behaviors[0].Inline || program.Behaviors[0].Signature != "" {
		t.Fatalf("json-ir frontend preserved stale target extraction metadata: %#v", program.Behaviors[0])
	}
}

func TestValidateRejectsInvalidLanguageMetadata(t *testing.T) {
	cases := []struct {
		name    string
		program *Program
		want    string
	}{
		{
			name:    "unsupported import language",
			program: &Program{Imports: []Import{{Path: "fmt", Language: Language("ruby")}}},
			want:    `imports[0].language "ruby" is not supported`,
		},
		{
			name:    "empty import path",
			program: &Program{Imports: []Import{{Path: " \t", Language: LanguageTS}}},
			want:    `imports[0] path is empty`,
		},
		{
			name:    "missing import specifier name",
			program: &Program{Imports: []Import{{Path: "./helpers", Language: LanguageTS, Specifiers: []ImportSpecifier{{Alias: "helper"}}}}},
			want:    `imports[0] import "./helpers" contains specifier with empty name`,
		},
		{
			name:    "hidden import specifier unsupported by source language",
			program: &Program{Imports: []Import{{Path: "./helpers", Language: LanguageTS, Hidden: []string{"debugRecord"}}}},
			want:    `imports[0] import "./helpers" uses hidden specifiers unsupported by typescript`,
		},
		{
			name:    "empty import local name",
			program: &Program{Imports: []Import{{Path: "./helpers", Language: LanguageTS, LocalNames: []string{" \t"}}}},
			want:    `imports[0] import "./helpers" contains empty local name`,
		},
		{
			name:    "import local name surrounding whitespace",
			program: &Program{Imports: []Import{{Path: "./helpers", Language: LanguageTS, LocalNames: []string{" helper "}}}},
			want:    `imports[0] import "./helpers" local name " helper " contains surrounding whitespace`,
		},
		{
			name:    "duplicate import local name",
			program: &Program{Imports: []Import{{Path: "./helpers", Language: LanguageTS, LocalNames: []string{"helper", "helper"}}}},
			want:    `imports[0] import "./helpers" contains duplicate local binding "helper"`,
		},
		{
			name:    "source import shape rejected",
			program: &Program{Imports: []Import{{Path: "fmt", Language: LanguageGo, Specifiers: []ImportSpecifier{{Name: "Sprintf"}}}}},
			want:    `imports[0] import "fmt" uses default or named specifiers unsupported by Go`,
		},
		{
			name: "behavior import shape rejected",
			program: validProgramWithBehaviorLanguage(func(behavior *Behavior) {
				behavior.Imports = []Import{{Path: "helpers", Language: LanguagePython, Default: "helpers"}}
			}),
			want: `behaviors[0].imports[0] import "helpers" uses default import unsupported by Python`,
		},
		{
			name:    "transport global language",
			program: &Program{Globals: []CodeBlock{{ID: "global_1", Language: LanguageJSONIR}}},
			want:    `globals[0].language must be an implementation language`,
		},
		{
			name: "non-canonical event language",
			program: &Program{
				Events: []EventDecl{{ID: "event_1", Symbol: "goEvent", Name: "go", Language: Language("ts")}},
			},
			want: `events[0].language "ts" must be canonical "typescript"`,
		},
		{
			name:    "unsupported behavior target language",
			program: validProgramWithBehaviorLanguage(func(behavior *Behavior) { behavior.TargetLanguage = Language("ruby") }),
			want:    `behaviors[0].target_language "ruby" is not supported`,
		},
		{
			name:    "target language without target code",
			program: validProgramWithBehaviorLanguage(func(behavior *Behavior) { behavior.TargetLanguage = LanguageTS }),
			want:    `behavior "entry_1" target_language "typescript" requires target body or code`,
		},
		{
			name: "unsupported target abi language",
			program: validProgramWithBehaviorLanguage(func(behavior *Behavior) {
				behavior.TargetABI = &BehaviorABI{Language: Language("ruby"), Signature: "entry"}
			}),
			want: `behaviors[0].target_abi.language "ruby" is not supported`,
		},
		{
			name: "empty target abi language",
			program: validProgramWithBehaviorLanguage(func(behavior *Behavior) {
				behavior.TargetABI = &BehaviorABI{Signature: "entry"}
			}),
			want: `behavior "entry_1" target_abi.language is empty`,
		},
		{
			name: "empty target abi signature",
			program: validProgramWithBehaviorLanguage(func(behavior *Behavior) {
				behavior.TargetABI = &BehaviorABI{Language: LanguageTS, Signature: " \t"}
			}),
			want: `behavior "entry_1" target_abi.signature is empty`,
		},
		{
			name: "padded target abi signature",
			program: validProgramWithBehaviorLanguage(func(behavior *Behavior) {
				behavior.TargetABI = &BehaviorABI{Language: LanguageTS, Signature: " entry "}
			}),
			want: `behavior "entry_1" target_abi.signature has leading or trailing whitespace`,
		},
		{
			name: "empty target abi return type",
			program: validProgramWithBehaviorLanguage(func(behavior *Behavior) {
				behavior.TargetABI = &BehaviorABI{Language: LanguageTS, Signature: "entry", ReturnType: " \t"}
			}),
			want: `behavior "entry_1" target_abi.return_type is empty`,
		},
		{
			name: "padded target abi return type",
			program: validProgramWithBehaviorLanguage(func(behavior *Behavior) {
				behavior.TargetABI = &BehaviorABI{Language: LanguageTS, Signature: "entry", ReturnType: " boolean "}
			}),
			want: `behavior "entry_1" target_abi.return_type " boolean " has leading or trailing whitespace`,
		},
		{
			name: "empty target abi parameter name",
			program: validProgramWithBehaviorLanguage(func(behavior *Behavior) {
				behavior.TargetABI = &BehaviorABI{Language: LanguageTS, Signature: "entry", Parameters: []ABIParameter{{Name: " \t", Type: "hsm.Context"}}}
			}),
			want: `behavior "entry_1" target_abi.parameters[0] name is empty`,
		},
		{
			name: "padded target abi parameter type",
			program: validProgramWithBehaviorLanguage(func(behavior *Behavior) {
				behavior.TargetABI = &BehaviorABI{Language: LanguageTS, Signature: "entry", Parameters: []ABIParameter{{Name: "ctx", Type: " hsm.Context "}}}
			}),
			want: `behavior "entry_1" target_abi.parameters[0] type " hsm.Context " has leading or trailing whitespace`,
		},
		{
			name: "duplicate target abi parameter name",
			program: validProgramWithBehaviorLanguage(func(behavior *Behavior) {
				behavior.TargetABI = &BehaviorABI{Language: LanguageTS, Signature: "entry", Parameters: []ABIParameter{{Name: "ctx", Type: "hsm.Context"}, {Name: "ctx", Type: "hsm.Context"}}}
			}),
			want: `behavior "entry_1" target_abi contains duplicate parameter "ctx"`,
		},
		{
			name: "target abi language mismatch",
			program: validProgramWithBehaviorLanguage(func(behavior *Behavior) {
				behavior.TargetLanguage = LanguageTS
				behavior.TargetABI = &BehaviorABI{Language: LanguagePython, Signature: "entry"}
			}),
			want: `behavior "entry_1" target_abi.language "python" does not match target_language "typescript"`,
		},
		{
			name: "transport trigger source language",
			program: &Program{Models: []Model{{
				ID:          "model_1",
				Name:        "Door",
				Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
				States:      []State{{ID: "state_1", Name: "closed", Kind: StateKindState}},
				Transitions: []Transition{{
					OwnerID: "model_1",
					ID:      "transition_1",
					Trigger: &Trigger{Kind: TriggerAfter, Value: "1000", Language: LanguageJSONIR},
				}},
			}}},
			want: `models[0].transitions[0].trigger.language must be an implementation language`,
		},
		{
			name: "on trigger expression behavior",
			program: &Program{
				SourceLanguage: LanguageGo,
				Models: []Model{{
					ID:          "model_1",
					Name:        "Door",
					Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
					States: []State{{
						ID:   "state_1",
						Name: "closed",
						Kind: StateKindState,
						Transitions: []Transition{{
							OwnerID: "state_1",
							ID:      "transition_1",
							Trigger: &Trigger{Kind: TriggerOn, Value: "open", IsString: true, Expr: &BehaviorRef{ID: "trigger_1", Kind: BehaviorTrigger}},
						}},
					}},
				}},
				Behaviors: []Behavior{{
					ID:             "trigger_1",
					OwnerID:        "state_1",
					Kind:           BehaviorTrigger,
					TriggerKind:    TriggerOn,
					SourceLanguage: LanguageGo,
					Body:           "return true",
				}},
			},
			want: `on trigger must not use a behavior expression`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.program)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() error = %v, want %q", err, tc.want)
			}
		})
	}
}

type fakeFrontend struct {
	language Language
	program  *Program
}

func (frontend fakeFrontend) Language() Language {
	return frontend.language
}

func (frontend fakeFrontend) Parse(ctx context.Context, input SourceInput) (*Program, error) {
	return frontend.program, nil
}

func validProgramWithBehaviorLanguage(mutator func(*Behavior)) *Program {
	behavior := Behavior{
		ID:             "entry_1",
		OwnerID:        "state_1",
		Kind:           BehaviorEntry,
		SourceLanguage: LanguageGo,
	}
	mutator(&behavior)
	return &Program{
		Models: []Model{{
			ID:          "model_1",
			Name:        "Door",
			Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
			States: []State{{
				ID:        "state_1",
				Name:      "closed",
				Kind:      StateKindState,
				Behaviors: []BehaviorRef{{ID: "entry_1", Kind: BehaviorEntry}},
			}},
		}},
		Behaviors: []Behavior{behavior},
	}
}
