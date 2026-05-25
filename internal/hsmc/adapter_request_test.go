package hsmc

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"testing"
)

func TestBuildAdapterRequestDeepCopiesMutableContext(t *testing.T) {
	program := &Program{
		SourceLanguage: LanguageGo,
		PackageName:    "sample",
		Imports: []Import{{
			Path:       "fmt",
			Specifiers: []ImportSpecifier{{Name: "Println", Alias: "printLine"}},
			Hidden:     []string{"Sprintf"},
			LocalNames: []string{"fmt"},
			Language:   LanguageGo,
		}},
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageGo,
			Code:     "func helper() {}",
			Imports: []Import{{
				Path:       "strings",
				Specifiers: []ImportSpecifier{{Name: "Builder"}},
				Hidden:     []string{"Reader"},
				LocalNames: []string{"strings"},
				Language:   LanguageGo,
			}},
		}},
		Events: []EventDecl{{ID: "event_1", Symbol: "goEvent", Name: "go"}},
		Models: []Model{{
			ID:          "model_1",
			Name:        "Door",
			Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed", Effects: []BehaviorRef{{ID: "init_effect_1", Kind: BehaviorEffect}}},
			Operations:  []Operation{{ID: "operation_1", Name: "approve", Behavior: &BehaviorRef{ID: "operation_1", Kind: BehaviorOperation}}},
			Transitions: []Transition{{
				ID:      "transition_0",
				OwnerID: "model_1",
				Trigger: &Trigger{
					Kind: TriggerWhen,
					Expr: &BehaviorRef{ID: "model_trigger_1", Kind: BehaviorTrigger},
				},
				Effects: []BehaviorRef{{ID: "model_effect_1", Kind: BehaviorEffect}},
			}},
			States: []State{
				{
					ID:        "state_1",
					Name:      "closed",
					Kind:      StateKindState,
					Behaviors: []BehaviorRef{{ID: "entry_1", Kind: BehaviorEntry}},
					Defers:    []string{"knock"},
					States: []State{{
						ID:        "state_1_child",
						Name:      "child",
						Kind:      StateKindState,
						Behaviors: []BehaviorRef{{ID: "child_entry_1", Kind: BehaviorEntry}},
						Transitions: []Transition{{
							ID:      "transition_child",
							OwnerID: "state_1_child",
							Target:  "../../open",
							Trigger: &Trigger{
								Kind: TriggerWhen,
								Expr: &BehaviorRef{ID: "child_trigger_1", Kind: BehaviorTrigger},
							},
						}},
					}},
					Transitions: []Transition{{
						ID:      "transition_1",
						OwnerID: "state_1",
						Target:  "../open",
						Trigger: &Trigger{
							Kind:   TriggerOn,
							Value:  "goEvent",
							Values: []TriggerValue{{Value: "goEvent", EventSymbol: "goEvent"}},
						},
					}},
				},
				{ID: "state_2", Name: "open", Kind: StateKindState},
			},
		}},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			Imports: []Import{{
				Path:       "context",
				Specifiers: []ImportSpecifier{{Name: "Context", Alias: "Ctx"}},
				Hidden:     []string{"Canceled"},
				LocalNames: []string{"context"},
				Language:   LanguageGo,
			}},
			TargetABI: &BehaviorABI{
				Language:   LanguageTS,
				Signature:  "entry(ctx: hsm.Context): void",
				ReturnType: "void",
				Parameters: []ABIParameter{{Name: "ctx", Type: "hsm.Context"}},
			},
		}},
	}

	request := buildAdapterRequest(program, LanguageTS)
	request.Imports[0].LocalNames[0] = "mutated"
	request.Imports[0].Specifiers[0].Name = "Mutated"
	request.Imports[0].Specifiers[0].Alias = "mutated"
	request.Imports[0].Hidden[0] = "Mutated"
	request.Globals[0].Imports[0].Path = "mutated-global"
	request.Globals[0].Imports[0].LocalNames[0] = "mutated"
	request.Globals[0].Imports[0].Specifiers[0].Name = "Mutated"
	request.Globals[0].Imports[0].Hidden[0] = "Mutated"
	request.Behaviors[0].Imports[0].Path = "mutated-behavior"
	request.Behaviors[0].Imports[0].LocalNames[0] = "mutated"
	request.Behaviors[0].Imports[0].Specifiers[0].Name = "Mutated"
	request.Behaviors[0].Imports[0].Specifiers[0].Alias = "mutated"
	request.Behaviors[0].Imports[0].Hidden[0] = "Mutated"
	request.Behaviors[0].TargetABI.ReturnType = "boolean"
	request.Behaviors[0].TargetABI.Parameters[0].Type = "mutated"
	request.Events[0].Name = "mutated"
	request.Models[0].InitialEffects[0].ID = "mutated"
	request.Models[0].Operations[0].Behavior.ID = "mutated"
	request.Models[0].Transitions[0].Trigger.Expr.ID = "mutated"
	request.Models[0].Transitions[0].Effects[0].ID = "mutated"
	request.Models[0].States[0].Behaviors[0].ID = "mutated"
	request.Models[0].States[0].Defers[0] = "mutated"
	request.Models[0].States[0].States[0].Behaviors[0].ID = "mutated"
	request.Models[0].States[0].States[0].Transitions[0].Trigger.Expr.ID = "mutated"
	request.Models[0].States[0].Transitions[0].Trigger.Values[0].EventSymbol = "missingEvent"
	request.Models[0].States[0].Transitions[0].Guard = &BehaviorRef{ID: "mutated", Kind: BehaviorGuard}
	request.Models[0].States[0].Transitions[0].Effects = []BehaviorRef{{ID: "mutated", Kind: BehaviorEffect}}

	if program.Imports[0].LocalNames[0] != "fmt" {
		t.Fatalf("program imports mutated: %#v", program.Imports)
	}
	if program.Imports[0].Specifiers[0].Name != "Println" || program.Imports[0].Specifiers[0].Alias != "printLine" || program.Imports[0].Hidden[0] != "Sprintf" {
		t.Fatalf("program structured imports mutated: %#v", program.Imports)
	}
	if program.Globals[0].Imports[0].Path != "strings" || program.Globals[0].Imports[0].LocalNames[0] != "strings" {
		t.Fatalf("program global imports mutated: %#v", program.Globals[0].Imports)
	}
	if program.Globals[0].Imports[0].Specifiers[0].Name != "Builder" || program.Globals[0].Imports[0].Hidden[0] != "Reader" {
		t.Fatalf("program structured global imports mutated: %#v", program.Globals[0].Imports)
	}
	if program.Behaviors[0].Imports[0].Path != "context" || program.Behaviors[0].Imports[0].LocalNames[0] != "context" {
		t.Fatalf("program behavior imports mutated: %#v", program.Behaviors[0].Imports)
	}
	if program.Behaviors[0].Imports[0].Specifiers[0].Name != "Context" || program.Behaviors[0].Imports[0].Specifiers[0].Alias != "Ctx" || program.Behaviors[0].Imports[0].Hidden[0] != "Canceled" {
		t.Fatalf("program structured behavior imports mutated: %#v", program.Behaviors[0].Imports)
	}
	if program.Behaviors[0].TargetABI.ReturnType != "void" || program.Behaviors[0].TargetABI.Parameters[0].Type != "hsm.Context" {
		t.Fatalf("program behavior ABI mutated: %#v", program.Behaviors[0].TargetABI)
	}
	if program.Events[0].Name != "go" {
		t.Fatalf("program events mutated: %#v", program.Events)
	}
	if program.Models[0].Initializer.Effects[0].ID != "init_effect_1" {
		t.Fatalf("program initial effects mutated: %#v", program.Models[0].Initializer.Effects)
	}
	if program.Models[0].Operations[0].Behavior.ID != "operation_1" {
		t.Fatalf("program operation behavior mutated: %#v", program.Models[0].Operations)
	}
	if program.Models[0].Transitions[0].Trigger.Expr.ID != "model_trigger_1" {
		t.Fatalf("program model transition trigger expression mutated: %#v", program.Models[0].Transitions[0].Trigger.Expr)
	}
	if program.Models[0].Transitions[0].Effects[0].ID != "model_effect_1" {
		t.Fatalf("program model transition effects mutated: %#v", program.Models[0].Transitions[0].Effects)
	}
	if program.Models[0].States[0].Behaviors[0].ID != "entry_1" {
		t.Fatalf("program state behaviors mutated: %#v", program.Models[0].States[0].Behaviors)
	}
	if program.Models[0].States[0].Defers[0] != "knock" {
		t.Fatalf("program state defers mutated: %#v", program.Models[0].States[0].Defers)
	}
	if program.Models[0].States[0].States[0].Behaviors[0].ID != "child_entry_1" {
		t.Fatalf("program nested state behaviors mutated: %#v", program.Models[0].States[0].States[0].Behaviors)
	}
	if program.Models[0].States[0].States[0].Transitions[0].Trigger.Expr.ID != "child_trigger_1" {
		t.Fatalf("program nested transition trigger mutated: %#v", program.Models[0].States[0].States[0].Transitions[0].Trigger.Expr)
	}
	trigger := program.Models[0].States[0].Transitions[0].Trigger
	if trigger.Values[0].EventSymbol != "goEvent" {
		t.Fatalf("program trigger values mutated: %#v", trigger.Values)
	}
	if program.Models[0].States[0].Transitions[0].Guard != nil {
		t.Fatalf("program transition guard mutated: %#v", program.Models[0].States[0].Transitions[0].Guard)
	}
	if len(program.Models[0].States[0].Transitions[0].Effects) != 0 {
		t.Fatalf("program transition effects mutated: %#v", program.Models[0].States[0].Transitions[0].Effects)
	}
}

