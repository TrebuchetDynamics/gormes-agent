# Phase 2.B.2 Gateway Chassis + Discord — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract a reusable gateway chassis from the shipped Telegram adapter, refactor Telegram onto it, and port Discord from picoclaw as the second channel — validating the abstraction with two real channels while preserving byte-identical Telegram behavior.

**Architecture:** Shared `internal/gateway/` owns cross-channel mechanics (Channel interface, capability sub-interfaces, Manager, coalescer, render helpers, session-map persistence, auth gate, command normalization). Individual channels live under `internal/channels/<name>/` and translate their SDK events to `gateway.InboundEvent`. `gormes gateway` subcommand wires any configured channels into one Manager.

**Tech Stack:** Go 1.23+, `github.com/go-telegram-bot-api/telegram-bot-api/v5` (existing), `github.com/bwmarrin/discordgo` (new), `github.com/spf13/cobra` (existing), `bbolt` (via `internal/session`).

**Spec:** [`docs/superpowers/specs/2026-04-21-gormes-phase2b2-chassis-design.md`](../specs/2026-04-21-gormes-phase2b2-chassis-design.md)

---

## File Structure

**Created:**
- `internal/gateway/channel.go` — `Channel` interface + capability sub-interfaces
- `internal/gateway/event.go` — `InboundEvent`, `EventKind`, `ChatKey()` helper
- `internal/gateway/manager.go` — `Manager`, `NewManager`, `Register`, `Run`
- `internal/gateway/coalesce.go` — moved + generalized from `internal/telegram/coalesce.go`
- `internal/gateway/render.go` — moved + generalized from `internal/telegram/render.go`
- `internal/gateway/channel_test.go` — interface contract tests
- `internal/gateway/event_test.go` — InboundEvent helpers
- `internal/gateway/manager_test.go` — Manager inbound+outbound tests with fakeChannel
- `internal/gateway/coalesce_test.go` — moved + generalized from telegram
- `internal/gateway/fake_test.go` — shared `fakeChannel` implementing every capability
- `internal/channels/discord/bot.go` — Discord Channel implementation
- `internal/channels/discord/client.go` — `discordSession` interface (testing seam)
- `internal/channels/discord/real_client.go` — `discordgo.Session` wrapper
- `internal/channels/discord/bot_test.go` — Discord adapter tests
- `internal/channels/discord/mock_test.go` — fake discordSession
- `cmd/gormes/gateway.go` — new `gormes gateway` cobra subcommand
- `docs/superpowers/plans/2026-04-21-gormes-phase2b2-chassis.md` — this file

**Moved (package rename `internal/telegram/` → `internal/channels/telegram/`):**
- `internal/telegram/bot.go` → `internal/channels/telegram/bot.go` (refactored to implement `gateway.Channel`)
- `internal/telegram/client.go` → `internal/channels/telegram/client.go`
- `internal/telegram/real_client.go` → `internal/channels/telegram/real_client.go`
- `internal/telegram/bot_test.go` → `internal/channels/telegram/bot_test.go`
- `internal/telegram/mock_test.go` → `internal/channels/telegram/mock_test.go`
- `internal/telegram/render_test.go` → deleted (render moved to gateway, tests move with it)
- `internal/telegram/coalesce.go` → deleted (moved to `internal/gateway/coalesce.go`)
- `internal/telegram/coalesce_test.go` → deleted (moved to `internal/gateway/coalesce_test.go`)
- `internal/telegram/render.go` → deleted (moved to `internal/gateway/render.go`)

**Modified:**
- `internal/config/config.go` — add `DiscordCfg` struct + `[discord]` TOML section + defaults
- `internal/config/config_test.go` — tests for discord defaults and parsing
- `internal/session/session.go` — add `DiscordKey(channelID string) string` helper
- `internal/session/mem_test.go` — test for `DiscordKey`
- `cmd/gormes/telegram.go` — refactor to construct a single-channel `gateway.Manager`
- `cmd/gormes/main.go` — register the new `gatewayCmd`
- `cmd/gormes/doctor.go` — add `CheckGateway` covering both channels
- `go.mod` / `go.sum` — add `github.com/bwmarrin/discordgo`

**Deleted (at end of plan):**
- `internal/telegram/` — empty after move; directory removed

---

## Workspace Conventions

This repository root is already `gormes/`. Every repo-local path and shell command in this plan is relative to this directory.

- Strip a mistaken leading `gormes/` from repo paths anywhere in this document.
- Strip a mistaken leading `cd gormes &&` from shell commands anywhere in this document.
- Do not rewrite Go import paths such as `github.com/TrebuchetDynamics/gormes-agent/...`, and do not rewrite CLI command names like `gormes gateway`.

---

## Task Ordering Rationale

The chassis is built first with only fakes so its shape is proven before any real SDK work. Coalescer and render helpers move next as isolated refactors. Telegram is refactored onto the chassis as a pure move + interface-adapter commit — zero behavior change. Discord is ported on top of the proven chassis. Config and cobra wiring come last because they depend on everything above.

Every task ends with a green `go test ./...` and a commit. No task batches red-test + impl into one step.

---

## Task 1: Gateway package skeleton — `InboundEvent` + `EventKind`

**Files:**
- Create: `internal/gateway/event.go`
- Create: `internal/gateway/event_test.go`

- [ ] **Step 1.1: Write the failing test**

Create `internal/gateway/event_test.go`:

```go
package gateway

import "testing"

func TestInboundEvent_ChatKey(t *testing.T) {
	tests := []struct {
		name string
		e    InboundEvent
		want string
	}{
		{"telegram", InboundEvent{Platform: "telegram", ChatID: "42"}, "telegram:42"},
		{"discord", InboundEvent{Platform: "discord", ChatID: "987654321"}, "discord:987654321"},
		{"empty chat id", InboundEvent{Platform: "telegram", ChatID: ""}, "telegram:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.ChatKey(); got != tt.want {
				t.Errorf("ChatKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEventKind_String(t *testing.T) {
	tests := []struct {
		k    EventKind
		want string
	}{
		{EventUnknown, "unknown"},
		{EventSubmit, "submit"},
		{EventCancel, "cancel"},
		{EventReset, "reset"},
		{EventStart, "start"},
	}
	for _, tt := range tests {
		if got := tt.k.String(); got != tt.want {
			t.Errorf("EventKind(%d).String() = %q, want %q", tt.k, got, tt.want)
		}
	}
}
```

- [ ] **Step 1.2: Run test to verify it fails**

Run: `cd gormes && go test ./internal/gateway/...`
Expected: compile error — package `gateway` does not exist.

- [ ] **Step 1.3: Implement `event.go`**

Create `internal/gateway/event.go`:

```go
// Package gateway is the channel-agnostic chassis that every gormes
// messaging adapter plugs into. Adapters translate their SDK's events
// into InboundEvent and implement the Channel interface (plus whichever
// capability sub-interfaces their platform supports). The Manager owns
// cross-channel mechanics: auth gate, command normalization, coalescing,
// session-map persistence, and frame routing.
package gateway

// EventKind is the normalized command kind on an inbound message.
// Platform-specific commands ("/new", "/stop", etc.) are translated
// to EventKind by each channel before pushing to the Manager.
type EventKind int

const (
	// EventUnknown is an unrecognized slash-command. Manager sends a
	// "unknown command" reply via the channel's Send.
	EventUnknown EventKind = iota
	// EventSubmit carries user text for kernel.PlatformEventSubmit.
	EventSubmit
	// EventCancel maps to kernel.PlatformEventCancel ("/stop").
	EventCancel
	// EventReset maps to kernel.PlatformEventResetSession ("/new").
	EventReset
	// EventStart is the help/welcome command ("/start"). No kernel
	// submission; Manager replies with a greeting string.
	EventStart
)

// String returns a human-readable name used in logs and tests.
func (k EventKind) String() string {
	switch k {
	case EventSubmit:
		return "submit"
	case EventCancel:
		return "cancel"
	case EventReset:
		return "reset"
	case EventStart:
		return "start"
	default:
		return "unknown"
	}
}

// InboundEvent is the normalized form of a platform message that the
// Manager consumes. Every channel produces these; no platform-specific
// types cross the chassis boundary.
type InboundEvent struct {
	// Platform is the channel.Name() that produced this event.
	Platform string
	// ChatID is the platform-native chat/channel ID, encoded as string
	// so it fits the session.Map key shape ("<platform>:<chat_id>").
	ChatID string
	// UserID is the sender's platform ID. Logged only; no behavior.
	UserID string
	// MsgID is the platform message ID of the inbound message.
	// Used by Manager for reaction acks (ReactionCapable).
	MsgID string
	// Kind is the normalized command kind.
	Kind EventKind
	// Text is the message body for EventSubmit; empty for commands.
	Text string
}

// ChatKey returns "<platform>:<chat_id>" — the format the
// internal/session.Map uses for its bolt/memory keys.
func (e InboundEvent) ChatKey() string {
	return e.Platform + ":" + e.ChatID
}
```

- [ ] **Step 1.4: Run test to verify it passes**

Run: `cd gormes && go test ./internal/gateway/...`
Expected: `ok  github.com/TrebuchetDynamics/gormes-agent/internal/gateway`

- [ ] **Step 1.5: Commit**

```bash
git add internal/gateway/event.go internal/gateway/event_test.go
git commit -m "$(cat <<'EOF'
feat(gateway): InboundEvent + EventKind for chassis

First brick of the Phase 2.B.2 chassis. Normalizes platform
commands (/new /stop /start) to a single EventKind enum so the
Manager sees one event shape regardless of channel. ChatKey()
produces the same "<platform>:<chat_id>" format internal/session
already uses.
EOF
)"
```

---

## Task 2: `Channel` interface + capability sub-interfaces

**Files:**
- Create: `internal/gateway/channel.go`
- Create: `internal/gateway/channel_test.go`
- Create: `internal/gateway/fake_test.go`

- [ ] **Step 2.1: Write the failing test**

Create `internal/gateway/channel_test.go`:

```go
package gateway

import "testing"

// Compile-time assertion that fakeChannel implements every interface
// the chassis defines. If a field rename drifts away, this file
// stops compiling — a cheap drift alarm for contributors.
var (
	_ Channel            = (*fakeChannel)(nil)
	_ MessageEditor      = (*fakeChannel)(nil)
	_ PlaceholderCapable = (*fakeChannel)(nil)
	_ TypingCapable      = (*fakeChannel)(nil)
	_ ReactionCapable    = (*fakeChannel)(nil)
)

func TestChannel_NameStable(t *testing.T) {
	ch := newFakeChannel("test")
	if got := ch.Name(); got != "test" {
		t.Errorf("Name() = %q, want %q", got, "test")
	}
}
```

- [ ] **Step 2.2: Run test to verify it fails**

Run: `cd gormes && go test ./internal/gateway/...`
Expected: compile error — `Channel`, `MessageEditor`, etc. undefined.

- [ ] **Step 2.3: Implement `channel.go`**

Create `internal/gateway/channel.go`:

```go
package gateway

import "context"

// Channel is the minimum every adapter implements. Capabilities beyond
// this (editing, typing, placeholders, reactions) are declared via
// separate optional interfaces the Manager type-asserts at runtime.
type Channel interface {
	// Name returns a stable identifier used as the Platform component
	// of ChatKey ("telegram", "discord", ...).
	Name() string

	// Run starts the inbound loop and blocks until ctx cancellation.
	// The adapter translates SDK events to InboundEvent and pushes
	// them to inbox. MUST NOT close inbox — the Manager owns it.
	Run(ctx context.Context, inbox chan<- InboundEvent) error

	// Send delivers a plain-text message to chatID. Returns the
	// platform's message ID so Manager can later edit it via
	// MessageEditor. An empty msgID is valid only when err != nil.
	Send(ctx context.Context, chatID, text string) (msgID string, err error)
}

// MessageEditor — channels that can edit an existing message in place.
// Required for streaming turns where the coalescer edits a single
// placeholder/bot message rather than spamming new messages.
type MessageEditor interface {
	EditMessage(ctx context.Context, chatID, msgID, text string) error
}

// PlaceholderCapable — channels that can send a placeholder message
// ("⏳") that the Manager will later edit with streamed content via
// MessageEditor. The platform-message-ID returned here is what the
// coalescer uses for subsequent edits.
type PlaceholderCapable interface {
	SendPlaceholder(ctx context.Context, chatID string) (msgID string, err error)
}

// TypingCapable — channels that can show a typing indicator.
// The returned stop function MUST be idempotent; Manager may call
// it multiple times in error or cancellation paths.
type TypingCapable interface {
	StartTyping(ctx context.Context, chatID string) (stop func(), err error)
}

// ReactionCapable — channels that can add a reaction to an inbound
// message as an acknowledgment ("👀"). The returned undo function
// removes the reaction and MUST be idempotent.
type ReactionCapable interface {
	ReactToMessage(ctx context.Context, chatID, msgID string) (undo func(), err error)
}
```

- [ ] **Step 2.4: Implement `fakeChannel` for shared test usage**

Create `internal/gateway/fake_test.go`:

