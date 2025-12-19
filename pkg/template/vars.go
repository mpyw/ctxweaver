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
