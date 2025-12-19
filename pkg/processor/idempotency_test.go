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

// TestIdempotency_FromTestdata tests idempotency using testdata files.
// Each subdirectory in testdata/idempotency/ contains a before.go and optional config.yaml.
func TestIdempotency_FromTestdata(t *testing.T) {
	testdataRoot := filepath.Join("..", "..", "internal", "testdata", "idempotency")

	// Walk testdata/idempotency directory
	entries, err := os.ReadDir(testdataRoot)
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		testDir := filepath.Join(testdataRoot, entry.Name())

		// Check for nested test directories (e.g., trailing_newlines/no_trailing)
		subEntries, err := os.ReadDir(testDir)
		if err != nil {
			t.Fatalf("failed to read %s: %v", testDir, err)
		}

		hasBeforeGo := false
		for _, sub := range subEntries {
			if sub.Name() == "before.go" {
				hasBeforeGo = true
				break
			}
		}

		if hasBeforeGo {
			// Single test case
			runIdempotencyTest(t, entry.Name(), testDir)
		} else {
			// Nested test cases
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
		beforePath := filepath.Join(dir, "before.go")
		before, err := os.ReadFile(beforePath)
		if err != nil {
			t.Fatalf("failed to read before.go: %v", err)
		}

		afterPath := filepath.Join(dir, "after.go")
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

		// First transformation
		got, err := proc.TransformSource(before, "test")
		if err != nil {
			t.Fatalf("transform failed: %v", err)
		}

		// Check against expected output
		if diff := cmp.Diff(string(want), string(got)); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}

		// Check idempotency (second run should produce same result)
		second, err := proc.TransformSource(got, "test")
		if err != nil {
			t.Fatalf("second transform failed: %v", err)
		}

		if diff := cmp.Diff(string(got), string(second)); diff != "" {
			t.Errorf("NOT IDEMPOTENT:\n%s", diff)
		}
	})
}

// TestGeneratedFiles_FromTestdata tests that generated files are skipped.
func TestGeneratedFiles_FromTestdata(t *testing.T) {
	testdataRoot := filepath.Join("..", "..", "internal", "testdata", "idempotency", "generated_files")

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
				t.Errorf("generated file should not be modified:\ngot:\n%s\nwant:\n%s", string(result), string(before))
			}
		})
	}
}

// TestIdempotency_TemplateChange_SkeletonMode tests behavior when template changes.
// In skeleton mode, different function names are considered different structures,
// so a template change will INSERT a new statement rather than UPDATE.
func TestIdempotency_TemplateChange_SkeletonMode(t *testing.T) {
	registry, err := config.NewCarrierRegistry()
	if err != nil {
		t.Fatalf("failed to create carrier registry: %v", err)
	}

	before := readTestdataFile(t, "basic", "before.go")

	// First template
	tmpl1, _ := template.Parse(`defer trace1({{.Ctx}}, {{.FuncName | quote}})`)
	proc1 := processor.New(registry, tmpl1, []string{"trace1"})

	first, err := proc1.TransformSource(before, "test")
	if err != nil {
		t.Fatalf("first transform failed: %v", err)
	}

	if !strings.Contains(string(first), "trace1") {
		t.Error("first transform should contain trace1")
	}

	// Second template - IN SKELETON MODE, this inserts a new statement
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

// TestIdempotency_TemplateChange_MarkerMode tests that marker mode updates on template change.
func TestIdempotency_TemplateChange_MarkerMode(t *testing.T) {
	registry, err := config.NewCarrierRegistry()
	if err != nil {
		t.Fatalf("failed to create carrier registry: %v", err)
	}

	before := readTestdataFile(t, "basic", "before.go")

	// First template with marker mode
	tmpl1, _ := template.Parse(`defer trace1({{.Ctx}}, {{.FuncName | quote}})`)
	proc1 := processor.New(registry, tmpl1, []string{"trace1"}, processor.WithMatchMode(processor.MatchModeMarker))

	first, err := proc1.TransformSource(before, "test")
	if err != nil {
		t.Fatalf("first transform failed: %v", err)
	}

	if !strings.Contains(string(first), "trace1") {
		t.Error("first transform should contain trace1")
	}
	if !strings.Contains(string(first), "//ctxweaver:generated") {
		t.Error("marker mode should add marker")
	}

	// Second template - IN MARKER MODE, this updates the marked statement
	tmpl2, _ := template.Parse(`defer trace2({{.Ctx}}, {{.FuncName | quote}})`)
	proc2 := processor.New(registry, tmpl2, []string{"trace2"}, processor.WithMatchMode(processor.MatchModeMarker))

	second, err := proc2.TransformSource(first, "test")
	if err != nil {
		t.Fatalf("second transform failed: %v", err)
	}

	// trace2 replaces trace1
	if !strings.Contains(string(second), "trace2") {
		t.Error("second transform should contain trace2")
	}
	if strings.Contains(string(second), "trace1") {
		t.Error("marker mode: trace1 should be replaced by trace2")
	}

	// Third run should be stable
	third, err := proc2.TransformSource(second, "test")
	if err != nil {
		t.Fatalf("third transform failed: %v", err)
	}

	if diff := cmp.Diff(string(second), string(third)); diff != "" {
		t.Errorf("marker mode not stable:\n%s", diff)
	}
}

// TestIdempotency_MarkerMode tests marker mode with default template.
func TestIdempotency_MarkerMode(t *testing.T) {
	registry, _ := config.NewCarrierRegistry()
	tmpl, _ := template.Parse(`defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`)
	proc := processor.New(
		registry,
		tmpl,
		[]string{"github.com/newrelic/go-agent/v3/newrelic"},
		processor.WithMatchMode(processor.MatchModeMarker),
	)

	before := readTestdataFile(t, "basic", "before.go")

	first, err := proc.TransformSource(before, "test")
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}

	if !strings.Contains(string(first), "//ctxweaver:generated") {
		t.Error("marker mode should add marker")
	}

	second, err := proc.TransformSource(first, "test")
	if err != nil {
		t.Fatalf("second transform failed: %v", err)
	}

	if diff := cmp.Diff(string(first), string(second)); diff != "" {
		t.Errorf("marker mode not stable:\n%s", diff)
	}
}

