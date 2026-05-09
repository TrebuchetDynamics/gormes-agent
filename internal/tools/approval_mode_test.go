package tools

import (
	"strings"
	"testing"
)

func TestNormalizeApprovalModeYAMLBooleanOff(t *testing.T) {
	mode := NormalizeApprovalMode(false)
	if mode.Mode != "off" {
		t.Fatalf("NormalizeApprovalMode(false) mode = %q, want off", mode.Mode)
	}
	if mode.Defaulted {
		t.Fatalf("NormalizeApprovalMode(false) defaulted = true, want false")
	}

	mode = NormalizeApprovalMode(true)
	if mode.Mode != "manual" {
		t.Fatalf("NormalizeApprovalMode(true) mode = %q, want manual", mode.Mode)
	}
	if mode.Defaulted {
		t.Fatalf("NormalizeApprovalMode(true) defaulted = true, want false")
	}
}

func TestNormalizeApprovalModeStringModes(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{input: " manual ", want: "manual"},
		{input: "SMART", want: "smart"},
		{input: "off", want: "off"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			mode := NormalizeApprovalMode(tc.input)
			if mode.Mode != tc.want {
				t.Fatalf("NormalizeApprovalMode(%q) mode = %q, want %q", tc.input, mode.Mode, tc.want)
			}
			if mode.Defaulted {
				t.Fatalf("NormalizeApprovalMode(%q) defaulted = true, want false", tc.input)
			}
		})
	}
}

func TestNormalizeApprovalModeDefaultsUnsupportedValues(t *testing.T) {
	for _, input := range []any{nil, "", "sometimes", 1, []string{"off"}, map[string]string{"mode": "off"}} {
		t.Run("default", func(t *testing.T) {
			mode := NormalizeApprovalMode(input)
			if mode.Mode != "manual" {
				t.Fatalf("NormalizeApprovalMode(%#v) mode = %q, want manual", input, mode.Mode)
			}
			if !mode.Defaulted {
				t.Fatalf("NormalizeApprovalMode(%#v) defaulted = false, want true", input)
			}
			if mode.Evidence["approval_mode_defaulted"] != "true" {
				t.Fatalf("NormalizeApprovalMode(%#v) evidence = %#v, want approval_mode_defaulted=true", input, mode.Evidence)
			}
		})
	}
}

func TestNormalizeCronApprovalModeDefaultDeny(t *testing.T) {
	for _, input := range []any{nil, "", "deny", "manual", "sometimes", true, false, 1, []string{"approve"}, map[string]string{"mode": "approve"}} {
		t.Run("default", func(t *testing.T) {
			mode := NormalizeCronApprovalMode(input)
			if mode.Mode != "deny" {
				t.Fatalf("NormalizeCronApprovalMode(%#v) mode = %q, want deny", input, mode.Mode)
			}
			if _, ok := input.(string); ok && mode.Defaulted && input == "deny" {
				t.Fatalf("NormalizeCronApprovalMode(%#v) defaulted = true, want false", input)
			}
			if mode.Evidence["cron_approval_mode"] != "deny" {
				t.Fatalf("NormalizeCronApprovalMode(%#v) evidence = %#v, want cron_approval_mode=deny", input, mode.Evidence)
			}
		})
	}
}

func TestNormalizeCronApprovalModeApproveAliases(t *testing.T) {
	for _, input := range []string{"approve", " APPROVE ", "off", "allow", "yes"} {
		t.Run(input, func(t *testing.T) {
			mode := NormalizeCronApprovalMode(input)
			if mode.Mode != "approve" {
				t.Fatalf("NormalizeCronApprovalMode(%q) mode = %q, want approve", input, mode.Mode)
			}
			if mode.Defaulted {
				t.Fatalf("NormalizeCronApprovalMode(%q) defaulted = true, want false", input)
			}
			if mode.Evidence["cron_approval_mode"] != "approve" {
				t.Fatalf("NormalizeCronApprovalMode(%q) evidence = %#v, want cron_approval_mode=approve", input, mode.Evidence)
			}
		})
	}
}

func TestCronApprovalModeApproveDoesNotBypassHardline(t *testing.T) {
	mode := NormalizeCronApprovalMode("approve")
	result := GuardCommand("rm -rf /", mode.Mode)
	if result.Approved {
		t.Fatalf("GuardCommand hardline/%s approved = true, want false: %#v", mode.Mode, result)
	}
	if !result.Hardline {
		t.Fatalf("GuardCommand hardline/%s hardline = false, want true: %#v", mode.Mode, result)
	}
	if result.ApprovalRequired {
		t.Fatalf("GuardCommand hardline/%s approval required = true, want false: %#v", mode.Mode, result)
	}
}

