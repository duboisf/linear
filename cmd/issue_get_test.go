package cmd_test

import (
	"strings"
	"testing"

	"github.com/duboisf/linear/cmd"
)

const getIssueResponse = `{
	"data": {
		"issue": {
			"id": "issue-1",
			"identifier": "ENG-42",
			"title": "Implement feature X",
			"description": "Detailed description here.",
			"url": "https://linear.app/team/ENG-42",
			"priority": 2,
			"estimate": 5,
			"dueDate": "2025-12-31",
			"createdAt": "2025-01-01T00:00:00Z",
			"updatedAt": "2025-01-15T00:00:00Z",
			"branchName": "feat/implement-feature-x",
			"state": {
				"name": "In Progress",
				"type": "started"
			},
			"assignee": {
				"name": "Jane Doe",
				"email": "jane@example.com"
			},
			"team": {
				"name": "Engineering",
				"key": "ENG"
			},
			"project": {
				"name": "Project Alpha"
			},
			"labels": {
				"nodes": [
					{"name": "bug"},
					{"name": "frontend"}
				]
			},
			"parent": {
				"identifier": "ENG-1",
				"title": "Parent Epic"
			}
		}
	}
}`

const getIssueNotFoundResponse = `{
	"data": {
		"issue": null
	},
	"errors": [
		{
			"message": "Entity not found"
		}
	]
}`

// getIssueNullResponse returns null issue without GraphQL errors, which
// exercises the resp.Issue == nil branch in newIssueGetCmd.
const getIssueNullResponse = `{
	"data": {
		"issue": null
	}
}`

func TestIssueGet_Success(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": getIssueResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "get", "ENG-42"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue get returned error: %v", err)
	}

	output := stdout.String()
	checks := []string{
		"ENG-42",
		"Implement feature X",
		"In Progress",
		"High",
		"Jane Doe",
		"Engineering",
		"Project Alpha",
		"bug, frontend",
		"2025-12-31",
		"feat/implement-feature-x",
		"ENG-1 Parent Epic",
		"Detailed description here.",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("output does not contain %q", check)
		}
	}
}

func TestIssueGet_OutputJSON(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": getIssueResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "get", "ENG-42", "--output", "json"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue get --output json returned error: %v", err)
	}

	output := stdout.String()
	checks := []string{
		`"identifier": "ENG-42"`,
		`"title": "Implement feature X"`,
		`"state": "In Progress"`,
		`"priority": "High"`,
		`"assignee": "Jane Doe"`,
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("JSON output does not contain %q", check)
		}
	}
}

func TestIssueGet_OutputYAML(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": getIssueResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "get", "ENG-42", "-o", "yaml"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue get -o yaml returned error: %v", err)
	}

	output := stdout.String()
	checks := []string{
		"identifier: ENG-42",
		"state: In Progress",
		"priority: High",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("YAML output does not contain %q", check)
		}
	}
}

func TestIssueGet_OutputMarkdown(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": getIssueResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "get", "ENG-42", "-o", "markdown"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue get -o markdown returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Field") || !strings.Contains(output, "Value") {
		t.Error("markdown output missing table header")
	}
	if !strings.Contains(output, "Identifier") || !strings.Contains(output, "ENG-42") {
		t.Error("markdown output missing Identifier row")
	}
}

func TestIssueGet_NotFound(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": getIssueNotFoundResponse,
	})

	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "get", "NONEXIST-1"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for not found issue")
	}
}

func TestIssueGet_NoArgs_NonInteractive(t *testing.T) {
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
	root.SetArgs([]string{"issue", "get"})

	err := root.Execute()
	// fzf is not available in test environment
	if err == nil {
		t.Fatal("expected error when fzf is not available in test")
	}
}

func TestIssueGet_Aliases(t *testing.T) {
	t.Parallel()

	aliases := []string{"show", "view"}
	for _, alias := range aliases {
		t.Run(alias, func(t *testing.T) {
			t.Parallel()

			server := newMockGraphQLServer(t, map[string]string{
				"GetIssue": getIssueResponse,
			})

			opts, stdout, _ := testOptionsWithBuffers(t, server)
			root := cmd.NewRootCmd(opts)
			root.SetArgs([]string{"issue", alias, "ENG-42"})

			err := root.Execute()
			if err != nil {
				t.Fatalf("issue %s returned error: %v", alias, err)
			}

			output := stdout.String()
			if !strings.Contains(output, "ENG-42") {
				t.Errorf("alias %q output does not contain ENG-42", alias)
			}
		})
	}
}

func TestIssueGet_NullIssue(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": getIssueNullResponse,
	})

	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "get", "NONEXIST-1"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for null issue response")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q should contain 'not found'", err.Error())
	}
}

func TestIssueGet_ResolveClientError(t *testing.T) {
	t.Parallel()

	opts, _, _ := testOptionsKeyringError(t)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "get", "ENG-1"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when resolveClient fails")
	}
	if !strings.Contains(err.Error(), "resolving API key") {
		t.Errorf("error %q should contain 'resolving API key'", err.Error())
	}
}

