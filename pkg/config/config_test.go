package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestNewCarrierRegistry(t *testing.T) {
	r := NewCarrierRegistry(true)

	// Check default carriers are loaded
	tests := map[string]struct {
		pkg      string
		typ      string
		accessor string
	}{
		"context.Context": {
			pkg:      "context",
			typ:      "Context",
			accessor: "",
		},
		"http.Request": {
			pkg:      "net/http",
			typ:      "Request",
			accessor: ".Context()",
		},
		"echo.Context": {
			pkg:      "github.com/labstack/echo/v4",
			typ:      "Context",
			accessor: ".Request().Context()",
		},
		"cli.Context": {
			pkg:      "github.com/urfave/cli/v2",
			typ:      "Context",
			accessor: ".Context",
		},
		"cobra.Command": {
			pkg:      "github.com/spf13/cobra",
			typ:      "Command",
			accessor: ".Context()",
		},
		"gin.Context": {
			pkg:      "github.com/gin-gonic/gin",
			typ:      "Context",
			accessor: ".Request.Context()",
		},
		"fiber.Ctx": {
			pkg:      "github.com/gofiber/fiber/v2",
			typ:      "Ctx",
			accessor: ".Context()",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			c, ok := r.Lookup(tt.pkg, tt.typ)
			if !ok {
				t.Errorf("Lookup(%q, %q) not found", tt.pkg, tt.typ)
				return
			}
			if c.Accessor != tt.accessor {
				t.Errorf("Accessor = %q, want %q", c.Accessor, tt.accessor)
			}
		})
	}
}

