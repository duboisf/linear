package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/format"
)

// newIssueEditCmd creates the "issue edit" subcommand that modifies
// properties of an existing issue.
func newIssueEditCmd(opts Options) *cobra.Command {
	var (
		cycle string
		user  string
	)

	cmd := &cobra.Command{
		Use:     "edit [IDENTIFIER]",
		Aliases: []string{"e"},
		Short:   "Edit an issue",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cycle == "" {
				return fmt.Errorf("at least one edit flag is required (e.g. --cycle)")
			}

			client, err := resolveClient(cmd, opts)
			if err != nil {
				return err
			}

			var identifier string
			if len(args) > 0 {
				identifier = args[0]
			} else {
				var issues []issueForCompletion
				if user != "" {
					issues, err = fetchUserIssues(cmd.Context(), client, user)
				} else {
					issues, err = fetchMyIssues(cmd.Context(), client)
				}
				if err != nil {
					return fmt.Errorf("listing issues: %w", err)
				}
				identifier, err = fzfPickIssue(issues)
				if err != nil {
					return err
				}
				if identifier == "" {
					return nil // user cancelled
				}
			}

			// Resolve the issue to get its UUID.
			resp, err := api.GetIssue(cmd.Context(), client, identifier)
			if err != nil {
				return fmt.Errorf("getting issue: %w", err)
			}
			if resp.Issue == nil {
				return fmt.Errorf("issue %s not found", identifier)
			}

			// Resolve --cycle flag.
			timeNow := opts.TimeNow
			if timeNow == nil {
				timeNow = time.Now
			}
			ci, err := resolveCycle(cmd.Context(), client, opts.Cache, timeNow, cycle)
			if err != nil {
				return err
			}

			updateResp, err := api.UpdateIssueCycle(cmd.Context(), client, resp.Issue.Id, ci.Id)
			if err != nil {
				return fmt.Errorf("updating issue: %w", err)
			}
			if updateResp.IssueUpdate == nil || !updateResp.IssueUpdate.Success {
				return fmt.Errorf("issue update was not successful")
			}

			// Re-fetch and re-cache the issue preview so interactive browsing
			// shows fresh data immediately (e.g. after ctrl-y in fzf).
			if opts.Cache != nil {
				refreshIssueCache(cmd.Context(), client, opts.Cache, identifier)
			}

			colorEnabled := format.ColorEnabled(cmd.OutOrStdout())
			issueID := format.Colorize(colorEnabled, format.Bold, identifier)
			cycleLabel := format.Colorize(colorEnabled, format.Bold+format.Cyan, fmt.Sprintf("Cycle %.0f", ci.Number))
			if ci.Name != "" {
				cycleLabel += " - " + ci.Name
			}
			fmt.Fprintf(opts.Stdout, "Updated %s cycle to %s\n", issueID, cycleLabel)

			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			if user != "" {
				return completeUserIssues(cmd, opts, user)
			}
			return completeMyIssues(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&cycle, "cycle", "c", "", "Set cycle: current, next, previous, or a cycle number")
	_ = cmd.RegisterFlagCompletionFunc("cycle", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeCycleValues(cmd, opts)
	})

	cmd.Flags().StringVarP(&user, "user", "u", "", "User whose issues to browse")
	_ = cmd.RegisterFlagCompletionFunc("user", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeUserNames(cmd, opts)
	})

	return cmd
}
