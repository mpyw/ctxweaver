package internal

import (
	"testing"
)

func TestStdoutColor(t *testing.T) {
	orig := stdoutColorable
	defer func() { stdoutColorable = orig }()

	t.Run("TTY", func(t *testing.T) {
		stdoutColorable = true
		if got := StdoutColor(ColorGreen); got != ColorGreen {
			t.Errorf("StdoutColor(ColorGreen) = %q, want %q", got, ColorGreen)
		}
	})

	t.Run("not TTY", func(t *testing.T) {
		stdoutColorable = false
		if got := StdoutColor(ColorGreen); got != "" {
			t.Errorf("StdoutColor(ColorGreen) = %q, want empty", got)
		}
	})
}

func TestStderrColor(t *testing.T) {
	orig := stderrColorable
	defer func() { stderrColorable = orig }()

	t.Run("TTY", func(t *testing.T) {
		stderrColorable = true
		if got := StderrColor(ColorRed); got != ColorRed {
			t.Errorf("StderrColor(ColorRed) = %q, want %q", got, ColorRed)
		}
	})

	t.Run("not TTY", func(t *testing.T) {
		stderrColorable = false
		if got := StderrColor(ColorRed); got != "" {
			t.Errorf("StderrColor(ColorRed) = %q, want empty", got)
		}
	})
}
