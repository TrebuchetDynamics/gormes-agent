package gormescli

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config/externalsecrets"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestSecretsConfigureCommandOutputsTypedRefWithoutSecret(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("OPENROUTER_API_KEY", "sk-openrouter-secret")

	cmd := newSecretsRootCommandForTest()
	stdout, stderr, err := executeRootCommandForTest(cmd,
		"secrets", "configure", "providers.openrouter.api_key",
		"--source", "env",
		"--id", "OPENROUTER_API_KEY",
		"--json",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout+stderr, "sk-openrouter-secret") {
		t.Fatalf("secrets configure leaked secret:\nstdout=%s\nstderr=%s", stdout, stderr)
	}

	var got toolspkg.SecretsConfigureResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("json.Unmarshal: %v\nstdout=%s", err, stdout)
	}
	if got.Code != toolspkg.SecretsEvidenceConfigured || !got.PreflightOK {
		t.Fatalf("configure result = %+v, want configured preflight OK", got)
	}
	if got.Target.Path != "providers.openrouter.api_key" ||
		got.Target.Ref.Source != toolspkg.SecretRefSourceEnv ||
		got.Target.Ref.Provider != toolspkg.DefaultSecretProviderAlias ||
		got.Target.Ref.ID != "OPENROUTER_API_KEY" {
		t.Fatalf("configured target = %+v, want typed env SecretRef", got.Target)
	}
}

