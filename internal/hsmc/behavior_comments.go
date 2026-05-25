package hsmc

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

func behaviorSourceText(behavior Behavior) string {
	body := strings.TrimSpace(behavior.Body)
	code := strings.TrimSpace(behavior.Code)
	switch {
	case body != "" && code != "":
		return "body:\n" + body + "\n\ncode:\n" + code
	case body != "":
		return body
	default:
		return code
	}
}

func writeCommentedBehaviorSource(out *bytes.Buffer, indent string, commentPrefix string, behavior Behavior) {
	fmt.Fprintf(out, "%s%s Original %s behavior %s preserved for manual porting:\n", indent, commentPrefix, behavior.SourceLanguage, behavior.ID)
	if target := behaviorCommentTargetLanguage(behavior); target != "" && target != behavior.SourceLanguage {
		fmt.Fprintf(out, "%s%s Target language: %s\n", indent, commentPrefix, target)
	}
	if behavior.Kind != "" {
		fmt.Fprintf(out, "%s%s Behavior kind: %s\n", indent, commentPrefix, behavior.Kind)
	}
	if len(behavior.Imports) > 0 {
		fmt.Fprintf(out, "%s%s Source imports:\n", indent, commentPrefix)
		for _, imp := range behavior.Imports {
			if line := sourceImportText(importCommentLanguage(behavior.SourceLanguage, imp), imp); line != "" {
				fmt.Fprintf(out, "%s%s %s\n", indent, commentPrefix, line)
			}
		}
	}
	source := behaviorSourceText(behavior)
	if source == "" {
		fmt.Fprintf(out, "%s%s No source behavior body was captured.\n", indent, commentPrefix)
		return
	}
	for _, line := range strings.Split(source, "\n") {
		if strings.TrimSpace(line) == "" {
			fmt.Fprintf(out, "%s%s\n", indent, commentPrefix)
			continue
		}
		fmt.Fprintf(out, "%s%s %s\n", indent, commentPrefix, line)
	}
}

func behaviorCommentTargetLanguage(behavior Behavior) Language {
	if behavior.TargetLanguage != "" {
		return behavior.TargetLanguage
	}
	if behavior.TargetABI != nil {
		return behavior.TargetABI.Language
	}
	return ""
}

func sourceImportText(language Language, imp Import) string {
	switch language {
	case LanguageCSharp:
		return formatCSharpImport(imp)
	case LanguageCPP:
		return formatCPPImport(imp)
	case LanguageDart:
		return formatDartImport(imp)
	case LanguageGo:
		if imp.Path == "" {
			return ""
		}
		if imp.Alias != "" {
			return "import " + imp.Alias + " " + strconv.Quote(imp.Path)
		}
		return "import " + strconv.Quote(imp.Path)
	case LanguageJava:
		return formatJavaImport(imp)
	case LanguageJS, LanguageTS:
		return formatESImport(imp)
	case LanguagePython:
		return formatPythonImport(imp)
	case LanguageRust:
		return formatRustImport(imp)
	case LanguageZig:
		return formatZigImport(imp)
	default:
		if imp.Path == "" {
			return ""
		}
		return imp.Path
	}
}

func writeCommentedGlobalSource(out *bytes.Buffer, indent string, commentPrefix string, global CodeBlock, target Language) {
	source := strings.TrimSpace(global.Code)
	if source == "" {
		return
	}
	fmt.Fprintf(out, "%s%s Original %s global %s preserved for manual porting:\n", indent, commentPrefix, global.Language, global.ID)
	if target != "" && target != global.Language {
		fmt.Fprintf(out, "%s%s Target language: %s\n", indent, commentPrefix, target)
	}
	if len(global.Imports) > 0 {
		fmt.Fprintf(out, "%s%s Source imports:\n", indent, commentPrefix)
		for _, imp := range global.Imports {
			if line := sourceImportText(importCommentLanguage(global.Language, imp), imp); line != "" {
				fmt.Fprintf(out, "%s%s %s\n", indent, commentPrefix, line)
			}
		}
	}
	for _, line := range strings.Split(source, "\n") {
		if strings.TrimSpace(line) == "" {
			fmt.Fprintf(out, "%s%s\n", indent, commentPrefix)
			continue
		}
		fmt.Fprintf(out, "%s%s %s\n", indent, commentPrefix, line)
	}
	out.WriteString("\n")
}

func importCommentLanguage(regionLanguage Language, imp Import) Language {
	if imp.Language != "" {
		return imp.Language
	}
	return regionLanguage
}

