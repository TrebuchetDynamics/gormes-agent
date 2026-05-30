package skin

import (
	"errors"
	"testing"
)

func TestHandleSlashStatusAndApply(t *testing.T) {
	var got Request
	status := HandleSlash("/skin", "sess-skin", func(req Request) (Result, error) {
		got = req
		return Result{Name: "poseidon"}, nil
	})
	if got != (Request{SessionID: "sess-skin"}) {
		t.Fatalf("status request = %#v, want empty name/session", got)
	}
	if status.Apply || status.Body != "skin: poseidon" || status.StatusMessage != "skin: poseidon" {
		t.Fatalf("status result = %#v, want display-only poseidon", status)
	}
	apply := HandleSlash("/skin dracula", "sess-skin", func(req Request) (Result, error) {
		got = req
		return Result{Name: "dark"}, nil
	})
	if got != (Request{Name: "dracula", SessionID: "sess-skin"}) {
		t.Fatalf("apply request = %#v, want requested name/session", got)
	}
	if !apply.Apply || apply.AcceptedName != "dark" {
		t.Fatalf("apply result = %#v, want accepted dark", apply)
	}
}

func TestHandleSlashUnavailableAndError(t *testing.T) {
	if got := HandleSlash("/skin", "sess", nil).StatusMessage; got != "skin: configuration unavailable" {
		t.Fatalf("nil config status = %q, want unavailable", got)
	}
	errored := HandleSlash("/skin", "sess", func(Request) (Result, error) {
		return Result{}, errors.New("config locked")
	})
	if errored.Err == nil || errored.StatusMessage != "skin: config locked" {
		t.Fatalf("error result = %#v, want config error", errored)
	}
}

func TestSlashName(t *testing.T) {
	if got := SlashName("/skin ocean blue"); got != "ocean blue" {
		t.Fatalf("SlashName = %q, want full arg tail", got)
	}
	if got := SlashName("/skin"); got != "" {
		t.Fatalf("SlashName without arg = %q, want empty", got)
	}
}

func TestDisplayName(t *testing.T) {
	if got := DisplayName("  "); got != "default" {
		t.Fatalf("DisplayName empty = %q, want default", got)
	}
	if got := DisplayName(" poseidon "); got != "poseidon" {
		t.Fatalf("DisplayName trimmed = %q", got)
	}
}
