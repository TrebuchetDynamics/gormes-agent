package skills

import (
	"strings"
	"testing"
)

func TestBuildBrowseView_PaginatesInstalledAndAvailable(t *testing.T) {
	installed := []InstalledSkill{
		{Name: "zeta", Description: "zeta desc", Ref: "bundled/ops/zeta"},
		{Name: "alpha", Description: "alpha desc", Ref: "bundled/ops/alpha"},
		{Name: "beta", Description: "beta desc"},
	}
	available := []SkillMeta{
		{Source: "clawhub", Ref: "clawhub/devops/docker-management", Name: "docker-management", Description: "Manage Docker"},
		{Source: "official", Ref: "official/research/arxiv", Name: "arxiv", Description: "Search arxiv"},
	}

	view := BuildBrowseView(installed, available, 1, 2)

	if view.Page != 1 {
		t.Fatalf("Page = %d, want 1", view.Page)
	}
	if view.PerPage != 2 {
		t.Fatalf("PerPage = %d, want 2", view.PerPage)
	}
	if view.TotalInstalled != 3 {
		t.Fatalf("TotalInstalled = %d, want 3", view.TotalInstalled)
	}
	if view.TotalAvailable != 2 {
		t.Fatalf("TotalAvailable = %d, want 2", view.TotalAvailable)
	}
	if view.TotalPages != 3 {
		t.Fatalf("TotalPages = %d, want 3", view.TotalPages)
	}

	if len(view.Installed) != 2 {
		t.Fatalf("Installed len = %d, want 2", len(view.Installed))
	}
	if view.Installed[0].Name != "alpha" || view.Installed[1].Name != "beta" {
		t.Fatalf("Installed names = %q, want alpha,beta", installedNames(view.Installed))
	}
	if len(view.Available) != 0 {
		t.Fatalf("Available on page 1 len = %d, want 0 (installed fills the page)", len(view.Available))
	}

	view2 := BuildBrowseView(installed, available, 2, 2)
	if view2.Page != 2 {
		t.Fatalf("Page2 = %d, want 2", view2.Page)
	}
	if len(view2.Installed) != 1 || view2.Installed[0].Name != "zeta" {
		t.Fatalf("Installed page 2 = %q, want zeta", installedNames(view2.Installed))
	}
	if len(view2.Available) != 1 || view2.Available[0].Ref != "clawhub/devops/docker-management" {
		t.Fatalf("Available page 2 refs = %q, want docker-management first", availableRefs(view2.Available))
	}
}

func TestBuildBrowseView_ClampsPageAndPerPage(t *testing.T) {
	installed := []InstalledSkill{
		{Name: "alpha", Description: "alpha desc"},
	}
	view := BuildBrowseView(installed, nil, 0, 0)
	if view.Page != 1 {
		t.Fatalf("Page = %d, want 1 (clamped)", view.Page)
	}
	if view.PerPage != DefaultBrowsePerPage {
		t.Fatalf("PerPage = %d, want %d (clamped)", view.PerPage, DefaultBrowsePerPage)
	}

	big := BuildBrowseView(installed, nil, 99, 2)
	if big.Page != 1 {
		t.Fatalf("Page for out-of-range request = %d, want 1 (empty page fallback)", big.Page)
	}
}

func TestFormatBrowseSummary_IsDeterministicAndIncludesSections(t *testing.T) {
	installed := []InstalledSkill{
		{Name: "alpha", Description: "alpha desc", Ref: "bundled/ops/alpha"},
	}
	available := []SkillMeta{
		{Source: "clawhub", Ref: "clawhub/devops/docker-management", Name: "docker-management", Description: "Manage Docker"},
	}
	view := BuildBrowseView(installed, available, 1, DefaultBrowsePerPage)

	text := FormatBrowseSummary(view)
	for _, want := range []string{
		"Skills browser",
		"Installed (1 total)",
		"alpha — alpha desc",
		"bundled/ops/alpha",
		"Available (1 total)",
		"docker-management — Manage Docker",
		"clawhub/devops/docker-management",
		"Page 1/1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("FormatBrowseSummary missing %q, got:\n%s", want, text)
		}
	}

	if text != FormatBrowseSummary(view) {
		t.Fatal("FormatBrowseSummary not deterministic across calls")
	}
}

func TestFormatBrowseSummary_EmptyView(t *testing.T) {
	text := FormatBrowseSummary(BrowseView{Page: 1, PerPage: DefaultBrowsePerPage, TotalPages: 1})
	for _, want := range []string{
		"Skills browser",
		"Installed (0 total)",
		"Available (0 total)",
		"Page 1/1",
		"No installed skills",
		"No available skills",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("empty view missing %q, got:\n%s", want, text)
		}
	}
}

func installedNames(entries []InstalledSkill) string {
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name)
	}
	return strings.Join(names, ",")
}

func availableRefs(entries []SkillMeta) string {
	refs := make([]string, 0, len(entries))
	for _, e := range entries {
		refs = append(refs, e.Ref)
	}
	return strings.Join(refs, ",")
}
