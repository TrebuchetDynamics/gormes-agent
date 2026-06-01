package platformconfig

import "testing"

func TestToolsetTeamsPlatformScoped(t *testing.T) {
	cfg := PlatformToolsetConfig{PlatformToolsets: map[string][]string{
		"teams": {"teams"},
		"cli":   {"teams"},
	}}

	teamsStatus, err := cfg.PlatformStatus("teams")
	if err != nil {
		t.Fatalf("PlatformStatus teams: %v", err)
	}
	if countString(teamsStatus.RuntimeToolsets, "teams") != 1 {
		t.Fatalf("teams runtime toolsets = %v, want teams", teamsStatus.RuntimeToolsets)
	}

	cliStatus, err := cfg.PlatformStatus("cli")
	if err != nil {
		t.Fatalf("PlatformStatus cli: %v", err)
	}
	if countString(cliStatus.RuntimeToolsets, "teams") != 0 {
		t.Fatalf("cli runtime toolsets = %v, want teams excluded", cliStatus.RuntimeToolsets)
	}
	assertPlatformToolsetIssue(t, cliStatus.Issues, PlatformToolsetIssueRestrictedToolset, "cli", "teams")
}

func assertPlatformToolsetIssue(t *testing.T, issues []PlatformToolsetIssue, kind PlatformToolsetIssueKind, platform, toolset string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Kind == kind && issue.Platform == platform && issue.Toolset == toolset {
			return
		}
	}
	t.Fatalf("missing issue kind=%s platform=%s toolset=%s in %+v", kind, platform, toolset, issues)
}
