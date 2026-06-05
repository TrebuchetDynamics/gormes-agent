# TTS/STT Architecture

This document inventories the current Gormes contracts related to text-to-speech (TTS), speech-to-text (STT), voice mode, audio ingress, and audio media delivery.

## Contract Sources

Primary planning contracts:

- `webpages/docs/content/building-gormes/architecture_plan/progress.json/modules/tts.json`
- `webpages/docs/content/building-gormes/architecture_plan/progress.json/modules/stt.json`

Primary runtime packages:

- `internal/tools/tts/`
- `internal/tools/transcription/`
- `internal/adapters/channels/telegram/audio/`
- `internal/gateway/delivery/media/`
- `internal/config/audio/`
- `internal/config/profile/`
- `internal/gateway/tts_command.go`
- `internal/gateway/auto_tts.go`

Upstream Hermes behavior anchors:

- `hermes-agent/tools/tts_tool.py`
- `hermes-agent/tools/transcription_tools.py`
- `hermes-agent/tools/voice_mode.py`
- `hermes-agent/gateway/run.py`
- `hermes-agent/gateway/platforms/telegram.py`
- `hermes-agent/hermes_cli/setup.py`
- `hermes-agent/hermes_cli/voice.py`
- `hermes-agent/tui_gateway/server.py`

## Runtime Architecture

```text
User / Gateway / Tool Call
        │
        ├── STT ingress: Telegram voice/audio → audio transcriber → text turn
        │
        ├── Model-visible STT tool: transcribe_audio → TranscriptionRunner → STT providers
        │
        ├── Model-visible TTS tool: text_to_speech → TTSRunner → TTS providers
        │
        └── Final assistant text → MEDIA tag extraction → native channel media send
```

## TTS Tool Contract

Runtime file: `internal/tools/tts/tool.go`

Model-visible tool:

- `text_to_speech`

Input contract:

- `text`
- `output_path`

Internal request fields also support:

- `provider`
- `platform`
- `voice`
- `speed`

Result envelope:

- `success`
- `file_path`
- `media_tag`
- `provider`
- `voice_compatible`
- `truncated`
- `evidence`
- `error`

Stable evidence codes:

- `tts_synthesized`
- `tts_disabled`
- `tts_invalid_arguments`
- `unsupported_audio_format`
- `tts_provider_unavailable`
- `tts_api_error`
- `tts_output_missing`

Provider seam:

```go
type TTSProvider interface {
    Available(context.Context) bool
    Synthesize(context.Context, TTSProviderRequest) (TTSProviderResult, error)
}
```

Provider selection contract:

- Default provider is `edge` when available.
- Explicit provider selection must not silently fall back on failure.
- `auto` tries built-in/cloud/local providers in configured order.
- `local` maps to Go-owned/local fixture/neural-compatible local candidates.
- Command providers cannot shadow built-in provider names.

## STT Tool Contract

Runtime file: `internal/tools/transcription/tool.go`

Model-visible tool:

- `transcribe_audio`

Input contract:

- `audio_path`
- `provider`
- `model`
- `language`
- `format`

Result envelope:

- `success`
- `transcript`
- `provider`
- `model`
- `language`
- `evidence`
- `error`

Stable evidence codes:

- `stt_transcribed`
- `stt_disabled`
- `audio_not_found`
- `audio_not_file`
- `unsupported_audio_format`
- `audio_too_large`
- `stt_provider_unavailable`
- `stt_api_error`
- `stt_invalid_arguments`

Provider seam:

```go
type TranscriptionProvider interface {
    Available(context.Context) bool
    Transcribe(context.Context, TranscriptionProviderRequest) (TranscriptionProviderResult, error)
}
```

Provider selection contract:

- Auto order: `local`, `local_command`, `groq`, `openai`, `mistral`, `xai`.
- Explicit provider selection must not silently fall back.
- Local and cloud provider model defaults normalize through the runner.
- File validation happens before provider invocation.

## Telegram STT Ingress Contract

Runtime file: `internal/adapters/channels/telegram/audio/transcriber.go`

Telegram voice/audio messages flow through a small adapter seam:

```go
type Transcriber interface {
    Transcribe(ctx context.Context, audio Input) (string, error)
}
```

`Input` carries sanitized media evidence:

- `kind`
- `file_id`
- `media_type`
- `file_name`
- `duration`
- `data`

Contract:

- Voice/audio-only Telegram updates must never become blank turns.
- Captions are preserved and media markers are appended.
- Attachment descriptors include kind, media type, file name where present, and source evidence.
- Telegram file IDs, bot tokens, and token-bearing file URLs must not leak.
- Download/transcription failures degrade to readable markers and sanitized error evidence.

Transcriber priority:

```text
local CLI whisper → in-binary WASI Whisper → HTTP STT fallback → nil
```

## WASI Whisper / Local STT Contract

Runtime packages:

- `internal/wasi/whisper/`
- `internal/wasi/whisper/audio/`
- `internal/tools/whisper*`

