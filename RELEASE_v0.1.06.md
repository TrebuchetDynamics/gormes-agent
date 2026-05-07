# Gormes Agent v0.1.06 (v2026.5.7)

**Release Date:** May 7, 2026
**Date Alias:** `v2026.5.7`
**Since v0.1.05:** ~691 commits on `development` (autonomous loop + operator-driven strategic pivot)

> **The Pivot Release** — Gormes adopts the Hermes-style dual versioning taxonomy (semver `v0.1.06` paired with date alias `v2026.5.7`), captures a 12-month methodology-first strategy in a checked-in `success-plan.md`, and stands up a new Phase 8 in `progress.json` so the autonomous loop can ship reputation-building rows alongside parity rows.

---

## Versioning Taxonomy

This is the first Gormes release to adopt the Hermes-style dual taxonomy:

- **Canonical git tag:** `v0.1.06` (semver — gates with `cmd/gormes/version.go` and the release workflow's tag-vs-source check).
- **Date alias:** `v2026.5.7` (Hermes-style `vYYYY.M.D`, surfaced in this release-notes filename header, in `webpages/landing/src/data/release.json`'s new `date_tag` field, and in the GitHub release title).

A future release will migrate the canonical git tag to the date form once the release workflow learns to extract version from `cmd/gormes/version.go` independently of the tag string. Until then, both labels reference the same merge commit on `main`.

---

## ✨ Highlights

- **Methodology-first repositioning** — `docs/content/building-gormes/strategy/success-plan.md` is now the North Star: *"TrebuchetDynamics is the team that figured out how to autonomously port large Python projects to Go in production. Gormes is the receipt."* The autonomous porting loop, validation-gated commits, and reusable agentic-engineering toolkit are the differentiators; Hermes-parity is supporting evidence.

- **Phase 8 — Reputation & Publication** added to `progress.json` with seven subphases (8.A Publication Infrastructure, 8.B Repository Messaging, 8.C Engineering Writeups, 8.D Sharp v1.0, 8.E Toolkit Extraction, 8.F Cost Discipline & Loop Economics, 8.G Community & External Contributions) and ten gormes-owned builder-ready rows.

- **Landing page rewrite** (`#methodology` section, new hero) — outcome-first headline, autonomous-loop trust play, live measured metrics from `benchmarks.json` (770+ rows, 725 Go files / 180k lines / 4464 tests, ~39.2 MB binary). Five Playwright assertions updated and passing across iPhone SE through iPhone Plus widths.

- **Hermes parity sweep — six builder-ready rows** materialized from upstream v2026.4.30 + v2026.5.7:
  - `5.Q` Provider client lazy-init for TUI cold-start budget (P2)
  - `5.I` Plugin lifecycle hook: `transform_llm_output` (P2)
  - `5.D` Native `video_analyze` tool contract (P2)
  - `5.M` Kanban worker heartbeat, reclaim, and zombie detection (P1)
  - `7.E` Google Chat shared-chassis platform adapter seam (P2)
  - `5.J` Auth-state TOCTOU close + redaction default-on parity (**P0**)

- **`hermes-agent` is now a git submodule** pinned to upstream `NousResearch/hermes-agent` `main` (sha `7e2af0c2e`), replacing the previous gitignored side-clone. The `gormes-hermes-parity` skill now refreshes the submodule to upstream `main` HEAD at the start of every parity sweep, so classification runs against current upstream truth.

- **Opencode + deepseek-v4-pro builder backend** added alongside the codexu path. `scripts/codexu-gormes-builder-cron.sh` and the loop wrapper accept `GORMES_BUILDER_BACKEND={codexu,opencode}` (default `codexu`), with `GORMES_OPENCODE_MODEL` defaulting to `opencode-go/deepseek-v4-pro`. The production cron loop is currently running on opencode.

- **Phase 6.L code executor and dependency resolver** rows marked complete with passing tests (4 code executor + 5 dependency resolver/composer).

---

## 🧠 Strategy & Reputation Surface

### `success-plan.md` — checked-in 12-month strategy
- **North Star:** "TrebuchetDynamics is the team that figured out how to autonomously port large Python projects to Go in production. Gormes is the receipt."
- **Strategic pivot table** — explicit `stop` vs `start` columns swapping row-count optimization for publication cadence.
- **Quarterly roadmap:** Q1 foundation + first writeup, Q2 toolkit OSS extraction, Q3 sharp v1.0 demo, Q4 compound.
- **30-day action sprint** — TD blog scaffold, README rewrite, writeup #1 draft, sharp differentiator decision, `$/iteration` cost telemetry.
- **Operating principles** — publication > commit cadence, opinion > scope, the loop is the product, cost discipline.
- **Reputation metrics scoreboard** replacing row-count vanity metrics.
- **Risk register** — publication paralysis, identity drift, loop spend without learning, upstream pivot, burnout.
- **30-day stop-doing list** — chasing every Hermes release indiscriminately, hand-mirroring docs without rewriting links, treating Goncho parity as a critical path, running the loop without cost discipline.

### Phase 8 builder-ready rows
Each row carries the full builder contract (`contract`, `contract_status`, `slice_size`, `execution_owner`, `trust_class`, `degraded_mode`, `fixture`, `source_refs`, `ready_when`, `not_ready_when`, `blocked_by`, `unblocks`, `acceptance`, `write_scope`, `test_commands` or `no_test_required`, `done_signal`, `note`, `provenance`). All ten Phase 8 rows are `provenance.origin_type=gormes` (owned divergence). Strategy-only rows requiring operator taste (writeup voice, differentiator decision) carry `no_test_required` with a human-checked acceptance contract.

---

## 🌐 Landing Page

The hero now leads with `AI agent runtime in one Go binary.` and follows with a sub-line explicitly naming TrebuchetDynamics' autonomous engineering loop. The new `#methodology` section sits between hero and install, presenting:
- **Live metrics** bound to `benchmarks.json`: validated rows shipped, code base size, binary size — auto-refreshing each landing build.
- **Four pillar cards:** validation-gated commits, `progress.json` as system of record, Hermes is the parity oracle, reusable porting toolkit.
- **Trust posture row** added: "Every autonomous-loop commit passes a validation gate (go test, progress validate, git diff --check) before landing."
- **Top nav restructured:** `How it is built · Install · Trust · Roadmap · GitHub`.

Playwright tests cover 5 assertions on the new section across desktop and 4 mobile viewports (iPhone SE 320×568 through iPhone Plus 430×932).

---

## 🔧 Hermes Parity Rows Materialized

Six rows added with full builder contracts. Each row's `unblocks` field names sibling work explicitly deferred this pass so the next planner pick has an obvious next step.

| Phase | Priority | Source PRs |
|---|---|---|
| 5.Q Provider client lazy-init for TUI cold-start budget | P2 | v0.12.0 #17046 (lead of ~57% cold-start cut family) |
| 5.I Plugin lifecycle hook: `transform_llm_output` | P2 | v0.13.0 #21235 |
| 5.D Native `video_analyze` tool contract | P2 | v0.13.0 #19301 |
| 5.M Kanban worker heartbeat, reclaim, and zombie detection | P1 | v0.13.0 #21183, #21214, #20188, #20410 |
| 7.E Google Chat shared-chassis platform adapter seam | P2 | v0.13.0 #21306, #21331 |
| 5.J Auth-state TOCTOU close + redaction default-on parity | **P0** | v0.13.0 #21193, #21194, #21241; reverts v0.12.0 #16794 |

---

## 🔁 Builder Loop Backend

`scripts/codexu-gormes-builder-cron.sh` and `scripts/codexu-gormes-builder-loop.sh` now support an `opencode/deepseek-v4-pro` backend alongside the existing codexu path:

```sh
GORMES_BUILDER_BACKEND=opencode \
GORMES_OPENCODE_MODEL=opencode-go/deepseek-v4-pro \
  /home/xel/.local/bin/gormes-codexu-builder-loop run
```

The runner pipes the same builder prompt to `opencode run --format json --dangerously-skip-permissions`, persists the JSON event stream to `$LOG_DIR/$RUN_ID.opencode.jsonl`, merges into `$RUN_LOG`, and extracts the last assistant text into `last-message.txt` for status reporting. Loop wrapper's pre-flight `runner_ready()` check is backend-aware. Existing tests (`internal/codexubuilderloopscript_test.go`) remain green; they exercise the loop wrapper with a fake runner so backend swap doesn't disturb them.

---

## 📚 Submodule Conversion

`hermes-agent/` was converted from a gitignored side-clone to a tracked git submodule:

- `.gitmodules` adds the `hermes-agent` submodule pinned to `https://github.com/NousResearch/hermes-agent.git` `main`.
- `.gitignore` no longer hides the path.
- Pre-existing local mods (Karpathy guardrails template added by onboarding tooling) were backed up to `/tmp` and dropped from the clean submodule.
- `gormes-hermes-parity` workflow step 1 now runs `git submodule update --init --remote -- hermes-agent` (fetch+ff-only fallback for legacy clones, no-op if absent) so every parity sweep classifies against current upstream truth, not a stale checkout.

The submodule is currently pinned at sha `7e2af0c2e` (= upstream tag `v2026.5.7-2-g7e2af0c2e`).

---

## 🐛 Notable Fixes

- `progress.json` schema slips in the v0.12.0 + v0.13.0 parity rows corrected: `provenance` now uses the struct shape `{origin_type: "upstream"|"gormes"}`, and `trust_class: "plugin"` was replaced with the valid `system` value.
- 7 new upstream-hermes docs (`developer-guide/image-gen-provider-plugin.md`, `model-provider-plugin.md`, `guides/google-gemini.md`, `local-ollama-setup.md`, `user-guide/features/web-search.md`, `messaging/google_chat.md`, `windows-wsl-quickstart.md`) mirrored to keep `TestMirroredDocsCoverage` green after the submodule HEAD bump.

---

## ⚖️ Versioning Taxonomy Migration Notes

This release establishes the dual-naming convention. Future work (planned, not in this release):

- A new row will migrate the canonical git tag from semver to date form once the release workflow's `Verify version matches tag` step is updated to read the version from `cmd/gormes/version.go` independently of the tag string.
- The `date_tag` field in `release.json` is now the source of truth for the date alias and feeds future landing-page surfacing of the date form.

---

## Validation Evidence

Local gates (run on the `development` branch before opening the merge PR):

```sh
go test ./... -count=1                         # ok (full suite)
go run ./cmd/progress validate                 # validated 8 phases
git diff --check                               # ok
(cd webpages/landing && npm run build)         # 1 page built
(cd webpages/landing && npm run test:e2e)      # 5 passed (homepage + 4 mobile viewports)
```

Remote gates: GitHub Actions `Release Binaries` workflow (validate → build matrix on linux/amd64+arm64, darwin/amd64+arm64, windows/amd64+arm64 → publish with checksums, attestations, and SBOM).

---

**Full changelog (semver line):** [CHANGELOG.md#0.1.06](https://github.com/TrebuchetDynamics/gormes-agent/blob/main/CHANGELOG.md#0106---2026-05-07)
**Compare:** [v0.1.05...v0.1.06](https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.05...v0.1.06)
**Strategy doc:** [success-plan.md](https://github.com/TrebuchetDynamics/gormes-agent/blob/main/webpages/docs/content/building-gormes/strategy/success-plan.md)
