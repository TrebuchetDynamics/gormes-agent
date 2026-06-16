package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

func TestGatewayRestartCommand_RegistryCarriesGatewaySafeActiveTurnPolicy(t *testing.T) {
	cmd, ok := ResolveCommand("/restart")
	if !ok {
		t.Fatal("ResolveCommand(/restart) not found")
	}
	if cmd.Kind != EventRestart {
		t.Fatalf("/restart kind = %v, want %v", cmd.Kind, EventRestart)
	}
	if cmd.ActiveTurnPolicy != CommandActiveTurnPolicyDrain {
		t.Fatalf("/restart active-turn policy = %q, want %q", cmd.ActiveTurnPolicy, CommandActiveTurnPolicyDrain)
	}

	kind, body := ParseInboundText("/restart")
	if kind != EventRestart || body != "" {
		t.Fatalf("ParseInboundText(/restart) = (%v, %q), want (%v, empty)", kind, body, EventRestart)
	}

	help := strings.Join(GatewayHelpLines(), "\n")
	if !strings.Contains(help, "/restart") {
		t.Fatalf("GatewayHelpLines missing /restart:\n%s", help)
	}
}

func TestGatewayRestartCommand_WritesTakeoverMarkerAndReturnsServiceManagerExitCode(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 25, 17, 40, 0, 0, time.UTC)
	store := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))
	store.now = func() time.Time { return now }
	if err := store.UpdateRuntimeStatus(ctx, RuntimeStatusUpdate{GatewayState: GatewayStateRunning}); err != nil {
		t.Fatalf("seed runtime status: %v", err)
	}

	markerStore := NewRestartTakeoverStore(filepath.Join(t.TempDir(), "restart_takeover.json"))
	markerStore.now = func() time.Time { return now }
	ch := newFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:  map[string]string{"telegram": "42"},
		RuntimeStatus: store,
		Restart: RestartConfig{
			MarkerStore:             markerStore,
			ServiceManagerAvailable: func() bool { return true },
			DrainTimeout:            time.Second,
		},
		Now: func() time.Time { return now },
	}, &fakeKernel{}, nil)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	err := m.handleInbound(ctx, InboundEvent{
		Platform:  "telegram",
		ChatID:    "42",
		ThreadID:  "thread-7",
		MsgID:     "msg-9",
		MessageID: "update-101",
		Kind:      EventRestart,
		Text:      "/restart",
	})
	if err == nil {
		t.Fatal("handleInbound(/restart) returned nil, want service-manager exit error")
	}
	var exitErr interface{ ExitCode() int }
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != GatewayServiceRestartExitCode {
		t.Fatalf("restart error = %v, exit code %d, want %d", err, commandExitCodeForGatewayTest(err), GatewayServiceRestartExitCode)
	}

	marker := readRestartMarkerFixture(t, markerStore.path)
	if marker.SourcePlatform != "telegram" || marker.ChatID != "42" || marker.ThreadID != "thread-7" {
		t.Fatalf("marker source = %+v, want telegram/42/thread-7", marker)
	}
	if marker.UpdateID != "update-101" || marker.MessageID != "msg-9" {
		t.Fatalf("marker IDs = update %q message %q, want update-101/msg-9", marker.UpdateID, marker.MessageID)
	}
	if marker.Generation == 0 {
		t.Fatalf("marker generation = 0, want runtime generation evidence: %+v", marker)
	}
	if marker.RequestedAt != now.Format(time.RFC3339Nano) {
		t.Fatalf("marker requested_at = %q, want %q", marker.RequestedAt, now.Format(time.RFC3339Nano))
	}

	status, err := store.ReadRuntimeStatus(ctx)
	if err != nil {
		t.Fatalf("ReadRuntimeStatus: %v", err)
	}
	if !status.RestartRequested {
		t.Fatalf("RestartRequested = false, want true: %+v", status)
	}
	if len(status.TakeoverMarkers) != 1 || status.TakeoverMarkers[0].Status != RestartTakeoverMarkerStatusWritten {
		t.Fatalf("TakeoverMarkers = %+v, want one written marker evidence", status.TakeoverMarkers)
	}
}

