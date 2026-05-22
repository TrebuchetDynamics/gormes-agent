---
title: "Navivox and F-Droid"
description: "Why Navivox should treat F-Droid as its natural Android distribution channel, and what must be true before claiming availability there."
---

# Navivox and F-Droid

F-Droid is the open-source Android app store: closer to a Linux package manager
for Android than a commercial app mall. That makes it culturally aligned with
Navivox users who already live in Termux, self-hosting, privacy tooling, local
AI, and OSS Android workflows.

Navivox should treat F-Droid as a first-class distribution target, not an
afterthought. The same operators who install Gormes from Termux are the people
most likely to trust an Android companion app that is reproducible,
open-source, and usable without a Google account.

## Current status

Navivox is not documented as published on F-Droid yet. Until repository build
metadata, release artifacts, and store listing evidence exist, describe F-Droid
as a distribution target rather than an available install path.

Use the current Gormes/Android path today:

1. Install and operate Gormes from [Termux](../../install/termux/).
2. Configure the Navivox HTTP/WebSocket gateway with `gormes setup gateway`.
3. Pair the Android app using the token-redacted connect information or the
   generated QR image.

## Why F-Droid fits

- Open-source users already search F-Droid before Google Play.
- Privacy-first users expect no Google account dependency for core workflows.
- Termux users understand local services, wake locks, SSH, Tailscale, and
  self-hosted endpoints.
- AI tinkerers are more likely to accept a companion app that pairs to their
  own Gormes runtime instead of a hosted SaaS backend.
- F-Droid packaging sets a public expectation that the app build is auditable
  and reproducible.

## Audience fit matrix

| Audience | Why F-Droid fits |
|---|---|
| Termux users | Already treat Android like a Linux workstation and expect package-manager-style installs. |
| Self-hosters | Prefer companion apps that pair to their own services instead of opaque hosted accounts. |
| AI tinkerers | Want inspectable local tooling before trusting an assistant on a phone. |
| Privacy-first users | Avoid Google-account-only distribution and look for explicit permission boundaries. |
| OSS Android users | Discover tools through source-available, reproducible, community-reviewed channels. |

## Channel strategy

| Channel | Role | Use when |
|---|---|---|
| F-Droid | Primary trust channel | Launching to open-source Android, Termux, self-hosting, privacy, and developer-tool users who already expect auditable builds. |
| Google Play | Later reach channel | Expanding beyond the OSS Android audience after the app has release evidence, policy fit, and a support model. |
| Direct APK or GitHub release | Test and fallback channel | Capturing early smoke evidence, hashes, and operator testing before a store listing is ready. |

Do not treat Google Play as the first proof that Navivox belongs on Android.
F-Droid is the audience-fit proof: open-source Android users can inspect, build,
and install it without a Google account.

## Why not start with Google Play

Google Play is useful for reach later, but it is a poor first proof point for Navivox.

- Store policy review can mistake self-hosted gateway setup for an unfinished app.
- Privacy-first users may distrust a Google-account-only install path.
- Commercial-store screenshots and copy reward broad consumer positioning, not developer-tool clarity.
- F-Droid lets the first audience validate source, builds, permissions, and gateway pairing before a mass-market listing.

## Listing copy guardrails

Use this message shape for F-Droid-facing copy before the listing is real:

- Short summary: Navivox is an open-source Android companion for a user-owned Gormes runtime.
- Audience: Termux users, self-hosters, AI tinkerers, privacy-first users, and OSS Android users.
- Promise: pair the app to your own Gormes gateway over HTTP/WebSocket; do not imply a hosted SaaS assistant.
- Avoid saying: one-tap install, no setup required, hosted cloud assistant, or available on F-Droid.

## Permission and privacy guardrails

- Ask for only the permissions the pairing flow needs.
- Explain network access as communication with a local or user-configured Gormes gateway.
- Do not bundle analytics SDKs, proprietary crash reporters, ad networks, or account trackers.
- Document any broad-looking permission before release review so F-Droid users can audit the tradeoff.
- Screenshots must not show real gateway URLs, pairing tokens, chat payloads, workspace paths, or provider names.

## Metadata readiness checklist

- Keep the listing title, summary, and description focused on a user-owned Gormes companion, not a generic chatbot.
- Record the source repository, license, app ID, release tag, and build commit beside the listing copy.
- Prepare icon, screenshots, feature graphic, and alt text that show setup with placeholder endpoints only.
- Name Termux, self-hosting, privacy, local AI, and OSS Android in the long description so the right users recognize themselves.
- Keep Google Play language out of the F-Droid metadata except as a later reach channel in internal strategy notes.

## Build and artifact evidence

- Build the candidate app from a clean public checkout at the same tag and commit named in the listing notes.
- Record the Android or Flutter build command, SDK versions, dependency lockfiles, and any reproducibility caveats.
- Publish the APK SHA-256 beside the release tag, commit, and install smoke result.
- Mark direct APK or GitHub artifacts as test/fallback evidence until F-Droid metadata and source-build review are complete.
- Keep debug, unsigned, locally patched, or secret-bearing APKs out of the F-Droid handoff.

## Screenshot review checklist

- Show the expected path: Gormes running in Termux, Navivox pairing, then a local gateway conversation.
- Use placeholder gateway URLs, fake pairing tokens, demo profile names, and sample chat payloads only.
- Add captions or alt text that explain the app pairs to a user-owned Gormes runtime.
- Avoid Google Play badges, hosted-account claims, cloud-assistant screenshots, and generic chatbot positioning.
- Keep every screenshot reproducible from public fixtures or redacted demo profiles before submission.

## F-Droid submission handoff checklist

- Link the source repository, license, release tag, build commit, APK SHA-256, and clean install smoke result in one handoff note.
- Include F-Droid metadata status, screenshots, permission rationale, and first-run pairing copy for reviewer context.
- Name any reproducibility caveat or non-free dependency risk before opening the submission request.
- Keep Google Play, direct APK, and GitHub release notes labeled as adjacent channels, not proof of F-Droid availability.
- Track reviewer questions and follow-up fixes beside the handoff so availability claims wait for accepted metadata.

## First-run pairing acceptance checklist

- The first screen should say Navivox connects to a user-owned Gormes gateway, not a hosted account.
- Show Termux as the recommended operator path and pair through `gormes setup gateway` or the generated QR/connect information.
- Require users to confirm the gateway URL, profile name, and token source before the first conversation.
- Keep setup failure copy actionable: start Gormes, check HTTP/WebSocket reachability, regenerate redacted pairing info, or retry on the same network/VPN.
- Do not promise no setup, one-tap onboarding, Google-account sync, or cloud chatbot behavior in the listing or first-run flow.

## Release gate before claiming F-Droid availability

Do not say "install from F-Droid" until all of these are evidenced in the repo
or release lane:

- Android build flavor for the public Navivox app is reproducible without
  private services, secrets, or proprietary SDK requirements.
- App metadata, icon assets, license, screenshots, and summary copy are ready
  for an F-Droid listing.
- The app's first-run flow explains that it pairs with a user-owned Gormes
  gateway over HTTP/WebSocket.
- No token, URL, workspace path, session payload, or provider secret is bundled
  into the APK or screenshots.
- A release checklist records the exact commit, tag, artifact hash, and install
  smoke evidence.

## Positioning rule

Google Play can still matter later, but F-Droid is the trust-building channel
for the first open-source Android audience. Lead with the OSS/self-hosted story:
Navivox is the Android companion for a local Gormes runtime, not a closed cloud
assistant app.
