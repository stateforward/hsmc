package hsmc

import (
	"context"
	"strings"
	"testing"
)

func TestNativeLanguageReturnsErrorForUnsupportedGrammar(t *testing.T) {
	if _, err := nativeLanguage(parserGrammar("ruby")); err == nil || !strings.Contains(err.Error(), `unsupported parser grammar "ruby"`) {
		t.Fatalf("nativeLanguage(ruby) error = %v, want unsupported grammar", err)
	}
}

func TestParseTreeReturnsErrorForUnsupportedGrammar(t *testing.T) {
	tree, err := parseTree(context.Background(), SourceInput{Path: "sample.rb", Data: []byte("puts 'closed'")}, parserGrammar("ruby"))
	if err == nil || !strings.Contains(err.Error(), `unsupported parser grammar "ruby"`) {
		t.Fatalf("parseTree unsupported grammar error = %v, want unsupported grammar", err)
	}
	if tree != nil {
		t.Fatalf("parseTree unsupported grammar returned tree %#v", tree)
	}
}

func TestNativeLanguagesResolveRegisteredGrammars(t *testing.T) {
	for _, grammar := range []parserGrammar{parserGrammarCSharp, parserGrammarCPP, parserGrammarDart, parserGrammarGo, parserGrammarJava, parserGrammarJavaScript, parserGrammarPython, parserGrammarRust, parserGrammarTypeScript, parserGrammarTSX, parserGrammarZig} {
		language, err := nativeLanguage(grammar)
		if err != nil {
			t.Fatalf("nativeLanguage(%s) error = %v", grammar, err)
		}
		if language == nil {
			t.Fatalf("nativeLanguage(%s) returned nil language", grammar)
		}
	}
}

func TestParseTreeRejectsSyntaxErrorsForRegisteredGrammars(t *testing.T) {
	for _, tc := range []struct {
		grammar parserGrammar
		path    string
		source  string
	}{
		{grammar: parserGrammarCSharp, path: "broken.cs", source: "class Broken { void M( }"},
		{grammar: parserGrammarCPP, path: "broken.cpp", source: "int main( {"},
		{grammar: parserGrammarDart, path: "broken.dart", source: "void main( {"},
		{grammar: parserGrammarGo, path: "broken.go", source: "package main\nfunc main( {"},
		{grammar: parserGrammarJava, path: "Broken.java", source: "class Broken { void main( }"},
		{grammar: parserGrammarJavaScript, path: "broken.js", source: "function main( {"},
		{grammar: parserGrammarPython, path: "broken.py", source: "def main(:\n    pass"},
		{grammar: parserGrammarRust, path: "broken.rs", source: "fn main( {"},
		{grammar: parserGrammarTypeScript, path: "broken.ts", source: "function main( {"},
		{grammar: parserGrammarTSX, path: "broken.tsx", source: "function main( {"},
		{grammar: parserGrammarZig, path: "broken.zig", source: "pub fn main( {"},
	} {
		t.Run(string(tc.grammar), func(t *testing.T) {
			tree, err := parseTree(context.Background(), SourceInput{Path: tc.path, Data: []byte(tc.source)}, tc.grammar)
			if err == nil || !strings.Contains(err.Error(), "parse errors in "+tc.path) {
				t.Fatalf("parseTree(%s) error = %v, want syntax error", tc.grammar, err)
			}
			if tree != nil {
				t.Fatalf("parseTree(%s) returned tree %#v for invalid source", tc.grammar, tree)
			}
		})
	}
}
