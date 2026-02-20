package format

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/duboisf/linear/internal/api"
)

// formatShortDate extracts "Jan _2" from an ISO timestamp.
func formatShortDate(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ""
	}
	return t.Format("Jan _2")
}

// issueField holds a single key-value pair for issue detail rendering.
type issueField struct {
	Label string
	Value string
	// Color holds the ANSI color code for the value (used by plaintext only).
	Color string
}

// extractIssueFields extracts display fields from a GetIssueIssue.
// The returned slice omits Parent when nil. Description is not included
// since it is rendered separately after the metadata.
func extractIssueFields(issue *api.GetIssueIssue) []issueField {
	var fields []issueField

	add := func(label, value, color string) {
		fields = append(fields, issueField{Label: label, Value: value, Color: color})
	}

	add("Identifier", issue.Identifier, "")
	add("Title", issue.Title, "")

	stateName, stateColor := "", ""
	if issue.State != nil {
		stateName = issue.State.Name
		stateColor = StateColor(issue.State.Type)
	}
	add("State", stateName, stateColor)

	add("Priority", PriorityLabel(issue.Priority), PriorityColor(issue.Priority))

	assignee := "Unassigned"
	if issue.Assignee != nil {
		assignee = issue.Assignee.Name
	}
	add("Assignee", assignee, "")

	team := ""
	if issue.Team != nil {
		team = issue.Team.Name
	}
	add("Team", team, "")

	cycle := ""
	if issue.Cycle != nil {
		cycle = fmt.Sprintf("%.0f", issue.Cycle.Number)
		if issue.Cycle.Name != nil && *issue.Cycle.Name != "" {
			cycle += " - " + *issue.Cycle.Name
		}
		start := formatShortDate(issue.Cycle.StartsAt)
		end := formatShortDate(issue.Cycle.EndsAt)
		if start != "" && end != "" {
			cycle += fmt.Sprintf(" (%s – %s)", start, end)
		}
	}
	add("Cycle", cycle, Cyan)

	project := ""
	if issue.Project != nil {
		project = issue.Project.Name
	}
	add("Project", project, "")

	labelStr := ""
	if issue.Labels != nil && len(issue.Labels.Nodes) > 0 {
		names := make([]string, len(issue.Labels.Nodes))
		for i, l := range issue.Labels.Nodes {
			names[i] = l.Name
		}
		labelStr = strings.Join(names, ", ")
	}
	add("Labels", labelStr, "")

	dueDate := ""
	if issue.DueDate != nil {
		dueDate = *issue.DueDate
	}
	add("Due Date", dueDate, "")

	estimate := ""
	if issue.Estimate != nil {
		estimate = fmt.Sprintf("%.0f", *issue.Estimate)
	}
	add("Estimate", estimate, "")

	add("Branch Name", issue.BranchName, "")
	add("URL", issue.Url, "")

	if issue.Parent != nil {
		add("Parent", fmt.Sprintf("%s %s", issue.Parent.Identifier, issue.Parent.Title), "")
	}

	return fields
}

// PriorityLabel returns a human-readable label for the given priority value.
func PriorityLabel(p float64) string {
	switch p {
	case 0:
		return "None"
	case 1:
		return "Urgent"
	case 2:
		return "High"
	case 3:
		return "Normal"
	case 4:
		return "Low"
	default:
		return fmt.Sprintf("Unknown(%.0f)", p)
	}
}

// PriorityColor returns the ANSI color code for the given priority value.
func PriorityColor(p float64) string {
	switch p {
	case 1:
		return Red
	case 2:
		return Yellow
	case 3:
		return Green
	case 4:
		return Gray
	default:
		return ""
	}
}

// StateColor returns the ANSI color code for the given workflow state type.
func StateColor(stateType string) string {
	switch stateType {
	case "started":
		return Yellow
	case "completed":
		return Green
	case "canceled":
		return Red
	case "backlog":
		return Gray
	default:
		return ""
	}
}

// issueLabels returns a comma-separated label string for an issue.
func issueLabels(issue *api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue) string {
	if issue.Labels == nil || len(issue.Labels.Nodes) == 0 {
		return ""
	}
	names := make([]string, len(issue.Labels.Nodes))
	for i, l := range issue.Labels.Nodes {
		names[i] = l.Name
	}
	return strings.Join(names, ", ")
}

