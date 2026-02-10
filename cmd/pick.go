package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Khan/genqlient/graphql"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/format"
)

// fetchMyIssues returns the current user's active issues.
func fetchMyIssues(ctx context.Context, client graphql.Client) ([]issueForCompletion, error) {
	resp, err := api.ActiveIssuesForCompletion(ctx, client, 100)
	if err != nil {
		return nil, err
	}
	if resp.Viewer == nil || resp.Viewer.AssignedIssues == nil {
		return nil, nil
	}

	issues := make([]issueForCompletion, len(resp.Viewer.AssignedIssues.Nodes))
	for i, n := range resp.Viewer.AssignedIssues.Nodes {
		issues[i] = issueForCompletion{
			Identifier: n.Identifier,
			Title:      n.Title,
			Priority:   n.Priority,
		}
		if n.State != nil {
			issues[i].StateName = n.State.Name
			issues[i].StateType = n.State.Type
		}
	}
	return issues, nil
}

// fetchUserIssues returns a specific user's active issues.
func fetchUserIssues(ctx context.Context, client graphql.Client, userName string) ([]issueForCompletion, error) {
	resp, err := api.UserIssuesForCompletion(ctx, client, 100, userName)
	if err != nil {
		return nil, err
	}
	if resp.Issues == nil {
		return nil, nil
	}

	issues := make([]issueForCompletion, len(resp.Issues.Nodes))
	for i, n := range resp.Issues.Nodes {
		issues[i] = issueForCompletion{
			Identifier: n.Identifier,
			Title:      n.Title,
			Priority:   n.Priority,
		}
		if n.State != nil {
			issues[i].StateName = n.State.Name
			issues[i].StateType = n.State.Type
		}
	}
	return issues, nil
}

// fetchIssuesForUser fetches issues for the given user arg ("@my" or a user name).
func fetchIssuesForUser(ctx context.Context, client graphql.Client, userArg string) ([]issueForCompletion, error) {
	if userArg == "@my" {
		return fetchMyIssues(ctx, client)
	}
	return fetchUserIssues(ctx, client, userArg)
}

// formatFzfLines formats issues into aligned, ANSI-colored lines for fzf,
// including a header line. Returns the header and the data lines separately.
func formatFzfLines(issues []issueForCompletion) (header string, lines []string) {
	if len(issues) == 0 {
		return "", nil
	}

	// Include header label widths in column calculations so header and data
	// columns align. The shell completion version skips this because zsh pads
	// the VALUE column independently, but fzf doesn't.
	maxID := len("IDENTIFIER")
	maxState := len("STATUS")
	maxPri := len("PRIORITY")
	for _, issue := range issues {
		if len(issue.Identifier) > maxID {
			maxID = len(issue.Identifier)
		}
		if len(issue.StateName) > maxState {
			maxState = len(issue.StateName)
		}
		if l := len(format.PriorityLabel(issue.Priority)); l > maxPri {
			maxPri = l
		}
	}

	gap := "  "

	header = fmt.Sprintf("%-*s%s%-*s%s%-*s%s%s",
		maxID, "IDENTIFIER", gap,
		maxState, "STATUS", gap,
		maxPri, "PRIORITY", gap,
		"TITLE")

	lines = make([]string, len(issues))
	for i, issue := range issues {
		stateCol := format.PadColor(true, format.StateColor(issue.StateType), issue.StateName, maxState)
		priCol := format.PadColor(true, format.PriorityColor(issue.Priority), format.PriorityLabel(issue.Priority), maxPri)

		lines[i] = fmt.Sprintf("%-*s%s%s%s%s%s%s",
			maxID, issue.Identifier, gap,
			stateCol, gap,
			priCol, gap,
			issue.Title)
	}
	return header, lines
}

// fzfPickIssue presents issues in fzf with aligned columns and returns the
// selected identifier. Returns empty string if the user cancelled (ESC/Ctrl-C).
func fzfPickIssue(issues []issueForCompletion) (string, error) {
	if len(issues) == 0 {
		return "", fmt.Errorf("no issues to select from")
	}

	header, lines := formatFzfLines(issues)

	// Pass header as the first input line with --header-lines=1 so fzf
	// applies the same left margin as data lines (pointer-width aware).
	input := header + "\n" + strings.Join(lines, "\n") + "\n"

	cmd := exec.Command("fzf", "--ansi", "--header-lines=1", "--no-sort", "--layout=reverse")
	cmd.Stdin = strings.NewReader(input)

	var out bytes.Buffer
	cmd.Stdout = &out
	// Let fzf use /dev/tty for interactive input.
	cmd.Stderr = nil

	err := cmd.Run()
	if err != nil {
		// fzf exits 130 on ESC/Ctrl-C.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return "", nil
		}
		// fzf exits 1 when no match.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", nil
		}
		return "", fmt.Errorf("running fzf: %w", err)
	}

	selected := strings.TrimSpace(out.String())
	if selected == "" {
		return "", nil
	}

	// First field is the identifier.
	fields := strings.Fields(selected)
	if len(fields) == 0 {
		return "", nil
	}
	return fields[0], nil
}
