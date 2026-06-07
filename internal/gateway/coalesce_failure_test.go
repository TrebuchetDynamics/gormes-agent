package gateway

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// failOnFinalEditor is a placeholderEditor whose EditMessage always succeeds
// for non-final calls but returns editErr on the final (finalize=true) call.
// It also implements coalescerMessageSender so the fallback plain-Send path
// is exercised.
type failOnFinalEditor struct {
	mu sync.Mutex

	// SendPlaceholder state
	nextMsgID int
	sent      []fakeSent

	// EditMessage state
	editErr error // returned for ALL edit calls when non-nil
	edits   []fakeEdit

	// Plain-Send state (coalescerMessageSender)
	plainSendErr error
	plainSends   []fakeSent
}

func newFailOnFinalEditor() *failOnFinalEditor {
	return &failOnFinalEditor{nextMsgID: 5000}
}

func (f *failOnFinalEditor) SendPlaceholder(_ context.Context, chatID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := "msg-" + itoa(f.nextMsgID)
	f.nextMsgID++
	f.sent = append(f.sent, fakeSent{ChatID: chatID, Text: "⏳", MsgID: id})
	return id, nil
}

func (f *failOnFinalEditor) EditMessage(_ context.Context, chatID, msgID, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.editErr != nil {
		return f.editErr
	}
	f.edits = append(f.edits, fakeEdit{ChatID: chatID, MsgID: msgID, Text: text})
	return nil
}

// EditMessageFinal implements FinalizingMessageEditor so the coalescer uses
// the finalize flag. It returns editErr only when finalize is true.
func (f *failOnFinalEditor) EditMessageFinal(_ context.Context, chatID, msgID, text string, finalize bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if finalize && f.editErr != nil {
		return f.editErr
	}
	f.edits = append(f.edits, fakeEdit{ChatID: chatID, MsgID: msgID, Text: text})
	return nil
}

// Send implements coalescerMessageSender (plain-Send fallback).
func (f *failOnFinalEditor) Send(_ context.Context, chatID, text string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.plainSendErr != nil {
		return "", f.plainSendErr
	}
	id := "msg-" + itoa(f.nextMsgID)
	f.nextMsgID++
	f.plainSends = append(f.plainSends, fakeSent{ChatID: chatID, Text: text, MsgID: id})
	return id, nil
}

func (f *failOnFinalEditor) plainSendsSnapshot() []fakeSent {
	f.mu.Lock()
	defer f.mu.Unlock()
	return cloneSlice(f.plainSends)
}

// itoa is a minimal helper so this file has no strconv import dependency.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}

// ─────────────────────────────────────────────────────────────────────────────
// Test 1: edit failure on finalize triggers one plain Send + evidence.
// ─────────────────────────────────────────────────────────────────────────────

func TestCoalescer_EditFailure_PlainSendFallback(t *testing.T) {
	fake := newFailOnFinalEditor()

	var mu sync.Mutex
	var evidences []CoalescerEvidence
	sink := func(ev CoalescerEvidence) {
		mu.Lock()
		evidences = append(evidences, ev)
		mu.Unlock()
	}

	c := newCoalescer(fake, time.Second, "chat42",
		coalescerEvidenceSink(sink),
	)

	// First, establish a placeholder so flushImmediateFinal has a msgID to edit.
	ctx := context.Background()
	c.flushImmediate(ctx, "streaming text")

	// Now finalize — edit will fail, fallback Send should fire.
	fake.editErr = errors.New("telegram: message can't be edited")
	c.flushImmediateFinal(ctx, "final answer", true)

	// Exactly one plain Send: the fallback final text after the terminal edit
	// fails. The initial stream preview used the placeholder/edit path.
	sends := fake.plainSendsSnapshot()
	if len(sends) != 1 {
		t.Fatalf("plain Send calls = %d, want 1; sends=%#v", len(sends), sends)
	}
	if sends[0].Text != "final answer" {
		t.Fatalf("fallback plain Send text = %q, want %q", sends[0].Text, "final answer")
	}
	if sends[0].ChatID != "chat42" {
		t.Fatalf("fallback plain Send chatID = %q, want %q", sends[0].ChatID, "chat42")
	}

	// Exactly one edit_failed_fallback evidence.
	mu.Lock()
	evCopy := make([]CoalescerEvidence, len(evidences))
	copy(evCopy, evidences)
	mu.Unlock()

	if len(evCopy) != 1 {
		t.Fatalf("evidence count = %d, want 1; evidences=%#v", len(evCopy), evCopy)
	}
	if evCopy[0].Code != "edit_failed_fallback" {
		t.Fatalf("evidence code = %q, want %q", evCopy[0].Code, "edit_failed_fallback")
	}
}

