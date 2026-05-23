---
title: "Autonomous Hermes-porting loop review packet"
weight: 20
---

# Autonomous Hermes-porting loop review packet

Status: local review packet only. This file is not a public publication signal,
does not choose a launch date or target platform, and does not require social
credentials. It prepares Juan to review Engineering Writeup #1 before any public
posting.

## Source artifacts

- Draft post: `webpages/blog/src/content/posts/autonomous-hermes-porting-loop.md`.
- Local post URL after blog build: `/posts/autonomous-hermes-porting-loop/`.
- Public planned post URL: `https://engineering.trebuchetdynamics.com/posts/autonomous-hermes-porting-loop/`.
- Feed URL: `https://engineering.trebuchetdynamics.com/feed.xml`.
- Publication stack source: `webpages/docs/content/building-gormes/strategy/success-plan.md#publication-stack`.
- Progress parent row: `Engineering writeup #1: autonomous Hermes-porting loop` in Phase `8.C`.
- Builder row for this packet: `Engineering writeup #1 local publication review packet` in Phase `8.C`.

## Draft evidence inventory

The draft is ready for local editorial review because it already carries the
core evidence shape:

1. Loop invariant: `Intent → Oracle → Surface → Work package → Proof`.
2. Architecture diagram: `progress.json` planner, builder, TDD slice,
   validation, generated docs, and feedback loop.
3. Walkthrough 1: commit `2e3a56a21`, Termux installer safety via the
   `update_active_command` seam in `install.sh`, with a hermetic install-test
   proof rather than live phone state.
4. Walkthrough 2: commit `5cd4f870f`, Navivox `connect` command compatibility
   alias and user-visible diagnostics, with command-help coverage.
5. Walkthrough 3: commit `c2a267dad`, development-loop receipt logging for
   overlapping local work and validation evidence.
6. Validation gate ladder: focused test, package test, `go run ./cmd/progress
   validate`, `git diff --check`, and broad `go test ./... -count=1` before
   integration-ready claims.
7. Honest non-final section: cost-per-feature is not ready to publish until
   measured telemetry exists.

## Local validation commands

Run these before asking Juan for publication review:

```bash
(cd webpages/blog && npm run test)
go test ./webpages/docs -count=1
go run ./cmd/progress validate
git diff --check
```

The social preview stays dry-run only:

```bash
node scripts/blog-feed-social/publish.mjs --dry-run --feed webpages/blog/dist/feed.xml --out /tmp/gormes-social-post.json
```

Dry-run social automation reads the built feed and writes deterministic preview
JSON. It must keep `network_publish` false and must state that no social tokens
are read in dry-run mode.

## Pending final-publication inputs

These are intentionally unresolved. Do not invent them in the draft or in this
packet.

- One week of measured cost telemetry for the autonomous loop, including token
  or provider spend that can support cost-per-feature claims.
- Publication date selected by Juan.
- Target platform selected by Juan, for example TrebuchetDynamics feed only,
  Hacker News, Lobsters, Reddit, LinkedIn, or another operator-approved outlet.
- Social account or posting credential location supplied outside the repository
  if Juan wants live syndication.
- Final public title and summary once the platform is selected.

## Secret and live-publication boundary

- No social tokens belong in this repository.
- No credential name, token value, cookie, or account session should be copied
  into this review packet.
- Live posting remains blocked until Juan selects the platform/account and
  provides credentials outside the repo.
- The dry-run command above is the only social automation this row authorizes.
- If a future agent cannot find telemetry or social credentials, it must report
  those as pending operator inputs instead of using estimates or attempting live
  publication.

## Juan-facing review questions

1. Voice/tone: should the draft sound more like a field note, a launch essay,
   or a technical postmortem?
2. Audience: is the intended reader a Go developer, an engineering leader, an
   agent-infra builder, or a potential TrebuchetDynamics customer?
3. Evidence threshold: are the three walkthrough commits enough for writeup #1,
   or should the post wait for one more fresh slice with cleaner before/after
   screenshots or logs?
4. Cost disclosure: once one week of measured cost telemetry exists, should the
   post include a table, a short caveat, or a separate follow-up economics post?
5. Publication date: what date should be used for the public launch?
6. Target platform: should the first public push go to the TrebuchetDynamics
   engineering feed only, or also to an external platform?
7. Social copy: should the dry-run text stay terse, or should it include the
   sharper claim that Gormes is the receipt for autonomous Python-to-Go porting?

## Recommended local next step

After this packet validates, keep the parent row blocked until the missing
telemetry and operator publication decisions exist. The next safe builder slice
is to wire measured loop-cost telemetry or prepare an operator-reviewed launch
copy variant, not to publish live from this repository.