func TestNewAdapterProtocolDeepCopiesRequestContext(t *testing.T) {
	request := AdapterRequest{
		SourceLanguage: LanguageGo,
		TargetLanguage: LanguageTS,
		Imports: []Import{{
			Path:       "fmt",
			Specifiers: []ImportSpecifier{{Name: "Println", Alias: "printLine"}},
			Hidden:     []string{"Sprintf"},
			LocalNames: []string{"fmt"},
			Language:   LanguageGo,
		}},
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageGo,
			Code:     "func helper() {}",
			Imports: []Import{{
				Path:       "strings",
				Specifiers: []ImportSpecifier{{Name: "Builder"}},
				LocalNames: []string{"strings"},
				Language:   LanguageGo,
			}},
		}},
		Models: []ModelView{{
			ID:             "model_1",
			Name:           "Door",
			Path:           "/Door",
			InitialEffects: []BehaviorRef{{ID: "initial_effect_1", Kind: BehaviorEffect}},
			Operations:     []OperationView{{ID: "operation_1", Name: "approve", Behavior: &BehaviorRef{ID: "operation_behavior_1", Kind: BehaviorOperation}}},
			States: []StateView{{
				ID:        "state_1",
				Name:      "closed",
				Path:      "/Door/closed",
				Kind:      StateKindState,
				Behaviors: []BehaviorRef{{ID: "entry_1", Kind: BehaviorEntry}},
				Defers:    []string{"knock"},
				Transitions: []TransitionView{{
					ID:      "transition_1",
					OwnerID: "state_1",
					Trigger: &Trigger{
						Kind:   TriggerOn,
						Value:  "OpenEvent",
						Values: []TriggerValue{{Value: "OpenEvent", EventSymbol: "openEvent"}},
					},
					Effects: []BehaviorRef{{ID: "effect_1", Kind: BehaviorEffect}},
				}},
			}},
		}},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			Imports: []Import{{
				Path:       "context",
				Specifiers: []ImportSpecifier{{Name: "Context", Alias: "Ctx"}},
				LocalNames: []string{"context"},
				Language:   LanguageGo,
			}},
			TargetABI: &BehaviorABI{
				Language:   LanguageTS,
				Signature:  "entry(ctx: hsm.Context): void",
				ReturnType: "void",
				Parameters: []ABIParameter{{Name: "ctx", Type: "hsm.Context"}},
			},
		}},
	}

	protocol := NewAdapterProtocol(request)
	request.Imports[0].Path = "mutated-import"
	request.Imports[0].Specifiers[0].Name = "Mutated"
	request.Globals[0].Imports[0].Path = "mutated-global"
	request.Models[0].InitialEffects[0].ID = "mutated-initial"
	request.Models[0].Operations[0].Behavior.ID = "mutated-operation"
	request.Models[0].States[0].Behaviors[0].ID = "mutated-entry"
	request.Models[0].States[0].Defers[0] = "mutated-defer"
	request.Models[0].States[0].Transitions[0].Trigger.Values[0].EventSymbol = "mutated-event"
	request.Models[0].States[0].Transitions[0].Effects[0].ID = "mutated-effect"
	request.Behaviors[0].Imports[0].Path = "mutated-behavior"
	request.Behaviors[0].TargetABI.Parameters[0].Type = "mutated-abi"

	if protocol.Request.Imports[0].Path != "fmt" || protocol.Request.Imports[0].Specifiers[0].Name != "Println" {
		t.Fatalf("protocol request imports changed after caller mutation: %#v", protocol.Request.Imports)
	}
	if protocol.Request.Globals[0].Imports[0].Path != "strings" {
		t.Fatalf("protocol request globals changed after caller mutation: %#v", protocol.Request.Globals)
	}
	if protocol.Request.Models[0].InitialEffects[0].ID != "initial_effect_1" ||
		protocol.Request.Models[0].Operations[0].Behavior.ID != "operation_behavior_1" ||
		protocol.Request.Models[0].States[0].Behaviors[0].ID != "entry_1" ||
		protocol.Request.Models[0].States[0].Defers[0] != "knock" ||
		protocol.Request.Models[0].States[0].Transitions[0].Trigger.Values[0].EventSymbol != "openEvent" ||
		protocol.Request.Models[0].States[0].Transitions[0].Effects[0].ID != "effect_1" {
		t.Fatalf("protocol request model views changed after caller mutation: %#v", protocol.Request.Models)
	}
	if protocol.Request.Behaviors[0].Imports[0].Path != "context" ||
		protocol.Request.Behaviors[0].TargetABI.Parameters[0].Type != "hsm.Context" {
		t.Fatalf("protocol request behaviors changed after caller mutation: %#v", protocol.Request.Behaviors)
	}

	protocol.Request.Imports[0].Path = "protocol-import"
	protocol.Request.Globals[0].Imports[0].Path = "protocol-global"
	protocol.Request.Models[0].States[0].Transitions[0].Trigger.Values[0].EventSymbol = "protocol-event"
	protocol.Request.Behaviors[0].TargetABI.Parameters[0].Type = "protocol-abi"

	if request.Imports[0].Path != "mutated-import" ||
		request.Globals[0].Imports[0].Path != "mutated-global" ||
		request.Models[0].States[0].Transitions[0].Trigger.Values[0].EventSymbol != "mutated-event" ||
		request.Behaviors[0].TargetABI.Parameters[0].Type != "mutated-abi" {
		t.Fatalf("protocol request mutation changed caller request: %#v", request)
	}
}

