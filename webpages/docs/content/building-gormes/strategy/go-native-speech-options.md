# Go-Native Speech Options

Date: 2026-05-24
Status: architecture direction

## Intent

Gormes should own a Go-native speech path for both STT and TTS instead of
making Python, Node, shell tools, or cloud providers the only practical path.
Cloud and command providers remain compatibility fallbacks, but the target
operator experience is: one Gormes binary plus explicit model/voice artifacts,
with no hidden Python/venv dependency.

## Current STT Shape

The STT side is already close to the target architecture:

- `internal/wasi/whisper` owns a wazero + whisper.wasm transcriber API.
- `internal/wasi/whisper/audio` owns preprocessing and chunking.
- `internal/tools/transcription_providers_local.go` exposes the local provider
  behind the existing `TranscriptionRunner` interface.
- `cmd/gormes/telegram_transcriber.go` can route Telegram voice through the
  in-binary WASI Whisper path when a model is configured.

Remaining STT architecture work is optimization and polish: model choice,
cache/install UX, ffmpeg-free decoding where possible, benchmark budgets, and
language/voice-profile routing. It does not need a new public tool contract.

## Current TTS Shape

The TTS side does not yet satisfy the Go-owned target:

- `internal/tools/tts_tool.go` has a useful runner/provider seam and redacted
  result envelope.
- Cloud HTTP providers and command providers are implemented.
- `LazyLocalTTSProvider` currently proves dependency status only; it returns
  `TTS synthesis provider is not wired in this build` when selected.
- `webpages/docs/content/building-gormes/strategy/tts-decision.md` previously chose the
  command-provider pattern as the local TTS answer.

That old decision keeps compatibility but does not meet the new product goal:
Gormes should be able to synthesize speech through Go-owned runtime code.

## Options

### A. Keep command providers as the local path

This is the previous decision. It is small and reliable for users who can
install `piper`, `edge-tts`, or platform tools, but it keeps local TTS outside
Gormes and creates the same Python/Node/package-manager failure class Gormes is
trying to remove.

### B. Add a Go-owned WASM neural TTS backend

Compile or consume a small TTS engine artifact through wazero, mirror the STT
model-cache pattern, and expose it as a normal `TTSProvider` such as
`piper_wasm` or `local_wasm`.

Benefits:

- Preserves `CGO_ENABLED=0` and the single-binary runtime shape.
- Reuses the proven WASI model from STT: checksum-verified artifacts,
  explicit cache directory, typed unavailable evidence, and benchmark budget.
- Keeps current cloud/command providers as fallbacks instead of hard
  dependencies.

Costs:

- Voice/model artifacts are large and must not be embedded by default.
- Latency and memory must be measured before becoming the default provider.
- The exact engine artifact and license need a sourcing pass before coding.

### C. Add a Go-native non-neural speech fallback

Use a small formant/diphone-style engine or vendored Go library for an offline
robotic fallback.

Benefits: likely smaller and simpler than neural TTS.
Costs: quality may be too poor for normal gateway voice replies; language
coverage may be narrow.

## Decision

Choose **Option B as the target architecture**: a Go-owned WASM neural TTS
backend behind the existing `TTSProvider` interface, with command/cloud
providers retained as compatibility fallbacks.

This reverses the 2026-05-10 local-TTS decision because the operator goal has
changed: Gormes should provide its own Go-native speech implementation where
feasible, not only bridges to external tools.

## Target Module Shape

Do not put the TTS engine directly in gateway/channel code. The deep module is
a speech runtime package with the same locality that `internal/wasi/whisper`
now gives STT:

```text
internal/speech/artifact
  checksum-verified model/voice artifact cache shared by STT and TTS

internal/speech/tts or internal/wasi/piper/
  TTS engine manifest + model/voice artifact selection
  wazero runtime construction
  text normalization limits
  synthesize-to-audio implementation
  typed errors and redaction

internal/tools
  TTSRunner keeps provider selection and tool result envelope
  Go-owned provider adapter calls the speech runtime

internal/gateway / channels
  unchanged: consume MEDIA/audio delivery only
```

The first implementation slice should not change the model-facing
`text_to_speech` schema or Telegram delivery behavior. The shared artifact
cache is the first common STT/TTS building block; the first provider slice
should then add one provider adapter and prove it through fakeable
model/runtime seams. See `go-native-tts-source-study.md` for the current
engine-source evidence.

## Proof Gates

A builder-ready Go-owned TTS slice must prove:

1. no Python, Node, shell, or CGO dependency is required for the new provider;
2. model/voice artifacts are fetched or discovered through the shared
   `internal/speech/artifact` checksum-verified cache helpers and never
   committed to git;
3. the provider returns typed unavailable evidence when the artifact/runtime is
   missing;
4. a fixture synthesis path writes a supported audio file through Go code;
5. `text_to_speech` and gateway media delivery behavior remain unchanged;
6. benchmark data records load time, synthesis time, memory, output format,
   and binary-size delta before the provider becomes a default.

## Compatibility Boundary

- STT: keep WASI Whisper as the Go-owned path; keep HTTP STT fallback.
- TTS: add Go-owned WASM TTS as an opt-in/local provider first; keep cloud and
  command providers as compatibility fallbacks.
- Voice profiles: use profile `stt_provider` / `tts_provider` names to select
  Go-owned providers without exposing secrets in status JSON.
- Slim/lite builds: keep speech helpers excluded or degraded behind existing
  build-tag behavior.
