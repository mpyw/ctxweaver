package service

import (
	"context"
)

func Process[T any](ctx context.Context, value T) error {

	// process the value
	return nil
}

func Transform[T, U any](ctx context.Context, input T) (U, error) {

	// transform input to output
	var zero U
	return zero, nil
}
