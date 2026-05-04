package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestSecretsConfigureCommandOutputsTypedRefWithoutSecret(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("OPENROUTER_API_KEY", "sk-openrouter-secret")

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd,
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

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "secrets", "apply", "--plan", goodPlan, "--json")
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
	cmd = newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err = executeOneshotFlagCommand(cmd, "secrets", "audit", "--plan", auditPlan, "--json")
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

	cmd = newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err = executeOneshotFlagCommand(cmd, "secrets", "reload", "--plan", auditPlan, "--json")
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
