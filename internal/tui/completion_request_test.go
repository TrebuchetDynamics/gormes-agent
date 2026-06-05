package tui

import "testing"

func TestTUICompletionRequestCompatibilityWrapper(t *testing.T) {
	got, ok := CompletionRequestForInput("/help")
	if !ok {
		t.Fatal("CompletionRequestForInput(/help) ok = false, want true")
	}
	want := TUICompletionRequest{Method: TUICompletionSlash, Text: "/help", ReplaceFrom: 1}
	if got != want {
		t.Fatalf("CompletionRequestForInput(/help) = %+v, want %+v", got, want)
	}
}
