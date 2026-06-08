package gateway

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

// statusReplyChannel records both Send and SendReply targets so tests can
// assert that /status threads its outbound message as a Telegram-style reply
// to the triggering inbound message.
type statusReplyChannel struct {
	*fakeChannel

	replyTo []string
}

func (c *statusReplyChannel) SendReply(ctx context.Context, chatID, replyToMsgID, text string) (string, error) {
	c.replyTo = append(c.replyTo, replyToMsgID)
	return c.fakeChannel.Send(ctx, chatID, text)
}

type statusCommandHarness struct {
	ctx        context.Context
	kernel     *fakeKernel
	sessionMap *session.MemMap
	manager    *Manager
	channel    *fakeChannel
	now        time.Time
}

func newStatusCommandHarness(t *testing.T, smap *session.MemMap, now time.Time, opts ...func(*ManagerConfig)) *statusCommandHarness {
	t.Helper()
	if smap == nil {
		smap = session.NewMemMap()
	}
	if now.IsZero() {
		now = time.Date(2026, 4, 29, 9, 42, 0, 0, time.UTC)
	}
	k := &fakeKernel{}
	cfg := ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		Now:          func() time.Time { return now },
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	m := NewManagerWithSubmitter(cfg, k, slog.Default())
	ch := newFakeChannel("telegram")
	if err := m.Register(ch); err != nil {
		t.Fatal(err)
	}
	return &statusCommandHarness{
		ctx:        context.Background(),
		kernel:     k,
		sessionMap: smap,
		manager:    m,
		channel:    ch,
		now:        now,
	}
}

func TestParseInboundTextStatus(t *testing.T) {
	kind, body := ParseInboundText("/status")
	if kind != EventStatus || body != "" {
		t.Fatalf("ParseInboundText(/status) = (%v, %q), want EventStatus empty body", kind, body)
	}
	cmd, ok := ResolveCommand("/status")
	if !ok {
		t.Fatal("/status did not resolve through gateway CommandRegistry")
	}
	if cmd.Kind != EventStatus || cmd.ActiveTurnPolicy != CommandActiveTurnPolicyImmediate {
		t.Fatalf("/status command = (%v, %q), want EventStatus immediate", cmd.Kind, cmd.ActiveTurnPolicy)
	}
}

func TestTitleCommand_SetsSessionMetadataAndStatusRendersIt(t *testing.T) {
	ctx := context.Background()
	smap := session.NewMemMap()
	now := time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC)
	if err := smap.Put(ctx, "telegram:42", "sess-title-command"); err != nil {
		t.Fatalf("seed session map: %v", err)
	}
	fk := &fakeKernel{}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		Now:          func() time.Time { return now },
	}, fk, slog.Default())
	ch := newFakeChannel("telegram")
	if err := m.Register(ch); err != nil {
		t.Fatal(err)
	}

	ev := InboundEvent{Platform: "telegram", ChatID: "42", MsgID: "99", Kind: EventTitle, Text: "/title   Friendly\nGreeting\twith Juan  "}
	if err := m.handleInbound(ctx, ev); err != nil {
		t.Fatal(err)
	}

	meta, ok, err := smap.GetMetadata(ctx, "sess-title-command")
	if err != nil {
		t.Fatalf("get metadata: %v", err)
	}
	if !ok {
		t.Fatal("metadata was not persisted")
	}
	if meta.Title != "Friendly Greeting with Juan" {
		t.Fatalf("metadata title = %q, want sanitized title", meta.Title)
	}
	if meta.Source != "telegram" || meta.ChatID != "42" {
		t.Fatalf("metadata source/chat = %q/%q, want telegram/42", meta.Source, meta.ChatID)
	}
	sent := ch.sentSnapshot()
	if len(sent) != 1 || !strings.Contains(sent[0].Text, "Session title set: Friendly Greeting with Juan") {
		t.Fatalf("title confirmation not sent: %#v", sent)
	}
	if got := fk.submitsSnapshot(); len(got) != 0 {
		t.Fatalf("/title reached provider/kernel submits: %#v", got)
	}

	if err := m.handleInbound(ctx, InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventTitle, Text: "/title"}); err != nil {
		t.Fatal(err)
	}
	sent = ch.sentSnapshot()
	if len(sent) != 2 || !strings.Contains(sent[1].Text, "Title: Friendly Greeting with Juan") {
		t.Fatalf("current title response missing title: %#v", sent)
	}

	if err := m.handleInbound(ctx, InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventStatus}); err != nil {
		t.Fatal(err)
	}
	sent = ch.sentSnapshot()
	if len(sent) != 3 {
		t.Fatalf("sent count = %d, want 3: %#v", len(sent), sent)
	}
	if !strings.Contains(sent[2].Text, "**Title:** Friendly Greeting with Juan") {
		t.Fatalf("status did not render manual title:\n%s", sent[2].Text)
	}
}

