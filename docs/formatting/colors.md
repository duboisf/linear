# Colors

ANSI color handling for terminal output. Source: `internal/format/color.go`.

## Color Detection

`ColorEnabled(w io.Writer) bool` decides whether to emit ANSI escapes:

1. If `NO_COLOR` env var is set (any value), return `false`. Follows [no-color.org](https://no-color.org).
2. Type-assert `w` to the `fdWriter` interface (`Fd() uintptr`), e.g. `*os.File`.
3. Call `term.IsTerminal(fd)` from `golang.org/x/term`. Non-file writers return `false`.

## ANSI Constants

```go
Reset  = "\033[0m"
Bold   = "\033[1m"
Red    = "\033[31m"
Green  = "\033[32m"
Yellow = "\033[33m"
Cyan   = "\033[36m"
Gray   = "\033[90m"
```

## State Colors

`StateColor(stateType string)` maps Linear workflow state types:

| State type  | Color  |
|-------------|--------|
| `started`   | Yellow |
| `completed` | Green  |
| `canceled`  | Red    |
| `backlog`   | Gray   |
| `unstarted` | (none) |
| `triage`    | (none) |

## Priority Colors

`PriorityColor(p float64)` maps priority rank:

| Priority | Label    | Color  |
|----------|----------|--------|
| 1        | Urgent   | Red    |
| 2        | High     | Yellow |
| 3        | Normal   | Green  |
| 4        | Low      | Gray   |
| 0        | None     | (none) |

Priority 0 (None) sorts last — `issuePriorityRank()` in `cmd/issue_list.go` maps it to 99.

## Colorize and PadColor

- `Colorize(enabled, code, text)` — wraps text in `code + text + Reset` when enabled.
- `PadColor(enabled, code, text, width)` — colorizes text, then appends plain spaces to reach `width`. Padding stays outside ANSI escapes so column alignment is preserved.

## Glamour Markdown Rendering

Used in fzf preview for issue descriptions (`cmd/pick.go`):

- `glamourStyle()` detects dark/light terminal background:
  1. Check `LINEAR_GLAMOUR_STYLE` env var (set by parent process for subprocesses).
  2. Fall back to `termenv.HasDarkBackground()`, which sends an OSC 11 query to the terminal.
  3. Result is cached per-process via `sync.Once`.
- `renderMarkdown(markdown)` creates a glamour renderer with:
  - TrueColor profile (forced regardless of TTY detection)
  - Standard dark or light style
  - 80-column word wrap
