# Command Structure

The CLI uses [Cobra](https://github.com/spf13/cobra) with a strict hierarchy.

## Command Tree

```
linear                          (root)
  |-- issue  (alias: i)         [Core Commands]
  |     |-- list
  |     |-- get
  |     |-- edit
  |     |-- edit-interactive    (hidden)
  |     +-- worktree
  |-- user   (alias: u)         [Core Commands]
  |     |-- list
  |     +-- get
  |-- cache                     [Setup Commands]
  |     +-- clear
  |-- completion                [Setup Commands]
  +-- version                   [Setup Commands]
```

## Command Groups

Commands are organized into two groups displayed in help output:

```go
root.AddGroup(
    &cobra.Group{ID: "core",  Title: "Core Commands:"},
    &cobra.Group{ID: "setup", Title: "Setup Commands:"},
)
```

- **Core Commands**: `issue`, `user` -- day-to-day issue tracking
- **Setup Commands**: `cache`, `completion`, `version` -- maintenance and shell setup

## Root Command Configuration

```go
root := &cobra.Command{
    Use:           "linear",
    SilenceUsage:  true,   // don't dump usage on every error
    SilenceErrors: true,   // errors handled by caller (main.go)
    PersistentPreRunE: ..., // --refresh flag clears cache before any subcommand
}
```

The `--refresh` / `-r` persistent flag clears the cache before running any subcommand.

## Parent Command Pattern

Parent commands (`issue`, `user`, `cache`) have **no `RunE`**. They exist only to group
subcommands via `AddCommand`:

```go
func newIssueCmd(opts Options) *cobra.Command {
    cmd := &cobra.Command{
        Use:     "issue",
        Aliases: []string{"i"},
        Short:   "Manage Linear issues",
        ValidArgsFunction: func(...) ([]string, cobra.ShellCompDirective) {
            return nil, cobra.ShellCompDirectiveNoFileComp
        },
    }
    cmd.AddCommand(
        newIssueEditCmd(opts),
        newIssueEditInteractiveCmd(opts),
        newIssueGetCmd(opts),
        newIssueListCmd(opts),
        newIssueWorktreeCmd(opts),
    )
    return cmd
}
```

## Shell Completion Convention

Every parent command and the root set `ValidArgsFunction` to return
`cobra.ShellCompDirectiveNoFileComp`. This prevents shell completion from falling
back to filesystem paths when no subcommand matches.

Leaf commands with flags use `cmd.RegisterFlagCompletionFunc` for flag values
(e.g., `--sort`, `--format`).

## Hidden Commands

`edit-interactive` is marked `Hidden: true`. It is invoked internally by the
fzf-based issue browser (ctrl-e binding) and not intended for direct user
invocation. It launches nested fzf pickers for field and value selection, then
updates the issue via the API.

## Adding a New Command

1. Create `cmd/<parent>_<action>.go` with `new<Parent><Action>Cmd(opts Options)`.
2. Add it to the parent's `AddCommand(...)` call.
3. Set `ValidArgsFunction` (return `NoFileComp` if no completions apply).
4. Register flag completion functions for flags with enumerated values.
5. Create `cmd/<parent>_<action>_test.go` (`package cmd_test`) using `helpers_test.go` mocks.
