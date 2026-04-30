package session

import (
	"context"
	"errors"
	"testing"
)

// TestSessionTitleStore_TitleReturnsManualTrueWhenFlagSet proves that after
// PutMetadata with TitleManuallySet=true, Title() returns manual=true.
func TestSessionTitleStore_TitleReturnsManualTrueWhenFlagSet(t *testing.T) {
	ctx := context.Background()
	smap := NewMemMap()
	if err := smap.PutMetadata(ctx, Metadata{
		SessionID:        "sess-manual",
		Source:           "telegram",
		ChatID:           "42",
		Title:            "Operator Name",
		TitleManuallySet: true,
	}); err != nil {
		t.Fatalf("PutMetadata: %v", err)
	}

	store := NewMetadataTitleStore(ctx, smap)
	current, manual, ok, err := store.Title("sess-manual")
	if err != nil {
		t.Fatalf("Title: %v", err)
	}
	if !ok {
		t.Fatal("Title ok = false, want true")
	}
	if !manual {
		t.Fatal("manual = false, want true when TitleManuallySet=true")
	}
	if current != "Operator Name" {
		t.Fatalf("current = %q, want %q", current, "Operator Name")
	}
}

// TestSessionTitleStore_TitleReturnsManualFalseForLegacyRow proves that a row
// written without TitleManuallySet (or with false) returns manual=false.
func TestSessionTitleStore_TitleReturnsManualFalseForLegacyRow(t *testing.T) {
	ctx := context.Background()
	smap := NewMemMap()
	if err := smap.PutMetadata(ctx, Metadata{
		SessionID: "sess-auto",
		Source:    "telegram",
		ChatID:    "42",
		Title:     "Auto-generated name",
		// TitleManuallySet intentionally absent (false)
	}); err != nil {
		t.Fatalf("PutMetadata: %v", err)
	}

	store := NewMetadataTitleStore(ctx, smap)
	_, manual, ok, err := store.Title("sess-auto")
	if err != nil {
		t.Fatalf("Title: %v", err)
	}
	if !ok {
		t.Fatal("Title ok = false, want true")
	}
	if manual {
		t.Fatal("manual = true, want false for row without TitleManuallySet")
	}
}

// TestSessionTitleStore_SetTitleClearsManualFlag proves that SetTitle writes
// the new title with TitleManuallySet=false, satisfying the auto_title.go
// interface contract: "auto-titles are non-manual".
// Because mergeMetadata is sticky-true, SetTitle must use an explicit-clear
// path rather than relying on a plain PutMetadata with false.
func TestSessionTitleStore_SetTitleClearsManualFlag(t *testing.T) {
	ctx := context.Background()
	smap := NewMemMap()
	// Seed with a manual title.
	if err := smap.PutMetadata(ctx, Metadata{
		SessionID:        "sess-was-manual",
		Source:           "telegram",
		ChatID:           "42",
		Title:            "Operator Name",
		TitleManuallySet: true,
	}); err != nil {
		t.Fatalf("PutMetadata: %v", err)
	}

	store := NewMetadataTitleStore(ctx, smap)
	if err := store.SetTitle("sess-was-manual", "Auto Generated Title"); err != nil {
		t.Fatalf("SetTitle: %v", err)
	}

	// After SetTitle, the flag must be false (auto-title is non-manual).
	_, manual, ok, err := store.Title("sess-was-manual")
	if err != nil {
		t.Fatalf("Title after SetTitle: %v", err)
	}
	if !ok {
		t.Fatal("Title ok = false after SetTitle, want true")
	}
	if manual {
		t.Fatal("manual = true after SetTitle, want false (auto-title clears manual flag)")
	}
}

// TestSessionTitleStore_TitleReturnsOKFalseForUnknownSession confirms that
// Title() returns ok=false (not an error) for a session that does not exist.
func TestSessionTitleStore_TitleReturnsOKFalseForUnknownSession(t *testing.T) {
	ctx := context.Background()
	smap := NewMemMap()
	store := NewMetadataTitleStore(ctx, smap)
	_, _, ok, err := store.Title("no-such-session")
	if err != nil {
		t.Fatalf("Title: %v", err)
	}
	if ok {
		t.Fatal("ok = true for unknown session, want false")
	}
}

// errMap is a minimal sessionMetadataGetter that always returns an error, used
// to prove Title() propagates read errors from the underlying store.
type errMap struct{ err error }

func (e *errMap) GetMetadata(_ context.Context, _ string) (Metadata, bool, error) {
	return Metadata{}, false, e.err
}

// putMetadataDirecter is the write side used by SetTitle.
type putMetadataDirecter interface {
	PutMetadata(ctx context.Context, meta Metadata) error
}

// TestSessionTitleStore_TitleReturnsErrorOnStoreFailure proves that a read
// error from the underlying store propagates out of Title().
func TestSessionTitleStore_TitleReturnsErrorOnStoreFailure(t *testing.T) {
	ctx := context.Background()
	sentinel := errors.New("bolt: I/O error")
	store := NewMetadataTitleStoreFromReader(ctx, &errMap{err: sentinel})
	_, _, _, err := store.Title("any")
	if !errors.Is(err, sentinel) {
		t.Fatalf("Title err = %v, want sentinel bolt I/O error", err)
	}
}

// TestPerformAutoTitle_ManualShortCircuitViaConcreteAdapter is the integration
// fixture that wires PerformAutoTitle against MetadataTitleStore and proves
// the manual short-circuit (AutoTitleCodeSkippedManual) fires when the session
// was set with /title (TitleManuallySet=true) first.
func TestPerformAutoTitle_ManualShortCircuitViaConcreteAdapter(t *testing.T) {
	ctx := context.Background()
	smap := NewMemMap()
	if err := smap.PutMetadata(ctx, Metadata{
		SessionID:        "sess-operator",
		Source:           "telegram",
		ChatID:           "42",
		Title:            "Operator Chosen Title",
		TitleManuallySet: true,
	}); err != nil {
		t.Fatalf("PutMetadata: %v", err)
	}

	store := NewMetadataTitleStore(ctx, smap)

	providerCalled := false
	gen := TitleGenerator(func(_ context.Context, _ []TitleTurn) (string, error) {
		providerCalled = true
		return "Should Never Appear", nil
	})

	transcript := []TitleTurn{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}

	evidence := PerformAutoTitle(ctx, store, gen, "sess-operator", transcript)

	if evidence.Code != AutoTitleCodeSkippedManual {
		t.Fatalf("evidence.Code = %q, want %q", evidence.Code, AutoTitleCodeSkippedManual)
	}
	if providerCalled {
		t.Fatal("title generator was called despite manual short-circuit")
	}
	if evidence.Title != "Operator Chosen Title" {
		t.Fatalf("evidence.Title = %q, want %q", evidence.Title, "Operator Chosen Title")
	}
}
