package cmd_test

import (
	"strings"
	"testing"

	"github.com/duboisf/linear/cmd"
)

const listMyIssuesResponse = `{
	"data": {
		"viewer": {
			"assignedIssues": {
				"nodes": [
					{
						"id": "id-1",
						"identifier": "ENG-101",
						"title": "Fix login bug",
						"state": {
							"name": "In Progress",
							"type": "started"
						},
						"priority": 1,
						"updatedAt": "2025-01-01T00:00:00Z",
						"labels": {"nodes": []}
					},
					{
						"id": "id-2",
						"identifier": "ENG-102",
						"title": "Add dark mode",
						"state": {
							"name": "Backlog",
							"type": "backlog"
						},
						"priority": 3,
						"updatedAt": "2025-01-02T00:00:00Z",
						"labels": {"nodes": []}
					}
				],
				"pageInfo": {
					"hasNextPage": false,
					"endCursor": null
				}
			}
		}
	}
}`

const emptyListResponse = `{
	"data": {
		"viewer": {
			"assignedIssues": {
				"nodes": [],
				"pageInfo": {
					"hasNextPage": false,
					"endCursor": null
				}
			}
		}
	}
}`

func TestIssueList_DefaultFilter(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyActiveIssues": listMyIssuesResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue list returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "ENG-101") {
		t.Error("expected output to contain ENG-101")
	}
	if !strings.Contains(output, "ENG-102") {
		t.Error("expected output to contain ENG-102")
	}
	if !strings.Contains(output, "Fix login bug") {
		t.Error("expected output to contain issue title")
	}
}

func TestIssueList_AllFlag(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyAllIssues": listMyIssuesResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "--all"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue list --all returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "ENG-101") {
		t.Error("expected output to contain ENG-101")
	}
}

func TestIssueList_LimitFlag(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyActiveIssues": listMyIssuesResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "--limit", "10"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue list --limit returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "ENG-101") {
		t.Error("expected output to contain ENG-101")
	}
}

func TestIssueList_EmptyResult(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyActiveIssues": emptyListResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue list with empty result returned error: %v", err)
	}

	output := stdout.String()
	// Should still have the header
	if !strings.Contains(output, "IDENTIFIER") {
		t.Error("expected header in output even with no issues")
	}
}

func TestIssueList_APIError(t *testing.T) {
	t.Parallel()

	server := newErrorGraphQLServer(t)
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error from API failure")
	}
	if !strings.Contains(err.Error(), "listing issues") {
		t.Errorf("error %q should contain 'listing issues'", err.Error())
	}
}

func TestIssueList_Alias(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyActiveIssues": listMyIssuesResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "ls"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue ls alias returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "ENG-101") {
		t.Error("expected output from ls alias")
	}
}

func TestIssueList_LimitZero(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyActiveIssues": listMyIssuesResponse,
	})

	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "--limit", "0"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for --limit 0")
	}
	if !strings.Contains(err.Error(), "--limit must be greater than 0") {
		t.Errorf("error %q should contain '--limit must be greater than 0'", err.Error())
	}
}

func TestIssueList_LimitNegative(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyActiveIssues": listMyIssuesResponse,
	})

	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "--limit", "-5"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for negative --limit")
	}
	if !strings.Contains(err.Error(), "--limit must be greater than 0") {
		t.Errorf("error %q should contain '--limit must be greater than 0'", err.Error())
	}
}

const sortableIssuesResponse = `{
	"data": {
		"viewer": {
			"assignedIssues": {
				"nodes": [
					{
						"id": "id-1",
						"identifier": "AIS-273",
						"title": "Replace polling",
						"state": {"name": "In Progress", "type": "started"},
						"priority": 3,
						"updatedAt": "2025-01-01T00:00:00Z",
						"labels": {"nodes": []}
					},
					{
						"id": "id-2",
						"identifier": "AIS-265",
						"title": "Add middleware",
						"state": {"name": "In Progress", "type": "started"},
						"priority": 2,
						"updatedAt": "2025-01-02T00:00:00Z",
						"labels": {"nodes": []}
					},
					{
						"id": "id-3",
						"identifier": "AIS-271",
						"title": "Rotate keys",
						"state": {"name": "Todo", "type": "unstarted"},
						"priority": 4,
						"updatedAt": "2025-01-03T00:00:00Z",
						"labels": {"nodes": []}
					},
					{
						"id": "id-4",
						"identifier": "AIS-215",
						"title": "Lightweight memory",
						"state": {"name": "Todo", "type": "unstarted"},
						"priority": 2,
						"updatedAt": "2025-01-04T00:00:00Z",
						"labels": {"nodes": []}
					},
					{
						"id": "id-5",
						"identifier": "AIS-147",
						"title": "Security review",
						"state": {"name": "Backlog", "type": "backlog"},
						"priority": 3,
						"updatedAt": "2025-01-05T00:00:00Z",
						"labels": {"nodes": []}
					}
				],
				"pageInfo": {"hasNextPage": false, "endCursor": null}
			}
		}
	}
}`

