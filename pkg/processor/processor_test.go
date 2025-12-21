package processor_test

import (
	"io/fs"
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

// testCase represents a single test case discovered from testdata.
type testCase struct {
	name      string
	hasAfter  bool
	hasBefore bool
}

// testConfig holds test-specific configuration from config.yaml.
type testConfig struct {
	Template   string   `yaml:"template"`
	Imports    []string `yaml:"imports"`
	SkipRemove bool     `yaml:"skip_remove"` // skip this case in remove tests
}

// defaultConfig returns the default newrelic template config.
func defaultConfig() testConfig {
	return testConfig{
		Template: `defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`,
		Imports:  []string{"github.com/newrelic/go-agent/v3/newrelic"},
	}
}

// loadTestConfig loads config.yaml from dir or returns default.
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
	// Fill in defaults for missing fields
	if cfg.Template == "" {
		cfg.Template = defaultConfig().Template
	}
	if len(cfg.Imports) == 0 {
		cfg.Imports = defaultConfig().Imports
	}
	return cfg
}

// discoverTestCases discovers all test cases from testdata directory.
func discoverTestCases(t *testing.T, testdataRoot string) []testCase {
	t.Helper()

	entries, err := os.ReadDir(testdataRoot)
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	var cases []testCase

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "_stubs" {
			continue
		}

		testDir := filepath.Join(testdataRoot, entry.Name())
		subEntries, err := os.ReadDir(testDir)
		if err != nil {
			t.Fatalf("failed to read %s: %v", testDir, err)
		}

		var hasBefore, hasAfter bool
		for _, sub := range subEntries {
			switch sub.Name() {
			case "before.go":
				hasBefore = true
			case "after.go":
				hasAfter = true
			}
		}

		if hasBefore {
			// Single test case
			cases = append(cases, testCase{
				name:      entry.Name(),
				hasBefore: true,
				hasAfter:  hasAfter,
			})
		} else {
			// Nested test cases
			for _, sub := range subEntries {
				if !sub.IsDir() {
					continue
				}
				cases = append(cases, testCase{
					name:      entry.Name() + "/" + sub.Name(),
					hasBefore: true,
					hasAfter:  true, // nested cases should have both
				})
			}
		}
	}
	return cases
}

// copyDir recursively copies a directory tree, with optional exclusions.
func copyDir(src, dst string, exclude map[string]bool) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Skip excluded files
		if exclude[filepath.Base(path)] {
			return nil
		}

		dstPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, 0o644)
	})
}

// setupOpts configures setupTestdataModule behavior.
type setupOpts struct {
	forRemove bool // if true, use after.go as input for remove testing
}

