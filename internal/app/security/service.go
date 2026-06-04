package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type AuditOptions struct {
	Deep bool
	Fix  bool
	JSON bool
}

type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

func RunAudit(ctx context.Context, out io.Writer, opts AuditOptions, build BuildProvenance) (toolspkg.SecurityAuditResult, error) {
	cfg, err := config.Load(nil)
	if err != nil {
		return toolspkg.SecurityAuditResult{}, fmt.Errorf("config: %w", err)
	}
	req := BuildAuditRequest(ctx, cfg, opts.Deep, opts.Fix)
	result := toolspkg.AuditSecurity(req)
	WriteAuditResult(out, result, opts.JSON, build)
	return result, nil
}

func BuildAuditRequest(ctx context.Context, cfg config.Config, deep, fix bool) toolspkg.SecurityAuditRequest {
	secrets := cliSecurityAuditSecrets(cfg)
	secretRefs := cliSecurityAuditSecretRefs(cfg, secrets)
	secrets = appendUniqueSecurityAuditSecrets(secrets, secretRefs.Secrets...)
	cwd, err := os.Getwd()
	if err != nil || strings.TrimSpace(cwd) == "" {
		cwd = config.GormesHome()
	}
	secretRefAvailability := cliSecurityAuditSecretRefAvailability(secretRefs.Refs)
	return toolspkg.SecurityAuditRequest{
		Deep: deep,
		Fix:  fix,
		GatewayAuth: toolspkg.SecurityAuditGatewayAuth{
			TokenConfigured:       strings.TrimSpace(cfg.Gateway.ProxyKey) != "",
			Exposure:              cliSecurityAuditGatewayExposure(cfg),
			Probe:                 cliSecurityAuditGatewayProbe(ctx, deep, secrets),
			GenerateTokenWhenFix:  true,
			GeneratedTokenPath:    "GATEWAY_PROXY_KEY",
			GeneratedTokenMessage: "generated gateway bearer token in dotenv",
		},
		State: toolspkg.SecurityAuditState{
			Files: cliSecurityAuditStateFiles(),
		},
		Channels:            cliSecurityAuditChannels(cfg, secretRefAvailability),
		Filesystem:          cliSecurityAuditFilesystem(cwd),
		CredentialRedaction: cliSecurityAuditCredentialRedaction(secrets),
		SecretRefs:          secretRefs.Refs,
		FixCandidates:       cliSecurityAuditFixCandidates(),
		FixApplier:          cliSecurityAuditFixApplier,
		TokenGenerator:      cliSecurityAuditGenerateGatewayToken,
	}
}

func cliSecurityAuditGatewayProbe(ctx context.Context, deep bool, secrets []string) toolspkg.SecurityAuditProbe {
	if !deep {
		return toolspkg.SecurityAuditProbe{}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	snapshot, err := gateway.NewRuntimeStatusStore(config.GatewayRuntimeStatusPath()).ReadValidatedRuntimeStatusSnapshot(ctx)
	if err != nil {
		return toolspkg.SecurityAuditProbe{
			Required:  true,
			Available: false,
			Message:   toolspkg.RedactSecurityAuditText(err.Error(), secrets),
		}
	}
	if snapshot.Missing {
		return toolspkg.SecurityAuditProbe{
			Required:  true,
			Available: false,
			Message:   "runtime status is missing",
		}
	}
	return toolspkg.SecurityAuditProbe{
		Required:  true,
		Available: snapshot.Validation.Live,
		Message:   toolspkg.RedactSecurityAuditText(snapshot.Validation.Message, secrets),
	}
}

func cliSecurityAuditGatewayExposure(cfg config.Config) string {
	if strings.TrimSpace(cfg.Gateway.ProxyURL) != "" {
		return "remote"
	}
	return "local"
}

func cliSecurityAuditStateFiles() []toolspkg.SecurityAuditStateFile {
	paths := []string{
		config.ConfigPath(),
		config.EnvPath(),
		config.GatewayRuntimeStatusPath(),
		config.SessionDBPath(),
		config.MemoryDBPath(),
		filepath.Join(config.GormesHome(), "secrets-runtime.json"),
	}
	files := make([]toolspkg.SecurityAuditStateFile, 0, len(paths))
	seen := map[string]struct{}{}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		files = append(files, cliSecurityAuditStateFile(path))
	}
	return files
}

