package hsmc

import "testing"

func TestBuildModelViewsProvidesReadOnlyAdapterContext(t *testing.T) {
	program := &Program{
		Events: []EventDecl{{ID: "event_1", Symbol: "goEvent", Name: "go", Schema: "{ kind: string }", Language: LanguageTS}},
		Models: []Model{{
			ID:          "model_1",
			Name:        "Door",
			Initializer: &Initial{OwnerID: "model_1", ID: "initial_1", Target: "closed", Effects: []BehaviorRef{{ID: "init_effect_1", Kind: BehaviorEffect}}},
			Attributes:  []Attribute{{ID: "attribute_1", Name: "count", Type: "int", HasDefault: true, Default: "1", Language: LanguageGo}},
			Operations:  []Operation{{ID: "operation_1", Name: "approve", Behavior: &BehaviorRef{ID: "operation_behavior_1", Kind: BehaviorOperation}}},
			States: []State{{
				ID:          "state_1",
				Name:        "closed",
				Kind:        StateKindState,
				Initializer: &Initial{OwnerID: "state_1", ID: "initial_2", Target: "child", Effects: []BehaviorRef{{ID: "nested_init_effect_1", Kind: BehaviorEffect}}},
				Behaviors:   []BehaviorRef{{ID: "entry_1", Kind: BehaviorEntry}},
				Transitions: []Transition{{
					ID:      "transition_1",
					OwnerID: "state_1",
					Target:  "../open",
					Trigger: &Trigger{Kind: TriggerOnCall, Value: "approve", IsString: true},
					Effects: []BehaviorRef{{ID: "effect_1", Kind: BehaviorEffect}},
				}},
			}},
		}}}
	views := BuildModelViews(program)
	if len(views) != 1 {
		t.Fatalf("views = %#v", views)
	}
	model := views[0]
	if model.Path != "/Door" || model.InitialTarget != "closed" {
		t.Fatalf("model view = %#v", model)
	}
	if len(model.InitialEffects) != 1 || model.InitialEffects[0].ID != "init_effect_1" {
		t.Fatalf("model initial effects = %#v", model.InitialEffects)
	}
	if len(model.Attributes) != 1 || model.Attributes[0].Name != "count" || model.Attributes[0].Type != "int" || model.Attributes[0].Default != "1" {
		t.Fatalf("model symbols = %#v / %#v", model.Attributes, model.Operations)
	}
	if len(model.Events) != 1 || model.Events[0].Symbol != "goEvent" || model.Events[0].Name != "go" || model.Events[0].Schema == "" {
		t.Fatalf("model events = %#v", model.Events)
	}
	if len(model.Operations) != 1 || model.Operations[0].Name != "approve" || model.Operations[0].Behavior == nil {
		t.Fatalf("operation symbols = %#v", model.Operations)
	}
	state := model.States[0]
	if state.Path != "/Door/closed" || len(state.Behaviors) != 1 {
		t.Fatalf("state view = %#v", state)
	}
	if state.InitialTarget != "child" || len(state.InitialEffects) != 1 || state.InitialEffects[0].ID != "nested_init_effect_1" {
		t.Fatalf("state initial view = %#v", state)
	}
	transition := state.Transitions[0]
	if transition.Trigger == nil || transition.Trigger.Value != "approve" || len(transition.Effects) != 1 {
		t.Fatalf("transition view = %#v", transition)
	}
}
