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

// Config represents the user configuration file.
type Config struct {
	// Template is the Go template for the statement to insert
	Template string `yaml:"template" json:"template,omitempty"`
	// TemplateFile is a path to a file containing the template (alternative to Template)
	TemplateFile string `yaml:"template_file" json:"template_file,omitempty"`
	// Imports are the imports to add when the template is inserted
	Imports []string `yaml:"imports" json:"imports,omitempty"`
	// Carriers are additional context carrier definitions
	Carriers []CarrierDef `yaml:"carriers" json:"carriers,omitempty"`
	// Patterns are the package patterns to process (e.g., "./...")
	Patterns []string `yaml:"patterns" json:"patterns,omitempty"`
	// Test indicates whether to process test files
	Test bool `yaml:"test" json:"test,omitempty"`
	// Hooks are shell commands to run before and after processing
	Hooks Hooks `yaml:"hooks" json:"hooks,omitempty"`
}

// CarrierRegistry holds all registered carriers for quick lookup.
type CarrierRegistry struct {
	carriers map[string]CarrierDef // key: "package.Type"
}

// NewCarrierRegistry creates a registry with default carriers loaded.
func NewCarrierRegistry() *CarrierRegistry {
	r := &CarrierRegistry{
		carriers: make(map[string]CarrierDef),
	}
	for _, c := range defaultCarriers {
		r.Register(c)
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
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Load template from file if specified
	if cfg.TemplateFile != "" && cfg.Template == "" {
		tmplData, err := os.ReadFile(cfg.TemplateFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read template file: %w", err)
		}
		cfg.Template = string(tmplData)
	}

	return &cfg, nil
}

// validateSchema validates data against the embedded JSON Schema.
func validateSchema(data any) error {
	return configSchema.Validate(data)
}

// BuildContextExpr builds the expression to access context.Context from a variable.
func (c CarrierDef) BuildContextExpr(varName string) string {
	return varName + c.Accessor
}
