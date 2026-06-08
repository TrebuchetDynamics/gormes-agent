package externalsecrets

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials"
)

const (
	BitwardenSourceLabel            = "bitwarden"
	BitwardenBWSVersion             = "2.0.0"
	BitwardenChecksumName           = "bws-sha256-checksums-" + BitwardenBWSVersion + ".txt"
	DefaultBitwardenAccessTokenEnv  = "BWS_ACCESS_TOKEN"
	DefaultBitwardenCacheTTLSeconds = 300
	DefaultBitwardenRunTimeout      = 30 * time.Second
	DefaultBitwardenDownloadTimeout = 60 * time.Second
)

const defaultBitwardenReleaseBase = "https://github.com/bitwarden/sdk-sm/releases/download/bws-v" + BitwardenBWSVersion

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
	DryRun    bool
}

type BitwardenInstallOptions struct {
	HomeDir     string
	Force       bool
	ReleaseBase string
	System      string
	Machine     string
	Libc        string
	Download    func(context.Context, string) ([]byte, error)
	Timeout     time.Duration
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

var bitwardenProcessCache = map[bitwardenCacheKey]bitwardenCachedFetch{}

func ResetSecretSourcesForTests() {
	secretSources = map[string]string{}
	bitwardenProcessCache = map[bitwardenCacheKey]bitwardenCachedFetch{}
}

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
	secrets, warnings, binary, err := fetchBitwardenSecrets(ctx, cfg, tokenEnv, token, projectID, opts)
	if err != nil {
		report.Error = err.Error()
		return report
	}
	report.BinaryPath = binary
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
		if opts.DryRun {
			report.Applied = append(report.Applied, key)
			continue
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

func FindBitwardenBinary(cfg BitwardenConfig, opts BitwardenOptions) (string, error) {
	return findBWS(cfg, opts)
}

func findBWS(cfg BitwardenConfig, opts BitwardenOptions) (string, error) {
	name := bitwardenBinaryName(BitwardenInstallOptions{})
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
		return InstallBitwardenBWS(context.Background(), BitwardenInstallOptions{HomeDir: opts.HomeDir})
	}
	return "", errors.New("bws binary not available and auto_install is false")
}

func InstallBitwardenBWS(ctx context.Context, opts BitwardenInstallOptions) (string, error) {
	home := strings.TrimSpace(opts.HomeDir)
	if home == "" {
		return "", errors.New("bitwarden install: home dir is empty")
	}
	binName := bitwardenBinaryName(opts)
	target := filepath.Join(home, "bin", binName)
	if info, err := os.Stat(target); err == nil && !info.IsDir() && !opts.Force {
		return target, nil
	}
	assetName, err := BitwardenAssetName(opts)
	if err != nil {
		return "", err
	}
	releaseBase := strings.TrimRight(strings.TrimSpace(opts.ReleaseBase), "/")
	if releaseBase == "" {
		releaseBase = defaultBitwardenReleaseBase
	}
	downloader := opts.Download
	if downloader == nil {
		downloader = httpDownloadBitwarden(opts.Timeout)
	}
	zipBytes, err := downloader(ctx, releaseBase+"/"+assetName)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", assetName, err)
	}
	checksumBytes, err := downloader(ctx, releaseBase+"/"+BitwardenChecksumName)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", BitwardenChecksumName, err)
	}
	expected, err := expectedBitwardenSHA256(checksumBytes, assetName)
	if err != nil {
		return "", err
	}
	actualSum := sha256.Sum256(zipBytes)
	actual := hex.EncodeToString(actualSum[:])
	if !strings.EqualFold(expected, actual) {
		return "", fmt.Errorf("checksum mismatch for %s: expected %s, got %s", assetName, expected, actual)
	}
	body, err := extractBitwardenBinary(zipBytes, binName)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return "", fmt.Errorf("mkdir bws bin dir: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), ".bws-*")
	if err != nil {
		return "", fmt.Errorf("stage bws: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return "", fmt.Errorf("write staged bws: %w", err)
	}
	if err := tmp.Chmod(0o755); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return "", fmt.Errorf("chmod staged bws: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return "", fmt.Errorf("close staged bws: %w", err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		os.Remove(tmpName)
		return "", fmt.Errorf("install bws: %w", err)
	}
	return target, nil
}

func InstallBitwardenBinary(ctx context.Context, opts BitwardenInstallOptions) (string, error) {
	return InstallBitwardenBWS(ctx, opts)
}

func BitwardenAssetName(opts BitwardenInstallOptions) (string, error) {
	system := strings.ToLower(strings.TrimSpace(opts.System))
	if system == "" {
		system = runtime.GOOS
	}
	machine := strings.ToLower(strings.TrimSpace(opts.Machine))
	if machine == "" {
		machine = runtime.GOARCH
	}
	switch system {
	case "darwin":
		return "bws-macos-universal-" + BitwardenBWSVersion + ".zip", nil
	case "windows":
		return "bws-" + bitwardenArch(machine) + "-pc-windows-msvc-" + BitwardenBWSVersion + ".zip", nil
	case "linux":
		libc := strings.ToLower(strings.TrimSpace(opts.Libc))
		if libc == "" {
			libc = detectLinuxLibc()
		}
		if libc != "musl" {
			libc = "gnu"
		}
		return "bws-" + bitwardenArch(machine) + "-unknown-linux-" + libc + "-" + BitwardenBWSVersion + ".zip", nil
	default:
		return "", fmt.Errorf("unsupported platform for bws auto-install: %s %s", system, machine)
	}
}

func bitwardenBinaryName(opts BitwardenInstallOptions) string {
	system := strings.ToLower(strings.TrimSpace(opts.System))
	if system == "" {
		system = runtime.GOOS
	}
	if system == "windows" {
		return "bws.exe"
	}
	return "bws"
}

func bitwardenArch(machine string) string {
	switch strings.ToLower(machine) {
	case "arm64", "aarch64":
		return "aarch64"
	default:
		return "x86_64"
	}
}

func detectLinuxLibc() string {
	out, err := exec.Command("ldd", "--version").CombinedOutput()
	if err == nil && strings.Contains(strings.ToLower(string(out)), "musl") {
		return "musl"
	}
	return "gnu"
}

func expectedBitwardenSHA256(checksumBytes []byte, assetName string) (string, error) {
	for _, line := range strings.Split(string(checksumBytes), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) >= 2 && fields[len(fields)-1] == assetName {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no checksum entry for %s in %s", assetName, BitwardenChecksumName)
}

func extractBitwardenBinary(zipBytes []byte, binaryName string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, fmt.Errorf("open bws zip: %w", err)
	}
	for _, f := range zr.File {
		name := filepath.ToSlash(f.Name)
		if filepath.IsAbs(f.Name) || name == ".." || strings.HasPrefix(name, "../") || strings.Contains(name, "/../") {
			return nil, fmt.Errorf("unsafe archive member %q escapes extraction directory", f.Name)
		}
	}
	var chosen *zip.File
	for _, f := range zr.File {
		if filepath.Base(filepath.ToSlash(f.Name)) == binaryName && (chosen == nil || len(f.Name) < len(chosen.Name)) {
			chosen = f
		}
	}
	if chosen == nil {
		return nil, fmt.Errorf("could not find %s inside downloaded archive", binaryName)
	}
	r, err := chosen.Open()
	if err != nil {
		return nil, fmt.Errorf("open bws archive member: %w", err)
	}
	defer r.Close()
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read bws archive member: %w", err)
	}
	return body, nil
}

