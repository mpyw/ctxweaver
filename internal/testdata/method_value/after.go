package handler

import (
	"context"
	"github.com/newrelic/go-agent/v3/newrelic"
)

type Handler struct {
	name string
}

func (h Handler) Handle(ctx context.Context) error {
	defer newrelic.FromContext(ctx).StartSegment("handler.Handler.Handle").End() //ctxweaver:generated

	// handle request
	return nil
}

func (h Handler) String(ctx context.Context) string {
	defer newrelic.FromContext(ctx).StartSegment("handler.Handler.String").End() //ctxweaver:generated

	return h.name
}
