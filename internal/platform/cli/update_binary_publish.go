package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type UpdateBinaryPublishOptions struct {
	CheckoutDir       string
	ManagedBinPath    string
	PublishedBinPath  string
	ActivePathPath    string
	RefreshActivePath bool
	Runner            UpdateCommandRunner
	Git               UpdateGitRunner
}

type UpdateCommandRunner interface {
	RunCommand(ctx context.Context, cwd string, env []string, name string, args ...string) UpdateCommandResult
}

type UpdateCommandResult struct {
	Stdout string
	Stderr string
	Err    error
}

type RealUpdateCommandRunner struct{}

func (RealUpdateCommandRunner) RunCommand(ctx context.Context, cwd string, env []string, name string, args ...string) UpdateCommandResult {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = cwd
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	out, err := cmd.CombinedOutput()
	result := UpdateCommandResult{Stdout: string(out), Err: err}
	if err != nil {
		result.Stderr = string(out)
	}
	return result
}

func RunUpdateBinaryPublish(ctx context.Context, options UpdateBinaryPublishOptions) UpdateReport {
	report := UpdateReport{}
	runner := options.Runner
	if runner == nil {
		runner = RealUpdateCommandRunner{}
	}
	checkoutDir := strings.TrimSpace(options.CheckoutDir)
	if checkoutDir == "" {
		report.Failed = true
		report.add(UpdateEvidenceBuildFailed, "missing checkout directory")
		return report
	}
	managedBin := strings.TrimSpace(options.ManagedBinPath)
	publishedBin := strings.TrimSpace(options.PublishedBinPath)
	if managedBin == "" || publishedBin == "" {
		report.Failed = true
		report.add(UpdateEvidencePublishFailed, "missing managed or published binary path")
		return report
	}

	buildDir := filepath.Dir(managedBin)
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		report.Failed = true
		report.add(UpdateEvidenceBuildFailed, err.Error())
		return report
	}
	tmp, err := os.CreateTemp(buildDir, ".gormes-update-build-*")
	if err != nil {
		report.Failed = true
		report.add(UpdateEvidenceBuildFailed, err.Error())
		return report
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	_ = os.Remove(tmpPath)
	defer os.Remove(tmpPath)

	commit := updateBuildCommit(ctx, options.Git, checkoutDir)
	dirty := updateBuildDirty(ctx, options.Git, checkoutDir)
	version := updateBuildVersion(checkoutDir)
	ldflags := fmt.Sprintf("-s -w -X main.Version=%s -X main.GitCommit=%s -X main.GitDirty=%t", version, commit, dirty)
	build := runner.RunCommand(ctx, checkoutDir, []string{"CGO_ENABLED=0"}, "go", "build", "-trimpath", "-ldflags", ldflags, "-o", tmpPath, "./cmd/gormes")
	if build.Err != nil {
		report.Failed = true
		report.add(UpdateEvidenceBuildFailed, commandFailureDetail(build))
		return report
	}
	if info, err := os.Stat(tmpPath); err != nil || info.IsDir() {
		report.Failed = true
		report.add(UpdateEvidenceBuildFailed, fmt.Sprintf("build completed but %s was not created", tmpPath))
		return report
	}
	_ = os.Chmod(tmpPath, 0o755)
	report.add(UpdateEvidenceBuildCompleted, fmt.Sprintf("built %s from %s (%s)", managedBin, checkoutDir, commit))

	managedReport := publishBinaryPath(ctx, runner, tmpPath, managedBin, false, "managed binary")
	report.append(managedReport)
	if report.Failed {
		return report
	}

	publishedReport := publishBinaryPath(ctx, runner, managedBin, publishedBin, true, "published command")
	report.append(publishedReport)
	if report.Failed {
		return report
	}

	activePath := strings.TrimSpace(options.ActivePathPath)
	switch {
	case activePath == "":
		report.add(UpdateEvidenceActivePathSkipped, "no active PATH command detected")
	case !options.RefreshActivePath:
		report.add(UpdateEvidenceActivePathSkipped, "sandbox bin dir set; respecting active PATH boundary")
	case samePath(activePath, managedBin), samePath(activePath, publishedBin):
		report.add(UpdateEvidenceActivePathSkipped, "active PATH command already points at the updated binary")
	case sameBinary(activePath, managedBin):
		report.add(UpdateEvidenceActivePathSkipped, "active PATH command already has the updated binary content")
	default:
		activeReport := publishBinaryPath(ctx, runner, managedBin, activePath, true, "active PATH command")
		if activeReport.Failed {
			report.add(UpdateEvidenceActivePathFailed, summarizeUpdateEvidence(activeReport.Evidence))
			return report
		}
		report.add(UpdateEvidenceActivePathRefreshed, activePath)
	}

	return report
}

func updateBuildCommit(ctx context.Context, git UpdateGitRunner, checkoutDir string) string {
	if git == nil {
		return "unknown"
	}
	result := git.RunGit(ctx, checkoutDir, "rev-parse", "--short", "HEAD")
	commit := strings.TrimSpace(result.Stdout)
	if result.Err != nil || commit == "" {
		return "unknown"
	}
	return commit
}

func updateBuildDirty(ctx context.Context, git UpdateGitRunner, checkoutDir string) bool {
	if git == nil {
		return false
	}
	worktree := git.RunGit(ctx, checkoutDir, "diff", "--quiet")
	index := git.RunGit(ctx, checkoutDir, "diff", "--cached", "--quiet")
	return worktree.Err != nil || index.Err != nil
}

var updateVersionPattern = regexp.MustCompile(`(?m)var\s+Version\s*=\s*"([^"]+)"`)

