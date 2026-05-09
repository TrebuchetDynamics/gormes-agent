## Provenance

These files are vendored from the Hermes-Agent WhatsApp Baileys bridge
(origin: NousResearch/hermes-agent, scripts/whatsapp-bridge/).

- `bridge.js` — Baileys-based WhatsApp Web bridge with QR pairing, reconnect,
  session persistence, and mode routing (bot / self-chat).
- `package.json` / `package-lock.json` — Node dependency manifest.
- `allowlist.js` — Allowlist enforcement for bot-mode message filtering.

Gormes vendors this bridge as a dependency-free JavaScript runtime component
that requires Node.js and npm at pair time only. The bridge does not depend
on Python, pip, or a Hermes checkout at runtime.

See LICENSE for applicable terms from the upstream NousResearch/hermes-agent
repository.
