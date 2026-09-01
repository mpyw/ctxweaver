package service

import (
	"context"

	"go.opentelemetry.io/otel"
)

type Container[T any] struct {
	value T
}

func (c Container[T]) Map[U any](ctx context.Context, f func(T) U) Container[U] {
	ctx, span := otel.Tracer("").Start(ctx, "service.Container[...].Map[...]")
	defer span.End()

	// map the value
	return Container[U]{value: f(c.value)}
}

func (c *Container[T]) Fold[A, B any](ctx context.Context, a A, b B) (A, B) {
	ctx, span := otel.Tracer("").Start(ctx, "service.(*Container[...]).Fold[...]")
	defer span.End()

	// fold with two type parameters
	return a, b
}

func (c *Container[T]) Get(ctx context.Context) T {
	ctx, span := otel.Tracer("").Start(ctx, "service.(*Container[...]).Get")
	defer span.End()

	// non-generic method on a generic receiver
	return c.value
}
