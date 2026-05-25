package hsmc

import (
	"context"
	"strings"
	"testing"
)

const tsQueueConfigSource = `import * as hsm from "@stateforward/hsm";

const priorityQueue = hsm.Queue({
	Push: (ctx, event) => event,
	Pop: (ctx) => [undefined, false],
	Len: (ctx) => 0,
});

export const runtimeConfig = hsm.Config({
	ID: "door-1",
	Queue: priorityQueue,
});

export const DoorModel = hsm.Define(
	"Door",
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed"),
);
`

const tsClockConfigSource = `import * as hsm from "@stateforward/hsm";

const deterministicClock = hsm.Clock({
	After: (duration) => Promise.resolve(duration),
	NewTimer: (duration) => ({ cancel: () => undefined }),
});

export const runtimeConfig = hsm.Config({
	ID: "door-1",
	Clock: deterministicClock,
});

export const DoorModel = hsm.Define(
	"Door",
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed"),
);
`

const tsRenamedImportClockConfigSource = `import {
	Clock as HClock,
	Config as HConfig,
	Define as HDefine,
	Initial as HInitial,
	Target as HTarget,
	State as HState,
} from "@stateforward/hsm";

export const deterministicClock = HClock({
	After: (duration) => Promise.resolve(duration),
	NewTimer: (duration) => ({ cancel: () => undefined }),
}), runtimeConfig = HConfig({
	ID: "door-1",
	Clock: deterministicClock,
}), DoorModel = HDefine(
	"Door",
	HInitial(HTarget("closed")),
	HState("closed"),
);
`

const tsDefaultClockConfigSource = `import * as hsm from "@stateforward/hsm";

hsm.DefaultClock = hsm.Clock({
	After: (duration) => Promise.resolve(duration),
	NewTimer: (duration) => ({ cancel: () => undefined }),
});

export const DoorModel = hsm.Define(
	"Door",
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed"),
);
`

const tsDefaultClockAliasConfigSource = `import * as runtime from "@stateforward/hsm";

runtime.DefaultClock = runtime.Clock({
	After: (duration) => Promise.resolve(duration),
	NewTimer: (duration) => ({ cancel: () => undefined }),
});

export const DoorModel = runtime.Define(
	"Door",
	runtime.Initial(runtime.Target("closed")),
	runtime.State("closed"),
);
`

const tsDefaultClockDefaultImportConfigSource = `import runtime from "@stateforward/hsm";

runtime.DefaultClock = runtime.Clock({
	After: (duration) => Promise.resolve(duration),
	NewTimer: (duration) => ({ cancel: () => undefined }),
});

export const DoorModel = runtime.Define(
	"Door",
	runtime.Initial(runtime.Target("closed")),
	runtime.State("closed"),
);
`

const tsDefaultClockCamelCaseAliasConfigSource = `import * as hsm from "@stateforward/hsm";

hsm.defaultClock = hsm.clock({
	after: (duration) => Promise.resolve(duration),
	newTimer: (duration) => ({ cancel: () => undefined }),
});

export const DoorModel = hsm.define(
	"Door",
	hsm.initial(hsm.target("closed")),
	hsm.state("closed"),
);
`

const tsDefaultClockNamedCamelCaseImportConfigSource = `import { defaultClock, clock, define, initial, target, state } from "@stateforward/hsm";

defaultClock = clock({
	after: (duration) => Promise.resolve(duration),
	newTimer: (duration) => ({ cancel: () => undefined }),
});

export const DoorModel = define(
	"Door",
	initial(target("closed")),
	state("closed"),
);
`

const tsGroupedQueueConfigSource = `import * as hsm from "@stateforward/hsm";

export const priorityQueue = hsm.Queue({
	Push: (ctx, event) => event,
	Pop: (ctx) => [undefined, false],
	Len: (ctx) => 0,
}), runtimeConfig = hsm.Config({
	ID: "door-1",
	Queue: priorityQueue,
}), DoorModel = hsm.Define(
	"Door",
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed"),
);
`

const tsCamelCaseQueueConfigSource = `import * as hsm from "@stateforward/hsm";

export const priorityQueue = hsm.queue({
	push: (ctx, event) => event,
	pop: (ctx) => [undefined, false],
	len: (ctx) => 0,
}), runtimeConfig = hsm.config({
	id: "door-1",
	queue: priorityQueue,
}), DoorModel = hsm.define(
	"Door",
	hsm.initial(hsm.target("closed")),
	hsm.state("closed"),
);
`

const tsNamedImportQueueConfigSource = `import { Queue, Config, Define, Initial, Target, State } from "@stateforward/hsm";

export const priorityQueue = Queue({
	Push: (ctx, event) => event,
	Pop: (ctx) => [undefined, false],
	Len: (ctx) => 0,
}), runtimeConfig = Config({
	ID: "door-1",
	Queue: priorityQueue,
}), DoorModel = Define(
	"Door",
	Initial(Target("closed")),
	State("closed"),
);
`

const tsRenamedImportQueueConfigSource = `import {
	Queue as HQueue,
	Config as HConfig,
	Define as HDefine,
	Initial as HInitial,
	Target as HTarget,
	State as HState,
} from "@stateforward/hsm";

export const priorityQueue = HQueue({
	Push: (ctx, event) => event,
	Pop: (ctx) => [undefined, false],
	Len: (ctx) => 0,
}), runtimeConfig = HConfig({
	ID: "door-1",
	Queue: priorityQueue,
}), DoorModel = HDefine(
	"Door",
	HInitial(HTarget("closed")),
	HState("closed"),
);
`

const jsGroupedQueueConfigSource = `import * as hsm from "@stateforward/hsm";

export const priorityQueue = hsm.Queue({
	Push: (ctx, event) => event,
	Pop: (ctx) => [undefined, false],
	Len: (ctx) => 0,
}), runtimeConfig = hsm.Config({
	ID: "door-1",
	Queue: priorityQueue,
}), DoorModel = hsm.Define(
	"Door",
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed"),
);
`

const jsCamelCaseQueueConfigSource = `import * as hsm from "@stateforward/hsm";

export const priorityQueue = hsm.queue({
	push: (ctx, event) => event,
	pop: (ctx) => [undefined, false],
	len: (ctx) => 0,
}), runtimeConfig = hsm.config({
	id: "door-1",
	queue: priorityQueue,
}), DoorModel = hsm.define(
	"Door",
	hsm.initial(hsm.target("closed")),
	hsm.state("closed"),
);
`

const jsNamedImportQueueConfigSource = `import { Queue, Config, Define, Initial, Target, State } from "@stateforward/hsm";

export const priorityQueue = Queue({
	Push: (ctx, event) => event,
	Pop: (ctx) => [undefined, false],
	Len: (ctx) => 0,
}), runtimeConfig = Config({
	ID: "door-1",
	Queue: priorityQueue,
}), DoorModel = Define(
	"Door",
	Initial(Target("closed")),
	State("closed"),
);
`

const jsRenamedImportQueueConfigSource = `import {
	Queue as HQueue,
	Config as HConfig,
	Define as HDefine,
	Initial as HInitial,
	Target as HTarget,
	State as HState,
} from "@stateforward/hsm";

export const priorityQueue = HQueue({
	Push: (ctx, event) => event,
	Pop: (ctx) => [undefined, false],
	Len: (ctx) => 0,
}), runtimeConfig = HConfig({
	ID: "door-1",
	Queue: priorityQueue,
}), DoorModel = HDefine(
	"Door",
	HInitial(HTarget("closed")),
	HState("closed"),
);
`

const goGroupedQueueConfigSource = `package sample

import hsm "github.com/stateforward/hsm.go"

var (
	priorityQueue = hsm.Queue(
		hsm.Push(func(ctx hsm.Context, event hsm.Event) hsm.DispatchResult {
			return hsm.Processed
		}),
		hsm.Pop(func(ctx hsm.Context) (hsm.Event, bool) {
			return hsm.Event{}, false
		}),
		hsm.Len(func(ctx hsm.Context) int {
			return 0
		}),
	)
	RuntimeConfig = hsm.Config(hsm.ID("door-1"), hsm.Queue(priorityQueue))
	DoorModel = hsm.Define(
		"Door",
		hsm.Initial(hsm.Target("closed")),
		hsm.State("closed"),
	)
)
`

const goAliasedQueueConfigSource = `package sample

import sfhsm "github.com/stateforward/hsm.go"

var (
	priorityQueue = sfhsm.Queue(
		sfhsm.Push(func(ctx sfhsm.Context, event sfhsm.Event) sfhsm.DispatchResult {
			return sfhsm.Processed
		}),
		sfhsm.Pop(func(ctx sfhsm.Context) (sfhsm.Event, bool) {
			return sfhsm.Event{}, false
		}),
		sfhsm.Len(func(ctx sfhsm.Context) int {
			return 0
		}),
	)
	RuntimeConfig = sfhsm.Config(sfhsm.ID("door-1"), sfhsm.Queue(priorityQueue))
	DoorModel = sfhsm.Define(
		"Door",
		sfhsm.Initial(sfhsm.Target("closed")),
		sfhsm.State("closed"),
	)
)
`

const goGroupedClockConfigSource = `package sample

import hsm "github.com/stateforward/hsm.go"

var (
	deterministicClock = hsm.Clock(
		hsm.After(func(duration any) any {
			return duration
		}),
		hsm.NewTimer(func(duration any) any {
			return nil
		}),
	)
	RuntimeConfig = hsm.Config(hsm.ID("door-1"), hsm.Clock(deterministicClock))
	DoorModel = hsm.Define(
		"Door",
		hsm.Initial(hsm.Target("closed")),
		hsm.State("closed"),
	)
)
`

const goAliasedClockConfigSource = `package sample

import sfhsm "github.com/stateforward/hsm.go"

var (
	deterministicClock = sfhsm.Clock(
		sfhsm.After(func(duration any) any {
			return duration
		}),
		sfhsm.NewTimer(func(duration any) any {
			return nil
		}),
	)
	RuntimeConfig = sfhsm.Config(sfhsm.ID("door-1"), sfhsm.Clock(deterministicClock))
	DoorModel = sfhsm.Define(
		"Door",
		sfhsm.Initial(sfhsm.Target("closed")),
		sfhsm.State("closed"),
	)
)
`

const goDefaultClockConfigSource = `package sample

import hsm "github.com/stateforward/hsm.go"

func init() {
	hsm.DefaultClock = hsm.Clock(
		hsm.After(func(duration any) any { return duration }),
		hsm.NewTimer(func(duration any) any { return nil }),
	)
}

var DoorModel = hsm.Define(
	"Door",
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed"),
)
`

const goAliasedDefaultClockConfigSource = `package sample

import sfhsm "github.com/stateforward/hsm.go"

func init() {
	sfhsm.DefaultClock = sfhsm.Clock(
		sfhsm.After(func(duration any) any { return duration }),
		sfhsm.NewTimer(func(duration any) any { return nil }),
	)
}

var DoorModel = sfhsm.Define(
	"Door",
	sfhsm.Initial(sfhsm.Target("closed")),
	sfhsm.State("closed"),
)
`

const dartGroupedQueueConfigSource = `import 'package:hsm/hsm.dart';

final priorityQueue = queue(
  push: (ctx, event) => event,
  pop: (ctx) => (null, false),
  len: (ctx) => 0,
), runtimeConfig = config([
  id('door-1'),
  queue(priorityQueue),
]), doorModel = define('Door', [
  initial(target('closed')),
  state('closed'),
]);
`

const dartAliasedQueueConfigSource = `import 'package:hsm/hsm.dart' as hsm;

final priorityQueue = hsm.queue(
  push: (ctx, event) => event,
  pop: (ctx) => (null, false),
  len: (ctx) => 0,
), runtimeConfig = hsm.config([
  hsm.id('door-1'),
  hsm.queue(priorityQueue),
]), doorModel = hsm.define('Door', [
  hsm.initial(hsm.target('closed')),
  hsm.state('closed'),
]);
`

const dartPascalCaseQueueConfigSource = `import 'package:hsm/hsm.dart';

final priorityQueue = Queue(
  Push: (ctx, event) => event,
  Pop: (ctx) => (null, false),
  Len: (ctx) => 0,
), runtimeConfig = Config([
  ID('door-1'),
  Queue(priorityQueue),
]), doorModel = Define('Door', [
  Initial(Target('closed')),
  State('closed'),
]);
`

