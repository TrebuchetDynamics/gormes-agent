// Package kernel is the single-owner state machine for Gormes. It owns the
// turn phase, the assistant draft buffer, the conversation history (in
// memory only in Phase 1), and the render snapshot. TUI, hermes, and store
// are edge adapters that communicate with the kernel through bounded mailboxes.
package kernel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/XelHaku/golang-hermes-agent/gormes/internal/hermes"
	"github.com/XelHaku/golang-hermes-agent/gormes/internal/store"
	"github.com/XelHaku/golang-hermes-agent/gormes/internal/telemetry"
)

// ErrEventMailboxFull is returned by Submit when the platform-event mailbox
// is saturated. The TUI should react by re-enabling input briefly; in
// practice this is rare with a 16-slot buffer.
var ErrEventMailboxFull = errors.New("kernel: event mailbox full")

type Config struct {
	Model     string
	Endpoint  string
	Admission Admission
}

type Kernel struct {
	cfg    Config
	client hermes.Client
	store  store.Store
	tm     *telemetry.Telemetry
	log    *slog.Logger

	render chan RenderFrame
	events chan PlatformEvent

	// Atomic — shared-read, kernel-write. Monotonically increasing per process.
	seq atomic.Uint64

	// All fields below this line are OWNED EXCLUSIVELY by the Run goroutine.
	// No other goroutine may read or write them without a channel-based
	// handshake. Violating this invariant is a race.
	phase     Phase
	draft     string
	history   []hermes.Message
	soul      []SoulEntry
	sessionID string
	lastError string
}

func New(cfg Config, c hermes.Client, s store.Store, tm *telemetry.Telemetry, log *slog.Logger) *Kernel {
	if log == nil {
		log = slog.Default()
	}
	tm.SetModel(cfg.Model)
	return &Kernel{
		cfg:    cfg,
		client: c,
		store:  s,
		tm:     tm,
		log:    log,
		render: make(chan RenderFrame, RenderMailboxCap),
		events: make(chan PlatformEvent, PlatformEventMailboxCap),
	}
}

// Render returns the receive side of the render mailbox. The channel is
// closed when Run exits.
func (k *Kernel) Render() <-chan RenderFrame { return k.render }

// Submit enqueues a platform event. Returns ErrEventMailboxFull if the
// mailbox is saturated; the caller decides whether to retry or drop.
// Safe to call from any goroutine.
func (k *Kernel) Submit(e PlatformEvent) error {
	select {
	case k.events <- e:
		return nil
	default:
		return ErrEventMailboxFull
	}
}

// Run is the kernel loop. MUST be called from exactly one goroutine. Exits
// when ctx is cancelled or a PlatformEventQuit is received. Closes the
// render channel on exit.
func (k *Kernel) Run(ctx context.Context) error {
	defer close(k.render)
	k.emitFrame("idle")
	for {
		select {
		case <-ctx.Done():
			return nil
		case e := <-k.events:
			switch e.Kind {
			case PlatformEventSubmit:
				if k.phase != PhaseIdle {
					k.lastError = ErrTurnInFlight.Error()
					k.emitFrame("still processing previous turn")
					continue
				}
				k.runTurn(ctx, e.Text)
			case PlatformEventCancel:
				// No active turn; ignore (cancel during a turn is handled
				// inside runTurn's select on k.events).
			case PlatformEventQuit:
				return nil
			}
		}
	}
}

