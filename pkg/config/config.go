// Package config provides configuration loading for ctxweaver.
package config

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"

	"github.com/mpyw/ctxweaver/internal"
)

//go:embed carriers.yaml
var defaultCarriersYAML []byte

//go:embed schema.json
var schemaJSON []byte

// Parsed at init time - failure here means corrupted embedded files.
var (
	defaultCarriers []CarrierDef
	configSchema    *jsonschema.Schema
)

func init() {
	// Parse embedded carriers.yaml
	var carriersFile CarriersFile
	defaultCarriers = internal.Must(carriersFile, yaml.Unmarshal(defaultCarriersYAML, &carriersFile)).Carriers

	// Parse and compile embedded schema.json
	schemaDoc := internal.Must(jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON)))
	compiler := jsonschema.NewCompiler()
	internal.Must(struct{}{}, compiler.AddResource("schema.json", schemaDoc))
	configSchema = internal.Must(compiler.Compile("schema.json"))
}

// CarrierDef defines a context carrier type.
type CarrierDef struct {
	Package  string `yaml:"package" json:"package"`
	Type     string `yaml:"type" json:"type"`
	Accessor string `yaml:"accessor" json:"accessor,omitempty"`
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

// CarrierRegistry holds all registered carriers for quick lookup.
type CarrierRegistry struct {
	carriers map[string]CarrierDef // key: "package.Type"
}

// NewCarrierRegistry creates a registry, optionally loading default carriers.
func NewCarrierRegistry(includeDefaults bool) *CarrierRegistry {
	r := &CarrierRegistry{
		carriers: make(map[string]CarrierDef),
	}
	if includeDefaults {
		for _, c := range defaultCarriers {
			r.Register(c)
		}
	}
	return r
}

// Register adds a carrier to the registry.
func (r *CarrierRegistry) Register(c CarrierDef) {
	key := c.Package + "." + c.Type
	r.carriers[key] = c
}

// Lookup finds a carrier by package path and type name.
func (r *CarrierRegistry) Lookup(packagePath, typeName string) (CarrierDef, bool) {
	key := packagePath + "." + typeName
	c, ok := r.carriers[key]
	return c, ok
}

// All returns all registered carriers.
func (r *CarrierRegistry) All() []CarrierDef {
	result := make([]CarrierDef, 0, len(r.carriers))
	for _, c := range r.carriers {
		result = append(result, c)
	}
	return result
}

// LoadConfig loads a configuration file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML to generic interface for schema validation
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate against JSON Schema
	if err := validateSchema(raw); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Unmarshal directly into struct
	// This error is unreachable in normal flow: if schema validation passes,
	// struct unmarshaling should succeed. Only reachable if schema and struct diverge.
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set defaults
	cfg.SetDefaults()

	return &cfg, nil
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

// validateSchema validates data against the embedded JSON Schema.
func validateSchema(data any) error {
	return configSchema.Validate(data)
}

// BuildContextExpr builds the expression to access context.Context from a variable.
func (c CarrierDef) BuildContextExpr(varName string) string {
	return varName + c.Accessor
}
