import * as fs from "node:fs";
import * as path from "node:path";
import type { ExtensionAPI, ExtensionCommandContext } from "@earendil-works/pi-coding-agent";

type LoopState = {
  active: boolean;
  topic: string;
  iteration: number;
  maxIterations: number;
  startedAt: string;
  lastDecision?: string;
  logPath?: string;
  selfImproveQueued?: boolean;
};

type LoopLogEvent = {
  at: string;
  event: string;
  topic: string;
  iteration: number;
  maxIterations: number;
  decision?: string;
  reason?: string;
  logPath?: string;
  ciGreen?: boolean;
  ciGreenSource?: string;
};

const CUSTOM_STATE_TYPE = "gormes-delivery-loop-state";
const DEFAULT_TOPIC = "auto-select the highest-impact builder-ready progress.json row toward finishing full Hermes-in-Go parity";
const DEFAULT_ITERATIONS = 10;
const HARD_MAX_ITERATIONS = 50;
const DEFAULT_LOG_PATH = path.join(".pi", "gormes-loop", "logs.jsonl");
const RECENT_LOG_RECORDS = 12;
const REQUIRED_ITERATION_SKILLS = [
  "gormes-skill-manager",
  "gormes-delivery-loop",
  "gormes-pi-parity",
  "gormes-tdd-slice (gormes-tdd)",
  "gormes-git",
  "caveman",
];
const FULL_CI_GATE = [
  "go test ./... -count=1",
  "go run ./cmd/progress validate",
  "git diff --check",
];

let state: LoopState = inactiveState();
let sendingFollowUp = false;
let pendingSelfImproveReason: string | undefined;

export default function gormesDeliveryLoopExtension(pi: ExtensionAPI) {
  pi.on("session_start", async (_event, ctx) => {
    state = restoreState(ctx.sessionManager.getEntries()) ?? inactiveState();
    if (ctx.hasUI) {
      ctx.ui.setStatus("gormes-loop", statusLine(state));
    }
  });

  pi.on("message_end", async (event, _ctx) => {
    if (!state.active || event.message?.role !== "assistant") return;
    const text = messageText(event.message);
    const decision = parseDecision(text);
    const ciGate = parseCIGate(text, state);
    if (decision && requiresCIGreen(decision) && !ciGate.green) {
      state = { ...state, active: false, lastDecision: "blocked" };
      appendLoopLog("ci_gate_missing", { decision, reason: "missing_CI_GREEN_yes", ciGreen: false });
      pendingSelfImproveReason = "ci_gate_missing";
      pi.appendEntry(CUSTOM_STATE_TYPE, state);
      return;
    }
    if (decision) {
      state.lastDecision = decision;
      appendLoopLog("assistant_decision", { decision, ciGreen: ciGate.green, ciGreenSource: ciGate.source });
    }
    if (decision === "stop" || decision === "blocked" || decision === "done") {
      state = { ...state, active: false };
      if (decision !== "stop" && !state.selfImproveQueued) {
        pendingSelfImproveReason = decision;
      }
      pi.appendEntry(CUSTOM_STATE_TYPE, state);
    }
  });

  pi.on("agent_end", async (_event, ctx) => {
    if (pendingSelfImproveReason && !sendingFollowUp) {
      queueSelfImprovement(pi, ctx, pendingSelfImproveReason);
      pendingSelfImproveReason = undefined;
      return;
    }

    if (!state.active || sendingFollowUp) return;
    if (state.iteration >= state.maxIterations) {
      state = { ...state, active: false, lastDecision: "max_iterations_reached" };
      appendLoopLog("loop_finished", { reason: "max_iterations_reached" });
      pi.appendEntry(CUSTOM_STATE_TYPE, state);
      if (ctx.hasUI) {
        ctx.ui.notify(`Gormes loop stopped after ${state.iteration}/${state.maxIterations} iteration(s).`, "info");
        ctx.ui.setStatus("gormes-loop", statusLine(state));
      }
      queueSelfImprovement(pi, ctx, "max_iterations_reached");
      return;
    }

    state = { ...state, iteration: state.iteration + 1 };
    appendLoopLog("iteration_queued");
    pi.appendEntry(CUSTOM_STATE_TYPE, state);
    if (ctx.hasUI) ctx.ui.setStatus("gormes-loop", statusLine(state));

    sendingFollowUp = true;
    try {
      pi.sendUserMessage(buildIterationPrompt(state), { deliverAs: "followUp" });
    } finally {
      sendingFollowUp = false;
    }
  });

  pi.registerCommand("gormes-loop", {
    description: "Run a bounded Gormes architecture→planner→parity→builder TDD delivery loop",
    getArgumentCompletions: (prefix) => {
      const values = ["start", "restart", "stop", "status"];
      return values.filter((v) => v.startsWith(prefix)).map((value) => ({ value, label: value }));
    },
    handler: async (args, ctx) => {
      const parsed = parseArgs(args);
      switch (parsed.command) {
        case "stop":
          state = { ...state, active: false, lastDecision: "stopped_by_user" };
          appendLoopLog("loop_stopped", { reason: "stopped_by_user" });
          pi.appendEntry(CUSTOM_STATE_TYPE, state);
          ctx.ui.notify("Gormes delivery loop stopped.", "info");
          ctx.ui.setStatus("gormes-loop", statusLine(state));
          return;
        case "status":
          publishStatus(pi, ctx);
          return;
        case "restart":
          await startLoop(pi, ctx, parsed.topic, parsed.iterations, true);
          return;
        case "start":
        default:
          await startLoop(pi, ctx, parsed.topic, parsed.iterations, false);
          return;
      }
    },
  });
}

