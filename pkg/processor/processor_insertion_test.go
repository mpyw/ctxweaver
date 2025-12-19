package processor_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"

	"github.com/mpyw/ctxweaver/pkg/config"
	"github.com/mpyw/ctxweaver/pkg/processor"
	"github.com/mpyw/ctxweaver/pkg/template"
)

// testConfig holds test-specific configuration from config.yaml
type testConfig struct {
	Template string   `yaml:"template"`
	Imports  []string `yaml:"imports"`
}

// defaultConfig returns the default newrelic template config
func defaultConfig() testConfig {
	return testConfig{
		Template: `defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`,
		Imports:  []string{"github.com/newrelic/go-agent/v3/newrelic"},
	}
}

// loadTestConfig loads config.yaml from dir or returns default
func loadTestConfig(dir string) testConfig {
	configPath := filepath.Join(dir, "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return defaultConfig()
	}

	var cfg testConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return defaultConfig()
	}
	return cfg
}

// TestTransformSource_Golden tests transformation using testdata files.
// Each subdirectory in testdata/ contains before.go, after.go, and optional config.yaml.
func TestTransformSource_Golden(t *testing.T) {
	testdataRoot := filepath.Join("..", "..", "internal", "testdata")

	entries, err := os.ReadDir(testdataRoot)
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		testDir := filepath.Join(testdataRoot, entry.Name())

		// Check for nested test directories
		subEntries, err := os.ReadDir(testDir)
		if err != nil {
			t.Fatalf("failed to read %s: %v", testDir, err)
		}

		hasBeforeGo := false
		hasAfterGo := false
		for _, sub := range subEntries {
			if sub.Name() == "before.go" || sub.Name() == "before.raw.go" {
				hasBeforeGo = true
			}
			if sub.Name() == "after.go" {
				hasAfterGo = true
			}
		}

		if hasBeforeGo && hasAfterGo {
			// Single test case with after.go
			runGoldenTest(t, entry.Name(), testDir)
		} else if hasBeforeGo {
			// Test without after.go - skip golden test (idempotency only)
			continue
		} else {
			// Nested test cases
			for _, sub := range subEntries {
				if !sub.IsDir() {
					continue
				}
				subDir := filepath.Join(testDir, sub.Name())
				testName := entry.Name() + "/" + sub.Name()
				runGoldenTest(t, testName, subDir)
			}
		}
	}
}

