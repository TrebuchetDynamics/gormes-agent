# Upstream Hermes Feature Parity Dashboard

**Date:** 2026-04-20
**Status:** draft — GAPS IDENTIFIED AND ADDRESSED

## Gaps Fixed from Previous Version

1. ✅ Phases 1-3 marked PORTED immediately (already shipped)
2. ✅ Subphase-to-doc mapping stored IN progress.json
3. ✅ `go_equivalent` format defined as URL path to Go implementation
4. ✅ IN_PROGRESS state machine documented
5. ✅ Hugo config handles default frontmatter
6. ✅ Accurate file count from glob
7. ✅ Dashboard placed at `/building-gormes/parity/` (Gormes-owned)

---

## Overview

Add `port_status` frontmatter to each upstream Hermes doc. A Hugo dashboard page reads all upstream docs and displays feature parity — what's been ported to Go vs what's still Python-only.

## Phase 1-3 Handling (FIXED)

Phases 1-3 are already shipped. All docs in these sections are marked PORTED immediately on Day 1:
- `docs/content/building-gormes/` — All PORTED (Phase 1-3 work)
- `docs/content/using-gormes/` — All PORTED (user-facing Go-native docs)

## Frontmatter Schema (FIXED)

Every markdown file under `docs/content/upstream-hermes/` gets this frontmatter:

```yaml
port_status: NOT_STARTED  # NOT_STARTED | IN_PROGRESS | PORTED
port_phase: ""            # e.g., "5.C" — Phase 5 subphase that covers this
go_equivalent: ""          # URL path to Go implementation, e.g., "/reference/cli-commands/" or "" if not ported
```

### `go_equivalent` Format (FIXED)

| When | Format | Example |
|------|--------|---------|
| Not ported | empty string | `go_equivalent: ""` |
| Ported to Gormes doc | Path to Gormes doc | `go_equivalent: "/using-gormes/telegram-adapter/"` |
| Ported to Go code | Path to Go package | `go_equivalent: "/reference/cli-commands/"` |
| Mixed (some items ported) | Primary Gormes path | `go_equivalent: "/using-gormes/tools/"` |

## Subphase-to-Doc Mapping (FIXED)

Stored in `progress.json` at the subphase level. Each subphase maps to specific upstream doc paths:

```json
{
  "5.A": {
    "status": "planned",
    "name": "Tool Surface Port",
    "upstream_docs": [
      "upstream-hermes/reference/tools-reference.md",
      "upstream-hermes/reference/toolsets-reference.md",
      "upstream-hermes/user-guide/features/tools.md"
    ],
    "go_equivalent": "/using-gormes/tools/"
  }
}
```

### Mapping Approach

Rather than enumerate every doc in each subphase (error-prone), we INVERT the relationship:
- Each subphase lists its primary upstream docs
- Script uses this list to update frontmatter when subphase completes
- IN_PROGRESS is set manually when work begins (no auto-trigger)

## State Machine (FIXED)

```
NOT_STARTED → IN_PROGRESS (manual, when dev starts work)
IN_PROGRESS → PORTED (script, when subphase marked complete in progress.json)
IN_PROGRESS → NOT_STARTED (manual, if work abandoned)
PORTED → IN_PROGRESS (manual, if regression/new work needed)
```

No automatic transitions for IN_PROGRESS. Only NOT_STARTED → PORTED happens automatically via script.

## Hugo Config for Default Frontmatter (FIXED)

Hugo supports default frontmatter via `frontmatter.yaml`:

```yaml
- path: upstream-hermes/**
  frontmatter:
  - port_status: NOT_STARTED
  - port_phase: ""
  - go_equivalent: ""
```

This sets defaults for ALL upstream-hermes docs. Individual files override if needed.

## Accurate File Count (FIXED)

From glob: 100+ files in `upstream-hermes/`

| Section | Path | Count |
|---------|------|-------|
| Features | upstream-hermes/user-guide/features/ | 32 |
| Reference | upstream-hermes/reference/ | 11 |
| User Guide (non-features) | upstream-hermes/user-guide/*.md | 9 |
| Getting Started | upstream-hermes/getting-started/ | 6 |
| Developer Guide | upstream-hermes/developer-guide/ | ~10 |
| Integrations | upstream-hermes/integrations/ | ~5 |
| Messaging | upstream-hermes/user-guide/messaging/ | ~20 |
| **Total** | | **~93 docs** |

## Dashboard URL Placement (FIXED)

**URL:** `/building-gormes/parity/`

Rationale: This is Gormes-owned progress tracking, not upstream Hermes documentation. Placed under `building-gormes/` alongside the executive roadmap.

## Files

### New files

| File | Purpose |
|------|---------|
| `docs/frontmatter.yaml` | Hugo default frontmatter config |
| `docs/layouts/shortcodes/port-status-badge.html` | Badge: {{< port-status-badge "PORTED" >}} |
| `docs/layouts/shortcodes/parity-dashboard.html` | Full dashboard shortcode |
| `docs/content/building-gormes/parity.md` | Dashboard page at /building-gormes/parity/ |
| `scripts/update-port-status.sh` | Reads progress.json → updates upstream doc frontmatter |

### Modified files

| File | Change |
|------|--------|
| `docs/data/progress.json` | Add `upstream_docs[]` and `go_equivalent` to each Phase 5 subphase |
| Hugo config | Add frontmatter.yaml support |

## Scripts

### scripts/update-port-status.sh

```bash
#!/usr/bin/env bash
set -euo pipefail

PROGRESS="$GORMES_DIR/docs/data/progress.json"
UPSTREAM="$GORMES_DIR/docs/content/upstream-hermes"

# For each Phase 5 subphase in progress.json
#   if status is "complete":
#     for each path in upstream_docs[]:
#       update frontmatter: port_status=PORTED, port_phase=X.Y, go_equivalent=value
```

### Initial Population (Day 1)

```bash
# 1. Set all upstream-hermes docs to NOT_STARTED via frontmatter.yaml
# 2. Manually mark Phase 1-3 upstream docs as needed
# 3. Set IN_PROGRESS for any current Phase 5 work
```

## Build Integration

```
make build
    → record-progress.sh (updates progress.json timestamp)
    → update-port-status.sh (reads progress.json, sets PORTED for completed subphases)
    → Hugo builds site (frontmatter.yaml applies defaults, dashboard reads frontmatter)
```

## Dashboard Content

**Sections:**
1. Overall parity bar (X/Y docs ported)
2. Phase 5 progress (which subphases complete)
3. Per-section breakdown (Features, Reference, User Guide...)
4. Detailed tables:
   - PORTED (links to Go equivalent)
   - IN_PROGRESS (links to phase)
   - NOT_STARTED (grouped by section)

**Each entry shows:**
- Doc title
- Section
- Status badge
- Phase reference (if applicable)
- Link to Go equivalent (if ported)

## Out of Scope

- Auto-detecting upstream doc content changes
- Item-level tracking within docs (doc-level only)
- Modifying upstream doc body content (frontmatter only)
- Automatically setting IN_PROGRESS (manual trigger)

## Migration Path

1. **Day 1:** Add frontmatter.yaml, run Hugo to generate defaults
2. **Day 1:** Manually set IN_PROGRESS for any active Phase 5 work
3. **Ongoing:** As subphases complete, script sets PORTED
4. **Ongoing:** Dashboard shows real-time parity
