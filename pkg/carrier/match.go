// Package carrier provides carrier type matching for context propagation.
package carrier

import (
	"github.com/dave/dst"

	"github.com/mpyw/ctxweaver/pkg/config"
)

// Match extracts carrier info from a function parameter.
// It returns the carrier definition, variable name, and a boolean indicating success.
//
// The function supports:
//   - Direct types with resolved paths (from NewDecoratorFromPackage)
//   - Selector expressions (pkg.Type) with path set by decorator
//   - Pointer types (*T)
//
// Note: This requires type-resolved DST (via NewDecoratorFromPackage).
// The dst.Ident.Path field must be set for carrier matching to work.
//
// Parameters:
//   - param: The function parameter field to analyze
//   - registry: The carrier registry to lookup types
//
// Returns:
//   - config.CarrierDef: The matched carrier definition
//   - string: The variable name of the parameter
//   - bool: true if a carrier was matched, false otherwise
func Match(param *dst.Field, registry *config.CarrierRegistry) (config.CarrierDef, string, bool) {
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
		// SelectorExpr: pkg.Type with path set by NewDecoratorFromPackage
		pkgIdent, ok := t.X.(*dst.Ident)
		if !ok {
			return config.CarrierDef{}, "", false
		}
		pkgPath = pkgIdent.Path
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

	carrier, found := registry.Lookup(pkgPath, typeName)
	if !found {
		return config.CarrierDef{}, "", false
	}

	return carrier, varName, true
}
