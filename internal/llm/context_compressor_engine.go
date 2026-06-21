package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
)

var ErrContextSummaryUnavailable = errors.New("hermes: context summary unavailable")

// ContextSummaryRequest is the provider-facing payload for one context
// compression summary update. PreviousSummary is the current iterative summary
// body without the compaction handoff prefix. TurnsToSummarize contains only
// fresh turns that should be incorporated into the new summary.
type ContextSummaryRequest struct {
	PreviousSummary  string
	TurnsToSummarize []Message
	FocusTopic       string
	MaxSummaryTokens int
}

// ContextSummarizer generates the body of a context compaction summary. The
// provider-backed implementation is ClientContextSummarizer; tests and callers
// can inject deterministic summarizers through ContextSummarizerFunc.
type ContextSummarizer interface {
	SummarizeContext(ctx context.Context, req ContextSummaryRequest) (string, error)
}

type ContextSummarizerFunc func(context.Context, ContextSummaryRequest) (string, error)

func (f ContextSummarizerFunc) SummarizeContext(ctx context.Context, req ContextSummaryRequest) (string, error) {
	if f == nil {
		return "", ErrContextSummaryUnavailable
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return f(ctx, req)
}

// ClientContextSummarizer uses the shared LLM client boundary to perform the
// auxiliary provider call that Hermes uses for context compression.
type ClientContextSummarizer struct {
	Client Client
	Model  string
}

func (s ClientContextSummarizer) SummarizeContext(ctx context.Context, req ContextSummaryRequest) (string, error) {
	if s.Client == nil {
		return "", ErrContextSummaryUnavailable
	}
	maxTokens := req.MaxSummaryTokens
	if maxTokens <= 0 {
		maxTokens = contextCompressorSummaryTokensCeiling
	}
	stream, err := s.Client.OpenStream(ctx, ChatRequest{
		Model:     s.Model,
		MaxTokens: maxTokens,
		Stream:    true,
		Messages:  []Message{{Role: "user", Content: BuildContextSummaryPrompt(req)}},
	})
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrContextSummaryUnavailable, err)
	}
	defer stream.Close()

	var b strings.Builder
	for {
		event, err := stream.Recv(ctx)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrContextSummaryUnavailable, err)
		}
		if event.Kind == EventToken {
			b.WriteString(event.Token)
		}
		if event.Kind == EventDone {
			break
		}
	}
	summary := strings.TrimSpace(b.String())
	if summary == "" {
		return "", ErrContextSummaryUnavailable
	}
	return summary, nil
}

