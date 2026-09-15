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

//go:embed schema.json
var configSchemaJSON []byte

// Compiled at init time - failure here means corrupted embedded files.
var configSchema *jsonschema.Schema

func init() {
	// Parse and compile embedded schema.json
	schemaDoc := internal.Must(jsonschema.UnmarshalJSON(bytes.NewReader(configSchemaJSON)))
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
	if err := configSchema.Validate(raw); err != nil {
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
