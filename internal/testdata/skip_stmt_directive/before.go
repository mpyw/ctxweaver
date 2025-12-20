package test

import (
	"context"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func Foo(ctx context.Context) error {
	// ctxweaver:skip
	defer newrelic.FromContext(ctx).StartSegment("manually.Added").End()

	return nil
}
