package tools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var checkpointHashRE = regexp.MustCompile(`^[0-9a-fA-F]{4,64}$`)

var defaultCheckpointExcludes = []string{
	"node_modules/",
	"dist/",
	"build/",
	".env",
	".env.*",
	".env.local",
	".env.*.local",
	"__pycache__/",
	"*.pyc",
	"*.pyo",
	".DS_Store",
	"*.log",
	".cache/",
	".next/",
	".nuxt/",
	"coverage/",
	".pytest_cache/",
	".venv/",
	"venv/",
	".git/",
}

// CheckpointManagerConfig configures the atomic checkpoint runtime.
type CheckpointManagerConfig struct {
	Enabled      bool
	MaxSnapshots int
	BaseDir      string
	GitTimeout   time.Duration
}

// Checkpoint describes one shadow-repo snapshot.
type Checkpoint struct {
	Hash         string
	ShortHash    string
	Timestamp    string
	Reason       string
	FilesChanged int
	Insertions   int
	Deletions    int
}

// CheckpointDiff holds the user-facing diff preview against a checkpoint.
type CheckpointDiff struct {
	Stat string
	Diff string
}

// RestoreResult reports the outcome of a rollback operation.
type RestoreResult struct {
	RestoredTo string
	Reason     string
	Directory  string
	File       string
}

// CheckpointManager manages per-turn deduplicated shadow-repo snapshots.
type CheckpointManager struct {
	enabled          bool
	maxSnapshots     int
	baseDir          string
	gitTimeout       time.Duration
	gitPath          string
	mu               sync.Mutex
	checkpointedDirs map[string]struct{}
}

// NewCheckpointManager constructs a Go-native atomic checkpoint manager.
func NewCheckpointManager(cfg CheckpointManagerConfig) *CheckpointManager {
	maxSnapshots := cfg.MaxSnapshots
	if maxSnapshots <= 0 {
		maxSnapshots = 50
	}
	gitTimeout := cfg.GitTimeout
	if gitTimeout <= 0 {
		gitTimeout = 30 * time.Second
	}
	baseDir := strings.TrimSpace(cfg.BaseDir)
	if baseDir == "" {
		baseDir = filepath.Join(xdgDataHome(), "gormes", "checkpoints")
	}
	gitPath, _ := exec.LookPath("git")
	return &CheckpointManager{
		enabled:          cfg.Enabled,
		maxSnapshots:     maxSnapshots,
		baseDir:          baseDir,
		gitTimeout:       gitTimeout,
		gitPath:          gitPath,
		checkpointedDirs: make(map[string]struct{}),
	}
}

// NewTurn resets the once-per-directory dedup boundary.
func (m *CheckpointManager) NewTurn() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkpointedDirs = make(map[string]struct{})
}

// EnsureCheckpoint snapshots the working directory once per turn.
func (m *CheckpointManager) EnsureCheckpoint(workingDir, reason string) bool {
	if !m.enabled || m.gitPath == "" {
		return false
	}
	normalized, err := normalizeCheckpointPath(workingDir)
	if err != nil {
		return false
	}
	home, _ := os.UserHomeDir()
	if normalized == string(filepath.Separator) || (home != "" && normalized == filepath.Clean(home)) {
		return false
	}

	m.mu.Lock()
	if _, ok := m.checkpointedDirs[normalized]; ok {
		m.mu.Unlock()
		return false
	}
	m.checkpointedDirs[normalized] = struct{}{}
	m.mu.Unlock()

	return m.take(normalized, reason)
}

// List returns the most recent checkpoints for a working directory.
func (m *CheckpointManager) List(workingDir string) ([]Checkpoint, error) {
	normalized, err := normalizeCheckpointPath(workingDir)
	if err != nil {
		return nil, err
	}
	ok, err := m.hasShadowRepo(normalized)
	if err != nil {
		return nil, err
	}
	if !ok {
		return []Checkpoint{}, nil
	}

	result, err := m.runGit(normalized, map[int]struct{}{}, "log", "--format=%H|%h|%aI|%s", "-n", strconv.Itoa(m.maxSnapshots))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(result.stdout) == "" {
		return []Checkpoint{}, nil
	}

	lines := strings.Split(strings.TrimSpace(result.stdout), "\n")
	out := make([]Checkpoint, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, "|", 4)
		if len(parts) != 4 {
			continue
		}
		cp := Checkpoint{
			Hash:      parts[0],
			ShortHash: parts[1],
			Timestamp: parts[2],
			Reason:    parts[3],
		}
		stat, err := m.runGit(normalized, map[int]struct{}{128: {}}, "diff", "--shortstat", parts[0]+"~1", parts[0])
		if err == nil {
			parseShortStat(stat.stdout, &cp)
		}
		out = append(out, cp)
	}
	return out, nil
}

