package urlsafety

import "net"

// IsBlockedIP reports whether the IP should be blocked for SSRF protection.
func IsBlockedIP(ip net.IP) bool {
	return isBlockedIP(ip)
}
