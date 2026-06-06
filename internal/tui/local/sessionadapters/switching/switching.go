package switching

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/transcript"
)

type ResumeFunc func(string, []llm.Message) error

type ResidentSessionSwitcher struct {
	rootCtx context.Context
	resume  ResumeFunc
}

func NewResidentSessionSwitcher(rootCtx context.Context, resume ResumeFunc) ResidentSessionSwitcher {
	return ResidentSessionSwitcher{rootCtx: rootCtx, resume: resume}
}

func (s ResidentSessionSwitcher) RequireResume() error {
	if s.resume == nil {
		return fmt.Errorf("kernel resume unavailable")
	}
	return nil
}

func (s ResidentSessionSwitcher) Context(ctx context.Context) context.Context {
	return ContextWithRootFallback(s.rootCtx, ctx)
}

func ContextWithRootFallback(rootCtx, ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	if rootCtx != nil {
		return rootCtx
	}
	return context.Background()
}

func (s ResidentSessionSwitcher) SwitchWithTranscriptDB(ctx context.Context, db *sql.DB, sessionID string, history []llm.Message) ([]llm.Message, error) {
	if err := s.RequireResume(); err != nil {
		return nil, err
	}
	ctx = s.Context(ctx)
	resolvedHistory := CloneHermesMessages(history)
	if len(resolvedHistory) == 0 {
		messages, err := transcript.LoadMessages(ctx, db, sessionID)
		if err != nil {
			return nil, err
		}
		resolvedHistory = HermesMessagesFromTranscript(messages)
	}
	if err := s.resume(sessionID, resolvedHistory); err != nil {
		return nil, err
	}
	return resolvedHistory, nil
}

func HermesMessagesFromTranscript(messages []transcript.Message) []llm.Message {
	if len(messages) == 0 {
		return nil
	}
	out := make([]llm.Message, 0, len(messages))
	for _, msg := range messages {
		out = append(out, llm.Message{Role: msg.Role, Content: msg.Content})
	}
	return out
}

func CloneHermesMessages(messages []llm.Message) []llm.Message {
	if len(messages) == 0 {
		return nil
	}
	out := make([]llm.Message, len(messages))
	for i, msg := range messages {
		out[i] = msg
		out[i].ContentParts = append([]llm.MessageContentPart(nil), msg.ContentParts...)
	}
	return out
}