func cliSecurityAuditStateFile(path string) toolspkg.SecurityAuditStateFile {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return toolspkg.SecurityAuditStateFile{Path: path, Exists: false, Valid: false}
	}
	if err != nil {
		return toolspkg.SecurityAuditStateFile{Path: path, Exists: true, Valid: false, Error: err.Error()}
	}
	if info.IsDir() {
		return toolspkg.SecurityAuditStateFile{Path: path, Exists: true, Valid: false, Error: "state path is a directory"}
	}
	if file, err := os.Open(path); err != nil {
		return toolspkg.SecurityAuditStateFile{Path: path, Exists: true, Valid: false, Error: err.Error()}
	} else {
		_ = file.Close()
	}
	return toolspkg.SecurityAuditStateFile{Path: path, Exists: true, Valid: true}
}

func cliSecurityAuditChannels(cfg config.Config, secretRefs map[string]bool) []toolspkg.SecurityAuditChannel {
	return []toolspkg.SecurityAuditChannel{
		{
			Name:                   "telegram",
			Enabled:                strings.TrimSpace(cfg.Telegram.BotToken) != "" || cfg.Telegram.AllowedChatID != 0 || len(cfg.Telegram.AllowedUserIDs) > 0 || cfg.Telegram.FirstRunDiscovery,
			TokenConfigured:        strings.TrimSpace(cfg.Telegram.BotToken) != "" || secretRefs["telegram.bot_token"],
			AllowedScopeConfigured: cfg.Telegram.AllowedChatID != 0 || len(cfg.Telegram.AllowedUserIDs) > 0,
			FirstRunDiscovery:      cfg.Telegram.FirstRunDiscovery,
		},
		{
			Name:                   "discord",
			Enabled:                strings.TrimSpace(cfg.Discord.Token) != "" || strings.TrimSpace(cfg.Discord.AllowedChannelID) != "" || cfg.Discord.FirstRunDiscovery,
			TokenConfigured:        strings.TrimSpace(cfg.Discord.Token) != "" || secretRefs["discord.token"],
			AllowedScopeConfigured: strings.TrimSpace(cfg.Discord.AllowedChannelID) != "",
			FirstRunDiscovery:      cfg.Discord.FirstRunDiscovery,
		},
		{
			Name:                   "slack",
			Enabled:                cfg.Slack.Enabled || strings.TrimSpace(cfg.Slack.BotToken) != "" || strings.TrimSpace(cfg.Slack.AppToken) != "" || strings.TrimSpace(cfg.Slack.AllowedChannelID) != "" || cfg.Slack.FirstRunDiscovery,
			TokenConfigured:        (strings.TrimSpace(cfg.Slack.BotToken) != "" || secretRefs["slack.bot_token"]) && (strings.TrimSpace(cfg.Slack.AppToken) != "" || secretRefs["slack.app_token"]),
			AllowedScopeConfigured: strings.TrimSpace(cfg.Slack.AllowedChannelID) != "",
			FirstRunDiscovery:      cfg.Slack.FirstRunDiscovery,
		},
	}
}

func cliSecurityAuditFilesystem(cwd string) toolspkg.SecurityAuditFilesystem {
	parent := filepath.Dir(cwd)
	outsideWrite := filepath.Join(parent, ".gormes-security-audit-write-probe")
	if parent == cwd {
		outsideWrite = filepath.Join(os.TempDir(), ".gormes-security-audit-write-probe")
	}
	return toolspkg.SecurityAuditFilesystem{
		CWD:              cwd,
		ScopeConfigured:  true,
		ProbeReadPath:    "/etc/passwd",
		ProbeWritePath:   outsideWrite,
		ExpectDenyProbes: true,
	}
}

func cliSecurityAuditCredentialRedaction(secrets []string) toolspkg.SecurityAuditCredentialRedaction {
	samples := make([]string, 0, len(secrets))
	for _, secret := range secrets {
		secret = strings.TrimSpace(secret)
		if secret == "" {
			continue
		}
		samples = append(samples, "token="+secret)
	}
	return toolspkg.SecurityAuditCredentialRedaction{
		Secrets: secrets,
		Samples: samples,
	}
}

