package platforms

import "strings"

func NormalizePlatformID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
