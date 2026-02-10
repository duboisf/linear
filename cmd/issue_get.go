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

// validOutputFormats lists the accepted values for --output.
var validOutputFormats = []string{"plain", "markdown", "json", "yaml"}

// newIssueGetCmd creates the "issue get" subcommand that displays detailed
// information for a specific issue.
func newIssueGetCmd(opts Options) *cobra.Command {
	var outputFormat string

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

			var out string
			switch outputFormat {
			case "json":
				out, err = format.FormatIssueDetailJSON(resp.Issue)
				if err != nil {
					return err
				}
			case "yaml":
				out = format.FormatIssueDetailYAML(resp.Issue)
			case "markdown", "md":
				out = format.FormatIssueDetailMarkdown(resp.Issue)
			default:
				out = format.FormatIssueDetail(resp.Issue, format.ColorEnabled(cmd.OutOrStdout()))
			}
			fmt.Fprint(opts.Stdout, out)

			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return completeMyIssues(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "plain", "Output format: plain, markdown, json, yaml")
	cmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return validOutputFormats, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}
