package gateway

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

const DefaultStaleCodeCacheFreshness = 30 * time.Second

// RuntimeStaleCodeStatus classifies whether the live gateway process was
// booted from the same git revision as the managed source checkout.
type RuntimeStaleCodeStatus string

const (
	RuntimeStaleCodeFresh          RuntimeStaleCodeStatus = "fresh"
	RuntimeStaleCodeStale          RuntimeStaleCodeStatus = "stale"
	RuntimeStaleCodeGitUnavailable RuntimeStaleCodeStatus = "git_unavailable"
)

// RuntimeStaleCodeEvidence is dynamic read-only status evidence. It is
// generated during status reads and is not used for PID identity.
type RuntimeStaleCodeEvidence struct {
	Status           RuntimeStaleCodeStatus `json:"status,omitempty"`
	BootGitSHA       string                 `json:"boot_git_sha,omitempty"`
	CurrentGitSHA    string                 `json:"current_git_sha,omitempty"`
	Stale            bool                   `json:"stale"`
	RestartSuggested bool                   `json:"restart_suggested"`
	Evidence         []string               `json:"evidence,omitempty"`
	Message          string                 `json:"message,omitempty"`
	CheckedAt        string                 `json:"checked_at,omitempty"`
}

type StaleCodeChecker struct {
	SourceRoot     string
	CacheFreshness time.Duration
	now            func() time.Time

	mu     sync.Mutex
	cached staleCodeHeadCache
}

type staleCodeHeadCache struct {
	sourceRoot string
	sha        string
	err        error
	expiresAt  time.Time
	valid      bool
}

func NewStaleCodeChecker(sourceRoot string) *StaleCodeChecker {
	return &StaleCodeChecker{
		SourceRoot:     strings.TrimSpace(sourceRoot),
		CacheFreshness: DefaultStaleCodeCacheFreshness,
		now:            func() time.Time { return time.Now().UTC() },
	}
}

func (c *StaleCodeChecker) Check(bootGitSHA string) RuntimeStaleCodeEvidence {
	now := time.Now().UTC()
	if c != nil && c.now != nil {
		now = c.now().UTC()
	}
	checkedAt := now.Format(time.RFC3339Nano)
	boot := normalizeGitSHA(bootGitSHA)
	if boot == "" {
		return RuntimeStaleCodeEvidence{
			Status:    RuntimeStaleCodeGitUnavailable,
			Evidence:  []string{"stale_code_git_unavailable", "stale_code_boot_sha_unavailable"},
			Message:   "boot git SHA unavailable; skipping stale-code restart advice",
			CheckedAt: checkedAt,
		}
	}

	current, err := c.currentHEAD(now)
	if err != nil || current == "" {
		return RuntimeStaleCodeEvidence{
			Status:     RuntimeStaleCodeGitUnavailable,
			BootGitSHA: boot,
			Evidence:   []string{"stale_code_git_unavailable"},
			Message:    "git HEAD unavailable; skipping stale-code restart advice",
			CheckedAt:  checkedAt,
		}
	}

	if gitSHAMatches(boot, current) {
		return RuntimeStaleCodeEvidence{
			Status:        RuntimeStaleCodeFresh,
			BootGitSHA:    boot,
			CurrentGitSHA: current,
			Evidence:      []string{"stale_code_head_unchanged"},
			CheckedAt:     checkedAt,
		}
	}

	return RuntimeStaleCodeEvidence{
		Status:           RuntimeStaleCodeStale,
		BootGitSHA:       boot,
		CurrentGitSHA:    current,
		Stale:            true,
		RestartSuggested: true,
		Evidence:         []string{"stale_code_head_changed", "stale_code_restart_gateway"},
		Message:          "gateway restart recommended to load current git HEAD",
		CheckedAt:        checkedAt,
	}
}

func (c *StaleCodeChecker) currentHEAD(now time.Time) (string, error) {
	if c == nil {
		return "", errStaleCodeGitUnavailable
	}
	sourceRoot := strings.TrimSpace(c.SourceRoot)
	if sourceRoot == "" {
		return "", errStaleCodeGitUnavailable
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.CacheFreshness > 0 &&
		c.cached.valid &&
		c.cached.sourceRoot == sourceRoot &&
		now.Before(c.cached.expiresAt) {
		return c.cached.sha, c.cached.err
	}

	sha, err := currentGitHEADSHA(sourceRoot)
	c.cached = staleCodeHeadCache{
		sourceRoot: sourceRoot,
		sha:        sha,
		err:        err,
		expiresAt:  now.Add(c.CacheFreshness),
		valid:      true,
	}
	return sha, err
}

func RuntimeBootGitSHA() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return normalizeGitSHA(setting.Value)
			}
		}
	}
	if root := DefaultStaleCodeSourceRoot(); root != "" {
		if sha, err := currentGitHEADSHA(root); err == nil {
			return sha
		}
	}
	return ""
}

func DefaultStaleCodeSourceRoot() string {
	for _, candidate := range defaultStaleCodeSourceRootCandidates() {
		if root, ok := gitRootFromCandidate(candidate); ok {
			return root
		}
	}
	return ""
}

func defaultStaleCodeSourceRootCandidates() []string {
	candidates := []string{
		os.Getenv("GORMES_SOURCE_ROOT"),
		os.Getenv("GORMES_SOURCE_DIR"),
		os.Getenv("GORMES_INSTALL_DIR"),
		filepath.Join(config.GormesHome(), "gormes-agent"),
		"/usr/local/lib/gormes-agent",
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, cwd)
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			exeDir,
			filepath.Dir(exeDir),
			filepath.Join(filepath.Dir(exeDir), "gormes-agent"),
		)
	}
	return dedupeNonEmptyPaths(candidates)
}

