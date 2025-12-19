package processor

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strconv"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

// GeneratedMarker is the marker comment used to identify ctxweaver-generated statements.
// This enables idempotency: running ctxweaver multiple times will update existing
// generated statements rather than inserting duplicates.
const GeneratedMarker = "//ctxweaver:generated"

// hasGeneratedMarker checks if a statement has the ctxweaver:generated marker.
func hasGeneratedMarker(stmt dst.Stmt) bool {
	for _, comment := range stmt.Decorations().End.All() {
		if strings.Contains(comment, "ctxweaver:generated") {
			return true
		}
	}
	// Also check start decorations for backward compatibility
	for _, comment := range stmt.Decorations().Start.All() {
		if strings.Contains(comment, "ctxweaver:generated") {
			return true
		}
	}
	return false
}

// isMatchingStatement checks if a statement matches our pattern with the expected function name.
// This is used to detect if the statement is already up-to-date.
func isMatchingStatement(stmt dst.Stmt, expectedFuncName string) bool {
	// Currently we detect the pattern: defer XXX.StartSegment(ctx, "funcName").End()
	// This is specific to the APM use case but can be generalized later

	def, ok := stmt.(*dst.DeferStmt)
	if !ok {
		return false
	}

	funcName := extractFuncNameFromDefer(def)
	return funcName == expectedFuncName
}

// isMatchingStatementPattern checks if a statement matches our general pattern.
// Returns true if it's a defer statement with a similar structure, regardless of function name.
func isMatchingStatementPattern(stmt dst.Stmt) bool {
	def, ok := stmt.(*dst.DeferStmt)
	if !ok {
		return false
	}

	// Check if it's a defer with .End() call
	if def.Call == nil {
		return false
	}

	sel, ok := def.Call.Fun.(*dst.SelectorExpr)
	if !ok {
		return false
	}

	// Check for .End() pattern
	if sel.Sel == nil || sel.Sel.Name != "End" {
		return false
	}

	// Check if the X is a call expression (StartSegment call)
	call, ok := sel.X.(*dst.CallExpr)
	if !ok {
		return false
	}

	// Check for StartSegment pattern
	innerSel, ok := call.Fun.(*dst.SelectorExpr)
	if !ok {
		return false
	}

	if innerSel.Sel == nil || innerSel.Sel.Name != "StartSegment" {
		return false
	}

	return true
}

// extractFuncNameFromDefer extracts the function name from a defer statement.
// Assumes the pattern: defer XXX.StartSegment(ctx, "funcName").End()
func extractFuncNameFromDefer(def *dst.DeferStmt) string {
	if def.Call == nil {
		return ""
	}

	sel, ok := def.Call.Fun.(*dst.SelectorExpr)
	if !ok {
		return ""
	}

	call, ok := sel.X.(*dst.CallExpr)
	if !ok {
		return ""
	}

	if len(call.Args) < 2 {
		return ""
	}

	lit, ok := call.Args[1].(*dst.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}

	// Unquote the string
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return lit.Value
	}
	return s
}

// insertStatement inserts a statement at the beginning of a function body.
// If useMarker is true, the statement is marked with GeneratedMarker for idempotency.
func insertStatement(body *dst.BlockStmt, stmtStr string, useMarker bool) bool {
	stmt, err := parseStatement(stmtStr)
	if err != nil {
		return false
	}

	// Optionally add the generated marker as an end-of-line comment
	if useMarker {
		stmt.Decorations().End.Append(GeneratedMarker)
	}

	// Add empty line after the statement
	stmt.Decorations().After = dst.EmptyLine

	body.List = append([]dst.Stmt{stmt}, body.List...)
	return true
}

// updateStatement updates a statement at the given index.
// If useMarker is true, the statement is marked with GeneratedMarker for idempotency.
func updateStatement(body *dst.BlockStmt, index int, stmtStr string, useMarker bool) bool {
	if index < 0 || index >= len(body.List) {
		return false
	}

	stmt, err := parseStatement(stmtStr)
	if err != nil {
		return false
	}

	// Optionally add the generated marker as an end-of-line comment
	if useMarker {
		stmt.Decorations().End.Append(GeneratedMarker)
	}

	// Preserve Before and After decorations from the old statement
	oldStmt := body.List[index]
	stmt.Decorations().Before = oldStmt.Decorations().Before
	stmt.Decorations().After = oldStmt.Decorations().After

	body.List[index] = stmt
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

	// Handle multi-line statements (multiple statements)
	if len(funcDecl.Body.List) == 1 {
		return funcDecl.Body.List[0], nil
	}

	// For multi-line templates, we need to handle multiple statements
	// Return a block statement isn't valid here, so we just take the first
	// TODO: Support multi-statement insertion
	return funcDecl.Body.List[0], nil
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

// normalizeStatement normalizes whitespace in a statement string.
func normalizeStatement(s string) string {
	return strings.TrimSpace(s)
}
