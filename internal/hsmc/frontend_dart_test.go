package hsmc

import (
	"context"
	"strings"
	"testing"
)

const dartDoorSource = `import 'package:hsm/hsm.dart';
import 'package:sample/format.dart' as format;

final opened = Event(name: 'open');

class Door extends Instance {
  final List<String> log = [];
}

String record(String value) => format.record(value);

final doorModel = define('Door', [
  initial(target('closed')),
  state('closed', [
    entry((ctx, instance, event) {
      record('closed');
    }),
    transition([
      on(opened.name),
      guard((ctx, instance, event) => true),
      target('../open'),
      effect((ctx, instance, event) {
        record(event.name);
      }),
    ]),
  ]),
  state('open'),
]);
`

func TestDartFrontendLowersModelAndBehaviors(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartDoorSource)})
	if err != nil {
		t.Fatal(err)
	}
	if program.SourceLanguage != LanguageDart {
		t.Fatalf("source language = %q, want Dart", program.SourceLanguage)
	}
	if len(program.Imports) != 1 || program.Imports[0].Path != "package:sample/format.dart" || program.Imports[0].Alias != "format" {
		t.Fatalf("imports = %#v", program.Imports)
	}
	if len(program.Events) != 1 || program.Events[0].Symbol != "opened" || program.Events[0].Name != "open" {
		t.Fatalf("events = %#v", program.Events)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %d, want class and helper function", len(program.Globals))
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
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerOn || len(transition.Trigger.Values) != 1 || transition.Trigger.Values[0].EventSymbol != "opened" || transition.Trigger.Language != LanguageDart {
		t.Fatalf("trigger = %#v", transition.Trigger)
	}
	if transition.Guard == nil || len(transition.Effects) != 1 {
		t.Fatalf("transition behaviors not lowered: %#v", transition)
	}
	if len(program.Behaviors) != 3 {
		t.Fatalf("behaviors = %d, want 3", len(program.Behaviors))
	}
	var guardBody string
	for _, behavior := range program.Behaviors {
		if behavior.SourceLanguage != LanguageDart {
			t.Fatalf("behavior source language = %q", behavior.SourceLanguage)
		}
		if behavior.Kind == BehaviorGuard {
			guardBody = behavior.Body
		}
	}
	if guardBody != "return true;" {
		t.Fatalf("guard expression body not normalized: %q", guardBody)
	}
}

const dartPascalCaseSource = `import 'package:hsm/hsm.dart';

final opened = Event(Name: 'open');

final doorModel = Define('Door', [
  Initial(Target('closed')),
  State('closed', [
    Entry((ctx, instance, event) {
      instance.set('entered', true);
    }),
    Transition([
      On(opened.name),
      Target('../open'),
    ]),
  ]),
  State('open'),
]);
`

func TestDartFrontendLowersCanonicalPascalCaseDSL(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartPascalCaseSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Events) != 1 || program.Events[0].Symbol != "opened" || program.Events[0].Name != "open" {
		t.Fatalf("events = %#v", program.Events)
	}
	if len(program.Models) != 1 {
		t.Fatalf("models = %#v", program.Models)
	}
	model := program.Models[0]
	if model.Name != "Door" || model.Initializer == nil || model.Initializer.Target != "closed" {
		t.Fatalf("model = %#v", model)
	}
	if len(model.States) != 2 || len(model.States[0].Behaviors) != 1 || len(model.States[0].Transitions) != 1 {
		t.Fatalf("states = %#v", model.States)
	}
	transition := model.States[0].Transitions[0]
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerOn || len(transition.Trigger.Values) != 1 || transition.Trigger.Values[0].EventSymbol != "opened" {
		t.Fatalf("trigger = %#v", transition.Trigger)
	}
}

const dartShowImportSource = `import 'package:hsm/hsm.dart';
import 'package:sample/format.dart' show record, normalize;

final doorModel = define('Door', [
  initial(target('closed')),
  state('closed', [
    entry((ctx, instance, event) {
      record(normalize('closed'));
    }),
  ]),
]);
`

const dartPrefixedShowImportSource = `import 'package:hsm/hsm.dart';
import 'package:sample/format.dart' as fmt show record, normalize;

final doorModel = define('Door', [
  initial(target('closed')),
  state('closed', [
    entry((ctx, instance, event) {
      fmt.record(fmt.normalize('closed'));
    }),
  ]),
]);
`

