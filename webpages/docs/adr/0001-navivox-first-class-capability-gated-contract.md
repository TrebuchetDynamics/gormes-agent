# Navivox uses a first-class capability-gated channel contract

Navivox clients use the dedicated `/v1/navivox/*` contract, with `/v1/navivox/capabilities` as the authoritative feature gate, instead of directly consuming dashboard `/api/profiles` or OpenAI-style `/v1/runs` surfaces. This keeps mobile profile management, stream events, auth, attachments, and voice affordances stable and safe for Navivox even though Gormes has other API surfaces for dashboards and OpenAI-compatible automation.
