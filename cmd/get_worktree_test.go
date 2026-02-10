package cmd_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/duboisf/linear/cmd"
)

// --- Mock GitWorktreeCreator ---

type fetchCall struct {
	remote, branch string
}

type createCall struct {
	path, branch, startPoint string
}

type mockGitWorktreeCreator struct {
	repoRoot    string
	repoRootErr error
	fetchErr    error
	createErr   error
	fetchCalls  []fetchCall
	createCalls []createCall
}

func (m *mockGitWorktreeCreator) RepoRootDir() (string, error) {
	return m.repoRoot, m.repoRootErr
}

func (m *mockGitWorktreeCreator) FetchBranch(remote, branch string) error {
	m.fetchCalls = append(m.fetchCalls, fetchCall{remote, branch})
	return m.fetchErr
}

func (m *mockGitWorktreeCreator) CreateWorktree(path, branch, startPoint string) error {
	m.createCalls = append(m.createCalls, createCall{path, branch, startPoint})
	return m.createErr
}

// --- Tests ---

func TestGetWorktree_MyWorktree(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": getIssueResponse,
	})

	mock := &mockGitWorktreeCreator{repoRoot: "/tmp/test-repo"}
	opts, stdout, _ := testOptionsWithBuffers(t, server)
	opts.GitWorktreeCreator = mock

	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"get", "@my", "worktree", "ENG-42"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("get @my worktree returned error: %v", err)
	}

	// Verify fetch was called correctly
	if len(mock.fetchCalls) != 1 {
		t.Fatalf("expected 1 fetch call, got %d", len(mock.fetchCalls))
	}
	if mock.fetchCalls[0].remote != "origin" || mock.fetchCalls[0].branch != "main" {
		t.Errorf("fetch call = %+v, want {origin main}", mock.fetchCalls[0])
	}

	// Verify worktree was created correctly
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

	// Verify output contains the worktree path
	output := stdout.String()
	if !strings.Contains(output, wantPath) {
		t.Errorf("output %q does not contain worktree path %q", output, wantPath)
	}
}

func TestGetWorktree_UserWorktree(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": getIssueResponse,
	})

	mock := &mockGitWorktreeCreator{repoRoot: "/tmp/test-repo"}
	opts, stdout, _ := testOptionsWithBuffers(t, server)
	opts.GitWorktreeCreator = mock

	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"get", "marc", "worktree", "ENG-42"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("get marc worktree returned error: %v", err)
	}

	// Verify the same git operations happen regardless of user arg
	if len(mock.fetchCalls) != 1 {
		t.Fatalf("expected 1 fetch call, got %d", len(mock.fetchCalls))
	}
	if len(mock.createCalls) != 1 {
		t.Fatalf("expected 1 create call, got %d", len(mock.createCalls))
	}

	wantPath := "/tmp/test-repo--eng-42"
	if mock.createCalls[0].path != wantPath {
		t.Errorf("create path = %q, want %q", mock.createCalls[0].path, wantPath)
	}

	output := stdout.String()
	if !strings.Contains(output, wantPath) {
		t.Errorf("output %q does not contain worktree path %q", output, wantPath)
	}
}

func TestGetWorktree_IssueNotFound(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": getIssueNullResponse,
	})

	mock := &mockGitWorktreeCreator{repoRoot: "/tmp/test-repo"}
	opts, _, _ := testOptionsWithBuffers(t, server)
	opts.GitWorktreeCreator = mock

	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"get", "@my", "worktree", "NONEXIST-1"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for null issue response")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q should contain 'not found'", err.Error())
	}
}

func TestGetWorktree_APIError(t *testing.T) {
	t.Parallel()

	server := newErrorGraphQLServer(t)
	mock := &mockGitWorktreeCreator{repoRoot: "/tmp/test-repo"}
	opts, _, _ := testOptionsWithBuffers(t, server)
	opts.GitWorktreeCreator = mock

	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"get", "@my", "worktree", "ENG-1"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error from API failure")
	}
	if !strings.Contains(err.Error(), "getting issue") {
		t.Errorf("error %q should contain 'getting issue'", err.Error())
	}
}

func TestGetWorktree_ResolveClientError(t *testing.T) {
	t.Parallel()

	opts, _, _ := testOptionsKeyringError(t)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"get", "@my", "worktree", "ENG-1"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when resolveClient fails")
	}
	if !strings.Contains(err.Error(), "resolving API key") {
		t.Errorf("error %q should contain 'resolving API key'", err.Error())
	}
}

func TestGetWorktree_FetchError(t *testing.T) {
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
	root.SetArgs([]string{"get", "@my", "worktree", "ENG-42"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when git fetch fails")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("error %q should contain 'connection refused'", err.Error())
	}
}

func TestGetWorktree_CreateWorktreeError(t *testing.T) {
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
	root.SetArgs([]string{"get", "@my", "worktree", "ENG-42"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when git worktree add fails")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error %q should contain 'already exists'", err.Error())
	}
}

func TestGetWorktree_RepoRootError(t *testing.T) {
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
	root.SetArgs([]string{"get", "@my", "worktree", "ENG-42"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when repo root detection fails")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("error %q should contain 'not a git repository'", err.Error())
	}
}

func TestGetWorktree_EmptyBranchName(t *testing.T) {
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
	root.SetArgs([]string{"get", "@my", "worktree", "ENG-42"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for empty branch name")
	}
	if !strings.Contains(err.Error(), "no branch name") {
		t.Errorf("error %q should contain 'no branch name'", err.Error())
	}
}

func TestGet_ValidArgsFunction_ResourcesIncludeWorktree(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	getCmd, _, _ := root.Find([]string{"get"})
	if getCmd.ValidArgsFunction == nil {
		t.Fatal("get command should have ValidArgsFunction")
	}

	completions, directive := getCmd.ValidArgsFunction(getCmd, []string{"@my"}, "")
	if directive != 4 { // cobra.ShellCompDirectiveNoFileComp
		t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp (4)", directive)
	}

	foundWorktree := false
	for _, comp := range completions {
		if strings.Contains(comp, "worktree") {
			foundWorktree = true
			break
		}
	}
	if !foundWorktree {
		t.Errorf("resource completions %v should contain 'worktree'", completions)
	}
}
