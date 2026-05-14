package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/doctor"
)

// TestDoctorCommand_OfflineRoutedThroughCobra proves that `gormes
// doctor --offline` writes its output through cmd.OutOrStdout() (so
// tests can capture it via cmd.SetOut) and returns a normal RunE error
// instead of calling os.Exit on failure paths. This is the
// testability-enabling refactor: previously the command hard-exited
// the test process and bypassed cobra's stdout writer, so end-to-end
// fixtures were impossible.
func TestDoctorCommand_OfflineRoutedThroughCobra(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--offline"})
	_ = cmd.Execute()

	combined := stdout.String() + stderr.String()
	if combined == "" {
		t.Fatalf("doctor --offline produced no captured output; output likely went to os.Stdout/Stderr instead of cobra writers")
	}
	if !strings.Contains(combined, "Toolbox") {
		t.Fatalf("doctor --offline output should mention Toolbox check; got:\n%s", combined)
	}
}

// TestDoctorCommand_JSONIncludesBuildProvenance proves
// `gormes doctor --json` carries the running binary's build SHA and
// version. Same contract as `update --json`'s `build` block — fleet
// health snapshots stay attributable to a specific binary even when
// captured and shipped off-host.
func TestDoctorCommand_JSONIncludesBuildProvenance(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--offline", "--json"})
	_ = cmd.Execute()

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout.String(), err)
	}
	if got.Build.Version != Version {
		t.Fatalf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Build.GitCommit == "" {
		t.Fatalf("got.build.git_commit must be non-empty")
	}
}

// TestDoctorCommand_JSONFieldOrderPutsFailedBeforeChecks proves the
// JSON output uses a stable field order with summary fields (`failed`)
// before the per-check array. This matches `update --json`'s
// convention so downstream tooling that pretty-prints / diffs JSON
// reports gets a predictable structure across surfaces. Relying on
// `map[string]any` alphabetic sort would put `checks` before `failed`
// — inconsistent with the rest of the --json arc.
func TestDoctorCommand_JSONFieldOrderPutsFailedBeforeChecks(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--offline", "--json"})
	_ = cmd.Execute()

	body := stdout.String()
	failedIdx := strings.Index(body, `"failed"`)
	checksIdx := strings.Index(body, `"checks"`)
	if failedIdx < 0 || checksIdx < 0 {
		t.Fatalf("output missing failed/checks fields:\n%s", body)
	}
	if failedIdx >= checksIdx {
		t.Fatalf("`failed` must precede `checks` in JSON for stable consumer rendering; got failedIdx=%d checksIdx=%d", failedIdx, checksIdx)
	}
}

// TestDoctorCommand_JSONReportsFailedFieldFromWorstCheck proves the
// JSON document carries a top-level "failed" boolean derived from the
// worst-status check encountered. Monitoring consumers branch on this
// field rather than scanning every entry — same contract as
// `gormes update --json` and friends.
func TestDoctorCommand_JSONReportsFailedFieldFromWorstCheck(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--offline", "--json"})
	_ = cmd.Execute()

	// Parse as a generic map first to assert the `failed` key is
	// PRESENT (zero-value false would otherwise let a missing field
	// silently pass through json.Unmarshal into a struct).
	var raw map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout.String(), err)
	}
	gotFailedRaw, hasFailed := raw["failed"]
	if !hasFailed {
		t.Fatalf("JSON must include top-level `failed` boolean; got keys=%v", mapKeys(raw))
	}
	gotFailed, ok := gotFailedRaw.(bool)
	if !ok {
		t.Fatalf("`failed` must be a bool; got %T %v", gotFailedRaw, gotFailedRaw)
	}

	// Recompute expected from collected check statuses so the test
	// stays correct across host-environment variation.
	checksRaw, _ := raw["checks"].([]any)
	wantFailed := false
	for _, entry := range checksRaw {
		m, _ := entry.(map[string]any)
		if status, _ := m["status"].(string); status == "FAIL" {
			wantFailed = true
			break
		}
	}
	if gotFailed != wantFailed {
		t.Fatalf("got.failed = %t, want %t (any FAIL status implies failed=true)", gotFailed, wantFailed)
	}
}

