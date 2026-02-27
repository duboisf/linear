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

// ShellQuote wraps s in single quotes with embedded single quotes escaped.
// Use this to safely interpolate untrusted strings into shell commands.
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// shellSafeData wraps IssueData with all string fields pre-quoted for safe
// shell interpolation. Access unquoted values via the Raw field.
type shellSafeData struct {
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
	Labels      string
	DueDate     string
	Parent      string
	Raw         IssueData
}

func newShellSafeData(d IssueData) shellSafeData {
	labels := strings.Join(d.Labels, ",")
	return shellSafeData{
		Identifier:  ShellQuote(d.Identifier),
		Title:       ShellQuote(d.Title),
		Description: ShellQuote(d.Description),
		URL:         ShellQuote(d.URL),
		BranchName:  ShellQuote(d.BranchName),
		State:       ShellQuote(d.State),
		Priority:    ShellQuote(d.Priority),
		Assignee:    ShellQuote(d.Assignee),
		Team:        ShellQuote(d.Team),
		TeamKey:     ShellQuote(d.TeamKey),
		Cycle:       ShellQuote(d.Cycle),
		Project:     ShellQuote(d.Project),
		Labels:      ShellQuote(labels),
		DueDate:     ShellQuote(d.DueDate),
		Parent:      ShellQuote(d.Parent),
		Raw:         d,
	}
}

// templateFuncs are custom functions available in command templates.
var templateFuncs = template.FuncMap{
	"sq":  ShellQuote,
	"raw": func(s string) string { return s },
}

// IsTemplate reports whether the prompt string uses Go template syntax.
func IsTemplate(s string) bool {
	return strings.Contains(s, "{{")
}

// Render renders a prompt template with issue data.
// All template fields are shell-quoted by default to prevent injection.
// Use {{.Raw.Field}} for the unquoted value in display-only contexts.
// The "sq" and "raw" template functions are also available.
func Render(tmpl string, data IssueData) (string, error) {
	if !IsTemplate(tmpl) {
		return strings.ReplaceAll(tmpl, "{identifier}", data.Identifier), nil
	}
	t, err := template.New("prompt").Funcs(templateFuncs).Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, newShellSafeData(data)); err != nil {
		return "", err
	}
	return buf.String(), nil
}