func TestTitleCommand_InvalidTitleReturnsGuidanceWithoutMutation(t *testing.T) {
	ctx := context.Background()
	smap := session.NewMemMap()
	if err := smap.Put(ctx, "telegram:42", "sess-title-invalid"); err != nil {
		t.Fatalf("seed session map: %v", err)
	}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		Now:          func() time.Time { return time.Date(2026, 4, 29, 11, 45, 0, 0, time.UTC) },
	}, &fakeKernel{}, slog.Default())
	ch := newFakeChannel("telegram")
	if err := m.Register(ch); err != nil {
		t.Fatal(err)
	}

	if err := m.handleInbound(ctx, InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventTitle, Text: "/title " + strings.Repeat("x", 101)}); err != nil {
		t.Fatal(err)
	}
	meta, ok, err := smap.GetMetadata(ctx, "sess-title-invalid")
	if err != nil {
		t.Fatalf("get metadata: %v", err)
	}
	if ok && meta.Title != "" {
		t.Fatalf("invalid title mutated metadata title to %q", meta.Title)
	}
	sent := ch.sentSnapshot()
	if len(sent) != 1 || !strings.Contains(sent[0].Text, "Title too long (101 chars, max 100)") {
		t.Fatalf("invalid title guidance not sent: %#v", sent)
	}
}

func TestParseInboundTextTitlePreservesPayload(t *testing.T) {
	kind, body := ParseInboundText("/title Friendly Greeting")
	if kind != EventTitle || body != "/title Friendly Greeting" {
		t.Fatalf("ParseInboundText(/title) = (%v, %q), want EventTitle with original payload", kind, body)
	}
	cmd, ok := ResolveCommand("/title")
	if !ok {
		t.Fatal("/title did not resolve through gateway CommandRegistry")
	}
	if cmd.Kind != EventTitle || cmd.ActiveTurnPolicy != CommandActiveTurnPolicyImmediate {
		t.Fatalf("/title command = (%v, %q), want EventTitle immediate", cmd.Kind, cmd.ActiveTurnPolicy)
	}
}

// TestStatusCommand_RendersAllRequiredFields locks in the Hermes-compatible
// field order and labels documented in the row contract: Session ID, Title,
// Created, Last Activity, Tokens, Agent Running, Connected Platforms — each
// rendered with **bold** MarkdownV2 labels so Telegram displays them as bold.
func TestStatusCommand_PersistsAndRendersAccumulatedSessionTokenTotals(t *testing.T) {
	ctx := context.Background()
	smap := session.NewMemMap()
	now := time.Date(2026, 4, 29, 9, 42, 0, 0, time.UTC)
	if err := smap.Put(ctx, "telegram:42", "sess-token-totals"); err != nil {
		t.Fatalf("seed session map: %v", err)
	}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		Now:          func() time.Time { return now },
	}, &fakeKernel{}, slog.Default())
	ch := newFakeChannel("telegram")
	if err := m.Register(ch); err != nil {
		t.Fatal(err)
	}

	m.rememberUsageFrame(kernel.RenderFrame{
		SessionID: "sess-token-totals",
		Telemetry: telemetry.Snapshot{TokensInTotal: 3, TokensOutTotal: 4},
	})
	m.rememberUsageFrame(kernel.RenderFrame{
		SessionID: "sess-token-totals",
		Telemetry: telemetry.Snapshot{TokensInTotal: 600, TokensOutTotal: 634},
	})
	// Missing usage must not erase the durable total.
	m.rememberUsageFrame(kernel.RenderFrame{SessionID: "sess-token-totals"})

	meta, ok, err := smap.GetMetadata(ctx, "sess-token-totals")
	if err != nil {
		t.Fatalf("get metadata: %v", err)
	}
	if !ok {
		t.Fatal("usage metadata was not persisted")
	}
	if meta.TokensInTotal != 600 || meta.TokensOutTotal != 634 {
		t.Fatalf("metadata token totals = %d/%d, want 600/634", meta.TokensInTotal, meta.TokensOutTotal)
	}

	if err := m.handleInbound(ctx, InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventStatus}); err != nil {
		t.Fatal(err)
	}
	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1: %#v", len(sent), sent)
	}
	wantTokens := "**Cumulative API tokens (re-sent each call):** 1,234"
	if !strings.Contains(sent[0].Text, wantTokens) {
		t.Fatalf("status did not render Hermes token line %q:\n%s", wantTokens, sent[0].Text)
	}
}

