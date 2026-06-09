package gateway

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type fleetStatusHarness struct {
	cfg      config.Config
	worker   *fakeFleetWorker
	resolver *fakeFleetSecretResolver
	hashes   map[string]string
	homeRoot string
}

func (h fleetStatusHarness) status(t *testing.T) FleetStatus {
	t.Helper()
	if h.homeRoot == "" {
		h.homeRoot = "/fleet/home"
	}
	worker := FleetProfileWorker(h.worker)
	if worker == nil {
		worker = &fakeFleetWorker{}
	}
	status, err := NewFleetSupervisor(h.cfg, FleetSupervisorOptions{
		HomeRoot:           h.homeRoot,
		CredentialHashes:   h.hashes,
		CredentialResolver: h.resolver,
		Worker:             worker,
	}).Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	return status
}

func TestFleetSupervisorStatusListsConfiguredProfilesWithIsolationAndTokenConflictEvidence(t *testing.T) {
	sharedHash := TokenCredentialHash("telegram-token-that-must-not-leak")
	cfg := config.Config{
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled: true,
				Channels: map[string]config.ProfileChannelCfg{
					"telegram": {Enabled: true, Credential: "main-telegram", AllowedUsers: []string{"42"}},
				},
			},
			"ops": {
				Enabled: true,
				Name:    "Operations",
				Channels: map[string]config.ProfileChannelCfg{
					"telegram": {Enabled: true, Credential: "ops-telegram", AllowedUsers: []string{"99"}},
				},
			},
			"disabled": {
				Enabled: false,
				Name:    "Dormant",
				Channels: map[string]config.ProfileChannelCfg{
					"discord": {Enabled: true, Credential: "disabled-discord"},
				},
			},
		},
		Credentials: map[string]config.CredentialCfg{
			"main-telegram": channelCredential("telegram", "main", "GORMES_MAIN_TELEGRAM_TOKEN"),
			"ops-telegram":  channelCredential("telegram", "ops", "GORMES_OPS_TELEGRAM_TOKEN"),
		},
	}
	worker := &fakeFleetWorker{
		statuses: map[string]FleetProfileRuntime{
			"main": {Owner: FleetRuntimeOwnerProfileServiceBridge, Version: "v0.2.4", State: FleetRuntimeStateRunning},
			"ops":  {Owner: FleetRuntimeOwnerProfileServiceBridge, Version: "v0.2.4", State: FleetRuntimeStateStopped, LastError: "manual stop requested"},
		},
	}

	status := fleetStatusHarness{
		cfg:      cfg,
		worker:   worker,
		homeRoot: "/operator/private/gormes",
		hashes: map[string]string{
			"main-telegram": sharedHash,
			"ops-telegram":  sharedHash,
		},
	}.status(t)
	if len(status.Profiles) != 3 {
		t.Fatalf("profiles = %d, want every configured profile: %+v", len(status.Profiles), status.Profiles)
	}
	if status.Summary.ConfiguredProfiles != 3 || status.Summary.EnabledProfiles != 2 || status.Summary.ConflictProfiles != 2 {
		t.Fatalf("summary = %+v, want 3 configured, 2 enabled, 2 conflict/degraded", status.Summary)
	}

	disabled := findFleetProfile(t, status, "disabled")
	main := findFleetProfile(t, status, "main")
	ops := findFleetProfile(t, status, "ops")
	if disabled.Enabled || disabled.Health != FleetHealthDisabled {
		t.Fatalf("disabled profile = %+v, want disabled health without runtime worker", disabled)
	}
	if disabled.Runtime.State != "" {
		t.Fatalf("disabled runtime = %+v, want no worker status for disabled profile", disabled.Runtime)
	}
	if main.ProfileHomeHash == "" || ops.ProfileHomeHash == "" || main.ProfileHomeHash == ops.ProfileHomeHash {
		t.Fatalf("profile home hashes main=%q ops=%q, want non-empty distinct isolation evidence", main.ProfileHomeHash, ops.ProfileHomeHash)
	}
	for _, profile := range []FleetProfileStatus{main, ops} {
		if profile.Runtime.Owner != FleetRuntimeOwnerProfileServiceBridge {
			t.Fatalf("%s runtime owner = %q, want compatibility bridge owner", profile.ProfileID, profile.Runtime.Owner)
		}
		channel := findFleetChannel(t, profile, "telegram")
		if channel.Ready {
			t.Fatalf("%s channel Ready = true, want duplicate token conflict evidence: %+v", profile.ProfileID, channel)
		}
		if !hasProfileChannelEvidence(channel.Evidence, ProfileChannelEvidenceTokenHashConflict) {
			t.Fatalf("%s channel evidence = %+v, want token conflict", profile.ProfileID, channel.Evidence)
		}
		if channel.CredentialHash != sharedHash {
			t.Fatalf("%s credential hash = %q, want %q", profile.ProfileID, channel.CredentialHash, sharedHash)
		}
	}

	body, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("marshal status: %v", err)
	}
	for _, leaked := range []string{"telegram-token-that-must-not-leak", "GORMES_MAIN_TELEGRAM_TOKEN", "GORMES_OPS_TELEGRAM_TOKEN", "/operator/private/gormes"} {
		if strings.Contains(string(body), leaked) {
			t.Fatalf("fleet status leaked sensitive value %q:\n%s", leaked, body)
		}
	}
}

