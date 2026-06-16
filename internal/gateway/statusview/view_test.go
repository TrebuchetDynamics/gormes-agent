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

func TestRenderStatusSummaryRedactsAuthorizationStatusFields(t *testing.T) {
	got := RenderStatusSummary(StatusSummary{
		Channels: []StatusChannel{{Name: "telegram", Detail: "authorization=Bearer channel-secret-token"}},
		Runtime: RuntimeStatus{
			GatewayState:   "running authorization=Bearer state-secret-token",
			ExitReason:     "authorization=Bearer exit-secret-token",
			MemoryPressure: RuntimeMemoryPressureEvidence{Status: "warn", Message: "authorization=Bearer memory-secret-token"},
			Platforms: map[string]PlatformRuntimeStatus{
				"telegram": {State: "failed", ErrorMessage: "authorization=Bearer platform-secret-token"},
			},
			ExpiryFinalize: []RuntimeExpiryFinalizeEvidence{{Status: "expiry_finalize_failed", Error: "authorization=Bearer expiry-secret-token"}},
		},
		Pairing: PairingStatus{
			Pending:  []PairingPendingRecord{{Platform: "telegram", UserID: "authorization=Bearer user-secret-token", Code: "authorization=Bearer code-secret-token"}},
			Degraded: []PairingDegradedEvidence{{Platform: "telegram", Reason: "authorization=Bearer reason-secret-token", Message: "authorization=Bearer degraded-secret-token"}},
		},
	})
	for _, forbidden := range []string{"channel-secret-token", "state-secret-token", "exit-secret-token", "memory-secret-token", "platform-secret-token", "expiry-secret-token", "user-secret-token", "code-secret-token", "reason-secret-token", "degraded-secret-token", "authorization", "Bearer", "bearer"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("RenderStatusSummary leaked authorization field %q in:\n%s", forbidden, got)
		}
	}
	if !strings.Contains(got, "[redacted]") {
		t.Fatalf("RenderStatusSummary missing redaction marker in:\n%s", got)
	}
}

func TestRenderStatusSummaryRemovesHiddenFormattingStatusFields(t *testing.T) {
	got := RenderStatusSummary(StatusSummary{
		Channels: []StatusChannel{{Name: "telegram\u202e", Detail: "allowed_chat_id=42\u200d"}},
		Runtime: RuntimeStatus{
			GatewayState:   "running\u202e",
			MemoryPressure: RuntimeMemoryPressureEvidence{Status: "warn\u200d", Message: "memory ok\u2066"},
			Platforms: map[string]PlatformRuntimeStatus{
				"telegram\u202e": {State: "failed\u202e", ErrorMessage: "denied\u200d"},
			},
		},
		Pairing: PairingStatus{Pending: []PairingPendingRecord{{Platform: "telegram\u202e", UserID: "ada\u200d", Code: "ABC\u2066", AgeSeconds: 1}}},
	})
	for _, forbidden := range []string{"\u202e", "\u200d", "\u2066"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("RenderStatusSummary leaked hidden formatting rune %q in:\n%s", forbidden, got)
		}
	}
	for _, want := range []string{"runtime: running", "memory_pressure: warn", "- telegram: lifecycle=failed error=\"denied\"", "- pending telegram user=ada code=ABC age=1s"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderStatusSummary missing sanitized value %q in:\n%s", want, got)
		}
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
		"- telegram - injected: lifecycle=failed forged error=\"denied next=line\"; pairing=unpaired pending=0 approved=0; target=allowed_chat_id=42 [redacted]",
		"- pending telegram forged user=ada root code=ABC 123 age=1s",
		"- pairing telegram forged locked out: bad line",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderStatusSummary missing sanitized value %q in:\n%s", want, got)
		}
	}
}

func TestRenderStatusSummaryClampsNegativeRuntimeActiveAgents(t *testing.T) {
	got := RenderStatusSummary(StatusSummary{Runtime: RuntimeStatus{
		GatewayState: "running",
		ActiveAgents: -4,
	}})
	if strings.Contains(got, "active_agents=-") {
		t.Fatalf("RenderStatusSummary leaked negative active agent count in:\n%s", got)
	}
	if !strings.Contains(got, "active_agents=0") {
		t.Fatalf("RenderStatusSummary missing clamped active agent count in:\n%s", got)
	}
}