function publishStatus(pi: ExtensionAPI, ctx: { ui?: { notify(message: string, level?: string): void } }) {
  const text = statusReport(state);
  ctx.ui?.notify(text, "info");
  pi.sendMessage({
    customType: "gormes-loop-status",
    content: text,
    display: true,
  });
}

function statusReport(s: LoopState): string {
  return `${statusLine(s)}\nlog: ${s.logPath ?? DEFAULT_LOG_PATH}`;
}

async function startLoop(pi: ExtensionAPI, ctx: ExtensionCommandContext, topic: string, requestedIterations: number, replaceActive: boolean) {
  const maxIterations = Math.max(1, Math.min(requestedIterations, HARD_MAX_ITERATIONS));
  if (requestedIterations > HARD_MAX_ITERATIONS) {
    ctx.ui.notify(`Capped Gormes loop at ${HARD_MAX_ITERATIONS} iterations.`, "warning");
  }

  if (state.active && !replaceActive) {
    ctx.ui.notify(`${statusLine(state)}\nUse /gormes-loop restart to replace it; /gormes-loop status to inspect; /gormes-loop stop to stop.`, "info");
    ctx.ui.setStatus("gormes-loop", statusLine(state));
    return;
  }

  if (state.active) {
    const ok = await ctx.ui.confirm("Restart Gormes delivery loop", "Replace the current active loop state?");
    if (!ok) return;
  }

  state = {
    active: true,
    topic: topic || DEFAULT_TOPIC,
    iteration: 1,
    maxIterations,
    startedAt: new Date().toISOString(),
    logPath: DEFAULT_LOG_PATH,
    selfImproveQueued: false,
  };
  appendLoopLog("loop_started");
  pi.appendEntry(CUSTOM_STATE_TYPE, state);
  ctx.ui.setStatus("gormes-loop", statusLine(state));
  ctx.ui.notify(`Starting Gormes delivery loop: 1/${maxIterations}; log: ${state.logPath}`, "info");
  pi.sendUserMessage(buildIterationPrompt(state));
}

function parseArgs(raw: string): { command: "start" | "restart" | "stop" | "status"; topic: string; iterations: number } {
  const tokens = raw.trim().split(/\s+/).filter(Boolean);
  const command = tokens[0] === "stop" || tokens[0] === "status" || tokens[0] === "start" || tokens[0] === "restart" ? tokens.shift()! : "start";
  let iterations = DEFAULT_ITERATIONS;
  const topicParts: string[] = [];

  for (const token of tokens) {
    const match = token.match(/^--iterations=(\d+)$/) ?? token.match(/^-n=(\d+)$/);
    if (match) {
      iterations = Number(match[1]);
      continue;
    }
    topicParts.push(token);
  }

  return {
    command: command as "start" | "restart" | "stop" | "status",
    topic: topicParts.join(" ").trim() || DEFAULT_TOPIC,
    iterations: Number.isFinite(iterations) && iterations > 0 ? Math.floor(iterations) : DEFAULT_ITERATIONS,
  };
}

