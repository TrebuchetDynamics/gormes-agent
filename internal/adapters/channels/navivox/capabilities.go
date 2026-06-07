package navivox

import (
	"net/http"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/navivox/capability"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

const navivoxMaxTurnRequestBytes = capability.MaxTurnRequestBytes

type capabilityEndpoint = capability.Endpoint
type capabilityAuth = capability.Auth
type capabilityHealth = capability.Health
type capabilityProfileManagement = capability.ProfileManagement
type capabilityAttachments = capability.Attachments
type capabilityVoice = capability.Voice
type capabilityStreams = capability.Streams
type capabilityDurableReconnect = capability.DurableReconnectSchema
type capabilityDocument = capability.DocumentSchema

func (c *Channel) handleCapabilities(w http.ResponseWriter, r *http.Request, _ string) {
	if r.Method != http.MethodGet {
		writeNavivoxError(w, http.StatusMethodNotAllowed, "", "bad_request", "Method not allowed")
		return
	}
	writeNavivoxJSON(w, http.StatusOK, c.capabilityDocumentForRequest(r))
}

func (c *Channel) capabilityDocument() capabilityDocument {
	return c.capabilityDocumentForRequest(nil)
}

func (c *Channel) capabilityDocumentForRequest(r *http.Request) capabilityDocument {
	matrix := navivoxVoiceProviderMatrix()
	return capability.Document(capability.DocumentParams{
		ProtocolVersion:   navivoxWebSocketProtocol,
		AuthMode:          c.cfg.AuthMode,
		STTProviders:      matrix.STTProviders,
		TTSProviders:      matrix.TTSProviders,
		EffectiveSecurity: navivoxTransportSecurityStatusForRequest(r, c.cfg).EffectiveSecurity,
	})
}

func navivoxDurableReconnectCapability(r *http.Request, cfg config.NavivoxCfg) capabilityDurableReconnect {
	return capability.DurableReconnect(navivoxTransportSecurityStatusForRequest(r, cfg).EffectiveSecurity)
}

func navivoxCapabilityAuthHeaders(mode string) []string {
	return capability.AuthHeaders(mode)
}

func navivoxCapabilityWebSocketProtocols(mode string) []string {
	return capability.WebSocketProtocols(mode)
}

func navivoxAuthModeUsesToken(mode string) bool {
	return capability.AuthModeUsesToken(mode)
}

func navivoxAuthModeUsesTailscale(mode string) bool {
	return capability.AuthModeUsesTailscale(mode)
}

func navivoxCapabilityNames() []string {
	return capability.CapabilityNames()
}

func navivoxEventKinds() []string {
	return capability.EventKinds()
}

func navivoxCapabilityEndpoints() []capabilityEndpoint {
	return capability.Endpoints()
}
