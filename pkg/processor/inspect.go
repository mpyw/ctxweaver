package processor

import (
	"fmt"

	"github.com/dave/dst"

	"github.com/mpyw/ctxweaver/internal/directive"
	"github.com/mpyw/ctxweaver/internal/dstutil"
	"github.com/mpyw/ctxweaver/pkg/carrier"
	"github.com/mpyw/ctxweaver/pkg/template"
)

func extractFirstParam(decl *dst.FuncDecl) *dst.Field {
	if decl.Type == nil || decl.Type.Params == nil || len(decl.Type.Params.List) == 0 {
		return nil
	}
	return decl.Type.Params.List[0]
}

// processFunctions processes functions in the DST file.
// Relies on dst.Ident.Path set by NewDecoratorFromPackage for import resolution.
func (p *Processor) processFunctions(df *dst.File, pkgPath string) (bool, error) {
	modified := false
	var renderErr error

	dst.Inspect(df, func(n dst.Node) bool {
		decl, ok := n.(*dst.FuncDecl)
		if !ok {
			return true
		}

		// Skip if function has skip directive
		if directive.HasSkipDirective(decl.Decorations()) {
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
		carrierDef, varName, ok := carrier.Match(param, p.registry)
		if !ok {
			return true
		}

		// Build template variables
		vars := template.BuildVars(df, decl, pkgPath, carrierDef, varName)

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
