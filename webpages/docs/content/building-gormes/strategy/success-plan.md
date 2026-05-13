---
title: "Gormes Success Plan"
weight: 5
---

# Gormes Success Plan — 12 months

> **Status**: living strategy doc. Owns the answer to "what does success look like for Gormes?" Pairs with Phase 8 rows in `progress.json` for executable tracking.

## North Star (one sentence)

> **TrebuchetDynamics is the team that figured out how to autonomously port large Python projects to Go in production. Gormes is the receipt.**

The product is the methodology. Hermes-parity is evidence the methodology works, not the goal itself.

## Why This Doc Exists

Gormes started as "Hermes in Go" — a parity port. That framing is *unwinnable solo*: Hermes ships ~800–1,150 commits per week from a 200-contributor base. Even an autonomous Opus-class loop landing 1–2 features/hour falls behind at ~10×, and the gap compounds. As a parity chase, this is treadmill running.

As a **reputation play for TrebuchetDynamics**, the calculus inverts: the methodology, the autonomous loop, the validation-gated agentic engineering, and the one or two sharp differentiators are the assets. The 700+ ported rows are the *receipt* that proves the methodology works.

This doc captures that pivot.

## Strategic Pivot

| Stop                                              | Start                                                                  |
|---------------------------------------------------|------------------------------------------------------------------------|
| "Gormes is Hermes in Go"                          | "Gormes is the artifact our agentic engineering system produces"       |
| Optimizing row count                              | Optimizing publication & differentiation                               |
| Chasing every weekly Hermes release indiscriminately | Tracking Hermes selectively, on a Pareto cut                       |
| Private commits as "progress"                     | Public writeups as progress                                            |
| 100% effort into substrate                        | ~70% substrate / ~30% publication                                      |

## Operating Principles

1. **Reputation is built through publication cadence, not commit cadence.** A strangers-can-find-it artifact every 30–45 days beats 1,000 invisible commits.
2. **Solo devs win on opinion, not scope.** Pick sharp differentiators; let parity rot in the background.
3. **The loop is the product.** Open-source it; that is what other teams cannot easily replicate.
4. **Cost discipline.** Track $/feature shipped from the autonomous loop. If a week of loop spend produces zero publishable artifacts, downshift the cron cadence.
5. **Owned divergence is fine when it serves the strategy.** Hermes parity is the oracle, not a contract — Gormes can decline upstream features that contradict the sharp v1.0 cut.

## Quarterly Roadmap

### Q1 — Foundation + First Public Artifact

Goal: establish TD's public presence and ship one writeup-shaped artifact.

- Stand up the publication stack (TD engineering blog, social account, posting cadence).
- Pick the sharp v1.0 differentiator (working title: *"runs the 30 most-used Hermes skills unchanged, in a single 30 MB Go binary, on Termux + Windows-without-Python + locked-down corp Linux"*).
- Ship engineering writeup #1: *"How an autonomous loop ships 1–2 Hermes-parity features per hour with TDD discipline."*
- Rewrite the Gormes README to lead with methodology, not parity.
- Wire `$/iteration` cost telemetry into the autonomous loop.

Q1 ship metric: blog live, 1 writeup public, README rewritten, cost telemetry in place.

### Q2 — Open-Source the Toolkit

Goal: make the agentic-engineering skill set a citable asset.

- Extract `gormes-planner`, `gormes-builder`, `gormes-tdd-slice`, `gormes-parity-auditor`, `gormes-references`, `gormes-skill-manager` into a separate public repo (`TrebuchetDynamics/agentic-porting-kit` or equivalent).
- Demonstrate the kit on a second porting target.
- Ship writeup #2: *"Validation-gated agentic engineering: how we keep autonomous commits green for 700+ rows."*
- Ship writeup #3: *"Cold start, memory, and skill-execution: Gormes vs Hermes-Python head-to-head."*
- Submit at least one conference CFP (GopherCon / Strange Loop / AI Engineer Summit).

Q2 ship metric: toolkit repo public with ≥50 GitHub stars, 2 more writeups, ≥1 CFP submitted.

### Q3 — Sharp v1.0 Demo

Goal: a demo strangers can run today.

- Tag Gormes 1.0 as a *bounded* product:
  - Single binary on Linux/macOS/Windows.
  - Runs the curated 30 most-used Hermes skills unchanged.
  - Honest "not in 1.0" page so users self-select.
- Stand up `gormes.ai/benchmarks` with reproducible cold-start, RSS, install-size, and skill-execution numbers.
- Stand up a "Built with Gormes" page (even with a single entry).
- HN launch post: *"Show HN: Gormes 1.0 — single-binary Go runtime that runs Hermes skills."*
- Writeup #4: post-launch retrospective.

Q3 ship metric: v1.0 tagged release, benchmarks page live, HN front-page attempt made.

### Q4 — Compound

Goal: convert public attention into community.

- Deliver the conference talk(s) accepted in Q2/Q3.
- Engage actively on issues / discussions in public repos. Even single-digit outside contributors compound reputation faster than 100 internal commits.
- Newsletter / monthly digest of "what the loop did this month" — solves publication cadence on autopilot.
- Begin a second autonomous-port project using the toolkit; even a small one demonstrates portability of the methodology.

