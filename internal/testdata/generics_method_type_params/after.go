package service

import (
	"context"

	"go.opentelemetry.io/otel"
)

type Registry struct {
	values map[string]any
}

func (r Registry) Lookup[T any](ctx context.Context, key string) (T, bool) {
	ctx, span := otel.Tracer("").Start(ctx, "service.Registry.Lookup[...]")
	defer span.End()

	// look up a typed value
	v, ok := r.values[key].(T)
	return v, ok
}

func (r *Registry) Register[T any](ctx context.Context, key string, value T) {
	ctx, span := otel.Tracer("").Start(ctx, "service.(*Registry).Register[...]")
	defer span.End()

	// register a typed value
	r.values[key] = value
}
