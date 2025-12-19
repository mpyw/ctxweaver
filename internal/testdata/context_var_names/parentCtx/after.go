package test

import (
	"context"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func Foo(parentCtx context.Context) error {
	defer newrelic.FromContext(parentCtx).StartSegment("test.Foo").End()

	return nil
}