func TestStatusCommand_RendersAllRequiredFields(t *testing.T) {
	ctx := context.Background()
	k := &fakeKernel{}
	smap := session.NewMemMap()
	now := time.Date(2026, 4, 29, 9, 42, 0, 0, time.UTC)
	if err := smap.Put(ctx, "telegram:42", "sess-123"); err != nil {
		t.Fatalf("seed session map: %v", err)
	}
	if err := smap.PutMetadata(ctx, session.Metadata{
		SessionID: "sess-123",
		Source:    "telegram",
		ChatID:    "42",
		Title:     "Friendly Greeting with Juan",
		CreatedAt: now.Add(-2 * time.Hour).Unix(),
		UpdatedAt: now.Unix(),
	}); err != nil {
		t.Fatalf("seed metadata: %v", err)
	}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		Now:          func() time.Time { return now },
	}, k, slog.Default())
	ch := newFakeChannel("telegram")
	if err := m.Register(ch); err != nil {
		t.Fatal(err)
	}
	m.rememberUsageFrame(kernel.RenderFrame{
		SessionID: "sess-123",
		Telemetry: telemetry.Snapshot{TokensInTotal: 3, TokensOutTotal: 4},
	})

	ev := InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventStatus}
	if err := m.handleInbound(ctx, ev); err != nil {
		t.Fatal(err)
	}

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1: %#v", len(sent), sent)
	}
	got := sent[0].Text

	// Required field labels in order.
	wantOrder := []string{
		"**Session ID:**",
		"**Title:**",
		"**Created:**",
		"**Last Activity:**",
		"**Cumulative API tokens (re-sent each call):**",
		"**Agent Running:**",
		"**Connected Platforms:**",
	}
	prev := -1
	for _, label := range wantOrder {
		idx := strings.Index(got, label)
		if idx < 0 {
			t.Fatalf("status response missing %q in:\n%s", label, got)
		}
		if idx <= prev {
			t.Fatalf("status field %q out of order in:\n%s", label, got)
		}
		prev = idx
	}
}

func TestStatusCommand_RendersHermesRunningIndicator(t *testing.T) {
	h := newStatusCommandHarness(t, nil, time.Time{})
	if err := h.sessionMap.Put(h.ctx, "telegram:42", "sess-running"); err != nil {
		t.Fatalf("seed session map: %v", err)
	}
	h.manager.pinTurn("telegram", "42", "running-message")

	if err := h.manager.handleInbound(h.ctx, InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventStatus}); err != nil {
		t.Fatal(err)
	}

	sent := h.channel.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1: %#v", len(sent), sent)
	}
	if !strings.Contains(sent[0].Text, "**Agent Running:** Yes ⚡") {
		t.Fatalf("status response missing Hermes running marker in:\n%s", sent[0].Text)
	}
}

