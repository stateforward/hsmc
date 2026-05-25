package hsmc

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestCPPBackendEmitsModelStructureAndCommentedForeignBehaviors(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageCPP,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`#include "hsm/hsm.hpp"`,
		"#include <chrono>",
		"namespace generated_hsm",
		"using namespace hsm;",
		"static constexpr auto entry_1 = [](auto& signal, auto& instance, const auto& event) -> void",
		"// Original go behavior entry_1 preserved for manual porting:",
		`// sm.log = append(sm.log, record("closed"))`,
		"static constexpr auto guard_1 = [](auto& signal, auto& instance, const auto& event) -> bool",
		"return false;",
		"static constexpr auto door_model = hsm::define(",
		`hsm::initial(hsm::target("/Door/closed"))`,
		`hsm::state("closed"`,
		"hsm::entry(entry_1)",
		`hsm::transition(hsm::on("open"), hsm::guard(guard_1), hsm::target("/Door/open"), hsm::effect(effect_1))`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C++ output missing %q:\n%s", needle, text)
		}
	}
}

func TestCPPTargetABI(t *testing.T) {
	guard := BehaviorABIFor(LanguageCPP, BehaviorGuard)
	if guard == nil || guard.ReturnType != "bool" || !strings.Contains(guard.Signature, "auto& signal") {
		t.Fatalf("C++ guard ABI = %#v", guard)
	}
	timer := BehaviorABIForTrigger(LanguageCPP, BehaviorTrigger, TriggerAfter)
	if timer == nil || timer.ReturnType != "std::chrono::milliseconds" {
		t.Fatalf("C++ after ABI = %#v", timer)
	}
	at := BehaviorABIForTrigger(LanguageCPP, BehaviorTrigger, TriggerAt)
	if at == nil || at.ReturnType != "std::chrono::system_clock::time_point" {
		t.Fatalf("C++ at ABI = %#v", at)
	}
	operation := BehaviorABIFor(LanguageCPP, BehaviorOperation)
	if operation == nil || operation.ReturnType != "void" || operation.Signature != "void ()" || len(operation.Parameters) != 0 {
		t.Fatalf("C++ operation ABI = %#v", operation)
	}
}

func TestCPPBackendEmitsAttributesWithPortableFallbacks(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			Attributes: []Attribute{
				{ID: "attribute_1", Name: "count", Type: "int", HasDefault: true, Default: "1", Language: LanguageCPP},
				{ID: "attribute_2", Name: "label", Type: "String", HasDefault: true, Default: `"closed"`, Language: LanguageJava},
				{ID: "attribute_3", Name: "flag", Type: "bool", HasDefault: true, Default: "True", Language: LanguagePython},
				{ID: "attribute_4", Name: "type_only", Type: "i32", Language: LanguageRust},
				{ID: "attribute_5", Name: "payload", HasDefault: true, Default: "makePayload()", Language: LanguagePython},
				{ID: "attribute_6", Name: "maybe", HasDefault: true, Default: "None", Language: LanguagePython},
			},
		}},
	}

	output, err := NewCPPBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`hsm::attribute<int>("count", 1)`,
		`hsm::attribute<std::string>("label", "closed")`,
		`hsm::attribute<bool>("flag", true)`,
		`hsm::attribute<std::int32_t>("type_only")`,
		`// Default: makePayload()`,
		`// Original attribute payload preserved for manual porting.`,
		`hsm::attribute("maybe", nullptr)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C++ attribute output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{
		`hsm::attribute<bool>("flag", True)`,
		`hsm::attribute("payload", makePayload())`,
		`hsm::attribute("maybe", None)`,
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("C++ attribute output emitted invalid foreign default %q:\n%s", forbidden, text)
		}
	}
}

func TestCPPBackendQuotesOperationReferencesInBehaviorPartials(t *testing.T) {
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
			States: []State{{ID: "state_1", Name: "closed", Kind: StateKindState}},
		}},
	}

	output, err := NewCPPBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, `hsm::effect("approve")`) {
		t.Fatalf("C++ operation reference effect should be quoted:\n%s", text)
	}
	if strings.Contains(text, `hsm::effect(approve)`) {
		t.Fatalf("C++ operation reference effect emitted as identifier:\n%s", text)
	}
}

func TestCPPBackendUsesAdapterFullCodeBehavior(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageCPP,
			Code:           `[](auto& signal, auto& instance, const auto& event) -> void { adapter_format("entry"); }`,
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
	AssignTargetABI(program, LanguageCPP)

	output, err := NewCPPBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`static constexpr auto entry_1 = [](auto& signal, auto& instance, const auto& event) -> void { adapter_format("entry"); };`,
		`hsm::entry(entry_1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C++ output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original go behavior") {
		t.Fatalf("C++ output commented adapter full code instead of using it:\n%s", text)
	}
}

