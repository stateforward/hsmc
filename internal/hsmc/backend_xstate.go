package hsmc

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"
)

type XStateBackend struct{}

func NewXStateBackend() XStateBackend {
	return XStateBackend{}
}

func (XStateBackend) Language() Language { return LanguageXState }

type xstateModelContext struct {
	model         Model
	pathsByID     map[string]string
	behaviorNames map[string]string
	everyActors   map[string]xstateEveryActor
}

type xstateEveryActor struct {
	Name       string
	DelayExpr  string
	EventType  string
	Transition Transition
}

type xstateDelay struct {
	Name string
	Expr string
}

func (backend XStateBackend) Emit(_ context.Context, program *Program) ([]byte, error) {
	var out bytes.Buffer
	backend.emitImports(&out, program)
	out.WriteString("type XStateContext = Record<string, unknown>;\n")
	out.WriteString("type XStateEvent = { type: string } & Record<string, unknown>;\n")
	out.WriteString("type XStateActorInput = { context: XStateContext; event: XStateEvent };\n")
	out.WriteString("type XStateBehaviorArgs = {\n")
	out.WriteString("\tcontext: XStateContext;\n")
	out.WriteString("\tevent: XStateEvent;\n")
	out.WriteString("\tself?: unknown;\n")
	out.WriteString("\tsendBack?: (event: XStateEvent) => void;\n")
	out.WriteString("\treceive?: (listener: (event: XStateEvent) => void) => void;\n")
	out.WriteString("\tinput?: XStateActorInput;\n")
	out.WriteString("};\n\n")
	for _, event := range program.Events {
		backend.emitEvent(&out, event)
	}
	for _, global := range program.Globals {
		if global.Language != "" && global.Language != LanguageXState {
			writeCommentedGlobalSource(&out, "", "//", global, LanguageXState)
			continue
		}
		code := strings.TrimSpace(global.Code)
		if code == "" {
			continue
		}
		out.WriteString(code)
		if !strings.HasSuffix(code, "\n") {
			out.WriteString("\n")
		}
		out.WriteString("\n")
	}
	behaviorNames := map[string]string{}
	for _, behavior := range program.Behaviors {
		name := lowerCamel(exportName(behavior.ID))
		behaviorNames[behavior.ID] = name
		if err := backend.emitBehavior(&out, name, behavior); err != nil {
			return nil, err
		}
	}
	for _, model := range program.Models {
		ctx := xstateModelContext{
			model:         model,
			pathsByID:     xstatePathsByID(model),
			behaviorNames: behaviorNames,
			everyActors:   map[string]xstateEveryActor{},
		}
		backend.collectEveryActors(&ctx, model.States)
		backend.emitModel(&out, &ctx)
	}
	return out.Bytes(), nil
}

func (backend XStateBackend) emitImports(out *bytes.Buffer, program *Program) {
	out.WriteString(`import { fromCallback, setup } from "xstate";` + "\n")
	for _, imp := range collectRenderableImports(program, LanguageXState, compilerOwnedImports(LanguageXState)...) {
		fmt.Fprintln(out, formatESImport(imp))
	}
	out.WriteString("\n")
}

func (backend XStateBackend) emitEvent(out *bytes.Buffer, event EventDecl) {
	fmt.Fprintf(out, "export const %s = { type: %s } as const;\n\n", eventSymbolFor(LanguageXState, event.Symbol), quoteTS(event.Name))
}

