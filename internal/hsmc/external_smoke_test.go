package hsmc

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestExternalGeneratedTargetSmoke(t *testing.T) {
	if os.Getenv("HSMC_EXTERNAL_SMOKE") != "1" {
		t.Skip("set HSMC_EXTERNAL_SMOKE=1 to run generated target toolchain checks")
	}
	if runtime.GOOS == "windows" {
		t.Skip("external smoke paths use Unix-style shell-free local paths")
	}

	root := findMonorepoRoot(t)
	tmp := t.TempDir()
	source := SourceInput{Path: "door.go", Data: []byte(goDoorSource)}
	compiler := NewCompiler()

	outputs := map[Language]string{
		LanguageGo:     filepath.Join(tmp, "go", "door.go"),
		LanguagePython: filepath.Join(tmp, "python", "door.py"),
		LanguageCPP:    filepath.Join(tmp, "cpp", "door.cpp"),
		LanguageCSharp: filepath.Join(tmp, "csharp", "GeneratedHsm.cs"),
		LanguageDart:   filepath.Join(tmp, "dart", "lib", "door.dart"),
		LanguageJava:   filepath.Join(tmp, "java", "GeneratedHsm.java"),
		LanguageZig:    filepath.Join(tmp, "zig", "door.zig"),
		LanguageRust:   filepath.Join(tmp, "rust", "src", "lib.rs"),
		LanguageJS:     filepath.Join(tmp, "javascript", "door.js"),
		LanguageTS:     filepath.Join(tmp, "typescript", "door.ts"),
	}
	for language, path := range outputs {
		writeCompiledTarget(t, compiler, source, language, path)
	}

	t.Run("go", func(t *testing.T) {
		requireGoSmokeToolchain(t, root)
		dir := filepath.Dir(outputs[LanguageGo])
		writeGoSmokeMod(t, root, dir)
		runTool(t, dir, "go", "mod", "tidy")
		runTool(t, dir, "go", "test", "./...")

		richDir := filepath.Join(tmp, "go-rich")
		writeCompiledTarget(t, compiler, SourceInput{Path: "rich.go", Data: []byte(operationSmokeGoSource)}, LanguageGo, filepath.Join(richDir, "rich.go"))
		writeGoSmokeMod(t, root, richDir)
		runTool(t, richDir, "go", "mod", "tidy")
		runTool(t, richDir, "go", "test", "./...")

		timerDir := filepath.Join(tmp, "go-timer")
		writeCompiledTarget(t, compiler, SourceInput{Path: "timer.go", Data: []byte(timerSmokeGoSource)}, LanguageGo, filepath.Join(timerDir, "timer.go"))
		writeGoSmokeMod(t, root, timerDir)
		runTool(t, timerDir, "go", "mod", "tidy")
		runTool(t, timerDir, "go", "test", "./...")

		everyDir := filepath.Join(tmp, "go-every")
		writeCompiledTarget(t, compiler, SourceInput{Path: "every.go", Data: []byte(everySmokeGoSource)}, LanguageGo, filepath.Join(everyDir, "every.go"))
		writeGoSmokeMod(t, root, everyDir)
		runTool(t, everyDir, "go", "mod", "tidy")
		runTool(t, everyDir, "go", "test", "./...")

		whenDir := filepath.Join(tmp, "go-when")
		writeCompiledTarget(t, compiler, SourceInput{Path: "when.go", Data: []byte(attributeWhenSmokeGoSource)}, LanguageGo, filepath.Join(whenDir, "when.go"))
		writeGoSmokeMod(t, root, whenDir)
		runTool(t, whenDir, "go", "mod", "tidy")
		runTool(t, whenDir, "go", "test", "./...")

		predicateWhenDir := filepath.Join(tmp, "go-predicate-when")
		writeCompiledTarget(t, compiler, SourceInput{Path: "predicate_when.go", Data: []byte(predicateWhenSmokeGoSource)}, LanguageGo, filepath.Join(predicateWhenDir, "predicate_when.go"))
		writeGoSmokeMod(t, root, predicateWhenDir)
		runTool(t, predicateWhenDir, "go", "mod", "tidy")
		runTool(t, predicateWhenDir, "go", "test", "./...")
	})

	t.Run("python", func(t *testing.T) {
		dir := filepath.Dir(outputs[LanguagePython])
		writeCompiledTarget(t, compiler, SourceInput{Path: "rich.go", Data: []byte(operationSmokeGoSource)}, LanguagePython, filepath.Join(dir, "rich.py"))
		writeCompiledTarget(t, compiler, SourceInput{Path: "timer.go", Data: []byte(timerSmokeGoSource)}, LanguagePython, filepath.Join(dir, "timer.py"))
		writeCompiledTarget(t, compiler, SourceInput{Path: "when.go", Data: []byte(attributeWhenSmokeGoSource)}, LanguagePython, filepath.Join(dir, "when.py"))
		writeCompiledTarget(t, compiler, SourceInput{Path: "predicate_when.go", Data: []byte(predicateWhenSmokeGoSource)}, LanguagePython, filepath.Join(dir, "predicate_when.py"))
		runTool(t, tmp, "python3", "-m", "py_compile", outputs[LanguagePython], filepath.Join(dir, "rich.py"), filepath.Join(dir, "timer.py"), filepath.Join(dir, "when.py"), filepath.Join(dir, "predicate_when.py"))
		runToolWithEnv(t, dir, []string{"PYTHONPATH=" + filepath.Join(root, "hsm.py")}, "python3", filepath.Base(outputs[LanguagePython]))
		runToolWithEnv(t, dir, []string{"PYTHONPATH=" + filepath.Join(root, "hsm.py")}, "python3", "rich.py")
		runToolWithEnv(t, dir, []string{"PYTHONPATH=" + filepath.Join(root, "hsm.py")}, "python3", "when.py")
		runToolWithEnv(t, dir, []string{"PYTHONPATH=" + filepath.Join(root, "hsm.py")}, "python3", "predicate_when.py")
	})

	t.Run("javascript", func(t *testing.T) {
		dir := filepath.Dir(outputs[LanguageJS])
		writeCompiledTarget(t, compiler, SourceInput{Path: "rich.go", Data: []byte(operationSmokeGoSource)}, LanguageJS, filepath.Join(dir, "rich.js"))
		writeCompiledTarget(t, compiler, SourceInput{Path: "timer.go", Data: []byte(timerSmokeGoSource)}, LanguageJS, filepath.Join(dir, "timer.js"))
		writeCompiledTarget(t, compiler, SourceInput{Path: "when.go", Data: []byte(attributeWhenSmokeGoSource)}, LanguageJS, filepath.Join(dir, "when.js"))
		writeCompiledTarget(t, compiler, SourceInput{Path: "predicate_when.go", Data: []byte(predicateWhenSmokeGoSource)}, LanguageJS, filepath.Join(dir, "predicate_when.js"))
		writeFile(t, filepath.Join(dir, "package.json"), `{"type":"module"}`+"\n")
		linkNodePackage(t, dir, "@stateforward", "hsm", filepath.Join(root, "hsm.js"))
		runTool(t, dir, "node", filepath.Base(outputs[LanguageJS]))
		runTool(t, dir, "node", "rich.js")
		runTool(t, dir, "node", "timer.js")
		runTool(t, dir, "node", "when.js")
		runTool(t, dir, "node", "predicate_when.js")
	})

	t.Run("typescript", func(t *testing.T) {
		dir := filepath.Dir(outputs[LanguageTS])
		writeCompiledTarget(t, compiler, SourceInput{Path: "rich.go", Data: []byte(operationSmokeGoSource)}, LanguageTS, filepath.Join(dir, "rich.ts"))
		writeCompiledTarget(t, compiler, SourceInput{Path: "timer.go", Data: []byte(timerSmokeGoSource)}, LanguageTS, filepath.Join(dir, "timer.ts"))
		writeCompiledTarget(t, compiler, SourceInput{Path: "when.go", Data: []byte(attributeWhenSmokeGoSource)}, LanguageTS, filepath.Join(dir, "when.ts"))
		writeCompiledTarget(t, compiler, SourceInput{Path: "predicate_when.go", Data: []byte(predicateWhenSmokeGoSource)}, LanguageTS, filepath.Join(dir, "predicate_when.ts"))
		writeFile(t, filepath.Join(dir, "package.json"), `{"type":"module"}`+"\n")
		linkNodePackage(t, dir, "@stateforward", "hsm.ts", filepath.Join(root, "hsm.ts"))
		writeFile(t, filepath.Join(dir, "tsconfig.json"), strings.Join([]string{
			"{",
			`  "compilerOptions": {`,
			`    "target": "ES2022",`,
			`    "module": "NodeNext",`,
			`    "moduleResolution": "NodeNext",`,
			`    "strict": true,`,
			`    "skipLibCheck": true,`,
			`    "noEmit": true,`,
			`    "types": ["node"],`,
			`    "typeRoots": ["` + filepath.ToSlash(filepath.Join(root, "hsm.ts", "node_modules", "@types")) + `"]`,
			`  },`,
			`  "include": ["door.ts", "rich.ts", "timer.ts", "when.ts", "predicate_when.ts"]`,
			"}",
			"",
		}, "\n"))
		runTool(t, dir, filepath.Join(root, "hsm.ts", "node_modules", ".bin", "tsc"), "-p", "tsconfig.json", "--pretty", "false")
	})

	t.Run("cpp", func(t *testing.T) {
		dir := filepath.Dir(outputs[LanguageCPP])
		writeCompiledTarget(t, compiler, SourceInput{Path: "rich.go", Data: []byte(operationSmokeGoSource)}, LanguageCPP, filepath.Join(dir, "rich.cpp"))
		writeCompiledTarget(t, compiler, SourceInput{Path: "timer.go", Data: []byte(timerSmokeGoSource)}, LanguageCPP, filepath.Join(dir, "timer.cpp"))
		writeCompiledTarget(t, compiler, SourceInput{Path: "when.go", Data: []byte(attributeWhenSmokeGoSource)}, LanguageCPP, filepath.Join(dir, "when.cpp"))
		writeCompiledTarget(t, compiler, SourceInput{Path: "predicate_when.go", Data: []byte(predicateWhenSmokeGoSource)}, LanguageCPP, filepath.Join(dir, "predicate_when.cpp"))
		runTool(t, tmp, "clang++", "-std=c++20", "-I"+filepath.Join(root, "hsm.cpp", "include"), "-fsyntax-only", outputs[LanguageCPP], filepath.Join(dir, "rich.cpp"), filepath.Join(dir, "timer.cpp"), filepath.Join(dir, "when.cpp"), filepath.Join(dir, "predicate_when.cpp"))
	})

	t.Run("csharp", func(t *testing.T) {
		dir := filepath.Dir(outputs[LanguageCSharp])
		writeFile(t, filepath.Join(dir, "Check.csproj"), `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net10.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
    <LangVersion>latest</LangVersion>
  </PropertyGroup>
  <ItemGroup>
    <ProjectReference Include="`+filepath.Join(root, "hsm.cs", "Stateforward.Hsm.csproj")+`" />
  </ItemGroup>
</Project>
`)
		runTool(t, dir, "dotnet", "build", "Check.csproj", "--nologo", "-v:minimal")

		richDir := filepath.Join(tmp, "csharp-rich")
		writeCompiledTarget(t, compiler, SourceInput{Path: "rich.go", Data: []byte(operationSmokeGoSource)}, LanguageCSharp, filepath.Join(richDir, "GeneratedHsm.cs"))
		writeFile(t, filepath.Join(richDir, "Check.csproj"), `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net10.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
    <LangVersion>latest</LangVersion>
  </PropertyGroup>
  <ItemGroup>
    <ProjectReference Include="`+filepath.Join(root, "hsm.cs", "Stateforward.Hsm.csproj")+`" />
  </ItemGroup>
</Project>
`)
		runTool(t, richDir, "dotnet", "build", "Check.csproj", "--nologo", "-v:minimal")

		timerDir := filepath.Join(tmp, "csharp-timer")
		writeCompiledTarget(t, compiler, SourceInput{Path: "timer.go", Data: []byte(timerSmokeGoSource)}, LanguageCSharp, filepath.Join(timerDir, "GeneratedHsm.cs"))
		writeFile(t, filepath.Join(timerDir, "Check.csproj"), `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net10.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
    <LangVersion>latest</LangVersion>
  </PropertyGroup>
  <ItemGroup>
    <ProjectReference Include="`+filepath.Join(root, "hsm.cs", "Stateforward.Hsm.csproj")+`" />
  </ItemGroup>
</Project>
`)
		runTool(t, timerDir, "dotnet", "build", "Check.csproj", "--nologo", "-v:minimal")

		whenDir := filepath.Join(tmp, "csharp-when")
		writeCompiledTarget(t, compiler, SourceInput{Path: "when.go", Data: []byte(attributeWhenSmokeGoSource)}, LanguageCSharp, filepath.Join(whenDir, "GeneratedHsm.cs"))
		writeFile(t, filepath.Join(whenDir, "Check.csproj"), `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net10.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
    <LangVersion>latest</LangVersion>
  </PropertyGroup>
  <ItemGroup>
    <ProjectReference Include="`+filepath.Join(root, "hsm.cs", "Stateforward.Hsm.csproj")+`" />
  </ItemGroup>
</Project>
`)
		runTool(t, whenDir, "dotnet", "build", "Check.csproj", "--nologo", "-v:minimal")

		predicateWhenDir := filepath.Join(tmp, "csharp-predicate-when")
		writeCompiledTarget(t, compiler, SourceInput{Path: "predicate_when.go", Data: []byte(predicateWhenSmokeGoSource)}, LanguageCSharp, filepath.Join(predicateWhenDir, "GeneratedHsm.cs"))
		writeFile(t, filepath.Join(predicateWhenDir, "Check.csproj"), `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net10.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
    <LangVersion>latest</LangVersion>
  </PropertyGroup>
  <ItemGroup>
    <ProjectReference Include="`+filepath.Join(root, "hsm.cs", "Stateforward.Hsm.csproj")+`" />
  </ItemGroup>
</Project>
`)
		runTool(t, predicateWhenDir, "dotnet", "build", "Check.csproj", "--nologo", "-v:minimal")
	})

	t.Run("java", func(t *testing.T) {
		dir := filepath.Dir(outputs[LanguageJava])
		runJavaSmoke(t, dir, outputs[LanguageJava])

		richDir := filepath.Join(tmp, "java-rich")
		richPath := filepath.Join(richDir, "GeneratedHsm.java")
		writeCompiledTarget(t, compiler, SourceInput{Path: "rich.go", Data: []byte(operationSmokeGoSource)}, LanguageJava, richPath)
		runJavaSmoke(t, richDir, richPath)

		timerDir := filepath.Join(tmp, "java-timer")
		timerPath := filepath.Join(timerDir, "GeneratedHsm.java")
		writeCompiledTarget(t, compiler, SourceInput{Path: "timer.go", Data: []byte(timerSmokeGoSource)}, LanguageJava, timerPath)
		runJavaSmoke(t, timerDir, timerPath)

		whenDir := filepath.Join(tmp, "java-when")
		whenPath := filepath.Join(whenDir, "GeneratedHsm.java")
		writeCompiledTarget(t, compiler, SourceInput{Path: "when.go", Data: []byte(attributeWhenSmokeGoSource)}, LanguageJava, whenPath)
		runJavaSmoke(t, whenDir, whenPath)

		predicateWhenDir := filepath.Join(tmp, "java-predicate-when")
		predicateWhenPath := filepath.Join(predicateWhenDir, "GeneratedHsm.java")
		writeCompiledTarget(t, compiler, SourceInput{Path: "predicate_when.go", Data: []byte(predicateWhenSmokeGoSource)}, LanguageJava, predicateWhenPath)
		runJavaSmoke(t, predicateWhenDir, predicateWhenPath)
	})

	t.Run("dart", func(t *testing.T) {
		dir := filepath.Dir(filepath.Dir(outputs[LanguageDart]))
		writeCompiledTarget(t, compiler, SourceInput{Path: "rich.go", Data: []byte(operationSmokeGoSource)}, LanguageDart, filepath.Join(dir, "lib", "rich.dart"))
		writeCompiledTarget(t, compiler, SourceInput{Path: "timer.go", Data: []byte(timerSmokeGoSource)}, LanguageDart, filepath.Join(dir, "lib", "timer.dart"))
		writeCompiledTarget(t, compiler, SourceInput{Path: "every.go", Data: []byte(everySmokeGoSource)}, LanguageDart, filepath.Join(dir, "lib", "every.dart"))
		writeCompiledTarget(t, compiler, SourceInput{Path: "when.go", Data: []byte(attributeWhenSmokeGoSource)}, LanguageDart, filepath.Join(dir, "lib", "when.dart"))
		writeCompiledTarget(t, compiler, SourceInput{Path: "predicate_when.go", Data: []byte(predicateWhenSmokeGoSource)}, LanguageDart, filepath.Join(dir, "lib", "predicate_when.dart"))
		writeCompiledTarget(t, compiler, SourceInput{Path: "pseudo.go", Data: []byte(goPseudostateSource)}, LanguageDart, filepath.Join(dir, "lib", "pseudo.dart"))
		writeFile(t, filepath.Join(dir, "pubspec.yaml"), strings.Join([]string{
			"name: hsmc_smoke",
			"environment:",
			"  sdk: '>=3.0.0 <4.0.0'",
			"dependencies:",
			"  hsm:",
			"    path: " + filepath.Join(root, "hsm.dart"),
			"",
		}, "\n"))
		runTool(t, dir, "dart", "pub", "get")
		runTool(t, dir, "dart", "analyze")
	})

	t.Run("zig", func(t *testing.T) {
		richPath := filepath.Join(tmp, "zig", "rich.zig")
		timerPath := filepath.Join(tmp, "zig", "timer.zig")
		whenPath := filepath.Join(tmp, "zig", "when.zig")
		predicateWhenPath := filepath.Join(tmp, "zig", "predicate_when.zig")
		writeCompiledTarget(t, compiler, SourceInput{Path: "rich.go", Data: []byte(operationSmokeGoSource)}, LanguageZig, richPath)
		writeCompiledTarget(t, compiler, SourceInput{Path: "timer.go", Data: []byte(timerSmokeGoSource)}, LanguageZig, timerPath)
		writeCompiledTarget(t, compiler, SourceInput{Path: "when.go", Data: []byte(attributeWhenSmokeGoSource)}, LanguageZig, whenPath)
		writeCompiledTarget(t, compiler, SourceInput{Path: "predicate_when.go", Data: []byte(predicateWhenSmokeGoSource)}, LanguageZig, predicateWhenPath)
		runTool(t, tmp, "zig", "test", "--dep", "hsm", "-Mroot="+outputs[LanguageZig], "-Mhsm="+filepath.Join(root, "hsm.zig", "src", "hsm.zig"))
		runTool(t, tmp, "zig", "test", "--dep", "hsm", "-Mroot="+richPath, "-Mhsm="+filepath.Join(root, "hsm.zig", "src", "hsm.zig"))
		runTool(t, tmp, "zig", "test", "--dep", "hsm", "-Mroot="+timerPath, "-Mhsm="+filepath.Join(root, "hsm.zig", "src", "hsm.zig"))
		runTool(t, tmp, "zig", "test", "--dep", "hsm", "-Mroot="+whenPath, "-Mhsm="+filepath.Join(root, "hsm.zig", "src", "hsm.zig"))
		runTool(t, tmp, "zig", "test", "--dep", "hsm", "-Mroot="+predicateWhenPath, "-Mhsm="+filepath.Join(root, "hsm.zig", "src", "hsm.zig"))
	})

	t.Run("rust", func(t *testing.T) {
		dir := filepath.Dir(filepath.Dir(outputs[LanguageRust]))
		doorModule := filepath.Join(dir, "src", "door.rs")
		if err := os.Rename(outputs[LanguageRust], doorModule); err != nil {
			t.Fatal(err)
		}
		writeCompiledTarget(t, compiler, SourceInput{Path: "rich.go", Data: []byte(operationSmokeGoSource)}, LanguageRust, filepath.Join(dir, "src", "rich.rs"))
		writeCompiledTarget(t, compiler, SourceInput{Path: "timer.go", Data: []byte(timerSmokeGoSource)}, LanguageRust, filepath.Join(dir, "src", "timer.rs"))
		writeCompiledTarget(t, compiler, SourceInput{Path: "when.go", Data: []byte(attributeWhenSmokeGoSource)}, LanguageRust, filepath.Join(dir, "src", "when.rs"))
		writeCompiledTarget(t, compiler, SourceInput{Path: "predicate_when.go", Data: []byte(predicateWhenSmokeGoSource)}, LanguageRust, filepath.Join(dir, "src", "predicate_when.rs"))
		writeFile(t, outputs[LanguageRust], "mod door;\nmod rich;\nmod timer;\nmod when;\nmod predicate_when;\n")
		writeFile(t, filepath.Join(dir, "Cargo.toml"), strings.Join([]string{
			"[package]",
			`name = "hsmc_smoke"`,
			`version = "0.1.0"`,
			`edition = "2024"`,
			"",
			"[dependencies]",
			`hsm = { package = "rust", path = "` + filepath.ToSlash(filepath.Join(root, "hsm.rs")) + `" }`,
			"",
		}, "\n"))
		runTool(t, dir, "cargo", "check")
	})
}

