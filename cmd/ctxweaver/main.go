// Command ctxweaver weaves statements into context-aware functions.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mpyw/ctxweaver/config"
	"github.com/mpyw/ctxweaver/processor"
	"github.com/mpyw/ctxweaver/template"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ctxweaver: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		configFile string
		dryRun     bool
		verbose    bool
		test       bool
	)

	flag.StringVar(&configFile, "config", "ctxweaver.yaml", "path to configuration file")
	flag.BoolVar(&dryRun, "dry-run", false, "print changes without writing files")
	flag.BoolVar(&verbose, "verbose", false, "print processed files")
	flag.BoolVar(&test, "test", false, "process test files")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override config with flags
	if test {
		cfg.Test = true
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
		processor.WithVerbose(verbose),
	)

	// Process
	result, err := proc.Process(patterns)
	if err != nil {
		return err
	}

	// Report results
	if verbose || dryRun {
		fmt.Printf("Files processed: %d\n", result.FilesProcessed)
		fmt.Printf("Files modified: %d\n", result.FilesModified)
	}

	if len(result.Errors) > 0 {
		fmt.Fprintln(os.Stderr, "Errors:")
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  %v\n", e)
		}
	}

	return nil
}

// stringSliceFlag implements flag.Value for a slice of strings.
type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}
