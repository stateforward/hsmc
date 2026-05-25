package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestCSharpBackendEmitsModelStructureAndCommentedForeignBehaviors(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageCSharp,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"using Stateforward.Hsm;",
		"public static class GeneratedHsm",
		"private static void Entry1(Context ctx, Instance instance, Event @event)",
		"// Original go behavior entry_1 preserved for manual porting:",
		`// sm.log = append(sm.log, record("closed"))`,
		"private static bool Guard1(Context ctx, Instance instance, Event @event)",
		"return false;",
		"public static readonly Model DoorModel = Hsm.Define(",
		`Hsm.Initial(`,
		`Hsm.Target("closed")`,
		`Hsm.State(`,
		`Hsm.Entry<Instance>(Entry1)`,
		`Hsm.Transition(`,
		`Hsm.On("open")`,
		`Hsm.Guard<Instance>(Guard1)`,
		`Hsm.Effect<Instance>(Effect1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C# output missing %q:\n%s", needle, text)
		}
	}
}

func TestCSharpTargetABI(t *testing.T) {
	guard := BehaviorABIFor(LanguageCSharp, BehaviorGuard)
	if guard == nil || guard.ReturnType != "bool" || !strings.Contains(guard.Signature, "Context ctx") {
		t.Fatalf("C# guard ABI = %#v", guard)
	}
	timer := BehaviorABIForTrigger(LanguageCSharp, BehaviorTrigger, TriggerAfter)
	if timer == nil || timer.ReturnType != "System.TimeSpan" {
		t.Fatalf("C# after ABI = %#v", timer)
	}
	when := BehaviorABIForTrigger(LanguageCSharp, BehaviorTrigger, TriggerWhen)
	if when == nil || when.ReturnType != "System.Threading.Tasks.Task" || len(when.Parameters) != 4 {
		t.Fatalf("C# when ABI = %#v", when)
	}
}

func TestCSharpBackendPreservesTypedAttributes(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:          "model_1",
			Name:        "Door",
			Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
			Attributes: []Attribute{
				{ID: "attribute_1", Name: "count", Type: "typeof(int)", HasDefault: true, Default: "1"},
				{ID: "attribute_2", Name: "label", Type: "string"},
				{ID: "attribute_3", Name: "payload", HasDefault: true, Default: "new object()"},
			},
			States: []State{{ID: "state_1", Name: "closed", Kind: StateKindState}},
		}},
	}

	output, err := NewCSharpBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`Hsm.Attribute<int>("count", 1)`,
		`Hsm.Attribute<string>("label")`,
		`Hsm.Attribute<object>("payload", new object())`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C# output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, `Hsm.Attribute<object>("count", 1)`) {
		t.Fatalf("C# output dropped typed attribute:\n%s", text)
	}
}

func TestCSharpBackendEmitsAttributesWithPortableFallbacks(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			Attributes: []Attribute{
				{ID: "attribute_1", Name: "count", Type: "typeof(int)", HasDefault: true, Default: "1", Language: LanguageCSharp},
				{ID: "attribute_2", Name: "label", HasDefault: true, Default: `"closed"`, Language: LanguageGo},
				{ID: "attribute_3", Name: "ready", HasDefault: true, Default: "True", Language: LanguagePython},
				{ID: "attribute_4", Name: "payload", Type: "object", HasDefault: true, Default: "makePayload()", Language: LanguagePython},
				{ID: "attribute_5", Name: "type_only", Type: "string", Language: LanguageCSharp},
			},
		}},
	}

	output, err := NewCSharpBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`Hsm.Attribute<int>("count", 1)`,
		`Hsm.Attribute<object>("label", "closed")`,
		`Hsm.Attribute<object>("ready", true)`,
		`// Default: makePayload()`,
		`Hsm.Attribute<object>("payload")`,
		`Hsm.Attribute<string>("type_only")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C# attribute output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{
		`Hsm.Attribute<object>("ready", True)`,
		`Hsm.Attribute<object>("payload", makePayload())`,
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("C# attribute output emitted invalid foreign default %q:\n%s", forbidden, text)
		}
	}
}

func TestCSharpBackendSynthesizesOperationReferenceBehaviors(t *testing.T) {
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

	output, err := NewCSharpBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`Hsm.Entry<Instance>((ctx, instance, @event) => Hsm.Call(ctx, instance, "approve"))`,
		`Hsm.Guard<Instance>((ctx, instance, @event) => Hsm.Call(ctx, instance, "approve") is true)`,
		`Hsm.Effect<Instance>((ctx, instance, @event) => Hsm.Call(ctx, instance, "approve"))`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C# operation reference output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{
		`Hsm.Entry<Instance>(approve)`,
		`Hsm.Guard<Instance>(approve)`,
		`Hsm.Effect<Instance>(approve)`,
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("C# operation reference emitted as identifier %q:\n%s", forbidden, text)
		}
	}
}

func TestCSharpBackendUsesAdapterFullCodeBehavior(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageCSharp,
			Code:           `(ctx, instance, @event) => System.GC.KeepAlive("entry")`,
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
	AssignTargetABI(program, LanguageCSharp)

	output, err := NewCSharpBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`private static readonly Operation<Instance> Entry1 = (ctx, instance, @event) => System.GC.KeepAlive("entry");`,
		`Hsm.Entry<Instance>(Entry1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C# output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original go behavior") {
		t.Fatalf("C# output commented adapter full code instead of using it:\n%s", text)
	}
}

