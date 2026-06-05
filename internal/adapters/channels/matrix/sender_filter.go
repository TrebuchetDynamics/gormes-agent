package matrix

import "strings"

// SenderFilter classifies Matrix senders before pairing or gateway dispatch.
// It is pure string logic: no SDK types, no network state.
type SenderFilter struct {
	ownUserID string
}

func NewSenderFilter(ownUserID string) SenderFilter {
	return SenderFilter{ownUserID: strings.TrimSpace(ownUserID)}
}

func (f SenderFilter) IsSelfSender(sender string) bool {
	own := strings.TrimSpace(f.ownUserID)
	if own == "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(sender), own)
}

func (f SenderFilter) IsSystemOrBridgeSender(sender string) bool {
	s := strings.TrimSpace(sender)
	if s == "" {
		return true
	}
	if strings.HasPrefix(s, "@") {
		s = strings.TrimPrefix(s, "@")
	}
	localpart, _, _ := strings.Cut(s, ":")
	if localpart == "" {
		return true
	}
	return strings.HasPrefix(localpart, "_")
}

func (f SenderFilter) ShouldDropSender(sender string) bool {
	return f.IsSelfSender(sender) || f.IsSystemOrBridgeSender(sender)
}