const dartGroupedClockConfigSource = `import 'package:hsm/hsm.dart';

final deterministicClock = Clock(
  after: (duration) => Future.value(duration),
  newTimer: (duration) => Timer(duration, () {}),
), runtimeConfig = Config([
  id('door-1'),
  Clock(deterministicClock),
]), doorModel = define('Door', [
  initial(target('closed')),
  state('closed'),
]);
`

const dartAliasedClockConfigSource = `import 'package:hsm/hsm.dart' as hsm;

final deterministicClock = hsm.Clock(
  after: (duration) => Future.value(duration),
  newTimer: (duration) => Timer(duration, () {}),
), runtimeConfig = hsm.Config([
  hsm.id('door-1'),
  hsm.Clock(deterministicClock),
]), doorModel = hsm.define('Door', [
  hsm.initial(hsm.target('closed')),
  hsm.state('closed'),
]);
`

const dartDefaultClockConfigSource = `import 'package:hsm/hsm.dart' as hsm;

final configureDefaultClock = () {
  hsm.defaultClock = hsm.clock(
    after: (duration) => Future.value(duration),
    newTimer: (duration) => Timer(duration, () {}),
  );
};

final doorModel = hsm.define('Door', [
  hsm.initial(hsm.target('closed')),
  hsm.state('closed'),
]);
`

const pythonGroupedQueueConfigSource = `import hsm

priority_queue, runtime_config, model = hsm.Queue(
    Push=lambda ctx, event: event,
    Pop=lambda ctx: (None, False),
    Len=lambda ctx: 0,
), hsm.Config(
    ID="door-1",
    Queue=priority_queue,
), hsm.Define(
    "Door",
    hsm.Initial(hsm.Target("closed")),
    hsm.State("closed"),
)
`

const pythonGroupedClockConfigSource = `import hsm

deterministic_clock, runtime_config, model = hsm.Clock(
    After=lambda duration: duration,
    NewTimer=lambda duration: None,
), hsm.Config(
    ID="door-1",
    Clock=deterministic_clock,
), hsm.Define(
    "Door",
    hsm.Initial(hsm.Target("closed")),
    hsm.State("closed"),
)
`

const pythonDefaultClockConfigSource = `import hsm

hsm.default_clock = hsm.clock(
    after=lambda duration: duration,
    new_timer=lambda duration: None,
)

model = hsm.define(
    "Door",
    hsm.initial(hsm.target("closed")),
    hsm.state("closed"),
)
`

const pythonSnakeCaseQueueConfigSource = `import hsm

priority_queue, runtime_config, model = hsm.queue(
    push=lambda ctx, event: event,
    pop=lambda ctx: (None, False),
    len=lambda ctx: 0,
), hsm.config(
    id="door-1",
    queue=priority_queue,
), hsm.define(
    "Door",
    hsm.initial(hsm.target("closed")),
    hsm.state("closed"),
)
`

const pythonRenamedImportQueueConfigSource = `from hsm import (
    Queue as HQueue,
    Config as HConfig,
    Define as HDefine,
    Initial as HInitial,
    Target as HTarget,
    State as HState,
)

priority_queue, runtime_config, model = HQueue(
    Push=lambda ctx, event: event,
    Pop=lambda ctx: (None, False),
    Len=lambda ctx: 0,
), HConfig(
    ID="door-1",
    Queue=priority_queue,
), HDefine(
    "Door",
    HInitial(HTarget("closed")),
    HState("closed"),
)
`

const csharpGroupedQueueConfigSource = `using Stateforward.Hsm;

public static class Sample
{
    static readonly object PriorityQueue = Hsm.Queue(
        Push: (ctx, @event) => @event,
        Pop: ctx => (null, false),
        Len: ctx => 0
    ), RuntimeConfig = Hsm.Config(
        Hsm.ID("door-1"),
        Hsm.Queue(PriorityQueue)
    ), DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed")
    );
}
`

const csharpAliasedQueueConfigSource = `using H = Stateforward.Hsm.Hsm;
using Stateforward.Hsm;

public static class Sample
{
    static readonly object PriorityQueue = H.Queue(
        Push: (ctx, @event) => @event,
        Pop: ctx => (null, false),
        Len: ctx => 0
    ), RuntimeConfig = H.Config(
        H.ID("door-1"),
        H.Queue(PriorityQueue)
    ), DoorModel = H.Define(
        "Door",
        H.Initial(H.Target("closed")),
        H.State("closed")
    );
}
`

const csharpGroupedClockConfigSource = `using Stateforward.Hsm;

public static class Sample
{
    static readonly object DeterministicClock = Hsm.Clock(
        Hsm.After(duration => duration),
        Hsm.NewTimer(duration => new object())
    ), RuntimeConfig = Hsm.Config(
        Hsm.ID("door-1"),
        Hsm.Clock(DeterministicClock)
    ), DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed")
    );
}
`

const csharpAliasedClockConfigSource = `using H = Stateforward.Hsm.Hsm;
using Stateforward.Hsm;

public static class Sample
{
    static readonly object DeterministicClock = H.Clock(
        H.After(duration => duration),
        H.NewTimer(duration => new object())
    ), RuntimeConfig = H.Config(
        H.ID("door-1"),
        H.Clock(DeterministicClock)
    ), DoorModel = H.Define(
        "Door",
        H.Initial(H.Target("closed")),
        H.State("closed")
    );
}
`

const csharpDefaultClockConfigSource = `using Stateforward.Hsm;

public static class Sample
{
    static Sample()
    {
        Hsm.DefaultClock = Hsm.Clock(
            Hsm.After(duration => duration),
            Hsm.NewTimer(duration => new object())
        );
    }

    static readonly Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed")
    );
}
`

const csharpAliasedDefaultClockConfigSource = `using H = Stateforward.Hsm.Hsm;
using Stateforward.Hsm;

public static class Sample
{
    static Sample()
    {
        H.DefaultClock = H.Clock(
            H.After(duration => duration),
            H.NewTimer(duration => new object())
        );
    }

    static readonly Model DoorModel = H.Define(
        "Door",
        H.Initial(H.Target("closed")),
        H.State("closed")
    );
}
`

const javaGroupedQueueConfigSource = `import com.stateforward.hsm.*;

public final class Sample {
    static final Object PriorityQueue = Hsm.Queue(
        Hsm.Push((ctx, event) -> event),
        Hsm.Pop(ctx -> new Object[] { null, false }),
        Hsm.Len(ctx -> 0)
    ), RuntimeConfig = Hsm.Config(
        Hsm.ID("door-1"),
        Hsm.Queue(PriorityQueue)
    ), DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed")
    );
}
`

const javaStaticImportQueueConfigSource = `import static com.stateforward.hsm.Hsm.*;
import com.stateforward.hsm.*;

public final class Sample {
    static final Object PriorityQueue = Queue(
        Push((ctx, event) -> event),
        Pop(ctx -> new Object[] { null, false }),
        Len(ctx -> 0)
    ), RuntimeConfig = Config(
        ID("door-1"),
        Queue(PriorityQueue)
    ), DoorModel = Define(
        "Door",
        Initial(Target("closed")),
        State("closed")
    );
}
`

const javaGroupedClockConfigSource = `import com.stateforward.hsm.*;

public final class Sample {
    static final Object DeterministicClock = Hsm.Clock(
        Hsm.After(duration -> duration),
        Hsm.NewTimer(duration -> new Object())
    ), RuntimeConfig = Hsm.Config(
        Hsm.ID("door-1"),
        Hsm.Clock(DeterministicClock)
    ), DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed")
    );
}
`

const javaStaticImportClockConfigSource = `import static com.stateforward.hsm.Hsm.*;
import com.stateforward.hsm.*;

public final class Sample {
    static final Object DeterministicClock = Clock(
        After(duration -> duration),
        NewTimer(duration -> new Object())
    ), RuntimeConfig = Config(
        ID("door-1"),
        Clock(DeterministicClock)
    ), DoorModel = Define(
        "Door",
        Initial(Target("closed")),
        State("closed")
    );
}
`

const javaDefaultClockConfigSource = `import com.stateforward.hsm.*;

public final class Sample {
    static {
        Hsm.DefaultClock = Hsm.Clock(
            Hsm.After(duration -> duration),
            Hsm.NewTimer(duration -> new Object())
        );
    }

    static final Model DoorModel = Hsm.Define(
        "Door",
        Hsm.Initial(Hsm.Target("closed")),
        Hsm.State("closed")
    );
}
`

const javaStaticImportDefaultClockConfigSource = `import static com.stateforward.hsm.Hsm.*;
import com.stateforward.hsm.*;

public final class Sample {
    static {
        DefaultClock = Clock(
            After(duration -> duration),
            NewTimer(duration -> new Object())
        );
    }

    static final Model DoorModel = Define(
        "Door",
        Initial(Target("closed")),
        State("closed")
    );
}
`

const cppGroupedQueueConfigSource = `#include "hsm/hsm.hpp"

static auto PriorityQueue = hsm::queue(
  hsm::push([](auto& ctx, auto& event) { return event; }),
  hsm::pop([](auto& ctx) { return std::pair<hsm::EventBase, bool>{hsm::EventBase{}, false}; }),
  hsm::len([](auto& ctx) { return 0; })
), RuntimeConfig = hsm::config(
  hsm::id("door-1"),
  hsm::queue(PriorityQueue)
), DoorModel = hsm::define(
  "Door",
  hsm::initial(hsm::target("closed")),
  hsm::state("closed")
);
`

const cppGroupedClockConfigSource = `#include "hsm/hsm.hpp"

static auto DeterministicClock = hsm::clock(
  hsm::after([](auto duration) { return duration; }),
  hsm::new_timer([](auto duration) { return nullptr; })
), RuntimeConfig = hsm::config(
  hsm::id("door-1"),
  hsm::clock(DeterministicClock)
), DoorModel = hsm::define(
  "Door",
  hsm::initial(hsm::target("closed")),
  hsm::state("closed")
);
`

const zigQueueConfigSource = `const hsm = @import("hsm");

pub const priority_queue = hsm.queue(.{
    .push = queue_push,
    .pop = queue_pop,
    .len = queue_len,
});

pub const runtime_config = hsm.config(.{
    hsm.id("door-1"),
    hsm.queue(priority_queue),
});

pub const door_model = hsm.define("Door", .{
    hsm.initial(hsm.target("closed")),
    hsm.state("closed", .{}),
});
`

const zigClockConfigSource = `const hsm = @import("hsm");

pub const deterministic_clock = hsm.clock(.{
    .after = clock_after,
    .new_timer = clock_new_timer,
});

pub const runtime_config = hsm.config(.{
    hsm.id("door-1"),
    hsm.clock(deterministic_clock),
});

pub const door_model = hsm.define("Door", .{
    hsm.initial(hsm.target("closed")),
    hsm.state("closed", .{}),
});
`

const rustQueueConfigSource = `use hsm::*;

const PRIORITY_QUEUE: Queue = queue!(
    push!(queue_push),
    pop!(queue_pop),
    len!(queue_len)
);

const RUNTIME_CONFIG: Config = config!(
    id!("door-1"),
    queue!(PRIORITY_QUEUE)
);

const DOOR: Model = define!("Door",
    initial!(target!("closed")),
    state!("closed")
);
`

const rustQualifiedQueueConfigSource = `use hsm::{Config, Model, Queue};

const PRIORITY_QUEUE: Queue = hsm::queue!(
    hsm::push!(queue_push),
    hsm::pop!(queue_pop),
    hsm::len!(queue_len)
);

const RUNTIME_CONFIG: Config = hsm::config!(
    hsm::id!("door-1"),
    hsm::queue!(PRIORITY_QUEUE)
);

const DOOR: Model = hsm::define!("Door",
    hsm::initial!(hsm::target!("closed")),
    hsm::state!("closed")
);
`

const rustClockConfigSource = `use hsm::*;

const DETERMINISTIC_CLOCK: Clock = clock!(
    after!(clock_after),
    new_timer!(clock_new_timer)
);

const RUNTIME_CONFIG: Config = config!(
    id!("door-1"),
    clock!(DETERMINISTIC_CLOCK)
);

const DOOR: Model = define!("Door",
    initial!(target!("closed")),
    state!("closed")
);
`

const rustQualifiedClockConfigSource = `use hsm::{Clock, Config, Model};

const DETERMINISTIC_CLOCK: Clock = hsm::clock!(
    hsm::after!(clock_after),
    hsm::new_timer!(clock_new_timer)
);

const RUNTIME_CONFIG: Config = hsm::config!(
    hsm::id!("door-1"),
    hsm::clock!(DETERMINISTIC_CLOCK)
);

const DOOR: Model = hsm::define!("Door",
    hsm::initial!(hsm::target!("closed")),
    hsm::state!("closed")
);
`

