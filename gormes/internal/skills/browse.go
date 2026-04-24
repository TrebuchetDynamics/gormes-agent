package skills

import (
	"fmt"
	"sort"
	"strings"
)

// DefaultBrowsePerPage caps the number of entries rendered per page when
// operators browse skills from the TUI or Telegram without specifying a
// page size. It is deliberately small so the formatted summary fits within
// one Telegram message and one TUI pane.
const DefaultBrowsePerPage = 10

// BrowseView is the typed snapshot rendered by the /skills command and the
// TUI skills pane. Both surfaces consume the same view so operators see a
// consistent listing across edges.
type BrowseView struct {
	Page           int
	PerPage        int
	TotalPages     int
	TotalInstalled int
	TotalAvailable int
	Installed      []InstalledSkill
	Available      []SkillMeta
}

// BuildBrowseView assembles a deterministic browse page from the installed
// skill list and the optional hub catalog. Installed entries always appear
// before available entries; both are sorted by Ref then Name so the output
// is stable across runs.
func BuildBrowseView(installed []InstalledSkill, available []SkillMeta, page, perPage int) BrowseView {
	if perPage <= 0 {
		perPage = DefaultBrowsePerPage
	}
	if page <= 0 {
		page = 1
	}

	sortedInstalled := sortInstalled(installed)
	sortedAvailable := sortAvailable(available)

	total := len(sortedInstalled) + len(sortedAvailable)
	totalPages := 1
	if total > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	if page > totalPages {
		page = 1
	}

	start := (page - 1) * perPage
	end := start + perPage
	if end > total {
		end = total
	}

	view := BrowseView{
		Page:           page,
		PerPage:        perPage,
		TotalPages:     totalPages,
		TotalInstalled: len(sortedInstalled),
		TotalAvailable: len(sortedAvailable),
	}

	for i := start; i < end; i++ {
		if i < len(sortedInstalled) {
			view.Installed = append(view.Installed, sortedInstalled[i])
			continue
		}
		view.Available = append(view.Available, sortedAvailable[i-len(sortedInstalled)])
	}

	return view
}

// FormatBrowseSummary renders the browse view as deterministic text that is
// safe to drop into a Telegram message body or print in the TUI.
func FormatBrowseSummary(view BrowseView) string {
	if view.Page <= 0 {
		view.Page = 1
	}
	if view.PerPage <= 0 {
		view.PerPage = DefaultBrowsePerPage
	}
	if view.TotalPages <= 0 {
		view.TotalPages = 1
	}

	var b strings.Builder
	b.WriteString("Skills browser\n")
	fmt.Fprintf(&b, "Installed (%d total):\n", view.TotalInstalled)
	if len(view.Installed) == 0 {
		if view.TotalInstalled == 0 {
			b.WriteString("  No installed skills on this page.\n")
		} else {
			b.WriteString("  (none on this page)\n")
		}
	} else {
		for _, item := range view.Installed {
			b.WriteString(formatBrowseLine(item.Name, item.Description, item.Ref))
		}
	}

	fmt.Fprintf(&b, "Available (%d total):\n", view.TotalAvailable)
	if len(view.Available) == 0 {
		if view.TotalAvailable == 0 {
			b.WriteString("  No available skills on this page.\n")
		} else {
			b.WriteString("  (none on this page)\n")
		}
	} else {
		for _, item := range view.Available {
			b.WriteString(formatBrowseLine(item.Name, item.Description, item.Ref))
		}
	}

	fmt.Fprintf(&b, "Page %d/%d", view.Page, view.TotalPages)
	return b.String()
}

func formatBrowseLine(name, description, ref string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "(unnamed)"
	}
	description = strings.TrimSpace(description)
	if description == "" {
		description = "(no description)"
	}
	line := fmt.Sprintf("  - %s — %s", name, description)
	if ref = strings.TrimSpace(ref); ref != "" {
		line += " [" + ref + "]"
	}
	return line + "\n"
}

func sortInstalled(in []InstalledSkill) []InstalledSkill {
	out := append([]InstalledSkill(nil), in...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		if out[i].Ref != out[j].Ref {
			return out[i].Ref < out[j].Ref
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func sortAvailable(in []SkillMeta) []SkillMeta {
	out := append([]SkillMeta(nil), in...)
	sortSkillMeta(out)
	return out
}
