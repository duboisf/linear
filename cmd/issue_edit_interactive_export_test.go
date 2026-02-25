package cmd

import "github.com/duboisf/linear/internal/api"

// EditableField is an exported alias of editableField for testing.
type EditableField = editableField

// TestGetIssueIssue is a simplified builder for creating test api.GetIssueIssue instances.
type TestGetIssueIssue struct {
	StateName    string
	CycleNumber  float64
	CycleName    string
	Priority     float64
	AssigneeName string
	ProjectName  string
	Labels       []string
	Title        string
}

// BuildEditableFields is an exported wrapper for testing.
func BuildEditableFields(t *TestGetIssueIssue) []editableField {
	issue := &api.GetIssueIssue{
		Title:    t.Title,
		Priority: t.Priority,
	}

	if t.StateName != "" {
		issue.State = &api.GetIssueIssueStateWorkflowState{Name: t.StateName}
	}

	if t.CycleNumber != 0 {
		name := t.CycleName
		issue.Cycle = &api.GetIssueIssueCycle{Number: t.CycleNumber, Name: &name}
	}

	if t.AssigneeName != "" {
		issue.Assignee = &api.GetIssueIssueAssigneeUser{Name: t.AssigneeName}
	}

	if t.ProjectName != "" {
		issue.Project = &api.GetIssueIssueProject{Name: t.ProjectName}
	}

	if len(t.Labels) > 0 {
		nodes := make([]*api.GetIssueIssueLabelsIssueLabelConnectionNodesIssueLabel, len(t.Labels))
		for i, name := range t.Labels {
			nodes[i] = &api.GetIssueIssueLabelsIssueLabelConnectionNodesIssueLabel{Name: name}
		}
		issue.Labels = &api.GetIssueIssueLabelsIssueLabelConnection{Nodes: nodes}
	}

	return buildEditableFields(issue)
}

// Truncate is an exported wrapper for testing.
func Truncate(s string, maxLen int) string {
	return truncate(s, maxLen)
}

// EditorCmd is an exported wrapper for testing.
func EditorCmd() string {
	return editorCmd()
}
