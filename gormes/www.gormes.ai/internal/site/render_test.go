package site

import (
	"bytes"
	"io/fs"
	"strings"
	"testing"
)

func TestRenderIndex_RendersOperationalMoatStory(t *testing.T) {
	body, err := RenderIndex()
	if err != nil {
		t.Fatalf("RenderIndex: %v", err)
	}

	text := string(body)
	wants := []string{
		"Gormes.ai | The Agent That GOes With You.",
		"The Agent That GOes With You.",
		"Open Source • MIT License • 7.9 MB Static Binary • Zero-CGO",
		"API_SERVER_ENABLED=true hermes gateway start",
		"./bin/gormes doctor --offline",
		"./bin/gormes-telegram",
		"Phase 2 is live on trunk: 2.A Tool Registry, 2.B.1 Telegram Scout, and 2.C thin bbolt resume are shipped.",
		"The Port Is Already Moving",
		"Help Finish the Port",
		"Go-native tool registry",
		"Telegram Scout",
		"https://github.com/XelHaku/gormes-agent",
	}

	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered page missing %q\nbody:\n%s", want, text)
		}
	}

	if !bytes.Contains(body, []byte(`href="/static/site.css"`)) {
		t.Fatalf("rendered page missing stylesheet link\n%s", text)
	}
}

func TestEmbeddedTemplates_ArePresentAndParse(t *testing.T) {
	files := []string{
		"templates/layout.tmpl",
		"templates/index.tmpl",
		"templates/partials/code_block.tmpl",
		"templates/partials/feature_card.tmpl",
		"templates/partials/phase_item.tmpl",
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

	for _, want := range []string{"layout", "index", "code_block", "feature_card", "phase_item"} {
		if templates.Lookup(want) == nil {
			t.Fatalf("parsed templates missing %q", want)
		}
	}
}
