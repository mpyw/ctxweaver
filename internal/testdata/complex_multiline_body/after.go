package test

import (
	"context"
	"errors"
	"fmt"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// Foo is a complex function with various control flows.
// It demonstrates handling of nested structures and comments.
func Foo(ctx context.Context, input string) (result string, err error) {
	// ==================================================
	// ctxweaver: auto-generated tracing code
	// DO NOT EDIT - this block is managed by ctxweaver
	// ==================================================
	txn := newrelic.FromContext(ctx)    // Extract transaction from context
	seg := txn.StartSegment("test.Foo") // Start a new segment
	defer seg.End()                     // End the segment when function returns
	// ==================================================
	// End of ctxweaver generated code
	// ==================================================

	// Early return with inline comment
	if input == "" {
		return "", errors.New("empty input")
	}

	// Early return with newline
	if ctx == nil {
		return "", errors.New("nil context")
	}

	/*
	 * Block comment before complex logic
	 * This should be preserved
	 */
	for i := 0; i < 10; i++ {
		// Loop iteration comment
		switch i {
		case 0:
			// First case
			fmt.Println("zero")
		case 1, 2, 3: // Multiple values
			fmt.Println("small")
		default:
			// Default case with nested if
			if i > 5 {
				fmt.Println("large") // Inline comment
			}
		}
	}

	// Nested conditionals
	if input == "special" {
		if ctx.Err() != nil {
			return "", ctx.Err()
		} // Inline early return

		// Another block comment
		result = "handled"
	} else {
		result = input
	}

	// Final return
	return result, nil
}

// Bar is another function to test multiple function handling.
func Bar(ctx context.Context) error {
	// ==================================================
	// ctxweaver: auto-generated tracing code
	// DO NOT EDIT - this block is managed by ctxweaver
	// ==================================================
	txn := newrelic.FromContext(ctx)    // Extract transaction from context
	seg := txn.StartSegment("test.Bar") // Start a new segment
	defer seg.End()                     // End the segment when function returns
	// ==================================================
	// End of ctxweaver generated code
	// ==================================================

	// Simple early return
	if ctx == nil {
		return errors.New("nil")
	}

	return nil
}
