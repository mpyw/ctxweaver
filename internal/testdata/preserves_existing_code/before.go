package test

import "context"

func Foo(ctx context.Context) error {
	// This comment should be preserved
	x := 1
	y := 2
	return doSomething(x, y)
}

func doSomething(a, b int) error {
	return nil
}
