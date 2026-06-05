package providers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestAuthBareEmptyPoolMessage(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth")
	if err != nil {
		t.Fatalf("auth: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "(no credentials configured)") {
		t.Fatalf("bare auth empty stdout = %q, want no-credentials message", stdout)
	}
}

func TestAuthBareCorruptStoreReportsTypedEvidence(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	if err := os.MkdirAll(config.GormesHome(), 0o700); err != nil {
		t.Fatalf("mkdir gormes home: %v", err)
	}
	if err := os.WriteFile(filepath.Join(config.GormesHome(), "auth.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatalf("write corrupt auth store: %v", err)
	}

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth")
	if err == nil {
		t.Fatalf("auth corrupt error = nil\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if !strings.Contains(err.Error(), "credential_pool_corrupt") {
		t.Fatalf("auth corrupt err = %v, want credential_pool_corrupt", err)
	}
	if strings.Contains(stdout+stderr+err.Error(), config.GormesHome()) {
		t.Fatalf("corrupt auth evidence leaked host path:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
	}
}

func TestAuthBareAWSIdentityOptionalProbe(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	oldProbe := authBareAWSIdentityProbe
	t.Cleanup(func() { authBareAWSIdentityProbe = oldProbe })

	authBareAWSIdentityProbe = func() (string, error) { return "arn:aws:sts::123456789012:assumed-role/gormes/test", nil }
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth")
	if err != nil {
		t.Fatalf("auth with AWS probe: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "bedrock identity: arn:aws:sts::123456789012:assumed-role/gormes/test") {
		t.Fatalf("stdout = %q, want successful bedrock identity", stdout)
	}

	authBareAWSIdentityProbe = func() (string, error) { return "", errAuthBareAWSIdentityUnavailable }
	cmd = newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err = executeOneshotFlagCommand(cmd, "auth")
	if err != nil {
		t.Fatalf("auth with failing AWS probe: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "bedrock identity: aws_identity_unavailable") {
		t.Fatalf("stdout = %q, want aws_identity_unavailable", stdout)
	}
}

func TestAuthBareNonTTYRendersPoolTable(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	seedAuthCommandCredentials(t, "openrouter", []config.PooledCredential{
		{ID: "openrouter-primary", Label: "primary", AuthType: config.CredentialAuthAPIKey, Source: "manual", AccessToken: "plain-openrouter-access", LastStatus: config.CredentialStatusOK},
		{ID: "openrouter-backup", Label: "backup", AuthType: config.CredentialAuthAPIKey, Source: "manual:rotated", AccessToken: "plain-openrouter-backup", LastStatus: config.CredentialStatusExhausted, LastErrorReason: "rate_limit"},
	})

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth")
	if err != nil {
		t.Fatalf("auth: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"openrouter (2 credentials):",
		"label\tauth_type\tsource\tstatus\tcurrent",
		"primary\tapi_key\tmanual\tok\t←",
		"backup\tapi_key\tmanual:rotated\texhausted(rate_limit)\t",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("bare auth stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, leak := range []string{"plain-openrouter-access", "plain-openrouter-backup"} {
		if strings.Contains(stdout+stderr, leak) {
			t.Fatalf("bare auth leaked %q:\nstdout=%s\nstderr=%s", leak, stdout, stderr)
		}
	}
}
