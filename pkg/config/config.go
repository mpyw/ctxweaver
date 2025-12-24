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

// validateSchema validates data against the embedded JSON Schema.
func validateSchema(data any) error {
	return configSchema.Validate(data)
}
