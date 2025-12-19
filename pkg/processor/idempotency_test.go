package processor_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/mpyw/ctxweaver/pkg/config"
	"github.com/mpyw/ctxweaver/pkg/processor"
	"github.com/mpyw/ctxweaver/pkg/template"
)

// TestIdempotency_EdgeCases tests various edge cases for idempotency.
// These tests ensure that running ctxweaver multiple times produces stable results.
func TestIdempotency_EdgeCases(t *testing.T) {
	testCases := map[string]struct {
		template string
		imports  []string
		before   string
		pkgName  string
	}{
		// === Basic Cases ===
		"simple_defer": {
			template: `defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`,
			imports:  []string{"github.com/newrelic/go-agent/v3/newrelic"},
			before: `package test
import "context"
func Foo(ctx context.Context) error {
	return nil
}`,
			pkgName: "test",
		},

		// === Multi-line Templates ===
		"multiline_func_literal": {
			template: `defer func() {
	txn := newrelic.FromContext({{.Ctx}})
	defer txn.StartSegment({{.FuncName | quote}}).End()
}()`,
			imports: []string{"github.com/newrelic/go-agent/v3/newrelic"},
			before: `package test
import "context"
func Foo(ctx context.Context) error {
	return nil
}`,
			pkgName: "test",
		},

		// === If Statement Templates ===
		"if_statement": {
			template: `if txn := newrelic.FromContext({{.Ctx}}); txn != nil {
	defer txn.StartSegment({{.FuncName | quote}}).End()
}`,
			imports: []string{"github.com/newrelic/go-agent/v3/newrelic"},
			before: `package test
import "context"
func Foo(ctx context.Context) error {
	return nil
}`,
			pkgName: "test",
		},

		// === Simple Function Call ===
		"simple_call": {
			template: `trace.Start({{.Ctx}}, {{.FuncName | quote}})`,
			imports:  []string{"mytrace"},
			before: `package test
import "context"
func Foo(ctx context.Context) error {
	return nil
}`,
			pkgName: "test",
		},

		// === Variable Assignment ===
		"assignment": {
			template: `span, _ := tracer.StartSpanFromContext({{.Ctx}}, {{.FuncName | quote}})
defer span.Finish()`,
			imports: []string{"opentracing"},
			before: `package test
import "context"
func Foo(ctx context.Context) error {
	return nil
}`,
			pkgName: "test",
		},

		// === Existing Code with Comments ===
		"with_existing_comment": {
			template: `defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`,
			imports:  []string{"github.com/newrelic/go-agent/v3/newrelic"},
			before: `package test
import "context"
func Foo(ctx context.Context) error {
	// Important business logic below
	return nil
}`,
			pkgName: "test",
		},

		// === Method with Pointer Receiver ===
		"method_pointer_receiver": {
			template: `defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`,
			imports:  []string{"github.com/newrelic/go-agent/v3/newrelic"},
			before: `package test
import "context"
type Service struct{}
func (s *Service) Do(ctx context.Context) error {
	return nil
}`,
			pkgName: "test",
		},

		// === Method with Value Receiver ===
		"method_value_receiver": {
			template: `defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`,
			imports:  []string{"github.com/newrelic/go-agent/v3/newrelic"},
			before: `package test
import "context"
type Handler struct{}
func (h Handler) Handle(ctx context.Context) error {
	return nil
}`,
			pkgName: "test",
		},

		// === Different Context Variable Name ===
		"different_ctx_var_name": {
			template: `defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`,
			imports:  []string{"github.com/newrelic/go-agent/v3/newrelic"},
			before: `package test
import "context"
func Foo(c context.Context) error {
	return nil
}`,
			pkgName: "test",
		},

		// === Multiple Functions in One File ===
		"multiple_functions": {
			template: `defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`,
			imports:  []string{"github.com/newrelic/go-agent/v3/newrelic"},
			before: `package test
import "context"
func Foo(ctx context.Context) error {
	return nil
}
func Bar(ctx context.Context) error {
	return nil
}
func Baz(ctx context.Context) error {
	return nil
}`,
			pkgName: "test",
		},

		// === Complex Nested Structure ===
		"complex_body": {
			template: `defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`,
			imports:  []string{"github.com/newrelic/go-agent/v3/newrelic"},
			before: `package test
import "context"
func Foo(ctx context.Context) error {
	if true {
		for i := 0; i < 10; i++ {
			switch i {
			case 1:
				break
			}
		}
	}
	return nil
}`,
			pkgName: "test",
		},

		// === Blank Lines Preservation ===
		"blank_lines": {
			template: `defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`,
			imports:  []string{"github.com/newrelic/go-agent/v3/newrelic"},
			before: `package test

import "context"

func Foo(ctx context.Context) error {

	// Some comment

	return nil
}`,
			pkgName: "test",
		},

		// === Unicode in Function Name ===
		"unicode_package": {
			template: `defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`,
			imports:  []string{"github.com/newrelic/go-agent/v3/newrelic"},
			before: `package test
import "context"
func 処理(ctx context.Context) error {
	return nil
}`,
			pkgName: "test",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			registry, err := config.NewCarrierRegistry()
			if err != nil {
				t.Fatalf("failed to create carrier registry: %v", err)
			}

			tmpl, err := template.Parse(tc.template)
			if err != nil {
				t.Fatalf("failed to parse template: %v", err)
			}

			proc := processor.New(
				registry,
				tmpl,
				tc.imports,
			)

			// First transformation
			first, err := proc.TransformSource([]byte(tc.before), tc.pkgName)
			if err != nil {
				t.Fatalf("first TransformSource failed: %v", err)
			}

			// Second transformation (should be stable)
			second, err := proc.TransformSource(first, tc.pkgName)
			if err != nil {
				t.Fatalf("second TransformSource failed: %v", err)
			}

			// Third transformation (should still be stable)
			third, err := proc.TransformSource(second, tc.pkgName)
			if err != nil {
				t.Fatalf("third TransformSource failed: %v", err)
			}

			// Compare first and second
			if diff := cmp.Diff(string(first), string(second)); diff != "" {
				t.Errorf("first vs second mismatch (NOT IDEMPOTENT):\n%s\n\nFirst:\n%s\n\nSecond:\n%s",
					diff, string(first), string(second))
			}

			// Compare second and third
			if diff := cmp.Diff(string(second), string(third)); diff != "" {
				t.Errorf("second vs third mismatch (NOT STABLE):\n%s", diff)
			}
		})
	}
}