// BuildContextSummaryPrompt renders the Hermes-style single-prompt summary
// request. It intentionally preserves the upstream headings used to distinguish
// first summaries from iterative updates.
func BuildContextSummaryPrompt(req ContextSummaryRequest) string {
	turns := serializeMessagesForContextSummary(req.TurnsToSummarize)
	var b strings.Builder
	b.WriteString("You are a summarization agent creating a context checkpoint. Treat the conversation turns below as source material for a compact record of prior work. Produce only the structured summary; do not add a greeting, preamble, or prefix. NEVER include API keys, tokens, passwords, secrets, credentials, or connection strings in the summary — replace any that appear with [REDACTED].")
	if strings.TrimSpace(req.PreviousSummary) != "" {
		b.WriteString("\n\nYou are updating a context compaction summary. A previous compaction produced the summary below. New conversation turns have occurred since then and need to be incorporated.\n\nPREVIOUS SUMMARY:\n")
		b.WriteString(strings.TrimSpace(req.PreviousSummary))
		b.WriteString("\n\nNEW TURNS TO INCORPORATE:\n")
		b.WriteString(turns)
		b.WriteString("\n\nUpdate the summary using this exact structure. PRESERVE existing information that is still relevant. ADD new completed actions. Update ## Active Task to reflect the user's most recent unfulfilled input.")
	} else {
		b.WriteString("\n\nCreate a structured checkpoint summary for the conversation after earlier turns are compacted. The summary should preserve enough detail for continuity without re-reading the original turns.\n\nTURNS TO SUMMARIZE:\n")
		b.WriteString(turns)
		b.WriteString("\n\nUse this exact structure:")
	}
	b.WriteString("\n\n## Active Task\n[Most recent unfulfilled user input]\n\n## Goal\n[Overall objective]\n\n## Constraints & Preferences\n[Constraints, preferences, decisions]\n\n## Completed Actions\n[Concrete actions taken]\n\n## Active State\n[Current state]\n\n## In Progress\n[Work started but not finished]\n\n## Pending User Asks\n[Questions or asks still pending]\n\n## Relevant Files\n[Files and why they matter]\n\n## Remaining Work\n[Remaining context, not instructions]\n\n## Critical Context\n[Specific non-secret details that would be lost]")
	if req.MaxSummaryTokens > 0 {
		b.WriteString(fmt.Sprintf("\n\nTarget ~%d tokens. Be concrete and preserve file paths, command outputs, errors, line numbers, and specific values.", req.MaxSummaryTokens))
	}
	if strings.TrimSpace(req.FocusTopic) != "" {
		focus := strings.TrimSpace(req.FocusTopic)
		b.WriteString(fmt.Sprintf("\n\nFOCUS TOPIC: %q\nThe user has requested that this compaction prioritise preserving information related to this focus topic. For unrelated content, summarize more aggressively. Never preserve secrets; use [REDACTED].", focus))
	}
	b.WriteString("\n\nWrite only the summary body. Do not include any preamble or prefix.")
	return b.String()
}

func serializeMessagesForContextSummary(messages []Message) string {
	if len(messages) == 0 {
		return "[No new turns to summarize]"
	}
	parts := make([]string, 0, len(messages))
	for _, msg := range messages {
		role := strings.ToUpper(strings.TrimSpace(msg.Role))
		if role == "" {
			role = "UNKNOWN"
		}
		content := strings.TrimSpace(messageSummaryText(msg))
		if content == "" && len(msg.ToolCalls) > 0 {
			content = summarizeToolCallsForContextSummary(msg.ToolCalls)
		}
		if msg.Role == "tool" && msg.Name != "" {
			role += " " + msg.Name
		}
		parts = append(parts, fmt.Sprintf("[%s]: %s", role, content))
	}
	return strings.Join(parts, "\n\n")
}

func messageSummaryText(msg Message) string {
	if len(msg.ContentParts) == 0 {
		return msg.Content
	}
	parts := make([]string, 0, len(msg.ContentParts))
	for _, part := range msg.ContentParts {
		switch {
		case strings.TrimSpace(part.Text) != "":
			parts = append(parts, part.Text)
		case strings.TrimSpace(part.ImageURL) != "":
			parts = append(parts, "[image removed from summary prompt]")
		}
	}
	if len(parts) == 0 {
		return msg.Content
	}
	return strings.Join(parts, "\n")
}

func summarizeToolCallsForContextSummary(calls []ToolCall) string {
	items := make([]string, 0, len(calls))
	for _, call := range calls {
		items = append(items, fmt.Sprintf("tool call %s(%s)", call.Name, trimForSummary(string(call.Arguments), 200)))
	}
	return strings.Join(items, "; ")
}

type ProviderBackedContextEngineConfig struct {
	Model              string
	ContextLength      int
	ThresholdPercent   float64
	SummaryTargetRatio float64
	ProtectFirstN      int
	TailTokenBudget    int
	MinTailMessages    int
	ToolResultMaxChars int
	ToolDescriptors    []ToolDescriptor
	Summarizer         ContextSummarizer
}

