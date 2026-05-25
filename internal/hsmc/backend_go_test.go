package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestGoBackendEmitsAttributesWithPortableFallbacks(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			Attributes: []Attribute{
				{ID: "attribute_1", Name: "count", Type: "int", HasDefault: true, Default: "1", Language: LanguageGo},
				{ID: "attribute_2", Name: "label", HasDefault: true, Default: `"closed"`, Language: LanguageJava},
				{ID: "attribute_3", Name: "flag", Type: "bool", HasDefault: true, Default: "True", Language: LanguagePython},
				{ID: "attribute_4", Name: "type_only", Type: "string", Language: LanguageGo},
				{ID: "attribute_5", Name: "payload", HasDefault: true, Default: "makePayload()", Language: LanguagePython},
				{ID: "attribute_6", Name: "maybe", HasDefault: true, Default: "None", Language: LanguagePython},
			},
		}},
	}

	output, err := NewGoBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`// Type: int`,
		`hsm.Attribute("count", 1)`,
		`hsm.Attribute("label", "closed")`,
		`// Type: bool`,
		`hsm.Attribute("flag", true)`,
		`// Type: string`,
		`hsm.Attribute("type_only")`,
		`// Default: makePayload()`,
		`hsm.Attribute("payload")`,
		`hsm.Attribute("maybe", nil)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go attribute output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{
		`hsm.Attribute("flag", True)`,
		`hsm.Attribute("payload", makePayload())`,
		`hsm.Attribute("maybe", None)`,
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Go attribute output emitted invalid foreign default %q:\n%s", forbidden, text)
		}
	}
}

func TestGoBackendUsesAdapterFullFunctionDeclaration(t *testing.T) {
	program := &Program{
		PackageName: "sample",
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageTS,
			TargetLanguage: LanguageGo,
			Code: `func Entry1(ctx context.Context, instance hsm.Instance, event hsm.Event) {
	instance.Set("ready", true)
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
	AssignTargetABI(program, LanguageGo)

	output, err := NewGoBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`func Entry1(ctx context.Context, instance hsm.Instance, event hsm.Event)`,
		`instance.Set("ready", true)`,
		`hsm.Entry(Entry1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "var Entry1 = func Entry1") {
		t.Fatalf("Go output wrapped a full function declaration as a function literal:\n%s", text)
	}
}

func TestGoBackendRejectsAdapterFullCodeBehaviorNameChanges(t *testing.T) {
	program := &Program{
		PackageName: "sample",
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageTS,
			TargetLanguage: LanguageGo,
			Code: `func RenamedEntry(ctx context.Context, instance hsm.Instance, event hsm.Event) {
	instance.Set("ready", true)
}`,
		}},
	}
	AssignTargetABI(program, LanguageGo)

	_, err := NewGoBackend().Emit(context.Background(), program)
	if err == nil || !strings.Contains(err.Error(), `declares "RenamedEntry", want compiler-owned name "Entry1"`) {
		t.Fatalf("error = %v, want compiler-owned Go behavior name rejection", err)
	}
}

func TestGoBackendRejectsAdapterFullCodeBehaviorSignatureChanges(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "function declaration",
			code: `func Entry1() {
}`,
			want: `declares signature "()", want compiler-owned signature "(ctx context.Context, instance hsm.Instance, event hsm.Event)"`,
		},
		{
			name: "function literal",
			code: `func(ctx context.Context) {
}`,
			want: `declares function literal signature "(ctx context.Context)", want compiler-owned signature "(ctx context.Context, instance hsm.Instance, event hsm.Event)"`,
		},
		{
			name: "top level variable function literal",
			code: `var Entry1 = func(ctx context.Context) {
}`,
			want: `declares function literal signature "(ctx context.Context)", want compiler-owned signature "(ctx context.Context, instance hsm.Instance, event hsm.Event)"`,
		},
		{
			name: "non-callable variable declaration",
			code: `var Entry1 = 1`,
			want: `must be a compiler-owned callable declaration`,
		},
		{
			name: "non-callable type declaration",
			code: `type Entry1 struct{}`,
			want: `must be a compiler-owned callable declaration`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := &Program{
				PackageName: "sample",
				Behaviors: []Behavior{{
					ID:             "entry_1",
					OwnerID:        "state_1",
					Kind:           BehaviorEntry,
					SourceLanguage: LanguageTS,
					TargetLanguage: LanguageGo,
					Code:           tt.code,
				}},
			}
			AssignTargetABI(program, LanguageGo)

			_, err := NewGoBackend().Emit(context.Background(), program)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}
