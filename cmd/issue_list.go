package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/cache"
	"github.com/duboisf/linear/internal/format"
)

// stateTypeOrder maps Linear workflow state types to a sort rank.
// Lower rank = shown first.
var stateTypeOrder = map[string]int{
	"started":   1,
	"unstarted": 2,
	"triage":    3,
	"backlog":   4,
	"completed": 5,
	"canceled":  6,
}

func issueStateRank(issue *issueNode) int {
	if issue.State == nil {
		return 99
	}
	if r, ok := stateTypeOrder[issue.State.Type]; ok {
		return r
	}
	return 99
}

// issuePriorityRank returns a sort rank for priority (lower = more important).
// Priority 0 (None) sorts last.
func issuePriorityRank(p float64) float64 {
	if p == 0 {
		return 99
	}
	return p
}

// newIssueListCmd creates the "issue list" subcommand that lists issues
// assigned to the authenticated user.
func newIssueListCmd(opts Options) *cobra.Command {
	var (
		cycle        string
		interactive  bool
		limit        int
		sortBy       string
		statusFilter string
		user         string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List issues assigned to you",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return fmt.Errorf("--limit must be greater than 0, got %d", limit)
			}

			client, err := resolveClient(cmd, opts)
			if err != nil {
				return err
			}

			filter, ci, err := buildIssueFilter(statusFilter, user, cycle, cmd.Context(), client, opts.Cache)
			if err != nil {
				return err
			}
			if ci != nil {
				fmt.Fprintln(opts.Stdout, ci.formatHeader(format.ColorEnabled(cmd.OutOrStdout())))
			}

			var nodes []*issueNode

			if user != "" {
				resp, err := api.ListIssues(cmd.Context(), client, limit, nil, filter)
				if err != nil {
					return fmt.Errorf("listing issues: %w", err)
				}
				if resp.Issues == nil {
					return fmt.Errorf("no issues data returned from API")
				}
				for _, n := range resp.Issues.Nodes {
					nodes = append(nodes, convertListIssuesNode(n))
				}
			} else {
				resp, err := api.ListMyIssues(cmd.Context(), client, limit, nil, filter)
				if err != nil {
					return fmt.Errorf("listing issues: %w", err)
				}
				if resp.Viewer == nil {
					return fmt.Errorf("no viewer data returned from API")
				}
				if resp.Viewer.AssignedIssues == nil {
					return fmt.Errorf("no assigned issues data returned from API")
				}
				nodes = resp.Viewer.AssignedIssues.Nodes
			}

			sortIssues(nodes, sortBy)

			if interactive {
				issues := issuesToCompletions(nodes)
				selected, err := fzfBrowseIssues(cmd.Context(), client, issues, opts.Cache)
				if err != nil {
					return err
				}
				if selected != "" {
					fmt.Fprintln(opts.Stdout, selected)
				}
				return nil
			}

			out := format.FormatIssueList(nodes, format.ColorEnabled(cmd.OutOrStdout()))
			fmt.Fprint(opts.Stdout, out)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Browse issues interactively with fzf preview")
	_ = cmd.RegisterFlagCompletionFunc("interactive", cobra.NoFileCompletions)
	cmd.Flags().StringVarP(&cycle, "cycle", "c", "", "Filter by cycle: current, next, previous, or a cycle number")
	_ = cmd.RegisterFlagCompletionFunc("cycle", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeCycleValues(cmd, opts)
	})
	cmd.Flags().IntVarP(&limit, "limit", "n", 50, "Maximum number of issues to return")
	cmd.Flags().StringVarP(&sortBy, "sort", "s", "status", "Sort by column: status, priority, identifier, title")
	_ = cmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"status", "priority", "identifier", "title"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().StringVarP(&statusFilter, "status", "S", "", "Filter by status type: all, or comma-separated list of started, unstarted, triage, backlog, completed, canceled")
	_ = cmd.RegisterFlagCompletionFunc("status", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"all", "started", "unstarted", "triage", "backlog", "completed", "canceled"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().StringVarP(&user, "user", "u", "", "User whose issues to list")
	_ = cmd.RegisterFlagCompletionFunc("user", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeUserNames(cmd, opts)
	})

	return cmd
}

