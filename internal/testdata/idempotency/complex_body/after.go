package test

import (
	"context"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func Foo(ctx context.Context) error {
	defer newrelic.FromContext(ctx).StartSegment("test.Foo").End()

	if true {
		for i := 0; i < 10; i++ {
			switch i {
			case 1:
				break
			}
		}
	}
	return nil
}
