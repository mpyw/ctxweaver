package processor

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

func TestParseStatements(t *testing.T) {
	tests := map[string]struct {
		input     string
		wantCount int
		wantErr   bool
	}{
		"single defer statement": {
			input:     `defer apm.StartSegment(ctx, "pkg.Func").End()`,
			wantCount: 1,
		},
		"single assignment statement": {
			input:     `ctx, span := tracer.Start(ctx, "func")`,
			wantCount: 1,
		},
		"single expression statement": {
			input:     `log.Info().Msg("hello")`,
			wantCount: 1,
		},
		"two statements": {
			input: `ctx, span := otel.Tracer("").Start(ctx, "test.Foo")
defer span.End()`,
			wantCount: 2,
		},
		"three statements": {
			input: `txn := newrelic.FromContext(ctx)
seg := txn.StartSegment("test.Foo")
defer seg.End()`,
			wantCount: 3,
		},
		"invalid statement": {
			input:   `if {`,
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			stmts, err := parseStatements(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseStatements() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(stmts) != tt.wantCount {
				t.Errorf("parseStatements() returned %d statements, want %d", len(stmts), tt.wantCount)
			}
		})
	}
}

func TestInsertStatements(t *testing.T) {
	t.Run("single statement", func(t *testing.T) {
		body := &dst.BlockStmt{
			List: []dst.Stmt{
				&dst.ExprStmt{X: &dst.Ident{Name: "existing"}},
			},
		}

		stmt := `defer trace(ctx)`
		if !insertStatements(body, stmt) {
			t.Error("insertStatements() returned false")
		}

		if len(body.List) != 2 {
			t.Errorf("body.List length = %d, want 2", len(body.List))
		}

		// First statement should be the inserted one
		def, ok := body.List[0].(*dst.DeferStmt)
		if !ok {
			t.Error("first statement is not a defer")
		}
		if def == nil {
			t.Error("first statement is nil")
		}
	})

	t.Run("multiple statements", func(t *testing.T) {
		body := &dst.BlockStmt{
			List: []dst.Stmt{
				&dst.ExprStmt{X: &dst.Ident{Name: "existing"}},
			},
		}

		stmt := `ctx, span := otel.Tracer("").Start(ctx, "test.Foo")
defer span.End()`
		if !insertStatements(body, stmt) {
			t.Error("insertStatements() returned false")
		}

		if len(body.List) != 3 {
			t.Errorf("body.List length = %d, want 3", len(body.List))
		}

		// First statement should be assignment
		_, ok := body.List[0].(*dst.AssignStmt)
		if !ok {
			t.Error("first statement is not an assignment")
		}

		// Second statement should be defer
		_, ok = body.List[1].(*dst.DeferStmt)
		if !ok {
			t.Error("second statement is not a defer")
		}
	})
}

func TestUpdateStatements(t *testing.T) {
	t.Run("single statement", func(t *testing.T) {
		body := &dst.BlockStmt{
			List: []dst.Stmt{
				mustParseStmt(t, `defer apm.StartSegment(ctx, "old.Func").End()`),
				&dst.ExprStmt{X: &dst.Ident{Name: "other"}},
			},
		}

		stmt := `defer apm.StartSegment(ctx, "new.Func").End()`
		if !updateStatements(body, 0, 1, stmt) {
			t.Error("updateStatements() returned false")
		}

		if len(body.List) != 2 {
			t.Errorf("body.List length = %d, want 2", len(body.List))
		}

		// First statement should be updated
		def, ok := body.List[0].(*dst.DeferStmt)
		if !ok {
			t.Fatal("first statement is not a defer")
		}
		if def == nil {
			t.Fatal("first statement is nil")
		}
	})

	t.Run("multiple statements", func(t *testing.T) {
		body := &dst.BlockStmt{
			List: []dst.Stmt{
				mustParseStmt(t, `ctx, span := otel.Tracer("").Start(ctx, "old.Func")`),
				mustParseStmt(t, `defer span.End()`),
				&dst.ExprStmt{X: &dst.Ident{Name: "other"}},
			},
		}

		stmt := `ctx, span := otel.Tracer("").Start(ctx, "new.Func")
defer span.End()`
		if !updateStatements(body, 0, 2, stmt) {
			t.Error("updateStatements() returned false")
		}

		if len(body.List) != 3 {
			t.Errorf("body.List length = %d, want 3", len(body.List))
		}

		// First statement should be assignment
		_, ok := body.List[0].(*dst.AssignStmt)
		if !ok {
			t.Fatal("first statement is not an assignment")
		}

		// Second statement should be defer
		_, ok = body.List[1].(*dst.DeferStmt)
		if !ok {
			t.Fatal("second statement is not a defer")
		}
	})

	t.Run("replace more with fewer statements", func(t *testing.T) {
		body := &dst.BlockStmt{
			List: []dst.Stmt{
				mustParseStmt(t, `a := 1`),
				mustParseStmt(t, `b := 2`),
				mustParseStmt(t, `c := 3`),
				&dst.ExprStmt{X: &dst.Ident{Name: "other"}},
			},
		}

		stmt := `x := 100`
		if !updateStatements(body, 0, 3, stmt) {
			t.Error("updateStatements() returned false")
		}

		if len(body.List) != 2 {
			t.Errorf("body.List length = %d, want 2", len(body.List))
		}
	})
}

