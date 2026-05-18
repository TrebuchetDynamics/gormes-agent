---
title: "Config Module Roadmap"
---

# Config Module Roadmap

Generated from the single logical backlog. This page is a scoped review view; `progress.json` remains canonical.

**Module:** `config`
**Rows:** 31
**Status counts:** `complete`: 30 · `in_progress`: 0 · `planned`: 1
**Priority counts:** `P0`: 3 · `P1`: 8 · `P2`: 7 · `P3`: 1 · `unset`: 12

## Phase 3 — The Black Box (Memory)

### 3.A — SQLite + FTS5 Lattice

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `config` | Schema migrations v3a->v3d |

### 3.E.1 — Session Index Mirror

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `config` | Read-only bbolt sessions.db -> index.yaml mirror |

## Phase 5 — The Final Purge

### 5.A — Tool Surface Port

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `config` | Stateful tool migration queue |

### 5.G — MCP Integration

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `config` | MCP server config/env resolver |

### 5.J — Approval / Security Guards

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `config` | Approval mode config normalization |
| `complete` | `P2` | `config` | Cron approval mode config normalizer |

### 5.L — File Ops + Patches

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `config` | Terminal cwd config bridge |

### 5.N — Misc Operator Tools

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P0` | `config` | Cross-agent config isolation |

### 5.O — Hermes CLI Parity

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P3` | `config` | CLI onboarding seen-state map helpers |
| `complete` | `P2` | `config` | CLI contextual first-touch onboarding hint renderers |
| `complete` | `P2` | `config` | Gormes setup minimal sectioned wizard slice |
| `complete` | `P2` | `config` | Gormes setup top-level chooser menu |
| `complete` | `P2` | `config` | Gormes setup full-wizard shell and branded summary |
| `complete` | `P0` | `config` | Hermes setup entry-mode and reset semantics |
| `complete` | `P2` | `config` | Gormes setup tools checklist command binding |
| `complete` | `unset` | `config` | Gormes config command surface |
| `complete` | `P1` | `config` | Gormes config set comment-preserving TOML writes |
| `complete` | `unset` | `config` | Gormes config edit/check/native schema-migrate closeout |
| `complete` | `unset` | `config` | Hermes config migration dry-run manifest |
| `complete` | `unset` | `config` | Hermes config migration writer |
| `complete` | `unset` | `config` | OpenClaw migration dry-run manifest |
| `complete` | `unset` | `config` | OpenClaw migration writer and cleanup command |
| `complete` | `unset` | `config` | Platform toolset config persistence + MCP sentinel |
| `complete` | `P1` | `config` | gormes setup <section> boxed header + completion footer (UX parity) |
| `complete` | `P1` | `config` | Interactive Onboarding |
| `complete` | `P1` | `config` | Internal onboarding interactive action runner |
| `complete` | `P0` | `config` | CLI setup/onboard/help text fidelity matrix |
| `complete` | `P1` | `config` | Root config.toml v2 profile service schema |
| `planned` | `P1` | `config` | Legacy profile config v2 migration planner |

### 5.R — Code Execution Mode Policy

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `config` | Execution-mode resolver + config precedence |
| `complete` | `P2` | `config` | Default mode selection + config cut-over |
