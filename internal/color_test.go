package internal

import (
	"testing"
)

func TestStdoutColor(t *testing.T) {
	orig := stdoutIsTTY
	defer func() { stdoutIsTTY = orig }()

	t.Run("TTY", func(t *testing.T) {
		stdoutIsTTY = true
		if got := StdoutColor(ColorGreen); got != ColorGreen {
			t.Errorf("StdoutColor(ColorGreen) = %q, want %q", got, ColorGreen)
		}
	})

	t.Run("not TTY", func(t *testing.T) {
		stdoutIsTTY = false
		if got := StdoutColor(ColorGreen); got != "" {
			t.Errorf("StdoutColor(ColorGreen) = %q, want empty", got)
		}
	})
}

func TestStderrColor(t *testing.T) {
	orig := stderrIsTTY
	defer func() { stderrIsTTY = orig }()

	t.Run("TTY", func(t *testing.T) {
		stderrIsTTY = true
		if got := StderrColor(ColorRed); got != ColorRed {
			t.Errorf("StderrColor(ColorRed) = %q, want %q", got, ColorRed)
		}
	})

	t.Run("not TTY", func(t *testing.T) {
		stderrIsTTY = false
		if got := StderrColor(ColorRed); got != "" {
			t.Errorf("StderrColor(ColorRed) = %q, want empty", got)
		}
	})
}