// setupTestdataModule prepares a testdata case for Process-based testing.
// It copies _stubs and the case directory to a temp location.
func setupTestdataModule(t *testing.T, testdataRoot, caseName string, opts setupOpts) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Resolve symlinks (macOS /var -> /private/var)
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	// Copy _stubs directory
	stubsSrc := filepath.Join(testdataRoot, "_stubs")
	stubsDst := filepath.Join(tmpDir, "_stubs")
	if err := copyDir(stubsSrc, stubsDst, nil); err != nil {
		t.Fatalf("failed to copy _stubs: %v", err)
	}

	caseSrc := filepath.Join(testdataRoot, caseName)
	caseDst := filepath.Join(tmpDir, "case")

	if opts.forRemove {
		// For remove tests: copy only go.mod and after.go (as input.go)
		if err := os.MkdirAll(caseDst, 0o755); err != nil {
			t.Fatalf("failed to create case dir: %v", err)
		}

		// Copy and update go.mod
		goModData, err := os.ReadFile(filepath.Join(caseSrc, "go.mod"))
		if err != nil {
			t.Fatalf("failed to read go.mod: %v", err)
		}
		updated := strings.ReplaceAll(string(goModData), "../../_stubs", stubsDst)
		updated = strings.ReplaceAll(updated, "../_stubs", stubsDst)
		if err := os.WriteFile(filepath.Join(caseDst, "go.mod"), []byte(updated), 0o644); err != nil {
			t.Fatalf("failed to write go.mod: %v", err)
		}

		// Copy after.go as input.go
		afterData, err := os.ReadFile(filepath.Join(caseSrc, "after.go"))
		if err != nil {
			t.Fatalf("failed to read after.go: %v", err)
		}
		if err := os.WriteFile(filepath.Join(caseDst, "input.go"), afterData, 0o644); err != nil {
			t.Fatalf("failed to write input.go: %v", err)
		}
	} else {
		// For insertion tests: copy case directory excluding after.go and config.yaml
		exclude := map[string]bool{
			"after.go":    true,
			"config.yaml": true,
		}
		if err := copyDir(caseSrc, caseDst, exclude); err != nil {
			t.Fatalf("failed to copy case %s: %v", caseName, err)
		}

		// Update go.mod replace directives
		goModPath := filepath.Join(caseDst, "go.mod")
		goModData, err := os.ReadFile(goModPath)
		if err == nil {
			updated := strings.ReplaceAll(string(goModData), "../../_stubs", stubsDst)
			updated = strings.ReplaceAll(updated, "../_stubs", stubsDst)
			if err := os.WriteFile(goModPath, []byte(updated), 0o644); err != nil {
				t.Fatalf("failed to update go.mod: %v", err)
			}
		}
	}

	return caseDst
}

// TestInsertion tests transformation using testdata files.
// Each subdirectory in testdata/ contains before.go, after.go, and optional config.yaml.
func TestInsertion(t *testing.T) {
	testdataRoot, err := filepath.Abs(filepath.Join("..", "..", "internal", "testdata"))
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	cases := discoverTestCases(t, testdataRoot)
	for _, tc := range cases {
		if !tc.hasAfter {
			// Skip cases without after.go (idempotency only)
			continue
		}
		runInsertionTest(t, testdataRoot, tc.name)
	}
}

