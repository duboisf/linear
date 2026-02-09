package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/format"
	"github.com/duboisf/linear/internal/tui"
)

// newIssueGetCmd creates the "issue get" subcommand that displays detailed
// information for a specific issue.
func newIssueGetCmd(opts Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get [IDENTIFIER]",
		Aliases: []string{"show", "view"},
		Short:   "Get details for a specific issue",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := resolveClient(cmd, opts)
			if err != nil {
				return err
			}

			var identifier string
			if len(args) > 0 {
				identifier = args[0]
			} else {
				// Interactive mode: select from list
				f, ok := opts.Stdin.(*os.File)
				if !ok || !term.IsTerminal(int(f.Fd())) {
					return fmt.Errorf("no issue identifier provided; run interactively or pass an identifier")
				}

				resp, err := api.ListMyActiveIssues(cmd.Context(), client, 50, nil)
				if err != nil {
					return fmt.Errorf("listing issues: %w", err)
				}
				if resp.Viewer == nil || resp.Viewer.AssignedIssues == nil {
					return fmt.Errorf("no assigned issues data returned from API")
				}

				wrapped := tui.WrapIssues[
					*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState,
					*api.ListMyActiveIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue,
				](resp.Viewer.AssignedIssues.Nodes)

				identifier, err = tui.RunSelector(wrapped, opts.Stdin, opts.Stdout)
				if err != nil {
					return fmt.Errorf("selecting issue: %w", err)
				}
				if identifier == "" {
					return nil // user cancelled
				}
			}

			resp, err := api.GetIssue(cmd.Context(), client, identifier)
			if err != nil {
				return fmt.Errorf("getting issue: %w", err)
			}

			if resp.Issue == nil {
				return fmt.Errorf("issue %s not found", identifier)
			}

			out := format.FormatIssueDetail(resp.Issue, format.ColorEnabled(cmd.OutOrStdout()))
			fmt.Fprint(opts.Stdout, out)

			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			client, err := resolveClient(cmd, opts)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			resp, err := api.ActiveIssuesForCompletion(cmd.Context(), client, 100)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			if resp.Viewer == nil || resp.Viewer.AssignedIssues == nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			var completions []string
			for _, issue := range resp.Viewer.AssignedIssues.Nodes {
				completions = append(completions, fmt.Sprintf("%s\t%s", issue.Identifier, issue.Title))
			}

			return completions, cobra.ShellCompDirectiveNoFileComp
		},
	}

	return cmd
}
