package messaging

import (
	"os"
	"strings"
)

// ReactionsEnabledFromEnv reports whether Discord lifecycle reactions are enabled.
func ReactionsEnabledFromEnv() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("DISCORD_REACTIONS"))) {
	case "false", "0", "no":
		return false
	default:
		return true
	}
}
