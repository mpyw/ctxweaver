package dstutil

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

// InsertStatements inserts statements at the beginning of a function body.
func InsertStatements(body *dst.BlockStmt, stmtStr string) bool {
	stmts, err := ParseStatements(stmtStr)
	if err != nil || len(stmts) == 0 {
		return false
	}

	// Add empty line after the last inserted statement
	stmts[len(stmts)-1].Decorations().After = dst.EmptyLine

	body.List = append(stmts, body.List...)
	return true
}

// UpdateStatements updates statements starting at the given index.
// It replaces `count` statements with the parsed statements from stmtStr.
func UpdateStatements(body *dst.BlockStmt, index, count int, stmtStr string) bool {
	if index < 0 || index >= len(body.List) || count <= 0 || index+count > len(body.List) {
		return false
	}

	stmts, err := ParseStatements(stmtStr)
	if err != nil || len(stmts) == 0 {
		return false
	}

	// Preserve Before decoration from the first old statement
	stmts[0].Decorations().Before = body.List[index].Decorations().Before
	// Preserve After decoration from the last old statement
	stmts[len(stmts)-1].Decorations().After = body.List[index+count-1].Decorations().After

	// Replace: body.List[:index] + stmts + body.List[index+count:]
	newList := make([]dst.Stmt, 0, len(body.List)-count+len(stmts))
	newList = append(newList, body.List[:index]...)
	newList = append(newList, stmts...)
	newList = append(newList, body.List[index+count:]...)
	body.List = newList

	return true
}

// RemoveStatements removes `count` statements starting at the given index.
func RemoveStatements(body *dst.BlockStmt, index, count int) bool {
	if index < 0 || index >= len(body.List) || count <= 0 || index+count > len(body.List) {
		return false
	}

	body.List = append(body.List[:index], body.List[index+count:]...)
	return true
}

// ParseStatements parses a statement string into DST statements.
// Supports multiple statements separated by newlines.
func ParseStatements(stmtStr string) ([]dst.Stmt, error) {
	// Wrap in a function to parse as statements
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

	// Extract the statements from the function body
	funcDecl := df.Decls[0].(*dst.FuncDecl)
	if len(funcDecl.Body.List) == 0 {
		return nil, nil
	}

	stmts := funcDecl.Body.List

	// Trailing comments after the last statement end up in the function body's
	// End decoration (before the closing brace). We need to capture these and
	// attach them to the last statement so they're preserved during insertion/update.
	if endComments := funcDecl.Body.Decs.End; len(endComments) > 0 {
		lastStmt := stmts[len(stmts)-1]
		for _, c := range endComments {
			lastStmt.Decorations().End.Append(c)
		}
	}

	return stmts, nil
}

// StmtsToStrings converts DST statements to their string representations.
func StmtsToStrings(stmts []dst.Stmt) []string {
	result := make([]string, len(stmts))
	for i, stmt := range stmts {
		result[i] = StmtToString(stmt)
	}
	return result
}

// StmtToString converts a DST statement back to a string.
// This is used for exact comparison in skeleton matching.
func StmtToString(stmt dst.Stmt) string {
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
