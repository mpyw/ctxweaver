package basic_datadog

import (
	"context"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func ProcessData(ctx context.Context, data string) error {
	span, ctx := tracer.StartSpanFromContext(ctx, "basic_datadog.ProcessData")
	defer span.Finish()

	// business logic
	return nil
}