func (backend XStateBackend) emitBehavior(out *bytes.Buffer, name string, behavior Behavior) error {
	returnType := xstateReturnType(behavior)
	if (behavior.SourceLanguage == LanguageXState || behavior.TargetLanguage == LanguageXState) && strings.TrimSpace(behavior.Body) == "" {
		code := strings.TrimSpace(behavior.Code)
		if code != "" {
			if declaredName, ok := esTopLevelDeclarationName(code); ok {
				if declaredName != name {
					return fmt.Errorf("xstate behavior full code declares %q, want compiler-owned name %q", declaredName, name)
				}
				out.WriteString(ensureTrailingSemicolonForESDeclaration(code))
				out.WriteString("\n\n")
				return nil
			}
			fmt.Fprintf(out, "const %s = %s;\n\n", name, code)
			return nil
		}
	}
	fmt.Fprintf(out, "function %s(args: XStateBehaviorArgs): %s {\n", name, returnType)
	body := ""
	if behavior.TargetLanguage == LanguageXState || behavior.SourceLanguage == LanguageXState {
		body = strings.TrimSpace(behavior.Body)
	}
	if body == "" {
		writeCommentedBehaviorSource(out, "\t", "//", behavior)
		if ret := xstateUntranslatedReturn(behavior); ret != "" {
			fmt.Fprintf(out, "\t%s\n", ret)
		}
	} else {
		out.WriteString(indentTS(body))
		out.WriteString("\n")
		if behavior.Kind == BehaviorGuard && !strings.Contains(body, "return") {
			out.WriteString("\treturn false;\n")
		}
	}
	out.WriteString("}\n\n")
	return nil
}

func xstateReturnType(behavior Behavior) string {
	if behavior.TargetABI != nil && behavior.TargetABI.ReturnType != "" {
		return behavior.TargetABI.ReturnType
	}
	switch behavior.Kind {
	case BehaviorGuard:
		return "boolean"
	case BehaviorTrigger:
		switch behavior.TriggerKind {
		case TriggerAfter, TriggerEvery:
			return "number | Promise<number>"
		case TriggerAt:
			return "Date | Promise<Date>"
		case TriggerWhen:
			return "boolean | Promise<boolean>"
		}
	}
	return "void"
}

func xstateUntranslatedReturn(behavior Behavior) string {
	switch behavior.Kind {
	case BehaviorGuard:
		return "return false;"
	case BehaviorTrigger:
		switch behavior.TriggerKind {
		case TriggerAt:
			return "return new Date(0);"
		case TriggerAfter, TriggerEvery:
			return "return 0;"
		case TriggerWhen:
			return "return false;"
		}
	}
	return ""
}

func (backend XStateBackend) emitModel(out *bytes.Buffer, ctx *xstateModelContext) {
	fmt.Fprintf(out, "export const %s = setup({\n", modelSymbolFor(LanguageXState, ctx.model.Name))
	backend.emitSetupSection(out, "\t", "actions", xstateActionNames(ctx))
	backend.emitSetupSection(out, "\t", "guards", xstateGuardNames(ctx))
	backend.emitDelaySetup(out, "\t", ctx)
	backend.emitActorSetup(out, "\t", ctx)
	out.WriteString("}).createMachine({\n")
	fmt.Fprintf(out, "\tid: %s,\n", quoteTS(ctx.model.Name))
	backend.emitContext(out, "\t", ctx.model.Attributes)
	if ctx.model.Initializer != nil && ctx.model.Initializer.Target != "" {
		fmt.Fprintf(out, "\tinitial: %s,\n", quoteTS(xstateInitialTarget(ctx, ctx.model.ID, ctx.model.Initializer.Target)))
	}
	backend.emitBehaviorArray(out, "\t", "entry", xstateRefs(ctx.model.InitializerEffects(), ctx.behaviorNames))
	if len(ctx.model.States) > 0 {
		out.WriteString("\tstates: {\n")
		for _, state := range ctx.model.States {
			backend.emitState(out, "\t\t", ctx, state)
		}
		out.WriteString("\t},\n")
	}
	backend.emitTransitions(out, "\t", ctx, ctx.model.Transitions, ctx.model.ID)
	out.WriteString("});\n")
}

func (model Model) InitializerEffects() []BehaviorRef {
	if model.Initializer == nil {
		return nil
	}
	return model.Initializer.Effects
}

func (backend XStateBackend) emitContext(out *bytes.Buffer, indent string, attributes []Attribute) {
	out.WriteString(indent + "context: {\n")
	for _, attribute := range attributes {
		value := "undefined"
		if defaultValue, ok := esAttributeTargetDefault(attribute, LanguageXState); ok {
			value = defaultValue
		}
		fmt.Fprintf(out, "%s\t%s: %s,\n", indent, strconv.Quote(attribute.Name), value)
	}
	out.WriteString(indent + "},\n")
}

