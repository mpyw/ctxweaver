package template

import (
	"fmt"

	"github.com/dave/dst"

	"github.com/mpyw/ctxweaver/pkg/config"
)

// BuildVars constructs a Vars instance from AST nodes and carrier definition.
// This function extracts all necessary information from the function declaration
// and builds template variables that can be used for statement rendering.
func BuildVars(df *dst.File, decl *dst.FuncDecl, pkgPath string, carrier config.CarrierDef, varName string) Vars {
	vars := Vars{
		Ctx:          carrier.BuildContextExpr(varName),
		CtxVar:       varName,
		PackageName:  df.Name.Name,
		PackagePath:  pkgPath,
		FuncBaseName: decl.Name.Name,
	}

	// Check if the function itself has type parameters
	funcHasTypeParams := decl.Type.TypeParams != nil && len(decl.Type.TypeParams.List) > 0
	vars.IsGenericFunc = funcHasTypeParams

	// Build fully qualified function name
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		vars.IsMethod = true
		recv := decl.Recv.List[0]

		if len(recv.Names) > 0 {
			vars.ReceiverVar = recv.Names[0].Name
		}

		// Extract receiver type name and check for generics
		recvTypeName, recvHasGenerics := extractReceiverTypeName(recv.Type)
		vars.ReceiverType = recvTypeName
		vars.IsGenericReceiver = recvHasGenerics

		switch recv.Type.(type) {
		case *dst.StarExpr:
			vars.IsPointerReceiver = true
			if recvHasGenerics {
				vars.FuncName = fmt.Sprintf("%s.(*%s[...]).%s", vars.PackageName, recvTypeName, decl.Name.Name)
			} else {
				vars.FuncName = fmt.Sprintf("%s.(*%s).%s", vars.PackageName, recvTypeName, decl.Name.Name)
			}
		default:
			if recvHasGenerics {
				vars.FuncName = fmt.Sprintf("%s.%s[...].%s", vars.PackageName, recvTypeName, decl.Name.Name)
			} else {
				vars.FuncName = fmt.Sprintf("%s.%s.%s", vars.PackageName, recvTypeName, decl.Name.Name)
			}
		}
	} else {
		// Regular function (not a method)
		if funcHasTypeParams {
			vars.FuncName = fmt.Sprintf("%s.%s[...]", vars.PackageName, decl.Name.Name)
		} else {
			vars.FuncName = fmt.Sprintf("%s.%s", vars.PackageName, decl.Name.Name)
		}
	}

	return vars
}

// extractReceiverTypeName extracts the base type name from a receiver type expression.
// It handles regular types, pointer types, and generic types (IndexExpr, IndexListExpr).
// Returns the type name and a boolean indicating whether it has type parameters.
func extractReceiverTypeName(expr dst.Expr) (name string, hasGenerics bool) {
	// Unwrap pointer if present
	if star, ok := expr.(*dst.StarExpr); ok {
		expr = star.X
	}

	switch t := expr.(type) {
	case *dst.Ident:
		// Simple type: T
		return t.Name, false

	case *dst.IndexExpr:
		// Generic type with single type parameter: T[X]
		if ident, ok := t.X.(*dst.Ident); ok {
			return ident.Name, true
		}
		// Nested generics: T[X[Y]] - recursively extract the outermost type name.
		// These branches handle extremely rare nested generic receiver patterns.
		// In practice, receiver types are almost always simple generics like T[X].
		if inner, ok := t.X.(*dst.IndexExpr); ok {
			name, _ := extractReceiverTypeName(inner)
			return name, true
		}
		if inner, ok := t.X.(*dst.IndexListExpr); ok {
			name, _ := extractReceiverTypeName(inner)
			return name, true
		}

	case *dst.IndexListExpr:
		// Generic type with multiple type parameters: T[X, Y]
		// Note: t.X is always *dst.Ident because T[X, Y][Z] is syntactically invalid in Go.
		// Nesting occurs in Indices (e.g., T[X, Y[Z]]), not in X.
		if ident, ok := t.X.(*dst.Ident); ok {
			return ident.Name, true
		}
	}

	return "", false
}
