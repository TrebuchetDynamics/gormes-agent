package rpcmode

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/audit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

// ProviderClientFactory resolves the provider client used by the kernel-backed
// RPC runtime. cmd/gormes passes its shared provider pool to preserve existing
// startup/caching behavior.
type ProviderClientFactory func(config.Config, string) (llm.Client, error)

// KernelRuntimeOptions wires the local kernel into the stdio RPC protocol.
type KernelRuntimeOptions struct {
	Inference         config.InferenceResolution
	Config            config.Config
	ForcedSkills      []string
	NoSession         bool
	ProviderClient    ProviderClientFactory
	MaxToolIterations int
	PrefillMessages   []llm.Message
	RedactSecretText  func(string, ...string) string
}

// KernelRuntime adapts the local Gormes kernel to the LF-framed RPC runtime
// contract.
type KernelRuntime struct {
	options KernelRuntimeOptions
	mu      sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
	kernel  *kernel.Kernel
	render  <-chan kernel.RenderFrame
	runDone chan error

	initialSeq uint64
	last       kernel.RenderFrame
	active     bool
	steering   []string
	followUp   []string
}

func NewKernelRuntime(options KernelRuntimeOptions) *KernelRuntime {
	return &KernelRuntime{options: options}
}

func (r *KernelRuntime) Header(context.Context) RPCRecord {
	cwd, _ := os.Getwd()
	return RPCRecord{
		"type":       "session",
		"version":    1,
		"session_id": r.sessionIDSnapshot(),
		"cwd":        cwd,
		"mode":       "rpc",
	}
}

func (r *KernelRuntime) State(context.Context) (RPCRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	frame := r.last
	model := frame.Model
	if strings.TrimSpace(model) == "" {
		model = r.resolvedModel()
	}
	return RPCRecord{
		"model":                 model,
		"provider":              r.options.Inference.Provider,
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

func (r *KernelRuntime) Messages(context.Context) ([]RPCRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return rpcMessagesFromHermes(r.last.History), nil
}

func (r *KernelRuntime) Prompt(ctx context.Context, req RPCPromptRequest) (<-chan RPCRecord, error) {
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

	out := make(chan RPCRecord, 16)
	go r.forwardPromptFrames(ctx, render, startSeq, out)
	return out, nil
}

func (r *KernelRuntime) Steer(_ context.Context, message string) (RPCQueueState, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return RPCQueueState{}, errors.New("message is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.steering = append(r.steering, message)
	if r.kernel != nil {
		_ = r.kernel.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSteer, Text: message})
	}
	return RPCQueueState{Steering: append([]string(nil), r.steering...), FollowUp: append([]string(nil), r.followUp...)}, nil
}

func (r *KernelRuntime) FollowUp(_ context.Context, message string) (RPCQueueState, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return RPCQueueState{}, errors.New("message is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.followUp = append(r.followUp, message)
	return RPCQueueState{Steering: append([]string(nil), r.steering...), FollowUp: append([]string(nil), r.followUp...)}, nil
}

func (r *KernelRuntime) Abort(context.Context) error {
	r.mu.Lock()
	k := r.kernel
	r.mu.Unlock()
	if k == nil {
		return nil
	}
	return k.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventCancel})
}

