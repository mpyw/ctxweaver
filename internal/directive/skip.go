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

// MalformedMessage is the warning for a comment that [Malformed] matches.
const MalformedMessage = "malformed ctxweaver directive: write it as //ctxweaver:name"

// Malformed reports whether a comment is addressed to ctxweaver but is not a
// valid directive.
//
// A comment is addressed to ctxweaver when its body, after "//" or "/*",
// starts with "ctxweaver:" once optional whitespace is skipped. It is valid
// when [ast.ParseDirective] accepts it with the tool "ctxweaver". Prose that
// mentions "ctxweaver:skip" partway through a sentence is not addressed.
func Malformed(text string) bool {
	body, ok := strings.CutPrefix(text, "//")
	if !ok {
		if body, ok = strings.CutPrefix(text, "/*"); !ok {
			return false
		}
	}
	if !strings.HasPrefix(strings.TrimLeft(body, " \t"), directiveTool+":") {
		return false
	}
	d, ok := ast.ParseDirective(token.NoPos, text)
	return !ok || d.Tool != directiveTool
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
