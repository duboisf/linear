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

	defaultCols := []string{"id", "status", "priority", "title"}
	defaultColsWithLabels := []string{"id", "status", "priority", "labels", "title"}

	tests := []struct {
		name    string
		issues  []*api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue
		color   bool
		columns []string
		checks  func(t *testing.T, got string)
	}{
		{
			name:    "empty list shows header only",
			issues:  nil,
			color:   false,
			columns: defaultCols,
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
			name:    "single issue without color",
			columns: defaultCols,
			issues: []*api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue{
				{
					Identifier: "ENG-123",
					Title:      "Fix the bug",
					Priority:   2,
					State: &api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState{
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
			name:    "multiple issues",
			columns: defaultCols,
			issues: []*api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue{
				{
					Identifier: "ENG-1",
					Title:      "First issue",
					Priority:   1,
					State: &api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState{
						Name: "Done",
						Type: "completed",
					},
				},
				{
					Identifier: "ENG-2",
					Title:      "Second issue",
					Priority:   4,
					State: &api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState{
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
			name:    "nil state handling",
			columns: defaultCols,
			issues: []*api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue{
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
			name:    "labels column shown when issues have labels",
			columns: defaultColsWithLabels,
			issues: []*api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue{
				{
					Identifier: "ENG-1",
					Title:      "Labeled issue",
					Priority:   2,
					State: &api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState{
						Name: "In Progress",
						Type: "started",
					},
					Labels: &api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnection{
						Nodes: []*api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnectionNodesIssueLabel{
							{Name: "bug"},
							{Name: "frontend"},
						},
					},
				},
				{
					Identifier: "ENG-2",
					Title:      "No labels",
					Priority:   3,
					State: &api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState{
						Name: "Todo",
						Type: "unstarted",
					},
				},
			},
			color: false,
			checks: func(t *testing.T, got string) {
				t.Helper()
				if !strings.Contains(got, "LABELS") {
					t.Error("expected LABELS header when issues have labels")
				}
				if !strings.Contains(got, "bug, frontend") {
					t.Error("expected 'bug, frontend' in output")
				}
			},
		},
		{
			name:    "labels column hidden when no issues have labels",
			columns: defaultCols,
			issues: []*api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue{
				{
					Identifier: "ENG-1",
					Title:      "No labels",
					Priority:   2,
					State: &api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState{
						Name: "In Progress",
						Type: "started",
					},
				},
			},
			color: false,
			checks: func(t *testing.T, got string) {
				t.Helper()
				if strings.Contains(got, "LABELS") {
					t.Error("expected no LABELS header when no issues have labels")
				}
			},
		},
		{
			name:    "with color enabled",
			columns: defaultCols,
			issues: []*api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue{
				{
					Identifier: "ENG-10",
					Title:      "Color test",
					Priority:   1,
					State: &api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState{
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
			got := format.FormatIssueList(tt.issues, tt.color, tt.columns)
			tt.checks(t, got)
		})
	}
}

// hasField checks that the output contains a line with the given label and value,
// separated by any amount of whitespace. Used for testing aligned key-value output.
func hasField(output, label, value string) bool {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, label) && strings.Contains(line, value) {
			return true
		}
	}
	return false
}

func TestFormatIssueDetail(t *testing.T) {
	t.Parallel()

	strPtr := func(s string) *string { return &s }
	floatPtr := func(f float64) *float64 { return &f }

	fullIssue := &api.GetIssueIssue{
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
	}

	t.Run("full issue fields present", func(t *testing.T) {
		t.Parallel()
		got := format.FormatIssueDetail(fullIssue, false)
		fields := map[string]string{
			"Identifier":  "ENG-42",
			"Title":       "Implement feature X",
			"State":       "In Progress",
			"Priority":    "High",
			"Assignee":    "Jane Doe",
			"Team":        "Engineering",
			"Project":     "Project Alpha",
			"Labels":      "bug, frontend",
			"Due Date":    "2025-12-31",
			"Estimate":    "5",
			"Branch Name": "feat/implement-feature-x",
			"URL":         "https://linear.app/team/ENG-42",
			"Parent":      "ENG-1 Parent Epic",
		}
		for label, value := range fields {
			if !hasField(got, label, value) {
				t.Errorf("missing field %s: %s", label, value)
			}
		}
		if !strings.Contains(got, "This is a detailed description.") {
			t.Error("expected description in output")
		}
	})

	t.Run("values are column-aligned", func(t *testing.T) {
		t.Parallel()
		got := format.FormatIssueDetail(fullIssue, false)
		lines := strings.Split(strings.TrimSpace(got), "\n")
		// Find where values start on each metadata line (before blank line for description).
		// The format is "Label  Value" with the label padded to a fixed width.
		var valuePositions []int
		for _, line := range lines {
			if line == "" {
				break // description separator
			}
			// Find first run of 2+ spaces (gap between label and value)
			gapIdx := strings.Index(line, "  ")
			if gapIdx < 0 {
				continue
			}
			// Skip all spaces to find value start
			valStart := gapIdx
			for valStart < len(line) && line[valStart] == ' ' {
				valStart++
			}
			valuePositions = append(valuePositions, valStart)
		}
		if len(valuePositions) < 2 {
			t.Fatal("expected multiple fields to check alignment")
		}
		for i := 1; i < len(valuePositions); i++ {
			if valuePositions[i] != valuePositions[0] {
				t.Errorf("value column not aligned: position %d vs %d on line %d", valuePositions[i], valuePositions[0], i)
			}
		}
	})

	t.Run("nil optional fields", func(t *testing.T) {
		t.Parallel()
		issue := &api.GetIssueIssue{
			Id:         "issue-2",
			Identifier: "ENG-100",
			Title:      "Minimal issue",
			Url:        "https://linear.app/team/ENG-100",
			Priority:   0,
			BranchName: "fix/minimal",
		}
		got := format.FormatIssueDetail(issue, false)
		if !hasField(got, "Identifier", "ENG-100") {
			t.Error("expected Identifier")
		}
		if !hasField(got, "Priority", "None") {
			t.Error("expected Priority: None for 0")
		}
		if !hasField(got, "Assignee", "Unassigned") {
			t.Error("expected Assignee: Unassigned")
		}
		if strings.Contains(got, "Parent") {
			t.Error("expected no Parent field when parent is nil")
		}
	})

	t.Run("empty description omitted", func(t *testing.T) {
		t.Parallel()
		issue := &api.GetIssueIssue{
			Id:          "issue-3",
			Identifier:  "ENG-200",
			Title:       "Empty desc",
			Description: strPtr(""),
			Url:         "https://linear.app/team/ENG-200",
			Priority:    3,
			BranchName:  "fix/empty",
			Team:        &api.GetIssueIssueTeam{Name: "Eng", Key: "ENG"},
		}
		got := format.FormatIssueDetail(issue, false)
		if !hasField(got, "Priority", "Normal") {
			t.Error("expected Priority: Normal for 3")
		}
		// Last non-empty line should be URL
		lines := strings.Split(got, "\n")
		lastNonEmpty := ""
		for i := len(lines) - 1; i >= 0; i-- {
			if strings.TrimSpace(lines[i]) != "" {
				lastNonEmpty = lines[i]
				break
			}
		}
		if !strings.HasPrefix(strings.TrimSpace(lastNonEmpty), "URL") {
			t.Errorf("expected last content line to be URL, got %q", lastNonEmpty)
		}
	})

	t.Run("color enabled", func(t *testing.T) {
		t.Parallel()
		issue := &api.GetIssueIssue{
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
		}
		got := format.FormatIssueDetail(issue, true)
		if !strings.Contains(got, format.Red) {
			t.Error("expected red ANSI code for urgent priority")
		}
		if !strings.Contains(got, format.Green) {
			t.Error("expected green ANSI code for completed state")
		}
		if !strings.Contains(got, format.Bold) {
			t.Error("expected bold ANSI codes in field labels")
		}
	})

	t.Run("empty labels", func(t *testing.T) {
		t.Parallel()
		issue := &api.GetIssueIssue{
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
		}
		got := format.FormatIssueDetail(issue, false)
		if !hasField(got, "Priority", "Low") {
			t.Error("expected Priority: Low for 4")
		}
		if !strings.Contains(got, "Labels") {
			t.Error("expected Labels field")
		}
	})
}

func TestFormatIssueDetailMarkdown(t *testing.T) {
	t.Parallel()

	strPtr := func(s string) *string { return &s }
	floatPtr := func(f float64) *float64 { return &f }

	issue := &api.GetIssueIssue{
		Id:          "issue-1",
		Identifier:  "ENG-42",
		Title:       "Implement feature X",
		Description: strPtr("Some **markdown** description."),
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
		Team:    &api.GetIssueIssueTeam{Name: "Engineering", Key: "ENG"},
		Project: &api.GetIssueIssueProject{Name: "Project Alpha"},
		Labels: &api.GetIssueIssueLabelsIssueLabelConnection{
			Nodes: []*api.GetIssueIssueLabelsIssueLabelConnectionNodesIssueLabel{
				{Name: "bug"},
			},
		},
	}

	got := format.FormatIssueDetailMarkdown(issue)

	// Should start with a heading
	if !strings.HasPrefix(got, "# ENG-42\n") {
		t.Errorf("expected heading '# ENG-42', got first line %q", strings.SplitN(got, "\n", 2)[0])
	}

	// Should have markdown table header
	if !strings.Contains(got, "| Field") || !strings.Contains(got, "Value") {
		t.Error("expected markdown table header")
	}

	lines := strings.Split(got, "\n")

	// Find the table header line (first line starting with "|")
	tableStart := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "|") {
			tableStart = i
			break
		}
	}
	if tableStart < 0 {
		t.Fatal("no table found in output")
	}

	// Separator line should be all dashes between pipes
	if !strings.Contains(lines[tableStart+1], "|--") {
		t.Errorf("expected separator line, got %q", lines[tableStart+1])
	}

	// Check that columns are aligned: all pipes at same positions
	var pipePositions []int
	for i, ch := range lines[tableStart] {
		if ch == '|' {
			pipePositions = append(pipePositions, i)
		}
	}
	for lineNum, line := range lines[tableStart+2:] {
		if line == "" {
			break // description separator
		}
		var pos []int
		for i, ch := range line {
			if ch == '|' {
				pos = append(pos, i)
			}
		}
		if len(pos) != len(pipePositions) {
			t.Errorf("line %d has %d pipes, header has %d", lineNum+tableStart+2, len(pos), len(pipePositions))
			continue
		}
		for j := range pos {
			if pos[j] != pipePositions[j] {
				t.Errorf("line %d pipe %d at col %d, want %d", lineNum+tableStart+2, j, pos[j], pipePositions[j])
			}
		}
	}

	// Check rows contain expected values
	checks := map[string]string{
		"Identifier": "ENG-42",
		"Title":      "Implement feature X",
		"State":      "In Progress",
		"Priority":   "High",
		"Assignee":   "Jane Doe",
		"Team":       "Engineering",
		"Labels":     "bug",
	}
	for label, value := range checks {
		found := false
		for _, line := range lines {
			if strings.Contains(line, label) && strings.Contains(line, value) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected row with %s = %s", label, value)
		}
	}

	// Description should appear after the table
	tableEnd := strings.LastIndex(got, "|")
	descIdx := strings.Index(got, "Some **markdown** description.")
	if descIdx < 0 || descIdx < tableEnd {
		t.Error("expected description after table")
	}
}

func TestFormatIssueDetailJSON(t *testing.T) {
	t.Parallel()

	strPtr := func(s string) *string { return &s }

	issue := &api.GetIssueIssue{
		Id:          "issue-1",
		Identifier:  "ENG-42",
		Title:       "Test issue",
		Description: strPtr("A description"),
		Url:         "https://linear.app/ENG-42",
		Priority:    2,
		BranchName:  "feat/test",
		State: &api.GetIssueIssueStateWorkflowState{
			Name: "In Progress",
			Type: "started",
		},
		Assignee: &api.GetIssueIssueAssigneeUser{Name: "Jane", Email: "jane@example.com"},
		Team:     &api.GetIssueIssueTeam{Name: "Eng", Key: "ENG"},
		Labels: &api.GetIssueIssueLabelsIssueLabelConnection{
			Nodes: []*api.GetIssueIssueLabelsIssueLabelConnectionNodesIssueLabel{
				{Name: "bug"},
				{Name: "frontend"},
			},
		},
	}

	got, err := format.FormatIssueDetailJSON(issue)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := []string{
		`"identifier": "ENG-42"`,
		`"title": "Test issue"`,
		`"state": "In Progress"`,
		`"priority": "High"`,
		`"labels": [`,
		`"bug"`,
		`"frontend"`,
		`"description": "A description"`,
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Errorf("JSON output missing %q", check)
		}
	}
}

func TestFormatIssueDetailYAML(t *testing.T) {
	t.Parallel()

	strPtr := func(s string) *string { return &s }

	issue := &api.GetIssueIssue{
		Id:          "issue-1",
		Identifier:  "ENG-42",
		Title:       "Test issue",
		Description: strPtr("A description"),
		Url:         "https://linear.app/ENG-42",
		Priority:    2,
		BranchName:  "feat/test",
		State: &api.GetIssueIssueStateWorkflowState{
			Name: "In Progress",
			Type: "started",
		},
		Team: &api.GetIssueIssueTeam{Name: "Eng", Key: "ENG"},
		Labels: &api.GetIssueIssueLabelsIssueLabelConnection{
			Nodes: []*api.GetIssueIssueLabelsIssueLabelConnectionNodesIssueLabel{
				{Name: "bug"},
			},
		},
	}

	got := format.FormatIssueDetailYAML(issue)

	checks := []string{
		"identifier: ENG-42",
		"state: In Progress",
		"priority: High",
		"labels:\n  - bug\n",
		"description: |\n  A description\n",
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Errorf("YAML output missing %q\ngot:\n%s", check, got)
		}
	}
}
