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
	HeroKicker          string
	HeroHeadline        string
	HeroSubhead         string
	HeroImage           string
	PrimaryCTA          Link
	SecondaryCTA        Link
	InstallSteps        []InstallStep
	InstallFootnote     string
	InstallFootnoteLink string
	InstallFootnoteHref string
	DocsNote            string
	DocsLinkLabel       string
	DocsLinkHref        string
	FeaturesLabel       string
	FeaturesHeadline    string
	FeatureCards        []FeatureCard
	RoadmapLabel        string
	RoadmapHeadline     string
	RoadmapPhases       []RoadmapPhase
	ProgressTracker     string
	ProgressTrackerURL  string
	// FooterLeft is typed as template.HTML so it can carry the anchor
	// tag linking to the TrebuchetDynamics company site. Must not
	// carry user input; DefaultPage is the only writer.
	FooterLeft  template.HTML
	FooterRight string
}

func DefaultPage() LandingPage {
	return LandingPage{
		Title:       "Gormes — One Go Binary. No Python. No Drift.",
		Description: "Go-native rewrite of Hermes Agent. One static binary, no Python, no virtualenvs. Built for agents that don't crash, drift, or fail to deploy. Hermes-free. Under construction.",
		Nav: []NavLink{
			{Label: "Install", Href: "#install"},
			{Label: "Features", Href: "#features"},
			{Label: "Docs", Href: "https://docs.gormes.ai/"},
			{Label: "Roadmap", Href: "#roadmap"},
			{Label: "GitHub", Href: "https://github.com/TrebuchetDynamics/gormes-agent"},
			{Label: "Company", Href: "https://trebuchetdynamics.com/"},
		},
		HeroKicker:   "§ 01 · OPEN SOURCE · MIT LICENSE · UNDER CONSTRUCTION",
		HeroHeadline: "One Go Binary. No Python. No Drift.",
		HeroSubhead:  "Gormes is a Go-native rewrite of Hermes Agent — built to solve the operations problem, not the AI problem. One static binary, no virtualenvs, no dependency hell. Hermes is no longer required. The Go-native runtime that replaces it is still under active construction — not production-ready yet.",
		HeroImage:    "/static/go-gopher-bear-lowpoly.png",
		PrimaryCTA:   Link{Label: "Install", Href: "#install"},
		SecondaryCTA: Link{Label: "View Source", Href: "https://github.com/TrebuchetDynamics/gormes-agent"},
		InstallSteps: []InstallStep{
			{Label: "1. UNIX / MACOS / TERMUX", Command: "curl -fsSL https://gormes.ai/install.sh | sh"},
			{Label: "2. WINDOWS POWERSHELL", Command: "irm https://gormes.ai/install.ps1 | iex"},
			{Label: "3. RUN", Command: "gormes"},
		},
		InstallFootnote:     "Rerun the installer to update the managed Gormes checkout.",
		InstallFootnoteLink: "Source-backed for now →",
		InstallFootnoteHref: "https://github.com/TrebuchetDynamics/gormes-agent/tree/main/scripts",
		DocsNote:            "Deeper reference material lives at",
		DocsLinkLabel:       "docs.gormes.ai →",
		DocsLinkHref:        "https://docs.gormes.ai/",
		FeaturesLabel:       "§ 02 · WHY GORMES",
		FeaturesHeadline:    "Why Hermes breaks in production — and how Gormes fixes it.",
		FeatureCards: []FeatureCard{
			{Title: "Single Static Binary", Body: "Zero CGO. ~" + binarySizeMB() + " MB. scp it to Termux, Alpine, a fresh VPS — it runs. No Python, no virtualenv, no Nix."},
			{Title: "No Runtime Drift", Body: "Pure Go. No pip, no npm, no env activation. The binary you tested is the binary that deploys."},
			{Title: "Streams That Don't Drop", Body: "Route-B reconnect treats SSE drops as recoverable, not fatal. Your agent doesn't lose work to a flaky network."},
			{Title: "Local Validation", Body: "gormes doctor --offline checks tool schemas before you burn tokens. Catch bad wiring before a model round-trip."},
		},
		RoadmapLabel:       "§ 03 · BUILD STATE",
		RoadmapHeadline:    "What works today, and what's still being wired up.",
		ProgressTracker:    progressTrackerLabel(),
		ProgressTrackerURL: "https://docs.gormes.ai/building-gormes/architecture_plan/",
		RoadmapPhases:      buildRoadmapPhases(loadEmbeddedProgress()),
		FooterLeft:  `Gormes v0.1.0 · <a href="https://trebuchetdynamics.com/">TrebuchetDynamics</a>`,
		FooterRight: "MIT License · 2026",
	}
}