func cliSecurityAuditSecrets(cfg config.Config) []string {
	values := []string{
		cfg.Hermes.APIKey,
		cfg.Gateway.ProxyKey,
		cfg.Telegram.BotToken,
		cfg.Discord.Token,
		cfg.Slack.BotToken,
		cfg.Slack.AppToken,
		os.Getenv("GORMES_API_KEY"),
		os.Getenv("GATEWAY_PROXY_KEY"),
		os.Getenv("GORMES_TELEGRAM_BOT_TOKEN"),
		os.Getenv("GORMES_TELEGRAM_TOKEN"),
		os.Getenv("HERMES_TELEGRAM_BOT_TOKEN"),
		os.Getenv("HERMES_TELEGRAM_TOKEN"),
		os.Getenv("TELEGRAM_BOT_TOKEN"),
		os.Getenv("TELEGRAM_TOKEN"),
		os.Getenv("GORMES_DISCORD_TOKEN"),
		os.Getenv("GORMES_SLACK_BOT_TOKEN"),
		os.Getenv("GORMES_SLACK_APP_TOKEN"),
	}
	secrets := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		secrets = append(secrets, value)
	}
	return secrets
}

type cliSecurityAuditSecretRefsResult struct {
	Refs    []toolspkg.SecurityAuditSecretRef
	Secrets []string
}

type cliSecurityAuditSecretRefTarget struct {
	Path   string
	Ref    *config.SecretRef
	Active bool
}

func cliSecurityAuditSecretRefs(cfg config.Config, baseSecrets []string) cliSecurityAuditSecretRefsResult {
	resolver := config.NewSecretResolver(config.SecretResolverConfig{Secrets: cfg.Secrets})
	result := cliSecurityAuditSecretRefsResult{}
	targets := []cliSecurityAuditSecretRefTarget{
		{Path: "hermes.api_key", Ref: cfg.Hermes.APIKeyRef, Active: cfg.Hermes.APIKeyRef != nil},
		{Path: "telegram.bot_token", Ref: cfg.Telegram.BotTokenRef, Active: cfg.Telegram.BotTokenRef != nil},
		{Path: "discord.token", Ref: cfg.Discord.TokenRef, Active: cfg.Discord.TokenRef != nil},
		{Path: "slack.bot_token", Ref: cfg.Slack.BotTokenRef, Active: cfg.Slack.BotTokenRef != nil},
		{Path: "slack.app_token", Ref: cfg.Slack.AppTokenRef, Active: cfg.Slack.AppTokenRef != nil},
	}
	redactions := append([]string{}, baseSecrets...)
	for _, target := range targets {
		if target.Ref == nil {
			continue
		}
		ref := normalizeCLISecurityAuditSecretRef(*target.Ref)
		entry := toolspkg.SecurityAuditSecretRef{
			Path:     target.Path,
			Active:   target.Active,
			Source:   string(ref.Source),
			Provider: ref.Provider,
			ID:       ref.ID,
		}
		if ref.Source == config.SecretRefSourceExec {
			entry.Available = false
			entry.EvidenceCode = config.SecretRefEvidenceUnsupported
			entry.Message = "exec SecretRef providers are not supported by security audit"
			result.Refs = append(result.Refs, entry)
			continue
		}
		value, evidence, err := resolver.ResolveString(ref)
		entry.Available = err == nil
		entry.EvidenceCode = evidence.Code
		if evidence.Source != "" {
			entry.Source = evidence.Source
		}
		if evidence.Provider != "" {
			entry.Provider = evidence.Provider
		}
		if evidence.ID != "" {
			entry.ID = evidence.ID
		}
		if err != nil {
			entry.Message = toolspkg.RedactSecurityAuditText(err.Error(), redactions)
		} else {
			entry.Message = "SecretRef resolved"
			result.Secrets = append(result.Secrets, value)
			redactions = append(redactions, value)
		}
		result.Refs = append(result.Refs, entry)
	}
	result.Secrets = appendUniqueSecurityAuditSecrets(nil, result.Secrets...)
	return result
}

