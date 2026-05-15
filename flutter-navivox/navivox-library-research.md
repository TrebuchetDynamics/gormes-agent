# Navivox Library Research

Status: planning draft
Source: web research + pub.dev docs (2025-2026)

## 1. Core Libraries — Platform Support Matrix

### 1.1 SSH & Terminal

| Library | Purpose | Android | iOS | Linux | Windows | macOS | Stars | Issues | Notes |
|---------|---------|---------|-----|-------|---------|-------|-------|--------|-------|
| **dartssh2** ^2.17 | SSH client | ✅ | ✅ | ✅ | ✅ | ✅ | 140+ | Active | Pure Dart, supports all auth methods, SFTP, forwarding. Key pair from PEM. `compute()` for passphrase decrypt in Flutter. |
| **xterm.dart** ^4.0 | Terminal widget | ✅ | ✅ | ✅ | ✅ | ✅ | 634 | 91 | 60fps, CJK/emoji, IME support. SSH example in ~100 lines with dartssh2. |

SSH tips:
- `SSHKeyPair.fromPem(pemString, passphrase)` supports encrypted keys
- Use `compute()` in Flutter for passphrase decryption to avoid UI jank
- `client.execute('gormes navivox serve --stdio')` returns a session with stdin/stdout streams
- Host key verification: access `socket.remoteHostKey` after connect, compute SHA256 fingerprint
- `client.run('which gormes')` for Gormes probe

Terminal tips:
- `TerminalView(terminal)` renders the terminal widget
- `terminal.onOutput` captures user key presses; pipe to SSH stdin
- `terminal.onResize` sends PTY resize to SSH session
- The xterm.dart SSH example (github.com/TerminalStudio/xterm.dart/blob/master/example/lib/ssh.dart) is the canonical starting point
- Keep terminal sessions visually distinct from chat — use separate `Terminal` instance per session

### 1.2 Storage

| Library | Purpose | Android | iOS | Linux | Windows | macOS | Notes |
|---------|---------|---------|-----|-------|---------|-------|-------|
| **flutter_secure_storage** ^10 | Secure key/value | ✅ KeyStore | ✅ Keychain | ✅ libsecret | ✅ DPAPI | ✅ Keychain | Requires gnome-keyring or kwallet on Linux |
| **drift** ^2.x | SQLite database | ✅ | ✅ | ✅ | ✅ | ✅ | Reactive, type-safe, formerly Moor |
| **sqlite3_flutter_libs** | SQLite runtime | ✅ | ✅ | ✅ | ✅ | ✅ | Bundles SQLite for all platforms |

Secure storage tips:
- Linux: Requires `libsecret-1-0` + `libjsoncpp1` runtime deps, and `gnome-keyring` or `kwalletmanager` running
- Linux v10+ uses non-pretty JSON to avoid Gnome Keyring empty-password bug
- For large key blobs that don't fit in secure storage limits, store encrypted blob on disk with wrapping key in secure storage
- Test read-back on all platforms early; Linux `null` returns are common when secret service isn't running
- Surface clear setup errors when secure storage is unavailable

Drift tips:
- Use `driftDatabase(name:)` for cross-platform database file
- Tables defined as Dart classes extending `Table`, auto-generate companion classes
- Use `.watch()` for reactive streams that auto-update UI
- DAOs for encapsulated query logic
- Schema migrations via `schemaVersion` bump + `MigrationStrategy.onUpgrade`
- Use `@UseRowClass` to bind domain models to table rows
- Managers API for common CRUD: `database.managers.serversTable.filter(...).watch()`

### 1.3 State & Navigation

| Library | Purpose | Notes |
|---------|---------|-------|
| **riverpod** ^2.x | State management | Provider-based, testable, code-gen optional |
| **go_router** ^14.x | Navigation | Official Flutter team package, deep links, shell routes |

Riverpod + GoRouter tips:
- Create a `routerProvider` that `watch`es auth/connection state
- Pass a `refreshListenable` (ChangeNotifier) to GoRouter so it re-evaluates redirects
- `redirect` callback returns `null` to allow, or a path string to redirect
- Use ShellRoute for persistent bottom nav / sidebar
- Gate config/admin routes in the `redirect` callback using role providers
- Avoid `state.extra` for navigation data — use path/query params for deep link compatibility

### 1.4 Code Generation

| Library | Purpose | Notes |
|---------|---------|-------|
| **freezed** ^2.x | Immutable data classes, unions/sealed | `copyWith`, `==`, `hashCode`, JSON support |
| **json_serializable** ^6.x | JSON codec generation | Works with freezed for `fromJson`/`toJson` |
| **build_runner** ^2.x | Code generation orchestrator | `dart run build_runner build --delete-conflicting-outputs` |

Code gen tips:
- Freezed `sealed` classes for protocol events (union types per event kind)
- `@JsonKey(name: 'snake_case')` for server field mapping
- Custom `fromJson` when server data is untrustworthy (external APIs)
- `build.yaml` config: `freezed` runs before `json_serializable`
- Use `include_if_null: false` and `explicit_to_json: true` globally in `build.yaml`
- Generated files committed to Git (standard practice for most teams)