// TestSecretsApplyJSONIncludesBuildProvenance proves
// `gormes secrets apply --json` carries the running binary's build
// version + SHA. Same contract as update/doctor/status/restore/auth-status —
// captured secrets-runtime mutations stay attributable to a specific
// binary.
func TestSecretsApplyJSONIncludesBuildProvenance(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("GORMES_API_KEY", "sk-build-prov-secret")

	plan := filepath.Join(t.TempDir(), "secrets.json")
	if err := os.WriteFile(plan, []byte(`{
		"targets": [
			{
				"path": "hermes.api_key",
				"required": true,
				"ref": {"source": "env", "provider": "default", "id": "GORMES_API_KEY"}
			}
		]
	}`), 0o600); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	cmd := newSecretsRootCommandForTest()
	stdout, stderr, err := executeRootCommandForTest(cmd, "secrets", "apply", "--plan", plan, "--json")
	if err != nil {
		t.Fatalf("secrets apply --json: %v\nstdout=%s stderr=%s", err, stdout, stderr)
	}
	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Build.Version != Version {
		t.Fatalf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Build.GitCommit == "" {
		t.Fatalf("got.build.git_commit must be non-empty")
	}
}

func TestSecretsApplyAuditAndReloadCommandsUseRedactedSnapshot(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("GORMES_API_KEY", "sk-apply-secret")

	goodPlan := filepath.Join(t.TempDir(), "good-secrets.json")
	if err := os.WriteFile(goodPlan, []byte(`{
		"targets": [
			{
				"path": "hermes.api_key",
				"required": true,
				"ref": {"source": "env", "provider": "default", "id": "GORMES_API_KEY"}
			}
		]
	}`), 0o600); err != nil {
		t.Fatalf("write good plan: %v", err)
	}

	cmd := newSecretsRootCommandForTest()
	stdout, stderr, err := executeRootCommandForTest(cmd, "secrets", "apply", "--plan", goodPlan, "--json")
	if err != nil {
		t.Fatalf("apply: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout+stderr, "sk-apply-secret") {
		t.Fatalf("secrets apply leaked secret:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	var applied toolspkg.SecretsApplyResult
	if err := json.Unmarshal([]byte(stdout), &applied); err != nil {
		t.Fatalf("decode apply: %v\nstdout=%s", err, stdout)
	}
	if applied.Code != toolspkg.SecretsEvidenceApplied || applied.Snapshot.Generation != 1 {
		t.Fatalf("apply result = %+v, want generation 1 applied", applied)
	}
	assertSecretsSnapshotFile(t, "GORMES_API_KEY", "sk-apply-secret")

	auditPlan := filepath.Join(t.TempDir(), "audit-secrets.json")
	if err := os.WriteFile(auditPlan, []byte(`{
		"targets": [
			{
				"path": "hermes.api_key",
				"required": true,
				"plaintext": "sk-apply-secret",
				"ref": {"source": "env", "provider": "default", "id": "MISSING_API_KEY"}
			}
		]
	}`), 0o600); err != nil {
		t.Fatalf("write audit plan: %v", err)
	}
	cmd = newSecretsRootCommandForTest()
	stdout, stderr, err = executeRootCommandForTest(cmd, "secrets", "audit", "--plan", auditPlan, "--json")
	if err == nil {
		t.Fatalf("audit err = nil, want findings\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if strings.Contains(stdout+stderr+err.Error(), "sk-apply-secret") {
		t.Fatalf("secrets audit leaked secret:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
	}
	var audited toolspkg.SecretsAuditResult
	if err := json.Unmarshal([]byte(stdout), &audited); err != nil {
		t.Fatalf("decode audit: %v\nstdout=%s", err, stdout)
	}
	for _, want := range []string{toolspkg.SecretsFindingPlaintext, toolspkg.SecretsFindingUnresolvedRef, toolspkg.SecretsFindingPrecedenceDrift} {
		if !secretsCommandFindingPresent(audited.Findings, want) {
			t.Fatalf("audit findings = %+v, missing %s", audited.Findings, want)
		}
	}

	cmd = newSecretsRootCommandForTest()
	stdout, stderr, err = executeRootCommandForTest(cmd, "secrets", "reload", "--plan", auditPlan, "--json")
	if err == nil {
		t.Fatalf("reload err = nil, want secrets_unavailable\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if strings.Contains(stdout+stderr+err.Error(), "sk-apply-secret") {
		t.Fatalf("secrets reload leaked secret:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
	}
	var failedReload toolspkg.SecretsApplyResult
	if err := json.Unmarshal([]byte(stdout), &failedReload); err != nil {
		t.Fatalf("decode reload: %v\nstdout=%s", err, stdout)
	}
	if failedReload.Code != toolspkg.SecretsEvidenceUnavailable || failedReload.Snapshot.Generation != 1 {
		t.Fatalf("reload result = %+v, want unavailable with last good generation 1", failedReload)
	}
	assertSecretsSnapshotFile(t, "GORMES_API_KEY", "sk-apply-secret")
}

func TestSecretsBitwardenStatusSyncAndDisableCommands(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	home := config.GormesHome()
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(`
[secrets.bitwarden]
enabled = true
project_id = "project-123"
access_token_env = "BWS_ACCESS_TOKEN"
override_existing = true
auto_install = false
server_url = "https://vault.bitwarden.example"
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	fakeBWS := filepath.Join(home, "bws")
	if err := os.WriteFile(fakeBWS, []byte("#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo 'bws 2.0.0'; exit 0; fi\nprintf '%s' '[{\"key\":\"GORMES_API_KEY\",\"value\":\"sk-bitwarden-secret\"},{\"key\":\"BWS_ACCESS_TOKEN\",\"value\":\"0.malicious\"}]'\n"), 0o700); err != nil {
		t.Fatalf("write fake bws: %v", err)
	}
	managedBWS := filepath.Join(home, "bin", "bws")
	if err := os.MkdirAll(filepath.Dir(managedBWS), 0o700); err != nil {
		t.Fatalf("mkdir managed bws dir: %v", err)
	}
	if err := os.WriteFile(managedBWS, []byte("#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo 'bws 2.0.0'; exit 0; fi\nprintf '%s' '[{\"key\":\"GORMES_API_KEY\",\"value\":\"sk-bitwarden-secret\"},{\"key\":\"BWS_ACCESS_TOKEN\",\"value\":\"0.malicious\"}]'\n"), 0o700); err != nil {
		t.Fatalf("write managed bws: %v", err)
	}
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", filepath.Dir(fakeBWS)+string(os.PathListSeparator)+oldPath)
	t.Setenv("BWS_ACCESS_TOKEN", "0.bootstrap")
	t.Setenv("GORMES_API_KEY", "sk-existing-secret")

	cmd := newSecretsRootCommandForTest()
	stdout, stderr, err := executeRootCommandForTest(cmd, "secrets", "--help")
	if err != nil {
		t.Fatalf("secrets help: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "bitwarden") || !strings.Contains(stdout, "apply") || !strings.Contains(stdout, "audit") || !strings.Contains(stdout, "configure") || !strings.Contains(stdout, "reload") {
		t.Fatalf("secrets help missing existing commands or bitwarden:\n%s", stdout)
	}

	cmd = newSecretsRootCommandForTest()
	stdout, stderr, err = executeRootCommandForTest(cmd, "secrets", "bitwarden", "status")
	if err != nil {
		t.Fatalf("status: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "Enabled: yes") || !strings.Contains(stdout, "Token in env: yes") || !strings.Contains(stdout, "Project ID: project-123") || !strings.Contains(stdout, "bws binary:") {
		t.Fatalf("status output missing expected fields:\n%s", stdout)
	}
	if strings.Contains(stdout+stderr, "0.bootstrap") || strings.Contains(stdout+stderr, "sk-existing-secret") {
		t.Fatalf("status leaked secret:\nstdout=%s\nstderr=%s", stdout, stderr)
	}

	cmd = newSecretsRootCommandForTest()
	stdout, stderr, err = executeRootCommandForTest(cmd, "secrets", "bitwarden", "sync")
	if err != nil {
		t.Fatalf("sync dry-run: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "GORMES_API_KEY") || !strings.Contains(stdout, "skip (already set)") || !strings.Contains(stdout, "BWS_ACCESS_TOKEN") || !strings.Contains(stdout, "skip (bootstrap token)") {
		t.Fatalf("sync dry-run output missing expected actions:\n%s", stdout)
	}
	if strings.Contains(stdout+stderr, "sk-bitwarden-secret") || strings.Contains(stdout+stderr, "0.bootstrap") {
		t.Fatalf("sync dry-run leaked secret:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if got := os.Getenv("GORMES_API_KEY"); got != "sk-existing-secret" {
		t.Fatalf("dry-run mutated env: %q", got)
	}

	cmd = newSecretsRootCommandForTest()
	stdout, stderr, err = executeRootCommandForTest(cmd, "secrets", "bitwarden", "sync", "--apply")
	if err != nil {
		t.Fatalf("sync --apply: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if os.Getenv("GORMES_API_KEY") != "sk-bitwarden-secret" {
		t.Fatalf("sync --apply did not update process env")
	}
	if strings.Contains(stdout+stderr, "sk-bitwarden-secret") || strings.Contains(stdout+stderr, "0.bootstrap") {
		t.Fatalf("sync --apply leaked secret:\nstdout=%s\nstderr=%s", stdout, stderr)
	}

	cmd = newSecretsRootCommandForTest()
	stdout, stderr, err = executeRootCommandForTest(cmd, "secrets", "bitwarden", "install")
	if err != nil {
		t.Fatalf("install existing: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "Installed bws 2.0.0") || strings.Contains(stdout+stderr, "0.bootstrap") || strings.Contains(stdout+stderr, "sk-existing-secret") {
		t.Fatalf("install output mismatch/leak:\nstdout=%s\nstderr=%s", stdout, stderr)
	}

	cmd = newSecretsRootCommandForTest()
	stdout, stderr, err = executeRootCommandForTest(cmd, "secrets", "bitwarden", "setup", "--access-token", "0.setup-token", "--server-url", "https://vault.bitwarden.example", "--project-id", "project-123")
	if err != nil {
		t.Fatalf("setup noninteractive: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout+stderr, "0.setup-token") || strings.Contains(stdout+stderr, "sk-bitwarden-secret") || !strings.Contains(stdout, "Bitwarden Secrets Manager is enabled") {
		t.Fatalf("setup output mismatch/leak:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	envBody, err := os.ReadFile(filepath.Join(home, ".env"))
	if err != nil || !strings.Contains(string(envBody), "BWS_ACCESS_TOKEN") {
		t.Fatalf("setup did not write dotenv: %v\n%s", err, envBody)
	}

	cmd = newSecretsRootCommandForTest()
	stdout, stderr, err = executeRootCommandForTest(cmd, "secrets", "bitwarden", "disable")
	if err != nil {
		t.Fatalf("disable: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	body, err := os.ReadFile(filepath.Join(home, "config.toml"))
	if err != nil {
		t.Fatalf("read config after disable: %v", err)
	}
	if !strings.Contains(string(body), "enabled = false") || !strings.Contains(stdout, "bootstrap token is left in .env") {
		t.Fatalf("disable did not persist expected state/output:\nconfig=%s\nstdout=%s", body, stdout)
	}
}

func newSecretsRootCommandForTest() *cobra.Command {
	return newRootCommandWithFactoryForTest("secrets", func() *cobra.Command {
		zipBytes := secretsCommandBitwardenZipForTest("bws", "#!/bin/sh\necho bws 2.0.0\n")
		asset, _ := externalsecrets.BitwardenAssetName(externalsecrets.BitwardenInstallOptions{})
		checksum := fmt.Sprintf("%x  %s\n", sha256.Sum256(zipBytes), asset)
		return NewSecretsCommand(SecretsOptions{
			BuildProvenance: func() SecretsBuildProvenance {
				return SecretsBuildProvenance{Version: Version, GitCommit: "test-git"}
			},
			BitwardenInstallDownload: func(_ context.Context, url string) ([]byte, error) {
				if strings.HasSuffix(url, asset) {
					return zipBytes, nil
				}
				return []byte(checksum), nil
			},
		})
	})
}

func secretsCommandBitwardenZipForTest(name, body string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create(name)
	_, _ = w.Write([]byte(body))
	_ = zw.Close()
	return buf.Bytes()
}

func assertSecretsSnapshotFile(t *testing.T, wantID, forbiddenSecret string) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(config.GormesHome(), "secrets-runtime.json"))
	if err != nil {
		t.Fatalf("read secrets snapshot: %v", err)
	}
	if strings.Contains(string(body), forbiddenSecret) {
		t.Fatalf("snapshot leaked secret %q:\n%s", forbiddenSecret, body)
	}
	if !strings.Contains(string(body), wantID) {
		t.Fatalf("snapshot missing SecretRef id %q:\n%s", wantID, body)
	}
}

func secretsCommandFindingPresent(findings []toolspkg.SecretsAuditFinding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}
