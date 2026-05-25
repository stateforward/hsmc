package hsmc

import (
	"fmt"
	"strconv"
	"strings"
)

func collectRenderableImports(program *Program, language Language, reservedPaths ...string) []Import {
	seen := map[string]int{}
	for _, path := range reservedPaths {
		seen[importMergeKey(Import{Path: path, Language: language}, language)] = -1
	}
	var imports []Import
	emit := func(imp Import) {
		if imp.Path == "" {
			return
		}
		if imp.Language != "" && imp.Language != language {
			return
		}
		key := importMergeKey(imp, language)
		if existingIndex, ok := seen[key]; ok {
			if existingIndex >= 0 {
				merged, ok := mergeCompatibleImports(imports[existingIndex], imp)
				if ok {
					imports[existingIndex] = merged
					return
				}
				key = nextImportMergeKey(key, seen)
				seen[key] = len(imports)
				imports = append(imports, imp)
			}
			return
		}
		seen[key] = len(imports)
		imports = append(imports, imp)
	}
	for _, imp := range program.Imports {
		emit(imp)
	}
	for _, global := range program.Globals {
		for _, imp := range global.Imports {
			emit(imp)
		}
	}
	for _, behavior := range program.Behaviors {
		for _, imp := range behavior.Imports {
			emit(imp)
		}
	}
	return imports
}

func importMergeKey(imp Import, target Language) string {
	path := imp.Path
	if target == LanguageCPP {
		path = normalizeCPPIncludePath(path)
	}
	static := ""
	if imp.Static {
		static = "static"
	}
	deferred := ""
	if imp.Deferred {
		deferred = "deferred"
	}
	typeOnly := ""
	if imp.TypeOnly {
		typeOnly = "type"
	}
	if imp.Alias != "" {
		return strings.Join([]string{path, string(imp.Language), static, deferred, typeOnly, "alias", imp.Alias, "hidden", strings.Join(imp.Hidden, ",")}, "\x00")
	}
	if target == LanguageJS || target == LanguageTS || target == LanguageXState {
		if imp.Default != "" || len(imp.Specifiers) > 0 {
			return strings.Join([]string{path, string(imp.Language), static, deferred, typeOnly, "module-members"}, "\x00")
		}
	}
	if len(imp.Specifiers) > 0 {
		return strings.Join([]string{path, string(imp.Language), static, deferred, typeOnly, "members", strings.Join(imp.Hidden, ",")}, "\x00")
	}
	return strings.Join([]string{path, string(imp.Language), static, deferred, typeOnly, "bare", strings.Join(imp.Hidden, ",")}, "\x00")
}

func nextImportMergeKey(key string, seen map[string]int) string {
	for index := 1; ; index++ {
		next := key + "\x00conflict\x00" + strconv.Itoa(index)
		if _, ok := seen[next]; !ok {
			return next
		}
	}
}

func mergeCompatibleImports(existing Import, incoming Import) (Import, bool) {
	if existing.Static != incoming.Static {
		return Import{}, false
	}
	if existing.Deferred != incoming.Deferred {
		return Import{}, false
	}
	if existing.TypeOnly != incoming.TypeOnly {
		return Import{}, false
	}
	if existing.Default != "" && incoming.Default != "" && existing.Default != incoming.Default {
		return Import{}, false
	}
	if existing.Default == "" {
		existing.Default = incoming.Default
	}
	existing.Specifiers = appendUniqueSpecifiers(existing.Specifiers, incoming.Specifiers)
	existing.Hidden = appendUniqueStrings(existing.Hidden, incoming.Hidden)
	existing.LocalNames = appendUniqueStrings(existing.LocalNames, incoming.LocalNames)
	return existing, true
}

func appendUniqueSpecifiers(existing []ImportSpecifier, incoming []ImportSpecifier) []ImportSpecifier {
	for _, specifier := range incoming {
		var found bool
		for _, current := range existing {
			if current == specifier {
				found = true
				break
			}
		}
		if !found {
			existing = append(existing, specifier)
		}
	}
	return existing
}

func appendUniqueStrings(existing []string, incoming []string) []string {
	for _, value := range incoming {
		var found bool
		for _, current := range existing {
			if current == value {
				found = true
				break
			}
		}
		if !found {
			existing = append(existing, value)
		}
	}
	return existing
}

func formatESImport(imp Import) string {
	path := strconv.Quote(imp.Path)
	prefix := "import"
	if imp.TypeOnly {
		prefix = "import type"
	}
	if imp.Alias != "" {
		return fmt.Sprintf("%s * as %s from %s;", prefix, imp.Alias, path)
	}
	specifiers := formatESImportSpecifiers(imp.Specifiers, imp.TypeOnly)
	switch {
	case imp.Default != "" && specifiers != "":
		return fmt.Sprintf("%s %s, { %s } from %s;", prefix, imp.Default, specifiers, path)
	case imp.Default != "":
		return fmt.Sprintf("%s %s from %s;", prefix, imp.Default, path)
	case specifiers != "":
		return fmt.Sprintf("%s { %s } from %s;", prefix, specifiers, path)
	default:
		return fmt.Sprintf("%s %s;", prefix, path)
	}
}

func formatESImportSpecifiers(specifiers []ImportSpecifier, importTypeOnly bool) string {
	parts := make([]string, 0, len(specifiers))
	for _, specifier := range specifiers {
		if specifier.Name == "" {
			continue
		}
		name := specifier.Name
		if specifier.TypeOnly && !importTypeOnly {
			name = "type " + name
		}
		if specifier.Alias != "" && specifier.Alias != specifier.Name {
			parts = append(parts, name+" as "+specifier.Alias)
			continue
		}
		parts = append(parts, name)
	}
	return strings.Join(parts, ", ")
}

