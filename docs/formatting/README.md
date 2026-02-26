# Formatting

Output formatting, column system, and color handling.

## Key Rules

- Check `ColorEnabled()` before emitting ANSI — respects `NO_COLOR` env var.
- Use `PadColor` (not `Colorize`) when alignment matters — keeps padding outside ANSI codes.
- New columns: add to `_columnRegistry` and append to `ColumnNames`.

## Contents

- [Columns](columns.md) — column registry, alignment, and how to add new columns.
- [Colors](colors.md) — color detection, state/priority palettes, and glamour theming.