func TestFleetSupervisorStatusClampsNegativeRuntimePID(t *testing.T) {
	cfg := config.Config{Profiles: map[string]config.ProfileCfg{"main": {Enabled: true}}}
	worker := &fakeFleetWorker{statuses: map[string]FleetProfileRuntime{
		"main": {Owner: FleetRuntimeOwnerProfileServiceBridge, State: FleetRuntimeStateRunning, Live: true, PID: -42},
	}}

	status := fleetStatusHarness{cfg: cfg, worker: worker, homeRoot: t.TempDir()}.status(t)
	main := findFleetProfile(t, status, "main")
	if main.Runtime.PID < 0 {
		t.Fatalf("runtime PID = %d, want stale/corrupt negative PID clamped", main.Runtime.PID)
	}
	body, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("marshal status: %v", err)
	}
	if strings.Contains(string(body), `"pid":-`) {
		t.Fatalf("fleet status JSON leaked negative PID:\n%s", body)
	}
}

func TestFleetSupervisorStatusResolvesSecretRefTokenHashesForConflicts(t *testing.T) {
	const sharedToken = "123456:shared-secret-that-must-not-leak"
	sharedHash := TokenCredentialHash(sharedToken)
	cfg := config.Config{
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled: true,
				Channels: map[string]config.ProfileChannelCfg{
					"telegram": {Enabled: true, Credential: "main-telegram", AllowedUsers: []string{"42"}},
				},
			},
			"ops": {
				Enabled: true,
				Channels: map[string]config.ProfileChannelCfg{
					"telegram": {Enabled: true, Credential: "ops-telegram", AllowedUsers: []string{"99"}},
				},
			},
		},
		Credentials: map[string]config.CredentialCfg{
			"main-telegram": channelCredential("telegram", "main", "GORMES_MAIN_TELEGRAM_TOKEN"),
			"ops-telegram":  channelCredential("telegram", "ops", "GORMES_OPS_TELEGRAM_TOKEN"),
		},
	}
	resolver := &fakeFleetSecretResolver{values: map[string]string{
		"GORMES_MAIN_TELEGRAM_TOKEN": sharedToken,
		"GORMES_OPS_TELEGRAM_TOKEN":  sharedToken,
	}}

	status, err := NewFleetSupervisor(cfg, FleetSupervisorOptions{HomeRoot: "/fleet/home", CredentialResolver: resolver, Worker: &fakeFleetWorker{}}).Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	main := findFleetProfile(t, status, "main")
	ops := findFleetProfile(t, status, "ops")
	for _, profile := range []FleetProfileStatus{main, ops} {
		channel := findFleetChannel(t, profile, "telegram")
		if channel.CredentialHash != sharedHash {
			t.Fatalf("%s credential hash = %q, want %q", profile.ProfileID, channel.CredentialHash, sharedHash)
		}
		if channel.Ready || !hasProfileChannelEvidence(channel.Evidence, ProfileChannelEvidenceTokenHashConflict) {
			t.Fatalf("%s channel = %+v, want duplicate-token conflict evidence", profile.ProfileID, channel)
		}
	}
	if resolver.Count("GORMES_MAIN_TELEGRAM_TOKEN") != 1 || resolver.Count("GORMES_OPS_TELEGRAM_TOKEN") != 1 {
		t.Fatalf("resolver calls = %+v, want each SecretRef resolved once", resolver.calls)
	}
	body, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("marshal status: %v", err)
	}
	for _, leaked := range []string{sharedToken, "shared-secret", "GORMES_MAIN_TELEGRAM_TOKEN", "GORMES_OPS_TELEGRAM_TOKEN"} {
		if strings.Contains(string(body), leaked) {
			t.Fatalf("fleet status leaked sensitive value %q:\n%s", leaked, body)
		}
	}
}

