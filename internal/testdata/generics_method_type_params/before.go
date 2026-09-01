package service

import (
	"context"
)

type Registry struct {
	values map[string]any
}

func (r Registry) Lookup[T any](ctx context.Context, key string) (T, bool) {

	// look up a typed value
	v, ok := r.values[key].(T)
	return v, ok
}

func (r *Registry) Register[T any](ctx context.Context, key string, value T) {

	// register a typed value
	r.values[key] = value
}
