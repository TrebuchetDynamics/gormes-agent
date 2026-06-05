package tui

import "testing"

func TestBuildAutoTitleRequestCompatibilityWrapper(t *testing.T) {
	in := AutoTitleInput{
		SessionKey:    "session-key-1",
		Status:        "complete",
		UserText:      "  hello there  ",
		AssistantText: "  general kenobi  ",
		HistoryCount:  2,
	}

	got, ok := BuildAutoTitleRequest(in)
	if !ok {
		t.Fatalf("BuildAutoTitleRequest(%+v) ok = false; want true", in)
	}
	if got.SessionID != "session-key-1" || got.UserText != in.UserText || got.AssistantText != in.AssistantText || got.HistoryCount != 2 {
		t.Fatalf("BuildAutoTitleRequest(%+v) = (%+v, true); wrapper did not preserve autotitle request", in, got)
	}
}
