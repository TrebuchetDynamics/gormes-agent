package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/core/agenttemplate"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
)

// gormesHomeSubdirs is the Gormes-OWNED ~/.gormes runtime layout - NOT
// Hermes' cron/sessions/logs/skills/memories. These are created lazily on
// first use, so a missing one is WARN (informational), never FAIL.
var gormesHomeSubdirs = []string{"sessions", "memory", "skills", "cron", "subagents", "tools", "hooks"}

// CheckDirectoryStructure inspects the Gormes home for its owned directory
// layout and the agent-template starter files. Parity with
// hermes_cli/doctor.py@55c9f3206:812 ◆ Directory Structure, rendered in
// Gormes-owned paths/wording (~/.gormes, memory/ not memories/,
// `gormes setup`). Pure local FS: no network, identical under --offline.
// "Not yet created" memory starters are non-actionable PASS items - Gormes
// Status has no INFO, and they must not inflate the Found-N issue count.
func CheckDirectoryStructure(home string) CheckResult {
	items := make([]ItemInfo, 0, len(gormesHomeSubdirs)+4)
	worst := StatusPass
	bumpWarn := func() {
		if worst == StatusPass {
			worst = StatusWarn
		}
	}

	if dirExists(home) {
		items = append(items, ItemInfo{Name: "~/.gormes", Status: StatusPass, Note: "exists"})
	} else {
		items = append(items, ItemInfo{Name: "~/.gormes", Status: StatusWarn, Note: "will be created on first use"})
		bumpWarn()
	}

	for _, d := range gormesHomeSubdirs {
		if dirExists(filepath.Join(home, d)) {
			items = append(items, ItemInfo{Name: d + "/", Status: StatusPass, Note: "exists"})
		} else {
			items = append(items, ItemInfo{Name: d + "/", Status: StatusWarn, Note: "will be created on first use"})
			bumpWarn()
		}
	}

	for _, starter := range directoryStructureStarterTemplates() {
		abs := filepath.Join(home, filepath.FromSlash(starter.Path))
		switch starter.TemplateID {
		case "soul":
			soul := soulItem(starter.Path, abs)
			items = append(items, soul)
			if soul.Status == StatusWarn {
				bumpWarn()
			}
		case "memory-memory", "memory-user":
			items = append(items, memoryStarterItem(starter.Path, abs))
		}
	}

	summary := "Gormes home layout present"
	if worst == StatusWarn {
		summary = "some Gormes directories/files not yet present (created on first use or via `gormes setup`)"
	}
	return CheckResult{Name: "Directory Structure", Status: worst, Summary: summary, Items: items}
}

func directoryStructureStarterTemplates() []agenttemplate.TemplatePair {
	order := map[string]int{
		"soul":          0,
		"memory-memory": 1,
		"memory-user":   2,
	}
	pairs := make([]agenttemplate.TemplatePair, 0, len(order))
	for _, pair := range agenttemplate.TemplatePairManifest() {
		if _, ok := order[pair.TemplateID]; ok {
			pairs = append(pairs, pair)
		}
	}
	sort.SliceStable(pairs, func(i, j int) bool {
		return order[pairs[i].TemplateID] < order[pairs[j].TemplateID]
	})
	return pairs
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func soulItem(label, path string) ItemInfo {
	data, err := os.ReadFile(path)
	if err != nil {
		return ItemInfo{Name: label, Status: StatusWarn, Note: "missing - run `gormes setup` to give Gormes a persona"}
	}
	defaultContent, hasDefault := defaultTemplateContent(label)
	if soulHasRealContent(string(data), defaultContent, hasDefault) {
		return ItemInfo{Name: label, Status: StatusPass, Note: "persona configured"}
	}
	return ItemInfo{Name: label, Status: StatusPass, Note: "present but template-only - edit it to customize personality"}
}

// soulHasRealContent mirrors hermes doctor.py:840 for blank/comment-only
// files and also treats Gormes' seeded default template as template-only.
func soulHasRealContent(content, defaultContent string, hasDefault bool) bool {
	normalized := normalizeTemplateText(content)
	if normalized == "" {
		return false
	}
	if hasDefault && normalized == normalizeTemplateText(defaultContent) {
		return false
	}
	for _, t := range textvalue.TrimmedLines(content) {
		if t == "" || strings.HasPrefix(t, "<!--") || strings.HasPrefix(t, "-->") || strings.HasPrefix(t, "#") {
			continue
		}
		return true
	}
	return false
}

func defaultTemplateContent(rel string) (string, bool) {
	for _, file := range agenttemplate.DefaultFiles() {
		if filepath.ToSlash(file.Path) == filepath.ToSlash(rel) {
			return file.Content, true
		}
	}
	return "", false
}

func normalizeTemplateText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.TrimSpace(s)
}

func memoryStarterItem(label, path string) ItemInfo {
	data, err := os.ReadFile(path)
	if err != nil {
		// Non-actionable: written on first memory. PASS (never WARN) so it
		// does not enter the computed Found-N issue summary.
		return ItemInfo{Name: label, Status: StatusPass, Note: "not yet created (written on first memory)"}
	}
	return ItemInfo{Name: label, Status: StatusPass, Note: fmt.Sprintf("%d chars", len(strings.TrimSpace(string(data))))}
}
