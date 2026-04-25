package site

import (
	"encoding/json"
	"html/template"
	"strconv"
)

func binarySizeMB() string {
	if len(benchmarksJSON) == 0 {
		return "17"
	}
	var data struct {
		Binary struct {
			SizeMB string `json:"size_mb"`
		} `json:"binary"`
	}
	if err := json.Unmarshal(benchmarksJSON, &data); err != nil {
		return "17"
	}
	return data.Binary.SizeMB
}

func binarySizeMBFloat() float64 {
	if len(benchmarksJSON) == 0 {
		return 17.0
	}
	var data struct {
		Binary struct {
			SizeMB string `json:"size_mb"`
		} `json:"binary"`
	}
	if err := json.Unmarshal(benchmarksJSON, &data); err != nil {
		return 17.0
	}
	size, _ := strconv.ParseFloat(data.Binary.SizeMB, 64)
	return size
}

type NavLink struct {
	Label string
	Href  string
}

type Link struct {
	Label string
	Href  string
}

type InstallStep struct {
	Label   string
	Command string
}

type FeatureCard struct {
	Title string
	Body  string
}

// AudienceCard is one "Who Gormes is for" persona row in the audience
// section. Title is a short noun-phrase; Body is one sentence of
// concrete framing so a visitor can self-identify quickly.
type AudienceCard struct {
	Title string
	Body  string
}

// RoadmapItem is one sub-phase or work item inside a RoadmapPhase.
// Icon is the glyph shown at the start of the row — "✓" (shipped),
// "⏳" (pending), or "◌" (ongoing polish).
// Tone is the CSS-class suffix used by .roadmap-item-<tone>.
// Label is typed as template.HTML so that + and · characters render
// literally (html/template would otherwise escape + to &#43;). Must
// not carry user input; DefaultPage is the only writer.
type RoadmapItem struct {
	Icon  string
	Tone  string
	Label template.HTML
}

// RoadmapPhase groups sub-phase items under one phase header.
// StatusLabel is the pill text, e.g. "SHIPPED · EVOLVING" or
// "IN PROGRESS · 3/7" — picked to convey both the state and the
// shipped-count so visitors see granularity without hunting.
// StatusTone is the CSS-class suffix used by .roadmap-status-<tone>.
// Subtitle is optional one-line context shown below the title.
type RoadmapPhase struct {
	StatusLabel string
	StatusTone  string
	Title       string
	Subtitle    string
	Items       []RoadmapItem
}

type LandingPage struct {
	Title               string
	Description         string
	Nav                 []NavLink
	FooterNav           []NavLink
	HeroKicker          string
	HeroHeadline        string
	// HeroSubheadLines is rendered as a stack of short paragraphs —
	// three tight lines instead of one dense block, so the operations
	// pitch reads as a punch on mobile rather than a wall of prose.
	HeroSubheadLines []string
	HeroFilterLine   string
	HeroStatusLine   string
	PrimaryCTA       Link
	SecondaryCTA     Link
	InstallSteps        []InstallStep
	InstallFootnote     string
	InstallFootnoteLink string
	InstallFootnoteHref string
	DocsNote            string
	DocsLinkLabel       string
	DocsLinkHref        string

	// "Why Gormes" section: manifesto + pain frame + fix cards.
	// All three sub-blocks render under a single #why section so the
	// reader gets identity → problem → solution in one visual unit.
	WhyLabel            string
	WhyManifestoLine    string
	WhyManifestoBullets []string
	WhyPainHeadline     string
	WhyPainBullets      []string
	WhyFixSubhead       string
	FeatureCards        []FeatureCard

	// "Who Gormes is for" — audience filter section. Three personas
	// aimed at production-agent operators, not AI tinkerers.
	AudienceLabel    string
	AudienceHeadline string
	AudienceCards    []AudienceCard

	// Roadmap section: summary block (current focus + next milestone)
	// up top, then the full phase-by-phase checklist behind a <details>
	// disclosure. RoadmapPhases comes from progress.json via
	// buildRoadmapPhases — that wiring is unchanged.
	RoadmapLabel          string
	RoadmapHeadline       string
	RoadmapCurrentFocus   []string
	RoadmapNextMilestone  string
	RoadmapDetailsSummary string
	RoadmapPhases         []RoadmapPhase
	ProgressTracker       string
	ProgressTrackerURL    string

	// FooterLeft is typed as template.HTML so it can carry the anchor
	// tag linking to the TrebuchetDynamics company site. Must not
	// carry user input; DefaultPage is the only writer.
	FooterLeft  template.HTML
	FooterRight string
}

