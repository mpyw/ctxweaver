package test

import (
	"context"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func Foo(ctx context.Context) error {

	defer someOtherFunc()
	return nil
}

// Dummy to use newrelic import
var _ = newrelic.Version

func someOtherFunc() {}
