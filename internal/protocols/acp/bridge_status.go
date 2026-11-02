package acp

const (
	BridgeServerEvidenceReady = "acp_stdio_jsonrpc_ready"
	BridgeRemoteEvidence      = "acp_bridge_unavailable"
)

type BridgeEndpointStatus string

const (
	BridgeEndpointReady       BridgeEndpointStatus = "ready"
	BridgeEndpointUnavailable BridgeEndpointStatus = "unavailable"
	BridgeEndpointUnsupported BridgeEndpointStatus = "unsupported"
)

type BridgeStatus struct {
	ServerReady       bool
	ServerEvidence    string
	ServerSurfaces    int
	ServerImplemented int
	ServerRowBacked   int
	ClientReady       bool
	ClientEvidence    string
	RemoteStatus      BridgeEndpointStatus
	RemoteEvidence    string
	RemoteReason      string
}

func DefaultBridgeStatus() BridgeStatus {
	manifest := DefaultServerManifest()
	implemented, rowBacked := 0, 0
	for _, surface := range manifest.Surfaces {
		switch surface.Status {
		case ServerSurfaceStatusImplemented:
			implemented++
		case ServerSurfaceStatusRowBacked:
			rowBacked++
		}
	}
	return BridgeStatus{
		ServerReady:       true,
		ServerEvidence:    BridgeServerEvidenceReady,
		ServerSurfaces:    len(manifest.Surfaces),
		ServerImplemented: implemented,
		ServerRowBacked:   rowBacked,
		ClientReady:       true,
		ClientEvidence:    ClientEvidenceConnected,
		RemoteStatus:      BridgeEndpointUnsupported,
		RemoteEvidence:    BridgeRemoteEvidence,
		RemoteReason:      "unsupported_remote_acp_endpoint",
	}
}
