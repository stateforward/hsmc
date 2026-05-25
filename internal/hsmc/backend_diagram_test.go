package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestDiagramBackendsEmitStateTransitionsAndNotes(t *testing.T) {
	program := diagramTestProgram()
	cases := []struct {
		name    string
		target  Language
		backend Backend
		want    []string
	}{
		{
			name:    "mermaid",
			target:  LanguageMermaid,
			backend: NewMermaidBackend(),
			want: []string{
				"stateDiagram-v2",
				`state "Door" as Door {`,
				"[*] --> Door_closed",
				"Door_closed --> Door_open : on approve / guard guard_1 / effect effect_1",
				"note right of Door_closed",
				"Original go behavior entry_1 preserved for manual porting:",
				"Behavior kind: entry",
				"instance.Log = append(instance.Log, \"entered\")",
				"Source imports:",
				`import "fmt"`,
				"Original go global global_1 preserved for manual porting:",
				"var priorityQueue = NewQueue()",
			},
		},
		{
			name:    "plantuml",
			target:  LanguagePlantUML,
			backend: NewPlantUMLBackend(),
			want: []string{
				"@startuml",
				`state "Door" as Door {`,
				"[*] --> Door_closed",
				"Door_closed --> Door_open : on approve / guard guard_1 / effect effect_1",
				"note right of Door_closed",
				"Original go behavior entry_1 preserved for manual porting:",
				"Behavior kind: entry",
				"instance.Log = append(instance.Log, \"entered\")",
				"Source imports:",
				`import "fmt"`,
				"Original go global global_1 preserved for manual porting:",
				"var priorityQueue = NewQueue()",
				"@enduml",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			AssignTargetABI(program, tc.target)
			output, err := tc.backend.Emit(context.Background(), program)
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			for _, needle := range tc.want {
				if !strings.Contains(text, needle) {
					t.Fatalf("%s diagram missing %q:\n%s", tc.name, needle, text)
				}
			}
		})
	}
}

func TestCompilerCompilesGoSourceToDiagramTargets(t *testing.T) {
	cases := []struct {
		name   string
		target Language
		want   []string
	}{
		{
			name:   "mermaid",
			target: LanguageMermaid,
			want: []string{
				"stateDiagram-v2",
				"Original go behavior entry_1 preserved for manual porting:",
				"instance.Log = append(instance.Log, \"entered\")",
			},
		},
		{
			name:   "plantuml",
			target: LanguagePlantUML,
			want: []string{
				"@startuml",
				"Original go behavior entry_1 preserved for manual porting:",
				"instance.Log = append(instance.Log, \"entered\")",
				"@enduml",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(diagramGoSource)}, CompileOptions{
				From:    LanguageGo,
				To:      tc.target,
				Adapter: "none",
			})
			if err != nil {
				t.Fatal(err)
			}
			if program == nil || len(program.Models) != 1 {
				t.Fatalf("compiled diagram program = %#v", program)
			}
			text := string(output)
			for _, needle := range tc.want {
				if !strings.Contains(text, needle) {
					t.Fatalf("%s output missing %q:\n%s", tc.target, needle, text)
				}
			}
		})
	}
}

func diagramTestProgram() *Program {
	return &Program{
		SourceLanguage: LanguageGo,
		Imports:        []Import{{Path: "fmt", Language: LanguageGo}},
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageGo,
			Code:     "var priorityQueue = NewQueue()",
			Imports:  []Import{{Path: "fmt", Language: LanguageGo}},
		}},
		Models: []Model{{
			ID:          "model_1",
			Name:        "Door",
			Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed"},
			States: []State{
				{
					ID:   "state_1",
					Name: "closed",
					Kind: StateKindState,
					Behaviors: []BehaviorRef{{
						ID:   "entry_1",
						Kind: BehaviorEntry,
					}},
					Transitions: []Transition{{
						OwnerID: "state_1",
						ID:      "transition_1",
						Target:  "../open",
						Trigger: &Trigger{Kind: TriggerOn, Value: "approve", IsString: true},
						Guard:   &BehaviorRef{ID: "guard_1", Kind: BehaviorGuard},
						Effects: []BehaviorRef{{ID: "effect_1", Kind: BehaviorEffect}},
					}},
				},
				{ID: "state_2", Name: "open", Kind: StateKindState},
			},
		}},
		Behaviors: []Behavior{
			{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo, Body: `instance.Log = append(instance.Log, "entered")`, Imports: []Import{{Path: "fmt", Language: LanguageGo}}},
			{ID: "guard_1", OwnerID: "transition_1", Kind: BehaviorGuard, SourceLanguage: LanguageGo, Body: "return instance.Allowed"},
			{ID: "effect_1", OwnerID: "transition_1", Kind: BehaviorEffect, SourceLanguage: LanguageGo, Body: `instance.Log = append(instance.Log, "approved")`},
		},
	}
}

const diagramGoSource = `package sample

import hsm "github.com/stateforward/hsm.go"

type Instance struct {
	Log []string
}

var DoorModel = hsm.Define("Door",
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed",
		hsm.Entry(func(ctx hsm.Context, instance *Instance, event hsm.Event) {
			instance.Log = append(instance.Log, "entered")
		}),
		hsm.Transition(hsm.On("approve"), hsm.Target("../open")),
	),
	hsm.State("open"),
)
`
