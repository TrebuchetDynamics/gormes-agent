package slack

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/channelutil"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const (
	slackApprovalSectionLimit = 3000

	slackActionApproveOnce    = "hermes_approve_once"
	slackActionApproveSession = "hermes_approve_session"
	slackActionApproveAlways  = "hermes_approve_always"
	slackActionDeny           = "hermes_deny"
)

var ErrSlackApprovalUnavailable = errors.New("slack_approval_unavailable")

type ApprovalPrompt struct {
	ChannelID   string
	ThreadTS    string
	Command     string
	SessionKey  string
	TicketID    uint64
	Description string
}

type slackApprovalUnavailableError struct {
	reason string
	err    error
}

func (e slackApprovalUnavailableError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s: %s: %v", ErrSlackApprovalUnavailable, e.reason, e.err)
	}
	return fmt.Sprintf("%s: %s", ErrSlackApprovalUnavailable, e.reason)
}

func (e slackApprovalUnavailableError) Unwrap() error {
	return ErrSlackApprovalUnavailable
}

type blockMessageClient interface {
	PostBlockMessage(ctx context.Context, channelID, threadTS, text string, blocks []SlackBlock) (string, error)
	UpdateBlockMessage(ctx context.Context, channelID, ts, text string, blocks []SlackBlock) error
}

func (b *Bot) SendExecApproval(ctx context.Context, prompt ApprovalPrompt) (string, error) {
	blockClient, ok := b.client.(blockMessageClient)
	if b.client == nil || !ok {
		return "", slackApprovalUnavailable("not_connected", nil)
	}
	if b.cfg.ApprovalResolver == nil {
		return "", slackApprovalUnavailable("approval_storage_unavailable", nil)
	}

	channelID := strings.TrimSpace(prompt.ChannelID)
	if channelID == "" {
		return "", slackApprovalUnavailable("channel_required", nil)
	}
	if b.cfg.AllowedChannelID == "" || channelID != b.cfg.AllowedChannelID {
		return "", slackApprovalUnavailable("unregistered_channel", nil)
	}

	sessionKey := strings.TrimSpace(prompt.SessionKey)
	if sessionKey == "" {
		return "", slackApprovalUnavailable("session_key_required", nil)
	}

	blocks := buildSlackApprovalBlocks(prompt.Command, prompt.Description, sessionKey)
	text := slackApprovalFallbackText(prompt.Command)
	ts, err := blockClient.PostBlockMessage(ctx, channelID, strings.TrimSpace(prompt.ThreadTS), text, blocks)
	if err != nil {
		return "", slackApprovalUnavailable("post_failed", err)
	}
	b.rememberApprovalPrompt(ts, blocks, prompt.TicketID)
	return ts, nil
}

func (b *Bot) handleApprovalAction(ctx context.Context, e Event) error {
	action := e.ApprovalAction
	if action == nil {
		return nil
	}
	choice, ok := slackApprovalChoice(action.ActionID)
	if !ok {
		return nil
	}
	if b.cfg.ApprovalResolver == nil {
		return nil
	}
	blockClient, ok := b.client.(blockMessageClient)
	if b.client == nil || !ok {
		return slackApprovalUnavailable("not_connected", nil)
	}

	channelID := firstNonEmpty(action.ChannelID, e.ChannelID)
	if b.cfg.AllowedChannelID == "" || channelID != b.cfg.AllowedChannelID {
		return nil
	}
	sessionKey := strings.TrimSpace(action.SessionKey)
	messageTS := strings.TrimSpace(action.MessageTS)
	if sessionKey == "" || messageTS == "" {
		return nil
	}

	blocks, ticketID, claimed := b.claimApprovalPrompt(messageTS)
	if !claimed {
		return nil
	}

	actorID := firstNonEmpty(action.UserID, e.UserID)
	actorName := firstNonEmpty(action.UserName, actorID)
	decision := slackApprovalDecisionText(choice, actorName)
	updateBlocks := resolvedSlackApprovalBlocks(blocks, decision)
	updateErr := blockClient.UpdateBlockMessage(ctx, channelID, messageTS, decision, updateBlocks)
	resolveErr := b.cfg.ApprovalResolver.ResolveGatewayApproval(ctx, gateway.ApprovalResolution{
		SessionKey: sessionKey,
		TicketID:   ticketID,
		Choice:     choice,
		Platform:   "slack",
		ChatID:     channelID,
		MessageID:  messageTS,
		ActorID:    actorID,
		Evidence: map[string]string{
			"slack_action_id":  action.ActionID,
			"slack_message_ts": messageTS,
		},
	})
	if updateErr != nil {
		return updateErr
	}
	return resolveErr
}

func slackApprovalUnavailable(reason string, err error) error {
	return slackApprovalUnavailableError{reason: reason, err: err}
}

func buildSlackApprovalBlocks(command, description, sessionKey string) []SlackBlock {
	return []SlackBlock{
		{
			"type": "section",
			"text": SlackBlock{
				"type": "mrkdwn",
				"text": slackApprovalSectionText(command, description),
			},
		},
		{
			"type": "actions",
			"elements": []SlackBlock{
				slackApprovalButton("Allow Once", slackActionApproveOnce, sessionKey, "primary"),
				slackApprovalButton("Allow Session", slackActionApproveSession, sessionKey, ""),
				slackApprovalButton("Always Allow", slackActionApproveAlways, sessionKey, ""),
				slackApprovalButton("Deny", slackActionDeny, sessionKey, "danger"),
			},
		},
	}
}

func slackApprovalButton(label, actionID, sessionKey, style string) SlackBlock {
	button := SlackBlock{
		"type":      "button",
		"text":      SlackBlock{"type": "plain_text", "text": label},
		"action_id": actionID,
		"value":     sessionKey,
	}
	if style != "" {
		button["style"] = style
	}
	return button
}

