package choice

import (
	"context"
	"strings"
)

// Choice is the bounded decision a messaging-platform approval button may
// resolve for a pending gateway approval request.
type Choice string

const (
	ChoiceOnce    Choice = "once"
	ChoiceSession Choice = "session"
	ChoiceAlways  Choice = "always"
	ChoiceDeny    Choice = "deny"
)

// Resolution is the redacted evidence passed from channel callbacks into the
// gateway approval store/resolver.
type Resolution struct {
	SessionKey string
	Choice     Choice
	Platform   string
	ChatID     string
	MessageID  string
	ActorID    string
	Evidence   map[string]string
}

// Resolver owns the gateway-side approval state for pending dangerous
// operations. Channel implementations call it after a user chooses a bounded
// approval action.
type Resolver interface {
	ResolveGatewayApproval(context.Context, Resolution) error
}

// ResolverFunc adapts a function to Resolver.
type ResolverFunc func(context.Context, Resolution) error

func (f ResolverFunc) ResolveGatewayApproval(ctx context.Context, res Resolution) error {
	if f == nil {
		return nil
	}
	return f(ctx, res)
}

// ParseChoice normalizes a gateway approval decision label.
func ParseChoice(value string) (Choice, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(ChoiceOnce):
		return ChoiceOnce, true
	case string(ChoiceSession):
		return ChoiceSession, true
	case string(ChoiceAlways):
		return ChoiceAlways, true
	case string(ChoiceDeny):
		return ChoiceDeny, true
	default:
		return "", false
	}
}

// Valid reports whether choice is one of the bounded gateway approval choices.
func Valid(choice Choice) bool {
	switch choice {
	case ChoiceOnce, ChoiceSession, ChoiceAlways, ChoiceDeny:
		return true
	default:
		return false
	}
}
