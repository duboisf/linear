# Config File

User configuration is stored in `$XDG_CONFIG_HOME/linear/config.yaml` (typically `~/.config/linear/config.yaml`).

If the file does not exist, all settings use their default values.

## Format

```yaml
interactive:
  claude_prompt: "Let's work on linear issue {identifier}"
```

## Fields

### `interactive.claude_prompt`

The prompt template sent to `claude` when pressing `ctrl-w` in interactive mode. The placeholder `{identifier}` is replaced with the selected issue's identifier (e.g., `AIS-123`).

**Default:** `Let's work on linear issue {identifier}`

## Loading Behavior

- Missing file: defaults returned (not an error).
- Empty `claude_prompt`: falls back to the default.
- Invalid YAML: returns a parse error.
- Config directory resolution failure: defaults returned.
