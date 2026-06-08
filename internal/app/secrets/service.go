package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config/externalsecrets"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// BuildProvenance is the shared build metadata embedded in secrets JSON output.
type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

// Options supplies CLI-specific seams while keeping secrets behavior in this package.
type Options struct {
	BuildProvenance          func() BuildProvenance
	BitwardenInstallRelease  string
	BitwardenInstallDownload func(context.Context, string) ([]byte, error)
	BitwardenSetupInput      io.Reader
	IsTerminal               func() bool
}

func (o Options) buildProvenance() BuildProvenance {
	if o.BuildProvenance != nil {
		return o.BuildProvenance()
	}
	return BuildProvenance{}
}

func Apply(ctx context.Context, out io.Writer, planPath string, jsonOut bool, options Options) error {
	planFile, err := LoadPlanFile(planPath)
	if err != nil {
		return err
	}
	controller, err := NewRuntimeController(planFile)
	if err != nil {
		return err
	}
	result, err := controller.Apply(ctx, planFile.Plan())
	if err == nil {
		err = WriteSnapshotFile(result.Snapshot)
	}
	writeApplyResult(out, result, jsonOut, options)
	return err
}

func Audit(ctx context.Context, out io.Writer, planPath string, jsonOut bool, options Options) error {
	if strings.TrimSpace(planPath) == "" {
		const msg = `secrets audit: required flag "--plan <file>" not set`
		if jsonOut {
			return emitJSONInputError(out, "missing_flag", msg, options)
		}
		return fmt.Errorf("%s", msg)
	}
	planFile, err := LoadPlanFile(planPath)
	if err != nil {
		return err
	}
	previous, err := ReadSnapshotFile()
	if err != nil {
		return err
	}
	result := toolspkg.AuditSecrets(ctx, toolspkg.SecretsAuditRequest{
		Resolver:         NewConfigSecretResolver(planFile.Secrets),
		Plan:             planFile.Plan(),
		PreviousSnapshot: &previous,
	})
	writeAuditResult(out, result, jsonOut, options)
	if !result.OK {
		return newExitCodeError(1, errors.New("secrets audit found findings"))
	}
	return nil
}

func Configure(ctx context.Context, out io.Writer, path, source, provider, id string, optional, jsonOut bool, options Options) error {
	if provider == "" {
		provider = config.DefaultSecretProviderAlias
	}
	result, err := toolspkg.ConfigureSecretRef(ctx, toolspkg.SecretsConfigureRequest{
		Resolver: NewConfigSecretResolver(config.SecretsCfg{}),
		Path:     path,
		Required: !optional,
		Ref: toolspkg.SecretRef{
			Source:   source,
			Provider: provider,
			ID:       id,
		},
	})
	writeConfigureResult(out, result, jsonOut, options)
	return err
}

func Reload(ctx context.Context, out io.Writer, planPath string, jsonOut bool, options Options) error {
	planFile, err := LoadPlanFile(planPath)
	if err != nil {
		return err
	}
	controller, err := NewRuntimeController(planFile)
	if err != nil {
		return err
	}
	result, err := controller.Reload(ctx, planFile.Plan())
	if err == nil {
		err = WriteSnapshotFile(result.Snapshot)
	}
	writeApplyResult(out, result, jsonOut, options)
	return err
}

func BitwardenStatus(_ context.Context, out io.Writer) error {
	cfg, err := loadBitwardenConfig()
	if err != nil {
		return err
	}
	tokenEnv := bitwardenTokenEnv(cfg)
	_, tokenSet := os.LookupEnv(tokenEnv)
	statusCfg := bitwardenExternalConfig(cfg)
	statusCfg.AutoInstall = false
	binary, binErr := externalsecrets.FindBitwardenBinary(statusCfg, externalsecrets.BitwardenOptions{HomeDir: config.GormesHome()})
	fmt.Fprintf(out, "Bitwarden Secrets Manager\n")
	fmt.Fprintf(out, "Enabled: %s\n", yesNo(cfg.Enabled))
	fmt.Fprintf(out, "Token env var: %s\n", tokenEnv)
	fmt.Fprintf(out, "Token in env: %s\n", yesNo(tokenSet))
	fmt.Fprintf(out, "Project ID: %s\n", valueOrUnset(cfg.ProjectID))
	fmt.Fprintf(out, "Server URL: %s\n", serverURLDisplay(cfg.ServerURL))
	fmt.Fprintf(out, "Override existing: %s\n", yesNo(cfg.OverrideExisting))
	fmt.Fprintf(out, "Cache TTL (s): %d\n", cfg.CacheTTLSeconds)
	fmt.Fprintf(out, "Auto-install: %s\n", yesNo(cfg.AutoInstall))
	if binErr != nil {
		fmt.Fprintf(out, "bws binary: not installed (%s)\n", binErr.Error())
	} else {
		fmt.Fprintf(out, "bws binary: %s\n", binary)
	}
	if !cfg.Enabled {
		fmt.Fprintln(out, "Run `gormes secrets bitwarden setup` to enable.")
	} else if !tokenSet {
		fmt.Fprintf(out, "%s is not set; Gormes will skip Bitwarden on startup.\n", tokenEnv)
	} else if strings.TrimSpace(cfg.ProjectID) == "" {
		fmt.Fprintln(out, "project_id is empty; nothing to fetch.")
	}
	return nil
}

