package model

import (
	"strings"

	gatewaydelivery "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery"
)

// DeliveryTarget converts a directory entry into the gateway delivery target
// contract. Keeping this adapter separate from Entry normalization lets the
// core directory value contract stay independent from gateway delivery types.
func DeliveryTarget(platform string, entry Entry) gatewaydelivery.Target {
	chatID := trimText(entry.ChatID)
	threadID := trimText(entry.ThreadID)
	if chatID == "" {
		parts := strings.SplitN(trimText(entry.ID), ":", 2)
		chatID = parts[0]
		if len(parts) == 2 && threadID == "" {
			threadID = parts[1]
		}
	}
	return gatewaydelivery.Target{Platform: platform, ChatID: chatID, ThreadID: threadID, IsExplicit: true}
}
