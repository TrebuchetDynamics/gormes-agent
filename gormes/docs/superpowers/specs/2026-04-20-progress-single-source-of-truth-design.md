---
title: "Progress — Single Source of Truth"
date: 2026-04-20
status: approved
---

# Progress — Single Source of Truth

## Problem

`gormes/docs/content/building-gormes/architecture_plan/progress.json` is nominally
the source of truth for Gormes roadmap progress, but every downstream consumer is
hand-maintained and already drifting:

- `progress.json` stats block: **10 complete, 2 in progress, 40 planned** (52 subphases)
- `_index.md` top line: **"8/52 subphases shipped (15%)"** — stale
- Landing page (`gormes/www.gormes.ai/internal/site/content.go` line 157):
  **"8/52 shipped"** — stale
- Landing page Phase 2 label: **"IN PROGRESS · 3/7"** — but JSON has 8 subphases
- Landing page Phase 3 label: **"IN PROGRESS · 4/5"** — but JSON has 12 subphases
- README has a coarse phase table, also hand-written

The goal is one place to edit, with every public surface derived automatically.

## Approved choices

| Question | Choice |
|---|---|
| Granularity | Both — subphase rollup on README/landing, full item checklist in docs |
| Item schema | Explicit objects: `{name, status, ...}` |
| Generator architecture | `go:embed` for landing page + marker-based region replacement for README/docs |
| Update workflow | Hand-edit `progress.json`. No CLI yet. |

## Architecture

One source, many sinks:

```
                progress.json  (hand-edited, validated)
                      │
       ┌──────────────┼──────────────┐
       │              │              │
    go:embed       marker          marker
       │            replace        replace
       ▼              ▼              ▼
 landing page    README.md       _index.md
 (content.go)   (top surface)  (full checklist)
```

**Bubble-up rule.** Subphase `status` is derived from its items when items exist:

- all items `complete` → `complete`
- any item `complete` or `in_progress` → `in_progress`
- otherwise → `planned`

Phase status derives from its subphases the same way. Explicit `status`
overrides are allowed **at subphase level only**, for subphases that have no
item breakdown. Phase status is always derived — there is no phase-level override.

**No stored stats.** The top-level `stats` block in v1 is removed. Counts and
percentages are computed by generators, eliminating the drift class entirely.

## Schema — `progress.json` v2

```jsonc
{
  "meta": {
    "version": "2.0",
    "last_updated": "2026-04-20",
    "links": {
      "github_readme": "...",
      "landing_page": "...",
      "docs_site": "...",
      "source_code": "..."
    }
  },
  "phases": {
    "2": {
      "name": "Phase 2 — The Gateway",
      "deliverable": "Go-native tools + Telegram + session resume + wider adapters",
      "subphases": {
        "2.D": {
          "name": "Cron / Scheduled Automations",
          "priority": "P2",
          "items": [
            { "name": "Go ticker + bbolt job store", "status": "complete" },
            { "name": "Natural-language cron parsing", "status": "planned" }
          ]
        }
      }
    }
  }
}
```

**Allowed `status` values:** `complete`, `in_progress`, `planned`. Validator rejects anything else.

**Reserved optional item fields** (not rendered yet, schema allows them):
`pr` (URL), `owner`, `eta` (ISO date), `note`.

**Explicit override form** — for units with no item breakdown:

```jsonc
"4.A": {
  "name": "Provider Adapters",
  "status": "planned"
}
```

## Generators

New Go package at `gormes/internal/progress/`:

```
gormes/
  internal/progress/
    progress.go       // types + Load() + Derive() + Stats() + Validate()
    progress_test.go  // table-driven: bubble-up, stats, validation
    render.go         // RenderReadmeRollup(), RenderDocsChecklist()
  cmd/progress-gen/
    main.go           // CLI for the Makefile target
  www.gormes.ai/internal/site/
    content.go        // uses progress.Load() for RoadmapPhases
    progress.go       // status → tone/label mapping (presentation only)
```

Both the landing page and the CLI generator depend on the shared `internal/progress`
package. One parser, one set of tests.

### Marker convention

README.md and `_index.md` carry generated blocks between markers:

