// Command ctxweaver weaves statements into context-aware functions.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mpyw/ctxweaver/pkg/config"
	"github.com/mpyw/ctxweaver/pkg/processor"
	"github.com/mpyw/ctxweaver/pkg/template"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorCyan   = "\033[36m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorDim    = "\033[2m"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%sctxweaver: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
}

func run() error {
	var (
		configFile string
		dryRun     bool
		verbose    bool
		silent     bool
		test       bool
		remove     bool
		noHooks    bool
	)

	flag.StringVar(&configFile, "config", "ctxweaver.yaml", "path to configuration file")
	flag.BoolVar(&dryRun, "dry-run", false, "print changes without writing files")
	flag.BoolVar(&verbose, "verbose", false, "print processed files")
	flag.BoolVar(&silent, "silent", false, "suppress all output except errors")
	flag.BoolVar(&test, "test", false, "process test files")
	flag.BoolVar(&remove, "remove", false, "remove generated statements instead of adding them")
	flag.BoolVar(&noHooks, "no-hooks", false, "skip pre/post hooks")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override config with explicitly passed flags
	if isFlagPassed("test") {
		cfg.Test = test
	}

	// Get patterns from args or config
	patterns := flag.Args()
	if len(patterns) == 0 {
		patterns = cfg.Patterns
	}
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	// Validate
	if cfg.Template == "" {
		return fmt.Errorf("template is required in config file")
	}

	// Run pre hooks
	if !noHooks && len(cfg.Hooks.Pre) > 0 {
		if err := runHooks("pre", cfg.Hooks.Pre, silent); err != nil {
			return err
		}
	}

	// Parse template
	tmpl, err := template.Parse(cfg.Template)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create carrier registry
	registry, err := config.NewCarrierRegistry()
	if err != nil {
		return fmt.Errorf("failed to create carrier registry: %w", err)
	}

	// Register custom carriers from config
	for _, c := range cfg.Carriers {
		registry.Register(c)
	}

	// Create processor
	proc := processor.New(
		registry,
		tmpl,
		cfg.Imports,
		processor.WithTest(cfg.Test),
		processor.WithDryRun(dryRun),
		processor.WithVerbose(verbose && !silent),
		processor.WithRemove(remove),
	)

	// Print ctxweaver execution header
	if !silent {
		action := "weaving"
		if remove {
			action = "removing"
		}
		fmt.Printf("%s▶ ctxweaver%s %s%s %s%s\n", colorCyan, colorReset, colorDim, action, strings.Join(patterns, " "), colorReset)
	}

	// Process
	result, err := proc.Process(patterns)
	if err != nil {
		return err
	}

	// Report results
	if !silent && (verbose || dryRun) {
		fmt.Printf("  Files processed: %d\n", result.FilesProcessed)
		fmt.Printf("  Files modified: %d\n", result.FilesModified)
	} else if !silent {
		fmt.Printf("  %s✓%s %d files processed, %d modified\n", colorGreen, colorReset, result.FilesProcessed, result.FilesModified)
	}

	if len(result.Errors) > 0 {
		fmt.Fprintln(os.Stderr, "Errors:")
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  %v\n", e)
		}
		return fmt.Errorf("%d error(s) occurred", len(result.Errors))
	}

	// Run post hooks
	if !noHooks && len(cfg.Hooks.Post) > 0 {
		if err := runHooks("post", cfg.Hooks.Post, silent); err != nil {
			return err
		}
	}

	return nil
}

// runHooks executes a list of shell commands sequentially.
// If any command fails (non-zero exit code), execution stops and an error is returned.
func runHooks(phase string, commands []string, silent bool) error {
	if !silent {
		fmt.Printf("%s▶ %s%s\n", colorYellow, phase, colorReset)
	}

	for _, cmdStr := range commands {
		if !silent {
			fmt.Printf("  %s$ %s%s\n", colorDim, cmdStr, colorReset)
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
