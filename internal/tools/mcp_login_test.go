package tools

import (
	"context"
	"testing"
)

func TestRunMCPLoginFacade_Noninteractive(t *testing.T) {
	server := MCPServerDefinition{Name: "acme", Enabled: true, Transport: MCPTransportHTTP, URL: "https://mcp.example/oauth"}
	result, err := RunMCPLogin(context.Background(), MCPConfigResolution{Servers: []MCPServerDefinition{server}}, NewMCPOAuthStore(), NoninteractiveLoginFlow(), "acme")
	if err != nil {
		t.Fatalf("RunMCPLogin returned error: %v", err)
	}
	if result.Evidence != MCPLoginEvidenceNoninteractiveRequired {
		t.Fatalf("Evidence = %q, want noninteractive_required", result.Evidence)
	}
}
