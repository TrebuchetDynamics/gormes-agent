package gormescli

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/core/subagent"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/audit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type DelegationToolOptions struct {
	ParentCtx   context.Context
	Config      config.Config
	Registry    *tools.Registry
	ChildClient llm.Client
	ChildModel  string
}

// RegisterDelegationTool registers the delegate tool when delegation is enabled.
func RegisterDelegationTool(opts DelegationToolOptions) {
	cfg := opts.Config
	reg := opts.Registry
	if reg == nil || !cfg.Delegation.Enabled {
		return
	}
	var drafter subagent.CandidateDrafter
	skillsRoot := cfg.SkillsRoot()
	if skillsRoot != "" {
		drafter = skillsCandidateDrafter{store: skills.NewStore(skillsRoot, 0)}
	}
	managerOpts := subagent.ManagerOpts{
		ParentCtx:            opts.ParentCtx,
		ParentID:             "root",
		Depth:                0,
		Registry:             subagent.NewRegistry(),
		ToolExecutor:         tools.NewInProcessToolExecutor(reg),
		MaxDepth:             cfg.Delegation.MaxDepth,
		DefaultMaxIterations: cfg.Delegation.DefaultMaxIterations,
		DefaultMaxConcurrent: cfg.Delegation.MaxConcurrentChildren,
		DefaultTimeout:       cfg.Delegation.DefaultTimeout,
		RunLogPath:           cfg.Delegation.ResolvedRunLogPath(),
		ToolAudit:            audit.NewJSONLWriter(config.ToolAuditLogPath()),
	}
	if opts.ChildClient != nil {
		descs := registryDescriptors(reg)
		managerOpts.NewRunner = func() subagent.Runner {
			return subagent.NewHermesRunner(opts.ChildClient, opts.ChildModel, descs)
		}
	}
	reg.MustRegister(subagent.NewDelegateTool(subagent.NewManager(managerOpts), drafter))
}

func registryDescriptors(reg *tools.Registry) []llm.ToolDescriptor {
	descs := reg.Descriptors()
	out := make([]llm.ToolDescriptor, len(descs))
	for i, d := range descs {
		out[i] = llm.ToolDescriptor{Name: d.Name, Description: d.Description, Schema: d.Schema}
	}
	return out
}

type skillsCandidateDrafter struct {
	store *skills.Store
}

func (d skillsCandidateDrafter) DraftCandidate(_ context.Context, req subagent.CandidateDraftRequest) (string, error) {
	meta, err := d.store.DraftCandidate(skills.CandidateDraft{
		Slug:            req.Slug,
		Goal:            req.Goal,
		Summary:         req.Summary,
		SourceRunID:     req.SourceRunID,
		ParentSessionID: req.ParentSessionID,
		ChildAgentID:    req.ChildAgentID,
		ToolNames:       append([]string(nil), req.ToolNames...),
	})
	if err != nil {
		return "", err
	}
	return meta.CandidateID, nil
}
