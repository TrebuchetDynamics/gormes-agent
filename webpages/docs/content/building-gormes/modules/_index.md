---
title: "Module Roadmaps"
weight: 35
---

# Module Roadmaps

Generated from the single logical backlog. These pages are scoped review views; `progress.json` remains canonical.

## Agent Runtime

Core agent execution, local state, tools, and terminal/browser interaction surfaces.

| Module | Rows | Complete | In progress | Planned | Priorities |
|---|---:|---:|---:|---:|---|
| [Browser](agent-runtime/browser/) | 26 | 26 | 0 | 0 | `P1`: 16 · `P2`: 6 · `P3`: 1 · `unset`: 3 |
| [Learning Loop](agent-runtime/learning-loop/) | 6 | 6 | 0 | 0 | `P1`: 3 · `P2`: 2 · `P3`: 1 |
| [Memory](agent-runtime/memory/) | 29 | 29 | 0 | 0 | `P1`: 1 · `P2`: 3 · `P3`: 1 · `unset`: 24 |
| [Runtime](agent-runtime/runtime/) | 18 | 18 | 0 | 0 | `P0`: 2 · `P1`: 10 · `P2`: 6 |
| [Sessions](agent-runtime/sessions/) | 29 | 29 | 0 | 0 | `P1`: 9 · `P2`: 5 · `P3`: 1 · `unset`: 14 |
| [Tools](agent-runtime/tools/) | 151 | 151 | 0 | 0 | `P0`: 23 · `P1`: 57 · `P2`: 35 · `P3`: 8 · `P4`: 2 · `unset`: 26 |
| [TUI](agent-runtime/tui/) | 72 | 72 | 0 | 0 | `P0`: 3 · `P1`: 15 · `P2`: 43 · `P3`: 1 · `unset`: 10 |

## Channel Gateway

External channel adapters, gateway orchestration, fleet operation, and Navivox integration.

| Module | Rows | Complete | In progress | Planned | Priorities |
|---|---:|---:|---:|---:|---|
| [Channels](channel-gateway/channels/) | 134 | 134 | 0 | 0 | `P0`: 7 · `P1`: 47 · `P2`: 27 · `P3`: 4 · `P4`: 2 · `unset`: 47 |
| [Fleet](channel-gateway/fleet/) | 23 | 23 | 0 | 0 | `P0`: 2 · `P1`: 7 · `P2`: 6 · `P3`: 3 · `unset`: 5 |
| [Gateway](channel-gateway/gateway/) | 161 | 161 | 0 | 0 | `P0`: 14 · `P1`: 52 · `P2`: 37 · `P3`: 3 · `P4`: 1 · `unset`: 54 |
| [Navivox](channel-gateway/navivox/) | 28 | 28 | 0 | 0 | `P0`: 3 · `P1`: 19 · `unset`: 6 |

## Delivery Control Plane

Planner, builder, skill, documentation, kanban, and progress-control surfaces that steer Gormes delivery.

| Module | Rows | Complete | In progress | Planned | Priorities |
|---|---:|---:|---:|---:|---|
| [Builder](delivery-control-plane/builder/) | 12 | 12 | 0 | 0 | `P0`: 1 · `P1`: 6 · `P2`: 3 · `unset`: 2 |
| [Cross Cutting](delivery-control-plane/cross-cutting/) | 0 | 0 | 0 | 0 | - |
| [Docs](delivery-control-plane/docs/) | 22 | 19 | 0 | 3 | `P1`: 13 · `P2`: 7 · `P3`: 2 |
| [Kanban](delivery-control-plane/kanban/) | 33 | 33 | 0 | 0 | `P1`: 14 · `P2`: 19 |
| [Planner](delivery-control-plane/planner/) | 10 | 10 | 0 | 0 | `P0`: 1 · `P1`: 4 · `P2`: 1 · `P3`: 1 · `unset`: 3 |
| [Progress](delivery-control-plane/progress/) | 26 | 26 | 0 | 0 | `P1`: 11 · `P2`: 15 |
| [Skills](delivery-control-plane/skills/) | 52 | 52 | 0 | 0 | `P0`: 6 · `P1`: 14 · `P2`: 16 · `P3`: 6 · `P4`: 1 · `unset`: 9 |

## Operator Setup

CLI, installation, configuration, diagnostics, profiles, and release lifecycle pages for operators.

| Module | Rows | Complete | In progress | Planned | Priorities |
|---|---:|---:|---:|---:|---|
| [CLI](operator-setup/cli/) | 35 | 35 | 0 | 0 | `P1`: 14 · `P2`: 8 · `P3`: 3 · `unset`: 10 |
| [Config](operator-setup/config/) | 31 | 31 | 0 | 0 | `P0`: 3 · `P1`: 8 · `P2`: 7 · `P3`: 1 · `unset`: 12 |
| [Doctor](operator-setup/doctor/) | 16 | 16 | 0 | 0 | `P1`: 5 · `P2`: 7 · `P3`: 2 · `unset`: 2 |
| [Install](operator-setup/install/) | 30 | 30 | 0 | 0 | `P0`: 2 · `P1`: 17 · `P2`: 4 · `P3`: 3 · `unset`: 4 |
| [Profiles](operator-setup/profiles/) | 24 | 24 | 0 | 0 | `P0`: 2 · `P1`: 12 · `P2`: 4 · `unset`: 6 |
| [Release](operator-setup/release/) | 13 | 13 | 0 | 0 | `P0`: 3 · `P1`: 10 |

## Provider Models

Model-provider contracts plus Goncho/Honcho memory parity surfaces.

| Module | Rows | Complete | In progress | Planned | Priorities |
|---|---:|---:|---:|---:|---|
| [Goncho](provider-models/goncho/) | 45 | 45 | 0 | 0 | `P0`: 7 · `P1`: 12 · `P2`: 11 · `P3`: 12 · `P4`: 2 · `unset`: 1 |
| [Providers](provider-models/providers/) | 126 | 126 | 0 | 0 | `P0`: 9 · `P1`: 55 · `P2`: 24 · `P3`: 3 · `unset`: 35 |

## Voice And Web

Public web presence and speech input/output surfaces.

| Module | Rows | Complete | In progress | Planned | Priorities |
|---|---:|---:|---:|---:|---|
| [Landing](voice-and-web/landing/) | 4 | 4 | 0 | 0 | `P1`: 2 · `P2`: 2 |
| [STT](voice-and-web/stt/) | 4 | 4 | 0 | 0 | `P0`: 1 · `P1`: 3 |
| [TTS](voice-and-web/tts/) | 25 | 25 | 0 | 0 | `P0`: 2 · `P1`: 12 · `P2`: 4 · `P3`: 6 · `unset`: 1 |
