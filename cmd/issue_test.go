package cmd_test

import (
	"strings"
	"testing"

	"github.com/duboisf/linear/cmd"
)

func TestIssueCommand_Exists(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	issueCmd, _, err := root.Find([]string{"issue"})
	if err != nil {
		t.Fatalf("finding issue command: %v", err)
	}
	if issueCmd == nil {
		t.Fatal("issue command not found")
	}
	if issueCmd.Use != "issue" {
		t.Errorf("issue.Use = %q, want %q", issueCmd.Use, "issue")
	}
}

func TestIssueCommand_Alias(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	// The "i" alias should work
	stdout, _, err := executeCommand(root, "i", "--help")
	if err != nil {
		t.Fatalf("issue alias command returned error: %v", err)
	}
	if !strings.Contains(stdout, "issue") {
		t.Error("alias 'i' should show issue help")
	}
}

func TestIssueCommand_Subcommands(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	issueCmd, _, _ := root.Find([]string{"issue"})
	subcommands := make(map[string]bool)
	for _, c := range issueCmd.Commands() {
		subcommands[c.Name()] = true
	}

	expected := []string{"list", "get"}
	for _, name := range expected {
		if !subcommands[name] {
			t.Errorf("expected issue subcommand %q not found", name)
		}
	}
}
