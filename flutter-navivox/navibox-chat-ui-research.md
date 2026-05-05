# Navivox Chat UI Research

Status: planning draft
Source: analysis of 15+ Flutter open-source chat/UI projects (2025-2026)

## 1. Projects Analyzed

### 1.1 Chat SDKs (Production-Ready)

| Project | Stars | License | Relevance | Verdict |
|---------|-------|---------|-----------|--------|
| **flyerhq/flutter_chat_ui** | 2,262 | Apache 2.0 | ⭐⭐⭐⭐⭐ | Best-in-class chat SDK. Backend-agnostic, modular, actively maintained. |
| **SimformSolutionsPvtLtd/flutter_chatview** | Active | MIT | ⭐⭐⭐⭐ | Feature-rich with reactions, replies, voice messages, link previews. |
| **Tealseed-Lab/simple_chat** | 88 | MIT | ⭐⭐⭐ | Clean, minimal, supports custom message cells. Good reference for simplicity. |
| **v-chat-sdk/v_chat_bubbles** | New | - | ⭐⭐⭐ | Multi-platform bubble styles (Telegram, WhatsApp, Messenger, iMessage). |
| **chat_bubbles** (prahack) | 103 | - | ⭐⭐⭐ | WhatsApp-style bubbles, voice waveform, swipe actions, grouping. |
| **modern_chat_bubbles** | New | - | ⭐⭐⭐ | Glassmorphic/neomorphic designs, fluid animations. |
| **MOUKZ/chat_package** | - | - | ⭐⭐ | Voice record with slide-to-cancel, press-and-hold audio. |

### 1.2 AI Agent Chat Interfaces

| Project | Stars | Relevance | Key Takeaways |
|---------|-------|-----------|---------------|
| **flutter/ai** (Flutter AI Toolkit) | 252 | ⭐⭐⭐⭐ | Official Flutter AI chat, streaming, voice input, pluggable LLM. |
| **dartantic_chat** | - | ⭐⭐⭐ | Fork of flutter/ai, multi-provider, tool calling, voice, drag-drop. |
| **flutter_gen_ai_chat_ui** | - | ⭐⭐⭐ | Modern streaming text animations, markdown, result renderer registry. |
| **flutter_ai_agent_tool** | - | ⭐⭐⭐ | Complete agent toolkit: streaming, memory, tools, pre-built UI. |
| **dart_agent_core** | 15 | ⭐⭐⭐⭐ | Agent framework with streaming events, tool use, controller hooks. |

### 1.3 Flutter Templates & Architecture

| Project | Stars | Key Takeaways |
|---------|-------|---------------|
| **momshaddinury/flutter_template** | 128 | Clean Arch + Riverpod + go_router. Best Go template reference. |
| **AhmedLSayed9/deliverzler** | 715 | DDD + Riverpod + GoRouter. Code-gen variant is closest to our stack. |
| **wednesday-solutions/flutter_template** | 399 | Riverpod + Drift + Freezed + testing. Enterprise-grade template. |
| **Gradoid/scalable_flutter_app_starter** | 180 | flutter_bloc + go_router. Good separation-of-concerns patterns. |
| **cevheri/flutter-bloc-advanced** | - | Feature-first clean arch, responsive shell, role-based routing. |

### 1.4 Admin/Config UIs

| Project | Stars | Key Takeaways |
|---------|-------|---------------|
| **abuanwar072/flutter-responsive-admin-panel** | 7,090 | Dashboard layout, charts, tables, responsive design patterns. |
| **FlutterFlareLine/FlareLine** | 149 | Modern admin with localization, auth, firebase. |
| **libsFlutter/flutter_adminpanel** | New | Universal adapter system, resource-based CRUD, schema-driven forms. |
| **serverpod_admin_dashboard** | - | Full customization with builder pattern. Schema-driven CRUD from API. |

### 1.5 Telegram Clones (UI Reference)

| Project | Stars | Notes |
|---------|-------|-------|
| **babakcode/flutter_chat** | 45 | Telegram-like chat app. Material 3 branch. |
| **codingcafe1/telegram** | 53 | Firebase + Telegram UI clone. |
| **Yogabayu/clone_telegram_flutter** | 11 | Pure UI clone. Clean code. |
| **dima-xd/nullgram** | 3 | Custom Telegram client from scratch in Flutter. Ambitious. |
| **team113/messenger** | 31 | Production messenger, 40 contributors, 76 releases. |

