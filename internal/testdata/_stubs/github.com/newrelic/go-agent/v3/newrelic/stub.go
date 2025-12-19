// Package newrelic is a stub for testing.
package newrelic

import "context"

const Version = "stub"

type Transaction struct{}

func (t *Transaction) StartSegment(name string) *Segment {
	return &Segment{}
}

type Segment struct{}

func (s *Segment) End() {}

func FromContext(ctx context.Context) *Transaction {
	return &Transaction{}
}
