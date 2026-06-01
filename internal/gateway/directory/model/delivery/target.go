package delivery

import (
	"strings"

	gatewaydelivery "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery"
	entrymodel "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model/entry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model/policy"
)

// Target converts a directory entry into the gateway delivery target contract.
// Keeping this adapter separate from Entry normalization lets the core
// directory value contract stay independent from gateway delivery types.
func Target(platform string, entry entrymodel.Entry) gatewaydelivery.Target {
	chatID := policy.TrimText(entry.ChatID)
	threadID := policy.TrimText(entry.ThreadID)
	if chatID == "" {
		parts := strings.SplitN(policy.TrimText(entry.ID), ":", 2)
		chatID = parts[0]
		if len(parts) == 2 && threadID == "" {
			threadID = parts[1]
		}
	}
	return gatewaydelivery.Target{Platform: platform, ChatID: chatID, ThreadID: threadID, IsExplicit: true}
}
