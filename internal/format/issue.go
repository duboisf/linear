package format

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/duboisf/linear/internal/api"
)

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

// FormatIssueList formats a slice of issues as an aligned table for terminal output.
func FormatIssueList(issues []*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue, color bool) string {
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', tabwriter.StripEscape)

	// Header
	header := fmt.Sprintf("%s\t%s\t%s\t%s",
		ColorizeTab(color, Bold, "IDENTIFIER"),
		ColorizeTab(color, Bold, "STATUS"),
		ColorizeTab(color, Bold, "PRIORITY"),
		ColorizeTab(color, Bold, "TITLE"),
	)
	fmt.Fprintln(w, header)

	for _, issue := range issues {
		stateName := ""
		stateType := ""
		if issue.State != nil {
			stateName = issue.State.Name
			stateType = issue.State.Type
		}

		stateStr := ColorizeTab(color, StateColor(stateType), stateName)

		priorityStr := ColorizeTab(color, PriorityColor(issue.Priority), PriorityLabel(issue.Priority))

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			issue.Identifier,
			stateStr,
			priorityStr,
			issue.Title,
		)
	}

	w.Flush()
	return buf.String()
}

// FormatIssueDetail formats a single issue in a detailed key-value format.
func FormatIssueDetail(issue *api.GetIssueIssue, color bool) string {
	var buf strings.Builder

	field := func(label, value string) {
		fmt.Fprintf(&buf, "%s %s\n", Colorize(color, Bold, label+":"), value)
	}

	field("Identifier", issue.Identifier)
	field("Title", issue.Title)

	// State
	if issue.State != nil {
		stateName := issue.State.Name
		if sc := StateColor(issue.State.Type); sc != "" {
			stateName = Colorize(color, sc, stateName)
		}
		field("State", stateName)
	} else {
		field("State", "")
	}

	// Priority
	priorityStr := PriorityLabel(issue.Priority)
	if pc := PriorityColor(issue.Priority); pc != "" {
		priorityStr = Colorize(color, pc, priorityStr)
	}
	field("Priority", priorityStr)

	// Assignee
	if issue.Assignee != nil {
		field("Assignee", issue.Assignee.Name)
	} else {
		field("Assignee", "Unassigned")
	}

	// Team
	if issue.Team != nil {
		field("Team", issue.Team.Name)
	} else {
		field("Team", "")
	}

	// Project
	if issue.Project != nil {
		field("Project", issue.Project.Name)
	} else {
		field("Project", "")
	}

	// Labels
	labelStr := ""
	if issue.Labels != nil && len(issue.Labels.Nodes) > 0 {
		names := make([]string, len(issue.Labels.Nodes))
		for i, l := range issue.Labels.Nodes {
			names[i] = l.Name
		}
		labelStr = strings.Join(names, ", ")
	}
	field("Labels", labelStr)

	// Due date
	dueDate := ""
	if issue.DueDate != nil {
		dueDate = *issue.DueDate
	}
	field("Due Date", dueDate)

	// Estimate
	estimate := ""
	if issue.Estimate != nil {
		estimate = fmt.Sprintf("%.0f", *issue.Estimate)
	}
	field("Estimate", estimate)

	// Branch name
	field("Branch Name", issue.BranchName)

	// URL
	field("URL", issue.Url)

	// Parent
	if issue.Parent != nil {
		field("Parent", fmt.Sprintf("%s %s", issue.Parent.Identifier, issue.Parent.Title))
	}

	// Description at the end, separated by a blank line
	if issue.Description != nil && *issue.Description != "" {
		fmt.Fprintln(&buf)
		fmt.Fprintln(&buf, *issue.Description)
	}

	return buf.String()
}