// TestDoctorCommand_SourceBuildIdentitySummary proves that when the
// binary was built without ldflags-injected provenance (the default
// `go run` / `go build` path leaves GitCommit at the literal sentinel
// "unknown"), the doctor summary labels it as a "source build". A bare
// `commit=unknown` summary is technically accurate but cryptic — it
// looks like a malformed value. The explicit label tells operators
// "this binary wasn't built by the release pipeline" without forcing
// them to know what `unknown` means in this codebase.
func TestDoctorCommand_SourceBuildIdentitySummary(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	prevDirty := GitDirty
	prevCommit := GitCommit
	GitDirty = "false"
	GitCommit = "unknown"
	t.Cleanup(func() {
		GitDirty = prevDirty
		GitCommit = prevCommit
	})

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--offline", "--json"})
	_ = cmd.Execute()

	var got struct {
		Checks []struct {
			Name    string `json:"name"`
			Status  string `json:"status"`
			Summary string `json:"summary"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout.String(), err)
	}
	for _, c := range got.Checks {
		if c.Name != "build identity" {
			continue
		}
		if c.Status != "PASS" {
			t.Fatalf("source build identity status = %q, want PASS", c.Status)
		}
		if !strings.Contains(c.Summary, "source build") {
			t.Fatalf("summary must label sentinel-commit builds as `source build`; got %q", c.Summary)
		}
		if strings.Contains(c.Summary, "commit=unknown") {
			t.Fatalf("summary must NOT show bare `commit=unknown`; replaced with `source build` label; got %q", c.Summary)
		}
		return
	}
	t.Fatalf("doctor must emit `build identity` check; got checks=%+v", got.Checks)
}

// TestDoctorCommand_DirtyBuildEmitsBuildIdentityWarning proves that when
// the binary was built from a dirty source tree (`-X main.GitDirty=true`
// at build time), `gormes doctor` surfaces an explicit warn-status
// "build identity" check. Operators reading doctor output should know
// they are NOT running a clean release artifact — otherwise stale or
// uncommitted local changes silently ride along into production with no
// signal to the operator.
func TestDoctorCommand_DirtyBuildEmitsBuildIdentityWarning(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	prev := GitDirty
	GitDirty = "true"
	t.Cleanup(func() { GitDirty = prev })

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--offline", "--json"})
	_ = cmd.Execute()

	var got struct {
		Checks []struct {
			Name    string `json:"name"`
			Status  string `json:"status"`
			Summary string `json:"summary"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout.String(), err)
	}
	var found bool
	for _, c := range got.Checks {
		if c.Name == "build identity" {
			found = true
			if c.Status != "WARN" {
				t.Fatalf("build identity status = %q, want WARN for dirty build", c.Status)
			}
			if !strings.Contains(c.Summary, "dirty") {
				t.Fatalf("build identity summary should mention 'dirty'; got %q", c.Summary)
			}
		}
	}
	if !found {
		t.Fatalf("doctor must emit `build identity` check on dirty builds; got checks=%+v", got.Checks)
	}
}