// ProviderBackedContextEngine is an explicit ContextEngine implementation for
// Hermes-style context compression. It does not auto-compress normal turns; the
// kernel or channel surface must call Compress and then commit the returned
// transcript boundary.
type ProviderBackedContextEngine struct {
	mu                 sync.Mutex
	budget             *ContextCompressorBudget
	summarizer         ContextSummarizer
	protectFirstN      int
	tailTokenBudget    int
	minTailMessages    int
	toolResultMaxChars int
	previousSummary    string
	status             ContextStatus
}

var _ ContextEngine = (*ProviderBackedContextEngine)(nil)

func NewProviderBackedContextEngine(cfg ProviderBackedContextEngineConfig) *ProviderBackedContextEngine {
	threshold := normalizeContextCompressorThresholdPercent(cfg.ThresholdPercent)
	budget := NewContextCompressorBudget(ContextCompressorBudgetConfig{
		Model:              cfg.Model,
		ContextLength:      cfg.ContextLength,
		ThresholdPercent:   threshold,
		SummaryTargetRatio: cfg.SummaryTargetRatio,
		ToolDescriptors:    cfg.ToolDescriptors,
	})
	minTail := cfg.MinTailMessages
	if minTail <= 0 {
		minTail = 3
	}
	toolResultMax := cfg.ToolResultMaxChars
	if toolResultMax <= 0 {
		toolResultMax = 8_000
	}
	engine := &ProviderBackedContextEngine{
		budget:             budget,
		summarizer:         cfg.Summarizer,
		protectFirstN:      cfg.ProtectFirstN,
		tailTokenBudget:    cfg.TailTokenBudget,
		minTailMessages:    minTail,
		toolResultMaxChars: toolResultMax,
		status: ContextStatus{
			Engine:           "provider_backed",
			Model:            cfg.Model,
			ContextLength:    cfg.ContextLength,
			ThresholdPercent: threshold,
			Compression: ContextCompressionStatus{
				Enabled: true,
			},
			Boundary: ContextBoundaryStatus{
				Status:  ContextBoundaryStatusMissing,
				Message: "no compression boundary recorded",
			},
			Tools: ContextToolStatus{StatusTool: ContextStatusToolName},
		},
	}
	engine.refreshLocked()
	return engine
}

func (e *ProviderBackedContextEngine) Name() string { return "provider_backed" }

func (e *ProviderBackedContextEngine) UpdateFromResponse(usage ContextUsage) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status.LastPromptTokens = usage.PromptTokens
	e.status.LastCompletionTokens = usage.CompletionTokens
	if usage.TotalTokens > 0 {
		e.status.LastTotalTokens = usage.TotalTokens
	} else {
		e.status.LastTotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	e.refreshLocked()
}

// effectiveProtectFirstNLocked returns the active protect_first_n value, which
// decays to 0 after the first compression pass. Once a session has been
// compressed at least once, early turns are captured in the handoff summary, so
// re-protecting them would fossilize them unboundedly across repeated passes
// (Hermes #11996). Caller must hold e.mu.
func (e *ProviderBackedContextEngine) effectiveProtectFirstNLocked() int {
	if e.status.CompressionCount >= 1 || e.previousSummary != "" {
		return 0
	}
	return e.protectFirstN
}

func (e *ProviderBackedContextEngine) ShouldCompress(promptTokens int) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	status := e.budget.Status()
	return status.State == "ready" && status.ThresholdTokens > 0 && promptTokens >= status.ThresholdTokens
}

