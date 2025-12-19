// Package processor provides DST-based code transformation.
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

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"

	"github.com/mpyw/ctxweaver/pkg/config"
	"github.com/mpyw/ctxweaver/pkg/template"
)

// Processor handles code transformation.
type Processor struct {
	registry *config.CarrierRegistry
	tmpl     *template.Template
	imports  []string
	test     bool
	dryRun   bool
	verbose  bool
}

// Option configures a Processor.
type Option func(*Processor)

// WithTest enables processing of test files.
func WithTest(test bool) Option {
	return func(p *Processor) {
		p.test = test
	}
}

// WithDryRun enables dry run mode (no file writes).
func WithDryRun(dryRun bool) Option {
	return func(p *Processor) {
		p.dryRun = dryRun
	}
}

// WithVerbose enables verbose output.
func WithVerbose(verbose bool) Option {
	return func(p *Processor) {
		p.verbose = verbose
	}
}

// New creates a new Processor.
func New(registry *config.CarrierRegistry, tmpl *template.Template, imports []string, opts ...Option) *Processor {
	p := &Processor{
		registry: registry,
		tmpl:     tmpl,
		imports:  imports,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// ProcessResult holds the result of processing.
type ProcessResult struct {
	FilesProcessed int
	FilesModified  int
	Errors         []error
}

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

		for i, file := range pkg.Syntax {
			filename := pkg.CompiledGoFiles[i]

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
	modified := p.processFunctions(df, pkg)
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

	// Write if not dry run
	if !p.dryRun {
		if err := os.WriteFile(filename, buf.Bytes(), 0o644); err != nil {
			return false, fmt.Errorf("failed to write file: %w", err)
		}
	}

	return true, nil
}

func (p *Processor) processFunctions(df *dst.File, pkg *packages.Package) bool {
	modified := false
	aliases := resolveAliases(df.Imports)

	dst.Inspect(df, func(n dst.Node) bool {
		decl, ok := n.(*dst.FuncDecl)
		if !ok {
			return true
		}

		// Skip if function has skip directive
		if hasSkipDirective(decl.Decorations()) {
			return true
		}

		// Skip if no body
		if decl.Body == nil {
			return true
		}

		// Get first parameter
		param := extractFirstParam(decl)
		if param == nil {
			return true
		}

		// Check if first param is a context carrier
		carrier, varName, ok := p.matchCarrier(param, aliases)
		if !ok {
			return true
		}

		// Build template variables
		vars := p.buildVars(df, decl, pkg, carrier, varName)

		// Render statement
		stmt, err := p.tmpl.Render(vars)
		if err != nil {
			// Skip on render error
			return true
		}

		// Check existing statement and update if needed
		action := p.detectAction(decl.Body, vars.FuncName)

		switch action.Type {
		case ActionInsert:
			if insertStatement(decl.Body, stmt) {
				modified = true
			}
		case ActionUpdate:
			if updateStatement(decl.Body, action.Index, stmt) {
				modified = true
			}
		case ActionSkip:
			// Already up to date
		}

		return true
	})

	return modified
}

func (p *Processor) matchCarrier(param *dst.Field, aliases map[string]string) (config.CarrierDef, string, bool) {
	if len(param.Names) == 0 || param.Names[0].Name == "_" {
		return config.CarrierDef{}, "", false
	}

	varName := param.Names[0].Name

	// Handle pointer types
	typ := param.Type
	if star, ok := typ.(*dst.StarExpr); ok {
		typ = star.X
	}

	// Must be a selector expression (pkg.Type)
	sel, ok := typ.(*dst.SelectorExpr)
	if !ok {
		return config.CarrierDef{}, "", false
	}

	pkgIdent, ok := sel.X.(*dst.Ident)
	if !ok {
		return config.CarrierDef{}, "", false
	}

	pkgPath := aliases[pkgIdent.Name]
	typeName := sel.Sel.Name

	carrier, found := p.registry.Lookup(pkgPath, typeName)
	if !found {
		return config.CarrierDef{}, "", false
	}

	return carrier, varName, true
}

func (p *Processor) buildVars(df *dst.File, decl *dst.FuncDecl, pkg *packages.Package, carrier config.CarrierDef, varName string) template.Vars {
	vars := template.Vars{
		Ctx:          carrier.BuildContextExpr(varName),
		CtxVar:       varName,
		PackageName:  df.Name.Name,
		PackagePath:  pkg.PkgPath,
		FuncBaseName: decl.Name.Name,
	}

	// Build fully qualified function name
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		vars.IsMethod = true
		recv := decl.Recv.List[0]

		if len(recv.Names) > 0 {
			vars.ReceiverVar = recv.Names[0].Name
		}

		switch typ := recv.Type.(type) {
		case *dst.StarExpr:
			vars.IsPointerReceiver = true
			if ident, ok := typ.X.(*dst.Ident); ok {
				vars.ReceiverType = ident.Name
				vars.FuncName = fmt.Sprintf("%s.(*%s).%s", vars.PackageName, ident.Name, decl.Name.Name)
			}
		case *dst.Ident:
			vars.ReceiverType = typ.Name
			vars.FuncName = fmt.Sprintf("%s.%s.%s", vars.PackageName, typ.Name, decl.Name.Name)
		}
	} else {
		vars.FuncName = fmt.Sprintf("%s.%s", vars.PackageName, decl.Name.Name)
	}

	return vars
}

// ActionType represents the action to take on a function.
type ActionType int

const (
	ActionSkip ActionType = iota
	ActionInsert
	ActionUpdate
)

// Action represents the action to take and related info.
type Action struct {
	Type  ActionType
	Index int // For ActionUpdate, the index of the statement to update
}

func (p *Processor) detectAction(body *dst.BlockStmt, expectedFuncName string) Action {
	// Look for existing defer statement that matches our pattern
	for i, stmt := range body.List {
		if isMatchingStatement(stmt, expectedFuncName) {
			return Action{Type: ActionSkip, Index: i}
		}
		if isMatchingStatementPattern(stmt) {
			// Pattern matches but func name differs - needs update
			return Action{Type: ActionUpdate, Index: i}
		}
	}
	return Action{Type: ActionInsert}
}

func extractFirstParam(decl *dst.FuncDecl) *dst.Field {
	if decl.Type == nil || decl.Type.Params == nil || len(decl.Type.Params.List) == 0 {
		return nil
	}
	return decl.Type.Params.List[0]
}

func resolveAliases(imports []*dst.ImportSpec) map[string]string {
	result := make(map[string]string)
	for _, imp := range imports {
		path := strings.Trim(imp.Path.Value, `"`)
		var local string
		if imp.Name != nil {
			local = imp.Name.Name
		} else {
			local = defaultLocalName(path)
		}
		result[local] = path
	}
	return result
}

func defaultLocalName(importPath string) string {
	if importPath == "" || !strings.Contains(importPath, "/") {
		return importPath
	}
	parts := strings.Split(importPath, "/")
	last := parts[len(parts)-1]
	if isMajorVersionSuffix(last) && len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return last
}

func isMajorVersionSuffix(s string) bool {
	if len(s) < 2 || s[0] != 'v' {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func hasSkipDirective(decs *dst.NodeDecs) bool {
	for _, c := range decs.Start.All() {
		if strings.Contains(c, "ctxweaver:skip") || strings.Contains(c, "DO NOT EDIT") {
			return true
		}
	}
	return false
}

// TransformSource transforms a single Go source file.
// This is useful for testing without requiring packages.Load.
func (p *Processor) TransformSource(src []byte, pkgName string) ([]byte, error) {
	// Parse with fresh fset for DST conversion
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source: %w", err)
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
	modified := p.processFunctionsForSource(df, pkgName)
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

	return buf.Bytes(), nil
}

func (p *Processor) processFunctionsForSource(df *dst.File, pkgName string) bool {
	modified := false
	aliases := resolveAliases(df.Imports)

	dst.Inspect(df, func(n dst.Node) bool {
		decl, ok := n.(*dst.FuncDecl)
		if !ok {
			return true
		}

		// Skip if function has skip directive
		if hasSkipDirective(decl.Decorations()) {
			return true
		}

		// Skip if no body
		if decl.Body == nil {
			return true
		}

		// Get first parameter
		param := extractFirstParam(decl)
		if param == nil {
			return true
		}

		// Check if first param is a context carrier
		carrier, varName, ok := p.matchCarrier(param, aliases)
		if !ok {
			return true
		}

		// Build template variables
		vars := p.buildVarsForSource(df, decl, pkgName, carrier, varName)

		// Render statement
		stmt, err := p.tmpl.Render(vars)
		if err != nil {
			// Skip on render error
			return true
		}

		// Check existing statement and update if needed
		action := p.detectAction(decl.Body, vars.FuncName)

		switch action.Type {
		case ActionInsert:
			if insertStatement(decl.Body, stmt) {
				modified = true
			}
		case ActionUpdate:
			if updateStatement(decl.Body, action.Index, stmt) {
				modified = true
			}
		case ActionSkip:
			// Already up to date
		}

		return true
	})

	return modified
}

func (p *Processor) buildVarsForSource(df *dst.File, decl *dst.FuncDecl, pkgName string, carrier config.CarrierDef, varName string) template.Vars {
	vars := template.Vars{
		Ctx:          carrier.BuildContextExpr(varName),
		CtxVar:       varName,
		PackageName:  df.Name.Name,
		PackagePath:  pkgName,
		FuncBaseName: decl.Name.Name,
	}

	// Build fully qualified function name
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		vars.IsMethod = true
		recv := decl.Recv.List[0]

		if len(recv.Names) > 0 {
			vars.ReceiverVar = recv.Names[0].Name
		}

		switch typ := recv.Type.(type) {
		case *dst.StarExpr:
			vars.IsPointerReceiver = true
			if ident, ok := typ.X.(*dst.Ident); ok {
				vars.ReceiverType = ident.Name
				vars.FuncName = fmt.Sprintf("%s.(*%s).%s", vars.PackageName, ident.Name, decl.Name.Name)
			}
		case *dst.Ident:
			vars.ReceiverType = typ.Name
			vars.FuncName = fmt.Sprintf("%s.%s.%s", vars.PackageName, typ.Name, decl.Name.Name)
		}
	} else {
		vars.FuncName = fmt.Sprintf("%s.%s", vars.PackageName, decl.Name.Name)
	}

	return vars
}
