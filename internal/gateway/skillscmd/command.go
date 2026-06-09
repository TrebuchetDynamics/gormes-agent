package skillscmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/commandline"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
)

// SkillsCommandOptions injects read-only seams for /skills command tests and
// non-network gateway/TUI callers. Zero values preserve the installed-skill
// list/inspect defaults and use an empty hub provider set for browse/search.
type SkillsCommandOptions struct {
	SkillsRoot          string
	BundledRoot         string
	ExternalDirs        []string
	URLInstall          skills.URLInstallPolicy
	Disabled            map[string]struct{}
	HubProviders        []skills.HubRegistryProvider
	PageSize            int
	MaxRows             int
	MaxDescriptionRunes int
}

// HandleSkillsCommand parses and executes /skills subcommands.
// Returns the text output to render in the channel.
func HandleSkillsCommand(body string) string {
	opts := SkillsCommandOptions{}
	if skillsCommandNeedsInstalledRoots(body) {
		opts = defaultSkillsCommandOptions()
	}
	return HandleSkillsCommandWithOptions(context.Background(), body, opts)
}

func skillsCommandNeedsInstalledRoots(body string) bool {
	text := strings.TrimSpace(body)
	text = strings.TrimPrefix(text, "/skills")
	parts := strings.Fields(strings.TrimSpace(text))
	if len(parts) == 0 {
		return false
	}
	switch strings.ToLower(parts[0]) {
	case "list", "inspect", "view", "show":
		return true
	default:
		return false
	}
}

func defaultSkillsCommandOptions() SkillsCommandOptions {
	cfg, err := config.Load(nil)
	if err != nil {
		return SkillsCommandOptions{}
	}
	externalDirs, _ := cfg.ExternalSkillsDirs()
	return SkillsCommandOptions{
		SkillsRoot:   cfg.SkillsRoot(),
		BundledRoot:  skills.BundledRoot(),
		ExternalDirs: externalDirs,
	}
}

// HandleSkillsCommandWithOptions parses and executes /skills subcommands using
// explicit read-only dependencies. It is the shared gateway/TUI seam for
// installed list/inspect, hub browse/search, and mutating-action unavailable
// evidence.
func HandleSkillsCommandWithOptions(ctx context.Context, body string, opts SkillsCommandOptions) string {
	text := strings.TrimSpace(body)
	if payload, ok := commandline.PayloadIfCommand(text, "skills"); ok {
		text = payload
	} else {
		text = strings.TrimPrefix(text, "/skills")
		text = strings.TrimSpace(text)
	}

	parts := strings.Fields(text)
	if len(parts) == 0 {
		return renderSkillsHelp()
	}

	subcommand := strings.ToLower(parts[0])
	switch subcommand {
	case "list":
		return handleSkillsList(parts[1:], opts)
	case "inspect", "view", "show":
		if len(parts) < 2 {
			return "Usage: /skills inspect <skill-name>\n"
		}
		return handleSkillsInspect(parts[1], opts)
	case "search":
		return handleSkillsSearch(ctx, parts[1:], opts)
	case "browse":
		return handleSkillsBrowse(ctx, parts[1:], opts)
	case "install":
		return handleSkillsInstall(ctx, parts[1:], opts)
	case "edit", "disable", "review":
		return renderSkillsManageUnavailable(subcommand, opts)
	case "help":
		return renderSkillsHelp()
	default:
		return fmt.Sprintf("Unknown /skills subcommand: %q. Try /skills list, /skills inspect <name>, /skills search <query>, or /skills browse\n", sanitizeSkillCommandText(subcommand, opts))
	}
}

