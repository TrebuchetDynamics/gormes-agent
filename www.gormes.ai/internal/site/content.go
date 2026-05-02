package site

import (
	"encoding/json"
	"html/template"
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
	Title        string
	Description  string
	Nav          []NavLink
	HeroKicker   string
	HeroHeadline string
	HeroLines    []string
	// HeroFilterStamp + HeroFilterLine: the stamp ("Early-stage.") reads
	// as identity in accent-colored mono caps; the body line below
	// carries the filter caveat in muted body color.
	HeroFilterStamp     string
	HeroFilterLine      string
	PrimaryCTA          Link
	SecondaryCTA        Link
	InstallHeadline     string
	InstallIntro        string
	InstallSteps        []InstallStep
	InstallFootnote     string
	InstallFootnoteLink string
	InstallFootnoteHref string
	DocsNote            string
	DocsLinkLabel       string
	DocsLinkHref        string

	// "Why Gormes" section: pain frame + technical fix cards.
	WhyLabel        string
	WhyPainHeadline string
	WhyPainIntro    string
	WhyPainBullets  []string
	// WhyFixSubhead introduces the fix cards as a distinct sub-block
	// within the Why-Gormes section. v19 split the previous combined
	// "Why Hermes breaks in production — and how Gormes fixes it."
	// into two scannable headers: pain block has its own headline,
	// fix cards have this subhead.
	WhyFixSubhead string
	FeatureCards  []FeatureCard

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

	// Proof strip: credibility signals (binary size, platforms, license).
	ProofStrip []string

	// Demo section: terminal block showing the tool in action.
	DemoHeadline string
	DemoIntro    string
	DemoCommand  string
	DemoCTA      Link

	// BuiltFor section: operator-focused capabilities.
	BuiltForHeadline string
	BuiltForItems    []string

	// Explore section: links to docs, quickstart, architecture, GitHub.
	ExploreHeadline string
	ExploreLinks    []Link

	// FinalCTA section: last-chance conversion before footer.
	FinalCTAHeadline  string
	FinalCTABody      string
	FinalPrimaryCTA   Link
	FinalSecondaryCTA Link

	FooterNav []NavLink
	// FooterLeft is typed as template.HTML so it can carry the anchor
	// tag linking to the TrebuchetDynamics company site. Must not
	// carry user input; DefaultPage is the only writer.
	FooterLeft  template.HTML
	FooterRight template.HTML
}

