package send

import "strings"

func IsMarkdownParseError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "parse") || strings.Contains(lower, "markdown")
}

func IsThreadNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "thread not found")
}

func IsReplyNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "message to be replied not found") ||
		strings.Contains(lower, "reply message not found") ||
		strings.Contains(lower, "replied message not found")
}

func IsTimedOutError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "timedout") ||
		strings.Contains(lower, "timed out") ||
		strings.Contains(lower, "i/o timeout")
}

func IsTransientNetworkError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "network") ||
		strings.Contains(lower, "connection") ||
		strings.Contains(lower, "eof") ||
		strings.Contains(lower, "reset") ||
		strings.Contains(lower, "broken pipe")
}