func normalizeCLISecurityAuditSecretRef(ref config.SecretRef) config.SecretRef {
	ref.Source = config.SecretRefSource(strings.ToLower(strings.TrimSpace(string(ref.Source))))
	ref.Provider = strings.TrimSpace(ref.Provider)
	if ref.Provider == "" {
		ref.Provider = config.DefaultSecretProviderAlias
	}
	ref.ID = strings.TrimSpace(ref.ID)
	return ref
}

func cliSecurityAuditSecretRefAvailability(refs []toolspkg.SecurityAuditSecretRef) map[string]bool {
	availability := map[string]bool{}
	for _, ref := range refs {
		if ref.Active && ref.Available {
			availability[ref.Path] = true
		}
	}
	return availability
}

func appendUniqueSecurityAuditSecrets(secrets []string, values ...string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(secrets)+len(values))
	for _, value := range append(append([]string{}, secrets...), values...) {
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
	return out
}

func cliSecurityAuditFixCandidates() []toolspkg.SecurityAuditFixCandidate {
	paths := []string{
		config.ConfigPath(),
		config.EnvPath(),
		filepath.Join(config.GormesHome(), "secrets-runtime.json"),
	}
	var fixes []toolspkg.SecurityAuditFixCandidate
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		mode := int(info.Mode().Perm())
		if mode&0o077 == 0 {
			continue
		}
		fixes = append(fixes, toolspkg.SecurityAuditFixCandidate{
			Code:        toolspkg.SecurityAuditFixFilePermissions,
			Category:    toolspkg.SecurityAuditCategoryStateIntegrity,
			Path:        path,
			CurrentMode: mode,
			DesiredMode: 0o600,
			Safe:        true,
			Message:     "restrict file permissions to the operator account",
		})
	}
	return fixes
}

func cliSecurityAuditFixApplier(candidate toolspkg.SecurityAuditFixCandidate) error {
	if candidate.Code != toolspkg.SecurityAuditFixFilePermissions {
		return fmt.Errorf("unsupported security fix %s", candidate.Code)
	}
	if strings.TrimSpace(candidate.Path) == "" {
		return errors.New("empty fix path")
	}
	mode := os.FileMode(candidate.DesiredMode)
	if mode == 0 {
		mode = 0o600
	}
	return os.Chmod(candidate.Path, mode)
}

func cliSecurityAuditGenerateGatewayToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw[:])
	if err := config.WriteEnvValue(config.EnvPath(), "GATEWAY_PROXY_KEY", token); err != nil {
		return "", err
	}
	return token, nil
}

// securityAuditReportJSON wraps toolspkg.SecurityAuditResult with
// build provenance so fleet automation aggregating audit findings
// across machines can attribute each result to the binary version
// that emitted it. Existing top-level fields stay addressable via
// struct embedding.
type securityAuditReportJSON struct {
	Build BuildProvenance `json:"build"`
	toolspkg.SecurityAuditResult
}

func WriteAuditResult(out io.Writer, result toolspkg.SecurityAuditResult, jsonOut bool, build BuildProvenance) {
	if out == nil {
		out = io.Discard
	}
	if jsonOut {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		_ = enc.Encode(securityAuditReportJSON{
			Build:               build,
			SecurityAuditResult: result,
		})
		return
	}
	fmt.Fprintf(out, "%s ok=%t pass=%d warn=%d fail=%d fixed=%d redacted=%t\n", result.Code, result.OK, result.Summary.Pass, result.Summary.Warn, result.Summary.Fail, result.Summary.Fixed, result.Redacted)
	for _, category := range result.Categories {
		fmt.Fprintf(out, "%s status=%s\n", category.Name, category.Status)
	}
	for _, finding := range result.Findings {
		fmt.Fprintf(out, "%s category=%s severity=%s path=%s action=%s redacted=%t\n", finding.Code, finding.Category, finding.Severity, finding.Path, finding.Action, finding.Redacted)
	}
	for _, fix := range result.Fixes {
		fmt.Fprintf(out, "%s category=%s applied=%t safe=%t path=%s redacted=%t\n", fix.Code, fix.Category, fix.Applied, fix.Safe, fix.Path, fix.Redacted)
	}
}
