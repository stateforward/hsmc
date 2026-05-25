package hsmc

import (
	"context"
	"fmt"

	tree_sitter_dart "github.com/UserNobody14/tree-sitter-dart/bindings/go"
	tree_sitter_zig "github.com/tree-sitter-grammars/tree-sitter-zig/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_csharp "github.com/tree-sitter/tree-sitter-c-sharp/bindings/go"
	tree_sitter_cpp "github.com/tree-sitter/tree-sitter-cpp/bindings/go"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tree_sitter_java "github.com/tree-sitter/tree-sitter-java/bindings/go"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
	tree_sitter_rust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	tree_sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

type parserGrammar string

const (
	parserGrammarCPP        parserGrammar = "cpp"
	parserGrammarCSharp     parserGrammar = "csharp"
	parserGrammarDart       parserGrammar = "dart"
	parserGrammarGo         parserGrammar = "go"
	parserGrammarJava       parserGrammar = "java"
	parserGrammarJavaScript parserGrammar = "javascript"
	parserGrammarPython     parserGrammar = "python"
	parserGrammarRust       parserGrammar = "rust"
	parserGrammarTypeScript parserGrammar = "typescript"
	parserGrammarTSX        parserGrammar = "tsx"
	parserGrammarZig        parserGrammar = "zig"
)

func parseTree(ctx context.Context, input SourceInput, grammar parserGrammar) (*sitter.Tree, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	language, err := nativeLanguage(grammar)
	if err != nil {
		return nil, err
	}
	if err := parser.SetLanguage(language); err != nil {
		return nil, err
	}
	tree := parser.ParseCtx(ctx, input.Data, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse %s", input.Path)
	}
	if tree.RootNode().HasError() {
		tree.Close()
		return nil, fmt.Errorf("parse errors in %s", input.Path)
	}
	return tree, nil
}

func nativeLanguage(grammar parserGrammar) (*sitter.Language, error) {
	switch grammar {
	case parserGrammarCSharp:
		return sitter.NewLanguage(tree_sitter_csharp.Language()), nil
	case parserGrammarCPP:
		return sitter.NewLanguage(tree_sitter_cpp.Language()), nil
	case parserGrammarDart:
		return sitter.NewLanguage(tree_sitter_dart.Language()), nil
	case parserGrammarGo:
		return sitter.NewLanguage(tree_sitter_go.Language()), nil
	case parserGrammarJava:
		return sitter.NewLanguage(tree_sitter_java.Language()), nil
	case parserGrammarJavaScript:
		return sitter.NewLanguage(tree_sitter_javascript.Language()), nil
	case parserGrammarPython:
		return sitter.NewLanguage(tree_sitter_python.Language()), nil
	case parserGrammarRust:
		return sitter.NewLanguage(tree_sitter_rust.Language()), nil
	case parserGrammarTypeScript:
		return sitter.NewLanguage(tree_sitter_typescript.LanguageTypescript()), nil
	case parserGrammarTSX:
		return sitter.NewLanguage(tree_sitter_typescript.LanguageTSX()), nil
	case parserGrammarZig:
		return sitter.NewLanguage(tree_sitter_zig.Language()), nil
	default:
		return nil, fmt.Errorf("unsupported parser grammar %q", grammar)
	}
}
