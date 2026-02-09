package cmd_test

import (
	"strings"
	"testing"

	"github.com/duboisf/linear/cmd"
)

const listUserIssuesResponse = `{
	"data": {
		"issues": {
			"nodes": [
				{
					"id": "id-1",
					"identifier": "ENG-201",
					"title": "Marc's task",
					"state": {"name": "In Progress", "type": "started"},
					"priority": 2,
					"updatedAt": "2025-01-01T00:00:00Z",
					"labels": {"nodes": []}
				}
			],
			"pageInfo": {"hasNextPage": false, "endCursor": null}
		}
	}
}`

const usersForCompletionResponse = `{
	"data": {
		"users": {
			"nodes": [
				{"id": "u1", "name": "Marc Dupont", "displayName": "Marc Dupont"},
				{"id": "u2", "name": "Jane Smith", "displayName": "Jane Smith"}
			]
		}
	}
}`

func TestList_MyIssues(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyActiveIssues": listMyIssuesResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"list", "my", "issues"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("list my issues returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "ENG-101") {
		t.Error("expected output to contain ENG-101")
	}
}

func TestList_UserIssues(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListUserIssues": listUserIssuesResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"list", "marc", "issues"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("list marc issues returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "ENG-201") {
		t.Error("expected output to contain ENG-201")
	}
}

func TestList_MyIssues_AllFlag(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyAllIssues": listMyIssuesResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"list", "my", "issues", "--all"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("list my issues --all returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "ENG-101") {
		t.Error("expected output to contain ENG-101")
	}
}

func TestList_UserIssues_AllFlag(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListAllUserIssues": listUserIssuesResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"list", "marc", "issues", "--all"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("list marc issues --all returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "ENG-201") {
		t.Error("expected output to contain ENG-201")
	}
}

func TestList_MissingArgs(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"list"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
	if !strings.Contains(err.Error(), "accepts 2 arg(s)") {
		t.Errorf("error %q should contain 'accepts 2 arg(s)'", err.Error())
	}
}

func TestList_MissingResource(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"list", "my"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when only 1 arg provided")
	}
	if !strings.Contains(err.Error(), "accepts 2 arg(s)") {
		t.Errorf("error %q should contain 'accepts 2 arg(s)'", err.Error())
	}
}

func TestList_UnsupportedResource(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"list", "my", "projects"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for unsupported resource")
	}
	if !strings.Contains(err.Error(), "unsupported resource") {
		t.Errorf("error %q should contain 'unsupported resource'", err.Error())
	}
}

func TestList_ResolveClientError(t *testing.T) {
	t.Parallel()

	opts, _, _ := testOptionsKeyringError(t)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"list", "my", "issues"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when resolveClient fails")
	}
	if !strings.Contains(err.Error(), "resolving API key") {
		t.Errorf("error %q should contain 'resolving API key'", err.Error())
	}
}

func TestList_APIError(t *testing.T) {
	t.Parallel()

	server := newErrorGraphQLServer(t)
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"list", "my", "issues"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error from API failure")
	}
	if !strings.Contains(err.Error(), "listing issues") {
		t.Errorf("error %q should contain 'listing issues'", err.Error())
	}
}

func TestList_LimitZero(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyActiveIssues": listMyIssuesResponse,
	})

	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"list", "my", "issues", "--limit", "0"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for --limit 0")
	}
	if !strings.Contains(err.Error(), "--limit must be greater than 0") {
		t.Errorf("error %q should contain '--limit must be greater than 0'", err.Error())
	}
}

func TestList_ValidArgsFunction_Users(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"UsersForCompletion": usersForCompletionResponse,
	})

	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	listCmd, _, _ := root.Find([]string{"list"})
	if listCmd.ValidArgsFunction == nil {
		t.Fatal("list command should have ValidArgsFunction")
	}

	completions, directive := listCmd.ValidArgsFunction(listCmd, []string{}, "")
	if directive != 4 { // cobra.ShellCompDirectiveNoFileComp
		t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp (4)", directive)
	}
	if len(completions) != 3 {
		t.Fatalf("expected 3 completions (my + 2 users), got %d: %v", len(completions), completions)
	}
	if !strings.Contains(completions[0], "my") {
		t.Errorf("first completion should contain 'my', got %q", completions[0])
	}
	if !strings.Contains(completions[1], "marc") {
		t.Errorf("second completion should contain 'marc', got %q", completions[1])
	}
	if !strings.Contains(completions[2], "jane") {
		t.Errorf("third completion should contain 'jane', got %q", completions[2])
	}
}

func TestList_ValidArgsFunction_Resources(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	listCmd, _, _ := root.Find([]string{"list"})
	if listCmd.ValidArgsFunction == nil {
		t.Fatal("list command should have ValidArgsFunction")
	}

	completions, directive := listCmd.ValidArgsFunction(listCmd, []string{"my"}, "")
	if directive != 4 { // cobra.ShellCompDirectiveNoFileComp
		t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp (4)", directive)
	}
	if len(completions) != 1 {
		t.Fatalf("expected 1 completion, got %d: %v", len(completions), completions)
	}
	if !strings.Contains(completions[0], "issues") {
		t.Errorf("completion should contain 'issues', got %q", completions[0])
	}
}
