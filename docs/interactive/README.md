# Interactive Mode

fzf-based interactive browsing and editing.

## Key Rules

- Launch fzf immediately, fetch data in background via `io.Pipe`.
- Call `glamourStyle()` once before launching fzf or goroutines — never concurrently.
- Pass `LINEAR_GLAMOUR_STYLE` to fzf subprocesses to avoid repeated OSC 11 queries.
- Prefetch with semaphore of 5 — skip already-cached issues.

## Contents

- [fzf Integration](fzf-integration.md) — concurrent fetching, preview cache, keybindings, and prefetch.
