package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/format"
)

// newIssueListCmd creates the "issue list" subcommand that lists issues
// assigned to the authenticated user.
func newIssueListCmd(opts Options) *cobra.Command {
	var (
		all   bool
		limit int
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

			out := format.FormatIssueList(nodes, format.ColorEnabled(cmd.OutOrStdout()))
			fmt.Fprint(opts.Stdout, out)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Include completed and canceled issues")
	cmd.Flags().IntVarP(&limit, "limit", "n", 50, "Maximum number of issues to return")

	return cmd
}