func httpDownloadBitwarden(timeout time.Duration) func(context.Context, string) ([]byte, error) {
	if timeout <= 0 {
		timeout = DefaultBitwardenDownloadTimeout
	}
	client := &http.Client{Timeout: timeout}
	return func(ctx context.Context, url string) ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "gormes-agent")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		return io.ReadAll(resp.Body)
	}
}

type bitwardenCacheKey struct {
	tokenFingerprint string
	projectID        string
	serverURL        string
}

type bitwardenCachedFetch struct {
	Secrets   map[string]string
	FetchedAt float64
}

func (c bitwardenCachedFetch) fresh(ttlSeconds int) bool {
	if ttlSeconds <= 0 {
		return false
	}
	return time.Since(time.Unix(0, int64(c.FetchedAt*float64(time.Second)))) < time.Duration(ttlSeconds)*time.Second
}

func fetchBitwardenSecrets(ctx context.Context, cfg BitwardenConfig, tokenEnv, token, projectID string, opts BitwardenOptions) (map[string]string, []string, string, error) {
	serverURL := strings.TrimSpace(cfg.ServerURL)
	ttl := cfg.CacheTTLSeconds
	useCache := ttl > 0
	key := bitwardenCacheKey{tokenFingerprint: bitwardenTokenFingerprint(token), projectID: projectID, serverURL: serverURL}
	if useCache {
		if cached, ok := bitwardenProcessCache[key]; ok && cached.fresh(ttl) {
			return cloneStringMap(cached.Secrets), nil, "", nil
		}
		if cached, ok := readBitwardenDiskCache(opts.HomeDir, key, ttl); ok {
			bitwardenProcessCache[key] = cached
			return cloneStringMap(cached.Secrets), nil, "", nil
		}
	}
	binary, err := findBWS(cfg, opts)
	if err != nil {
		return nil, nil, "", err
	}
	stdout, stderr, err := runBWS(ctx, binary, tokenEnv, token, projectID, serverURL, opts)
	if err != nil {
		message := strings.TrimSpace(string(stderr))
		if message == "" {
			message = err.Error()
		}
		return nil, nil, binary, fmt.Errorf("bws failed: %s", truncate(message, 200))
	}
	secrets, warnings, err := parseBWSSecrets(stdout)
	if err != nil {
		return nil, nil, binary, err
	}
	entry := bitwardenCachedFetch{Secrets: cloneStringMap(secrets), FetchedAt: float64(time.Now().UnixNano()) / float64(time.Second)}
	if useCache {
		bitwardenProcessCache[key] = entry
		writeBitwardenDiskCache(opts.HomeDir, key, entry)
	}
	return secrets, warnings, binary, nil
}

