# Config File

User configuration is stored in `$XDG_CONFIG_HOME/linear/config.yaml` (typically `~/.config/linear/config.yaml`).

If the file does not exist, all settings use their default values.

## Format

```yaml
interactive:
  commands:
    - name: "Claude"
      command: "claude \"Work on {{.Identifier}}: {{.Title}}\""
    - name: "Open in browser"
      command: "xdg-open {{.URL}}"
```

## Fields

### `interactive.commands`

A list of custom commands available via `ctrl-o` in interactive mode. Each command has a `name` (shown in the picker) and a `command` (a shell command using Go template syntax for issue fields). When pressing `ctrl-o`, a nested fzf picker lets you choose which command to run. If only one command is configured, the picker is skipped.

If no commands are configured, pressing `ctrl-o` shows a help message explaining how to set them up.

**Default:** none (empty list)

#### Go template syntax

Command strings are parsed as [Go templates](https://pkg.go.dev/text/template), giving access to all issue fields:

| Field        | Description                          | Example                           |
|-------------|--------------------------------------|-----------------------------------|
| Identifier  | Issue ID                             | `AIS-123`                         |
| Title       | Issue title                          | `Fix login bug`                   |
| Description | Markdown description                 | `Users can't log in when...`      |
| URL         | Linear URL                           | `https://linear.app/team/AIS-123` |
| BranchName  | Suggested git branch                 | `fred/ais-123-fix-login-bug`      |
| State       | Workflow state name                  | `In Progress`                     |
| Priority    | Priority label                       | `High`                            |
| Assignee    | Assigned user name                   | `Fred`                            |
| Team        | Team name                            | `Aisystems`                       |
| TeamKey     | Team key prefix                      | `AIS`                             |
| Cycle       | Cycle name                           | `Sprint 12`                       |
| Project     | Project name                         | `My Project`                      |
| Labels      | Label names (slice, use with range)  | `["bug", "frontend"]`            |
| DueDate     | Due date string                      | `2026-03-01`                      |
| Parent      | Parent issue identifier              | `AIS-10`                          |

Examples:

```yaml
interactive:
  commands:
    - name: "Claude"
      command: "claude \"Work on {{.Identifier}}: {{.Title}}\""
    - name: "Claude (skip permissions)"
      command: "claude --dangerously-skip-permissions \"Work on {{.Identifier}}: {{.Title}}\""
    - name: "Open in browser"
      command: "xdg-open {{.URL}}"
    - name: "Checkout branch"
      command: "git checkout -b {{.BranchName}}"
```

Go templates support conditionals (`{{if .Field}}...{{end}}`), range loops (`{{range .Labels}}...{{end}}`), and other standard template actions.

## Editing

Run `linear config edit` to open the config file in your `$VISUAL` or `$EDITOR` (falls back to `vi`). If the file doesn't exist, it is created with a commented-out example before opening.

## Loading Behavior

- Missing file: defaults returned (not an error).
- Empty `commands`: no commands available (ctrl-o shows help).
- Invalid YAML: returns a parse error.
- Config directory resolution failure: defaults returned.