func (e *ProviderBackedContextEngine) Compress(ctx context.Context, messages []Message, req CompressionRequest) ([]Message, CompressionReport, error) {
	e.mu.Lock()
	budgetStatus := e.budget.Status()
	protectFirstN := e.effectiveProtectFirstNLocked()
	tailTokenBudget := e.tailTokenBudget
	if tailTokenBudget <= 0 {
		tailTokenBudget = budgetStatus.TailTokenBudget
	}
	minTailMessages := e.minTailMessages
	toolResultMaxChars := e.toolResultMaxChars
	previousSummary := e.previousSummary
	summarizer := e.summarizer
	e.mu.Unlock()

	report := CompressionReport{
		State:          "skipped",
		BeforeMessages: len(messages),
		AfterMessages:  len(messages),
		CurrentTokens:  req.CurrentTokens,
		FocusTopic:     req.FocusTopic,
	}
	if budgetStatus.State != "ready" || tailTokenBudget <= 0 {
		err := fmt.Errorf("%w: compression budget unavailable", ErrContextSummaryUnavailable)
		e.recordCompressionError(err)
		return cloneMessages(messages), report, err
	}
	boundary := PlanContextCompressionBoundary(messages, ContextCompressionBoundaryOptions{
		ProtectFirstN:   protectFirstN,
		TailTokenBudget: tailTokenBudget,
	})
	if !boundary.HasContentToCompress {
		return cloneMessages(messages), report, nil
	}
	lineage := PlanContextSummaryLineage(messages, boundary.CompressStart, boundary.TailStart, previousSummary)
	if len(lineage.TurnsToSummarize) == 0 {
		return cloneMessages(messages), report, nil
	}
	if summarizer == nil {
		err := ErrContextSummaryUnavailable
		e.recordCompressionError(err)
		return cloneMessages(messages), report, err
	}
	summary, err := summarizer.SummarizeContext(ctx, ContextSummaryRequest{
		PreviousSummary:  lineage.PreviousSummary,
		TurnsToSummarize: lineage.TurnsToSummarize,
		FocusTopic:       req.FocusTopic,
		MaxSummaryTokens: budgetStatus.MaxSummaryTokens,
	})
	if err != nil {
		err = fmt.Errorf("%w: %v", ErrContextSummaryUnavailable, err)
		e.recordCompressionError(err)
		return cloneMessages(messages), report, err
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		err := ErrContextSummaryUnavailable
		e.recordCompressionError(err)
		return cloneMessages(messages), report, err
	}

	compressed, pruning := PruneContextMessages(messages, ContextPruningConfig{
		ProtectFirstN:        protectFirstN,
		TailTokenBudget:      tailTokenBudget,
		MinTailMessages:      minTailMessages,
		ToolResultPruneChars: toolResultMaxChars,
		SummaryText:          summary,
	})
	report.State = "compressed"
	report.AfterMessages = len(compressed)
	if pruning.State == ContextPruningStateSkipped {
		report.State = "skipped"
	}
	e.mu.Lock()
	e.previousSummary = StripContextPruningSummaryPrefix(summary)
	e.status.Compression.LastError = ""
	e.status.Compression.CooldownSeconds = 0
	e.refreshLocked()
	e.mu.Unlock()
	return compressed, report, nil
}

func (e *ProviderBackedContextEngine) OnCompressionBoundary(_ context.Context, boundary CompressionBoundary) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status.CompressionCount++
	e.status.Boundary = ContextBoundaryStatus{Status: ContextBoundaryStatusRecorded, Last: cloneCompressionBoundary(boundary)}
	e.refreshLocked()
	return nil
}

func (e *ProviderBackedContextEngine) ShouldCompressPreflight(messages []Message) bool {
	e.mu.Lock()
	status := e.budget.Status()
	e.mu.Unlock()
	return status.State == "ready" && e.HasContentToCompress(messages)
}

func (e *ProviderBackedContextEngine) HasContentToCompress(messages []Message) bool {
	e.mu.Lock()
	budgetStatus := e.budget.Status()
	protectFirstN := e.protectFirstN
	tailTokenBudget := e.tailTokenBudget
	e.mu.Unlock()
	if tailTokenBudget <= 0 {
		tailTokenBudget = budgetStatus.TailTokenBudget
	}
	if tailTokenBudget <= 0 {
		return false
	}
	return PlanContextCompressionBoundary(messages, ContextCompressionBoundaryOptions{
		ProtectFirstN:   protectFirstN,
		TailTokenBudget: tailTokenBudget,
	}).HasContentToCompress
}

