package acp

import (
	"strings"
)

// TerminalSetupAuthMethodID is the ACP registry-compatible terminal setup
// method. Hermes uses this exact id, so Gormes keeps it as a compatibility
// contract even though the setup command is Gormes-owned.
const TerminalSetupAuthMethodID = "hermes-setup"

// AuthMethod is one ACP auth method advertised during initialize.
type AuthMethod struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        string   `json:"type,omitempty"`
	Args        []string `json:"args,omitempty"`
}

// DetectProvider resolves the active runtime provider only when a usable
// credential is also present. A provider without a credential must not be
// advertised, or the ACP auth handshake rejects the editor.
func DetectProvider(provider, apiKey string) (string, bool) {
	name := strings.ToLower(strings.TrimSpace(provider))
	if name == "" {
		return "", false
	}
	if strings.TrimSpace(apiKey) == "" {
		return "", false
	}
	return name, true
}

// BuildAuthMethods returns the ACP auth methods advertised during initialize.
// The terminal setup method is always present so a fresh install can configure
// credentials. The provider method appears only when credentials resolve.
func BuildAuthMethods(provider string, credentialAvailable bool) []AuthMethod {
	methods := make([]AuthMethod, 0, 2)
	name := strings.ToLower(strings.TrimSpace(provider))
	if name != "" && credentialAvailable {
		methods = append(methods, AuthMethod{
			ID:          name,
			Name:        name + " runtime credentials",
			Description: "Authenticate Gormes using the currently configured " + name + " runtime credentials.",
		})
	}
	methods = append(methods, AuthMethod{
		ID:          TerminalSetupAuthMethodID,
		Name:        "Configure Gormes provider",
		Type:        "terminal",
		Args:        []string{"--setup"},
		Description: "Open Gormes' interactive model/provider setup in a terminal. Use this when Gormes has not been configured on this machine yet.",
	})
	return methods
}
