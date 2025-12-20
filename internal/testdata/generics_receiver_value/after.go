package service

import (
	"context"

	"go.opentelemetry.io/otel"
)

type Wrapper[T any] struct {
	inner T
}

func (w Wrapper[T]) Get(ctx context.Context) T {
	ctx, span := otel.Tracer("").Start(ctx, "service.Wrapper[...].Get")
	defer span.End()

	// return the inner value
	return w.inner
}
