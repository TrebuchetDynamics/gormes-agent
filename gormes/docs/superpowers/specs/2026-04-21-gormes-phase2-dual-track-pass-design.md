# Gormes Phase 2 Dual-Track Pass Design Spec

**Date:** 2026-04-21
**Author:** Xel (via Codex brainstorm)
**Status:** Approved design direction; pending implementation plan
**Scope:** One implementation pass that advances two independent but adjacent Phase 2 tracks: `2.E subagent runtime core` and `2.B.2 gateway chassis + second channel`.

## Related Documents

- [Phase 2.E — Subagent System Spec](2026-04-20-gormes-phase2e-subagent-design.md)
- [Phase 2.B.2 — Gateway Chassis + Discord Design Spec](2026-04-21-gormes-phase2b2-chassis-design.md)
- [Phase 2.G — Skills System Spec](2026-04-20-gormes-phase2g-skills-design.md)
- [Phase 2 — The Gateway](../../content/building-gormes/architecture_plan/phase-2-gateway.md)

---

## 1. Purpose

This pass is not "do all of Phase 2." It is a controlled execution pass that lands one vertical slice in each of the two most important remaining Phase 2 fronts:

- a **usable, deterministic subagent runtime core** under `internal/subagent/`
- a **shared gateway chassis** under `internal/gateway/` plus one real second adapter built on top of it

The point of doing both in one pass is strategic, not aesthetic.

Phase 2.E matters because Gormes needs a real execution-isolation primitive, not a vague future hook. Without a working `delegate_task`, cancellation model, and lifecycle manager, future delegation, reviewed promotion, and skills all sit on an unstable base.

Phase 2.B.2 matters because Telegram alone is not a reusable gateway architecture. Gormes needs one shared adapter scaffold plus a second real channel to prove that Telegram's mechanics were generalized honestly rather than wrapped in Telegram-specific abstractions.

This pass therefore lands:

1. a subagent runtime core that is callable today
2. a gateway chassis that owns shared channel mechanics
3. a real second adapter, chosen as **Discord**
4. tests and migration notes that make the next channels cheaper

Discord is chosen because Slack credentials are not available in this environment. Slack remains a follow-up consumer of the same chassis.

---

## 2. Locked Scope and Non-Goals

### 2.1 In scope

**Track A — Phase 2.E runtime core**

- stabilize and finish the current `internal/subagent/` lifecycle surface
- keep `Manager`, `Registry`, `Runner`, `Subagent`, `SubagentResult`, and `delegate_task` as the public runtime seam
- wire `[delegation]` config into the binary
- ensure deterministic cancellation, timeout, depth-limit enforcement, and batch concurrency
- verify the tool can run and return a structured result through the existing tool surface

**Track B — Phase 2.B.2 chassis + second channel**

- create `internal/gateway/` as the shared chassis
- move shared Telegram runtime mechanics into the chassis
- refactor Telegram to become a chassis consumer rather than the owner of special-case runtime logic
- add Discord as the second real channel built on the same chassis
- add a `gormes gateway` entrypoint that wires any configured channels into the same manager
- preserve existing `gormes telegram` behavior through a Telegram-only shared-manager path

**Cross-cutting deliverables**

- adapter contract tests
- at least one smoke end-to-end path for the gateway chassis
- migration notes for future channels
- full verification with focused tests and `go test ./... -race`

### 2.2 Explicitly out of scope

- a full LLM-powered child loop beyond the current runtime seam if the existing runner surface is still intentionally stubbed or deferred
- Phase 2.F hooks/lifecycle port
- Phase 2.G reviewed promotion or dynamic skill learning
- Slack, WhatsApp, Signal, Email, SMS, or any third channel
- Discord slash commands, embeds, threads, voice, or multi-guild routing
- multi-chat concurrent turn ownership inside the kernel
- replacing kernel architecture with a gateway-centric model

This pass is deliberately a **bounded dual-track ship**, not a Phase 2 mega-merge.

---

## 3. Approved Decisions

### 3.1 Two tracks, one pass, strict boundaries

The subagent runtime and the gateway chassis are implemented in the same pass, but they do not depend on each other at the package layer.

- `internal/subagent` stays independent of `internal/gateway`
- `internal/gateway` stays independent of `internal/subagent`
- any wiring happens in `cmd/gormes/`, not by coupling the two subsystems directly

This keeps the pass large in outcome but still decomposable in code and testing.

### 3.2 Discord is the second channel

Slack would have been acceptable only if credentials were already ready in this environment. They are not, so Discord is locked as the second channel for this pass.

### 3.3 The gateway owns shared transport mechanics

The chassis is responsible for:

- normalized inbound events
- outbound routing from kernel frames to the correct channel
- coalescing and placeholder-edit behavior
- auth gating and command normalization
- session-map persistence keyed by `<platform>:<chat_id>`

Adapters are responsible for:

- SDK initialization
- transport-edge event translation
- send/edit/reaction/typing primitives
- platform-specific limits and quirks

### 3.4 Capabilities are optional

The shared contract is intentionally split into a minimum base interface plus optional capability interfaces. If a channel does not support edit, typing, or reactions, the manager degrades to basic send rather than forcing one giant interface.

### 3.5 Telegram is no longer special

Telegram stops owning the only gateway runtime path. It becomes one adapter implementation of the chassis. The goal is zero user-visible regression with a more reusable internal shape.

### 3.6 Existing subagent runtime is consolidated, not replaced

This pass starts from the existing `internal/subagent/` code already present in the tree. The work is to finish and harden that runtime core, not to throw it away and redesign it from zero.

---

## 4. Architecture

### 4.1 Track A — subagent runtime core

`internal/subagent/` remains the single home of child execution lifecycle.

