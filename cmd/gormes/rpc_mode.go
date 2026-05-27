package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/audit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
)

func runResolvedRPC(cmd *cobra.Command, invocation rpcInvocation) error {
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	runtime := newKernelRPCRuntime(invocation)
	defer runtime.shutdown()
	if err := gateway.RunRPCMode(rootCtx, gateway.RPCModeOptions{
		In:      cmd.InOrStdin(),
		Out:     cmd.OutOrStdout(),
		Runtime: runtime,
	}); err != nil {
		return newExitCodeError(1, fmt.Errorf("gormes rpc: %w", err))
	}
	return nil
}

type kernelRPCRuntime struct {
	invocation rpcInvocation
	mu         sync.Mutex
	ctx        context.Context
	cancel     context.CancelFunc
	kernel     *kernel.Kernel
	render     <-chan kernel.RenderFrame
	runDone    chan error
	initialSeq uint64
	last       kernel.RenderFrame
	active     bool
	steering   []string
	followUp   []string
}

func newKernelRPCRuntime(invocation rpcInvocation) *kernelRPCRuntime {
	return &kernelRPCRuntime{invocation: invocation}
}

func (r *kernelRPCRuntime) Header(context.Context) gateway.RPCRecord {
	cwd, _ := os.Getwd()
	return gateway.RPCRecord{
		"type":       "session",
		"version":    1,
		"session_id": r.sessionIDSnapshot(),
		"cwd":        cwd,
		"mode":       "rpc",
	}
}

func (r *kernelRPCRuntime) State(context.Context) (gateway.RPCRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	frame := r.last
	model := frame.Model
	if strings.TrimSpace(model) == "" {
		model = r.resolvedModel()
	}
	return gateway.RPCRecord{
		"model":                 model,
		"provider":              r.invocation.Inference.Provider,
		"thinkingLevel":         string(frame.ReasoningEffort.Effort),
		"isStreaming":           r.active,
		"isCompacting":          false,
		"steeringMode":          "all",
		"followUpMode":          "one-at-a-time",
		"sessionId":             r.sessionIDLocked(),
		"autoCompactionEnabled": false,
		"messageCount":          len(frame.History),
		"pendingMessageCount":   len(r.steering) + len(r.followUp),
	}, nil
}

func (r *kernelRPCRuntime) Messages(context.Context) ([]gateway.RPCRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return rpcMessagesFromHermes(r.last.History), nil
}

func (r *kernelRPCRuntime) Prompt(ctx context.Context, req gateway.RPCPromptRequest) (<-chan gateway.RPCRecord, error) {
	r.mu.Lock()
	if r.active {
		r.mu.Unlock()
		return nil, errors.New("agent already streaming; send steer or follow_up")
	}
	if err := r.ensureKernelLocked(ctx); err != nil {
		r.mu.Unlock()
		return nil, err
	}
	r.active = true
	k := r.kernel
	render := r.render
	startSeq := r.initialSeq
	r.mu.Unlock()

	if err := k.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSubmit, Text: req.Message}); err != nil {
		r.mu.Lock()
		r.active = false
		r.mu.Unlock()
		return nil, err
	}

	out := make(chan gateway.RPCRecord, 16)
	go r.forwardPromptFrames(ctx, render, startSeq, out)
	return out, nil
}

func (r *kernelRPCRuntime) Steer(ctx context.Context, message string) (gateway.RPCQueueState, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return gateway.RPCQueueState{}, errors.New("message is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.steering = append(r.steering, message)
	if r.kernel != nil {
		_ = r.kernel.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSteer, Text: message})
	}
	return gateway.RPCQueueState{Steering: append([]string(nil), r.steering...), FollowUp: append([]string(nil), r.followUp...)}, nil
}

func (r *kernelRPCRuntime) FollowUp(_ context.Context, message string) (gateway.RPCQueueState, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return gateway.RPCQueueState{}, errors.New("message is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.followUp = append(r.followUp, message)
	return gateway.RPCQueueState{Steering: append([]string(nil), r.steering...), FollowUp: append([]string(nil), r.followUp...)}, nil
}

func (r *kernelRPCRuntime) Abort(context.Context) error {
	r.mu.Lock()
	k := r.kernel
	r.mu.Unlock()
	if k == nil {
		return nil
	}
	return k.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventCancel})
}

