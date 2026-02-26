# Caching

## File-Based Cache

Implemented in `internal/cache/cache.go`. The cache stores serialized data as plain files on disk.

### Location

- Primary: `$XDG_CACHE_HOME/linear/` (via `os.UserCacheDir()`)
- Fallback: `$TMPDIR/linear-cache` (set in `cmd/root.go` `DefaultOptions`)

### Key Structure

Keys are nested file paths under the cache directory:

- `issues/AIS-42` -- cached preview for a single issue
- `cycles/list` -- cached cycle list response

### Default TTL

- **5 minutes** -- set in `DefaultOptions` via `cache.New(cacheDir, 5*time.Minute)`
- Per-key override available via `GetWithTTL(key, ttl)` (e.g., cycles use 24h TTL)
- Expiry is based on file mtime, checked at read time

## Atomic Writes

`Set()` writes to a temp file in the same directory, then renames it to the final path. This guarantees concurrent readers never see partial content:

```go
tmp, _ := os.CreateTemp(dir, ".tmp-*")
tmp.WriteString(content)
tmp.Close()
os.Rename(tmp.Name(), path)
```

## Cache Invalidation Patterns

### Global: `--refresh` / `-r` Flag

Defined in `cmd/root.go` as a persistent flag. Clears the entire cache directory before running any subcommand:

```go
root.PersistentFlags().BoolVarP(&refresh, "refresh", "r", false, "Clear cached data before running")
```

### Selective: After Edits

`cmd/issue_edit.go` calls `refreshIssueCache()` after a successful update. This re-fetches the single issue from the API and overwrites its cache entry, so interactive fzf previews show fresh data immediately:

```go
refreshIssueCache(cmd.Context(), client, opts.Cache, identifier)
```

The same pattern is used in `cmd/issue_edit_interactive.go` after multi-field edits.

### Cycle Boundary Detection

Cycle data is cached with a 24h TTL (`_cycleCacheTTL`), but the boolean flags (`IsActive`, `IsNext`, etc.) are point-in-time snapshots. `cycleBoundaryCrossed()` in `cmd/issue_list.go` checks whether `now > endsAt` for the active cycle. If so, the cache is treated as stale regardless of TTL:

```go
func cycleBoundaryCrossed(resp *api.ListCyclesResponse, now time.Time) bool {
    // Find active cycle, check if now > endsAt
}
```

## Issue Prefetching

In interactive mode (`cmd/pick.go`), after the issue list is fetched and piped to fzf, a background goroutine prefetches full details for every issue in parallel:

```go
func prefetchIssueDetails(ctx context.Context, client graphql.Client, c *cache.Cache, identifiers []string) {
    sem := make(chan struct{}, 5) // max 5 concurrent API calls
    for _, id := range identifiers {
        if _, ok := c.Get("issues/" + id); ok {
            continue // skip already-cached
        }
        // fetch and cache in goroutine
    }
}
```

- **Concurrency limit**: semaphore of 5 prevents API rate limiting
- **Skip cached**: already-cached issues are not re-fetched
- **Non-blocking**: runs in a goroutine so fzf is interactive immediately
- **Preview fallback**: fzf preview command polls for the cache file (50 retries, 100ms apart) to handle the race where a user previews before prefetch completes

## Key Files

| File | Purpose |
|---|---|
| `internal/cache/cache.go` | Cache implementation (Get, Set, Clear, Delete) |
| `cmd/root.go` | `--refresh` flag, cache directory setup |
| `cmd/issue_list.go` | Cycle boundary detection, cycle caching |
| `cmd/pick.go` | Prefetching, preview cache, `refreshIssueCache` |
| `cmd/issue_edit.go` | Selective invalidation after edits |
