package cmd_test

import (
	"strings"
	"testing"

	"github.com/duboisf/linear/cmd"
)

func TestGet_MyIssue(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": getIssueResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"get", "@my", "issue", "ENG-42"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("get @my issue returned error: %v", err)
	}

	output := stdout.String()
	checks := []string{
		"ENG-42",
		"Implement feature X",
		"In Progress",
		"Jane Doe",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("output does not contain %q", check)
		}
	}
}

func TestGet_UserIssue(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": getIssueResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"get", "marc", "issue", "ENG-42"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("get marc issue returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "ENG-42") {
		t.Error("expected output to contain ENG-42")
	}
	if !strings.Contains(output, "Implement feature X") {
		t.Error("expected output to contain 'Implement feature X'")
	}
}

func TestGet_OutputMarkdown(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": getIssueResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"get", "@my", "issue", "ENG-42", "-o", "markdown"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("get @my issue -o markdown returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Field") || !strings.Contains(output, "Value") {
		t.Error("markdown output missing table header")
	}
	if !strings.Contains(output, "Identifier") || !strings.Contains(output, "ENG-42") {
		t.Error("markdown output missing Identifier row")
	}
}

func TestGet_UnsupportedResource(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"get", "@my", "project", "ENG-42"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for unsupported resource")
	}
	if !strings.Contains(err.Error(), "unsupported resource") {
		t.Errorf("error %q should contain 'unsupported resource'", err.Error())
	}
}

func TestGet_MissingArgs(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"get"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when only 1 arg provided")
	}
	if !strings.Contains(err.Error(), "accepts between 2 and 3 arg(s)") {
		t.Errorf("error %q should contain 'accepts between 2 and 3 arg(s)'", err.Error())
	}
}

func TestGet_NoIdentifier_LaunchesFzf(t *testing.T) {
	t.Parallel()

	completionResponse := `{
		"data": {
			"viewer": {
				"assignedIssues": {
					"nodes": [
						{"identifier": "ENG-1", "title": "First issue", "state": {"name": "In Progress", "type": "started"}, "priority": 2}
					]
				}
			}
		}
	}`

	server := newMockGraphQLServer(t, map[string]string{
		"ActiveIssuesForCompletion": completionResponse,
	})

	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"get", "@my", "issue"})

	err := root.Execute()
	// fzf is not available in test environment
	if err == nil {
		t.Fatal("expected error when fzf is not available in test")
	}
}

func TestGet_IssueNotFound(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": getIssueNullResponse,
	})

	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"get", "@my", "issue", "NONEXIST-1"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for null issue response")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q should contain 'not found'", err.Error())
	}
}

func TestGet_ResolveClientError(t *testing.T) {
	t.Parallel()

	opts, _, _ := testOptionsKeyringError(t)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"get", "@my", "issue", "ENG-1"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when resolveClient fails")
	}
	if !strings.Contains(err.Error(), "resolving API key") {
		t.Errorf("error %q should contain 'resolving API key'", err.Error())
	}
}

func TestGet_APIError(t *testing.T) {
	t.Parallel()

	server := newErrorGraphQLServer(t)
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"get", "@my", "issue", "ENG-1"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error from API failure")
	}
	if !strings.Contains(err.Error(), "getting issue") {
		t.Errorf("error %q should contain 'getting issue'", err.Error())
	}
}

func TestGet_ValidArgsFunction_Users(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"UsersForCompletion": usersForCompletionResponse,
	})

	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	getCmd, _, _ := root.Find([]string{"get"})
	if getCmd.ValidArgsFunction == nil {
		t.Fatal("get command should have ValidArgsFunction")
	}

	completions, directive := getCmd.ValidArgsFunction(getCmd, []string{}, "")
	if directive != 36 { // cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
		t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp|ShellCompDirectiveKeepOrder (36)", directive)
	}
	if len(completions) != 3 {
		t.Fatalf("expected 3 completions (@my + 2 users), got %d: %v", len(completions), completions)
	}
	if !strings.Contains(completions[0], "@my") {
		t.Errorf("first completion should contain '@my', got %q", completions[0])
	}
	if !strings.Contains(completions[1], "marc") {
		t.Errorf("second completion should contain 'marc', got %q", completions[1])
	}
	if !strings.Contains(completions[2], "jane") {
		t.Errorf("third completion should contain 'jane', got %q", completions[2])
	}
}

