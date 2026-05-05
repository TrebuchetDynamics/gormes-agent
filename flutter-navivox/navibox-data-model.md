# Navivox Data Model

Status: planning draft
Source: derived from navibox-prd.md

## 1. Database Schema (Drift/SQLite)

### 1.1 Tables

#### servers

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | TEXT | PRIMARY KEY | UUID |
| display_name | TEXT | NOT NULL | User-friendly name |
| hostname | TEXT | NOT NULL | |
| port | INTEGER | NOT NULL, DEFAULT 22 | |
| username | TEXT | NOT NULL | SSH username |
| identity_id | TEXT | REFERENCES identities(id) | Selected SSH key |
| pinned_host_key | TEXT | | SHA256 fingerprint |
| last_connected_at | INTEGER | | Unix timestamp |
| last_status | TEXT | | 'unknown', 'reachable', 'unreachable', 'gormes_ready' |
| gormes_version | TEXT | | From server.status |
| gormes_config_version | TEXT | | Config version hash |
| preferred_agent_id | TEXT | | Last used agent |
| terminal_profile | TEXT | | JSON blob for terminal settings |
| import_source | TEXT | | 'manual', 'termius', 'ssh_config' |
| import_metadata | TEXT | | JSON blob for import tracking |
| pairing_device_id | TEXT | | Current paired device ID |
| pairing_role | TEXT | | 'owner', 'admin', 'operator', 'viewer' |
| created_at | INTEGER | NOT NULL | Unix timestamp |
| updated_at | INTEGER | NOT NULL | Unix timestamp |

#### identities

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | TEXT | PRIMARY KEY | UUID |
| label | TEXT | NOT NULL | Display name |
| key_type | TEXT | NOT NULL | 'ed25519', 'rsa', 'ecdsa' |
| public_key | TEXT | NOT NULL | OpenSSH format public key |
| public_key_fingerprint | TEXT | NOT NULL | SHA256 fingerprint |
| private_key_secure_id | TEXT | NOT NULL | Secure storage key reference |
| is_encrypted | INTEGER | NOT NULL, DEFAULT 0 | Has passphrase? |
| key_size | INTEGER | | For RSA keys |
| comment | TEXT | | SSH key comment field |
| import_source | TEXT | | 'generated', 'termius', 'file_import' |
| created_at | INTEGER | NOT NULL | Unix timestamp |

#### host_known_keys

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | TEXT | PRIMARY KEY | UUID |
| server_id | TEXT | REFERENCES servers(id) | |
| hostname | TEXT | NOT NULL | |
| key_type | TEXT | NOT NULL | 'ssh-ed25519', 'ssh-rsa', etc. |
| fingerprint | TEXT | NOT NULL | SHA256 fingerprint |
| raw_key | TEXT | NOT NULL | Full known_hosts line |
| is_pinned | INTEGER | NOT NULL, DEFAULT 1 | User explicitly trusted? |
| pinned_at | INTEGER | | Unix timestamp |
| last_seen_at | INTEGER | | Unix timestamp |
| trust_status | TEXT | NOT NULL | 'trusted', 'changed', 'revoked' |

#### agents (local cache)

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | TEXT | PRIMARY KEY | Agent ID from Gormes |
| server_id | TEXT | NOT NULL | Owning server |
| display_name | TEXT | NOT NULL | |
| workspace_dir | TEXT | | Remote path |
| agent_dir | TEXT | | Remote path |
| is_default | INTEGER | DEFAULT 0 | |
| model_override | TEXT | | |
| skills | TEXT | | JSON array |
| tool_allow_list | TEXT | | JSON array |
| tool_deny_list | TEXT | | JSON array |
| voice_provider | TEXT | | TTS provider |
| voice_id | TEXT | | Voice model ID |
| voice_locale | TEXT | | e.g., 'en-US' |
| voice_speed | REAL | | 0.5 - 2.0 |
| language_policy | TEXT | | JSON blob |
| is_archived | INTEGER | DEFAULT 0 | |
| synced_at | INTEGER | | Unix timestamp |

