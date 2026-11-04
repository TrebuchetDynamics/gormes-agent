+++
schema = 1
id = "TERMINAL-REGISTRY"
title = "Terminal background process registry"
short = "Terminal background registry"
status = "open"
goal = "PHASE-5"
section = "Next"
rank = 30
depends_on = []
touches = ["internal/tools/processregistry/", "internal/tools/terminal_tool.go", "internal/tools/terminal_tool_test.go", "webpages/docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md", "webpages/docs/content/building-gormes/architecture_plan/progress.json"]
resources = []
kind = "implementation"
ownership = "worker"
scope = "Add a bounded in-process registry for background terminal executions. Start with terminal background=true, return a stable process id, list/poll running status and bounded output, terminate by id, and clean up on session end. Reuse the terminal tool's approval and command-safety path; no PTY, WSL-specific, or remote process ownership behavior in this slice."
acceptance = ["TestTerminalToolBackgroundStartsAndLists proves background=true starts a process, returns a stable id, and list shows it running.", "TestProcessRegistryPollsBoundedOutput proves polling returns bounded, redacted output and a terminal status after exit.", "TestProcessRegistryTerminate proves terminate stops the process and a second terminate/list is safe.", "TestProcessRegistryUnknownID proves unknown ids return not_found without mutation.", "TestProcessRegistryCleanup proves session-end cleanup terminates and removes owned handles."]
checks = ["go test ./internal/tools/processregistry -count=1", "go test ./internal/tools -run 'Background|ProcessRegistry' -count=1", "go run ./cmd/progress validate"]
exclusions = ["Background execution bypasses the existing approval or dangerous-command checks.", "The registry stores unbounded output or leaks process handles after session end.", "PTY, WSL process ownership, or remote process behavior is mixed into this slice."]
+++
- Sources: webpages/docs/content/building-gormes/architecture_plan/completion-plan.md#current-finish-ledger
