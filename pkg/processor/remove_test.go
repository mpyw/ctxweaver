package processor_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/mpyw/ctxweaver/pkg/config"
	"github.com/mpyw/ctxweaver/pkg/processor"
	"github.com/mpyw/ctxweaver/pkg/template"
)

// TestRemove_Basic tests that remove mode removes matching statements.
func TestRemove_Basic(t *testing.T) {
	registry, _ := config.NewCarrierRegistry()
	tmpl, _ := template.Parse(`defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`)

	insertProc := processor.New(registry, tmpl, []string{"github.com/newrelic/go-agent/v3/newrelic"})
	before := readTestdataFile(t, "basic_newrelic", "before.go")

	instrumented, err := insertProc.TransformSource(before, "test")
	if err != nil {
		t.Fatalf("insert transform failed: %v", err)
	}

	if !strings.Contains(string(instrumented), "StartSegment") {
		t.Fatal("instrumentation should have been added")
	}

	removeProc := processor.New(registry, tmpl, []string{"github.com/newrelic/go-agent/v3/newrelic"}, processor.WithRemove(true))

	removed, err := removeProc.TransformSource(instrumented, "test")
	if err != nil {
		t.Fatalf("remove transform failed: %v", err)
	}

	if strings.Contains(string(removed), "StartSegment") {
		t.Error("instrumentation should have been removed")
	}

	// Should be idempotent
	removedAgain, err := removeProc.TransformSource(removed, "test")
	if err != nil {
		t.Fatalf("second remove transform failed: %v", err)
	}

	if diff := cmp.Diff(string(removed), string(removedAgain)); diff != "" {
		t.Errorf("remove not idempotent:\n%s", diff)
	}
}

// TestRemove_MultiStatement tests remove mode with multi-statement templates.
func TestRemove_MultiStatement(t *testing.T) {
	registry, _ := config.NewCarrierRegistry()
	tmpl, _ := template.Parse(`ctx, span := otel.Tracer("").Start({{.Ctx}}, {{.FuncName | quote}})
defer span.End()`)

	insertProc := processor.New(registry, tmpl, []string{"go.opentelemetry.io/otel"})
	before := readTestdataFile(t, "basic_otel", "before.go")

	instrumented, err := insertProc.TransformSource(before, "test")
	if err != nil {
		t.Fatalf("insert transform failed: %v", err)
	}

	if !strings.Contains(string(instrumented), "otel.Tracer") {
		t.Fatal("instrumentation should have been added")
	}
	if !strings.Contains(string(instrumented), "defer span.End()") {
		t.Fatal("defer should have been added")
	}

	removeProc := processor.New(registry, tmpl, []string{"go.opentelemetry.io/otel"}, processor.WithRemove(true))

	removed, err := removeProc.TransformSource(instrumented, "test")
	if err != nil {
		t.Fatalf("remove transform failed: %v", err)
	}

	if strings.Contains(string(removed), "otel.Tracer") {
		t.Error("instrumentation should have been removed")
	}
	if strings.Contains(string(removed), "defer span.End()") {
		t.Error("defer should have been removed")
	}

	// Should be idempotent
	removedAgain, err := removeProc.TransformSource(removed, "test")
	if err != nil {
		t.Fatalf("second remove transform failed: %v", err)
	}

	if diff := cmp.Diff(string(removed), string(removedAgain)); diff != "" {
		t.Errorf("remove not idempotent:\n%s", diff)
	}
}

// TestRemove_PreservesOtherCode tests that remove mode preserves unrelated code.
func TestRemove_PreservesOtherCode(t *testing.T) {
	registry, _ := config.NewCarrierRegistry()
	tmpl, _ := template.Parse(`defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`)

	insertProc := processor.New(registry, tmpl, []string{"github.com/newrelic/go-agent/v3/newrelic"})
	before := readTestdataFile(t, "preserves_existing_code", "before.go")

	instrumented, err := insertProc.TransformSource(before, "test")
	if err != nil {
		t.Fatalf("insert transform failed: %v", err)
	}

	removeProc := processor.New(registry, tmpl, []string{"github.com/newrelic/go-agent/v3/newrelic"}, processor.WithRemove(true))

	removed, err := removeProc.TransformSource(instrumented, "test")
	if err != nil {
		t.Fatalf("remove transform failed: %v", err)
	}

	checks := []string{
		"This comment should be preserved",
		"x := 1",
		"y := 2",
		"doSomething(x, y)",
	}

	for _, check := range checks {
		if !strings.Contains(string(removed), check) {
			t.Errorf("missing expected code: %q", check)
		}
	}
}

// TestRemove_DifferentTemplate tests that remove only removes matching statements.
func TestRemove_DifferentTemplate(t *testing.T) {
	registry, _ := config.NewCarrierRegistry()

	before := readTestdataFile(t, "basic_newrelic", "before.go")

	tmpl1, _ := template.Parse(`defer trace1({{.Ctx}}, {{.FuncName | quote}})`)
	proc1 := processor.New(registry, tmpl1, []string{"trace1"})

	withTrace1, err := proc1.TransformSource(before, "test")
	if err != nil {
		t.Fatalf("insert trace1 failed: %v", err)
	}

	tmpl2, _ := template.Parse(`defer trace2({{.Ctx}}, {{.FuncName | quote}})`)
	proc2 := processor.New(registry, tmpl2, []string{"trace2"})

	withBoth, err := proc2.TransformSource(withTrace1, "test")
	if err != nil {
		t.Fatalf("insert trace2 failed: %v", err)
	}

	if !strings.Contains(string(withBoth), "trace1") || !strings.Contains(string(withBoth), "trace2") {
		t.Fatal("both traces should be present")
	}

	removeProc1 := processor.New(registry, tmpl1, []string{"trace1"}, processor.WithRemove(true))

	afterRemove1, err := removeProc1.TransformSource(withBoth, "test")
	if err != nil {
		t.Fatalf("remove trace1 failed: %v", err)
	}

	if strings.Contains(string(afterRemove1), "trace1") {
		t.Error("trace1 should have been removed")
	}
	if !strings.Contains(string(afterRemove1), "trace2") {
		t.Error("trace2 should still be present")
	}
}

