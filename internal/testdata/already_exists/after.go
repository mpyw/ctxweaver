package existing

import (
	"context"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func AlreadyInstrumented(ctx context.Context) error {
	defer newrelic.FromContext(ctx).StartSegment("existing.AlreadyInstrumented").End()

	// business logic
	return nil
}

func NeedsUpdate(ctx context.Context) error {
	defer newrelic.FromContext(ctx).StartSegment("existing.NeedsUpdate").End()

	// name is wrong, should be updated
	return nil
}
