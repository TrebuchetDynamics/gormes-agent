package coalescing

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

type PlaceholderEditor interface {
	SendPlaceholder(ctx context.Context, chatID string) (msgID string, err error)
	EditMessage(ctx context.Context, chatID, msgID, text string) error
}

type MessageSender interface {
	Send(ctx context.Context, chatID, text string) (msgID string, err error)
}

// CoalescerEvidence carries a redacted operator signal from the coalescer.
// It never contains raw chatIDs, API response bodies, or credentials.
type Evidence struct {
	// Code is a stable machine-readable key:
	//   "edit_failed_fallback" – edit error on finalize; plain Send succeeded.
	//   "send_final_failed"    – edit error on finalize AND fallback Send failed.
	Code string
	// Message is a human-readable summary. It is always redacted; it must
	// never include raw chatIDs, API response bodies, or API keys.
	Message string
}

// CoalescerEvidenceSink receives CoalescerEvidence for non-happy-path finalize
// outcomes. The sink must not block or panic; panics are not recovered here —
// callers must ensure the sink is safe to call from any goroutine.
type EvidenceSink func(Evidence)

var errNoSender = errors.New("coalescer sender is not configured")

func EvidenceSinkOption(sink EvidenceSink) Option {
	return func(c *Coalescer) {
		if sink != nil {
			c.evidenceSink = sink
		}
	}
}

type Option func(*Coalescer)

func FreshFinalAfter(d time.Duration) Option {
	return func(c *Coalescer) {
		c.freshFinalAfter = d
	}
}

func Now(now func() time.Time) Option {
	return func(c *Coalescer) {
		if now != nil {
			c.now = now
		}
	}
}

func InitialTextSend() Option {
	return func(c *Coalescer) {
		c.initialTextSend = true
	}
}

// StreamCursor sets the cursor suffix appended to streaming preview messages.
// When a terminal edit fails and the last visible text was the full answer
// with the cursor appended, the coalescer treats the content as already
// delivered (suppressing a duplicate full-send). Mirrors Hermes
// fix(gateway): suppress duplicate final stream sends (197337cc4).
func StreamCursor(cursor string) Option {
	return func(c *Coalescer) {
		c.streamCursor = cursor
	}
}

// coalescer batches outbound edits for one turn. The manager owns one
// instance per active turn and tears it down on terminal phases.
type Coalescer struct {
	sender       PlaceholderEditor
	window       time.Duration
	chatID       string
	now          func() time.Time
	evidenceSink EvidenceSink
	streamCursor string // suffix appended to streaming previews; used to detect already-visible finals

	// deliveryMu serializes visible send/edit attempts. State snapshots are
	// only valid for a delivery decision while this lock is held; otherwise a
	// final flush can race an in-flight initial placeholder and create a second
	// visible message for the same turn.
	deliveryMu       sync.Mutex
	mu               sync.Mutex
	pendingText      string
	pendingMsgID     string
	messageCreatedAt time.Time
	lastSentText     string
	lastEditAt       time.Time
	retryAfter       time.Time
	freshFinalAfter  time.Duration
	initialTextSend  bool
	wakeupCh         chan struct{}
}

func New(pe PlaceholderEditor, window time.Duration, chatID string, opts ...Option) *Coalescer {
	if window <= 0 {
		window = time.Second
	}
	c := &Coalescer{
		sender:   pe,
		window:   window,
		chatID:   chatID,
		now:      time.Now,
		wakeupCh: make(chan struct{}, 1),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Coalescer) emitEvidence(code, message string) {
	if c.evidenceSink != nil {
		c.evidenceSink(Evidence{Code: code, Message: message})
	}
}

func (c *Coalescer) clearPendingAfterDeliveryLocked(deliveredText string, terminal bool) {
	if terminal || c.pendingText == deliveredText {
		c.pendingText = ""
	}
}

func (c *Coalescer) SetPending(text string) {
	c.mu.Lock()
	c.pendingText = text
	c.mu.Unlock()

	select {
	case c.wakeupCh <- struct{}{}:
	default:
	}
}

func (c *Coalescer) CurrentMessageID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.pendingMsgID
}

func (c *Coalescer) FlushImmediate(ctx context.Context, text string) {
	c.FlushImmediateFinal(ctx, text, false)
}

