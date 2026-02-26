# Shell Completions

Every command and subcommand MUST provide shell completions for all arguments and flag values:

- **Positional arguments**: Use `ValidArgsFunction` to provide dynamic completions (e.g., user names, issue identifiers from the API).
- **Flag values**: Use `cmd.RegisterFlagCompletionFunc` for any flag that accepts a fixed set of values (e.g., `--sort`, `--format`).
- **File suppression**: Return `cobra.ShellCompDirectiveNoFileComp` when completions should not fall back to file paths.

## Verifying Completions

When adding a new command, verify completions work by running:

```
linear completion zsh | source /dev/stdin
linear <command> <TAB>
```
