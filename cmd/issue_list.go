package cmd

import (
	"fmt"
	"slices"
	"strings"

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
		all    bool
		limit  int
		sortBy string
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

			var nodes []*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue
			if all {
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
				// Convert to the active type for formatting
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

			out := format.FormatIssueList(nodes, format.ColorEnabled(cmd.OutOrStdout()))
			fmt.Fprint(opts.Stdout, out)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Include completed and canceled issues")
	cmd.Flags().IntVarP(&limit, "limit", "n", 50, "Maximum number of issues to return")
	cmd.Flags().StringVarP(&sortBy, "sort", "s", "status", "Sort by column: status, priority, identifier, title")
	_ = cmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"status", "priority", "identifier", "title"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
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