func TestFleetSupervisorStatusReportsMissingSecretRefHashAsDegradedEvidence(t *testing.T) {
	cfg := config.Config{
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled: true,
				Channels: map[string]config.ProfileChannelCfg{
					"telegram": {Enabled: true, Credential: "main-telegram", AllowedUsers: []string{"42"}},
				},
			},
		},
		Credentials: map[string]config.CredentialCfg{
			"main-telegram": channelCredential("telegram", "main", "GORMES_MAIN_TELEGRAM_TOKEN"),
		},
	}
	resolver := &fakeFleetSecretResolver{values: map[string]string{}}

	status, err := NewFleetSupervisor(cfg, FleetSupervisorOptions{HomeRoot: "/fleet/home", CredentialResolver: resolver, Worker: &fakeFleetWorker{}}).Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	profile := findFleetProfile(t, status, "main")
	channel := findFleetChannel(t, profile, "telegram")
	if profile.Health != FleetHealthDegraded {
		t.Fatalf("profile health = %q, want degraded when token hash cannot be validated", profile.Health)
	}
	if channel.Ready || !hasProfileChannelEvidence(channel.Evidence, ProfileChannelEvidenceCredentialHashUnavailable) {
		t.Fatalf("channel = %+v, want credential hash unavailable evidence", channel)
	}
	body, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("marshal status: %v", err)
	}
	if strings.Contains(string(body), "GORMES_MAIN_TELEGRAM_TOKEN") {
		t.Fatalf("fleet status leaked SecretRef id:\n%s", body)
	}
}

func TestCommandFleetWorkerStartAllUsesProfileHomeEnvAndDoesNotRestartLiveProfiles(t *testing.T) {
	cfg := config.Config{Profiles: map[string]config.ProfileCfg{
		"main": {Enabled: true},
		"ops":  {Enabled: true},
	}}
	runner := &fakeFleetCommandRunner{results: []FleetCommandResult{{Stdout: `{"action":"started","mode":"runtime","new_pid":5151}`}}}
	worker := NewCommandFleetWorker(CommandFleetWorkerOptions{
		Command: "/usr/local/bin/gormes",
		Env:     []string{"PATH=/bin", "GORMES_HOME=/wrong"},
		Runner:  runner,
		StatusWorker: &fakeFleetWorker{statuses: map[string]FleetProfileRuntime{
			"main": {Owner: FleetRuntimeOwnerProfileServiceBridge, State: FleetRuntimeStateRunning, Live: true, PID: 4141},
			"ops":  {Owner: FleetRuntimeOwnerProfileServiceBridge, State: FleetRuntimeStateStopped, Live: false},
		}},
	})

	report, err := NewFleetSupervisor(cfg, FleetSupervisorOptions{HomeRoot: "/fleet/home", Worker: worker}).StartAll(context.Background())
	if err != nil {
		t.Fatalf("StartAll: %v", err)
	}
	if report.Summary.TargetedProfiles != 2 || report.Summary.Succeeded != 2 {
		t.Fatalf("summary = %+v, want two successful start results", report.Summary)
	}
	main := findFleetOperationResult(t, report, "main")
	ops := findFleetOperationResult(t, report, "ops")
	if main.Status != FleetOperationStatusStarted || !strings.Contains(main.Message, "already running") {
		t.Fatalf("main result = %+v, want already-running start success", main)
	}
	if ops.Status != FleetOperationStatusStarted || ops.RuntimeOwner != FleetRuntimeOwnerProfileCommandWorker {
		t.Fatalf("ops result = %+v, want started by command worker", ops)
	}
	if len(runner.commands) != 1 {
		t.Fatalf("commands = %+v, want only stopped profile command", runner.commands)
	}
	cmd := runner.commands[0]
	if cmd.Command != "/usr/local/bin/gormes" || strings.Join(cmd.Args, " ") != "gateway restart --json --service gormes-gateway-ops.service" {
		t.Fatalf("command = %+v, want profile-scoped gateway restart", cmd)
	}
	if got := fakeFleetCommandEnvValue(cmd.Env, "GORMES_HOME"); got != "/fleet/home/profiles/ops" {
		t.Fatalf("GORMES_HOME = %q, want ops profile home", got)
	}
	body, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	for _, leaked := range []string{"/fleet/home", "/wrong"} {
		if strings.Contains(string(body), leaked) {
			t.Fatalf("operation report leaked private path %q:\n%s", leaked, body)
		}
	}
}

