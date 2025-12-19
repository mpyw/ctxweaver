package test

import (
	"context"
	"errors"
	"fmt"
)

// Foo is a complex function with various control flows.
// It demonstrates handling of nested structures and comments.
func Foo(ctx context.Context, input string) (result string, err error) {
	// Early return with inline comment
	if input == "" { return "", errors.New("empty input") }

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
		if ctx.Err() != nil { return "", ctx.Err() } // Inline early return

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
	// Simple early return
	if ctx == nil { return errors.New("nil") }

	return nil
}
