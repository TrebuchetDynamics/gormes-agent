package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/stateful"

const (
	ToolStateContractRegistered = stateful.ToolStateContractRegistered
	ToolStateContractMissing    = stateful.ToolStateContractMissing
	ToolStatePathAllowed        = stateful.ToolStatePathAllowed
	ToolPathDenied              = stateful.ToolPathDenied
	ToolConcurrencyBlocked      = stateful.ToolConcurrencyBlocked
	ToolStateRuntimeNotPorted   = stateful.ToolStateRuntimeNotPorted
)

const (
	ToolStateDomainReadOnly   = stateful.ToolStateDomainReadOnly
	ToolStateDomainFile       = stateful.ToolStateDomainFile
	ToolStateDomainSession    = stateful.ToolStateDomainSession
	ToolStateDomainCheckpoint = stateful.ToolStateDomainCheckpoint
	ToolStateDomainProcess    = stateful.ToolStateDomainProcess
)

const (
	ToolRootPolicyInjectedXDG = stateful.ToolRootPolicyInjectedXDG
	ToolRootPolicyGormesData  = stateful.ToolRootPolicyGormesData
)

const (
	ToolRollbackPolicyNone       = stateful.ToolRollbackPolicyNone
	ToolRollbackPolicyAuditLog   = stateful.ToolRollbackPolicyAuditLog
	ToolRollbackPolicyCheckpoint = stateful.ToolRollbackPolicyCheckpoint
)

const (
	ToolConcurrencyConcurrentReads  = stateful.ToolConcurrencyConcurrentReads
	ToolConcurrencySerializedWrites = stateful.ToolConcurrencySerializedWrites
)

type ToolStateDomain = stateful.ToolStateDomain
type ToolRootPolicy = stateful.ToolRootPolicy
type ToolRollbackPolicy = stateful.ToolRollbackPolicy
type ToolConcurrencyPolicy = stateful.ToolConcurrencyPolicy
type StatefulToolPlan = stateful.StatefulToolPlan
type StatefulToolEvidence = stateful.StatefulToolEvidence
type StatefulToolQueueOptions = stateful.StatefulToolQueueOptions
type StatefulToolMigrationQueue = stateful.StatefulToolMigrationQueue

func NewStatefulToolMigrationQueue(opts StatefulToolQueueOptions) *StatefulToolMigrationQueue {
	return stateful.NewStatefulToolMigrationQueue(opts)
}
