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

## Channel strategy

| Channel | Role | Use when |
|---|---|---|
| F-Droid | Primary trust channel | Launching to open-source Android, Termux, self-hosting, privacy, and developer-tool users who already expect auditable builds. |
| Google Play | Later reach channel | Expanding beyond the OSS Android audience after the app has release evidence, policy fit, and a support model. |
| Direct APK or GitHub release | Test and fallback channel | Capturing early smoke evidence, hashes, and operator testing before a store listing is ready. |

Do not treat Google Play as the first proof that Navivox belongs on Android.
F-Droid is the audience-fit proof: open-source Android users can inspect, build,
and install it without a Google account.

## Listing copy guardrails

Use this message shape for F-Droid-facing copy before the listing is real:

- Short summary: Navivox is an open-source Android companion for a user-owned Gormes runtime.
- Audience: Termux users, self-hosters, AI tinkerers, privacy-first users, and OSS Android users.
- Promise: pair the app to your own Gormes gateway over HTTP/WebSocket; do not imply a hosted SaaS assistant.
- Avoid saying: one-tap install, no setup required, hosted cloud assistant, or available on F-Droid.

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