func (c *Coalescer) FlushImmediateFinal(ctx context.Context, text string, finalize bool) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return
	}
	c.deliveryMu.Lock()
	defer c.deliveryMu.Unlock()
	if err := ctx.Err(); err != nil {
		return
	}

	state := c.finalDeliverySnapshot()
	msgID := state.msgID
	createdAt := state.createdAt
	freshFinalAfter := state.freshFinalAfter
	lastSentText := state.lastSentText

	now := c.now()
	if shouldSendFreshFinal(finalize, msgID, createdAt, freshFinalAfter, now, lastSentText, text) {
		if sentID, ok := c.tryFreshFinal(ctx, msgID, text); ok {
			c.mu.Lock()
			c.pendingMsgID = sentID
			c.messageCreatedAt = now
			c.lastSentText = text
			c.lastEditAt = now
			c.clearPendingAfterDeliveryLocked(text, finalize)
			c.mu.Unlock()
			return
		}
	}

	var sentID string
	var err error
	if msgID == "" {
		sentAt := c.now()
		sentID, err = c.sendInitialVisibleMessage(ctx, text, finalize)
		if err == nil || sentID != "" {
			createdAt = sentAt
		}
	} else {
		sentID = msgID
		err = editCoalescedMessage(ctx, c.sender, c.chatID, msgID, text, finalize)
	}

	if err != nil {
		// Mid-stream (non-finalize) edit errors are expected during streaming
		// throttle (e.g., Telegram rate limits). Keep the silent-swallow
		// behavior for those; only act on finalize errors.
		if sentID != "" {
			c.mu.Lock()
			if c.pendingMsgID == "" {
				c.pendingMsgID = sentID
				c.messageCreatedAt = createdAt
			}
			c.mu.Unlock()
		}
		if !finalize {
			return
		}
		// Hermes suppresses a second final delivery when the streamed preview
		// already contains the completed answer. If the terminal edit fails here,
		// a plain Send would duplicate the visible reply. Also suppress when the
		// preview was the answer plus the streaming cursor (e.g. "answer ▉") and
		// the final text is just "answer" — the user already sees the full
		// answer, only the cursor is stuck on screen. Mirrors Hermes
		// fix(gateway): suppress duplicate final stream sends (197337cc4).
		cursorStrippedMatch := c.streamCursor != "" &&
			strings.HasSuffix(lastSentText, c.streamCursor) &&
			strings.TrimSuffix(lastSentText, c.streamCursor) == text
		if lastSentText == text || cursorStrippedMatch {
			c.mu.Lock()
			c.clearPendingAfterDeliveryLocked(text, finalize)
			c.mu.Unlock()
			return
		}
		// Finalize edit failed. Attempt one plain Send as a fallback so the
		// user receives the final text even if the placeholder can no longer
		// be edited (e.g., Telegram "message can't be edited").
		//
		// Race-window note: we do NOT re-acquire c.mu here before calling
		// Send. The contract is: at most one flushImmediateFinal(finalize=true)
		// is ever called per coalescer lifetime (the manager cancels the
		// coalescer immediately after). The race test (TestCoalescer_FinalizeRace_NoDuplicateMessage)
		// exercises concurrent setPending + flushImmediateFinal; those paths
		// write to different fields (pendingText vs lastSentText/pendingMsgID),
		// so the final-text guard (lastSentText == text) in tryFlush prevents
		// a duplicate even if tryFlush fires concurrently with this fallback.
		sender, hasSender := c.sender.(MessageSender)
		if !hasSender {
			c.emitEvidence("send_final_failed", "edit failed on finalize; no plain-sender available")
			return
		}
		newMsgID, sendErr := sender.Send(ctx, c.chatID, text)
		if sendErr != nil {
			c.emitEvidence("send_final_failed", "edit failed on finalize; fallback Send also failed")
			// Do not mutate state: leave pendingMsgID intact so the caller
			// knows something was attempted. No retry, no loop.
			return
		}
		c.emitEvidence("edit_failed_fallback", "edit failed on finalize; plain Send fallback succeeded")
		c.mu.Lock()
		c.pendingMsgID = newMsgID
		c.lastSentText = text
		c.lastEditAt = c.now()
		c.clearPendingAfterDeliveryLocked(text, finalize)
		c.mu.Unlock()
		return
	}

	c.mu.Lock()
	if c.pendingMsgID == "" {
		c.pendingMsgID = sentID
		c.messageCreatedAt = createdAt
	}
	c.lastSentText = text
	c.lastEditAt = c.now()
	c.clearPendingAfterDeliveryLocked(text, finalize)
	c.mu.Unlock()
}

func (c *Coalescer) sendInitialVisibleMessage(ctx context.Context, text string, finalize bool) (string, error) {
	if c.sender == nil {
		return "", errNoSender
	}
	if c.initialTextSend {
		if msgID, err, ok := c.sendInitialText(ctx, text); ok {
			if err == nil {
				return msgID, nil
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return "", err
			}
		}
	}
	msgID, err := c.sender.SendPlaceholder(ctx, c.chatID)
	if err != nil {
		return "", err
	}
	if err := editCoalescedMessage(ctx, c.sender, c.chatID, msgID, text, finalize); err != nil {
		return msgID, err
	}
	return msgID, nil
}

