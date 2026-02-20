package format_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/format"
)

func TestDefaultColumns_WithLabels(t *testing.T) {
	t.Parallel()

	issues := []*api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue{
		{
			Identifier: "ENG-1",
			Labels: &api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnection{
				Nodes: []*api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnectionNodesIssueLabel{
					{Name: "bug"},
				},
			},
		},
	}

	got := format.DefaultColumns(issues)
	want := []string{"id", "status", "priority", "labels", "title"}
	if !slices.Equal(got, want) {
		t.Errorf("DefaultColumns = %v, want %v", got, want)
	}
}

func TestDefaultColumns_WithoutLabels(t *testing.T) {
	t.Parallel()

	issues := []*api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue{
		{Identifier: "ENG-1"},
	}

	got := format.DefaultColumns(issues)
	want := []string{"id", "status", "priority", "title"}
	if !slices.Equal(got, want) {
		t.Errorf("DefaultColumns = %v, want %v", got, want)
	}
}

func TestDefaultColumns_Nil(t *testing.T) {
	t.Parallel()

	got := format.DefaultColumns(nil)
	want := []string{"id", "status", "priority", "title"}
	if !slices.Equal(got, want) {
		t.Errorf("DefaultColumns(nil) = %v, want %v", got, want)
	}
}

func TestParseColumns_Replacement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec string
		want []string
	}{
		{
			name: "single column",
			spec: "id",
			want: []string{"id"},
		},
		{
			name: "multiple columns",
			spec: "id,status,title",
			want: []string{"id", "status", "title"},
		},
		{
			name: "all columns",
			spec: "id,status,priority,labels,title,updated",
			want: []string{"id", "status", "priority", "labels", "title", "updated"},
		},
		{
			name: "reordered",
			spec: "title,id,priority",
			want: []string{"title", "id", "priority"},
		},
		{
			name: "case insensitive",
			spec: "ID,Status,TITLE",
			want: []string{"id", "status", "title"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := format.ParseColumns(tt.spec)
			if err != nil {
				t.Fatalf("ParseColumns(%q) returned error: %v", tt.spec, err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("ParseColumns(%q) = %v, want %v", tt.spec, got, tt.want)
			}
		})
	}
}

func TestParseColumns_Additive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec string
		want []string
	}{
		{
			name: "append updated",
			spec: "+updated",
			want: []string{"id", "status", "priority", "labels", "title", "updated"},
		},
		{
			name: "insert at position 1",
			spec: "+updated:1",
			want: []string{"updated", "id", "status", "priority", "labels", "title"},
		},
		{
			name: "insert at position 2",
			spec: "+updated:2",
			want: []string{"id", "updated", "status", "priority", "labels", "title"},
		},
		{
			name: "insert at last position",
			spec: "+updated:5",
			want: []string{"id", "status", "priority", "labels", "updated", "title"},
		},
		{
			name: "position exceeds length appends",
			spec: "+updated:99",
			want: []string{"id", "status", "priority", "labels", "title", "updated"},
		},
		{
			name: "skip duplicate silently",
			spec: "+labels",
			want: []string{"id", "status", "priority", "labels", "title"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := format.ParseColumns(tt.spec)
			if err != nil {
				t.Fatalf("ParseColumns(%q) returned error: %v", tt.spec, err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("ParseColumns(%q) = %v, want %v", tt.spec, got, tt.want)
			}
		})
	}
}

