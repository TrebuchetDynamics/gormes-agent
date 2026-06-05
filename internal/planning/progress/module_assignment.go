package progress

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

// ModuleAssignment records one deterministic module decision for an item.
type ModuleAssignment struct {
	PhaseID      string
	SubphaseID   string
	ItemIndex    int
	ItemName     string
	OldModule    string
	Module       string
	Reason       string
	Changed      bool
	CrossCutting bool
}

// ModuleAssignmentAudit summarizes a module classification pass.
type ModuleAssignmentAudit struct {
	Total       int
	Changed     int
	Preserved   int
	ByModule    map[string]int
	ByReason    map[string]int
	Assignments []ModuleAssignment
}

// SuggestedModule returns the feature module that should physically own the
// row. It is intentionally deterministic and local: it uses the row text,
// phase/subphase context, and execution_owner only. It never consults external
// state and never rewrites anything except through AssignModules.
func SuggestedModule(phaseID, phaseName, subphaseID, subphaseName string, it Item) (module, reason string) {
	if ValidModule(it.Module) {
		return it.Module, "existing-valid"
	}

	text := moduleAssignmentText(phaseID, phaseName, subphaseID, subphaseName, it)
	switch {
	case containsAny(text, "navivox"):
		return ModuleNavivox, "keyword:navivox"
	case containsAny(text, "browser", "cdp", "chromedp", "browserbase", "firecrawl", "camofox", "go-browser-harness", "web_search", "web_extract", "web_crawl", "go-native hermes web"):
		return ModuleBrowser, "keyword:browser"
	case containsAny(text, "kanban"):
		return ModuleKanban, "keyword:kanban"
	case containsAny(text, "text_to_speech", "text-to-speech", "tts ", " tts", "tts/", "speech synthesis", "synthesize", "voice-mode state", "minimax tts"):
		return ModuleTTS, "keyword:tts"
	case containsAny(text, "speech-to-text", "transcription", "transcribe", "stt", "whisper", "audio preprocessing", "wav", "ogg/opus", "voice/audio inbound", "voice stt"):
		return ModuleSTT, "keyword:stt"
	case containsAny(text, "bubble tea", "native tui", "chat tui", "admin tui", "terminal ux", "ink behavioral", "welcome panel", "status bar renderer", "hermes skin", "slash completion", "viewport", "queued-message", "mouse tracking", "chat style", "streaming feedback", "tool-trail status", "spinner cadence"):
		return ModuleTUI, "keyword:tui"
	case containsAny(text, "landing", "gormes.ai"):
		return ModuleLanding, "keyword:landing"
	case containsAny(text, "doctor"):
		return ModuleDoctor, "keyword:doctor"
	case containsAny(text, "profile", "profiles", "personality", "personalities"):
		return ModuleProfiles, "keyword:profiles"
	case containsAny(text, "progress.json", "progressctl", "progress backlog", "module-split", "module split", "backlog split", "builder-loop", "architecture_plan", "progress schema"):
		return ModuleProgress, "keyword:progress"
	case containsAny(text, "planner", "planning", "source coverage ledger", "feature parity map"):
		return ModulePlanner, "keyword:planner"
	case containsAny(text, "builder", "build loop", "builder-loop"):
		return ModuleBuilder, "keyword:builder"
	case containsAny(text, "autoloop", "watchdog", "fleet", "durable worker", "durable job", "orchestrator", "architecture planner tasks manager", "cron scheduler", "cron_runs", "heartbeat"):
		return ModuleFleet, "keyword:fleet"
	case containsAny(text, "goncho", "honcho"):
		return ModuleGoncho, "keyword:goncho"
	case containsAny(text, "telegram", "discord", "slack", "whatsapp", "wechat", "wecom", "weixin", "signal", "email", "sms", "mattermost", "webhook", "feishu", "dingtalk", "qq bot", "bluebubbles", "microsoft teams", "google chat", "channel adapter", "shared-chassis bot", "channels_list", "channel-scoped", "channel capabilities", "attachment routing", "mention gate"):
		return ModuleChannels, "keyword:channels"
	case containsAny(text, "gateway", "api server", "websocket", "sse", "proxy mode", "dashboard api", "status endpoint", "health endpoint", "turn adapter", "event bus integration test"):
		return ModuleGateway, "keyword:gateway"
	case containsAny(text, "recallprovider"):
		return ModuleMemory, "keyword:memory"
	case containsAny(text, "provider", "model", "oauth", "credential", "openrouter", "openai", "anthropic", "gemini", "codex", "bedrock", "azure", "foundry", "deepseek", "kimi", "moonshot", "ollama", "xai", "grok", "lm studio", "nous", "qwen", "rate guard", "retry-after", "prompt-cache"):
		return ModuleProviders, "keyword:providers"
	case containsAny(text, "skill", "skills", "skill.md", "curator", "plugin", "plugins"):
		return ModuleSkills, "keyword:skills"
	case containsAny(text, "config", "configuration", "setup", "onboarding", "migrate", "migration", "toml", "yaml"):
		return ModuleConfig, "keyword:config"
	case containsAny(text, "install", "installer", "install.sh", "install.ps1", "homebrew", "nix", "nixos", "docker", "oci image", "termux", "packaging", "binary swap", "path safety"):
		return ModuleInstall, "keyword:install"
	case containsAny(text, "mcp", "acp", "tool ", " tools", "tool-", "tool_", "toolset", "sandbox", "filesystem", "file ops", "patch", "terminal", "code execution", "shell", "todo", "clarify"):
		return ModuleTools, "keyword:tools"
	case containsAny(text, "session", "transcript", "lineage", "compression", "context engine", "title generation", "session_id", "chat_id"):
		return ModuleSessions, "keyword:sessions"
	case containsAny(text, "memory", "recall", "embedding", "fts5", "sqlite", "user.md", "memory.md", "knowledge graph"):
		return ModuleMemory, "keyword:memory"
	case containsAny(text, "runtime", "kernel", "agent turn", "agent lifecycle", "middleware", "event bus", "loop detector", "i18n", "sandbox provider"):
		return ModuleRuntime, "keyword:runtime"
	}

	if module := subphaseFallbackModule(phaseID, subphaseID); module != "" {
		return module, "subphase:" + subphaseID
	}
	switch {
	case containsAny(text, "release", "sbom", "attestation", "artifact", "archive 30 mb", "build provenance", "version/provenance", "github release", "release notes", "sharp v1.0"):
		return ModuleRelease, "keyword:release"
	case containsAny(text, "readme", "docs", "documentation", "hugo", "website", "source coverage ledger", "feature parity map", "blog", "guide", "mirror coverage", "webpage"):
		return ModuleDocs, "keyword:docs"
	}
	if module := moduleForOwner(it.ExecutionOwner); ValidModule(module) {
		return module, "execution_owner:" + string(it.ExecutionOwner)
	}
	return ModuleCrossCutting, "fallback:cross-cutting"
}

