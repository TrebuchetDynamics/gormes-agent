# Navivox Architecture

Status: planning draft
Source: derived from navibox-prd.md

## 1. High-Level Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                     Flutter App (Client)                      │
│  ┌─────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │   UI    │ │  Riverpod │ │  Drift   │ │ fl_secure_store  │ │
│  │ Widgets │ │  State    │ │  SQLite  │ │   (keys/auth)    │ │
│  └────┬────┘ └────┬─────┘ └────┬─────┘ └────────┬─────────┘ │
│       │            │            │                 │           │
│  ┌────┴────────────┴────────────┴─────────────────┴─────────┐ │
│  │                    Service Layer                          │ │
│  │  SSHSession  NavivoxChannel  VoiceEngine  ConfigAdmin     │ │
│  │  KeyManager  ServerRepo     AgentRepo    TermiusImport    │ │
│  └────────────────────────┬─────────────────────────────────┘ │
│                           │ SSH (dartssh2)                    │
└───────────────────────────┼──────────────────────────────────┘
                            │
                    ┌───────┴────────┐
                    │  Remote Server  │
                    │  Gormes Agent   │
                    │  navivox serve  │
                    │  --stdio        │
                    └────────────────┘
```

## 2. Component Tree

### 2.1 Package Structure

```
lib/
├── main.dart                          # App entry, ProviderScope, GoRouter
├── app.dart                           # MaterialApp.router wrapper
├── router/
│   ├── app_router.dart                # GoRouter config, redirects, shell routes
│   └── routes.dart                    # Route path constants and enums
├── core/
│   ├── ssh/
│   │   ├── ssh_session.dart           # SSHClient wrapper, connect/reconnect
│   │   ├── key_manager.dart           # Key gen/import/decrypt/storage
│   │   └── host_key_pinner.dart       # Host fingerprint verification
│   ├── protocol/
│   │   ├── frame.dart                 # Navivox protocol framing
│   │   ├── codec.dart                 # Encode/decode frames
│   │   └── events.dart                # All protocol event types (freezed)
│   ├── channel/
│   │   ├── navivox_channel.dart       # Channel over SSH stdio
│   │   └── channel_state.dart         # Connection state machine
│   ├── crypto/
│   │   └── ed25519_keys.dart          # Key generation helpers
│   └── platform/
│       └── secure_storage.dart        # Secure storage abstraction
├── data/
│   ├── database/
│   │   ├── app_database.dart          # Drift database definition
│   │   ├── tables/
│   │   │   ├── servers_table.dart
│   │   │   ├── identities_table.dart
│   │   │   ├── host_keys_table.dart
│   │   │   ├── agents_table.dart
│   │   │   ├── messages_table.dart
│   │   │   ├── tool_calls_table.dart
│   │   │   └── settings_table.dart
│   │   └── daos/
│   │       ├── server_dao.dart
│   │       ├── identity_dao.dart
│   │       ├── agent_dao.dart
│   │       └── chat_dao.dart
│   ├── repositories/
│   │   ├── server_repository.dart
│   │   ├── identity_repository.dart
│   │   ├── agent_repository.dart
│   │   ├── chat_repository.dart
│   │   └── config_repository.dart
│   └── imports/
│       ├── termius_importer.dart      # Termius JSON/export parser
│       └── termius_mapper.dart        # Map Termius fields to Navivox models
├── features/
│   ├── servers/
│   │   ├── providers/
│   │   │   └── servers_provider.dart
│   │   ├── screens/
│   │   │   ├── servers_screen.dart
│   │   │   └── server_detail_screen.dart
│   │   └── widgets/
│   │       ├── server_card.dart
│   │       └── server_form.dart
│   ├── keys/
│   │   ├── providers/
│   │   │   └── keys_provider.dart
│   │   ├── screens/
│   │   │   ├── keys_screen.dart
│   │   │   └── key_import_screen.dart
│   │   └── widgets/
│   │       ├── key_card.dart
│   │       └── key_generation_dialog.dart
│   ├── chat/
│   │   ├── providers/
│   │   │   ├── chat_provider.dart
│   │   │   ├── channel_provider.dart
│   │   │   └── message_stream_provider.dart
│   │   ├── screens/
│   │   │   └── chat_screen.dart
│   │   └── widgets/
│   │       ├── message_bubble.dart
│   │       ├── message_composer.dart
│   │       ├── tool_call_card.dart
│   │       ├── agent_switcher.dart
│   │       ├── voice_control_bar.dart
│   │       └── typing_indicator.dart
│   ├── voice/
│   │   ├── providers/
│   │   │   ├── voice_provider.dart
│   │   │   └── stt_provider.dart
│   │   ├── services/
│   │   │   ├── mic_capture_service.dart
│   │   │   ├── local_stt_service.dart
│   │   │   └── wake_word_detector.dart
│   │   └── widgets/
│   │       ├── voice_record_button.dart
│   │       ├── voice_waveform.dart
│   │       └── language_selector.dart
│   ├── agents/
│   │   ├── providers/
│   │   │   └── agents_provider.dart
│   │   ├── screens/
│   │   │   ├── agents_screen.dart
│   │   │   └── agent_editor_screen.dart
│   │   └── widgets/
│   │       ├── agent_card.dart
│   │       └── workspace_validator.dart
│   ├── config/
│   │   ├── providers/
│   │   │   ├── config_schema_provider.dart
│   │   │   ├── config_values_provider.dart
│   │   │   └── config_diff_provider.dart
│   │   ├── screens/
│   │   │   ├── config_overview_screen.dart
│   │   │   ├── config_section_screen.dart
│   │   │   └── secret_editor_screen.dart
│   │   └── widgets/
│   │       ├── config_form_field.dart
│   │       ├── secret_status_indicator.dart
│   │       ├── config_diff_viewer.dart
│   │       └── apply_confirm_sheet.dart
│   ├── terminal/
│   │   ├── providers/
│   │   │   └── terminal_provider.dart
│   │   └── screens/
│   │       └── terminal_screen.dart
│   └── settings/
│       ├── providers/
│       │   └── settings_provider.dart
│       └── screens/
│           └── settings_screen.dart
└── shared/
    ├── theme/
    │   ├── app_theme.dart
    │   └── colors.dart
    ├── widgets/
    │   ├── app_scaffold.dart
    │   ├── connection_status_bar.dart
    │   └── error_recovery_sheet.dart
    └── utils/
        ├── log_redactor.dart           # Redact secrets from logs
        └── platform_check.dart
