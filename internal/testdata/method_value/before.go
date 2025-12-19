package handler

import (
	"context"
)

type Handler struct {
	name string
}

func (h Handler) Handle(ctx context.Context) error {

	// handle request
	return nil
}

func (h Handler) String(ctx context.Context) string {

	return h.name
}