func (backend XStateBackend) emitSetupSection(out *bytes.Buffer, indent string, name string, symbols []string) {
	if len(symbols) == 0 {
		return
	}
	fmt.Fprintf(out, "%s%s: {\n", indent, name)
	for _, symbol := range symbols {
		fmt.Fprintf(out, "%s\t%s,\n", indent, symbol)
	}
	fmt.Fprintf(out, "%s},\n", indent)
}

func (backend XStateBackend) emitDelaySetup(out *bytes.Buffer, indent string, ctx *xstateModelContext) {
	delays := xstateDelays(ctx)
	if len(delays) == 0 {
		return
	}
	fmt.Fprintf(out, "%sdelays: {\n", indent)
	for _, delay := range delays {
		if delay.Expr == delay.Name {
			fmt.Fprintf(out, "%s\t%s,\n", indent, delay.Name)
			continue
		}
		fmt.Fprintf(out, "%s\t%s: %s,\n", indent, delay.Name, delay.Expr)
	}
	fmt.Fprintf(out, "%s},\n", indent)
}

func (backend XStateBackend) emitActorSetup(out *bytes.Buffer, indent string, ctx *xstateModelContext) {
	actors := xstateActivityNames(ctx)
	for _, actor := range ctx.everyActors {
		actors = append(actors, actor.Name)
	}
	if len(actors) == 0 {
		return
	}
	fmt.Fprintf(out, "%sactors: {\n", indent)
	for _, actor := range actors {
		fmt.Fprintf(out, "%s\t%s: fromCallback<XStateEvent, XStateActorInput>(({ input, sendBack, receive }) => {\n", indent, actor)
		fmt.Fprintf(out, "%s\t\tconst args = { context: input.context, event: input.event, sendBack, receive, input };\n", indent)
		if every, ok := xstateEveryActorByName(ctx, actor); ok {
			fmt.Fprintf(out, "%s\t\tlet active = true;\n", indent)
			fmt.Fprintf(out, "%s\t\tlet cleanup: (() => void) | undefined;\n", indent)
			fmt.Fprintf(out, "%s\t\tPromise.resolve(%s(args)).then((delay) => {\n", indent, every.DelayExpr)
			fmt.Fprintf(out, "%s\t\t\tif (!active) return;\n", indent)
			fmt.Fprintf(out, "%s\t\t\tconst interval = Number(delay);\n", indent)
			fmt.Fprintf(out, "%s\t\t\tif (!(interval > 0)) return;\n", indent)
			fmt.Fprintf(out, "%s\t\t\tconst handle = setInterval(() => sendBack({ type: %s }), interval);\n", indent, quoteTS(every.EventType))
			fmt.Fprintf(out, "%s\t\t\tcleanup = () => clearInterval(handle);\n", indent)
			fmt.Fprintf(out, "%s\t\t});\n", indent)
			fmt.Fprintf(out, "%s\t\treturn () => { active = false; cleanup?.(); };\n", indent)
		} else {
			fmt.Fprintf(out, "%s\t\tvoid %s(args);\n", indent, actor)
		}
		fmt.Fprintf(out, "%s\t}),\n", indent)
	}
	fmt.Fprintf(out, "%s},\n", indent)
}

func xstateEveryActorByName(ctx *xstateModelContext, name string) (xstateEveryActor, bool) {
	for _, actor := range ctx.everyActors {
		if actor.Name == name {
			return actor, true
		}
	}
	return xstateEveryActor{}, false
}

