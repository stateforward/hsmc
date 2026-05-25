package hsmc

// Language identifies a source or target implementation language.
type Language string

const (
	LanguageCSharp   Language = "csharp"
	LanguageCPP      Language = "cpp"
	LanguageDart     Language = "dart"
	LanguageGo       Language = "go"
	LanguageJava     Language = "java"
	LanguageJS       Language = "javascript"
	LanguageJSONIR   Language = "json-ir"
	LanguageMermaid  Language = "mermaid"
	LanguagePlantUML Language = "plantuml"
	LanguagePython   Language = "python"
	LanguageRust     Language = "rust"
	LanguageTS       Language = "typescript"
	LanguageZig      Language = "zig"
)

// SourceRange ties IR nodes back to their parsed source. Byte offsets are
// stable enough for adapter prompts and diagnostics without binding the IR to
// a particular parser node representation.
type SourceRange struct {
	File      string `json:"file,omitempty"`
	StartByte uint   `json:"start_byte"`
	EndByte   uint   `json:"end_byte"`
	StartLine uint   `json:"start_line"`
	EndLine   uint   `json:"end_line"`
}

// Program is the canonical handoff between frontends, semantic passes,
// backends, and behavior adapters.
type Program struct {
	SourceLanguage Language    `json:"source_language"`
	PackageName    string      `json:"package_name,omitempty"`
	Imports        []Import    `json:"imports,omitempty"`
	Globals        []CodeBlock `json:"globals,omitempty"`
	Events         []EventDecl `json:"events,omitempty"`
	Models         []Model     `json:"models"`
	Behaviors      []Behavior  `json:"behaviors,omitempty"`
}

// Import preserves source-level import dependencies separately from behavior
// code so target backends can decide which imports are structural and which
// are behavior-only.
type Import struct {
	Path       string            `json:"path"`
	Alias      string            `json:"alias,omitempty"`
	Default    string            `json:"default,omitempty"`
	Specifiers []ImportSpecifier `json:"specifiers,omitempty"`
	Hidden     []string          `json:"hidden,omitempty"`
	Static     bool              `json:"static,omitempty"`
	Deferred   bool              `json:"deferred,omitempty"`
	TypeOnly   bool              `json:"type_only,omitempty"`
	LocalNames []string          `json:"local_names,omitempty"`
	Language   Language          `json:"language,omitempty"`
	Range      SourceRange       `json:"range,omitempty"`
}

type ImportSpecifier struct {
	Name     string `json:"name"`
	Alias    string `json:"alias,omitempty"`
	TypeOnly bool   `json:"type_only,omitempty"`
}

// CodeBlock is mutable adapter territory. The compiler owns models; adapters
// may revise globals and behavior bodies to make generated code compile in the
// target language.
type CodeBlock struct {
	ID       string      `json:"id"`
	Language Language    `json:"language"`
	Code     string      `json:"code"`
	Imports  []Import    `json:"imports,omitempty"`
	Range    SourceRange `json:"range,omitempty"`
}

type EventDecl struct {
	ID       string      `json:"id"`
	Symbol   string      `json:"symbol"`
	Name     string      `json:"name"`
	Schema   string      `json:"schema,omitempty"`
	Language Language    `json:"language,omitempty"`
	Range    SourceRange `json:"range,omitempty"`
}

// Model is the language-neutral representation of HSM structure.
type Model struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Initializer *Initial     `json:"initial,omitempty"`
	Attributes  []Attribute  `json:"attributes,omitempty"`
	Operations  []Operation  `json:"operations,omitempty"`
	States      []State      `json:"states,omitempty"`
	Transitions []Transition `json:"transitions,omitempty"`
	Range       SourceRange  `json:"range,omitempty"`
}

type Attribute struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Type       string      `json:"type,omitempty"`
	HasDefault bool        `json:"has_default,omitempty"`
	Default    string      `json:"default,omitempty"`
	Language   Language    `json:"language,omitempty"`
	Range      SourceRange `json:"range,omitempty"`
}

type Operation struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Behavior *BehaviorRef `json:"behavior,omitempty"`
	Range    SourceRange  `json:"range,omitempty"`
}

type StateKind string

const (
	StateKindState          StateKind = "state"
	StateKindFinal          StateKind = "final"
	StateKindChoice         StateKind = "choice"
	StateKindShallowHistory StateKind = "shallow_history"
	StateKindDeepHistory    StateKind = "deep_history"
)

// State represents a state or pseudostate in the normalized model.
type State struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Kind        StateKind     `json:"kind"`
	Initializer *Initial      `json:"initial,omitempty"`
	States      []State       `json:"states,omitempty"`
	Transitions []Transition  `json:"transitions,omitempty"`
	Behaviors   []BehaviorRef `json:"behaviors,omitempty"`
	Defers      []string      `json:"defers,omitempty"`
	Range       SourceRange   `json:"range,omitempty"`
}

