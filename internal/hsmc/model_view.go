package hsmc

import (
	"path"
	"strings"
)

func BuildModelViews(program *Program) []ModelView {
	if program == nil {
		return nil
	}
	views := make([]ModelView, 0, len(program.Models))
	for _, model := range program.Models {
		modelPath := cleanModelPath(model.Name)
		view := ModelView{
			ID:          model.ID,
			Name:        model.Name,
			Path:        modelPath,
			Events:      eventViews(program.Events),
			Attributes:  attributeViews(model.Attributes),
			Operations:  operationViews(model.Operations),
			Transitions: transitionViews(model.Transitions),
		}
		if model.Initializer != nil {
			view.InitialTarget = model.Initializer.Target
			view.InitialEffects = append([]BehaviorRef(nil), model.Initializer.Effects...)
		}
		view.States = stateViews(model.States, modelPath)
		views = append(views, view)
	}
	return views
}

func stateViews(states []State, ownerPath string) []StateView {
	views := make([]StateView, 0, len(states))
	for _, state := range states {
		statePath := cleanJoinedPath(ownerPath, state.Name)
		view := StateView{
			ID:          state.ID,
			Name:        state.Name,
			Path:        statePath,
			Kind:        state.Kind,
			Behaviors:   append([]BehaviorRef(nil), state.Behaviors...),
			Defers:      append([]string(nil), state.Defers...),
			Transitions: transitionViews(state.Transitions),
		}
		if state.Initializer != nil {
			view.InitialTarget = state.Initializer.Target
			view.InitialEffects = append([]BehaviorRef(nil), state.Initializer.Effects...)
		}
		view.States = stateViews(state.States, statePath)
		views = append(views, view)
	}
	return views
}

func transitionViews(transitions []Transition) []TransitionView {
	views := make([]TransitionView, 0, len(transitions))
	for _, transition := range transitions {
		view := TransitionView{
			ID:      transition.ID,
			OwnerID: transition.OwnerID,
			Source:  transition.Source,
			Target:  transition.Target,
			Trigger: cloneTrigger(transition.Trigger),
			Guard:   cloneBehaviorRef(transition.Guard),
			Effects: append([]BehaviorRef(nil), transition.Effects...),
		}
		views = append(views, view)
	}
	return views
}

func cloneTrigger(trigger *Trigger) *Trigger {
	if trigger == nil {
		return nil
	}
	cloned := *trigger
	cloned.Values = append([]TriggerValue(nil), trigger.Values...)
	cloned.Expr = cloneBehaviorRef(trigger.Expr)
	return &cloned
}

func cloneBehaviorRef(ref *BehaviorRef) *BehaviorRef {
	if ref == nil {
		return nil
	}
	cloned := *ref
	return &cloned
}

func eventViews(events []EventDecl) []EventView {
	views := make([]EventView, 0, len(events))
	for _, event := range events {
		views = append(views, EventView{
			ID:       event.ID,
			Symbol:   event.Symbol,
			Name:     event.Name,
			Schema:   event.Schema,
			Language: event.Language,
		})
	}
	return views
}

func attributeViews(attributes []Attribute) []AttributeView {
	views := make([]AttributeView, 0, len(attributes))
	for _, attribute := range attributes {
		views = append(views, AttributeView{
			ID:         attribute.ID,
			Name:       attribute.Name,
			Type:       attribute.Type,
			HasDefault: attribute.HasDefault,
			Default:    attribute.Default,
			Language:   attribute.Language,
		})
	}
	return views
}

func operationViews(operations []Operation) []OperationView {
	views := make([]OperationView, 0, len(operations))
	for _, operation := range operations {
		views = append(views, OperationView{
			ID:       operation.ID,
			Name:     operation.Name,
			Behavior: cloneBehaviorRef(operation.Behavior),
		})
	}
	return views
}

func cleanModelPath(name string) string {
	if name == "" {
		return "/"
	}
	return cleanJoinedPath("/", name)
}

func cleanJoinedPath(ownerPath string, name string) string {
	cleaned := path.Clean(path.Join(ownerPath, name))
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	return cleaned
}
