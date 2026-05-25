package hsmc

import "strings"

func javaMethodDeclarationName(code string) (string, bool) {
	return cLikeMethodDeclarationName(code)
}

func javaImportDeclarationName(code string) (string, bool) {
	trimmed := strings.TrimSpace(strings.TrimSuffix(code, ";"))
	if !strings.HasPrefix(trimmed, "import ") {
		return "", false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "import "))
	if strings.HasPrefix(rest, "static ") {
		rest = strings.TrimSpace(strings.TrimPrefix(rest, "static "))
	}
	if rest == "" || strings.HasSuffix(rest, ".*") {
		return "", false
	}
	for _, name := range javaImportLocalNames(rest) {
		if isSimpleIdentifier(name) {
			return name, true
		}
	}
	return "", false
}

func csharpMethodDeclarationName(code string) (string, bool) {
	return cLikeMethodDeclarationName(code)
}

func csharpUsingAliasDeclarationName(code string) (string, bool) {
	trimmed := strings.TrimSpace(strings.TrimSuffix(code, ";"))
	trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "global "))
	if !strings.HasPrefix(trimmed, "using ") {
		return "", false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "using "))
	if strings.HasPrefix(rest, "static ") {
		return "", false
	}
	left, _, ok := strings.Cut(rest, "=")
	if !ok {
		return "", false
	}
	name := strings.TrimSpace(left)
	if !isSimpleIdentifier(name) {
		return "", false
	}
	return name, true
}

func csharpExternAliasDeclarationName(code string) (string, bool) {
	trimmed := strings.TrimSpace(strings.TrimSuffix(code, ";"))
	if !strings.HasPrefix(trimmed, "extern alias ") {
		return "", false
	}
	name := strings.TrimSpace(strings.TrimPrefix(trimmed, "extern alias "))
	if !isSimpleIdentifier(name) {
		return "", false
	}
	return name, true
}

func csharpDelegateDeclarationName(code string) (string, bool) {
	code = stripLeadingCMetadata(code)
	fields := strings.Fields(strings.TrimSpace(code))
	for index, field := range fields {
		if field != "delegate" {
			continue
		}
		if index+2 >= len(fields) {
			return "", false
		}
		if !cLikeTypeDeclarationPrefix(fields[:index]) {
			return "", false
		}
		name := trimDeclarationName(fields[index+2])
		if !isSimpleIdentifier(name) {
			return "", false
		}
		return name, true
	}
	return "", false
}

func csharpPropertyDeclarationName(code string) (string, bool) {
	code = stripLeadingCMetadata(code)
	openBrace := strings.Index(code, "{")
	arrow := strings.Index(code, "=>")
	end := -1
	switch {
	case openBrace >= 0 && arrow >= 0:
		end = minInt(openBrace, arrow)
	case openBrace >= 0:
		end = openBrace
	case arrow >= 0:
		end = arrow
	default:
		return "", false
	}
	prefix := strings.TrimSpace(code[:end])
	if prefix == "" || containsDeclarationBoundary(prefix) {
		return "", false
	}
	fields := strings.Fields(prefix)
	if len(fields) < 2 {
		return "", false
	}
	for _, field := range fields[:len(fields)-2] {
		if !cLikeDeclarationModifier(field) {
			return "", false
		}
	}
	name := trimDeclarationName(fields[len(fields)-1])
	if !isSimpleIdentifier(name) {
		return "", false
	}
	return name, true
}

func cLikeMethodDeclarationName(code string) (string, bool) {
	_, name, _, ok := cLikeMethodDeclarationShape(code)
	return name, ok
}

func cLikeMethodPrototypeName(code string) (string, bool) {
	_, name, _, ok := cLikeMethodPrototypeShape(code)
	return name, ok
}

func cLikeConstructorDeclarationName(code string) (string, bool) {
	code = stripLeadingCMetadata(code)
	paren := strings.Index(code, "(")
	body := strings.Index(code, "{")
	if paren < 0 || body < 0 || body < paren {
		return "", false
	}
	prefix := strings.TrimSpace(code[:paren])
	if containsDeclarationBoundary(prefix) {
		return "", false
	}
	fields := strings.Fields(prefix)
	if len(fields) == 0 {
		return "", false
	}
	for _, field := range fields[:len(fields)-1] {
		if !cLikeDeclarationModifier(field) {
			return "", false
		}
	}
	name := fields[len(fields)-1]
	if !isSimpleIdentifier(name) {
		return "", false
	}
	parameters := strings.TrimSpace(code[paren:body])
	if parameters == "" || strings.ContainsAny(parameters, "{};=") {
		return "", false
	}
	return name, true
}

func stripLeadingCMetadata(code string) string {
	for {
		trimmed := strings.TrimSpace(code)
		switch {
		case strings.HasPrefix(trimmed, "["):
			close := strings.Index(trimmed, "]")
			if close < 0 {
				return trimmed
			}
			code = trimmed[close+1:]
			continue
		case strings.HasPrefix(trimmed, "@"):
			space := strings.IndexAny(trimmed, " \t\r\n")
			paren := strings.Index(trimmed, "(")
			if paren >= 0 && (space < 0 || paren < space) {
				closeParen := matchingCloseParen(trimmed[paren:])
				if closeParen < 0 {
					return trimmed
				}
				code = trimmed[paren+closeParen+1:]
				continue
			}
			if space < 0 {
				return ""
			}
			code = trimmed[space+1:]
			continue
		default:
			return trimmed
		}
	}
}

func cLikeDeclarationModifier(field string) bool {
	switch field {
	case "public", "private", "protected", "internal", "static", "final", "abstract", "sealed", "partial", "extern", "unsafe":
		return true
	default:
		return false
	}
}

