package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestJavaBackendCompilesGoModel(t *testing.T) {
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageJava,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if program == nil || len(program.Models) != 1 {
		t.Fatalf("program = %#v, want one model", program)
	}
	text := string(output)
	for _, needle := range []string{
		"import com.stateforward.hsm.*;",
		"public final class GeneratedHsm",
		"private static void Entry1(Context ctx, Instance instance, Event event)",
		"// Original go behavior entry_1 preserved for manual porting:",
		"public static final Model DoorModel = Hsm.Define(",
		"Hsm.Initial(Hsm.Target(\"closed\"))",
		"Hsm.On(\"open\")",
		"Hsm.Entry(GeneratedHsm::Entry1)",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Java output missing %q:\n%s", needle, text)
		}
	}
}

func TestJavaBehaviorABI(t *testing.T) {
	guard := BehaviorABIFor(LanguageJava, BehaviorGuard)
	if guard.Language != LanguageJava || guard.ReturnType != "boolean" {
		t.Fatalf("guard ABI = %#v", guard)
	}
	timer := BehaviorABIForTrigger(LanguageJava, BehaviorTrigger, TriggerAfter)
	if timer.ReturnType != "java.time.Duration" {
		t.Fatalf("timer ABI = %#v", timer)
	}
	at := BehaviorABIForTrigger(LanguageJava, BehaviorTrigger, TriggerAt)
	if at.ReturnType != "java.time.Instant" {
		t.Fatalf("at ABI = %#v", at)
	}
	when := BehaviorABIForTrigger(LanguageJava, BehaviorTrigger, TriggerWhen)
	if when.ReturnType != "boolean" {
		t.Fatalf("when ABI = %#v", when)
	}
}

