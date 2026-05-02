package gateway

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type agentRuntimeSnapshot struct {
	Tools      *tools.Registry
	Skills     kernel.SkillProvider
	ToolSafety kernel.ToolSafetyPolicy
}

type agentSkillProvider struct {
	runtime *skills.Runtime
	opts    skills.RuntimeOptions
}

func (m *Manager) agentRuntimeSnapshot(route agentRuntimeRoute) agentRuntimeSnapshot {
	if !route.Enabled {
		return agentRuntimeSnapshot{}
	}
	policy := route.Decision.Tools
	var filtered *tools.Registry
	if m.cfg.ToolRegistry != nil {
		filtered = m.cfg.ToolRegistry.FilterPolicy(policy.Allow, policy.Deny)
	}
	snapshot := agentRuntimeSnapshot{
		Tools: filtered,
		ToolSafety: kernel.NewAgentToolSafetyPolicy(kernel.AgentToolSafetyOptions{
			AgentID: route.Decision.AgentID,
			Allow:   policy.Allow,
			Deny:    policy.Deny,
		}),
	}
	if m.cfg.SkillRuntime != nil {
		snapshot.Skills = agentSkillProvider{
			runtime: m.cfg.SkillRuntime,
			opts: skills.RuntimeOptions{
				AllowedSkillNames: policyNameMap(route.Decision.Skills),
				AvailableTools:    registryNames(filtered),
			},
		}
	}
	return snapshot
}

func (m *Manager) agentRuntimeForRoute(ctx context.Context, route agentRuntimeRoute, snapshot agentRuntimeSnapshot) (KernelSubmitter, error) {
	if m.cfg.AgentRuntimeFactory == nil {
		return nil, nil
	}
	key := strings.TrimSpace(route.SessionKey)
	if key == "" {
		key = route.Decision.SessionKey()
	}
	if key == "" {
		return nil, errors.New("agent runtime key is empty")
	}

	m.agentRuntimeMu.Lock()
	if runtime, ok := m.agentRuntimes[key]; ok {
		m.agentRuntimeMu.Unlock()
		return runtime, nil
	}
	m.agentRuntimeMu.Unlock()

	authHome := strings.TrimSpace(route.Decision.AgentDir)
	authStore := ""
	if authHome != "" {
		authStore = filepath.Join(authHome, "auth.json")
	}
	req := AgentRuntimeRequest{
		AgentID:     route.Decision.AgentID,
		Name:        route.Decision.Name,
		SessionKey:  key,
		Workspace:   route.Decision.Workspace,
		AgentDir:    route.Decision.AgentDir,
		AuthHome:    authHome,
		AuthStore:   authStore,
		Model:       route.Decision.Model,
		BindingTier: string(route.Decision.BindingTier),
		ToolPolicy:  route.Decision.Tools,
		SkillNames:  append([]string(nil), route.Decision.Skills...),
		Tools:       snapshot.Tools,
		Skills:      snapshot.Skills,
		ToolSafety:  snapshot.ToolSafety,
	}
	runtime, err := m.cfg.AgentRuntimeFactory(ctx, req)
	if err != nil {
		return nil, err
	}
	if runtime == nil {
		return nil, errors.New("agent runtime factory returned nil runtime")
	}

	m.agentRuntimeMu.Lock()
	if existing, ok := m.agentRuntimes[key]; ok {
		m.agentRuntimeMu.Unlock()
		return existing, nil
	}
	m.agentRuntimes[key] = runtime
	m.agentRuntimeMu.Unlock()

	if render := runtime.Render(); render != nil && m.agentRuntimeRender != nil {
		go m.forwardAgentRuntimeFrames(ctx, render)
	}
	return runtime, nil
}

func (m *Manager) forwardAgentRuntimeFrames(ctx context.Context, render <-chan kernel.RenderFrame) {
	for {
		select {
		case <-ctx.Done():
			return
		case frame, ok := <-render:
			if !ok {
				return
			}
			select {
			case m.agentRuntimeRender <- frame:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (m *Manager) setPinnedTurnKernel(k KernelSubmitter) {
	m.turnMu.Lock()
	defer m.turnMu.Unlock()
	m.turnKernel = k
}

func (m *Manager) activeTurnKernel() KernelSubmitter {
	m.turnMu.Lock()
	defer m.turnMu.Unlock()
	if m.turnKernel != nil {
		return m.turnKernel
	}
	return m.kernel
}

func (p agentSkillProvider) BuildSkillBlock(ctx context.Context, userMessage string) (string, []string, error) {
	if p.runtime == nil {
		return "", nil, nil
	}
	block, names, _, err := p.runtime.BuildSkillBlockWithOptions(ctx, userMessage, p.opts)
	return block, names, err
}

func registryNames(reg *tools.Registry) []string {
	if reg == nil {
		return nil
	}
	descs := reg.Descriptors()
	out := make([]string, 0, len(descs))
	for _, desc := range descs {
		out = append(out, desc.Name)
	}
	return out
}

func policyNameMap(values []string) map[string]bool {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			out[value] = true
		}
	}
	return out
}
