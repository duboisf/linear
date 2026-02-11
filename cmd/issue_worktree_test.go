package cmd_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/duboisf/linear/cmd"
)

func TestIssueWorktree_Success(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": getIssueResponse,
	})

	mock := &mockGitWorktreeCreator{repoRoot: "/tmp/test-repo"}
	opts, stdout, _ := testOptionsWithBuffers(t, server)
	opts.GitWorktreeCreator = mock

	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "worktree", "ENG-42"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue worktree returned error: %v", err)
	}

	if len(mock.fetchCalls) != 1 {
		t.Fatalf("expected 1 fetch call, got %d", len(mock.fetchCalls))
	}
	if mock.fetchCalls[0].remote != "origin" || mock.fetchCalls[0].branch != "main" {
		t.Errorf("fetch call = %+v, want {origin main}", mock.fetchCalls[0])
	}

	if len(mock.createCalls) != 1 {
		t.Fatalf("expected 1 create call, got %d", len(mock.createCalls))
	}
	wantPath := "/tmp/test-repo--eng-42"
	if mock.createCalls[0].path != wantPath {
		t.Errorf("create path = %q, want %q", mock.createCalls[0].path, wantPath)
	}
	if mock.createCalls[0].branch != "feat/implement-feature-x" {
		t.Errorf("create branch = %q, want %q", mock.createCalls[0].branch, "feat/implement-feature-x")
	}
	if mock.createCalls[0].startPoint != "origin/main" {
		t.Errorf("create startPoint = %q, want %q", mock.createCalls[0].startPoint, "origin/main")
	}

	if len(mock.postCreateCalls) != 1 {
		t.Fatalf("expected 1 postCreate call, got %d", len(mock.postCreateCalls))
	}
	if mock.postCreateCalls[0].dir != wantPath {
		t.Errorf("postCreate dir = %q, want %q", mock.postCreateCalls[0].dir, wantPath)
	}

	output := stdout.String()
	if !strings.Contains(output, wantPath) {
		t.Errorf("output %q does not contain worktree path %q", output, wantPath)
	}
}

func TestIssueWorktree_Alias(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": getIssueResponse,
	})

	mock := &mockGitWorktreeCreator{repoRoot: "/tmp/test-repo"}
	opts, stdout, _ := testOptionsWithBuffers(t, server)
	opts.GitWorktreeCreator = mock

	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "wt", "ENG-42"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue wt returned error: %v", err)
	}

	wantPath := "/tmp/test-repo--eng-42"
	output := stdout.String()
	if !strings.Contains(output, wantPath) {
		t.Errorf("output %q does not contain worktree path %q", output, wantPath)
	}
}

func TestIssueWorktree_UserFlag(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": getIssueResponse,
	})

	mock := &mockGitWorktreeCreator{repoRoot: "/tmp/test-repo"}
	opts, stdout, _ := testOptionsWithBuffers(t, server)
	opts.GitWorktreeCreator = mock

	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "worktree", "--user", "marc", "ENG-42"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue worktree --user returned error: %v", err)
	}

	wantPath := "/tmp/test-repo--eng-42"
	output := stdout.String()
	if !strings.Contains(output, wantPath) {
		t.Errorf("output %q does not contain worktree path %q", output, wantPath)
	}
}

func TestIssueWorktree_IssueNotFound(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": getIssueNullResponse,
	})

	mock := &mockGitWorktreeCreator{repoRoot: "/tmp/test-repo"}
	opts, _, _ := testOptionsWithBuffers(t, server)
	opts.GitWorktreeCreator = mock

	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "worktree", "NONEXIST-1"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for null issue response")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q should contain 'not found'", err.Error())
	}
}

func TestIssueWorktree_APIError(t *testing.T) {
	t.Parallel()

	server := newErrorGraphQLServer(t)
	mock := &mockGitWorktreeCreator{repoRoot: "/tmp/test-repo"}
	opts, _, _ := testOptionsWithBuffers(t, server)
	opts.GitWorktreeCreator = mock

	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "worktree", "ENG-1"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error from API failure")
	}
	if !strings.Contains(err.Error(), "getting issue") {
		t.Errorf("error %q should contain 'getting issue'", err.Error())
	}
}

func TestIssueWorktree_FetchError(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": getIssueResponse,
	})

	mock := &mockGitWorktreeCreator{
		repoRoot: "/tmp/test-repo",
		fetchErr: fmt.Errorf("fetching origin/main: connection refused"),
	}
	opts, _, _ := testOptionsWithBuffers(t, server)
	opts.GitWorktreeCreator = mock

	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "worktree", "ENG-42"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when git fetch fails")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("error %q should contain 'connection refused'", err.Error())
	}
}

func TestIssueWorktree_CreateWorktreeError(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": getIssueResponse,
	})

	mock := &mockGitWorktreeCreator{
		repoRoot:  "/tmp/test-repo",
		createErr: fmt.Errorf("creating worktree: already exists"),
	}
	opts, _, _ := testOptionsWithBuffers(t, server)
	opts.GitWorktreeCreator = mock

	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "worktree", "ENG-42"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when git worktree add fails")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error %q should contain 'already exists'", err.Error())
	}
}