// buildIssueFilter constructs an IssueFilter from the flag values.
// When cycle is set, the resolved cycleInfo is returned for header rendering.
func buildIssueFilter(statusFilter, user, cycle string, ctx context.Context, client graphql.Client, c *cache.Cache) (*api.IssueFilter, *cycleInfo, error) {
	var filter *api.IssueFilter

	// State filter based on --status flag.
	statusLower := strings.ToLower(strings.TrimSpace(statusFilter))
	switch {
	case statusLower == "all":
		// No state filter — include everything.
		filter = nil
	case statusLower != "":
		// Explicit status list → use "in" filter.
		var types []string
		for s := range strings.SplitSeq(statusLower, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				types = append(types, s)
			}
		}
		filter = &api.IssueFilter{
			State: &api.WorkflowStateFilter{
				Type: &api.StringComparator{In: types},
			},
		}
	default:
		// Default: exclude completed and canceled.
		filter = &api.IssueFilter{
			State: &api.WorkflowStateFilter{
				Type: &api.StringComparator{Nin: []string{"completed", "canceled"}},
			},
		}
	}

	// Assignee filter for --user flag (only used with ListIssues query).
	if user != "" {
		if filter == nil {
			filter = &api.IssueFilter{}
		}
		filter.Assignee = &api.NullableUserFilter{
			DisplayName: &api.StringComparator{EqIgnoreCase: &user},
		}
	}

	// Cycle filter for --cycle flag.
	var resolvedCycle *cycleInfo
	if cycle != "" {
		if filter == nil {
			filter = &api.IssueFilter{}
		}
		ci, err := resolveCycle(ctx, client, c, cycle)
		if err != nil {
			return nil, nil, err
		}
		resolvedCycle = &ci
		filter.Cycle = &api.NullableCycleFilter{
			Number: &api.NumberComparator{Eq: &ci.Number},
		}
	}

	return filter, resolvedCycle, nil
}

// convertListIssuesNode converts a ListIssues node to the canonical issueNode type.
func convertListIssuesNode(n *api.ListIssuesIssuesIssueConnectionNodesIssue) *issueNode {
	var labels *api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnection
	if n.Labels != nil {
		convertedNodes := make([]*api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnectionNodesIssueLabel, len(n.Labels.Nodes))
		for i, l := range n.Labels.Nodes {
			convertedNodes[i] = &api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnectionNodesIssueLabel{
				Name: l.Name,
			}
		}
		labels = &api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnection{
			Nodes: convertedNodes,
		}
	}
	return &issueNode{
		Id:         n.Id,
		Identifier: n.Identifier,
		Title:      n.Title,
		State:      (*api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState)(n.State),
		Priority:   n.Priority,
		UpdatedAt:  n.UpdatedAt,
		Labels:     labels,
	}
}

type issueNode = api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue

// cycleInfo holds resolved cycle metadata for display.
type cycleInfo struct {
	Number   float64
	Name     string
	StartsAt string
	EndsAt   string
}

// formatHeader returns a colorized display string like "Cycle 11 - Sprint 11 (Jan 15 – Jan 28)".
func (c cycleInfo) formatHeader(color bool) string {
	header := format.Colorize(color, format.Bold+format.Cyan, fmt.Sprintf("Cycle %.0f", c.Number))
	if c.Name != "" {
		header += " - " + c.Name
	}
	start := formatCycleDate(c.StartsAt)
	end := formatCycleDate(c.EndsAt)
	if start != "" && end != "" {
		header += " " + format.Colorize(color, format.Gray, fmt.Sprintf("(%s – %s)", start, end))
	}
	return header
}