func (r *KernelRuntime) ensureKernelLocked(ctx context.Context) error {
	if r.kernel != nil {
		return nil
	}
	cfg := r.options.Config
	if r.options.ProviderClient == nil {
		return errors.New("provider setup failed: provider client factory is required")
	}
	client, err := r.options.ProviderClient(cfg, r.options.Inference.Provider)
	if err != nil {
		return fmt.Errorf("provider setup failed: %s", r.redact(err.Error(), cfg.Hermes.APIKey))
	}
	if client == nil {
		return errors.New("provider setup failed: nil hermes client")
	}
	model := r.resolvedModel()
	toolSafety, err := kernel.NewOneshotToolSafetyPolicy(kernel.OneshotToolSafetyOptions{TrustClass: kernel.TrustClassOperator})
	if err != nil {
		return fmt.Errorf("safety policy setup failed: %w", err)
	}
	maxToolIterations := r.options.MaxToolIterations
	if maxToolIterations <= 0 {
		maxToolIterations = kernel.DefaultMaxToolIterations
	}
	kernelCfg := kernel.Config{
		Model:             model,
		Provider:          cfg.Hermes.Provider,
		Endpoint:          cfg.Hermes.Endpoint,
		Admission:         kernel.Admission{MaxBytes: cfg.Input.MaxBytes, MaxLines: cfg.Input.MaxLines},
		MaxToolIterations: maxToolIterations,
		ToolAudit:         audit.NewJSONLWriter(config.ToolAuditLogPath()),
		ToolSafety:        toolSafety,
		PrefillMessages:   append([]llm.Message(nil), r.options.PrefillMessages...),
	}
	if skillRuntime := newForcedSkillRuntime(cfg, r.options.ForcedSkills); skillRuntime != nil {
		kernelCfg.Skills = skillRuntime
		kernelCfg.SkillUsage = skillRuntime
	}
	r.ctx, r.cancel = context.WithCancel(ctx)
	r.kernel = kernel.New(kernelCfg, client, store.NewNoop(), telemetry.New(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	r.render = r.kernel.Render()
	r.runDone = make(chan error, 1)
	go func() { r.runDone <- r.kernel.Run(r.ctx) }()
	initial, err := readKernelFrame(ctx, r.render)
	if err != nil {
		r.cancel()
		return fmt.Errorf("kernel startup failed: %w", err)
	}
	r.initialSeq = initial.Seq
	r.last = initial
	return nil
}

func (r *KernelRuntime) forwardPromptFrames(ctx context.Context, render <-chan kernel.RenderFrame, startSeq uint64, out chan<- RPCRecord) {
	defer close(out)
	defer func() {
		r.mu.Lock()
		r.active = false
		r.initialSeq = r.last.Seq
		r.mu.Unlock()
	}()
	out <- RPCRecord{"type": "agent_start"}
	out <- RPCRecord{"type": "turn_start"}
	out <- RPCRecord{"type": "message_start", "message": RPCRecord{"role": "assistant", "content": ""}}
	lastDraft := ""
	for {
		frame, err := readKernelFrame(ctx, render)
		if err != nil {
			out <- RPCRecord{"type": "agent_end", "error": rpcErrorText(err.Error())}
			return
		}
		r.mu.Lock()
		r.last = frame
		r.mu.Unlock()
		if strings.HasPrefix(frame.DraftText, lastDraft) && len(frame.DraftText) > len(lastDraft) {
			delta := frame.DraftText[len(lastDraft):]
			lastDraft = frame.DraftText
			out <- RPCRecord{"type": "message_update", "message": RPCRecord{"role": "assistant", "content": frame.DraftText}, "assistantMessageEvent": RPCRecord{"type": "text_delta", "delta": delta}}
		}
		if frame.LastError != "" || frame.Phase == kernel.PhaseFailed {
			errorText := rpcErrorText(frame.LastError)
			out <- RPCRecord{"type": "message_end", "message": RPCRecord{"role": "assistant", "content": frame.DraftText}, "error": errorText}
			out <- RPCRecord{"type": "agent_end", "messages": rpcMessagesFromHermes(frame.History), "error": errorText}
			return
		}
		if frame.Phase == kernel.PhaseIdle && frame.Seq > startSeq {
			content, _ := finalAssistantContent(frame.History)
			if content != "" && content != lastDraft {
				out <- RPCRecord{"type": "message_update", "message": RPCRecord{"role": "assistant", "content": content}, "assistantMessageEvent": RPCRecord{"type": "text_delta", "delta": strings.TrimPrefix(content, lastDraft)}}
			}
			out <- RPCRecord{"type": "message_end", "message": RPCRecord{"role": "assistant", "content": content}}
			out <- RPCRecord{"type": "turn_end", "message": RPCRecord{"role": "assistant", "content": content}, "toolResults": []RPCRecord{}}
			out <- RPCRecord{"type": "agent_end", "messages": rpcMessagesFromHermes(frame.History)}
			return
		}
	}
}

func (r *KernelRuntime) resolvedModel() string {
	model := r.options.Inference.Model
	if strings.TrimSpace(model) == "" {
		model = r.options.Config.Hermes.Model
	}
	return model
}

func (r *KernelRuntime) sessionIDSnapshot() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sessionIDLocked()
}

func (r *KernelRuntime) sessionIDLocked() string {
	if r.last.SessionID != "" {
		return r.last.SessionID
	}
	if r.options.NoSession {
		return ""
	}
	return "rpc"
}

func (r *KernelRuntime) redact(text string, secrets ...string) string {
	if r.options.RedactSecretText == nil {
		return text
	}
	return r.options.RedactSecretText(text, secrets...)
}

// Shutdown cancels the backing kernel and waits up to kernel.ShutdownBudget for
// it to exit.
func (r *KernelRuntime) Shutdown() {
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

func rpcMessagesFromHermes(messages []llm.Message) []RPCRecord {
	out := make([]RPCRecord, 0, len(messages))
	for _, msg := range messages {
		rec := RPCRecord{"role": msg.Role, "content": msg.Content}
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

func readKernelFrame(ctx context.Context, frames <-chan kernel.RenderFrame) (kernel.RenderFrame, error) {
	select {
	case frame, ok := <-frames:
		if !ok {
			return kernel.RenderFrame{}, errors.New("render stream closed")
		}
		return frame, nil
	case <-ctx.Done():
		return kernel.RenderFrame{}, ctx.Err()
	}
}

func finalAssistantContent(history []llm.Message) (string, bool) {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "assistant" {
			return history[i].Content, true
		}
	}
	return "", false
}

type forcedSkillRuntime struct {
	runtime *skills.Runtime
	names   []string
}

func newForcedSkillRuntime(cfg config.Config, names []string) *forcedSkillRuntime {
	if len(names) == 0 {
		return nil
	}
	return &forcedSkillRuntime{
		runtime: skills.NewRuntime(cfg.SkillsRoot(), cfg.Skills.MaxDocumentBytes, cfg.Skills.SelectionCap, cfg.SkillsUsageLogPath()),
		names:   append([]string(nil), names...),
	}
}

func (r *forcedSkillRuntime) BuildSkillBlock(ctx context.Context, userMessage string) (string, []string, error) {
	if r == nil || r.runtime == nil || len(r.names) == 0 {
		return "", nil, nil
	}
	allowed := make(map[string]bool, len(r.names))
	for _, name := range r.names {
		name = strings.TrimSpace(name)
		if name != "" {
			allowed[strings.ToLower(name)] = true
		}
	}
	query := strings.TrimSpace(strings.Join(append([]string{userMessage}, r.names...), " "))
	block, names, _, err := r.runtime.BuildSkillBlockWithOptions(ctx, query, skills.RuntimeOptions{AllowedSkillNames: allowed})
	return block, names, err
}

func (r *forcedSkillRuntime) RecordSkillUsage(ctx context.Context, skillNames []string) error {
	if r == nil || r.runtime == nil {
		return nil
	}
	return r.runtime.RecordSkillUsage(ctx, skillNames)
}
