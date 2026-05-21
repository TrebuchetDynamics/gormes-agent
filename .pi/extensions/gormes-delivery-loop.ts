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
};

const CUSTOM_STATE_TYPE = "gormes-delivery-loop-state";
const DEFAULT_TOPIC = "auto-select the highest-impact builder-ready progress.json row toward finishing full Hermes-in-Go parity";
const DEFAULT_ITERATIONS = 10;
const HARD_MAX_ITERATIONS = 50;
const DEFAULT_LOG_PATH = path.join(".pi", "gormes-loop", "logs.jsonl");

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
    if (decision) {
      state.lastDecision = decision;
      appendLoopLog("assistant_decision", { decision });
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
      const values = ["start", "stop", "status"];
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
          ctx.ui.notify(`${statusLine(state)}\nlog: ${state.logPath ?? DEFAULT_LOG_PATH}`, "info");
          return;
        case "start":
        default:
          await startLoop(pi, ctx, parsed.topic, parsed.iterations);
          return;
      }
    },
  });
}

async function startLoop(pi: ExtensionAPI, ctx: ExtensionCommandContext, topic: string, requestedIterations: number) {
  const maxIterations = Math.max(1, Math.min(requestedIterations, HARD_MAX_ITERATIONS));
  if (requestedIterations > HARD_MAX_ITERATIONS) {
    ctx.ui.notify(`Capped Gormes loop at ${HARD_MAX_ITERATIONS} iterations.`, "warning");
  }

  if (state.active) {
    const ok = await ctx.ui.confirm("Gormes loop already active", "Replace the current delivery loop state?");
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
  await ctx.sendUserMessage(buildIterationPrompt(state));
}

function parseArgs(raw: string): { command: "start" | "stop" | "status"; topic: string; iterations: number } {
  const tokens = raw.trim().split(/\s+/).filter(Boolean);
  const command = tokens[0] === "stop" || tokens[0] === "status" || tokens[0] === "start" ? tokens.shift()! : "start";
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
    command: command as "start" | "stop" | "status",
    topic: topicParts.join(" ").trim() || DEFAULT_TOPIC,
    iterations: Number.isFinite(iterations) && iterations > 0 ? Math.floor(iterations) : DEFAULT_ITERATIONS,
  };
}

function buildIterationPrompt(s: LoopState): string {
  return `Use the gormes-delivery-loop skill now. Delivery loop iteration ${s.iteration}/${s.maxIterations}.

Topic/objective: ${s.topic}.

Loop log path: ${s.logPath ?? DEFAULT_LOG_PATH}. At the end of this iteration, include exact changed files, validations, commit hash, push status, and the final LOOP_DECISION line so the extension can learn from the run.

Run one complete vertical Gormes delivery iteration:
1. Start with scope lock and preflight for /home/xel/git/sages-openclaw/workspace-mineru/gormes-agent on development.
2. Preserve unrelated dirty work; stage only this iteration's explicit files.
3. Use gormes-architecture-zoomout to find one A/B-evidence architecture candidate for the topic.
4. Use gormes-planner if progress.json or feature-map row shaping is required; progress.json is the only backlog.
5. Use gormes-hermes-parity to confirm the active Hermes/Honcho contract and source refs.
6. Use gormes-builder plus TDD: add or identify a failing characterization test first, implement the smallest builder-ready slice, include E2E/focused coverage where appropriate.
7. Validate with row tests, focused package tests, go run ./cmd/progress validate, git diff --check, and full go test ./... -count=1 when the slice touches shared runtime behavior.
8. Commit and push the validated slice to origin/development with a coherent commit.
9. End with exactly one line: LOOP_DECISION: continue, LOOP_DECISION: stop, LOOP_DECISION: blocked, or LOOP_DECISION: done.

Stop instead of coding if the next row is not builder-ready, upstream evidence is missing, validation fails twice with the same blocker, or the slice would touch unrelated dirty Navivox work.`;
}

function buildSelfImprovementPrompt(s: LoopState, reason: string): string {
  return `Improve the project-local Pi extension that just ran the Gormes delivery loop.

Reason the loop ended: ${reason}.
Loop log path: ${s.logPath ?? DEFAULT_LOG_PATH}.
Extension file: /home/xel/git/sages-openclaw/workspace-mineru/gormes-agent/.pi/extensions/gormes-delivery-loop.ts.
Repo: /home/xel/git/sages-openclaw/workspace-mineru/gormes-agent on development.

Use the gormes-delivery-loop skill and improve-codebase-architecture on the extension itself:
1. Read the loop log and this extension file.
2. Identify one concrete improvement to loop reliability, logging, stop conditions, prompts, or self-learning behavior.
3. Add or run a smoke test with mocked Pi API proving the extension still loads and the changed behavior works.
4. Preserve unrelated dirty Navivox work.
5. Commit and push the extension improvement to origin/development.
6. End with LOOP_DECISION: stop.`;
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
