// Package otel is a stub for testing.
package otel

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

func Tracer(name string, opts ...any) trace.Tracer {
	return trace.Tracer{}
}

func GetTracerProvider() trace.TracerProvider {
	return trace.TracerProvider{}
}

func SetTracerProvider(tp trace.TracerProvider) {}

type Span = trace.Span

func SpanFromContext(ctx context.Context) Span {
	return trace.Span{}
}
