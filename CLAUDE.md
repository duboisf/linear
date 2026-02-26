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

## Architecture

Options-based dependency injection, Cobra command hierarchy.

- [Architecture](docs/architecture/README.md) — DI pattern, command structure.

## API

genqlient for typed GraphQL, file-based cache with atomic writes and smart invalidation.

- [API](docs/api/README.md) — GraphQL client, caching strategies.

## Authentication

Provider chain: env var → native keyring → file fallback → interactive prompt.

- [Authentication](docs/auth/README.md) — credential resolution and storage.

## Interactive Mode

fzf launched immediately, data fetched in background, preview pre-cached.

- [Interactive Mode](docs/interactive/README.md) — fzf integration, keybindings, prefetch.

## Formatting

Column registry for tabular output, NO_COLOR support, glamour for markdown.

- [Formatting](docs/formatting/README.md) — columns, colors, output formats.

## Development

Every command must have shell completions and tests. Tests run with `-race`.

- [Development](docs/development/README.md) — shell completions, testing, code generation, adding commands.

## Configuration

All runtime behavior controlled via environment variables.

- [Configuration](docs/configuration/README.md) — environment variables.