```go
package gateway

import (
	"context"
	"strconv"
	"sync"
)

// fakeChannel is a Channel implementation for chassis tests. Implements
// every capability so Manager type-asserts succeed. Push inbound events
// via pushInbound; observe outbound via the sent/edits/placeholders/
// reactions slices (guarded by mu).
type fakeChannel struct {
	name    string
	inbox   chan<- InboundEvent
	started chan struct{} // closed when Run starts

	mu            sync.Mutex
	sent          []fakeSent
	edits         []fakeEdit
	placeholders  []string // chat IDs where a placeholder was sent
	reactions     []fakeReaction
	typingChats   []string
	nextMsgID     int
	sendErr       error // injected by tests if non-nil
	editErr       error // injected by tests
	reactionUndos int   // counts undo invocations
	typingStops   int   // counts typing-stop invocations
}

type fakeSent struct{ ChatID, Text, MsgID string }
type fakeEdit struct{ ChatID, MsgID, Text string }
type fakeReaction struct{ ChatID, MsgID string }

func newFakeChannel(name string) *fakeChannel {
	return &fakeChannel{
		name:      name,
		started:   make(chan struct{}),
		nextMsgID: 1000,
	}
}

func (f *fakeChannel) Name() string { return f.name }

func (f *fakeChannel) Run(ctx context.Context, inbox chan<- InboundEvent) error {
	f.mu.Lock()
	f.inbox = inbox
	f.mu.Unlock()
	close(f.started)
	<-ctx.Done()
	return nil
}

func (f *fakeChannel) Send(ctx context.Context, chatID, text string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.sendErr != nil {
		return "", f.sendErr
	}
	id := strconv.Itoa(f.nextMsgID)
	f.nextMsgID++
	f.sent = append(f.sent, fakeSent{ChatID: chatID, Text: text, MsgID: id})
	return id, nil
}

func (f *fakeChannel) EditMessage(ctx context.Context, chatID, msgID, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.editErr != nil {
		return f.editErr
	}
	f.edits = append(f.edits, fakeEdit{ChatID: chatID, MsgID: msgID, Text: text})
	return nil
}

func (f *fakeChannel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.sendErr != nil {
		return "", f.sendErr
	}
	f.placeholders = append(f.placeholders, chatID)
	id := strconv.Itoa(f.nextMsgID)
	f.nextMsgID++
	return id, nil
}

func (f *fakeChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	f.mu.Lock()
	f.typingChats = append(f.typingChats, chatID)
	f.mu.Unlock()
	return func() {
		f.mu.Lock()
		f.typingStops++
		f.mu.Unlock()
	}, nil
}

func (f *fakeChannel) ReactToMessage(ctx context.Context, chatID, msgID string) (func(), error) {
	f.mu.Lock()
	f.reactions = append(f.reactions, fakeReaction{ChatID: chatID, MsgID: msgID})
	f.mu.Unlock()
	return func() {
		f.mu.Lock()
		f.reactionUndos++
		f.mu.Unlock()
	}, nil
}

// pushInbound is a test helper. Blocks until Run has installed the inbox.
func (f *fakeChannel) pushInbound(e InboundEvent) {
	<-f.started
	f.mu.Lock()
	in := f.inbox
	f.mu.Unlock()
	in <- e
}

// snapshots return copies under the lock for race-safe assertions.
func (f *fakeChannel) sentSnapshot() []fakeSent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]fakeSent, len(f.sent))
	copy(out, f.sent)
	return out
}

func (f *fakeChannel) editsSnapshot() []fakeEdit {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]fakeEdit, len(f.edits))
	copy(out, f.edits)
	return out
}

func (f *fakeChannel) reactionsSnapshot() []fakeReaction {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]fakeReaction, len(f.reactions))
	copy(out, f.reactions)
	return out
}
```

- [ ] **Step 2.5: Run tests to verify they pass**

Run: `cd gormes && go test ./internal/gateway/... -v`
Expected: `PASS` for `TestChannel_NameStable`, `TestInboundEvent_ChatKey`, `TestEventKind_String`.

- [ ] **Step 2.6: Commit**

```bash
git add internal/gateway/channel.go internal/gateway/channel_test.go internal/gateway/fake_test.go
git commit -m "$(cat <<'EOF'
feat(gateway): Channel interface + capability sub-interfaces

Picoclaw-shaped capability split — every channel implements Channel
(Name/Run/Send); optional MessageEditor, PlaceholderCapable,
TypingCapable, ReactionCapable declare richer behaviors that Manager
type-asserts at runtime. fakeChannel covers every capability so
chassis tests don't need per-feature fakes.
EOF
)"
```

---

## Task 3: `Manager` skeleton — `NewManager`, `Register`, duplicate-name guard

**Files:**
- Create: `internal/gateway/manager.go`
- Create: `internal/gateway/manager_test.go`

- [ ] **Step 3.1: Write the failing test**

Create `internal/gateway/manager_test.go`:

```go
package gateway

import (
	"log/slog"
	"testing"
)

func TestManager_RegisterChannel(t *testing.T) {
	m := NewManager(ManagerConfig{}, nil, slog.Default())

	tg := newFakeChannel("telegram")
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register telegram: %v", err)
	}

	dc := newFakeChannel("discord")
	if err := m.Register(dc); err != nil {
		t.Fatalf("Register discord: %v", err)
	}

	if got := m.ChannelCount(); got != 2 {
		t.Errorf("ChannelCount() = %d, want 2", got)
	}
}

func TestManager_RegisterDuplicateName(t *testing.T) {
	m := NewManager(ManagerConfig{}, nil, slog.Default())

	if err := m.Register(newFakeChannel("telegram")); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	err := m.Register(newFakeChannel("telegram"))
	if err == nil {
		t.Fatal("expected duplicate-name error, got nil")
	}
}

func TestManager_RegisterEmptyName(t *testing.T) {
	m := NewManager(ManagerConfig{}, nil, slog.Default())
	if err := m.Register(newFakeChannel("")); err == nil {
		t.Fatal("expected empty-name error, got nil")
	}
}
```

- [ ] **Step 3.2: Run test to verify it fails**

Run: `cd gormes && go test ./internal/gateway/...`
Expected: compile error — `NewManager`, `Manager`, `ManagerConfig` undefined.

- [ ] **Step 3.3: Implement `manager.go` skeleton**

Create `internal/gateway/manager.go`:

```go
package gateway

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
)

// ManagerConfig drives the Manager. All fields are optional unless noted.
type ManagerConfig struct {
	// AllowedChats maps platform -> allowed chat ID (as string). Inbound
	// events whose ChatID does not match the entry for their Platform
	// are dropped. Empty string or missing entry means "deny all" unless
	// AllowDiscovery[platform] is true.
	AllowedChats map[string]string

	// AllowDiscovery enables first-run discovery logging per platform.
	// When true and no allowlist is set, inbound events are logged (so
	// operators can read chat_ids from logs) but still not routed to
	// the kernel.
	AllowDiscovery map[string]bool

	// CoalesceMs is the default coalescer window for outbound streaming.
	// Zero promotes to 1000ms.
	CoalesceMs int

	// SessionMap, when non-nil, persists session_id per ChatKey on every
	// frame where SessionID changed.
	SessionMap session.Map
}

// Manager owns cross-channel mechanics. One Manager per binary.
type Manager struct {
	cfg    ManagerConfig
	kernel *kernel.Kernel
	log    *slog.Logger

	mu       sync.Mutex
	channels map[string]Channel // keyed by Name()
}

// NewManager constructs a Manager. kernel may be nil only in tests that
// do not exercise Run; Register and ChannelCount work either way.
func NewManager(cfg ManagerConfig, k *kernel.Kernel, log *slog.Logger) *Manager {
	if log == nil {
		log = slog.Default()
	}
	if cfg.CoalesceMs <= 0 {
		cfg.CoalesceMs = 1000
	}
	if cfg.AllowedChats == nil {
		cfg.AllowedChats = map[string]string{}
	}
	if cfg.AllowDiscovery == nil {
		cfg.AllowDiscovery = map[string]bool{}
	}
	return &Manager{
		cfg:      cfg,
		kernel:   k,
		log:      log,
		channels: map[string]Channel{},
	}
}

// ErrDuplicateChannel is returned by Register when a channel with the
// same Name() is already registered.
var ErrDuplicateChannel = errors.New("gateway: duplicate channel name")

// ErrEmptyChannelName is returned by Register when Name() is empty.
var ErrEmptyChannelName = errors.New("gateway: channel Name() must be non-empty")

// Register adds a channel. MUST be called before Run.
func (m *Manager) Register(ch Channel) error {
	name := ch.Name()
	if name == "" {
		return ErrEmptyChannelName
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.channels[name]; exists {
		return fmt.Errorf("%w: %q", ErrDuplicateChannel, name)
	}
	m.channels[name] = ch
	return nil
}

// ChannelCount returns the number of registered channels. Test helper
// and also used by cmd/gormes to verify at least one channel is wired.
func (m *Manager) ChannelCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.channels)
}

// Run is implemented in a later task. Declared here to keep the public
// API surface visible for compile-time callers of NewManager.
func (m *Manager) Run(ctx context.Context) error {
	// Filled in by a later task; the skeleton just blocks on ctx so
	// early commits remain linkable.
	<-ctx.Done()
	return nil
}
```

- [ ] **Step 3.4: Run tests to verify they pass**

Run: `cd gormes && go test ./internal/gateway/... -v -run TestManager_Register`
Expected: all three register tests PASS.

- [ ] **Step 3.5: Commit**

```bash
git add internal/gateway/manager.go internal/gateway/manager_test.go
git commit -m "$(cat <<'EOF'
feat(gateway): Manager skeleton — NewManager, Register

Duplicate- and empty-name guards on Register keep the channel
registry injective. Run() is stubbed; inbound + outbound paths
land in follow-up tasks.
EOF
)"
```

---

## Task 4: Move coalescer to gateway, generalize to `MessageEditor`

**Files:**
- Delete: `internal/telegram/coalesce.go` (content absorbed)
- Delete: `internal/telegram/coalesce_test.go` (content absorbed)
- Create: `internal/gateway/coalesce.go`
- Create: `internal/gateway/coalesce_test.go`

The current coalescer hardcodes `telegramClient` + `chatID int64`. It edits via `tgbotapi.NewEditMessageText` and sends via `tgbotapi.NewMessage`. The generalized version takes a `PlaceholderCapable + MessageEditor` and a `chatID string`.

- [ ] **Step 4.1: Write the failing test**

Create `internal/gateway/coalesce_test.go`:

```go
package gateway

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Coalescer must:
//   1. Create the placeholder on first setPending (no msgID yet).
//   2. Edit the placeholder on subsequent setPending, throttled by window.
//   3. flushImmediate bypasses the window.
//   4. Survive transient send/edit errors without crashing.

func TestCoalescer_PlaceholderThenEdit(t *testing.T) {
	ch := newFakeChannel("test")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := newCoalescer(ch, 20*time.Millisecond, "chat1")
	go c.run(ctx)

	c.setPending("first")
	// First flush creates placeholder.
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(ch.sentSnapshot()) == 1
	})

	time.Sleep(25 * time.Millisecond) // exceed the window
	c.setPending("second")
	// Second flush edits the existing placeholder.
	waitFor(t, 200*time.Millisecond, func() bool {
		edits := ch.editsSnapshot()
		return len(edits) == 1 && edits[0].Text == "second"
	})
}

func TestCoalescer_FlushImmediateBypassesWindow(t *testing.T) {
	ch := newFakeChannel("test")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := newCoalescer(ch, 5*time.Second, "chat1") // long window
	go c.run(ctx)

	c.flushImmediate("final")
	waitFor(t, 200*time.Millisecond, func() bool {
		sent := ch.sentSnapshot()
		return len(sent) == 1 && sent[0].Text == "final"
	})
}

func TestCoalescer_SendErrorIsSwallowed(t *testing.T) {
	ch := newFakeChannel("test")
	ch.sendErr = errors.New("transient")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := newCoalescer(ch, 10*time.Millisecond, "chat1")
	go c.run(ctx)

	c.setPending("x")
	time.Sleep(50 * time.Millisecond)
	// No panic; coalescer just didn't advance.
}

// waitFor polls cond until it returns true or the deadline elapses.
func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}
```

- [ ] **Step 4.2: Run test to verify it fails**

Run: `cd gormes && go test ./internal/gateway/... -run TestCoalescer`
Expected: compile error — `newCoalescer` undefined.

- [ ] **Step 4.3: Implement `coalesce.go`**

Create `internal/gateway/coalesce.go`:

