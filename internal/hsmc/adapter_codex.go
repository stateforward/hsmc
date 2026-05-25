package hsmc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// CodexAdapter delegates behavior/global translation to an external command
// over a JSON contract. The compiler never hands model edits to the adapter.
type CodexAdapter struct {
	Command string
	Args    []string
}

func (adapter CodexAdapter) Name() string { return "codex" }

func (adapter CodexAdapter) Translate(ctx context.Context, request AdapterRequest) (*BehaviorPatch, error) {
	output, err := os.CreateTemp("", "hsmc-codex-patch-*.json")
	if err != nil {
		return nil, err
	}
	outputPath := output.Name()
	if err := output.Close(); err != nil {
		return nil, err
	}
	defer os.Remove(outputPath)

	args := make([]string, 0, len(adapter.Args)+4)
	args = append(args, "exec")
	args = append(args, adapter.Args...)
	args = append(args, "--output-last-message", outputPath)
	args = append(args, "-")
	return CommandAdapter{
		NameValue:      adapter.Name(),
		Command:        adapter.Command,
		DefaultCommand: "codex",
		Args:           args,
		PatchOutput:    outputPath,
	}.Translate(ctx, request)
}

func ParseBehaviorPatch(output []byte) (*BehaviorPatch, error) {
	payload := bytes.TrimSpace(output)
	if len(payload) == 0 {
		return nil, nil
	}
	if json.Valid(payload) {
		return parseBehaviorPatchJSON(payload)
	}
	fenced, ok := fencedJSON(string(payload))
	if !ok {
		return nil, fmt.Errorf("adapter returned invalid patch JSON")
	}
	patch, err := parseBehaviorPatchJSON([]byte(fenced))
	if err != nil {
		return nil, fmt.Errorf("adapter returned invalid fenced patch JSON: %w", err)
	}
	return patch, nil
}

func ParseBehaviorPatchWithContext(output []byte) (*BehaviorPatch, error) {
	patch, err := ParseBehaviorPatch(output)
	if err == nil {
		return patch, nil
	}
	return nil, fmt.Errorf("%w: output %q", err, diagnosticSnippet(output))
}

func fencedJSON(text string) (string, bool) {
	start := strings.Index(text, "```")
	if start < 0 {
		return "", false
	}
	rest := text[start+3:]
	rest = strings.TrimLeft(rest, "\t ")
	if lineEnd := strings.IndexAny(rest, "\r\n"); lineEnd >= 0 {
		info := strings.TrimSpace(rest[:lineEnd])
		if info != "" {
			if !strings.EqualFold(info, "json") {
				return "", false
			}
			rest = rest[lineEnd:]
		}
	} else if strings.TrimSpace(rest) != "" {
		return "", false
	}
	rest = strings.TrimLeft(rest, "\r\n\t ")
	end, fenceEnd := closingFence(rest)
	if end < 0 {
		return "", false
	}
	if strings.Contains(rest[fenceEnd:], "```") {
		return "", false
	}
	return strings.TrimSpace(rest[:end]), true
}

func closingFence(text string) (int, int) {
	for index := 0; index < len(text); {
		lineStart := index
		lineEnd := len(text)
		if next := strings.IndexAny(text[index:], "\r\n"); next >= 0 {
			lineEnd = index + next
		}
		line := text[lineStart:lineEnd]
		trimmed := strings.TrimLeft(line, "\t ")
		if strings.HasPrefix(trimmed, "```") && strings.TrimSpace(trimmed[3:]) == "" {
			fenceStart := lineStart + len(line) - len(trimmed)
			return fenceStart, fenceStart + 3
		}
		if lineEnd == len(text) {
			break
		}
		index = lineEnd + 1
		if text[lineEnd] == '\r' && index < len(text) && text[index] == '\n' {
			index++
		}
	}
	return -1, -1
}

