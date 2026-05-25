package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestRustBackendEmitsModelStructureAndCommentedForeignBehaviors(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageRust,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"use std::future::Future;",
		"pub struct HsmcInstance;",
		"impl hsm::Instance for HsmcInstance",
		"fn entry_1(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) -> Pin<Box<dyn Future<Output = ()> + Send>>",
		"// Original go behavior entry_1 preserved for manual porting:",
		`// sm.log = append(sm.log, record("closed"))`,
		"Box::pin(async {})",
		"fn guard_1(ctx: &hsm::Context, instance: &HsmcInstance, event: &hsm::Event) -> bool",
		"false",
		"pub fn door_model() -> hsm::Model<HsmcInstance> {",
		`hsm::define!("Door",`,
		`hsm::initial!(hsm::target!("closed")),`,
		`hsm::state!("closed",`,
		`hsm::entry!(entry_1),`,
		`hsm::transition!(hsm::on!("open"), hsm::guard!(guard_1), hsm::target!("../open"), hsm::effect!(effect_1)),`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Rust output missing %q:\n%s", needle, text)
		}
	}
}

func TestRustBackendDoesNotDuplicateCompilerOwnedImports(t *testing.T) {
	program := &Program{
		SourceLanguage: LanguageRust,
		Imports: []Import{
			{Path: "std::any::Any", Language: LanguageRust},
			{Path: "std::future::Future", Language: LanguageRust},
			{Path: "std::pin::Pin", Language: LanguageRust},
			{Path: "std::time::Duration", Language: LanguageRust},
		},
		Models: []Model{{ID: "model_1", Name: "Door", Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"}, States: []State{{ID: "state_1", Name: "closed", Kind: StateKindState}}}},
	}

	output, err := NewRustBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"use std::any::Any;",
		"use std::future::Future;",
		"use std::pin::Pin;",
		"use std::time::Duration;",
	} {
		if strings.Count(text, needle) != 1 {
			t.Fatalf("Rust output should emit %q exactly once:\n%s", needle, text)
		}
	}
}

func TestRustTargetABI(t *testing.T) {
	guard := BehaviorABIFor(LanguageRust, BehaviorGuard)
	if guard == nil || guard.ReturnType != "bool" || !strings.Contains(guard.Signature, "hsm::Context") {
		t.Fatalf("rust guard ABI = %#v", guard)
	}
	timer := BehaviorABIForTrigger(LanguageRust, BehaviorTrigger, TriggerAfter)
	if timer == nil || timer.ReturnType != "Duration" {
		t.Fatalf("rust timer ABI = %#v", timer)
	}
	at := BehaviorABIForTrigger(LanguageRust, BehaviorTrigger, TriggerAt)
	if at == nil || at.ReturnType != "std::time::SystemTime" {
		t.Fatalf("rust at ABI = %#v", at)
	}
	when := BehaviorABIForTrigger(LanguageRust, BehaviorTrigger, TriggerWhen)
	if when == nil || when.ReturnType != "bool" {
		t.Fatalf("rust when ABI = %#v", when)
	}
}

func TestRustBackendEmitsAttributesWithPortableFallbacks(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			Attributes: []Attribute{
				{ID: "attribute_1", Name: "count", Type: "i32", HasDefault: true, Default: "1", Language: LanguageRust},
				{ID: "attribute_2", Name: "label", HasDefault: true, Default: `"closed"`, Language: LanguageGo},
				{ID: "attribute_3", Name: "flag", Type: "bool", HasDefault: true, Default: "True", Language: LanguagePython},
				{ID: "attribute_4", Name: "type_only", Type: "String", Language: LanguageRust},
				{ID: "attribute_5", Name: "maybe", HasDefault: true, Default: "None", Language: LanguagePython},
			},
		}},
	}

	output, err := NewRustBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`// type: i32`,
		`hsm::attribute("count", 1)`,
		`hsm::attribute("label", "closed")`,
		`// type: bool`,
		`hsm::attribute("flag", true)`,
		`// type: String`,
		`// Attribute "type_only" preserved for manual porting; hsm.rs attributes require a default value.`,
		`// default: None`,
		`// Attribute "maybe" preserved for manual porting; hsm.rs attributes require a default value.`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Rust attribute output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, `hsm::attribute("flag", True)`) ||
		strings.Contains(text, `hsm::attribute("maybe", None)`) ||
		strings.Contains(text, `hsm.rs does not expose an attribute partial`) {
		t.Fatalf("Rust attribute output used stale or invalid emission:\n%s", text)
	}
}

func TestRustBackendUsesRuntimeOnSetPartial(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			States: []State{{
				ID:   "state_1",
				Name: "closed",
				Kind: StateKindState,
				Transitions: []Transition{{
					ID:      "transition_1",
					OwnerID: "state_1",
					Trigger: &Trigger{
						Kind:     TriggerOnSet,
						Value:    "count",
						IsString: true,
					},
					Target: "../open",
				}, {
					ID:      "transition_2",
					OwnerID: "state_1",
					Trigger: &Trigger{
						Kind:     TriggerOnCall,
						Value:    "approve",
						IsString: true,
					},
					Target: "../open",
				}},
			}},
		}},
	}

	output, err := NewRustBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`hsm::transition!(hsm::on_set("count"), hsm::target!("../open"))`,
		`hsm::transition!(hsm::on_call("approve"), hsm::target!("../open"))`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Rust trigger output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{
		"on_set trigger lowered to event trigger",
		"on_call trigger lowered to event trigger",
		`hsm::on!("approve")`,
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Rust trigger output used stale fallback %q:\n%s", forbidden, text)
		}
	}
}

