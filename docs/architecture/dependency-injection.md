# Dependency Injection

All commands receive an `Options` struct -- no globals, no singletons.

## The `Options` Struct

Defined in `cmd/root.go`:

```go
type Options struct {
    NewAPIClient       func(apiKey string) graphql.Client // GraphQL client factory
    KeyringProvider    keyring.Provider                   // resolves API keys (chain: env -> native -> file)
    Prompter           keyring.Prompter                   // interactive API key prompt
    NativeStore        keyring.Provider                   // platform-specific credential store
    FileStore          keyring.Provider                   // file-based fallback credential store
    GitWorktreeCreator GitWorktreeCreator                 // git worktree operations
    Cache              *cache.Cache                       // file-based cache with TTL
    TimeNow            func() time.Time                   // clock (overridable for tests)
    Stdin              io.Reader                          // standard input
    Stdout             io.Writer                          // standard output
    Stderr             io.Writer                          // standard error
}
```

Every command constructor takes `opts Options` as its only parameter:

```go
func newIssueListCmd(opts Options) *cobra.Command { ... }
func newUserGetCmd(opts Options)   *cobra.Command { ... }
```

## Production Configuration

`DefaultOptions()` builds the real dependency graph:

- **KeyringProvider**: `ChainProvider` that tries `EnvProvider` -> native keyring -> `FileProvider`
- **Native keyring**: `KeychainProvider` on macOS, `SecretToolProvider` on Linux
- **Cache**: file-based in `$XDG_CACHE_HOME/linear` (or `$TMPDIR/linear-cache`), 5-minute TTL
- **I/O**: wired to `os.Stdin`, `os.Stdout`, `os.Stderr`
- **TimeNow**: `time.Now`

## Testing with Mocks

Tests replace every dependency. See `cmd/helpers_test.go` for shared helpers:

```go
// testOptions wires a mock GraphQL server into Options
func testOptions(t *testing.T, server *httptest.Server) cmd.Options {
    var stdout, stderr bytes.Buffer
    return cmd.Options{
        NewAPIClient:    func(apiKey string) graphql.Client {
            return graphql.NewClient(server.URL, server.Client())
        },
        KeyringProvider: &staticProvider{key: "test-api-key"},
        Prompter:        &noopPrompter{},
        NativeStore:     &staticProvider{key: "test-api-key"},
        Stdout:          &stdout,
        Stderr:          &stderr,
    }
}
```

Key mock types in `helpers_test.go`:

| Mock | Replaces | Purpose |
|------|----------|---------|
| `staticProvider` | `keyring.Provider` | Returns a fixed API key |
| `errorProvider` | `keyring.Provider` | Always errors (tests error paths) |
| `noopPrompter` | `keyring.Prompter` | Returns a test key without prompting |
| `mockGitWorktreeCreator` | `GitWorktreeCreator` | Records calls, returns configurable results |
| `newMockGraphQLServer` | Real Linear API | Routes by `operationName`, returns canned JSON |

### Typical test pattern

```go
func TestIssueList(t *testing.T) {
    server := newMockGraphQLServer(t, map[string]string{
        "ListMyActiveIssues": `{"data": {...}}`,
    })
    opts, stdout, _ := testOptionsWithBuffers(t, server)
    root := cmd.NewRootCmd(opts)
    root.SetArgs([]string{"issue", "list"})
    err := root.Execute()
    // assert on err, stdout.String(), etc.
}
```

No real API calls, no real keyring, no real filesystem -- fully deterministic.
