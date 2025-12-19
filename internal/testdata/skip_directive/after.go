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
func LegacySimpleHandler(ctx context.Context) error {
	// existing business logic
	return nil
}

//ctxweaver:skip
func LegacyEnrichedHandler(ctx context.Context) error {
	defer newrelic.FromContext(ctx).StartSegment("skip.LegacyEnrichedHandler").End()

	// existing business logic
	return nil
}

func AnotherFunc(ctx context.Context) error {
	defer newrelic.FromContext(ctx).StartSegment("skip.AnotherFunc").End()

	// should be modified
	return nil
}
