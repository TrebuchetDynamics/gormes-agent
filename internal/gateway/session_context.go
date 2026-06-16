package gateway

import (
	"context"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/sessionctx"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

// SessionSource describes the gateway-facing origin of a turn.
type SessionSource = sessionctx.Source

// SessionContext is the deterministic per-turn prompt block the gateway
// injects so the agent knows where the turn came from and which delivery
// targets are available.
type SessionContext = sessionctx.Context

// AgentContext is redacted route evidence for a multi-agent gateway turn.
// It intentionally contains only operator-safe identifiers and local paths.
type AgentContext = sessionctx.Agent

type resolvedSession = sessionctx.ResolvedSession

func sessionSourceFromInbound(ev InboundEvent) SessionSource {
	platform := normalizedPlatformName(ev.Platform)
	chatType := strings.TrimSpace(ev.ChatType)
	if chatType == "" {
		chatType = "dm"
		if strings.TrimSpace(ev.ThreadID) != "" {
			chatType = "thread"
		}
	}
	messageID := strings.TrimSpace(ev.MessageID)
	if messageID == "" {
		switch platformBaseName(platform) {
		case "discord", "matrix", "mattermost":
			messageID = strings.TrimSpace(ev.MsgID)
		}
	}
	return SessionSource{
		Platform:     platform,
		ChatID:       strings.TrimSpace(ev.ChatID),
		ChatName:     strings.TrimSpace(ev.ChatName),
		ChatType:     chatType,
		UserID:       strings.TrimSpace(ev.UserID),
		UserName:     strings.TrimSpace(ev.UserName),
		ThreadID:     strings.TrimSpace(ev.ThreadID),
		GuildID:      strings.TrimSpace(ev.GuildID),
		ParentChatID: strings.TrimSpace(ev.ParentChatID),
		MessageID:    messageID,
	}
}

func resolveSessionID(ctx context.Context, smap session.Map, chatKey string) (string, error) {
	return sessionctx.ResolveSessionID(ctx, smap, chatKey)
}

func resolveSession(ctx context.Context, smap session.Map, chatKey string) (resolvedSession, error) {
	return sessionctx.ResolveSession(ctx, smap, chatKey)
}

func cleanResumePath(path []string) []string {
	return sessionctx.CleanResumePath(path)
}

// BuildSessionContextPrompt renders the gateway's per-turn session metadata as
// a stable system block. Ordering is deterministic so prompt caching and tests
// stay predictable.
func BuildSessionContextPrompt(ctx SessionContext) string {
	return sessionctx.BuildPrompt(ctx)
}
