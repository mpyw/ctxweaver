package skip

import (
	"context"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func ProcessWithTrace(ctx context.Context) error {
	defer newrelic.FromContext(ctx).StartSegment("skip.ProcessWithTrace").End()

	// should be modified
	return nil
}

//ctxweaver:skip
func LegacyHandler(ctx context.Context) error {
	// should NOT be modified
	return nil
}

func AnotherFunc(ctx context.Context) error {
	defer newrelic.FromContext(ctx).StartSegment("skip.AnotherFunc").End()

	// should be modified
	return nil
}
