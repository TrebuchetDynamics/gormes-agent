package session

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRecapEnvelope_EmptyStore(t *testing.T) {
	m := NewMemMap()
	ctx := context.Background()

	envelope, err := GenerateRecap(ctx, m, RecapConfig{MaxEntries: 10})
	if err != nil {
		t.Fatalf("GenerateRecap: %v", err)
	}
	if len(envelope.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(envelope.Entries))
	}
	if envelope.TotalSessions != 0 {
		t.Errorf("expected TotalSessions=0, got %d", envelope.TotalSessions)
	}
}

func TestRecapEnvelope_SingleSession(t *testing.T) {
	m := NewMemMap()
	ctx := context.Background()
	now := time.Now().Unix()

	meta := Metadata{
		SessionID: "sess-abc123",
		Title:     "Test session",
		CreatedAt: now - 3600,
		UpdatedAt: now,
	}
	if err := m.PutMetadata(ctx, meta); err != nil {
		t.Fatalf("PutMetadata: %v", err)
	}

	envelope, err := GenerateRecap(ctx, m, RecapConfig{MaxEntries: 10})
	if err != nil {
		t.Fatalf("GenerateRecap: %v", err)
	}
	if len(envelope.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(envelope.Entries))
	}
	e := envelope.Entries[0]
	if e.SessionID != "sess-abc123" {
		t.Errorf("SessionID=%q, want sess-abc123", e.SessionID)
	}
	if e.Title != "Test session" {
		t.Errorf("Title=%q, want 'Test session'", e.Title)
	}
	if envelope.TotalSessions != 1 {
		t.Errorf("TotalSessions=%d, want 1", envelope.TotalSessions)
	}
}

func TestRecapEnvelope_MissingSession(t *testing.T) {
	m := NewMemMap()
	ctx := context.Background()

	result, err := GenerateSessionRecap(ctx, m, "nonexistent", RecapConfig{MaxEntries: 10})
	if err != nil {
		t.Fatalf("GenerateSessionRecap: %v", err)
	}
	if !result.NotFound {
		t.Error("expected NotFound=true for missing session")
	}
	if result.SessionID != "nonexistent" {
		t.Errorf("SessionID=%q, want nonexistent", result.SessionID)
	}
}

func TestRecapEnvelope_KnownSession(t *testing.T) {
	m := NewMemMap()
	ctx := context.Background()
	now := time.Now().Unix()

	meta := Metadata{
		SessionID:      "sess-known",
		Title:          "Known session",
		CreatedAt:      now - 7200,
		UpdatedAt:      now,
		TokensInTotal:  1500,
		TokensOutTotal: 2300,
	}
	if err := m.PutMetadata(ctx, meta); err != nil {
		t.Fatalf("PutMetadata: %v", err)
	}

	result, err := GenerateSessionRecap(ctx, m, "sess-known", RecapConfig{MaxEntries: 10})
	if err != nil {
		t.Fatalf("GenerateSessionRecap: %v", err)
	}
	if result.NotFound {
		t.Error("expected NotFound=false for known session")
	}
	if result.Title != "Known session" {
		t.Errorf("Title=%q, want 'Known session'", result.Title)
	}
	if result.TokensIn != 1500 {
		t.Errorf("TokensIn=%d, want 1500", result.TokensIn)
	}
	if result.TokensOut != 2300 {
		t.Errorf("TokensOut=%d, want 2300", result.TokensOut)
	}
}

func TestRecapEnvelope_TruncationEvidence(t *testing.T) {
	m := NewMemMap()
	ctx := context.Background()
	now := time.Now().Unix()

	// Create 5 sessions
	for i := 0; i < 5; i++ {
		meta := Metadata{
			SessionID: "sess-" + string(rune('a'+i)),
			Title:     "Session " + string(rune('A'+i)),
			CreatedAt: now - int64(i*3600),
			UpdatedAt: now - int64(i*3600),
		}
		if err := m.PutMetadata(ctx, meta); err != nil {
			t.Fatalf("PutMetadata[%d]: %v", i, err)
		}
	}

	envelope, err := GenerateRecap(ctx, m, RecapConfig{MaxEntries: 3})
	if err != nil {
		t.Fatalf("GenerateRecap: %v", err)
	}
	if len(envelope.Entries) != 3 {
		t.Errorf("expected 3 entries (truncated), got %d", len(envelope.Entries))
	}
	if !envelope.Truncated {
		t.Error("expected Truncated=true when MaxEntries < TotalSessions")
	}
	if envelope.TotalSessions != 5 {
		t.Errorf("TotalSessions=%d, want 5", envelope.TotalSessions)
	}
}