func TestCoalescer_FirstVisibleStreamingSendUsesContent(t *testing.T) {
	ch := newFakeChannel("telegram")
	c := newCoalescer(ch, time.Second, "chat42", coalescerInitialTextSend())

	c.flushImmediate(context.Background(), "partial response")

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("Send calls = %d, want 1 first visible content send; sent=%#v", len(sent), sent)
	}
	if sent[0].Text != "partial response" {
		t.Fatalf("first sent text = %q, want first stream content without hourglass placeholder", sent[0].Text)
	}
	if edits := ch.editsSnapshot(); len(edits) != 0 {
		t.Fatalf("initial visible content should be sent directly, edits=%#v", edits)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Test 2: concurrent flushImmediate + flushImmediateFinal → at most one final.
// ─────────────────────────────────────────────────────────────────────────────

func TestCoalescer_FinalizeRace_NoDuplicateMessage(t *testing.T) {
	// freshFinalFakeChannel already implements coalescerMessageSender (Send),
	// FinalizingMessageEditor (EditMessageFinal), and MessageDeleter.
	ch := &freshFinalFakeChannel{fakeChannel: newFakeChannel("race-test")}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := newCoalescer(ch, 50*time.Millisecond, "chatRace")
	go c.run(ctx)

	// Establish a placeholder so there is a live msgID.
	c.flushImmediate(ctx, "streaming")
	// Brief pause to let the initial placeholder + edit settle.
	waitFor(t, 500*time.Millisecond, func() bool {
		return c.currentMessageID() != ""
	})

	const targetText = "final race text"

	// Barrier: release both goroutines simultaneously.
	var ready sync.WaitGroup
	var go1, go2 sync.WaitGroup
	ready.Add(2)
	go1.Add(1)
	go2.Add(1)

	go func() {
		ready.Done()
		ready.Wait()
		c.setPending(targetText)
		go1.Done()
	}()
	go func() {
		ready.Done()
		ready.Wait()
		c.flushImmediateFinal(ctx, targetText, true)
		go2.Done()
	}()
	go1.Wait()
	go2.Wait()

	// Count messages that contain the target text (across both Send and Edit paths).
	sent := ch.sentSnapshot()
	edits := ch.editsSnapshot()

	finalSends := 0
	for _, s := range sent {
		if s.Text == targetText {
			finalSends++
		}
	}
	finalEdits := 0
	for _, e := range edits {
		if e.Text == targetText {
			finalEdits++
		}
	}
	total := finalSends + finalEdits
	if total > 1 {
		t.Fatalf("duplicate final messages: %d terminal-text message(s) delivered (sends=%d edits=%d)",
			total, finalSends, finalEdits)
	}
}

func TestCoalescer_FinalEditFailureDoesNotResendAlreadyVisibleFinal(t *testing.T) {
	fake := newFailOnFinalEditor()
	c := newCoalescer(fake, time.Second, "chat42", coalescerInitialTextSend())

	ctx := context.Background()
	c.flushImmediate(ctx, "already final")
	fake.editErr = errors.New("telegram: message can't be edited")
	c.flushImmediateFinal(ctx, "already final", true)

	sends := fake.plainSendsSnapshot()
	if len(sends) != 1 {
		t.Fatalf("plain Send calls = %d, want only initial visible message; sends=%#v", len(sends), sends)
	}
	if sends[0].Text != "already final" {
		t.Fatalf("initial visible text = %q, want %q", sends[0].Text, "already final")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Test 3: both edit AND fallback Send fail → send_final_failed evidence, no panic.
// ─────────────────────────────────────────────────────────────────────────────

func TestCoalescer_BothEditAndSendFail(t *testing.T) {
	fake := newFailOnFinalEditor()

	var mu sync.Mutex
	var evidences []CoalescerEvidence
	sink := func(ev CoalescerEvidence) {
		mu.Lock()
		evidences = append(evidences, ev)
		mu.Unlock()
	}

	c := newCoalescer(fake, time.Second, "chatBothFail",
		coalescerEvidenceSink(sink),
	)

	ctx := context.Background()
	c.flushImmediate(ctx, "streaming")
	fake.editErr = errors.New("telegram: message can't be edited")
	fake.plainSendErr = errors.New("telegram: flood wait")
	c.flushImmediateFinal(ctx, "final text", true) // must not panic

	mu.Lock()
	evCopy := make([]CoalescerEvidence, len(evidences))
	copy(evCopy, evidences)
	mu.Unlock()

	if len(evCopy) != 1 {
		t.Fatalf("evidence count = %d, want 1; evidences=%#v", len(evCopy), evCopy)
	}
	if evCopy[0].Code != "send_final_failed" {
		t.Fatalf("evidence code = %q, want %q", evCopy[0].Code, "send_final_failed")
	}

	// State must not be mutated: pendingMsgID should still be the original
	// visible streaming message.
	msgID := c.currentMessageID()
	if msgID == "" {
		t.Fatal("currentMessageID is empty after both-fail: state was unexpectedly cleared")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Test 4: fresh-final-after behavior is unchanged (regression guard).
// ─────────────────────────────────────────────────────────────────────────────

func TestCoalescer_FreshFinalAfter_StillRespected(t *testing.T) {
	ch := &freshFinalFakeChannel{fakeChannel: newFakeChannel("fresh-final-regression")}
	now := time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC)
	c := newCoalescer(ch, time.Second, "chatFF",
		coalescerFreshFinalAfter(time.Minute),
		coalescerNow(func() time.Time { return now }),
	)

	ctx := context.Background()
	c.flushImmediate(ctx, "preview")
	oldID := c.currentMessageID()

	// Advance clock past the threshold.
	now = now.Add(2 * time.Minute)
	c.flushImmediateFinal(ctx, "fresh final", true)

	// Two Send calls: initial visible preview + fresh final.
	sent := ch.sentSnapshot()
	if len(sent) != 2 {
		t.Fatalf("Send calls = %d, want 2 (preview + fresh final); sent=%#v", len(sent), sent)
	}
	if sent[1].Text != "fresh final" {
		t.Fatalf("fresh final text = %q, want %q", sent[1].Text, "fresh final")
	}

	// currentMessageID updated to the fresh message.
	if got := c.currentMessageID(); got != sent[1].MsgID {
		t.Fatalf("currentMessageID = %q, want fresh id %q", got, sent[1].MsgID)
	}

	// Old placeholder was deleted.
	deletes := ch.deletesSnapshot()
	if len(deletes) != 1 || deletes[0].MsgID != oldID {
		t.Fatalf("DeleteMessage calls = %#v, want one delete of %q", deletes, oldID)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Test 5: on successful edit, no evidence is emitted.
// ─────────────────────────────────────────────────────────────────────────────

func TestCoalescer_NoEvidenceOnSuccess(t *testing.T) {
	ch := &freshFinalFakeChannel{fakeChannel: newFakeChannel("no-evidence")}

	var mu sync.Mutex
	var evidences []CoalescerEvidence
	sink := func(ev CoalescerEvidence) {
		mu.Lock()
		evidences = append(evidences, ev)
		mu.Unlock()
	}

	c := newCoalescer(ch, time.Second, "chatOK",
		coalescerEvidenceSink(sink),
	)

	ctx := context.Background()
	c.flushImmediate(ctx, "preview")
	c.flushImmediateFinal(ctx, "final answer", true)

	mu.Lock()
	count := len(evidences)
	mu.Unlock()

	if count != 0 {
		t.Fatalf("evidence count = %d, want 0 on success; evidences=%#v", count, evidences)
	}
}
