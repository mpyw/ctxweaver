package test

import (
	"context"
)

//ctxweaver:skip
func trace(_ context.Context) {}

func Foo(ctx context.Context) {

	println("hello")
}
