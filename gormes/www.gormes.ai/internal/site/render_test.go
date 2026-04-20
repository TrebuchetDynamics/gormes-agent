package site

import (
	"io/fs"
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
		// Hero
		"OPEN SOURCE · MIT LICENSE",
		"Hermes, In a Single Static Binary.",
		"Zero-CGO. No Python runtime on the host. One file you scp anywhere",
		// Install
		"1. INSTALL",
		"curl -fsSL https://gormes.ai/install.sh | sh",
		"2. RUN",
		"Requires Hermes backend at localhost:8642.",
		"Install Hermes →",
		// Features
		"FEATURES",
		"Why a Go layer matters.",
		"Single Static Binary",
		"Boots Like a Tool",
		"In-Process Tool Loop",
		"Survives Dropped Streams",
		"Route-B reconnect treats SSE drops",
		// Shipping ledger
		"SHIPPING STATE",
		"What ships now, what doesn&#39;t.",
		"Phase 1 — Bubble Tea TUI shell.",
		"Phase 2 — Tool registry + Telegram adapter + session resume.",
		"Phase 3 — SQLite + FTS5 transcript memory.",
		"Phase 4 — Native prompt building + agent orchestration.",
		// Footer
		"Gormes v0.1.0 · TrebuchetDynamics",
		"MIT License · 2026",
		// CSS link
		`href="/static/site.css"`,
	}
	rejects := []string{
		"Run Hermes Through a Go Operator Console.",
		"Boot Sequence",
		"Proof Rail",
		"01 / INSTALL HERMES",
		"Why Hermes users switch",
		"Inspect the Machine",
		"<script",
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
}

func TestEmbeddedTemplates_ArePresentAndParse(t *testing.T) {
	files := []string{
		"templates/layout.tmpl",
		"templates/index.tmpl",
		"templates/partials/install_step.tmpl",
		"templates/partials/feature_card.tmpl",
		"templates/partials/ship_state.tmpl",
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

	for _, want := range []string{"layout", "index", "install_step", "feature_card", "ship_state"} {
		if templates.Lookup(want) == nil {
			t.Fatalf("parsed templates missing %q", want)
		}
	}
}
