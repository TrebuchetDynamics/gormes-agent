# Runtime Channels Specs

Owns tool-registry, gateway, messaging-channel, cron, subagent, skills, and thin persistence design decisions.

Exposes dated design specs for runtime surfaces that connect Gormes to operators and tools.

Must never know about web presentation internals, installer orchestration, or memory graph implementation except through stable runtime contracts.
