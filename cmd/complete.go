package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/format"
)

// completeUsers returns shell completions for user selection: @my first, then
// team member first names from the API.
func completeUsers(cmd *cobra.Command, opts Options) ([]string, cobra.ShellCompDirective) {
	client, err := resolveClient(cmd, opts)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	resp, err := api.UsersForCompletion(cmd.Context(), client, 100)
	if err != nil || resp.Users == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	comps := make([]string, 0, len(resp.Users.Nodes)+1)
	comps = append(comps, "@my\tYour own issues")
	for _, u := range resp.Users.Nodes {
		comps = append(comps, userCompletionEntry(u.DisplayName, u.Name))
	}
	return comps, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
}

// userCompletionEntry formats a user as a shell completion entry:
// "lowercase_first_name\tFull Name".
func userCompletionEntry(displayName, fullName string) string {
	parts := strings.Fields(displayName)
	if len(parts) == 0 {
		return fmt.Sprintf("%s\t%s", strings.ToLower(displayName), fullName)
	}
	return fmt.Sprintf("%s\t%s", strings.ToLower(parts[0]), fullName)
}

// issueForCompletion holds the fields needed to format an issue completion entry.
type issueForCompletion struct {
	Identifier string
	Title      string
	StateName  string
	StateType  string
	Priority   float64
}

// formatIssueCompletions builds shell completion entries for issues with aligned
// status, priority, and title columns. A column-header ActiveHelp line is
// prepended so zsh displays column titles above the list.
//
// In zsh, completions render as "VALUE  -- DESCRIPTION". The header is padded
// to account for the identifier width + separator so columns align visually.
func formatIssueCompletions(issues []issueForCompletion) []string {
	if len(issues) == 0 {
		return nil
	}

	// Compute max widths for alignment.
	// maxID is computed from actual identifiers only (not the header label)
	// because zsh pads the completion VALUE independently based on the longest
	// actual value, not our header text.
	var maxID int
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

	// ActiveHelp header (non-selectable line shown in zsh/bash).
	// Zsh renders completions as "VALUE  -- DESCRIPTION" where VALUE is padded
	// to the longest value width then followed by "  -- " (5 chars). The header
	// must match that offset so description columns align with data rows.
	descStart := maxID + 5
	header := fmt.Sprintf("%-*s%-*s%s%-*s%s%s",
		descStart, "IDENTIFIER",
		maxState, "STATUS", gap,
		maxPri, "PRIORITY", gap,
		"TITLE")
	comps := cobra.AppendActiveHelp(nil, header)

	for _, issue := range issues {
		stateName := issue.StateName
		stateColor := format.StateColor(issue.StateType)
		priLabel := format.PriorityLabel(issue.Priority)
		priColor := format.PriorityColor(issue.Priority)

		desc := fmt.Sprintf("%s%s%s%s%s",
			format.PadColor(true, stateColor, stateName, maxState), gap,
			format.PadColor(true, priColor, priLabel, maxPri), gap,
			issue.Title)

		comps = append(comps, fmt.Sprintf("%s\t%s", issue.Identifier, desc))
	}

	return comps
}

// completeUserNames returns shell completions for the --user flag: team member
// first names from the API (without the @my entry).
func completeUserNames(cmd *cobra.Command, opts Options) ([]string, cobra.ShellCompDirective) {
	client, err := resolveClient(cmd, opts)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	resp, err := api.UsersForCompletion(cmd.Context(), client, 100)
	if err != nil || resp.Users == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	comps := make([]string, 0, len(resp.Users.Nodes))
	for _, u := range resp.Users.Nodes {
		comps = append(comps, userCompletionEntry(u.DisplayName, u.Name))
	}
	return comps, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
}

// completeMyIssues fetches the current user's active issues and returns
// formatted completion entries with status, priority, and title.
func completeMyIssues(cmd *cobra.Command, opts Options) ([]string, cobra.ShellCompDirective) {
	client, err := resolveClient(cmd, opts)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	issues, err := fetchMyIssues(cmd.Context(), client)
	if err != nil || len(issues) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return formatIssueCompletions(issues), cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
}

// completeUserIssues fetches a specific user's active issues and returns
// formatted completion entries with status, priority, and title.
func completeUserIssues(cmd *cobra.Command, opts Options, userName string) ([]string, cobra.ShellCompDirective) {
	client, err := resolveClient(cmd, opts)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	issues, err := fetchUserIssues(cmd.Context(), client, userName)
	if err != nil || len(issues) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return formatIssueCompletions(issues), cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
}
