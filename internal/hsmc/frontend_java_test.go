package hsmc

import (
	"context"
	"strings"
	"testing"
)

const javaDoorSource = `import com.stateforward.hsm.*;
import com.example.Format;

public final class DoorHsm {
  static final Event Open = new Event("open");

  static String record(String value) {
    return Format.record(value);
  }

  static final Model DoorModel = Hsm.Define(
    "Door",
    Hsm.Initial(Hsm.Target("closed")),
    Hsm.State(
      "closed",
      Hsm.Entry((ctx, instance, event) -> {
        instance.set("log", record("closed"));
      }),
      Hsm.Transition(
        Hsm.On(Open),
        Hsm.Guard((ctx, instance, event) -> {
          return true;
        }),
        Hsm.Target("../open"),
        Hsm.Effect((ctx, instance, event) -> {
          instance.set("last", event);
        })
      )
    ),
    Hsm.State("open")
  );
}`

func TestJavaFrontendLowersModel(t *testing.T) {
	program, err := NewJavaFrontend().Parse(context.Background(), SourceInput{Path: "DoorHsm.java", Data: []byte(javaDoorSource)})
	if err != nil {
		t.Fatal(err)
	}
	if program.SourceLanguage != LanguageJava {
		t.Fatalf("source language = %q, want java", program.SourceLanguage)
	}
	if len(program.Imports) != 1 || program.Imports[0].Path != "com.example.Format" {
		t.Fatalf("imports = %#v, want helper import only", program.Imports)
	}
	if len(program.Events) != 1 || program.Events[0].Symbol != "Open" || program.Events[0].Name != "open" {
		t.Fatalf("events = %#v", program.Events)
	}
	if len(program.Globals) != 1 || !strings.Contains(program.Globals[0].Code, "static String record") {
		t.Fatalf("globals = %#v", program.Globals)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	closed := program.Models[0].States[0]
	if closed.Name != "closed" || len(closed.Behaviors) != 1 || len(closed.Transitions) != 1 {
		t.Fatalf("closed state = %#v", closed)
	}
	transition := closed.Transitions[0]
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerOn || len(transition.Trigger.Values) != 1 || transition.Trigger.Values[0].EventSymbol != "Open" {
		t.Fatalf("transition trigger = %#v", transition.Trigger)
	}
	if transition.Guard == nil || len(transition.Effects) != 1 {
		t.Fatalf("transition behaviors = guard %#v effects %#v", transition.Guard, transition.Effects)
	}
	if len(program.Behaviors) != 3 {
		t.Fatalf("behaviors = %#v", program.Behaviors)
	}
	for _, behavior := range program.Behaviors {
		if behavior.SourceLanguage != LanguageJava || behavior.OwnerID == "" || behavior.Kind == "" || behavior.Body == "" {
			t.Fatalf("behavior = %#v", behavior)
		}
	}
}

const javaWildcardImportSource = `import com.stateforward.hsm.*;
import com.example.helpers.*;

public final class DoorHsm {
  static final Model DoorModel = Hsm.Define(
    "Door",
    Hsm.Initial(Hsm.Target("closed")),
    Hsm.State("closed", Hsm.Entry((ctx, instance, event) -> {
      Format.record("closed");
    }))
  );
}`

func TestJavaFrontendPreservesWildcardImport(t *testing.T) {
	program, err := NewJavaFrontend().Parse(context.Background(), SourceInput{Path: "DoorHsm.java", Data: []byte(javaWildcardImportSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 1 {
		t.Fatalf("imports = %#v", program.Imports)
	}
	imp := program.Imports[0]
	if imp.Path != "com.example.helpers" || imp.Language != LanguageJava {
		t.Fatalf("import metadata = %#v", imp)
	}
	if len(imp.Specifiers) != 1 || imp.Specifiers[0].Name != "*" {
		t.Fatalf("wildcard specifiers = %#v", imp.Specifiers)
	}
	if len(imp.LocalNames) != 1 || imp.LocalNames[0] != "*" {
		t.Fatalf("local names = %#v", imp.LocalNames)
	}
}

const javaStaticHelperImportSource = `import com.stateforward.hsm.*;
import static com.example.Format.record;

public final class DoorHsm {
  static final Model DoorModel = Hsm.Define(
    "Door",
    Hsm.Initial(Hsm.Target("closed")),
    Hsm.State("closed", Hsm.Entry((ctx, instance, event) -> {
      record("closed");
    }))
  );
}`

func TestJavaFrontendPreservesStaticHelperImport(t *testing.T) {
	program, err := NewJavaFrontend().Parse(context.Background(), SourceInput{Path: "DoorHsm.java", Data: []byte(javaStaticHelperImportSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 1 {
		t.Fatalf("imports = %#v", program.Imports)
	}
	imp := program.Imports[0]
	if imp.Path != "com.example.Format.record" || !imp.Static || strings.Join(imp.LocalNames, ",") != "record" {
		t.Fatalf("static import metadata = %#v", imp)
	}
}

func TestCompilerPreservesJavaStaticHelperImportInJavaTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "DoorHsm.java", Data: []byte(javaStaticHelperImportSource)}, CompileOptions{
		From:    LanguageJava,
		To:      LanguageJava,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, "import static com.example.Format.record;") {
		t.Fatalf("Java output missing static helper import:\n%s", text)
	}
	if strings.Contains(text, "import com.example.Format.record;") {
		t.Fatalf("Java output rendered static helper import as ordinary import:\n%s", text)
	}
}

const javaStaticRuntimeImportSource = `import static com.stateforward.hsm.Hsm.*;
import com.stateforward.hsm.*;

public final class DoorHsm {
  static final Event Open = new Event("open");

  static final Model DoorModel = Define(
    "Door",
    Initial(Target("closed")),
    State(
      "closed",
      Entry((ctx, instance, event) -> {
        instance.set("log", "closed");
      }),
      Transition(
        On(Open),
        Target("../open")
      )
    ),
    State("open")
  );
}`

func TestJavaFrontendLowersStaticImportedRuntimeCalls(t *testing.T) {
	program, err := NewJavaFrontend().Parse(context.Background(), SourceInput{Path: "DoorHsm.java", Data: []byte(javaStaticRuntimeImportSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 0 {
		t.Fatalf("runtime imports should be compiler-owned, got %#v", program.Imports)
	}
	if len(program.Events) != 1 || program.Events[0].Symbol != "Open" || program.Events[0].Name != "open" {
		t.Fatalf("events = %#v", program.Events)
	}
	if len(program.Models) != 1 {
		t.Fatalf("models = %#v, want one static-imported runtime model", program.Models)
	}
	model := program.Models[0]
	if model.Name != "Door" || model.Initializer == nil || model.Initializer.Target != "closed" {
		t.Fatalf("model = %#v", model)
	}
	if len(model.States) != 2 || len(model.States[0].Behaviors) != 1 || len(model.States[0].Transitions) != 1 {
		t.Fatalf("states = %#v", model.States)
	}
	transition := model.States[0].Transitions[0]
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerOn || len(transition.Trigger.Values) != 1 || transition.Trigger.Values[0].EventSymbol != "Open" {
		t.Fatalf("trigger = %#v", transition.Trigger)
	}
	if len(program.Behaviors) != 1 || !strings.Contains(program.Behaviors[0].Body, `instance.set("log", "closed");`) {
		t.Fatalf("behaviors = %#v", program.Behaviors)
	}
}

func TestJavaFrontendRejectsUnknownHSMDslPartials(t *testing.T) {
	source := `import static com.stateforward.hsm.Hsm.*;
import com.stateforward.hsm.*;

public final class DoorHsm {
  static final Model DoorModel = Hsm.Define(
    "Door",
    Hsm.UnknownModelPartial(),
    Hsm.Initial(Hsm.UnknownInitialPartial(), Hsm.Target("closed")),
    Hsm.State(
      "closed",
      Hsm.UnknownStatePartial(),
      Hsm.Transition(
        UnknownTransitionPartial(),
        Hsm.Target("../open")
      )
    )
  );
}`
	_, err := NewJavaFrontend().Parse(context.Background(), SourceInput{Path: "DoorHsm.java", Data: []byte(source)})
	if err == nil {
		t.Fatal("expected unknown HSM partial diagnostics")
	}
	for _, want := range []string{
		`unknown or unsupported Hsm.UnknownModelPartial partial in model context`,
		`unknown or unsupported Hsm.UnknownInitialPartial partial in initial context`,
		`unknown or unsupported Hsm.UnknownStatePartial partial in state context`,
		`unknown or unsupported Hsm.UnknownTransitionPartial partial in transition context`,
	} {
		if !diagnosticsContain(err, want) {
			t.Fatalf("diagnostics = %v, want %q", err, want)
		}
	}
}

func TestJavaSourceCompilesToGoAndJava(t *testing.T) {
	compiler := NewCompiler()
	for _, target := range []Language{LanguageGo, LanguageJava} {
		t.Run(string(target), func(t *testing.T) {
			output, program, err := compiler.Compile(context.Background(), SourceInput{Path: "DoorHsm.java", Data: []byte(javaDoorSource)}, CompileOptions{
				From:    LanguageJava,
				To:      target,
				Adapter: "none",
			})
			if err != nil {
				t.Fatal(err)
			}
			if program.SourceLanguage != LanguageJava || len(program.Models) != 1 {
				t.Fatalf("program = %#v", program)
			}
			text := string(output)
			if !strings.Contains(text, "Door") {
				t.Fatalf("%s output missing model:\n%s", target, text)
			}
		})
	}
}

func TestJavaFrontendReingestsGeneratedBehaviorMethods(t *testing.T) {
	source := `import com.stateforward.hsm.*;

public final class GeneratedHsm {
  private GeneratedHsm() {}

  private static void Entry1(Context ctx, Instance instance, Event event) {
    instance.set("entered", true);
  }

  private static boolean Guard1(Context ctx, Instance instance, Event event) {
    return true;
  }

  public static final Model DoorModel = Hsm.Define(
    "Door",
    Hsm.Initial(Hsm.Target("closed")),
    Hsm.State(
      "closed",
      Hsm.Entry(GeneratedHsm::Entry1),
      Hsm.Transition(
        Hsm.On("open"),
        Hsm.Guard(GeneratedHsm::Guard1),
        Hsm.Target("../open")
      )
    ),
    Hsm.State("open")
  );
}
`
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "generated.java", Data: []byte(source)}, CompileOptions{
		From:    LanguageJava,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("generated behavior methods should not remain globals: %#v", program.Globals)
	}
	if len(program.Behaviors) != 2 {
		t.Fatalf("behaviors = %d, want 2", len(program.Behaviors))
	}
	var entryBody, guardBody string
	for _, behavior := range program.Behaviors {
		switch behavior.Kind {
		case BehaviorEntry:
			entryBody = behavior.Body
		case BehaviorGuard:
			guardBody = behavior.Body
		}
	}
	if !strings.Contains(entryBody, `instance.set("entered", true);`) {
		t.Fatalf("entry body = %q", entryBody)
	}
	if !strings.Contains(guardBody, "return true;") {
		t.Fatalf("guard body = %q", guardBody)
	}
	text := string(output)
	for _, needle := range []string{
		`// instance.set("entered", true);`,
		`// return true;`,
		`hsm.Entry(Entry1)`,
		`hsm.Guard(Guard1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original java global") || strings.Contains(text, "GeneratedHsm::Entry1") || strings.Contains(text, "GeneratedHsm::Guard1") {
		t.Fatalf("Go output leaked generated Java methods as globals or callable wrappers:\n%s", text)
	}
}

func TestJavaFrontendLowersNamedBehaviorMethods(t *testing.T) {
	source := `import com.stateforward.hsm.*;

public final class DoorHsm {
  private static void entered(Context ctx, Instance instance, Event event) {
    instance.set("entered", true);
  }

  private static boolean allowed(Context ctx, Instance instance, Event event) {
    return event.name().equals("open");
  }

  private static boolean approve(Instance instance) {
    return true;
  }

  private static String helper(String value) {
    return value;
  }

  public static final Model DoorModel = Hsm.Define(
    "Door",
    Hsm.Operation("approve", DoorHsm::approve),
    Hsm.Initial(Hsm.Target("closed")),
    Hsm.State(
      "closed",
      Hsm.Entry(DoorHsm::entered),
      Hsm.Transition(
        Hsm.On("open"),
        Hsm.Guard(DoorHsm::allowed),
        Hsm.Target("../open")
      )
    ),
    Hsm.State("open")
  );
}
`
	program, err := NewJavaFrontend().Parse(context.Background(), SourceInput{Path: "DoorHsm.java", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want operation helper and ordinary helper", program.Globals)
	}
	if !strings.Contains(program.Globals[0].Code+program.Globals[1].Code, "approve") ||
		!strings.Contains(program.Globals[0].Code+program.Globals[1].Code, "helper") {
		t.Fatalf("globals = %#v, want non-HSM-callback methods preserved", program.Globals)
	}
	if len(program.Behaviors) != 3 {
		t.Fatalf("behaviors = %d, want operation reference plus entry and guard", len(program.Behaviors))
	}
	var entryBody, guardBody, operationCode string
	for _, behavior := range program.Behaviors {
		switch behavior.Kind {
		case BehaviorEntry:
			entryBody = behavior.Body
		case BehaviorGuard:
			guardBody = behavior.Body
		case BehaviorOperation:
			operationCode = behavior.Code
		}
		if strings.Contains(behavior.Code, "DoorHsm::entered") || strings.Contains(behavior.Code, "DoorHsm::allowed") {
			t.Fatalf("behavior kept only method reference instead of resolved method body: %#v", behavior)
		}
	}
	if !strings.Contains(entryBody, `instance.set("entered", true);`) {
		t.Fatalf("entry body = %q", entryBody)
	}
	if !strings.Contains(guardBody, `return event.name().equals("open");`) {
		t.Fatalf("guard body = %q", guardBody)
	}
	if operationCode != "DoorHsm::approve" {
		t.Fatalf("operation code = %q, want unresolved operation reference", operationCode)
	}
}

func TestJavaFrontendReingestsGeneratedOperationMethod(t *testing.T) {
	source := `import com.stateforward.hsm.*;

public final class GeneratedHsm {
  private GeneratedHsm() {}

  private static void Operation1(Context ctx, Instance instance, Event event) {
    instance.set("approved", true);
  }

  public static final Model DoorModel = Hsm.Define(
    "Door",
    Hsm.Operation("approve", GeneratedHsm::Operation1),
    Hsm.Initial(Hsm.Target("closed")),
    Hsm.State(
      "closed",
      Hsm.Transition(
        Hsm.OnCall("approve"),
        Hsm.Target("../open")
      )
    ),
    Hsm.State("open")
  );
}
`
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "generated.java", Data: []byte(source)}, CompileOptions{
		From:    LanguageJava,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("generated operation method should not remain global: %#v", program.Globals)
	}
	if len(program.Behaviors) != 1 || program.Behaviors[0].Kind != BehaviorOperation {
		t.Fatalf("behaviors = %#v, want one operation behavior", program.Behaviors)
	}
	if !strings.Contains(program.Behaviors[0].Body, `instance.set("approved", true);`) {
		t.Fatalf("operation body = %q", program.Behaviors[0].Body)
	}
	if len(program.Models) != 1 || len(program.Models[0].Operations) != 1 || program.Models[0].Operations[0].Behavior == nil {
		t.Fatalf("operations = %#v", program.Models)
	}
	text := string(output)
	for _, needle := range []string{
		`// instance.set("approved", true);`,
		`hsm.Operation("approve", Operation1)`,
		`hsm.OnCall("approve")`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original java global") || strings.Contains(text, "GeneratedHsm::Operation1") {
		t.Fatalf("Go output leaked generated Java operation wrapper:\n%s", text)
	}
}