func TestGatewayRestartCommand_DuplicateRedeliveryIsSuppressedOnce(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 25, 17, 41, 0, 0, time.UTC)
	statusStore := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))
	statusStore.now = func() time.Time { return now }
	markerStore := NewRestartTakeoverStore(filepath.Join(t.TempDir(), "restart_takeover.json"))
	markerStore.now = func() time.Time { return now }
	if err := markerStore.Write(ctx, RestartTakeoverMarker{
		SourcePlatform: "telegram",
		ChatID:         "42",
		UpdateID:       "update-101",
		MessageID:      "msg-9",
		Generation:     3,
		RequestedAt:    now.Format(time.RFC3339Nano),
	}); err != nil {
		t.Fatalf("seed marker: %v", err)
	}

	ch := newFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:  map[string]string{"telegram": "42"},
		RuntimeStatus: statusStore,
		Restart: RestartConfig{
			MarkerStore:             markerStore,
			ServiceManagerAvailable: func() bool { return true },
			DrainTimeout:            time.Second,
		},
		Now: func() time.Time { return now },
	}, &fakeKernel{}, nil)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	err := m.handleInbound(ctx, InboundEvent{
		Platform:  "telegram",
		ChatID:    "42",
		MsgID:     "msg-9",
		MessageID: "update-101",
		Kind:      EventRestart,
		Text:      "/restart",
	})
	if err != nil {
		t.Fatalf("duplicate /restart returned error %v, want suppressed without exit", err)
	}
	if _, statErr := os.Stat(markerStore.path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("marker stat after duplicate suppression = %v, want removed", statErr)
	}
	status, err := statusStore.ReadRuntimeStatus(ctx)
	if err != nil {
		t.Fatalf("ReadRuntimeStatus: %v", err)
	}
	if len(status.DuplicateRestarts) != 1 || status.DuplicateRestarts[0].Status != RestartDuplicateStatusSuppressed {
		t.Fatalf("DuplicateRestarts = %+v, want one suppressed evidence", status.DuplicateRestarts)
	}
	if sent := ch.sentSnapshot(); len(sent) != 1 || !strings.Contains(sent[0].Text, "duplicate_restart_suppressed") {
		t.Fatalf("duplicate suppression reply = %#v, want duplicate_restart_suppressed evidence", sent)
	}

	err = m.handleInbound(ctx, InboundEvent{
		Platform:  "telegram",
		ChatID:    "42",
		MsgID:     "msg-9",
		MessageID: "update-101",
		Kind:      EventRestart,
		Text:      "/restart",
	})
	if err == nil {
		t.Fatal("second /restart after marker consumption returned nil, want fresh service-manager exit")
	}
}

func TestRestartTakeoverStoreAllowsNilContext(t *testing.T) {
	now := time.Date(2026, 5, 10, 10, 30, 0, 0, time.UTC)
	store := NewRestartTakeoverStore(filepath.Join(t.TempDir(), "restart_takeover.json"))
	store.now = func() time.Time { return now }
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("restart takeover store panicked with nil context: %v", r)
		}
	}()

	seed := RestartTakeoverMarker{
		SourcePlatform: "telegram",
		ChatID:         "42",
		UpdateID:       "update-101",
		MessageID:      "msg-9",
	}
	if err := store.Write(nil, seed); err != nil {
		t.Fatalf("Write nil context: %v", err)
	}
	marker, ok, _, err := store.Read(nil)
	if err != nil || !ok {
		t.Fatalf("Read nil context marker=%+v ok=%v err=%v, want existing marker", marker, ok, err)
	}
	if err := store.MarkNotificationSent(nil, marker, now.Add(time.Second)); err != nil {
		t.Fatalf("MarkNotificationSent nil context: %v", err)
	}
	_, suppressed, err := store.SuppressDuplicate(nil, InboundEvent{
		Platform:  "telegram",
		ChatID:    "42",
		MsgID:     "msg-9",
		MessageID: "update-101",
		Kind:      EventRestart,
	})
	if err != nil || !suppressed {
		t.Fatalf("SuppressDuplicate nil context suppressed=%v err=%v, want suppressed", suppressed, err)
	}
}

