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

	t.Run("only patterns filter packages", func(t *testing.T) {
		tmpDir := setupTestModule(t, map[string]string{
			"main.go": `package main

import "context"

func Foo(ctx context.Context) {
}
`,
			"handler/handler.go": `package handler

import "context"

func Handle(ctx context.Context) {
}
`,
			"service/service.go": `package service

import "context"

func Serve(ctx context.Context) {
}
`,
		})

		// Only process packages containing "handler"
		proc := processor.New(registry, tmpl, nil, processor.WithPackageRegexps(config.Regexps{Only: []string{"/handler$"}}))

		oldWd, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(oldWd) }()

		result, err := proc.Process([]string{"./..."})
		if err != nil {
			t.Fatalf("Process failed: %v", err)
		}

		// Should only process handler/handler.go
		if result.FilesProcessed != 1 {
			t.Errorf("FilesProcessed = %d, want 1", result.FilesProcessed)
		}

		// Verify handler/handler.go WAS modified
		content, _ := os.ReadFile(filepath.Join(tmpDir, "handler/handler.go"))
		if !strings.Contains(string(content), "defer trace(ctx)") {
			t.Errorf("handler/handler.go should be modified")
		}

		// Verify main.go was NOT modified
		content, _ = os.ReadFile(filepath.Join(tmpDir, "main.go"))
		if strings.Contains(string(content), "defer trace(ctx)") {
			t.Errorf("main.go should not be modified (not in only list)")
		}
	})

	t.Run("only and omit patterns combined", func(t *testing.T) {
		tmpDir := setupTestModule(t, map[string]string{
			"handler/public.go": `package handler

import "context"

func Public(ctx context.Context) {
}
`,
			"handler/internal.go": `package handler

import "context"

func Internal(ctx context.Context) {
}
`,
			"service/service.go": `package service

import "context"

func Serve(ctx context.Context) {
}
`,
		})

		// Only process handler packages, but omit internal files
		// Note: omit is checked after only, so internal.go will still be processed
		// because the package path matches only pattern
		proc := processor.New(registry, tmpl, nil, processor.WithPackageRegexps(config.Regexps{
			Only: []string{"/handler$"},
			Omit: []string{"/service$"},
		}))

		oldWd, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		defer func() { _ = os.Chdir(oldWd) }()

		result, err := proc.Process([]string{"./..."})
		if err != nil {
			t.Fatalf("Process failed: %v", err)
		}

		// Should process 2 files in handler package
		if result.FilesProcessed != 2 {
			t.Errorf("FilesProcessed = %d, want 2", result.FilesProcessed)
		}

		// Verify service was NOT processed
		content, _ := os.ReadFile(filepath.Join(tmpDir, "service/service.go"))
		if strings.Contains(string(content), "defer trace(ctx)") {
			t.Errorf("service/service.go should not be modified")
		}
	})

	t.Run("invalid only pattern warns and skips", func(t *testing.T) {
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

		// Invalid regex pattern in only
		proc := processor.New(registry, tmpl, nil, processor.WithPackageRegexps(config.Regexps{Only: []string{"[invalid"}}))

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

		// With invalid only pattern skipped, empty only list means process all
		result, err := proc.Process([]string{"./..."})
		if err != nil {
			t.Fatalf("Process failed: %v", err)
		}

		if result.FilesProcessed != 1 {
			t.Errorf("FilesProcessed = %d, want 1", result.FilesProcessed)
		}
	})
}

