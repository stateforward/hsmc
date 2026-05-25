package hsmc

import (
	"strings"
	"testing"
)

func TestAdapterCodeDeclarationNameDetection(t *testing.T) {
	tests := []struct {
		name string
		code string
		fn   func(string) (string, bool)
		want string
		ok   bool
	}{
		{
			name: "csharp method declaration",
			code: `private static void Entry1(Context ctx, Instance instance, Event @event) {
	System.GC.KeepAlive(event);
}`,
			fn:   csharpMethodDeclarationName,
			want: "Entry1",
			ok:   true,
		},
		{
			name: "csharp attributed method declaration",
			code: `[Obsolete]
private static void Entry1(Context ctx, Instance instance, Event @event) {
	System.GC.KeepAlive(event);
}`,
			fn:   csharpMethodDeclarationName,
			want: "Entry1",
			ok:   true,
		},
		{
			name: "csharp extern method prototype",
			code: `private static extern void Entry1(Context ctx, Instance instance, Event @event);`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageCSharp, code)
			},
			want: "Entry1",
			ok:   true,
		},
		{
			name: "java abstract method prototype",
			code: `private static abstract void Entry1(Context ctx, Instance instance, Event event);`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageJava, code)
			},
			want: "Entry1",
			ok:   true,
		},
		{
			name: "cpp function prototype",
			code: `void entry_1(auto& signal, auto& instance, const auto& event);`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageCPP, code)
			},
			want: "entry_1",
			ok:   true,
		},
		{
			name: "csharp alias using declaration",
			code: `using Entry1 = Sample.Helpers.Entry1;`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageCSharp, code)
			},
			want: "Entry1",
			ok:   true,
		},
		{
			name: "csharp global alias using declaration",
			code: `global using DoorModel = Sample.Helpers.DoorModel;`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageCSharp, code)
			},
			want: "DoorModel",
			ok:   true,
		},
		{
			name: "csharp extern alias declaration",
			code: `extern alias Entry1;`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageCSharp, code)
			},
			want: "Entry1",
			ok:   true,
		},
		{
			name: "csharp ordinary using is not declaration",
			code: `using Sample.Helpers;`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageCSharp, code)
			},
		},
		{
			name: "csharp static using is not declaration",
			code: `using static Sample.Helpers.Entry1;`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageCSharp, code)
			},
		},
		{
			name: "csharp lambda expression is not declaration",
			code: `(ctx, instance, @event) => System.GC.KeepAlive(event)`,
			fn:   csharpMethodDeclarationName,
		},
		{
			name: "csharp nested method is not declaration",
			code: `class Helper {
	private static void Entry1(Context ctx, Instance instance, Event @event) {
		System.GC.KeepAlive(event);
	}
}`,
			fn: csharpMethodDeclarationName,
		},
		{
			name: "csharp delegate declaration",
			code: `public delegate void Entry1(Context ctx, Instance instance, Event @event);`,
			fn:   csharpDelegateDeclarationName,
			want: "Entry1",
			ok:   true,
		},
		{
			name: "csharp attributed delegate declaration",
			code: `[Obsolete]
public delegate void Entry1(Context ctx, Instance instance, Event @event);`,
			fn:   csharpDelegateDeclarationName,
			want: "Entry1",
			ok:   true,
		},
		{
			name: "csharp property declaration",
			code: `private static Model DoorModel { get; } = Hsm.Define("Door");`,
			fn:   csharpPropertyDeclarationName,
			want: "DoorModel",
			ok:   true,
		},
		{
			name: "csharp expression-bodied property declaration",
			code: `private static Model DoorModel => Hsm.Define("Door");`,
			fn:   csharpPropertyDeclarationName,
			want: "DoorModel",
			ok:   true,
		},
		{
			name: "csharp attributed field declaration",
			code: `[Obsolete]
private static readonly Event OpenEvent = null;`,
			fn:   cLikeFieldDeclarationName,
			want: "OpenEvent",
			ok:   true,
		},
		{
			name: "csharp record struct declaration",
			code: `public readonly record struct DoorModel(string Name);`,
			fn:   cLikeTypeDeclarationName,
			want: "DoorModel",
			ok:   true,
		},
		{
			name: "csharp attributed record struct declaration",
			code: `[Obsolete]
public readonly record struct DoorModel(string Name);`,
			fn:   cLikeTypeDeclarationName,
			want: "DoorModel",
			ok:   true,
		},
		{
			name: "java method declaration",
			code: `private static void Entry1(Context ctx, Instance instance, Event event) {
	AdapterFormat.record("entry");
}`,
			fn:   javaMethodDeclarationName,
			want: "Entry1",
			ok:   true,
		},
		{
			name: "java annotated method declaration",
			code: `@Deprecated
private static void Entry1(Context ctx, Instance instance, Event event) {
	AdapterFormat.record("entry");
}`,
			fn:   javaMethodDeclarationName,
			want: "Entry1",
			ok:   true,
		},
		{
			name: "java annotated field declaration",
			code: `@Deprecated
private static final Event OpenEvent = null;`,
			fn:   cLikeFieldDeclarationName,
			want: "OpenEvent",
			ok:   true,
		},
		{
			name: "java array suffix field declaration",
			code: `private static Event Entry1[];`,
			fn:   cLikeFieldDeclarationName,
			want: "Entry1",
			ok:   true,
		},
		{
			name: "java same statement array suffix declaration",
			code: `private static Event Helper, Entry1[];`,
			fn: func(code string) (string, bool) {
				names := targetTopLevelDeclarationNames(LanguageJava, code)
				if len(names) < 2 {
					return "", false
				}
				return names[1], true
			},
			want: "Entry1",
			ok:   true,
		},
		{
			name: "java import declaration",
			code: `import com.example.Entry1;`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageJava, code)
			},
			want: "Entry1",
			ok:   true,
		},
		{
			name: "java static import declaration",
			code: `import static com.example.Helpers.Entry1;`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageJava, code)
			},
			want: "Entry1",
			ok:   true,
		},
		{
			name: "java annotated type declaration",
			code: `@Deprecated
public final class DoorModel {
}`,
			fn:   cLikeTypeDeclarationName,
			want: "DoorModel",
			ok:   true,
		},
		{
			name: "java lambda expression is not declaration",
			code: `(ctx, instance, event) -> AdapterFormat.record("entry")`,
			fn:   javaMethodDeclarationName,
		},
		{
			name: "java annotation declaration",
			code: `public @interface DoorModel {
}`,
			fn:   cLikeTypeDeclarationName,
			want: "DoorModel",
			ok:   true,
		},
		{
			name: "java nested method is not declaration",
			code: `class Helper {
	private static void Entry1(Context ctx, Instance instance, Event event) {
		AdapterFormat.record("entry");
	}
}`,
			fn: javaMethodDeclarationName,
		},
		{
			name: "python def declaration",
			code: `def entry_1(ctx, instance, event):
	instance.ready = True`,
			fn:   pythonFunctionDeclarationName,
			want: "entry_1",
			ok:   true,
		},
		{
			name: "python async def declaration",
			code: `async def entry_1(ctx, instance, event):
	instance.ready = True`,
			fn:   pythonFunctionDeclarationName,
			want: "entry_1",
			ok:   true,
		},
		{
			name: "python class declaration",
			code: `class door_model:
	pass`,
			fn:   pythonTopLevelDeclarationName,
			want: "door_model",
			ok:   true,
		},
		{
			name: "python parenthesized class declaration",
			code: `class door_model(BaseModel):
	pass`,
			fn:   pythonTopLevelDeclarationName,
			want: "door_model",
			ok:   true,
		},
		{
			name: "python assignment declaration",
			code: `door_model = hsm.define_model("Door")`,
			fn:   pythonTopLevelDeclarationName,
			want: "door_model",
			ok:   true,
		},
		{
			name: "python annotated assignment declaration",
			code: `door_model: hsm.Model = hsm.define_model("Door")`,
			fn:   pythonTopLevelDeclarationName,
			want: "door_model",
			ok:   true,
		},
		{
			name: "python chained assignment declaration",
			code: `entry_1 = helper = lambda ctx, instance, event: None`,
			fn:   pythonTopLevelDeclarationName,
			want: "entry_1",
			ok:   true,
		},
		{
			name: "python tuple assignment declaration",
			code: `entry_1, helper = make_behaviors()`,
			fn:   pythonTopLevelDeclarationName,
			want: "entry_1",
			ok:   true,
		},
		{
			name: "python from import alias declaration",
			code: `from helpers import record as entry_1, normalize`,
			fn:   pythonTopLevelDeclarationName,
			want: "entry_1",
			ok:   true,
		},
		{
			name: "python import alias declaration",
			code: `import helpers.record as door_model`,
			fn:   pythonTopLevelDeclarationName,
			want: "door_model",
			ok:   true,
		},
		{
			name: "python comparison is not declaration",
			code: `door_model == hsm.define_model("Door")`,
			fn:   pythonTopLevelDeclarationName,
		},
		{
			name: "python lambda expression is not declaration",
			code: `lambda ctx, instance, event: setattr(instance, "ready", True)`,
			fn:   pythonFunctionDeclarationName,
		},
		{
			name: "go func declaration",
			code: `func Entry1(ctx context.Context, instance hsm.Instance, event hsm.Event) {
	instance.Set("ready", true)
}`,
			fn:   goTopLevelDeclarationName,
			want: "Entry1",
			ok:   true,
		},
		{
			name: "go var func declaration",
			code: `var Entry1 = func(ctx context.Context, instance hsm.Instance, event hsm.Event) {
	instance.Set("ready", true)
}`,
			fn:   goTopLevelDeclarationName,
			want: "Entry1",
			ok:   true,
		},
		{
			name: "go type declaration",
			code: `type Entry1 struct {
	Ready bool
}`,
			fn:   goTopLevelDeclarationName,
			want: "Entry1",
			ok:   true,
		},
		{
			name: "go aliased import declaration",
			code: `import Entry1 "example.com/helpers"`,
			fn:   goTopLevelDeclarationName,
			want: "Entry1",
			ok:   true,
		},
		{
			name: "go anonymous func is not declaration",
			code: `func(ctx context.Context, instance hsm.Instance, event hsm.Event) {
	instance.Set("ready", true)
}`,
			fn: goTopLevelDeclarationName,
		},
		{
			name: "cpp auto declaration",
			code: `static constexpr auto entry_1 = [](auto& signal, auto& instance, const auto& event) -> void { adapter_format("entry"); };`,
			fn:   cppTopLevelDeclarationName,
			want: "entry_1",
			ok:   true,
		},
		{
			name: "cpp typed variable declaration",
			code: `inline constexpr int entry_1 = 0;`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageCPP, code)
			},
			want: "entry_1",
			ok:   true,
		},
		{
			name: "cpp pointer variable declaration",
			code: `hsm::Model *door_model = nullptr;`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageCPP, code)
			},
			want: "door_model",
			ok:   true,
		},
		{
			name: "cpp function pointer declaration",
			code: `void (*entry_1)(hsm::Context&, HsmcInstance&, const hsm::Event&) = nullptr;`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageCPP, code)
			},
			want: "entry_1",
			ok:   true,
		},
		{
			name: "cpp function reference declaration",
			code: `void (&entry_1)(hsm::Context&, HsmcInstance&, const hsm::Event&) = default_entry;`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageCPP, code)
			},
			want: "entry_1",
			ok:   true,
		},
		{
			name: "cpp same statement pointer declarations",
			code: `hsm::Model *helper = nullptr, *door_model = nullptr;`,
			fn: func(code string) (string, bool) {
				names := targetTopLevelDeclarationNames(LanguageCPP, code)
				if len(names) < 2 {
					return "", false
				}
				return names[1], true
			},
			want: "door_model",
			ok:   true,
		},
		{
			name: "cpp lambda expression is not declaration",
			code: `[](auto& signal, auto& instance, const auto& event) -> void { adapter_format("entry"); }`,
			fn:   cppTopLevelDeclarationName,
		},
		{
			name: "cpp nested auto member is not declaration",
			code: `struct Helper {
	static constexpr auto entry_1 = [](auto& signal, auto& instance, const auto& event) -> void { adapter_format("entry"); };
};`,
			fn: cppTopLevelDeclarationName,
		},
		{
			name: "cpp enum class declaration",
			code: `enum class OpenEvent {
	Open,
};`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageCPP, code)
			},
			want: "OpenEvent",
			ok:   true,
		},
		{
			name: "cpp union declaration",
			code: `union OpenEvent {
	int value;
};`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageCPP, code)
			},
			want: "OpenEvent",
			ok:   true,
		},
		{
			name: "cpp namespace declaration",
			code: `namespace entry_1 {
	void helper();
}`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageCPP, code)
			},
			want: "entry_1",
			ok:   true,
		},
		{
			name: "cpp using declaration",
			code: `using door_model = hsm::Model;`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageCPP, code)
			},
			want: "door_model",
			ok:   true,
		},
		{
			name: "cpp typedef declaration",
			code: `typedef hsm::Model door_model;`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageCPP, code)
			},
			want: "door_model",
			ok:   true,
		},
		{
			name: "javascript function declaration",
			code: `function entry1(ctx, instance, event) {
	instance.ready = true;
}`,
			fn:   esTopLevelDeclarationName,
			want: "entry1",
			ok:   true,
		},
		{
			name: "javascript async function declaration",
			code: `async function entry1(ctx, instance, event) {
	instance.ready = true;
}`,
			fn:   esTopLevelDeclarationName,
			want: "entry1",
			ok:   true,
		},
		{
			name: "typescript const declaration",
			code: `const entry1: hsm.Operation<InstanceType<typeof hsm.Instance>> = (ctx, instance, event) => {
	instance.ready = true;
};`,
			fn:   esTopLevelDeclarationName,
			want: "entry1",
			ok:   true,
		},
		{
			name: "typescript named import alias declaration",
			code: `import { record as entry1, normalize } from "./helpers";`,
			fn:   esTopLevelDeclarationName,
			want: "entry1",
			ok:   true,
		},
		{
			name: "javascript namespace import declaration",
			code: `import * as DoorModel from "./model";`,
			fn:   esTopLevelDeclarationName,
			want: "DoorModel",
			ok:   true,
		},
		{
			name: "javascript object destructuring declaration",
			code: `const { entry1 } = helpers;`,
			fn:   esTopLevelDeclarationName,
			want: "entry1",
			ok:   true,
		},
		{
			name: "javascript object destructuring alias declaration",
			code: `const { format: entry1 } = helpers;`,
			fn:   esTopLevelDeclarationName,
			want: "entry1",
			ok:   true,
		},
		{
			name: "typescript array destructuring declaration",
			code: `const [entry1] = helpers;`,
			fn:   esTopLevelDeclarationName,
			want: "entry1",
			ok:   true,
		},
		{
			name: "javascript class declaration",
			code: `class DoorModel {
}`,
			fn:   esTopLevelDeclarationName,
			want: "DoorModel",
			ok:   true,
		},
		{
			name: "typescript exported enum declaration",
			code: `export enum DoorModel {
	Closed,
}`,
			fn:   esTopLevelDeclarationName,
			want: "DoorModel",
			ok:   true,
		},
		{
			name: "typescript interface declaration",
			code: `export interface DoorModel {
	ready: boolean;
}`,
			fn:   esTopLevelDeclarationName,
			want: "DoorModel",
			ok:   true,
		},
		{
			name: "typescript declare const declaration",
			code: `declare const openEvent: hsm.Event;`,
			fn:   esTopLevelDeclarationName,
			want: "openEvent",
			ok:   true,
		},
		{
			name: "typescript exported declare class declaration",
			code: `export declare class DoorModel {
	ready: boolean;
}`,
			fn:   esTopLevelDeclarationName,
			want: "DoorModel",
			ok:   true,
		},
		{
			name: "typescript abstract class declaration",
			code: `abstract class DoorModel {
	ready: boolean;
}`,
			fn:   esTopLevelDeclarationName,
			want: "DoorModel",
			ok:   true,
		},
		{
			name: "typescript namespace declaration",
			code: `export declare namespace DoorModel {
	const ready: boolean;
}`,
			fn:   esTopLevelDeclarationName,
			want: "DoorModel",
			ok:   true,
		},
		{
			name: "typescript type declaration",
			code: `type DoorModel = {
	ready: boolean;
};`,
			fn:   esTopLevelDeclarationName,
			want: "DoorModel",
			ok:   true,
		},
		{
			name: "javascript default exported class declaration",
			code: `export default class DoorModel {
}`,
			fn:   esTopLevelDeclarationName,
			want: "DoorModel",
			ok:   true,
		},
		{
			name: "javascript arrow expression is not declaration",
			code: `(ctx, instance, event) => {
	instance.ready = true;
}`,
			fn: esTopLevelDeclarationName,
		},
		{
			name: "java package private constructor declaration",
			code: `GeneratedHsm() {
}`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageJava, code)
			},
			want: "GeneratedHsm",
			ok:   true,
		},
		{
			name: "java annotated constructor declaration",
			code: `@SuppressWarnings("unused")
GeneratedHsm() {
}`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageJava, code)
			},
			want: "GeneratedHsm",
			ok:   true,
		},
		{
			name: "csharp static constructor declaration",
			code: `static GeneratedHsm() {
}`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageCSharp, code)
			},
			want: "GeneratedHsm",
			ok:   true,
		},
		{
			name: "csharp attributed static constructor declaration",
			code: `[Obsolete]
static GeneratedHsm() {
}`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageCSharp, code)
			},
			want: "GeneratedHsm",
			ok:   true,
		},
		{
			name: "rust fn declaration",
			code: `fn entry_1(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) {
	let _ = (ctx, instance, event);
}`,
			fn:   rustFunctionDeclarationName,
			want: "entry_1",
			ok:   true,
		},
		{
			name: "rust static mut declaration",
			code: `pub(crate) static mut entry_1: Option<fn(&hsm::Context, &mut HsmcInstance, &hsm::Event)> = None;`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageRust, code)
			},
			want: "entry_1",
			ok:   true,
		},
		{
			name: "rust use alias declaration",
			code: `pub use crate::helpers::record as entry_1;`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageRust, code)
			},
			want: "entry_1",
			ok:   true,
		},
		{
			name: "rust trait declaration",
			code: `pub trait HsmcInstance {
	fn ready(&self) -> bool;
}`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageRust, code)
			},
			want: "HsmcInstance",
			ok:   true,
		},
		{
			name: "rust union declaration",
			code: `pub union HsmcInstance {
	ready: bool,
}`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageRust, code)
			},
			want: "HsmcInstance",
			ok:   true,
		},
		{
			name: "rust mod declaration",
			code: `pub mod entry_1 {
	pub fn helper() {}
}`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageRust, code)
			},
			want: "entry_1",
			ok:   true,
		},
		{
			name: "rust closure expression is not declaration",
			code: `|ctx, instance, event| {
	let _ = (ctx, instance, event);
}`,
			fn: rustFunctionDeclarationName,
		},
		{
			name: "zig fn declaration",
			code: `fn entry_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void {
	_ = ctx;
	_ = inst;
	_ = event;
}`,
			fn:   zigFunctionDeclarationName,
			want: "entry_1",
			ok:   true,
		},
		{
			name: "zig extern var declaration",
			code: `pub extern var entry_1: ?*const fn (*hsm.Context, *hsm.Instance, hsm.Event) void;`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageZig, code)
			},
			want: "entry_1",
			ok:   true,
		},
		{
			name: "zig threadlocal var declaration",
			code: `pub threadlocal var entry_1: ?*const fn (*hsm.Context, *hsm.Instance, hsm.Event) void = null;`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageZig, code)
			},
			want: "entry_1",
			ok:   true,
		},
		{
			name: "zig anonymous struct callback is not declaration",
			code: `struct {
	fn call(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void {
		_ = ctx;
		_ = inst;
		_ = event;
	}
}.call`,
			fn: zigFunctionDeclarationName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tt.fn(tt.code)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("name = %q, ok = %v; want name = %q, ok = %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestDartAdapterCodeDeclarationNameDetection(t *testing.T) {
	tests := []struct {
		name string
		code string
		fn   func(string) (string, bool)
		want string
		ok   bool
	}{
		{
			name: "function declaration",
			code: `FutureOr<void> entry1(Context ctx, Instance instance, Event event) {
	adapterFormat("entry");
}`,
			fn:   dartFunctionDeclarationName,
			want: "entry1",
			ok:   true,
		},
		{
			name: "external function declaration",
			code: `external FutureOr<void> entry1(Context ctx, Instance instance, Event event);`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageDart, code)
			},
			want: "entry1",
			ok:   true,
		},
		{
			name: "prefixed import declaration",
			code: `import 'helpers.dart' as entry1;`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageDart, code)
			},
			want: "entry1",
			ok:   true,
		},
		{
			name: "show import declaration",
			code: `import 'helpers.dart' show entry1, DoorModel;`,
			fn: func(code string) (string, bool) {
				return targetTopLevelDeclarationName(LanguageDart, code)
			},
			want: "entry1",
			ok:   true,
		},
		{
			name: "typed variable declaration",
			code: `final FutureOr<void> Function(Context, Instance, Event) entry1 = (ctx, instance, event) {
	adapterFormat("entry");
};`,
			fn:   dartTopLevelVariableName,
			want: "entry1",
			ok:   true,
		},
		{
			name: "typed variable declaration without storage keyword",
			code: `FutureOr<void> Function(Context, Instance, Event) entry1 = (ctx, instance, event) {
	adapterFormat("entry");
};`,
			fn:   dartTopLevelVariableName,
			want: "entry1",
			ok:   true,
		},
		{
			name: "typed model assignment without storage keyword",
			code: `Model doorModel = define("Door", []);`,
			fn:   dartTopLevelVariableName,
			want: "doorModel",
			ok:   true,
		},
		{
			name: "untyped const variable declaration",
			code: `const entry1 = _entryAdapter;`,
			fn:   dartTopLevelVariableName,
			want: "entry1",
			ok:   true,
		},
		{
			name: "class declaration",
			code: `class entry1 {
}`,
			fn:   dartTypeDeclarationName,
			want: "entry1",
			ok:   true,
		},
		{
			name: "abstract class declaration",
			code: `abstract class doorModel {
}`,
			fn:   dartTypeDeclarationName,
			want: "doorModel",
			ok:   true,
		},
		{
			name: "typedef declaration",
			code: `typedef entry1 = void Function(Context ctx, Instance instance, Event event);`,
			fn:   dartTypeDeclarationName,
			want: "entry1",
			ok:   true,
		},
		{
			name: "extension type declaration",
			code: `extension type entry1(int value) {
	int call() => value;
}`,
			fn:   dartTypeDeclarationName,
			want: "entry1",
			ok:   true,
		},
		{
			name: "anonymous function expression is not declaration",
			code: `(ctx, instance, event) {
	adapterFormat("entry");
}`,
			fn: dartFunctionDeclarationName,
		},
		{
			name: "class method is not top-level function declaration",
			code: `class Helper {
	static FutureOr<void> entry1(Context ctx, Instance instance, Event event) {
		adapterFormat("entry");
	}
}`,
			fn: dartFunctionDeclarationName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tt.fn(tt.code)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("name = %q, ok = %v; want name = %q, ok = %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestDartFunctionDeclarationShapeDetection(t *testing.T) {
	tests := []struct {
		name       string
		code       string
		returnType string
		wantName   string
		parameters string
		ok         bool
	}{
		{
			name: "function declaration",
			code: `FutureOr<void> entry1(Context ctx, Instance instance, Event event) {
	adapterFormat("entry");
}`,
			returnType: "FutureOr<void>",
			wantName:   "entry1",
			parameters: "(Context ctx, Instance instance, Event event)",
			ok:         true,
		},
		{
			name:       "arrow function declaration",
			code:       `FutureOr<void> entry1(Context ctx, Instance instance, Event event) => adapterFormat("entry");`,
			returnType: "FutureOr<void>",
			wantName:   "entry1",
			parameters: "(Context ctx, Instance instance, Event event)",
			ok:         true,
		},
		{
			name:       "async function declaration",
			code:       `FutureOr<void> entry1(Context ctx, Instance instance, Event event) async {}`,
			returnType: "FutureOr<void>",
			wantName:   "entry1",
			parameters: "(Context ctx, Instance instance, Event event)",
			ok:         true,
		},
		{
			name: "anonymous function expression is not declaration",
			code: `(Context ctx, Instance instance, Event event) {
	adapterFormat("entry");
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			returnType, name, parameters, ok := dartFunctionDeclarationShape(tt.code)
			if returnType != tt.returnType || name != tt.wantName || parameters != tt.parameters || ok != tt.ok {
				t.Fatalf("shape = %q, %q, %q, %v; want %q, %q, %q, %v", returnType, name, parameters, ok, tt.returnType, tt.wantName, tt.parameters, tt.ok)
			}
		})
	}
}

func TestESTopLevelDeclarationNamesDetectDestructuringBindings(t *testing.T) {
	code := `const { entry1, model: DoorModel } = helpers, [openEvent] = events;`
	got := targetTopLevelDeclarationNames(LanguageTS, code)
	want := []string{"entry1", "DoorModel", "openEvent"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("declaration names = %#v, want %#v", got, want)
	}
}

func TestESTopLevelDeclarationNamesDetectImportBindings(t *testing.T) {
	code := `import entry1, { DoorModel as Model, type OpenEvent } from "./helpers";
import * as helperNamespace from "./helpers";`
	got := targetTopLevelDeclarationNames(LanguageTS, code)
	want := []string{"entry1", "Model", "OpenEvent", "helperNamespace"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("declaration names = %#v, want %#v", got, want)
	}
}

func TestRustTopLevelDeclarationNamesDetectUseBindings(t *testing.T) {
	code := `use crate::helpers::{helper, record as entry_1};
pub use crate::model::DoorModel;
use crate::nested::{events::{open_event}};`
	got := targetTopLevelDeclarationNames(LanguageRust, code)
	want := []string{"helper", "entry_1", "DoorModel", "open_event"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("declaration names = %#v, want %#v", got, want)
	}
}

func TestPythonTopLevelDeclarationNamesDetectImportBindings(t *testing.T) {
	code := `from helpers import record as entry_1, DoorModel
import generated.open_event as open_event`
	got := targetTopLevelDeclarationNames(LanguagePython, code)
	want := []string{"entry_1", "DoorModel", "open_event"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("declaration names = %#v, want %#v", got, want)
	}
}

func TestPythonTopLevelDeclarationNamesDetectParenthesizedFromImportBindings(t *testing.T) {
	code := `from helpers import (
	record as entry_1,
	DoorModel,
)
import generated.open_event as open_event`
	got := targetTopLevelDeclarationNames(LanguagePython, code)
	want := []string{"entry_1", "DoorModel", "open_event"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("declaration names = %#v, want %#v", got, want)
	}
}

func TestDartTopLevelDeclarationNamesDetectImportBindings(t *testing.T) {
	code := `import 'helpers.dart' show entry1, DoorModel;
import 'events.dart' as openEvent;`
	got := targetTopLevelDeclarationNames(LanguageDart, code)
	want := []string{"entry1", "DoorModel", "openEvent"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("declaration names = %#v, want %#v", got, want)
	}
}

func TestGoTopLevelDeclarationNamesDetectImportBindings(t *testing.T) {
	code := `import (
	Entry1 "example.com/helpers"
	"example.com/DoorModel"
)`
	got := targetTopLevelDeclarationNames(LanguageGo, code)
	want := []string{"Entry1", "DoorModel"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("declaration names = %#v, want %#v", got, want)
	}
}

func TestGoGroupedDeclarationNamesDetectSameLineBindings(t *testing.T) {
	code := `var (
	Helper, Entry1 func(ctx context.Context, instance hsm.Instance, event hsm.Event) = func(ctx context.Context, instance hsm.Instance, event hsm.Event) {}, func(ctx context.Context, instance hsm.Instance, event hsm.Event) {}
)`
	got := targetTopLevelDeclarationNames(LanguageGo, code)
	want := []string{"Helper", "Entry1"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("declaration names = %#v, want %#v", got, want)
	}
}

func TestGoDeclarationNamesDetectUntypedSameLineBindings(t *testing.T) {
	code := `var Helper, Entry1 = func(value string) string { return value }, func(ctx context.Context, instance hsm.Instance, event hsm.Event) {}`
	got := targetTopLevelDeclarationNames(LanguageGo, code)
	want := []string{"Helper", "Entry1"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("declaration names = %#v, want %#v", got, want)
	}
}

func TestGoDeclarationNamesDetectSameLineBindingsWithoutInitializer(t *testing.T) {
	code := `var Helper, Entry1 func(ctx context.Context, instance hsm.Instance, event hsm.Event)`
	got := targetTopLevelDeclarationNames(LanguageGo, code)
	want := []string{"Helper", "Entry1"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("declaration names = %#v, want %#v", got, want)
	}
}

func TestGoFunctionDeclarationSignatureDetection(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
		ok   bool
	}{
		{
			name: "canonical signature",
			code: `func Entry1(ctx context.Context, instance hsm.Instance, event hsm.Event) {
	instance.Set("ready", true)
}`,
			want: "(ctx context.Context, instance hsm.Instance, event hsm.Event)",
			ok:   true,
		},
		{
			name: "wrong arity still reports signature",
			code: `func Entry1() {
}`,
			want: "()",
			ok:   true,
		},
		{
			name: "var function is not function declaration signature",
			code: `var Entry1 = func(ctx context.Context, instance hsm.Instance, event hsm.Event) {
}`,
		},
		{
			name: "wrong name is not matched",
			code: `func RenamedEntry(ctx context.Context, instance hsm.Instance, event hsm.Event) {
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := goFunctionDeclarationSignature(tt.code, "Entry1")
			if got != tt.want || ok != tt.ok {
				t.Fatalf("signature = %q, ok = %v; want %q, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestGoFunctionLiteralSignatureDetection(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
		ok   bool
	}{
		{
			name: "literal",
			code: `func(ctx context.Context, instance hsm.Instance, event hsm.Event) {
	instance.Set("ready", true)
}`,
			want: "(ctx context.Context, instance hsm.Instance, event hsm.Event)",
			ok:   true,
		},
		{
			name: "literal with return type",
			code: `func(ctx context.Context, instance hsm.Instance, event hsm.Event) bool {
	return true
}`,
			want: "(ctx context.Context, instance hsm.Instance, event hsm.Event) bool",
			ok:   true,
		},
		{
			name: "var literal",
			code: `var Entry1 = func(ctx context.Context, instance hsm.Instance, event hsm.Event) {
}`,
			want: "(ctx context.Context, instance hsm.Instance, event hsm.Event)",
			ok:   true,
		},
		{
			name: "declaration is not literal",
			code: `func Entry1(ctx context.Context, instance hsm.Instance, event hsm.Event) {
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := goFunctionLiteralSignature(tt.code)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("signature = %q, ok = %v; want %q, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestGoBehaviorSignatureSuffix(t *testing.T) {
	tests := []struct {
		name      string
		signature string
		want      string
	}{
		{
			name:      "function declaration",
			signature: "func approve(ctx context.Context, instance hsm.Instance, event hsm.Event) bool",
			want:      "(ctx context.Context, instance hsm.Instance, event hsm.Event) bool",
		},
		{
			name:      "function literal",
			signature: "func(ctx context.Context, instance hsm.Instance, event hsm.Event)",
			want:      "(ctx context.Context, instance hsm.Instance, event hsm.Event)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := goBehaviorSignatureSuffix(tt.signature); got != tt.want {
				t.Fatalf("goBehaviorSignatureSuffix(%q) = %q, want %q", tt.signature, got, tt.want)
			}
		})
	}
}

func TestAdapterBehaviorBodyBoundaryRejectsScopeEscape(t *testing.T) {
	body := `translated();
}
const DoorModel = null;`
	for _, target := range []Language{LanguageCSharp, LanguageCPP, LanguageDart, LanguageGo, LanguageJava, LanguageJS, LanguageRust, LanguageTS, LanguageZig} {
		t.Run(string(target), func(t *testing.T) {
			err := validateAdapterBehaviorBodyBoundary(target, "entry_1", body)
			if err == nil || !strings.Contains(err.Error(), `body attempts to close the compiler-owned behavior scope`) {
				t.Fatalf("error = %v, want body scope escape rejection", err)
			}
		})
	}
}

func TestAdapterBehaviorBodyBoundaryAllowsNestedStatementsAndComments(t *testing.T) {
	body := `if (ready) {
	translated("} ignored in string");
	// } ignored in comment
}`
	if err := validateAdapterBehaviorBodyBoundary(LanguageTS, "entry_1", body); err != nil {
		t.Fatalf("validateAdapterBehaviorBodyBoundary() = %v, want nil", err)
	}
}

func TestCLikeMethodDeclarationShapeDetection(t *testing.T) {
	tests := []struct {
		name       string
		code       string
		returnType string
		methodName string
		parameters string
		ok         bool
	}{
		{
			name: "csharp method",
			code: `private static void Entry1(Context ctx, Instance instance, Event @event)
{
    System.GC.KeepAlive("entry");
}`,
			returnType: "void",
			methodName: "Entry1",
			parameters: "(Context ctx, Instance instance, Event @event)",
			ok:         true,
		},
		{
			name: "java method",
			code: `private static boolean Guard1(Context ctx, Instance instance, Event event) {
    return true;
}`,
			returnType: "boolean",
			methodName: "Guard1",
			parameters: "(Context ctx, Instance instance, Event event)",
			ok:         true,
		},
		{
			name: "nested class is not top-level method",
			code: `class Helper {
    private static void Entry1(Context ctx, Instance instance, Event event) {
    }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			returnType, methodName, parameters, ok := cLikeMethodDeclarationShape(tt.code)
			if returnType != tt.returnType || methodName != tt.methodName || parameters != tt.parameters || ok != tt.ok {
				t.Fatalf("shape = %q, %q, %q, %v; want %q, %q, %q, %v", returnType, methodName, parameters, ok, tt.returnType, tt.methodName, tt.parameters, tt.ok)
			}
		})
	}
}

func TestCPPLambdaShapeDetection(t *testing.T) {
	tests := []struct {
		name       string
		code       string
		parameters string
		returnType string
		ok         bool
	}{
		{
			name:       "declared lambda",
			code:       `static constexpr auto entry_1 = [](auto& signal, auto& instance, const auto& event) -> void { adapter_format("entry"); };`,
			parameters: "(auto& signal, auto& instance, const auto& event)",
			returnType: "void",
			ok:         true,
		},
		{
			name:       "expression lambda",
			code:       `[](auto& signal, auto& instance, const auto& event) { adapter_format("entry"); }`,
			parameters: "(auto& signal, auto& instance, const auto& event)",
			ok:         true,
		},
		{
			name: "not lambda",
			code: `static constexpr auto entry_1 = adapter_format;`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parameters, returnType, ok := cppLambdaShape(tt.code)
			if parameters != tt.parameters || returnType != tt.returnType || ok != tt.ok {
				t.Fatalf("shape = %q, %q, %v; want %q, %q, %v", parameters, returnType, ok, tt.parameters, tt.returnType, tt.ok)
			}
		})
	}
}

func TestKeywordFunctionDeclarationShapeDetection(t *testing.T) {
	tests := []struct {
		name       string
		code       string
		fn         func(string) (string, string, string, bool)
		wantName   string
		parameters string
		returnType string
		ok         bool
	}{
		{
			name: "rust function",
			code: `fn entry_1(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) -> Pin<Box<dyn Future<Output = ()> + Send>> {
	Box::pin(async {})
}`,
			fn:         rustFunctionDeclarationShape,
			wantName:   "entry_1",
			parameters: "(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event)",
			returnType: "Pin<Box<dyn Future<Output = ()> + Send>>",
			ok:         true,
		},
		{
			name: "rust unit return default",
			code: `pub fn entry_1(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) {
}`,
			fn:         rustFunctionDeclarationShape,
			wantName:   "entry_1",
			parameters: "(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event)",
			returnType: "()",
			ok:         true,
		},
		{
			name: "zig function",
			code: `fn entry_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void {
}`,
			fn:         zigFunctionDeclarationShape,
			wantName:   "entry_1",
			parameters: "(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event)",
			returnType: "void",
			ok:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, parameters, returnType, ok := tt.fn(tt.code)
			if name != tt.wantName || parameters != tt.parameters || returnType != tt.returnType || ok != tt.ok {
				t.Fatalf("shape = %q, %q, %q, %v; want %q, %q, %q, %v", name, parameters, returnType, ok, tt.wantName, tt.parameters, tt.returnType, tt.ok)
			}
		})
	}
}

func TestESFunctionDeclarationShapeDetection(t *testing.T) {
	tests := []struct {
		name       string
		code       string
		wantName   string
		parameters string
		returnType string
		ok         bool
	}{
		{
			name: "typescript function",
			code: `function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {
	instance.ready = true;
}`,
			wantName:   "entry1",
			parameters: "(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event)",
			returnType: "void",
			ok:         true,
		},
		{
			name: "javascript async export function",
			code: `export async function entry1(ctx, instance, event) {
	instance.ready = true;
}`,
			wantName:   "entry1",
			parameters: "(ctx, instance, event)",
			ok:         true,
		},
		{
			name: "arrow expression is not declaration",
			code: `const entry1 = (ctx, instance, event) => {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, parameters, returnType, ok := esFunctionDeclarationShape(tt.code)
			if name != tt.wantName || parameters != tt.parameters || returnType != tt.returnType || ok != tt.ok {
				t.Fatalf("shape = %q, %q, %q, %v; want %q, %q, %q, %v", name, parameters, returnType, ok, tt.wantName, tt.parameters, tt.returnType, tt.ok)
			}
		})
	}
}

func TestESFunctionPrototypeShapeDetection(t *testing.T) {
	tests := []struct {
		name       string
		code       string
		wantName   string
		parameters string
		returnType string
		ok         bool
	}{
		{
			name:       "typescript declare function",
			code:       `declare function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void;`,
			wantName:   "entry1",
			parameters: "(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event)",
			returnType: "void",
			ok:         true,
		},
		{
			name:       "exported javascript prototype",
			code:       `export function entry1(ctx, instance, event);`,
			wantName:   "entry1",
			parameters: "(ctx, instance, event)",
			ok:         true,
		},
		{
			name: "function body is not prototype",
			code: `function entry1(ctx, instance, event) {
	instance.ready = true;
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, parameters, returnType, ok := esFunctionPrototypeShape(tt.code)
			if name != tt.wantName || parameters != tt.parameters || returnType != tt.returnType || ok != tt.ok {
				t.Fatalf("shape = %q, %q, %q, %v; want %q, %q, %q, %v", name, parameters, returnType, ok, tt.wantName, tt.parameters, tt.returnType, tt.ok)
			}
		})
	}
}

func TestPythonFunctionDeclarationShapeDetection(t *testing.T) {
	tests := []struct {
		name       string
		code       string
		async      bool
		wantName   string
		parameters string
		returnType string
		ok         bool
	}{
		{
			name: "async annotated function",
			code: `async def entry_1(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> None:
	instance.ready = True`,
			async:      true,
			wantName:   "entry_1",
			parameters: "(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event)",
			returnType: "None",
			ok:         true,
		},
		{
			name: "sync unannotated function defaults return type",
			code: `def entry_1(ctx, instance, event):
	pass`,
			wantName:   "entry_1",
			parameters: "(ctx, instance, event)",
			returnType: "None",
			ok:         true,
		},
		{
			name: "lambda is not declaration",
			code: `lambda ctx, instance, event: None`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			async, name, parameters, returnType, ok := pythonFunctionDeclarationShape(tt.code)
			if async != tt.async || name != tt.wantName || parameters != tt.parameters || returnType != tt.returnType || ok != tt.ok {
				t.Fatalf("shape = %v, %q, %q, %q, %v; want %v, %q, %q, %q, %v", async, name, parameters, returnType, ok, tt.async, tt.wantName, tt.parameters, tt.returnType, tt.ok)
			}
		})
	}
}
