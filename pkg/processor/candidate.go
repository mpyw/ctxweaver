package processor

import (
	"fmt"
	"go/token"

	"github.com/dave/dst"

	"github.com/mpyw/ctxweaver/internal/directive"
	"github.com/mpyw/ctxweaver/pkg/carrier"
	"github.com/mpyw/ctxweaver/pkg/template"
)

// funcCandidate represents a validated function that has a context carrier.
// This struct captures the validated state after filtering, eliminating
// the need to re-validate in subsequent processing steps.
type funcCandidate struct {
	decl  *dst.FuncDecl
	match *carrier.MatchResult
}

// shouldSkipCandidate checks if a function declaration should be skipped.
func shouldSkipCandidate(decl *dst.FuncDecl) bool {
	if directive.HasSkipDirective(decl.Decorations()) {
		return true
	}
	if decl.Body == nil {
		return true
	}
	return false
}

// candidateMatchesFilter checks if a function matches the configured filter.
func (p *Processor) candidateMatchesFilter(decl *dst.FuncDecl) bool {
	if p.funcFilter == nil {
		return true
	}
	isMethod := decl.Recv != nil && len(decl.Recv.List) > 0
	// token.IsExported handles non-ASCII uppercase letters, unlike a byte-wise check.
	return p.funcFilter.Match(decl.Name.Name, isMethod, token.IsExported(decl.Name.Name))
}

// candidateFor attempts to match the first parameter of decl against
// registered carriers. Returns nil if no match is found.
func (p *Processor) candidateFor(decl *dst.FuncDecl) *funcCandidate {
	if decl.Type == nil || decl.Type.Params == nil || len(decl.Type.Params.List) == 0 {
		return nil
	}

	result := carrier.Match(decl.Type.Params.List[0], p.registry)
	if result == nil {
		return nil
	}

	return &funcCandidate{
		decl:  decl,
		match: result,
	}
}

// collectCandidates traverses the DST file and collects all function candidates
// that have a context carrier and pass the configured filters.
func (p *Processor) collectCandidates(df *dst.File) []funcCandidate {
	var candidates []funcCandidate

	dst.Inspect(df, func(n dst.Node) bool {
		decl, ok := n.(*dst.FuncDecl)
		if !ok {
			return true
		}

		if shouldSkipCandidate(decl) {
			return true
		}

		if !p.candidateMatchesFilter(decl) {
			return true
		}

		if c := p.candidateFor(decl); c != nil {
			candidates = append(candidates, *c)
		}

		return true
	})

	return candidates
}

// processCandidate processes a single function candidate:
// renders the template, detects the required action, and applies it.
func (p *Processor) processCandidate(c funcCandidate, df *dst.File, pkgPath string) (bool, error) {
	vars := template.BuildVars(df, c.decl, pkgPath, c.match.Carrier, c.match.VarName)

	rendered, err := p.tmpl.Render(vars)
	if err != nil {
		return false, fmt.Errorf("function %s: %w", c.decl.Name.Name, err)
	}

	action, err := p.detectAction(c.decl.Body, rendered)
	if err != nil {
		return false, fmt.Errorf("function %s: %w", c.decl.Name.Name, err)
	}

	return action.Apply(c.decl.Body, rendered), nil
}

// processCandidates collects and processes the candidate functions in the DST
// file. Relies on dst.Ident.Path set by NewDecoratorFromPackage for import
// resolution.
//
// This is the entry point of the candidate pipeline: process.go hands each
// decorated file here.
//
//declscope:package
func (p *Processor) processCandidates(df *dst.File, pkgPath string) (bool, error) {
	candidates := p.collectCandidates(df)

	var modified bool
	for _, c := range candidates {
		m, err := p.processCandidate(c, df, pkgPath)
		if err != nil {
			return false, err
		}
		modified = modified || m
	}

	return modified, nil
}
