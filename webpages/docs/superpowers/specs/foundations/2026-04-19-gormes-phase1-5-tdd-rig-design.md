# Gormes — Phase 1.5 TDD Rig Design Spec

**Date:** 2026-04-19
**Author:** Xel (via Claude Code brainstorm)
**Status:** Approved — ready for plan
**Scope:** Three test / tooling artifacts plus one one-line enum addition, planting "red-test beacons" for the Phase 1.5 resilience feature (Route-B reconnect) and proving the Phase 1 kernel's replace-latest mailbox invariant.

---

## 1. Purpose

Phase 1 shipped a green-tested vertical slice. Three follow-ups from the Phase-1 completion report remain and all three resist "just dispatch a subagent" because they touch questions of correctness under failure. This spec installs the **test-first scaffolding** so those questions are answered by passing or failing code, not by human judgment:

1. A portability script that makes the **Go 1.22 vs 1.24** decision mechanical — runs `make build` inside a `golang:1.22-alpine` container and, on failure, names the exact package/symbol forcing the drift.
2. A red test case in `internal/kernel/` that proves the current kernel **does not** reconnect after a mid-stream TCP drop — and names the required Route-B semantics from spec §9.2 in its assertions. Implementation is explicitly out of scope for this spec; a future plan flips the test green by shipping the reconnect feature.
3. A green test case in `internal/kernel/` that proves the Phase-1 capacity-1 replace-latest render mailbox **actually works as designed** under a maliciously-stalled consumer.

No production kernel behaviour changes in this spec. The only production-code edit is one new enum value (`PhaseReconnecting`) in `kernel/frame.go` so the red test can reference it by name.

---

## 2. Locked Architectural Decisions

| Decision | Value | Rationale |
|---|---|---|
| Reconnect semantics | **Route B** — client-side restart with visual continuity | Only feasible choice against stateless OpenAI-compatible servers; C ("continue") is hallucination bait, A (server resume) is fantasy, D (no-retry) is status quo |
| Red-test delivery | **`t.Skip` in main branch** | Green is sacred in CI; skipped tests are Technical-Debt Beacons with one-line flip to un-skip |
| Scaffolding policy | **Minimal** — one enum value in `frame.go` | Production vocabulary exists; behaviour does not. Future Route-B implementation adds behaviour only |
| Stall-test observability | **Black-box via render channel** (not internal state read) | Treats kernel the way the TUI treats it; no test-only methods on production types |

---

## 3. Artifacts

### 3.1 `gormes/scripts/check-go1.22-compat.sh`

Standalone bash script. Exit codes are meaningful:

| Exit | Meaning |
|---|---|
| `0` | Build succeeded under Go 1.22 — Termux/LTS portability preserved |
| `1` | Build failed under Go 1.22 — script printed the offending package + the first symbol/API that the Go 1.22 toolchain refused |
| `2` | Neither Docker nor `go1.22` download-toolchain is available — script cannot decide |

**Mechanism preference order:**

1. **Docker path (preferred).** Mounts the repo read-only into `golang:1.22-alpine`, runs `go build ./cmd/gormes 2>&1 | tee /tmp/compat.log`. If exit != 0, greps the build log for `requires go1.23` / `requires go1.24` / `undefined:` lines and extracts the offending package path + symbol. Prints:
   ```
   FAIL: github.com/charmbracelet/bubbletea requires go1.24 (symbol: ...)
         github.com/charmbracelet/bubbles requires go1.23 (symbol: ...)
   ```
2. **Fallback path.** `go install golang.org/dl/go1.22.10@latest && go1.22.10 download && go1.22.10 build ./cmd/gormes`. Same log-parsing logic.
3. **Fail path.** If neither available, exit 2 with instructions on installing Docker **or** `golang.org/dl/go1.22.10`.

**Log-parsing specifics.**
- Lines matching `requires go\d\.\d+` → extract the full Go-version requirement.
- Lines matching `undefined: <symbol>` immediately following `could not import <pkg>` → pair them up.
- Standard error in Go's build output already prefixes lines with `<package path>: <error>`, which gives us the offending package for free.

