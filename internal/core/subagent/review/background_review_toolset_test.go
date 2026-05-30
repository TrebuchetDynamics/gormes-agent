package review

import (
	"sort"
	"strings"
	"testing"
)

func TestBackgroundReviewToolsetRestriction_DefaultAllowlist(t *testing.T) {
	t.Parallel()

	cfg := DefaultBackgroundReviewToolsetConfig()
	got := cfg.AllowedToolsets()
	sort.Strings(got)
	want := []string{"memory", "skills"}
	if len(got) != len(want) {
		t.Fatalf("AllowedToolsets() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("AllowedToolsets()[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	if !cfg.AllowsToolset("memory") {
		t.Errorf("AllowsToolset(memory) = false, want true")
	}
	if !cfg.AllowsToolset("skills") {
		t.Errorf("AllowsToolset(skills) = false, want true")
	}
	if cfg.AllowsToolset("execute_code") {
		t.Errorf("AllowsToolset(execute_code) = true, want false")
	}
}

func TestBackgroundReviewToolsetRestriction_DeniesExecutableToolsets(t *testing.T) {
	t.Parallel()

	cfg := DefaultBackgroundReviewToolsetConfig()

	cases := []struct {
		name    string
		toolset string
	}{
		{"execute_code", "execute_code"},
		{"shell", "shell"},
		{"browser", "browser"},
		{"file_write", "file_write"},
		{"provider_management", "provider_management"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ev, ok := cfg.CheckToolset(tc.toolset)
			if ok {
				t.Fatalf("CheckToolset(%q) ok = true, want false", tc.toolset)
			}
			if ev.Status != BackgroundReviewToolsetRestricted {
				t.Errorf("Status = %q, want %q", ev.Status, BackgroundReviewToolsetRestricted)
			}
			if ev.DeniedToolset != tc.toolset {
				t.Errorf("DeniedToolset = %q, want %q", ev.DeniedToolset, tc.toolset)
			}
			if len(ev.AllowedToolsets) != 2 {
				t.Errorf("AllowedToolsets = %v, want 2 entries", ev.AllowedToolsets)
			}
			if ev.Reason == "" {
				t.Errorf("Reason is empty; want non-empty restriction reason")
			}
		})
	}

	// Unknown toolsets must also be denied (positive allowlist).
	ev, ok := cfg.CheckToolset("unknown_toolset_xyz")
	if ok {
		t.Fatalf("CheckToolset(unknown) ok = true, want false (positive allowlist)")
	}
	if ev.Status != BackgroundReviewToolsetRestricted {
		t.Errorf("unknown Status = %q, want %q", ev.Status, BackgroundReviewToolsetRestricted)
	}

	// Empty/blank toolset name must be rejected with the unavailable status.
	for _, blank := range []string{"", "   ", "\t"} {
		ev, ok := cfg.CheckToolset(blank)
		if ok {
			t.Fatalf("CheckToolset(%q) ok = true, want false", blank)
		}
		if ev.Status != BackgroundReviewToolsetUnavailable {
			t.Errorf("CheckToolset(%q) Status = %q, want %q", blank, ev.Status, BackgroundReviewToolsetUnavailable)
		}
	}

	// Allowed toolsets must succeed and return the allowlist evidence.
	for _, allowed := range []string{"memory", "skills", "  memory  ", "Skills"} {
		ev, ok := cfg.CheckToolset(allowed)
		if !ok {
			t.Fatalf("CheckToolset(%q) ok = false, want true", allowed)
		}
		if ev.Status != BackgroundReviewToolsetAllowed {
			t.Errorf("CheckToolset(%q) Status = %q, want %q", allowed, ev.Status, BackgroundReviewToolsetAllowed)
		}
		if ev.DeniedToolset != "" {
			t.Errorf("CheckToolset(%q) DeniedToolset = %q, want empty", allowed, ev.DeniedToolset)
		}
	}
}

func TestBackgroundReviewToolsetRestriction_TelemetryIncludesEvidence(t *testing.T) {
	t.Parallel()

	cfg := DefaultBackgroundReviewToolsetConfig()
	const promptContent = "review memory and skills: secret prompt body"

	telemetry := cfg.Telemetry(promptContent, []string{"execute_code", "shell"})

	// Allowlist must be present and stable.
	allowed := append([]string(nil), telemetry.AllowedToolsets...)
	sort.Strings(allowed)
	if len(allowed) != 2 || allowed[0] != "memory" || allowed[1] != "skills" {
		t.Fatalf("Telemetry.AllowedToolsets = %v, want [memory skills]", allowed)
	}

	if len(telemetry.DeniedToolsets) != 2 {
		t.Fatalf("Telemetry.DeniedToolsets = %v, want 2 entries", telemetry.DeniedToolsets)
	}
	denied := append([]string(nil), telemetry.DeniedToolsets...)
	sort.Strings(denied)
	if denied[0] != "execute_code" || denied[1] != "shell" {
		t.Errorf("Telemetry.DeniedToolsets sorted = %v, want [execute_code shell]", denied)
	}

	if telemetry.Status != BackgroundReviewToolsetRestricted {
		t.Errorf("Telemetry.Status = %q, want %q", telemetry.Status, BackgroundReviewToolsetRestricted)
	}

	// Telemetry must NOT contain prompt body. Stringify the whole struct and
	// scan to make sure the secret prompt content never leaks via any field.
	rendered := telemetry.String()
	if strings.Contains(rendered, "secret prompt body") || strings.Contains(rendered, promptContent) {
		t.Errorf("Telemetry.String() leaked prompt content: %q", rendered)
	}
	if telemetry.PromptContent() != "" {
		t.Errorf("Telemetry.PromptContent() = %q, want empty (must not leak)", telemetry.PromptContent())
	}

	// With no denials, telemetry status flips to allowed but allowlist stays.
	telemetryNoDenied := cfg.Telemetry(promptContent, nil)
	if telemetryNoDenied.Status != BackgroundReviewToolsetAllowed {
		t.Errorf("Telemetry.Status (no denials) = %q, want %q", telemetryNoDenied.Status, BackgroundReviewToolsetAllowed)
	}
	if len(telemetryNoDenied.DeniedToolsets) != 0 {
		t.Errorf("Telemetry.DeniedToolsets (no denials) = %v, want empty", telemetryNoDenied.DeniedToolsets)
	}
	if len(telemetryNoDenied.AllowedToolsets) != 2 {
		t.Errorf("Telemetry.AllowedToolsets (no denials) = %v, want [memory skills]", telemetryNoDenied.AllowedToolsets)
	}
}

func TestBackgroundReviewToolsetRestriction_AllowedToolsetsImmutable(t *testing.T) {
	t.Parallel()

	cfg := DefaultBackgroundReviewToolsetConfig()
	first := cfg.AllowedToolsets()
	first[0] = "tampered"

	second := cfg.AllowedToolsets()
	for _, name := range second {
		if name == "tampered" {
			t.Fatalf("AllowedToolsets() returned shared slice; mutation leaked: %v", second)
		}
	}
}
