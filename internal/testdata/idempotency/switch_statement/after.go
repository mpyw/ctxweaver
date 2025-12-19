package test

import (
	"context"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func Foo(ctx context.Context) error {
	switch txn := newrelic.FromContext(ctx); {
	case txn != nil:
		defer txn.StartSegment("test.Foo").End()
	}

	return nil
}
