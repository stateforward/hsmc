package hsmc

import (
	"context"
	"errors"
	"fmt"
	"go/token"
	"path"
	"sort"
	"strings"
)

type Frontend interface {
	Language() Language
	Parse(ctx context.Context, input SourceInput) (*Program, error)
}

type Backend interface {
	Language() Language
	Emit(ctx context.Context, program *Program) ([]byte, error)
}

type BehaviorAdapter interface {
	Name() string
	Translate(ctx context.Context, request AdapterRequest) (*BehaviorPatch, error)
}

type SourceInput struct {
	Path string
	Data []byte
}

type AdapterRequest struct {
	SourceLanguage Language    `json:"source_language"`
	TargetLanguage Language    `json:"target_language"`
	PackageName    string      `json:"package_name,omitempty"`
	Imports        []Import    `json:"imports,omitempty"`
	Globals        []CodeBlock `json:"globals,omitempty"`
	Events         []EventDecl `json:"events,omitempty"`
	Models         []ModelView `json:"models,omitempty"`
	Behaviors      []Behavior  `json:"behaviors,omitempty"`
}

func NewAdapterProtocol(request AdapterRequest) AdapterProtocol {
	request = cloneAdapterRequest(request)
	return AdapterProtocol{
		Version: "hsmc.adapter.v1",
		Instruction: strings.Join([]string{
			"The compiler translates HSM structure; use provided imports as source context and translate only globals and behavior implementation code for the target implementation language.",
			"Do not attempt to compile or rewrite arbitrary behavior semantics through model edits; behavior and global body translation is the adapter's only semantic-code responsibility.",
			"Return only a JSON BehaviorPatch object, either as plain JSON or a fenced json code block.",
			"Do not create, remove, rename, reorder, or modify models, events, attributes, operations, states, initial transitions, transitions, triggers, defers, behavior IDs, owner IDs, behavior kinds, inline markers, signatures, ABIs, or source languages.",
			"DSL runtime/helper globals such as Config, Queue, Clock, MakeGroup, MakeKind, IsKind, and TakeSnapshot are translation context only and must not be represented as model edits.",
			"Top-level imports in the request are source context; return target-language imports only when translated target globals or behaviors require program-level imports.",
			"Imports may be adjusted only for target-language behavior/global code and must not include compiler-owned runtime imports, collide with compiler-owned runtime bindings, bind names listed in guidance.compiler_owned_symbols, or bind names declared by returned target-language globals.",
			"Globals and behavior bodies/code may be edited only by their existing IDs; helper globals must use adapter_global_ IDs with only letters, digits, and underscores after the prefix.",
			"Translated globals and adapter helper globals must not declare names listed in guidance.compiler_owned_symbols or guidance.compiler_owned_bindings.",
			"Use behaviors[].body for translated statements inside the compiler-owned behavior signature; use behaviors[].code only for a single complete target-language callable with the compiler-owned behavior name and no extra top-level declarations.",
			"Omit behavior/global patches you cannot translate; do not return target imports for untranslated foreign code.",
			"The compiler will preserve untranslated behavior source as target-language comments inside generated target behavior bodies, and untranslated globals as target-language comments near top-level output.",
			"Follow the response field allowlists exactly; unknown fields and null array/scalar fields are rejected.",
		}, " "),
		Guidance: adapterGuidance(request),
		Response: adapterResponseContract(),
		AllowedEdits: AllowedEdits{
			ImportPaths:     importPaths(request.Imports),
			GlobalIDs:       globalIDs(request.Globals),
			BehaviorIDs:     behaviorIDs(request.Behaviors),
			GlobalImports:   codeBlockImportRegions(request.Globals),
			BehaviorImports: behaviorImportRegions(request.Behaviors),
		},
		Request: request,
	}
}

func adapterResponseContract() AdapterResponse {
	return AdapterResponse{
		OutputFormat:           "Return only a JSON object, or a single fenced json code block containing that object.",
		TopLevelFields:         []string{"imports", "globals", "behaviors"},
		ImportFields:           []string{"path", "alias", "default", "specifiers", "hidden", "static", "deferred", "type_only", "language"},
		ImportSpecifierFields:  []string{"name", "alias", "type_only"},
		GlobalFields:           []string{"id", "code", "imports"},
		BehaviorFields:         []string{"id", "body", "code", "imports"},
		NullPolicy:             "Omit fields instead of returning null. Null arrays and null scalar fields are rejected.",
		ScalarPolicy:           "String identity and language fields must not contain surrounding whitespace. Behavior body is for translated statements inside the compiler-owned signature; behavior code is for a single complete target-language callable implementation using the compiler-owned name and no extra top-level declarations. Code/body fields may be omitted or empty for import-only target-language patches, but whitespace-only code/body fields are rejected.",
		UnknownFieldPolicy:     "Unknown top-level or nested fields are rejected; do not return models, events, attributes, operations, states, initial transitions, transitions, triggers, defers, ranges, local_names, owner_id, kind, inline, source_language, target_language, signature, or target_abi.",
		ExplicitEmptyListUsage: "Use an explicit empty imports array only to clear target-language imports for that program/global/behavior region. Non-empty program imports are accepted only with translated target global or behavior code in the same patch. Target imports must not bind compiler-owned symbols listed in guidance.compiler_owned_symbols, compiler-owned runtime bindings listed in guidance.compiler_owned_bindings, or names declared by returned target-language globals. Omit a global or behavior entirely when you cannot translate it; omitted regions remain compiler-owned and are emitted as commented source references when needed. Import-only global/behavior patches are accepted only for code that is already in the target language.",
	}
}

type Compiler struct {
	frontends map[Language]Frontend
	backends  map[Language]Backend
	adapters  map[string]BehaviorAdapter
}

func NewCompiler() *Compiler {
	compiler := &Compiler{
		frontends: map[Language]Frontend{},
		backends:  map[Language]Backend{},
		adapters:  map[string]BehaviorAdapter{},
	}
	compiler.RegisterFrontend(NewCPPFrontend())
	compiler.RegisterFrontend(NewCSharpFrontend())
	compiler.RegisterFrontend(NewDartFrontend())
	compiler.RegisterFrontend(NewGoFrontend())
	compiler.RegisterFrontend(NewJavaFrontend())
	compiler.RegisterFrontend(NewJavaScriptFrontend())
	compiler.RegisterFrontend(NewPythonFrontend())
	compiler.RegisterFrontend(NewRustFrontend())
	compiler.RegisterFrontend(NewTypeScriptFrontend())
	compiler.RegisterFrontend(NewZigFrontend())
	compiler.RegisterFrontend(NewJSONIRFrontend())
	compiler.RegisterBackend(NewCSharpBackend())
	compiler.RegisterBackend(NewCPPBackend())
	compiler.RegisterBackend(NewDartBackend())
	compiler.RegisterBackend(NewGoBackend())
	compiler.RegisterBackend(NewJavaBackend())
	compiler.RegisterBackend(NewJavaScriptBackend())
	compiler.RegisterBackend(NewJSONIRBackend())
	compiler.RegisterBackend(NewMermaidBackend())
	compiler.RegisterBackend(NewPlantUMLBackend())
	compiler.RegisterBackend(NewPythonBackend())
	compiler.RegisterBackend(NewRustBackend())
	compiler.RegisterBackend(NewTypeScriptBackend())
	compiler.RegisterBackend(NewZigBackend())
	compiler.RegisterAdapter(IdentityAdapter{})
	compiler.RegisterAdapter(CodexAdapter{})
	compiler.RegisterAdapter(CommandAdapter{})
	return compiler
}

func (compiler *Compiler) RegisterFrontend(frontend Frontend) {
	compiler.frontends[frontend.Language()] = frontend
}

func (compiler *Compiler) RegisterBackend(backend Backend) {
	compiler.backends[backend.Language()] = backend
}

func (compiler *Compiler) RegisterAdapter(adapter BehaviorAdapter) {
	compiler.adapters[adapter.Name()] = adapter
}

