# Understand-Anything Refactor Plan

This deterministic plan is generated from an existing Understand-Anything knowledge graph plus live repo checks when the command runs inside a checkout. It does not run an LLM. Use the follow-up prompt when you want model reasoning over this file.

## Inputs

- **Project:** gormes-agent
- **Focus:** @internal/channels/telegram/
- **Graph source:** `.understand-anything/knowledge-graph.json`
- **This file:** `refactor-plan-understand-refactor.md`
- **Analyzed at:** 2026-05-26T19:51:43.341Z
- **Git commit:** 4155335cc4916178a016327af28983fbb0925ee1
- **Live repo evidence:** collected from current files/tests
- **Previous plan:** none found at output path

## Previous plan continuity

- No previous refactor plan found at the output path.

## Likely tangled hotspots

| Candidate | Type | Score | Confidence | Evidence |
| --- | --- | ---: | --- | --- |
| main.go | file | 2339 | strong | complex graph complexity, 1158 relationships, `cmd/gormes/main.go`; 1471 live LOC, 299 branches, 1 imports, 2 related tests |
| event_bus_integration_test.go | file | 2313 | needs-review | moderate graph complexity, 1147 relationships, `internal/gateway/event_bus_integration_test.go`; 220 live LOC, 39 branches, 1 imports, 0 related tests |
| gateway_test.go | file | 2297 | strong | complex graph complexity, 1137 relationships, `cmd/gormes/gateway_test.go`; 844 live LOC, 127 branches, 1 imports, 5 related tests |
| gateway.go | file | 2125 | strong | complex graph complexity, 1051 relationships, `cmd/gormes/gateway.go`; 1331 live LOC, 242 branches, 1 imports, 8 related tests |
| manager_multi_agent_isolation_test.go | file | 2071 | needs-review | complex graph complexity, 1024 relationships, `internal/gateway/manager_multi_agent_isolation_test.go`; 311 live LOC, 39 branches, 1 imports, 0 related tests |
| goncho_turn_integration_test.go | file | 1997 | needs-review | complex graph complexity, 987 relationships, `internal/kernel/goncho_turn_integration_test.go`; 267 live LOC, 50 branches, 1 imports, 0 related tests |
| manager.go | file | 1997 | needs-review | complex graph complexity, 987 relationships, `internal/gateway/manager.go`; 2789 live LOC, 541 branches, 1 imports, 0 related tests |
| setup.go | file | 1993 | strong | complex graph complexity, 985 relationships, `cmd/gormes/setup.go`; 3462 live LOC, 797 branches, 1 imports, 8 related tests |
| doctor.go | file | 1875 | strong | complex graph complexity, 926 relationships, `cmd/gormes/doctor.go`; 1042 live LOC, 169 branches, 1 imports, 8 related tests |
| tool_interrupt_test.go | file | 1815 | needs-review | moderate graph complexity, 898 relationships, `internal/kernel/tool_interrupt_test.go`; 181 live LOC, 19 branches, 1 imports, 0 related tests |
| auto_tts_test.go | file | 1807 | needs-review | moderate graph complexity, 894 relationships, `internal/gateway/auto_tts_test.go`; 140 live LOC, 20 branches, 1 imports, 0 related tests |
| gateway_mutating_unavailable_test.go | file | 1801 | strong | complex graph complexity, 889 relationships, `cmd/gormes/gateway_mutating_unavailable_test.go`; 1217 live LOC, 168 branches, 1 imports, 1 related tests |

## Top recommendation

Start with **main.go** at `cmd/gormes/main.go`. It has combined score 2339, complex graph complexity, 1158 graph relationships, and confidence **strong**.

### Live file/test evidence

- Live file confirmed: `cmd/gormes/main.go` (1471 non-empty LOC, 299 branch points, 1 imports, 0 public-surface hints).

Related tests to inspect or create:

- `cmd/progress/main_test.go`
- `cmd/gormes-repo/main_test.go`

### Relationship evidence

- **main.go** --contains→ **main**
- **main.go** --contains→ **sanitizeTermuxExecArgs**
- **main.go** --contains→ **sanitizeTermuxExecArgsWithExe**
- **main.go** --contains→ **termuxExecArgMatchesExecutable**
- **main.go** --contains→ **executeRootCommand**
- **main.go** --contains→ **removedRootFlagSuggestion**
- **main.go** --contains→ **newRootCommandWithRuntime**
- **main.go** --contains→ **installRootRPCModeFlags**

## Refactor slices

1. **Characterize the seam** — read the candidate, graph relationships, live file stats, callers, and related tests; confirm the graph is current against live files.
2. **Add or tighten behavior tests** — lock observable behavior through the public interface before moving code; create a focused test if no related test was found.
3. **Deepen the module** — move repeated orchestration or branching behind one smaller interface; avoid new pass-through wrappers.
4. **Delete replaced shallow paths** — remove tests or modules that only exercise implementation details after the deeper interface is covered.
5. **Run focused validation** — run the smallest relevant test command, then broader validation if the seam crosses modules.

## Architecture layers to inspect

- **Tests and Fixtures** (1442 nodes): Go tests, web tests, fixtures, testdata, and validation assets that protect Gormes behavior.
- **CLI Entrypoints** (121 nodes): Cobra commands and binary entrypoints under cmd/ that expose Gormes to operators.
- **Gateway and Channels** (195 nodes): Gateway orchestration, channel adapters, Telegram/Discord/Slack style ingress, and channel-facing runtime glue.
- **Provider and Agent Runtime** (119 nodes): Provider integrations, Hermes wire/client behavior, kernel turn processing, agent execution, and runtime coordination.
- **Tools and Skills** (168 nodes): Built-in tools, skill loading/runtime, browser helpers, MCP, and repo-local development skills.
- **Memory Sessions and Goncho** (46 nodes): Session persistence, stores, local memory, Goncho recall, audit logs, and durable state mechanics.
- **TUI and Operator UI** (72 nodes): Bubble Tea TUI screens, admin UI, status rendering, and interactive operator flows.
- **Progress Planning System** (225 nodes): Progress schema, progress CLI, architecture plan data, generated readiness surfaces, and roadmap documentation.
- **Web Docs and Landing** (721 nodes): Astro/Hugo docs, landing pages, public website data, and public messaging assets.
- **Install Build and CI** (89 nodes): Installers, scripts, GitHub Actions, module manifests, release configs, and build automation.
- **Public API Facade** (1 nodes): Public pkg/ re-export surface for consumers embedding or importing Gormes.
- **Internal Support Packages** (272 nodes): Internal support packages that do not belong to the main gateway, provider, TUI, memory, tool, or progress lanes.
- **Repository Metadata** (29 nodes): Top-level repository metadata, benchmarks, examples, and miscellaneous project assets.

## Follow-up LLM prompt

```text
/understand-chat Use refactor-plan-understand-refactor.md, the previous-plan continuity section, the current Understand graph, and the live file/test evidence to grill the selected refactor candidate: <candidate>. Stress-test domain terms, tests, risks, and small validation-backed slices before editing code. Focus: @internal/channels/telegram/.
```
