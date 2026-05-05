# Parity Audit Output

Use this structure for parity reports.

## Subsystem

Name the audited surface and why it matters for Hermes-in-Go completion.

## Upstream Behavior

List exact upstream references:

- repo path;
- type/function/command/schema names;
- behavior observed;
- relevant tests or fixtures if present.

## Gormes Evidence

List exact Gormes references:

- package/command paths;
- public interfaces;
- tests/fixtures;
- progress rows.

## Classification

Use one of:

- `covered`
- `planned`
- `vague`
- `missing`
- `owned`

Explain the evidence in one sentence.

## Progress Row Candidate

For `vague` or `missing` items, provide:

- name;
- contract;
- write_scope;
- source_refs;
- test_commands;
- acceptance;
- ready_when;
- not_ready_when;
- done_signal;
- blocked_by/unblocks if needed.
