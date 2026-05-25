# Changelog

## Unreleased

- Adds recursive directory compilation for project/directory inputs, preserving relative paths while converting every supported source file to the requested target language.
- Adds Mermaid and PlantUML target backends for generating state diagrams from HSM source code.
- Preserves captured behavior and global source as diagram notes for manual review and porting context.

## v0.1.0-alpha.1

Initial alpha release candidate.

- Adds `hsmc`, a Go-based structural HSM compiler.
- Supports tree-sitter frontends for C#, C++, Dart, Go, Java, JavaScript, Python, Rust, TypeScript, and Zig.
- Supports targets for C#, C++, Dart, Go, Java, JavaScript, Python, Rust, TypeScript, Zig, and JSON IR.
- Preserves untranslated foreign behavior as target-language comments in generated behavior bodies.
- Preserves untranslated foreign globals as target-language comments near top-level output.
- Adds Codex and command adapter protocols for translating behavior/global code and target imports without model edits.