const jsonIRQueueConfigSource = `{
  "source_language": "typescript",
  "globals": [
    {
      "id": "global_1",
      "code": "const priorityQueue = hsm.Queue({ Push: (ctx, event) => event, Pop: (ctx) => [undefined, false], Len: (ctx) => 0 });"
    },
    {
      "id": "global_2",
      "code": "const runtimeConfig = hsm.Config({ ID: \"door-1\", Queue: priorityQueue });"
    }
  ],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "states": [{ "id": "state_1", "name": "closed", "kind": "state" }]
  }]
}`

const jsonIRClockConfigSource = `{
  "source_language": "typescript",
  "globals": [
    {
      "id": "global_1",
      "code": "const deterministicClock = hsm.Clock({ After: (duration) => duration, NewTimer: (duration) => undefined });"
    },
    {
      "id": "global_2",
      "code": "const runtimeConfig = hsm.Config({ ID: \"door-1\", Clock: deterministicClock });"
    }
  ],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "states": [{ "id": "state_1", "name": "closed", "kind": "state" }]
  }]
}`

const jsonIRDefaultClockConfigSource = `{
  "source_language": "typescript",
  "globals": [
    {
      "id": "global_1",
      "code": "hsm.DefaultClock = hsm.Clock({ After: (duration) => duration, NewTimer: (duration) => undefined });"
    }
  ],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "states": [{ "id": "state_1", "name": "closed", "kind": "state" }]
  }]
}`