func TestCommandFleetWorkerReportSanitizesCommandJSONFields(t *testing.T) {
	cfg := config.Config{Profiles: map[string]config.ProfileCfg{"main": {Enabled: true}}}
	runner := &fakeFleetCommandRunner{results: []FleetCommandResult{{Stdout: `{"action":"started\nstatus: forged api_key=plain-secret-token","mode":"runtime"}`}}}
	worker := NewCommandFleetWorker(CommandFleetWorkerOptions{
		Command: "gormes",
		Runner:  runner,
		StatusWorker: &fakeFleetWorker{statuses: map[string]FleetProfileRuntime{
			"main": {Owner: FleetRuntimeOwnerProfileServiceBridge, State: FleetRuntimeStateStopped, Live: false},
		}},
	})

	report, err := NewFleetSupervisor(cfg, FleetSupervisorOptions{HomeRoot: t.TempDir(), Worker: worker}).StartAll(context.Background())
	if err != nil {
		t.Fatalf("StartAll: %v", err)
	}
	result := findFleetOperationResult(t, report, "main")
	for _, forbidden := range []string{"\nstatus: forged", "plain-secret-token", "api_key"} {
		if strings.Contains(result.Message, forbidden) {
			t.Fatalf("fleet operation message leaked unsafe command field %q in %q", forbidden, result.Message)
		}
	}
	if !strings.Contains(result.Message, "action=started status: forged [redacted]") {
		t.Fatalf("fleet operation message missing sanitized action evidence: %q", result.Message)
	}
}

