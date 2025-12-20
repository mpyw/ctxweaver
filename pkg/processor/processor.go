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

// Processor handles code transformation.
type Processor struct {
	registry *config.CarrierRegistry
	tmpl     *template.Template
	imports  []string
	exclude  []*regexp.Regexp // Regex patterns for package paths to exclude
	remove   bool             // Remove mode: remove generated statements instead of adding
	test     bool
	dryRun   bool
	verbose  bool
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

// WithExcludeRegexps sets regex patterns for package paths to exclude.
// Each pattern is compiled as a regular expression.
// Invalid patterns are skipped with a warning to stderr.
func WithExcludeRegexps(patterns []string) Option {
	return func(p *Processor) {
		for _, pattern := range patterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%swarning:%s invalid exclude pattern %q: %v\n",
					internal.StderrColor(internal.ColorYellow),
					internal.StderrColor(internal.ColorReset),
					pattern, err)
				continue
			}
			p.exclude = append(p.exclude, re)
		}
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
