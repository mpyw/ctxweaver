package test

import (
	"context"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func Foo(c context.Context) error {
	defer newrelic.FromContext(c).StartSegment("test.Foo").End()

	return nil
}
