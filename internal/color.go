// Package internal provides shared utilities for ctxweaver.
package internal

import (
	"os"

	"golang.org/x/term"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorCyan   = "\033[36m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorRed    = "\033[31m"
	ColorDim    = "\033[2m"
)

var (
	stdoutIsTTY = term.IsTerminal(int(os.Stdout.Fd()))
	stderrIsTTY = term.IsTerminal(int(os.Stderr.Fd()))
)

// StdoutColor returns the color code if stdout is a TTY, otherwise empty string.
func StdoutColor(color string) string {
	if stdoutIsTTY {
		return color
	}
	return ""
}

// StderrColor returns the color code if stderr is a TTY, otherwise empty string.
func StderrColor(color string) string {
	if stderrIsTTY {
		return color
	}
	return ""
}