func (backend XStateBackend) emitState(out *bytes.Buffer, indent string, ctx *xstateModelContext, state State) {
	fmt.Fprintf(out, "%s%s: {\n", indent, quoteTS(state.Name))
	fmt.Fprintf(out, "%sid: %s,\n", indent+"\t", quoteTS(xstateIDForPath(ctx.pathsByID[state.ID])))
	if state.Kind == StateKindFinal {
		fmt.Fprintf(out, "%stype: \"final\",\n", indent+"\t")
	}
	if state.Initializer != nil && state.Initializer.Target != "" {
		fmt.Fprintf(out, "%sinitial: %s,\n", indent+"\t", quoteTS(xstateInitialTarget(ctx, state.ID, state.Initializer.Target)))
	}
	backend.emitBehaviorArray(out, indent+"\t", "entry", xstateRefsOfKind(state.Behaviors, BehaviorEntry, ctx.behaviorNames))
	backend.emitBehaviorArray(out, indent+"\t", "exit", xstateRefsOfKind(state.Behaviors, BehaviorExit, ctx.behaviorNames))
	backend.emitInvokes(out, indent+"\t", ctx, state)
	if len(state.States) > 0 {
		fmt.Fprintf(out, "%sstates: {\n", indent+"\t")
		for _, child := range state.States {
			backend.emitState(out, indent+"\t\t", ctx, child)
		}
		fmt.Fprintf(out, "%s},\n", indent+"\t")
	}
	backend.emitTransitions(out, indent+"\t", ctx, state.Transitions, state.ID)
	fmt.Fprintf(out, "%s},\n", indent)
}

func (backend XStateBackend) emitInvokes(out *bytes.Buffer, indent string, ctx *xstateModelContext, state State) {
	var invokes []string
	for _, ref := range state.Behaviors {
		if ref.Kind != BehaviorActivity {
			continue
		}
		if name := xstateBehaviorName(ref, ctx.behaviorNames); name != "" {
			invokes = append(invokes, name)
		}
	}
	for _, actor := range ctx.everyActors {
		if actor.Transition.OwnerID == state.ID {
			invokes = append(invokes, actor.Name)
		}
	}
	if len(invokes) == 0 {
		return
	}
	fmt.Fprintf(out, "%sinvoke: [\n", indent)
	for _, invoke := range invokes {
		fmt.Fprintf(out, "%s\t{ src: %s, input: ({ context, event }) => ({ context, event }) },\n", indent, quoteTS(invoke))
	}
	fmt.Fprintf(out, "%s],\n", indent)
}

func (backend XStateBackend) emitTransitions(out *bytes.Buffer, indent string, ctx *xstateModelContext, transitions []Transition, ownerID string) {
	ons := map[string][]Transition{}
	var always []Transition
	var afters []Transition
	for _, transition := range transitions {
		if transition.Trigger == nil {
			always = append(always, transition)
			continue
		}
		switch transition.Trigger.Kind {
		case TriggerOn:
			for _, eventName := range xstateTriggerEventNames(*transition.Trigger) {
				ons[eventName] = append(ons[eventName], transition)
			}
		case TriggerOnSet, TriggerOnCall:
			ons[transition.Trigger.Value] = append(ons[transition.Trigger.Value], transition)
		case TriggerAfter, TriggerAt:
			afters = append(afters, transition)
		case TriggerEvery:
			if actor, ok := ctx.everyActors[transition.ID]; ok {
				ons[actor.EventType] = append(ons[actor.EventType], transition)
			}
		case TriggerWhen:
			if transition.Trigger.IsString && transition.Trigger.Value != "" && transition.Trigger.Expr == nil {
				ons[transition.Trigger.Value] = append(ons[transition.Trigger.Value], transition)
			} else {
				always = append(always, transition)
			}
		}
	}
	if len(always) > 0 {
		fmt.Fprintf(out, "%salways: [\n", indent)
		for _, transition := range always {
			backend.emitTransitionObject(out, indent+"\t", ctx, transition, ownerID, true)
		}
		fmt.Fprintf(out, "%s],\n", indent)
	}
	if len(afters) > 0 {
		fmt.Fprintf(out, "%safter: {\n", indent)
		for _, transition := range afters {
			delay := xstateDelayName(ctx, transition)
			fmt.Fprintf(out, "%s\t%s: ", indent, quoteTS(delay))
			backend.emitInlineTransitionObject(out, ctx, transition, ownerID, false)
			out.WriteString(",\n")
		}
		fmt.Fprintf(out, "%s},\n", indent)
	}
	if len(ons) > 0 {
		fmt.Fprintf(out, "%son: {\n", indent)
		for _, eventName := range sortedMapKeys(ons) {
			transitionsForEvent := ons[eventName]
			fmt.Fprintf(out, "%s\t%s: ", indent, quoteTS(eventName))
			if len(transitionsForEvent) == 1 {
				backend.emitInlineTransitionObject(out, ctx, transitionsForEvent[0], ownerID, false)
				out.WriteString(",\n")
				continue
			}
			out.WriteString("[\n")
			for _, transition := range transitionsForEvent {
				backend.emitTransitionObject(out, indent+"\t\t", ctx, transition, ownerID, false)
			}
			fmt.Fprintf(out, "%s\t],\n", indent)
		}
		fmt.Fprintf(out, "%s},\n", indent)
	}
}