Output example on failure:
```
=== Go 1.22 compatibility check ===
Using Docker (golang:1.22-alpine)
Build failed. Incompatible dependencies:

  github.com/charmbracelet/bubbletea  requires go1.24
  (symbol: iter.Seq used in tea/stream.go:42)

  github.com/charmbracelet/lipgloss   requires go1.23
  (symbol: slices.Chunk used in lipgloss/layout.go:118)

Decision data for "Portability vs. Progress":
  - Accept Go 1.24 floor  → 2 deps satisfied, 0 downgrades
  - Pin to Go 1.22        → need to downgrade bubbletea to <X>, lipgloss to <Y>
```

### 3.2 `PhaseReconnecting` enum value in `kernel/frame.go`

One-line change. Currently the `Phase` enum has six values; we add a seventh:

```go
const (
    PhaseIdle Phase = iota
    PhaseConnecting
    PhaseStreaming
    PhaseFinalizing
    PhaseCancelling
    PhaseFailed
    PhaseReconnecting   // Phase-1.5 Route-B resilience; no transitions to this state exist yet.
)
```

String table extended to `"Reconnecting"`. **No kernel code transitions into this state yet** — that's the Route-B feature, deferred. The value exists solely so the red test can assert against a named constant rather than a magic integer.

This is a TDD seed per §2: the production vocabulary leads the behaviour.

### 3.3 `internal/kernel/reconnect_test.go`

Red chaos test. Ships with `t.Skip` at the top of the function so it compiles and runs (as a skip) in `make test`.

**Test body (paraphrased):**

```go
func TestKernel_HandlesMidStreamNetworkDrop(t *testing.T) {
    t.Skip("RED TEST: Route B Resilience — Implementation pending (see §9.2 of " +
           "2026-04-18-gormes-frontend-adapter-design.md)")

    // 1. First server: emits 5 SSE tokens then hangs open (client will see
    //    the chaos-monkey disconnect, not a clean end).
    srv1 := httptest.NewServer(fiveTokenHandler())

    // 2. Build kernel with a REAL hermes.NewHTTPClient(srv1.URL, "").
    //    Not MockClient — this test specifically exercises the HTTP+SSE
    //    drop path.
    k, tm := newRealKernel(t, srv1.URL)

    // 3. Submit a turn. Consume render frames until 5 tokens visible.
    go k.Run(ctx)
    <-k.Render() // initial idle
    _ = k.Submit(kernel.PlatformEvent{Kind: PlatformEventSubmit, Text: "hi"})
    waitForDraftOfLength(t, k.Render(), 5, 2*time.Second)

    // 4. CHAOS MONKEY: abruptly close all active connections.
    srv1.CloseClientConnections()

    // 5. ASSERT 1: within 500ms the kernel transitions to PhaseReconnecting.
    //    CURRENTLY FAILS — kernel goes to PhaseFailed.
    got := waitForPhase(t, k.Render(), PhaseReconnecting, 500*time.Millisecond)
    if got.Phase != PhaseReconnecting {
        t.Fatalf("phase = %v, want PhaseReconnecting", got.Phase)
    }

    // 6. ASSERT 2: the draft still contains the 5 tokens (no data loss
    //    during reconnect window). CURRENTLY FAILS — kernel clears draft.
    if got.DraftText != "xxxxx" {
        t.Errorf("DraftText = %q, want xxxxx (5 tokens preserved)", got.DraftText)
    }

    // 7. Stand up a second server on the same URL (port rebind) emitting
    //    10 new tokens + Done. Simulates Python restart.
    srv2 := httptest.NewServer(tenTokenHandler())
    rebindTo(srv1, srv2.URL)

    // 8. ASSERT 3: kernel transitions back to PhaseStreaming then Idle.
    //    CURRENTLY FAILS — no auto-retry exists.
    waitForPhase(t, k.Render(), PhaseStreaming, 20*time.Second) // full backoff budget
    final := waitForPhase(t, k.Render(), PhaseIdle, 20*time.Second)

    // 9. ASSERT 4: final.History contains exactly ONE assistant message
    //    equal to the 10 new tokens — no Frankenstein hybrid, no appended
    //    old+new. CURRENTLY FAILS — no retry ever ran.
    assistant := lastAssistantMessage(final.History)
    if assistant.Content != "yyyyyyyyyy" {
        t.Errorf("final assistant = %q, want yyyyyyyyyy (fresh-retry content)", assistant.Content)
    }
}
```