## 2. Which Projects to Adopt, Inspire From, or Skip

### 2.1 Adopt (Direct Dependency)

| Package | Why | Navivox Integration |
|---------|-----|-------------------|
| **flyerhq/flutter_chat_ui** v2 | Best chat SDK. Backend-agnostic → works with Navivox protocol. Streaming text messages via `flyer_chat_text_stream_message`. Custom message types via `Builders`. | Use as the core chat UI widget. Register custom `ToolCallMessage` type for tool call cards. Use `ChatController` → bridge to Navivox channel events. |
| **flyerhq/flyer_chat_text_stream_message** | Fade-in animated streaming text with markdown. Exactly what we need for `chat.update` events. | Register as message type for streaming assistant responses. |

**Why not others:**
- `flutter_chatview`: Too opinionated about backend (firebase), more complex API
- `chat_bubbles`: Widget-level only, we need a full chat framework
- `modern_chat_bubbles`: Nice designs but not a framework

### 2.2 Inspire From (Design Patterns, No Direct Dep)

**Chat Bubbles Package (prahack):**
- Voice message waveform visualization (`waveformData` + scrub interaction) → Our `VoiceRecordButton`
- Swipe-to-reply/delete on message bubbles (`SwipeableBubble`) → Navivox swipe actions
- Message grouping by consecutive sender with tails (`BubbleGroupBuilder`) → Navivox chat grouping
- Typing indicator animations (`TypingIndicator`, `TypingIndicatorWave`) → Navivox typing state

**MOUKZ/chat_package:**
- Press-and-hold audio recording with slide-to-cancel → Our voice input UX
- Wave animation style for recording state (`WaveAnimationStyle`) → Our mic UI
- Recording button with smooth mic/send icon transition → Our composer mic button

**Tealseed-Lab/simple_chat:**
- ViewFactory pattern for custom message cell registration → Our tool call card registration
- Force new block for different message types → Our tool call separation from text
- Loading indicator message type → Our streaming indicator

**flutter_gen_ai_chat_ui:**
- Result Renderer Registry pattern → Navivox artifact/tool result rendering
- Streaming text with word-by-word animations → Our `chat.update` rendering
- Example questions / welcome message → Navivox first-run chat suggestions

**dart_agent_core:**
- `runStream()` yielding typed events → Our Navivox channel event stream
- Controller hooks (before/after tool call) → Our tool approval flow
- Loop detection for tool calls → Navivox progress tracking

**deliverzler (AhmedLSayed9):**
- Riverpod code-gen + GoRouter + Freezed architecture
- Feature-based folder structure with presentation/domain/infrastructure
- DDD layered architecture adapted for Riverpod

**flutter_template (momshaddinury):**
- Clean Architecture with Riverpod + go_router + retrofit
- Feature-based organization
- Theme system with Figma semantic tokens → Our design tokens

**flutter_template (wednesday-solutions):**
- Connector pattern (separate data-fetching widgets from UI widgets)
- Riverpod + Drift + Freezed combo (matches our stack exactly)
- Golden tests, E2E tests, CI/CD setup

**serverpod_admin_dashboard:**
- Custom builder pattern for full UI replacement
- Schema-driven CRUD from API → Our config schema → form generation
- Role-based access control at UI level → Our pairing role gates

### 2.3 Skip (Not Suitable)

- **Flutter AI Toolkit** / **dartantic_chat**: Tied to Firebase/LLM providers. Navivox talks to Gormes, not directly to LLMs.
- **flutter_ai_agent_tool**: Agent orchestration happens on Gormes server, not in Flutter app.
- **Telegram clones**: UI reference only. Their architecture (Firebase, hardcoded users) doesn't match Navivox's SSH+protocol model.
- **abuanwar072 admin panel**: Too generic. Navivox needs schema-driven config forms, not static dashboards.

## 3. Key Chat UI Patterns to Implement

### 3.1 Message Bubble System

Adopt Flyer Chat's approach but extend:

```
┌─────────────────────────────────────────┐
│ [Avatar]  Agent Name                     │
│           ┌──────────────────────┐       │
│           │ Hello! How can I help│       │
│           │ you today?           │       │
│           └──────────────────────┘       │
│           12:30 PM  ✓✓                   │
│                                          │
│           ┌──────────────────────────────┤
│           │ Here's the analysis:         │
│           │ ```json                      │
│           │ {"status": "ok"}             │
│           │ ```                          │
│           └──────────────────────────────┤
│           12:31 PM                        │
│                                          │
│ ┌────────────────────────────────────┐   │
│ │ 🔧 find_files            Running  │   │
│ │ Scanning: /home/user/project       │   │
│ │ ████████░░░░░░░░ 60%              │   │
│ │ [Approve]          [Deny]         │   │
│ └────────────────────────────────────┘   │
│           12:31 PM                        │
│                                          │
│ ┌─────────────────────────────┐          │
│ │         My message       ✓✓ │          │
│ └─────────────────────────────┘          │
│                             12:32 PM      │
└─────────────────────────────────────────┘
```

**Message Types:**
1. `TextMessage` — Standard text with markdown (use Flyer Chat)
2. `StreamingTextMessage` — Streaming text with cursor (use Flyer Chat)
3. `VoiceMessage` — Waveform + play button + transcript
4. `ToolCallCard` — Tool name, status, progress, approval, artifacts
5. `SystemMessage` — "Agent switched to mineru", "Connected to server"
6. `ErrorMessage` — Redacted errors with recovery actions

**Bubble Grouping:**
- Consecutive messages from same sender → group, only last shows tail
- Tool call cards always start new block (never grouped with text)
- Voice messages always start new block
- Time gap > 5 min → new block even if same sender

### 3.2 Message Composer

```
┌──────────────────────────────────────────────┐
│ 📎  [Text input area with placeholder]  🎤  ▶ │
└──────────────────────────────────────────────┘
```

States:
- **Default**: Paperclip (attachment), text field, mic, send (when text non-empty)
- **Recording**: Wave animation replaces mic icon, slide-to-cancel behavior
- **Voice active**: Continuous voice mode indicator below composer
- **Disabled**: "Connecting..." / "No agent selected" placeholder

### 3.3 Tool Call Cards

```
┌─────────────────────────────────────────────┐
│ 🔧 Tool Name                         Status │
│ Selected Agent: mineru          Workspace   │
│ ─────────────────────────────────────────── │
│ Preview: "Searching for files..."           │
│                                             │
│ ████████████░░░░░░░░ 60%                    │
│ Elapsed: 2.3s                               │
│ Risk: 🟡 Medium  ⚠️ Mutating                │
│                                             │
│ [⏸ Stop]  [Approve]  [Deny]                │
│ ─────────────────────────────────────────── │
│ Artifacts:                                  │
│ 📄 search_results.json (23KB)  [View]       │
│ 🖼️ screenshot.png (1.2MB)      [View]       │
└─────────────────────────────────────────────┘
```

States:
- `started` — Tool name, preview, start time
- `progress` — Progress bar/indicator, elapsed time, summary updates
- `completed` — Result summary, artifacts, green checkmark
- `failed` — Redacted error, retry option
- `cancelled` — Stopped by user, dimmed
- `blocked` — Needs approval, prominent Approve/Deny buttons
- `pending_approval` — Lock icon, approval prompt

### 3.4 Voice Control Bar

```
┌─────────────────────────────────────────────┐
│ 🎤 NAVI Listening...                    ⏹   │
│ Partial: "switch agent min..."              │
│ ▓▓▓▓▓▓▓░░░░░░░░░░░ -12dB                   │
│ [Text Fallback]                             │
└─────────────────────────────────────────────┘
```

States:
- **Idle**: Hidden
- **Listening (no wake word)**: Waveform, VU meter, "Listening..."
- **Wake word detected**: "NAVI" highlight, confidence indicator
- **Command recognized**: Parsed command shown ("Switching to agent mineru")
- **Processing**: Spinner, "Processing..."
- **Speaking**: Agent voice indicator, volume visualization

### 3.5 Agent Switcher (Overlay)

```
┌──────────────────────────────────────┐
│ Select Agent                    ✕    │
│──────────────────────────────────────│
│ ● mineru (default)      🟢 online    │
│   /home/xel/gormes-agent             │
│   Voice: ElevenLabs - Adam           │
│──────────────────────────────────────│
│ ○ builder                🟢 online   │
│   /home/xel/projects/build           │
│   Voice: OpenAI - Nova               │
│──────────────────────────────────────│
│ ○ archived-agent         ⚫ offline  │
│──────────────────────────────────────│
│ [+ Create New Agent]                 │
└──────────────────────────────────────┘
```