func TestCommandFleetWorkerStopAndRestartAllUseProfileScopedCommands(t *testing.T) {
	cfg := config.Config{Profiles: map[string]config.ProfileCfg{
		"main": {Enabled: true},
		"ops":  {Enabled: true},
	}}
	runner := &fakeFleetCommandRunner{results: []FleetCommandResult{
		{Stdout: `{"action":"stopped","final_status":"stale_pid"}`},
		{Stdout: `{"action":"noop","initial_status":"missing"}`},
		{Stdout: `{"action":"restarted","mode":"runtime","old_pid":100,"new_pid":200}`},
		{Stdout: `{"action":"restarted","mode":"service","service":"gormes-gateway-ops.service","outcome":"restarted"}`},
	}}
	worker := NewCommandFleetWorker(CommandFleetWorkerOptions{Command: "gormes", Env: []string{"PATH=/bin"}, Runner: runner})
	supervisor := NewFleetSupervisor(cfg, FleetSupervisorOptions{HomeRoot: "/fleet/home", Worker: worker})

	stopReport, err := supervisor.StopAll(context.Background())
	if err != nil {
		t.Fatalf("StopAll: %v", err)
	}
	if stopReport.Summary.TargetedProfiles != 2 || stopReport.Summary.Succeeded != 2 {
		t.Fatalf("stop summary = %+v, want two successes", stopReport.Summary)
	}
	restartReport, err := supervisor.RestartAll(context.Background())
	if err != nil {
		t.Fatalf("RestartAll: %v", err)
	}
	if restartReport.Summary.TargetedProfiles != 2 || restartReport.Summary.Succeeded != 2 {
		t.Fatalf("restart summary = %+v, want two successes", restartReport.Summary)
	}
	if len(runner.commands) != 4 {
		t.Fatalf("commands = %+v, want two stop and two restart commands", runner.commands)
	}
	want := []struct {
		args string
		home string
	}{
		{args: "gateway stop --json", home: "/fleet/home"},
		{args: "gateway stop --json", home: "/fleet/home/profiles/ops"},
		{args: "gateway restart --json --service gormes-gateway.service", home: "/fleet/home"},
		{args: "gateway restart --json --service gormes-gateway-ops.service", home: "/fleet/home/profiles/ops"},
	}
	for i, wantCommand := range want {
		cmd := runner.commands[i]
		if strings.Join(cmd.Args, " ") != wantCommand.args {
			t.Fatalf("command %d args = %q, want %q", i, strings.Join(cmd.Args, " "), wantCommand.args)
		}
		if got := fakeFleetCommandEnvValue(cmd.Env, "GORMES_HOME"); got != wantCommand.home {
			t.Fatalf("command %d GORMES_HOME = %q, want %q", i, got, wantCommand.home)
		}
	}
	if findFleetOperationResult(t, restartReport, "main").Status != FleetOperationStatusRestarted || findFleetOperationResult(t, restartReport, "ops").Status != FleetOperationStatusRestarted {
		t.Fatalf("restart report = %+v, want restarted statuses", restartReport)
	}
}

func TestFleetSupervisorRestartAllUsesIsolatedEnabledProfileTargets(t *testing.T) {
	cfg := config.Config{Profiles: map[string]config.ProfileCfg{
		"main":     {Enabled: true},
		"ops":      {Enabled: true, Name: "Operations"},
		"disabled": {Enabled: false},
	}}
	worker := &fakeFleetWorker{operationStatus: FleetOperationStatusRestarted}
	supervisor := NewFleetSupervisor(cfg, FleetSupervisorOptions{HomeRoot: "/tmp/gormes-fleet", Worker: worker})

	report, err := supervisor.RestartAll(context.Background())
	if err != nil {
		t.Fatalf("RestartAll: %v", err)
	}
	if report.Action != FleetOperationRestartAll {
		t.Fatalf("action = %q, want %q", report.Action, FleetOperationRestartAll)
	}
	if len(report.Results) != 2 || len(worker.restartTargets) != 2 {
		t.Fatalf("restart results=%d targets=%d, want enabled profiles only; report=%+v targets=%+v", len(report.Results), len(worker.restartTargets), report, worker.restartTargets)
	}
	mainTarget := worker.restartTargets[0]
	opsTarget := worker.restartTargets[1]
	if mainTarget.ProfileID != "main" || opsTarget.ProfileID != "ops" {
		t.Fatalf("restart target order = %q, %q; want main, ops", mainTarget.ProfileID, opsTarget.ProfileID)
	}
	if mainTarget.HomeRoot != "/tmp/gormes-fleet" {
		t.Fatalf("main home root = %q, want root profile home", mainTarget.HomeRoot)
	}
	if opsTarget.HomeRoot != "/tmp/gormes-fleet/profiles/ops" {
		t.Fatalf("ops home root = %q, want named profile home", opsTarget.HomeRoot)
	}
	if mainTarget.RuntimeStatusPath == opsTarget.RuntimeStatusPath || !strings.HasSuffix(opsTarget.RuntimeStatusPath, "/profiles/ops/runtime/gateway_state.json") {
		t.Fatalf("runtime status paths main=%q ops=%q, want isolated per-profile paths", mainTarget.RuntimeStatusPath, opsTarget.RuntimeStatusPath)
	}
}

type fakeFleetCommandRunner struct {
	commands []FleetCommand
	results  []FleetCommandResult
	err      error
}

func (r *fakeFleetCommandRunner) Run(_ context.Context, cmd FleetCommand) (FleetCommandResult, error) {
	r.commands = append(r.commands, cmd)
	if r.err != nil {
		return FleetCommandResult{}, r.err
	}
	if len(r.results) == 0 {
		return FleetCommandResult{}, nil
	}
	result := r.results[0]
	r.results = r.results[1:]
	return result, nil
}

