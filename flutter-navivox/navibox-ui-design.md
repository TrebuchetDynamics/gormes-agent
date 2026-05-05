# Navivox UI Design Guide

Status: planning draft
Source: PRD requirements + open-source chat UI analysis + admin panel patterns

## 1. Design Principles

1. **Chat-first, not landing-page-first** — The first meaningful screen is always chat.
2. **Work-focused, not marketing** — Config and admin screens favor density over whitespace.
3. **Dense Telegram-style, not airy iMessage-style** — Information density matters for operator workflows.
4. **Tool calls are first-class UI** — Not hidden behind raw logs or JSON dumps.
5. **Secrets are invisible by design** — Write-only fields, status indicators, no read-back.
6. **Recovery over stack traces** — Every error state maps to a user action.
7. **Adaptive, not responsive** — Different layouts for mobile (bottom nav) vs desktop (left rail + split views).
8. **Dark mode by default** — Terminal operators prefer dark themes. Light mode available.

## 2. Screen Designs

### 2.1 Chat Screen (Primary)

```
┌──────────────────────────────────────────────┐
│ ☰ my-server                    mineru ▼  🟢  │
├──────────────────────────────────────────────┤
│                                              │
│  ┌── Today ──────────────────────────────┐   │
│  │                                        │   │
│  │  ┌──────────────────────────┐          │   │
│  │  │ Hello Gormes, can you    │     ✓✓   │   │
│  │  │ check the server status? │  12:30   │   │
│  │  └──────────────────────────┘          │   │
│  │                                        │   │
│  │  🤖 mineru                             │   │
│  │  ┌──────────────────────────────┐      │   │
│  │  │ Let me check that for you... │      │   │
│  │  └──────────────────────────────┘      │   │
│  │  12:30 PM                              │   │
│  │                                        │   │
│  │  ┌─── 🔧 execute_command ──────────┐   │   │
│  │  │ Running: uptime            🟢   │   │   │
│  │  │ Completed in 0.3s               │   │   │
│  │  │ Result: 14:30:42 up 5 days...   │   │   │
│  │  │ [Expand ▼]                      │   │   │
│  │  └─────────────────────────────────┘   │   │
│  │                                        │   │
│  │  🤖 mineru                             │   │
│  │  ┌──────────────────────────────┐      │   │
│  │  │ Server is up 5 days, load    │      │   │
│  │  │ average 0.15. All services   │      │   │
│  │  │ healthy ✅                    │      │   │
│  │  └──────────────────────────────┘      │   │
│  │  12:31 PM                              │   │
│  └────────────────────────────────────────┘   │
│                                              │
├──────────────────────────────────────────────┤
│ 📎  Type a message...                   🎤 ▶ │
└──────────────────────────────────────────────┘
```

**Key UI Elements:**
- **Hamburger menu** (mobile): Opens server/agent list drawer
- **Server name**: Tappable, opens server switcher dropdown
- **Agent pill**: Shows current agent name, tappable for quick switch
- **Green dot**: Connection status indicator
- **Date separator**: "Today", "Yesterday", date groups
- **User bubbles**: Right-aligned, colored, with delivery status (✓, ✓✓)
- **Agent bubbles**: Left-aligned, with agent name header
- **Markdown rendering**: Code blocks, bold, italic, links, lists
- **Tool call cards**: Expandable, status-badged, inline in message flow
- **Composer**: Paperclip, text field, mic/voice button, send button
- **Mic button**: Toggle voice mode / press-and-hold for recording

### 2.2 Chat Screen — Voice Active

```
┌──────────────────────────────────────────────┐
│ my-server                     mineru ▼  🟢   │
├──────────────────────────────────────────────┤
│                                              │
│  🎤 NAVI Listening...                 ⏹      │
│  ┌──────────────────────────────────────┐    │
│  │ "switch agent builder and run tests" │    │
│  └──────────────────────────────────────┘    │
│  ▓▓▓▓▓▓▓▓▓▓░░░░░░░░░░  -18dB    🟢 High   │
│                                              │
│  ┌── Command Detected ──────────────────┐    │
│  │ Switch to agent: builder             │    │
│  │ Run: tests                           │    │
│  │ [Execute]  [Cancel]                  │    │
│  └──────────────────────────────────────┘    │
│                                              │
│  [messages continue below...]                │
│                                              │
├──────────────────────────────────────────────┤
│  NAVI Command Mode  [⌨ Text]  [⏹ Stop]      │
└──────────────────────────────────────────────┘
```

