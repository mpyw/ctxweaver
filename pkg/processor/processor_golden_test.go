package processor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/mpyw/ctxweaver/pkg/config"
	"github.com/mpyw/ctxweaver/pkg/processor"
	"github.com/mpyw/ctxweaver/pkg/template"
)

// newrelicTemplate is the standard New Relic instrumentation template
const newrelicTemplate = `defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`

func TestTransformSource_Golden(t *testing.T) {
	testCases := map[string]struct {
		dir string
	}{
		"basic":          {dir: "basic"},
		"method_pointer": {dir: "method_pointer"},
		"method_value":   {dir: "method_value"},
		"echo_context":   {dir: "echo_context"},
		"skip_directive": {dir: "skip_directive"},
		"already_exists": {dir: "already_exists"},
		"multiple_funcs": {dir: "multiple_funcs"},
		"no_context":     {dir: "no_context"},
	}

	// Setup processor
	registry, err := config.NewCarrierRegistry()
	if err != nil {
		t.Fatalf("failed to create carrier registry: %v", err)
	}

	tmpl, err := template.Parse(newrelicTemplate)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}

	proc := processor.New(
		registry,
		tmpl,
		[]string{"github.com/newrelic/go-agent/v3/newrelic"},
	)

	testdataRoot := filepath.Join("..", "..", "internal", "testdata")

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(testdataRoot, tc.dir)

			beforePath := filepath.Join(dir, "before.go")
			afterPath := filepath.Join(dir, "after.go")

			before, err := os.ReadFile(beforePath)
			if err != nil {
				t.Fatalf("failed to read before.go: %v", err)
			}

			want, err := os.ReadFile(afterPath)
			if err != nil {
				t.Fatalf("failed to read after.go: %v", err)
			}

			got, err := proc.TransformSource(before, tc.dir)
			if err != nil {
				t.Fatalf("TransformSource failed: %v", err)
			}

			if diff := cmp.Diff(string(want), string(got)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
