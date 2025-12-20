package processor_test

import (
	"bytes"
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

// TestProcess_ExcludeRegexps tests package exclusion by regex patterns.
func TestProcess_ExcludeRegexps(t *testing.T) {
	tmpl, _ := template.Parse(`defer trace({{.Ctx}})`)
	registry := config.NewCarrierRegistry()

	t.Run("exclude packages matching regex", func(t *testing.T) {
		tmpDir := setupTestModule(t, map[string]string{
			"main.go": `package main

import "context"

func Foo(ctx context.Context) {
}
`,
			"internal/repo/repo.go": `package repo

import "context"

func Get(ctx context.Context) {
}
`,
			"handler/handler.go": `package handler

import "context"

func Handle(ctx context.Context) {
}
`,
		})

		// Exclude packages containing "internal"
		proc := processor.New(registry, tmpl, nil, processor.WithPackageRegexps(config.Regexps{Omit: []string{"/internal/"}}))

		oldWd, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(oldWd) }()

		result, err := proc.Process([]string{"./..."})
		if err != nil {
			t.Fatalf("Process failed: %v", err)
		}

		// Should process main.go and handler/handler.go, but not internal/repo/repo.go
		if result.FilesProcessed != 2 {
			t.Errorf("FilesProcessed = %d, want 2", result.FilesProcessed)
		}
		if result.FilesModified != 2 {
			t.Errorf("FilesModified = %d, want 2", result.FilesModified)
		}

		// Verify internal/repo/repo.go was NOT modified
		content, _ := os.ReadFile(filepath.Join(tmpDir, "internal/repo/repo.go"))
		if strings.Contains(string(content), "defer trace(ctx)") {
			t.Errorf("internal/repo/repo.go should not be modified (excluded)")
		}

		// Verify handler/handler.go WAS modified
		content, _ = os.ReadFile(filepath.Join(tmpDir, "handler/handler.go"))
		if !strings.Contains(string(content), "defer trace(ctx)") {
			t.Errorf("handler/handler.go should be modified")
		}
	})

	t.Run("multiple exclude patterns", func(t *testing.T) {
		tmpDir := setupTestModule(t, map[string]string{
			"main.go": `package main

import "context"

func Foo(ctx context.Context) {
}
`,
			"mock/mock.go": `package mock

import "context"

func Mock(ctx context.Context) {
}
`,
			"internal/repo/repo.go": `package repo

import "context"

func Get(ctx context.Context) {
}
`,
		})

		// Exclude packages containing "internal" or "mock"
		proc := processor.New(registry, tmpl, nil, processor.WithPackageRegexps(config.Regexps{Omit: []string{"/internal/", "/mock$"}}))

		oldWd, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(oldWd) }()

		result, err := proc.Process([]string{"./..."})
		if err != nil {
			t.Fatalf("Process failed: %v", err)
		}

		// Should only process main.go
		if result.FilesProcessed != 1 {
			t.Errorf("FilesProcessed = %d, want 1", result.FilesProcessed)
		}
	})

	t.Run("no exclusions when patterns empty", func(t *testing.T) {
		tmpDir := setupTestModule(t, map[string]string{
			"main.go": `package main

import "context"

func Foo(ctx context.Context) {
}
`,
			"internal/repo/repo.go": `package repo

import "context"

func Get(ctx context.Context) {
}
`,
		})

		// Empty exclude patterns
		proc := processor.New(registry, tmpl, nil, processor.WithPackageRegexps(config.Regexps{}))

		oldWd, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(oldWd) }()

		result, err := proc.Process([]string{"./..."})
		if err != nil {
			t.Fatalf("Process failed: %v", err)
		}

		// Should process all files
		if result.FilesProcessed != 2 {
			t.Errorf("FilesProcessed = %d, want 2", result.FilesProcessed)
		}
	})

	t.Run("invalid regex pattern warns and skips", func(t *testing.T) {
		tmpDir := setupTestModule(t, map[string]string{
			"main.go": `package main

import "context"

func Foo(ctx context.Context) {
}
`,
		})

		// Capture stderr
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		// Invalid regex pattern: unclosed bracket
		proc := processor.New(registry, tmpl, nil, processor.WithPackageRegexps(config.Regexps{Omit: []string{"[invalid"}}))

		// Restore stderr and read captured output
		_ = w.Close()
		os.Stderr = oldStderr
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		captured := buf.String()

		// Verify warning was printed
		if !strings.Contains(captured, "warning:") || !strings.Contains(captured, "[invalid") {
			t.Errorf("expected warning for invalid pattern, got: %q", captured)
		}

		oldWd, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(oldWd) }()

		// Should still process files (invalid pattern is skipped)
		result, err := proc.Process([]string{"./..."})
		if err != nil {
			t.Fatalf("Process failed: %v", err)
		}

		if result.FilesProcessed != 1 {
			t.Errorf("FilesProcessed = %d, want 1", result.FilesProcessed)
		}
	})
}