func TestGuardCronCommand_DenyModeBlocksRecoverableDangerous(t *testing.T) {
	result := GuardCronCommand("git reset --hard", "deny")
	if result.Approved {
		t.Fatalf("GuardCronCommand recoverable/deny approved = true, want false: %#v", result)
	}
	if result.Hardline {
		t.Fatalf("GuardCronCommand recoverable/deny hardline = true, want false: %#v", result)
	}
	if result.ApprovalRequired {
		t.Fatalf("GuardCronCommand recoverable/deny approval required = true, want noninteractive block: %#v", result)
	}
	if got := result.Evidence["cron_approval_mode"]; got != "deny" {
		t.Fatalf("cron_approval_mode evidence = %q, want deny: %#v", got, result.Evidence)
	}
	if !strings.Contains(result.Message, "cron_mode") {
		t.Fatalf("message = %q, want cron_mode guidance", result.Message)
	}
}

func TestGuardCronCommand_ApproveModeAllowsRecoverableDangerous(t *testing.T) {
	for _, input := range []string{"approve", "off", "allow", "yes"} {
		t.Run(input, func(t *testing.T) {
			result := GuardCronCommand("git reset --hard", input)
			if !result.Approved {
				t.Fatalf("GuardCronCommand recoverable/%s approved = false, want true: %#v", input, result)
			}
			if result.ApprovalRequired {
				t.Fatalf("GuardCronCommand recoverable/%s approval required = true, want false: %#v", input, result)
			}
			if result.Hardline {
				t.Fatalf("GuardCronCommand recoverable/%s hardline = true, want false: %#v", input, result)
			}
			if got := result.Evidence["cron_approval_mode"]; got != "approve" {
				t.Fatalf("cron_approval_mode evidence = %q, want approve: %#v", got, result.Evidence)
			}
			if got := result.Evidence["detector"]; got != "dangerous" {
				t.Fatalf("detector evidence = %q, want dangerous: %#v", got, result.Evidence)
			}
		})
	}
}

func TestGuardCronCommand_ApproveModeDoesNotBypassHardline(t *testing.T) {
	result := GuardCronCommand("rm -rf /", "approve")
	if result.Approved {
		t.Fatalf("GuardCronCommand hardline/approve approved = true, want false: %#v", result)
	}
	if !result.Hardline {
		t.Fatalf("GuardCronCommand hardline/approve hardline = false, want true: %#v", result)
	}
	if result.ApprovalRequired {
		t.Fatalf("GuardCronCommand hardline/approve approval required = true, want false: %#v", result)
	}
	if got := result.Evidence["cron_approval_mode"]; got != "approve" {
		t.Fatalf("cron_approval_mode evidence = %q, want approve: %#v", got, result.Evidence)
	}
}

func TestApprovalModeOffBypassesRecoverableDangerousCommand(t *testing.T) {
	result := GuardCommand("git reset --hard", "off")
	if !result.Approved {
		t.Fatalf("GuardCommand recoverable/off approved = false, want true: %#v", result)
	}
	if result.ApprovalRequired {
		t.Fatalf("GuardCommand recoverable/off approval required = true, want false: %#v", result)
	}
	if result.Hardline {
		t.Fatalf("GuardCommand recoverable/off hardline = true, want false: %#v", result)
	}
	if got := result.Evidence["approval_mode"]; got != "off" {
		t.Fatalf("approval_mode evidence = %q, want off", got)
	}
	if got := result.Evidence["detector"]; got != "dangerous" {
		t.Fatalf("detector evidence = %q, want dangerous", got)
	}
}

func TestApprovalModeDoesNotBypassHardline(t *testing.T) {
	for _, input := range []string{"off", "smart", "manual"} {
		t.Run(input, func(t *testing.T) {
			result := GuardCommand("rm -rf /", input)
			if result.Approved {
				t.Fatalf("GuardCommand hardline/%s approved = true, want false: %#v", input, result)
			}
			if !result.Hardline {
				t.Fatalf("GuardCommand hardline/%s hardline = false, want true: %#v", input, result)
			}
			if result.ApprovalRequired {
				t.Fatalf("GuardCommand hardline/%s approval required = true, want false: %#v", input, result)
			}
		})
	}
}
