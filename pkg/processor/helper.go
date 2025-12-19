package processor

import (
	"strings"

	"github.com/dave/dst"
)

func extractFirstParam(decl *dst.FuncDecl) *dst.Field {
	if decl.Type == nil || decl.Type.Params == nil || len(decl.Type.Params.List) == 0 {
		return nil
	}
	return decl.Type.Params.List[0]
}

// resolveAliases builds a map from local import names to package paths.
// This is used as a fallback for TransformSource when type info is not available.
// When using packages.Load + NewDecoratorFromPackage, dst.Ident.Path is set
// directly by the decorator, making this function unnecessary.
func resolveAliases(importSpecs []*dst.ImportSpec) map[string]string {
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

func hasSkipDirective(decs *dst.NodeDecs) bool {
	for _, c := range decs.Start.All() {
		if strings.Contains(c, "ctxweaver:skip") {
			return true
		}
	}
	return false
}

// hasStmtSkipDirective checks if a statement has a skip directive comment.
// This handles both "//ctxweaver:skip" and "// ctxweaver:skip" variants.
func hasStmtSkipDirective(stmt dst.Stmt) bool {
	decs := stmt.Decorations()
	// Check Start decorations (comments before the statement on the same line group)
	for _, c := range decs.Start.All() {
		if strings.Contains(c, "ctxweaver:skip") {
			return true
		}
	}
	// Check End decorations (trailing comments on the same line)
	for _, c := range decs.End.All() {
		if strings.Contains(c, "ctxweaver:skip") {
			return true
		}
	}
	return false
}
