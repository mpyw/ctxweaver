package processor_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mpyw/ctxweaver/pkg/config"
	"github.com/mpyw/ctxweaver/pkg/processor"
	"github.com/mpyw/ctxweaver/pkg/template"
)

// setupTestModule creates a temporary Go module for testing Process.
func setupTestModule(t *testing.T, files map[string]string) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := "module testmod\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create test files
	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	return tmpDir
}

// TestProcess_Options tests processor options that cannot be tested via testdata.
// Transformation tests (before -> after) are in processor_insertion_test.go using testdata.
func TestProcess_Options(t *testing.T) {
	tmpl, _ := template.Parse(`defer trace({{.Ctx}})`)
	registry := config.NewCarrierRegistry()

	t.Run("dry-run mode does not modify files", func(t *testing.T) {
		tmpDir := setupTestModule(t, map[string]string{
			"main.go": `package main

import "context"

func Foo(ctx context.Context) {
}
`,
		})

		proc := processor.New(registry, tmpl, nil, processor.WithDryRun(true))

		oldWd, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(oldWd) }()

		result, err := proc.Process([]string{"./..."})
		if err != nil {
			t.Fatalf("Process failed: %v", err)
		}

		if result.FilesModified != 1 {
			t.Errorf("FilesModified = %d, want 1", result.FilesModified)
		}

		// Verify file was NOT modified (dry-run)
		content, _ := os.ReadFile(filepath.Join(tmpDir, "main.go"))
		if strings.Contains(string(content), "defer trace(ctx)") {
			t.Errorf("file should not be modified in dry-run mode")
		}
	})

	t.Run("verbose mode prints modified files", func(t *testing.T) {
		tmpDir := setupTestModule(t, map[string]string{
			"main.go": `package main

import "context"

func Foo(ctx context.Context) {
}
`,
		})

		proc := processor.New(registry, tmpl, nil, processor.WithVerbose(true))

		oldWd, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(oldWd) }()

		result, err := proc.Process([]string{"./..."})
		if err != nil {
			t.Fatalf("Process failed: %v", err)
		}

		if result.FilesModified != 1 {
			t.Errorf("FilesModified = %d, want 1", result.FilesModified)
		}
	})

	t.Run("remove mode with nothing to remove", func(t *testing.T) {
		tmpDir := setupTestModule(t, map[string]string{
			"main.go": `package main

import "context"

func Foo(ctx context.Context) {
	println("hello")
}
`,
		})

		proc := processor.New(registry, tmpl, nil, processor.WithRemove(true))

		oldWd, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(oldWd) }()

		result, err := proc.Process([]string{"./..."})
		if err != nil {
			t.Fatalf("Process failed: %v", err)
		}

		if result.FilesModified != 0 {
			t.Errorf("FilesModified = %d, want 0 (nothing to remove)", result.FilesModified)
		}
	})
}

// TestProcess_TestFiles tests the behavior of processing test files.
func TestProcess_TestFiles(t *testing.T) {
	tmpl, _ := template.Parse(`defer trace({{.Ctx}})`)
	registry := config.NewCarrierRegistry()

	t.Run("skip test files by default", func(t *testing.T) {
		tmpDir := setupTestModule(t, map[string]string{
			"main.go": `package main

import "context"

func Foo(ctx context.Context) {
}
`,
			"main_test.go": `package main

import "context"

func TestBar(ctx context.Context) {
}
`,
		})

		proc := processor.New(registry, tmpl, nil)

		oldWd, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(oldWd) }()

		result, err := proc.Process([]string{"./..."})
		if err != nil {
			t.Fatalf("Process failed: %v", err)
		}

		if result.FilesProcessed != 1 {
			t.Errorf("FilesProcessed = %d, want 1", result.FilesProcessed)
		}
	})

	t.Run("process test files when enabled", func(t *testing.T) {
		tmpDir := setupTestModule(t, map[string]string{
			"main.go": `package main

import "context"

func Foo(ctx context.Context) {
}
`,
			"main_test.go": `package main

import "context"

func TestBar(ctx context.Context) {
}
`,
		})

		proc := processor.New(registry, tmpl, nil, processor.WithTest(true))

		oldWd, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(oldWd) }()

		result, err := proc.Process([]string{"./..."})
		if err != nil {
			t.Fatalf("Process failed: %v", err)
		}

		if result.FilesProcessed < 2 {
			t.Errorf("FilesProcessed = %d, want >= 2", result.FilesProcessed)
		}
	})
}

// TestProcess_SkipTestdataDirectory tests that testdata directories are skipped.
func TestProcess_SkipTestdataDirectory(t *testing.T) {
	tmpl, _ := template.Parse(`defer trace({{.Ctx}})`)
	registry := config.NewCarrierRegistry()

	tmpDir := setupTestModule(t, map[string]string{
		"main.go": `package main

import "context"

func Foo(ctx context.Context) {
}
`,
		"testdata/fixture.go": `package testdata

import "context"

func Fixture(ctx context.Context) {
}
`,
	})

	proc := processor.New(registry, tmpl, nil)

	oldWd, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(oldWd) }()

	result, err := proc.Process([]string{"./..."})
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if result.FilesProcessed != 1 {
		t.Errorf("FilesProcessed = %d, want 1 (testdata should be skipped)", result.FilesProcessed)
	}
}
