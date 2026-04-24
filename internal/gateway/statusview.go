package gateway

import (
	"fmt"
	"sort"
	"strings"
)

// ConfiguredChannel describes one channel as seen by the loaded TOML config
// before any transport starts. It is the read-only shape consumed by
// RenderStatusSummary and the `gormes gateway status` command, so callers
// can assemble a view without depending on adapter packages.
//
// Platform is the channel name ("telegram", "discord", ...).
// AllowedChatID carries the platform's allowlist identifier (chat_id for
// Telegram, channel_id for Discord). Empty string means no allowlist.
// FirstRunDiscovery mirrors the config flag of the same name.
type ConfiguredChannel struct {
	Platform          string
	AllowedChatID     string
	FirstRunDiscovery bool
}

// ConfiguredChannels bundles the per-platform configuration (if any) plus a
// pointer to the in-process runtime status model. A nil StatusModel is
// treated as "no runtime data yet" — the typical case for the read-only
// `gormes gateway status` command invoked without a live gateway.
type ConfiguredChannels struct {
	Telegram *ConfiguredChannel
	Discord  *ConfiguredChannel

	// Runtime, when non-nil, is queried for per-platform lifecycle data.
	// The manager's in-process StatusModel lives under
	// internal/gateway/status.go; on-disk pairing.json / gateway_state.json
	// persistence planned under 2.F.3 can later be merged into this struct
	// without changing the RenderStatusSummary signature.
	Runtime *StatusModel
}

// RenderStatusSummary produces a stable, human-readable multi-line summary
// of the configured channels plus any runtime lifecycle data available from
// the in-process StatusModel. The output is deterministic (platform rows are
// alphabetically ordered) so operators and snapshot tests can rely on it.
//
// Empty input returns a single "no channels configured" line. Channels that
// are configured but lack a runtime status entry render with
// `lifecycle=unknown` and `paired=unknown` — the planned pairing.json read
// model will later fill these in without widening this signature.
func RenderStatusSummary(in ConfiguredChannels) string {
	var b strings.Builder
	b.WriteString("gateway status\n")

	rows := collectChannelRows(in)
	if len(rows) == 0 {
		b.WriteString("  no channels configured (set [telegram] or [discord] in config.toml)\n")
		return b.String()
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].Platform < rows[j].Platform })
	for _, row := range rows {
		b.WriteString(renderChannelRow(row, in.Runtime))
	}
	return b.String()
}

func collectChannelRows(in ConfiguredChannels) []ConfiguredChannel {
	var rows []ConfiguredChannel
	if in.Telegram != nil {
		row := *in.Telegram
		if row.Platform == "" {
			row.Platform = "telegram"
		}
		rows = append(rows, row)
	}
	if in.Discord != nil {
		row := *in.Discord
		if row.Platform == "" {
			row.Platform = "discord"
		}
		rows = append(rows, row)
	}
	return rows
}

func renderChannelRow(c ConfiguredChannel, runtime *StatusModel) string {
	idField := "allowed_chat_id"
	if c.Platform == "discord" {
		idField = "allowed_channel_id"
	}
	idValue := c.AllowedChatID
	if idValue == "" {
		idValue = "(none)"
	}

	lifecycle := "unknown"
	lastErr := ""
	paired := "unknown"
	if runtime != nil {
		if entry, ok := runtime.Lookup(c.Platform); ok {
			lifecycle = string(entry.Phase)
			lastErr = entry.LastError
			paired = pairedStateFromLifecycle(entry.Phase)
		}
	}

	line := fmt.Sprintf(
		"  %s: configured %s=%s first_run_discovery=%t lifecycle=%s paired=%s",
		c.Platform,
		idField,
		idValue,
		c.FirstRunDiscovery,
		lifecycle,
		paired,
	)
	if lastErr != "" {
		line += fmt.Sprintf(" last_error=%q", lastErr)
	}
	return line + "\n"
}

// pairedStateFromLifecycle maps the in-process lifecycle phase to a coarse
// paired/unpaired answer for operators. Later on-disk pairing.json slices
// will replace this with authoritative paired state; until then, anything
// past "registered" is treated as "in-session" (effectively paired for the
// duration of the live gateway process).
func pairedStateFromLifecycle(phase LifecyclePhase) string {
	switch phase {
	case LifecyclePhaseRunning:
		return "in-session"
	case LifecyclePhaseDisconnected, LifecyclePhaseFailed:
		return "unpaired"
	case LifecyclePhaseRegistered:
		return "pending"
	default:
		return "unknown"
	}
}
