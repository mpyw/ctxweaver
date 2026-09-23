package processor

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"os"
	"strings"

	"github.com/dave/dst/decorator"
	"github.com/dave/dst/decorator/resolver/guess"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/imports"

	"github.com/mpyw/ctxweaver/internal/directive"
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
	warned := make(map[string]bool)

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			for _, e := range pkg.Errors {
				result.Errors = append(result.Errors, fmt.Errorf("package %s: %v", pkg.PkgPath, e))
			}
			continue
		}

		// Check if package should be excluded by regex patterns
		if !p.shouldProcessPackage(pkg.PkgPath) {
			if p.verbose {
				fmt.Printf("excluded: %s\n", pkg.PkgPath)
			}
			continue
		}

		// Create decorator once per package for efficient type-resolved DST conversion
		dec := decorator.NewDecoratorFromPackage(pkg)

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

			// A file can belong to more than one package when tests are
			// loaded, so each warning is kept once.
			for _, w := range processDirectiveWarnings(pkg.Fset, file) {
				if !warned[w] {
					warned[w] = true
					result.Warnings = append(result.Warnings, w)
				}
			}

			modified, err := p.processFile(pkg, dec, file, filename)
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

// shouldProcessPackage checks if the package path passes the regex filters.
func (p *Processor) shouldProcessPackage(pkgPath string) bool {
	return p.pkgRegexps.Match(pkgPath)
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

// processDirectiveWarnings returns a warning for each malformed ctxweaver
// directive in the file, prefixed with its file:line position. A malformed
// directive has no effect. Generated files are not processed, so they are not
// checked.
func processDirectiveWarnings(fset *token.FileSet, file *ast.File) []string {
	if ast.IsGenerated(file) {
		return nil
	}
	var warnings []string
	for _, group := range file.Comments {
		for _, c := range group.List {
			if message, ok := directive.Malformed(c.Text); ok {
				pos := fset.Position(c.Slash)
				warnings = append(warnings, fmt.Sprintf("%s:%d: %s", pos.Filename, pos.Line, message))
			}
		}
	}
	return warnings
}

func (p *Processor) processFile(pkg *packages.Package, dec *decorator.Decorator, astFile *ast.File, filename string) (bool, error) {
	// Skip generated files (files with "// Code generated" comment)
	if ast.IsGenerated(astFile) {
		return false, nil
	}

	// Convert to DST using type-resolved decorator (sets dst.Ident.Path automatically)
	df, err := dec.DecorateFile(astFile)
	if err != nil {
		return false, fmt.Errorf("failed to decorate file: %w", err)
	}

	// Check for file-level skip directive
	if directive.HasSkipDirective(df.Decorations()) {
		return false, nil
	}

	// Process functions
	modified, err := p.processCandidates(df, pkg.PkgPath)
	if err != nil {
		return false, err
	}
	if !modified {
		return false, nil
	}

	// Convert back to AST using package import info. Resolving from
	// packages.Package.Imports avoids additional packages.Load calls while
	// providing accurate package names.
	resolver := make(map[string]string, len(pkg.Imports))
	for path, imported := range pkg.Imports {
		resolver[path] = imported.Name
	}
	restorer := decorator.NewRestorerWithImports(pkg.PkgPath, guess.WithMap(resolver))
	f, err := restorer.RestoreFile(df)
	if err != nil {
		return false, fmt.Errorf("failed to restore file: %w", err)
	}
	fset := restorer.Fset

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
