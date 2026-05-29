package channels

import (
	"errors"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestCapabilitiesDeriveSupportFromGatewayManifest(t *testing.T) {
	reports, err := BuildCapabilityReports(CapabilityOptions{
		Configured: map[string]string{"telegram": "allowed_chat_id=42"},
		Channel:    "telegram",
	})
	if err != nil {
		t.Fatalf("BuildCapabilityReports: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("reports = %d, want 1", len(reports))
	}

	got := reports[0]
	if got.Channel != "telegram" || got.DisplayName != "Telegram" {
		t.Fatalf("channel identity = %q/%q, want telegram/Telegram", got.Channel, got.DisplayName)
	}
	if !got.Configured || got.ConfigDetail != "allowed_chat_id=42" {
		t.Fatalf("configured = %t detail=%q, want configured detail", got.Configured, got.ConfigDetail)
	}
	if got.Support.Media != string(gateway.PlatformSurfacePartial) {
		t.Fatalf("media support = %q, want manifest partial", got.Support.Media)
	}
	for _, want := range []string{"receive", "send", "native_commands", "media"} {
		if !containsString(got.Intents, want) {
			t.Fatalf("intents = %#v, missing %q", got.Intents, want)
		}
	}
	for _, want := range []string{"channel:telegram", "kind:channel", "credentials:required"} {
		if !containsString(got.Scopes, want) {
			t.Fatalf("scopes = %#v, missing %q", got.Scopes, want)
		}
	}
	if !containsString(got.Features, "media=partial") {
		t.Fatalf("features = %#v, want manifest-derived media=partial", got.Features)
	}
	if !containsString(got.FormatLimitations, "media=partial") {
		t.Fatalf("format limitations = %#v, want media=partial", got.FormatLimitations)
	}
}

func TestCapabilitiesListAllManifestChannelsWithUnconfiguredEvidence(t *testing.T) {
	reports, err := BuildCapabilityReports(CapabilityOptions{
		Configured: map[string]string{"slack": "allowed_channel_id=C123"},
	})
	if err != nil {
		t.Fatalf("BuildCapabilityReports: %v", err)
	}
	if len(reports) < 10 {
		t.Fatalf("reports = %d, want manifest channel inventory", len(reports))
	}
	assertSortedCapabilities(t, reports)

	slack := findCapabilityReport(t, reports, "slack")
	if !slack.Configured || slack.ConfigDetail != "allowed_channel_id=C123" {
		t.Fatalf("slack configured=%t detail=%q, want configured detail", slack.Configured, slack.ConfigDetail)
	}
	discord := findCapabilityReport(t, reports, "discord")
	if discord.Configured {
		t.Fatalf("discord configured = true, want false")
	}
	if !containsString(discord.Degraded, "not_configured") {
		t.Fatalf("discord degraded = %#v, want not_configured", discord.Degraded)
	}
}

func TestCapabilitiesUnknownChannelReturnsTypedError(t *testing.T) {
	_, err := BuildCapabilityReports(CapabilityOptions{Channel: "missing"})
	if err == nil {
		t.Fatal("BuildCapabilityReports returned nil error, want unknown channel")
	}
	var unknown UnknownChannelError
	if !errors.As(err, &unknown) {
		t.Fatalf("err = %T %v, want UnknownChannelError", err, err)
	}
	if unknown.Channel != "missing" || !strings.Contains(err.Error(), "unknown_channel") {
		t.Fatalf("unknown error = %+v %q, want channel and evidence", unknown, err.Error())
	}
}

func findCapabilityReport(t *testing.T, reports []CapabilityReport, channel string) CapabilityReport {
	t.Helper()
	for _, report := range reports {
		if report.Channel == channel {
			return report
		}
	}
	t.Fatalf("report for %q not found in %#v", channel, reports)
	return CapabilityReport{}
}

func assertSortedCapabilities(t *testing.T, reports []CapabilityReport) {
	t.Helper()
	for i := 1; i < len(reports); i++ {
		if reports[i-1].Channel > reports[i].Channel {
			t.Fatalf("reports not sorted at %d: %q > %q", i, reports[i-1].Channel, reports[i].Channel)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
