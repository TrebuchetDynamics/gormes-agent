package main

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/doctor"
)

func TestDoctorACPBridgeStatusRendersLocalReadyAndRemoteUnavailable(t *testing.T) {
	result := doctorACPBridgeStatus()

	if result.Name != "ACP bridge" {
		t.Fatalf("Name = %q, want ACP bridge", result.Name)
	}
	if result.Status != doctor.StatusWarn {
		t.Fatalf("Status = %v, want WARN for unsupported remote endpoint", result.Status)
	}
	if len(result.Items) != 3 {
		t.Fatalf("items = %d, want server, client, and remote", len(result.Items))
	}

	out := result.Format()
	for _, want := range []string{
		"[WARN] ACP bridge:",
		"server=ready",
		"client=ready",
		"remote=unsupported",
		"acp_stdio_jsonrpc_ready",
		"acp_client_connected",
		"acp_bridge_unavailable",
		"unsupported_remote_acp_endpoint",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("doctor ACP output missing %q:\n%s", want, out)
		}
	}
}
