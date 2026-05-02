package gateway

import (
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

// AgentRoutingConfig enables OpenClaw-style agent/workspace bindings in the
// gateway manager without changing legacy single-agent behavior by default.
type AgentRoutingConfig struct {
	Enabled  bool
	Agents   config.AgentsCfg
	Bindings []config.AgentBindingCfg
}

type agentRuntimeRoute struct {
	Enabled    bool
	Decision   AgentRouteDecision
	SessionKey string
	ProfileDir string
	CWD        string
	MemoryDir  string
}

func (m *Manager) agentRouteForInbound(ev InboundEvent) agentRuntimeRoute {
	if !m.agentRoutingEnabled {
		return agentRuntimeRoute{SessionKey: ev.ChatKey()}
	}
	decision := m.agentRouter.Resolve(agentRouteRequestFromInbound(ev))
	sessionKey := decision.SessionKey()
	workspace := strings.TrimSpace(decision.Workspace)
	agentDir := strings.TrimSpace(decision.AgentDir)
	route := agentRuntimeRoute{
		Enabled:    true,
		Decision:   decision,
		SessionKey: sessionKey,
		ProfileDir: workspace,
		CWD:        workspace,
	}
	if agentDir != "" {
		route.MemoryDir = filepath.Join(agentDir, "memory")
	}
	return route
}

func (m *Manager) sessionKeyForInbound(ev InboundEvent) string {
	route := m.agentRouteForInbound(ev)
	if route.SessionKey != "" {
		return route.SessionKey
	}
	return ev.ChatKey()
}

func (m *Manager) liveTurnPromptSeamsForAgent(route agentRuntimeRoute) liveTurnPromptSeams {
	seams := m.liveTurnPromptSeams
	if !route.Enabled {
		return seams
	}
	if dir := strings.TrimSpace(route.ProfileDir); dir != "" {
		seams.ProfileDir = func() string { return dir }
	}
	if cwd := strings.TrimSpace(route.CWD); cwd != "" {
		seams.CWD = func() string { return cwd }
	}
	if memDir := strings.TrimSpace(route.MemoryDir); memDir != "" {
		seams.MemoryDir = func() string { return memDir }
	}
	return seams
}

func (m *Manager) prepareKernelForAgentSession(sessionKey string) error {
	if !m.agentRoutingEnabled {
		return nil
	}
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return nil
	}
	m.turnMu.Lock()
	previous := m.kernelSessionKey
	if previous == sessionKey {
		m.turnMu.Unlock()
		return nil
	}
	if previous == "" {
		m.kernelSessionKey = sessionKey
		m.turnMu.Unlock()
		return nil
	}
	m.turnMu.Unlock()

	if m.kernel != nil {
		if err := m.kernel.ResetSession(); err != nil {
			return err
		}
	}

	m.turnMu.Lock()
	m.kernelSessionKey = sessionKey
	m.turnMu.Unlock()
	return nil
}

func (r agentRuntimeRoute) SessionContext() AgentContext {
	if !r.Enabled {
		return AgentContext{}
	}
	return AgentContext{
		ID:          r.Decision.AgentID,
		Name:        r.Decision.Name,
		Workspace:   r.Decision.Workspace,
		AgentDir:    r.Decision.AgentDir,
		BindingTier: string(r.Decision.BindingTier),
	}
}

func agentRouteRequestFromInbound(ev InboundEvent) AgentRouteRequest {
	peerKind := strings.ToLower(strings.TrimSpace(ev.ChatType))
	if ev.IsDirectMessage() {
		peerKind = "direct"
	}
	if peerKind == "" {
		peerKind = "group"
	}
	peerID := strings.TrimSpace(ev.ChatID)
	parentPeerID := strings.TrimSpace(ev.ParentChatID)
	parentPeerKind := ""
	if strings.TrimSpace(ev.ThreadID) != "" {
		peerID = strings.TrimSpace(ev.ThreadID)
		if parentPeerID == "" {
			parentPeerID = strings.TrimSpace(ev.ChatID)
		}
		parentPeerKind = "parent"
	}
	return AgentRouteRequest{
		Channel:        ev.Platform,
		AccountID:      ev.AccountID,
		PeerKind:       peerKind,
		PeerID:         peerID,
		ParentPeerKind: parentPeerKind,
		ParentPeerID:   parentPeerID,
		GuildID:        ev.GuildID,
		TeamID:         ev.TeamID,
		Roles:          ev.Roles,
		MainKey:        ev.ChatKey(),
	}
}