func TestStatusCommand_RendersQueuedFollowUps(t *testing.T) {
	h := newStatusCommandHarness(t, nil, time.Time{})
	if err := h.sessionMap.Put(h.ctx, "telegram:42", "sess-queued-status"); err != nil {
		t.Fatalf("seed session map: %v", err)
	}
	h.manager.pinTurn("telegram", "42", "running-message")

	if err := h.manager.handleInbound(h.ctx, InboundEvent{Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "queue-message", Kind: EventSubmit, Text: "/queue run this after the current turn"}); err != nil {
		t.Fatal(err)
	}
	if err := h.manager.handleInbound(h.ctx, InboundEvent{Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "status-message", Kind: EventStatus, Text: "/status"}); err != nil {
		t.Fatal(err)
	}

	sent := h.channel.sentSnapshot()
	if len(sent) != 2 {
		t.Fatalf("sent count = %d, want queue ack and status: %#v", len(sent), sent)
	}
	got := sent[1].Text
	assertContainsAll(t, got, "**Agent Running:** Yes ⚡", "**Queued follow-ups:** 1", "**Connected Platforms:** telegram")
	if strings.Index(got, "**Queued follow-ups:** 1") <= strings.Index(got, "**Agent Running:** Yes ⚡") {
		t.Fatalf("queued follow-up line should follow Agent Running in:\n%s", got)
	}
	if strings.Index(got, "**Queued follow-ups:** 1") >= strings.Index(got, "**Connected Platforms:** telegram") {
		t.Fatalf("queued follow-up line should precede Connected Platforms in:\n%s", got)
	}
}

func TestStatusCommandIncludesKanbanDispatcherStatus(t *testing.T) {
	ctx := context.Background()
	smap := session.NewMemMap()
	now := time.Date(2026, 5, 7, 13, 14, 15, 0, time.UTC)
	statusStore := NewRuntimeStatusStore(filepath.Join(t.TempDir(), "runtime-status.json"))
	if err := statusStore.UpdateRuntimeStatus(ctx, RuntimeStatusUpdate{
		KanbanDispatcher: &KanbanDispatcherStatus{
			State:       KanbanDispatcherStateDegraded,
			LastTickAt:  now.Format(time.RFC3339Nano),
			LastError:   "worker_spawn_failed: missing profile",
			Spawned:     2,
			SpawnFailed: 1,
			AutoBlocked: 3,
		},
	}); err != nil {
		t.Fatalf("seed runtime status: %v", err)
	}

	h := newStatusCommandHarness(t, smap, now, func(cfg *ManagerConfig) {
		cfg.RuntimeStatus = statusStore
	})

	if err := h.manager.handleInbound(ctx, InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventStatus}); err != nil {
		t.Fatal(err)
	}

	sent := h.channel.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1: %#v", len(sent), sent)
	}
	got := sent[0].Text
	assertContainsAll(t, got,
		"**Kanban Dispatcher:** `degraded`",
		"**Kanban Last Tick:** `"+now.Format(time.RFC3339Nano)+"`",
		"**Kanban Spawned:** 2",
		"**Kanban Spawn Failed:** 1",
		"**Kanban Auto Blocked:** 3",
		"**Kanban Last Error:** "+tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, "worker_spawn_failed: missing profile"),
	)
}

type unreadableRuntimeStatus struct{}

func (unreadableRuntimeStatus) UpdateRuntimeStatus(context.Context, RuntimeStatusUpdate) error {
	return nil
}

func (unreadableRuntimeStatus) ReadRuntimeStatus(context.Context) (RuntimeStatus, error) {
	return RuntimeStatus{}, errors.New("decode runtime status: invalid character")
}

func TestStatusCommandOmitsKanbanDispatcherWhenRuntimeStatusUnreadable(t *testing.T) {
	h := newStatusCommandHarness(t, nil, time.Date(2026, 5, 7, 13, 20, 0, 0, time.UTC), func(cfg *ManagerConfig) {
		cfg.RuntimeStatus = unreadableRuntimeStatus{}
	})

	if err := h.manager.handleInbound(h.ctx, InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventStatus}); err != nil {
		t.Fatal(err)
	}

	sent := h.channel.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1: %#v", len(sent), sent)
	}
	got := sent[0].Text
	if !strings.Contains(got, "📊 **Gormes Gateway Status**") {
		t.Fatalf("status did not render base response:\n%s", got)
	}
	if strings.Contains(got, "**Kanban Dispatcher:**") {
		t.Fatalf("status invented Kanban dispatcher section for unreadable runtime status:\n%s", got)
	}
}

