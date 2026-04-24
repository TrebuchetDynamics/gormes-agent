package gateway

import (
	"context"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/skills"
)

// SkillsBrowser is the gateway seam that resolves the shared skills browse
// view on demand. Implementations load installed + hub skills from disk and
// return the formatted summary text so the gateway handler can send it back
// without pulling skills storage internals into the gateway package.
type SkillsBrowser interface {
	Browse(ctx context.Context, page, perPage int) (skills.BrowseView, string, error)
}

const skillsBrowserUnconfiguredMessage = "Skills browser is not configured on this gateway."

func (m *Manager) handleSkillsCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	if m.cfg.SkillsBrowser == nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, skillsBrowserUnconfiguredMessage)
		return
	}

	page := parseSkillsPageArg(ev.Text)
	_, text, err := m.cfg.SkillsBrowser.Browse(ctx, page, skills.DefaultBrowsePerPage)
	if err != nil {
		m.log.Warn("browse skills", "platform", ev.Platform, "chat_id", ev.ChatID, "err", err)
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Failed to browse skills: "+err.Error())
		return
	}
	if strings.TrimSpace(text) == "" {
		text = "No skills to browse."
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, text)
}

func parseSkillsPageArg(text string) int {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return 1
	}
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return 1
	}
	page, err := strconv.Atoi(fields[0])
	if err != nil || page <= 0 {
		return 1
	}
	return page
}
