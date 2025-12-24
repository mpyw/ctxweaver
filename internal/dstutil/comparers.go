package dstutil

import (
	"fmt"

	"github.com/dave/dst"
)

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
