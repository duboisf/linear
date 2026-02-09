package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/format"
)

func newListCmd(opts Options) *cobra.Command {
	var (
		all   bool
		limit int
	)

	cmd := &cobra.Command{
		Use:   "list <user> <resource>",
		Short: "List resources for a user",
		Args:  cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			switch len(args) {
			case 0: // completing user
				client, err := resolveClient(cmd, opts)
				if err != nil {
					return nil, cobra.ShellCompDirectiveNoFileComp
				}
				resp, err := api.UsersForCompletion(cmd.Context(), client, 100)
				if err != nil || resp.Users == nil {
					return nil, cobra.ShellCompDirectiveNoFileComp
				}
				comps := []string{"@my\tYour own issues"}
				for _, u := range resp.Users.Nodes {
					firstName := strings.ToLower(strings.Fields(u.DisplayName)[0])
					comps = append(comps, fmt.Sprintf("%s\t%s", firstName, u.DisplayName))
				}
				return comps, cobra.ShellCompDirectiveNoFileComp
			case 1: // completing resource
				return []string{"issues\tList issues"}, cobra.ShellCompDirectiveNoFileComp
			default:
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			user := args[0]
			resource := args[1]

			if resource != "issues" {
				return fmt.Errorf("unsupported resource %q; valid resources: issues", resource)
			}

			if limit <= 0 {
				return fmt.Errorf("--limit must be greater than 0, got %d", limit)
			}

			client, err := resolveClient(cmd, opts)
			if err != nil {
				return err
			}

			var nodes []*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue

			if user == "@my" {
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
			} else {
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
