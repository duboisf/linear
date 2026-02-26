# fzf Integration

Interactive issue browsing is built on fzf, with concurrent data fetching and cached previews.

## Core Design Pattern

`fzfBrowseIssues` (in `cmd/pick.go`) launches fzf **immediately** using an `io.Pipe`. A background goroutine fetches issues from the API and writes formatted lines to the pipe's writer. fzf displays its built-in loading indicator until data arrives on stdin.

```
io.Pipe() --> pw (goroutine writes issues) --> pr (fzf reads stdin)
```

A `context.WithCancel` wraps the fetch. If the user exits fzf early (ESC/Ctrl-C), `fetchCancel()` fires, aborting the in-flight API call. The goroutine's error is collected via a `chan error` and surfaced over fzf's own exit code when relevant.

## Preview Rendering and Cache

Each issue preview is pre-rendered to ANSI text via glamour (markdown) and stored in the file-based cache at `<cache-dir>/issues/<IDENTIFIER>`. The fzf `--preview` command is just `cat <cache-file>` with a polling fallback (up to 5s) for the race where the user navigates before prefetch completes.

`renderMarkdown` forces `termenv.TrueColor` so ANSI codes are written regardless of TTY detection -- the output goes to a file, not a terminal.

## Issue Prefetching

`prefetchIssueDetails` runs in a separate goroutine (launched from the fetch goroutine, after issue list arrives). It fetches full issue details **in parallel** with a semaphore of 5 (`sem := make(chan struct{}, 5)`). Already-cached issues are skipped. This populates previews before the user navigates to them.

## Terminal Background Detection (OSC 11)

`glamourStyle()` calls `termenv.HasDarkBackground()`, which sends an OSC 11 terminal query. This is:

1. **Called once synchronously** before launching fzf or goroutines, to avoid concurrent queries that leak escape sequences as garbage.
2. **Cached** in `_glamourStyleOnce` (`sync.Once`) so subsequent calls reuse the result.
3. **Overridable** via `LINEAR_GLAMOUR_STYLE` env var (values: `dark` or `light`). The parent process sets this in fzf's subprocess environment (`cmd.Env`) so that child commands spawned by fzf bindings skip the OSC 11 query entirely.

## fzf Keybindings

### ctrl-y: Set Current Cycle

Runs `linear issue edit {1} --cycle current` via `transform-header`. The result message replaces the fzf header. If a reload command is configured, fzf also calls `reload()` to refresh the issue list, plus `refresh-preview` to update the detail pane.

### ctrl-e: Interactive Edit

Runs `linear issue edit-interactive {1}` via `execute()`, which hands the terminal to the subprocess. This enables **nested fzf pickers**: the user picks a field (Status, Priority, Cycle, Assignee, Project, Labels-Add, Labels-Remove, Title, Description), then picks or edits the value. After the edit, fzf reloads the list and refreshes the preview.

### Scroll bindings

- `ctrl-d` / `ctrl-u`: half-page scroll in preview
- `shift-down` / `shift-up`: line-by-line scroll in preview

## edit-interactive (Hidden Command)

`newIssueEditInteractiveCmd` (in `cmd/issue_edit_interactive.go`) is a **hidden** cobra command (`Hidden: true`). It exists solely as the target of fzf's ctrl-e binding. Flow:

1. Fetch full issue via API.
2. `fzfPickField` -- presents editable fields with current values.
3. `applyFieldEdit` -- dispatches to field-specific editor (fzf picker, `$EDITOR`, or multi-select).
4. `refreshIssueCache` -- re-fetches and updates the cached preview.

## Reload Mechanism

`buildFzfReloadCmd` constructs a shell command (`linear issue list --fzf-data ...`) that fzf calls via `reload()` after edits. The `--fzf-data` hidden flag outputs header + data lines in fzf's expected format, preserving all active filters (cycle, status, label, user, sort, column, limit).

## Key Files

| File | Purpose |
|------|---------|
| `cmd/pick.go` | `fzfBrowseIssues`, prefetch, preview cache, glamour rendering |
| `cmd/issue_edit_interactive.go` | Hidden edit command, field/value pickers |
| `cmd/issue_list.go` | `--interactive` flag, reload command builder |
