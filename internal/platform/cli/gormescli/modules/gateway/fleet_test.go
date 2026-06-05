package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	runtimegateway "github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func TestGatewayFleetJSONListsProfilesWithoutOpeningRuntimeStores(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	writeGatewayStatusConfig(t, []byte(`
config_version = 2

[profiles.main]
enabled = true
name = ""

[profiles.main.channels.telegram]
enabled = true
credential = "main-telegram"
allowed_users = ["42"]

[profiles.ops]
enabled = true
name = "Operations"
workspaces = ["/private/workspace/ops"]

[profiles.ops.channels.whatsapp]
enabled = true
credential = "ops-whatsapp"
allowed_chats = ["12025550123@s.whatsapp.net"]

[profiles.disabled]
enabled = false
name = "Dormant"

[credentials.main-telegram]
kind = "channel"
channel = "telegram"
owner_profile = "main"

[credentials.main-telegram.secret_ref]
source = "env"
id = "GORMES_MAIN_TELEGRAM_TOKEN"

[credentials.ops-whatsapp]
kind = "channel"
channel = "whatsapp"
owner_profile = "ops"

[credentials.ops-whatsapp.secret_ref]
source = "env"
id = "GORMES_OPS_WHATSAPP_TOKEN"
`))

	stdout, stderr, err := executeGatewayFleetCommand(t, "--json")
	if err != nil {
		t.Fatalf("gateway fleet --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Status runtimegateway.FleetStatus `json:"status"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Build.Version != testGatewayVersion || got.Build.GitCommit == "" {
		t.Fatalf("build provenance missing/wrong: %+v", got.Build)
	}
	if len(got.Status.Profiles) != 3 {
		t.Fatalf("profiles = %d, want every configured profile: %+v", len(got.Status.Profiles), got.Status.Profiles)
	}
	main := findFleetProfile(t, got.Status, "main")
	ops := findFleetProfile(t, got.Status, "ops")
	disabled := findFleetProfile(t, got.Status, "disabled")
	if !main.Enabled || !ops.Enabled || disabled.Enabled {
		t.Fatalf("enabled flags main=%v ops=%v disabled=%v", main.Enabled, ops.Enabled, disabled.Enabled)
	}
	if main.ProfileHomeHash == "" || ops.ProfileHomeHash == "" || main.ProfileHomeHash == ops.ProfileHomeHash {
		t.Fatalf("home hashes main=%q ops=%q, want isolated profile home evidence", main.ProfileHomeHash, ops.ProfileHomeHash)
	}
	if findFleetChannel(t, main, "telegram").Channel != "telegram" || findFleetChannel(t, ops, "whatsapp").Channel != "whatsapp" {
		t.Fatalf("desired channels not rendered: main=%+v ops=%+v", main.Channels, ops.Channels)
	}
	for _, leaked := range []string{"GORMES_MAIN_TELEGRAM_TOKEN", "GORMES_OPS_WHATSAPP_TOKEN", "12025550123", "/private/workspace", config.GormesHome()} {
		if strings.Contains(stdout+stderr, leaked) {
			t.Fatalf("gateway fleet leaked sensitive value %q:\nstdout=%s\nstderr=%s", leaked, stdout, stderr)
		}
	}
	assertGatewayStatusDidNotOpenRuntimeStores(t)
}

