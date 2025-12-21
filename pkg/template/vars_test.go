package template

import (
	"testing"

	"github.com/dave/dst"

	"github.com/mpyw/ctxweaver/pkg/config"
)

func TestBuildVars(t *testing.T) {
	tests := map[string]struct {
		file     *dst.File
		decl     *dst.FuncDecl
		pkgPath  string
		carrier  config.CarrierDef
		varName  string
		expected Vars
	}{
		"simple function": {
			file: &dst.File{Name: &dst.Ident{Name: "main"}},
			decl: &dst.FuncDecl{
				Name: &dst.Ident{Name: "Foo"},
				Type: &dst.FuncType{},
			},
			pkgPath: "github.com/example/myapp",
			carrier: config.CarrierDef{},
			varName: "ctx",
			expected: Vars{
				Ctx:          "ctx",
				CtxVar:       "ctx",
				PackageName:  "main",
				PackagePath:  "github.com/example/myapp",
				FuncBaseName: "Foo",
				FuncName:     "main.Foo",
			},
		},
		"generic function": {
			file: &dst.File{Name: &dst.Ident{Name: "pkg"}},
			decl: &dst.FuncDecl{
				Name: &dst.Ident{Name: "Transform"},
				Type: &dst.FuncType{
					TypeParams: &dst.FieldList{
						List: []*dst.Field{{Names: []*dst.Ident{{Name: "T"}}}},
					},
				},
			},
			pkgPath: "github.com/example/myapp/pkg",
			carrier: config.CarrierDef{},
			varName: "ctx",
			expected: Vars{
				Ctx:           "ctx",
				CtxVar:        "ctx",
				PackageName:   "pkg",
				PackagePath:   "github.com/example/myapp/pkg",
				FuncBaseName:  "Transform",
				FuncName:      "pkg.Transform[...]",
				IsGenericFunc: true,
			},
		},
		"method with pointer receiver": {
			file: &dst.File{Name: &dst.Ident{Name: "service"}},
			decl: &dst.FuncDecl{
				Name: &dst.Ident{Name: "Process"},
				Recv: &dst.FieldList{
					List: []*dst.Field{{
						Names: []*dst.Ident{{Name: "s"}},
						Type:  &dst.StarExpr{X: &dst.Ident{Name: "Service"}},
					}},
				},
				Type: &dst.FuncType{},
			},
			pkgPath: "github.com/example/myapp/service",
			carrier: config.CarrierDef{},
			varName: "ctx",
			expected: Vars{
				Ctx:               "ctx",
				CtxVar:            "ctx",
				PackageName:       "service",
				PackagePath:       "github.com/example/myapp/service",
				FuncBaseName:      "Process",
				FuncName:          "service.(*Service).Process",
				ReceiverType:      "Service",
				ReceiverVar:       "s",
				IsMethod:          true,
				IsPointerReceiver: true,
			},
		},
		"method with value receiver": {
			file: &dst.File{Name: &dst.Ident{Name: "service"}},
			decl: &dst.FuncDecl{
				Name: &dst.Ident{Name: "String"},
				Recv: &dst.FieldList{
					List: []*dst.Field{{
						Names: []*dst.Ident{{Name: "s"}},
						Type:  &dst.Ident{Name: "Service"},
					}},
				},
				Type: &dst.FuncType{},
			},
			pkgPath: "github.com/example/myapp/service",
			carrier: config.CarrierDef{},
			varName: "ctx",
			expected: Vars{
				Ctx:          "ctx",
				CtxVar:       "ctx",
				PackageName:  "service",
				PackagePath:  "github.com/example/myapp/service",
				FuncBaseName: "String",
				FuncName:     "service.Service.String",
				ReceiverType: "Service",
				ReceiverVar:  "s",
				IsMethod:     true,
			},
		},
		"generic pointer receiver": {
			file: &dst.File{Name: &dst.Ident{Name: "container"}},
			decl: &dst.FuncDecl{
				Name: &dst.Ident{Name: "Get"},
				Recv: &dst.FieldList{
					List: []*dst.Field{{
						Names: []*dst.Ident{{Name: "c"}},
						Type: &dst.StarExpr{
							X: &dst.IndexExpr{
								X:     &dst.Ident{Name: "Container"},
								Index: &dst.Ident{Name: "T"},
							},
						},
					}},
				},
				Type: &dst.FuncType{},
			},
			pkgPath: "github.com/example/myapp/container",
			carrier: config.CarrierDef{},
			varName: "ctx",
			expected: Vars{
				Ctx:               "ctx",
				CtxVar:            "ctx",
				PackageName:       "container",
				PackagePath:       "github.com/example/myapp/container",
				FuncBaseName:      "Get",
				FuncName:          "container.(*Container[...]).Get",
				ReceiverType:      "Container",
				ReceiverVar:       "c",
				IsMethod:          true,
				IsPointerReceiver: true,
				IsGenericReceiver: true,
			},
		},
		"generic value receiver": {
			file: &dst.File{Name: &dst.Ident{Name: "wrapper"}},
			decl: &dst.FuncDecl{
				Name: &dst.Ident{Name: "Unwrap"},
				Recv: &dst.FieldList{
					List: []*dst.Field{{
						Names: []*dst.Ident{{Name: "w"}},
						Type: &dst.IndexExpr{
							X:     &dst.Ident{Name: "Wrapper"},
							Index: &dst.Ident{Name: "T"},
						},
					}},
				},
				Type: &dst.FuncType{},
			},
			pkgPath: "github.com/example/myapp/wrapper",
			carrier: config.CarrierDef{},
			varName: "ctx",
			expected: Vars{
				Ctx:               "ctx",
				CtxVar:            "ctx",
				PackageName:       "wrapper",
				PackagePath:       "github.com/example/myapp/wrapper",
				FuncBaseName:      "Unwrap",
				FuncName:          "wrapper.Wrapper[...].Unwrap",
				ReceiverType:      "Wrapper",
				ReceiverVar:       "w",
				IsMethod:          true,
				IsGenericReceiver: true,
			},
		},
		"method without receiver name": {
			file: &dst.File{Name: &dst.Ident{Name: "service"}},
			decl: &dst.FuncDecl{
				Name: &dst.Ident{Name: "Process"},
				Recv: &dst.FieldList{
					List: []*dst.Field{{
						Type: &dst.StarExpr{X: &dst.Ident{Name: "Service"}},
					}},
				},
				Type: &dst.FuncType{},
			},
			pkgPath: "github.com/example/myapp/service",
			carrier: config.CarrierDef{},
			varName: "ctx",
			expected: Vars{
				Ctx:               "ctx",
				CtxVar:            "ctx",
				PackageName:       "service",
				PackagePath:       "github.com/example/myapp/service",
				FuncBaseName:      "Process",
				FuncName:          "service.(*Service).Process",
				ReceiverType:      "Service",
				ReceiverVar:       "",
				IsMethod:          true,
				IsPointerReceiver: true,
			},
		},
		"with accessor": {
			file: &dst.File{Name: &dst.Ident{Name: "handler"}},
			decl: &dst.FuncDecl{
				Name: &dst.Ident{Name: "Handle"},
				Type: &dst.FuncType{},
			},
			pkgPath: "github.com/example/myapp/handler",
			carrier: config.CarrierDef{Accessor: ".Request().Context()"},
			varName: "c",
			expected: Vars{
				Ctx:          "c.Request().Context()",
				CtxVar:       "c",
				PackageName:  "handler",
				PackagePath:  "github.com/example/myapp/handler",
				FuncBaseName: "Handle",
				FuncName:     "handler.Handle",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := BuildVars(tt.file, tt.decl, tt.pkgPath, tt.carrier, tt.varName)

			if got.Ctx != tt.expected.Ctx {
				t.Errorf("Ctx = %q, want %q", got.Ctx, tt.expected.Ctx)
			}
			if got.CtxVar != tt.expected.CtxVar {
				t.Errorf("CtxVar = %q, want %q", got.CtxVar, tt.expected.CtxVar)
			}
			if got.PackageName != tt.expected.PackageName {
				t.Errorf("PackageName = %q, want %q", got.PackageName, tt.expected.PackageName)
			}
			if got.PackagePath != tt.expected.PackagePath {
				t.Errorf("PackagePath = %q, want %q", got.PackagePath, tt.expected.PackagePath)
			}
			if got.FuncBaseName != tt.expected.FuncBaseName {
				t.Errorf("FuncBaseName = %q, want %q", got.FuncBaseName, tt.expected.FuncBaseName)
			}
			if got.FuncName != tt.expected.FuncName {
				t.Errorf("FuncName = %q, want %q", got.FuncName, tt.expected.FuncName)
			}
			if got.ReceiverType != tt.expected.ReceiverType {
				t.Errorf("ReceiverType = %q, want %q", got.ReceiverType, tt.expected.ReceiverType)
			}
			if got.ReceiverVar != tt.expected.ReceiverVar {
				t.Errorf("ReceiverVar = %q, want %q", got.ReceiverVar, tt.expected.ReceiverVar)
			}
			if got.IsMethod != tt.expected.IsMethod {
				t.Errorf("IsMethod = %v, want %v", got.IsMethod, tt.expected.IsMethod)
			}
			if got.IsPointerReceiver != tt.expected.IsPointerReceiver {
				t.Errorf("IsPointerReceiver = %v, want %v", got.IsPointerReceiver, tt.expected.IsPointerReceiver)
			}
			if got.IsGenericFunc != tt.expected.IsGenericFunc {
				t.Errorf("IsGenericFunc = %v, want %v", got.IsGenericFunc, tt.expected.IsGenericFunc)
			}
			if got.IsGenericReceiver != tt.expected.IsGenericReceiver {
				t.Errorf("IsGenericReceiver = %v, want %v", got.IsGenericReceiver, tt.expected.IsGenericReceiver)
			}
		})
	}
}

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
