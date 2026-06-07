package testfixtures

import (
	"context"
	"encoding/json"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel/recall"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// ContextEngine is a controllable llm.ContextEngine test double for kernel tests.
type ContextEngine struct {
	BoundaryCalls []llm.CompressionBoundary
	ModelUpdates  []llm.ContextModelContext
	CompressErr   error
	CompressCalls int
}

func (e *ContextEngine) Name() string                          { return "fake" }
func (e *ContextEngine) ToolDescriptors() []llm.ToolDescriptor { return nil }
func (e *ContextEngine) UpdateModelContext(update llm.ContextModelContext) {
	e.ModelUpdates = append(e.ModelUpdates, update)
}
func (e *ContextEngine) OnSessionStart(_ context.Context, _ string, _ llm.ContextSessionMeta) error {
	return nil
}
func (e *ContextEngine) OnSessionEnd(_ context.Context, _ string, _ []llm.Message) error {
	return nil
}
func (e *ContextEngine) OnSessionReset()                            {}
func (e *ContextEngine) UpdateFromResponse(llm.ContextUsage)        {}
func (e *ContextEngine) Status() llm.ContextStatus                  { return llm.ContextStatus{} }
func (e *ContextEngine) ShouldCompress(int) bool                    { return false }
func (e *ContextEngine) ShouldCompressPreflight([]llm.Message) bool { return false }
func (e *ContextEngine) HasContentToCompress([]llm.Message) bool    { return false }
func (e *ContextEngine) GetHistoryTokenEstimate() int               { return 0 }
func (e *ContextEngine) HandleToolCall(_ context.Context, _ string, _ json.RawMessage, _ llm.ContextToolCallOptions) (json.RawMessage, error) {
	return nil, nil
}
func (e *ContextEngine) Compress(_ context.Context, msgs []llm.Message, _ llm.CompressionRequest) ([]llm.Message, llm.CompressionReport, error) {
	e.CompressCalls++
	if e.CompressErr != nil {
		return msgs, llm.CompressionReport{}, e.CompressErr
	}
	return msgs, llm.CompressionReport{}, nil
}
func (e *ContextEngine) OnCompressionBoundary(_ context.Context, b llm.CompressionBoundary) error {
	e.BoundaryCalls = append(e.BoundaryCalls, b)
	return nil
}

// SkillProvider records skill block requests for kernel tests.
type SkillProvider struct {
	Block string
	Names []string
	Err   error
	Calls int
	Last  string
}

func (s *SkillProvider) BuildSkillBlock(_ context.Context, userMessage string) (string, []string, error) {
	s.Calls++
	s.Last = userMessage
	return s.Block, append([]string(nil), s.Names...), s.Err
}

// SkillUsageRecorder records selected skill names for kernel tests.
type SkillUsageRecorder struct {
	Calls int
	Got   [][]string
	Err   error
}

func (s *SkillUsageRecorder) RecordSkillUsage(_ context.Context, skillNames []string) error {
	s.Calls++
	s.Got = append(s.Got, append([]string(nil), skillNames...))
	return s.Err
}

// RecallProvider implements recall.Provider for kernel tests.
type RecallProvider struct {
	ReturnContent string
	Delay         time.Duration
	Calls         int
	LastInput     recall.Params
}

func (m *RecallProvider) GetContext(ctx context.Context, p recall.Params) string {
	m.Calls++
	m.LastInput = p
	if m.Delay > 0 {
		select {
		case <-time.After(m.Delay):
		case <-ctx.Done():
			return "" // honor the kernel's deadline cutoff
		}
	}
	return m.ReturnContent
}
