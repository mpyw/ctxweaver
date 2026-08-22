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
// Visitor Pattern: NodeComparer and Comparator
// ============================================================================

// NodeComparer compares two DST nodes of the concrete type T.
// Implementations receive already-narrowed nodes and delegate child
// comparisons back to the Comparator via c.Compare.
type NodeComparer[T dst.Node] func(a, b T, path string, exact bool, c *Comparator) bool

// erasedComparer is the type-erased form of a NodeComparer stored in the registry.
type erasedComparer = func(a, b dst.Node, path string, exact bool, c *Comparator) bool

// Comparator manages NodeComparer implementations and performs comparisons.
// It acts as a registry for node-specific comparers and handles dispatch.
type Comparator struct {
	comparers map[reflect.Type]erasedComparer
}

// NewComparator creates a new Comparator with the default set of comparers.
func NewComparator() *Comparator {
	c := &Comparator{
		comparers: make(map[reflect.Type]erasedComparer),
	}
	c.registerDefaults()
	return c
}

// Register adds a NodeComparer for the node type T.
// The node type is inferred from cmp, so callers state each type exactly once
// and comparers never assert their own argument types.
func (c *Comparator) Register[T dst.Node](cmp NodeComparer[T]) {
	c.comparers[reflect.TypeFor[T]()] = func(a, b dst.Node, path string, exact bool, comparator *Comparator) bool {
		return cmp(a.(T), b.(T), path, exact, comparator)
	}
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

	if cmp, ok := c.comparers[reflect.TypeOf(a)]; ok {
		return cmp(a, b, path, exact, c)
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
	c.Register(compareDeferStmt)
	c.Register(compareExprStmt)
	c.Register(compareIfStmt)
	c.Register(compareSwitchStmt)
	c.Register(compareBlockStmt)
	c.Register(compareAssignStmt)
	c.Register(compareReturnStmt)
	c.Register(compareCaseClause)

	// Expressions
	c.Register(compareCallExpr)
	c.Register(compareSelectorExpr)
	c.Register(compareIdent)
	c.Register(compareBasicLit)
	c.Register(compareUnaryExpr)
	c.Register(compareBinaryExpr)
	c.Register(compareParenExpr)
	c.Register(compareIndexExpr)
	c.Register(compareFuncLit)
	c.Register(compareFuncType)
	c.Register(compareCompositeLit)
	c.Register(compareKeyValueExpr)
	c.Register(compareStarExpr)
	c.Register(compareTypeAssertExpr)
}

// defaultComparator is the singleton instance used by public API.
var defaultComparator = NewComparator()

// ============================================================================
// Helper Functions
// ============================================================================

// compareNodeLists compares two slices of nodes element-wise.
func compareNodeLists[T dst.Node](a, b []T, path string, exact bool, c *Comparator) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !c.Compare(a[i], b[i], fmt.Sprintf("%s[%d]", path, i), exact) {
			return false
		}
	}
	return true
}

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
		// Compare types only (names are dynamic)
		if !c.Compare(a.List[i].Type, b.List[i].Type, fmt.Sprintf("%s[%d].Type", path, i), exact) {
			return false
		}
	}
	return true
}