func parseBehaviorPatchJSON(payload []byte) (*BehaviorPatch, error) {
	if err := rejectDuplicateJSONFields(payload, "adapter patch"); err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(payload)
	if bytes.Equal(trimmed, []byte("null")) {
		return nil, fmt.Errorf("adapter patch must be a JSON object, got null")
	}
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, fmt.Errorf("adapter patch must be a JSON object")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, err
	}
	for field := range raw {
		switch field {
		case "imports", "globals", "behaviors":
		default:
			return nil, fmt.Errorf("adapter patch contains unsupported top-level field %q", field)
		}
	}
	if err := validatePatchJSONFields(raw); err != nil {
		return nil, err
	}
	var patch BehaviorPatch
	if err := json.Unmarshal(payload, &patch); err != nil {
		return nil, err
	}
	_, patch.ImportsSet = raw["imports"]
	return &patch, nil
}

func rejectDuplicateJSONFields(payload []byte, owner string) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	if err := rejectDuplicateJSONFieldsInValue(decoder, owner); err != nil {
		return err
	}
	return nil
}

func rejectDuplicateJSONFieldsInValue(decoder *json.Decoder, owner string) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("%s object key is not a string", owner)
			}
			if seen[key] {
				return fmt.Errorf("%s contains duplicate field %q", owner, key)
			}
			seen[key] = true
			if err := rejectDuplicateJSONFieldsInValue(decoder, jsonPath(owner, key)); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err
	case '[':
		index := 0
		for decoder.More() {
			if err := rejectDuplicateJSONFieldsInValue(decoder, fmt.Sprintf("%s[%d]", owner, index)); err != nil {
				return err
			}
			index++
		}
		_, err := decoder.Token()
		return err
	default:
		return nil
	}
}

func jsonPath(owner string, field string) string {
	if owner == "adapter patch" {
		return owner + " " + field
	}
	return owner + "." + field
}

func validatePatchJSONFields(raw map[string]json.RawMessage) error {
	if err := validateImportArrayFields("imports", raw["imports"]); err != nil {
		return err
	}
	if err := validateCodeBlockArrayFields("globals", raw["globals"]); err != nil {
		return err
	}
	if err := validateBehaviorArrayFields("behaviors", raw["behaviors"]); err != nil {
		return err
	}
	return nil
}

func validateCodeBlockArrayFields(owner string, raw json.RawMessage) error {
	if isJSONMissing(raw) {
		return nil
	}
	if isJSONNull(raw) {
		return fmt.Errorf("adapter patch field %q must be an array, got null", owner)
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return fmt.Errorf("adapter patch field %q must be an array: %w", owner, err)
	}
	allowed := map[string]bool{
		"id":      true,
		"code":    true,
		"imports": true,
	}
	for index, item := range items {
		if err := rejectUnsupportedObjectFields(fmt.Sprintf("%s[%d]", owner, index), item, allowed); err != nil {
			return err
		}
		if err := validateStringFields(fmt.Sprintf("%s[%d]", owner, index), item, "id", "code"); err != nil {
			return err
		}
		if err := validateRequiredStringField(fmt.Sprintf("%s[%d]", owner, index), item, "id"); err != nil {
			return err
		}
		if err := validateOptionalNonWhitespaceCodeFields(fmt.Sprintf("%s[%d]", owner, index), item, "code"); err != nil {
			return err
		}
		if err := validateImportArrayFields(fmt.Sprintf("%s[%d].imports", owner, index), item["imports"]); err != nil {
			return err
		}
	}
	return nil
}

func validateBehaviorArrayFields(owner string, raw json.RawMessage) error {
	if isJSONMissing(raw) {
		return nil
	}
	if isJSONNull(raw) {
		return fmt.Errorf("adapter patch field %q must be an array, got null", owner)
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return fmt.Errorf("adapter patch field %q must be an array: %w", owner, err)
	}
	allowed := map[string]bool{
		"id":      true,
		"body":    true,
		"code":    true,
		"imports": true,
	}
	for index, item := range items {
		if err := rejectUnsupportedObjectFields(fmt.Sprintf("%s[%d]", owner, index), item, allowed); err != nil {
			return err
		}
		if err := validateStringFields(fmt.Sprintf("%s[%d]", owner, index), item, "id", "body", "code"); err != nil {
			return err
		}
		if err := validateRequiredStringField(fmt.Sprintf("%s[%d]", owner, index), item, "id"); err != nil {
			return err
		}
		if err := validateOptionalNonWhitespaceCodeFields(fmt.Sprintf("%s[%d]", owner, index), item, "body", "code"); err != nil {
			return err
		}
		if err := validateBehaviorPatchJSONCodeChoice(fmt.Sprintf("%s[%d]", owner, index), item); err != nil {
			return err
		}
		if err := validateImportArrayFields(fmt.Sprintf("%s[%d].imports", owner, index), item["imports"]); err != nil {
			return err
		}
	}
	return nil
}

