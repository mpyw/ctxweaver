package basic_otel

import (
	"context"

	"go.opentelemetry.io/otel"
)

func ProcessData(ctx context.Context, data string) error {
	ctx, span := otel.Tracer("").Start(ctx, "basic_otel.ProcessData")
	defer span.End()

	// business logic
	return nil
}
