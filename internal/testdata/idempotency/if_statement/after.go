package test

import (
	"context"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func Foo(ctx context.Context) error {
	if txn := newrelic.FromContext(ctx); txn != nil {
		defer txn.StartSegment("test.Foo").End()
	}

	return nil
}