#### messages (local cache)

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | TEXT | PRIMARY KEY | Message ID from server |
| server_id | TEXT | NOT NULL | |
| thread_id | TEXT | NOT NULL | Conversation/device thread ID |
| role | TEXT | NOT NULL | 'user', 'assistant', 'system' |
| content | TEXT | NOT NULL | Markdown text |
| content_type | TEXT | NOT NULL | 'text', 'voice', 'mixed' |
| turn_id | TEXT | | Groups messages in a turn |
| voice_transcript | TEXT | | Device or server transcript |
| voice_confidence | REAL | | 0.0 - 1.0 |
| has_audio | INTEGER | DEFAULT 0 | Has voice recording? |
| audio_duration_ms | INTEGER | | |
| is_final | INTEGER | DEFAULT 0 | Stream finished? |
| is_deleted | INTEGER | DEFAULT 0 | |
| agent_id | TEXT | | Which agent responded |
| created_at | INTEGER | NOT NULL | Server timestamp |

#### tool_calls (local cache)

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | TEXT | PRIMARY KEY | Tool call ID |
| message_id | TEXT | REFERENCES messages(id) | Parent message |
| server_id | TEXT | NOT NULL | |
| turn_id | TEXT | | Parent turn |
| tool_name | TEXT | NOT NULL | |
| display_name | TEXT | | Human-readable name |
| icon_hint | TEXT | | Emoji or icon hint |
| preview | TEXT | | Short summary |
| status | TEXT | NOT NULL | 'started','progress','completed','failed','cancelled','blocked' |
| risk_level | TEXT | | 'low', 'medium', 'high' |
| is_mutating | INTEGER | DEFAULT 0 | |
| requires_approval | INTEGER | DEFAULT 0 | |
| approval_status | TEXT | | 'pending', 'approved', 'denied' |
| result_summary | TEXT | | |
| error_message | TEXT | | Redacted error |
| artifact_refs | TEXT | | JSON array of refs |
| agent_id | TEXT | | |
| workspace | TEXT | | |
| started_at | INTEGER | | Server timestamp |
| completed_at | INTEGER | | Server timestamp |
| elapsed_ms | INTEGER | | Duration |

#### config_cache

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | INTEGER | PRIMARY KEY AUTOINCREMENT | |
| server_id | TEXT | NOT NULL | |
| section | TEXT | NOT NULL | e.g., 'hermes', 'telegram' |
| key_path | TEXT | NOT NULL | e.g., 'hermes.model' |
| value | TEXT | | Always string/JSON |
| is_secret | INTEGER | NOT NULL, DEFAULT 0 | |
| schema_type | TEXT | | 'string', 'int', 'bool', 'enum', 'object' |
| schema_enum_values | TEXT | | JSON array |
| schema_default | TEXT | | |
| is_dirty | INTEGER | DEFAULT 0 | Pending local change? |
| synced_at | INTEGER | | Unix timestamp |

#### settings

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| key | TEXT | PRIMARY KEY | |
| value | TEXT | NOT NULL | |
| updated_at | INTEGER | NOT NULL | |

### 1.2 Relationships

```
servers 1───* identities (via identity_id FK)
servers 1───* host_known_keys (via server_id FK)
servers 1───* agents (via server_id FK)
servers 1───* messages (via server_id FK)
messages 1───* tool_calls (via message_id FK)
servers 1───* config_cache (via server_id FK)
```

## 2. Domain Models (Freezed)

### 2.1 Core Models

