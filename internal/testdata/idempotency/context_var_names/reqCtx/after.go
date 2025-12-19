package test

import (
	"context"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func Foo(reqCtx context.Context) error {
	defer newrelic.FromContext(reqCtx).StartSegment("test.Foo").End()

	return nil
}
