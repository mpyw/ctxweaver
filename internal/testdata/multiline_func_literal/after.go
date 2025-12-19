package test

import (
	"context"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func Foo(ctx context.Context) error {
	defer func() {
		txn := newrelic.FromContext(ctx)
		defer txn.StartSegment("test.Foo").End()
	}()

	return nil
}