func DefaultPage() LandingPage {
	return LandingPage{
		Title:       "Gormes — One Go Binary. No Python. No Drift.",
		Description: "A Go-native runtime for AI agents — one static binary, no Python, no virtualenvs. Built for developers who care about reliability over polish. Under construction.",
		Nav: []NavLink{
			{Label: "Install", Href: "#install"},
			{Label: "Roadmap", Href: "#roadmap"},
			{Label: "GitHub", Href: "https://github.com/TrebuchetDynamics/gormes-agent"},
		},
		FooterNav: []NavLink{
			{Label: "Why Gormes", Href: "#why"},
			{Label: "Who it's for", Href: "#audience"},
			{Label: "Docs", Href: "https://docs.gormes.ai/"},
			{Label: "Company", Href: "https://trebuchetdynamics.com/"},
		},
		HeroKicker:   "§ 01 · OPEN SOURCE · MIT · UNDER CONSTRUCTION",
		HeroHeadline: "One Go Binary. No Python. No Drift.",
		HeroSubheadLines: []string{
			"Gormes is a Go-native runtime for AI agents.",
			"Built to solve the operations problem — not the AI problem.",
			"One static binary. No virtualenvs. No dependency hell.",
		},
		HeroFilterLine: "Early-stage. Built for developers who care about reliability over polish.",
		HeroStatusLine: "Hermes is no longer required. The full Go runtime is still under active construction.",
		PrimaryCTA:     Link{Label: "Install", Href: "#install"},
		SecondaryCTA:   Link{Label: "View Source", Href: "https://github.com/TrebuchetDynamics/gormes-agent"},
		InstallSteps: []InstallStep{
			{Label: "1. UNIX / MACOS / TERMUX", Command: "curl -fsSL https://gormes.ai/install.sh | sh"},
			{Label: "2. WINDOWS POWERSHELL", Command: "irm https://gormes.ai/install.ps1 | iex"},
			{Label: "3. RUN", Command: "gormes"},
		},
		InstallFootnote:     "Installs a prebuilt static binary. Rerun the installer to update.",
		InstallFootnoteLink: "Source-backed installer is temporary during early development →",
		InstallFootnoteHref: "https://github.com/TrebuchetDynamics/gormes-agent/tree/main/scripts",
		DocsNote:            "Deeper reference material lives at",
		DocsLinkLabel:       "docs.gormes.ai →",
		DocsLinkHref:        "https://docs.gormes.ai/",
		WhyLabel:            "§ 02 · WHY GORMES",
		WhyManifestoLine:    "Gormes is not about smarter agents.",
		WhyManifestoBullets: []string{
			"It's about agents that don't fail to install.",
			"It's about agents that don't drift between environments.",
			"It's about agents that don't crash after six hours.",
			"It's about agents that don't lose work on dropped connections.",
		},
		WhyPainHeadline: "Why Hermes-stack agents break in production.",
		WhyPainBullets: []string{
			"Python environments drift between dev, staging, and prod.",
			"npm and Nix builds break silently on host package skew.",
			"Multi-process Python orchestration crashes or hangs under load.",
			"SSE streams drop on flaky networks and kill long-running agents.",
			"Debugging a single failure spans Python, Node, and OS runtimes.",
		},
		WhyFixSubhead: "How Gormes fixes it.",
		FeatureCards: []FeatureCard{
			{Title: "Single Static Binary", Body: "Zero CGO. ~" + binarySizeMB() + " MB. scp it to Termux, Alpine, a fresh VPS — it runs. No Python, no virtualenv, no Nix."},
			{Title: "No Runtime Drift", Body: "Pure Go. No pip, no npm, no env activation. The binary you tested is the binary that deploys."},
			{Title: "Streams That Don't Drop", Body: "Route-B reconnect treats SSE drops as recoverable, not fatal. Your agent doesn't lose work to a flaky network."},
			{Title: "Local Validation", Body: "gormes doctor --offline checks tool schemas before you burn tokens. Catch bad wiring before a model round-trip."},
		},
		AudienceLabel:    "§ 03 · WHO GORMES IS FOR",
		AudienceHeadline: "Production-runtime concerns, not AI demos.",
		AudienceCards: []AudienceCard{
			{Title: "Operators of long-running agents", Body: "You need agents that survive restarts, network blips, and host upgrades — not just impressive demos."},
			{Title: "Developers tired of Python/Nix/npm breakage", Body: "You're tired of an agent that worked yesterday breaking today because a transitive dep ticked over."},
			{Title: "Builders who want one binary that just runs", Body: "You'd rather scp one file to a Termux session or Alpine VPS than reproduce a virtualenv."},
		},
		RoadmapLabel:    "§ 04 · BUILD STATE",
		RoadmapHeadline: "What works today, and what's still being wired up.",
		RoadmapCurrentFocus: []string{
			"Gateway stability — Slack shared runtime, WhatsApp, WeChat adapters.",
			"Memory system — SQLite + FTS5 lattice, ontological graph, neural recall.",
			"Brain transplant — replacing the Hermes runtime with a Go-native agent loop.",
		},
		RoadmapNextMilestone:  "Fully independent Go-native brain — agent orchestration with no Hermes backend.",
		RoadmapDetailsSummary: "View full phase-by-phase checklist",
		ProgressTracker:       progressTrackerLabel(),
		ProgressTrackerURL:    "https://docs.gormes.ai/building-gormes/architecture_plan/",
		RoadmapPhases:         buildRoadmapPhases(loadEmbeddedProgress()),
		FooterLeft:            `Gormes v0.1.0 · <a href="https://trebuchetdynamics.com/">TrebuchetDynamics</a>`,
		FooterRight:           "MIT License · 2026",
	}
}