const dartHideImportSource = `import 'package:hsm/hsm.dart';
import 'package:sample/format.dart' hide debugRecord, privateNormalize;

final doorModel = define('Door', [
  initial(target('closed')),
  state('closed', [
    entry((ctx, instance, event) {
      record('closed');
    }),
  ]),
]);
`

const dartDeferredImportSource = `import 'package:hsm/hsm.dart';
import 'package:sample/lazy_format.dart' deferred as lazyFormat;

final doorModel = define('Door', [
  initial(target('closed')),
  state('closed', [
    entry((ctx, instance, event) {
      lazyFormat.record('closed');
    }),
  ]),
]);
`

func TestDartFrontendPreservesShowImportSpecifiers(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartShowImportSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 1 {
		t.Fatalf("imports = %#v", program.Imports)
	}
	imp := program.Imports[0]
	if imp.Path != "package:sample/format.dart" || imp.Alias != "" || imp.Language != LanguageDart {
		t.Fatalf("import metadata = %#v", imp)
	}
	if len(imp.Specifiers) != 2 || imp.Specifiers[0].Name != "record" || imp.Specifiers[1].Name != "normalize" {
		t.Fatalf("import specifiers = %#v", imp.Specifiers)
	}
	if len(imp.LocalNames) != 2 || imp.LocalNames[0] != "record" || imp.LocalNames[1] != "normalize" {
		t.Fatalf("import local names = %#v", imp.LocalNames)
	}
}

func TestDartFrontendPreservesPrefixedShowImportSpecifiers(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartPrefixedShowImportSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 1 {
		t.Fatalf("imports = %#v", program.Imports)
	}
	imp := program.Imports[0]
	if imp.Path != "package:sample/format.dart" || imp.Alias != "fmt" || imp.Language != LanguageDart {
		t.Fatalf("import metadata = %#v", imp)
	}
	if len(imp.Specifiers) != 2 || imp.Specifiers[0].Name != "record" || imp.Specifiers[1].Name != "normalize" {
		t.Fatalf("import specifiers = %#v", imp.Specifiers)
	}
	if strings.Join(imp.LocalNames, ",") != "fmt" {
		t.Fatalf("import local names = %#v, want only prefix alias", imp.LocalNames)
	}
}

func TestCompilerPreservesDartPrefixedShowImportInDartTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartPrefixedShowImportSource)}, CompileOptions{
		From:    LanguageDart,
		To:      LanguageDart,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, `import "package:sample/format.dart" as fmt show record, normalize;`) {
		t.Fatalf("Dart output missing prefixed show import:\n%s", text)
	}
}

func TestDartFrontendPreservesHideImportSpecifiers(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartHideImportSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 1 {
		t.Fatalf("imports = %#v", program.Imports)
	}
	imp := program.Imports[0]
	if imp.Path != "package:sample/format.dart" || imp.Alias != "" || imp.Language != LanguageDart {
		t.Fatalf("import metadata = %#v", imp)
	}
	if len(imp.Specifiers) != 0 {
		t.Fatalf("hide import should not become show specifiers: %#v", imp.Specifiers)
	}
	if strings.Join(imp.Hidden, ",") != "debugRecord,privateNormalize" {
		t.Fatalf("hidden names = %#v", imp.Hidden)
	}
}

func TestDartFrontendPreservesDeferredImports(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartDeferredImportSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 1 {
		t.Fatalf("imports = %#v", program.Imports)
	}
	imp := program.Imports[0]
	if imp.Path != "package:sample/lazy_format.dart" || imp.Alias != "lazyFormat" || !imp.Deferred || imp.Language != LanguageDart {
		t.Fatalf("deferred import metadata = %#v", imp)
	}
	if strings.Join(imp.LocalNames, ",") != "lazyFormat" {
		t.Fatalf("deferred import local names = %#v", imp.LocalNames)
	}
}

