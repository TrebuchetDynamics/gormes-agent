package gateway

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPairingStore_PersistsSnapshotAndSortsEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pairing", "state.json")

	store, err := OpenPairingStore(path)
	if err != nil {
		t.Fatalf("OpenPairingStore: %v", err)
	}

	requestedPending := time.Date(2026, 4, 23, 8, 21, 30, 0, time.UTC)
	requestedApproved := requestedPending.Add(2 * time.Minute)
	approvedAt := requestedApproved.Add(3 * time.Minute)

	if err := store.Put(PairingRecord{
		Platform:    " discord ",
		UserID:      " user-9 ",
		UserName:    " Grace ",
		ChatID:      " ops ",
		Status:      PairingStatusApproved,
		RequestedAt: requestedApproved,
		ApprovedAt:  &approvedAt,
		ApprovedBy:  " operator-1 ",
	}); err != nil {
		t.Fatalf("Put approved pairing: %v", err)
	}
	if err := store.Put(PairingRecord{
		Platform:    " Telegram ",
		UserID:      " 42 ",
		UserName:    " Ada ",
		ChatID:      " 42 ",
		Code:        " XKGH5N7P ",
		Status:      PairingStatusPending,
		RequestedAt: requestedPending,
	}); err != nil {
		t.Fatalf("Put pending pairing: %v", err)
	}

	reopened, err := OpenPairingStore(path)
	if err != nil {
		t.Fatalf("OpenPairingStore reopen: %v", err)
	}

	got := reopened.Snapshot()
	if len(got) != 2 {
		t.Fatalf("Snapshot len = %d, want 2", len(got))
	}

	if got[0].Platform != "discord" || got[0].UserID != "user-9" || got[0].ApprovedBy != "operator-1" {
		t.Fatalf("first snapshot entry = %+v, want normalized approved discord entry", got[0])
	}
	if got[0].ApprovedAt == nil || !got[0].ApprovedAt.Equal(approvedAt) {
		t.Fatalf("first ApprovedAt = %v, want %v", got[0].ApprovedAt, approvedAt)
	}
	if got[1].Platform != "telegram" || got[1].UserID != "42" || got[1].Code != "XKGH5N7P" {
		t.Fatalf("second snapshot entry = %+v, want normalized pending telegram entry", got[1])
	}
}

func TestManager_Inbound_StatusReportsPairingReadout(t *testing.T) {
	tg := newFakeChannel("telegram")
	dc := newFakeChannel("discord")
	fk := &fakeKernel{}

	pairings, err := OpenPairingStore(filepath.Join(t.TempDir(), "pairing", "state.json"))
	if err != nil {
		t.Fatalf("OpenPairingStore: %v", err)
	}

	requestedAt := time.Date(2026, 4, 23, 8, 21, 30, 0, time.UTC)
	approvedAt := requestedAt.Add(5 * time.Minute)
	if err := pairings.Put(PairingRecord{
		Platform:    "telegram",
		UserID:      "42",
		UserName:    "Ada",
		ChatID:      "42",
		Code:        "XKGH5N7P",
		Status:      PairingStatusPending,
		RequestedAt: requestedAt,
	}); err != nil {
		t.Fatalf("Put pending pairing: %v", err)
	}
	if err := pairings.Put(PairingRecord{
		Platform:    "discord",
		UserID:      "u9",
		UserName:    "Grace",
		ChatID:      "ops",
		Status:      PairingStatusApproved,
		RequestedAt: requestedAt.Add(time.Minute),
		ApprovedAt:  &approvedAt,
		ApprovedBy:  "operator-1",
	}); err != nil {
		t.Fatalf("Put approved pairing: %v", err)
	}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		Pairings:     pairings,
	}, fk, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register telegram: %v", err)
	}
	if err := m.Register(dc); err != nil {
		t.Fatalf("Register discord: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		Kind:     EventStatus,
		Text:     "/status",
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		return len(tg.sentSnapshot()) == 1
	})

	if n := len(fk.submitsSnapshot()); n != 0 {
		t.Fatalf("status should not submit to kernel, got %d submits", n)
	}

	got := tg.sentSnapshot()[0].Text
	for _, want := range []string{
		"Gateway status",
		"channels: discord, telegram",
		"pairings_pending: 1",
		"pairings_approved: 1",
		"- approved platform=discord user_id=u9 chat_id=ops user_name=\"Grace\" approved_by=operator-1",
		"- pending platform=telegram user_id=42 chat_id=42 user_name=\"Ada\" code=XKGH5N7P",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("status output = %q, want substring %q", got, want)
		}
	}
}