```

### 2.2 Top-Level Widget Tree

```
MaterialApp.router
├── GoRouter
│   ├── ShellRoute (BottomNavBar)
│   │   ├── ChatsScreen        # /chats
│   │   ├── ServersScreen      # /servers
│   │   ├── AgentsScreen       # /agents
│   │   ├── ConfigScreen       # /config
│   │   ├── KeysScreen         # /keys
│   │   ├── TerminalScreen     # /terminal
│   │   └── SettingsScreen     # /settings
│   ├── ChatScreen             # /chats/:serverId/:threadId
│   ├── ServerDetailScreen     # /servers/:id
│   ├── AgentEditorScreen      # /agents/:id/edit
│   ├── KeyImportScreen        # /keys/import
│   ├── ConfigSectionScreen    # /config/:section
│   ├── SecretEditorScreen     # /config/secrets/:key
│   └── FirstRunWizard         # /setup
└── FirstRunWizard (shown when no servers configured)
    ├── Step 1: TermiusImport / ManualServer
    ├── Step 2: KeyImportOrGenerate
    ├── Step 3: HostVerification
    ├── Step 4: GormesProbe
    ├── Step 5: DevicePairing
    └── Step 6: AgentSelect/Create
```

## 3. State Management Architecture (Riverpod)

### 3.1 Provider Layers

```
┌─────────────────────────────────────────┐
│             UI Layer (Widgets)           │
│  ConsumerWidget / ConsumerStatefulWidget │
└──────────────────┬──────────────────────┘
                   │ watch/read