func TestRestartTakeoverStoreSuppressDuplicateTrimsMarkerIDs(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 10, 11, 0, 0, 0, time.UTC)
	store := NewRestartTakeoverStore(filepath.Join(t.TempDir(), "restart_takeover.json"))
	store.now = func() time.Time { return now }
	if err := store.Write(ctx, RestartTakeoverMarker{
		SourcePlatform: "telegram",
		ChatID:         "42",
		UpdateID:       " update-101 ",
		MessageID:      " msg-9 ",
		RequestedAt:    now.Format(time.RFC3339Nano),
	}); err != nil {
		t.Fatalf("seed marker: %v", err)
	}

	marker, suppressed, err := store.SuppressDuplicate(ctx, InboundEvent{
		Platform:  "telegram",
		ChatID:    "42",
		MsgID:     "msg-9",
		MessageID: "update-101",
		Kind:      EventRestart,
	})
	if err != nil {
		t.Fatalf("SuppressDuplicate: %v", err)
	}
	if !suppressed {
		t.Fatalf("SuppressDuplicate suppressed=false marker=%+v, want trimmed marker IDs to match redelivered event", marker)
	}
	if _, statErr := os.Stat(store.path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("marker stat after duplicate suppression = %v, want removed", statErr)
	}
}

func TestGatewayRestartCommand_ServiceManagerUnavailableReportsDegradedWithoutExit(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 25, 17, 42, 0, 0, time.UTC)
	statusStore := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))
	statusStore.now = func() time.Time { return now }
	ch := newFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:  map[string]string{"telegram": "42"},
		RuntimeStatus: statusStore,
		Restart: RestartConfig{
			ServiceManagerAvailable: func() bool { return false },
			DrainTimeout:            time.Second,
		},
		Now: func() time.Time { return now },
	}, &fakeKernel{}, nil)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := m.handleInbound(ctx, InboundEvent{Platform: "telegram", ChatID: "42", MsgID: "msg-1", Kind: EventRestart}); err != nil {
		t.Fatalf("handleInbound unavailable service manager = %v, want nil", err)
	}
	status, err := statusStore.ReadRuntimeStatus(ctx)
	if err != nil {
		t.Fatalf("ReadRuntimeStatus: %v", err)
	}
	if !status.RestartRequested {
		t.Fatalf("RestartRequested = false, want degraded request evidence")
	}
	if len(status.ServiceManagerUnavailable) != 1 {
		t.Fatalf("ServiceManagerUnavailable = %+v, want one evidence row", status.ServiceManagerUnavailable)
	}
	if sent := ch.sentSnapshot(); len(sent) != 1 || !strings.Contains(sent[0].Text, "Restart unavailable") {
		t.Fatalf("service-manager unavailable reply = %#v, want restart unavailable evidence", sent)
	}
}

