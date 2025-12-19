package processor

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

func TestParseStatement(t *testing.T) {
	tests := map[string]struct {
		input   string
		wantErr bool
	}{
		"defer statement": {
			input: `defer apm.StartSegment(ctx, "pkg.Func").End()`,
		},
		"assignment statement": {
			input: `ctx, span := tracer.Start(ctx, "func")`,
		},
		"expression statement": {
			input: `log.Info().Msg("hello")`,
		},
		"invalid statement": {
			input:   `if {`,
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			stmt, err := parseStatement(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseStatement() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if stmt == nil {
				t.Error("parseStatement() returned nil statement")
			}
		})
	}
}

func TestInsertStatement(t *testing.T) {
	body := &dst.BlockStmt{
		List: []dst.Stmt{
			&dst.ExprStmt{X: &dst.Ident{Name: "existing"}},
		},
	}

	stmt := `defer trace(ctx)`
	if !insertStatement(body, stmt) {
		t.Error("insertStatement() returned false")
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
}

func TestUpdateStatement(t *testing.T) {
	body := &dst.BlockStmt{
		List: []dst.Stmt{
			mustParseStmt(t, `defer apm.StartSegment(ctx, "old.Func").End()`),
			&dst.ExprStmt{X: &dst.Ident{Name: "other"}},
		},
	}

	stmt := `defer apm.StartSegment(ctx, "new.Func").End()`
	if !updateStatement(body, 0, stmt) {
		t.Error("updateStatement() returned false")
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
}

func TestRemoveStatement(t *testing.T) {
	tests := map[string]struct {
		initialLen int
		removeIdx  int
		wantLen    int
		wantResult bool
	}{
		"remove first": {
			initialLen: 3,
			removeIdx:  0,
			wantLen:    2,
			wantResult: true,
		},
		"remove middle": {
			initialLen: 3,
			removeIdx:  1,
			wantLen:    2,
			wantResult: true,
		},
		"remove last": {
			initialLen: 3,
			removeIdx:  2,
			wantLen:    2,
			wantResult: true,
		},
		"invalid index negative": {
			initialLen: 3,
			removeIdx:  -1,
			wantLen:    3,
			wantResult: false,
		},
		"invalid index too large": {
			initialLen: 3,
			removeIdx:  3,
			wantLen:    3,
			wantResult: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create body with n statements
			body := &dst.BlockStmt{List: make([]dst.Stmt, tt.initialLen)}
			for i := range body.List {
				body.List[i] = &dst.ExprStmt{X: &dst.Ident{Name: "stmt"}}
			}

			got := removeStatement(body, tt.removeIdx)
			if got != tt.wantResult {
				t.Errorf("removeStatement() = %v, want %v", got, tt.wantResult)
			}
			if len(body.List) != tt.wantLen {
				t.Errorf("body.List length = %d, want %d", len(body.List), tt.wantLen)
			}
		})
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
