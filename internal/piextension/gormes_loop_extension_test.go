package piextension

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGormesLoopPromptRequiresRecentLogsSkillsAndFullCIGate(t *testing.T) {
	content := readGormesLoopExtension(t)

	mustContainAll(t, content, []string{
		"recentLoopLogSnippet",
		"Recent loop log records",
		"gormes-skill-manager",
		"gormes-delivery-loop",
		"gormes-pi-parity",
		"gormes-tdd-slice",
		"gormes-tdd",
		"gormes-git",
		"caveman",
		"go test ./... -count=1",
		"go run ./cmd/progress validate",
		"git diff --check",
		"CI_GREEN",
	})
}

func TestGormesLoopActiveDefaultDoesNotAskToReplaceState(t *testing.T) {
	content := readGormesLoopExtension(t)

	mustContainAll(t, content, []string{
		`const values = ["start", "restart", "stop", "status"]`,
		`case "restart":`,
		`startLoop(pi, ctx, parsed.topic, parsed.iterations, true)`,
		"if (state.active && !replaceActive)",
		"Use /gormes-loop restart",
	})
}

func TestGormesLoopExtensionAcceptsLoggedFullCIGateEvidence(t *testing.T) {
	repoRoot := repoRootFromTest(t)
	jitiEntry := "/home/xel/.nvm/versions/node/v22.21.1/lib/node_modules/@earendil-works/pi-coding-agent/node_modules/jiti/lib/jiti.cjs"
	if _, err := os.Stat(jitiEntry); err != nil {
		t.Skipf("Pi-bundled jiti unavailable: %v", err)
	}

	script := filepath.Join(t.TempDir(), "smoke-ci-gate.mjs")
	mustWriteFile(t, script, strings.Join([]string{
		`import fs from "node:fs";`,
		`import path from "node:path";`,
		`import { createRequire } from "node:module";`,
		`const require = createRequire(import.meta.url);`,
		`const { createJiti } = require("` + jitiEntry + `");`,
		`const jiti = createJiti(import.meta.url, { interopDefault: true });`,
		`const mod = await jiti.import("` + filepath.ToSlash(filepath.Join(repoRoot, ".pi", "extensions", "gormes-delivery-loop.ts")) + `");`,
		`const logDir = path.join(process.cwd(), ".pi", "gormes-loop");`,
		`fs.mkdirSync(logDir, { recursive: true });`,
		`const logPath = path.join(logDir, "logs.jsonl");`,
		`const startedAt = "2026-05-21T14:40:00.000Z";`,
		`fs.writeFileSync(logPath, JSON.stringify({ at: "2026-05-21T14:43:51.837Z", event: "iteration_result", decision: "continue", validation: ["go test ./internal/tui -count=1", "go test ./... -count=1", "go run ./cmd/progress validate", "git diff --check"] }) + "\n");`,
		`const gate = mod.__test__.parseCIGate("LOOP_DECISION: continue", { logPath, startedAt });`,
		`if (!gate.green || gate.source !== "loop_log_full_gate") {`,
		`  throw new Error("expected loop_log_full_gate, got " + JSON.stringify(gate));`,
		`}`,
	}, "\n"))

	cmd := exec.Command("node", script)
	cmd.Dir = t.TempDir()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("extension smoke failed: %v\n%s", err, out)
	}
}

func readGormesLoopExtension(t *testing.T) string {
	t.Helper()
	repoRoot := repoRootFromTest(t)
	extensionPath := filepath.Join(repoRoot, ".pi", "extensions", "gormes-delivery-loop.ts")
	contentBytes, err := os.ReadFile(extensionPath)
	if err != nil {
		t.Fatalf("read extension: %v", err)
	}
	return string(contentBytes)
}

func repoRootFromTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustContainAll(t *testing.T, content string, needles []string) {
	t.Helper()
	for _, needle := range needles {
		if !strings.Contains(content, needle) {
			t.Fatalf("extension prompt contract missing %q", needle)
		}
	}
}