func cLikeFieldDeclarationName(code string) (string, bool) {
	code = stripLeadingCMetadata(code)
	if code == "" {
		return "", false
	}
	end := strings.Index(code, ";")
	if end < 0 {
		end = len(code)
	}
	statement := strings.TrimSpace(code[:end])
	if statement == "" {
		return "", false
	}
	if eq := strings.Index(statement, "="); eq >= 0 {
		statement = strings.TrimSpace(statement[:eq])
	}
	if paren := strings.Index(statement, "("); paren >= 0 {
		return "", false
	}
	fields := strings.Fields(statement)
	if len(fields) < 2 {
		return "", false
	}
	name := trimCLikeDeclaratorName(fields[len(fields)-1])
	if !isSimpleIdentifier(name) {
		return "", false
	}
	return name, true
}

func trimCLikeDeclaratorName(name string) string {
	name = strings.TrimSpace(name)
	for strings.HasSuffix(name, "[]") {
		name = strings.TrimSpace(strings.TrimSuffix(name, "[]"))
	}
	return name
}

func cLikeTypeDeclarationName(code string) (string, bool) {
	code = stripLeadingCMetadata(code)
	fields := strings.Fields(strings.TrimSpace(code))
	for index, field := range fields {
		if field != "class" && field != "struct" && field != "interface" && field != "@interface" && field != "enum" && field != "record" {
			continue
		}
		nameIndex := index + 1
		if field == "record" && nameIndex < len(fields) && (fields[nameIndex] == "class" || fields[nameIndex] == "struct") {
			nameIndex++
		}
		if nameIndex >= len(fields) {
			return "", false
		}
		if !cLikeTypeDeclarationPrefix(fields[:index]) {
			return "", false
		}
		name := trimDeclarationName(fields[nameIndex])
		if !isSimpleIdentifier(name) {
			return "", false
		}
		return name, true
	}
	return "", false
}

func cLikeMethodDeclarationShape(code string) (returnType string, name string, parameters string, ok bool) {
	code = stripLeadingCMetadata(code)
	paren := strings.Index(code, "(")
	body := strings.Index(code, "{")
	if paren < 0 || body < 0 || body < paren {
		return "", "", "", false
	}
	prefix := strings.TrimSpace(code[:paren])
	if containsDeclarationBoundary(prefix) {
		return "", "", "", false
	}
	fields := strings.Fields(prefix)
	if len(fields) < 2 {
		return "", "", "", false
	}
	name = fields[len(fields)-1]
	if !isSimpleIdentifier(name) {
		return "", "", "", false
	}
	returnType = fields[len(fields)-2]
	parameters = strings.TrimSpace(code[paren:body])
	if parameters == "" || strings.ContainsAny(parameters, "{};=") {
		return "", "", "", false
	}
	return returnType, name, parameters, true
}

func cLikeMethodPrototypeShape(code string) (returnType string, name string, parameters string, ok bool) {
	code = stripLeadingCMetadata(code)
	paren := strings.Index(code, "(")
	semicolon := firstTopLevelByte(code, ';')
	if paren < 0 || semicolon < 0 || semicolon < paren {
		return "", "", "", false
	}
	if body := firstTopLevelByte(code, '{'); body >= 0 && body < semicolon {
		return "", "", "", false
	}
	closeParen := matchingCloseParen(code[paren:semicolon])
	if closeParen < 0 {
		return "", "", "", false
	}
	closeParen += paren
	if strings.TrimSpace(code[closeParen+1:semicolon]) != "" {
		return "", "", "", false
	}
	prefix := strings.TrimSpace(code[:paren])
	if containsDeclarationBoundary(prefix) {
		return "", "", "", false
	}
	fields := strings.Fields(prefix)
	if len(fields) < 2 {
		return "", "", "", false
	}
	name = fields[len(fields)-1]
	if !isSimpleIdentifier(name) {
		return "", "", "", false
	}
	returnType = fields[len(fields)-2]
	parameters = strings.TrimSpace(code[paren : closeParen+1])
	if parameters == "" || strings.ContainsAny(parameters, "{};=") {
		return "", "", "", false
	}
	return returnType, name, parameters, true
}

func pythonFunctionDeclarationName(code string) (string, bool) {
	_, name, _, _, ok := pythonFunctionDeclarationShape(code)
	return name, ok
}

func pythonTopLevelDeclarationName(code string) (string, bool) {
	if names := pythonImportDeclarationNames(code); len(names) > 0 {
		return names[0], true
	}
	if name, ok := pythonFunctionDeclarationName(code); ok {
		return name, true
	}
	if name, ok := pythonClassDeclarationName(code); ok {
		return name, true
	}
	return pythonAssignmentDeclarationName(code)
}

func pythonImportDeclarationNames(code string) []string {
	if names := pythonParenthesizedFromImportDeclarationNames(code); len(names) > 0 {
		return names
	}
	line := strings.TrimSpace(strings.SplitN(code, "\n", 2)[0])
	if strings.HasPrefix(line, "import ") {
		return pythonImportListBindingNames(strings.TrimSpace(strings.TrimPrefix(line, "import ")), true)
	}
	if strings.HasPrefix(line, "from ") {
		_, imports, ok := strings.Cut(strings.TrimPrefix(line, "from "), " import ")
		if !ok {
			return nil
		}
		return pythonImportListBindingNames(strings.TrimSpace(imports), false)
	}
	return nil
}

func pythonParenthesizedFromImportDeclarationNames(code string) []string {
	trimmed := strings.TrimSpace(code)
	if !strings.HasPrefix(trimmed, "from ") {
		return nil
	}
	_, imports, ok := strings.Cut(strings.TrimPrefix(trimmed, "from "), " import ")
	if !ok {
		return nil
	}
	imports = strings.TrimSpace(imports)
	if !strings.HasPrefix(imports, "(") {
		return nil
	}
	closeParen := matchingCloseParen(imports)
	if closeParen < 0 {
		return nil
	}
	return pythonImportListBindingNames(imports[1:closeParen], false)
}

