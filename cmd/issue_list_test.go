package cmd_test

import (
	"strings"
	"testing"
	"time"

	"github.com/duboisf/linear/cmd"
	"github.com/duboisf/linear/internal/cache"
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
		"ListMyIssues": listMyIssuesResponse,
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

func TestIssueList_StatusAll(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyIssues": listMyIssuesResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "--status", "all"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue list --status all returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "ENG-101") {
		t.Error("expected output to contain ENG-101")
	}
}

func TestIssueList_LimitFlag(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyIssues": listMyIssuesResponse,
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
		"ListMyIssues": emptyListResponse,
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
		"ListMyIssues": listMyIssuesResponse,
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
		"ListMyIssues": listMyIssuesResponse,
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
		"ListMyIssues": listMyIssuesResponse,
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
		"ListMyIssues": sortableIssuesResponse,
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
		"ListMyIssues": sortableIssuesResponse,
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
		"ListMyIssues": sortableIssuesResponse,
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
		"ListMyIssues": nullViewerResponse,
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
		"ListMyIssues": nullAssignedIssuesResponse,
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
		"ListIssues": listUserIssuesResponse,
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

func TestIssueList_UserFlag_StatusAll(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListIssues": listUserIssuesResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "--user", "bob", "--status", "all"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue list --user --status all returned error: %v", err)
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
		"ListIssues": nullIssuesResponse,
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

func TestIssueList_UserFlag_StatusAll_NilIssues(t *testing.T) {
	t.Parallel()

	nullIssuesResponse := `{
		"data": {
			"issues": null
		}
	}`

	server := newMockGraphQLServer(t, map[string]string{
		"ListIssues": nullIssuesResponse,
	})

	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "--user", "alice", "--status", "all"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for nil issues with --user --status all")
	}
	if !strings.Contains(err.Error(), "no issues data") {
		t.Errorf("error %q should contain 'no issues data'", err.Error())
	}
}

func TestIssueList_SortByIdentifier(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyIssues": sortableIssuesResponse,
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

// --- --cycle flag tests ---

const listCyclesResponse = `{
	"data": {
		"cycles": {
			"nodes": [
				{
					"id": "cycle-1",
					"number": 10,
					"name": "Sprint 10",
					"startsAt": "2025-01-01T00:00:00Z",
					"endsAt": "2025-01-14T00:00:00Z",
					"isActive": false,
					"isNext": false,
					"isPast": true,
					"isPrevious": true
				},
				{
					"id": "cycle-2",
					"number": 11,
					"name": "Sprint 11",
					"startsAt": "2025-01-15T00:00:00Z",
					"endsAt": "2025-01-28T00:00:00Z",
					"isActive": true,
					"isNext": false,
					"isPast": false,
					"isPrevious": false
				},
				{
					"id": "cycle-3",
					"number": 12,
					"name": "Sprint 12",
					"startsAt": "2025-01-29T00:00:00Z",
					"endsAt": "2025-02-11T00:00:00Z",
					"isActive": false,
					"isNext": true,
					"isPast": false,
					"isPrevious": false
				}
			]
		}
	}
}`

const cycleIssuesResponse = `{
	"data": {
		"viewer": {
			"assignedIssues": {
				"nodes": [
					{
						"id": "c-id-1",
						"identifier": "ENG-301",
						"title": "Cycle task one",
						"state": {"name": "In Progress", "type": "started"},
						"priority": 2,
						"updatedAt": "2025-01-20T00:00:00Z",
						"labels": {"nodes": []}
					},
					{
						"id": "c-id-2",
						"identifier": "ENG-302",
						"title": "Cycle task two",
						"state": {"name": "Todo", "type": "unstarted"},
						"priority": 3,
						"updatedAt": "2025-01-21T00:00:00Z",
						"labels": {"nodes": []}
					}
				],
				"pageInfo": {"hasNextPage": false, "endCursor": null}
			}
		}
	}
}`

const cycleUserIssuesResponse = `{
	"data": {
		"issues": {
			"nodes": [
				{
					"id": "cu-id-1",
					"identifier": "TEAM-401",
					"title": "User cycle task",
					"state": {"name": "In Progress", "type": "started"},
					"priority": 1,
					"updatedAt": "2025-01-22T00:00:00Z",
					"labels": {"nodes": []}
				}
			],
			"pageInfo": {"hasNextPage": false, "endCursor": null}
		}
	}
}`

func TestIssueList_CycleCurrentFlag(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListCycles":   listCyclesResponse,
		"ListMyIssues": cycleIssuesResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "--cycle", "current"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue list --cycle current returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Cycle 11") {
		t.Error("expected output to contain cycle header 'Cycle 11'")
	}
	if !strings.Contains(output, "Sprint 11") {
		t.Error("expected output to contain cycle name 'Sprint 11'")
	}
	if !strings.Contains(output, "ENG-301") {
		t.Error("expected output to contain ENG-301")
	}
	if !strings.Contains(output, "ENG-302") {
		t.Error("expected output to contain ENG-302")
	}
}

