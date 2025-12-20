package carrier

import (
	"testing"

	"github.com/dave/dst"

	"github.com/mpyw/ctxweaver/pkg/config"
)

func TestMatch(t *testing.T) {
	// Create a test registry with known carriers
	registry := config.NewCarrierRegistry()
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
		wantOK      bool
	}{
		"empty names": {
			param: &dst.Field{
				Names: []*dst.Ident{},
				Type:  &dst.Ident{Name: "Request", Path: "net/http"},
			},
			wantOK: false,
		},
		"underscore name": {
			param: &dst.Field{
				Names: []*dst.Ident{{Name: "_"}},
				Type:  &dst.Ident{Name: "Request", Path: "net/http"},
			},
			wantOK: false,
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
			wantOK:      true,
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
			wantOK:      true,
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
			wantOK:      true,
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
			wantOK:      true,
		},
		"ident without path": {
			param: &dst.Field{
				Names: []*dst.Ident{{Name: "r"}},
				Type:  &dst.Ident{Name: "Request"},
			},
			wantOK: false,
		},
		"selector expr without path": {
			param: &dst.Field{
				Names: []*dst.Ident{{Name: "r"}},
				Type: &dst.SelectorExpr{
					X:   &dst.Ident{Name: "http"},
					Sel: &dst.Ident{Name: "Request"},
				},
			},
			wantOK: false,
		},
		"carrier not in registry": {
			param: &dst.Field{
				Names: []*dst.Ident{{Name: "w"}},
				Type:  &dst.Ident{Name: "ResponseWriter", Path: "net/http"},
			},
			wantOK: false,
		},
		"unsupported type - func type": {
			param: &dst.Field{
				Names: []*dst.Ident{{Name: "f"}},
				Type:  &dst.FuncType{},
			},
			wantOK: false,
		},
		"unsupported type - array type": {
			param: &dst.Field{
				Names: []*dst.Ident{{Name: "arr"}},
				Type:  &dst.ArrayType{Elt: &dst.Ident{Name: "int"}},
			},
			wantOK: false,
		},
		"selector expr with non-ident X": {
			param: &dst.Field{
				Names: []*dst.Ident{{Name: "x"}},
				Type: &dst.SelectorExpr{
					X:   &dst.CallExpr{Fun: &dst.Ident{Name: "getType"}},
					Sel: &dst.Ident{Name: "Request"},
				},
			},
			wantOK: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotCarrier, gotVarName, gotOK := Match(tt.param, registry)

			if gotOK != tt.wantOK {
				t.Errorf("Match() ok = %v, want %v", gotOK, tt.wantOK)
				return
			}

			if !tt.wantOK {
				return
			}

			if gotVarName != tt.wantVarName {
				t.Errorf("Match() varName = %q, want %q", gotVarName, tt.wantVarName)
			}

			if gotCarrier.Package != tt.wantCarrier.Package {
				t.Errorf("Match() carrier.Package = %q, want %q", gotCarrier.Package, tt.wantCarrier.Package)
			}
			if gotCarrier.Type != tt.wantCarrier.Type {
				t.Errorf("Match() carrier.Type = %q, want %q", gotCarrier.Type, tt.wantCarrier.Type)
			}
			if gotCarrier.Accessor != tt.wantCarrier.Accessor {
				t.Errorf("Match() carrier.Accessor = %q, want %q", gotCarrier.Accessor, tt.wantCarrier.Accessor)
			}
		})
	}
}