// TestIdempotency_TemplateChange_SkeletonMode tests behavior when template changes in skeleton mode.
// In skeleton mode, different function names (trace1 vs trace2) are considered different structures,
// so a template change will INSERT a new statement rather than UPDATE the existing one.
// For template updates, use marker mode instead.
func TestIdempotency_TemplateChange_SkeletonMode(t *testing.T) {
	registry, err := config.NewCarrierRegistry()
	if err != nil {
		t.Fatalf("failed to create carrier registry: %v", err)
	}

	before := `package test
import "context"
func Foo(ctx context.Context) error {
	return nil
}`

	// First template
	tmpl1, _ := template.Parse(`defer trace1({{.Ctx}}, {{.FuncName | quote}})`)
	proc1 := processor.New(registry, tmpl1, []string{"trace1"})

	first, err := proc1.TransformSource([]byte(before), "test")
	if err != nil {
		t.Fatalf("first transform failed: %v", err)
	}

	if !strings.Contains(string(first), "trace1") {
		t.Error("first transform should contain trace1")
	}

	// Second template (different from first) - IN SKELETON MODE, this inserts a new statement
	tmpl2, _ := template.Parse(`defer trace2({{.Ctx}}, {{.FuncName | quote}})`)
	proc2 := processor.New(registry, tmpl2, []string{"trace2"})

	second, err := proc2.TransformSource(first, "test")
	if err != nil {
		t.Fatalf("second transform failed: %v", err)
	}

	// In skeleton mode: BOTH trace1 and trace2 will exist (insert, not update)
	if !strings.Contains(string(second), "trace2") {
		t.Error("second transform should contain trace2")
	}
	if !strings.Contains(string(second), "trace1") {
		t.Error("skeleton mode: trace1 should still exist (different structure)")
	}
}