```dart
@freezed
class Server with _$Server {
  const factory Server({
    required String id,
    required String displayName,
    required String hostname,
    @Default(22) int port,
    required String username,
    String? identityId,
    String? pinnedHostKey,
    DateTime? lastConnectedAt,
    @Default(ServerStatus.unknown) ServerStatus lastStatus,
    String? gormesVersion,
    String? gormesConfigVersion,
    String? preferredAgentId,
    TerminalProfile? terminalProfile,
    @Default(ImportSource.manual) ImportSource importSource,
    Map<String, dynamic>? importMetadata,
    String? pairingDeviceId,
    PairingRole? pairingRole,
    required DateTime createdAt,
    required DateTime updatedAt,
  }) = _Server;

  factory Server.fromJson(Map<String, dynamic> json) => _$ServerFromJson(json);
}

enum ServerStatus { unknown, reachable, unreachable, gormesReady }
enum ImportSource { manual, termius, sshConfig }
enum PairingRole { owner, admin, operator, viewer }

@freezed
class Identity with _$Identity {
  const factory Identity({
    required String id,
    required String label,
    required KeyType keyType,
    required String publicKey,
    required String publicKeyFingerprint,
    required String privateKeySecureId,
    @Default(false) bool isEncrypted,
    int? keySize,
    String? comment,
    ImportSource? importSource,
    required DateTime createdAt,
  }) = _Identity;

  factory Identity.fromJson(Map<String, dynamic> json) => _$IdentityFromJson(json);
}

enum KeyType { ed25519, rsa, ecdsa }

@freezed
class HostKnownKey with _$HostKnownKey {
  const factory HostKnownKey({
    required String id,
    required String serverId,
    required String hostname,
    required String keyType,
    required String fingerprint,
    required String rawKey,
    @Default(true) bool isPinned,
    DateTime? pinnedAt,
    DateTime? lastSeenAt,
    @Default(TrustStatus.trusted) TrustStatus trustStatus,
  }) = _HostKnownKey;

  factory HostKnownKey.fromJson(Map<String, dynamic> json) => _$HostKnownKeyFromJson(json);
}

enum TrustStatus { trusted, changed, revoked }

@freezed
class Agent with _$Agent {
  const factory Agent({
    required String id,
    required String serverId,
    required String displayName,
    String? workspaceDir,
    String? agentDir,
    @Default(false) bool isDefault,
    String? modelOverride,
    @Default([]) List<String> skills,
    @Default([]) List<String> toolAllowList,
    @Default([]) List<String> toolDenyList,
    String? voiceProvider,
    String? voiceId,
    String? voiceLocale,
    @Default(1.0) double voiceSpeed,
    LanguagePolicy? languagePolicy,
    @Default(false) bool isArchived,
    DateTime? syncedAt,
  }) = _Agent;

  factory Agent.fromJson(Map<String, dynamic> json) => _$AgentFromJson(json);
}
```

### 2.2 Chat Models

```dart
@freezed
class ChatMessage with _$ChatMessage {
  const factory ChatMessage({
    required String id,
    required String serverId,
    required String threadId,
    required MessageRole role,
    required String content,
    @Default(ContentType.text) ContentType contentType,
    String? turnId,
    String? voiceTranscript,
    double? voiceConfidence,
    @Default(false) bool hasAudio,
    int? audioDurationMs,
    @Default(false) bool isFinal,
    @Default(false) bool isDeleted,
    String? agentId,
    required DateTime createdAt,
  }) = _ChatMessage;

  factory ChatMessage.fromJson(Map<String, dynamic> json) => _$ChatMessageFromJson(json);
}

enum MessageRole { user, assistant, system }
enum ContentType { text, voice, mixed }

@freezed
class ToolCall with _$ToolCall {
  const factory ToolCall({
    required String id,
    required String messageId,
    required String serverId,
    String? turnId,
    required String toolName,
    String? displayName,
    String? iconHint,
    String? preview,
    required ToolCallStatus status,
    @Default(RiskLevel.low) RiskLevel riskLevel,
    @Default(false) bool isMutating,
    @Default(false) bool requiresApproval,
    ApprovalStatus? approvalStatus,
    String? resultSummary,
    String? errorMessage,
    @Default([]) List<String> artifactRefs,
    String? agentId,
    String? workspace,
    DateTime? startedAt,
    DateTime? completedAt,
    int? elapsedMs,
  }) = _ToolCall;

  factory ToolCall.fromJson(Map<String, dynamic> json) => _$ToolCallFromJson(json);
}

enum ToolCallStatus { started, progress, completed, failed, cancelled, blocked }
enum RiskLevel { low, medium, high }
enum ApprovalStatus { pending, approved, denied }
```

