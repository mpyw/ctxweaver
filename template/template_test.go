package template

import (
	"testing"
)

func TestParse(t *testing.T) {
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
			_, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTemplate_Render(t *testing.T) {
	tests := map[string]struct {
		template string
		vars     Vars
		want     string
	}{
		"simple context": {
			template: `defer trace({{.Ctx}})`,
			vars:     Vars{Ctx: "ctx"},
			want:     `defer trace(ctx)`,
		},
		"echo context with accessor": {
			template: `defer trace({{.Ctx}})`,
			vars:     Vars{Ctx: "c.Request().Context()"},
			want:     `defer trace(c.Request().Context())`,
		},
		"with quoted func name": {
			template: `defer apm.StartSegment({{.Ctx}}, {{.FuncName | quote}}).End()`,
			vars: Vars{
				Ctx:      "ctx",
				FuncName: "(*pkg.Service).Method",
			},
			want: `defer apm.StartSegment(ctx, "(*pkg.Service).Method").End()`,
		},
		"with backtick func name": {
			template: `defer trace({{.Ctx}}, {{.FuncName | backtick}})`,
			vars: Vars{
				Ctx:      "ctx",
				FuncName: "pkg.Func",
			},
			want: "defer trace(ctx, `pkg.Func`)",
		},
		"all variables": {
			template: `// {{.FuncName}} in {{.PackagePath}}
defer trace({{.Ctx}}, {{.FuncBaseName | quote}})`,
			vars: Vars{
				Ctx:          "ctx",
				CtxVar:       "ctx",
				FuncName:     "(*myapp.Service).Process",
				PackageName:  "myapp",
				PackagePath:  "github.com/example/myapp",
				FuncBaseName: "Process",
				ReceiverType: "Service",
				ReceiverVar:  "s",
				IsMethod:     true,
			},
			want: `// (*myapp.Service).Process in github.com/example/myapp
defer trace(ctx, "Process")`,
		},
		"conditional method": {
			template: `{{if .IsMethod}}// method on {{.ReceiverType}}{{else}}// function{{end}}
defer trace({{.Ctx}})`,
			vars: Vars{
				Ctx:          "ctx",
				IsMethod:     true,
				ReceiverType: "Handler",
			},
			want: `// method on Handler
defer trace(ctx)`,
		},
		"conditional function": {
			template: `{{if .IsMethod}}// method{{else}}// function{{end}}
defer trace({{.Ctx}})`,
			vars: Vars{
				Ctx:      "ctx",
				IsMethod: false,
			},
			want: `// function
defer trace(ctx)`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpl, err := Parse(tt.template)
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
	raw := `defer trace({{.Ctx}})`
	tmpl := MustParse(raw)

	if tmpl.Raw() != raw {
		t.Errorf("Raw() = %q, want %q", tmpl.Raw(), raw)
	}
}

func TestMustParse_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParse() should panic on invalid template")
		}
	}()

	MustParse(`{{.Invalid`)
}