// TestStatusCommand_TitleFromMetadata verifies the renderer surfaces the
// session metadata Title verbatim when present.
func TestStatusCommand_TitleFromMetadata(t *testing.T) {
	ctx := context.Background()
	k := &fakeKernel{}
	smap := session.NewMemMap()
	now := time.Date(2026, 4, 29, 9, 42, 0, 0, time.UTC)
	if err := smap.Put(ctx, "telegram:42", "sess-title"); err != nil {
		t.Fatalf("seed session map: %v", err)
	}
	if err := smap.PutMetadata(ctx, session.Metadata{
		SessionID: "sess-title",
		Source:    "telegram",
		ChatID:    "42",
		Title:     "Friendly Greeting with Juan",
		CreatedAt: now.Unix(),
		UpdatedAt: now.Unix(),
	}); err != nil {
		t.Fatalf("seed metadata: %v", err)
	}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		Now:          func() time.Time { return now },
	}, k, slog.Default())
	ch := newFakeChannel("telegram")
	if err := m.Register(ch); err != nil {
		t.Fatal(err)
	}

	ev := InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventStatus}
	if err := m.handleInbound(ctx, ev); err != nil {
		t.Fatal(err)
	}

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1: %#v", len(sent), sent)
	}
	got := sent[0].Text
	want := "**Title:** Friendly Greeting with Juan"
	if !strings.Contains(got, want) {
		t.Fatalf("status response missing %q in:\n%s", want, got)
	}
	if strings.Contains(got, "title_unavailable") {
		t.Fatalf("metadata title present but rendered title_unavailable:\n%s", got)
	}
	if strings.Contains(got, "(untitled)") {
		t.Fatalf("status leaked legacy (untitled) placeholder:\n%s", got)
	}
}

// TestStatusCommand_TitleUnavailable proves that an empty metadata title
// renders the documented degraded-mode sentinel rather than silently
// omitting the Title field or substituting a synthetic title.
func TestStatusCommand_TitleUnavailable(t *testing.T) {
	ctx := context.Background()
	k := &fakeKernel{}
	smap := session.NewMemMap()
	now := time.Date(2026, 4, 29, 9, 42, 0, 0, time.UTC)
	if err := smap.Put(ctx, "telegram:42", "sess-no-title"); err != nil {
		t.Fatalf("seed session map: %v", err)
	}
	if err := smap.PutMetadata(ctx, session.Metadata{
		SessionID: "sess-no-title",
		Source:    "telegram",
		ChatID:    "42",
		Title:     "",
		CreatedAt: now.Unix(),
		UpdatedAt: now.Unix(),
	}); err != nil {
		t.Fatalf("seed metadata: %v", err)
	}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		Now:          func() time.Time { return now },
	}, k, slog.Default())
	ch := newFakeChannel("telegram")
	if err := m.Register(ch); err != nil {
		t.Fatal(err)
	}

	ev := InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventStatus}
	if err := m.handleInbound(ctx, ev); err != nil {
		t.Fatal(err)
	}

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1: %#v", len(sent), sent)
	}
	got := sent[0].Text
	want := "**Title:** " + tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, "title_unavailable")
	if !strings.Contains(got, want) {
		t.Fatalf("status response missing degraded %q in:\n%s", want, got)
	}
	if strings.Contains(got, "(untitled)") {
		t.Fatalf("status leaked legacy (untitled) placeholder:\n%s", got)
	}
	if strings.Contains(got, "Telegram conversation with") {
		t.Fatalf("status fell back to synthetic legacy title:\n%s", got)
	}
}