### 2.3 Config Models

```dart
@freezed
class ConfigSchema with _$ConfigSchema {
  const factory ConfigSchema({
    required String version,
    required List<ConfigSection> sections,
  }) = _ConfigSchema;

  factory ConfigSchema.fromJson(Map<String, dynamic> json) => _$ConfigSchemaFromJson(json);
}

@freezed
class ConfigSection with _$ConfigSection {
  const factory ConfigSection({
    required String name,
    required String displayName,
    required String description,
    required List<ConfigField> fields,
  }) = _ConfigSection;

  factory ConfigSection.fromJson(Map<String, dynamic> json) => _$ConfigSectionFromJson(json);
}

@freezed
class ConfigField with _$ConfigField {
  const factory ConfigField({
    required String keyPath,
    required String displayName,
    String? description,
    required ConfigFieldType type,
    @Default(false) bool isSecret,
    @Default(false) bool sensitive,
    bool? required,
    dynamic defaultValue,
    List<String>? enumValues,
    dynamic minValue,
    dynamic maxValue,
    String? placeholder,
    String? hint,
  }) = _ConfigField;

  factory ConfigField.fromJson(Map<String, dynamic> json) => _$ConfigFieldFromJson(json);
}

enum ConfigFieldType { string, integer, boolean, enum_choice, object, array, secret, host, port, filepath }

@freezed
class ConfigDiff with _$ConfigDiff {
  const factory ConfigDiff({
    required List<ConfigChange> changes,
    required List<String> warnings,
    required List<String> errors,
    required bool isValid,
  }) = _ConfigDiff;

  factory ConfigDiff.fromJson(Map<String, dynamic> json) => _$ConfigDiffFromJson(json);
}

@freezed
class ConfigChange with _$ConfigChange {
  const factory ConfigChange({
    required String keyPath,
    required String section,
    String? before,
    required String after,
    required bool isSensitive,
    @Default(false) bool isSecret,
  }) = _ConfigChange;

  factory ConfigChange.fromJson(Map<String, dynamic> json) => _$ConfigChangeFromJson(json);
}

@freezed
class ApplyResult with _$ApplyResult {
  const factory ApplyResult({
    required ApplyStatus status,
    String? message,
    List<String>? warnings,
    List<String>? errors,
    bool? requiresRestart,
  }) = _ApplyResult;

  factory ApplyResult.fromJson(Map<String, dynamic> json) => _$ApplyResultFromJson(json);
}

enum ApplyStatus { applied, pendingRestart, rolledBack, failed }
```

### 2.4 Voice Models

```dart
@freezed
class VoiceState with _$VoiceState {
  const factory VoiceState.idle() = VoiceIdle;
  const factory VoiceState.listening({
    @Default(false) bool wakeWordDetected,
    @Default(0.0) double audioLevel,
  }) = VoiceListening;
  const factory VoiceState.processing({
    String? partialTranscript,
  }) = VoiceProcessing;
  const factory VoiceState.speaking({
    required String agentId,
    required String voiceId,
  }) = VoiceSpeaking;
  const factory VoiceState.error({
    required String message,
  }) = VoiceError;
}

@freezed
class VoiceProfile with _$VoiceProfile {
  const factory VoiceProfile({
    required String provider,
    required String voiceId,
    @Default('en-US') String locale,
    @Default(1.0) double speed,
    double? pitch,
    String? style,
    String? fallbackVoiceId,
  }) = _VoiceProfile;

  factory VoiceProfile.fromJson(Map<String, dynamic> json) => _$VoiceProfileFromJson(json);
}

@freezed
class LanguagePolicy with _$LanguagePolicy {
  const factory LanguagePolicy({
    @Default('en') String defaultLanguage,
    @Default([]) List<String> allowedLanguages,
    @Default(false) bool autoDetect,
  }) = _LanguagePolicy;

  factory LanguagePolicy.fromJson(Map<String, dynamic> json) => _$LanguagePolicyFromJson(json);
}
```

