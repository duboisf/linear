# Adding Commands

Step-by-step checklist for adding a new command to the Linear CLI.

## Checklist

### 1. Create `cmd/<command>.go`

Use `cmd/<parent>_<subcommand>.go` for subcommands (e.g., `cmd/issue_get.go`).

### 2. Write the constructor taking `opts Options`

```go
func newMyCmd(opts Options) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "mycommand [ARGS]",
        Short: "One-line description",
        RunE: func(cmd *cobra.Command, args []string) error {
            client, err := resolveClient(cmd, opts)
            if err != nil {
                return err
            }
            // command logic using opts.Stdout, opts.Cache, etc.
            return nil
        },
        ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
            if len(args) > 0 {
                return nil, cobra.ShellCompDirectiveNoFileComp
            }
            return dynamicCompletions, cobra.ShellCompDirectiveNoFileComp
        },
    }
```

The `RunE` closure captures `opts` for dependency injection. See `cmd/issue_get.go`.

### 3. Register flag completions

For every flag with a fixed set of values, call `RegisterFlagCompletionFunc`:

```go
cmd.Flags().StringVarP(&format, "output", "o", "plain", "Output format")
cmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    return []string{"plain", "markdown", "json", "yaml"}, cobra.ShellCompDirectiveNoFileComp
})
```

### 4. Wire into the parent command

In the parent's file (e.g., `cmd/root.go` or `cmd/issue.go`):

```go
parent.AddCommand(newMyCmd(opts))
```

### 5. Create `cmd/<command>_test.go`

Use the **external test package** (`package cmd_test`) and mock helpers:

```go
package cmd_test

func TestMyCmd_Success(t *testing.T) {
    t.Parallel()
    server := newMockGraphQLServer(t, map[string]string{
        "OperationName": `{"data": { ... }}`,
    })
    opts, stdout, _ := testOptionsWithBuffers(t, server)
    root := cmd.NewRootCmd(opts)
    root.SetArgs([]string{"mycommand", "arg1"})
    if err := root.Execute(); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    // assert on stdout.String()
}
```

Test helpers in `cmd/helpers_test.go`:
- `newMockGraphQLServer(t, handlers)` -- routes by GraphQL operation name
- `newErrorGraphQLServer(t)` -- always returns errors
- `testOptionsWithBuffers(t, server)` -- returns stdout/stderr buffers
- `testOptionsKeyringError(t)` -- simulates keyring resolution failure

### 6. Run tests and verify completions

```bash
go test -race ./cmd/...
linear completion zsh | source /dev/stdin
linear mycommand <TAB>
```
