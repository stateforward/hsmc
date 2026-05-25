package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/stateforward/hsmc/internal/hsmc"
)

func main() {
	var from string
	var to string
	var adapterName string
	var adapterCommand string
	var adapterArgs string
	var inputPath string
	var outputPath string
	var listLanguages bool
	var listSourceLanguages bool
	var listTargetLanguages bool
	var listAdapters bool
	var showVersion bool
	var printAdapterProtocol bool
	flag.StringVar(&from, "from", "", "source language; inferred from -in when omitted")
	flag.StringVar(&to, "to", "", "target language; inferred from -out when omitted, otherwise go")
	flag.StringVar(&adapterName, "adapter", "none", "behavior adapter: none, codex, or command")
	flag.StringVar(&adapterCommand, "adapter-command", "", "command used by the codex or command adapter")
	flag.StringVar(&adapterArgs, "adapter-args", "", "comma-separated arguments passed to the adapter command")
	flag.StringVar(&inputPath, "in", "", "source file")
	flag.StringVar(&outputPath, "out", "", "output file; stdout when empty")
	flag.BoolVar(&listLanguages, "list-languages", false, "list all supported canonical source or target languages")
	flag.BoolVar(&listSourceLanguages, "list-source-languages", false, "list supported canonical source languages")
	flag.BoolVar(&listTargetLanguages, "list-target-languages", false, "list supported canonical target languages")
	flag.BoolVar(&listAdapters, "list-adapters", false, "list supported behavior adapters")
	flag.BoolVar(&showVersion, "version", false, "print hsmc version")
	flag.BoolVar(&printAdapterProtocol, "print-adapter-protocol", false, "print the adapter protocol JSON without invoking an adapter")
	flag.Parse()

	if showVersion {
		fmt.Fprintln(os.Stdout, hsmc.Version)
		return
	}
	if listLanguages {
		fmt.Fprintln(os.Stdout, formatLanguages(hsmc.SupportedLanguages()))
		return
	}
	if listSourceLanguages {
		fmt.Fprintln(os.Stdout, formatLanguages(hsmc.SupportedSourceLanguages()))
		return
	}
	if listTargetLanguages {
		fmt.Fprintln(os.Stdout, formatLanguages(hsmc.SupportedTargetLanguages()))
		return
	}
	if listAdapters {
		fmt.Fprintln(os.Stdout, formatAdapters(hsmc.NewCompiler().AdapterNames()))
		return
	}
	if inputPath == "" {
		exitf("-in is required")
	}
	data, sourcePath, err := readInput(inputPath, os.Stdin)
	if err != nil {
		exitf("read input: %v", err)
	}
	adapterName = strings.TrimSpace(adapterName)
	compiler := hsmc.NewCompiler()
	switch adapterName {
	case "codex":
		compiler.RegisterAdapter(hsmc.CodexAdapter{Command: adapterCommand, Args: splitArgs(adapterArgs)})
	case "command":
		compiler.RegisterAdapter(hsmc.CommandAdapter{Command: adapterCommand, Args: splitArgs(adapterArgs)})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	options := compileOptions(from, to, sourcePath, outputPath, adapterName)
	if printAdapterProtocol {
		protocolOptions := adapterProtocolOptions(from, to, sourcePath, outputPath)
		output, err := adapterProtocolJSON(ctx, compiler, hsmc.SourceInput{Path: sourcePath, Data: data}, hsmc.CompileOptions{From: protocolOptions.From, To: protocolOptions.To})
		if err != nil {
			exitf("adapter protocol: %v", err)
		}
		if err := writeOutput(outputPath, output); err != nil {
			exitf("write output: %v", err)
		}
		return
	}
	output, _, err := compiler.Compile(ctx, hsmc.SourceInput{Path: sourcePath, Data: data}, hsmc.CompileOptions{
		From:    options.From,
		To:      options.To,
		Adapter: options.Adapter,
	})
	if err != nil {
		exitf("compile: %v", err)
	}
	if err := writeOutput(outputPath, output); err != nil {
		exitf("write output: %v", err)
	}
}

func adapterProtocolJSON(ctx context.Context, compiler *hsmc.Compiler, input hsmc.SourceInput, options hsmc.CompileOptions) ([]byte, error) {
	protocol, _, err := compiler.BuildAdapterProtocol(ctx, input, options)
	if err != nil {
		return nil, err
	}
	output, err := json.MarshalIndent(protocol, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(output, '\n'), nil
}

func writeOutput(outputPath string, output []byte) error {
	if outputPath == "" {
		_, err := os.Stdout.Write(output)
		return err
	}
	if dir := filepath.Dir(outputPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(outputPath, output, 0o644)
}

func readInput(path string, stdin io.Reader) ([]byte, string, error) {
	if path == "-" {
		data, err := io.ReadAll(stdin)
		return data, "stdin", err
	}
	data, err := os.ReadFile(path)
	return data, path, err
}

func splitArgs(value string) []string {
	if value == "" {
		return nil
	}
	raw := strings.Split(value, ",")
	args := make([]string, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item != "" {
			args = append(args, item)
		}
	}
	return args
}

func parseLanguage(value string) hsmc.Language {
	if language, ok := hsmc.ParseLanguage(value); ok {
		return language
	}
	return hsmc.Language(value)
}

func compileOptions(from string, to string, inputPath string, outputPath string, adapter string) hsmc.CompileOptions {
	return hsmc.CompileOptions{
		From:    resolveSourceLanguage(from, inputPath),
		To:      resolveTargetLanguage(to, outputPath),
		Adapter: strings.TrimSpace(adapter),
	}
}

func adapterProtocolOptions(from string, to string, inputPath string, outputPath string) hsmc.CompileOptions {
	target := resolveTargetLanguage(to, outputPath)
	if strings.TrimSpace(to) == "" && target == hsmc.LanguageJSONIR {
		target = hsmc.LanguageGo
	}
	return hsmc.CompileOptions{
		From: resolveSourceLanguage(from, inputPath),
		To:   target,
	}
}

func resolveSourceLanguage(value string, inputPath string) hsmc.Language {
	if strings.TrimSpace(value) != "" {
		return parseLanguage(value)
	}
	if language, ok := inferLanguageFromPath(inputPath); ok {
		return language
	}
	return hsmc.LanguageGo
}

func resolveTargetLanguage(value string, outputPath string) hsmc.Language {
	if strings.TrimSpace(value) != "" {
		return parseLanguage(value)
	}
	if language, ok := inferLanguageFromPath(outputPath); ok {
		return language
	}
	return hsmc.LanguageGo
}

func inferLanguageFromPath(path string) (hsmc.Language, bool) {
	if path == "" || path == "stdin" || path == "-" {
		return "", false
	}
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".hsm.json"):
		return hsmc.LanguageJSONIR, true
	}
	switch filepath.Ext(lower) {
	case ".cs", ".csx":
		return hsmc.LanguageCSharp, true
	case ".c++", ".cc", ".cpp", ".cppm", ".cxx", ".h", ".h++", ".hh", ".hpp", ".hxx", ".inl", ".ipp", ".ixx", ".mpp", ".tpp":
		return hsmc.LanguageCPP, true
	case ".dart":
		return hsmc.LanguageDart, true
	case ".go":
		return hsmc.LanguageGo, true
	case ".java":
		return hsmc.LanguageJava, true
	case ".js", ".mjs", ".cjs", ".jsx":
		return hsmc.LanguageJS, true
	case ".json":
		return hsmc.LanguageJSONIR, true
	case ".py":
		return hsmc.LanguagePython, true
	case ".rs":
		return hsmc.LanguageRust, true
	case ".ts", ".mts", ".cts", ".tsx":
		return hsmc.LanguageTS, true
	case ".zig":
		return hsmc.LanguageZig, true
	default:
		return "", false
	}
}

func formatLanguages(languages []hsmc.Language) string {
	values := make([]string, 0, len(languages))
	for _, language := range languages {
		values = append(values, string(language))
	}
	return strings.Join(values, "\n")
}

func formatAdapters(adapters []string) string {
	return strings.Join(adapters, "\n")
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "hsmc: "+format+"\n", args...)
	os.Exit(1)
}
