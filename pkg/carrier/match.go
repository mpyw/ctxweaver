package carrier

import (
	"strings"

	"github.com/dave/dst"

	"github.com/mpyw/ctxweaver/pkg/config"
)

// Match extracts carrier info from a function parameter.
// It returns the carrier definition, variable name, and a boolean indicating success.
//
// The function supports:
//   - Direct types with resolved paths (from NewDecoratorFromPackage)
//   - Selector expressions (pkg.Type) with fuzzy alias resolution fallback
//   - Pointer types (*T)
//
// Parameters:
//   - param: The function parameter field to analyze
//   - aliases: Optional map of import aliases to package paths (for fuzzy resolution)
//   - registry: The carrier registry to lookup types
//
// Returns:
//   - config.CarrierDef: The matched carrier definition
//   - string: The variable name of the parameter
//   - bool: true if a carrier was matched, false otherwise
func Match(param *dst.Field, aliases map[string]string, registry *config.CarrierRegistry) (config.CarrierDef, string, bool) {
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

	carrier, found := registry.Lookup(pkgPath, typeName)
	if !found {
		return config.CarrierDef{}, "", false
	}

	return carrier, varName, true
}

// ResolveAliases builds a map from local import names to package paths.
// This is used as a fallback for TransformSource when type info is not available.
// When using packages.Load + NewDecoratorFromPackage, dst.Ident.Path is set
// directly by the decorator, making this function unnecessary.
func ResolveAliases(importSpecs []*dst.ImportSpec) map[string]string {
	result := make(map[string]string)
	for _, imp := range importSpecs {
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
