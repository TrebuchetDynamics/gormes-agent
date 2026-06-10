package unauthorizeddm

import (
	"context"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/pairing"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
)

// Behavior controls the shared response contract for direct messages from
// users that did not pass gateway authorization.
type Behavior string

const (
	BehaviorDeny   Behavior = "deny"
	BehaviorPair   Behavior = "pair"
	BehaviorIgnore Behavior = "ignore"
)

// DenialText is intentionally terse so it does not disclose configured chats,
// sessions, or pairing state.
const DenialText = "Access denied."

const maxPromptTokenRunes = 32

// Event is the channel-neutral unauthorized direct-message input.
type Event struct {
	Platform      string
	ChatID        string
	ChatName      string
	UserID        string
	UserName      string
	DirectMessage bool
	PairingUserID string
}

// GeneratePairingCodeFunc persists or evaluates a pairing-code request.
type GeneratePairingCodeFunc func(context.Context, pairing.PairingCodeRequest) (pairing.PairingCodeResult, error)

// SendFunc sends a bounded public reply to the originating chat.
type SendFunc func(context.Context, string, string) error

// Policy carries the shared unauthorized-DM dependencies.
type Policy struct {
	Behavior            Behavior
	GeneratePairingCode GeneratePairingCodeFunc
	Send                SendFunc
}

// Decision reports the policy outcome without exposing authorized-session state.
type Decision struct {
	Handled       bool
	StartAgent    bool
	ReplySent     bool
	PairingStatus pairing.PairingCodeStatus
}

// NormalizeBehavior returns a supported unauthorized-DM mode. The open-gateway
// default mirrors upstream Hermes: unknown DMs are offered a pairing code unless
// the operator configures a quieter mode.
func NormalizeBehavior(behavior Behavior) Behavior {
	switch Behavior(strings.ToLower(strings.TrimSpace(string(behavior)))) {
	case BehaviorDeny:
		return BehaviorDeny
	case BehaviorIgnore:
		return BehaviorIgnore
	case BehaviorPair:
		return BehaviorPair
	default:
		return BehaviorPair
	}
}

// Handle applies the shared unauthorized-direct-message policy. Callers invoke
// this only after normal authorization fails; the function never starts an
// agent turn.
func Handle(ctx context.Context, ev Event, policy Policy) (Decision, error) {
	decision := Decision{Handled: true}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return decision, err
	}
	if !ev.DirectMessage {
		return decision, nil
	}

	switch NormalizeBehavior(policy.Behavior) {
	case BehaviorDeny:
		if policy.GeneratePairingCode != nil {
			result, err := policy.GeneratePairingCode(ctx, pairingRequestFromEvent(ev, true))
			if err != nil {
				return decision, err
			}
			decision.PairingStatus = result.Status
		}
		return sendReply(ctx, policy.Send, ev.ChatID, DenialText, decision)
	case BehaviorIgnore:
		return decision, nil
	case BehaviorPair:
		if policy.GeneratePairingCode == nil {
			return decision, nil
		}
		result, err := policy.GeneratePairingCode(ctx, pairingRequestFromEvent(ev, false))
		if err != nil {
			return decision, err
		}
		decision.PairingStatus = result.Status
		if result.Status != pairing.PairingCodeIssued {
			return decision, nil
		}
		text := FormatPairingPrompt(ev.Platform, result.Code)
		return sendReply(ctx, policy.Send, ev.ChatID, text, decision)
	default:
		return decision, nil
	}
}

// FormatPairingPrompt returns the bounded public prompt sent for pair-mode
// unauthorized DMs.
func FormatPairingPrompt(platformName, code string) string {
	platformName = promptToken(platformName)
	code = promptToken(code)
	return fmt.Sprintf("Hi. I don't recognize this DM yet.\n\nPairing code: `%s`\nAsk the operator to run: `gormes pairing approve %s %s`", code, platformName, code)
}

func promptToken(value string) string {
	value = redaction.RedactSecrets(value)
	value = strings.NewReplacer(
		"api_key=[redacted]", "[redacted]",
		"api-key=[redacted]", "[redacted]",
		"apikey=[redacted]", "[redacted]",
		"token=[redacted]", "[redacted]",
		"secret=[redacted]", "[redacted]",
		"password=[redacted]", "[redacted]",
		"`", "'",
	).Replace(value)
	return truncatePromptToken(strings.Join(strings.Fields(value), " "))
}

func truncatePromptToken(value string) string {
	runes := []rune(value)
	if len(runes) <= maxPromptTokenRunes {
		return value
	}
	return string(runes[:maxPromptTokenRunes])
}

func pairingRequestFromEvent(ev Event, allowlistDenied bool) pairing.PairingCodeRequest {
	userID := strings.TrimSpace(ev.PairingUserID)
	if userID == "" {
		userID = strings.TrimSpace(ev.UserID)
	}
	userName := strings.TrimSpace(ev.UserName)
	if userName == "" && userID != "" && userID == strings.TrimSpace(ev.ChatID) && ev.DirectMessage {
		userName = strings.TrimSpace(ev.ChatName)
	}
	return pairing.PairingCodeRequest{
		Platform:        ev.Platform,
		UserID:          userID,
		UserName:        userName,
		AllowlistDenied: allowlistDenied,
	}
}

func sendReply(ctx context.Context, send SendFunc, chatID, text string, decision Decision) (Decision, error) {
	if send == nil {
		return decision, nil
	}
	if err := ctx.Err(); err != nil {
		return decision, err
	}
	if err := send(ctx, chatID, text); err != nil {
		return decision, err
	}
	decision.ReplySent = true
	return decision, nil
}
