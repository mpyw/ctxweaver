package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsFlagPassed(t *testing.T) {
	tests := map[string]struct {
		args     []string
		flagName string
		want     bool
	}{
		"flag passed": {
			args:     []string{"-test=true"},
			flagName: "test",
			want:     true,
		},
		"flag not passed": {
			args:     []string{},
			flagName: "test",
			want:     false,
		},
		"different flag passed": {
			args:     []string{"-verbose"},
			flagName: "test",
			want:     false,
		},
		"multiple flags, target passed": {
			args:     []string{"-verbose", "-test"},
			flagName: "test",
			want:     true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset flags for each test
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			flag.CommandLine.SetOutput(&bytes.Buffer{}) // Suppress output

			// Define flags
			var test, verbose bool
			flag.BoolVar(&test, "test", false, "")
			flag.BoolVar(&verbose, "verbose", false, "")

			// Parse args
			_ = flag.CommandLine.Parse(tt.args)

			got := isFlagPassed(tt.flagName)
			if got != tt.want {
				t.Errorf("isFlagPassed(%q) = %v, want %v", tt.flagName, got, tt.want)
			}
		})
	}
}

func TestRunHooks(t *testing.T) {
	tests := map[string]struct {
		commands []string
		silent   bool
		wantErr  bool
	}{
		"single successful command": {
			commands: []string{"echo hello"},
			silent:   true,
			wantErr:  false,
		},
		"multiple successful commands": {
			commands: []string{"echo one", "echo two", "echo three"},
			silent:   true,
			wantErr:  false,
		},
		"failing command": {
			commands: []string{"exit 1"},
			silent:   true,
			wantErr:  true,
		},
		"fail on second command": {
			commands: []string{"echo ok", "exit 1", "echo never"},
			silent:   true,
			wantErr:  true,
		},
		"empty commands": {
			commands: []string{},
			silent:   true,
			wantErr:  false,
		},
		"with output (not silent)": {
			commands: []string{"echo visible"},
			silent:   false,
			wantErr:  false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := runHooks("test", tt.commands, tt.silent)
			if (err != nil) != tt.wantErr {
				t.Errorf("runHooks() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunHooks_ErrorMessage(t *testing.T) {
	err := runHooks("pre", []string{"exit 42"}, true)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "pre hook failed") {
		t.Errorf("error should mention 'pre hook failed', got: %v", err)
	}
	if !strings.Contains(err.Error(), "exit 42") {
		t.Errorf("error should mention the command, got: %v", err)
	}
}

// TestCLI_Integration runs integration tests for the CLI binary.
// These tests actually build and run the binary.
func TestCLI_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Build the binary
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "ctxweaver")
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build: %v\n%s", err, out)
	}

	t.Run("missing config file", func(t *testing.T) {
		cmd := exec.Command(binPath, "-config", "nonexistent.yaml")
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("expected error for missing config")
		}
		if !strings.Contains(string(out), "failed to load config") {
			t.Errorf("unexpected output: %s", out)
		}
	})

	t.Run("help flag", func(t *testing.T) {
		cmd := exec.Command(binPath, "-help")
		out, _ := cmd.CombinedOutput()
		// -help returns exit code 2 but that's ok
		output := string(out)
		if !strings.Contains(output, "-config") {
			t.Errorf("help should mention -config: %s", output)
		}
		if !strings.Contains(output, "-test") {
			t.Errorf("help should mention -test: %s", output)
		}
		if !strings.Contains(output, "-remove") {
			t.Errorf("help should mention -remove: %s", output)
		}
		if !strings.Contains(output, "-silent") {
			t.Errorf("help should mention -silent: %s", output)
		}
	})

	t.Run("with valid config", func(t *testing.T) {
		// Create a minimal config
		configPath := filepath.Join(tmpDir, "ctxweaver.yaml")
		config := `template: "defer trace({{.Ctx}})"
imports: []
packages:
  patterns:
    - ./...
`
		if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		// Create an empty Go file to process
		goFile := filepath.Join(tmpDir, "test.go")
		goCode := `package test

import "context"

func trace(context.Context) {}

func Foo(ctx context.Context) {
}
`
		if err := os.WriteFile(goFile, []byte(goCode), 0o644); err != nil {
			t.Fatalf("failed to write go file: %v", err)
		}

		// Create go.mod
		goMod := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goMod, []byte("module test\n\ngo 1.21\n"), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		cmd := exec.Command(binPath, "-config", configPath, "-silent", "./...")
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("unexpected error: %v\n%s", err, out)
		}
	})

	t.Run("dry-run mode", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "ctxweaver.yaml")
		cmd := exec.Command(binPath, "-config", configPath, "-dry-run", "./...")
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("unexpected error: %v\n%s", err, out)
		}
		if !strings.Contains(string(out), "Files processed") {
			t.Errorf("dry-run should show files processed: %s", out)
		}
	})

	t.Run("malformed directive warns without failing", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "ctxweaver.yaml")
		config := `template: "defer trace({{.Ctx}})"
imports: []
packages:
  patterns:
    - ./...
`
		files := map[string]string{
			"ctxweaver.yaml": config,
			"go.mod":         "module test\n\ngo 1.21\n",
			"test.go": `package test

import "context"

func trace(context.Context) {}

// ctxweaver:skip
func Foo(ctx context.Context) {
}
`,
		}
		for name, content := range files {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
				t.Fatalf("failed to write %s: %v", name, err)
			}
		}

		cmd := exec.Command(binPath, "-config", configPath, "-silent", "./...")
		cmd.Dir = dir
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("a malformed directive should not fail the run: %v\n%s", err, stderr.String())
		}
		if !strings.Contains(stderr.String(), "warning: ") ||
			!strings.Contains(stderr.String(), "test.go:7: malformed ctxweaver directive: write it as //ctxweaver:name") {
			t.Errorf("stderr should carry the warning with its position: %q", stderr.String())
		}
		content, _ := os.ReadFile(filepath.Join(dir, "test.go"))
		if !strings.Contains(string(content), "defer trace(ctx)") {
			t.Errorf("a malformed directive should not skip the function:\n%s", content)
		}
	})

	t.Run("pre hook failure", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "hook_fail.yaml")
		config := `template: "defer trace({{.Ctx}})"
imports: []
packages:
  patterns:
    - ./...
hooks:
  pre:
    - "exit 1"
`
		if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		cmd := exec.Command(binPath, "-config", configPath, "-silent", "./...")
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("expected error for pre hook failure")
		}
		if !strings.Contains(string(out), "pre hook failed") {
			t.Errorf("should mention pre hook failed: %s", out)
		}
	})
}