func (compiler *Compiler) AdapterNames() []string {
	names := make([]string, 0, len(compiler.adapters))
	for name := range compiler.adapters {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (compiler *Compiler) SourceLanguages() []Language {
	languages := make([]Language, 0, len(compiler.frontends))
	for language := range compiler.frontends {
		languages = append(languages, language)
	}
	sortLanguages(languages)
	return languages
}

func (compiler *Compiler) TargetLanguages() []Language {
	languages := make([]Language, 0, len(compiler.backends))
	for language := range compiler.backends {
		languages = append(languages, language)
	}
	sortLanguages(languages)
	return languages
}

type CompileOptions struct {
	From    Language
	To      Language
	Adapter string
}

func (compiler *Compiler) Compile(ctx context.Context, input SourceInput, options CompileOptions) ([]byte, *Program, error) {
	options = normalizeCompileOptions(options)
	backend := compiler.backends[options.To]
	if backend == nil {
		return nil, nil, fmt.Errorf("unsupported target language %q", options.To)
	}
	program, err := compiler.prepareProgram(ctx, input, options)
	if err != nil {
		return nil, nil, err
	}
	if behaviorAdapterRequested(options.Adapter) {
		if !isImplementationLanguage(options.To) {
			return nil, nil, fmt.Errorf("behavior adapter %q cannot target transport language %q", options.Adapter, options.To)
		}
		adapter := compiler.adapters[options.Adapter]
		if adapter == nil {
			return nil, nil, fmt.Errorf("unsupported behavior adapter %q", options.Adapter)
		}
		adapterRequest := buildAdapterRequest(program, options.To)
		patch, err := adapter.Translate(ctx, adapterRequest)
		if err != nil {
			return nil, nil, err
		}
		if err := applyBehaviorPatch(program, adapterRequest, patch); err != nil {
			return nil, nil, err
		}
		if err := Validate(program); err != nil {
			return nil, nil, err
		}
	}
	output, err := backend.Emit(ctx, program)
	if err != nil {
		return nil, nil, err
	}
	return output, program, nil
}

func (compiler *Compiler) BuildAdapterProtocol(ctx context.Context, input SourceInput, options CompileOptions) (*AdapterProtocol, *Program, error) {
	options = normalizeCompileOptions(options)
	if !isImplementationLanguage(options.To) {
		return nil, nil, fmt.Errorf("adapter protocol cannot target transport language %q", options.To)
	}
	program, err := compiler.prepareProgram(ctx, input, options)
	if err != nil {
		return nil, nil, err
	}
	protocol := NewAdapterProtocol(buildAdapterRequest(program, options.To))
	return &protocol, program, nil
}

func (compiler *Compiler) prepareProgram(ctx context.Context, input SourceInput, options CompileOptions) (*Program, error) {
	options = normalizeCompileOptions(options)
	frontend := compiler.frontends[options.From]
	if frontend == nil {
		return nil, fmt.Errorf("unsupported source language %q", options.From)
	}
	if compiler.backends[options.To] == nil {
		return nil, fmt.Errorf("unsupported target language %q", options.To)
	}
	program, err := frontend.Parse(ctx, input)
	if err != nil {
		return nil, err
	}
	if program == nil {
		return nil, errors.New("frontend returned nil program")
	}
	NormalizeSourceContext(program, options.From)
	if options.From != LanguageJSONIR && program.SourceLanguage != options.From {
		return nil, fmt.Errorf("frontend %q returned source_language %q", options.From, program.SourceLanguage)
	}
	AssignMutableImports(program)
	if err := Validate(program); err != nil {
		return nil, err
	}
	AssignTargetABI(program, options.To)
	if err := validateTargetPackageName(program, options.To); err != nil {
		return nil, err
	}
	if err := validateTargetImports(program, options.To); err != nil {
		return nil, err
	}
	if err := validateTargetImportGlobalDeclarationCollisions(program, options.To); err != nil {
		return nil, err
	}
	if err := validateGeneratedTopLevelSymbolUniqueness(program, options.To); err != nil {
		return nil, err
	}
	if err := validateTargetGlobalRuntimeBindingCollisions(program, options.To); err != nil {
		return nil, err
	}
	if err := validateTargetGlobalSymbolCollisions(program, options.To); err != nil {
		return nil, err
	}
	if err := validateTargetGlobalDeclarationUniqueness(program, options.To); err != nil {
		return nil, err
	}
	return program, nil
}

func validateTargetPackageName(program *Program, target Language) error {
	if program == nil || target != LanguageGo {
		return nil
	}
	name := program.PackageName
	if name == "" {
		return nil
	}
	if name != strings.TrimSpace(name) {
		return fmt.Errorf("target go package_name %q has leading or trailing whitespace", name)
	}
	if name == "_" || !token.IsIdentifier(name) || token.Lookup(name).IsKeyword() {
		return fmt.Errorf("target go package_name %q is not a valid Go package identifier", name)
	}
	return nil
}

func normalizeCompileOptions(options CompileOptions) CompileOptions {
	if language, ok := ParseLanguage(string(options.From)); ok {
		options.From = language
	}
	if language, ok := ParseLanguage(string(options.To)); ok {
		options.To = language
	}
	options.Adapter = strings.TrimSpace(options.Adapter)
	return options
}

func behaviorAdapterRequested(name string) bool {
	return name != "" && name != "none"
}

func isImplementationLanguage(language Language) bool {
	return language != "" && language != LanguageJSONIR && language != LanguageMermaid && language != LanguagePlantUML
}

func buildAdapterRequest(program *Program, target Language) AdapterRequest {
	return AdapterRequest{
		SourceLanguage: program.SourceLanguage,
		TargetLanguage: target,
		PackageName:    program.PackageName,
		Imports:        cloneImports(program.Imports),
		Globals:        cloneCodeBlocks(program.Globals),
		Events:         append([]EventDecl(nil), program.Events...),
		Models:         BuildModelViews(program),
		Behaviors:      cloneBehaviors(program.Behaviors),
	}
}

func applyBehaviorPatch(program *Program, request AdapterRequest, patch *BehaviorPatch) error {
	if program == nil {
		return errors.New("cannot apply behavior patch to nil program")
	}
	working := cloneProgram(program)
	if err := applyBehaviorPatchInPlace(working, request, patch); err != nil {
		return err
	}
	*program = *working
	return nil
}

func applyBehaviorPatchInPlace(program *Program, request AdapterRequest, patch *BehaviorPatch) error {
	if patch == nil {
		return nil
	}
	if patch.ImportsSet || patch.Imports != nil {
		imports := cloneImports(patch.Imports)
		if err := normalizePatchImports(imports, request.TargetLanguage, "program"); err != nil {
			return err
		}
		if len(imports) > 0 && !behaviorPatchContainsTranslatedCode(patch) {
			return errors.New("adapter patch changes program imports without translated target global or behavior code")
		}
		program.Imports = mergeProgramImportPatch(program.Imports, imports, request.TargetLanguage)
	}
	if len(patch.Globals) > 0 {
		byID := map[string]CodeBlock{}
		for _, block := range program.Globals {
			byID[block.ID] = block
		}
		editableGlobalIDs := codeBlockIDSet(request.Globals)
		seenPatchGlobals := map[string]bool{}
		var added []CodeBlock
		for _, block := range patch.Globals {
			if strings.TrimSpace(block.ID) == "" {
				return errors.New("adapter patch contains global with empty id")
			}
			if block.ID != strings.TrimSpace(block.ID) {
				return fmt.Errorf("adapter patch global id %q has leading or trailing whitespace", block.ID)
			}
			if seenPatchGlobals[block.ID] {
				return fmt.Errorf("adapter patch contains duplicate global %q", block.ID)
			}
			seenPatchGlobals[block.ID] = true
			original, ok := byID[block.ID]
			if !ok {
				helper, err := normalizeAdapterGlobal(block, request.TargetLanguage)
				if err != nil {
					return err
				}
				added = append(added, helper)
				continue
			}
			if !editableGlobalIDs[block.ID] {
				return fmt.Errorf("adapter patch references uneditable global %q", block.ID)
			}
			if err := validateGlobalPatchImmutableFields(block); err != nil {
				return err
			}
			if hasWhitespaceOnlyCode(block.Code) {
				return fmt.Errorf("adapter patch global %q contains whitespace-only code", block.ID)
			}
			translatedCode := strings.TrimSpace(block.Code) != ""
			if block.Language == "" && translatedCode {
				block.Language = request.TargetLanguage
			} else if block.Language == "" {
				block.Language = original.Language
			} else {
				language, err := normalizePatchLanguage(block.Language, request.TargetLanguage, fmt.Sprintf("global %q", block.ID), "language")
				if err != nil {
					return err
				}
				block.Language = language
			}
			if translatedCode && block.Language != request.TargetLanguage {
				return fmt.Errorf("adapter patch changes global %q language to %q, want %q", block.ID, block.Language, request.TargetLanguage)
			}
			if block.Language != original.Language && block.Language != request.TargetLanguage {
				return fmt.Errorf("adapter patch changes global %q language to %q, want %q", block.ID, block.Language, request.TargetLanguage)
			}
			if block.Language == request.TargetLanguage && !translatedCode && original.Language != request.TargetLanguage {
				return fmt.Errorf("adapter patch changes global %q to target language without target code", block.ID)
			}
			if block.Code == "" {
				block.Code = original.Code
			}
			importsPatched := block.Imports != nil
			if importsPatched && !translatedCode && original.Language != request.TargetLanguage {
				return fmt.Errorf("adapter patch changes global %q imports without target code", block.ID)
			}
			if err := normalizePatchImports(block.Imports, request.TargetLanguage, fmt.Sprintf("global %q", block.ID)); err != nil {
				return err
			}
			if block.Imports == nil {
				if translatedCode && original.Language != request.TargetLanguage {
					block.Imports = nil
				} else {
					block.Imports = cloneImports(original.Imports)
				}
			} else {
				block.Imports = cloneImports(block.Imports)
			}
			block.ID = original.ID
			block.Range = original.Range
			byID[block.ID] = block
		}
		for index, block := range program.Globals {
			if patched, ok := byID[block.ID]; ok {
				program.Globals[index] = patched
			}
		}
		program.Globals = append(program.Globals, added...)
	}
	if len(patch.Behaviors) > 0 {
		byID := map[string]Behavior{}
		for _, behavior := range program.Behaviors {
			byID[behavior.ID] = behavior
		}
		editableBehaviorIDs := behaviorIDSet(request.Behaviors)
		seenPatchBehaviors := map[string]bool{}
		for _, behavior := range patch.Behaviors {
			if strings.TrimSpace(behavior.ID) == "" {
				return errors.New("adapter patch contains behavior with empty id")
			}
			if behavior.ID != strings.TrimSpace(behavior.ID) {
				return fmt.Errorf("adapter patch behavior id %q has leading or trailing whitespace", behavior.ID)
			}
			if seenPatchBehaviors[behavior.ID] {
				return fmt.Errorf("adapter patch contains duplicate behavior %q", behavior.ID)
			}
			seenPatchBehaviors[behavior.ID] = true
			original, ok := byID[behavior.ID]
			if !ok {
				return fmt.Errorf("adapter patch references unknown behavior %q", behavior.ID)
			}
			if !editableBehaviorIDs[behavior.ID] {
				return fmt.Errorf("adapter patch references uneditable behavior %q", behavior.ID)
			}
			if err := validateBehaviorPatchImmutableFields(behavior); err != nil {
				return err
			}
			if hasWhitespaceOnlyCode(behavior.Body) {
				return fmt.Errorf("adapter patch behavior %q contains whitespace-only body", behavior.ID)
			}
			if hasWhitespaceOnlyCode(behavior.Code) {
				return fmt.Errorf("adapter patch behavior %q contains whitespace-only code", behavior.ID)
			}
			if strings.TrimSpace(behavior.Body) != "" && strings.TrimSpace(behavior.Code) != "" {
				return fmt.Errorf("adapter patch behavior %q must set either body or code, not both", behavior.ID)
			}
			behavior.OwnerID = original.OwnerID
			behavior.Kind = original.Kind
			behavior.TriggerKind = original.TriggerKind
			behavior.Inline = original.Inline
			behavior.SourceLanguage = original.SourceLanguage
			behavior.Signature = original.Signature
			behavior.TargetABI = original.TargetABI
			translatedBody := strings.TrimSpace(behavior.Body) != ""
			translatedFullCode := strings.TrimSpace(behavior.Code) != ""
			translatedCode := translatedBody || translatedFullCode
			if translatedBody {
				if err := validateAdapterBehaviorBodyBoundary(request.TargetLanguage, behavior.ID, behavior.Body); err != nil {
					return err
				}
			}
			if translatedFullCode {
				if err := validateAdapterBehaviorFullCodeBoundary(request.TargetLanguage, behavior, behavior.Code); err != nil {
					return err
				}
			}
			alreadyTarget := original.SourceLanguage == request.TargetLanguage || original.TargetLanguage == request.TargetLanguage
			if behavior.TargetLanguage == "" {
				if translatedCode {
					behavior.TargetLanguage = request.TargetLanguage
				} else {
					behavior.TargetLanguage = original.TargetLanguage
				}
			} else {
				language, err := normalizePatchLanguage(behavior.TargetLanguage, request.TargetLanguage, fmt.Sprintf("behavior %q", behavior.ID), "target_language")
				if err != nil {
					return err
				}
				behavior.TargetLanguage = language
				if !translatedCode && !alreadyTarget {
					return fmt.Errorf("adapter patch changes behavior %q to target language without target body or code", behavior.ID)
				}
			}
			if behavior.Body == "" {
				if translatedFullCode {
					behavior.Body = ""
				} else if !translatedCode || alreadyTarget {
					behavior.Body = original.Body
				}
			}
			if behavior.Code == "" {
				if translatedBody {
					behavior.Code = ""
				} else if !translatedCode || alreadyTarget {
					behavior.Code = original.Code
				}
			}
			importsPatched := behavior.Imports != nil
			if importsPatched && !translatedCode && !alreadyTarget {
				return fmt.Errorf("adapter patch changes behavior %q imports without target body or code", behavior.ID)
			}
			if err := normalizePatchImports(behavior.Imports, request.TargetLanguage, fmt.Sprintf("behavior %q", behavior.ID)); err != nil {
				return err
			}
			if behavior.Imports == nil {
				if translatedCode && !alreadyTarget {
					behavior.Imports = nil
				} else {
					behavior.Imports = cloneImports(original.Imports)
				}
			} else {
				behavior.Imports = cloneImports(behavior.Imports)
			}
			behavior.Range = original.Range
			byID[behavior.ID] = behavior
		}
		for index, behavior := range program.Behaviors {
			if patched, ok := byID[behavior.ID]; ok {
				program.Behaviors[index] = patched
			}
		}
	}
	if err := validateTargetImports(program, request.TargetLanguage); err != nil {
		return err
	}
	if err := validateTargetImportGlobalDeclarationCollisions(program, request.TargetLanguage); err != nil {
		return err
	}
	if err := validateAdapterHelperGlobalSymbolCollisions(program, request.TargetLanguage); err != nil {
		return err
	}
	if err := validateTargetGlobalRuntimeBindingCollisions(program, request.TargetLanguage); err != nil {
		return err
	}
	if err := validateTargetGlobalSymbolCollisions(program, request.TargetLanguage); err != nil {
		return err
	}
	if err := validateTargetGlobalDeclarationUniqueness(program, request.TargetLanguage); err != nil {
		return err
	}
	return nil
}

func hasWhitespaceOnlyCode(value string) bool {
	return value != "" && strings.TrimSpace(value) == ""
}

func behaviorPatchContainsTranslatedCode(patch *BehaviorPatch) bool {
	if patch == nil {
		return false
	}
	for _, block := range patch.Globals {
		if strings.TrimSpace(block.Code) != "" {
			return true
		}
	}
	for _, behavior := range patch.Behaviors {
		if strings.TrimSpace(behavior.Body) != "" || strings.TrimSpace(behavior.Code) != "" {
			return true
		}
	}
	return false
}

func validateAdapterHelperGlobalSymbolCollisions(program *Program, target Language) error {
	if program == nil {
		return nil
	}
	compilerOwned := generatedTopLevelSymbols(program, target)
	if len(compilerOwned) == 0 {
		return nil
	}
	for _, global := range program.Globals {
		if global.Language != target || !strings.HasPrefix(global.ID, "adapter_global_") {
			continue
		}
		for _, name := range targetTopLevelDeclarationNames(target, global.Code) {
			if compilerOwned[name] {
				return fmt.Errorf("adapter-created global %q declares %q, which collides with compiler-owned %s symbol", global.ID, name, target)
			}
		}
	}
	return nil
}

func validateTargetGlobalSymbolCollisions(program *Program, target Language) error {
	if program == nil {
		return nil
	}
	compilerOwned := generatedTopLevelSymbols(program, target)
	if len(compilerOwned) == 0 {
		return nil
	}
	for _, global := range program.Globals {
		if global.Language != target {
			continue
		}
		for _, name := range targetTopLevelDeclarationNames(target, global.Code) {
			if compilerOwned[name] {
				return fmt.Errorf("target-language global %q declares %q, which collides with compiler-owned %s symbol", global.ID, name, target)
			}
		}
	}
	return nil
}

func validateTargetGlobalRuntimeBindingCollisions(program *Program, target Language) error {
	if program == nil {
		return nil
	}
	compilerOwned := map[string]bool{}
	for _, binding := range compilerOwnedBindings(target) {
		if binding != "" {
			compilerOwned[binding] = true
		}
	}
	if len(compilerOwned) == 0 {
		return nil
	}
	for _, global := range program.Globals {
		if global.Language != target {
			continue
		}
		for _, name := range targetTopLevelDeclarationNames(target, global.Code) {
			if compilerOwned[name] {
				return fmt.Errorf("target-language global %q declares %q, which collides with compiler-owned %s runtime binding", global.ID, name, target)
			}
		}
	}
	return nil
}

func validateTargetGlobalDeclarationUniqueness(program *Program, target Language) error {
	if program == nil {
		return nil
	}
	seen := map[string]string{}
	for _, global := range program.Globals {
		if global.Language != target {
			continue
		}
		owner := fmt.Sprintf("target-language global %q", global.ID)
		for _, name := range targetTopLevelDeclarationNames(target, global.Code) {
			if name == "" {
				continue
			}
			if previous := seen[name]; previous != "" {
				if previous == owner {
					return fmt.Errorf("%s declares %q more than once", owner, name)
				}
				return fmt.Errorf("%s declares %q, which collides with %s", owner, name, previous)
			}
			seen[name] = owner
		}
	}
	return nil
}

func generatedTopLevelSymbols(program *Program, target Language) map[string]bool {
	symbols := map[string]bool{}
	for _, use := range generatedTopLevelSymbolUses(program, target) {
		if use.Symbol != "" {
			symbols[use.Symbol] = true
		}
	}
	return symbols
}

type generatedTopLevelSymbolUse struct {
	Symbol string
	Owner  string
}

func validateGeneratedTopLevelSymbolUniqueness(program *Program, target Language) error {
	uses := generatedTopLevelSymbolUses(program, target)
	if len(uses) == 0 {
		return nil
	}
	seen := map[string]string{}
	for _, use := range uses {
		if use.Symbol == "" {
			continue
		}
		if !isGeneratedTopLevelSymbolValid(target, use.Symbol) {
			return fmt.Errorf("compiler-owned %s symbol %q for %s is not a valid target identifier", target, use.Symbol, use.Owner)
		}
		if previous := seen[use.Symbol]; previous != "" {
			return fmt.Errorf("compiler-owned %s symbol %q for %s collides with %s", target, use.Symbol, use.Owner, previous)
		}
		seen[use.Symbol] = use.Owner
	}
	return nil
}

func generatedTopLevelSymbolUses(program *Program, target Language) []generatedTopLevelSymbolUse {
	switch target {
	case LanguageCPP, LanguageCSharp, LanguageDart, LanguageGo, LanguageJava, LanguageJS, LanguagePython, LanguageRust, LanguageTS, LanguageZig:
	default:
		return nil
	}
	if program == nil {
		return nil
	}
	var uses []generatedTopLevelSymbolUse
	for _, event := range program.Events {
		uses = append(uses, generatedTopLevelSymbolUse{
			Symbol: eventSymbolFor(target, event.Symbol),
			Owner:  fmt.Sprintf("event %q", event.ID),
		})
		if target == LanguageCPP {
			uses = append(uses, generatedTopLevelSymbolUse{
				Symbol: cppEventTypeName(event.Name),
				Owner:  fmt.Sprintf("event type for event %q", event.ID),
			})
		}
	}
	if target == LanguageCPP {
		for _, eventName := range cppDeferredEventNames(program.Models) {
			uses = append(uses, generatedTopLevelSymbolUse{
				Symbol: cppEventTypeName(eventName),
				Owner:  fmt.Sprintf("deferred event type %q", eventName),
			})
		}
	}
	for _, behavior := range program.Behaviors {
		uses = append(uses, generatedTopLevelSymbolUse{
			Symbol: behaviorSymbolFor(target, behavior.ID),
			Owner:  fmt.Sprintf("behavior %q", behavior.ID),
		})
	}
	for _, model := range program.Models {
		uses = append(uses, generatedTopLevelSymbolUse{
			Symbol: modelSymbolFor(target, model.Name),
			Owner:  fmt.Sprintf("model %q", model.ID),
		})
	}
	for _, symbol := range generatedScaffoldSymbols(program, target) {
		uses = append(uses, generatedTopLevelSymbolUse{
			Symbol: symbol,
			Owner:  "compiler scaffold",
		})
	}
	return uses
}

func isGeneratedTopLevelSymbolValid(target Language, symbol string) bool {
	if !isSimpleIdentifier(symbol) {
		return false
	}
	switch target {
	case LanguageCPP:
		return !cppReservedIdentifier(symbol)
	case LanguageDart:
		return !dartReservedIdentifier(symbol)
	case LanguageJS, LanguageTS:
		return !esReservedIdentifier(symbol)
	case LanguagePython:
		return !pythonReservedIdentifier(symbol)
	case LanguageRust:
		return !rustReservedIdentifier(symbol)
	case LanguageZig:
		return !zigReservedIdentifier(symbol)
	default:
		return true
	}
}

func cppReservedIdentifier(symbol string) bool {
	return stringInSlice([]string{
		"alignas", "alignof", "and", "and_eq", "asm", "atomic_cancel", "atomic_commit", "atomic_noexcept",
		"auto", "bitand", "bitor", "bool", "break", "case", "catch", "char", "char8_t", "char16_t", "char32_t",
		"class", "compl", "concept", "const", "consteval", "constexpr", "constinit", "const_cast", "continue",
		"co_await", "co_return", "co_yield", "decltype", "default", "delete", "do", "double", "dynamic_cast",
		"else", "enum", "explicit", "export", "extern", "false", "float", "for", "friend", "goto", "if",
		"inline", "int", "long", "mutable", "namespace", "new", "noexcept", "not", "not_eq", "nullptr",
		"operator", "or", "or_eq", "private", "protected", "public", "reflexpr", "register", "reinterpret_cast",
		"requires", "return", "short", "signed", "sizeof", "static", "static_assert", "static_cast", "struct",
		"switch", "synchronized", "template", "this", "thread_local", "throw", "true", "try", "typedef",
		"typeid", "typename", "union", "unsigned", "using", "virtual", "void", "volatile", "wchar_t", "while",
		"xor", "xor_eq",
	}, symbol)
}

func dartReservedIdentifier(symbol string) bool {
	return stringInSlice([]string{
		"assert", "break", "case", "catch", "class", "const", "continue", "default", "do", "else", "enum",
		"extends", "false", "final", "finally", "for", "if", "in", "is", "new", "null", "rethrow", "return",
		"super", "switch", "this", "throw", "true", "try", "var", "void", "while", "with",
	}, symbol)
}

func esReservedIdentifier(symbol string) bool {
	return stringInSlice([]string{
		"await", "break", "case", "catch", "class", "const", "continue", "debugger", "default", "delete",
		"do", "else", "enum", "export", "extends", "false", "finally", "for", "function", "if", "import",
		"in", "instanceof", "new", "null", "return", "super", "switch", "this", "throw", "true", "try",
		"typeof", "var", "void", "while", "with", "yield", "let", "static", "implements", "interface",
		"package", "private", "protected", "public",
	}, symbol)
}

func pythonReservedIdentifier(symbol string) bool {
	return stringInSlice([]string{
		"False", "None", "True", "and", "as", "assert", "async", "await", "break", "class", "continue",
		"def", "del", "elif", "else", "except", "finally", "for", "from", "global", "if", "import",
		"in", "is", "lambda", "nonlocal", "not", "or", "pass", "raise", "return", "try", "while",
		"with", "yield",
	}, symbol)
}

func rustReservedIdentifier(symbol string) bool {
	return stringInSlice([]string{
		"as", "async", "await", "break", "const", "continue", "crate", "dyn", "else", "enum", "extern",
		"false", "fn", "for", "if", "impl", "in", "let", "loop", "match", "mod", "move", "mut", "pub",
		"ref", "return", "Self", "self", "static", "struct", "super", "trait", "true", "type", "unsafe",
		"use", "where", "while", "abstract", "become", "box", "do", "final", "macro", "override", "priv",
		"typeof", "unsized", "virtual", "yield",
	}, symbol)
}

func zigReservedIdentifier(symbol string) bool {
	return stringInSlice([]string{
		"addrspace", "align", "allowzero", "and", "anyframe", "anytype", "asm", "async", "await", "break",
		"callconv", "catch", "comptime", "const", "continue", "defer", "else", "enum", "errdefer", "error",
		"export", "extern", "false", "fn", "for", "if", "inline", "noalias", "noinline", "nosuspend",
		"null", "opaque", "or", "orelse", "packed", "pub", "resume", "return", "linksection", "struct",
		"suspend", "switch", "test", "threadlocal", "true", "try", "undefined", "union", "unreachable",
		"usingnamespace", "var", "volatile", "while",
	}, symbol)
}

func targetTopLevelDeclarationName(target Language, code string) (string, bool) {
	switch target {
	case LanguageCPP:
		if name, ok := cLikeMethodDeclarationName(code); ok {
			return name, true
		}
		if name, ok := cLikeMethodPrototypeName(code); ok {
			return name, true
		}
		if name, ok := cppTypeDeclarationName(code); ok {
			return name, true
		}
		if name, ok := cppUsingDeclarationName(code); ok {
			return name, true
		}
		if name, ok := cppTypedefDeclarationName(code); ok {
			return name, true
		}
		if name, ok := cppTypedVariableDeclarationName(code); ok {
			return name, true
		}
		return cppTopLevelDeclarationName(code)
	case LanguageCSharp:
		if name, ok := csharpUsingAliasDeclarationName(code); ok {
			return name, true
		}
		if name, ok := csharpExternAliasDeclarationName(code); ok {
			return name, true
		}
		if name, ok := csharpMethodDeclarationName(code); ok {
			return name, true
		}
		if name, ok := cLikeMethodPrototypeName(code); ok {
			return name, true
		}
		if name, ok := cLikeConstructorDeclarationName(code); ok {
			return name, true
		}
		if name, ok := csharpPropertyDeclarationName(code); ok {
			return name, true
		}
		if name, ok := csharpDelegateDeclarationName(code); ok {
			return name, true
		}
		if name, ok := cLikeTypeDeclarationName(code); ok {
			return name, true
		}
		return cLikeFieldDeclarationName(code)
	case LanguageDart:
		if names := dartImportDeclarationNames(code); len(names) > 0 {
			return names[0], true
		}
		if name, ok := dartTopLevelVariableName(code); ok {
			return name, true
		}
		if name, ok := dartTypeDeclarationName(code); ok {
			return name, true
		}
		if name, ok := dartExternalFunctionDeclarationName(code); ok {
			return name, true
		}
		return dartFunctionDeclarationName(code)
	case LanguageGo:
		return goTopLevelDeclarationName(code)
	case LanguageJava:
		if name, ok := javaImportDeclarationName(code); ok {
			return name, true
		}
		if name, ok := javaMethodDeclarationName(code); ok {
			return name, true
		}
		if name, ok := cLikeMethodPrototypeName(code); ok {
			return name, true
		}
		if name, ok := cLikeConstructorDeclarationName(code); ok {
			return name, true
		}
		if name, ok := cLikeTypeDeclarationName(code); ok {
			return name, true
		}
		return cLikeFieldDeclarationName(code)
	case LanguagePython:
		return pythonTopLevelDeclarationName(code)
	case LanguageRust:
		if names := rustUseDeclarationNames(code); len(names) > 0 {
			return names[0], true
		}
		if name, ok := rustFunctionDeclarationName(code); ok {
			return name, true
		}
		if name, ok := rustVariableDeclarationName(code); ok {
			return name, true
		}
		return keywordTypeDeclarationName(code, "struct", "enum", "type", "trait", "union", "mod")
	case LanguageTS, LanguageJS:
		return esTopLevelDeclarationName(code)
	case LanguageZig:
		if name, ok := zigFunctionDeclarationName(code); ok {
			return name, true
		}
		return keywordVariableDeclarationName(code, "const", "var")
	default:
		return "", false
	}
}

func targetTopLevelDeclarationNames(target Language, code string) []string {
	if target == LanguagePython {
		return pythonTopLevelDeclarationNames(code)
	}
	var names []string
	remaining := strings.TrimSpace(code)
	for remaining != "" {
		construct := firstTopLevelConstruct(remaining)
		if construct == "" {
			construct = remaining
		}
		if target == LanguageGo {
			if importNames := goImportDeclarationNames(construct); len(importNames) > 0 {
				names = append(names, importNames...)
				next := strings.TrimSpace(remainderAfterFirstTopLevelConstruct(remaining))
				if next == "" || next == remaining {
					break
				}
				remaining = next
				continue
			}
			if groupedNames := goGroupedDeclarationNames(construct); len(groupedNames) > 0 {
				names = append(names, groupedNames...)
				next := strings.TrimSpace(remainderAfterFirstTopLevelConstruct(remaining))
				if next == "" || next == remaining {
					break
				}
				remaining = next
				continue
			}
			if sameStatementNames := goSameStatementDeclarationNames(construct); len(sameStatementNames) > 0 {
				names = append(names, sameStatementNames...)
				next := strings.TrimSpace(remainderAfterFirstTopLevelConstruct(remaining))
				if next == "" || next == remaining {
					break
				}
				remaining = next
				continue
			}
		}
		if target == LanguageJS || target == LanguageTS {
			if esImportNames := esImportDeclarationNames(construct); len(esImportNames) > 0 {
				names = append(names, esImportNames...)
				next := strings.TrimSpace(remainderAfterFirstTopLevelConstruct(remaining))
				if next == "" || next == remaining {
					break
				}
				remaining = next
				continue
			}
			if esNames := esVariableDeclarationNames(construct); len(esNames) > 0 {
				names = append(names, esNames...)
				next := strings.TrimSpace(remainderAfterFirstTopLevelConstruct(remaining))
				if next == "" || next == remaining {
					break
				}
				remaining = next
				continue
			}
		}
		if target == LanguageDart {
			if dartImportNames := dartImportDeclarationNames(construct); len(dartImportNames) > 0 {
				names = append(names, dartImportNames...)
				next := strings.TrimSpace(remainderAfterFirstTopLevelConstruct(remaining))
				if next == "" || next == remaining {
					break
				}
				remaining = next
				continue
			}
		}
		if target == LanguageRust {
			if rustUseNames := rustUseDeclarationNames(construct); len(rustUseNames) > 0 {
				names = append(names, rustUseNames...)
				next := strings.TrimSpace(remainderAfterFirstTopLevelConstruct(remaining))
				if next == "" || next == remaining {
					break
				}
				remaining = next
				continue
			}
		}
		if statementNames := sameStatementDeclarationNames(target, construct); len(statementNames) > 0 {
			names = append(names, statementNames...)
		} else if name, ok := targetTopLevelDeclarationName(target, remaining); ok && name != "" {
			names = append(names, name)
		}
		next := strings.TrimSpace(remainderAfterFirstTopLevelConstruct(remaining))
		if next == "" || next == remaining {
			break
		}
		remaining = next
	}
	return names
}

func goGroupedDeclarationNames(code string) []string {
	trimmed := strings.TrimSpace(code)
	var rest string
	kind := ""
	switch {
	case strings.HasPrefix(trimmed, "var"):
		kind = "var"
		rest = strings.TrimSpace(strings.TrimPrefix(trimmed, "var"))
	case strings.HasPrefix(trimmed, "const"):
		kind = "const"
		rest = strings.TrimSpace(strings.TrimPrefix(trimmed, "const"))
	case strings.HasPrefix(trimmed, "type"):
		kind = "type"
		rest = strings.TrimSpace(strings.TrimPrefix(trimmed, "type"))
	default:
		return nil
	}
	if !strings.HasPrefix(rest, "(") {
		return nil
	}
	closeParen := matchingCloseParen(rest)
	if closeParen < 0 {
		return nil
	}
	body := rest[1:closeParen]
	var names []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if kind == "var" || kind == "const" {
			if sameStatementNames := goSameStatementDeclarationNames(kind + " " + line); len(sameStatementNames) > 0 {
				names = append(names, sameStatementNames...)
				continue
			}
		}
		name := trimDeclarationName(strings.Fields(line)[0])
		if isSimpleIdentifier(name) {
			names = append(names, name)
		}
	}
	return names
}

func goSameStatementDeclarationNames(code string) []string {
	trimmed := strings.TrimSpace(strings.TrimSuffix(code, ";"))
	var rest string
	switch {
	case strings.HasPrefix(trimmed, "var "):
		rest = strings.TrimSpace(strings.TrimPrefix(trimmed, "var"))
	case strings.HasPrefix(trimmed, "const "):
		rest = strings.TrimSpace(strings.TrimPrefix(trimmed, "const"))
	default:
		return nil
	}
	left := rest
	if eq := firstTopLevelByte(rest, '='); eq >= 0 {
		left = strings.TrimSpace(rest[:eq])
	}
	if strings.HasPrefix(left, "(") {
		return nil
	}
	segments := splitTopLevelCommaSegments(left)
	if len(segments) <= 1 {
		return nil
	}
	var names []string
	for _, segment := range segments {
		name := goIdentifierListSegmentName(segment)
		if name == "" {
			return nil
		}
		names = append(names, name)
	}
	return names
}

func goIdentifierListSegmentName(segment string) string {
	fields := strings.Fields(strings.TrimSpace(segment))
	if len(fields) == 0 {
		return ""
	}
	name := trimDeclarationName(fields[0])
	if !isSimpleIdentifier(name) {
		return ""
	}
	return name
}

func sameStatementDeclarationNames(target Language, code string) []string {
	switch target {
	case LanguageCPP, LanguageCSharp, LanguageDart, LanguageJava, LanguageJS, LanguageTS:
	default:
		return nil
	}
	segments := splitTopLevelCommaSegments(strings.TrimSpace(strings.TrimSuffix(code, ";")))
	if len(segments) <= 1 {
		return nil
	}
	var names []string
	if name, ok := targetTopLevelDeclarationName(target, segments[0]); ok && name != "" {
		names = append(names, name)
	} else {
		return nil
	}
	for _, segment := range segments[1:] {
		if name := declarationContinuationNameFor(target, segment); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func splitTopLevelCommaSegments(code string) []string {
	var segments []string
	start := 0
	state := codeScanState{}
	for index := 0; index < len(code); index++ {
		if state.step(code, index) {
			continue
		}
		if state.depth() == 0 && code[index] == ',' {
			segments = append(segments, strings.TrimSpace(code[start:index]))
			start = index + 1
			continue
		}
		state.trackDelimiter(code[index])
	}
	segments = append(segments, strings.TrimSpace(code[start:]))
	return segments
}

func declarationContinuationName(segment string) string {
	return declarationContinuationNameFor("", segment)
}

func declarationContinuationNameFor(target Language, segment string) string {
	segment = strings.TrimSpace(segment)
	if segment == "" {
		return ""
	}
	for _, separator := range []string{"=", ":", ";", "{", "(", ","} {
		if index := strings.Index(segment, separator); index >= 0 {
			segment = segment[:index]
		}
	}
	fields := strings.Fields(strings.TrimSpace(segment))
	if len(fields) == 0 {
		return ""
	}
	name := fields[len(fields)-1]
	if target == LanguageCPP {
		name = trimCPPDeclaratorName(name)
	} else if target == LanguageJava {
		name = trimCLikeDeclaratorName(name)
	}
	if !isSimpleIdentifier(name) {
		return ""
	}
	return name
}

func pythonTopLevelDeclarationNames(code string) []string {
	var names []string
	lines := strings.Split(code, "\n")
	for index := 0; index < len(lines); index++ {
		line := lines[index]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		importCode := trimmed
		if strings.HasPrefix(trimmed, "from ") && strings.Contains(trimmed, " import (") {
			var block []string
			block = append(block, line)
			for index+1 < len(lines) {
				index++
				block = append(block, lines[index])
				if strings.Contains(lines[index], ")") {
					break
				}
			}
			importCode = strings.Join(block, "\n")
		}
		if importNames := pythonImportDeclarationNames(importCode); len(importNames) > 0 {
			names = append(names, importNames...)
			continue
		}
		if assignmentNames := pythonAssignmentDeclarationNames(trimmed); len(assignmentNames) > 0 {
			names = append(names, assignmentNames...)
			continue
		}
		if name, ok := targetTopLevelDeclarationName(LanguagePython, trimmed); ok && name != "" {
			names = append(names, name)
		}
	}
	return names
}

func firstTopLevelConstruct(code string) string {
	if index := firstTopLevelSemicolon(code); index >= 0 {
		return strings.TrimSpace(code[:index+1])
	}
	openBrace := firstTopLevelByte(code, '{')
	if openBrace < 0 {
		return strings.TrimSpace(code)
	}
	closeBrace := matchingCloseBrace(code, openBrace)
	if closeBrace < 0 {
		return strings.TrimSpace(code)
	}
	construct := strings.TrimSpace(code[:closeBrace+1])
	rest := strings.TrimSpace(code[closeBrace+1:])
	if strings.HasPrefix(rest, ";") {
		construct += ";"
	}
	return construct
}

func remainderAfterFirstTopLevelConstruct(code string) string {
	if index := firstTopLevelSemicolon(code); index >= 0 {
		return code[index+1:]
	}
	openBrace := firstTopLevelByte(code, '{')
	if openBrace < 0 {
		return ""
	}
	closeBrace := matchingCloseBrace(code, openBrace)
	if closeBrace < 0 {
		return ""
	}
	rest := strings.TrimSpace(code[closeBrace+1:])
	if strings.HasPrefix(rest, ";") {
		rest = strings.TrimSpace(rest[1:])
	}
	return rest
}

func behaviorSymbolFor(target Language, id string) string {
	switch target {
	case LanguageCPP, LanguageRust, LanguageZig:
		return snakeName(id)
	case LanguageCSharp, LanguageGo, LanguageJava:
		return exportName(id)
	case LanguagePython:
		return snakeName(id)
	case LanguageDart, LanguageJS, LanguageTS:
		return lowerCamel(exportName(id))
	default:
		return ""
	}
}

func modelSymbolFor(target Language, name string) string {
	switch target {
	case LanguageCPP, LanguageRust, LanguageZig:
		return snakeName(name) + "_model"
	case LanguageCSharp, LanguageGo, LanguageJava:
		return exportName(name) + "Model"
	case LanguagePython:
		return snakeName(name) + "_model"
	case LanguageDart:
		return lowerCamel(exportName(name)) + "Model"
	case LanguageJS, LanguageTS:
		return exportName(name) + "Model"
	default:
		return ""
	}
}

func generatedScaffoldSymbols(program *Program, target Language) []string {
	switch target {
	case LanguageCSharp, LanguageJava:
		return []string{"GeneratedHsm"}
	case LanguagePython:
		fallbacks := collectPythonTriggerFallbacks(program)
		if len(fallbacks) == 0 {
			return nil
		}
		symbols := make([]string, 0, len(fallbacks))
		for _, fallback := range fallbacks {
			symbols = append(symbols, fallback.Name)
		}
		return symbols
	case LanguageRust:
		if program != nil && len(program.Models) > 0 {
			return []string{"HsmcInstance", "hsmc_true_guard", "hsmc_zero_duration", "hsmc_unix_epoch"}
		}
	case LanguageZig:
		var symbols []string
		if program != nil && zigNeedsTriggerZeroFallback(program) {
			symbols = append(symbols, "hsmc_trigger_zero")
		}
		for _, fallback := range collectZigOperationReferenceFallbacks(program) {
			symbols = append(symbols, fallback.Name)
		}
		return symbols
	}
	return nil
}

func validateGlobalPatchImmutableFields(block CodeBlock) error {
	if block.Language != "" {
		return fmt.Errorf("adapter patch global %q attempts to change language", block.ID)
	}
	if block.Range != (SourceRange{}) {
		return fmt.Errorf("adapter patch global %q attempts to change range", block.ID)
	}
	return nil
}

func validateBehaviorPatchImmutableFields(behavior Behavior) error {
	if behavior.OwnerID != "" {
		return fmt.Errorf("adapter patch behavior %q attempts to change owner_id", behavior.ID)
	}
	if behavior.Kind != "" {
		return fmt.Errorf("adapter patch behavior %q attempts to change kind", behavior.ID)
	}
	if behavior.TriggerKind != "" {
		return fmt.Errorf("adapter patch behavior %q attempts to change trigger_kind", behavior.ID)
	}
	if behavior.Inline {
		return fmt.Errorf("adapter patch behavior %q attempts to change inline", behavior.ID)
	}
	if behavior.SourceLanguage != "" {
		return fmt.Errorf("adapter patch behavior %q attempts to change source_language", behavior.ID)
	}
	if behavior.TargetLanguage != "" {
		return fmt.Errorf("adapter patch behavior %q attempts to change target_language", behavior.ID)
	}
	if behavior.Signature != "" {
		return fmt.Errorf("adapter patch behavior %q attempts to change signature", behavior.ID)
	}
	if behavior.TargetABI != nil {
		return fmt.Errorf("adapter patch behavior %q attempts to change target_abi", behavior.ID)
	}
	if behavior.Range != (SourceRange{}) {
		return fmt.Errorf("adapter patch behavior %q attempts to change range", behavior.ID)
	}
	return nil
}

func normalizeAdapterGlobal(block CodeBlock, target Language) (CodeBlock, error) {
	if strings.TrimSpace(block.ID) == "" {
		return CodeBlock{}, errors.New("adapter-created global id is empty")
	}
	if block.ID != strings.TrimSpace(block.ID) {
		return CodeBlock{}, fmt.Errorf("adapter-created global id %q has leading or trailing whitespace", block.ID)
	}
	if !strings.HasPrefix(block.ID, "adapter_global_") {
		return CodeBlock{}, fmt.Errorf("adapter patch references unknown global %q", block.ID)
	}
	if block.ID == "adapter_global_" {
		return CodeBlock{}, errors.New(`adapter-created global id "adapter_global_" is missing a helper name`)
	}
	if !validAdapterGlobalID(block.ID) {
		return CodeBlock{}, fmt.Errorf("adapter-created global id %q must contain only letters, digits, and underscores after adapter_global_", block.ID)
	}
	if err := validateGlobalPatchImmutableFields(block); err != nil {
		return CodeBlock{}, err
	}
	if strings.TrimSpace(block.Code) == "" {
		return CodeBlock{}, fmt.Errorf("adapter-created global %q is missing target code", block.ID)
	}
	block.Language = target
	if err := normalizePatchImports(block.Imports, target, fmt.Sprintf("global %q", block.ID)); err != nil {
		return CodeBlock{}, err
	}
	block.Imports = cloneImports(block.Imports)
	return block, nil
}

func validAdapterGlobalID(id string) bool {
	const prefix = "adapter_global_"
	if !strings.HasPrefix(id, prefix) || len(id) == len(prefix) {
		return false
	}
	for _, r := range id[len(prefix):] {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func normalizePatchLanguage(value Language, target Language, owner string, field string) (Language, error) {
	if value == "" {
		return "", nil
	}
	language, ok := ParseLanguage(string(value))
	if !ok {
		return "", fmt.Errorf("adapter patch %s has unsupported %s %q", owner, field, value)
	}
	if language != target {
		return "", fmt.Errorf("adapter patch %s has %s %q, want target language %q", owner, field, language, target)
	}
	return language, nil
}

func adapterGuidance(request AdapterRequest) AdapterGuidance {
	target := request.TargetLanguage
	return AdapterGuidance{
		TargetLanguage:        target,
		CompilerOwnedImports:  compilerOwnedImports(target),
		CompilerOwnedBindings: compilerOwnedBindings(target),
		CompilerOwnedSymbols:  adapterCompilerOwnedSymbols(request),
		EditableFields: []string{
			"imports (target-language program imports for translated code only)",
			"globals[].code",
			"globals[].imports",
			"behaviors[].body (translated statements inside the compiler-owned behavior signature)",
			"behaviors[].code (single complete target-language callable using the compiler-owned behavior name, with no extra top-level declarations)",
			"behaviors[].imports",
		},
		ForbiddenFields: []string{
			"source_language",
			"target_language",
			"package_name",
			"models",
			"events",
			"attributes",
			"operations",
			"states",
			"initial",
			"transitions",
			"triggers",
			"defers",
			"imports[].local_names",
			"imports[].range",
			"imports[].specifiers[].range",
			"globals[].imports[].local_names",
			"globals[].imports[].range",
			"globals[].imports[].specifiers[].range",
			"global declarations that collide with compiler-owned symbols",
			"global declarations that collide with compiler-owned runtime bindings",
			"imports that bind compiler-owned symbols",
			"imports that bind compiler-owned runtime bindings",
			"imports that bind names declared by returned target-language globals",
			"global ID changes (id is required only as a selector; helper globals must use adapter_global_ IDs)",
			"globals[].language",
			"globals[].range",
			"behavior ID changes (id is required only as a selector)",
			"behaviors[].owner_id",
			"behaviors[].kind",
			"behaviors[].trigger_kind",
			"behaviors[].inline",
			"behaviors[].source_language",
			"behaviors[].target_language",
			"behaviors[].signature",
			"behaviors[].target_abi",
			"behaviors[].range",
			"behaviors[].imports[].local_names",
			"behaviors[].imports[].range",
			"behaviors[].imports[].specifiers[].range",
		},
		TargetImportShape: targetImportShape(target),
	}
}

func adapterCompilerOwnedSymbols(request AdapterRequest) []string {
	target := request.TargetLanguage
	if !isImplementationLanguage(target) {
		return nil
	}
	symbols := map[string]bool{}
	for _, event := range request.Events {
		if event.Symbol != "" {
			symbols[eventSymbolFor(target, event.Symbol)] = true
		}
		if target == LanguageCPP && event.Name != "" {
			symbols[cppEventTypeName(event.Name)] = true
		}
	}
	if target == LanguageCPP {
		for _, eventName := range cppDeferredEventNamesFromViews(request.Models) {
			symbols[cppEventTypeName(eventName)] = true
		}
	}
	for _, behavior := range request.Behaviors {
		if behavior.ID != "" {
			symbols[behaviorSymbolFor(target, behavior.ID)] = true
		}
	}
	for _, model := range request.Models {
		if model.Name != "" {
			symbols[modelSymbolFor(target, model.Name)] = true
		}
	}
	for _, symbol := range adapterScaffoldSymbols(request) {
		symbols[symbol] = true
	}
	if len(symbols) == 0 {
		return nil
	}
	result := make([]string, 0, len(symbols))
	for symbol := range symbols {
		if symbol != "" {
			result = append(result, symbol)
		}
	}
	sort.Strings(result)
	return result
}

func adapterScaffoldSymbols(request AdapterRequest) []string {
	switch request.TargetLanguage {
	case LanguageCSharp, LanguageJava:
		return []string{"GeneratedHsm"}
	case LanguagePython:
		return pythonTriggerFallbackNamesFromViews(request.Models)
	case LanguageRust:
		if len(request.Models) > 0 {
			return []string{"HsmcInstance", "hsmc_true_guard", "hsmc_zero_duration", "hsmc_unix_epoch"}
		}
	case LanguageZig:
		var symbols []string
		if zigNeedsTriggerZeroFallbackFromViews(request.Models) {
			symbols = append(symbols, "hsmc_trigger_zero")
		}
		symbols = append(symbols, zigOperationReferenceFallbackNamesFromViews(request.Models)...)
		return symbols
	}
	return nil
}

func pythonTriggerFallbackNamesFromViews(models []ModelView) []string {
	seen := map[string]bool{}
	var names []string
	var visitTransition func(TransitionView)
	var visitState func(StateView)
	visitTransition = func(transition TransitionView) {
		if transition.Trigger == nil || !pythonNeedsTriggerFallback(*transition.Trigger) {
			return
		}
		key := pythonTriggerFallbackKey(*transition.Trigger)
		if seen[key] {
			return
		}
		seen[key] = true
		names = append(names, fmt.Sprintf("hsmc_trigger_fallback_%d", len(names)+1))
	}
	visitState = func(state StateView) {
		for _, transition := range state.Transitions {
			visitTransition(transition)
		}
		for _, child := range state.States {
			visitState(child)
		}
	}
	for _, model := range models {
		for _, transition := range model.Transitions {
			visitTransition(transition)
		}
		for _, state := range model.States {
			visitState(state)
		}
	}
	return names
}

func cppDeferredEventNamesFromViews(models []ModelView) []string {
	seen := map[string]bool{}
	var names []string
	var visitState func(StateView)
	visitState = func(state StateView) {
		for _, name := range state.Defers {
			name = strings.TrimSpace(name)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			names = append(names, name)
		}
		for _, child := range state.States {
			visitState(child)
		}
	}
	for _, model := range models {
		for _, state := range model.States {
			visitState(state)
		}
	}
	return names
}

func zigNeedsTriggerZeroFallbackFromViews(models []ModelView) bool {
	for _, model := range models {
		for _, transition := range model.Transitions {
			if zigTransitionViewNeedsTriggerZeroFallback(transition) {
				return true
			}
		}
		for _, state := range model.States {
			if zigStateViewNeedsTriggerZeroFallback(state) {
				return true
			}
		}
	}
	return false
}

func zigStateViewNeedsTriggerZeroFallback(state StateView) bool {
	for _, transition := range state.Transitions {
		if zigTransitionViewNeedsTriggerZeroFallback(transition) {
			return true
		}
	}
	for _, child := range state.States {
		if zigStateViewNeedsTriggerZeroFallback(child) {
			return true
		}
	}
	return false
}

func zigTransitionViewNeedsTriggerZeroFallback(transition TransitionView) bool {
	if transition.Trigger == nil {
		return false
	}
	trigger := *transition.Trigger
	if trigger.Kind != TriggerAfter && trigger.Kind != TriggerEvery && trigger.Kind != TriggerAt {
		return false
	}
	return trigger.Expr == nil && !trigger.IsString && trigger.Language != "" && trigger.Language != LanguageZig
}

func zigOperationReferenceFallbackNamesFromViews(models []ModelView) []string {
	seen := map[string]bool{}
	var names []string
	add := func(ref BehaviorRef) {
		if ref.Operation == "" {
			return
		}
		switch ref.Kind {
		case BehaviorEntry, BehaviorExit, BehaviorActivity, BehaviorGuard, BehaviorEffect:
		default:
			return
		}
		name := zigOperationReferenceFallbackName(ref.Kind, ref.Operation)
		if seen[name] {
			return
		}
		seen[name] = true
		names = append(names, name)
	}
	var visitState func(StateView)
	var visitTransition func(TransitionView)
	visitTransition = func(transition TransitionView) {
		if transition.Guard != nil {
			add(*transition.Guard)
		}
		for _, effect := range transition.Effects {
			add(effect)
		}
	}
	visitState = func(state StateView) {
		for _, behavior := range state.Behaviors {
			add(behavior)
		}
		for _, effect := range state.InitialEffects {
			add(effect)
		}
		for _, transition := range state.Transitions {
			visitTransition(transition)
		}
		for _, child := range state.States {
			visitState(child)
		}
	}
	for _, model := range models {
		for _, effect := range model.InitialEffects {
			add(effect)
		}
		for _, transition := range model.Transitions {
			visitTransition(transition)
		}
		for _, state := range model.States {
			visitState(state)
		}
	}
	return names
}

func compilerOwnedBindings(target Language) []string {
	seen := map[string]bool{}
	var bindings []string
	for _, imp := range compilerOwnedBindingImports(target) {
		for _, binding := range importBindings(imp, target) {
			if ignorableImportLocal(binding.Local) || seen[binding.Local] {
				continue
			}
			seen[binding.Local] = true
			bindings = append(bindings, binding.Local)
		}
	}
	sort.Strings(bindings)
	return bindings
}

func compilerOwnedImports(target Language) []string {
	switch target {
	case LanguageCSharp:
		return []string{
			"Stateforward.Hsm",
			"Stateforward.Hsm.Context",
			"Stateforward.Hsm.ConditionChannel",
			"Stateforward.Hsm.DurationProvider",
			"Stateforward.Hsm.Event",
			"Stateforward.Hsm.Expression",
			"Stateforward.Hsm.Hsm",
			"Stateforward.Hsm.Instance",
			"Stateforward.Hsm.Model",
			"Stateforward.Hsm.Operation",
			"Stateforward.Hsm.TimeProvider",
			"System",
			"System.Threading",
			"System.Threading.Tasks",
		}
	case LanguageCPP:
		return []string{"hsm/hsm.hpp", "chrono", "cstdint", "string"}
	case LanguageDart:
		return []string{"package:hsm/hsm.dart", "dart:async", "dart:core"}
	case LanguageGo:
		return []string{"github.com/stateforward/hsm.go", "context", "time", "builtin", "builtin.bool", "builtin.false", "builtin.nil", "builtin.true"}
	case LanguageJava:
		return []string{
			"com.stateforward.hsm",
			"com.stateforward.hsm.*",
			"com.stateforward.hsm.Context",
			"com.stateforward.hsm.Event",
			"com.stateforward.hsm.Hsm",
			"com.stateforward.hsm.Instance",
			"com.stateforward.hsm.Model",
			"java.lang.Object",
		}
	case LanguageJS:
		return []string{"@stateforward/hsm", "javascript", "javascript.Date", "javascript.undefined"}
	case LanguageTS:
		return []string{"@stateforward/hsm.ts", "@stateforward/hsm", "typescript", "typescript.Date", "typescript.InstanceType", "typescript.Promise", "typescript.undefined"}
	case LanguagePython:
		return []string{"hsm", "datetime", "typing", "builtins", "builtins.False", "builtins.None", "builtins.True", "builtins.bool", "builtins.float"}
	case LanguageRust:
		return []string{"hsm", "hsm::*", "std::any", "std::any::Any", "std::boxed", "std::boxed::Box", "std::future", "std::future::Future", "std::marker", "std::marker::Send", "std::pin", "std::pin::Pin", "std::time", "std::time::Duration"}
	case LanguageZig:
		return []string{"std", "hsm"}
	default:
		return nil
	}
}

func targetImportShape(target Language) ImportShapeRules {
	switch target {
	case LanguageCSharp:
		return ImportShapeRules{
			Language:             target,
			SupportsBareImport:   true,
			SupportsModuleAlias:  true,
			SupportsStaticImport: true,
			Notes: []string{
				"Use C# namespace imports such as System.Text, optional aliases, or static usings by setting static.",
				"Do not combine static with alias; C# static usings cannot be alias usings.",
				"Default imports and named specifiers are rejected.",
			},
		}
	case LanguageCPP:
		return ImportShapeRules{
			Language:           target,
			SupportsBareImport: true,
			Notes: []string{
				"Use C++ include paths such as hsm/hsm.hpp, vector, or ./helpers.hpp.",
				"Aliases, default imports, and named specifiers are rejected.",
			},
		}
	case LanguageDart:
		return ImportShapeRules{
			Language:                 target,
			SupportsBareImport:       true,
			SupportsNamedSpecifiers:  true,
			SupportsModuleAlias:      true,
			SupportsDeferredImport:   true,
			DisallowAliasWithMembers: true,
			Notes: []string{
				"Use Dart import URIs such as package:foo/foo.dart or relative paths.",
				"Alias maps to `as`; deferred imports must set alias; named specifiers map to `show`; default imports, hidden specifiers, and specifier aliases are rejected for adapter-created target imports.",
				"Do not combine a Dart prefix alias with named specifiers in translated target imports; use either a prefix import or a show import.",
				"Native Dart source context may preserve hide combinators, but translated target imports cannot use them because they implicitly bind unlisted names.",
			},
		}
	case LanguageGo:
		return ImportShapeRules{
			Language:            target,
			SupportsBareImport:  true,
			SupportsModuleAlias: true,
			Notes: []string{
				"Use path plus optional alias only; default imports and named specifiers are rejected.",
			},
		}
	case LanguageJava:
		return ImportShapeRules{
			Language:             target,
			SupportsBareImport:   true,
			SupportsStaticImport: true,
			Notes: []string{
				"Use Java package or class imports such as java.util.List or com.example.Helper; set static for Java static imports such as com.example.Helper.record.",
				"Aliases, default imports, and named specifiers are rejected except source-context wildcard specifiers.",
			},
		}
	case LanguageJS, LanguageTS:
		return ImportShapeRules{
			Language:                  target,
			SupportsBareImport:        true,
			SupportsDefaultImport:     true,
			SupportsNamedSpecifiers:   true,
			SupportsNamespaceAlias:    true,
			SupportsTypeOnlyImport:    target == LanguageTS,
			SupportsSpecifierTypeOnly: target == LanguageTS,
			DisallowAliasWithMembers:  true,
			Notes: []string{
				"Use alias for namespace imports only, for example import * as alias from path.",
				"Do not combine namespace alias with default or named specifiers.",
				"Type-only imports and type-only named specifiers are supported only for TypeScript.",
			},
		}
	case LanguagePython:
		return ImportShapeRules{
			Language:                 target,
			SupportsBareImport:       true,
			SupportsNamedSpecifiers:  true,
			SupportsModuleAlias:      true,
			DisallowAliasWithMembers: true,
			Notes: []string{
				"Default imports are rejected.",
				"Do not combine module alias with from-import specifiers.",
				"Wildcard from-imports are rejected for adapter-created target imports because they implicitly bind unlisted names.",
			},
		}
	case LanguageRust:
		return ImportShapeRules{
			Language:                 target,
			SupportsBareImport:       true,
			SupportsNamedSpecifiers:  true,
			SupportsModuleAlias:      true,
			DisallowAliasWithMembers: true,
			Notes: []string{
				"Use Rust module paths such as crate::helpers or std::time.",
				"Default imports are rejected; do not combine a module alias with named specifiers.",
			},
		}
	case LanguageZig:
		return ImportShapeRules{
			Language:            target,
			SupportsBareImport:  true,
			SupportsModuleAlias: true,
			Notes: []string{
				"Use Zig @import paths such as helpers.zig or package module names.",
				"Alias maps to the const binding name; default imports and named specifiers are rejected.",
			},
		}
	default:
		return ImportShapeRules{Language: target}
	}
}

func normalizePatchImports(imports []Import, target Language, owner string) error {
	for index, imp := range imports {
		if strings.TrimSpace(imp.Path) == "" {
			return fmt.Errorf("adapter patch %s contains import with empty path", owner)
		}
		if len(imp.LocalNames) > 0 {
			return fmt.Errorf("adapter patch %s import %q attempts to set local_names", owner, imp.Path)
		}
		if imp.Range != (SourceRange{}) {
			return fmt.Errorf("adapter patch %s import %q attempts to change range", owner, imp.Path)
		}
		if isCompilerOwnedImportPath(imp.Path, target) {
			return fmt.Errorf("adapter patch %s import %q is compiler-owned for target language %q", owner, imp.Path, target)
		}
		if imp.Language == "" {
			imports[index].Language = target
		} else {
			language, err := normalizePatchLanguage(imp.Language, target, fmt.Sprintf("%s import %q", owner, imp.Path), "language")
			if err != nil {
				return err
			}
			imports[index].Language = language
		}
		imports[index].LocalNames = nil
		imports[index].Range = SourceRange{}
		if err := validatePatchImportShape(imports[index], target, owner); err != nil {
			return err
		}
	}
	if err := validatePatchImportBindings(imports, target, owner); err != nil {
		return err
	}
	return nil
}

func isCompilerOwnedImportPath(path string, target Language) bool {
	if target == LanguageCPP {
		path = normalizeCPPIncludePath(path)
	}
	for _, owned := range compilerOwnedImports(target) {
		if target == LanguageCPP {
			owned = normalizeCPPIncludePath(owned)
		}
		if path == owned {
			return true
		}
	}
	return false
}

func normalizeCPPIncludePath(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (strings.HasPrefix(value, "<") && strings.HasSuffix(value, ">")) ||
			(strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) {
			value = value[1 : len(value)-1]
		}
	}
	return strings.TrimSpace(value)
}

func mergeProgramImportPatch(existing []Import, patched []Import, target Language) []Import {
	merged := make([]Import, 0, len(existing)+len(patched))
	for _, imp := range existing {
		if imp.Language != "" && imp.Language != target {
			merged = append(merged, imp)
		}
	}
	return append(merged, patched...)
}

func validatePatchImportShape(imp Import, target Language, owner string) error {
	if target == LanguageJava && isJavaWildcardImport(imp) {
		return fmt.Errorf("adapter patch %s import %q uses Java wildcard import unsupported for translated target imports", owner, imp.Path)
	}
	if target == LanguageRust && isRustWildcardImport(imp) {
		return fmt.Errorf("adapter patch %s import %q uses Rust wildcard import unsupported for translated target imports", owner, imp.Path)
	}
	if target == LanguageDart && len(imp.Hidden) > 0 {
		return fmt.Errorf("adapter patch %s import %q uses Dart hidden specifiers unsupported for translated target imports", owner, imp.Path)
	}
	if target == LanguageDart && imp.Alias != "" && len(imp.Specifiers) > 0 {
		return fmt.Errorf("adapter patch %s import %q combines Dart prefix alias with show specifiers unsupported for translated target imports", owner, imp.Path)
	}
	if target == LanguagePython && isPythonWildcardImport(imp) {
		return fmt.Errorf("adapter patch %s import %q uses Python wildcard import unsupported for translated target imports", owner, imp.Path)
	}
	if target == LanguageJava && len(imp.Specifiers) > 0 {
		return fmt.Errorf("adapter patch %s import %q uses named specifiers unsupported by Java", owner, imp.Path)
	}
	return validateImportShape(imp, target, "adapter patch "+owner)
}

func validateImportShape(imp Import, target Language, context string) error {
	if imp.Static && target != LanguageCSharp && target != LanguageJava {
		return fmt.Errorf("%s import %q uses static import unsupported by %s", context, imp.Path, target)
	}
	if imp.Deferred && target != LanguageDart {
		return fmt.Errorf("%s import %q uses deferred import unsupported by %s", context, imp.Path, target)
	}
	if imp.TypeOnly && target != LanguageTS {
		return fmt.Errorf("%s import %q uses type-only import unsupported by %s", context, imp.Path, target)
	}
	localNames := map[string]bool{}
	explicitLocalNames := map[string]bool{}
	for _, localName := range imp.LocalNames {
		if strings.TrimSpace(localName) == "" {
			return fmt.Errorf("%s import %q contains empty local name", context, imp.Path)
		}
		if localName != strings.TrimSpace(localName) {
			return fmt.Errorf("%s import %q local name %q contains surrounding whitespace", context, imp.Path, localName)
		}
		if explicitLocalNames[localName] {
			return fmt.Errorf("%s import %q contains duplicate local binding %q", context, imp.Path, localName)
		}
		explicitLocalNames[localName] = true
	}
	if imp.Path != strings.TrimSpace(imp.Path) {
		return fmt.Errorf("%s import path %q contains surrounding whitespace", context, imp.Path)
	}
	if imp.Default != "" {
		if strings.TrimSpace(imp.Default) == "" {
			return fmt.Errorf("%s import %q contains empty default binding", context, imp.Path)
		}
		if imp.Default != strings.TrimSpace(imp.Default) {
			return fmt.Errorf("%s import %q default binding %q contains surrounding whitespace", context, imp.Path, imp.Default)
		}
		localNames[imp.Default] = true
	}
	if imp.Alias != "" && strings.TrimSpace(imp.Alias) == "" {
		return fmt.Errorf("%s import %q contains empty alias binding", context, imp.Path)
	}
	if imp.Alias != "" && imp.Alias != strings.TrimSpace(imp.Alias) {
		return fmt.Errorf("%s import %q alias binding %q contains surrounding whitespace", context, imp.Path, imp.Alias)
	}
	for _, specifier := range imp.Specifiers {
		if specifier.TypeOnly && target != LanguageTS {
			return fmt.Errorf("%s import %q specifier %q uses type-only import unsupported by %s", context, imp.Path, specifier.Name, target)
		}
		if strings.TrimSpace(specifier.Name) == "" {
			return fmt.Errorf("%s import %q contains specifier with empty name", context, imp.Path)
		}
		if specifier.Name != strings.TrimSpace(specifier.Name) {
			return fmt.Errorf("%s import %q specifier %q contains surrounding whitespace", context, imp.Path, specifier.Name)
		}
		if specifier.Alias != "" && strings.TrimSpace(specifier.Alias) == "" {
			return fmt.Errorf("%s import %q contains specifier %q with empty alias", context, imp.Path, specifier.Name)
		}
		if specifier.Alias != "" && specifier.Alias != strings.TrimSpace(specifier.Alias) {
			return fmt.Errorf("%s import %q specifier %q alias %q contains surrounding whitespace", context, imp.Path, specifier.Name, specifier.Alias)
		}
		localName := specifier.Name
		if specifier.Alias != "" {
			localName = specifier.Alias
		}
		if localNames[localName] {
			return fmt.Errorf("%s import %q contains duplicate local binding %q", context, imp.Path, localName)
		}
		localNames[localName] = true
	}
	hiddenNames := map[string]bool{}
	for _, hidden := range imp.Hidden {
		if strings.TrimSpace(hidden) == "" {
			return fmt.Errorf("%s import %q contains empty hidden specifier", context, imp.Path)
		}
		if hidden != strings.TrimSpace(hidden) {
			return fmt.Errorf("%s import %q hidden specifier %q contains surrounding whitespace", context, imp.Path, hidden)
		}
		if hiddenNames[hidden] {
			return fmt.Errorf("%s import %q contains duplicate hidden specifier %q", context, imp.Path, hidden)
		}
		hiddenNames[hidden] = true
	}
	switch target {
	case LanguageCSharp:
		if len(imp.Hidden) > 0 {
			return fmt.Errorf("%s import %q uses hidden specifiers unsupported by C#", context, imp.Path)
		}
		if csharpImportIsStatic(imp) && imp.Alias != "" {
			return fmt.Errorf("%s import %q combines static using with alias", context, imp.Path)
		}
		if imp.Default != "" || len(imp.Specifiers) > 0 {
			return fmt.Errorf("%s import %q uses default or named specifiers unsupported by C#", context, imp.Path)
		}
	case LanguageCPP:
		if len(imp.Hidden) > 0 {
			return fmt.Errorf("%s import %q uses hidden specifiers unsupported by C++", context, imp.Path)
		}
		if imp.Alias != "" || imp.Default != "" || len(imp.Specifiers) > 0 {
			return fmt.Errorf("%s import %q uses aliases, defaults, or named specifiers unsupported by C++", context, imp.Path)
		}
	case LanguageDart:
		if imp.Default != "" {
			return fmt.Errorf("%s import %q uses default import unsupported by Dart", context, imp.Path)
		}
		if imp.Deferred && imp.Alias == "" {
			return fmt.Errorf("%s import %q uses deferred import without a Dart alias", context, imp.Path)
		}
		for _, specifier := range imp.Specifiers {
			if specifier.Alias != "" {
				return fmt.Errorf("%s import %q uses specifier alias %q unsupported by Dart", context, imp.Path, specifier.Alias)
			}
		}
	case LanguageJS, LanguageTS:
		if len(imp.Hidden) > 0 {
			return fmt.Errorf("%s import %q uses hidden specifiers unsupported by %s", context, imp.Path, target)
		}
		if imp.Alias != "" && (imp.Default != "" || len(imp.Specifiers) > 0) {
			return fmt.Errorf("%s import %q combines namespace alias with default or named specifiers", context, imp.Path)
		}
	case LanguagePython:
		if len(imp.Hidden) > 0 {
			return fmt.Errorf("%s import %q uses hidden specifiers unsupported by Python", context, imp.Path)
		}
		if imp.Default != "" {
			return fmt.Errorf("%s import %q uses default import unsupported by Python", context, imp.Path)
		}
		if imp.Alias != "" && len(imp.Specifiers) > 0 {
			return fmt.Errorf("%s import %q combines module alias with from-import specifiers", context, imp.Path)
		}
	case LanguageGo:
		if len(imp.Hidden) > 0 {
			return fmt.Errorf("%s import %q uses hidden specifiers unsupported by Go", context, imp.Path)
		}
		if imp.Default != "" || len(imp.Specifiers) > 0 {
			return fmt.Errorf("%s import %q uses default or named specifiers unsupported by Go", context, imp.Path)
		}
	case LanguageJava:
		if len(imp.Hidden) > 0 {
			return fmt.Errorf("%s import %q uses hidden specifiers unsupported by Java", context, imp.Path)
		}
		if imp.Alias != "" || imp.Default != "" || (len(imp.Specifiers) > 0 && !isWildcardImportSpecifier(imp.Specifiers)) {
			return fmt.Errorf("%s import %q uses aliases, defaults, or named specifiers unsupported by Java", context, imp.Path)
		}
	case LanguageRust:
		if len(imp.Hidden) > 0 {
			return fmt.Errorf("%s import %q uses hidden specifiers unsupported by Rust", context, imp.Path)
		}
		if imp.Default != "" {
			return fmt.Errorf("%s import %q uses default import unsupported by Rust", context, imp.Path)
		}
		if imp.Alias != "" && len(imp.Specifiers) > 0 {
			return fmt.Errorf("%s import %q combines module alias with named specifiers", context, imp.Path)
		}
	case LanguageZig:
		if len(imp.Hidden) > 0 {
			return fmt.Errorf("%s import %q uses hidden specifiers unsupported by Zig", context, imp.Path)
		}
		if imp.Default != "" || len(imp.Specifiers) > 0 {
			return fmt.Errorf("%s import %q uses default or named specifiers unsupported by Zig", context, imp.Path)
		}
	}
	return nil
}

func isWildcardImportSpecifier(specifiers []ImportSpecifier) bool {
	return len(specifiers) == 1 && specifiers[0].Name == "*" && specifiers[0].Alias == ""
}

func isJavaWildcardImport(imp Import) bool {
	return isWildcardImportSpecifier(imp.Specifiers) || strings.HasSuffix(strings.TrimSpace(imp.Path), ".*")
}

func isRustWildcardImport(imp Import) bool {
	return isWildcardImportSpecifier(imp.Specifiers) || strings.HasSuffix(strings.TrimSpace(imp.Path), "::*")
}

func isPythonWildcardImport(imp Import) bool {
	return isWildcardImportSpecifier(imp.Specifiers)
}

func validatePatchImportBindings(imports []Import, target Language, owner string) error {
	localToUse := map[string]importBindingUse{}
	for _, imp := range imports {
		for _, binding := range importBindings(imp, target) {
			if ignorableImportLocal(binding.Local) {
				continue
			}
			current := importBindingUse{
				Path:   imp.Path,
				Owner:  owner,
				Source: binding.Source,
			}
			if previous, exists := localToUse[binding.Local]; exists && previous.Source != current.Source {
				return fmt.Errorf("adapter patch %s imports %q and %q both bind local name %q", owner, previous.Path, current.Path, binding.Local)
			}
			localToUse[binding.Local] = current
		}
	}
	return nil
}

func validateTargetImports(program *Program, target Language) error {
	if err := validateTargetImportShapes(program, target); err != nil {
		return err
	}
	return validateTargetImportBindings(program, target)
}

func validateTargetImportShapes(program *Program, target Language) error {
	check := func(owner string, imp Import) error {
		if strings.TrimSpace(imp.Path) == "" {
			return fmt.Errorf("target %s contains import with empty path", owner)
		}
		if imp.Language != "" && imp.Language != target {
			return nil
		}
		if program.SourceLanguage != target && isCompilerOwnedImportPath(imp.Path, target) {
			return fmt.Errorf("target %s import %q is compiler-owned for target language %q", owner, imp.Path, target)
		}
		if target == LanguageJava && program.SourceLanguage != target && isJavaWildcardImport(imp) {
			return fmt.Errorf("target %s import %q uses Java wildcard import unsupported for translated target imports", owner, imp.Path)
		}
		if target == LanguageRust && program.SourceLanguage != target && isRustWildcardImport(imp) {
			return fmt.Errorf("target %s import %q uses Rust wildcard import unsupported for translated target imports", owner, imp.Path)
		}
		if target == LanguageDart && program.SourceLanguage != target && len(imp.Hidden) > 0 {
			return fmt.Errorf("target %s import %q uses Dart hidden specifiers unsupported for translated target imports", owner, imp.Path)
		}
		if target == LanguageDart && program.SourceLanguage != target && imp.Alias != "" && len(imp.Specifiers) > 0 {
			return fmt.Errorf("target %s import %q combines Dart prefix alias with show specifiers unsupported for translated target imports", owner, imp.Path)
		}
		if target == LanguagePython && program.SourceLanguage != target && isPythonWildcardImport(imp) {
			return fmt.Errorf("target %s import %q uses Python wildcard import unsupported for translated target imports", owner, imp.Path)
		}
		return validateImportShape(imp, target, "target "+owner)
	}
	for _, imp := range program.Imports {
		if err := check("program", imp); err != nil {
			return err
		}
	}
	for _, global := range program.Globals {
		for _, imp := range global.Imports {
			if err := check(fmt.Sprintf("global %q", global.ID), imp); err != nil {
				return err
			}
		}
	}
	for _, behavior := range program.Behaviors {
		for _, imp := range behavior.Imports {
			if err := check(fmt.Sprintf("behavior %q", behavior.ID), imp); err != nil {
				return err
			}
		}
	}
	return nil
}

func importLocalBindings(imp Import, target Language) []string {
	bindings := importBindings(imp, target)
	locals := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		locals = append(locals, binding.Local)
	}
	return locals
}

type importBinding struct {
	Local  string
	Source string
}

type importBindingUse struct {
	Path   string
	Owner  string
	Source string
}

func ignorableImportLocal(local string) bool {
	return local == "" || local == "." || local == "_" || local == "*"
}

func importBindings(imp Import, target Language) []importBinding {
	var bindings []importBinding
	add := func(value string, source string) {
		value = strings.TrimSpace(value)
		if value != "" {
			bindings = append(bindings, importBinding{Local: value, Source: source})
		}
	}
	switch target {
	case LanguageCSharp:
		if csharpImportIsStatic(imp) {
			return nil
		}
		if imp.Alias != "" {
			add(imp.Alias, imp.Path+"\x00namespace")
		} else {
			parts := strings.Split(imp.Path, ".")
			add(parts[len(parts)-1], imp.Path+"\x00namespace")
		}
	case LanguageCPP:
		add(cppImportBindingName(imp.Path), imp.Path+"\x00include")
	case LanguageDart:
		if imp.Alias != "" {
			add(imp.Alias, imp.Path+"\x00library")
		} else if len(imp.Specifiers) > 0 {
			for _, specifier := range imp.Specifiers {
				add(specifier.Name, imp.Path+"\x00specifier\x00"+specifier.Name)
			}
		}
	case LanguageJS, LanguageTS:
		add(imp.Default, imp.Path+"\x00default")
		add(imp.Alias, imp.Path+"\x00namespace")
		for _, specifier := range imp.Specifiers {
			if specifier.Alias != "" {
				add(specifier.Alias, imp.Path+"\x00specifier\x00"+specifier.Name)
			} else {
				add(specifier.Name, imp.Path+"\x00specifier\x00"+specifier.Name)
			}
		}
	case LanguagePython:
		if len(imp.Specifiers) > 0 {
			for _, specifier := range imp.Specifiers {
				if specifier.Alias != "" {
					add(specifier.Alias, imp.Path+"\x00specifier\x00"+specifier.Name)
				} else {
					add(specifier.Name, imp.Path+"\x00specifier\x00"+specifier.Name)
				}
			}
		} else if imp.Alias != "" {
			add(imp.Alias, imp.Path+"\x00module")
		} else {
			module, _, _ := strings.Cut(imp.Path, ".")
			add(module, imp.Path+"\x00module")
		}
	case LanguageGo:
		if imp.Alias != "" {
			add(imp.Alias, imp.Path+"\x00package")
		} else {
			add(path.Base(imp.Path), imp.Path+"\x00package")
		}
	case LanguageJava:
		if isJavaWildcardImport(imp) {
			return nil
		}
		parts := strings.Split(imp.Path, ".")
		add(parts[len(parts)-1], imp.Path+"\x00class")
	case LanguageRust:
		if isRustWildcardImport(imp) {
			return nil
		}
		if len(imp.Specifiers) > 0 {
			for _, specifier := range imp.Specifiers {
				if specifier.Alias != "" {
					add(specifier.Alias, imp.Path+"\x00specifier\x00"+specifier.Name)
				} else {
					add(specifier.Name, imp.Path+"\x00specifier\x00"+specifier.Name)
				}
			}
		} else if imp.Alias != "" {
			add(imp.Alias, imp.Path+"\x00module")
		} else {
			parts := strings.Split(imp.Path, "::")
			add(parts[len(parts)-1], imp.Path+"\x00module")
		}
	case LanguageZig:
		if imp.Alias != "" {
			add(imp.Alias, imp.Path+"\x00module")
		} else {
			add(zigImportBindingName(imp.Path), imp.Path+"\x00module")
		}
	}
	return bindings
}

func validateTargetImportBindings(program *Program, target Language) error {
	localToUse := map[string]importBindingUse{}
	compilerOwnedSymbols := generatedTopLevelSymbols(program, target)
	check := func(owner string, imp Import) error {
		if imp.Path == "" {
			return nil
		}
		if imp.Language != "" && imp.Language != target {
			return nil
		}
		renderedLocals := map[string]bool{}
		for _, binding := range importBindings(imp, target) {
			if ignorableImportLocal(binding.Local) {
				continue
			}
			renderedLocals[binding.Local] = true
			if compilerOwnedSymbols[binding.Local] {
				return fmt.Errorf("target imports for %s import %q bind local name %q, which collides with compiler-owned %s symbol", owner, imp.Path, binding.Local, target)
			}
			current := importBindingUse{Path: imp.Path, Owner: owner, Source: binding.Source}
			if previous, exists := localToUse[binding.Local]; exists && previous.Source != current.Source {
				return fmt.Errorf("target imports for %s import %q and %s import %q both bind local name %q", previous.Owner, previous.Path, owner, current.Path, binding.Local)
			}
			localToUse[binding.Local] = current
		}
		for _, local := range imp.LocalNames {
			local = strings.TrimSpace(local)
			if ignorableImportLocal(local) || renderedLocals[local] {
				continue
			}
			if compilerOwnedSymbols[local] {
				return fmt.Errorf("target imports for %s import %q bind local name %q, which collides with compiler-owned %s symbol", owner, imp.Path, local, target)
			}
			current := importBindingUse{Path: imp.Path, Owner: owner, Source: imp.Path + "\x00local_names\x00" + local}
			if previous, exists := localToUse[local]; exists && previous.Source != current.Source {
				return fmt.Errorf("target imports for %s import %q and %s import %q both bind local name %q", previous.Owner, previous.Path, owner, current.Path, local)
			}
			localToUse[local] = current
		}
		return nil
	}
	for _, imp := range compilerOwnedBindingImports(target) {
		if err := check("compiler-owned runtime", imp); err != nil {
			return err
		}
	}
	for _, imp := range program.Imports {
		if err := check("program", imp); err != nil {
			return err
		}
	}
	for _, global := range program.Globals {
		for _, imp := range global.Imports {
			if err := check(fmt.Sprintf("global %q", global.ID), imp); err != nil {
				return err
			}
		}
	}
	for _, behavior := range program.Behaviors {
		for _, imp := range behavior.Imports {
			if err := check(fmt.Sprintf("behavior %q", behavior.ID), imp); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateTargetImportGlobalDeclarationCollisions(program *Program, target Language) error {
	if program == nil {
		return nil
	}
	declarations := map[string]string{}
	for _, global := range program.Globals {
		if global.Language != target {
			continue
		}
		owner := fmt.Sprintf("target-language global %q", global.ID)
		for _, name := range targetTopLevelDeclarationNames(target, global.Code) {
			if name != "" && declarations[name] == "" {
				declarations[name] = owner
			}
		}
	}
	if len(declarations) == 0 {
		return nil
	}
	check := func(owner string, imp Import) error {
		if imp.Path == "" {
			return nil
		}
		if imp.Language != "" && imp.Language != target {
			return nil
		}
		for _, local := range importGlobalDeclarationCollisionLocals(imp, target) {
			if ignorableImportLocal(local) {
				continue
			}
			if declarationOwner := declarations[local]; declarationOwner != "" {
				return fmt.Errorf("target imports for %s import %q bind local name %q, which collides with %s", owner, imp.Path, local, declarationOwner)
			}
		}
		return nil
	}
	for _, imp := range program.Imports {
		if err := check("program", imp); err != nil {
			return err
		}
	}
	for _, global := range program.Globals {
		for _, imp := range global.Imports {
			if err := check(fmt.Sprintf("global %q", global.ID), imp); err != nil {
				return err
			}
		}
	}
	for _, behavior := range program.Behaviors {
		for _, imp := range behavior.Imports {
			if err := check(fmt.Sprintf("behavior %q", behavior.ID), imp); err != nil {
				return err
			}
		}
	}
	return nil
}

func importGlobalDeclarationCollisionLocals(imp Import, target Language) []string {
	switch target {
	case LanguageCPP:
		return nil
	case LanguageCSharp:
		if csharpImportIsStatic(imp) || imp.Alias == "" {
			return nil
		}
	}
	locals := map[string]bool{}
	var names []string
	add := func(local string) {
		local = strings.TrimSpace(local)
		if local == "" || locals[local] {
			return
		}
		locals[local] = true
		names = append(names, local)
	}
	for _, binding := range importBindings(imp, target) {
		add(binding.Local)
	}
	for _, local := range imp.LocalNames {
		add(local)
	}
	return names
}

func compilerOwnedBindingImports(target Language) []Import {
	switch target {
	case LanguageCSharp:
		return []Import{
			{Path: "Stateforward.Hsm.ConditionChannel", Language: target},
			{Path: "Stateforward.Hsm.Context", Language: target},
			{Path: "Stateforward.Hsm.DurationProvider", Language: target},
			{Path: "Stateforward.Hsm.Event", Language: target},
			{Path: "Stateforward.Hsm.Expression", Language: target},
			{Path: "Stateforward.Hsm.Hsm", Language: target},
			{Path: "Stateforward.Hsm.Instance", Language: target},
			{Path: "Stateforward.Hsm.Model", Language: target},
			{Path: "Stateforward.Hsm.Operation", Language: target},
			{Path: "Stateforward.Hsm.TimeProvider", Language: target},
			{Path: "System", Language: target},
			{Path: "System.Threading", Language: target},
			{Path: "System.Threading.Tasks", Language: target},
		}
	case LanguageCPP:
		return []Import{
			{Path: "hsm/hsm.hpp", Language: target},
			{Path: "chrono", Language: target},
			{Path: "cstdint", Language: target},
			{Path: "string", Language: target},
		}
	case LanguageDart:
		return []Import{
			{
				Path:     "package:hsm/hsm.dart",
				Language: target,
				Specifiers: []ImportSpecifier{
					{Name: "Behavior"},
					{Name: "Context"},
					{Name: "Element"},
					{Name: "Event"},
					{Name: "Hsm"},
					{Name: "Instance"},
					{Name: "Kind"},
					{Name: "Model"},
					{Name: "PartialFunction"},
					{Name: "State"},
					{Name: "Transition"},
					{Name: "Vertex"},
					{Name: "activity"},
					{Name: "after"},
					{Name: "attribute"},
					{Name: "at"},
					{Name: "choice"},
					{Name: "define"},
					{Name: "defer"},
					{Name: "effect"},
					{Name: "entry"},
					{Name: "every"},
					{Name: "exit"},
					{Name: "deepHistory"},
					{Name: "finalState"},
					{Name: "guard"},
					{Name: "initial"},
					{Name: "on"},
					{Name: "onCall"},
					{Name: "onSet"},
					{Name: "operation"},
					{Name: "shallowHistory"},
					{Name: "source"},
					{Name: "state"},
					{Name: "target"},
					{Name: "transition"},
					{Name: "when"},
				},
			},
			{
				Path:       "dart:async",
				Language:   target,
				Specifiers: []ImportSpecifier{{Name: "FutureOr"}, {Name: "Timer"}},
			},
			{
				Path:     "dart:core",
				Language: target,
				Specifiers: []ImportSpecifier{
					{Name: "Object"},
					{Name: "String"},
					{Name: "bool"},
					{Name: "DateTime"},
					{Name: "Duration"},
					{Name: "double"},
					{Name: "Future"},
					{Name: "int"},
					{Name: "num"},
				},
			},
		}
	case LanguageGo:
		return []Import{
			{Path: "builtin", Alias: "bool", Language: target},
			{Path: "builtin", Alias: "false", Language: target},
			{Path: "builtin", Alias: "nil", Language: target},
			{Path: "builtin", Alias: "true", Language: target},
			{Path: "github.com/stateforward/hsm.go", Alias: "hsm", Language: target},
			{Path: "context", Language: target},
			{Path: "time", Language: target},
		}
	case LanguageJava:
		return []Import{
			{Path: "com.stateforward.hsm.Context", Language: target},
			{Path: "com.stateforward.hsm.Event", Language: target},
			{Path: "com.stateforward.hsm.Hsm", Language: target},
			{Path: "com.stateforward.hsm.Instance", Language: target},
			{Path: "com.stateforward.hsm.Model", Language: target},
			{Path: "java.lang.Object", Language: target},
		}
	case LanguageJS:
		return []Import{
			{Path: "@stateforward/hsm", Alias: "hsm", Language: target},
			{Path: "javascript", Language: target, Specifiers: []ImportSpecifier{{Name: "Date"}, {Name: "undefined"}}},
		}
	case LanguageTS:
		return []Import{
			{Path: "@stateforward/hsm.ts", Alias: "hsm", Language: target},
			{Path: "typescript", Language: target, Specifiers: []ImportSpecifier{{Name: "Date"}, {Name: "InstanceType"}, {Name: "Promise"}, {Name: "undefined"}}},
		}
	case LanguagePython:
		return []Import{
			{Path: "builtins", Language: target, Specifiers: []ImportSpecifier{{Name: "False"}, {Name: "None"}, {Name: "True"}, {Name: "bool"}, {Name: "float"}}},
			{Path: "datetime", Language: target},
			{Path: "hsm", Language: target},
			{Path: "typing", Language: target},
		}
	case LanguageRust:
		return []Import{
			{Path: "hsm", Language: target},
			{Path: "std::any", Language: target, Specifiers: []ImportSpecifier{{Name: "Any"}}},
			{Path: "std::boxed", Language: target, Specifiers: []ImportSpecifier{{Name: "Box"}}},
			{Path: "std::future", Language: target, Specifiers: []ImportSpecifier{{Name: "Future"}}},
			{Path: "std::marker", Language: target, Specifiers: []ImportSpecifier{{Name: "Send"}}},
			{Path: "std::pin", Language: target, Specifiers: []ImportSpecifier{{Name: "Pin"}}},
			{Path: "std::time", Language: target, Specifiers: []ImportSpecifier{{Name: "Duration"}}},
		}
	case LanguageZig:
		return []Import{
			{Path: "std", Alias: "std", Language: target},
			{Path: "hsm", Alias: "hsm", Language: target},
		}
	default:
		return nil
	}
}

func cloneImports(imports []Import) []Import {
	if imports == nil {
		return nil
	}
	cloned := make([]Import, 0, len(imports))
	for _, imp := range imports {
		imp.LocalNames = append([]string(nil), imp.LocalNames...)
		imp.Specifiers = append([]ImportSpecifier(nil), imp.Specifiers...)
		imp.Hidden = append([]string(nil), imp.Hidden...)
		cloned = append(cloned, imp)
	}
	return cloned
}

func cloneProgram(program *Program) *Program {
	if program == nil {
		return nil
	}
	cloned := *program
	cloned.Imports = cloneImports(program.Imports)
	cloned.Globals = cloneCodeBlocks(program.Globals)
	cloned.Events = append([]EventDecl(nil), program.Events...)
	cloned.Models = cloneModels(program.Models)
	cloned.Behaviors = cloneBehaviors(program.Behaviors)
	return &cloned
}

func cloneAdapterRequest(request AdapterRequest) AdapterRequest {
	request.Imports = cloneImports(request.Imports)
	request.Globals = cloneCodeBlocks(request.Globals)
	request.Events = append([]EventDecl(nil), request.Events...)
	request.Models = cloneModelViews(request.Models)
	request.Behaviors = cloneBehaviors(request.Behaviors)
	return request
}

func cloneModelViews(views []ModelView) []ModelView {
	if views == nil {
		return nil
	}
	cloned := make([]ModelView, 0, len(views))
	for _, view := range views {
		view.InitialEffects = append([]BehaviorRef(nil), view.InitialEffects...)
		view.Events = append([]EventView(nil), view.Events...)
		view.Attributes = append([]AttributeView(nil), view.Attributes...)
		view.Operations = cloneOperationViews(view.Operations)
		view.States = cloneStateViews(view.States)
		view.Transitions = cloneTransitionViews(view.Transitions)
		cloned = append(cloned, view)
	}
	return cloned
}

func cloneOperationViews(views []OperationView) []OperationView {
	if views == nil {
		return nil
	}
	cloned := make([]OperationView, 0, len(views))
	for _, view := range views {
		view.Behavior = cloneBehaviorRef(view.Behavior)
		cloned = append(cloned, view)
	}
	return cloned
}

func cloneStateViews(views []StateView) []StateView {
	if views == nil {
		return nil
	}
	cloned := make([]StateView, 0, len(views))
	for _, view := range views {
		view.InitialEffects = append([]BehaviorRef(nil), view.InitialEffects...)
		view.Behaviors = append([]BehaviorRef(nil), view.Behaviors...)
		view.Defers = append([]string(nil), view.Defers...)
		view.States = cloneStateViews(view.States)
		view.Transitions = cloneTransitionViews(view.Transitions)
		cloned = append(cloned, view)
	}
	return cloned
}

func cloneTransitionViews(views []TransitionView) []TransitionView {
	if views == nil {
		return nil
	}
	cloned := make([]TransitionView, 0, len(views))
	for _, view := range views {
		view.Trigger = cloneTrigger(view.Trigger)
		view.Guard = cloneBehaviorRef(view.Guard)
		view.Effects = append([]BehaviorRef(nil), view.Effects...)
		cloned = append(cloned, view)
	}
	return cloned
}

func cloneModels(models []Model) []Model {
	if models == nil {
		return nil
	}
	cloned := make([]Model, 0, len(models))
	for _, model := range models {
		model.Initializer = cloneInitial(model.Initializer)
		model.Attributes = append([]Attribute(nil), model.Attributes...)
		model.Operations = cloneOperations(model.Operations)
		model.States = cloneStates(model.States)
		model.Transitions = cloneTransitions(model.Transitions)
		cloned = append(cloned, model)
	}
	return cloned
}

func cloneOperations(operations []Operation) []Operation {
	if operations == nil {
		return nil
	}
	cloned := make([]Operation, 0, len(operations))
	for _, operation := range operations {
		operation.Behavior = cloneBehaviorRef(operation.Behavior)
		cloned = append(cloned, operation)
	}
	return cloned
}

func cloneStates(states []State) []State {
	if states == nil {
		return nil
	}
	cloned := make([]State, 0, len(states))
	for _, state := range states {
		state.Initializer = cloneInitial(state.Initializer)
		state.States = cloneStates(state.States)
		state.Transitions = cloneTransitions(state.Transitions)
		state.Behaviors = append([]BehaviorRef(nil), state.Behaviors...)
		state.Defers = append([]string(nil), state.Defers...)
		cloned = append(cloned, state)
	}
	return cloned
}

func cloneInitial(initial *Initial) *Initial {
	if initial == nil {
		return nil
	}
	cloned := *initial
	cloned.Effects = append([]BehaviorRef(nil), initial.Effects...)
	return &cloned
}

func cloneTransitions(transitions []Transition) []Transition {
	if transitions == nil {
		return nil
	}
	cloned := make([]Transition, 0, len(transitions))
	for _, transition := range transitions {
		transition.Trigger = cloneTrigger(transition.Trigger)
		transition.Guard = cloneBehaviorRef(transition.Guard)
		transition.Effects = append([]BehaviorRef(nil), transition.Effects...)
		cloned = append(cloned, transition)
	}
	return cloned
}

func cloneCodeBlocks(blocks []CodeBlock) []CodeBlock {
	if blocks == nil {
		return nil
	}
	cloned := make([]CodeBlock, 0, len(blocks))
	for _, block := range blocks {
		block.Imports = cloneImports(block.Imports)
		cloned = append(cloned, block)
	}
	return cloned
}

func cloneBehaviors(behaviors []Behavior) []Behavior {
	if behaviors == nil {
		return nil
	}
	cloned := make([]Behavior, 0, len(behaviors))
	for _, behavior := range behaviors {
		behavior.Imports = cloneImports(behavior.Imports)
		behavior.TargetABI = cloneBehaviorABI(behavior.TargetABI)
		cloned = append(cloned, behavior)
	}
	return cloned
}

func cloneBehaviorABI(abi *BehaviorABI) *BehaviorABI {
	if abi == nil {
		return nil
	}
	cloned := *abi
	cloned.Parameters = append([]ABIParameter(nil), abi.Parameters...)
	return &cloned
}

func importPaths(imports []Import) []string {
	paths := make([]string, 0, len(imports))
	for _, imp := range imports {
		paths = append(paths, imp.Path)
	}
	return paths
}

func globalIDs(globals []CodeBlock) []string {
	ids := make([]string, 0, len(globals))
	for _, block := range globals {
		ids = append(ids, block.ID)
	}
	return ids
}

func codeBlockIDSet(blocks []CodeBlock) map[string]bool {
	ids := make(map[string]bool, len(blocks))
	for _, block := range blocks {
		ids[block.ID] = true
	}
	return ids
}

func behaviorIDs(behaviors []Behavior) []string {
	ids := make([]string, 0, len(behaviors))
	for _, behavior := range behaviors {
		ids = append(ids, behavior.ID)
	}
	return ids
}

func behaviorIDSet(behaviors []Behavior) map[string]bool {
	ids := make(map[string]bool, len(behaviors))
	for _, behavior := range behaviors {
		ids[behavior.ID] = true
	}
	return ids
}

func codeBlockImportRegions(blocks []CodeBlock) []ImportRegion {
	regions := make([]ImportRegion, 0, len(blocks))
	for _, block := range blocks {
		if len(block.Imports) == 0 {
			continue
		}
		regions = append(regions, ImportRegion{ID: block.ID, ImportPaths: importPaths(block.Imports), Imports: cloneImports(block.Imports)})
	}
	return regions
}

func behaviorImportRegions(behaviors []Behavior) []ImportRegion {
	regions := make([]ImportRegion, 0, len(behaviors))
	for _, behavior := range behaviors {
		if len(behavior.Imports) == 0 {
			continue
		}
		regions = append(regions, ImportRegion{ID: behavior.ID, ImportPaths: importPaths(behavior.Imports), Imports: cloneImports(behavior.Imports)})
	}
	return regions
}

func sortLanguages(languages []Language) {
	order := map[Language]int{}
	for index, language := range SupportedLanguages() {
		order[language] = index
	}
	sort.Slice(languages, func(i, j int) bool {
		left, leftKnown := order[languages[i]]
		right, rightKnown := order[languages[j]]
		if leftKnown && rightKnown {
			return left < right
		}
		if leftKnown != rightKnown {
			return leftKnown
		}
		return languages[i] < languages[j]
	})
}

type IdentityAdapter struct{}

func (IdentityAdapter) Name() string { return "none" }

func (IdentityAdapter) Translate(context.Context, AdapterRequest) (*BehaviorPatch, error) {
	return nil, nil
}
