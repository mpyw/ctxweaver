package multi

import (
	"context"
)

func First(ctx context.Context) error {

	return nil
}

func Second(ctx context.Context, value int) (int, error) {

	return value * 2, nil
}

func Third(ctx context.Context, a, b string) string {

	return a + b
}