func TestCarrierDef_BuildContextExpr(t *testing.T) {
	tests := map[string]struct {
		carrier CarrierDef
		varName string
		want    string
	}{
		"context.Context": {
			carrier: CarrierDef{Accessor: ""},
			varName: "ctx",
			want:    "ctx",
		},
		"echo.Context": {
			carrier: CarrierDef{Accessor: ".Request().Context()"},
			varName: "c",
			want:    "c.Request().Context()",
		},
		"cli.Context": {
			carrier: CarrierDef{Accessor: ".Context"},
			varName: "cliCtx",
			want:    "cliCtx.Context",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.carrier.BuildContextExpr(tt.varName)
			if got != tt.want {
				t.Errorf("BuildContextExpr() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ctxweaver.yaml")

	configContent := `template: |
  defer apm.StartSegment({{.Ctx}}, {{.FuncName | quote}}).End()

imports:
  - github.com/example/myapp/internal/apm

carriers:
  - package: github.com/example/custom
    type: Context
    accessor: .GetContext()

packages:
  patterns:
    - ./...

test: true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Check template
	expectedTemplate := "defer apm.StartSegment({{.Ctx}}, {{.FuncName | quote}}).End()\n"
	tmplContent, err := cfg.Template.Content()
	if err != nil {
		t.Fatalf("Template.Content() error = %v", err)
	}
	if tmplContent != expectedTemplate {
		t.Errorf("Template = %q, want %q", tmplContent, expectedTemplate)
	}

	// Check imports
	if len(cfg.Imports) != 1 || cfg.Imports[0] != "github.com/example/myapp/internal/apm" {
		t.Errorf("Imports = %v, want [github.com/example/myapp/internal/apm]", cfg.Imports)
	}

	// Check carriers
	if len(cfg.Carriers.Custom) != 1 {
		t.Errorf("Carriers.Custom count = %d, want 1", len(cfg.Carriers.Custom))
	} else {
		c := cfg.Carriers.Custom[0]
		if c.Package != "github.com/example/custom" || c.Type != "Context" || c.Accessor != ".GetContext()" {
			t.Errorf("Carrier = %+v, unexpected", c)
		}
	}
	if !cfg.Carriers.UseDefault() {
		t.Error("Carriers.UseDefault() should be true by default")
	}

	// Check patterns
	if len(cfg.Packages.Patterns) != 1 || cfg.Packages.Patterns[0] != "./..." {
		t.Errorf("Packages.Patterns = %v, want [./...]", cfg.Packages.Patterns)
	}

	// Check test flag
	if !cfg.Test {
		t.Error("Test should be true")
	}
}

func TestLoadConfig_WithTemplateFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create template file
	templatePath := filepath.Join(tmpDir, "template.txt")
	templateContent := "defer trace({{.Ctx}})"
	if err := os.WriteFile(templatePath, []byte(templateContent), 0o644); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}

	// Create config file
	configPath := filepath.Join(tmpDir, "ctxweaver.yaml")
	configContent := `template:
  file: ` + templatePath + `
packages:
  patterns:
    - ./...
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	tmplContent, err := cfg.Template.Content()
	if err != nil {
		t.Fatalf("Template.Content() error = %v", err)
	}
	if tmplContent != templateContent {
		t.Errorf("Template = %q, want %q", tmplContent, templateContent)
	}
}

func TestCarrierRegistry_All(t *testing.T) {
	r := NewCarrierRegistry(true)

	all := r.All()
	if len(all) == 0 {
		t.Error("All() returned empty slice")
	}

	// Should have at least context.Context
	found := false
	for _, c := range all {
		if c.Package == "context" && c.Type == "Context" {
			found = true
			break
		}
	}
	if !found {
		t.Error("context.Context not found in All()")
	}
}

func TestLoadConfig_UnknownField(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ctxweaver.yaml")

	configContent := `template: "defer trace({{.Ctx}})"
packages:
  patterns:
    - ./...
unknown_field: "should cause error"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("expected error for unknown field")
	}
	if !strings.Contains(err.Error(), "additional properties") {
		t.Errorf("error should mention 'additional properties', got: %v", err)
	}
}

func TestLoadConfig_InvalidCarrier_MissingPackage(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ctxweaver.yaml")

	configContent := `template: "defer trace({{.Ctx}})"
packages:
  patterns:
    - ./...
carriers:
  - type: Context
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("expected error for carrier missing package")
	}
	if !strings.Contains(err.Error(), "package") {
		t.Errorf("error should mention 'package', got: %v", err)
	}
}

func TestLoadConfig_InvalidCarrier_MissingType(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ctxweaver.yaml")

	configContent := `template: "defer trace({{.Ctx}})"
packages:
  patterns:
    - ./...
carriers:
  - package: github.com/example/pkg
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("expected error for carrier missing type")
	}
	if !strings.Contains(err.Error(), "type") {
		t.Errorf("error should mention 'type', got: %v", err)
	}
}

func TestLoadConfig_InvalidCarrier_UnknownField(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ctxweaver.yaml")

	configContent := `template: "defer trace({{.Ctx}})"
packages:
  patterns:
    - ./...
carriers:
  - package: github.com/example/pkg
    type: Context
    unknown: value
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("expected error for carrier with unknown field")
	}
}

func TestLoadConfig_InvalidHooks_UnknownField(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ctxweaver.yaml")

	configContent := `template: "defer trace({{.Ctx}})"
packages:
  patterns:
    - ./...
hooks:
  pre:
    - echo hello
  unknown: value
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("expected error for hooks with unknown field")
	}
}

func TestLoadConfig_WrongType(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ctxweaver.yaml")

	configContent := `template: 123
packages:
  patterns:
    - ./...
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("expected error for wrong type")
	}
}

func TestLoadConfig_NonExistentFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ctxweaver.yaml")

	// Invalid YAML syntax (bad indentation, unclosed quote)
	configContent := `template: "defer trace({{.Ctx}})
  imports:
- broken
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("expected error for invalid YAML syntax")
	}
}

func TestLoadConfig_WithHooks(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ctxweaver.yaml")

	configContent := `template: "defer trace({{.Ctx}})"
packages:
  patterns:
    - ./...
hooks:
  pre:
    - go mod tidy
  post:
    - gofmt -w .
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(cfg.Hooks.Pre) != 1 || cfg.Hooks.Pre[0] != "go mod tidy" {
		t.Errorf("Hooks.Pre = %v, want [go mod tidy]", cfg.Hooks.Pre)
	}
	if len(cfg.Hooks.Post) != 1 || cfg.Hooks.Post[0] != "gofmt -w ." {
		t.Errorf("Hooks.Post = %v, want [gofmt -w .]", cfg.Hooks.Post)
	}
}