func pythonImportListBindingNames(imports string, moduleImport bool) []string {
	imports = strings.TrimSpace(strings.Trim(imports, "()"))
	if imports == "" || imports == "*" {
		return nil
	}
	var names []string
	for _, segment := range splitTopLevelCommaSegments(imports) {
		segment = strings.TrimSpace(segment)
		if segment == "" || segment == "*" {
			continue
		}
		fields := strings.Fields(segment)
		if len(fields) >= 3 && fields[len(fields)-2] == "as" {
			name := trimDeclarationName(fields[len(fields)-1])
			if isSimpleIdentifier(name) {
				names = append(names, name)
			}
			continue
		}
		name := trimDeclarationName(fields[0])
		if moduleImport {
			if dot := strings.Index(name, "."); dot >= 0 {
				name = name[:dot]
			}
		}
		if isSimpleIdentifier(name) {
			names = append(names, name)
		}
	}
	return names
}

func pythonClassDeclarationName(code string) (string, bool) {
	trimmed := strings.TrimSpace(code)
	if !strings.HasPrefix(trimmed, "class ") {
		return "", false
	}
	remaining := strings.TrimSpace(strings.TrimPrefix(trimmed, "class "))
	if remaining == "" {
		return "", false
	}
	fields := strings.Fields(remaining)
	if len(fields) == 0 {
		return "", false
	}
	name := trimDeclarationName(fields[0])
	if !isSimpleIdentifier(name) {
		return "", false
	}
	return name, true
}

func pythonAssignmentDeclarationName(code string) (string, bool) {
	names := pythonAssignmentDeclarationNames(code)
	if len(names) == 0 {
		return "", false
	}
	return names[0], true
}

func pythonAssignmentDeclarationNames(code string) []string {
	line := strings.TrimSpace(strings.SplitN(code, "\n", 2)[0])
	if line == "" || strings.HasPrefix(line, "#") {
		return nil
	}
	for _, prefix := range []string{"def ", "async def ", "class ", "import ", "from ", "if ", "while ", "for ", "with ", "try", "except ", "finally", "else:", "elif "} {
		if strings.HasPrefix(line, prefix) {
			return nil
		}
	}
	var names []string
	start := 0
	state := codeScanState{}
	for index := 0; index < len(line); index++ {
		if state.step(line, index) {
			continue
		}
		if state.depth() == 0 && line[index] == '=' && !pythonAssignmentOperatorIsNotBinding(line, index) {
			names = append(names, pythonAssignmentBindingNames(line[start:index])...)
			start = index + 1
		}
		state.trackDelimiter(line[index])
	}
	return names
}

func pythonAssignmentBindingNames(left string) []string {
	left = strings.TrimSpace(left)
	if colon := strings.Index(left, ":"); colon >= 0 {
		left = strings.TrimSpace(left[:colon])
	}
	if left == "" {
		return nil
	}
	segments := splitTopLevelCommaSegments(strings.Trim(left, "()[]"))
	names := make([]string, 0, len(segments))
	for _, segment := range segments {
		name := strings.TrimSpace(segment)
		if !isSimpleIdentifier(name) {
			return nil
		}
		names = append(names, name)
	}
	return names
}

func pythonAssignmentOperatorIsNotBinding(line string, eq int) bool {
	if eq > 0 {
		switch line[eq-1] {
		case '=', '!', '<', '>', '+', '-', '*', '/', '%', '@', '&', '|', '^', '~', ':':
			return true
		}
	}
	if eq+1 < len(line) && line[eq+1] == '=' {
		return true
	}
	return false
}

func pythonFunctionDeclarationShape(code string) (async bool, name string, parameters string, returnType string, ok bool) {
	fields := strings.Fields(strings.TrimSpace(code))
	if len(fields) < 2 {
		return false, "", "", "", false
	}
	nameField := ""
	switch fields[0] {
	case "def":
		nameField = fields[1]
	case "async":
		if len(fields) < 3 || fields[1] != "def" {
			return false, "", "", "", false
		}
		async = true
		nameField = fields[2]
	default:
		return false, "", "", "", false
	}
	if paren := strings.Index(nameField, "("); paren >= 0 {
		nameField = nameField[:paren]
	}
	if !isSimpleIdentifier(nameField) {
		return false, "", "", "", false
	}
	name = nameField
	firstLine := strings.TrimSpace(strings.SplitN(strings.TrimSpace(code), "\n", 2)[0])
	paren := strings.Index(firstLine, "(")
	if paren < 0 {
		return false, "", "", "", false
	}
	closeParen := matchingCloseParen(firstLine[paren:])
	if closeParen < 0 {
		return false, "", "", "", false
	}
	closeParen += paren
	parameters = strings.TrimSpace(firstLine[paren : closeParen+1])
	trailer := strings.TrimSpace(firstLine[closeParen+1:])
	if !strings.HasSuffix(trailer, ":") {
		return false, "", "", "", false
	}
	trailer = strings.TrimSpace(strings.TrimSuffix(trailer, ":"))
	returnType = "None"
	if strings.HasPrefix(trailer, "->") {
		returnType = strings.TrimSpace(strings.TrimPrefix(trailer, "->"))
	}
	return async, name, parameters, returnType, true
}

func pythonLambdaParameterList(code string) (string, bool, bool) {
	code = strings.TrimSpace(code)
	if !strings.HasPrefix(code, "lambda ") {
		return "", false, false
	}
	remaining := strings.TrimSpace(strings.TrimPrefix(code, "lambda "))
	colon := pythonLambdaParameterColon(remaining)
	if colon < 0 {
		return "", true, false
	}
	parameters := strings.TrimSpace(remaining[:colon])
	if parameters == "" {
		return "", true, false
	}
	names := strings.Split(parameters, ",")
	for index, name := range names {
		name = strings.TrimSpace(name)
		if !isSimpleIdentifier(name) {
			return "", true, false
		}
		names[index] = name
	}
	return "(" + strings.Join(names, ", ") + ")", true, true
}

func pythonLambdaParameterColon(value string) int {
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	var quote rune
	escaped := false
	for index, r := range value {
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == quote {
				quote = 0
			}
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		case ':':
			if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 {
				return index
			}
		}
	}
	return -1
}

