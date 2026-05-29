package gateway

import (
	"context"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
)

type ReloadSkillsReplyRequest struct {
	SkillCount int
	ScanError  string
	Refreshes  []SkillGroupRefreshResult
}

func RenderReloadSkillsReply(req ReloadSkillsReplyRequest) string {
	degraded := strings.TrimSpace(req.ScanError) != ""
	for _, refresh := range req.Refreshes {
		if strings.TrimSpace(refresh.Error) != "" {
			degraded = true
			break
		}
	}
	header := "Skills Reloaded"
	if degraded {
		header = "Skills reload degraded"
	}
	lines := []string{header}
	if req.ScanError != "" {
		lines = append(lines, "skill scan: "+req.ScanError)
	} else {
		lines = append(lines, fmt.Sprintf("%d skill(s) available", req.SkillCount))
	}
	for _, refresh := range req.Refreshes {
		channel := strings.TrimSpace(refresh.Channel)
		if channel == "" {
			channel = "unknown"
		}
		if strings.TrimSpace(refresh.Error) != "" {
			lines = append(lines, fmt.Sprintf("%s: refresh error: %s", channel, refresh.Error))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: refreshed %d command(s), %d hidden", channel, refresh.Count, refresh.Hidden))
	}
	if len(req.Refreshes) == 0 {
		lines = append(lines, "adapter refresh: none")
	}
	return strings.Join(lines, "\n")
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
