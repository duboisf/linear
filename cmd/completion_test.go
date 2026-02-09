package cmd_test

import (
	"strings"
	"testing"

	"github.com/duboisf/linear/cmd"
)

func TestCompletion_Bash(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	stdout, _, err := executeCommand(root, "completion", "bash")
	if err != nil {
		t.Fatalf("completion bash returned error: %v", err)
	}
	if !strings.Contains(stdout, "bash") || !strings.Contains(stdout, "completion") {
		// Bash completions typically contain these keywords
		if stdout == "" {
			t.Error("expected non-empty bash completion output")
		}
	}
}

func TestCompletion_Zsh(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	stdout, _, err := executeCommand(root, "completion", "zsh")
	if err != nil {
		t.Fatalf("completion zsh returned error: %v", err)
	}
	if stdout == "" {
		t.Error("expected non-empty zsh completion output")
	}
}

func TestCompletion_InvalidShell(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	_, _, err := executeCommand(root, "completion", "fish")
	if err == nil {
		t.Fatal("expected error for unsupported shell")
	}
	if !strings.Contains(err.Error(), "unsupported shell") {
		t.Errorf("error %q should mention unsupported shell", err.Error())
	}
}

func TestCompletion_NoArgs(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	_, _, err := executeCommand(root, "completion")
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}
