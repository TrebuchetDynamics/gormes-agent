package kernel

import "context"

// RecallProvider is the thin bridge the kernel uses to ask for memory
// context before sending a turn to the LLM. Implemented by
// internal/memory's Provider (or any other future source). Must be fast:
// the kernel applies a ~100ms ctx deadline around the call; if it trips,
// the turn proceeds without memory injection.
//
// Declared IN internal/kernel so the kernel stays ignorant of any
// persistence or transport details — memory imports kernel to implement,
// not the other way around. T12 build-isolation test is thereby
// maintained: the kernel's dep graph never contains internal/memory.
type RecallProvider interface {
	GetContext(ctx context.Context, params RecallParams) string
}

// RecallParams is what the kernel knows about the current turn at the
// moment GetContext is invoked.
type RecallParams struct {
	UserMessage string // the raw turn text
	ChatKey     string // "<platform>:<chat_id>" scope (e.g. "telegram:42")
	SessionID   string // the server-assigned session_id, for diagnostic use
}
