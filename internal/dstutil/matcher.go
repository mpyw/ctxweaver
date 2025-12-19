// Package dstutil provides utilities for DST (Decorated Syntax Tree) manipulation.
package dstutil

import (
	"fmt"
	"reflect"

	"github.com/dave/dst"
)

// DebugSkeleton enables debug output for skeleton matching
var DebugSkeleton = false

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
		if DebugSkeleton {
			fmt.Printf("[skeleton] %s: nil mismatch (a=%v, b=%v)\n", path, a, b)
		}
		return false
	}

	// Must be same type, but handle SelectorExpr vs Ident with Path
	// NewDecoratorFromPackage converts `pkg.Func` (SelectorExpr) to `Func` (Ident with Path set)
	if reflect.TypeOf(a) != reflect.TypeOf(b) {
		// Special case: SelectorExpr vs Ident (due to import resolution)
		if selA, okA := a.(*dst.SelectorExpr); okA {
			if identB, okB := b.(*dst.Ident); okB && identB.Path != "" {
				// Compare: selA.Sel.Name should match identB.Name
				if selA.Sel.Name != identB.Name {
					if DebugSkeleton {
						fmt.Printf("[skeleton] %s: SelectorExpr.Sel vs Ident name mismatch %q vs %q\n", path, selA.Sel.Name, identB.Name)
					}
					return false
				}
				if DebugSkeleton {
					fmt.Printf("[skeleton] %s: SelectorExpr(%s) matches Ident(%s, path=%s)\n", path, selA.Sel.Name, identB.Name, identB.Path)
				}
				return true
			}
		}
		if identA, okA := a.(*dst.Ident); okA && identA.Path != "" {
			if selB, okB := b.(*dst.SelectorExpr); okB {
				if identA.Name != selB.Sel.Name {
					if DebugSkeleton {
						fmt.Printf("[skeleton] %s: Ident vs SelectorExpr.Sel name mismatch %q vs %q\n", path, identA.Name, selB.Sel.Name)
					}
					return false
				}
				if DebugSkeleton {
					fmt.Printf("[skeleton] %s: Ident(%s, path=%s) matches SelectorExpr(%s)\n", path, identA.Name, identA.Path, selB.Sel.Name)
				}
				return true
			}
		}
		if DebugSkeleton {
			fmt.Printf("[skeleton] %s: type mismatch %T vs %T\n", path, a, b)
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
			if DebugSkeleton {
				fmt.Printf("[skeleton] %s: list length mismatch %d vs %d\n", path, len(nodeA.List), len(nodeB.List))
			}
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
			if DebugSkeleton {
				fmt.Printf("[skeleton] %s: token mismatch %v vs %v\n", path, nodeA.Tok, nodeB.Tok)
			}
			return false
		}
		if len(nodeA.Lhs) != len(nodeB.Lhs) || len(nodeA.Rhs) != len(nodeB.Rhs) {
			if DebugSkeleton {
				fmt.Printf("[skeleton] %s: lhs/rhs length mismatch\n", path)
			}
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
			if DebugSkeleton {
				fmt.Printf("[skeleton] %s: args length mismatch %d vs %d\n", path, len(nodeA.Args), len(nodeB.Args))
			}
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
		// Compare selector name (static identifier like method name)
		if nodeA.Sel.Name != nodeB.Sel.Name {
			if DebugSkeleton {
				fmt.Printf("[skeleton] %s: Sel.Name mismatch %q vs %q\n", path, nodeA.Sel.Name, nodeB.Sel.Name)
			}
			return false
		}
		return compareNodes(nodeA.X, nodeB.X, path+".X", exactMode)

	case *dst.Ident:
		nodeB := b.(*dst.Ident)
		// Compare identifier names exactly
		if nodeA.Name != nodeB.Name {
			if DebugSkeleton {
				fmt.Printf("[skeleton] %s: Ident.Name mismatch %q vs %q\n", path, nodeA.Name, nodeB.Name)
			}
			return false
		}
		return true

	case *dst.BasicLit:
		nodeB := b.(*dst.BasicLit)
		// For literals, always compare types
		if nodeA.Kind != nodeB.Kind {
			if DebugSkeleton {
				fmt.Printf("[skeleton] %s: BasicLit.Kind mismatch %v vs %v\n", path, nodeA.Kind, nodeB.Kind)
			}
			return false
		}
		// In exact mode, also compare values
		if exactMode && nodeA.Value != nodeB.Value {
			if DebugSkeleton {
				fmt.Printf("[skeleton] %s: BasicLit.Value mismatch %q vs %q\n", path, nodeA.Value, nodeB.Value)
			}
			return false
		}
		return true

	case *dst.UnaryExpr:
		nodeB := b.(*dst.UnaryExpr)
		if nodeA.Op != nodeB.Op {
			if DebugSkeleton {
				fmt.Printf("[skeleton] %s: UnaryExpr.Op mismatch %v vs %v\n", path, nodeA.Op, nodeB.Op)
			}
			return false
		}
		return compareNodes(nodeA.X, nodeB.X, path+".X", exactMode)

	case *dst.BinaryExpr:
		nodeB := b.(*dst.BinaryExpr)
		if nodeA.Op != nodeB.Op {
			if DebugSkeleton {
				fmt.Printf("[skeleton] %s: BinaryExpr.Op mismatch %v vs %v\n", path, nodeA.Op, nodeB.Op)
			}
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
			if DebugSkeleton {
				fmt.Printf("[skeleton] %s: Results length mismatch %d vs %d\n", path, len(nodeA.Results), len(nodeB.Results))
			}
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
			if DebugSkeleton {
				fmt.Printf("[skeleton] %s: CaseClause length mismatch\n", path)
			}
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
			if DebugSkeleton {
				fmt.Printf("[skeleton] %s: Elts length mismatch %d vs %d\n", path, len(nodeA.Elts), len(nodeB.Elts))
			}
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
		if DebugSkeleton {
			fmt.Printf("[skeleton] %s: unhandled type %T (allowing)\n", path, a)
		}
		return true
	}
}

// compareFieldLists compares two field lists for structural equality.
func compareFieldLists(a, b *dst.FieldList, path string, exactMode bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		if DebugSkeleton {
			fmt.Printf("[skeleton] %s: FieldList nil mismatch\n", path)
		}
		return false
	}
	if len(a.List) != len(b.List) {
		if DebugSkeleton {
			fmt.Printf("[skeleton] %s: FieldList length mismatch %d vs %d\n", path, len(a.List), len(b.List))
		}
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