func TestExternalSupportedSourcesToGoSmoke(t *testing.T) {
	if os.Getenv("HSMC_EXTERNAL_SMOKE") != "1" {
		t.Skip("set HSMC_EXTERNAL_SMOKE=1 to run generated target toolchain checks")
	}
	if runtime.GOOS == "windows" {
		t.Skip("external smoke paths use Unix-style shell-free local paths")
	}

	root := findMonorepoRoot(t)
	requireGoSmokeToolchain(t, root)
	tmp := t.TempDir()
	compiler := NewCompiler()

	for _, source := range matrixSources(t, compiler) {
		source := source
		t.Run(source.name, func(t *testing.T) {
			dir := filepath.Join(tmp, source.name)
			writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
				From:    source.language,
				To:      LanguageGo,
				Adapter: "none",
			}, filepath.Join(dir, "door.go"))
			writeGoSmokeMod(t, root, dir)
			runTool(t, dir, "go", "mod", "tidy")
			runTool(t, dir, "go", "test", "./...")
		})
	}
}

func TestExternalJSONIRSourceToGeneratedTargetsSmoke(t *testing.T) {
	if os.Getenv("HSMC_EXTERNAL_SMOKE") != "1" {
		t.Skip("set HSMC_EXTERNAL_SMOKE=1 to run generated target toolchain checks")
	}
	if runtime.GOOS == "windows" {
		t.Skip("external smoke paths use Unix-style shell-free local paths")
	}

	root := findMonorepoRoot(t)
	tmp := t.TempDir()
	compiler := NewCompiler()
	ir, _, err := compiler.Compile(context.Background(), SourceInput{Path: "door.go", Data: []byte(goDoorSource)}, CompileOptions{
		From:    LanguageGo,
		To:      LanguageJSONIR,
		Adapter: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	source := SourceInput{Path: "door.hsm.json", Data: ir}
	outputs := map[Language]string{
		LanguageGo:     filepath.Join(tmp, "go", "door.go"),
		LanguagePython: filepath.Join(tmp, "python", "door.py"),
		LanguageCPP:    filepath.Join(tmp, "cpp", "door.cpp"),
		LanguageCSharp: filepath.Join(tmp, "csharp", "GeneratedHsm.cs"),
		LanguageDart:   filepath.Join(tmp, "dart", "lib", "door.dart"),
		LanguageJava:   filepath.Join(tmp, "java", "GeneratedHsm.java"),
		LanguageZig:    filepath.Join(tmp, "zig", "door.zig"),
		LanguageRust:   filepath.Join(tmp, "rust", "src", "lib.rs"),
		LanguageJS:     filepath.Join(tmp, "javascript", "door.js"),
		LanguageTS:     filepath.Join(tmp, "typescript", "door.ts"),
	}
	for language, path := range outputs {
		writeCompiledTargetWithOptions(t, compiler, source, CompileOptions{
			From:    LanguageJSONIR,
			To:      language,
			Adapter: "none",
		}, path)
	}

	t.Run("go", func(t *testing.T) {
		requireGoSmokeToolchain(t, root)
		dir := filepath.Dir(outputs[LanguageGo])
		writeGoSmokeMod(t, root, dir)
		runTool(t, dir, "go", "mod", "tidy")
		runTool(t, dir, "go", "test", "./...")
	})

	t.Run("python", func(t *testing.T) {
		dir := filepath.Dir(outputs[LanguagePython])
		runTool(t, dir, "python3", "-m", "py_compile", filepath.Base(outputs[LanguagePython]))
		runToolWithEnv(t, dir, []string{"PYTHONPATH=" + filepath.Join(root, "hsm.py")}, "python3", filepath.Base(outputs[LanguagePython]))
	})

	t.Run("javascript", func(t *testing.T) {
		dir := filepath.Dir(outputs[LanguageJS])
		writeFile(t, filepath.Join(dir, "package.json"), `{"type":"module"}`+"\n")
		linkNodePackage(t, dir, "@stateforward", "hsm", filepath.Join(root, "hsm.js"))
		runTool(t, dir, "node", filepath.Base(outputs[LanguageJS]))
	})

	t.Run("typescript", func(t *testing.T) {
		dir := filepath.Dir(outputs[LanguageTS])
		writeFile(t, filepath.Join(dir, "package.json"), `{"type":"module"}`+"\n")
		linkNodePackage(t, dir, "@stateforward", "hsm.ts", filepath.Join(root, "hsm.ts"))
		writeFile(t, filepath.Join(dir, "tsconfig.json"), strings.Join([]string{
			"{",
			`  "compilerOptions": {`,
			`    "target": "ES2022",`,
			`    "module": "NodeNext",`,
			`    "moduleResolution": "NodeNext",`,
			`    "strict": true,`,
			`    "skipLibCheck": true,`,
			`    "noEmit": true,`,
			`    "types": ["node"],`,
			`    "typeRoots": ["` + filepath.ToSlash(filepath.Join(root, "hsm.ts", "node_modules", "@types")) + `"]`,
			`  },`,
			`  "include": ["door.ts"]`,
			"}",
			"",
		}, "\n"))
		runTool(t, dir, filepath.Join(root, "hsm.ts", "node_modules", ".bin", "tsc"), "-p", "tsconfig.json", "--pretty", "false")
	})

	t.Run("cpp", func(t *testing.T) {
		runTool(t, tmp, "clang++", "-std=c++20", "-I"+filepath.Join(root, "hsm.cpp", "include"), "-fsyntax-only", outputs[LanguageCPP])
	})

	t.Run("csharp", func(t *testing.T) {
		dir := filepath.Dir(outputs[LanguageCSharp])
		writeFile(t, filepath.Join(dir, "Check.csproj"), `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net10.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
    <LangVersion>latest</LangVersion>
  </PropertyGroup>
  <ItemGroup>
    <ProjectReference Include="`+filepath.Join(root, "hsm.cs", "Stateforward.Hsm.csproj")+`" />
  </ItemGroup>
</Project>
`)
		runTool(t, dir, "dotnet", "build", "Check.csproj", "--nologo", "-v:minimal")
	})

	t.Run("java", func(t *testing.T) {
		dir := filepath.Dir(outputs[LanguageJava])
		runJavaSmoke(t, dir, outputs[LanguageJava])
	})

	t.Run("dart", func(t *testing.T) {
		dir := filepath.Dir(filepath.Dir(outputs[LanguageDart]))
		writeFile(t, filepath.Join(dir, "pubspec.yaml"), strings.Join([]string{
			"name: hsmc_json_ir_smoke",
			"environment:",
			"  sdk: '>=3.0.0 <4.0.0'",
			"dependencies:",
			"  hsm:",
			"    path: " + filepath.Join(root, "hsm.dart"),
			"",
		}, "\n"))
		runTool(t, dir, "dart", "pub", "get")
		runTool(t, dir, "dart", "analyze")
	})

	t.Run("zig", func(t *testing.T) {
		runTool(t, tmp, "zig", "test", "--dep", "hsm", "-Mroot="+outputs[LanguageZig], "-Mhsm="+filepath.Join(root, "hsm.zig", "src", "hsm.zig"))
	})

	t.Run("rust", func(t *testing.T) {
		dir := filepath.Dir(filepath.Dir(outputs[LanguageRust]))
		writeFile(t, filepath.Join(dir, "Cargo.toml"), strings.Join([]string{
			"[package]",
			`name = "hsmc_json_ir_smoke"`,
			`version = "0.1.0"`,
			`edition = "2024"`,
			"",
			"[dependencies]",
			`hsm = { package = "rust", path = "` + filepath.ToSlash(filepath.Join(root, "hsm.rs")) + `" }`,
			"",
		}, "\n"))
		runTool(t, dir, "cargo", "check")
	})
}

func TestExternalForeignSourcesToTypeScriptSmoke(t *testing.T) {
	if os.Getenv("HSMC_EXTERNAL_SMOKE") != "1" {
		t.Skip("set HSMC_EXTERNAL_SMOKE=1 to run generated target toolchain checks")
	}
	if runtime.GOOS == "windows" {
		t.Skip("external smoke paths use Unix-style shell-free local paths")
	}

	root := findMonorepoRoot(t)
	dir := t.TempDir()
	compiler := NewCompiler()
	include := make([]string, 0)
	for _, source := range matrixSources(t, compiler) {
		if source.language == LanguageTS {
			continue
		}
		path := source.name + ".ts"
		include = append(include, path)
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
			From:    source.language,
			To:      LanguageTS,
			Adapter: "none",
		}, filepath.Join(dir, path))
	}
	writeFile(t, filepath.Join(dir, "package.json"), `{"type":"module"}`+"\n")
	linkNodePackage(t, dir, "@stateforward", "hsm.ts", filepath.Join(root, "hsm.ts"))
	writeFile(t, filepath.Join(dir, "tsconfig.json"), strings.Join([]string{
		"{",
		`  "compilerOptions": {`,
		`    "target": "ES2022",`,
		`    "module": "NodeNext",`,
		`    "moduleResolution": "NodeNext",`,
		`    "strict": true,`,
		`    "skipLibCheck": true,`,
		`    "noEmit": true,`,
		`    "types": ["node"],`,
		`    "typeRoots": ["` + filepath.ToSlash(filepath.Join(root, "hsm.ts", "node_modules", "@types")) + `"]`,
		`  },`,
		`  "include": [` + quoteJSONArray(include) + `]`,
		"}",
		"",
	}, "\n"))
	runTool(t, dir, filepath.Join(root, "hsm.ts", "node_modules", ".bin", "tsc"), "-p", "tsconfig.json", "--pretty", "false")
}