func TestIssueList_CycleNumberFlag(t *testing.T) {
	t.Parallel()

	listCyclesWithNumber42 := `{
		"data": {
			"cycles": {
				"nodes": [
					{
						"id": "cycle-42",
						"number": 42,
						"name": "Sprint 42",
						"startsAt": "2025-06-01T00:00:00Z",
						"endsAt": "2025-06-14T00:00:00Z",
						"isActive": false,
						"isNext": false,
						"isPast": true,
						"isPrevious": false
					}
				]
			}
		}
	}`

	server := newMockGraphQLServer(t, map[string]string{
		"ListCycles":   listCyclesWithNumber42,
		"ListMyIssues": cycleIssuesResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "-c", "42"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue list -c 42 returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Cycle 42") {
		t.Error("expected output to contain cycle header 'Cycle 42'")
	}
	if !strings.Contains(output, "ENG-301") {
		t.Error("expected output to contain ENG-301")
	}
}

func TestIssueList_CycleWithUserFlag(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListCycles": listCyclesResponse,
		"ListIssues": cycleUserIssuesResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "-c", "current", "--user", "alice"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue list -c current --user alice returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "TEAM-401") {
		t.Error("expected output to contain TEAM-401")
	}
}

func TestIssueList_CycleWithStatusAll(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListCycles":   listCyclesResponse,
		"ListMyIssues": cycleIssuesResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "-c", "current", "--status", "all"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue list -c current --status all returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "ENG-301") {
		t.Error("expected output to contain ENG-301")
	}
}

func TestIssueList_CycleInvalidValue(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{})

	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "-c", "banana"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for invalid --cycle value")
	}
	if !strings.Contains(err.Error(), "invalid --cycle value") {
		t.Errorf("error %q should contain 'invalid --cycle value'", err.Error())
	}
}

func TestIssueList_CycleNoCycleFound(t *testing.T) {
	t.Parallel()

	noCyclesResponse := `{
		"data": {
			"cycles": {
				"nodes": []
			}
		}
	}`

	server := newMockGraphQLServer(t, map[string]string{
		"ListCycles": noCyclesResponse,
	})

	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "-c", "current"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when no current cycle exists")
	}
	if !strings.Contains(err.Error(), "no current cycle found") {
		t.Errorf("error %q should contain 'no current cycle found'", err.Error())
	}
}

