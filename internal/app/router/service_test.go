package router

import "testing"

func TestOpenAIBaseURL(t *testing.T) {
	got := OpenAIBaseURL("127.0.0.1:9898")
	if got != "http://127.0.0.1:9898/v1" {
		t.Fatalf("OpenAIBaseURL = %q", got)
	}
}
