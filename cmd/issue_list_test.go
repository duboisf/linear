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
