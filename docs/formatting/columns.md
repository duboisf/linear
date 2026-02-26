# Columns

The column system controls which fields appear in `linear issue list` output.

## Column Registry

`_columnRegistry` in `internal/format/column.go` maps column name strings to `ColumnDef` structs:

```go
type ColumnDef struct {
    Header string                           // e.g. "STATUS", "PRIORITY"
    Value  func(issue *issueListNode) string // extracts the display value
    Color  func(issue *issueListNode) string // returns an ANSI code (or "" for no color)
}
```

## Available Columns

| Name       | Header     | Color logic                        |
|------------|------------|------------------------------------|
| `id`       | IDENTIFIER | none                               |
| `status`   | STATUS     | `StateColor()` by workflow type    |
| `priority` | PRIORITY   | `PriorityColor()` by rank          |
| `labels`   | LABELS     | always Cyan                        |
| `title`    | TITLE      | none                               |
| `updated`  | UPDATED    | Gray                               |
| `created`  | CREATED    | Gray                               |
| `cycle`    | CYCLE      | none                               |
| `assignee` | ASSIGNEE   | none                               |
| `project`  | PROJECT    | none                               |
| `estimate` | ESTIMATE   | none                               |
| `duedate`  | DUE DATE   | Gray                               |

Canonical order is defined by `ColumnNames` (exported slice).

## Default Columns

`DefaultColumns(issues)` inspects the data dynamically:
- If **any** issue has labels: `id, status, priority, labels, title`
- Otherwise: `id, status, priority, title`

The `--column` flag overrides defaults. It supports two syntaxes:
- **Replacement**: `--column id,status,title` shows exactly those columns.
- **Additive**: `--column +updated` appends to defaults; `+updated:2` inserts at position 2 (1-based).

Mixing additive and replacement in one flag value is an error.

## Fixed-Width Alignment

Both `FormatIssueList` and `FormatFzfLines` compute column widths:

1. Initialize each column width to its header length.
2. Scan all issue values; widen if any value exceeds the header.
3. Pad all columns except the last (last column has no trailing padding).
4. Columns are separated by a two-space gap (`"  "`).

## PadColor

`PadColor(enabled, code, text, width)` applies ANSI color **inside** the text and adds plain-space padding **outside** the color codes. This prevents ANSI escapes from breaking alignment:

```
// Result: "\033[31mUrgent\033[0m    " — color stops before padding
```

## Adding a New Column

1. Add an entry to `_columnRegistry` with `Header`, `Value`, and `Color` functions.
2. Append the name to `ColumnNames` (controls completion order and validation).
3. Optionally add to `_defaultColumnNames` if it should appear by default.
4. Shell completions for `--column` are auto-generated from `ColumnNames`.