func TestRecapEnvelope_HumanOutput(t *testing.T) {
	m := NewMemMap()
	ctx := context.Background()
	now := time.Now().Unix()

	meta := Metadata{
		SessionID: "sess-human",
		Title:     "Human test",
		CreatedAt: now - 3600,
		UpdatedAt: now,
	}
	if err := m.PutMetadata(ctx, meta); err != nil {
		t.Fatalf("PutMetadata: %v", err)
	}

	envelope, err := GenerateRecap(ctx, m, RecapConfig{MaxEntries: 10})
	if err != nil {
		t.Fatalf("GenerateRecap: %v", err)
	}

	human := envelope.HumanOutput()
	if !strings.Contains(human, "sess-human") {
		t.Errorf("HumanOutput missing session ID: %s", human)
	}
	if !strings.Contains(human, "Human test") {
		t.Errorf("HumanOutput missing title: %s", human)
	}
}

func TestRecapEnvelope_HumanOutput_Empty(t *testing.T) {
	m := NewMemMap()
	ctx := context.Background()

	envelope, err := GenerateRecap(ctx, m, RecapConfig{MaxEntries: 10})
	if err != nil {
		t.Fatalf("GenerateRecap: %v", err)
	}

	human := envelope.HumanOutput()
	if !strings.Contains(human, "No sessions") {
		t.Errorf("HumanOutput should indicate no sessions: %s", human)
	}
}

func TestRecapEnvelope_SessionRecapHumanOutput(t *testing.T) {
	m := NewMemMap()
	ctx := context.Background()
	now := time.Now().Unix()

	meta := Metadata{
		SessionID:      "sess-detail",
		Title:          "Detail test",
		CreatedAt:      now - 7200,
		UpdatedAt:      now,
		TokensInTotal:  500,
		TokensOutTotal: 800,
	}
	if err := m.PutMetadata(ctx, meta); err != nil {
		t.Fatalf("PutMetadata: %v", err)
	}

	result, err := GenerateSessionRecap(ctx, m, "sess-detail", RecapConfig{MaxEntries: 10})
	if err != nil {
		t.Fatalf("GenerateSessionRecap: %v", err)
	}

	human := result.HumanOutput()
	if !strings.Contains(human, "Detail test") {
		t.Errorf("HumanOutput missing title: %s", human)
	}
	if !strings.Contains(human, "500") {
		t.Errorf("HumanOutput missing tokens_in: %s", human)
	}
}

func TestRecapEnvelope_SessionRecapNotFoundHumanOutput(t *testing.T) {
	m := NewMemMap()
	ctx := context.Background()

	result, err := GenerateSessionRecap(ctx, m, "missing-sess", RecapConfig{MaxEntries: 10})
	if err != nil {
		t.Fatalf("GenerateSessionRecap: %v", err)
	}

	human := result.HumanOutput()
	if !strings.Contains(human, "missing-sess") {
		t.Errorf("HumanOutput missing session ID: %s", human)
	}
	if !strings.Contains(human, "not found") {
		t.Errorf("HumanOutput should indicate not found: %s", human)
	}
}

func TestRecapEnvelope_SortedByUpdatedAt(t *testing.T) {
	m := NewMemMap()
	ctx := context.Background()
	now := time.Now().Unix()

	// Create sessions with different update times (oldest first)
	for i := 0; i < 3; i++ {
		meta := Metadata{
			SessionID: "sess-sort-" + string(rune('a'+i)),
			Title:     "Sort " + string(rune('A'+i)),
			CreatedAt: now - int64((3-i)*3600),
			UpdatedAt: now - int64((3-i)*3600),
		}
		if err := m.PutMetadata(ctx, meta); err != nil {
			t.Fatalf("PutMetadata[%d]: %v", i, err)
		}
	}

	envelope, err := GenerateRecap(ctx, m, RecapConfig{MaxEntries: 10})
	if err != nil {
		t.Fatalf("GenerateRecap: %v", err)
	}

	// Most recent first: C, B, A
	if len(envelope.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(envelope.Entries))
	}
	if !strings.HasSuffix(envelope.Entries[0].SessionID, "c") {
		t.Errorf("first entry should be most recent (c), got %s", envelope.Entries[0].SessionID)
	}
	if !strings.HasSuffix(envelope.Entries[2].SessionID, "a") {
		t.Errorf("last entry should be oldest (a), got %s", envelope.Entries[2].SessionID)
	}
}