// TestStatusCommand_ProviderBypass proves /status is a gateway-only command
// and never reaches the kernel/provider/model path. The fake kernel records
// every PlatformEvent submit; after dispatch the recorder must be empty.
func TestStatusCommand_ProviderBypass(t *testing.T) {
	h := newStatusCommandHarness(t, nil, time.Time{})

	ev := InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		UserID:   "user-juan",
		MsgID:    "1234",
		Kind:     EventStatus,
		Text:     "/status",
	}
	if err := h.manager.handleInbound(h.ctx, ev); err != nil {
		t.Fatal(err)
	}

	if submits := h.kernel.submitsSnapshot(); len(submits) != 0 {
		t.Fatalf("/status leaked into kernel as model submits: %#v", submits)
	}
}

// TestStatusCommand_RepliesWithReplyToMsgID proves the gateway threads the
// triggering inbound MsgID through to the channel's SendReply call so the
// outbound /status message quotes the user's /status request.
func TestStatusCommand_RepliesWithReplyToMsgID(t *testing.T) {
	ctx := context.Background()
	k := &fakeKernel{}
	smap := session.NewMemMap()
	now := time.Date(2026, 4, 29, 9, 42, 0, 0, time.UTC)
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		Now:          func() time.Time { return now },
	}, k, slog.Default())
	ch := &statusReplyChannel{fakeChannel: newFakeChannel("telegram")}
	if err := m.Register(ch); err != nil {
		t.Fatal(err)
	}

	ev := InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		UserID:   "user-juan",
		MsgID:    "status-message-77",
		Kind:     EventStatus,
	}
	if err := m.handleInbound(ctx, ev); err != nil {
		t.Fatal(err)
	}

	if len(ch.replyTo) != 1 || ch.replyTo[0] != ev.MsgID {
		t.Fatalf("status reply_to = %#v, want one reply to %q", ch.replyTo, ev.MsgID)
	}
	if len(ch.sentSnapshot()) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(ch.sentSnapshot()))
	}
}

// TestStatusCommand_BoldFieldLabels asserts the rendered MarkdownV2 body
// contains the bold **Session ID:** label so Telegram (with ParseMode set
// to MarkdownV2 by the bot) displays it as visible bold rather than two
// literal asterisks.
func TestStatusCommand_BoldFieldLabels(t *testing.T) {
	h := newStatusCommandHarness(t, nil, time.Time{})

	ev := InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventStatus}
	if err := h.manager.handleInbound(h.ctx, ev); err != nil {
		t.Fatal(err)
	}

	sent := h.channel.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1", len(sent))
	}
	got := sent[0].Text
	if !strings.Contains(got, "**Session ID:**") {
		t.Fatalf("status response missing **Session ID:** bold label in:\n%s", got)
	}
	if !strings.Contains(got, "📊 **Gormes Gateway Status**") {
		t.Fatalf("status response missing bold gateway header in:\n%s", got)
	}
}

// TestStatusCommand_EscapesMarkdownV2InValueSubstrings confirms that
// metadata-derived value substrings (titles, model names, session IDs) are
// escaped via tgbotapi.EscapeText so MarkdownV2-special characters
// (underscores, asterisks, brackets) cannot break the bold-label parse.
func TestStatusCommand_EscapesMarkdownV2InValueSubstrings(t *testing.T) {
	ctx := context.Background()
	k := &fakeKernel{}
	smap := session.NewMemMap()
	now := time.Date(2026, 4, 29, 9, 42, 0, 0, time.UTC)
	if err := smap.Put(ctx, "telegram:42", "sess-tricky"); err != nil {
		t.Fatalf("seed session map: %v", err)
	}
	if err := smap.PutMetadata(ctx, session.Metadata{
		SessionID: "sess-tricky",
		Source:    "telegram",
		ChatID:    "42",
		Title:     "danger * _ [ ] title",
		CreatedAt: now.Unix(),
		UpdatedAt: now.Unix(),
	}); err != nil {
		t.Fatalf("seed metadata: %v", err)
	}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		Now:          func() time.Time { return now },
	}, k, slog.Default())
	ch := newFakeChannel("telegram")
	if err := m.Register(ch); err != nil {
		t.Fatal(err)
	}

	ev := InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventStatus}
	if err := m.handleInbound(ctx, ev); err != nil {
		t.Fatal(err)
	}

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1", len(sent))
	}
	got := sent[0].Text
	wantEscapedTitle := "**Title:** " + tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, "danger * _ [ ] title")
	if !strings.Contains(got, wantEscapedTitle) {
		t.Fatalf("status response did not escape title MarkdownV2 specials.\nwant substring: %q\ngot:\n%s", wantEscapedTitle, got)
	}
}

