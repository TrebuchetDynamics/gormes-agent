package title

import (
	"errors"
	"testing"
)

func TestSlashArgCollapsesTitleArgument(t *testing.T) {
	got, ok := SlashArg("/title my   session title")
	if !ok || got != "my session title" {
		t.Fatalf("SlashArg set = (%q, %v), want collapsed title", got, ok)
	}

	got, ok = SlashArg("/title")
	if ok || got != "" {
		t.Fatalf("SlashArg query = (%q, %v), want no argument", got, ok)
	}

	got, ok = SlashArg("   ")
	if ok || got != "" {
		t.Fatalf("SlashArg empty = (%q, %v), want no argument", got, ok)
	}
}

func TestHandleSlashGetsSetsAndValidatesTitle(t *testing.T) {
	var calls []struct{ sessionID, title string }
	fn := func(sessionID, nextTitle string) (SessionTitleResult, error) {
		calls = append(calls, struct{ sessionID, title string }{sessionID: sessionID, title: nextTitle})
		if nextTitle == "" {
			return SessionTitleResult{Title: "current title"}, nil
		}
		return SessionTitleResult{Title: nextTitle, Pending: true}, nil
	}

	got := HandleSlash(SlashRequest{Input: "/title", SessionID: " sess-title ", TitleFunc: fn})
	if got.StatusMessage != "title: current title" {
		t.Fatalf("query status = %q", got.StatusMessage)
	}
	got = HandleSlash(SlashRequest{Input: "/title my   title", SessionID: "sess-title", TitleFunc: fn})
	if got.StatusMessage != "session title set: my title (queued while session initializes)" {
		t.Fatalf("set status = %q", got.StatusMessage)
	}
	if len(calls) != 2 || calls[0].sessionID != "sess-title" || calls[0].title != "" || calls[1].title != "my title" {
		t.Fatalf("calls = %+v", calls)
	}

	got = HandleSlash(SlashRequest{Input: "/title x", TitleFunc: fn})
	if got.StatusMessage != "no active session" {
		t.Fatalf("missing session status = %q", got.StatusMessage)
	}
	got = HandleSlash(SlashRequest{Input: "/title x", SessionID: "sess-title"})
	if got.StatusMessage != "title: session title unavailable" {
		t.Fatalf("missing adapter status = %q", got.StatusMessage)
	}
}

func TestHandleSlashReportsAdapterErrorsAndEmptyTitles(t *testing.T) {
	got := HandleSlash(SlashRequest{
		Input:     "/title",
		SessionID: "sess-title",
		TitleFunc: func(string, string) (SessionTitleResult, error) {
			return SessionTitleResult{}, nil
		},
	})
	if got.StatusMessage != "no title set" {
		t.Fatalf("empty current title status = %q", got.StatusMessage)
	}

	got = HandleSlash(SlashRequest{
		Input:     "/title new",
		SessionID: "sess-title",
		TitleFunc: func(string, string) (SessionTitleResult, error) {
			return SessionTitleResult{}, errors.New("boom")
		},
	})
	if got.StatusMessage != "title: boom" {
		t.Fatalf("error status = %q", got.StatusMessage)
	}
}
