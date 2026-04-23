package internal_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

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
		filepath.Join(repoRoot, "scripts", "gormes-auto-codexu-orchestrator.sh"),
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
	cmd.Env = append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"REPO_ROOT="+tmpRepo,
		"RUN_ROOT="+filepath.Join(tmpRepo, ".codex", "orchestrator"),
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

func TestRecordBenchmarkHandlesArchPlanStub(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	repoRoot := filepath.Dir(filepath.Dir(file))
	tmpRepo := t.TempDir()

	copyFile(t,
		filepath.Join(repoRoot, "scripts", "record-benchmark.sh"),
		filepath.Join(tmpRepo, "scripts", "record-benchmark.sh"),
		0o755,
	)
	copyFile(t,
		filepath.Join(repoRoot, "docs", "ARCH_PLAN.md"),
		filepath.Join(tmpRepo, "docs", "ARCH_PLAN.md"),
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

	out := runCommand(t, tmpRepo, "bash", "scripts/record-benchmark.sh")
	if len(out) == 0 {
		t.Fatal("record-benchmark.sh produced no output")
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
