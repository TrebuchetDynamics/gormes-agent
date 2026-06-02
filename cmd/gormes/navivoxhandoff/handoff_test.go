package navivoxhandoff

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

func TestPairDescriptorBuildsSetupHandoffDescriptor(t *testing.T) {
	descriptor := PairDescriptor("pairing_token", "local", "nvbx_test", "http://127.0.0.1:8765", "ws://127.0.0.1:8765/v1/navivox/stream")
	parsed, err := url.Parse(descriptor)
	if err != nil {
		t.Fatalf("parse descriptor: %v", err)
	}
	if parsed.Scheme != "navivox" || parsed.Host != "connect" {
		t.Fatalf("target = %s://%s", parsed.Scheme, parsed.Host)
	}
	values := parsed.Query()
	for key, want := range map[string]string{
		"base_url":                  "http://127.0.0.1:8765",
		"websocket_url":             "ws://127.0.0.1:8765/v1/navivox/stream",
		"status_url":                "http://127.0.0.1:8765/v1/navivox/status",
		"capabilities_url":          "http://127.0.0.1:8765/v1/navivox/capabilities",
		"setup_handoff":             "true",
		"bridge_keepalive_required": "true",
		"auth_mode":                 "pairing_token",
		"exposure_mode":             "local",
		"rest_token":                "nvbx_test",
	} {
		if got := values.Get(key); got != want {
			t.Fatalf("%s = %q, want %q in %s", key, got, want, descriptor)
		}
	}
}

func TestSharePayloadConvertsDescriptorQueryToJSON(t *testing.T) {
	payload := SharePayload("navivox://connect?base_url=http%3A%2F%2F127.0.0.1%3A8765&setup_handoff=true&token_required=true&rest_token=nvbx_share")
	var got map[string]any
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatalf("payload is not JSON: %v: %s", err, payload)
	}
	if got["base_url"] != "http://127.0.0.1:8765" || got["setup_handoff"] != true || got["token_required"] != true || got["rest_token"] != "nvbx_share" {
		t.Fatalf("payload = %#v", got)
	}
}

func TestRedactRemovesDescriptorSecrets(t *testing.T) {
	redacted := Redact("navivox://connect?rest_token=nvbx_secret&token=raw-token pairing_token=nvbx_pair raw nvbx_extra")
	for _, forbidden := range []string{"nvbx_secret", "nvbx_pair", "nvbx_extra", "rest_token=", "token=raw-token", "pairing_token="} {
		if strings.Contains(redacted, forbidden) {
			t.Fatalf("redaction leaked %q in %q", forbidden, redacted)
		}
	}
	if !strings.Contains(redacted, "[redacted]") {
		t.Fatalf("redaction marker missing: %q", redacted)
	}
}
