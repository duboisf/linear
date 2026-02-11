package cmd

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/api"
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

func issueStateRank(issue *api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue) int {
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
		all         bool
		cycle       string
		interactive bool
		limit       int
		refresh     bool
		sortBy      string
		user        string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List issues assigned to you",
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return fmt.Errorf("--limit must be greater than 0, got %d", limit)
			}

			client, err := resolveClient(cmd, opts)
			if err != nil {
				return err
			}

			var nodes []*issueNode

			if cycle != "" {
				ci, err := resolveCycle(cmd.Context(), client, cycle)
				if err != nil {
					return err
				}
				fmt.Fprintln(opts.Stdout, ci.formatHeader(format.ColorEnabled(cmd.OutOrStdout())))
				nodes, err = listIssuesByCycle(cmd.Context(), client, limit, user, all, ci.Number)
				if err != nil {
					return err
				}
			} else if user != "" {
				if all {
					resp, err := api.ListAllUserIssues(cmd.Context(), client, limit, nil, user)
					if err != nil {
						return fmt.Errorf("listing issues: %w", err)
					}
					if resp.Issues == nil {
						return fmt.Errorf("no issues data returned from API")
					}
					for _, n := range resp.Issues.Nodes {
						nodes = append(nodes, convertAllUserIssueNode(n))
					}
				} else {
					resp, err := api.ListUserIssues(cmd.Context(), client, limit, nil, user)
					if err != nil {
						return fmt.Errorf("listing issues: %w", err)
					}
					if resp.Issues == nil {
						return fmt.Errorf("no issues data returned from API")
					}
					for _, n := range resp.Issues.Nodes {
						nodes = append(nodes, convertUserIssueNode(n))
					}
				}
			} else if all {
				resp, err := api.ListMyAllIssues(cmd.Context(), client, limit, nil)
				if err != nil {
					return fmt.Errorf("listing issues: %w", err)
				}
				if resp.Viewer == nil {
					return fmt.Errorf("no viewer data returned from API")
				}
				if resp.Viewer.AssignedIssues == nil {
					return fmt.Errorf("no assigned issues data returned from API")
				}
				for _, n := range resp.Viewer.AssignedIssues.Nodes {
					var labels *api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnection
					if n.Labels != nil {
						convertedNodes := make([]*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnectionNodesIssueLabel, len(n.Labels.Nodes))
						for i, l := range n.Labels.Nodes {
							convertedNodes[i] = &api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnectionNodesIssueLabel{
								Name: l.Name,
							}
						}
						labels = &api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnection{
							Nodes: convertedNodes,
						}
					}
					nodes = append(nodes, &api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue{
						Id:         n.Id,
						Identifier: n.Identifier,
						Title:      n.Title,
						State:      (*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState)(n.State),
						Priority:   n.Priority,
						UpdatedAt:  n.UpdatedAt,
						Labels:     labels,
					})
				}
			} else {
				resp, err := api.ListMyActiveIssues(cmd.Context(), client, limit, nil)
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
				if refresh {
					if _, err := opts.Cache.Clear(); err != nil {
						return fmt.Errorf("clearing cache: %w", err)
					}
				}
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
	cmd.Flags().BoolVarP(&refresh, "refresh", "r", false, "Clear cached issue details before browsing (use with -i)")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Include completed and canceled issues")
	cmd.Flags().StringVarP(&cycle, "cycle", "c", "", "Filter by cycle: current, next, previous, or a cycle number")
	_ = cmd.RegisterFlagCompletionFunc("cycle", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeCycleValues(cmd, opts)
	})
	cmd.Flags().IntVarP(&limit, "limit", "n", 50, "Maximum number of issues to return")
	cmd.Flags().StringVarP(&sortBy, "sort", "s", "status", "Sort by column: status, priority, identifier, title")
	_ = cmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"status", "priority", "identifier", "title"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().StringVarP(&user, "user", "u", "", "User whose issues to list")
	_ = cmd.RegisterFlagCompletionFunc("user", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeUserNames(cmd, opts)
	})

	return cmd
}

