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

// Malformed reports whether a comment is addressed to ctxweaver but is not a
// valid directive, and returns the warning for it.
//
// A comment is addressed to ctxweaver when its body, after "//" or "/*",
// starts with "ctxweaver:" once optional whitespace is skipped. It is a
// directive when [ast.ParseDirective] accepts it with the tool "ctxweaver". So
// a space or tab after "//", a space after the colon, a block comment, a name
// that is missing or does not start with a lowercase letter or digit are all
// malformed. A lookalike name such as "skipx" is valid syntax and not
// malformed. Prose that mentions "ctxweaver:skip" partway through a sentence
// is not addressed.
//
// The warning suggests "//ctxweaver:<name>" only when that rewritten text is
// itself a directive. The name is never changed or guessed.
func Malformed(text string) (string, bool) {
	body, ok := strings.CutPrefix(text, "//")
	if !ok {
		if body, ok = strings.CutPrefix(text, "/*"); !ok {
			return "", false
		}
		body = strings.TrimSuffix(body, "*/")
	}
	rest, ok := strings.CutPrefix(strings.TrimLeft(body, " \t"), directiveTool+":")
	if !ok {
		return "", false
	}
	if d, ok := ast.ParseDirective(token.NoPos, text); ok && d.Tool == directiveTool {
		return "", false
	}

	const message = "malformed " + directiveTool + " directive"
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return message, true
	}
	suggestion := "//" + directiveTool + ":" + fields[0]
	if d, ok := ast.ParseDirective(token.NoPos, suggestion); !ok || d.Tool != directiveTool {
		return message, true
	}
	return message + ": write " + suggestion, true
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
