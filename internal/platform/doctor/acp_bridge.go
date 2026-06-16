package doctor

import (
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/protocols/acp"
)

// CheckACPBridgeStatus renders the local ACP bridge readiness check for the
// doctor report. The Go runtime currently supports local stdio JSON-RPC server
// and client surfaces; remote ACP endpoints remain explicitly unsupported.
func CheckACPBridgeStatus() CheckResult {
	status := acp.DefaultBridgeStatus()
	checkStatus := StatusPass
	if !status.ServerReady || !status.ClientReady || status.RemoteStatus != acp.BridgeEndpointReady {
		checkStatus = StatusWarn
	}

	serverStatus := StatusPass
	if !status.ServerReady {
		serverStatus = StatusWarn
	}
	clientStatus := StatusPass
	if !status.ClientReady {
		clientStatus = StatusWarn
	}
	remoteStatus := StatusPass
	if status.RemoteStatus != acp.BridgeEndpointReady {
		remoteStatus = StatusWarn
	}

	return CheckResult{
		Name:   "ACP bridge",
		Status: checkStatus,
		Summary: fmt.Sprintf("server=%s client=%s remote=%s evidence=%s",
			acpBridgeReadyWord(status.ServerReady),
			acpBridgeReadyWord(status.ClientReady),
			status.RemoteStatus,
			status.RemoteEvidence,
		),
		Items: []ItemInfo{
			{
				Name:   "server",
				Status: serverStatus,
				Note:   fmt.Sprintf("evidence=%s surfaces=%d row_backed=%d", status.ServerEvidence, status.ServerSurfaces, status.ServerRowBacked),
			},
			{
				Name:   "client",
				Status: clientStatus,
				Note:   "evidence=" + status.ClientEvidence,
			},
			{
				Name:   "remote",
				Status: remoteStatus,
				Note:   fmt.Sprintf("evidence=%s reason=%s", status.RemoteEvidence, status.RemoteReason),
			},
		},
	}
}

func acpBridgeReadyWord(ok bool) string {
	if ok {
		return "ready"
	}
	return "unavailable"
}
