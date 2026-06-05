package gateway

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/unauthorizeddm"
)

// UnauthorizedDMBehavior controls the shared response contract for direct
// messages from users that did not pass gateway authorization.
type UnauthorizedDMBehavior = unauthorizeddm.Behavior

const (
	UnauthorizedDMDeny   UnauthorizedDMBehavior = unauthorizeddm.BehaviorDeny
	UnauthorizedDMPair   UnauthorizedDMBehavior = unauthorizeddm.BehaviorPair
	UnauthorizedDMIgnore UnauthorizedDMBehavior = unauthorizeddm.BehaviorIgnore
)

// UnauthorizedDMDenialText is intentionally terse so it does not disclose
// configured chats, sessions, or pairing state.
const UnauthorizedDMDenialText = unauthorizeddm.DenialText

// UnauthorizedDMPolicy carries the shared gateway policy dependencies for
// unknown direct-message senders.
type UnauthorizedDMPolicy struct {
	Behavior     UnauthorizedDMBehavior
	PairingStore *PairingStore
}

// UnauthorizedDMDecision reports the policy outcome without exposing
// authorized-session state.
type UnauthorizedDMDecision = unauthorizeddm.Decision

// NormalizeUnauthorizedDMBehavior returns a supported unauthorized-DM mode.
// The open-gateway default mirrors upstream Hermes: unknown DMs are offered a
// pairing code unless the operator configures a quieter mode.
func NormalizeUnauthorizedDMBehavior(behavior UnauthorizedDMBehavior) UnauthorizedDMBehavior {
	return unauthorizeddm.NormalizeBehavior(behavior)
}

// HandleUnauthorizedDM applies the shared unauthorized-direct-message policy.
// Callers invoke this only after normal authorization fails; the function never
// starts an agent turn.
func HandleUnauthorizedDM(ctx context.Context, ch Channel, ev InboundEvent, policy UnauthorizedDMPolicy) (UnauthorizedDMDecision, error) {
	var generate unauthorizeddm.GeneratePairingCodeFunc
	if policy.PairingStore != nil {
		generate = policy.PairingStore.GeneratePairingCode
	}
	return unauthorizeddm.Handle(ctx, unauthorizeddm.Event{
		Platform:      ev.Platform,
		ChatID:        ev.ChatID,
		ChatName:      ev.ChatName,
		UserID:        ev.UserID,
		UserName:      ev.UserName,
		DirectMessage: ev.IsDirectMessage(),
		PairingUserID: ev.PairingUserID(),
	}, unauthorizeddm.Policy{
		Behavior:            policy.Behavior,
		GeneratePairingCode: generate,
		Send: func(ctx context.Context, chatID, text string) error {
			if ch == nil {
				return nil
			}
			_, err := ch.Send(ctx, chatID, text)
			return err
		},
	})
}

// FormatUnauthorizedDMPairingPrompt returns the bounded public prompt sent for
// pair-mode unauthorized DMs.
func FormatUnauthorizedDMPairingPrompt(platformName, code string) string {
	return unauthorizeddm.FormatPairingPrompt(platformName, code)
}
