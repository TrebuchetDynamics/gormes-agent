package secrets

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/externalsecrets"
)

func TestBitwardenInstallUsesManagedInstallerAndRedactsOutput(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	asset, err := externalsecrets.BitwardenAssetName(externalsecrets.BitwardenInstallOptions{})
	if err != nil {
		t.Fatalf("asset name: %v", err)
	}
	binaryName := "bws"
	if strings.Contains(asset, "windows") {
		binaryName = "bws.exe"
	}
	zipBytes := appBitwardenTestZip(t, binaryName, "#!/bin/sh\necho bws 2.0.0\n")
	checksum := fmt.Sprintf("%x  %s\n", sha256.Sum256(zipBytes), asset)
	var out bytes.Buffer
	if err := BitwardenInstall(context.Background(), &out, false, Options{
		BitwardenInstallDownload: func(_ context.Context, url string) ([]byte, error) {
			if strings.HasSuffix(url, asset) {
				return zipBytes, nil
			}
			return []byte(checksum), nil
		},
	}); err != nil {
		t.Fatalf("BitwardenInstall: %v\nout=%s", err, out.String())
	}
	if !strings.Contains(out.String(), "Installed bws 2.0.0") || strings.Contains(out.String(), "BWS_ACCESS_TOKEN") {
		t.Fatalf("unexpected install output:\n%s", out.String())
	}
	if _, err := os.Stat(filepath.Join(home, "bin", "bws")); err != nil {
		t.Fatalf("installed bws missing: %v", err)
	}
}

func TestBitwardenSetupNonInteractiveWritesBootstrapAndConfigSecretSafe(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	writeFakeSetupBWS(t, home)
	t.Setenv("GORMES_API_KEY", "sk-existing")

	var out bytes.Buffer
	err := BitwardenSetup(context.Background(), &out, "0.test-token", "https://vault.bitwarden.example", "project-123", Options{IsTerminal: func() bool { return false }})
	if err != nil {
		t.Fatalf("BitwardenSetup: %v\nout=%s", err, out.String())
	}
	if strings.Contains(out.String(), "0.test-token") || strings.Contains(out.String(), "sk-bitwarden-secret") {
		t.Fatalf("setup leaked secret:\n%s", out.String())
	}
	envBody, err := os.ReadFile(filepath.Join(home, ".env"))
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	if !strings.Contains(string(envBody), "BWS_ACCESS_TOKEN='0.test-token'") && !strings.Contains(string(envBody), "BWS_ACCESS_TOKEN=0.test-token") {
		t.Fatalf(".env missing bootstrap token under env name:\n%s", envBody)
	}
	info, err := os.Stat(filepath.Join(home, ".env"))
	if err != nil {
		t.Fatalf("stat .env: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf(".env mode = %v, want 0600", info.Mode().Perm())
	}
	cfgBody, err := os.ReadFile(filepath.Join(home, "config.toml"))
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	for _, want := range []string{"enabled = true", `project_id = "project-123"`, `server_url = "https://vault.bitwarden.example"`, `access_token_env = "BWS_ACCESS_TOKEN"`, "cache_ttl_seconds = 300", "override_existing = true", "auto_install = true"} {
		if !strings.Contains(string(cfgBody), want) {
			t.Fatalf("config missing %q:\n%s", want, cfgBody)
		}
	}
	if strings.Contains(string(cfgBody), "sk-bitwarden-secret") {
		t.Fatalf("config persisted fetched secret:\n%s", cfgBody)
	}
	if !strings.Contains(out.String(), "Status: gormes secrets bitwarden status") || !strings.Contains(out.String(), "GORMES_API_KEY: new") || !strings.Contains(out.String(), "BWS_ACCESS_TOKEN: bootstrap token") {
		t.Fatalf("setup output missing hints/preview:\n%s", out.String())
	}
}

func TestBitwardenSetupNonInteractiveMissingFlagsIsSecretSafe(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	var out bytes.Buffer
	err := BitwardenSetup(context.Background(), &out, "", "", "", Options{IsTerminal: func() bool { return false }})
	if err == nil {
		t.Fatal("BitwardenSetup missing flags err = nil")
	}
	for _, want := range []string{"--access-token", "--server-url", "--project-id"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing flags output lacks %s:\n%s", want, out.String())
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".env")); !os.IsNotExist(err) {
		t.Fatalf(".env created on missing flags: %v", err)
	}
}

