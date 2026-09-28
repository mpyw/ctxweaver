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

// Warning messages for a comment that [Problem] reports.
const (
	MalformedMessage = "malformed ctxweaver directive: write it as //ctxweaver:name"
	UnknownMessage   = "unknown ctxweaver directive: ctxweaver:"
	SkipArgMessage   = "ctxweaver:skip takes no argument; write a reason after //"
	HiddenMessage    = "ctxweaver directive after another comment: write it as its own //ctxweaver:name comment"
)

// read returns the name of the directive a comment holds, and the warning for
// a comment addressed to ctxweaver that ctxweaver does not read. A comment
// that is not addressed to ctxweaver returns two empty strings.
//
// A trailing comment is a reason. The text is cut at the first "//" after the
// leading one, so "//ctxweaver:skip // reason" and "//ctxweaver:skip//reason"
// are both a skip. The part before the cut is addressed to ctxweaver when it
// starts with "ctxweaver:" once any space is skipped. It is a directive when
// [ast.ParseDirective] reads it as a line comment with the tool "ctxweaver".
// Prose that mentions ctxweaver:skip partway through is not addressed. A part
// after a cut that is addressed to ctxweaver, as in
// "//nolint:foo //ctxweaver:skip", is reported: it looks like a directive, but
// is not one.
func read(text string) (name, problem string) {
	body, line := strings.CutPrefix(text, "//")
	if !line {
		body = strings.TrimSuffix(strings.TrimPrefix(text, "/*"), "*/")
	}
	parts := strings.Split(body, "//")
	if !addressed(parts[0]) {
		if slices.ContainsFunc(parts[1:], addressed) {
			return "", HiddenMessage
		}
		return "", ""
	}
	d, ok := ast.ParseDirective(token.NoPos, "//"+parts[0])
	switch {
	case !line || !ok || d.Tool != directiveTool:
		return "", MalformedMessage
	case d.Name != skipName:
		return d.Name, UnknownMessage + d.Name
	case d.Args != "" && d.Args != "-" && !strings.HasPrefix(d.Args, "- "):
		// " - reason" is accepted for compatibility.
		return d.Name, SkipArgMessage
	}
	return d.Name, ""
}

func addressed(s string) bool {
	return strings.HasPrefix(strings.TrimSpace(s), directiveTool+":")
}

// isSkipComment reports whether a comment is the skip directive:
// "//ctxweaver:skip", optionally followed by a reason after "//" or " - ".
func isSkipComment(text string) bool {
	name, problem := read(text)
	return name == skipName && problem == ""
}

// Problem returns the warning for a comment addressed to ctxweaver that is
// not a directive ctxweaver reads, or "" for any other comment. Such a comment
// has no effect, and the caller must not rewrite the file holding it.
func Problem(text string) string {
	_, problem := read(text)
	return problem
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