func TestExternalForeignSourcesToPythonSmoke(t *testing.T) {
	if os.Getenv("HSMC_EXTERNAL_SMOKE") != "1" {
		t.Skip("set HSMC_EXTERNAL_SMOKE=1 to run generated target toolchain checks")
	}
	if runtime.GOOS == "windows" {
		t.Skip("external smoke paths use Unix-style shell-free local paths")
	}

	root := findMonorepoRoot(t)
	dir := t.TempDir()
	compiler := NewCompiler()
	var paths []string
	for _, source := range matrixSources(t, compiler) {
		if source.language == LanguagePython || source.language == LanguageJSONIR {
			continue
		}
		path := filepath.Join(dir, source.name+".py")
		paths = append(paths, path)
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
			From:    source.language,
			To:      LanguagePython,
			Adapter: "none",
		}, path)
	}
	args := append([]string{"-m", "py_compile"}, paths...)
	runTool(t, dir, "python3", args...)
	for _, path := range paths {
		runToolWithEnv(t, dir, []string{"PYTHONPATH=" + filepath.Join(root, "hsm.py")}, "python3", filepath.Base(path))
	}
}

func TestExternalForeignSourcesToJavaScriptSmoke(t *testing.T) {
	if os.Getenv("HSMC_EXTERNAL_SMOKE") != "1" {
		t.Skip("set HSMC_EXTERNAL_SMOKE=1 to run generated target toolchain checks")
	}
	if runtime.GOOS == "windows" {
		t.Skip("external smoke paths use Unix-style shell-free local paths")
	}

	root := findMonorepoRoot(t)
	dir := t.TempDir()
	compiler := NewCompiler()
	var paths []string
	for _, source := range matrixSources(t, compiler) {
		if source.language == LanguageJS || source.language == LanguageJSONIR {
			continue
		}
		path := filepath.Join(dir, source.name+".js")
		paths = append(paths, path)
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
			From:    source.language,
			To:      LanguageJS,
			Adapter: "none",
		}, path)
	}
	writeFile(t, filepath.Join(dir, "package.json"), `{"type":"module"}`+"\n")
	linkNodePackage(t, dir, "@stateforward", "hsm", filepath.Join(root, "hsm.js"))
	for _, path := range paths {
		runTool(t, dir, "node", filepath.Base(path))
	}
}

func TestExternalForeignSourcesToDartSmoke(t *testing.T) {
	if os.Getenv("HSMC_EXTERNAL_SMOKE") != "1" {
		t.Skip("set HSMC_EXTERNAL_SMOKE=1 to run generated target toolchain checks")
	}
	if runtime.GOOS == "windows" {
		t.Skip("external smoke paths use Unix-style shell-free local paths")
	}

	root := findMonorepoRoot(t)
	dir := t.TempDir()
	compiler := NewCompiler()
	for _, source := range matrixSources(t, compiler) {
		if source.language == LanguageDart || source.language == LanguageJSONIR {
			continue
		}
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
			From:    source.language,
			To:      LanguageDart,
			Adapter: "none",
		}, filepath.Join(dir, "lib", source.name+".dart"))
	}
	writeFile(t, filepath.Join(dir, "pubspec.yaml"), strings.Join([]string{
		"name: hsmc_foreign_sources_dart_smoke",
		"environment:",
		"  sdk: '>=3.0.0 <4.0.0'",
		"dependencies:",
		"  hsm:",
		"    path: " + filepath.Join(root, "hsm.dart"),
		"",
	}, "\n"))
	runTool(t, dir, "dart", "pub", "get")
	runTool(t, dir, "dart", "analyze")
}

func TestExternalForeignSourcesToCPPSmoke(t *testing.T) {
	if os.Getenv("HSMC_EXTERNAL_SMOKE") != "1" {
		t.Skip("set HSMC_EXTERNAL_SMOKE=1 to run generated target toolchain checks")
	}
	if runtime.GOOS == "windows" {
		t.Skip("external smoke paths use Unix-style shell-free local paths")
	}

	root := findMonorepoRoot(t)
	dir := t.TempDir()
	compiler := NewCompiler()
	var paths []string
	for _, source := range matrixSources(t, compiler) {
		if source.language == LanguageCPP || source.language == LanguageJSONIR {
			continue
		}
		path := filepath.Join(dir, source.name+".cpp")
		paths = append(paths, path)
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
			From:    source.language,
			To:      LanguageCPP,
			Adapter: "none",
		}, path)
	}
	args := []string{"-std=c++20", "-I" + filepath.Join(root, "hsm.cpp", "include"), "-fsyntax-only"}
	args = append(args, paths...)
	runTool(t, dir, "clang++", args...)
}

