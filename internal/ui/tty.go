package ui

import (
	"os"
	"strconv"

	"golang.org/x/term"
)

var noColor = os.Getenv("NO_COLOR") != ""

// IsTTY returns true if stdout is a terminal.
func IsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// TerminalWidth returns the width of the terminal, defaulting to 80.
func TerminalWidth() int {
	if !IsTTY() {
		return 80
	}
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80
	}
	return w
}

// SupportsColor returns true if the terminal supports color output.
func SupportsColor() bool {
	if noColor {
		return false
	}
	if !IsTTY() {
		return false
	}
	// Check TERM for color support
	termEnv := os.Getenv("TERM")
	if termEnv == "dumb" {
		return false
	}
	if termEnv != "" {
		return true
	}
	// Check COLORTERM
	ct := os.Getenv("COLORTERM")
	return ct == "truecolor" || ct == "24bit" || ct == "yes"
}

// Colorize wraps text in ANSI color codes if color is supported.
func Colorize(text, ansiCode string) string {
	if !SupportsColor() {
		return text
	}
	return ansiCode + text + "\033[0m"
}

// Color constants
const (
	ColorReset   = "\033[0m"
	ColorBold    = "\033[1m"
	ColorCyan    = "\033[36m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorRed     = "\033[31m"
	ColorGray    = "\033[90m"
	ColorMagenta = "\033[35m"
)

// ReadInt reads an integer from a string with a default fallback.
func ReadInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return n
}