func BitwardenSync(ctx context.Context, out io.Writer, apply bool) error {
	cfg, err := loadBitwardenConfig()
	if err != nil {
		return err
	}
	if !cfg.Enabled {
		return errors.New("Bitwarden integration is disabled; run `gormes secrets bitwarden setup` first")
	}
	if strings.TrimSpace(cfg.ProjectID) == "" {
		return errors.New("No project_id configured")
	}
	if _, ok := os.LookupEnv(bitwardenTokenEnv(cfg)); !ok {
		return fmt.Errorf("%s is not set", bitwardenTokenEnv(cfg))
	}
	external := bitwardenExternalConfig(cfg)
	if !apply {
		external.OverrideExisting = false
	} else {
		external.OverrideExisting = true
	}
	wouldSet := map[string]string{}
	report := externalsecrets.ApplyBitwarden(ctx, external, externalsecrets.BitwardenOptions{
		HomeDir: config.GormesHome(),
		DryRun:  !apply,
		SetEnv: func(key, value string) error {
			if apply {
				return os.Setenv(key, value)
			}
			wouldSet[key] = value
			return nil
		},
	})
	if !report.OK() {
		return errors.New(report.Error)
	}
	fmt.Fprintln(out, "Bitwarden Secrets Manager sync")
	if !apply {
		fmt.Fprintln(out, "Mode: dry-run")
	} else {
		fmt.Fprintln(out, "Mode: apply")
	}
	for _, key := range sortedUnique(append(append([]string{}, report.Applied...), report.Skipped...)) {
		action := "would export"
		if apply {
			action = "exported"
		}
		if stringSliceContainsLocal(report.Skipped, key) {
			if key == bitwardenTokenEnv(cfg) {
				action = "skip (bootstrap token)"
			} else {
				action = "skip (already set)"
			}
		} else if !apply && wouldSet[key] == "" {
			action = "would export"
		}
		fmt.Fprintf(out, "%s: %s\n", key, action)
	}
	for _, warning := range report.Warnings {
		fmt.Fprintf(out, "warning: %s\n", warning)
	}
	if !apply {
		fmt.Fprintln(out, "This was a dry-run; re-run with --apply to export into the current process.")
	}
	return nil
}

func BitwardenDisable(_ context.Context, out io.Writer) error {
	if err := config.WriteTOMLValue(config.ConfigPath(), "secrets.bitwarden.enabled", "false"); err != nil {
		return err
	}
	fmt.Fprintln(out, "Disabled. Bitwarden secrets will NOT be pulled on the next Gormes invocation.")
	fmt.Fprintln(out, "Your bootstrap token is left in .env; remove it manually if you also want to revoke the credential.")
	return nil
}

func BitwardenInstall(ctx context.Context, out io.Writer, force bool, options Options) error {
	path, err := externalsecrets.InstallBitwardenBWS(ctx, externalsecrets.BitwardenInstallOptions{
		HomeDir:     config.GormesHome(),
		Force:       force,
		ReleaseBase: options.BitwardenInstallRelease,
		Download:    options.BitwardenInstallDownload,
	})
	if err != nil {
		return fmt.Errorf("Bitwarden install failed: %w", err)
	}
	fmt.Fprintf(out, "Installed bws %s at %s\n", externalsecrets.BitwardenBWSVersion, path)
	return nil
}