### 2.3 Servers Screen

```
┌──────────────────────────────────────────────┐
│ Servers                              [+ Add] │
├──────────────────────────────────────────────┤
│                                              │
│ 🔍 Search servers...                         │
│                                              │
│ ┌─ Production ──────────────────────────┐    │
│ │                                        │    │
│ │ 🟢 gormes-prod                    ⚙️   │    │
│ │    192.168.1.100:22 • root            │    │
│ │    Gormes v1.2.3 • mineru active      │    │
│ │                                        │    │
│ │ 🟢 build-server                   ⚙️   │    │
│ │    10.0.0.50:2222 • builder           │    │
│ │    Gormes v1.2.1 • builder active     │    │
│ └────────────────────────────────────────┘    │
│                                              │
│ ┌─ Staging ─────────────────────────────┐    │
│ │                                        │    │
│ │ 🟡 staging-api                     ⚙️   │    │
│ │    staging.example.com:22 • deploy    │    │
│ │    Generic SSH • no Gormes detected   │    │
│ └────────────────────────────────────────┘    │
│                                              │
│ ┌─ Unreachable ─────────────────────────┐    │
│ │ 🔴 old-server      Last: 3 days ago    │    │
│ └────────────────────────────────────────┘    │
└──────────────────────────────────────────────┘
```

### 2.4 Server Detail Screen

```
┌──────────────────────────────────────────────┐
│ ← Server Detail                              │
├──────────────────────────────────────────────┤
│                                              │
│ gormes-prod                          🟢      │
│ ───────────────────────────────────────       │
│                                              │
│ 📋 Display Name    gormes-prod               │
│ 🌐 Hostname        192.168.1.100             │
│ 🔌 Port            22                        │
│ 👤 Username        root                      │
│ 🔑 SSH Key         ed25519-gormes-key        │
│ 🔒 Host Key        SHA256:AbCd... pinned     │
│ 🤖 Preferred Agent mineru                    │
│                                              │
│ ═══════════════════════════════════════       │
│ Gormes Status                                │
│                                              │
│ Version            v1.2.3                    │
│ Config Version     abc123def                 │
│ Active Channels    telegram, navivox         │
│ Paired Device      This device (Owner)       │
│ Last Connected     2 minutes ago             │
│                                              │
│ [Connect] [Open Terminal] [Edit] [Delete]    │
└──────────────────────────────────────────────┘
```

### 2.5 Keys Screen

```
┌──────────────────────────────────────────────┐
│ SSH Keys                          [+ Import] │
│                                    [+ Generate]│
├──────────────────────────────────────────────┤
│                                              │
│ ed25519-gormes-key                    🔑 ★    │
│ Type: Ed25519                                │
│ Fingerprint: SHA256:jK8mN...                 │
│ Source: Generated                            │
│ Servers: gormes-prod, build-server           │
│                                              │
│ ───────────────────────────────────────────── │
│                                              │
│ termius-imported-key                  🔑      │
│ Type: Ed25519 (encrypted)                    │
│ Fingerprint: SHA256:XyZ9p...                 │
│ Source: Termius Import                       │
│ Servers: staging-api                         │
│                                              │
│ ───────────────────────────────────────────── │
│                                              │
│ old-rsa-key                           🔑      │
│ Type: RSA 4096-bit                           │
│ Fingerprint: SHA256:qRsT3...                 │
│ Source: File Import                          │
│ Servers: (none)                              │
└──────────────────────────────────────────────┘
```

### 2.6 Agent Screen

