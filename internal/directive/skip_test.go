package directive

import (
	"testing"

	"github.com/dave/dst"
)

func TestIsSkipComment(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input string
		want  bool
	}{
		"canonical": {
			input: "//ctxweaver:skip",
			want:  true,
		},
		"canonical with trailing content": {
			input: "//ctxweaver:skip this function",
			want:  true,
		},
		"canonical with trailing whitespace": {
			input: "//ctxweaver:skip  ",
			want:  true,
		},
		"space after //": {
			input: "// ctxweaver:skip",
			want:  false,
		},
		"multiple spaces after //": {
			input: "//  ctxweaver:skip",
			want:  false,
		},
		"tab after //": {
			input: "//\tctxweaver:skip",
			want:  false,
		},
		"space after the colon": {
			input: "//ctxweaver: skip",
			want:  false,
		},
		"block comment": {
			input: "/*ctxweaver:skip*/",
			want:  false,
		},
		"different directive": {
			input: "//nolint:errcheck",
			want:  false,
		},
		"contains but not prefix": {
			input: "// some ctxweaver:skip comment",
			want:  false,
		},
		"different directive name sharing the skip prefix": {
			input: "//ctxweaver:skipx",
			want:  false, // the directive name must be exactly "skip"
		},
		"skip name in a different tool namespace": {
			input: "//other:skip",
			want:  false,
		},
		"tool name without a directive name": {
			input: "//ctxweaver:",
			want:  false,
		},
		"empty comment": {
			input: "//",
			want:  false,
		},
		"just whitespace": {
			input: "//   ",
			want:  false,
		},
		"lowercase variant": {
			input: "//CTXWEAVER:SKIP",
			want:  false, // case sensitive
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := isSkipComment(tt.input)
			if got != tt.want {
				t.Errorf("IsSkipComment(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestHasSkipDirective(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		decs *dst.NodeDecs
		want bool
	}{
		"spaced directive does not skip": {
			decs: &dst.NodeDecs{
				Start: dst.Decorations{"// ctxweaver:skip"},
			},
			want: false,
		},
		"block directive does not skip": {
			decs: &dst.NodeDecs{
				Start: dst.Decorations{"/*ctxweaver:skip*/"},
			},
			want: false,
		},
		"has skip directive": {
			decs: &dst.NodeDecs{
				Start: dst.Decorations{"//ctxweaver:skip"},
			},
			want: true,
		},
		"no skip directive": {
			decs: &dst.NodeDecs{
				Start: dst.Decorations{"// some comment"},
			},
			want: false,
		},
		"empty decorations": {
			decs: &dst.NodeDecs{},
			want: false,
		},
		"multiple comments with skip": {
			decs: &dst.NodeDecs{
				Start: dst.Decorations{
					"// first comment",
					"//ctxweaver:skip",
					"// third comment",
				},
			},
			want: true,
		},
		"skip in wrong position (not prefix)": {
			decs: &dst.NodeDecs{
				Start: dst.Decorations{"// do not ctxweaver:skip"},
			},
			want: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := HasSkipDirective(tt.decs)
			if got != tt.want {
				t.Errorf("HasSkipDirective() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasStmtSkipDirective(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		stmt dst.Stmt
		want bool
	}{
		"skip in Start decoration": {
			stmt: &dst.ExprStmt{
				X: &dst.Ident{Name: "foo"},
				Decs: dst.ExprStmtDecorations{
					NodeDecs: dst.NodeDecs{
						Start: dst.Decorations{"//ctxweaver:skip"},
					},
				},
			},
			want: true,
		},
		"skip in End decoration": {
			stmt: &dst.ExprStmt{
				X: &dst.Ident{Name: "foo"},
				Decs: dst.ExprStmtDecorations{
					NodeDecs: dst.NodeDecs{
						End: dst.Decorations{"//ctxweaver:skip"},
					},
				},
			},
			want: true,
		},
		"spaced directive in Start does not skip": {
			stmt: &dst.ExprStmt{
				X: &dst.Ident{Name: "foo"},
				Decs: dst.ExprStmtDecorations{
					NodeDecs: dst.NodeDecs{
						Start: dst.Decorations{"// ctxweaver:skip"},
					},
				},
			},
			want: false,
		},
		"spaced-colon directive in End does not skip": {
			stmt: &dst.ExprStmt{
				X: &dst.Ident{Name: "foo"},
				Decs: dst.ExprStmtDecorations{
					NodeDecs: dst.NodeDecs{
						End: dst.Decorations{"//ctxweaver: skip"},
					},
				},
			},
			want: false,
		},
		"block directive in End does not skip": {
			stmt: &dst.ExprStmt{
				X: &dst.Ident{Name: "foo"},
				Decs: dst.ExprStmtDecorations{
					NodeDecs: dst.NodeDecs{
						End: dst.Decorations{"/* ctxweaver:skip */"},
					},
				},
			},
			want: false,
		},
		"no skip directive": {
			stmt: &dst.ExprStmt{
				X: &dst.Ident{Name: "foo"},
				Decs: dst.ExprStmtDecorations{
					NodeDecs: dst.NodeDecs{
						Start: dst.Decorations{"// other comment"},
					},
				},
			},
			want: false,
		},
		"empty decorations": {
			stmt: &dst.ExprStmt{
				X: &dst.Ident{Name: "foo"},
			},
			want: false,
		},
		"skip in both Start and End": {
			stmt: &dst.ExprStmt{
				X: &dst.Ident{Name: "foo"},
				Decs: dst.ExprStmtDecorations{
					NodeDecs: dst.NodeDecs{
						Start: dst.Decorations{"//ctxweaver:skip"},
						End:   dst.Decorations{"//ctxweaver:skip"},
					},
				},
			},
			want: true,
		},
		"defer statement with skip": {
			stmt: &dst.DeferStmt{
				Call: &dst.CallExpr{
					Fun: &dst.Ident{Name: "cleanup"},
				},
				Decs: dst.DeferStmtDecorations{
					NodeDecs: dst.NodeDecs{
						End: dst.Decorations{"//ctxweaver:skip"},
					},
				},
			},
			want: true,
		},
		"assign statement with skip": {
			stmt: &dst.AssignStmt{
				Lhs: []dst.Expr{&dst.Ident{Name: "x"}},
				Tok: 47, // :=
				Rhs: []dst.Expr{&dst.BasicLit{Kind: 5, Value: "1"}},
				Decs: dst.AssignStmtDecorations{
					NodeDecs: dst.NodeDecs{
						Start: dst.Decorations{"//ctxweaver:skip"},
					},
				},
			},
			want: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := HasStmtSkipDirective(tt.stmt)
			if got != tt.want {
				t.Errorf("HasStmtSkipDirective() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMalformed(t *testing.T) {
	t.Parallel()

	const (
		plain   = "malformed ctxweaver directive"
		suggest = plain + ": write //ctxweaver:skip"
	)

	tests := map[string]struct {
		input string
		want  string // empty when the comment is not malformed
	}{
		"canonical":                       {input: "//ctxweaver:skip"},
		"canonical with trailing content": {input: "//ctxweaver:skip legacy code"},
		"canonical lookalike name":        {input: "//ctxweaver:skipx"},
		"space after //":                  {input: "// ctxweaver:skip", want: suggest},
		"spaces after //":                 {input: "//   ctxweaver:skip", want: suggest},
		"tab after //":                    {input: "//\tctxweaver:skip", want: suggest},
		"space after the colon":           {input: "//ctxweaver: skip", want: suggest},
		"space after // and the colon":    {input: "// ctxweaver: skip legacy", want: suggest},
		"block comment":                   {input: "/*ctxweaver:skip*/", want: suggest},
		"block comment with spaces":       {input: "/* ctxweaver:skip */", want: suggest},
		"spaced lookalike keeps its name": {input: "// ctxweaver:skipx", want: plain + ": write //ctxweaver:skipx"},
		"uppercase name":                  {input: "//ctxweaver:Skip", want: plain},
		"uppercase name with a space":     {input: "// ctxweaver:SKIP", want: plain},
		"uppercase name in a block":       {input: "/*ctxweaver:Skip*/", want: plain},
		"no name":                         {input: "//ctxweaver:", want: plain},
		"no name with a space":            {input: "// ctxweaver:", want: plain},
		"no name with trailing spaces":    {input: "//ctxweaver:   ", want: plain},
		"no name in a block":              {input: "/* ctxweaver: */", want: plain},
		"prose mentioning the directive":  {input: "// write ctxweaver:skip to opt out"},
		"prose in a block comment":        {input: "/* see ctxweaver:skip */"},
		"other tool":                      {input: "// nolint:errcheck"},
		"other tool canonical":            {input: "//go:generate foo"},
		"plain comment":                   {input: "// hello"},
		"uppercase tool":                  {input: "// CTXWEAVER:skip"},
		"tool name as a prefix of a word": {input: "// ctxweaverish:skip"},
		"empty line comment":              {input: "//"},
		"empty block comment":             {input: "/**/"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, ok := Malformed(tt.input)
			if ok != (tt.want != "") || got != tt.want {
				t.Errorf("Malformed(%q) = (%q, %v), want %q", tt.input, got, ok, tt.want)
			}
		})
	}
}