func TestExternalForeignSourcesToCSharpSmoke(t *testing.T) {
	if os.Getenv("HSMC_EXTERNAL_SMOKE") != "1" {
		t.Skip("set HSMC_EXTERNAL_SMOKE=1 to run generated target toolchain checks")
	}
	if runtime.GOOS == "windows" {
		t.Skip("external smoke paths use Unix-style shell-free local paths")
	}

	root := findMonorepoRoot(t)
	tmp := t.TempDir()
	compiler := NewCompiler()
	for _, source := range matrixSources(t, compiler) {
		if source.language == LanguageCSharp || source.language == LanguageJSONIR {
			continue
		}
		dir := filepath.Join(tmp, source.name)
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
			From:    source.language,
			To:      LanguageCSharp,
			Adapter: "none",
		}, filepath.Join(dir, "GeneratedHsm.cs"))
		writeFile(t, filepath.Join(dir, "Check.csproj"), `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net10.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
    <LangVersion>latest</LangVersion>
  </PropertyGroup>
  <ItemGroup>
    <ProjectReference Include="`+filepath.Join(root, "hsm.cs", "Stateforward.Hsm.csproj")+`" />
  </ItemGroup>
</Project>
`)
		runTool(t, dir, "dotnet", "build", "Check.csproj", "--nologo", "-v:minimal")
	}
}

func TestExternalForeignSourcesToJavaSmoke(t *testing.T) {
	if os.Getenv("HSMC_EXTERNAL_SMOKE") != "1" {
		t.Skip("set HSMC_EXTERNAL_SMOKE=1 to run generated target toolchain checks")
	}
	if runtime.GOOS == "windows" {
		t.Skip("external smoke paths use Unix-style shell-free local paths")
	}

	tmp := t.TempDir()
	compiler := NewCompiler()
	for _, source := range matrixSources(t, compiler) {
		if source.language == LanguageJava || source.language == LanguageJSONIR {
			continue
		}
		dir := filepath.Join(tmp, source.name)
		target := filepath.Join(dir, "GeneratedHsm.java")
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
			From:    source.language,
			To:      LanguageJava,
			Adapter: "none",
		}, target)
		runJavaSmoke(t, dir, target)
	}
}

func TestExternalForeignSourcesToZigSmoke(t *testing.T) {
	if os.Getenv("HSMC_EXTERNAL_SMOKE") != "1" {
		t.Skip("set HSMC_EXTERNAL_SMOKE=1 to run generated target toolchain checks")
	}
	if runtime.GOOS == "windows" {
		t.Skip("external smoke paths use Unix-style shell-free local paths")
	}

	root := findMonorepoRoot(t)
	tmp := t.TempDir()
	compiler := NewCompiler()
	for _, source := range matrixSources(t, compiler) {
		if source.language == LanguageZig || source.language == LanguageJSONIR {
			continue
		}
		target := filepath.Join(tmp, source.name, "door.zig")
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
			From:    source.language,
			To:      LanguageZig,
			Adapter: "none",
		}, target)
		runTool(t, tmp, "zig", "test", "--dep", "hsm", "-Mroot="+target, "-Mhsm="+filepath.Join(root, "hsm.zig", "src", "hsm.zig"))
	}
}

func TestExternalForeignSourcesToRustSmoke(t *testing.T) {
	if os.Getenv("HSMC_EXTERNAL_SMOKE") != "1" {
		t.Skip("set HSMC_EXTERNAL_SMOKE=1 to run generated target toolchain checks")
	}
	if runtime.GOOS == "windows" {
		t.Skip("external smoke paths use Unix-style shell-free local paths")
	}

	root := findMonorepoRoot(t)
	dir := t.TempDir()
	compiler := NewCompiler()
	var modules []string
	for _, source := range matrixSources(t, compiler) {
		if source.language == LanguageRust || source.language == LanguageJSONIR {
			continue
		}
		modules = append(modules, source.name)
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
			From:    source.language,
			To:      LanguageRust,
			Adapter: "none",
		}, filepath.Join(dir, "src", source.name+".rs"))
	}
	var lib strings.Builder
	for _, module := range modules {
		fmt.Fprintf(&lib, "mod %s;\n", module)
	}
	writeFile(t, filepath.Join(dir, "src", "lib.rs"), lib.String())
	writeFile(t, filepath.Join(dir, "Cargo.toml"), strings.Join([]string{
		"[package]",
		`name = "hsmc_foreign_sources_rust_smoke"`,
		`version = "0.1.0"`,
		`edition = "2024"`,
		"",
		"[dependencies]",
		`hsm = { package = "rust", path = "` + filepath.ToSlash(filepath.Join(root, "hsm.rs")) + `" }`,
		"",
	}, "\n"))
	runTool(t, dir, "cargo", "check")
}

func TestExternalTypeScriptSourceToTypeScriptSmoke(t *testing.T) {
	if os.Getenv("HSMC_EXTERNAL_SMOKE") != "1" {
		t.Skip("set HSMC_EXTERNAL_SMOKE=1 to run generated target toolchain checks")
	}
	if runtime.GOOS == "windows" {
		t.Skip("external smoke paths use Unix-style shell-free local paths")
	}

	root := findMonorepoRoot(t)
	dir := t.TempDir()
	compiler := NewCompiler()
	writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "door.ts", Data: []byte(tsDoorSource)}, CompileOptions{
		From:    LanguageTS,
		To:      LanguageTS,
		Adapter: "none",
	}, filepath.Join(dir, "door.ts"))
	writeFile(t, filepath.Join(dir, "format.ts"), "export function record(value: string): string { return value; }\n")
	writeFile(t, filepath.Join(dir, "package.json"), `{"type":"module"}`+"\n")
	linkNodePackage(t, dir, "@stateforward", "hsm.ts", filepath.Join(root, "hsm.ts"))
	writeFile(t, filepath.Join(dir, "tsconfig.json"), strings.Join([]string{
		"{",
		`  "compilerOptions": {`,
		`    "target": "ES2022",`,
		`    "module": "ESNext",`,
		`    "moduleResolution": "Bundler",`,
		`    "strict": true,`,
		`    "skipLibCheck": true,`,
		`    "noEmit": true,`,
		`    "types": ["node"],`,
		`    "typeRoots": ["` + filepath.ToSlash(filepath.Join(root, "hsm.ts", "node_modules", "@types")) + `"]`,
		`  },`,
		`  "include": ["door.ts", "format.ts"]`,
		"}",
		"",
	}, "\n"))
	runTool(t, dir, filepath.Join(root, "hsm.ts", "node_modules", ".bin", "tsc"), "-p", "tsconfig.json", "--pretty", "false")
}

func TestExternalPythonSourceToPythonSmoke(t *testing.T) {
	if os.Getenv("HSMC_EXTERNAL_SMOKE") != "1" {
		t.Skip("set HSMC_EXTERNAL_SMOKE=1 to run generated target toolchain checks")
	}
	if runtime.GOOS == "windows" {
		t.Skip("external smoke paths use Unix-style shell-free local paths")
	}

	root := findMonorepoRoot(t)
	dir := t.TempDir()
	compiler := NewCompiler()
	target := filepath.Join(dir, "door.py")
	writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "door.py", Data: []byte(pythonDoorSource)}, CompileOptions{
		From:    LanguagePython,
		To:      LanguagePython,
		Adapter: "none",
	}, target)
	writeFile(t, filepath.Join(dir, "helpers.py"), "def record(value):\n    return value\n")
	runTool(t, dir, "python3", "-m", "py_compile", filepath.Base(target), "helpers.py")
	runToolWithEnv(t, dir, []string{"PYTHONPATH=" + filepath.Join(root, "hsm.py")}, "python3", filepath.Base(target))
}

func TestExternalJavaScriptSourceToJavaScriptSmoke(t *testing.T) {
	if os.Getenv("HSMC_EXTERNAL_SMOKE") != "1" {
		t.Skip("set HSMC_EXTERNAL_SMOKE=1 to run generated target toolchain checks")
	}
	if runtime.GOOS == "windows" {
		t.Skip("external smoke paths use Unix-style shell-free local paths")
	}

	root := findMonorepoRoot(t)
	dir := t.TempDir()
	compiler := NewCompiler()
	target := filepath.Join(dir, "door.js")
	writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "door.js", Data: []byte(jsDoorSource)}, CompileOptions{
		From:    LanguageJS,
		To:      LanguageJS,
		Adapter: "none",
	}, target)
	writeFile(t, filepath.Join(dir, "helpers"), "export function record(value) { return value; }\n")
	writeFile(t, filepath.Join(dir, "package.json"), `{"type":"module"}`+"\n")
	linkNodePackage(t, dir, "@stateforward", "hsm", filepath.Join(root, "hsm.js"))
	runTool(t, dir, "node", filepath.Base(target))
}

func TestExternalDartSourceToDartSmoke(t *testing.T) {
	if os.Getenv("HSMC_EXTERNAL_SMOKE") != "1" {
		t.Skip("set HSMC_EXTERNAL_SMOKE=1 to run generated target toolchain checks")
	}
	if runtime.GOOS == "windows" {
		t.Skip("external smoke paths use Unix-style shell-free local paths")
	}

	root := findMonorepoRoot(t)
	dir := t.TempDir()
	compiler := NewCompiler()
	writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "door.dart", Data: []byte(dartDoorSource)}, CompileOptions{
		From:    LanguageDart,
		To:      LanguageDart,
		Adapter: "none",
	}, filepath.Join(dir, "lib", "door.dart"))
	writeFile(t, filepath.Join(dir, "lib", "format.dart"), "String record(String value) => value;\n")
	writeFile(t, filepath.Join(dir, "pubspec.yaml"), strings.Join([]string{
		"name: sample",
		"environment:",
		"  sdk: '>=3.0.0 <4.0.0'",
		"dependencies:",
		"  hsm:",
		"    path: " + filepath.Join(root, "hsm.dart"),
		"",
	}, "\n"))
	runTool(t, dir, "dart", "pub", "get")
	runTool(t, dir, "dart", "analyze")
}

func TestExternalZigSourceToZigSmoke(t *testing.T) {
	if os.Getenv("HSMC_EXTERNAL_SMOKE") != "1" {
		t.Skip("set HSMC_EXTERNAL_SMOKE=1 to run generated target toolchain checks")
	}
	if runtime.GOOS == "windows" {
		t.Skip("external smoke paths use Unix-style shell-free local paths")
	}

	root := findMonorepoRoot(t)
	dir := t.TempDir()
	compiler := NewCompiler()
	target := filepath.Join(dir, "door.zig")
	writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "door.zig", Data: []byte(zigDoorSource)}, CompileOptions{
		From:    LanguageZig,
		To:      LanguageZig,
		Adapter: "none",
	}, target)
	writeFile(t, filepath.Join(dir, "helper.zig"), "pub fn record(value: []const u8) []const u8 { return value; }\n")
	runTool(t, dir, "zig", "test", "--dep", "hsm", "-Mroot="+target, "-Mhsm="+filepath.Join(root, "hsm.zig", "src", "hsm.zig"))
}

func TestExternalCSharpSourceToCSharpSmoke(t *testing.T) {
	if os.Getenv("HSMC_EXTERNAL_SMOKE") != "1" {
		t.Skip("set HSMC_EXTERNAL_SMOKE=1 to run generated target toolchain checks")
	}
	if runtime.GOOS == "windows" {
		t.Skip("external smoke paths use Unix-style shell-free local paths")
	}

	root := findMonorepoRoot(t)
	dir := t.TempDir()
	compiler := NewCompiler()
	writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "door.cs", Data: []byte(csharpDoorSource)}, CompileOptions{
		From:    LanguageCSharp,
		To:      LanguageCSharp,
		Adapter: "none",
	}, filepath.Join(dir, "GeneratedHsm.cs"))
	writeFile(t, filepath.Join(dir, "Helpers.cs"), "namespace Helpers { public static class Placeholder {} }\n")
	writeFile(t, filepath.Join(dir, "Check.csproj"), `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net10.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
    <LangVersion>latest</LangVersion>
  </PropertyGroup>
  <ItemGroup>
    <ProjectReference Include="`+filepath.Join(root, "hsm.cs", "Stateforward.Hsm.csproj")+`" />
  </ItemGroup>
</Project>
`)
	runTool(t, dir, "dotnet", "build", "Check.csproj", "--nologo", "-v:minimal")
}

