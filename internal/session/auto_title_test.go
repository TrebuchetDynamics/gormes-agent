package session

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// fakeTitleStore is the minimal SessionTitleStore used by auto-title tests.
// It records every read and write so each test can assert the exact number
// of provider calls and persistence writes performed.
type fakeTitleStore struct {
	current string
	manual  bool
	exists  bool

	getErr error
	setErr error

	titleReads int
	setCalls   []string
}

func (f *fakeTitleStore) Title(sessionID string) (string, bool, bool, error) {
	f.titleReads++
	if f.getErr != nil {
		return "", false, false, f.getErr
	}
	return f.current, f.manual, f.exists, nil
}

func (f *fakeTitleStore) SetTitle(sessionID, title string) error {
	f.setCalls = append(f.setCalls, title)
	if f.setErr != nil {
		return f.setErr
	}
	f.current = title
	f.manual = false
	f.exists = true
	return nil
}

// countingGenerator wraps a TitleGenerator and counts invocations so tests
// can prove the helper performs exactly one provider call per invocation
// (no retry storms on failure).
type countingGenerator struct {
	calls  int
	title  string
	err    error
	custom func(ctx context.Context, transcript []TitleTurn) (string, error)
}

func (c *countingGenerator) generate(ctx context.Context, transcript []TitleTurn) (string, error) {
	c.calls++
	if c.custom != nil {
		return c.custom(ctx, transcript)
	}
	return c.title, c.err
}

func TestAutoTitleSession_WritesOnceForUntitledSession(t *testing.T) {
	t.Parallel()

	store := &fakeTitleStore{exists: false}
	gen := &countingGenerator{title: "Bounded Prompt Title"}
	transcript := []TitleTurn{
		{Role: "user", Content: "first user turn"},
		{Role: "assistant", Content: "first assistant reply"},
	}

	evidence := PerformAutoTitle(context.Background(), store, gen.generate, "session-id", transcript)

	if evidence.Code != "auto_title_complete" {
		t.Fatalf("evidence.Code = %q; want %q", evidence.Code, "auto_title_complete")
	}
	if evidence.Title != "Bounded Prompt Title" {
		t.Fatalf("evidence.Title = %q; want %q", evidence.Title, "Bounded Prompt Title")
	}
	if gen.calls != 1 {
		t.Fatalf("generator calls = %d; want 1", gen.calls)
	}
	if len(store.setCalls) != 1 {
		t.Fatalf("set calls = %d; want exactly 1 persistence write", len(store.setCalls))
	}
	if store.setCalls[0] != "Bounded Prompt Title" {
		t.Fatalf("persisted title = %q; want %q", store.setCalls[0], "Bounded Prompt Title")
	}
	if store.current != "Bounded Prompt Title" {
		t.Fatalf("store.current after write = %q; want %q", store.current, "Bounded Prompt Title")
	}
}

func TestAutoTitleSession_ManualTitlePreserved(t *testing.T) {
	t.Parallel()

	store := &fakeTitleStore{
		current: "user picked title",
		manual:  true,
		exists:  true,
	}
	gen := &countingGenerator{title: "auto override should never appear"}
	transcript := []TitleTurn{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "hello"},
	}

	evidence := PerformAutoTitle(context.Background(), store, gen.generate, "session-id", transcript)

	if evidence.Code != "auto_title_skipped_manual" {
		t.Fatalf("evidence.Code = %q; want %q", evidence.Code, "auto_title_skipped_manual")
	}
	if gen.calls != 0 {
		t.Fatalf("generator calls = %d; want 0 (manual short-circuit)", gen.calls)
	}
	if len(store.setCalls) != 0 {
		t.Fatalf("set calls = %d; want 0 (manual title must not be overwritten)", len(store.setCalls))
	}
	if store.current != "user picked title" {
		t.Fatalf("store.current = %q; want preserved manual title %q", store.current, "user picked title")
	}
}

func TestAutoTitleSession_FailureEvidenceNoRetryStorm(t *testing.T) {
	t.Parallel()

	providerErr := errors.New("openrouter 402: credits exhausted")
	store := &fakeTitleStore{exists: false}
	gen := &countingGenerator{err: providerErr}
	transcript := []TitleTurn{
		{Role: "user", Content: "any prompt"},
		{Role: "assistant", Content: "any reply"},
	}

	evidence := PerformAutoTitle(context.Background(), store, gen.generate, "session-id", transcript)

	if evidence.Code != "title_provider_failed" {
		t.Fatalf("evidence.Code = %q; want %q", evidence.Code, "title_provider_failed")
	}
	if evidence.Reason == "" {
		t.Fatalf("evidence.Reason is empty; want provider failure detail")
	}
	if gen.calls != 1 {
		t.Fatalf("generator calls = %d; want exactly 1 (no retry storm)", gen.calls)
	}
	if len(store.setCalls) != 0 {
		t.Fatalf("set calls = %d; want 0 (no title persisted on provider failure)", len(store.setCalls))
	}
	if evidence.Title != "" {
		t.Fatalf("evidence.Title = %q; want empty title on failure", evidence.Title)
	}
}

func TestAutoTitleSession_DoesNotMutateTranscript(t *testing.T) {
	t.Parallel()

	store := &fakeTitleStore{exists: false}
	gen := &countingGenerator{title: "Transcript Safety Check"}

	transcript := []TitleTurn{
		{Role: "user", Content: "first user turn"},
		{Role: "assistant", Content: "first assistant reply"},
	}
	before := append([]TitleTurn(nil), transcript...)
	beforeBytes := make([][]byte, len(transcript))
	for i, turn := range transcript {
		beforeBytes[i] = []byte(turn.Content)
	}

	gen.custom = func(ctx context.Context, ts []TitleTurn) (string, error) {
		// Generator must not mutate any caller-owned transcript bytes.
		// Tests treat the transcript as read-only input.
		return "Transcript Safety Check", nil
	}

	evidence := PerformAutoTitle(context.Background(), store, gen.generate, "session-id", transcript)

	if evidence.Code != "auto_title_complete" {
		t.Fatalf("evidence.Code = %q; want %q", evidence.Code, "auto_title_complete")
	}

	if !reflect.DeepEqual(before, transcript) {
		t.Fatalf("transcript mutated:\n  before = %#v\n   after = %#v", before, transcript)
	}
	for i, turn := range transcript {
		if turn.Content != string(beforeBytes[i]) {
			t.Fatalf("transcript[%d].Content mutated: before=%q after=%q",
				i, string(beforeBytes[i]), turn.Content)
		}
	}
}