// FormatIssueList formats a slice of issues as an aligned table for terminal output.
func FormatIssueList(issues []*api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue, color bool) string {
	const gap = "  "

	// Check if any issues have labels.
	hasLabels := false
	for _, issue := range issues {
		if issue.Labels != nil && len(issue.Labels.Nodes) > 0 {
			hasLabels = true
			break
		}
	}

	// Compute max visible widths per column.
	maxID := len("IDENTIFIER")
	maxState := len("STATUS")
	maxPri := len("PRIORITY")
	maxLabels := len("LABELS")
	for _, issue := range issues {
		if len(issue.Identifier) > maxID {
			maxID = len(issue.Identifier)
		}
		if issue.State != nil && len(issue.State.Name) > maxState {
			maxState = len(issue.State.Name)
		}
		if l := len(PriorityLabel(issue.Priority)); l > maxPri {
			maxPri = l
		}
		if hasLabels {
			if l := len(issueLabels(issue)); l > maxLabels {
				maxLabels = l
			}
		}
	}

	var buf strings.Builder

	// Header
	buf.WriteString(PadColor(color, Bold, "IDENTIFIER", maxID))
	buf.WriteString(gap)
	buf.WriteString(PadColor(color, Bold, "STATUS", maxState))
	buf.WriteString(gap)
	buf.WriteString(PadColor(color, Bold, "PRIORITY", maxPri))
	buf.WriteString(gap)
	if hasLabels {
		buf.WriteString(PadColor(color, Bold, "LABELS", maxLabels))
		buf.WriteString(gap)
	}
	buf.WriteString(Colorize(color, Bold, "TITLE"))
	buf.WriteByte('\n')

	// Rows
	for _, issue := range issues {
		stateName := ""
		stateType := ""
		if issue.State != nil {
			stateName = issue.State.Name
			stateType = issue.State.Type
		}

		buf.WriteString(fmt.Sprintf("%-*s", maxID, issue.Identifier))
		buf.WriteString(gap)
		buf.WriteString(PadColor(color, StateColor(stateType), stateName, maxState))
		buf.WriteString(gap)
		buf.WriteString(PadColor(color, PriorityColor(issue.Priority), PriorityLabel(issue.Priority), maxPri))
		buf.WriteString(gap)
		if hasLabels {
			buf.WriteString(PadColor(color, Cyan, issueLabels(issue), maxLabels))
			buf.WriteString(gap)
		}
		buf.WriteString(issue.Title)
		buf.WriteByte('\n')
	}

	return buf.String()
}

// FormatIssueDetail formats a single issue as aligned key-value plaintext.
func FormatIssueDetail(issue *api.GetIssueIssue, color bool) string {
	fields := extractIssueFields(issue)

	// Find the widest label for alignment.
	maxLabel := 0
	for _, f := range fields {
		if len(f.Label) > maxLabel {
			maxLabel = len(f.Label)
		}
	}

	var buf strings.Builder
	for _, f := range fields {
		label := fmt.Sprintf("%-*s", maxLabel, f.Label)
		value := f.Value
		if color && f.Color != "" {
			value = Colorize(true, f.Color, value)
		}
		fmt.Fprintf(&buf, "%s  %s\n", Colorize(color, Bold, label), value)
	}

	if issue.Description != nil && *issue.Description != "" {
		buf.WriteByte('\n')
		buf.WriteString(*issue.Description)
		buf.WriteByte('\n')
	}

	return buf.String()
}

// FormatIssueDetailMarkdown formats a single issue as a markdown table
// with the description as the body.
func FormatIssueDetailMarkdown(issue *api.GetIssueIssue) string {
	fields := extractIssueFields(issue)

	// Compute max widths for aligned columns.
	maxLabel := len("Field")
	maxValue := len("Value")
	type escapedField struct {
		Label string
		Value string
	}
	escaped := make([]escapedField, len(fields))
	for i, f := range fields {
		value := strings.ReplaceAll(f.Value, "|", "\\|")
		escaped[i] = escapedField{Label: f.Label, Value: value}
		if len(f.Label) > maxLabel {
			maxLabel = len(f.Label)
		}
		if len(value) > maxValue {
			maxValue = len(value)
		}
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, "# %s\n\n", issue.Identifier)
	fmt.Fprintf(&buf, "| %-*s | %-*s |\n", maxLabel, "Field", maxValue, "Value")
	fmt.Fprintf(&buf, "|-%s-|-%s-|\n", strings.Repeat("-", maxLabel), strings.Repeat("-", maxValue))
	for _, f := range escaped {
		fmt.Fprintf(&buf, "| %-*s | %-*s |\n", maxLabel, f.Label, maxValue, f.Value)
	}

	if issue.Description != nil && *issue.Description != "" {
		buf.WriteByte('\n')
		buf.WriteString(*issue.Description)
		buf.WriteByte('\n')
	}

	return buf.String()
}