func TestExternalJavaSourceToJavaSmoke(t *testing.T) {
	if os.Getenv("HSMC_EXTERNAL_SMOKE") != "1" {
		t.Skip("set HSMC_EXTERNAL_SMOKE=1 to run generated target toolchain checks")
	}
	if runtime.GOOS == "windows" {
		t.Skip("external smoke paths use Unix-style shell-free local paths")
	}

	dir := t.TempDir()
	compiler := NewCompiler()
	target := filepath.Join(dir, "GeneratedHsm.java")
	writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "DoorHsm.java", Data: []byte(javaDoorSource)}, CompileOptions{
		From:    LanguageJava,
		To:      LanguageJava,
		Adapter: "none",
	}, target)
	helper := filepath.Join(dir, "com", "example", "Format.java")
	writeFile(t, helper, "package com.example;\npublic final class Format { public static String record(String value) { return value; } }\n")
	stubs := writeJavaSmokeRuntimeStubs(t, dir)
	classesDir := filepath.Join(dir, "classes")
	if err := os.MkdirAll(classesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	args := []string{"-d", classesDir}
	args = append(args, stubs...)
	args = append(args, helper, target)
	runTool(t, dir, "javac", args...)
}

func TestExternalCPPSourceToCPPSmoke(t *testing.T) {
	if os.Getenv("HSMC_EXTERNAL_SMOKE") != "1" {
		t.Skip("set HSMC_EXTERNAL_SMOKE=1 to run generated target toolchain checks")
	}
	if runtime.GOOS == "windows" {
		t.Skip("external smoke paths use Unix-style shell-free local paths")
	}

	root := findMonorepoRoot(t)
	dir := t.TempDir()
	compiler := NewCompiler()
	target := filepath.Join(dir, "door.cpp")
	writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "door.cpp", Data: []byte(cppDoorSource)}, CompileOptions{
		From:    LanguageCPP,
		To:      LanguageCPP,
		Adapter: "none",
	}, target)
	writeFile(t, filepath.Join(dir, "format.hpp"), "#pragma once\nnamespace format { inline const char* record(const char* value) { return value; } }\n")
	runTool(t, dir, "clang++", "-std=c++20", "-I"+filepath.Join(root, "hsm.cpp", "include"), "-fsyntax-only", target)
}

func TestExternalRustSourceToRustSmoke(t *testing.T) {
	if os.Getenv("HSMC_EXTERNAL_SMOKE") != "1" {
		t.Skip("set HSMC_EXTERNAL_SMOKE=1 to run generated target toolchain checks")
	}
	if runtime.GOOS == "windows" {
		t.Skip("external smoke paths use Unix-style shell-free local paths")
	}

	root := findMonorepoRoot(t)
	dir := t.TempDir()
	compiler := NewCompiler()
	writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "door.rs", Data: []byte(rustGeneratedDoorSource)}, CompileOptions{
		From:    LanguageRust,
		To:      LanguageRust,
		Adapter: "none",
	}, filepath.Join(dir, "src", "lib.rs"))
	writeFile(t, filepath.Join(dir, "Cargo.toml"), strings.Join([]string{
		"[package]",
		`name = "hsmc_rust_reemit_smoke"`,
		`version = "0.1.0"`,
		`edition = "2024"`,
		"",
		"[dependencies]",
		`hsm = { package = "rust", path = "` + filepath.ToSlash(filepath.Join(root, "hsm.rs")) + `" }`,
		"",
	}, "\n"))
	runTool(t, dir, "cargo", "check")
}