func fakeFleetCommandEnvValue(env []string, key string) string {
	prefix := key + "="
	var out string
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			out = strings.TrimPrefix(entry, prefix)
		}
	}
	return out
}

func findFleetOperationResult(t *testing.T, report FleetOperationReport, profileID string) FleetOperationResult {
	t.Helper()
	for _, result := range report.Results {
		if result.ProfileID == profileID {
			return result
		}
	}
	t.Fatalf("profile %q not found in report %+v", profileID, report.Results)
	return FleetOperationResult{}
}

type fakeFleetSecretResolver struct {
	values map[string]string
	calls  map[string]int
}

func (r *fakeFleetSecretResolver) ResolveString(ref config.SecretRef) (string, config.SecretRefEvidence, error) {
	if r.calls == nil {
		r.calls = map[string]int{}
	}
	r.calls[ref.ID]++
	evidence := config.SecretRefEvidence{Source: string(ref.Source), Provider: ref.Provider, ID: ref.ID, Redacted: true}
	if value := strings.TrimSpace(r.values[ref.ID]); value != "" {
		evidence.Code = config.SecretRefEvidenceResolved
		return value, evidence, nil
	}
	evidence.Code = config.SecretRefEvidenceMissing
	return "", evidence, errFakeFleetSecretMissing(ref.ID)
}

func (r *fakeFleetSecretResolver) Count(id string) int {
	if r == nil || r.calls == nil {
		return 0
	}
	return r.calls[id]
}

type errFakeFleetSecretMissing string

func (e errFakeFleetSecretMissing) Error() string { return "missing secret ref " + string(e) }

type fakeFleetWorker struct {
	statuses        map[string]FleetProfileRuntime
	operationStatus FleetOperationStatus
	startTargets    []FleetProfileTarget
	stopTargets     []FleetProfileTarget
	restartTargets  []FleetProfileTarget
}

func (f *fakeFleetWorker) Status(_ context.Context, target FleetProfileTarget) (FleetProfileRuntime, error) {
	if status, ok := f.statuses[target.ProfileID]; ok {
		return status, nil
	}
	return FleetProfileRuntime{Owner: FleetRuntimeOwnerProfileServiceBridge, State: FleetRuntimeStateMissing}, nil
}

func (f *fakeFleetWorker) Start(_ context.Context, target FleetProfileTarget) (FleetOperationEvidence, error) {
	f.startTargets = append(f.startTargets, target)
	return FleetOperationEvidence{Status: f.operationStatus, RuntimeOwner: FleetRuntimeOwnerProfileServiceBridge}, nil
}

func (f *fakeFleetWorker) Stop(_ context.Context, target FleetProfileTarget) (FleetOperationEvidence, error) {
	f.stopTargets = append(f.stopTargets, target)
	return FleetOperationEvidence{Status: f.operationStatus, RuntimeOwner: FleetRuntimeOwnerProfileServiceBridge}, nil
}

func (f *fakeFleetWorker) Restart(_ context.Context, target FleetProfileTarget) (FleetOperationEvidence, error) {
	f.restartTargets = append(f.restartTargets, target)
	return FleetOperationEvidence{Status: f.operationStatus, RuntimeOwner: FleetRuntimeOwnerProfileServiceBridge}, nil
}

func findFleetProfile(t *testing.T, status FleetStatus, profileID string) FleetProfileStatus {
	t.Helper()
	for _, profile := range status.Profiles {
		if profile.ProfileID == profileID {
			return profile
		}
	}
	t.Fatalf("profile %q not found in %+v", profileID, status.Profiles)
	return FleetProfileStatus{}
}

func findFleetChannel(t *testing.T, profile FleetProfileStatus, channel string) FleetProfileChannelStatus {
	t.Helper()
	for _, got := range profile.Channels {
		if got.Channel == channel {
			return got
		}
	}
	t.Fatalf("channel %q not found in %+v", channel, profile.Channels)
	return FleetProfileChannelStatus{}
}
