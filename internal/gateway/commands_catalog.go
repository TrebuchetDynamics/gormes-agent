package gateway

import (
	"context"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/commandcatalog"
)

type CommandsCatalogRequest struct {
	Platform      string
	RawArgs       string
	BuiltinLines  []string
	SkillCommands []PlatformCommand
}

func RenderCommandsCatalog(req CommandsCatalogRequest) string {
	return commandcatalog.Render(commandcatalog.Request{
		Platform:      req.Platform,
		RawArgs:       req.RawArgs,
		BuiltinLines:  req.BuiltinLines,
		SkillCommands: catalogCommandsFromPlatform(req.SkillCommands),
	})
}

func catalogCommandsFromPlatform(commands []PlatformCommand) []commandcatalog.Command {
	out := make([]commandcatalog.Command, 0, len(commands))
	for _, command := range commands {
		out = append(out, commandcatalog.Command{Name: command.Name, Description: command.Description})
	}
	return out
}

func (m *Manager) handleCommandsCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	platform := strings.TrimSpace(ev.Platform)
	if platform == "" && ch != nil {
		platform = ch.Name()
	}
	text := RenderCommandsCatalog(CommandsCatalogRequest{
		Platform:      platform,
		RawArgs:       strings.Join(commandArgs(ev.Text), " "),
		BuiltinLines:  GatewayHelpLines(),
		SkillCommands: m.enabledSkillPlatformCommands(ctx),
	})
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, text)
}

func (m *Manager) enabledSkillPlatformCommands(ctx context.Context) []PlatformCommand {
	if m == nil || m.cfg.SkillRuntime == nil {
		return nil
	}
	commands, _, err := m.cfg.SkillRuntime.SkillSlashCommands(ctx, skills.RuntimeOptions{})
	if err != nil {
		if m.log != nil {
			m.log.Warn("gateway: skill command catalog scan failed", "err", err)
		}
		return nil
	}
	out := make([]PlatformCommand, 0, len(commands))
	seen := map[string]struct{}{}
	for _, command := range commands {
		name := strings.TrimPrefix(strings.TrimSpace(command.Command), "/")
		if name == "" {
			continue
		}
		if _, collides := ResolveCommand(name); collides {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, PlatformCommand{Name: name, Description: command.Description})
	}
	return sortedPlatformCommands(out)
}