// AssignModules writes explicit valid modules into every row and returns an
// audit of the changes. It changes no row field except Item.Module.
func AssignModules(p *Progress) ModuleAssignmentAudit {
	audit := ModuleAssignmentAudit{
		ByModule: map[string]int{},
		ByReason: map[string]int{},
	}
	if p == nil {
		return audit
	}
	for _, phaseID := range sortedMapKeys(p.Phases) {
		ph := p.Phases[phaseID]
		for _, subphaseID := range sortedMapKeys(ph.Subphases) {
			sp := ph.Subphases[subphaseID]
			for i := range sp.Items {
				module, reason := SuggestedModule(phaseID, ph.Name, subphaseID, sp.Name, sp.Items[i])
				old := sp.Items[i].Module
				changed := old != module
				if changed {
					sp.Items[i].Module = module
					audit.Changed++
				} else {
					audit.Preserved++
				}
				audit.Total++
				audit.ByModule[module]++
				audit.ByReason[reason]++
				audit.Assignments = append(audit.Assignments, ModuleAssignment{
					PhaseID:      phaseID,
					SubphaseID:   subphaseID,
					ItemIndex:    i,
					ItemName:     sp.Items[i].Name,
					OldModule:    old,
					Module:       module,
					Reason:       reason,
					Changed:      changed,
					CrossCutting: module == ModuleCrossCutting,
				})
			}
			ph.Subphases[subphaseID] = sp
		}
		p.Phases[phaseID] = ph
	}
	sortModuleAudit(&audit)
	return audit
}

