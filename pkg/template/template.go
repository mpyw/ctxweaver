// Package template provides template rendering for ctxweaver.
package template

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"
)

// Vars holds the variables available in templates.
type Vars struct {
	// Ctx is the expression to access context.Context (e.g., "ctx", "c.Request().Context()")
	Ctx string
	// CtxVar is the name of the context parameter variable (e.g., "ctx", "c")
	CtxVar string
	// FuncName is the fully qualified function name (e.g., "(*pkg.Service).Method")
	FuncName string
	// PackageName is the package name (e.g., "service")
	PackageName string
	// PackagePath is the full import path (e.g., "github.com/example/myapp/pkg/service")
	PackagePath string
	// FuncBaseName is the function name without package/receiver (e.g., "Method")
	FuncBaseName string
	// ReceiverType is the receiver type name (empty if not a method)
	ReceiverType string
	// ReceiverVar is the receiver variable name (empty if not a method)
	ReceiverVar string
	// IsMethod indicates whether this is a method
	IsMethod bool
	// IsPointerReceiver indicates whether the receiver is a pointer
	IsPointerReceiver bool
	// IsGenericFunc indicates whether the function has type parameters
	IsGenericFunc bool
	// IsGenericReceiver indicates whether the receiver type has type parameters
	IsGenericReceiver bool
}

// Template wraps a parsed template for statement generation.
type Template struct {
	tmpl *template.Template
	raw  string
}

// funcs returns the template function map.
func funcs() template.FuncMap {
	return template.FuncMap{
		"quote":    strconv.Quote,
		"backtick": func(s string) string { return "`" + s + "`" },
	}
}

// Parse parses a template string.
func Parse(text string) (*Template, error) {
	tmpl, err := template.New("stmt").Funcs(funcs()).Parse(text)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}
	return &Template{tmpl: tmpl, raw: text}, nil
}

// MustParse parses a template string and panics on error.
func MustParse(text string) *Template {
	t, err := Parse(text)
	if err != nil {
		panic(err)
	}
	return t
}

// Render executes the template with the given variables.
func (t *Template) Render(vars Vars) (string, error) {
	var buf bytes.Buffer
	if err := t.tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return strings.TrimSpace(buf.String()), nil
}

// Raw returns the original template string.
func (t *Template) Raw() string {
	return t.raw
}