func TestGatewayRestartCommand_NoServiceManagerUsesMarkerExitPath(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 25, 17, 42, 30, 0, time.UTC)
	statusStore := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))
	statusStore.now = func() time.Time { return now }
	markerStore := NewRestartTakeoverStore(filepath.Join(t.TempDir(), "restart_takeover.json"))
	markerStore.now = func() time.Time { return now }
	ch := newFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:  map[string]string{"telegram": "42"},
		RuntimeStatus: statusStore,
		Restart: RestartConfig{
			MarkerStore:             markerStore,
			ServiceManagerAvailable: func() bool { return false },
			DrainTimeout:            time.Second,
		},
		Now: func() time.Time { return now },
	}, &fakeKernel{}, nil)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	err := m.handleInbound(ctx, InboundEvent{
		Platform:  "telegram",
		ChatID:    "42",
		ThreadID:  "thread-7",
		MsgID:     "msg-1",
		MessageID: "update-1",
		Kind:      EventRestart,
		Text:      "/restart",
	})
	if err == nil {
		t.Fatal("handleInbound(/restart) returned nil, want restart exit error even without service manager env")
	}
	var exitErr interface{ ExitCode() int }
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != GatewayServiceRestartExitCode {
		t.Fatalf("restart error = %v, exit code %d, want %d", err, commandExitCodeForGatewayTest(err), GatewayServiceRestartExitCode)
	}

	marker := readRestartMarkerFixture(t, markerStore.path)
	if marker.SourcePlatform != "telegram" || marker.ChatID != "42" || marker.ThreadID != "thread-7" {
		t.Fatalf("marker source = %+v, want telegram/42/thread-7", marker)
	}
	if marker.UpdateID != "update-1" || marker.MessageID != "msg-1" {
		t.Fatalf("marker IDs = update %q message %q, want update-1/msg-1", marker.UpdateID, marker.MessageID)
	}

	status, err := statusStore.ReadRuntimeStatus(ctx)
	if err != nil {
		t.Fatalf("ReadRuntimeStatus: %v", err)
	}
	if !status.RestartRequested {
		t.Fatalf("RestartRequested = false, want true: %+v", status)
	}
	if len(status.TakeoverMarkers) != 1 || status.TakeoverMarkers[0].Status != RestartTakeoverMarkerStatusWritten {
		t.Fatalf("TakeoverMarkers = %+v, want one written marker evidence", status.TakeoverMarkers)
	}
	if len(status.ServiceManagerUnavailable) != 0 {
		t.Fatalf("ServiceManagerUnavailable = %+v, want none when marker exit path is available", status.ServiceManagerUnavailable)
	}
	if sent := ch.sentSnapshot(); len(sent) != 1 || !strings.Contains(sent[0].Text, "gateway restart path") {
		t.Fatalf("restart reply = %#v, want gateway restart path handoff", sent)
	}
}

func TestGatewayRestartCommand_NoServiceManagerUsesSelfRestartHook(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 25, 17, 42, 45, 0, time.UTC)
	statusStore := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))
	statusStore.now = func() time.Time { return now }
	ch := newFakeChannel("telegram")
	selfRestarts := 0
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:  map[string]string{"telegram": "42"},
		RuntimeStatus: statusStore,
		Restart: RestartConfig{
			ServiceManagerAvailable: func() bool { return false },
			SelfRestart: func() error {
				selfRestarts++
				return nil
			},
			DrainTimeout: time.Second,
		},
		Now: func() time.Time { return now },
	}, &fakeKernel{}, nil)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := m.handleInbound(ctx, InboundEvent{Platform: "telegram", ChatID: "42", MsgID: "msg-1", Kind: EventRestart}); err != nil {
		t.Fatalf("handleInbound(/restart) with self-restart hook = %v, want nil", err)
	}
	if selfRestarts != 1 {
		t.Fatalf("self restarts = %d, want 1", selfRestarts)
	}
	status, err := statusStore.ReadRuntimeStatus(ctx)
	if err != nil {
		t.Fatalf("ReadRuntimeStatus: %v", err)
	}
	if !status.RestartRequested {
		t.Fatalf("RestartRequested = false, want true: %+v", status)
	}
	if len(status.ServiceManagerUnavailable) != 0 {
		t.Fatalf("ServiceManagerUnavailable = %+v, want none when self-restart hook is available", status.ServiceManagerUnavailable)
	}
	if sent := ch.sentSnapshot(); len(sent) != 1 || !strings.Contains(sent[0].Text, "gateway restart path") {
		t.Fatalf("restart reply = %#v, want gateway restart path handoff", sent)
	}
}