func TestParseColumns_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		spec    string
		wantErr string
	}{
		{
			name:    "empty string",
			spec:    "",
			wantErr: "--column requires at least one column name",
		},
		{
			name:    "unknown column",
			spec:    "bogus",
			wantErr: `unknown column "bogus"`,
		},
		{
			name:    "duplicate column",
			spec:    "id,id",
			wantErr: `duplicate column "id"`,
		},
		{
			name:    "mixed additive and replacement",
			spec:    "id,+updated",
			wantErr: "cannot mix additive",
		},
		{
			name:    "invalid position",
			spec:    "+updated:abc",
			wantErr: "invalid position",
		},
		{
			name:    "position zero",
			spec:    "+updated:0",
			wantErr: "position must be >= 1",
		},
		{
			name:    "unknown additive column",
			spec:    "+bogus",
			wantErr: `unknown column "bogus"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := format.ParseColumns(tt.spec)
			if err == nil {
				t.Fatalf("ParseColumns(%q) expected error containing %q", tt.spec, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ParseColumns(%q) error = %q, want containing %q", tt.spec, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestFormatIssueList_CustomColumns(t *testing.T) {
	t.Parallel()

	issues := []*api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue{
		{
			Identifier: "ENG-1",
			Title:      "First issue",
			Priority:   2,
			UpdatedAt:  "2025-06-15T10:00:00Z",
			State: &api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState{
				Name: "In Progress",
				Type: "started",
			},
		},
	}

	t.Run("id and title only", func(t *testing.T) {
		t.Parallel()
		got := format.FormatIssueList(issues, false, []string{"id", "title"})
		if !strings.Contains(got, "IDENTIFIER") {
			t.Error("expected IDENTIFIER header")
		}
		if !strings.Contains(got, "TITLE") {
			t.Error("expected TITLE header")
		}
		if strings.Contains(got, "STATUS") {
			t.Error("should not contain STATUS header")
		}
		if strings.Contains(got, "PRIORITY") {
			t.Error("should not contain PRIORITY header")
		}
		if !strings.Contains(got, "ENG-1") {
			t.Error("expected ENG-1 in output")
		}
		if !strings.Contains(got, "First issue") {
			t.Error("expected title in output")
		}
	})

	t.Run("updated column", func(t *testing.T) {
		t.Parallel()
		got := format.FormatIssueList(issues, false, []string{"id", "updated", "title"})
		if !strings.Contains(got, "UPDATED") {
			t.Error("expected UPDATED header")
		}
		if !strings.Contains(got, "2025-06-15") {
			t.Error("expected formatted date in output")
		}
	})

	t.Run("reversed order", func(t *testing.T) {
		t.Parallel()
		got := format.FormatIssueList(issues, false, []string{"title", "status", "id"})
		lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
		if len(lines) < 1 {
			t.Fatal("expected at least header line")
		}
		header := lines[0]
		titleIdx := strings.Index(header, "TITLE")
		statusIdx := strings.Index(header, "STATUS")
		idIdx := strings.Index(header, "IDENTIFIER")
		if titleIdx >= statusIdx || statusIdx >= idIdx {
			t.Errorf("expected TITLE before STATUS before IDENTIFIER, got positions %d, %d, %d",
				titleIdx, statusIdx, idIdx)
		}
	})
}

func TestFormatIssueList_NewColumns(t *testing.T) {
	t.Parallel()

	estimate := float64(5)
	dueDate := "2025-07-01"
	cycleName := "Sprint 11"

	issues := []*api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue{
		{
			Identifier: "ENG-1",
			Title:      "First issue",
			Priority:   2,
			CreatedAt:  "2025-06-01T10:00:00Z",
			UpdatedAt:  "2025-06-15T10:00:00Z",
			DueDate:    &dueDate,
			Estimate:   &estimate,
			Assignee: &api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueAssigneeUser{
				Name: "Alice",
			},
			Cycle: &api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueCycle{
				Number: 11,
				Name:   &cycleName,
			},
			Project: &api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueProject{
				Name: "Backend Rewrite",
			},
			State: &api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState{
				Name: "In Progress",
				Type: "started",
			},
		},
	}

	t.Run("cycle column", func(t *testing.T) {
		t.Parallel()
		got := format.FormatIssueList(issues, false, []string{"id", "cycle", "title"})
		if !strings.Contains(got, "CYCLE") {
			t.Error("expected CYCLE header")
		}
		if !strings.Contains(got, "11") {
			t.Error("expected cycle number 11 in output")
		}
	})

	t.Run("assignee column", func(t *testing.T) {
		t.Parallel()
		got := format.FormatIssueList(issues, false, []string{"id", "assignee", "title"})
		if !strings.Contains(got, "ASSIGNEE") {
			t.Error("expected ASSIGNEE header")
		}
		if !strings.Contains(got, "Alice") {
			t.Error("expected assignee name 'Alice' in output")
		}
	})

	t.Run("project column", func(t *testing.T) {
		t.Parallel()
		got := format.FormatIssueList(issues, false, []string{"id", "project", "title"})
		if !strings.Contains(got, "PROJECT") {
			t.Error("expected PROJECT header")
		}
		if !strings.Contains(got, "Backend Rewrite") {
			t.Error("expected project name in output")
		}
	})

	t.Run("estimate column", func(t *testing.T) {
		t.Parallel()
		got := format.FormatIssueList(issues, false, []string{"id", "estimate", "title"})
		if !strings.Contains(got, "ESTIMATE") {
			t.Error("expected ESTIMATE header")
		}
		if !strings.Contains(got, "5") {
			t.Error("expected estimate value 5 in output")
		}
	})

	t.Run("duedate column", func(t *testing.T) {
		t.Parallel()
		got := format.FormatIssueList(issues, false, []string{"id", "duedate", "title"})
		if !strings.Contains(got, "DUE DATE") {
			t.Error("expected DUE DATE header")
		}
		if !strings.Contains(got, "2025-07-01") {
			t.Error("expected due date in output")
		}
	})

	t.Run("created column", func(t *testing.T) {
		t.Parallel()
		got := format.FormatIssueList(issues, false, []string{"id", "created", "title"})
		if !strings.Contains(got, "CREATED") {
			t.Error("expected CREATED header")
		}
		if !strings.Contains(got, "2025-06-01") {
			t.Error("expected formatted created date in output")
		}
	})
}

func TestFormatIssueList_NilOptionalColumns(t *testing.T) {
	t.Parallel()

	issues := []*api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue{
		{
			Identifier: "ENG-2",
			Title:      "No extras",
			Priority:   1,
			CreatedAt:  "2025-06-01T10:00:00Z",
			UpdatedAt:  "2025-06-15T10:00:00Z",
		},
	}

	// All optional columns should render empty without panicking.
	got := format.FormatIssueList(issues, false, []string{"id", "cycle", "assignee", "project", "estimate", "duedate", "title"})
	if !strings.Contains(got, "CYCLE") {
		t.Error("expected CYCLE header")
	}
	if !strings.Contains(got, "ENG-2") {
		t.Error("expected ENG-2 in output")
	}
}

func TestParseColumns_NewColumns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec string
		want []string
	}{
		{
			name: "cycle column",
			spec: "id,cycle,title",
			want: []string{"id", "cycle", "title"},
		},
		{
			name: "assignee column",
			spec: "id,assignee,title",
			want: []string{"id", "assignee", "title"},
		},
		{
			name: "project column",
			spec: "id,project,title",
			want: []string{"id", "project", "title"},
		},
		{
			name: "estimate column",
			spec: "id,estimate,title",
			want: []string{"id", "estimate", "title"},
		},
		{
			name: "duedate column",
			spec: "id,duedate,title",
			want: []string{"id", "duedate", "title"},
		},
		{
			name: "created column",
			spec: "id,created,title",
			want: []string{"id", "created", "title"},
		},
		{
			name: "additive cycle",
			spec: "+cycle",
			want: []string{"id", "status", "priority", "labels", "title", "cycle"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := format.ParseColumns(tt.spec)
			if err != nil {
				t.Fatalf("ParseColumns(%q) returned error: %v", tt.spec, err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("ParseColumns(%q) = %v, want %v", tt.spec, got, tt.want)
			}
		})
	}
}
