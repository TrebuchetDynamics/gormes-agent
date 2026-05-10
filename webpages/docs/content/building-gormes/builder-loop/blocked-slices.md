---
title: "Blocked Slices"
weight: 40
aliases:
  - /building-gormes/blocked-slices/
---

# Blocked Slices

This page is generated from canonical `progress.json` rows that declare
`blocked_by`.

Use it to avoid assigning work before the dependency chain is ready.

<!-- PROGRESS:START kind=blocked-slices -->
| Phase | Slice | Blocked by | Ready when | Unblocks |
|---|---|---|---|---|
| 4 / 4.D | Image input mode resolver + vision_analyze text fallback | Multimodal photo attachment passthrough | Multimodal photo attachment passthrough (2.B.12 row 6) is complete (true at HEAD = 0a7f8d02d)., Model-capability metadata is queryable from gateway code via internal/hermes (true — model_metadata.go)., A vision_analyze tool/seam exists OR the slice ships its first concrete one (e.g. wrapping the same provider stack with a vision-capable secondary model). | - |
| 5 / 5.E | Pure-Go Whisper transcribe one WAV | whisper.cpp WASI module discovery | Row B 'whisper.cpp WASI module discovery' is complete; whisper.wasm loads via wazero., ggml-tiny.en.bin model file is decided: either committed under testdata (~75 MB — acceptable for a one-time vendoring) or fetched in CI from a known SHA-256 (test-only, never in the runtime binary). | Wire Pure-Go Whisper into Telegram resolver, Whisper benchmark harness + perf budget |
| 5 / 5.E | Wire Pure-Go Whisper into Telegram resolver | Pure-Go Whisper transcribe one WAV | Row C 'Pure-Go Whisper transcribe one WAV' is complete; Transcriber API is stable. | Audio preprocessing and chunking pipeline |
| 5 / 5.E | Audio preprocessing and chunking pipeline | Wire Pure-Go Whisper into Telegram resolver | Row D 'Wire Pure-Go Whisper into Telegram resolver' is complete; the resolver is sending OGG bytes to a tempfile and Row E supplies the decoder/chunker that lives between. | Go-native audio decoding (ffmpeg replacement) |
| 5 / 5.E | Whisper benchmark harness + perf budget | Pure-Go Whisper transcribe one WAV | Row C 'Pure-Go Whisper transcribe one WAV' is complete; Transcriber API is stable. | - |
| 5 / 5.E | Go-native audio decoding (ffmpeg replacement) | Audio preprocessing and chunking pipeline | Row E 'Audio preprocessing and chunking pipeline' is complete and shipping ffmpeg-based preprocessing., Operator feedback or telemetry shows ffmpeg-absence is a real blocker on a target platform (Termux, Windows-portable, etc.). | - |
| 8 / 8.A | TD social presence connected to blog feed | TD engineering blog scaffolded and live | TD blog (8.A row 1) is live and emitting a feed., Operator has chosen a social platform and created the account. | - |
| 8 / 8.C | Engineering writeup #1: autonomous Hermes-porting loop | TD engineering blog scaffolded and live, Loop $/iteration cost metric in status file | TD blog (8.A row 1) is live., Loop $/iteration cost telemetry (8.F) has at least one week of data., Operator has decided the publication date and platform (HN/Lobsters/Reddit). | Engineering writeup #2: validation-gated agentic engineering, Engineering writeup #3: Gormes vs Hermes-Python benchmarks, HN launch post for Gormes 1.0 |
| 8 / 8.D | Single-binary cross-platform release pipeline | Sharp v1.0 differentiator decision | Sharp v1.0 differentiator (8.D row 1) is decided., go build ./cmd/gormes succeeds on all seven target GOOS/GOARCH combinations from CI or a Linux runner where cross-compilation is supported. | - |
<!-- PROGRESS:END -->
