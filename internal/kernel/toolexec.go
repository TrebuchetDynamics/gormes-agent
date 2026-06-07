package kernel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel/toolpayload"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel/toolpreview"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/audit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// toolResult is the internal per-call output feeding back into the next
// ChatRequest as a role=tool Message.
type toolResult struct {
	ID           string
	Name         string
	Content      string // JSON string or text fallback — errors are JSON-encoded {"error":"..."}
	ContentParts []llm.MessageContentPart
}

type toolBatchOutcome struct {
	Results   []toolResult
	Cancelled bool
}

type indexedToolResult struct {
	Index  int
	Result toolResult
	Status string
	Err    error
	Audit  *audit.Record
}

var errToolExecutionCancelled = errors.New("tool execution cancelled")

const toolExecutionCancelledContent = `{"error":"tool execution cancelled"}`

// executeToolCalls runs tool calls with per-call timeout and panic recovery.
// It preserves result order for the model-facing transcript. Existing unit
// tests call this wrapper directly; runTurn uses executeToolCallsInterruptible
// so a PlatformEventCancel can stop the active tool batch.
func (k *Kernel) executeToolCalls(runCtx context.Context, calls []llm.ToolCall) []toolResult {
	return k.executeToolCallsInterruptible(runCtx, calls, "").Results
}

// executeToolCallsInterruptible fans a tool-call batch out to worker goroutines
// that share one cancellation context. The kernel goroutine stays in this
// function while workers run, so it can keep servicing k.events and propagate a
// single interrupt to every in-flight worker before returning one coherent
// cancellation envelope per call.
func (k *Kernel) executeToolCallsInterruptible(runCtx context.Context, calls []llm.ToolCall, turnKey string) toolBatchOutcome {
	results := make([]toolResult, len(calls))
	auditRecords := make([]*audit.Record, len(calls))
	if len(calls) == 0 {
		return toolBatchOutcome{Results: results}
	}

	execCtx, cancelAll := context.WithCancel(runCtx)
	defer cancelAll()

	resultCh := make(chan indexedToolResult, len(calls))
	for i, call := range calls {
		k.addSoul(toolpreview.SoulText(call))
		k.emitFrame("executing tool: " + call.Name)
		go func(index int, toolCall llm.ToolCall, sessionID string) {
			resultCh <- k.executeOneToolCall(execCtx, index, toolCall, sessionID, turnKey)
		}(i, call, k.sessionID)
	}

	cancelled := false
	runDone := runCtx.Done()
	remaining := len(calls)
	for remaining > 0 {
		select {
		case <-runDone:
			cancelled = true
			cancelAll()
			runDone = nil
		case e := <-k.events:
			switch e.Kind {
			case PlatformEventCancel, PlatformEventQuit:
				cancelled = true
				cancelAll()
				k.phase = PhaseCancelling
				k.emitFrame("cancelling tools")
			case PlatformEventSubmit:
				k.lastError = ErrTurnInFlight.Error()
				k.emitFrame("still processing previous turn")
			case PlatformEventResetSession:
				if e.ack != nil {
					e.ack <- ErrResetDuringTurn
				}
			case PlatformEventSteer:
				k.queueSteerGuidance(e.Text)
			}
		case res := <-resultCh:
			remaining--
			results[res.Index] = res.Result
			auditRecords[res.Index] = res.Audit
			switch res.Status {
			case "completed":
				// Channel-visible Hermes progress is keyed off tool.started.
				// Audit records and role=tool messages carry completion detail.
			case "cancelled":
				k.addSoul("tool cancelled: " + res.Result.Name)
			case "failed":
				if res.Err != nil {
					k.addSoul("tool error: " + res.Result.Name + ": " + res.Err.Error())
				} else {
					k.addSoul("tool error: " + res.Result.Name)
				}
			default:
				if res.Status != "" {
					k.addSoul("tool status: " + res.Result.Name + ": " + res.Status)
				}
			}
		}
	}

	if k.cfg.SubdirectoryHints != nil {
		for i, call := range calls {
			results[i] = k.appendSubdirectoryHint(call, results[i])
		}
	}

	for _, rec := range auditRecords {
		k.recordToolAudit(rec)
	}

	return toolBatchOutcome{Results: results, Cancelled: cancelled}
}