// TestDoctorCommand_CleanBuildEmitsBuildIdentityPass proves that on a
// clean build (the default), `gormes doctor` reports a PASS-status
// "build identity" check naming the version + short SHA. The check
// must be present in BOTH dirty and clean states so consumers always
// see binary identity in the snapshot — not only when something is
// wrong.
func TestDoctorCommand_CleanBuildEmitsBuildIdentityPass(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	prev := GitDirty
	GitDirty = "false"
	t.Cleanup(func() { GitDirty = prev })

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--offline", "--json"})
	_ = cmd.Execute()

	var got struct {
		Checks []struct {
			Name    string `json:"name"`
			Status  string `json:"status"`
			Summary string `json:"summary"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout.String(), err)
	}
	var found bool
	for _, c := range got.Checks {
		if c.Name == "build identity" {
			found = true
			if c.Status != "PASS" {
				t.Fatalf("build identity status = %q, want PASS for clean build", c.Status)
			}
			if !strings.Contains(c.Summary, Version) {
				t.Fatalf("build identity summary must name version %q; got %q", Version, c.Summary)
			}
		}
	}
	if !found {
		t.Fatalf("doctor must emit `build identity` check on clean builds; got checks=%+v", got.Checks)
	}
}

func mapKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestDoctorCommand_OfflineJSONEmitsCheckArray proves
// `gormes doctor --offline --json` emits a parseable
// `{"checks": [...]}` document where each entry has the same fields
// the human surface renders. Monitoring/CI consumers can ingest
// fleet-wide doctor results without scraping the bracketed text.
func TestDoctorCommand_OfflineJSONEmitsCheckArray(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--offline", "--json"})
	_ = cmd.Execute()

	var got struct {
		Checks []struct {
			Name    string `json:"name"`
			Status  string `json:"status"`
			Summary string `json:"summary"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout.String(), err)
	}
	if len(got.Checks) == 0 {
		t.Fatalf("got 0 checks, want at least the SecretRef runtime + Toolbox + provider-skipped entries")
	}
	wantNames := map[string]bool{"Toolbox": false, "SecretRef runtime": false, "provider health": false}
	for _, c := range got.Checks {
		if _, ok := wantNames[c.Name]; ok {
			wantNames[c.Name] = true
		}
	}
	for name, found := range wantNames {
		if !found {
			t.Fatalf("checks array missing entry %q; got names=%v", name, checkNames(got.Checks))
		}
	}
	if strings.Contains(stdout.String(), "[PASS]") || strings.Contains(stdout.String(), "[WARN]") {
		t.Fatalf("--json must not emit bracketed human lines; got:\n%s", stdout.String())
	}
}

