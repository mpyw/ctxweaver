package processor

import (
	"fmt"

	"github.com/dave/dst"

	"github.com/mpyw/ctxweaver/internal/directive"
	"github.com/mpyw/ctxweaver/internal/dstutil"
)

// Action represents an operation to apply to a function body.
// Implementations encapsulate the logic for each action type.
type Action interface {
	// Apply executes the action on the function body.
	// Returns true if the body was modified.
	Apply(body *dst.BlockStmt, rendered string) bool
}

// skipAction represents no modification needed.
type skipAction struct{}

func (skipAction) Apply(_ *dst.BlockStmt, _ string) bool {
	return false
}

// insertAction represents inserting new statements at the beginning.
type insertAction struct{}

func (insertAction) Apply(body *dst.BlockStmt, rendered string) bool {
	return dstutil.InsertStatements(body, rendered)
}

// updateAction represents replacing existing statements.
type updateAction struct {
	index int
	count int
}

func (a updateAction) Apply(body *dst.BlockStmt, rendered string) bool {
	return dstutil.UpdateStatements(body, a.index, a.count, rendered)
}

// removeAction represents removing existing statements.
type removeAction struct {
	index int
	count int
}

func (a removeAction) Apply(body *dst.BlockStmt, _ string) bool {
	return dstutil.RemoveStatements(body, a.index, a.count)
}

// detectAction determines what action to take for a function body.
// Uses skeleton matching to compare AST structure. Supports multi-statement templates.
func (p *Processor) detectAction(body *dst.BlockStmt, renderedStmt string) (Action, error) {
	// Parse the rendered statements for skeleton comparison
	targetStmts, err := dstutil.ParseStatements(renderedStmt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rendered statement: %w", err)
	}
	if len(targetStmts) == 0 {
		return nil, fmt.Errorf("rendered statement is empty")
	}

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
			// Check if exact match (use skeleton match with exact mode)
			if !dstutil.MatchesExact(targetStmt, existingStmt) {
				allExact = false
			}
		}

		if allMatch {
			// Check if first statement has skip directive (manually added, should not be touched)
			if directive.HasStmtSkipDirective(body.List[i]) {
				return skipAction{}, nil
			}
			if p.remove {
				// In remove mode, remove all matching statements
				return removeAction{index: i, count: stmtCount}, nil
			}
			if allExact {
				return skipAction{}, nil
			}
			// Structure matches but content differs - needs update
			return updateAction{index: i, count: stmtCount}, nil
		}
	}

	// No matching statement found
	if p.remove {
		return skipAction{}, nil // Nothing to remove
	}
	return insertAction{}, nil
}