type Initial struct {
	OwnerID string        `json:"owner_id,omitempty"`
	ID      string        `json:"id"`
	Target  string        `json:"target,omitempty"`
	Effects []BehaviorRef `json:"effects,omitempty"`
	Range   SourceRange   `json:"range,omitempty"`
}

type Transition struct {
	OwnerID string        `json:"owner_id,omitempty"`
	ID      string        `json:"id"`
	Source  string        `json:"source,omitempty"`
	Target  string        `json:"target,omitempty"`
	Trigger *Trigger      `json:"trigger,omitempty"`
	Guard   *BehaviorRef  `json:"guard,omitempty"`
	Effects []BehaviorRef `json:"effects,omitempty"`
	Range   SourceRange   `json:"range,omitempty"`
}

type TriggerKind string

const (
	TriggerOn     TriggerKind = "on"
	TriggerOnSet  TriggerKind = "on_set"
	TriggerOnCall TriggerKind = "on_call"
	TriggerAfter  TriggerKind = "after"
	TriggerEvery  TriggerKind = "every"
	TriggerAt     TriggerKind = "at"
	TriggerWhen   TriggerKind = "when"
)

type Trigger struct {
	Kind     TriggerKind    `json:"kind"`
	Value    string         `json:"value"`
	IsString bool           `json:"is_string,omitempty"`
	Values   []TriggerValue `json:"values,omitempty"`
	Language Language       `json:"language,omitempty"`
	Expr     *BehaviorRef   `json:"expr,omitempty"`
}

type TriggerValue struct {
	Value       string   `json:"value"`
	IsString    bool     `json:"is_string,omitempty"`
	EventSymbol string   `json:"event_symbol,omitempty"`
	Language    Language `json:"language,omitempty"`
}

type BehaviorKind string

const (
	BehaviorEntry     BehaviorKind = "entry"
	BehaviorExit      BehaviorKind = "exit"
	BehaviorActivity  BehaviorKind = "activity"
	BehaviorGuard     BehaviorKind = "guard"
	BehaviorEffect    BehaviorKind = "effect"
	BehaviorTrigger   BehaviorKind = "trigger"
	BehaviorOperation BehaviorKind = "operation"
)

type BehaviorRef struct {
	ID        string       `json:"id,omitempty"`
	Kind      BehaviorKind `json:"kind"`
	Operation string       `json:"operation,omitempty"`
}

// Behavior carries source context plus compiler-owned structural bindings.
// Adapter patches may edit only Body, Code, and Imports selected by ID.
type Behavior struct {
	ID             string       `json:"id"`
	OwnerID        string       `json:"owner_id"`
	Kind           BehaviorKind `json:"kind"`
	TriggerKind    TriggerKind  `json:"trigger_kind,omitempty"`
	Inline         bool         `json:"inline,omitempty"`
	SourceLanguage Language     `json:"source_language"`
	TargetLanguage Language     `json:"target_language,omitempty"`
	TargetABI      *BehaviorABI `json:"target_abi,omitempty"`
	Signature      string       `json:"signature,omitempty"`
	Body           string       `json:"body,omitempty"`
	Code           string       `json:"code,omitempty"`
	Imports        []Import     `json:"imports,omitempty"`
	Range          SourceRange  `json:"range,omitempty"`
}

type BehaviorABI struct {
	Language   Language       `json:"language"`
	Signature  string         `json:"signature"`
	ReturnType string         `json:"return_type,omitempty"`
	Parameters []ABIParameter `json:"parameters,omitempty"`
}

type ABIParameter struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// BehaviorPatch is the only mutation contract adapters receive. ImportsSet
// distinguishes omitted JSON imports from an explicit empty import list; Go
// adapters can also use a non-nil empty Imports slice to clear target imports.
type BehaviorPatch struct {
	Imports    []Import    `json:"imports,omitempty"`
	Globals    []CodeBlock `json:"globals,omitempty"`
	Behaviors  []Behavior  `json:"behaviors,omitempty"`
	ImportsSet bool        `json:"-"`
}

// AdapterProtocol carries enough metadata for an external agent to translate
// mutable code regions without receiving authority over model structure.
type AdapterProtocol struct {
	Version      string          `json:"version"`
	Instruction  string          `json:"instruction"`
	Guidance     AdapterGuidance `json:"guidance"`
	Response     AdapterResponse `json:"response"`
	AllowedEdits AllowedEdits    `json:"allowed_edits"`
	Request      AdapterRequest  `json:"request"`
}

type AdapterGuidance struct {
	TargetLanguage        Language         `json:"target_language"`
	CompilerOwnedImports  []string         `json:"compiler_owned_imports,omitempty"`
	CompilerOwnedBindings []string         `json:"compiler_owned_bindings,omitempty"`
	CompilerOwnedSymbols  []string         `json:"compiler_owned_symbols,omitempty"`
	EditableFields        []string         `json:"editable_fields,omitempty"`
	ForbiddenFields       []string         `json:"forbidden_fields,omitempty"`
	TargetImportShape     ImportShapeRules `json:"target_import_shape"`
}

