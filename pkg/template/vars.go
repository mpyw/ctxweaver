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

	// Build fully qualified function name.
	//
	// The format mirrors how the Go runtime prints functions in stack traces:
	// type arguments are collapsed to "[...]", and a generic method carries its
	// own "[...]" independently of its receiver (e.g. "pkg.(*T[...]).M[...]").
	// Methods gained type parameters in Go 1.27.
	baseName := decl.Name.Name
	if funcHasTypeParams {
		baseName += "[...]"
	}

	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		vars.IsMethod = true
		recv := decl.Recv.List[0]

		if len(recv.Names) > 0 {
			vars.ReceiverVar = recv.Names[0].Name
		}

		// Extract receiver type name and check for generics
		recvTypeName, recvHasGenerics := extractReceiverVars(recv.Type)
		vars.ReceiverType = recvTypeName
		vars.IsGenericReceiver = recvHasGenerics

		recvName := recvTypeName
		if recvHasGenerics {
			recvName += "[...]"
		}

		if _, ok := recv.Type.(*dst.StarExpr); ok {
			vars.IsPointerReceiver = true
			vars.FuncName = fmt.Sprintf("%s.(*%s).%s", vars.PackageName, recvName, baseName)
		} else {
			vars.FuncName = fmt.Sprintf("%s.%s.%s", vars.PackageName, recvName, baseName)
		}
	} else {
		// Regular function (not a method)
		vars.FuncName = fmt.Sprintf("%s.%s", vars.PackageName, baseName)
	}

	return vars
}

// extractReceiverVars extracts the receiver-derived template variables from a
// receiver type expression: the base type name and whether it has type
// parameters. It handles regular types, pointer types, and generic types
// (IndexExpr, IndexListExpr).
func extractReceiverVars(expr dst.Expr) (name string, hasGenerics bool) {
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
			name, _ := extractReceiverVars(inner)
			return name, true
		}
		if inner, ok := t.X.(*dst.IndexListExpr); ok {
			name, _ := extractReceiverVars(inner)
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