func TestIssueWorktree_PostCreateError(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": getIssueResponse,
	})

	mock := &mockGitWorktreeCreator{
		repoRoot:      "/tmp/test-repo",
		postCreateErr: fmt.Errorf("running mise trust: permission denied"),
	}
	opts, _, _ := testOptionsWithBuffers(t, server)
	opts.GitWorktreeCreator = mock

	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "worktree", "ENG-42"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when post-create hook fails")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("error %q should contain 'permission denied'", err.Error())
	}
}

func TestIssueWorktree_RepoRootError(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": getIssueResponse,
	})

	mock := &mockGitWorktreeCreator{
		repoRootErr: fmt.Errorf("getting repo root: not a git repository"),
	}
	opts, _, _ := testOptionsWithBuffers(t, server)
	opts.GitWorktreeCreator = mock

	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "worktree", "ENG-42"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when repo root detection fails")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("error %q should contain 'not a git repository'", err.Error())
	}
}

func TestIssueWorktree_EmptyBranchName(t *testing.T) {
	t.Parallel()

	emptyBranchResponse := `{
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
				"branchName": "",
				"state": {"name": "In Progress", "type": "started"},
				"assignee": {"name": "Jane Doe", "email": "jane@example.com"},
				"team": {"name": "Engineering", "key": "ENG"},
				"project": {"name": "Project Alpha"},
				"labels": {"nodes": [{"name": "bug"}]},
				"parent": {"identifier": "ENG-1", "title": "Parent Epic"}
			}
		}
	}`

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": emptyBranchResponse,
	})

	mock := &mockGitWorktreeCreator{repoRoot: "/tmp/test-repo"}
	opts, _, _ := testOptionsWithBuffers(t, server)
	opts.GitWorktreeCreator = mock

	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "worktree", "ENG-42"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for empty branch name")
	}
	if !strings.Contains(err.Error(), "no branch name") {
		t.Errorf("error %q should contain 'no branch name'", err.Error())
	}
}

func TestIssueWorktree_ResolveClientError(t *testing.T) {
	t.Parallel()

	opts, _, _ := testOptionsKeyringError(t)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "worktree", "ENG-1"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when resolveClient fails")
	}
	if !strings.Contains(err.Error(), "resolving API key") {
		t.Errorf("error %q should contain 'resolving API key'", err.Error())
	}
}

func TestIssueWorktree_ValidArgsFunction(t *testing.T) {
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
		if c.Name() == "worktree" {
			if c.ValidArgsFunction == nil {
				t.Fatal("worktree command should have ValidArgsFunction")
			}

			completions, directive := c.ValidArgsFunction(c, []string{}, "")
			if directive != 36 { // cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
				t.Errorf("directive = %d, want 36", directive)
			}
			if len(completions) != 3 {
				t.Fatalf("expected 3 completions (1 header + 2 issues), got %d: %v", len(completions), completions)
			}
			if !strings.Contains(completions[1], "ENG-1") {
				t.Errorf("second completion should contain 'ENG-1', got %q", completions[1])
			}
			if !strings.Contains(completions[2], "ENG-2") {
				t.Errorf("third completion should contain 'ENG-2', got %q", completions[2])
			}

			// With args already present, should return nil
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
	t.Fatal("worktree command not found")
}

func TestIssueWorktree_ValidArgsFunction_UserFlag(t *testing.T) {
	t.Parallel()

	userIssuesResponse := `{
		"data": {
			"issues": {
				"nodes": [
					{"identifier": "ENG-10", "title": "User issue one", "state": {"name": "In Progress", "type": "started"}, "priority": 2},
					{"identifier": "ENG-11", "title": "User issue two", "state": {"name": "Todo", "type": "unstarted"}, "priority": 1}
				]
			}
		}
	}`

	server := newMockGraphQLServer(t, map[string]string{
		"UserIssuesForCompletion": userIssuesResponse,
	})

	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	issueCmd, _, _ := root.Find([]string{"issue"})
	for _, c := range issueCmd.Commands() {
		if c.Name() == "worktree" {
			if err := c.Flags().Set("user", "marc"); err != nil {
				t.Fatalf("setting --user flag: %v", err)
			}

			completions, directive := c.ValidArgsFunction(c, []string{}, "")
			if directive != 36 {
				t.Errorf("directive = %d, want 36", directive)
			}
			if len(completions) != 3 {
				t.Fatalf("expected 3 completions (1 header + 2 issues), got %d: %v", len(completions), completions)
			}
			if !strings.Contains(completions[1], "ENG-10") {
				t.Errorf("completion should contain ENG-10, got %q", completions[1])
			}
			if !strings.Contains(completions[2], "ENG-11") {
				t.Errorf("completion should contain ENG-11, got %q", completions[2])
			}
			return
		}
	}
	t.Fatal("worktree command not found")
}

func TestIssueWorktree_UserFlagCompletion(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"UsersForCompletion": usersForCompletionResponse,
	})

	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	stdout, _, err := executeCommand(root, "__complete", "issue", "worktree", "--user", "")
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
