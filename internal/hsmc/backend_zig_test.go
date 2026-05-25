package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestZigBackendEmitsModelStructureAndCommentedForeignBehaviors(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageZig,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`const hsm = @import("hsm");`,
		"fn entry_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void",
		"_ = ctx;",
		"// Original go behavior entry_1 preserved for manual porting:",
		`// sm.log = append(sm.log, record("closed"))`,
		"fn guard_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) bool",
		"return false;",
		`pub const door_model = hsm.define("Door", .{`,
		`hsm.initial(hsm.target("closed")),`,
		`hsm.state("closed", .{`,
		`hsm.entry(entry_1),`,
		`hsm.transition(.{`,
		`hsm.on("open"),`,
		`hsm.guard(guard_1),`,
		`hsm.target("../open"),`,
		`hsm.effect(effect_1),`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Zig output missing %q:\n%s", needle, text)
		}
	}
}

func TestZigTargetABI(t *testing.T) {
	guard := BehaviorABIFor(LanguageZig, BehaviorGuard)
	if guard == nil || guard.ReturnType != "bool" || !strings.Contains(guard.Signature, "*hsm.Context") {
		t.Fatalf("Zig guard ABI = %#v", guard)
	}
	timer := BehaviorABIForTrigger(LanguageZig, BehaviorTrigger, TriggerAfter)
	if timer == nil || timer.ReturnType != "u64" {
		t.Fatalf("Zig timer ABI = %#v", timer)
	}
	when := BehaviorABIForTrigger(LanguageZig, BehaviorTrigger, TriggerWhen)
	if when == nil || when.ReturnType != "bool" {
		t.Fatalf("Zig when ABI = %#v", when)
	}
}

func TestZigBackendLowersAtTriggerToAfterWithPreservedSourceComment(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:   "model_1",
			Name: "Clock",
			States: []State{{
				ID:   "state_1",
				Name: "waiting",
				Kind: StateKindState,
				Transitions: []Transition{{
					ID:     "transition_1",
					Target: "done",
					Trigger: &Trigger{
						Kind:     TriggerAt,
						Value:    "deadline",
						Language: LanguageGo,
					},
				}},
			}},
		}},
	}
	output, err := NewZigBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"fn hsmc_trigger_zero(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) u64",
		"// Original go at trigger expression preserved for manual porting:",
		"// deadline",
		"hsm.at(hsmc_trigger_zero),",
		`hsm.target("done"),`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Zig output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{"at trigger lowered to after trigger", "hsm.after(hsmc_trigger_zero)"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Zig at output used stale fallback %q:\n%s", forbidden, text)
		}
	}
}

func TestZigBackendEmitsFallbackForForeignRawTimerTrigger(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:   "model_1",
			Name: "Clock",
			Initializer: &Initial{
				ID:      "initial_1",
				OwnerID: "model_1",
				Target:  "waiting",
			},
			States: []State{{
				ID:   "state_1",
				Name: "waiting",
				Kind: StateKindState,
				Transitions: []Transition{{
					ID:      "transition_1",
					OwnerID: "state_1",
					Target:  "../done",
					Trigger: &Trigger{
						Kind:     TriggerAfter,
						Value:    "time.Second",
						Language: LanguageGo,
					},
				}},
			}, {
				ID:   "state_2",
				Name: "done",
				Kind: StateKindState,
			}},
		}},
	}

	output, err := NewZigBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"fn hsmc_trigger_zero(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) u64",
		"return 0;",
		"// Original go after trigger expression preserved for manual porting:",
		"// time.Second",
		"hsm.after(hsmc_trigger_zero),",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Zig output missing %q:\n%s", needle, text)
		}
	}
}

func TestZigBackendEmitsAttributeAndOperationTriggers(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			States: []State{{
				ID:   "state_1",
				Name: "closed",
				Kind: StateKindState,
				Transitions: []Transition{
					{
						ID:     "transition_1",
						Target: "../open",
						Trigger: &Trigger{
							Kind:     TriggerOnSet,
							Value:    "count",
							IsString: true,
							Language: LanguageGo,
						},
					},
					{
						ID:     "transition_2",
						Target: "../open",
						Trigger: &Trigger{
							Kind:     TriggerOnCall,
							Value:    "approve",
							IsString: true,
							Language: LanguageGo,
						},
					},
				},
			}},
		}},
	}

	output, err := NewZigBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`hsm.onSet("count"),`,
		`hsm.onCall("approve"),`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Zig output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{
		"on_set trigger lowered to event trigger",
		"on_call trigger lowered to event trigger",
		`hsm.on("count")`,
		`hsm.on("approve")`,
		"Unsupported on_set trigger",
		"Unsupported on_call trigger",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Zig backend used stale trigger fallback %q:\n%s", forbidden, text)
		}
	}
}

