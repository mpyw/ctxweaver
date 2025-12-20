package service

import (
	"context"

	"go.opentelemetry.io/otel"
)

type Container[T any] struct {
	value T
}

func (c *Container[T]) Process(ctx context.Context) error {
	ctx, span := otel.Tracer("").Start(ctx, "service.(*Container[...]).Process")
	defer span.End()

	// process the value
	return nil
}

func (c *Container[T]) Handle(ctx context.Context, input T) (T, error) {
	ctx, span := otel.Tracer("").Start(ctx, "service.(*Container[...]).Handle")
	defer span.End()

	// handle the input
	return c.value, nil
}
