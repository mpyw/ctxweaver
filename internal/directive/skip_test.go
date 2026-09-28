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
		"free text after the name": {
			input: "//ctxweaver:skip this function",
			want:  false, // a reason goes after "//"
		},
		"reason after //": {
			input: "//ctxweaver:skip // this function",
			want:  true,
		},
		"reason after // without a space": {
			input: "//ctxweaver:skip //this function",
			want:  true,
		},
		"reason glued to the name": {
			input: "//ctxweaver:skip//this function",
			want:  true,
		},
		"reason after a dash": {
			input: "//ctxweaver:skip - this function",
			want:  true,
		},
		"misspelled name": {
			input: "//ctxweaver:skp",
			want:  false,
		},
		"directive after another directive": {
			input: "//nolint:foo //ctxweaver:skip",
			want:  false,
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

func TestProblem(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		input string
		want  string
	}{
		"canonical":                          {input: "//ctxweaver:skip", want: ""},
		"trailing whitespace":                {input: "//ctxweaver:skip \t", want: ""},
		"reason after //":                    {input: "//ctxweaver:skip // legacy code", want: ""},
		"reason after // without a space":    {input: "//ctxweaver:skip //legacy code", want: ""},
		"reason glued to the name":           {input: "//ctxweaver:skip//legacy code", want: ""},
		"reason after a dash":                {input: "//ctxweaver:skip - legacy code", want: ""},
		"bare dash":                          {input: "//ctxweaver:skip -", want: ""},
		"reason mentioning another tool":     {input: "//ctxweaver:skip // see //nolint:foo", want: ""},
		"free text":                          {input: "//ctxweaver:skip legacy code", want: SkipArgMessage},
		"dash without a space":               {input: "//ctxweaver:skip -legacy", want: SkipArgMessage},
		"misspelled":                         {input: "//ctxweaver:skp", want: UnknownMessage + "skp"},
		"misspelled with a reason":           {input: "//ctxweaver:skp // reason", want: UnknownMessage + "skp"},
		"hyphenated name":                    {input: "//ctxweaver:skip-legacy", want: UnknownMessage + "skip-legacy"},
		"lookalike":                          {input: "//ctxweaver:skipx", want: UnknownMessage + "skipx"},
		"after another directive":            {input: "//nolint:foo //ctxweaver:skip", want: HiddenMessage},
		"after another directive with space": {input: "//nolint:foo // ctxweaver:skip", want: HiddenMessage},
		"after prose":                        {input: "// see //ctxweaver:skip", want: HiddenMessage},
		"inside a block comment":             {input: "/* foo //ctxweaver:skip */", want: HiddenMessage},
		"space after //":                     {input: "// ctxweaver:skip", want: MalformedMessage},
		"space after the colon":              {input: "//ctxweaver: skip", want: MalformedMessage},
		"no name":                            {input: "//ctxweaver:", want: MalformedMessage},
		"block comment":                      {input: "/*ctxweaver:skip*/", want: MalformedMessage},
		"uppercase name":                     {input: "//ctxweaver:Skip", want: MalformedMessage},
		"prose":                              {input: "// write ctxweaver:skip to opt out", want: ""},
		"other tool":                         {input: "//nolint:errcheck", want: ""},
		"other tool with a reason":           {input: "//nolint:errcheck // reason", want: ""},
		"url":                                {input: "// see https://example.com/ctxweaver:skip", want: ""},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := Problem(tt.input); got != tt.want {
				t.Errorf("Problem(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