func TestCSharpBackendUsesAdapterFullMethodDeclaration(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageCSharp,
			Code: `private static void Entry1(Context ctx, Instance instance, Event @event)
{
    System.GC.KeepAlive("entry");
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
	AssignTargetABI(program, LanguageCSharp)

	output, err := NewCSharpBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`private static void Entry1(Context ctx, Instance instance, Event @event)`,
		`System.GC.KeepAlive("entry");`,
		`Hsm.Entry<Instance>(Entry1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C# output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "private static readonly Operation<Instance> Entry1 = private static void Entry1") {
		t.Fatalf("C# output wrapped a full method declaration as a delegate expression:\n%s", text)
	}
}

func TestCSharpBackendUsesAdapterFullFieldDeclaration(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageCSharp,
			Code:           `private static readonly Operation<Instance> Entry1 = (ctx, instance, @event) => System.GC.KeepAlive("entry");`,
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
	AssignTargetABI(program, LanguageCSharp)

	output, err := NewCSharpBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, `private static readonly Operation<Instance> Entry1 = (ctx, instance, @event) => System.GC.KeepAlive("entry");`) {
		t.Fatalf("C# output missing full field declaration:\n%s", text)
	}
	if strings.Contains(text, `private static readonly Operation<Instance> Entry1 = private static readonly Operation<Instance> Entry1`) {
		t.Fatalf("C# output wrapped a full field declaration as a delegate expression:\n%s", text)
	}
}

func TestCSharpBackendRejectsAdapterFullCodeBehaviorNameChanges(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{
			name: "method declaration",
			code: `private static void RenamedEntry(Context ctx, Instance instance, Event @event)
{
    System.GC.KeepAlive("entry");
}`,
		},
		{
			name: "field declaration",
			code: `private static readonly Operation<Instance> RenamedEntry = (ctx, instance, @event) => System.GC.KeepAlive("entry");`,
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
					TargetLanguage: LanguageCSharp,
					Code:           tt.code,
				}},
			}
			AssignTargetABI(program, LanguageCSharp)

			_, err := NewCSharpBackend().Emit(context.Background(), program)
			if err == nil || !strings.Contains(err.Error(), `declares "RenamedEntry", want compiler-owned name "Entry1"`) {
				t.Fatalf("error = %v, want compiler-owned C# behavior name rejection", err)
			}
		})
	}
}

