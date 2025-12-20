// Package processor provides DST-based code transformation.
package processor

import (
	"fmt"
	"os"
	"regexp"

	"github.com/mpyw/ctxweaver/internal"
	"github.com/mpyw/ctxweaver/pkg/config"
	"github.com/mpyw/ctxweaver/pkg/template"
)

// CompiledRegexps holds compiled regex patterns for filtering.
type CompiledRegexps struct {
	Only []*regexp.Regexp
	Omit []*regexp.Regexp
}

// CompileRegexps compiles regex patterns from config.
func CompileRegexps(r config.Regexps) CompiledRegexps {
	var result CompiledRegexps
	for _, pattern := range r.Only {
		re, err := regexp.Compile(pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%swarning:%s invalid regex pattern %q: %v\n",
				internal.StderrColor(internal.ColorYellow),
				internal.StderrColor(internal.ColorReset),
				pattern, err)
			continue
		}
		result.Only = append(result.Only, re)
	}
	for _, pattern := range r.Omit {
		re, err := regexp.Compile(pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%swarning:%s invalid regex pattern %q: %v\n",
				internal.StderrColor(internal.ColorYellow),
				internal.StderrColor(internal.ColorReset),
				pattern, err)
			continue
		}
		result.Omit = append(result.Omit, re)
	}
	return result
}

// Match checks if a string matches the filter criteria.
// Returns true if the string should be included.
func (r *CompiledRegexps) Match(s string) bool {
	// If only patterns are specified, the string must match at least one
	if len(r.Only) > 0 {
		matched := false
		for _, re := range r.Only {
			if re.MatchString(s) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	// Check omit patterns - if any matches, exclude
	for _, re := range r.Omit {
		if re.MatchString(s) {
			return false
		}
	}
	return true
}

// FuncFilter holds compiled function filter settings.
type FuncFilter struct {
	Types   []config.FuncType
	Scopes  []config.FuncScope
	Regexps CompiledRegexps
}

// NewFuncFilter creates a FuncFilter from config.Functions.
func NewFuncFilter(f config.Functions) *FuncFilter {
	return &FuncFilter{
		Types:   f.Types,
		Scopes:  f.Scopes,
		Regexps: CompileRegexps(f.Regexps),
	}
}

// Match checks if a function should be processed.
func (f *FuncFilter) Match(funcName string, isMethod bool, isExported bool) bool {
	// Check types filter
	if len(f.Types) > 0 {
		var funcType config.FuncType
		if isMethod {
			funcType = config.FuncTypeMethod
		} else {
			funcType = config.FuncTypeFunction
		}
		matched := false
		for _, t := range f.Types {
			if t == funcType {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check scopes filter
	if len(f.Scopes) > 0 {
		var scope config.FuncScope
		if isExported {
			scope = config.FuncScopeExported
		} else {
			scope = config.FuncScopeUnexported
		}
		matched := false
		for _, s := range f.Scopes {
			if s == scope {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check regexps filter
	return f.Regexps.Match(funcName)
}

// Processor handles code transformation.
type Processor struct {
	registry   *config.CarrierRegistry
	tmpl       *template.Template
	imports    []string
	pkgRegexps CompiledRegexps // Regex patterns for package paths
	funcFilter *FuncFilter     // Function filter
	remove     bool            // Remove mode: remove generated statements instead of adding
	test       bool
	dryRun     bool
	verbose    bool
}

// Option configures a Processor.
type Option func(*Processor)

// WithTest enables processing of test files.
func WithTest(test bool) Option {
	return func(p *Processor) {
		p.test = test
	}
}

// WithDryRun enables dry run mode (no file writes).
func WithDryRun(dryRun bool) Option {
	return func(p *Processor) {
		p.dryRun = dryRun
	}
}

// WithVerbose enables verbose output.
func WithVerbose(verbose bool) Option {
	return func(p *Processor) {
		p.verbose = verbose
	}
}

// WithRemove enables remove mode (remove generated statements instead of adding).
func WithRemove(remove bool) Option {
	return func(p *Processor) {
		p.remove = remove
	}
}

// WithPackageRegexps sets regex patterns for filtering packages.
func WithPackageRegexps(r config.Regexps) Option {
	return func(p *Processor) {
		p.pkgRegexps = CompileRegexps(r)
	}
}

// WithFunctions sets function filtering options.
func WithFunctions(f config.Functions) Option {
	return func(p *Processor) {
		p.funcFilter = NewFuncFilter(f)
	}
}

// New creates a new Processor.
func New(registry *config.CarrierRegistry, tmpl *template.Template, importPaths []string, opts ...Option) *Processor {
	p := &Processor{
		registry: registry,
		tmpl:     tmpl,
		imports:  importPaths,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// ProcessResult holds the result of processing.
type ProcessResult struct {
	FilesProcessed int
	FilesModified  int
	Errors         []error
}