```go
package gateway

import (
	"context"
	"sync"
	"time"
)

// coalescer batches outbound edits for one turn. The chassis owns this
// type (ported from internal/telegram/coalesce.go) so every channel
// shares the same "send a placeholder once, edit-throttle to window,
// flushImmediate on final/error/cancel" behavior.
//
// Edits go through whatever PlaceholderCapable + MessageEditor the
// coalescer was constructed with. One coalescer per turn; caller
// tears it down on PhaseIdle/Failed/Cancelling.
type coalescer struct {
	sender placeholderEditor
	window time.Duration
	chatID string

	mu           sync.Mutex
	pendingText  string
	pendingMsgID string // empty until first send creates the message
	lastSentText string
	lastEditAt   time.Time
	retryAfter   time.Time // unused (reserved for rate-limit backoff)
	wakeupCh     chan struct{}
}

// placeholderEditor is the tight dependency the coalescer needs. Every
// Channel that implements both PlaceholderCapable AND MessageEditor
// satisfies it automatically.
type placeholderEditor interface {
	SendPlaceholder(ctx context.Context, chatID string) (msgID string, err error)
	EditMessage(ctx context.Context, chatID, msgID, text string) error
}

func newCoalescer(pe placeholderEditor, window time.Duration, chatID string) *coalescer {
	if window <= 0 {
		window = time.Second
	}
	return &coalescer{
		sender:   pe,
		window:   window,
		chatID:   chatID,
		wakeupCh: make(chan struct{}, 1),
	}
}

func (c *coalescer) setPending(text string) {
	c.mu.Lock()
	c.pendingText = text
	c.mu.Unlock()
	select {
	case c.wakeupCh <- struct{}{}:
	default:
	}
}

func (c *coalescer) currentMessageID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.pendingMsgID
}

func (c *coalescer) flushImmediate(ctx context.Context, text string) {
	c.mu.Lock()
	msgID := c.pendingMsgID
	c.mu.Unlock()

	var sentID string
	var err error
	if msgID == "" {
		sentID, err = c.sender.SendPlaceholder(ctx, c.chatID)
		if err == nil && sentID != "" {
			err = c.sender.EditMessage(ctx, c.chatID, sentID, text)
		}
	} else {
		sentID = msgID
		err = c.sender.EditMessage(ctx, c.chatID, msgID, text)
	}
	if err != nil {
		return
	}

	c.mu.Lock()
	if c.pendingMsgID == "" {
		c.pendingMsgID = sentID
	}
	c.lastSentText = text
	c.lastEditAt = time.Now()
	c.pendingText = ""
	c.mu.Unlock()
}

func (c *coalescer) run(ctx context.Context) {
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

func (c *coalescer) tryFlush(ctx context.Context) {
	c.mu.Lock()
	text := c.pendingText
	msgID := c.pendingMsgID
	last := c.lastSentText
	lastAt := c.lastEditAt
	retryAfter := c.retryAfter
	c.mu.Unlock()

	if text == "" || text == last {
		return
	}
	now := time.Now()
	if now.Before(retryAfter) {
		return
	}
	if msgID != "" && now.Sub(lastAt) < c.window {
		return
	}

	var sentID string
	var err error
	if msgID == "" {
		sentID, err = c.sender.SendPlaceholder(ctx, c.chatID)
		if err == nil && sentID != "" {
			err = c.sender.EditMessage(ctx, c.chatID, sentID, text)
		}
	} else {
		sentID = msgID
		err = c.sender.EditMessage(ctx, c.chatID, msgID, text)
	}
	if err != nil {
		return
	}

	c.mu.Lock()
	if c.pendingMsgID == "" {
		c.pendingMsgID = sentID
	}
	c.lastSentText = text
	c.lastEditAt = time.Now()
	c.mu.Unlock()
}
```

- [ ] **Step 4.4: Delete the old files**

```bash
rm internal/telegram/coalesce.go internal/telegram/coalesce_test.go
```

- [ ] **Step 4.5: Fix `internal/telegram/` build**

The old telegram package still references `coalescer`. Temporarily add a compatibility shim at the top of `internal/telegram/bot.go`. Replace the import block with:

```go
import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
)

// coalescer compatibility shim — the coalescer moved to internal/gateway
// as part of Phase 2.B.2. This adapter lets the legacy bot.go in this
// package keep compiling through the chassis-extraction commits; it is
// deleted when internal/telegram/ is removed in the final plan task.
type coalescer = gateway.TelegramCoalescerCompat

var newCoalescer = gateway.NewTelegramCoalescerCompat
```

Realistically the cleaner move is: **defer telegram's use of the new coalescer until Task 8 (the Telegram refactor)**. Instead of a compat shim, inline the old coalescer code as an unexported local type `telegramCoalescer` within `internal/telegram/bot.go` (copy the pre-move source 1:1), and let gateway's new `coalescer` live independently. Telegram picks up `gateway.coalescer` at refactor time.

Revised approach — skip Step 4.5 above. Instead:

- [ ] **Step 4.5 (revised): Re-add the old coalescer locally to telegram**

Create `internal/telegram/coalesce.go` with the same 171 lines as the pre-delete version (copy verbatim from git: `git show HEAD:internal/telegram/coalesce.go > internal/telegram/coalesce.go`). This keeps telegram compiling without touching its behavior. It is deleted in Task 8 when telegram is refactored onto `gateway.coalescer`.

```bash
git show HEAD:internal/telegram/coalesce.go > internal/telegram/coalesce.go
git show HEAD:internal/telegram/coalesce_test.go > internal/telegram/coalesce_test.go
```

- [ ] **Step 4.6: Run tests**

Run: `cd gormes && go test ./internal/gateway/... ./internal/telegram/...`
Expected: gateway PASS for all three coalescer tests; telegram PASS for unchanged existing tests.

- [ ] **Step 4.7: Commit**

```bash
git add internal/gateway/coalesce.go internal/gateway/coalesce_test.go internal/telegram/coalesce.go internal/telegram/coalesce_test.go
git commit -m "$(cat <<'EOF'
feat(gateway): coalescer generalized to MessageEditor

The coalescer is now channel-agnostic: takes a PlaceholderCapable +
MessageEditor and operates on string chat/msg IDs. Telegram keeps a
local copy of the old coalescer temporarily so the adapter compiles
through the chassis-extraction commits; it migrates onto the new
gateway.coalescer in the Phase 2.B.2 Telegram refactor task.
EOF
)"
```

---

## Task 5: Manager inbound path — allowlist + command normalization

**Files:**
- Modify: `internal/gateway/manager.go`
- Modify: `internal/gateway/manager_test.go`

This task wires `Manager.Run` to spawn every registered channel's `Run`, consume the shared inbox, enforce the allowlist, translate `EventKind` to `kernel.PlatformEvent`, and pin the turn origin.

- [ ] **Step 5.1: Write the failing tests**

Append to `internal/gateway/manager_test.go`:

```go
import (
	"context"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

// fakeKernel implements the subset of *kernel.Kernel the Manager uses.
// Manager consumes *kernel.Kernel concretely; we test via a narrow
// kernelSubmitter interface injected for tests.
type fakeKernel struct {
	mu        sync.Mutex
	submits   []kernel.PlatformEvent
	resets    int
	submitErr error
	resetErr  error
}

func (f *fakeKernel) Submit(e kernel.PlatformEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.submitErr != nil {
		return f.submitErr
	}
	f.submits = append(f.submits, e)
	return nil
}

func (f *fakeKernel) ResetSession() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.resetErr != nil {
		return f.resetErr
	}
	f.resets++
	return nil
}

func (f *fakeKernel) Render() <-chan kernel.RenderFrame {
	return nil // outbound path covered in Task 7
}

func (f *fakeKernel) submitsSnapshot() []kernel.PlatformEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]kernel.PlatformEvent, len(f.submits))
	copy(out, f.submits)
	return out
}

func TestManager_Inbound_AllowedChat_Submit(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "m",
		Kind: EventSubmit, Text: "hello",
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})
	got := fk.submitsSnapshot()[0]
	if got.Kind != kernel.PlatformEventSubmit || got.Text != "hello" {
		t.Errorf("kernel submit = %+v", got)
	}
}

func TestManager_Inbound_BlockedChat_NoSubmit(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, slog.Default())
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "999", Kind: EventSubmit, Text: "hi",
	})

	time.Sleep(50 * time.Millisecond)
	if n := len(fk.submitsSnapshot()); n != 0 {
		t.Errorf("blocked chat should produce 0 submits, got %d", n)
	}
}

func TestManager_Inbound_Cancel(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, slog.Default())
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", Kind: EventCancel,
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		s := fk.submitsSnapshot()
		return len(s) == 1 && s[0].Kind == kernel.PlatformEventCancel
	})
}

func TestManager_Inbound_Reset(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, slog.Default())
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", Kind: EventReset,
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		fk.mu.Lock()
		defer fk.mu.Unlock()
		return fk.resets == 1
	})
}

func TestManager_Inbound_Start_RepliesHelp(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, slog.Default())
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", Kind: EventStart,
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		sent := tg.sentSnapshot()
		return len(sent) == 1 &&
			sent[0].ChatID == "42" &&
			strings.Contains(sent[0].Text, "online")
	})
	if n := len(fk.submitsSnapshot()); n != 0 {
		t.Errorf("EventStart should not submit to kernel, got %d", n)
	}
}
```

Add these imports to the top of the file: `"context"`, `"strings"`, `"sync"`, `"time"`, `"log/slog"`, `"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"`.

- [ ] **Step 5.2: Run test to verify it fails**

Run: `cd gormes && go test ./internal/gateway/... -run TestManager_Inbound`
Expected: compile error — `NewManagerWithSubmitter` undefined; `fakeKernel` does not satisfy the Manager's type.

- [ ] **Step 5.3: Extract `kernelSubmitter` interface + add `NewManagerWithSubmitter`**

Edit `internal/gateway/manager.go` — replace the `Manager` type and constructors:

```go
// kernelSubmitter is the narrow Manager-side view of *kernel.Kernel.
// Production wires *kernel.Kernel; tests wire fakeKernel.
type kernelSubmitter interface {
	Submit(kernel.PlatformEvent) error
	ResetSession() error
	Render() <-chan kernel.RenderFrame
}

type Manager struct {
	cfg    ManagerConfig
	kernel kernelSubmitter
	log    *slog.Logger

	mu       sync.Mutex
	channels map[string]Channel

	// currentTurn origin — set on accepted Submit, cleared on final frame.
	turnMu       sync.Mutex
	turnPlatform string
	turnChatID   string
	turnMsgID    string
}

// NewManager is the production constructor — takes the concrete kernel.
func NewManager(cfg ManagerConfig, k *kernel.Kernel, log *slog.Logger) *Manager {
	return newManagerInternal(cfg, k, log)
}

// NewManagerWithSubmitter lets tests inject a fake kernel.
func NewManagerWithSubmitter(cfg ManagerConfig, k kernelSubmitter, log *slog.Logger) *Manager {
	return newManagerInternal(cfg, k, log)
}

func newManagerInternal(cfg ManagerConfig, k kernelSubmitter, log *slog.Logger) *Manager {
	if log == nil {
		log = slog.Default()
	}
	if cfg.CoalesceMs <= 0 {
		cfg.CoalesceMs = 1000
	}
	if cfg.AllowedChats == nil {
		cfg.AllowedChats = map[string]string{}
	}
	if cfg.AllowDiscovery == nil {
		cfg.AllowDiscovery = map[string]bool{}
	}
	return &Manager{cfg: cfg, kernel: k, log: log, channels: map[string]Channel{}}
}
```

- [ ] **Step 5.4: Implement `Manager.Run` + inbound pipeline**

Replace the existing `Run` stub in `internal/gateway/manager.go`:

```go
// startGreeting is the reply text for EventStart. Kept here instead of
// per-channel so all adapters deliver the same help message.
const startGreeting = "Gormes is online. Send a message to start a turn. Commands: /stop /new"

func (m *Manager) Run(ctx context.Context) error {
	m.mu.Lock()
	channels := make([]Channel, 0, len(m.channels))
	for _, ch := range m.channels {
		channels = append(channels, ch)
	}
	m.mu.Unlock()

	inbox := make(chan InboundEvent, len(channels)*4)

	var wg sync.WaitGroup
	for _, ch := range channels {
		wg.Add(1)
		go func(c Channel) {
			defer wg.Done()
			if err := c.Run(ctx, inbox); err != nil && !errors.Is(err, context.Canceled) {
				m.log.Warn("channel exited with error", "channel", c.Name(), "err", err)
			}
		}(ch)
	}

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return nil
		case ev := <-inbox:
			m.handleInbound(ctx, ev)
		}
	}
}

func (m *Manager) handleInbound(ctx context.Context, ev InboundEvent) {
	if !m.allowed(ev) {
		if m.cfg.AllowDiscovery[ev.Platform] {
			m.log.Info("first-run discovery: unknown chat",
				"platform", ev.Platform,
				"chat_id", ev.ChatID)
		} else {
			m.log.Warn("unauthorised chat blocked",
				"platform", ev.Platform,
				"chat_id", ev.ChatID)
		}
		return
	}

	ch := m.lookupChannel(ev.Platform)
	if ch == nil {
		m.log.Warn("inbound for unknown channel", "platform", ev.Platform)
		return
	}

	switch ev.Kind {
	case EventStart:
		if _, err := ch.Send(ctx, ev.ChatID, startGreeting); err != nil {
			m.log.Warn("send greeting", "err", err)
		}
	case EventCancel:
		_ = m.kernel.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventCancel})
	case EventReset:
		if err := m.kernel.ResetSession(); err != nil {
			if errors.Is(err, kernel.ErrResetDuringTurn) {
				_, _ = ch.Send(ctx, ev.ChatID,
					"Cannot reset during active turn — send /stop first.")
			} else {
				_, _ = ch.Send(ctx, ev.ChatID, "Session reset failed: "+err.Error())
			}
			return
		}
		_, _ = ch.Send(ctx, ev.ChatID, "Session reset. Next message starts fresh.")
	case EventSubmit:
		if err := m.kernel.Submit(kernel.PlatformEvent{
			Kind: kernel.PlatformEventSubmit,
			Text: ev.Text,
		}); err != nil {
			_, _ = ch.Send(ctx, ev.ChatID, "Busy — try again in a second.")
			return
		}
		m.pinTurn(ev.Platform, ev.ChatID, ev.MsgID)
	case EventUnknown:
		_, _ = ch.Send(ctx, ev.ChatID, "unknown command")
	}
}

func (m *Manager) allowed(ev InboundEvent) bool {
	want, ok := m.cfg.AllowedChats[ev.Platform]
	if !ok || want == "" {
		return false
	}
	return ev.ChatID == want
}

func (m *Manager) lookupChannel(name string) Channel {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.channels[name]
}

func (m *Manager) pinTurn(platform, chatID, msgID string) {
	m.turnMu.Lock()
	m.turnPlatform = platform
	m.turnChatID = chatID
	m.turnMsgID = msgID
	m.turnMu.Unlock()
}

func (m *Manager) clearTurn() {
	m.turnMu.Lock()
	m.turnPlatform = ""
	m.turnChatID = ""
	m.turnMsgID = ""
	m.turnMu.Unlock()
}
```

- [ ] **Step 5.5: Run tests**