func TestGatewayFleetJSONResolvesSecretRefTokenHashesWithoutLeaking(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	const sharedToken = "123456:shared-token-that-must-not-leak"
	t.Setenv("GORMES_MAIN_TELEGRAM_TOKEN", sharedToken)
	t.Setenv("GORMES_OPS_TELEGRAM_TOKEN", sharedToken)
	writeGatewayStatusConfig(t, []byte(`
config_version = 2

[profiles.main]
enabled = true
[profiles.main.channels.telegram]
enabled = true
credential = "main-telegram"
allowed_users = ["42"]

[profiles.ops]
enabled = true
[profiles.ops.channels.telegram]
enabled = true
credential = "ops-telegram"
allowed_users = ["99"]

[credentials.main-telegram]
kind = "channel"
channel = "telegram"
owner_profile = "main"
[credentials.main-telegram.secret_ref]
source = "env"
id = "GORMES_MAIN_TELEGRAM_TOKEN"

[credentials.ops-telegram]
kind = "channel"
channel = "telegram"
owner_profile = "ops"
[credentials.ops-telegram.secret_ref]
source = "env"
id = "GORMES_OPS_TELEGRAM_TOKEN"
`))

	stdout, stderr, err := executeGatewayFleetCommand(t, "--json")
	if err != nil {
		t.Fatalf("gateway fleet --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var got struct {
		Status runtimegateway.FleetStatus `json:"status"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	sharedHash := runtimegateway.TokenCredentialHash(sharedToken)
	for _, profileID := range []string{"main", "ops"} {
		profile := findFleetProfile(t, got.Status, profileID)
		channel := findFleetChannel(t, profile, "telegram")
		if channel.CredentialHash != sharedHash {
			t.Fatalf("%s credential hash = %q, want %q", profileID, channel.CredentialHash, sharedHash)
		}
		if channel.Ready || !hasGatewayFleetEvidence(channel.Evidence, runtimegateway.ProfileChannelEvidenceTokenHashConflict) {
			t.Fatalf("%s channel = %+v, want duplicate token conflict evidence", profileID, channel)
		}
	}
	for _, leaked := range []string{sharedToken, "shared-token", "GORMES_MAIN_TELEGRAM_TOKEN", "GORMES_OPS_TELEGRAM_TOKEN"} {
		if strings.Contains(stdout+stderr, leaked) {
			t.Fatalf("gateway fleet leaked sensitive value %q:\nstdout=%s\nstderr=%s", leaked, stdout, stderr)
		}
	}
}

func TestGatewayFleetRestartAllJSONUsesSupervisorSeam(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	writeGatewayStatusConfig(t, []byte(`
config_version = 2
[profiles.main]
enabled = true
`))
	previous := newGatewayFleetSupervisor
	fake := &fakeGatewayFleetSupervisor{
		restartReport: runtimegateway.FleetOperationReport{
			Action:  runtimegateway.FleetOperationRestartAll,
			Results: []runtimegateway.FleetOperationResult{{ProfileID: "main", Status: runtimegateway.FleetOperationStatusRestarted, RuntimeOwner: runtimegateway.FleetRuntimeOwnerProfileServiceBridge}},
		},
	}
	newGatewayFleetSupervisor = func(config.Config) gatewayFleetSupervisor { return fake }
	defer func() { newGatewayFleetSupervisor = previous }()

	stdout, stderr, err := executeGatewayFleetCommand(t, "restart-all", "--json")
	if err != nil {
		t.Fatalf("gateway fleet restart-all --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if fake.restartCalls != 1 {
		t.Fatalf("restart calls = %d, want 1", fake.restartCalls)
	}
	var got struct {
		Build  gormescli.BuildProvenance           `json:"build"`
		Report runtimegateway.FleetOperationReport `json:"report"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Report.Action != runtimegateway.FleetOperationRestartAll || len(got.Report.Results) != 1 || got.Report.Results[0].Status != runtimegateway.FleetOperationStatusRestarted {
		t.Fatalf("report = %+v, want one restarted profile", got.Report)
	}
}

func executeGatewayFleetCommand(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd := NewFleetCommand(testGatewayOptions())
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func findFleetProfile(t *testing.T, status runtimegateway.FleetStatus, profileID string) runtimegateway.FleetProfileStatus {
	t.Helper()
	for _, profile := range status.Profiles {
		if profile.ProfileID == profileID {
			return profile
		}
	}
	t.Fatalf("profile %q not found in %+v", profileID, status.Profiles)
	return runtimegateway.FleetProfileStatus{}
}

func findFleetChannel(t *testing.T, profile runtimegateway.FleetProfileStatus, channel string) runtimegateway.FleetProfileChannelStatus {
	t.Helper()
	for _, got := range profile.Channels {
		if got.Channel == channel {
			return got
		}
	}
	t.Fatalf("channel %q not found in %+v", channel, profile.Channels)
	return runtimegateway.FleetProfileChannelStatus{}
}

func hasGatewayFleetEvidence(evidence []runtimegateway.ProfileChannelReadinessEvidence, code string) bool {
	for _, item := range evidence {
		if item.Code == code {
			return true
		}
	}
	return false
}

type fakeGatewayFleetSupervisor struct {
	status        runtimegateway.FleetStatus
	restartReport runtimegateway.FleetOperationReport
	restartCalls  int
}

func (f *fakeGatewayFleetSupervisor) Status(context.Context) (runtimegateway.FleetStatus, error) {
	return f.status, nil
}

func (f *fakeGatewayFleetSupervisor) StartAll(context.Context) (runtimegateway.FleetOperationReport, error) {
	return runtimegateway.FleetOperationReport{Action: runtimegateway.FleetOperationStartAll}, nil
}

func (f *fakeGatewayFleetSupervisor) StopAll(context.Context) (runtimegateway.FleetOperationReport, error) {
	return runtimegateway.FleetOperationReport{Action: runtimegateway.FleetOperationStopAll}, nil
}

func (f *fakeGatewayFleetSupervisor) RestartAll(context.Context) (runtimegateway.FleetOperationReport, error) {
	f.restartCalls++
	return f.restartReport, nil
}
