package kernel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/audit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

// cancelEnvelope is the single coherent cancel envelope every concurrent
// tool worker returns when the parent context is cancelled. Phase 2.E.2
// freezes this shape so parallel tool_calls fan-out, sidecar sandbox jobs,
// and delegated subagent children can interoperate under one interrupt
// contract.
const cancelEnvelope = `{"error":"cancelled","reason":"parent_cancelled"}`

// executeToolCallsConcurrent runs tool calls in parallel, bounded by
// maxConcurrent, and preserves the input order in the returned slice.
// When runCtx is cancelled, every in-flight and queued worker returns the
// single coherent cancel envelope; no result is dropped. Each cancelled
// call emits an audit record with status "cancelled" so observers see a
// consistent picture. Unlike executeToolCalls (sequential), this entry
// point is the seam that upstream parallel tool_calls, sandbox sidecars,
// and delegate_task children will share once Phase 2.F.5 lands on top.
func (k *Kernel) executeToolCallsConcurrent(runCtx context.Context, calls []hermes.ToolCall, maxConcurrent int) []toolResult {
	if len(calls) == 0 {
		return nil
	}
	if maxConcurrent <= 0 {
		maxConcurrent = len(calls)
	}

	results := make([]toolResult, len(calls))
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	wg.Add(len(calls))

	for i := range calls {
		i, call := i, calls[i]
		go func() {
			defer wg.Done()
			start := time.Now()

			// Acquire a slot or observe parent cancel.
			select {
			case sem <- struct{}{}:
			case <-runCtx.Done():
				results[i] = toolResult{ID: call.ID, Name: call.Name, Content: cancelEnvelope}
				k.recordToolAudit(auditArgs{
					start:  start,
					call:   call,
					status: "cancelled",
					err:    errors.New("cancelled before execution"),
				})
				return
			}
			defer func() { <-sem }()

			// If the parent cancelled while we were queued, short-circuit.
			if err := runCtx.Err(); err != nil {
				results[i] = toolResult{ID: call.ID, Name: call.Name, Content: cancelEnvelope}
				k.recordToolAudit(auditArgs{
					start:  start,
					call:   call,
					status: "cancelled",
					err:    errors.New("cancelled before execution"),
				})
				return
			}

			results[i] = k.executeSingleCall(runCtx, call, start)
		}()
	}

	wg.Wait()
	return results
}

// executeSingleCall is a one-call helper shared by the concurrent executor.
// It mirrors the sequential path's contract for unknown-tool, panic, and
// timeout handling, but normalises parent-cancel outcomes to the coherent
// cancel envelope so fan-out observers see one stable shape.
func (k *Kernel) executeSingleCall(runCtx context.Context, call hermes.ToolCall, start time.Time) toolResult {
	if k.cfg.Tools == nil {
		err := errors.New("no tool registry configured")
		k.recordToolAudit(auditArgs{start: start, call: call, status: "failed", err: err})
		return toolResult{
			ID: call.ID, Name: call.Name,
			Content: `{"error":"no tool registry configured"}`,
		}
	}

	tool, ok := k.cfg.Tools.Get(call.Name)
	if !ok {
		err := fmt.Errorf("unknown tool: %q", call.Name)
		k.recordToolAudit(auditArgs{start: start, call: call, status: "failed", err: err})
		return toolResult{
			ID: call.ID, Name: call.Name,
			Content: fmt.Sprintf(`{"error":"unknown tool: %q"}`, call.Name),
		}
	}

	timeout := tool.Timeout()
	if timeout <= 0 {
		timeout = k.cfg.MaxToolDuration
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	callCtx, cancel := context.WithTimeout(runCtx, timeout)
	defer cancel()

	payload, err := safeExecute(callCtx, tool, call.Arguments)

	// If the parent context was cancelled during Execute, fold into the
	// coherent cancel envelope even if the tool returned a deadline error.
	if runCtx.Err() != nil {
		k.recordToolAudit(auditArgs{
			start:  start,
			call:   call,
			status: "cancelled",
			err:    runCtx.Err(),
		})
		return toolResult{ID: call.ID, Name: call.Name, Content: cancelEnvelope}
	}

	if err != nil {
		k.recordToolAudit(auditArgs{start: start, call: call, status: "failed", err: err})
		return toolResult{
			ID: call.ID, Name: call.Name,
			Content: fmt.Sprintf(`{"error":%q}`, err.Error()),
		}
	}

	k.recordToolAudit(auditArgs{
		start:  start,
		call:   call,
		status: "completed",
		result: payload,
	})
	return toolResult{ID: call.ID, Name: call.Name, Content: string(payload)}
}

type auditArgs struct {
	start  time.Time
	call   hermes.ToolCall
	status string
	result json.RawMessage
	err    error
}

// recordToolAudit is a small helper that keeps the concurrent executor free
// of inline audit boilerplate. Nil-audit and nil-log are both safe.
func (k *Kernel) recordToolAudit(a auditArgs) {
	if k.cfg.ToolAudit == nil {
		return
	}
	rec := audit.Record{
		Timestamp:       time.Now().UTC(),
		Source:          "kernel",
		SessionID:       k.sessionID,
		Tool:            a.call.Name,
		Args:            append(json.RawMessage(nil), a.call.Arguments...),
		DurationMs:      time.Since(a.start).Milliseconds(),
		Status:          a.status,
		ResultSizeBytes: len(a.result),
	}
	if a.err != nil {
		rec.Error = a.err.Error()
	}
	if err := k.cfg.ToolAudit.Record(rec); err != nil && k.log != nil {
		k.log.Warn("kernel: append tool audit failed", "tool", a.call.Name, "err", err)
	}
}
