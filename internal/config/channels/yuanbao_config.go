package channels

import "github.com/TrebuchetDynamics/gormes-agent/internal/config/channels/yuanbao"

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
	return c.asYuanbaoCfg().RuntimeEnabled()
}

// MissingCredentials lists the credential field names the runtime still needs
// to start. The order is stable so doctor/gateway status output stays
// deterministic.
func (c YuanbaoCfg) MissingCredentials() []string {
	return c.asYuanbaoCfg().MissingCredentials()
}

// RedactedStatus returns a single-line status descriptor with every credential
// and session field replaced by a presence boolean. The shape is shared by
// gateway status and doctor renderers.
func (c YuanbaoCfg) RedactedStatus() string {
	return c.asYuanbaoCfg().RedactedStatus()
}

func (c YuanbaoCfg) asYuanbaoCfg() yuanbao.Cfg {
	return yuanbao.Cfg{
		Enabled:               c.Enabled,
		LoginToken:            c.LoginToken,
		HySource:              c.HySource,
		AgentID:               c.AgentID,
		AllowedConversationID: c.AllowedConversationID,
		CoalesceMs:            c.CoalesceMs,
		FirstRunDiscovery:     c.FirstRunDiscovery,
	}
}