func TestZigBackendEmitsAttributesAndOperations(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "operation_1",
			OwnerID:        "operation_decl_1",
			Kind:           BehaviorOperation,
			SourceLanguage: LanguageZig,
			Body:           "_ = ctx;\n_ = inst;\n_ = event;",
		}},
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			Attributes: []Attribute{
				{ID: "attribute_1", Name: "count", HasDefault: true, Default: "1", Language: LanguageGo},
				{ID: "attribute_2", Name: "ready", HasDefault: true, Default: "True", Language: LanguagePython},
				{ID: "attribute_3", Name: "label", HasDefault: true, Default: `"closed"`, Language: LanguageTS},
				{ID: "attribute_4", Name: "payload", Type: "Payload", Language: LanguageGo},
				{ID: "attribute_5", Name: "mode", Type: "Mode", Language: LanguageZig},
			},
			Operations: []Operation{{
				ID:   "operation_decl_1",
				Name: "approve",
				Behavior: &BehaviorRef{
					ID:   "operation_1",
					Kind: BehaviorOperation,
				},
			}},
		}},
	}
	AssignTargetABI(program, LanguageZig)

	output, err := NewZigBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`hsm.attribute("count", 1),`,
		`hsm.attribute("ready", true),`,
		`hsm.attribute("label", "closed"),`,
		`hsm.attribute("mode", Mode),`,
		`// Attribute "payload" preserved for manual porting; hsm.zig attributes require a Zig type or default value.`,
		`// Type: Payload`,
		`hsm.operation("approve", operation_1),`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Zig attribute/operation output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{
		`Original attribute count preserved`,
		`Original operation approve preserved`,
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Zig attribute/operation output used stale manual-port fallback %q:\n%s", forbidden, text)
		}
	}
}

func TestZigBackendEmitsOperationReferenceFallbackBehaviors(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			Initializer: &Initial{
				ID:      "initial_1",
				Target:  "closed",
				Effects: []BehaviorRef{{Kind: BehaviorEffect, Operation: "approve"}},
			},
			Operations: []Operation{{
				ID:   "operation_decl_1",
				Name: "approve",
			}},
			States: []State{{
				ID:   "state_1",
				Name: "closed",
				Kind: StateKindState,
				Behaviors: []BehaviorRef{
					{Kind: BehaviorEntry, Operation: "approve"},
					{Kind: BehaviorExit, Operation: "approve"},
					{Kind: BehaviorActivity, Operation: "approve"},
				},
				Transitions: []Transition{{
					ID:      "transition_1",
					OwnerID: "state_1",
					Target:  "../closed",
					Guard:   &BehaviorRef{Kind: BehaviorGuard, Operation: "approve"},
					Effects: []BehaviorRef{{Kind: BehaviorEffect, Operation: "approve"}},
				}},
			}},
		}},
	}

	output, err := NewZigBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	entryName := zigOperationReferenceFallbackName(BehaviorEntry, "approve")
	exitName := zigOperationReferenceFallbackName(BehaviorExit, "approve")
	activityName := zigOperationReferenceFallbackName(BehaviorActivity, "approve")
	guardName := zigOperationReferenceFallbackName(BehaviorGuard, "approve")
	effectName := zigOperationReferenceFallbackName(BehaviorEffect, "approve")
	for _, needle := range []string{
		"fn " + entryName + "(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void",
		"fn " + exitName + "(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void",
		"fn " + activityName + "(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void",
		"fn " + guardName + "(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) bool",
		"fn " + effectName + "(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void",
		`// Operation reference "approve" preserved for manual porting; hsm.zig behavior callbacks cannot call model operations without a state machine handle.`,
		"hsm.entry(" + entryName + "),",
		"hsm.exit(" + exitName + "),",
		"hsm.activity(" + activityName + "),",
		"hsm.guard(" + guardName + "),",
		"hsm.effect(" + effectName + "),",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Zig operation reference output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{
		`hsm.entry(approve)`,
		`hsm.exit(approve)`,
		`hsm.activity(approve)`,
		`hsm.guard(approve)`,
		`hsm.effect(approve)`,
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Zig operation reference emitted as operation identifier %q:\n%s", forbidden, text)
		}
	}
}

func TestZigBackendPreservesHistoryDefaultGuardAndEffects(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:          "model_1",
			Name:        "Door",
			Initializer: &Initial{ID: "initial_1", Target: "closed"},
			Operations: []Operation{
				{ID: "operation_decl_1", Name: "canResume"},
				{ID: "operation_decl_2", Name: "restore"},
			},
			States: []State{{
				ID:          "state_1",
				Name:        "closed",
				Kind:        StateKindState,
				Initializer: &Initial{ID: "initial_2", Target: "a"},
				States: []State{
					{ID: "state_2", Name: "a", Kind: StateKindState},
					{
						ID:   "state_3",
						Name: "history",
						Kind: StateKindShallowHistory,
						Transitions: []Transition{{
							ID:      "transition_1",
							OwnerID: "state_3",
							Target:  "../a",
							Guard:   &BehaviorRef{Kind: BehaviorGuard, Operation: "canResume"},
							Effects: []BehaviorRef{{Kind: BehaviorEffect, Operation: "restore"}},
						}},
					},
				},
			}},
		}},
	}

	output, err := NewZigBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	guardName := zigOperationReferenceFallbackName(BehaviorGuard, "canResume")
	effectName := zigOperationReferenceFallbackName(BehaviorEffect, "restore")
	for _, needle := range []string{
		"fn " + guardName + "(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) bool",
		"fn " + effectName + "(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void",
		`// History default transition guard/effects preserved for manual porting; hsm.zig history helpers accept only a target.`,
		"// hsm.guard(" + guardName + "),",
		"// hsm.effect(" + effectName + "),",
		`hsm.history("history", hsm.target("../a")),`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Zig history default output missing %q:\n%s", needle, text)
		}
	}
}

