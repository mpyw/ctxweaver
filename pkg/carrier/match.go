// Package carrier provides carrier type matching for context propagation.
package carrier

import (
	"github.com/dave/dst"

	"github.com/mpyw/ctxweaver/pkg/config"
)

// MatchResult represents a successful carrier match.
// It captures the validated state after matching, eliminating
// the need for callers to handle multiple return values.
type MatchResult struct {
	Carrier config.CarrierDef
	VarName string
}

// Match extracts carrier info from a function parameter.
// It returns a MatchResult if the parameter matches a registered carrier,
// or nil if no match is found.
//
// The function supports:
//   - Direct types with resolved paths (from NewDecoratorFromPackage)
//   - Selector expressions (pkg.Type) with path set by decorator
//   - Pointer types (*T)
//
// Note: This requires type-resolved DST (via NewDecoratorFromPackage).
// The dst.Ident.Path field must be set for carrier matching to work.
func Match(param *dst.Field, registry *config.CarrierRegistry) *MatchResult {
	if len(param.Names) == 0 || param.Names[0].Name == "_" {
		return nil
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
		// SelectorExpr: pkg.Type with path set by NewDecoratorFromPackage
		pkgIdent, ok := t.X.(*dst.Ident)
		if !ok {
			return nil
		}
		pkgPath = pkgIdent.Path
		typeName = t.Sel.Name

	case *dst.Ident:
		// Ident with Path: NewDecoratorFromPackage resolves the type and sets Path
		pkgPath = t.Path
		typeName = t.Name

	default:
		return nil
	}

	if pkgPath == "" {
		return nil
	}

	carrier, found := registry.Lookup(pkgPath, typeName)
	if !found {
		return nil
	}

	return &MatchResult{
		Carrier: carrier,
		VarName: varName,
	}
}
