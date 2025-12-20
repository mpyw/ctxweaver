package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewCarrierRegistry(t *testing.T) {
	r := NewCarrierRegistry()

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
	if cfg.Template != expectedTemplate {
		t.Errorf("Template = %q, want %q", cfg.Template, expectedTemplate)
	}

	// Check imports
	if len(cfg.Imports) != 1 || cfg.Imports[0] != "github.com/example/myapp/internal/apm" {
		t.Errorf("Imports = %v, want [github.com/example/myapp/internal/apm]", cfg.Imports)
	}

	// Check carriers
	if len(cfg.Carriers) != 1 {
		t.Errorf("Carriers count = %d, want 1", len(cfg.Carriers))
	} else {
		c := cfg.Carriers[0]
		if c.Package != "github.com/example/custom" || c.Type != "Context" || c.Accessor != ".GetContext()" {
			t.Errorf("Carrier = %+v, unexpected", c)
		}
	}

	// Check patterns
	if len(cfg.Patterns) != 1 || cfg.Patterns[0] != "./..." {
		t.Errorf("Patterns = %v, want [./...]", cfg.Patterns)
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
	configContent := "template_file: " + templatePath + "\n"
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Template != templateContent {
		t.Errorf("Template = %q, want %q", cfg.Template, templateContent)
	}
}

func TestCarrierRegistry_All(t *testing.T) {
	r := NewCarrierRegistry()

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