func TestGatewayRestartCommand_DrainTimeoutUsesResumePendingRecovery(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 25, 17, 43, 0, 0, time.UTC)
	statusStore := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))
	statusStore.now = func() time.Time { return now }
	markerStore := NewRestartTakeoverStore(filepath.Join(t.TempDir(), "restart_takeover.json"))
	markerStore.now = func() time.Time { return now }
	smap := session.NewMemMap()
	if err := smap.Put(ctx, "telegram:42", "sess-running"); err != nil {
		t.Fatalf("Put session: %v", err)
	}
	ch := newFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:  map[string]string{"telegram": "42"},
		RuntimeStatus: statusStore,
		SessionMap:    smap,
		Restart: RestartConfig{
			MarkerStore:             markerStore,
			ServiceManagerAvailable: func() bool { return true },
			DrainTimeout:            5 * time.Millisecond,
		},
		Now: func() time.Time { return now },
	}, &fakeKernel{}, nil)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	m.pinTurn("telegram", "42", "active-msg")
	m.setPinnedTurnSession("telegram:42", "sess-running", SessionSource{
		Platform: "telegram",
		ChatID:   "42",
		UserID:   "user-42",
	})

	err := m.handleInbound(ctx, InboundEvent{Platform: "telegram", ChatID: "42", MsgID: "restart-msg", Kind: EventRestart})
	if err == nil {
		t.Fatal("handleInbound restart returned nil, want service-manager exit after drain timeout")
	}
	if !errors.As(err, new(interface{ ExitCode() int })) {
		t.Fatalf("restart timeout error = %v, want coded service-manager exit", err)
	}

	meta, ok, err := smap.GetMetadata(ctx, "sess-running")
	if err != nil {
		t.Fatalf("GetMetadata: %v", err)
	}
	if !ok || !meta.ResumePending || meta.ResumeReason != session.ResumeReasonRestartTimeout {
		t.Fatalf("metadata = %+v ok=%v, want restart_timeout resume_pending", meta, ok)
	}
	status, err := statusStore.ReadRuntimeStatus(ctx)
	if err != nil {
		t.Fatalf("ReadRuntimeStatus: %v", err)
	}
	if len(status.DrainTimeouts) != 1 || status.DrainTimeouts[0].Reason != session.ResumeReasonRestartTimeout {
		t.Fatalf("DrainTimeouts = %+v, want restart_timeout evidence", status.DrainTimeouts)
	}
	if len(status.ResumePending) != 1 || status.ResumePending[0].SessionID != "sess-running" {
		t.Fatalf("ResumePending = %+v, want sess-running evidence", status.ResumePending)
	}
}

func TestGatewayRestartCommand_TakeoverStartupNotificationTrimsMarkerAddress(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	statusStore := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))
	statusStore.now = func() time.Time { return now }
	markerStore := NewRestartTakeoverStore(filepath.Join(t.TempDir(), "restart_takeover.json"))
	markerStore.now = func() time.Time { return now }
	if err := markerStore.Write(ctx, RestartTakeoverMarker{
		SourcePlatform: " telegram ",
		ChatID:         " 42 ",
		UpdateID:       " update-101 ",
		MessageID:      " msg-9 ",
		Generation:     3,
		RequestedAt:    now.Format(time.RFC3339Nano),
	}); err != nil {
		t.Fatalf("seed marker: %v", err)
	}

	ch := newFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:  map[string]string{"telegram": "42"},
		RuntimeStatus: statusStore,
		Restart:       RestartConfig{MarkerStore: markerStore},
		Now:           func() time.Time { return now },
	}, &fakeKernel{}, nil)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := m.ConsumeRestartTakeoverMarker(ctx); err != nil {
		t.Fatalf("ConsumeRestartTakeoverMarker: %v", err)
	}
	if sent := ch.sentSnapshot(); len(sent) != 1 || sent[0].ChatID != "42" || !strings.Contains(sent[0].Text, "Gateway restarted") {
		t.Fatalf("restart notification sends = %#v, want trimmed telegram/42 restarted notification", sent)
	}
}

