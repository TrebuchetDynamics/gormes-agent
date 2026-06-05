package session

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
)

func TestBoltMap_MetadataRoundTripAndListByUserID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	m, err := OpenBolt(path)
	if err != nil {
		t.Fatalf("OpenBolt: %v", err)
	}
	defer m.Close()

	ctx := context.Background()
	if err := m.PutMetadata(ctx, Metadata{
		SessionID: "sess-telegram",
		Source:    "telegram",
		ChatID:    "42",
		UserID:    "user-juan",
		UpdatedAt: 10,
	}); err != nil {
		t.Fatalf("PutMetadata telegram: %v", err)
	}
	if err := m.PutMetadata(ctx, Metadata{
		SessionID: "sess-discord",
		Source:    "discord",
		ChatID:    "chan-9",
		UserID:    "user-juan",
		UpdatedAt: 20,
	}); err != nil {
		t.Fatalf("PutMetadata discord: %v", err)
	}

	got, ok, err := m.GetMetadata(ctx, "sess-telegram")
	if err != nil {
		t.Fatalf("GetMetadata: %v", err)
	}
	if !ok {
		t.Fatal("GetMetadata() ok = false, want true")
	}
	if got.Source != "telegram" || got.ChatID != "42" || got.UserID != "user-juan" {
		t.Fatalf("GetMetadata() = %+v, want telegram/42/user-juan", got)
	}

	userID, ok, err := m.ResolveUserID(ctx, "telegram", "42")
	if err != nil {
		t.Fatalf("ResolveUserID: %v", err)
	}
	if !ok {
		t.Fatal("ResolveUserID() ok = false, want true")
	}
	if userID != "user-juan" {
		t.Fatalf("ResolveUserID() = %q, want user-juan", userID)
	}

	listed, err := m.ListMetadataByUserID(ctx, "user-juan")
	if err != nil {
		t.Fatalf("ListMetadataByUserID: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("ListMetadataByUserID() len = %d, want 2", len(listed))
	}
	if listed[0].SessionID != "sess-discord" || listed[1].SessionID != "sess-telegram" {
		t.Fatalf("ListMetadataByUserID() = %+v, want UpdatedAt-desc deterministic order", listed)
	}
}

func TestBoltMap_PutMetadataInheritsUserBindingFromChat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	m, err := OpenBolt(path)
	if err != nil {
		t.Fatalf("OpenBolt: %v", err)
	}
	defer m.Close()

	ctx := context.Background()
	if err := m.PutMetadata(ctx, Metadata{
		SessionID: "sess-1",
		Source:    "telegram",
		ChatID:    "42",
		UserID:    "user-juan",
	}); err != nil {
		t.Fatalf("PutMetadata first session: %v", err)
	}
	if err := m.PutMetadata(ctx, Metadata{
		SessionID: "sess-2",
		Source:    "telegram",
		ChatID:    "42",
	}); err != nil {
		t.Fatalf("PutMetadata second session: %v", err)
	}

	got, ok, err := m.GetMetadata(ctx, "sess-2")
	if err != nil {
		t.Fatalf("GetMetadata second session: %v", err)
	}
	if !ok {
		t.Fatal("GetMetadata second session ok = false, want true")
	}
	if got.UserID != "user-juan" {
		t.Fatalf("GetMetadata second session user_id = %q, want inherited user-juan", got.UserID)
	}
}

func TestBoltMap_PutMetadataRejectsConflictingUserBinding(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	m, err := OpenBolt(path)
	if err != nil {
		t.Fatalf("OpenBolt: %v", err)
	}
	defer m.Close()

	ctx := context.Background()
	if err := m.PutMetadata(ctx, Metadata{
		SessionID: "sess-1",
		Source:    "telegram",
		ChatID:    "42",
		UserID:    "user-juan",
	}); err != nil {
		t.Fatalf("PutMetadata first binding: %v", err)
	}

	err = m.PutMetadata(ctx, Metadata{
		SessionID: "sess-2",
		Source:    "telegram",
		ChatID:    "42",
		UserID:    "user-maria",
	})
	if !errors.Is(err, ErrUserBindingConflict) {
		t.Fatalf("PutMetadata conflicting binding err = %v, want ErrUserBindingConflict", err)
	}
}