func TestLoadConfig_WithPackageRegexps(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ctxweaver.yaml")

	configContent := `template: "defer trace({{.Ctx}})"
packages:
  patterns:
    - ./...
  regexps:
    only:
      - "^github\\.com/myorg"
    omit:
      - "_test$"
      - "/internal/"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(cfg.Packages.Regexps.Only) != 1 {
		t.Errorf("Packages.Regexps.Only = %v, want 1 element", cfg.Packages.Regexps.Only)
	}
	if len(cfg.Packages.Regexps.Omit) != 2 {
		t.Errorf("Packages.Regexps.Omit = %v, want 2 elements", cfg.Packages.Regexps.Omit)
	}
}

func TestLoadConfig_WithFunctions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ctxweaver.yaml")

	configContent := `template: "defer trace({{.Ctx}})"
packages:
  patterns:
    - ./...
functions:
  types:
    - function
    - method
  scopes:
    - exported
  regexps:
    only:
      - "^Handle"
    omit:
      - "Mock$"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(cfg.Functions.Types) != 2 {
		t.Errorf("Functions.Types = %v, want 2 elements", cfg.Functions.Types)
	}
	if len(cfg.Functions.Scopes) != 1 || cfg.Functions.Scopes[0] != FuncScopeExported {
		t.Errorf("Functions.Scopes = %v, want [exported]", cfg.Functions.Scopes)
	}
	if len(cfg.Functions.Regexps.Only) != 1 {
		t.Errorf("Functions.Regexps.Only = %v, want 1 element", cfg.Functions.Regexps.Only)
	}
	if len(cfg.Functions.Regexps.Omit) != 1 {
		t.Errorf("Functions.Regexps.Omit = %v, want 1 element", cfg.Functions.Regexps.Omit)
	}
}

func TestTemplate_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name       string
		yaml       string
		wantInline string
		wantFile   string
		wantErr    bool
	}{
		{
			name:       "inline string",
			yaml:       `template: "defer trace({{.Ctx}})"`,
			wantInline: "defer trace({{.Ctx}})",
		},
		{
			name:     "file reference",
			yaml:     `template: {file: "./template.txt"}`,
			wantFile: "./template.txt",
		},
		{
			name: "multiline inline",
			yaml: `template: |
  defer trace({{.Ctx}})`,
			wantInline: "defer trace({{.Ctx}})\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "ctxweaver.yaml")

			configContent := tt.yaml + `
packages:
  patterns:
    - ./...
`
			if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			cfg, err := LoadConfig(configPath)
			if (err != nil) != tt.wantErr {
				t.Fatalf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if cfg.Template.Inline != tt.wantInline {
				t.Errorf("Template.Inline = %q, want %q", cfg.Template.Inline, tt.wantInline)
			}
			if cfg.Template.File != tt.wantFile {
				t.Errorf("Template.File = %q, want %q", cfg.Template.File, tt.wantFile)
			}
		})
	}
}

