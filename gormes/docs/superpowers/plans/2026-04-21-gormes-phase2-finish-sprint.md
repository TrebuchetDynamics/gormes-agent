# Gormes Phase 2 Finish Sprint

> Execution: strict TDD (`RED -> GREEN -> REFACTOR`), atomic commits, no stacking on red suite.

**Goal:** Finish all remaining Phase 2 gateway/runtime items in roadmap order and leave Phase 2 fully closed in docs and tests.

## Verified baseline

From `docs/content/building-gormes/architecture_plan/progress.json`:

- Phase 2 status: **33 complete / 60 total**.
- Remaining (non-complete): **27** items.
- Already shipped: shared gateway chassis + Telegram + Discord + Slack, Phase 2.E.0 runtime core, and most of skills runtime.

## Remaining queue (from roadmap)

- **2.B.4 (P1):** WhatsApp adapter (3 items)
- **2.B.5 (P1):** Session Context + Delivery Routing (4 items)
- **2.B.6 (P2):** Signal adapter (2 items)
- **2.B.7 (P3):** Email + SMS adapters (2 items)
- **2.B.8 (P4):** Matrix + Mattermost adapters (2 items)
- **2.B.9 (P4):** Webhook + trigger ingress (2 items)
- **2.B.10 (P4):** Regional/device adapter flood (3 items)
- **2.E.1 (P0):** Real child Hermes stream loop (1 item)
- **2.F.1 (P1):** Slash command registry + gateway dispatch (2 items)
- **2.F.2 (P2):** Built-in BOOT.md startup hook (1 item)
- **2.F.3 (P2):** Restart/pairing/status (2 items)
- **2.F.4 (P3):** Home channel + operator surfaces (3 items)

---

## Phase 2 Definition of Done

1. Every Phase 2 item in `progress.json` is `complete`.
2. `go test ./... -count=1` passes twice in a row.
3. `go run ./cmd/progress-gen --validate` passes.
4. `go test ./docs -count=1` passes and phase-2 docs match implementation.

---

## Execution order (strict)

1. **P2-A** Finish 2.E.1 (`Real child Hermes stream loop`) — unblock runtime realism first.
2. **P2-B** Finish 2.F.1 + 2.F.2 (commands + BOOT hook) — shared control plane.
3. **P2-C** Finish 2.F.3 + 2.F.4 (lifecycle + home-channel operator surfaces).
4. **P2-D** Finish 2.B.5 (session context + delivery router) before more adapters.
5. **P2-E** Finish adapter waves: 2.B.4, 2.B.6, 2.B.7.
6. **P2-F** Finish long-tail adapter/ingress waves: 2.B.8, 2.B.9, 2.B.10.
7. **P2-G** Docs/ledger closeout.

---

## Slice P2-A — 2.E.1 real child stream loop

**Targets**
- `2.E.1` Real child Hermes stream loop

**Files (expected)**
- `internal/subagent/runner.go`
- `internal/subagent/manager.go`
- `internal/subagent/*_test.go`
- `internal/hermes/*` (stream seam)

**Verify**
```bash
go test ./internal/subagent ./internal/hermes ./internal/kernel -count=1 -race
```

**Commit**
`feat(subagent): replace stub child loop with real hermes stream execution`

---

## Slice P2-B — 2.F.1 + 2.F.2 command/hook closeout

**Targets**
- Canonical `CommandDef` registry
- Gateway slash dispatch + per-platform exposure
- Built-in `BOOT.md` startup hook

**Files**
- `internal/gateway/*`
- `cmd/gormes/gateway.go`
- `cmd/gormes/doctor.go`
- `internal/config/*`

**Verify**
```bash
go test ./internal/gateway ./cmd/gormes ./internal/config -count=1
```

**Commit**
`feat(gateway): finalize command registry and boot hook surfaces`

---

## Slice P2-C — 2.F.3 + 2.F.4 lifecycle/operator surfaces

**Targets**
- Graceful restart drain + managed shutdown
- Pairing state + status surfaces
- Home channel ownership + notify routing
- Channel/contact directory
- Mirror + sticker cache surfaces

**Files**
- `internal/gateway/*`
- `internal/session/*`
- `cmd/gormes/*`

**Verify**
```bash
go test ./internal/gateway ./internal/session ./cmd/gormes -count=1
```

**Commit**
`feat(gateway): finalize lifecycle and home-channel operator surfaces`

---

## Slice P2-D — 2.B.5 session context + delivery routing

**Targets**
- SessionSource parity store
- SessionContext prompt injection
- DeliveryRouter and `--deliver` parsing
- Gateway stream consumer event fan-out

**Files**
- `internal/gateway/*`
- `internal/session/*`
- `internal/kernel/*`
- `cmd/gormes/gateway.go`

**Verify**
```bash
go test ./internal/gateway ./internal/session ./internal/kernel ./cmd/gormes -count=1
```

**Commit**
`feat(gateway): complete session-context and delivery-routing stack`

---

## Slice P2-E — adapter wave 1 (P1–P3)

**Targets**
- `2.B.4` WhatsApp adapter (3 items)
- `2.B.6` Signal adapter (2 items)
- `2.B.7` Email + SMS adapters (2 items)

**Files**
- `internal/channels/whatsapp/*`
- `internal/channels/signal/*`
- `internal/channels/email/*`
- `internal/channels/sms/*`
- `internal/config/*`
- `cmd/gormes/gateway.go`

**Verify**
```bash
go test ./internal/channels/... ./internal/gateway ./internal/config ./cmd/gormes -count=1
```

**Commit strategy**
- one adapter family per commit:
  - `feat(channel): add whatsapp adapter on shared gateway contract`
  - `feat(channel): add signal adapter on shared gateway contract`
  - `feat(channel): add email/sms adapters on shared gateway contract`

---

## Slice P2-F — adapter wave 2 + webhook ingress (P4)

**Targets**
- `2.B.8` Matrix + Mattermost
- `2.B.9` Signed webhook ingress + prompt-to-delivery bridge
- `2.B.10` BlueBubbles/HomeAssistant, Feishu/WeChat/WeCom, DingTalk/QQ

**Verify**
```bash
go test ./internal/channels/... ./internal/gateway ./internal/config ./cmd/gormes -count=1
```

**Commit strategy**
- one adapter/ingress family per commit

---

## Slice P2-G — docs and ledger closeout

**Files**
- `docs/content/building-gormes/architecture_plan/progress.json`
- `docs/content/building-gormes/architecture_plan/phase-2-gateway.md`
- `docs/content/building-gormes/architecture_plan/_index.md`

**Verify**
```bash
go run ./cmd/progress-gen --write
go run ./cmd/progress-gen --validate
go test ./docs -count=1
go test ./... -count=1
go test ./... -count=1
```

**Commit**
`docs(phase2): finalize gateway/runtime phase closeout`

---

## Global guardrail

After every slice:

```bash
go test ./... -count=1
```

If red, land immediate fix commit:

`fix(regression): phase2 <short-description>`

No new feature commits until green.