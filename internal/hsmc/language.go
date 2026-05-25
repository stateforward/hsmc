package hsmc

import "strings"

var supportedSourceLanguages = []Language{
	LanguageCSharp,
	LanguageCPP,
	LanguageDart,
	LanguageGo,
	LanguageJava,
	LanguageJS,
	LanguagePython,
	LanguageTS,
	LanguageRust,
	LanguageZig,
	LanguageJSONIR,
}

var supportedTargetLanguages = []Language{
	LanguageCSharp,
	LanguageCPP,
	LanguageDart,
	LanguageGo,
	LanguageJava,
	LanguageJS,
	LanguagePython,
	LanguageTS,
	LanguageRust,
	LanguageZig,
	LanguageJSONIR,
	LanguageMermaid,
	LanguagePlantUML,
}

// SupportedLanguages returns the canonical language IDs understood by hsmc as
// either source or target languages.
func SupportedLanguages() []Language {
	languages := append([]Language(nil), supportedTargetLanguages...)
	seen := map[Language]bool{}
	for _, language := range languages {
		seen[language] = true
	}
	for _, language := range supportedSourceLanguages {
		if !seen[language] {
			languages = append(languages, language)
		}
	}
	return languages
}

func SupportedSourceLanguages() []Language {
	return append([]Language(nil), supportedSourceLanguages...)
}

func SupportedTargetLanguages() []Language {
	return append([]Language(nil), supportedTargetLanguages...)
}

func ParseLanguage(value string) (Language, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "cs", "csharp", "c#", "c-sharp":
		return LanguageCSharp, true
	case "c++", "cc", "cpp", "cxx":
		return LanguageCPP, true
	case "dart":
		return LanguageDart, true
	case "go", "golang":
		return LanguageGo, true
	case "java":
		return LanguageJava, true
	case "cjs", "js", "javascript", "jsx", "mjs":
		return LanguageJS, true
	case "py", "python":
		return LanguagePython, true
	case "rs", "rust":
		return LanguageRust, true
	case "cts", "mts", "ts", "tsx", "typescript":
		return LanguageTS, true
	case "zig":
		return LanguageZig, true
	case "ir", "json", "json-ir":
		return LanguageJSONIR, true
	case "mermaid", "mmd":
		return LanguageMermaid, true
	case "plantuml", "puml", "plant":
		return LanguagePlantUML, true
	default:
		return "", false
	}
}