func TestCSharpBackendRejectsAdapterFullCodeBehaviorSignatureChanges(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "return type",
			code: `private static bool Entry1(Context ctx, Instance instance, Event @event)
{
    return true;
}`,
			want: `declares return type "bool", want compiler-owned return type "void"`,
		},
		{
			name: "parameters",
			code: `private static void Entry1()
{
}`,
			want: `declares parameters "()", want compiler-owned parameters "(Context ctx, Instance instance, Event @event)"`,
		},
		{
			name: "lambda parameters",
			code: `(ctx, instance) => System.GC.KeepAlive("entry")`,
			want: `declares lambda parameters "(ctx, instance)", want compiler-owned parameters "(ctx, instance, @event)"`,
		},
		{
			name: "field declaration lambda parameters",
			code: `private static readonly Operation<Instance> Entry1 = (ctx, instance) => System.GC.KeepAlive("entry");`,
			want: `declares lambda parameters "(ctx, instance)", want compiler-owned parameters "(ctx, instance, @event)"`,
		},
		{
			name: "single lambda parameter",
			code: `ctx => System.GC.KeepAlive("entry")`,
			want: `declares lambda parameters "(ctx)", want compiler-owned parameters "(ctx, instance, @event)"`,
		},
		{
			name: "async lambda parameters",
			code: `async (ctx, instance) => await System.Threading.Tasks.Task.CompletedTask`,
			want: `declares lambda parameters "(ctx, instance)", want compiler-owned parameters "(ctx, instance, @event)"`,
		},
		{
			name: "non-callable field declaration",
			code: `private static readonly Operation<Instance> Entry1 = null;`,
			want: `must be a compiler-owned callable declaration`,
		},
		{
			name: "same-name class declaration",
			code: `private class Entry1
{
}`,
			want: `must be a compiler-owned callable declaration`,
		},
		{
			name: "same-name delegate declaration",
			code: `private delegate void Entry1(Context ctx, Instance instance, Event @event);`,
			want: `must be a compiler-owned callable declaration`,
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
					TargetLanguage: LanguageCSharp,
					Code:           tt.code,
				}},
			}
			AssignTargetABI(program, LanguageCSharp)

			_, err := NewCSharpBackend().Emit(context.Background(), program)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestCSharpBackendRejectsAdapterFullCodeWhenTriggerLambdaSignatureChanges(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "trigger_1",
			Kind:           BehaviorTrigger,
			TriggerKind:    TriggerWhen,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageCSharp,
			Code:           `(ctx, instance, @event) => System.Threading.Tasks.Task.CompletedTask`,
		}},
	}
	AssignTargetABI(program, LanguageCSharp)

	_, err := NewCSharpBackend().Emit(context.Background(), program)
	if err == nil || !strings.Contains(err.Error(), `declares lambda parameters "(ctx, instance, @event)", want compiler-owned parameters "(ctx, instance, @event, cancellationToken)"`) {
		t.Fatalf("error = %v, want compiler-owned C# when-trigger lambda signature rejection", err)
	}
}

func TestCSharpBackendUsesTypedAdapterFullCodeTriggerBehaviors(t *testing.T) {
	cases := []struct {
		name        string
		triggerKind TriggerKind
		code        string
		want        string
	}{
		{
			name:        "after",
			triggerKind: TriggerAfter,
			code:        `(ctx, instance, @event) => System.TimeSpan.Zero`,
			want:        `private static readonly DurationProvider<Instance> Trigger1 = (ctx, instance, @event) => System.TimeSpan.Zero;`,
		},
		{
			name:        "at",
			triggerKind: TriggerAt,
			code:        `(ctx, instance, @event) => System.DateTimeOffset.UnixEpoch`,
			want:        `private static readonly TimeProvider<Instance> Trigger1 = (ctx, instance, @event) => System.DateTimeOffset.UnixEpoch;`,
		},
		{
			name:        "when",
			triggerKind: TriggerWhen,
			code:        `(ctx, instance, @event, cancellationToken) => System.Threading.Tasks.Task.CompletedTask`,
			want:        `private static readonly ConditionChannel<Instance> Trigger1 = (ctx, instance, @event, cancellationToken) => System.Threading.Tasks.Task.CompletedTask;`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := &Program{
				Behaviors: []Behavior{{
					ID:             "trigger_1",
					Kind:           BehaviorTrigger,
					TriggerKind:    tc.triggerKind,
					SourceLanguage: LanguageGo,
					TargetLanguage: LanguageCSharp,
					Code:           tc.code,
				}},
			}
			AssignTargetABI(program, LanguageCSharp)

			output, err := NewCSharpBackend().Emit(context.Background(), program)
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			if !strings.Contains(text, tc.want) {
				t.Fatalf("C# output missing %q:\n%s", tc.want, text)
			}
		})
	}
}