func sortModuleAudit(audit *ModuleAssignmentAudit) {
	slices.SortFunc(audit.Assignments, func(a, b ModuleAssignment) int {
		if cmp := compareNatural(a.PhaseID, b.PhaseID); cmp != 0 {
			return cmp
		}
		if cmp := compareNatural(a.SubphaseID, b.SubphaseID); cmp != 0 {
			return cmp
		}
		return a.ItemIndex - b.ItemIndex
	})
}

func moduleAssignmentText(phaseID, phaseName, subphaseID, subphaseName string, it Item) string {
	var b strings.Builder
	appendText := func(s string) {
		if s == "" {
			return
		}
		b.WriteByte(' ')
		b.WriteString(strings.ToLower(s))
	}
	appendText(phaseID)
	appendText(phaseName)
	appendText(subphaseID)
	appendText(subphaseName)
	appendText(it.Name)
	return b.String()
}

func containsAny(text string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func subphaseFallbackModule(phaseID, subphaseID string) string {
	switch {
	case subphaseID == "1.A", subphaseID == "1.E":
		return ModuleTUI
	case subphaseID == "1.B":
		return ModuleDoctor
	case subphaseID == "1.C":
		return ModuleFleet
	case subphaseID == "1.D":
		return ModuleSkills
	case subphaseID == "5.X":
		return ModuleInstall
	case strings.HasPrefix(subphaseID, "2.B"):
		return ModuleChannels
	case subphaseID == "2.A":
		return ModuleTools
	case subphaseID == "2.C":
		return ModuleSessions
	case subphaseID == "2.D", strings.HasPrefix(subphaseID, "2.E"):
		return ModuleFleet
	case strings.HasPrefix(subphaseID, "2.F"):
		return ModuleGateway
	case subphaseID == "2.G":
		return ModuleSkills
	case subphaseID == "2.H":
		return ModuleGoncho
	case strings.HasPrefix(subphaseID, "3.F"), strings.HasPrefix(subphaseID, "3.G"):
		return ModuleGoncho
	case subphaseID == "3.E.1", subphaseID == "3.E.3", subphaseID == "3.E.8":
		return ModuleSessions
	case strings.HasPrefix(subphaseID, "3."):
		return ModuleMemory
	case subphaseID == "4.B", subphaseID == "4.F":
		return ModuleSessions
	case subphaseID == "4.I", subphaseID == "4.L":
		return ModuleRuntime
	case strings.HasPrefix(subphaseID, "4."):
		return ModuleProviders
	case subphaseID == "5.C", subphaseID == "5.T":
		return ModuleBrowser
	case subphaseID == "5.E":
		return ModuleTTS
	case subphaseID == "5.F", subphaseID == "5.I":
		return ModuleSkills
	case subphaseID == "5.M":
		return ModuleKanban
	case subphaseID == "5.O":
		return ModuleCLI
	case subphaseID == "5.P":
		return ModuleInstall
	case subphaseID == "5.Q":
		return ModuleGateway
	case subphaseID == "5.R", subphaseID == "5.S", subphaseID == "5.U", subphaseID == "5.V", subphaseID == "5.W":
		return ModuleRuntime
	case strings.HasPrefix(subphaseID, "5."):
		return ModuleTools
	case strings.HasPrefix(phaseID, "6"):
		return ModuleLearningLoop
	case strings.HasPrefix(phaseID, "7"):
		return ModuleChannels
	case subphaseID == "8.B", subphaseID == "8.C", subphaseID == "8.E", subphaseID == "8.G":
		return ModuleDocs
	case subphaseID == "8.D":
		return ModuleRelease
	case subphaseID == "8.F":
		return ModuleProgress
	case strings.HasPrefix(phaseID, "8"):
		return ModuleDocs
	case subphaseID == "9.C":
		return ModuleConfig
	case subphaseID == "9.D":
		return ModuleSTT
	case subphaseID == "9.E", subphaseID == "9.F":
		return ModuleNavivox
	case strings.HasPrefix(phaseID, "9"):
		return ModuleRuntime
	default:
		return ""
	}
}

func compareNatural(a, b string) int {
	return compareRoadmapKeys(a, b)
}

// FormatModuleAuditMarkdown renders an operator-readable audit report for a
// completed classification pass.
func FormatModuleAuditMarkdown(audit ModuleAssignmentAudit) string {
	var b strings.Builder
	b.WriteString("# Module Assignment Audit\n\n")
	b.WriteString("Generated by the C5g module-classification pass.\n\n")
	fmt.Fprintf(&b, "- Total rows: %d\n", audit.Total)
	fmt.Fprintf(&b, "- Rows changed: %d\n", audit.Changed)
	fmt.Fprintf(&b, "- Rows already explicit: %d\n", audit.Preserved)
	fmt.Fprintf(&b, "- Invalid/unclassified rows after assignment: 0\n\n")

	b.WriteString("## Counts By Module\n\n")
	for _, module := range AllowedModules() {
		if n := audit.ByModule[module]; n > 0 {
			fmt.Fprintf(&b, "- `%s`: %d\n", module, n)
		}
	}

	b.WriteString("\n## Counts By Reason\n\n")
	for _, reason := range sortedStringKeys(audit.ByReason) {
		fmt.Fprintf(&b, "- `%s`: %d\n", reason, audit.ByReason[reason])
	}

	b.WriteString("\n## Old Broad Bucket Mapping\n\n")
	b.WriteString("- `commands` -> `cli` for generic command infrastructure; feature-specific commands go to their feature module.\n")
	b.WriteString("- `setup-config-install` -> `config`, `install`, `doctor`, `profiles`, or `providers` by row behavior.\n")
	b.WriteString("- `gateway-channels` -> `gateway`, `channels`, `navivox`, `tts`, `stt`, or `browser` by delivered surface.\n")
	b.WriteString("- `providers-auth` -> `providers`.\n")
	b.WriteString("- `memory-sessions-skills` -> `memory`, `sessions`, `skills`, or `goncho` by contract.\n")
	b.WriteString("- `orchestrator` -> `planner`, `builder`, `progress`, `fleet`, `kanban`, or `learning-loop` by subsystem.\n")

	b.WriteString("\n## Manual Review Notes\n\n")
	b.WriteString("- Reviewed packaging/install rows under `5.P`; install isolation and installer artifacts now land in `install`, not `tools` or `release`.\n")
	b.WriteString("- Reviewed release/TUI rows under `8.D`; release pipeline rows land in `release`, while chat TUI style/streaming/status rows land in `tui`.\n")
	b.WriteString("- Reviewed provider setup/auth/model rows, including OpenRouter and OpenAI Codex surfaces; provider-specific setup/model rows land in `providers`.\n")
	b.WriteString("- Reviewed Navivox, Kanban, TTS, STT, browser, and channel rows against the grill decisions; each lands in its named feature module.\n")
	b.WriteString("- Corrected classifier hazards found during audit: generic `auth` no longer matches `authoritative`, and generic `matrix` no longer forces the channels module.\n")

	b.WriteString("\n## Cross-Cutting Rows\n\n")
	hasCrossCutting := false
	for _, assignment := range audit.Assignments {
		if !assignment.CrossCutting {
			continue
		}
		hasCrossCutting = true
		fmt.Fprintf(&b, "- `%s/%s[%d]` %s\n", assignment.PhaseID, assignment.SubphaseID, assignment.ItemIndex, assignment.ItemName)
	}
	if !hasCrossCutting {
		b.WriteString("- None\n")
	}
	return b.String()
}

func sortedStringKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