func goTopLevelDeclarationName(code string) (string, bool) {
	if names := goImportDeclarationNames(code); len(names) > 0 {
		return names[0], true
	}
	fields := strings.Fields(strings.TrimSpace(code))
	if len(fields) < 2 {
		return "", false
	}
	switch fields[0] {
	case "func":
		return esIdentifierBeforeParen(fields[1])
	case "var", "const", "type":
		name := fields[1]
		if eq := strings.Index(name, "="); eq >= 0 {
			name = name[:eq]
		}
		if fields[0] == "type" {
			name = trimDeclarationName(name)
		}
		if !isSimpleIdentifier(name) {
			return "", false
		}
		return name, true
	default:
		return "", false
	}
}

func goImportDeclarationNames(code string) []string {
	trimmed := strings.TrimSpace(code)
	if !strings.HasPrefix(trimmed, "import") {
		return nil
	}
	rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "import"))
	if rest == "" {
		return nil
	}
	if strings.HasPrefix(rest, "(") {
		close := matchingCloseParen(rest)
		if close < 0 {
			return nil
		}
		var names []string
		for _, line := range strings.Split(rest[1:close], "\n") {
			if name := goImportSpecBindingName(line); name != "" {
				names = append(names, name)
			}
		}
		return names
	}
	if name := goImportSpecBindingName(strings.SplitN(rest, "\n", 2)[0]); name != "" {
		return []string{name}
	}
	return nil
}

func goImportSpecBindingName(spec string) string {
	spec = strings.TrimSpace(spec)
	if spec == "" || strings.HasPrefix(spec, "//") {
		return ""
	}
	fields := strings.Fields(spec)
	if len(fields) == 0 {
		return ""
	}
	alias := ""
	importPath := fields[0]
	if len(fields) >= 2 {
		alias = fields[0]
		importPath = fields[1]
	}
	if alias != "" {
		if !isSimpleIdentifier(alias) {
			return ""
		}
		return alias
	}
	importPath = strings.Trim(importPath, "`\"")
	if importPath == "" {
		return ""
	}
	for _, name := range goImportLocalNames(importPath, "") {
		if isSimpleIdentifier(name) {
			return name
		}
	}
	return ""
}

func goFunctionDeclarationSignature(code string, name string) (string, bool) {
	code = strings.TrimSpace(code)
	prefix := "func " + name
	if !strings.HasPrefix(code, prefix) {
		return "", false
	}
	remaining := strings.TrimSpace(strings.TrimPrefix(code, prefix))
	if !strings.HasPrefix(remaining, "(") {
		return "", false
	}
	body := strings.Index(remaining, "{")
	if body < 0 {
		return "", false
	}
	signature := strings.TrimSpace(remaining[:body])
	if signature == "" || strings.ContainsAny(signature, "{};=") {
		return "", false
	}
	return signature, true
}

func goFunctionLiteralSignature(code string) (string, bool) {
	code = strings.TrimSpace(code)
	if declaredName, ok := goTopLevelDeclarationName(code); ok && declaredName != "" {
		if eq := firstTopLevelByte(code, '='); eq < 0 {
			return "", false
		} else {
			code = strings.TrimSpace(code[eq+1:])
		}
	}
	if !strings.HasPrefix(code, "func") {
		return "", false
	}
	remaining := strings.TrimSpace(strings.TrimPrefix(code, "func"))
	if !strings.HasPrefix(remaining, "(") {
		return "", false
	}
	body := strings.Index(remaining, "{")
	if body < 0 {
		return "", false
	}
	signature := strings.TrimSpace(remaining[:body])
	if signature == "" || strings.ContainsAny(signature, "{};=") {
		return "", false
	}
	return signature, true
}

func cppTopLevelDeclarationName(code string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(code))
	for index, field := range fields {
		if field != "auto" || index+1 >= len(fields) {
			continue
		}
		if !cppAutoDeclarationPrefix(fields[:index]) {
			return "", false
		}
		name := strings.TrimSpace(fields[index+1])
		if eq := strings.Index(name, "="); eq >= 0 {
			name = name[:eq]
		}
		if !isSimpleIdentifier(name) {
			return "", false
		}
		return name, true
	}
	return "", false
}

func cppTypedVariableDeclarationName(code string) (string, bool) {
	trimmed := strings.TrimSpace(code)
	fields := strings.Fields(trimmed)
	if len(fields) < 2 {
		return "", false
	}
	switch fields[0] {
	case "class", "enum", "namespace", "struct", "template", "typedef", "union", "using":
		return "", false
	}
	end := firstTopLevelByte(trimmed, ';')
	if end < 0 {
		end = len(trimmed)
	}
	statement := strings.TrimSpace(trimmed[:end])
	if statement == "" {
		return "", false
	}
	if eq := firstTopLevelByte(statement, '='); eq >= 0 {
		statement = strings.TrimSpace(statement[:eq])
	}
	if name, ok := cppFunctionPointerDeclaratorName(statement); ok {
		return name, true
	}
	if paren := firstTopLevelByte(statement, '('); paren >= 0 {
		return "", false
	}
	fields = strings.Fields(statement)
	if len(fields) < 2 {
		return "", false
	}
	name := trimCPPDeclaratorName(fields[len(fields)-1])
	if !isSimpleIdentifier(name) {
		return "", false
	}
	return name, true
}

func cppFunctionPointerDeclaratorName(statement string) (string, bool) {
	for _, marker := range []string{"(*", "(&"} {
		start := strings.Index(statement, marker)
		if start < 0 {
			continue
		}
		rest := strings.TrimSpace(statement[start+len(marker):])
		close := strings.Index(rest, ")")
		if close < 0 {
			return "", false
		}
		name := strings.TrimSpace(rest[:close])
		name = strings.TrimLeft(name, "*&")
		if isSimpleIdentifier(name) {
			return name, true
		}
	}
	return "", false
}

