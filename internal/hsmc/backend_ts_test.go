package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestTypeScriptBackendEmitsAttributesWithPortableFallbacks(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			Attributes: []Attribute{
				{ID: "attribute_1", Name: "count", Type: "Number", HasDefault: true, Default: "1", Language: LanguageTS},
				{ID: "attribute_2", Name: "label", HasDefault: true, Default: `"closed"`, Language: LanguageGo},
				{ID: "attribute_3", Name: "ready", HasDefault: true, Default: "True", Language: LanguagePython},
				{ID: "attribute_4", Name: "payload", Type: "Object", HasDefault: true, Default: "makePayload()", Language: LanguagePython},
				{ID: "attribute_5", Name: "type_only", Type: "String", Language: LanguageTS},
			},
		}},
	}

	output, err := NewTypeScriptBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`hsm.Attribute("count", Number, 1)`,
		`hsm.Attribute("label", "closed")`,
		`hsm.Attribute("ready", true)`,
		`// Type: Object`,
		`// Default: makePayload()`,
		`hsm.Attribute("payload")`,
		`hsm.Attribute("type_only", String)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript attribute output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{
		`hsm.Attribute("ready", True)`,
		`hsm.Attribute("payload", Object, makePayload())`,
		`hsm.Attribute("payload", makePayload())`,
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("TypeScript attribute output emitted invalid foreign default %q:\n%s", forbidden, text)
		}
	}
}

func TestTypeScriptBackendUsesAdapterFullCodeBehavior(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageTS,
			Code: `function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
	instance.ready = true;
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
	AssignTargetABI(program, LanguageTS)

	output, err := NewTypeScriptBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void`,
		`instance.ready = true;`,
		`hsm.Entry(entry1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "const entry1 = function entry1") {
		t.Fatalf("TypeScript output wrapped a full function declaration as an expression:\n%s", text)
	}
}

func TestTypeScriptBackendRejectsAdapterFullCodeBehaviorNameChanges(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageTS,
			Code: `const renamedEntry: hsm.Operation<InstanceType<typeof hsm.Instance>> = (ctx, instance, event) => {
	instance.ready = true;
};`,
		}},
	}
	AssignTargetABI(program, LanguageTS)

	_, err := NewTypeScriptBackend().Emit(context.Background(), program)
	if err == nil || !strings.Contains(err.Error(), `declares "renamedEntry", want compiler-owned name "entry1"`) {
		t.Fatalf("error = %v, want compiler-owned TypeScript behavior name rejection", err)
	}
}

func TestTypeScriptBackendRejectsAdapterFullCodeBehaviorSignatureChanges(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "parameters",
			code: `function entry1(ctx: hsm.Context): void {
}`,
			want: `declares parameters "(ctx: hsm.Context)", want compiler-owned parameters "(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event)"`,
		},
		{
			name: "return type",
			code: `function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): boolean {
	return true;
}`,
			want: `declares return type "boolean", want compiler-owned return type "void"`,
		},
		{
			name: "arrow expression parameters",
			code: `(ctx: hsm.Context): void => {
}`,
			want: `declares parameters "(ctx: hsm.Context)", want compiler-owned parameters "(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event)"`,
		},
		{
			name: "top level variable arrow expression parameters",
			code: `const entry1 = (ctx: hsm.Context): void => {
};`,
			want: `declares parameters "(ctx: hsm.Context)", want compiler-owned parameters "(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event)"`,
		},
		{
			name: "arrow expression return type",
			code: `(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): boolean => {
	return true;
}`,
			want: `declares return type "boolean", want compiler-owned return type "void"`,
		},
		{
			name: "non-callable class declaration",
			code: `class entry1 {
}`,
			want: `must be a compiler-owned callable declaration`,
		},
		{
			name: "non-callable type declaration",
			code: `type entry1 = {
	ready: boolean;
};`,
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
					TargetLanguage: LanguageTS,
					Code:           tt.code,
				}},
			}
			AssignTargetABI(program, LanguageTS)

			_, err := NewTypeScriptBackend().Emit(context.Background(), program)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}
