package tui

import (
	"strings"
	"testing"
)

func TestBannerLogo_NonEmpty(t *testing.T) {
	skin := DefaultHermesSkin()
	logo := bannerLogo(skin)
	if strings.TrimSpace(logo) == "" {
		t.Fatal("banner logo is empty")
	}
	if strings.Contains(logo, "HERMES") {
		t.Fatal("banner logo leaked upstream Hermes branding")
	}
	if !strings.Contains(logo, "████") {
		t.Fatal("banner logo missing expected content")
	}
}

func TestBannerCaduceus_NonEmpty(t *testing.T) {
	cad := bannerCaduceus()
	if strings.TrimSpace(cad) == "" {
		t.Fatal("banner caduceus is empty")
	}
}

func TestBannerWelcome(t *testing.T) {
	w := bannerWelcome()
	if !strings.Contains(w, "Gormes") {
		t.Fatalf("welcome = %q, want Gormes branding", w)
	}
}