func trimCPPDeclaratorName(name string) string {
	name = trimDeclarationName(name)
	name = strings.TrimLeft(name, "*&")
	if bracket := strings.Index(name, "["); bracket >= 0 {
		name = name[:bracket]
	}
	return strings.TrimSpace(name)
}

func cppTypeDeclarationName(code string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(code))
	for index, field := range fields {
		switch field {
		case "struct", "class", "union":
			if index+1 >= len(fields) {
				return "", false
			}
			name := trimDeclarationName(fields[index+1])
			if !isSimpleIdentifier(name) {
				return "", false
			}
			return name, true
		case "enum":
			nameIndex := index + 1
			if nameIndex < len(fields) && (fields[nameIndex] == "class" || fields[nameIndex] == "struct") {
				nameIndex++
			}
			if nameIndex >= len(fields) {
				return "", false
			}
			name := trimDeclarationName(fields[nameIndex])
			if !isSimpleIdentifier(name) {
				return "", false
			}
			return name, true
		case "namespace":
			if index+1 >= len(fields) {
				return "", false
			}
			name := trimDeclarationName(fields[index+1])
			if !isSimpleIdentifier(name) {
				return "", false
			}
			return name, true
		}
	}
	return "", false
}

func cppUsingDeclarationName(code string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(code))
	if len(fields) < 3 || fields[0] != "using" {
		return "", false
	}
	name := trimDeclarationName(fields[1])
	if !isSimpleIdentifier(name) {
		return "", false
	}
	return name, true
}

func cppTypedefDeclarationName(code string) (string, bool) {
	trimmed := strings.TrimSpace(strings.TrimSuffix(code, ";"))
	if !strings.HasPrefix(trimmed, "typedef ") {
		return "", false
	}
	fields := strings.Fields(trimmed)
	if len(fields) < 3 {
		return "", false
	}
	name := trimDeclarationName(fields[len(fields)-1])
	if !isSimpleIdentifier(name) {
		return "", false
	}
	return name, true
}

func cppLambdaShape(code string) (parameters string, returnType string, ok bool) {
	code = strings.TrimSpace(code)
	lambda := strings.Index(code, "[]")
	if lambda < 0 {
		return "", "", false
	}
	remaining := strings.TrimSpace(code[lambda+len("[]"):])
	if !strings.HasPrefix(remaining, "(") {
		return "", "", false
	}
	closeParen := matchingCloseParen(remaining)
	if closeParen < 0 {
		return "", "", false
	}
	parameters = strings.TrimSpace(remaining[:closeParen+1])
	afterParams := strings.TrimSpace(remaining[closeParen+1:])
	body := strings.Index(afterParams, "{")
	if body < 0 {
		return "", "", false
	}
	returnType = strings.TrimSpace(afterParams[:body])
	if strings.HasPrefix(returnType, "->") {
		returnType = strings.TrimSpace(strings.TrimPrefix(returnType, "->"))
	}
	return parameters, returnType, true
}

func cppAutoDeclarationPrefix(fields []string) bool {
	for _, field := range fields {
		if field == "" {
			continue
		}
		if field == "static" || field == "constexpr" || field == "const" || field == "inline" {
			continue
		}
		return false
	}
	return true
}

func ensureTrailingSemicolonForCPPDeclaration(code string) string {
	code = strings.TrimSpace(code)
	if strings.HasSuffix(code, ";") {
		return code
	}
	return code + ";"
}

func esTopLevelDeclarationName(code string) (string, bool) {
	if names := esImportDeclarationNames(code); len(names) > 0 {
		return names[0], true
	}
	if names := esVariableDeclarationNames(code); len(names) > 0 {
		return names[0], true
	}
	fields := strings.Fields(strings.TrimSpace(code))
	if len(fields) < 2 {
		return "", false
	}
	if fields[0] == "export" {
		fields = fields[1:]
		if len(fields) < 2 {
			return "", false
		}
		if fields[0] == "default" {
			fields = fields[1:]
			if len(fields) < 2 {
				return "", false
			}
		}
	}
	for len(fields) >= 2 && (fields[0] == "declare" || fields[0] == "abstract") {
		fields = fields[1:]
	}
	if fields[0] == "async" && len(fields) >= 3 && fields[1] == "function" {
		return esIdentifierBeforeParen(fields[2])
	}
	if fields[0] == "function" {
		return esIdentifierBeforeParen(fields[1])
	}
	if fields[0] == "class" || fields[0] == "enum" || fields[0] == "interface" || fields[0] == "type" || fields[0] == "namespace" || fields[0] == "module" {
		name := trimDeclarationName(fields[1])
		if !isSimpleIdentifier(name) {
			return "", false
		}
		return name, true
	}
	return "", false
}

func esImportDeclarationNames(code string) []string {
	trimmed := strings.TrimSpace(strings.TrimSuffix(code, ";"))
	if !strings.HasPrefix(trimmed, "import ") {
		return nil
	}
	rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "import "))
	if strings.HasPrefix(rest, "type ") {
		rest = strings.TrimSpace(strings.TrimPrefix(rest, "type "))
	}
	if rest == "" || strings.HasPrefix(rest, "\"") || strings.HasPrefix(rest, "'") {
		return nil
	}
	from := strings.LastIndex(rest, " from ")
	if from < 0 {
		return nil
	}
	bindings := strings.TrimSpace(rest[:from])
	var names []string
	for _, segment := range splitTopLevelCommaSegments(bindings) {
		names = append(names, esImportBindingNames(segment)...)
	}
	return names
}