// Diff returns a staged diff preview between the current tree and a checkpoint.
func (m *CheckpointManager) Diff(workingDir, commitHash string) (CheckpointDiff, error) {
	normalized, err := normalizeCheckpointPath(workingDir)
	if err != nil {
		return CheckpointDiff{}, err
	}
	if err := validateCommitHash(commitHash); err != nil {
		return CheckpointDiff{}, err
	}
	if err := m.ensureCommitExists(normalized, commitHash); err != nil {
		return CheckpointDiff{}, err
	}

	if _, err := m.runGit(normalized, map[int]struct{}{}, "add", "-A"); err != nil {
		return CheckpointDiff{}, err
	}
	defer func() {
		_, _ = m.runGit(normalized, map[int]struct{}{}, "reset", "HEAD", "--quiet")
	}()

	stat, statErr := m.runGit(normalized, map[int]struct{}{}, "diff", "--stat", commitHash, "--cached")
	diff, diffErr := m.runGit(normalized, map[int]struct{}{}, "diff", commitHash, "--cached", "--no-color")
	if statErr != nil && diffErr != nil {
		return CheckpointDiff{}, fmt.Errorf("generate checkpoint diff: %w", diffErr)
	}

	var out CheckpointDiff
	if statErr == nil {
		out.Stat = stat.stdout
	}
	if diffErr == nil {
		out.Diff = diff.stdout
	}
	return out, nil
}

// Restore restores either the whole working tree or a single relative path.
func (m *CheckpointManager) Restore(workingDir, commitHash, filePath string) (RestoreResult, error) {
	normalized, err := normalizeCheckpointPath(workingDir)
	if err != nil {
		return RestoreResult{}, err
	}
	if err := validateCommitHash(commitHash); err != nil {
		return RestoreResult{}, err
	}
	if err := validateRestorePath(normalized, filePath); err != nil {
		return RestoreResult{}, err
	}
	if err := m.ensureCommitExists(normalized, commitHash); err != nil {
		return RestoreResult{}, err
	}

	shortHash, err := m.shortHash(normalized, commitHash)
	if err != nil {
		return RestoreResult{}, err
	}
	_ = m.take(normalized, fmt.Sprintf("pre-rollback snapshot (restoring to %s)", shortHash))

	target := "."
	if strings.TrimSpace(filePath) != "" {
		target = filePath
	}
	if _, err := m.runGit(normalized, map[int]struct{}{}, "checkout", commitHash, "--", target); err != nil {
		return RestoreResult{}, err
	}
	reason := "unknown"
	if res, err := m.runGit(normalized, map[int]struct{}{}, "log", "--format=%s", "-1", commitHash); err == nil && strings.TrimSpace(res.stdout) != "" {
		reason = strings.TrimSpace(res.stdout)
	}

	out := RestoreResult{
		RestoredTo: shortHash,
		Reason:     reason,
		Directory:  normalized,
	}
	if target != "." {
		out.File = target
	}
	return out, nil
}

func (m *CheckpointManager) take(workingDir, reason string) bool {
	if err := m.initShadowRepo(workingDir); err != nil {
		return false
	}
	if _, err := m.runGit(workingDir, map[int]struct{}{}, "add", "-A"); err != nil {
		return false
	}
	diff, err := m.runGit(workingDir, map[int]struct{}{1: {}}, "diff", "--cached", "--quiet")
	if err != nil {
		return false
	}
	if diff.code == 0 {
		return false
	}
	if _, err := m.runGit(workingDir, map[int]struct{}{}, "commit", "-m", reason, "--allow-empty-message", "--no-gpg-sign"); err != nil {
		return false
	}
	return true
}