func commentLines(indent string, commentPrefix string, text string) string {
	var out strings.Builder
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		if strings.TrimSpace(line) == "" {
			fmt.Fprintf(&out, "%s%s\n", indent, commentPrefix)
			continue
		}
		fmt.Fprintf(&out, "%s%s %s\n", indent, commentPrefix, line)
	}
	return strings.TrimRight(out.String(), "\n")
}

func goUntranslatedReturn(behavior Behavior) string {
	returnType := ""
	if behavior.TargetABI != nil {
		returnType = behavior.TargetABI.ReturnType
	}
	switch returnType {
	case "":
		return ""
	case "bool":
		return "return false"
	case "time.Duration":
		return "return 0"
	case "time.Time":
		return "return time.Time{}"
	case "<-chan struct{}":
		return "return nil"
	default:
		return "return nil"
	}
}

func csharpUntranslatedReturn(behavior Behavior) string {
	returnType := "void"
	if behavior.TargetABI != nil && behavior.TargetABI.ReturnType != "" {
		returnType = behavior.TargetABI.ReturnType
	}
	switch returnType {
	case "void":
		return ""
	case "bool":
		return "return false;"
	case "System.TimeSpan":
		return "return System.TimeSpan.Zero;"
	case "System.DateTimeOffset":
		return "return System.DateTimeOffset.UnixEpoch;"
	case "System.Threading.Tasks.Task":
		return "return System.Threading.Tasks.Task.CompletedTask;"
	default:
		return "return null;"
	}
}

func cppUntranslatedReturn(behavior Behavior) string {
	returnType := "void"
	if behavior.TargetABI != nil && behavior.TargetABI.ReturnType != "" {
		returnType = behavior.TargetABI.ReturnType
	}
	switch returnType {
	case "void":
		return ""
	case "bool":
		return "return false;"
	case "std::chrono::milliseconds":
		return "return std::chrono::milliseconds{0};"
	case "std::chrono::system_clock::time_point":
		return "return std::chrono::system_clock::time_point{};"
	default:
		return "return {};"
	}
}

func dartUntranslatedReturn(behavior Behavior) string {
	returnType := "void"
	if behavior.TargetABI != nil && behavior.TargetABI.ReturnType != "" {
		returnType = behavior.TargetABI.ReturnType
	}
	switch returnType {
	case "void", "FutureOr<void>":
		return ""
	case "bool":
		return "return false;"
	case "Duration", "FutureOr<Duration>":
		return "return Duration.zero;"
	case "DateTime", "FutureOr<DateTime>":
		return "return DateTime.fromMillisecondsSinceEpoch(0);"
	default:
		return "return null;"
	}
}

func jsUntranslatedReturn(behavior Behavior) string {
	if behavior.Kind == BehaviorGuard {
		return "return false;"
	}
	if behavior.Kind == BehaviorTrigger {
		switch behavior.TriggerKind {
		case TriggerAt:
			return "return new Date(0);"
		case TriggerAfter, TriggerEvery:
			return "return 0;"
		default:
			return "return undefined;"
		}
	}
	return ""
}

func tsUntranslatedReturn(behavior Behavior) string {
	returnType := "void"
	if behavior.TargetABI != nil && behavior.TargetABI.ReturnType != "" {
		returnType = behavior.TargetABI.ReturnType
	}
	switch returnType {
	case "void":
		return ""
	case "boolean":
		return "return false;"
	case "number", "number | Date | Promise<number> | Promise<Date>":
		return "return 0;"
	case "Date | Promise<Date>":
		return "return new Date(0);"
	default:
		return "return undefined;"
	}
}

func pythonUntranslatedReturn(behavior Behavior) string {
	returnType := "None"
	if behavior.TargetABI != nil && behavior.TargetABI.ReturnType != "" {
		returnType = behavior.TargetABI.ReturnType
	}
	switch returnType {
	case "None":
		return "pass"
	case "bool":
		return "return False"
	case "float":
		return "return 0.0"
	case "datetime.datetime":
		return "return datetime.datetime.fromtimestamp(0)"
	default:
		return "return None"
	}
}

func zigUntranslatedReturn(behavior Behavior) string {
	returnType := "void"
	if behavior.TargetABI != nil && behavior.TargetABI.ReturnType != "" {
		returnType = behavior.TargetABI.ReturnType
	}
	switch returnType {
	case "void":
		return ""
	case "bool":
		return "return false;"
	case "u64":
		return "return 0;"
	default:
		return "return undefined;"
	}
}