func esImportBindingNames(binding string) []string {
	binding = strings.TrimSpace(binding)
	if binding == "" {
		return nil
	}
	if strings.HasPrefix(binding, "*") {
		fields := strings.Fields(binding)
		for index, field := range fields {
			if field == "as" && index+1 < len(fields) {
				name := trimDeclarationName(fields[index+1])
				if isSimpleIdentifier(name) {
					return []string{name}
				}
				return nil
			}
		}
		return nil
	}
	if strings.HasPrefix(binding, "{") {
		close := matchingCloseBrace(binding, 0)
		if close < 0 {
			return nil
		}
		var names []string
		for _, specifier := range splitTopLevelCommaSegments(binding[1:close]) {
			specifier = strings.TrimSpace(specifier)
			if strings.HasPrefix(specifier, "type ") {
				specifier = strings.TrimSpace(strings.TrimPrefix(specifier, "type "))
			}
			fields := strings.Fields(specifier)
			if len(fields) >= 3 && fields[len(fields)-2] == "as" {
				name := trimDeclarationName(fields[len(fields)-1])
				if isSimpleIdentifier(name) {
					names = append(names, name)
				}
				continue
			}
			name := trimDeclarationName(specifier)
			if isSimpleIdentifier(name) {
				names = append(names, name)
			}
		}
		return names
	}
	name := trimDeclarationName(binding)
	if isSimpleIdentifier(name) {
		return []string{name}
	}
	return nil
}

func esVariableDeclarationNames(code string) []string {
	trimmed := strings.TrimSpace(strings.TrimSuffix(code, ";"))
	trimmed = strings.TrimPrefix(trimmed, "export ")
	trimmed = strings.TrimPrefix(trimmed, "declare ")
	fields := strings.Fields(trimmed)
	if len(fields) < 2 {
		return nil
	}
	keyword := fields[0]
	if keyword != "const" && keyword != "let" && keyword != "var" {
		return nil
	}
	rest := strings.TrimSpace(strings.TrimPrefix(trimmed, keyword))
	var names []string
	for _, segment := range splitTopLevelCommaSegments(rest) {
		left := strings.TrimSpace(segment)
		if eq := firstTopLevelByte(left, '='); eq >= 0 {
			left = strings.TrimSpace(left[:eq])
		}
		names = append(names, esBindingPatternNames(left)...)
	}
	return names
}

func esBindingPatternNames(pattern string) []string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil
	}
	if strings.HasPrefix(pattern, "...") {
		pattern = strings.TrimSpace(strings.TrimPrefix(pattern, "..."))
	}
	if strings.HasPrefix(pattern, "{") {
		close := matchingCloseBrace(pattern, 0)
		if close < 0 {
			return nil
		}
		return esObjectBindingPatternNames(pattern[1:close])
	}
	if strings.HasPrefix(pattern, "[") {
		close := matchingCloseBracket(pattern, 0)
		if close < 0 {
			return nil
		}
		return esArrayBindingPatternNames(pattern[1:close])
	}
	if colon := firstTopLevelByte(pattern, ':'); colon >= 0 {
		pattern = strings.TrimSpace(pattern[:colon])
	}
	name := trimDeclarationName(pattern)
	if isSimpleIdentifier(name) {
		return []string{name}
	}
	return nil
}

func esObjectBindingPatternNames(pattern string) []string {
	var names []string
	for _, part := range splitTopLevelCommaSegments(pattern) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, "...") {
			names = append(names, esBindingPatternNames(part)...)
			continue
		}
		if colon := firstTopLevelByte(part, ':'); colon >= 0 {
			names = append(names, esBindingPatternNames(strings.TrimSpace(part[colon+1:]))...)
			continue
		}
		if eq := firstTopLevelByte(part, '='); eq >= 0 {
			part = strings.TrimSpace(part[:eq])
		}
		names = append(names, esBindingPatternNames(part)...)
	}
	return names
}

func esArrayBindingPatternNames(pattern string) []string {
	var names []string
	for _, part := range splitTopLevelCommaSegments(pattern) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if eq := firstTopLevelByte(part, '='); eq >= 0 {
			part = strings.TrimSpace(part[:eq])
		}
		names = append(names, esBindingPatternNames(part)...)
	}
	return names
}

func matchingCloseBracket(value string, open int) int {
	state := codeScanState{}
	depth := 0
	for index := 0; index < len(value); index++ {
		if state.step(value, index) {
			continue
		}
		switch value[index] {
		case '[':
			depth++
		case ']':
			depth--
			if index > open && depth == 0 {
				return index
			}
		}
		state.trackDelimiterExceptBracket(value[index])
	}
	return -1
}

func esFunctionDeclarationShape(code string) (name string, parameters string, returnType string, ok bool) {
	trimmed := strings.TrimSpace(code)
	if strings.HasPrefix(trimmed, "export ") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "export "))
	}
	if strings.HasPrefix(trimmed, "async ") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "async "))
	}
	if !strings.HasPrefix(trimmed, "function ") {
		return "", "", "", false
	}
	remaining := strings.TrimSpace(strings.TrimPrefix(trimmed, "function "))
	paren := strings.Index(remaining, "(")
	if paren < 0 {
		return "", "", "", false
	}
	name = strings.TrimSpace(remaining[:paren])
	if !isSimpleIdentifier(name) {
		return "", "", "", false
	}
	afterName := remaining[paren:]
	closeParen := matchingCloseParen(afterName)
	if closeParen < 0 {
		return "", "", "", false
	}
	parameters = strings.TrimSpace(afterName[:closeParen+1])
	afterParams := strings.TrimSpace(afterName[closeParen+1:])
	body := strings.Index(afterParams, "{")
	if body < 0 {
		return "", "", "", false
	}
	returnType = strings.TrimSpace(afterParams[:body])
	if strings.HasPrefix(returnType, ":") {
		returnType = strings.TrimSpace(strings.TrimPrefix(returnType, ":"))
	}
	return name, parameters, returnType, true
}

func esFunctionPrototypeName(code string) (string, bool) {
	name, _, _, ok := esFunctionPrototypeShape(code)
	return name, ok
}

