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
	return defaultMatcher.Match(a, b, "root", false)
}

// MatchesExact compares two statements for exact equality.
// Unlike MatchesSkeleton, this also compares literal values.
func MatchesExact(a, b dst.Stmt) bool {
	return defaultMatcher.Match(a, b, "root", true)
}

// ============================================================================
// Visitor Pattern: NodeMatcher and Matcher
// ============================================================================

// NodeMatcher compares two DST nodes of the concrete type T.
// Implementations receive already-narrowed nodes and delegate child
// comparisons back to the Matcher via c.Match.
type NodeMatcher[T dst.Node] func(a, b T, path string, exact bool, c *Matcher) bool

// erasedMatcher is the type-erased form of a NodeMatcher stored in the registry.
type erasedMatcher = func(a, b dst.Node, path string, exact bool, c *Matcher) bool

// Matcher manages NodeMatcher implementations and performs comparisons.
// It acts as a registry for node-specific matchers and handles dispatch.
type Matcher struct {
	matchers map[reflect.Type]erasedMatcher
}

// NewMatcher creates a new Matcher with the default set of matchers.
func NewMatcher() *Matcher {
	c := &Matcher{
		matchers: make(map[reflect.Type]erasedMatcher),
	}
	c.registerDefaults()
	return c
}

// Register adds a NodeMatcher for the node type T.
// The node type is inferred from cmp, so callers state each type exactly once
// and matchers never assert their own argument types.
func (c *Matcher) Register[T dst.Node](cmp NodeMatcher[T]) {
	c.matchers[reflect.TypeFor[T]()] = func(a, b dst.Node, path string, exact bool, matcher *Matcher) bool {
		return cmp(a.(T), b.(T), path, exact, matcher)
	}
}

// Match reports whether two DST nodes match, using the registered matchers.
func (c *Matcher) Match(a, b dst.Node, path string, exact bool) bool {
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

	if cmp, ok := c.matchers[reflect.TypeOf(a)]; ok {
		return cmp(a, b, path, exact, c)
	}

	// Fallback: unsupported node types pass by default
	return true
}

// importEquivalent checks if two nodes of different types are equivalent
// due to import resolution (SelectorExpr vs Ident with Path).
// NewDecoratorFromPackage converts `pkg.Func` (SelectorExpr) to `Func` (Ident with Path set).
func (c *Matcher) importEquivalent(a, b dst.Node) bool {
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

// registerDefaults registers all built-in node matchers.
func (c *Matcher) registerDefaults() {
	// Statements
	c.Register(matchDeferStmt)
	c.Register(matchExprStmt)
	c.Register(matchIfStmt)
	c.Register(matchSwitchStmt)
	c.Register(matchBlockStmt)
	c.Register(matchAssignStmt)
	c.Register(matchReturnStmt)
	c.Register(matchCaseClause)

	// Expressions
	c.Register(matchCallExpr)
	c.Register(matchSelectorExpr)
	c.Register(matchIdent)
	c.Register(matchBasicLit)
	c.Register(matchUnaryExpr)
	c.Register(matchBinaryExpr)
	c.Register(matchParenExpr)
	c.Register(matchIndexExpr)
	c.Register(matchFuncLit)
	c.Register(matchFuncType)
	c.Register(matchCompositeLit)
	c.Register(matchKeyValueExpr)
	c.Register(matchStarExpr)
	c.Register(matchTypeAssertExpr)
}

// defaultMatcher is the singleton instance used by public API.
var defaultMatcher = NewMatcher()

// ============================================================================
// Helper Functions
// ============================================================================

// matchNodeLists compares two slices of nodes element-wise.
func matchNodeLists[T dst.Node](a, b []T, path string, exact bool, c *Matcher) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !c.Match(a[i], b[i], fmt.Sprintf("%s[%d]", path, i), exact) {
			return false
		}
	}
	return true
}

// matchFieldLists compares two field lists for structural equality.
func matchFieldLists(a, b *dst.FieldList, path string, exact bool, c *Matcher) bool {
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
		if !c.Match(a.List[i].Type, b.List[i].Type, fmt.Sprintf("%s[%d].Type", path, i), exact) {
			return false
		}
	}
	return true
}
