package externalsecrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials"
)

const (
	BitwardenSourceLabel            = "bitwarden"
	DefaultBitwardenAccessTokenEnv  = "BWS_ACCESS_TOKEN"
	DefaultBitwardenCacheTTLSeconds = 300
	DefaultBitwardenRunTimeout      = 30 * time.Second
)

var envNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type BitwardenConfig struct {
	Enabled          bool   `toml:"enabled" yaml:"enabled"`
	AccessTokenEnv   string `toml:"access_token_env" yaml:"access_token_env"`
	ProjectID        string `toml:"project_id" yaml:"project_id"`
	CacheTTLSeconds  int    `toml:"cache_ttl_seconds" yaml:"cache_ttl_seconds"`
	OverrideExisting bool   `toml:"override_existing" yaml:"override_existing"`
	AutoInstall      bool   `toml:"auto_install" yaml:"auto_install"`
	ServerURL        string `toml:"server_url" yaml:"server_url"`
}

type BitwardenOptions struct {
	HomeDir   string
	LookupEnv func(string) (string, bool)
	SetEnv    func(string, string) error
	LookPath  func(string) (string, error)
	Run       func(context.Context, string, []string, []string) ([]byte, []byte, error)
	Timeout   time.Duration
}

type BitwardenReport struct {
	Enabled    bool
	BinaryPath string
	Applied    []string
	Skipped    []string
	Warnings   []string
	Error      string
}

func (r BitwardenReport) OK() bool { return r.Error == "" }

var secretSources = map[string]string{}

func ResetSecretSourcesForTests() { secretSources = map[string]string{} }

func GetSecretSource(envVar string) string { return secretSources[envVar] }

func ApplyBitwarden(ctx context.Context, cfg BitwardenConfig, opts BitwardenOptions) BitwardenReport {
	report := BitwardenReport{Enabled: cfg.Enabled}
	if !cfg.Enabled {
		return report
	}
	lookup := opts.LookupEnv
	if lookup == nil {
		lookup = os.LookupEnv
	}
	setenv := opts.SetEnv
	if setenv == nil {
		setenv = os.Setenv
	}
	tokenEnv := strings.TrimSpace(cfg.AccessTokenEnv)
	if tokenEnv == "" {
		tokenEnv = DefaultBitwardenAccessTokenEnv
	}
	token, ok := lookup(tokenEnv)
	if !ok || strings.TrimSpace(token) == "" {
		report.Error = fmt.Sprintf("%s is not set", tokenEnv)
		return report
	}
	projectID := strings.TrimSpace(cfg.ProjectID)
	if projectID == "" {
		report.Error = "project_id is empty"
		return report
	}
	binary, err := findBWS(cfg, opts)
	if err != nil {
		report.Error = err.Error()
		return report
	}
	report.BinaryPath = binary
	stdout, stderr, err := runBWS(ctx, binary, tokenEnv, token, projectID, strings.TrimSpace(cfg.ServerURL), opts)
	if err != nil {
		message := strings.TrimSpace(string(stderr))
		if message == "" {
			message = err.Error()
		}
		report.Error = "bws failed: " + truncate(message, 200)
		return report
	}
	secrets, warnings, err := parseBWSSecrets(stdout)
	if err != nil {
		report.Error = err.Error()
		return report
	}
	report.Warnings = append(report.Warnings, warnings...)
	for key, value := range secrets {
		if key == tokenEnv {
			report.Skipped = append(report.Skipped, key)
			continue
		}
		if !cfg.OverrideExisting {
			if existing, ok := lookup(key); ok && strings.TrimSpace(existing) != "" {
				report.Skipped = append(report.Skipped, key)
				continue
			}
		}
		if err := setenv(key, credentials.SanitizeCredentialValue(key, value)); err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("skipping %s: %v", key, err))
			continue
		}
		secretSources[key] = BitwardenSourceLabel
		report.Applied = append(report.Applied, key)
	}
	return report
}

func findBWS(cfg BitwardenConfig, opts BitwardenOptions) (string, error) {
	name := "bws"
	if opts.HomeDir != "" {
		managed := filepath.Join(opts.HomeDir, "bin", name)
		if info, err := os.Stat(managed); err == nil && !info.IsDir() {
			return managed, nil
		}
	}
	lookPath := opts.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	if path, err := lookPath(name); err == nil && path != "" {
		return path, nil
	}
	if cfg.AutoInstall {
		return "", errors.New("bws binary not available; auto-install is not implemented in Gormes yet")
	}
	return "", errors.New("bws binary not available and auto_install is false")
}

func runBWS(ctx context.Context, binary, tokenEnv, token, projectID, serverURL string, opts BitwardenOptions) ([]byte, []byte, error) {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultBitwardenRunTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	args := []string{"secret", "list", projectID, "--output", "json"}
	env := os.Environ()
	env = append(env, tokenEnv+"="+token, "NO_COLOR=1")
	if serverURL != "" {
		env = append(env, "BWS_SERVER_URL="+serverURL)
	}
	if opts.Run != nil {
		return opts.Run(ctx, binary, args, env)
	}
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = env
	stdout, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return stdout, exitErr.Stderr, err
		}
		return stdout, nil, err
	}
	return stdout, nil, nil
}

func parseBWSSecrets(stdout []byte) (map[string]string, []string, error) {
	var payload []map[string]any
	if err := json.Unmarshal(stdout, &payload); err != nil {
		return nil, nil, fmt.Errorf("bws returned non-JSON output")
	}
	secrets := map[string]string{}
	var warnings []string
	for _, item := range payload {
		key, _ := item["key"].(string)
		value, _ := item["value"].(string)
		if key == "" || value == "" {
			continue
		}
		if !envNamePattern.MatchString(key) {
			warnings = append(warnings, fmt.Sprintf("skipping %q: not a valid env-var name", key))
			continue
		}
		secrets[key] = value
	}
	return secrets, warnings, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