// issueDetailJSON is the serialization struct for JSON/YAML output.
type issueDetailJSON struct {
	Identifier  string   `json:"identifier"`
	Title       string   `json:"title"`
	State       string   `json:"state"`
	Priority    string   `json:"priority"`
	Assignee    string   `json:"assignee"`
	Team        string   `json:"team"`
	Cycle       string   `json:"cycle,omitempty"`
	Project     string   `json:"project"`
	Labels      []string `json:"labels"`
	DueDate     string   `json:"due_date,omitempty"`
	Estimate    *float64 `json:"estimate,omitempty"`
	BranchName  string   `json:"branch_name"`
	URL         string   `json:"url"`
	Parent      string   `json:"parent,omitempty"`
	Description string   `json:"description,omitempty"`
}

func newIssueDetailJSON(issue *api.GetIssueIssue) issueDetailJSON {
	d := issueDetailJSON{
		Identifier: issue.Identifier,
		Title:      issue.Title,
		Priority:   PriorityLabel(issue.Priority),
		BranchName: issue.BranchName,
		URL:        issue.Url,
		Estimate:   issue.Estimate,
	}

	if issue.State != nil {
		d.State = issue.State.Name
	}
	if issue.Assignee != nil {
		d.Assignee = issue.Assignee.Name
	} else {
		d.Assignee = "Unassigned"
	}
	if issue.Team != nil {
		d.Team = issue.Team.Name
	}
	if issue.Cycle != nil {
		d.Cycle = fmt.Sprintf("%.0f", issue.Cycle.Number)
		if issue.Cycle.Name != nil && *issue.Cycle.Name != "" {
			d.Cycle += " - " + *issue.Cycle.Name
		}
	}
	if issue.Project != nil {
		d.Project = issue.Project.Name
	}
	if issue.Labels != nil {
		d.Labels = make([]string, len(issue.Labels.Nodes))
		for i, l := range issue.Labels.Nodes {
			d.Labels[i] = l.Name
		}
	}
	if issue.DueDate != nil {
		d.DueDate = *issue.DueDate
	}
	if issue.Parent != nil {
		d.Parent = fmt.Sprintf("%s %s", issue.Parent.Identifier, issue.Parent.Title)
	}
	if issue.Description != nil {
		d.Description = *issue.Description
	}

	return d
}

// FormatIssueDetailJSON formats a single issue as indented JSON.
func FormatIssueDetailJSON(issue *api.GetIssueIssue) (string, error) {
	data := newIssueDetailJSON(issue)
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling issue to JSON: %w", err)
	}
	return string(b) + "\n", nil
}

// FormatIssueDetailYAML formats a single issue as YAML.
// Hand-written to avoid a gopkg.in/yaml.v3 dependency.
func FormatIssueDetailYAML(issue *api.GetIssueIssue) string {
	d := newIssueDetailJSON(issue)

	var buf strings.Builder

	yamlStr := func(key, value string) {
		if strings.ContainsAny(value, ":#{}[]|>&*!%@`'\"") || value == "" {
			fmt.Fprintf(&buf, "%s: %q\n", key, value)
		} else {
			fmt.Fprintf(&buf, "%s: %s\n", key, value)
		}
	}

	yamlStr("identifier", d.Identifier)
	yamlStr("title", d.Title)
	yamlStr("state", d.State)
	yamlStr("priority", d.Priority)
	yamlStr("assignee", d.Assignee)
	yamlStr("team", d.Team)
	if d.Cycle != "" {
		yamlStr("cycle", d.Cycle)
	}
	yamlStr("project", d.Project)

	if len(d.Labels) == 0 {
		buf.WriteString("labels: []\n")
	} else {
		buf.WriteString("labels:\n")
		for _, l := range d.Labels {
			fmt.Fprintf(&buf, "  - %s\n", l)
		}
	}

	if d.DueDate != "" {
		yamlStr("due_date", d.DueDate)
	}
	if d.Estimate != nil {
		fmt.Fprintf(&buf, "estimate: %.0f\n", *d.Estimate)
	}

	yamlStr("branch_name", d.BranchName)
	yamlStr("url", d.URL)

	if d.Parent != "" {
		yamlStr("parent", d.Parent)
	}

	if d.Description != "" {
		buf.WriteString("description: |\n")
		for _, line := range strings.Split(d.Description, "\n") {
			fmt.Fprintf(&buf, "  %s\n", line)
		}
	}

	return buf.String()
}