func TestAdapterProtocolAllowedEditsCarriesStructuredRegionImports(t *testing.T) {
	request := AdapterRequest{
		SourceLanguage: LanguageTS,
		TargetLanguage: LanguagePython,
		Globals: []CodeBlock{{
			ID: "global_1",
			Imports: []Import{{
				Path:       "./format",
				Default:    "format",
				LocalNames: []string{"format"},
				Language:   LanguageTS,
			}},
		}},
		Behaviors: []Behavior{{
			ID: "entry_1",
			Imports: []Import{{
				Path:       "./record",
				Specifiers: []ImportSpecifier{{Name: "record", Alias: "rec"}},
				LocalNames: []string{"rec"},
				Language:   LanguageTS,
			}},
		}},
	}

	protocol := NewAdapterProtocol(request)
	if len(protocol.AllowedEdits.GlobalImports) != 1 || len(protocol.AllowedEdits.GlobalImports[0].Imports) != 1 {
		t.Fatalf("global import regions missing structured imports: %#v", protocol.AllowedEdits.GlobalImports)
	}
	if len(protocol.AllowedEdits.BehaviorImports) != 1 || len(protocol.AllowedEdits.BehaviorImports[0].Imports) != 1 {
		t.Fatalf("behavior import regions missing structured imports: %#v", protocol.AllowedEdits.BehaviorImports)
	}
	if protocol.AllowedEdits.GlobalImports[0].ImportPaths[0] != "./format" {
		t.Fatalf("global import paths not preserved: %#v", protocol.AllowedEdits.GlobalImports[0])
	}
	if got := protocol.AllowedEdits.BehaviorImports[0].Imports[0].Specifiers[0].Alias; got != "rec" {
		t.Fatalf("behavior structured import alias = %q, want rec", got)
	}

	protocol.AllowedEdits.BehaviorImports[0].Imports[0].Specifiers[0].Alias = "mutated"
	if request.Behaviors[0].Imports[0].Specifiers[0].Alias != "rec" {
		t.Fatalf("allowed edits import mutation changed request imports: %#v", request.Behaviors[0].Imports)
	}
}