func validateBehaviorPatchJSONCodeChoice(owner string, item map[string]json.RawMessage) error {
	body := jsonStringFieldValue(item["body"])
	code := jsonStringFieldValue(item["code"])
	if strings.TrimSpace(body) != "" && strings.TrimSpace(code) != "" {
		return fmt.Errorf("adapter patch %s must set either body or code, not both", owner)
	}
	return nil
}

func validateImportArrayFields(owner string, raw json.RawMessage) error {
	if isJSONMissing(raw) {
		return nil
	}
	if isJSONNull(raw) {
		return fmt.Errorf("adapter patch field %q must be an array, got null", owner)
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return fmt.Errorf("adapter patch field %q must be an array: %w", owner, err)
	}
	allowed := map[string]bool{
		"path":       true,
		"alias":      true,
		"default":    true,
		"specifiers": true,
		"hidden":     true,
		"static":     true,
		"deferred":   true,
		"type_only":  true,
		"language":   true,
	}
	for index, item := range items {
		if err := rejectUnsupportedObjectFields(fmt.Sprintf("%s[%d]", owner, index), item, allowed); err != nil {
			return err
		}
		if err := validateStringFields(fmt.Sprintf("%s[%d]", owner, index), item, "path", "alias", "default", "language"); err != nil {
			return err
		}
		if err := validateRequiredStringField(fmt.Sprintf("%s[%d]", owner, index), item, "path"); err != nil {
			return err
		}
		if err := validateOptionalTrimmedStringFields(fmt.Sprintf("%s[%d]", owner, index), item, "alias", "default", "language"); err != nil {
			return err
		}
		if err := validateSpecifierArrayFields(fmt.Sprintf("%s[%d].specifiers", owner, index), item["specifiers"]); err != nil {
			return err
		}
		if err := validateStringArrayField(fmt.Sprintf("%s[%d].hidden", owner, index), item["hidden"]); err != nil {
			return err
		}
		if err := validateBoolFields(fmt.Sprintf("%s[%d]", owner, index), item, "static", "deferred", "type_only"); err != nil {
			return err
		}
	}
	return nil
}

func validateBoolFields(owner string, item map[string]json.RawMessage, fields ...string) error {
	for _, field := range fields {
		raw, ok := item[field]
		if !ok {
			continue
		}
		if isJSONNull(raw) {
			return fmt.Errorf("adapter patch field %q must be a boolean, got null", owner+"."+field)
		}
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			return fmt.Errorf("adapter patch field %q must be a boolean", owner+"."+field)
		}
	}
	return nil
}

func validateSpecifierArrayFields(owner string, raw json.RawMessage) error {
	if isJSONMissing(raw) {
		return nil
	}
	if isJSONNull(raw) {
		return fmt.Errorf("adapter patch field %q must be an array, got null", owner)
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return fmt.Errorf("adapter patch field %q must be an array: %w", owner, err)
	}
	allowed := map[string]bool{
		"name":      true,
		"alias":     true,
		"type_only": true,
	}
	for index, item := range items {
		if err := rejectUnsupportedObjectFields(fmt.Sprintf("%s[%d]", owner, index), item, allowed); err != nil {
			return err
		}
		if err := validateStringFields(fmt.Sprintf("%s[%d]", owner, index), item, "name", "alias"); err != nil {
			return err
		}
		if err := validateRequiredStringField(fmt.Sprintf("%s[%d]", owner, index), item, "name"); err != nil {
			return err
		}
		if err := validateOptionalTrimmedStringFields(fmt.Sprintf("%s[%d]", owner, index), item, "alias"); err != nil {
			return err
		}
		if err := validateBoolFields(fmt.Sprintf("%s[%d]", owner, index), item, "type_only"); err != nil {
			return err
		}
	}
	return nil
}