┌──────────────────┴──────────────────────┐
│          Presentation Providers          │
│  chatStateProvider, configFormProvider   │
│  voiceStateProvider, navigationProvider  │
└──────────────────┬──────────────────────┘
                   │ watch/read
┌──────────────────┴──────────────────────┐
│           Service Providers              │
│  sshSessionProvider(serverId)            │
│  navivoxChannelProvider(serverId)        │
│  voiceEngineProvider                     │
│  configAdminProvider(serverId)           │
└──────────────────┬──────────────────────┘
                   │ watch/read
┌──────────────────┴──────────────────────┐
│         Repository Providers             │
│  serverRepoProvider, agentRepoProvider   │
│  chatRepoProvider, configRepoProvider    │
│  identityRepoProvider                    │
└──────────────────┬──────────────────────┘
                   │ watch/read
┌──────────────────┴──────────────────────┐
│          Database Provider               │
│  appDatabaseProvider (Drift singleton)   │
└─────────────────────────────────────────┘
```

### 3.2 Key Providers

```dart
// Core infrastructure
final appDatabaseProvider = Provider<AppDatabase>((ref) => AppDatabase());

// Server connection (scoped per server)
final sshSessionProvider = AsyncNotifierProvider.family<SSHSessionNotifier, SSHSession?, String>(
  SSHSessionNotifier.new,
);

// Navivox channel (scoped per server)
final navivoxChannelProvider = AsyncNotifierProvider.family<NavivoxChannelNotifier, NavivoxChannel?, String>(
  NavivoxChannelNotifier.new,
);

// Active chat messages (streamed from channel)
final chatMessagesProvider = StreamProvider.family<List<ChatMessage>, String>(
  (ref, threadId) => ref.watch(navivoxChannelProvider(threadId)).requireValue!.messages,
);

// Currently selected agent
final selectedAgentProvider = StateProvider.family<String?, String>(
  (ref, serverId) => ref.watch(agentRepoProvider).getDefaultAgentId(serverId),
);

// Config schema (cached from server, refreshed on connect)
final configSchemaProvider = AsyncNotifierProvider.family<ConfigSchemaNotifier, ConfigSchema, String>(
  ConfigSchemaNotifier.new,
);

// Voice state
final voiceStateProvider = StateNotifierProvider<VoiceStateNotifier, VoiceState>(
  VoiceStateNotifier.new,
);

// Auth guard state
final pairingRoleProvider = StateProvider.family<PairingRole?, String>(
  (ref, serverId) => PairingRole.operator, // default until paired
);
```

### 3.3 Cross-Cutting State

- **ConnectionState**: Tracks SSH connection lifecycle per server (disconnected → connecting → connected → paired → ready). Drives UI state across all screens for that server.
- **AuthGate**: `pairingRoleProvider` gates config/agent mutation. Used by route redirect and UI visibility.
- **LocalCache**: Drift database holds local copies. `StreamProvider`s from Drift `watch()` queries auto-update UI.

## 4. Data Flow

### 4.1 Chat Message Flow

```
User types message
      │
      ▼
MessageComposer → chatProvider.sendMessage(text)
      │
      ▼
NavivoxChannelProvider.send(ChatSubmitEvent)
      │
      ▼
Protocol Codec encodes frame
      │
      ▼
SSHSession writes to stdin (dartssh2 shell session)
      │
      ▼
─────── SSH tunnel ───────
      │
      ▼
Gormes navivox serve --stdio receives frame
      │
      ▼
Gateway → Agent processing → Tool calls
      │
      ▼
Events stream back: chat.update, tool.call.started, chat.final
      │
      ▼
SSHSession stdout → Protocol Codec decodes frames
      │
      ▼
NavivoxChannelProvider emits typed events to stream
      │
      ▼
chatMessagesProvider updates (StreamProvider)
      │
      ▼
ChatScreen rebuilds with new messages and tool cards
```

### 4.2 Config Mutation Flow

```
User edits config field in ConfigSectionScreen
      │
      ▼
ConfigFormProvider accumulates changes locally
      │
      ▼
User taps "Review Changes"
      │
      ▼
ConfigDiffProvider requests config.diff from server
      │
      ▼
