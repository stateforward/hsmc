package hsmc

func AssignTargetABI(program *Program, target Language) {
	if program == nil {
		return
	}
	for index := range program.Behaviors {
		program.Behaviors[index].TargetABI = BehaviorABIForTrigger(target, program.Behaviors[index].Kind, program.Behaviors[index].TriggerKind)
	}
}

func BehaviorABIFor(language Language, kind BehaviorKind) *BehaviorABI {
	return BehaviorABIForTrigger(language, kind, "")
}

func BehaviorABIForTrigger(language Language, kind BehaviorKind, triggerKind TriggerKind) *BehaviorABI {
	switch language {
	case LanguageCSharp:
		return csharpBehaviorABI(kind, triggerKind)
	case LanguageCPP:
		return cppBehaviorABI(kind, triggerKind)
	case LanguageDart:
		return dartBehaviorABI(kind, triggerKind)
	case LanguageGo:
		return goBehaviorABI(kind, triggerKind)
	case LanguageJava:
		return javaBehaviorABI(kind, triggerKind)
	case LanguageJS:
		return jsBehaviorABI(kind, triggerKind)
	case LanguageTS:
		return tsBehaviorABI(kind, triggerKind)
	case LanguageXState:
		return xstateBehaviorABI(kind, triggerKind)
	case LanguagePython:
		return pythonBehaviorABI(kind, triggerKind)
	case LanguageRust:
		return rustBehaviorABI(kind, triggerKind)
	case LanguageZig:
		return zigBehaviorABI(kind, triggerKind)
	default:
		return &BehaviorABI{Language: language}
	}
}

func javaBehaviorABI(kind BehaviorKind, triggerKind TriggerKind) *BehaviorABI {
	returnType := "void"
	if kind == BehaviorGuard {
		returnType = "boolean"
	} else if kind == BehaviorTrigger {
		switch triggerKind {
		case TriggerAfter, TriggerEvery:
			returnType = "java.time.Duration"
		case TriggerAt:
			returnType = "java.time.Instant"
		case TriggerWhen:
			returnType = "boolean"
		default:
			returnType = "Object"
		}
	}
	return &BehaviorABI{
		Language:   LanguageJava,
		Signature:  returnType + " (Context ctx, Instance instance, Event event)",
		ReturnType: returnType,
		Parameters: []ABIParameter{
			{Name: "ctx", Type: "Context"},
			{Name: "instance", Type: "Instance"},
			{Name: "event", Type: "Event"},
		},
	}
}

func cppBehaviorABI(kind BehaviorKind, triggerKind TriggerKind) *BehaviorABI {
	returnType := "void"
	if kind == BehaviorGuard {
		returnType = "bool"
	} else if kind == BehaviorTrigger {
		switch triggerKind {
		case TriggerAfter, TriggerEvery:
			returnType = "std::chrono::milliseconds"
		case TriggerAt:
			returnType = "std::chrono::system_clock::time_point"
		case TriggerWhen:
			returnType = "bool"
		default:
			returnType = "void"
		}
	}
	if kind == BehaviorOperation {
		return &BehaviorABI{
			Language:   LanguageCPP,
			Signature:  returnType + " ()",
			ReturnType: returnType,
		}
	}
	return &BehaviorABI{
		Language:   LanguageCPP,
		Signature:  returnType + " (auto& signal, auto& instance, const auto& event)",
		ReturnType: returnType,
		Parameters: []ABIParameter{
			{Name: "signal", Type: "auto&"},
			{Name: "instance", Type: "auto&"},
			{Name: "event", Type: "const auto&"},
		},
	}
}

func dartBehaviorABI(kind BehaviorKind, triggerKind TriggerKind) *BehaviorABI {
	returnType := "FutureOr<void>"
	if kind == BehaviorGuard {
		returnType = "bool"
	} else if kind == BehaviorTrigger {
		switch triggerKind {
		case TriggerAfter, TriggerEvery:
			returnType = "FutureOr<Duration>"
		case TriggerAt:
			returnType = "FutureOr<DateTime>"
		case TriggerWhen:
			returnType = "bool"
		default:
			returnType = "dynamic"
		}
	}
	return &BehaviorABI{
		Language:   LanguageDart,
		Signature:  returnType + " Function(Context ctx, Instance instance, Event event)",
		ReturnType: returnType,
		Parameters: []ABIParameter{
			{Name: "ctx", Type: "Context"},
			{Name: "instance", Type: "Instance"},
			{Name: "event", Type: "Event"},
		},
	}
}