func esFunctionPrototypeShape(code string) (name string, parameters string, returnType string, ok bool) {
	trimmed := strings.TrimSpace(code)
	if strings.HasPrefix(trimmed, "export ") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "export "))
	}
	if strings.HasPrefix(trimmed, "default ") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "default "))
	}
	if strings.HasPrefix(trimmed, "declare ") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "declare "))
	}
	if strings.HasPrefix(trimmed, "async ") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "async "))
	}
	if !strings.HasPrefix(trimmed, "function ") {
		return "", "", "", false
	}
	remaining := strings.TrimSpace(strings.TrimPrefix(trimmed, "function "))
	paren := strings.Index(remaining, "(")
	if paren < 0 {
		return "", "", "", false
	}
	name = strings.TrimSpace(remaining[:paren])
	if !isSimpleIdentifier(name) {
		return "", "", "", false
	}
	afterName := remaining[paren:]
	closeParen := matchingCloseParen(afterName)
	if closeParen < 0 {
		return "", "", "", false
	}
	parameters = strings.TrimSpace(afterName[:closeParen+1])
	afterParams := strings.TrimSpace(afterName[closeParen+1:])
	semicolon := firstTopLevelByte(afterParams, ';')
	if semicolon < 0 {
		return "", "", "", false
	}
	body := firstTopLevelByte(afterParams, '{')
	if body >= 0 && body < semicolon {
		return "", "", "", false
	}
	returnType = strings.TrimSpace(afterParams[:semicolon])
	if strings.HasPrefix(returnType, ":") {
		returnType = strings.TrimSpace(strings.TrimPrefix(returnType, ":"))
	}
	return name, parameters, returnType, true
}

func esFunctionExpressionShape(code string) (parameters string, returnType string, ok bool) {
	expression := strings.TrimSpace(code)
	if _, ok := esTopLevelDeclarationName(expression); ok {
		eq := firstTopLevelByte(expression, '=')
		if eq < 0 {
			return "", "", false
		}
		expression = strings.TrimSpace(expression[eq+1:])
	}
	expression = strings.TrimSuffix(strings.TrimSpace(expression), ";")
	if strings.HasPrefix(expression, "async ") {
		expression = strings.TrimSpace(strings.TrimPrefix(expression, "async "))
	}
	if strings.HasPrefix(expression, "function ") {
		remaining := strings.TrimSpace(strings.TrimPrefix(expression, "function "))
		if strings.HasPrefix(remaining, "(") {
			return esFunctionExpressionParametersAndReturn(remaining)
		}
		return "", "", false
	}
	if strings.HasPrefix(expression, "(") {
		return esFunctionExpressionParametersAndReturn(expression)
	}
	if arrow := strings.Index(expression, "=>"); arrow >= 0 {
		parameter := strings.TrimSpace(expression[:arrow])
		if parameter == "" || strings.ContainsAny(parameter, " \t\n\r,") {
			return "", "", false
		}
		return "(" + parameter + ")", "", true
	}
	return "", "", false
}

func esFunctionExpressionParametersAndReturn(expression string) (parameters string, returnType string, ok bool) {
	closeParen := matchingCloseParen(expression)
	if closeParen < 0 {
		return "", "", false
	}
	parameters = strings.TrimSpace(expression[:closeParen+1])
	rest := strings.TrimSpace(expression[closeParen+1:])
	if strings.HasPrefix(rest, "async") {
		rest = strings.TrimSpace(strings.TrimPrefix(rest, "async"))
	}
	if strings.HasPrefix(rest, ":") {
		afterColon := strings.TrimSpace(strings.TrimPrefix(rest, ":"))
		arrow := strings.Index(afterColon, "=>")
		body := strings.Index(afterColon, "{")
		end := -1
		switch {
		case arrow >= 0 && body >= 0:
			end = minInt(arrow, body)
		case arrow >= 0:
			end = arrow
		case body >= 0:
			end = body
		default:
			return "", "", false
		}
		returnType = strings.TrimSpace(afterColon[:end])
		rest = strings.TrimSpace(afterColon[end:])
	}
	if strings.HasPrefix(rest, "=>") || strings.HasPrefix(rest, "{") {
		return parameters, returnType, true
	}
	return "", "", false
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func esIdentifierBeforeParen(value string) (string, bool) {
	if paren := strings.Index(value, "("); paren >= 0 {
		value = value[:paren]
	}
	if !isSimpleIdentifier(value) {
		return "", false
	}
	return value, true
}

func ensureTrailingSemicolonForESDeclaration(code string) string {
	code = strings.TrimSpace(code)
	if strings.HasSuffix(code, "}") || strings.HasSuffix(code, ";") {
		return code
	}
	return code + ";"
}

func rustFunctionDeclarationName(code string) (string, bool) {
	return keywordFunctionDeclarationName(code, "fn")
}

func rustVariableDeclarationName(code string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(code))
	for index, field := range fields {
		if field != "const" && field != "static" {
			continue
		}
		if index+1 >= len(fields) {
			return "", false
		}
		if !variableDeclarationPrefix(fields[:index]) {
			return "", false
		}
		nameIndex := index + 1
		if field == "static" && fields[nameIndex] == "mut" {
			nameIndex++
			if nameIndex >= len(fields) {
				return "", false
			}
		}
		name := trimDeclarationName(fields[nameIndex])
		if !isSimpleIdentifier(name) {
			return "", false
		}
		return name, true
	}
	return "", false
}

func rustUseDeclarationNames(code string) []string {
	trimmed := strings.TrimSpace(strings.TrimSuffix(code, ";"))
	fields := strings.Fields(trimmed)
	useIndex := -1
	for index, field := range fields {
		if field == "use" {
			useIndex = index
			break
		}
		if index == 0 && strings.HasPrefix(field, "pub") {
			continue
		}
		return nil
	}
	if useIndex < 0 || useIndex+1 >= len(fields) {
		return nil
	}
	useStart := strings.Index(trimmed, "use")
	if useStart < 0 {
		return nil
	}
	rest := strings.TrimSpace(trimmed[useStart+len("use"):])
	return rustUseBindingNames(rest)
}

