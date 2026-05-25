package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestPythonBackendEmitsModelStructureAndCommentedForeignBehaviors(t *testing.T) {
	source := `package sample

import hsm "github.com/stateforward/hsm.go"

type DoorHSM struct{ hsm.HSM }

func approve(sm *DoorHSM) bool { return true }

var DoorModel = hsm.Define(
	"Door",
	hsm.Attribute("count", 1),
	hsm.Operation("approve", approve),
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed",
		hsm.Entry(func(sm *DoorHSM) {}),
		hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open")),
	),
	hsm.State("open"),
)
`
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(source)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguagePython,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`import hsm`,
		`async def operation_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:`,
		`# Original go behavior operation_1 preserved for manual porting:`,
		`# approve`,
		`pass`,
		`door_model = hsm.Define(`,
		`hsm.Attribute("count", 1)`,
		`hsm.Operation("approve", operation_1)`,
		`hsm.Entry(entry_1)`,
		`hsm.OnCall("approve")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python output missing %q:\n%s", needle, text)
		}
	}
}

func TestPythonTargetABI(t *testing.T) {
	abi := BehaviorABIForTrigger(LanguagePython, BehaviorTrigger, TriggerAt)
	if abi == nil || abi.ReturnType != "datetime.datetime" || !strings.Contains(abi.Signature, "async def") {
		t.Fatalf("python trigger ABI = %#v", abi)
	}
}

func TestPythonBackendEmitsAttributesWithPortableFallbacks(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			Attributes: []Attribute{
				{ID: "attribute_1", Name: "count", Type: "int", HasDefault: true, Default: "1", Language: LanguagePython},
				{ID: "attribute_2", Name: "label", HasDefault: true, Default: `"closed"`, Language: LanguageGo},
				{ID: "attribute_3", Name: "ready", HasDefault: true, Default: "true", Language: LanguageGo},
				{ID: "attribute_4", Name: "payload", HasDefault: true, Default: "makePayload()", Language: LanguageGo},
				{ID: "attribute_5", Name: "type_only", Type: "str", Language: LanguagePython},
			},
		}},
	}

	output, err := NewPythonBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`# Type: int`,
		`hsm.Attribute("count", 1)`,
		`hsm.Attribute("label", "closed")`,
		`hsm.Attribute("ready", True)`,
		`# Default: makePayload()`,
		`hsm.Attribute("payload")`,
		`# Type: str`,
		`hsm.Attribute("type_only")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python attribute output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{
		`hsm.Attribute("ready", true)`,
		`hsm.Attribute("payload", makePayload())`,
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Python attribute output emitted invalid foreign default %q:\n%s", forbidden, text)
		}
	}
}

func TestPythonBackendPreservesForeignRawAtAndWhenTriggerFallbacks(t *testing.T) {
	program := &Program{
		SourceLanguage: LanguageTS,
		Models: []Model{{
			ID:   "model_1",
			Name: "Timer",
			Initializer: &Initial{
				ID:      "initial_1",
				OwnerID: "model_1",
				Target:  "idle",
			},
			States: []State{{
				ID:   "state_1",
				Name: "idle",
				Kind: StateKindState,
				Transitions: []Transition{{
					ID:      "transition_1",
					OwnerID: "state_1",
					Target:  "../done",
					Trigger: &Trigger{
						Kind:     TriggerAt,
						Value:    "new Date()",
						Language: LanguageTS,
					},
				}, {
					ID:      "transition_2",
					OwnerID: "state_1",
					Target:  "../done",
					Trigger: &Trigger{
						Kind:     TriggerWhen,
						Value:    "ready()",
						Language: LanguageTS,
					},
				}},
			}, {
				ID:   "state_2",
				Name: "done",
				Kind: StateKindState,
			}},
		}},
	}

	output, err := NewPythonBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"def hsmc_trigger_fallback_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> datetime.datetime:",
		"# Original typescript at trigger expression preserved for manual porting:",
		"# new Date()",
		"return datetime.datetime.fromtimestamp(0)",
		"hsm.At(hsmc_trigger_fallback_1)",
		"def hsmc_trigger_fallback_2(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> bool:",
		"# Original typescript when trigger expression preserved for manual porting:",
		"# ready()",
		"return False",
		"hsm.When(hsmc_trigger_fallback_2)",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python output missing %q:\n%s", needle, text)
		}
	}
}

func TestPythonBackendUsesAdapterFullCodeBehavior(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguagePython,
			Code: `async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:
	instance.ready = True`,
		}},
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			States: []State{{
				ID:        "state_1",
				Name:      "closed",
				Kind:      StateKindState,
				Behaviors: []BehaviorRef{{ID: "entry_1", Kind: BehaviorEntry}},
			}},
		}},
	}
	AssignTargetABI(program, LanguagePython)

	output, err := NewPythonBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:`,
		`instance.ready = True`,
		`hsm.Entry(entry_1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "entry_1 = async def entry_1") {
		t.Fatalf("Python output wrapped a full function declaration as an expression:\n%s", text)
	}
}

func TestPythonBackendRejectsAdapterFullCodeBehaviorNameChanges(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguagePython,
			Code: `def renamed_entry(ctx, instance, event):
	instance.ready = True`,
		}},
	}
	AssignTargetABI(program, LanguagePython)

	_, err := NewPythonBackend().Emit(context.Background(), program)
	if err == nil || !strings.Contains(err.Error(), `declares "renamed_entry", want compiler-owned name "entry_1"`) {
		t.Fatalf("error = %v, want compiler-owned Python behavior name rejection", err)
	}
}

func TestPythonBackendRejectsAdapterFullCodeBehaviorSignatureChanges(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "sync function",
			code: `def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:
	instance.ready = True`,
			want: `declares synchronous function "entry_1", want compiler-owned async function`,
		},
		{
			name: "parameters",
			code: `async def entry_1(ctx: hsm.Context) -> None:
	pass`,
			want: `declares parameters "(ctx: hsm.Context)", want compiler-owned parameters "(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event)"`,
		},
		{
			name: "return type",
			code: `async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> bool:
	return True`,
			want: `declares return type "bool", want compiler-owned return type "None"`,
		},
		{
			name: "lambda parameters",
			code: `lambda ctx: None`,
			want: `declares lambda parameters "(ctx)", want compiler-owned parameters "(ctx, instance, event)"`,
		},
		{
			name: "lambda default parameter",
			code: `lambda ctx, instance, event=None: None`,
			want: `declares unsupported lambda signature, want compiler-owned parameters "(ctx, instance, event)"`,
		},
		{
			name: "lambda variadic parameter",
			code: `lambda ctx, instance, *event: None`,
			want: `declares unsupported lambda signature, want compiler-owned parameters "(ctx, instance, event)"`,
		},
		{
			name: "non-callable expression",
			code: `1`,
			want: `must be a compiler-owned callable expression`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := &Program{
				Behaviors: []Behavior{{
					ID:             "entry_1",
					OwnerID:        "state_1",
					Kind:           BehaviorEntry,
					SourceLanguage: LanguageGo,
					TargetLanguage: LanguagePython,
					Code:           tt.code,
				}},
			}
			AssignTargetABI(program, LanguagePython)

			_, err := NewPythonBackend().Emit(context.Background(), program)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}
