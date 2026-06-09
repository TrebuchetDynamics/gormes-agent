package reloadskills

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/gatewaytest"
)

func TestRenderReplyReportsSuccessAndAdapterRefreshes(t *testing.T) {
	got := RenderReply(ReplyRequest{
		SkillCount: 2,
		Refreshes:  []RefreshResult{{Channel: " discord ", Count: 3, Hidden: 1}},
	})
	gatewaytest.AssertContainsAll(t, got, "Skills Reloaded", "2 skill(s) available", "discord: refreshed 3 command(s), 1 hidden")
}

func TestRenderReplySanitizesOperatorFields(t *testing.T) {
	got := RenderReply(ReplyRequest{
		ScanError: "scan failed\n**Injected scan:** `token`",
		Refreshes: []RefreshResult{{Channel: "discord\n**Injected channel:**", Error: "cache denied\n**Injected error:**"}},
	})
	for _, forbidden := range []string{"**Injected scan:**", "**Injected channel:**", "**Injected error:**", "`token`", "discord\n"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("RenderReply leaked operator injection %q in:\n%s", forbidden, got)
		}
	}
	gatewaytest.AssertContainsAll(t, got,
		"skill scan: scan failed ''Injected scan:'' 'token'",
		"discord ''Injected channel:'': refresh error: cache denied ''Injected error:''",
	)
}

func TestRenderReplyRedactsSecretBearingErrors(t *testing.T) {
	got := RenderReply(ReplyRequest{
		ScanError: "scan failed\n**Injected:** api key plain-secret",
		Refreshes: []RefreshResult{{Channel: "telegram", Error: "refresh failed\n**Injected:** bearer other-secret"}},
	})
	for _, forbidden := range []string{"plain-secret", "other-secret", "**Injected:**", "scan failed", "refresh failed"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("RenderReply leaked secret-bearing error %q in:\n%s", forbidden, got)
		}
	}
	gatewaytest.AssertContainsAll(t, got, "skill scan: [redacted]", "telegram: refresh error: [redacted]")
}

func TestRenderReplyClampsNegativeRefreshCounts(t *testing.T) {
	got := RenderReply(ReplyRequest{
		SkillCount: -7,
		Refreshes:  []RefreshResult{{Channel: "discord", Count: -3, Hidden: -2}},
	})
	for _, forbidden := range []string{"-7 skill", "-3 command", "-2 hidden"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("RenderReply leaked impossible negative count %q in:\n%s", forbidden, got)
		}
	}
	gatewaytest.AssertContainsAll(t, got, "0 skill(s) available", "discord: refreshed 0 command(s), 0 hidden")
}

func TestRenderReplyReportsDegradedScanAndUnknownChannel(t *testing.T) {
	got := RenderReply(ReplyRequest{
		ScanError: "scan denied",
		Refreshes: []RefreshResult{{Error: "cache denied"}},
	})
	gatewaytest.AssertContainsAll(t, got, "Skills reload degraded", "skill scan: scan denied", "unknown: refresh error: cache denied")
}
