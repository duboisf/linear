# Linear CLI - Project Rules

## Documentation

This file is a table of contents. Detailed rules live under `docs/`.

### Maintenance Rules

- CLAUDE.md links to each `docs/` subdirectory's `README.md`.
- Each `docs/` subdirectory has a `README.md` with a brief description and links to all files in it.
- Every file under `docs/` **must** be reachable from CLAUDE.md via markdown links.
- When adding a new doc, add a link to its directory's `README.md`.
- When adding a new `docs/` subdirectory, create a `README.md` in it and link it from CLAUDE.md.
- When removing a doc file, remove its link too.
- Keep individual doc files focused on one topic. If a file covers multiple distinct topics or exceeds 100 lines, split it.
- Run `make lint-docs` to validate all links and structure.

## Development

Every command must have shell completions and tests. Tests run with `-race`.

- [Development](docs/development/README.md) — shell completions, testing conventions.
