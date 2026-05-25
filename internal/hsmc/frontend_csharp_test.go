package hsmc

import (
	"context"
	"strings"
	"testing"
)

const csharpDoorSource = `using Stateforward.Hsm;
using Helpers;

public static class Sample
{
    static string Record(string value) => value;
    static readonly Event Opened = new Event("open");
    public static readonly Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed",
            Hsm.Entry<Instance>((ctx, instance, @event) => Record("closed")),
            Hsm.Transition(
                Hsm.On(Opened),
                Hsm.Guard<Instance>((ctx, instance, @event) => true),
                Hsm.Target("../open"),
                Hsm.Effect<Instance>((ctx, instance, @event) => Record(@event.Name))
            )
        ),
        Hsm.State("open")
    );
}
`

func TestCSharpFrontendLowersModelAndBehaviorBoundaries(t *testing.T) {
	program, err := NewCSharpFrontend().Parse(context.Background(), SourceInput{Path: "door.cs", Data: []byte(csharpDoorSource)})
	if err != nil {
		t.Fatal(err)
	}
	if program.SourceLanguage != LanguageCSharp {
		t.Fatalf("source language = %q, want csharp", program.SourceLanguage)
	}
	if len(program.Imports) != 1 || program.Imports[0].Path != "Helpers" {
		t.Fatalf("imports = %#v", program.Imports)
	}
	if len(program.Events) != 1 || program.Events[0].Symbol != "Opened" || program.Events[0].Name != "open" {
		t.Fatalf("events = %#v", program.Events)
	}
	if len(program.Models) != 1 {
		t.Fatalf("models = %d, want 1", len(program.Models))
	}
	model := program.Models[0]
	if model.Name != "Door" || model.Initializer == nil || model.Initializer.Target != "closed" {
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
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerOn || transition.Trigger.Values[0].EventSymbol != "Opened" {
		t.Fatalf("trigger = %#v", transition.Trigger)
	}
	if transition.Guard == nil || len(transition.Effects) != 1 || transition.Target != "../open" {
		t.Fatalf("transition = %#v", transition)
	}
	if len(program.Behaviors) != 3 {
		t.Fatalf("behaviors = %d, want 3", len(program.Behaviors))
	}
	for _, behavior := range program.Behaviors {
		if behavior.SourceLanguage != LanguageCSharp || behavior.OwnerID == "" || behavior.Kind == "" || behavior.Body == "" {
			t.Fatalf("incomplete behavior = %#v", behavior)
		}
	}
}

const csharpAliasUsingSource = `using Stateforward.Hsm;
using Format = Sample.Formatting.Helper;

public static class Sample
{
    public static readonly Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed",
            Hsm.Entry<Instance>((ctx, instance, @event) => Format.Record("closed"))
        )
    );
}
`

func TestCSharpFrontendPreservesAliasUsing(t *testing.T) {
	program, err := NewCSharpFrontend().Parse(context.Background(), SourceInput{Path: "door.cs", Data: []byte(csharpAliasUsingSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 1 {
		t.Fatalf("imports = %#v", program.Imports)
	}
	imp := program.Imports[0]
	if imp.Path != "Sample.Formatting.Helper" || imp.Alias != "Format" || imp.Language != LanguageCSharp {
		t.Fatalf("import metadata = %#v", imp)
	}
	if len(imp.LocalNames) != 1 || imp.LocalNames[0] != "Format" {
		t.Fatalf("import local names = %#v", imp.LocalNames)
	}
}

func TestCSharpFrontendPreservesStaticUsingMetadata(t *testing.T) {
	program, err := NewCSharpFrontend().Parse(context.Background(), SourceInput{Path: "door.cs", Data: []byte(csharpStaticHelperUsingSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 1 {
		t.Fatalf("imports = %#v", program.Imports)
	}
	imp := program.Imports[0]
	if imp.Path != "Sample.Formatting.Helper" || !imp.Static || imp.Language != LanguageCSharp {
		t.Fatalf("static using metadata = %#v", imp)
	}
	if len(imp.LocalNames) != 1 || imp.LocalNames[0] != "*" {
		t.Fatalf("static using local names = %#v", imp.LocalNames)
	}
}

const csharpStaticRuntimeUsingSource = `using Stateforward.Hsm;
using static Stateforward.Hsm.Hsm;

public static class Sample
{
    static readonly Event Opened = new Event("open");
    public static readonly Model DoorModel = Define(
        "Door",
        Initial(Target("closed")),
        State("closed",
            Entry<Instance>((ctx, instance, @event) => instance.Set("log", "closed")),
            Transition(
                On(Opened),
                Target("../open")
            )
        ),
        State("open")
    );
}
`

func TestCSharpFrontendLowersStaticImportedRuntimeCalls(t *testing.T) {
	program, err := NewCSharpFrontend().Parse(context.Background(), SourceInput{Path: "door.cs", Data: []byte(csharpStaticRuntimeUsingSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 0 {
		t.Fatalf("runtime usings should be compiler-owned, got %#v", program.Imports)
	}
	if len(program.Events) != 1 || program.Events[0].Symbol != "Opened" || program.Events[0].Name != "open" {
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
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerOn || len(transition.Trigger.Values) != 1 || transition.Trigger.Values[0].EventSymbol != "Opened" {
		t.Fatalf("trigger = %#v", transition.Trigger)
	}
	if len(program.Behaviors) != 1 || !strings.Contains(program.Behaviors[0].Body, `instance.Set("log", "closed");`) {
		t.Fatalf("behaviors = %#v", program.Behaviors)
	}
}

func TestCompilerPreservesCSharpStaticHelperUsingInCSharpTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.cs", Data: []byte(csharpStaticHelperUsingSource)}, CompileOptions{
		From:    LanguageCSharp,
		To:      LanguageCSharp,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"using static Sample.Formatting.Helper;",
		`Record("closed");`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C# output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "using Sample.Formatting.Helper;") {
		t.Fatalf("C# output rendered static helper using as ordinary using:\n%s", text)
	}
}

const csharpAliasedRuntimeUsingSource = `using H = Stateforward.Hsm.Hsm;
using Stateforward.Hsm;

public static class Sample
{
    static readonly Event Opened = new Event("open");
    public static readonly Model DoorModel = H.Define(
        "Door",
        H.Initial(H.Target("closed")),
        H.State("closed",
            H.Entry<Instance>((ctx, instance, @event) => instance.Set("log", "closed")),
            H.Transition(
                H.On(Opened),
                H.Target("../open")
            )
        ),
        H.State("open")
    );
}
`

func TestCSharpFrontendNormalizesAliasedRuntimeUsing(t *testing.T) {
	program, err := NewCSharpFrontend().Parse(context.Background(), SourceInput{Path: "door.cs", Data: []byte(csharpAliasedRuntimeUsingSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 0 {
		t.Fatalf("runtime using aliases should be compiler-owned, got %#v", program.Imports)
	}
	if len(program.Events) != 1 || program.Events[0].Symbol != "Opened" {
		t.Fatalf("events = %#v", program.Events)
	}
	if len(program.Behaviors) != 1 || strings.Contains(program.Behaviors[0].Code, "H.") {
		t.Fatalf("behavior did not normalize runtime alias: %#v", program.Behaviors)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
}

func TestCSharpFrontendRejectsUnknownHSMDslPartials(t *testing.T) {
	source := `using H = Stateforward.Hsm.Hsm;
using Stateforward.Hsm;
using static Stateforward.Hsm.Hsm;

public static class Sample
{
    public static readonly Model DoorModel = H.Define(
        "Door",
        H.UnknownModelPartial(),
        H.Initial(H.UnknownInitialPartial(), H.Target("closed")),
        H.State("closed",
            H.UnknownStatePartial(),
            H.Transition(
                UnknownTransitionPartial(),
                H.Target("../open")
            )
        )
    );
}
`
	_, err := NewCSharpFrontend().Parse(context.Background(), SourceInput{Path: "door.cs", Data: []byte(source)})
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

func TestCompilerConvertsAliasedCSharpRuntimeSourceToCSharp(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.cs", Data: []byte(csharpAliasedRuntimeUsingSource)}, CompileOptions{
		From:    LanguageCSharp,
		To:      LanguageCSharp,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"using Stateforward.Hsm;",
		"private static readonly Event Opened = new Event(\"open\");",
		"public static readonly Model DoorModel = Hsm.Define(",
		"Hsm.Entry<Instance>(Entry1)",
		"Hsm.On(Opened)",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C# output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "H.") || strings.Contains(text, "using H = Stateforward.Hsm.Hsm") {
		t.Fatalf("C# output retained source runtime alias:\n%s", text)
	}
}

func TestCompilerCompilesCSharpToCSharp(t *testing.T) {
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.cs", Data: []byte(csharpDoorSource)}, CompileOptions{
		From:    LanguageCSharp,
		To:      LanguageCSharp,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if program.SourceLanguage != LanguageCSharp {
		t.Fatalf("source language = %q, want csharp", program.SourceLanguage)
	}
	text := string(output)
	for _, needle := range []string{
		"using Stateforward.Hsm;",
		"using Helpers;",
		"private static readonly Event Opened = new Event(\"open\");",
		"public static readonly Model DoorModel = Hsm.Define(",
		"Hsm.On(Opened)",
		"Hsm.Guard<Instance>(Guard1)",
		`Record("closed");`,
		"return true;",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C# output missing %q:\n%s", needle, text)
		}
	}
}

func TestCSharpFrontendLowersAttributesOperationsAndExtendedTriggers(t *testing.T) {
	source := `using Stateforward.Hsm;

public static class Sample
{
    static string Record(string value) => value;
    static object Operation1(object ctx, object instance, object @event) => Record("approve");
    public static readonly Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Attribute("count", typeof(int), 1),
        Hsm.Operation("approve", Operation1),
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed",
            Hsm.Transition(Hsm.OnCall("approve"), Hsm.Target("../open")),
            Hsm.Transition(Hsm.OnSet("count"), Hsm.Target("../open")),
            Hsm.Transition(Hsm.At((ctx, instance, @event) => DateTimeOffset.UnixEpoch), Hsm.Target("../open")),
            Hsm.Transition(Hsm.When((ctx, instance, @event) => true), Hsm.Target("../open"))
        ),
        Hsm.State("open")
    );
}
`
	program, err := NewCSharpFrontend().Parse(context.Background(), SourceInput{Path: "door.cs", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	model := program.Models[0]
	if len(model.Attributes) != 1 || model.Attributes[0].Name != "count" || model.Attributes[0].Type != "typeof(int)" || model.Attributes[0].Default != "1" {
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
	if err := Validate(program); err != nil {
		t.Fatalf("Validate() = %v", err)
	}

	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.cs", Data: []byte(source)}, CompileOptions{
		From:    LanguageCSharp,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`// Type: typeof(int)`,
		`hsm.Attribute("count", 1)`,
		`hsm.Operation("approve", Operation1)`,
		`hsm.OnCall("approve")`,
		`hsm.OnSet("count")`,
		`// Original csharp behavior trigger_1 preserved for manual porting:`,
		`// Original csharp behavior trigger_2 preserved for manual porting:`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
}

func TestCSharpFrontendLowersGeneratedOperationWrapper(t *testing.T) {
	source := `using Stateforward.Hsm;

public static class GeneratedHsm
{
    private static void Operation1(Context ctx, Instance instance, Event @event)
    {
        System.GC.KeepAlive("approve");
    }

    public static readonly Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Operation("approve", new Operation<Instance>(Operation1)),
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed", Hsm.Transition(Hsm.OnCall("approve"), Hsm.Target("../open"))),
        Hsm.State("open")
    );
}
`
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "generated.cs", Data: []byte(source)}, CompileOptions{
		From:    LanguageCSharp,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("generated operation method should not remain global: %#v", program.Globals)
	}
	if len(program.Behaviors) != 1 || program.Behaviors[0].Kind != BehaviorOperation || !strings.Contains(program.Behaviors[0].Body, `System.GC.KeepAlive("approve");`) {
		t.Fatalf("operation behavior = %#v", program.Behaviors)
	}
	text := string(output)
	for _, needle := range []string{
		`hsm.Operation("approve", Operation1)`,
		`// System.GC.KeepAlive("approve");`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, `new Operation<Instance>`) || strings.Contains(text, `Operation1(ctx, instance, @event)`) {
		t.Fatalf("Go output preserved generated C# operation wrapper as behavior body:\n%s", text)
	}
}

func TestCSharpFrontendReingestsGeneratedBehaviorMethods(t *testing.T) {
	source := `using Stateforward.Hsm;

public static class GeneratedHsm
{
    private static void Entry1(Context ctx, Instance instance, Event @event)
    {
        instance.Set("entered", true);
    }

    private static bool Guard1(Context ctx, Instance instance, Event @event)
    {
        return true;
    }

    public static readonly Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed",
            Hsm.Entry<Instance>(Entry1),
            Hsm.Transition(
                Hsm.On("open"),
                Hsm.Guard<Instance>(Guard1),
                Hsm.Target("../open")
            )
        ),
        Hsm.State("open")
    );
}
`
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "generated.cs", Data: []byte(source)}, CompileOptions{
		From:    LanguageCSharp,
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
	if !strings.Contains(entryBody, `instance.Set("entered", true);`) {
		t.Fatalf("entry body = %q", entryBody)
	}
	if !strings.Contains(guardBody, "return true;") {
		t.Fatalf("guard body = %q", guardBody)
	}
	text := string(output)
	for _, needle := range []string{
		`// instance.Set("entered", true);`,
		`// return true;`,
		`hsm.Entry(Entry1)`,
		`hsm.Guard(Guard1)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original csharp global") || strings.Contains(text, "Entry1(ctx, instance, @event)") || strings.Contains(text, "Guard1(ctx, instance, @event)") {
		t.Fatalf("Go output leaked generated C# methods as globals or callable wrappers:\n%s", text)
	}
}

func TestCSharpFrontendLowersNamedBehaviorMethods(t *testing.T) {
	source := `using Stateforward.Hsm;

public static class Sample
{
    private static void Entered(Context ctx, Instance instance, Event @event)
    {
        instance.Set("entered", true);
    }

    private static bool Allowed(Context ctx, Instance instance, Event @event)
    {
        return @event.Name == "open";
    }

    private static bool Approve(Instance instance)
    {
        return true;
    }

    private static string Helper(string value)
    {
        return value;
    }

    public static readonly Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Operation("approve", Approve),
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed",
            Hsm.Entry<Instance>(Entered),
            Hsm.Transition(
                Hsm.On("open"),
                Hsm.Guard<Instance>(Allowed),
                Hsm.Target("../open")
            )
        ),
        Hsm.State("open")
    );
}
`
	program, err := NewCSharpFrontend().Parse(context.Background(), SourceInput{Path: "named.cs", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want operation helper and ordinary helper", program.Globals)
	}
	if !strings.Contains(program.Globals[0].Code+program.Globals[1].Code, "Approve") ||
		!strings.Contains(program.Globals[0].Code+program.Globals[1].Code, "Helper") {
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
		if behavior.Code == "Entered" || behavior.Code == "Allowed" {
			t.Fatalf("behavior kept only identifier instead of resolved method body: %#v", behavior)
		}
	}
	if !strings.Contains(entryBody, `instance.Set("entered", true);`) {
		t.Fatalf("entry body = %q", entryBody)
	}
	if !strings.Contains(guardBody, `return @event.Name == "open";`) {
		t.Fatalf("guard body = %q", guardBody)
	}
	if operationCode != "Approve" {
		t.Fatalf("operation code = %q, want unresolved operation reference", operationCode)
	}
}