**Why each assertion fails today (proving the red):**
- Current kernel's stream-error path transitions straight to `PhaseFailed` with `LastError` set; it never touches `PhaseReconnecting`.
- Current kernel clears `k.draft` only when a new valid turn is admitted; the error path leaves draft in place, but the phase is already wrong.
- No retry loop exists — `k.runTurn` returns on fatal error.
- No second turn ever starts, so `final.History` at assertion time is empty or contains only the user turn.

All four assertions will produce distinct failure signatures, giving the future Route-B implementer a precise TDD target.

**Helpers (`reconnect_test_helpers.go` or inline):**

- `fiveTokenHandler()` / `tenTokenHandler()` — standard `httptest.HandlerFunc`s that flush 5 / 10 SSE `data: {...}` frames with `content: "x"` / `content: "y"`.
- `waitForDraftOfLength` / `waitForPhase` — drain `k.Render()` until predicate matches or deadline.
- `rebindTo` — swaps a live `httptest.Server`'s URL so a new server listens where the old one was. Implementation note: this may require either `net.Listen` on a specific port (not the default `0`) or a proxy pattern. Simpler alternative for the test: make `newRealKernel` construct its client against a small **stable-URL test proxy** that forwards to whichever backend is currently registered.

### 3.4 `internal/kernel/stall_test.go`

Green kernel-discipline test. No `t.Skip`. Must pass against the current kernel today. Proves the replace-latest invariant.

**Test body:**

```go
func TestKernel_NonBlockingUnderTUIStall(t *testing.T) {
    k, mc := fixture(t)

    // Script 1000 tokens + Done.
    events := make([]hermes.Event, 0, 1001)
    for i := 0; i < 1000; i++ {
        events = append(events, hermes.Event{
            Kind: hermes.EventToken, Token: "t", TokensOut: i + 1,
        })
    }
    events = append(events, hermes.Event{Kind: hermes.EventDone, FinishReason: "stop"})
    mc.Script(events, "")

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    go k.Run(ctx)

    // INTENTIONAL STALL: do NOT consume k.Render() during the turn. Read
    // only the initial idle frame (so the kernel can start), then submit
    // and immediately stop reading.
    initial := <-k.Render()
    if initial.Phase != PhaseIdle {
        t.Fatalf("initial = %v, want idle", initial.Phase)
    }
    _ = k.Submit(kernel.PlatformEvent{Kind: PlatformEventSubmit, Text: "hi"})

    // Give the kernel 2 seconds to complete the turn without the render
    // channel being drained. If the kernel blocks on emit, it will not
    // finish in time.
    time.Sleep(2 * time.Second)

    // Now, as a TUI would after a stall, we start draining. The render
    // mailbox is capacity-1 replace-latest, so the ONE frame we observe
    // must be the LATEST state — which is either the PhaseIdle finalization
    // frame or a very-late streaming frame (both acceptable).
    // REPLACE-LATEST INVARIANT: the frame we see is NOT an old mid-stream
    // frame from 1.9s ago. Its draft must contain the full 1000 tokens or
    // it must be the completed Idle frame with History populated.
    select {
    case f := <-k.Render():
        ok := false
        if f.Phase == PhaseIdle && len(f.History) >= 2 {
            // Completed turn — Idle frame after finalization.
            assistant := lastAssistantMessage(f.History)
            if assistant.Content == strings.Repeat("t", 1000) {
                ok = true
            }
        }
        if f.Phase == PhaseStreaming && len(f.DraftText) == 1000 {
            // Very-late streaming frame with the full draft.
            ok = true
        }
        if !ok {
            t.Fatalf("replace-latest violated: got stale frame phase=%v draftLen=%d historyLen=%d",
                f.Phase, len(f.DraftText), len(f.History))
        }
    default:
        t.Fatal("no frame available — kernel may have blocked on emit")
    }

    // SANITY: drain any remaining frames so the kernel can shut down cleanly.
    go func() {
        for range k.Render() {
        }
    }()
}
```

