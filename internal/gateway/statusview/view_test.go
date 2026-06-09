package statusview

import (
	"reflect"
	"strings"
	"testing"
)

func TestRenderStatusSummary_DeterministicChannelRows(t *testing.T) {
	summary := StatusSummary{
		Channels: []StatusChannel{
			{Name: "telegram", Detail: "allowed_chat_id=42"},
			{Name: "slack", Detail: "allowed_channel_id=C999"},
			{Name: "discord", Detail: "allowed_channel_id=D123"},
		},
		Runtime: RuntimeStatus{
			PID:          4242,
			GatewayState: "running",
			ActiveAgents: 2,
			Platforms: map[string]PlatformRuntimeStatus{
				"telegram": {State: "running"},
				"discord":  {State: "failed", ErrorMessage: "discord: open session: denied"},
				"slack":    {State: "stopped"},
			},
		},
		Pairing: PairingStatus{
			Platforms: []PairingPlatformStatus{
				{Platform: "telegram", State: "paired", PendingCount: 1, ApprovedCount: 1},
				{Platform: "slack", State: "unpaired", PendingCount: 1, ApprovedCount: 0},
				{Platform: "discord", State: "paired", PendingCount: 0, ApprovedCount: 1},
			},
			Pending: []PairingPendingRecord{
				{Platform: "telegram", UserID: "telegram-user", Code: "TGREADY", AgeSeconds: 60},
				{Platform: "slack", UserID: "slack-user", Code: "SLREADY", AgeSeconds: 120},
			},
			Approved: []PairingApprovedRecord{
				{Platform: "telegram", UserID: "telegram-owner"},
				{Platform: "discord", UserID: "discord-owner", UserName: "Grace"},
			},
			Degraded: []PairingDegradedEvidence{
				{Platform: "discord", Reason: "locked_out", Message: "platform locked after repeated invalid pairing approvals"},
			},
		},
	}

	got := RenderStatusSummary(summary)
	want := strings.Join([]string{
		"Gateway status",
		"runtime: running (pid=4242 active_agents=2)",
		"channels:",
		"- discord: lifecycle=failed error=\"discord: open session: denied\"; pairing=paired pending=0 approved=1; target=allowed_channel_id=D123",
		"- slack: lifecycle=stopped; pairing=unpaired pending=1 approved=0; target=allowed_channel_id=C999",
		"- telegram: lifecycle=running; pairing=paired pending=1 approved=1; target=allowed_chat_id=42",
		"pairing:",
		"- pending slack user=slack-user code=SLREADY age=120s",
		"- pending telegram user=telegram-user code=TGREADY age=60s",
		"- approved discord user=discord-owner name=Grace",
		"- approved telegram user=telegram-owner",
		"degraded:",
		"- pairing discord locked_out: platform locked after repeated invalid pairing approvals",
		"",
	}, "\n")
	if got != want {
		t.Fatalf("RenderStatusSummary() mismatch\n--- got ---\n%s--- want ---\n%s", got, want)
	}
}

func TestRenderStatusSummary_NoChannelsAndMissingState(t *testing.T) {
	got := RenderStatusSummary(StatusSummary{
		Pairing: PairingStatus{
			Degraded: []PairingDegradedEvidence{
				{Reason: "missing", Message: "pairing state is missing"},
			},
		},
	})
	want := strings.Join([]string{
		"Gateway status",
		"runtime: missing",
		"channels: none configured",
		"degraded:",
		"- pairing missing: pairing state is missing",
		"",
	}, "\n")
	if got != want {
		t.Fatalf("RenderStatusSummary() mismatch\n--- got ---\n%s--- want ---\n%s", got, want)
	}
}

