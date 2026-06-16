# Cross-Project Feature Map & Long-Term Plan for Gormes-Agent

## Executive Summary

This map tracks the public donor surface that remains relevant after the
knowledge-runtime donor was retired from Gormes planning.

Active sources:

- **hermes-agent**: Python upstream and primary parity contract.
- **honcho**: memory/session platform that defines the Goncho compatibility target.
- **browser-harness** and **go-browser-harness**: browser automation references.
- **mercury-agent**: safety, token budget, loop detection, and operator UX patterns.
- **space-agent**: browser-first UX and metadata-driven skill placement patterns.
- **picoclaw**: Go-native channel/provider examples.
- **go-agent-os references**: Go patterns for OAuth, retry, tools, state, stores, and queues.
- **goscrapling**: local Go-native Scrapling-style extraction engine embedded
  behind Gormes `web_extract` for static extraction, with future crawler/browser
  behavior gated separately.

The rule is unchanged: Hermes defines what to port, Honcho defines memory
compatibility, and Go donors only shape implementation details.

## Feature Matrix

| Area | Canonical source | Gormes state | Next focus |
|---|---|---|---|
| Provider adapters | Hermes | Many core providers are implemented; fallback, pricing, and error parity continue | Provider fallback chain, retry-after parsing, model metadata |
| Browser automation | Hermes + go-browser-harness | Core actions are implemented; daemon/profile diagnostics remain | Browser doctor, profile lifecycle, event drain |
| Web extraction | Hermes + goscrapling | `web_search`, `web_extract`, and `web_crawl` exist; static goscrapling extraction is embedded behind `web_extract` with structured `extraction` evidence | Builder-ready browser-backed `web_extract` and local `web_crawl` gates once goscrapling prerequisites validate |
| Tool registry | Hermes | Native registry exists, but broad file/web/MCP/tool safety parity remains | Permission-hardened shell, tool descriptor layer, truncation |
| Memory | Honcho + Go donors | Goncho exists as the local compatibility lane | Typed memory, provenance, retrieval evaluation, durable write queue |
| Gateway/channels | Hermes + Picoclaw | Telegram, Discord, and Slack are runtime-ready; wider channels remain row-backed | Channel status labels, adapter parity fixtures |
| CLI/operator UX | Hermes | Setup, doctor, model/profile/auth surfaces are partially aligned | Command tree parity, setup text matrix, logs/status parity |
| Safety | Hermes + Mercury | Security audit exists; executor enforcement still broadens | Shell blocklist, filesystem scopes, approval gates |
| Scheduling/jobs | Hermes + Go donors | Durable ledger slices exist; advanced worker behavior remains row-backed | Queue health, replay/inbox, worker supervisor evidence |
| Skills | Hermes + Space Agent | SKILL.md parsing, validation, and registry basics are present | Metadata placement, resolver evals, review/version state |

## Suggested Roadmap Inputs

| Priority | Row idea | Source refs | Target |
|---|---|---|---|
| P0 | Permission-hardened tool execution | `mercury-agent/src/core/permissions.ts`, `../hermes-agent/tools/path_security.py` | `internal/tools` |
| P0 | Native prompt builder | `../hermes-agent/agent/prompt_builder.py` | `internal/agenttemplate`, prompt assembly packages |
| P1 | Provider fallback chain | `mercury-agent/src/core/providers.ts`, `picoclaw/pkg/providers/` | `internal/provider` |
| P1 | Browser harness doctor | `go-browser-harness` docs and tests | browser harness CLI and Gormes doctor evidence |
| P1 | goscrapling browser/crawler gate planning | `../goscrapling/webpages/docs/content/building-goscrapling/strategy/portfolio-and-gormes-fit.md`, `../goscrapling/webpages/docs/content/building-goscrapling/architecture_plan/progress.json` | Keep goscrapling behind `web_extract`/`web_crawl`; require browser fetcher and spider prerequisite fixtures before dynamic extraction or local crawling |
| P1 | Loop detection | Mercury loop-detection patterns | `internal/kernel`, `internal/agent` |
| P2 | Structured memory types | Honcho models, Engram store references | `internal/goncho` |
| P2 | Skill metadata placement | Space Agent skill metadata patterns, Hermes skills | `internal/skills` |
| P2 | Durable write queue | Engram write queue | `internal/goncho`, `internal/subagent` |

## Not Recommended For Port

| Feature | Source | Why |
|---|---|---|
| Browser-first runtime | Space Agent | Different product model; Gormes remains a Go-native runtime. |
| WebLLM in-browser inference | Space Agent | Requires browser runtime and conflicts with the one-binary operator path. |
| Replacing all web tools with goscrapling at once | goscrapling | goscrapling should enter as a narrow extraction adapter first; search routing, browser actions, approval policy, and channel rendering remain Gormes concerns. |
| Full TUI gateway server at once | Hermes | Too large; port incrementally through visible contracts. |
| Kubernetes ACP controllers | Go references | Use local state machines first. |
| Telegram org model | Mercury | Gormes gateway channels use a different session/routing abstraction. |

## Success Metrics

| Horizon | Metric | Target |
|---|---|---|
| 30 days | Permission hardening | Shell blocklist and filesystem scopes shipped |
| 30 days | Provider parity | Core providers and failure taxonomy aligned |
| 30 days | Browser diagnostics | Doctor covers browser runtime readiness |
| 90 days | Loop detection | 5-type detector with replay fixtures |
| 90 days | Memory | Typed memory/provenance fixtures |
| 90 days | Skills | Metadata placement and resolver evals |
| 6 months | Context compression | Reconciled with current Hermes behavior |
| 6 months | Durable jobs | Queue health, replay/inbox, and supervisor evidence |
| 12 months | Dashboard | Session, skills, and gateway visibility |

## Next Actions

1. Use `gormes-planner` for one subsystem at a time.
2. Use `gormes-parity-auditor` to verify Hermes/Honcho behavior before adding rows.
3. Use `gormes-references` only when a Go implementation shape is unclear.