func BitwardenSetup(ctx context.Context, out io.Writer, accessToken, serverURL, projectID string, options Options) error {
	cfg, err := loadBitwardenConfig()
	if err != nil {
		return err
	}
	tokenEnv := bitwardenTokenEnv(cfg)
	accessToken = strings.TrimSpace(accessToken)
	serverURL = strings.TrimSpace(serverURL)
	projectID = strings.TrimSpace(projectID)

	if !bitwardenSetupIsTerminal(options) {
		var missing []string
		if accessToken == "" {
			missing = append(missing, "--access-token")
		}
		if serverURL == "" && strings.TrimSpace(os.Getenv("BWS_SERVER_URL")) == "" {
			missing = append(missing, "--server-url")
		}
		if projectID == "" {
			missing = append(missing, "--project-id")
		}
		if len(missing) > 0 {
			msg := "Non-interactive mode requires all setup flags. Missing: " + strings.Join(missing, ", ")
			fmt.Fprintf(out, "%s\nUsage: gormes secrets bitwarden setup --access-token '0.xxx' --server-url 'https://vault.bitwarden.com' --project-id 'xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx'\n", msg)
			return newExitCodeError(1, errors.New(msg))
		}
	}

	binary, err := externalsecrets.FindBitwardenBinary(cfg, externalsecrets.BitwardenOptions{HomeDir: config.GormesHome()})
	if err != nil {
		binary, err = externalsecrets.InstallBitwardenBWS(ctx, externalsecrets.BitwardenInstallOptions{HomeDir: config.GormesHome(), ReleaseBase: options.BitwardenInstallRelease, Download: options.BitwardenInstallDownload})
		if err != nil {
			fmt.Fprintf(out, "Could not install bws: %v\nManual install: https://github.com/bitwarden/sdk-sm/releases\n", err)
			return newExitCodeError(1, fmt.Errorf("bitwarden setup: install bws: %w", err))
		}
	}
	if accessToken == "" {
		accessToken, err = promptLine(options, fmt.Sprintf("Paste access token (%s): ", tokenEnv))
		if err != nil {
			return err
		}
		accessToken = strings.TrimSpace(accessToken)
	}
	if accessToken == "" {
		fmt.Fprintln(out, "Empty token, aborting.")
		return newExitCodeError(1, errors.New("empty Bitwarden access token"))
	}
	if err := config.WriteEnvValue(config.EnvPath(), tokenEnv, accessToken); err != nil {
		return fmt.Errorf("bitwarden setup: write dotenv: %w", err)
	}
	_ = os.Setenv(tokenEnv, accessToken)

	if serverURL == "" {
		serverURL = strings.TrimSpace(os.Getenv("BWS_SERVER_URL"))
	}
	if serverURL == "" {
		serverURL, err = chooseBitwardenServerURL(options, cfg.ServerURL)
		if err != nil {
			return err
		}
	}
	if projectID == "" {
		projects, err := listBitwardenProjects(ctx, binary, tokenEnv, accessToken, serverURL)
		if err != nil {
			fmt.Fprintf(out, "Project list failed: %v\n", err)
			return newExitCodeError(1, err)
		}
		if len(projects) == 0 {
			fmt.Fprintln(out, "No projects visible to this machine account.")
			return newExitCodeError(1, errors.New("no Bitwarden projects visible"))
		}
		for i, p := range projects {
			fmt.Fprintf(out, "%d. %s (%s)\n", i+1, valueOrUnset(p.Name), p.ID)
		}
		for {
			choice, err := promptLine(options, fmt.Sprintf("Select project [1-%d]: ", len(projects)))
			if err != nil {
				return err
			}
			idx, err := strconv.Atoi(strings.TrimSpace(choice))
			if err == nil && idx >= 1 && idx <= len(projects) {
				projectID = projects[idx-1].ID
				break
			}
			fmt.Fprintf(out, "Out of range — pick 1-%d.\n", len(projects))
		}
	}

	report := externalsecrets.ApplyBitwarden(ctx, externalsecrets.BitwardenConfig{Enabled: true, AccessTokenEnv: tokenEnv, ProjectID: projectID, OverrideExisting: true, AutoInstall: false, ServerURL: serverURL}, externalsecrets.BitwardenOptions{HomeDir: config.GormesHome(), DryRun: true})
	if !report.OK() {
		fmt.Fprintf(out, "Fetch failed: %s\n", report.Error)
		return newExitCodeError(1, errors.New(report.Error))
	}
	for _, key := range sortedUnique(append(append([]string{}, report.Applied...), report.Skipped...)) {
		action := "new"
		if key == tokenEnv {
			action = "bootstrap token — never overrides itself"
		} else if stringSliceContainsLocal(report.Skipped, key) {
			action = "already set in env"
		}
		fmt.Fprintf(out, "%s: %s\n", key, action)
	}
	if err := persistBitwardenConfig(tokenEnv, projectID, serverURL); err != nil {
		return err
	}
	fmt.Fprintf(out, "Bitwarden Secrets Manager is enabled.\nStatus: gormes secrets bitwarden status\nRefresh: gormes secrets bitwarden sync\nDisable: gormes secrets bitwarden disable\n")
	return nil
}