Server returns diff with validation errors/warnings
      │
      ▼
ConfigDiffViewer shows before/after, errors, sensitivity
      │
      ▼
User confirms → ApplyConfirmSheet (if sensitive)
      │
      ▼
ConfigAdminProvider sends config.apply with diff
      │
      ▼
Server validates, writes atomically, reloads
      │
      ▼
Server returns: applied | pending_restart | rolled_back
      │
      ▼
ConfigSchemaProvider refreshes (new schema state)
```

### 4.3 Voice Flow

```
User presses mic button or says wake word "NAVI"
      │
      ▼
VoiceStateNotifier → listening state
      │
      ▼
MicCaptureService starts record stream (PCM 16-bit, 16kHz mono)
      │
      ├── Local STT (speech_to_text) runs for wake/command detection
      │   └── If "NAVI switch agent mineru" → agent switch locally
      │
      └── Audio chunks streamed to NavivoxChannelProvider
            │
            ▼
          voice.submit frame with audio + device transcript
            │
            ▼
          Server processes: STT → Agent → TTS
            │
            ▼
          voice.audio frames streamed back (binary chunks)
            │
            ▼
          AudioPlayer (just_audio) plays back TTS audio
            │
            ▼
          chat.update frames with final text transcripts
```

### 4.4 Termius Import Flow

```
User selects Termius export file (file_picker)
      │
      ▼
TermiusImporter parses JSON/CSV format
      │
      ▼
Mapper extracts: hosts, groups, identities, keys, fingerprints
      │
      ▼
Deduplication against existing server/identity records
      │
      ▼
User reviews import preview (diff screen)
      │
      ▼
User confirms → ServerRepo.batchInsert(servers)
                 IdentityRepo.importKeys(identities)
      │
      ▼
Host key pinning prompts for new hosts on first connect
```

## 5. SSH Session Lifecycle

```
createSession(serverId, keyIdentity)
      │
      ▼
SSHSocket.connect(host, port) via dartssh2
      │
      ▼
SSHClient(identities: [keyPair])
      │
      ▼
Host key verification
      ├── Known host → compare pinned fingerprint → match?
      │   ├── Match → proceed
      │   └── Mismatch → ERROR: Host key changed (block + user action)
      └── New host → show fingerprint → user pins → proceed
      │
      ▼
client.execute('which gormes') → Gormes probe
      ├── Found → client.execute('gormes navivox serve --stdio')
      └── Not found → mark as generic SSH (terminal-only)
      │
      ▼
NavivoxChannel: send hello frame → receive server.status
      │
      ▼
Device pairing (if first time) → role assignment
      │
      ▼
Ready state → chat, config, agents available
```

## 6. Protocol Model (Freezed)

### 6.1 NavivoxFrame

```dart
@freezed
class NavivoxFrame with _$NavivoxFrame {
  const factory NavivoxFrame({
    required int version,           // protocol version
    required String messageId,      // unique per frame
    String? correlationId,          // links request/response
    required DateTime timestamp,
    required NavivoxEventType type,
    Map<String, dynamic>? metadata, // JSON header
    @Default([]) List<int> payload, // optional binary (audio/files)
  }) = _NavivoxFrame;

  factory NavivoxFrame.fromJson(Map<String, dynamic> json) =>
      _$NavivoxFrameFromJson(json);
}
```

### 6.2 EventType (sealed union)

```dart
@freezed
sealed class NavivoxEvent with _$NavivoxEvent {
  // Handshake
  const factory NavivoxEvent.hello({required DeviceInfo device}) = HelloEvent;
  const factory NavivoxEvent.serverStatus({required ServerInfo info}) = ServerStatusEvent;

  // Chat
  const factory NavivoxEvent.chatSubmit({required String text, String? voiceTranscript}) = ChatSubmitEvent;
  const factory NavivoxEvent.chatMessage({required String messageId, required String text, required String role}) = ChatMessageEvent;
  const factory NavivoxEvent.chatUpdate({required String messageId, required String delta}) = ChatUpdateEvent;
  const factory NavivoxEvent.chatFinal({required String messageId, required String text}) = ChatFinalEvent;