func handleSkillsList(args []string, opts SkillsCommandOptions) string {
	listOpts := skills.ListOptions{}
	parsed := parseSkillsCommandArgs(args)
	if source := parsed.value("source"); source != "" {
		listOpts.Source = source
	}
	if parsed.bool("enabled-only") {
		listOpts.EnabledOnly = true
	}

	rows := listInstalledSkillsForCommand(listOpts, opts)

	if len(rows) == 0 {
		return "No skills installed.\n"
	}

	var b strings.Builder
	b.WriteString("Installed Skills\n\n")

	for _, row := range rows {
		status := "enabled"
		if row.Status == skills.SkillStatusDisabled {
			status = "disabled"
		} else if row.Status != "" {
			status = string(row.Status)
		}
		category := row.Category
		if category == "" {
			category = "-"
		}
		b.WriteString(fmt.Sprintf("%s  %s  %s  %s  %s\n",
			sanitizeSkillCommandText(row.Name, opts),
			sanitizeSkillCommandText(category, opts),
			sanitizeSkillCommandText(row.Source, opts),
			sanitizeSkillCommandText(row.Trust, opts),
			sanitizeSkillCommandText(status, opts),
		))
	}

	// Summary
	hubCount := 0
	builtinCount := 0
	localCount := 0
	externalCount := 0
	enabledCount := 0
	disabledCount := 0

	for _, row := range rows {
		switch row.Source {
		case "hub":
			hubCount++
		case "builtin":
			builtinCount++
		case "external":
			externalCount++
		default:
			localCount++
		}
		if row.Status == skills.SkillStatusDisabled {
			disabledCount++
		} else {
			enabledCount++
		}
	}

	b.WriteString(fmt.Sprintf("\n(%d hub-installed, %d builtin, %d local, %d external — %d enabled, %d disabled)\n",
		hubCount, builtinCount, localCount, externalCount, enabledCount, disabledCount))

	return b.String()
}

func handleSkillsInspect(name string, opts SkillsCommandOptions) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Usage: /skills inspect <skill-name>\n"
	}

	// Find the skill by name
	rows := listInstalledSkillsForCommand(skills.ListOptions{}, opts)

	var found *skills.SkillRow
	for i := range rows {
		if strings.EqualFold(rows[i].Name, name) {
			found = &rows[i]
			break
		}
	}

	if found == nil {
		// Try partial match
		lowerName := strings.ToLower(name)
		for i := range rows {
			if strings.Contains(strings.ToLower(rows[i].Name), lowerName) {
				found = &rows[i]
				break
			}
		}
	}

	if found == nil {
		return fmt.Sprintf("Skill %q not found. Use /skills list to see available skills.\n", name)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Skill: %s\n", sanitizeSkillCommandText(found.Name, opts)))

	// Try to read the skill file for more details
	path := found.Path
	if path != "" {
		skillDir := filepath.Dir(path)
		skillFile := filepath.Join(skillDir, "SKILL.md")
		if raw, err := os.ReadFile(skillFile); err == nil {
			// Try to parse frontmatter
			skill, err := skills.Parse(raw, skills.DefaultMaxDocumentBytes)
			if err == nil {
				b.WriteString(fmt.Sprintf("Description: %s\n", sanitizeSkillCommandText(skill.Description, opts)))
				if len(skill.HermesTags) > 0 {
					b.WriteString(fmt.Sprintf("Tags: %s\n", sanitizeSkillCommandText(strings.Join(skill.HermesTags, ", "), opts)))
				}
				if len(skill.RelatedSkills) > 0 {
					b.WriteString(fmt.Sprintf("Related: %s\n", sanitizeSkillCommandText(strings.Join(skill.RelatedSkills, ", "), opts)))
				}
				preview := buildSkillBodyPreview(skill.Body, skillInspectBodyPreviewRunes, opts)
				b.WriteString(fmt.Sprintf("\n---\n%s\n", preview.Text))
				if preview.MoreRunes > 0 {
					b.WriteString(fmt.Sprintf("\n... (%d more characters)", preview.MoreRunes))
				}
			} else {
				b.WriteString("Evidence: skills_external_dir_skipped reason=skill_parse_failed\n")
			}
		} else {
			b.WriteString("Evidence: skills_external_dir_skipped reason=skill_file_unavailable\n")
		}
	}

	b.WriteString(fmt.Sprintf("\nSource: %s\n", sanitizeSkillCommandText(found.Source, opts)))
	b.WriteString(fmt.Sprintf("Trust: %s\n", sanitizeSkillCommandText(found.Trust, opts)))
	b.WriteString(fmt.Sprintf("Status: %s\n", sanitizeSkillCommandText(string(found.Status), opts)))

	return b.String()
}

func handleSkillsInstall(ctx context.Context, args []string, opts SkillsCommandOptions) string {
	parsed := parseSkillsCommandArgs(args)
	if len(parsed.positionals) == 0 {
		return "Usage: /skills install <https://.../SKILL.md> --name <safe-name> [--category <category>]\n"
	}
	req := skills.URLInstallRequest{
		URL:              parsed.positionals[0],
		NameOverride:     parsed.value("name"),
		CategoryOverride: parsed.value("category"),
		Interactive:      false,
	}
	ev := skills.PerformURLInstall(ctx, opts.URLInstall, req)
	return renderSkillsInstallEvidence(ev, opts)
}