// TestManagerStatusCommandRendersHermesStyleGatewayStatus is the original
// session-mapping fixture, updated for the bold-label MarkdownV2 contract.
func TestManagerStatusCommandRendersHermesStyleGatewayStatus(t *testing.T) {
	ctx := context.Background()
	k := &fakeKernel{}
	smap := session.NewMemMap()
	now := time.Date(2026, 4, 29, 9, 42, 0, 0, time.UTC)
	if err := smap.Put(ctx, "telegram:42", "sess-123"); err != nil {
		t.Fatalf("seed session map: %v", err)
	}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		Now:          func() time.Time { return now },
	}, k, slog.Default())
	ch := newFakeChannel("telegram")
	if err := m.Register(ch); err != nil {
		t.Fatal(err)
	}
	m.rememberUsageFrame(kernel.RenderFrame{
		SessionID: "sess-123",
		Telemetry: telemetry.Snapshot{TokensInTotal: 3, TokensOutTotal: 4},
	})

	ev := InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventStatus}
	if err := m.handleInbound(ctx, ev); err != nil {
		t.Fatal(err)
	}

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1: %#v", len(sent), sent)
	}
	got := sent[0].Text
	wantActivity := "**Last Activity:** " + tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, time.Unix(now.Unix(), 0).Format("2006-01-02 15:04"))
	wantCreated := "**Created:** " + tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, time.Unix(now.Unix(), 0).Format("2006-01-02 15:04"))
	assertContainsAll(t, got,
		"📊 **Gormes Gateway Status**",
		"**Session ID:** `sess-123`",
		wantCreated,
		wantActivity,
		"**Cumulative API tokens (re-sent each call):** 7",
		"**Agent Running:** No",
		"**Connected Platforms:** telegram",
	)
	if submits := k.submitsSnapshot(); len(submits) != 0 {
		t.Fatalf("/status submitted to kernel: %#v", submits)
	}
}

func TestManagerStatusCommandInitializesMissingChatSession(t *testing.T) {
	ctx := context.Background()
	k := &fakeKernel{}
	smap := session.NewMemMap()
	now := time.Date(2026, 4, 29, 9, 42, 0, 0, time.UTC)
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		Now:          func() time.Time { return now },
	}, k, slog.Default())
	ch := newFakeChannel("telegram")
	if err := m.Register(ch); err != nil {
		t.Fatal(err)
	}

	ev := InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		UserID:   "user-juan",
		Kind:     EventStatus,
	}
	if err := m.handleInbound(ctx, ev); err != nil {
		t.Fatal(err)
	}

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1: %#v", len(sent), sent)
	}
	got := sent[0].Text
	wantActivity := "**Last Activity:** " + tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, time.Unix(now.Unix(), 0).Format("2006-01-02 15:04"))
	wantCreated := "**Created:** " + tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, time.Unix(now.Unix(), 0).Format("2006-01-02 15:04"))
	assertContainsAll(t, got,
		"**Session ID:** `20260429_094200_",
		wantCreated,
		wantActivity,
		"**Connected Platforms:** telegram",
	)
	if strings.Contains(got, "**Session ID:** `(none)`") || strings.Contains(got, "**Session ID:** `telegram:42`") {
		t.Fatalf("status response returned invalid session id:\n%s", got)
	}
	mapped, err := smap.Get(ctx, "telegram:42")
	if err != nil {
		t.Fatalf("Get session map: %v", err)
	}
	if !strings.HasPrefix(mapped, "20260429_094200_") {
		t.Fatalf("session map = %q, want generated Hermes-style session id", mapped)
	}
	meta, ok, err := smap.GetMetadata(ctx, mapped)
	if err != nil {
		t.Fatalf("GetMetadata: %v", err)
	}
	if !ok {
		t.Fatal("status did not create session metadata")
	}
	if meta.Source != "telegram" || meta.ChatID != "42" || meta.UserID != "user-juan" || meta.UpdatedAt != now.Unix() {
		t.Fatalf("metadata = %+v, want telegram/42/user-juan updated_at=%d", meta, now.Unix())
	}
	// Status-created sessions deliberately leave Title empty rather than
	// invent a synthetic "Telegram conversation with X" string. The row's
	// degraded_mode requires the renderer to surface title_unavailable.
	if strings.TrimSpace(meta.Title) != "" {
		t.Fatalf("metadata title = %q, want empty so renderer surfaces title_unavailable degraded mode", meta.Title)
	}
	wantSentinel := "**Title:** " + tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, "title_unavailable")
	if !strings.Contains(got, wantSentinel) {
		t.Fatalf("status response missing degraded title sentinel %q in:\n%s", wantSentinel, got)
	}
	if strings.Contains(got, "Telegram conversation with") || strings.Contains(got, "Telegram chat ") {
		t.Fatalf("status response leaked synthetic legacy title:\n%s", got)
	}
	if submits := k.submitsSnapshot(); len(submits) != 0 {
		t.Fatalf("/status submitted to kernel: %#v", submits)
	}
}

