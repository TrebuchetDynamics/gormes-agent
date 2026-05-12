---
title: "Agent Queue"
weight: 20
aliases:
  - /building-gormes/agent-queue/
---

# Agent Queue

This page is generated from the canonical progress file:
`docs/content/building-gormes/architecture_plan/progress.json`.

It lists unblocked, non-umbrella contract rows that are ready for a focused
skill-driven implementation attempt. Each card carries the execution owner,
slice size, contract, trust class, degraded-mode requirement, fixture target,
write scope, test commands, done signal, acceptance checks, and source
references.

Shared skill handoff facts live in [Skill Builder Handoff](../builder-loop-handoff/):
the main skill entrypoint, plan, candidate source, generated docs, tests, and
candidate policy. Keep those control-plane facts in `meta.builder_loop`, and
keep row-specific execution facts in `progress.json`.

If the generated list is empty, do not switch to an ad hoc TODO list. Route
through `gormes-planner`, repair one planned/draft row until it satisfies the
handoff contract, validate `progress.json`, and then return to builder
selection.

<!-- PROGRESS:START kind=agent-queue -->
## 1. Goncho-backed dynamic agent registry

- Phase: 2 / 2.H
- Owner: `memory`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: internal/goncho/dynamic_agents.go exposes a DynamicAgentRegistry (Create, Get, List, Bind, Unbind, Resolve) persisting agents and channel/peer bindings in new SQLite tables alongside Goncho memory. The gateway's existing agentRouter consults the registry as an overlay on top of config.AgentsCfg/AgentBindingCfg: static config wins on AgentID conflict (operator-defined identity), but dynamic bindings supplement runtime peer matches that static config does not cover. Public Go interface stays small; persistence, migration, and conflict semantics stay hidden behind the registry boundary.
- Trust class: operator
- Ready when: config.AgentsCfg and config.AgentBindingCfg remain the static-config source of truth; the registry overlays runtime entries without rewriting config.toml., Existing internal/goncho schema migrations apply cleanly under a new migration adding dynamic_agents and dynamic_agent_bindings tables., The registry interface is testable with a tempdir SQLite DB; no live channel adapter or gateway integration test is needed for this slice.
- Not ready when: The slice edits internal/config/agents.go schema, internal/gateway/agent_runtime.go resolver semantics beyond a single overlay-lookup hook, or any platform adapter., The slice writes to config.toml, persists agent state in cleartext outside Goncho, or shares storage with cross-session memory rows., The slice changes static-vs-dynamic conflict order — static config must win on AgentID conflict so operator-defined identity is not silently shadowed.
- Degraded mode: Without runtime mutation, every new agent persona requires editing config.toml and reloading; in-chat spawn UX from rows 2.H.2-4 cannot exist.
- Fixture: `internal/goncho/dynamic_agents_test.go tempdir SQLite + fake clock`
- Write scope: `internal/goncho/dynamic_agents.go`, `internal/goncho/dynamic_agents_test.go`, `internal/goncho/migrations/ (one new migration file)`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/goncho -run '^TestDynamicAgentRegistry_' -count=1`, `go test ./internal/goncho -count=1`, `go run ./cmd/progress validate`
- Done signal: Registry interface, persistence migration, and four named tests land in internal/goncho with the wider goncho suite still green; no runtime change in cmd/gormes or gateway in this row.
- Acceptance: TestDynamicAgentRegistry_CreateRoundTrips proves Create + Get returns the same AgentRecord with stable ID and persona seed across a registry reopen., TestDynamicAgentRegistry_BindResolvesByPeer proves Bind + Resolve returns the dynamic AgentID for a (channel, chat_id, thread_id) match., TestDynamicAgentRegistry_StaticConfigWinsOnIDConflict proves an overlapping static AgentCfg.ID is returned by the resolver even when a dynamic record with the same ID exists., TestDynamicAgentRegistry_UnbindRemovesMatch proves Unbind removes the persisted binding and Resolve falls back to static config (or returns not-found).
- Source refs: https://github.com/OpenYabby/OpenYabby#whatsapp-but-agent-native, internal/goncho/local_markdown_memory.go (AgentID partition key precedent), internal/config/agents.go (AgentsCfg, AgentCfg, AgentBindingCfg schema this layers on top of), internal/gateway/agent_runtime.go (agentRouteRequestFromInbound — thread-aware peer resolution already wired)
- Unblocks: gormes agent spawn/list/inspect/bind/unbind CLI, Telegram /spawn opens forum topic bound to spawned agent
- Why now: Unblocks gormes agent spawn/list/inspect/bind/unbind CLI, Telegram /spawn opens forum topic bound to spawned agent.

## 2. TD engineering blog scaffolded and live

- Phase: 8 / 8.A
- Owner: `docs`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: TrebuchetDynamics has a publicly reachable engineering blog with a working Atom/RSS feed, an /about page that names the org and the methodology, and a deploy pipeline so a markdown commit becomes a published post without manual intervention. Hosting choice is owner's call (Astro/Hugo/Eleventy + Cloudflare/Vercel/GitHub Pages); the row is done when a stranger can subscribe to a feed and read one published post.
- Trust class: operator
- Ready when: Hosting choice and blog framework are decided (operator decision; not loop-driven)., A subdomain or path on an existing TD-controlled domain is available.
- Not ready when: The blog is private, password-protected, or behind authentication., There is no Atom/RSS feed at a stable URL., The first post is empty or placeholder text rather than the writeup #1 draft or a real introduction.
- Degraded mode: Without a publication outlet, every loop commit is invisible in the reputation market; the strategy described in success-plan.md cannot start.
- Fixture: `webpages/blog/ (or chosen blog repo path)`
- Write scope: `webpages/blog/ (or external blog repo path)`, `DNS / Cloudflare / hosting config (operator-only)`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: -
- No test required: Documentation/research/planning row — automated tests not applicable
- Done signal: Public blog URL + feed URL recorded in success-plan.md and README.md.
- Acceptance: Blog is reachable at a public URL with at least one real (non-placeholder) post., An Atom or RSS feed exists at a stable, discoverable URL., Publishing a new post is a markdown-commit-and-merge operation; no console click-through required., An /about page exists that names TrebuchetDynamics and points at gormes-agent + agentic-porting-kit.
- Source refs: docs/content/building-gormes/strategy/success-plan.md, webpages/landing/
- Unblocks: Engineering writeup #1: autonomous Hermes-porting loop, Monthly digest pipeline
- Why now: Unblocks Engineering writeup #1: autonomous Hermes-porting loop, Monthly digest pipeline.

## 3. Agentic-porting-kit repo scaffold

- Phase: 8 / 8.E
- Owner: `skills`
- Size: `large`
- Status: `planned`
- Priority: `P2`
- Contract: The gormes-* skill set (gormes-planner, gormes-builder, gormes-tdd-slice, gormes-parity-auditor, gormes-references, gormes-skill-manager) is extracted into a separate public TrebuchetDynamics repo (`agentic-porting-kit` or equivalent), with a README that frames the kit as a generic Python→Go porting toolkit, a worked example using a small non-Hermes target, and a clear license. The kit must work standalone — its rows must be loadable by Codex or Claude Code in any repo, not just Gormes.
- Trust class: operator
- Ready when: All listed skills have a README of their own that does not assume the Gormes repo layout., Skills' references that hard-code Gormes paths have been parameterized or generalized.
- Not ready when: Skills still hard-code paths under docs/content/building-gormes/., The extracted kit cannot be tested without cloning Gormes.
- Degraded mode: Without extraction, the methodology is invisible to other teams; "the loop is the product" cannot be substantiated externally.
- Fixture: `(separate repo: TrebuchetDynamics/agentic-porting-kit)`
- Write scope: `(separate repo)`, `webpages/docs/development-skills/ (de-Gormes-fy paths)`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: -
- No test required: Documentation/research/planning row — automated tests not applicable
- Done signal: Repo URL recorded in success-plan.md and README.md; star count tracked monthly.
- Acceptance: Public repo TrebuchetDynamics/agentic-porting-kit exists with the listed skills., Repo README explains the kit independent of Gormes/Hermes., A worked example demonstrates the kit on a non-Hermes target (any small Python project being ported to Go)., Skills can be loaded into a fresh Codex or Claude Code session and successfully plan-and-execute one row in the example target.
- Source refs: docs/content/building-gormes/strategy/success-plan.md, webpages/docs/development-skills/gormes-planner/SKILL.md, webpages/docs/development-skills/gormes-builder/SKILL.md, webpages/docs/development-skills/gormes-tdd-slice/SKILL.md, webpages/docs/development-skills/gormes-parity-auditor/SKILL.md, webpages/docs/development-skills/gormes-references/SKILL.md, webpages/docs/development-skills/gormes-skill-manager/SKILL.md
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