func TestIssueList_SortByStatus(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyActiveIssues": sortableIssuesResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "--sort", "status"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue list --sort status returned error: %v", err)
	}

	output := stdout.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	// Skip header, check order: started (high pri first), then unstarted, then backlog
	if len(lines) < 6 {
		t.Fatalf("expected 6 lines (header + 5 issues), got %d", len(lines))
	}
	// First data line should be AIS-265 (started, High)
	if !strings.Contains(lines[1], "AIS-265") {
		t.Errorf("line 1 should be AIS-265 (started/High), got %q", lines[1])
	}
	// Second should be AIS-273 (started, Normal)
	if !strings.Contains(lines[2], "AIS-273") {
		t.Errorf("line 2 should be AIS-273 (started/Normal), got %q", lines[2])
	}
	// Third should be AIS-215 (todo, High)
	if !strings.Contains(lines[3], "AIS-215") {
		t.Errorf("line 3 should be AIS-215 (todo/High), got %q", lines[3])
	}
	// Last should be AIS-147 (backlog)
	if !strings.Contains(lines[5], "AIS-147") {
		t.Errorf("line 5 should be AIS-147 (backlog), got %q", lines[5])
	}
}

func TestIssueList_SortByPriority(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyActiveIssues": sortableIssuesResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "-s", "priority"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue list -s priority returned error: %v", err)
	}

	output := stdout.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 6 {
		t.Fatalf("expected 6 lines, got %d", len(lines))
	}
	// High priority issues first (AIS-265 and AIS-215 are both High=2)
	firstTwo := lines[1] + lines[2]
	if !strings.Contains(firstTwo, "AIS-265") || !strings.Contains(firstTwo, "AIS-215") {
		t.Errorf("first two issues should be High priority (AIS-265, AIS-215), got %q and %q", lines[1], lines[2])
	}
	// Low priority last
	if !strings.Contains(lines[5], "AIS-271") {
		t.Errorf("last issue should be AIS-271 (Low), got %q", lines[5])
	}
}

func TestIssueList_SortByTitle(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyActiveIssues": sortableIssuesResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "-s", "title"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue list -s title returned error: %v", err)
	}

	output := stdout.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 6 {
		t.Fatalf("expected 6 lines, got %d", len(lines))
	}
	// Alphabetical: Add middleware, Lightweight memory, Replace polling, Rotate keys, Security review
	if !strings.Contains(lines[1], "Add middleware") {
		t.Errorf("first issue should be 'Add middleware', got %q", lines[1])
	}
	if !strings.Contains(lines[5], "Security review") {
		t.Errorf("last issue should be 'Security review', got %q", lines[5])
	}
}

func TestIssueList_ResolveClientError(t *testing.T) {
	t.Parallel()

	opts, _, _ := testOptionsKeyringError(t)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when resolveClient fails")
	}
	if !strings.Contains(err.Error(), "resolving API key") {
		t.Errorf("error %q should contain 'resolving API key'", err.Error())
	}
}

func TestIssueList_NilViewer(t *testing.T) {
	t.Parallel()

	nullViewerResponse := `{
		"data": {
			"viewer": null
		}
	}`

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyActiveIssues": nullViewerResponse,
	})

	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for nil viewer")
	}
	if !strings.Contains(err.Error(), "no viewer data") {
		t.Errorf("error %q should contain 'no viewer data'", err.Error())
	}
}

