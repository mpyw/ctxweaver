package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewCarrierRegistry(t *testing.T) {
	r, err := NewCarrierRegistry()
	if err != nil {
		t.Fatalf("NewCarrierRegistry() error = %v", err)
	}

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
	r, err := NewCarrierRegistry()
	if err != nil {
		t.Fatalf("NewCarrierRegistry() error = %v", err)
	}

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