func TestRemoveStatements(t *testing.T) {
	tests := map[string]struct {
		initialLen  int
		removeIdx   int
		removeCount int
		wantLen     int
		wantResult  bool
	}{
		"remove first single": {
			initialLen:  3,
			removeIdx:   0,
			removeCount: 1,
			wantLen:     2,
			wantResult:  true,
		},
		"remove middle single": {
			initialLen:  3,
			removeIdx:   1,
			removeCount: 1,
			wantLen:     2,
			wantResult:  true,
		},
		"remove last single": {
			initialLen:  3,
			removeIdx:   2,
			removeCount: 1,
			wantLen:     2,
			wantResult:  true,
		},
		"remove first two": {
			initialLen:  4,
			removeIdx:   0,
			removeCount: 2,
			wantLen:     2,
			wantResult:  true,
		},
		"remove middle two": {
			initialLen:  4,
			removeIdx:   1,
			removeCount: 2,
			wantLen:     2,
			wantResult:  true,
		},
		"invalid index negative": {
			initialLen:  3,
			removeIdx:   -1,
			removeCount: 1,
			wantLen:     3,
			wantResult:  false,
		},
		"invalid index too large": {
			initialLen:  3,
			removeIdx:   3,
			removeCount: 1,
			wantLen:     3,
			wantResult:  false,
		},
		"invalid count exceeds bounds": {
			initialLen:  3,
			removeIdx:   1,
			removeCount: 3,
			wantLen:     3,
			wantResult:  false,
		},
		"invalid count zero": {
			initialLen:  3,
			removeIdx:   0,
			removeCount: 0,
			wantLen:     3,
			wantResult:  false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create body with n statements
			body := &dst.BlockStmt{List: make([]dst.Stmt, tt.initialLen)}
			for i := range body.List {
				body.List[i] = &dst.ExprStmt{X: &dst.Ident{Name: "stmt"}}
			}

			got := removeStatements(body, tt.removeIdx, tt.removeCount)
			if got != tt.wantResult {
				t.Errorf("removeStatements() = %v, want %v", got, tt.wantResult)
			}
			if len(body.List) != tt.wantLen {
				t.Errorf("body.List length = %d, want %d", len(body.List), tt.wantLen)
			}
		})
	}
}

func TestStmtsToStrings(t *testing.T) {
	stmts := []dst.Stmt{
		mustParseStmt(t, `x := 1`),
		mustParseStmt(t, `defer foo()`),
	}

	result := stmtsToStrings(stmts)

	if len(result) != 2 {
		t.Fatalf("stmtsToStrings() returned %d strings, want 2", len(result))
	}

	if result[0] != "x := 1" {
		t.Errorf("stmtsToStrings()[0] = %q, want %q", result[0], "x := 1")
	}
	if result[1] != "defer foo()" {
		t.Errorf("stmtsToStrings()[1] = %q, want %q", result[1], "defer foo()")
	}
}

func mustParseStmt(t *testing.T, code string) dst.Stmt {
	t.Helper()
	src := "package p\nfunc f() {\n" + code + "\n}"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	df, err := decorator.DecorateFile(fset, f)
	if err != nil {
		t.Fatalf("failed to decorate: %v", err)
	}
	funcDecl := df.Decls[0].(*dst.FuncDecl)
	if len(funcDecl.Body.List) == 0 {
		t.Fatal("no statements in function body")
	}
	return funcDecl.Body.List[0]
}
