# Understand-Anything Refactor Plan

This deterministic plan is generated from an existing Understand-Anything knowledge graph plus live repo checks when the command runs inside a checkout. It does not run an LLM. Use the follow-up prompt when you want model reasoning over this file.

## Inputs

- **Project:** navivox
- **Focus:** whole graph
- **Graph source:** `.understand-anything/knowledge-graph.json`
- **This file:** `refactor-plan-understand-refactor.md`
- **Analyzed at:** 2026-05-26T20:06:22.105Z
- **Git commit:** 4155335cc4916178a016327af28983fbb0925ee1
- **Live repo evidence:** collected from current files/tests
- **Previous plan:** read from existing output before regeneration

## Previous plan continuity

- Previous refactor plan was read before regenerating; use it as continuity context, not source of truth.
- Previous top recommendation: Start with **channel_test.go** at `channel_test.go`. It has combined score 139, complex graph complexity, 58 graph relationships, and confidence **strong**. ### Live file/test evidence - Live file confirmed: `channel_test.go` (1289 non-empty LOC, 334 branch points, 1 imports, 0 public-surface hints). Related tests to inspect or create: - `channel_test.go` ### Relationship evidence - **channel_test.go** --contains→ **TestNavivoxStatusRequiresAuthAndHealthzIsPublic** - **channel_test.go** --contains→ **TestNavivoxProfileSeedEndpointCreatesDraftAndApplyShowsContact** - **channel_test.go** --contains→ **TestNavivoxProfileRoutingEndpointIsAuthBoundedAndSecretFree** - **channel_test.go** --contai…
- Previous slice outline: 1. **Characterize the seam** — read the candidate, graph relationships, live file stats, callers, and related tests; confirm the graph is current against live files. 2. **Pre-refactor bug search** — inspect existing behavior, TODO/FIXME/error paths, callers, and current tests for likely bugs before changing code; turn confirmed bugs into focused regression tests or explicit bug notes. 3. **Add or tighten behavior tests** — lock observable behavior through the public interface before moving code; create a focused test if no related test was found. 4. **Deepen the module** — move repeated orchestration or branching behind one smaller interface; avoid new pass-through wrappers. 5. **During-ref…
- Previous notes/decisions detected:
  - - Previous slice outline: 1. **Characterize the seam** — read the candidate, graph relationships, live file stats, callers, and related tests; confirm the graph is current against…
  - - 2. **Pre-refactor bug search** — inspect existing behavior, TODO/FIXME/error paths, callers, and current tests for likely bugs before changing code; turn confirmed bugs into foc…
  - - - **Before refactor:** establish a baseline, inspect known-risk branches and error paths, search nearby TODO/FIXME notes, and document any suspected existing bugs separately fro…
  - 2. **Pre-refactor bug search** — inspect existing behavior, TODO/FIXME/error paths, callers, and current tests for likely bugs before changing code; turn confirmed bugs into focus…
  - - **Before refactor:** establish a baseline, inspect known-risk branches and error paths, search nearby TODO/FIXME notes, and document any suspected existing bugs separately from…

## Likely tangled hotspots

| Candidate | Type | Score | Confidence | Evidence |
| --- | --- | ---: | --- | --- |
| channel_test.go | file | 139 | strong | complex graph complexity, 58 relationships, `channel_test.go`; 1295 live LOC, 336 branches, 1 imports, 1 related tests |
| channel.go | file | 133 | strong | complex graph complexity, 55 relationships, `channel.go`; 577 live LOC, 72 branches, 1 imports, 1 related tests |
| profile_contacts.go | file | 77 | strong | complex graph complexity, 27 relationships, `profile_contacts.go`; 441 live LOC, 89 branches, 1 imports, 1 related tests |
| stream.go | file | 77 | needs-review | complex graph complexity, 27 relationships, `stream.go`; 355 live LOC, 66 branches, 1 imports, 0 related tests |
| config_admin.go | file | 67 | strong | complex graph complexity, 22 relationships, `config_admin.go`; 492 live LOC, 105 branches, 1 imports, 1 related tests |
| capabilities_test.go | file | 43 | strong | complex graph complexity, 10 relationships, `capabilities_test.go`; 286 live LOC, 72 branches, 1 imports, 1 related tests |
| config_admin_test.go | file | 43 | strong | complex graph complexity, 10 relationships, `config_admin_test.go`; 370 live LOC, 83 branches, 1 imports, 1 related tests |
| capabilities.go | file | 40 | strong | complex graph complexity, 9 relationships, `capabilities.go`; 214 live LOC, 8 branches, 1 imports, 1 related tests |
| auth.go | file | 39 | needs-review | moderate graph complexity, 10 relationships, `auth.go`; 125 live LOC, 29 branches, 1 imports, 0 related tests |
| turn.go | file | 39 | needs-review | moderate graph complexity, 10 relationships, `turn.go`; 118 live LOC, 10 branches, 1 imports, 0 related tests |
| voice_profiles.go | file | 39 | strong | moderate graph complexity, 10 relationships, `voice_profiles.go`; 194 live LOC, 48 branches, 1 imports, 1 related tests |
| voice_profiles_test.go | file | 37 | strong | complex graph complexity, 7 relationships, `voice_profiles_test.go`; 222 live LOC, 44 branches, 1 imports, 1 related tests |

