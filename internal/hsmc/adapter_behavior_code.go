package hsmc

import (
	"fmt"
	"strings"
)

func validateAdapterBehaviorBodyBoundary(target Language, behaviorID string, body string) error {
	if !behaviorBodyUsesBraceScope(target) {
		return nil
	}
	if adapterBodyClosesCompilerOwnedScope(body) {
		return fmt.Errorf("adapter patch behavior %q body attempts to close the compiler-owned behavior scope; use code for complete callables and globals for helper declarations", behaviorID)
	}
	return nil
}

func behaviorBodyUsesBraceScope(target Language) bool {
	switch target {
	case LanguageCSharp, LanguageCPP, LanguageDart, LanguageGo, LanguageJava, LanguageJS, LanguageRust, LanguageTS, LanguageZig:
		return true
	default:
		return false
	}
}

func adapterBodyClosesCompilerOwnedScope(body string) bool {
	state := codeScanState{}
	for index := 0; index < len(body); index++ {
		if state.step(body, index) {
			continue
		}
		switch body[index] {
		case '}':
			if state.brace == 0 {
				return true
			}
			state.brace--
		case ')':
			if state.paren == 0 {
				return true
			}
			state.paren--
		case ']':
			if state.bracket == 0 {
				return true
			}
			state.bracket--
		default:
			state.trackOpeningDelimiter(body[index])
		}
	}
	return false
}

func validateAdapterBehaviorFullCodeBoundary(target Language, behavior Behavior, code string) error {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return nil
	}
	behaviorID := behavior.ID
	expectedName := behaviorSymbolFor(target, behaviorID)
	if declaredName, ok := targetTopLevelDeclarationName(target, trimmed); ok && expectedName != "" && declaredName != expectedName {
		return fmt.Errorf("%s behavior full code declares %q, want compiler-owned name %q", target, declaredName, expectedName)
	}
	if behaviorCodeRequiresDeclaration(target) {
		if _, ok := targetTopLevelDeclarationName(target, trimmed); !ok {
			return behaviorCodeDeclarationRequiredError(target, expectedName)
		}
	}
	if err := validateAdapterBehaviorFullCodeContract(target, expectedName, behavior, trimmed); err != nil {
		return err
	}
	if target == LanguagePython {
		if pythonLooksLikeFunctionDeclaration(trimmed) && !pythonFullCodeIsSupportedFunctionDeclaration(trimmed) {
			return fmt.Errorf("adapter patch behavior %q full %s code must be a supported compiler-owned function declaration", behaviorID, target)
		}
		if pythonExpressionFullCodeHasExtraTopLevel(trimmed) {
			return fmt.Errorf("adapter patch behavior %q full %s code contains extra top-level code after the compiler-owned callable", behaviorID, target)
		}
		if pythonFullCodeHasExtraTopLevel(trimmed) {
			return fmt.Errorf("adapter patch behavior %q full %s code contains extra top-level code after the compiler-owned callable", behaviorID, target)
		}
		return nil
	}
	if target == LanguageJS || target == LanguageTS {
		if esExpressionFullCodeHasExtraTopLevel(trimmed) {
			return fmt.Errorf("adapter patch behavior %q full %s code contains extra top-level code after the compiler-owned callable", behaviorID, target)
		}
	}
	if target == LanguageCSharp || target == LanguageDart {
		if callableExpressionFullCodeHasExtraTopLevel(target, trimmed) {
			return fmt.Errorf("adapter patch behavior %q full %s code contains extra top-level code after the compiler-owned callable", behaviorID, target)
		}
	}
	openBrace := firstTopLevelByte(trimmed, '{')
	if openBrace >= 0 {
		closeBrace := matchingCloseBrace(trimmed, openBrace)
		if closeBrace < 0 {
			return nil
		}
		rest := strings.TrimSpace(trimmed[closeBrace+1:])
		if strings.HasPrefix(rest, ";") {
			rest = strings.TrimSpace(rest[1:])
		}
		if !onlyIgnorableCode(rest) {
			return fmt.Errorf("adapter patch behavior %q full %s code contains extra top-level code after the compiler-owned callable", behaviorID, target)
		}
		return nil
	}
	if index := firstTopLevelSemicolon(trimmed); index >= 0 {
		if !onlyIgnorableCode(strings.TrimSpace(trimmed[index+1:])) {
			return fmt.Errorf("adapter patch behavior %q full %s code contains extra top-level code after the compiler-owned callable", behaviorID, target)
		}
	}
	return nil
}

