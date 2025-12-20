package service

import (
	"context"
)

type Container[T any] struct {
	value T
}

func (c *Container[T]) Process(ctx context.Context) error {

	// process the value
	return nil
}

func (c *Container[T]) Handle(ctx context.Context, input T) (T, error) {

	// handle the input
	return c.value, nil
}