// TestIdempotency_TemplateChange_MarkerMode tests that marker mode correctly updates on template change.
func TestIdempotency_TemplateChange_MarkerMode(t *testing.T) {
	registry, err := config.NewCarrierRegistry()
	if err != nil {
		t.Fatalf("failed to create carrier registry: %v", err)
	}

	before := `package test
import "context"
func Foo(ctx context.Context) error {
	return nil
}`

	// First template with marker mode
	tmpl1, _ := template.Parse(`defer trace1({{.Ctx}}, {{.FuncName | quote}})`)
	proc1 := processor.New(registry, tmpl1, []string{"trace1"}, processor.WithMatchMode(processor.MatchModeMarker))

	first, err := proc1.TransformSource([]byte(before), "test")
	if err != nil {
		t.Fatalf("first transform failed: %v", err)
	}

	if !strings.Contains(string(first), "trace1") {
		t.Error("first transform should contain trace1")
	}
	if !strings.Contains(string(first), "//ctxweaver:generated") {
		t.Error("marker mode should add marker")
	}

	// Second template (different from first) - IN MARKER MODE, this updates the marked statement
	tmpl2, _ := template.Parse(`defer trace2({{.Ctx}}, {{.FuncName | quote}})`)
	proc2 := processor.New(registry, tmpl2, []string{"trace2"}, processor.WithMatchMode(processor.MatchModeMarker))

	second, err := proc2.TransformSource(first, "test")
	if err != nil {
		t.Fatalf("second transform failed: %v", err)
	}

	// In marker mode: trace2 replaces trace1
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
		t.Errorf("marker mode template change not stable:\n%s", diff)
	}
}

// TestIdempotency_PreservesExistingCode tests that existing code is not modified.
func TestIdempotency_PreservesExistingCode(t *testing.T) {
	registry, _ := config.NewCarrierRegistry()
	tmpl, _ := template.Parse(`defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`)
	proc := processor.New(registry, tmpl, []string{"github.com/newrelic/go-agent/v3/newrelic"})

	before := `package test
import "context"
func Foo(ctx context.Context) error {
	// This comment should be preserved
	x := 1
	y := 2
	return doSomething(x, y)
}
func doSomething(a, b int) error {
	return nil
}`

	first, err := proc.TransformSource([]byte(before), "test")
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}

	// Check that existing code is preserved
	checks := []string{
		"This comment should be preserved",
		"x := 1",
		"y := 2",
		"doSomething(x, y)",
		"func doSomething",
	}

	for _, check := range checks {
		if !strings.Contains(string(first), check) {
			t.Errorf("missing expected code: %q", check)
		}
	}
}

// TestIdempotency_SkipDirective tests that skip directive works with idempotency.
func TestIdempotency_SkipDirective(t *testing.T) {
	registry, _ := config.NewCarrierRegistry()
	tmpl, _ := template.Parse(`defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`)
	proc := processor.New(registry, tmpl, []string{"github.com/newrelic/go-agent/v3/newrelic"})

	before := `package test
import "context"
//ctxweaver:skip
func SkippedFunc(ctx context.Context) error {
	return nil
}
func NormalFunc(ctx context.Context) error {
	return nil
}`

	first, err := proc.TransformSource([]byte(before), "test")
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}

	// Skipped function should not have instrumentation
	if strings.Contains(string(first), `"test.SkippedFunc"`) {
		t.Error("SkippedFunc should not be instrumented")
	}

	// Normal function should have instrumentation
	if !strings.Contains(string(first), `"test.NormalFunc"`) {
		t.Error("NormalFunc should be instrumented")
	}

	// Run again - should be stable
	second, err := proc.TransformSource(first, "test")
	if err != nil {
		t.Fatalf("second transform failed: %v", err)
	}

	if diff := cmp.Diff(string(first), string(second)); diff != "" {
		t.Errorf("skip directive not stable:\n%s", diff)
	}
}

