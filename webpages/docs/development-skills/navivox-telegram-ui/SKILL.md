---
name: navivox-telegram-ui
description: Use when planning, implementing, or reviewing Flutter Navivox chat/contact UI that should feel closer to Telegram, including chat-thread chrome, bottom navigation removal, bubbles, composer, profile contacts, action sheets, voice affordances, or Telegram/Flutter clone reference research.
---

# Navivox Telegram UI

## Mission

Make the Navivox Flutter app feel Telegram-like while keeping it Gormes-owned. Borrow navigation density, bubble rhythm, composer behavior, sheets, and contact-list scanning; do not import Telegram network assumptions, TDLib, MTProto, Telegram credentials, or consumer-messenger scope unless a future row explicitly authorizes it.

## Use This For

- Chat thread UX should resemble Telegram more closely.
- Mobile chat has bottom app navigation or other non-chat chrome.
- Message bubbles, timestamps, tails, grouping, status chips, or composer need polish.
- Profile contacts should behave like Telegram chats/contacts.
- Voice transcript bubbles, mic affordances, action trays, long-press menus, or tool cards need Telegram-like treatment.
- The user provides Telegram official repos, Flutter clones, or UI kits as references.

Do not use this for Gormes backend provider/runtime work unless UI changes require a typed Navivox API contract; then route backend work through `gormes-tdd-slice` or `gormes-interface-designer`.

## Current App Root

The Navivox Flutter app is now a sibling repo, not an in-repo `flutter-navivox/` tree.

- Local app root: `/home/xel/git/gormes/navivox-app`
- Before UI work, run `test -d /home/xel/git/gormes/navivox-app` and then `cd /home/xel/git/gormes/navivox-app`.
- Historical Gormes progress rows may mention `flutter-navivox/app`, `../navivox-app/app`, or `/home/xel/git/sages-openclaw/workspace-mineru/navivox-app/app`; treat those as old evidence. For new work and tests, use the sibling repo path above.
- Keep Gormes backend/progress edits in `gormes-agent` and Flutter implementation edits in `navivox-app` unless Juan explicitly scopes both repos.

## Reference Rules

Treat references as design evidence, not dependencies.

Official Telegram repos to study for behavior patterns:

- Telegram Android: `https://github.com/DrKLO/Telegram`
- Telegram iOS: `https://github.com/TelegramMessenger/Telegram-iOS`
- Telegram X Android: `https://github.com/TelegramMessenger/Telegram-X-Android`
- Telegram Desktop: `https://github.com/telegramdesktop/tdesktop`
- Telegram Web: `https://github.com/morethanwords/tweb`
- TDLib: `https://github.com/tdlib/td`

Flutter/community references to verify before adoption:

- `telware_cross_platform`
- `telega2`
- Telegram Clone by `birukbr7`
- Telegram Clone UI 2021
- `telegram_ios_ui_kit`
- `tele_web_app`
- `v_chat_bubbles`

Before adding any package or copying an implementation, verify: source URL, license, maintenance status, platform support, API ownership, accessibility, and whether it introduces TDLib/API credential requirements. If not verified, use it only as visual inspiration.

## Product Translation

Telegram pattern | Navivox translation
---|---
Chat list | Profile contact list keyed by `server_id + profile_id`, with active-turn `typing…` state and Telegram-like search/filter
Chats bottom tab | Contact list may have top-level app nav; chat thread should not show bottom nav
Message bubbles | User right, assistant left, grouped tails, compact timestamps, assistant typing/streaming indicator
Read ticks | Local send/queued/streaming/done/error status, not read receipts
Attachment button | Telegram-style share tray for upload file, media, workspace file, approvals, tools, workspace/config, future files
Emoji/composer | Quick emoji strip inserts into the message field without leaving chat
Voice message | Device transcript bubble, explicit STT unavailable state, TTS/read-aloud action, tappable continuous voice banner, and voice control sheet
Long press | Selectable text, copy, forward to another profile/contact, retry, inspect event/tool, reveal redacted fields only when allowed
Telegram backend | Never: Navivox talks to Gormes gateway only

## Implementation Workflow

1. **Scope the surface**
   - Work under `/home/xel/git/gormes/navivox-app` for Flutter UI changes unless docs/tests require Gormes repo updates.
   - State whether the change is chat thread, contact list, composer, bubbles, voice, or action sheet.

2. **Write/adjust widget tests first**
   - Use `flutter_test` for shell and widget behavior.
   - Pin mobile layout with a mobile surface size.
   - Test absence as well as presence: for example, chat thread has no `NavigationBar`; contact list still has one.

3. **Keep chat immersive**
   - Mobile chat thread should prioritize app bar, transcript, and composer.
   - Do not put top-level bottom navigation under the active chat thread.
   - Navigation to servers/settings belongs in contact list, drawers, app bar menus, or sheets.

4. **Prefer small local widgets**
   - Current app already owns `SimpleChatAdapter`; improve it incrementally.
   - Add package dependencies only after reference verification.
   - Keep server/profile selection state in `NavivoxChannel`; UI widgets should not infer backend state from raw config.

5. **Validate**
   - Run focused Flutter tests first, for example:
     `cd /home/xel/git/gormes/navivox-app && flutter test test/shared/app_shell_test.dart`
   - Then run the relevant broader Flutter suite when practical:
     `cd /home/xel/git/gormes/navivox-app && flutter test`
   - If generated files or progress docs changed, route through `gormes-git` before push.

## Common Mistakes

| Mistake | Fix |
|---|---|
| Adding Telegram/TDLib dependencies for UI polish | Use visual patterns only; Navivox backend is Gormes gateway. |
| Leaving bottom app nav inside mobile chat thread | Hide top-level nav on `/chats/<server>/<profile>` routes. |
| Treating Gormes profiles as Telegram accounts | Profiles are Gormes homes/config/runtime scopes. |
| Copying clone code without license/maintenance proof | Verify first; otherwise write local Flutter widgets. |
| Hiding broken STT/TTS controls | Keep voice controls visible with clear unavailable states, continuous voice status, and settings guidance. |
| Making UI pretty without tests | Add widget tests for the visible contract. |
