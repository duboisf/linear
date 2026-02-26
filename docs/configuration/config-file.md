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

The prompt template sent to `claude` when pressing `ctrl-w` in interactive mode.

**Default:** `Let's work on linear issue {identifier}`

#### Legacy syntax

The placeholder `{identifier}` is replaced with the selected issue's identifier (e.g., `AIS-123`).

```yaml
interactive:
  claude_prompt: "Let's work on linear issue {identifier}"
```

#### Go template syntax

If the prompt contains `{{`, it is parsed as a [Go template](https://pkg.go.dev/text/template). This gives access to all issue fields:

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

Example:

```yaml
interactive:
  claude_prompt: "Work on {{.Identifier}}: {{.Title}} ({{.State}}, {{.Priority}} priority){{if .BranchName}}\nBranch: {{.BranchName}}{{end}}"
```

Go templates support conditionals (`{{if .Field}}...{{end}}`), range loops (`{{range .Labels}}...{{end}}`), and other standard template actions.

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