func rustUseBindingNames(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, "*") {
		return nil
	}
	if open := firstTopLevelByte(value, '{'); open >= 0 {
		close := matchingCloseBrace(value, open)
		if close < 0 {
			return nil
		}
		var names []string
		for _, segment := range splitTopLevelCommaSegments(value[open+1 : close]) {
			names = append(names, rustUseBindingNames(segment)...)
		}
		return names
	}
	fields := strings.Fields(value)
	for index, field := range fields {
		if field == "as" && index+1 < len(fields) {
			name := trimDeclarationName(fields[index+1])
			if isSimpleIdentifier(name) {
				return []string{name}
			}
			return nil
		}
	}
	if lastSeparator := strings.LastIndex(value, "::"); lastSeparator >= 0 {
		value = value[lastSeparator+len("::"):]
	}
	name := trimDeclarationName(value)
	if isSimpleIdentifier(name) {
		return []string{name}
	}
	return nil
}

func zigFunctionDeclarationName(code string) (string, bool) {
	return keywordFunctionDeclarationName(code, "fn")
}

func keywordVariableDeclarationName(code string, keywords ...string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(code))
	for index, field := range fields {
		if !stringInSlice(keywords, field) {
			continue
		}
		if index+1 >= len(fields) {
			return "", false
		}
		if !variableDeclarationPrefix(fields[:index]) {
			return "", false
		}
		name := trimDeclarationName(fields[index+1])
		if !isSimpleIdentifier(name) {
			return "", false
		}
		return name, true
	}
	return "", false
}

func keywordTypeDeclarationName(code string, keywords ...string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(code))
	for index, field := range fields {
		if !stringInSlice(keywords, field) {
			continue
		}
		if index+1 >= len(fields) {
			return "", false
		}
		if !typeDeclarationPrefix(fields[:index]) {
			return "", false
		}
		name := trimDeclarationName(fields[index+1])
		if !isSimpleIdentifier(name) {
			return "", false
		}
		return name, true
	}
	return "", false
}

func rustFunctionDeclarationShape(code string) (name string, parameters string, returnType string, ok bool) {
	return keywordFunctionDeclarationShape(code, "fn", "()")
}

func zigFunctionDeclarationShape(code string) (name string, parameters string, returnType string, ok bool) {
	return keywordFunctionDeclarationShape(code, "fn", "")
}

func keywordFunctionDeclarationShape(code string, keyword string, defaultReturnType string) (name string, parameters string, returnType string, ok bool) {
	name, ok = keywordFunctionDeclarationName(code, keyword)
	if !ok {
		return "", "", "", false
	}
	pattern := keyword + " " + name
	start := strings.Index(code, pattern)
	if start < 0 {
		return "", "", "", false
	}
	remaining := strings.TrimSpace(code[start+len(pattern):])
	if !strings.HasPrefix(remaining, "(") {
		return "", "", "", false
	}
	closeParen := matchingCloseParen(remaining)
	if closeParen < 0 {
		return "", "", "", false
	}
	parameters = strings.TrimSpace(remaining[:closeParen+1])
	afterParams := strings.TrimSpace(remaining[closeParen+1:])
	body := strings.Index(afterParams, "{")
	if body < 0 {
		return "", "", "", false
	}
	returnType = strings.TrimSpace(afterParams[:body])
	if strings.HasPrefix(returnType, "->") {
		returnType = strings.TrimSpace(strings.TrimPrefix(returnType, "->"))
	}
	if returnType == "" {
		returnType = defaultReturnType
	}
	return name, parameters, returnType, true
}

func matchingCloseParen(value string) int {
	depth := 0
	for index, r := range value {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func keywordFunctionDeclarationName(code string, keyword string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(code))
	for index, field := range fields {
		if field != keyword || index+1 >= len(fields) {
			continue
		}
		if !functionDeclarationPrefix(fields[:index]) {
			return "", false
		}
		name := fields[index+1]
		if paren := strings.Index(name, "("); paren >= 0 {
			name = name[:paren]
		}
		if !isSimpleIdentifier(name) {
			return "", false
		}
		return name, true
	}
	return "", false
}

func functionDeclarationPrefix(fields []string) bool {
	for _, field := range fields {
		if field == "" {
			continue
		}
		if strings.HasPrefix(field, "pub") || field == "export" || field == "extern" || field == "inline" ||
			field == "unsafe" || field == "async" || field == "const" {
			continue
		}
		return false
	}
	return true
}

func variableDeclarationPrefix(fields []string) bool {
	for _, field := range fields {
		if field == "" {
			continue
		}
		if strings.HasPrefix(field, "pub") || field == "export" || field == "extern" || field == "static" ||
			field == "threadlocal" || field == "const" || field == "inline" || field == "constexpr" {
			continue
		}
		return false
	}
	return true
}

func typeDeclarationPrefix(fields []string) bool {
	for _, field := range fields {
		if field == "" {
			continue
		}
		if strings.HasPrefix(field, "pub") || field == "export" {
			continue
		}
		return false
	}
	return true
}

func cLikeTypeDeclarationPrefix(fields []string) bool {
	for _, field := range fields {
		if field == "" {
			continue
		}
		if field == "public" || field == "private" || field == "protected" || field == "internal" ||
			field == "static" || field == "abstract" || field == "sealed" || field == "partial" ||
			field == "final" || field == "readonly" || field == "non-sealed" {
			continue
		}
		return false
	}
	return true
}

func trimDeclarationName(name string) string {
	name = strings.TrimSpace(name)
	for _, separator := range []string{"=", ":", ";", "{", "(", ","} {
		if index := strings.Index(name, separator); index >= 0 {
			name = name[:index]
		}
	}
	return strings.TrimSpace(name)
}

func stringInSlice(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsDeclarationBoundary(value string) bool {
	return strings.ContainsAny(value, "{};=")
}

func isSimpleIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for index, r := range value {
		if index == 0 {
			if r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
				continue
			}
			return false
		}
		if r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}