The owner boundaries are:

- `types.go` and `subagent.go` own data structures and blocking APIs such as `Events()` and `WaitForResult()`
- `runner.go` owns the inner-loop contract and current runtime implementation
- `manager.go` owns defaults, context composition, spawn, interrupt, collect, close, and result publication order
- `registry.go` owns process-wide tracking of live children
- `delegate_tool.go` owns the tool bridge into the runtime

The critical invariant is unchanged:

- the manager owns channel lifecycle and result publication
- the runner never closes channels it does not own
- cancellation flows through `context.Context`, not ad hoc flags

### 4.2 Track B — gateway chassis

`internal/gateway/` becomes the shared runtime that today is embedded inside the Telegram path.

Its minimum surfaces are:

- a normalized event model for inbound messages
- a base `Channel` contract for all adapters
- capability interfaces for edit, typing, reactions, and placeholders
- a manager that consumes inbound events and kernel render frames
- coalescing and rendering helpers moved out of Telegram

`internal/channels/telegram/` and `internal/channels/discord/` become thin SDK-facing packages on top of that chassis.

### 4.3 Wiring

`cmd/gormes/gateway.go` becomes the shared multi-channel entrypoint.

`cmd/gormes/telegram.go` remains available, but it should internally route through the shared manager with only Telegram enabled. That preserves existing operational commands and systemd units while reducing long-term duplication.

### 4.4 Kernel contract

The kernel remains the single owner of turn state, phase transitions, tool execution, and memory integration.

The gateway chassis submits normalized `kernel.PlatformEvent`s and consumes `kernel.Render()` frames. No new gateway-specific kernel abstraction is introduced in this pass.

---

## 5. Data Flow

### 5.1 Gateway inbound path

1. Adapter SDK event arrives.
2. Adapter translates it into a normalized `gateway.InboundEvent`.
3. `gateway.Manager` validates chat authorization and command semantics.
4. Manager maps it to the correct `kernel.PlatformEvent`.
5. Manager pins the active turn origin as `<platform, chat_id, inbound message id>`.

### 5.2 Gateway outbound path

1. Manager consumes `kernel.Render()` frames.
2. It resolves the adapter owning the active turn origin.
3. It uses shared render helpers and coalescing logic to compute the visible outbound text.
4. It uses optional adapter capabilities when available:
   - placeholder send
   - message edit
   - reaction ack/undo
   - typing start/stop
5. It persists changed session IDs through `internal/session.Map`.

### 5.3 Subagent runtime path

1. `delegate_task` receives JSON tool args.
2. It constructs a `SubagentConfig` and calls `Manager.Spawn`.
3. Manager creates a child context, registers the child, and starts the runner.
4. Runner emits `SubagentEvent`s and returns `SubagentResult`.
5. Manager forwards events, fixes result ordering, closes `done`, and unregisters the child.
6. Tool returns a structured result payload to the caller.

The two paths remain operationally adjacent but structurally independent.

---

## 6. Error Handling

### 6.1 Gateway

- unauthorized chats are rejected before kernel submission
- adapter capability absence is not an error; the manager falls back to plain send
- edit/placeholder failures degrade to basic outbound send where practical
- adapter SDK errors are logged with channel identity and do not automatically crash unrelated channels

### 6.2 Subagents

- parent cancellation propagates to every child through context
- interrupt is explicit and recorded in result status / exit reason
- timeout is explicit and recorded in result status / exit reason
- unknown child interrupts return a real error, not a silent no-op

In both tracks, failures should be visible, typed, and testable.

---

## 7. Testing Strategy

### 7.1 Subagent runtime tests

- spawn happy path
- default iteration and timeout application
- interrupt flow
- parent cancellation propagation
- depth limit enforcement
- batch concurrency behavior
- registry tracking correctness
- delegate tool success and validation paths

### 7.2 Gateway chassis tests

- contract tests against a `fakeChannel`
- shared manager tests for inbound normalization and outbound routing
- coalescer tests after extraction
- render helper tests after extraction

### 7.3 Adapter tests

- Telegram regression tests updated to run through the shared chassis
- Discord adapter tests with SDK seam / mocks
- config validation tests for Discord wiring

### 7.4 Smoke tests

At least one end-to-end smoke path should validate:

`InboundEvent -> kernel.PlatformEvent -> RenderFrame -> visible outbound send/edit`

The pass is not done until the focused tests are green and the broad run `go test ./... -race` is green in the worktree.

---

## 8. Migration Notes for Future Channels

The chassis created in this pass is the template for the rest of `2.B.2+`.

Every future channel should:

1. implement the base channel contract
2. implement only the optional capabilities it truly supports
3. declare platform-specific limits in one place
4. translate SDK-native inbound events into `gateway.InboundEvent`
5. reuse shared auth, session-map, render, and coalescing behavior

The next-channel checklist becomes:

- add config block
- add adapter package
- add contract tests
- add one smoke path
- add migration notes or docs entry

Shared logic should change only when a new platform reveals a genuinely reusable capability boundary, not because a single SDK wants a custom shortcut.

---

## 9. Success Criteria

This pass is successful when all of the following are true:

- `delegate_task` is wired and returns structured results through the subagent runtime core
- the current `internal/subagent/` lifecycle surface is deterministic and race-clean
- `internal/gateway/` exists and owns shared adapter mechanics
- Telegram is running through the chassis, not bespoke runtime code
- Discord works end-to-end as the second real adapter
- adapter contract tests and smoke tests are present
- future adapters have explicit migration notes
- broad verification passes with `go test ./... -race`

This is the smallest pass that materially advances both of the highest-value remaining Phase 2 fronts without turning into an uncontrolled merge.
