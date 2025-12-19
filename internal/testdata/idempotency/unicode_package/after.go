package test

import (
	"context"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func 処理(ctx context.Context) error {
	defer newrelic.FromContext(ctx).StartSegment("test.処理").End()

	return nil
}