## Top recommendation

Start with **channel_test.go** at `channel_test.go`. It has combined score 139, complex graph complexity, 58 graph relationships, and confidence **strong**.

### Live file/test evidence

- Live file confirmed: `channel_test.go` (1295 non-empty LOC, 336 branch points, 1 imports, 0 public-surface hints).

Related tests to inspect or create:

- `channel_test.go`

### Relationship evidence

- **channel_test.go** --contains→ **TestNavivoxStatusRequiresAuthAndHealthzIsPublic**
- **channel_test.go** --contains→ **TestNavivoxProfileSeedEndpointCreatesDraftAndApplyShowsContact**
- **channel_test.go** --contains→ **TestNavivoxProfileRoutingEndpointIsAuthBoundedAndSecretFree**
- **channel_test.go** --contains→ **TestNavivoxStatusIncludesServerScopedProfileRoutingWithoutDefaultProfile**
- **channel_test.go** --contains→ **TestNavivoxStatusIncludesSetupHandoffForAppContinuation**
- **channel_test.go** --contains→ **TestNavivoxHTTPStartTurnRequiresAuthAndEnqueuesTypedGatewayEvent**
- **channel_test.go** --contains→ **TestNavivoxWebSocketAuthAcceptsBrowserSubprotocolToken**
- **channel_test.go** --contains→ **TestNavivoxLayeredAuthRequiresTokenAndAllowedTailscaleIdentity**

## Refactor slices

1. **Characterize the seam** — read the candidate, graph relationships, live file stats, callers, and related tests; confirm the graph is current against live files.
2. **Pre-refactor bug search** — inspect existing behavior, TODO/FIXME/error paths, callers, and current tests for likely bugs before changing code; turn confirmed bugs into focused regression tests or explicit bug notes.
3. **Add or tighten behavior tests** — lock observable behavior through the public interface before moving code; create a focused test if no related test was found.
4. **Deepen the module** — move repeated orchestration or branching behind one smaller interface; avoid new pass-through wrappers.
5. **During-refactor bug search** — after each small move, compare behavior against the baseline, run focused validation, and stop to diagnose any new or suspicious failure before continuing.
6. **Delete replaced shallow paths** — remove tests or modules that only exercise implementation details after the deeper interface is covered.
7. **Post-refactor bug search** — rerun baseline and focused validation, inspect the diff for accidental behavior changes, and add regressions for any bug found before broad validation.

## Bug search checkpoints

- **Before refactor:** establish a baseline, inspect known-risk branches and error paths, search nearby TODO/FIXME notes, and document any suspected existing bugs separately from refactor intent.
- **During refactor:** keep changes small, run focused validation after each slice, compare outputs against baseline behavior, and diagnose suspicious failures immediately instead of batching them.
- **After refactor:** rerun the baseline plus focused/broad validation, review public behavior changes in the diff, and create regression tests or bug notes for anything discovered.

## Architecture layers to inspect

- **Navivox Channel Runtime** (4 nodes): Gateway.Channel implementation, auth gates, HTTP route dispatch, WebSocket stream sessions, and text/voice turn enqueue mechanics.
- **Operator App Surfaces** (5 nodes): App-facing capability, profile-contact, config-admin, memory overview, and voice-profile read models with redaction and validation boundaries.
- **Contract Test Coverage** (7 nodes): Package tests that preserve Navivox auth, HTTP, WebSocket, profile, config, memory, voice, tool-event, and runtime safety contracts.
- **Architecture Documentation** (1 nodes): Repo-local codemap explaining Navivox responsibilities, module seams, data flows, integrations, and test surfaces.

## Follow-up LLM prompt

```text
/understand-chat Use refactor-plan-understand-refactor.md, the previous-plan continuity section, the current Understand graph, and the live file/test evidence to grill the selected refactor candidate: <candidate>. Stress-test domain terms, bug risks before/during/after the refactor, tests, and small validation-backed slices before editing code. Focus: whole graph.
```
