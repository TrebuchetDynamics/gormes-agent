package acp

import "testing"

func TestACPBridgeStatusDistinguishesLocalAndRemoteSupport(t *testing.T) {
	status := DefaultBridgeStatus()

	if !status.ServerReady {
		t.Fatalf("ServerReady = false, want local ACP stdio server ready")
	}
	if status.ServerEvidence != "acp_stdio_jsonrpc_ready" {
		t.Fatalf("ServerEvidence = %q, want acp_stdio_jsonrpc_ready", status.ServerEvidence)
	}
	if status.ServerSurfaces == 0 || status.ServerRowBacked != status.ServerSurfaces {
		t.Fatalf("server surfaces = %d row_backed=%d, want all surfaces row-backed", status.ServerSurfaces, status.ServerRowBacked)
	}
	if !status.ClientReady {
		t.Fatalf("ClientReady = false, want local ACP client bridge ready")
	}
	if status.ClientEvidence != ClientEvidenceConnected {
		t.Fatalf("ClientEvidence = %q, want %q", status.ClientEvidence, ClientEvidenceConnected)
	}
	if status.RemoteStatus != BridgeEndpointUnsupported {
		t.Fatalf("RemoteStatus = %q, want %q", status.RemoteStatus, BridgeEndpointUnsupported)
	}
	if status.RemoteEvidence != "acp_bridge_unavailable" {
		t.Fatalf("RemoteEvidence = %q, want acp_bridge_unavailable", status.RemoteEvidence)
	}
	if status.RemoteReason == "" {
		t.Fatalf("RemoteReason is empty, want explicit unsupported remote endpoint evidence")
	}
}