// TestIdempotency_SimilarStatements tests detection of similar but different statements.
func TestIdempotency_SimilarStatements(t *testing.T) {
	registry, _ := config.NewCarrierRegistry()
	tmpl, _ := template.Parse(`defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`)
	proc := processor.New(registry, tmpl, []string{"github.com/newrelic/go-agent/v3/newrelic"})

	// Code with a similar-looking defer statement that's NOT our generated code
	before := `package test
import (
	"context"
	"github.com/newrelic/go-agent/v3/newrelic"
)
func Foo(ctx context.Context) error {
	defer someOtherFunc()
	return nil
}`

	first, err := proc.TransformSource([]byte(before), "test")
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}

	// Should have both: our generated statement AND the existing defer
	if !strings.Contains(string(first), `"test.Foo"`) {
		t.Error("should have generated statement")
	}
	if !strings.Contains(string(first), "someOtherFunc()") {
		t.Error("should preserve existing defer")
	}

	// Count defer statements
	deferCount := strings.Count(string(first), "defer ")
	if deferCount != 2 {
		t.Errorf("expected 2 defer statements, got %d\n%s", deferCount, string(first))
	}

	// Run again - should be stable
	second, err := proc.TransformSource(first, "test")
	if err != nil {
		t.Fatalf("second transform failed: %v", err)
	}

	if diff := cmp.Diff(string(first), string(second)); diff != "" {
		t.Errorf("similar statements not stable:\n%s", diff)
	}
}

// TestIdempotency_MarkerMode tests that marker mode works correctly.
func TestIdempotency_MarkerMode(t *testing.T) {
	registry, _ := config.NewCarrierRegistry()
	tmpl, _ := template.Parse(`defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`)
	proc := processor.New(
		registry,
		tmpl,
		[]string{"github.com/newrelic/go-agent/v3/newrelic"},
		processor.WithMatchMode(processor.MatchModeMarker),
	)

	before := `package test
import "context"
func Foo(ctx context.Context) error {
	return nil
}`

	first, err := proc.TransformSource([]byte(before), "test")
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}

	// Should have marker
	if !strings.Contains(string(first), "//ctxweaver:generated") {
		t.Error("marker mode should add marker")
	}

	// Run again - should be stable
	second, err := proc.TransformSource(first, "test")
	if err != nil {
		t.Fatalf("second transform failed: %v", err)
	}

	if diff := cmp.Diff(string(first), string(second)); diff != "" {
		t.Errorf("marker mode not stable:\n%s", diff)
	}
}

// TestIdempotency_BlockStatement tests block statement templates.
func TestIdempotency_BlockStatement(t *testing.T) {
	registry, _ := config.NewCarrierRegistry()
	tmpl, _ := template.Parse(`{
	txn := newrelic.FromContext({{.Ctx}})
	defer txn.StartSegment({{.FuncName | quote}}).End()
}`)
	proc := processor.New(registry, tmpl, []string{"github.com/newrelic/go-agent/v3/newrelic"})

	before := `package test
import "context"
func Foo(ctx context.Context) error {
	return nil
}`

	first, err := proc.TransformSource([]byte(before), "test")
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}

	second, err := proc.TransformSource(first, "test")
	if err != nil {
		t.Fatalf("second transform failed: %v", err)
	}

	third, err := proc.TransformSource(second, "test")
	if err != nil {
		t.Fatalf("third transform failed: %v", err)
	}

	if diff := cmp.Diff(string(first), string(second)); diff != "" {
		t.Errorf("block statement not idempotent (first vs second):\n%s", diff)
	}

	if diff := cmp.Diff(string(second), string(third)); diff != "" {
		t.Errorf("block statement not stable (second vs third):\n%s", diff)
	}
}

