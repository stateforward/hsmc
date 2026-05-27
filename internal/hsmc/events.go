package hsmc

import "strings"

func eventSymbolFor(language Language, symbol string) string {
	switch language {
	case LanguageCSharp, LanguageJava:
		return exportName(symbol)
	case LanguageCPP, LanguageElixir, LanguageRust, LanguageZig:
		return snakeName(symbol)
	case LanguageGo:
		return exportName(symbol)
	case LanguagePython:
		return snakeName(symbol)
	default:
		return lowerCamel(exportName(symbol))
	}
}

func eventMemberSymbol(value string, suffixes ...string) (string, bool) {
	value = strings.TrimSpace(value)
	for _, suffix := range suffixes {
		if strings.HasSuffix(value, suffix) && len(value) > len(suffix) {
			return strings.TrimSpace(strings.TrimSuffix(value, suffix)), true
		}
	}
	return "", false
}
