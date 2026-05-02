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
		Title:       "Gormes — Run AI Agents as One Go Binary",
		Description: "A Go-native runtime for long-running AI agents: one static binary, no Python runtime, no virtualenv drift, and no Hermes process to keep alive. Early-stage and not production-stable yet.",
		Nav: []NavLink{
			{Label: "Docs", Href: "https://docs.gormes.ai/"},
			{Label: "Roadmap", Href: "#roadmap"},
			{Label: "GitHub", Href: "https://github.com/TrebuchetDynamics/gormes-agent"},
		},
		HeroKicker:   "§ 01 · OPEN SOURCE · MIT LICENSE · EARLY SCOUT RELEASE",
		HeroHeadline: "Run AI Agents as One Go Binary.",
		HeroLines: []string{
			"Gormes is a Go-native runtime for long-running AI agents.",
			"One static binary. No Python runtime. No Hermes process.",
			"Ship the same binary you test. Run it anywhere.",
		},
		HeroFilterStamp: "Early-stage and shipping.",
		HeroFilterLine:  "Offline TUI, local doctor, provider-backed one-shots, and Goncho memory are ready today.",
		PrimaryCTA:      Link{Label: "Install", Href: "#install"},
		SecondaryCTA:    Link{Label: "View on GitHub", Href: "https://github.com/TrebuchetDynamics/gormes-agent"},
		InstallIntro:    "One command to install from the public repo. Review the script first, then run it.",
		InstallSteps: []InstallStep{
			{Label: "1. INSTALL", Command: "curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.sh | sh"},
			{Label: "2. OFFLINE TUI", Command: "gormes --offline"},
			{Label: "3. LOCAL DOCTOR", Command: "gormes doctor --offline"},
			{Label: "4. MEMORY AUDIT", Command: "gormes goncho doctor --json"},
			{Label: "5. MODEL-BACKED TURN", Command: "GORMES_ENDPOINT=\"https://your-provider.example/v1\" \\\nGORMES_API_KEY=\"...\" \\\nGORMES_MODEL=\"your-model\" \\\ngormes --oneshot \"hello from Gormes\""},
			{Label: "6. BUILD FROM SOURCE", Command: "git clone https://github.com/TrebuchetDynamics/gormes-agent.git\ncd gormes-agent\nmake build"},
		},
		InstallFootnote:     "The installer clones, builds, and links gormes into your PATH. Or build from source if you prefer.",
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
			"install scripts depend on host package luck",
			"long turns die on dropped streams",
			"tool wiring fails after tokens are already burning",
		},
		FeatureCards: []FeatureCard{
			{Title: "One Binary To Ship", Body: "Zero CGO. ~" + binarySizeMB() + " MB. The same stripped static binary you test is the artifact you deploy."},
			{Title: "No Runtime Drift", Body: "No runtime Node or npm, no pip, no env activation. Gormes removes the Python-stack failure class from the shipped runtime."},
			{Title: "Recoverable Streams", Body: "Route-B reconnect treats SSE drops as recoverable events, so a flaky network does not automatically erase a long turn."},
			{Title: "Local Preflight", Body: "gormes doctor --offline checks the local runtime surface before a model round-trip. Bad wiring fails before tokens burn."},
			{Title: "Transparent Local Memory", Body: "gormes goncho doctor --json reports memory DB paths, schema state, queue status, degraded modes, and provider readiness for Goncho."},
			{Title: "Release Trust Roadmap", Body: "Package-manager manifests, checksums, detached signatures, Windows signing, and AV false-positive submission are explicit release-hardening targets."},
		},
		RoadmapLabel:    "§ 03 · BUILD STATE",
		RoadmapHeadline: "Useful today, still early.",
		RoadmapCurrentFocus: []string{
			"Offline TUI, local doctor, and provider-backed one-shots",
			"Gateway stability and shared channel contracts",
			"Goncho memory plus subagent safety",
		},
		RoadmapNextMilestone:  "Production-stable Go-native runtime, no Hermes process",
		RoadmapDetailsSummary: "View full phase-by-phase checklist",
		ProgressTracker:       progressTrackerLabel(),
		ProgressTrackerURL:    "https://docs.gormes.ai/building-gormes/architecture_plan/",
		RoadmapPhases:         buildRoadmapPhases(loadEmbeddedProgress()),
		ProofStrip: []string{
			"~" + binarySizeMB() + " MB static binary",
			"Linux · macOS · Windows",
			"MIT License",
			"Zero runtime dependencies",
		},
		DemoHeadline:     "See it work in 30 seconds",
		DemoIntro:        "Build from source, run the offline TUI, and verify your local runtime before touching a model.",
		DemoCommand:      "curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.sh | sh\ngormes --offline\ngormes doctor --offline",
		DemoCTA:          Link{Label: "Read the quickstart →", Href: "https://docs.gormes.ai/getting-started/quickstart/"},
		BuiltForHeadline: "Built for operators who ship",
		BuiltForItems: []string{
			"Static binary you can copy to any server",
			"Offline TUI for local development and testing",
			"Doctor commands that catch misconfig before tokens burn",
			"Goncho memory with transparent local storage",
			"Provider-backed one-shots for CI and automation",
		},
		ExploreHeadline: "Explore",
		ExploreLinks: []Link{
			{Label: "Quickstart", Href: "https://docs.gormes.ai/getting-started/quickstart/"},
			{Label: "Architecture", Href: "https://docs.gormes.ai/building-gormes/architecture_plan/"},
			{Label: "CLI Reference", Href: "https://docs.gormes.ai/reference/cli/"},
			{Label: "GitHub", Href: "https://github.com/TrebuchetDynamics/gormes-agent"},
		},
		FinalCTAHeadline:  "Ready to try Gormes?",
		FinalCTABody:      "Build from source in under a minute. No Python runtime. No Hermes process. One static binary.",
		FinalPrimaryCTA:   Link{Label: "Install Gormes", Href: "#install"},
		FinalSecondaryCTA: Link{Label: "Star on GitHub", Href: "https://github.com/TrebuchetDynamics/gormes-agent"},
		FooterNav: []NavLink{
			{Label: "Docs", Href: "https://docs.gormes.ai/"},
			{Label: "Company", Href: "https://trebuchetdynamics.com/"},
		},
		FooterLeft:  `Gormes v0.1.0 · <a href="https://trebuchetdynamics.com/">TrebuchetDynamics</a>`,
		FooterRight: "MIT License · 2026",
	}
}