func runGoldenTest(t *testing.T, name, dir string) {
	t.Run(name, func(t *testing.T) {
		// Check if after.go exists
		afterPath := filepath.Join(dir, "after.go")
		if _, err := os.Stat(afterPath); os.IsNotExist(err) {
			t.Skip("no after.go - skipping golden test")
			return
		}

		// Try before.raw.go first (intentionally unformatted), then before.go
		beforePath := filepath.Join(dir, "before.raw.go")
		before, err := os.ReadFile(beforePath)
		if err != nil {
			beforePath = filepath.Join(dir, "before.go")
			before, err = os.ReadFile(beforePath)
			if err != nil {
				t.Fatalf("failed to read before.go: %v", err)
			}
		}

		want, err := os.ReadFile(afterPath)
		if err != nil {
			t.Fatalf("failed to read after.go: %v", err)
		}

		cfg := loadTestConfig(dir)

		registry, err := config.NewCarrierRegistry()
		if err != nil {
			t.Fatalf("failed to create carrier registry: %v", err)
		}

		tmpl, err := template.Parse(cfg.Template)
		if err != nil {
			t.Fatalf("failed to parse template: %v", err)
		}

		proc := processor.New(registry, tmpl, cfg.Imports)

		got, err := proc.TransformSource(before, "test")
		if err != nil {
			t.Fatalf("TransformSource failed: %v", err)
		}

		if diff := cmp.Diff(string(want), string(got)); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
}

// TestTransformSource_Idempotency tests that transformation is idempotent.
func TestTransformSource_Idempotency(t *testing.T) {
	testdataRoot := filepath.Join("..", "..", "internal", "testdata")

	entries, err := os.ReadDir(testdataRoot)
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		testDir := filepath.Join(testdataRoot, entry.Name())

		subEntries, err := os.ReadDir(testDir)
		if err != nil {
			t.Fatalf("failed to read %s: %v", testDir, err)
		}

		hasBeforeGo := false
		for _, sub := range subEntries {
			if sub.Name() == "before.go" || sub.Name() == "before.raw.go" {
				hasBeforeGo = true
				break
			}
		}

		if hasBeforeGo {
			runIdempotencyTest(t, entry.Name(), testDir)
		} else {
			for _, sub := range subEntries {
				if !sub.IsDir() {
					continue
				}
				subDir := filepath.Join(testDir, sub.Name())
				testName := entry.Name() + "/" + sub.Name()
				runIdempotencyTest(t, testName, subDir)
			}
		}
	}
}

func runIdempotencyTest(t *testing.T, name, dir string) {
	t.Run(name, func(t *testing.T) {
		// Try before.raw.go first, then before.go
		beforePath := filepath.Join(dir, "before.raw.go")
		before, err := os.ReadFile(beforePath)
		if err != nil {
			beforePath = filepath.Join(dir, "before.go")
			before, err = os.ReadFile(beforePath)
			if err != nil {
				t.Fatalf("failed to read before.go: %v", err)
			}
		}

		cfg := loadTestConfig(dir)

		registry, err := config.NewCarrierRegistry()
		if err != nil {
			t.Fatalf("failed to create carrier registry: %v", err)
		}

		tmpl, err := template.Parse(cfg.Template)
		if err != nil {
			t.Fatalf("failed to parse template: %v", err)
		}

		proc := processor.New(registry, tmpl, cfg.Imports)

		// First transformation
		first, err := proc.TransformSource(before, "test")
		if err != nil {
			t.Fatalf("first TransformSource failed: %v", err)
		}

		// Second transformation
		second, err := proc.TransformSource(first, "test")
		if err != nil {
			t.Fatalf("second TransformSource failed: %v", err)
		}

		// Third transformation
		third, err := proc.TransformSource(second, "test")
		if err != nil {
			t.Fatalf("third TransformSource failed: %v", err)
		}

		if diff := cmp.Diff(string(first), string(second)); diff != "" {
			t.Errorf("first vs second mismatch:\n%s", diff)
		}

		if diff := cmp.Diff(string(second), string(third)); diff != "" {
			t.Errorf("second vs third mismatch:\n%s", diff)
		}
	})
}

// TestGeneratedFiles tests that generated files are skipped.
func TestGeneratedFiles(t *testing.T) {
	testdataRoot := filepath.Join("..", "..", "internal", "testdata", "generated_files")

	entries, err := os.ReadDir(testdataRoot)
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	registry, _ := config.NewCarrierRegistry()
	tmpl, _ := template.Parse(`defer trace({{.Ctx}})`)
	proc := processor.New(registry, tmpl, []string{"trace"})

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			beforePath := filepath.Join(testdataRoot, entry.Name(), "before.go")
			before, err := os.ReadFile(beforePath)
			if err != nil {
				t.Fatalf("failed to read before.go: %v", err)
			}

			result, err := proc.TransformSource(before, "test")
			if err != nil {
				t.Fatalf("transform failed: %v", err)
			}

			// Generated files should not be modified
			if string(result) != string(before) {
				t.Errorf("generated file should not be modified")
			}
		})
	}
}

// TestIdempotency_TemplateChange tests skeleton mode behavior.
func TestIdempotency_TemplateChange(t *testing.T) {
	registry, err := config.NewCarrierRegistry()
	if err != nil {
		t.Fatalf("failed to create carrier registry: %v", err)
	}

	before := readTestdataFile(t, "basic_newrelic", "before.go")

	tmpl1, _ := template.Parse(`defer trace1({{.Ctx}}, {{.FuncName | quote}})`)
	proc1 := processor.New(registry, tmpl1, []string{"trace1"})

	first, err := proc1.TransformSource(before, "test")
	if err != nil {
		t.Fatalf("first transform failed: %v", err)
	}

	if !strings.Contains(string(first), "trace1") {
		t.Error("first transform should contain trace1")
	}

	tmpl2, _ := template.Parse(`defer trace2({{.Ctx}}, {{.FuncName | quote}})`)
	proc2 := processor.New(registry, tmpl2, []string{"trace2"})

	second, err := proc2.TransformSource(first, "test")
	if err != nil {
		t.Fatalf("second transform failed: %v", err)
	}

	// Both should exist (insert, not update)
	if !strings.Contains(string(second), "trace2") {
		t.Error("second transform should contain trace2")
	}
	if !strings.Contains(string(second), "trace1") {
		t.Error("skeleton mode: trace1 should still exist")
	}
}

// readTestdataFile reads a file from testdata directory
func readTestdataFile(t *testing.T, subdir, filename string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "internal", "testdata", subdir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return data
}
