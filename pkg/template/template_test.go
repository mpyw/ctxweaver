package template_test

import (
	"testing"

	"github.com/mpyw/ctxweaver/pkg/template"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input   string
		wantErr bool
	}{
		"simple template": {
			input: `defer trace({{.Ctx}})`,
		},
		"with quote function": {
			input: `defer trace({{.Ctx}}, {{.FuncName | quote}})`,
		},
		"with backtick function": {
			input: `defer trace({{.Ctx}}, {{.FuncName | backtick}})`,
		},
		"multiline": {
			input: `ctx, span := tracer.Start({{.Ctx}}, {{.FuncName | quote}})
defer span.End()`,
		},
		"invalid template": {
			input:   `defer trace({{.Ctx}`,
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := template.Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTemplate_Render(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		tmpl string
		vars template.Vars
		want string
	}{
		"simple context": {
			tmpl: `defer trace({{.Ctx}})`,
			vars: template.Vars{Ctx: "ctx"},
			want: `defer trace(ctx)`,
		},
		"echo context with accessor": {
			tmpl: `defer trace({{.Ctx}})`,
			vars: template.Vars{Ctx: "c.Request().Context()"},
			want: `defer trace(c.Request().Context())`,
		},
		"with quoted func name": {
			tmpl: `defer newrelic.FromContext({{.Ctx}}).StartSegment({{.FuncName | quote}}).End()`,
			vars: template.Vars{
				Ctx:      "ctx",
				FuncName: "pkg.(*Service).Method",
			},
			want: `defer newrelic.FromContext(ctx).StartSegment("pkg.(*Service).Method").End()`,
		},
		"with backtick func name": {
			tmpl: `defer trace({{.Ctx}}, {{.FuncName | backtick}})`,
			vars: template.Vars{
				Ctx:      "ctx",
				FuncName: "pkg.Func",
			},
			want: "defer trace(ctx, `pkg.Func`)",
		},
		"all variables": {
			tmpl: `// {{.FuncName}} in {{.PackagePath}}
defer trace({{.Ctx}}, {{.FuncBaseName | quote}})`,
			vars: template.Vars{
				Ctx:          "ctx",
				CtxVar:       "ctx",
				FuncName:     "myapp.(*Service).Process",
				PackageName:  "myapp",
				PackagePath:  "github.com/example/myapp",
				FuncBaseName: "Process",
				ReceiverType: "Service",
				ReceiverVar:  "s",
				IsMethod:     true,
			},
			want: `// myapp.(*Service).Process in github.com/example/myapp
defer trace(ctx, "Process")`,
		},
		"conditional method": {
			tmpl: `{{if .IsMethod}}// method on {{.ReceiverType}}{{else}}// function{{end}}
defer trace({{.Ctx}})`,
			vars: template.Vars{
				Ctx:          "ctx",
				IsMethod:     true,
				ReceiverType: "Handler",
			},
			want: `// method on Handler
defer trace(ctx)`,
		},
		"conditional function": {
			tmpl: `{{if .IsMethod}}// method{{else}}// function{{end}}
defer trace({{.Ctx}})`,
			vars: template.Vars{
				Ctx:      "ctx",
				IsMethod: false,
			},
			want: `// function
defer trace(ctx)`,
		},
		"generic receiver": {
			tmpl: `{{if .IsGenericReceiver}}// generic type{{end}}
defer trace({{.Ctx}}, {{.FuncName | quote}})`,
			vars: template.Vars{
				Ctx:               "ctx",
				FuncName:          "pkg.(*Container[...]).Process",
				ReceiverType:      "Container",
				IsMethod:          true,
				IsPointerReceiver: true,
				IsGenericReceiver: true,
			},
			want: `// generic type
defer trace(ctx, "pkg.(*Container[...]).Process")`,
		},
		"generic function": {
			tmpl: `{{if .IsGenericFunc}}// generic func{{end}}
defer trace({{.Ctx}}, {{.FuncName | quote}})`,
			vars: template.Vars{
				Ctx:           "ctx",
				FuncName:      "pkg.Transform[...]",
				FuncBaseName:  "Transform",
				IsGenericFunc: true,
			},
			want: `// generic func
defer trace(ctx, "pkg.Transform[...]")`,
		},
		"conditional generic handling": {
			tmpl: `{{if or .IsGenericFunc .IsGenericReceiver}}// has generics{{else}}// no generics{{end}}`,
			vars: template.Vars{
				IsGenericFunc:     false,
				IsGenericReceiver: true,
			},
			want: `// has generics`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmpl, err := template.Parse(tt.tmpl)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			got, err := tmpl.Render(tt.vars)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("Render() =\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestTemplate_Raw(t *testing.T) {
	t.Parallel()

	raw := `defer trace({{.Ctx}})`
	tmpl := template.MustParse(raw)

	if tmpl.Raw() != raw {
		t.Errorf("Raw() = %q, want %q", tmpl.Raw(), raw)
	}
}

func TestMustParse_Panic(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParse() should panic on invalid template")
		}
	}()

	template.MustParse(`{{.Invalid`)
}

func TestTemplate_Render_Error(t *testing.T) {
	t.Parallel()

	// Test rendering with missing required variable causes error
	tmpl, err := template.Parse(`{{.NonExistent.Field}}`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	_, err = tmpl.Render(template.Vars{})
	if err == nil {
		t.Error("Render() should error when accessing non-existent field")
	}
}
