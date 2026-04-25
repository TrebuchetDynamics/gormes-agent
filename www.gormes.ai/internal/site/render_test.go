package site

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

func TestRenderIndex_RendersRedesignedLanding(t *testing.T) {
	body, err := RenderIndex()
	if err != nil {
		t.Fatalf("RenderIndex: %v", err)
	}

	text := string(body)
	wants := []string{
		// Hero — multi-line subhead (3 punchy lines), filter line for
		// audience self-selection, status line tucked below CTAs as
		// tertiary, no hero illustration.
		"OPEN SOURCE · MIT",
		"UNDER CONSTRUCTION",
		"One Go Binary. No Python. No Drift.",
		`class="hero-subhead-line"`,
		"Gormes is a Go-native runtime for AI agents.",
		"Built to solve the operations problem — not the AI problem.",
		"One static binary. No virtualenvs. No dependency hell.",
		`class="hero-filter"`,
		"Early-stage. Built for developers who care about reliability over polish.",
		`class="hero-status"`,
		"Hermes is no longer required. The full Go runtime is still under active construction.",
		// CTA hierarchy — primary dominant, ghost secondary.
		`class="btn btn-primary"`,
		`class="btn btn-ghost"`,
		// Install — footnote rewritten for clarity (prebuilt binary,
		// source-backed installer is temporary).
		"1. UNIX / MACOS / TERMUX",
		"curl -fsSL https://gormes.ai/install.sh | sh",
		"2. WINDOWS POWERSHELL",
		"irm https://gormes.ai/install.ps1 | iex",
		"3. RUN",
		"Installs a prebuilt static binary",
		"Source-backed installer is temporary during early development →",
		// Copy button (clipboard JS is allowed for this widget only)
		`class="copy-btn"`,
		"navigator.clipboard.writeText",
		// Why Gormes — manifesto + pain frame + fix cards under one section.
		`id="why"`,
		"WHY GORMES",
		"Gormes is not about smarter agents.",
		"It&#39;s about agents that don&#39;t fail to install.",
		"It&#39;s about agents that don&#39;t crash after six hours.",
		"Why Hermes-stack agents break in production.",
		"Python environments drift between dev, staging, and prod.",
		"SSE streams drop on flaky networks and kill long-running agents.",
		"How Gormes fixes it.",
		"Single Static Binary",
		"No Runtime Drift",
		"Streams That Don&#39;t Drop",
		"Local Validation",
		"Route-B reconnect treats SSE drops",
		"gormes doctor --offline",
		// Audience filter — "Who Gormes is for" personas.
		`id="audience"`,
		"WHO GORMES IS FOR",
		"Operators of long-running agents",
		"Developers tired of Python/Nix/npm breakage",
		"Builders who want one binary that just runs",
		// Roadmap section — summary block (current focus + next milestone)
		// up top, full phase checklist behind a <details> disclosure.
		"BUILD STATE",
		"What works today, and what&#39;s still being wired up.",
		"Current focus",
		"Next milestone",
		"Gateway stability",
		"Memory system",
		"Brain transplant",
		"Fully independent Go-native brain",
		`<details class="roadmap-details">`,
		"View full phase-by-phase checklist",
		// Fuzzy phase-title presence (each phase renders)
		"Phase 1",
		"Phase 2",
		"Phase 3",
		"Phase 4",
		"Phase 5",
		"Phase 6",
		// Status tone classes driven by current phase-level data.
		"roadmap-status-progress",
		"roadmap-status-planned",
		// Complete work still appears at item level even when no whole phase is complete.
		"roadmap-item-shipped",
		// Structural class anchors
		"roadmap-phase",
		// Footer — brand text + license. Footer-nav now carries the
		// secondary links (Why Gormes, Who it's for, Docs, Company) so
		// the topnav can stay minimal (Install / Roadmap / GitHub).
		`Gormes v0.1.0 · <a href="https://trebuchetdynamics.com/">TrebuchetDynamics</a>`,
		"MIT License · 2026",
		`class="footer-nav"`,
		`<a href="https://trebuchetdynamics.com/">Company</a>`,
		`<a href="https://docs.gormes.ai/">Docs</a>`,
		// In-page note pointing at the Hugo docs site
		"Deeper reference material lives at",
		`<a href="https://docs.gormes.ai/">docs.gormes.ai →</a>`,
		// CSS link
		`href="/static/site.css"`,
		// Favicons — full set wired into <head>.
		`href="/static/favicon.ico"`,
		`href="/static/favicon-16x16.png"`,
		`href="/static/favicon-32x32.png"`,
		`href="/static/apple-touch-icon.png"`,
		// Open Graph + Twitter cards. Asserts the property/name keys
		// and the canonical URL/image so social previews stay healthy.
		`property="og:type" content="website"`,
		`property="og:site_name" content="Gormes"`,
		`property="og:url" content="https://gormes.ai/"`,
		`property="og:image" content="https://gormes.ai/static/social-card.png"`,
		`property="og:image:width" content="1200"`,
		`property="og:image:height" content="630"`,
		`name="twitter:card" content="summary_large_image"`,
		`name="twitter:image" content="https://gormes.ai/static/social-card.png"`,
	}
	rejects := []string{
		"Run Hermes Through a Go Operator Console.",
		"Hermes, In a Single Static Binary.",
		"Requires Hermes backend at localhost:8642.",
		"Install Hermes →",
		"No Python runtime on the host",
		"Boot Sequence",
		"Proof Rail",
		"01 / INSTALL HERMES",
		"Why Hermes users switch",
		"Inspect the Machine",
		"~8 MB",
		"~12 MB",
		// Old hero/features copy that conflated frontend with full replacement
		"One Go Binary. Same Hermes Brain.",
		"A static Go binary that talks to your Hermes backend over HTTP.",
		"Why a Go layer matters.",
		"Boots Like a Tool",
		"In-Process Tool Loop",
		"Survives Dropped Streams",
		// Operations-first rewrite that buried the "what is Gormes for"
		// answer behind lineage detail. Replaced with "Go-native runtime
		// for AI agents" framing.
		"Gormes is a Go-native rewrite of Hermes Agent — built to solve the operations problem, not the AI problem.",
		"Why Hermes breaks in production — and how Gormes fixes it.",
		"Rerun the installer to update the managed Gormes checkout.",
		"Source-backed for now →",
		"not production-ready yet",
		// v2 single-paragraph subhead replaced by 3-line stack.
		"Gormes is a Go-native runtime for AI agents — built to fix the reliability and deployment problems",
		// Hero illustration removed in v3 — assert the gopher PNG and
		// the .hero-image / .hero-content flex wrappers no longer ship.
		`alt="Gormes Gopher"`,
		"go-gopher-bear-lowpoly.png",
		`class="hero-image"`,
		`class="hero-content"`,
		`class="btn-secondary"`,
		// Obsolete single-row ledger copy replaced by grouped roadmap
		"Phase 3 — SQLite + FTS5 transcript memory.",
		"Phase 3.A–C — SQLite + FTS5 lattice, ontological graph, neural recall.",
		"Phase 4 — Native prompt building + agent orchestration.",
		"Phase 4 — Brain transplant. Hermes backend becomes optional.",
		"Phase 5 — 100% Go. Python tool scripts ported. Hermes-off.",
	}

	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered page missing %q\nbody:\n%s", want, text)
		}
	}
	for _, reject := range rejects {
		if strings.Contains(text, reject) {
			t.Fatalf("rendered page still contains stale token %q", reject)
		}
	}

	// There should be exactly 7 roadmap phase blocks — not a specific
	// phase-name assertion, just that the roadmap actually renders the
	// full phase set from progress.json.
	if n := strings.Count(text, `class="roadmap-phase"`); n != 7 {
		t.Errorf("roadmap phase count = %d, want 7", n)
	}

	// The progress tracker label follows a "N/M shipped" shape driven
	// by progress.json Stats(). We assert the shape, not the numbers.
	trackerRE := regexp.MustCompile(`\d+/\d+ shipped`)
	if !trackerRE.MatchString(text) {
		t.Errorf("missing N/M shipped progress tracker label; body:\n%s", text)
	}
}

func TestEmbeddedTemplates_ArePresentAndParse(t *testing.T) {
	files := []string{
		"templates/layout.tmpl",
		"templates/index.tmpl",
		"templates/partials/install_step.tmpl",
		"templates/partials/feature_card.tmpl",
		"templates/partials/audience_card.tmpl",
		"templates/partials/roadmap_phase.tmpl",
	}

	for _, name := range files {
		body, err := fs.ReadFile(templateFS, name)
		if err != nil {
			t.Fatalf("embedded template %q missing: %v", name, err)
		}
		if len(body) == 0 {
			t.Fatalf("embedded template %q is empty", name)
		}
	}

	templates, err := parseTemplates()
	if err != nil {
		t.Fatalf("parseTemplates: %v", err)
	}

	for _, want := range []string{"layout", "index", "install_step", "feature_card", "audience_card", "roadmap_phase"} {
		if templates.Lookup(want) == nil {
			t.Fatalf("parsed templates missing %q", want)
		}
	}
}
