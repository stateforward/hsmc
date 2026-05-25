package hsmc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CommandAdapter delegates behavior/global translation to an external command
// over the adapter JSON protocol.
type CommandAdapter struct {
	NameValue      string
	Command        string
	DefaultCommand string
	Args           []string
	PatchOutput    string
}

func (adapter CommandAdapter) Name() string {
	name := strings.TrimSpace(adapter.NameValue)
	if name == "" {
		return "command"
	}
	return name
}

func (adapter CommandAdapter) Translate(ctx context.Context, request AdapterRequest) (*BehaviorPatch, error) {
	command := strings.TrimSpace(adapter.Command)
	if command == "" {
		command = strings.TrimSpace(adapter.DefaultCommand)
	}
	if command == "" {
		return nil, fmt.Errorf("%s adapter command is required", adapter.Name())
	}
	protocol := NewAdapterProtocol(request)
	input, err := json.MarshalIndent(protocol, "", "  ")
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, command, adapter.Args...)
	cmd.Stdin = bytes.NewReader(input)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s adapter failed: %w%s", adapter.Name(), err, adapterFailureContext(stdout.Bytes(), stderr.Bytes()))
	}
	if adapter.PatchOutput != "" {
		data, err := os.ReadFile(adapter.PatchOutput)
		if err != nil {
			return nil, fmt.Errorf("%s adapter patch output %q could not be read: %w", adapter.Name(), adapter.PatchOutput, err)
		}
		if len(bytes.TrimSpace(data)) > 0 {
			return ParseBehaviorPatchWithContext(data)
		}
	}
	return ParseBehaviorPatchWithContext(stdout.Bytes())
}

func adapterFailureContext(stdout []byte, stderr []byte) string {
	var parts []string
	if text := diagnosticSnippet(stderr); text != "" {
		parts = append(parts, "stderr "+fmt.Sprintf("%q", text))
	}
	if text := diagnosticSnippet(stdout); text != "" {
		parts = append(parts, "stdout "+fmt.Sprintf("%q", text))
	}
	if len(parts) == 0 {
		return ""
	}
	return ": " + strings.Join(parts, "; ")
}
