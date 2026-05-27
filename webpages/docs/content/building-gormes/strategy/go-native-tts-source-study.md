# Go-Native TTS Source Study

Date: 2026-05-24
Status: source-backed implementation guidance (updated 2026-05-26)

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
| `Mintplex-Labs/piper-tts-web` / `diffusion-studio/vits-web` | Browser Piper TTS through ONNX Runtime Web and local WASM paths; project notes it is a frontend library and will not work with NodeJS. | Good proof that Piper+ONNX can run in a web WASM environment, but not a wazero/WASI backend for Gormes. |
| `second-state/WasmEdge-WASINN-examples/wasmedge-piper` | Piper example exists for WasmEdge WASI-NN. | Useful WASI-NN reference, but it depends on WasmEdge host extensions rather than the current wazero runtime. |
| `shota3506/onnxruntime-purego` | Pure Go binding surface over ONNX Runtime using `purego`; README still requires an installed `libonnxruntime`/DLL/shared library. | Possible optional future strategy, but not the default static/no-native-library provider. |
| eSpeak/formant synthesis | eSpeak documents compact formant synthesis, WAV output, many languages, and a small footprint, but is C/GPL. | Use only as concept evidence for a deliberately low-quality Go-owned fixture fallback; do not embed eSpeak code or data. |

## Decision For The Next Slice

Do **not** pick a native/CGO Piper, browser-only Piper WASM, WasmEdge
WASI-NN Piper, or purego-ONNX shared-library path as the default Gormes
provider yet. The safe implementation path is now two staged pieces. The first
(shared speech artifact cache) is already complete. The next builder-ready slice
is a Go-owned local TTS runtime seam plus a deliberately low-quality pure-Go
fixture/formant fallback that proves synthesis, WAV output, provider selection,
error redaction, and benchmark metadata without claiming neural Piper quality.

The already-completed cache slice provides:

- `internal/speech/artifact` owns checksum-verified model/voice artifact
  discovery and atomic partial-file cleanup.
- `internal/wasi/whisper` adapts the shared cache while preserving its existing
  `ModelCacheError` codes and public STT behavior.
- Future TTS providers can reuse the same cache for `.onnx`, `.onnx.json`,
  phonemizer data, WASM runtime, or voice packs without copying whisper-specific
  model-download code.

This resolves the immediate builder-selection blocker by narrowing the runtime
choice: implement the Go runtime interface and fixture-quality provider first;
keep neural Piper/ONNX/WASI work as a later upgrade behind that same interface.
The fixture provider is intentionally not the final voice-quality target, but it
prevents the row from waiting on an unsourced external engine while preserving
the no-Python/no-Node/no-shell/no-CGO contract.

## Rejected For First Provider

- Shelling out to `piper`, `edge-tts`, platform `say`, or downloaded native
  command binaries: keeps Gormes dependent on external executables.
- CGO bindings to Piper or sherpa-onnx: conflicts with `CGO_ENABLED=0` and
  single-binary release goals.
- Python package embedding: conflicts with the no-venv/no-pip operator target.
- GPL engine embedding without a release/legal plan: unsafe for default binary
  distribution.

## Selected Runtime Source For Builder Handoff

Selected for the next builder pass: **Gormes-owned fixture runtime seam**. The
implementation should create an internal TTS runtime interface and a tiny
Go-only synthesizer that writes valid WAV output for bounded text. It may be
robotic/formant-like and English-only; quality is explicitly degraded. It must
not import eSpeak code/data, native ONNX Runtime, browser WASM runtimes,
WasmEdge-only extensions, command providers, Python, Node, or CGO.

Builder acceptance should prove:

1. provider selection reaches the new Go-owned provider through the existing
   `TTSProvider`/`TTSRunner` seam;
2. tests synthesize a valid WAV from Go code only;
3. missing/disabled runtime paths return typed, redacted unavailable evidence;
4. `Provider=auto` fallback behavior remains explicit and does not mask an
   explicitly selected local provider failure;
5. benchmark metadata records load time, synthesis time, memory, format, and
   binary-size delta.

## Next Engine Sourcing Question

Find or produce one small TTS fixture engine that is safe to run under Go
without CGO:

1. a wazero-compatible WASI/Emscripten TTS artifact with a tiny fixture model;
2. or a purego ONNX runtime path whose native-library dependency is explicit,
   checksum-pinned, and optional rather than default;
3. or a deliberately low-quality but fully Go-owned formant fallback for offline
   proof before neural quality work.
