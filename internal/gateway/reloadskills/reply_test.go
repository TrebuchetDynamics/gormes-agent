package reloadskills

import (
	"strings"
	"testing"
)

func TestRenderReplyReportsSuccessAndAdapterRefreshes(t *testing.T) {
	got := RenderReply(ReplyRequest{
		SkillCount: 2,
		Refreshes:  []RefreshResult{{Channel: " discord ", Count: 3, Hidden: 1}},
	})
	for _, want := range []string{"Skills Reloaded", "2 skill(s) available", "discord: refreshed 3 command(s), 1 hidden"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderReply() = %q, missing %q", got, want)
		}
	}
}

func TestRenderReplyReportsDegradedScanAndUnknownChannel(t *testing.T) {
	got := RenderReply(ReplyRequest{
		ScanError: "scan denied",
		Refreshes: []RefreshResult{{Error: "cache denied"}},
	})
	for _, want := range []string{"Skills reload degraded", "skill scan: scan denied", "unknown: refresh error: cache denied"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderReply() = %q, missing %q", got, want)
		}
	}
}
