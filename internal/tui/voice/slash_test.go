package voice

import (
	"reflect"
	"testing"
)

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
