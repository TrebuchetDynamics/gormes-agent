package gateway

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

const (
	FleetRuntimeOwnerProfileServiceBridge = "profile_service_bridge"
	FleetRuntimeOwnerProfileCommandWorker = "profile_command_worker"

	FleetRuntimeStateMissing = "missing"
	FleetRuntimeStateRunning = "running"
	FleetRuntimeStateStopped = "stopped"
	FleetRuntimeStateError   = "error"

	FleetHealthHealthy  = "healthy"
	FleetHealthStopped  = "stopped"
	FleetHealthDegraded = "degraded"
	FleetHealthDisabled = "disabled"
)

type FleetOperation string

const (
	FleetOperationStartAll   FleetOperation = "start-all"
	FleetOperationStopAll    FleetOperation = "stop-all"
	FleetOperationRestartAll FleetOperation = "restart-all"
)

type FleetOperationStatus string

const (
	FleetOperationStatusStarted     FleetOperationStatus = "started"
	FleetOperationStatusStopped     FleetOperationStatus = "stopped"
	FleetOperationStatusRestarted   FleetOperationStatus = "restarted"
	FleetOperationStatusUnavailable FleetOperationStatus = "unavailable"
	FleetOperationStatusFailed      FleetOperationStatus = "failed"
)

type FleetSupervisor struct {
	cfg    config.Config
	opts   FleetSupervisorOptions
	worker FleetProfileWorker
}

type FleetSupervisorOptions struct {
	HomeRoot           string
	CredentialHashes   map[string]string
	CredentialResolver FleetSecretResolver
	Worker             FleetProfileWorker
}

type FleetSecretResolver interface {
	ResolveString(config.SecretRef) (string, config.SecretRefEvidence, error)
}

type FleetProfileWorker interface {
	Status(context.Context, FleetProfileTarget) (FleetProfileRuntime, error)
	Start(context.Context, FleetProfileTarget) (FleetOperationEvidence, error)
	Stop(context.Context, FleetProfileTarget) (FleetOperationEvidence, error)
	Restart(context.Context, FleetProfileTarget) (FleetOperationEvidence, error)
}

type FleetStatus struct {
	Summary  FleetSummary         `json:"summary"`
	Profiles []FleetProfileStatus `json:"profiles"`
}

type FleetSummary struct {
	ConfiguredProfiles int `json:"configured_profiles"`
	EnabledProfiles    int `json:"enabled_profiles"`
	HealthyProfiles    int `json:"healthy_profiles"`
	DegradedProfiles   int `json:"degraded_profiles"`
	ConflictProfiles   int `json:"conflict_profiles"`
	DisabledProfiles   int `json:"disabled_profiles"`
}

type FleetProfileStatus struct {
	ProfileID       string                      `json:"profile_id"`
	DisplayName     string                      `json:"display_name,omitempty"`
	Enabled         bool                        `json:"enabled"`
	Health          string                      `json:"health"`
	ProfileHomeHash string                      `json:"profile_home_hash"`
	Runtime         FleetProfileRuntime         `json:"runtime"`
	Channels        []FleetProfileChannelStatus `json:"channels,omitempty"`
}

type FleetProfileRuntime struct {
	Owner     string `json:"owner,omitempty"`
	Version   string `json:"version,omitempty"`
	State     string `json:"state,omitempty"`
	PID       int    `json:"pid,omitempty"`
	Live      bool   `json:"live"`
	LastError string `json:"last_error,omitempty"`
}

type FleetProfileChannelStatus struct {
	Channel        string                            `json:"channel"`
	Ready          bool                              `json:"ready"`
	CredentialID   string                            `json:"credential_id,omitempty"`
	CredentialHash string                            `json:"credential_hash,omitempty"`
	Evidence       []ProfileChannelReadinessEvidence `json:"evidence,omitempty"`
}

type FleetProfileTarget struct {
	ProfileID         string
	DisplayName       string
	Enabled           bool
	HomeRoot          string
	RuntimeStatusPath string
	DesiredChannels   []string
}

type FleetOperationEvidence struct {
	Status       FleetOperationStatus `json:"status"`
	RuntimeOwner string               `json:"runtime_owner,omitempty"`
	Message      string               `json:"message,omitempty"`
}

type FleetOperationReport struct {
	Action  FleetOperation         `json:"action"`
	Results []FleetOperationResult `json:"results"`
	Summary FleetOperationSummary  `json:"summary"`
}