func TestRun(t *testing.T) {
	// Helper to reset flags and set args
	setup := func(args ...string) {
		flag.CommandLine = flag.NewFlagSet("ctxweaver", flag.ContinueOnError)
		flag.CommandLine.SetOutput(&bytes.Buffer{})
		os.Args = append([]string{"ctxweaver"}, args...)
	}

	t.Run("config load failure", func(t *testing.T) {
		setup("-config", "nonexistent.yaml", "-silent")
		err := run()
		if err == nil {
			t.Fatal("expected error for missing config")
		}
		if !strings.Contains(err.Error(), "failed to load config") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("missing template in config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ctxweaver.yaml")
		config := `imports: []
packages:
  patterns:
    - ./...
`
		if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		setup("-config", configPath, "-silent")
		err := run()
		if err == nil {
			t.Fatal("expected error for missing template")
		}
		// Schema validation will catch this
		if !strings.Contains(err.Error(), "failed to load config") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid template", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ctxweaver.yaml")
		config := `template: "defer trace({{.Invalid"
imports: []
packages:
  patterns:
    - ./...
`
		if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		setup("-config", configPath, "-silent")
		err := run()
		if err == nil {
			t.Fatal("expected error for invalid template")
		}
		if !strings.Contains(err.Error(), "failed to parse template") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("successful run with patterns from config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ctxweaver.yaml")
		config := `template: "defer trace({{.Ctx}})"
imports: []
packages:
  patterns:
    - ./...
`
		if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		goMod := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goMod, []byte("module test\n\ngo 1.21\n"), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		goFile := filepath.Join(tmpDir, "test.go")
		goCode := `package test

import "context"

func trace(context.Context) {}

func Foo(ctx context.Context) {
}
`
		if err := os.WriteFile(goFile, []byte(goCode), 0o644); err != nil {
			t.Fatalf("failed to write go file: %v", err)
		}

		oldWd, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(oldWd) }()

		setup("-config", configPath, "-silent")
		err := run()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("successful run with dry-run and verbose", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ctxweaver.yaml")
		config := `template: "defer trace({{.Ctx}})"
imports: []
packages:
  patterns:
    - ./...
`
		if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		goMod := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goMod, []byte("module test\n\ngo 1.21\n"), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		goFile := filepath.Join(tmpDir, "test.go")
		goCode := `package test

import "context"

func Foo(ctx context.Context) {
}
`
		if err := os.WriteFile(goFile, []byte(goCode), 0o644); err != nil {
			t.Fatalf("failed to write go file: %v", err)
		}

		oldWd, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(oldWd) }()

		setup("-config", configPath, "-dry-run", "-verbose", "./...")
		err := run()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("successful run with remove mode", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ctxweaver.yaml")
		config := `template: "defer trace({{.Ctx}})"
imports: []
packages:
  patterns:
    - ./...
`
		if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		goMod := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goMod, []byte("module test\n\ngo 1.21\n"), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		goFile := filepath.Join(tmpDir, "test.go")
		goCode := `package test

import "context"

func trace(context.Context) {}

func Foo(ctx context.Context) {
	defer trace(ctx)
}
`
		if err := os.WriteFile(goFile, []byte(goCode), 0o644); err != nil {
			t.Fatalf("failed to write go file: %v", err)
		}

		oldWd, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(oldWd) }()

		setup("-config", configPath, "-remove", "-silent", "./...")
		err := run()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("with post hooks", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ctxweaver.yaml")
		config := `template: "defer trace({{.Ctx}})"
imports: []
packages:
  patterns:
    - ./...
hooks:
  post:
    - "echo done"
`
		if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		goMod := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goMod, []byte("module test\n\ngo 1.21\n"), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		oldWd, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(oldWd) }()

		setup("-config", configPath, "-silent", "./...")
		err := run()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("with pre hooks (no-hooks flag)", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ctxweaver.yaml")
		config := `template: "defer trace({{.Ctx}})"
imports: []
packages:
  patterns:
    - ./...
hooks:
  pre:
    - "exit 1"
`
		if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		goMod := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goMod, []byte("module test\n\ngo 1.21\n"), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		oldWd, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(oldWd) }()

		setup("-config", configPath, "-no-hooks", "-silent", "./...")
		err := run()
		if err != nil {
			t.Errorf("unexpected error (no-hooks should skip failing hook): %v", err)
		}
	})

	t.Run("pre hook failure", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ctxweaver.yaml")
		config := `template: "defer trace({{.Ctx}})"
imports: []
packages:
  patterns:
    - ./...
hooks:
  pre:
    - "exit 1"
`
		if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		goMod := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goMod, []byte("module test\n\ngo 1.21\n"), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		oldWd, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(oldWd) }()

		setup("-config", configPath, "-silent", "./...")
		err := run()
		if err == nil {
			t.Fatal("expected error for pre hook failure")
		}
		if !strings.Contains(err.Error(), "pre hook failed") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("post hook failure", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ctxweaver.yaml")
		config := `template: "defer trace({{.Ctx}})"
imports: []
packages:
  patterns:
    - ./...
hooks:
  post:
    - "exit 1"
`
		if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		goMod := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goMod, []byte("module test\n\ngo 1.21\n"), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		oldWd, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(oldWd) }()

		setup("-config", configPath, "-silent", "./...")
		err := run()
		if err == nil {
			t.Fatal("expected error for post hook failure")
		}
		if !strings.Contains(err.Error(), "post hook failed") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("test flag override", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ctxweaver.yaml")
		config := `template: "defer trace({{.Ctx}})"
imports: []
packages:
  patterns:
    - ./...
test: false
`
		if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		goMod := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goMod, []byte("module test\n\ngo 1.21\n"), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		oldWd, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(oldWd) }()

		setup("-config", configPath, "-test=true", "-silent", "./...")
		err := run()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("with custom carriers", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ctxweaver.yaml")
		config := `template: "defer trace({{.Ctx}})"
imports: []
packages:
  patterns:
    - ./...
carriers:
  - package: "net/http"
    type: "Request"
    accessor: ".Context()"
`
		if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		goMod := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goMod, []byte("module test\n\ngo 1.21\n"), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		goFile := filepath.Join(tmpDir, "test.go")
		goCode := `package test

import "net/http"

func trace(interface{}) {}

func Handler(r *http.Request) {
}
`
		if err := os.WriteFile(goFile, []byte(goCode), 0o644); err != nil {
			t.Fatalf("failed to write go file: %v", err)
		}

		oldWd, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(oldWd) }()

		setup("-config", configPath, "-silent", "./...")
		err := run()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("no patterns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ctxweaver.yaml")
		// Config without packages.patterns - should error
		config := `template: "defer trace({{.Ctx}})"
imports: []
packages:
  patterns: []
`
		if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		goMod := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goMod, []byte("module test\n\ngo 1.21\n"), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		oldWd, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(oldWd) }()

		setup("-config", configPath, "-silent")
		err := run()
		if err == nil {
			t.Error("expected error for empty patterns")
		}
	})

	t.Run("non-silent output", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ctxweaver.yaml")
		config := `template: "defer trace({{.Ctx}})"
imports: []
packages:
  patterns:
    - ./...
`
		if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		goMod := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goMod, []byte("module test\n\ngo 1.21\n"), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		goFile := filepath.Join(tmpDir, "test.go")
		goCode := `package test

import "context"

func trace(context.Context) {}

func Foo(ctx context.Context) {
}
`
		if err := os.WriteFile(goFile, []byte(goCode), 0o644); err != nil {
			t.Fatalf("failed to write go file: %v", err)
		}

		oldWd, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(oldWd) }()

		// Without -silent, output will be printed
		setup("-config", configPath, "./...")
		err := run()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestCLI_ConfigOverride(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "ctxweaver")
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build: %v\n%s", err, out)
	}

	// Create config with test: true
	configPath := filepath.Join(tmpDir, "ctxweaver.yaml")
	config := `template: "defer trace({{.Ctx}})"
imports: []
packages:
  patterns:
    - ./...
test: true
`
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Create go.mod
	goMod := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module test\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	t.Run("--test=false flag is accepted", func(t *testing.T) {
		// Just verify the flag is parsed without error
		cmd := exec.Command(binPath, "-config", configPath, "-silent", "-test=false", "./...")
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("--test=false should be accepted: %v\n%s", err, out)
		}
	})

	t.Run("--test=true flag is accepted", func(t *testing.T) {
		cmd := exec.Command(binPath, "-config", configPath, "-silent", "-test=true", "./...")
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("--test=true should be accepted: %v\n%s", err, out)
		}
	})
}

