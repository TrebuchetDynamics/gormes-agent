package transcript

import (
	"errors"
	"testing"
)

func TestCompareGoldenJSONReportsExactFieldPath(t *testing.T) {
	want := []byte(`{"provider_requests":[{"messages":[{"role":"user","content":"hi"}]}]}`)
	got := []byte(`{"provider_requests":[{"messages":[{"role":"user","content":"bye"}]}]}`)

	err := CompareGoldenJSON(want, got)
	if err == nil {
		t.Fatal("CompareGoldenJSON returned nil, want mismatch")
	}
	var diff JSONDiff
	if !errors.As(err, &diff) {
		t.Fatalf("mismatch error type = %T, want JSONDiff", err)
	}
	if diff.Path != "$.provider_requests[0].messages[0].content" {
		t.Fatalf("diff path = %q, want provider message content path", diff.Path)
	}
}

func TestCompareGoldenJSONIgnoresObjectKeyOrder(t *testing.T) {
	want := []byte(`{"status":{"phase":"Idle","history_length":2},"fixture":"text_only"}`)
	got := []byte(`{"fixture":"text_only","status":{"history_length":2,"phase":"Idle"}}`)

	if err := CompareGoldenJSON(want, got); err != nil {
		t.Fatalf("CompareGoldenJSON: %v", err)
	}
}