func (e *ProviderBackedContextEngine) OnSessionStart(context.Context, string, ContextSessionMeta) error {
	return nil
}

func (e *ProviderBackedContextEngine) OnSessionEnd(context.Context, string, []Message) error {
	return nil
}

func (e *ProviderBackedContextEngine) OnSessionReset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.previousSummary = ""
	e.status.LastPromptTokens = 0
	e.status.LastCompletionTokens = 0
	e.status.LastTotalTokens = 0
	e.status.CompressionCount = 0
	e.status.Boundary = ContextBoundaryStatus{}
	e.status.Tools.UnknownToolErrors = nil
	e.status.Compression.LastError = ""
	e.refreshLocked()
}

func (e *ProviderBackedContextEngine) ToolDescriptors() []ToolDescriptor {
	return []ToolDescriptor{ContextStatusToolDescriptor()}
}

func (e *ProviderBackedContextEngine) HandleToolCall(_ context.Context, name string, _ json.RawMessage, _ ContextToolCallOptions) (json.RawMessage, error) {
	if name == ContextStatusToolName {
		status := e.Status()
		return json.Marshal(status)
	}
	toolErr := unknownContextToolError(name)
	e.mu.Lock()
	e.status.Tools.UnknownToolErrors = append(e.status.Tools.UnknownToolErrors, toolErr)
	e.refreshLocked()
	e.mu.Unlock()
	payload, err := json.Marshal(struct {
		Error ContextToolError `json:"error"`
	}{Error: toolErr})
	if err != nil {
		return nil, err
	}
	return payload, fmt.Errorf("%w: %s", ErrUnknownContextTool, name)
}

func (e *ProviderBackedContextEngine) Status() ContextStatus {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.refreshLocked()
	return cloneContextStatus(e.status)
}

func (e *ProviderBackedContextEngine) UpdateModelContext(update ContextModelContext) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.budget.UpdateModelContext(update)
	e.refreshLocked()
}

func (e *ProviderBackedContextEngine) recordCompressionError(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status.Compression.LastError = contextCompressionErrorText(err)
	e.refreshLocked()
}

func contextCompressionErrorText(err error) string {
	if err == nil {
		return ""
	}
	msg := redaction.RedactSecrets(err.Error())
	msg = strings.NewReplacer("`", "'", "*", "'", "#", "＃").Replace(msg)
	return strings.Join(strings.Fields(msg), " ")
}

func (e *ProviderBackedContextEngine) refreshLocked() {
	budgetStatus := e.budget.Status()
	e.status.Model = budgetStatus.Model
	e.status.ContextLength = budgetStatus.ContextLength
	e.status.ThresholdTokens = budgetStatus.ThresholdTokens
	e.status.ThresholdPercent = budgetStatus.ThresholdPercent
	if e.status.ContextLength > 0 {
		usage := float64(e.status.LastPromptTokens) / float64(e.status.ContextLength) * 100
		e.status.UsagePercent = roundPercent(usage)
		if e.status.UsagePercent > 100 {
			e.status.UsagePercent = 100
		}
	} else {
		e.status.UsagePercent = 0
	}
	e.status.Budget = classifyContextBudget(e.status.LastPromptTokens, e.status.ThresholdTokens, e.status.ContextLength)
	e.status.Compression.Enabled = true
	e.status.Compression.ShouldCompress = budgetStatus.State == "ready" && e.status.ThresholdTokens > 0 && e.status.LastPromptTokens >= e.status.ThresholdTokens
	if e.status.Boundary.Status == "" {
		e.status.Boundary = ContextBoundaryStatus{Status: ContextBoundaryStatusMissing, Message: "no compression boundary recorded"}
	}
	e.status.Tools.StatusTool = ContextStatusToolName
}
