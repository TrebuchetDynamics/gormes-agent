package local

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/transcript"
)

type residentSessionSwitcher struct {
	rootCtx context.Context
	resume  func(string, []llm.Message) error
}

func newResidentSessionSwitcher(rootCtx context.Context, resume func(string, []llm.Message) error) residentSessionSwitcher {
	return residentSessionSwitcher{rootCtx: rootCtx, resume: resume}
}

func (s residentSessionSwitcher) requireResume() error {
	if s.resume == nil {
		return fmt.Errorf("kernel resume unavailable")
	}
	return nil
}

func (s residentSessionSwitcher) context(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	if s.rootCtx != nil {
		return s.rootCtx
	}
	return context.Background()
}

func (s residentSessionSwitcher) switchWithTranscriptDB(ctx context.Context, db *sql.DB, sessionID string, history []llm.Message) ([]llm.Message, error) {
	if err := s.requireResume(); err != nil {
		return nil, err
	}
	ctx = s.context(ctx)
	resolvedHistory := cloneHermesMessages(history)
	if len(resolvedHistory) == 0 {
		messages, err := transcript.LoadMessages(ctx, db, sessionID)
		if err != nil {
			return nil, err
		}
		resolvedHistory = hermesMessagesFromTranscript(messages)
	}
	if err := s.resume(sessionID, resolvedHistory); err != nil {
		return nil, err
	}
	return resolvedHistory, nil
}

func hermesMessagesFromTranscript(messages []transcript.Message) []llm.Message {
	if len(messages) == 0 {
		return nil
	}
	out := make([]llm.Message, 0, len(messages))
	for _, msg := range messages {
		out = append(out, llm.Message{Role: msg.Role, Content: msg.Content})
	}
	return out
}

func cloneHermesMessages(messages []llm.Message) []llm.Message {
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
