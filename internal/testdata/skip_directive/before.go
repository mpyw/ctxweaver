package skip

import "context"

func ProcessWithTrace(ctx context.Context) error {
	// should be modified
	return nil
}

//ctxweaver:skip
func LegacyHandler(ctx context.Context) error {
	// should NOT be modified
	return nil
}

func AnotherFunc(ctx context.Context) error {
	// should be modified
	return nil
}