func formatCSharpImport(imp Import) string {
	if imp.Alias != "" {
		return fmt.Sprintf("using %s = %s;", imp.Alias, imp.Path)
	}
	if csharpImportIsStatic(imp) {
		return fmt.Sprintf("using static %s;", imp.Path)
	}
	return fmt.Sprintf("using %s;", imp.Path)
}

func csharpImportIsStatic(imp Import) bool {
	if imp.Static {
		return true
	}
	for _, localName := range imp.LocalNames {
		if localName == "*" {
			return true
		}
	}
	return false
}

func formatJavaImport(imp Import) string {
	if len(imp.Specifiers) == 1 && imp.Specifiers[0].Name == "*" {
		if imp.Static {
			return fmt.Sprintf("import static %s.*;", imp.Path)
		}
		return fmt.Sprintf("import %s.*;", imp.Path)
	}
	if imp.Static {
		return fmt.Sprintf("import static %s;", imp.Path)
	}
	return fmt.Sprintf("import %s;", imp.Path)
}

func formatCPPImport(imp Import) string {
	path := strings.TrimSpace(imp.Path)
	if strings.HasPrefix(path, "<") || strings.HasPrefix(path, "\"") {
		return "#include " + path
	}
	if strings.Contains(path, "/") || strings.Contains(path, ".") {
		return fmt.Sprintf("#include %q", path)
	}
	return fmt.Sprintf("#include <%s>", path)
}

func cppImportBindingName(path string) string {
	path = strings.Trim(path, "<>\"")
	path = strings.TrimSpace(path)
	path = strings.TrimSuffix(path, ".hpp")
	path = strings.TrimSuffix(path, ".h")
	path = strings.TrimSuffix(path, ".hh")
	path = strings.TrimSuffix(path, ".hxx")
	if index := strings.LastIndex(path, "/"); index >= 0 {
		path = path[index+1:]
	}
	if path == "" {
		return ""
	}
	return path
}

func formatDartImport(imp Import) string {
	var out strings.Builder
	fmt.Fprintf(&out, "import %s", strconv.Quote(imp.Path))
	if imp.Deferred {
		out.WriteString(" deferred")
	}
	if imp.Alias != "" {
		fmt.Fprintf(&out, " as %s", imp.Alias)
	}
	if len(imp.Specifiers) > 0 {
		fmt.Fprintf(&out, " show %s", formatDartImportSpecifiers(imp.Specifiers))
	}
	if len(imp.Hidden) > 0 {
		fmt.Fprintf(&out, " hide %s", strings.Join(imp.Hidden, ", "))
	}
	out.WriteString(";")
	return out.String()
}

func formatDartImportSpecifiers(specifiers []ImportSpecifier) string {
	parts := make([]string, 0, len(specifiers))
	for _, specifier := range specifiers {
		if specifier.Name == "" {
			continue
		}
		parts = append(parts, specifier.Name)
	}
	return strings.Join(parts, ", ")
}

func formatPythonImport(imp Import) string {
	if len(imp.Specifiers) > 0 {
		return fmt.Sprintf("from %s import %s", imp.Path, formatPythonImportSpecifiers(imp.Specifiers))
	}
	if imp.Alias != "" {
		return fmt.Sprintf("import %s as %s", imp.Path, imp.Alias)
	}
	return fmt.Sprintf("import %s", imp.Path)
}

func formatPythonImportSpecifiers(specifiers []ImportSpecifier) string {
	parts := make([]string, 0, len(specifiers))
	for _, specifier := range specifiers {
		if specifier.Name == "" {
			continue
		}
		if specifier.Alias != "" && specifier.Alias != specifier.Name {
			parts = append(parts, specifier.Name+" as "+specifier.Alias)
			continue
		}
		parts = append(parts, specifier.Name)
	}
	return strings.Join(parts, ", ")
}

func formatRustImport(imp Import) string {
	if len(imp.Specifiers) > 0 {
		return fmt.Sprintf("use %s::{%s};", imp.Path, formatRustImportSpecifiers(imp.Specifiers))
	}
	if imp.Alias != "" {
		return fmt.Sprintf("use %s as %s;", imp.Path, imp.Alias)
	}
	return fmt.Sprintf("use %s;", imp.Path)
}

func formatRustImportSpecifiers(specifiers []ImportSpecifier) string {
	parts := make([]string, 0, len(specifiers))
	for _, specifier := range specifiers {
		if specifier.Name == "" {
			continue
		}
		if specifier.Alias != "" && specifier.Alias != specifier.Name {
			parts = append(parts, specifier.Name+" as "+specifier.Alias)
			continue
		}
		parts = append(parts, specifier.Name)
	}
	return strings.Join(parts, ", ")
}

func formatZigImport(imp Import) string {
	binding := imp.Alias
	if binding == "" {
		binding = zigImportBindingName(imp.Path)
	}
	return fmt.Sprintf("const %s = @import(%s);", binding, strconv.Quote(imp.Path))
}

func zigImportBindingName(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "\"")
	path = strings.TrimSuffix(path, ".zig")
	if index := strings.LastIndex(path, "/"); index >= 0 {
		path = path[index+1:]
	}
	name := snakeName(path)
	if name == "" {
		return "imported"
	}
	return name
}
