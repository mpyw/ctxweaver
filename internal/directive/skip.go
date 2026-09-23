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

// isSkipComment reports whether a comment is the skip directive.
//
// Only Go's canonical directive form counts: "//ctxweaver:skip", with no space
// after "//" and none after the colon. Trailing arguments are allowed. A
// spelling that is close but not canonical is reported by [IsMalformed] and
// has no effect here.
func isSkipComment(text string) bool {
	d, ok := ast.ParseDirective(token.NoPos, text)
	return ok && d.Tool == directiveTool && d.Name == skipName
}

// IsMalformed reports whether a comment looks like a ctxweaver directive but
// is not in the canonical "//ctxweaver:name" form.
//
// It matches a comment whose body starts with "ctxweaver:" after optional
// whitespace, following "//" or "/*". So a space or tab after "//", a space
// after the colon, and block comments are malformed. Prose that mentions
// "ctxweaver:skip" partway through a sentence is not.
func IsMalformed(text string) bool {
	body, ok := strings.CutPrefix(text, "//")
	if !ok {
		if body, ok = strings.CutPrefix(text, "/*"); !ok {
			return false
		}
		body = strings.TrimSuffix(body, "*/")
	}
	if !strings.HasPrefix(strings.TrimLeft(body, " \t"), directiveTool+":") {
		return false
	}
	d, ok := ast.ParseDirective(token.NoPos, text)
	return !ok || d.Tool != directiveTool
}

// MalformedMessage is the warning for a comment that [IsMalformed] matches.
const MalformedMessage = "malformed ctxweaver directive: write //ctxweaver:skip"

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