func TestJavaBackendEmitsAttributesWithPortableFallbacks(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			Attributes: []Attribute{
				{ID: "attribute_1", Name: "count", Type: "Integer.class", HasDefault: true, Default: "1", Language: LanguageJava},
				{ID: "attribute_2", Name: "label", HasDefault: true, Default: `"closed"`, Language: LanguageGo},
				{ID: "attribute_3", Name: "ready", HasDefault: true, Default: "True", Language: LanguagePython},
				{ID: "attribute_4", Name: "payload", Type: "Object.class", HasDefault: true, Default: "makePayload()", Language: LanguagePython},
				{ID: "attribute_5", Name: "typeOnly", Type: "String.class", Language: LanguageJava},
			},
		}},
	}

	output, err := NewJavaBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`Hsm.Attribute("count", Integer.class, 1)`,
		`Hsm.Attribute("label", "closed")`,
		`Hsm.Attribute("ready", true)`,
		`/* Type: Object.class */ Hsm.Attribute("payload", null /* Original python default preserved for manual porting: makePayload() */)`,
		`Hsm.Attribute("typeOnly", String.class)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Java attribute output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{
		`Hsm.Attribute("ready", True)`,
		`Hsm.Attribute("payload", Object.class, makePayload())`,
		`Hsm.Attribute("payload", Object.class, null`,
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Java attribute output emitted invalid foreign default %q:\n%s", forbidden, text)
		}
	}
}

func TestJavaBackendSynthesizesOperationReferenceBehaviors(t *testing.T) {
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

	output, err := NewJavaBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`Hsm.Entry((ctx, instance, event) -> Hsm.Call(ctx, instance, "approve"))`,
		`Hsm.Guard((ctx, instance, event) -> Boolean.TRUE.equals(Hsm.Call(ctx, instance, "approve")))`,
		`Hsm.Effect((ctx, instance, event) -> Hsm.Call(ctx, instance, "approve"))`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Java operation reference output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{
		`Hsm.Entry(GeneratedHsm::Generated)`,
		`Hsm.Guard(GeneratedHsm::Generated)`,
		`Hsm.Effect(GeneratedHsm::Generated)`,
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Java operation reference emitted as generated method reference %q:\n%s", forbidden, text)
		}
	}
}

func TestJavaBackendEmitsPredicateWhenBehavior(t *testing.T) {
	program := &Program{
		SourceLanguage: LanguageGo,
		Behaviors: []Behavior{{
			ID:             "trigger_1",
			OwnerID:        "transition_1",
			Kind:           BehaviorTrigger,
			TriggerKind:    TriggerWhen,
			SourceLanguage: LanguageGo,
			Body:           "return true",
		}},
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			Initializer: &Initial{
				OwnerID: "model_1",
				ID:      "initial_1",
				Target:  "closed",
			},
			States: []State{
				{
					ID:   "state_1",
					Name: "closed",
					Kind: StateKindState,
					Transitions: []Transition{{
						OwnerID: "state_1",
						ID:      "transition_1",
						Target:  "../open",
						Trigger: &Trigger{
							Kind: TriggerWhen,
							Expr: &BehaviorRef{ID: "trigger_1", Kind: BehaviorTrigger},
						},
					}},
				},
				{ID: "state_2", Name: "open", Kind: StateKindState},
			},
		}},
	}
	AssignTargetABI(program, LanguageJava)

	output, err := NewJavaBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"private static boolean Trigger1(Context ctx, Instance instance, Event event)",
		"Hsm.When(GeneratedHsm::Trigger1)",
		"return false;",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Java output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "java.util.concurrent.CompletionStage") || strings.Contains(text, "Hsm.When(() -> false)") {
		t.Fatalf("Java output used stale predicate When fallback:\n%s", text)
	}
}

func TestJavaBackendUsesAdapterFullCodeBehavior(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageJava,
			Code: `private static void Entry1(Context ctx, Instance instance, Event event) {
    AdapterFormat.record("entry");
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
	AssignTargetABI(program, LanguageJava)

	output, err := NewJavaBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`private static void Entry1(Context ctx, Instance instance, Event event)`,
		`AdapterFormat.record("entry");`,
		`Hsm.Entry(GeneratedHsm::Entry1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Java output missing %q:\n%s", needle, text)
		}
	}
}

func TestJavaBackendRejectsAdapterFullCodeBehaviorNameChanges(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageJava,
			Code: `private static void RenamedEntry(Context ctx, Instance instance, Event event) {
    AdapterFormat.record("entry");
}`,
		}},
	}
	AssignTargetABI(program, LanguageJava)

	_, err := NewJavaBackend().Emit(context.Background(), program)
	if err == nil || !strings.Contains(err.Error(), `declares "RenamedEntry", want compiler-owned name "Entry1"`) {
		t.Fatalf("error = %v, want compiler-owned Java behavior name rejection", err)
	}
}

func TestJavaBackendRejectsAdapterFullCodeBehaviorSignatureChanges(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "return type",
			code: `private static boolean Entry1(Context ctx, Instance instance, Event event) {
    return true;
}`,
			want: `declares return type "boolean", want compiler-owned return type "void"`,
		},
		{
			name: "parameters",
			code: `private static void Entry1() {
}`,
			want: `declares parameters "()", want compiler-owned parameters "(Context ctx, Instance instance, Event event)"`,
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
					TargetLanguage: LanguageJava,
					Code:           tt.code,
				}},
			}
			AssignTargetABI(program, LanguageJava)

			_, err := NewJavaBackend().Emit(context.Background(), program)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestJavaBackendRejectsAdapterCodeThatIsNotMethodDeclaration(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageJava,
			Code:           `(ctx, instance, event) -> AdapterFormat.record("entry")`,
		}},
	}
	AssignTargetABI(program, LanguageJava)

	_, err := NewJavaBackend().Emit(context.Background(), program)
	if err == nil || !strings.Contains(err.Error(), `must be a compiler-owned method declaration`) {
		t.Fatalf("error = %v, want method declaration rejection", err)
	}
}
