package legacy

import "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/discord/legacy/transport"

func NewRealClient(token string) (Client, error) { return transport.NewRealClient(token) }
