# Navivox Testing Plan

Status: planning draft
Source: derived from navibox-prd.md §18

## 1. Testing Pyramid

```
           ┌──────────┐
           │   E2E    │  5-10% — Full flows with fake server
           │  Tests   │
          ┌┴──────────┴┐
          │ Integration │  15-20% — Widget + service integration
          │   Tests     │
         ┌┴─────────────┴┐
         │   Unit Tests   │  70-75% — Protocol, models, logic
         └────────────────┘
```

## 2. Unit Tests

### 2.1 Protocol Codecs

**File**: `test/core/protocol/codec_test.dart`

| Test | Description | Input | Expected |
|------|-------------|-------|----------|
| Encode/decode hello frame | Round-trip handshake | HelloEvent with device info | Identical decoded frame |
| Encode chat submit | User text message | ChatSubmitEvent(text: "hello") | Correct header + text payload |
| Encode voice submit | Audio + transcript | VoiceSubmitEvent with PCM bytes | Correct binary payload + metadata |
| Payload length mismatch | Header says 120 bytes, payload has 119 | Corrupt frame | Throws PayloadLengthMismatchError |
| Frame size limit exceeded | Reject oversized frames | 10MB payload | Throws FrameSizeExceededError |
| Version mismatch | Server sends newer prelude version | v3 frame, client v1 | Throws UnsupportedProtocolVersion before reading event body |
| Hello advertised versions | v1 hello payload includes supported_versions | JSON payload | server.status reports selected v1 |
| Corrupt magic bytes | Random data | [0x00, 0x00...] | Throws InvalidFrameError before JSON parse |
| Empty payload | Frame with no body | Header only, empty payload | Decodes with empty data list |
| Concurrent frames | Multiple frames in buffer | 3 frames back-to-back | Each decoded separately |
| UTF-8 text in header | Non-ASCII metadata | Japanese agent name in metadata | Correct round-trip |
| Binary payload integrity | Large binary blob | 1MB random bytes | Byte-for-byte match |

### 2.2 Termius Import

**File**: `test/data/imports/termius_importer_test.dart`

| Test | Description | Expected |
|------|-------------|----------|
| Parse valid Termius JSON | Full export: 3 hosts, 2 keys, 1 group | 3 servers, 2 identities, correct mappings |
| Reject password-only identity | Identity with no private_key | Rejected with clear reason |
| Skip empty hostname | Host with empty hostname field | Skipped, warning logged |
| Handle missing optional fields | Host without tags/group/known_host | Fills defaults, doesn't crash |
| Deduplicate by Termius ID | Import same file twice | No duplicates, updated timestamps |
| Parse encrypted key metadata | Identity with passphrase set | Marked isEncrypted=true |
| Parse port forwarding | Host with port_forwarding array | Stored in import_metadata |
| Handle malformed JSON | Random text, not JSON | Returns ImportError with message |
| Handle empty export | Valid JSON, no hosts | Returns empty result, no error |
| Map group folder hierarchy | Groups with parent references | Flat groups with parent reference metadata |
| Extract known_hosts fingerprint | known_host field present | SHA256 fingerprint computed and stored |

### 2.3 Key Management

**File**: `test/core/ssh/key_manager_test.dart`

| Test | Description | Expected |
|------|-------------|----------|
| Generate Ed25519 key pair | newKeyPair() | Valid key pair, 32-byte private |
| Verify generated key with ssh-keygen | Write to temp file, run ssh-keygen -y | Public key matches |
| Import OpenSSH Ed25519 private key | PEM format | Parsed by dartssh2 SSHKeyPair.fromPem |
| Import encrypted key with passphrase | PEM + passphrase | Decrypts successfully |
| Import encrypted key with wrong passphrase | PEM + wrong passphrase | Throws PassphraseError |
| Import RSA key | PEM RSA format | Parsed, key type detected as RSA |
| Import ECDSA key | PEM ECDSA format | Parsed, key type detected as ECDSA |
| Generate public key string | From key pair | OpenSSH format pubkey string |
| Compute fingerprint | From public key bytes | Correct SHA256:BASE64 format |
| Reject password-only identity | No key material | Rejected with message |