func TestExternalAdapterTargetSmoke(t *testing.T) {
	if os.Getenv("HSMC_EXTERNAL_SMOKE") != "1" {
		t.Skip("set HSMC_EXTERNAL_SMOKE=1 to run generated target toolchain checks")
	}
	if runtime.GOOS == "windows" {
		t.Skip("external smoke paths use Unix-style shell-free local paths")
	}

	root := findMonorepoRoot(t)
	tmp := t.TempDir()
	source := SourceInput{Path: "door.go", Data: []byte(goDoorSource)}
	compiler := NewCompiler()
	adapter := selfContainedSmokeAdapter{}
	compiler.RegisterAdapter(adapter)

	t.Run("go", func(t *testing.T) {
		requireGoSmokeToolchain(t, root)
		dir := filepath.Join(tmp, "adapter-go")
		writeCompiledTargetWithOptions(t, compiler, source, CompileOptions{From: LanguageGo, To: LanguageGo, Adapter: adapter.Name()}, filepath.Join(dir, "door.go"))
		writeGoSmokeMod(t, root, dir)
		runTool(t, dir, "go", "mod", "tidy")
		runTool(t, dir, "go", "test", "./...")

		timerDir := filepath.Join(tmp, "adapter-go-timer")
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "timer.go", Data: []byte(timerSmokeGoSource)}, CompileOptions{From: LanguageGo, To: LanguageGo, Adapter: adapter.Name()}, filepath.Join(timerDir, "timer.go"))
		writeGoSmokeMod(t, root, timerDir)
		runTool(t, timerDir, "go", "mod", "tidy")
		runTool(t, timerDir, "go", "test", "./...")

		whenDir := filepath.Join(tmp, "adapter-go-predicate-when")
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "predicate_when.go", Data: []byte(predicateWhenSmokeGoSource)}, CompileOptions{From: LanguageGo, To: LanguageGo, Adapter: adapter.Name()}, filepath.Join(whenDir, "predicate_when.go"))
		writeGoSmokeMod(t, root, whenDir)
		runTool(t, whenDir, "go", "mod", "tidy")
		runTool(t, whenDir, "go", "test", "./...")
	})

	t.Run("foreign_sources_to_go", func(t *testing.T) {
		requireGoSmokeToolchain(t, root)
		for _, source := range matrixSources(t, compiler) {
			source := source
			if source.language == LanguageGo || source.language == LanguageJSONIR {
				continue
			}
			t.Run(source.name, func(t *testing.T) {
				dir := filepath.Join(tmp, "adapter-"+source.name+"-source-go")
				writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
					From:    source.language,
					To:      LanguageGo,
					Adapter: adapter.Name(),
				}, filepath.Join(dir, "door.go"))
				writeGoSmokeMod(t, root, dir)
				runTool(t, dir, "go", "mod", "tidy")
				runTool(t, dir, "go", "test", "./...")
			})
		}
	})

	t.Run("python", func(t *testing.T) {
		dir := filepath.Join(tmp, "adapter-python")
		target := filepath.Join(dir, "door.py")
		writeCompiledTargetWithOptions(t, compiler, source, CompileOptions{From: LanguageGo, To: LanguagePython, Adapter: adapter.Name()}, target)
		timerPath := filepath.Join(dir, "timer.py")
		whenPath := filepath.Join(dir, "predicate_when.py")
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "timer.go", Data: []byte(timerSmokeGoSource)}, CompileOptions{From: LanguageGo, To: LanguagePython, Adapter: adapter.Name()}, timerPath)
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "predicate_when.go", Data: []byte(predicateWhenSmokeGoSource)}, CompileOptions{From: LanguageGo, To: LanguagePython, Adapter: adapter.Name()}, whenPath)
		runTool(t, tmp, "python3", "-m", "py_compile", target, timerPath, whenPath)
		runToolWithEnv(t, dir, []string{"PYTHONPATH=" + filepath.Join(root, "hsm.py")}, "python3", filepath.Base(target))
		runToolWithEnv(t, dir, []string{"PYTHONPATH=" + filepath.Join(root, "hsm.py")}, "python3", filepath.Base(timerPath))
		runToolWithEnv(t, dir, []string{"PYTHONPATH=" + filepath.Join(root, "hsm.py")}, "python3", filepath.Base(whenPath))
	})

	t.Run("foreign_sources_to_python", func(t *testing.T) {
		dir := filepath.Join(tmp, "adapter-foreign-sources-python")
		var paths []string
		for _, source := range matrixSources(t, compiler) {
			source := source
			if source.language == LanguagePython || source.language == LanguageJSONIR {
				continue
			}
			path := filepath.Join(dir, source.name+".py")
			paths = append(paths, path)
			writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
				From:    source.language,
				To:      LanguagePython,
				Adapter: adapter.Name(),
			}, path)
		}
		args := append([]string{"-m", "py_compile"}, paths...)
		runTool(t, dir, "python3", args...)
		for _, path := range paths {
			runToolWithEnv(t, dir, []string{"PYTHONPATH=" + filepath.Join(root, "hsm.py")}, "python3", filepath.Base(path))
		}
	})

	t.Run("javascript", func(t *testing.T) {
		dir := filepath.Join(tmp, "adapter-javascript")
		target := filepath.Join(dir, "door.js")
		writeCompiledTargetWithOptions(t, compiler, source, CompileOptions{From: LanguageGo, To: LanguageJS, Adapter: adapter.Name()}, target)
		timerPath := filepath.Join(dir, "timer.js")
		whenPath := filepath.Join(dir, "predicate_when.js")
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "timer.go", Data: []byte(timerSmokeGoSource)}, CompileOptions{From: LanguageGo, To: LanguageJS, Adapter: adapter.Name()}, timerPath)
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "predicate_when.go", Data: []byte(predicateWhenSmokeGoSource)}, CompileOptions{From: LanguageGo, To: LanguageJS, Adapter: adapter.Name()}, whenPath)
		writeFile(t, filepath.Join(dir, "package.json"), `{"type":"module"}`+"\n")
		linkNodePackage(t, dir, "@stateforward", "hsm", filepath.Join(root, "hsm.js"))
		runTool(t, dir, "node", filepath.Base(target))
		runTool(t, dir, "node", filepath.Base(timerPath))
		runTool(t, dir, "node", filepath.Base(whenPath))
	})

	t.Run("foreign_sources_to_javascript", func(t *testing.T) {
		dir := filepath.Join(tmp, "adapter-foreign-sources-javascript")
		var paths []string
		for _, source := range matrixSources(t, compiler) {
			source := source
			if source.language == LanguageJS || source.language == LanguageJSONIR {
				continue
			}
			path := filepath.Join(dir, source.name+".js")
			paths = append(paths, path)
			writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
				From:    source.language,
				To:      LanguageJS,
				Adapter: adapter.Name(),
			}, path)
		}
		writeFile(t, filepath.Join(dir, "package.json"), `{"type":"module"}`+"\n")
		linkNodePackage(t, dir, "@stateforward", "hsm", filepath.Join(root, "hsm.js"))
		for _, path := range paths {
			runTool(t, dir, "node", filepath.Base(path))
		}
	})

	t.Run("typescript", func(t *testing.T) {
		dir := filepath.Join(tmp, "adapter-typescript")
		target := filepath.Join(dir, "door.ts")
		writeCompiledTargetWithOptions(t, compiler, source, CompileOptions{From: LanguageGo, To: LanguageTS, Adapter: adapter.Name()}, target)
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "timer.go", Data: []byte(timerSmokeGoSource)}, CompileOptions{From: LanguageGo, To: LanguageTS, Adapter: adapter.Name()}, filepath.Join(dir, "timer.ts"))
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "predicate_when.go", Data: []byte(predicateWhenSmokeGoSource)}, CompileOptions{From: LanguageGo, To: LanguageTS, Adapter: adapter.Name()}, filepath.Join(dir, "predicate_when.ts"))
		writeFile(t, filepath.Join(dir, "package.json"), `{"type":"module"}`+"\n")
		linkNodePackage(t, dir, "@stateforward", "hsm.ts", filepath.Join(root, "hsm.ts"))
		writeFile(t, filepath.Join(dir, "tsconfig.json"), strings.Join([]string{
			"{",
			`  "compilerOptions": {`,
			`    "target": "ES2022",`,
			`    "module": "NodeNext",`,
			`    "moduleResolution": "NodeNext",`,
			`    "strict": true,`,
			`    "skipLibCheck": true,`,
			`    "noEmit": true,`,
			`    "types": ["node"],`,
			`    "typeRoots": ["` + filepath.ToSlash(filepath.Join(root, "hsm.ts", "node_modules", "@types")) + `"]`,
			`  },`,
			`  "include": ["door.ts", "timer.ts", "predicate_when.ts"]`,
			"}",
			"",
		}, "\n"))
		runTool(t, dir, filepath.Join(root, "hsm.ts", "node_modules", ".bin", "tsc"), "-p", "tsconfig.json", "--pretty", "false")
	})

	t.Run("foreign_sources_to_typescript", func(t *testing.T) {
		dir := filepath.Join(tmp, "adapter-foreign-sources-typescript")
		include := make([]string, 0)
		for _, source := range matrixSources(t, compiler) {
			source := source
			if source.language == LanguageTS || source.language == LanguageJSONIR {
				continue
			}
			path := source.name + ".ts"
			include = append(include, path)
			writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
				From:    source.language,
				To:      LanguageTS,
				Adapter: adapter.Name(),
			}, filepath.Join(dir, path))
		}
		writeFile(t, filepath.Join(dir, "package.json"), `{"type":"module"}`+"\n")
		linkNodePackage(t, dir, "@stateforward", "hsm.ts", filepath.Join(root, "hsm.ts"))
		writeFile(t, filepath.Join(dir, "tsconfig.json"), strings.Join([]string{
			"{",
			`  "compilerOptions": {`,
			`    "target": "ES2022",`,
			`    "module": "NodeNext",`,
			`    "moduleResolution": "NodeNext",`,
			`    "strict": true,`,
			`    "skipLibCheck": true,`,
			`    "noEmit": true,`,
			`    "types": ["node"],`,
			`    "typeRoots": ["` + filepath.ToSlash(filepath.Join(root, "hsm.ts", "node_modules", "@types")) + `"]`,
			`  },`,
			`  "include": [` + quoteJSONArray(include) + `]`,
			"}",
			"",
		}, "\n"))
		runTool(t, dir, filepath.Join(root, "hsm.ts", "node_modules", ".bin", "tsc"), "-p", "tsconfig.json", "--pretty", "false")
	})

	t.Run("dart", func(t *testing.T) {
		dir := filepath.Join(tmp, "adapter-dart")
		writeCompiledTargetWithOptions(t, compiler, source, CompileOptions{From: LanguageGo, To: LanguageDart, Adapter: adapter.Name()}, filepath.Join(dir, "lib", "door.dart"))
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "timer.go", Data: []byte(timerSmokeGoSource)}, CompileOptions{From: LanguageGo, To: LanguageDart, Adapter: adapter.Name()}, filepath.Join(dir, "lib", "timer.dart"))
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "every.go", Data: []byte(everySmokeGoSource)}, CompileOptions{From: LanguageGo, To: LanguageDart, Adapter: adapter.Name()}, filepath.Join(dir, "lib", "every.dart"))
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "predicate_when.go", Data: []byte(predicateWhenSmokeGoSource)}, CompileOptions{From: LanguageGo, To: LanguageDart, Adapter: adapter.Name()}, filepath.Join(dir, "lib", "predicate_when.dart"))
		writeFile(t, filepath.Join(dir, "pubspec.yaml"), strings.Join([]string{
			"name: hsmc_adapter_smoke",
			"environment:",
			"  sdk: '>=3.0.0 <4.0.0'",
			"dependencies:",
			"  hsm:",
			"    path: " + filepath.Join(root, "hsm.dart"),
			"",
		}, "\n"))
		runTool(t, dir, "dart", "pub", "get")
		runTool(t, dir, "dart", "analyze")
	})

	t.Run("foreign_sources_to_dart", func(t *testing.T) {
		dir := filepath.Join(tmp, "adapter-foreign-sources-dart")
		for _, source := range matrixSources(t, compiler) {
			source := source
			if source.language == LanguageDart || source.language == LanguageJSONIR {
				continue
			}
			writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
				From:    source.language,
				To:      LanguageDart,
				Adapter: adapter.Name(),
			}, filepath.Join(dir, "lib", source.name+".dart"))
		}
		writeFile(t, filepath.Join(dir, "pubspec.yaml"), strings.Join([]string{
			"name: hsmc_adapter_foreign_sources_smoke",
			"environment:",
			"  sdk: '>=3.0.0 <4.0.0'",
			"dependencies:",
			"  hsm:",
			"    path: " + filepath.Join(root, "hsm.dart"),
			"",
		}, "\n"))
		runTool(t, dir, "dart", "pub", "get")
		runTool(t, dir, "dart", "analyze")
	})

	t.Run("cpp", func(t *testing.T) {
		dir := filepath.Join(tmp, "adapter-cpp")
		target := filepath.Join(dir, "door.cpp")
		writeCompiledTargetWithOptions(t, compiler, source, CompileOptions{From: LanguageGo, To: LanguageCPP, Adapter: adapter.Name()}, target)
		timerPath := filepath.Join(dir, "timer.cpp")
		whenPath := filepath.Join(dir, "predicate_when.cpp")
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "timer.go", Data: []byte(timerSmokeGoSource)}, CompileOptions{From: LanguageGo, To: LanguageCPP, Adapter: adapter.Name()}, timerPath)
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "predicate_when.go", Data: []byte(predicateWhenSmokeGoSource)}, CompileOptions{From: LanguageGo, To: LanguageCPP, Adapter: adapter.Name()}, whenPath)
		runTool(t, tmp, "clang++", "-std=c++20", "-I"+filepath.Join(root, "hsm.cpp", "include"), "-fsyntax-only", target, timerPath, whenPath)
	})

	t.Run("foreign_sources_to_cpp", func(t *testing.T) {
		dir := filepath.Join(tmp, "adapter-foreign-sources-cpp")
		var paths []string
		for _, source := range matrixSources(t, compiler) {
			source := source
			if source.language == LanguageCPP || source.language == LanguageJSONIR {
				continue
			}
			path := filepath.Join(dir, source.name+".cpp")
			paths = append(paths, path)
			writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
				From:    source.language,
				To:      LanguageCPP,
				Adapter: adapter.Name(),
			}, path)
		}
		args := []string{"-std=c++20", "-I" + filepath.Join(root, "hsm.cpp", "include"), "-fsyntax-only"}
		args = append(args, paths...)
		runTool(t, tmp, "clang++", args...)
	})

	t.Run("csharp", func(t *testing.T) {
		dir := filepath.Join(tmp, "adapter-csharp")
		writeCompiledTargetWithOptions(t, compiler, source, CompileOptions{From: LanguageGo, To: LanguageCSharp, Adapter: adapter.Name()}, filepath.Join(dir, "GeneratedHsm.cs"))
		writeFile(t, filepath.Join(dir, "Check.csproj"), `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net10.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
    <LangVersion>latest</LangVersion>
  </PropertyGroup>
  <ItemGroup>
    <ProjectReference Include="`+filepath.Join(root, "hsm.cs", "Stateforward.Hsm.csproj")+`" />
  </ItemGroup>
</Project>
`)
		runTool(t, dir, "dotnet", "build", "Check.csproj", "--nologo", "-v:minimal")

		timerDir := filepath.Join(tmp, "adapter-csharp-timer")
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "timer.go", Data: []byte(timerSmokeGoSource)}, CompileOptions{From: LanguageGo, To: LanguageCSharp, Adapter: adapter.Name()}, filepath.Join(timerDir, "GeneratedHsm.cs"))
		writeFile(t, filepath.Join(timerDir, "Check.csproj"), `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net10.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
    <LangVersion>latest</LangVersion>
  </PropertyGroup>
  <ItemGroup>
    <ProjectReference Include="`+filepath.Join(root, "hsm.cs", "Stateforward.Hsm.csproj")+`" />
  </ItemGroup>
</Project>
`)
		runTool(t, timerDir, "dotnet", "build", "Check.csproj", "--nologo", "-v:minimal")

		whenDir := filepath.Join(tmp, "adapter-csharp-predicate-when")
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "predicate_when.go", Data: []byte(predicateWhenSmokeGoSource)}, CompileOptions{From: LanguageGo, To: LanguageCSharp, Adapter: adapter.Name()}, filepath.Join(whenDir, "GeneratedHsm.cs"))
		writeFile(t, filepath.Join(whenDir, "Check.csproj"), `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net10.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
    <LangVersion>latest</LangVersion>
  </PropertyGroup>
  <ItemGroup>
    <ProjectReference Include="`+filepath.Join(root, "hsm.cs", "Stateforward.Hsm.csproj")+`" />
  </ItemGroup>
</Project>
`)
		runTool(t, whenDir, "dotnet", "build", "Check.csproj", "--nologo", "-v:minimal")
	})

	t.Run("foreign_sources_to_csharp", func(t *testing.T) {
		for _, source := range matrixSources(t, compiler) {
			source := source
			if source.language == LanguageCSharp || source.language == LanguageJSONIR {
				continue
			}
			t.Run(source.name, func(t *testing.T) {
				dir := filepath.Join(tmp, "adapter-"+source.name+"-source-csharp")
				writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
					From:    source.language,
					To:      LanguageCSharp,
					Adapter: adapter.Name(),
				}, filepath.Join(dir, "GeneratedHsm.cs"))
				writeFile(t, filepath.Join(dir, "Check.csproj"), `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net10.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
    <LangVersion>latest</LangVersion>
  </PropertyGroup>
  <ItemGroup>
    <ProjectReference Include="`+filepath.Join(root, "hsm.cs", "Stateforward.Hsm.csproj")+`" />
  </ItemGroup>
</Project>
`)
				runTool(t, dir, "dotnet", "build", "Check.csproj", "--nologo", "-v:minimal")
			})
		}
	})

	t.Run("java", func(t *testing.T) {
		dir := filepath.Join(tmp, "adapter-java")
		target := filepath.Join(dir, "GeneratedHsm.java")
		writeCompiledTargetWithOptions(t, compiler, source, CompileOptions{From: LanguageGo, To: LanguageJava, Adapter: adapter.Name()}, target)
		runJavaSmoke(t, dir, target)

		timerDir := filepath.Join(tmp, "adapter-java-timer")
		timerPath := filepath.Join(timerDir, "GeneratedHsm.java")
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "timer.go", Data: []byte(timerSmokeGoSource)}, CompileOptions{From: LanguageGo, To: LanguageJava, Adapter: adapter.Name()}, timerPath)
		runJavaSmoke(t, timerDir, timerPath)

		whenDir := filepath.Join(tmp, "adapter-java-predicate-when")
		whenPath := filepath.Join(whenDir, "GeneratedHsm.java")
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "predicate_when.go", Data: []byte(predicateWhenSmokeGoSource)}, CompileOptions{From: LanguageGo, To: LanguageJava, Adapter: adapter.Name()}, whenPath)
		runJavaSmoke(t, whenDir, whenPath)
	})

	t.Run("foreign_sources_to_java", func(t *testing.T) {
		for _, source := range matrixSources(t, compiler) {
			source := source
			if source.language == LanguageJava || source.language == LanguageJSONIR {
				continue
			}
			t.Run(source.name, func(t *testing.T) {
				dir := filepath.Join(tmp, "adapter-"+source.name+"-source-java")
				target := filepath.Join(dir, "GeneratedHsm.java")
				writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
					From:    source.language,
					To:      LanguageJava,
					Adapter: adapter.Name(),
				}, target)
				runJavaSmoke(t, dir, target)
			})
		}
	})

	t.Run("zig", func(t *testing.T) {
		target := filepath.Join(tmp, "adapter-zig", "door.zig")
		writeCompiledTargetWithOptions(t, compiler, source, CompileOptions{From: LanguageGo, To: LanguageZig, Adapter: adapter.Name()}, target)
		timerPath := filepath.Join(tmp, "adapter-zig", "timer.zig")
		whenPath := filepath.Join(tmp, "adapter-zig", "predicate_when.zig")
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "timer.go", Data: []byte(timerSmokeGoSource)}, CompileOptions{From: LanguageGo, To: LanguageZig, Adapter: adapter.Name()}, timerPath)
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "predicate_when.go", Data: []byte(predicateWhenSmokeGoSource)}, CompileOptions{From: LanguageGo, To: LanguageZig, Adapter: adapter.Name()}, whenPath)
		runTool(t, tmp, "zig", "test", "--dep", "hsm", "-Mroot="+target, "-Mhsm="+filepath.Join(root, "hsm.zig", "src", "hsm.zig"))
		runTool(t, tmp, "zig", "test", "--dep", "hsm", "-Mroot="+timerPath, "-Mhsm="+filepath.Join(root, "hsm.zig", "src", "hsm.zig"))
		runTool(t, tmp, "zig", "test", "--dep", "hsm", "-Mroot="+whenPath, "-Mhsm="+filepath.Join(root, "hsm.zig", "src", "hsm.zig"))
	})

	t.Run("foreign_sources_to_zig", func(t *testing.T) {
		for _, source := range matrixSources(t, compiler) {
			source := source
			if source.language == LanguageZig || source.language == LanguageJSONIR {
				continue
			}
			t.Run(source.name, func(t *testing.T) {
				target := filepath.Join(tmp, "adapter-"+source.name+"-source-zig", "door.zig")
				writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
					From:    source.language,
					To:      LanguageZig,
					Adapter: adapter.Name(),
				}, target)
				runTool(t, tmp, "zig", "test", "--dep", "hsm", "-Mroot="+target, "-Mhsm="+filepath.Join(root, "hsm.zig", "src", "hsm.zig"))
			})
		}
	})

	t.Run("rust", func(t *testing.T) {
		dir := filepath.Join(tmp, "adapter-rust")
		writeCompiledTargetWithOptions(t, compiler, source, CompileOptions{From: LanguageGo, To: LanguageRust, Adapter: adapter.Name()}, filepath.Join(dir, "src", "lib.rs"))
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "timer.go", Data: []byte(timerSmokeGoSource)}, CompileOptions{From: LanguageGo, To: LanguageRust, Adapter: adapter.Name()}, filepath.Join(dir, "src", "timer.rs"))
		writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: "predicate_when.go", Data: []byte(predicateWhenSmokeGoSource)}, CompileOptions{From: LanguageGo, To: LanguageRust, Adapter: adapter.Name()}, filepath.Join(dir, "src", "predicate_when.rs"))
		writeFile(t, filepath.Join(dir, "src", "lib.rs"), "mod timer;\nmod predicate_when;\n"+mustReadFile(t, filepath.Join(dir, "src", "lib.rs")))
		writeFile(t, filepath.Join(dir, "Cargo.toml"), strings.Join([]string{
			"[package]",
			`name = "hsmc_adapter_smoke"`,
			`version = "0.1.0"`,
			`edition = "2024"`,
			"",
			"[dependencies]",
			`hsm = { package = "rust", path = "` + filepath.ToSlash(filepath.Join(root, "hsm.rs")) + `" }`,
			"",
		}, "\n"))
		runTool(t, dir, "cargo", "check")
	})

	t.Run("foreign_sources_to_rust", func(t *testing.T) {
		dir := filepath.Join(tmp, "adapter-foreign-sources-rust")
		var modules []string
		for _, source := range matrixSources(t, compiler) {
			source := source
			if source.language == LanguageRust || source.language == LanguageJSONIR {
				continue
			}
			modules = append(modules, source.name)
			writeCompiledTargetWithOptions(t, compiler, SourceInput{Path: source.path, Data: source.data}, CompileOptions{
				From:    source.language,
				To:      LanguageRust,
				Adapter: adapter.Name(),
			}, filepath.Join(dir, "src", source.name+".rs"))
		}
		var lib strings.Builder
		for _, module := range modules {
			fmt.Fprintf(&lib, "mod %s;\n", module)
		}
		writeFile(t, filepath.Join(dir, "src", "lib.rs"), lib.String())
		writeFile(t, filepath.Join(dir, "Cargo.toml"), strings.Join([]string{
			"[package]",
			`name = "hsmc_adapter_foreign_sources_smoke"`,
			`version = "0.1.0"`,
			`edition = "2024"`,
			"",
			"[dependencies]",
			`hsm = { package = "rust", path = "` + filepath.ToSlash(filepath.Join(root, "hsm.rs")) + `" }`,
			"",
		}, "\n"))
		runTool(t, dir, "cargo", "check")
	})
}

