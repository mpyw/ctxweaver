package test

import (
	"context"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func Foo(ctx context.Context) error {
	defer newrelic.FromContext(ctx).StartSegment("test.Foo").End()

	defer someOtherFunc()
	return nil
}

// Dummy to use newrelic import
var _ = newrelic.Version

func someOtherFunc() {}
