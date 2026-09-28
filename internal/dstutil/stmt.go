package dstutil

import (
	"go/parser"
	"go/token"
	"slices"

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
	body.List = slices.Concat(body.List[:index], stmts, body.List[index+count:])

	return true
}

// RemoveStatements removes `count` statements starting at the given index.
func RemoveStatements(body *dst.BlockStmt, index, count int) bool {
	if index < 0 || index >= len(body.List) || count <= 0 || index+count > len(body.List) {
		return false
	}

	body.List = slices.Delete(body.List, index, index+count)
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

	return funcDecl.Body.List, nil
}
