package hsmc

import (
	"context"
	"strings"
	"testing"
)

const cppDoorSource = `#include "hsm/hsm.hpp"
#include "format.hpp"

using namespace hsm;

struct OpenEvent {
  static constexpr auto name = "open";
};

struct Door {
  int count = 0;
};

static void entered(Signal &, Door &, const EventBase &) {
  format::record("closed");
}

static constexpr auto DoorModel = define(
  "Door",
  initial(target("/Door/closed")),
  state("closed",
    entry(entered),
    transition(
      on<OpenEvent>(),
      guard([](Door &door, const OpenEvent &) {
        return door.count >= 0;
      }),
      target("/Door/open"),
      effect([](Door &door, const OpenEvent &) {
        door.count++;
      })
    )
  ),
  state("open")
);
`

func TestCPPFrontendLowersModelAndBehaviors(t *testing.T) {
	program, err := NewCPPFrontend().Parse(context.Background(), SourceInput{Path: "door.cpp", Data: []byte(cppDoorSource)})
	if err != nil {
		t.Fatal(err)
	}
	if program.SourceLanguage != LanguageCPP {
		t.Fatalf("source language = %q, want C++", program.SourceLanguage)
	}
	if len(program.Imports) != 1 || program.Imports[0].Path != "format.hpp" {
		t.Fatalf("imports = %#v", program.Imports)
	}
	for _, global := range program.Globals {
		if strings.TrimSpace(global.Code) == "using namespace hsm;" {
			t.Fatalf("runtime namespace using should be compiler-owned, got global %#v", global)
		}
	}
	if len(program.Events) != 1 || program.Events[0].Symbol != "OpenEvent" || program.Events[0].Name != "open" {
		t.Fatalf("events = %#v", program.Events)
	}
	if len(program.Globals) != 1 || !strings.Contains(program.Globals[0].Code, "struct Door") {
		t.Fatalf("globals = %#v, want instance struct only", program.Globals)
	}
	for _, global := range program.Globals {
		if strings.Contains(global.Code, "static void entered") {
			t.Fatalf("referenced behavior function leaked as global: %#v", program.Globals)
		}
	}
	if len(program.Models) != 1 {
		t.Fatalf("models = %d, want 1", len(program.Models))
	}
	model := program.Models[0]
	if model.Name != "Door" || model.Initializer == nil || model.Initializer.Target != "/Door/closed" {
		t.Fatalf("model = %#v", model)
	}
	if len(model.States) != 2 {
		t.Fatalf("states = %d, want 2", len(model.States))
	}
	closed := model.States[0]
	if closed.Name != "closed" || len(closed.Behaviors) != 1 || len(closed.Transitions) != 1 {
		t.Fatalf("closed state = %#v", closed)
	}
	transition := closed.Transitions[0]
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerOn || len(transition.Trigger.Values) != 1 || transition.Trigger.Values[0].EventSymbol != "OpenEvent" {
		t.Fatalf("trigger = %#v", transition.Trigger)
	}
	if transition.Guard == nil || len(transition.Effects) != 1 {
		t.Fatalf("transition behaviors not lowered: %#v", transition)
	}
	if len(program.Behaviors) != 3 {
		t.Fatalf("behaviors = %d, want entry, guard and effect", len(program.Behaviors))
	}
	var entryBody, guardBody string
	for _, behavior := range program.Behaviors {
		if behavior.SourceLanguage != LanguageCPP {
			t.Fatalf("behavior source language = %q", behavior.SourceLanguage)
		}
		if behavior.Kind == BehaviorEntry {
			entryBody = behavior.Body
		}
		if behavior.Kind == BehaviorGuard {
			guardBody = behavior.Body
		}
	}
	if !strings.Contains(entryBody, `format::record("closed");`) {
		t.Fatalf("entry body = %q", entryBody)
	}
	if !strings.Contains(guardBody, "return door.count >= 0;") {
		t.Fatalf("guard body = %q", guardBody)
	}
}

