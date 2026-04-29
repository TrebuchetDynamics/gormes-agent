package tools

import "testing"

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