  // Voice
  const factory NavivoxEvent.voiceSubmit({required List<int> audio, String? transcript, double? confidence}) = VoiceSubmitEvent;
  const factory NavivoxEvent.voiceTranscript({required String transcript, required bool isFinal}) = VoiceTranscriptEvent;
  const factory NavivoxEvent.voiceAudio({required List<int> audio, required String format}) = VoiceAudioEvent;

  // Agents
  const factory NavivoxEvent.agentList({required List<AgentInfo> agents}) = AgentListEvent;
  const factory NavivoxEvent.agentSelect({required String agentId}) = AgentSelectEvent;
  // ... more agent events

  // Tools
  const factory NavivoxEvent.toolCallStarted({required ToolCallInfo info}) = ToolCallStartedEvent;
  const factory NavivoxEvent.toolCallProgress({required String callId, required String summary}) = ToolCallProgressEvent;
  const factory NavivoxEvent.toolCallCompleted({required ToolCallResult result}) = ToolCallCompletedEvent;
  const factory NavivoxEvent.toolApprovalRequested({required String callId, required String toolName}) = ToolApprovalRequestedEvent;

  // Config
  const factory NavivoxEvent.configSchema({required ConfigSchema schema}) = ConfigSchemaEvent;
  const factory NavivoxEvent.configGet({required Map<String, dynamic> config}) = ConfigGetEvent;
  const factory NavivoxEvent.configDiff({required DiffResult diff}) = ConfigDiffEvent;
  const factory NavivoxEvent.configApplyResult({required ApplyResult result}) = ConfigApplyResultEvent;