func updateBuildVersion(checkoutDir string) string {
	body, err := os.ReadFile(filepath.Join(checkoutDir, "cmd", "gormes", "version.go"))
	if err != nil {
		return "0.0.0"
	}
	match := updateVersionPattern.FindSubmatch(body)
	if len(match) < 2 {
		return "0.0.0"
	}
	return string(match[1])
}

func publishBinaryPath(ctx context.Context, runner UpdateCommandRunner, source, target string, allowSymlink bool, label string) UpdateReport {
	report := UpdateReport{}
	source = strings.TrimSpace(source)
	target = strings.TrimSpace(target)
	if source == "" || target == "" {
		report.Failed = true
		report.add(UpdateEvidencePublishFailed, fmt.Sprintf("%s: missing source or target", label))
		return report
	}
	if samePath(source, target) {
		_ = os.Chmod(target, 0o755)
		if verify := runner.RunCommand(ctx, "", nil, target, "version"); verify.Err != nil {
			report.Failed = true
			report.add(UpdateEvidenceVerifyFailed, fmt.Sprintf("%s: %s", label, commandFailureDetail(verify)))
			return report
		}
		report.add(UpdateEvidencePublishCompleted, fmt.Sprintf("%s ready at %s", label, target))
		return report
	}
	if info, err := os.Stat(target); err == nil && info.IsDir() {
		report.Failed = true
		report.add(UpdateEvidencePublishFailed, fmt.Sprintf("%s: cannot replace directory %s", label, target))
		return report
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		report.Failed = true
		report.add(UpdateEvidencePublishFailed, fmt.Sprintf("%s: %v", label, err))
		return report
	}

	backup := fmt.Sprintf("%s.rollback.%d", target, os.Getpid())
	tmp := fmt.Sprintf("%s.tmp.%d", target, os.Getpid())
	_ = os.Remove(tmp)
	_ = os.Remove(backup)
	existed := false
	if _, err := os.Lstat(target); err == nil {
		existed = true
		if err := os.Rename(target, backup); err != nil {
			report.Failed = true
			report.add(UpdateEvidencePublishFailed, fmt.Sprintf("%s: prepare rollback: %v", label, err))
			return report
		}
	}
	if err := preparePublishedBinary(source, tmp, allowSymlink); err != nil {
		restorePublishRollback(&report, backup, target, existed)
		report.Failed = true
		report.add(UpdateEvidencePublishFailed, fmt.Sprintf("%s: %v", label, err))
		return report
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		restorePublishRollback(&report, backup, target, existed)
		report.Failed = true
		report.add(UpdateEvidencePublishFailed, fmt.Sprintf("%s: %v", label, err))
		return report
	}
	if verify := runner.RunCommand(ctx, "", nil, target, "version"); verify.Err != nil {
		restorePublishRollback(&report, backup, target, existed)
		report.Failed = true
		report.add(UpdateEvidenceVerifyFailed, fmt.Sprintf("%s: %s", label, commandFailureDetail(verify)))
		return report
	}
	if existed {
		_ = os.Remove(backup)
	}
	report.add(UpdateEvidencePublishCompleted, fmt.Sprintf("%s ready at %s", label, target))
	return report
}

func preparePublishedBinary(source, target string, allowSymlink bool) error {
	if allowSymlink {
		if err := os.Symlink(source, target); err == nil {
			return nil
		}
	}
	if err := copyFile(source, target); err != nil {
		return err
	}
	return os.Chmod(target, 0o755)
}

func restorePublishRollback(report *UpdateReport, backup, target string, existed bool) {
	_ = os.Remove(target)
	if !existed {
		report.add(UpdateEvidencePublishRollbackCompleted, fmt.Sprintf("removed failed publish target %s", target))
		return
	}
	if err := os.Rename(backup, target); err != nil {
		report.add(UpdateEvidencePublishRollbackFailed, err.Error())
		return
	}
	report.add(UpdateEvidencePublishRollbackCompleted, fmt.Sprintf("restored previous %s", target))
}

func copyFile(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func samePath(a, b string) bool {
	if strings.TrimSpace(a) == "" || strings.TrimSpace(b) == "" {
		return false
	}
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	if errA == nil && errB == nil && aa == bb {
		return true
	}
	ea, errA := filepath.EvalSymlinks(a)
	eb, errB := filepath.EvalSymlinks(b)
	return errA == nil && errB == nil && ea == eb
}

func sameBinary(a, b string) bool {
	if samePath(a, b) {
		return true
	}
	infoA, errA := os.Stat(a)
	infoB, errB := os.Stat(b)
	if errA != nil || errB != nil || infoA.IsDir() || infoB.IsDir() {
		return false
	}
	if os.SameFile(infoA, infoB) {
		return true
	}
	sumA, errA := fileSHA256(a)
	sumB, errB := fileSHA256(b)
	return errA == nil && errB == nil && sumA == sumB
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func commandFailureDetail(result UpdateCommandResult) string {
	parts := []string{}
	if result.Err != nil {
		parts = append(parts, result.Err.Error())
	}
	if stderr := strings.TrimSpace(result.Stderr); stderr != "" {
		parts = append(parts, stderr)
	} else if stdout := strings.TrimSpace(result.Stdout); stdout != "" {
		parts = append(parts, stdout)
	}
	if len(parts) == 0 {
		return "command failed"
	}
	return strings.Join(parts, ": ")
}

func summarizeUpdateEvidence(evidence []UpdateEvidence) string {
	if len(evidence) == 0 {
		return "active PATH refresh failed"
	}
	parts := make([]string, 0, len(evidence))
	for _, ev := range evidence {
		detail := strings.TrimSpace(ev.Detail)
		if detail == "" {
			parts = append(parts, string(ev.Kind))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s: %s", ev.Kind, detail))
	}
	return strings.Join(parts, "; ")
}