func validateAdapterBehaviorFullCodeContract(target Language, expectedName string, behavior Behavior, code string) error {
	switch target {
	case LanguageCSharp:
		if _, ok := csharpMethodDeclarationName(code); ok {
			if returnType, _, parameters, ok := cLikeMethodDeclarationShape(code); ok {
				if wantReturnType := csharpReturnType(behavior); returnType != wantReturnType {
					return fmt.Errorf("csharp behavior full code declares return type %q, want compiler-owned return type %q", returnType, wantReturnType)
				}
				if wantParameters := csharpParameterList(behavior); parameters != wantParameters {
					return fmt.Errorf("csharp behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
				}
			}
			return nil
		}
		if _, ok := cLikeMethodPrototypeName(code); ok {
			return fmt.Errorf("csharp behavior full code for %q must be a compiler-owned callable declaration with a body", expectedName)
		}
		if _, ok := cLikeFieldDeclarationName(code); ok {
			parameters, ok := csharpLambdaParameterNames(csharpFieldInitializer(code))
			if !ok {
				return fmt.Errorf("csharp behavior full code for %q must be a compiler-owned callable declaration", expectedName)
			}
			return validateCSharpLambdaParameters(behavior, parameters)
		}
		if _, ok := csharpNonCallableDeclarationName(code); ok {
			return fmt.Errorf("csharp behavior full code for %q must be a compiler-owned callable declaration", expectedName)
		}
		if parameters, ok := csharpLambdaParameterNames(code); ok {
			return validateCSharpLambdaParameters(behavior, parameters)
		}
		if behavior.TargetLanguage == LanguageCSharp {
			return fmt.Errorf("csharp behavior full code for %q must be a compiler-owned callable expression", expectedName)
		}
	case LanguageCPP:
		if _, ok := cLikeMethodDeclarationName(code); ok {
			return validateCPPFunctionDeclarationShape(code, behavior)
		}
		if _, ok := cppNonCallableDeclarationName(code); ok {
			return fmt.Errorf("cpp behavior full code for %q must be a compiler-owned callable declaration, not a type or alias declaration", behavior.ID)
		}
		if _, ok := cppTopLevelDeclarationName(code); ok {
			return validateCPPLambdaShape(code, behavior)
		}
		if strings.HasPrefix(code, "[") {
			return validateCPPLambdaShape(code, behavior)
		}
		return fmt.Errorf("cpp behavior full code for %q must be a compiler-owned function definition, lambda declaration, or lambda expression; use body for statements", expectedName)
	case LanguageDart:
		_, _, err := dartFullCodeBehavior(expectedName, behavior, code)
		return err
	case LanguageGo:
		declarationSignature, hasDeclarationSignature := goFunctionDeclarationSignature(code, expectedName)
		literalSignature, hasLiteralSignature := goFunctionLiteralSignature(code)
		if _, ok := goTopLevelDeclarationName(code); ok && !hasDeclarationSignature && !hasLiteralSignature {
			return fmt.Errorf("go behavior full code for %q must be a compiler-owned callable declaration", expectedName)
		}
		if hasDeclarationSignature && declarationSignature != goABISuffix(behavior) {
			return fmt.Errorf("go behavior full code declares signature %q, want compiler-owned signature %q", declarationSignature, goABISuffix(behavior))
		}
		if hasLiteralSignature && literalSignature != goABISuffix(behavior) {
			return fmt.Errorf("go behavior full code declares function literal signature %q, want compiler-owned signature %q", literalSignature, goABISuffix(behavior))
		}
	case LanguageJava:
		if returnType, _, parameters, ok := cLikeMethodDeclarationShape(code); ok {
			if wantReturnType := javaReturnType(behavior); returnType != wantReturnType {
				return fmt.Errorf("java behavior full code declares return type %q, want compiler-owned return type %q", returnType, wantReturnType)
			}
			const wantParameters = "(Context ctx, Instance instance, Event event)"
			if parameters != wantParameters {
				return fmt.Errorf("java behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
			}
			return nil
		}
		if _, ok := cLikeMethodPrototypeName(code); ok {
			return fmt.Errorf("java behavior code for %q must be a compiler-owned method declaration with a body", expectedName)
		}
		if _, ok := targetTopLevelDeclarationName(LanguageJava, code); ok {
			return fmt.Errorf("java behavior code for %q must be a compiler-owned method declaration; use body for method statements", expectedName)
		}
	case LanguageJS:
		if _, ok := esTopLevelDeclarationName(code); ok {
			callable := false
			if _, parameters, _, ok := esFunctionDeclarationShape(code); ok {
				callable = true
				if wantParameters := jsParameterList(); parameters != wantParameters {
					return fmt.Errorf("javascript behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
				}
			}
			if _, ok := esFunctionPrototypeName(code); ok {
				return fmt.Errorf("javascript behavior full code for %q must be a compiler-owned callable declaration with a body", expectedName)
			}
			if parameters, _, ok := esFunctionExpressionShape(code); ok {
				callable = true
				if wantParameters := jsParameterList(); parameters != wantParameters {
					return fmt.Errorf("javascript behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
				}
			}
			if !callable {
				return fmt.Errorf("javascript behavior full code for %q must be a compiler-owned callable declaration", expectedName)
			}
			return nil
		}
		parameters, _, ok := esFunctionExpressionShape(code)
		if !ok {
			if behavior.TargetLanguage == LanguageJS {
				return fmt.Errorf("javascript behavior full code for %q must be a compiler-owned callable expression", expectedName)
			}
			return nil
		}
		if wantParameters := jsParameterList(); parameters != wantParameters {
			return fmt.Errorf("javascript behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
		}
	case LanguagePython:
		if _, ok := pythonFunctionDeclarationName(code); ok {
			if async, _, parameters, returnType, ok := pythonFunctionDeclarationShape(code); ok {
				if !async {
					return fmt.Errorf("python behavior full code declares synchronous function %q, want compiler-owned async function", expectedName)
				}
				if wantParameters := pythonParameterList(); parameters != wantParameters {
					return fmt.Errorf("python behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
				}
				if wantReturnType := pythonReturnType(behavior); returnType != wantReturnType {
					return fmt.Errorf("python behavior full code declares return type %q, want compiler-owned return type %q", returnType, wantReturnType)
				}
			}
			return nil
		}
		parameters, isLambda, ok := pythonLambdaParameterList(code)
		if !isLambda {
			if behavior.TargetLanguage == LanguagePython {
				return fmt.Errorf("python behavior full code for %q must be a compiler-owned callable expression", expectedName)
			}
			return nil
		}
		if !ok {
			return fmt.Errorf("python behavior full code declares unsupported lambda signature, want compiler-owned parameters %q", pythonExpressionParameterList())
		}
		if wantParameters := pythonExpressionParameterList(); parameters != wantParameters {
			return fmt.Errorf("python behavior full code declares lambda parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
		}
	case LanguageRust:
		if _, parameters, returnType, ok := rustFunctionDeclarationShape(code); ok {
			if wantParameters := rustParameterList(behavior); parameters != wantParameters {
				return fmt.Errorf("rust behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
			}
			if wantReturnType := rustFullCodeReturnType(behavior); returnType != wantReturnType {
				return fmt.Errorf("rust behavior full code declares return type %q, want compiler-owned return type %q", returnType, wantReturnType)
			}
			return nil
		}
		if _, ok := targetTopLevelDeclarationName(LanguageRust, code); ok {
			return fmt.Errorf("rust behavior code for %q must be a compiler-owned function declaration; use body for function statements", expectedName)
		}
	case LanguageTS:
		if _, ok := esTopLevelDeclarationName(code); ok {
			callable := false
			if _, parameters, returnType, ok := esFunctionDeclarationShape(code); ok {
				callable = true
				if wantParameters := tsParameterList(); parameters != wantParameters {
					return fmt.Errorf("typescript behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
				}
				if wantReturnType := tsReturnType(behavior); returnType != wantReturnType {
					return fmt.Errorf("typescript behavior full code declares return type %q, want compiler-owned return type %q", returnType, wantReturnType)
				}
			}
			if _, ok := esFunctionPrototypeName(code); ok {
				return fmt.Errorf("typescript behavior full code for %q must be a compiler-owned callable declaration with a body", expectedName)
			}
			if parameters, returnType, ok := esFunctionExpressionShape(code); ok {
				callable = true
				if wantParameters := tsParameterList(); parameters != wantParameters {
					return fmt.Errorf("typescript behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
				}
				if returnType != "" {
					if wantReturnType := tsReturnType(behavior); returnType != wantReturnType {
						return fmt.Errorf("typescript behavior full code declares return type %q, want compiler-owned return type %q", returnType, wantReturnType)
					}
				}
			}
			if !callable {
				return fmt.Errorf("typescript behavior full code for %q must be a compiler-owned callable declaration", expectedName)
			}
			return nil
		}
		parameters, returnType, ok := esFunctionExpressionShape(code)
		if !ok {
			if behavior.TargetLanguage == LanguageTS {
				return fmt.Errorf("typescript behavior full code for %q must be a compiler-owned callable expression", expectedName)
			}
			return nil
		}
		if wantParameters := tsParameterList(); parameters != wantParameters {
			return fmt.Errorf("typescript behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
		}
		if returnType != "" {
			if wantReturnType := tsReturnType(behavior); returnType != wantReturnType {
				return fmt.Errorf("typescript behavior full code declares return type %q, want compiler-owned return type %q", returnType, wantReturnType)
			}
		}
	case LanguageZig:
		if _, parameters, returnType, ok := zigFunctionDeclarationShape(code); ok {
			if wantParameters := zigParameterList(); parameters != wantParameters {
				return fmt.Errorf("zig behavior full code declares parameters %q, want compiler-owned parameters %q", parameters, wantParameters)
			}
			if wantReturnType := zigReturnType(behavior); returnType != wantReturnType {
				return fmt.Errorf("zig behavior full code declares return type %q, want compiler-owned return type %q", returnType, wantReturnType)
			}
			return nil
		}
		if _, ok := targetTopLevelDeclarationName(LanguageZig, code); ok {
			return fmt.Errorf("zig behavior code for %q must be a compiler-owned function declaration; use body for function statements", expectedName)
		}
	}
	return nil
}

func pythonLooksLikeFunctionDeclaration(code string) bool {
	trimmed := strings.TrimSpace(code)
	return strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "async def ")
}

func pythonFullCodeIsSupportedFunctionDeclaration(code string) bool {
	_, _, _, _, ok := pythonFunctionDeclarationShape(code)
	return ok
}

func behaviorCodeRequiresDeclaration(target Language) bool {
	switch target {
	case LanguageJava, LanguageRust, LanguageZig:
		return true
	default:
		return false
	}
}

func behaviorCodeDeclarationRequiredError(target Language, expectedName string) error {
	switch target {
	case LanguageJava:
		return fmt.Errorf("java behavior code for %q must be a compiler-owned method declaration; use body for method statements", expectedName)
	case LanguageRust:
		return fmt.Errorf("rust behavior code for %q must be a compiler-owned function declaration; use body for function statements", expectedName)
	case LanguageZig:
		return fmt.Errorf("zig behavior code for %q must be a compiler-owned function declaration; use body for function statements", expectedName)
	default:
		return fmt.Errorf("%s behavior code for %q must be a compiler-owned declaration", target, expectedName)
	}
}

func pythonFullCodeHasExtraTopLevel(code string) bool {
	lines := strings.Split(code, "\n")
	if len(lines) <= 1 {
		return false
	}
	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		return true
	}
	return false
}

func esExpressionFullCodeHasExtraTopLevel(code string) bool {
	if _, _, ok := esFunctionExpressionShape(code); !ok {
		return false
	}
	arrow := firstTopLevelArrow(code)
	if arrow < 0 {
		return false
	}
	afterArrow := code[arrow+len("=>"):]
	newline := firstTopLevelNewline(afterArrow)
	if newline < 0 {
		return false
	}
	if openBrace := firstTopLevelByte(code, '{'); openBrace >= 0 && openBrace < arrow+len("=>")+newline {
		return false
	}
	return true
}

func callableExpressionFullCodeHasExtraTopLevel(target Language, code string) bool {
	switch target {
	case LanguageCSharp, LanguageDart:
	default:
		return false
	}
	arrow := firstTopLevelArrow(code)
	if arrow < 0 {
		return false
	}
	afterArrow := code[arrow+len("=>"):]
	newline := firstTopLevelNewline(afterArrow)
	if newline < 0 {
		return false
	}
	if openBrace := firstTopLevelByte(code, '{'); openBrace >= 0 && openBrace < arrow+len("=>")+newline {
		return false
	}
	return true
}

func firstTopLevelArrow(code string) int {
	state := codeScanState{}
	for index := 0; index+1 < len(code); index++ {
		if state.step(code, index) {
			continue
		}
		if state.depth() == 0 && code[index] == '=' && code[index+1] == '>' {
			return index
		}
		state.trackDelimiter(code[index])
	}
	return -1
}

func firstTopLevelNewline(code string) int {
	state := codeScanState{}
	for index := 0; index < len(code); index++ {
		if state.step(code, index) {
			continue
		}
		if state.depth() == 0 && code[index] == '\n' {
			return index
		}
		state.trackDelimiter(code[index])
	}
	return -1
}

func pythonExpressionFullCodeHasExtraTopLevel(code string) bool {
	if _, isLambda, _ := pythonLambdaParameterList(code); !isLambda {
		return false
	}
	return firstTopLevelSemicolon(code) >= 0
}

func firstTopLevelSemicolon(code string) int {
	return firstTopLevelByte(code, ';')
}

func firstTopLevelByte(code string, want byte) int {
	state := codeScanState{}
	for index := 0; index < len(code); index++ {
		if state.step(code, index) {
			continue
		}
		if state.depth() == 0 && code[index] == want {
			return index
		}
		state.trackDelimiter(code[index])
	}
	return -1
}

func matchingCloseBrace(code string, open int) int {
	state := codeScanState{}
	depth := 0
	for index := 0; index < len(code); index++ {
		if state.step(code, index) {
			continue
		}
		switch code[index] {
		case '{':
			depth++
		case '}':
			depth--
			if index > open && depth == 0 {
				return index
			}
		}
		state.trackDelimiterExceptBrace(code[index])
	}
	return -1
}

func onlyIgnorableCode(code string) bool {
	code = strings.TrimSpace(code)
	for code != "" {
		switch {
		case strings.HasPrefix(code, "//"):
			newline := strings.IndexByte(code, '\n')
			if newline < 0 {
				return true
			}
			code = strings.TrimSpace(code[newline+1:])
		case strings.HasPrefix(code, "#"):
			newline := strings.IndexByte(code, '\n')
			if newline < 0 {
				return true
			}
			code = strings.TrimSpace(code[newline+1:])
		case strings.HasPrefix(code, "/*"):
			end := strings.Index(code, "*/")
			if end < 0 {
				return false
			}
			code = strings.TrimSpace(code[end+len("*/"):])
		default:
			return false
		}
	}
	return true
}

type codeScanState struct {
	paren        int
	bracket      int
	brace        int
	stringQuote  byte
	lineComment  bool
	blockComment bool
	escape       bool
}

func (state *codeScanState) depth() int {
	return state.paren + state.bracket + state.brace
}

func (state *codeScanState) step(code string, index int) bool {
	ch := code[index]
	if state.lineComment {
		if ch == '\n' {
			state.lineComment = false
		}
		return true
	}
	if state.blockComment {
		if ch == '/' && index > 0 && code[index-1] == '*' {
			state.blockComment = false
		}
		return true
	}
	if state.stringQuote != 0 {
		if state.escape {
			state.escape = false
			return true
		}
		if ch == '\\' {
			state.escape = true
			return true
		}
		if ch == state.stringQuote {
			state.stringQuote = 0
		}
		return true
	}
	if ch == '/' && index+1 < len(code) && code[index+1] == '/' {
		state.lineComment = true
		return true
	}
	if ch == '/' && index+1 < len(code) && code[index+1] == '*' {
		state.blockComment = true
		return true
	}
	if ch == '"' || ch == '\'' || ch == '`' {
		state.stringQuote = ch
		return true
	}
	return false
}

func (state *codeScanState) trackDelimiter(ch byte) {
	switch ch {
	case '(':
		state.paren++
	case ')':
		if state.paren > 0 {
			state.paren--
		}
	case '[':
		state.bracket++
	case ']':
		if state.bracket > 0 {
			state.bracket--
		}
	case '{':
		state.brace++
	case '}':
		if state.brace > 0 {
			state.brace--
		}
	}
}

func (state *codeScanState) trackOpeningDelimiter(ch byte) {
	switch ch {
	case '(':
		state.paren++
	case '[':
		state.bracket++
	case '{':
		state.brace++
	}
}

func (state *codeScanState) trackDelimiterExceptBrace(ch byte) {
	switch ch {
	case '(':
		state.paren++
	case ')':
		if state.paren > 0 {
			state.paren--
		}
	case '[':
		state.bracket++
	case ']':
		if state.bracket > 0 {
			state.bracket--
		}
	}
}

func (state *codeScanState) trackDelimiterExceptBracket(ch byte) {
	switch ch {
	case '(':
		state.paren++
	case ')':
		if state.paren > 0 {
			state.paren--
		}
	case '{':
		state.brace++
	case '}':
		if state.brace > 0 {
			state.brace--
		}
	}
}
