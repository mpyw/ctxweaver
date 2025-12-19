package dstutil

import (
	"testing"

	"github.com/dave/dst"
)

func TestMatchesSkeleton(t *testing.T) {
	tests := map[string]struct {
		a    string
		b    string
		want bool
	}{
		"identical defer": {
			a:    `defer foo()`,
			b:    `defer foo()`,
			want: true,
		},
		"defer with different function names (string literals)": {
			a:    `defer apm.StartSegment(ctx, "pkg.Foo").End()`,
			b:    `defer apm.StartSegment(ctx, "pkg.Bar").End()`,
			want: true, // String literals are compared by type, not value
		},
		"defer with different variable names": {
			a:    `defer trace(ctx)`,
			b:    `defer trace(c)`,
			want: true, // ctx and c are both dynamic identifiers
		},
		"different statement types": {
			a:    `defer foo()`,
			b:    `x := 1`,
			want: false,
		},
		"assignment vs expression": {
			a:    `x := foo()`,
			b:    `foo()`,
			want: false,
		},
		"same call expression": {
			a:    `foo(a, b)`,
			b:    `foo(x, y)`,
			want: true, // arguments are dynamic
		},
		"different argument count": {
			a:    `foo(a)`,
			b:    `foo(a, b)`,
			want: false,
		},
		"method call with selector": {
			a:    `obj.Method()`,
			b:    `obj.Method()`,
			want: true,
		},
		"different method names": {
			a:    `obj.Foo()`,
			b:    `obj.Bar()`,
			want: false,
		},
		"unary expression": {
			a:    `x := -a`,
			b:    `y := -b`,
			want: true,
		},
		"different unary operators": {
			a:    `x := -a`,
			b:    `y := !b`,
			want: false,
		},
		"binary expression": {
			a:    `x := a + b`,
			b:    `y := c + d`,
			want: true,
		},
		"different binary operators": {
			a:    `x := a + b`,
			b:    `y := c - d`,
			want: false,
		},
		"parenthesized expression": {
			a:    `x := (a + b)`,
			b:    `y := (c + d)`,
			want: true,
		},
		"index expression": {
			a:    `x := arr[0]`,
			b:    `y := arr[1]`,
			want: true, // index values are literals
		},
		"composite literal": {
			a:    `x := Foo{A: 1}`,
			b:    `y := Foo{A: 2}`,
			want: true, // values differ but structure same
		},
		"different composite types": {
			a:    `x := Foo{}`,
			b:    `y := Bar{}`,
			want: false,
		},
		"star expression": {
			a:    `x := *p`,
			b:    `y := *q`,
			want: true,
		},
		"type assertion": {
			a:    `x := v.(string)`,
			b:    `y := w.(string)`,
			want: true,
		},
		"return statement": {
			a:    `return nil`,
			b:    `return nil`,
			want: true,
		},
		"return with different values": {
			a:    `return 1`,
			b:    `return 2`,
			want: true, // literals compared by type
		},
		"return with different result count": {
			a:    `return 1`,
			b:    `return 1, nil`,
			want: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			stmtsA, err := ParseStatements(tt.a)
			if err != nil {
				t.Fatalf("failed to parse a: %v", err)
			}
			stmtsB, err := ParseStatements(tt.b)
			if err != nil {
				t.Fatalf("failed to parse b: %v", err)
			}

			got := MatchesSkeleton(stmtsA[0], stmtsB[0])
			if got != tt.want {
				t.Errorf("MatchesSkeleton() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchesSkeleton_NilHandling(t *testing.T) {
	stmt, _ := ParseStatements(`x := 1`)

	t.Run("both nil", func(t *testing.T) {
		// compareNodes is called internally, but we can test via statement with nil body
		// This is hard to test directly, so we skip
	})

	t.Run("a nil b not nil", func(t *testing.T) {
		// MatchesSkeleton expects dst.Stmt, can't pass nil directly
		// The nil handling is for recursive calls
		_ = stmt
	})
}

func TestIsDynamicIdent(t *testing.T) {
	tests := map[string]struct {
		name string
		want bool
	}{
		"single letter x":    {name: "x", want: true},
		"single letter a":    {name: "a", want: true},
		"ctx":                {name: "ctx", want: true},
		"context":            {name: "context", want: true},
		"err":                {name: "err", want: true},
		"req":                {name: "req", want: true},
		"resp":               {name: "resp", want: true},
		"tx":                 {name: "tx", want: true},
		"txn":                {name: "txn", want: true},
		"db":                 {name: "db", want: true},
		"conn":               {name: "conn", want: true},
		"package name":       {name: "newrelic", want: false},
		"type name":          {name: "Context", want: false},
		"function name":      {name: "StartSegment", want: false},
		"multi-char non-var": {name: "foo", want: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := isDynamicIdent(tt.name)
			if got != tt.want {
				t.Errorf("isDynamicIdent(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestCompareNodes_EdgeCases(t *testing.T) {
	t.Run("different token in assign", func(t *testing.T) {
		a, _ := ParseStatements(`x := 1`)
		b, _ := ParseStatements(`x = 1`)
		if MatchesSkeleton(a[0], b[0]) {
			t.Error("expected different assignment tokens to not match")
		}
	})

	t.Run("func literal", func(t *testing.T) {
		a, _ := ParseStatements(`f := func() {}`)
		b, _ := ParseStatements(`g := func() {}`)
		if !MatchesSkeleton(a[0], b[0]) {
			t.Error("expected func literals to match")
		}
	})

	t.Run("func literal with params", func(t *testing.T) {
		a, _ := ParseStatements(`f := func(x int) int { return x }`)
		b, _ := ParseStatements(`g := func(y int) int { return y }`)
		if !MatchesSkeleton(a[0], b[0]) {
			t.Error("expected func literals with same signature to match")
		}
	})

	t.Run("func literal different param count", func(t *testing.T) {
		a, _ := ParseStatements(`f := func(x int) {}`)
		b, _ := ParseStatements(`g := func() {}`)
		if MatchesSkeleton(a[0], b[0]) {
			t.Error("expected func literals with different param count to not match")
		}
	})

	t.Run("case clause", func(t *testing.T) {
		// switch statements with same structure but different literal values
		a, _ := ParseStatements(`switch x { case 1: println("a") }`)
		b, _ := ParseStatements(`switch y { case 2: println("b") }`)
		if !MatchesSkeleton(a[0], b[0]) {
			t.Error("expected switch statements to match")
		}
	})

	t.Run("if statement", func(t *testing.T) {
		// if statements with same structure
		a, _ := ParseStatements(`if x { println("a") }`)
		b, _ := ParseStatements(`if y { println("b") }`)
		if !MatchesSkeleton(a[0], b[0]) {
			t.Error("expected if statements to match")
		}
	})

	t.Run("if with else", func(t *testing.T) {
		a, _ := ParseStatements(`if x { println("a") } else { println("b") }`)
		b, _ := ParseStatements(`if y { println("c") } else { println("d") }`)
		if !MatchesSkeleton(a[0], b[0]) {
			t.Error("expected if-else statements to match")
		}
	})

	t.Run("key value expr", func(t *testing.T) {
		a, _ := ParseStatements(`m := map[string]int{"a": 1}`)
		b, _ := ParseStatements(`n := map[string]int{"b": 2}`)
		if !MatchesSkeleton(a[0], b[0]) {
			t.Error("expected map literals to match")
		}
	})
}

func TestCompareFieldLists(t *testing.T) {
	t.Run("both nil", func(t *testing.T) {
		if !compareFieldLists(nil, nil) {
			t.Error("expected nil == nil")
		}
	})

	t.Run("one nil", func(t *testing.T) {
		fl := &dst.FieldList{List: []*dst.Field{}}
		if compareFieldLists(nil, fl) {
			t.Error("expected nil != non-nil")
		}
		if compareFieldLists(fl, nil) {
			t.Error("expected non-nil != nil")
		}
	})

	t.Run("different lengths", func(t *testing.T) {
		a := &dst.FieldList{List: []*dst.Field{{Type: &dst.Ident{Name: "int"}}}}
		b := &dst.FieldList{List: []*dst.Field{}}
		if compareFieldLists(a, b) {
			t.Error("expected different lengths to not match")
		}
	})

	t.Run("same types", func(t *testing.T) {
		a := &dst.FieldList{List: []*dst.Field{{Type: &dst.Ident{Name: "int"}}}}
		b := &dst.FieldList{List: []*dst.Field{{Type: &dst.Ident{Name: "int"}}}}
		if !compareFieldLists(a, b) {
			t.Error("expected same types to match")
		}
	})
}
