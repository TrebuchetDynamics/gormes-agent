---
title: "Agent Queue"
weight: 20
aliases:
  - /building-gormes/agent-queue/
---

# Agent Queue

This page is generated from the canonical progress file:
`docs/content/building-gormes/architecture_plan/progress.json`.

It lists unblocked, non-umbrella contract rows that are ready for a focused
skill-driven implementation attempt. Each card carries the execution owner,
slice size, contract, trust class, degraded-mode requirement, fixture target,
write scope, test commands, done signal, acceptance checks, and source
references.

Shared skill handoff facts live in [Skill Builder Handoff](../builder-loop-handoff/):
the main skill entrypoint, plan, candidate source, generated docs, tests, and
candidate policy. Keep those control-plane facts in `meta.builder_loop`, and
keep row-specific execution facts in `progress.json`.

If the generated list is empty, do not switch to an ad hoc TODO list. Route
through `gormes-planner`, repair one planned/draft row until it satisfies the
handoff contract, validate `progress.json`, and then return to builder
selection.

<!-- PROGRESS:START kind=agent-queue -->
## 1. Go-owned local TTS runtime seam + fixture fallback

- Phase: 5 / 5.E
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Implement the first Go-owned local TTS backend behind the existing text_to_speech/TTSProvider contract as a fakeable runtime seam plus a deliberately low-quality pure-Go fixture/formant provider. This clears the runtime-source blocker without pretending Piper/ONNX/WASI neural quality is solved: the provider must synthesize a valid WAV through Go code with no Python, Node, shell command, CGO, native ONNX Runtime, browser/Emscripten runtime, or cloud dependency. Neural Piper/WASI work remains a later source-backed upgrade behind the same seam.
- Trust class: operator, gateway, system
- Ready when: The first slice is explicitly scoped to the runtime interface and a fixture-quality pure-Go synthesizer; it does not claim neural Piper/ONNX/WASI quality., Tests can synthesize a small WAV through fakeable Go seams without live cloud credentials, microphone/speaker devices, Python, Node, shell TTS binaries, CGO, native shared libraries, or browser runtimes., The existing TTSRunner/text_to_speech schema and gateway MEDIA delivery behavior remain the public contract; the new backend is one provider adapter behind that seam.
- Not ready when: The plan embeds large voice/model artifacts directly in git or the default binary instead of using checksum-verified artifact cache/discovery helpers for future neural backends., The provider shells out to piper, edge-tts, Python, Node, platform TTS, or a downloaded native command and calls that Go-owned., The implementation depends on CGO, native ONNX Runtime shared libraries, WasmEdge host extensions, or browser/Emscripten APIs as the default runtime path., The change rewrites gateway/channel media delivery or the model-facing text_to_speech schema before a provider adapter works behind the existing seam.
- Degraded mode: Missing fixture/runtime support, unsupported text, disabled build tags, synthesis errors, and benchmark budget overruns return typed tts_provider_unavailable or tts_api_error evidence without changing gateway delivery, leaking temp paths, or silently shelling out to Python/Node/platform commands. Provider=auto may still use existing cloud/command compatibility fallbacks; explicitly selecting the Go-owned provider must not fall through silently.
- Fixture: `internal/tools/tts_go_native_provider_test.go; internal/speech/tts/fixture_test.go`
- Write scope: `internal/speech/tts/`, `internal/tools/tts_tool.go`, `internal/tools/tts_providers.go`, `internal/tools/tts_provider_matrix_test.go`, `internal/tools/tts_go_native_provider_test.go`, `cmd/gormes/registry_audio.go`, `internal/config/profile_config_v2.go`, `internal/config/profile_voice_profile_test.go`, `docs/content/building-gormes/strategy/tts-decision.md`, `docs/content/building-gormes/strategy/go-native-speech-options.md`, `docs/content/building-gormes/strategy/go-native-tts-source-study.md`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/speech/tts -count=1`, `go test ./internal/tools -run 'Test(TTSGoNative\|TTSProviderMatrix\|TextToSpeech)' -count=1`, `go test ./internal/config -run TestProfileVoiceProfile -count=1`, `CGO_ENABLED=0 go test ./internal/speech/tts ./internal/tools ./cmd/gormes -run 'Test(TTS\|TextToSpeech\|RegistryAudio)' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: A user can configure a Go-owned local fixture TTS provider, run text_to_speech, and receive a valid WAV synthesized through Go code without Python, Node, shell commands, CGO, native shared libraries, browser runtimes, or cloud credentials; neural-quality backend work remains explicitly separated behind the same runtime seam.
- Acceptance: `internal/speech/tts` (or an equivalently named package) defines a small runtime interface, typed unavailable/synthesis errors, text limits, WAV output metadata, and redaction rules for Go-owned local synthesis., A provider such as `local_fixture` or `local_go` registers in the existing TTS provider matrix and can be selected by `TTSConfig.Provider` / profile voice settings without changing the tool schema., Unit tests prove the provider succeeds through Go-only fakeable runtime seams and writes a valid WAV file without invoking shell commands, Python, Node, CGO, native shared libraries, browser runtimes, or live cloud APIs., Missing/disabled runtime cases return typed redacted evidence and do not fall through to command providers unless provider selection is `auto` and a compatibility fallback is explicitly available., Benchmark metadata captures fixture load time, synthesis time, memory, output format, and binary-size delta; the row documents that neural-quality Piper/WASI remains a later upgrade, not the default local voice yet.
- Source refs: docs/content/building-gormes/strategy/go-native-speech-options.md, docs/content/building-gormes/strategy/tts-decision.md, docs/content/building-gormes/strategy/go-native-tts-source-study.md, internal/tools/tts_tool.go:TTSProvider/TTSRunner, internal/tools/tts_providers.go:LazyLocalTTSProvider, internal/tools/tts_command_provider.go:TTSCommandProvider, internal/speech/artifact/cache.go:Ensure, docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md:Voice, TTS, transcription, https://github.com/Mintplex-Labs/piper-tts-web/: browser-only Piper/ONNX WASM reference, not a wazero-ready backend, https://github.com/shota3506/onnxruntime-purego: purego binding still requires native ONNX Runtime shared library, https://github.com/second-state/WasmEdge-WASINN-examples/tree/master/wasmedge-piper: WASI-NN/Piper reference tied to WasmEdge host extensions, https://espeak.sourceforge.net/: compact formant-synthesis reference; C/GPL, use as concept evidence only
- Unblocks: Go-native continuous voice mode defaults, Profile-scoped voice profiles with local TTS fallback, Navivox BYO STT/TTS profile implementation
- Why now: Unblocks Go-native continuous voice mode defaults, Profile-scoped voice profiles with local TTS fallback, Navivox BYO STT/TTS profile implementation.

<!-- PROGRESS:END -->
