// Package testutil provides testing utilities for ctxweaver.
package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// GoldenTest runs a golden file test comparing before.go transformation to after.go.
func GoldenTest(t *testing.T, testdataDir string, transform func(input []byte) ([]byte, error)) {
	t.Helper()

	beforePath := filepath.Join(testdataDir, "before.go")
	afterPath := filepath.Join(testdataDir, "after.go")

	before, err := os.ReadFile(beforePath)
	if err != nil {
		t.Fatalf("failed to read before.go: %v", err)
	}

	want, err := os.ReadFile(afterPath)
	if err != nil {
		t.Fatalf("failed to read after.go: %v", err)
	}

	got, err := transform(before)
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}

	if diff := cmp.Diff(string(want), string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

// ReadTestdata reads a file from testdata directory.
func ReadTestdata(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return data
}
