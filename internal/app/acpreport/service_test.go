package acpreport

import (
	"bytes"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/protocols/acp"
)

func TestWriteBrowserBootstrapTextIncludesStepsAndEvidence(t *testing.T) {
	var out bytes.Buffer
	WriteBrowserBootstrapText(&out, acp.BrowserBootstrapReport{
		OK:         true,
		DryRun:     true,
		Platform:   "linux/amd64",
		NodePrefix: "/tmp/node",
		Evidence:   acp.ClientEvidence{Code: "acp_setup_browser_plan"},
		Steps: []acp.BrowserBootstrapStep{{
			Name:    "agent-browser",
			Status:  "planned",
			Command: []string{"npm", "install", "agent-browser@^0.26.0"},
			Message: "ready",
		}},
		Message: "dry run only",
	})

	got := out.String()
	for _, want := range []string{
		"ACP browser bootstrap dry-run",
		"evidence: acp_setup_browser_plan",
		"platform: linux/amd64",
		"node_prefix: /tmp/node",
		"- agent-browser: planned `npm install agent-browser@^0.26.0` — ready",
		"message: dry run only",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func TestWriteClientTextIncludesDegradedEvidence(t *testing.T) {
	var out bytes.Buffer
	WriteClientText(&out, acp.ClientResult{
		OK:             false,
		SessionKey:     "agent:missing:main",
		SessionLabel:   "missing",
		ProvenanceMode: acp.ProvenanceMeta,
		Evidence: acp.ClientEvidence{
			Code:   acp.ClientEvidenceRowBacked,
			Reason: "session_key_not_found",
		},
		Message: "session not found",
	})

	got := out.String()
	for _, want := range []string{
		"ACP client degraded",
		"session_key: agent:missing:main",
		"session_label: missing",
		"provenance: meta",
		"evidence: acp_client_row_backed",
		"reason: session_key_not_found",
		"message: session not found",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}
