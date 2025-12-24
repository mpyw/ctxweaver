package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// CarrierDef defines a context carrier type.
type CarrierDef struct {
	Package  string `yaml:"package" json:"package"`
	Type     string `yaml:"type" json:"type"`
	Accessor string `yaml:"accessor" json:"accessor,omitempty"`
}

// BuildContextExpr builds the expression to access context.Context from a variable.
func (c CarrierDef) BuildContextExpr(varName string) string {
	return varName + c.Accessor
}

// CarriersFile represents the structure of carriers.yaml.
type CarriersFile struct {
	Carriers []CarrierDef `yaml:"carriers"`
}

// Hooks defines shell commands to run before and after processing.
type Hooks struct {
	// Pre are shell commands to run before processing
	Pre []string `yaml:"pre" json:"pre,omitempty"`
	// Post are shell commands to run after processing
	Post []string `yaml:"post" json:"post,omitempty"`
}

// Template can be an inline string or a reference to a file.
type Template struct {
	Inline string
	File   string
}

// UnmarshalYAML implements custom unmarshaling for Template.
// Accepts either a string (inline template) or an object with "file" field.
func (t *Template) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		// Simple string value
		t.Inline = value.Value
		return nil
	case yaml.MappingNode:
		// Object with "file" field
		var obj struct {
			File string `yaml:"file"`
		}
		if err := value.Decode(&obj); err != nil {
			return err // unreachable via LoadConfig: schema validation catches malformed objects first
		}
		t.File = obj.File
		return nil
	default:
		return fmt.Errorf("template must be a string or an object with 'file' field")
	}
}

// MarshalYAML implements custom marshaling for Template.
func (t Template) MarshalYAML() (any, error) {
	if t.File != "" {
		return map[string]string{"file": t.File}, nil
	}
	return t.Inline, nil
}

// Content returns the template content, loading from file if necessary.
func (t *Template) Content() (string, error) {
	if t.Inline != "" {
		return t.Inline, nil
	}
	if t.File != "" {
		data, err := os.ReadFile(t.File)
		if err != nil {
			return "", fmt.Errorf("failed to read template file: %w", err)
		}
		return string(data), nil
	}
	return "", fmt.Errorf("template is empty")
}

// Carriers can be a simple array of CarrierDef or an object with custom/default fields.
// Simple form: carriers: []
// Extended form: carriers: { custom: [], default: true }
type Carriers struct {
	// Custom are user-defined carrier definitions
	Custom []CarrierDef
	// Default indicates whether to include default carriers (default: true)
	Default *bool
}

// UseDefault returns whether default carriers should be used.
func (c *Carriers) UseDefault() bool {
	if c.Default == nil {
		return true // default is true
	}
	return *c.Default
}

// UnmarshalYAML implements custom unmarshaling for Carriers.
// Accepts either an array (simple form) or an object with "custom" and "default" fields.
func (c *Carriers) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.SequenceNode:
		// Simple array form: carriers: []
		var arr []CarrierDef
		if err := value.Decode(&arr); err != nil {
			return err // unreachable via LoadConfig: schema validation catches malformed arrays first
		}
		c.Custom = arr
		return nil
	case yaml.MappingNode:
		// Extended object form: carriers: { custom: [], default: true }
		var obj struct {
			Custom  []CarrierDef `yaml:"custom"`
			Default *bool        `yaml:"default"`
		}
		if err := value.Decode(&obj); err != nil {
			return err // unreachable via LoadConfig: schema validation catches malformed objects first
		}
		c.Custom = obj.Custom
		c.Default = obj.Default
		return nil
	default:
		return fmt.Errorf("carriers must be an array or an object with 'custom' and 'default' fields")
	}
}

// MarshalYAML implements custom marshaling for Carriers.
func (c Carriers) MarshalYAML() (any, error) {
	// If Default is explicitly set (not nil), use object form
	if c.Default != nil {
		return map[string]any{
			"custom":  c.Custom,
			"default": *c.Default,
		}, nil
	}
	// Otherwise use simple array form
	return c.Custom, nil
}

// Regexps defines regex patterns for filtering.
type Regexps struct {
	// Only includes items matching these patterns (if specified)
	Only []string `yaml:"only" json:"only,omitempty"`
	// Omit excludes items matching these patterns
	Omit []string `yaml:"omit" json:"omit,omitempty"`
}

// Packages defines package filtering options.
type Packages struct {
	// Patterns are the package patterns to process (e.g., "./...")
	Patterns []string `yaml:"patterns" json:"patterns"`
	// Regexps for filtering packages by import path
	Regexps Regexps `yaml:"regexps" json:"regexps,omitempty"`
}

// FuncType represents function type for filtering.
type FuncType string

const (
	FuncTypeFunction FuncType = "function"
	FuncTypeMethod   FuncType = "method"
)

// FuncScope represents function scope for filtering.
type FuncScope string

const (
	FuncScopeExported   FuncScope = "exported"
	FuncScopeUnexported FuncScope = "unexported"
)

// Functions defines function filtering options.
type Functions struct {
	// Types filters by function type (function, method). Default: both.
	Types []FuncType `yaml:"types" json:"types,omitempty"`
	// Scopes filters by visibility (exported, unexported). Default: both.
	Scopes []FuncScope `yaml:"scopes" json:"scopes,omitempty"`
	// Regexps for filtering functions by name
	Regexps Regexps `yaml:"regexps" json:"regexps,omitempty"`
}

// Config represents the user configuration file.
type Config struct {
	// Template is the Go template for the statement to insert
	Template Template `yaml:"template" json:"template"`
	// Imports are the imports to add when the template is inserted
	Imports []string `yaml:"imports" json:"imports,omitempty"`
	// Carriers defines context carrier configuration (custom carriers and default toggle)
	Carriers Carriers `yaml:"carriers" json:"carriers,omitempty"`
	// Packages defines package filtering options
	Packages Packages `yaml:"packages" json:"packages"`
	// Functions defines function filtering options
	Functions Functions `yaml:"functions" json:"functions,omitempty"`
	// Test indicates whether to process test files
	Test bool `yaml:"test" json:"test,omitempty"`
	// Hooks are shell commands to run before and after processing
	Hooks Hooks `yaml:"hooks" json:"hooks,omitempty"`
}

// SetDefaults sets default values for optional fields.
func (c *Config) SetDefaults() {
	// Set default function types (both function and method)
	if len(c.Functions.Types) == 0 {
		c.Functions.Types = []FuncType{FuncTypeFunction, FuncTypeMethod}
	}
	// Set default function scopes (both exported and unexported)
	if len(c.Functions.Scopes) == 0 {
		c.Functions.Scopes = []FuncScope{FuncScopeExported, FuncScopeUnexported}
	}
}
