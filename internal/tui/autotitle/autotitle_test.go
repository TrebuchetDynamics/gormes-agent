package autotitle

import "testing"

// TestBuildRequestCompletePromptReturnsRequest covers the happy path: a clean
// completed turn with session_key, non-empty texts, and HistoryCount=2 must
// return ok=true and preserve the original bytes.
func TestBuildRequestCompletePromptReturnsRequest(t *testing.T) {
	in := Input{
		SessionKey:    "session-key-1",
		Status:        "complete",
		UserText:      "  hello there  ",
		AssistantText: "  general kenobi  ",
		HistoryCount:  2,
	}

	got, ok := BuildRequest(in)
	if !ok {
		t.Fatalf("BuildRequest(%+v) ok = false; want true", in)
	}
	if got.SessionID != "session-key-1" {
		t.Errorf("SessionID = %q; want %q", got.SessionID, "session-key-1")
	}
	if got.UserText != in.UserText {
		t.Errorf("UserText = %q; want original %q", got.UserText, in.UserText)
	}
	if got.AssistantText != in.AssistantText {
		t.Errorf("AssistantText = %q; want original %q", got.AssistantText, in.AssistantText)
	}
	if got.HistoryCount != 2 {
		t.Errorf("HistoryCount = %d; want 2", got.HistoryCount)
	}
}

// TestBuildRequestFallbackSessionID covers session ID resolution: an empty
// SessionKey with FallbackSessionID="sid" must resolve to "sid".
func TestBuildRequestFallbackSessionID(t *testing.T) {
	in := Input{
		SessionKey:        "",
		FallbackSessionID: "sid",
		Status:            "complete",
		UserText:          "u",
		AssistantText:     "a",
		HistoryCount:      2,
	}

	got, ok := BuildRequest(in)
	if !ok {
		t.Fatalf("BuildRequest(%+v) ok = false; want true", in)
	}
	if got.SessionID != "sid" {
		t.Errorf("SessionID = %q; want %q", got.SessionID, "sid")
	}
}

// TestBuildRequestFallbackSessionIDWhitespaceSessionKey covers the trimming
// rule: a whitespace-only SessionKey must fall back to FallbackSessionID.
func TestBuildRequestFallbackSessionIDWhitespaceSessionKey(t *testing.T) {
	in := Input{
		SessionKey:        "   \t\n",
		FallbackSessionID: "sid",
		Status:            "complete",
		UserText:          "u",
		AssistantText:     "a",
	}

	got, ok := BuildRequest(in)
	if !ok {
		t.Fatalf("BuildRequest(%+v) ok = false; want true", in)
	}
	if got.SessionID != "sid" {
		t.Errorf("SessionID = %q; want %q (whitespace SessionKey must fall back)", got.SessionID, "sid")
	}
}

// TestBuildRequestSkipsInterrupted asserts that Interrupted=true vetoes
// eligibility regardless of otherwise-clean inputs.
func TestBuildRequestSkipsInterrupted(t *testing.T) {
	in := Input{
		SessionKey:    "session-key-1",
		Status:        "complete",
		UserText:      "user prompt",
		AssistantText: "assistant reply",
		Interrupted:   true,
		HistoryCount:  2,
	}

	if got, ok := BuildRequest(in); ok {
		t.Fatalf("BuildRequest(interrupted) = (%+v, true); want ok=false", got)
	}
}

// TestBuildRequestSkipsEmptyPromptOrResponse asserts that whitespace-only
// UserText or AssistantText vetoes eligibility.
func TestBuildRequestSkipsEmptyPromptOrResponse(t *testing.T) {
	cases := []struct {
		name string
		in   Input
	}{
		{name: "empty user text", in: Input{SessionKey: "session-key-1", Status: "complete", UserText: "", AssistantText: "assistant reply"}},
		{name: "whitespace-only user text", in: Input{SessionKey: "session-key-1", Status: "complete", UserText: "  \t\n  ", AssistantText: "assistant reply"}},
		{name: "empty assistant text", in: Input{SessionKey: "session-key-1", Status: "complete", UserText: "user prompt", AssistantText: ""}},
		{name: "whitespace-only assistant text", in: Input{SessionKey: "session-key-1", Status: "complete", UserText: "user prompt", AssistantText: " \n\t "}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := BuildRequest(tc.in); ok {
				t.Fatalf("BuildRequest(%+v) = (%+v, true); want ok=false", tc.in, got)
			}
		})
	}
}

// TestBuildRequestSkipsNonCompleteOrMissingSession asserts that a
// non-"complete" status or an empty resolved session ID vetoes eligibility.
func TestBuildRequestSkipsNonCompleteOrMissingSession(t *testing.T) {
	cases := []struct {
		name string
		in   Input
	}{
		{name: "empty status", in: Input{SessionKey: "session-key-1", Status: "", UserText: "user prompt", AssistantText: "assistant reply"}},
		{name: "in_progress status", in: Input{SessionKey: "session-key-1", Status: "in_progress", UserText: "user prompt", AssistantText: "assistant reply"}},
		{name: "Complete with capital C is not complete", in: Input{SessionKey: "session-key-1", Status: "Complete", UserText: "user prompt", AssistantText: "assistant reply"}},
		{name: "missing session id both empty", in: Input{SessionKey: "", FallbackSessionID: "", Status: "complete", UserText: "user prompt", AssistantText: "assistant reply"}},
		{name: "whitespace-only session id", in: Input{SessionKey: "   ", FallbackSessionID: "\t\n", Status: "complete", UserText: "user prompt", AssistantText: "assistant reply"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := BuildRequest(tc.in); ok {
				t.Fatalf("BuildRequest(%+v) = (%+v, true); want ok=false", tc.in, got)
			}
		})
	}
}
