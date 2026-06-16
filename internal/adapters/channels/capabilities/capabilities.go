// Package capabilities contains adapter-neutral channel metadata helpers.
package capabilities

import (
	"fmt"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/channelutil"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// CapabilityOptions configures a read-only channel capability report.
type CapabilityOptions struct {
	// Configured maps channel IDs to redacted config details.
	Configured map[string]string
	// Channel optionally filters the report to one channel ID.
	Channel string
}

// CapabilitySupport mirrors the manifest surface statuses that operators need
// when deciding whether a channel supports a workflow.
type CapabilitySupport struct {
	Inbound  string `json:"inbound"`
	Outbound string `json:"outbound"`
	Media    string `json:"media"`
	Commands string `json:"commands"`
	Toolset  string `json:"toolset"`
	Config   string `json:"config"`
	Pairing  string `json:"pairing"`
	Delivery string `json:"delivery"`
}

// CapabilityReport is a redacted, live-SDK-free channel capability row.
type CapabilityReport struct {
	Channel           string            `json:"channel"`
	DisplayName       string            `json:"display_name"`
	Kind              string            `json:"kind"`
	Implementation    string            `json:"implementation"`
	Configured        bool              `json:"configured"`
	ConfigDetail      string            `json:"config_detail,omitempty"`
	Support           CapabilitySupport `json:"support"`
	Intents           []string          `json:"intents"`
	Scopes            []string          `json:"scopes"`
	Features          []string          `json:"features"`
	FormatLimitations []string          `json:"format_limitations,omitempty"`
	Degraded          []string          `json:"degraded,omitempty"`
	BacklogOwner      string            `json:"backlog_owner,omitempty"`
	Notes             string            `json:"notes,omitempty"`
}

// UnknownChannelError reports an operator-requested channel that is absent
// from the source-backed platform manifest.
type UnknownChannelError struct {
	Channel string
}

func (e UnknownChannelError) Error() string {
	return fmt.Sprintf("unknown_channel: %s", e.Channel)
}

// BuildCapabilityReports returns source-backed channel capability metadata in
// stable order. It deliberately reads only the checked-in gateway manifest and
// redacted configured-channel names supplied by the caller.
func BuildCapabilityReports(opts CapabilityOptions) ([]CapabilityReport, error) {
	filter := normalizeID(opts.Channel)
	configured := normalizeConfigured(opts.Configured)
	reports := []CapabilityReport{}

	for _, entry := range gateway.OperatorPlatformManifest() {
		if entry.Kind != gateway.PlatformKindChannel {
			continue
		}
		id := normalizeID(entry.ID)
		if filter != "" && id != filter {
			continue
		}
		reports = append(reports, buildCapabilityReport(entry, configured[id]))
	}
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Channel < reports[j].Channel
	})
	if filter != "" && len(reports) == 0 {
		return nil, UnknownChannelError{Channel: filter}
	}
	return reports, nil
}

func buildCapabilityReport(entry gateway.PlatformManifestEntry, configDetail string) CapabilityReport {
	support := CapabilitySupport{
		Inbound:  string(entry.Inbound),
		Outbound: string(entry.Outbound),
		Media:    string(entry.Media),
		Commands: string(entry.Commands),
		Toolset:  string(entry.Toolset),
		Config:   string(entry.Config),
		Pairing:  string(entry.Pairing),
		Delivery: string(entry.Delivery),
	}
	report := CapabilityReport{
		Channel:        normalizeID(entry.ID),
		DisplayName:    strings.TrimSpace(entry.DisplayName),
		Kind:           string(entry.Kind),
		Implementation: string(entry.Status),
		Configured:     strings.TrimSpace(configDetail) != "",
		ConfigDetail:   strings.TrimSpace(configDetail),
		Support:        support,
		Intents:        capabilityIntents(entry),
		Scopes:         capabilityScopes(entry),
		Features:       capabilityFeatures(support),
		BacklogOwner:   strings.TrimSpace(entry.BacklogOwner),
		Notes:          strings.TrimSpace(entry.Notes),
	}
	report.FormatLimitations = capabilityLimitations(support)
	if !report.Configured {
		report.Degraded = append(report.Degraded, "not_configured")
	}
	return report
}

func capabilityIntents(entry gateway.PlatformManifestEntry) []string {
	intents := []string{}
	if hasSurface(entry.Inbound) {
		intents = append(intents, "receive")
	}
	if hasSurface(entry.Outbound) || hasSurface(entry.Delivery) {
		intents = append(intents, "send")
	}
	if hasSurface(entry.Commands) {
		intents = append(intents, "native_commands")
	}
	if hasSurface(entry.Media) {
		intents = append(intents, "media")
	}
	if hasSurface(entry.Toolset) {
		intents = append(intents, "tools")
	}
	if hasSurface(entry.Pairing) {
		intents = append(intents, "pairing")
	}
	return uniqueSorted(intents)
}

func capabilityScopes(entry gateway.PlatformManifestEntry) []string {
	scopes := []string{
		"channel:" + normalizeID(entry.ID),
		"kind:" + string(entry.Kind),
	}
	if entry.RequiresLiveCredentials {
		scopes = append(scopes, "credentials:required")
	} else {
		scopes = append(scopes, "credentials:not_required")
	}
	if owner := strings.TrimSpace(entry.BacklogOwner); owner != "" {
		scopes = append(scopes, "backlog:"+owner)
	}
	return scopes
}

func capabilityFeatures(support CapabilitySupport) []string {
	features := []string{
		"inbound=" + support.Inbound,
		"outbound=" + support.Outbound,
		"media=" + support.Media,
		"commands=" + support.Commands,
		"toolset=" + support.Toolset,
		"config=" + support.Config,
		"pairing=" + support.Pairing,
		"delivery=" + support.Delivery,
	}
	return features
}

func capabilityLimitations(support CapabilitySupport) []string {
	values := []struct {
		name   string
		status string
	}{
		{"media", support.Media},
		{"commands", support.Commands},
		{"toolset", support.Toolset},
		{"pairing", support.Pairing},
		{"delivery", support.Delivery},
	}
	limitations := []string{}
	for _, value := range values {
		switch value.status {
		case string(gateway.PlatformSurfacePartial), string(gateway.PlatformSurfaceRowBacked):
			limitations = append(limitations, value.name+"="+value.status)
		}
	}
	return limitations
}

func hasSurface(status gateway.PlatformSurfaceStatus) bool {
	switch status {
	case gateway.PlatformSurfaceNotApplicable:
		return false
	default:
		return strings.TrimSpace(string(status)) != ""
	}
}

func normalizeConfigured(configured map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range configured {
		id := normalizeID(key)
		detail := strings.TrimSpace(value)
		if id != "" && detail != "" {
			out[id] = detail
		}
	}
	return out
}

func normalizeID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func uniqueSorted(values []string) []string { return channelutil.UniqueSortedStrings(values) }