func TestRenderStatusSummaryOmitsEmptyMemoryPressureEvidenceItems(t *testing.T) {
	got := RenderStatusSummary(StatusSummary{Runtime: RuntimeStatus{MemoryPressure: RuntimeMemoryPressureEvidence{
		Status:   "warn",
		RSSMB:    512,
		Evidence: []string{"\n", "\t"},
	}}})
	if strings.Contains(got, "evidence=") {
		t.Fatalf("RenderStatusSummary rendered empty evidence field in:\n%s", got)
	}
	if !strings.Contains(got, "memory_pressure: warn rss=512MB") {
		t.Fatalf("RenderStatusSummary missing memory pressure line in:\n%s", got)
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

func TestRenderStatusSummaryClampsNegativeExpiryFinalizeAttempts(t *testing.T) {
	got := RenderStatusSummary(StatusSummary{Runtime: RuntimeStatus{ExpiryFinalize: []RuntimeExpiryFinalizeEvidence{{
		SessionID: "sess-expire",
		Status:    "expiry_finalize_failed",
		Attempts:  -3,
	}}}})
	if strings.Contains(got, "attempts=-") {
		t.Fatalf("RenderStatusSummary leaked negative expiry finalize attempts in:\n%s", got)
	}
	if !strings.Contains(got, "attempts=0") {
		t.Fatalf("RenderStatusSummary missing clamped expiry finalize attempts in:\n%s", got)
	}
}

func TestRenderStatusSummaryOmitsBlankApprovedPairingFields(t *testing.T) {
	got := RenderStatusSummary(StatusSummary{Pairing: PairingStatus{Approved: []PairingApprovedRecord{{
		Platform: "telegram",
		UserID:   "\n\t",
		UserName: " ",
	}}}})
	for _, forbidden := range []string{"user= ", "name=", "user=\n"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("RenderStatusSummary rendered blank approved field %q in:\n%s", forbidden, got)
		}
	}
	if !strings.Contains(got, "- approved telegram\n") {
		t.Fatalf("RenderStatusSummary missing approved line without blank fields in:\n%s", got)
	}
}

func TestRenderStatusSummaryOmitsBlankPendingPairingFields(t *testing.T) {
	got := RenderStatusSummary(StatusSummary{Pairing: PairingStatus{Pending: []PairingPendingRecord{{
		Platform: "telegram",
		UserID:   "\n\t",
		Code:     " ",
	}}}})
	for _, forbidden := range []string{"user= ", "code= ", "user= code="} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("RenderStatusSummary rendered blank pending field %q in:\n%s", forbidden, got)
		}
	}
	if !strings.Contains(got, "- pending telegram age=0s") {
		t.Fatalf("RenderStatusSummary missing pending line without blank fields in:\n%s", got)
	}
}

func TestRenderStatusSummaryOmitsBlankPairingDegradedFields(t *testing.T) {
	got := RenderStatusSummary(StatusSummary{Pairing: PairingStatus{Degraded: []PairingDegradedEvidence{{
		Platform: "\n\t",
		Reason:   "\n",
		Message:  "\t",
	}}}})
	if strings.Contains(got, "- pairing  :") || strings.Contains(got, ": \n") {
		t.Fatalf("RenderStatusSummary rendered blank degraded fields in:\n%s", got)
	}
	if !strings.Contains(got, "- pairing\n") {
		t.Fatalf("RenderStatusSummary missing bare degraded pairing line in:\n%s", got)
	}
}