Run: `cd gormes && go test ./internal/gateway/... -run TestManager -v`
Expected: all inbound tests PASS.

- [ ] **Step 5.6: Commit**

```bash
git add internal/gateway/manager.go internal/gateway/manager_test.go
git commit -m "$(cat <<'EOF'
feat(gateway): Manager inbound path

Manager.Run spawns each channel's Run, fans inbound events into a
shared inbox, and routes them: allowlist gate first, then kernel
submit for EventSubmit/Cancel/Reset, greeting for EventStart, "busy"
fallback when kernel rejects. Turn origin pinned for the outbound
path landing in a follow-up task.
EOF
)"
```

---

## Task 6: Move render helpers to gateway

**Files:**
- Delete: `internal/telegram/render.go`
- Delete: `internal/telegram/render_test.go`
- Create: `internal/gateway/render.go`
- Create: `internal/gateway/render_test.go`

The Telegram render helpers (`formatStream`, `formatFinal`, `formatError`) emit MarkdownV2 today. Discord messages are plain text; Telegram escaping must not bleed into Discord. So the move keeps the Telegram-style escaping but renames the helpers explicitly and adds a plain variant.

- [ ] **Step 6.1: Write the failing test**

Create `internal/gateway/render_test.go`:

```go
package gateway

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestFormatStreamPlain_DraftPassThrough(t *testing.T) {
	f := kernel.RenderFrame{DraftText: "hello world"}
	if got := FormatStreamPlain(f); got != "hello world" {
		t.Errorf("FormatStreamPlain = %q", got)
	}
}

func TestFormatStreamPlain_IncludesSoulLine(t *testing.T) {
	f := kernel.RenderFrame{
		DraftText: "draft",
		SoulEvents: []kernel.SoulEvent{
			{Text: "running tool foo"},
		},
	}
	got := FormatStreamPlain(f)
	if !strings.Contains(got, "draft") || !strings.Contains(got, "running tool foo") {
		t.Errorf("FormatStreamPlain = %q", got)
	}
}

func TestFormatFinalPlain_LastAssistant(t *testing.T) {
	f := kernel.RenderFrame{
		History: []kernel.HistoryEntry{
			{Role: "user", Content: "hi"},
			{Role: "assistant", Content: "the answer"},
		},
	}
	if got := FormatFinalPlain(f); got != "the answer" {
		t.Errorf("FormatFinalPlain = %q", got)
	}
}

func TestFormatErrorPlain(t *testing.T) {
	f := kernel.RenderFrame{LastError: "boom"}
	if got := FormatErrorPlain(f); got != "❌ boom" {
		t.Errorf("FormatErrorPlain = %q", got)
	}
}

func TestFormatStreamTelegram_EscapesAndEmits(t *testing.T) {
	// MarkdownV2 requires "!" escape; verify the Telegram variant still
	// escapes while the Plain variant leaves it alone.
	f := kernel.RenderFrame{DraftText: "wow!"}
	plain := FormatStreamPlain(f)
	tg := FormatStreamTelegram(f)
	if plain == tg {
		t.Fatalf("plain and telegram outputs should differ; both = %q", plain)
	}
	if !strings.Contains(tg, "wow") {
		t.Errorf("telegram output lost body: %q", tg)
	}
}
```

- [ ] **Step 6.2: Run test to verify it fails**

Run: `cd gormes && go test ./internal/gateway/... -run TestFormat`
Expected: compile error — `FormatStreamPlain`, `FormatFinalPlain`, `FormatErrorPlain`, `FormatStreamTelegram` undefined.

- [ ] **Step 6.3: Implement `internal/gateway/render.go`**

```go
package gateway

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

// maxMessageLen is a conservative cross-channel budget. Telegram is
// 4096; Discord is 2000; WhatsApp is ~65k. We clamp to 4000 (Telegram
// safe) with per-channel overrides planned for follow-up specs.
const maxMessageLen = 4000

// FormatStreamPlain renders a streaming RenderFrame as plain text.
// No platform-specific escaping. Used by channels whose messages are
// plain strings (Discord, Slack, etc.).
func FormatStreamPlain(f kernel.RenderFrame) string {
	body := f.DraftText
	tail := ""
	if len(f.SoulEvents) > 0 {
		last := f.SoulEvents[len(f.SoulEvents)-1]
		if last.Text != "" && last.Text != "idle" {
			tail = "\n\n🔧 " + last.Text
		}
	}
	if f.Phase == kernel.PhaseReconnecting {
		tail += "\n\nreconnecting…"
	}
	return truncate(body + tail)
}

// FormatFinalPlain returns the most recent assistant message from
// History (DraftText is cleared on PhaseIdle) as plain text.
func FormatFinalPlain(f kernel.RenderFrame) string {
	for i := len(f.History) - 1; i >= 0; i-- {
		if f.History[i].Role == "assistant" {
			return truncate(f.History[i].Content)
		}
	}
	return "(empty reply)"
}

// FormatErrorPlain renders a PhaseFailed/Cancelling frame.
func FormatErrorPlain(f kernel.RenderFrame) string {
	text := "❌ " + f.LastError
	if f.LastError == "" {
		text = "❌ cancelled"
	}
	return truncate(text)
}

// FormatStreamTelegram / FormatFinalTelegram / FormatErrorTelegram are
// the MarkdownV2-escaped variants the telegram channel uses. The split
// is a small code duplication in service of keeping render logic in
// gateway/ rather than scattered across adapters.

func FormatStreamTelegram(f kernel.RenderFrame) string {
	body := tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, f.DraftText)
	body = truncate(body)
	tail := ""
	if len(f.SoulEvents) > 0 {
		last := f.SoulEvents[len(f.SoulEvents)-1]
		if last.Text != "" && last.Text != "idle" {
			tail = "\n\n_" + tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, "🔧 "+last.Text) + "_"
		}
	}
	if f.Phase == kernel.PhaseReconnecting {
		tail += "\n\n_reconnecting…_"
	}
	return body + tail
}

func FormatFinalTelegram(f kernel.RenderFrame) string {
	for i := len(f.History) - 1; i >= 0; i-- {
		if f.History[i].Role == "assistant" {
			return truncate(
				tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, f.History[i].Content))
		}
	}
	return "_\\(empty reply\\)_"
}

func FormatErrorTelegram(f kernel.RenderFrame) string {
	text := "❌ " + f.LastError
	if f.LastError == "" {
		text = "❌ cancelled"
	}
	return truncate(tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, text))
}

func truncate(s string) string {
	r := []rune(s)
	if len(r) <= maxMessageLen {
		return s
	}
	return string(r[:maxMessageLen-1]) + "…"
}
```

- [ ] **Step 6.4: Delete old telegram render files**

```bash
rm internal/telegram/render.go internal/telegram/render_test.go
```

- [ ] **Step 6.5: Fix telegram package references**

The old `internal/telegram/bot.go` references `formatStream`, `formatFinal`, `formatError`. Redirect these to the new gateway helpers. Open `internal/telegram/bot.go` and add to the import block:

```go
"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
```

Then find each reference and replace:
- `formatStream(f)` → `gateway.FormatStreamTelegram(f)`
- `formatFinal(f)` → `gateway.FormatFinalTelegram(f)`
- `formatError(f)` → `gateway.FormatErrorTelegram(f)`

- [ ] **Step 6.6: Run tests**

Run: `cd gormes && go test ./internal/gateway/... ./internal/telegram/...`
Expected: gateway render tests PASS; telegram tests stay green (behavior unchanged).

- [ ] **Step 6.7: Commit**

```bash
git add -A internal/gateway/render.go internal/gateway/render_test.go internal/telegram/
git commit -m "$(cat <<'EOF'
feat(gateway): render helpers — Plain + Telegram variants

Plain variants emit raw UTF-8 for Discord/Slack/etc.; Telegram
variants keep the MarkdownV2 escaping the shipped adapter relies on.
internal/telegram/bot.go now calls FormatStreamTelegram etc. to keep
byte-identical output through the refactor.
EOF
)"
```

---

## Task 7: Manager outbound path — turn-pinned frame routing via coalescer

**Files:**
- Modify: `internal/gateway/manager.go`
- Modify: `internal/gateway/manager_test.go`

- [ ] **Step 7.1: Write the failing tests**

Append to `internal/gateway/manager_test.go`:

```go
func TestManager_Outbound_StreamsToPinnedChannel(t *testing.T) {
	tg := newFakeChannel("telegram")
	frames := make(chan kernel.RenderFrame, 8)
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		CoalesceMs:   10,
	}, fk, slog.Default())
	m.setRenderChan(frames) // test-only override
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	// Pin the turn by submitting an inbound message first.
	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", MsgID: "m1",
		Kind: EventSubmit, Text: "hi",
	})

	// Stream a frame: should trigger placeholder + edit flow.
	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseStreaming, DraftText: "partial",
	}

	waitFor(t, 500*time.Millisecond, func() bool {
		return len(tg.sentSnapshot()) >= 1 && // placeholder or direct send
			len(tg.editsSnapshot()) >= 1
	})
}

func TestManager_Outbound_FinalFrameClearsTurn(t *testing.T) {
	tg := newFakeChannel("telegram")
	frames := make(chan kernel.RenderFrame, 8)
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		CoalesceMs:   10,
	}, fk, slog.Default())
	m.setRenderChan(frames)
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", MsgID: "m1",
		Kind: EventSubmit, Text: "hi",
	})

	frames <- kernel.RenderFrame{Phase: kernel.PhaseStreaming, DraftText: "p1"}
	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []kernel.HistoryEntry{
			{Role: "user", Content: "hi"},
			{Role: "assistant", Content: "hello back"},
		},
	}

	waitFor(t, 500*time.Millisecond, func() bool {
		m.turnMu.Lock()
		defer m.turnMu.Unlock()
		return m.turnPlatform == ""
	})
}
```

- [ ] **Step 7.2: Run test to verify it fails**

Run: `cd gormes && go test ./internal/gateway/... -run TestManager_Outbound`
Expected: compile error — `setRenderChan` undefined; Manager does not consume frames.

- [ ] **Step 7.3: Implement outbound loop**

In `internal/gateway/manager.go`:

1. Add `renderChan` field and `setRenderChan` helper:

```go
type Manager struct {
	cfg    ManagerConfig
	kernel kernelSubmitter
	log    *slog.Logger

	mu       sync.Mutex
	channels map[string]Channel

	turnMu       sync.Mutex
	turnPlatform string
	turnChatID   string
	turnMsgID    string

	renderChan <-chan kernel.RenderFrame // test override; nil means use m.kernel.Render()
}

// setRenderChan is a test helper that lets tests inject frames without
// building a real Kernel.
func (m *Manager) setRenderChan(c <-chan kernel.RenderFrame) { m.renderChan = c }
```

2. Extend `Run` to spawn an outbound goroutine:

```go
func (m *Manager) Run(ctx context.Context) error {
	m.mu.Lock()
	channels := make([]Channel, 0, len(m.channels))
	for _, ch := range m.channels {
		channels = append(channels, ch)
	}
	m.mu.Unlock()

	inbox := make(chan InboundEvent, len(channels)*4)

	var wg sync.WaitGroup
	for _, ch := range channels {
		wg.Add(1)
		go func(c Channel) {
			defer wg.Done()
			if err := c.Run(ctx, inbox); err != nil && !errors.Is(err, context.Canceled) {
				m.log.Warn("channel exited with error", "channel", c.Name(), "err", err)
			}
		}(ch)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		m.runOutbound(ctx)
	}()

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return nil
		case ev := <-inbox:
			m.handleInbound(ctx, ev)
		}
	}
}

func (m *Manager) runOutbound(ctx context.Context) {
	frames := m.renderChan
	if frames == nil {
		frames = m.kernel.Render()
	}

	var (
		co      *coalescer
		coCtx   context.Context
		coCancel context.CancelFunc
	)

	for {
		select {
		case <-ctx.Done():
			if coCancel != nil {
				coCancel()
			}
			return
		case f, ok := <-frames:
			if !ok {
				if coCancel != nil {
					coCancel()
				}
				return
			}
			m.persistSession(ctx, f)
			m.dispatchFrame(ctx, f, &co, &coCtx, &coCancel)
		}
	}
}

func (m *Manager) dispatchFrame(
	ctx context.Context,
	f kernel.RenderFrame,
	co **coalescer,
	coCtx *context.Context,
	coCancel *context.CancelFunc,
) {
	m.turnMu.Lock()
	platform := m.turnPlatform
	chatID := m.turnChatID
	m.turnMu.Unlock()

	if platform == "" || chatID == "" {
		// No pinned turn — ignore the frame.
		return
	}

	ch := m.lookupChannel(platform)
	if ch == nil {
		return
	}
	pe, ok := ch.(placeholderEditor)
	if !ok {
		// Channel cannot stream; only send final/error messages directly.
		m.sendFinalNoStream(ctx, ch, f, chatID)
		if f.Phase == kernel.PhaseIdle || f.Phase == kernel.PhaseFailed || f.Phase == kernel.PhaseCancelling {
			m.clearTurn()
		}
		return
	}

	switch f.Phase {
	case kernel.PhaseIdle:
		if *co != nil {
			(*co).flushImmediate(ctx, m.formatFinal(platform, f))
			(*coCancel)()
			*co = nil
		}
		m.clearTurn()
	case kernel.PhaseFailed, kernel.PhaseCancelling:
		text := m.formatError(platform, f)
		if *co != nil {
			(*co).flushImmediate(ctx, text)
			(*coCancel)()
			*co = nil
		} else {
			_, _ = ch.Send(ctx, chatID, text)
		}
		m.clearTurn()
	case kernel.PhaseConnecting, kernel.PhaseStreaming, kernel.PhaseReconnecting, kernel.PhaseFinalizing:
		if *co == nil {
			cCtx, cancel := context.WithCancel(ctx)
			*coCtx = cCtx
			*coCancel = cancel
			nc := newCoalescer(pe, time.Duration(m.cfg.CoalesceMs)*time.Millisecond, chatID)
			*co = nc
			go nc.run(cCtx)
		}
		(*co).setPending(m.formatStream(platform, f))
	}
}

func (m *Manager) sendFinalNoStream(ctx context.Context, ch Channel, f kernel.RenderFrame, chatID string) {
	switch f.Phase {
	case kernel.PhaseIdle:
		_, _ = ch.Send(ctx, chatID, m.formatFinal(ch.Name(), f))
	case kernel.PhaseFailed, kernel.PhaseCancelling:
		_, _ = ch.Send(ctx, chatID, m.formatError(ch.Name(), f))
	}
}

func (m *Manager) formatStream(platform string, f kernel.RenderFrame) string {
	if platform == "telegram" {
		return FormatStreamTelegram(f)
	}
	return FormatStreamPlain(f)
}

func (m *Manager) formatFinal(platform string, f kernel.RenderFrame) string {
	if platform == "telegram" {
		return FormatFinalTelegram(f)
	}
	return FormatFinalPlain(f)
}

func (m *Manager) formatError(platform string, f kernel.RenderFrame) string {
	if platform == "telegram" {
		return FormatErrorTelegram(f)
	}
	return FormatErrorPlain(f)
}

func (m *Manager) persistSession(ctx context.Context, f kernel.RenderFrame) {
	if m.cfg.SessionMap == nil {
		return
	}
	m.turnMu.Lock()
	platform := m.turnPlatform
	chatID := m.turnChatID
	m.turnMu.Unlock()
	if platform == "" || chatID == "" || f.SessionID == "" {
		return
	}
	key := platform + ":" + chatID
	if err := m.cfg.SessionMap.Put(ctx, key, f.SessionID); err != nil {
		m.log.Warn("persist session_id",
			"key", key,
			"session_id", f.SessionID,
			"err", err)
	}
}
```