func validateStringArrayField(owner string, raw json.RawMessage) error {
	if isJSONMissing(raw) {
		return nil
	}
	if isJSONNull(raw) {
		return fmt.Errorf("adapter patch field %q must be an array, got null", owner)
	}
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return fmt.Errorf("adapter patch field %q must be an array: %w", owner, err)
	}
	for index, rawValue := range values {
		if isJSONNull(rawValue) {
			return fmt.Errorf("adapter patch %s[%d] must be a string, got null", owner, index)
		}
		var value string
		if err := json.Unmarshal(rawValue, &value); err != nil {
			return fmt.Errorf("adapter patch %s[%d] must be a string: %w", owner, index, err)
		}
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("adapter patch %s[%d] is empty", owner, index)
		}
		if value != strings.TrimSpace(value) {
			return fmt.Errorf("adapter patch %s[%d] %q has leading or trailing whitespace", owner, index, value)
		}
	}
	return nil
}

func rejectUnsupportedObjectFields(owner string, item map[string]json.RawMessage, allowed map[string]bool) error {
	for field := range item {
		if !allowed[field] {
			return fmt.Errorf("adapter patch %s contains unsupported field %q", owner, field)
		}
	}
	return nil
}

func validateStringFields(owner string, item map[string]json.RawMessage, fields ...string) error {
	for _, field := range fields {
		raw, ok := item[field]
		if !ok {
			continue
		}
		if isJSONNull(raw) {
			return fmt.Errorf("adapter patch %s.%s must be a string, got null", owner, field)
		}
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return fmt.Errorf("adapter patch %s.%s must be a string: %w", owner, field, err)
		}
	}
	return nil
}

func validateRequiredStringField(owner string, item map[string]json.RawMessage, field string) error {
	raw, ok := item[field]
	if !ok {
		return fmt.Errorf("adapter patch %s.%s is required", owner, field)
	}
	value := jsonStringFieldValue(raw)
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("adapter patch %s.%s is required", owner, field)
	}
	if value != trimmed {
		return fmt.Errorf("adapter patch %s.%s %q has leading or trailing whitespace", owner, field, value)
	}
	return nil
}

func validateOptionalTrimmedStringFields(owner string, item map[string]json.RawMessage, fields ...string) error {
	for _, field := range fields {
		raw, ok := item[field]
		if !ok {
			continue
		}
		value := jsonStringFieldValue(raw)
		if value == "" {
			continue
		}
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return fmt.Errorf("adapter patch %s.%s is empty", owner, field)
		}
		if value != trimmed {
			return fmt.Errorf("adapter patch %s.%s %q has leading or trailing whitespace", owner, field, value)
		}
	}
	return nil
}

func validateOptionalNonWhitespaceCodeFields(owner string, item map[string]json.RawMessage, fields ...string) error {
	for _, field := range fields {
		raw, ok := item[field]
		if !ok {
			continue
		}
		value := jsonStringFieldValue(raw)
		if value != "" && strings.TrimSpace(value) == "" {
			return fmt.Errorf("adapter patch %s.%s contains whitespace-only code", owner, field)
		}
	}
	return nil
}

func jsonStringFieldValue(raw json.RawMessage) string {
	if isJSONMissing(raw) || isJSONNull(raw) {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return value
}

func isJSONNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

func isJSONMissing(raw json.RawMessage) bool {
	return len(bytes.TrimSpace(raw)) == 0
}

func diagnosticSnippet(output []byte) string {
	text := strings.TrimSpace(string(output))
	text = strings.Join(strings.Fields(text), " ")
	const maxLen = 300
	if len(text) > maxLen {
		return text[:maxLen] + "..."
	}
	return text
}