func (r *kernelRPCRuntime) ensureKernelLocked(ctx context.Context) error {
	if r.kernel != nil {
		return nil
	}
	cfg := r.invocation.Config
	client, err := getOrCreateProviderClient(cfg, r.invocation.Inference.Provider)
	if err != nil {
		return fmt.Errorf("provider setup failed: %s", redactRuntimeSecretText(err.Error(), cfg.Hermes.APIKey))
	}
	if client == nil {
		return errors.New("provider setup failed: nil hermes client")
	}
	model := r.resolvedModel()
	toolSafety, err := kernel.NewOneshotToolSafetyPolicy(kernel.OneshotToolSafetyOptions{TrustClass: kernel.TrustClassOperator})
	if err != nil {
		return fmt.Errorf("safety policy setup failed: %w", err)
	}
	kernelCfg := kernel.Config{
		Model:             model,
		Provider:          cfg.Hermes.Provider,
		Endpoint:          cfg.Hermes.Endpoint,
		Admission:         kernel.Admission{MaxBytes: cfg.Input.MaxBytes, MaxLines: cfg.Input.MaxLines},
		MaxToolIterations: configuredMaxToolIterations(cfg),
		ToolAudit:         audit.NewJSONLWriter(config.ToolAuditLogPath()),
		ToolSafety:        toolSafety,
		PrefillMessages:   configuredPrefillMessages(cfg),
	}
	if skillRuntime := newForcedSkillRuntime(cfg, r.invocation.ForcedSkills); skillRuntime != nil {
		kernelCfg.Skills = skillRuntime
		kernelCfg.SkillUsage = skillRuntime
	}
	r.ctx, r.cancel = context.WithCancel(ctx)
	r.kernel = kernel.New(kernelCfg, client, store.NewNoop(), telemetry.New(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	r.render = r.kernel.Render()
	r.runDone = make(chan error, 1)
	go func() { r.runDone <- r.kernel.Run(r.ctx) }()
	initial, err := readOneshotFrame(ctx, r.render)
	if err != nil {
		r.cancel()
		return fmt.Errorf("kernel startup failed: %w", err)
	}
	r.initialSeq = initial.Seq
	r.last = initial
	return nil
}

func (r *kernelRPCRuntime) forwardPromptFrames(ctx context.Context, render <-chan kernel.RenderFrame, startSeq uint64, out chan<- gateway.RPCRecord) {
	defer close(out)
	defer func() {
		r.mu.Lock()
		r.active = false
		r.initialSeq = r.last.Seq
		r.mu.Unlock()
	}()
	out <- gateway.RPCRecord{"type": "agent_start"}
	out <- gateway.RPCRecord{"type": "turn_start"}
	out <- gateway.RPCRecord{"type": "message_start", "message": gateway.RPCRecord{"role": "assistant", "content": ""}}
	lastDraft := ""
	for {
		frame, err := readOneshotFrame(ctx, render)
		if err != nil {
			out <- gateway.RPCRecord{"type": "agent_end", "error": err.Error()}
			return
		}
		r.mu.Lock()
		r.last = frame
		r.mu.Unlock()
		if strings.HasPrefix(frame.DraftText, lastDraft) && len(frame.DraftText) > len(lastDraft) {
			delta := frame.DraftText[len(lastDraft):]
			lastDraft = frame.DraftText
			out <- gateway.RPCRecord{"type": "message_update", "message": gateway.RPCRecord{"role": "assistant", "content": frame.DraftText}, "assistantMessageEvent": gateway.RPCRecord{"type": "text_delta", "delta": delta}}
		}
		if frame.LastError != "" || frame.Phase == kernel.PhaseFailed {
			out <- gateway.RPCRecord{"type": "message_end", "message": gateway.RPCRecord{"role": "assistant", "content": frame.DraftText}, "error": frame.LastError}
			out <- gateway.RPCRecord{"type": "agent_end", "messages": rpcMessagesFromHermes(frame.History), "error": frame.LastError}
			return
		}
		if frame.Phase == kernel.PhaseIdle && frame.Seq > startSeq {
			content, _ := finalAssistantContent(frame.History)
			if content != "" && content != lastDraft {
				out <- gateway.RPCRecord{"type": "message_update", "message": gateway.RPCRecord{"role": "assistant", "content": content}, "assistantMessageEvent": gateway.RPCRecord{"type": "text_delta", "delta": strings.TrimPrefix(content, lastDraft)}}
			}
			out <- gateway.RPCRecord{"type": "message_end", "message": gateway.RPCRecord{"role": "assistant", "content": content}}
			out <- gateway.RPCRecord{"type": "turn_end", "message": gateway.RPCRecord{"role": "assistant", "content": content}, "toolResults": []gateway.RPCRecord{}}
			out <- gateway.RPCRecord{"type": "agent_end", "messages": rpcMessagesFromHermes(frame.History)}
			return
		}
	}
}

func (r *kernelRPCRuntime) resolvedModel() string {
	model := r.invocation.Inference.Model
	if strings.TrimSpace(model) == "" {
		model = r.invocation.Config.Hermes.Model
	}
	return model
}

func (r *kernelRPCRuntime) sessionIDSnapshot() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sessionIDLocked()
}

func (r *kernelRPCRuntime) sessionIDLocked() string {
	if r.last.SessionID != "" {
		return r.last.SessionID
	}
	if r.invocation.NoSession {
		return ""
	}
	return "rpc"
}

func rpcMessagesFromHermes(messages []hermes.Message) []gateway.RPCRecord {
	out := make([]gateway.RPCRecord, 0, len(messages))
	for _, msg := range messages {
		rec := gateway.RPCRecord{"role": msg.Role, "content": msg.Content}
		if msg.ToolCallID != "" {
			rec["toolCallId"] = msg.ToolCallID
		}
		if msg.Name != "" {
			rec["toolName"] = msg.Name
		}
		out = append(out, rec)
	}
	return out
}

func (r *kernelRPCRuntime) shutdown() {
	r.mu.Lock()
	cancel := r.cancel
	done := r.runDone
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(kernel.ShutdownBudget):
		}
	}
}