```
┌──────────────────────────────────────────────┐
│ Agents                             [+ Create] │
├──────────────────────────────────────────────┤
│ my-server (gormes-prod)                      │
│                                              │
│ ● mineru (default)                   ⚙️      │
│   /home/xel/gormes-agent                     │
│   Model: gpt-4-turbo                         │
│   Voice: ElevenLabs • Adam • en-US           │
│   Tools: all allowed                         │
│                                              │
│ ○ builder                            ⚙️      │
│   /home/xel/projects/build                   │
│   Model: default                             │
│   Voice: OpenAI • Nova • en-US               │
│   Tools: shell, git, test                    │
│                                              │
│ build-server                                 │
│                                              │
│ ● ci-agent (default)                 ⚙️      │
│   /opt/ci/workspace                          │
│   Model: gpt-4-turbo                         │
│   Tools: shell, http                         │
└──────────────────────────────────────────────┘
```

### 2.7 Agent Editor Screen

```
┌──────────────────────────────────────────────┐
│ ← Edit Agent                                 │
├──────────────────────────────────────────────┤
│                                              │
│ 📝 Display Name                               │
│ ┌──────────────────────────────────────┐     │
│ │ mineru                               │     │
│ └──────────────────────────────────────┘     │
│                                              │
│ 📁 Workspace Directory                        │
│ ┌──────────────────────────────────────┐     │
│ │ /home/xel/gormes-agent               │     │
│ └──────────────────────────────────────┘     │
│ 🟢 Directory exists and is accessible         │
│                                              │
│ ⭐ Default Agent                    [Toggle]  │
│                                              │
│ 🤖 Model Override                            │
│ [Use server default ▼]                       │
│                                              │
│ 🛠️ Tools                                      │
│ Allow All                              [◉]   │
│ Allow List                             [○]   │
│ ┌──────────────────────────────────────┐     │
│ │ ☑ shell                              │     │
│ │ ☑ git                                │     │
│ │ ☑ file_read                          │     │
│ │ ☑ file_write                         │     │
│ │ ☐ browser                            │     │
│ │ ☐ http                               │     │
│ └──────────────────────────────────────┘     │
│                                              │
│ 🎤 Voice                                      │
│ Provider: [ElevenLabs ▼]                     │
│ Voice: [Adam ▼]                              │
│ Locale: [English (US) ▼]                     │
│ Speed: ●────────────○ 1.0x                   │
│                                              │
│ 🌐 Language Policy                            │
│ Default: [English ▼]                         │
│ Allowed: ☑ EN ☑ ES ☐ FR ☐ DE                 │
│ Auto-detect: [Toggle]                        │
│                                              │
│ [Cancel]                        [Save Agent] │
└──────────────────────────────────────────────┘
```

### 2.8 Config Overview Screen

```
┌──────────────────────────────────────────────┐
│ Config: gormes-prod                          │
├──────────────────────────────────────────────┤
│                                              │
│ Config Path: /home/xel/.gormes/config.toml   │
│ Env Path:    /home/xel/.gormes/.env          │
│ Version:     abc123def  |  Gormes v1.2.3     │
│ Reload Status: ✅ Ready                      │
│                                              │
│ ┌─ Provider & Models ───────────────────▶ │   │
│ │ OpenAI • gpt-4-turbo • api.openai.com    │   │
│ │ API Key: 🔴 [REDACTED]                   │   │
│ └──────────────────────────────────────────┘  │
│                                              │
│ ┌─ Channels ────────────────────────────▶ │   │
│ │ Telegram: 🟢 Active • 3 chats allowed     │   │
│ │ Discord: ⚫ Not configured                │   │
│ │ Slack: ⚫ Not configured                  │   │
│ │ Navivox: 🟢 Active (this connection)      │   │
│ └──────────────────────────────────────────┘  │
│                                              │
│ ┌─ Agents & Bindings ──────────────────▶ │   │
│ │ 2 agents • 0 bindings                    │   │
│ │ Default: mineru                           │   │
│ └──────────────────────────────────────────┘  │
│                                              │
│ ┌─ Tools & Display ────────────────────▶ │   │
│ │ Progress: fancy • Iterations: 20          │   │
│ └──────────────────────────────────────────┘  │
│                                              │
│ ┌─ Voice ─────────────────────────────▶ │   │
│ │ TTS: ElevenLabs • Wake: NAVI              │   │
│ └──────────────────────────────────────────┘  │
│                                              │
│ ┌─ Runtime ───────────────────────────▶ │   │
│ │ Terminal: bash • Input: 16KB max          │   │
│ └──────────────────────────────────────────┘  │
│                                              │
│ ┌─ Security ──────────────────────────▶ │   │
│ │ Blocklist: 0 sites • Host trust: pinned   │   │
│ └──────────────────────────────────────────┘  │
│                                              │
│ ┌─ Secrets ───────────────────────────▶ │   │
│ │ 3 configured • 0 missing                  │   │
│ └──────────────────────────────────────────┘  │
│                                              │
│ ┌─ Advanced ──────────────────────────▶ │   │
│ │ Cron, Skills, Delegation, Goncho          │   │
│ └──────────────────────────────────────────┘  │
│                                              │
│ [Redacted TOML View]  [Reload Config]        │
└──────────────────────────────────────────────┘
```