func TestMemMap_MetadataRoundTripAndConflictRules(t *testing.T) {
	m := NewMemMap()
	ctx := context.Background()

	if err := m.PutMetadata(ctx, Metadata{
		SessionID: "sess-telegram",
		Source:    "telegram",
		ChatID:    "42",
		UserID:    "user-juan",
		UpdatedAt: 10,
	}); err != nil {
		t.Fatalf("PutMetadata telegram: %v", err)
	}
	if err := m.PutMetadata(ctx, Metadata{
		SessionID: "sess-discord",
		Source:    "discord",
		ChatID:    "chan-9",
		UserID:    "user-juan",
		UpdatedAt: 20,
	}); err != nil {
		t.Fatalf("PutMetadata discord: %v", err)
	}

	listed, err := m.ListMetadataByUserID(ctx, "user-juan")
	if err != nil {
		t.Fatalf("ListMetadataByUserID: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("ListMetadataByUserID() len = %d, want 2", len(listed))
	}

	if err := m.PutMetadata(ctx, Metadata{
		SessionID: "sess-telegram-2",
		Source:    "telegram",
		ChatID:    "42",
	}); err != nil {
		t.Fatalf("PutMetadata inherited binding: %v", err)
	}
	got, ok, err := m.GetMetadata(ctx, "sess-telegram-2")
	if err != nil {
		t.Fatalf("GetMetadata inherited binding: %v", err)
	}
	if !ok || got.UserID != "user-juan" {
		t.Fatalf("GetMetadata inherited binding = %+v, %v, want user-juan", got, ok)
	}

	err = m.PutMetadata(ctx, Metadata{
		SessionID: "sess-conflict",
		Source:    "telegram",
		ChatID:    "42",
		UserID:    "user-maria",
	})
	if !errors.Is(err, ErrUserBindingConflict) {
		t.Fatalf("PutMetadata conflicting binding err = %v, want ErrUserBindingConflict", err)
	}
}

// TestMetadata_TitleManuallySetRoundTripsThroughBolt confirms that writing
// TitleManuallySet=true and reading back via GetMetadata returns the flag
// intact after a bolt round-trip.
func TestMetadata_TitleManuallySetRoundTripsThroughBolt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	m, err := OpenBolt(path)
	if err != nil {
		t.Fatalf("OpenBolt: %v", err)
	}
	defer m.Close()

	ctx := context.Background()
	if err := m.PutMetadata(ctx, Metadata{
		SessionID:        "sess-manual",
		Source:           "telegram",
		ChatID:           "42",
		Title:            "Operator Title",
		TitleManuallySet: true,
	}); err != nil {
		t.Fatalf("PutMetadata with TitleManuallySet=true: %v", err)
	}

	got, ok, err := m.GetMetadata(ctx, "sess-manual")
	if err != nil {
		t.Fatalf("GetMetadata: %v", err)
	}
	if !ok {
		t.Fatal("GetMetadata ok = false, want true")
	}
	if !got.TitleManuallySet {
		t.Fatal("TitleManuallySet = false after bolt round-trip, want true")
	}
	if got.Title != "Operator Title" {
		t.Fatalf("Title = %q, want %q", got.Title, "Operator Title")
	}
}

// TestMergeMetadata_ManualFlagIsStickyTrue verifies three sticky-merge cases:
//  1. false->true is allowed (incoming=true sets the flag).
//  2. true->false is NOT allowed (incoming=false must NOT clear an existing true).
//  3. incoming=true on existing=true stays true.
func TestMergeMetadata_ManualFlagIsStickyTrue(t *testing.T) {
	// Case 1: false -> true allowed.
	existing := Metadata{SessionID: "s", Title: "t", TitleManuallySet: false}
	incoming := Metadata{SessionID: "s", Title: "t", TitleManuallySet: true}
	out, err := mergeMetadata(existing, incoming)
	if err != nil {
		t.Fatalf("mergeMetadata case1: %v", err)
	}
	if !out.TitleManuallySet {
		t.Fatal("case1: mergeMetadata false->true: want TitleManuallySet=true")
	}

	// Case 2: true -> false NOT allowed — sticky.
	existing2 := Metadata{SessionID: "s", Title: "t", TitleManuallySet: true}
	incoming2 := Metadata{SessionID: "s", Title: "t", TitleManuallySet: false}
	out2, err := mergeMetadata(existing2, incoming2)
	if err != nil {
		t.Fatalf("mergeMetadata case2: %v", err)
	}
	if !out2.TitleManuallySet {
		t.Fatal("case2: mergeMetadata true->false must NOT clear existing true (sticky)")
	}

	// Case 3: true -> true stays true.
	existing3 := Metadata{SessionID: "s", Title: "t", TitleManuallySet: true}
	incoming3 := Metadata{SessionID: "s", Title: "t", TitleManuallySet: true}
	out3, err := mergeMetadata(existing3, incoming3)
	if err != nil {
		t.Fatalf("mergeMetadata case3: %v", err)
	}
	if !out3.TitleManuallySet {
		t.Fatal("case3: mergeMetadata true->true: want TitleManuallySet=true")
	}
}

// TestMetadata_LegacyDecodeHasManualFlagFalse confirms that a row that was
// encoded before the TitleManuallySet field existed (JSON without the key)
// decodes with TitleManuallySet=false so legacy sessions remain auto-title-eligible.
func TestMetadata_LegacyDecodeHasManualFlagFalse(t *testing.T) {
	// Simulate a pre-slice row: JSON without title_manually_set field.
	legacyJSON := `{"session_id":"legacy","source":"telegram","chat_id":"42","title":"Old Title","updated_at":1000}`
	var meta Metadata
	if err := json.Unmarshal([]byte(legacyJSON), &meta); err != nil {
		t.Fatalf("unmarshal legacy row: %v", err)
	}
	if meta.TitleManuallySet {
		t.Fatal("legacy row decoded with TitleManuallySet=true, want false (zero value)")
	}
	if meta.Title != "Old Title" {
		t.Fatalf("Title = %q, want %q", meta.Title, "Old Title")
	}
}
