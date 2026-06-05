---
name: gormes-readme
description: Use when the user asks to audit, score, refresh, fact-check, or periodically improve the Gormes README, public repository messaging, usage docs, install docs, status claims, positioning, or repository-quality evidence.
---

# Gormes README

## Repository Branch Rule

For Gormes work, stay on the existing `development` branch. Do not create or
use feature branches, short-lived branches, or git worktrees. If the checkout
is not on `development`, stop before editing and switch safely or report the
blocker.

## Mission

Keep `README.md` accurate as Gormes moves toward Hermes-in-Go parity. This
skill turns current repo evidence into bounded README improvements. It is not a
runtime implementation path and must not create a parallel backlog.

## Periodic Usage
Use this skill after meaningful changes to install flow, command usage,
supported channels, Goncho/memory behavior, binary size, public docs links, or
progress rollups. Also use it before releases or public demos, monthly while
Gormes is moving quickly, and quarterly when the repo is quiet.

## Workflow

### 1. Bound The Pass
State the specific README surface being checked:
- quick start, install commands, and diagnostics;
- build/runtime status, feature claims, channels, and Goncho/Honcho wording;
- binary size or benchmark-derived claims;
- docs, website, architecture-plan, license, release, and contribution links;
- 100/100 repo signals: clarity, DX, docs, trust, performance, and polish.
If the request implies a feature is shipped but repo evidence is missing, keep
the README conservative and route implementation or roadmap work through
`gormes-skill-manager`.

### 2. Gather Evidence
Start with local truth:
```sh
git status --short
sed -n '1,220p' README.md
go run ./cmd/progress validate
rg -n "gormes doctor|install|offline|gateway|Goncho|Honcho|Hermes-compatible|single static binary|supported|deferred" README.md cmd internal webpages/docs webpages/landing
```
Use exact source files, tests, generated progress docs, and the parity evidence doc as
evidence. When checking external project, SDK, CLI, or cloud-service facts, use
Context7 or official sources as required by repo instructions. Do not rely on
memory for time-sensitive claims.

### 3. Apply The Quality Lens
When the user asks for a score, audit, 100/100 repo pass, or broad README
improvement, read `references/readme-quality-rubric.md`. Prioritize changes
that help a stranger understand Gormes quickly, run it without pain, trust its
claims, and find docs, tests, release status, license, and contribution paths.

### 4. Improve README.md
Edit only the README sections needed for the pass. Prefer:
- executable commands over broad descriptions;
- current limitations over implied readiness;
- links to deeper docs instead of duplicating architecture detail;
- sober operator-facing language over launch copy;
- generated markers left intact.
Do not hand-edit content between progress markers:
```text
<!-- PROGRESS:START kind=readme-rollup -->
<!-- PROGRESS:END -->
```
Regenerate that block instead.

### 5. Regenerate Derived Data
Use existing repo tools rather than new scripts:
```sh
go run ./cmd/progress write
go run ./cmd/progress validate
```
If binary-size text changed because a fresh built binary exists, run:
```sh
bash scripts/record-benchmark.sh
bash scripts/update-readme.sh
```
Skip benchmark updates when no trusted local binary exists; do not invent sizes.

### 6. Validate
Run focused validation for touched surfaces:
```sh
go test ./webpages/docs -count=1
go test ./internal/planning/progress -count=1
go test ./internal/repoctl -count=1
```
Broaden tests when README changes depend on runtime behavior, installer
behavior, or website data.

## Final Report
Report:
1. README surfaces checked
2. Evidence used
3. 100/100 rubric gaps found, if requested
4. Changes made
5. Generated blocks or benchmark data refreshed
6. Validation run
7. Claims intentionally left conservative
8. Follow-up rows or skills needed, if any
