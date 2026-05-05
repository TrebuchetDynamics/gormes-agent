package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

const SecretRuntimeEvidenceIgnoredInactiveSurface = "secret_ref_ignored_inactive_surface"

type SecretStringResolver interface {
	ResolveString(config.SecretRef) (string, config.SecretRefEvidence, error)
}

type GatewaySecretRuntimeOptions struct {
	Resolver SecretStringResolver
}

type GatewaySecretActivation struct {
	Config      config.Config         `json:"-"`
	Snapshot    SecretRuntimeSnapshot `json:"snapshot"`
	Diagnostics []SecretRuntimeEntry  `json:"diagnostics,omitempty"`
	Redacted    bool                  `json:"redacted"`
}

type SecretRuntimeSnapshot struct {
	Generation int                           `json:"generation,omitempty"`
	Entries    map[string]SecretRuntimeEntry `json:"entries"`
	Redacted   bool                          `json:"redacted"`
}

func (s SecretRuntimeSnapshot) String() string {
	body, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"redacted":true,"marshal_error":%q}`, err.Error())
	}
	return string(body)
}

type SecretRuntimeEntry struct {
	Path     string                   `json:"path"`
	Ref      config.SecretRef         `json:"ref"`
	Active   bool                     `json:"active"`
	Resolved bool                     `json:"resolved"`
	Evidence config.SecretRefEvidence `json:"evidence"`
	Reason   string                   `json:"reason,omitempty"`
	Redacted bool                     `json:"redacted"`
}

type gatewaySecretTarget struct {
	path   string
	ref    *config.SecretRef
	active bool
	reason string
	apply  func(*config.Config, string)
}

type GatewaySecretRuntimeController struct {
	mu         sync.Mutex
	opts       GatewaySecretRuntimeOptions
	generation int
	hasLast    bool
	last       GatewaySecretActivation
}

func NewGatewaySecretRuntimeController(opts GatewaySecretRuntimeOptions) *GatewaySecretRuntimeController {
	return &GatewaySecretRuntimeController{opts: opts}
}

func (c *GatewaySecretRuntimeController) Activate(ctx context.Context, cfg config.Config) (GatewaySecretActivation, error) {
	return c.activate(ctx, cfg)
}

func (c *GatewaySecretRuntimeController) Reload(ctx context.Context, cfg config.Config) (GatewaySecretActivation, error) {
	activation, err := c.activate(ctx, cfg)
	if err == nil || !c.hasLastActivation() {
		return activation, err
	}
	return c.LastActivation(), err
}

func (c *GatewaySecretRuntimeController) Snapshot() SecretRuntimeSnapshot {
	return c.LastActivation().Snapshot
}

func (c *GatewaySecretRuntimeController) LastActivation() GatewaySecretActivation {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.hasLast {
		return GatewaySecretActivation{
			Snapshot: SecretRuntimeSnapshot{
				Entries:  map[string]SecretRuntimeEntry{},
				Redacted: true,
			},
			Redacted: true,
		}
	}
	return cloneGatewaySecretActivation(c.last)
}

func (c *GatewaySecretRuntimeController) hasLastActivation() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hasLast
}

func (c *GatewaySecretRuntimeController) activate(ctx context.Context, cfg config.Config) (GatewaySecretActivation, error) {
	activation, err := ActivateGatewaySecretRefs(ctx, cfg, c.opts)
	if err != nil {
		return activation, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.generation++
	activation.Snapshot.Generation = c.generation
	c.last = cloneGatewaySecretActivation(activation)
	c.hasLast = true
	return cloneGatewaySecretActivation(activation), nil
}

func ActivateGatewaySecretRefs(ctx context.Context, cfg config.Config, opts GatewaySecretRuntimeOptions) (GatewaySecretActivation, error) {
	resolver := opts.Resolver
	if resolver == nil {
		resolver = config.NewSecretResolver(config.SecretResolverConfig{Secrets: cfg.Secrets})
	}

	activated := cfg
	snapshot := SecretRuntimeSnapshot{
		Entries:  map[string]SecretRuntimeEntry{},
		Redacted: true,
	}
	activation := GatewaySecretActivation{
		Config:   activated,
		Snapshot: snapshot,
		Redacted: true,
	}

	var failures []SecretRuntimeEntry
	for _, target := range gatewaySecretTargets(cfg) {
		if target.ref == nil {
			continue
		}
		if err := ctx.Err(); err != nil {
			return activation, fmt.Errorf("secret runtime activation canceled: %w", err)
		}
		ref := normalizeGatewaySecretRef(*target.ref)
		entry := SecretRuntimeEntry{
			Path:     target.path,
			Ref:      ref,
			Active:   target.active,
			Redacted: true,
		}

		if !target.active {
			entry.Evidence = config.SecretRefEvidence{
				Code:     SecretRuntimeEvidenceIgnoredInactiveSurface,
				Source:   string(ref.Source),
				Provider: ref.Provider,
				ID:       ref.ID,
				Redacted: true,
			}
			entry.Reason = target.reason
			activation.Snapshot.Entries[target.path] = entry
			activation.Diagnostics = append(activation.Diagnostics, entry)
			continue
		}

		target.apply(&activated, "")
		value, evidence, err := resolver.ResolveString(ref)
		evidence = normalizeSecretRuntimeEvidence(ref, evidence)
		entry.Evidence = evidence
		entry.Resolved = err == nil
		if err != nil {
			failures = append(failures, entry)
			activation.Snapshot.Entries[target.path] = entry
			continue
		}
		target.apply(&activated, value)
		activation.Snapshot.Entries[target.path] = entry
	}

	activation.Config = activated
	if len(failures) > 0 {
		activation.Diagnostics = append(activation.Diagnostics, failures...)
		return activation, secretRuntimeActivationError{failures: failures}
	}
	return activation, nil
}

func cloneGatewaySecretActivation(activation GatewaySecretActivation) GatewaySecretActivation {
	activation.Snapshot = cloneSecretRuntimeSnapshot(activation.Snapshot)
	activation.Diagnostics = cloneSecretRuntimeEntries(activation.Diagnostics)
	return activation
}

func cloneSecretRuntimeSnapshot(snapshot SecretRuntimeSnapshot) SecretRuntimeSnapshot {
	clone := snapshot
	clone.Entries = make(map[string]SecretRuntimeEntry, len(snapshot.Entries))
	for path, entry := range snapshot.Entries {
		clone.Entries[path] = entry
	}
	return clone
}

func cloneSecretRuntimeEntries(entries []SecretRuntimeEntry) []SecretRuntimeEntry {
	if entries == nil {
		return nil
	}
	clone := make([]SecretRuntimeEntry, len(entries))
	copy(clone, entries)
	return clone
}

func normalizeSecretRuntimeEvidence(ref config.SecretRef, evidence config.SecretRefEvidence) config.SecretRefEvidence {
	if evidence.Source == "" {
		evidence.Source = string(ref.Source)
	}
	if evidence.Provider == "" {
		evidence.Provider = ref.Provider
	}
	if evidence.ID == "" {
		evidence.ID = ref.ID
	}
	evidence.Redacted = true
	return evidence
}

func gatewaySecretTargets(cfg config.Config) []gatewaySecretTarget {
	telegramActive := cfg.Telegram.BotToken != "" || cfg.Telegram.BotTokenRef != nil
	discordCredentialConfigured := cfg.Discord.Token != "" || cfg.Discord.TokenRef != nil
	discordActive := discordCredentialConfigured && (cfg.Discord.AllowedChannelID != "" || cfg.Discord.FirstRunDiscovery)
	slackActive := cfg.Slack.Enabled
	providerActive := telegramActive || discordActive || slackActive

	return []gatewaySecretTarget{
		{
			path:   "hermes.api_key",
			ref:    cfg.Hermes.APIKeyRef,
			active: providerActive,
			reason: inactiveReason(providerActive, "no active gateway channel requires provider credentials"),
			apply: func(c *config.Config, value string) {
				c.Hermes.APIKey = value
			},
		},
		{
			path:   "telegram.bot_token",
			ref:    cfg.Telegram.BotTokenRef,
			active: telegramActive,
			reason: inactiveReason(telegramActive, "telegram gateway is not configured"),
			apply: func(c *config.Config, value string) {
				c.Telegram.BotToken = value
			},
		},
		{
			path:   "discord.token",
			ref:    cfg.Discord.TokenRef,
			active: discordActive,
			reason: inactiveReason(discordActive, "discord gateway has no allowed channel and first-run discovery is disabled"),
			apply: func(c *config.Config, value string) {
				c.Discord.Token = value
			},
		},
		{
			path:   "slack.bot_token",
			ref:    cfg.Slack.BotTokenRef,
			active: slackActive,
			reason: inactiveReason(slackActive, "slack gateway is disabled"),
			apply: func(c *config.Config, value string) {
				c.Slack.BotToken = value
			},
		},
		{
			path:   "slack.app_token",
			ref:    cfg.Slack.AppTokenRef,
			active: slackActive,
			reason: inactiveReason(slackActive, "slack gateway is disabled"),
			apply: func(c *config.Config, value string) {
				c.Slack.AppToken = value
			},
		},
	}
}

func normalizeGatewaySecretRef(ref config.SecretRef) config.SecretRef {
	ref.Source = config.SecretRefSource(strings.ToLower(strings.TrimSpace(string(ref.Source))))
	ref.Provider = strings.TrimSpace(ref.Provider)
	if ref.Provider == "" {
		ref.Provider = config.DefaultSecretProviderAlias
	}
	ref.ID = strings.TrimSpace(ref.ID)
	return ref
}

func inactiveReason(active bool, reason string) string {
	if active {
		return ""
	}
	return reason
}

type secretRuntimeActivationError struct {
	failures []SecretRuntimeEntry
}

func (e secretRuntimeActivationError) Error() string {
	parts := make([]string, 0, len(e.failures))
	for _, failure := range e.failures {
		parts = append(parts, fmt.Sprintf("%s code=%s source=%s provider=%s id=%s",
			failure.Path,
			failure.Evidence.Code,
			failure.Evidence.Source,
			failure.Evidence.Provider,
			failure.Evidence.ID,
		))
	}
	return "secret runtime activation failed: " + strings.Join(parts, "; ")
}
