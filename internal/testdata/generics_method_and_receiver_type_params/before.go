package service

import (
	"context"
)

type Container[T any] struct {
	value T
}

func (c Container[T]) Map[U any](ctx context.Context, f func(T) U) Container[U] {

	// map the value
	return Container[U]{value: f(c.value)}
}

func (c *Container[T]) Fold[A, B any](ctx context.Context, a A, b B) (A, B) {

	// fold with two type parameters
	return a, b
}

func (c *Container[T]) Get(ctx context.Context) T {

	// non-generic method on a generic receiver
	return c.value
}