func TestGatewayRestartCommand_TakeoverStartupNotificationIsOnceAndExpiryClearsMarker(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 25, 17, 44, 0, 0, time.UTC)
	statusStore := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))
	statusStore.now = func() time.Time { return now }
	markerStore := NewRestartTakeoverStore(filepath.Join(t.TempDir(), "restart_takeover.json"))
	markerStore.now = func() time.Time { return now }
	if err := markerStore.Write(ctx, RestartTakeoverMarker{
		SourcePlatform: "telegram",
		ChatID:         "42",
		UpdateID:       "update-101",
		MessageID:      "msg-9",
		Generation:     3,
		RequestedAt:    now.Format(time.RFC3339Nano),
	}); err != nil {
		t.Fatalf("seed marker: %v", err)
	}

	ch := newFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:  map[string]string{"telegram": "42"},
		RuntimeStatus: statusStore,
		Restart: RestartConfig{
			MarkerStore: markerStore,
		},
		Now: func() time.Time { return now },
	}, &fakeKernel{}, nil)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := m.ConsumeRestartTakeoverMarker(ctx); err != nil {
		t.Fatalf("ConsumeRestartTakeoverMarker first: %v", err)
	}
	if sent := ch.sentSnapshot(); len(sent) != 1 || !strings.Contains(sent[0].Text, "Gateway restarted") {
		t.Fatalf("restart notification sends = %#v, want one restarted notification", sent)
	}
	if err := m.ConsumeRestartTakeoverMarker(ctx); err != nil {
		t.Fatalf("ConsumeRestartTakeoverMarker second: %v", err)
	}
	if sent := ch.sentSnapshot(); len(sent) != 1 {
		t.Fatalf("restart notification sent more than once: %#v", sent)
	}
	status, err := statusStore.ReadRuntimeStatus(ctx)
	if err != nil {
		t.Fatalf("ReadRuntimeStatus: %v", err)
	}
	if len(status.TakeoverMarkers) != 1 || status.TakeoverMarkers[0].Status != RestartTakeoverMarkerStatusSeen {
		t.Fatalf("TakeoverMarkers = %+v, want one seen marker evidence", status.TakeoverMarkers)
	}

	markerStore.now = func() time.Time { return now.Add(RestartTakeoverMarkerTTL + time.Second) }
	if err := m.ConsumeRestartTakeoverMarker(ctx); err != nil {
		t.Fatalf("ConsumeRestartTakeoverMarker expired: %v", err)
	}
	if _, err := os.Stat(markerStore.path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expired marker stat = %v, want removed", err)
	}
}

func TestRestartTakeoverStoreReadExpiresFutureMarkerBeyondTTL(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC)
	store := NewRestartTakeoverStore(filepath.Join(t.TempDir(), "restart_takeover.json"))
	store.now = func() time.Time { return now }
	if err := store.Write(ctx, RestartTakeoverMarker{
		SourcePlatform: "telegram",
		ChatID:         "42",
		UpdateID:       "update-101",
		RequestedAt:    now.Add(2 * RestartTakeoverMarkerTTL).Format(time.RFC3339Nano),
	}); err != nil {
		t.Fatalf("Write future marker: %v", err)
	}

	marker, ok, expired, err := store.Read(ctx)
	if err != nil {
		t.Fatalf("Read future marker: %v", err)
	}
	if ok || !expired {
		t.Fatalf("Read future marker = marker=%+v ok=%v expired=%v, want expired non-ok", marker, ok, expired)
	}
	if _, err := os.Stat(store.path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("future marker stat = %v, want removed", err)
	}
}

func TestRestartTakeoverStoreReadPropagatesUnreadableMarker(t *testing.T) {
	ctx := context.Background()
	store := NewRestartTakeoverStore(t.TempDir())

	marker, ok, expired, err := store.Read(ctx)
	if err == nil {
		t.Fatalf("Read err = nil marker=%+v ok=%v expired=%v, want read error for directory marker path", marker, ok, expired)
	}
	if ok || expired {
		t.Fatalf("Read ok=%v expired=%v, want unreadable marker to be an error not missing/expired", ok, expired)
	}
}