// TestRemove_SkipDirective tests that statements with //ctxweaver:skip are not removed.
func TestRemove_SkipDirective(t *testing.T) {
	tests := map[string]struct {
		source string
	}{
		"skip directive before statement (no space)": {
			source: `package test

import "context"

func Foo(ctx context.Context) {
	//ctxweaver:skip
	defer trace(ctx, "test.Foo")
}
`,
		},
		"skip directive before statement (with space)": {
			source: `package test

import "context"

func Foo(ctx context.Context) {
	// ctxweaver:skip
	defer trace(ctx, "test.Foo")
}
`,
		},
		"skip directive trailing (no space)": {
			source: `package test

import "context"

func Foo(ctx context.Context) {
	defer trace(ctx, "test.Foo") //ctxweaver:skip
}
`,
		},
		"skip directive trailing (with space)": {
			source: `package test

import "context"

func Foo(ctx context.Context) {
	defer trace(ctx, "test.Foo") // ctxweaver:skip
}
`,
		},
		"LegacySimpleHandler - manually written exact match": {
			source: `package test

import "context"

func LegacySimpleHandler(ctx context.Context) {
	// ctxweaver:skip - manually written before ctxweaver existed
	defer trace(ctx, "test.LegacySimpleHandler")
	doWork()
}
`,
		},
		"LegacyEnrichedHandler - manually written with extra context": {
			source: `package test

import "context"

func LegacyEnrichedHandler(ctx context.Context) {
	// ctxweaver:skip
	// This trace was added manually for debugging production issue #1234
	// Do not auto-generate or remove this
	defer trace(ctx, "test.LegacyEnrichedHandler")
	doImportantWork()
}
`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			registry, _ := config.NewCarrierRegistry()
			tmpl, _ := template.Parse(`defer trace({{.Ctx}}, {{.FuncName | quote}})`)

			removeProc := processor.New(registry, tmpl, []string{"trace"}, processor.WithRemove(true))

			result, err := removeProc.TransformSource([]byte(tt.source), "test")
			if err != nil {
				t.Fatalf("remove transform failed: %v", err)
			}

			if !strings.Contains(string(result), "defer trace(ctx") {
				t.Error("statement with skip directive should NOT be removed")
			}

			if !strings.Contains(string(result), "ctxweaver:skip") {
				t.Error("skip directive should be preserved")
			}
		})
	}
}

// TestRemove_SkipDirective_PreservesImports tests that imports used by skipped statements are preserved.
func TestRemove_SkipDirective_PreservesImports(t *testing.T) {
	registry, _ := config.NewCarrierRegistry()
	tmpl, _ := template.Parse(`defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`)

	source := `package test

import (
	"context"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func LegacyHandler(ctx context.Context) {
	// ctxweaver:skip - manually instrumented
	defer newrelic.FromContext(ctx).StartSegment("test.LegacyHandler").End()
	doWork()
}
`

	removeProc := processor.New(registry, tmpl, []string{"github.com/newrelic/go-agent/v3/newrelic"}, processor.WithRemove(true))

	result, err := removeProc.TransformSource([]byte(source), "test")
	if err != nil {
		t.Fatalf("remove transform failed: %v", err)
	}

	if !strings.Contains(string(result), "newrelic.FromContext") {
		t.Error("statement with skip directive should NOT be removed")
	}

	if !strings.Contains(string(result), `"github.com/newrelic/go-agent/v3/newrelic"`) {
		t.Error("import used by skipped statement should be preserved")
	}
}

// TestSkipDirective_NoInsertOrUpdate tests that statements with skip directive are not updated.
func TestSkipDirective_NoInsertOrUpdate(t *testing.T) {
	registry, _ := config.NewCarrierRegistry()
	tmpl, _ := template.Parse(`defer trace({{.Ctx}}, {{.FuncName | quote}})`)

	source := `package test

import "context"

func Foo(ctx context.Context) {
	// ctxweaver:skip
	defer trace(ctx, "manually.Written")
}
`

	proc := processor.New(registry, tmpl, []string{"trace"})

	result, err := proc.TransformSource([]byte(source), "test")
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}

	if !strings.Contains(string(result), `"manually.Written"`) {
		t.Error("manually written statement should NOT be updated")
	}

	if strings.Contains(string(result), `"test.Foo"`) {
		t.Error("should not have auto-generated func name")
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

	if !strings.Contains(string(first), `"test.Foo"`) {
		t.Error("should have generated statement")
	}
	if !strings.Contains(string(first), "someOtherFunc()") {
		t.Error("should preserve existing defer")
	}

	deferCount := strings.Count(string(first), "defer ")
	if deferCount != 2 {
		t.Errorf("expected 2 defer statements, got %d", deferCount)
	}

	second, err := proc.TransformSource(first, "test")
	if err != nil {
		t.Fatalf("second transform failed: %v", err)
	}

	if diff := cmp.Diff(string(first), string(second)); diff != "" {
		t.Errorf("not stable:\n%s", diff)
	}
}
