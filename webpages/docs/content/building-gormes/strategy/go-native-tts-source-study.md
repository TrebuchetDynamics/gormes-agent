# Go-Native TTS Source Study

Date: 2026-05-24
Status: source-backed implementation guidance

## Question

Which source path can move Gormes toward an owned Go TTS solution without
reintroducing hidden Python, Node, shell, CGO, Docker, or cloud dependencies?

## Evidence Summary

| Candidate | Evidence | Fit for Gormes |
|---|---|---|
| `rhasspy/piper` | `https://github.com/rhasspy/piper` currently points development to `OHF-Voice/piper1-gpl`; historical `LICENSE.md` is MIT. | Useful lineage and voice format, but not a direct current engine source by itself. |
| `OHF-Voice/piper1-gpl` | `README.md` describes Piper as a Python package embedding `espeak-ng`; `COPYING` is GPL-3.0; `libpiper/CMakeLists.txt` builds `espeak-ng` and links ONNX Runtime; `docs/BUILDING.md` uses Python/scikit-build. | Not suitable as the first in-repo provider because it changes licensing/redistribution risk and reintroduces native build/Python mechanics. |
| `espeak-ng` | `README.md` says it is compact, formant synthesis, written in C, and supports WAV output; `COPYING` is GPL-3.0. | Good phonemizer evidence, but not a Go-owned neural TTS backend and license/build shape is not ideal for Gormes' static-binary goal. |
| `k2-fsa/sherpa-onnx` | `README.md` lists TTS, Go, and WebAssembly support; `go-api-examples/non-streaming-tts/main.go` uses `sherpa.NewOfflineTts`; `wasm/tts/CMakeLists.txt` exports TTS C APIs and preloads model assets into an Emscripten bundle. | Promising reference for a future WASM/ONNX path, but the shipped Go package is CGO/native-library based and the current wasm target is browser/Emscripten shaped, not immediately wazero/WASI-ready. |
| `k2-fsa/sherpa-onnx-go` | `go.mod` depends on platform packages; `sherpa_onnx_linux.go` re-exports TTS types; `sherpa-onnx-go-linux/build_linux_amd64.go` uses `#cgo` and native `onnxruntime`. | Strong Go API reference, but fails the no-CGO/static-binary gate for the default Gormes local provider. |
| `GetcharZp/go-speech` | `README.md` describes Go + ONNX TTS/ASR; `tts/pipertts/engine.go` runs Piper ONNX through `onnxruntime_purego`; `onnx.go` requires a local ONNX Runtime dynamic library path. | Useful donor for Go-side Piper tensor flow and purego loading, but still requires external native dynamic libraries, so it is not the final single-binary path. |
| `amitybell/piper` | `README.md` calls it an embedded distribution of Piper for Go; `go.mod` depends on platform-specific Piper binary and voice modules. | Good packaging lesson, but it embeds/ships native command binaries rather than owning synthesis in Go. |
| `stnmrshx/go-piper` | `README.md` describes CGO bindings over Piper and requires submodules/native libraries. | Not compatible with the default no-CGO goal. |

## Decision For The Next Slice

Do **not** pick a native/CGO Piper or sherpa-onnx Go package as the default
Gormes provider yet. The safe next implementation is the shared speech artifact
cache used by both STT and future TTS:

- `internal/speech/artifact` owns checksum-verified model/voice artifact
  discovery and atomic partial-file cleanup.
- `internal/wasi/whisper` adapts the shared cache while preserving its existing
  `ModelCacheError` codes and public STT behavior.
- Future TTS providers can reuse the same cache for `.onnx`, `.onnx.json`,
  phonemizer data, WASM runtime, or voice packs without copying whisper-specific
  model-download code.

This narrows the TTS engine blocker: the remaining open choice is **runtime
execution** (a wazero/WASI-friendly TTS engine artifact or a constrained purego
ONNX runtime strategy), not artifact lifecycle.

## Rejected For First Provider

- Shelling out to `piper`, `edge-tts`, platform `say`, or downloaded native
  command binaries: keeps Gormes dependent on external executables.
- CGO bindings to Piper or sherpa-onnx: conflicts with `CGO_ENABLED=0` and
  single-binary release goals.
- Python package embedding: conflicts with the no-venv/no-pip operator target.
- GPL engine embedding without a release/legal plan: unsafe for default binary
  distribution.

## Next Engine Sourcing Question

Find or produce one small TTS fixture engine that is safe to run under Go
without CGO:

1. a wazero-compatible WASI/Emscripten TTS artifact with a tiny fixture model;
2. or a purego ONNX runtime path whose native-library dependency is explicit,
   checksum-pinned, and optional rather than default;
3. or a deliberately low-quality but fully Go-owned formant fallback for offline
   proof before neural quality work.