func TestRun_PackageWithSyntaxError(t *testing.T) {
	// Helper to reset flags and set args
	setup := func(args ...string) {
		flag.CommandLine = flag.NewFlagSet("ctxweaver", flag.ContinueOnError)
		flag.CommandLine.SetOutput(&bytes.Buffer{})
		os.Args = append([]string{"ctxweaver"}, args...)
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ctxweaver.yaml")
	config := `template: "defer trace({{.Ctx}})"
imports: []
packages:
  patterns:
    - ./...
`
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	goMod := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module test\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create Go file with syntax error
	goFile := filepath.Join(tmpDir, "broken.go")
	goCode := `package test

import "context"

func Foo(ctx context.Context) {
	// missing closing brace
`
	if err := os.WriteFile(goFile, []byte(goCode), 0o644); err != nil {
		t.Fatalf("failed to write go file: %v", err)
	}

	oldWd, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(oldWd) }()

	setup("-config", configPath, "-silent", "./...")
	err := run()
	if err == nil {
		t.Fatal("expected error for package with syntax error")
	}
	if !strings.Contains(err.Error(), "error") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRun_PackageWithTypeError(t *testing.T) {
	// Helper to reset flags and set args
	setup := func(args ...string) {
		flag.CommandLine = flag.NewFlagSet("ctxweaver", flag.ContinueOnError)
		flag.CommandLine.SetOutput(&bytes.Buffer{})
		os.Args = append([]string{"ctxweaver"}, args...)
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ctxweaver.yaml")
	config := `template: "defer trace({{.Ctx}})"
imports: []
packages:
  patterns:
    - ./...
`
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	goMod := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module test\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create Go file with type error (undefined variable)
	goFile := filepath.Join(tmpDir, "typeerror.go")
	goCode := `package test

import "context"

func Foo(ctx context.Context) {
	_ = undefinedVariable
}
`
	if err := os.WriteFile(goFile, []byte(goCode), 0o644); err != nil {
		t.Fatalf("failed to write go file: %v", err)
	}

	oldWd, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(oldWd) }()

	setup("-config", configPath, "-silent", "./...")
	err := run()
	if err == nil {
		t.Fatal("expected error for package with type error")
	}
	if !strings.Contains(err.Error(), "error") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestCLI_LineDirectives runs the built binary on modules with //line
// directives. A directive must not change which file is read, filtered or
// written, and a directive in a rewritten file must stay a directive.
func TestCLI_LineDirectives(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := filepath.Join(t.TempDir(), "ctxweaver")
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build: %v\n%s", err, out)
	}

	const config = "template: |\n  _ = {{.Ctx}}\npackages:\n  patterns:\n    - ./...\n"
	const goMod = "module example.com/repro\n\ngo 1.22\n"
	// The modules have no dependencies, so go needs no network.
	env := append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off", "GOFLAGS=")

	// setup writes the files into a new module and returns its real path.
	// macOS temp dirs are symlinks, and the tool prints resolved paths.
	setup := func(t *testing.T, files map[string]string) string {
		t.Helper()
		dir, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		files["go.mod"] = goMod
		files["ctxweaver.yaml"] = config
		for name, content := range files {
			path := filepath.Join(dir, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return dir
	}

	// weave runs the binary and returns its stdout.
	weave := func(t *testing.T, dir string) string {
		t.Helper()
		cmd := exec.Command(binPath, "-verbose", "-no-hooks")
		cmd.Dir = dir
		cmd.Env = env
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("ctxweaver failed: %v\n%s%s", err, stdout.String(), stderr.String())
		}
		if stderr.Len() > 0 {
			t.Errorf("unexpected stderr: %s", stderr.String())
		}
		return stdout.String()
	}

	// files returns every file in dir with its content, keyed by slash path.
	files := func(t *testing.T, dir string) map[string]string {
		t.Helper()
		got := map[string]string{}
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(dir, path)
			got[filepath.ToSlash(rel)] = string(content)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		return got
	}

	checkFiles := func(t *testing.T, dir string, want map[string]string) {
		t.Helper()
		want["go.mod"] = goMod
		want["ctxweaver.yaml"] = config
		got := files(t, dir)
		for name, content := range got {
			if w, ok := want[name]; !ok {
				t.Errorf("unexpected file %s:\n%s", name, content)
			} else if content != w {
				t.Errorf("%s =\n%s\nwant\n%s", name, content, w)
			}
		}
		for name := range want {
			if _, ok := got[name]; !ok {
				t.Errorf("missing file %s", name)
			}
		}
	}

	// summary is the stdout of a run that processed the given number of
	// files and modified only p/a.go.
	summary := func(dir string, processed int) string {
		return "▶ ctxweaver weaving ./...\n" +
			"modified: " + filepath.Join(dir, "p", "a.go") + "\n" +
			fmt.Sprintf("  Files processed: %d\n", processed) +
			"  Files modified: 1\n"
	}

	// goRun runs the module's main package and returns its output.
	goRun := func(t *testing.T, dir string) string {
		t.Helper()
		cmd := exec.Command("go", "run", ".")
		cmd.Dir = dir
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go run failed: %v\n%s", err, out)
		}
		return string(out)
	}

	t.Run("directive before package clause", func(t *testing.T) {
		const body = `package p

import "context"

func F(ctx context.Context) {
	println("x")
}
`
		const woven = `package p

import "context"

func F(ctx context.Context) {
	_ = ctx

	println("x")
}
`
		tests := map[string]string{
			"control":                         "",
			"file name in directive":          "//line fake.tmpl:1\n",
			"test file name in directive":     "//line gen_test.go:1\n",
			"testdata file name in directive": "//line /src/testdata/gen.go:1\n",
		}
		for name, directive := range tests {
			t.Run(name, func(t *testing.T) {
				dir := setup(t, map[string]string{"p/a.go": directive + body})

				if got, want := weave(t, dir), summary(dir, 1); got != want {
					t.Errorf("stdout =\n%s\nwant\n%s", got, want)
				}
				checkFiles(t, dir, map[string]string{"p/a.go": directive + woven})
			})
		}
	})

	t.Run("directive in function body", func(t *testing.T) {
		const mainGo = `package main

import "example.com/repro/p"

func main() { println(p.H()) }
`
		source := func(directive string) string {
			return `package p

import (
	"context"
	"runtime"
)

func F(ctx context.Context) {` + weaveMark + `
	println("x")
}

func H() int {
` + directive + `	_, _, line, _ := runtime.Caller(0)
	return line
}
`
		}
		tests := map[string]struct {
			directive string
			before    string // go run output of the original program
			after     string // go run output of the rewritten program
		}{
			"control":   {directive: "", before: "13\n", after: "15\n"},
			"directive": {directive: "//line fake.tmpl:100\n", before: "100\n", after: "100\n"},
		}
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				original := strings.Replace(source(tt.directive), weaveMark, "", 1)
				dir := setup(t, map[string]string{"main.go": mainGo, "p/a.go": original})
				if got := goRun(t, dir); got != tt.before {
					t.Fatalf("original program printed %q, want %q", got, tt.before)
				}

				// main.go is processed too, and has nothing to weave.
				if got, want := weave(t, dir), summary(dir, 2); got != want {
					t.Errorf("stdout =\n%s\nwant\n%s", got, want)
				}
				woven := strings.Replace(source(tt.directive), weaveMark, "\n\t_ = ctx\n", 1)
				checkFiles(t, dir, map[string]string{"main.go": mainGo, "p/a.go": woven})
				if got := goRun(t, dir); got != tt.after {
					t.Errorf("rewritten program printed %q, want %q", got, tt.after)
				}
			})
		}
	})
}

// weaveMark marks where a test source gets the woven statement.
const weaveMark = "<weave>"
