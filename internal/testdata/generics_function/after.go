package service

import (
	"context"

	"go.opentelemetry.io/otel"
)

func Process[T any](ctx context.Context, value T) error {
	ctx, span := otel.Tracer("").Start(ctx, "service.Process[...]")
	defer span.End()

	// process the value
	return nil
}

func Transform[T, U any](ctx context.Context, input T) (U, error) {
	ctx, span := otel.Tracer("").Start(ctx, "service.Transform[...]")
	defer span.End()

	// transform input to output
	var zero U
	return zero, nil
}