func TestZigBackendLowersPredicateWhenTriggerToGuard(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "trigger_1",
			OwnerID:        "transition_1",
			Kind:           BehaviorTrigger,
			TriggerKind:    TriggerWhen,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageZig,
			Body:           "return ready(inst);",
		}},
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			States: []State{{
				ID:   "state_1",
				Name: "closed",
				Kind: StateKindState,
				Transitions: []Transition{{
					ID:     "transition_1",
					Target: "../open",
					Trigger: &Trigger{
						Kind: TriggerWhen,
						Expr: &BehaviorRef{ID: "trigger_1"},
					},
				}},
			}},
		}},
	}
	AssignTargetABI(program, LanguageZig)

	output, err := NewZigBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"fn trigger_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) bool",
		"return ready(inst);",
		"// when trigger lowered to guard for hsm.zig",
		"hsm.guard(trigger_1),",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Zig output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Unsupported when trigger") {
		t.Fatalf("Zig backend did not wire predicate when trigger through guard:\n%s", text)
	}
}

func TestZigBackendUsesAdapterFullCodeBehavior(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageZig,
			Code: `fn entry_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void {
    _ = ctx;
    _ = inst;
    _ = event;
    adapter_format("entry");
}`,
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
	AssignTargetABI(program, LanguageZig)

	output, err := NewZigBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`fn entry_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void`,
		`adapter_format("entry")`,
		`hsm.entry(entry_1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Zig output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original go behavior") {
		t.Fatalf("Zig output commented adapter full code instead of using it:\n%s", text)
	}
}

func TestZigBackendRejectsAdapterFullCodeBehaviorNameChanges(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageZig,
			Code: `fn renamed_entry(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void {
    _ = ctx;
    _ = inst;
    _ = event;
}`,
		}},
	}
	AssignTargetABI(program, LanguageZig)

	_, err := NewZigBackend().Emit(context.Background(), program)
	if err == nil || !strings.Contains(err.Error(), `declares "renamed_entry", want compiler-owned name "entry_1"`) {
		t.Fatalf("error = %v, want compiler-owned Zig behavior name rejection", err)
	}
}

func TestZigBackendRejectsAdapterFullCodeBehaviorSignatureChanges(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "parameters",
			code: `fn entry_1(ctx: *hsm.Context, instance: *hsm.Instance, event: hsm.Event) void {
    _ = ctx;
    _ = instance;
    _ = event;
}`,
			want: `declares parameters "(ctx: *hsm.Context, instance: *hsm.Instance, event: hsm.Event)", want compiler-owned parameters "(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event)"`,
		},
		{
			name: "return type",
			code: `fn entry_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) bool {
    _ = ctx;
    _ = inst;
    _ = event;
    return true;
}`,
			want: `declares return type "bool", want compiler-owned return type "void"`,
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
					TargetLanguage: LanguageZig,
					Code:           tt.code,
				}},
			}
			AssignTargetABI(program, LanguageZig)

			_, err := NewZigBackend().Emit(context.Background(), program)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestZigBackendRejectsAdapterCodeThatIsNotFunctionDeclaration(t *testing.T) {
	cases := []struct {
		name string
		code string
	}{
		{
			name: "struct callback expression",
			code: `struct {
	fn call(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void {
		_ = ctx;
		_ = inst;
		_ = event;
	}
}.call`,
		},
		{
			name: "var declaration",
			code: `pub var entry_1: ?*const fn (*hsm.Context, *hsm.Instance, hsm.Event) void = null;`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := &Program{
				Behaviors: []Behavior{{
					ID:             "entry_1",
					OwnerID:        "state_1",
					Kind:           BehaviorEntry,
					SourceLanguage: LanguageGo,
					TargetLanguage: LanguageZig,
					Code:           tc.code,
				}},
			}
			AssignTargetABI(program, LanguageZig)

			_, err := NewZigBackend().Emit(context.Background(), program)
			if err == nil || !strings.Contains(err.Error(), `must be a compiler-owned function declaration`) {
				t.Fatalf("error = %v, want function declaration rejection", err)
			}
		})
	}
}
