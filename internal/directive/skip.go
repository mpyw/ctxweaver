// Package directive provides utilities for processing ctxweaver directives in comments.
package directive

import (
	"strings"

	"github.com/dave/dst"
)

const skipDirective = "ctxweaver:skip"

// isSkipComment checks if a comment text is a skip directive.
// Supports both "//ctxweaver:skip" and "// ctxweaver:skip".
func isSkipComment(text string) bool {
	text = strings.TrimPrefix(text, "//")
	text = strings.TrimSpace(text)
	return strings.HasPrefix(text, skipDirective)
}

// HasSkipDirective checks if node decorations contain a skip directive.
// This is used for file-level and function-level skip directives.
func HasSkipDirective(decs *dst.NodeDecs) bool {
	for _, c := range decs.Start.All() {
		if isSkipComment(c) {
			return true
		}
	}
	return false
}

// HasStmtSkipDirective checks if a statement has a skip directive comment.
// Checks both Start (before) and End (trailing) decorations.
func HasStmtSkipDirective(stmt dst.Stmt) bool {
	decs := stmt.Decorations()
	for _, c := range decs.Start.All() {
		if isSkipComment(c) {
			return true
		}
	}
	for _, c := range decs.End.All() {
		if isSkipComment(c) {
			return true
		}
	}
	return false
}