// TestIdempotency_ManyRuns tests stability over 10 consecutive runs.
func TestIdempotency_ManyRuns(t *testing.T) {
	registry, _ := config.NewCarrierRegistry()
	tmpl, _ := template.Parse(`defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`)
	proc := processor.New(registry, tmpl, []string{"github.com/newrelic/go-agent/v3/newrelic"})

	before := readTestdataFile(t, "preserves_existing_code", "before.go")
	current := before

	for i := 0; i < 10; i++ {
		next, err := proc.TransformSource(current, "test")
		if err != nil {
			t.Fatalf("run %d failed: %v", i, err)
		}

		if i > 0 {
			if diff := cmp.Diff(string(current), string(next)); diff != "" {
				t.Errorf("run %d not stable:\n%s", i, diff)
			}
		}
		current = next
	}
}

// TestPreservesExistingCode tests that existing code is preserved.
func TestPreservesExistingCode(t *testing.T) {
	registry, _ := config.NewCarrierRegistry()
	tmpl, _ := template.Parse(`defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`)
	proc := processor.New(registry, tmpl, []string{"github.com/newrelic/go-agent/v3/newrelic"})

	before := readTestdataFile(t, "preserves_existing_code", "before.go")

	result, err := proc.TransformSource(before, "test")
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}

	checks := []string{
		"This comment should be preserved",
		"x := 1",
		"y := 2",
		"doSomething(x, y)",
		"func doSomething",
	}

	for _, check := range checks {
		if !strings.Contains(string(result), check) {
			t.Errorf("missing expected code: %q", check)
		}
	}
}

// TestSimilarStatements tests that similar existing statements are preserved.
func TestSimilarStatements(t *testing.T) {
	registry, _ := config.NewCarrierRegistry()
	tmpl, _ := template.Parse(`defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`)
	proc := processor.New(registry, tmpl, []string{"github.com/newrelic/go-agent/v3/newrelic"})

	before := readTestdataFile(t, "similar_statements", "before.go")

	first, err := proc.TransformSource(before, "test")
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}

	// Should have both: generated statement AND existing defer
	if !strings.Contains(string(first), `"test.Foo"`) {
		t.Error("should have generated statement")
	}
	if !strings.Contains(string(first), "someOtherFunc()") {
		t.Error("should preserve existing defer")
	}

	// Count defer statements
	deferCount := strings.Count(string(first), "defer ")
	if deferCount != 2 {
		t.Errorf("expected 2 defer statements, got %d", deferCount)
	}

	// Should be stable
	second, err := proc.TransformSource(first, "test")
	if err != nil {
		t.Fatalf("second transform failed: %v", err)
	}

	if diff := cmp.Diff(string(first), string(second)); diff != "" {
		t.Errorf("not stable:\n%s", diff)
	}
}

// readTestdataFile reads a file from testdata/idempotency directory
func readTestdataFile(t *testing.T, subdir, filename string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "internal", "testdata", "idempotency", subdir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return data
}