### 3.6 Config Diff Viewer

```
┌──────────────────────────────────────┐
│ Review Changes                  ✕    │
│──────────────────────────────────────│
│ 🔴 Hermes                          │
│   model: gpt-4 → gpt-4-turbo       │
│                                      │
│ 🔴 Telegram (SENSITIVE)             │
│   bot_token: [NOT SET] → [SET]      │
│                                      │
│ 🟡 Runtime                          │
│   max_tool_iterations: 10 → 20      │
│                                      │
│ Warnings:                            │
│ ⚠️ Model change requires restart    │
│ ⚠️ Token change affects all users   │
│                                      │
│ [Cancel]              [Apply All]    │
└──────────────────────────────────────┘
```

### 3.7 Secret Editor (Biometric-Gated)

```
┌──────────────────────────────────────┐
│ 🔐 Unlock Secret Editor              │
│──────────────────────────────────────│
│ Authenticate to manage secrets       │
│                                      │
│ [🔒 Touch ID / Face ID / PIN]        │
└──────────────────────────────────────┘

After unlock:
┌──────────────────────────────────────┐
│ Secrets                        ✕    │
│──────────────────────────────────────│
│ Telegram Bot Token                   │
│ Status: 🔴 Configured [REDACTED]     │
│ Source: env (GORMES_TELEGRAM_TOKEN)  │
│ [Rotate] [Delete]                    │
│──────────────────────────────────────│
│ Provider API Key                     │
│ Status: 🔴 Configured [REDACTED]     │
│ Source: SecretRef (1Password)        │
│ [Set New] [Delete] [Test Connection] │
│──────────────────────────────────────│
│ Gateway Proxy Key                    │
│ Status: 🟢 Not Configured            │
│ [Set]                                │
└──────────────────────────────────────┘
```

## 4. Responsive Layout Patterns

### 4.1 Mobile Chat Layout

```
┌──────────────────────┐
│ Server: my-server    │  ← Server switcher (tap)
│ Agent: mineru  ▼     │  ← Agent pill (tap for switcher)
├──────────────────────┤
│                      │
│  [Message list]      │
│                      │
│                      │
├──────────────────────┤
│ [Composer + Mic]     │
├──────────────────────┤
│ Chats │ Srv │ Config │  ← Bottom nav (primary 3)
└──────────────────────┘
```

### 4.2 Desktop Chat Layout

```
┌──────┬──────────────────────────────────────┐
│      │ Server: my-server  Agent: mineru ▼   │
│ Nav  ├──────────────────────────────────────┤
│      │                                      │
│ 💬   │  ┌─[Thread List]──┬─[Chat Area]──┐   │
│ 🖥️   │  │ Thread 1      │ Messages...   │   │
│ 🤖   │  │ Thread 2 ●    │               │   │
│ ⚙️   │  │ Thread 3      │ Tool card...  │   │
│ 🔑   │  │               │               │   │
│ ⌨️   │  │               │               │   │
│      │  │               ├───────────────┤   │
│      │  │               │ [Composer]    │   │
│      │  └───────────────┴───────────────┘   │
├──────┴──────────────────────────────────────┤
│ Status: Connected  |  Gormes v1.2.3  |  🔒  │
└─────────────────────────────────────────────┘
```

### 4.3 Desktop Config Layout

```
┌──────┬──────────────────────────────────────┐
│      │ Config: my-server              [✕]   │
│ Nav  ├──────────┬───────────────────────────┤
│      │ Sections │ Form Area                  │
│      │          │                            │
│      │ Overview │ Provider: [OpenAI ▼]      │
│      │ Provider │ Model: [gpt-4-turbo ▼]    │
│      │ Channels │ Endpoint: [api.openai...]  │
│      │ Agents   │ API Key: [REDACTED] [Set] │
│      │ Tools    │                            │
│      │ Voice    │ [Review Changes] [Apply]  │
│      │ Runtime  │                            │
│      │ Browser  │                            │
│      │ Security │                            │
│      │ Secrets  │                            │
│      │ Advanced │                            │
└──────┴──────────┴───────────────────────────┘
```

## 5. Design Tokens (Inspired by Research)