func slackApprovalSectionText(command, description string) string {
	cmd := sanitizeSlackApprovalText(command)
	if cmd == "" {
		cmd = "(empty command)"
	}
	desc := sanitizeSlackApprovalText(description)
	if desc == "" {
		desc = "dangerous command"
	}
	desc = channelutil.TruncateRunes(desc, 500)
	cmd = channelutil.TruncateRunes(cmd, 2900)

	for {
		text := formatSlackApprovalSection(cmd, desc)
		if runeLen(text) <= slackApprovalSectionLimit {
			return text
		}
		overhead := runeLen(formatSlackApprovalSection("", desc))
		maxCommand := slackApprovalSectionLimit - overhead
		if maxCommand > 10 {
			cmd = channelutil.TruncateRunes(cmd, maxCommand)
			continue
		}
		if runeLen(desc) > 80 {
			desc = channelutil.TruncateRunes(desc, 80)
			continue
		}
		return channelutil.TruncateRunes(text, slackApprovalSectionLimit)
	}
}

func sanitizeSlackApprovalText(value string) string {
	replacer := strings.NewReplacer(
		"`", "'",
		"*", "'",
		"<", "(",
		">", ")",
	)
	return strings.Join(strings.Fields(replacer.Replace(value)), " ")
}

func formatSlackApprovalSection(command, description string) string {
	return ":warning: *Command Approval Required*\n```" + command + "```\nReason: " + description
}

func slackApprovalFallbackText(command string) string {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		cmd = "(empty command)"
	}
	return "Command approval required: " + channelutil.TruncateRunes(cmd, 100)
}

func slackApprovalChoice(actionID string) (gateway.ApprovalChoice, bool) {
	switch strings.TrimSpace(actionID) {
	case slackActionApproveOnce:
		return gateway.ApprovalChoiceOnce, true
	case slackActionApproveSession:
		return gateway.ApprovalChoiceSession, true
	case slackActionApproveAlways:
		return gateway.ApprovalChoiceAlways, true
	case slackActionDeny:
		return gateway.ApprovalChoiceDeny, true
	default:
		return "", false
	}
}

func isSlackApprovalAction(actionID string) bool {
	_, ok := slackApprovalChoice(actionID)
	return ok
}

func slackApprovalDecisionText(choice gateway.ApprovalChoice, actor string) string {
	actor = sanitizeSlackApprovalText(actor)
	if actor == "" {
		actor = "unknown user"
	}
	switch choice {
	case gateway.ApprovalChoiceOnce:
		return "Approved once by " + actor
	case gateway.ApprovalChoiceSession:
		return "Approved for session by " + actor
	case gateway.ApprovalChoiceAlways:
		return "Approved permanently by " + actor
	case gateway.ApprovalChoiceDeny:
		return "Denied by " + actor
	default:
		return "Resolved by " + actor
	}
}

func resolvedSlackApprovalBlocks(blocks []SlackBlock, decision string) []SlackBlock {
	out := make([]SlackBlock, 0, 2)
	for _, block := range blocks {
		if block["type"] == "section" {
			out = append(out, cloneSlackBlock(block))
			break
		}
	}
	if len(out) == 0 {
		out = append(out, SlackBlock{
			"type": "section",
			"text": SlackBlock{"type": "mrkdwn", "text": decision},
		})
	}
	out = append(out, SlackBlock{
		"type": "context",
		"elements": []SlackBlock{
			{"type": "mrkdwn", "text": decision},
		},
	})
	return out
}

func (b *Bot) rememberApprovalPrompt(messageTS string, blocks []SlackBlock, ticketID uint64) {
	if strings.TrimSpace(messageTS) == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.approvalResolved == nil {
		b.approvalResolved = map[string]bool{}
	}
	if b.approvalBlocks == nil {
		b.approvalBlocks = map[string][]SlackBlock{}
	}
	if b.approvalTickets == nil {
		b.approvalTickets = map[string]uint64{}
	}
	b.approvalResolved[messageTS] = false
	b.approvalBlocks[messageTS] = cloneSlackBlocks(blocks)
	b.approvalTickets[messageTS] = ticketID
}

func (b *Bot) claimApprovalPrompt(messageTS string) ([]SlackBlock, uint64, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.approvalResolved == nil {
		return nil, 0, false
	}
	resolved, ok := b.approvalResolved[messageTS]
	if !ok || resolved {
		return nil, 0, false
	}
	b.approvalResolved[messageTS] = true
	return cloneSlackBlocks(b.approvalBlocks[messageTS]), b.approvalTickets[messageTS], true
}

func cloneSlackBlocks(blocks []SlackBlock) []SlackBlock {
	if blocks == nil {
		return nil
	}
	out := make([]SlackBlock, 0, len(blocks))
	for _, block := range blocks {
		out = append(out, cloneSlackBlock(block))
	}
	return out
}

func cloneSlackBlock(block SlackBlock) SlackBlock {
	if block == nil {
		return nil
	}
	out := make(SlackBlock, len(block))
	for k, v := range block {
		out[k] = cloneSlackValue(v)
	}
	return out
}

func cloneSlackValue(value any) any {
	switch typed := value.(type) {
	case SlackBlock:
		return cloneSlackBlock(typed)
	case map[string]any:
		return cloneSlackBlock(SlackBlock(typed))
	case []SlackBlock:
		return cloneSlackBlocks(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, cloneSlackValue(item))
		}
		return out
	default:
		return typed
	}
}

func runeLen(value string) int {
	return len([]rune(value))
}

func firstNonEmpty(values ...string) string { return channelutil.FirstNonEmpty(values...) }
