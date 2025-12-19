package processor

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"

	"github.com/dave/dst/decorator"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/imports"
)

// TransformSource transforms a single Go source file.
// This is useful for testing without requiring packages.Load.
func (p *Processor) TransformSource(src []byte, pkgName string) ([]byte, error) {
	// Parse with fresh fset for DST conversion
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source: %w", err)
	}

	// Skip generated files (files with "// Code generated" comment)
	if ast.IsGenerated(f) {
		return src, nil
	}

	// Convert to DST
	df, err := decorator.DecorateFile(fset, f)
	if err != nil {
		return nil, fmt.Errorf("failed to decorate file: %w", err)
	}

	// Check for file-level skip directive
	if hasSkipDirective(df.Decorations()) {
		return src, nil
	}

	// Process functions
	modified, err := p.processFunctionsForSource(df, pkgName)
	if err != nil {
		return nil, err
	}
	if !modified {
		return src, nil
	}

	// Convert back to AST
	fset, f, err = decorator.RestoreFile(df)
	if err != nil {
		return nil, fmt.Errorf("failed to restore file: %w", err)
	}

	// Add imports
	for _, imp := range p.imports {
		astutil.AddImport(fset, f, imp)
	}

	// Format
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		return nil, fmt.Errorf("failed to format file: %w", err)
	}

	// Clean up unused imports using goimports
	// This handles the case where template changes make old imports unused
	result, err := imports.Process("test.go", buf.Bytes(), &imports.Options{
		Comments:   true,
		TabIndent:  true,
		TabWidth:   8,
		FormatOnly: false, // Run full goimports (add missing + remove unused)
	})
	if err != nil {
		// If goimports fails, return the formatted output without cleanup
		return buf.Bytes(), nil
	}

	return result, nil
}