func TestRenderStatusSummary_SanitizesMultilineStatusFields(t *testing.T) {
	got := RenderStatusSummary(StatusSummary{
		Channels: []StatusChannel{{Name: "telegram\n- injected", Detail: "allowed_chat_id=42\nsecret=leak"}},
		Runtime: RuntimeStatus{
			GatewayState:   "running\nforged",
			ConfigReload:   RuntimeConfigReloadEvidence{Status: "failed\nforged"},
			MemoryPressure: RuntimeMemoryPressureEvidence{Status: "warn\nforged", RSSMB: 512, WarnRSSMB: 256, CriticalRSSMB: 1024},
			Platforms: map[string]PlatformRuntimeStatus{
				"telegram\n- injected": {State: "failed\nforged", ErrorMessage: "denied\nnext=line"},
			},
		},
		Pairing: PairingStatus{
			Pending:  []PairingPendingRecord{{Platform: "telegram\nforged", UserID: "ada\nroot", Code: "ABC\n123", AgeSeconds: 1}},
			Degraded: []PairingDegradedEvidence{{Platform: "telegram\nforged", Reason: "locked\nout", Message: "bad\nline"}},
		},
	})

	for _, forbidden := range []string{"\n- injected", "\nsecret=leak", "\nnext=line", "ada\nroot", "bad\nline", "config_reload=failed\n", "memory_pressure: warn\n"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("RenderStatusSummary leaked multiline field %q in:\n%s", forbidden, got)
		}
	}
	for _, want := range []string{
		"runtime: running forged (active_agents=0 config_reload=failed forged)",
		"memory_pressure: warn forged rss=512MB warn=256MB critical=1024MB",
		"- telegram - injected: lifecycle=failed forged error=\"denied next=line\"; pairing=unpaired pending=0 approved=0; target=allowed_chat_id=42 secret=leak",
		"- pending telegram forged user=ada root code=ABC 123 age=1s",
		"- pairing telegram forged locked out: bad line",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderStatusSummary missing sanitized value %q in:\n%s", want, got)
		}
	}
}

func TestRenderStatusSummaryClampsNegativeMemoryPressureCounters(t *testing.T) {
	got := RenderStatusSummary(StatusSummary{Runtime: RuntimeStatus{MemoryPressure: RuntimeMemoryPressureEvidence{
		Status:          "warn",
		RSSMB:           -512,
		WarnRSSMB:       -256,
		CriticalRSSMB:   -1024,
		UptimeSeconds:   -60,
		GoRoutines:      -3,
		TargetPID:       -42,
		TargetStartTime: -99,
	}}})
	for _, forbidden := range []string{"rss=-", "warn=-", "critical=-", "uptime=-", "goroutines=-", "target_pid=-", "target_start_time=-"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("RenderStatusSummary leaked negative memory pressure counter %q in:\n%s", forbidden, got)
		}
	}
	if !strings.Contains(got, "memory_pressure: warn rss=0MB warn=0MB critical=0MB") {
		t.Fatalf("RenderStatusSummary missing clamped memory pressure counters in:\n%s", got)
	}
}

func TestRenderStatusSummaryClampsNegativePairingCounters(t *testing.T) {
	got := RenderStatusSummary(StatusSummary{
		Channels: []StatusChannel{{Name: "telegram", Detail: "allowed_chat_id=42"}},
		Pairing: PairingStatus{
			Platforms: []PairingPlatformStatus{{Platform: "telegram", State: "paired", PendingCount: -2, ApprovedCount: -1}},
			Pending:   []PairingPendingRecord{{Platform: "telegram", UserID: "u1", Code: "ABC", AgeSeconds: -30}},
		},
	})
	for _, forbidden := range []string{"pending=-", "approved=-", "age=-"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("RenderStatusSummary leaked negative pairing counter %q in:\n%s", forbidden, got)
		}
	}
	for _, want := range []string{"pairing=paired pending=0 approved=0", "age=0s"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderStatusSummary missing clamped value %q in:\n%s", want, got)
		}
	}
}

func TestRenderStatusSummary_DoesNotMutateInputOrdering(t *testing.T) {
	channels := []StatusChannel{
		{Name: "telegram", Detail: "allowed_chat_id=42"},
		{Name: "discord", Detail: "allowed_channel_id=D123"},
	}
	original := append([]StatusChannel(nil), channels...)

	_ = RenderStatusSummary(StatusSummary{Channels: channels})

	if !reflect.DeepEqual(channels, original) {
		t.Fatalf("RenderStatusSummary mutated channels: got %+v want %+v", channels, original)
	}
}
