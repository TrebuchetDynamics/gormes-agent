package security

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestGuardEmptyComposerAllows(t *testing.T) {
	g := NewGuard()
	ev := g.Compose("/some/path")

	if !ev.Allow {
		t.Errorf("Guard.Compose().Allow = false, want true (no policies loaded)")
	}
	if ev.EvidenceType != "guard_no_policies" {
		t.Errorf("Guard.Compose().EvidenceType = %q, want %q", ev.EvidenceType, "guard_no_policies")
	}
	if ev.Reason == "" {
		t.Errorf("Guard.Compose().Reason is empty")
	}
}

func TestGuardDenyOverridesAllow(t *testing.T) {
	src := writeTempFindings(t, `{
		"findings": [
			{"rule_id":"SEC001","severity":"critical","message":"Hardcoded secret","file":"config.yaml"}
		]
	}`)
	client, err := NewTirithClient(src)
	if err != nil {
		t.Fatalf("NewTirithClient: %v", err)
	}

	g := NewGuard()
	g.SetTirith(client)
	g.SetPathAllowlist([]string{"/safe/path"})
	ev := g.Compose("/safe/path")

	if ev.Allow {
		t.Errorf("Guard.Compose().Allow = true, want false (Tirith deny with critical finding)")
	}
	if ev.EvidenceType != "guard_deny" {
		t.Errorf("Guard.Compose().EvidenceType = %q, want %q", ev.EvidenceType, "guard_deny")
	}
	if ev.Reason == "" {
		t.Errorf("Guard.Compose().Reason is empty")
	}
}

func TestGuardPolicyOverridesTirith(t *testing.T) {
	// Tirith has a critical finding — would deny.
	src := writeTempFindings(t, `{
		"findings": [
			{"rule_id":"SEC001","severity":"critical","message":"Hardcoded secret","file":"config.yaml"}
		]
	}`)
	client, err := NewTirithClient(src)
	if err != nil {
		t.Fatalf("NewTirithClient: %v", err)
	}

	// URL safety policy explicitly allows the target URL.
	policy := tools.DefaultURLSafetyPolicy()
	policy.Enabled = true
	policy.Blocklist = nil
	policy.Allowlist = []tools.URLSafetyAllowlistEntry{
		{Pattern: "trusted.example.com", Source: "config"},
	}
	checker := tools.NewURLSafetyChecker(policy)

	g := NewGuard()
	g.SetTirith(client)
	g.SetURLSafety(checker)
	ev := g.Compose("https://trusted.example.com/api/data")

	if !ev.Allow {
		t.Errorf("Guard.Compose().Allow = false, want true (URL policy overrides Tirith)")
	}
	if ev.EvidenceType != "guard_allow" {
		t.Errorf("Guard.Compose().EvidenceType = %q, want %q", ev.EvidenceType, "guard_allow")
	}
	if ev.Reason == "" {
		t.Errorf("Guard.Compose().Reason is empty")
	}
}
