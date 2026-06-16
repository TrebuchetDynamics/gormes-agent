package delivery

import (
	"strings"
	"unicode"

	gatewaydelivery "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery"
	entrymodel "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model/entry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model/policy"
)

// Target converts a directory entry into the gateway delivery target contract.
// Keeping this adapter separate from Entry normalization lets the core
// directory value contract stay independent from gateway delivery types.
func Target(platform string, entry entrymodel.Entry) gatewaydelivery.Target {
	platform = strings.ToLower(policy.TrimText(platform))
	if platform == "" || platform == "local" {
		return gatewaydelivery.Target{}
	}
	chatID := policy.TrimText(entry.ChatID)
	threadID := policy.TrimText(entry.ThreadID)
	if containsControlRune(chatID) || containsControlRune(threadID) {
		return gatewaydelivery.Target{}
	}
	if chatID == "" {
		entryID := policy.TrimText(entry.ID)
		if containsControlRune(entryID) {
			return gatewaydelivery.Target{}
		}
		if parsed, err := gatewaydelivery.ParseTarget(platform+":"+entryID, nil); err == nil && parsed.ChatID != "" {
			return parsed
		}
		parts := strings.SplitN(entryID, ":", 2)
		chatID = parts[0]
		if len(parts) == 2 && threadID == "" {
			threadID = parts[1]
		}
	}
	return gatewaydelivery.Target{Platform: platform, ChatID: chatID, ThreadID: threadID, IsExplicit: true}
}

func containsControlRune(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}