function buildIterationPrompt(s: LoopState): string {
  const logPath = s.logPath ?? DEFAULT_LOG_PATH;
  return `Use the gormes-delivery-loop skill now. Delivery loop iteration ${s.iteration}/${s.maxIterations}.

Topic/objective: ${s.topic}.

Required skills this iteration: ${REQUIRED_ITERATION_SKILLS.join(", ")}.
Load and use these before action where they apply. Use gormes-git for green-gate and commit/push discipline only when the worktree is safe for this slice; if unrelated dirty work would be staged, stop blocked instead.

Loop log path: ${logPath}. The extension read the recent log records before queuing this prompt.
Recent loop log records (newest last):
${recentLoopLogSnippet(logPath)}

At the end of this iteration, include exact changed files, validations, commit hash, push status, CI_GREEN status, and the final LOOP_DECISION line so the extension can learn from the run.

Run one complete vertical Gormes delivery iteration:
1. Start with scope lock and preflight for /home/xel/git/sages-openclaw/workspace-mineru/gormes-agent on development.
2. Preserve unrelated dirty work; stage only this iteration's explicit files.
3. Use gormes-architecture-zoomout to find one A/B-evidence architecture candidate for the topic.
4. Use gormes-planner if progress.json or feature-map row shaping is required; progress.json is the only backlog.
5. Use gormes-hermes-parity to confirm the active Hermes/Honcho contract and source refs.
6. Use gormes-builder plus gormes-tdd-slice: add or identify a failing characterization test first, implement the smallest builder-ready slice, include E2E/focused coverage where appropriate.
7. Make CI green every iteration before commit, push, or LOOP_DECISION: continue/done. Required full CI gate:
${FULL_CI_GATE.map((command) => `   - ${command}`).join("\n")}
8. If the full CI gate fails, fix the failure inside this iteration and rerun the failed command plus affected gate. After two same-failure attempts, stop with LOOP_DECISION: blocked and first failing stderr line.
9. Commit and push the validated slice to origin/development with a coherent commit.
10. End with CI_GREEN: yes only after the full CI gate above passes, then exactly one line: LOOP_DECISION: continue, LOOP_DECISION: stop, LOOP_DECISION: blocked, or LOOP_DECISION: done.

Stop instead of coding if the next row is not builder-ready, upstream evidence is missing, validation fails twice with the same blocker, CI cannot be made green, or the slice would touch unrelated dirty Navivox work.`;
}

function buildSelfImprovementPrompt(s: LoopState, reason: string): string {
  const logPath = s.logPath ?? DEFAULT_LOG_PATH;
  return `Improve the project-local Pi extension that just ran the Gormes delivery loop.

Reason the loop ended: ${reason}.
Loop log path: ${logPath}.
Extension file: /home/xel/git/sages-openclaw/workspace-mineru/gormes-agent/.pi/extensions/gormes-delivery-loop.ts.
Repo: /home/xel/git/sages-openclaw/workspace-mineru/gormes-agent on development.

Required skills for extension self-improvement: ${REQUIRED_ITERATION_SKILLS.join(", ")}.
Recent loop log records (newest last):
${recentLoopLogSnippet(logPath)}

Use the gormes-delivery-loop skill, gormes-pi-parity, and improve-codebase-architecture on the extension itself:
1. Read the loop log and this extension file.
2. Identify one concrete improvement to loop reliability, logging, stop conditions, prompts, or self-learning behavior.
3. Add or run a smoke test with mocked Pi API proving the extension still loads and the changed behavior works.
4. Preserve unrelated dirty Navivox work.
5. Make CI green before commit/push. Required full CI gate:
${FULL_CI_GATE.map((command) => `   - ${command}`).join("\n")}
6. Commit and push the extension improvement to origin/development if the worktree is safe.
7. End with CI_GREEN: yes only after the full CI gate passes, then LOOP_DECISION: stop.`;
}

function queueSelfImprovement(pi: ExtensionAPI, ctx: { hasUI?: boolean; ui?: { notify(message: string, level?: string): void; setStatus(key: string, value: string): void } }, reason: string) {
  if (state.selfImproveQueued || sendingFollowUp) return;
  state = { ...state, selfImproveQueued: true, active: false, lastDecision: reason };
  appendLoopLog("self_improvement_queued", { reason });
  pi.appendEntry(CUSTOM_STATE_TYPE, state);
  if (ctx.hasUI && ctx.ui) {
    ctx.ui.notify(`Gormes loop ended (${reason}); queued extension self-improvement.`, "info");
    ctx.ui.setStatus("gormes-loop", statusLine(state));
  }
  sendingFollowUp = true;
  try {
    pi.sendUserMessage(buildSelfImprovementPrompt(state, reason), { deliverAs: "followUp" });
  } finally {
    sendingFollowUp = false;
  }
}

function appendLoopLog(event: string, extra: Partial<LoopLogEvent> = {}) {
  const logPath = state.logPath ?? DEFAULT_LOG_PATH;
  const record: LoopLogEvent = {
    at: new Date().toISOString(),
    event,
    topic: state.topic,
    iteration: state.iteration,
    maxIterations: state.maxIterations,
    logPath,
    ...extra,
  };
  try {
    fs.mkdirSync(path.dirname(logPath), { recursive: true });
    fs.appendFileSync(logPath, `${JSON.stringify(record)}\n`, "utf8");
  } catch {
    // Logging must never break the delivery loop.
  }
}

function parseDecision(text: string): LoopState["lastDecision"] | undefined {
  const match = text.match(/LOOP_DECISION:\s*(continue|stop|blocked|done)/i);
  return match?.[1]?.toLowerCase() as LoopState["lastDecision"] | undefined;
}