### 2.4 Host Key Verification

**File**: `test/core/ssh/host_key_pinner_test.dart`

| Test | Description | Expected |
|------|-------------|----------|
| First connection (unknown host) | No pinned key for hostname | Returns 'unknown' with fingerprint for pinning |
| Known host with matching key | Fingerprint matches pinned | Returns 'trusted' |
| Known host with changed key | Different fingerprint | Returns 'changed' with old/new fingerprints |
| Pinning a new host key | User accepts | Stores fingerprint, marks pinned |
| Re-trust after warning | User accepts change | Updates fingerprint, unpins old |
| Revoke host trust | User manually revokes | Marks trust_status 'revoked' |

### 2.5 Local Command Parsing

**File**: `test/features/voice/wake_word_detector_test.dart`

| Test | Description | Expected |
|------|-------------|----------|
| Detect "NAVI" wake word | Text: "NAVI what's the weather" | wakeWordDetected=true |
| Detect "NAVI switch agent X" | Text: "NAVI switch agent mineru" | command=switch_agent, target=mineru |
| Ignore without wake word | Text: "what's the weather" | wakeWordDetected=false |
| Case insensitive | Text: "navi SWITCH AGENT mineru" | Parses correctly |
| Noise between wake and command | Text: "NAVI um please switch agent mineru" | Extracts "switch agent mineru" |
| Multiple commands | Text: "NAVI switch agent mineru and deploy" | Takes first command |
| Non-command after wake | Text: "NAVI hello" | wakeWordDetected=true, no command |
| Configurable wake word | Change wake word to "ASSISTANT" | Detects new wake word |

### 2.6 Config Diff Logic

**File**: `test/features/config/config_diff_test.dart`

| Test | Description | Expected |
|------|-------------|----------|
| Simple field change | model: gpt-4 → gpt-4-turbo | One change, not sensitive |
| Secret field change | telegram.bot_token: set | Change marked isSecret=true |
| No changes | Empty diff | isValid=true, no changes |
| Invalid value | port: "notanumber" | isValid=false, error message |
| Multiple section changes | hermes + telegram changes | Changes grouped by section |
| Enum validation | voice provider: "invalid" | Error: not in enum values |

## 3. Widget Tests

### 3.1 Chat Screen

**File**: `test/features/chat/widgets/message_bubble_test.dart`

| Test | Description |
|------|-------------|
| Renders user message | Blue/right-aligned bubble |
| Renders assistant message | Gray/left-aligned bubble |
| Renders markdown content | Bold, code blocks, links |
| Renders streaming indicator | Pulsing cursor during chat.update |
| Renders final state | No cursor after chat.final |
| Renders voice message | Play button, waveform, duration |
| Renders deleted message | "This message was deleted" placeholder |
| Accessibility labels | Semantic labels on all interactive elements |

**File**: `test/features/chat/widgets/tool_call_card_test.dart`

| Test | Description |
|------|-------------|
| Renders tool name and status | Started/progress/completed badges |
| Renders elapsed time | Duration since started |
| Renders risk level badge | Color-coded: low/medium/high |
| Expand/collapse details | Tap to show/hide logs |
| Renders approval buttons | Approve/Deny when pending |
| Handles approval tap | Calls onApprove/onDeny callback |
| Renders error state | Red error with redacted message |
| Renders artifact links | List of artifact refs |
| Renders mutating indicator | Warning icon for mutating tools |
| Accessibility | Tool name, status, risk level announced |

### 3.2 Config Forms

**File**: `test/features/config/widgets/config_form_test.dart`

| Test | Description |
|------|-------------|
| Renders string field | Text input, label, placeholder |
| Renders integer field | Number input, min/max validation |
| Renders boolean field | Toggle switch |
| Renders enum field | Dropdown/segmented control |
| Renders secret field | Status indicator + "Set Token" button |
| Shows validation errors | Red border + error text for invalid input |
| Shows sensitive field warning | Orange border for sensitive fields |
| Disables fields for Viewer role | Read-only display |
| Renders object/array field | Expandable nested fields |

### 3.3 Secret Editor

