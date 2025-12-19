package test

import (
	"context"

	"go.opentelemetry.io/otel"
)

func Foo(ctx context.Context) error {
	ctx, span := otel.Tracer("").Start(ctx, "old.WrongName")
	defer span.End()

	return nil
}
