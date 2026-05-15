package site

import (
	"encoding/json"
	"html/template"
	"strconv"
	"strings"
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

func runtimeRSSMB() string {
	if len(benchmarksJSON) == 0 {
		return ""
	}
	var data struct {
		Runtime struct {
			OfflineDoctor struct {
				Status    string  `json:"status"`
				PeakRSSMB float64 `json:"peak_rss_mb"`
			} `json:"offline_doctor"`
		} `json:"runtime"`
	}
	if err := json.Unmarshal(benchmarksJSON, &data); err != nil {
		return ""
	}
	if data.Runtime.OfflineDoctor.Status != "measured" || data.Runtime.OfflineDoctor.PeakRSSMB <= 0 {
		return ""
	}
	return formatLandingFloat(data.Runtime.OfflineDoctor.PeakRSSMB)
}

func runtimeRSSProof() string {
	if rss := runtimeRSSMB(); rss != "" {
		return "doctor RSS ~" + rss + " MB"
	}
	return "doctor RSS measured"
}

func formatLandingFloat(value float64) string {
	text := strconv.FormatFloat(value, 'f', 1, 64)
	text = strings.TrimRight(text, "0")
	text = strings.TrimRight(text, ".")
	return text
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
		Title:       "Gormes — AI Agents From One Go Binary",
		Description: "Gormes runs AI agents as a single static Go binary. Start offline, prove the machine works, then add provider and gateway credentials.",
		Nav: []NavLink{
			{Label: "Docs", Href: "https://docs.gormes.ai/"},
			{Label: "Roadmap", Href: "#roadmap"},
			{Label: "GitHub", Href: "https://github.com/TrebuchetDynamics/gormes-agent"},
		},
		HeroKicker:   "OPEN SOURCE · MIT LICENSE · EARLY SCOUT RELEASE",
		HeroHeadline: "Run Agents From One Go Binary.",
		HeroLines: []string{
			"Gormes runs AI agents as a single static binary.",
			"No Python runtime. No virtualenv repair. No backend service just to open the UI.",
			"Start offline, prove the machine works, then add provider and gateway credentials.",
		},
		HeroFilterStamp: "Scout release. Useful today, still early.",
		HeroFilterLine:  "Offline TUI, doctor diagnostics, provider one-shots, Goncho memory, dashboard, and configured Telegram/Discord/Slack paths are covered. Full parity is still hardening.",
		PrimaryCTA:      Link{Label: "Install", Href: "#install"},
		SecondaryCTA:    Link{Label: "View on GitHub", Href: "https://github.com/TrebuchetDynamics/gormes-agent"},
		InstallHeadline: "Install first. Build from source when needed.",
		InstallIntro:    "Use install.sh for the release-first managed install. Build from source when you need local inspection, custom flags, or unsupported platform fallback. The first proof does not need credentials, a model call, Python, Docker, or Hermes.",
		InstallSteps: []InstallStep{
			{Label: "1. INSTALL.SH", Command: "curl -fsSL https://github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh | sh\ngormes --version\ngormes doctor --offline"},
			{Label: "2. BUILD FROM SOURCE", Command: "git clone https://github.com/TrebuchetDynamics/gormes-agent.git\ncd gormes-agent\nmkdir -p bin\nCGO_ENABLED=0 go build -trimpath -o bin/gormes ./cmd/gormes\n./bin/gormes doctor --offline\n./bin/gormes --offline"},
		},
		InstallFootnote:     "Use install.sh for the published gormes command on PATH, or ./bin/gormes from a source checkout when you are developing Gormes itself.",
		InstallFootnoteLink: "Read the install docs ->",
		InstallFootnoteHref: "https://docs.gormes.ai/using-gormes/install/",
		DocsLinkLabel:       "docs.gormes.ai →",
		DocsLinkHref:        "https://docs.gormes.ai/",
		WhyLabel:            "WHY GORMES",
		WhyPainHeadline:     "Python-stack agents fail for boring reasons.",
		WhyPainIntro:        "The model is not usually the fragile part. Operations are:",
		WhyFixSubhead:       "Gormes cuts out that failure class",
		WhyPainBullets: []string{
			"dev, staging, and prod stop matching",
			"virtualenvs and package wheels drift across hosts",
			"long turns die on dropped streams",
			"tool wiring fails after tokens are already burning",
		},
		FeatureCards: []FeatureCard{
			{Title: "Single Static Binary", Body: "CGO_ENABLED=0 release builds produce a ~" + binarySizeMB() + " MB artifact for the runtime surface."},
			{Title: "Offline Proof", Body: "./bin/gormes --offline starts the native TUI without credentials, network calls, Python, Node, Docker, or Hermes."},
			{Title: "Built-In Doctor", Body: "./bin/gormes doctor --offline checks local readiness before provider calls or token spend; the benchmark mirror records peak RSS for that path."},
			{Title: "Provider Turns", Body: "One-shots and the TUI use configured provider-compatible endpoints from the same binary."},
			{Title: "Local Goncho Memory", Body: "Sessions, durable context, diagnostics, and queue state stay in local SQLite."},
			{Title: "Visible Limits", Body: "Full Hermes parity, broad channel parity, voice/TTS, MCP/plugin parity, and release hardening remain in progress."},
		},
		RoadmapLabel:    "BUILD STATE",
		RoadmapHeadline: "Useful today, still early.",
		RoadmapCurrentFocus: []string{
			"Offline TUI, doctor diagnostics, provider one-shots, dashboard, and Goncho memory",
			"Configured Telegram and Discord gateways; Slack with complete Socket Mode credentials",
			"Go-native tool registry, web/browser tools, and subagent safety",
		},
		RoadmapNextMilestone:  "Production-stable Go-native runtime with signed releases and broader Hermes parity",
		RoadmapDetailsSummary: "View full phase-by-phase checklist",
		ProgressTracker:       progressTrackerLabel(),
		ProgressTrackerURL:    "https://docs.gormes.ai/building-gormes/architecture_plan/",
		RoadmapPhases:         buildRoadmapPhases(loadEmbeddedProgress()),
		ProofStrip: []string{
			"~" + binarySizeMB() + " MB static binary",
			runtimeRSSProof(),
			"Source build recommended",
			"MIT License",
			"Scout release",
		},
		BuiltForHeadline: "What works today",
		BuiltForItems: []string{
			"Run a local agent UI with zero runtime dependencies on the offline path",
			"Send one-shot prompts to a provider-compatible endpoint",
			"Validate your environment before spending tokens",
			"Operate configured Telegram, Discord, or Slack agents from one binary",
			"Inspect and debug agent memory locally with Goncho",
			"Browse sessions, config, skills, and logs in the local dashboard",
		},
		ExploreHeadline: "Explore",
		ExploreLinks: []Link{
			{Label: "Quickstart", Href: "https://docs.gormes.ai/using-gormes/quickstart/"},
			{Label: "Install", Href: "https://docs.gormes.ai/using-gormes/install/"},
			{Label: "Configuration", Href: "https://docs.gormes.ai/using-gormes/configuration/"},
			{Label: "Architecture", Href: "https://docs.gormes.ai/building-gormes/architecture_plan/"},
			{Label: "GitHub", Href: "https://github.com/TrebuchetDynamics/gormes-agent"},
		},
		FinalCTAHeadline:  "Start offline. Add credentials later.",
		FinalCTABody:      "The offline path proves the runtime before provider calls, gateway traffic, or token spend.",
		FinalPrimaryCTA:   Link{Label: "Build from source", Href: "#install"},
		FinalSecondaryCTA: Link{Label: "Star on GitHub", Href: "https://github.com/TrebuchetDynamics/gormes-agent"},
		FooterNav: []NavLink{
			{Label: "Docs", Href: "https://docs.gormes.ai/"},
			{Label: "Company", Href: "https://trebuchetdynamics.com/"},
		},
		FooterLeft:  `Gormes 0.1.0 · <a href="https://trebuchetdynamics.com/">TrebuchetDynamics</a>`,
		FooterRight: "MIT License · 2026",
	}
}
