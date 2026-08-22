// Package directive provides utilities for processing ctxweaver directives in comments.
package directive

import (
	"go/ast"
	"go/token"
	"slices"
	"strings"

	"github.com/dave/dst"
)

const (
	directiveTool = "ctxweaver"
	skipName      = "skip"
)

// isSkipComment checks if a comment text is a skip directive.
// Supports both "//ctxweaver:skip" and "// ctxweaver:skip".
func isSkipComment(text string) bool {
	// go/ast only recognises the canonical, space-free form, so re-attach the
	// comment marker to the trimmed body before handing it over.
	if body, ok := strings.CutPrefix(text, "//"); ok {
		text = "//" + strings.TrimSpace(body)
	}
	d, ok := ast.ParseDirective(token.NoPos, text)
	return ok && d.Tool == directiveTool && d.Name == skipName
}

// HasSkipDirective checks if node decorations contain a skip directive.
// This is used for file-level and function-level skip directives.
func HasSkipDirective(decs *dst.NodeDecs) bool {
	return slices.ContainsFunc(decs.Start.All(), isSkipComment)
}

// HasStmtSkipDirective checks if a statement has a skip directive comment.
// Checks both Start (before) and End (trailing) decorations.
func HasStmtSkipDirective(stmt dst.Stmt) bool {
	decs := stmt.Decorations()
	return slices.ContainsFunc(decs.Start.All(), isSkipComment) ||
		slices.ContainsFunc(decs.End.All(), isSkipComment)
}