Add `"time"` to the imports if not already present.

- [ ] **Step 7.4: Run tests**

Run: `cd gormes && go test ./internal/gateway/... -v`
Expected: all manager tests + coalescer tests PASS.

- [ ] **Step 7.5: Commit**

```bash
git add internal/gateway/manager.go internal/gateway/manager_test.go
git commit -m "$(cat <<'EOF'
feat(gateway): Manager outbound path via coalescer

Manager now consumes kernel.Render(), pins each frame to the turn
origin captured on inbound, and routes through a per-turn coalescer
(PlaceholderCapable + MessageEditor). Session-map persistence lifts
out of the Telegram adapter into Manager.persistSession. Per-channel
format variants (telegram vs plain) keep MarkdownV2 escaping behavior
identical for Telegram.
EOF
)"
```

---

## Task 8: Telegram refactor — move package + implement `gateway.Channel`

**Files:**
- Move: `internal/telegram/` → `internal/channels/telegram/`
- Modify: `internal/channels/telegram/bot.go` (implements `gateway.Channel`)
- Delete: `internal/channels/telegram/coalesce.go` (chassis owns it now)
- Delete: `internal/channels/telegram/coalesce_test.go`
- Modify: `cmd/gormes/telegram.go` (one-line import path update + builder change)

- [ ] **Step 8.1: Move the package**

```bash
mkdir -p internal/channels
git mv internal/telegram internal/channels/telegram
```

- [ ] **Step 8.2: Fix package imports across the repo**

Run: `cd gormes && grep -rl 'internal/telegram"' . | xargs sed -i 's|internal/telegram"|internal/channels/telegram"|g'`

Confirm nothing else references the old path.

- [ ] **Step 8.3: Delete the locally-inlined coalescer (chassis owns it)**

```bash
rm internal/channels/telegram/coalesce.go internal/channels/telegram/coalesce_test.go
```

- [ ] **Step 8.4: Refactor `bot.go` to a `gateway.Channel`**

The current `Bot` type runs both inbound AND outbound (coalescer spawn + frame consumption). Strip the outbound half — Manager owns it now. Replace the entire contents of `internal/channels/telegram/bot.go` with:

```go
package telegram

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// Config drives the Telegram channel. AllowedChatID lives in
// gateway.ManagerConfig.AllowedChats now, but the Bot still needs the
// int64 form because tgbotapi is typed.
type Config struct {
	AllowedChatID     int64
	FirstRunDiscovery bool
}

// Bot implements gateway.Channel + the streaming capabilities Manager
// needs (MessageEditor, PlaceholderCapable). No TypingCapable or
// ReactionCapable yet — both are trivial follow-ups.
type Bot struct {
	cfg    Config
	client telegramClient
	log    *slog.Logger

	mu       sync.Mutex
	chatIDByMsg map[string]int64 // msgID string -> int64 (discord-shaped key)
}

var _ gateway.Channel = (*Bot)(nil)
var _ gateway.MessageEditor = (*Bot)(nil)
var _ gateway.PlaceholderCapable = (*Bot)(nil)

// New constructs a Bot with the given tgbotapi client.
func New(cfg Config, client telegramClient, log *slog.Logger) *Bot {
	if log == nil {
		log = slog.Default()
	}
	return &Bot{cfg: cfg, client: client, log: log, chatIDByMsg: map[string]int64{}}
}

func (b *Bot) Name() string { return "telegram" }

func (b *Bot) Run(ctx context.Context, inbox chan<- gateway.InboundEvent) error {
	ucfg := tgbotapi.NewUpdate(0)
	ucfg.Timeout = 30
	updates := b.client.GetUpdatesChan(ucfg)
	for {
		select {
		case <-ctx.Done():
			b.client.StopReceivingUpdates()
			return nil
		case u, ok := <-updates:
			if !ok {
				return nil
			}
			if ev, ok := b.toInboundEvent(u); ok {
				select {
				case inbox <- ev:
				case <-ctx.Done():
					return nil
				}
			}
		}
	}
}

func (b *Bot) toInboundEvent(u tgbotapi.Update) (gateway.InboundEvent, bool) {
	if u.Message == nil {
		return gateway.InboundEvent{}, false
	}
	chatID := u.Message.Chat.ID
	text := strings.TrimSpace(u.Message.Text)

	kind := gateway.EventSubmit
	body := text
	switch {
	case text == "/start":
		kind = gateway.EventStart
		body = ""
	case text == "/stop":
		kind = gateway.EventCancel
		body = ""
	case text == "/new":
		kind = gateway.EventReset
		body = ""
	case strings.HasPrefix(text, "/"):
		kind = gateway.EventUnknown
		body = ""
	}

	return gateway.InboundEvent{
		Platform: "telegram",
		ChatID:   int64ToString(chatID),
		UserID:   userIDString(u.Message.From),
		MsgID:    int64ToString(int64(u.Message.MessageID)),
		Kind:     kind,
		Text:     body,
	}, true
}

func int64ToString(v int64) string {
	return strconvFormatInt(v)
}

// split out so tests can stub if they really need to. In practice this
// is strconv.FormatInt(v, 10) — inlined here to avoid one more import
// at the top of the file.
func strconvFormatInt(v int64) string {
	// Allocation-free for typical IDs. Telegram chat IDs can be negative
	// (group chats), so handle sign.
	return strconvFormatIntSlow(v)
}

func strconvFormatIntSlow(v int64) string {
	// Keep this wrapper trivial; full impl imported from the standard
	// strconv package below.
	return _strconvFormatInt(v)
}

// _strconvFormatInt is a thin rename so the function body above stays
// single-line readable. Defined at end of file.
var _strconvFormatInt = func(v int64) string {
	return strconvFormatInt10(v)
}

func strconvFormatInt10(v int64) string {
	return _strconv(v)
}

// Use the standard library.
var _strconv = func(v int64) string {
	return strconvShim(v)
}

func strconvShim(v int64) string {
	return _stdStrconv(v)
}

// Realistically: just import "strconv" and call strconv.FormatInt(v, 10)
// once. The maze above is unnecessary. Replace the six helpers above with
// the import and a single use-site.
```

Simplification — delete the `int64ToString` maze and use `strconv.FormatInt` directly. Rewrite `toInboundEvent` + tail of the file as:

```go
// (replace the Bot.toInboundEvent section and everything below it with:)

func (b *Bot) toInboundEvent(u tgbotapi.Update) (gateway.InboundEvent, bool) {
	if u.Message == nil {
		return gateway.InboundEvent{}, false
	}
	chatID := u.Message.Chat.ID
	text := strings.TrimSpace(u.Message.Text)

	kind := gateway.EventSubmit
	body := text
	switch {
	case text == "/start":
		kind = gateway.EventStart
		body = ""
	case text == "/stop":
		kind = gateway.EventCancel
		body = ""
	case text == "/new":
		kind = gateway.EventReset
		body = ""
	case strings.HasPrefix(text, "/"):
		kind = gateway.EventUnknown
		body = ""
	}

	var userID string
	if u.Message.From != nil {
		userID = strconv.FormatInt(u.Message.From.ID, 10)
	}

	return gateway.InboundEvent{
		Platform: "telegram",
		ChatID:   strconv.FormatInt(chatID, 10),
		UserID:   userID,
		MsgID:    strconv.Itoa(u.Message.MessageID),
		Kind:     kind,
		Text:     body,
	}, true
}

// Send sends a plain-text message. tgbotapi returns the new Message;
// we return its ID as string.
func (b *Bot) Send(ctx context.Context, chatID, text string) (string, error) {
	_ = ctx // tgbotapi.Send is synchronous
	id, err := parseChatID(chatID)
	if err != nil {
		return "", err
	}
	msg, err := b.client.Send(tgbotapi.NewMessage(id, text))
	if err != nil {
		return "", err
	}
	return strconv.Itoa(msg.MessageID), nil
}

// SendPlaceholder sends "⏳" and returns the created message ID.
func (b *Bot) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	return b.Send(ctx, chatID, "⏳")
}

// EditMessage edits an existing message by ID.
func (b *Bot) EditMessage(ctx context.Context, chatID, msgID, text string) error {
	_ = ctx
	cid, err := parseChatID(chatID)
	if err != nil {
		return err
	}
	mid, err := strconv.Atoi(msgID)
	if err != nil {
		return fmt.Errorf("telegram: invalid msgID %q: %w", msgID, err)
	}
	_, err = b.client.Send(tgbotapi.NewEditMessageText(cid, mid, text))
	return err
}

// SendToChat is retained for the cron delivery sink, which takes an
// int64 chat_id directly.
func (b *Bot) SendToChat(ctx context.Context, chatID int64, text string) error {
	_ = ctx
	_, err := b.client.Send(tgbotapi.NewMessage(chatID, text))
	return err
}

func parseChatID(s string) (int64, error) {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("telegram: invalid chat ID %q: %w", s, err)
	}
	return v, nil
}
```

Add to imports: `"fmt"`, `"strconv"`. Remove the unused ones (`"errors"`, `"sync"` if the chatIDByMsg map is gone, `"time"` if coalescer is gone, `"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"`, `"github.com/TrebuchetDynamics/gormes-agent/internal/session"`). Delete the entire `runOutbound`, `persistIfChanged`, `handleFrame`, and `handleUpdate` methods — they belong to Manager now.

- [ ] **Step 8.5: Update the existing telegram tests**

`bot_test.go` currently drives the old `Bot.Run` expecting it to submit to a kernel and emit frames through a coalescer. Replace its assertions with tests that target the new surface:

- `TestBot_Name` → `Name() == "telegram"`
- `TestBot_ToInboundEvent_Submit` → pushing a text update translates to `EventSubmit`
- `TestBot_ToInboundEvent_Start/Stop/New/Unknown` — slash command mapping
- `TestBot_ToInboundEvent_NilMessage` → edits and non-message updates dropped
- `TestBot_Send_RoundTrip` → mockClient observes a `NewMessage` chattable
- `TestBot_SendPlaceholder_ReturnsID` → placeholder send returns the mock's nextMsgID
- `TestBot_EditMessage` → mockClient observes a `NewEditMessageText` chattable

Replace the full `bot_test.go` content with this (preserves the mockClient helpers imported from `mock_test.go`):

