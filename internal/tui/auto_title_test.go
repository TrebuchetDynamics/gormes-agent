package tui

import "testing"

// TestBuildAutoTitleRequest_CompletePromptReturnsRequest covers the happy
// path: a clean completed turn with session_key, non-empty texts, and
// HistoryCount=2 must return ok=true and preserve the original bytes.
func TestBuildAutoTitleRequest_CompletePromptReturnsRequest(t *testing.T) {
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

// TestBuildAutoTitleRequest_FallbackSessionID covers session ID resolution:
// an empty SessionKey with FallbackSessionID="sid" must resolve to "sid".
func TestBuildAutoTitleRequest_FallbackSessionID(t *testing.T) {
	in := AutoTitleInput{
		SessionKey:        "",
		FallbackSessionID: "sid",
		Status:            "complete",
		UserText:          "u",
		AssistantText:     "a",
		HistoryCount:      2,
	}

	got, ok := BuildAutoTitleRequest(in)
	if !ok {
		t.Fatalf("BuildAutoTitleRequest(%+v) ok = false; want true", in)
	}
	if got.SessionID != "sid" {
		t.Errorf("SessionID = %q; want %q", got.SessionID, "sid")
	}
}

// TestBuildAutoTitleRequest_FallbackSessionID_WhitespaceSessionKey covers
// the trimming rule: a whitespace-only SessionKey must fall back to
// FallbackSessionID rather than count as a present session ID.
func TestBuildAutoTitleRequest_FallbackSessionID_WhitespaceSessionKey(t *testing.T) {
	in := AutoTitleInput{
		SessionKey:        "   \t\n",
		FallbackSessionID: "sid",
		Status:            "complete",
		UserText:          "u",
		AssistantText:     "a",
	}

	got, ok := BuildAutoTitleRequest(in)
	if !ok {
		t.Fatalf("BuildAutoTitleRequest(%+v) ok = false; want true", in)
	}
	if got.SessionID != "sid" {
		t.Errorf("SessionID = %q; want %q (whitespace SessionKey must fall back)",
			got.SessionID, "sid")
	}
}

// TestBuildAutoTitleRequest_SkipsInterrupted asserts that Interrupted=true
// vetoes eligibility regardless of otherwise-clean inputs.
func TestBuildAutoTitleRequest_SkipsInterrupted(t *testing.T) {
	in := AutoTitleInput{
		SessionKey:    "session-key-1",
		Status:        "complete",
		UserText:      "user prompt",
		AssistantText: "assistant reply",
		Interrupted:   true,
		HistoryCount:  2,
	}

	if got, ok := BuildAutoTitleRequest(in); ok {
		t.Fatalf("BuildAutoTitleRequest(interrupted) = (%+v, true); want ok=false", got)
	}
}

// TestBuildAutoTitleRequest_SkipsEmptyPromptOrResponse asserts that
// whitespace-only UserText or AssistantText vetoes eligibility.
func TestBuildAutoTitleRequest_SkipsEmptyPromptOrResponse(t *testing.T) {
	cases := []struct {
		name string
		in   AutoTitleInput
	}{
		{
			name: "empty user text",
			in: AutoTitleInput{
				SessionKey:    "session-key-1",
				Status:        "complete",
				UserText:      "",
				AssistantText: "assistant reply",
			},
		},
		{
			name: "whitespace-only user text",
			in: AutoTitleInput{
				SessionKey:    "session-key-1",
				Status:        "complete",
				UserText:      "  \t\n  ",
				AssistantText: "assistant reply",
			},
		},
		{
			name: "empty assistant text",
			in: AutoTitleInput{
				SessionKey:    "session-key-1",
				Status:        "complete",
				UserText:      "user prompt",
				AssistantText: "",
			},
		},
		{
			name: "whitespace-only assistant text",
			in: AutoTitleInput{
				SessionKey:    "session-key-1",
				Status:        "complete",
				UserText:      "user prompt",
				AssistantText: " \n\t ",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := BuildAutoTitleRequest(tc.in); ok {
				t.Fatalf("BuildAutoTitleRequest(%+v) = (%+v, true); want ok=false",
					tc.in, got)
			}
		})
	}
}

// TestBuildAutoTitleRequest_SkipsNonCompleteOrMissingSession asserts that a
// non-"complete" status or an empty resolved session ID vetoes eligibility.
func TestBuildAutoTitleRequest_SkipsNonCompleteOrMissingSession(t *testing.T) {
	cases := []struct {
		name string
		in   AutoTitleInput
	}{
		{
			name: "empty status",
			in: AutoTitleInput{
				SessionKey:    "session-key-1",
				Status:        "",
				UserText:      "user prompt",
				AssistantText: "assistant reply",
			},
		},
		{
			name: "in_progress status",
			in: AutoTitleInput{
				SessionKey:    "session-key-1",
				Status:        "in_progress",
				UserText:      "user prompt",
				AssistantText: "assistant reply",
			},
		},
		{
			name: "Complete with capital C is not complete",
			in: AutoTitleInput{
				SessionKey:    "session-key-1",
				Status:        "Complete",
				UserText:      "user prompt",
				AssistantText: "assistant reply",
			},
		},
		{
			name: "missing session id (both empty)",
			in: AutoTitleInput{
				SessionKey:        "",
				FallbackSessionID: "",
				Status:            "complete",
				UserText:          "user prompt",
				AssistantText:     "assistant reply",
			},
		},
		{
			name: "whitespace-only session id (both whitespace)",
			in: AutoTitleInput{
				SessionKey:        "   ",
				FallbackSessionID: "\t\n",
				Status:            "complete",
				UserText:          "user prompt",
				AssistantText:     "assistant reply",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := BuildAutoTitleRequest(tc.in); ok {
				t.Fatalf("BuildAutoTitleRequest(%+v) = (%+v, true); want ok=false",
					tc.in, got)
			}
		})
	}
}