### 2.9 Config Section Screen (e.g., Provider & Models)

```
┌──────────────────────────────────────────────┐
│ ← Provider & Models                          │
├──────────────────────────────────────────────┤
│                                              │
│ 🔌 Provider                                  │
│ ┌──────────────────────────────────────┐     │
│ │ OpenAI                           [▼] │     │
│ └──────────────────────────────────────┘     │
│                                              │
│ 🤖 Default Model                             │
│ ┌──────────────────────────────────────┐     │
│ │ gpt-4-turbo                      [▼] │     │
│ └──────────────────────────────────────┘     │
│                                              │
│ 🌐 Endpoint                                  │
│ ┌──────────────────────────────────────┐     │
│ │ https://api.openai.com/v1            │     │
│ └──────────────────────────────────────┘     │
│                                              │
│ 🔑 API Key                                   │
│ Status: 🔴 Configured [REDACTED]              │
│ [Set New Key]  [Test Connection]  [Delete]    │
│                                              │
│ ───────────────────────────────────────────── │
│ Last applied: 2 hours ago                    │
│                                              │
│ [Review Changes]                    [Apply]  │
└──────────────────────────────────────────────┘
```

### 2.10 Terminal Screen

```
┌──────────────────────────────────────────────┐
│ Terminal: gormes-prod              [🗗] [✕]  │
├──────────────────────────────────────────────┤
│                                              │
│  Last login: Tue May  5 14:30:42 2026        │
│  root@gormes-prod:~# uptime                  │
│   14:30:42 up 5 days,  3:15,  0 users, ...   │
│  root@gormes-prod:~# █                       │
│                                              │
│                                              │
│                                              │
│                                              │
│                                              │
│                                              │
│ ═══════════════════════════════════════════   │
│ ⚠️ This is a direct SSH terminal.            │
│    Chat with agents in the Chats tab.         │
└──────────────────────────────────────────────┘
```

### 2.11 Settings Screen

```
┌──────────────────────────────────────────────┐
│ Settings                                     │
├──────────────────────────────────────────────┤
│                                              │
│ 🎨 Appearance                                │
│ Theme: [Dark ▼]                              │
│ Font Size: ●───────○ 14pt                    │
│ Chat Density: [Compact ▼]                    │
│                                              │
│ 🎤 Voice Defaults                            │
│ Wake Word: [NAVI _____]                      │
│ Default STT: [Auto (platform best) ▼]        │
│ Voice Feedback: [Toggle]                     │
│                                              │
│ 🔒 Security                                  │
│ App Lock: [Toggle]                           │
│ Lock Timeout: [5 minutes ▼]                  │
│ Clear Chat Cache on Lock: [Toggle]           │
│                                              │
│ 📊 Data & Storage                            │
│ Chat Cache Size: 142 MB                       │
│ [Clear Chat Cache]                           │
│ [Clear All Local Data]                       │
│                                              │
│ ℹ️ About                                     │
│ Navivox v1.0.0                               │
│ Flutter 3.x • dartssh2 2.17                  │
│ [View Licenses]                              │
└──────────────────────────────────────────────┘
```

## 3. Component Library

### 3.1 Shared Components

