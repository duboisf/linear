package cmd_test

import (
	"strings"
	"testing"
	"time"

	"github.com/duboisf/linear/cmd"
	"github.com/duboisf/linear/internal/cache"
)

const updateIssueCycleResponse = `{
	"data": {
		"issueUpdate": {
			"success": true,
			"issue": {
				"id": "issue-1",
				"identifier": "ENG-42",
				"title": "Implement feature X"
			}
		}
	}
}`

const updateIssueCycleFailedResponse = `{
	"data": {
		"issueUpdate": {
			"success": false,
			"issue": null
		}
	}
}`

func TestIssueEdit_SetCycleCurrent(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue":         getIssueResponse,
		"ListCycles":       listCyclesResponse,
		"UpdateIssueCycle": updateIssueCycleResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	opts.TimeNow = func() time.Time { return time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC) }
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "edit", "ENG-42", "--cycle", "current"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue edit returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "ENG-42") {
		t.Errorf("output should contain issue identifier, got %q", output)
	}
	if !strings.Contains(output, "Cycle 11") {
		t.Errorf("output should contain cycle number, got %q", output)
	}
}

func TestIssueEdit_SetCycleByNumber(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue":         getIssueResponse,
		"ListCycles":       listCyclesResponse,
		"UpdateIssueCycle": updateIssueCycleResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	opts.TimeNow = func() time.Time { return time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC) }
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "edit", "ENG-42", "--cycle", "12"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue edit returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Cycle 12") {
		t.Errorf("output should contain cycle number 12, got %q", output)
	}
}

func TestIssueEdit_NoCycleFlag(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{})
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "edit", "ENG-42"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when no edit flag is provided")
	}
	if !strings.Contains(err.Error(), "at least one edit flag") {
		t.Errorf("error %q should mention missing edit flag", err.Error())
	}
}

func TestIssueEdit_IssueNotFound(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue":   getIssueNullResponse,
		"ListCycles": listCyclesResponse,
	})

	opts, _, _ := testOptionsWithBuffers(t, server)
	opts.TimeNow = func() time.Time { return time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC) }
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "edit", "NONEXIST-1", "--cycle", "current"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for not found issue")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q should contain 'not found'", err.Error())
	}
}

func TestIssueEdit_UpdateFailed(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue":         getIssueResponse,
		"ListCycles":       listCyclesResponse,
		"UpdateIssueCycle": updateIssueCycleFailedResponse,
	})

	opts, _, _ := testOptionsWithBuffers(t, server)
	opts.TimeNow = func() time.Time { return time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC) }
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "edit", "ENG-42", "--cycle", "current"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for failed update")
	}
	if !strings.Contains(err.Error(), "not successful") {
		t.Errorf("error %q should contain 'not successful'", err.Error())
	}
}

func TestIssueEdit_ResolveClientError(t *testing.T) {
	t.Parallel()

	opts, _, _ := testOptionsKeyringError(t)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "edit", "ENG-1", "--cycle", "current"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when resolveClient fails")
	}
	if !strings.Contains(err.Error(), "resolving API key") {
		t.Errorf("error %q should contain 'resolving API key'", err.Error())
	}
}

func TestIssueEdit_InvalidCycleValue(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue":   getIssueResponse,
		"ListCycles": listCyclesResponse,
	})

	opts, _, _ := testOptionsWithBuffers(t, server)
	opts.TimeNow = func() time.Time { return time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC) }
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "edit", "ENG-42", "--cycle", "bogus"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for invalid cycle value")
	}
	if !strings.Contains(err.Error(), "invalid --cycle value") {
		t.Errorf("error %q should contain 'invalid --cycle value'", err.Error())
	}
}

func TestIssueEdit_APIError(t *testing.T) {
	t.Parallel()

	server := newErrorGraphQLServer(t)
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "edit", "ENG-1", "--cycle", "current"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error from API failure")
	}
}

func TestIssueEdit_Alias(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue":         getIssueResponse,
		"ListCycles":       listCyclesResponse,
		"UpdateIssueCycle": updateIssueCycleResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	opts.TimeNow = func() time.Time { return time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC) }
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "e", "ENG-42", "--cycle", "current"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue e (alias) returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "ENG-42") {
		t.Errorf("output should contain issue identifier, got %q", output)
	}
}

func TestIssueEdit_ValidArgsFunction(t *testing.T) {
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

	issueCmd, _, _ := root.Find([]string{"issue"})
	for _, c := range issueCmd.Commands() {
		if c.Name() == "edit" {
			if c.ValidArgsFunction == nil {
				t.Fatal("edit command should have ValidArgsFunction")
			}

			completions, directive := c.ValidArgsFunction(c, []string{}, "")
			if directive != 36 { // cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
				t.Errorf("directive = %d, want 36", directive)
			}
			if len(completions) != 3 {
				t.Fatalf("expected 3 completions (1 header + 2 issues), got %d: %v", len(completions), completions)
			}
			if !strings.Contains(completions[1], "ENG-1") {
				t.Errorf("completion should contain ENG-1, got %q", completions[1])
			}

			// With arg already provided, should return nil.
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
	t.Fatal("edit command not found")
}

func TestIssueEdit_CycleFlagCompletion(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListCycles": listCyclesResponse,
	})

	opts := testOptions(t, server)
	opts.TimeNow = func() time.Time { return time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC) }
	opts.Cache = cache.New(t.TempDir(), 5*time.Minute)
	root := cmd.NewRootCmd(opts)

	stdout, _, err := executeCommand(root, "__complete", "issue", "edit", "--cycle", "")
	if err != nil {
		t.Fatalf("completion returned error: %v", err)
	}

	for _, want := range []string{"current", "next", "previous", "all"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("--cycle completion should contain %q, got %q", want, stdout)
		}
	}
}

func TestIssueEdit_UserFlagCompletion(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"UsersForCompletion": usersForCompletionResponse,
	})

	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	stdout, _, err := executeCommand(root, "__complete", "issue", "edit", "--user", "")
	if err != nil {
		t.Fatalf("completion returned error: %v", err)
	}

	if !strings.Contains(stdout, "marc") {
		t.Errorf("--user completion should contain 'marc', got %q", stdout)
	}
	if !strings.Contains(stdout, "jane") {
		t.Errorf("--user completion should contain 'jane', got %q", stdout)
	}
}
