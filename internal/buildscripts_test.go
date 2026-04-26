package internal_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/repoctl"
)

func TestMain(m *testing.M) {
	for _, entry := range legacyCompanionTestEnvOverrides() {
		key, value, _ := strings.Cut(entry, "=")
		if err := os.Setenv(key, value); err != nil {
			panic(err)
		}
	}
	os.Exit(m.Run())
}

func TestLegacyOrchestratorTestProcessDisablesInheritedCompanions(t *testing.T) {
	for _, want := range legacyCompanionTestEnvOverrides() {
		key, value, _ := strings.Cut(want, "=")
		if got := os.Getenv(key); got != value {
			t.Fatalf("%s = %q, want %q", key, got, value)
		}
	}
}

func TestAutoCodexuOrchestratorScriptExistsAndIsExecutable(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	repoRoot := filepath.Dir(filepath.Dir(file))
	scriptPath := filepath.Join(repoRoot, "scripts", "gormes-auto-codexu-orchestrator.sh")

	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("stat %s: %v", scriptPath, err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("%s mode = %v, want executable", scriptPath, info.Mode())
	}
}

func TestLegacyOrchestratorTestEnvDisablesInheritedCompanions(t *testing.T) {
	base := []string{
		"PATH=/bin",
		"DISABLE_COMPANIONS=0",
		"COMPANION_ON_IDLE=0",
		"COMPANION_DOC_IMPROVER_CMD=/real/documentation-improver.sh",
	}

	env := legacyOrchestratorTestEnv(base, "/repo", "/tmp/repo", "/tmp/repo/bin",
		"LOOP_SLEEP_SECONDS=5",
	)

	for _, key := range []string{"DISABLE_COMPANIONS", "COMPANION_ON_IDLE", "COMPANION_DOC_IMPROVER_CMD"} {
		if got := countEnvKey(env, key); got != 1 {
			t.Fatalf("%s count = %d in %#v, want exactly one override", key, got, env)
		}
	}
	for _, want := range []string{
		"DISABLE_COMPANIONS=1",
		"COMPANION_ON_IDLE=1",
		"COMPANION_DOC_IMPROVER_CMD=:",
		"LOOP_SLEEP_SECONDS=5",
	} {
		if !envContains(env, want) {
			t.Fatalf("env missing %q in %#v", want, env)
		}
	}
}

