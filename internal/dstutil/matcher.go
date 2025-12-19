// Package dstutil provides utilities for DST (Decorated Syntax Tree) manipulation.
package dstutil

import (
	"reflect"

	"github.com/dave/dst"
)

// MatchesSkeleton compares two statements by their AST structure.
// It returns true if both statements have the same "skeleton" - same node types
// and static identifiers, but potentially different dynamic values (variables, literals).
func MatchesSkeleton(a, b dst.Stmt) bool {
	return compareNodes(a, b)
}

// compareNodes recursively compares two DST nodes for structural equality.
// Dynamic values (identifiers that are likely variables, string/number literals) are ignored.
func compareNodes(a, b dst.Node) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Must be same type
	if reflect.TypeOf(a) != reflect.TypeOf(b) {
		return false
	}

	switch nodeA := a.(type) {
	case *dst.DeferStmt:
		nodeB := b.(*dst.DeferStmt)
		return compareNodes(nodeA.Call, nodeB.Call)

	case *dst.ExprStmt:
		nodeB := b.(*dst.ExprStmt)
		return compareNodes(nodeA.X, nodeB.X)

	case *dst.IfStmt:
		nodeB := b.(*dst.IfStmt)
		return compareNodes(nodeA.Init, nodeB.Init) &&
			compareNodes(nodeA.Cond, nodeB.Cond) &&
			compareNodes(nodeA.Body, nodeB.Body) &&
			compareNodes(nodeA.Else, nodeB.Else)

	case *dst.SwitchStmt:
		nodeB := b.(*dst.SwitchStmt)
		return compareNodes(nodeA.Init, nodeB.Init) &&
			compareNodes(nodeA.Tag, nodeB.Tag) &&
			compareNodes(nodeA.Body, nodeB.Body)

	case *dst.BlockStmt:
		nodeB := b.(*dst.BlockStmt)
		if len(nodeA.List) != len(nodeB.List) {
			return false
		}
		for i := range nodeA.List {
			if !compareNodes(nodeA.List[i], nodeB.List[i]) {
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
			if !compareNodes(nodeA.Lhs[i], nodeB.Lhs[i]) {
				return false
			}
		}
		for i := range nodeA.Rhs {
			if !compareNodes(nodeA.Rhs[i], nodeB.Rhs[i]) {
				return false
			}
		}
		return true

	case *dst.CallExpr:
		nodeB := b.(*dst.CallExpr)
		if !compareNodes(nodeA.Fun, nodeB.Fun) {
			return false
		}
		if len(nodeA.Args) != len(nodeB.Args) {
			return false
		}
		for i := range nodeA.Args {
			if !compareNodes(nodeA.Args[i], nodeB.Args[i]) {
				return false
			}
		}
		return true

	case *dst.SelectorExpr:
		nodeB := b.(*dst.SelectorExpr)
		// Compare selector name (static identifier like method name)
		if nodeA.Sel.Name != nodeB.Sel.Name {
			return false
		}
		return compareNodes(nodeA.X, nodeB.X)

	case *dst.Ident:
		nodeB := b.(*dst.Ident)
		// Compare identifier names exactly
		return nodeA.Name == nodeB.Name

	case *dst.BasicLit:
		nodeB := b.(*dst.BasicLit)
		// For literals, compare types but not values
		// This allows different function names, different string values, etc.
		return nodeA.Kind == nodeB.Kind

	case *dst.UnaryExpr:
		nodeB := b.(*dst.UnaryExpr)
		return nodeA.Op == nodeB.Op && compareNodes(nodeA.X, nodeB.X)

	case *dst.BinaryExpr:
		nodeB := b.(*dst.BinaryExpr)
		return nodeA.Op == nodeB.Op &&
			compareNodes(nodeA.X, nodeB.X) &&
			compareNodes(nodeA.Y, nodeB.Y)

	case *dst.ParenExpr:
		nodeB := b.(*dst.ParenExpr)
		return compareNodes(nodeA.X, nodeB.X)

	case *dst.IndexExpr:
		nodeB := b.(*dst.IndexExpr)
		return compareNodes(nodeA.X, nodeB.X) && compareNodes(nodeA.Index, nodeB.Index)

	case *dst.FuncLit:
		nodeB := b.(*dst.FuncLit)
		return compareNodes(nodeA.Type, nodeB.Type) && compareNodes(nodeA.Body, nodeB.Body)

	case *dst.FuncType:
		nodeB := b.(*dst.FuncType)
		return compareFieldLists(nodeA.Params, nodeB.Params) &&
			compareFieldLists(nodeA.Results, nodeB.Results)

	case *dst.ReturnStmt:
		nodeB := b.(*dst.ReturnStmt)
		if len(nodeA.Results) != len(nodeB.Results) {
			return false
		}
		for i := range nodeA.Results {
			if !compareNodes(nodeA.Results[i], nodeB.Results[i]) {
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
			if !compareNodes(nodeA.List[i], nodeB.List[i]) {
				return false
			}
		}
		for i := range nodeA.Body {
			if !compareNodes(nodeA.Body[i], nodeB.Body[i]) {
				return false
			}
		}
		return true

	case *dst.CompositeLit:
		nodeB := b.(*dst.CompositeLit)
		if !compareNodes(nodeA.Type, nodeB.Type) {
			return false
		}
		if len(nodeA.Elts) != len(nodeB.Elts) {
			return false
		}
		for i := range nodeA.Elts {
			if !compareNodes(nodeA.Elts[i], nodeB.Elts[i]) {
				return false
			}
		}
		return true

	case *dst.KeyValueExpr:
		nodeB := b.(*dst.KeyValueExpr)
		return compareNodes(nodeA.Key, nodeB.Key) && compareNodes(nodeA.Value, nodeB.Value)

	case *dst.StarExpr:
		nodeB := b.(*dst.StarExpr)
		return compareNodes(nodeA.X, nodeB.X)

	case *dst.TypeAssertExpr:
		nodeB := b.(*dst.TypeAssertExpr)
		return compareNodes(nodeA.X, nodeB.X) && compareNodes(nodeA.Type, nodeB.Type)

	default:
		// For unsupported node types, fall back to type comparison only
		return true
	}
}

// compareFieldLists compares two field lists for structural equality.
func compareFieldLists(a, b *dst.FieldList) bool {
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
		if !compareFields(a.List[i], b.List[i]) {
			return false
		}
	}
	return true
}

// compareFields compares two fields for structural equality.
func compareFields(a, b *dst.Field) bool {
	// Compare types only (names are dynamic)
	return compareNodes(a.Type, b.Type)
}