```dart
// Navigation
AppScaffold          // Adaptive shell (mobile: bottom nav, desktop: left rail)
ConnectionStatusBar  // Bottom status bar (desktop) or top chip (mobile)
ServerSwitcher       // Dropdown/popup for active server
AgentPill            // Active agent indicator, tappable
ErrorRecoverySheet   // Modal bottom sheet with error + recovery action

// Chat
MessageBubble        // Text message with markdown, timestamps, status
ToolCallCard         // Expandable tool execution card
VoiceMessageBubble   // Waveform + play + transcript
TypingIndicator      // Animated dots for assistant thinking
DateSeparator        // "Today" / "Yesterday" / date divider
MessageComposer      // Text input + attach + mic + send
VoiceControlBar      // Recording/command mode bar

// Config
ConfigSectionCard    // Overview card with summary + nav arrow
ConfigFormField      // Typed form field (string/int/bool/enum/secret)
SecretStatusIndicator // Shows only status, not value
ConfigDiffViewer     // Before/after diff with validation
ApplyConfirmSheet    // Sensitive change confirmation dialog
BiometricGate        // local_auth prompt wrapper

// Agents
AgentCard            // Agent summary card
AgentSwitcherSheet   // Bottom sheet with agent list
WorkspaceValidator   // Directory existence/access checker

// Servers
ServerCard           // Server summary with status dot
ServerForm           // Add/edit server form
HostKeyVerifier      // Fingerprint display + trust button

// Keys
KeyCard              // Key summary with type/fingerprint/servers
KeyImportPicker      // File picker + parse + preview
KeyGenerateDialog    // Key type selection + passphrase

// Shared
StatusBadge          // Colored status chip
EmptyState           // Icon + message + action for empty lists
LoadingIndicator     // Consistent loading spinner/shimmer
SectionHeader        // Section title with optional action
```

### 3.2 Design Token Map

```dart
class NavivoxTheme extends ThemeExtension<NavivoxTheme> {
  // Message bubbles
  final Color userBubbleColor;
  final Color agentBubbleColor;
  final Color toolCallCardColor;
  final BorderRadiusGeometry bubbleRadius;

  // Status
  final Color onlineColor;
  final Color offlineColor;
  final Color gormesDetectedColor;
  final Color warningColor;
  final Color errorColor;

  // Voice
  final Color voiceActiveColor;
  final Color wakeWordColor;

  // Tool calls
  final Color riskLowColor;
  final Color riskMediumColor;
  final Color riskHighColor;
  final Color approvalPendingColor;

  // Config
  final Color secretFieldColor;
  final Color sensitiveFieldColor;
  final Color diffAddedColor;
  final Color diffRemovedColor;
  final Color validationErrorColor;

  // Typography
  final TextStyle chatMessageStyle;
  final TextStyle codeBlockStyle;
  final TextStyle timestampStyle;
  final TextStyle serverNameStyle;
  final TextStyle agentNameStyle;
}
```

## 4. First-Run Wizard Design

The first-run journey has ten product steps from the PRD, grouped into compact
wizard screens so users do not see ten separate route transitions. The visual
sequence below starts with Welcome, then branches into import/manual server,
keys, host verification, probe, pairing, agent selection/creation,
voice/language, and finally chat.

### Step 1: Welcome

```
┌──────────────────────────────────────────────┐
│                                              │
│              🚀 Welcome to Navivox           │
│                                              │
│    Connect to your Gormes agent servers      │
│    over SSH with chat, voice, and full       │
│    configuration management.                 │
│                                              │
│    Let's set up your first connection.       │
│                                              │
│    ┌──────────────────────────────────┐      │
│    │ 📥 Import from Termius            │      │
│    │    Import your existing servers,  │      │
│    │    keys, and host fingerprints    │      │
│    └──────────────────────────────────┘      │
│                                              │
│    ┌──────────────────────────────────┐      │
│    │ ➕ Add Server Manually            │      │
│    │    Enter hostname, port, user,    │      │
│    │    and key manually               │      │
│    └──────────────────────────────────┘      │
│                                              │
│    ┌──────────────────────────────────┐      │
│    │ 🔑 I'll add keys later            │      │
│    │    Skip to server setup, add      │      │
│    │    SSH keys in Settings           │      │
│    └──────────────────────────────────┘      │
└──────────────────────────────────────────────┘
```

### Step 2: Import or Add Server

**Termius Import:**
- File picker opens for `.json` or `.termius` files
- Parse and preview: "Found 3 hosts, 2 SSH keys, 1 group"
- Select which to import (checkboxes)
- Warning for password-only identities
- Import → proceed to Step 3