type selfContainedSmokeAdapter struct{}

func (adapter selfContainedSmokeAdapter) Name() string { return "self-contained-smoke" }

func (adapter selfContainedSmokeAdapter) Translate(_ context.Context, request AdapterRequest) (*BehaviorPatch, error) {
	patch := &BehaviorPatch{}
	for _, behavior := range request.Behaviors {
		patch.Behaviors = append(patch.Behaviors, Behavior{
			ID:   behavior.ID,
			Body: selfContainedSmokeBody(request.TargetLanguage, behavior),
		})
	}
	return patch, nil
}

func selfContainedSmokeBody(target Language, behavior Behavior) string {
	guard := behavior.Kind == BehaviorGuard
	trigger := behavior.Kind == BehaviorTrigger
	switch target {
	case LanguageDart:
		if trigger {
			switch behavior.TriggerKind {
			case TriggerAt:
				return "return DateTime.fromMillisecondsSinceEpoch(0);"
			case TriggerWhen:
				return "return false;"
			default:
				return "return Duration.zero;"
			}
		}
		if guard {
			return "return true;"
		}
		return "final _ = event.name;"
	case LanguageGo:
		if trigger {
			switch behavior.TriggerKind {
			case TriggerAt:
				return "return time.Time{}"
			case TriggerWhen:
				return "return make(chan struct{})"
			default:
				return "return 0"
			}
		}
		if guard {
			return "return true"
		}
		return "_ = event"
	case LanguageJS, LanguageTS:
		if trigger {
			if behavior.TriggerKind == TriggerAt {
				return "return new Date(0);"
			}
			if behavior.TriggerKind == TriggerWhen {
				return "return undefined;"
			}
			return "return 0;"
		}
		if guard {
			return "return true;"
		}
		return "void event;"
	case LanguagePython:
		if trigger {
			switch behavior.TriggerKind {
			case TriggerAt:
				return "return datetime.datetime.fromtimestamp(0)"
			case TriggerWhen:
				return "return None"
			default:
				return "return 0"
			}
		}
		if guard {
			return "return True"
		}
		return "pass"
	case LanguageCSharp:
		if trigger {
			switch behavior.TriggerKind {
			case TriggerAt:
				return "return System.DateTimeOffset.UnixEpoch;"
			case TriggerWhen:
				return "return System.Threading.Tasks.Task.CompletedTask;"
			default:
				return "return System.TimeSpan.Zero;"
			}
		}
		if guard {
			return "return true;"
		}
		return "System.GC.KeepAlive(@event);"
	case LanguageCPP:
		if behavior.Kind == BehaviorOperation {
			return "(void)0;"
		}
		if trigger {
			switch behavior.TriggerKind {
			case TriggerAt:
				return "return std::chrono::system_clock::time_point{};"
			case TriggerWhen:
				return "return true;"
			default:
				return "return std::chrono::milliseconds{0};"
			}
		}
		if guard {
			return "return true;"
		}
		return "(void)event;"
	case LanguageJava:
		if trigger {
			switch behavior.TriggerKind {
			case TriggerAt:
				return "return java.time.Instant.EPOCH;"
			case TriggerWhen:
				return "return true;"
			default:
				return "return java.time.Duration.ZERO;"
			}
		}
		if guard {
			return "return true;"
		}
		return "Object ignored = event;"
	case LanguageRust:
		if trigger {
			switch behavior.TriggerKind {
			case TriggerAt:
				return "return std::time::UNIX_EPOCH;"
			case TriggerWhen:
				return "return true;"
			default:
				return "return Duration::from_secs(0);"
			}
		}
		if guard {
			return "return true;"
		}
		return "let _ = event;"
	case LanguageZig:
		if trigger {
			if behavior.TriggerKind == TriggerWhen {
				return "_ = ctx;\n_ = inst;\n_ = event;\nreturn true;"
			}
			return "_ = ctx;\n_ = inst;\n_ = event;\nreturn 0;"
		}
		if guard {
			return "_ = ctx;\n_ = inst;\n_ = event;\nreturn true;"
		}
		return "_ = ctx;\n_ = inst;\n_ = event;"
	default:
		if guard {
			return "return true;"
		}
		return ""
	}
}

func writeGoSmokeMod(t *testing.T, root string, dir string) {
	t.Helper()
	writeFile(t, filepath.Join(dir, "go.mod"), strings.Join([]string{
		"module smoke",
		"",
		"go 1.25.1",
		"",
		"require (",
		" github.com/stateforward/hsm.go v0.0.0",
		" github.com/stateforward/hsm.go/kind v0.0.0",
		" github.com/stateforward/hsm.go/muid v0.0.0",
		")",
		"replace github.com/stateforward/hsm.go => " + filepath.Join(root, "hsm.go"),
		"replace github.com/stateforward/hsm.go/kind => " + filepath.Join(root, "hsm.go", "kind"),
		"replace github.com/stateforward/hsm.go/muid => " + filepath.Join(root, "hsm.go", "muid"),
		"",
	}, "\n"))
}

func requireGoSmokeToolchain(t *testing.T, root string) {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("go not available: %v", err)
	}
	required, ok := goModVersion(t, filepath.Join(root, "hsm.go", "go.mod"))
	if !ok {
		return
	}
	current := strings.TrimPrefix(runtime.Version(), "go")
	if current == "" || strings.Contains(current, "devel") {
		return
	}
	if compareGoVersions(current, required) < 0 {
		t.Skipf("go runtime %s is older than hsm.go requirement %s", current, required)
	}
}

func goModVersion(t *testing.T, path string) (string, bool) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if version, ok := strings.CutPrefix(line, "go "); ok {
			return strings.TrimSpace(version), true
		}
	}
	return "", false
}

func compareGoVersions(left string, right string) int {
	leftParts := goVersionParts(left)
	rightParts := goVersionParts(right)
	for index := 0; index < len(leftParts) && index < len(rightParts); index++ {
		if leftParts[index] < rightParts[index] {
			return -1
		}
		if leftParts[index] > rightParts[index] {
			return 1
		}
	}
	return 0
}

func goVersionParts(version string) [3]int {
	var parts [3]int
	for index, part := range strings.Split(version, ".") {
		if index >= len(parts) {
			break
		}
		part = strings.TrimFunc(part, func(r rune) bool {
			return r < '0' || r > '9'
		})
		if part == "" {
			continue
		}
		value, err := strconv.Atoi(part)
		if err == nil {
			parts[index] = value
		}
	}
	return parts
}