  // Health
  const factory NavivoxEvent.ping() = PingEvent;
  const factory NavivoxEvent.pong() = PongEvent;
  const factory NavivoxEvent.error({required int code, required String message}) = ErrorEvent;
}
```

## 7. Cross-Platform Strategy

### 7.1 Platform-Specific Considerations

| Concern | Android | iOS | Linux | Windows |
|---------|---------|-----|-------|---------|
| Secure storage | flutter_secure_storage (KeyStore) | flutter_secure_storage (Keychain) | libsecret (gnome-keyring/kwallet) | flutter_secure_storage (DPAPI) |
| Biometric/PIN | local_auth (fingerprint/face/PIN) | local_auth (Face ID/Touch ID) | App-level PIN fallback | local_auth (Windows Hello) |
| Mic capture | record (AudioRecord) | record (AVFoundation) | record (parecord+pactl+ffmpeg) | record (MediaFoundation) |
| Local STT | speech_to_text (Google/on-device) | speech_to_text (SFSpeechRecognizer) | Vosk (offline) | speech_to_text (WinRT) |
| Audio playback | just_audio (MediaPlayer) | just_audio (AVPlayer) | just_audio (GStreamer) | just_audio (MediaFoundation) |
| File picker | file_picker (SAF) | file_picker (UIDocumentPicker) | file_picker (GTK) | file_picker (Win32) |
| SSH client | dartssh2 (pure Dart) | dartssh2 (pure Dart) | dartssh2 (pure Dart) | dartssh2 (pure Dart) |
| Terminal | xterm.dart | xterm.dart | xterm.dart | xterm.dart |

### 7.2 Linux Caveats

- `flutter_secure_storage` requires `libsecret-1-0`, `libjsoncpp1`, and a running secret service (gnome-keyring or kwalletmanager)
- `record` package requires `parecord`, `pactl`, and `ffmpeg` system packages
- Local STT on Linux uses Vosk (offline) - requires model download and is not as accurate as platform STT
- No biometric support on most Linux distros; app-level PIN is the fallback
- App must surface clear errors when dependencies are missing

## 8. Security Architecture

```
┌─────────────────────────────────────────────────┐
│              Security Boundaries                 │
│                                                  │
│  ┌──────────────┐    ┌──────────────────────┐   │
│  │ Secure Store │    │  Encrypted Blob      │   │
│  │ (Keychain/   │◄───│  (large keys, Linux  │   │
│  │  KeyStore/   │    │   when needed)       │   │
│  │  libsecret)  │    └──────────────────────┘   │
│  └──────┬───────┘                                │
│         │ private keys never leave secure store  │
│  ┌──────┴───────┐                                │
│  │  Key Manager │  loads into memory only for   │
│  │  (in-memory) │  active SSH session            │
│  └──────┬───────┘                                │
│         │                                        │
│  ┌──────┴───────┐    ┌──────────────────────┐   │
│  │ Auth Gate    │    │  Log Redactor         │   │
│  │ (biometric/  │    │  (strips secrets from │   │
│  │  PIN unlock) │    │   all log output)     │   │
│  └──────────────┘    └──────────────────────┘   │
│                                                  │
│  ┌──────────────────────────────────────────┐   │
│  │  Navivox Protocol Layer                   │   │
│  │  - Secret values never in protocol errors │   │
│  │  - config.secret.set sends value only     │   │
│  │  - config.get returns [REDACTED] only     │   │
│  │  - Tool outputs marked sensitive redacted │   │
│  └──────────────────────────────────────────┘   │
└─────────────────────────────────────────────────┘
```

### 8.1 Data Classification

| Data Class | Storage | Access Control | Logging |
|------------|---------|---------------|---------|
| SSH private keys | Secure storage / encrypted blob | Biometric/PIN gate | NEVER logged |
| SSH passphrases | Memory only during decrypt | User prompt per session | NEVER logged |
| Host key fingerprints | Drift SQLite | App-level | OK to log |
| Chat messages | Drift SQLite (cache) | App-level | Redact if marked private |
| Agent config | Drift SQLite (cache) | Per pairing role | OK |
| Server config (non-secret) | Drift SQLite (cache) | Per pairing role | OK |
| Secret values (tokens, keys) | Never stored locally | Biometric/PIN to edit | NEVER logged |
| Tool output (sensitive) | Memory only | Per pairing role | Redacted |

## 9. Desktop vs Mobile Layout Strategy

### 9.1 Mobile (< 600dp width)
- Bottom navigation bar for primary sections
- Full-screen sheets for detail/edit operations
- Single-column chat layout
- FAB for quick actions (new chat, voice)

### 9.2 Desktop/Tablet (>= 600dp width)
- Left rail/sidebar for navigation
- Master-detail split view
- Side panel for config editors
- Multi-column chat (thread list + chat area)
- Keyboard shortcuts for power users

### 9.3 Adaptive Widgets

```dart
// Use LayoutBuilder or MediaQuery to switch layouts
class AdaptiveScaffold extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final isDesktop = MediaQuery.of(context).size.width >= 600;
    return isDesktop ? DesktopLayout() : MobileLayout();
  }
}
```

## 10. Error Recovery Architecture

All error states map to recovery actions, not stack traces:

```
Error Type ──────────► Recovery Action
─────────────────────────────────────────
SSH key rejected    → Key selector sheet
Wrong passphrase    → Passphrase retry dialog
Host key changed    → Fingerprint review + re-trust
Secure store fail   → System deps checklist
Gormes not found    → Install guide / open terminal
Protocol mismatch   → Version info + update prompt
Config write fail   → Rollback offer + retry
Tool approval timeout → Retry / cancel options
Mic permission deny  → Settings link + text fallback
TTS/STT unavailable  → Text-only fallback
```

All error surfaces use a shared `ErrorRecoverySheet` widget that presents the error, a human-readable message, and a recovery action button.

## 11. Build and Code Generation

```
# Code generation pipeline
dart run build_runner build --delete-conflicting-outputs

# Generates:
# - *.freezed.dart  (immutable data classes, unions)
# - *.g.dart        (JSON serialization)
# - *.drift.dart    (database tables, DAOs)
# - *.g.dart        (Riverpod providers if using @riverpod)
```

`build.yaml` configuration:
```yaml
targets:
  $default:
    builders:
      freezed:
        options:
          runs_before:
            - json_serializable
      json_serializable:
        options:
          include_if_null: false
          explicit_to_json: true
      drift_dev:
        options:
          generate_connectivity: true
```