func (k *Kernel) executeOneToolCall(ctx context.Context, index int, call llm.ToolCall, sessionID, turnKey string) indexedToolResult {
	start := time.Now()
	buildAudit := func(status string, result json.RawMessage, err error) *audit.Record {
		if k.cfg.ToolAudit == nil {
			return nil
		}
		rec := audit.Record{
			Timestamp:       time.Now().UTC(),
			Source:          "kernel",
			SessionID:       sessionID,
			Tool:            call.Name,
			Args:            append(json.RawMessage(nil), call.Arguments...),
			DurationMs:      time.Since(start).Milliseconds(),
			Status:          status,
			ResultSizeBytes: len(result),
		}
		if err != nil {
			rec.Error = err.Error()
		}
		return &rec
	}
	k.observeGonchoToolCall(ctx, call, sessionID, turnKey)
	finish := func(res indexedToolResult) indexedToolResult {
		k.observeGonchoToolOutcome(ctx, call, sessionID, turnKey, res)
		return res
	}

	cancelled := func() indexedToolResult {
		return finish(indexedToolResult{
			Index:  index,
			Result: cancelledToolResult(call),
			Status: "cancelled",
			Err:    errToolExecutionCancelled,
			Audit:  buildAudit("cancelled", nil, errToolExecutionCancelled),
		})
	}

	select {
	case <-ctx.Done():
		return cancelled()
	default:
	}

	if k.cfg.ToolSafety != nil {
		decision := k.cfg.ToolSafety.DecideToolCall(call)
		if !decision.Allow {
			status := decision.Status
			if status == "" {
				status = "blocked"
			}
			payload := decision.Content
			if len(payload) == 0 {
				payload = json.RawMessage(fmt.Sprintf(`{"status":%q}`, status))
			}
			err := decision.Err
			if err == nil {
				err = errors.New(status)
			}
			result := newToolResult(call, payload)
			return finish(indexedToolResult{
				Index:  index,
				Result: result,
				Status: status,
				Err:    err,
				Audit:  buildAudit(status, payload, err),
			})
		}
	}

	executeContextEngineTool := func() indexedToolResult {
		payload, err := k.cfg.ContextEngine.HandleToolCall(ctx, call.Name, call.Arguments, llm.ContextToolCallOptions{})
		if len(payload) == 0 && err != nil {
			payload = json.RawMessage(fmt.Sprintf(`{"error":%q}`, err.Error()))
		}
		status := "completed"
		if err != nil {
			status = "failed"
		}
		result := newToolResult(call, payload)
		return finish(indexedToolResult{
			Index:  index,
			Result: result,
			Status: status,
			Err:    err,
			Audit:  buildAudit(status, payload, err),
		})
	}

	var tool tools.Tool
	if k.cfg.Tools != nil {
		var ok bool
		tool, ok = k.cfg.Tools.Get(call.Name)
		if !ok && k.cfg.ContextEngine != nil {
			return executeContextEngineTool()
		}
		if !ok {
			err := fmt.Errorf("unknown tool: %q", call.Name)
			result := toolResult{
				ID: call.ID, Name: call.Name,
				Content: fmt.Sprintf(`{"error":"unknown tool: %q"}`, call.Name),
			}
			return finish(indexedToolResult{
				Index:  index,
				Result: result,
				Status: "failed",
				Err:    err,
				Audit:  buildAudit("failed", nil, err),
			})
		}
	} else {
		if k.cfg.ContextEngine != nil {
			return executeContextEngineTool()
		}
		err := errors.New("no tool registry configured")
		result := toolResult{
			ID: call.ID, Name: call.Name,
			Content: `{"error":"no tool registry configured"}`,
		}
		return finish(indexedToolResult{
			Index:  index,
			Result: result,
			Status: "failed",
			Err:    err,
			Audit:  buildAudit("failed", nil, err),
		})
	}

	timeout := tool.Timeout()
	if timeout <= 0 {
		timeout = k.cfg.MaxToolDuration
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	payload, err := safeExecute(callCtx, tool, call.Arguments)
	callErr := callCtx.Err()
	cancel()

	if errors.Is(callErr, context.Canceled) || errors.Is(err, context.Canceled) {
		return cancelled()
	}

	if err == nil && errors.Is(callErr, context.DeadlineExceeded) {
		err = callErr
	}

	if err != nil {
		result := toolResult{
			ID: call.ID, Name: call.Name,
			Content: fmt.Sprintf(`{"error":%q}`, err.Error()),
		}
		return finish(indexedToolResult{
			Index:  index,
			Result: result,
			Status: "failed",
			Err:    err,
			Audit:  buildAudit("failed", nil, err),
		})
	}

	result := newToolResult(call, payload)
	return finish(indexedToolResult{
		Index:  index,
		Result: result,
		Status: "completed",
		Audit:  buildAudit("completed", payload, nil),
	})
}

func (k *Kernel) observeGonchoToolCall(ctx context.Context, call llm.ToolCall, sessionID, turnKey string) {
	k.observeGoncho(ctx, GonchoObservation{
		Kind:       GonchoObservationToolCall,
		PeerID:     "gormes",
		SessionKey: sessionID,
		ContextID:  gonchoToolContextID(turnKey, call.ID),
		Input:      string(call.Arguments),
		Metadata:   gonchoToolMetadata(call, turnKey, ""),
		Reason:     "gormes tool call capture",
	})
}

func (k *Kernel) observeGonchoToolOutcome(ctx context.Context, call llm.ToolCall, sessionID, turnKey string, res indexedToolResult) {
	success := res.Status == "completed"
	kind := GonchoObservationToolResult
	if !success {
		kind = GonchoObservationToolError
	}
	output := res.Result.Content
	if output == "" && res.Err != nil {
		output = res.Err.Error()
	}
	k.observeGoncho(ctx, GonchoObservation{
		Kind:       kind,
		PeerID:     "gormes",
		SessionKey: sessionID,
		ContextID:  gonchoToolContextID(turnKey, call.ID),
		Input:      string(call.Arguments),
		Output:     output,
		Success:    &success,
		Metadata:   gonchoToolMetadata(call, turnKey, res.Status),
		Reason:     "gormes tool result capture",
	})
}

func gonchoToolContextID(turnKey, callID string) string {
	switch {
	case turnKey != "" && callID != "":
		return turnKey + ":" + callID
	case callID != "":
		return callID
	default:
		return turnKey
	}
}

func gonchoToolMetadata(call llm.ToolCall, turnKey, status string) map[string]string {
	metadata := map[string]string{
		"source":       "kernel",
		"tool_name":    call.Name,
		"tool_call_id": call.ID,
	}
	if turnKey != "" {
		metadata["turn_key"] = turnKey
	}
	if status != "" {
		metadata["status"] = status
	}
	return metadata
}

func cancelledToolResult(call llm.ToolCall) toolResult {
	return toolResult{ID: call.ID, Name: call.Name, Content: toolExecutionCancelledContent}
}

func newToolResult(call llm.ToolCall, payload json.RawMessage) toolResult {
	result := toolResult{ID: call.ID, Name: call.Name, Content: string(payload)}
	if summary, parts, ok := toolpayload.MultimodalResult(payload); ok {
		result.Content = summary
		result.ContentParts = parts
	}
	return result
}

func (k *Kernel) appendSubdirectoryHint(call llm.ToolCall, result toolResult) toolResult {
	if k == nil || k.cfg.SubdirectoryHints == nil {
		return result
	}
	var args map[string]any
	if len(call.Arguments) == 0 || json.Unmarshal(call.Arguments, &args) != nil || len(args) == 0 {
		return result
	}
	hint := k.cfg.SubdirectoryHints.CheckToolCall(call.Name, args).Text
	if strings.TrimSpace(hint) == "" {
		return result
	}
	result.Content += hint
	if len(result.ContentParts) > 0 {
		result.ContentParts = toolpayload.AppendSubdirectoryHintToContentParts(result.ContentParts, hint)
	}
	return result
}

func (k *Kernel) recordToolAudit(rec *audit.Record) {
	if rec == nil || k.cfg.ToolAudit == nil {
		return
	}
	if auditErr := k.cfg.ToolAudit.Record(*rec); auditErr != nil && k.log != nil {
		k.log.Warn("kernel: append tool audit failed", "tool", rec.Tool, "err", auditErr)
	}
}

// safeExecute wraps Tool.Execute with panic recovery so a misbehaving tool
// cannot crash the kernel goroutine.
func safeExecute(ctx context.Context, t tools.Tool, args json.RawMessage) (result json.RawMessage, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("tool panicked: %v", r)
			result = nil
		}
	}()
	return t.Execute(ctx, args)
}
