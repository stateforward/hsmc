package hsmc

import "strings"

func classifyAttributeSecondArg(language Language, arg string) (field string, value string) {
	value = strings.TrimSpace(arg)
	if attributeArgLooksLikeType(language, value) {
		return "type", value
	}
	return "default", value
}

func attributeArgLooksLikeType(language Language, value string) bool {
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "typeof(") && strings.HasSuffix(value, ")") {
		return true
	}
	if strings.HasSuffix(value, ".class") {
		return true
	}
	normalized := strings.TrimPrefix(value, "*")
	switch language {
	case LanguageCSharp:
		return isOneOf(normalized, "bool", "byte", "char", "decimal", "double", "float", "int", "long", "object", "sbyte", "short", "string", "uint", "ulong", "ushort")
	case LanguageCPP:
		return strings.HasPrefix(normalized, "std::type_identity<") && strings.HasSuffix(normalized, ">{}") ||
			isOneOf(normalized, "bool", "char", "char16_t", "char32_t", "double", "float", "int", "long", "short", "signed", "size_t", "std::string", "string", "unsigned", "void", "wchar_t")
	case LanguageDart:
		return isOneOf(normalized, "bool", "DateTime", "double", "Duration", "dynamic", "int", "num", "Object", "String")
	case LanguageGo:
		return isOneOf(normalized, "any", "bool", "byte", "complex64", "complex128", "error", "float32", "float64", "int", "int8", "int16", "int32", "int64", "rune", "string", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr")
	case LanguageJava:
		return isOneOf(normalized, "boolean", "byte", "char", "double", "float", "int", "long", "short", "Boolean.class", "Byte.class", "Character.class", "Double.class", "Float.class", "Integer.class", "Long.class", "Object.class", "Short.class", "String.class")
	case LanguageJS, LanguageTS:
		return isOneOf(normalized, "Array", "BigInt", "Boolean", "Date", "Map", "Number", "Object", "Set", "String", "Symbol", "boolean", "number", "object", "string")
	case LanguagePython:
		return isOneOf(normalized, "Any", "bool", "bytes", "dict", "float", "int", "list", "object", "set", "str", "tuple")
	case LanguageRust:
		return isOneOf(normalized, "bool", "char", "f32", "f64", "i8", "i16", "i32", "i64", "i128", "isize", "String", "str", "u8", "u16", "u32", "u64", "u128", "usize")
	case LanguageZig:
		return isOneOf(normalized, "bool", "comptime_float", "comptime_int", "f16", "f32", "f64", "f80", "f128", "i8", "i16", "i32", "i64", "i128", "isize", "usize", "u8", "u16", "u32", "u64", "u128", "void")
	default:
		return false
	}
}

func attributeTypeIsTargetExpression(attribute Attribute, target Language) bool {
	return strings.TrimSpace(attribute.Type) != "" && (attribute.Language == "" || attribute.Language == target)
}

func isOneOf(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if value == candidate {
			return true
		}
	}
	return false
}
