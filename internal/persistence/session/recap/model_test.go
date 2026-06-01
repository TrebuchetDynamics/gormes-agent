package recap

import (
	"strings"
	"testing"
)

func TestEnvelopeHumanOutput(t *testing.T) {
	envelope := &Envelope{
		Entries:       []Entry{{SessionID: "sess-1", Title: "Title", Source: "cli", UpdatedAt: 1700000000, TokensIn: 3, TokensOut: 5}},
		TotalSessions: 2,
		Truncated:     true,
	}
	got := envelope.HumanOutput()
	for _, want := range []string{"sess-1", "Title", "tokens: 3 in / 5 out", "1 more sessions not shown"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Envelope.HumanOutput() missing %q in %q", want, got)
		}
	}
}

func TestSessionResultHumanOutputNotFound(t *testing.T) {
	got := (&SessionResult{SessionID: "missing", NotFound: true}).HumanOutput()
	if !strings.Contains(got, "missing") || !strings.Contains(got, "not found") {
		t.Fatalf("SessionResult.HumanOutput() = %q, want not-found evidence", got)
	}
}
