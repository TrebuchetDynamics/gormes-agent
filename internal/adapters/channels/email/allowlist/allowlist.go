package allowlist

import "strings"

// SenderDeniedEvidence is bounded denial evidence for non-allowlisted email.
type SenderDeniedEvidence struct {
	Code   string
	Sender string
	Domain string
	Reason string
}

func NormalizeAddress(addr string) string {
	return strings.ToLower(strings.TrimSpace(addr))
}

func SenderAllowed(sender string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, candidate := range allowed {
		if NormalizeAddress(candidate) == sender {
			return true
		}
	}
	return false
}

func AddressDomain(sender string) string {
	_, domain, ok := strings.Cut(sender, "@")
	if !ok {
		return ""
	}
	return strings.TrimSpace(domain)
}

func EvidenceSender(sender string) string {
	local, domain, ok := strings.Cut(sender, "@")
	if !ok {
		if local == "" {
			return "***"
		}
		return local[:1] + "***"
	}
	if local == "" {
		return "***@" + domain
	}
	return local[:1] + "***@" + domain
}
