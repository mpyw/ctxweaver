package processor

import (
	"fmt"

	"github.com/dave/dst"

	"github.com/mpyw/ctxweaver/internal/directive"
	"github.com/mpyw/ctxweaver/pkg/carrier"
	"github.com/mpyw/ctxweaver/pkg/config"
	"github.com/mpyw/ctxweaver/pkg/template"
)

// funcCandidate represents a validated function that has a context carrier.
// This struct captures the validated state after filtering, eliminating
// the need to re-validate in subsequent processing steps.
type funcCandidate struct {
	decl    *dst.FuncDecl
	carrier config.CarrierDef
	varName string
}

func extractFirstParam(decl *dst.FuncDecl) *dst.Field {
	if decl.Type == nil || decl.Type.Params == nil || len(decl.Type.Params.List) == 0 {
		return nil
	}
	return decl.Type.Params.List[0]
}

// isExportedFunc checks if a function name is exported (starts with uppercase).
// The empty name check is defensive: Go parser rejects functions without names,
// so this branch is unreachable in normal operation.
func isExportedFunc(name string) bool {
	if name == "" {
		return false
	}
	r := rune(name[0])
	return r >= 'A' && r <= 'Z'
}

// shouldSkipDecl checks if a function declaration should be skipped.
func shouldSkipDecl(decl *dst.FuncDecl) bool {
	if directive.HasSkipDirective(decl.Decorations()) {
		return true
	}
	if decl.Body == nil {
		return true
	}
	return false
}

// matchesFuncFilter checks if a function matches the configured filter.
func (p *Processor) matchesFuncFilter(decl *dst.FuncDecl) bool {
	if p.funcFilter == nil {
		return true
	}
	isMethod := decl.Recv != nil && len(decl.Recv.List) > 0
	isExported := isExportedFunc(decl.Name.Name)
	return p.funcFilter.Match(decl.Name.Name, isMethod, isExported)
}

// tryMatchCarrier attempts to match the first parameter against registered carriers.
// Returns nil if no match is found.
func (p *Processor) tryMatchCarrier(decl *dst.FuncDecl) *funcCandidate {
	param := extractFirstParam(decl)
	if param == nil {
		return nil
	}

	carrierDef, varName, ok := carrier.Match(param, p.registry)
	if !ok {
		return nil
	}

	return &funcCandidate{
		decl:    decl,
		carrier: carrierDef,
		varName: varName,
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

		if shouldSkipDecl(decl) {
			return true
		}

		if !p.matchesFuncFilter(decl) {
			return true
		}

		if c := p.tryMatchCarrier(decl); c != nil {
			candidates = append(candidates, *c)
		}

		return true
	})

	return candidates
}

// processCandidate processes a single function candidate:
// renders the template, detects the required action, and applies it.
func (p *Processor) processCandidate(c funcCandidate, df *dst.File, pkgPath string) (bool, error) {
	vars := template.BuildVars(df, c.decl, pkgPath, c.carrier, c.varName)

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

// processFunctions processes functions in the DST file.
// Relies on dst.Ident.Path set by NewDecoratorFromPackage for import resolution.
func (p *Processor) processFunctions(df *dst.File, pkgPath string) (bool, error) {
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