func TestCPPBackendUsesAdapterFullDeclarationBehavior(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageCPP,
			Code:           `static constexpr auto entry_1 = [](auto& signal, auto& instance, const auto& event) -> void { adapter_format("entry"); };`,
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
	AssignTargetABI(program, LanguageCPP)

	output, err := NewCPPBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`static constexpr auto entry_1 = [](auto& signal, auto& instance, const auto& event) -> void { adapter_format("entry"); };`,
		`hsm::entry(entry_1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C++ output missing %q:\n%s", needle, text)
		}
	}
}

func TestCPPBackendRejectsAdapterFullCodeBehaviorNameChanges(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		kind       BehaviorKind
		code       string
		wantName   string
		wantTarget string
	}{
		{
			name:       "lambda declaration",
			id:         "entry_1",
			kind:       BehaviorEntry,
			code:       `static constexpr auto renamed_entry = [](auto& signal, auto& instance, const auto& event) -> void { adapter_format("entry"); };`,
			wantName:   "renamed_entry",
			wantTarget: "entry_1",
		},
		{
			name:       "operation function declaration",
			id:         "operation_1",
			kind:       BehaviorOperation,
			code:       `static void renamed_operation() { adapter_format("approve"); }`,
			wantName:   "renamed_operation",
			wantTarget: "operation_1",
		},
		{
			name:       "type declaration",
			id:         "entry_1",
			kind:       BehaviorEntry,
			code:       `struct renamed_entry {};`,
			wantName:   "renamed_entry",
			wantTarget: "entry_1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := &Program{
				Behaviors: []Behavior{{
					ID:             tt.id,
					OwnerID:        "state_1",
					Kind:           tt.kind,
					SourceLanguage: LanguageGo,
					TargetLanguage: LanguageCPP,
					Code:           tt.code,
				}},
			}
			AssignTargetABI(program, LanguageCPP)

			_, err := NewCPPBackend().Emit(context.Background(), program)
			want := fmt.Sprintf(`declares %q, want compiler-owned name %q`, tt.wantName, tt.wantTarget)
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want compiler-owned C++ behavior name rejection %q", err, want)
			}
		})
	}
}

func TestCPPBackendRejectsAdapterFullCodeBehaviorNonCallableDeclarations(t *testing.T) {
	tests := []struct {
		name string
		id   string
		kind BehaviorKind
		code string
		want string
	}{
		{
			name: "entry auto declaration",
			id:   "entry_1",
			kind: BehaviorEntry,
			code: `static constexpr auto entry_1 = 1;`,
			want: `cpp behavior full code for "entry_1" must be a compiler-owned lambda declaration or lambda expression; use body for statements`,
		},
		{
			name: "operation auto declaration",
			id:   "operation_1",
			kind: BehaviorOperation,
			code: `static constexpr auto operation_1 = 1;`,
			want: `cpp behavior full code for "operation_1" must be a compiler-owned lambda declaration or lambda expression; use body for statements`,
		},
		{
			name: "same-name type declaration",
			id:   "entry_1",
			kind: BehaviorEntry,
			code: `struct entry_1 {};`,
			want: `cpp behavior full code for "entry_1" must be a compiler-owned callable declaration, not a type or alias declaration`,
		},
		{
			name: "same-name alias declaration",
			id:   "entry_1",
			kind: BehaviorEntry,
			code: `using entry_1 = int;`,
			want: `cpp behavior full code for "entry_1" must be a compiler-owned callable declaration, not a type or alias declaration`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := &Program{
				Behaviors: []Behavior{{
					ID:             tt.id,
					OwnerID:        "state_1",
					Kind:           tt.kind,
					SourceLanguage: LanguageGo,
					TargetLanguage: LanguageCPP,
					Code:           tt.code,
				}},
			}
			AssignTargetABI(program, LanguageCPP)

			_, err := NewCPPBackend().Emit(context.Background(), program)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestCPPBackendRejectsAdapterFullCodeBehaviorSignatureChanges(t *testing.T) {
	tests := []struct {
		name string
		code string
		kind BehaviorKind
		want string
	}{
		{
			name: "declaration parameters",
			code: `static constexpr auto entry_1 = [](auto& signal) -> void { adapter_format("entry"); };`,
			kind: BehaviorEntry,
			want: `declares parameters "(auto& signal)", want compiler-owned parameters "(auto& signal, auto& instance, const auto& event)"`,
		},
		{
			name: "declaration return type",
			code: `static constexpr auto entry_1 = [](auto& signal, auto& instance, const auto& event) -> bool { return true; };`,
			kind: BehaviorEntry,
			want: `declares return type "bool", want compiler-owned return type "void"`,
		},
		{
			name: "expression parameters",
			code: `[](auto& signal) -> void { adapter_format("entry"); }`,
			kind: BehaviorEntry,
			want: `declares parameters "(auto& signal)", want compiler-owned parameters "(auto& signal, auto& instance, const auto& event)"`,
		},
		{
			name: "guard omitted return type",
			code: `[](auto& signal, auto& instance, const auto& event) { return true; }`,
			kind: BehaviorGuard,
			want: `omits return type, want compiler-owned return type "bool"`,
		},
		{
			name: "operation function declaration parameters",
			code: `static void entry_1(auto& signal) { adapter_format("approve"); }`,
			kind: BehaviorOperation,
			want: `declares parameters "(auto& signal)", want compiler-owned parameters "()"`,
		},
		{
			name: "operation function declaration return type",
			code: `static bool entry_1() { return true; }`,
			kind: BehaviorOperation,
			want: `declares return type "bool", want compiler-owned return type "void"`,
		},
		{
			name: "non-operation function declaration",
			code: `static void entry_1(auto& signal, auto& instance, const auto& event) { adapter_format("entry"); }`,
			kind: BehaviorEntry,
			want: `declares function parameters "(auto& signal, auto& instance, const auto& event)", want compiler-owned lambda parameters "(auto& signal, auto& instance, const auto& event)"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := &Program{
				Behaviors: []Behavior{{
					ID:             "entry_1",
					OwnerID:        "state_1",
					Kind:           tt.kind,
					SourceLanguage: LanguageGo,
					TargetLanguage: LanguageCPP,
					Code:           tt.code,
				}},
			}
			AssignTargetABI(program, LanguageCPP)

			_, err := NewCPPBackend().Emit(context.Background(), program)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}