func TestGet_ValidArgsFunction_Resources(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	getCmd, _, _ := root.Find([]string{"get"})
	if getCmd.ValidArgsFunction == nil {
		t.Fatal("get command should have ValidArgsFunction")
	}

	completions, directive := getCmd.ValidArgsFunction(getCmd, []string{"@my"}, "")
	if directive != 4 {
		t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp (4)", directive)
	}
	if len(completions) != 1 {
		t.Fatalf("expected 1 completion, got %d: %v", len(completions), completions)
	}
	if !strings.Contains(completions[0], "issue") {
		t.Errorf("completion should contain 'issue', got %q", completions[0])
	}
}

func TestGet_ValidArgsFunction_MyIssueIds(t *testing.T) {
	t.Parallel()

	completionResponse := `{
		"data": {
			"viewer": {
				"assignedIssues": {
					"nodes": [
						{"identifier": "ENG-1", "title": "First issue", "state": {"name": "In Progress", "type": "started"}, "priority": 2},
						{"identifier": "ENG-2", "title": "Second issue", "state": {"name": "Todo", "type": "unstarted"}, "priority": 3}
					]
				}
			}
		}
	}`

	server := newMockGraphQLServer(t, map[string]string{
		"ActiveIssuesForCompletion": completionResponse,
	})

	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	getCmd, _, _ := root.Find([]string{"get"})
	if getCmd.ValidArgsFunction == nil {
		t.Fatal("get command should have ValidArgsFunction")
	}

	completions, directive := getCmd.ValidArgsFunction(getCmd, []string{"@my", "issue"}, "")
	if directive != 36 { // cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
		t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp|ShellCompDirectiveKeepOrder (36)", directive)
	}
	// 1 ActiveHelp header + 2 issue completions
	if len(completions) != 3 {
		t.Fatalf("expected 3 completions (1 header + 2 issues), got %d: %v", len(completions), completions)
	}
	if !strings.Contains(completions[1], "ENG-1") || !strings.Contains(completions[1], "In Progress") {
		t.Errorf("completion should contain 'ENG-1' and status, got %q", completions[1])
	}
	if !strings.Contains(completions[2], "ENG-2") || !strings.Contains(completions[2], "Todo") {
		t.Errorf("completion should contain 'ENG-2' and status, got %q", completions[2])
	}
}

func TestGet_ValidArgsFunction_UserIssueIds(t *testing.T) {
	t.Parallel()

	userCompletionResponse := `{
		"data": {
			"issues": {
				"nodes": [
					{"identifier": "ENG-201", "title": "Marc's task", "state": {"name": "In Review", "type": "started"}, "priority": 1},
					{"identifier": "ENG-202", "title": "Marc's other task", "state": {"name": "Backlog", "type": "backlog"}, "priority": 4}
				]
			}
		}
	}`

	server := newMockGraphQLServer(t, map[string]string{
		"UserIssuesForCompletion": userCompletionResponse,
	})

	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	getCmd, _, _ := root.Find([]string{"get"})
	if getCmd.ValidArgsFunction == nil {
		t.Fatal("get command should have ValidArgsFunction")
	}

	completions, directive := getCmd.ValidArgsFunction(getCmd, []string{"marc", "issue"}, "")
	if directive != 36 { // cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
		t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp|ShellCompDirectiveKeepOrder (36)", directive)
	}
	// 1 ActiveHelp header + 2 issue completions
	if len(completions) != 3 {
		t.Fatalf("expected 3 completions (1 header + 2 issues), got %d: %v", len(completions), completions)
	}
	if !strings.Contains(completions[1], "ENG-201") || !strings.Contains(completions[1], "Urgent") {
		t.Errorf("completion should contain 'ENG-201' and priority, got %q", completions[1])
	}
	if !strings.Contains(completions[2], "ENG-202") || !strings.Contains(completions[2], "Backlog") {
		t.Errorf("completion should contain 'ENG-202' and status, got %q", completions[2])
	}
}