### 1.5 Voice & Audio

| Library | Purpose | Android | iOS | Linux | Windows | macOS | Notes |
|---------|---------|---------|-----|-------|---------|-------|-------|
| **record** ^6.x | Microphone capture | ✅ MediaRecorder/Codec | ✅ AVFoundation | ✅ parecord+ffmpeg | ✅ MediaFoundation | ✅ AVFoundation | Stream PCM 16-bit @ 16kHz mono |
| **speech_to_text** ^6.x | Local STT | ✅ Google/on-device | ✅ SFSpeechRecognizer | ❌ | ✅ WinRT | ✅ | Commands/short phrases only; not continuous |
| **just_audio** ^0.9.x | Audio playback | ✅ | ✅ | ✅ | ✅ | ✅ | StreamAudioSource for real-time TTS |
| **flutter_tts** | Local TTS | ✅ | ✅ | ❌ limited | ✅ | ✅ | Not primary TTS; command confirmations only |
| **vosk_flutter** | Offline STT | ✅ | ❌ | ✅ | ✅ | ❌ | Linux offline fallback, requires model download |
| **runanywhere_onnx** | On-device STT/TTS/VAD | ✅ | ✅ | ❌ | ❌ | ❌ | Whisper STT + Piper TTS; mobile only |

Voice/audio tips:
- `record` streaming: `await recorder.startStream(RecordConfig(encoder: AudioEncoder.pcm16bits, sampleRate: 16000, numChannels: 1))`
- `record` Linux requires: `sudo apt install pulseaudio-utils ffmpeg`
- `speech_to_text` returns partial results with `isFinal` flag; use for wake word detection
- `speech_to_text` Linux/Win support is build-only (not functional) per official docs — use Vosk fallback
- `just_audio` `StreamAudioSource` supports real-time streaming via local HTTP proxy
- For low-latency TTS streaming, create a `StreamAudioSource` subclass that feeds chunks as they arrive from server
- `flutter_tts` is NOT reliable on Linux desktop — prefer server TTS
- Microphone permissions: `permission_handler` for runtime, platform manifest entries for build
- Wake/control word "NAVI" detection: use `speech_to_text` partial results, match regex/confidence

### 1.6 Security & Crypto

| Library | Purpose | Notes |
|---------|---------|-------|
| **local_auth** ^3.x | Biometric/PIN gate | Fingerprint, Face ID, device PIN. `biometricOnly` flag. |
| **cryptography** ^2.x | Ed25519, crypto primitives | Key generation, signing, verification |
| **permission_handler** ^11.x | Runtime permissions | Mic, notifications, files |
| **ssh_key** | SSH key format encode/decode | OpenSSH, RFC 4716, PKCS formats |

Crypto tips:
- `Ed25519().newKeyPair()` for key generation
- Convert to OpenSSH format using `ssh_key` package for `.pub` files
- `dartssh2`'s `SSHKeyPair.fromPem()` handles OpenSSH private key format directly
- Verify generated keys with `ssh-keygen -y -f keyfile` in tests
- `local_auth.authenticate(localizedReason:, biometricOnly: true)` for secret editor gate
- Linux biometric: not available; use app-level PIN fallback
- `permission_handler` required for microphone on Android/iOS; Linux/Windows may vary

### 1.7 File & UI Utilities

| Library | Purpose | Notes |
|---------|---------|-------|
| **file_picker** ^8.x | File selection | Cross-platform, supports Termius export files |
| **path_provider** ^2.x | App data directories | DB file path, cache dir |
| **path** ^1.x | Path manipulation | Cross-platform path joins |
| **uuid** ^4.x | UUID generation | For all entity IDs |
| **intl** ^0.19.x | Date/number formatting | Message timestamps, durations |

## 2. TTS/STT Architecture Decision

### Recommended: Server-First TTS, Hybrid STT

```
STT Path (Speech → Text):
  Mobile (Android/iOS):
    1. speech_to_text (local, fast for wake/commands)
    2. Audio stream + device transcript → Gormes (server STT for accuracy)

  Desktop (Linux/Windows):
    1. Vosk (offline fallback for wake words)
    2. Audio stream + device transcript → Gormes (server STT primary)

TTS Path (Text → Speech):
  All platforms:
    1. Server-generated audio (Gormes TTS provider: ElevenLabs, OpenAI TTS, etc.)
    2. Streamed via voice.audio frames to just_audio StreamAudioSource
    3. Local flutter_tts only for short confirmations ("Agent switched", "Connected")
```

### Why Not Local TTS on Linux

- `flutter_tts` Linux support is inconsistent
- No reliable cross-platform TTS library with quality comparable to cloud TTS
- Gormes already configures TTS providers — reuse that infrastructure
- Server TTS enables per-agent voice profiles, which is a core feature

### Why Hybrid STT

- Mobile platforms have good local STT — use it for wake word + short commands
- Desktop platforms lack reliable local STT — Vosk is the only offline option
- Server STT (via Gormes) provides the best accuracy and language coverage
- The PRD requires local command detection (wake word, agent switching)

