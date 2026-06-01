---
title: "Parity Audit: Hermes agent/prompt_builder.py, tools/terminal_tool.py, tools/code_execution_tool.py"
aliases:
  - /building-gormes/architecture_plan/parity-audit-prompt-terminal-code/
---
# Parity Audit: Hermes agent/prompt_builder.py, tools/terminal_tool.py, tools/code_execution_tool.py

**Date:** 2026-04-30
**Auditor:** Sisyphus
**Scope:** Prompt builder, terminal tool, code execution tool parity gaps

---

## 1. agent/prompt_builder.py (1,122 lines)

### Upstream Behavior

The Hermes prompt builder is a comprehensive system prompt assembly engine:

| Feature | Hermes Implementation | Gormes Status | Gap |
|---------|----------------------|---------------|-----|
| **Context file scanning** | `_scan_context_content` — detects prompt injection, invisible unicode, threat patterns in AGENTS.md/.cursorrules/SOUL.md | Not implemented | **Critical** |
| **Git root discovery** | `_find_git_root` — walks parents looking for `.git` | Not implemented | Medium |
| **.hermes.md discovery** | `_find_hermes_md` — searches cwd → git root for `.hermes.md`/`HERMES.md` | Not implemented | Medium |
| **YAML frontmatter stripping** | `_strip_yaml_frontmatter` — removes `---` delimited frontmatter | Not implemented | Low |
| **Default identity** | `DEFAULT_AGENT_IDENTITY` — "You are Hermes Agent..." | Not implemented | Medium |
| **Help guidance** | `HERMES_AGENT_HELP_GUIDANCE` — references skill_view | Not implemented | Low |
| **Memory guidance** | `MEMORY_GUIDANCE` — durable facts, compact focus, declarative facts | Not implemented | **Critical** |
| **Session search guidance** | `SESSION_SEARCH_GUIDANCE` — cross-session recall | Not implemented | Medium |
| **Skills guidance** | `SKILLS_GUIDANCE` — save complex tasks as skills | Partial (skills_list/view exist) | Medium |
| **Tool use enforcement** | `TOOL_USE_ENFORCEMENT_GUIDANCE` — MUST use tools, no promises | Not implemented | **Critical** |
| **Model-specific guidance** | Various model substrings trigger different guidance | Not implemented | Medium |
| **Context file assembly** | Combines identity + platform + skills index + context files + memory | Not implemented | **Critical** |

### Gormes Current State

- No native prompt builder exists in `internal/llm/`
- The `Kernel tool loop` exists but lacks prompt assembly
- Skills tools (`skills_list`, `skill_view`) are implemented
- Memory tool exists but no memory guidance injection

### Recommended Row

**4.C Native Prompt Builder** (existing row in progress.json)
- Priority: P0
- Critical path for Python-free normal agent turn

---

## 2. tools/terminal_tool.py (2,307 lines)

### Upstream Behavior

Hermes terminal tool supports 7 execution backends:

| Backend | Hermes | Gormes | Gap |
|---------|--------|--------|-----|
| **local** | Full foreground/background, PTY, interrupt polling | Basic foreground only | Partial |
| **docker** | Containerized execution, auto-cleanup | Not implemented | Gap |
| **modal** | Cloud sandbox, direct or managed gateway | Not implemented | Gap |
| **vercel_sandbox** | Vercel Sandbox with runtime selection | Not implemented | Gap |
| **ssh** | Remote SSH execution | Not implemented | Gap |
| **singularity** | Singularity containers | Not implemented | Gap |
| **daytona** | Daytona sandbox | Not implemented | Gap |

**Additional Hermes features:**

| Feature | Hermes | Gormes | Gap |
|---------|--------|--------|-----|
| **Background tasks** | Full background process registry | Rejected as unsupported | Large |
| **Interrupt handling** | Global interrupt event, kills subprocesses | Not implemented | Medium |
| **Disk usage warnings** | Warns when scratch dir exceeds 500GB | Not implemented | Low |
| **Environment selection** | `TERMINAL_ENV` variable | Not implemented | Medium |
| **Timeout enforcement** | Foreground max 600s, configurable | Basic timeout (180s default) | Partial |
| **PTY support** | Real PTY allocation | Schema accepts but rejects | Gap |
| **Cleanup** | Auto-cleanup after inactivity | Not implemented | Low |

### Gormes Current State

- `internal/tools/terminal_tool.go` (227 lines)
- Local execution only via `bash -lc`
- Basic timeout, output truncation, ANSI stripping
- Dangerous command guardrails (36+ patterns)
- Permission approval system (manual/smart/off)
- Background processes explicitly rejected
- PTY accepted in schema but rejected at runtime

### Recommended Rows

1. **5.B.2 Terminal Background Process Registry** — Background task support
2. **5.B.3 Terminal Docker Backend** — Docker containerized execution
3. **5.B.4 Terminal Cloud Backends** — Modal/Vercel/SSH/Daytona backends
4. **5.B.5 Terminal Interrupt Handling** — Global interrupt event for subprocesses

---

## 3. tools/code_execution_tool.py (1,609 lines)

### Upstream Behavior

Hermes code execution provides a Programmatic Tool Calling (PTC) sandbox:

| Feature | Hermes | Gormes | Gap |
|---------|--------|--------|-----|
| **UDS RPC** | Unix domain socket for local tool calls | Not implemented | **Critical** |
| **File-based RPC** | Remote backends via file polling | Not implemented | Large |
| **hermes_tools.py stub** | Auto-generated Python module with tool stubs | Not implemented | Large |
| **Sandbox allowed tools** | 7 tools: web_search, web_extract, read_file, write_file, search_files, patch, terminal | Partial (tools exist but no sandbox wrapper) | Large |
| **Resource limits** | Timeout (300s), max tool calls (50), stdout (50KB), stderr (10KB) | Basic timeout only | Medium |
| **Transport selection** | UDS vs file-based based on backend | Not implemented | Medium |
| **Sandbox availability** | POSIX-only gate (`SANDBOX_AVAILABLE`) | Go equivalent needed | Medium |

### Gormes Current State

- `internal/tools/execute_code.go` exists but is minimal
- Basic sandboxed execution with `sh`/`python` snippets
- Filesystem/network blocking via pre-exec
- Runtime selection (python/sh)
- No RPC architecture
- No hermes_tools.py stub generation

### Recommended Row

**5.K Code Execution Sandbox** (already exists in progress.json as umbrella)
- Needs splitting into: RPC transport, stub generation, resource limits, backend selection

---

## Summary: Critical Path Gaps

| Priority | File | Gap | Blocks Dogfood? |
|----------|------|-----|-----------------|
| P0 | `agent/prompt_builder.py` | No native prompt builder | **Yes** |
| P0 | `tools/code_execution_tool.py` | No UDS RPC sandbox | No |
| P1 | `tools/terminal_tool.py` | No background/docker/cloud backends | No |
| P1 | `agent/prompt_builder.py` | No context file scanning | No |
| P1 | `agent/prompt_builder.py` | No memory/session/skills guidance injection | **Yes** |
| P2 | `tools/terminal_tool.py` | No interrupt handling | No |
| P2 | `tools/code_execution_tool.py` | No hermes_tools.py stub generation | No |

---

## Next Actions

1. **Implement native prompt builder** (4.C) — highest priority, blocks dogfood
2. **Expand terminal tool backends** (5.B.x) — medium priority
3. **Build code execution RPC** (5.K split) — medium priority
4. **Add context file scanning** to prompt builder — security-critical
