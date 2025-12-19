// Package tracer is a stub for testing.
package tracer

import "context"

type Span struct{}

func (s Span) Finish(opts ...any) {}

func StartSpanFromContext(ctx context.Context, operationName string, opts ...any) (Span, context.Context) {
	return Span{}, ctx
}