func (backend XStateBackend) emitTransitionObject(out *bytes.Buffer, indent string, ctx *xstateModelContext, transition Transition, ownerID string, always bool) {
	out.WriteString(indent)
	backend.emitInlineTransitionObject(out, ctx, transition, ownerID, always)
	out.WriteString(",\n")
}

func (backend XStateBackend) emitInlineTransitionObject(out *bytes.Buffer, ctx *xstateModelContext, transition Transition, ownerID string, always bool) {
	out.WriteString("{")
	wrote := false
	if transition.Target != "" {
		fmt.Fprintf(out, " target: %s", quoteTS(xstateTarget(ctx, ownerID, transition.Target)))
		wrote = true
	}
	if transition.Guard != nil {
		if wrote {
			out.WriteString(",")
		}
		fmt.Fprintf(out, " guard: %s", xstateTypedObjectLiteral(xstateBehaviorName(*transition.Guard, ctx.behaviorNames)))
		wrote = true
	} else if always && transition.Trigger != nil && transition.Trigger.Kind == TriggerWhen && transition.Trigger.Expr != nil {
		if name := ctx.behaviorNames[transition.Trigger.Expr.ID]; name != "" {
			if wrote {
				out.WriteString(",")
			}
			fmt.Fprintf(out, " guard: %s", xstateTypedObjectLiteral(name))
			wrote = true
		}
	}
	actions := xstateRefs(transition.Effects, ctx.behaviorNames)
	if len(actions) > 0 {
		if wrote {
			out.WriteString(",")
		}
		fmt.Fprintf(out, " actions: %s", xstateActionArrayLiteral(actions))
	}
	out.WriteString(" }")
}

func (backend XStateBackend) emitBehaviorArray(out *bytes.Buffer, indent string, field string, names []string) {
	if len(names) == 0 {
		return
	}
	fmt.Fprintf(out, "%s%s: %s,\n", indent, field, xstateActionArrayLiteral(names))
}

func (backend XStateBackend) collectEveryActors(ctx *xstateModelContext, states []State) {
	for _, state := range states {
		for _, transition := range state.Transitions {
			if transition.Trigger == nil || transition.Trigger.Kind != TriggerEvery {
				continue
			}
			name := lowerCamel(exportName(transition.ID + "_every"))
			ctx.everyActors[transition.ID] = xstateEveryActor{
				Name:       name,
				DelayExpr:  xstateDelayExpression(ctx, transition),
				EventType:  "__hsmc.every." + transition.ID,
				Transition: transition,
			}
		}
		backend.collectEveryActors(ctx, state.States)
	}
}

func xstatePathsByID(model Model) map[string]string {
	paths := map[string]string{model.ID: "/" + model.Name}
	var walk func(parent string, states []State)
	walk = func(parent string, states []State) {
		for _, state := range states {
			current := path.Join(parent, state.Name)
			paths[state.ID] = current
			walk(current, state.States)
		}
	}
	walk(paths[model.ID], model.States)
	return paths
}

func xstateIDForPath(value string) string {
	return strings.TrimPrefix(strings.ReplaceAll(value, "/", "."), ".")
}

