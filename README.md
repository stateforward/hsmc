# hsmc

`hsmc` is the StateForward HSM compiler.

It converts StateForward HSM definitions between implementation languages while keeping the state machine model deterministic and reviewable. The compiler translates the HSM structure. Coding-agent adapters, such as Codex, can translate the host-language behavior bodies.

```text
source HSM code -> tree-sitter frontend -> HSM IR -> validator -> target backend
                                                   -> optional behavior adapter
```

## Why

State machines often outlive the language they were first written in. `hsmc` is built for moving those models across runtimes without turning the model itself into an AI-generated guess.

The compiler owns the parts that need to be exact:

- models, states, pseudostates, transitions, triggers, events, defers, attributes, and operation references
- behavior names, signatures, and target ABI
- runtime scaffolding and compiler-owned imports
- validation before and after adapter patches

Adapters own the parts that are inherently language-specific:

- behavior bodies
- helper/global code used by those bodies
- target-language imports needed by translated code

If you do not use an adapter, `hsmc` still emits the target HSM. Foreign behavior is preserved as target-language comments inside the generated target behavior body, and foreign globals are preserved as comments near the top of the output so you can port them manually.

## Status

`hsmc` is in alpha.

The compiler and adapter boundary are covered by the Go test suite, and the release workflow publishes Linux amd64 binaries. Native tree-sitter grammars currently require cgo; broader no-cgo cross-platform binaries are planned with a WASM tree-sitter parser layer.

## Install

Install with Go:

```sh
go install github.com/stateforward/hsmc/cmd/hsmc@v0.1.0-alpha.1
```

Or download a release artifact from:

```text
https://github.com/stateforward/hsmc/releases
```

Check the installed version:

```sh
hsmc -version
```

## Quick Start

Convert a Go HSM definition to TypeScript:

```sh
hsmc -from go -to typescript -in door.go -out door.ts
```

Infer languages from file extensions:

```sh
hsmc -in door.py -out door.go
```

Emit JSON IR:

```sh
hsmc -from typescript -to json-ir -in door.ts -out door.hsm.json
```

Generate a diagram from source code:

```sh
hsmc -from go -to mermaid -in door.go -out door.mmd
hsmc -from go -to plantuml -in door.go -out door.puml
```

Pipe from stdin:

```sh
cat door.go | hsmc -from go -to python -in -
```

List supported languages:

```sh
hsmc -list-source-languages
hsmc -list-target-languages
```

## Using Codex For Behavior Porting

Without an adapter, untranslated behavior is commented in place:

```python
async def entry_1(ctx, instance, event) -> None:
    # Original go behavior entry_1 preserved for manual porting:
    # instance.Log = append(instance.Log, "entered")
    return None
```

With the Codex adapter, `hsmc` asks Codex to translate only editable behavior/global code and target imports:

```sh
hsmc \
  -from go \
  -to python \
  -adapter codex \
  -adapter-command codex \
  -in door.go \
  -out door.py
```

The compiler rejects adapter patches that try to alter compiler-owned model structure, including models, events, states, transitions, behavior IDs, signatures, source languages, and target ABI.

For custom integrations, use `-adapter command` with any executable that speaks the same JSON patch protocol.

## Supported Languages

Source languages:

- C#
- C++
- Dart
- Go
- Java
- JavaScript
- Python
- Rust
- TypeScript
- Zig
- JSON IR

Target languages:

- C#
- C++
- Dart
- Go
- Java
- JavaScript
- Python
- Rust
- TypeScript
- Zig
- JSON IR
- Mermaid
- PlantUML

## Design Boundary

`hsmc` is a structural HSM compiler, not a general-purpose semantic compiler for arbitrary host-language code.

That boundary is intentional. The model is compiled deterministically; behavior bodies are either preserved for manual porting or delegated to an adapter with a narrow, validated patch surface.

Runtime/helper declarations such as `Config`, `Queue`, `Clock`, `MakeGroup`, `MakeKind`, `IsKind`, and `TakeSnapshot` are treated as adapter-owned global context rather than model IR. They can be translated by an adapter or preserved as comments for manual porting.

Diagram targets are adapter-free. Mermaid and PlantUML output includes state hierarchy, initial transitions, transition labels, and notes containing captured behavior/global source so generated diagrams carry the implementation context from the original code.

## Development

Run the normal test suite:

```sh
go test ./...
```

Run vet:

```sh
go vet ./...
```

Run generated-target smoke tests against installed language toolchains:

```sh
HSMC_EXTERNAL_SMOKE=1 go test ./internal/hsmc -run 'External' -count=1
```

When running the external smoke suite from the standalone repository, set `HSMC_MONOREPO_ROOT` if the tests need sibling runtime repositories:

```sh
HSMC_MONOREPO_ROOT=/path/to/hsm HSMC_EXTERNAL_SMOKE=1 go test ./internal/hsmc -run 'External' -count=1
```

## More Detail

See [cmd/hsmc/README.md](cmd/hsmc/README.md) for the full CLI reference, adapter protocol, import rules, and requirement/test map.
