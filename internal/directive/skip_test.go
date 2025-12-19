package directive

import (
	"testing"

	"github.com/dave/dst"
)

func TestIsSkipComment(t *testing.T) {
	tests := map[string]struct {
		input string
		want  bool
	}{
		"exact match without space": {
			input: "//ctxweaver:skip",
			want:  true,
		},
		"exact match with space": {
			input: "// ctxweaver:skip",
			want:  true,
		},
		"multiple spaces after //": {
			input: "//  ctxweaver:skip",
			want:  true,
		},
		"with trailing content": {
			input: "//ctxweaver:skip this function",
			want:  true,
		},
		"with trailing content and space": {
			input: "// ctxweaver:skip this function",
			want:  true,
		},
		"different directive": {
			input: "//nolint:errcheck",
			want:  false,
		},
		"contains but not prefix": {
			input: "// some ctxweaver:skip comment",
			want:  false,
		},
		"partial match": {
			input: "//ctxweaver:skipme",
			want:  true, // HasPrefix allows this
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
			got := isSkipComment(tt.input)
			if got != tt.want {
				t.Errorf("isSkipComment(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestHasSkipDirective(t *testing.T) {
	tests := map[string]struct {
		decs *dst.NodeDecs
		want bool
	}{
		"has skip directive": {
			decs: &dst.NodeDecs{
				Start: dst.Decorations{"// ctxweaver:skip"},
			},
			want: true,
		},
		"has skip directive without space": {
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
					"// ctxweaver:skip",
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
			got := HasSkipDirective(tt.decs)
			if got != tt.want {
				t.Errorf("HasSkipDirective() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasStmtSkipDirective(t *testing.T) {
	tests := map[string]struct {
		stmt dst.Stmt
		want bool
	}{
		"skip in Start decoration": {
			stmt: &dst.ExprStmt{
				X: &dst.Ident{Name: "foo"},
				Decs: dst.ExprStmtDecorations{
					NodeDecs: dst.NodeDecs{
						Start: dst.Decorations{"// ctxweaver:skip"},
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
						End: dst.Decorations{"// ctxweaver:skip"},
					},
				},
			},
			want: true,
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
						Start: dst.Decorations{"// ctxweaver:skip"},
						End:   dst.Decorations{"// ctxweaver:skip"},
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
						Start: dst.Decorations{"// ctxweaver:skip"},
					},
				},
			},
			want: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := HasStmtSkipDirective(tt.stmt)
			if got != tt.want {
				t.Errorf("HasStmtSkipDirective() = %v, want %v", got, tt.want)
			}
		})
	}
}