type ImportShapeRules struct {
	Language                  Language `json:"language"`
	SupportsBareImport        bool     `json:"supports_bare_import"`
	SupportsDefaultImport     bool     `json:"supports_default_import"`
	SupportsNamedSpecifiers   bool     `json:"supports_named_specifiers"`
	SupportsHiddenSpecifiers  bool     `json:"supports_hidden_specifiers,omitempty"`
	SupportsNamespaceAlias    bool     `json:"supports_namespace_alias"`
	SupportsModuleAlias       bool     `json:"supports_module_alias"`
	SupportsStaticImport      bool     `json:"supports_static_import,omitempty"`
	SupportsDeferredImport    bool     `json:"supports_deferred_import,omitempty"`
	SupportsTypeOnlyImport    bool     `json:"supports_type_only_import,omitempty"`
	SupportsSpecifierTypeOnly bool     `json:"supports_specifier_type_only,omitempty"`
	DisallowAliasWithMembers  bool     `json:"disallow_alias_with_members,omitempty"`
	Notes                     []string `json:"notes,omitempty"`
}

type AdapterResponse struct {
	OutputFormat           string   `json:"output_format"`
	TopLevelFields         []string `json:"top_level_fields"`
	ImportFields           []string `json:"import_fields"`
	ImportSpecifierFields  []string `json:"import_specifier_fields"`
	GlobalFields           []string `json:"global_fields"`
	BehaviorFields         []string `json:"behavior_fields"`
	NullPolicy             string   `json:"null_policy"`
	ScalarPolicy           string   `json:"scalar_policy"`
	UnknownFieldPolicy     string   `json:"unknown_field_policy"`
	ExplicitEmptyListUsage string   `json:"explicit_empty_list_usage"`
}

// ModelView is read-only adapter context. It describes compiler-owned model
// structure without making model nodes part of the adapter patch contract.
type ModelView struct {
	ID             string           `json:"id"`
	Name           string           `json:"name"`
	Path           string           `json:"path"`
	InitialTarget  string           `json:"initial_target,omitempty"`
	InitialEffects []BehaviorRef    `json:"initial_effects,omitempty"`
	Events         []EventView      `json:"events,omitempty"`
	Attributes     []AttributeView  `json:"attributes,omitempty"`
	Operations     []OperationView  `json:"operations,omitempty"`
	States         []StateView      `json:"states,omitempty"`
	Transitions    []TransitionView `json:"transitions,omitempty"`
}

type EventView struct {
	ID       string   `json:"id"`
	Symbol   string   `json:"symbol"`
	Name     string   `json:"name"`
	Schema   string   `json:"schema,omitempty"`
	Language Language `json:"language,omitempty"`
}

type AttributeView struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Type       string   `json:"type,omitempty"`
	HasDefault bool     `json:"has_default,omitempty"`
	Default    string   `json:"default,omitempty"`
	Language   Language `json:"language,omitempty"`
}

type OperationView struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Behavior *BehaviorRef `json:"behavior,omitempty"`
}

type StateView struct {
	ID             string           `json:"id"`
	Name           string           `json:"name"`
	Path           string           `json:"path"`
	Kind           StateKind        `json:"kind"`
	InitialTarget  string           `json:"initial_target,omitempty"`
	InitialEffects []BehaviorRef    `json:"initial_effects,omitempty"`
	Behaviors      []BehaviorRef    `json:"behaviors,omitempty"`
	Defers         []string         `json:"defers,omitempty"`
	States         []StateView      `json:"states,omitempty"`
	Transitions    []TransitionView `json:"transitions,omitempty"`
}

type TransitionView struct {
	ID      string        `json:"id"`
	OwnerID string        `json:"owner_id,omitempty"`
	Source  string        `json:"source,omitempty"`
	Target  string        `json:"target,omitempty"`
	Trigger *Trigger      `json:"trigger,omitempty"`
	Guard   *BehaviorRef  `json:"guard,omitempty"`
	Effects []BehaviorRef `json:"effects,omitempty"`
}

type AllowedEdits struct {
	ImportPaths     []string       `json:"import_paths,omitempty"`
	GlobalIDs       []string       `json:"global_ids,omitempty"`
	BehaviorIDs     []string       `json:"behavior_ids,omitempty"`
	GlobalImports   []ImportRegion `json:"global_imports,omitempty"`
	BehaviorImports []ImportRegion `json:"behavior_imports,omitempty"`
}

type ImportRegion struct {
	ID          string   `json:"id"`
	ImportPaths []string `json:"import_paths,omitempty"`
	Imports     []Import `json:"imports,omitempty"`
}