func TestBitwardenSetupInteractiveChoosesProject(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	writeFakeSetupBWS(t, home)
	var out bytes.Buffer
	err := BitwardenSetup(context.Background(), &out, "0.interactive", "", "", Options{IsTerminal: func() bool { return true }, BitwardenSetupInput: strings.NewReader("eu\n2\n")})
	if err != nil {
		t.Fatalf("BitwardenSetup interactive: %v\nout=%s", err, out.String())
	}
	cfgBody, err := os.ReadFile(filepath.Join(home, "config.toml"))
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	for _, want := range []string{`server_url = "https://vault.bitwarden.eu"`, `project_id = "project-2"`} {
		if !strings.Contains(string(cfgBody), want) {
			t.Fatalf("interactive config missing %q:\n%s", want, cfgBody)
		}
	}
}

func writeFakeSetupBWS(t *testing.T, home string) {
	t.Helper()
	path := filepath.Join(home, "bin", "bws")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir fake bws dir: %v", err)
	}
	body := `#!/bin/sh
if [ "$1" = "--version" ]; then echo 'bws 2.0.0'; exit 0; fi
if [ "$1" = "project" ]; then printf '%s' '[{"id":"project-1","name":"One"},{"id":"project-2","name":"Two"}]'; exit 0; fi
printf '%s' '[{"key":"GORMES_API_KEY","value":"sk-bitwarden-secret"},{"key":"BWS_ACCESS_TOKEN","value":"0.malicious"}]'
`
	if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
		t.Fatalf("write fake bws: %v", err)
	}
}

func TestBitwardenSyncDryRunAndApplyAreSecretSafe(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(`
[secrets.bitwarden]
enabled = true
project_id = "project-123"
access_token_env = "BWS_ACCESS_TOKEN"
auto_install = false
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	fakeBWS := filepath.Join(home, "bws")
	if err := os.WriteFile(fakeBWS, []byte("#!/bin/sh\nprintf '%s' '[{\"key\":\"GORMES_API_KEY\",\"value\":\"sk-bitwarden-secret\"},{\"key\":\"BWS_ACCESS_TOKEN\",\"value\":\"0.malicious\"}]'\n"), 0o700); err != nil {
		t.Fatalf("write fake bws: %v", err)
	}
	t.Setenv("PATH", filepath.Dir(fakeBWS)+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("BWS_ACCESS_TOKEN", "0.bootstrap")
	t.Setenv("GORMES_API_KEY", "sk-existing-secret")

	var out bytes.Buffer
	if err := BitwardenSync(context.Background(), &out, false); err != nil {
		t.Fatalf("dry-run sync: %v", err)
	}
	if got := os.Getenv("GORMES_API_KEY"); got != "sk-existing-secret" {
		t.Fatalf("dry-run mutated GORMES_API_KEY = %q", got)
	}
	if !strings.Contains(out.String(), "skip (already set)") || !strings.Contains(out.String(), "skip (bootstrap token)") {
		t.Fatalf("dry-run output missing skip actions:\n%s", out.String())
	}
	if strings.Contains(out.String(), "sk-bitwarden-secret") || strings.Contains(out.String(), "0.bootstrap") {
		t.Fatalf("dry-run leaked secret:\n%s", out.String())
	}

	out.Reset()
	if err := BitwardenSync(context.Background(), &out, true); err != nil {
		t.Fatalf("apply sync: %v", err)
	}
	if got := os.Getenv("GORMES_API_KEY"); got != "sk-bitwarden-secret" {
		t.Fatalf("apply did not update env, got %q", got)
	}
	if strings.Contains(out.String(), "sk-bitwarden-secret") || strings.Contains(out.String(), "0.bootstrap") {
		t.Fatalf("apply leaked secret:\n%s", out.String())
	}
}

func appBitwardenTestZip(t *testing.T, name, body string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	if _, err := w.Write([]byte(body)); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

func TestBitwardenDisablePersistsConfigOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte("[secrets.bitwarden]\nenabled = true\nproject_id = \"project-123\"\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	var out bytes.Buffer
	if err := BitwardenDisable(context.Background(), &out); err != nil {
		t.Fatalf("disable: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(home, "config.toml"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(body), "enabled = false") || !strings.Contains(out.String(), "bootstrap token is left in .env") {
		t.Fatalf("disable evidence mismatch:\nconfig=%s\nout=%s", body, out.String())
	}
}