func csharpBehaviorABI(kind BehaviorKind, triggerKind TriggerKind) *BehaviorABI {
	returnType := "void"
	parameters := []ABIParameter{
		{Name: "ctx", Type: "Context"},
		{Name: "instance", Type: "Instance"},
		{Name: "@event", Type: "Event"},
	}
	if kind == BehaviorGuard {
		returnType = "bool"
	} else if kind == BehaviorTrigger {
		switch triggerKind {
		case TriggerAfter, TriggerEvery:
			returnType = "System.TimeSpan"
		case TriggerAt:
			returnType = "System.DateTimeOffset"
		case TriggerWhen:
			returnType = "System.Threading.Tasks.Task"
			parameters = append(parameters, ABIParameter{Name: "cancellationToken", Type: "System.Threading.CancellationToken"})
		default:
			returnType = "object?"
		}
	}
	signature := "static " + returnType + " (Context ctx, Instance instance, Event @event"
	if kind == BehaviorTrigger && triggerKind == TriggerWhen {
		signature += ", System.Threading.CancellationToken cancellationToken"
	}
	signature += ")"
	return &BehaviorABI{
		Language:   LanguageCSharp,
		Signature:  signature,
		ReturnType: returnType,
		Parameters: parameters,
	}
}

func goBehaviorABI(kind BehaviorKind, triggerKind TriggerKind) *BehaviorABI {
	returnType := ""
	if kind == BehaviorGuard {
		returnType = "bool"
	} else if kind == BehaviorTrigger {
		switch triggerKind {
		case TriggerAfter, TriggerEvery:
			returnType = "time.Duration"
		case TriggerAt:
			returnType = "time.Time"
		case TriggerWhen:
			returnType = "<-chan struct{}"
		}
	}
	signature := "func(ctx context.Context, instance hsm.Instance, event hsm.Event)"
	if returnType != "" {
		signature += " " + returnType
	}
	return &BehaviorABI{
		Language:   LanguageGo,
		Signature:  signature,
		ReturnType: returnType,
		Parameters: []ABIParameter{
			{Name: "ctx", Type: "context.Context"},
			{Name: "instance", Type: "hsm.Instance"},
			{Name: "event", Type: "hsm.Event"},
		},
	}
}

func tsBehaviorABI(kind BehaviorKind, triggerKind TriggerKind) *BehaviorABI {
	returnType := "void"
	if kind == BehaviorGuard {
		returnType = "boolean"
	} else if kind == BehaviorTrigger {
		switch triggerKind {
		case TriggerAfter, TriggerEvery:
			returnType = "number | Date | Promise<number> | Promise<Date>"
		case TriggerAt:
			returnType = "Date | Promise<Date>"
		case TriggerWhen:
			returnType = "unknown"
		default:
			returnType = "unknown"
		}
	}
	return &BehaviorABI{
		Language:   LanguageTS,
		Signature:  "function(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): " + returnType,
		ReturnType: returnType,
		Parameters: []ABIParameter{
			{Name: "ctx", Type: "hsm.Context"},
			{Name: "instance", Type: "InstanceType<typeof hsm.Instance>"},
			{Name: "event", Type: "hsm.Event"},
		},
	}
}

func xstateBehaviorABI(kind BehaviorKind, triggerKind TriggerKind) *BehaviorABI {
	returnType := "void"
	if kind == BehaviorGuard {
		returnType = "boolean"
	} else if kind == BehaviorTrigger {
		switch triggerKind {
		case TriggerAfter, TriggerEvery:
			returnType = "number | Promise<number>"
		case TriggerAt:
			returnType = "Date | Promise<Date>"
		case TriggerWhen:
			returnType = "boolean | Promise<boolean>"
		default:
			returnType = "unknown"
		}
	}
	return &BehaviorABI{
		Language:   LanguageXState,
		Signature:  "function(args: XStateBehaviorArgs): " + returnType,
		ReturnType: returnType,
		Parameters: []ABIParameter{
			{Name: "args", Type: "XStateBehaviorArgs"},
		},
	}
}

