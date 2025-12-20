package service

import (
	"context"

	"go.opentelemetry.io/otel"
)

type Cache[K comparable, V any] struct {
	data map[K]V
}

func (c *Cache[K, V]) Get(ctx context.Context, key K) (V, bool) {
	ctx, span := otel.Tracer("").Start(ctx, "service.(*Cache[...]).Get")
	defer span.End()

	// get value from cache
	v, ok := c.data[key]
	return v, ok
}

func (c *Cache[K, V]) Set(ctx context.Context, key K, value V) {
	ctx, span := otel.Tracer("").Start(ctx, "service.(*Cache[...]).Set")
	defer span.End()

	// set value in cache
	c.data[key] = value
}

type Pair[A, B any] struct {
	First  A
	Second B
}

func (p Pair[A, B]) Swap(ctx context.Context) Pair[B, A] {
	ctx, span := otel.Tracer("").Start(ctx, "service.Pair[...].Swap")
	defer span.End()

	// swap the pair
	return Pair[B, A]{First: p.Second, Second: p.First}
}
