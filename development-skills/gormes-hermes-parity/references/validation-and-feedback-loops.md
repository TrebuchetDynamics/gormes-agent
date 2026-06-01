# Validation And Feedback Loops

Use this before claiming coverage, closing progress rows, or handing a parity
gap to implementation.

## Feedback-Loop Rule

Pick a focused pass/fail signal before fixing or claiming parity. Prefer:

1. focused Go test through the public command/tool/package interface;
2. CLI invocation with temp `GORMES_HOME` and fixture input;
3. gateway/channel transcript fixture;
4. replayed sanitized event/log payload;
5. manual smoke only when the behavior cannot be captured yet.

The loop must exercise the real behavior. A passing test with `[no tests to
run]`, an empty selector, or a stub-only assertion does not count.

## Vertical Slice Rule

One behavior atom becomes one implementation slice:

```text
upstream behavior atom
  -> source-backed progress row
  -> one failing behavioral test
  -> minimal implementation
  -> row evidence update
```

Do not write all tests first and then all implementation. Do not hand broad
umbrella rows to builders when a thinner row can prove the path end to end.

## Validation Sets

For skill-only edits:

```sh
python3 /home/xel/.codex/skills/.system/skill-creator/scripts/quick_validate.py development-skills/gormes-hermes-parity
find -L .agents/skills .claude/skills .codex/skills -maxdepth 2 -name SKILL.md -print | sort
go run ./cmd/progress validate
git diff --check
```

For progress/docs edits:

```sh
go run ./cmd/progress write
go run ./cmd/progress validate
go test ./internal/progress -count=1
go test ./webpages/docs -count=1
git diff --check
```

For upstream coverage claims:

```sh
go test ./webpages/docs -run TestUpstreamCoverageLedgerMatchesSourceClasses -count=1
```

For runtime identifiers, commands, tools, config, persistence, or public APIs,
run the focused package tests and then `go test ./... -count=1` when feasible.

## Report Shape

```text
scope:
source_shas:
upstream_refs:
gormes_refs:
evidence_boundary:
parity_definition:
classification_summary:
taxonomy_changes:
rows_changed:
compatibility_notes:
delegated_task_packets:
validation:
next_builder_rows:
blockers:
```

Do not claim full parity unless the feature map, coverage ledger, progress
rows, and validation all support it.