func DefaultPage() LandingPage {
	return LandingPage{
		Title:       "Gormes — Go-Native Agents Without Python Runtime Drift",
		Description: "Gormes runs the TUI, doctor, provider turns, memory, dashboard, and configured gateways from one Go-native binary. No Python runtime, virtualenv repair, or running Hermes backend on the offline path.",
		Nav: []NavLink{
			{Label: "Docs", Href: "https://docs.gormes.ai/"},
			{Label: "Roadmap", Href: "#roadmap"},
			{Label: "GitHub", Href: "https://github.com/TrebuchetDynamics/gormes-agent"},
		},
		HeroKicker:   "§ 01 · OPEN SOURCE · MIT LICENSE · EARLY SCOUT RELEASE",
		HeroHeadline: "Run Agents From One Go Binary.",
		HeroLines: []string{
			"Gormes is a Go-native runtime for long-running AI agents.",
			"One static binary for the TUI, doctor, provider turns, memory, dashboard, and gateways.",
			"No Python runtime. No virtualenv repair. No running Hermes backend.",
		},
		HeroFilterStamp: "Scout release. Useful today, still early.",
		HeroFilterLine:  "Offline TUI, doctor --offline, provider one-shots, Goncho memory, dashboard, and configured Telegram/Discord/Slack gateway paths have implementation and tests.",
		PrimaryCTA:      Link{Label: "Build from source", Href: "#install"},
		SecondaryCTA:    Link{Label: "View on GitHub", Href: "https://github.com/TrebuchetDynamics/gormes-agent"},
		InstallHeadline: "Build from source. Prove it offline.",
		InstallIntro:    "The recommended trust path is boring: inspect the repository, build the binary, run the offline TUI, then run diagnostics before adding credentials. No runtime Node or npm is required.",
		InstallSteps: []InstallStep{
			{Label: "1. BUILD FROM SOURCE", Command: "git clone https://github.com/TrebuchetDynamics/gormes-agent.git\ncd gormes-agent\nmake build"},
			{Label: "2. OFFLINE TUI", Command: "./bin/gormes --offline"},
			{Label: "3. LOCAL DOCTOR", Command: "./bin/gormes doctor --offline"},
			{Label: "4. MEMORY AUDIT", Command: "./bin/gormes goncho doctor --json"},
			{Label: "5. MODEL-BACKED TURN", Command: "GORMES_ENDPOINT=\"https://your-provider.example/v1\" \\\nGORMES_API_KEY=\"...\" \\\nGORMES_MODEL=\"your-model\" \\\n./bin/gormes --oneshot \"hello from Gormes\""},
			{Label: "6. INSPECT INSTALLER", Command: "curl -fsSLO https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.sh\nless install.sh\nsh install.sh"},
		},
		InstallFootnote:     "The installer manages a source checkout, builds gormes, and links it into your PATH. The README keeps the inspect-first path explicit.",
		InstallFootnoteLink: "Read the installer source →",
		InstallFootnoteHref: "https://github.com/TrebuchetDynamics/gormes-agent/tree/main/scripts",
		DocsLinkLabel:       "docs.gormes.ai →",
		DocsLinkHref:        "https://docs.gormes.ai/",
		WhyLabel:            "§ 02 · WHY GORMES",
		WhyPainHeadline:     "Why agent runtimes fail in production",
		WhyPainIntro:        "Python-stack agents fail operationally when:",
		WhyFixSubhead:       "How Gormes fixes the runtime surface",
		WhyPainBullets: []string{
			"dev, staging, and prod stop matching",
			"virtualenvs and package wheels drift across hosts",
			"long turns die on dropped streams",
			"tool wiring fails after tokens are already burning",
		},
		FeatureCards: []FeatureCard{
			{Title: "Single Static Binary", Body: "CGO_ENABLED=0 release builds produce a ~" + binarySizeMB() + " MB artifact. Offline TUI startup does not require Python, Node, Docker, or a Hermes process."},
			{Title: "Source-First Install", Body: "Build the exact source tree you inspected. Convenience installers stay inspect-first from GitHub raw URLs instead of README-level pipe-to-shell snippets."},
			{Title: "Offline Doctor", Body: "./bin/gormes doctor --offline checks local TUI, tools, Goncho, gateway configuration, and provider endpoint readiness without contacting a model provider."},
			{Title: "Provider Turns", Body: "One-shots and the TUI use configured provider-compatible endpoints from the same runtime surface as the offline tooling."},
			{Title: "Local Goncho Memory", Body: "Session history, durable user context, diagnostics, and queue state live in local SQLite rather than external Redis or vector sidecars."},
			{Title: "Scout Limits Are Visible", Body: "Full Hermes parity, broad channel parity, voice/TTS, MCP/plugin parity, stable signed releases, and release hardening remain in progress."},
		},
		RoadmapLabel:    "§ 03 · BUILD STATE",
		RoadmapHeadline: "Useful today, still early.",
		RoadmapCurrentFocus: []string{
			"Offline TUI, doctor diagnostics, provider-backed one-shots, and dashboard",
			"Configured Telegram and Discord gateways; Slack when complete credentials are present",
			"Goncho memory, Go-native tool registry, and subagent safety",
		},
		RoadmapNextMilestone:  "Production-stable Go-native runtime with signed releases and broader Hermes parity",
		RoadmapDetailsSummary: "View full phase-by-phase checklist",
		ProgressTracker:       progressTrackerLabel(),
		ProgressTrackerURL:    "https://docs.gormes.ai/building-gormes/architecture_plan/",
		RoadmapPhases:         buildRoadmapPhases(loadEmbeddedProgress()),
		ProofStrip: []string{
			"~" + binarySizeMB() + " MB static binary",
			"Source build recommended",
			"MIT License",
			"Scout release",
		},
		DemoHeadline:     "See it work in 30 seconds",
		DemoIntro:        "Build from source, run the offline TUI, and verify your local runtime before touching a model.",
		DemoCommand:      "git clone https://github.com/TrebuchetDynamics/gormes-agent.git\ncd gormes-agent\nmake build\n./bin/gormes --offline\n./bin/gormes doctor --offline",
		DemoCTA:          Link{Label: "Read the quickstart →", Href: "https://docs.gormes.ai/using-gormes/quickstart/"},
		BuiltForHeadline: "What works today",
		BuiltForItems: []string{
			"Offline TUI without Python, Node, Docker, provider credentials, or Hermes",
			"Provider-backed one-shots from one binary",
			"Local Goncho SQLite memory and session export",
			"Configured Telegram and Discord gateways; Slack with complete Socket Mode credentials",
			"htmx dashboard for sessions, config, skills, and logs",
		},
		ExploreHeadline: "Explore",
		ExploreLinks: []Link{
			{Label: "Quickstart", Href: "https://docs.gormes.ai/using-gormes/quickstart/"},
			{Label: "Install", Href: "https://docs.gormes.ai/using-gormes/install/"},
			{Label: "Configuration", Href: "https://docs.gormes.ai/using-gormes/configuration/"},
			{Label: "Architecture", Href: "https://docs.gormes.ai/building-gormes/architecture_plan/"},
			{Label: "GitHub", Href: "https://github.com/TrebuchetDynamics/gormes-agent"},
		},
		FinalCTAHeadline:  "Build it. Prove it offline. Add credentials later.",
		FinalCTABody:      "The offline path proves the runtime, diagnostics, tools, memory, and configured gateway readiness before provider calls or token spend.",
		FinalPrimaryCTA:   Link{Label: "Build from source", Href: "#install"},
		FinalSecondaryCTA: Link{Label: "Star on GitHub", Href: "https://github.com/TrebuchetDynamics/gormes-agent"},
		FooterNav: []NavLink{
			{Label: "Docs", Href: "https://docs.gormes.ai/"},
			{Label: "Company", Href: "https://trebuchetdynamics.com/"},
		},
		FooterLeft:  `Gormes 0.2.0-scout · <a href="https://trebuchetdynamics.com/">TrebuchetDynamics</a>`,
		FooterRight: "MIT License · 2026",
	}
}
