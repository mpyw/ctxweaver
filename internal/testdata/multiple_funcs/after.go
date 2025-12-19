package multi

import (
	"context"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func First(ctx context.Context) error {
	defer newrelic.FromContext(ctx).StartSegment("multi.First").End() //ctxweaver:generated

	return nil
}

func Second(ctx context.Context, value int) (int, error) {
	defer newrelic.FromContext(ctx).StartSegment("multi.Second").End() //ctxweaver:generated

	return value * 2, nil
}

func Third(ctx context.Context, a, b string) string {
	defer newrelic.FromContext(ctx).StartSegment("multi.Third").End() //ctxweaver:generated

	return a + b
}
