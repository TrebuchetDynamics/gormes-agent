package navivox

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	descriptorSecretParamPattern = regexp.MustCompile(`(?i)(^|[?&\s;])(?:rest_token|token|pairing_token)=[^\s&;]+`)
	descriptorSecretJSONPattern  = regexp.MustCompile(`(?i)("(?:rest_token|token|pairing_token)"\s*:\s*")[^"]*(")`)
	tokenValuePattern            = regexp.MustCompile(`nvbx_[A-Za-z0-9._~+\-/%=]*`)
)

// SharePayload converts a navivox://connect descriptor query into the compact
// JSON payload used by Android SEND fallback. Invalid descriptors pass through.
func SharePayload(descriptor string) string {
	parsed, err := url.Parse(descriptor)
	if err != nil {
		return descriptor
	}
	values := parsed.Query()
	if len(values) == 0 {
		return descriptor
	}
	payload := make(map[string]any, len(values))
	for key, vals := range values {
		if len(vals) == 0 {
			continue
		}
		value := vals[0]
		if shareBoolKey(key) {
			payload[key] = strings.EqualFold(value, "true")
			continue
		}
		payload[key] = value
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return descriptor
	}
	return string(raw)
}

func shareBoolKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "bridge_keepalive_required", "setup_handoff", "token_required":
		return true
	default:
		return false
	}
}

// Redact removes Navivox token material from command output and Android errors.
func Redact(text string) string {
	if text == "" {
		return ""
	}
	redacted := descriptorSecretParamPattern.ReplaceAllString(text, "${1}[redacted]")
	redacted = descriptorSecretJSONPattern.ReplaceAllString(redacted, "${1}[redacted]${2}")
	redacted = tokenValuePattern.ReplaceAllString(redacted, "[redacted]")
	return redacted
}

// PairDescriptor builds the one-terminal setup handoff descriptor.
func PairDescriptor(authMode, exposureMode, token, baseURL, wsURL string) string {
	values := url.Values{}
	values.Set("base_url", baseURL)
	values.Set("websocket_url", wsURL)
	values.Set("status_url", strings.TrimRight(baseURL, "/")+"/v1/navivox/status")
	values.Set("capabilities_url", strings.TrimRight(baseURL, "/")+"/v1/navivox/capabilities")
	values.Set("setup_handoff", "true")
	values.Set("setup_mutation_policy", "read_only_handoff")
	values.Set("setup_sections", "provider,model,workspace,channels")
	values.Set("setup_entry_screen", "setup.provider")
	values.Set("bridge_keepalive_required", "true")
	values.Set("bridge_lifecycle", "termux_pair_command")
	values.Set("recommended_path", "navivox")
	values.Set("pairing_token_temporary", "true")
	values.Set("pairing_token_expires_when", "bridge_stops")
	values.Set("pairing_device_limit", "1")
	values.Set("auth_mode", authMode)
	values.Set("exposure_mode", exposureMode)
	values.Set("token_required", "true")
	values.Set("rest_token", token)
	return (&url.URL{Scheme: "navivox", Host: "connect", RawQuery: values.Encode()}).String()
}

// ConnectDescriptor builds a Navivox import descriptor for an existing gateway endpoint.
func ConnectDescriptor(baseURL, websocketURL, capabilitiesURL, authMode, exposureMode, token string, tokenRequired bool) (string, error) {
	values := url.Values{}
	values.Set("base_url", baseURL)
	values.Set("websocket_url", websocketURL)
	values.Set("capabilities_url", capabilitiesURL)
	values.Set("auth_mode", strings.TrimSpace(authMode))
	values.Set("exposure_mode", strings.TrimSpace(exposureMode))
	values.Set("token_required", strconv.FormatBool(tokenRequired))
	if tokenRequired {
		token = strings.TrimSpace(token)
		if token == "" {
			return "", fmt.Errorf("navivox connect: token auth selected but token is empty")
		}
		values.Set("rest_token", token)
	}
	return (&url.URL{Scheme: "navivox", Host: "connect", RawQuery: values.Encode()}).String(), nil
}