// TestProcess_FunctionFiltering tests function filtering by type, scope, and regex.
func TestProcess_FunctionFiltering(t *testing.T) {
	tmpl, _ := template.Parse(`defer trace({{.Ctx}})`)
	registry := config.NewCarrierRegistry()

	t.Run("filter by function type - functions only", func(t *testing.T) {
		tmpDir := setupTestModule(t, map[string]string{
			"main.go": `package main

import "context"

func Foo(ctx context.Context) {
}

type Service struct{}

func (s *Service) Bar(ctx context.Context) {
}
`,
		})

		proc := processor.New(registry, tmpl, nil, processor.WithFunctions(config.Functions{
			Types:  []config.FuncType{config.FuncTypeFunction},
			Scopes: []config.FuncScope{config.FuncScopeExported, config.FuncScopeUnexported},
		}))

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

		content, _ := os.ReadFile(filepath.Join(tmpDir, "main.go"))
		contentStr := string(content)
		// Foo should be modified
		if !strings.Contains(contentStr, "func Foo(ctx context.Context) {\n\tdefer trace(ctx)") {
			t.Errorf("Foo should be modified")
		}
		// Bar (method) should NOT be modified
		if strings.Contains(contentStr, "func (s *Service) Bar(ctx context.Context) {\n\tdefer trace(ctx)") {
			t.Errorf("Bar (method) should not be modified")
		}
	})

	t.Run("filter by function type - methods only", func(t *testing.T) {
		tmpDir := setupTestModule(t, map[string]string{
			"main.go": `package main

import "context"

func Foo(ctx context.Context) {
}

type Service struct{}

func (s *Service) Bar(ctx context.Context) {
}
`,
		})

		proc := processor.New(registry, tmpl, nil, processor.WithFunctions(config.Functions{
			Types:  []config.FuncType{config.FuncTypeMethod},
			Scopes: []config.FuncScope{config.FuncScopeExported, config.FuncScopeUnexported},
		}))

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

		content, _ := os.ReadFile(filepath.Join(tmpDir, "main.go"))
		contentStr := string(content)
		// Foo should NOT be modified
		if strings.Contains(contentStr, "func Foo(ctx context.Context) {\n\tdefer trace(ctx)") {
			t.Errorf("Foo should not be modified (not a method)")
		}
		// Bar (method) should be modified
		if !strings.Contains(contentStr, "func (s *Service) Bar(ctx context.Context) {\n\tdefer trace(ctx)") {
			t.Errorf("Bar (method) should be modified")
		}
	})

	t.Run("filter by scope - exported only", func(t *testing.T) {
		tmpDir := setupTestModule(t, map[string]string{
			"main.go": `package main

import "context"

func PublicFunc(ctx context.Context) {
}

func privateFunc(ctx context.Context) {
}
`,
		})

		proc := processor.New(registry, tmpl, nil, processor.WithFunctions(config.Functions{
			Types:  []config.FuncType{config.FuncTypeFunction, config.FuncTypeMethod},
			Scopes: []config.FuncScope{config.FuncScopeExported},
		}))

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

		content, _ := os.ReadFile(filepath.Join(tmpDir, "main.go"))
		contentStr := string(content)
		// PublicFunc should be modified
		if !strings.Contains(contentStr, "func PublicFunc(ctx context.Context) {\n\tdefer trace(ctx)") {
			t.Errorf("PublicFunc should be modified")
		}
		// privateFunc should NOT be modified
		if strings.Contains(contentStr, "func privateFunc(ctx context.Context) {\n\tdefer trace(ctx)") {
			t.Errorf("privateFunc should not be modified (unexported)")
		}
	})

	t.Run("filter by scope - unexported only", func(t *testing.T) {
		tmpDir := setupTestModule(t, map[string]string{
			"main.go": `package main

import "context"

func PublicFunc(ctx context.Context) {
}

func privateFunc(ctx context.Context) {
}
`,
		})

		proc := processor.New(registry, tmpl, nil, processor.WithFunctions(config.Functions{
			Types:  []config.FuncType{config.FuncTypeFunction, config.FuncTypeMethod},
			Scopes: []config.FuncScope{config.FuncScopeUnexported},
		}))

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

		content, _ := os.ReadFile(filepath.Join(tmpDir, "main.go"))
		contentStr := string(content)
		// PublicFunc should NOT be modified
		if strings.Contains(contentStr, "func PublicFunc(ctx context.Context) {\n\tdefer trace(ctx)") {
			t.Errorf("PublicFunc should not be modified (exported)")
		}
		// privateFunc should be modified
		if !strings.Contains(contentStr, "func privateFunc(ctx context.Context) {\n\tdefer trace(ctx)") {
			t.Errorf("privateFunc should be modified")
		}
	})

	t.Run("filter by function name regex - only", func(t *testing.T) {
		tmpDir := setupTestModule(t, map[string]string{
			"main.go": `package main

import "context"

func HandleRequest(ctx context.Context) {
}

func ProcessData(ctx context.Context) {
}
`,
		})

		proc := processor.New(registry, tmpl, nil, processor.WithFunctions(config.Functions{
			Types:  []config.FuncType{config.FuncTypeFunction, config.FuncTypeMethod},
			Scopes: []config.FuncScope{config.FuncScopeExported, config.FuncScopeUnexported},
			Regexps: config.Regexps{
				Only: []string{"^Handle"},
			},
		}))

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

		content, _ := os.ReadFile(filepath.Join(tmpDir, "main.go"))
		contentStr := string(content)
		// HandleRequest should be modified
		if !strings.Contains(contentStr, "func HandleRequest(ctx context.Context) {\n\tdefer trace(ctx)") {
			t.Errorf("HandleRequest should be modified")
		}
		// ProcessData should NOT be modified
		if strings.Contains(contentStr, "func ProcessData(ctx context.Context) {\n\tdefer trace(ctx)") {
			t.Errorf("ProcessData should not be modified (doesn't match only pattern)")
		}
	})

	t.Run("filter by function name regex - omit", func(t *testing.T) {
		tmpDir := setupTestModule(t, map[string]string{
			"main.go": `package main

import "context"

func HandleRequest(ctx context.Context) {
}

func ProcessDataHelper(ctx context.Context) {
}
`,
		})

		proc := processor.New(registry, tmpl, nil, processor.WithFunctions(config.Functions{
			Types:  []config.FuncType{config.FuncTypeFunction, config.FuncTypeMethod},
			Scopes: []config.FuncScope{config.FuncScopeExported, config.FuncScopeUnexported},
			Regexps: config.Regexps{
				Omit: []string{"Helper$"},
			},
		}))

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

		content, _ := os.ReadFile(filepath.Join(tmpDir, "main.go"))
		contentStr := string(content)
		// HandleRequest should be modified
		if !strings.Contains(contentStr, "func HandleRequest(ctx context.Context) {\n\tdefer trace(ctx)") {
			t.Errorf("HandleRequest should be modified")
		}
		// ProcessDataHelper should NOT be modified
		if strings.Contains(contentStr, "func ProcessDataHelper(ctx context.Context) {\n\tdefer trace(ctx)") {
			t.Errorf("ProcessDataHelper should not be modified (matches omit pattern)")
		}
	})

	t.Run("filter by function name regex - only and omit combined", func(t *testing.T) {
		tmpDir := setupTestModule(t, map[string]string{
			"main.go": `package main

import "context"

func HandleRequest(ctx context.Context) {
}

func HandleRequestHelper(ctx context.Context) {
}

func ProcessData(ctx context.Context) {
}
`,
		})

		proc := processor.New(registry, tmpl, nil, processor.WithFunctions(config.Functions{
			Types:  []config.FuncType{config.FuncTypeFunction, config.FuncTypeMethod},
			Scopes: []config.FuncScope{config.FuncScopeExported, config.FuncScopeUnexported},
			Regexps: config.Regexps{
				Only: []string{"^Handle"},
				Omit: []string{"Helper$"},
			},
		}))

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

		content, _ := os.ReadFile(filepath.Join(tmpDir, "main.go"))
		contentStr := string(content)
		// HandleRequest should be modified (matches only, not omit)
		if !strings.Contains(contentStr, "func HandleRequest(ctx context.Context) {\n\tdefer trace(ctx)") {
			t.Errorf("HandleRequest should be modified")
		}
		// HandleRequestHelper should NOT be modified (matches only but also matches omit)
		if strings.Contains(contentStr, "func HandleRequestHelper(ctx context.Context) {\n\tdefer trace(ctx)") {
			t.Errorf("HandleRequestHelper should not be modified (matches omit pattern)")
		}
		// ProcessData should NOT be modified (doesn't match only)
		if strings.Contains(contentStr, "func ProcessData(ctx context.Context) {\n\tdefer trace(ctx)") {
			t.Errorf("ProcessData should not be modified (doesn't match only pattern)")
		}
	})

	t.Run("invalid function regex warns and skips", func(t *testing.T) {
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

		// Invalid regex pattern in function filter
		proc := processor.New(registry, tmpl, nil, processor.WithFunctions(config.Functions{
			Types:  []config.FuncType{config.FuncTypeFunction, config.FuncTypeMethod},
			Scopes: []config.FuncScope{config.FuncScopeExported, config.FuncScopeUnexported},
			Regexps: config.Regexps{
				Only: []string{"[invalid"},
			},
		}))

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

		// With invalid only pattern skipped, empty only list means process all
		result, err := proc.Process([]string{"./..."})
		if err != nil {
			t.Fatalf("Process failed: %v", err)
		}

		if result.FilesProcessed != 1 {
			t.Errorf("FilesProcessed = %d, want 1", result.FilesProcessed)
		}
	})
}
