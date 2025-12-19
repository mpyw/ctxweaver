package test

import (
	"context"
)

//ctxweaver:skip
func trace(_ context.Context) {}

func Foo(ctx context.Context) {
	defer trace(ctx)

	// trailing comment after blank line

	println("hello")
}
