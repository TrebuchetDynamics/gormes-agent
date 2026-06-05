package gateway

import (
	"context"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/reloadskills"
)

type ReloadSkillsReplyRequest struct {
	SkillCount int
	ScanError  string
	Refreshes  []SkillGroupRefreshResult
}

func RenderReloadSkillsReply(req ReloadSkillsReplyRequest) string {
	return reloadskills.RenderReply(reloadSkillsReplyRequest(req))
}

func reloadSkillsReplyRequest(req ReloadSkillsReplyRequest) reloadskills.ReplyRequest {
	refreshes := make([]reloadskills.RefreshResult, 0, len(req.Refreshes))
	for _, refresh := range req.Refreshes {
		refreshes = append(refreshes, reloadskills.RefreshResult{
			Channel: refresh.Channel,
			Count:   refresh.Count,
			Hidden:  refresh.Hidden,
			Error:   refresh.Error,
		})
	}
	return reloadskills.ReplyRequest{
		SkillCount: req.SkillCount,
		ScanError:  req.ScanError,
		Refreshes:  refreshes,
	}
}

func (m *Manager) handleReloadSkillsCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	commands, scanErr := m.reloadSkillCommands(ctx)
	refreshes := m.RefreshSkillGroups(ctx)
	text := RenderReloadSkillsReply(ReloadSkillsReplyRequest{
		SkillCount: len(commands),
		ScanError:  sanitizeReloadSkillsError(scanErr),
		Refreshes:  refreshes,
	})
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, text)
}

func (m *Manager) reloadSkillCommands(ctx context.Context) ([]PlatformCommand, error) {
	if m == nil || m.cfg.SkillRuntime == nil {
		return nil, nil
	}
	commands, _, err := m.cfg.SkillRuntime.SkillSlashCommands(ctx, skillsRuntimeOptions())
	if err != nil {
		if m.log != nil {
			m.log.Warn("gateway: reload-skills scan failed", "err", err)
		}
		return nil, err
	}
	out := make([]PlatformCommand, 0, len(commands))
	for _, command := range commands {
		name := strings.TrimPrefix(strings.TrimSpace(command.Command), "/")
		if name == "" {
			continue
		}
		out = append(out, PlatformCommand{Name: name, Description: command.Description})
	}
	return out, nil
}

func skillsRuntimeOptions() skills.RuntimeOptions { return skills.RuntimeOptions{} }

func sanitizeReloadSkillsError(err error) string {
	if err == nil {
		return ""
	}
	return sanitizeConfigReloadError(err)
}
