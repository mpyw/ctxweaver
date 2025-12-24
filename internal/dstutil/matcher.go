// Package dstutil provides utilities for DST (Decorated Syntax Tree) manipulation.
package dstutil

import (
	"fmt"
	"reflect"

	"github.com/dave/dst"
)

// ============================================================================
// Public API
// ============================================================================

// MatchesSkeleton compares two statements by their AST structure.
// It returns true if both statements have the same "skeleton" - same node types
// and static identifiers, but potentially different dynamic values (variables, literals).
func MatchesSkeleton(a, b dst.Stmt) bool {
	return defaultComparator.Compare(a, b, "root", false)
}

// MatchesExact compares two statements for exact equality.
// Unlike MatchesSkeleton, this also compares literal values.
func MatchesExact(a, b dst.Stmt) bool {
	return defaultComparator.Compare(a, b, "root", true)
}

// ============================================================================
// Visitor Pattern: NodeComparer interface and Comparator
// ============================================================================

// NodeComparer defines the interface for comparing specific DST node types.
// Implementations handle the comparison logic for a single node type,
// delegating child node comparisons back to the Comparator.
type NodeComparer interface {
	// Compare compares two nodes of the same type.
	// The nodes are guaranteed to be of the type this comparer handles.
	// Use c.Compare for recursive child comparisons.
	Compare(a, b dst.Node, path string, exact bool, c *Comparator) bool
}

// Comparator manages NodeComparer implementations and performs comparisons.
// It acts as a registry for node-specific comparers and handles dispatch.
type Comparator struct {
	comparers map[reflect.Type]NodeComparer
}

// NewComparator creates a new Comparator with the default set of comparers.
func NewComparator() *Comparator {
	c := &Comparator{
		comparers: make(map[reflect.Type]NodeComparer),
	}
	c.registerDefaults()
	return c
}

// Register adds a NodeComparer for a specific node type.
func (c *Comparator) Register(nodeType reflect.Type, comparer NodeComparer) {
	c.comparers[nodeType] = comparer
}

// Compare compares two DST nodes using the registered comparers.
func (c *Comparator) Compare(a, b dst.Node, path string, exact bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Handle SelectorExpr vs Ident with Path (import resolution difference)
	// If types differ but are import-equivalent, comparison is complete
	if reflect.TypeOf(a) != reflect.TypeOf(b) {
		return c.importEquivalent(a, b)
	}

	nodeType := reflect.TypeOf(a)
	if comparer, ok := c.comparers[nodeType]; ok {
		return comparer.Compare(a, b, path, exact, c)
	}

	// Fallback: unsupported node types pass by default
	return true
}

// importEquivalent checks if two nodes of different types are equivalent
// due to import resolution (SelectorExpr vs Ident with Path).
// NewDecoratorFromPackage converts `pkg.Func` (SelectorExpr) to `Func` (Ident with Path set).
func (c *Comparator) importEquivalent(a, b dst.Node) bool {
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

// registerDefaults registers all built-in node comparers.
func (c *Comparator) registerDefaults() {
	// Statements
	c.Register(reflect.TypeOf((*dst.DeferStmt)(nil)), &deferStmtComparer{})
	c.Register(reflect.TypeOf((*dst.ExprStmt)(nil)), &exprStmtComparer{})
	c.Register(reflect.TypeOf((*dst.IfStmt)(nil)), &ifStmtComparer{})
	c.Register(reflect.TypeOf((*dst.SwitchStmt)(nil)), &switchStmtComparer{})
	c.Register(reflect.TypeOf((*dst.BlockStmt)(nil)), &blockStmtComparer{})
	c.Register(reflect.TypeOf((*dst.AssignStmt)(nil)), &assignStmtComparer{})
	c.Register(reflect.TypeOf((*dst.ReturnStmt)(nil)), &returnStmtComparer{})
	c.Register(reflect.TypeOf((*dst.CaseClause)(nil)), &caseClauseComparer{})

	// Expressions
	c.Register(reflect.TypeOf((*dst.CallExpr)(nil)), &callExprComparer{})
	c.Register(reflect.TypeOf((*dst.SelectorExpr)(nil)), &selectorExprComparer{})
	c.Register(reflect.TypeOf((*dst.Ident)(nil)), &identComparer{})
	c.Register(reflect.TypeOf((*dst.BasicLit)(nil)), &basicLitComparer{})
	c.Register(reflect.TypeOf((*dst.UnaryExpr)(nil)), &unaryExprComparer{})
	c.Register(reflect.TypeOf((*dst.BinaryExpr)(nil)), &binaryExprComparer{})
	c.Register(reflect.TypeOf((*dst.ParenExpr)(nil)), &parenExprComparer{})
	c.Register(reflect.TypeOf((*dst.IndexExpr)(nil)), &indexExprComparer{})
	c.Register(reflect.TypeOf((*dst.FuncLit)(nil)), &funcLitComparer{})
	c.Register(reflect.TypeOf((*dst.FuncType)(nil)), &funcTypeComparer{})
	c.Register(reflect.TypeOf((*dst.CompositeLit)(nil)), &compositeLitComparer{})
	c.Register(reflect.TypeOf((*dst.KeyValueExpr)(nil)), &keyValueExprComparer{})
	c.Register(reflect.TypeOf((*dst.StarExpr)(nil)), &starExprComparer{})
	c.Register(reflect.TypeOf((*dst.TypeAssertExpr)(nil)), &typeAssertExprComparer{})
}

// defaultComparator is the singleton instance used by public API.
var defaultComparator = NewComparator()

// ============================================================================
// Helper Functions
// ============================================================================

// compareFieldLists compares two field lists for structural equality.
func compareFieldLists(a, b *dst.FieldList, path string, exact bool, c *Comparator) bool {
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
		if !compareFields(a.List[i], b.List[i], fmt.Sprintf("%s[%d]", path, i), exact, c) {
			return false
		}
	}
	return true
}

// compareFields compares two fields for structural equality.
func compareFields(a, b *dst.Field, path string, exact bool, c *Comparator) bool {
	// Compare types only (names are dynamic)
	return c.Compare(a.Type, b.Type, path+".Type", exact)
}
