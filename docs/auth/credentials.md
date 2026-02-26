# Credentials

How the CLI resolves and stores Linear API keys.

Source: `internal/keyring/` package; wired up in `cmd/root.go` (`DefaultOptions`, `resolveClient`).

## Provider Interface

Every credential backend implements `keyring.Provider`:

```go
type Provider interface {
    GetAPIKey() (string, error)
    StoreAPIKey(key string) error
}
```

## ChainProvider — Resolution Order

`ChainProvider` wraps multiple providers and tries each in sequence. The first provider that returns a key without error wins. The default chain configured in `DefaultOptions()` is:

| Priority | Provider             | Source                                              |
|----------|----------------------|-----------------------------------------------------|
| 1        | `EnvProvider`        | `LINEAR_API_KEY` environment variable               |
| 2        | Native store         | OS keyring (see platform table below)               |
| 3        | `FileProvider`       | `$XDG_CONFIG_HOME/linear/credentials` (file, 0600)  |

If all three fail, `ChainProvider.GetAPIKey()` returns `ErrNoAPIKey`.

## Platform-Specific Native Stores

Selected at runtime in `nativeKeyringProvider()` (`cmd/root.go`):

| OS      | Provider              | External tool   | Keyring attributes                              |
|---------|-----------------------|-----------------|--------------------------------------------------|
| macOS   | `KeychainProvider`    | `security` CLI  | service=`linear`, account=`default`              |
| Linux   | `SecretToolProvider`  | `secret-tool`   | service=`linear`, account=`default`              |

`SecretToolProvider` passes the key via stdin to avoid exposing it in process arguments. `KeychainProvider` uses `-U` to upsert.

If the native tool binary is not found, providers return `ErrToolNotFound`.

## File Storage Fallback

`FileProvider` stores the key at:

```
$(os.UserConfigDir)/linear/credentials
```

Typically `~/.config/linear/credentials` on Linux, `~/Library/Application Support/linear/credentials` on macOS.

- Directory created with `0700`, file written with `0600`.
- `FileSystem` interface abstracts `os.ReadFile`/`os.WriteFile`/`os.MkdirAll` for testing.

## Interactive Resolution Flow (`keyring.Resolve`)

When the chain returns no key, `Resolve()` falls through to an interactive prompt:

```
Chain lookup ──► key found? ──yes──► return key
                    │
                   no
                    ▼
         Prompter.PromptForAPIKey()   (reads key without echo via term.ReadPassword)
                    │
                    ▼
         Try NativeStore.StoreAPIKey()
              │              │
           success      ErrToolNotFound ──► print install hint
              │              │
           return       Ask user: "Store in local config file? [y/N]"
                             │          │
                            yes        no/decline
                             │          │
                     FileStore.Store   "You will be prompted again next time."
```

Key detail: **non-blocking** -- if the user declines to save, the key is still used for the current session. Nothing is persisted and the user is simply prompted again next run.

## Prompter Interface

```go
type Prompter interface {
    PromptForAPIKey(stdin io.Reader, stdout io.Writer) (string, error)
}
```

Production: `InteractivePrompter` (uses `term.ReadPassword` to hide input).
Tests: inject a mock `Prompter` via the `Options` struct to avoid terminal I/O.

## Testing Seams

Every provider exposes a field for dependency injection:

- `EnvProvider.LookupEnv` -- override `os.LookupEnv`.
- `KeychainProvider.CommandRunner` / `SecretToolProvider.CommandRunner` -- override `exec.Command`.
- `FileProvider.FS` / `FileProvider.ConfigDir` -- override filesystem and config path.
- `InteractivePrompter.ReadPassword` -- override `term.ReadPassword`.
- `ResolveOptions.ReadLine` -- override stdin line reads in confirmation prompts.