type FleetOperationResult struct {
	ProfileID    string               `json:"profile_id"`
	Status       FleetOperationStatus `json:"status"`
	RuntimeOwner string               `json:"runtime_owner,omitempty"`
	Message      string               `json:"message,omitempty"`
}

type FleetOperationSummary struct {
	TargetedProfiles int `json:"targeted_profiles"`
	Succeeded        int `json:"succeeded"`
	Unavailable      int `json:"unavailable"`
	Failed           int `json:"failed"`
}

func NewFleetSupervisor(cfg config.Config, opts FleetSupervisorOptions) *FleetSupervisor {
	worker := opts.Worker
	if worker == nil {
		worker = RuntimeStatusFleetWorker{}
	}
	if strings.TrimSpace(opts.HomeRoot) == "" {
		opts.HomeRoot = config.GormesHome()
	}
	return &FleetSupervisor{cfg: cfg, opts: opts, worker: worker}
}

func (s *FleetSupervisor) Status(ctx context.Context) (FleetStatus, error) {
	if s == nil {
		return FleetStatus{}, nil
	}
	credentialHashes, unavailableCredentialHashes := s.resolveCredentialHashes()
	readiness := BuildProfileChannelReadinessWithOptions(s.cfg, ProfileChannelReadinessOptions{CredentialHashes: credentialHashes})
	applyFleetCredentialHashUnavailable(&readiness, unavailableCredentialHashes)
	readinessByProfile := fleetReadinessByProfile(readiness.Bindings)
	status := FleetStatus{}
	for _, target := range s.profileTargets() {
		if err := ctx.Err(); err != nil {
			return FleetStatus{}, err
		}
		profile := FleetProfileStatus{
			ProfileID:       target.ProfileID,
			DisplayName:     target.DisplayName,
			Enabled:         target.Enabled,
			ProfileHomeHash: fleetScopeHash(target.HomeRoot),
			Channels:        fleetChannelStatuses(target, readinessByProfile[target.ProfileID]),
		}
		if target.Enabled {
			runtimeStatus, err := s.worker.Status(ctx, target)
			if err != nil {
				runtimeStatus = FleetProfileRuntime{Owner: FleetRuntimeOwnerProfileServiceBridge, State: FleetRuntimeStateError, LastError: err.Error()}
			}
			profile.Runtime = normalizeFleetRuntime(runtimeStatus)
		}
		profile.Health = fleetProfileHealth(profile)
		status.Profiles = append(status.Profiles, profile)
	}
	status.Summary = summarizeFleetStatus(status.Profiles)
	return status, nil
}

func (s *FleetSupervisor) StartAll(ctx context.Context) (FleetOperationReport, error) {
	return s.runAll(ctx, FleetOperationStartAll, FleetOperationStatusStarted, func(ctx context.Context, target FleetProfileTarget) (FleetOperationEvidence, error) {
		return s.worker.Start(ctx, target)
	})
}

func (s *FleetSupervisor) StopAll(ctx context.Context) (FleetOperationReport, error) {
	return s.runAll(ctx, FleetOperationStopAll, FleetOperationStatusStopped, func(ctx context.Context, target FleetProfileTarget) (FleetOperationEvidence, error) {
		return s.worker.Stop(ctx, target)
	})
}

func (s *FleetSupervisor) RestartAll(ctx context.Context) (FleetOperationReport, error) {
	return s.runAll(ctx, FleetOperationRestartAll, FleetOperationStatusRestarted, func(ctx context.Context, target FleetProfileTarget) (FleetOperationEvidence, error) {
		return s.worker.Restart(ctx, target)
	})
}

func (s *FleetSupervisor) runAll(ctx context.Context, action FleetOperation, defaultStatus FleetOperationStatus, run func(context.Context, FleetProfileTarget) (FleetOperationEvidence, error)) (FleetOperationReport, error) {
	report := FleetOperationReport{Action: action}
	if s == nil {
		return report, nil
	}
	for _, target := range s.profileTargets() {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		if !target.Enabled {
			continue
		}
		evidence, err := run(ctx, target)
		if err != nil {
			evidence = FleetOperationEvidence{Status: FleetOperationStatusFailed, RuntimeOwner: FleetRuntimeOwnerProfileServiceBridge, Message: err.Error()}
		}
		if evidence.Status == "" {
			evidence.Status = defaultStatus
		}
		if evidence.RuntimeOwner == "" {
			evidence.RuntimeOwner = FleetRuntimeOwnerProfileServiceBridge
		}
		report.Results = append(report.Results, FleetOperationResult{
			ProfileID:    target.ProfileID,
			Status:       evidence.Status,
			RuntimeOwner: evidence.RuntimeOwner,
			Message:      evidence.Message,
		})
	}
	report.Summary = summarizeFleetOperation(report.Results)
	return report, nil
}

