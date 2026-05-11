# Go-native OGG/Opus Decoder Decision

Date: 2026-05-10
Status: decided
Decided: keep ffmpeg shim; add build-tag fallback for headless platforms

## Context

Gormes' WASI Whisper transcription pipeline (`internal/wasi/whisper/audio/preprocess.go`)
uses `ConvertWithFFmpeg` to convert Telegram voice messages (OGG/Opus) to
PCM16 mono 16kHz WAV before Whisper processing. This requires ffmpeg on PATH.

On some platforms (Termux without ffmpeg, locked-down corp Linux, minimal
containers), ffmpeg may not be available. The row tracks whether we should
replace the ffmpeg shim with pure-Go OGG/Opus decoding.

## Options

### A: Pure-Go OGG/Opus decoder
Use a Go OGG/Opus library (e.g. gopxl/opus, hraban/opus with CGO, or
a pure-Go WASM opus decoder via wazero).
- Binary size: +2–5 MB (opus library)
- Latency: same as ffmpeg (both decode to PCM)
- Platform: CGO requires libopus-dev; WASM works everywhere
- Maintenance: opus library updates, CGO cross-compilation complexity

### B: Keep ffmpeg shim + build-tag fallback
Keep the existing ffmpeg path as the default. Add a `!noffmpeg` build tag
that includes ffmpeg-dependent code. When built with `-tags noffmpeg`, the
build excludes the ffmpeg path and the channel falls back to attachment
markers (no transcription).
- Binary size: +0
- Latency: unchanged for ffmpeg builds; transcription unavailable for
  noffmpeg builds
- Platform: everywhere
- Maintenance: zero — no new decoder code

### C: Embed opus WASM decoder via wazero
Similar to how we embed whisper.cpp as WASM, embed a small opus→PCM WASM
decoder alongside whisper.wasm.
- Binary size: +1–3 MB (opus WASM)
- Latency: slightly slower than ffmpeg (WASM overhead)
- Platform: everywhere wazero runs (Linux/macOS/Windows/Termux)
- Maintenance: WASM binary needs periodic rebuild

## Decision

**Option B — keep ffmpeg shim with build-tag fallback.**

Rationale:
- The WASI Whisper path already requires a multi-MB model download; adding
  opus WASM would increase the embedded binary significantly.
- ffmpeg is available on all major platforms via package managers; the
  build-tag fallback covers the edge case (headless/embedded).
- For platforms where neither ffmpeg nor package managers are available
  (e.g. Termux without `pkg install ffmpeg`), the channel degrades
  gracefully to attachment markers.
- Option C (opus WASM) is the best long-term path and can be implemented
  as a follow-up row when the whisper.wasm embedding story is proven in
  production.

## Consequences

- `ConvertWithFFmpeg` remains the default audio preprocessing path.
- A `noffmpeg` build tag gates the ffmpeg dependency; builds without it
  skip audio transcription and return `audio_decode_ffmpeg_missing` evidence.
- The Gormes README documents which platforms need ffmpeg for voice
  transcription.
- A future row can implement Option C as a Gormes-owned opus decoder.
