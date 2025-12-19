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

func TestIsMatchingStatementPattern(t *testing.T) {
	tests := map[string]struct {
		code string
		want bool
	}{
		"matching defer": {
			code: `defer apm.StartSegment(ctx, "pkg.Func").End()`,
			want: true,
		},
		"different func name": {
			code: `defer apm.StartSegment(ctx, "other.Func").End()`,
			want: true,
		},
		"not a defer": {
			code: `apm.StartSegment(ctx, "pkg.Func").End()`,
			want: false,
		},
		"different method": {
			code: `defer apm.StartSpan(ctx, "pkg.Func").End()`,
			want: false,
		},
		"no End call": {
			code: `defer apm.StartSegment(ctx, "pkg.Func")`,
			want: false,
		},
		"simple defer": {
			code: `defer close()`,
			want: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			stmt := mustParseStmt(t, tt.code)
			got := isMatchingStatementPattern(stmt)
			if got != tt.want {
				t.Errorf("isMatchingStatementPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsMatchingStatement(t *testing.T) {
	tests := map[string]struct {
		code     string
		funcName string
		want     bool
	}{
		"exact match": {
			code:     `defer apm.StartSegment(ctx, "pkg.Func").End()`,
			funcName: "pkg.Func",
			want:     true,
		},
		"different func name": {
			code:     `defer apm.StartSegment(ctx, "other.Func").End()`,
			funcName: "pkg.Func",
			want:     false,
		},
		"method name match": {
			code:     `defer apm.StartSegment(ctx, "pkg.(*Service).Method").End()`,
			funcName: "pkg.(*Service).Method",
			want:     true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			stmt := mustParseStmt(t, tt.code)
			got := isMatchingStatement(stmt, tt.funcName)
			if got != tt.want {
				t.Errorf("isMatchingStatement() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractFuncNameFromDefer(t *testing.T) {
	tests := map[string]struct {
		code string
		want string
	}{
		"simple func": {
			code: `defer apm.StartSegment(ctx, "pkg.Func").End()`,
			want: "pkg.Func",
		},
		"pointer receiver method": {
			code: `defer apm.StartSegment(ctx, "pkg.(*Service).Method").End()`,
			want: "pkg.(*Service).Method",
		},
		"value receiver method": {
			code: `defer apm.StartSegment(ctx, "pkg.Service.Method").End()`,
			want: "pkg.Service.Method",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			stmt := mustParseStmt(t, tt.code)
			def, ok := stmt.(*dst.DeferStmt)
			if !ok {
				t.Fatal("not a defer statement")
			}
			got := extractFuncNameFromDefer(def)
			if got != tt.want {
				t.Errorf("extractFuncNameFromDefer() = %q, want %q", got, tt.want)
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
	if !insertStatement(body, stmt, false) {
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
	if !updateStatement(body, 0, stmt, false) {
		t.Error("updateStatement() returned false")
	}

	// Verify the statement was updated
	def, ok := body.List[0].(*dst.DeferStmt)
	if !ok {
		t.Fatal("first statement is not a defer")
	}

	funcName := extractFuncNameFromDefer(def)
	if funcName != "new.Func" {
		t.Errorf("updated func name = %q, want %q", funcName, "new.Func")
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