// runTurn handles exactly one user turn end-to-end. On entry k.phase must be
// PhaseIdle; on exit it is PhaseIdle (or PhaseFailed on a fatal error).
// All state mutations happen on the calling goroutine, which is the Run
// goroutine — this is part of the single-owner invariant.
func (k *Kernel) runTurn(ctx context.Context, text string) {
	prov := newProvenance(k.cfg.Endpoint)

	// 1. Admission. Reject locally before any HTTP.
	if err := k.cfg.Admission.Validate(text); err != nil {
		k.lastError = err.Error()
		k.emitFrame(err.Error())
		return
	}
	prov.LogAdmitted(k.log)

	// 2. Persist user turn with hard 250ms ack deadline (spec §7.8 store row).
	storeCtx, storeCancel := context.WithTimeout(ctx, StoreAckDeadline)
	payload := []byte(fmt.Sprintf(`{"text":%q}`, text))
	_, err := k.store.Exec(storeCtx, store.Command{Kind: store.AppendUserTurn, Payload: payload})
	storeCancel()
	if err != nil {
		k.phase = PhaseFailed
		k.lastError = fmt.Sprintf("store ack timeout: %v", err)
		k.emitFrame(k.lastError)
		return
	}

	// 3. Update state for the new turn. These mutations are safe because we
	// are on the Run goroutine.
	k.history = append(k.history, hermes.Message{Role: "user", Content: text})
	k.draft = ""
	k.lastError = ""
	k.phase = PhaseConnecting
	k.emitFrame("connecting")
	prov.LogPOSTSent(k.log)

	// 4. Open the stream with a run-scoped context so cancel can cascade.
	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()

	stream, err := k.client.OpenStream(runCtx, hermes.ChatRequest{
		Model:     k.cfg.Model,
		SessionID: k.sessionID,
		Stream:    true,
		Messages:  []hermes.Message{{Role: "user", Content: text}},
	})
	if err != nil {
		prov.ErrorClass = hermes.Classify(err).String()
		prov.ErrorText = err.Error()
		prov.LogError(k.log)
		k.phase = PhaseFailed
		k.lastError = err.Error()
		k.emitFrame("open stream failed")
		return
	}
	defer stream.Close()

	k.phase = PhaseStreaming
	k.emitFrame("streaming")
	k.tm.StartTurn()
	start := time.Now()

	// 5. Pump goroutine: Recv in a loop, forward each result to deltaCh.
	// Exits on runCtx.Done or when Recv returns an error (including io.EOF).
	// Always closes deltaCh on exit — this is the cleanup handshake.
	type streamResult struct {
		event hermes.Event
		err   error
	}
	deltaCh := make(chan streamResult, 8)
	go func() {
		defer close(deltaCh)
		for {
			ev, err := stream.Recv(runCtx)
			// Forward or abort on cancel.
			select {
			case deltaCh <- streamResult{event: ev, err: err}:
			case <-runCtx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()

	// 6. Streaming loop with 16ms coalescer + concurrent event handling.
	ticker := time.NewTicker(FlushInterval)
	defer ticker.Stop()

	var (
		cancelled  bool
		fatalErr   error
		finalDelta hermes.Event
		gotFinal   bool
		dirty      bool
	)

streamLoop:
	for {
		select {
		case <-ctx.Done():
			cancelled = true
			cancelRun()
			break streamLoop

		case e := <-k.events:
			switch e.Kind {
			case PlatformEventCancel:
				cancelled = true
				cancelRun()
				// Drain continues below in the drain loop.
			case PlatformEventSubmit:
				// Reject — keep the current turn running; render a
				// rejection frame so the user sees immediate feedback.
				k.lastError = ErrTurnInFlight.Error()
				k.emitFrame("still processing previous turn")
			case PlatformEventQuit:
				cancelled = true
				cancelRun()
				break streamLoop
			}

		case res, ok := <-deltaCh:
			if !ok {
				// Pump exited — stream over.
				break streamLoop
			}
			if res.err != nil {
				if res.err == io.EOF {
					break streamLoop
				}
				if runCtx.Err() != nil {
					// Cancelled mid-recv; not a real error.
					cancelled = true
					break streamLoop
				}
				fatalErr = res.err
				break streamLoop
			}
			ev := res.event
			switch ev.Kind {
			case hermes.EventToken:
				k.draft += ev.Token
				k.tm.Tick(ev.TokensOut)
				dirty = true
			case hermes.EventReasoning:
				k.addSoul("reasoning: " + truncate(ev.Reasoning, 60))
				dirty = true
			case hermes.EventDone:
				finalDelta = ev
				gotFinal = true
				break streamLoop
			}

		case <-ticker.C:
			if dirty {
				k.emitFrame("streaming")
				dirty = false
			}
		}
	}

	// 7. Drain deltaCh to ensure the pump goroutine exits. cancelRun() has
	// been called if we cancelled; otherwise the pump is exiting on its own
	// via Recv returning io.EOF or an error. Either way, closing happens.
	cancelRun()
	for range deltaCh {
		// discard remaining results
	}

	// 8. Finalisation. At this point no other goroutine holds references to
	// kernel state; all further mutations are safe on the Run goroutine.
	latency := time.Since(start)
	k.tm.FinishTurn(latency)
	prov.LatencyMs = int(latency / time.Millisecond)

	if fatalErr != nil {
		prov.ErrorClass = hermes.Classify(fatalErr).String()
		prov.ErrorText = fatalErr.Error()
		prov.LogError(k.log)
		k.phase = PhaseFailed
		k.lastError = fatalErr.Error()
		k.emitFrame("stream error")
		return
	}

	if gotFinal {
		prov.FinishReason = finalDelta.FinishReason
		prov.TokensIn = finalDelta.TokensIn
		prov.TokensOut = finalDelta.TokensOut
		if finalDelta.TokensIn > 0 {
			k.tm.SetTokensIn(finalDelta.TokensIn)
		}
	}

	if sid := stream.SessionID(); sid != "" {
		k.sessionID = sid
		prov.ServerSessionID = sid
		prov.LogSSEStart(k.log)
	}

	if cancelled {
		k.phase = PhaseCancelling
		k.emitFrame("cancelled")
	} else if k.draft != "" {
		k.history = append(k.history, hermes.Message{Role: "assistant", Content: k.draft})
	}

	prov.LogDone(k.log)
	k.phase = PhaseIdle
	k.emitFrame("idle")
}

// addSoul appends a Soul Monitor entry with a ring-buffer cap.
func (k *Kernel) addSoul(text string) {
	k.soul = append(k.soul, SoulEntry{At: time.Now(), Text: text})
	if len(k.soul) > SoulBufferSize {
		k.soul = k.soul[len(k.soul)-SoulBufferSize:]
	}
}

// emitFrame builds a RenderFrame snapshot and publishes it to the render
// mailbox with replace-latest semantics: if an unread frame already sits
// in the capacity-1 buffer, drain it and drop it before enqueueing the new
// one. This is what keeps a slow TUI from backpressuring the kernel.
func (k *Kernel) emitFrame(status string) {
	frame := RenderFrame{
		Seq:        k.seq.Add(1),
		Phase:      k.phase,
		DraftText:  k.draft,
		History:    append([]hermes.Message(nil), k.history...),
		Telemetry:  k.tm.Snapshot(),
		StatusText: status,
		SessionID:  k.sessionID,
		Model:      k.cfg.Model,
		LastError:  k.lastError,
		SoulEvents: append([]SoulEntry(nil), k.soul...),
	}
	// Drain old frame if present, then enqueue new.
	select {
	case <-k.render:
	default:
	}
	select {
	case k.render <- frame:
	default:
		// Should be unreachable after the drain above.
	}
}

// truncate returns s clamped to n runes with an ellipsis suffix. Safe on
// non-ASCII input.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
