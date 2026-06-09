package sessionctx

import (
	"context"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
)

// Source describes the gateway-facing origin of a turn.
type Source struct {
	Platform     string
	ChatID       string
	ChatName     string
	ChatType     string
	UserID       string
	UserName     string
	ThreadID     string
	GuildID      string
	ParentChatID string
	MessageID    string
}

// Context is the deterministic per-turn prompt block the gateway injects so
// the agent knows where the turn came from and which delivery targets are available.
type Context struct {
	Source                Source
	Agent                 Agent
	SessionKey            string
	SessionID             string
	RequestedSessionID    string
	ResumePath            []string
	ResumeStatus          string
	NonResumableSessionID string
	NonResumableReason    string
	ConnectedPlatforms    []string
}

// Agent is redacted route evidence for a multi-agent gateway turn.
type Agent struct {
	ID          string
	Name        string
	Workspace   string
	AgentDir    string
	BindingTier string
}

type ResolvedSession struct {
	SessionID             string
	RequestedSessionID    string
	ResumePath            []string
	ResumeStatus          string
	NonResumableSessionID string
	NonResumableReason    string
}

func ResolveSessionID(ctx context.Context, smap session.Map, chatKey string) (string, error) {
	resolved, err := ResolveSession(ctx, smap, chatKey)
	return resolved.SessionID, err
}

func ResolveSession(ctx context.Context, smap session.Map, chatKey string) (ResolvedSession, error) {
	key := strings.TrimSpace(chatKey)
	if key == "" || smap == nil {
		return ResolvedSession{SessionID: key}, nil
	}
	stored, err := smap.Get(ctx, key)
	if err != nil {
		return ResolvedSession{SessionID: key}, err
	}
	if stored = strings.TrimSpace(stored); stored != "" {
		resolved := ResolvedSession{SessionID: stored}
		resolver, ok := smap.(session.LineageResolver)
		if !ok {
			return resolved, nil
		}
		lineage, err := resolver.ResolveLineageTip(ctx, stored)
		if err != nil {
			resolved.RequestedSessionID = stored
			resolved.ResumeStatus = session.LineageStatusError
			return resolved, err
		}
		status := strings.TrimSpace(lineage.Status)
		if status == "" {
			status = session.LineageStatusOK
		}
		resolved.ResumePath = CleanResumePath(lineage.Path)
		if status != session.LineageStatusOK {
			if status != session.LineageStatusMissing || len(resolved.ResumePath) > 1 {
				resolved.RequestedSessionID = stored
				resolved.ResumeStatus = status
			}
			if resolved.ResumeStatus == "" {
				resolved.ResumePath = nil
			}
			return resolved, nil
		}
		if live := strings.TrimSpace(lineage.LiveSessionID); live != "" {
			resolved.SessionID = live
		}
		if resolved.SessionID == stored {
			resolved.ResumePath = nil
		} else {
			resolved.RequestedSessionID = stored
		}
		return resolved, nil
	}
	return ResolvedSession{SessionID: key}, nil
}

func CleanResumePath(path []string) []string {
	out := make([]string, 0, len(path))
	for _, id := range path {
		id = strings.TrimSpace(id)
		if id != "" {
			out = append(out, id)
		}
	}
	return out
}

