// @formatter:off
package test

import (
	"context"
	"fmt"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func Foo(ctx context.Context) error {
	defer newrelic.FromContext(ctx).StartSegment("test.Foo").End()

	x := 1
	y := 2
	if x > 0 {
		fmt.Println(y)
	}
	return nil
}
