package voice

import (
	"errors"
	"reflect"
	"testing"
)

func TestHandleSlashCallsToggleAndBuildsStatus(t *testing.T) {
	var got Request
	result := HandleSlash("/voice on", "sess-1", func(req Request) (Result, error) {
		got = req
		return Result{Enabled: true, RecordKey: "alt+r"}, nil
	})
	if !reflect.DeepEqual(got, Request{Action: "on", SessionID: "sess-1"}) {
		t.Fatalf("toggle request = %#v, want action/session", got)
	}
	if !result.UpdateRecordKey || result.RecordKey != "alt+r" {
		t.Fatalf("record-key update = (%v, %q), want alt+r", result.UpdateRecordKey, result.RecordKey)
	}
	if result.StatusMessage != "Voice mode enabled" {
		t.Fatalf("StatusMessage = %q, want first rendered line", result.StatusMessage)
	}
}

func TestHandleSlashUnavailableAndError(t *testing.T) {
	unavailable := HandleSlash("/voice status", "sess-1", nil)
	if unavailable.StatusMessage != "Voice Mode Status" {
		t.Fatalf("unavailable StatusMessage = %q, want status page title", unavailable.StatusMessage)
	}
	if unavailable.UpdateRecordKey {
		t.Fatal("nil toggle should not update record key")
	}
	errored := HandleSlash("/voice status", "sess-1", func(Request) (Result, error) {
		return Result{}, errors.New("mic unavailable")
	})
	if errored.Err == nil || errored.StatusMessage != "voice: mic unavailable" {
		t.Fatalf("error result = (%v, %q), want voice error", errored.Err, errored.StatusMessage)
	}
}

func TestAction(t *testing.T) {
	cases := map[string]string{
		"/voice":        "status",
		"/voice on":     "on",
		"/voice off":    "off",
		"/voice tts":    "tts",
		"/voice status": "status",
		"/voice nope":   "status",
	}
	for input, want := range cases {
		if got := Action(input); got != want {
			t.Fatalf("Action(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestLines(t *testing.T) {
	if got := Lines("tts", Result{TTS: true}); !reflect.DeepEqual(got, []string{"Voice TTS enabled."}) {
		t.Fatalf("Lines tts = %v", got)
	}
	got := Lines("status", Result{Enabled: true, Details: "install sox\nconfigure mic"})
	want := []string{"Voice Mode Status", "  Mode:       ON", "  TTS:        OFF", "  Record key: Ctrl+B", "", "  Requirements:", "    install sox", "    configure mic"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Lines status = %#v, want %#v", got, want)
	}
}
