package format_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/duboisf/linear/internal/format"
)

func TestColorConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		constant string
		want     string
	}{
		{name: "Reset", constant: format.Reset, want: "\033[0m"},
		{name: "Bold", constant: format.Bold, want: "\033[1m"},
		{name: "Red", constant: format.Red, want: "\033[31m"},
		{name: "Green", constant: format.Green, want: "\033[32m"},
		{name: "Yellow", constant: format.Yellow, want: "\033[33m"},
		{name: "Cyan", constant: format.Cyan, want: "\033[36m"},
		{name: "Gray", constant: format.Gray, want: "\033[90m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.constant != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.want)
			}
		})
	}
}

func TestColorEnabled_NoColorEnvSet(t *testing.T) {
	// DO NOT use t.Parallel(): this test modifies the process environment.
	t.Setenv("NO_COLOR", "1")

	if format.ColorEnabled(os.Stdout) {
		t.Error("ColorEnabled() = true, want false when NO_COLOR is set")
	}
}

func TestColorEnabled_NoColorEnvEmpty(t *testing.T) {
	// NO_COLOR spec says presence matters, not value; empty string still disables.
	t.Setenv("NO_COLOR", "")

	if format.ColorEnabled(os.Stdout) {
		t.Error("ColorEnabled() = true, want false when NO_COLOR is set (even empty)")
	}
}

func TestColorEnabled_NoTerminal(t *testing.T) {
	// Unset NO_COLOR so that branch is skipped. In test mode, stdout is
	// typically a pipe (not a terminal), so ColorEnabled should return false.
	os.Unsetenv("NO_COLOR")

	// We cannot guarantee the test runner stdout is NOT a terminal in every
	// environment, but in CI / test harnesses it almost always is not.
	// We just call the function to exercise the code path.
	_ = format.ColorEnabled(os.Stdout)
}

func TestColorEnabled_NonFileWriter(t *testing.T) {
	// A bytes.Buffer does not have Fd(), so ColorEnabled should return false.
	os.Unsetenv("NO_COLOR")

	var buf bytes.Buffer
	if format.ColorEnabled(&buf) {
		t.Error("ColorEnabled(bytes.Buffer) = true, want false for non-file writer")
	}
}

func TestColorize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		enabled bool
		code    string
		text    string
		want    string
	}{
		{
			name:    "enabled wraps text in ANSI codes",
			enabled: true,
			code:    format.Red,
			text:    "error",
			want:    "\033[31m" + "error" + "\033[0m",
		},
		{
			name:    "enabled with bold",
			enabled: true,
			code:    format.Bold,
			text:    "TITLE",
			want:    "\033[1m" + "TITLE" + "\033[0m",
		},
		{
			name:    "enabled with green",
			enabled: true,
			code:    format.Green,
			text:    "success",
			want:    "\033[32m" + "success" + "\033[0m",
		},
		{
			name:    "disabled returns plain text",
			enabled: false,
			code:    format.Red,
			text:    "error",
			want:    "error",
		},
		{
			name:    "disabled with bold returns plain text",
			enabled: false,
			code:    format.Bold,
			text:    "TITLE",
			want:    "TITLE",
		},
		{
			name:    "enabled with empty text",
			enabled: true,
			code:    format.Red,
			text:    "",
			want:    "\033[31m" + "\033[0m",
		},
		{
			name:    "disabled with empty text",
			enabled: false,
			code:    format.Red,
			text:    "",
			want:    "",
		},
		{
			name:    "enabled with empty code",
			enabled: true,
			code:    "",
			text:    "hello",
			want:    "hello" + "\033[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := format.Colorize(tt.enabled, tt.code, tt.text)
			if got != tt.want {
				t.Errorf("Colorize(%v, %q, %q) = %q, want %q", tt.enabled, tt.code, tt.text, got, tt.want)
			}
		})
	}
}