func TestAdapterProtocolCarriesEventsOnlyAsReadOnlyContext(t *testing.T) {
	request := AdapterRequest{
		SourceLanguage: LanguageTS,
		TargetLanguage: LanguagePython,
		Events:         []EventDecl{{ID: "event_1", Symbol: "goEvent", Name: "go", Language: LanguageTS}},
		Models: []ModelView{{
			ID:     "model_1",
			Name:   "Door",
			Path:   "/Door",
			Events: []EventView{{ID: "event_1", Symbol: "goEvent", Name: "go", Language: LanguageTS}},
		}},
		Globals:   []CodeBlock{{ID: "global_1", Language: LanguageTS, Code: "const helper = true;"}},
		Behaviors: []Behavior{{ID: "entry_1", SourceLanguage: LanguageTS, Body: "record('go');"}},
	}

	protocol := NewAdapterProtocol(request)
	if len(protocol.Request.Events) != 1 || protocol.Request.Events[0].Symbol != "goEvent" {
		t.Fatalf("request events missing read-only context: %#v", protocol.Request.Events)
	}
	if len(protocol.Request.Models) != 1 || len(protocol.Request.Models[0].Events) != 1 || protocol.Request.Models[0].Events[0].Name != "go" {
		t.Fatalf("model event view missing read-only context: %#v", protocol.Request.Models)
	}
	allowed, err := json.Marshal(protocol.AllowedEdits)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(allowed), "event") || strings.Contains(strings.ToLower(string(allowed)), "goevent") {
		t.Fatalf("events leaked into allowed edits: %s", allowed)
	}
	if !containsString(protocol.Response.TopLevelFields, "globals") || !containsString(protocol.Response.TopLevelFields, "behaviors") || containsString(protocol.Response.TopLevelFields, "events") {
		t.Fatalf("response top-level fields = %#v, want globals/behaviors editable but not events", protocol.Response.TopLevelFields)
	}
	if !strings.Contains(protocol.Response.UnknownFieldPolicy, "events") {
		t.Fatalf("response contract should explicitly forbid event edits: %q", protocol.Response.UnknownFieldPolicy)
	}
}

func TestAdapterProtocolCarriesAttributesAndOperationsOnlyAsReadOnlyContext(t *testing.T) {
	request := AdapterRequest{
		SourceLanguage: LanguageTS,
		TargetLanguage: LanguagePython,
		Models: []ModelView{{
			ID:         "model_1",
			Name:       "Door",
			Path:       "/Door",
			Attributes: []AttributeView{{ID: "attribute_1", Name: "count", Type: "number", HasDefault: true, Default: "0", Language: LanguageTS}},
			Operations: []OperationView{{ID: "operation_1", Name: "approve", Behavior: &BehaviorRef{ID: "operation_behavior_1", Kind: BehaviorOperation}}},
		}},
		Behaviors: []Behavior{{
			ID:             "operation_behavior_1",
			OwnerID:        "model_1",
			Kind:           BehaviorOperation,
			SourceLanguage: LanguageTS,
			Body:           "return instance.get('count') > 0;",
		}},
	}

	protocol := NewAdapterProtocol(request)
	if len(protocol.Request.Models) != 1 || len(protocol.Request.Models[0].Attributes) != 1 || protocol.Request.Models[0].Attributes[0].Name != "count" {
		t.Fatalf("model attributes missing read-only context: %#v", protocol.Request.Models)
	}
	if len(protocol.Request.Models[0].Operations) != 1 || protocol.Request.Models[0].Operations[0].Name != "approve" || protocol.Request.Models[0].Operations[0].Behavior == nil {
		t.Fatalf("model operations missing read-only context: %#v", protocol.Request.Models)
	}
	if got := strings.Join(protocol.AllowedEdits.BehaviorIDs, ","); got != "operation_behavior_1" {
		t.Fatalf("operation implementation behavior should be editable by ID, got allowed behavior IDs %q", got)
	}
	allowed, err := json.Marshal(protocol.AllowedEdits)
	if err != nil {
		t.Fatal(err)
	}
	allowedText := strings.ToLower(string(allowed))
	for _, forbidden := range []string{"attribute", "operation_1", "approve"} {
		if strings.Contains(allowedText, forbidden) {
			t.Fatalf("metadata leaked into allowed edits: %s", allowed)
		}
	}
	if containsString(protocol.Response.TopLevelFields, "attributes") || containsString(protocol.Response.TopLevelFields, "operations") {
		t.Fatalf("response top-level fields should not allow metadata edits: %#v", protocol.Response.TopLevelFields)
	}
	if !strings.Contains(protocol.Response.UnknownFieldPolicy, "attributes, operations") {
		t.Fatalf("response contract should explicitly forbid metadata edits: %q", protocol.Response.UnknownFieldPolicy)
	}
}

func TestAdapterProtocolCarriesDefersOnlyAsReadOnlyContext(t *testing.T) {
	request := AdapterRequest{
		SourceLanguage: LanguageTS,
		TargetLanguage: LanguagePython,
		Models: []ModelView{{
			ID:   "model_1",
			Name: "Door",
			Path: "/Door",
			States: []StateView{{
				ID:     "state_1",
				Name:   "closed",
				Path:   "/Door/closed",
				Kind:   StateKindState,
				Defers: []string{"knock", "timeout"},
				Behaviors: []BehaviorRef{{
					ID:   "entry_1",
					Kind: BehaviorEntry,
				}},
			}},
		}},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageTS,
			Body:           "record('closed');",
		}},
	}

	protocol := NewAdapterProtocol(request)
	if len(protocol.Request.Models) != 1 || len(protocol.Request.Models[0].States) != 1 {
		t.Fatalf("protocol model state context = %#v", protocol.Request.Models)
	}
	if got := strings.Join(protocol.Request.Models[0].States[0].Defers, ","); got != "knock,timeout" {
		t.Fatalf("protocol defers context = %q, want knock,timeout", got)
	}
	allowed, err := json.Marshal(protocol.AllowedEdits)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(allowed)), "defer") || strings.Contains(string(allowed), "knock") || strings.Contains(string(allowed), "timeout") {
		t.Fatalf("defers leaked into allowed edits: %s", allowed)
	}
	if containsString(protocol.Response.TopLevelFields, "defers") {
		t.Fatalf("response top-level fields should not allow defer edits: %#v", protocol.Response.TopLevelFields)
	}
	if !strings.Contains(protocol.Response.UnknownFieldPolicy, "defers") {
		t.Fatalf("response contract should explicitly forbid defer edits: %q", protocol.Response.UnknownFieldPolicy)
	}
}