func TestDoctorTargetTelegramReportsMissingChannelCommand(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommand()
	stdout, stderr, err := executeRootCommandForTest(cmd, "doctor", "--offline", "--target", "telegram")
	if err != nil {
		t.Fatalf("doctor --offline --target telegram: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	for _, want := range []string{
		"target readiness",
		"not ready: provider endpoint is not configured",
		"Telegram channel is not configured",
		"gormes setup --quick --target telegram",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestDoctorJSONIncludesTargetReadiness(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommand()
	stdout, stderr, err := executeRootCommandForTest(cmd, "doctor", "--offline", "--target", "whatsapp", "--json")
	if err != nil {
		t.Fatalf("doctor --offline --target whatsapp --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Failed bool `json:"failed"`
		Target struct {
			Name        string   `json:"name"`
			Ready       bool     `json:"ready"`
			Summary     string   `json:"summary"`
			NextCommand string   `json:"next_command"`
			Missing     []string `json:"missing"`
		} `json:"target"`
		Checks []struct {
			Name string `json:"name"`
		} `json:"checks"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Fatalf("build.version = %q, want %q", got.Build.Version, Version)
	}
	if len(got.Checks) == 0 {
		t.Fatalf("existing checks array must remain populated; stdout=%s", stdout)
	}
	if got.Target.Name != "whatsapp" {
		t.Fatalf("target.name = %q, want whatsapp", got.Target.Name)
	}
	if got.Target.Ready {
		t.Fatalf("target.ready = true, want false for fresh home")
	}
	if !strings.Contains(got.Target.Summary, "provider endpoint is not configured") {
		t.Fatalf("target.summary = %q", got.Target.Summary)
	}
	if got.Target.NextCommand != "gormes setup --quick --target whatsapp" {
		t.Fatalf("target.next_command = %q", got.Target.NextCommand)
	}
	wantMissing := []string{"provider", "auth", "channel"}
	if !reflect.DeepEqual(got.Target.Missing, wantMissing) {
		t.Fatalf("target.missing = %v, want %v", got.Target.Missing, wantMissing)
	}
}

func TestDoctorTargetReadinessUsesResolvedSecretRefAuth(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("GORMES_PROVIDER_SECRET", "sk-doctor-target-secretref")
	writeOneshotFlagConfig(t, []byte(`
[hermes]
endpoint = "https://provider.example/v1"
model = "fixture-model"

[hermes.api_key_ref]
source = "env"
id = "GORMES_PROVIDER_SECRET"

[slack]
enabled = true
allowed_channel_id = "C123"
`))

	cmd := newRootCommand()
	stdout, stderr, err := executeRootCommandForTest(cmd, "doctor", "--offline", "--target", "slack", "--json")
	if err != nil {
		t.Fatalf("doctor --offline --target slack --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	var got struct {
		Target struct {
			Name        string   `json:"name"`
			Ready       bool     `json:"ready"`
			Summary     string   `json:"summary"`
			NextCommand string   `json:"next_command"`
			Missing     []string `json:"missing"`
		} `json:"target"`
		Checks []struct {
			Name    string `json:"name"`
			Status  string `json:"status"`
			Summary string `json:"summary"`
		} `json:"checks"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Target.Name != "slack" {
		t.Fatalf("target.name = %q, want slack", got.Target.Name)
	}
	if !got.Target.Ready {
		t.Fatalf("target.ready = false, want true after SecretRef auth activation; target=%+v stdout=%s", got.Target, stdout)
	}
	if got.Target.NextCommand != "gormes gateway" {
		t.Fatalf("target.next_command = %q, want gormes gateway", got.Target.NextCommand)
	}
	if len(got.Target.Missing) != 0 {
		t.Fatalf("target.missing = %v, want none after SecretRef auth activation", got.Target.Missing)
	}
	for _, c := range got.Checks {
		if c.Name == "target readiness" && strings.Contains(c.Summary, "provider credential is not configured") {
			t.Fatalf("target readiness used pre-activation auth state: %+v\nstdout=%s", c, stdout)
		}
	}
	if strings.Contains(stdout, "sk-doctor-target-secretref") {
		t.Fatalf("doctor target JSON leaked resolved secret:\n%s", stdout)
	}
}

func TestDoctorTargetTerminalReadinessUsesResolvableSecretRefAuth(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	secret := "sk-terminal-secretref"
	t.Setenv("GORMES_PROVIDER_SECRET", secret)
	writeOneshotFlagConfig(t, []byte(`
[hermes]
endpoint = "https://provider.example/v1"
model = "fixture-model"

[hermes.api_key_ref]
source = "env"
id = "GORMES_PROVIDER_SECRET"
`))

	cmd := newRootCommand()
	stdout, stderr, err := executeRootCommandForTest(cmd, "doctor", "--offline", "--target", "terminal", "--json")
	if err != nil {
		t.Fatalf("doctor --offline --target terminal --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	var got struct {
		Target struct {
			Name    string   `json:"name"`
			Ready   bool     `json:"ready"`
			Missing []string `json:"missing"`
		} `json:"target"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Target.Name != "terminal" {
		t.Fatalf("target.name = %q, want terminal", got.Target.Name)
	}
	if !got.Target.Ready {
		t.Fatalf("target.ready = false, want true for resolvable API key SecretRef; target=%+v stdout=%s", got.Target, stdout)
	}
	if len(got.Target.Missing) != 0 {
		t.Fatalf("target.missing = %v, want none", got.Target.Missing)
	}
	if strings.Contains(stdout+stderr, secret) {
		t.Fatalf("doctor target readiness leaked resolved SecretRef value:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
}

func TestDoctorInvalidTargetJSONMarksFailed(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommand()
	stdout, stderr, err := executeRootCommandForTest(cmd, "doctor", "--offline", "--target", "pagerduty", "--json")
	if err == nil {
		t.Fatalf("doctor --target pagerduty --json err = nil, want nonzero failure\nstdout=%s\nstderr=%s", stdout, stderr)
	}

	var got struct {
		Failed bool `json:"failed"`
		Checks []struct {
			Name    string `json:"name"`
			Status  string `json:"status"`
			Summary string `json:"summary"`
		} `json:"checks"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must remain finalized JSON on invalid target: %v\nstdout=%s\nstderr=%s", jsonErr, stdout, stderr)
	}
	if !got.Failed {
		t.Fatalf("failed = false, want true for invalid target\nstdout=%s", stdout)
	}
	for _, c := range got.Checks {
		if c.Name == "target readiness" && c.Status == "FAIL" && strings.Contains(c.Summary, "unsupported target") {
			return
		}
	}
	t.Fatalf("checks missing FAIL target readiness unsupported-target entry: %+v\nstdout=%s", got.Checks, stdout)
}

func TestDoctorOfflineGatewayTokensDoNotConstructRealClientsOrLeakSecrets(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	telegramToken := "123456:offline-telegram-secret"
	discordToken := "discord-offline-secret"
	writeOneshotFlagConfig(t, []byte(`
[telegram]
bot_token = "`+telegramToken+`"
allowed_chat_id = 42

[discord]
token = "`+discordToken+`"
allowed_channel_id = "C123"
`))

	var telegramCalls, discordCalls int
	prevTelegram := doctorNewTelegramClient
	prevDiscord := doctorNewDiscordSession
	doctorNewTelegramClient = func(token string) error {
		telegramCalls++
		t.Fatalf("doctor --offline must not construct Telegram client; token=%q", token)
		return nil
	}
	doctorNewDiscordSession = func(token string) error {
		discordCalls++
		t.Fatalf("doctor --offline must not construct Discord session; token=%q", token)
		return nil
	}
	t.Cleanup(func() {
		doctorNewTelegramClient = prevTelegram
		doctorNewDiscordSession = prevDiscord
	})

	cmd := newRootCommand()
	stdout, stderr, err := executeRootCommandForTest(cmd, "doctor", "--offline", "--json")
	if err != nil {
		t.Fatalf("doctor --offline --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if telegramCalls != 0 || discordCalls != 0 {
		t.Fatalf("offline constructor calls = telegram:%d discord:%d, want zero", telegramCalls, discordCalls)
	}
	combined := stdout + stderr + errString(err)
	for _, secret := range []string{telegramToken, discordToken} {
		if strings.Contains(combined, secret) {
			t.Fatalf("doctor --offline leaked secret %q:\nstdout=%s\nstderr=%s\nerr=%v", secret, stdout, stderr, err)
		}
	}
	for _, want := range []string{
		`"name": "gateway/telegram"`,
		`"summary": "allowed_chat_id=42 (network validation skipped --offline)"`,
		`"name": "gateway/discord"`,
		`"summary": "allowed_channel_id=C123 (network validation skipped --offline)"`,
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestDoctorOfflineGitHubAuthDoesNotRunCLIAndStillReportsEnvToken(t *testing.T) {
	for _, tc := range []struct {
		name       string
		envToken   string
		wantStatus string
		wantText   string
	}{
		{name: "no env token", wantStatus: `"status": "SKIP"`, wantText: "skipped (--offline"},
		{name: "env token", envToken: "ghp_offline_secret", wantStatus: `"status": "PASS"`, wantText: "token configured"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupOneshotFlagTestEnv(t)
			t.Setenv("GITHUB_TOKEN", "")
			t.Setenv("GH_TOKEN", "")
			if tc.envToken != "" {
				t.Setenv("GITHUB_TOKEN", tc.envToken)
			}
			called := false
			prevRunner := doctorGitHubAuthRunner
			doctorGitHubAuthRunner = func(ctx context.Context) doctor.GitHubAuthStatusResult {
				called = true
				t.Fatalf("doctor --offline must not run gh auth status")
				return doctor.GitHubAuthStatusResult{}
			}
			t.Cleanup(func() { doctorGitHubAuthRunner = prevRunner })

			cmd := newRootCommand()
			stdout, stderr, err := executeRootCommandForTest(cmd, "doctor", "--offline", "--json")
			if err != nil {
				t.Fatalf("doctor --offline --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
			}
			if called {
				t.Fatalf("gh auth runner was called in offline mode")
			}
			if tc.envToken != "" && strings.Contains(stdout+stderr, tc.envToken) {
				t.Fatalf("doctor leaked GitHub token:\nstdout=%s\nstderr=%s", stdout, stderr)
			}
			if !strings.Contains(stdout, `"name": "GitHub auth"`) || !strings.Contains(stdout, tc.wantStatus) || !strings.Contains(stdout, tc.wantText) {
				t.Fatalf("GitHub auth offline report missing status/text %q/%q:\n%s", tc.wantStatus, tc.wantText, stdout)
			}
		})
	}
}

func TestDoctorGatewayInitErrorsRedactTokens(t *testing.T) {
	for _, tc := range []struct {
		name          string
		configBody    string
		secret        string
		replaceSeam   func(t *testing.T, secret string)
		wantCheckName string
	}{
		{
			name:   "telegram",
			secret: "123456:live-telegram-secret",
			configBody: `
[telegram]
bot_token = "123456:live-telegram-secret"
allowed_chat_id = 42
`,
			replaceSeam: func(t *testing.T, secret string) {
				prev := doctorNewTelegramClient
				doctorNewTelegramClient = func(token string) error {
					if token != secret {
						t.Fatalf("telegram token = %q, want fixture secret", token)
					}
					return fmt.Errorf("telegram getMe failed at https://api.telegram.org/bot%s/getMe", token)
				}
				t.Cleanup(func() { doctorNewTelegramClient = prev })
			},
			wantCheckName: "gateway/telegram",
		},
		{
			name:   "discord",
			secret: "discord-live-secret",
			configBody: `
[discord]
token = "discord-live-secret"
allowed_channel_id = "C123"
`,
			replaceSeam: func(t *testing.T, secret string) {
				prev := doctorNewDiscordSession
				doctorNewDiscordSession = func(token string) error {
					if token != secret {
						t.Fatalf("discord token = %q, want fixture secret", token)
					}
					return fmt.Errorf("discord session failed for Bot %s", token)
				}
				t.Cleanup(func() { doctorNewDiscordSession = prev })
			},
			wantCheckName: "gateway/discord",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupOneshotFlagTestEnv(t)
			writeOneshotFlagConfig(t, []byte(`
[hermes]
provider = "openai-codex"
model = "gpt-5.1-codex"
`+tc.configBody))
			if _, err := config.NewCodexOAuthStateStore(config.CodexOAuthStateStoreOptions{}).SaveTokens(config.CodexOAuthTokens{
				AccessToken:  "codex-access",
				RefreshToken: "codex-refresh",
			}); err != nil {
				t.Fatalf("save codex auth: %v", err)
			}
			tc.replaceSeam(t, tc.secret)

			cmd := newRootCommand()
			stdout, stderr, err := executeRootCommandForTest(cmd, "doctor", "--json")
			if err == nil {
				t.Fatalf("doctor --json err = nil, want gateway init failure\nstdout=%s\nstderr=%s", stdout, stderr)
			}
			combined := stdout + stderr + err.Error()
			if strings.Contains(combined, tc.secret) {
				t.Fatalf("doctor leaked %s token %q:\nstdout=%s\nstderr=%s\nerr=%v", tc.name, tc.secret, stdout, stderr, err)
			}
			if !strings.Contains(stdout, `"failed": true`) || !strings.Contains(stdout, `"name": "`+tc.wantCheckName+`"`) || !strings.Contains(stdout, "[REDACTED]") {
				t.Fatalf("doctor JSON missing redacted failure evidence:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
			}
		})
	}
}

func checkNames(checks []struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Summary string `json:"summary"`
}) []string {
	out := make([]string, len(checks))
	for i, c := range checks {
		out[i] = c.Name
	}
	return out
}
