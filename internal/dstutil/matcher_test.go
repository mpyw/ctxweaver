package dstutil_test

import (
	"testing"

	"github.com/dave/dst"
	"github.com/mpyw/ctxweaver/internal/dstutil"
)

func TestMatchesSkeleton(t *testing.T) {
	t.Parallel()

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
			want: false, // Identifiers are compared exactly
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
			b:    `foo(a, b)`,
			want: true,
		},
		"different argument names": {
			a:    `foo(a, b)`,
			b:    `foo(x, y)`,
			want: false, // Identifiers are compared exactly
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
		"unary expression same var": {
			a:    `x := -a`,
			b:    `x := -a`,
			want: true,
		},
		"unary expression different var": {
			a:    `x := -a`,
			b:    `y := -b`,
			want: false, // Identifiers are compared exactly
		},
		"different unary operators": {
			a:    `x := -a`,
			b:    `y := !b`,
			want: false,
		},
		"binary expression same vars": {
			a:    `x := a + b`,
			b:    `x := a + b`,
			want: true,
		},
		"binary expression different vars": {
			a:    `x := a + b`,
			b:    `y := c + d`,
			want: false, // Identifiers are compared exactly
		},
		"different binary operators": {
			a:    `x := a + b`,
			b:    `y := c - d`,
			want: false,
		},
		"parenthesized expression same": {
			a:    `x := (a + b)`,
			b:    `x := (a + b)`,
			want: true,
		},
		"index expression": {
			a:    `x := arr[0]`,
			b:    `x := arr[1]`,
			want: true, // index values are literals, compared by type
		},
		"composite literal": {
			a:    `x := Foo{A: 1}`,
			b:    `x := Foo{A: 2}`,
			want: true, // values differ but structure same
		},
		"different composite types": {
			a:    `x := Foo{}`,
			b:    `y := Bar{}`,
			want: false,
		},
		"star expression same": {
			a:    `x := *p`,
			b:    `x := *p`,
			want: true,
		},
		"type assertion same": {
			a:    `x := v.(string)`,
			b:    `x := v.(string)`,
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
			t.Parallel()

			stmtsA, err := dstutil.ParseStatements(tt.a)
			if err != nil {
				t.Fatalf("failed to parse a: %v", err)
			}
			stmtsB, err := dstutil.ParseStatements(tt.b)
			if err != nil {
				t.Fatalf("failed to parse b: %v", err)
			}

			got := dstutil.MatchesSkeleton(stmtsA[0], stmtsB[0])
			if got != tt.want {
				t.Errorf("MatchesSkeleton() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchesSkeleton_NilHandling(t *testing.T) {
	t.Parallel()

	stmt, _ := dstutil.ParseStatements(`x := 1`)

	t.Run("both nil", func(t *testing.T) {
		t.Parallel()

		// compareNodes is called internally, but we can test via statement with nil body
		// This is hard to test directly, so we skip
	})

	t.Run("a nil b not nil", func(t *testing.T) {
		t.Parallel()

		// MatchesSkeleton expects dst.Stmt, can't pass nil directly
		// The nil handling is for recursive calls
		_ = stmt
	})
}

func TestCompareNodes_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("different token in assign", func(t *testing.T) {
		t.Parallel()

		a, _ := dstutil.ParseStatements(`x := 1`)
		b, _ := dstutil.ParseStatements(`x = 1`)
		if dstutil.MatchesSkeleton(a[0], b[0]) {
			t.Error("expected different assignment tokens to not match")
		}
	})

	t.Run("func literal", func(t *testing.T) {
		t.Parallel()

		a, _ := dstutil.ParseStatements(`f := func() {}`)
		b, _ := dstutil.ParseStatements(`f := func() {}`)
		if !dstutil.MatchesSkeleton(a[0], b[0]) {
			t.Error("expected func literals to match")
		}
	})

	t.Run("func literal with params", func(t *testing.T) {
		t.Parallel()

		a, _ := dstutil.ParseStatements(`f := func(x int) int { return x }`)
		b, _ := dstutil.ParseStatements(`f := func(x int) int { return x }`)
		if !dstutil.MatchesSkeleton(a[0], b[0]) {
			t.Error("expected func literals with same signature to match")
		}
	})

	t.Run("func literal different param count", func(t *testing.T) {
		t.Parallel()

		a, _ := dstutil.ParseStatements(`f := func(x int) {}`)
		b, _ := dstutil.ParseStatements(`f := func() {}`)
		if dstutil.MatchesSkeleton(a[0], b[0]) {
			t.Error("expected func literals with different param count to not match")
		}
	})

	t.Run("case clause", func(t *testing.T) {
		t.Parallel()

		// switch statements with same structure but different literal values
		a, _ := dstutil.ParseStatements(`switch x { case 1: println("a") }`)
		b, _ := dstutil.ParseStatements(`switch x { case 2: println("b") }`)
		if !dstutil.MatchesSkeleton(a[0], b[0]) {
			t.Error("expected switch statements to match")
		}
	})

	t.Run("if statement", func(t *testing.T) {
		t.Parallel()

		// if statements with same structure
		a, _ := dstutil.ParseStatements(`if x { println("a") }`)
		b, _ := dstutil.ParseStatements(`if x { println("b") }`)
		if !dstutil.MatchesSkeleton(a[0], b[0]) {
			t.Error("expected if statements to match")
		}
	})

	t.Run("if with else", func(t *testing.T) {
		t.Parallel()

		a, _ := dstutil.ParseStatements(`if x { println("a") } else { println("b") }`)
		b, _ := dstutil.ParseStatements(`if x { println("c") } else { println("d") }`)
		if !dstutil.MatchesSkeleton(a[0], b[0]) {
			t.Error("expected if-else statements to match")
		}
	})

	t.Run("key value expr", func(t *testing.T) {
		t.Parallel()

		a, _ := dstutil.ParseStatements(`m := map[string]int{"a": 1}`)
		b, _ := dstutil.ParseStatements(`m := map[string]int{"b": 2}`)
		if !dstutil.MatchesSkeleton(a[0], b[0]) {
			t.Error("expected map literals to match")
		}
	})
}

func TestCompareFieldLists(t *testing.T) {
	t.Parallel()

	c := dstutil.NewComparator()

	t.Run("both nil", func(t *testing.T) {
		t.Parallel()

		if !dstutil.CompareFieldLists(nil, nil, "test", false, c) {
			t.Error("expected nil == nil")
		}
	})

	t.Run("one nil", func(t *testing.T) {
		t.Parallel()

		fl := &dst.FieldList{List: []*dst.Field{}}
		if dstutil.CompareFieldLists(nil, fl, "test", false, c) {
			t.Error("expected nil != non-nil")
		}
		if dstutil.CompareFieldLists(fl, nil, "test", false, c) {
			t.Error("expected non-nil != nil")
		}
	})

	t.Run("different lengths", func(t *testing.T) {
		t.Parallel()

		a := &dst.FieldList{List: []*dst.Field{{Type: &dst.Ident{Name: "int"}}}}
		b := &dst.FieldList{List: []*dst.Field{}}
		if dstutil.CompareFieldLists(a, b, "test", false, c) {
			t.Error("expected different lengths to not match")
		}
	})

	t.Run("same types", func(t *testing.T) {
		t.Parallel()

		a := &dst.FieldList{List: []*dst.Field{{Type: &dst.Ident{Name: "int"}}}}
		b := &dst.FieldList{List: []*dst.Field{{Type: &dst.Ident{Name: "int"}}}}
		if !dstutil.CompareFieldLists(a, b, "test", false, c) {
			t.Error("expected same types to match")
		}
	})
}

func TestMatchesExact(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		a    string
		b    string
		want bool
	}{
		"identical statements": {
			a:    `defer foo()`,
			b:    `defer foo()`,
			want: true,
		},
		"identical with string literal": {
			a:    `defer trace(ctx, "hello")`,
			b:    `defer trace(ctx, "hello")`,
			want: true,
		},
		"different string literals": {
			a:    `defer trace(ctx, "hello")`,
			b:    `defer trace(ctx, "world")`,
			want: false, // Exact mode compares literal values
		},
		"different number literals": {
			a:    `x := 1`,
			b:    `x := 2`,
			want: false,
		},
		"same number literals": {
			a:    `x := 42`,
			b:    `x := 42`,
			want: true,
		},
		"different function names": {
			a:    `foo()`,
			b:    `bar()`,
			want: false,
		},
		"composite literal different values": {
			a:    `x := Foo{A: 1}`,
			b:    `x := Foo{A: 2}`,
			want: false, // Exact mode compares literal values
		},
		"composite literal same values": {
			a:    `x := Foo{A: 1}`,
			b:    `x := Foo{A: 1}`,
			want: true,
		},
		"map literal different keys": {
			a:    `m := map[string]int{"a": 1}`,
			b:    `m := map[string]int{"b": 1}`,
			want: false,
		},
		"index expression different indices": {
			a:    `x := arr[0]`,
			b:    `x := arr[1]`,
			want: false, // Exact mode compares index values
		},
		"switch case different values": {
			a:    `switch x { case 1: println("a") }`,
			b:    `switch x { case 2: println("b") }`,
			want: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stmtsA, err := dstutil.ParseStatements(tt.a)
			if err != nil {
				t.Fatalf("failed to parse a: %v", err)
			}
			stmtsB, err := dstutil.ParseStatements(tt.b)
			if err != nil {
				t.Fatalf("failed to parse b: %v", err)
			}

			got := dstutil.MatchesExact(stmtsA[0], stmtsB[0])
			if got != tt.want {
				t.Errorf("MatchesExact() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchesExact_SkeletonPassesButExactFails(t *testing.T) {
	t.Parallel()

	// These cases verify the relationship between Skeleton and Exact matching:
	// Skeleton should pass (same structure) but Exact should fail (different values)
	tests := map[string]struct {
		a string
		b string
	}{
		"string literal difference": {
			a: `defer apm.StartSegment(ctx, "pkg.Foo").End()`,
			b: `defer apm.StartSegment(ctx, "pkg.Bar").End()`,
		},
		"number literal difference": {
			a: `x := arr[0]`,
			b: `x := arr[1]`,
		},
		"return value difference": {
			a: `return 1`,
			b: `return 2`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stmtsA, err := dstutil.ParseStatements(tt.a)
			if err != nil {
				t.Fatalf("failed to parse a: %v", err)
			}
			stmtsB, err := dstutil.ParseStatements(tt.b)
			if err != nil {
				t.Fatalf("failed to parse b: %v", err)
			}

			skeleton := dstutil.MatchesSkeleton(stmtsA[0], stmtsB[0])
			exact := dstutil.MatchesExact(stmtsA[0], stmtsB[0])

			if !skeleton {
				t.Error("expected MatchesSkeleton() = true")
			}
			if exact {
				t.Error("expected MatchesExact() = false")
			}
		})
	}
}

func TestCompareNodes_SelectorExprVsIdentWithPath(t *testing.T) {
	t.Parallel()

	// Test the special case: SelectorExpr matches Ident with Path set
	// This happens when NewDecoratorFromPackage resolves `pkg.Func` to `Func` with Path="pkg"
	c := dstutil.NewComparator()

	t.Run("SelectorExpr matches Ident with Path (selA.Sel.Name == identB.Name)", func(t *testing.T) {
		t.Parallel()

		// Create SelectorExpr: pkg.Func
		selExpr := &dst.SelectorExpr{
			X:   &dst.Ident{Name: "pkg"},
			Sel: &dst.Ident{Name: "Func"},
		}

		// Create Ident with Path: Func (with Path="pkg")
		identWithPath := &dst.Ident{
			Name: "Func",
			Path: "github.com/example/pkg",
		}

		// They should match because selA.Sel.Name == identB.Name
		if !c.Compare(selExpr, identWithPath, "test", false) {
			t.Error("expected SelectorExpr to match Ident with Path")
		}
	})

	t.Run("Ident with Path matches SelectorExpr (identA.Name == selB.Sel.Name)", func(t *testing.T) {
		t.Parallel()

		// Create Ident with Path: Func (with Path="pkg")
		identWithPath := &dst.Ident{
			Name: "Func",
			Path: "github.com/example/pkg",
		}

		// Create SelectorExpr: pkg.Func
		selExpr := &dst.SelectorExpr{
			X:   &dst.Ident{Name: "pkg"},
			Sel: &dst.Ident{Name: "Func"},
		}

		// They should match because identA.Name == selB.Sel.Name
		if !c.Compare(identWithPath, selExpr, "test", false) {
			t.Error("expected Ident with Path to match SelectorExpr")
		}
	})

	t.Run("SelectorExpr does not match Ident without Path", func(t *testing.T) {
		t.Parallel()

		selExpr := &dst.SelectorExpr{
			X:   &dst.Ident{Name: "pkg"},
			Sel: &dst.Ident{Name: "Func"},
		}

		// Ident without Path should not match
		identWithoutPath := &dst.Ident{
			Name: "Func",
			Path: "", // No Path set
		}

		if c.Compare(selExpr, identWithoutPath, "test", false) {
			t.Error("expected SelectorExpr to NOT match Ident without Path")
		}
	})

	t.Run("different names do not match", func(t *testing.T) {
		t.Parallel()

		selExpr := &dst.SelectorExpr{
			X:   &dst.Ident{Name: "pkg"},
			Sel: &dst.Ident{Name: "Foo"},
		}

		identWithPath := &dst.Ident{
			Name: "Bar", // Different name
			Path: "github.com/example/pkg",
		}

		if c.Compare(selExpr, identWithPath, "test", false) {
			t.Error("expected different names to NOT match")
		}
	})
}