func TestBuildAdapterProtocolFromJSONIRRetargetsBehaviorABI(t *testing.T) {
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
      "signature": "def entry(ctx, instance, event):"
    },
    "body": "fmt.Println(\"closed\")"
  }]
}`

	protocol, program, err := NewCompiler().BuildAdapterProtocol(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From: LanguageJSONIR,
		To:   LanguageTS,
	})
	if err != nil {
		t.Fatal(err)
	}
	if program.SourceLanguage != LanguageGo || protocol.Request.SourceLanguage != LanguageGo {
		t.Fatalf("source language = program %q request %q, want go", program.SourceLanguage, protocol.Request.SourceLanguage)
	}
	if len(protocol.Request.Behaviors) != 1 {
		t.Fatalf("request behaviors = %#v", protocol.Request.Behaviors)
	}
	behavior := protocol.Request.Behaviors[0]
	if behavior.SourceLanguage != LanguageGo {
		t.Fatalf("request behavior source language = %q, want go", behavior.SourceLanguage)
	}
	if behavior.TargetLanguage != "" {
		t.Fatalf("stale json-ir behavior target language reached adapter request: %#v", behavior)
	}
	if behavior.TargetABI == nil || behavior.TargetABI.Language != LanguageTS || behavior.TargetABI.Signature == "" {
		t.Fatalf("request behavior target ABI = %#v, want TypeScript ABI", behavior.TargetABI)
	}
	if behavior.TargetABI.Language == LanguagePython || strings.Contains(behavior.TargetABI.Signature, "def entry") {
		t.Fatalf("stale json-ir target ABI reached adapter request: %#v", behavior.TargetABI)
	}
}

func TestBuildAdapterProtocolRejectsTargetGlobalSymbolCollisions(t *testing.T) {
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
    "source_language": "typescript",
    "body": "instance.log = \"closed\";"
  }]
}`

	_, _, err := NewCompiler().BuildAdapterProtocol(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From: LanguageJSONIR,
		To:   LanguageTS,
	})
	want := `target-language global "global_1" declares "entry1", which collides with compiler-owned typescript symbol`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestBuildAdapterProtocolFromJSONIRCarriesStructuredMutableImportContext(t *testing.T) {
	source := `{
  "source_language": "typescript",
  "globals": [{
    "id": "global_1",
    "language": "typescript",
    "code": "export function record(value: string): string { return normalize(value); }",
    "imports": [{
      "path": "./global-format",
      "language": "typescript",
      "specifiers": [{ "name": "normalize", "alias": "norm" }],
      "local_names": ["norm"]
    }]
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
    "source_language": "typescript",
    "body": "instance.log = record(format(\"closed\"));",
    "imports": [{
      "path": "./behavior-format",
      "language": "typescript",
      "specifiers": [{ "name": "format", "alias": "fmt" }],
      "local_names": ["fmt"]
    }]
  }]
}`

	protocol, _, err := NewCompiler().BuildAdapterProtocol(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From: LanguageJSONIR,
		To:   LanguagePython,
	})
	if err != nil {
		t.Fatal(err)
	}
	if protocol.Request.SourceLanguage != LanguageTS || protocol.Request.TargetLanguage != LanguagePython {
		t.Fatalf("protocol request languages = %q -> %q, want typescript -> python", protocol.Request.SourceLanguage, protocol.Request.TargetLanguage)
	}
	if len(protocol.Request.Globals) != 1 || len(protocol.Request.Globals[0].Imports) != 1 {
		t.Fatalf("protocol request globals missing source import context: %#v", protocol.Request.Globals)
	}
	globalImport := protocol.Request.Globals[0].Imports[0]
	if globalImport.Path != "./global-format" || globalImport.Language != LanguageTS || len(globalImport.Specifiers) != 1 || globalImport.Specifiers[0].Alias != "norm" || strings.Join(globalImport.LocalNames, ",") != "norm" {
		t.Fatalf("protocol global import context = %#v", globalImport)
	}
	if len(protocol.Request.Behaviors) != 1 || len(protocol.Request.Behaviors[0].Imports) != 1 {
		t.Fatalf("protocol request behaviors missing source import context: %#v", protocol.Request.Behaviors)
	}
	behaviorImport := protocol.Request.Behaviors[0].Imports[0]
	if behaviorImport.Path != "./behavior-format" || behaviorImport.Language != LanguageTS || len(behaviorImport.Specifiers) != 1 || behaviorImport.Specifiers[0].Alias != "fmt" || strings.Join(behaviorImport.LocalNames, ",") != "fmt" {
		t.Fatalf("protocol behavior import context = %#v", behaviorImport)
	}
	if len(protocol.AllowedEdits.GlobalImports) != 1 || protocol.AllowedEdits.GlobalImports[0].Imports[0].Path != "./global-format" {
		t.Fatalf("protocol allowed global import context = %#v", protocol.AllowedEdits.GlobalImports)
	}
	if len(protocol.AllowedEdits.BehaviorImports) != 1 || protocol.AllowedEdits.BehaviorImports[0].Imports[0].Path != "./behavior-format" {
		t.Fatalf("protocol allowed behavior import context = %#v", protocol.AllowedEdits.BehaviorImports)
	}
}

func TestAdapterProtocolJavaGuidanceListsWildcardRuntimeImport(t *testing.T) {
	protocol := NewAdapterProtocol(AdapterRequest{TargetLanguage: LanguageJava})
	if !containsString(protocol.Guidance.CompilerOwnedImports, "com.stateforward.hsm") {
		t.Fatalf("Java compiler-owned imports missing package path: %#v", protocol.Guidance.CompilerOwnedImports)
	}
	if !containsString(protocol.Guidance.CompilerOwnedImports, "com.stateforward.hsm.*") {
		t.Fatalf("Java compiler-owned imports missing wildcard path: %#v", protocol.Guidance.CompilerOwnedImports)
	}
	if !containsString(protocol.Guidance.CompilerOwnedImports, "java.lang.Object") {
		t.Fatalf("Java compiler-owned imports missing signature Object path: %#v", protocol.Guidance.CompilerOwnedImports)
	}
	if !containsString(protocol.Guidance.CompilerOwnedBindings, "Object") {
		t.Fatalf("Java compiler-owned bindings missing Object: %#v", protocol.Guidance.CompilerOwnedBindings)
	}
}

