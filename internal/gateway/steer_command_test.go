package gateway

import "testing"

func TestSteerCommandCompatibilityWrapper(t *testing.T) {
	got := ParseSteerCommand("/steer keep investigating", SteerPayloadMetadata{})
	if got.Guidance != "keep investigating" || got.Preview != "keep investigating" || got.Evidence != "" {
		t.Fatalf("ParseSteerCommand wrapper = %+v, want parsed guidance", got)
	}
	if got := SteerPreview("keep investigating"); got != "keep investigating" {
		t.Fatalf("SteerPreview wrapper = %q, want passthrough preview", got)
	}
}