func TestIssueList_NilAssignedIssues(t *testing.T) {
	t.Parallel()

	// Viewer is present but assignedIssues is null -- should not panic.
	nullAssignedIssuesResponse := `{
		"data": {
			"viewer": {
				"assignedIssues": null
			}
		}
	}`

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyActiveIssues": nullAssignedIssuesResponse,
	})

	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for nil assigned issues")
	}
	if !strings.Contains(err.Error(), "no assigned issues data") {
		t.Errorf("error %q should contain 'no assigned issues data'", err.Error())
	}
}

// --- --user flag tests ---

const listUserIssuesResponse = `{
	"data": {
		"issues": {
			"nodes": [
				{
					"id": "u-id-1",
					"identifier": "TEAM-201",
					"title": "Refactor auth module",
					"state": {"name": "In Progress", "type": "started"},
					"priority": 2,
					"updatedAt": "2025-02-01T00:00:00Z",
					"labels": {"nodes": [{"name": "backend"}]}
				},
				{
					"id": "u-id-2",
					"identifier": "TEAM-202",
					"title": "Update API docs",
					"state": {"name": "Todo", "type": "unstarted"},
					"priority": 3,
					"updatedAt": "2025-02-02T00:00:00Z",
					"labels": {"nodes": []}
				}
			],
			"pageInfo": {"hasNextPage": false, "endCursor": null}
		}
	}
}`

func TestIssueList_UserFlag(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListUserIssues": listUserIssuesResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "--user", "alice"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue list --user returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "TEAM-201") {
		t.Error("expected output to contain TEAM-201")
	}
	if !strings.Contains(output, "TEAM-202") {
		t.Error("expected output to contain TEAM-202")
	}
	if !strings.Contains(output, "Refactor auth module") {
		t.Error("expected output to contain issue title 'Refactor auth module'")
	}
}

func TestIssueList_UserFlag_AllFlag(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListAllUserIssues": listUserIssuesResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "--user", "bob", "--all"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue list --user --all returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "TEAM-201") {
		t.Error("expected output to contain TEAM-201")
	}
	if !strings.Contains(output, "TEAM-202") {
		t.Error("expected output to contain TEAM-202")
	}
}

func TestIssueList_UserFlag_NilIssues(t *testing.T) {
	t.Parallel()

	nullIssuesResponse := `{
		"data": {
			"issues": null
		}
	}`

	server := newMockGraphQLServer(t, map[string]string{
		"ListUserIssues": nullIssuesResponse,
	})

	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "--user", "alice"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for nil issues with --user")
	}
	if !strings.Contains(err.Error(), "no issues data") {
		t.Errorf("error %q should contain 'no issues data'", err.Error())
	}
}

func TestIssueList_UserFlag_AllFlag_NilIssues(t *testing.T) {
	t.Parallel()

	nullIssuesResponse := `{
		"data": {
			"issues": null
		}
	}`

	server := newMockGraphQLServer(t, map[string]string{
		"ListAllUserIssues": nullIssuesResponse,
	})

	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "--user", "alice", "--all"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for nil issues with --user --all")
	}
	if !strings.Contains(err.Error(), "no issues data") {
		t.Errorf("error %q should contain 'no issues data'", err.Error())
	}
}

func TestIssueList_SortByIdentifier(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyActiveIssues": sortableIssuesResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "--sort", "identifier"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue list --sort identifier returned error: %v", err)
	}

	output := stdout.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 6 {
		t.Fatalf("expected 6 lines (header + 5 issues), got %d", len(lines))
	}
	// Alphabetical by identifier: AIS-147, AIS-215, AIS-265, AIS-271, AIS-273
	if !strings.Contains(lines[1], "AIS-147") {
		t.Errorf("line 1 should be AIS-147, got %q", lines[1])
	}
	if !strings.Contains(lines[2], "AIS-215") {
		t.Errorf("line 2 should be AIS-215, got %q", lines[2])
	}
	if !strings.Contains(lines[3], "AIS-265") {
		t.Errorf("line 3 should be AIS-265, got %q", lines[3])
	}
	if !strings.Contains(lines[4], "AIS-271") {
		t.Errorf("line 4 should be AIS-271, got %q", lines[4])
	}
	if !strings.Contains(lines[5], "AIS-273") {
		t.Errorf("line 5 should be AIS-273, got %q", lines[5])
	}
}