func TestAdapterProtocolCSharpGuidanceListsRuntimeClassImports(t *testing.T) {
	protocol := NewAdapterProtocol(AdapterRequest{TargetLanguage: LanguageCSharp})
	for _, path := range []string{"Stateforward.Hsm", "Stateforward.Hsm.Hsm", "Stateforward.Hsm.Event", "Stateforward.Hsm.Instance", "Stateforward.Hsm.Model", "Stateforward.Hsm.Operation", "Stateforward.Hsm.Expression", "Stateforward.Hsm.DurationProvider", "Stateforward.Hsm.TimeProvider", "Stateforward.Hsm.ConditionChannel"} {
		if !containsString(protocol.Guidance.CompilerOwnedImports, path) {
			t.Fatalf("C# compiler-owned imports missing %q: %#v", path, protocol.Guidance.CompilerOwnedImports)
		}
	}
	for _, binding := range []string{"Operation", "Expression", "DurationProvider", "TimeProvider", "ConditionChannel"} {
		if !containsString(protocol.Guidance.CompilerOwnedBindings, binding) {
			t.Fatalf("C# compiler-owned bindings missing %q: %#v", binding, protocol.Guidance.CompilerOwnedBindings)
		}
	}
}

func TestAdapterProtocolTypeScriptGuidanceListsRuntimeImportSpellings(t *testing.T) {
	protocol := NewAdapterProtocol(AdapterRequest{TargetLanguage: LanguageTS})
	if !containsString(protocol.Guidance.CompilerOwnedImports, "@stateforward/hsm.ts") {
		t.Fatalf("TypeScript compiler-owned imports missing generated runtime path: %#v", protocol.Guidance.CompilerOwnedImports)
	}
	if !containsString(protocol.Guidance.CompilerOwnedImports, "@stateforward/hsm") {
		t.Fatalf("TypeScript compiler-owned imports missing package runtime path: %#v", protocol.Guidance.CompilerOwnedImports)
	}
	for _, path := range []string{"typescript", "typescript.Date", "typescript.InstanceType", "typescript.Promise", "typescript.undefined"} {
		if !containsString(protocol.Guidance.CompilerOwnedImports, path) {
			t.Fatalf("TypeScript compiler-owned imports missing %q: %#v", path, protocol.Guidance.CompilerOwnedImports)
		}
	}
	for _, binding := range []string{"Date", "InstanceType", "Promise", "undefined"} {
		if !containsString(protocol.Guidance.CompilerOwnedBindings, binding) {
			t.Fatalf("TypeScript compiler-owned bindings missing %q: %#v", binding, protocol.Guidance.CompilerOwnedBindings)
		}
	}
}

func TestAdapterProtocolJavaScriptGuidanceListsCompilerOwnedDateBinding(t *testing.T) {
	protocol := NewAdapterProtocol(AdapterRequest{TargetLanguage: LanguageJS})
	for _, path := range []string{"@stateforward/hsm", "javascript", "javascript.Date", "javascript.undefined"} {
		if !containsString(protocol.Guidance.CompilerOwnedImports, path) {
			t.Fatalf("JavaScript compiler-owned imports missing %q: %#v", path, protocol.Guidance.CompilerOwnedImports)
		}
	}
	for _, binding := range []string{"Date", "hsm", "undefined"} {
		if !containsString(protocol.Guidance.CompilerOwnedBindings, binding) {
			t.Fatalf("JavaScript compiler-owned bindings missing %q: %#v", binding, protocol.Guidance.CompilerOwnedBindings)
		}
	}
}

func TestAdapterProtocolDartGuidanceListsAsyncRuntimeImport(t *testing.T) {
	protocol := NewAdapterProtocol(AdapterRequest{TargetLanguage: LanguageDart})
	if !containsString(protocol.Guidance.CompilerOwnedImports, "package:hsm/hsm.dart") {
		t.Fatalf("Dart compiler-owned imports missing hsm runtime path: %#v", protocol.Guidance.CompilerOwnedImports)
	}
	if !containsString(protocol.Guidance.CompilerOwnedImports, "dart:async") {
		t.Fatalf("Dart compiler-owned imports missing async helper path: %#v", protocol.Guidance.CompilerOwnedImports)
	}
	if !containsString(protocol.Guidance.CompilerOwnedImports, "dart:core") {
		t.Fatalf("Dart compiler-owned imports missing core helper path: %#v", protocol.Guidance.CompilerOwnedImports)
	}
	if protocol.Guidance.TargetImportShape.SupportsHiddenSpecifiers {
		t.Fatalf("Dart target import shape should reject adapter hide combinators: %#v", protocol.Guidance.TargetImportShape)
	}
	if !containsString(protocol.Guidance.TargetImportShape.Notes, "Native Dart source context may preserve hide combinators, but translated target imports cannot use them because they implicitly bind unlisted names.") {
		t.Fatalf("Dart target import shape should explain hide combinator boundary: %#v", protocol.Guidance.TargetImportShape.Notes)
	}
	if !containsString(protocol.Guidance.TargetImportShape.Notes, "Do not combine a Dart prefix alias with named specifiers in translated target imports; use either a prefix import or a show import.") {
		t.Fatalf("Dart target import shape should explain prefix/show boundary: %#v", protocol.Guidance.TargetImportShape.Notes)
	}
	for _, binding := range []string{"at", "every", "onCall", "when", "DateTime", "Duration", "Future", "Object", "String", "bool", "double", "int", "num"} {
		if !containsString(protocol.Guidance.CompilerOwnedBindings, binding) {
			t.Fatalf("Dart compiler-owned bindings missing %q: %#v", binding, protocol.Guidance.CompilerOwnedBindings)
		}
	}
}

