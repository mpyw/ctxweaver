package test

import (
	"context"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func Foo(ctx context.Context) error {
	span, ctx := tracer.StartSpanFromContext(ctx, "test.Foo")
	defer span.Finish()

	return nil
}