func xstateInitialTarget(ctx *xstateModelContext, ownerID string, target string) string {
	resolved := xstateResolvePath(ctx, ownerID, target)
	ownerPath := ctx.pathsByID[ownerID]
	prefix := strings.TrimSuffix(ownerPath, "/") + "/"
	if strings.HasPrefix(resolved, prefix) {
		relative := strings.TrimPrefix(resolved, prefix)
		if !strings.Contains(relative, "/") {
			return relative
		}
	}
	return "#" + xstateIDForPath(resolved)
}

func xstateTarget(ctx *xstateModelContext, ownerID string, target string) string {
	if target == "" {
		return ""
	}
	return "#" + xstateIDForPath(xstateResolvePath(ctx, ownerID, target))
}

func xstateResolvePath(ctx *xstateModelContext, ownerID string, target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ctx.pathsByID[ownerID]
	}
	if strings.HasPrefix(target, "/") {
		if strings.HasPrefix(target, "/"+ctx.model.Name) {
			return path.Clean(target)
		}
		return path.Join("/", ctx.model.Name, strings.TrimPrefix(target, "/"))
	}
	base := ctx.pathsByID[ownerID]
	return path.Clean(path.Join(base, target))
}

func xstateActionNames(ctx *xstateModelContext) []string {
	seen := map[string]bool{}
	var names []string
	for _, ref := range xstateAllBehaviorRefs(ctx.model) {
		if ref.Kind == BehaviorEntry || ref.Kind == BehaviorExit || ref.Kind == BehaviorEffect {
			if name := xstateBehaviorName(ref, ctx.behaviorNames); name != "" && !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	return names
}

func xstateGuardNames(ctx *xstateModelContext) []string {
	seen := map[string]bool{}
	var names []string
	add := func(ref BehaviorRef) {
		if name := xstateBehaviorName(ref, ctx.behaviorNames); name != "" && !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	var visitTransitions func([]Transition)
	visitTransitions = func(transitions []Transition) {
		for _, transition := range transitions {
			if transition.Guard != nil {
				add(*transition.Guard)
			}
			if transition.Trigger != nil && transition.Trigger.Kind == TriggerWhen && transition.Trigger.Expr != nil {
				add(*transition.Trigger.Expr)
			}
		}
	}
	var walk func([]State)
	walk = func(states []State) {
		for _, state := range states {
			visitTransitions(state.Transitions)
			walk(state.States)
		}
	}
	visitTransitions(ctx.model.Transitions)
	walk(ctx.model.States)
	return names
}

func xstateDelays(ctx *xstateModelContext) []xstateDelay {
	seen := map[string]bool{}
	var delays []xstateDelay
	var visitTransitions func([]Transition)
	visitTransitions = func(transitions []Transition) {
		for _, transition := range transitions {
			if transition.Trigger == nil {
				continue
			}
			if transition.Trigger.Kind == TriggerAfter || transition.Trigger.Kind == TriggerAt {
				name := xstateDelayName(ctx, transition)
				if !seen[name] {
					seen[name] = true
					delays = append(delays, xstateDelay{Name: name, Expr: xstateDelayExpression(ctx, transition)})
				}
			}
		}
	}
	var walk func([]State)
	walk = func(states []State) {
		for _, state := range states {
			visitTransitions(state.Transitions)
			walk(state.States)
		}
	}
	visitTransitions(ctx.model.Transitions)
	walk(ctx.model.States)
	return delays
}

func xstateActivityNames(ctx *xstateModelContext) []string {
	seen := map[string]bool{}
	var names []string
	for _, ref := range xstateAllBehaviorRefs(ctx.model) {
		if ref.Kind == BehaviorActivity {
			if name := xstateBehaviorName(ref, ctx.behaviorNames); name != "" && !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	return names
}

func xstateDelayName(ctx *xstateModelContext, transition Transition) string {
	if transition.Trigger == nil {
		return lowerCamel(exportName(transition.ID + "_delay"))
	}
	if transition.Trigger.Expr != nil && transition.Trigger.Kind != TriggerAt {
		if name := ctx.behaviorNames[transition.Trigger.Expr.ID]; name != "" {
			return name
		}
	}
	return lowerCamel(exportName(transition.ID + "_delay"))
}

func xstateDelayExpression(ctx *xstateModelContext, transition Transition) string {
	if transition.Trigger == nil {
		return "() => 0"
	}
	if transition.Trigger.Expr != nil {
		if name := ctx.behaviorNames[transition.Trigger.Expr.ID]; name != "" {
			if transition.Trigger.Kind == TriggerAt {
				return fmt.Sprintf("(args: XStateBehaviorArgs) => Promise.resolve(%s(args)).then((deadline) => Math.max(0, Number(deadline) - Date.now()))", name)
			}
			return name
		}
	}
	if transition.Trigger.IsString {
		return fmt.Sprintf("({ context }: XStateBehaviorArgs) => Number(context?.[%s] ?? 0)", quoteTS(transition.Trigger.Value))
	}
	if transition.Trigger.Kind == TriggerAt {
		return fmt.Sprintf("(args: XStateBehaviorArgs) => Math.max(0, Number(%s(args)) - Date.now())", transition.Trigger.Value)
	}
	return fmt.Sprintf("(_args: XStateBehaviorArgs) => Number(%s)", transition.Trigger.Value)
}

func xstateRefsOfKind(refs []BehaviorRef, kind BehaviorKind, behaviorNames map[string]string) []string {
	var names []string
	for _, ref := range refs {
		if ref.Kind == kind {
			if name := xstateBehaviorName(ref, behaviorNames); name != "" {
				names = append(names, name)
			}
		}
	}
	return names
}

func xstateRefs(refs []BehaviorRef, behaviorNames map[string]string) []string {
	var names []string
	for _, ref := range refs {
		if name := xstateBehaviorName(ref, behaviorNames); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func xstateBehaviorName(ref BehaviorRef, behaviorNames map[string]string) string {
	if ref.Operation != "" {
		return lowerCamel(exportName(ref.Operation))
	}
	if name := behaviorNames[ref.ID]; name != "" {
		return name
	}
	return ref.ID
}

func xstateAllBehaviorRefs(model Model) []BehaviorRef {
	var refs []BehaviorRef
	if model.Initializer != nil {
		refs = append(refs, model.Initializer.Effects...)
	}
	for _, operation := range model.Operations {
		if operation.Behavior != nil {
			refs = append(refs, *operation.Behavior)
		}
	}
	var walk func([]State)
	walk = func(states []State) {
		for _, state := range states {
			refs = append(refs, state.Behaviors...)
			if state.Initializer != nil {
				refs = append(refs, state.Initializer.Effects...)
			}
			for _, transition := range state.Transitions {
				if transition.Guard != nil {
					refs = append(refs, *transition.Guard)
				}
				if transition.Trigger != nil && transition.Trigger.Expr != nil {
					refs = append(refs, *transition.Trigger.Expr)
				}
				refs = append(refs, transition.Effects...)
			}
			walk(state.States)
		}
	}
	walk(model.States)
	for _, transition := range model.Transitions {
		if transition.Guard != nil {
			refs = append(refs, *transition.Guard)
		}
		if transition.Trigger != nil && transition.Trigger.Expr != nil {
			refs = append(refs, *transition.Trigger.Expr)
		}
		refs = append(refs, transition.Effects...)
	}
	return refs
}

func xstateTriggerEventNames(trigger Trigger) []string {
	values := trigger.Values
	if len(values) == 0 {
		values = []TriggerValue{{Value: trigger.Value, IsString: trigger.IsString, Language: trigger.Language}}
	}
	var names []string
	for _, value := range values {
		if value.IsString || value.EventSymbol == "" {
			names = append(names, value.Value)
			continue
		}
		names = append(names, value.Value)
	}
	return names
}

func xstateArrayLiteral(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, quoteTS(value))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func xstateActionArrayLiteral(values []string) string {
	actions := make([]string, 0, len(values))
	for _, value := range values {
		actions = append(actions, xstateTypedObjectLiteral(value))
	}
	return "[" + strings.Join(actions, ", ") + "]"
}

func xstateTypedObjectLiteral(value string) string {
	return "{ type: " + quoteTS(value) + " }"
}

func sortedMapKeys[T any](items map[string]T) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sortStrings(keys)
	return keys
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
