package processor

import (
	"fmt"

	"github.com/dave/dst"
	"golang.org/x/tools/go/packages"

	"github.com/mpyw/ctxweaver/internal/dstutil"
	"github.com/mpyw/ctxweaver/pkg/config"
	"github.com/mpyw/ctxweaver/pkg/template"
)

// processFunctions processes functions using type info from packages.Package.
// Relies on dst.Ident.Path set by NewDecoratorFromPackage for import resolution.
func (p *Processor) processFunctions(df *dst.File, pkg *packages.Package) (bool, error) {
	return p.processFunctionsCore(df, pkg.PkgPath, nil)
}

// processFunctionsForSource processes functions using fuzzy alias resolution.
// Used by TransformSource when type info is not available.
func (p *Processor) processFunctionsForSource(df *dst.File, pkgPath string) (bool, error) {
	aliases := resolveAliases(df.Imports)
	return p.processFunctionsCore(df, pkgPath, aliases)
}

// processFunctionsCore is the shared implementation for processing functions.
// If aliases is nil, uses dst.Ident.Path for import resolution.
func (p *Processor) processFunctionsCore(df *dst.File, pkgPath string, aliases map[string]string) (bool, error) {
	modified := false
	var renderErr error

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
		vars := p.buildVars(df, decl, pkgPath, carrier, varName)

		// Render statement
		stmt, err := p.tmpl.Render(vars)
		if err != nil {
			renderErr = fmt.Errorf("function %s: %w", decl.Name.Name, err)
			return false // Stop inspection
		}

		// Check existing statement and determine action
		action, err := p.detectAction(decl.Body, stmt)
		if err != nil {
			renderErr = fmt.Errorf("function %s: %w", decl.Name.Name, err)
			return false // Stop inspection
		}

		switch action.actionType {
		case actionInsert:
			if dstutil.InsertStatements(decl.Body, stmt) {
				modified = true
			}
		case actionUpdate:
			if dstutil.UpdateStatements(decl.Body, action.index, action.count, stmt) {
				modified = true
			}
		case actionRemove:
			if dstutil.RemoveStatements(decl.Body, action.index, action.count) {
				modified = true
			}
		case actionSkip:
			// Already up to date (or nothing to remove)
		}

		return true
	})

	if renderErr != nil {
		return false, renderErr
	}
	return modified, nil
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

	var pkgPath, typeName string

	switch t := typ.(type) {
	case *dst.SelectorExpr:
		// SelectorExpr: pkg.Type (e.g., context.Context from source without type info)
		pkgIdent, ok := t.X.(*dst.Ident)
		if !ok {
			return config.CarrierDef{}, "", false
		}
		// Prefer type-resolved path from decorator (set by NewDecoratorFromPackage)
		// Fall back to fuzzy alias resolution when type info is not available
		pkgPath = pkgIdent.Path
		if pkgPath == "" && aliases != nil {
			pkgPath = aliases[pkgIdent.Name]
		}
		typeName = t.Sel.Name

	case *dst.Ident:
		// Ident with Path: NewDecoratorFromPackage resolves the type and sets Path
		pkgPath = t.Path
		typeName = t.Name

	default:
		return config.CarrierDef{}, "", false
	}

	if pkgPath == "" {
		return config.CarrierDef{}, "", false
	}

	carrier, found := p.registry.Lookup(pkgPath, typeName)
	if !found {
		return config.CarrierDef{}, "", false
	}

	return carrier, varName, true
}

func (p *Processor) buildVars(df *dst.File, decl *dst.FuncDecl, pkgPath string, carrier config.CarrierDef, varName string) template.Vars {
	vars := template.Vars{
		Ctx:          carrier.BuildContextExpr(varName),
		CtxVar:       varName,
		PackageName:  df.Name.Name,
		PackagePath:  pkgPath,
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
