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
	return defaultMatcher.match(a, b, "root", false)
}

// MatchesExact compares two statements for exact equality.
// Unlike MatchesSkeleton, this also compares literal values.
func MatchesExact(a, b dst.Stmt) bool {
	return defaultMatcher.match(a, b, "root", true)
}

// ============================================================================
// Visitor Pattern: nodeMatcher and matcher
// ============================================================================

// nodeMatcher compares two DST nodes of the concrete type T.
// Implementations receive already-narrowed nodes and delegate child
// comparisons back to the matcher via c.match.
type nodeMatcher[T dst.Node] func(a, b T, path string, exact bool, c *matcher) bool

// erasedMatcher is the type-erased form of a nodeMatcher stored in the registry.
type erasedMatcher = func(a, b dst.Node, path string, exact bool, c *matcher) bool

// matcher manages nodeMatcher implementations and performs comparisons.
// It acts as a registry for node-specific matchers and handles dispatch.
type matcher struct {
	matchers map[reflect.Type]erasedMatcher
}

// newMatcher creates a new matcher with the default set of matchers.
func newMatcher() *matcher {
	c := &matcher{
		matchers: make(map[reflect.Type]erasedMatcher),
	}
	c.registerDefaults()
	return c
}

// register adds a nodeMatcher for the node type T.
// The node type is inferred from cmp, so callers state each type exactly once
// and matchers never assert their own argument types.
func (c *matcher) register[T dst.Node](cmp nodeMatcher[T]) {
	c.matchers[reflect.TypeFor[T]()] = func(a, b dst.Node, path string, exact bool, matcher *matcher) bool {
		return cmp(a.(T), b.(T), path, exact, matcher)
	}
}

// match reports whether two DST nodes match, using the registered matchers.
func (c *matcher) match(a, b dst.Node, path string, exact bool) bool {
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
func (c *matcher) importEquivalent(a, b dst.Node) bool {
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
func (c *matcher) registerDefaults() {
	// Statements
	c.register(matchDeferStmt)
	c.register(matchExprStmt)
	c.register(matchIfStmt)
	c.register(matchSwitchStmt)
	c.register(matchBlockStmt)
	c.register(matchAssignStmt)
	c.register(matchReturnStmt)
	c.register(matchCaseClause)

	// Expressions
	c.register(matchCallExpr)
	c.register(matchSelectorExpr)
	c.register(matchIdent)
	c.register(matchBasicLit)
	c.register(matchUnaryExpr)
	c.register(matchBinaryExpr)
	c.register(matchParenExpr)
	c.register(matchIndexExpr)
	c.register(matchFuncLit)
	c.register(matchFuncType)
	c.register(matchCompositeLit)
	c.register(matchKeyValueExpr)
	c.register(matchStarExpr)
	c.register(matchTypeAssertExpr)
}

// defaultMatcher is the singleton instance used by public API.
var defaultMatcher = newMatcher()

// ============================================================================
// Helper Functions
// ============================================================================

// matchNodeLists compares two slices of nodes element-wise.
func matchNodeLists[T dst.Node](a, b []T, path string, exact bool, c *matcher) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !c.match(a[i], b[i], fmt.Sprintf("%s[%d]", path, i), exact) {
			return false
		}
	}
	return true
}

// matchFieldLists compares two field lists for structural equality.
func matchFieldLists(a, b *dst.FieldList, path string, exact bool, c *matcher) bool {
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
		if !c.match(a.List[i].Type, b.List[i].Type, fmt.Sprintf("%s[%d].Type", path, i), exact) {
			return false
		}
	}
	return true
}
