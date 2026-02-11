package cmd_test

import (
	"strings"
	"testing"

	"github.com/duboisf/linear/cmd"
)

func TestCreate_UnsupportedResource(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"create", "@my", "project", "ENG-42"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for unsupported resource")
	}
	if !strings.Contains(err.Error(), "unsupported resource") {
		t.Errorf("error %q should contain 'unsupported resource'", err.Error())
	}
}

func TestCreate_MissingArgs(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"create"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when only 1 arg provided")
	}
	if !strings.Contains(err.Error(), "accepts between 2 and 3 arg(s)") {
		t.Errorf("error %q should contain 'accepts between 2 and 3 arg(s)'", err.Error())
	}
}


func TestCreate_ResolveClientError(t *testing.T) {
	t.Parallel()

	opts, _, _ := testOptionsKeyringError(t)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"create", "@my", "worktree", "ENG-1"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when resolveClient fails")
	}
	if !strings.Contains(err.Error(), "resolving API key") {
		t.Errorf("error %q should contain 'resolving API key'", err.Error())
	}
}

func TestCreate_ValidArgsFunction_Users(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"UsersForCompletion": usersForCompletionResponse,
	})

	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	createCmd, _, _ := root.Find([]string{"create"})
	if createCmd.ValidArgsFunction == nil {
		t.Fatal("create command should have ValidArgsFunction")
	}

	completions, directive := createCmd.ValidArgsFunction(createCmd, []string{}, "")
	if directive != 36 { // cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
		t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp|ShellCompDirectiveKeepOrder (36)", directive)
	}
	if len(completions) != 3 {
		t.Fatalf("expected 3 completions (@my + 2 users), got %d: %v", len(completions), completions)
	}
	if !strings.Contains(completions[0], "@my") {
		t.Errorf("first completion should contain '@my', got %q", completions[0])
	}
}

func TestCreate_ValidArgsFunction_Resources(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	createCmd, _, _ := root.Find([]string{"create"})
	if createCmd.ValidArgsFunction == nil {
		t.Fatal("create command should have ValidArgsFunction")
	}

	completions, directive := createCmd.ValidArgsFunction(createCmd, []string{"@my"}, "")
	if directive != 4 { // cobra.ShellCompDirectiveNoFileComp
		t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp (4)", directive)
	}
	if len(completions) != 1 {
		t.Fatalf("expected 1 completion, got %d: %v", len(completions), completions)
	}
	if !strings.Contains(completions[0], "worktree") {
		t.Errorf("completion should contain 'worktree', got %q", completions[0])
	}
}