func TestIssueList_CycleCacheHit(t *testing.T) {
	t.Parallel()

	// First call: server has both ListCycles and issue responses.
	server := newMockGraphQLServer(t, map[string]string{
		"ListCycles":   listCyclesResponse,
		"ListMyIssues": cycleIssuesResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	opts.Cache = cache.New(t.TempDir(), 5*time.Minute)

	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "-c", "current"})
	if err := root.Execute(); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if !strings.Contains(stdout.String(), "Cycle 11") {
		t.Fatal("first call should contain Cycle 11")
	}

	// Second call: server does NOT handle ListCycles.
	// If cycles come from cache, the command still succeeds.
	server2 := newMockGraphQLServer(t, map[string]string{
		"ListMyIssues": cycleIssuesResponse,
	})
	opts2, stdout2, _ := testOptionsWithBuffers(t, server2)
	opts2.Cache = opts.Cache // reuse the same cache

	root2 := cmd.NewRootCmd(opts2)
	root2.SetArgs([]string{"issue", "list", "-c", "current"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("second call should succeed from cache: %v", err)
	}
	if !strings.Contains(stdout2.String(), "Cycle 11") {
		t.Error("second call should contain Cycle 11 from cache")
	}
}

func TestIssueList_RefreshBypassesCycleCache(t *testing.T) {
	t.Parallel()

	// First call: populate the cycle cache.
	server1 := newMockGraphQLServer(t, map[string]string{
		"ListCycles":   listCyclesResponse,
		"ListMyIssues": cycleIssuesResponse,
	})

	opts1, _, _ := testOptionsWithBuffers(t, server1)
	c := cache.New(t.TempDir(), 24*time.Hour)
	opts1.Cache = c

	root1 := cmd.NewRootCmd(opts1)
	root1.SetArgs([]string{"issue", "list", "-c", "current"})
	if err := root1.Execute(); err != nil {
		t.Fatalf("first call: %v", err)
	}

	// Second call with --refresh: server does NOT handle ListCycles.
	// Without --refresh this would succeed from cache (see TestIssueList_CycleCacheHit).
	// With --refresh the cache is cleared, forcing a ListCycles API call that
	// the server can't handle, so the command must fail.
	server2 := newMockGraphQLServer(t, map[string]string{
		"ListMyIssues": cycleIssuesResponse,
	})
	opts2, _, _ := testOptionsWithBuffers(t, server2)
	opts2.Cache = c // reuse same cache

	root2 := cmd.NewRootCmd(opts2)
	root2.SetArgs([]string{"issue", "list", "--refresh", "-c", "current"})
	err := root2.Execute()
	if err == nil {
		t.Fatal("expected error: --refresh should have cleared cache, forcing an API call")
	}
}

// --- --status flag tests ---

func TestIssueList_StatusFilter(t *testing.T) {
	t.Parallel()

	// Use sortableIssuesResponse which has started, unstarted, and backlog issues.
	// With --status started, server-side filtering means the mock returns only
	// what we give it. We simulate the server returning only started issues.
	startedOnlyResponse := `{
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
						}
					],
					"pageInfo": {"hasNextPage": false, "endCursor": null}
				}
			}
		}
	}`

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyIssues": startedOnlyResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "--status", "started"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue list --status started returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "AIS-273") {
		t.Error("expected output to contain AIS-273 (started)")
	}
	if !strings.Contains(output, "AIS-265") {
		t.Error("expected output to contain AIS-265 (started)")
	}
}

func TestIssueList_StatusFilterMultiple(t *testing.T) {
	t.Parallel()

	startedAndBacklogResponse := `{
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

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyIssues": startedAndBacklogResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "-S", "started,backlog"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue list -S started,backlog returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "AIS-273") {
		t.Error("expected output to contain AIS-273 (started)")
	}
	if !strings.Contains(output, "AIS-265") {
		t.Error("expected output to contain AIS-265 (started)")
	}
	if !strings.Contains(output, "AIS-147") {
		t.Error("expected output to contain AIS-147 (backlog)")
	}
}

func TestIssueList_StatusFilterNoMatch(t *testing.T) {
	t.Parallel()

	// Server returns empty when filtering for completed (no completed issues exist).
	server := newMockGraphQLServer(t, map[string]string{
		"ListMyIssues": emptyListResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "--status", "completed"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("issue list --status completed returned error: %v", err)
	}

	output := stdout.String()
	// No completed issues, should have header only
	if !strings.Contains(output, "IDENTIFIER") {
		t.Error("expected header in output")
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Errorf("expected only header line, got %d lines", len(lines))
	}
}

func TestIssueList_InteractiveFlag_FzfNotAvailable(t *testing.T) {
	getIssueResponse := `{
		"data": {
			"issue": {
				"id": "id-1",
				"identifier": "ENG-101",
				"title": "Fix login bug",
				"state": {"name": "In Progress", "type": "started"},
				"priority": 1,
				"url": "https://linear.app/team/ENG-101",
				"branchName": "eng-101-fix-login-bug"
			}
		}
	}`

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyIssues": listMyIssuesResponse,
		"GetIssue":     getIssueResponse,
	})

	opts, _, _ := testOptionsWithBuffers(t, server)
	opts.Cache = cache.New(t.TempDir(), 5*time.Minute)

	root := cmd.NewRootCmd(opts)
	// Use PATH override to ensure fzf is not found.
	t.Setenv("PATH", t.TempDir())
	root.SetArgs([]string{"issue", "list", "--interactive"})

	err := root.Execute()
	// Should fail because fzf is not available.
	if err == nil {
		t.Fatal("expected error when fzf is not available")
	}
	if !strings.Contains(err.Error(), "fzf") {
		t.Errorf("error %q should mention fzf", err.Error())
	}
}

func TestIssueList_InteractiveFlag_EmptyResult(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListMyIssues": emptyListResponse,
	})

	opts, _, _ := testOptionsWithBuffers(t, server)
	opts.Cache = cache.New(t.TempDir(), 5*time.Minute)

	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "list", "-i"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for interactive mode with empty list")
	}
	if !strings.Contains(err.Error(), "no issues to browse") {
		t.Errorf("error %q should contain 'no issues to browse'", err.Error())
	}
}