Contracts:

- `Transcriber` API supports `NewTranscriber`, `TranscribeWAV`, and `Close`.
- WASI Whisper preserves `CGO_ENABLED=0` and avoids Python/faster-whisper runtime dependencies.
- `ggml-tiny.en.bin` is resolved through a checksum-verified cache; it is not committed to git or embedded in the binary.
- Audio preprocessing accepts compatible 16 kHz mono PCM WAV directly.
- Non-WAV input converts through ffmpeg-style conversion unless unavailable.
- Missing conversion returns typed `audio_preprocess_unavailable` evidence.
- Chunking is fixed-window and offline/batched, not streaming.

## TTS Media Delivery Contract

Runtime file: `internal/gateway/delivery/media/media.go`

Hermes-compatible media tags:

- `MEDIA:/path/to/file`
- `[MEDIA:/path/to/file]`
- `[[audio_as_voice]]\nMEDIA:/path/to/audio`

Contract:

- Final assistant text may contain media tags.
- Gateway extracts valid media paths into channel-neutral media records.
- Visible user text strips raw `MEDIA:` syntax.
- Unsafe or unsupported media paths are redacted.
- Audio marked `[[audio_as_voice]]` is delivered as voice-compatible media when the channel supports it.

Media evidence codes:

- `media_extracted`
- `media_ignored`

## Gateway Auto-TTS Contract

Runtime files:

- `internal/gateway/auto_tts.go`
- `internal/gateway/tts_command.go`

Contract:

- Per-session `/tts` settings control automatic voice replies.
- Final assistant text can be synthesized to audio when the user requested audio or session TTS is enabled.
- Auto-TTS uses the registered `text_to_speech` tool.
- If the tool fails or returns no deliverable media, gateway logs bounded evidence and keeps text delivery.

TTS engines:

- `openai`
- `elevenlabs`
- `edge`
- `local`
- `disabled`

Speeds:

- `slow`
- `normal`
- `fast`
- `very-fast`

## Config Contracts

### STT and Voice Config

Runtime file: `internal/config/audio/config.go`

```toml
[stt]
enabled = true
provider = "local"

[stt.local]
model = "base"
language = ""

[stt.openai]
model = "whisper-1"

[voice]
record_key = "ctrl+b"
```

### Profile Voice Config

Runtime file: `internal/config/profile/config.go`

Profile voice fields:

- `stt_provider`
- `tts_provider`
- `voice_id`
- `language_policy`
- `fallback_voice`
- `stt_credential`
- `tts_credential`

Provider matrix:

- `stt`
- `tts`

## Planning Contract Inventory

### TTS module rows

All are in `webpages/docs/content/building-gormes/architecture_plan/progress.json/modules/tts.json`.

- Voice mode port
- Voice mode environment detector + audio provider seam
- Transcription tool contract
- Telegram voice/audio STT ingress hook
- TTS tool contract + media delivery seam
- MiniMax TTS v1 `text_to_speech` raw-audio compatibility
- TTS provider matrix + dotenv/command-provider resolution
- TTS synthesis + voice-mode state
- Voice record-key config binding for native TUI
- Telegram voice STT HTTP-provider fallback
- Pure-Go STT exploration
- wazero WASI smoke harness
- whisper.cpp WASI module discovery
- Pure-Go Whisper transcribe one WAV
- Whisper tiny.en model cache fetcher
- Wire Pure-Go Whisper into Telegram resolver
- WASI Whisper ffmpeg preprocess + fixed-window chunker
- Audio preprocessing and chunking pipeline
- Whisper benchmark harness + perf budget
- Go-native OGG/Opus decoder decision
- Go-native OGG/Opus decoder implementation
- Pure-Go TTS decision research
- Shared speech artifact cache for Go-owned TTS
- Go-owned local TTS runtime seam + fixture fallback
- Gormes setup terminal TTS and agent-settings section bindings

### STT module rows

All are in `webpages/docs/content/building-gormes/architecture_plan/progress.json/modules/stt.json`.

- Telegram voice/audio inbound attachment markers
- Hermes/Honcho Go runtime plan second-wave reconciliation
- Session auto-reset + STT config parity
- Transcribe audio tool registration + local whisper provider

## Safety and Degradation Rules

Across TTS/STT contracts:

- No live cloud calls in unit tests.
- No live Telegram token required in tests.
- No Python, Node, shell command, CGO, browser runtime, or native shared library is allowed for Go-owned local TTS/STT contracts unless explicitly row-scoped.
- Explicit provider selection must not silently fall back.
- Auto mode may fall back in the documented provider order.
- Secrets, bot tokens, file IDs, credential-shaped strings, provider URLs with credentials, local command output, and raw HTTP bodies must be redacted or bounded.
- Missing optional audio/runtime dependencies degrade with typed evidence instead of panics.
- `MEDIA:` tags must not leak in final operator-visible channel text after native media delivery.
