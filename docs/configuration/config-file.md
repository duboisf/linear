# Config File

User configuration is stored in `$XDG_CONFIG_HOME/linear/config.yaml` (typically `~/.config/linear/config.yaml`).

If the file does not exist, all settings use their default values. An annotated example file (`config.example.yaml`) is written alongside it on every invocation.

## Format

```yaml
interactive:
  commands:
    - name: "Claude"
      exec: true
      command: "claude {{.Title}}"
    - name: "Open in browser"
      command: "xdg-open {{.Raw.URL}}"
```

## Fields

### `interactive.commands`

A list of custom commands available via `ctrl-o` in interactive mode. Each command has:

- `name` — shown in the picker
- `command` — a shell command using Go template syntax for issue fields
- `exec` (optional, default `false`) — when `true`, exits fzf before running the command. Use this for long-running or interactive programs (e.g. Claude, editors) so you don't return to fzf when they exit.

When pressing `ctrl-o`, a nested fzf picker lets you choose which command to run. If only one command is configured, the picker is skipped.

If no commands are configured, pressing `ctrl-o` shows a help message explaining how to set them up.

**Default:** none (empty list)

#### Go template syntax

Command strings are parsed as [Go templates](https://pkg.go.dev/text/template), giving access to all issue fields. **All fields are shell-quoted by default** to prevent injection from untrusted issue data (e.g. malicious titles). Use `{{.Raw.Field}}` for the unquoted value when needed (e.g. URLs passed to `xdg-open`).

| Field        | Description                          | Example output                      |
|-------------|--------------------------------------|-------------------------------------|
| Identifier  | Issue ID (quoted)                    | `'AIS-123'`                         |
| Title       | Issue title (quoted)                 | `'Fix login bug'`                   |
| Description | Markdown description (quoted)        | `'Users can'\''t log in'`           |
| URL         | Linear URL (quoted)                  | `'https://linear.app/team/AIS-123'` |
| BranchName  | Suggested git branch (quoted)        | `'fred/ais-123-fix-login-bug'`      |
| State       | Workflow state name (quoted)         | `'In Progress'`                     |
| Priority    | Priority label (quoted)              | `'High'`                            |
| Assignee    | Assigned user name (quoted)          | `'Fred'`                            |
| Team        | Team name (quoted)                   | `'Aisystems'`                       |
| TeamKey     | Team key prefix (quoted)             | `'AIS'`                             |
| Cycle       | Cycle name (quoted)                  | `'Sprint 12'`                       |
| Project     | Project name (quoted)                | `'My Project'`                      |
| Labels      | Comma-separated label names (quoted) | `'bug,frontend'`                    |
| DueDate     | Due date string (quoted)             | `'2026-03-01'`                      |
| Parent      | Parent issue identifier (quoted)     | `'AIS-10'`                          |
| Raw.*       | Any field above, unquoted            | `AIS-123`                           |

Examples:

```yaml
interactive:
  commands:
    # Fields are shell-quoted by default — safe to use directly as arguments
    - name: "Claude"
      exec: true
      command: "claude {{.Title}}"
    # Use .Raw for values that shouldn't be quoted (e.g. URLs)
    - name: "Open in browser"
      command: "xdg-open {{.Raw.URL}}"
    - name: "Checkout branch"
      exec: true
      command: "git checkout -b {{.Raw.BranchName}}"
    # Use printf %s with quoted args for formatted display
    - name: "Summary"
      command: "printf '%s [%s]\\n' {{.Identifier}} {{.State}}"
```

Go templates support conditionals (`{{if .Raw.Field}}...{{end}}`), and other standard template actions. Use `.Raw` for truthiness checks since quoted empty strings are non-empty (`''`).

## Editing

Run `linear config edit` to open the config file in your `$VISUAL` or `$EDITOR` (falls back to `vi`). If the file doesn't exist, it is created with a starter file before opening.

## Loading Behavior

- Missing file: defaults returned (not an error).
- Empty `commands`: no commands available (ctrl-o shows help).
- Invalid YAML: returns a parse error.
- Config directory resolution failure: defaults returned.