func TestAdapterProtocolRustGuidanceListsCompilerOwnedSignatureBindings(t *testing.T) {
	protocol := NewAdapterProtocol(AdapterRequest{TargetLanguage: LanguageRust})
	for _, path := range []string{"std::boxed", "std::boxed::Box", "std::marker", "std::marker::Send"} {
		if !containsString(protocol.Guidance.CompilerOwnedImports, path) {
			t.Fatalf("Rust compiler-owned imports missing %q: %#v", path, protocol.Guidance.CompilerOwnedImports)
		}
	}
	for _, binding := range []string{"Any", "Box", "Duration", "Future", "Pin", "Send"} {
		if !containsString(protocol.Guidance.CompilerOwnedBindings, binding) {
			t.Fatalf("Rust compiler-owned bindings missing %q: %#v", binding, protocol.Guidance.CompilerOwnedBindings)
		}
	}
}

func TestAdapterProtocolPythonGuidanceListsCompilerOwnedSignatureBindings(t *testing.T) {
	protocol := NewAdapterProtocol(AdapterRequest{TargetLanguage: LanguagePython})
	for _, path := range []string{"builtins", "builtins.False", "builtins.None", "builtins.True", "builtins.bool", "builtins.float"} {
		if !containsString(protocol.Guidance.CompilerOwnedImports, path) {
			t.Fatalf("Python compiler-owned imports missing %q: %#v", path, protocol.Guidance.CompilerOwnedImports)
		}
	}
	for _, binding := range []string{"False", "None", "True", "bool", "float"} {
		if !containsString(protocol.Guidance.CompilerOwnedBindings, binding) {
			t.Fatalf("Python compiler-owned bindings missing %q: %#v", binding, protocol.Guidance.CompilerOwnedBindings)
		}
	}
}

func TestAdapterProtocolGoGuidanceListsCompilerOwnedSignatureBindings(t *testing.T) {
	protocol := NewAdapterProtocol(AdapterRequest{TargetLanguage: LanguageGo})
	for _, path := range []string{"builtin", "builtin.bool", "builtin.false", "builtin.nil", "builtin.true"} {
		if !containsString(protocol.Guidance.CompilerOwnedImports, path) {
			t.Fatalf("Go compiler-owned imports missing %q: %#v", path, protocol.Guidance.CompilerOwnedImports)
		}
	}
	for _, binding := range []string{"bool", "false", "nil", "true"} {
		if !containsString(protocol.Guidance.CompilerOwnedBindings, binding) {
			t.Fatalf("Go compiler-owned bindings missing %q: %#v", binding, protocol.Guidance.CompilerOwnedBindings)
		}
	}
}

func TestAdapterProtocolGuidanceListsCompilerOwnedGeneratedSymbols(t *testing.T) {
	csharpProtocol := NewAdapterProtocol(AdapterRequest{
		TargetLanguage: LanguageCSharp,
		Models:         []ModelView{{ID: "model_1", Name: "Door"}},
		Behaviors:      []Behavior{{ID: "entry_1", Kind: BehaviorEntry}},
	})
	for _, symbol := range []string{"GeneratedHsm", "DoorModel", "Entry1"} {
		if !containsString(csharpProtocol.Guidance.CompilerOwnedSymbols, symbol) {
			t.Fatalf("C# compiler-owned generated symbols missing %q: %#v", symbol, csharpProtocol.Guidance.CompilerOwnedSymbols)
		}
	}

	request := AdapterRequest{
		TargetLanguage: LanguageCPP,
		Events:         []EventDecl{{ID: "event_1", Symbol: "open_event", Name: "open"}},
		Models: []ModelView{{
			ID:   "model_1",
			Name: "Door",
			States: []StateView{{
				ID:     "state_1",
				Name:   "closed",
				Defers: []string{"timeout"},
			}},
		}},
		Behaviors: []Behavior{{ID: "entry_1", Kind: BehaviorEntry}},
	}
	protocol := NewAdapterProtocol(request)
	for _, symbol := range []string{"OpenEvent", "TimeoutEvent", "door_model", "entry_1", "open_event"} {
		if !containsString(protocol.Guidance.CompilerOwnedSymbols, symbol) {
			t.Fatalf("compiler-owned generated symbols missing %q: %#v", symbol, protocol.Guidance.CompilerOwnedSymbols)
		}
	}
}

func TestAdapterProtocolCompilerOwnedSymbolsMatchGeneratedTopLevelSymbols(t *testing.T) {
	program := &Program{
		Events: []EventDecl{{ID: "event_1", Symbol: "open_event", Name: "open"}},
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			States: []State{{
				ID:     "state_1",
				Name:   "closed",
				Defers: []string{"timeout"},
				States: []State{{
					ID:     "state_2",
					Name:   "locked",
					Defers: []string{"maintenance"},
					Transitions: []Transition{{
						ID: "transition_1",
						Trigger: &Trigger{
							Kind:     TriggerAfter,
							Value:    "deadline()",
							Language: LanguageGo,
						},
					}},
				}},
				Transitions: []Transition{{
					ID: "transition_2",
					Trigger: &Trigger{
						Kind:     TriggerWhen,
						Value:    "ready()",
						Language: LanguageTS,
					},
				}},
			}},
		}},
		Behaviors: []Behavior{{ID: "entry_1", Kind: BehaviorEntry}},
	}

	for _, target := range SupportedTargetLanguages() {
		if !isImplementationLanguage(target) {
			continue
		}
		t.Run(string(target), func(t *testing.T) {
			request := buildAdapterRequest(program, target)
			protocol := NewAdapterProtocol(request)
			want := uniqueGeneratedTopLevelSymbols(program, target)
			got := protocol.Guidance.CompilerOwnedSymbols
			if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
				t.Fatalf("compiler-owned symbols for %s = %#v, want %#v", target, got, want)
			}
		})
	}
}

