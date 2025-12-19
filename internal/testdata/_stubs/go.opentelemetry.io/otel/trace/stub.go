// Package trace is a stub for testing.
package trace

import "context"

type Tracer struct{}

func (t Tracer) Start(ctx context.Context, name string, opts ...any) (context.Context, Span) {
	return ctx, Span{}
}

type Span struct{}

func (s Span) End(opts ...any) {}

type TracerProvider struct{}

func (tp TracerProvider) Tracer(name string, opts ...any) Tracer {
	return Tracer{}
}
