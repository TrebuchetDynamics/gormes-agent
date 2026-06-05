package plugins

import (
	"slices"
	"testing"
)

// TestGoogleMeetPluginMetadata is the parity fixture for the first-party
// Google Meet plugin. Every sub-test stays inert: no Chrome, no audio device,
// no OpenAI Realtime call, no node websocket, no Python import, no meeting.
func TestGoogleMeetPluginMetadata(t *testing.T) {
	t.Run("LoadsManifest", testGoogleMeetPluginMetadataLoadsManifest)
	t.Run("ToolInventory", testGoogleMeetPluginMetadataToolInventory)
	t.Run("RealtimeAndNodeCapabilities", testGoogleMeetPluginMetadataRealtimeAndNodeCapabilities)
	t.Run("SafetyEvidence", testGoogleMeetPluginMetadataSafetyEvidence)
}

func googleMeetPluginFixture(t *testing.T) string {
	t.Helper()
	return writePluginFixture(t, "google_meet", map[string]string{
		"plugin.yaml": GoogleMeetPluginYAML,
		"__init__.py": GoogleMeetPluginInitPy,
		"tools.py":    GoogleMeetPluginToolsPy,
	})
}

func loadGoogleMeetForTest(t *testing.T) PluginStatus {
	t.Helper()
	dir := googleMeetPluginFixture(t)
	status := LoadGoogleMeet(dir, LoadOptions{
		Source:               SourceBundled,
		CurrentGormesVersion: "1.0.0",
		EnvLookup:            func(string) bool { return false },
		AuthLookup:           func(string) bool { return false },
	})
	if status.RuntimeCodeExecuted {
		t.Fatal("Google Meet plugin metadata load executed runtime code")
	}
	return status
}

func testGoogleMeetPluginMetadataLoadsManifest(t *testing.T) {
	status := loadGoogleMeetForTest(t)

	if status.State != StateDisabled {
		t.Fatalf("state = %q, want disabled; evidence=%+v", status.State, status.Evidence)
	}
	if status.Manifest.Name != "google_meet" {
		t.Fatalf("manifest name = %q, want google_meet", status.Manifest.Name)
	}
	if status.Manifest.Version != "0.2.0" {
		t.Fatalf("manifest version = %q, want 0.2.0", status.Manifest.Version)
	}
	if status.Manifest.Kind != "standalone" {
		t.Fatalf("manifest kind = %q, want standalone", status.Manifest.Kind)
	}
	if status.Manifest.Author != "NousResearch" {
		t.Fatalf("manifest author = %q, want NousResearch", status.Manifest.Author)
	}
	if !slices.Equal(status.Manifest.Platforms, []string{"linux", "macos"}) {
		t.Fatalf("manifest platforms = %#v, want [linux macos]", status.Manifest.Platforms)
	}
	assertEvidence(t, status.Evidence, EvidenceExecutionDisabled, "runtime")
	assertEvidence(t, status.Evidence, EvidenceGoogleMeetRuntimeUnavailable, "playwright")
	assertEvidence(t, status.Evidence, EvidenceBrowserProfileRequired, "chromium")
}

