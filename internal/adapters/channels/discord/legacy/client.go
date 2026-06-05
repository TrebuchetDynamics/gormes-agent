package legacy

import "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/discord/legacy/protocol"

type InboundMessage = protocol.InboundMessage

type Client = protocol.Client

func SessionKey(channelID string) string { return protocol.SessionKey(channelID) }
