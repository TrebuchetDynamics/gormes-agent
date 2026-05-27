---
title: "Next Slices"
weight: 30
aliases:
  - /building-gormes/next-slices/
---

# Next Slices

This page is generated from the canonical progress file and lists the highest
leverage contract-bearing roadmap rows to execute next.

The ordering is:

1. unblocked `P0` handoffs;
2. active `in_progress` rows;
3. `fixture_ready` rows;
4. unblocked rows that unblock other slices;
5. remaining `draft` contract rows.

Use this page when choosing implementation work. If a row is too broad, split
the row in `progress.json` before assigning it.

If no slices are listed, the next correct action is planner work: choose one
planned row from `progress.json` or a phase page and add enough contract detail
for it to appear here. Do not infer that an empty generated list means the
roadmap is complete.

<!-- PROGRESS:START kind=next-slices -->
| Phase | Slice | Contract | Trust class | Fixture | Why now |
|---|---|---|---|---|---|
| 5 / 5.E | Go-owned local TTS runtime seam + fixture fallback | Implement the first Go-owned local TTS backend behind the existing text_to_speech/TTSProvider contract as a fakeable runtime seam plus a deliberately low-quality pure-Go fixture/formant provider. This clears the runtime-source blocker without pretending Piper/ONNX/WASI neural quality is solved: the provider must synthesize a valid WAV through Go code with no Python, Node, shell command, CGO, native ONNX Runtime, browser/Emscripten runtime, or cloud dependency. Neural Piper/WASI work remains a later source-backed upgrade behind the same seam. | operator, gateway, system | `internal/tools/tts_go_native_provider_test.go; internal/speech/tts/fixture_test.go` | Unblocks Go-native continuous voice mode defaults, Profile-scoped voice profiles with local TTS fallback, Navivox BYO STT/TTS profile implementation. |
<!-- PROGRESS:END -->