```go
package telegram

import (
	"context"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestBot_Name(t *testing.T) {
	b := New(Config{AllowedChatID: 42}, newMockClient(), nil)
	if got := b.Name(); got != "telegram" {
		t.Errorf("Name() = %q, want telegram", got)
	}
}

func TestBot_ToInboundEvent_Submit(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)
	inbox := make(chan gateway.InboundEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	mc.pushTextUpdate(42, "hello there")

	select {
	case ev := <-inbox:
		if ev.Kind != gateway.EventSubmit || ev.Text != "hello there" {
			t.Errorf("got %+v", ev)
		}
		if ev.Platform != "telegram" || ev.ChatID != "42" {
			t.Errorf("got %+v", ev)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no inbound event")
	}
}

func TestBot_ToInboundEvent_Commands(t *testing.T) {
	cases := []struct {
		text string
		want gateway.EventKind
	}{
		{"/start", gateway.EventStart},
		{"/stop", gateway.EventCancel},
		{"/new", gateway.EventReset},
		{"/gibberish", gateway.EventUnknown},
		{"plain text", gateway.EventSubmit},
	}
	for _, c := range cases {
		t.Run(c.text, func(t *testing.T) {
			mc := newMockClient()
			b := New(Config{AllowedChatID: 42}, mc, nil)
			inbox := make(chan gateway.InboundEvent, 1)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() { _ = b.Run(ctx, inbox) }()
			mc.pushTextUpdate(42, c.text)
			select {
			case ev := <-inbox:
				if ev.Kind != c.want {
					t.Errorf("text=%q got Kind=%v want=%v", c.text, ev.Kind, c.want)
				}
			case <-time.After(200 * time.Millisecond):
				t.Fatal("no inbound event")
			}
		})
	}
}

func TestBot_Send(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)
	id, err := b.Send(context.Background(), "42", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Errorf("empty msg ID")
	}
	sent := mc.sentMessages()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d", len(sent))
	}
	if _, ok := sent[0].(tgbotapi.MessageConfig); !ok {
		t.Errorf("sent type = %T", sent[0])
	}
	if mc.lastSentText() != "hello" {
		t.Errorf("lastSentText = %q", mc.lastSentText())
	}
}

func TestBot_SendPlaceholder(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)
	id, err := b.SendPlaceholder(context.Background(), "42")
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Errorf("placeholder id empty")
	}
	if !strings.Contains(mc.lastSentText(), "⏳") {
		t.Errorf("placeholder text = %q", mc.lastSentText())
	}
}

func TestBot_EditMessage(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)
	if err := b.EditMessage(context.Background(), "42", "1234", "updated"); err != nil {
		t.Fatal(err)
	}
	sent := mc.sentMessages()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d", len(sent))
	}
	if _, ok := sent[0].(tgbotapi.EditMessageTextConfig); !ok {
		t.Errorf("sent type = %T want EditMessageTextConfig", sent[0])
	}
	if mc.lastSentText() != "updated" {
		t.Errorf("edit text = %q", mc.lastSentText())
	}
}

func TestBot_EditMessage_BadMsgID(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)
	if err := b.EditMessage(context.Background(), "42", "nope", "x"); err == nil {
		t.Fatal("expected error for non-numeric msgID")
	}
}

func TestBot_NilMessage_Skipped(t *testing.T) {
	mc := newMockClient()
	b := New(Config{AllowedChatID: 42}, mc, nil)
	inbox := make(chan gateway.InboundEvent, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	mc.updatesCh <- tgbotapi.Update{UpdateID: 7} // no Message field
	select {
	case ev := <-inbox:
		t.Fatalf("expected no inbound, got %+v", ev)
	case <-time.After(50 * time.Millisecond):
	}
}
```

- [ ] **Step 8.6: Update `cmd/gormes/telegram.go` to use Manager**

Open `gormes/cmd/gormes/telegram.go`. Replace the `bot := telegram.New(...)` + `return bot.Run(rootCtx)` tail with a Manager build:

```go
	tc, err := telegram.NewRealClient(cfg.Telegram.BotToken)
	if err != nil {
		return err
	}

	bot := telegram.New(telegram.Config{
		AllowedChatID:     cfg.Telegram.AllowedChatID,
		FirstRunDiscovery: cfg.Telegram.FirstRunDiscovery,
	}, tc, slog.Default())

	// Build a single-channel Manager — same observable behavior as the
	// previous direct-drive telegram.Bot, now flowing through the
	// chassis. Zero user-visible change; systemd units keep working.
	mgr := gateway.NewManager(gateway.ManagerConfig{
		AllowedChats: map[string]string{
			"telegram": strconv.FormatInt(cfg.Telegram.AllowedChatID, 10),
		},
		AllowDiscovery: map[string]bool{"telegram": cfg.Telegram.FirstRunDiscovery},
		CoalesceMs:     cfg.Telegram.CoalesceMs,
		SessionMap:     smap,
	}, k, slog.Default())
	if err := mgr.Register(bot); err != nil {
		return fmt.Errorf("register telegram: %w", err)
	}

	return mgr.Run(rootCtx)
```

Do not modify the cron scheduler block or the shutdown watchdog above this tail replacement. This step changes only the Telegram adapter construction + final `Run` call so the observable behavior stays byte-identical.

Add imports: `"strconv"`, `"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"`.

- [ ] **Step 8.7: Run the full test suite**

Run: `go test ./...`
Expected: green across all packages. Telegram tests reflect the new surface; chassis tests from earlier tasks still green; kernel + memory + tools unchanged.

- [ ] **Step 8.8: Commit**

```bash
git add -A gormes/
git commit -m "$(cat <<'EOF'
refactor(channels/telegram): move onto gateway chassis

Moves internal/telegram/ to internal/channels/telegram/ and
refactors Bot to satisfy gateway.Channel + MessageEditor +
PlaceholderCapable. The outbound half (coalescer spawn, frame
consumption, session-map persistence) moves to gateway.Manager;
Bot is now a pure SDK translator. cmd/gormes/telegram.go wraps a
single-channel Manager for byte-identical behavior; systemd units
do not need to change.
EOF
)"
```

---

## Task 9: Config — add `DiscordCfg`

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 9.1: Write the failing test**

Append to `internal/config/config_test.go`:

```go
func TestDiscordDefaults(t *testing.T) {
	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Discord.CoalesceMs != 1000 {
		t.Errorf("Discord.CoalesceMs default = %d, want 1000", cfg.Discord.CoalesceMs)
	}
	if cfg.Discord.Enabled() {
		t.Error("Discord should be disabled when no token is set")
	}
}

func TestDiscordLoadFromTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	os.WriteFile(path, []byte(`
[discord]
token = "bot-abc"
allowed_channel_id = "9999"
coalesce_ms = 500
first_run_discovery = true
`), 0644)
	t.Setenv("GORMES_CONFIG_FILE", path)

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Discord.Token != "bot-abc" {
		t.Errorf("Token = %q", cfg.Discord.Token)
	}
	if cfg.Discord.AllowedChannelID != "9999" {
		t.Errorf("AllowedChannelID = %q", cfg.Discord.AllowedChannelID)
	}
	if cfg.Discord.CoalesceMs != 500 {
		t.Errorf("CoalesceMs = %d", cfg.Discord.CoalesceMs)
	}
	if !cfg.Discord.FirstRunDiscovery {
		t.Error("FirstRunDiscovery = false, want true")
	}
	if !cfg.Discord.Enabled() {
		t.Error("Discord should be enabled with token + channel id")
	}
}
```

- [ ] **Step 9.2: Run test to verify it fails**

Run: `cd gormes && go test ./internal/config/... -run TestDiscord`
Expected: compile error — `Discord` field on Config undefined.

- [ ] **Step 9.3: Implement `DiscordCfg`**

Edit `internal/config/config.go`. In the `Config` struct, after `Telegram TelegramCfg`:

```go
	Discord DiscordCfg `toml:"discord"`
```

Add the type definition after `TelegramCfg`:

```go
// DiscordCfg drives the Discord channel adapter (Phase 2.B.2).
type DiscordCfg struct {
	// Token is the bot token from the Discord developer portal
	// ("Bot <token>" prefix is added by the adapter).
	Token string `toml:"token"`
	// AllowedChannelID restricts the bot to a single channel ID.
	// Discord channel IDs are 64-bit snowflakes; stored as string to
	// fit the session.Map key shape "discord:<id>".
	AllowedChannelID string `toml:"allowed_channel_id"`
	// CoalesceMs overrides gateway.ManagerConfig.CoalesceMs for the
	// Discord edge. Zero falls back to Manager default (1000).
	CoalesceMs int `toml:"coalesce_ms"`
	// FirstRunDiscovery logs inbound channel IDs from non-allowlisted
	// channels so the operator can populate AllowedChannelID.
	FirstRunDiscovery bool `toml:"first_run_discovery"`
}

// Enabled reports whether the Discord channel should be wired in
// this process. Token AND AllowedChannelID must be set (or
// FirstRunDiscovery must be true).
func (c DiscordCfg) Enabled() bool {
	if c.Token == "" {
		return false
	}
	return c.AllowedChannelID != "" || c.FirstRunDiscovery
}
```

In the defaults function (`withDefaults`), add:

```go
	if cfg.Discord.CoalesceMs == 0 {
		cfg.Discord.CoalesceMs = 1000
	}
```

- [ ] **Step 9.4: Run tests**

Run: `cd gormes && go test ./internal/config/... -v`
Expected: PASS for both new tests and all pre-existing config tests.

- [ ] **Step 9.5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "$(cat <<'EOF'
feat(config): DiscordCfg + [discord] TOML block

Adds the config surface the Phase 2.B.2 gateway wires. Enabled()
gates whether the Discord channel is registered by cmd/gormes.
Defaults match Telegram: 1000ms coalesce window, first-run
discovery off.
EOF
)"
```

---

## Task 10: Discord adapter — `discordSession` interface + mock

**Files:**
- Create: `internal/channels/discord/client.go`
- Create: `internal/channels/discord/mock_test.go`

- [ ] **Step 10.1: Add `discordgo` to `go.mod`**

```bash
cd gormes && go get github.com/bwmarrin/discordgo@latest
```

- [ ] **Step 10.2: Create `client.go`**

```go
// Package discord adapts Discord bot traffic into the gormes gateway
// chassis. Lives as a sibling to internal/channels/telegram — both
// implement gateway.Channel and the PlaceholderCapable, MessageEditor,
// and ReactionCapable sub-interfaces.
package discord

import "github.com/bwmarrin/discordgo"

// discordSession is the narrow surface of *discordgo.Session that the
// adapter uses. Keeping this interface tight means tests never need
// a live Discord websocket connection. Production wires a thin
// realSession wrapper in real_client.go.
type discordSession interface {
	// Open starts the websocket + begins receiving events.
	Open() error
	// Close tears down the websocket.
	Close() error
	// AddHandler registers an event callback. discordgo dispatches by
	// the second-arg type; we handle *discordgo.MessageCreate only.
	AddHandler(handler interface{}) func()
	// ChannelMessageSend sends a plain-text message. Returns the
	// created Message so the adapter can surface its ID.
	ChannelMessageSend(channelID, content string) (*discordgo.Message, error)
	// ChannelMessageEdit edits an existing message.
	ChannelMessageEdit(channelID, messageID, content string) (*discordgo.Message, error)
	// MessageReactionAdd adds a reaction to a message.
	MessageReactionAdd(channelID, messageID, emoji string) error
	// MessageReactionRemoveMe removes the bot's own reaction.
	MessageReactionRemoveMe(channelID, messageID, emoji string) error
}
```

- [ ] **Step 10.3: Create `mock_test.go`**

```go
package discord

import (
	"sync"

	"github.com/bwmarrin/discordgo"
)

type mockSession struct {
	mu              sync.Mutex
	opened          bool
	closed          bool
	handlers        []interface{}
	sent            []mockSent
	edits           []mockEdit
	reactionsAdded  []mockReaction
	reactionsRemove []mockReaction
	nextMsgID       int
	sendErr         error
	editErr         error
	reactionErr     error
}

type mockSent struct{ ChannelID, Content, MsgID string }
type mockEdit struct{ ChannelID, MsgID, Content string }
type mockReaction struct{ ChannelID, MsgID, Emoji string }

var _ discordSession = (*mockSession)(nil)

func newMockSession() *mockSession {
	return &mockSession{nextMsgID: 1000}
}

func (m *mockSession) Open() error {
	m.mu.Lock()
	m.opened = true
	m.mu.Unlock()
	return nil
}

func (m *mockSession) Close() error {
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
	return nil
}

func (m *mockSession) AddHandler(handler interface{}) func() {
	m.mu.Lock()
	m.handlers = append(m.handlers, handler)
	m.mu.Unlock()
	return func() {}
}

func (m *mockSession) ChannelMessageSend(channelID, content string) (*discordgo.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sendErr != nil {
		return nil, m.sendErr
	}
	id := nextID(&m.nextMsgID)
	m.sent = append(m.sent, mockSent{ChannelID: channelID, Content: content, MsgID: id})
	return &discordgo.Message{ID: id, ChannelID: channelID, Content: content}, nil
}

func (m *mockSession) ChannelMessageEdit(channelID, messageID, content string) (*discordgo.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.editErr != nil {
		return nil, m.editErr
	}
	m.edits = append(m.edits, mockEdit{ChannelID: channelID, MsgID: messageID, Content: content})
	return &discordgo.Message{ID: messageID, ChannelID: channelID, Content: content}, nil
}

func (m *mockSession) MessageReactionAdd(channelID, messageID, emoji string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.reactionErr != nil {
		return m.reactionErr
	}
	m.reactionsAdded = append(m.reactionsAdded, mockReaction{channelID, messageID, emoji})
	return nil
}

func (m *mockSession) MessageReactionRemoveMe(channelID, messageID, emoji string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reactionsRemove = append(m.reactionsRemove, mockReaction{channelID, messageID, emoji})
	return nil
}

// deliver invokes the first registered *discordgo.MessageCreate handler
// with a synthetic message. Returns false if no handler is registered
// or if its signature doesn't match.
func (m *mockSession) deliver(msg *discordgo.MessageCreate) bool {
	m.mu.Lock()
	handlers := append([]interface{}{}, m.handlers...)
	m.mu.Unlock()
	for _, h := range handlers {
		if fn, ok := h.(func(*discordgo.Session, *discordgo.MessageCreate)); ok {
			fn(nil, msg)
			return true
		}
	}
	return false
}

func (m *mockSession) sentSnapshot() []mockSent {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]mockSent, len(m.sent))
	copy(out, m.sent)
	return out
}

func (m *mockSession) editsSnapshot() []mockEdit {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]mockEdit, len(m.edits))
	copy(out, m.edits)
	return out
}

func (m *mockSession) reactionsAddedSnapshot() []mockReaction {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]mockReaction, len(m.reactionsAdded))
	copy(out, m.reactionsAdded)
	return out
}

func nextID(n *int) string {
	id := *n
	*n++
	return intToString(id)
}

