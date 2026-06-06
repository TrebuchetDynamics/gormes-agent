package gateway

import (
	"strings"
	"testing"
)

func TestGatewayStatusCommand_RendersTeamsConfiguredUnavailable(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	writeGatewayStatusConfig(t, []byte(`
[teams]
enabled = true
port = 3978
allowed_users = ["aad-1"]
`))

	stdout, stderr, err := executeGatewayStatusCommand(t)
	if err != nil {
		t.Fatalf("Execute: %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	if !strings.Contains(stdout, "- teams: lifecycle=unknown") ||
		!strings.Contains(stdout, "target=missing_credentials=client_id,client_secret,tenant_id port=3978 allowed_users=1") {
		t.Fatalf("stdout missing Teams configured-unavailable row\n%s", stdout)
	}
}