func TestRenderStatusSummaryOmitsBlankExpiryFinalizeFields(t *testing.T) {
	got := RenderStatusSummary(StatusSummary{Runtime: RuntimeStatus{ExpiryFinalize: []RuntimeExpiryFinalizeEvidence{{
		SessionID: "\n\t",
		Source:    "\n",
		ChatID:    "\t",
		UserID:    " ",
		Status:    "expiry_finalize_failed",
		Error:     "\n",
	}}}})
	for _, forbidden := range []string{"session=", "source=", "chat=", "user=", "error=\"\""} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("RenderStatusSummary rendered blank expiry finalize field %q in:\n%s", forbidden, got)
		}
	}
	if !strings.Contains(got, "- expiry_finalize_failed attempts=0") {
		t.Fatalf("RenderStatusSummary missing expiry finalize evidence line in:\n%s", got)
	}
}

func TestRenderStatusSummaryOmitsBlankExpiryFinalizedFields(t *testing.T) {
	got := RenderStatusSummary(StatusSummary{Runtime: RuntimeStatus{ExpiryFinalized: []RuntimeExpiryFinalizedEvidence{{
		SessionID:       "\n\t",
		Source:          "\n",
		ChatID:          "\t",
		UserID:          " ",
		ExpiryFinalized: true,
	}}}})
	for _, forbidden := range []string{"session=", "source=", "chat=", "user="} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("RenderStatusSummary rendered blank expiry finalized field %q in:\n%s", forbidden, got)
		}
	}
	if !strings.Contains(got, "- finalized expiry_finalized=true") {
		t.Fatalf("RenderStatusSummary missing finalized evidence line in:\n%s", got)
	}
}

func TestRenderStatusSummaryOmitsBlankExitReason(t *testing.T) {
	got := RenderStatusSummary(StatusSummary{Runtime: RuntimeStatus{GatewayState: "stopped", ExitReason: "\n\t"}})
	if strings.Contains(got, "exit_reason=\"\"") {
		t.Fatalf("RenderStatusSummary rendered blank exit reason in:\n%s", got)
	}
	if !strings.Contains(got, "runtime: stopped (active_agents=0)") {
		t.Fatalf("RenderStatusSummary missing runtime without blank exit reason in:\n%s", got)
	}
}

func TestRenderStatusSummaryOmitsBlankLifecycleError(t *testing.T) {
	got := RenderStatusSummary(StatusSummary{
		Channels: []StatusChannel{{Name: "telegram", Detail: "allowed_chat_id=42"}},
		Runtime:  RuntimeStatus{Platforms: map[string]PlatformRuntimeStatus{"telegram": {State: "failed", ErrorMessage: "\n\t"}}},
	})
	if strings.Contains(got, "error=\"\"") {
		t.Fatalf("RenderStatusSummary rendered blank lifecycle error in:\n%s", got)
	}
	if !strings.Contains(got, "lifecycle=failed; pairing=unpaired") {
		t.Fatalf("RenderStatusSummary missing lifecycle without blank error in:\n%s", got)
	}
}

func TestRenderStatusSummaryDefaultsBlankLifecycleState(t *testing.T) {
	got := RenderStatusSummary(StatusSummary{
		Channels: []StatusChannel{{Name: "telegram", Detail: "allowed_chat_id=42"}},
		Runtime:  RuntimeStatus{Platforms: map[string]PlatformRuntimeStatus{"telegram": {State: "\n\t"}}},
	})
	if strings.Contains(got, "lifecycle=;") {
		t.Fatalf("RenderStatusSummary rendered blank lifecycle state in:\n%s", got)
	}
	if !strings.Contains(got, "lifecycle=unknown") {
		t.Fatalf("RenderStatusSummary missing default lifecycle state in:\n%s", got)
	}
}

func TestRenderStatusSummaryDefaultsBlankPairingState(t *testing.T) {
	got := RenderStatusSummary(StatusSummary{
		Channels: []StatusChannel{{Name: "telegram", Detail: "allowed_chat_id=42"}},
		Pairing:  PairingStatus{Platforms: []PairingPlatformStatus{{Platform: "telegram", State: "\n\t", PendingCount: 1}}},
	})
	if strings.Contains(got, "pairing= pending=") {
		t.Fatalf("RenderStatusSummary rendered blank pairing state in:\n%s", got)
	}
	if !strings.Contains(got, "pairing=unpaired pending=1 approved=0") {
		t.Fatalf("RenderStatusSummary missing default pairing state in:\n%s", got)
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
