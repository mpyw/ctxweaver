package processor

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"github.com/dave/dst/decorator"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/imports"
)

// Process processes the given package patterns.
func (p *Processor) Process(patterns []string) (*ProcessResult, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedImports,
		Tests: p.test,
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %w", err)
	}

	result := &ProcessResult{}

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			for _, e := range pkg.Errors {
				result.Errors = append(result.Errors, fmt.Errorf("package %s: %v", pkg.PkgPath, e))
			}
			continue
		}

		for _, file := range pkg.Syntax {
			// Get filename from AST position (more reliable than index-based access)
			pos := pkg.Fset.Position(file.Pos())
			if !pos.IsValid() {
				continue
			}
			filename := pos.Filename

			if !p.shouldProcessFile(filename) {
				continue
			}

			result.FilesProcessed++

			modified, err := p.processFile(pkg, file, filename)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("%s: %w", filename, err))
				continue
			}

			if modified {
				result.FilesModified++
				if p.verbose {
					fmt.Printf("modified: %s\n", filename)
				}
			}
		}
	}

	return result, nil
}

func (p *Processor) shouldProcessFile(filename string) bool {
	// Skip test files if not enabled
	if !p.test && strings.HasSuffix(filename, "_test.go") {
		return false
	}
	// Skip testdata directories (convention for test fixtures)
	if strings.Contains(filename, "/testdata/") || strings.Contains(filename, "\\testdata\\") {
		return false
	}
	return true
}

func (p *Processor) processFile(pkg *packages.Package, astFile *ast.File, filename string) (bool, error) {
	// Read original source
	src, err := os.ReadFile(filename)
	if err != nil {
		return false, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse with fresh fset for DST conversion
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return false, fmt.Errorf("failed to parse file: %w", err)
	}

	// Skip generated files (files with "// Code generated" comment)
	if ast.IsGenerated(f) {
		return false, nil
	}

	// Convert to DST
	df, err := decorator.DecorateFile(fset, f)
	if err != nil {
		return false, fmt.Errorf("failed to decorate file: %w", err)
	}

	// Check for file-level skip directive
	if hasSkipDirective(df.Decorations()) {
		return false, nil
	}

	// Process functions
	modified, err := p.processFunctions(df, pkg)
	if err != nil {
		return false, err
	}
	if !modified {
		return false, nil
	}

	// Convert back to AST
	fset, f, err = decorator.RestoreFile(df)
	if err != nil {
		return false, fmt.Errorf("failed to restore file: %w", err)
	}

	// Add imports
	for _, imp := range p.imports {
		astutil.AddImport(fset, f, imp)
	}

	// Format
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		return false, fmt.Errorf("failed to format file: %w", err)
	}

	// Clean up unused imports using goimports
	// This handles the case where template changes make old imports unused
	result, err := imports.Process(filename, buf.Bytes(), &imports.Options{
		Comments:   true,
		TabIndent:  true,
		TabWidth:   8,
		FormatOnly: false, // Run full goimports (add missing + remove unused)
	})
	if err != nil {
		// If goimports fails, use the formatted output without cleanup
		result = buf.Bytes()
	}

	// Write if not dry run
	if !p.dryRun {
		if err := os.WriteFile(filename, result, 0o644); err != nil {
			return false, fmt.Errorf("failed to write file: %w", err)
		}
	}

	return true, nil
}
