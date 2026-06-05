package tools

import "testing"

func TestBrowserMCPLoginFlowFacade_BuildAuthorizeURL(t *testing.T) {
	flow := NewBrowserMCPLoginFlow(MCPBrowserLoginOptions{State: "state-1"})
	server := MCPServerDefinition{Name: "acme", Enabled: true, Transport: MCPTransportHTTP, URL: "https://mcp.example/oauth"}

	launchURL, redirectURI, err := flow.BuildAuthorizeURL(server)
	if err != nil {
		t.Fatalf("BuildAuthorizeURL: %v", err)
	}
	if launchURL == "" || redirectURI == "" {
		t.Fatalf("BuildAuthorizeURL returned launch=%q redirect=%q", launchURL, redirectURI)
	}
}
