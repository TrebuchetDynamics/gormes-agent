package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func newSecurityCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "security",
		Short:        "Audit gateway, channel, tool, filesystem, and credential security",
		SilenceUsage: true,
	}
	cmd.AddCommand(newSecurityAuditCommand())
	return cmd
}

func newSecurityAuditCommand() *cobra.Command {
	var deep bool
	var fix bool
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Run a redacted security audit with optional deep probes and safe fixes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(nil)
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}
			req := buildCLISecurityAuditRequest(cmd.Context(), cfg, deep, fix)
			result := toolspkg.AuditSecurity(req)
			writeSecurityAuditResult(cmd, result, jsonOut)
			if !result.OK {
				return newExitCodeError(1, errors.New("security audit found failures"))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&deep, "deep", false, "include live gateway probe checks")
	cmd.Flags().BoolVar(&fix, "fix", false, "apply safe deterministic fixes")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print machine-readable JSON")
	return cmd
}

func buildCLISecurityAuditRequest(ctx context.Context, cfg config.Config, deep, fix bool) toolspkg.SecurityAuditRequest {
	secrets := cliSecurityAuditSecrets(cfg)
	cwd, err := os.Getwd()
	if err != nil || strings.TrimSpace(cwd) == "" {
		cwd = config.GormesHome()
	}
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
		Channels:            cliSecurityAuditChannels(cfg),
		Filesystem:          cliSecurityAuditFilesystem(cwd),
		CredentialRedaction: cliSecurityAuditCredentialRedaction(secrets),
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
	snapshot, err := newGatewayStatusRuntimeStore(config.GatewayRuntimeStatusPath()).ReadValidatedRuntimeStatusSnapshot(ctx)
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
		secretsSnapshotPath(),
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

func cliSecurityAuditChannels(cfg config.Config) []toolspkg.SecurityAuditChannel {
	return []toolspkg.SecurityAuditChannel{
		{
			Name:                   "telegram",
			Enabled:                strings.TrimSpace(cfg.Telegram.BotToken) != "" || cfg.Telegram.AllowedChatID != 0 || cfg.Telegram.FirstRunDiscovery,
			TokenConfigured:        strings.TrimSpace(cfg.Telegram.BotToken) != "",
			AllowedScopeConfigured: cfg.Telegram.AllowedChatID != 0,
			FirstRunDiscovery:      cfg.Telegram.FirstRunDiscovery,
		},
		{
			Name:                   "discord",
			Enabled:                strings.TrimSpace(cfg.Discord.Token) != "" || strings.TrimSpace(cfg.Discord.AllowedChannelID) != "" || cfg.Discord.FirstRunDiscovery,
			TokenConfigured:        strings.TrimSpace(cfg.Discord.Token) != "",
			AllowedScopeConfigured: strings.TrimSpace(cfg.Discord.AllowedChannelID) != "",
			FirstRunDiscovery:      cfg.Discord.FirstRunDiscovery,
		},
		{
			Name:                   "slack",
			Enabled:                cfg.Slack.Enabled || strings.TrimSpace(cfg.Slack.BotToken) != "" || strings.TrimSpace(cfg.Slack.AppToken) != "" || strings.TrimSpace(cfg.Slack.AllowedChannelID) != "" || cfg.Slack.FirstRunDiscovery,
			TokenConfigured:        strings.TrimSpace(cfg.Slack.BotToken) != "" && strings.TrimSpace(cfg.Slack.AppToken) != "",
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
		os.Getenv("GORMES_TELEGRAM_TOKEN"),
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

func cliSecurityAuditFixCandidates() []toolspkg.SecurityAuditFixCandidate {
	paths := []string{
		config.ConfigPath(),
		config.EnvPath(),
		secretsSnapshotPath(),
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

func writeSecurityAuditResult(cmd *cobra.Command, result toolspkg.SecurityAuditResult, jsonOut bool) {
	if jsonOut {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s ok=%t pass=%d warn=%d fail=%d fixed=%d redacted=%t\n", result.Code, result.OK, result.Summary.Pass, result.Summary.Warn, result.Summary.Fail, result.Summary.Fixed, result.Redacted)
	for _, category := range result.Categories {
		fmt.Fprintf(cmd.OutOrStdout(), "%s status=%s\n", category.Name, category.Status)
	}
	for _, finding := range result.Findings {
		fmt.Fprintf(cmd.OutOrStdout(), "%s category=%s severity=%s path=%s action=%s redacted=%t\n", finding.Code, finding.Category, finding.Severity, finding.Path, finding.Action, finding.Redacted)
	}
	for _, fix := range result.Fixes {
		fmt.Fprintf(cmd.OutOrStdout(), "%s category=%s applied=%t safe=%t path=%s redacted=%t\n", fix.Code, fix.Category, fix.Applied, fix.Safe, fix.Path, fix.Redacted)
	}
}