// TestIdempotency_SwitchStatement tests switch statement templates.
func TestIdempotency_SwitchStatement(t *testing.T) {
	registry, _ := config.NewCarrierRegistry()
	tmpl, _ := template.Parse(`switch txn := newrelic.FromContext({{.Ctx}}); {
case txn != nil:
	defer txn.StartSegment({{.FuncName | quote}}).End()
}`)
	proc := processor.New(registry, tmpl, []string{"github.com/newrelic/go-agent/v3/newrelic"})

	before := `package test
import "context"
func Foo(ctx context.Context) error {
	return nil
}`

	first, err := proc.TransformSource([]byte(before), "test")
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}

	second, err := proc.TransformSource(first, "test")
	if err != nil {
		t.Fatalf("second transform failed: %v", err)
	}

	if diff := cmp.Diff(string(first), string(second)); diff != "" {
		t.Errorf("switch statement not idempotent:\n%s", diff)
	}
}

// TestIdempotency_ManyRuns tests stability over many runs.
func TestIdempotency_ManyRuns(t *testing.T) {
	registry, _ := config.NewCarrierRegistry()
	tmpl, _ := template.Parse(`defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`)
	proc := processor.New(registry, tmpl, []string{"github.com/newrelic/go-agent/v3/newrelic"})

	before := `package test
import "context"
func Foo(ctx context.Context) error {
	// existing code
	x := 1
	return process(x)
}
func Bar(ctx context.Context, a, b int) int {
	return a + b
}`

	current := []byte(before)
	var err error

	// Run 10 times
	for i := 0; i < 10; i++ {
		next, err := proc.TransformSource(current, "test")
		if err != nil {
			t.Fatalf("run %d failed: %v", i, err)
		}

		// After the first run, all subsequent runs should produce identical output
		if i > 0 {
			if diff := cmp.Diff(string(current), string(next)); diff != "" {
				t.Errorf("run %d not stable:\n%s", i, diff)
			}
		}
		current = next
	}
	_ = err // suppress unused
}

// TestIdempotency_TrailingNewlines tests handling of various trailing newline patterns.
func TestIdempotency_TrailingNewlines(t *testing.T) {
	registry, _ := config.NewCarrierRegistry()
	tmpl, _ := template.Parse(`defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`)
	proc := processor.New(registry, tmpl, []string{"github.com/newrelic/go-agent/v3/newrelic"})

	testCases := map[string]string{
		"no_trailing": `package test
import "context"
func Foo(ctx context.Context) error {
	return nil
}`,
		"one_trailing": `package test
import "context"
func Foo(ctx context.Context) error {
	return nil
}
`,
		"multiple_trailing": `package test
import "context"
func Foo(ctx context.Context) error {
	return nil
}


`,
	}

	for name, before := range testCases {
		t.Run(name, func(t *testing.T) {
			first, err := proc.TransformSource([]byte(before), "test")
			if err != nil {
				t.Fatalf("first transform failed: %v", err)
			}

			second, err := proc.TransformSource(first, "test")
			if err != nil {
				t.Fatalf("second transform failed: %v", err)
			}

			if diff := cmp.Diff(string(first), string(second)); diff != "" {
				t.Errorf("not idempotent:\n%s", diff)
			}
		})
	}
}

// TestIdempotency_VariousContextVarNames tests different context variable naming patterns.
func TestIdempotency_VariousContextVarNames(t *testing.T) {
	registry, _ := config.NewCarrierRegistry()
	tmpl, _ := template.Parse(`defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`)
	proc := processor.New(registry, tmpl, []string{"github.com/newrelic/go-agent/v3/newrelic"})

	testCases := map[string]string{
		"ctx": `package test
import "context"
func Foo(ctx context.Context) error { return nil }`,
		"c": `package test
import "context"
func Foo(c context.Context) error { return nil }`,
		"parentCtx": `package test
import "context"
func Foo(parentCtx context.Context) error { return nil }`,
		"reqCtx": `package test
import "context"
func Foo(reqCtx context.Context) error { return nil }`,
	}

	for name, before := range testCases {
		t.Run(name, func(t *testing.T) {
			first, err := proc.TransformSource([]byte(before), "test")
			if err != nil {
				t.Fatalf("first transform failed: %v", err)
			}

			second, err := proc.TransformSource(first, "test")
			if err != nil {
				t.Fatalf("second transform failed: %v", err)
			}

			if diff := cmp.Diff(string(first), string(second)); diff != "" {
				t.Errorf("not idempotent for var name %s:\n%s", name, diff)
			}
		})
	}
}