func testGoogleMeetPluginMetadataToolInventory(t *testing.T) {
	status := loadGoogleMeetForTest(t)

	wantTools := []string{"meet_join", "meet_leave", "meet_say", "meet_status", "meet_transcript"}
	if got := toolMetadataNames(status.Tools); !slices.Equal(got, wantTools) {
		t.Fatalf("tool metadata names = %#v, want %#v", got, wantTools)
	}
	if got := capabilityNames(status.Capabilities, CapabilityTool); !slices.Equal(got, wantTools) {
		t.Fatalf("tool capabilities = %#v, want %#v", got, wantTools)
	}
	for _, name := range wantTools {
		capability := findCapability(status.Capabilities, CapabilityTool, name)
		if capability == nil {
			t.Fatalf("missing capability for %s", name)
		}
		if capability.State != StateDisabled {
			t.Fatalf("%s state = %q, want disabled", name, capability.State)
		}
		assertEvidence(t, capability.Evidence, EvidenceExecutionDisabled, "runtime")
		assertEvidence(t, capability.Evidence, EvidenceGoogleMeetRuntimeUnavailable, "playwright")

		meta := findToolMetadata(status.Tools, name)
		if meta == nil {
			t.Fatalf("missing tool metadata for %s", name)
		}
		if meta.Toolset != "google_meet" {
			t.Fatalf("%s toolset = %q, want google_meet", name, meta.Toolset)
		}
		if meta.ResultEnvelope.Encoding != "json-string" {
			t.Fatalf("%s result envelope encoding = %q, want json-string", name, meta.ResultEnvelope.Encoding)
		}
		if !slices.Contains(meta.ResultEnvelope.ErrorFields, "error") {
			t.Fatalf("%s result envelope error fields = %#v, want error", name, meta.ResultEnvelope.ErrorFields)
		}
	}

	join := requireToolMetadata(t, status.Tools, "meet_join")
	assertSchemaRequired(t, join.Schema, "url")
	assertSchemaProperty(t, join.Schema, "url", "string")
	assertSchemaEnumValue(t, join.Schema, "mode", "transcribe")
	assertSchemaEnumValue(t, join.Schema, "mode", "realtime")
	assertSchemaProperty(t, join.Schema, "node", "string")
	assertSchemaProperty(t, join.Schema, "guest_name", "string")

	say := requireToolMetadata(t, status.Tools, "meet_say")
	assertSchemaRequired(t, say.Schema, "text")
	assertSchemaProperty(t, say.Schema, "text", "string")
}

func testGoogleMeetPluginMetadataRealtimeAndNodeCapabilities(t *testing.T) {
	status := loadGoogleMeetForTest(t)

	realtime := findCapability(status.Capabilities, CapabilityRealtime, "google_meet_realtime_audio")
	if realtime == nil {
		t.Fatalf("missing realtime capability in %+v", capabilityNamesAll(status.Capabilities))
	}
	if realtime.State != StateDisabled {
		t.Fatalf("realtime capability state = %q, want disabled", realtime.State)
	}
	assertEvidence(t, realtime.Evidence, EvidenceGoogleMeetRealtimeUnconfigured, "OPENAI_API_KEY")
	assertEvidence(t, realtime.Evidence, EvidenceGoogleMeetRealtimeUnconfigured, "audio_bridge")

	node := findCapability(status.Capabilities, CapabilityRemoteNode, "google_meet_node")
	if node == nil {
		t.Fatalf("missing remote-node capability in %+v", capabilityNamesAll(status.Capabilities))
	}
	if node.State != StateDisabled {
		t.Fatalf("node capability state = %q, want disabled", node.State)
	}
	assertEvidence(t, node.Evidence, EvidenceNodeAuthRequired, "bearer_token")
	assertEvidence(t, node.Evidence, EvidenceExecutionDisabled, "runtime")

	assertEvidence(t, status.Evidence, EvidenceGoogleMeetRealtimeUnconfigured, "OPENAI_API_KEY")
	assertEvidence(t, status.Evidence, EvidenceNodeAuthRequired, "bearer_token")
}

func testGoogleMeetPluginMetadataSafetyEvidence(t *testing.T) {
	status := loadGoogleMeetForTest(t)

	assertEvidence(t, status.Evidence, EvidenceMeetURLGate, "url")
	assertEvidence(t, status.Evidence, EvidenceNoCalendarAutoDial, "calendar")
	assertEvidence(t, status.Evidence, EvidenceOneActiveMeeting, "meet_join")

	say := findCapability(status.Capabilities, CapabilityTool, "meet_say")
	if say == nil {
		t.Fatal("meet_say capability missing for safety evidence assertion")
	}
	assertEvidence(t, say.Evidence, EvidenceMeetSayRealtimeRequired, "mode")
}

func capabilityNamesAll(capabilities []CapabilityStatus) []string {
	out := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		out = append(out, string(capability.Kind)+":"+capability.Name)
	}
	return out
}