func TestCompilerPreservesDartDeferredImportsInDartTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartDeferredImportSource)}, CompileOptions{
		From:    LanguageDart,
		To:      LanguageDart,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, `import "package:sample/lazy_format.dart" deferred as lazyFormat;`) {
		t.Fatalf("Dart output missing deferred import:\n%s", text)
	}
	if strings.Contains(text, `import "package:sample/lazy_format.dart" as lazyFormat;`) {
		t.Fatalf("Dart output rendered deferred import as eager prefix import:\n%s", text)
	}
}

const dartExportDirectiveSource = `import 'package:hsm/hsm.dart';
export 'package:sample/public_api.dart';

final doorModel = define('Door', [
  initial(target('closed')),
  state('closed'),
]);
`

func TestDartFrontendDoesNotTreatExportDirectivesAsImports(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartExportDirectiveSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 0 {
		t.Fatalf("imports = %#v, want export directive ignored instead of rendered as import", program.Imports)
	}
}

func TestCompilerConvertsDartSourceToDartTarget(t *testing.T) {
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartDoorSource)}, CompileOptions{
		From:    LanguageDart,
		To:      LanguageDart,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if program.SourceLanguage != LanguageDart {
		t.Fatalf("program source = %q, want Dart", program.SourceLanguage)
	}
	text := string(output)
	for _, needle := range []string{
		`import "package:sample/format.dart" as format;`,
		"final doorModel = define(",
		"entry(entry1)",
		"on(opened.name)",
		"guard(guard1)",
		"return true;",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Dart output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original dart behavior") {
		t.Fatalf("Dart-to-Dart should preserve behavior bodies, got:\n%s", text)
	}
}

func TestCompilerPreservesDartHideImportsInDartTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartHideImportSource)}, CompileOptions{
		From:    LanguageDart,
		To:      LanguageDart,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, `import "package:sample/format.dart" hide debugRecord, privateNormalize;`) {
		t.Fatalf("Dart output did not preserve hide import:\n%s", text)
	}
}

const dartExtensionTypeGlobalSource = `import 'package:hsm/hsm.dart';

extension type DoorTag(String value) {
  bool get isOpen => value == 'open';
}

final doorModel = define('Door', [
  initial(target('closed')),
  state('closed'),
]);
`

func TestDartFrontendPreservesExtensionTypeGlobals(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartExtensionTypeGlobalSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 1 || !strings.Contains(program.Globals[0].Code, "extension type DoorTag") {
		t.Fatalf("globals = %#v, want extension type global", program.Globals)
	}
}

func TestNoAdapterDartExtensionTypeGlobalIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartExtensionTypeGlobalSource)}, CompileOptions{
		From:    LanguageDart,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original dart global",
		"// extension type DoorTag(String value) {",
		"//   bool get isOpen => value == 'open';",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

const dartAliasedRuntimeSource = `import 'package:hsm/hsm.dart' as hsm;

final opened = hsm.Event(name: 'open');

final doorModel = hsm.define('Door', [
  hsm.initial(hsm.target('closed')),
  hsm.state('closed', [
    hsm.entry((ctx, instance, event) {
      instance.set('entered', true);
    }),
    hsm.transition([
      hsm.on(opened.name),
      hsm.target('../open'),
    ]),
  ]),
  hsm.state('open'),
]);
`

func TestDartFrontendLowersAliasedRuntimeCalls(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "aliased.dart", Data: []byte(dartAliasedRuntimeSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 0 {
		t.Fatalf("hsm runtime alias import should be compiler-owned, got %#v", program.Imports)
	}
	if len(program.Events) != 1 || program.Events[0].Symbol != "opened" || program.Events[0].Name != "open" {
		t.Fatalf("events = %#v", program.Events)
	}
	if len(program.Models) != 1 {
		t.Fatalf("models = %#v, want one aliased runtime model", program.Models)
	}
	model := program.Models[0]
	if model.Name != "Door" || model.Initializer == nil || model.Initializer.Target != "closed" {
		t.Fatalf("model = %#v", model)
	}
	if len(model.States) != 2 || len(model.States[0].Behaviors) != 1 || len(model.States[0].Transitions) != 1 {
		t.Fatalf("states = %#v", model.States)
	}
	transition := model.States[0].Transitions[0]
	if transition.Trigger == nil || transition.Trigger.Kind != TriggerOn || len(transition.Trigger.Values) != 1 || transition.Trigger.Values[0].EventSymbol != "opened" {
		t.Fatalf("trigger = %#v", transition.Trigger)
	}
	if len(program.Behaviors) != 1 || !strings.Contains(program.Behaviors[0].Body, "instance.set('entered', true);") {
		t.Fatalf("behaviors = %#v", program.Behaviors)
	}
}

func TestDartFrontendRejectsUnknownHSMDslPartials(t *testing.T) {
	source := `import 'package:hsm/hsm.dart';
import 'package:hsm/hsm.dart' as hsm;

final doorModel = hsm.define('Door', [
  hsm.unknownModelPartial(),
  hsm.initial([hsm.unknownInitialPartial(), hsm.target('closed')]),
  hsm.state('closed', [
    hsm.unknownStatePartial(),
    hsm.transition([
      unknownTransitionPartial(),
      hsm.target('../open'),
    ]),
  ]),
]);
`
	_, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "door.dart", Data: []byte(source)})
	if err == nil {
		t.Fatal("expected unknown hsm partial diagnostics")
	}
	for _, want := range []string{
		`unknown or unsupported hsm.unknownModelPartial partial in model context`,
		`unknown or unsupported hsm.unknownInitialPartial partial in initial context`,
		`unknown or unsupported hsm.unknownStatePartial partial in state context`,
		`unknown or unsupported hsm.unknownTransitionPartial partial in transition context`,
	} {
		if !diagnosticsContain(err, want) {
			t.Fatalf("diagnostics = %v, want %q", err, want)
		}
	}
}

