package format_test

import (
	"strings"
	"testing"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/format"
)

func TestPriorityLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		priority float64
		want     string
	}{
		{name: "none (0)", priority: 0, want: "None"},
		{name: "urgent (1)", priority: 1, want: "Urgent"},
		{name: "high (2)", priority: 2, want: "High"},
		{name: "normal (3)", priority: 3, want: "Normal"},
		{name: "low (4)", priority: 4, want: "Low"},
		{name: "unknown (5)", priority: 5, want: "Unknown(5)"},
		{name: "unknown (99)", priority: 99, want: "Unknown(99)"},
		{name: "unknown (-1)", priority: -1, want: "Unknown(-1)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := format.PriorityLabel(tt.priority)
			if got != tt.want {
				t.Errorf("PriorityLabel(%v) = %q, want %q", tt.priority, got, tt.want)
			}
		})
	}
}

func TestPriorityColor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		priority float64
		want     string
	}{
		{name: "none (0) - no color", priority: 0, want: ""},
		{name: "urgent (1) - red", priority: 1, want: format.Red},
		{name: "high (2) - yellow", priority: 2, want: format.Yellow},
		{name: "normal (3) - green", priority: 3, want: format.Green},
		{name: "low (4) - gray", priority: 4, want: format.Gray},
		{name: "unknown (5) - no color", priority: 5, want: ""},
		{name: "unknown (-1) - no color", priority: -1, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := format.PriorityColor(tt.priority)
			if got != tt.want {
				t.Errorf("PriorityColor(%v) = %q, want %q", tt.priority, got, tt.want)
			}
		})
	}
}

func TestStateColor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		stateType string
		want      string
	}{
		{name: "started - yellow", stateType: "started", want: format.Yellow},
		{name: "completed - green", stateType: "completed", want: format.Green},
		{name: "canceled - red", stateType: "canceled", want: format.Red},
		{name: "backlog - gray", stateType: "backlog", want: format.Gray},
		{name: "unknown - no color", stateType: "unknown", want: ""},
		{name: "empty - no color", stateType: "", want: ""},
		{name: "triage - no color", stateType: "triage", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := format.StateColor(tt.stateType)
			if got != tt.want {
				t.Errorf("StateColor(%q) = %q, want %q", tt.stateType, got, tt.want)
			}
		})
	}
}

func TestFormatIssueList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		issues []*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue
		color  bool
		checks func(t *testing.T, got string)
	}{
		{
			name:   "empty list shows header only",
			issues: nil,
			color:  false,
			checks: func(t *testing.T, got string) {
				t.Helper()
				if !strings.Contains(got, "IDENTIFIER") {
					t.Error("expected header to contain IDENTIFIER")
				}
				if !strings.Contains(got, "STATUS") {
					t.Error("expected header to contain STATUS")
				}
				if !strings.Contains(got, "PRIORITY") {
					t.Error("expected header to contain PRIORITY")
				}
				if !strings.Contains(got, "TITLE") {
					t.Error("expected header to contain TITLE")
				}
				// Should only contain the header line
				lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
				if len(lines) != 1 {
					t.Errorf("expected 1 line (header only), got %d", len(lines))
				}
			},
		},
		{
			name: "single issue without color",
			issues: []*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue{
				{
					Identifier: "ENG-123",
					Title:      "Fix the bug",
					Priority:   2,
					State: &api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState{
						Name: "In Progress",
						Type: "started",
					},
				},
			},
			color: false,
			checks: func(t *testing.T, got string) {
				t.Helper()
				if !strings.Contains(got, "ENG-123") {
					t.Error("expected output to contain ENG-123")
				}
				if !strings.Contains(got, "In Progress") {
					t.Error("expected output to contain In Progress")
				}
				if !strings.Contains(got, "High") {
					t.Error("expected output to contain High (priority 2)")
				}
				if !strings.Contains(got, "Fix the bug") {
					t.Error("expected output to contain Fix the bug")
				}
			},
		},
		{
			name: "multiple issues",
			issues: []*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue{
				{
					Identifier: "ENG-1",
					Title:      "First issue",
					Priority:   1,
					State: &api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState{
						Name: "Done",
						Type: "completed",
					},
				},
				{
					Identifier: "ENG-2",
					Title:      "Second issue",
					Priority:   4,
					State: &api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState{
						Name: "Backlog",
						Type: "backlog",
					},
				},
			},
			color: false,
			checks: func(t *testing.T, got string) {
				t.Helper()
				if !strings.Contains(got, "ENG-1") {
					t.Error("expected output to contain ENG-1")
				}
				if !strings.Contains(got, "ENG-2") {
					t.Error("expected output to contain ENG-2")
				}
				if !strings.Contains(got, "Urgent") {
					t.Error("expected output to contain Urgent (priority 1)")
				}
				if !strings.Contains(got, "Low") {
					t.Error("expected output to contain Low (priority 4)")
				}
				lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
				// header + 2 issue lines
				if len(lines) != 3 {
					t.Errorf("expected 3 lines, got %d", len(lines))
				}
			},
		},
		{
			name: "nil state handling",
			issues: []*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue{
				{
					Identifier: "ENG-99",
					Title:      "No state",
					Priority:   0,
					State:      nil,
				},
			},
			color: false,
			checks: func(t *testing.T, got string) {
				t.Helper()
				if !strings.Contains(got, "ENG-99") {
					t.Error("expected output to contain ENG-99")
				}
				if !strings.Contains(got, "None") {
					t.Error("expected output to contain None (priority 0)")
				}
			},
		},
		{
			name: "with color enabled",
			issues: []*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue{
				{
					Identifier: "ENG-10",
					Title:      "Color test",
					Priority:   1,
					State: &api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState{
						Name: "In Progress",
						Type: "started",
					},
				},
			},
			color: true,
			checks: func(t *testing.T, got string) {
				t.Helper()
				// Header should have Bold ANSI codes
				if !strings.Contains(got, format.Bold) {
					t.Error("expected bold ANSI codes in header")
				}
				// Started state should have Yellow
				if !strings.Contains(got, format.Yellow) {
					t.Error("expected yellow ANSI code for started state")
				}
				// Urgent priority should have Red
				if !strings.Contains(got, format.Red) {
					t.Error("expected red ANSI code for urgent priority")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := format.FormatIssueList(tt.issues, tt.color)
			tt.checks(t, got)
		})
	}
}