func bitwardenTokenFingerprint(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])[:16]
}

func bitwardenCacheKeyString(key bitwardenCacheKey) string {
	return key.tokenFingerprint + "|" + key.projectID + "|" + key.serverURL
}

func bitwardenDiskCachePath(home string) string {
	home = strings.TrimSpace(home)
	if home == "" {
		home = os.Getenv("GORMES_HOME")
	}
	if home == "" {
		home = filepath.Join(os.Getenv("HOME"), ".gormes")
	}
	return filepath.Join(home, "cache", "bws_cache.json")
}

func readBitwardenDiskCache(home string, key bitwardenCacheKey, ttlSeconds int) (bitwardenCachedFetch, bool) {
	var zero bitwardenCachedFetch
	if ttlSeconds <= 0 {
		return zero, false
	}
	body, err := os.ReadFile(bitwardenDiskCachePath(home))
	if err != nil {
		return zero, false
	}
	var payload struct {
		Key       string            `json:"key"`
		FetchedAt float64           `json:"fetched_at"`
		Secrets   map[string]string `json:"secrets"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return zero, false
	}
	if payload.Key != bitwardenCacheKeyString(key) || payload.FetchedAt == 0 || payload.Secrets == nil {
		return zero, false
	}
	for name := range payload.Secrets {
		if !envNamePattern.MatchString(name) {
			return zero, false
		}
	}
	entry := bitwardenCachedFetch{Secrets: cloneStringMap(payload.Secrets), FetchedAt: payload.FetchedAt}
	if !entry.fresh(ttlSeconds) {
		return zero, false
	}
	return entry, true
}

func writeBitwardenDiskCache(home string, key bitwardenCacheKey, entry bitwardenCachedFetch) {
	path := bitwardenDiskCachePath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	payload := struct {
		Key       string            `json:"key"`
		FetchedAt float64           `json:"fetched_at"`
		Secrets   map[string]string `json:"secrets"`
	}{Key: bitwardenCacheKeyString(key), FetchedAt: entry.FetchedAt, Secrets: cloneStringMap(entry.Secrets)}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".bws_cache_*.tmp")
	if err != nil {
		return
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
	}
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
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
