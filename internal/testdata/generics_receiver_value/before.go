package service

import (
	"context"
)

type Wrapper[T any] struct {
	inner T
}

func (w Wrapper[T]) Get(ctx context.Context) T {

	// return the inner value
	return w.inner
}