func TestCPPFrontendLowersCanonicalPascalCaseDSL(t *testing.T) {
	source := `#include "hsm/hsm.hpp"

static constexpr auto DoorModel = hsm::Define(
  "Door",
  hsm::Initial(hsm::Target("/Door/closed")),
  hsm::State("closed",
    hsm::Transition(
      hsm::OnSet("open"),
      hsm::Target("/Door/open")
    )
  ),
  hsm::State("open"),
  hsm::ShallowHistory("recent", hsm::Target("/Door/closed"))
);
`
	program, err := NewCPPFrontend().Parse(context.Background(), SourceInput{Path: "door.cpp", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("PascalCase model declaration leaked into globals: %#v", program.Globals)
	}
	if len(program.Models) != 1 {
		t.Fatalf("models = %#v, want one model", program.Models)
	}
	model := program.Models[0]
	if model.Name != "Door" || model.Initializer == nil || model.Initializer.Target != "/Door/closed" {
		t.Fatalf("model = %#v", model)
	}
	if len(model.States) != 3 {
		t.Fatalf("states = %#v, want closed, open, and history", model.States)
	}
	if model.States[2].Kind != StateKindShallowHistory {
		t.Fatalf("history state = %#v, want shallow history", model.States[2])
	}
	if len(model.States[0].Transitions) != 1 || model.States[0].Transitions[0].Trigger == nil ||
		model.States[0].Transitions[0].Trigger.Kind != TriggerOnSet {
		t.Fatalf("PascalCase OnSet transition not lowered: %#v", model.States[0].Transitions)
	}
}

func TestCPPFrontendFiltersRuntimeNamespaceUsing(t *testing.T) {
	source := `#include "hsm/hsm.hpp"

using namespace hsm;

static constexpr auto DoorModel = define(
  "Door",
  initial(target("/Door/closed")),
  state("closed")
);
`
	program, err := NewCPPFrontend().Parse(context.Background(), SourceInput{Path: "door.cpp", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 0 {
		t.Fatalf("runtime include should be compiler-owned, got imports %#v", program.Imports)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("runtime namespace using should not be global context: %#v", program.Globals)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
}

func TestCPPFrontendRejectsUnknownHSMDslPartials(t *testing.T) {
	source := `#include "hsm/hsm.hpp"

using namespace hsm;

static constexpr auto DoorModel = hsm::define(
  "Door",
  hsm::unknown_model_partial(),
  hsm::initial(hsm::unknown_initial_partial(), hsm::target("closed")),
  hsm::state("closed",
    hsm::unknown_state_partial(),
    hsm::transition(
      unknown_transition_partial(),
      hsm::target("/Door/open")
    )
  )
);
`
	_, err := NewCPPFrontend().Parse(context.Background(), SourceInput{Path: "door.cpp", Data: []byte(source)})
	if err == nil {
		t.Fatal("expected unknown HSM partial diagnostics")
	}
	for _, want := range []string{
		`unknown or unsupported hsm::unknown_model_partial partial in model context`,
		`unknown or unsupported hsm::unknown_initial_partial partial in initial context`,
		`unknown or unsupported hsm::unknown_state_partial partial in state context`,
		`unknown or unsupported hsm::unknown_transition_partial partial in transition context`,
	} {
		if !diagnosticsContain(err, want) {
			t.Fatalf("diagnostics = %v, want %q", err, want)
		}
	}
}

func TestCompilerConvertsCPPSourceToCPPTarget(t *testing.T) {
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.cpp", Data: []byte(cppDoorSource)}, CompileOptions{
		From:    LanguageCPP,
		To:      LanguageCPP,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if program.SourceLanguage != LanguageCPP {
		t.Fatalf("program source = %q, want C++", program.SourceLanguage)
	}
	text := string(output)
	for _, needle := range []string{
		`#include "format.hpp"`,
		"static constexpr auto door_model = hsm::define(",
		"hsm::entry(entry_1)",
		"hsm::on(open_event)",
		"hsm::guard(guard_1)",
		`format::record("closed");`,
		"return door.count >= 0;",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C++ output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original cpp behavior") {
		t.Fatalf("C++-to-C++ should preserve behavior bodies, got:\n%s", text)
	}
}

func TestCPPFrontendReingestsGeneratedBehaviorLambdas(t *testing.T) {
	source := `#include "hsm/hsm.hpp"
#include <chrono>

namespace generated_hsm {

static constexpr auto entry_1 = [](auto& signal, auto& instance, const auto& event) -> void {
    instance.entered = true;
};

static constexpr auto guard_1 = [](auto& signal, auto& instance, const auto& event) -> bool {
    return true;
};

static constexpr auto door_model = hsm::define(
    "Door",
    hsm::initial(hsm::target("/Door/closed")),
    hsm::state("closed",
        hsm::entry(entry_1),
        hsm::transition(hsm::on("open"), hsm::guard(guard_1), hsm::target("/Door/open"))
    ),
    hsm::state("open")
);

}  // namespace generated_hsm
`
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "generated.cpp", Data: []byte(source)}, CompileOptions{
		From:    LanguageCPP,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("generated behavior lambdas should not remain globals: %#v", program.Globals)
	}
	if len(program.Behaviors) != 2 || !strings.Contains(program.Behaviors[0].Body, `instance.entered = true;`) || !strings.Contains(program.Behaviors[1].Body, "return true;") {
		t.Fatalf("behaviors = %#v", program.Behaviors)
	}
	text := string(output)
	for _, needle := range []string{
		`// instance.entered = true;`,
		`// return true;`,
		`hsm.Entry(Entry1)`,
		`hsm.Guard(Guard1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original cpp global") {
		t.Fatalf("Go output preserved generated behavior lambdas as globals:\n%s", text)
	}
}

func TestCPPFrontendReingestsGeneratedOperationLambda(t *testing.T) {
	source := `#include "hsm/hsm.hpp"

namespace generated_hsm {

static constexpr auto operation_1 = [](auto& signal, auto& instance, const auto& event) -> void {
    instance.approved = true;
};

static constexpr auto door_model = hsm::define(
    "Door",
    hsm::operation("approve", operation_1),
    hsm::initial(hsm::target("/Door/closed")),
    hsm::state("closed",
        hsm::transition(hsm::on_call("approve"), hsm::target("/Door/open"))
    ),
    hsm::state("open")
);

}  // namespace generated_hsm
`
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "generated.cpp", Data: []byte(source)}, CompileOptions{
		From:    LanguageCPP,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("generated operation lambda should not remain global: %#v", program.Globals)
	}
	if len(program.Behaviors) != 1 || program.Behaviors[0].Kind != BehaviorOperation {
		t.Fatalf("behaviors = %#v, want one operation behavior", program.Behaviors)
	}
	if !strings.Contains(program.Behaviors[0].Body, `instance.approved = true;`) {
		t.Fatalf("operation body = %q", program.Behaviors[0].Body)
	}
	text := string(output)
	for _, needle := range []string{
		`// instance.approved = true;`,
		`hsm.Operation("approve", Operation1)`,
		`hsm.OnCall("approve")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original cpp global") {
		t.Fatalf("Go output preserved generated operation lambda as global:\n%s", text)
	}
}

func TestCPPFrontendLowersAttributesOperationsAndExtendedTriggers(t *testing.T) {
	source := `#include "hsm/hsm.hpp"

using namespace hsm;

struct Door {
  int count = 0;
};

static constexpr auto approve = [](Signal &, Door &, const EventBase &) {
  return true;
};

static constexpr auto DoorModel = define(
  "Door",
  attribute<int>("count", 1),
  operation("approve", approve),
  initial(target("closed")),
  state("closed",
    transition(on_call("approve"), target("../open")),
    transition(on_set("count"), target("../open")),
    transition(at([](Signal &, Door &, const EventBase &) {
      return std::chrono::system_clock::time_point{};
    }), target("../open")),
    transition(when([](Signal &, Door &door, const EventBase &) {
      return door.count > 0;
    }), target("../open"))
  ),
  state("open")
);
`
	program, err := NewCPPFrontend().Parse(context.Background(), SourceInput{Path: "rich.cpp", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	model := program.Models[0]
	if len(model.Attributes) != 1 || model.Attributes[0].Name != "count" || model.Attributes[0].Type != "int" || model.Attributes[0].Default != "1" {
		t.Fatalf("attributes = %#v", model.Attributes)
	}
	if len(model.Operations) != 1 || model.Operations[0].Name != "approve" || model.Operations[0].Behavior == nil {
		t.Fatalf("operations = %#v", model.Operations)
	}
	transitions := model.States[0].Transitions
	if len(transitions) != 4 {
		t.Fatalf("transitions = %#v", transitions)
	}
	if transitions[0].Trigger == nil || transitions[0].Trigger.Kind != TriggerOnCall || transitions[0].Trigger.Value != "approve" {
		t.Fatalf("on-call trigger = %#v", transitions[0].Trigger)
	}
	if transitions[1].Trigger == nil || transitions[1].Trigger.Kind != TriggerOnSet || transitions[1].Trigger.Value != "count" {
		t.Fatalf("on-set trigger = %#v", transitions[1].Trigger)
	}
	if transitions[2].Trigger == nil || transitions[2].Trigger.Kind != TriggerAt || transitions[2].Trigger.Expr == nil {
		t.Fatalf("at trigger = %#v", transitions[2].Trigger)
	}
	if transitions[3].Trigger == nil || transitions[3].Trigger.Kind != TriggerWhen || transitions[3].Trigger.Expr == nil {
		t.Fatalf("when trigger = %#v", transitions[3].Trigger)
	}
	if len(program.Behaviors) != 3 {
		t.Fatalf("behaviors = %d, want operation plus At and When triggers", len(program.Behaviors))
	}
	if err := Validate(program); err != nil {
		t.Fatalf("Validate() = %v", err)
	}

	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "rich.cpp", Data: []byte(source)}, CompileOptions{
		From:    LanguageCPP,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`// Type: int`,
		`hsm.Attribute("count", 1)`,
		`hsm.Operation("approve", Operation1)`,
		`hsm.OnCall("approve")`,
		`hsm.OnSet("count")`,
		`hsm.At(Trigger1)`,
		`hsm.When(Trigger2)`,
		`// Original cpp behavior trigger_2 preserved for manual porting:`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
}
