# Environment Variables

All environment variables recognized by the Linear CLI.

## Authentication

| Variable | Description |
|----------|-------------|
| `LINEAR_API_KEY` | API key for Linear. Highest priority in the credential provider chain (checked before native keyring and file store). Set this to skip interactive setup entirely. |

## Display

| Variable | Description |
|----------|-------------|
| `NO_COLOR` | Disable all ANSI color output. Follows the [no-color.org](https://no-color.org) convention. Any value (including empty) disables color when the variable is set. |
| `LINEAR_GLAMOUR_STYLE` | Force `dark` or `light` theme for glamour markdown rendering. Used internally: the parent process sets this when spawning fzf subprocesses so the child inherits the detected terminal background. You can set it manually to override terminal detection. |

## Directories

| Variable | Default | Description |
|----------|---------|-------------|
| `XDG_CACHE_HOME` | `~/.cache` | Base cache directory. The CLI stores cached API responses at `$XDG_CACHE_HOME/linear/`. Cache has a 5-minute default TTL (24 hours for users, labels, and cycles). |
| `XDG_CONFIG_HOME` | `~/.config` | Base config directory. File-based credentials are stored at `$XDG_CONFIG_HOME/linear/credentials` with 0600 permissions. |

### Cache fallback

If `os.UserCacheDir()` fails (which reads `XDG_CACHE_HOME` on Linux), the cache falls back to:

```
$TMPDIR/linear-cache    (or /tmp/linear-cache if TMPDIR is unset)
```

This is determined by Go's `os.TempDir()`, which reads the `TMPDIR` environment variable.

## Credential Resolution Order

The CLI resolves the API key using a chain provider, tried in order:

1. `LINEAR_API_KEY` environment variable (`EnvProvider`)
2. Native OS keyring (`KeychainProvider` on macOS, `SecretToolProvider` on Linux)
3. File at `$XDG_CONFIG_HOME/linear/credentials` (`FileProvider`)
4. Interactive prompt (if all above fail, prompts user and stores the key)
