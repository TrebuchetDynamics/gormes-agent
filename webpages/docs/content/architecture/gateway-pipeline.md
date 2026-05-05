---
title: "Gateway Pipeline"
description: "How messaging channels enter the shared manager and render visible replies."
weight: 20
---

# Gateway Pipeline

Gateway adapters normalize channel-specific events into shared inbound events. The manager routes those events through command handling, session selection, provider/tool execution, and channel rendering.

Visible gateway behavior is part of parity. Formatting bugs, leaked tool-call text, duplicated progress, or raw internal errors are not cosmetic when the channel is the user interface.

Document live gateway support from runtime commands and channel smoke tests, not only from progress rows.
