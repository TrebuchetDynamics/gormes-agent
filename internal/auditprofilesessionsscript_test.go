package internal_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuditProfileSessionsGatewayStatusUsesConfiguredHome(t *testing.T) {
	repoRoot := testRepoRoot(t)
	tmpRepo := t.TempDir()

	copyFile(t,
		filepath.Join(repoRoot, "scripts", "audit-profile-sessions.sh"),
		filepath.Join(tmpRepo, "scripts", "audit-profile-sessions.sh"),
		0o755,
	)
	writeFile(t,
		filepath.Join(tmpRepo, "docs", "content", "building-gormes", "modules", "runtime.md"),
		[]byte("# Runtime\n"),
		0o644,
	)
	runCommand(t, tmpRepo, "git", "init")

	binDir := filepath.Join(tmpRepo, "bin")
	writeFile(t,
		filepath.Join(binDir, "gormes"),
		[]byte(`#!/usr/bin/env bash
printf '{"observed_gormes_home":"%s","args":"%s"}\n' "${GORMES_HOME:-}" "$*"
`),
		0o755,
	)

	auditHome := filepath.Join(tmpRepo, "audit-home")
	ambientHome := filepath.Join(tmpRepo, "ambient-home")
	outDir := filepath.Join(tmpRepo, "audit-out")
	writeFile(t,
		filepath.Join(auditHome, "config.toml"),
		[]byte("[hermes]\nprovider = 'openai-codex'\nmodel = 'gpt-5.5'\n"),
		0o600,
	)

	cmd := exec.Command("bash", "scripts/audit-profile-sessions.sh",
		"--dry-run",
		"--no-hermes",
		"--gormes-home", auditHome,
		"--out", outDir,
	)
	cmd.Dir = tmpRepo
	cmd.Env = overlayEnv(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"HOME="+filepath.Join(tmpRepo, "home"),
		"GORMES_HOME="+ambientHome,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("audit-profile-sessions failed: %v\noutput:\n%s", err, string(out))
	}

	bundle, err := os.ReadFile(filepath.Join(outDir, "bundle.md"))
	if err != nil {
		t.Fatalf("read bundle: %v", err)
	}
	want := `"observed_gormes_home":"` + auditHome + `"`
	if !strings.Contains(string(bundle), want) {
		t.Fatalf("bundle missing gateway status with configured GORMES_HOME %q\nbundle:\n%s", want, string(bundle))
	}
	if !strings.Contains(string(bundle), "profile_config_toml\tpresent") {
		t.Fatalf("bundle missing config.toml profile provenance\nbundle:\n%s", string(bundle))
	}
}

func TestAuditProfileSessionsHonorsSessionLineLimitForYAML(t *testing.T) {
	repoRoot := testRepoRoot(t)
	tmpRepo := t.TempDir()

	copyFile(t,
		filepath.Join(repoRoot, "scripts", "audit-profile-sessions.sh"),
		filepath.Join(tmpRepo, "scripts", "audit-profile-sessions.sh"),
		0o755,
	)
	writeFile(t,
		filepath.Join(tmpRepo, "docs", "content", "building-gormes", "modules", "sessions.md"),
		[]byte("# Sessions\n"),
		0o644,
	)
	runCommand(t, tmpRepo, "git", "init")

	binDir := filepath.Join(tmpRepo, "bin")
	writeFile(t,
		filepath.Join(binDir, "gormes"),
		[]byte("#!/usr/bin/env bash\nprintf '{}\\n'\n"),
		0o755,
	)

	auditHome := filepath.Join(tmpRepo, "audit-home")
	writeFile(t,
		filepath.Join(auditHome, "sessions", "recent.yaml"),
		[]byte("line1\nline2\nline3\n"),
		0o644,
	)
	outDir := filepath.Join(tmpRepo, "audit-out")

	cmd := exec.Command("bash", "scripts/audit-profile-sessions.sh",
		"--dry-run",
		"--no-hermes",
		"--no-journal",
		"--gormes-home", auditHome,
		"--out", outDir,
		"--max-session-lines", "2",
	)
	cmd.Dir = tmpRepo
	cmd.Env = overlayEnv(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"HOME="+filepath.Join(tmpRepo, "home"),
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("audit-profile-sessions failed: %v\noutput:\n%s", err, string(out))
	}

	bundle, err := os.ReadFile(filepath.Join(outDir, "bundle.md"))
	if err != nil {
		t.Fatalf("read bundle: %v", err)
	}
	if strings.Contains(string(bundle), "line3") {
		t.Fatalf("bundle included YAML content beyond --max-session-lines:\n%s", string(bundle))
	}
}
