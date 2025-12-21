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
// Statement Comparers
// ============================================================================

type deferStmtComparer struct{}

func (deferStmtComparer) Compare(a, b dst.Node, path string, exact bool, c *Comparator) bool {
	nodeA, nodeB := a.(*dst.DeferStmt), b.(*dst.DeferStmt)
	return c.Compare(nodeA.Call, nodeB.Call, path+".Call", exact)
}

type exprStmtComparer struct{}

func (exprStmtComparer) Compare(a, b dst.Node, path string, exact bool, c *Comparator) bool {
	nodeA, nodeB := a.(*dst.ExprStmt), b.(*dst.ExprStmt)
	return c.Compare(nodeA.X, nodeB.X, path+".X", exact)
}

type ifStmtComparer struct{}

func (ifStmtComparer) Compare(a, b dst.Node, path string, exact bool, c *Comparator) bool {
	nodeA, nodeB := a.(*dst.IfStmt), b.(*dst.IfStmt)
	return c.Compare(nodeA.Init, nodeB.Init, path+".Init", exact) &&
		c.Compare(nodeA.Cond, nodeB.Cond, path+".Cond", exact) &&
		c.Compare(nodeA.Body, nodeB.Body, path+".Body", exact) &&
		c.Compare(nodeA.Else, nodeB.Else, path+".Else", exact)
}

type switchStmtComparer struct{}

func (switchStmtComparer) Compare(a, b dst.Node, path string, exact bool, c *Comparator) bool {
	nodeA, nodeB := a.(*dst.SwitchStmt), b.(*dst.SwitchStmt)
	return c.Compare(nodeA.Init, nodeB.Init, path+".Init", exact) &&
		c.Compare(nodeA.Tag, nodeB.Tag, path+".Tag", exact) &&
		c.Compare(nodeA.Body, nodeB.Body, path+".Body", exact)
}

type blockStmtComparer struct{}

func (blockStmtComparer) Compare(a, b dst.Node, path string, exact bool, c *Comparator) bool {
	nodeA, nodeB := a.(*dst.BlockStmt), b.(*dst.BlockStmt)
	if len(nodeA.List) != len(nodeB.List) {
		return false
	}
	for i := range nodeA.List {
		if !c.Compare(nodeA.List[i], nodeB.List[i], fmt.Sprintf("%s.List[%d]", path, i), exact) {
			return false
		}
	}
	return true
}

type assignStmtComparer struct{}

func (assignStmtComparer) Compare(a, b dst.Node, path string, exact bool, c *Comparator) bool {
	nodeA, nodeB := a.(*dst.AssignStmt), b.(*dst.AssignStmt)
	if nodeA.Tok != nodeB.Tok {
		return false
	}
	if len(nodeA.Lhs) != len(nodeB.Lhs) || len(nodeA.Rhs) != len(nodeB.Rhs) {
		return false
	}
	for i := range nodeA.Lhs {
		if !c.Compare(nodeA.Lhs[i], nodeB.Lhs[i], fmt.Sprintf("%s.Lhs[%d]", path, i), exact) {
			return false
		}
	}
	for i := range nodeA.Rhs {
		if !c.Compare(nodeA.Rhs[i], nodeB.Rhs[i], fmt.Sprintf("%s.Rhs[%d]", path, i), exact) {
			return false
		}
	}
	return true
}

type returnStmtComparer struct{}

func (returnStmtComparer) Compare(a, b dst.Node, path string, exact bool, c *Comparator) bool {
	nodeA, nodeB := a.(*dst.ReturnStmt), b.(*dst.ReturnStmt)
	if len(nodeA.Results) != len(nodeB.Results) {
		return false
	}
	for i := range nodeA.Results {
		if !c.Compare(nodeA.Results[i], nodeB.Results[i], fmt.Sprintf("%s.Results[%d]", path, i), exact) {
			return false
		}
	}
	return true
}

type caseClauseComparer struct{}

func (caseClauseComparer) Compare(a, b dst.Node, path string, exact bool, c *Comparator) bool {
	nodeA, nodeB := a.(*dst.CaseClause), b.(*dst.CaseClause)
	if len(nodeA.List) != len(nodeB.List) || len(nodeA.Body) != len(nodeB.Body) {
		return false
	}
	for i := range nodeA.List {
		if !c.Compare(nodeA.List[i], nodeB.List[i], fmt.Sprintf("%s.List[%d]", path, i), exact) {
			return false
		}
	}
	for i := range nodeA.Body {
		if !c.Compare(nodeA.Body[i], nodeB.Body[i], fmt.Sprintf("%s.Body[%d]", path, i), exact) {
			return false
		}
	}
	return true
}

// ============================================================================
// Expression Comparers
// ============================================================================

type callExprComparer struct{}

func (callExprComparer) Compare(a, b dst.Node, path string, exact bool, c *Comparator) bool {
	nodeA, nodeB := a.(*dst.CallExpr), b.(*dst.CallExpr)
	if !c.Compare(nodeA.Fun, nodeB.Fun, path+".Fun", exact) {
		return false
	}
	if len(nodeA.Args) != len(nodeB.Args) {
		return false
	}
	for i := range nodeA.Args {
		if !c.Compare(nodeA.Args[i], nodeB.Args[i], fmt.Sprintf("%s.Args[%d]", path, i), exact) {
			return false
		}
	}
	return true
}

