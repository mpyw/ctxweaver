// Package echo is a stub for testing.
package echo

import "context"

type Context interface {
	Request() *Request
	Param(name string) string
}

type Request struct{}

func (r *Request) Context() context.Context {
	return context.Background()
}