func jsBehaviorABI(kind BehaviorKind, triggerKind TriggerKind) *BehaviorABI {
	returnType := "void"
	if kind == BehaviorGuard {
		returnType = "boolean"
	} else if kind == BehaviorTrigger {
		switch triggerKind {
		case TriggerAfter, TriggerEvery:
			returnType = "number"
		case TriggerAt:
			returnType = "Date"
		case TriggerWhen:
			returnType = "unknown"
		default:
			returnType = "unknown"
		}
	}
	return &BehaviorABI{
		Language:   LanguageJS,
		Signature:  "function(ctx, instance, event)",
		ReturnType: returnType,
		Parameters: []ABIParameter{
			{Name: "ctx", Type: "hsm.Context"},
			{Name: "instance", Type: "hsm.Instance"},
			{Name: "event", Type: "hsm.Event"},
		},
	}
}

func pythonBehaviorABI(kind BehaviorKind, triggerKind TriggerKind) *BehaviorABI {
	returnType := "None"
	if kind == BehaviorGuard {
		returnType = "bool"
	} else if kind == BehaviorTrigger {
		switch triggerKind {
		case TriggerAfter, TriggerEvery:
			returnType = "float"
		case TriggerAt:
			returnType = "datetime.datetime"
		case TriggerWhen:
			returnType = "typing.Any"
		default:
			returnType = "typing.Any"
		}
	}
	return &BehaviorABI{
		Language:   LanguagePython,
		Signature:  "async def(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event) -> " + returnType,
		ReturnType: returnType,
		Parameters: []ABIParameter{
			{Name: "ctx", Type: "hsm.Context"},
			{Name: "instance", Type: "hsm.Instance"},
			{Name: "event", Type: "hsm.Event"},
		},
	}
}

func rustBehaviorABI(kind BehaviorKind, triggerKind TriggerKind) *BehaviorABI {
	returnType := "()"
	if kind == BehaviorGuard {
		returnType = "bool"
	} else if kind == BehaviorEntry || kind == BehaviorExit || kind == BehaviorActivity || kind == BehaviorEffect || kind == BehaviorOperation {
		returnType = "Pin<Box<dyn Future<Output = ()> + Send>>"
	} else if kind == BehaviorTrigger {
		switch triggerKind {
		case TriggerAfter, TriggerEvery:
			returnType = "Duration"
		case TriggerAt:
			returnType = "std::time::SystemTime"
		case TriggerWhen:
			returnType = "bool"
		}
	}
	instanceType := "&mut HsmcInstance"
	if kind == BehaviorGuard || kind == BehaviorTrigger {
		instanceType = "&HsmcInstance"
	}
	signature := "fn(ctx: &hsm::Context, instance: " + instanceType + ", event: &hsm::Event)"
	if returnType != "()" {
		signature += " -> " + returnType
	}
	return &BehaviorABI{
		Language:   LanguageRust,
		Signature:  signature,
		ReturnType: returnType,
		Parameters: []ABIParameter{
			{Name: "ctx", Type: "&hsm::Context"},
			{Name: "instance", Type: instanceType},
			{Name: "event", Type: "&hsm::Event"},
		},
	}
}

func zigBehaviorABI(kind BehaviorKind, triggerKind TriggerKind) *BehaviorABI {
	returnType := "void"
	if kind == BehaviorGuard {
		returnType = "bool"
	} else if kind == BehaviorTrigger {
		switch triggerKind {
		case TriggerAfter, TriggerEvery, TriggerAt:
			returnType = "u64"
		case TriggerWhen:
			returnType = "bool"
		default:
			returnType = "void"
		}
	}
	return &BehaviorABI{
		Language:   LanguageZig,
		Signature:  "fn(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) " + returnType,
		ReturnType: returnType,
		Parameters: []ABIParameter{
			{Name: "ctx", Type: "*hsm.Context"},
			{Name: "inst", Type: "*hsm.Instance"},
			{Name: "event", Type: "hsm.Event"},
		},
	}
}
