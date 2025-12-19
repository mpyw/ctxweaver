package handler

import (
	"context"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type Handler struct {
	name string
}

func (h Handler) Handle(ctx context.Context) error {
	span, ctx := tracer.StartSpanFromContext(ctx, "handler.Handler.Handle")
	defer span.Finish()

	// handle request
	return nil
}

func (h Handler) String(ctx context.Context) string {
	span, ctx := tracer.StartSpanFromContext(ctx, "handler.Handler.String")
	defer span.Finish()

	return h.name
}