**File**: `test/features/config/widgets/secret_editor_test.dart`

| Test | Description |
|------|-------------|
| Biometric gate on open | local_auth prompt before showing |
| Shows current status | "Set [REDACTED]" or "Not configured" |
| Set new secret | Text field + confirm button |
| Rotate secret | Shows old status + new value field |
| Delete secret | Confirmation dialog with warning |
| Shows provider info | "Source: SecretRef (1Password)" |
| Test connection button | Available for provider API keys |

### 3.4 Navigation

**File**: `test/router/app_router_test.dart`

| Test | Description |
|------|-------------|
| First run redirects to setup | No servers → /setup |
| Authenticated redirects to chats | Has servers → /chats |
| Viewer can't access config | Viewer role → redirect from /config |
| Admin can access config | Admin role → /config loads |
| Deep link chat | navivox://chat/server1/thread1 → chat screen |
| Unknown route | /foobar → 404 screen |
| Back navigation | Chat → pop → chats list |

## 4. Integration Tests

### 4.1 Navivox Channel

**File**: `test/core/channel/navivox_channel_test.dart`

Uses a fake Navivox server (Dart process) that echoes frames.

| Test | Description |
|------|-------------|
| Hello handshake | Send hello, receive server.status |
| Text chat round trip | chat.submit → chat.message + chat.final |
| Streaming response | chat.submit → multiple chat.update → chat.final |
| Voice submit | voice.submit with audio + transcript → voice.transcript + voice.audio |
| Error handling | Server sends error frame → channel emits error |
| Reconnect | Drop connection, reconnect, resume |
| Protocol version compatibility | v1 hello succeeds; unsupported prelude version fails before event handling |
| Ping/pong | ping → pong within timeout |
| Agent list | agent.list → parsed agent list |

### 4.2 SSH Session

**File**: `test/core/ssh/ssh_session_test.dart`

Requires a real SSH server (Docker container or test server).

| Test | Description |
|------|-------------|
| Connect with Ed25519 key | Generated key → successful auth |
| Connect with imported key | Termius-exported key → successful auth |
| Host key verification | First connect → fingerprint shown |
| Host key mismatch | Changed server key → blocked |
| Gormes probe | Execute 'which gormes' → detects/not detects |
| navivox serve start | Execute 'gormes navivox serve --stdio' |
| Encrypted key with passphrase | Prompt for passphrase, decrypt, connect |

### 4.3 Drift Database

**File**: `test/data/database/app_database_test.dart`

| Test | Description |
|------|-------------|
| Create and read server | Insert, query by ID |
| Watch servers stream | Insert → stream emits update |
| Cascade delete | Delete server → related messages, agents, keys deleted |
| Migration from v1 to v2 | Bump schemaVersion, verify migration |
| Concurrent reads/writes | Multiple simultaneous operations |
| Server deduplication | Insert duplicate hostname+port → constraint error or dedup |

## 5. End-to-End Tests

### 5.1 Critical Path Tests

| # | Test | Steps | Success Criteria |
|---|------|-------|-----------------|
| E1 | Fresh app to text chat | 1. Import Termius file<br>2. Select generated key<br>3. Connect + verify host<br>4. Pair device<br>5. Select agent<br>6. Send "hello" | "hello" response received, displayed in chat |
| E2 | Password SSH rejected | Try to add server with password auth | UI blocks it, clear reason shown |
| E3 | Voice capture + submit | 1. Open chat<br>2. Press mic<br>3. Speak "NAVI what's the weather"<br>4. Release mic | Audio captured, transcript sent, response received |
| E4 | "NAVI switch agent" | 1. Say "NAVI switch agent mineru"<br>2. Wait for switch | Agent switched, UI updates to show "mineru" |
| E5 | Config change provider/model | 1. Navigate to Config > Provider<br>2. Change model to "gpt-4-turbo"<br>3. Review diff<br>4. Apply | Config updated, no errors |
| E6 | Set Telegram bot token | 1. Config > Channels > Telegram<br>2. Tap "Set Token"<br>3. Enter token<br>4. Confirm | Token set, status shows "[REDACTED]", token not readable |
| E7 | Tool call UI | 1. Send message that triggers tool<br>2. Wait for tool_call_started<br>3. Observe progress<br>4. Approve if needed | Tool card shows all states, approval works |
| E8 | Host key change blocked | 1. Connect to known server<br>2. Server key changes (simulated)<br>3. Try to connect | Connection blocked, fingerprint diff shown |
| E9 | Secret never leaked | 1. Set various secrets<br>2. Check app logs<br>3. Check config output<br>4. Check error messages | No secret values visible in any output |

