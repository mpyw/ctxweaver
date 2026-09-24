//declscope:core

// Package processor provides DST-based code transformation.
package processor

import (
	"fmt"
	"os"
	"regexp"
	"slices"

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
	return CompiledRegexps{
		Only: compilePatterns(r.Only),
		Omit: compilePatterns(r.Omit),
	}
}

// compilePatterns compiles each pattern, warning about and dropping invalid ones.
func compilePatterns(patterns []string) []*regexp.Regexp {
	var compiled []*regexp.Regexp
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%swarning:%s invalid regex pattern %q: %v\n",
				internal.StderrColor(internal.ColorYellow),
				internal.StderrColor(internal.ColorReset),
				pattern, err)
			continue
		}
		compiled = append(compiled, re)
	}
	return compiled
}

// Match checks if a string matches the filter criteria.
// Returns true if the string should be included.
func (r *CompiledRegexps) Match(s string) bool {
	matches := func(re *regexp.Regexp) bool { return re.MatchString(s) }
	// If only patterns are specified, the string must match at least one
	if len(r.Only) > 0 && !slices.ContainsFunc(r.Only, matches) {
		return false
	}
	// Any omit pattern that matches excludes the string
	return !slices.ContainsFunc(r.Omit, matches)
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
func (f *FuncFilter) Match(funcName string, isMethod, isExported bool) bool {
	// Check types filter
	funcType := config.FuncTypeFunction
	if isMethod {
		funcType = config.FuncTypeMethod
	}
	if len(f.Types) > 0 && !slices.Contains(f.Types, funcType) {
		return false
	}

	// Check scopes filter
	scope := config.FuncScopeUnexported
	if isExported {
		scope = config.FuncScopeExported
	}
	if len(f.Scopes) > 0 && !slices.Contains(f.Scopes, scope) {
		return false
	}

	// Check regexps filter
	return f.Regexps.Match(funcName)
}

// Processor handles code transformation.
//
// Its behavior is implemented across process.go, candidate.go and action.go,
// so the fields are shared with the whole package.
//
//declscope:package
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
	// Warnings are problems that do not fail the run, such as a malformed
	// directive. Each one starts with its file:line position.
	Warnings []string
}
