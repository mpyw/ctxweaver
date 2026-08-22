package dstutil

import (
	"github.com/dave/dst"
)

// ============================================================================
// Statement Comparers
// ============================================================================

func compareDeferStmt(a, b *dst.DeferStmt, path string, exact bool, c *Comparator) bool {
	return c.Compare(a.Call, b.Call, path+".Call", exact)
}

func compareExprStmt(a, b *dst.ExprStmt, path string, exact bool, c *Comparator) bool {
	return c.Compare(a.X, b.X, path+".X", exact)
}

func compareIfStmt(a, b *dst.IfStmt, path string, exact bool, c *Comparator) bool {
	return c.Compare(a.Init, b.Init, path+".Init", exact) &&
		c.Compare(a.Cond, b.Cond, path+".Cond", exact) &&
		c.Compare(a.Body, b.Body, path+".Body", exact) &&
		c.Compare(a.Else, b.Else, path+".Else", exact)
}

func compareSwitchStmt(a, b *dst.SwitchStmt, path string, exact bool, c *Comparator) bool {
	return c.Compare(a.Init, b.Init, path+".Init", exact) &&
		c.Compare(a.Tag, b.Tag, path+".Tag", exact) &&
		c.Compare(a.Body, b.Body, path+".Body", exact)
}

func compareBlockStmt(a, b *dst.BlockStmt, path string, exact bool, c *Comparator) bool {
	return compareNodeLists(a.List, b.List, path+".List", exact, c)
}

func compareAssignStmt(a, b *dst.AssignStmt, path string, exact bool, c *Comparator) bool {
	if a.Tok != b.Tok {
		return false
	}
	return compareNodeLists(a.Lhs, b.Lhs, path+".Lhs", exact, c) &&
		compareNodeLists(a.Rhs, b.Rhs, path+".Rhs", exact, c)
}

func compareReturnStmt(a, b *dst.ReturnStmt, path string, exact bool, c *Comparator) bool {
	return compareNodeLists(a.Results, b.Results, path+".Results", exact, c)
}

func compareCaseClause(a, b *dst.CaseClause, path string, exact bool, c *Comparator) bool {
	return compareNodeLists(a.List, b.List, path+".List", exact, c) &&
		compareNodeLists(a.Body, b.Body, path+".Body", exact, c)
}

// ============================================================================
// Expression Comparers
// ============================================================================

func compareCallExpr(a, b *dst.CallExpr, path string, exact bool, c *Comparator) bool {
	return c.Compare(a.Fun, b.Fun, path+".Fun", exact) &&
		compareNodeLists(a.Args, b.Args, path+".Args", exact, c)
}

func compareSelectorExpr(a, b *dst.SelectorExpr, path string, exact bool, c *Comparator) bool {
	if a.Sel.Name != b.Sel.Name {
		return false
	}
	return c.Compare(a.X, b.X, path+".X", exact)
}

func compareIdent(a, b *dst.Ident, _ string, _ bool, _ *Comparator) bool {
	return a.Name == b.Name
}

func compareBasicLit(a, b *dst.BasicLit, _ string, exact bool, _ *Comparator) bool {
	if a.Kind != b.Kind {
		return false
	}
	return !exact || a.Value == b.Value
}

func compareUnaryExpr(a, b *dst.UnaryExpr, path string, exact bool, c *Comparator) bool {
	if a.Op != b.Op {
		return false
	}
	return c.Compare(a.X, b.X, path+".X", exact)
}

func compareBinaryExpr(a, b *dst.BinaryExpr, path string, exact bool, c *Comparator) bool {
	if a.Op != b.Op {
		return false
	}
	return c.Compare(a.X, b.X, path+".X", exact) &&
		c.Compare(a.Y, b.Y, path+".Y", exact)
}

func compareParenExpr(a, b *dst.ParenExpr, path string, exact bool, c *Comparator) bool {
	return c.Compare(a.X, b.X, path+".X", exact)
}

func compareIndexExpr(a, b *dst.IndexExpr, path string, exact bool, c *Comparator) bool {
	return c.Compare(a.X, b.X, path+".X", exact) &&
		c.Compare(a.Index, b.Index, path+".Index", exact)
}

func compareFuncLit(a, b *dst.FuncLit, path string, exact bool, c *Comparator) bool {
	return c.Compare(a.Type, b.Type, path+".Type", exact) &&
		c.Compare(a.Body, b.Body, path+".Body", exact)
}

func compareFuncType(a, b *dst.FuncType, path string, exact bool, c *Comparator) bool {
	return compareFieldLists(a.Params, b.Params, path+".Params", exact, c) &&
		compareFieldLists(a.Results, b.Results, path+".Results", exact, c)
}

func compareCompositeLit(a, b *dst.CompositeLit, path string, exact bool, c *Comparator) bool {
	return c.Compare(a.Type, b.Type, path+".Type", exact) &&
		compareNodeLists(a.Elts, b.Elts, path+".Elts", exact, c)
}

func compareKeyValueExpr(a, b *dst.KeyValueExpr, path string, exact bool, c *Comparator) bool {
	return c.Compare(a.Key, b.Key, path+".Key", exact) &&
		c.Compare(a.Value, b.Value, path+".Value", exact)
}

func compareStarExpr(a, b *dst.StarExpr, path string, exact bool, c *Comparator) bool {
	return c.Compare(a.X, b.X, path+".X", exact)
}

func compareTypeAssertExpr(a, b *dst.TypeAssertExpr, path string, exact bool, c *Comparator) bool {
	return c.Compare(a.X, b.X, path+".X", exact) &&
		c.Compare(a.Type, b.Type, path+".Type", exact)
}
