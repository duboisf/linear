# Config File

User configuration is stored in `$XDG_CONFIG_HOME/linear/config.yaml` (typically `~/.config/linear/config.yaml`).

If the file does not exist, all settings use their default values.

## Format

```yaml
interactive:
  claude_prompt: "Let's work on linear issue {identifier}"
  claude_modes:
    - label: "Claude"
    - label: "Claude (skip permissions)"
      args: "--dangerously-skip-permissions"
```

## Fields

### `interactive.claude_prompt`

The prompt template sent to `claude` when pressing `ctrl-w` in interactive mode. The placeholder `{identifier}` is replaced with the selected issue's identifier (e.g., `AIS-123`).

**Default:** `Let's work on linear issue {identifier}`

### `interactive.claude_modes`

A list of named launch modes for `claude`. Each mode has a `label` (shown in the picker) and optional `args` (extra CLI flags passed to `claude`). When pressing `ctrl-w`, a nested fzf picker lets you choose which mode to use. If only one mode is configured, the picker is skipped.

**Default:**

```yaml
claude_modes:
  - label: "Claude"
  - label: "Claude (skip permissions)"
    args: "--dangerously-skip-permissions"
```

## Editing

Run `linear config edit` to open the config file in your `$VISUAL` or `$EDITOR` (falls back to `vi`). If the file doesn't exist, it is created with default values before opening.

## Loading Behavior

- Missing file: defaults returned (not an error).
- Empty `claude_prompt`: falls back to the default.
- Invalid YAML: returns a parse error.
- Config directory resolution failure: defaults returned.