func runJavaSmoke(t *testing.T, dir string, generatedPath string) {
	t.Helper()
	stubs := writeJavaSmokeRuntimeStubs(t, dir)
	classesDir := filepath.Join(dir, "classes")
	if err := os.MkdirAll(classesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	args := []string{"-d", classesDir}
	args = append(args, stubs...)
	args = append(args, generatedPath)
	runTool(t, dir, "javac", args...)
}

func writeJavaSmokeRuntimeStubs(t *testing.T, dir string) []string {
	t.Helper()
	runtimeDir := filepath.Join(dir, "com", "stateforward", "hsm")
	files := []struct {
		name string
		code string
	}{
		{name: "Model.java", code: "package com.stateforward.hsm;\npublic final class Model {}\n"},
		{name: "Context.java", code: "package com.stateforward.hsm;\npublic final class Context {}\n"},
		{name: "Event.java", code: "package com.stateforward.hsm;\npublic final class Event { public final String name; public Event(String name) { this.name = name; } }\n"},
		{name: "Instance.java", code: "package com.stateforward.hsm;\npublic final class Instance { public void set(String name, Object value) {} }\n"},
		{name: "Behavior.java", code: "package com.stateforward.hsm;\n@FunctionalInterface public interface Behavior { void apply(Context ctx, Instance instance, Event event); }\n"},
		{name: "GuardBehavior.java", code: "package com.stateforward.hsm;\n@FunctionalInterface public interface GuardBehavior { boolean apply(Context ctx, Instance instance, Event event); }\n"},
		{name: "WhenBehavior.java", code: "package com.stateforward.hsm;\n@FunctionalInterface public interface WhenBehavior { boolean apply(Context ctx, Instance instance, Event event); }\n"},
		{name: "WhenPredicate.java", code: "package com.stateforward.hsm;\n@FunctionalInterface public interface WhenPredicate { boolean test(); }\n"},
		{name: "DurationBehavior.java", code: "package com.stateforward.hsm;\n@FunctionalInterface public interface DurationBehavior { java.time.Duration apply(Context ctx, Instance instance, Event event); }\n"},
		{name: "InstantBehavior.java", code: "package com.stateforward.hsm;\n@FunctionalInterface public interface InstantBehavior { java.time.Instant apply(Context ctx, Instance instance, Event event); }\n"},
		{name: "Hsm.java", code: strings.Join([]string{
			"package com.stateforward.hsm;",
			"public final class Hsm {",
			"  private Hsm() {}",
			"  public static Model Define(String name, Object... parts) { return new Model(); }",
			"  public static Object Initial(Object... parts) { return new Object(); }",
			"  public static Object Target(String target) { return new Object(); }",
			"  public static Object Source(String source) { return new Object(); }",
			"  public static Object State(String name, Object... parts) { return new Object(); }",
			"  public static Object Final(String name, Object... parts) { return new Object(); }",
			"  public static Object Choice(String name, Object... parts) { return new Object(); }",
			"  public static Object ShallowHistory(String name, Object... parts) { return new Object(); }",
			"  public static Object DeepHistory(String name, Object... parts) { return new Object(); }",
			"  public static Object Transition(Object... parts) { return new Object(); }",
			"  public static Object Entry(Behavior behavior) { return new Object(); }",
			"  public static Object Exit(Behavior behavior) { return new Object(); }",
			"  public static Object Activity(Behavior behavior) { return new Object(); }",
			"  public static Object Effect(Behavior behavior) { return new Object(); }",
			"  public static Object Guard(GuardBehavior behavior) { return new Object(); }",
			"  public static Object Operation(String name, Behavior behavior) { return new Object(); }",
			"  public static Object Call(Context ctx, Instance instance, String operation) { return null; }",
			"  public static Object Attribute(String name) { return new Object(); }",
			"  public static Object Attribute(String name, Object value) { return new Object(); }",
			"  public static Object Defer(String event) { return new Object(); }",
			"  public static Object On(Event event) { return new Object(); }",
			"  public static Object On(String event) { return new Object(); }",
			"  public static Object OnCall(String operation) { return new Object(); }",
			"  public static Object OnSet(String attribute) { return new Object(); }",
			"  public static Object When(String attribute) { return new Object(); }",
			"  public static Object When(WhenBehavior behavior) { return new Object(); }",
			"  public static Object When(WhenPredicate predicate) { return new Object(); }",
			"  public static Object After(java.time.Duration duration) { return new Object(); }",
			"  public static Object After(DurationBehavior behavior) { return new Object(); }",
			"  public static Object Every(java.time.Duration duration) { return new Object(); }",
			"  public static Object Every(DurationBehavior behavior) { return new Object(); }",
			"  public static Object At(java.time.Instant instant) { return new Object(); }",
			"  public static Object At(InstantBehavior behavior) { return new Object(); }",
			"}",
			"",
		}, "\n")},
	}
	paths := make([]string, 0, len(files))
	for _, file := range files {
		path := filepath.Join(runtimeDir, file.name)
		writeFile(t, path, file.code)
		paths = append(paths, path)
	}
	return paths
}

const operationSmokeGoSource = `package sample

import hsm "github.com/stateforward/hsm.go"

type DoorHSM struct{ hsm.HSM }

func approve(sm *DoorHSM) bool { return true }

var DoorModel = hsm.Define(
	"OperationDoor",
	hsm.Attribute("count", 1),
	hsm.Operation("approve", approve),
	hsm.Initial(hsm.Target("closed")),
	hsm.State("closed", hsm.Transition(hsm.OnCall("approve"), hsm.Target("../open"))),
	hsm.State("open"),
)
`

const timerSmokeGoSource = `package sample

import (
	"context"
	"time"

	hsm "github.com/stateforward/hsm.go"
)

type TimerHSM struct{ hsm.HSM }

var TimerModel = hsm.Define(
	"TimerDoor",
	hsm.Initial(hsm.Target("idle")),
	hsm.State("idle",
		hsm.Transition(
			hsm.After(func(ctx context.Context, sm *TimerHSM, event hsm.Event) time.Duration {
				return time.Second
			}),
			hsm.Target("../done"),
		),
		hsm.Transition(
			hsm.At(func(ctx context.Context, sm *TimerHSM, event hsm.Event) time.Time {
				return time.Now()
			}),
			hsm.Target("../done"),
		),
	),
	hsm.State("done"),
)
`

const everySmokeGoSource = `package sample

import (
	"context"
	"time"

	hsm "github.com/stateforward/hsm.go"
)

type EveryHSM struct{ hsm.HSM }

var EveryModel = hsm.Define(
	"EveryDoor",
	hsm.Initial(hsm.Target("idle")),
	hsm.State("idle",
		hsm.Transition(
			hsm.Every(func(ctx context.Context, sm *EveryHSM, event hsm.Event) time.Duration {
				return time.Second
			}),
			hsm.Target("../tick"),
		),
	),
	hsm.State("tick"),
)
`

const attributeWhenSmokeGoSource = `package sample

import hsm "github.com/stateforward/hsm.go"

var AttributeModel = hsm.Define(
	"AttributeDoor",
	hsm.Attribute("count", 0),
	hsm.Initial(hsm.Target("idle")),
	hsm.State("idle",
		hsm.Transition(
			hsm.When("count"),
			hsm.Target("../changed"),
		),
	),
	hsm.State("changed"),
)
`

const predicateWhenSmokeGoSource = `package sample

import (
	"context"

	hsm "github.com/stateforward/hsm.go"
)

type PredicateHSM struct{ hsm.HSM }

var PredicateWhenModel = hsm.Define(
	"PredicateWhenDoor",
	hsm.Initial(hsm.Target("idle")),
		hsm.State("idle",
			hsm.Transition(
			hsm.When(func(ctx context.Context, sm *PredicateHSM, event hsm.Event) <-chan struct{} {
				return make(chan struct{})
			}),
			hsm.Target("../ready"),
		),
	),
	hsm.State("ready"),
)
`

func writeCompiledTarget(t *testing.T, compiler *Compiler, source SourceInput, target Language, path string) {
	t.Helper()
	writeCompiledTargetWithOptions(t, compiler, source, CompileOptions{From: LanguageGo, To: target, Adapter: "none"}, path)
}

func writeCompiledTargetWithOptions(t *testing.T, compiler *Compiler, source SourceInput, options CompileOptions, path string) {
	t.Helper()
	output, _, err := compiler.Compile(context.Background(), source, options)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, string(output))
}

func writeFile(t *testing.T, path string, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func quoteJSONArray(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, strconv.Quote(value))
	}
	return strings.Join(quoted, ", ")
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func runTool(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s not available: %v", name, err)
	}
	command := exec.Command(name, args...)
	command.Dir = dir
	command.Env = externalSmokeToolEnv(name)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, output)
	}
}

func runToolWithEnv(t *testing.T, dir string, env []string, name string, args ...string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s not available: %v", name, err)
	}
	command := exec.Command(name, args...)
	command.Dir = dir
	command.Env = append(externalSmokeToolEnv(name), env...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, output)
	}
}

func externalSmokeToolEnv(name string) []string {
	env := os.Environ()
	if name != "go" || os.Getenv("ASDF_GOLANG_VERSION") != "" {
		return env
	}
	version := strings.TrimPrefix(runtime.Version(), "go")
	if version == "" || strings.Contains(version, "devel") {
		return env
	}
	return append(env, "ASDF_GOLANG_VERSION="+version)
}

func TestExternalSmokeToolEnvPinsNestedGoToCurrentToolchain(t *testing.T) {
	previous, hadPrevious := os.LookupEnv("ASDF_GOLANG_VERSION")
	if err := os.Unsetenv("ASDF_GOLANG_VERSION"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if hadPrevious {
			_ = os.Setenv("ASDF_GOLANG_VERSION", previous)
			return
		}
		_ = os.Unsetenv("ASDF_GOLANG_VERSION")
	}()

	version := strings.TrimPrefix(runtime.Version(), "go")
	if version == "" || strings.Contains(version, "devel") {
		t.Skipf("runtime version %q cannot be used as an asdf Go version", runtime.Version())
	}

	if got := envValue(externalSmokeToolEnv("go"), "ASDF_GOLANG_VERSION"); got != version {
		t.Fatalf("nested go ASDF_GOLANG_VERSION = %q, want %q", got, version)
	}
	if got := envValue(externalSmokeToolEnv("dart"), "ASDF_GOLANG_VERSION"); got != "" {
		t.Fatalf("non-go tool ASDF_GOLANG_VERSION = %q, want empty", got)
	}

	if err := os.Setenv("ASDF_GOLANG_VERSION", "9.9.9"); err != nil {
		t.Fatal(err)
	}
	if got := envValue(externalSmokeToolEnv("go"), "ASDF_GOLANG_VERSION"); got != "9.9.9" {
		t.Fatalf("existing ASDF_GOLANG_VERSION = %q, want preserved override", got)
	}
}

func TestCompareGoVersions(t *testing.T) {
	cases := []struct {
		left  string
		right string
		want  int
	}{
		{left: "1.25.1", right: "1.25.3", want: -1},
		{left: "1.25.3", right: "1.25.3", want: 0},
		{left: "1.26.0", right: "1.25.3", want: 1},
		{left: "1.25", right: "1.25.0", want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.left+"_"+tc.right, func(t *testing.T) {
			if got := compareGoVersions(tc.left, tc.right); got != tc.want {
				t.Fatalf("compareGoVersions(%q, %q) = %d, want %d", tc.left, tc.right, got, tc.want)
			}
		})
	}
}

func envValue(env []string, key string) string {
	prefix := key + "="
	for index := len(env) - 1; index >= 0; index-- {
		if strings.HasPrefix(env[index], prefix) {
			return strings.TrimPrefix(env[index], prefix)
		}
	}
	return ""
}

func findMonorepoRoot(t *testing.T) string {
	t.Helper()
	if root := strings.TrimSpace(os.Getenv("HSMC_MONOREPO_ROOT")); root != "" {
		if exists(filepath.Join(root, "hsm.go")) && exists(filepath.Join(root, "hsm.zig")) {
			return root
		}
		t.Fatalf("HSMC_MONOREPO_ROOT %q does not look like the hsm monorepo root", root)
	}
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if exists(filepath.Join(dir, "hsmc")) && exists(filepath.Join(dir, "hsm.go")) && exists(filepath.Join(dir, "hsm.zig")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("could not locate hsm monorepo root; set HSMC_MONOREPO_ROOT to run external runtime smoke tests from standalone hsmc")
		}
		dir = parent
	}
}

func linkNodePackage(t *testing.T, dir string, scope string, name string, target string) {
	t.Helper()
	linkDir := filepath.Join(dir, "node_modules", scope)
	if err := os.MkdirAll(linkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(linkDir, name)); err != nil {
		t.Skipf("symlink not available for %s/%s package resolution: %v", scope, name, err)
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