func (s *FleetSupervisor) resolveCredentialHashes() (map[string]string, map[string]struct{}) {
	if s == nil {
		return nil, nil
	}
	hashes := normalizedProfileChannelCredentialHashes(s.opts.CredentialHashes)
	unavailable := map[string]struct{}{}
	resolver := s.opts.CredentialResolver
	if resolver == nil {
		resolver = config.NewSecretResolver(config.SecretResolverConfig{Secrets: s.cfg.Secrets})
	}
	for _, service := range s.cfg.EnabledProfileServices() {
		for _, channelConfig := range sortedProfileChannelConfigBindings(service.Profile.Channels) {
			if !channelConfig.Config.Enabled {
				continue
			}
			credentialID := strings.TrimSpace(channelConfig.Config.Credential)
			if credentialID == "" || hashes[credentialID] != "" {
				continue
			}
			credential, ok := s.cfg.Credentials[credentialID]
			if !ok || credential.SecretRef == nil || strings.TrimSpace(credential.SecretRef.ID) == "" {
				continue
			}
			value, _, err := resolver.ResolveString(*credential.SecretRef)
			if err != nil || strings.TrimSpace(value) == "" {
				unavailable[credentialID] = struct{}{}
				continue
			}
			if hashes == nil {
				hashes = map[string]string{}
			}
			hashes[credentialID] = TokenCredentialHash(value)
		}
	}
	if len(unavailable) == 0 {
		unavailable = nil
	}
	return hashes, unavailable
}

func applyFleetCredentialHashUnavailable(report *ProfileChannelReadinessReport, unavailable map[string]struct{}) {
	if report == nil || len(unavailable) == 0 {
		return
	}
	for i := range report.Bindings {
		binding := &report.Bindings[i]
		if _, ok := unavailable[binding.CredentialID]; !ok {
			continue
		}
		binding.Evidence = append(binding.Evidence, newProfileChannelEvidence(ProfileChannelEvidenceCredentialHashUnavailable, binding.ProfileID, binding.Channel, binding.CredentialID, "credential_hash", "profile channel credential SecretRef could not be resolved for token ownership validation"))
		binding.Ready = false
	}
	report.Evidence = collectProfileChannelReadinessEvidence(report.Bindings)
}