func convertUserIssueNode(n *api.ListUserIssuesIssuesIssueConnectionNodesIssue) *api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue {
	var labels *api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnection
	if n.Labels != nil {
		convertedNodes := make([]*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnectionNodesIssueLabel, len(n.Labels.Nodes))
		for i, l := range n.Labels.Nodes {
			convertedNodes[i] = &api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnectionNodesIssueLabel{
				Name: l.Name,
			}
		}
		labels = &api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnection{
			Nodes: convertedNodes,
		}
	}
	return &api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue{
		Id:         n.Id,
		Identifier: n.Identifier,
		Title:      n.Title,
		State:      (*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState)(n.State),
		Priority:   n.Priority,
		UpdatedAt:  n.UpdatedAt,
		Labels:     labels,
	}
}

func convertAllUserIssueNode(n *api.ListAllUserIssuesIssuesIssueConnectionNodesIssue) *api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue {
	var labels *api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnection
	if n.Labels != nil {
		convertedNodes := make([]*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnectionNodesIssueLabel, len(n.Labels.Nodes))
		for i, l := range n.Labels.Nodes {
			convertedNodes[i] = &api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnectionNodesIssueLabel{
				Name: l.Name,
			}
		}
		labels = &api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnection{
			Nodes: convertedNodes,
		}
	}
	return &api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue{
		Id:         n.Id,
		Identifier: n.Identifier,
		Title:      n.Title,
		State:      (*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState)(n.State),
		Priority:   n.Priority,
		UpdatedAt:  n.UpdatedAt,
		Labels:     labels,
	}
}

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

// formatCycleDate extracts "Jan 2" from an ISO timestamp.
func formatCycleDate(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ""
	}
	return t.Format("Jan 2")
}