func renderSkillsInstallEvidence(ev skills.URLInstallEvidence, opts SkillsCommandOptions) string {
	code := strings.TrimSpace(ev.Code)
	if code == "" {
		code = "skills_install_unknown"
	}
	reason := sanitizeSkillCommandText(ev.Reason, opts)
	if reason == "" {
		return code + "\n"
	}
	return fmt.Sprintf("%s: %s\n", code, reason)
}

func handleSkillsSearch(ctx context.Context, args []string, opts SkillsCommandOptions) string {
	parsed := parseSkillsCommandArgs(args)
	query := strings.TrimSpace(strings.Join(parsed.positionals, " "))
	if query == "" {
		return "Usage: /skills search <query>\n"
	}
	resp, err := skills.Search(ctx, query, opts.HubProviders, skills.HubSearchOptions{Limit: skillsCommandMaxRows(opts)})
	if err != nil {
		return "Skill Hub Search\nerror: " + sanitizeSkillCommandText(err.Error(), opts) + "\n"
	}
	var b strings.Builder
	b.WriteString("Skill Hub Search\n")
	b.WriteString(fmt.Sprintf("query: %s\n", sanitizeSkillCommandText(query, opts)))
	b.WriteString(fmt.Sprintf("%d result(s)\n", len(resp.Results)))
	if resp.Evidence != "" {
		b.WriteString(fmt.Sprintf("evidence: %s\n", resp.Evidence))
	}
	renderHubRows(&b, resp.Results, opts)
	return b.String()
}

func handleSkillsBrowse(ctx context.Context, args []string, opts SkillsCommandOptions) string {
	parsed := parseSkillsCommandArgs(args)
	page := parsed.intValue("page")
	if page == 0 && len(parsed.positionals) > 0 {
		if parsedPage, err := strconv.Atoi(parsed.positionals[0]); err == nil {
			page = parsedPage
		}
	}
	pageSize := parsed.intValue("page-size")
	if pageSize == 0 {
		pageSize = skillsCommandPageSize(opts)
	}
	resp, err := skills.Browse(ctx, opts.HubProviders, skills.HubBrowseOptions{Page: page, PageSize: pageSize})
	if err != nil {
		return "Skill Hub Browse\nerror: " + sanitizeSkillCommandText(err.Error(), opts) + "\n"
	}
	var b strings.Builder
	b.WriteString("Skill Hub Browse\n")
	b.WriteString(fmt.Sprintf("page %d/%d, %d total\n", resp.Page, resp.TotalPages, resp.Total))
	if resp.Evidence != "" {
		b.WriteString(fmt.Sprintf("evidence: %s\n", resp.Evidence))
	}
	renderHubRows(&b, resp.Results, opts)
	return b.String()
}

func renderHubRows(b *strings.Builder, rows []skills.HubSearchResult, opts SkillsCommandOptions) {
	if len(rows) == 0 {
		return
	}
	for i, row := range rows {
		name := sanitizeSkillCommandText(row.Name, opts)
		desc := sanitizeSkillCommandText(row.Description, opts)
		if desc == "" {
			desc = "no description"
		}
		source := sanitizeSkillCommandText(row.Source, opts)
		if source == "" {
			source = "unknown"
		}
		trust := sanitizeSkillCommandText(row.TrustLevel, opts)
		if trust == "" {
			trust = "unknown"
		}
		b.WriteString(fmt.Sprintf("%d. %s — %s\n", i+1, name, desc))
		b.WriteString(fmt.Sprintf("   source=%s status=available trust=%s", source, trust))
		if len(row.Tags) > 0 {
			cleanTags := make([]string, 0, len(row.Tags))
			for _, tag := range row.Tags {
				if clean := sanitizeSkillCommandText(tag, opts); clean != "" {
					cleanTags = append(cleanTags, clean)
				}
			}
			if len(cleanTags) > 0 {
				b.WriteString(" tags=" + strings.Join(cleanTags, ","))
			}
		}
		b.WriteString("\n")
	}
}

func renderSkillsManageUnavailable(action string, opts SkillsCommandOptions) string {
	return fmt.Sprintf("skills_manage_unavailable: /skills %s is row 6.F read-only in this build; mutating skills.manage actions are unavailable, and no skill store was changed.\n", sanitizeSkillCommandText(action, opts))
}

func renderSkillsHelp() string {
	return `Skills commands:
  /skills list            List all installed skills
  /skills list --source hub|builtin|local|external  Filter by source
  /skills inspect <name>  Show details for a specific installed skill
  /skills install <https://.../SKILL.md> --name <safe-name>  Install a direct URL skill
  /skills search <query>  Search read-only skill hub metadata
  /skills browse [page]   Browse read-only skill hub metadata
  /skills help            Show this help

Read-only note:
  /skills edit, disable, and review return row-backed unavailable evidence in this build.

Examples:
  /skills list
  /skills list --source builtin
  /skills inspect gormes-builder
  /skills search planner
  /skills browse --page 2 --page-size 10
`
}

