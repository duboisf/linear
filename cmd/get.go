package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/format"
)

func newGetCmd(opts Options) *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "get <user> <resource> <identifier>",
		Short: "Get or create a resource (issue, worktree)",
		Args:  cobra.ExactArgs(3),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			switch len(args) {
			case 0: // completing user
				return completeUsers(cmd, opts)
			case 1: // completing resource
				return []string{
					"issue\tGet issue details",
					"worktree\tCreate a git worktree for an issue",
				}, cobra.ShellCompDirectiveNoFileComp
			case 2: // completing identifier
				if args[0] == "@my" {
					return completeMyIssues(cmd, opts)
				}
				return completeUserIssues(cmd, opts, args[0])
			default:
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			resource := args[1]
			identifier := args[2]

			client, err := resolveClient(cmd, opts)
			if err != nil {
				return err
			}

			switch resource {
			case "issue":
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

			case "worktree":
				return runWorktreeCreate(cmd.Context(), client, identifier, opts.GitWorktreeCreator, opts.Stdout)

				default:
				return fmt.Errorf("unsupported resource %q; valid resources: issue, worktree", resource)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "plain", "Output format: plain, markdown, json, yaml")
	cmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return validOutputFormats, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}