type bitwardenProject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func bitwardenSetupIsTerminal(options Options) bool {
	if options.IsTerminal != nil {
		return options.IsTerminal()
	}
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func promptLine(options Options, prompt string) (string, error) {
	input := options.BitwardenSetupInput
	if input == nil {
		input = os.Stdin
	}
	var value string
	if _, err := fmt.Fscan(input, &value); err != nil {
		return "", err
	}
	return value, nil
}

func chooseBitwardenServerURL(options Options, existing string) (string, error) {
	choice, err := promptLine(options, "")
	if err != nil {
		return "", err
	}
	switch strings.ToLower(strings.TrimSpace(choice)) {
	case "", "keep":
		return strings.TrimSpace(existing), nil
	case "1", "us", "default":
		return "https://vault.bitwarden.com", nil
	case "2", "eu":
		return "https://vault.bitwarden.eu", nil
	case "3", "custom":
		custom, err := promptLine(options, "")
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(custom), nil
	default:
		return strings.TrimSpace(choice), nil
	}
}

func listBitwardenProjects(ctx context.Context, binary, tokenEnv, token, serverURL string) ([]bitwardenProject, error) {
	cmd := exec.CommandContext(ctx, binary, "project", "list", "--output", "json")
	env := os.Environ()
	env = append(env, tokenEnv+"="+token, "NO_COLOR=1")
	if strings.TrimSpace(serverURL) != "" {
		env = append(env, "BWS_SERVER_URL="+strings.TrimSpace(serverURL))
	}
	cmd.Env = env
	stdout, err := cmd.Output()
	if err != nil {
		message := err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok && strings.TrimSpace(string(exitErr.Stderr)) != "" {
			message = strings.TrimSpace(string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bws project list failed: %s", truncateString(message, 200))
	}
	var projects []bitwardenProject
	if err := json.Unmarshal(stdout, &projects); err != nil {
		return nil, fmt.Errorf("bws project list returned non-JSON output")
	}
	return projects, nil
}

func truncateString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func persistBitwardenConfig(tokenEnv, projectID, serverURL string) error {
	path := config.ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("bitwarden setup: mkdir config dir: %w", err)
	}
	body := fmt.Sprintf("[secrets.bitwarden]\nenabled = true\nproject_id = %q\nserver_url = %q\naccess_token_env = %q\ncache_ttl_seconds = 300\noverride_existing = true\nauto_install = true\n", projectID, serverURL, tokenEnv)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		return fmt.Errorf("bitwarden setup: write config: %w", err)
	}
	return nil
}

// PlanFile is the JSON plan consumed by secrets commands.
type PlanFile struct {
	Targets []toolspkg.SecretTarget `json:"targets"`
	Secrets config.SecretsCfg       `json:"secrets,omitempty"`
}

func (p PlanFile) Plan() toolspkg.SecretsPlan {
	return toolspkg.SecretsPlan{Targets: p.Targets}
}

func LoadPlanFile(path string) (PlanFile, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return PlanFile{}, errors.New("gormes secrets: --plan is required")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return PlanFile{}, fmt.Errorf("gormes secrets: read plan: %w", err)
	}
	var plan PlanFile
	if err := json.Unmarshal(body, &plan); err != nil {
		return PlanFile{}, fmt.Errorf("gormes secrets: parse plan: %w", err)
	}
	return plan, nil
}

func NewRuntimeController(plan PlanFile) (*toolspkg.SecretsRuntimeController, error) {
	snapshot, err := ReadSnapshotFile()
	if err != nil {
		return nil, err
	}
	return toolspkg.NewSecretsRuntimeController(toolspkg.SecretsRuntimeControllerConfig{
		Resolver:        NewConfigSecretResolver(plan.Secrets),
		InitialSnapshot: &snapshot,
	}), nil
}