func intToString(n int) string {
	// Avoid an extra import of strconv in a test-only file.
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}
```

- [ ] **Step 10.4: Run `go build`**

Run: `cd gormes && go build ./...`
Expected: successful build — mockSession compiles, discordgo dependency resolved.

- [ ] **Step 10.5: Commit**

```bash
git add internal/channels/discord/client.go internal/channels/discord/mock_test.go gormes/go.mod gormes/go.sum
git commit -m "$(cat <<'EOF'
feat(channels/discord): discordSession interface + mock

Tight interface mirrors the *discordgo.Session subset the adapter
needs. The mock backs every capability test the Discord bot gets
in follow-up tasks.
EOF
)"
```

---

## Task 11: Discord adapter — `Bot.Name/Run/Send` + inbound event translation

**Files:**
- Create: `internal/channels/discord/bot.go`
- Create: `internal/channels/discord/bot_test.go`

- [ ] **Step 11.1: Write the failing tests**

```go
package discord

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestBot_Name(t *testing.T) {
	b := New(Config{AllowedChannelID: "42"}, newMockSession(), nil)
	if b.Name() != "discord" {
		t.Errorf("Name() = %q", b.Name())
	}
}

func TestBot_ToInboundEvent_Submit(t *testing.T) {
	ms := newMockSession()
	b := New(Config{AllowedChannelID: "42"}, ms, nil)

	inbox := make(chan gateway.InboundEvent, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	// Allow Run to register its handler.
	time.Sleep(10 * time.Millisecond)

	ms.deliver(&discordgo.MessageCreate{Message: &discordgo.Message{
		ID:        "m99",
		ChannelID: "42",
		Content:   "hello from discord",
		Author:    &discordgo.User{ID: "u1", Bot: false},
	}})

	select {
	case ev := <-inbox:
		if ev.Platform != "discord" || ev.ChatID != "42" {
			t.Errorf("got %+v", ev)
		}
		if ev.Kind != gateway.EventSubmit || ev.Text != "hello from discord" {
			t.Errorf("got %+v", ev)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no inbound event")
	}
}

func TestBot_ToInboundEvent_IgnoresBotMessages(t *testing.T) {
	ms := newMockSession()
	b := New(Config{AllowedChannelID: "42"}, ms, nil)

	inbox := make(chan gateway.InboundEvent, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()
	time.Sleep(10 * time.Millisecond)

	ms.deliver(&discordgo.MessageCreate{Message: &discordgo.Message{
		ID: "m1", ChannelID: "42", Content: "bot reply",
		Author: &discordgo.User{ID: "b1", Bot: true},
	}})

	select {
	case ev := <-inbox:
		t.Fatalf("expected no inbound from bot, got %+v", ev)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestBot_ToInboundEvent_Commands(t *testing.T) {
	cases := []struct {
		text string
		want gateway.EventKind
	}{
		{"/start", gateway.EventStart},
		{"/stop", gateway.EventCancel},
		{"/new", gateway.EventReset},
		{"/xyzzy", gateway.EventUnknown},
		{"ordinary words", gateway.EventSubmit},
	}
	for _, c := range cases {
		t.Run(c.text, func(t *testing.T) {
			ms := newMockSession()
			b := New(Config{AllowedChannelID: "42"}, ms, nil)
			inbox := make(chan gateway.InboundEvent, 1)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() { _ = b.Run(ctx, inbox) }()
			time.Sleep(10 * time.Millisecond)

			ms.deliver(&discordgo.MessageCreate{Message: &discordgo.Message{
				ID: "1", ChannelID: "42", Content: c.text,
				Author: &discordgo.User{ID: "u", Bot: false},
			}})
			select {
			case ev := <-inbox:
				if ev.Kind != c.want {
					t.Errorf("text=%q got Kind=%v want=%v", c.text, ev.Kind, c.want)
				}
			case <-time.After(200 * time.Millisecond):
				t.Fatal("no inbound event")
			}
		})
	}
}

func TestBot_Send(t *testing.T) {
	ms := newMockSession()
	b := New(Config{AllowedChannelID: "42"}, ms, nil)

	id, err := b.Send(context.Background(), "42", "hi")
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Errorf("empty id")
	}
	sent := ms.sentSnapshot()
	if len(sent) != 1 || sent[0].Content != "hi" {
		t.Errorf("sent = %+v", sent)
	}
}

func TestBot_Send_ForwardsError(t *testing.T) {
	ms := newMockSession()
	ms.sendErr = errUnderlying
	b := New(Config{AllowedChannelID: "42"}, ms, nil)
	if _, err := b.Send(context.Background(), "42", "x"); err == nil {
		t.Fatal("expected send error")
	}
}

func TestBot_SendPlaceholder(t *testing.T) {
	ms := newMockSession()
	b := New(Config{AllowedChannelID: "42"}, ms, nil)

	id, err := b.SendPlaceholder(context.Background(), "42")
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Errorf("empty id")
	}
	sent := ms.sentSnapshot()
	if len(sent) != 1 || !strings.Contains(sent[0].Content, "⏳") {
		t.Errorf("placeholder content = %+v", sent)
	}
}

func TestBot_EditMessage(t *testing.T) {
	ms := newMockSession()
	b := New(Config{AllowedChannelID: "42"}, ms, nil)

	if err := b.EditMessage(context.Background(), "42", "m1", "updated"); err != nil {
		t.Fatal(err)
	}
	edits := ms.editsSnapshot()
	if len(edits) != 1 || edits[0].Content != "updated" {
		t.Errorf("edits = %+v", edits)
	}
}

func TestBot_ReactToMessage(t *testing.T) {
	ms := newMockSession()
	b := New(Config{AllowedChannelID: "42"}, ms, nil)

	undo, err := b.ReactToMessage(context.Background(), "42", "m1")
	if err != nil {
		t.Fatal(err)
	}
	if undo == nil {
		t.Fatal("undo is nil")
	}
	reacts := ms.reactionsAddedSnapshot()
	if len(reacts) != 1 {
		t.Errorf("reactions added = %+v", reacts)
	}

	// Idempotent.
	undo()
	undo()
}

var errUnderlying = &simpleErr{"underlying"}

type simpleErr struct{ s string }

func (e *simpleErr) Error() string { return e.s }
```

- [ ] **Step 11.2: Run test to verify it fails**

Run: `cd gormes && go test ./internal/channels/discord/...`
Expected: compile error — `New`, `Config`, `Bot` undefined.

- [ ] **Step 11.3: Implement `bot.go`**

```go
package discord

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// ackEmoji is the reaction added to every inbound message as a read
// acknowledgment. Eye emoji matches the picoclaw convention.
const ackEmoji = "👀"

// placeholderText mirrors the Telegram placeholder for the streaming
// edit flow — short so an operator recognizes "Gormes is thinking".
const placeholderText = "⏳"

type Config struct {
	AllowedChannelID  string
	FirstRunDiscovery bool
}

type Bot struct {
	cfg     Config
	session discordSession
	log     *slog.Logger

	// reactionsUndone tracks which (chatID, msgID) pairs have had their
	// ack reaction already removed, so the undo func is idempotent
	// even under concurrent Manager callbacks.
	reactionsMu sync.Mutex
	reactions   map[string]bool
}

var (
	_ gateway.Channel            = (*Bot)(nil)
	_ gateway.MessageEditor      = (*Bot)(nil)
	_ gateway.PlaceholderCapable = (*Bot)(nil)
	_ gateway.ReactionCapable    = (*Bot)(nil)
)

func New(cfg Config, session discordSession, log *slog.Logger) *Bot {
	if log == nil {
		log = slog.Default()
	}
	return &Bot{
		cfg:       cfg,
		session:   session,
		log:       log,
		reactions: map[string]bool{},
	}
}

func (b *Bot) Name() string { return "discord" }

func (b *Bot) Run(ctx context.Context, inbox chan<- gateway.InboundEvent) error {
	b.session.AddHandler(func(_ *discordgo.Session, m *discordgo.MessageCreate) {
		if m == nil || m.Message == nil {
			return
		}
		if m.Author == nil || m.Author.Bot {
			return
		}
		ev, ok := b.toInboundEvent(m.Message)
		if !ok {
			return
		}
		select {
		case inbox <- ev:
		case <-ctx.Done():
		}
	})
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("discord: open session: %w", err)
	}
	<-ctx.Done()
	_ = b.session.Close()
	return nil
}

func (b *Bot) toInboundEvent(m *discordgo.Message) (gateway.InboundEvent, bool) {
	text := strings.TrimSpace(m.Content)

	kind := gateway.EventSubmit
	body := text
	switch {
	case text == "/start":
		kind = gateway.EventStart
		body = ""
	case text == "/stop":
		kind = gateway.EventCancel
		body = ""
	case text == "/new":
		kind = gateway.EventReset
		body = ""
	case strings.HasPrefix(text, "/"):
		kind = gateway.EventUnknown
		body = ""
	}

	userID := ""
	if m.Author != nil {
		userID = m.Author.ID
	}
	return gateway.InboundEvent{
		Platform: "discord",
		ChatID:   m.ChannelID,
		UserID:   userID,
		MsgID:    m.ID,
		Kind:     kind,
		Text:     body,
	}, true
}

func (b *Bot) Send(ctx context.Context, chatID, text string) (string, error) {
	_ = ctx // discordgo is synchronous
	msg, err := b.session.ChannelMessageSend(chatID, text)
	if err != nil {
		return "", fmt.Errorf("discord: send: %w", err)
	}
	return msg.ID, nil
}

func (b *Bot) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	return b.Send(ctx, chatID, placeholderText)
}

func (b *Bot) EditMessage(ctx context.Context, chatID, msgID, text string) error {
	_ = ctx
	if _, err := b.session.ChannelMessageEdit(chatID, msgID, text); err != nil {
		return fmt.Errorf("discord: edit: %w", err)
	}
	return nil
}

func (b *Bot) ReactToMessage(ctx context.Context, chatID, msgID string) (func(), error) {
	_ = ctx
	if err := b.session.MessageReactionAdd(chatID, msgID, ackEmoji); err != nil {
		return nil, fmt.Errorf("discord: reaction add: %w", err)
	}
	key := chatID + ":" + msgID
	return func() {
		b.reactionsMu.Lock()
		if b.reactions[key] {
			b.reactionsMu.Unlock()
			return
		}
		b.reactions[key] = true
		b.reactionsMu.Unlock()
		_ = b.session.MessageReactionRemoveMe(chatID, msgID, ackEmoji)
	}, nil
}
```

- [ ] **Step 11.4: Run tests**

Run: `cd gormes && go test ./internal/channels/discord/... -v`
Expected: PASS for every test in the block.

- [ ] **Step 11.5: Commit**

```bash
git add internal/channels/discord/bot.go internal/channels/discord/bot_test.go
git commit -m "$(cat <<'EOF'
feat(channels/discord): Bot + inbound translation + capabilities

Discord channel implements gateway.Channel + MessageEditor +
PlaceholderCapable + ReactionCapable. Inbound MessageCreate events
are translated to gateway.InboundEvent with the same /start /stop
/new command set Telegram uses. Bot-authored messages are ignored
(otherwise the bot would reply to itself). Reaction undo is
idempotent.
EOF
)"
```

---

## Task 12: Discord real client — wrap `*discordgo.Session`

**Files:**
- Create: `internal/channels/discord/real_client.go`

- [ ] **Step 12.1: Implement the real session wrapper**

```go
package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// realSession wraps *discordgo.Session to satisfy discordSession.
// Production-only — tests use mockSession.
type realSession struct {
	s *discordgo.Session
}

var _ discordSession = (*realSession)(nil)

// NewRealSession constructs a live Discord session from a bot token.
// Fails fast if the token is missing or malformed (discordgo.New
// doesn't reach the network until Open, so an HTTP error here means
// the token string itself is unusable).
func NewRealSession(token string) (discordSession, error) {
	if token == "" {
		return nil, fmt.Errorf("discord: empty bot token")
	}
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("discord: new session: %w", err)
	}
	// Enable message content intent — required for reading message text
	// since Discord's August 2022 privileged-intents change.
	s.Identify.Intents = discordgo.IntentsGuildMessages |
		discordgo.IntentsDirectMessages |
		discordgo.IntentsMessageContent
	return &realSession{s: s}, nil
}

func (r *realSession) Open() error  { return r.s.Open() }
func (r *realSession) Close() error { return r.s.Close() }

func (r *realSession) AddHandler(handler interface{}) func() {
	return r.s.AddHandler(handler)
}

func (r *realSession) ChannelMessageSend(channelID, content string) (*discordgo.Message, error) {
	return r.s.ChannelMessageSend(channelID, content)
}

func (r *realSession) ChannelMessageEdit(channelID, messageID, content string) (*discordgo.Message, error) {
	return r.s.ChannelMessageEdit(channelID, messageID, content)
}

func (r *realSession) MessageReactionAdd(channelID, messageID, emoji string) error {
	return r.s.MessageReactionAdd(channelID, messageID, emoji)
}

func (r *realSession) MessageReactionRemoveMe(channelID, messageID, emoji string) error {
	return r.s.MessageReactionRemoveMe(channelID, messageID, emoji)
}
```

- [ ] **Step 12.2: Run build**

Run: `cd gormes && go build ./...`
Expected: clean build.

- [ ] **Step 12.3: Commit**

```bash
git add internal/channels/discord/real_client.go
git commit -m "$(cat <<'EOF'
feat(channels/discord): real session wrapper

