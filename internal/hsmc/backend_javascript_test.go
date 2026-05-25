package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestJavaScriptBackendEmitsAttributesWithPortableFallbacks(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			Attributes: []Attribute{
				{ID: "attribute_1", Name: "count", Type: "Number", HasDefault: true, Default: "1", Language: LanguageJS},
				{ID: "attribute_2", Name: "label", HasDefault: true, Default: `"closed"`, Language: LanguageGo},
				{ID: "attribute_3", Name: "ready", HasDefault: true, Default: "True", Language: LanguagePython},
				{ID: "attribute_4", Name: "payload", HasDefault: true, Default: "makePayload()", Language: LanguagePython},
				{ID: "attribute_5", Name: "type_only", Type: "String", Language: LanguageJS},
			},
		}},
	}

	output, err := NewJavaScriptBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`// Type: Number`,
		`hsm.Attribute("count", 1)`,
		`hsm.Attribute("label", "closed")`,
		`hsm.Attribute("ready", true)`,
		`// Default: makePayload()`,
		`hsm.Attribute("payload")`,
		`// Type: String`,
		`hsm.Attribute("type_only")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("JavaScript attribute output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{
		`hsm.Attribute("ready", True)`,
		`hsm.Attribute("payload", makePayload())`,
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("JavaScript attribute output emitted invalid foreign default %q:\n%s", forbidden, text)
		}
	}
}

func TestJavaScriptBackendUsesAdapterFullCodeBehavior(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageJS,
			Code: `function entry1(ctx, instance, event) {
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
	AssignTargetABI(program, LanguageJS)

	output, err := NewJavaScriptBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`function entry1(ctx, instance, event)`,
		`instance.ready = true;`,
		`hsm.Entry(entry1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("JavaScript output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "const entry1 = function entry1") {
		t.Fatalf("JavaScript output wrapped a full function declaration as an expression:\n%s", text)
	}
}

func TestJavaScriptBackendRejectsAdapterFullCodeBehaviorNameChanges(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetLanguage: LanguageJS,
			Code: `const renamedEntry = (ctx, instance, event) => {
	instance.ready = true;
};`,
		}},
	}
	AssignTargetABI(program, LanguageJS)

	_, err := NewJavaScriptBackend().Emit(context.Background(), program)
	if err == nil || !strings.Contains(err.Error(), `declares "renamedEntry", want compiler-owned name "entry1"`) {
		t.Fatalf("error = %v, want compiler-owned JavaScript behavior name rejection", err)
	}
}

func TestJavaScriptBackendRejectsAdapterFullCodeBehaviorSignatureChanges(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "function declaration",
			code: `function entry1(ctx) {
}`,
			want: `declares parameters "(ctx)", want compiler-owned parameters "(ctx, instance, event)"`,
		},
		{
			name: "arrow expression",
			code: `(ctx) => {
}`,
			want: `declares parameters "(ctx)", want compiler-owned parameters "(ctx, instance, event)"`,
		},
		{
			name: "top level variable arrow expression",
			code: `const entry1 = (ctx) => {
};`,
			want: `declares parameters "(ctx)", want compiler-owned parameters "(ctx, instance, event)"`,
		},
		{
			name: "async arrow expression",
			code: `async (ctx) => {
}`,
			want: `declares parameters "(ctx)", want compiler-owned parameters "(ctx, instance, event)"`,
		},
		{
			name: "single parameter arrow expression",
			code: `ctx => {
}`,
			want: `declares parameters "(ctx)", want compiler-owned parameters "(ctx, instance, event)"`,
		},
		{
			name: "non-callable class declaration",
			code: `class entry1 {
}`,
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
					TargetLanguage: LanguageJS,
					Code:           tt.code,
				}},
			}
			AssignTargetABI(program, LanguageJS)

			_, err := NewJavaScriptBackend().Emit(context.Background(), program)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}
