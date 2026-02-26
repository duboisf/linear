# Architecture

High-level design patterns and structural decisions.

## Key Rules

- Every command receives `opts Options` — no globals, no singletons.
- Parent commands (`issue`, `user`, `cache`) have no `RunE`, only `AddCommand`.
- Set `ValidArgsFunction` returning `ShellCompDirectiveNoFileComp` on every command.
- Hidden commands (e.g., `edit-interactive`) are internal — used by fzf bindings, not users.

## Contents

- [Dependency Injection](dependency-injection.md) — Options struct pattern enabling testability.
- [Command Structure](command-structure.md) — Cobra command hierarchy and conventions.