func (s *FleetSupervisor) profileTargets() []FleetProfileTarget {
	if s == nil || len(s.cfg.Profiles) == 0 {
		return nil
	}
	ids := make([]string, 0, len(s.cfg.Profiles))
	for id := range s.cfg.Profiles {
		id = strings.TrimSpace(id)
		if id != "" {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	targets := make([]FleetProfileTarget, 0, len(ids))
	for _, id := range ids {
		profile := s.cfg.Profiles[id]
		home := fleetProfileHome(s.opts.HomeRoot, id)
		targets = append(targets, FleetProfileTarget{
			ProfileID:         id,
			DisplayName:       strings.TrimSpace(profile.Name),
			Enabled:           profile.Enabled,
			HomeRoot:          home,
			RuntimeStatusPath: filepath.Join(home, "runtime", "gateway_state.json"),
			DesiredChannels:   fleetDesiredChannels(profile.Channels),
		})
	}
	return targets
}

func fleetReadinessByProfile(bindings []ProfileChannelBindingReadiness) map[string][]ProfileChannelBindingReadiness {
	out := map[string][]ProfileChannelBindingReadiness{}
	for _, binding := range bindings {
		profileID := strings.TrimSpace(binding.ProfileID)
		if profileID == "" {
			continue
		}
		out[profileID] = append(out[profileID], binding)
	}
	for profileID := range out {
		sort.Slice(out[profileID], func(i, j int) bool { return out[profileID][i].Channel < out[profileID][j].Channel })
	}
	return out
}

func fleetChannelStatuses(target FleetProfileTarget, readiness []ProfileChannelBindingReadiness) []FleetProfileChannelStatus {
	byChannel := map[string]ProfileChannelBindingReadiness{}
	for _, binding := range readiness {
		byChannel[binding.Channel] = binding
	}
	channels := target.DesiredChannels
	out := make([]FleetProfileChannelStatus, 0, len(channels))
	for _, channel := range channels {
		status := FleetProfileChannelStatus{Channel: channel}
		if binding, ok := byChannel[channel]; ok {
			status.Ready = binding.Ready
			status.CredentialID = binding.CredentialID
			status.CredentialHash = binding.CredentialHash
			status.Evidence = append([]ProfileChannelReadinessEvidence(nil), binding.Evidence...)
		} else {
			status.Ready = target.Enabled
		}
		out = append(out, status)
	}
	return out
}

func fleetDesiredChannels(channels map[string]config.ProfileChannelCfg) []string {
	if len(channels) == 0 {
		return nil
	}
	out := make([]string, 0, len(channels))
	for channel, cfg := range channels {
		channel = strings.ToLower(strings.TrimSpace(channel))
		if channel != "" && cfg.Enabled {
			out = append(out, channel)
		}
	}
	sort.Strings(out)
	return out
}

func fleetProfileHealth(profile FleetProfileStatus) string {
	if !profile.Enabled {
		return FleetHealthDisabled
	}
	for _, channel := range profile.Channels {
		if len(channel.Evidence) > 0 || !channel.Ready {
			return FleetHealthDegraded
		}
	}
	if strings.TrimSpace(profile.Runtime.LastError) != "" || profile.Runtime.State == FleetRuntimeStateError {
		return FleetHealthDegraded
	}
	if profile.Runtime.State == FleetRuntimeStateRunning && profile.Runtime.Live {
		return FleetHealthHealthy
	}
	return FleetHealthStopped
}

func summarizeFleetStatus(profiles []FleetProfileStatus) FleetSummary {
	summary := FleetSummary{ConfiguredProfiles: len(profiles)}
	for _, profile := range profiles {
		if !profile.Enabled {
			summary.DisabledProfiles++
			continue
		}
		summary.EnabledProfiles++
		switch profile.Health {
		case FleetHealthHealthy:
			summary.HealthyProfiles++
		case FleetHealthDegraded:
			summary.DegradedProfiles++
			summary.ConflictProfiles++
		}
	}
	return summary
}

func summarizeFleetOperation(results []FleetOperationResult) FleetOperationSummary {
	summary := FleetOperationSummary{TargetedProfiles: len(results)}
	for _, result := range results {
		switch result.Status {
		case FleetOperationStatusFailed:
			summary.Failed++
		case FleetOperationStatusUnavailable:
			summary.Unavailable++
		default:
			summary.Succeeded++
		}
	}
	return summary
}

func normalizeFleetRuntime(runtime FleetProfileRuntime) FleetProfileRuntime {
	runtime.Owner = strings.TrimSpace(runtime.Owner)
	if runtime.Owner == "" {
		runtime.Owner = FleetRuntimeOwnerProfileServiceBridge
	}
	runtime.State = strings.TrimSpace(runtime.State)
	if runtime.State == "" {
		runtime.State = FleetRuntimeStateMissing
	}
	runtime.Version = strings.TrimSpace(runtime.Version)
	runtime.LastError = strings.TrimSpace(runtime.LastError)
	return runtime
}

func fleetProfileHome(homeRoot, profileID string) string {
	homeRoot = filepath.Clean(strings.TrimSpace(homeRoot))
	if homeRoot == "." || homeRoot == "" {
		homeRoot = filepath.Clean(config.GormesHome())
	}
	profileID = strings.TrimSpace(profileID)
	if profileID == "" || profileID == config.DefaultProfileID {
		return homeRoot
	}
	return filepath.Join(homeRoot, "profiles", profileID)
}

func fleetScopeHash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// RuntimeStatusFleetWorker is the compatibility worker for the current
// per-profile gateway-service bridge. It reads profile-local gateway_state.json
// files without opening session, memory, auth, provider, or channel clients.
type RuntimeStatusFleetWorker struct{}

func (RuntimeStatusFleetWorker) Status(ctx context.Context, target FleetProfileTarget) (FleetProfileRuntime, error) {
	path := strings.TrimSpace(target.RuntimeStatusPath)
	if path == "" {
		path = filepath.Join(target.HomeRoot, "gateway_state.json")
	}
	snapshot, err := NewRuntimeStatusStore(path).ReadValidatedRuntimeStatusSnapshot(ctx)
	if err != nil {
		return FleetProfileRuntime{}, err
	}
	if snapshot.Missing {
		return FleetProfileRuntime{Owner: FleetRuntimeOwnerProfileServiceBridge, State: FleetRuntimeStateMissing}, nil
	}
	state := string(snapshot.Status.GatewayState)
	if state == "" {
		state = FleetRuntimeStateStopped
	}
	lastError := strings.TrimSpace(snapshot.Status.ExitReason)
	if lastError == "" && snapshot.Validation.Status != "" && !snapshot.Validation.Live && snapshot.Validation.Status != RuntimeProcessValidationMissingState {
		lastError = snapshot.Validation.Message
	}
	return FleetProfileRuntime{
		Owner:     FleetRuntimeOwnerProfileServiceBridge,
		Version:   snapshot.Status.BootGitSHA,
		State:     state,
		PID:       snapshot.Status.PID,
		Live:      snapshot.Validation.Live,
		LastError: lastError,
	}, nil
}

func (RuntimeStatusFleetWorker) Start(context.Context, FleetProfileTarget) (FleetOperationEvidence, error) {
	return fleetOperationUnavailable("start-all")
}

func (RuntimeStatusFleetWorker) Stop(context.Context, FleetProfileTarget) (FleetOperationEvidence, error) {
	return fleetOperationUnavailable("stop-all")
}

func (RuntimeStatusFleetWorker) Restart(context.Context, FleetProfileTarget) (FleetOperationEvidence, error) {
	return fleetOperationUnavailable("restart-all")
}

func fleetOperationUnavailable(action string) (FleetOperationEvidence, error) {
	return FleetOperationEvidence{
		Status:       FleetOperationStatusUnavailable,
		RuntimeOwner: FleetRuntimeOwnerProfileServiceBridge,
		Message:      fmt.Sprintf("profile_fleet_%s_unavailable: current runtime is the per-profile gateway-service compatibility bridge", action),
	}, nil
}

type CommandFleetWorkerOptions struct {
	Command      string
	Env          []string
	Runner       FleetCommandRunner
	StatusWorker FleetProfileWorker
}

type CommandFleetWorker struct {
	opts CommandFleetWorkerOptions
}

type FleetCommand struct {
	Command string
	Args    []string
	Env     []string
}

type FleetCommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type FleetCommandRunner interface {
	Run(context.Context, FleetCommand) (FleetCommandResult, error)
}

type OSFleetCommandRunner struct{}

func NewCommandFleetWorker(opts CommandFleetWorkerOptions) CommandFleetWorker {
	return CommandFleetWorker{opts: opts}
}

func (w CommandFleetWorker) Status(ctx context.Context, target FleetProfileTarget) (FleetProfileRuntime, error) {
	worker := w.opts.StatusWorker
	if worker == nil {
		worker = RuntimeStatusFleetWorker{}
	}
	return worker.Status(ctx, target)
}

func (w CommandFleetWorker) Start(ctx context.Context, target FleetProfileTarget) (FleetOperationEvidence, error) {
	runtime, err := w.Status(ctx, target)
	if err != nil {
		return FleetOperationEvidence{Status: FleetOperationStatusFailed, RuntimeOwner: FleetRuntimeOwnerProfileCommandWorker, Message: "profile gateway start status check failed"}, nil
	}
	if runtime.Live {
		return FleetOperationEvidence{Status: FleetOperationStatusStarted, RuntimeOwner: FleetRuntimeOwnerProfileCommandWorker, Message: "profile gateway already running"}, nil
	}
	return w.runGatewayCommand(ctx, target, FleetOperationStartAll, FleetOperationStatusStarted, "restart", []string{"gateway", "restart", "--json", "--service", fleetGatewayServiceName(target)})
}

func (w CommandFleetWorker) Stop(ctx context.Context, target FleetProfileTarget) (FleetOperationEvidence, error) {
	return w.runGatewayCommand(ctx, target, FleetOperationStopAll, FleetOperationStatusStopped, "stop", []string{"gateway", "stop", "--json"})
}

func (w CommandFleetWorker) Restart(ctx context.Context, target FleetProfileTarget) (FleetOperationEvidence, error) {
	return w.runGatewayCommand(ctx, target, FleetOperationRestartAll, FleetOperationStatusRestarted, "restart", []string{"gateway", "restart", "--json", "--service", fleetGatewayServiceName(target)})
}

func (w CommandFleetWorker) runGatewayCommand(ctx context.Context, target FleetProfileTarget, action FleetOperation, success FleetOperationStatus, commandName string, args []string) (FleetOperationEvidence, error) {
	result, err := w.runner().Run(ctx, FleetCommand{Command: w.command(), Args: append([]string(nil), args...), Env: w.envForTarget(target)})
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return FleetOperationEvidence{}, ctxErr
		}
		return FleetOperationEvidence{Status: FleetOperationStatusFailed, RuntimeOwner: FleetRuntimeOwnerProfileCommandWorker, Message: fmt.Sprintf("profile gateway %s command failed", commandName)}, nil
	}
	evidence := fleetCommandOperationEvidence(action, success, commandName, result.Stdout)
	if evidence.RuntimeOwner == "" {
		evidence.RuntimeOwner = FleetRuntimeOwnerProfileCommandWorker
	}
	if evidence.Status == "" {
		evidence.Status = success
	}
	return evidence, nil
}

