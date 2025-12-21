// Command ctxweaver weaves statements into context-aware functions.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mpyw/ctxweaver/internal"
	"github.com/mpyw/ctxweaver/pkg/config"
	"github.com/mpyw/ctxweaver/pkg/processor"
	"github.com/mpyw/ctxweaver/pkg/template"
)

// Color helper functions for stdout and stderr
func co(color string) string { return internal.StdoutColor(color) }
func ce(color string) string { return internal.StderrColor(color) }

// options holds the parsed command-line flags.
type options struct {
	configFile string
	dryRun     bool
	verbose    bool
	silent     bool
	test       bool
	remove     bool
	noHooks    bool
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%sctxweaver: %v%s\n", ce(internal.ColorRed), err, ce(internal.ColorReset))
		os.Exit(1)
	}
}

// parseFlags parses command-line flags and returns the options.
func parseFlags() *options {
	opts := &options{}
	flag.StringVar(&opts.configFile, "config", "ctxweaver.yaml", "path to configuration file")
	flag.BoolVar(&opts.dryRun, "dry-run", false, "print changes without writing files")
	flag.BoolVar(&opts.verbose, "verbose", false, "print processed files")
	flag.BoolVar(&opts.silent, "silent", false, "suppress all output except errors")
	flag.BoolVar(&opts.test, "test", false, "process test files")
	flag.BoolVar(&opts.remove, "remove", false, "remove generated statements instead of adding them")
	flag.BoolVar(&opts.noHooks, "no-hooks", false, "skip pre/post hooks")
	flag.Parse()
	return opts
}

// getPatterns returns the package patterns from CLI args or config.
func getPatterns(cfg *config.Config) ([]string, error) {
	patterns := flag.Args()
	if len(patterns) == 0 {
		patterns = cfg.Packages.Patterns
	}
	if len(patterns) == 0 {
		return nil, fmt.Errorf("no patterns specified: use command line args or packages.patterns in config")
	}
	return patterns, nil
}

// createProcessor creates a new processor with the given configuration.
func createProcessor(cfg *config.Config, tmpl *template.Template, opts *options) *processor.Processor {
	registry := config.NewCarrierRegistry(cfg.Carriers.UseDefault())
	for _, c := range cfg.Carriers.Custom {
		registry.Register(c)
	}
	return processor.New(
		registry,
		tmpl,
		cfg.Imports,
		processor.WithTest(cfg.Test),
		processor.WithDryRun(opts.dryRun),
		processor.WithVerbose(opts.verbose && !opts.silent),
		processor.WithRemove(opts.remove),
		processor.WithPackageRegexps(cfg.Packages.Regexps),
		processor.WithFunctions(cfg.Functions),
	)
}

// printHeader prints the ctxweaver execution header.
func printHeader(patterns []string, remove, silent bool) {
	if silent {
		return
	}
	action := "weaving"
	if remove {
		action = "removing"
	}
	fmt.Printf("%s▶ ctxweaver%s %s%s %s%s\n", co(internal.ColorCyan), co(internal.ColorReset), co(internal.ColorDim), action, strings.Join(patterns, " "), co(internal.ColorReset))
}

// reportResults prints the processing results and returns an error if there were any.
func reportResults(result *processor.ProcessResult, verbose, dryRun, silent bool) error {
	if !silent {
		if verbose || dryRun {
			fmt.Printf("  Files processed: %d\n", result.FilesProcessed)
			fmt.Printf("  Files modified: %d\n", result.FilesModified)
		} else {
			fmt.Printf("  %s✓%s %d files processed, %d modified\n", co(internal.ColorGreen), co(internal.ColorReset), result.FilesProcessed, result.FilesModified)
		}
	}
	if len(result.Errors) > 0 {
		fmt.Fprintln(os.Stderr, "Errors:")
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  %v\n", e)
		}
		return fmt.Errorf("%d error(s) occurred", len(result.Errors))
	}
	return nil
}

func run() error {
	opts := parseFlags()

	cfg, err := config.LoadConfig(opts.configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if isFlagPassed("test") {
		cfg.Test = opts.test
	}

	patterns, err := getPatterns(cfg)
	if err != nil {
		return err
	}

	tmplContent, err := cfg.Template.Content()
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	if !opts.noHooks && len(cfg.Hooks.Pre) > 0 {
		if err := runHooks("pre", cfg.Hooks.Pre, opts.silent); err != nil {
			return err
		}
	}

	tmpl, err := template.Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	proc := createProcessor(cfg, tmpl, opts)
	printHeader(patterns, opts.remove, opts.silent)

	result, err := proc.Process(patterns)
	if err != nil {
		return err
	}

	if err := reportResults(result, opts.verbose, opts.dryRun, opts.silent); err != nil {
		return err
	}

	if !opts.noHooks && len(cfg.Hooks.Post) > 0 {
		if err := runHooks("post", cfg.Hooks.Post, opts.silent); err != nil {
			return err
		}
	}

	return nil
}

// runHooks executes a list of shell commands sequentially.
// If any command fails (non-zero exit code), execution stops and an error is returned.
func runHooks(phase string, commands []string, silent bool) error {
	if !silent {
		fmt.Printf("%s▶ %s%s\n", co(internal.ColorYellow), phase, co(internal.ColorReset))
	}

	for _, cmdStr := range commands {
		if !silent {
			fmt.Printf("  %s$ %s%s\n", co(internal.ColorDim), cmdStr, co(internal.ColorReset))
		}

		cmd := exec.Command("sh", "-c", cmdStr)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s hook failed: %s: %w", phase, cmdStr, err)
		}
	}

	return nil
}

// isFlagPassed checks if a flag was explicitly passed on the command line.
func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
