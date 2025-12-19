// Package dstutil provides utilities for DST (Decorated Syntax Tree) manipulation.
package dstutil

import (
	"fmt"
	"reflect"

	"github.com/dave/dst"
)

// MatchesSkeleton compares two statements by their AST structure.
// It returns true if both statements have the same "skeleton" - same node types
// and static identifiers, but potentially different dynamic values (variables, literals).
func MatchesSkeleton(a, b dst.Stmt) bool {
	return compareNodes(a, b, "root", false)
}

// MatchesExact compares two statements for exact equality.
// Unlike MatchesSkeleton, this also compares literal values.
func MatchesExact(a, b dst.Stmt) bool {
	return compareNodes(a, b, "root", true)
}

// compareNodes recursively compares two DST nodes for structural equality.
// When exactMode is false, dynamic values (string/number literals) are ignored.
// When exactMode is true, literal values are also compared.
func compareNodes(a, b dst.Node, path string, exactMode bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Must be same type, but handle SelectorExpr vs Ident with Path
	// NewDecoratorFromPackage converts `pkg.Func` (SelectorExpr) to `Func` (Ident with Path set)
	if reflect.TypeOf(a) != reflect.TypeOf(b) {
		// Special case: SelectorExpr vs Ident (due to import resolution)
		if selA, okA := a.(*dst.SelectorExpr); okA {
			if identB, okB := b.(*dst.Ident); okB && identB.Path != "" {
				return selA.Sel.Name == identB.Name
			}
		}
		if identA, okA := a.(*dst.Ident); okA && identA.Path != "" {
			if selB, okB := b.(*dst.SelectorExpr); okB {
				return identA.Name == selB.Sel.Name
			}
		}
		return false
	}

	switch nodeA := a.(type) {
	case *dst.DeferStmt:
		nodeB := b.(*dst.DeferStmt)
		return compareNodes(nodeA.Call, nodeB.Call, path+".Call", exactMode)

	case *dst.ExprStmt:
		nodeB := b.(*dst.ExprStmt)
		return compareNodes(nodeA.X, nodeB.X, path+".X", exactMode)

	case *dst.IfStmt:
		nodeB := b.(*dst.IfStmt)
		return compareNodes(nodeA.Init, nodeB.Init, path+".Init", exactMode) &&
			compareNodes(nodeA.Cond, nodeB.Cond, path+".Cond", exactMode) &&
			compareNodes(nodeA.Body, nodeB.Body, path+".Body", exactMode) &&
			compareNodes(nodeA.Else, nodeB.Else, path+".Else", exactMode)

	case *dst.SwitchStmt:
		nodeB := b.(*dst.SwitchStmt)
		return compareNodes(nodeA.Init, nodeB.Init, path+".Init", exactMode) &&
			compareNodes(nodeA.Tag, nodeB.Tag, path+".Tag", exactMode) &&
			compareNodes(nodeA.Body, nodeB.Body, path+".Body", exactMode)

	case *dst.BlockStmt:
		nodeB := b.(*dst.BlockStmt)
		if len(nodeA.List) != len(nodeB.List) {
			return false
		}
		for i := range nodeA.List {
			if !compareNodes(nodeA.List[i], nodeB.List[i], fmt.Sprintf("%s.List[%d]", path, i), exactMode) {
				return false
			}
		}
		return true

	case *dst.AssignStmt:
		nodeB := b.(*dst.AssignStmt)
		if nodeA.Tok != nodeB.Tok {
			return false
		}
		if len(nodeA.Lhs) != len(nodeB.Lhs) || len(nodeA.Rhs) != len(nodeB.Rhs) {
			return false
		}
		for i := range nodeA.Lhs {
			if !compareNodes(nodeA.Lhs[i], nodeB.Lhs[i], fmt.Sprintf("%s.Lhs[%d]", path, i), exactMode) {
				return false
			}
		}
		for i := range nodeA.Rhs {
			if !compareNodes(nodeA.Rhs[i], nodeB.Rhs[i], fmt.Sprintf("%s.Rhs[%d]", path, i), exactMode) {
				return false
			}
		}
		return true

	case *dst.CallExpr:
		nodeB := b.(*dst.CallExpr)
		if !compareNodes(nodeA.Fun, nodeB.Fun, path+".Fun", exactMode) {
			return false
		}
		if len(nodeA.Args) != len(nodeB.Args) {
			return false
		}
		for i := range nodeA.Args {
			if !compareNodes(nodeA.Args[i], nodeB.Args[i], fmt.Sprintf("%s.Args[%d]", path, i), exactMode) {
				return false
			}
		}
		return true

	case *dst.SelectorExpr:
		nodeB := b.(*dst.SelectorExpr)
		if nodeA.Sel.Name != nodeB.Sel.Name {
			return false
		}
		return compareNodes(nodeA.X, nodeB.X, path+".X", exactMode)

	case *dst.Ident:
		nodeB := b.(*dst.Ident)
		return nodeA.Name == nodeB.Name

	case *dst.BasicLit:
		nodeB := b.(*dst.BasicLit)
		if nodeA.Kind != nodeB.Kind {
			return false
		}
		if exactMode && nodeA.Value != nodeB.Value {
			return false
		}
		return true

	case *dst.UnaryExpr:
		nodeB := b.(*dst.UnaryExpr)
		if nodeA.Op != nodeB.Op {
			return false
		}
		return compareNodes(nodeA.X, nodeB.X, path+".X", exactMode)

	case *dst.BinaryExpr:
		nodeB := b.(*dst.BinaryExpr)
		if nodeA.Op != nodeB.Op {
			return false
		}
		return compareNodes(nodeA.X, nodeB.X, path+".X", exactMode) &&
			compareNodes(nodeA.Y, nodeB.Y, path+".Y", exactMode)

	case *dst.ParenExpr:
		nodeB := b.(*dst.ParenExpr)
		return compareNodes(nodeA.X, nodeB.X, path+".X", exactMode)

	case *dst.IndexExpr:
		nodeB := b.(*dst.IndexExpr)
		return compareNodes(nodeA.X, nodeB.X, path+".X", exactMode) &&
			compareNodes(nodeA.Index, nodeB.Index, path+".Index", exactMode)

	case *dst.FuncLit:
		nodeB := b.(*dst.FuncLit)
		return compareNodes(nodeA.Type, nodeB.Type, path+".Type", exactMode) &&
			compareNodes(nodeA.Body, nodeB.Body, path+".Body", exactMode)

	case *dst.FuncType:
		nodeB := b.(*dst.FuncType)
		return compareFieldLists(nodeA.Params, nodeB.Params, path+".Params", exactMode) &&
			compareFieldLists(nodeA.Results, nodeB.Results, path+".Results", exactMode)

	case *dst.ReturnStmt:
		nodeB := b.(*dst.ReturnStmt)
		if len(nodeA.Results) != len(nodeB.Results) {
			return false
		}
		for i := range nodeA.Results {
			if !compareNodes(nodeA.Results[i], nodeB.Results[i], fmt.Sprintf("%s.Results[%d]", path, i), exactMode) {
				return false
			}
		}
		return true

	case *dst.CaseClause:
		nodeB := b.(*dst.CaseClause)
		if len(nodeA.List) != len(nodeB.List) || len(nodeA.Body) != len(nodeB.Body) {
			return false
		}
		for i := range nodeA.List {
			if !compareNodes(nodeA.List[i], nodeB.List[i], fmt.Sprintf("%s.List[%d]", path, i), exactMode) {
				return false
			}
		}
		for i := range nodeA.Body {
			if !compareNodes(nodeA.Body[i], nodeB.Body[i], fmt.Sprintf("%s.Body[%d]", path, i), exactMode) {
				return false
			}
		}
		return true

	case *dst.CompositeLit:
		nodeB := b.(*dst.CompositeLit)
		if !compareNodes(nodeA.Type, nodeB.Type, path+".Type", exactMode) {
			return false
		}
		if len(nodeA.Elts) != len(nodeB.Elts) {
			return false
		}
		for i := range nodeA.Elts {
			if !compareNodes(nodeA.Elts[i], nodeB.Elts[i], fmt.Sprintf("%s.Elts[%d]", path, i), exactMode) {
				return false
			}
		}
		return true

	case *dst.KeyValueExpr:
		nodeB := b.(*dst.KeyValueExpr)
		return compareNodes(nodeA.Key, nodeB.Key, path+".Key", exactMode) &&
			compareNodes(nodeA.Value, nodeB.Value, path+".Value", exactMode)

	case *dst.StarExpr:
		nodeB := b.(*dst.StarExpr)
		return compareNodes(nodeA.X, nodeB.X, path+".X", exactMode)

	case *dst.TypeAssertExpr:
		nodeB := b.(*dst.TypeAssertExpr)
		return compareNodes(nodeA.X, nodeB.X, path+".X", exactMode) &&
			compareNodes(nodeA.Type, nodeB.Type, path+".Type", exactMode)

	default:
		// For unsupported node types, fall back to type comparison only
		return true
	}
}

// compareFieldLists compares two field lists for structural equality.
func compareFieldLists(a, b *dst.FieldList, path string, exactMode bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a.List) != len(b.List) {
		return false
	}
	for i := range a.List {
		if !compareFields(a.List[i], b.List[i], fmt.Sprintf("%s[%d]", path, i), exactMode) {
			return false
		}
	}
	return true
}

// compareFields compares two fields for structural equality.
func compareFields(a, b *dst.Field, path string, exactMode bool) bool {
	// Compare types only (names are dynamic)
	return compareNodes(a.Type, b.Type, path+".Type", exactMode)
}