Q4 ship metric: ≥1 conference talk delivered, ≥3 outside contributors, ≥500 toolkit stars, recurring publication cadence locked in.

## What to STOP Doing

- **Treating row #844 as equal priority to row #1.** Most late rows are decorative. The loop grinds because grinding feels productive.
- **Adding new rows from every Hermes release indiscriminately.** Pareto-filter: does this affect the 30-skill demo or a known user-visible regression? If not, it goes to a `parked` lane, not the active queue.
- **Hand-mirroring every upstream-hermes doc.** Already a maintenance liability. Decide: automate the mirror with a sync script that handles `/docs/...` link rewriting, or stop mirroring and link to upstream. Do not half-do it.
- **Running the autonomous loop 24/7 without cost discipline.** A daily $/iteration line in the loop's status file is non-negotiable from Q1 onward.
- **Building Goncho parity in parallel as a critical path.** Either Goncho is its own product with its own reputation arc, or it is a deferred row. Do not let it eat planner cycles in Q1–Q2.

## Reputation Metrics

| Metric                                              | Q1 target | Q4 target |
|-----------------------------------------------------|-----------|-----------|
| Blog/feed live                                      | yes       | yes       |
| Public writeups (cumulative)                        | 1         | 5+        |
| HN front-page hits                                  | 0         | 1         |
| Toolkit repo stars                                  | n/a       | 500+      |
| Outside contributors (lifetime)                     | 0         | 3+        |
| Conference talks delivered                          | 0         | 1+        |
| "Built with Gormes" / kit-uses mentions             | 0         | 5+        |
| $/published-artifact (loop spend ÷ artifacts)       | tracked   | <$200     |

**Stop tracking as primary**: progress.json completion %, row count, commits/day. These are leading indicators *only if* publication metrics are also moving. If they are not, row count is just expensive treadmill running.

## Risk Register

| Risk                                                | Mitigation                                                                                          |
|-----------------------------------------------------|-----------------------------------------------------------------------------------------------------|
| Publication paralysis ("not ready yet")             | Time-box: each writeup ships at 80% polish or it does not ship. Calendar publish dates.             |
| Identity drift back to "we are a port"              | Quarterly review: read the README and writeups; if methodology is not the lede, fix it that day.    |
| Loop spend without learning                         | Monthly cost review. If $/feature crosses a threshold (e.g., $5), downshift cadence.                |
| Upstream Hermes pivots in a way that invalidates parity | Embrace it as content: *"What we learned when our parity oracle pivoted."* Better artifact anyway. |
| Burnout (multi-year arc, solo)                      | The loop runs while you do not. Use that. Sustainable cadence beats sprint-then-collapse.           |

## 30-Day Action Sprint

If nothing else gets done, do these:

1. Stand up the TD engineering blog (Astro + a static host = a weekend).
2. Rewrite the Gormes README to lead with the autonomous-porting methodology, not "Hermes in Go".
3. Draft writeup #1 using actual loop commits as the example diff walkthrough.
4. Pick the sharp v1.0 differentiator and write it down.
5. Add `$/iteration` to the loop's status output.

If item 1 takes longer than a weekend, that is the signal that **publication is the actual blocker**, not engineering. Solve that first.

### Publication Stack

- Blog URL: <https://engineering.trebuchetdynamics.com/>
- Feed URL: <https://engineering.trebuchetdynamics.com/feed.xml>
- Source path: `webpages/blog/`
- Deploy path: `.github/workflows/deploy-td-blog.yml` builds the Astro site and deploys `webpages/blog/dist` to Cloudflare Pages on `main`.
- First post: "Autonomous Hermes-porting loop" at `/posts/autonomous-hermes-porting-loop/`.

## Mapping To `progress.json`

Phase 8 (`Reputation & Publication`) carries the executable rows for this plan. Subphases:

- `8.A` — Publication infrastructure (blog, social, feed, posting cadence).
- `8.B` — Repository messaging (README, landing page, positioning).
- `8.C` — Engineering writeups (the writeup pipeline).
- `8.D` — Sharp v1.0 (differentiator definition, benchmarks, demo).
- `8.E` — Toolkit extraction (open-sourcing the gormes-* skill set).
- `8.F` — Cost discipline & loop economics.
- `8.G` — Community & external contributions.

Rows in Phase 8 are explicitly classified `provenance: gormes` (owned divergence). They reference this strategy doc as the source-of-truth in `source_refs`.

## Cross-Reference

- `docs/content/building-gormes/architecture_plan/progress.json` — executable rows, Phase 8.
- `README.md` — operator-facing positioning, must align with the North Star.
- `webpages/landing/` — public messaging, must align with the North Star.
- `hermes-agent/` (submodule) — parity oracle, not contract.

## Living Doc Discipline

Update this doc when:

- A quarterly milestone ships or is missed (note the actual outcome).
- The differentiator shifts (record the change and the reason).
- A new risk emerges or a listed risk fires.
- The publication cadence changes.

Do not update this doc to inflate progress. The reputation metrics table is the scoreboard.
