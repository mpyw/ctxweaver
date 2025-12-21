package carrier

import (
	"testing"

	"github.com/dave/dst"

	"github.com/mpyw/ctxweaver/pkg/config"
)

func TestMatch(t *testing.T) {
	// Create a test registry with known carriers
	registry := config.NewCarrierRegistry(true)
	registry.Register(config.CarrierDef{
		Package:  "net/http",
		Type:     "Request",
		Accessor: ".Context()",
	})
	registry.Register(config.CarrierDef{
		Package:  "github.com/labstack/echo/v4",
		Type:     "Context",
		Accessor: ".Request().Context()",
	})

	tests := map[string]struct {
		param       *dst.Field
		wantCarrier config.CarrierDef
		wantVarName string
		wantMatch   bool
	}{
		"empty names": {
			param: &dst.Field{
				Names: []*dst.Ident{},
				Type:  &dst.Ident{Name: "Request", Path: "net/http"},
			},
			wantMatch: false,
		},
		"underscore name": {
			param: &dst.Field{
				Names: []*dst.Ident{{Name: "_"}},
				Type:  &dst.Ident{Name: "Request", Path: "net/http"},
			},
			wantMatch: false,
		},
		"ident with path": {
			param: &dst.Field{
				Names: []*dst.Ident{{Name: "r"}},
				Type:  &dst.Ident{Name: "Request", Path: "net/http"},
			},
			wantCarrier: config.CarrierDef{
				Package:  "net/http",
				Type:     "Request",
				Accessor: ".Context()",
			},
			wantVarName: "r",
			wantMatch:   true,
		},
		"pointer to ident with path": {
			param: &dst.Field{
				Names: []*dst.Ident{{Name: "req"}},
				Type: &dst.StarExpr{
					X: &dst.Ident{Name: "Request", Path: "net/http"},
				},
			},
			wantCarrier: config.CarrierDef{
				Package:  "net/http",
				Type:     "Request",
				Accessor: ".Context()",
			},
			wantVarName: "req",
			wantMatch:   true,
		},
		"selector expr with path": {
			param: &dst.Field{
				Names: []*dst.Ident{{Name: "c"}},
				Type: &dst.SelectorExpr{
					X:   &dst.Ident{Name: "echo", Path: "github.com/labstack/echo/v4"},
					Sel: &dst.Ident{Name: "Context"},
				},
			},
			wantCarrier: config.CarrierDef{
				Package:  "github.com/labstack/echo/v4",
				Type:     "Context",
				Accessor: ".Request().Context()",
			},
			wantVarName: "c",
			wantMatch:   true,
		},
		"pointer to selector expr with path": {
			param: &dst.Field{
				Names: []*dst.Ident{{Name: "ctx"}},
				Type: &dst.StarExpr{
					X: &dst.SelectorExpr{
						X:   &dst.Ident{Name: "http", Path: "net/http"},
						Sel: &dst.Ident{Name: "Request"},
					},
				},
			},
			wantCarrier: config.CarrierDef{
				Package:  "net/http",
				Type:     "Request",
				Accessor: ".Context()",
			},
			wantVarName: "ctx",
			wantMatch:   true,
		},
		"ident without path": {
			param: &dst.Field{
				Names: []*dst.Ident{{Name: "r"}},
				Type:  &dst.Ident{Name: "Request"},
			},
			wantMatch: false,
		},
		"selector expr without path": {
			param: &dst.Field{
				Names: []*dst.Ident{{Name: "r"}},
				Type: &dst.SelectorExpr{
					X:   &dst.Ident{Name: "http"},
					Sel: &dst.Ident{Name: "Request"},
				},
			},
			wantMatch: false,
		},
		"carrier not in registry": {
			param: &dst.Field{
				Names: []*dst.Ident{{Name: "w"}},
				Type:  &dst.Ident{Name: "ResponseWriter", Path: "net/http"},
			},
			wantMatch: false,
		},
		"unsupported type - func type": {
			param: &dst.Field{
				Names: []*dst.Ident{{Name: "f"}},
				Type:  &dst.FuncType{},
			},
			wantMatch: false,
		},
		"unsupported type - array type": {
			param: &dst.Field{
				Names: []*dst.Ident{{Name: "arr"}},
				Type:  &dst.ArrayType{Elt: &dst.Ident{Name: "int"}},
			},
			wantMatch: false,
		},
		"selector expr with non-ident X": {
			param: &dst.Field{
				Names: []*dst.Ident{{Name: "x"}},
				Type: &dst.SelectorExpr{
					X:   &dst.CallExpr{Fun: &dst.Ident{Name: "getType"}},
					Sel: &dst.Ident{Name: "Request"},
				},
			},
			wantMatch: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := Match(tt.param, registry)

			gotMatch := result != nil
			if gotMatch != tt.wantMatch {
				t.Errorf("Match() returned %v, want match=%v", result, tt.wantMatch)
				return
			}

			if !tt.wantMatch {
				return
			}

			if result.VarName != tt.wantVarName {
				t.Errorf("Match() VarName = %q, want %q", result.VarName, tt.wantVarName)
			}

			if result.Carrier.Package != tt.wantCarrier.Package {
				t.Errorf("Match() Carrier.Package = %q, want %q", result.Carrier.Package, tt.wantCarrier.Package)
			}
			if result.Carrier.Type != tt.wantCarrier.Type {
				t.Errorf("Match() Carrier.Type = %q, want %q", result.Carrier.Type, tt.wantCarrier.Type)
			}
			if result.Carrier.Accessor != tt.wantCarrier.Accessor {
				t.Errorf("Match() Carrier.Accessor = %q, want %q", result.Carrier.Accessor, tt.wantCarrier.Accessor)
			}
		})
	}
}