func TestGatewayRestartCommand_TakeoverStartupNotificationCanBeMutedPerPlatform(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC)
	statusStore := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))
	statusStore.now = func() time.Time { return now }
	markerStore := NewRestartTakeoverStore(filepath.Join(t.TempDir(), "restart_takeover.json"))
	markerStore.now = func() time.Time { return now }
	if err := markerStore.Write(ctx, RestartTakeoverMarker{
		SourcePlatform: "telegram",
		ChatID:         "42",
		UpdateID:       "update-101",
		MessageID:      "msg-9",
		Generation:     3,
		RequestedAt:    now.Format(time.RFC3339Nano),
	}); err != nil {
		t.Fatalf("seed marker: %v", err)
	}

	ch := newFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:  map[string]string{"telegram": "42"},
		RuntimeStatus: statusStore,
		Restart: RestartConfig{
			MarkerStore: markerStore,
		},
		RestartNotifications: map[string]bool{"telegram": false},
		Now:                  func() time.Time { return now },
	}, &fakeKernel{}, nil)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := m.ConsumeRestartTakeoverMarker(ctx); err != nil {
		t.Fatalf("ConsumeRestartTakeoverMarker: %v", err)
	}
	if sent := ch.sentSnapshot(); len(sent) != 0 {
		t.Fatalf("restart notification sends = %#v, want none when disabled", sent)
	}
	marker, ok, expired, err := markerStore.Read(ctx)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if !ok || expired || marker.NotificationSentAt == "" {
		t.Fatalf("marker after muted notification = %+v ok=%v expired=%v, want consumed with notification_sent_at", marker, ok, expired)
	}
	if err := m.ConsumeRestartTakeoverMarker(ctx); err != nil {
		t.Fatalf("ConsumeRestartTakeoverMarker second: %v", err)
	}
	if sent := ch.sentSnapshot(); len(sent) != 0 {
		t.Fatalf("restart notification retried after muted consumption: %#v", sent)
	}
	status, err := statusStore.ReadRuntimeStatus(ctx)
	if err != nil {
		t.Fatalf("ReadRuntimeStatus: %v", err)
	}
	if len(status.TakeoverMarkers) != 1 || status.TakeoverMarkers[0].Status != RestartTakeoverMarkerStatusSeen {
		t.Fatalf("TakeoverMarkers = %+v, want one seen marker evidence", status.TakeoverMarkers)
	}
}

func readRestartMarkerFixture(t *testing.T, path string) RestartTakeoverMarker {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	var marker RestartTakeoverMarker
	if err := json.Unmarshal(raw, &marker); err != nil {
		t.Fatalf("decode marker: %v\n%s", err, raw)
	}
	return marker
}

func commandExitCodeForGatewayTest(err error) int {
	var exitErr interface{ ExitCode() int }
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 0
}

func TestGatewayRestartCommand_SelfRestartCalledWhenNoServiceManager(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	statusStore := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "gateway_state.json"))
	statusStore.now = func() time.Time { return now }
	if err := statusStore.UpdateRuntimeStatus(ctx, RuntimeStatusUpdate{GatewayState: GatewayStateRunning}); err != nil {
		t.Fatalf("seed runtime status: %v", err)
	}

	markerStore := NewRestartTakeoverStore(filepath.Join(t.TempDir(), "restart_takeover.json"))
	markerStore.now = func() time.Time { return now }

	selfRestartCalled := false
	ch := newFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:  map[string]string{"telegram": "42"},
		RuntimeStatus: statusStore,
		Restart: RestartConfig{
			MarkerStore:             markerStore,
			ServiceManagerAvailable: func() bool { return false },
			SelfRestart: func() error {
				selfRestartCalled = true
				return nil
			},
			DrainTimeout: time.Second,
		},
		Now: func() time.Time { return now },
	}, &fakeKernel{}, nil)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	err := m.handleInbound(ctx, InboundEvent{
		Platform:  "telegram",
		ChatID:    "42",
		MsgID:     "msg-restart",
		MessageID: "update-restart",
		Kind:      EventRestart,
		Text:      "/restart",
	})
	if err != nil {
		t.Fatalf("handleInbound self-restart returned error: %v", err)
	}
	if !selfRestartCalled {
		t.Fatal("SelfRestart was not called when no service manager is available")
	}

	marker := readRestartMarkerFixture(t, markerStore.path)
	if marker.SourcePlatform != "telegram" || marker.ChatID != "42" {
		t.Fatalf("marker source = %+v, want telegram/42", marker)
	}
}