func runInsertionTest(t *testing.T, testdataRoot, caseName string) {
	t.Run(caseName, func(t *testing.T) {
		origDir := filepath.Join(testdataRoot, caseName)

		// Check if go.mod exists (required for Process)
		goModPath := filepath.Join(origDir, "go.mod")
		if _, err := os.Stat(goModPath); os.IsNotExist(err) {
			t.Skip("no go.mod - skipping (needs setup)")
			return
		}

		afterPath := filepath.Join(origDir, "after.go")
		want, err := os.ReadFile(afterPath)
		if err != nil {
			t.Fatalf("failed to read after.go: %v", err)
		}

		cfg := loadTestConfig(origDir)
		caseDir := setupTestdataModule(t, testdataRoot, caseName, setupOpts{})

		registry := config.NewCarrierRegistry(true)

		tmpl, err := template.Parse(cfg.Template)
		if err != nil {
			t.Fatalf("failed to parse template: %v", err)
		}

		proc := processor.New(registry, tmpl, cfg.Imports)

		oldWd, _ := os.Getwd()
		if err := os.Chdir(caseDir); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}
		defer func() { _ = os.Chdir(oldWd) }()

		if _, err = proc.Process([]string{"./..."}); err != nil {
			t.Fatalf("Process failed: %v", err)
		}

		got, err := os.ReadFile(filepath.Join(caseDir, "before.go"))
		if err != nil {
			t.Fatalf("failed to read result: %v", err)
		}

		if diff := cmp.Diff(string(want), string(got)); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
}

// TestRemove tests remove mode using testdata files (reverse of TestInsertion).
// Uses after.go as input and expects before.go as output.
func TestRemove(t *testing.T) {
	testdataRoot, err := filepath.Abs(filepath.Join("..", "..", "internal", "testdata"))
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	cases := discoverTestCases(t, testdataRoot)
	for _, tc := range cases {
		if !tc.hasAfter {
			continue
		}
		runRemoveTest(t, testdataRoot, tc.name)
	}
}

func runRemoveTest(t *testing.T, testdataRoot, caseName string) {
	t.Run(caseName, func(t *testing.T) {
		origDir := filepath.Join(testdataRoot, caseName)

		// Check if go.mod exists (required for Process)
		goModPath := filepath.Join(origDir, "go.mod")
		if _, err := os.Stat(goModPath); os.IsNotExist(err) {
			t.Skip("no go.mod - skipping (needs setup)")
			return
		}

		cfg := loadTestConfig(origDir)
		if cfg.SkipRemove {
			t.Skip("skip_remove is set in config.yaml")
			return
		}

		beforePath := filepath.Join(origDir, "before.go")
		want, err := os.ReadFile(beforePath)
		if err != nil {
			t.Fatalf("failed to read before.go: %v", err)
		}

		caseDir := setupTestdataModule(t, testdataRoot, caseName, setupOpts{forRemove: true})

		registry := config.NewCarrierRegistry(true)

		tmpl, err := template.Parse(cfg.Template)
		if err != nil {
			t.Fatalf("failed to parse template: %v", err)
		}

		proc := processor.New(registry, tmpl, cfg.Imports, processor.WithRemove(true))

		oldWd, _ := os.Getwd()
		if err := os.Chdir(caseDir); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}
		defer func() { _ = os.Chdir(oldWd) }()

		if _, err = proc.Process([]string{"./..."}); err != nil {
			t.Fatalf("Process failed: %v", err)
		}

		got, err := os.ReadFile(filepath.Join(caseDir, "input.go"))
		if err != nil {
			t.Fatalf("failed to read result: %v", err)
		}

		if diff := cmp.Diff(string(want), string(got)); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
}

// TestIdempotency tests that transformation is idempotent.
func TestIdempotency(t *testing.T) {
	testdataRoot, err := filepath.Abs(filepath.Join("..", "..", "internal", "testdata"))
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	cases := discoverTestCases(t, testdataRoot)
	for _, tc := range cases {
		runIdempotencyTest(t, testdataRoot, tc.name)
	}
}

func runIdempotencyTest(t *testing.T, testdataRoot, caseName string) {
	t.Run(caseName, func(t *testing.T) {
		origDir := filepath.Join(testdataRoot, caseName)

		// Check if go.mod exists (required for Process)
		goModPath := filepath.Join(origDir, "go.mod")
		if _, err := os.Stat(goModPath); os.IsNotExist(err) {
			t.Skip("no go.mod - skipping (needs setup)")
			return
		}

		cfg := loadTestConfig(origDir)
		caseDir := setupTestdataModule(t, testdataRoot, caseName, setupOpts{})

		registry := config.NewCarrierRegistry(true)

		tmpl, err := template.Parse(cfg.Template)
		if err != nil {
			t.Fatalf("failed to parse template: %v", err)
		}

		proc := processor.New(registry, tmpl, cfg.Imports)

		oldWd, _ := os.Getwd()
		if err := os.Chdir(caseDir); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}
		defer func() { _ = os.Chdir(oldWd) }()

		resultPath := filepath.Join(caseDir, "before.go")

		// Run three transformations and verify idempotency
		var results [3][]byte
		for i := range results {
			if _, err = proc.Process([]string{"./..."}); err != nil {
				t.Fatalf("Process #%d failed: %v", i+1, err)
			}
			results[i], err = os.ReadFile(resultPath)
			if err != nil {
				t.Fatalf("failed to read result #%d: %v", i+1, err)
			}
		}

		if diff := cmp.Diff(string(results[0]), string(results[1])); diff != "" {
			t.Errorf("first vs second mismatch:\n%s", diff)
		}
		if diff := cmp.Diff(string(results[1]), string(results[2])); diff != "" {
			t.Errorf("second vs third mismatch:\n%s", diff)
		}
	})
}
