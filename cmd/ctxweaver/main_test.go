package main

import (
	"bytes"
	"flag"
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
patterns: []
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

	t.Run("pre hook failure", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "hook_fail.yaml")
		config := `template: "defer trace({{.Ctx}})"
imports: []
patterns: []
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
patterns: []
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
patterns:
  - "./..."
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

	t.Run("default patterns when none specified", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ctxweaver.yaml")
		// Config without patterns - should default to "./..."
		config := `template: "defer trace({{.Ctx}})"
imports: []
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

		// No patterns in args, should use default "./..."
		setup("-config", configPath, "-silent")
		err := run()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("non-silent output", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ctxweaver.yaml")
		config := `template: "defer trace({{.Ctx}})"
imports: []
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
patterns: []
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
