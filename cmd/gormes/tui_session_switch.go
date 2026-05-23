package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/transcript"
)

type tuiResidentSessionSwitcher struct {
	rootCtx context.Context
	resume  func(string, []hermes.Message) error
}

func newTUIResidentSessionSwitcher(rootCtx context.Context, resume func(string, []hermes.Message) error) tuiResidentSessionSwitcher {
	return tuiResidentSessionSwitcher{rootCtx: rootCtx, resume: resume}
}

func (s tuiResidentSessionSwitcher) requireResume() error {
	if s.resume == nil {
		return fmt.Errorf("kernel resume unavailable")
	}
	return nil
}

func (s tuiResidentSessionSwitcher) context(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	if s.rootCtx != nil {
		return s.rootCtx
	}
	return context.Background()
}

func (s tuiResidentSessionSwitcher) switchWithTranscriptDB(ctx context.Context, db *sql.DB, sessionID string, history []hermes.Message) ([]hermes.Message, error) {
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

func hermesMessagesFromTranscript(messages []transcript.Message) []hermes.Message {
	if len(messages) == 0 {
		return nil
	}
	out := make([]hermes.Message, 0, len(messages))
	for _, msg := range messages {
		out = append(out, hermes.Message{Role: msg.Role, Content: msg.Content})
	}
	return out
}

func cloneHermesMessages(messages []hermes.Message) []hermes.Message {
	if len(messages) == 0 {
		return nil
	}
	out := make([]hermes.Message, len(messages))
	for i, msg := range messages {
		out[i] = msg
		out[i].ContentParts = append([]hermes.MessageContentPart(nil), msg.ContentParts...)
	}
	return out
}