func TestTemplate_UnmarshalYAML_InvalidType(t *testing.T) {
	// Test that template as array/other type returns error via schema validation
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ctxweaver.yaml")

	// Template as array (invalid)
	configContent := `template:
  - item1
  - item2
packages:
  patterns:
    - ./...
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("expected error for template as array")
	}
	// Schema validation catches this before UnmarshalYAML
	if !strings.Contains(err.Error(), "template") {
		t.Errorf("error should mention 'template', got: %v", err)
	}
}

func TestTemplate_UnmarshalYAML_DirectInvalidType(t *testing.T) {
	// Directly test UnmarshalYAML with invalid node types (bypassing schema validation)
	// This tests the library API when used directly without LoadConfig
	var tmpl Template

	// Test with sequence node (array)
	seqNode := &yaml.Node{
		Kind: yaml.SequenceNode,
	}
	err := tmpl.UnmarshalYAML(seqNode)
	if err == nil {
		t.Error("expected error for sequence node")
	}
	if !strings.Contains(err.Error(), "template must be a string or an object with 'file' field") {
		t.Errorf("error should mention expected format, got: %v", err)
	}
}

func TestTemplate_Content_FileReadFailure(t *testing.T) {
	tmpl := Template{
		File: "/nonexistent/path/template.txt",
	}

	_, err := tmpl.Content()
	if err == nil {
		t.Error("expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "failed to read template file") {
		t.Errorf("error should mention 'failed to read template file', got: %v", err)
	}
}

func TestTemplate_Content_EmptyTemplate(t *testing.T) {
	tmpl := Template{}

	_, err := tmpl.Content()
	if err == nil {
		t.Error("expected error for empty template")
	}
	if !strings.Contains(err.Error(), "template is empty") {
		t.Errorf("error should mention 'template is empty', got: %v", err)
	}
}

func TestTemplate_MarshalYAML(t *testing.T) {
	t.Run("marshal inline template", func(t *testing.T) {
		tmpl := Template{Inline: "defer trace({{.Ctx}})"}
		result, err := tmpl.MarshalYAML()
		if err != nil {
			t.Fatalf("MarshalYAML() error = %v", err)
		}
		if result != "defer trace({{.Ctx}})" {
			t.Errorf("MarshalYAML() = %v, want inline string", result)
		}
	})

	t.Run("marshal file reference", func(t *testing.T) {
		tmpl := Template{File: "./template.txt"}
		result, err := tmpl.MarshalYAML()
		if err != nil {
			t.Fatalf("MarshalYAML() error = %v", err)
		}
		mapResult, ok := result.(map[string]string)
		if !ok {
			t.Errorf("MarshalYAML() = %T, want map[string]string", result)
		}
		if mapResult["file"] != "./template.txt" {
			t.Errorf("MarshalYAML()[file] = %v, want ./template.txt", mapResult["file"])
		}
	})
}

func TestCarriers_UnmarshalYAML(t *testing.T) {
	t.Run("simple array form", func(t *testing.T) {
		yamlContent := `
carriers:
  - package: github.com/example/custom
    type: Context
    accessor: .GetContext()
`
		var cfg struct {
			Carriers Carriers `yaml:"carriers"`
		}
		if err := yaml.Unmarshal([]byte(yamlContent), &cfg); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if len(cfg.Carriers.Custom) != 1 {
			t.Errorf("Custom count = %d, want 1", len(cfg.Carriers.Custom))
		}
		if !cfg.Carriers.UseDefault() {
			t.Error("UseDefault() should be true for simple form")
		}
	})

	t.Run("extended form with default true", func(t *testing.T) {
		yamlContent := `
carriers:
  custom:
    - package: github.com/example/custom
      type: Context
  default: true
`
		var cfg struct {
			Carriers Carriers `yaml:"carriers"`
		}
		if err := yaml.Unmarshal([]byte(yamlContent), &cfg); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if len(cfg.Carriers.Custom) != 1 {
			t.Errorf("Custom count = %d, want 1", len(cfg.Carriers.Custom))
		}
		if !cfg.Carriers.UseDefault() {
			t.Error("UseDefault() should be true")
		}
	})

	t.Run("extended form with default false", func(t *testing.T) {
		yamlContent := `
carriers:
  custom:
    - package: github.com/example/custom
      type: Context
  default: false
`
		var cfg struct {
			Carriers Carriers `yaml:"carriers"`
		}
		if err := yaml.Unmarshal([]byte(yamlContent), &cfg); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if len(cfg.Carriers.Custom) != 1 {
			t.Errorf("Custom count = %d, want 1", len(cfg.Carriers.Custom))
		}
		if cfg.Carriers.UseDefault() {
			t.Error("UseDefault() should be false")
		}
	})

	t.Run("extended form with empty custom", func(t *testing.T) {
		yamlContent := `
carriers:
  default: false
`
		var cfg struct {
			Carriers Carriers `yaml:"carriers"`
		}
		if err := yaml.Unmarshal([]byte(yamlContent), &cfg); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if len(cfg.Carriers.Custom) != 0 {
			t.Errorf("Custom count = %d, want 0", len(cfg.Carriers.Custom))
		}
		if cfg.Carriers.UseDefault() {
			t.Error("UseDefault() should be false")
		}
	})
}

func TestCarriers_UnmarshalYAML_DirectInvalidType(t *testing.T) {
	var carriers Carriers
	scalarNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "invalid"}
	err := carriers.UnmarshalYAML(scalarNode)
	if err == nil {
		t.Error("expected error for scalar node")
	}
	if err.Error() != "carriers must be an array or an object with 'custom' and 'default' fields" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCarriers_MarshalYAML(t *testing.T) {
	t.Run("marshal simple form", func(t *testing.T) {
		carriers := Carriers{
			Custom: []CarrierDef{{Package: "pkg", Type: "T"}},
		}
		result, err := carriers.MarshalYAML()
		if err != nil {
			t.Fatalf("MarshalYAML() error = %v", err)
		}
		arr, ok := result.([]CarrierDef)
		if !ok {
			t.Errorf("MarshalYAML() = %T, want []CarrierDef", result)
		}
		if len(arr) != 1 {
			t.Errorf("MarshalYAML() len = %d, want 1", len(arr))
		}
	})

	t.Run("marshal extended form with default set", func(t *testing.T) {
		defaultVal := false
		carriers := Carriers{
			Custom:  []CarrierDef{{Package: "pkg", Type: "T"}},
			Default: &defaultVal,
		}
		result, err := carriers.MarshalYAML()
		if err != nil {
			t.Fatalf("MarshalYAML() error = %v", err)
		}
		mapResult, ok := result.(map[string]any)
		if !ok {
			t.Errorf("MarshalYAML() = %T, want map[string]any", result)
		}
		if mapResult["default"] != false {
			t.Errorf("MarshalYAML()[default] = %v, want false", mapResult["default"])
		}
	})
}

func TestLoadConfig_CarriersExtendedForm(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ctxweaver.yaml")

	configContent := `template: "defer trace({{.Ctx}})"
packages:
  patterns:
    - ./...
carriers:
  custom:
    - package: github.com/example/custom
      type: Context
  default: false
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(cfg.Carriers.Custom) != 1 {
		t.Errorf("Carriers.Custom count = %d, want 1", len(cfg.Carriers.Custom))
	}
	if cfg.Carriers.UseDefault() {
		t.Error("Carriers.UseDefault() should be false")
	}
}