## 3. Drift Database Definition

```dart
@DriftDatabase(
  tables: [Servers, Identities, HostKnownKeys, Agents, Messages, ToolCalls, ConfigCache, Settings],
)
class AppDatabase extends _$AppDatabase {
  AppDatabase() : super(_openConnection());

  @override
  int get schemaVersion => 1;

  static QueryExecutor _openConnection() {
    return driftDatabase(name: 'navivox_database');
  }

  // Server queries
  Future<List<ServerData>> getAllServers() => select(servers).get();
  Stream<List<ServerData>> watchServers() => select(servers).watch();
  Future<ServerData?> getServerById(String id) =>
      (select(servers)..where((s) => s.id.equals(id))).getSingleOrNull();

  // Identity queries
  Future<List<IdentityData>> getAllIdentities() => select(identities).get();
  Stream<List<IdentityData>> watchIdentities() => select(identities).watch();

  // Chat queries
  Stream<List<MessageData>> watchMessages(String serverId, String threadId) =>
      (select(messages)
        ..where((m) => m.serverId.equals(serverId) & m.threadId.equals(threadId))
        ..orderBy([(m) => OrderingTerm(expression: m.createdAt, mode: OrderingMode.asc)]))
      .watch();

  // Tool call queries
  Future<List<ToolCallData>> getToolCallsForMessage(String messageId) =>
      (select(toolCalls)..where((t) => t.messageId.equals(messageId))).get();

  // Config cache
  Future<void> cacheConfigSection(String serverId, String section, List<ConfigCacheCompanion> entries) {
    return transaction(() async {
      await (delete(configCache)..where((c) => c.serverId.equals(serverId) & c.section.equals(section))).go();
      await batch((b) {
        b.insertAll(configCache, entries);
      });
    });
  }
}
```

## 4. Termius Export Format

Termius exports hosts as JSON. Expected structure:

```json
{
  "hosts": [
    {
      "id": "uuid",
      "label": "My Server",
      "hostname": "example.com",
      "port": 22,
      "username": "root",
      "identity": "uuid",          // references identities
      "group": "Production",
      "tags": ["web", "api"],
      "known_host": "ssh-ed25519 AAAAC3NzaC1lZ...",
      "port_forwarding": [
        {
          "local_port": 8080,
          "remote_host": "localhost",
          "remote_port": 80
        }
      ]
    }
  ],
  "identities": [
    {
      "id": "uuid",
      "label": "My Key",
      "private_key": "-----BEGIN OPENSSH PRIVATE KEY-----\n...",
      "public_key": "ssh-ed25519 AAAAC3NzaC1lZ...",
      "passphrase": null
    }
  ],
  "groups": [
    {
      "id": "uuid",
      "name": "Production"
    }
  ]
}
```

### Termius Import Mapping

| Termius Field | Navivox Field | Notes |
|---------------|---------------|-------|
| hosts[].id | import_metadata.termius_id | For dedup |
| hosts[].label | servers.display_name | |
| hosts[].hostname | servers.hostname | |
| hosts[].port | servers.port | |
| hosts[].username | servers.username | |
| hosts[].identity | servers.identity_id | Look up after importing keys |
| hosts[].group | servers.import_metadata.group | |
| hosts[].tags | servers.import_metadata.tags | |
| hosts[].known_host | host_known_keys.raw_key | Parse and fingerprint |
| hosts[].port_forwarding | servers.import_metadata.port_forwarding | V1 stores, V2 renders |
| identities[].id | identities.import_source | |
| identities[].label | identities.label | |
| identities[].private_key | identities.private_key_secure_id | Store in secure storage |
| identities[].public_key | identities.public_key | |
| identities[].passphrase | identities.is_encrypted | Mark encrypted |

### Import Validation Rules

- Reject identity if `private_key` is null/empty and no `public_key` present (password-only identity)
- Warn if `private_key` is encrypted (will need passphrase on first use)
- Skip host if hostname is empty
- Deduplicate by hostname + port + username + key fingerprint
- If Termius ID matches existing import_metadata, update rather than duplicate
