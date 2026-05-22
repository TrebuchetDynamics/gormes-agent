---
title: "Install Module Roadmap"
---

# Install Module Roadmap

Generated from the single logical backlog. This page is a scoped review view; `progress.json` remains canonical.

**Module:** `install`
**Rows:** 30
**Status counts:** `complete`: 30 · `in_progress`: 0 · `planned`: 0
**Priority counts:** `P0`: 2 · `P1`: 17 · `P2`: 4 · `P3`: 3 · `unset`: 4

## Phase 1 — The Dashboard

### 5.X — Termux Runtime Compatibility

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `install` | Gormes Termux Runtime Compatibility |
| `complete` | `P1` | `install` | Termux install and release smoke guide |
| `complete` | `P1` | `install` | Termux storage and path safety audit |
| `complete` | `P2` | `install` | Termux notification bridge via termux-api |
| `complete` | `P1` | `install` | Termux real-device smoke evidence |
| `complete` | `P2` | `install` | Termux remote execution guidance |

## Phase 5 — The Final Purge

### 5.B — Sandboxing Backends

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `install` | Docker execution backend (container lifecycle + mount policy) |
| `complete` | `P3` | `install` | Docker backend top-level container reuse semantics |

### 5.O — Hermes CLI Parity

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P3` | `install` | Gormes uninstall dry-run command contract |
| `complete` | `P1` | `install` | Gormes update verified binary swap and rollback |

### 5.P — Docker / Packaging

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `install` | OCI image |
| `complete` | `unset` | `install` | Homebrew |
| `complete` | `P1` | `install` | Nix flake package and NixOS module contract |
| `complete` | `unset` | `install` | Unix installer (install.sh) source-backed update flow |
| `complete` | `unset` | `install` | Unix installer root/FHS layout policy |
| `complete` | `P2` | `install` | Windows installer (install.ps1 + install.cmd) parity |
| `complete` | `P3` | `install` | Installer script serving and MIME validation |
| `complete` | `P1` | `install` | Install isolation: GORMES_BIN_DIR is an authoritative sandbox boundary |
| `complete` | `P2` | `install` | Install isolation: skip shell-rc PATH write when bin dir is under /tmp |
| `complete` | `P0` | `install` | Install isolation: skip system service install when sandbox bin dir is set |
| `complete` | `P1` | `install` | Install: prefer pre-built release binary over source build by default |
| `complete` | `P1` | `install` | Install: Termux publishes a real $PREFIX/bin binary, not an $HOME-targeting symlink |
| `complete` | `P1` | `install` | Termux exec argv path-alias sanitizer |
| `complete` | `P1` | `install` | Termux binary-fetch publish verification source fallback |

## Phase 8 — Reputation & Publication

### 8.B — Repository Messaging

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `install` | No-stack first-run proof path from install to offline doctor |

### 8.D — Sharp v1.0

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `install` | CI and installer Go toolchain floor sync |
| `complete` | `P1` | `install` | Windows install.ps1 release binary fetch selector |
| `complete` | `P1` | `install` | OCI image PR build and arm64 smoke workflow |
| `complete` | `P1` | `install` | Termux android/arm64 release artifact and installer selector |

## Phase 9 — Design & Security Hardening

### 9.G — External Issue Radar Regression Guards

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P0` | `install` | PicoClaw-derived tool path safety regression pack |