## 3. SSH Architecture Decision

### Recommended: dartssh2 with Custom Protocol Over stdio

- `dartssh2` is the only actively maintained pure-Dart SSH library
- No native dependencies → consistent behavior across all platforms
- `client.execute('gormes navivox serve --stdio')` gives stdin/stdout streams
- Custom binary protocol framed over these streams (not raw newline JSON)
- Terminal widget uses `client.shell(pty: SSHPtyConfig(...))` with xterm.dart

### Protocol Framing

The protocol needs binary framing because audio can't be newline-delimited JSON:

```
[4 bytes: magic "NVOX"] [4 bytes: version] [4 bytes: header length] [N bytes: JSON header] [M bytes: binary payload]
```

All integer prelude fields use network byte order. The JSON header must include
`payload_length`; the decoder reads exactly that many payload bytes after the
header. This is simpler than WebSocket-style framing and avoids JSON overhead
for audio passthrough.

## 4. Config UI Architecture

### Schema-Driven Forms

Gormes exposes `config.schema` which returns the structure:
- Sections with fields
- Field types: string, integer, boolean, enum, object, array, secret
- Validation rules: required, min, max, enum values
- Secret fields are rendered as status indicators + set/rotate/delete buttons

### Staged Apply Flow

```
Edit (local) → Diff (server validates) → Confirm → Apply (server writes) → Reload
```

The local app never edits `config.toml` directly. All changes go through the protocol.

## 5. Known Library Limitations & Workarounds

| Limitation | Impact | Workaround |
|------------|--------|------------|
| `speech_to_text` no Linux STT | Wake word detection fails on Linux desktop | Vosk as offline fallback; accept no local STT on Linux |
| `flutter_secure_storage` Linux needs secret service | Key storage fails on minimal Linux installs | Clear setup error message; package dependency docs |
| `record` Linux needs system packages | Mic capture fails without parecord/ffmpeg | Install instructions in setup wizard; check at runtime |
| `just_audio` StreamAudioSource latency | TTS may have initial buffering delay | Tune `AudioLoadConfiguration`; pre-buffer small chunks |
| No Flutter biometric on Linux | Secret editor can't use biometric gate | App-level PIN fallback |
| `xterm.dart` 91 open issues | Terminal may have edge cases | Test core flows; accept minor rendering issues for V1 |
| `dartssh2` key format coverage | Some key formats may not parse | Document supported formats; test against Termius exports |

## 6. Package Version Targets (for pubspec.yaml)

```yaml
dependencies:
  flutter:
    sdk: flutter
  dartssh2: ^2.17.1
  xterm: ^4.0.1
  flutter_secure_storage: ^10.0.0
  local_auth: ^3.0.0
  file_picker: ^8.0.0
  record: ^6.1.2
  speech_to_text: ^6.6.0
  just_audio: ^0.9.40
  flutter_tts: ^4.0.0          # optional, command confirmations
  riverpod: ^2.5.0
  flutter_riverpod: ^2.5.0
  riverpod_annotation: ^2.3.0
  go_router: ^14.0.0
  drift: ^2.20.0
  sqlite3_flutter_libs: ^0.5.0
  drift_flutter: ^0.1.0
  cryptography: ^2.7.0
  ssh_key: ^0.8.0
  permission_handler: ^11.3.0
  freezed_annotation: ^2.4.0
  json_annotation: ^4.9.0
  path_provider: ^2.1.0
  path: ^1.9.0
  uuid: ^4.4.0
  intl: ^0.19.0

dev_dependencies:
  flutter_test:
    sdk: flutter
  build_runner: ^2.4.0
  freezed: ^2.5.0
  json_serializable: ^6.8.0
  drift_dev: ^2.20.0
  riverpod_generator: ^2.4.0
  mockito: ^5.4.0
  mocktail: ^1.0.0
  flutter_lints: ^4.0.0
```

## 7. CI/CD Setup Tips

```yaml
# .github/workflows/flutter.yml
- run: flutter pub get
- run: dart run build_runner build --delete-conflicting-outputs
- run: flutter analyze
- run: flutter test
- run: flutter test integration_test
```

Key CI considerations:
- `build_runner` must run before `flutter analyze` (generated files)
- Generated files committed to repo (avoids build_runner in CI for most changes)
- Linux CI needs: `sudo apt install -y pulseaudio-utils ffmpeg libsecret-1-dev libjsoncpp-dev`
- Integration tests need a fake Navivox server (can be a simple Dart stdio echo server)

## 8. Performance Considerations

- **SSH connection pool**: One SSH session per server. Don't open multiple.
- **Protocol framing**: Binary frames minimize overhead vs JSON for audio passthrough.
- **Chat list**: Use Drift's lazy loading + `LIMIT` queries for large message histories.
- **Voice streaming**: Use `record` stream mode (not file mode) for real-time audio.
- **Audio playback**: `StreamAudioSource` with small chunk sizes for low-latency TTS.
- **Config schema**: Cache config schema in Drift; refresh on connect and on config change.
- **Terminal**: xterm.dart renders at 60fps natively. Don't block UI thread with SSH I/O.