### 5.1 Typography

| Token | Size | Weight | Use |
|-------|------|--------|-----|
| `displayLarge` | 32 | 700 | Server name in header |
| `headlineMedium` | 24 | 600 | Section titles |
| `titleMedium` | 16 | 500 | Message sender name, agent name |
| `bodyLarge` | 16 | 400 | Message text |
| `bodyMedium` | 14 | 400 | Timestamps, status, metadata |
| `labelLarge` | 14 | 500 | Button text, input labels |
| `labelMedium` | 12 | 500 | Status badges, tool call status |
| `codeStyle` | 13 | 400 | Monospace for code blocks |

### 5.2 Color System

```
Primary:      Navivox Blue (#2563EB)
On Primary:   White (#FFFFFF)
Secondary:    Slate (#64748B)
Surface:      Light (#F8FAFC) / Dark (#0F172A)
Error:        Red (#EF4444)
Warning:      Amber (#F59E0B)
Success:      Green (#22C55E)
Tool Call:    Purple (#7C3AED)
Voice Active: Cyan (#06B6D4)
Server Online:Green (#22C55E)
Server Offline:Gray (#94A3B8)
```

### 5.3 Spacing

| Token | Value | Use |
|-------|-------|-----|
| `xxs` | 4 | Icon-text gap |
| `xs` | 8 | Within bubbles |
| `sm` | 12 | Between grouped messages |
| `md` | 16 | Standard padding |
| `lg` | 24 | Section spacing |
| `xl` | 32 | Large section breaks |

### 5.4 Border Radius

| Token | Value | Use |
|-------|-------|-----|
| `bubble` | 16 | Message bubbles |
| `card` | 12 | Tool call cards, config cards |
| `input` | 24 | Composer (pill-shaped) |
| `button` | 8 | Standard buttons |

## 6. Animation Patterns

### 6.1 Message Animations

- **New message (not current user)**: Slide up + fade in, 300ms
- **New message (current user)**: Slide up from right, 250ms
- **Streaming text**: Word-by-word fade in, 30ms per word (from `flutter_gen_ai_chat_ui`)
- **Tool call card appear**: Expand from collapsed, 200ms
- **Tool call progress update**: Smooth progress bar transition
- **Message delete**: Fade out + shrink, 200ms

### 6.2 Voice Animations

- **Mic button idle → recording**: Scale + color shift, 200ms
- **Recording waveform**: Animated bars (from `chat_package` WaveAnimationStyle)
- **Wake word detected**: Pulse glow on NAVI chip
- **Speaking**: Equalizer animation on agent avatar/speaker icon

### 6.3 Navigation Animations

- **Tab switch**: Crossfade (from `deliverzler`)
- **Push detail**: Slide right → left
- **Bottom sheet**: Slide up
- **Dialog**: Scale from center

## 7. Accessibility Requirements

- All interactive elements have `Semantics` labels
- Message bubbles announce: sender, time, content preview
- Tool call cards announce: tool name, status, risk
- Voice button announces current recording state
- Config form fields have proper labels + error announcements
- Minimum touch target 48x48dp for all interactive elements
- Support screen reader navigation order
- High contrast mode support via theme variants

## 8. Key Decision: Build vs Adopt Chat UI

**Decision: Adopt `flyerhq/flutter_chat_ui` v2 as foundation, extend with custom message types.**

Rationale:
- **Saves months** of chat UI development (message grouping, animations, scroll, input bar, theming)
- **Backend-agnostic** — works with our custom Navivox protocol over SSH
- **Modular** — we only use what we need, swap out pieces
- **2,200+ stars, 80 contributors** — community-validated, actively maintained
- **v2 has streaming text** (`flyer_chat_text_stream_message`) — exactly our `chat.update` use case
- **Custom message types** via `Builders` — we register `ToolCallMessage`, `VoiceMessage`, etc.

What we build custom:
- **Tool call cards** — custom `ChatMessage` subtype with progress, approval, artifacts
- **Voice message bubbles** — waveform visualization, playback speed
- **Agent switcher** — bottom sheet overlay
- **Server context bar** — server selector + agent pill in top bar
- **Config forms** — schema-driven from `config.schema` response
- **Secret editor** — biometric-gated write-only UI
- **Terminal** — separate view using xterm.dart (not part of chat)
- **First-run wizard** — multi-step setup flow
