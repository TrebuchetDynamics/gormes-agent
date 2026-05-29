package channels

import (
	"fmt"
	"strings"
)

// YuanbaoCfg drives the disabled-by-default Yuanbao gateway adapter. Live
// websocket/QR-login behavior is deferred; the runtime slice binds fake
// clients only and treats every credential field as a redacted secret in
// status output.
type YuanbaoCfg struct {
	Enabled               bool   `toml:"enabled"`
	LoginToken            string `toml:"login_token"`
	HySource              string `toml:"hy_source"`
	AgentID               string `toml:"agent_id"`
	AllowedConversationID string `toml:"allowed_conversation_id"`
	CoalesceMs            int    `toml:"coalesce_ms"`
	FirstRunDiscovery     bool   `toml:"first_run_discovery"`
}

// RuntimeEnabled reports whether the adapter has both the operator opt-in and
// every credential the fake/live transports require. Missing pieces keep the
// adapter degraded instead of starting an unauthenticated session.
func (c YuanbaoCfg) RuntimeEnabled() bool {
	if !c.Enabled {
		return false
	}
	return strings.TrimSpace(c.LoginToken) != "" &&
		strings.TrimSpace(c.HySource) != "" &&
		strings.TrimSpace(c.AgentID) != ""
}

// MissingCredentials lists the credential field names the runtime still needs
// to start. The order is stable so doctor/gateway status output stays
// deterministic.
func (c YuanbaoCfg) MissingCredentials() []string {
	missing := []string{}
	if strings.TrimSpace(c.LoginToken) == "" {
		missing = append(missing, "login_token")
	}
	if strings.TrimSpace(c.HySource) == "" {
		missing = append(missing, "hy_source")
	}
	if strings.TrimSpace(c.AgentID) == "" {
		missing = append(missing, "agent_id")
	}
	return missing
}

// RedactedStatus returns a single-line status descriptor with every credential
// and session field replaced by a presence boolean. The shape is shared by
// gateway status and doctor renderers.
func (c YuanbaoCfg) RedactedStatus() string {
	parts := []string{
		"yuanbao",
		fmt.Sprintf("enabled=%t", c.Enabled),
		fmt.Sprintf("login_token_set=%t", strings.TrimSpace(c.LoginToken) != ""),
		fmt.Sprintf("hy_source_set=%t", strings.TrimSpace(c.HySource) != ""),
		fmt.Sprintf("agent_id_set=%t", strings.TrimSpace(c.AgentID) != ""),
	}
	if conv := strings.TrimSpace(c.AllowedConversationID); conv != "" {
		parts = append(parts, "allowed_conversation_id="+conv)
	} else {
		parts = append(parts, fmt.Sprintf("first_run_discovery=%t", c.FirstRunDiscovery))
	}
	return strings.Join(parts, " ")
}
