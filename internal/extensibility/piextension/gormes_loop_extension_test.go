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
	jitiEntry := piJitiEntry(t)

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
		`fs.writeFileSync(logPath, JSON.stringify({ at: "2026-05-21T14:43:51.837Z", event: "iteration_result", iteration: 2, decision: "continue", validation: ["go test ./internal/tui -count=1", "go test ./... -count=1", "go run ./cmd/progress validate", "git diff --check"] }) + "\n");`,
		`const gate = mod.__test__.parseCIGate("LOOP_DECISION: continue", { logPath, startedAt, iteration: 2 });`,
		`if (!gate.green || gate.source !== "loop_log_full_gate") {`,
		`  throw new Error("expected loop_log_full_gate, got " + JSON.stringify(gate));`,
		`}`,
	}, "\n"))

	runNodeScript(t, script)
}

func TestGormesLoopExtensionAcceptsCompletedPreviousIterationLogSchema(t *testing.T) {
	repoRoot := repoRootFromTest(t)
	jitiEntry := piJitiEntry(t)

	script := filepath.Join(t.TempDir(), "smoke-previous-iteration-ci-gate.mjs")
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
		`fs.writeFileSync(logPath, JSON.stringify({ timestamp: "2026-05-21T15:36:00.661188+00:00", type: "iteration_result", iteration: "4/10", loop_decision: "continue", ci_gate: "local_full_gate_passed", validation: ["go test ./... -count=1", "go run ./cmd/progress validate", "git diff --check"] }) + "\n");`,
		`const text = "Delivery loop iteration 4/10 complete.\\nCI_GREEN: omitted by mistake\\nLOOP_DECISION: continue".replace("CI_GREEN: omitted by mistake\\n", "");`,
		`const gate = mod.__test__.parseCIGate(text, { logPath, startedAt: "2026-05-21T15:00:00.000Z", iteration: 5 });`,
		`if (!gate.green || gate.source !== "loop_log_full_gate") {`,
		`  throw new Error("expected loop_log_full_gate from completed previous iteration schema, got " + JSON.stringify(gate));`,
		`}`,
	}, "\n"))

	runNodeScript(t, script)
}

func TestGormesLoopStatusReportExplainsQueuedFollowUpState(t *testing.T) {
	repoRoot := repoRootFromTest(t)
	jitiEntry := piJitiEntry(t)

	script := filepath.Join(t.TempDir(), "smoke-status-queued-state.mjs")
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
		`fs.writeFileSync(logPath, JSON.stringify({ at: "2026-05-21T17:47:09.382Z", event: "iteration_queued", topic: "providers", iteration: 5, maxIterations: 10, logPath }) + "\n");`,
		`const report = mod.__test__.statusReport({ active: true, topic: "providers", iteration: 5, maxIterations: 10, startedAt: "2026-05-21T16:14:35.645Z", logPath, selfImproveQueued: false });`,
		`if (!report.includes("Queued follow-up")) throw new Error("status report does not explain queued state: " + report);`,
		`if (!report.includes("Last event: iteration_queued")) throw new Error("status report omits last event: " + report);`,
		`if (!report.includes("/gormes-loop status")) throw new Error("status report omits operator commands: " + report);`,
	}, "\n"))

	runNodeScript(t, script)
}

func TestGormesLoopStatusCommandSendsDisplayedMessage(t *testing.T) {
	repoRoot := repoRootFromTest(t)
	jitiEntry := piJitiEntry(t)

	script := filepath.Join(t.TempDir(), "smoke-status-command.mjs")
	mustWriteFile(t, script, strings.Join([]string{
		`import { createRequire } from "node:module";`,
		`const require = createRequire(import.meta.url);`,
		`const { createJiti } = require("` + jitiEntry + `");`,
		`const jiti = createJiti(import.meta.url, { interopDefault: true });`,
		`const mod = await jiti.import("` + filepath.ToSlash(filepath.Join(repoRoot, ".pi", "extensions", "gormes-delivery-loop.ts")) + `");`,
		`const handlers = new Map();`,
		`let statusCommand;`,
		`const sent = [];`,
		`const pi = {`,
		`  on(name, handler) { handlers.set(name, handler); },`,
		`  registerCommand(name, command) { if (name === "gormes-loop") statusCommand = command; },`,
		`  appendEntry() {},`,
		`  sendUserMessage() {},`,
		`  sendMessage(message) { sent.push(message); },`,
		`};`,
		`mod.default(pi);`,
		`await handlers.get("session_start")({}, { hasUI: true, ui: { setStatus() {}, notify() {} }, sessionManager: { getEntries() { return [{ type: "custom", customType: "gormes-delivery-loop-state", data: { active: true, topic: "auto-select", iteration: 3, maxIterations: 10, startedAt: "2026-05-21T15:00:00.000Z", logPath: ".pi/gormes-loop/logs.jsonl" } }]; } } });`,
		`await statusCommand.handler("status", { ui: { notify() {} } });`,
		`if (sent.length !== 1) throw new Error("expected one displayed status message, got " + sent.length);`,
		`if (sent[0].customType !== "gormes-loop-status" || sent[0].display !== true) throw new Error("unexpected status message envelope: " + JSON.stringify(sent[0]));`,
		`if (!String(sent[0].content).includes("Gormes loop: active 3/10 auto-select") || !String(sent[0].content).includes("log: .pi/gormes-loop/logs.jsonl")) {`,
		`  throw new Error("unexpected status content: " + sent[0].content);`,
		`}`,
	}, "\n"))

	runNodeScript(t, script)
}

func TestGormesLoopExtensionRejectsStaleLoggedFullCIGateEvidence(t *testing.T) {
	repoRoot := repoRootFromTest(t)
	jitiEntry := piJitiEntry(t)

	script := filepath.Join(t.TempDir(), "smoke-stale-ci-gate.mjs")
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
		`fs.writeFileSync(logPath, JSON.stringify({ at: "2026-05-21T14:43:51.837Z", event: "iteration_result", iteration: 1, decision: "continue", validation: ["go test ./... -count=1", "go run ./cmd/progress validate", "git diff --check"] }) + "\n");`,
		`const gate = mod.__test__.parseCIGate("LOOP_DECISION: continue", { logPath, startedAt, iteration: 2 });`,
		`if (gate.green || gate.source !== "missing") {`,
		`  throw new Error("expected missing gate for stale iteration evidence, got " + JSON.stringify(gate));`,
		`}`,
	}, "\n"))

	runNodeScript(t, script)
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
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func piJitiEntry(t *testing.T) string {
	t.Helper()
	jitiEntry := "/home/xel/.nvm/versions/node/v22.21.1/lib/node_modules/@earendil-works/pi-coding-agent/node_modules/jiti/lib/jiti.cjs"
	if _, err := os.Stat(jitiEntry); err != nil {
		t.Skipf("Pi-bundled jiti unavailable: %v", err)
	}
	return jitiEntry
}

func runNodeScript(t *testing.T, script string) {
	t.Helper()
	cmd := exec.Command("node", script)
	cmd.Dir = t.TempDir()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("extension smoke failed: %v\n%s", err, out)
	}
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