type ConfigSecretResolver struct {
	resolver *config.SecretResolver
}

func NewConfigSecretResolver(secrets config.SecretsCfg) ConfigSecretResolver {
	return ConfigSecretResolver{resolver: config.NewSecretResolver(config.SecretResolverConfig{Secrets: secrets})}
}

func (r ConfigSecretResolver) ResolveSecretString(ref toolspkg.SecretRef) (string, toolspkg.SecretRefEvidence, error) {
	value, evidence, err := r.resolver.ResolveString(config.SecretRef{
		Source:   config.SecretRefSource(ref.Source),
		Provider: ref.Provider,
		ID:       ref.ID,
	})
	return value, toolspkg.SecretRefEvidence{
		Code:     evidence.Code,
		Source:   evidence.Source,
		Provider: evidence.Provider,
		ID:       evidence.ID,
		Redacted: evidence.Redacted,
	}, err
}

func ReadSnapshotFile() (toolspkg.SecretsRuntimeSnapshot, error) {
	path := SnapshotPath()
	body, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return toolspkg.SecretsRuntimeSnapshot{Entries: map[string]toolspkg.SecretsRuntimeEntry{}, Redacted: true}, nil
	}
	if err != nil {
		return toolspkg.SecretsRuntimeSnapshot{}, fmt.Errorf("gormes secrets: read snapshot: %w", err)
	}
	var snapshot toolspkg.SecretsRuntimeSnapshot
	if err := json.Unmarshal(body, &snapshot); err != nil {
		return toolspkg.SecretsRuntimeSnapshot{}, fmt.Errorf("gormes secrets: parse snapshot: %w", err)
	}
	if snapshot.Entries == nil {
		snapshot.Entries = map[string]toolspkg.SecretsRuntimeEntry{}
	}
	snapshot.Redacted = true
	return snapshot, nil
}

func WriteSnapshotFile(snapshot toolspkg.SecretsRuntimeSnapshot) error {
	path := SnapshotPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("gormes secrets: mkdir snapshot dir: %w", err)
	}
	snapshot.Redacted = true
	body, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("gormes secrets: marshal snapshot: %w", err)
	}
	body = append(body, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".secrets-runtime.*")
	if err != nil {
		return fmt.Errorf("gormes secrets: create snapshot temp: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("gormes secrets: write snapshot temp: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("gormes secrets: chmod snapshot temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("gormes secrets: close snapshot temp: %w", err)
	}
	if _, err := toolspkg.AtomicReplace(tmpName, path, toolspkg.AtomicReplaceOptions{FirstWriteMode: 0o600}); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("gormes secrets: replace snapshot: %w", err)
	}
	return nil
}

func SnapshotPath() string {
	return filepath.Join(config.GormesHome(), "secrets-runtime.json")
}

func loadBitwardenConfig() (externalsecrets.BitwardenConfig, error) {
	cfg := externalsecrets.BitwardenConfig{
		AccessTokenEnv:   externalsecrets.DefaultBitwardenAccessTokenEnv,
		CacheTTLSeconds:  externalsecrets.DefaultBitwardenCacheTTLSeconds,
		OverrideExisting: true,
		AutoInstall:      true,
	}
	body, err := os.ReadFile(config.ConfigPath())
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("gormes secrets bitwarden: read config: %w", err)
	}
	type rawBitwardenConfig struct {
		Enabled          *bool  `toml:"enabled"`
		AccessTokenEnv   string `toml:"access_token_env"`
		ProjectID        string `toml:"project_id"`
		CacheTTLSeconds  *int   `toml:"cache_ttl_seconds"`
		OverrideExisting *bool  `toml:"override_existing"`
		AutoInstall      *bool  `toml:"auto_install"`
		ServerURL        string `toml:"server_url"`
	}
	var doc struct {
		Secrets struct {
			Bitwarden rawBitwardenConfig `toml:"bitwarden"`
		} `toml:"secrets"`
	}
	if err := toml.Unmarshal(body, &doc); err != nil {
		return cfg, fmt.Errorf("gormes secrets bitwarden: parse config: %w", err)
	}
	loaded := doc.Secrets.Bitwarden
	if loaded.Enabled != nil {
		cfg.Enabled = *loaded.Enabled
	}
	if strings.TrimSpace(loaded.AccessTokenEnv) != "" {
		cfg.AccessTokenEnv = loaded.AccessTokenEnv
	}
	cfg.ProjectID = loaded.ProjectID
	if loaded.CacheTTLSeconds != nil {
		cfg.CacheTTLSeconds = *loaded.CacheTTLSeconds
	}
	if loaded.OverrideExisting != nil {
		cfg.OverrideExisting = *loaded.OverrideExisting
	}
	if loaded.AutoInstall != nil {
		cfg.AutoInstall = *loaded.AutoInstall
	}
	cfg.ServerURL = loaded.ServerURL
	return cfg, nil
}