function parseCIGate(text: string, s: LoopState): { green: boolean; source: string } {
  if (/CI_GREEN:\s*yes/i.test(text)) return { green: true, source: "assistant_text" };
  if (iterationHasRecentFullCIGateEvidence(s.logPath ?? DEFAULT_LOG_PATH, s.startedAt, s.iteration)) {
    return { green: true, source: "loop_log_full_gate" };
  }
  return { green: false, source: "missing" };
}

function requiresCIGreen(decision: string): boolean {
  return decision === "continue" || decision === "done";
}

function recentLoopLogSnippet(logPath: string): string {
  try {
    const content = fs.readFileSync(logPath, "utf8").trim();
    if (!content) return "(no prior loop log records)";
    return content.split(/\r?\n/).filter(Boolean).slice(-RECENT_LOG_RECORDS).join("\n");
  } catch (error) {
    return `(no readable loop log at ${logPath}: ${errorMessage(error)})`;
  }
}

function iterationHasRecentFullCIGateEvidence(logPath: string, startedAt: string, currentIteration: number): boolean {
  try {
    const content = fs.readFileSync(logPath, "utf8").trim();
    if (!content) return false;
    const started = Date.parse(startedAt);
    const recentRecords = content.split(/\r?\n/).filter(Boolean).slice(-RECENT_LOG_RECORDS).map((line) => parseLogRecord(line)).filter((record): record is Record<string, unknown> => Boolean(record));
    for (let i = recentRecords.length - 1; i >= 0; i--) {
      const record = recentRecords[i];
      if (record.event === "ci_gate_missing") return false;
      if (Number.isFinite(started) && typeof record.at === "string" && Date.parse(record.at) < started) continue;
      if (record.event !== "iteration_result") continue;
      if (!recordMatchesCurrentIteration(record, currentIteration)) continue;
      if (record.ci_green === "yes" || record.ciGreen === true) return true;
      if (hasFullCIGateValidation(record.validation)) return true;
    }
  } catch {
    return false;
  }
  return false;
}

function recordMatchesCurrentIteration(record: Record<string, unknown>, currentIteration: number): boolean {
  const iteration = typeof record.iteration === "number" ? record.iteration : Number(record.iteration);
  return Number.isFinite(iteration) && iteration === currentIteration;
}

function hasFullCIGateValidation(validation: unknown): boolean {
  if (!Array.isArray(validation)) return false;
  const normalized = validation.map((item) => String(item).trim());
  return FULL_CI_GATE.every((required) => normalized.includes(required));
}

function parseLogRecord(line: string): Record<string, unknown> | undefined {
  try {
    const value = JSON.parse(line);
    return value && typeof value === "object" ? value as Record<string, unknown> : undefined;
  } catch {
    return undefined;
  }
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

function statusLine(s: LoopState): string {
  if (!s.active) return `Gormes loop: idle${s.lastDecision ? ` (${s.lastDecision})` : ""}`;
  return `Gormes loop: active ${s.iteration}/${s.maxIterations} ${s.topic}`;
}

function inactiveState(): LoopState {
  return {
    active: false,
    topic: DEFAULT_TOPIC,
    iteration: 0,
    maxIterations: DEFAULT_ITERATIONS,
    startedAt: new Date(0).toISOString(),
    logPath: DEFAULT_LOG_PATH,
    selfImproveQueued: false,
  };
}

function restoreState(entries: Array<{ type?: string; customType?: string; data?: unknown }>): LoopState | undefined {
  for (let i = entries.length - 1; i >= 0; i--) {
    const entry = entries[i];
    if (entry.type === "custom" && entry.customType === CUSTOM_STATE_TYPE && isLoopState(entry.data)) {
      return { ...entry.data, logPath: entry.data.logPath ?? DEFAULT_LOG_PATH };
    }
  }
  return undefined;
}

function isLoopState(value: unknown): value is LoopState {
  if (!value || typeof value !== "object") return false;
  const candidate = value as Partial<LoopState>;
  return typeof candidate.active === "boolean" &&
    typeof candidate.topic === "string" &&
    typeof candidate.iteration === "number" &&
    typeof candidate.maxIterations === "number" &&
    typeof candidate.startedAt === "string";
}

export const __test__ = {
  parseCIGate,
  iterationHasRecentFullCIGateEvidence,
  hasFullCIGateValidation,
  statusReport,
};

function messageText(message: { content?: unknown }): string {
  const content = message.content;
  if (typeof content === "string") return content;
  if (Array.isArray(content)) {
    return content.map((part) => {
      if (typeof part === "string") return part;
      if (part && typeof part === "object" && "text" in part) return String((part as { text?: unknown }).text ?? "");
      return "";
    }).join("\n");
  }
  return "";
}