func TestFormatIssueDetail(t *testing.T) {
	t.Parallel()

	strPtr := func(s string) *string { return &s }
	floatPtr := func(f float64) *float64 { return &f }

	tests := []struct {
		name   string
		issue  *api.GetIssueIssue
		color  bool
		checks func(t *testing.T, got string)
	}{
		{
			name: "full issue with all fields",
			issue: &api.GetIssueIssue{
				Id:          "issue-1",
				Identifier:  "ENG-42",
				Title:       "Implement feature X",
				Description: strPtr("This is a detailed description."),
				Url:         "https://linear.app/team/ENG-42",
				Priority:    2,
				Estimate:    floatPtr(5),
				DueDate:     strPtr("2025-12-31"),
				BranchName:  "feat/implement-feature-x",
				State: &api.GetIssueIssueStateWorkflowState{
					Name: "In Progress",
					Type: "started",
				},
				Assignee: &api.GetIssueIssueAssigneeUser{
					Name:  "Jane Doe",
					Email: "jane@example.com",
				},
				Team: &api.GetIssueIssueTeam{
					Name: "Engineering",
					Key:  "ENG",
				},
				Project: &api.GetIssueIssueProject{
					Name: "Project Alpha",
				},
				Labels: &api.GetIssueIssueLabelsIssueLabelConnection{
					Nodes: []*api.GetIssueIssueLabelsIssueLabelConnectionNodesIssueLabel{
						{Name: "bug"},
						{Name: "frontend"},
					},
				},
				Parent: &api.GetIssueIssueParentIssue{
					Identifier: "ENG-1",
					Title:      "Parent Epic",
				},
			},
			color: false,
			checks: func(t *testing.T, got string) {
				t.Helper()
				expectations := []string{
					"Identifier: ENG-42",
					"Title: Implement feature X",
					"State: In Progress",
					"Priority: High",
					"Assignee: Jane Doe",
					"Team: Engineering",
					"Project: Project Alpha",
					"Labels: bug, frontend",
					"Due Date: 2025-12-31",
					"Estimate: 5",
					"Branch Name: feat/implement-feature-x",
					"URL: https://linear.app/team/ENG-42",
					"Parent: ENG-1 Parent Epic",
					"This is a detailed description.",
				}
				for _, exp := range expectations {
					if !strings.Contains(got, exp) {
						t.Errorf("expected output to contain %q", exp)
					}
				}
			},
		},
		{
			name: "issue with nil optional fields",
			issue: &api.GetIssueIssue{
				Id:          "issue-2",
				Identifier:  "ENG-100",
				Title:       "Minimal issue",
				Description: nil,
				Url:         "https://linear.app/team/ENG-100",
				Priority:    0,
				Estimate:    nil,
				DueDate:     nil,
				BranchName:  "fix/minimal",
				State:       nil,
				Assignee:    nil,
				Team:        nil,
				Project:     nil,
				Labels:      nil,
				Parent:      nil,
			},
			color: false,
			checks: func(t *testing.T, got string) {
				t.Helper()
				if !strings.Contains(got, "Identifier: ENG-100") {
					t.Error("expected Identifier")
				}
				if !strings.Contains(got, "Title: Minimal issue") {
					t.Error("expected Title")
				}
				if !strings.Contains(got, "State: ") {
					t.Error("expected empty State field")
				}
				if !strings.Contains(got, "Priority: None") {
					t.Error("expected Priority: None for 0")
				}
				if !strings.Contains(got, "Assignee: Unassigned") {
					t.Error("expected Assignee: Unassigned")
				}
				if !strings.Contains(got, "Team: ") {
					t.Error("expected empty Team field")
				}
				if !strings.Contains(got, "Project: ") {
					t.Error("expected empty Project field")
				}
				if !strings.Contains(got, "Labels: ") {
					t.Error("expected empty Labels field")
				}
				if !strings.Contains(got, "Due Date: ") {
					t.Error("expected empty Due Date field")
				}
				if !strings.Contains(got, "Estimate: ") {
					t.Error("expected empty Estimate field")
				}
				// Should NOT contain Parent line when parent is nil
				if strings.Contains(got, "Parent:") {
					t.Error("expected no Parent field when parent is nil")
				}
				// Should NOT contain description block when description is nil
				// Count blank lines - description adds a blank line before it
				// The output should not have description content
			},
		},
		{
			name: "issue with empty description",
			issue: &api.GetIssueIssue{
				Id:          "issue-3",
				Identifier:  "ENG-200",
				Title:       "Empty desc",
				Description: strPtr(""),
				Url:         "https://linear.app/team/ENG-200",
				Priority:    3,
				BranchName:  "fix/empty",
				Team:        &api.GetIssueIssueTeam{Name: "Eng", Key: "ENG"},
			},
			color: false,
			checks: func(t *testing.T, got string) {
				t.Helper()
				if !strings.Contains(got, "Priority: Normal") {
					t.Error("expected Priority: Normal for 3")
				}
				// Empty description should not add extra blank line
				lines := strings.Split(got, "\n")
				lastNonEmpty := ""
				for i := len(lines) - 1; i >= 0; i-- {
					if strings.TrimSpace(lines[i]) != "" {
						lastNonEmpty = lines[i]
						break
					}
				}
				// Last non-empty line should be URL since description is empty
				if !strings.Contains(lastNonEmpty, "URL:") {
					t.Errorf("expected last content line to be URL, got %q", lastNonEmpty)
				}
			},
		},
		{
			name: "priority colors with color enabled",
			issue: &api.GetIssueIssue{
				Id:         "issue-4",
				Identifier: "ENG-300",
				Title:      "Urgent issue",
				Url:        "https://linear.app/ENG-300",
				Priority:   1,
				BranchName: "fix/urgent",
				State: &api.GetIssueIssueStateWorkflowState{
					Name: "Done",
					Type: "completed",
				},
				Team: &api.GetIssueIssueTeam{Name: "Eng", Key: "ENG"},
			},
			color: true,
			checks: func(t *testing.T, got string) {
				t.Helper()
				// Urgent priority should use Red
				if !strings.Contains(got, format.Red) {
					t.Error("expected red ANSI code for urgent priority")
				}
				// Completed state should use Green
				if !strings.Contains(got, format.Green) {
					t.Error("expected green ANSI code for completed state")
				}
				// Bold should be in field labels
				if !strings.Contains(got, format.Bold) {
					t.Error("expected bold ANSI codes in field labels")
				}
			},
		},
		{
			name: "issue with labels but empty nodes",
			issue: &api.GetIssueIssue{
				Id:         "issue-5",
				Identifier: "ENG-400",
				Title:      "Empty labels",
				Url:        "https://linear.app/ENG-400",
				Priority:   4,
				BranchName: "fix/labels",
				Labels: &api.GetIssueIssueLabelsIssueLabelConnection{
					Nodes: []*api.GetIssueIssueLabelsIssueLabelConnectionNodesIssueLabel{},
				},
				Team: &api.GetIssueIssueTeam{Name: "Eng", Key: "ENG"},
			},
			color: false,
			checks: func(t *testing.T, got string) {
				t.Helper()
				if !strings.Contains(got, "Priority: Low") {
					t.Error("expected Priority: Low for 4")
				}
				// Labels field should be present but empty
				if !strings.Contains(got, "Labels: ") {
					t.Error("expected empty Labels field")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := format.FormatIssueDetail(tt.issue, tt.color)
			tt.checks(t, got)
		})
	}
}
