---
title: "Good and Bad"
weight: 20
---

# Good And Bad

## What GBrain Gets Right

### 1. Contract-First Tooling

The operation catalog is the best architectural move. One operation definition
feeds CLI, MCP, generated tool JSON, and subagent allowlists. Tests pin the
mapping. This prevents the classic agent-runtime failure where a CLI command,
an MCP tool, and the model's tool schema all mean slightly different things.

Gormes should keep its typed Go `Tool` interface, but add an operation
descriptor layer around it so command surfaces, gateway surfaces, doctor output,
and model tool schemas are generated from one source.

### 2. Explicit Trust Boundaries

GBrain distinguishes local CLI from remote/agent calls. That makes policy
visible in code instead of hidden in handler comments. Good examples include
remote file confinement, subagent namespace guards, MCP rejection of protected
shell jobs, and remote auto-link suppression.

Gormes should model trust class in Go types, not as ad hoc booleans.

### 3. Durable Deterministic Work

Minions correctly separates deterministic work from judgment work. Shell jobs,
sync, extract, embed, and cron-like maintenance should not burn an LLM turn or
depend on an ephemeral agent process. The SQL queue gives jobs durable state,
retries, locks, idempotency, progress, child completion, and operator inspection.

Gormes Phase 2.E already gives subagents context cancellation and bounded
execution. The upgrade path is a durable ledger for jobs and subagent turns.

### 4. Graph Provenance

The `links` table records link source and origin page. Auto-generated edges can
coexist with manual edges, and reconciliation can remove only the edges a page
actually authored. This is a better graph model than "relationship exists"
without evidence.

Gormes memory should add comparable provenance to relationships: source turn,
extractor version, origin field, confidence, first seen, last seen, and whether
the edge was manual, inferred, or imported.

### 5. Graceful Degradation

If embeddings are unavailable, search falls back to keyword. If expansion fails,
query still runs. Some post-write enrichment failures are non-fatal. This keeps
the operator path usable when a model key or provider is missing.

The correct Gormes version is: degrade visibly. Return useful results, but emit
health and audit evidence so silent quality loss does not become normal.

### 6. Test Surface Around Architecture

GBrain has tests for operation parity, MCP tool definition equivalence,
allowlist correctness, file-upload security, queue state transitions, stalls,
timeouts, plugin loading, resolver checks, search behavior, and migrations.

This is not just unit testing. It is architecture testing.

### 7. Agent-Readable Repo Protocol

`AGENTS.md`, `CLAUDE.md`, `llms.txt`, skills, and migration docs make the repo
usable by an agent that has never seen it before. The docs are part of the
runtime strategy.

Gormes should preserve the same spirit, but keep the Go repo's source of truth
tight: code contracts, docs, and generated references should cross-check each
other.

## What GBrain Gets Wrong Or Risky

### 1. Too Much Gravity In Central Files

The main operation catalog is 1334 lines. The queue is 1281 lines. Each engine
implementation is over 1000 lines. These files are heavily commented and tested,
but they carry too many policy decisions at once.

Risk for Gormes: copying the "single huge file" style would fight Go package
boundaries. Gormes should keep one logical registry, but split implementation by
domain with small typed specs.

### 2. The Storage Story Is Powerful But Heavy

Postgres plus pgvector plus PGLite gives a strong database model, but it also
creates a large migration and engine-compatibility burden. The interface is
clean in principle, yet SQL details still leak into migrations, job queue,
graph traversal, search, and health checks.

Gormes should stay SQLite-first for the default product. Optional Postgres can
come later behind an interface, but the default operator promise is still one
binary and one local data directory.

### 3. Remote Safety Is Incomplete Unless Every Handler Participates

`OperationContext.remote` is a good start, but it only works when every handler
checks it correctly. Some policies live in operations, some in queue submission,
some in worker registration, and some in schemas.

Gormes should make trust policy declarative on the tool descriptor and enforce
it in a shared executor before handler code runs.

### 4. Auto-Linking Can Poison The Graph

GBrain's own code comments admit the risk: untrusted content can plant strings
that become graph edges and influence backlink-boosted ranking. Remote writes
skip auto-link for this reason.

Gormes should treat graph extraction as an untrusted derivation pipeline:
candidate edge, evidence, confidence, queue state, review policy, then promote.
Direct graph mutation from arbitrary inbound text should be constrained.

### 5. Non-Fatal Failures Can Become Invisible

Embedding, expansion, backlink boost, auto-link, and writer lint failures often
fall back silently or return partial side fields. This is good for uptime, but
bad for long-term brain quality if health checks are not prominent.

Gormes should define "degraded but honest" as a product rule: every fallback
must be visible in status, audit logs, and maybe docs-generated health reports.

### 6. Skills Are Powerful But Easy To Over-Trust

Fat markdown skills encode real workflow knowledge, but they are still text.
They can drift, overlap, route poorly, or become stale. GBrain counters this
with conformance checks and routing evals, but the more skills exist, the more
the resolver becomes a core subsystem.

Gormes should not ship a large skillpack before it can prove activation,
coverage, conflicts, versioning, and operator review.

### 7. Universal Agent Claims Still Depend On Provider-Specific Code

The durable subagent handler is Anthropic-shaped at the code boundary. That is
reasonable for a first implementation, but it means the "universal agent
protocol" is partly aspirational.

Gormes Phase 4 should keep provider adapters behind its native `internal/hermes`
contract and avoid letting one provider's streaming model leak into the
subagent/job ledger.

### 8. Shell Jobs Remain Dangerous

GBrain protects shell jobs with CLI-only submission and an explicit worker env
flag. That is correct, but the shell handler is still a broad filesystem and
process execution surface. It is operationally useful, not safe by default.

Gormes's `execute_code` and future scheduled shell surfaces should remain
separate from ordinary child-agent tools, with audit, allowlists, timeouts,
working-directory policy, output caps, and opt-in enablement.

### 9. Docs Can Drift Faster Than Code

GBrain has rich docs, but the project is evolving quickly. README claims,
versioned docs, skill counts, and code reality can diverge. This is visible in
the repo's large changelog-style `CLAUDE.md` and many migration notes.

Gormes should generate capability reference pages from code where possible, and
keep high-level narrative docs short enough to maintain.

## Bottom Line

GBrain's best ideas are structural:

- one operation contract
- explicit trust class
- durable work ledger
- knowledge pages with graph provenance
- skill conformance checks
- brain-first read/write loop

Its risks come from scale and coupling:

- large central files
- heavy storage assumptions
- implicit handler participation in safety policy
- graph poisoning risk
- hidden degraded modes
- markdown skill sprawl

Gormes should adopt the structural ideas in Go, not the mass of the TypeScript
implementation.

