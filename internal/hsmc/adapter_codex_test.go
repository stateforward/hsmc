package hsmc

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestNewAdapterProtocolRestrictsEditableIDs(t *testing.T) {
	request := AdapterRequest{
		SourceLanguage: LanguageGo,
		TargetLanguage: LanguageTS,
		Imports:        []Import{{Path: "fmt", Language: LanguageGo}},
		Globals:        []CodeBlock{{ID: "global_1", Language: LanguageGo, Code: "func helper() {}", Imports: []Import{{Path: "strings", Language: LanguageGo}}}},
		Models:         []ModelView{{ID: "model_1", Name: "Door", Path: "/Door"}},
		Behaviors:      []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo, Imports: []Import{{Path: "context", Language: LanguageGo}}}},
	}
	protocol := NewAdapterProtocol(request)
	if protocol.Version != "hsmc.adapter.v1" {
		t.Fatalf("version = %q", protocol.Version)
	}
	if !strings.Contains(protocol.Instruction, "Do not create, remove, rename, reorder, or modify models") ||
		!strings.Contains(protocol.Instruction, "The compiler translates HSM structure") ||
		!strings.Contains(protocol.Instruction, "use provided imports as source context") ||
		!strings.Contains(protocol.Instruction, "behavior implementation code") ||
		!strings.Contains(protocol.Instruction, "Do not attempt to compile or rewrite arbitrary behavior semantics through model edits") ||
		!strings.Contains(protocol.Instruction, "behavior and global body translation is the adapter's only semantic-code responsibility") ||
		!strings.Contains(protocol.Instruction, "events, attributes, operations") ||
		!strings.Contains(protocol.Instruction, "initial transitions") ||
		!strings.Contains(protocol.Instruction, "defers") ||
		!strings.Contains(protocol.Instruction, "signatures, ABIs") ||
		!strings.Contains(protocol.Instruction, "DSL runtime/helper globals such as Config, Queue, Clock, MakeGroup, MakeKind, IsKind, and TakeSnapshot are translation context only") ||
		!strings.Contains(protocol.Instruction, "Top-level imports in the request are source context") ||
		!strings.Contains(protocol.Instruction, "must not include compiler-owned runtime imports") ||
		!strings.Contains(protocol.Instruction, "collide with compiler-owned runtime bindings") ||
		!strings.Contains(protocol.Instruction, "bind names listed in guidance.compiler_owned_symbols") ||
		!strings.Contains(protocol.Instruction, "bind names declared by returned target-language globals") ||
		!strings.Contains(protocol.Instruction, "helper globals must use adapter_global_ IDs with only letters, digits, and underscores after the prefix") ||
		!strings.Contains(protocol.Instruction, "Translated globals and adapter helper globals must not declare names listed in guidance.compiler_owned_symbols or guidance.compiler_owned_bindings") ||
		!strings.Contains(protocol.Instruction, "Use behaviors[].body for translated statements inside the compiler-owned behavior signature") ||
		!strings.Contains(protocol.Instruction, "use behaviors[].code only for a single complete target-language callable with the compiler-owned behavior name and no extra top-level declarations") ||
		!strings.Contains(protocol.Instruction, "do not return target imports for untranslated foreign code") ||
		!strings.Contains(protocol.Instruction, "preserve untranslated behavior source as target-language comments inside generated target behavior bodies") ||
		!strings.Contains(protocol.Instruction, "untranslated globals as target-language comments near top-level output") ||
		!strings.Contains(protocol.Instruction, "unknown fields and null array/scalar fields are rejected") {
		t.Fatalf("instruction missing model restriction: %s", protocol.Instruction)
	}
	if protocol.Guidance.TargetLanguage != LanguageTS {
		t.Fatalf("guidance target language = %q, want typescript", protocol.Guidance.TargetLanguage)
	}
	if got := strings.Join(protocol.Guidance.CompilerOwnedImports, ","); got != "@stateforward/hsm.ts,@stateforward/hsm,typescript,typescript.Date,typescript.InstanceType,typescript.Promise,typescript.undefined" {
		t.Fatalf("compiler-owned imports = %q", got)
	}
	if got := strings.Join(protocol.Guidance.CompilerOwnedBindings, ","); got != "Date,InstanceType,Promise,hsm,undefined" {
		t.Fatalf("compiler-owned bindings = %q", got)
	}
	if got := strings.Join(protocol.Guidance.CompilerOwnedSymbols, ","); got != "DoorModel,entry1" {
		t.Fatalf("compiler-owned symbols = %q", got)
	}
	if !protocol.Guidance.TargetImportShape.SupportsNamedSpecifiers || !protocol.Guidance.TargetImportShape.SupportsNamespaceAlias {
		t.Fatalf("target import shape missing TypeScript capabilities: %#v", protocol.Guidance.TargetImportShape)
	}
	if !containsString(protocol.Guidance.EditableFields, "behaviors[].body (translated statements inside the compiler-owned behavior signature)") ||
		!containsString(protocol.Guidance.EditableFields, "behaviors[].code (single complete target-language callable using the compiler-owned behavior name, with no extra top-level declarations)") ||
		!containsString(protocol.Guidance.EditableFields, "imports (target-language program imports for translated code only)") ||
		!containsString(protocol.Guidance.ForbiddenFields, "attributes") ||
		!containsString(protocol.Guidance.ForbiddenFields, "operations") ||
		!containsString(protocol.Guidance.ForbiddenFields, "initial") ||
		!containsString(protocol.Guidance.ForbiddenFields, "defers") ||
		!containsString(protocol.Guidance.ForbiddenFields, "globals[].language") ||
		!containsString(protocol.Guidance.ForbiddenFields, "imports[].local_names") ||
		!containsString(protocol.Guidance.ForbiddenFields, "imports[].range") ||
		!containsString(protocol.Guidance.ForbiddenFields, "imports[].specifiers[].range") ||
		!containsString(protocol.Guidance.ForbiddenFields, "globals[].imports[].local_names") ||
		!containsString(protocol.Guidance.ForbiddenFields, "behaviors[].imports[].local_names") ||
		!containsString(protocol.Guidance.ForbiddenFields, "global declarations that collide with compiler-owned symbols") ||
		!containsString(protocol.Guidance.ForbiddenFields, "global declarations that collide with compiler-owned runtime bindings") ||
		!containsString(protocol.Guidance.ForbiddenFields, "imports that bind compiler-owned symbols") ||
		!containsString(protocol.Guidance.ForbiddenFields, "imports that bind compiler-owned runtime bindings") ||
		!containsString(protocol.Guidance.ForbiddenFields, "imports that bind names declared by returned target-language globals") ||
		!containsString(protocol.Guidance.ForbiddenFields, "global ID changes (id is required only as a selector; helper globals must use adapter_global_ IDs)") ||
		!containsString(protocol.Guidance.ForbiddenFields, "behavior ID changes (id is required only as a selector)") ||
		!containsString(protocol.Guidance.ForbiddenFields, "behaviors[].inline") ||
		!containsString(protocol.Guidance.ForbiddenFields, "behaviors[].target_language") ||
		!containsString(protocol.Guidance.ForbiddenFields, "behaviors[].target_abi") {
		t.Fatalf("guidance fields = %#v / %#v", protocol.Guidance.EditableFields, protocol.Guidance.ForbiddenFields)
	}
	if containsString(protocol.Guidance.ForbiddenFields, "behaviors[].id") {
		t.Fatalf("guidance should not forbid the behavior id selector: %#v", protocol.Guidance.ForbiddenFields)
	}
	if protocol.Response.OutputFormat == "" || !containsString(protocol.Response.TopLevelFields, "behaviors") || !containsString(protocol.Response.ImportFields, "specifiers") {
		t.Fatalf("response contract = %#v", protocol.Response)
	}
	if containsString(protocol.Response.BehaviorFields, "target_language") || containsString(protocol.Response.BehaviorFields, "owner_id") {
		t.Fatalf("behavior response fields = %#v", protocol.Response.BehaviorFields)
	}
	if containsString(protocol.Response.GlobalFields, "language") {
		t.Fatalf("global response fields = %#v", protocol.Response.GlobalFields)
	}
	if !strings.Contains(protocol.Response.NullPolicy, "Omit fields") ||
		!strings.Contains(protocol.Response.ScalarPolicy, "Behavior body is for translated statements inside the compiler-owned signature") ||
		!strings.Contains(protocol.Response.ScalarPolicy, "behavior code is for a single complete target-language callable implementation using the compiler-owned name and no extra top-level declarations") ||
		!strings.Contains(protocol.Response.ScalarPolicy, "whitespace-only code/body fields are rejected") ||
		!strings.Contains(protocol.Response.UnknownFieldPolicy, "Unknown") ||
		!strings.Contains(protocol.Response.UnknownFieldPolicy, "attributes, operations") ||
		!strings.Contains(protocol.Response.UnknownFieldPolicy, "initial transitions") ||
		!strings.Contains(protocol.Response.UnknownFieldPolicy, "defers") ||
		!strings.Contains(protocol.Response.ExplicitEmptyListUsage, "Non-empty program imports are accepted only with translated target global or behavior code in the same patch") ||
		!strings.Contains(protocol.Response.ExplicitEmptyListUsage, "Target imports must not bind compiler-owned symbols listed in guidance.compiler_owned_symbols") ||
		!strings.Contains(protocol.Response.ExplicitEmptyListUsage, "compiler-owned runtime bindings listed in guidance.compiler_owned_bindings") ||
		!strings.Contains(protocol.Response.ExplicitEmptyListUsage, "names declared by returned target-language globals") ||
		!strings.Contains(protocol.Response.ExplicitEmptyListUsage, "Omit a global or behavior entirely when you cannot translate it") ||
		!strings.Contains(protocol.Response.ExplicitEmptyListUsage, "Import-only global/behavior patches are accepted only for code that is already in the target language") {
		t.Fatalf("response policies = %#v", protocol.Response)
	}
	if got := strings.Join(protocol.AllowedEdits.GlobalIDs, ","); got != "global_1" {
		t.Fatalf("global IDs = %q", got)
	}
	if got := strings.Join(protocol.AllowedEdits.BehaviorIDs, ","); got != "entry_1" {
		t.Fatalf("behavior IDs = %q", got)
	}
	if len(protocol.AllowedEdits.GlobalImports) != 1 || protocol.AllowedEdits.GlobalImports[0].ID != "global_1" || strings.Join(protocol.AllowedEdits.GlobalImports[0].ImportPaths, ",") != "strings" {
		t.Fatalf("global import regions = %#v", protocol.AllowedEdits.GlobalImports)
	}
	if len(protocol.AllowedEdits.BehaviorImports) != 1 || protocol.AllowedEdits.BehaviorImports[0].ID != "entry_1" || strings.Join(protocol.AllowedEdits.BehaviorImports[0].ImportPaths, ",") != "context" {
		t.Fatalf("behavior import regions = %#v", protocol.AllowedEdits.BehaviorImports)
	}
	if len(protocol.Request.Models) != 1 || protocol.Request.Models[0].Name != "Door" {
		t.Fatalf("model context = %#v", protocol.Request.Models)
	}
}

func TestAdapterProtocolRustGuidanceDisallowsAliasWithSpecifiers(t *testing.T) {
	protocol := NewAdapterProtocol(AdapterRequest{TargetLanguage: LanguageRust})
	shape := protocol.Guidance.TargetImportShape
	if !shape.SupportsNamedSpecifiers || !shape.SupportsModuleAlias || !shape.DisallowAliasWithMembers {
		t.Fatalf("Rust target import shape should allow aliases and specifiers separately, but reject combining them: %#v", shape)
	}
	if !containsString(shape.Notes, "Default imports are rejected; do not combine a module alias with named specifiers.") {
		t.Fatalf("Rust target import shape notes should warn about alias/specifier combinations: %#v", shape.Notes)
	}
}

func TestAdapterProtocolGuidanceAdvertisesTargetImportMetadataFlags(t *testing.T) {
	cases := []struct {
		name        string
		target      Language
		check       func(ImportShapeRules) bool
		wantNote    string
		forbidFlags func(ImportShapeRules) bool
	}{
		{
			name:   "csharp static",
			target: LanguageCSharp,
			check: func(shape ImportShapeRules) bool {
				return shape.SupportsStaticImport && shape.SupportsModuleAlias
			},
			wantNote: "Do not combine static with alias; C# static usings cannot be alias usings.",
			forbidFlags: func(shape ImportShapeRules) bool {
				return shape.SupportsDeferredImport || shape.SupportsTypeOnlyImport || shape.SupportsSpecifierTypeOnly
			},
		},
		{
			name:   "java static",
			target: LanguageJava,
			check: func(shape ImportShapeRules) bool {
				return shape.SupportsStaticImport && shape.SupportsBareImport
			},
			wantNote: "Use Java package or class imports such as java.util.List or com.example.Helper; set static for Java static imports such as com.example.Helper.record.",
			forbidFlags: func(shape ImportShapeRules) bool {
				return shape.SupportsDeferredImport || shape.SupportsTypeOnlyImport || shape.SupportsSpecifierTypeOnly
			},
		},
		{
			name:   "dart deferred",
			target: LanguageDart,
			check: func(shape ImportShapeRules) bool {
				return shape.SupportsDeferredImport && shape.SupportsModuleAlias && shape.SupportsNamedSpecifiers && shape.DisallowAliasWithMembers
			},
			wantNote: "Do not combine a Dart prefix alias with named specifiers in translated target imports; use either a prefix import or a show import.",
			forbidFlags: func(shape ImportShapeRules) bool {
				return shape.SupportsStaticImport || shape.SupportsTypeOnlyImport || shape.SupportsSpecifierTypeOnly
			},
		},
		{
			name:   "typescript type-only",
			target: LanguageTS,
			check: func(shape ImportShapeRules) bool {
				return shape.SupportsTypeOnlyImport && shape.SupportsSpecifierTypeOnly && shape.SupportsNamedSpecifiers
			},
			wantNote: "Type-only imports and type-only named specifiers are supported only for TypeScript.",
			forbidFlags: func(shape ImportShapeRules) bool {
				return shape.SupportsStaticImport || shape.SupportsDeferredImport
			},
		},
		{
			name:   "javascript not type-only",
			target: LanguageJS,
			check: func(shape ImportShapeRules) bool {
				return shape.SupportsNamedSpecifiers && !shape.SupportsTypeOnlyImport && !shape.SupportsSpecifierTypeOnly
			},
			wantNote: "Type-only imports and type-only named specifiers are supported only for TypeScript.",
			forbidFlags: func(shape ImportShapeRules) bool {
				return shape.SupportsStaticImport || shape.SupportsDeferredImport
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			protocol := NewAdapterProtocol(AdapterRequest{TargetLanguage: tc.target})
			shape := protocol.Guidance.TargetImportShape
			if !tc.check(shape) {
				t.Fatalf("target import shape = %#v", shape)
			}
			if tc.forbidFlags != nil && tc.forbidFlags(shape) {
				t.Fatalf("target import shape advertises unsupported metadata flags: %#v", shape)
			}
			if !containsString(shape.Notes, tc.wantNote) {
				t.Fatalf("target import shape notes missing %q: %#v", tc.wantNote, shape.Notes)
			}
		})
	}
}

func TestAdapterResponseContractMatchesAcceptedPatchFields(t *testing.T) {
	contract := adapterResponseContract()
	cases := []struct {
		name string
		got  []string
		want []string
	}{
		{
			name: "top level",
			got:  contract.TopLevelFields,
			want: []string{"imports", "globals", "behaviors"},
		},
		{
			name: "imports",
			got:  contract.ImportFields,
			want: []string{"path", "alias", "default", "specifiers", "hidden", "static", "deferred", "type_only", "language"},
		},
		{
			name: "import specifiers",
			got:  contract.ImportSpecifierFields,
			want: []string{"name", "alias", "type_only"},
		},
		{
			name: "globals",
			got:  contract.GlobalFields,
			want: []string{"id", "code", "imports"},
		},
		{
			name: "behaviors",
			got:  contract.BehaviorFields,
			want: []string{"id", "body", "code", "imports"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !reflect.DeepEqual(tc.got, tc.want) {
				t.Fatalf("%s fields = %#v, want %#v", tc.name, tc.got, tc.want)
			}
		})
	}
}

func TestParseBehaviorPatchAcceptsPlainAndFencedJSON(t *testing.T) {
	plain, err := ParseBehaviorPatch([]byte(`{"behaviors":[{"id":"entry_1","body":"return;"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(plain.Behaviors) != 1 || plain.Behaviors[0].ID != "entry_1" {
		t.Fatalf("plain patch = %#v", plain)
	}
	fenced, err := ParseBehaviorPatch([]byte("Here is the patch:\n```json\n{\"imports\":[{\"path\":\"./format\"}]}\n```\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(fenced.Imports) != 1 || fenced.Imports[0].Path != "./format" {
		t.Fatalf("fenced patch = %#v", fenced)
	}
}

func TestParseBehaviorPatchRejectsTopLevelNull(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "plain", body: `null`},
		{name: "fenced", body: "```json\nnull\n```"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseBehaviorPatch([]byte(tc.body))
			if err == nil || !strings.Contains(err.Error(), "adapter patch must be a JSON object, got null") {
				t.Fatalf("ParseBehaviorPatch(%q) error = %v, want top-level null rejection", tc.body, err)
			}
		})
	}
}

func TestParseBehaviorPatchRejectsTopLevelNonObjectJSON(t *testing.T) {
	cases := []string{
		`[]`,
		`"translated"`,
		`true`,
		`123`,
	}
	for _, body := range cases {
		t.Run(body, func(t *testing.T) {
			_, err := ParseBehaviorPatch([]byte(body))
			if err == nil || !strings.Contains(err.Error(), "adapter patch must be a JSON object") {
				t.Fatalf("ParseBehaviorPatch(%q) error = %v, want top-level object rejection", body, err)
			}
		})
	}
}

