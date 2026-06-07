package gateway

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/configreload"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
)

var ErrConfigReloadUnavailable = configreload.ErrUnavailable

func (m *Manager) Reload(ctx context.Context) error {
	reload := m.configReloader()
	if reload == nil {
		err := ErrConfigReloadUnavailable
		m.recordConfigReloadFailure(ctx, err)
		return err
	}
	next, err := reload(ctx)
	if err != nil {
		m.recordConfigReloadFailure(ctx, err)
		return err
	}
	m.applyReloadableConfig(ctx, next)
	return nil
}

func (m *Manager) configReloader() func(context.Context) (ManagerConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cfg.ReloadConfig
}

func (m *Manager) applyReloadableConfig(ctx context.Context, next ManagerConfig) {
	m.mu.Lock()
	m.cfg.AllowedChats = configreload.CloneStringMap(next.AllowedChats)
	m.cfg.AllowedUsers = configreload.CloneNestedBoolMap(next.AllowedUsers)
	m.cfg.AllowedChatWhitelists = next.AllowedChatWhitelists
	m.cfg.AllowDiscovery = configreload.CloneBoolMap(next.AllowDiscovery)
	m.cfg.CoalesceMs = next.CoalesceMs
	if m.cfg.CoalesceMs <= 0 {
		m.cfg.CoalesceMs = 1000
	}
	m.cfg.FreshFinalAfter = next.FreshFinalAfter
	m.cfg.ToolProgressMode = next.ToolProgressMode
	m.cfg.ToolProgressCommandEnabled = next.ToolProgressCommandEnabled
	m.cfg.BusyInputMode = textvalue.FirstNonEmptyTrimmed(next.BusyInputMode, "interrupt")
	m.cfg.ReplyMode = next.ReplyMode
	m.cfg.ToolProgressModes = configreload.CloneStringMap(next.ToolProgressModes)
	if next.PersistToolProgressMode != nil {
		m.cfg.PersistToolProgressMode = next.PersistToolProgressMode
	}
	if next.LiveTurnActiveModel != nil {
		m.cfg.LiveTurnActiveModel = next.LiveTurnActiveModel
		m.liveTurnPromptSeams.ActiveModel = next.LiveTurnActiveModel
	}
	if next.LiveTurnActiveProvider != nil {
		m.cfg.LiveTurnActiveProvider = next.LiveTurnActiveProvider
		m.liveTurnPromptSeams.ActiveProvider = next.LiveTurnActiveProvider
	}
	m.cfg.ImageInputMode = next.ImageInputMode
	m.cfg.AuxiliaryVision = next.AuxiliaryVision
	if next.AccountUsage != nil {
		m.cfg.AccountUsage = next.AccountUsage
	}
	if next.ToolRegistry != nil {
		m.cfg.ToolRegistry = next.ToolRegistry
	}
	if next.SkillRuntime != nil {
		m.cfg.SkillRuntime = next.SkillRuntime
	}
	m.cfg.AgentRouting = next.AgentRouting
	m.agentRouter = NewAgentRouter(next.AgentRouting.Agents, next.AgentRouting.Bindings)
	m.agentRoutingEnabled = next.AgentRouting.Enabled
	if next.AgentRuntimeFactory != nil {
		m.cfg.AgentRuntimeFactory = next.AgentRuntimeFactory
	}
	if next.ReloadConfig != nil {
		m.cfg.ReloadConfig = next.ReloadConfig
	}
	m.mu.Unlock()

	evidence := RuntimeConfigReloadEvidence{Status: RuntimeConfigReloadApplied, Redacted: true}
	m.writeRuntimeStatus(ctx, RuntimeStatusUpdate{ConfigReloadEvidence: &evidence})
}

func (m *Manager) recordConfigReloadFailure(ctx context.Context, err error) {
	evidence := RuntimeConfigReloadEvidence{
		Status:   RuntimeConfigReloadFailed,
		Error:    sanitizeConfigReloadError(err),
		Redacted: true,
	}
	m.writeRuntimeStatus(ctx, RuntimeStatusUpdate{ConfigReloadEvidence: &evidence})
}

func (m *Manager) handleReloadCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	if err := m.Reload(ctx); err != nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Config reload failed; continuing with the last good config. error="+sanitizeConfigReloadError(err))
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Config reloaded. Reloadable gateway settings are active without a restart.")
}

func sanitizeConfigReloadError(err error) string { return configreload.SanitizeError(err) }
