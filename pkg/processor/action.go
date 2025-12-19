package processor

import (
	"github.com/dave/dst"

	"github.com/mpyw/ctxweaver/internal/dstutil"
)

// actionType represents the action to take on a function.
type actionType int

const (
	actionSkip actionType = iota
	actionInsert
	actionUpdate
	actionRemove
)

// action represents the action to take and related info.
type action struct {
	actionType actionType
	index      int // For actionUpdate/actionRemove, the starting index
	count      int // Number of statements to update/remove
}

// detectAction determines what action to take for a function body.
// Uses skeleton matching to compare AST structure. Supports multi-statement templates.
func (p *Processor) detectAction(body *dst.BlockStmt, renderedStmt string) action {
	// Parse the rendered statements for skeleton comparison
	targetStmts, err := dstutil.ParseStatements(renderedStmt)
	if err != nil || len(targetStmts) == 0 {
		// If we can't parse the rendered statement, fall back to insert (or skip for remove)
		if p.remove {
			return action{actionType: actionSkip}
		}
		return action{actionType: actionInsert}
	}

	// Format the target statements for consistent comparison
	targetStrs := dstutil.StmtsToStrings(targetStmts)
	stmtCount := len(targetStmts)

	for i := range body.List {
		// Check if we have enough statements remaining to match
		if i+stmtCount > len(body.List) {
			break
		}

		// Try to match all target statements starting at this index
		allMatch := true
		allExact := true
		for j, targetStmt := range targetStmts {
			existingStmt := body.List[i+j]
			if !dstutil.MatchesSkeleton(targetStmt, existingStmt) {
				allMatch = false
				break
			}
			// Check if exact match
			if dstutil.StmtToString(existingStmt) != targetStrs[j] {
				allExact = false
			}
		}

		if allMatch {
			// Check if first statement has skip directive (manually added, should not be touched)
			if hasStmtSkipDirective(body.List[i]) {
				return action{actionType: actionSkip, index: i, count: stmtCount}
			}
			if p.remove {
				// In remove mode, remove all matching statements
				return action{actionType: actionRemove, index: i, count: stmtCount}
			}
			if allExact {
				return action{actionType: actionSkip, index: i, count: stmtCount}
			}
			// Structure matches but content differs - needs update
			return action{actionType: actionUpdate, index: i, count: stmtCount}
		}
	}

	// No matching statement found
	if p.remove {
		return action{actionType: actionSkip} // Nothing to remove
	}
	return action{actionType: actionInsert}
}
