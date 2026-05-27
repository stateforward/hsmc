package hsmc

import (
	"path"
	"regexp"
	"strings"
)

func AssignMutableImports(program *Program) {
	if program == nil || len(program.Imports) == 0 {
		return
	}
	var structural []Import
	for _, imp := range program.Imports {
		if isHSMImportPath(imp.Path) {
			continue
		}
		imp.LocalNames = normalizeLocalNames(imp)
		used := false
		for index := range program.Globals {
			if regionUsesImport(program.Globals[index].Language, program.Globals[index].Code, imp) {
				program.Globals[index].Imports = appendImport(program.Globals[index].Imports, imp)
				used = true
			}
		}
		for index := range program.Behaviors {
			code := program.Behaviors[index].Code + "\n" + program.Behaviors[index].Body + "\n" + program.Behaviors[index].Signature
			if regionUsesImport(program.Behaviors[index].SourceLanguage, code, imp) {
				program.Behaviors[index].Imports = appendImport(program.Behaviors[index].Imports, imp)
				used = true
			}
		}
		if !used {
			structural = append(structural, imp)
		}
	}
	program.Imports = structural
}

func regionUsesImport(language Language, code string, imp Import) bool {
	if code == "" {
		return false
	}
	if imp.Language != "" && language != "" && imp.Language != language {
		return false
	}
	if imp.Alias == "." {
		return true
	}
	for _, specifier := range imp.Specifiers {
		if specifier.Name == "*" {
			return true
		}
	}
	if imp.Language == LanguageDart && imp.Alias == "" && len(imp.Specifiers) == 0 {
		return true
	}
	for _, name := range normalizeLocalNames(imp) {
		if name == "*" {
			return true
		}
		if name == "." || name == "_" || name == "" {
			continue
		}
		if identifierInCode(code, name) {
			return true
		}
	}
	return false
}

func normalizeLocalNames(imp Import) []string {
	seen := map[string]bool{}
	var names []string
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		names = append(names, name)
	}
	for _, name := range imp.LocalNames {
		add(name)
	}
	if imp.Alias != "" {
		add(imp.Alias)
		if imp.Language == LanguageDart {
			return names
		}
	}
	if imp.Default != "" {
		add(imp.Default)
	}
	for _, specifier := range imp.Specifiers {
		if specifier.Alias != "" {
			add(specifier.Alias)
		} else {
			add(specifier.Name)
		}
	}
	if len(names) == 0 {
		base := path.Base(imp.Path)
		base = strings.TrimSuffix(base, ".go")
		base = strings.TrimSuffix(base, ".ts")
		if imp.Language == LanguageCPP {
			base = strings.TrimSuffix(base, ".hpp")
			base = strings.TrimSuffix(base, ".hxx")
			base = strings.TrimSuffix(base, ".hh")
			base = strings.TrimSuffix(base, ".h")
		}
		if imp.Language == LanguagePython {
			base, _, _ = strings.Cut(base, ".")
		}
		if imp.Language == LanguageElixir {
			parts := strings.Split(base, ".")
			base = parts[len(parts)-1]
		}
		add(base)
	}
	return names
}

func appendImport(imports []Import, imp Import) []Import {
	for _, existing := range imports {
		if existing.Path == imp.Path && existing.Alias == imp.Alias && existing.Default == imp.Default && importSpecifiersEqual(existing.Specifiers, imp.Specifiers) && stringSlicesEqual(existing.Hidden, imp.Hidden) && existing.Static == imp.Static && existing.Deferred == imp.Deferred && existing.TypeOnly == imp.TypeOnly && existing.Language == imp.Language {
			return imports
		}
	}
	return append(imports, imp)
}

func importSpecifiersEqual(left []ImportSpecifier, right []ImportSpecifier) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func stringSlicesEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func isHSMImportPath(path string) bool {
	return path == "github.com/stateforward/hsm.go" ||
		path == "@stateforward/hsm" ||
		path == "@stateforward/hsm.ts" ||
		path == "hsm/hsm.hpp" ||
		path == "package:hsm/hsm.dart" ||
		path == "hsm.ex" ||
		path == "Stateforward.Hsm" ||
		path == "Stateforward.Hsm.Hsm" ||
		path == "com.stateforward.hsm" ||
		path == "com.stateforward.hsm.Hsm" ||
		path == "hsm" ||
		path == "hsm::*"
}

func identifierInCode(code string, name string) bool {
	pattern := regexp.MustCompile(`(^|[^A-Za-z0-9_$])` + regexp.QuoteMeta(name) + `([^A-Za-z0-9_$]|$)`)
	return pattern.FindStringIndex(code) != nil
}
