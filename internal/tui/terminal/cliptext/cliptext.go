package cliptext

import (
	"strings"
	"unicode/utf8"
)

type Result struct {
	Text     string
	OK       bool
	Evidence string
	Attempts []string
}

// IsUsable rejects empty or binary-looking terminal clipboard payloads.
func IsUsable(text string) bool {
	if strings.TrimSpace(text) == "" || !utf8.ValidString(text) {
		return false
	}
	for _, r := range text {
		switch r {
		case '\t', '\n', '\r':
			continue
		case '\uFFFD':
			return false
		}
		if r < 0x20 {
			return false
		}
	}
	return true
}
