package cmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/format"
)

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

// completeCycleValues returns shell completions for the --cycle flag.
// It fetches cycles from the API to show cycle numbers, dates, and names.
// Upcoming future cycles are included as numbered entries so users can pick
// them directly.
func completeCycleValues(cmd *cobra.Command, opts Options) ([]string, cobra.ShellCompDirective) {
	dir := cobra.ShellCompDirectiveNoFileComp

	client, err := resolveClient(cmd, opts)
	if err != nil {
		return staticCycleCompletions(), dir
	}
	resp, err := api.ListCycles(cmd.Context(), client, 50)
	if err != nil || resp.Cycles == nil {
		return staticCycleCompletions(), dir
	}

	currentDesc := "Current active cycle"
	nextDesc := "Next upcoming cycle"
	previousDesc := "Previous completed cycle"

	// Collect future cycles (not active, not next) for extra entries.
	var futureCycles []api.ListCyclesCyclesCycleConnectionNodesCycle

	// statusWidth is the visible width to pad all status labels to (len("Upcoming") == len("Previous")).
	const statusWidth = 8

	for _, c := range resp.Cycles.Nodes {
		dates := formatCycleDateRange(c.StartsAt, c.EndsAt)
		label := fmt.Sprintf("#%.0f", c.Number)
		if c.Name != nil && *c.Name != "" {
			label += " " + *c.Name
		}
		if dates != "" {
			label += "  " + format.Colorize(true, format.Gray, dates)
		}
		if c.IsActive {
			currentDesc = format.PadColor(true, format.Green, "Active", statusWidth) + " " + label
		}
		if c.IsNext {
			nextDesc = format.PadColor(true, format.Yellow, "Next", statusWidth) + " " + label
		}
		if c.IsPrevious {
			previousDesc = format.PadColor(true, format.Gray, "Previous", statusWidth) + " " + label
		}
		if c.IsFuture && !c.IsNext {
			futureCycles = append(futureCycles, *c)
		}
	}

	comps := cobra.AppendActiveHelp(nil, "Or use any cycle number directly")
	comps = append(comps,
		"current\t"+currentDesc,
		"next\t"+nextDesc,
		"previous\t"+previousDesc,
	)

	slices.SortFunc(futureCycles, func(a, b api.ListCyclesCyclesCycleConnectionNodesCycle) int {
		if a.Number < b.Number {
			return -1
		}
		if a.Number > b.Number {
			return 1
		}
		return 0
	})

	for _, c := range futureCycles {
		num := fmt.Sprintf("%.0f", c.Number)
		desc := format.PadColor(true, format.Cyan, "Upcoming", statusWidth) + fmt.Sprintf(" #%s", num)
		if c.Name != nil && *c.Name != "" {
			desc += " " + *c.Name
		}
		dates := formatCycleDateRange(c.StartsAt, c.EndsAt)
		if dates != "" {
			desc += "  " + format.Colorize(true, format.Gray, dates)
		}
		comps = append(comps, num+"\t"+desc)
	}

	return comps, dir | cobra.ShellCompDirectiveKeepOrder
}

// formatCycleDateRange returns "Jan 2 – Jan 15" from two ISO timestamps.
func formatCycleDateRange(startsAt, endsAt string) string {
	start := formatCycleDate(startsAt)
	end := formatCycleDate(endsAt)
	if start != "" && end != "" {
		return start + " – " + end
	}
	return ""
}

func staticCycleCompletions() []string {
	return []string{
		"current\tCurrent active cycle",
		"next\tNext upcoming cycle",
		"previous\tPrevious completed cycle",
	}
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

	sortCompletionIssues(issues)
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

	sortCompletionIssues(issues)
	return formatIssueCompletions(issues), cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
}
