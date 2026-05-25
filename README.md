# hsmc

`hsmc` is the monorepo-level HSM compiler. It translates HSM DSL structure between supported implementation languages while keeping model structure compiler-owned.

The compiler owns parsing, IR lowering, validation, signatures, structural imports, and target model emission. Coding-agent adapters own only mutable implementation code: behavior bodies, helper globals, and the imports needed by those translated code regions. If no adapter is used, foreign behavior code is preserved as target-language comments inside generated target behavior bodies, while foreign globals are preserved as target-language comments near the top-level output for manual porting.

The CLI entrypoint is in `cmd/hsmc`. See [cmd/hsmc/README.md](cmd/hsmc/README.md) for language support, adapter protocol details, and CLI usage.

## Install

For the alpha release, install from source with Go:

```sh
go install github.com/stateforward/hsmc/cmd/hsmc@v0.1.0-alpha.1
```

Native tree-sitter grammars currently require cgo. Release binaries are intentionally conservative until the parser layer moves to WASM grammars for no-cgo cross-platform builds.

## Verification

Run the normal compiler suite from `hsmc/`:

```sh
go test ./...
```

To verify generated target code against installed language toolchains, run the opt-in smoke suite:

```sh
HSMC_EXTERNAL_SMOKE=1 go test ./internal/hsmc -run 'TestExternalGeneratedTargetSmoke|TestExternalJSONIRSourceToGeneratedTargetsSmoke|TestExternalAdapterTargetSmoke|TestExternalSupportedSourcesToGoSmoke|TestExternalForeignSourcesToTypeScriptSmoke|TestExternalForeignSourcesToPythonSmoke|TestExternalForeignSourcesToJavaScriptSmoke|TestExternalForeignSourcesToDartSmoke|TestExternalForeignSourcesToCPPSmoke|TestExternalForeignSourcesToCSharpSmoke|TestExternalForeignSourcesToJavaSmoke|TestExternalForeignSourcesToZigSmoke|TestExternalForeignSourcesToRustSmoke|TestExternalTypeScriptSourceToTypeScriptSmoke|TestExternalPythonSourceToPythonSmoke|TestExternalJavaScriptSourceToJavaScriptSmoke|TestExternalDartSourceToDartSmoke|TestExternalZigSourceToZigSmoke|TestExternalCSharpSourceToCSharpSmoke|TestExternalJavaSourceToJavaSmoke|TestExternalCPPSourceToCPPSmoke|TestExternalRustSourceToRustSmoke' -count=1
```

The smoke suite compiles or checks generated Go, Python, JavaScript, TypeScript, C++, C#, Java, Dart, Zig, and Rust outputs when those tools are available, and includes JSON IR transport-to-target checks, no-adapter cross-source checks for generated Go and foreign-source output to TypeScript, Python, JavaScript, Dart, C++, C#, Java, Zig, and Rust, adapter-filled foreign-source-to-Go, foreign-source-to-Python, foreign-source-to-JavaScript, foreign-source-to-TypeScript, foreign-source-to-Dart, foreign-source-to-C++, foreign-source-to-C#, foreign-source-to-Java, foreign-source-to-Zig, and foreign-source-to-Rust behavior output, same-language TypeScript re-emission, same-language Python re-emission, same-language JavaScript re-emission, same-language Dart re-emission, same-language Zig re-emission, same-language C# re-emission, same-language Java re-emission, same-language C++ re-emission, and same-language Rust re-emission. Generated-Go smoke legs are skipped when the active Go toolchain is older than the sibling `hsm.go` runtime module's required Go version.
