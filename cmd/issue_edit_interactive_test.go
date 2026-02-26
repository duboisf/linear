package cmd_test

import (
	"strings"
	"testing"

	"github.com/duboisf/linear/cmd"
)

// getIssueWithIDsResponse is a full GetIssue response including id fields
// on team, state, labels, cycle, project, and assignee.
const getIssueWithIDsResponse = `{
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
				"id": "state-started",
				"name": "In Progress",
				"type": "started"
			},
			"assignee": {
				"id": "user-1",
				"name": "Jane Doe",
				"email": "jane@example.com"
			},
			"team": {
				"id": "team-eng",
				"name": "Engineering",
				"key": "ENG"
			},
			"cycle": {
				"id": "cycle-11",
				"number": 11,
				"name": "Sprint 11",
				"startsAt": "2025-01-15T00:00:00Z",
				"endsAt": "2025-01-28T00:00:00Z"
			},
			"project": {
				"id": "proj-alpha",
				"name": "Project Alpha"
			},
			"labels": {
				"nodes": [
					{"id": "label-bug", "name": "bug"},
					{"id": "label-frontend", "name": "frontend"}
				]
			},
			"parent": {
				"identifier": "ENG-1",
				"title": "Parent Epic"
			}
		}
	}
}`

const updateIssueResponse = `{
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

const updateIssueFailedResponse = `{
	"data": {
		"issueUpdate": {
			"success": false,
			"issue": null
		}
	}
}`

const listWorkflowStatesResponse = `{
	"data": {
		"workflowStates": {
			"nodes": [
				{"id": "state-backlog", "name": "Backlog", "type": "backlog", "position": 0},
				{"id": "state-todo", "name": "Todo", "type": "unstarted", "position": 0},
				{"id": "state-started", "name": "In Progress", "type": "started", "position": 0},
				{"id": "state-review", "name": "In Review", "type": "started", "position": 1},
				{"id": "state-done", "name": "Done", "type": "completed", "position": 0},
				{"id": "state-canceled", "name": "Canceled", "type": "canceled", "position": 0}
			]
		}
	}
}`

const listProjectsResponse = `{
	"data": {
		"projects": {
			"nodes": [
				{"id": "proj-alpha", "name": "Project Alpha"},
				{"id": "proj-beta", "name": "Project Beta"}
			]
		}
	}
}`

func TestEditInteractive_MissingIdentifier(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{})
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "edit-interactive"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when no identifier provided")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("error %q should mention args requirement", err.Error())
	}
}

func TestEditInteractive_ResolveClientError(t *testing.T) {
	t.Parallel()

	opts, _, _ := testOptionsKeyringError(t)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "edit-interactive", "ENG-42"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when resolveClient fails")
	}
	if !strings.Contains(err.Error(), "not authenticated") {
		t.Errorf("error %q should contain 'not authenticated'", err.Error())
	}
}

func TestEditInteractive_IssueNotFound(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetIssue": getIssueNullResponse,
	})
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "edit-interactive", "NONEXIST-1"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for not found issue")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q should contain 'not found'", err.Error())
	}
}

func TestEditInteractive_APIError(t *testing.T) {
	t.Parallel()

	server := newErrorGraphQLServer(t)
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"issue", "edit-interactive", "ENG-42"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error from API failure")
	}
}

func TestEditInteractive_IsHidden(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{})
	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	issueCmd, _, _ := root.Find([]string{"issue"})
	for _, c := range issueCmd.Commands() {
		if c.Name() == "edit-interactive" {
			if !c.Hidden {
				t.Error("edit-interactive command should be hidden")
			}
			return
		}
	}
	t.Fatal("edit-interactive command not found")
}

func TestBuildEditableFields(t *testing.T) {
	t.Parallel()

	fields := cmd.BuildEditableFields(&cmd.TestGetIssueIssue{
		StateName:    "In Progress",
		CycleNumber:  11,
		CycleName:    "Sprint 11",
		Priority:     2,
		AssigneeName: "Jane Doe",
		ProjectName:  "Project Alpha",
		Labels:       []string{"bug", "frontend"},
		Title:        "Implement feature X",
	})

	// With labels, we should have: Status, Priority, Cycle, Labels-Add,
	// Labels-Remove, Assignee, Project, Title, Description = 9 fields.
	if len(fields) != 9 {
		t.Fatalf("expected 9 fields, got %d: %v", len(fields), fieldNames(fields))
	}

	// Verify field names are in expected order.
	expected := []string{"Status", "Priority", "Cycle", "Labels-Add", "Labels-Remove", "Assignee", "Project", "Title", "Description"}
	got := fieldNames(fields)
	for i, name := range expected {
		if got[i] != name {
			t.Errorf("field %d: want %q, got %q", i, name, got[i])
		}
	}
}

func TestBuildEditableFields_NoLabels(t *testing.T) {
	t.Parallel()

	fields := cmd.BuildEditableFields(&cmd.TestGetIssueIssue{
		StateName: "Todo",
		Priority:  3,
		Title:     "Simple issue",
	})

	// Without labels, Labels-Remove should be absent.
	got := fieldNames(fields)
	for _, name := range got {
		if name == "Labels-Remove" {
			t.Error("Labels-Remove should not appear when issue has no labels")
		}
	}
	// Should have 8 fields: Status, Priority, Cycle, Labels-Add, Assignee,
	// Project, Title, Description.
	if len(fields) != 8 {
		t.Fatalf("expected 8 fields, got %d: %v", len(fields), got)
	}
}

func fieldNames(fields []cmd.EditableField) []string {
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.Name
	}
	return names
}

func TestTruncate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly ten", 11, "exactly ten"},
		{"this is a long string", 10, "this is..."},
		{"", 5, ""},
		{"abc", 3, "abc"},
		{"abcd", 3, "..."},
		// UTF-8 safety: truncate by rune, not byte.
		{"日本語テスト文字列", 6, "日本語..."},
	}
	for _, tt := range tests {
		got := cmd.Truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestEditorCmd(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv.

	// Default when VISUAL and EDITOR are both unset.
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	if got := cmd.EditorCmd(); got != "vi" {
		t.Errorf("EditorCmd() = %q, want %q", got, "vi")
	}

	// EDITOR takes precedence over default.
	t.Setenv("EDITOR", "nano")
	if got := cmd.EditorCmd(); got != "nano" {
		t.Errorf("EditorCmd() = %q, want %q", got, "nano")
	}

	// VISUAL takes precedence over EDITOR.
	t.Setenv("VISUAL", "code --wait")
	if got := cmd.EditorCmd(); got != "code --wait" {
		t.Errorf("EditorCmd() = %q, want %q", got, "code --wait")
	}
}
