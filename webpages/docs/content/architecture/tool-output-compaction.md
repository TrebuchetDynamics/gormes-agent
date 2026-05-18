---
title: "Tool output compaction"
description: "Deterministic reduction of large model-visible terminal tool output."
weight: 45
---

# Tool output compaction

Tool output compaction is a model-context optimization for terminal-heavy tool
results. It reduces large stdout/stderr payloads before they become prompt
debt, while preserving status, exit codes, and repair-critical diagnostics.

This is separate from both:

- context compression, which prunes older conversation history; and
- TOON context encoding, which serializes structured JSON-shaped prompt context.

## Pilot boundary

The first Gormes slice is intentionally narrow:

- reducer code lives in `internal/toolcompact`
- `execute_code` and `terminal` stdout/stderr are wired
- compaction is enabled for the built-in registry through internal tool
  configuration
- `full_output: true` bypasses compaction for exact output
- provider HTTP bodies, public APIs, persisted storage, and config formats stay
  JSON and unchanged

The model-visible tool result keeps the existing result shape and adds
structured `compaction` evidence only when a stream is reduced. Evidence records
the reducer, original byte count, compacted byte count, and stable reason codes.

## Reducers

The first reducers are compiled Go code rather than user/project rule files:

- `go_test` counts passing package walls and keeps failing tests, package
  failures, panic lines, and `file.go:line` diagnostics.
- `build_diagnostics` keeps compiler/build package headers, `file:line`
  diagnostics, failures, panic lines, and error lines while dropping repeated
  dependency or cache noise.
- `git_status` summarizes branch state and file status counts while keeping
  conflicted files and a bounded sample list.
- `head_tail` keeps bounded leading and trailing context for unknown oversized
  text output.

This keeps the pilot deterministic and testable. A future rule layer should only
be added after reducer knobs, evidence codes, and safety boundaries stabilize.

## Raw output

Compaction is not a substitute for exact bytes. Callers that need exact command
output must request `full_output: true` before execution. Raw artifact storage
is deferred for these terminal-style tools until a session-scoped output
directory is available at the call site.
