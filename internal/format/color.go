package format

import (
	"io"
	"os"

	"golang.org/x/term"
)

// ANSI color codes.
const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Cyan   = "\033[36m"
	Gray   = "\033[90m"
)

// fdWriter is an interface for writers that expose a file descriptor.
type fdWriter interface {
	Fd() uintptr
}

// ColorEnabled returns true if color output should be used.
// Respects NO_COLOR env var (https://no-color.org) and checks whether the
// given writer is a terminal. The writer is inspected via a type assertion
// for an Fd() method (e.g., *os.File); non-file writers are treated as
// non-TTY and will return false.
func ColorEnabled(w io.Writer) bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	if f, ok := w.(fdWriter); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

// Colorize wraps text in ANSI escape codes if enabled is true.
func Colorize(enabled bool, code, text string) string {
	if !enabled {
		return text
	}
	return code + text + Reset
}
