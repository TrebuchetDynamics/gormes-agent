# Pure-Go TTS Decision

Date: 2026-05-10
Status: decided
Decided: keep command-provider pattern with edge-tts/piper shell-out

## Context

Gormes ships TTS via `internal/tools/tts_tool.go` using cloud providers (Edge,
OpenAI, MiniMax, etc.) and local command providers (edge-tts, piper). For
platforms without easy Python/node installation (Termux, Windows-sans-tooling,
locked-down corp Linux), users may not have `edge-tts` on PATH. We could ship
a pure-Go TTS engine to eliminate the external dependency.

## Options

### A: Pure-Go TTS library
Use a Go speech synthesis library (e.g. go-tts via espeak-ng bindings).
- Binary size: +15–30 MB (espeak-ng data files)
- Voice quality: robotic, English-only
- Platform: works everywhere Go compiles
- Maintenance: bindings need upkeep as espeak-ng evolves

### B: Embed WASM TTS engine
Embed a small WASM TTS engine (e.g. piper compiled to WASM) via wazero.
- Binary size: +10–25 MB (model + WASM binary)
- Voice quality: good (neural, multi-language via piper voices)
- Platform: WASM runtime works on Linux/macOS/Windows/Termux
- Maintenance: model files need periodic updates, WASM binary needs recompile

### C: Keep command-provider pattern (current)
Users install `edge-tts` or `piper` via their package manager.
- Binary size: +0
- Voice quality: best (native piper/edge-tts)
- Platform: requires package manager access
- Maintenance: zero — providers update independently

## Decision

**Option C — keep command-provider pattern.** 

Rationale:
- Gormes' static-binary promise is mostly about the agent runtime, not TTS
  voices. The command-provider path already works on all major platforms
  where users can install dependencies.
- For Termux/Android, piper-termux is available via pkg.
- For Windows, edge-tts works natively.
- Adding 15–30 MB of voice data to every Gormes binary would slow
  cold-start and inflate the binary for a feature most users don't need.
- A future follow-up could implement Option B (piper WASM) as a
  Gormes-owned TTS backend when the wazero embedding story matures.

## Consequences

- Gormes continues to shell out to `edge-tts` / `piper` / `say` for local TTS.
- Cloud TTS providers (Edge API, OpenAI, MiniMax) remain available as
  the primary TTS path.
- The `tts_command_provider.go` placeholder rendering stays as the local
  bridge.
- A future row can add piper WASM embedding without changing the TTSRunner
  interface.
