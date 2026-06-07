package reloadskills

import (
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

func TestRenderReplyReportsDegradedScanAndUnknownChannel(t *testing.T) {
	got := RenderReply(ReplyRequest{
		ScanError: "scan denied",
		Refreshes: []RefreshResult{{Error: "cache denied"}},
	})
	gatewaytest.AssertContainsAll(t, got, "Skills reload degraded", "skill scan: scan denied", "unknown: refresh error: cache denied")
}