func TestRustBackendUsesOperationReferenceHelpersInBehaviorPartials(t *testing.T) {
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
				ID:        "state_1",
				Name:      "closed",
				Kind:      StateKindState,
				Behaviors: []BehaviorRef{{Kind: BehaviorEntry, Operation: "approve"}},
				Transitions: []Transition{{
					ID:      "transition_1",
					OwnerID: "state_1",
					Target:  "../closed",
					Guard:   &BehaviorRef{Kind: BehaviorGuard, Operation: "approve"},
				}},
			}},
		}},
	}

	output, err := NewRustBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`hsm::entry_operation("approve")`,
		`hsm::guard_operation_ref("approve")`,
		`hsm::effect_operation("approve")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Rust operation reference output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{
		`hsm::entry!(approve)`,
		`hsm::guard!(approve)`,
		`hsm::effect!(approve)`,
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Rust operation reference emitted as behavior function %q:\n%s", forbidden, text)
		}
	}
}

func TestRustBackendEmitsDefaultReturnForUntranslatedTimerTrigger(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "trigger_1",
			OwnerID:        "state_1",
			Kind:           BehaviorTrigger,
			TriggerKind:    TriggerAfter,
			SourceLanguage: LanguageGo,
			Body:           "return time.Second",
		}},
	}
	AssignTargetABI(program, LanguageRust)

	output, err := NewRustBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"fn trigger_1(ctx: &hsm::Context, instance: &HsmcInstance, event: &hsm::Event) -> Duration",
		"// Original go behavior trigger_1 preserved for manual porting:",
		"Duration::from_secs(0)",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Rust output missing %q:\n%s", needle, text)
		}
	}
}

func TestRustBackendUsesFallbackForForeignRawTimerTrigger(t *testing.T) {
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

	output, err := NewRustBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"fn hsmc_zero_duration(ctx: &hsm::Context, instance: &HsmcInstance, event: &hsm::Event) -> Duration",
		"/* Original go after trigger expression preserved for manual porting:",
		" * time.Second",
		"*/ hsmc_zero_duration",
		"hsm::after!(/* Original go after trigger expression preserved for manual porting:",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Rust output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "|| {") {
		t.Fatalf("Rust output used zero-arg closure for timer fallback:\n%s", text)
	}
}

func TestRustBackendUsesAdapterFullCodeBehavior(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageRust,
			Code: `fn entry_1(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) -> Pin<Box<dyn Future<Output = ()> + Send>> {
	let _ = (ctx, instance, event);
	adapter_format("entry");
	Box::pin(async {})
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
	AssignTargetABI(program, LanguageRust)

	output, err := NewRustBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`fn entry_1(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) -> Pin<Box<dyn Future<Output = ()> + Send>>`,
		`adapter_format("entry")`,
		`hsm::entry!(entry_1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Rust output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original go behavior") {
		t.Fatalf("Rust output commented adapter full code instead of using it:\n%s", text)
	}
}

func TestRustBackendRejectsAdapterFullCodeBehaviorNameChanges(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageRust,
			Code: `fn renamed_entry(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) -> Pin<Box<dyn Future<Output = ()> + Send>> {
	let _ = (ctx, instance, event);
	Box::pin(async {})
}`,
		}},
	}
	AssignTargetABI(program, LanguageRust)

	_, err := NewRustBackend().Emit(context.Background(), program)
	if err == nil || !strings.Contains(err.Error(), `declares "renamed_entry", want compiler-owned name "entry_1"`) {
		t.Fatalf("error = %v, want compiler-owned Rust behavior name rejection", err)
	}
}

func TestRustBackendRejectsAdapterFullCodeBehaviorSignatureChanges(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "parameters",
			code: `fn entry_1(ctx: &hsm::Context, instance: &HsmcInstance, event: &hsm::Event) -> Pin<Box<dyn Future<Output = ()> + Send>> {
	Box::pin(async {})
}`,
			want: `declares parameters "(ctx: &hsm::Context, instance: &HsmcInstance, event: &hsm::Event)", want compiler-owned parameters "(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event)"`,
		},
		{
			name: "return type",
			code: `fn entry_1(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) {
	let _ = (ctx, instance, event);
}`,
			want: `declares return type "()", want compiler-owned return type "Pin<Box<dyn Future<Output = ()> + Send>>"`,
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
					TargetLanguage: LanguageRust,
					Code:           tt.code,
				}},
			}
			AssignTargetABI(program, LanguageRust)

			_, err := NewRustBackend().Emit(context.Background(), program)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestRustBackendRejectsAdapterCodeThatIsNotFunctionDeclaration(t *testing.T) {
	cases := []struct {
		name string
		code string
	}{
		{
			name: "closure expression",
			code: `|ctx, instance, event| {
	let _ = (ctx, instance, event);
}`,
		},
		{
			name: "static declaration",
			code: `pub static entry_1: Option<fn(&hsm::Context, &mut HsmcInstance, &hsm::Event)> = None;`,
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
					TargetLanguage: LanguageRust,
					Code:           tc.code,
				}},
			}
			AssignTargetABI(program, LanguageRust)

			_, err := NewRustBackend().Emit(context.Background(), program)
			if err == nil || !strings.Contains(err.Error(), `must be a compiler-owned function declaration`) {
				t.Fatalf("error = %v, want function declaration rejection", err)
			}
		})
	}
}
