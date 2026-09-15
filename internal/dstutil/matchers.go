// The node matchers live apart from match.go only for length; they and the
// Matcher that dispatches to them form one unit.
//
//declscope:namespace match
package dstutil

import (
	"github.com/dave/dst"
)

// ============================================================================
// Statement Matchers
// ============================================================================

func matchDeferStmt(a, b *dst.DeferStmt, path string, exact bool, c *Matcher) bool {
	return c.Match(a.Call, b.Call, path+".Call", exact)
}

func matchExprStmt(a, b *dst.ExprStmt, path string, exact bool, c *Matcher) bool {
	return c.Match(a.X, b.X, path+".X", exact)
}

func matchIfStmt(a, b *dst.IfStmt, path string, exact bool, c *Matcher) bool {
	return c.Match(a.Init, b.Init, path+".Init", exact) &&
		c.Match(a.Cond, b.Cond, path+".Cond", exact) &&
		c.Match(a.Body, b.Body, path+".Body", exact) &&
		c.Match(a.Else, b.Else, path+".Else", exact)
}

func matchSwitchStmt(a, b *dst.SwitchStmt, path string, exact bool, c *Matcher) bool {
	return c.Match(a.Init, b.Init, path+".Init", exact) &&
		c.Match(a.Tag, b.Tag, path+".Tag", exact) &&
		c.Match(a.Body, b.Body, path+".Body", exact)
}

func matchBlockStmt(a, b *dst.BlockStmt, path string, exact bool, c *Matcher) bool {
	return matchNodeLists(a.List, b.List, path+".List", exact, c)
}

func matchAssignStmt(a, b *dst.AssignStmt, path string, exact bool, c *Matcher) bool {
	if a.Tok != b.Tok {
		return false
	}
	return matchNodeLists(a.Lhs, b.Lhs, path+".Lhs", exact, c) &&
		matchNodeLists(a.Rhs, b.Rhs, path+".Rhs", exact, c)
}

func matchReturnStmt(a, b *dst.ReturnStmt, path string, exact bool, c *Matcher) bool {
	return matchNodeLists(a.Results, b.Results, path+".Results", exact, c)
}

func matchCaseClause(a, b *dst.CaseClause, path string, exact bool, c *Matcher) bool {
	return matchNodeLists(a.List, b.List, path+".List", exact, c) &&
		matchNodeLists(a.Body, b.Body, path+".Body", exact, c)
}

// ============================================================================
// Expression Matchers
// ============================================================================

func matchCallExpr(a, b *dst.CallExpr, path string, exact bool, c *Matcher) bool {
	return c.Match(a.Fun, b.Fun, path+".Fun", exact) &&
		matchNodeLists(a.Args, b.Args, path+".Args", exact, c)
}

func matchSelectorExpr(a, b *dst.SelectorExpr, path string, exact bool, c *Matcher) bool {
	if a.Sel.Name != b.Sel.Name {
		return false
	}
	return c.Match(a.X, b.X, path+".X", exact)
}

func matchIdent(a, b *dst.Ident, _ string, _ bool, _ *Matcher) bool {
	return a.Name == b.Name
}

func matchBasicLit(a, b *dst.BasicLit, _ string, exact bool, _ *Matcher) bool {
	if a.Kind != b.Kind {
		return false
	}
	return !exact || a.Value == b.Value
}

func matchUnaryExpr(a, b *dst.UnaryExpr, path string, exact bool, c *Matcher) bool {
	if a.Op != b.Op {
		return false
	}
	return c.Match(a.X, b.X, path+".X", exact)
}

func matchBinaryExpr(a, b *dst.BinaryExpr, path string, exact bool, c *Matcher) bool {
	if a.Op != b.Op {
		return false
	}
	return c.Match(a.X, b.X, path+".X", exact) &&
		c.Match(a.Y, b.Y, path+".Y", exact)
}

func matchParenExpr(a, b *dst.ParenExpr, path string, exact bool, c *Matcher) bool {
	return c.Match(a.X, b.X, path+".X", exact)
}

func matchIndexExpr(a, b *dst.IndexExpr, path string, exact bool, c *Matcher) bool {
	return c.Match(a.X, b.X, path+".X", exact) &&
		c.Match(a.Index, b.Index, path+".Index", exact)
}

func matchFuncLit(a, b *dst.FuncLit, path string, exact bool, c *Matcher) bool {
	return c.Match(a.Type, b.Type, path+".Type", exact) &&
		c.Match(a.Body, b.Body, path+".Body", exact)
}

func matchFuncType(a, b *dst.FuncType, path string, exact bool, c *Matcher) bool {
	return matchFieldLists(a.Params, b.Params, path+".Params", exact, c) &&
		matchFieldLists(a.Results, b.Results, path+".Results", exact, c)
}

func matchCompositeLit(a, b *dst.CompositeLit, path string, exact bool, c *Matcher) bool {
	return c.Match(a.Type, b.Type, path+".Type", exact) &&
		matchNodeLists(a.Elts, b.Elts, path+".Elts", exact, c)
}

func matchKeyValueExpr(a, b *dst.KeyValueExpr, path string, exact bool, c *Matcher) bool {
	return c.Match(a.Key, b.Key, path+".Key", exact) &&
		c.Match(a.Value, b.Value, path+".Value", exact)
}

func matchStarExpr(a, b *dst.StarExpr, path string, exact bool, c *Matcher) bool {
	return c.Match(a.X, b.X, path+".X", exact)
}

func matchTypeAssertExpr(a, b *dst.TypeAssertExpr, path string, exact bool, c *Matcher) bool {
	return c.Match(a.X, b.X, path+".X", exact) &&
		c.Match(a.Type, b.Type, path+".Type", exact)
}