func TestAdapterProtocolGuidanceListsCompilerOwnedScaffoldSymbols(t *testing.T) {
	pythonProtocol := NewAdapterProtocol(AdapterRequest{
		TargetLanguage: LanguagePython,
		Models: []ModelView{{
			ID:   "model_1",
			Name: "Door",
			States: []StateView{{
				ID:   "state_1",
				Name: "closed",
				Transitions: []TransitionView{{
					ID: "transition_1",
					Trigger: &Trigger{
						Kind:     TriggerWhen,
						Value:    "ready()",
						Language: LanguageTS,
					},
				}},
			}},
		}},
	})
	if !containsString(pythonProtocol.Guidance.CompilerOwnedSymbols, "hsmc_trigger_fallback_1") {
		t.Fatalf("Python compiler-owned fallback symbols missing hsmc_trigger_fallback_1: %#v", pythonProtocol.Guidance.CompilerOwnedSymbols)
	}

	rustProtocol := NewAdapterProtocol(AdapterRequest{
		TargetLanguage: LanguageRust,
		Models:         []ModelView{{ID: "model_1", Name: "Door"}},
	})
	for _, symbol := range []string{"HsmcInstance", "hsmc_true_guard", "hsmc_zero_duration", "hsmc_unix_epoch"} {
		if !containsString(rustProtocol.Guidance.CompilerOwnedSymbols, symbol) {
			t.Fatalf("Rust compiler-owned scaffold symbols missing %q: %#v", symbol, rustProtocol.Guidance.CompilerOwnedSymbols)
		}
	}

	zigProtocol := NewAdapterProtocol(AdapterRequest{
		TargetLanguage: LanguageZig,
		Models: []ModelView{{
			ID:             "model_1",
			Name:           "Door",
			InitialEffects: []BehaviorRef{{Kind: BehaviorEffect, Operation: "approve"}},
			States: []StateView{{
				ID:   "state_1",
				Name: "closed",
				Behaviors: []BehaviorRef{
					{Kind: BehaviorEntry, Operation: "approve"},
					{Kind: BehaviorExit, Operation: "approve"},
					{Kind: BehaviorActivity, Operation: "approve"},
				},
				Transitions: []TransitionView{{
					ID: "transition_1",
					Trigger: &Trigger{
						Kind:     TriggerAfter,
						Language: LanguageGo,
					},
					Guard:   &BehaviorRef{Kind: BehaviorGuard, Operation: "approve"},
					Effects: []BehaviorRef{{Kind: BehaviorEffect, Operation: "approve"}},
				}},
			}},
		}},
	})
	if !containsString(zigProtocol.Guidance.CompilerOwnedSymbols, "hsmc_trigger_zero") {
		t.Fatalf("Zig compiler-owned scaffold symbols missing hsmc_trigger_zero: %#v", zigProtocol.Guidance.CompilerOwnedSymbols)
	}
	for _, symbol := range []string{
		zigOperationReferenceFallbackName(BehaviorEffect, "approve"),
		zigOperationReferenceFallbackName(BehaviorEntry, "approve"),
		zigOperationReferenceFallbackName(BehaviorExit, "approve"),
		zigOperationReferenceFallbackName(BehaviorActivity, "approve"),
		zigOperationReferenceFallbackName(BehaviorGuard, "approve"),
	} {
		if !containsString(zigProtocol.Guidance.CompilerOwnedSymbols, symbol) {
			t.Fatalf("Zig compiler-owned operation fallback symbols missing %q: %#v", symbol, zigProtocol.Guidance.CompilerOwnedSymbols)
		}
	}
}

func TestBuildAdapterProtocolListsZigOperationReferenceFallbackSymbols(t *testing.T) {
	protocol, _, err := NewCompiler().BuildAdapterProtocol(context.Background(), SourceInput{
		Path: "door.go",
		Data: []byte(goOperationReferenceMatrixSource),
	}, CompileOptions{
		From: LanguageGo,
		To:   LanguageZig,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, symbol := range []string{
		zigOperationReferenceFallbackName(BehaviorEffect, "approve"),
		zigOperationReferenceFallbackName(BehaviorEntry, "approve"),
		zigOperationReferenceFallbackName(BehaviorExit, "approve"),
		zigOperationReferenceFallbackName(BehaviorActivity, "approve"),
		zigOperationReferenceFallbackName(BehaviorGuard, "approve"),
	} {
		if !containsString(protocol.Guidance.CompilerOwnedSymbols, symbol) {
			t.Fatalf("Zig adapter protocol compiler-owned symbols missing %q: %#v", symbol, protocol.Guidance.CompilerOwnedSymbols)
		}
	}
}

func uniqueGeneratedTopLevelSymbols(program *Program, target Language) []string {
	seen := map[string]bool{}
	for _, use := range generatedTopLevelSymbolUses(program, target) {
		if use.Symbol != "" {
			seen[use.Symbol] = true
		}
	}
	symbols := make([]string, 0, len(seen))
	for symbol := range seen {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)
	return symbols
}

func TestCompilerOwnedImportPathsCoverRuntimeBindingImports(t *testing.T) {
	for _, target := range SupportedTargetLanguages() {
		if !isImplementationLanguage(target) {
			continue
		}
		t.Run(string(target), func(t *testing.T) {
			for _, imp := range compilerOwnedBindingImports(target) {
				if !isCompilerOwnedImportPath(imp.Path, target) {
					t.Fatalf("compiler-owned binding import %q for %s is not listed in compiler-owned import paths: %#v", imp.Path, target, compilerOwnedImports(target))
				}
			}
		})
	}
}

func TestCompilerOwnedRuntimeBindingImportsDoNotConflict(t *testing.T) {
	for _, target := range SupportedTargetLanguages() {
		if !isImplementationLanguage(target) {
			continue
		}
		t.Run(string(target), func(t *testing.T) {
			bindings := map[string]importBindingUse{}
			for _, imp := range compilerOwnedBindingImports(target) {
				for _, binding := range importBindings(imp, target) {
					if binding.Local == "" || binding.Local == "." || binding.Local == "_" {
						continue
					}
					current := importBindingUse{Path: imp.Path, Owner: "compiler-owned runtime", Source: binding.Source}
					if previous, exists := bindings[binding.Local]; exists && previous.Source != current.Source {
						t.Fatalf("compiler-owned runtime imports %q and %q both bind local name %q for %s", previous.Path, current.Path, binding.Local, target)
					}
					bindings[binding.Local] = current
				}
			}
		})
	}
}
