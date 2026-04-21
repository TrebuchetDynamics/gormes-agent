# Gormes Repository Instructions

## Gortex-First Workflow

Before any manual file read, grep, glob, symbol hunt, or ad hoc code search, use Gortex first.

- Always begin codebase exploration with Gortex graph queries before opening files directly.
- Prefer Gortex for symbol lookup, file summaries, callers, callees, usages, dependencies, impact analysis, and test targeting.
- Treat even "small" reads as graph-navigation work first: map the symbol and its connections in Gortex before reading source.
- Only fall back to direct file reads/search after Gortex has already been used and the graph result is insufficient.
- When falling back, keep the read scope narrow and target the exact file or symbol identified through Gortex.

This repository is in a Go port phase. Unless the user explicitly changes this policy, follow these rules:

- Do not write or modify any Python code.
- Do not edit legacy Python paths such as `run_agent.py`, `cli.py`, `agent/`, `gateway/`, `hermes_cli/`, `tools/`, `tui_gateway/`, `cron/`, `acp_adapter/`, `tests/`, or any other `.py` files.
- Only edit the root `README.md` and files under `gormes/`.
- Treat the existing Python codebase as upstream reference only.
- If a requested change requires touching Python, stop and ask the user whether they want to override this rule.