func TestParseBehaviorPatchTracksExplicitProgramImports(t *testing.T) {
	omitted, err := ParseBehaviorPatch([]byte(`{"behaviors":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	if omitted.ImportsSet {
		t.Fatalf("imports set = true for omitted imports: %#v", omitted)
	}
	empty, err := ParseBehaviorPatch([]byte(`{"imports":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	if !empty.ImportsSet || len(empty.Imports) != 0 {
		t.Fatalf("explicit empty imports not tracked: %#v", empty)
	}
}

func TestParseBehaviorPatchAcceptsDartHiddenImportSpecifiers(t *testing.T) {
	patch, err := ParseBehaviorPatch([]byte(`{"imports":[{"path":"helpers.dart","hidden":["debugRecord","privateNormalize"],"language":"dart"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(patch.Imports) != 1 || strings.Join(patch.Imports[0].Hidden, ",") != "debugRecord,privateNormalize" {
		t.Fatalf("hidden imports = %#v", patch.Imports)
	}
}

func TestParseBehaviorPatchAcceptsStaticDeferredAndTypeOnlyImportFlags(t *testing.T) {
	patch, err := ParseBehaviorPatch([]byte(`{"imports":[{"path":"com.example.Format.record","static":true,"language":"java"},{"path":"helpers.dart","alias":"helpers","deferred":true,"language":"dart"},{"path":"./types","type_only":true,"language":"typescript"},{"path":"./metrics","specifiers":[{"name":"DoorMetrics","type_only":true},{"name":"recordMetric"}],"language":"typescript"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(patch.Imports) != 4 || !patch.Imports[0].Static || !patch.Imports[1].Deferred || !patch.Imports[2].TypeOnly || len(patch.Imports[3].Specifiers) != 2 || !patch.Imports[3].Specifiers[0].TypeOnly {
		t.Fatalf("import flags = %#v", patch.Imports)
	}
}

func TestAdapterProtocolForbidsCompilerOwnedResponseFields(t *testing.T) {
	protocol := NewAdapterProtocol(AdapterRequest{TargetLanguage: LanguageTS})
	forbidden := stringSet(protocol.Guidance.ForbiddenFields)
	allowedTopLevel := stringSet(protocol.Response.TopLevelFields)
	for _, field := range jsonFieldNames(reflect.TypeOf(Program{})) {
		if allowedTopLevel[field] {
			continue
		}
		if !forbidden[field] {
			t.Fatalf("program response field %q is neither allowed nor forbidden in adapter guidance: %#v", field, protocol.Guidance.ForbiddenFields)
		}
	}

	allowedGlobals := stringSet(protocol.Response.GlobalFields)
	for _, field := range jsonFieldNames(reflect.TypeOf(CodeBlock{})) {
		if allowedGlobals[field] {
			continue
		}
		forbiddenName := "globals[]." + field
		if !forbidden[forbiddenName] {
			t.Fatalf("global response field %q is neither allowed nor forbidden in adapter guidance: %#v", field, protocol.Guidance.ForbiddenFields)
		}
	}

	allowedBehaviors := stringSet(protocol.Response.BehaviorFields)
	for _, field := range jsonFieldNames(reflect.TypeOf(Behavior{})) {
		if allowedBehaviors[field] {
			continue
		}
		forbiddenName := "behaviors[]." + field
		if !forbidden[forbiddenName] {
			t.Fatalf("behavior response field %q is neither allowed nor forbidden in adapter guidance: %#v", field, protocol.Guidance.ForbiddenFields)
		}
	}
}

func jsonFieldNames(typ reflect.Type) []string {
	fields := make([]string, 0, typ.NumField())
	for index := 0; index < typ.NumField(); index++ {
		field := typ.Field(index)
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			continue
		}
		fields = append(fields, name)
	}
	return fields
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func TestParseBehaviorPatchRejectsUnsupportedTopLevelFields(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "models",
			body: `{"models":[{"id":"model_1","name":"Changed"}]}`,
			want: `unsupported top-level field "models"`,
		},
		{
			name: "fenced models",
			body: "```json\n{\"models\":[{\"id\":\"model_1\",\"name\":\"Changed\"}]}\n```",
			want: `unsupported top-level field "models"`,
		},
		{
			name: "events",
			body: `{"events":[{"id":"event_1","name":"Changed"}]}`,
			want: `unsupported top-level field "events"`,
		},
		{
			name: "attributes",
			body: `{"attributes":[{"id":"attribute_1","name":"changed"}]}`,
			want: `unsupported top-level field "attributes"`,
		},
		{
			name: "operations",
			body: `{"operations":[{"id":"operation_1","name":"changed"}]}`,
			want: `unsupported top-level field "operations"`,
		},
		{
			name: "states",
			body: `{"states":[{"id":"state_1","name":"Changed"}]}`,
			want: `unsupported top-level field "states"`,
		},
		{
			name: "initial",
			body: `{"initial":{"target":"changed"}}`,
			want: `unsupported top-level field "initial"`,
		},
		{
			name: "transitions",
			body: `{"transitions":[{"id":"transition_1","target":"changed"}]}`,
			want: `unsupported top-level field "transitions"`,
		},
		{
			name: "triggers",
			body: `{"triggers":[{"kind":"on","value":"changed"}]}`,
			want: `unsupported top-level field "triggers"`,
		},
		{
			name: "defers",
			body: `{"defers":["changed"]}`,
			want: `unsupported top-level field "defers"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseBehaviorPatch([]byte(tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ParseBehaviorPatch(%q) error = %v, want %q", tc.body, err, tc.want)
			}
		})
	}
}

func TestParseBehaviorPatchAcceptsFlexibleFencedJSON(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			name: "space before tag",
			body: "``` json\n{\"behaviors\":[{\"id\":\"entry_1\",\"body\":\"translated();\"}]}\n```",
		},
		{
			name: "uppercase tag",
			body: "```JSON\n{\"behaviors\":[{\"id\":\"entry_1\",\"body\":\"translated();\"}]}\n```",
		},
		{
			name: "no tag",
			body: "```\n{\"behaviors\":[{\"id\":\"entry_1\",\"body\":\"translated();\"}]}\n```",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			patch, err := ParseBehaviorPatch([]byte(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			if len(patch.Behaviors) != 1 || patch.Behaviors[0].ID != "entry_1" || patch.Behaviors[0].Body != "translated();" {
				t.Fatalf("patch = %#v", patch)
			}
		})
	}
}

func TestParseBehaviorPatchAcceptsFencedJSONWithBackticksInBehaviorBody(t *testing.T) {
	body := "```json\n{\"behaviors\":[{\"id\":\"entry_1\",\"body\":\"instance.log = \\\"```\\\";\"}]}\n```"
	patch, err := ParseBehaviorPatch([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(patch.Behaviors) != 1 || patch.Behaviors[0].Body != "instance.log = \"```\";" {
		t.Fatalf("patch = %#v", patch)
	}
}

func TestParseBehaviorPatchRejectsMultipleFencedJSONBlocks(t *testing.T) {
	body := "```json\n{\"behaviors\":[{\"id\":\"entry_1\",\"body\":\"translated();\"}]}\n```\n```json\n{\"models\":[{\"id\":\"model_1\",\"name\":\"Changed\"}]}\n```"
	_, err := ParseBehaviorPatch([]byte(body))
	if err == nil || !strings.Contains(err.Error(), "adapter returned invalid patch JSON") {
		t.Fatalf("ParseBehaviorPatch accepted multiple fenced blocks: %v", err)
	}
}

func TestParseBehaviorPatchRejectsDuplicateJSONFields(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "top level",
			body: `{"behaviors":[],"behaviors":[{"id":"entry_1","body":"translated();"}]}`,
			want: `adapter patch contains duplicate field "behaviors"`,
		},
		{
			name: "behavior",
			body: `{"behaviors":[{"id":"entry_1","body":"one();","body":"two();"}]}`,
			want: `adapter patch behaviors[0] contains duplicate field "body"`,
		},
		{
			name: "nested import",
			body: `{"globals":[{"id":"global_1","code":"const x = 1;","imports":[{"path":"./one","path":"./two"}]}]}`,
			want: `adapter patch globals[0].imports[0] contains duplicate field "path"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseBehaviorPatch([]byte(tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestParseBehaviorPatchRejectsUnsupportedNestedFields(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "behavior owner",
			body: `{"behaviors":[{"id":"entry_1","owner_id":"state_2","body":"return;"}]}`,
			want: `behaviors[0] contains unsupported field "owner_id"`,
		},
		{
			name: "behavior kind",
			body: `{"behaviors":[{"id":"entry_1","kind":"guard","body":"return true;"}]}`,
			want: `behaviors[0] contains unsupported field "kind"`,
		},
		{
			name: "behavior trigger kind",
			body: `{"behaviors":[{"id":"entry_1","trigger_kind":"when","body":"return true;"}]}`,
			want: `behaviors[0] contains unsupported field "trigger_kind"`,
		},
		{
			name: "behavior source language",
			body: `{"behaviors":[{"id":"entry_1","source_language":"python","body":"return;"}]}`,
			want: `behaviors[0] contains unsupported field "source_language"`,
		},
		{
			name: "behavior target language",
			body: `{"behaviors":[{"id":"entry_1","target_language":"typescript","body":"return;"}]}`,
			want: `behaviors[0] contains unsupported field "target_language"`,
		},
		{
			name: "behavior signature",
			body: `{"behaviors":[{"id":"entry_1","signature":"def changed():","body":"return;"}]}`,
			want: `behaviors[0] contains unsupported field "signature"`,
		},
		{
			name: "behavior abi",
			body: `{"behaviors":[{"id":"entry_1","target_abi":{"signature":"wrong"},"body":"return;"}]}`,
			want: `behaviors[0] contains unsupported field "target_abi"`,
		},
		{
			name: "behavior inline",
			body: `{"behaviors":[{"id":"entry_1","inline":true,"body":"return;"}]}`,
			want: `behaviors[0] contains unsupported field "inline"`,
		},
		{
			name: "behavior range",
			body: `{"behaviors":[{"id":"entry_1","body":"return;","range":{"start_byte":1}}]}`,
			want: `behaviors[0] contains unsupported field "range"`,
		},
		{
			name: "global range",
			body: `{"globals":[{"id":"global_1","code":"const x = 1;","range":{"start_byte":1}}]}`,
			want: `globals[0] contains unsupported field "range"`,
		},
		{
			name: "global language",
			body: `{"globals":[{"id":"global_1","language":"typescript","code":"const x = 1;"}]}`,
			want: `globals[0] contains unsupported field "language"`,
		},
		{
			name: "import local names",
			body: `{"imports":[{"path":"./helper","local_names":["helper"]}]}`,
			want: `imports[0] contains unsupported field "local_names"`,
		},
		{
			name: "global import local names",
			body: `{"globals":[{"id":"global_1","code":"const x = 1;","imports":[{"path":"./helper","local_names":["helper"]}]}]}`,
			want: `globals[0].imports[0] contains unsupported field "local_names"`,
		},
		{
			name: "behavior import local names",
			body: `{"behaviors":[{"id":"entry_1","body":"return;","imports":[{"path":"./helper","local_names":["helper"]}]}]}`,
			want: `behaviors[0].imports[0] contains unsupported field "local_names"`,
		},
		{
			name: "specifier range",
			body: `{"imports":[{"path":"./helper","specifiers":[{"name":"format","range":{"start_byte":1}}]}]}`,
			want: `imports[0].specifiers[0] contains unsupported field "range"`,
		},
		{
			name: "global import specifier range",
			body: `{"globals":[{"id":"global_1","code":"const x = 1;","imports":[{"path":"./helper","specifiers":[{"name":"format","range":{"start_byte":1}}]}]}]}`,
			want: `globals[0].imports[0].specifiers[0] contains unsupported field "range"`,
		},
		{
			name: "behavior import specifier range",
			body: `{"behaviors":[{"id":"entry_1","body":"return;","imports":[{"path":"./helper","specifiers":[{"name":"format","range":{"start_byte":1}}]}]}]}`,
			want: `behaviors[0].imports[0].specifiers[0] contains unsupported field "range"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseBehaviorPatch([]byte(tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestParseBehaviorPatchRejectsInvalidNestedFieldTypes(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "behavior body number",
			body: `{"behaviors":[{"id":"entry_1","body":42}]}`,
			want: `behaviors[0].body must be a string`,
		},
		{
			name: "behavior id null",
			body: `{"behaviors":[{"id":null,"body":"return;"}]}`,
			want: `behaviors[0].id must be a string, got null`,
		},
		{
			name: "behavior body null",
			body: `{"behaviors":[{"id":"entry_1","body":null}]}`,
			want: `behaviors[0].body must be a string, got null`,
		},
		{
			name: "global code object",
			body: `{"globals":[{"id":"global_1","code":{"text":"const x = 1;"}}]}`,
			want: `globals[0].code must be a string`,
		},
		{
			name: "global code null",
			body: `{"globals":[{"id":"global_1","code":null}]}`,
			want: `globals[0].code must be a string, got null`,
		},
		{
			name: "import path number",
			body: `{"imports":[{"path":123}]}`,
			want: `imports[0].path must be a string`,
		},
		{
			name: "import alias null",
			body: `{"imports":[{"path":"./helper","alias":null}]}`,
			want: `imports[0].alias must be a string, got null`,
		},
		{
			name: "specifier alias bool",
			body: `{"imports":[{"path":"./helper","specifiers":[{"name":"format","alias":true}]}]}`,
			want: `imports[0].specifiers[0].alias must be a string`,
		},
		{
			name: "specifier alias null",
			body: `{"imports":[{"path":"./helper","specifiers":[{"name":"format","alias":null}]}]}`,
			want: `imports[0].specifiers[0].alias must be a string, got null`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseBehaviorPatch([]byte(tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestParseBehaviorPatchRequiresGlobalAndBehaviorIDs(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "global missing id",
			body: `{"globals":[{"code":"const helper = () => {};"}]}`,
			want: `globals[0].id is required`,
		},
		{
			name: "global blank id",
			body: `{"globals":[{"id":" \t","code":"const helper = () => {};"}]}`,
			want: `globals[0].id is required`,
		},
		{
			name: "global padded id",
			body: `{"globals":[{"id":" global_1 ","code":"const helper = () => {};"}]}`,
			want: `globals[0].id " global_1 " has leading or trailing whitespace`,
		},
		{
			name: "behavior missing id",
			body: `{"behaviors":[{"body":"translated();"}]}`,
			want: `behaviors[0].id is required`,
		},
		{
			name: "behavior blank id",
			body: `{"behaviors":[{"id":" \t","body":"translated();"}]}`,
			want: `behaviors[0].id is required`,
		},
		{
			name: "behavior padded id",
			body: `{"behaviors":[{"id":" entry_1 ","body":"translated();"}]}`,
			want: `behaviors[0].id " entry_1 " has leading or trailing whitespace`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseBehaviorPatch([]byte(tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestParseBehaviorPatchRequiresImportPaths(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "program import missing path",
			body: `{"imports":[{"language":"typescript"}]}`,
			want: `imports[0].path is required`,
		},
		{
			name: "program import blank path",
			body: `{"imports":[{"path":" \t"}]}`,
			want: `imports[0].path is required`,
		},
		{
			name: "program import padded path",
			body: `{"imports":[{"path":" ./helpers "}]}`,
			want: `imports[0].path " ./helpers " has leading or trailing whitespace`,
		},
		{
			name: "global import missing path",
			body: `{"globals":[{"id":"global_1","code":"const x = 1;","imports":[{"language":"typescript"}]}]}`,
			want: `globals[0].imports[0].path is required`,
		},
		{
			name: "behavior import missing path",
			body: `{"behaviors":[{"id":"entry_1","body":"translated();","imports":[{"language":"typescript"}]}]}`,
			want: `behaviors[0].imports[0].path is required`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseBehaviorPatch([]byte(tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestParseBehaviorPatchRejectsAmbiguousOptionalImportFields(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "blank import alias",
			body: `{"imports":[{"path":"./helpers","alias":" \t"}]}`,
			want: `imports[0].alias is empty`,
		},
		{
			name: "padded import alias",
			body: `{"imports":[{"path":"./helpers","alias":" helpers "}]}`,
			want: `imports[0].alias " helpers " has leading or trailing whitespace`,
		},
		{
			name: "blank import default",
			body: `{"imports":[{"path":"./helpers","default":" \t"}]}`,
			want: `imports[0].default is empty`,
		},
		{
			name: "padded import default",
			body: `{"imports":[{"path":"./helpers","default":" helpers "}]}`,
			want: `imports[0].default " helpers " has leading or trailing whitespace`,
		},
		{
			name: "blank import language",
			body: `{"imports":[{"path":"./helpers","language":" \t"}]}`,
			want: `imports[0].language is empty`,
		},
		{
			name: "padded import language",
			body: `{"imports":[{"path":"./helpers","language":" typescript "}]}`,
			want: `imports[0].language " typescript " has leading or trailing whitespace`,
		},
		{
			name: "blank specifier alias",
			body: `{"imports":[{"path":"./helpers","specifiers":[{"name":"format","alias":" \t"}]}]}`,
			want: `imports[0].specifiers[0].alias is empty`,
		},
		{
			name: "padded specifier alias",
			body: `{"imports":[{"path":"./helpers","specifiers":[{"name":"format","alias":" fmt "}]}]}`,
			want: `imports[0].specifiers[0].alias " fmt " has leading or trailing whitespace`,
		},
		{
			name: "null hidden specifiers",
			body: `{"imports":[{"path":"./helpers","hidden":null}]}`,
			want: `adapter patch field "imports[0].hidden" must be an array, got null`,
		},
		{
			name: "blank hidden specifier",
			body: `{"imports":[{"path":"./helpers","hidden":[" \t"]}]}`,
			want: `imports[0].hidden[0] is empty`,
		},
		{
			name: "padded hidden specifier",
			body: `{"imports":[{"path":"./helpers","hidden":[" helper "]}]}`,
			want: `imports[0].hidden[0] " helper " has leading or trailing whitespace`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseBehaviorPatch([]byte(tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestParseBehaviorPatchRequiresImportSpecifierNames(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "program import specifier missing name",
			body: `{"imports":[{"path":"./helpers","specifiers":[{"alias":"helper"}]}]}`,
			want: `imports[0].specifiers[0].name is required`,
		},
		{
			name: "program import specifier blank name",
			body: `{"imports":[{"path":"./helpers","specifiers":[{"name":" \t"}]}]}`,
			want: `imports[0].specifiers[0].name is required`,
		},
		{
			name: "program import specifier padded name",
			body: `{"imports":[{"path":"./helpers","specifiers":[{"name":" format "}]}]}`,
			want: `imports[0].specifiers[0].name " format " has leading or trailing whitespace`,
		},
		{
			name: "global import specifier missing name",
			body: `{"globals":[{"id":"global_1","code":"const x = 1;","imports":[{"path":"./helpers","specifiers":[{"alias":"helper"}]}]}]}`,
			want: `globals[0].imports[0].specifiers[0].name is required`,
		},
		{
			name: "behavior import specifier missing name",
			body: `{"behaviors":[{"id":"entry_1","body":"translated();","imports":[{"path":"./helpers","specifiers":[{"alias":"helper"}]}]}]}`,
			want: `behaviors[0].imports[0].specifiers[0].name is required`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseBehaviorPatch([]byte(tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestParseBehaviorPatchRejectsBehaviorWithBodyAndCode(t *testing.T) {
	_, err := ParseBehaviorPatch([]byte(`{"behaviors":[{"id":"entry_1","body":"translated();","code":"function entry() {}"}]}`))
	if err == nil || !strings.Contains(err.Error(), `behaviors[0] must set either body or code, not both`) {
		t.Fatalf("error = %v, want body/code exclusivity rejection", err)
	}
}

func TestParseBehaviorPatchRejectsWhitespaceOnlyCodeFields(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "global code",
			body: `{"globals":[{"id":"global_1","code":" \t\n"}]}`,
			want: `globals[0].code contains whitespace-only code`,
		},
		{
			name: "behavior body",
			body: `{"behaviors":[{"id":"entry_1","body":" \t\n"}]}`,
			want: `behaviors[0].body contains whitespace-only code`,
		},
		{
			name: "behavior code",
			body: `{"behaviors":[{"id":"entry_1","code":" \t\n"}]}`,
			want: `behaviors[0].code contains whitespace-only code`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseBehaviorPatch([]byte(tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestParseBehaviorPatchRejectsNullArrayFields(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "top-level imports",
			body: `{"imports":null}`,
			want: `field "imports" must be an array, got null`,
		},
		{
			name: "top-level globals",
			body: `{"globals":null}`,
			want: `field "globals" must be an array, got null`,
		},
		{
			name: "top-level behaviors",
			body: `{"behaviors":null}`,
			want: `field "behaviors" must be an array, got null`,
		},
		{
			name: "global imports",
			body: `{"globals":[{"id":"global_1","code":"const x = 1;","imports":null}]}`,
			want: `field "globals[0].imports" must be an array, got null`,
		},
		{
			name: "behavior imports",
			body: `{"behaviors":[{"id":"entry_1","body":"return;","imports":null}]}`,
			want: `field "behaviors[0].imports" must be an array, got null`,
		},
		{
			name: "import specifiers",
			body: `{"imports":[{"path":"./helper","specifiers":null}]}`,
			want: `field "imports[0].specifiers" must be an array, got null`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseBehaviorPatch([]byte(tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestParseBehaviorPatchWithContextIncludesInvalidOutputSnippet(t *testing.T) {
	_, err := ParseBehaviorPatchWithContext([]byte("not json\nand not fenced"))
	if err == nil || !strings.Contains(err.Error(), `output "not json and not fenced"`) {
		t.Fatalf("error = %v, want invalid output snippet", err)
	}
}

func TestCodexAdapterSendsProtocolEnvelopeAndReadsPatch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test is unix-specific")
	}
	dir := t.TempDir()
	capture := filepath.Join(dir, "request.json")
	script := filepath.Join(dir, "adapter.sh")
	body := "#!/bin/sh\ncat > " + shellQuote(capture) + "\nprintf '%s\\n' '{\"behaviors\":[{\"id\":\"entry_1\",\"body\":\"translated();\"}]}'\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	adapter := CodexAdapter{Command: script}
	patch, err := adapter.Translate(context.Background(), AdapterRequest{
		SourceLanguage: LanguageGo,
		TargetLanguage: LanguageTS,
		Behaviors:      []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(patch.Behaviors) != 1 || patch.Behaviors[0].Body != "translated();" {
		t.Fatalf("patch = %#v", patch)
	}
	data, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	var protocol AdapterProtocol
	if err := json.Unmarshal(data, &protocol); err != nil {
		t.Fatal(err)
	}
	if protocol.Version != "hsmc.adapter.v1" || len(protocol.AllowedEdits.BehaviorIDs) != 1 {
		t.Fatalf("captured protocol = %#v", protocol)
	}
}

func TestCodexAdapterRunsExecWithConfiguredArgsAndStdinPrompt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test is unix-specific")
	}
	dir := t.TempDir()
	captureArgs := filepath.Join(dir, "args.txt")
	script := filepath.Join(dir, "adapter.sh")
	body := "#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' \"$@\" > " + shellQuote(captureArgs) + "\nprintf '%s\\n' '{\"behaviors\":[]}'\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	adapter := CodexAdapter{Command: " " + script + " ", Args: []string{"--model", "gpt-5", "--ask-for-approval=never"}}
	patch, err := adapter.Translate(context.Background(), AdapterRequest{TargetLanguage: LanguageTS})
	if err != nil {
		t.Fatal(err)
	}
	if patch == nil || len(patch.Behaviors) != 0 {
		t.Fatalf("patch = %#v", patch)
	}
	data, err := os.ReadFile(captureArgs)
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(args) != 7 || args[0] != "exec" || args[1] != "--model" || args[2] != "gpt-5" || args[3] != "--ask-for-approval=never" || args[4] != "--output-last-message" || args[5] == "" || args[6] != "-" {
		t.Fatalf("args = %#v", args)
	}
}

func TestCodexAdapterPrefersOutputLastMessageFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test is unix-specific")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "adapter.sh")
	body := `#!/bin/sh
output=
while [ "$#" -gt 0 ]; do
	if [ "$1" = "--output-last-message" ]; then
		shift
		output=$1
	fi
	shift
done
cat >/dev/null
printf '%s\n' '{"behaviors":[{"id":"entry_1","body":"from_file();"}]}' > "$output"
printf '%s\n' 'stdout is not patch json'
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	adapter := CodexAdapter{Command: script}
	patch, err := adapter.Translate(context.Background(), AdapterRequest{TargetLanguage: LanguageTS})
	if err != nil {
		t.Fatal(err)
	}
	if len(patch.Behaviors) != 1 || patch.Behaviors[0].Body != "from_file();" {
		t.Fatalf("patch = %#v", patch)
	}
}

func TestCodexAdapterReadsFlexibleFencedOutputLastMessagePatch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test is unix-specific")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "adapter.sh")
	body := "#!/bin/sh\n" +
		"output=\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"\tif [ \"$1\" = \"--output-last-message\" ]; then\n" +
		"\t\tshift\n" +
		"\t\toutput=$1\n" +
		"\tfi\n" +
		"\tshift\n" +
		"done\n" +
		"cat >/dev/null\n" +
		"{\n" +
		"\tprintf '%s\\n' '``` JSON'\n" +
		"\tprintf '%s\\n' '{\"behaviors\":[{\"id\":\"entry_1\",\"body\":\"from_fenced_file();\"}]}'\n" +
		"\tprintf '%s\\n' '```'\n" +
		"} > \"$output\"\n" +
		"printf '%s\\n' 'stdout is not patch json'\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	adapter := CodexAdapter{Command: script}
	patch, err := adapter.Translate(context.Background(), AdapterRequest{TargetLanguage: LanguageTS})
	if err != nil {
		t.Fatal(err)
	}
	if len(patch.Behaviors) != 1 || patch.Behaviors[0].Body != "from_fenced_file();" {
		t.Fatalf("patch = %#v", patch)
	}
}

func TestCodexAdapterFallsBackToStdoutWhenOutputLastMessageFileIsEmpty(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test is unix-specific")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "adapter.sh")
	body := `#!/bin/sh
while [ "$#" -gt 0 ]; do
	if [ "$1" = "--output-last-message" ]; then
		shift
		: > "$1"
	fi
	shift
done
cat >/dev/null
printf '%s\n' '{"behaviors":[{"id":"entry_1","body":"from_stdout();"}]}'
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	adapter := CodexAdapter{Command: script}
	patch, err := adapter.Translate(context.Background(), AdapterRequest{TargetLanguage: LanguageTS})
	if err != nil {
		t.Fatal(err)
	}
	if len(patch.Behaviors) != 1 || patch.Behaviors[0].Body != "from_stdout();" {
		t.Fatalf("patch = %#v", patch)
	}
}

func TestCodexAdapterRejectsStructuralPatchFromOutputLastMessageFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test is unix-specific")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "adapter.sh")
	body := `#!/bin/sh
output=
while [ "$#" -gt 0 ]; do
	if [ "$1" = "--output-last-message" ]; then
		shift
		output=$1
	fi
	shift
done
cat >/dev/null
printf '%s\n' '{"models":[{"id":"model_1","name":"Changed"}]}' > "$output"
printf '%s\n' '{"behaviors":[{"id":"entry_1","body":"stdout_should_not_win();"}]}'
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	adapter := CodexAdapter{Command: script}
	_, err := adapter.Translate(context.Background(), AdapterRequest{TargetLanguage: LanguageTS})
	if err == nil || !strings.Contains(err.Error(), `unsupported top-level field "models"`) {
		t.Fatalf("error = %v, want structural patch rejection from output-last-message file", err)
	}
}

func TestCodexAdapterRejectsNullPatchFromOutputLastMessageFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test is unix-specific")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "adapter.sh")
	body := `#!/bin/sh
output=
while [ "$#" -gt 0 ]; do
	if [ "$1" = "--output-last-message" ]; then
		shift
		output=$1
	fi
	shift
done
cat >/dev/null
printf '%s\n' 'null' > "$output"
printf '%s\n' '{"behaviors":[{"id":"entry_1","body":"stdout_should_not_win();"}]}'
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	adapter := CodexAdapter{Command: script}
	_, err := adapter.Translate(context.Background(), AdapterRequest{TargetLanguage: LanguageTS})
	if err == nil || !strings.Contains(err.Error(), `adapter patch must be a JSON object, got null`) {
		t.Fatalf("error = %v, want null patch rejection from output-last-message file", err)
	}
}

func TestCodexAdapterRejectsAmbiguousBehaviorPatchFromOutputLastMessageFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test is unix-specific")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "adapter.sh")
	body := `#!/bin/sh
output=
while [ "$#" -gt 0 ]; do
	if [ "$1" = "--output-last-message" ]; then
		shift
		output=$1
	fi
	shift
done
cat >/dev/null
printf '%s\n' '{"behaviors":[{"id":"entry_1","body":"translated();","code":"function entry_1() {}"}]}' > "$output"
printf '%s\n' '{"behaviors":[{"id":"entry_1","body":"stdout_should_not_win();"}]}'
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	adapter := CodexAdapter{Command: script}
	_, err := adapter.Translate(context.Background(), AdapterRequest{TargetLanguage: LanguageTS})
	if err == nil || !strings.Contains(err.Error(), `behaviors[0] must set either body or code, not both`) {
		t.Fatalf("error = %v, want behavior body/code rejection from output-last-message file", err)
	}
}

func TestCommandAdapterReportsFailureOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test is unix-specific")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "adapter.sh")
	body := "#!/bin/sh\nprintf '%s\\n' 'partial stdout'\nprintf '%s\\n' 'adapter stderr' >&2\nexit 7\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	adapter := CommandAdapter{NameValue: "custom", Command: script}
	_, err := adapter.Translate(context.Background(), AdapterRequest{TargetLanguage: LanguageTS})
	if err == nil {
		t.Fatal("expected adapter failure")
	}
	for _, want := range []string{"custom adapter failed", "adapter stderr", "partial stdout"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want %q", err.Error(), want)
		}
	}
}

func TestCommandAdapterReportsMalformedPatchOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test is unix-specific")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "adapter.sh")
	body := "#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' 'adapter explanation without json'\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	adapter := CommandAdapter{NameValue: "custom", Command: script}
	_, err := adapter.Translate(context.Background(), AdapterRequest{TargetLanguage: LanguageTS})
	if err == nil || !strings.Contains(err.Error(), `output "adapter explanation without json"`) {
		t.Fatalf("error = %v, want malformed output context", err)
	}
}

func TestCommandAdapterRequiresCommand(t *testing.T) {
	adapter := CommandAdapter{}
	_, err := adapter.Translate(context.Background(), AdapterRequest{TargetLanguage: LanguageTS})
	if err == nil || !strings.Contains(err.Error(), "command adapter command is required") {
		t.Fatalf("Translate() error = %v, want command required", err)
	}
}

func TestCommandAdapterUsesConfiguredNameAndCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test is unix-specific")
	}
	dir := t.TempDir()
	capture := filepath.Join(dir, "request.json")
	script := filepath.Join(dir, "adapter.sh")
	body := "#!/bin/sh\ncat > " + shellQuote(capture) + "\nprintf '%s\\n' '{\"behaviors\":[{\"id\":\"entry_1\",\"body\":\"translatedByCommand();\"}]}'\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	adapter := CommandAdapter{NameValue: "custom", Command: script}
	if adapter.Name() != "custom" {
		t.Fatalf("Name() = %q, want custom", adapter.Name())
	}
	patch, err := adapter.Translate(context.Background(), AdapterRequest{
		SourceLanguage: LanguageGo,
		TargetLanguage: LanguageTS,
		Behaviors:      []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(patch.Behaviors) != 1 || patch.Behaviors[0].Body != "translatedByCommand();" {
		t.Fatalf("patch = %#v", patch)
	}
	data, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	var protocol AdapterProtocol
	if err := json.Unmarshal(data, &protocol); err != nil {
		t.Fatal(err)
	}
	if protocol.Version != "hsmc.adapter.v1" || protocol.Request.TargetLanguage != LanguageTS {
		t.Fatalf("captured protocol = %#v", protocol)
	}
}

func TestApplyBehaviorPatchDefaultsTargetLanguageAndPreservesIdentity(t *testing.T) {
	program := &Program{
		SourceLanguage: LanguageGo,
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			Initializer: &Initial{
				OwnerID: "model_1",
				ID:      "initial_1",
				Target:  "closed",
			},
			States: []State{{ID: "state_1", Name: "closed", Kind: StateKindState}},
		}},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			TriggerKind:    TriggerAt,
			SourceLanguage: LanguageGo,
			Signature:      "func(ctx context.Context, sm *Door, event hsm.Event)",
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Behaviors: program.Behaviors}
	err := applyBehaviorPatch(program, request, &BehaviorPatch{
		Imports: []Import{{Path: "./format"}},
		Behaviors: []Behavior{{
			ID:   "entry_1",
			Body: "translated();",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if program.Imports[0].Language != LanguageTS {
		t.Fatalf("import language = %q, want target", program.Imports[0].Language)
	}
	behavior := program.Behaviors[0]
	if behavior.OwnerID != "state_1" || behavior.Kind != BehaviorEntry || behavior.TriggerKind != TriggerAt || behavior.SourceLanguage != LanguageGo {
		t.Fatalf("identity not preserved: %#v", behavior)
	}
	if behavior.Signature != "func(ctx context.Context, sm *Door, event hsm.Event)" {
		t.Fatalf("signature not preserved: %#v", behavior)
	}
	if behavior.TargetLanguage != LanguageTS {
		t.Fatalf("target language = %q", behavior.TargetLanguage)
	}
}

func TestApplyBehaviorPatchRejectsImmutableBehaviorFields(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Behaviors: program.Behaviors}
	cases := []struct {
		name     string
		behavior Behavior
		want     string
	}{
		{
			name:     "owner",
			behavior: Behavior{ID: "entry_1", OwnerID: "wrong"},
			want:     "owner_id",
		},
		{
			name:     "kind",
			behavior: Behavior{ID: "entry_1", Kind: BehaviorGuard},
			want:     "kind",
		},
		{
			name:     "trigger kind",
			behavior: Behavior{ID: "entry_1", TriggerKind: TriggerAfter},
			want:     "trigger_kind",
		},
		{
			name:     "source language",
			behavior: Behavior{ID: "entry_1", SourceLanguage: LanguageTS},
			want:     "source_language",
		},
		{
			name:     "target language",
			behavior: Behavior{ID: "entry_1", TargetLanguage: LanguageTS},
			want:     "target_language",
		},
		{
			name:     "signature",
			behavior: Behavior{ID: "entry_1", Signature: "function wrong(): void"},
			want:     "signature",
		},
		{
			name:     "target abi",
			behavior: Behavior{ID: "entry_1", TargetABI: &BehaviorABI{Language: LanguageTS}},
			want:     "target_abi",
		},
		{
			name:     "range",
			behavior: Behavior{ID: "entry_1", Range: SourceRange{StartByte: 1}},
			want:     "range",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{tc.behavior}})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want immutable field %q rejection", err, tc.want)
			}
		})
	}
}

func TestApplyBehaviorPatchIsAtomicOnRejectedPatch(t *testing.T) {
	program := &Program{
		Imports: []Import{{Path: "fmt", Language: LanguageGo}},
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageGo,
			Code:     "func helper() {}",
			Imports:  []Import{{Path: "strings", Language: LanguageGo}},
		}},
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			States: []State{{
				ID:        "state_1",
				Name:      "closed",
				Kind:      StateKindState,
				Behaviors: []BehaviorRef{{ID: "entry_1", Kind: BehaviorEntry}},
			}},
		}},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			Imports:        []Import{{Path: "context", Language: LanguageGo}},
		}},
	}
	before := cloneProgram(program)
	request := AdapterRequest{TargetLanguage: LanguageTS, Imports: program.Imports, Globals: program.Globals, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{
		Imports: []Import{{Path: "./program"}},
		Globals: []CodeBlock{{
			ID:      "global_1",
			Code:    "const helper = () => true;",
			Imports: []Import{{Path: "./global"}},
		}},
		Behaviors: []Behavior{{
			ID:      "entry_1",
			OwnerID: "state_2",
			Body:    "translated();",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "owner_id") {
		t.Fatalf("error = %v, want immutable owner_id rejection", err)
	}
	if !reflect.DeepEqual(program, before) {
		t.Fatalf("program mutated after rejected patch:\n got %#v\nwant %#v", program, before)
	}
}

func TestApplyBehaviorPatchIsAtomicOnRejectedFinalImportValidation(t *testing.T) {
	program := &Program{
		Imports: []Import{{Path: "fmt", Language: LanguageGo}},
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageGo,
			Code:     "func helper() {}",
			Imports:  []Import{{Path: "strings", Language: LanguageGo}},
		}},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			Imports:        []Import{{Path: "context", Language: LanguageGo}},
		}},
	}
	before := cloneProgram(program)
	request := AdapterRequest{TargetLanguage: LanguageTS, Imports: program.Imports, Globals: program.Globals, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{
		Imports: []Import{{Path: "./program-helper", Default: "shared"}},
		Globals: []CodeBlock{{
			ID:      "global_1",
			Code:    "const helper = () => true;",
			Imports: []Import{{Path: "./global-helper", Default: "globalHelper"}},
		}},
		Behaviors: []Behavior{{
			ID:      "entry_1",
			Body:    "translated();",
			Imports: []Import{{Path: "./behavior-helper", Default: "shared"}},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), `both bind local name "shared"`) {
		t.Fatalf("error = %v, want final import binding rejection", err)
	}
	if !reflect.DeepEqual(program, before) {
		t.Fatalf("program mutated after rejected final import validation:\n got %#v\nwant %#v", program, before)
	}
}

func TestCloneProgramDeepCopiesBehaviorABI(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			TargetABI: &BehaviorABI{
				Language:   LanguageTS,
				Signature:  "entry(ctx: hsm.Context): void",
				ReturnType: "void",
				Parameters: []ABIParameter{{Name: "ctx", Type: "hsm.Context"}},
			},
		}},
	}

	cloned := cloneProgram(program)
	cloned.Behaviors[0].TargetABI.Signature = "mutated"
	cloned.Behaviors[0].TargetABI.Parameters[0].Type = "mutated.Type"

	if program.Behaviors[0].TargetABI.Signature != "entry(ctx: hsm.Context): void" {
		t.Fatalf("cloneProgram shared behavior ABI signature: %#v", program.Behaviors[0].TargetABI)
	}
	if program.Behaviors[0].TargetABI.Parameters[0].Type != "hsm.Context" {
		t.Fatalf("cloneProgram shared behavior ABI parameters: %#v", program.Behaviors[0].TargetABI.Parameters)
	}
}

func TestApplyBehaviorPatchDeepCopiesAcceptedPatchImports(t *testing.T) {
	program := &Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageGo,
			Code:     "func helper() {}",
		}},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Globals: program.Globals, Behaviors: program.Behaviors}
	patch := &BehaviorPatch{
		Imports: []Import{{
			Path:       "./program-helper",
			Specifiers: []ImportSpecifier{{Name: "programFormat"}},
		}},
		Globals: []CodeBlock{
			{
				ID:   "global_1",
				Code: "const format = globalFormat;",
				Imports: []Import{{
					Path:       "./global-helper",
					Specifiers: []ImportSpecifier{{Name: "globalFormat"}},
				}},
			},
			{
				ID:   "adapter_global_helper",
				Code: "const helper = adapterHelper;",
				Imports: []Import{{
					Path:       "./adapter-helper",
					Specifiers: []ImportSpecifier{{Name: "adapterHelper"}},
				}},
			},
		},
		Behaviors: []Behavior{{
			ID:   "entry_1",
			Body: "behaviorFormat();",
			Imports: []Import{{
				Path:       "./behavior-helper",
				Specifiers: []ImportSpecifier{{Name: "behaviorFormat"}},
			}},
		}},
	}

	if err := applyBehaviorPatch(program, request, patch); err != nil {
		t.Fatal(err)
	}

	patch.Imports[0].Specifiers[0].Name = "mutatedProgramFormat"
	patch.Globals[0].Imports[0].Specifiers[0].Name = "mutatedGlobalFormat"
	patch.Globals[1].Imports[0].Specifiers[0].Name = "mutatedAdapterHelper"
	patch.Behaviors[0].Imports[0].Specifiers[0].Name = "mutatedBehaviorFormat"

	if program.Imports[0].Specifiers[0].Name != "programFormat" {
		t.Fatalf("program import shares patch storage: %#v", program.Imports)
	}
	if program.Globals[0].Imports[0].Specifiers[0].Name != "globalFormat" {
		t.Fatalf("global import shares patch storage: %#v", program.Globals[0].Imports)
	}
	if program.Globals[1].Imports[0].Specifiers[0].Name != "adapterHelper" {
		t.Fatalf("adapter helper global import shares patch storage: %#v", program.Globals[1].Imports)
	}
	if program.Behaviors[0].Imports[0].Specifiers[0].Name != "behaviorFormat" {
		t.Fatalf("behavior import shares patch storage: %#v", program.Behaviors[0].Imports)
	}
}

func TestApplyBehaviorPatchPreservesNonTargetProgramImports(t *testing.T) {
	program := &Program{
		Imports: []Import{
			{Path: "fmt", Language: LanguageGo},
			{Path: "./old-target", Language: LanguageTS},
		},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			Body:           `fmt.Println("closed")`,
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Imports: program.Imports, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{
		Imports:   []Import{{Path: "./new-target"}},
		Behaviors: []Behavior{{ID: "entry_1", Body: `record("closed");`}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 2 {
		t.Fatalf("program imports = %#v, want source structural plus patched target import", program.Imports)
	}
	if program.Imports[0].Path != "fmt" || program.Imports[0].Language != LanguageGo {
		t.Fatalf("source structural import not preserved: %#v", program.Imports)
	}
	if program.Imports[1].Path != "./new-target" || program.Imports[1].Language != LanguageTS {
		t.Fatalf("target import not patched/defaulted: %#v", program.Imports)
	}
}

func TestApplyBehaviorPatchRejectsProgramImportOnlyPatch(t *testing.T) {
	program := &Program{
		Imports: []Import{{Path: "fmt", Language: LanguageGo}},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			Body:           `fmt.Println("closed")`,
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Imports: program.Imports, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{
		Imports: []Import{{Path: "./format"}},
	})
	if err == nil || !strings.Contains(err.Error(), "program imports without translated target global or behavior code") {
		t.Fatalf("error = %v, want program import-only rejection", err)
	}
	if len(program.Imports) != 1 || program.Imports[0].Path != "fmt" || program.Imports[0].Language != LanguageGo {
		t.Fatalf("program imports mutated after rejected patch: %#v", program.Imports)
	}
}

func TestApplyBehaviorPatchCanClearTargetProgramImports(t *testing.T) {
	program := &Program{
		Imports: []Import{
			{Path: "fmt", Language: LanguageGo},
			{Path: "./target-helper", Language: LanguageTS},
		},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Imports: program.Imports}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{ImportsSet: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 1 || program.Imports[0].Path != "fmt" || program.Imports[0].Language != LanguageGo {
		t.Fatalf("program imports = %#v, want only non-target source import", program.Imports)
	}
}

func TestApplyBehaviorPatchCanClearTargetProgramImportsWithNonNilEmptySlice(t *testing.T) {
	program := &Program{
		Imports: []Import{
			{Path: "fmt", Language: LanguageGo},
			{Path: "./target-helper", Language: LanguageTS},
		},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Imports: program.Imports}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{Imports: []Import{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 1 || program.Imports[0].Path != "fmt" || program.Imports[0].Language != LanguageGo {
		t.Fatalf("program imports = %#v, want only non-target source import", program.Imports)
	}
}

func TestApplyBehaviorPatchRejectsUnknownGlobalAndBehaviorIDs(t *testing.T) {
	program := &Program{
		Globals:   []CodeBlock{{ID: "global_1", Language: LanguageGo, Code: "func helper() {}"}},
		Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Globals: program.Globals, Behaviors: program.Behaviors}

	if err := applyBehaviorPatch(program, request, &BehaviorPatch{Globals: []CodeBlock{{ID: "missing", Code: "const helper = () => {};"}}}); err == nil {
		t.Fatal("expected unknown global error")
	}
	if err := applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{{ID: "missing", Body: "translated();"}}}); err == nil {
		t.Fatal("expected unknown behavior error")
	}
}

func TestApplyBehaviorPatchRejectsIDsNotExposedInAdapterRequest(t *testing.T) {
	program := &Program{
		Globals: []CodeBlock{
			{ID: "global_1", Language: LanguageGo, Code: "func helperOne() {}"},
			{ID: "global_2", Language: LanguageGo, Code: "func helperTwo() {}"},
		},
		Behaviors: []Behavior{
			{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo},
			{ID: "entry_2", OwnerID: "state_2", Kind: BehaviorEntry, SourceLanguage: LanguageGo},
		},
	}
	request := AdapterRequest{
		TargetLanguage: LanguageTS,
		Globals:        []CodeBlock{program.Globals[0]},
		Behaviors:      []Behavior{program.Behaviors[0]},
	}

	if err := applyBehaviorPatch(program, request, &BehaviorPatch{Globals: []CodeBlock{{ID: "global_2", Code: "const helperTwo = () => {};"}}}); err == nil || !strings.Contains(err.Error(), `uneditable global "global_2"`) {
		t.Fatalf("global patch error = %v, want uneditable global", err)
	}
	if err := applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{{ID: "entry_2", Body: "translated();"}}}); err == nil || !strings.Contains(err.Error(), `uneditable behavior "entry_2"`) {
		t.Fatalf("behavior patch error = %v, want uneditable behavior", err)
	}
	if err := applyBehaviorPatch(program, request, &BehaviorPatch{Globals: []CodeBlock{{ID: "adapter_global_helper", Code: "const helper = () => {};"}}}); err != nil {
		t.Fatalf("adapter helper global should not require existing editable ID: %v", err)
	}
}

func TestValidateRejectsEmptyAndDuplicateGlobalIDs(t *testing.T) {
	cases := []struct {
		name    string
		globals []CodeBlock
		want    string
	}{
		{
			name:    "empty",
			globals: []CodeBlock{{ID: ""}},
			want:    "global id is empty",
		},
		{
			name:    "duplicate",
			globals: []CodeBlock{{ID: "global_1"}, {ID: "global_1"}},
			want:    `duplicate global id "global_1"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(&Program{Globals: tc.globals})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestValidateRejectsEmptyAndDuplicateBehaviorIDs(t *testing.T) {
	cases := []struct {
		name      string
		behaviors []Behavior
		want      string
	}{
		{
			name:      "empty",
			behaviors: []Behavior{{ID: ""}},
			want:      "behavior id is empty",
		},
		{
			name: "duplicate",
			behaviors: []Behavior{
				{ID: "entry_1", Kind: BehaviorEntry, Range: SourceRange{StartLine: 10}},
				{ID: "entry_1", Kind: BehaviorEntry, Range: SourceRange{StartLine: 20}},
			},
			want: `duplicate behavior id "entry_1"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(&Program{Behaviors: tc.behaviors})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() error = %v, want %q", err, tc.want)
			}
			if tc.name == "duplicate" {
				diagnostics, ok := err.(Diagnostics)
				if !ok {
					t.Fatalf("diagnostics = %#v, want both duplicate behavior ranges", err)
				}
				var duplicateRanges []SourceRange
				for _, diagnostic := range diagnostics {
					if diagnostic.Message == `duplicate behavior id "entry_1"` {
						duplicateRanges = append(duplicateRanges, diagnostic.Range)
					}
				}
				if len(duplicateRanges) != 2 || duplicateRanges[0].StartLine != 10 || duplicateRanges[1].StartLine != 20 {
					t.Fatalf("diagnostic ranges = %#v, want first and duplicate behavior ranges", diagnostics)
				}
			}
		})
	}
}

func TestApplyBehaviorPatchRejectsEmptyAndDuplicatePatchIDs(t *testing.T) {
	program := &Program{
		Globals:   []CodeBlock{{ID: "global_1", Language: LanguageTS, Code: "const helper = () => true;"}},
		Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageTS}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Globals: program.Globals, Behaviors: program.Behaviors}

	cases := []BehaviorPatch{
		{Globals: []CodeBlock{{ID: "", Code: "const helper = () => true;"}}},
		{Globals: []CodeBlock{{ID: "global_1", Code: "const one = true;"}, {ID: "global_1", Code: "const two = true;"}}},
		{Globals: []CodeBlock{{ID: "adapter_global_one", Code: "const one = true;"}, {ID: "adapter_global_one", Code: "const two = true;"}}},
		{Behaviors: []Behavior{{ID: "", Body: "translated();"}}},
		{Behaviors: []Behavior{{ID: "entry_1", Body: "one();"}, {ID: "entry_1", Body: "two();"}}},
	}
	for _, patch := range cases {
		if err := applyBehaviorPatch(program, request, &patch); err == nil {
			t.Fatalf("expected duplicate/empty id error for %#v", patch)
		}
	}
}

func TestApplyBehaviorPatchRejectsPaddedPatchIDs(t *testing.T) {
	program := &Program{
		Globals:   []CodeBlock{{ID: "global_1", Language: LanguageTS, Code: "const helper = () => true;"}},
		Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageTS}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Globals: program.Globals, Behaviors: program.Behaviors}

	cases := []struct {
		name  string
		patch BehaviorPatch
		want  string
	}{
		{
			name:  "global id",
			patch: BehaviorPatch{Globals: []CodeBlock{{ID: " global_1 ", Code: "const helper = () => false;"}}},
			want:  `adapter patch global id " global_1 " has leading or trailing whitespace`,
		},
		{
			name:  "behavior id",
			patch: BehaviorPatch{Behaviors: []Behavior{{ID: " entry_1 ", Body: "translated();"}}},
			want:  `adapter patch behavior id " entry_1 " has leading or trailing whitespace`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := applyBehaviorPatch(program, request, &tc.patch)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestApplyBehaviorPatchAllowsAdapterHelperGlobals(t *testing.T) {
	program := &Program{
		Globals: []CodeBlock{{ID: "global_1", Language: LanguageGo, Code: "func helper() {}"}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Globals: program.Globals}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{Globals: []CodeBlock{{
		ID:      "adapter_global_format",
		Code:    "const format = (value: string) => value;",
		Imports: []Import{{Path: "./format"}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals) != 2 {
		t.Fatalf("globals = %#v", program.Globals)
	}
	helper := program.Globals[1]
	if helper.ID != "adapter_global_format" || helper.Language != LanguageTS || helper.Imports[0].Language != LanguageTS {
		t.Fatalf("helper global = %#v", helper)
	}
}

func TestApplyBehaviorPatchRejectsDuplicateTargetGlobalDeclarations(t *testing.T) {
	program := &Program{
		Globals: []CodeBlock{{ID: "global_1", Language: LanguageGo, Code: "func sourceHelper() {}"}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Globals: program.Globals}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{Globals: []CodeBlock{
		{ID: "adapter_global_format_one", Code: "const format = (value: string) => value;"},
		{ID: "adapter_global_format_two", Code: "function format(value: string): string { return value; }"},
	}})
	if err == nil || !strings.Contains(err.Error(), `target-language global "adapter_global_format_two" declares "format", which collides with target-language global "adapter_global_format_one"`) {
		t.Fatalf("error = %v, want duplicate target global declaration rejection", err)
	}
	if len(program.Globals) != 1 {
		t.Fatalf("failed patch mutated globals: %#v", program.Globals)
	}
}

func TestApplyBehaviorPatchRejectsAdapterHelperGlobalRuntimeBindingCollisions(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{Globals: []CodeBlock{{
		ID:   "adapter_global_collision",
		Code: `const hsm = createHelperRuntime();`,
	}}})
	if err == nil || !strings.Contains(err.Error(), `target-language global "adapter_global_collision" declares "hsm", which collides with compiler-owned typescript runtime binding`) {
		t.Fatalf("error = %v, want runtime binding collision rejection", err)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("failed patch mutated globals: %#v", program.Globals)
	}
}

func TestApplyBehaviorPatchRejectsTargetImportGlobalDeclarationCollisions(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{
		Globals: []CodeBlock{{
			ID:   "adapter_global_format",
			Code: `const format = (value: string): string => value;`,
		}},
		Behaviors: []Behavior{{
			ID:      "entry_1",
			Body:    `format("entry");`,
			Imports: []Import{{Path: "./format", Specifiers: []ImportSpecifier{{Name: "format"}}}},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), `target imports for behavior "entry_1" import "./format" bind local name "format", which collides with target-language global "adapter_global_format"`) {
		t.Fatalf("error = %v, want import/global declaration collision rejection", err)
	}
	if len(program.Globals) != 0 {
		t.Fatalf("failed patch mutated globals: %#v", program.Globals)
	}
}

func TestApplyBehaviorPatchRejectsPythonGlobalChainedAndTupleGeneratedSymbolCollisions(t *testing.T) {
	cases := []struct {
		name string
		code string
		want string
	}{
		{
			name: "chained behavior",
			code: "entry_1 = helper = lambda ctx, instance, event: None",
			want: `adapter-created global "adapter_global_collision" declares "entry_1", which collides with compiler-owned python symbol`,
		},
		{
			name: "tuple model",
			code: "door_model, helper = make_helpers()",
			want: `adapter-created global "adapter_global_collision" declares "door_model", which collides with compiler-owned python symbol`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := &Program{
				Models:    []Model{{ID: "model_1", Name: "Door"}},
				Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo}},
			}
			request := AdapterRequest{TargetLanguage: LanguagePython, Behaviors: program.Behaviors}

			err := applyBehaviorPatch(program, request, &BehaviorPatch{Globals: []CodeBlock{{
				ID:   "adapter_global_collision",
				Code: tc.code,
			}}})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestApplyBehaviorPatchRejectsESGlobalDestructuringGeneratedSymbolCollisions(t *testing.T) {
	cases := []struct {
		name   string
		target Language
		code   string
		want   string
	}{
		{
			name:   "typescript object destructured behavior",
			target: LanguageTS,
			code:   "const { entry1 } = helpers;",
			want:   `adapter-created global "adapter_global_collision" declares "entry1", which collides with compiler-owned typescript symbol`,
		},
		{
			name:   "typescript object destructured model alias",
			target: LanguageTS,
			code:   "const { model: DoorModel } = helpers;",
			want:   `adapter-created global "adapter_global_collision" declares "DoorModel", which collides with compiler-owned typescript symbol`,
		},
		{
			name:   "javascript array destructured behavior",
			target: LanguageJS,
			code:   "const [entry1] = helpers;",
			want:   `adapter-created global "adapter_global_collision" declares "entry1", which collides with compiler-owned javascript symbol`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := &Program{
				Models:    []Model{{ID: "model_1", Name: "Door"}},
				Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo}},
			}
			request := AdapterRequest{TargetLanguage: tc.target, Behaviors: program.Behaviors}

			err := applyBehaviorPatch(program, request, &BehaviorPatch{Globals: []CodeBlock{{
				ID:   "adapter_global_collision",
				Code: tc.code,
			}}})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestApplyBehaviorPatchRejectsAdapterHelperGlobalGeneratedSymbolCollisions(t *testing.T) {
	tests := []struct {
		name   string
		target Language
		code   string
		want   string
	}{
		{
			name:   "typescript behavior",
			target: LanguageTS,
			code:   `const entry1 = () => "helper";`,
			want:   `adapter-created global "adapter_global_collision" declares "entry1", which collides with compiler-owned typescript symbol`,
		},
		{
			name:   "javascript model",
			target: LanguageJS,
			code:   `function DoorModel() { return null; }`,
			want:   `adapter-created global "adapter_global_collision" declares "DoorModel", which collides with compiler-owned javascript symbol`,
		},
		{
			name:   "javascript model class",
			target: LanguageJS,
			code:   `class DoorModel {}`,
			want:   `adapter-created global "adapter_global_collision" declares "DoorModel", which collides with compiler-owned javascript symbol`,
		},
		{
			name:   "typescript event",
			target: LanguageTS,
			code:   `const openEvent = "helper";`,
			want:   `adapter-created global "adapter_global_collision" declares "openEvent", which collides with compiler-owned typescript symbol`,
		},
		{
			name:   "typescript named import behavior alias",
			target: LanguageTS,
			code:   `import { record as entry1 } from "./helpers";`,
			want:   `adapter-created global "adapter_global_collision" declares "entry1", which collides with compiler-owned typescript symbol`,
		},
		{
			name:   "javascript namespace import model",
			target: LanguageJS,
			code:   `import * as DoorModel from "./model";`,
			want:   `adapter-created global "adapter_global_collision" declares "DoorModel", which collides with compiler-owned javascript symbol`,
		},
		{
			name:   "typescript declare event",
			target: LanguageTS,
			code:   `declare const openEvent: hsm.Event;`,
			want:   `adapter-created global "adapter_global_collision" declares "openEvent", which collides with compiler-owned typescript symbol`,
		},
		{
			name:   "typescript model interface",
			target: LanguageTS,
			code:   `interface DoorModel { ready: boolean; }`,
			want:   `adapter-created global "adapter_global_collision" declares "DoorModel", which collides with compiler-owned typescript symbol`,
		},
		{
			name:   "typescript exported declare class",
			target: LanguageTS,
			code:   `export declare class DoorModel { ready: boolean; }`,
			want:   `adapter-created global "adapter_global_collision" declares "DoorModel", which collides with compiler-owned typescript symbol`,
		},
		{
			name:   "typescript namespace model",
			target: LanguageTS,
			code:   `export declare namespace DoorModel { const ready: boolean; }`,
			want:   `adapter-created global "adapter_global_collision" declares "DoorModel", which collides with compiler-owned typescript symbol`,
		},
		{
			name:   "dart external behavior declaration",
			target: LanguageDart,
			code:   `external FutureOr<void> entry1(Context ctx, Instance instance, Event event);`,
			want:   `adapter-created global "adapter_global_collision" declares "entry1", which collides with compiler-owned dart symbol`,
		},
		{
			name:   "typescript model type alias",
			target: LanguageTS,
			code:   `type DoorModel = { ready: boolean };`,
			want:   `adapter-created global "adapter_global_collision" declares "DoorModel", which collides with compiler-owned typescript symbol`,
		},
		{
			name:   "typescript later behavior declaration",
			target: LanguageTS,
			code: `const helper = (value: string): string => value;
function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {}`,
			want: `adapter-created global "adapter_global_collision" declares "entry1", which collides with compiler-owned typescript symbol`,
		},
		{
			name:   "typescript same statement behavior declaration",
			target: LanguageTS,
			code:   `const helper = (value: string): string => value, entry1 = (ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void => {};`,
			want:   `adapter-created global "adapter_global_collision" declares "entry1", which collides with compiler-owned typescript symbol`,
		},
		{
			name:   "go behavior",
			target: LanguageGo,
			code:   `func Entry1() {}`,
			want:   `adapter-created global "adapter_global_collision" declares "Entry1", which collides with compiler-owned go symbol`,
		},
		{
			name:   "go behavior type",
			target: LanguageGo,
			code:   `type Entry1 struct{}`,
			want:   `adapter-created global "adapter_global_collision" declares "Entry1", which collides with compiler-owned go symbol`,
		},
		{
			name:   "go grouped behavior",
			target: LanguageGo,
			code: `var (
	Helper = func(value string) string { return value }
	Entry1 = func(ctx context.Context, instance hsm.Instance, event hsm.Event) {}
)`,
			want: `adapter-created global "adapter_global_collision" declares "Entry1", which collides with compiler-owned go symbol`,
		},
		{
			name:   "go grouped type behavior",
			target: LanguageGo,
			code: `type (
	Helper struct{}
	Entry1 struct{}
)`,
			want: `adapter-created global "adapter_global_collision" declares "Entry1", which collides with compiler-owned go symbol`,
		},
		{
			name:   "go same statement behavior",
			target: LanguageGo,
			code:   `var Helper, Entry1 = func(value string) string { return value }, func(ctx context.Context, instance hsm.Instance, event hsm.Event) {}`,
			want:   `adapter-created global "adapter_global_collision" declares "Entry1", which collides with compiler-owned go symbol`,
		},
		{
			name:   "go grouped same line behavior",
			target: LanguageGo,
			code: `var (
	Helper, Entry1 = func(value string) string { return value }, func(ctx context.Context, instance hsm.Instance, event hsm.Event) {}
)`,
			want: `adapter-created global "adapter_global_collision" declares "Entry1", which collides with compiler-owned go symbol`,
		},
		{
			name:   "go aliased import behavior",
			target: LanguageGo,
			code:   `import Entry1 "example.com/helpers"`,
			want:   `adapter-created global "adapter_global_collision" declares "Entry1", which collides with compiler-owned go symbol`,
		},
		{
			name:   "go grouped import behavior",
			target: LanguageGo,
			code: `import (
	Helper "example.com/helper"
	Entry1 "example.com/entry"
)`,
			want: `adapter-created global "adapter_global_collision" declares "Entry1", which collides with compiler-owned go symbol`,
		},
		{
			name:   "go later grouped behavior",
			target: LanguageGo,
			code: `func Helper(value string) string { return value }

var (
	Entry1 = func(ctx context.Context, instance hsm.Instance, event hsm.Event) {}
)`,
			want: `adapter-created global "adapter_global_collision" declares "Entry1", which collides with compiler-owned go symbol`,
		},
		{
			name:   "python model",
			target: LanguagePython,
			code:   "def door_model():\n\tpass",
			want:   `adapter-created global "adapter_global_collision" declares "door_model", which collides with compiler-owned python symbol`,
		},
		{
			name:   "python model class",
			target: LanguagePython,
			code:   "class door_model:\n\tpass",
			want:   `adapter-created global "adapter_global_collision" declares "door_model", which collides with compiler-owned python symbol`,
		},
		{
			name:   "python model assignment",
			target: LanguagePython,
			code:   `door_model = hsm.define_model("Door")`,
			want:   `adapter-created global "adapter_global_collision" declares "door_model", which collides with compiler-owned python symbol`,
		},
		{
			name:   "python model annotated assignment",
			target: LanguagePython,
			code:   `door_model: hsm.Model = hsm.define_model("Door")`,
			want:   `adapter-created global "adapter_global_collision" declares "door_model", which collides with compiler-owned python symbol`,
		},
		{
			name:   "python from import behavior alias",
			target: LanguagePython,
			code:   `from helpers import record as entry_1`,
			want:   `adapter-created global "adapter_global_collision" declares "entry_1", which collides with compiler-owned python symbol`,
		},
		{
			name:   "python parenthesized from import behavior alias",
			target: LanguagePython,
			code: `from helpers import (
	record as entry_1,
)`,
			want: `adapter-created global "adapter_global_collision" declares "entry_1", which collides with compiler-owned python symbol`,
		},
		{
			name:   "python import model alias",
			target: LanguagePython,
			code:   `import helpers.models as door_model`,
			want:   `adapter-created global "adapter_global_collision" declares "door_model", which collides with compiler-owned python symbol`,
		},
		{
			name:   "dart event",
			target: LanguageDart,
			code:   `final openEvent = "helper";`,
			want:   `adapter-created global "adapter_global_collision" declares "openEvent", which collides with compiler-owned dart symbol`,
		},
		{
			name:   "dart typed event without storage keyword",
			target: LanguageDart,
			code:   `Event openEvent = Event(name: "helper");`,
			want:   `adapter-created global "adapter_global_collision" declares "openEvent", which collides with compiler-owned dart symbol`,
		},
		{
			name:   "dart typed model without storage keyword",
			target: LanguageDart,
			code:   `Model doorModel = define("Door", []);`,
			want:   `adapter-created global "adapter_global_collision" declares "doorModel", which collides with compiler-owned dart symbol`,
		},
		{
			name:   "dart prefixed import behavior",
			target: LanguageDart,
			code:   `import 'helpers.dart' as entry1;`,
			want:   `adapter-created global "adapter_global_collision" declares "entry1", which collides with compiler-owned dart symbol`,
		},
		{
			name:   "dart show import model",
			target: LanguageDart,
			code:   `import 'helpers.dart' show doorModel;`,
			want:   `adapter-created global "adapter_global_collision" declares "doorModel", which collides with compiler-owned dart symbol`,
		},
		{
			name:   "dart behavior type",
			target: LanguageDart,
			code:   `class entry1 {}`,
			want:   `adapter-created global "adapter_global_collision" declares "entry1", which collides with compiler-owned dart symbol`,
		},
		{
			name:   "csharp behavior",
			target: LanguageCSharp,
			code:   `private static void Entry1(Context ctx, Instance instance, Event @event) {}`,
			want:   `adapter-created global "adapter_global_collision" declares "Entry1", which collides with compiler-owned csharp symbol`,
		},
		{
			name:   "csharp attributed behavior",
			target: LanguageCSharp,
			code:   `[Obsolete] private static void Entry1(Context ctx, Instance instance, Event @event) {}`,
			want:   `adapter-created global "adapter_global_collision" declares "Entry1", which collides with compiler-owned csharp symbol`,
		},
		{
			name:   "csharp alias using behavior",
			target: LanguageCSharp,
			code:   `using Entry1 = Sample.Helpers.Entry1;`,
			want:   `adapter-created global "adapter_global_collision" declares "Entry1", which collides with compiler-owned csharp symbol`,
		},
		{
			name:   "csharp extern alias behavior",
			target: LanguageCSharp,
			code:   `extern alias Entry1;`,
			want:   `adapter-created global "adapter_global_collision" declares "Entry1", which collides with compiler-owned csharp symbol`,
		},
		{
			name:   "csharp model",
			target: LanguageCSharp,
			code:   `private static readonly Model DoorModel = null;`,
			want:   `adapter-created global "adapter_global_collision" declares "DoorModel", which collides with compiler-owned csharp symbol`,
		},
		{
			name:   "csharp attributed model field",
			target: LanguageCSharp,
			code:   `[Obsolete] private static readonly Model DoorModel = null;`,
			want:   `adapter-created global "adapter_global_collision" declares "DoorModel", which collides with compiler-owned csharp symbol`,
		},
		{
			name:   "csharp event",
			target: LanguageCSharp,
			code:   `private static readonly Event OpenEvent = null;`,
			want:   `adapter-created global "adapter_global_collision" declares "OpenEvent", which collides with compiler-owned csharp symbol`,
		},
		{
			name:   "csharp generated class",
			target: LanguageCSharp,
			code:   `public static class GeneratedHsm {}`,
			want:   `adapter-created global "adapter_global_collision" declares "GeneratedHsm", which collides with compiler-owned csharp symbol`,
		},
		{
			name:   "csharp generated static constructor",
			target: LanguageCSharp,
			code:   `static GeneratedHsm() {}`,
			want:   `adapter-created global "adapter_global_collision" declares "GeneratedHsm", which collides with compiler-owned csharp symbol`,
		},
		{
			name:   "csharp attributed generated static constructor",
			target: LanguageCSharp,
			code:   `[Obsolete] static GeneratedHsm() {}`,
			want:   `adapter-created global "adapter_global_collision" declares "GeneratedHsm", which collides with compiler-owned csharp symbol`,
		},
		{
			name:   "csharp behavior delegate",
			target: LanguageCSharp,
			code:   `public delegate void Entry1(Context ctx, Instance instance, Event @event);`,
			want:   `adapter-created global "adapter_global_collision" declares "Entry1", which collides with compiler-owned csharp symbol`,
		},
		{
			name:   "csharp attributed behavior delegate",
			target: LanguageCSharp,
			code:   `[Obsolete] public delegate void Entry1(Context ctx, Instance instance, Event @event);`,
			want:   `adapter-created global "adapter_global_collision" declares "Entry1", which collides with compiler-owned csharp symbol`,
		},
		{
			name:   "csharp model record struct",
			target: LanguageCSharp,
			code:   `public readonly record struct DoorModel(string Name);`,
			want:   `adapter-created global "adapter_global_collision" declares "DoorModel", which collides with compiler-owned csharp symbol`,
		},
		{
			name:   "csharp attributed model record struct",
			target: LanguageCSharp,
			code:   `[Obsolete] public readonly record struct DoorModel(string Name);`,
			want:   `adapter-created global "adapter_global_collision" declares "DoorModel", which collides with compiler-owned csharp symbol`,
		},
		{
			name:   "java behavior",
			target: LanguageJava,
			code:   `private static void Entry1(Context ctx, Instance instance, Event event) {}`,
			want:   `adapter-created global "adapter_global_collision" declares "Entry1", which collides with compiler-owned java symbol`,
		},
		{
			name:   "java annotated behavior",
			target: LanguageJava,
			code:   `@Deprecated private static void Entry1(Context ctx, Instance instance, Event event) {}`,
			want:   `adapter-created global "adapter_global_collision" declares "Entry1", which collides with compiler-owned java symbol`,
		},
		{
			name:   "java model",
			target: LanguageJava,
			code:   `private static final Model DoorModel = null;`,
			want:   `adapter-created global "adapter_global_collision" declares "DoorModel", which collides with compiler-owned java symbol`,
		},
		{
			name:   "java annotated model field",
			target: LanguageJava,
			code:   `@Deprecated private static final Model DoorModel = null;`,
			want:   `adapter-created global "adapter_global_collision" declares "DoorModel", which collides with compiler-owned java symbol`,
		},
		{
			name:   "java event",
			target: LanguageJava,
			code:   `private static final Event OpenEvent = null;`,
			want:   `adapter-created global "adapter_global_collision" declares "OpenEvent", which collides with compiler-owned java symbol`,
		},
		{
			name:   "java import behavior",
			target: LanguageJava,
			code:   `import com.example.Entry1;`,
			want:   `adapter-created global "adapter_global_collision" declares "Entry1", which collides with compiler-owned java symbol`,
		},
		{
			name:   "java static import behavior",
			target: LanguageJava,
			code:   `import static com.example.Helpers.Entry1;`,
			want:   `adapter-created global "adapter_global_collision" declares "Entry1", which collides with compiler-owned java symbol`,
		},
		{
			name:   "java generated class",
			target: LanguageJava,
			code:   `public final class GeneratedHsm {}`,
			want:   `adapter-created global "adapter_global_collision" declares "GeneratedHsm", which collides with compiler-owned java symbol`,
		},
		{
			name:   "java annotated generated class",
			target: LanguageJava,
			code:   `@Deprecated public final class GeneratedHsm {}`,
			want:   `adapter-created global "adapter_global_collision" declares "GeneratedHsm", which collides with compiler-owned java symbol`,
		},
		{
			name:   "java generated package-private constructor",
			target: LanguageJava,
			code:   `GeneratedHsm() {}`,
			want:   `adapter-created global "adapter_global_collision" declares "GeneratedHsm", which collides with compiler-owned java symbol`,
		},
		{
			name:   "java annotated generated constructor",
			target: LanguageJava,
			code:   `@SuppressWarnings("unused") GeneratedHsm() {}`,
			want:   `adapter-created global "adapter_global_collision" declares "GeneratedHsm", which collides with compiler-owned java symbol`,
		},
		{
			name:   "java model annotation",
			target: LanguageJava,
			code:   `public @interface DoorModel {}`,
			want:   `adapter-created global "adapter_global_collision" declares "DoorModel", which collides with compiler-owned java symbol`,
		},
		{
			name:   "cpp behavior",
			target: LanguageCPP,
			code:   `static constexpr auto entry_1 = []() {};`,
			want:   `adapter-created global "adapter_global_collision" declares "entry_1", which collides with compiler-owned cpp symbol`,
		},
		{
			name:   "cpp same statement behavior",
			target: LanguageCPP,
			code:   `static constexpr auto helper = 0, entry_1 = []() {};`,
			want:   `adapter-created global "adapter_global_collision" declares "entry_1", which collides with compiler-owned cpp symbol`,
		},
		{
			name:   "cpp model",
			target: LanguageCPP,
			code:   `static constexpr auto door_model = nullptr;`,
			want:   `adapter-created global "adapter_global_collision" declares "door_model", which collides with compiler-owned cpp symbol`,
		},
		{
			name:   "cpp typed behavior variable",
			target: LanguageCPP,
			code:   `inline constexpr int entry_1 = 0;`,
			want:   `adapter-created global "adapter_global_collision" declares "entry_1", which collides with compiler-owned cpp symbol`,
		},
		{
			name:   "cpp pointer model variable",
			target: LanguageCPP,
			code:   `hsm::Model *door_model = nullptr;`,
			want:   `adapter-created global "adapter_global_collision" declares "door_model", which collides with compiler-owned cpp symbol`,
		},
		{
			name:   "cpp same statement pointer model variable",
			target: LanguageCPP,
			code:   `hsm::Model *helper = nullptr, *door_model = nullptr;`,
			want:   `adapter-created global "adapter_global_collision" declares "door_model", which collides with compiler-owned cpp symbol`,
		},
		{
			name:   "cpp event type",
			target: LanguageCPP,
			code:   `struct OpenEvent {};`,
			want:   `adapter-created global "adapter_global_collision" declares "OpenEvent", which collides with compiler-owned cpp symbol`,
		},
		{
			name:   "cpp event enum class",
			target: LanguageCPP,
			code:   `enum class OpenEvent { Open };`,
			want:   `adapter-created global "adapter_global_collision" declares "OpenEvent", which collides with compiler-owned cpp symbol`,
		},
		{
			name:   "cpp event union",
			target: LanguageCPP,
			code:   `union OpenEvent { int value; };`,
			want:   `adapter-created global "adapter_global_collision" declares "OpenEvent", which collides with compiler-owned cpp symbol`,
		},
		{
			name:   "cpp behavior namespace",
			target: LanguageCPP,
			code:   `namespace entry_1 {}`,
			want:   `adapter-created global "adapter_global_collision" declares "entry_1", which collides with compiler-owned cpp symbol`,
		},
		{
			name:   "cpp model using alias",
			target: LanguageCPP,
			code:   `using door_model = hsm::Model;`,
			want:   `adapter-created global "adapter_global_collision" declares "door_model", which collides with compiler-owned cpp symbol`,
		},
		{
			name:   "cpp model typedef",
			target: LanguageCPP,
			code:   `typedef hsm::Model door_model;`,
			want:   `adapter-created global "adapter_global_collision" declares "door_model", which collides with compiler-owned cpp symbol`,
		},
		{
			name:   "rust behavior",
			target: LanguageRust,
			code:   `fn entry_1(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) {}`,
			want:   `adapter-created global "adapter_global_collision" declares "entry_1", which collides with compiler-owned rust symbol`,
		},
		{
			name:   "rust event",
			target: LanguageRust,
			code:   `const open_event: &str = "helper";`,
			want:   `adapter-created global "adapter_global_collision" declares "open_event", which collides with compiler-owned rust symbol`,
		},
		{
			name:   "rust mutable static behavior",
			target: LanguageRust,
			code:   `pub(crate) static mut entry_1: Option<fn(&hsm::Context, &mut HsmcInstance, &hsm::Event)> = None;`,
			want:   `adapter-created global "adapter_global_collision" declares "entry_1", which collides with compiler-owned rust symbol`,
		},
		{
			name:   "rust use alias behavior",
			target: LanguageRust,
			code:   `pub use crate::helpers::record as entry_1;`,
			want:   `adapter-created global "adapter_global_collision" declares "entry_1", which collides with compiler-owned rust symbol`,
		},
		{
			name:   "rust grouped use behavior",
			target: LanguageRust,
			code:   `use crate::helpers::{helper, entry_1};`,
			want:   `adapter-created global "adapter_global_collision" declares "entry_1", which collides with compiler-owned rust symbol`,
		},
		{
			name:   "rust scaffold",
			target: LanguageRust,
			code:   `pub struct HsmcInstance;`,
			want:   `adapter-created global "adapter_global_collision" declares "HsmcInstance", which collides with compiler-owned rust symbol`,
		},
		{
			name:   "rust scaffold trait",
			target: LanguageRust,
			code:   `pub trait HsmcInstance {}`,
			want:   `adapter-created global "adapter_global_collision" declares "HsmcInstance", which collides with compiler-owned rust symbol`,
		},
		{
			name:   "rust behavior module",
			target: LanguageRust,
			code:   `pub mod entry_1 {}`,
			want:   `adapter-created global "adapter_global_collision" declares "entry_1", which collides with compiler-owned rust symbol`,
		},
		{
			name:   "rust scaffold union",
			target: LanguageRust,
			code:   `pub union HsmcInstance { ready: bool }`,
			want:   `adapter-created global "adapter_global_collision" declares "HsmcInstance", which collides with compiler-owned rust symbol`,
		},
		{
			name:   "zig behavior",
			target: LanguageZig,
			code:   `fn entry_1(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void {}`,
			want:   `adapter-created global "adapter_global_collision" declares "entry_1", which collides with compiler-owned zig symbol`,
		},
		{
			name:   "zig model",
			target: LanguageZig,
			code:   `pub const door_model = hsm.define("helper", .{});`,
			want:   `adapter-created global "adapter_global_collision" declares "door_model", which collides with compiler-owned zig symbol`,
		},
		{
			name:   "zig event",
			target: LanguageZig,
			code:   `pub const open_event = "helper";`,
			want:   `adapter-created global "adapter_global_collision" declares "open_event", which collides with compiler-owned zig symbol`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := &Program{
				Events:    []EventDecl{{ID: "event_1", Symbol: "open_event", Name: "open"}},
				Models:    []Model{{ID: "model_1", Name: "Door"}},
				Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo}},
			}
			request := AdapterRequest{TargetLanguage: tt.target, Behaviors: program.Behaviors}

			err := applyBehaviorPatch(program, request, &BehaviorPatch{Globals: []CodeBlock{{
				ID:   "adapter_global_collision",
				Code: tt.code,
			}}})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestApplyBehaviorPatchRejectsAdapterHelperGlobalPythonTriggerFallbackCollision(t *testing.T) {
	program := &Program{
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			States: []State{{
				ID:   "state_1",
				Name: "closed",
				Transitions: []Transition{{
					ID:      "transition_1",
					OwnerID: "state_1",
					Target:  "../open",
					Trigger: &Trigger{
						Kind:     TriggerWhen,
						Value:    "ready()",
						Language: LanguageTS,
					},
				}},
			}},
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguagePython, Models: BuildModelViews(program)}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{Globals: []CodeBlock{{
		ID:   "adapter_global_collision",
		Code: "def hsmc_trigger_fallback_1(ctx, instance, event):\n\treturn True",
	}}})
	if err == nil || !strings.Contains(err.Error(), `adapter-created global "adapter_global_collision" declares "hsmc_trigger_fallback_1", which collides with compiler-owned python symbol`) {
		t.Fatalf("error = %v, want Python trigger fallback collision", err)
	}
}

func TestApplyBehaviorPatchRejectsAdapterHelperGlobalZigOperationFallbackCollision(t *testing.T) {
	fallbackName := zigOperationReferenceFallbackName(BehaviorEntry, "approve")
	program := &Program{
		Models: []Model{{
			ID:   "model_1",
			Name: "Door",
			Operations: []Operation{{
				ID:   "operation_1",
				Name: "approve",
			}},
			States: []State{{
				ID:        "state_1",
				Name:      "closed",
				Behaviors: []BehaviorRef{{Kind: BehaviorEntry, Operation: "approve"}},
			}},
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguageZig, Models: BuildModelViews(program)}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{Globals: []CodeBlock{{
		ID:   "adapter_global_collision",
		Code: "fn " + fallbackName + "() void {}",
	}}})
	if err == nil || !strings.Contains(err.Error(), `adapter-created global "adapter_global_collision" declares "`+fallbackName+`", which collides with compiler-owned zig symbol`) {
		t.Fatalf("error = %v, want Zig operation fallback collision", err)
	}
}

func TestApplyBehaviorPatchRejectsTranslatedGlobalGeneratedSymbolCollisions(t *testing.T) {
	program := &Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageGo,
			Code:     "func helper() {}",
		}},
		Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo}},
	}
	request := AdapterRequest{
		TargetLanguage: LanguageTS,
		Globals:        program.Globals,
		Behaviors:      program.Behaviors,
	}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{Globals: []CodeBlock{{
		ID:   "global_1",
		Code: "function entry1(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void {}",
	}}})
	want := `target-language global "global_1" declares "entry1", which collides with compiler-owned typescript symbol`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestApplyBehaviorPatchAllowsProgramImportsForAdapterHelperGlobalCode(t *testing.T) {
	program := &Program{
		Imports: []Import{{Path: "fmt", Language: LanguageGo}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Imports: program.Imports}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{
		Imports: []Import{{
			Path:       "./helper-runtime",
			Specifiers: []ImportSpecifier{{Name: "normalize"}},
		}},
		Globals: []CodeBlock{{
			ID:   "adapter_global_format",
			Code: `const adapterFormat = (value: string): string => normalize(value);`,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 2 {
		t.Fatalf("imports = %#v, want source import plus helper target import", program.Imports)
	}
	if program.Imports[0].Path != "fmt" || program.Imports[0].Language != LanguageGo {
		t.Fatalf("source import was not preserved: %#v", program.Imports)
	}
	if program.Imports[1].Path != "./helper-runtime" || program.Imports[1].Language != LanguageTS || len(program.Imports[1].Specifiers) != 1 {
		t.Fatalf("target helper import was not normalized: %#v", program.Imports[1])
	}
	if len(program.Globals) != 1 || program.Globals[0].ID != "adapter_global_format" || program.Globals[0].Language != LanguageTS {
		t.Fatalf("helper global = %#v", program.Globals)
	}
}

func TestApplyBehaviorPatchAllowsJavaStaticTargetImports(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguageJava, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:      "entry_1",
			Body:    `record("closed");`,
			Imports: []Import{{Path: "com.example.Format.record", Static: true}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Behaviors) != 1 || len(program.Behaviors[0].Imports) != 1 {
		t.Fatalf("behavior imports = %#v", program.Behaviors)
	}
	imp := program.Behaviors[0].Imports[0]
	if imp.Path != "com.example.Format.record" || !imp.Static || imp.Language != LanguageJava {
		t.Fatalf("static target import was not normalized: %#v", imp)
	}
}

func TestApplyBehaviorPatchAllowsCSharpStaticTargetImports(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguageCSharp, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:      "entry_1",
			Body:    `Record("closed");`,
			Imports: []Import{{Path: "Sample.Formatting.Helper", Static: true}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Behaviors) != 1 || len(program.Behaviors[0].Imports) != 1 {
		t.Fatalf("behavior imports = %#v", program.Behaviors)
	}
	imp := program.Behaviors[0].Imports[0]
	if imp.Path != "Sample.Formatting.Helper" || !imp.Static || imp.Language != LanguageCSharp {
		t.Fatalf("static target import was not normalized: %#v", imp)
	}

	output, err := NewCSharpBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, "using static Sample.Formatting.Helper;") {
		t.Fatalf("C# output missing adapter static using:\n%s", text)
	}
	if strings.Contains(text, "using Sample.Formatting.Helper;") {
		t.Fatalf("C# output rendered adapter static using as ordinary using:\n%s", text)
	}
}

func TestApplyBehaviorPatchRejectsCSharpStaticAliasTargetImport(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguageCSharp, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:      "entry_1",
			Body:    `Record("closed");`,
			Imports: []Import{{Path: "Sample.Formatting.Helper", Static: true, Alias: "Format"}},
		}},
	})
	want := `adapter patch behavior "entry_1" import "Sample.Formatting.Helper" combines static using with alias`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestApplyBehaviorPatchAllowsCSharpStaticImportTypeNameMatchingGeneratedSymbol(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguageCSharp, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{
		Behaviors: []Behavior{{
			ID:      "entry_1",
			Body:    `Record("closed");`,
			Imports: []Import{{Path: "Sample.Formatting.Entry1", Static: true}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	output, err := NewCSharpBackend().Emit(context.Background(), program)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, "using static Sample.Formatting.Entry1;") {
		t.Fatalf("C# output missing static using:\n%s", text)
	}
	if strings.Contains(text, "using Sample.Formatting.Entry1;") {
		t.Fatalf("C# output rendered static using as ordinary using:\n%s", text)
	}
}

func TestApplyBehaviorPatchExplicitEmptyProgramImportsClearsTargetImportsWithHelperGlobalCode(t *testing.T) {
	program := &Program{
		Imports: []Import{
			{Path: "fmt", Language: LanguageGo},
			{Path: "./stale-helper-runtime", Language: LanguageTS},
		},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Imports: program.Imports}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{
		Imports: []Import{},
		Globals: []CodeBlock{{
			ID:   "adapter_global_format",
			Code: `const adapterFormat = (value: string): string => value;`,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Imports) != 1 || program.Imports[0].Path != "fmt" || program.Imports[0].Language != LanguageGo {
		t.Fatalf("imports = %#v, want only preserved non-target source import", program.Imports)
	}
	if len(program.Globals) != 1 || program.Globals[0].ID != "adapter_global_format" || program.Globals[0].Language != LanguageTS {
		t.Fatalf("helper global = %#v", program.Globals)
	}
}

func TestApplyBehaviorPatchRejectsMalformedAdapterHelperGlobals(t *testing.T) {
	program := &Program{}
	request := AdapterRequest{TargetLanguage: LanguageTS}

	cases := []struct {
		name  string
		patch BehaviorPatch
		want  string
	}{
		{
			name:  "missing code",
			patch: BehaviorPatch{Globals: []CodeBlock{{ID: "adapter_global_empty"}}},
			want:  `adapter-created global "adapter_global_empty" is missing target code`,
		},
		{
			name:  "empty helper suffix",
			patch: BehaviorPatch{Globals: []CodeBlock{{ID: "adapter_global_", Code: "const helper = true;"}}},
			want:  `adapter-created global id "adapter_global_" is missing a helper name`,
		},
		{
			name:  "leading padded id",
			patch: BehaviorPatch{Globals: []CodeBlock{{ID: " adapter_global_padded", Code: "const helper = () => {};"}}},
			want:  `adapter patch global id " adapter_global_padded" has leading or trailing whitespace`,
		},
		{
			name:  "trailing padded id",
			patch: BehaviorPatch{Globals: []CodeBlock{{ID: "adapter_global_padded ", Code: "const helper = () => {};"}}},
			want:  `adapter patch global id "adapter_global_padded " has leading or trailing whitespace`,
		},
		{
			name:  "path-like helper id",
			patch: BehaviorPatch{Globals: []CodeBlock{{ID: "adapter_global_helpers/format", Code: "const helper = () => {};"}}},
			want:  `adapter-created global id "adapter_global_helpers/format" must contain only letters, digits, and underscores after adapter_global_`,
		},
		{
			name:  "punctuated helper id",
			patch: BehaviorPatch{Globals: []CodeBlock{{ID: "adapter_global_format.helper", Code: "const helper = () => {};"}}},
			want:  `adapter-created global id "adapter_global_format.helper" must contain only letters, digits, and underscores after adapter_global_`,
		},
		{
			name:  "language is compiler owned",
			patch: BehaviorPatch{Globals: []CodeBlock{{ID: "adapter_global_wrong_language", Language: LanguageGo, Code: "func helper() {}"}}},
			want:  `adapter patch global "adapter_global_wrong_language" attempts to change language`,
		},
		{
			name:  "bad import language",
			patch: BehaviorPatch{Globals: []CodeBlock{{ID: "adapter_global_bad_import", Code: "const helper = () => {};", Imports: []Import{{Path: "./go", Language: LanguageGo}}}}},
			want:  `adapter patch global "adapter_global_bad_import" import "./go" has language "go", want target language "typescript"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := applyBehaviorPatch(program, request, &tc.patch)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestApplyBehaviorPatchRejectsWrongTargetLanguages(t *testing.T) {
	program := &Program{
		Globals:   []CodeBlock{{ID: "global_1", Language: LanguageGo, Code: "func helper() {}"}},
		Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Globals: program.Globals, Behaviors: program.Behaviors}

	if err := applyBehaviorPatch(program, request, &BehaviorPatch{Imports: []Import{{Path: "./helpers", Language: LanguagePython}}}); err == nil {
		t.Fatal("expected wrong import language error")
	}
	if err := applyBehaviorPatch(program, request, &BehaviorPatch{Globals: []CodeBlock{{ID: "global_1", Language: LanguageGo, Code: "func helper() {}"}}}); err == nil {
		t.Fatal("expected wrong global language error")
	}
	if err := applyBehaviorPatch(program, request, &BehaviorPatch{Globals: []CodeBlock{{ID: "global_1", Code: "const helper = () => {};", Imports: []Import{{Path: "./go-helper", Language: LanguageGo}}}}}); err == nil {
		t.Fatal("expected wrong global import language error")
	}
	if err := applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{{ID: "entry_1", Body: "translated();", Imports: []Import{{Path: "./go-helper", Language: LanguageGo}}}}}); err == nil {
		t.Fatal("expected wrong behavior import language error")
	}
	if err := applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{{ID: "entry_1", TargetLanguage: LanguagePython, Body: "translated();"}}}); err == nil {
		t.Fatal("expected immutable behavior target language error")
	}
}

func TestApplyBehaviorPatchRejectsImportMetadataFields(t *testing.T) {
	program := &Program{
		Globals:   []CodeBlock{{ID: "global_1", Language: LanguageGo, Code: "func helper() {}"}},
		Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Globals: program.Globals, Behaviors: program.Behaviors}
	sourceRange := SourceRange{File: "adapter.ts", StartByte: 1, EndByte: 2, StartLine: 3, EndLine: 4}

	cases := []struct {
		name  string
		patch BehaviorPatch
		want  string
	}{
		{
			name:  "program local names",
			patch: BehaviorPatch{Imports: []Import{{Path: "./helpers", LocalNames: []string{"helper"}}}},
			want:  `adapter patch program import "./helpers" attempts to set local_names`,
		},
		{
			name:  "program range",
			patch: BehaviorPatch{Imports: []Import{{Path: "./helpers", Range: sourceRange}}},
			want:  `adapter patch program import "./helpers" attempts to change range`,
		},
		{
			name: "global local names",
			patch: BehaviorPatch{Globals: []CodeBlock{{
				ID:      "global_1",
				Code:    "const helper = () => true;",
				Imports: []Import{{Path: "./global-helper", LocalNames: []string{"globalHelper"}}},
			}}},
			want: `adapter patch global "global_1" import "./global-helper" attempts to set local_names`,
		},
		{
			name: "behavior range",
			patch: BehaviorPatch{Behaviors: []Behavior{{
				ID:      "entry_1",
				Body:    "translated();",
				Imports: []Import{{Path: "./behavior-helper", Range: sourceRange}},
			}}},
			want: `adapter patch behavior "entry_1" import "./behavior-helper" attempts to change range`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			copyProgram := cloneProgram(program)
			err := applyBehaviorPatch(copyProgram, request, &tc.patch)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestApplyBehaviorPatchCanonicalizesLanguageAliases(t *testing.T) {
	program := &Program{
		Globals:   []CodeBlock{{ID: "global_1", Language: LanguageGo, Code: "func helper() {}"}},
		Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Globals: program.Globals, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{
		Imports: []Import{{Path: "./program", Language: Language("ts")}},
		Globals: []CodeBlock{
			{
				ID:      "global_1",
				Code:    "const helper = () => {};",
				Imports: []Import{{Path: "./global", Language: Language("typescript")}},
			},
			{
				ID:      "adapter_global_helper",
				Code:    "const adapterHelper = true;",
				Imports: []Import{{Path: "./adapter-global", Language: Language("TS")}},
			},
		},
		Behaviors: []Behavior{{
			ID:      "entry_1",
			Body:    "translated();",
			Imports: []Import{{Path: "./behavior", Language: Language(" ts ")}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if program.Imports[0].Language != LanguageTS {
		t.Fatalf("program import language = %q, want %q", program.Imports[0].Language, LanguageTS)
	}
	if program.Globals[0].Language != LanguageTS || program.Globals[0].Imports[0].Language != LanguageTS {
		t.Fatalf("global patch = %#v", program.Globals[0])
	}
	if program.Globals[1].Language != LanguageTS || program.Globals[1].Imports[0].Language != LanguageTS {
		t.Fatalf("adapter global patch = %#v", program.Globals[1])
	}
	if program.Behaviors[0].TargetLanguage != LanguageTS || program.Behaviors[0].Imports[0].Language != LanguageTS {
		t.Fatalf("behavior patch = %#v", program.Behaviors[0])
	}
}

func TestApplyBehaviorPatchRejectsUnsupportedLanguageAlias(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{Imports: []Import{{Path: "./helpers", Language: Language("ruby")}}})
	if err == nil || !strings.Contains(err.Error(), `unsupported language "ruby"`) {
		t.Fatalf("error = %v, want unsupported language", err)
	}
}

func TestApplyBehaviorPatchRejectsDartCompilerOwnedImportCollisions(t *testing.T) {
	tests := []struct {
		name  string
		local string
	}{
		{name: "future or", local: "FutureOr"},
		{name: "on set", local: "onSet"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := &Program{
				Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo}},
			}
			request := AdapterRequest{TargetLanguage: LanguageDart, Behaviors: program.Behaviors}

			err := applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{{
				ID:   "entry_1",
				Body: `adapterFormat("entry");`,
				Imports: []Import{{
					Path:       "package:sample/helper.dart",
					Specifiers: []ImportSpecifier{{Name: tt.local}},
				}},
			}}})
			if err == nil || !strings.Contains(err.Error(), `both bind local name "`+tt.local+`"`) {
				t.Fatalf("error = %v, want Dart %s compiler-owned import collision", err, tt.local)
			}
		})
	}
}

func TestApplyBehaviorPatchRejectsMalformedTargetImports(t *testing.T) {
	program := &Program{
		Globals:   []CodeBlock{{ID: "global_1", Language: LanguageGo, Code: "func helper() {}"}},
		Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo}},
	}

	cases := []struct {
		name    string
		target  Language
		imports []Import
	}{
		{
			name:    "blank path",
			target:  LanguageTS,
			imports: []Import{{Path: " \n\t "}},
		},
		{
			name:    "path surrounding whitespace",
			target:  LanguageTS,
			imports: []Import{{Path: " ./helpers "}},
		},
		{
			name:    "blank alias",
			target:  LanguageTS,
			imports: []Import{{Path: "./helpers", Alias: " \n\t "}},
		},
		{
			name:    "alias surrounding whitespace",
			target:  LanguageTS,
			imports: []Import{{Path: "./helpers", Alias: " helpers "}},
		},
		{
			name:    "blank default",
			target:  LanguageTS,
			imports: []Import{{Path: "./helpers", Default: " \n\t "}},
		},
		{
			name:    "default surrounding whitespace",
			target:  LanguageTS,
			imports: []Import{{Path: "./helpers", Default: " helpers "}},
		},
		{
			name:    "empty specifier",
			target:  LanguageTS,
			imports: []Import{{Path: "./helpers", Specifiers: []ImportSpecifier{{}}}},
		},
		{
			name:    "specifier surrounding whitespace",
			target:  LanguageTS,
			imports: []Import{{Path: "./helpers", Specifiers: []ImportSpecifier{{Name: " format"}}}},
		},
		{
			name:    "blank specifier alias",
			target:  LanguageTS,
			imports: []Import{{Path: "./helpers", Specifiers: []ImportSpecifier{{Name: "format", Alias: " \n\t "}}}},
		},
		{
			name:    "specifier alias surrounding whitespace",
			target:  LanguageTS,
			imports: []Import{{Path: "./helpers", Specifiers: []ImportSpecifier{{Name: "format", Alias: " fmt "}}}},
		},
		{
			name:    "typescript namespace plus named",
			target:  LanguageTS,
			imports: []Import{{Path: "./helpers", Alias: "helpers", Specifiers: []ImportSpecifier{{Name: "record"}}}},
		},
		{
			name:    "typescript duplicate local binding",
			target:  LanguageTS,
			imports: []Import{{Path: "./helpers", Specifiers: []ImportSpecifier{{Name: "record", Alias: "helper"}, {Name: "format", Alias: "helper"}}}},
		},
		{
			name:    "typescript default plus duplicate named local binding",
			target:  LanguageTS,
			imports: []Import{{Path: "./helpers", Default: "helper", Specifiers: []ImportSpecifier{{Name: "record", Alias: "helper"}}}},
		},
		{
			name:    "python default",
			target:  LanguagePython,
			imports: []Import{{Path: "helpers", Default: "helpers"}},
		},
		{
			name:    "python alias plus specifier",
			target:  LanguagePython,
			imports: []Import{{Path: "helpers", Alias: "h", Specifiers: []ImportSpecifier{{Name: "record"}}}},
		},
		{
			name:    "python wildcard specifier",
			target:  LanguagePython,
			imports: []Import{{Path: "helpers", Specifiers: []ImportSpecifier{{Name: "*"}}}},
		},
		{
			name:    "dart specifier alias",
			target:  LanguageDart,
			imports: []Import{{Path: "helpers.dart", Specifiers: []ImportSpecifier{{Name: "record", Alias: "rec"}}}},
		},
		{
			name:    "dart deferred without alias",
			target:  LanguageDart,
			imports: []Import{{Path: "helpers.dart", Deferred: true}},
		},
		{
			name:    "dart alias plus show specifier",
			target:  LanguageDart,
			imports: []Import{{Path: "helpers.dart", Alias: "helpers", Specifiers: []ImportSpecifier{{Name: "record"}}}},
		},
		{
			name:    "dart alias plus hidden specifier",
			target:  LanguageDart,
			imports: []Import{{Path: "helpers.dart", Alias: "helpers", Hidden: []string{"debugRecord"}}},
		},
		{
			name:    "dart hidden specifier",
			target:  LanguageDart,
			imports: []Import{{Path: "helpers.dart", Hidden: []string{"debugRecord"}}},
		},
		{
			name:    "rust alias plus specifier",
			target:  LanguageRust,
			imports: []Import{{Path: "crate::helpers", Alias: "helpers", Specifiers: []ImportSpecifier{{Name: "record"}}}},
		},
		{
			name:    "typescript deferred import",
			target:  LanguageTS,
			imports: []Import{{Path: "./helpers", Deferred: true}},
		},
		{
			name:    "javascript type-only import",
			target:  LanguageJS,
			imports: []Import{{Path: "./types", TypeOnly: true}},
		},
		{
			name:    "javascript type-only specifier",
			target:  LanguageJS,
			imports: []Import{{Path: "./types", Specifiers: []ImportSpecifier{{Name: "DoorSnapshot", TypeOnly: true}}}},
		},
		{
			name:    "python duplicate local binding",
			target:  LanguagePython,
			imports: []Import{{Path: "helpers", Specifiers: []ImportSpecifier{{Name: "record", Alias: "helper"}, {Name: "format", Alias: "helper"}}}},
		},
		{
			name:    "go named specifier",
			target:  LanguageGo,
			imports: []Import{{Path: "fmt", Specifiers: []ImportSpecifier{{Name: "Sprintf"}}}},
		},
		{
			name:    "go duplicate aliased local binding",
			target:  LanguageGo,
			imports: []Import{{Path: "fmt", Alias: "helper"}, {Path: "strings", Alias: "helper"}},
		},
		{
			name:    "java alias",
			target:  LanguageJava,
			imports: []Import{{Path: "com.example.Helper", Alias: "helper"}},
		},
		{
			name:    "java default",
			target:  LanguageJava,
			imports: []Import{{Path: "com.example.Helper", Default: "helper"}},
		},
		{
			name:    "java named specifier",
			target:  LanguageJava,
			imports: []Import{{Path: "com.example.Helper", Specifiers: []ImportSpecifier{{Name: "format"}}}},
		},
		{
			name:    "java wildcard specifier",
			target:  LanguageJava,
			imports: []Import{{Path: "com.example.helpers", Specifiers: []ImportSpecifier{{Name: "*"}}}},
		},
		{
			name:    "java wildcard path",
			target:  LanguageJava,
			imports: []Import{{Path: "com.example.helpers.*"}},
		},
		{
			name:    "java duplicate class binding",
			target:  LanguageJava,
			imports: []Import{{Path: "com.example.one.Helper"}, {Path: "com.example.two.Helper"}},
		},
		{
			name:    "rust wildcard path",
			target:  LanguageRust,
			imports: []Import{{Path: "crate::helpers::*"}},
		},
		{
			name:    "typescript duplicate local binding across imports",
			target:  LanguageTS,
			imports: []Import{{Path: "./one", Specifiers: []ImportSpecifier{{Name: "format"}}}, {Path: "./two", Specifiers: []ImportSpecifier{{Name: "track", Alias: "format"}}}},
		},
		{
			name:    "python duplicate local binding across imports",
			target:  LanguagePython,
			imports: []Import{{Path: "formatting", Specifiers: []ImportSpecifier{{Name: "format"}}}, {Path: "tracking", Specifiers: []ImportSpecifier{{Name: "track", Alias: "format"}}}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := AdapterRequest{TargetLanguage: tc.target, Globals: program.Globals, Behaviors: program.Behaviors}
			err := applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{{
				ID:      "entry_1",
				Body:    "translated",
				Imports: tc.imports,
			}}})
			if err == nil {
				t.Fatal("expected malformed import error")
			}
		})
	}
}

func TestCompileRejectsMalformedTargetImportsWithoutAdapter(t *testing.T) {
	source := `{
  "source_language": "typescript",
  "imports": [{
    "path": "./helpers",
    "language": "typescript",
    "alias": "helpers",
    "specifiers": [{ "name": "format" }]
  }],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "states": [{ "id": "state_1", "name": "closed", "kind": "state" }]
  }]
}`
	_, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err == nil || !strings.Contains(err.Error(), `combines namespace alias with default or named specifiers`) {
		t.Fatalf("error = %v, want malformed target import rejection", err)
	}
}

func TestCompileRejectsTargetImportCollisionsWithoutAdapter(t *testing.T) {
	source := `{
  "source_language": "typescript",
  "imports": [
    { "path": "./one", "language": "typescript", "specifiers": [{ "name": "format" }] },
    { "path": "./two", "language": "typescript", "specifiers": [{ "name": "track", "alias": "format" }] }
  ],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "states": [{ "id": "state_1", "name": "closed", "kind": "state" }]
  }]
}`
	_, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err == nil || !strings.Contains(err.Error(), `both bind local name "format"`) {
		t.Fatalf("error = %v, want target import collision rejection", err)
	}
}

func TestCompileRejectsTargetImportsThatCollideWithGeneratedSymbolsWithoutAdapter(t *testing.T) {
	source := `{
  "source_language": "typescript",
  "imports": [
    { "path": "./helpers", "language": "typescript", "specifiers": [{ "name": "entry1" }] }
  ],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "states": [{
      "id": "state_1",
      "name": "closed",
      "kind": "state",
      "behaviors": [{ "id": "entry_1", "kind": "entry" }]
    }]
  }],
  "behaviors": [{
    "id": "entry_1",
    "owner_id": "state_1",
    "kind": "entry",
    "source_language": "typescript",
    "body": "instance.ready = true;"
  }]
}`
	_, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err == nil || !strings.Contains(err.Error(), `bind local name "entry1", which collides with compiler-owned typescript symbol`) {
		t.Fatalf("error = %v, want generated symbol import collision rejection", err)
	}
}

func TestCompileRejectsTargetImportLocalNamesThatCollideWithGeneratedSymbolsWithoutAdapter(t *testing.T) {
	source := `{
  "source_language": "typescript",
  "imports": [
    { "path": "./helpers", "language": "typescript", "local_names": ["entry1"] }
  ],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "states": [{
      "id": "state_1",
      "name": "closed",
      "kind": "state",
      "behaviors": [{ "id": "entry_1", "kind": "entry" }]
    }]
  }],
  "behaviors": [{
    "id": "entry_1",
    "owner_id": "state_1",
    "kind": "entry",
    "source_language": "typescript",
    "body": "instance.ready = true;"
  }]
}`
	_, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err == nil || !strings.Contains(err.Error(), `bind local name "entry1", which collides with compiler-owned typescript symbol`) {
		t.Fatalf("error = %v, want generated symbol local_names import collision rejection", err)
	}
}

func TestCompileAllowsCSharpLegacyStaticLocalNameWithoutFalseGeneratedSymbolCollision(t *testing.T) {
	source := `{
  "source_language": "csharp",
  "imports": [
    { "path": "Sample.Formatting.Entry1", "language": "csharp", "local_names": ["*"] }
  ],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "states": [{
      "id": "state_1",
      "name": "closed",
      "kind": "state",
      "behaviors": [{ "id": "entry_1", "kind": "entry" }]
    }]
  }],
  "behaviors": [{
    "id": "entry_1",
    "owner_id": "state_1",
    "kind": "entry",
    "source_language": "csharp",
    "body": "Record(\"closed\");"
  }]
}`
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageCSharp,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, "using static Sample.Formatting.Entry1;") {
		t.Fatalf("C# output missing legacy static using:\n%s", text)
	}
	if strings.Contains(text, "using Sample.Formatting.Entry1;") {
		t.Fatalf("C# output rendered legacy static using as ordinary using:\n%s", text)
	}
}

func TestCompileRejectsCSharpLegacyStaticLocalNameWithAlias(t *testing.T) {
	source := `{
  "source_language": "csharp",
  "imports": [
    { "path": "Sample.Formatting.Helper", "language": "csharp", "alias": "Format", "local_names": ["*"] }
  ],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "states": [{ "id": "state_1", "name": "closed", "kind": "state" }]
  }]
}`
	_, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageCSharp,
		Adapter: "none",
	})
	if err == nil || !strings.Contains(err.Error(), `import "Sample.Formatting.Helper" combines static using with alias`) {
		t.Fatalf("error = %v, want C# legacy static alias rejection", err)
	}
}

func TestCompileRejectsMalformedDartTargetImportsWithoutAdapter(t *testing.T) {
	source := `{
  "source_language": "dart",
  "imports": [{
    "path": "helpers.dart",
    "language": "dart",
    "specifiers": [{ "name": "record", "alias": "rec" }]
  }],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "states": [{ "id": "state_1", "name": "closed", "kind": "state" }]
  }]
}`
	_, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageDart,
		Adapter: "none",
	})
	if err == nil || !strings.Contains(err.Error(), `uses specifier alias "rec" unsupported by Dart`) {
		t.Fatalf("error = %v, want Dart specifier alias rejection", err)
	}
}

func TestCompilePreservesDartTargetAliasWithHideImportsWithoutAdapter(t *testing.T) {
	source := `{
  "source_language": "dart",
  "imports": [{
    "path": "helpers.dart",
    "language": "dart",
    "alias": "helpers",
    "hidden": ["debugRecord"]
  }],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "states": [{ "id": "state_1", "name": "closed", "kind": "state" }]
  }]
}`
	output, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageDart,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), `import "helpers.dart" as helpers hide debugRecord;`) {
		t.Fatalf("Dart output missing prefixed hide import:\n%s", output)
	}
}

func TestCompileRejectsForeignDartHiddenTargetImportsWithoutAdapter(t *testing.T) {
	source := `{
  "source_language": "typescript",
  "imports": [{
    "path": "helpers.dart",
    "language": "dart",
    "hidden": ["debugRecord"]
  }],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "states": [{ "id": "state_1", "name": "closed", "kind": "state" }]
  }]
}`
	_, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageDart,
		Adapter: "none",
	})
	if err == nil || !strings.Contains(err.Error(), `uses Dart hidden specifiers unsupported for translated target imports`) {
		t.Fatalf("error = %v, want Dart hidden target import rejection", err)
	}
}

func TestCompileRejectsForeignDartAliasWithShowTargetImportsWithoutAdapter(t *testing.T) {
	source := `{
  "source_language": "typescript",
  "imports": [{
    "path": "helpers.dart",
    "language": "dart",
    "alias": "helpers",
    "specifiers": [{ "name": "record" }]
  }],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "states": [{ "id": "state_1", "name": "closed", "kind": "state" }]
  }]
}`
	_, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageDart,
		Adapter: "none",
	})
	if err == nil || !strings.Contains(err.Error(), `combines Dart prefix alias with show specifiers unsupported for translated target imports`) {
		t.Fatalf("error = %v, want Dart prefix/show target import rejection", err)
	}
}

func TestCompileRejectsForeignJavaWildcardTargetImportsWithoutAdapter(t *testing.T) {
	cases := []struct {
		name       string
		importJSON string
	}{
		{
			name: "specifier",
			importJSON: `{
    "path": "com.example.helpers",
    "language": "java",
    "specifiers": [{ "name": "*" }]
  }`,
		},
		{
			name: "path",
			importJSON: `{
    "path": "com.example.helpers.*",
    "language": "java"
  }`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := `{
  "source_language": "typescript",
  "imports": [` + tc.importJSON + `],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "states": [{ "id": "state_1", "name": "closed", "kind": "state" }]
  }]
}`
			_, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
				From:    LanguageJSONIR,
				To:      LanguageJava,
				Adapter: "none",
			})
			if err == nil || !strings.Contains(err.Error(), `uses Java wildcard import unsupported for translated target imports`) {
				t.Fatalf("error = %v, want Java wildcard target import rejection", err)
			}
		})
	}
}

func TestCompileRejectsForeignRustWildcardTargetImportsWithoutAdapter(t *testing.T) {
	cases := []struct {
		name       string
		importJSON string
	}{
		{
			name: "specifier",
			importJSON: `{
    "path": "crate::helpers",
    "language": "rust",
    "specifiers": [{ "name": "*" }]
  }`,
		},
		{
			name: "path",
			importJSON: `{
    "path": "crate::helpers::*",
    "language": "rust"
  }`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := `{
  "source_language": "typescript",
  "imports": [` + tc.importJSON + `],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "states": [{ "id": "state_1", "name": "closed", "kind": "state" }]
  }]
}`
			_, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
				From:    LanguageJSONIR,
				To:      LanguageRust,
				Adapter: "none",
			})
			if err == nil || !strings.Contains(err.Error(), `uses Rust wildcard import unsupported for translated target imports`) {
				t.Fatalf("error = %v, want Rust wildcard target import rejection", err)
			}
		})
	}
}

func TestCompileRejectsForeignPythonWildcardTargetImportsWithoutAdapter(t *testing.T) {
	source := `{
  "source_language": "typescript",
  "imports": [{
    "path": "helpers",
    "language": "python",
    "specifiers": [{ "name": "*" }]
  }],
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "states": [{ "id": "state_1", "name": "closed", "kind": "state" }]
  }]
}`
	_, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguagePython,
		Adapter: "none",
	})
	if err == nil || !strings.Contains(err.Error(), `uses Python wildcard import unsupported for translated target imports`) {
		t.Fatalf("error = %v, want Python wildcard target import rejection", err)
	}
}

func TestCompileAllowsNativeWildcardTargetImportsWithoutFalseLocalBindingCollisions(t *testing.T) {
	cases := []struct {
		name     string
		language Language
		imports  string
	}{
		{
			name:     "java wildcard specifiers",
			language: LanguageJava,
			imports: `[
    { "path": "com.example.helpers", "language": "java", "specifiers": [{ "name": "*" }] },
    { "path": "com.other.helpers", "language": "java", "specifiers": [{ "name": "*" }] }
  ]`,
		},
		{
			name:     "java wildcard paths",
			language: LanguageJava,
			imports: `[
    { "path": "com.example.helpers.*", "language": "java" },
    { "path": "com.other.helpers.*", "language": "java" }
  ]`,
		},
		{
			name:     "rust wildcard specifiers",
			language: LanguageRust,
			imports: `[
    { "path": "crate::example::helpers", "language": "rust", "specifiers": [{ "name": "*" }] },
    { "path": "crate::other::helpers", "language": "rust", "specifiers": [{ "name": "*" }] }
  ]`,
		},
		{
			name:     "rust wildcard paths",
			language: LanguageRust,
			imports: `[
    { "path": "crate::example::helpers::*", "language": "rust" },
    { "path": "crate::other::helpers::*", "language": "rust" }
  ]`,
		},
		{
			name:     "python wildcard specifiers",
			language: LanguagePython,
			imports: `[
    { "path": "example_helpers", "language": "python", "specifiers": [{ "name": "*" }] },
    { "path": "other_helpers", "language": "python", "specifiers": [{ "name": "*" }] }
  ]`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := `{
  "source_language": ` + strconv.Quote(string(tc.language)) + `,
  "imports": ` + tc.imports + `,
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "states": [{ "id": "state_1", "name": "closed", "kind": "state" }]
  }]
}`
			_, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
				From:    LanguageJSONIR,
				To:      tc.language,
				Adapter: "none",
			})
			if err != nil {
				t.Fatalf("native wildcard imports should not create local binding collisions: %v", err)
			}
		})
	}
}

func TestCompileRejectsCompilerOwnedRuntimeImportPathsWithoutAdapter(t *testing.T) {
	source := `{
  "source_language": "go",
  "models": [{
    "id": "model_1",
    "name": "Door",
    "initial": { "owner_id": "model_1", "id": "initial_1", "target": "closed" },
    "states": [{
      "id": "state_1",
      "name": "closed",
      "kind": "state",
      "behaviors": [{ "id": "entry_1", "kind": "entry" }]
    }]
  }],
  "behaviors": [{
    "id": "entry_1",
    "owner_id": "state_1",
    "kind": "entry",
    "source_language": "go",
    "target_language": "typescript",
    "body": "runtime.track('closed');",
    "imports": [{ "path": "@stateforward/hsm.ts", "alias": "runtime", "language": "typescript" }]
  }]
}`
	_, _, err := NewCompiler().Compile(context.Background(), SourceInput{Path: "door.hsm.json", Data: []byte(source)}, CompileOptions{
		From:    LanguageJSONIR,
		To:      LanguageTS,
		Adapter: "none",
	})
	if err == nil || !strings.Contains(err.Error(), `target behavior "entry_1" import "@stateforward/hsm.ts" is compiler-owned`) {
		t.Fatalf("error = %v, want compiler-owned target import path rejection", err)
	}
}

func TestApplyBehaviorPatchRejectsTargetImportCollisionsAcrossRegions(t *testing.T) {
	program := &Program{
		Globals:   []CodeBlock{{ID: "global_1", Language: LanguageGo, Code: "func helper() {}"}},
		Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Globals: program.Globals, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{
		Globals: []CodeBlock{{
			ID:      "global_1",
			Code:    "const helper = () => format('global');",
			Imports: []Import{{Path: "./global-format", Specifiers: []ImportSpecifier{{Name: "format"}}}},
		}},
		Behaviors: []Behavior{{
			ID:   "entry_1",
			Body: "format('behavior');",
			Imports: []Import{{
				Path:       "./behavior-format",
				Specifiers: []ImportSpecifier{{Name: "track", Alias: "format"}},
			}},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), `both bind local name "format"`) {
		t.Fatalf("error = %v, want cross-region import binding collision", err)
	}
}

func TestApplyBehaviorPatchRejectsTargetImportsThatCollideWithGeneratedSymbols(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{{
		ID:      "entry_1",
		Body:    "entry1('closed');",
		Imports: []Import{{Path: "./helpers", Specifiers: []ImportSpecifier{{Name: "entry1"}}}},
	}}})
	if err == nil || !strings.Contains(err.Error(), `bind local name "entry1", which collides with compiler-owned typescript symbol`) {
		t.Fatalf("error = %v, want generated symbol import collision rejection", err)
	}
}

func TestApplyBehaviorPatchRejectsCompilerOwnedRuntimeImportCollisions(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{{
		ID:      "entry_1",
		Body:    "hsm.track('closed');",
		Imports: []Import{{Path: "./not-runtime", Alias: "hsm"}},
	}}})
	if err == nil || !strings.Contains(err.Error(), `both bind local name "hsm"`) {
		t.Fatalf("error = %v, want runtime import binding collision", err)
	}
}

func TestApplyBehaviorPatchRejectsJavaRuntimeTypeImportCollisions(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		local   string
		body    string
		kind    BehaviorKind
		trigger TriggerKind
	}{
		{
			name:  "context",
			path:  "com.example.Context",
			local: "Context",
			body:  "Context.track();",
			kind:  BehaviorEntry,
		},
		{
			name:    "object fallback trigger",
			path:    "com.example.Object",
			local:   "Object",
			body:    "return new Object();",
			kind:    BehaviorTrigger,
			trigger: TriggerKind("custom"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := &Program{
				Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: tc.kind, TriggerKind: tc.trigger, SourceLanguage: LanguageGo}},
			}
			AssignTargetABI(program, LanguageJava)
			request := AdapterRequest{TargetLanguage: LanguageJava, Behaviors: program.Behaviors}

			err := applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{{
				ID:      "entry_1",
				Body:    tc.body,
				Imports: []Import{{Path: tc.path}},
			}}})
			if err == nil || !strings.Contains(err.Error(), `both bind local name "`+tc.local+`"`) {
				t.Fatalf("error = %v, want Java runtime %s import binding collision", err, tc.local)
			}
		})
	}
}

func TestApplyBehaviorPatchRejectsRuntimeTypeImportCollisions(t *testing.T) {
	cases := []struct {
		name    string
		target  Language
		imports []Import
		local   string
	}{
		{
			name:    "csharp context alias",
			target:  LanguageCSharp,
			imports: []Import{{Path: "Example.Runtime.Context", Alias: "Context"}},
			local:   "Context",
		},
		{
			name:    "csharp operation alias",
			target:  LanguageCSharp,
			imports: []Import{{Path: "Example.Runtime.Operation", Alias: "Operation"}},
			local:   "Operation",
		},
		{
			name:    "csharp expression alias",
			target:  LanguageCSharp,
			imports: []Import{{Path: "Example.Runtime.Expression", Alias: "Expression"}},
			local:   "Expression",
		},
		{
			name:    "csharp duration provider alias",
			target:  LanguageCSharp,
			imports: []Import{{Path: "Example.Runtime.DurationProvider", Alias: "DurationProvider"}},
			local:   "DurationProvider",
		},
		{
			name:    "csharp time provider alias",
			target:  LanguageCSharp,
			imports: []Import{{Path: "Example.Runtime.TimeProvider", Alias: "TimeProvider"}},
			local:   "TimeProvider",
		},
		{
			name:    "csharp condition channel alias",
			target:  LanguageCSharp,
			imports: []Import{{Path: "Example.Runtime.ConditionChannel", Alias: "ConditionChannel"}},
			local:   "ConditionChannel",
		},
		{
			name:   "dart context show",
			target: LanguageDart,
			imports: []Import{{
				Path:       "package:example/runtime.dart",
				Specifiers: []ImportSpecifier{{Name: "Context"}},
			}},
			local: "Context",
		},
		{
			name:   "dart behavior show",
			target: LanguageDart,
			imports: []Import{{
				Path:       "package:example/runtime.dart",
				Specifiers: []ImportSpecifier{{Name: "Behavior"}},
			}},
			local: "Behavior",
		},
		{
			name:   "dart hsm show",
			target: LanguageDart,
			imports: []Import{{
				Path:       "package:example/runtime.dart",
				Specifiers: []ImportSpecifier{{Name: "Hsm"}},
			}},
			local: "Hsm",
		},
		{
			name:   "dart kind show",
			target: LanguageDart,
			imports: []Import{{
				Path:       "package:example/model.dart",
				Specifiers: []ImportSpecifier{{Name: "Kind"}},
			}},
			local: "Kind",
		},
		{
			name:   "dart vertex show",
			target: LanguageDart,
			imports: []Import{{
				Path:       "package:example/model.dart",
				Specifiers: []ImportSpecifier{{Name: "Vertex"}},
			}},
			local: "Vertex",
		},
		{
			name:   "dart define show",
			target: LanguageDart,
			imports: []Import{{
				Path:       "package:example/dsl.dart",
				Specifiers: []ImportSpecifier{{Name: "define"}},
			}},
			local: "define",
		},
		{
			name:   "dart finalState show",
			target: LanguageDart,
			imports: []Import{{
				Path:       "package:example/dsl.dart",
				Specifiers: []ImportSpecifier{{Name: "finalState"}},
			}},
			local: "finalState",
		},
		{
			name:   "dart at show",
			target: LanguageDart,
			imports: []Import{{
				Path:       "package:example/timers.dart",
				Specifiers: []ImportSpecifier{{Name: "at"}},
			}},
			local: "at",
		},
		{
			name:   "dart every show",
			target: LanguageDart,
			imports: []Import{{
				Path:       "package:example/timers.dart",
				Specifiers: []ImportSpecifier{{Name: "every"}},
			}},
			local: "every",
		},
		{
			name:   "dart onCall show",
			target: LanguageDart,
			imports: []Import{{
				Path:       "package:example/operations.dart",
				Specifiers: []ImportSpecifier{{Name: "onCall"}},
			}},
			local: "onCall",
		},
		{
			name:   "dart when show",
			target: LanguageDart,
			imports: []Import{{
				Path:       "package:example/predicates.dart",
				Specifiers: []ImportSpecifier{{Name: "when"}},
			}},
			local: "when",
		},
		{
			name:   "dart target alias",
			target: LanguageDart,
			imports: []Import{{
				Path:  "package:example/dsl.dart",
				Alias: "target",
			}},
			local: "target",
		},
		{
			name:   "dart timer show",
			target: LanguageDart,
			imports: []Import{{
				Path:       "package:example/scheduler.dart",
				Specifiers: []ImportSpecifier{{Name: "Timer"}},
			}},
			local: "Timer",
		},
		{
			name:   "dart duration alias",
			target: LanguageDart,
			imports: []Import{{
				Path:  "package:example/time.dart",
				Alias: "Duration",
			}},
			local: "Duration",
		},
		{
			name:   "dart datetime show",
			target: LanguageDart,
			imports: []Import{{
				Path:       "package:example/time.dart",
				Specifiers: []ImportSpecifier{{Name: "DateTime"}},
			}},
			local: "DateTime",
		},
		{
			name:   "dart bool show",
			target: LanguageDart,
			imports: []Import{{
				Path:       "package:example/types.dart",
				Specifiers: []ImportSpecifier{{Name: "bool"}},
			}},
			local: "bool",
		},
		{
			name:   "dart int alias",
			target: LanguageDart,
			imports: []Import{{
				Path:  "package:example/types.dart",
				Alias: "int",
			}},
			local: "int",
		},
		{
			name:   "dart future show",
			target: LanguageDart,
			imports: []Import{{
				Path:       "package:example/async.dart",
				Specifiers: []ImportSpecifier{{Name: "Future"}},
			}},
			local: "Future",
		},
		{
			name:   "javascript Date specifier",
			target: LanguageJS,
			imports: []Import{{
				Path:       "./time.js",
				Specifiers: []ImportSpecifier{{Name: "Date"}},
			}},
			local: "Date",
		},
		{
			name:   "javascript undefined alias",
			target: LanguageJS,
			imports: []Import{{
				Path:       "./sentinel.js",
				Specifiers: []ImportSpecifier{{Name: "missing", Alias: "undefined"}},
			}},
			local: "undefined",
		},
		{
			name:   "typescript Date specifier",
			target: LanguageTS,
			imports: []Import{{
				Path:       "./time",
				Specifiers: []ImportSpecifier{{Name: "Date"}},
			}},
			local: "Date",
		},
		{
			name:   "typescript InstanceType specifier",
			target: LanguageTS,
			imports: []Import{{
				Path:       "./types",
				Specifiers: []ImportSpecifier{{Name: "InstanceType"}},
			}},
			local: "InstanceType",
		},
		{
			name:   "typescript Promise alias",
			target: LanguageTS,
			imports: []Import{{
				Path:       "./async",
				Specifiers: []ImportSpecifier{{Name: "AsyncResult", Alias: "Promise"}},
			}},
			local: "Promise",
		},
		{
			name:   "typescript undefined alias",
			target: LanguageTS,
			imports: []Import{{
				Path:       "./sentinel",
				Specifiers: []ImportSpecifier{{Name: "missing", Alias: "undefined"}},
			}},
			local: "undefined",
		},
		{
			name:    "go bool alias",
			target:  LanguageGo,
			imports: []Import{{Path: "example.com/runtime/bool", Alias: "bool"}},
			local:   "bool",
		},
		{
			name:    "go nil alias",
			target:  LanguageGo,
			imports: []Import{{Path: "example.com/runtime/nil", Alias: "nil"}},
			local:   "nil",
		},
		{
			name:    "go false alias",
			target:  LanguageGo,
			imports: []Import{{Path: "example.com/runtime/false", Alias: "false"}},
			local:   "false",
		},
		{
			name:    "go true alias",
			target:  LanguageGo,
			imports: []Import{{Path: "example.com/runtime/true", Alias: "true"}},
			local:   "true",
		},
		{
			name:   "python bool specifier",
			target: LanguagePython,
			imports: []Import{{
				Path:       "helpers",
				Specifiers: []ImportSpecifier{{Name: "bool"}},
			}},
			local: "bool",
		},
		{
			name:   "python float alias",
			target: LanguagePython,
			imports: []Import{{
				Path:  "helpers.FloatLike",
				Alias: "float",
			}},
			local: "float",
		},
		{
			name:   "python none alias",
			target: LanguagePython,
			imports: []Import{{
				Path:  "helpers.none_like",
				Alias: "None",
			}},
			local: "None",
		},
		{
			name:   "python false specifier",
			target: LanguagePython,
			imports: []Import{{
				Path:       "helpers",
				Specifiers: []ImportSpecifier{{Name: "False"}},
			}},
			local: "False",
		},
		{
			name:   "python true specifier",
			target: LanguagePython,
			imports: []Import{{
				Path:       "helpers",
				Specifiers: []ImportSpecifier{{Name: "True"}},
			}},
			local: "True",
		},
		{
			name:   "rust duration specifier",
			target: LanguageRust,
			imports: []Import{{
				Path:       "crate::clock",
				Specifiers: []ImportSpecifier{{Name: "Duration"}},
			}},
			local: "Duration",
		},
		{
			name:   "rust box specifier",
			target: LanguageRust,
			imports: []Import{{
				Path:       "crate::alloc",
				Specifiers: []ImportSpecifier{{Name: "Box"}},
			}},
			local: "Box",
		},
		{
			name:   "rust future alias",
			target: LanguageRust,
			imports: []Import{{
				Path:  "crate::async_runtime::Future",
				Alias: "Future",
			}},
			local: "Future",
		},
		{
			name:   "rust send alias",
			target: LanguageRust,
			imports: []Import{{
				Path:  "crate::threads::ThreadSafe",
				Alias: "Send",
			}},
			local: "Send",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := &Program{
				Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo}},
			}
			request := AdapterRequest{TargetLanguage: tc.target, Behaviors: program.Behaviors}

			err := applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{{
				ID:      "entry_1",
				Body:    "translated();",
				Imports: tc.imports,
			}}})
			if err == nil || !strings.Contains(err.Error(), `both bind local name "`+tc.local+`"`) {
				t.Fatalf("error = %v, want runtime %s import binding collision", err, tc.local)
			}
		})
	}
}

func TestApplyBehaviorPatchRejectsCompilerOwnedRuntimeImportPaths(t *testing.T) {
	program := &Program{
		Globals:   []CodeBlock{{ID: "global_1", Language: LanguageGo, Code: "func helper() {}"}},
		Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo}},
	}
	cases := []struct {
		name   string
		target Language
		patch  BehaviorPatch
		want   string
	}{
		{
			name:   "program typescript runtime",
			target: LanguageTS,
			patch:  BehaviorPatch{Imports: []Import{{Path: "@stateforward/hsm.ts", Alias: "runtime"}}},
			want:   `program import "@stateforward/hsm.ts" is compiler-owned`,
		},
		{
			name:   "program typescript package runtime",
			target: LanguageTS,
			patch:  BehaviorPatch{Imports: []Import{{Path: "@stateforward/hsm", Alias: "runtime"}}},
			want:   `program import "@stateforward/hsm" is compiler-owned`,
		},
		{
			name:   "global go context",
			target: LanguageGo,
			patch: BehaviorPatch{Globals: []CodeBlock{{
				ID:      "global_1",
				Code:    "func helper() {}",
				Imports: []Import{{Path: "context", Alias: "ctxpkg"}},
			}}},
			want: `global "global_1" import "context" is compiler-owned`,
		},
		{
			name:   "behavior csharp static hsm facade runtime",
			target: LanguageCSharp,
			patch: BehaviorPatch{Behaviors: []Behavior{{
				ID:      "entry_1",
				Body:    "Runtime.Track();",
				Imports: []Import{{Path: "Stateforward.Hsm.Hsm"}},
			}}},
			want: `behavior "entry_1" import "Stateforward.Hsm.Hsm" is compiler-owned`,
		},
		{
			name:   "global csharp hsm namespace runtime",
			target: LanguageCSharp,
			patch: BehaviorPatch{Globals: []CodeBlock{{
				ID:      "adapter_global_helper",
				Code:    "static class Helper {}",
				Imports: []Import{{Path: "Stateforward.Hsm"}},
			}}},
			want: `global "adapter_global_helper" import "Stateforward.Hsm" is compiler-owned`,
		},
		{
			name:   "behavior csharp context runtime",
			target: LanguageCSharp,
			patch: BehaviorPatch{Behaviors: []Behavior{{
				ID:      "entry_1",
				Body:    "Context.KeepAlive(instance);",
				Imports: []Import{{Path: "Stateforward.Hsm.Context"}},
			}}},
			want: `behavior "entry_1" import "Stateforward.Hsm.Context" is compiler-owned`,
		},
		{
			name:   "program cpp hsm include runtime",
			target: LanguageCPP,
			patch:  BehaviorPatch{Imports: []Import{{Path: "hsm/hsm.hpp"}}},
			want:   `program import "hsm/hsm.hpp" is compiler-owned`,
		},
		{
			name:   "program cpp quoted hsm include runtime",
			target: LanguageCPP,
			patch:  BehaviorPatch{Imports: []Import{{Path: `"hsm/hsm.hpp"`}}},
			want:   `program import "\"hsm/hsm.hpp\"" is compiler-owned`,
		},
		{
			name:   "behavior cpp chrono runtime",
			target: LanguageCPP,
			patch: BehaviorPatch{Behaviors: []Behavior{{
				ID:      "entry_1",
				Body:    "auto now = std::chrono::steady_clock::now();",
				Imports: []Import{{Path: "chrono"}},
			}}},
			want: `behavior "entry_1" import "chrono" is compiler-owned`,
		},
		{
			name:   "behavior cpp angled chrono runtime",
			target: LanguageCPP,
			patch: BehaviorPatch{Behaviors: []Behavior{{
				ID:      "entry_1",
				Body:    "auto now = std::chrono::steady_clock::now();",
				Imports: []Import{{Path: "<chrono>"}},
			}}},
			want: `behavior "entry_1" import "<chrono>" is compiler-owned`,
		},
		{
			name:   "behavior cpp cstdint scaffold",
			target: LanguageCPP,
			patch: BehaviorPatch{Behaviors: []Behavior{{
				ID:      "entry_1",
				Body:    "std::uint64_t value = 0;",
				Imports: []Import{{Path: "<cstdint>"}},
			}}},
			want: `behavior "entry_1" import "<cstdint>" is compiler-owned`,
		},
		{
			name:   "global cpp string scaffold",
			target: LanguageCPP,
			patch: BehaviorPatch{Globals: []CodeBlock{{
				ID:      "adapter_global_helper",
				Code:    `static std::string helper() { return "closed"; }`,
				Imports: []Import{{Path: "string"}},
			}}},
			want: `global "adapter_global_helper" import "string" is compiler-owned`,
		},
		{
			name:   "behavior python datetime",
			target: LanguagePython,
			patch: BehaviorPatch{Behaviors: []Behavior{{
				ID:      "entry_1",
				Body:    "now = dt.datetime.now()",
				Imports: []Import{{Path: "datetime", Alias: "dt"}},
			}}},
			want: `behavior "entry_1" import "datetime" is compiler-owned`,
		},
		{
			name:   "behavior dart hsm runtime",
			target: LanguageDart,
			patch: BehaviorPatch{Behaviors: []Behavior{{
				ID:      "entry_1",
				Body:    "hsm.record(instance);",
				Imports: []Import{{Path: "package:hsm/hsm.dart", Alias: "hsm"}},
			}}},
			want: `behavior "entry_1" import "package:hsm/hsm.dart" is compiler-owned`,
		},
		{
			name:   "behavior dart core runtime",
			target: LanguageDart,
			patch: BehaviorPatch{Behaviors: []Behavior{{
				ID:      "entry_1",
				Body:    "final value = Future.value();",
				Imports: []Import{{Path: "dart:core", Specifiers: []ImportSpecifier{{Name: "Future"}}}},
			}}},
			want: `behavior "entry_1" import "dart:core" is compiler-owned`,
		},
		{
			name:   "behavior dart async",
			target: LanguageDart,
			patch: BehaviorPatch{Behaviors: []Behavior{{
				ID:      "entry_1",
				Body:    "Timer.run(() {});",
				Imports: []Import{{Path: "dart:async", Alias: "runtimeAsync"}},
			}}},
			want: `behavior "entry_1" import "dart:async" is compiler-owned`,
		},
		{
			name:   "adapter helper javascript runtime",
			target: LanguageJS,
			patch: BehaviorPatch{Globals: []CodeBlock{{
				ID:      "adapter_global_helper",
				Code:    "const helper = true;",
				Imports: []Import{{Path: "@stateforward/hsm", Alias: "runtime"}},
			}}},
			want: `global "adapter_global_helper" import "@stateforward/hsm" is compiler-owned`,
		},
		{
			name:   "behavior java runtime",
			target: LanguageJava,
			patch: BehaviorPatch{Behaviors: []Behavior{{
				ID:      "entry_1",
				Body:    "Runtime.track();",
				Imports: []Import{{Path: "com.stateforward.hsm"}},
			}}},
			want: `behavior "entry_1" import "com.stateforward.hsm" is compiler-owned`,
		},
		{
			name:   "behavior java event runtime",
			target: LanguageJava,
			patch: BehaviorPatch{Behaviors: []Behavior{{
				ID:      "entry_1",
				Body:    "Event event = null;",
				Imports: []Import{{Path: "com.stateforward.hsm.Event"}},
			}}},
			want: `behavior "entry_1" import "com.stateforward.hsm.Event" is compiler-owned`,
		},
		{
			name:   "behavior java wildcard runtime",
			target: LanguageJava,
			patch: BehaviorPatch{Behaviors: []Behavior{{
				ID:      "entry_1",
				Body:    "Runtime.track();",
				Imports: []Import{{Path: "com.stateforward.hsm.*"}},
			}}},
			want: `behavior "entry_1" import "com.stateforward.hsm.*" is compiler-owned`,
		},
		{
			name:   "behavior java static hsm facade runtime",
			target: LanguageJava,
			patch: BehaviorPatch{Behaviors: []Behavior{{
				ID:      "entry_1",
				Body:    "Runtime.track();",
				Imports: []Import{{Path: "com.stateforward.hsm.Hsm"}},
			}}},
			want: `behavior "entry_1" import "com.stateforward.hsm.Hsm" is compiler-owned`,
		},
		{
			name:   "behavior rust future runtime",
			target: LanguageRust,
			patch: BehaviorPatch{Behaviors: []Behavior{{
				ID:      "entry_1",
				Body:    "Box::pin(async {})",
				Imports: []Import{{Path: "std::future"}},
			}}},
			want: `behavior "entry_1" import "std::future" is compiler-owned`,
		},
		{
			name:   "behavior rust wildcard hsm runtime",
			target: LanguageRust,
			patch: BehaviorPatch{Behaviors: []Behavior{{
				ID:      "entry_1",
				Body:    "hsm::dispatch(instance);",
				Imports: []Import{{Path: "hsm::*"}},
			}}},
			want: `behavior "entry_1" import "hsm::*" is compiler-owned`,
		},
		{
			name:   "behavior rust concrete pin runtime",
			target: LanguageRust,
			patch: BehaviorPatch{Behaviors: []Behavior{{
				ID:      "entry_1",
				Body:    "Pin::new(value)",
				Imports: []Import{{Path: "std::pin::Pin"}},
			}}},
			want: `behavior "entry_1" import "std::pin::Pin" is compiler-owned`,
		},
		{
			name:   "behavior zig hsm runtime",
			target: LanguageZig,
			patch: BehaviorPatch{Behaviors: []Behavior{{
				ID:      "entry_1",
				Body:    "hsm.dispatch(instance);",
				Imports: []Import{{Path: "hsm", Alias: "runtime"}},
			}}},
			want: `behavior "entry_1" import "hsm" is compiler-owned`,
		},
		{
			name:   "global zig std runtime",
			target: LanguageZig,
			patch: BehaviorPatch{Globals: []CodeBlock{{
				ID:      "adapter_global_helper",
				Code:    "const now = std.time.timestamp();",
				Imports: []Import{{Path: "std", Alias: "runtime_std"}},
			}}},
			want: `global "adapter_global_helper" import "std" is compiler-owned`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := AdapterRequest{TargetLanguage: tc.target, Globals: program.Globals, Behaviors: program.Behaviors}
			err := applyBehaviorPatch(program, request, &tc.patch)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestApplyBehaviorPatchAllowsSameTargetImportAcrossRegions(t *testing.T) {
	program := &Program{
		Globals:   []CodeBlock{{ID: "global_1", Language: LanguageGo, Code: "func helper() {}"}},
		Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Globals: program.Globals, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{
		Globals: []CodeBlock{{
			ID:      "global_1",
			Code:    "const helper = () => format('global');",
			Imports: []Import{{Path: "./format", Specifiers: []ImportSpecifier{{Name: "format"}}}},
		}},
		Behaviors: []Behavior{{
			ID:      "entry_1",
			Body:    "format('behavior');",
			Imports: []Import{{Path: "./format", Specifiers: []ImportSpecifier{{Name: "format"}}}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestApplyBehaviorPatchDefaultsNestedImportLanguagesToTarget(t *testing.T) {
	program := &Program{
		Globals:   []CodeBlock{{ID: "global_1", Language: LanguageGo, Code: "func helper() {}"}},
		Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageGo}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Globals: program.Globals, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Code:     "const helper = () => {};",
			Imports:  []Import{{Path: "./global-helper"}},
			Language: "",
		}},
		Behaviors: []Behavior{{
			ID:      "entry_1",
			Body:    "translated();",
			Imports: []Import{{Path: "./behavior-helper"}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if program.Globals[0].Language != LanguageTS || program.Globals[0].Imports[0].Language != LanguageTS {
		t.Fatalf("global patch = %#v", program.Globals[0])
	}
	if program.Behaviors[0].TargetLanguage != LanguageTS || program.Behaviors[0].Imports[0].Language != LanguageTS {
		t.Fatalf("behavior patch = %#v", program.Behaviors[0])
	}
}

func TestApplyBehaviorPatchRejectsAdapterOnlyMetadataFromPatchImports(t *testing.T) {
	sourceRange := SourceRange{File: "adapter.ts", StartByte: 1, EndByte: 2, StartLine: 3, EndLine: 4}
	base := &Program{
		Globals:   []CodeBlock{{ID: "global_1", Language: LanguageTS, Code: "const helper = () => true;"}},
		Behaviors: []Behavior{{ID: "entry_1", OwnerID: "state_1", Kind: BehaviorEntry, SourceLanguage: LanguageTS, TargetLanguage: LanguageTS, Body: "instance.ready = true;"}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Globals: base.Globals, Behaviors: base.Behaviors}
	cases := []BehaviorPatch{
		{Imports: []Import{{Path: "./program", LocalNames: []string{"program"}, Range: sourceRange}}},
		{Globals: []CodeBlock{{ID: "global_1", Imports: []Import{{Path: "./global", LocalNames: []string{"global"}}}}}},
		{Globals: []CodeBlock{{ID: "adapter_global_helper", Code: "const adapterHelper = true;", Imports: []Import{{Path: "./helper", Range: sourceRange}}}}},
		{Behaviors: []Behavior{{ID: "entry_1", Imports: []Import{{Path: "./behavior", LocalNames: []string{"behavior"}, Range: sourceRange}}}}},
	}
	for _, patch := range cases {
		program := cloneProgram(base)
		err := applyBehaviorPatch(program, request, &patch)
		if err == nil || (!strings.Contains(err.Error(), "attempts to set local_names") && !strings.Contains(err.Error(), "attempts to change range")) {
			t.Fatalf("error = %v, want adapter-only import metadata rejection for %#v", err, patch)
		}
	}
}

func TestApplyBehaviorPatchRejectsGlobalRangeMetadata(t *testing.T) {
	sourceRange := SourceRange{StartByte: 1}
	program := &Program{
		Globals: []CodeBlock{{ID: "global_1", Language: LanguageTS, Code: "const helper = () => true;"}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Globals: program.Globals}
	cases := []BehaviorPatch{
		{Globals: []CodeBlock{{ID: "global_1", Range: sourceRange}}},
		{Globals: []CodeBlock{{ID: "adapter_global_helper", Code: "const helper = true;", Range: sourceRange}}},
	}
	for _, patch := range cases {
		err := applyBehaviorPatch(program, request, &patch)
		if err == nil || !strings.Contains(err.Error(), "attempts to change range") {
			t.Fatalf("error = %v, want global range rejection", err)
		}
	}
}

func TestApplyBehaviorPatchRejectsGlobalLanguageMetadata(t *testing.T) {
	program := &Program{
		Globals: []CodeBlock{{ID: "global_1", Language: LanguageTS, Code: "const helper = () => true;"}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Globals: program.Globals}
	cases := []BehaviorPatch{
		{Globals: []CodeBlock{{ID: "global_1", Language: LanguageTS, Code: "const helper = () => false;"}}},
		{Globals: []CodeBlock{{ID: "adapter_global_helper", Language: LanguageTS, Code: "const helper = true;"}}},
	}
	for _, patch := range cases {
		err := applyBehaviorPatch(program, request, &patch)
		if err == nil || !strings.Contains(err.Error(), "attempts to change language") {
			t.Fatalf("error = %v, want global language rejection", err)
		}
	}
}

func TestApplyBehaviorPatchRejectsForeignImportOnlyBehaviorPatch(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			Body:           "fmt.Println(\"closed\")",
			Imports:        []Import{{Path: "fmt", Language: LanguageGo}},
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Behaviors: program.Behaviors}

	for _, imports := range [][]Import{
		{{Path: "./format"}},
		{},
	} {
		err := applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{{
			ID:      "entry_1",
			Imports: imports,
		}}})
		if err == nil || !strings.Contains(err.Error(), `behavior "entry_1" imports without target body or code`) {
			t.Fatalf("error = %v, want foreign import-only behavior rejection for imports %#v", err, imports)
		}
		if len(program.Behaviors[0].Imports) != 1 || program.Behaviors[0].Imports[0].Path != "fmt" {
			t.Fatalf("foreign import-only behavior rejection mutated imports: %#v", program.Behaviors[0].Imports)
		}
	}
}

func TestApplyBehaviorPatchAllowsAlreadyTargetImportOnlyBehaviorPatch(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageTS,
			TargetLanguage: LanguageTS,
			Body:           "record(\"closed\");",
			Imports:        []Import{{Path: "./stale-format", Language: LanguageTS}},
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{{
		ID:      "entry_1",
		Imports: []Import{{Path: "./format"}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if program.Behaviors[0].TargetLanguage != LanguageTS {
		t.Fatalf("target language = %q, want typescript for already-target patch", program.Behaviors[0].TargetLanguage)
	}
	if program.Behaviors[0].Imports[0].Language != LanguageTS {
		t.Fatalf("patched import language = %q, want target", program.Behaviors[0].Imports[0].Language)
	}

	err = applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{{
		ID:      "entry_1",
		Imports: []Import{},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Behaviors[0].Imports) != 0 {
		t.Fatalf("explicit empty behavior imports did not clear target imports: %#v", program.Behaviors[0].Imports)
	}
}

func TestApplyBehaviorPatchRejectsForeignImportOnlyGlobalPatch(t *testing.T) {
	program := &Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageGo,
			Code:     "func helper() string { return \"closed\" }",
			Imports:  []Import{{Path: "strings", Language: LanguageGo}},
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Globals: program.Globals}

	for _, imports := range [][]Import{
		{{Path: "./format"}},
		{},
	} {
		err := applyBehaviorPatch(program, request, &BehaviorPatch{Globals: []CodeBlock{{
			ID:      "global_1",
			Imports: imports,
		}}})
		if err == nil || !strings.Contains(err.Error(), `global "global_1" imports without target code`) {
			t.Fatalf("error = %v, want foreign import-only global rejection for imports %#v", err, imports)
		}
		if len(program.Globals[0].Imports) != 1 || program.Globals[0].Imports[0].Path != "strings" {
			t.Fatalf("foreign import-only global rejection mutated imports: %#v", program.Globals[0].Imports)
		}
	}
}

func TestApplyBehaviorPatchAllowsAlreadyTargetImportOnlyGlobalPatch(t *testing.T) {
	program := &Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageTS,
			Code:     `const helper = () => "closed";`,
			Imports:  []Import{{Path: "./stale-format", Language: LanguageTS}},
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Globals: program.Globals}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{Globals: []CodeBlock{{
		ID:      "global_1",
		Imports: []Import{{Path: "./format", Specifiers: []ImportSpecifier{{Name: "record"}}}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if program.Globals[0].Language != LanguageTS {
		t.Fatalf("global language = %q, want typescript for already-target patch", program.Globals[0].Language)
	}
	if program.Globals[0].Code != `const helper = () => "closed";` {
		t.Fatalf("import-only global patch changed code: %#v", program.Globals[0])
	}
	if len(program.Globals[0].Imports) != 1 || program.Globals[0].Imports[0].Language != LanguageTS {
		t.Fatalf("patched global imports = %#v, want target import", program.Globals[0].Imports)
	}

	err = applyBehaviorPatch(program, request, &BehaviorPatch{Globals: []CodeBlock{{
		ID:      "global_1",
		Imports: []Import{},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals[0].Imports) != 0 {
		t.Fatalf("explicit empty global imports did not clear target imports: %#v", program.Globals[0].Imports)
	}
}

func TestApplyBehaviorPatchRejectsTargetBehaviorWithoutTargetBodyOrCode(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			Body:           "fmt.Println(\"closed\")",
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{{
		ID:             "entry_1",
		TargetLanguage: LanguageTS,
		Imports:        []Import{{Path: "./format"}},
	}}})
	if err == nil {
		t.Fatal("expected target behavior without target body/code error")
	}
}

func TestApplyBehaviorPatchRejectsBehaviorWithBodyAndCode(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{{
		ID:   "entry_1",
		Body: "translated();",
		Code: "const entry = () => {};",
	}}})
	if err == nil || !strings.Contains(err.Error(), `behavior "entry_1" must set either body or code, not both`) {
		t.Fatalf("error = %v, want body/code exclusivity rejection", err)
	}
}

func TestApplyBehaviorPatchRejectsFullCodeBehaviorNameChangesBeforeBackend(t *testing.T) {
	cases := []struct {
		target Language
		code   string
		want   string
	}{
		{
			target: LanguageGo,
			code:   `func RenamedEntry(ctx context.Context, instance hsm.Instance, event hsm.Event) {}`,
			want:   `go behavior full code declares "RenamedEntry", want compiler-owned name "Entry1"`,
		},
		{
			target: LanguageTS,
			code:   `const renamedEntry = (ctx, instance, event) => { instance.ready = true; };`,
			want:   `typescript behavior full code declares "renamedEntry", want compiler-owned name "entry1"`,
		},
		{
			target: LanguagePython,
			code:   `def renamed_entry(ctx, instance, event):` + "\n\t" + `instance.ready = True`,
			want:   `python behavior full code declares "renamed_entry", want compiler-owned name "entry_1"`,
		},
		{
			target: LanguageJava,
			code:   `private static void RenamedEntry(Context ctx, Instance instance, Event event) {}`,
			want:   `java behavior full code declares "RenamedEntry", want compiler-owned name "Entry1"`,
		},
		{
			target: LanguageRust,
			code:   `fn renamed_entry(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event) {}`,
			want:   `rust behavior full code declares "renamed_entry", want compiler-owned name "entry_1"`,
		},
		{
			target: LanguageZig,
			code:   `fn renamed_entry(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void {}`,
			want:   `zig behavior full code declares "renamed_entry", want compiler-owned name "entry_1"`,
		},
	}
	for _, tc := range cases {
		t.Run(string(tc.target), func(t *testing.T) {
			program := &Program{Behaviors: []Behavior{{
				ID:             "entry_1",
				OwnerID:        "state_1",
				Kind:           BehaviorEntry,
				SourceLanguage: LanguageGo,
			}}}
			request := AdapterRequest{TargetLanguage: tc.target, Behaviors: program.Behaviors}

			err := applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{{
				ID:   "entry_1",
				Code: tc.code,
			}}})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestApplyBehaviorPatchRejectsFullCodeBehaviorSignatureChangesBeforeBackend(t *testing.T) {
	cases := []struct {
		target Language
		code   string
		want   string
	}{
		{
			target: LanguageCSharp,
			code:   `private static void Entry1() {}`,
			want:   `csharp behavior full code declares parameters "()", want compiler-owned parameters "(Context ctx, Instance instance, Event @event)"`,
		},
		{
			target: LanguageCPP,
			code:   `static constexpr auto entry_1 = []() -> void {};`,
			want:   `cpp behavior full code declares parameters "()", want compiler-owned parameters "(auto& signal, auto& instance, const auto& event)"`,
		},
		{
			target: LanguageDart,
			code:   `FutureOr<void> entry1() {}`,
			want:   `dart behavior full code declares parameters "()", want compiler-owned parameters "(Context ctx, Instance instance, Event event)"`,
		},
		{
			target: LanguageGo,
			code:   `func Entry1() {}`,
			want:   `go behavior full code declares signature "()", want compiler-owned signature "(ctx context.Context, instance hsm.Instance, event hsm.Event)"`,
		},
		{
			target: LanguageJava,
			code:   `private static void Entry1() {}`,
			want:   `java behavior full code declares parameters "()", want compiler-owned parameters "(Context ctx, Instance instance, Event event)"`,
		},
		{
			target: LanguageJS,
			code:   `function entry1() {}`,
			want:   `javascript behavior full code declares parameters "()", want compiler-owned parameters "(ctx, instance, event)"`,
		},
		{
			target: LanguagePython,
			code:   `async def entry_1() -> None:` + "\n\t" + `pass`,
			want:   `python behavior full code declares parameters "()", want compiler-owned parameters "(ctx: hsm.Context, instance: hsm.Instance, event: hsm.Event)"`,
		},
		{
			target: LanguageRust,
			code:   `fn entry_1() {}`,
			want:   `rust behavior full code declares parameters "()", want compiler-owned parameters "(ctx: &hsm::Context, instance: &mut HsmcInstance, event: &hsm::Event)"`,
		},
		{
			target: LanguageTS,
			code:   `function entry1(): void {}`,
			want:   `typescript behavior full code declares parameters "()", want compiler-owned parameters "(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event)"`,
		},
		{
			target: LanguageZig,
			code:   `fn entry_1() void {}`,
			want:   `zig behavior full code declares parameters "()", want compiler-owned parameters "(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event)"`,
		},
	}
	for _, tc := range cases {
		t.Run(string(tc.target), func(t *testing.T) {
			program := &Program{Behaviors: []Behavior{{
				ID:             "entry_1",
				OwnerID:        "state_1",
				Kind:           BehaviorEntry,
				SourceLanguage: LanguageGo,
			}}}
			AssignTargetABI(program, tc.target)
			request := AdapterRequest{TargetLanguage: tc.target, Behaviors: program.Behaviors}

			err := applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{{
				ID:   "entry_1",
				Code: tc.code,
			}}})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestApplyBehaviorPatchRejectsNonDeclarationFullCodeForDeclarationOnlyTargets(t *testing.T) {
	cases := []struct {
		target Language
		code   string
		want   string
	}{
		{
			target: LanguageJava,
			code:   `(ctx, instance, event) -> AdapterFormat.record("entry")`,
			want:   `java behavior code for "Entry1" must be a compiler-owned method declaration`,
		},
		{
			target: LanguageJava,
			code:   `private static Event Entry1 = null;`,
			want:   `java behavior code for "Entry1" must be a compiler-owned method declaration`,
		},
		{
			target: LanguageRust,
			code:   `|ctx, instance, event| { let _ = (ctx, instance, event); }`,
			want:   `rust behavior code for "entry_1" must be a compiler-owned function declaration`,
		},
		{
			target: LanguageRust,
			code:   `pub static entry_1: Option<fn(&hsm::Context, &mut HsmcInstance, &hsm::Event)> = None;`,
			want:   `rust behavior code for "entry_1" must be a compiler-owned function declaration`,
		},
		{
			target: LanguageZig,
			code:   `struct { fn call(ctx: *hsm.Context, inst: *hsm.Instance, event: hsm.Event) void {} }.call`,
			want:   `zig behavior code for "entry_1" must be a compiler-owned function declaration`,
		},
		{
			target: LanguageZig,
			code:   `pub var entry_1: ?*const fn (*hsm.Context, *hsm.Instance, hsm.Event) void = null;`,
			want:   `zig behavior code for "entry_1" must be a compiler-owned function declaration`,
		},
	}
	for _, tc := range cases {
		t.Run(string(tc.target), func(t *testing.T) {
			program := &Program{Behaviors: []Behavior{{
				ID:             "entry_1",
				OwnerID:        "state_1",
				Kind:           BehaviorEntry,
				SourceLanguage: LanguageGo,
			}}}
			request := AdapterRequest{TargetLanguage: tc.target, Behaviors: program.Behaviors}

			err := applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{{
				ID:   "entry_1",
				Code: tc.code,
			}}})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestApplyBehaviorPatchRejectsBehaviorInlineMetadata(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{{
		ID:     "entry_1",
		Inline: true,
		Body:   "translated();",
	}}})
	if err == nil || !strings.Contains(err.Error(), `behavior "entry_1" attempts to change inline`) {
		t.Fatalf("error = %v, want inline metadata rejection", err)
	}
}

func TestApplyBehaviorPatchPreservesBehaviorInlineMetadata(t *testing.T) {
	program := &Program{
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			Inline:         true,
			SourceLanguage: LanguageTS,
			TargetLanguage: LanguageTS,
			Body:           "record('closed');",
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{{
		ID:      "entry_1",
		Imports: []Import{{Path: "./record"}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if !program.Behaviors[0].Inline {
		t.Fatalf("inline metadata was not preserved: %#v", program.Behaviors[0])
	}
}

func TestApplyBehaviorPatchClearsSourceImportsWhenTranslatedCodeOmitsImports(t *testing.T) {
	program := &Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageGo,
			Code:     "func helper() string { return strings.ToUpper(\"closed\") }",
			Imports:  []Import{{Path: "strings", Language: LanguageGo}},
		}},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageGo,
			Body:           "fmt.Println(\"closed\")",
			Imports:        []Import{{Path: "fmt", Language: LanguageGo}},
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Globals: program.Globals, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{
		Globals: []CodeBlock{{
			ID:   "global_1",
			Code: "const helper = () => \"closed\";",
		}},
		Behaviors: []Behavior{{
			ID:   "entry_1",
			Body: "instance.ready = true;",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Globals[0].Imports) != 0 {
		t.Fatalf("translated global imports = %#v, want source imports cleared", program.Globals[0].Imports)
	}
	if len(program.Behaviors[0].Imports) != 0 {
		t.Fatalf("translated behavior imports = %#v, want source imports cleared", program.Behaviors[0].Imports)
	}
}

func TestApplyBehaviorPatchClearsStaleSourceBehaviorCodeFieldsWhenTranslated(t *testing.T) {
	translatedCode := "(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void => { instance.ready = true; }"
	program := &Program{
		Behaviors: []Behavior{
			{
				ID:             "entry_1",
				OwnerID:        "state_1",
				Kind:           BehaviorEntry,
				SourceLanguage: LanguageGo,
				Body:           "fmt.Println(\"closed\")",
				Code:           "func entered(ctx context.Context, instance hsm.Instance, event hsm.Event) {}",
			},
			{
				ID:             "entry_2",
				OwnerID:        "state_2",
				Kind:           BehaviorEntry,
				SourceLanguage: LanguageGo,
				Body:           "fmt.Println(\"open\")",
				Code:           "func opened(ctx context.Context, instance hsm.Instance, event hsm.Event) {}",
			},
		},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{
		{ID: "entry_1", Body: "instance.ready = true;"},
		{ID: "entry_2", Code: translatedCode},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if program.Behaviors[0].Body != "instance.ready = true;" || program.Behaviors[0].Code != "" {
		t.Fatalf("body-only translated behavior kept stale code: %#v", program.Behaviors[0])
	}
	if program.Behaviors[1].Body != "" || program.Behaviors[1].Code != translatedCode {
		t.Fatalf("code-only translated behavior kept stale body: %#v", program.Behaviors[1])
	}
}

func TestApplyBehaviorPatchClearsStaleTargetBehaviorCodeFieldsWhenTranslated(t *testing.T) {
	translatedCode := "(ctx: hsm.Context, instance: InstanceType<typeof hsm.Instance>, event: hsm.Event): void => { instance.newCode = true; }"
	program := &Program{
		Behaviors: []Behavior{
			{
				ID:             "entry_1",
				OwnerID:        "state_1",
				Kind:           BehaviorEntry,
				SourceLanguage: LanguageTS,
				TargetLanguage: LanguageTS,
				Body:           "instance.oldBody = true;",
				Code:           "(ctx, instance, event) => { instance.oldCode = true; }",
			},
			{
				ID:             "entry_2",
				OwnerID:        "state_2",
				Kind:           BehaviorEntry,
				SourceLanguage: LanguageTS,
				TargetLanguage: LanguageTS,
				Body:           "instance.oldBody = true;",
				Code:           "(ctx, instance, event) => { instance.oldCode = true; }",
			},
		},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{Behaviors: []Behavior{
		{ID: "entry_1", Body: "instance.newBody = true;"},
		{ID: "entry_2", Code: translatedCode},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if program.Behaviors[0].Body != "instance.newBody = true;" || program.Behaviors[0].Code != "" {
		t.Fatalf("body-only target behavior patch kept stale code: %#v", program.Behaviors[0])
	}
	if program.Behaviors[1].Body != "" || program.Behaviors[1].Code != translatedCode {
		t.Fatalf("code-only target behavior patch kept stale body: %#v", program.Behaviors[1])
	}
}

func TestApplyBehaviorPatchPreservesOmittedMutableCode(t *testing.T) {
	program := &Program{
		Globals: []CodeBlock{{
			ID:       "global_1",
			Language: LanguageTS,
			Code:     "const helper = () => true;",
			Imports:  []Import{{Path: "./old-global", Language: LanguageTS}},
		}},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageTS,
			TargetLanguage: LanguageTS,
			Body:           "instance.ready = true;",
			Code:           "(ctx, instance) => { instance.ready = true; }",
			Imports:        []Import{{Path: "./old-behavior", Language: LanguageTS}},
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Globals: program.Globals, Behaviors: program.Behaviors}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{
		Globals: []CodeBlock{{
			ID:      "global_1",
			Imports: []Import{{Path: "./new-global"}},
		}},
		Behaviors: []Behavior{{
			ID:      "entry_1",
			Imports: []Import{{Path: "./new-behavior"}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if program.Globals[0].Code != "const helper = () => true;" || program.Globals[0].Language != LanguageTS {
		t.Fatalf("global code/language not preserved: %#v", program.Globals[0])
	}
	if program.Behaviors[0].Body != "instance.ready = true;" || program.Behaviors[0].Code == "" {
		t.Fatalf("behavior code/body not preserved: %#v", program.Behaviors[0])
	}
	if program.Globals[0].Imports[0].Path != "./new-global" || program.Globals[0].Imports[0].Language != LanguageTS {
		t.Fatalf("global imports not patched/defaulted: %#v", program.Globals[0].Imports)
	}
	if program.Behaviors[0].Imports[0].Path != "./new-behavior" || program.Behaviors[0].Imports[0].Language != LanguageTS {
		t.Fatalf("behavior imports not patched/defaulted: %#v", program.Behaviors[0].Imports)
	}
}

func TestApplyBehaviorPatchRejectsTargetGlobalWithoutTargetCode(t *testing.T) {
	program := &Program{
		Globals: []CodeBlock{{ID: "global_1", Language: LanguageGo, Code: "func helper() bool { return true }"}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Globals: program.Globals}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{Globals: []CodeBlock{{ID: "global_1", Language: LanguageTS}}})
	if err == nil {
		t.Fatal("expected target global without target code error")
	}
}

func TestApplyBehaviorPatchRejectsWhitespaceOnlyTargetGlobalCode(t *testing.T) {
	program := &Program{
		Globals: []CodeBlock{{ID: "global_1", Language: LanguageGo, Code: "func helper() bool { return true }"}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Globals: program.Globals}

	err := applyBehaviorPatch(program, request, &BehaviorPatch{Globals: []CodeBlock{{ID: "global_1", Language: LanguageTS, Code: " \n\t "}}})
	if err == nil {
		t.Fatal("expected whitespace-only target global code error")
	}
}

func TestApplyBehaviorPatchRejectsWhitespaceOnlyMutableCodeFields(t *testing.T) {
	program := &Program{
		Globals: []CodeBlock{{ID: "global_1", Language: LanguageTS, Code: "const helper = () => true;"}},
		Behaviors: []Behavior{{
			ID:             "entry_1",
			OwnerID:        "state_1",
			Kind:           BehaviorEntry,
			SourceLanguage: LanguageTS,
			TargetLanguage: LanguageTS,
			Body:           "instance.ready = true;",
			Code:           "(ctx, instance, event) => { instance.ready = true; }",
		}},
	}
	request := AdapterRequest{TargetLanguage: LanguageTS, Globals: program.Globals, Behaviors: program.Behaviors}
	cases := []struct {
		name  string
		patch BehaviorPatch
		want  string
	}{
		{
			name:  "global code",
			patch: BehaviorPatch{Globals: []CodeBlock{{ID: "global_1", Code: " \n\t "}}},
			want:  `global "global_1" contains whitespace-only code`,
		},
		{
			name:  "behavior body",
			patch: BehaviorPatch{Behaviors: []Behavior{{ID: "entry_1", Body: " \n\t "}}},
			want:  `behavior "entry_1" contains whitespace-only body`,
		},
		{
			name:  "behavior code",
			patch: BehaviorPatch{Behaviors: []Behavior{{ID: "entry_1", Code: " \n\t "}}},
			want:  `behavior "entry_1" contains whitespace-only code`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := applyBehaviorPatch(program, request, &tc.patch)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