func gitRootFromCandidate(candidate string) (string, bool) {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return "", false
	}
	abs, err := filepath.Abs(candidate)
	if err != nil {
		return "", false
	}
	root := findGitRoot(abs)
	if root == "" {
		return "", false
	}
	if _, err := resolveGitMetadataDir(root); err != nil {
		return "", false
	}
	return root, true
}

func findGitRoot(start string) string {
	current := filepath.Clean(start)
	info, err := os.Stat(current)
	if err == nil && !info.IsDir() {
		current = filepath.Dir(current)
	}
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func dedupeNonEmptyPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	seen := map[string]struct{}{}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		clean := filepath.Clean(path)
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	return out
}

var errStaleCodeGitUnavailable = errors.New("stale code git unavailable")

type gitMetadataDir struct {
	gitDir    string
	commonDir string
}

func currentGitHEADSHA(sourceRoot string) (string, error) {
	meta, err := resolveGitMetadataDir(sourceRoot)
	if err != nil {
		return "", err
	}
	headRaw, err := os.ReadFile(filepath.Join(meta.gitDir, "HEAD"))
	if err != nil {
		return "", fmt.Errorf("%w: read HEAD: %w", errStaleCodeGitUnavailable, err)
	}
	head := strings.TrimSpace(string(headRaw))
	if strings.HasPrefix(head, "ref:") {
		ref := strings.TrimSpace(strings.TrimPrefix(head, "ref:"))
		if !safeGitRef(ref) {
			return "", fmt.Errorf("%w: unsafe HEAD ref", errStaleCodeGitUnavailable)
		}
		return resolveGitRef(meta, ref)
	}
	if sha := normalizeGitSHA(head); sha != "" {
		return sha, nil
	}
	return "", fmt.Errorf("%w: unrecognized HEAD", errStaleCodeGitUnavailable)
}

func resolveGitMetadataDir(sourceRoot string) (gitMetadataDir, error) {
	gitPath := filepath.Join(sourceRoot, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return gitMetadataDir{}, fmt.Errorf("%w: .git missing: %w", errStaleCodeGitUnavailable, err)
	}
	if info.IsDir() {
		return gitMetadataDir{gitDir: gitPath, commonDir: gitPath}, nil
	}

	raw, err := os.ReadFile(gitPath)
	if err != nil {
		return gitMetadataDir{}, fmt.Errorf("%w: read .git file: %w", errStaleCodeGitUnavailable, err)
	}
	line := strings.TrimSpace(string(raw))
	if !strings.HasPrefix(line, "gitdir:") {
		return gitMetadataDir{}, fmt.Errorf("%w: unsupported .git file", errStaleCodeGitUnavailable)
	}
	gitDir := strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))
	if gitDir == "" {
		return gitMetadataDir{}, fmt.Errorf("%w: empty gitdir", errStaleCodeGitUnavailable)
	}
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(sourceRoot, gitDir)
	}
	gitDir = filepath.Clean(gitDir)
	commonDir := gitDir
	if raw, err := os.ReadFile(filepath.Join(gitDir, "commondir")); err == nil {
		commonDir = strings.TrimSpace(string(raw))
		if commonDir == "" {
			commonDir = gitDir
		}
		if !filepath.IsAbs(commonDir) {
			commonDir = filepath.Join(gitDir, commonDir)
		}
		commonDir = filepath.Clean(commonDir)
	}
	return gitMetadataDir{gitDir: gitDir, commonDir: commonDir}, nil
}

func resolveGitRef(meta gitMetadataDir, ref string) (string, error) {
	refPath := filepath.FromSlash(ref)
	for _, root := range dedupeNonEmptyPaths([]string{meta.gitDir, meta.commonDir}) {
		raw, err := os.ReadFile(filepath.Join(root, refPath))
		if err == nil {
			if sha := normalizeGitSHA(strings.TrimSpace(string(raw))); sha != "" {
				return sha, nil
			}
			return "", fmt.Errorf("%w: invalid loose ref", errStaleCodeGitUnavailable)
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: read loose ref: %w", errStaleCodeGitUnavailable, err)
		}
	}
	for _, root := range dedupeNonEmptyPaths([]string{meta.gitDir, meta.commonDir}) {
		if sha, ok, err := readPackedRef(filepath.Join(root, "packed-refs"), ref); ok || err != nil {
			if err != nil {
				return "", err
			}
			return sha, nil
		}
	}
	return "", fmt.Errorf("%w: ref %s not found", errStaleCodeGitUnavailable, ref)
}

func readPackedRef(path, ref string) (string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("%w: read packed refs: %w", errStaleCodeGitUnavailable, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "^") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == ref {
			sha := normalizeGitSHA(fields[0])
			if sha == "" {
				return "", false, fmt.Errorf("%w: invalid packed ref", errStaleCodeGitUnavailable)
			}
			return sha, true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", false, fmt.Errorf("%w: scan packed refs: %w", errStaleCodeGitUnavailable, err)
	}
	return "", false, nil
}

func safeGitRef(ref string) bool {
	if ref == "" || filepath.IsAbs(ref) || strings.Contains(ref, "\\") {
		return false
	}
	clean := filepath.Clean(filepath.FromSlash(ref))
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return false
	}
	return strings.HasPrefix(ref, "refs/")
}

func normalizeGitSHA(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if len(raw) < 7 || len(raw) > 64 {
		return ""
	}
	for _, r := range raw {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return ""
		}
	}
	return raw
}

func gitSHAMatches(a, b string) bool {
	a = normalizeGitSHA(a)
	b = normalizeGitSHA(b)
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	if len(a) < len(b) {
		return strings.HasPrefix(b, a)
	}
	return strings.HasPrefix(a, b)
}