NewRealSession builds a *discordgo.Session with the
GuildMessages+DirectMessages+MessageContent intent set — the minimum
privilege set that lets the bot read text content since Discord's
2022 intent change.
EOF
)"
```

---

## Task 13: `gormes gateway` cobra subcommand

**Files:**
- Create: `cmd/gormes/gateway.go`
- Modify: `cmd/gormes/main.go`

- [ ] **Step 13.1: Register the subcommand**

Open `cmd/gormes/main.go`. Replace:

```go
root.AddCommand(doctorCmd, versionCmd, telegramCmd)
```

with:

```go
root.AddCommand(doctorCmd, versionCmd, telegramCmd, gatewayCmd)
```

- [ ] **Step 13.2: Implement the subcommand**

Create `cmd/gormes/gateway.go`. The structure mirrors `telegram.go` but builds a multi-channel Manager:

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/channels/discord"
	"github.com/TrebuchetDynamics/gormes-agent/internal/channels/telegram"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

var gatewayCmd = &cobra.Command{
	Use:          "gateway",
	Short:        "Run Gormes as a multi-channel messaging gateway",
	Long:         "Runs every configured channel ([telegram], [discord], ...) through one gateway.Manager that drives the same kernel + tool loop as the TUI.",
	SilenceUsage: true,
	RunE:         runGateway,
}

func runGateway(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	if cfg.Telegram.BotToken == "" && !cfg.Discord.Enabled() {
		return fmt.Errorf("no channels configured — set at least one of [telegram] or [discord] in config.toml")
	}

	smap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		return fmt.Errorf("session map: %w", err)
	}
	defer smap.Close()

	ctx := context.Background()
	mstore, err := memory.OpenSqlite(config.MemoryDBPath(), cfg.Telegram.MemoryQueueCap, slog.Default())
	if err != nil {
		return fmt.Errorf("memory store: %w", err)
	}
	defer func() {
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), kernel.ShutdownBudget)
		defer cancelShutdown()
		if err := mstore.Close(shutdownCtx); err != nil {
			slog.Warn("memory store close", "err", err)
		}
	}()

	hc := hermes.NewHTTPClient(cfg.Hermes.Endpoint, cfg.Hermes.APIKey)

	reg := tools.NewRegistry()
	reg.MustRegister(&tools.EchoTool{})
	reg.MustRegister(&tools.NowTool{})
	reg.MustRegister(&tools.RandIntTool{})

	tm := telemetry.New()

	k := kernel.New(kernel.Config{
		Model:             cfg.Hermes.Model,
		Endpoint:          cfg.Hermes.Endpoint,
		Admission:         kernel.Admission{MaxBytes: cfg.Input.MaxBytes, MaxLines: cfg.Input.MaxLines},
		Tools:             reg,
		MaxToolIterations: 10,
		MaxToolDuration:   30 * time.Second,
	}, hc, mstore, tm, slog.Default())

	allowedChats := map[string]string{}
	allowDiscovery := map[string]bool{}

	mgr := gateway.NewManager(gateway.ManagerConfig{
		AllowedChats:   allowedChats,
		AllowDiscovery: allowDiscovery,
		CoalesceMs:     cfg.Telegram.CoalesceMs,
		SessionMap:     smap,
	}, k, slog.Default())

	if cfg.Telegram.BotToken != "" {
		tc, err := telegram.NewRealClient(cfg.Telegram.BotToken)
		if err != nil {
			return err
		}
		tgBot := telegram.New(telegram.Config{
			AllowedChatID:     cfg.Telegram.AllowedChatID,
			FirstRunDiscovery: cfg.Telegram.FirstRunDiscovery,
		}, tc, slog.Default())
		if err := mgr.Register(tgBot); err != nil {
			return fmt.Errorf("register telegram: %w", err)
		}
		allowedChats["telegram"] = strconv.FormatInt(cfg.Telegram.AllowedChatID, 10)
		allowDiscovery["telegram"] = cfg.Telegram.FirstRunDiscovery
		slog.Info("gateway: telegram channel enabled",
			"allowed_chat_id", cfg.Telegram.AllowedChatID)
	}

	if cfg.Discord.Enabled() {
		ds, err := discord.NewRealSession(cfg.Discord.Token)
		if err != nil {
			return err
		}
		dBot := discord.New(discord.Config{
			AllowedChannelID:  cfg.Discord.AllowedChannelID,
			FirstRunDiscovery: cfg.Discord.FirstRunDiscovery,
		}, ds, slog.Default())
		if err := mgr.Register(dBot); err != nil {
			return fmt.Errorf("register discord: %w", err)
		}
		allowedChats["discord"] = cfg.Discord.AllowedChannelID
		allowDiscovery["discord"] = cfg.Discord.FirstRunDiscovery
		slog.Info("gateway: discord channel enabled",
			"allowed_channel_id", cfg.Discord.AllowedChannelID)
	}

	rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go k.Run(rootCtx)

	go func() {
		<-rootCtx.Done()
		time.AfterFunc(kernel.ShutdownBudget, func() {
			slog.Error("shutdown budget exceeded; forcing exit")
			os.Exit(3)
		})
	}()

	slog.Info("gormes gateway starting",
		"channels", mgr.ChannelCount(),
		"endpoint", cfg.Hermes.Endpoint)

	_ = ctx // unused; kept for future per-startup ctx wiring
	return mgr.Run(rootCtx)
}
```

> **Scope note:** Keep `runGateway` bounded to the chassis work in this spec. Do **not** port the Telegram-only recall/extractor/semantic/mirror/cron wiring into this command as a mechanical follow-up. Those blocks depend on `kernel.Config.ChatKey` / `Recall` semantics that are process-scoped today, and §8 of the spec keeps kernel/memory redesign out of scope for Phase 2.B.2.

- [ ] **Step 13.3: Run build + smoke**

Run: `go build ./... && go test ./...`
Expected: clean build; no test regressions.

- [ ] **Step 13.4: Commit**

```bash
git add cmd/gormes/gateway.go cmd/gormes/main.go
git commit -m "$(cat <<'EOF'
feat(cmd/gormes): new gateway subcommand

Registers any configured channels with one gateway.Manager. Telegram
stays available via `gormes telegram` (single-channel Manager);
`gormes gateway` is the new entry point for multi-channel operation.
Telegram-only Phase 3 memory wiring stays on the telegram subcommand
until a later spec defines multi-channel recall semantics.
EOF
)"
```

---

## Task 14: Doctor — `CheckGateway`

**Files:**
- Modify: `cmd/gormes/doctor.go`

- [ ] **Step 14.1: Add the check**

Open `cmd/gormes/doctor.go`. Extend the import block with:

```go
"github.com/TrebuchetDynamics/gormes-agent/internal/channels/discord"
"github.com/TrebuchetDynamics/gormes-agent/internal/channels/telegram"
```

Then, immediately after `fmt.Print(result.Format())`, append:

```go
if cfg.Telegram.BotToken == "" && !cfg.Discord.Enabled() {
	fmt.Println("[WARN] gateway: no channels configured ([telegram] or [discord])")
} else {
	if cfg.Telegram.BotToken != "" {
		if _, err := telegram.NewRealClient(cfg.Telegram.BotToken); err != nil {
			fmt.Printf("[FAIL] gateway/telegram: %v\n", err)
			os.Exit(2)
		}
		fmt.Printf("[PASS] gateway/telegram: allowed_chat_id=%d\n", cfg.Telegram.AllowedChatID)
	} else {
		fmt.Println("[SKIP] gateway/telegram: disabled")
	}

	if cfg.Discord.Enabled() {
		if _, err := discord.NewRealSession(cfg.Discord.Token); err != nil {
			fmt.Printf("[FAIL] gateway/discord: %v\n", err)
			os.Exit(2)
		}
		fmt.Printf("[PASS] gateway/discord: allowed_channel_id=%s\n", cfg.Discord.AllowedChannelID)
	} else {
		fmt.Println("[SKIP] gateway/discord: disabled")
	}
}
```

- [ ] **Step 14.2: Run build**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 14.3: Commit**

```bash
git add cmd/gormes/doctor.go
git commit -m "$(cat <<'EOF'
feat(cmd/doctor): CheckGateway for telegram + discord

Verifies each enabled channel's SDK constructor accepts the configured
token. This replaces the absence-of-check for messaging — operators
now get an early error instead of discovering the token is invalid
when the first message arrives.
EOF
)"
```

---

## Task 15: Progress + docs update

**Files:**
- Modify: `docs/content/building-gormes/architecture_plan/progress.json`
- Modify: `docs/content/building-gormes/architecture_plan/phase-2-gateway.md`

- [ ] **Step 15.1: Update `progress.json`**

In the `phases["2"].subphases["2.B.2"]` block, set the Discord item's status to `"complete"` (Slack/WhatsApp/Signal/Email/SMS stay `"planned"`):

```json
"2.B.2": {
  "name": "Wider Gateway Surface",
  "items": [
    { "name": "Discord", "status": "complete" },
    { "name": "Slack", "status": "planned" },
    { "name": "WhatsApp", "status": "planned" },
    { "name": "Signal", "status": "planned" },
    { "name": "Email", "status": "planned" },
    { "name": "SMS", "status": "planned" }
  ]
}
```

Also bump `meta.last_updated` to the current date.

- [ ] **Step 15.2: Run the progress generator**

Run: `make generate-progress`
Expected: stdout includes `Regenerating progress-driven markdown...`, followed by `progress-gen: _index.md regenerated` and `progress-gen: README.md regenerated`.

- [ ] **Step 15.3: Commit**

```bash
git add docs/content/building-gormes/architecture_plan/ README.md
git commit -m "$(cat <<'EOF'
docs(progress): mark 2.B.2 Discord as shipped

Discord channel ported from picoclaw onto the new gateway chassis.
Slack / WhatsApp / Signal / Email / SMS remain planned — each gets
its own follow-up spec consuming the same chassis.
EOF
)"
```

---

## Task 16: Final cleanup — verify empty package + race check

**Files:**
- Nothing to modify in this task; only verification.

This task intentionally does **not** port the Telegram-only memory/recall/cron startup wiring into `cmd/gormes/gateway.go`. That work is out of scope for this spec because `kernel.Config.ChatKey` / `Recall` remain process-scoped and §8 explicitly forbids kernel redesign here.

- [ ] **Step 16.1: Verify `internal/telegram/` is gone**

Run: `test ! -d internal/telegram && echo "ok: old path absent"`
Expected: `ok: old path absent`.

Run: `grep -r 'internal/telegram"' . 2>/dev/null`
Expected: no matches.

- [ ] **Step 16.2: Final green check**

Run: `go test ./... -race`
Expected: PASS for every package.

- [ ] **Step 16.3: Done.** No commit needed for this step.

---

## Self-Review

**Spec coverage:**
- §3 principal decision 1 (chassis + Telegram + Discord) → Tasks 1–8 (chassis), Task 8 (Telegram refactor), Tasks 10–12 (Discord). ✅
- §3 principal decision 2 (capability interfaces, not god interface) → Task 2 (Channel + capabilities). ✅
- §3 principal decision 3 (Manager owns mechanics) → Tasks 3, 5, 7. ✅
- §3 principal decision 4 (kernel untouched) → no kernel task exists. ✅
- §3 principal decision 5 (single-session pinning) → Task 5 `pinTurn`, Task 7 `clearTurn`. ✅
- §3 micro: package path → Task 8 move. ✅
- §3 micro: `ChatKey "<platform>:<chat_id>"` → Task 1 `InboundEvent.ChatKey`. ✅
- §3 micro: discordgo → Task 10 go get. ✅
- §3 micro: subcommand rather than new binary → Task 13 `gatewayCmd`. ✅
- §3 micro: Discord scope (text, commands, streaming, reaction; no slash/embeds/threads) → Tasks 11–12. ✅
- §3 micro: coalesce window override → Task 9 `DiscordCfg.CoalesceMs`. ✅
- §3 micro: error model (log + degrade) → Task 4 coalescer, Task 11 Send/Edit forwarding. ✅
- §3 micro: testing seam → Task 10 `discordSession`. ✅
- §3 micro: doctor → Task 14. ✅
- §4.1 package tree → Tasks 1–8 + 11. ✅
- §4.2 core interfaces → Task 2. ✅
- §4.3 normalized events → Task 1. ✅
- §4.4 Manager responsibilities → Tasks 3, 5, 7. ✅
- §4.5 Discord adapter → Tasks 10–12. ✅
- §4.6 Telegram refactor → Task 8. ✅
- §4.7 `gormes gateway` subcommand → Task 13. ✅
- §6 Discord config → Task 9. ✅
- §7 TDD plan (10 items in spec) → expanded to 16 tasks in this plan. ✅
- §10 success criteria items 1–8 → covered by Tasks 1–15. ✅

**Placeholder scan:** Searched for TBD/TODO/"implement later" — none present. Task 4 initially contained a placeholder compat-shim sketch; replaced with the correct `git show` recovery step in the revised Step 4.5.

**Type consistency:**
- `InboundEvent` fields (Platform, ChatID, UserID, MsgID, Kind, Text) consistent across Tasks 1, 5, 8, 11. ✅
- `Channel` interface method names (Name, Run, Send) consistent across Tasks 2, 8, 11. ✅
- `MessageEditor.EditMessage`, `PlaceholderCapable.SendPlaceholder`, `ReactionCapable.ReactToMessage` names match across definition (Task 2) and implementations (Tasks 8, 11). ✅
- `Manager.NewManager` (prod) vs `NewManagerWithSubmitter` (test) — both introduced in Task 5 and used consistently in Tasks 7, 13. ✅
- `coalescer.setPending` / `flushImmediate` / `run` — Task 4 defines, Task 7 uses. ✅
- `DiscordCfg.Enabled()` — Task 9 defines, Tasks 13, 14 use. ✅

No drift found. Plan is ready for execution.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-21-gormes-phase2b2-chassis.md`. Two execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints.

Which approach?
