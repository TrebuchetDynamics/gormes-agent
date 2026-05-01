// Package gateway is the channel-agnostic messaging chassis for Gormes.
package gateway

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
)

// HandleSkillsCommand parses and executes /skills subcommands (list, inspect).
// Returns the text output to render in the channel.
func HandleSkillsCommand(body string) string {
	// Strip "/skills" prefix
	text := strings.TrimSpace(body)
	text = strings.TrimPrefix(text, "/skills")
	text = strings.TrimSpace(text)

	parts := strings.Fields(text)
	if len(parts) == 0 {
		return renderSkillsHelp()
	}

	subcommand := strings.ToLower(parts[0])
	switch subcommand {
	case "list":
		return handleSkillsList(parts[1:])
	case "inspect":
		if len(parts) < 2 {
			return "Usage: /skills inspect <skill-name>\n"
		}
		return handleSkillsInspect(parts[1])
	case "help":
		return renderSkillsHelp()
	default:
		return fmt.Sprintf("Unknown /skills subcommand: %q. Try /skills list or /skills inspect <name>\n", subcommand)
	}
}

func handleSkillsList(args []string) string {
	opts := skills.ListOptions{}
	disabled := map[string]struct{}{}

	// Parse optional --source filter
	for _, arg := range args {
		if strings.HasPrefix(arg, "--source=") {
			opts.Source = strings.TrimPrefix(arg, "--source=")
		} else if arg == "--source" && len(args) > 1 {
			opts.Source = args[1]
		}
	}

	rows := skills.ListInstalledSkills(opts, disabled)

	if len(rows) == 0 {
		return "No skills installed.\n"
	}

	var b strings.Builder
	b.WriteString("Installed Skills\n\n")

	for _, row := range rows {
		status := "enabled"
		if row.Status == skills.SkillStatusDisabled {
			status = "disabled"
		}
		category := row.Category
		if category == "" {
			category = "-"
		}
		b.WriteString(fmt.Sprintf("%s  %s  %s  %s  %s\n",
			row.Name,
			category,
			row.Source,
			row.Trust,
			status,
		))
	}

	// Summary
	hubCount := 0
	builtinCount := 0
	localCount := 0
	enabledCount := 0
	disabledCount := 0

	for _, row := range rows {
		switch row.Source {
		case "hub":
			hubCount++
		case "builtin":
			builtinCount++
		default:
			localCount++
		}
		if row.Status == skills.SkillStatusDisabled {
			disabledCount++
		} else {
			enabledCount++
		}
	}

	b.WriteString(fmt.Sprintf("\n(%d hub-installed, %d builtin, %d local — %d enabled, %d disabled)\n",
		hubCount, builtinCount, localCount, enabledCount, disabledCount))

	return b.String()
}

func handleSkillsInspect(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Usage: /skills inspect <skill-name>\n"
	}

	// Find the skill by name
	opts := skills.ListOptions{}
	rows := skills.ListInstalledSkills(opts, nil)

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
	b.WriteString(fmt.Sprintf("Skill: %s\n", found.Name))

	// Try to read the skill file for more details
	path := found.Path
	if path != "" {
		skillDir := filepath.Dir(path)
		skillFile := filepath.Join(skillDir, "SKILL.md")
		if raw, err := os.ReadFile(skillFile); err == nil {
			// Try to parse frontmatter
			skill, err := skills.Parse(raw, skills.DefaultMaxDocumentBytes)
			if err == nil {
				b.WriteString(fmt.Sprintf("Description: %s\n", skill.Description))
				if len(skill.HermesTags) > 0 {
					b.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(skill.HermesTags, ", ")))
				}
				if len(skill.RelatedSkills) > 0 {
					b.WriteString(fmt.Sprintf("Related: %s\n", strings.Join(skill.RelatedSkills, ", ")))
				}
				b.WriteString(fmt.Sprintf("\n---\n%s\n", strings.TrimSpace(skill.Body[:min(len(skill.Body), 2000)])))
				if len(skill.Body) > 2000 {
					b.WriteString(fmt.Sprintf("\n... (%d more characters)", len(skill.Body)-2000))
				}
			}
		}
	}

	b.WriteString(fmt.Sprintf("\nSource: %s\n", found.Source))
	b.WriteString(fmt.Sprintf("Trust: %s\n", found.Trust))
	b.WriteString(fmt.Sprintf("Status: %s\n", found.Status))

	return b.String()
}

func renderSkillsHelp() string {
	return `Skills commands:
  /skills list            List all installed skills
  /skills list --source hub|builtin|local  Filter by source
  /skills inspect <name>  Show details for a specific skill
  /skills help            Show this help

Examples:
  /skills list
  /skills list --source builtin
  /skills inspect gormes-builder
`
}

func (m *Manager) handleSkillsCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, HandleSkillsCommand(ev.Text))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