const dartGeneratedBehaviorSource = `import 'package:hsm/hsm.dart';

void entry1(Context ctx, Instance instance, Event event) {
  instance.set('entered', true);
}

bool guard1(Context ctx, Instance instance, Event event) {
  return true;
}

final doorModel = define('Door', [
  initial(target('closed')),
  state('closed', [
    entry(entry1),
    transition([
      on('open'),
      guard(guard1),
      target('../open'),
    ]),
  ]),
  state('open'),
]);
`

func TestDartFrontendReingestsGeneratedBehaviorDeclarations(t *testing.T) {
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "generated.dart", Data: []byte(dartGeneratedBehaviorSource)}, CompileOptions{
		From:    LanguageDart,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("globals = %#v, want generated behavior declarations removed", program.Globals)
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
	if !strings.Contains(entryBody, "instance.set('entered', true);") {
		t.Fatalf("entry body = %q", entryBody)
	}
	if guardBody != "return true;" {
		t.Fatalf("guard body = %q, want return true;", guardBody)
	}
	text := string(output)
	for _, needle := range []string{
		"hsm.Entry(Entry1)",
		"hsm.Guard(Guard1)",
		"// Original dart behavior",
		"instance.set('entered', true);",
		"return true;",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original dart global") {
		t.Fatalf("generated behavior declaration leaked as global:\n%s", text)
	}
}

const dartNamedBehaviorSource = `import 'package:hsm/hsm.dart';

void entered(Context ctx, Instance instance, Event event) {
  instance.set('entered', true);
}

bool allowed(Context ctx, Instance instance, Event event) {
  return event.name == 'open';
}

String helper(String value) => value.toUpperCase();

final doorModel = define('Door', [
  initial(target('closed')),
  state('closed', [
    entry(entered),
    transition([
      on('open'),
      guard(allowed),
      target('../open'),
    ]),
  ]),
  state('open'),
]);
`