func (m *CheckpointManager) initShadowRepo(workingDir string) error {
	ok, err := m.hasShadowRepo(workingDir)
	if err != nil {
		return err
	}
	shadow := m.shadowRepoPath(workingDir)
	if ok {
		return nil
	}

	if err := os.MkdirAll(shadow, 0o755); err != nil {
		return err
	}
	if _, err := m.runGit(workingDir, map[int]struct{}{}, "init"); err != nil {
		return err
	}
	for _, args := range [][]string{
		{"config", "user.email", "gormes@local"},
		{"config", "user.name", "Gormes Checkpoint"},
		{"config", "commit.gpgsign", "false"},
		{"config", "tag.gpgSign", "false"},
	} {
		if _, err := m.runGit(workingDir, map[int]struct{}{}, args...); err != nil {
			return err
		}
	}

	infoDir := filepath.Join(shadow, "info")
	if err := os.MkdirAll(infoDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(infoDir, "exclude"), []byte(strings.Join(defaultCheckpointExcludes, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(shadow, "GORMES_WORKDIR"), []byte(workingDir+"\n"), 0o644); err != nil {
		return err
	}
	return nil
}

func (m *CheckpointManager) ensureCommitExists(workingDir, commitHash string) error {
	ok, err := m.hasShadowRepo(workingDir)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("no checkpoints exist for this directory")
	}
	if _, err := m.runGit(workingDir, map[int]struct{}{}, "cat-file", "-t", commitHash); err != nil {
		return fmt.Errorf("checkpoint %q not found: %w", commitHash, err)
	}
	return nil
}

func (m *CheckpointManager) shortHash(workingDir, commitHash string) (string, error) {
	res, err := m.runGit(workingDir, map[int]struct{}{}, "rev-parse", "--short", commitHash)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(res.stdout), nil
}

func (m *CheckpointManager) shadowRepoPath(workingDir string) string {
	sum := sha256.Sum256([]byte(workingDir))
	return filepath.Join(m.baseDir, fmt.Sprintf("%x", sum)[:16])
}

func (m *CheckpointManager) hasShadowRepo(workingDir string) (bool, error) {
	_, err := os.Stat(filepath.Join(m.shadowRepoPath(workingDir), "HEAD"))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

type gitCommandResult struct {
	stdout string
	stderr string
	code   int
}

func (m *CheckpointManager) runGit(workingDir string, allowedCodes map[int]struct{}, args ...string) (gitCommandResult, error) {
	shadow := m.shadowRepoPath(workingDir)
	ctx, cancel := context.WithTimeout(context.Background(), m.gitTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, m.gitPath, args...)
	cmd.Dir = workingDir
	cmd.Env = gitEnv(shadow, workingDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	res := gitCommandResult{
		stdout: strings.TrimSpace(stdout.String()),
		stderr: strings.TrimSpace(stderr.String()),
	}
	if err == nil {
		return res, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		res.code = exitErr.ExitCode()
		if _, ok := allowedCodes[res.code]; ok {
			return res, nil
		}
		return res, fmt.Errorf("git %s: exit %d: %s", strings.Join(args, " "), res.code, res.stderr)
	}
	if ctx.Err() != nil {
		return res, ctx.Err()
	}
	return res, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
}

func gitEnv(shadowRepo, workingDir string) []string {
	base := make([]string, 0, len(os.Environ())+5)
	for _, raw := range os.Environ() {
		if strings.HasPrefix(raw, "GIT_DIR=") ||
			strings.HasPrefix(raw, "GIT_WORK_TREE=") ||
			strings.HasPrefix(raw, "GIT_INDEX_FILE=") ||
			strings.HasPrefix(raw, "GIT_NAMESPACE=") ||
			strings.HasPrefix(raw, "GIT_ALTERNATE_OBJECT_DIRECTORIES=") ||
			strings.HasPrefix(raw, "GIT_CONFIG_GLOBAL=") ||
			strings.HasPrefix(raw, "GIT_CONFIG_SYSTEM=") ||
			strings.HasPrefix(raw, "GIT_CONFIG_NOSYSTEM=") {
			continue
		}
		base = append(base, raw)
	}
	base = append(base,
		"GIT_DIR="+shadowRepo,
		"GIT_WORK_TREE="+workingDir,
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_SYSTEM="+os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
	)
	return base
}

func normalizeCheckpointPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("empty path")
	}
	if path == "~" || strings.HasPrefix(path, "~"+string(filepath.Separator)) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~"+string(filepath.Separator)))
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err == nil {
		abs = resolved
	}
	return filepath.Clean(abs), nil
}

func validateCommitHash(commitHash string) error {
	if commitHash == "" {
		return errors.New("empty commit hash")
	}
	if strings.HasPrefix(commitHash, "-") {
		return fmt.Errorf("invalid commit hash %q", commitHash)
	}
	if !checkpointHashRE.MatchString(commitHash) {
		return fmt.Errorf("invalid commit hash %q", commitHash)
	}
	return nil
}

func validateRestorePath(workingDir, filePath string) error {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return nil
	}
	if filepath.IsAbs(filePath) {
		return fmt.Errorf("restore path must be relative: %q", filePath)
	}
	target := filepath.Clean(filepath.Join(workingDir, filePath))
	rel, err := filepath.Rel(workingDir, target)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("restore path escapes working directory: %q", filePath)
	}
	return nil
}

func parseShortStat(statLine string, cp *Checkpoint) {
	fields := strings.Split(statLine, ",")
	for _, field := range fields {
		field = strings.TrimSpace(field)
		switch {
		case strings.Contains(field, "file changed"), strings.Contains(field, "files changed"):
			cp.FilesChanged = leadingInt(field)
		case strings.Contains(field, "insertion"), strings.Contains(field, "insertions"):
			cp.Insertions = leadingInt(field)
		case strings.Contains(field, "deletion"), strings.Contains(field, "deletions"):
			cp.Deletions = leadingInt(field)
		}
	}
}

func leadingInt(text string) int {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return 0
	}
	n, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0
	}
	return n
}

func xdgDataHome() string {
	if v := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share")
}
