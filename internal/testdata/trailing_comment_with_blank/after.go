package test

import "context"

func Foo(ctx context.Context) {
	defer trace(ctx)

	// trailing comment after blank line

	println("hello")
}
