# Agent Template Reset Design

**Status:** Accepted
**Author:** Codex
**Date:** 2026-04-30

## Context

Hermes seeds a `SOUL.md` persona file into `HERMES_HOME` on install and
falls back to a built-in default identity when the file is absent. Gormes
already loads `SOUL.md`, project context files, and durable `USER.md` /
`MEMORY.md` files during live turns, but it has no first-party command that
creates the default Gormes development context files.

The active development workspace is
`/home/xel/git/sages-openclaw/workspace-gormes`, not the older
workspace-mineru Gormes checkout. The command must accept an explicit target
so local development environments can be reset without hard-coding that path
into product code.

## Decision

Add `gormes agent reset` as a Gormes-owned command. It creates a conservative
template set:

- `SOUL.md` with a Gormes persona adapted from Hermes' default direct,
  useful, efficient assistant identity.
- `AGENTS.md` with workspace-level Gormes development guidance.
- `IDENTITY.md` and `TOOLS.md` as optional context files for operators and
  future agent loaders.
- `memory/USER.md` and `memory/MEMORY.md`, matching the existing live-turn
  durable context lookup.

The default mode is non-destructive: create missing files, skip existing
files. `--force` overwrites existing files. `--dry-run` reports the same
actions without writing. `--target <dir>` chooses the root, defaulting to
`config.GormesHome()`.

## Non-Goals

- Change live-turn prompt assembly. The current loader already discovers
  `SOUL.md` and `memory/USER.md` / `memory/MEMORY.md`.
- Modify real user state during tests.
- Import or copy Hermes text verbatim beyond the behavioral defaults needed
  for parity.
- Encode a machine-specific workspace path in the command.

## Verification

- Unit tests cover template inventory, create, skip, force, and dry-run.
- CLI tests cover `gormes agent reset --target`, dry-run output, and root
  command registration.
- `cmd/progress validate` keeps the roadmap row schema-valid.
