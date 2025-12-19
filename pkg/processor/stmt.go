package processor

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

// insertStatement inserts a statement at the beginning of a function body.
func insertStatement(body *dst.BlockStmt, stmtStr string) bool {
	stmt, err := parseStatement(stmtStr)
	if err != nil {
		return false
	}

	// Add empty line after the statement
	stmt.Decorations().After = dst.EmptyLine

	body.List = append([]dst.Stmt{stmt}, body.List...)
	return true
}

// updateStatement updates a statement at the given index.
func updateStatement(body *dst.BlockStmt, index int, stmtStr string) bool {
	if index < 0 || index >= len(body.List) {
		return false
	}

	stmt, err := parseStatement(stmtStr)
	if err != nil {
		return false
	}

	// Preserve Before and After decorations from the old statement
	oldStmt := body.List[index]
	stmt.Decorations().Before = oldStmt.Decorations().Before
	stmt.Decorations().After = oldStmt.Decorations().After

	body.List[index] = stmt
	return true
}

// removeStatement removes a statement at the given index.
func removeStatement(body *dst.BlockStmt, index int) bool {
	if index < 0 || index >= len(body.List) {
		return false
	}

	body.List = append(body.List[:index], body.List[index+1:]...)
	return true
}

// parseStatement parses a statement string into a DST statement.
func parseStatement(stmtStr string) (dst.Stmt, error) {
	// Wrap in a function to parse as a statement
	src := "package p\nfunc f() {\n" + stmtStr + "\n}"

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	df, err := decorator.DecorateFile(fset, f)
	if err != nil {
		return nil, err
	}

	// Extract the statement from the function body
	funcDecl := df.Decls[0].(*dst.FuncDecl)
	if len(funcDecl.Body.List) == 0 {
		return nil, nil
	}

	// Get the last statement (or only statement)
	lastIdx := len(funcDecl.Body.List) - 1
	stmt := funcDecl.Body.List[lastIdx]

	// Trailing comments after the last statement end up in the function body's
	// End decoration (before the closing brace). We need to capture these and
	// attach them to the statement so they're preserved during insertion/update.
	if endComments := funcDecl.Body.Decs.End; len(endComments) > 0 {
		for _, c := range endComments {
			stmt.Decorations().End.Append(c)
		}
	}

	// For multi-line templates with multiple statements, we only return the first.
	// The trailing comments have already been attached to the last statement above,
	// but if there are multiple statements, we still only support the first.
	// TODO: Support multi-statement insertion
	if len(funcDecl.Body.List) > 1 {
		return funcDecl.Body.List[0], nil
	}

	return stmt, nil
}

// stmtToString converts a DST statement back to a string.
// This is used for exact comparison in skeleton matching.
func stmtToString(stmt dst.Stmt) string {
	// Create a minimal file containing just this statement
	df := &dst.File{
		Name: dst.NewIdent("p"),
		Decls: []dst.Decl{
			&dst.FuncDecl{
				Name: dst.NewIdent("f"),
				Type: &dst.FuncType{},
				Body: &dst.BlockStmt{
					List: []dst.Stmt{stmt},
				},
			},
		},
	}

	// Restore to AST
	fset, f, err := decorator.RestoreFile(df)
	if err != nil {
		return ""
	}

	// Extract just the statement part
	funcDecl := f.Decls[0].(*ast.FuncDecl)
	if len(funcDecl.Body.List) == 0 {
		return ""
	}

	var buf strings.Builder
	if err := format.Node(&buf, fset, funcDecl.Body.List[0]); err != nil {
		return ""
	}

	return strings.TrimSpace(buf.String())
}