### 5.2 Platform Smoke Tests

| Platform | Tests |
|----------|-------|
| Android | E1, E3, E4, E5, E8 |
| iOS | E1, E3, E4, E5, E8 |
| Linux | E1, E2, E5, E6, E8 |
| Windows | E1, E2, E5, E8 |

### 5.3 Test Infrastructure

**Fake Navivox Server**: A simple Dart program that:
- Listens on stdio for framed protocol messages
- Returns hardcoded responses for known events
- Simulates streaming, errors, protocol mismatches
- Can be started as a subprocess from tests

```dart
// test/helpers/fake_navivox_server.dart
class FakeNavivoxServer {
  Future<void> start() async {
    // Spawn process that reads stdin for frames, writes responses to stdout
  }

  void respondTo(String eventType, dynamic response) { ... }
  void simulateDisconnect() { ... }
  void simulateError(int code, String message) { ... }
  Future<void> stop() async;
}
```

**Test SSH Server**: Docker container running sshd with known keys:
```dockerfile
FROM alpine:latest
RUN apk add openssh
RUN ssh-keygen -A
COPY test_keys/authorized_keys /root/.ssh/authorized_keys
EXPOSE 22
CMD ["/usr/sbin/sshd", "-D"]
```

## 6. Gormes Tests (Server Side)

These tests live in the Gormes repo, but the contract must be verified:

| Test | Description |
|------|-------------|
| `navivox serve --stdio` starts | Binary starts, outputs hello frame |
| Gateway inbound mapping | chat.submit maps to InboundEvent correctly |
| Selected agent routes | agent_id in event routes to correct agent |
| Typed tool events | Tool lifecycle emits proper typed events |
| Config schema complete | All supported sections in config.schema |
| Config atomic write | Invalid config doesn't partially apply |
| Secret write safe | config.secret.set writes, config.get redacts |
| Reload/rollback | Apply with restart → status pending_restart |
| Pairing role gates | Operator can't call config.set |

## 7. Test Commands

```bash
# Unit tests only
flutter test test/unit/

# Widget tests
flutter test test/widget/

# Integration tests
flutter test test/integration/

# All tests
flutter test

# With coverage
flutter test --coverage
genhtml coverage/lcov.info -o coverage/html

# Specific test file
flutter test test/core/protocol/codec_test.dart

# Integration tests with fake server
dart run test/helpers/fake_navivox_server.dart &
flutter test test/integration/
kill %1
```

## 8. CI Test Matrix

| Platform | Flutter Channel | Test Suite |
|----------|----------------|------------|
| ubuntu-latest | stable | unit + widget + integration |
| ubuntu-latest | beta | unit + widget |
| macos-latest | stable | unit + widget |
| windows-latest | stable | unit + widget |

## 9. Test Data

- `test/fixtures/termius_export_valid.json` — Valid Termius export with 3 hosts, 2 keys
- `test/fixtures/termius_export_password_only.json` — Export with password-only identity
- `test/fixtures/termius_export_minimal.json` — Minimal export (1 host, no key)
- `test/fixtures/ssh_keys/ed25519_test` — Test Ed25519 private key (for test SSH server)
- `test/fixtures/ssh_keys/ed25519_test.pub` — Corresponding public key
- `test/fixtures/ssh_keys/ed25519_encrypted` — Encrypted Ed25519 key (passphrase: "test")
- `test/fixtures/ssh_keys/rsa_test` — Test RSA private key
- `test/fixtures/config_schema_v1.json` — Sample config schema response
- `test/fixtures/chat_messages.json` — Sample chat message history
- `test/fixtures/tool_calls.json` — Sample tool call events
