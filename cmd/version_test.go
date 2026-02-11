package cmd_test

import (
	"strings"
	"testing"

	"github.com/duboisf/linear/cmd"
)

func TestVersionCommand(t *testing.T) {
	t.Parallel()

	root := cmd.NewRootCmd(cmd.Options{})

	stdout, _, err := executeCommand(root, "version")
	if err != nil {
		t.Fatalf("version returned error: %v", err)
	}

	if !strings.Contains(stdout, "linear") {
		t.Errorf("output should contain 'linear', got %q", stdout)
	}
	if !strings.Contains(stdout, cmd.Version) {
		t.Errorf("output should contain version %q, got %q", cmd.Version, stdout)
	}
}

func TestVersionCommand_Output(t *testing.T) {
	t.Parallel()

	root := cmd.NewRootCmd(cmd.Options{})

	stdout, _, err := executeCommand(root, "version")
	if err != nil {
		t.Fatalf("version returned error: %v", err)
	}

	expected := "linear " + cmd.Version + "\n"
	if stdout != expected {
		t.Errorf("output = %q, want %q", stdout, expected)
	}
}