// formatCycleDate extracts "Jan _2" from an ISO timestamp.
// The _2 format pads single-digit days with a leading space for alignment.
func formatCycleDate(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ""
	}
	return t.Format("Jan _2")
}

const (
	_cycleCacheKey = "cycles/list"
	_cycleCacheTTL = 24 * time.Hour
)

// listCyclesCached returns cycle data, serving from cache when available.
// The cycle list is cached for 24 hours since cycles rarely change.
func listCyclesCached(ctx context.Context, client graphql.Client, c *cache.Cache) (*api.ListCyclesResponse, error) {
	if c != nil {
		if data, ok := c.GetWithTTL(_cycleCacheKey, _cycleCacheTTL); ok {
			var resp api.ListCyclesResponse
			if err := json.Unmarshal([]byte(data), &resp); err == nil {
				return &resp, nil
			}
		}
	}

	resp, err := api.ListCycles(ctx, client, 50)
	if err != nil {
		return nil, err
	}

	if c != nil {
		if data, err := json.Marshal(resp); err == nil {
			_ = c.Set(_cycleCacheKey, string(data))
		}
	}

	return resp, nil
}

// resolveCycle converts a cycle flag value to cycle info.
// Named values (current, next, previous) are resolved via the ListCycles API;
// numeric strings also query ListCycles to get the full metadata.
func resolveCycle(ctx context.Context, client graphql.Client, c *cache.Cache, value string) (cycleInfo, error) {
	isNumeric := false
	var numericVal float64
	if n, err := strconv.ParseFloat(value, 64); err == nil {
		isNumeric = true
		numericVal = n
	} else {
		switch strings.ToLower(value) {
		case "current", "next", "previous":
		default:
			return cycleInfo{}, fmt.Errorf("invalid --cycle value %q: must be current, next, previous, or a number", value)
		}
	}

	resp, err := listCyclesCached(ctx, client, c)
	if err != nil {
		return cycleInfo{}, fmt.Errorf("fetching cycles: %w", err)
	}
	if resp.Cycles == nil {
		return cycleInfo{}, fmt.Errorf("no cycles data returned from API")
	}

	for _, c := range resp.Cycles.Nodes {
		if isNumeric {
			if c.Number == numericVal {
				name := ""
				if c.Name != nil {
					name = *c.Name
				}
				return cycleInfo{Number: c.Number, Name: name, StartsAt: c.StartsAt, EndsAt: c.EndsAt}, nil
			}
			continue
		}
		matched := false
		switch strings.ToLower(value) {
		case "current":
			matched = c.IsActive
		case "next":
			matched = c.IsNext
		case "previous":
			matched = c.IsPrevious
		}
		if matched {
			name := ""
			if c.Name != nil {
				name = *c.Name
			}
			return cycleInfo{Number: c.Number, Name: name, StartsAt: c.StartsAt, EndsAt: c.EndsAt}, nil
		}
	}

	return cycleInfo{}, fmt.Errorf("no %s cycle found", value)
}

func sortIssues(nodes []*issueNode, sortBy string) {
	slices.SortFunc(nodes, func(a, b *issueNode) int {
		switch strings.ToLower(sortBy) {
		case "priority":
			if d := issuePriorityRank(a.Priority) - issuePriorityRank(b.Priority); d != 0 {
				if d < 0 {
					return -1
				}
				return 1
			}
			return issueStateRank(a) - issueStateRank(b)
		case "identifier":
			return strings.Compare(a.Identifier, b.Identifier)
		case "title":
			return strings.Compare(
				strings.ToLower(a.Title),
				strings.ToLower(b.Title),
			)
		default: // "status"
			if d := issueStateRank(a) - issueStateRank(b); d != 0 {
				return d
			}
			// Within same status, sort by priority (most important first)
			da := issuePriorityRank(a.Priority)
			db := issuePriorityRank(b.Priority)
			if da < db {
				return -1
			}
			if da > db {
				return 1
			}
			return 0
		}
	})
}