**Manual Add:**
- Hostname, Port (22 default), Username fields
- Optional display name
- SSH key selector (or "Add key later")
- Save → proceed to Step 3

### Step 3: SSH Key

**Generate:**
- Key type: Ed25519 (default), RSA, ECDSA
- Optional passphrase
- Generate → show public key fingerprint
- "Copy public key to server's authorized_keys"

**Import:**
- File picker for private key
- Parse and show type, fingerprint
- If encrypted, prompt for passphrase
- Save to secure storage

### Step 4: Connect & Verify Host

- Connect to server with selected key
- Show connection progress
- First time: "Host key fingerprint: SHA256:AbCdEf..."
  - [Trust and Connect] [Cancel]
- If Gormes probe succeeds: green check + version
- If Gormes not found: "Generic SSH server. Terminal access only."
  - Option to still proceed or go back

### Step 5: Pair Device

- If first device: "This will be the Owner device."
- If additional: "Requesting Operator access. An Owner/Admin must approve."
- After pairing: role assigned, token stored

### Step 6: Select Agent

- List existing agents from server
- Or create new agent: name, workspace directory (existing path)
- Validate directory exists and is accessible
- Select agent → "Agent 'mineru' is now active"

### Step 7: Voice Setup (Optional, Skip Available)

- Wake word (default: NAVI)
- Default voice provider/voice selection
- Language preferences
- [Test voice] button to hear sample
- [Skip] or [Finish Setup]

### Completion:

```
┌──────────────────────────────────────────────┐
│                                              │
│              ✅ Setup Complete!               │
│                                              │
│    Connected to: gormes-prod                 │
│    Active agent: mineru                      │
│    Voice: Ready (wake: NAVI)                 │
│                                              │
│    ┌──────────────────────────────────┐      │
│    │      💬 Open Chat                │      │
│    └──────────────────────────────────┘      │
│                                              │
│    Add more servers or keys in Settings.      │
└──────────────────────────────────────────────┘
```

## 5. Icon System

### 5.1 Material Icons Usage

| Context | Icon | `Icons.*` |
|---------|------|-----------|
| Chats tab | chat_bubble | `Icons.chat_bubble_outline` |
| Servers tab | dns/server | `Icons.dns_outlined` |
| Agents tab | smart_toy | `Icons.smart_toy_outlined` |
| Config tab | settings | `Icons.settings_outlined` |
| Keys tab | key | `Icons.key` |
| Terminal tab | terminal | `Icons.terminal` |
| Send message | send | `Icons.send_rounded` |
| Mic/voice | mic | `Icons.mic_none` |
| Recording | mic + pulse | Custom animated |
| Attach file | attach_file | `Icons.attach_file` |
| Online status | circle | `Icons.circle` (green) |
| Offline status | circle | `Icons.circle` (gray) |
| Warning | warning | `Icons.warning_amber_rounded` |
| Error | error | `Icons.error_outline` |
| Success | check_circle | `Icons.check_circle_outline` |
| Tool running | engineering | `Icons.engineering` |
| Tool complete | check | `Icons.check` |
| Tool failed | close | `Icons.close` |
| Tool approval | lock | `Icons.lock_outline` |
| Secret field | visibility_off | `Icons.visibility_off` |
| Secret set | shield | `Icons.shield` |
| Biometric | fingerprint | `Icons.fingerprint` |
| Host key | vpn_key | `Icons.vpn_key` |
| Gormes detected | check | `Icons.check` |
| No Gormes | help | `Icons.help_outline` |
| Edit | edit | `Icons.edit` |
| Delete | delete | `Icons.delete_outline` |
| Add | add | `Icons.add` |
| Import | file_download | `Icons.file_download` |
| Export | file_upload | `Icons.file_upload` |
| Refresh | refresh | `Icons.refresh` |
| Copy | content_copy | `Icons.content_copy` |
| Expand | expand_more | `Icons.expand_more` |
| Collapse | expand_less | `Icons.expand_less` |

## 6. Empty States

