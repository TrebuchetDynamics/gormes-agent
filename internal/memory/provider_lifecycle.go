package memory

import "github.com/TrebuchetDynamics/gormes-agent/internal/memory/lifecycle"

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
