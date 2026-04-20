package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportDir_WritesStaticSite(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dist")

	if err := ExportDir(root); err != nil {
		t.Fatalf("ExportDir: %v", err)
	}

	indexPath := filepath.Join(root, "index.html")
	body, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read %s: %v", indexPath, err)
	}
	text := string(body)
	wants := []string{
		"Gormes.ai | Run Hermes Through a Go Operator Console",
		"Run Hermes Through a Go Operator Console.",
		"curl -fsSL https://raw.githubusercontent.com/NousResearch/hermes-agent/main/scripts/install.sh | bash",
		"curl -fsSL https://gormes.ai/install.sh | sh",
		"Today the Gormes installer shells out to Go itself. Keep Go 1.25&#43; on PATH until release artifacts exist.",
		"If the installer prints an export PATH line, run it in this shell before launching gormes.",
		"source ~/.bashrc    # reload shell (or: source ~/.zshrc)",
		`class="hero hero-deck"`,
		`class="activation-grid"`,
		`class="shipping-ledger"`,
	}
	rejects := []string{
		"The Agent That GOes With You.",
		"7.9 MB",
		"8.2M shell",
		"15M telegram edge",
		"split Telegram edge",
	}

	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("index.html missing %q\n%s", want, text)
		}
	}

	for _, reject := range rejects {
		if strings.Contains(text, reject) {
			t.Fatalf("index.html still contains stale claim %q\n%s", reject, text)
		}
	}

	cssPath := filepath.Join(root, "static", "site.css")
	css, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatalf("read %s: %v", cssPath, err)
	}
	if !strings.Contains(string(css), "--bg-0") {
		t.Fatalf("site.css missing expected variable")
	}

	installPath := filepath.Join(root, "install.sh")
	install, err := os.ReadFile(installPath)
	if err != nil {
		t.Fatalf("read %s: %v", installPath, err)
	}
	installText := string(install)
	for _, want := range []string{
		"github.com/TrebuchetDynamics/gormes-agent/gormes/cmd/gormes",
		`go install "${MODULE}@${VERSION}"`,
	} {
		if !strings.Contains(installText, want) {
			t.Fatalf("install.sh missing %q\n%s", want, installText)
		}
	}
	for _, reject := range []string{
		"XelHaku/golang-hermes-agent",
		"XelHaku/gormes-agent",
		"releases/latest",
	} {
		if strings.Contains(installText, reject) {
			t.Fatalf("install.sh still contains stale installer path %q\n%s", reject, installText)
		}
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
