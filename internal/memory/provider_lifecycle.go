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
