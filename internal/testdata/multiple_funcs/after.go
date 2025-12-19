package multi

import (
	"context"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func First(ctx context.Context) error {
	txn := newrelic.FromContext(ctx)
	seg := txn.StartSegment("multi.First")
	defer seg.End()

	return nil
}

func Second(ctx context.Context, value int) (int, error) {
	txn := newrelic.FromContext(ctx)
	seg := txn.StartSegment("multi.Second")
	defer seg.End()

	return value * 2, nil
}

func Third(ctx context.Context, a, b string) string {
	txn := newrelic.FromContext(ctx)
	seg := txn.StartSegment("multi.Third")
	defer seg.End()

	return a + b
}