func listInstalledSkillsForCommand(listOpts skills.ListOptions, opts SkillsCommandOptions) []skills.SkillRow {
	listOpts.ExternalRoots = opts.ExternalDirs
	if strings.TrimSpace(opts.SkillsRoot) != "" || strings.TrimSpace(opts.BundledRoot) != "" || len(opts.ExternalDirs) > 0 {
		return skills.ListInstalledSkillsFromRoots(opts.SkillsRoot, opts.BundledRoot, listOpts, opts.Disabled)
	}
	return skills.ListInstalledSkills(listOpts, opts.Disabled)
}

type parsedSkillsArgs struct {
	flags       map[string]string
	positionals []string
}

func parseSkillsCommandArgs(args []string) parsedSkillsArgs {
	parsed := parsedSkillsArgs{flags: map[string]string{}}
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "" {
			continue
		}
		if strings.HasPrefix(arg, "--") {
			nameValue := strings.TrimPrefix(arg, "--")
			if name, value, ok := strings.Cut(nameValue, "="); ok {
				parsed.flags[strings.ToLower(strings.TrimSpace(name))] = strings.TrimSpace(value)
				continue
			}
			name := strings.ToLower(strings.TrimSpace(nameValue))
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				parsed.flags[name] = strings.TrimSpace(args[i+1])
				i++
			} else {
				parsed.flags[name] = "true"
			}
			continue
		}
		parsed.positionals = append(parsed.positionals, arg)
	}
	return parsed
}

func (p parsedSkillsArgs) value(name string) string {
	return strings.TrimSpace(p.flags[strings.ToLower(name)])
}

func (p parsedSkillsArgs) bool(name string) bool {
	switch strings.ToLower(p.value(name)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func (p parsedSkillsArgs) intValue(name string) int {
	value := p.value(name)
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
}

func skillsCommandPageSize(opts SkillsCommandOptions) int {
	if opts.PageSize > 0 {
		return opts.PageSize
	}
	return 20
}

func skillsCommandMaxRows(opts SkillsCommandOptions) int {
	if opts.MaxRows > 0 {
		return opts.MaxRows
	}
	return 20
}

func skillsCommandMaxDescriptionRunes(opts SkillsCommandOptions) int {
	if opts.MaxDescriptionRunes > 0 {
		return opts.MaxDescriptionRunes
	}
	return 160
}

const skillInspectBodyPreviewRunes = 2000

type skillBodyPreview struct {
	Text      string
	MoreRunes int
}

func buildSkillBodyPreview(body string, limit int, opts SkillsCommandOptions) skillBodyPreview {
	trimmed := sanitizeSkillCommandTextWithLimit(body, opts, -1)
	if limit <= 0 {
		return skillBodyPreview{MoreRunes: utf8RuneCount(trimmed)}
	}
	runes := []rune(trimmed)
	if len(runes) <= limit {
		return skillBodyPreview{Text: trimmed}
	}
	return skillBodyPreview{
		Text:      string(runes[:limit]),
		MoreRunes: len(runes) - limit,
	}
}

func utf8RuneCount(s string) int {
	return len([]rune(s))
}

func sanitizeSkillCommandText(text string, opts SkillsCommandOptions) string {
	return sanitizeSkillCommandTextWithLimit(text, opts, skillsCommandMaxDescriptionRunes(opts))
}

func sanitizeSkillCommandTextWithLimit(text string, opts SkillsCommandOptions, limit int) string {
	cleaned := redaction.RedactSecrets(strings.TrimSpace(text))
	replacer := strings.NewReplacer(
		"`", "'",
		"*", "'",
		"#", "＃",
	)
	cleaned = replacer.Replace(cleaned)
	cleaned = strings.ReplaceAll(cleaned, "\r", " ")
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	fields := strings.Fields(cleaned)
	for i, field := range fields {
		trimmed := strings.Trim(field, "()[]{}.,;:")
		if strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "~/") || strings.Contains(trimmed, ":\\") {
			fields[i] = "[path]"
		}
	}
	cleaned = strings.Join(fields, " ")
	if limit > 0 {
		runes := []rune(cleaned)
		if len(runes) > limit {
			cleaned = string(runes[:limit]) + "…"
		}
	}
	return cleaned
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