func TestDartFrontendLowersNamedBehaviorDeclarations(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "named.dart", Data: []byte(dartNamedBehaviorSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 1 || !strings.Contains(program.Globals[0].Code, "helper") {
		t.Fatalf("globals = %#v, want only non-behavior helper", program.Globals)
	}
	if len(program.Behaviors) != 2 {
		t.Fatalf("behaviors = %d, want entry and guard", len(program.Behaviors))
	}
	var entryBody, guardBody string
	for _, behavior := range program.Behaviors {
		switch behavior.Kind {
		case BehaviorEntry:
			entryBody = behavior.Body
		case BehaviorGuard:
			guardBody = behavior.Body
		}
		if behavior.Code == "entered" || behavior.Code == "allowed" {
			t.Fatalf("behavior kept only identifier instead of resolved function body: %#v", behavior)
		}
	}
	if !strings.Contains(entryBody, "instance.set('entered', true);") {
		t.Fatalf("entry body = %q", entryBody)
	}
	if guardBody != "return event.name == 'open';" {
		t.Fatalf("guard body = %q, want normalized return", guardBody)
	}
}

const dartNamedFunctionExpressionBehaviorSource = `import 'package:hsm/hsm.dart';

final entered = (Context ctx, Instance instance, Event event) {
  instance.set('entered', true);
};

final doorModel = define('Door', [
  initial(target('closed')),
  state('closed', [
    entry(entered),
  ]),
  state('open'),
]);
`

func TestDartFrontendLowersNamedFunctionExpressionBehavior(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "named_expr.dart", Data: []byte(dartNamedFunctionExpressionBehaviorSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("globals = %#v, want referenced behavior expression removed", program.Globals)
	}
	if len(program.Behaviors) != 1 || program.Behaviors[0].Kind != BehaviorEntry {
		t.Fatalf("behaviors = %#v, want one entry behavior", program.Behaviors)
	}
	if !strings.Contains(program.Behaviors[0].Body, "instance.set('entered', true);") {
		t.Fatalf("entry body = %q", program.Behaviors[0].Body)
	}
	if program.Behaviors[0].Code == "entered" {
		t.Fatalf("behavior kept only identifier instead of resolving function expression: %#v", program.Behaviors[0])
	}
}

const dartGeneratedFunctionExpressionSource = `import 'package:hsm/hsm.dart';

final entry1 = (Context ctx, Instance instance, Event event) {
  instance.set('entered', true);
}, helper = () => true;

final guard1 = (Context ctx, Instance instance, Event event) {
  return true;
};

final doorModel = define('Door', [
  initial(target('closed')),
  state('closed', [
    entry(entry1),
    transition([
      on('open'),
      guard(guard1),
      target('../open'),
    ]),
  ]),
  state('open'),
]);
`

func TestDartFrontendReingestsGeneratedBehaviorFunctionExpressions(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "generated_expr.dart", Data: []byte(dartGeneratedFunctionExpressionSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 1 || !strings.Contains(program.Globals[0].Code, "helper") {
		t.Fatalf("globals = %#v, want only helper", program.Globals)
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
	if !strings.Contains(entryBody, "instance.set('entered', true);") {
		t.Fatalf("entry body = %q", entryBody)
	}
	if guardBody != "return true;" {
		t.Fatalf("guard body = %q, want return true;", guardBody)
	}
}

const dartGeneratedOperationSource = `import 'package:hsm/hsm.dart';

void operation1(Context ctx, Instance instance, Event event) {
  instance.set('approved', true);
}

final doorModel = define('Door', [
  operation('approve', operation1),
  initial(target('closed')),
  state('closed', [
    transition([
      onCall('approve'),
      target('../open'),
    ]),
  ]),
  state('open'),
]);
`

func TestDartFrontendReingestsGeneratedOperationDeclaration(t *testing.T) {
	output, program, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "generated_operation.dart", Data: []byte(dartGeneratedOperationSource)}, CompileOptions{
		From:    LanguageDart,
		To:      LanguageGo,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("globals = %#v, want generated operation declaration removed", program.Globals)
	}
	if len(program.Behaviors) != 1 || program.Behaviors[0].Kind != BehaviorOperation {
		t.Fatalf("behaviors = %#v, want one operation behavior", program.Behaviors)
	}
	if !strings.Contains(program.Behaviors[0].Body, "instance.set('approved', true);") {
		t.Fatalf("operation body = %q", program.Behaviors[0].Body)
	}
	text := string(output)
	for _, needle := range []string{
		"hsm.Operation(\"approve\", Operation1)",
		"hsm.OnCall(\"approve\")",
		"// Original dart behavior operation_1 preserved for manual porting:",
		"// instance.set('approved', true);",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Original dart global") {
		t.Fatalf("generated operation declaration leaked as global:\n%s", text)
	}
}

const dartRichSource = `import 'package:hsm/hsm.dart';

bool approve(Instance instance) => true;

final richDoorModel = define('RichDoor', [
  attribute('count', int, 1),
  operation('approve', approve),
  initial(target('closed')),
  state('closed', [
    transition([
      onCall('approve'),
      target('../open'),
    ]),
    transition([
      onSet('count'),
      target('../open'),
    ]),
    transition([
      at((ctx, instance, event) => DateTime.now()),
      target('../open'),
    ]),
    transition([
      every((ctx, instance, event) => Duration(seconds: 1)),
      target('../open'),
    ]),
    transition([
      when((ctx, instance, event) => true),
      target('../open'),
    ]),
  ]),
  state('open'),
]);
`

func TestDartFrontendLowersAttributesOperationsAndExtendedTriggers(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "rich.dart", Data: []byte(dartRichSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 {
		t.Fatalf("models = %d, want 1", len(program.Models))
	}
	model := program.Models[0]
	if len(model.Attributes) != 1 || model.Attributes[0].Name != "count" || model.Attributes[0].Type != "int" || model.Attributes[0].Default != "1" {
		t.Fatalf("attributes = %#v", model.Attributes)
	}
	if len(model.Operations) != 1 || model.Operations[0].Name != "approve" || model.Operations[0].Behavior == nil {
		t.Fatalf("operations = %#v", model.Operations)
	}
	if len(model.States) == 0 || len(model.States[0].Transitions) != 5 {
		t.Fatalf("state transitions = %#v", model.States)
	}
	transitions := model.States[0].Transitions
	if transitions[0].Trigger == nil || transitions[0].Trigger.Kind != TriggerOnCall || transitions[0].Trigger.Value != "approve" {
		t.Fatalf("OnCall trigger = %#v", transitions[0].Trigger)
	}
	if transitions[1].Trigger == nil || transitions[1].Trigger.Kind != TriggerOnSet || transitions[1].Trigger.Value != "count" {
		t.Fatalf("OnSet trigger = %#v", transitions[1].Trigger)
	}
	if transitions[2].Trigger == nil || transitions[2].Trigger.Kind != TriggerAt || transitions[2].Trigger.Expr == nil {
		t.Fatalf("At trigger = %#v", transitions[2].Trigger)
	}
	if transitions[3].Trigger == nil || transitions[3].Trigger.Kind != TriggerEvery || transitions[3].Trigger.Expr == nil {
		t.Fatalf("Every trigger = %#v", transitions[3].Trigger)
	}
	if transitions[4].Trigger == nil || transitions[4].Trigger.Kind != TriggerWhen || transitions[4].Trigger.Expr == nil {
		t.Fatalf("When trigger = %#v", transitions[4].Trigger)
	}
	if len(program.Behaviors) != 4 {
		t.Fatalf("behaviors = %d, want operation plus At, Every, and When triggers", len(program.Behaviors))
	}
}

func TestCompilerConvertsDartExtendedSourceToGo(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "rich.dart", Data: []byte(dartRichSource)}, CompileOptions{
		From:    LanguageDart,
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
		`hsm.Every(Trigger2)`,
		`hsm.When(Trigger3)`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go output missing %q:\n%s", needle, text)
		}
	}
}

func TestCompilerConvertsDartExtendedSourceToDart(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "rich.dart", Data: []byte(dartRichSource)}, CompileOptions{
		From:    LanguageDart,
		To:      LanguageDart,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		`attribute("count", int, 1)`,
		`operation("approve", operation1)`,
		`onCall("approve")`,
		`onSet("count")`,
		`at(trigger1)`,
		`every(trigger2)`,
		`when(trigger3)`,
		"FutureOr<DateTime> trigger1(Context ctx, Instance instance, Event event)",
		"FutureOr<Duration> trigger2(Context ctx, Instance instance, Event event)",
		"bool trigger3(Context ctx, Instance instance, Event event)",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Dart output missing %q:\n%s", needle, text)
		}
	}
	for _, forbidden := range []string{"Attribute \"count\" preserved for manual porting", "Operation \"approve\" -> operation1 preserved for manual porting", "Unsupported at trigger", "Unsupported every trigger", "Unsupported when trigger", "on_call trigger lowered to event trigger", "on(\"approve\")", "on(\"count\")", "_hsmcAtTransition", "_hsmcEveryTransition"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Dart output emitted unsupported runtime trigger %q:\n%s", forbidden, text)
		}
	}
}