func TestRuntimeQueueConfigIsPreservedAsGlobalContext(t *testing.T) {
	program, err := NewTypeScriptFrontend().Parse(context.Background(), SourceInput{Path: "door.ts", Data: []byte(tsQueueConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want queue and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"hsm.Queue", "Push:", "Pop:", "Len:", "hsm.Config", "Queue: priorityQueue"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("runtime global context missing %q:\n%s", needle, text)
		}
	}
}

func TestRuntimeClockConfigIsPreservedAsGlobalContext(t *testing.T) {
	program, err := NewTypeScriptFrontend().Parse(context.Background(), SourceInput{Path: "door.ts", Data: []byte(tsClockConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want clock and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"hsm.Clock", "After:", "NewTimer:", "hsm.Config", "Clock: deterministicClock"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("runtime global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "hsm.Define(") {
		t.Fatalf("model declaration leaked into runtime globals:\n%s", text)
	}
}

func TestRuntimeDefaultClockAssignmentIsPreservedAsGlobalContext(t *testing.T) {
	cases := []struct {
		name        string
		language    Language
		path        string
		source      string
		clockNeedle string
		hookNeedles []string
	}{
		{name: "typescript", language: LanguageTS, path: "door.ts", source: tsDefaultClockConfigSource, clockNeedle: "hsm.DefaultClock = hsm.Clock({", hookNeedles: []string{"After", "NewTimer"}},
		{name: "javascript", language: LanguageJS, path: "door.js", source: tsDefaultClockConfigSource, clockNeedle: "hsm.DefaultClock = hsm.Clock({", hookNeedles: []string{"After", "NewTimer"}},
		{name: "typescript namespace alias", language: LanguageTS, path: "door.ts", source: tsDefaultClockAliasConfigSource, clockNeedle: "runtime.DefaultClock = runtime.Clock({", hookNeedles: []string{"After", "NewTimer"}},
		{name: "typescript default import alias", language: LanguageTS, path: "door.ts", source: tsDefaultClockDefaultImportConfigSource, clockNeedle: "runtime.DefaultClock = runtime.Clock({", hookNeedles: []string{"After", "NewTimer"}},
		{name: "javascript camelCase alias", language: LanguageJS, path: "door.js", source: tsDefaultClockCamelCaseAliasConfigSource, clockNeedle: "hsm.defaultClock = hsm.clock({", hookNeedles: []string{"after", "newTimer"}},
		{name: "typescript named camelCase import", language: LanguageTS, path: "door.ts", source: tsDefaultClockNamedCamelCaseImportConfigSource, clockNeedle: "defaultClock = clock({", hookNeedles: []string{"after", "newTimer"}},
		{name: "go init assignment", language: LanguageGo, path: "door.go", source: goDefaultClockConfigSource, clockNeedle: "hsm.DefaultClock = hsm.Clock(", hookNeedles: []string{"hsm.After", "hsm.NewTimer"}},
		{name: "dart closure assignment", language: LanguageDart, path: "door.dart", source: dartDefaultClockConfigSource, clockNeedle: "hsm.defaultClock = hsm.clock(", hookNeedles: []string{"after:", "newTimer:"}},
		{name: "python snake_case assignment", language: LanguagePython, path: "door.py", source: pythonDefaultClockConfigSource, clockNeedle: "hsm.default_clock = hsm.clock(", hookNeedles: []string{"after=", "new_timer="}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program, err := NewCompiler().frontends[tc.language].Parse(context.Background(), SourceInput{Path: tc.path, Data: []byte(tc.source)})
			if err != nil {
				t.Fatal(err)
			}
			if len(program.Models) != 1 || program.Models[0].Name != "Door" {
				t.Fatalf("models = %#v", program.Models)
			}
			if len(program.Globals) != 1 {
				t.Fatalf("globals = %#v, want default clock runtime global", program.Globals)
			}
			text := program.Globals[0].Code
			if !strings.Contains(text, tc.clockNeedle) {
				t.Fatalf("default clock global context missing %q:\n%s", tc.clockNeedle, text)
			}
			for _, needle := range tc.hookNeedles {
				if !strings.Contains(text, needle) {
					t.Fatalf("default clock global context missing hook %q:\n%s", needle, text)
				}
			}
		})
	}
}

func TestRuntimeDefaultClockAssignmentIgnoresUnrelatedProperty(t *testing.T) {
	source := `import * as hsm from "@stateforward/hsm";

other.DefaultClock = hsm.Clock({
	After: (duration) => Promise.resolve(duration),
});

export const DoorModel = hsm.Define(
	"Door",
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed"),
);
`
	program, err := NewTypeScriptFrontend().Parse(context.Background(), SourceInput{Path: "door.ts", Data: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("globals = %#v, want unrelated DefaultClock assignment ignored", program.Globals)
	}
}

func TestTypeScriptRuntimeQueueConfigIsPreservedFromGroupedDeclarationWithModel(t *testing.T) {
	program, err := NewTypeScriptFrontend().Parse(context.Background(), SourceInput{Path: "door.ts", Data: []byte(tsGroupedQueueConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want queue and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"export const priorityQueue = hsm.Queue({", "Push:", "Pop:", "Len:", "export const runtimeConfig = hsm.Config({", "Queue: priorityQueue"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript grouped runtime global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "hsm.Define(") {
		t.Fatalf("model declaration leaked into runtime globals:\n%s", text)
	}
}

func TestTypeScriptRuntimeQueueConfigIsPreservedFromCamelCaseGroupedDeclarationWithModel(t *testing.T) {
	program, err := NewTypeScriptFrontend().Parse(context.Background(), SourceInput{Path: "door.ts", Data: []byte(tsCamelCaseQueueConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want queue and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"export const priorityQueue = hsm.queue({", "push:", "pop:", "len:", "runtimeConfig = hsm.config({", "queue: priorityQueue"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript camelCase runtime global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "define(") {
		t.Fatalf("model declaration leaked into TypeScript camelCase runtime globals:\n%s", text)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
}

func TestTypeScriptRuntimeQueueConfigIsPreservedFromRenamedImports(t *testing.T) {
	program, err := NewTypeScriptFrontend().Parse(context.Background(), SourceInput{Path: "door.ts", Data: []byte(tsRenamedImportQueueConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 0 {
		t.Fatalf("hsm named import aliases should be compiler-owned, got %#v", program.Imports)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want queue and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"priorityQueue = HQueue({", "Push:", "Pop:", "Len:", "runtimeConfig = HConfig({", "Queue: priorityQueue"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("runtime global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "HDefine(") {
		t.Fatalf("model declaration leaked into runtime globals:\n%s", text)
	}
}

func TestJavaScriptRuntimeQueueConfigIsPreservedFromGroupedDeclarationWithModel(t *testing.T) {
	program, err := NewJavaScriptFrontend().Parse(context.Background(), SourceInput{Path: "door.js", Data: []byte(jsGroupedQueueConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want queue and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"export const priorityQueue = hsm.Queue({", "Push:", "Pop:", "Len:", "export const runtimeConfig = hsm.Config({", "Queue: priorityQueue"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("JavaScript grouped runtime global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "hsm.Define(") {
		t.Fatalf("model declaration leaked into runtime globals:\n%s", text)
	}
}

func TestJavaScriptRuntimeQueueConfigIsPreservedFromCamelCaseGroupedDeclarationWithModel(t *testing.T) {
	program, err := NewJavaScriptFrontend().Parse(context.Background(), SourceInput{Path: "door.js", Data: []byte(jsCamelCaseQueueConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want queue and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"export const priorityQueue = hsm.queue({", "push:", "pop:", "len:", "export const runtimeConfig = hsm.config({", "queue: priorityQueue"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("JavaScript camelCase grouped runtime global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "hsm.define(") {
		t.Fatalf("model declaration leaked into runtime globals:\n%s", text)
	}
}

func TestJavaScriptRuntimeQueueConfigIsPreservedFromRenamedImports(t *testing.T) {
	program, err := NewJavaScriptFrontend().Parse(context.Background(), SourceInput{Path: "door.js", Data: []byte(jsRenamedImportQueueConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 0 {
		t.Fatalf("hsm named import aliases should be compiler-owned, got %#v", program.Imports)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want queue and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"priorityQueue = HQueue({", "Push:", "Pop:", "Len:", "runtimeConfig = HConfig({", "Queue: priorityQueue"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("JavaScript renamed import runtime global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "HDefine(") {
		t.Fatalf("model declaration leaked into runtime globals:\n%s", text)
	}
}

func TestGoRuntimeQueueConfigIsPreservedFromGroupedVarWithModel(t *testing.T) {
	program, err := NewGoFrontend().Parse(context.Background(), SourceInput{Path: "door.go", Data: []byte(goGroupedQueueConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want queue and config globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"var priorityQueue = hsm.Queue(", "hsm.Push(", "hsm.Pop(", "hsm.Len(", "var RuntimeConfig = hsm.Config", "hsm.Queue(priorityQueue)"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go runtime global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "hsm.Define(") {
		t.Fatalf("model declaration leaked into runtime globals:\n%s", text)
	}
}

func TestGoRuntimeQueueConfigIsPreservedFromAliasedRuntimeImport(t *testing.T) {
	program, err := NewGoFrontend().Parse(context.Background(), SourceInput{Path: "door.go", Data: []byte(goAliasedQueueConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 0 {
		t.Fatalf("hsm runtime alias imports should be compiler-owned, got %#v", program.Imports)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want queue and config globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"var priorityQueue = hsm.Queue(", "hsm.Push(", "hsm.Pop(", "hsm.Len(", "var RuntimeConfig = hsm.Config", "hsm.ID(\"door-1\")", "hsm.Queue(priorityQueue)"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("aliased Go runtime global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "sfhsm.") || strings.Contains(text, "hsm.Define(") {
		t.Fatalf("aliased Go runtime globals were not normalized or model declaration leaked:\n%s", text)
	}
}

func TestGoRuntimeClockConfigIsPreservedFromGroupedVarWithModel(t *testing.T) {
	program, err := NewGoFrontend().Parse(context.Background(), SourceInput{Path: "door.go", Data: []byte(goGroupedClockConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want clock and config globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"var deterministicClock = hsm.Clock(", "hsm.After(", "hsm.NewTimer(", "var RuntimeConfig = hsm.Config", "hsm.Clock(deterministicClock)"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go runtime clock global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "hsm.Define(") {
		t.Fatalf("model declaration leaked into runtime clock globals:\n%s", text)
	}
}

func TestGoRuntimeClockConfigIsPreservedFromAliasedRuntimeImport(t *testing.T) {
	program, err := NewGoFrontend().Parse(context.Background(), SourceInput{Path: "door.go", Data: []byte(goAliasedClockConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 0 {
		t.Fatalf("hsm runtime alias imports should be compiler-owned, got %#v", program.Imports)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want clock and config globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"var deterministicClock = hsm.Clock(", "hsm.After(", "hsm.NewTimer(", "var RuntimeConfig = hsm.Config", "hsm.ID(\"door-1\")", "hsm.Clock(deterministicClock)"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("aliased Go runtime clock context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "sfhsm.") || strings.Contains(text, "hsm.Define(") {
		t.Fatalf("aliased Go runtime clock globals were not normalized or model declaration leaked:\n%s", text)
	}
}

func TestGoRuntimeDefaultClockAssignmentIsPreservedAsGlobalContext(t *testing.T) {
	program, err := NewGoFrontend().Parse(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDefaultClockConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 1 {
		t.Fatalf("globals = %#v, want default clock init global", program.Globals)
	}
	text := program.Globals[0].Code
	for _, needle := range []string{"func init() {", "hsm.DefaultClock = hsm.Clock(", "hsm.After(", "hsm.NewTimer("} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Go default clock global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "hsm.Define(") {
		t.Fatalf("model declaration leaked into Go default clock global:\n%s", text)
	}
}

func TestGoRuntimeDefaultClockAssignmentIsPreservedFromAliasedRuntimeImport(t *testing.T) {
	program, err := NewGoFrontend().Parse(context.Background(), SourceInput{Path: "door.go", Data: []byte(goAliasedDefaultClockConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 0 {
		t.Fatalf("hsm runtime alias imports should be compiler-owned, got %#v", program.Imports)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 1 {
		t.Fatalf("globals = %#v, want default clock init global", program.Globals)
	}
	text := program.Globals[0].Code
	for _, needle := range []string{"func init() {", "hsm.DefaultClock = hsm.Clock(", "hsm.After(", "hsm.NewTimer("} {
		if !strings.Contains(text, needle) {
			t.Fatalf("aliased Go default clock global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "sfhsm.") || strings.Contains(text, "hsm.Define(") {
		t.Fatalf("aliased Go default clock global was not normalized or model declaration leaked:\n%s", text)
	}
}

func TestDartRuntimeQueueConfigIsPreservedFromGroupedDeclarationWithModel(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartGroupedQueueConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want queue and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"final priorityQueue = queue(", "push:", "pop:", "len:", "final runtimeConfig = config([", "queue(priorityQueue)"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Dart grouped runtime global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "define(") {
		t.Fatalf("model declaration leaked into runtime globals:\n%s", text)
	}
}

func TestDartRuntimeQueueConfigIsPreservedFromPascalCaseGroupedDeclarationWithModel(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartPascalCaseQueueConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want queue and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"final priorityQueue = Queue(", "Push:", "Pop:", "Len:", "final runtimeConfig = Config([", "Queue(priorityQueue)"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Dart PascalCase runtime global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Define(") {
		t.Fatalf("model declaration leaked into Dart PascalCase runtime globals:\n%s", text)
	}
}

func TestDartRuntimeQueueConfigIsPreservedFromAliasedRuntimeDeclarationWithModel(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartAliasedQueueConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want queue and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"final priorityQueue = hsm.queue(", "push:", "pop:", "len:", "final runtimeConfig = hsm.config([", "hsm.queue(priorityQueue)"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Dart aliased runtime global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "define(") {
		t.Fatalf("model declaration leaked into Dart aliased runtime globals:\n%s", text)
	}
}

func TestDartRuntimeClockConfigIsPreservedFromGroupedDeclarationWithModel(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartGroupedClockConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want clock and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"final deterministicClock = Clock(", "after:", "newTimer:", "final runtimeConfig = Config([", "Clock(deterministicClock)"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Dart grouped runtime clock context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "define(") {
		t.Fatalf("model declaration leaked into Dart runtime clock globals:\n%s", text)
	}
}

func TestDartRuntimeClockConfigIsPreservedFromAliasedRuntimeDeclarationWithModel(t *testing.T) {
	program, err := NewDartFrontend().Parse(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartAliasedClockConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want clock and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"final deterministicClock = hsm.Clock(", "after:", "newTimer:", "final runtimeConfig = hsm.Config([", "hsm.Clock(deterministicClock)"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Dart aliased runtime clock context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "define(") {
		t.Fatalf("model declaration leaked into Dart aliased clock globals:\n%s", text)
	}
}

func TestPythonRuntimeQueueConfigIsPreservedFromGroupedAssignmentWithModel(t *testing.T) {
	program, err := NewPythonFrontend().Parse(context.Background(), SourceInput{Path: "door.py", Data: []byte(pythonGroupedQueueConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want queue and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"priority_queue = hsm.Queue(", "Push=", "Pop=", "Len=", "runtime_config = hsm.Config(", "Queue=priority_queue"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python grouped runtime global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "hsm.Define(") {
		t.Fatalf("model assignment leaked into runtime globals:\n%s", text)
	}
}

func TestPythonRuntimeClockConfigIsPreservedFromGroupedAssignmentWithModel(t *testing.T) {
	program, err := NewPythonFrontend().Parse(context.Background(), SourceInput{Path: "door.py", Data: []byte(pythonGroupedClockConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want clock and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"deterministic_clock = hsm.Clock(", "After=", "NewTimer=", "runtime_config = hsm.Config(", "Clock=deterministic_clock"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python grouped runtime clock context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "hsm.Define(") {
		t.Fatalf("model assignment leaked into runtime clock globals:\n%s", text)
	}
}

func TestPythonRuntimeDefaultClockAssignmentIsPreservedAsGlobalContext(t *testing.T) {
	program, err := NewPythonFrontend().Parse(context.Background(), SourceInput{Path: "door.py", Data: []byte(pythonDefaultClockConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 1 {
		t.Fatalf("globals = %#v, want default clock runtime global", program.Globals)
	}
	text := program.Globals[0].Code
	for _, needle := range []string{"hsm.default_clock = hsm.clock(", "after=", "new_timer="} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python default clock global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "hsm.define(") {
		t.Fatalf("model assignment leaked into Python default clock global:\n%s", text)
	}
}

func TestPythonRuntimeQueueConfigIsPreservedFromRenamedImports(t *testing.T) {
	program, err := NewPythonFrontend().Parse(context.Background(), SourceInput{Path: "door.py", Data: []byte(pythonRenamedImportQueueConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 0 {
		t.Fatalf("hsm runtime import aliases should be compiler-owned, got %#v", program.Imports)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want queue and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"priority_queue = HQueue(", "Push=", "Pop=", "Len=", "runtime_config = HConfig(", "Queue=priority_queue"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("renamed Python runtime global context missing %q:\n%s", needle, text)
		}
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
}

func TestCSharpRuntimeQueueConfigIsPreservedFromGroupedFieldWithModel(t *testing.T) {
	program, err := NewCSharpFrontend().Parse(context.Background(), SourceInput{Path: "door.cs", Data: []byte(csharpGroupedQueueConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want queue and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"private static readonly object PriorityQueue = Hsm.Queue(", "Push:", "Pop:", "Len:", "private static readonly object RuntimeConfig = Hsm.Config(", "Hsm.Queue(PriorityQueue)"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C# grouped runtime global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Hsm.Define(") {
		t.Fatalf("model field leaked into runtime globals:\n%s", text)
	}
}

func TestCSharpRuntimeQueueConfigIsPreservedFromAliasedRuntimeFieldWithModel(t *testing.T) {
	program, err := NewCSharpFrontend().Parse(context.Background(), SourceInput{Path: "door.cs", Data: []byte(csharpAliasedQueueConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 0 {
		t.Fatalf("hsm runtime alias imports should be compiler-owned, got %#v", program.Imports)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want queue and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"private static readonly object PriorityQueue = Hsm.Queue(", "Push:", "Pop:", "Len:", "private static readonly object RuntimeConfig = Hsm.Config(", "Hsm.ID(\"door-1\")", "Hsm.Queue(PriorityQueue)"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("aliased C# runtime global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "H.") || strings.Contains(text, "Hsm.Define(") {
		t.Fatalf("aliased runtime globals were not normalized or model field leaked:\n%s", text)
	}
}

func TestCSharpRuntimeClockConfigIsPreservedFromGroupedFieldWithModel(t *testing.T) {
	program, err := NewCSharpFrontend().Parse(context.Background(), SourceInput{Path: "door.cs", Data: []byte(csharpGroupedClockConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want clock and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"private static readonly object DeterministicClock = Hsm.Clock(", "Hsm.After(", "Hsm.NewTimer(", "private static readonly object RuntimeConfig = Hsm.Config(", "Hsm.Clock(DeterministicClock)"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C# grouped runtime clock context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Hsm.Define(") {
		t.Fatalf("model field leaked into runtime clock globals:\n%s", text)
	}
}

func TestCSharpRuntimeClockConfigIsPreservedFromAliasedRuntimeFieldWithModel(t *testing.T) {
	program, err := NewCSharpFrontend().Parse(context.Background(), SourceInput{Path: "door.cs", Data: []byte(csharpAliasedClockConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 0 {
		t.Fatalf("hsm runtime alias imports should be compiler-owned, got %#v", program.Imports)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want clock and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"private static readonly object DeterministicClock = Hsm.Clock(", "Hsm.After(", "Hsm.NewTimer(", "private static readonly object RuntimeConfig = Hsm.Config(", "Hsm.ID(\"door-1\")", "Hsm.Clock(DeterministicClock)"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("aliased C# runtime clock context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "H.") || strings.Contains(text, "Hsm.Define(") {
		t.Fatalf("aliased runtime clock globals were not normalized or model field leaked:\n%s", text)
	}
}

func TestCSharpRuntimeDefaultClockAssignmentIsPreservedAsGlobalContext(t *testing.T) {
	program, err := NewCSharpFrontend().Parse(context.Background(), SourceInput{Path: "door.cs", Data: []byte(csharpDefaultClockConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 1 {
		t.Fatalf("globals = %#v, want default clock runtime global", program.Globals)
	}
	text := program.Globals[0].Code
	for _, needle := range []string{"Hsm.DefaultClock = Hsm.Clock(", "Hsm.After(", "Hsm.NewTimer("} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C# default clock global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Hsm.Define(") {
		t.Fatalf("model field leaked into default clock global:\n%s", text)
	}
}

func TestCSharpRuntimeDefaultClockAssignmentIsPreservedFromAliasedRuntimeContext(t *testing.T) {
	program, err := NewCSharpFrontend().Parse(context.Background(), SourceInput{Path: "door.cs", Data: []byte(csharpAliasedDefaultClockConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 0 {
		t.Fatalf("hsm runtime alias imports should be compiler-owned, got %#v", program.Imports)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 1 {
		t.Fatalf("globals = %#v, want default clock runtime global", program.Globals)
	}
	text := program.Globals[0].Code
	for _, needle := range []string{"Hsm.DefaultClock = Hsm.Clock(", "Hsm.After(", "Hsm.NewTimer("} {
		if !strings.Contains(text, needle) {
			t.Fatalf("aliased C# default clock global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "H.") || strings.Contains(text, "Hsm.Define(") {
		t.Fatalf("aliased default clock global was not normalized or model field leaked:\n%s", text)
	}
}

func TestJavaRuntimeQueueConfigIsPreservedFromGroupedFieldWithModel(t *testing.T) {
	program, err := NewJavaFrontend().Parse(context.Background(), SourceInput{Path: "Door.java", Data: []byte(javaGroupedQueueConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want queue and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"static final Object PriorityQueue = Hsm.Queue(", "Hsm.Push(", "Hsm.Pop(", "Hsm.Len(", "static final Object RuntimeConfig = Hsm.Config(", "Hsm.Queue(PriorityQueue)"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Java grouped runtime global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Hsm.Define(") {
		t.Fatalf("model field leaked into runtime globals:\n%s", text)
	}
}

func TestJavaRuntimeQueueConfigIsPreservedFromStaticRuntimeImport(t *testing.T) {
	program, err := NewJavaFrontend().Parse(context.Background(), SourceInput{Path: "Door.java", Data: []byte(javaStaticImportQueueConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 0 {
		t.Fatalf("static hsm runtime imports should be compiler-owned, got %#v", program.Imports)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want queue and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"static final Object PriorityQueue = Queue(", "Push(", "Pop(", "Len(", "static final Object RuntimeConfig = Config(", "ID(\"door-1\")", "Queue(PriorityQueue)"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Java static-import runtime global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Define(") {
		t.Fatalf("model field leaked into static-import runtime globals:\n%s", text)
	}
}

func TestJavaRuntimeClockConfigIsPreservedFromGroupedFieldWithModel(t *testing.T) {
	program, err := NewJavaFrontend().Parse(context.Background(), SourceInput{Path: "Door.java", Data: []byte(javaGroupedClockConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want clock and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"static final Object DeterministicClock = Hsm.Clock(", "Hsm.After(", "Hsm.NewTimer(", "static final Object RuntimeConfig = Hsm.Config(", "Hsm.Clock(DeterministicClock)"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Java grouped runtime clock context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Hsm.Define(") {
		t.Fatalf("model field leaked into runtime globals:\n%s", text)
	}
}

func TestJavaRuntimeClockConfigIsPreservedFromStaticRuntimeImport(t *testing.T) {
	program, err := NewJavaFrontend().Parse(context.Background(), SourceInput{Path: "Door.java", Data: []byte(javaStaticImportClockConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 0 {
		t.Fatalf("static hsm runtime imports should be compiler-owned, got %#v", program.Imports)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want clock and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"static final Object DeterministicClock = Clock(", "After(", "NewTimer(", "static final Object RuntimeConfig = Config(", "ID(\"door-1\")", "Clock(DeterministicClock)"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Java static-import runtime clock context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Define(") {
		t.Fatalf("model field leaked into static-import runtime clock globals:\n%s", text)
	}
}

func TestJavaRuntimeDefaultClockAssignmentIsPreservedAsGlobalContext(t *testing.T) {
	program, err := NewJavaFrontend().Parse(context.Background(), SourceInput{Path: "Door.java", Data: []byte(javaDefaultClockConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 1 {
		t.Fatalf("globals = %#v, want default clock runtime global", program.Globals)
	}
	text := program.Globals[0].Code
	for _, needle := range []string{"Hsm.DefaultClock = Hsm.Clock(", "Hsm.After(", "Hsm.NewTimer("} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Java default clock global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Hsm.Define(") {
		t.Fatalf("model field leaked into runtime globals:\n%s", text)
	}
}

func TestJavaRuntimeDefaultClockAssignmentIsPreservedFromStaticRuntimeImport(t *testing.T) {
	program, err := NewJavaFrontend().Parse(context.Background(), SourceInput{Path: "Door.java", Data: []byte(javaStaticImportDefaultClockConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 0 {
		t.Fatalf("static hsm runtime imports should be compiler-owned, got %#v", program.Imports)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 1 {
		t.Fatalf("globals = %#v, want default clock runtime global", program.Globals)
	}
	text := program.Globals[0].Code
	for _, needle := range []string{"DefaultClock = Clock(", "After(", "NewTimer("} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Java static-import default clock global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "Define(") {
		t.Fatalf("model field leaked into static-import default clock global:\n%s", text)
	}
}

func TestCPPRuntimeQueueConfigIsPreservedFromGroupedDeclarationWithModel(t *testing.T) {
	program, err := NewCPPFrontend().Parse(context.Background(), SourceInput{Path: "door.cpp", Data: []byte(cppGroupedQueueConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want queue and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"static auto PriorityQueue = hsm::queue(", "hsm::push(", "hsm::pop(", "hsm::len(", "static auto RuntimeConfig = hsm::config(", "hsm::queue(PriorityQueue)"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C++ grouped runtime global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "define(") {
		t.Fatalf("model declaration leaked into runtime globals:\n%s", text)
	}
}

func TestCPPRuntimeClockConfigIsPreservedFromGroupedDeclarationWithModel(t *testing.T) {
	program, err := NewCPPFrontend().Parse(context.Background(), SourceInput{Path: "door.cpp", Data: []byte(cppGroupedClockConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want clock and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"static auto DeterministicClock = hsm::clock(", "hsm::after(", "hsm::new_timer(", "static auto RuntimeConfig = hsm::config(", "hsm::clock(DeterministicClock)"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("C++ grouped runtime clock context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "define(") {
		t.Fatalf("model declaration leaked into runtime clock globals:\n%s", text)
	}
}

func TestZigRuntimeQueueConfigIsPreservedAsGlobalContext(t *testing.T) {
	program, err := NewZigFrontend().Parse(context.Background(), SourceInput{Path: "door.zig", Data: []byte(zigQueueConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want queue and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"pub const priority_queue = hsm.queue(.{", ".push = queue_push", ".pop = queue_pop", ".len = queue_len", "pub const runtime_config = hsm.config(.{", "hsm.queue(priority_queue)"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Zig runtime global context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "hsm.define(") {
		t.Fatalf("model declaration leaked into runtime globals:\n%s", text)
	}
}

func TestZigRuntimeClockConfigIsPreservedAsGlobalContext(t *testing.T) {
	program, err := NewZigFrontend().Parse(context.Background(), SourceInput{Path: "door.zig", Data: []byte(zigClockConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want clock and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	for _, needle := range []string{"pub const deterministic_clock = hsm.clock(.{", ".after = clock_after", ".new_timer = clock_new_timer", "pub const runtime_config = hsm.config(.{", "hsm.clock(deterministic_clock)"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Zig runtime clock context missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "hsm.define(") {
		t.Fatalf("model declaration leaked into runtime clock globals:\n%s", text)
	}
}

func TestRustRuntimeQueueConfigIsPreservedAsGlobalContext(t *testing.T) {
	cases := []struct {
		name    string
		source  string
		needles []string
	}{
		{
			name:   "glob import",
			source: rustQueueConfigSource,
			needles: []string{
				"const PRIORITY_QUEUE: Queue = queue!(",
				"push!(queue_push)",
				"pop!(queue_pop)",
				"len!(queue_len)",
				"const RUNTIME_CONFIG: Config = config!(",
				"queue!(PRIORITY_QUEUE)",
			},
		},
		{
			name:   "qualified macros",
			source: rustQualifiedQueueConfigSource,
			needles: []string{
				"const PRIORITY_QUEUE: Queue = hsm::queue!(",
				"hsm::push!(queue_push)",
				"hsm::pop!(queue_pop)",
				"hsm::len!(queue_len)",
				"const RUNTIME_CONFIG: Config = hsm::config!(",
				"hsm::queue!(PRIORITY_QUEUE)",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program, err := NewRustFrontend().Parse(context.Background(), SourceInput{Path: "door.rs", Data: []byte(tc.source)})
			if err != nil {
				t.Fatal(err)
			}
			if len(program.Models) != 1 || program.Models[0].Name != "Door" {
				t.Fatalf("models = %#v", program.Models)
			}
			if len(program.Globals) != 2 {
				t.Fatalf("globals = %#v, want queue and config runtime globals", program.Globals)
			}
			text := program.Globals[0].Code + "\n" + program.Globals[1].Code
			for _, needle := range tc.needles {
				if !strings.Contains(text, needle) {
					t.Fatalf("Rust runtime global context missing %q:\n%s", needle, text)
				}
			}
			if strings.Contains(text, "define!(") {
				t.Fatalf("model declaration leaked into runtime globals:\n%s", text)
			}
		})
	}
}

func TestRustRuntimeClockConfigIsPreservedAsGlobalContext(t *testing.T) {
	cases := []struct {
		name    string
		source  string
		needles []string
	}{
		{
			name:   "glob import",
			source: rustClockConfigSource,
			needles: []string{
				"const DETERMINISTIC_CLOCK: Clock = clock!(",
				"after!(clock_after)",
				"new_timer!(clock_new_timer)",
				"const RUNTIME_CONFIG: Config = config!(",
				"clock!(DETERMINISTIC_CLOCK)",
			},
		},
		{
			name:   "qualified macros",
			source: rustQualifiedClockConfigSource,
			needles: []string{
				"const DETERMINISTIC_CLOCK: Clock = hsm::clock!(",
				"hsm::after!(clock_after)",
				"hsm::new_timer!(clock_new_timer)",
				"const RUNTIME_CONFIG: Config = hsm::config!(",
				"hsm::clock!(DETERMINISTIC_CLOCK)",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program, err := NewRustFrontend().Parse(context.Background(), SourceInput{Path: "door.rs", Data: []byte(tc.source)})
			if err != nil {
				t.Fatal(err)
			}
			if len(program.Models) != 1 || program.Models[0].Name != "Door" {
				t.Fatalf("models = %#v", program.Models)
			}
			if len(program.Globals) != 2 {
				t.Fatalf("globals = %#v, want clock and config runtime globals", program.Globals)
			}
			text := program.Globals[0].Code + "\n" + program.Globals[1].Code
			for _, needle := range tc.needles {
				if !strings.Contains(text, needle) {
					t.Fatalf("Rust runtime clock context missing %q:\n%s", needle, text)
				}
			}
			if strings.Contains(text, "define!(") {
				t.Fatalf("model declaration leaked into runtime clock globals:\n%s", text)
			}
		})
	}
}

func TestJSONIRRuntimeQueueConfigIsPreservedAsGlobalContext(t *testing.T) {
	program, err := NewJSONIRFrontend().Parse(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(jsonIRQueueConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	NormalizeSourceContext(program, program.SourceLanguage)
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want queue and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	if !strings.Contains(text, "hsm.Queue") || !strings.Contains(text, "hsm.Config") {
		t.Fatalf("JSON IR runtime global context missing queue/config:\n%s", text)
	}
	if program.Globals[0].Language != LanguageTS || program.Globals[1].Language != LanguageTS {
		t.Fatalf("JSON IR runtime globals not source-normalized: %#v", program.Globals)
	}
}

func TestJSONIRRuntimeClockConfigIsPreservedAsGlobalContext(t *testing.T) {
	program, err := NewJSONIRFrontend().Parse(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(jsonIRClockConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	NormalizeSourceContext(program, program.SourceLanguage)
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v, want clock and config runtime globals", program.Globals)
	}
	text := program.Globals[0].Code + "\n" + program.Globals[1].Code
	if !strings.Contains(text, "hsm.Clock") || !strings.Contains(text, "hsm.Config") {
		t.Fatalf("JSON IR runtime global context missing clock/config:\n%s", text)
	}
	if program.Globals[0].Language != LanguageTS || program.Globals[1].Language != LanguageTS {
		t.Fatalf("JSON IR runtime globals not source-normalized: %#v", program.Globals)
	}
}

func TestJSONIRRuntimeDefaultClockAssignmentIsPreservedAsGlobalContext(t *testing.T) {
	program, err := NewJSONIRFrontend().Parse(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(jsonIRDefaultClockConfigSource)})
	if err != nil {
		t.Fatal(err)
	}
	NormalizeSourceContext(program, program.SourceLanguage)
	if len(program.Models) != 1 || program.Models[0].Name != "Door" {
		t.Fatalf("models = %#v", program.Models)
	}
	if len(program.Globals) != 1 {
		t.Fatalf("globals = %#v, want default clock runtime global", program.Globals)
	}
	text := program.Globals[0].Code
	if !strings.Contains(text, "hsm.DefaultClock = hsm.Clock") || !strings.Contains(text, "NewTimer") {
		t.Fatalf("JSON IR default clock context missing expected code:\n%s", text)
	}
	if program.Globals[0].Language != LanguageTS {
		t.Fatalf("JSON IR default clock global not source-normalized: %#v", program.Globals[0])
	}
}

func TestNoAdapterQueueConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.ts", Data: []byte(tsQueueConfigSource)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguagePython,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"# Original typescript global global_1 preserved for manual porting:",
		"# const priorityQueue = hsm.Queue({",
		"# \tPush: (ctx, event) => event,",
		"# \tPop: (ctx) => [undefined, false],",
		"# \tLen: (ctx) => 0,",
		"# export const runtimeConfig = hsm.Config({",
		"# \tQueue: priorityQueue,",
		"door_model = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterClockConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.ts", Data: []byte(tsClockConfigSource)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguagePython,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"# Original typescript global global_1 preserved for manual porting:",
		"# const deterministicClock = hsm.Clock({",
		"# \tAfter: (duration) => Promise.resolve(duration),",
		"# \tNewTimer: (duration) => ({ cancel: () => undefined }),",
		"# export const runtimeConfig = hsm.Config({",
		"# \tClock: deterministicClock,",
		"door_model = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterDefaultClockAssignmentIsCommentedForForeignTarget(t *testing.T) {
	cases := []struct {
		name    string
		from    Language
		path    string
		source  string
		needles []string
	}{
		{
			name:   "typescript PascalCase",
			from:   LanguageTS,
			path:   "door.ts",
			source: tsDefaultClockConfigSource,
			needles: []string{
				"# Original typescript global global_1 preserved for manual porting:",
				"# hsm.DefaultClock = hsm.Clock({",
				"# \tAfter: (duration) => Promise.resolve(duration),",
				"# \tNewTimer: (duration) => ({ cancel: () => undefined }),",
			},
		},
		{
			name:   "javascript camelCase",
			from:   LanguageJS,
			path:   "door.js",
			source: tsDefaultClockCamelCaseAliasConfigSource,
			needles: []string{
				"# Original javascript global global_1 preserved for manual porting:",
				"# hsm.defaultClock = hsm.clock({",
				"# \tafter: (duration) => Promise.resolve(duration),",
				"# \tnewTimer: (duration) => ({ cancel: () => undefined }),",
			},
		},
		{
			name:   "dart closure assignment",
			from:   LanguageDart,
			path:   "door.dart",
			source: dartDefaultClockConfigSource,
			needles: []string{
				"# Original dart global global_1 preserved for manual porting:",
				"# configureDefaultClock = () {",
				"#   hsm.defaultClock = hsm.clock(",
				"#     after: (duration) => Future.value(duration),",
				"#     newTimer: (duration) => Timer(duration, () {}),",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: tc.path, Data: []byte(tc.source)}, CompileOptions{
				From:    tc.from,
				To:      LanguagePython,
				Adapter: "none",
			})
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			for _, needle := range append(tc.needles, "door_model = hsm.Define(") {
				if !strings.Contains(text, needle) {
					t.Fatalf("Python output missing %q:\n%s", needle, text)
				}
			}
		})
	}
}

func TestNoAdapterTypeScriptGroupedQueueConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.ts", Data: []byte(tsGroupedQueueConfigSource)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguagePython,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"# Original typescript global global_1 preserved for manual porting:",
		"# export const priorityQueue = hsm.Queue({",
		"# \tPush: (ctx, event) => event,",
		"# Original typescript global global_2 preserved for manual porting:",
		"# export const runtimeConfig = hsm.Config({",
		"# \tQueue: priorityQueue,",
		"door_model = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterTypeScriptCamelCaseQueueConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.ts", Data: []byte(tsCamelCaseQueueConfigSource)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguagePython,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"# Original typescript global global_1 preserved for manual porting:",
		"# export const priorityQueue = hsm.queue({",
		"# \tpush: (ctx, event) => event,",
		"# Original typescript global global_2 preserved for manual porting:",
		"# export const runtimeConfig = hsm.config({",
		"# \tqueue: priorityQueue,",
		"door_model = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterJavaScriptGroupedQueueConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.js", Data: []byte(jsGroupedQueueConfigSource)}, CompileOptions{
		From:    LanguageJS,
		To:      LanguagePython,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"# Original javascript global global_1 preserved for manual porting:",
		"# export const priorityQueue = hsm.Queue({",
		"# \tPush: (ctx, event) => event,",
		"# Original javascript global global_2 preserved for manual porting:",
		"# export const runtimeConfig = hsm.Config({",
		"# \tQueue: priorityQueue,",
		"door_model = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterJavaScriptCamelCaseQueueConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.js", Data: []byte(jsCamelCaseQueueConfigSource)}, CompileOptions{
		From:    LanguageJS,
		To:      LanguagePython,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"# Original javascript global global_1 preserved for manual porting:",
		"# export const priorityQueue = hsm.queue({",
		"# \tpush: (ctx, event) => event,",
		"# Original javascript global global_2 preserved for manual porting:",
		"# export const runtimeConfig = hsm.config({",
		"# \tqueue: priorityQueue,",
		"door_model = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterJavaScriptRenamedQueueConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.js", Data: []byte(jsRenamedImportQueueConfigSource)}, CompileOptions{
		From:    LanguageJS,
		To:      LanguagePython,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"# Original javascript global global_1 preserved for manual porting:",
		"# export const priorityQueue = HQueue({",
		"# \tPush: (ctx, event) => event,",
		"# Original javascript global global_2 preserved for manual porting:",
		"# export const runtimeConfig = HConfig({",
		"# \tQueue: priorityQueue,",
		"door_model = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterDartGroupedQueueConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartGroupedQueueConfigSource)}, CompileOptions{
		From:    LanguageDart,
		To:      LanguagePython,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"# Original dart global global_1 preserved for manual porting:",
		"# final priorityQueue = queue(",
		"#   push: (ctx, event) => event,",
		"# Original dart global global_2 preserved for manual porting:",
		"# final runtimeConfig = config([",
		"#   queue(priorityQueue),",
		"door_model = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterDartPascalCaseQueueConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartPascalCaseQueueConfigSource)}, CompileOptions{
		From:    LanguageDart,
		To:      LanguagePython,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"# Original dart global global_1 preserved for manual porting:",
		"# final priorityQueue = Queue(",
		"#   Push: (ctx, event) => event,",
		"# Original dart global global_2 preserved for manual porting:",
		"# final runtimeConfig = Config([",
		"#   Queue(priorityQueue),",
		"door_model = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterDartAliasedQueueConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartAliasedQueueConfigSource)}, CompileOptions{
		From:    LanguageDart,
		To:      LanguagePython,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"# Original dart global global_1 preserved for manual porting:",
		"# final priorityQueue = hsm.queue(",
		"#   push: (ctx, event) => event,",
		"# Original dart global global_2 preserved for manual porting:",
		"# final runtimeConfig = hsm.config([",
		"#   hsm.queue(priorityQueue),",
		"door_model = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterDartGroupedClockConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartGroupedClockConfigSource)}, CompileOptions{
		From:    LanguageDart,
		To:      LanguagePython,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"# Original dart global global_1 preserved for manual porting:",
		"# final deterministicClock = Clock(",
		"#   after: (duration) => Future.value(duration),",
		"#   newTimer: (duration) => Timer(duration, () {}),",
		"# Original dart global global_2 preserved for manual porting:",
		"# final runtimeConfig = Config([",
		"#   Clock(deterministicClock),",
		"door_model = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterDartAliasedClockConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.dart", Data: []byte(dartAliasedClockConfigSource)}, CompileOptions{
		From:    LanguageDart,
		To:      LanguagePython,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"# Original dart global global_1 preserved for manual porting:",
		"# final deterministicClock = hsm.Clock(",
		"#   after: (duration) => Future.value(duration),",
		"#   newTimer: (duration) => Timer(duration, () {}),",
		"# Original dart global global_2 preserved for manual porting:",
		"# final runtimeConfig = hsm.Config([",
		"#   hsm.Clock(deterministicClock),",
		"door_model = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterPythonGroupedQueueConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.py", Data: []byte(pythonGroupedQueueConfigSource)}, CompileOptions{
		From:    LanguagePython,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original python global global_1 preserved for manual porting:",
		"// priority_queue = hsm.Queue(",
		"//     Push=lambda ctx, event: event,",
		"// Original python global global_2 preserved for manual porting:",
		"// runtime_config = hsm.Config(",
		"//     Queue=priority_queue,",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterPythonGroupedClockConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.py", Data: []byte(pythonGroupedClockConfigSource)}, CompileOptions{
		From:    LanguagePython,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original python global global_1 preserved for manual porting:",
		"// deterministic_clock = hsm.Clock(",
		"//     After=lambda duration: duration,",
		"// Original python global global_2 preserved for manual porting:",
		"// runtime_config = hsm.Config(",
		"//     Clock=deterministic_clock,",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterPythonDefaultClockConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.py", Data: []byte(pythonDefaultClockConfigSource)}, CompileOptions{
		From:    LanguagePython,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original python global global_1 preserved for manual porting:",
		"// hsm.default_clock = hsm.clock(",
		"//     after=lambda duration: duration,",
		"//     new_timer=lambda duration: None,",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterPythonSnakeCaseQueueConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.py", Data: []byte(pythonSnakeCaseQueueConfigSource)}, CompileOptions{
		From:    LanguagePython,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original python global global_1 preserved for manual porting:",
		"// priority_queue = hsm.queue(",
		"//     push=lambda ctx, event: event,",
		"// Original python global global_2 preserved for manual porting:",
		"// runtime_config = hsm.config(",
		"//     queue=priority_queue,",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterCSharpGroupedQueueConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.cs", Data: []byte(csharpGroupedQueueConfigSource)}, CompileOptions{
		From:    LanguageCSharp,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original csharp global global_1 preserved for manual porting:",
		"// private static readonly object PriorityQueue = Hsm.Queue(",
		"//         Push: (ctx, @event) => @event,",
		"// Original csharp global global_2 preserved for manual porting:",
		"// private static readonly object RuntimeConfig = Hsm.Config(",
		"//         Hsm.Queue(PriorityQueue)",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterCSharpAliasedQueueConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.cs", Data: []byte(csharpAliasedQueueConfigSource)}, CompileOptions{
		From:    LanguageCSharp,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original csharp global global_1 preserved for manual porting:",
		"// private static readonly object PriorityQueue = Hsm.Queue(",
		"//         Push: (ctx, @event) => @event,",
		"// Original csharp global global_2 preserved for manual porting:",
		"// private static readonly object RuntimeConfig = Hsm.Config(",
		"//         Hsm.ID(\"door-1\"),",
		"//         Hsm.Queue(PriorityQueue)",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "H.Queue") || strings.Contains(text, "H.Config") {
		t.Fatalf("aliased C# runtime globals were not normalized in no-adapter comments:\n%s", text)
	}
}

func TestNoAdapterCSharpGroupedClockConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.cs", Data: []byte(csharpGroupedClockConfigSource)}, CompileOptions{
		From:    LanguageCSharp,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original csharp global global_1 preserved for manual porting:",
		"// private static readonly object DeterministicClock = Hsm.Clock(",
		"//         Hsm.After(duration => duration),",
		"// Original csharp global global_2 preserved for manual porting:",
		"// private static readonly object RuntimeConfig = Hsm.Config(",
		"//         Hsm.Clock(DeterministicClock)",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterCSharpAliasedClockConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.cs", Data: []byte(csharpAliasedClockConfigSource)}, CompileOptions{
		From:    LanguageCSharp,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original csharp global global_1 preserved for manual porting:",
		"// private static readonly object DeterministicClock = Hsm.Clock(",
		"//         Hsm.After(duration => duration),",
		"// Original csharp global global_2 preserved for manual porting:",
		"// private static readonly object RuntimeConfig = Hsm.Config(",
		"//         Hsm.ID(\"door-1\"),",
		"//         Hsm.Clock(DeterministicClock)",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "H.Clock") || strings.Contains(text, "H.Config") {
		t.Fatalf("aliased C# runtime clock globals were not normalized in no-adapter comments:\n%s", text)
	}
}

func TestNoAdapterCSharpDefaultClockConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.cs", Data: []byte(csharpDefaultClockConfigSource)}, CompileOptions{
		From:    LanguageCSharp,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original csharp global global_1 preserved for manual porting:",
		"// static Sample()",
		"//         Hsm.DefaultClock = Hsm.Clock(",
		"//             Hsm.After(duration => duration),",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterCSharpAliasedDefaultClockConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.cs", Data: []byte(csharpAliasedDefaultClockConfigSource)}, CompileOptions{
		From:    LanguageCSharp,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original csharp global global_1 preserved for manual porting:",
		"// static Sample()",
		"//         Hsm.DefaultClock = Hsm.Clock(",
		"//             Hsm.After(duration => duration),",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "H.DefaultClock") || strings.Contains(text, "H.Clock") {
		t.Fatalf("aliased C# default clock global was not normalized in no-adapter comments:\n%s", text)
	}
}

func TestNoAdapterJavaGroupedQueueConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "Door.java", Data: []byte(javaGroupedQueueConfigSource)}, CompileOptions{
		From:    LanguageJava,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original java global global_1 preserved for manual porting:",
		"// static final Object PriorityQueue = Hsm.Queue(",
		"//         Hsm.Push((ctx, event) -> event),",
		"// Original java global global_2 preserved for manual porting:",
		"// static final Object RuntimeConfig = Hsm.Config(",
		"//         Hsm.Queue(PriorityQueue)",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterJavaStaticImportQueueConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "Door.java", Data: []byte(javaStaticImportQueueConfigSource)}, CompileOptions{
		From:    LanguageJava,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original java global global_1 preserved for manual porting:",
		"// static final Object PriorityQueue = Queue(",
		"//         Push((ctx, event) -> event),",
		"// Original java global global_2 preserved for manual porting:",
		"// static final Object RuntimeConfig = Config(",
		"//         Queue(PriorityQueue)",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterJavaGroupedClockConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "Door.java", Data: []byte(javaGroupedClockConfigSource)}, CompileOptions{
		From:    LanguageJava,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original java global global_1 preserved for manual porting:",
		"// static final Object DeterministicClock = Hsm.Clock(",
		"//         Hsm.After(duration -> duration),",
		"// Original java global global_2 preserved for manual porting:",
		"// static final Object RuntimeConfig = Hsm.Config(",
		"//         Hsm.Clock(DeterministicClock)",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterJavaStaticImportClockConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "Door.java", Data: []byte(javaStaticImportClockConfigSource)}, CompileOptions{
		From:    LanguageJava,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original java global global_1 preserved for manual porting:",
		"// static final Object DeterministicClock = Clock(",
		"//         After(duration -> duration),",
		"// Original java global global_2 preserved for manual porting:",
		"// static final Object RuntimeConfig = Config(",
		"//         Clock(DeterministicClock)",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterJavaDefaultClockConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "Door.java", Data: []byte(javaDefaultClockConfigSource)}, CompileOptions{
		From:    LanguageJava,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original java global global_1 preserved for manual porting:",
		"// static {",
		"//         Hsm.DefaultClock = Hsm.Clock(",
		"//             Hsm.After(duration -> duration),",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterJavaStaticImportDefaultClockConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "Door.java", Data: []byte(javaStaticImportDefaultClockConfigSource)}, CompileOptions{
		From:    LanguageJava,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original java global global_1 preserved for manual porting:",
		"// static {",
		"//         DefaultClock = Clock(",
		"//             After(duration -> duration),",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterCPPGroupedQueueConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.cpp", Data: []byte(cppGroupedQueueConfigSource)}, CompileOptions{
		From:    LanguageCPP,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original cpp global global_1 preserved for manual porting:",
		"// static auto PriorityQueue = hsm::queue(",
		"//   hsm::push([](auto& ctx, auto& event) { return event; }),",
		"// Original cpp global global_2 preserved for manual porting:",
		"// static auto RuntimeConfig = hsm::config(",
		"//   hsm::queue(PriorityQueue)",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterCPPGroupedClockConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.cpp", Data: []byte(cppGroupedClockConfigSource)}, CompileOptions{
		From:    LanguageCPP,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original cpp global global_1 preserved for manual porting:",
		"// static auto DeterministicClock = hsm::clock(",
		"//   hsm::after([](auto duration) { return duration; }),",
		"// Original cpp global global_2 preserved for manual porting:",
		"// static auto RuntimeConfig = hsm::config(",
		"//   hsm::clock(DeterministicClock)",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterZigQueueConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.zig", Data: []byte(zigQueueConfigSource)}, CompileOptions{
		From:    LanguageZig,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original zig global global_1 preserved for manual porting:",
		"// pub const priority_queue = hsm.queue(.{",
		"//     .push = queue_push,",
		"// Original zig global global_2 preserved for manual porting:",
		"// pub const runtime_config = hsm.config(.{",
		"//     hsm.queue(priority_queue),",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterZigClockConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.zig", Data: []byte(zigClockConfigSource)}, CompileOptions{
		From:    LanguageZig,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original zig global global_1 preserved for manual porting:",
		"// pub const deterministic_clock = hsm.clock(.{",
		"//     .after = clock_after,",
		"// Original zig global global_2 preserved for manual porting:",
		"// pub const runtime_config = hsm.config(.{",
		"//     hsm.clock(deterministic_clock),",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterRustQueueConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.rs", Data: []byte(rustQueueConfigSource)}, CompileOptions{
		From:    LanguageRust,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original rust global global_1 preserved for manual porting:",
		"// const PRIORITY_QUEUE: Queue = queue!(",
		"//     push!(queue_push),",
		"// Original rust global global_2 preserved for manual porting:",
		"// const RUNTIME_CONFIG: Config = config!(",
		"//     queue!(PRIORITY_QUEUE)",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterRustClockConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.rs", Data: []byte(rustClockConfigSource)}, CompileOptions{
		From:    LanguageRust,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original rust global global_1 preserved for manual porting:",
		"// const DETERMINISTIC_CLOCK: Clock = clock!(",
		"//     after!(clock_after),",
		"// Original rust global global_2 preserved for manual porting:",
		"// const RUNTIME_CONFIG: Config = config!(",
		"//     clock!(DETERMINISTIC_CLOCK)",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterJSONIRQueueConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(jsonIRQueueConfigSource)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguagePython,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"# Original typescript global global_1 preserved for manual porting:",
		"# const priorityQueue = hsm.Queue({ Push: (ctx, event) => event, Pop: (ctx) => [undefined, false], Len: (ctx) => 0 });",
		"# Original typescript global global_2 preserved for manual porting:",
		"# const runtimeConfig = hsm.Config({ ID: \"door-1\", Queue: priorityQueue });",
		"door_model = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterJSONIRClockConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(jsonIRClockConfigSource)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguagePython,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"# Original typescript global global_1 preserved for manual porting:",
		"# const deterministicClock = hsm.Clock({ After: (duration) => duration, NewTimer: (duration) => undefined });",
		"# Original typescript global global_2 preserved for manual porting:",
		"# const runtimeConfig = hsm.Config({ ID: \"door-1\", Clock: deterministicClock });",
		"door_model = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterJSONIRDefaultClockConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(jsonIRDefaultClockConfigSource)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguagePython,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"# Original typescript global global_1 preserved for manual porting:",
		"# hsm.DefaultClock = hsm.Clock({ After: (duration) => duration, NewTimer: (duration) => undefined });",
		"door_model = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Python output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterGoGroupedQueueConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goGroupedQueueConfigSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original go global global_1 preserved for manual porting:",
		"// var priorityQueue = hsm.Queue(",
		"hsm.Push(func(ctx hsm.Context, event hsm.Event) hsm.DispatchResult {",
		"// Original go global global_2 preserved for manual porting:",
		"// var RuntimeConfig = hsm.Config(hsm.ID(\"door-1\"), hsm.Queue(priorityQueue))",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterGoAliasedQueueConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goAliasedQueueConfigSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original go global global_1 preserved for manual porting:",
		"// var priorityQueue = hsm.Queue(",
		"hsm.Push(func(ctx hsm.Context, event hsm.Event) hsm.DispatchResult {",
		"// Original go global global_2 preserved for manual porting:",
		"// var RuntimeConfig = hsm.Config(hsm.ID(\"door-1\"), hsm.Queue(priorityQueue))",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "sfhsm.") {
		t.Fatalf("aliased Go runtime globals were not normalized in no-adapter comments:\n%s", text)
	}
}

func TestNoAdapterGoGroupedClockConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goGroupedClockConfigSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original go global global_1 preserved for manual porting:",
		"// var deterministicClock = hsm.Clock(",
		"hsm.After(func(duration any) any {",
		"hsm.NewTimer(func(duration any) any {",
		"// Original go global global_2 preserved for manual porting:",
		"// var RuntimeConfig = hsm.Config(hsm.ID(\"door-1\"), hsm.Clock(deterministicClock))",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterGoAliasedClockConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goAliasedClockConfigSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original go global global_1 preserved for manual porting:",
		"// var deterministicClock = hsm.Clock(",
		"hsm.After(func(duration any) any {",
		"hsm.NewTimer(func(duration any) any {",
		"// Original go global global_2 preserved for manual porting:",
		"// var RuntimeConfig = hsm.Config(hsm.ID(\"door-1\"), hsm.Clock(deterministicClock))",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "sfhsm.") {
		t.Fatalf("aliased Go runtime clock globals were not normalized in no-adapter comments:\n%s", text)
	}
}

func TestNoAdapterGoDefaultClockConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDefaultClockConfigSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original go global global_1 preserved for manual porting:",
		"// func init() {",
		"// \thsm.DefaultClock = hsm.Clock(",
		"hsm.After(func(duration any) any { return duration })",
		"hsm.NewTimer(func(duration any) any { return nil })",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
}

func TestNoAdapterGoAliasedDefaultClockConfigIsCommentedForForeignTarget(t *testing.T) {
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goAliasedDefaultClockConfigSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, needle := range []string{
		"// Original go global global_1 preserved for manual porting:",
		"// func init() {",
		"// \thsm.DefaultClock = hsm.Clock(",
		"hsm.After(func(duration any) any { return duration })",
		"hsm.NewTimer(func(duration any) any { return nil })",
		"export const DoorModel = hsm.Define(",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("TypeScript output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "sfhsm.") {
		t.Fatalf("aliased Go default clock global was not normalized in no-adapter comments:\n%s", text)
	}
}

func TestAdapterRequestCarriesDefaultClockAssignmentGlobal(t *testing.T) {
	cases := []struct {
		name        string
		language    Language
		path        string
		source      string
		clockNeedle string
		hookNeedles []string
		target      Language
	}{
		{name: "typescript", language: LanguageTS, path: "door.ts", source: tsDefaultClockConfigSource, clockNeedle: "hsm.DefaultClock = hsm.Clock({", hookNeedles: []string{"After", "NewTimer"}, target: LanguagePython},
		{name: "typescript camelCase alias", language: LanguageTS, path: "door.ts", source: tsDefaultClockCamelCaseAliasConfigSource, clockNeedle: "hsm.defaultClock = hsm.clock({", hookNeedles: []string{"after", "newTimer"}, target: LanguagePython},
		{name: "javascript camelCase alias", language: LanguageJS, path: "door.js", source: tsDefaultClockCamelCaseAliasConfigSource, clockNeedle: "hsm.defaultClock = hsm.clock({", hookNeedles: []string{"after", "newTimer"}, target: LanguagePython},
		{name: "typescript named camelCase import", language: LanguageTS, path: "door.ts", source: tsDefaultClockNamedCamelCaseImportConfigSource, clockNeedle: "defaultClock = clock({", hookNeedles: []string{"after", "newTimer"}, target: LanguagePython},
		{name: "go init assignment", language: LanguageGo, path: "door.go", source: goDefaultClockConfigSource, clockNeedle: "hsm.DefaultClock = hsm.Clock(", hookNeedles: []string{"hsm.After", "hsm.NewTimer"}, target: LanguageTS},
		{name: "go aliased init assignment", language: LanguageGo, path: "door.go", source: goAliasedDefaultClockConfigSource, clockNeedle: "hsm.DefaultClock = hsm.Clock(", hookNeedles: []string{"hsm.After", "hsm.NewTimer"}, target: LanguageTS},
		{name: "dart closure assignment", language: LanguageDart, path: "door.dart", source: dartDefaultClockConfigSource, clockNeedle: "hsm.defaultClock = hsm.clock(", hookNeedles: []string{"after:", "newTimer:"}, target: LanguagePython},
		{name: "python snake_case assignment", language: LanguagePython, path: "door.py", source: pythonDefaultClockConfigSource, clockNeedle: "hsm.default_clock = hsm.clock(", hookNeedles: []string{"after=", "new_timer="}, target: LanguageTS},
		{name: "csharp", language: LanguageCSharp, path: "door.cs", source: csharpDefaultClockConfigSource, clockNeedle: "Hsm.DefaultClock = Hsm.Clock(", hookNeedles: []string{"After", "NewTimer"}, target: LanguageTS},
		{name: "csharp aliased runtime", language: LanguageCSharp, path: "door.cs", source: csharpAliasedDefaultClockConfigSource, clockNeedle: "Hsm.DefaultClock = Hsm.Clock(", hookNeedles: []string{"Hsm.After", "Hsm.NewTimer"}, target: LanguageTS},
		{name: "java", language: LanguageJava, path: "Door.java", source: javaDefaultClockConfigSource, clockNeedle: "Hsm.DefaultClock = Hsm.Clock(", hookNeedles: []string{"After", "NewTimer"}, target: LanguageTS},
		{name: "java static runtime import", language: LanguageJava, path: "Door.java", source: javaStaticImportDefaultClockConfigSource, clockNeedle: "DefaultClock = Clock(", hookNeedles: []string{"After", "NewTimer"}, target: LanguageTS},
		{name: "json-ir", language: LanguageJSONIR, path: "door.hsm.json", source: jsonIRDefaultClockConfigSource, clockNeedle: "hsm.DefaultClock = hsm.Clock", hookNeedles: []string{"After", "NewTimer"}, target: LanguagePython},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &capturingAdapter{}
			compiler := NewCompiler()
			compiler.RegisterAdapter(adapter)

			_, _, err := compiler.Compile(context.Background(), SourceInput{Path: tc.path, Data: []byte(tc.source)}, CompileOptions{
				From:    tc.language,
				To:      tc.target,
				Adapter: adapter.Name(),
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(adapter.request.Globals) != 1 {
				t.Fatalf("adapter globals = %#v, want default clock global", adapter.request.Globals)
			}
			text := adapter.request.Globals[0].Code
			if !strings.Contains(text, tc.clockNeedle) {
				t.Fatalf("adapter request missing default clock runtime context:\n%s", text)
			}
			for _, needle := range tc.hookNeedles {
				if !strings.Contains(text, needle) {
					t.Fatalf("adapter request missing default clock hook %q:\n%s", needle, text)
				}
			}
			if strings.Contains(text, "Define(") || strings.Contains(text, "define(") {
				t.Fatalf("model definition leaked into adapter default clock global:\n%s", text)
			}
			if len(adapter.request.Models) != 1 || adapter.request.Models[0].Name != "Door" {
				t.Fatalf("adapter model context changed by default clock assignment: %#v", adapter.request.Models)
			}
		})
	}
}

func TestAdapterRequestCarriesClockConfigGlobals(t *testing.T) {
	cases := []struct {
		name         string
		language     Language
		path         string
		source       string
		clockNeedle  string
		configNeedle string
		target       Language
	}{
		{name: "typescript", language: LanguageTS, path: "door.ts", source: tsClockConfigSource, clockNeedle: "hsm.Clock", configNeedle: "hsm.Config", target: LanguagePython},
		{name: "typescript renamed imports", language: LanguageTS, path: "door.ts", source: tsRenamedImportClockConfigSource, clockNeedle: "HClock", configNeedle: "HConfig", target: LanguagePython},
		{name: "go grouped", language: LanguageGo, path: "door.go", source: goGroupedClockConfigSource, clockNeedle: "hsm.Clock", configNeedle: "hsm.Config", target: LanguageTS},
		{name: "go aliased runtime import", language: LanguageGo, path: "door.go", source: goAliasedClockConfigSource, clockNeedle: "hsm.Clock", configNeedle: "hsm.Config", target: LanguageTS},
		{name: "dart grouped", language: LanguageDart, path: "door.dart", source: dartGroupedClockConfigSource, clockNeedle: "Clock(", configNeedle: "Config(", target: LanguagePython},
		{name: "dart aliased runtime", language: LanguageDart, path: "door.dart", source: dartAliasedClockConfigSource, clockNeedle: "hsm.Clock", configNeedle: "hsm.Config", target: LanguagePython},
		{name: "python grouped", language: LanguagePython, path: "door.py", source: pythonGroupedClockConfigSource, clockNeedle: "hsm.Clock", configNeedle: "hsm.Config", target: LanguageTS},
		{name: "csharp grouped", language: LanguageCSharp, path: "door.cs", source: csharpGroupedClockConfigSource, clockNeedle: "Hsm.Clock", configNeedle: "Hsm.Config", target: LanguageTS},
		{name: "csharp aliased runtime", language: LanguageCSharp, path: "door.cs", source: csharpAliasedClockConfigSource, clockNeedle: "Hsm.Clock", configNeedle: "Hsm.Config", target: LanguageTS},
		{name: "java grouped", language: LanguageJava, path: "Door.java", source: javaGroupedClockConfigSource, clockNeedle: "Hsm.Clock", configNeedle: "Hsm.Config", target: LanguageTS},
		{name: "java static runtime import", language: LanguageJava, path: "Door.java", source: javaStaticImportClockConfigSource, clockNeedle: "Clock(", configNeedle: "Config(", target: LanguageTS},
		{name: "cpp grouped", language: LanguageCPP, path: "door.cpp", source: cppGroupedClockConfigSource, clockNeedle: "hsm::clock", configNeedle: "hsm::config", target: LanguageTS},
		{name: "zig", language: LanguageZig, path: "door.zig", source: zigClockConfigSource, clockNeedle: "hsm.clock", configNeedle: "hsm.config", target: LanguageTS},
		{name: "rust", language: LanguageRust, path: "door.rs", source: rustClockConfigSource, clockNeedle: "clock!", configNeedle: "config!", target: LanguageTS},
		{name: "rust qualified macros", language: LanguageRust, path: "door.rs", source: rustQualifiedClockConfigSource, clockNeedle: "hsm::clock!", configNeedle: "hsm::config!", target: LanguageTS},
		{name: "json-ir", language: LanguageJSONIR, path: "door.hsm.json", source: jsonIRClockConfigSource, clockNeedle: "hsm.Clock", configNeedle: "hsm.Config", target: LanguagePython},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &capturingAdapter{}
			compiler := NewCompiler()
			compiler.RegisterAdapter(adapter)

			_, _, err := compiler.Compile(context.Background(), SourceInput{Path: tc.path, Data: []byte(tc.source)}, CompileOptions{
				From:    tc.language,
				To:      tc.target,
				Adapter: adapter.Name(),
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(adapter.request.Globals) != 2 {
				t.Fatalf("adapter globals = %#v, want clock and config globals", adapter.request.Globals)
			}
			text := adapter.request.Globals[0].Code + "\n" + adapter.request.Globals[1].Code
			if !strings.Contains(text, tc.clockNeedle) || !strings.Contains(text, tc.configNeedle) {
				t.Fatalf("adapter request missing clock runtime context:\n%s", text)
			}
			if strings.Contains(text, "Define(") || strings.Contains(text, "define(") {
				t.Fatalf("model definition leaked into adapter runtime globals:\n%s", text)
			}
			if len(adapter.request.Models) != 1 || adapter.request.Models[0].Name != "Door" {
				t.Fatalf("adapter model context changed by clock config: %#v", adapter.request.Models)
			}
		})
	}
}

func TestAdapterRequestCarriesQueueConfigGlobals(t *testing.T) {
	cases := []struct {
		name         string
		language     Language
		path         string
		source       string
		queueNeedle  string
		configNeedle string
		target       Language
	}{
		{name: "typescript", language: LanguageTS, path: "door.ts", source: tsQueueConfigSource, queueNeedle: "hsm.Queue", configNeedle: "hsm.Config", target: LanguagePython},
		{name: "typescript grouped", language: LanguageTS, path: "door.ts", source: tsGroupedQueueConfigSource, queueNeedle: "hsm.Queue", configNeedle: "hsm.Config", target: LanguagePython},
		{name: "typescript camelCase grouped", language: LanguageTS, path: "door.ts", source: tsCamelCaseQueueConfigSource, queueNeedle: "hsm.queue", configNeedle: "hsm.config", target: LanguagePython},
		{name: "typescript named imports", language: LanguageTS, path: "door.ts", source: tsNamedImportQueueConfigSource, queueNeedle: "Queue", configNeedle: "Config", target: LanguagePython},
		{name: "typescript renamed imports", language: LanguageTS, path: "door.ts", source: tsRenamedImportQueueConfigSource, queueNeedle: "HQueue", configNeedle: "HConfig", target: LanguagePython},
		{name: "javascript grouped", language: LanguageJS, path: "door.js", source: jsGroupedQueueConfigSource, queueNeedle: "hsm.Queue", configNeedle: "hsm.Config", target: LanguagePython},
		{name: "javascript camelCase grouped", language: LanguageJS, path: "door.js", source: jsCamelCaseQueueConfigSource, queueNeedle: "hsm.queue", configNeedle: "hsm.config", target: LanguagePython},
		{name: "javascript named imports", language: LanguageJS, path: "door.js", source: jsNamedImportQueueConfigSource, queueNeedle: "Queue", configNeedle: "Config", target: LanguagePython},
		{name: "javascript renamed imports", language: LanguageJS, path: "door.js", source: jsRenamedImportQueueConfigSource, queueNeedle: "HQueue", configNeedle: "HConfig", target: LanguagePython},
		{name: "go grouped", language: LanguageGo, path: "door.go", source: goGroupedQueueConfigSource, queueNeedle: "hsm.Queue", configNeedle: "hsm.Config", target: LanguageTS},
		{name: "go aliased runtime import", language: LanguageGo, path: "door.go", source: goAliasedQueueConfigSource, queueNeedle: "hsm.Queue", configNeedle: "hsm.Config", target: LanguageTS},
		{name: "dart grouped", language: LanguageDart, path: "door.dart", source: dartGroupedQueueConfigSource, queueNeedle: "queue(", configNeedle: "config(", target: LanguagePython},
		{name: "dart PascalCase grouped", language: LanguageDart, path: "door.dart", source: dartPascalCaseQueueConfigSource, queueNeedle: "Queue(", configNeedle: "Config(", target: LanguagePython},
		{name: "dart aliased runtime", language: LanguageDart, path: "door.dart", source: dartAliasedQueueConfigSource, queueNeedle: "hsm.queue", configNeedle: "hsm.config", target: LanguagePython},
		{name: "python grouped", language: LanguagePython, path: "door.py", source: pythonGroupedQueueConfigSource, queueNeedle: "hsm.Queue", configNeedle: "hsm.Config", target: LanguageTS},
		{name: "python snake_case grouped", language: LanguagePython, path: "door.py", source: pythonSnakeCaseQueueConfigSource, queueNeedle: "hsm.queue", configNeedle: "hsm.config", target: LanguageTS},
		{name: "python renamed imports", language: LanguagePython, path: "door.py", source: pythonRenamedImportQueueConfigSource, queueNeedle: "HQueue", configNeedle: "HConfig", target: LanguageTS},
		{name: "csharp grouped", language: LanguageCSharp, path: "door.cs", source: csharpGroupedQueueConfigSource, queueNeedle: "Hsm.Queue", configNeedle: "Hsm.Config", target: LanguageTS},
		{name: "csharp aliased runtime", language: LanguageCSharp, path: "door.cs", source: csharpAliasedQueueConfigSource, queueNeedle: "Hsm.Queue", configNeedle: "Hsm.Config", target: LanguageTS},
		{name: "java grouped", language: LanguageJava, path: "Door.java", source: javaGroupedQueueConfigSource, queueNeedle: "Hsm.Queue", configNeedle: "Hsm.Config", target: LanguageTS},
		{name: "java static runtime import", language: LanguageJava, path: "Door.java", source: javaStaticImportQueueConfigSource, queueNeedle: "Queue(", configNeedle: "Config(", target: LanguageTS},
		{name: "cpp grouped", language: LanguageCPP, path: "door.cpp", source: cppGroupedQueueConfigSource, queueNeedle: "queue(", configNeedle: "config(", target: LanguageTS},
		{name: "zig", language: LanguageZig, path: "door.zig", source: zigQueueConfigSource, queueNeedle: "hsm.queue", configNeedle: "hsm.config", target: LanguageTS},
		{name: "rust", language: LanguageRust, path: "door.rs", source: rustQueueConfigSource, queueNeedle: "queue!", configNeedle: "config!", target: LanguageTS},
		{name: "rust qualified macros", language: LanguageRust, path: "door.rs", source: rustQualifiedQueueConfigSource, queueNeedle: "hsm::queue!", configNeedle: "hsm::config!", target: LanguageTS},
		{name: "json-ir", language: LanguageJSONIR, path: "door.hsm.json", source: jsonIRQueueConfigSource, queueNeedle: "hsm.Queue", configNeedle: "hsm.Config", target: LanguagePython},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &capturingAdapter{}
			compiler := NewCompiler()
			compiler.RegisterAdapter(adapter)

			_, _, err := compiler.Compile(context.Background(), SourceInput{Path: tc.path, Data: []byte(tc.source)}, CompileOptions{
				From:    tc.language,
				To:      tc.target,
				Adapter: adapter.Name(),
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(adapter.request.Globals) != 2 {
				t.Fatalf("adapter globals = %#v, want queue and config globals", adapter.request.Globals)
			}
			text := adapter.request.Globals[0].Code + "\n" + adapter.request.Globals[1].Code
			if !strings.Contains(text, tc.queueNeedle) || !strings.Contains(text, tc.configNeedle) {
				t.Fatalf("adapter request missing queue runtime context:\n%s", text)
			}
			lowerText := strings.ToLower(text)
			for _, hook := range []string{"push", "pop", "len"} {
				if !strings.Contains(lowerText, hook) {
					t.Fatalf("adapter request missing queue hook %q:\n%s", hook, text)
				}
			}
			if strings.Contains(text, "Define(") || strings.Contains(text, "define(") {
				t.Fatalf("model definition leaked into adapter runtime globals:\n%s", text)
			}
			if len(adapter.request.Models) != 1 || adapter.request.Models[0].Name != "Door" {
				t.Fatalf("adapter model context changed by queue config: %#v", adapter.request.Models)
			}
		})
	}
}
