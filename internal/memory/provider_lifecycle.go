package memory

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory/lifecycle"
)

const memoryProviderUnavailableCode = "memory_provider_unavailable"

type LifecycleProvider = lifecycle.LifecycleProvider
type MemoryProviderSession = lifecycle.MemoryProviderSession
type MemoryPrefetchRequest = lifecycle.MemoryPrefetchRequest
type MemoryTurn = lifecycle.MemoryTurn
type MemoryProviderMessage = lifecycle.MemoryProviderMessage
type MemoryWriteEvent = lifecycle.MemoryWriteEvent
type MemoryDelegationEvent = lifecycle.MemoryDelegationEvent
type MemoryProviderEvidence = lifecycle.MemoryProviderEvidence
type MemoryProviderLifecycle = lifecycle.MemoryProviderLifecycle

func NewMemoryProviderLifecycle() *MemoryProviderLifecycle {
	return lifecycle.NewMemoryProviderLifecycle()
}

// KernelPreCompressAdapter wraps MemoryProviderLifecycle to satisfy the
// kernel.MemoryPreCompressor interface which takes []llm.Message and returns
// (string, error). Wire it as kernel.Config.MemoryLifecycle:
//
//	cfg.MemoryLifecycle = memory.NewKernelPreCompressAdapter(lc)
type KernelPreCompressAdapter struct {
	lc *MemoryProviderLifecycle
}

// NewKernelPreCompressAdapter wraps lc so it satisfies the kernel.MemoryPreCompressor
// interface. The adapter converts llm.Message to lifecycle.MemoryProviderMessage and
// drops the evidence slice (errors are non-fatal in compression context).
func NewKernelPreCompressAdapter(lc *MemoryProviderLifecycle) *KernelPreCompressAdapter {
	return &KernelPreCompressAdapter{lc: lc}
}

// PreCompress converts llm.Message slice to lifecycle messages and delegates
// to MemoryProviderLifecycle.PreCompress. Evidence is discarded; the hint
// string is returned (may be empty).
func (a *KernelPreCompressAdapter) PreCompress(ctx context.Context, messages []llm.Message) (string, error) {
	if a.lc == nil {
		return "", nil
	}
	msgs := make([]lifecycle.MemoryProviderMessage, len(messages))
	for i, m := range messages {
		msgs[i] = lifecycle.MemoryProviderMessage{Role: m.Role, Content: m.Content}
	}
	hint, _ := a.lc.PreCompress(ctx, msgs)
	return hint, nil
}

// KernelPrefetchAdapter wraps MemoryProviderLifecycle to satisfy the
// kernel.MemoryPrefetcher interface. Wire it as kernel.Config.MemoryPrefetch:
//
//	cfg.MemoryPrefetch = memory.NewKernelPrefetchAdapter(lc)
type KernelPrefetchAdapter struct {
	lc *MemoryProviderLifecycle
}

// NewKernelPrefetchAdapter wraps lc to expose the kernel.MemoryPrefetcher
// interface, mapping (query, sessionID) to lifecycle.MemoryPrefetchRequest.
func NewKernelPrefetchAdapter(lc *MemoryProviderLifecycle) *KernelPrefetchAdapter {
	return &KernelPrefetchAdapter{lc: lc}
}

// Prefetch calls lc.Prefetch with the supplied query and sessionID and returns
// the merged context hint. Evidence is discarded; the turn proceeds on error.
func (a *KernelPrefetchAdapter) Prefetch(ctx context.Context, query, sessionID string) (string, error) {
	if a.lc == nil {
		return "", nil
	}
	hint, _ := a.lc.Prefetch(ctx, lifecycle.MemoryPrefetchRequest{
		Query:     query,
		SessionID: sessionID,
	})
	return hint, nil
}

// KernelSyncTurnAdapter wraps MemoryProviderLifecycle to satisfy the
// kernel.MemorySyncTurnWriter interface. Wire it as kernel.Config.MemorySyncTurn:
//
//	cfg.MemorySyncTurn = memory.NewKernelSyncTurnAdapter(lc)
type KernelSyncTurnAdapter struct {
	lc *MemoryProviderLifecycle
}

// NewKernelSyncTurnAdapter wraps lc to expose the kernel.MemorySyncTurnWriter
// interface, mapping flat strings to lifecycle.MemoryTurn.
func NewKernelSyncTurnAdapter(lc *MemoryProviderLifecycle) *KernelSyncTurnAdapter {
	return &KernelSyncTurnAdapter{lc: lc}
}

// SyncTurn delegates to lc.SyncTurn. Evidence is discarded; errors are
// non-fatal at the kernel level (the turn already completed).
func (a *KernelSyncTurnAdapter) SyncTurn(ctx context.Context, userText, assistantText, platform, model, provider, sessionID string) error {
	if a.lc == nil {
		return nil
	}
	a.lc.SyncTurn(ctx, lifecycle.MemoryTurn{
		SessionID: sessionID,
		User:      userText,
		Assistant: assistantText,
		Platform:  platform,
		Model:     model,
		Provider:  provider,
	})
	return nil
}