// BuildPrompt renders the gateway's per-turn session metadata as a stable system block.
func BuildPrompt(ctx Context) string {
	source := ctx.Source
	lines := []string{"## Current Session Context", ""}

	platform := promptLineValue(strings.ToLower(strings.TrimSpace(source.Platform)))
	chatID := promptLineValue(source.ChatID)
	switch {
	case platform == "" && chatID == "":
		lines = append(lines, "**Source:** unknown")
	case chatID == "":
		lines = append(lines, "**Source:** "+platform)
	default:
		lines = append(lines, "**Source:** "+platform+" chat `"+chatID+"`")
	}
	if userID := promptLineValue(source.UserID); userID != "" {
		lines = append(lines, "**User ID:** `"+userID+"`")
	}
	if guildID := promptLineValue(source.GuildID); guildID != "" {
		lines = append(lines, "**Guild ID:** `"+guildID+"`")
	}
	if parentChatID := promptLineValue(source.ParentChatID); parentChatID != "" {
		lines = append(lines, "**Parent Chat ID:** `"+parentChatID+"`")
	}
	if threadID := promptLineValue(source.ThreadID); threadID != "" {
		lines = append(lines, "**Thread ID:** `"+threadID+"`")
	}
	if messageID := promptLineValue(source.MessageID); messageID != "" {
		lines = append(lines, "**Message ID:** `"+messageID+"`")
	}
	if agentID := promptLineValue(ctx.Agent.ID); agentID != "" {
		lines = append(lines, "**Agent ID:** `"+agentID+"`")
	}
	if agentName := promptLineValue(ctx.Agent.Name); agentName != "" && agentName != promptLineValue(ctx.Agent.ID) {
		lines = append(lines, "**Agent Name:** "+agentName)
	}
	if tier := promptLineValue(ctx.Agent.BindingTier); tier != "" {
		lines = append(lines, "**Agent Binding:** `"+tier+"`")
	}
	if workspace := promptLineValue(ctx.Agent.Workspace); workspace != "" {
		lines = append(lines, "**Agent Workspace:** `"+workspace+"`")
	}
	if agentDir := promptLineValue(ctx.Agent.AgentDir); agentDir != "" {
		lines = append(lines, "**Agent Dir:** `"+agentDir+"`")
	}
	if key := promptLineValue(ctx.SessionKey); key != "" {
		lines = append(lines, "**Session Key:** `"+key+"`")
	}
	if sessionID := promptLineValue(ctx.SessionID); sessionID != "" {
		lines = append(lines, "**Session ID:** `"+sessionID+"`")
	}
	if requested := promptLineValue(ctx.RequestedSessionID); requested != "" {
		lines = append(lines, "**Requested Session ID:** `"+requested+"`")
	}
	if len(ctx.ResumePath) > 1 {
		parts := make([]string, 0, len(ctx.ResumePath))
		for _, id := range ctx.ResumePath {
			if id = promptLineValue(id); id != "" {
				parts = append(parts, "`"+id+"`")
			}
		}
		if len(parts) > 1 {
			lines = append(lines, "**Resume Continuation:** "+strings.Join(parts, " -> "))
		}
	}
	if status := promptLineValue(ctx.ResumeStatus); status != "" && status != session.LineageStatusOK {
		lines = append(lines, "**Resume Continuation Status:** `"+status+"`")
	}
	if blockedSessionID := promptLineValue(ctx.NonResumableSessionID); blockedSessionID != "" {
		lines = append(lines, "**Non-Resumable Session ID:** `"+blockedSessionID+"`")
	}
	if blockedReason := promptLineValue(ctx.NonResumableReason); blockedReason != "" {
		lines = append(lines, "**Non-Resumable Reason:** `"+blockedReason+"`")
	}

	if platform == "bluebubbles" {
		lines = append(lines, "", "**Platform notes:** You are responding via iMessage. Keep responses short and conversational - think texts, not essays. Structure longer replies as separate short thoughts, each separated by a blank line (double newline). Each block between blank lines will be delivered as its own iMessage bubble, so write accordingly: one idea per bubble, 1-3 sentences each. If the user needs a detailed answer, give the short version first and offer to elaborate.")
	}

	targets := []string{"`origin`", "`local`"}
	if len(ctx.ConnectedPlatforms) > 0 {
		seen := map[string]struct{}{"origin": {}, "local": {}}
		platforms := make([]string, 0, len(ctx.ConnectedPlatforms))
		for _, name := range ctx.ConnectedPlatforms {
			name = promptLineValue(strings.ToLower(strings.TrimSpace(name)))
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			platforms = append(platforms, name)
		}
		sort.Strings(platforms)
		for _, name := range platforms {
			targets = append(targets, "`"+name+"`")
		}
	}
	lines = append(lines, "**Delivery Targets:** "+strings.Join(targets, ", "))
	return strings.Join(lines, "\n")
}

func promptLineValue(value string) string {
	replacer := strings.NewReplacer(
		"`", "'",
		"*", "'",
		"#", "＃",
	)
	value = replacer.Replace(value)
	value = redaction.RedactSecrets(value)
	fields := strings.Fields(value)
	for i, field := range fields {
		lower := strings.ToLower(field)
		if strings.Contains(lower, "[redacted]") && secretLikePromptField(lower) {
			fields[i] = "[redacted]"
		}
	}
	return strings.Join(fields, " ")
}

func secretLikePromptField(value string) bool {
	return strings.Contains(value, "api_key") || strings.Contains(value, "api-key") || strings.Contains(value, "apikey") || strings.Contains(value, "token") || strings.Contains(value, "secret") || strings.Contains(value, "password")
}
