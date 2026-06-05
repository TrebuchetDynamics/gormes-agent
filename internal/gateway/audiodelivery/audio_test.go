package audiodelivery

import "testing"

func TestRequestsAudioReply(t *testing.T) {
	if !RequestsAudioReply("", []string{"voice"}) {
		t.Fatal("voice attachment did not request audio reply")
	}
	if !RequestsAudioReply("I cannot read right now, send audio too", nil) {
		t.Fatal("cannot-read text did not request audio reply")
	}
	if !RequestsAudioReply("Mandamelo en audio, por favor", nil) {
		t.Fatal("Spanish audio request did not request audio reply")
	}
	if RequestsAudioReply("plain written answer is fine", nil) {
		t.Fatal("plain text unexpectedly requested audio reply")
	}
}

func TestAppendGuidance(t *testing.T) {
	if got := AppendGuidance("base", false); got != "base" {
		t.Fatalf("disabled guidance = %q, want base", got)
	}
	if got := AppendGuidance("", true); got != Guidance {
		t.Fatalf("empty guidance = %q, want Guidance", got)
	}
	want := "base\n\n" + Guidance
	if got := AppendGuidance("base\n", true); got != want {
		t.Fatalf("appended guidance = %q, want %q", got, want)
	}
}