func (w CommandFleetWorker) command() string {
	if command := strings.TrimSpace(w.opts.Command); command != "" {
		return command
	}
	command, err := os.Executable()
	if err != nil || strings.TrimSpace(command) == "" {
		return "gormes"
	}
	return command
}

func (w CommandFleetWorker) runner() FleetCommandRunner {
	if w.opts.Runner != nil {
		return w.opts.Runner
	}
	return OSFleetCommandRunner{}
}

func (w CommandFleetWorker) envForTarget(target FleetProfileTarget) []string {
	env := append([]string(nil), w.opts.Env...)
	if len(env) == 0 {
		env = append(env, os.Environ()...)
	}
	return append(env, "GORMES_HOME="+target.HomeRoot)
}

func (OSFleetCommandRunner) Run(ctx context.Context, command FleetCommand) (FleetCommandResult, error) {
	run := exec.CommandContext(ctx, command.Command, command.Args...)
	if len(command.Env) > 0 {
		run.Env = command.Env
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	run.Stdout = &stdout
	run.Stderr = &stderr
	err := run.Run()
	result := FleetCommandResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
	}
	return result, err
}

type fleetCommandJSONReport struct {
	Action      string `json:"action"`
	Mode        string `json:"mode"`
	Manager     string `json:"manager"`
	Service     string `json:"service"`
	Outcome     string `json:"outcome"`
	FinalStatus string `json:"final_status"`
	Live        bool   `json:"live"`
}

