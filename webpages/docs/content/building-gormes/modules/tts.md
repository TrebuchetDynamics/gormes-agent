---
title: "TTS Module Roadmap"
---

# TTS Module Roadmap

Generated from the single logical backlog. This page is a scoped review view; `progress.json` remains canonical.

**Module:** `tts`
**Rows:** 25
**Status counts:** `complete`: 24 · `in_progress`: 0 · `planned`: 1
**Priority counts:** `P0`: 2 · `P1`: 12 · `P2`: 4 · `P3`: 6 · `unset`: 1

## Phase 5 — The Final Purge

### 5.E — TTS / Voice / Transcription

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P3` | `tts` | Voice mode port |
| `complete` | `P1` | `tts` | Voice mode environment detector + audio provider seam |
| `complete` | `unset` | `tts` | Transcription tool contract |
| `complete` | `P0` | `tts` | Telegram voice/audio STT ingress hook |
| `complete` | `P0` | `tts` | TTS tool contract + media delivery seam |
| `complete` | `P1` | `tts` | MiniMax TTS v1 text_to_speech raw-audio compatibility |
| `complete` | `P1` | `tts` | TTS provider matrix + dotenv/command-provider resolution |
| `complete` | `P3` | `tts` | TTS synthesis + voice-mode state |
| `complete` | `P1` | `tts` | Voice record-key config binding for native TUI |
| `complete` | `P1` | `tts` | Telegram voice STT HTTP-provider fallback |
| `complete` | `P3` | `tts` | Pure-Go STT exploration |
| `complete` | `P1` | `tts` | wazero WASI smoke harness |
| `complete` | `P1` | `tts` | whisper.cpp WASI module discovery |
| `complete` | `P1` | `tts` | Pure-Go Whisper transcribe one WAV |
| `complete` | `P1` | `tts` | Whisper tiny.en model cache fetcher |
| `complete` | `P1` | `tts` | Wire Pure-Go Whisper into Telegram resolver |
| `complete` | `P2` | `tts` | WASI Whisper ffmpeg preprocess + fixed-window chunker |
| `complete` | `P2` | `tts` | Audio preprocessing and chunking pipeline |
| `complete` | `P2` | `tts` | Whisper benchmark harness + perf budget |
| `complete` | `P3` | `tts` | Go-native OGG/Opus decoder decision |
| `complete` | `P3` | `tts` | Go-native OGG/Opus decoder implementation |
| `complete` | `P3` | `tts` | Pure-Go TTS decision research |
| `complete` | `P1` | `tts` | Shared speech artifact cache for Go-owned TTS |
| `planned` | `P1` | `tts` | Go-owned local TTS runtime seam + fixture fallback |

### 5.O — Hermes CLI Parity

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P2` | `tts` | Gormes setup terminal TTS and agent-settings section bindings |
