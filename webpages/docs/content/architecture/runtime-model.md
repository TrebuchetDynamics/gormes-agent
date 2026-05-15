---
title: "Runtime Model"
description: "The runtime boundary shared by TUI, scripted chat, and gateway paths."
weight: 10
---

# Runtime Model

Gormes exposes several user surfaces, but they should converge on one runtime model:

- CLI flags and config resolve provider/model/runtime settings.
- The TUI and scripted chat paths run local interactive or single-turn sessions.
- Gateway paths adapt channel messages into the shared runtime.
- Tools execute through a registry with channel-neutral results.
- Renderers translate runtime events into terminal or chat UX.

The docs should describe this shared boundary before introducing subsystem internals.