func TestManagerStatusCommandRepliesToInboundTelegramMessage(t *testing.T) {
	ctx := context.Background()
	k := &fakeKernel{}
	smap := session.NewMemMap()
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		Now:          func() time.Time { return time.Date(2026, 4, 29, 9, 42, 0, 0, time.UTC) },
	}, k, slog.Default())
	ch := &statusReplyChannel{fakeChannel: newFakeChannel("telegram")}
	if err := m.Register(ch); err != nil {
		t.Fatal(err)
	}

	ev := InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		UserID:   "user-juan",
		MsgID:    "status-message-77",
		Kind:     EventStatus,
	}
	if err := m.handleInbound(ctx, ev); err != nil {
		t.Fatal(err)
	}

	if len(ch.replyTo) != 1 || ch.replyTo[0] != ev.MsgID {
		t.Fatalf("status reply_to = %#v, want one reply to %q", ch.replyTo, ev.MsgID)
	}
	if len(ch.sentSnapshot()) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(ch.sentSnapshot()))
	}
}

func TestManagerStatusCommandReplacesLegacyChatKeySessionID(t *testing.T) {
	ctx := context.Background()
	k := &fakeKernel{}
	smap := session.NewMemMap()
	now := time.Date(2026, 4, 29, 11, 42, 0, 0, time.UTC)
	if err := smap.Put(ctx, "telegram:42", "telegram:42"); err != nil {
		t.Fatalf("seed legacy chat-key session: %v", err)
	}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		Now:          func() time.Time { return now },
	}, k, slog.Default())
	ch := newFakeChannel("telegram")
	if err := m.Register(ch); err != nil {
		t.Fatal(err)
	}

	ev := InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventStatus}
	if err := m.handleInbound(ctx, ev); err != nil {
		t.Fatal(err)
	}

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1: %#v", len(sent), sent)
	}
	got := sent[0].Text
	if strings.Contains(got, "**Session ID:** `telegram:42`") {
		t.Fatalf("status leaked legacy chat-key session id:\n%s", got)
	}
	wantCreated := "**Created:** " + tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, time.Unix(now.Unix(), 0).Format("2006-01-02 15:04"))
	if !strings.Contains(got, wantCreated) {
		t.Fatalf("status response did not derive Created from generated session id:\n%s", got)
	}
	if strings.Contains(got, "**Title:** (untitled)") || strings.Contains(got, "**Created:** (unknown)") {
		t.Fatalf("status response leaked placeholder fields:\n%s", got)
	}
	mapped, err := smap.Get(ctx, "telegram:42")
	if err != nil {
		t.Fatalf("Get session map: %v", err)
	}
	if mapped == "telegram:42" || !strings.HasPrefix(mapped, "20260429_114200_") {
		t.Fatalf("session map = %q, want legacy chat-key replaced with generated id", mapped)
	}
}
