// Package internal provides shared internal utilities.
package internal

// Must panics if err is not nil, otherwise returns val.
// Use for initialization of embedded resources where failure is a build error.
func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}