func TestIssueGet_APIError(t *testing.T) {
	t.Parallel()

	server := newErrorGraphQLServer(t)
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "get", "ENG-1"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error from API failure")
	}
	if !strings.Contains(err.Error(), "getting issue") {
		t.Errorf("error %q should contain 'getting issue'", err.Error())
	}
}

func TestIssueGet_ValidArgsFunction(t *testing.T) {
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

	// Find the get command
	issueCmd, _, _ := root.Find([]string{"issue"})
	for _, c := range issueCmd.Commands() {
		if c.Name() == "get" {
			// Test that ValidArgsFunction exists
			if c.ValidArgsFunction == nil {
				t.Fatal("get command should have ValidArgsFunction")
			}

			// Call ValidArgsFunction with no args (should return completions)
			// Expect 3 entries: 1 ActiveHelp header + 2 issue completions
			completions, directive := c.ValidArgsFunction(c, []string{}, "")
			if directive != 36 { // cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder == 36
				t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp|ShellCompDirectiveKeepOrder (36)", directive)
			}
			if len(completions) != 3 {
				t.Fatalf("expected 3 completions (1 header + 2 issues), got %d: %v", len(completions), completions)
			}

			// First entry is ActiveHelp header with column titles
			if !strings.Contains(completions[0], "IDENTIFIER") || !strings.Contains(completions[0], "STATUS") || !strings.Contains(completions[0], "PRIORITY") {
				t.Errorf("first completion should be ActiveHelp header with column titles, got %q", completions[0])
			}

			// Issue completions include status and priority in description
			if !strings.Contains(completions[1], "ENG-1") || !strings.Contains(completions[1], "In Progress") || !strings.Contains(completions[1], "High") {
				t.Errorf("completion should contain identifier, status, and priority, got %q", completions[1])
			}
			if !strings.Contains(completions[2], "ENG-2") || !strings.Contains(completions[2], "Todo") || !strings.Contains(completions[2], "Normal") {
				t.Errorf("completion should contain identifier, status, and priority, got %q", completions[2])
			}

			// Call with args already present (should return nil)
			completions2, directive2 := c.ValidArgsFunction(c, []string{"ENG-1"}, "")
			if directive2 != 4 {
				t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp (4)", directive2)
			}
			if completions2 != nil {
				t.Errorf("expected nil completions when arg already provided, got %v", completions2)
			}
			return
		}
	}
	t.Fatal("get command not found")
}

func TestIssueGet_ValidArgsFunction_ResolveClientError(t *testing.T) {
	t.Parallel()

	opts, _, _ := testOptionsKeyringError(t)
	root := cmd.NewRootCmd(opts)

	issueCmd, _, _ := root.Find([]string{"issue"})
	for _, c := range issueCmd.Commands() {
		if c.Name() == "get" {
			completions, directive := c.ValidArgsFunction(c, []string{}, "")
			if directive != 4 {
				t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp (4)", directive)
			}
			if completions != nil {
				t.Errorf("expected nil completions on resolveClient error, got %v", completions)
			}
			return
		}
	}
	t.Fatal("get command not found")
}

func TestIssueGet_ValidArgsFunction_APIError(t *testing.T) {
	t.Parallel()

	server := newErrorGraphQLServer(t)
	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	issueCmd, _, _ := root.Find([]string{"issue"})
	for _, c := range issueCmd.Commands() {
		if c.Name() == "get" {
			completions, directive := c.ValidArgsFunction(c, []string{}, "")
			if directive != 4 {
				t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp (4)", directive)
			}
			if completions != nil {
				t.Errorf("expected nil completions on API error, got %v", completions)
			}
			return
		}
	}
	t.Fatal("get command not found")
}

func TestIssueGet_ValidArgsFunction_NilViewer(t *testing.T) {
	t.Parallel()

	nullViewerResponse := `{
		"data": {
			"viewer": null
		}
	}`

	server := newMockGraphQLServer(t, map[string]string{
		"ActiveIssuesForCompletion": nullViewerResponse,
	})

	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	issueCmd, _, _ := root.Find([]string{"issue"})
	for _, c := range issueCmd.Commands() {
		if c.Name() == "get" {
			completions, directive := c.ValidArgsFunction(c, []string{}, "")
			if directive != 4 {
				t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp (4)", directive)
			}
			if completions != nil {
				t.Errorf("expected nil completions on nil viewer, got %v", completions)
			}
			return
		}
	}
	t.Fatal("get command not found")
}

func TestIssueGet_ValidArgsFunction_NilAssignedIssues(t *testing.T) {
	t.Parallel()

	// Viewer present but assignedIssues is null -- should return nil completions
	// without panicking.
	nullAssignedIssuesResponse := `{
		"data": {
			"viewer": {
				"assignedIssues": null
			}
		}
	}`

	server := newMockGraphQLServer(t, map[string]string{
		"ActiveIssuesForCompletion": nullAssignedIssuesResponse,
	})

	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	issueCmd, _, _ := root.Find([]string{"issue"})
	for _, c := range issueCmd.Commands() {
		if c.Name() == "get" {
			completions, directive := c.ValidArgsFunction(c, []string{}, "")
			if directive != 4 {
				t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp (4)", directive)
			}
			if completions != nil {
				t.Errorf("expected nil completions on nil assignedIssues, got %v", completions)
			}
			return
		}
	}
	t.Fatal("get command not found")
}