// resolveCycle converts a cycle flag value to cycle info.
// Named values (current, next, previous) are resolved via the ListCycles API;
// numeric strings also query ListCycles to get the full metadata.
func resolveCycle(ctx context.Context, client graphql.Client, value string) (cycleInfo, error) {
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

	resp, err := api.ListCycles(ctx, client, 50)
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

// listIssuesByCycle dispatches to the appropriate ByCycle query variant based
// on the user and all flags, then converts results to the common issueNode type.
func listIssuesByCycle(ctx context.Context, client graphql.Client, limit int, user string, all bool, cycleNumber float64) ([]*issueNode, error) {
	if user != "" {
		if all {
			resp, err := api.ListAllUserIssuesByCycle(ctx, client, limit, nil, user, cycleNumber)
			if err != nil {
				return nil, fmt.Errorf("listing issues: %w", err)
			}
			if resp.Issues == nil {
				return nil, fmt.Errorf("no issues data returned from API")
			}
			var nodes []*issueNode
			for _, n := range resp.Issues.Nodes {
				nodes = append(nodes, convertAllUserIssuesByCycleNode(n))
			}
			return nodes, nil
		}
		resp, err := api.ListUserIssuesByCycle(ctx, client, limit, nil, user, cycleNumber)
		if err != nil {
			return nil, fmt.Errorf("listing issues: %w", err)
		}
		if resp.Issues == nil {
			return nil, fmt.Errorf("no issues data returned from API")
		}
		var nodes []*issueNode
		for _, n := range resp.Issues.Nodes {
			nodes = append(nodes, convertUserIssuesByCycleNode(n))
		}
		return nodes, nil
	}

	if all {
		resp, err := api.ListMyAllIssuesByCycle(ctx, client, limit, nil, cycleNumber)
		if err != nil {
			return nil, fmt.Errorf("listing issues: %w", err)
		}
		if resp.Viewer == nil {
			return nil, fmt.Errorf("no viewer data returned from API")
		}
		if resp.Viewer.AssignedIssues == nil {
			return nil, fmt.Errorf("no assigned issues data returned from API")
		}
		var nodes []*issueNode
		for _, n := range resp.Viewer.AssignedIssues.Nodes {
			nodes = append(nodes, convertMyAllIssuesByCycleNode(n))
		}
		return nodes, nil
	}

	resp, err := api.ListMyActiveIssuesByCycle(ctx, client, limit, nil, cycleNumber)
	if err != nil {
		return nil, fmt.Errorf("listing issues: %w", err)
	}
	if resp.Viewer == nil {
		return nil, fmt.Errorf("no viewer data returned from API")
	}
	if resp.Viewer.AssignedIssues == nil {
		return nil, fmt.Errorf("no assigned issues data returned from API")
	}
	var nodes []*issueNode
	for _, n := range resp.Viewer.AssignedIssues.Nodes {
		nodes = append(nodes, convertMyActiveIssuesByCycleNode(n))
	}
	return nodes, nil
}

func convertMyActiveIssuesByCycleNode(n *api.ListMyActiveIssuesByCycleViewerUserAssignedIssuesIssueConnectionNodesIssue) *issueNode {
	var labels *api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnection
	if n.Labels != nil {
		convertedNodes := make([]*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnectionNodesIssueLabel, len(n.Labels.Nodes))
		for i, l := range n.Labels.Nodes {
			convertedNodes[i] = &api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnectionNodesIssueLabel{
				Name: l.Name,
			}
		}
		labels = &api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnection{
			Nodes: convertedNodes,
		}
	}
	return &issueNode{
		Id:         n.Id,
		Identifier: n.Identifier,
		Title:      n.Title,
		State:      (*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState)(n.State),
		Priority:   n.Priority,
		UpdatedAt:  n.UpdatedAt,
		Labels:     labels,
	}
}

func convertMyAllIssuesByCycleNode(n *api.ListMyAllIssuesByCycleViewerUserAssignedIssuesIssueConnectionNodesIssue) *issueNode {
	var labels *api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnection
	if n.Labels != nil {
		convertedNodes := make([]*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnectionNodesIssueLabel, len(n.Labels.Nodes))
		for i, l := range n.Labels.Nodes {
			convertedNodes[i] = &api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnectionNodesIssueLabel{
				Name: l.Name,
			}
		}
		labels = &api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnection{
			Nodes: convertedNodes,
		}
	}
	return &issueNode{
		Id:         n.Id,
		Identifier: n.Identifier,
		Title:      n.Title,
		State:      (*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState)(n.State),
		Priority:   n.Priority,
		UpdatedAt:  n.UpdatedAt,
		Labels:     labels,
	}
}

func convertUserIssuesByCycleNode(n *api.ListUserIssuesByCycleIssuesIssueConnectionNodesIssue) *issueNode {
	var labels *api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnection
	if n.Labels != nil {
		convertedNodes := make([]*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnectionNodesIssueLabel, len(n.Labels.Nodes))
		for i, l := range n.Labels.Nodes {
			convertedNodes[i] = &api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnectionNodesIssueLabel{
				Name: l.Name,
			}
		}
		labels = &api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnection{
			Nodes: convertedNodes,
		}
	}
	return &issueNode{
		Id:         n.Id,
		Identifier: n.Identifier,
		Title:      n.Title,
		State:      (*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState)(n.State),
		Priority:   n.Priority,
		UpdatedAt:  n.UpdatedAt,
		Labels:     labels,
	}
}

func convertAllUserIssuesByCycleNode(n *api.ListAllUserIssuesByCycleIssuesIssueConnectionNodesIssue) *issueNode {
	var labels *api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnection
	if n.Labels != nil {
		convertedNodes := make([]*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnectionNodesIssueLabel, len(n.Labels.Nodes))
		for i, l := range n.Labels.Nodes {
			convertedNodes[i] = &api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnectionNodesIssueLabel{
				Name: l.Name,
			}
		}
		labels = &api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnection{
			Nodes: convertedNodes,
		}
	}
	return &issueNode{
		Id:         n.Id,
		Identifier: n.Identifier,
		Title:      n.Title,
		State:      (*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState)(n.State),
		Priority:   n.Priority,
		UpdatedAt:  n.UpdatedAt,
		Labels:     labels,
	}
}

type issueNode = api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue

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
