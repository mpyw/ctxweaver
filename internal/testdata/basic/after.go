package basic

import (
	"context"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func ProcessData(ctx context.Context, data string) error {
	defer newrelic.FromContext(ctx).StartSegment("basic.ProcessData").End() //ctxweaver:generated

	// business logic
	return nil
}
