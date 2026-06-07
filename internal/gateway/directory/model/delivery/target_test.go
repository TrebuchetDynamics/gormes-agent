package delivery

import (
	"testing"

	entrymodel "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model/entry"
)

func TestTargetUsesExplicitChatAndThreadIDs(t *testing.T) {
	target := Target("telegram", entrymodel.Entry{ID: "ignored", ChatID: " -100 ", ThreadID: " 12 "})
	if target.Platform != "telegram" || target.ChatID != "-100" || target.ThreadID != "12" || !target.IsExplicit {
		t.Fatalf("Target() = %+v, want explicit trimmed chat/thread target", target)
	}
}

func TestTargetFallsBackToCompositeEntryID(t *testing.T) {
	target := Target("discord", entrymodel.Entry{ID: " C01:99 "})
	if target.Platform != "discord" || target.ChatID != "C01" || target.ThreadID != "99" || !target.IsExplicit {
		t.Fatalf("Target() = %+v, want chat/thread parsed from entry ID", target)
	}
}