func (c *Coalescer) Run(ctx context.Context) {
	ticker := time.NewTicker(c.window)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.tryFlush(ctx)
		case <-c.wakeupCh:
			c.tryFlush(ctx)
		}
	}
}

func (c *Coalescer) tryFlush(ctx context.Context) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return
		}
	}
	c.deliveryMu.Lock()
	defer c.deliveryMu.Unlock()

	state := c.pendingDeliverySnapshot()
	text := state.text
	msgID := state.msgID
	last := state.lastSentText
	lastAt := state.lastEditAt
	retryAfter := state.retryAfter

	if text == "" || text == last {
		return
	}
	now := c.now()
	if now.Before(retryAfter) {
		return
	}
	if msgID != "" && now.Sub(lastAt) < c.window {
		return
	}

	if msgID == "" {
		sentID, err := c.sendInitialVisibleMessage(ctx, text, false)
		if err != nil {
			if sentID != "" {
				c.mu.Lock()
				if c.pendingMsgID == "" {
					c.pendingMsgID = sentID
					c.messageCreatedAt = now
				}
				c.mu.Unlock()
			}
			return
		}
		c.mu.Lock()
		if c.pendingMsgID == "" {
			c.pendingMsgID = sentID
			c.messageCreatedAt = now
		}
		c.lastSentText = text
		c.lastEditAt = now
		c.clearPendingAfterDeliveryLocked(text, false)
		c.mu.Unlock()
		return
	}

	if err := editCoalescedMessage(ctx, c.sender, c.chatID, msgID, text, false); err != nil {
		return
	}

	c.mu.Lock()
	c.lastSentText = text
	c.lastEditAt = now
	c.clearPendingAfterDeliveryLocked(text, false)
	c.mu.Unlock()
}

type finalDeliverySnapshot struct {
	msgID           string
	createdAt       time.Time
	freshFinalAfter time.Duration
	lastSentText    string
}

type pendingDeliverySnapshot struct {
	text         string
	msgID        string
	lastSentText string
	lastEditAt   time.Time
	retryAfter   time.Time
}

func (c *Coalescer) finalDeliverySnapshot() finalDeliverySnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return finalDeliverySnapshot{
		msgID:           c.pendingMsgID,
		createdAt:       c.messageCreatedAt,
		freshFinalAfter: c.freshFinalAfter,
		lastSentText:    c.lastSentText,
	}
}

func (c *Coalescer) pendingDeliverySnapshot() pendingDeliverySnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return pendingDeliverySnapshot{
		text:         c.pendingText,
		msgID:        c.pendingMsgID,
		lastSentText: c.lastSentText,
		lastEditAt:   c.lastEditAt,
		retryAfter:   c.retryAfter,
	}
}

func (c *Coalescer) sendInitialText(ctx context.Context, text string) (string, error, bool) {
	sender, ok := c.sender.(MessageSender)
	if !ok {
		return "", nil, false
	}
	msgID, err := sender.Send(ctx, c.chatID, text)
	return msgID, err, true
}

func shouldSendFreshFinal(finalize bool, msgID string, createdAt time.Time, threshold time.Duration, now time.Time, lastSentText, finalText string) bool {
	if !finalize || threshold <= 0 || msgID == "" || createdAt.IsZero() {
		return false
	}
	// Fresh finals are for replacing an old visible preview with newer terminal
	// content. If the terminal text is already visible, sending a new message can
	// duplicate the answer on channels that cannot delete the old preview.
	if finalText != "" && finalText == lastSentText {
		return false
	}
	return now.Sub(createdAt) >= threshold
}

func (c *Coalescer) tryFreshFinal(ctx context.Context, oldMsgID, text string) (string, bool) {
	sender, ok := c.sender.(MessageSender)
	if !ok {
		return "", false
	}
	msgID, err := sender.Send(ctx, c.chatID, text)
	if err != nil {
		return "", false
	}
	if deleter, ok := c.sender.(messageDeleter); ok {
		_ = deleter.DeleteMessage(ctx, c.chatID, oldMsgID)
	}
	return msgID, true
}

type finalizingMessageEditor interface {
	EditMessageFinal(ctx context.Context, chatID, msgID, text string, finalize bool) error
}

type messageDeleter interface {
	DeleteMessage(ctx context.Context, chatID, msgID string) error
}

func editCoalescedMessage(ctx context.Context, sender PlaceholderEditor, chatID, msgID, text string, finalize bool) error {
	if sender == nil {
		return errNoSender
	}
	if finalizer, ok := sender.(finalizingMessageEditor); ok {
		return finalizer.EditMessageFinal(ctx, chatID, msgID, text, finalize)
	}
	return sender.EditMessage(ctx, chatID, msgID, text)
}
