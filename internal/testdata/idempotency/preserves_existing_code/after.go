package test

import (
	"context"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func Foo(ctx context.Context) error {
	defer newrelic.FromContext(ctx).StartSegment("test.Foo").End()

	// This comment should be preserved
	x := 1
	y := 2
	return doSomething(x, y)
}

func doSomething(a, b int) error {
	return nil
}