func TestLoadConfig_DefaultValues(t *testing.T) {
	t.Run("sets default function types when empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ctxweaver.yaml")

		// Config without functions section
		configContent := `template: "defer trace({{.Ctx}})"
packages:
  patterns:
    - ./...
`
		if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		// Should have default types: function and method
		if len(cfg.Functions.Types) != 2 {
			t.Errorf("Functions.Types = %v, want 2 elements", cfg.Functions.Types)
		}
		hasFunction := false
		hasMethod := false
		for _, ft := range cfg.Functions.Types {
			if ft == FuncTypeFunction {
				hasFunction = true
			}
			if ft == FuncTypeMethod {
				hasMethod = true
			}
		}
		if !hasFunction || !hasMethod {
			t.Errorf("Functions.Types should contain both 'function' and 'method', got %v", cfg.Functions.Types)
		}
	})

	t.Run("sets default function scopes when empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ctxweaver.yaml")

		// Config without functions section
		configContent := `template: "defer trace({{.Ctx}})"
packages:
  patterns:
    - ./...
`
		if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		// Should have default scopes: exported and unexported
		if len(cfg.Functions.Scopes) != 2 {
			t.Errorf("Functions.Scopes = %v, want 2 elements", cfg.Functions.Scopes)
		}
		hasExported := false
		hasUnexported := false
		for _, fs := range cfg.Functions.Scopes {
			if fs == FuncScopeExported {
				hasExported = true
			}
			if fs == FuncScopeUnexported {
				hasUnexported = true
			}
		}
		if !hasExported || !hasUnexported {
			t.Errorf("Functions.Scopes should contain both 'exported' and 'unexported', got %v", cfg.Functions.Scopes)
		}
	})

	t.Run("preserves explicit types when specified", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ctxweaver.yaml")

		configContent := `template: "defer trace({{.Ctx}})"
packages:
  patterns:
    - ./...
functions:
  types:
    - function
`
		if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		// Should only have function type
		if len(cfg.Functions.Types) != 1 {
			t.Errorf("Functions.Types = %v, want 1 element", cfg.Functions.Types)
		}
		if cfg.Functions.Types[0] != FuncTypeFunction {
			t.Errorf("Functions.Types[0] = %v, want 'function'", cfg.Functions.Types[0])
		}
	})

	t.Run("preserves explicit scopes when specified", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ctxweaver.yaml")

		configContent := `template: "defer trace({{.Ctx}})"
packages:
  patterns:
    - ./...
functions:
  scopes:
    - exported
`
		if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		// Should only have exported scope
		if len(cfg.Functions.Scopes) != 1 {
			t.Errorf("Functions.Scopes = %v, want 1 element", cfg.Functions.Scopes)
		}
		if cfg.Functions.Scopes[0] != FuncScopeExported {
			t.Errorf("Functions.Scopes[0] = %v, want 'exported'", cfg.Functions.Scopes[0])
		}
	})
}
