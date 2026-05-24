# Pure-Go TTS Decision

Date: 2026-05-10
Updated: 2026-05-24
Status: superseded by Go-native speech direction

## Context

Gormes ships TTS via `internal/tools/tts_tool.go` using cloud providers (Edge,
OpenAI, MiniMax, etc.) and local command providers (`edge-tts`, `piper`,
`say`, or user-declared `tts.providers.<name>.command`). That bridge is useful
compatibility, but it does not satisfy the newer product goal: Gormes should
own a Go-native speech path where feasible and avoid making Python, Node,
shell tools, or cloud credentials the only practical local TTS answer.

See `go-native-speech-options.md` for the current STT/TTS architecture target.

## Original Options

### A: Pure-Go TTS library

Use a Go speech synthesis library or small speech engine.

- Binary size: depends on language data and voice assets.
- Voice quality: likely robotic unless paired with neural model artifacts.
- Platform: works everywhere Go compiles if it avoids CGO.
- Maintenance: library and voice-data upkeep stay with Gormes.

### B: Go-owned WASM neural TTS engine

Run a small neural TTS engine artifact through wazero, mirroring the shipped
WASI Whisper STT architecture.

- Binary size: runtime code stays in Go; model/voice artifacts are external
  cache files, not git-tracked or embedded by default.
- Voice quality: potentially good with Piper-style voices.
- Platform: preserves `CGO_ENABLED=0` and Termux/Windows/locked-down Linux
  deployment constraints.
- Maintenance: requires model artifact pinning, cache helpers, benchmark
  budgets, and occasional engine rebuilds.

### C: Keep command-provider pattern

Users install `edge-tts`, `piper`, or another command via their package manager.

- Binary size: +0.
- Voice quality: good when dependencies are present.
- Platform: depends on package-manager and shell availability.
- Maintenance: low for Gormes, but failures happen outside Gormes control.

## Superseded Decision

The 2026-05-10 decision chose **Option C — keep command-provider pattern**.
That remains a compatibility fallback, but it is no longer the target local TTS
architecture.

## Current Decision

Choose **Option B as the target architecture**: add a Go-owned WASM neural TTS
backend behind the existing `TTSProvider` interface, while retaining cloud and
command providers as fallback adapters.

Rationale:

- Gormes already proved the WASI/wazero deployment model for local STT through
  `internal/wasi/whisper`.
- The existing `TTSRunner` interface can absorb a Go-owned provider without
  changing the model-facing `text_to_speech` schema or gateway media delivery.
- Model/voice artifacts can follow the STT cache pattern: checksum verified,
  operator-visible, and excluded from git.
- The new provider can be opt-in until benchmarks prove load time, synthesis
  latency, memory, and binary-size impact are acceptable.

## Consequences

- `tts_command_provider.go` stays for compatibility and user-declared bridges.
- Cloud TTS providers stay available.
- A new progress row owns source selection and implementation of the Go-owned
  TTS backend; no runtime code should be faked before the engine/model artifact
  choice is sourced.
- Slim/lite builds must continue to exclude or gracefully degrade speech
  helpers.
