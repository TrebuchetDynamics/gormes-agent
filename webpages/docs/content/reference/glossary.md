---
title: "Glossary"
description: "Canonical Gormes terms used across the docs."
---

| Term | Meaning |
|---|---|
| Gormes | Go-native AI agent runtime that ports Hermes-style agent workflows into one binary. |
| Hermes | Upstream Python agent whose user-visible behavior is a compatibility reference for Gormes. |
| Hermes compatibility | Gormes matching intentional Hermes behavior where it matters to operators. |
| Hermes parity | Contributor-facing term for formal behavior matching and evidence. |
| Goncho | Gormes' local memory compatibility work; operator docs usually say SQLite memory. |
| Profile | Named Gormes state/config boundary for separating clients, projects, or agents. |
| Workspace | Filesystem location exposed to the model/tool runtime for a profile or invocation. |
| Provider | Model backend such as OpenAI-compatible APIs, OAuth providers, or local endpoints. |
| Model | The model selected within a provider. |
| Gateway | Long-running Gormes process that receives messages from channels such as Telegram, Discord, or Slack. |
| Channel binding | Routing rule that connects a channel/user/profile/agent/workspace intentionally. |
| Session | Persisted conversation and runtime state that can be inspected or resumed. |
| Memory | Durable local context used across sessions. |
| Skill | Reusable agent procedure or capability surfaced through the skills system. |
| Doctor | Readiness check command. `gormes doctor --offline` proves local readiness without network calls. |
| Runtime-ready | Promoted for configured use in the current Go runtime. |
| Planned | Tracked, but not user-ready. |
| Experimental or internal | Useful for contributors; not public setup guidance. |
