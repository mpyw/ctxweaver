package test

import "context"

func Foo(ctx context.Context) error {
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