### Chats (no messages)
```
┌──────────────────────────────────────────────┐
│                                              │
│                    💬                         │
│              No messages yet                  │
│    Start a conversation with your agent       │
│                                              │
│    Try:                                       │
│    "What's the server status?"                │
│    "List my agents"                           │
│    "Run the test suite"                       │
│                                              │
└──────────────────────────────────────────────┘
```

### Servers (none)
```
┌──────────────────────────────────────────────┐
│                                              │
│                    🖥️                         │
│              No servers yet                   │
│    Add a Gormes agent server to get started   │
│                                              │
│    [Import from Termius]                      │
│    [Add Server Manually]                      │
│                                              │
└──────────────────────────────────────────────┘
```

### Agents (none)
```
┌──────────────────────────────────────────────┐
│                                              │
│                    🤖                         │
│              No agents found                  │
│    Create an agent on your Gormes server      │
│    to start chatting                          │
│                                              │
│    [+ Create Agent]                           │
│                                              │
└──────────────────────────────────────────────┘
```

### Keys (none)
```
┌──────────────────────────────────────────────┐
│                                              │
│                    🔑                         │
│              No SSH keys yet                  │
│    Add a key to connect to your servers       │
│                                              │
│    [Generate Key]                             │
│    [Import Key]                               │
│                                              │
└──────────────────────────────────────────────┘
```

## 7. Error States

### Connection Failed
```
┌──────────────────────────────────────────────┐
│ ⚠️ Connection Failed                          │
│                                              │
│ Could not connect to gormes-prod              │
│ (192.168.1.100:22)                            │
│                                              │
│ Possible causes:                              │
│ • Server is unreachable                       │
│ • SSH key was rejected                        │
│ • Host key has changed                        │
│                                              │
│ [Try Again]  [Edit Server]  [Open Terminal]   │
└──────────────────────────────────────────────┘
```

### Gormes Not Found
```
┌──────────────────────────────────────────────┐
│ ⚠️ Gormes Not Installed                       │
│                                              │
│ The server gormes-prod does not have Gormes   │
│ installed or the 'gormes' command is not in   │
│ the PATH.                                     │
│                                              │
│ You can still use this server as a generic    │
│ SSH terminal, but chat, voice, config, and    │
│ agent features require Gormes.                │
│                                              │
│ [Open Terminal]  [Installation Guide]  [Skip] │
└──────────────────────────────────────────────┘
```

### Secret Storage Unavailable (Linux)
```
┌──────────────────────────────────────────────┐
│ ⚠️ Secure Storage Not Available               │
│                                              │
│ Navivox requires a secret service to store    │
│ SSH keys securely.                            │
│                                              │
│ On Linux, install one of:                     │
│ • gnome-keyring (GNOME)                       │
│ • kwalletmanager (KDE)                        │
│                                              │
│ Required packages:                            │
│ • libsecret-1-0                               │
│ • libjsoncpp1                                 │
│                                              │
│ [Retry]  [Dismiss]                            │
└──────────────────────────────────────────────┘
```

## 8. Loading States

### Connecting to Server
```
┌──────────────────────────────────────────────┐
│ Connecting to gormes-prod...                  │
│ ████████░░░░░░░░░░░░                          │
│                                              │
│ • Authenticating with SSH key...    ✅         │
│ • Verifying host fingerprint...     🔄         │
│ • Probing for Gormes...             ⏳         │
│ • Starting navivox channel...       ⏳         │
└──────────────────────────────────────────────┘
```

### Config Applying
```
┌──────────────────────────────────────────────┐
│ Applying Configuration...                     │
│                                              │
│ • Validating changes...              ✅         │
│ • Writing config.toml...             🔄         │
│ • Writing secrets...                 ⏳         │
│ • Reloading runtime...               ⏳         │
│                                              │
│ Please wait, this may take a moment.          │
└──────────────────────────────────────────────┘
```

## 9. Responsive Breakpoints

| Breakpoint | Width | Layout |
|------------|-------|--------|
| Compact | < 600dp | Single column, bottom nav, full-screen sheets |
| Medium | 600-840dp | Two column (list + detail), side nav rail |
| Expanded | 840-1200dp | Two column (list + detail), side nav rail |
| Large | 1200-1600dp | Three column (nav + list + detail) |
| Extra Large | > 1600dp | Three column with wider content area |