func TestAutoCodexuOrchestratorLoopsByDefaultWhenBacklogEmpty(t *testing.T) {
	if _, err := exec.LookPath("timeout"); err != nil {
		t.Skip("timeout command not available")
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	repoRoot := filepath.Dir(filepath.Dir(file))
	tmpRepo := t.TempDir()

	copyFile(t,
		legacyAutoCodexuOrchestratorPath(repoRoot),
		filepath.Join(tmpRepo, "scripts", "gormes-auto-codexu-orchestrator.sh"),
		0o755,
	)
	writeFile(t,
		filepath.Join(tmpRepo, "docs", "content", "building-gormes", "architecture_plan", "progress.json"),
		[]byte(`{"phases":{"1":{"subphases":{"1.A":{"items":[{"name":"Done","status":"complete"}]}}}}}`),
		0o644,
	)

	binDir := filepath.Join(tmpRepo, "bin")
	writeFile(t, filepath.Join(binDir, "codexu"), []byte("#!/usr/bin/env bash\necho codexu should not run >&2\nexit 99\n"), 0o755)
	writeFile(t, filepath.Join(binDir, "free"), []byte("#!/usr/bin/env bash\ncat <<'EOF'\n              total        used        free      shared  buff/cache   available\nMem:          32000        1000       30000          0        1000       30000\nEOF\n"), 0o755)

	runCommand(t, tmpRepo, "git", "init")
	runCommand(t, tmpRepo, "git", "config", "user.name", "Test User")
	runCommand(t, tmpRepo, "git", "config", "user.email", "test@example.com")
	runCommand(t, tmpRepo, "git", "add", ".")
	runCommand(t, tmpRepo, "git", "commit", "-m", "init")

	cmd := exec.Command("timeout", "1s", "bash", "scripts/gormes-auto-codexu-orchestrator.sh")
	cmd.Dir = tmpRepo
	cmd.Env = legacyOrchestratorTestEnv(os.Environ(), repoRoot, tmpRepo, binDir,
		"LOOP_SLEEP_SECONDS=5",
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("orchestrator exited without timeout; want default forever loop\noutput:\n%s", string(out))
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("orchestrator failed with %T, want timeout exit\noutput:\n%s", err, string(out))
	}
	if exitErr.ExitCode() != 124 {
		t.Fatalf("exit = %d, want timeout exit 124\noutput:\n%s", exitErr.ExitCode(), string(out))
	}
}

func TestAutoCodexuOrchestratorReusesExistingIntegrationWorktree(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	repoRoot := filepath.Dir(filepath.Dir(file))
	tmpRepo := t.TempDir()

	copyFile(t,
		legacyAutoCodexuOrchestratorPath(repoRoot),
		filepath.Join(tmpRepo, "scripts", "gormes-auto-codexu-orchestrator.sh"),
		0o755,
	)
	writeFile(t,
		filepath.Join(tmpRepo, "docs", "content", "building-gormes", "architecture_plan", "progress.json"),
		[]byte(`{"phases":{"1":{"subphases":{"1.A":{"items":[{"name":"Done","status":"complete"}]}}}}}`),
		0o644,
	)

	binDir := filepath.Join(tmpRepo, "bin")
	writeFile(t, filepath.Join(binDir, "codexu"), []byte("#!/usr/bin/env bash\necho codexu should not run >&2\nexit 99\n"), 0o755)
	writeFile(t, filepath.Join(binDir, "free"), []byte("#!/usr/bin/env bash\ncat <<'EOF'\n              total        used        free      shared  buff/cache   available\nMem:          32000        1000       30000          0        1000       30000\nEOF\n"), 0o755)

	runCommand(t, tmpRepo, "git", "init")
	runCommand(t, tmpRepo, "git", "config", "user.name", "Test User")
	runCommand(t, tmpRepo, "git", "config", "user.email", "test@example.com")
	runCommand(t, tmpRepo, "git", "add", ".")
	runCommand(t, tmpRepo, "git", "commit", "-m", "init")
	runCommand(t, tmpRepo, "git", "branch", "codexu/autoloop")
	runCommand(t, tmpRepo, "git", "worktree", "add", filepath.Join(tmpRepo, ".codex", "orchestrator", "integration", "codexu-autoloop"), "codexu/autoloop")

	cmd := exec.Command("bash", "scripts/gormes-auto-codexu-orchestrator.sh")
	cmd.Dir = tmpRepo
	cmd.Env = legacyOrchestratorTestEnv(os.Environ(), repoRoot, tmpRepo, binDir,
		"ORCHESTRATOR_ONCE=1",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("orchestrator failed with existing integration worktree: %v\noutput:\n%s", err, string(out))
	}
	if !strings.Contains(string(out), "No unfinished tasks") {
		t.Fatalf("output missing empty-backlog message:\n%s", string(out))
	}
}

func TestAutoCodexuOrchestratorDoesNotSigpipeExistingIntegrationWorktreeLookup(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	repoRoot := filepath.Dir(filepath.Dir(file))
	tmpRepo := t.TempDir()

	copyFile(t,
		legacyAutoCodexuOrchestratorPath(repoRoot),
		filepath.Join(tmpRepo, "scripts", "gormes-auto-codexu-orchestrator.sh"),
		0o755,
	)
	writeFile(t,
		filepath.Join(tmpRepo, "docs", "content", "building-gormes", "architecture_plan", "progress.json"),
		[]byte(`{"phases":{"1":{"subphases":{"1.A":{"items":[{"name":"Done","status":"complete"}]}}}}}`),
		0o644,
	)

	binDir := filepath.Join(tmpRepo, "bin")
	writeFile(t, filepath.Join(binDir, "codexu"), []byte("#!/usr/bin/env bash\necho codexu should not run >&2\nexit 99\n"), 0o755)
	writeFile(t, filepath.Join(binDir, "free"), []byte("#!/usr/bin/env bash\ncat <<'EOF'\n              total        used        free      shared  buff/cache   available\nMem:          32000        1000       30000          0        1000       30000\nEOF\n"), 0o755)
	writeFile(t, filepath.Join(binDir, "git"), []byte(`#!/usr/bin/env bash
set -Eeuo pipefail
repo="${TEST_REPO_ROOT:?}"
while [[ "${1:-}" == "-C" ]]; do
  shift 2
done
case "$*" in
  "rev-parse --show-toplevel") printf '%s\n' "$repo" ;;
  "rev-parse HEAD") printf '%s\n' "0123456789abcdef0123456789abcdef01234567" ;;
  "show-ref --verify --quiet refs/heads/codexu/autoloop") exit 0 ;;
  "worktree list --porcelain")
    printf 'worktree %s\n' "$repo"
    printf 'HEAD 0123456789abcdef0123456789abcdef01234567\n'
    printf 'branch refs/heads/codexu/autoloop\n'
    for i in $(seq 1 20000); do
      printf 'worktree %s/.codex/filler/%05d\n' "$repo" "$i"
      printf 'HEAD 0123456789abcdef0123456789abcdef01234567\n'
      printf 'branch refs/heads/filler-%05d\n' "$i"
    done
    ;;
  "status --short") exit 0 ;;
  "reset --hard codexu/autoloop") exit 0 ;;
  *) echo "unexpected fake git invocation: $*" >&2; exit 1 ;;
esac
`), 0o755)

	cmd := exec.Command("bash", "scripts/gormes-auto-codexu-orchestrator.sh")
	cmd.Dir = tmpRepo
	cmd.Env = legacyOrchestratorTestEnv(os.Environ(), repoRoot, tmpRepo, binDir,
		"TEST_REPO_ROOT="+tmpRepo,
		"ORCHESTRATOR_ONCE=1",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("orchestrator failed after existing worktree lookup: %v\noutput:\n%s", err, string(out))
	}
	if !strings.Contains(string(out), "No unfinished tasks") {
		t.Fatalf("output missing empty-backlog message:\n%s", string(out))
	}
}

func TestAutoCodexuOrchestratorPromotesSuccessBeforeNextCycle(t *testing.T) {
	if _, err := exec.LookPath("timeout"); err != nil {
		t.Skip("timeout command not available")
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	repoRoot := filepath.Dir(filepath.Dir(file))
	tmpRepo := t.TempDir()
	progressRel := filepath.Join("docs", "content", "building-gormes", "architecture_plan", "progress.json")

	copyFile(t,
		legacyAutoCodexuOrchestratorPath(repoRoot),
		filepath.Join(tmpRepo, "scripts", "gormes-auto-codexu-orchestrator.sh"),
		0o755,
	)
	writeFile(t,
		filepath.Join(tmpRepo, progressRel),
		[]byte(`{"phases":{"1":{"name":"Phase 1","subphases":{"1.A":{"name":"Alpha","items":[{"name":"Loop proof task","status":"planned"}]}}}}}`),
		0o644,
	)

	binDir := filepath.Join(tmpRepo, "bin")
	writeFile(t, filepath.Join(binDir, "free"), []byte("#!/usr/bin/env bash\ncat <<'EOF'\n              total        used        free      shared  buff/cache   available\nMem:          32000        1000       30000          0        1000       30000\nEOF\n"), 0o755)
	writeFile(t, filepath.Join(binDir, "codexu"), []byte(`#!/usr/bin/env bash
set -Eeuo pipefail

final_file=""
while (($#)); do
  case "$1" in
    --output-last-message)
      final_file="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

tmp="$(mktemp)"
jq '(.phases["1"].subphases["1.A"].items[0].status)="complete"' docs/content/building-gormes/architecture_plan/progress.json > "$tmp"
mv "$tmp" docs/content/building-gormes/architecture_plan/progress.json
mkdir -p fake-worker
branch="$(git rev-parse --abbrev-ref HEAD)"
printf '%s\n' "$branch" > "fake-worker/${branch//\//_}.txt"
git add docs/content/building-gormes/architecture_plan/progress.json fake-worker
git commit -m "test: complete loop proof task" >/dev/null
commit="$(git rev-parse HEAD)"
cat > "$final_file" <<EOF
1) Selected task
Task: 1 / 1.A / Loop proof task

2) Pre-doc baseline
Files:
- docs/content/building-gormes/architecture_plan/progress.json

3) RED proof
Command: go test ./fake -run TestLoopProof
Exit: 1
Snippet: missing behavior

4) GREEN proof
Command: go test ./fake -run TestLoopProof
Exit: 0
Snippet: ok

5) REFACTOR proof
Command: go test ./fake -run TestLoopProof
Exit: 0
Snippet: ok

6) Regression proof
Command: go test ./fake
Exit: 0
Snippet: ok

7) Post-doc closeout
Files:
- docs/content/building-gormes/architecture_plan/progress.json

8) Commit
Branch: $branch
Commit: $commit
Files:
- docs/content/building-gormes/architecture_plan/progress.json

9) Acceptance check
Criterion: Loop proof task selected once — PASS
Criterion: progress.json status promoted to complete — PASS
Criterion: worker commit recorded in final report — PASS
EOF
`), 0o755)

	runCommand(t, tmpRepo, "git", "init")
	runCommand(t, tmpRepo, "git", "config", "user.name", "Test User")
	runCommand(t, tmpRepo, "git", "config", "user.email", "test@example.com")
	runCommand(t, tmpRepo, "git", "add", ".")
	runCommand(t, tmpRepo, "git", "commit", "-m", "init")

	cmd := exec.Command("timeout", "4s", "bash", "scripts/gormes-auto-codexu-orchestrator.sh")
	cmd.Dir = tmpRepo
	cmd.Env = legacyOrchestratorTestEnv(os.Environ(), repoRoot, tmpRepo, binDir,
		"INTEGRATION_BRANCH=codexu/test-integration",
		"MAX_AGENTS=1",
		"HEARTBEAT_SECONDS=1",
		"LOOP_SLEEP_SECONDS=2",
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("orchestrator exited without timeout; want forever loop\noutput:\n%s", string(out))
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("orchestrator failed with %T, want timeout exit\noutput:\n%s", err, string(out))
	}
	if exitErr.ExitCode() != 124 {
		t.Fatalf("exit = %d, want timeout exit 124\noutput:\n%s", exitErr.ExitCode(), string(out))
	}
	if got := strings.Count(string(out), "claimed 1 / 1.A / Loop proof task"); got != 1 {
		t.Fatalf("task claim count = %d, want exactly one claim before promotion removes it\noutput:\n%s", got, string(out))
	}

	promoted := runCommand(t, tmpRepo, "git", "show", "codexu/test-integration:"+filepath.ToSlash(progressRel))
	if !strings.Contains(string(promoted), `"status": "complete"`) && !strings.Contains(string(promoted), `"status":"complete"`) {
		t.Fatalf("integration branch did not contain promoted complete status:\n%s", string(promoted))
	}
}

func TestAutoCodexuOrchestratorAcceptsNonZeroCodexExitWithValidCommitAndReport(t *testing.T) {
	if _, err := exec.LookPath("timeout"); err != nil {
		t.Skip("timeout command not available")
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	repoRoot := filepath.Dir(filepath.Dir(file))
	tmpRepo := t.TempDir()
	progressRel := filepath.Join("docs", "content", "building-gormes", "architecture_plan", "progress.json")

	copyFile(t,
		legacyAutoCodexuOrchestratorPath(repoRoot),
		filepath.Join(tmpRepo, "scripts", "gormes-auto-codexu-orchestrator.sh"),
		0o755,
	)
	writeFile(t,
		filepath.Join(tmpRepo, progressRel),
		[]byte(`{"phases":{"1":{"name":"Phase 1","subphases":{"1.A":{"name":"Alpha","items":[{"name":"Soft success task","status":"planned"}]}}}}}`),
		0o644,
	)

	binDir := filepath.Join(tmpRepo, "bin")
	writeFile(t, filepath.Join(binDir, "free"), []byte("#!/usr/bin/env bash\ncat <<'EOF'\n              total        used        free      shared  buff/cache   available\nMem:          32000        1000       30000          0        1000       30000\nEOF\n"), 0o755)
	writeFile(t, filepath.Join(binDir, "codexu"), []byte(`#!/usr/bin/env bash
set -Eeuo pipefail

final_file=""
while (($#)); do
  case "$1" in
    --output-last-message)
      final_file="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

tmp="$(mktemp)"
jq '(.phases["1"].subphases["1.A"].items[0].status)="complete"' docs/content/building-gormes/architecture_plan/progress.json > "$tmp"
mv "$tmp" docs/content/building-gormes/architecture_plan/progress.json
branch="$(git rev-parse --abbrev-ref HEAD)"
git add docs/content/building-gormes/architecture_plan/progress.json
git commit -m "test: complete soft success task" >/dev/null
commit="$(git rev-parse HEAD)"
cat > "$final_file" <<EOF
1) Selected task
Task: 1 / 1.A / Soft success task

2) Pre-doc baseline
Files:
- docs/content/building-gormes/architecture_plan/progress.json

3) RED proof
Command: go test ./fake -run TestSoft
Exit: 1
Snippet: expected missing behavior

4) GREEN proof
Command: go test ./fake -run TestSoft
Exit: 0
Snippet: ok

5) REFACTOR proof
Command: go test ./fake -run TestSoft
Exit: 0
Snippet: ok

6) Regression proof
Command: go test ./fake
Exit: 0
Snippet: ok

7) Post-doc closeout
Files:
- docs/content/building-gormes/architecture_plan/progress.json

8) Commit
Branch: $branch
Commit: $commit
Files:
- docs/content/building-gormes/architecture_plan/progress.json

9) Acceptance check
Criterion: Soft success task selected once — PASS
Criterion: progress.json status promoted to complete — PASS
Criterion: non-zero codex exit accepted with valid report — PASS
EOF
# Simulate codex non-zero despite valid commit/report.
exit 1
`), 0o755)

	runCommand(t, tmpRepo, "git", "init")
	runCommand(t, tmpRepo, "git", "config", "user.name", "Test User")
	runCommand(t, tmpRepo, "git", "config", "user.email", "test@example.com")
	runCommand(t, tmpRepo, "git", "add", ".")
	runCommand(t, tmpRepo, "git", "commit", "-m", "init")

	cmd := exec.Command("timeout", "4s", "bash", "scripts/gormes-auto-codexu-orchestrator.sh")
	cmd.Dir = tmpRepo
	cmd.Env = legacyOrchestratorTestEnv(os.Environ(), repoRoot, tmpRepo, binDir,
		"INTEGRATION_BRANCH=codexu/test-soft-success",
		"MAX_AGENTS=1",
		"HEARTBEAT_SECONDS=1",
		"LOOP_SLEEP_SECONDS=30",
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("orchestrator exited without timeout; want forever loop\noutput:\n%s", string(out))
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("orchestrator failed with %T, want timeout exit\noutput:\n%s", err, string(out))
	}
	if exitErr.ExitCode() != 124 {
		t.Fatalf("exit = %d, want timeout exit 124\noutput:\n%s", exitErr.ExitCode(), string(out))
	}

	promoted := runCommand(t, tmpRepo, "git", "show", "codexu/test-soft-success:"+filepath.ToSlash(progressRel))
	if !strings.Contains(string(promoted), `"status": "complete"`) && !strings.Contains(string(promoted), `"status":"complete"`) {
		t.Fatalf("integration branch did not contain promoted complete status for soft-success run:\n%s\noutput:\n%s", string(promoted), string(out))
	}
}

func TestRecordBenchmarkHandlesArchPlanStub(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t,
		filepath.Join(tmpRepo, "docs", "ARCH_PLAN.md"),
		[]byte("# Stub architecture plan\n\nNo current phase marker here.\n"),
		0o644,
	)
	writeFile(t,
		filepath.Join(tmpRepo, "docs", "content", "building-gormes", "architecture_plan", "progress.json"),
		[]byte("{\n  \"phases\": {\n    \"1\": {\n      \"name\": \"Phase 1 — The Dashboard\",\n      \"subphases\": {\n        \"1.A\": {\n          \"items\": [\n            {\"name\": \"Core TUI\", \"status\": \"complete\"}\n          ]\n        }\n      }\n    },\n    \"2\": {\n      \"name\": \"Phase 2 — The Gateway\",\n      \"subphases\": {\n        \"2.E\": {\n          \"items\": [\n            {\"name\": \"Execution isolation\", \"status\": \"planned\"}\n          ]\n        }\n      }\n    }\n  }\n}\n"),
		0o644,
	)

	if err := os.MkdirAll(filepath.Join(tmpRepo, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpRepo, "bin", "gormes"), []byte("fake-binary"), 0o755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	benchmarks := []byte("{\n  \"binary\": {},\n  \"history\": [\n    {\n      \"date\": \"" + yesterday + "\",\n      \"size_mb\": 1.0,\n      \"phase\": \"Phase 1\"\n    }\n  ]\n}\n")
	if err := os.WriteFile(filepath.Join(tmpRepo, "benchmarks.json"), benchmarks, 0o644); err != nil {
		t.Fatalf("write benchmarks.json: %v", err)
	}

	runCommand(t, tmpRepo, "git", "init")
	runCommand(t, tmpRepo, "git", "config", "user.name", "Test User")
	runCommand(t, tmpRepo, "git", "config", "user.email", "test@example.com")
	runCommand(t, tmpRepo, "git", "add", ".")
	runCommand(t, tmpRepo, "git", "commit", "-m", "init")

	if err := repoctl.RecordBenchmark(repoctl.BenchmarkOptions{Root: tmpRepo}); err != nil {
		t.Fatalf("RecordBenchmark: %v", err)
	}

	var got struct {
		Binary struct {
			LastMeasured string `json:"last_measured"`
			SizeBytes    int64  `json:"size_bytes"`
		} `json:"binary"`
		History []struct {
			Date   string `json:"date"`
			Phase  string `json:"phase"`
			Commit string `json:"commit"`
		} `json:"history"`
	}

	raw, err := os.ReadFile(filepath.Join(tmpRepo, "benchmarks.json"))
	if err != nil {
		t.Fatalf("read benchmarks.json: %v", err)
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal benchmarks.json: %v", err)
	}

	today := time.Now().Format("2006-01-02")
	if got.Binary.LastMeasured != today {
		t.Fatalf("binary.last_measured = %q, want %q", got.Binary.LastMeasured, today)
	}
	if got.Binary.SizeBytes == 0 {
		t.Fatal("binary.size_bytes = 0, want non-zero")
	}
	if len(got.History) == 0 {
		t.Fatal("history is empty, want new entry")
	}
	if got.History[0].Date != today {
		t.Fatalf("history[0].date = %q, want %q", got.History[0].Date, today)
	}
	if got.History[0].Phase != "Phase 2 — The Gateway" {
		t.Fatalf("history[0].phase = %q, want %q", got.History[0].Phase, "Phase 2 — The Gateway")
	}
	if got.History[0].Commit == "" {
		t.Fatal("history[0].commit is empty, want git commit")
	}
}

func copyFile(t *testing.T, src, dst string, mode os.FileMode) {
	t.Helper()

	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(dst), err)
	}
	if err := os.WriteFile(dst, data, mode); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}

func legacyAutoCodexuOrchestratorPath(repoRoot string) string {
	return filepath.Join(repoRoot, "testdata", "legacy-shell", "scripts", "gormes-auto-codexu-orchestrator.sh")
}

func legacyOrchestratorLibDir(repoRoot string) string {
	return filepath.Join(repoRoot, "testdata", "legacy-shell", "scripts", "orchestrator", "lib")
}

func legacyCompanionTestEnvOverrides() []string {
	return []string{
		"DISABLE_COMPANIONS=1",
		"COMPANION_ON_IDLE=1",
		"COMPANION_PLANNER_CMD=:",
		"COMPANION_DOC_IMPROVER_CMD=:",
		"COMPANION_LANDINGPAGE_CMD=:",
	}
}

func legacyOrchestratorTestEnv(base []string, repoRoot, tmpRepo, binDir string, extra ...string) []string {
	overrides := []string{
		"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"REPO_ROOT=" + tmpRepo,
		"RUN_ROOT=" + filepath.Join(tmpRepo, ".codex", "orchestrator"),
		"ORCHESTRATOR_LIB_DIR=" + legacyOrchestratorLibDir(repoRoot),
	}
	overrides = append(overrides, legacyCompanionTestEnvOverrides()...)
	overrides = append(overrides, extra...)
	return overlayEnv(base,
		overrides...,
	)
}

func overlayEnv(base []string, overrides ...string) []string {
	keys := make(map[string]bool, len(overrides))
	for _, entry := range overrides {
		if key, _, ok := strings.Cut(entry, "="); ok {
			keys[key] = true
		}
	}

	env := make([]string, 0, len(base)+len(overrides))
	for _, entry := range base {
		key, _, ok := strings.Cut(entry, "=")
		if ok && keys[key] {
			continue
		}
		env = append(env, entry)
	}
	return append(env, overrides...)
}

func countEnvKey(env []string, key string) int {
	prefix := key + "="
	var count int
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			count++
		}
	}
	return count
}

func envContains(env []string, want string) bool {
	for _, entry := range env {
		if entry == want {
			return true
		}
	}
	return false
}

func writeFile(t *testing.T, dst string, data []byte, mode os.FileMode) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(dst), err)
	}
	if err := os.WriteFile(dst, data, mode); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}

func runCommand(t *testing.T, dir string, name string, args ...string) []byte {
	t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\noutput:\n%s", name, args, err, string(out))
	}
	return out
}