type selectorExprComparer struct{}

func (selectorExprComparer) Compare(a, b dst.Node, path string, exact bool, c *Comparator) bool {
	nodeA, nodeB := a.(*dst.SelectorExpr), b.(*dst.SelectorExpr)
	if nodeA.Sel.Name != nodeB.Sel.Name {
		return false
	}
	return c.Compare(nodeA.X, nodeB.X, path+".X", exact)
}

type identComparer struct{}

func (identComparer) Compare(a, b dst.Node, _ string, _ bool, _ *Comparator) bool {
	nodeA, nodeB := a.(*dst.Ident), b.(*dst.Ident)
	return nodeA.Name == nodeB.Name
}

type basicLitComparer struct{}

func (basicLitComparer) Compare(a, b dst.Node, _ string, exact bool, _ *Comparator) bool {
	nodeA, nodeB := a.(*dst.BasicLit), b.(*dst.BasicLit)
	if nodeA.Kind != nodeB.Kind {
		return false
	}
	if exact && nodeA.Value != nodeB.Value {
		return false
	}
	return true
}

type unaryExprComparer struct{}

func (unaryExprComparer) Compare(a, b dst.Node, path string, exact bool, c *Comparator) bool {
	nodeA, nodeB := a.(*dst.UnaryExpr), b.(*dst.UnaryExpr)
	if nodeA.Op != nodeB.Op {
		return false
	}
	return c.Compare(nodeA.X, nodeB.X, path+".X", exact)
}

type binaryExprComparer struct{}

func (binaryExprComparer) Compare(a, b dst.Node, path string, exact bool, c *Comparator) bool {
	nodeA, nodeB := a.(*dst.BinaryExpr), b.(*dst.BinaryExpr)
	if nodeA.Op != nodeB.Op {
		return false
	}
	return c.Compare(nodeA.X, nodeB.X, path+".X", exact) &&
		c.Compare(nodeA.Y, nodeB.Y, path+".Y", exact)
}

type parenExprComparer struct{}

func (parenExprComparer) Compare(a, b dst.Node, path string, exact bool, c *Comparator) bool {
	nodeA, nodeB := a.(*dst.ParenExpr), b.(*dst.ParenExpr)
	return c.Compare(nodeA.X, nodeB.X, path+".X", exact)
}

type indexExprComparer struct{}

func (indexExprComparer) Compare(a, b dst.Node, path string, exact bool, c *Comparator) bool {
	nodeA, nodeB := a.(*dst.IndexExpr), b.(*dst.IndexExpr)
	return c.Compare(nodeA.X, nodeB.X, path+".X", exact) &&
		c.Compare(nodeA.Index, nodeB.Index, path+".Index", exact)
}

type funcLitComparer struct{}

func (funcLitComparer) Compare(a, b dst.Node, path string, exact bool, c *Comparator) bool {
	nodeA, nodeB := a.(*dst.FuncLit), b.(*dst.FuncLit)
	return c.Compare(nodeA.Type, nodeB.Type, path+".Type", exact) &&
		c.Compare(nodeA.Body, nodeB.Body, path+".Body", exact)
}

type funcTypeComparer struct{}

func (funcTypeComparer) Compare(a, b dst.Node, path string, exact bool, c *Comparator) bool {
	nodeA, nodeB := a.(*dst.FuncType), b.(*dst.FuncType)
	return compareFieldLists(nodeA.Params, nodeB.Params, path+".Params", exact, c) &&
		compareFieldLists(nodeA.Results, nodeB.Results, path+".Results", exact, c)
}

type compositeLitComparer struct{}

func (compositeLitComparer) Compare(a, b dst.Node, path string, exact bool, c *Comparator) bool {
	nodeA, nodeB := a.(*dst.CompositeLit), b.(*dst.CompositeLit)
	if !c.Compare(nodeA.Type, nodeB.Type, path+".Type", exact) {
		return false
	}
	if len(nodeA.Elts) != len(nodeB.Elts) {
		return false
	}
	for i := range nodeA.Elts {
		if !c.Compare(nodeA.Elts[i], nodeB.Elts[i], fmt.Sprintf("%s.Elts[%d]", path, i), exact) {
			return false
		}
	}
	return true
}

type keyValueExprComparer struct{}

func (keyValueExprComparer) Compare(a, b dst.Node, path string, exact bool, c *Comparator) bool {
	nodeA, nodeB := a.(*dst.KeyValueExpr), b.(*dst.KeyValueExpr)
	return c.Compare(nodeA.Key, nodeB.Key, path+".Key", exact) &&
		c.Compare(nodeA.Value, nodeB.Value, path+".Value", exact)
}

type starExprComparer struct{}

func (starExprComparer) Compare(a, b dst.Node, path string, exact bool, c *Comparator) bool {
	nodeA, nodeB := a.(*dst.StarExpr), b.(*dst.StarExpr)
	return c.Compare(nodeA.X, nodeB.X, path+".X", exact)
}

type typeAssertExprComparer struct{}

func (typeAssertExprComparer) Compare(a, b dst.Node, path string, exact bool, c *Comparator) bool {
	nodeA, nodeB := a.(*dst.TypeAssertExpr), b.(*dst.TypeAssertExpr)
	return c.Compare(nodeA.X, nodeB.X, path+".X", exact) &&
		c.Compare(nodeA.Type, nodeB.Type, path+".Type", exact)
}

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
