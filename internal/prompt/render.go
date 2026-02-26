package prompt

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/format"
)

// IssueData holds flattened issue fields for Go template rendering.
type IssueData struct {
	Identifier  string
	Title       string
	Description string
	URL         string
	BranchName  string
	State       string
	Priority    string
	Assignee    string
	Team        string
	TeamKey     string
	Cycle       string
	Project     string
	Labels      []string
	DueDate     string
	Parent      string
}

// NewIssueData constructs an IssueData from a GetIssueIssue response.
func NewIssueData(issue *api.GetIssueIssue) IssueData {
	d := IssueData{
		Identifier: issue.Identifier,
		Title:      issue.Title,
		URL:        issue.Url,
		BranchName: issue.BranchName,
		Priority:   format.PriorityLabel(issue.Priority),
	}
	if issue.Description != nil {
		d.Description = *issue.Description
	}
	if issue.State != nil {
		d.State = issue.State.Name
	}
	if issue.Assignee != nil {
		d.Assignee = issue.Assignee.Name
	}
	if issue.Team != nil {
		d.Team = issue.Team.Name
		d.TeamKey = issue.Team.Key
	}
	if issue.Cycle != nil {
		if issue.Cycle.Name != nil {
			d.Cycle = *issue.Cycle.Name
		}
	}
	if issue.Project != nil {
		d.Project = issue.Project.Name
	}
	if issue.Labels != nil {
		for _, l := range issue.Labels.Nodes {
			d.Labels = append(d.Labels, l.Name)
		}
	}
	if issue.DueDate != nil {
		d.DueDate = *issue.DueDate
	}
	if issue.Parent != nil {
		d.Parent = issue.Parent.Identifier
	}
	return d
}

// IsTemplate reports whether the prompt string uses Go template syntax.
func IsTemplate(s string) bool {
	return strings.Contains(s, "{{")
}

// Render renders a prompt template with issue data.
// If the template uses Go template syntax (contains "{{"), it is parsed and
// executed as a text/template. Otherwise, the legacy {identifier} placeholder
// is replaced with the issue's identifier.
func Render(tmpl string, data IssueData) (string, error) {
	if !IsTemplate(tmpl) {
		return strings.ReplaceAll(tmpl, "{identifier}", data.Identifier), nil
	}
	t, err := template.New("prompt").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
