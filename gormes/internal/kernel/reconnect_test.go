package kernel

import (
	"testing"
)

// TestKernel_HandlesMidStreamNetworkDrop is the Phase-1.5 red chaos test for
// the Route-B reconnect feature documented in spec §9.2 of
// 2026-04-18-gormes-frontend-adapter-design.md.
//
// Currently t.Skip'd. CURRENTLY FAILS (by design) against the shipped kernel
// at four distinct assertions:
//
//  1. PhaseReconnecting transition after TCP drop
//     → current kernel transitions to PhaseFailed
//
//  2. Draft preserved during reconnect window (5 tokens stay visible)
//     → current kernel leaves draft in place but phase is already wrong,
//       so the invariant is not coherent
//
//  3. Automatic recovery back to PhaseStreaming → PhaseIdle after backoff
//     → current kernel has no retry loop
//
//  4. Final history contains exactly ONE clean assistant message from the
//     successful retry (no Frankenstein concatenation)
//     → current kernel never retries, so no such history entry exists
//
// The future Route-B implementation plan flips this test from Skip to real
// pass by wiring PhaseReconnecting into runTurn's error path with jittered
// exponential backoff (1s, 2s, 4s, 8s, 16s caps).
func TestKernel_HandlesMidStreamNetworkDrop(t *testing.T) {
	t.Skip("RED TEST: Route B Resilience — Implementation pending (see spec §9.2 of 2026-04-18-gormes-frontend-adapter-design.md)")

	// When the implementation lands, delete the Skip above and implement the
	// assertions using the helpers in reconnect_helpers_test.go:
	//
	//   1. p := newStableProxy(t); defer p.Close()
	//   2. srv1 := httptest.NewServer(fiveTokenHandler()); p.Rebind(srv1.URL)
	//   3. k := newRealKernel(t, p.URL()); go k.Run(ctx)
	//   4. k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "hi"})
	//   5. Wait for draft of length 5 + PhaseStreaming
	//   6. srv1.CloseClientConnections()  // chaos monkey
	//   7. ASSERT 1: phase == PhaseReconnecting within 500ms
	//   8. ASSERT 2: draft still contains "xxxxx"
	//   9. srv2 := httptest.NewServer(tenTokenHandler()); p.Rebind(srv2.URL)
	//  10. ASSERT 3: phase transitions Streaming → Idle within 20s
	//  11. ASSERT 4: final history has one assistant msg == "yyyyyyyyyy"
}
