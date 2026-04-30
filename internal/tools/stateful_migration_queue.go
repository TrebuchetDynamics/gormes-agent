package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const (
	ToolStateContractRegistered = "tool_state_contract_registered"
	ToolStateContractMissing    = "tool_state_contract_missing"
	ToolStatePathAllowed        = "tool_path_allowed"
	ToolPathDenied              = "tool_path_denied"
	ToolConcurrencyBlocked      = "tool_concurrency_blocked"
	ToolStateRuntimeNotPorted   = "tool_state_runtime_not_ported"
)

type ToolStateDomain string

const (
	ToolStateDomainReadOnly    ToolStateDomain = "read_only"
	ToolStateDomainFile        ToolStateDomain = "file"
	ToolStateDomainSession     ToolStateDomain = "session"
	ToolStateDomainCheckpoint  ToolStateDomain = "checkpoint"
	ToolStateDomainProcess     ToolStateDomain = "process"
)

type ToolRootPolicy string

const (
	ToolRootPolicyInjectedXDG ToolRootPolicy = "injected_xdg_roots"
	ToolRootPolicyGormesData  ToolRootPolicy = "gormes_data_root"
)

type ToolRollbackPolicy string

const (
	ToolRollbackPolicyNone       ToolRollbackPolicy = "none"
	ToolRollbackPolicyAuditLog   ToolRollbackPolicy = "audit_log"
	ToolRollbackPolicyCheckpoint ToolRollbackPolicy = "checkpoint"
)

type ToolConcurrencyPolicy string

const (
	ToolConcurrencyConcurrentReads  ToolConcurrencyPolicy = "concurrent_reads"
	ToolConcurrencySerializedWrites ToolConcurrencyPolicy = "serialized_writes"
)

// StatefulToolPlan freezes the state contract a future write-capable tool must
// satisfy before the executable port is exposed to the native Gormes loop.
type StatefulToolPlan struct {
	Name              string
	Domain            ToolStateDomain
	RootPolicy        ToolRootPolicy
	RollbackPolicy    ToolRollbackPolicy
	ConcurrencyPolicy ToolConcurrencyPolicy
	OwnerRow          string
}

type StatefulToolEvidence struct {
	Code    string
	Tool    string
	Message string
}

type StatefulToolQueueOptions struct {
	MutationRoots []string
}

type StatefulToolMigrationQueue struct {
	mu       sync.RWMutex
	plans    map[string]StatefulToolPlan
	roots    []string
	writeMu  sync.Mutex
}

func NewStatefulToolMigrationQueue(opts StatefulToolQueueOptions) *StatefulToolMigrationQueue {
	roots := make([]string, 0, len(opts.MutationRoots))
	for _, root := range opts.MutationRoots {
		if normalized, ok := normalizeStatefulMutationRoot(root); ok {
			roots = append(roots, normalized)
		}
	}
	sort.Strings(roots)
	return &StatefulToolMigrationQueue{
		plans: make(map[string]StatefulToolPlan),
		roots: roots,
	}
}

func (q *StatefulToolMigrationQueue) Register(plan StatefulToolPlan) StatefulToolEvidence {
	if missing := missingStatefulPlanField(plan); missing != "" {
		return StatefulToolEvidence{Code: ToolStateContractMissing, Tool: plan.Name, Message: "missing stateful tool contract field: " + missing}
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.plans[plan.Name] = plan
	return StatefulToolEvidence{Code: ToolStateContractRegistered, Tool: plan.Name, Message: "stateful tool contract registered"}
}

func (q *StatefulToolMigrationQueue) Plan(name string) (StatefulToolPlan, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	plan, ok := q.plans[name]
	return plan, ok
}

func (q *StatefulToolMigrationQueue) Plans() []StatefulToolPlan {
	q.mu.RLock()
	defer q.mu.RUnlock()
	out := make([]StatefulToolPlan, 0, len(q.plans))
	for _, plan := range q.plans {
		out = append(out, plan)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (q *StatefulToolMigrationQueue) AuthorizePath(toolName, candidate string) StatefulToolEvidence {
	plan, ok := q.Plan(toolName)
	if !ok {
		return StatefulToolEvidence{Code: ToolStateContractMissing, Tool: toolName, Message: "tool has no stateful migration contract"}
	}
	if plan.RootPolicy != ToolRootPolicyInjectedXDG && plan.RootPolicy != ToolRootPolicyGormesData {
		return StatefulToolEvidence{Code: ToolPathDenied, Tool: toolName, Message: "tool has no injected root policy"}
	}
	candidateAbs, err := filepath.Abs(candidate)
	if err != nil {
		return StatefulToolEvidence{Code: ToolPathDenied, Tool: toolName, Message: "path cannot be normalized"}
	}
	candidateAbs = filepath.Clean(candidateAbs)
	for _, root := range q.roots {
		rel, err := filepath.Rel(root, candidateAbs)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return StatefulToolEvidence{Code: ToolStatePathAllowed, Tool: toolName, Message: "path is inside injected mutation root"}
		}
	}
	return StatefulToolEvidence{Code: ToolPathDenied, Tool: toolName, Message: "path is outside injected mutation roots"}
}

func (q *StatefulToolMigrationQueue) Run(ctx context.Context, toolName string, fn func(context.Context) error) error {
	plan, ok := q.Plan(toolName)
	if !ok {
		return fmt.Errorf("%s: %s", ToolStateContractMissing, toolName)
	}
	if fn == nil {
		return nil
	}
	if plan.ConcurrencyPolicy == ToolConcurrencySerializedWrites {
		q.writeMu.Lock()
		defer q.writeMu.Unlock()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return fn(ctx)
}

func (q *StatefulToolMigrationQueue) RuntimePortEvidence(toolName string) StatefulToolEvidence {
	if _, ok := q.Plan(toolName); !ok {
		return StatefulToolEvidence{Code: ToolStateContractMissing, Tool: toolName, Message: "tool has no stateful migration contract"}
	}
	return StatefulToolEvidence{Code: ToolStateRuntimeNotPorted, Tool: toolName, Message: "stateful runtime execution remains unported in this queue contract slice"}
}

func missingStatefulPlanField(plan StatefulToolPlan) string {
	if strings.TrimSpace(plan.Name) == "" {
		return "name"
	}
	if plan.Domain == "" {
		return "domain"
	}
	if plan.RootPolicy == "" {
		return "root_policy"
	}
	if plan.RollbackPolicy == "" {
		return "rollback_policy"
	}
	if plan.Domain != ToolStateDomainReadOnly && plan.RollbackPolicy == ToolRollbackPolicyNone {
		return "rollback_policy"
	}
	if plan.ConcurrencyPolicy == "" {
		return "concurrency_policy"
	}
	if strings.TrimSpace(plan.OwnerRow) == "" {
		return "owner_row"
	}
	return ""
}

func normalizeStatefulMutationRoot(root string) (string, bool) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", false
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	return filepath.Clean(abs), true
}