**The assertion strategy in plain English:** during the stall window, the kernel must be free to keep generating frames and replacing the capacity-1 mailbox — not blocked on a send. When we finally read, we observe the LATEST state, not some stale intermediate state. If the replace-latest drain-then-send in `emitFrame` were broken (e.g. an unbuffered send), the kernel would have deadlocked at the first streaming frame and our 2-second sleep would have returned with the render mailbox holding the stale frame #2.

**Note on the final-idle detection:** after `PhaseIdle` is emitted, the kernel goes back to reading `k.events` and the render channel stays in its last state. The drain-then-close goroutine at the bottom lets `Run` exit on ctx cancel cleanly.

---

## 4. Success Criteria

The spec is delivered when **all** hold:

1. `gormes/scripts/check-go1.22-compat.sh` is executable and runs without requiring arguments.
2. On a host with Docker: the script exits 0, 1, or 2 with the documented semantics and, on exit 1, names at least one offending package + Go-version requirement.
3. On a host without Docker but with `golang.org/dl/go1.22.10` installable: fallback path triggers; same exit semantics.
4. `go test ./internal/kernel/... -v` completes in under 10 seconds and includes a `--- SKIP: TestKernel_HandlesMidStreamNetworkDrop` line.
5. `go test ./internal/kernel/... -v -run NonBlockingUnderTUIStall` PASSES.
6. `go test -race ./... -timeout 60s` stays green repo-wide.
7. `go vet ./...` clean.
8. `PhaseReconnecting.String()` returns `"Reconnecting"`.
9. No kernel behaviour change — specifically, `reconnect_test.go` un-skipped would still FAIL today (confirmed by a one-off `go test -run TestKernel_HandlesMidStreamNetworkDrop -tags skip_off` if we add such a tag in a follow-up).
10. Spec itself passes the Goldmark docs lint.

---

## 5. Explicit Out-of-Scope

| Item | Future home |
|---|---|
| The Route-B reconnect implementation (jittered backoff, retry state machine, turn-resume logic) | dedicated Phase 1.5 spec + plan — this spec's red test becomes that plan's green target |
| Tool-call event preservation across reconnects | same future spec |
| Visual "dimming" of leaked-draft tokens in the TUI during PhaseReconnecting | Phase 1.5 TUI enhancement; depends on the reconnect impl shipping first |
| CI workflow that runs `check-go1.22-compat.sh` on every PR | Phase 1.5 tooling item |
| Downgrading bubbletea / lipgloss to regain Go 1.22 compatibility | decision awaits the compat script's output |
| Session picker / `--new` / `--session <id>` flags | separate Phase 1.5 spec (user-facing feature) |

---

## 6. Risks & Mitigations

| Risk | Mitigation |
|---|---|
| `httptest.Server`'s dynamic port makes "restart on the same URL" hard | `rebindTo` helper uses an in-process reverse-proxy pattern: the kernel always points at a stable `127.0.0.1:<test-port>` and the proxy is swapped to a new backend. Adds ~30 lines of test scaffolding; acceptable |
| Docker unavailable in common dev environments | Fallback path via `golang.org/dl/go1.22.10` — pure Go toolchain download, no system dependency |
| Log-parsing in the compat script is fragile to Go version output changes | Script exits 1 with the raw log tail as well as the structured summary; human reads the raw log if the structured summary misses a case |
| Stall test's 2-second sleep is flaky on slow CI | Acceptable — 2 s is far above the kernel's 16 ms flush interval; if CI is > 100× slower than expected, the test should fail loudly because something deeper is wrong |
| Future Route-B implementer may interpret Assertion 4 as "retry must produce the exact string `yyyyyyyyyy`" | Spec comment in the test body clarifies: assertion checks the retry used the SECOND server's response, not that LLMs are deterministic. For a real provider, assertion would check the assistant message is non-empty and comes from a completed finish_reason=stop response |

---

## 7. Next Step

After this spec is user-approved, `superpowers:writing-plans` produces the implementation plan. Expected plan size: ~5–6 tasks (script, enum line, red-test helpers, red test, green test, verification sweep).

This spec is the source of truth for *what* the TDD rig is. The plan is the source of truth for *how* it gets built.
