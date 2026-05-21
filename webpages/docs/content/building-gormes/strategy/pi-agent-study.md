# Pi Agent Study Notes for Gormes

Source studied: `https://github.com/earendil-works/pi` temporary clone plus installed Pi docs under `/home/xel/.nvm/versions/node/v22.21.1/lib/node_modules/@earendil-works/pi-coding-agent/`.

## Short Summary

Pi is a deliberately minimal coding harness with a small core and an aggressive extension surface. Its best ideas for Gormes are not feature count; they are composable runtime seams: resource discovery, event interception, strict session trees, tool output budgeting, safe file mutation queues, and packageable skills/prompts/extensions.

## Most Useful Tips for Gormes

1. Keep the core small and make workflows installable.
   - Pi’s philosophy is: no built-in subagents, plan mode, or permission popups; build them through extensions, skills, prompt templates, or packages.
   - Gormes should preserve a tight kernel and expose stable seams for channel adapters, tools, review loops, and skills instead of hardcoding every workflow.

2. Treat skills/prompts/extensions as first-class resources.
   - Pi discovers resources from global, project, package, and CLI locations.
   - Gormes should keep `.agents/skills`, repo-local skills, prompt templates, and future plugin/package metadata discoverable with explicit provenance.

3. Attach provenance to every discovered resource.
   - Pi exposes `sourceInfo` for commands/tools/resources.
   - Gormes should report where a skill, tool, model, or prompt came from so operators can debug collisions and trust boundaries.

4. Use event hooks around the agent loop.
   - Pi has hooks before agent start, provider request, tool call, tool result, compaction, tree navigation, and session replacement.
   - Gormes can benefit from typed lifecycle hooks for gateway adapters, audit logging, provider repair, permission gates, and telemetry without modifying the kernel each time.

5. Block dangerous operations before tool execution.
   - Pi examples show `tool_call` gates for destructive bash and protected paths.
   - Gormes should keep or add preflight hooks for writes to secrets, `.git`, lockfiles, state DBs, and destructive shell commands, especially in gateway mode.

6. Serialize same-file mutations.
   - Pi’s `withFileMutationQueue()` queues mutations by realpath so parallel tool calls do not overwrite each other.
   - Gormes should ensure edit/write/custom file tools share a per-file mutation queue.

7. Budget tool output aggressively.
   - Pi truncates large outputs to 2000 lines or 50KB and points to a full temp file when truncated.
   - Gormes tools should standardize head truncation for reads/searches, tail truncation for logs/commands, and always disclose truncation counts and full-output path.

8. Prefer strict JSONL framing.
   - Pi’s RPC reader splits only on `\n`, not generic line readers, because Unicode separators can appear inside JSON strings.
   - Gormes RPC/session readers should keep LF-only JSONL parsing and tests for Unicode separators.

9. Model session history as a tree, not only a linear log.
   - Pi stores `id` and `parentId` per session entry, enabling `/tree`, fork, clone, compaction, branch summaries, and labels.
   - Gormes should compare its session model against this for future in-place branching and safer long-running work.

10. Rebind runtime state after session replacement.
    - Pi’s `AgentSessionRuntime` tears down old session extensions, creates new cwd-bound services, and requires subscriptions/extensions to rebind to the new session.
    - Gormes should avoid stale session-bound objects after resume/fork/import/profile switches.

11. Use package install security as an operating principle.
    - Pi pins direct dependencies, uses `--ignore-scripts`, `min-release-age=2`, shrinkwraps the published CLI, audits signatures, and blocks accidental lockfile commits.
    - Gormes installer/package work should copy the same posture: no lifecycle scripts by default, pinned critical dependencies, release smoke from outside the repo, and explicit lockfile review.

12. Keep TUI components width-safe and cache-aware.
    - Pi TUI requires every rendered line to fit width, reapply styles per line, cache by width, and invalidate on theme changes.
    - Gormes Bubble Tea/TUI tests should keep narrow-terminal fixtures and explicit width/height assertions.

13. Expose read-only planning as a tool policy, not a separate brain.
    - Pi’s plan-mode example switches active tools to read-only and allowlists safe bash commands, then restores execution tools.
    - Gormes can implement planning/review modes by changing active tool policy and visible status, not by forking a separate backlog.

14. Make prompts lightweight slash commands.
    - Pi prompt templates are markdown files with optional frontmatter and positional args.
    - Gormes can use simple prompt-template files for recurring tasks like parity audit, release checklist, review, or handoff instead of adding bespoke commands.

15. Preserve extensibility without hiding security costs.
    - Pi repeatedly warns that packages/extensions/skills execute with user-level power.
    - Gormes should make plugin/skill trust explicit in docs and CLI output.

## Candidate Gormes Follow-Ups

- Add a progress row for shared per-file mutation queue across built-in and custom file tools.
- Audit Gormes tool output truncation for consistent limits, counts, and full-output file paths.
- Compare Gormes session storage with Pi’s tree model for future branch/fork/label parity.
- Add provider/resource provenance to tool, skill, prompt, and model listings if missing.
- Add install/package hardening checks inspired by Pi: ignore scripts, dependency age policy, and isolated release smoke.
- Add a read-only planning/review mode by constraining active tools and shell allowlist.

## Evidence Paths

- Pi README: minimal harness, default tools, sessions, skills, extensions, packages, SDK, RPC.
- Pi docs: `docs/extensions.md`, `docs/skills.md`, `docs/prompt-templates.md`, `docs/packages.md`, `docs/sdk.md`, `docs/tui.md`, `docs/session-format.md`.
- Pi examples: `permission-gate.ts`, `protected-paths.ts`, `truncated-tool.ts`, `plan-mode/README.md`, `sdk/13-session-runtime.ts`.
- Pi source: `packages/coding-agent/src/core/tools/file-mutation-queue.ts`, `packages/coding-agent/src/core/tools/truncate.ts`, `packages/coding-agent/src/modes/rpc/jsonl.ts`, `packages/coding-agent/src/core/agent-session-runtime.ts`.
- Pi repo hardening: `.npmrc`, root `README.md`, root `AGENTS.md`, root `package.json`.