```markdown
<!-- PROGRESS:START kind=readme-rollup -->
...generated content...
<!-- PROGRESS:END -->
```

`kind` selects the renderer:

- `readme-rollup` — 6-row phase table with derived icon and `X/Y shipped` counts.
  Lands in the existing README `## Architecture` section.
- `docs-full-checklist` — for `_index.md`: top stats line, per-phase table,
  per-subphase item checkbox list (`- [x]` / `- [ ]`).

Generator fails loudly if markers are missing, unbalanced, or contain an unknown
`kind`.

### Landing page

`content.go` drops its hardcoded `[]RoadmapPhase` slice. Instead it calls
`progress.Load()` at page-build time and maps status to presentation via a small
table in `www.gormes.ai/internal/site/progress.go`:

| Status | Tone | Status label format |
|---|---|---|
| `complete` | `shipped` | `SHIPPED · EVOLVING` (if ongoing) or `SHIPPED · <n>/<total>` |
| `in_progress` | `progress` | `IN PROGRESS · <n>/<total>` |
| `planned` | `planned` or `later` | `PLANNED · 0/<total>` |

Tone choice for planned phases (`planned` vs `later`) is decided per-phase in the
mapping table — e.g. Phase 5 stays `later` because it's post-Phase-4. This
mapping table is the only place presentation lives; it's hand-maintained.

## Build wiring

`gormes/Makefile`:

```make
build: $(BINARY_PATH)
	@$(call validate-progress)
	@$(call record-benchmark)
	@$(call record-progress)        # existing — touches last_updated
	@$(call generate-progress)      # NEW — regenerates README + _index.md blocks
	@$(call update-readme)
```

Both `validate-progress` and `generate-progress` call `go run ./cmd/progress-gen`
with flags (`-validate` / `-write`). Validation failure stops the build.

Landing page picks up schema changes automatically via `go:embed` on every
`go build` — no extra step.

## Validation rules

Validator (in `internal/progress.Validate()`):

1. Every item and every explicit override `status` is in `{complete, in_progress, planned}`.
2. Subphase has either `items` or an explicit `status`, never neither, never both.
3. Marker pairs in target files are balanced and enclose a single `kind=<name>`.
4. No top-level `stats` block (catches stale v1 files in PRs).
5. `meta.version` is `"2.0"`.

## Migration (one-shot)

Applied in the implementation PR:

1. Bump `meta.version` to `"2.0"`.
2. Convert every `"items": [<string>...]` to `[{"name": <string>, "status": ...}]`.
   Initial per-item statuses:
   - Subphase was `complete` → all items `complete`.
   - Subphase was `in_progress` → items start `planned`; operator marks the real
     ones `complete` in the same PR.
   - Subphase was `planned` → items `planned`.
3. Remove per-subphase/phase `status` fields where `items` exist. Keep the
   explicit form only where the subphase has no item breakdown (e.g. Phase 4's
   broad provider-adapter bucket — migration keeps it explicit for now).
4. Delete the top-level `stats` block.
5. Add markers to README.md (new block inside `## Architecture`) and
   `_index.md` (full checklist section). Generator populates them on first run.
6. Rewire `content.go` to use `internal/progress` + the tone map.
7. Drifted counts (`8/52`, `3/7`, `4/5`) disappear — computed on render.

## Out of scope

- **No CLI** to mark items complete. Hand-edit is fine for now; add later if
  friction proves real.
- **Phase narrative files** (`phase-1-dashboard.md` through `phase-6-*.md`) stay
  untouched. They're prose, not tracking.
- **No ETAs, owners, or PR links rendered yet.** Schema reserves the fields;
  generators ignore them until we design their presentation.

## Risk / open items

- **Cascading status overrides** when both an item list AND an explicit
  subphase status exist: validator rejects this (rule 2) to prevent ambiguity.
- **Hugo front-matter interaction:** `_index.md` front-matter must live OUTSIDE
  the PROGRESS markers so Hugo parses it correctly.
- **Test matrix** for the bubble-up function must cover: empty items,
  all-planned, all-complete, mixed, explicit override present, nested phase
  derivation.