func bitwardenExternalConfig(cfg externalsecrets.BitwardenConfig) externalsecrets.BitwardenConfig {
	cfg.AccessTokenEnv = bitwardenTokenEnv(cfg)
	return cfg
}

func bitwardenTokenEnv(cfg externalsecrets.BitwardenConfig) string {
	if strings.TrimSpace(cfg.AccessTokenEnv) != "" {
		return strings.TrimSpace(cfg.AccessTokenEnv)
	}
	return externalsecrets.DefaultBitwardenAccessTokenEnv
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func valueOrUnset(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "(unset)"
	}
	return value
}

func serverURLDisplay(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "default (US Cloud, https://vault.bitwarden.com)"
	}
	return value
}

func sortedUnique(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func stringSliceContainsLocal(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func writeApplyResult(out io.Writer, result toolspkg.SecretsApplyResult, jsonOut bool, options Options) {
	if jsonOut {
		writeJSON(out, result, options)
		return
	}
	fmt.Fprintf(out, "%s generation=%d entries=%d redacted=%t\n", result.Code, result.Snapshot.Generation, len(result.Snapshot.Entries), result.Redacted)
	for _, finding := range result.Findings {
		fmt.Fprintf(out, "%s path=%s severity=%s evidence=%s redacted=%t\n", finding.Code, finding.Path, finding.Severity, finding.Evidence.Code, finding.Redacted)
	}
}

func writeAuditResult(out io.Writer, result toolspkg.SecretsAuditResult, jsonOut bool, options Options) {
	if jsonOut {
		writeJSON(out, result, options)
		return
	}
	fmt.Fprintf(out, "%s ok=%t findings=%d redacted=%t\n", result.Code, result.OK, len(result.Findings), result.Redacted)
	for _, finding := range result.Findings {
		fmt.Fprintf(out, "%s path=%s severity=%s evidence=%s redacted=%t\n", finding.Code, finding.Path, finding.Severity, finding.Evidence.Code, finding.Redacted)
	}
}

func writeConfigureResult(out io.Writer, result toolspkg.SecretsConfigureResult, jsonOut bool, options Options) {
	if jsonOut {
		writeJSON(out, result, options)
		return
	}
	fmt.Fprintf(out, "%s path=%s source=%s provider=%s id=%s preflight_ok=%t redacted=%t\n", result.Code, result.Target.Path, result.Target.Ref.Source, result.Target.Ref.Provider, result.Target.Ref.ID, result.PreflightOK, result.Redacted)
}

func writeJSON(out io.Writer, value any, options Options) {
	bodyBytes, err := json.Marshal(value)
	if err != nil {
		return
	}
	merged := map[string]json.RawMessage{}
	if jerr := json.Unmarshal(bodyBytes, &merged); jerr != nil {
		return
	}
	buildBytes, err := json.Marshal(options.buildProvenance())
	if err != nil {
		return
	}
	merged["build"] = buildBytes
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	_ = enc.Encode(merged)
}

type jsonInputErrorReport struct {
	Build  BuildProvenance `json:"build"`
	Action string          `json:"action"`
	Error  string          `json:"error"`
}

func emitJSONInputError(out io.Writer, action, errMsg string, options Options) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(jsonInputErrorReport{
		Build:  options.buildProvenance(),
		Action: action,
		Error:  errMsg,
	})
	return newExitCodeError(1, fmt.Errorf("%s", errMsg))
}

type exitCodeError struct {
	code int
	err  error
}

func newExitCodeError(code int, err error) error {
	if err == nil {
		err = fmt.Errorf("exit code %d", code)
	}
	return exitCodeError{code: code, err: err}
}

func (e exitCodeError) Error() string { return e.err.Error() }
func (e exitCodeError) Unwrap() error { return e.err }
func (e exitCodeError) ExitCode() int { return e.code }
