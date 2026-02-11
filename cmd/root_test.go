package cmd_test

import (
	"strings"
	"testing"

	"github.com/duboisf/linear/cmd"
)

func TestRootCommand_Structure(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	if root.Use != "linear" {
		t.Errorf("root.Use = %q, want %q", root.Use, "linear")
	}
	if root.Short == "" {
		t.Error("root.Short should not be empty")
	}
}

func TestRootCommand_SubcommandsExist(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	subcommands := make(map[string]bool)
	for _, c := range root.Commands() {
		subcommands[c.Name()] = true
	}

	expected := []string{"issue", "user", "completion", "create", "version"}
	for _, name := range expected {
		if !subcommands[name] {
			t.Errorf("expected subcommand %q not found", name)
		}
	}
}

func TestRootCommand_HelpOutput(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	stdout, _, err := executeCommand(root, "--help")
	if err != nil {
		t.Fatalf("help command returned error: %v", err)
	}

	helpChecks := []string{
		"linear",
		"issue",
		"user",
		"completion",
		"create",
		"version",
		"Core Commands:",
		"Setup Commands:",
	}
	for _, check := range helpChecks {
		if !strings.Contains(stdout, check) {
			t.Errorf("help output does not contain %q", check)
		}
	}
}

func TestRootCommand_UnknownSubcommand(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	_, _, err := executeCommand(root, "nonexistent")
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}
