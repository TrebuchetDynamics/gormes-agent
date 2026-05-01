package site

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestExportDir_WritesStaticSite(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dist")

	if err := ExportDir(root); err != nil {
		t.Fatalf("ExportDir: %v", err)
	}

	indexBody, err := os.ReadFile(filepath.Join(root, "index.html"))
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}
	text := string(indexBody)
	wants := []string{
		"Run AI Agents as One Go Binary.",
		"Gormes — Run AI Agents as One Go Binary",
		`<span class="hero-prompt" aria-hidden="true">$</span>`,
		"Gormes is a Go-native runtime for long-running AI agents.",
		"One static binary. No Python runtime. No Hermes process.",
		"Ship the same binary you test. Run it anywhere.",
		"Early-stage and shipping.",
		"Offline TUI, local doctor, provider-backed one-shots, and Goncho memory are ready today.",
		`class="hero-note-stamp"`,
		`class="hero-note-body"`,
		// Top nav keeps docs, build state, and source visible.
		`<a href="https://docs.gormes.ai/">Docs</a>`,
		`<a href="#roadmap">Roadmap</a>`,
		`<a href="https://github.com/TrebuchetDynamics/gormes-agent">GitHub</a>`,
		"Install in one command. Verify everything works.",
		"One command to install. Review the script first, then run it.",
		"1. INSTALL",
		"curl -fsSL https://gormes.ai/install.sh | sh",
		"2. OFFLINE TUI",
		"gormes --offline",
		"3. LOCAL DOCTOR",
		"gormes doctor --offline",
		"4. MEMORY AUDIT",
		"gormes goncho doctor --json",
		"5. MODEL-BACKED TURN",
		"GORMES_ENDPOINT=&#34;https://your-provider.example/v1&#34;",
		"gormes --oneshot &#34;hello from Gormes&#34;",
		"6. BUILD FROM SOURCE",
		"git clone https://github.com/TrebuchetDynamics/gormes-agent.git",
		"cd gormes-agent",
		"make build",
		"The installer clones, builds, and links gormes into your PATH",
		"Read the installer source →",
		// Features: deployment pain frame before technical fix cards.
		"Why agent runtimes fail in production",
		"How Gormes fixes the runtime surface",
		`class="why-fix-subhead"`,
		"Python-stack agents fail operationally when:",
		"dev, staging, and prod stop matching",
		"install scripts depend on host package luck",
		"long turns die on dropped streams",
		"tool wiring fails after tokens are already burning",
		"Transparent Local Memory",
		"Release Trust Roadmap",
		"AV false-positive submission",
		// Roadmap summary + collapse
		"Useful today, still early.",
		"Current focus",
		"Offline TUI, local doctor, and provider-backed one-shots",
		"Gateway stability and shared channel contracts",
		"Next milestone",
		"Production-stable Go-native runtime, no Hermes process",
		"View full phase-by-phase checklist",
		`<details class="roadmap-details">`,
		`<nav class="footer-nav" aria-label="Secondary">`,
		`<span class="footer-nav-sep" aria-hidden="true">·</span>`,
		`<a href="https://docs.gormes.ai/">Docs</a>`,
		`<a href="https://trebuchetdynamics.com/">Company</a>`,
		// Favicons + social-card meta tags rendered in <head>.
		`href="/static/favicon.ico"`,
		`href="/static/apple-touch-icon.png"`,
		`property="og:image" content="https://gormes.ai/static/social-card.png"`,
		`name="twitter:card" content="summary_large_image"`,
		// Structural roadmap checks — no exact counts or item names,
		// those come from progress.json and must not be locked in here.
		"roadmap-status-progress",
		"roadmap-item-pending",
		"roadmap-item-shipped",
		"roadmap-phase",
		// Fuzzy phase-title presence — each phase renders, no subtitles
		"Phase 1",
		"Phase 2",
		"Phase 3",
		"Phase 4",
		"Phase 5",
		"Phase 6",
	}
	rejects := []string{
		`<div class="hero-image">`,
		`go-gopher-bear-lowpoly.png`,
		"Run Hermes Through a Go Operator Console.",
		"Hermes, In a Single Static Binary.",
		"Requires Hermes backend at localhost:8642.",
		"Install Hermes →",
		"No Python runtime on the host",
		"~8 MB",
		"~12 MB",
		"Phase 3 — SQLite + FTS5 transcript memory.",
		// Old single-row ledger copy must not survive the grouped rewrite
		"Phase 4 — Native prompt building + agent orchestration.",
		"Phase 4 — Brain transplant. Hermes backend becomes optional.",
		"Phase 5 — 100% Go. Python tool scripts ported. Hermes-off.",
		"Phase 3.A–C + 3.D.5 — SQLite + FTS5 lattice, ontological graph, neural recall, USER.md mirror",
		"Boot Sequence",
		"Proof Rail",
		// Old hero/features copy that conflated frontend with full replacement
		"One Go Binary. Same Hermes Brain.",
		"A static Go binary that talks to your Hermes backend over HTTP.",
		"Why a Go layer matters.",
		"Boots Like a Tool",
		// Older revisions that buried the first-screen hierarchy.
		"Gormes is a Go-native rewrite of Hermes Agent — built to solve the operations problem, not the AI problem.",
		"Gormes is a Go-native runtime for AI agents — built to fix",
		"Why Hermes-stack agents break in production.",
		"Why Hermes breaks in production",
		"Hermes breaks in production because:",
		"irm https://gormes.ai/install.ps1 | iex",
		"One Go Binary. No Python. No Drift.",
		"Reliability-first runtime for developers who ship agents, not demos.",
		"UNDER CONSTRUCTION",
		"Rerun the installer to update the managed Gormes checkout.",
		"Source-backed for now →",
	}
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("dist/index.html missing %q", want)
		}
	}
	for _, reject := range rejects {
		if strings.Contains(text, reject) {
			t.Fatalf("dist/index.html still contains stale token %q", reject)
		}
	}

	// Roadmap renders all 7 phases — structural, not copy-specific.
	if n := strings.Count(text, `class="roadmap-phase"`); n != 7 {
		t.Errorf("dist/index.html roadmap phase count = %d, want 7", n)
	}

	// The progress tracker follows "N/M shipped" — shape, not numbers.
	trackerRE := regexp.MustCompile(`\d+/\d+ shipped`)
	if !trackerRE.MatchString(text) {
		t.Errorf("dist/index.html missing N/M shipped tracker label")
	}

	cssPath := filepath.Join(root, "static", "site.css")
	css, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatalf("read %s: %v", cssPath, err)
	}
	if !strings.Contains(string(css), "--bg-0") {
		t.Fatalf("site.css missing --bg-0 design token")
	}
	cssText := string(css)
	for _, want := range []string{
		".hero-title {\n  font-family: var(--font-display);",
		".section-title {\n  font-family: var(--font-body);",
		".feature-card h3 {\n  font-family: var(--font-body);",
		".install-step {\n  background: var(--bg-1);",
		".roadmap-title {\n  font-family: var(--font-body);",
		".cmd {\n  background: var(--bg-1);",
	} {
		if !strings.Contains(cssText, want) {
			t.Fatalf("site.css missing typography/layout contract %q", want)
		}
	}
	// v6 dropped the .feature-card h3::after divider per the
	// "one accent per card" constraint — title:body distinction
	// now comes from typography weight + color, not a decorative
	// rule. Assert it's gone so the simplification can't regress.
	if strings.Contains(cssText, ".feature-card h3::after {") {
		t.Fatalf("feature-card h3::after divider should be removed (one-accent constraint)")
	}
	if strings.Contains(cssText, ".section-title {\n  font-family: var(--font-display);") {
		t.Fatalf("section titles should not use display serif")
	}
	if strings.Contains(cssText, ".feature-card h3 {\n  font-family: var(--font-display);") {
		t.Fatalf("feature titles should not use display serif")
	}
	if strings.Contains(cssText, ".roadmap-title {\n  font-family: var(--font-display);") {
		t.Fatalf("roadmap titles should not use display serif")
	}

	// Favicon set + OG social card must land in dist/static/. Guarding
	// against regressions in the embed list — if a new icon is added it
	// has to flow through ExportDir too.
	for _, asset := range []string{
		"favicon.ico",
		"favicon-16x16.png",
		"favicon-32x32.png",
		"apple-touch-icon.png",
		"android-chrome-192x192.png",
		"android-chrome-512x512.png",
		"social-card.png",
	} {
		path := filepath.Join(root, "static", asset)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("static export missing %s: %v", asset, err)
		}
		if info.Size() == 0 {
			t.Fatalf("static export %s is empty", asset)
		}
	}

	installBody, err := os.ReadFile(filepath.Join(root, "install.sh"))
	if err != nil {
		t.Fatalf("read install.sh: %v", err)
	}
	if !strings.Contains(string(installBody), "https://github.com/TrebuchetDynamics/gormes-agent.git") {
		t.Fatalf("install.sh missing TrebuchetDynamics repo URL")
	}

	for _, asset := range []string{"install.ps1", "install.cmd"} {
		body, err := os.ReadFile(filepath.Join(root, asset))
		if err != nil {
			t.Fatalf("read %s: %v", asset, err)
		}
		if len(body) == 0 {
			t.Fatalf("%s is empty in static export", asset)
		}
	}

	ps1Body, err := os.ReadFile(filepath.Join(root, "install.ps1"))
	if err != nil {
		t.Fatalf("read install.ps1: %v", err)
	}
	for _, want := range []string{"LOCALAPPDATA", "gormes-agent", "Invoke-Main"} {
		if !strings.Contains(string(ps1Body), want) {
			t.Fatalf("install.ps1 missing %q in static export", want)
		}
	}

	cmdBody, err := os.ReadFile(filepath.Join(root, "install.cmd"))
	if err != nil {
		t.Fatalf("read install.cmd: %v", err)
	}
	if !strings.Contains(string(cmdBody), "install.ps1") {
		t.Fatalf("install.cmd missing PowerShell handoff in static export")
	}
}

func TestExportDir_RecreatesDist(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dist")
	stalePath := filepath.Join(root, "stale.txt")

	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(stalePath, []byte("stale"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := ExportDir(root); err != nil {
		t.Fatalf("ExportDir: %v", err)
	}

	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Fatalf("stale file still present after export: err=%v", err)
	}
}