func fleetCommandOperationEvidence(action FleetOperation, success FleetOperationStatus, commandName, stdout string) FleetOperationEvidence {
	report := fleetCommandJSONReport{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &report); err != nil {
		return FleetOperationEvidence{Status: success, RuntimeOwner: FleetRuntimeOwnerProfileCommandWorker, Message: fmt.Sprintf("gateway %s command completed without parseable JSON", commandName)}
	}
	parts := []string{"gateway " + commandName}
	if report.Action != "" {
		parts = append(parts, "action="+report.Action)
	}
	if report.Mode != "" {
		parts = append(parts, "mode="+report.Mode)
	}
	if report.Manager != "" {
		parts = append(parts, "manager="+report.Manager)
	}
	if report.Service != "" {
		parts = append(parts, "service="+report.Service)
	}
	if report.Outcome != "" {
		parts = append(parts, "outcome="+report.Outcome)
	}
	if report.FinalStatus != "" {
		parts = append(parts, "final_status="+report.FinalStatus)
	}
	status := success
	if action == FleetOperationStopAll {
		status = FleetOperationStatusStopped
	}
	return FleetOperationEvidence{Status: status, RuntimeOwner: FleetRuntimeOwnerProfileCommandWorker, Message: strings.Join(parts, " ")}
}

func fleetGatewayServiceName(target FleetProfileTarget) string {
	profileID := strings.TrimSpace(target.ProfileID)
	if profileID == "" || profileID == config.DefaultProfileID {
		return "gormes-gateway.service"
	}
	return "gormes-gateway-" + profileID + ".service"
}
