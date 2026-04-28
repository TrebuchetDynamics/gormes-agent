# Gormes Skill Routing

Use this table to select the smallest useful skill chain.

Start from `docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md`.
Every non-trivial task should name the map area and the matching
`progress.json` row before picking a skill.

Skill files are edited in `docs/development-skills/<name>/`; `.agents/skills/`,
`.claude/skills/`, and `.codex/skills/` should remain symlink views.

## Existing Gormes Skills

| Situation | Use | Then |
|---|---|---|
| User wants to stress-test direction or make a hard decision | `grill-me` | `gormes-planner` after decision |
| Need to compare Hermes/Honcho/GBrain against Gormes | `gormes-parity-auditor` | `gormes-planner` to update rows |
| Need Hermes CLI/config/migration parity, including `migrate hermes`, `migrate openclaw`, or typo requests like `migrate ooenclaw` | `gormes-parity-auditor` | `gormes-planner` updates manifest/config/migration rows and keeps typos as suggestions unless a compatibility row exists |
| Need to update roadmap/progress rows | `gormes-planner` | `gormes-builder` for selected row |
| Need Go API/package shape before coding | `gormes-interface-designer` | `gormes-planner` row or `gormes-builder` implementation |
| Need to implement one progress row | `gormes-builder` | `gormes-tdd-slice` inside implementation |
| Need strict red-green-refactor on one behavior | `gormes-tdd-slice` | `gormes-builder` final report |
| Need to audit or periodically refresh README/public repository messaging | `gormes-readme` | `gormes-planner` only if roadmap rows need edits |
| Need to improve `www.gormes.ai` landing page content or UI | `gormes-landing-web` | `gormes-planner` only if roadmap/progress claims need edits |
| Need to create or improve a skill | `gormes-skill-manager` + system `skill-creator` | validate all affected skills |

## Feature-Map-First Routing

| Input state | Primary skill | Follow-up |
|---|---|---|
| Upstream behavior exists but the feature map lacks it | `gormes-parity-auditor` | `gormes-planner` updates map and row |
| Feature map is correct but no row exists | `gormes-planner` | `gormes-builder` after row is ready |
| Row exists but is broad, vague, blocked, or lacks tests | `gormes-planner` | `gormes-interface-designer` if API shape is unclear |
| Row is builder-ready and package boundary is known | `gormes-builder` | `gormes-tdd-slice` for red-green cycles |
| Row is builder-ready but public interface is unclear | `gormes-interface-designer` | `gormes-planner` records the contract |
| Tests fail for an existing slice | `gormes-tdd-slice` | `gormes-builder` updates progress evidence |
| Repeated work lacks a stable skill path | `gormes-skill-manager` + system `skill-creator` | `gormes-planner` adds/updates a skill row |

## Completion Lane Coverage

| Lane | Current route | Skill gap decision |
|---|---|---|
| 1.D skill control plane | `gormes-skill-manager` -> `skill-creator` -> `gormes-planner` | Covered by this routing table. |
| 3.G Goncho/Honcho compatibility | `gormes-parity-auditor` -> `gormes-planner` -> `gormes-builder` -> `gormes-tdd-slice` | Add `gormes-goncho-compat` only after SDK-style fixtures repeat across multiple rows. |
| 4.I normal-turn e2e | `gormes-builder` -> `gormes-tdd-slice` | Add `gormes-e2e-operator` when service orchestration or Playwright-style harnesses repeat. |
| Provider transcript parity | `gormes-parity-auditor` -> `gormes-builder` -> `gormes-tdd-slice` | Add `gormes-provider-parity` when transcript fixtures span multiple providers. |
| Hermes CLI/config/migration parity | `gormes-parity-auditor` -> `gormes-planner` -> `gormes-builder` | Covered by planner/auditor/builder updates; add a dedicated CLI parity skill only if command-manifest and handler rows repeat across many passes. |
| Gateway/channels | `gormes-parity-auditor` -> `gormes-interface-designer` -> `gormes-builder` | Add `gormes-channel-adapter` after at least two channel adapters need the same contract gates. |
| README/public repo messaging | `gormes-readme` | Covered by the periodic README evidence workflow. |
| Landing page content/UI | `gormes-landing-web` | Covered by the focused public homepage workflow. |
| Docs/Hugo/site sync | `gormes-planner` | Add `gormes-docs-web-sync` if progress docs and broader site data updates keep drifting. |
| Release/install packaging | `gormes-planner` -> future release rows | Add `gormes-release-packager` when installer, service, OCI, and version gates are active. |

## Good Skill Chains

- Upstream gap to implementation:
  `gormes-parity-auditor` -> `gormes-planner` -> `gormes-builder` -> `gormes-tdd-slice`

- Ambiguous package/interface:
  `gormes-interface-designer` -> `gormes-planner` -> `gormes-builder`

- User is uncertain about direction:
  `grill-me` -> `gormes-skill-manager` -> selected Gormes skill

- Broken implementation row:
  `gormes-tdd-slice` -> `gormes-builder`

## New Skill Candidates

Create these only when repeated work proves the need:

- `gormes-goncho-compat`: Honcho request/response compatibility fixtures, migration expectations, public `honcho_*` tool behavior.
- `gormes-provider-parity`: model/provider routing, provider-specific errors, streaming, token accounting, hermetic provider fixtures.
- `gormes-channel-adapter`: Telegram/Discord/Slack adapter contracts, gateway handoff, platform webhook fixtures.
- `gormes-docs-web-sync`: progress docs, Hugo docs, `www.gormes.ai` data and public progress messaging.
- `gormes-e2e-operator`: Playwright, CLI e2e, long-running loop smoke tests, local service orchestration.
- `gormes-release-packager`: install scripts, service units, binaries, versioning, release verification.

## Anti-Routing

- Do not use `gormes-planner` for runtime implementation.
- Do not use `gormes-builder` to invent backlog rows.
- Do not use `gormes-interface-designer` when the row already has a clear existing package pattern.
- Do not create a new skill because a task is large; split the task first.
