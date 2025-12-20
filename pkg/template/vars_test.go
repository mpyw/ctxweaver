package template

import (
	"testing"

	"github.com/dave/dst"
)

func TestExtractReceiverTypeName(t *testing.T) {
	tests := map[string]struct {
		expr        dst.Expr
		wantName    string
		wantGeneric bool
	}{
		"simple ident": {
			expr:        &dst.Ident{Name: "Service"},
			wantName:    "Service",
			wantGeneric: false,
		},
		"pointer to ident": {
			expr:        &dst.StarExpr{X: &dst.Ident{Name: "Service"}},
			wantName:    "Service",
			wantGeneric: false,
		},
		"generic with single type param": {
			expr: &dst.IndexExpr{
				X:     &dst.Ident{Name: "Container"},
				Index: &dst.Ident{Name: "T"},
			},
			wantName:    "Container",
			wantGeneric: true,
		},
		"pointer to generic with single type param": {
			expr: &dst.StarExpr{
				X: &dst.IndexExpr{
					X:     &dst.Ident{Name: "Container"},
					Index: &dst.Ident{Name: "T"},
				},
			},
			wantName:    "Container",
			wantGeneric: true,
		},
		"generic with multiple type params": {
			expr: &dst.IndexListExpr{
				X: &dst.Ident{Name: "Cache"},
				Indices: []dst.Expr{
					&dst.Ident{Name: "K"},
					&dst.Ident{Name: "V"},
				},
			},
			wantName:    "Cache",
			wantGeneric: true,
		},
		"pointer to generic with multiple type params": {
			expr: &dst.StarExpr{
				X: &dst.IndexListExpr{
					X: &dst.Ident{Name: "Cache"},
					Indices: []dst.Expr{
						&dst.Ident{Name: "K"},
						&dst.Ident{Name: "V"},
					},
				},
			},
			wantName:    "Cache",
			wantGeneric: true,
		},
		"nested generic single param": {
			// T[X[Y]] - outer generic wrapping inner generic
			expr: &dst.IndexExpr{
				X: &dst.IndexExpr{
					X:     &dst.Ident{Name: "Outer"},
					Index: &dst.Ident{Name: "Inner"},
				},
				Index: &dst.Ident{Name: "Y"},
			},
			wantName:    "Outer",
			wantGeneric: true,
		},
		"generic with nested type in indices": {
			// T[X, Y[Z]] - IndexListExpr with nested generic in Indices
			expr: &dst.IndexListExpr{
				X: &dst.Ident{Name: "Outer"},
				Indices: []dst.Expr{
					&dst.Ident{Name: "X"},
					&dst.IndexExpr{
						X:     &dst.Ident{Name: "Y"},
						Index: &dst.Ident{Name: "Z"},
					},
				},
			},
			wantName:    "Outer",
			wantGeneric: true,
		},
		"unknown expr type": {
			expr:        &dst.ArrayType{Elt: &dst.Ident{Name: "int"}},
			wantName:    "",
			wantGeneric: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotName, gotGeneric := extractReceiverTypeName(tt.expr)
			if gotName != tt.wantName {
				t.Errorf("extractReceiverTypeName() name = %q, want %q", gotName, tt.wantName)
			}
			if gotGeneric != tt.wantGeneric {
				t.Errorf("extractReceiverTypeName() hasGenerics = %v, want %v", gotGeneric, tt.wantGeneric)
			}
		})
	}
}
