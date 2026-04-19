# Gormes Frontend Adapter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a drop-in `gormes` CLI that gives Hermes users a Go-native Bubble Tea dashboard over the existing Python `tui_gateway` JSON-RPC backend, while preserving CLI muscle memory and introducing a Hugo docs mirror at `docs.gormes.io`.

**Architecture:** Go owns the CLI facade, dashboard, backend process launcher, JSON-RPC client, telemetry rendering, and Hugo site scaffold. Python remains the Phase 1 supplier for agent orchestration, sessions, tools, and persistence, with one narrow seam extension in `tui_gateway/server.py` to emit `telemetry.update`. Commands not yet native in Go proxy to the existing Python `hermes` CLI unchanged.

**Tech Stack:** Go 1.22, `spf13/cobra`, Bubble Tea/Bubbles/Lipgloss, stdio JSON-RPC, Python `tui_gateway`, Hugo Book, Goldmark-compatible markdown lint, Go `testing`, pytest via `scripts/run_tests.sh`.

---

## File Structure

**Go CLI + transport**

- Create: `gormes/go.mod`
- Create: `gormes/cmd/gormes/main.go`
- Create: `gormes/internal/cli/root.go`
- Create: `gormes/internal/cli/chat.go`
- Create: `gormes/internal/cli/proxy.go`
- Create: `gormes/internal/cli/parity.go`
- Create: `gormes/internal/backend/process.go`
- Create: `gormes/internal/backend/jsonrpc.go`
- Create: `gormes/internal/backend/types.go`
- Create: `gormes/internal/backend/client_test.go`
- Create: `gormes/internal/cli/root_test.go`

**Go TUI**

- Create: `gormes/internal/tui/model.go`
- Create: `gormes/internal/tui/update.go`
- Create: `gormes/internal/tui/view.go`
- Create: `gormes/internal/tui/soul.go`
- Create: `gormes/internal/tui/model_test.go`

**Parity fixtures**

- Create: `gormes/hack/export_surfaces.py`
- Create: `gormes/testdata/cli_surface.json`
- Create: `gormes/testdata/docs_surface.json`

**Python seam**

- Modify: `tui_gateway/server.py`
- Create: `tests/tui_gateway/test_telemetry_update.py`

**Docs**

- Create: `gormes/docs/hugo.toml`
- Create: `gormes/docs/themes/` or Hugo module config in `hugo.toml`
- Create: `gormes/docs/content/_index.md`
- Create: `gormes/docs/content/getting-started/_index.md`
- Create: `gormes/docs/content/user-guide/_index.md`
- Create: `gormes/docs/content/user-guide/features/_index.md`
- Create: `gormes/docs/content/user-guide/messaging/_index.md`
- Create: `gormes/docs/content/user-guide/skills/_index.md`
- Create: `gormes/docs/content/guides/_index.md`
- Create: `gormes/docs/content/integrations/_index.md`
- Create: `gormes/docs/content/reference/_index.md`
- Create: `gormes/docs/content/developer-guide/_index.md`
- Create: `gormes/docs/content/reference/cli-commands.md`
- Create: `gormes/docs/content/developer-guide/architecture.md`
- Create: `gormes/docs/content/getting-started/installation.md`
- Create: `gormes/docs/content/guides/migrate-from-hermes.md`
- Create: `gormes/docs/docs_test.go`

**Scope discipline**

- Do not edit shared agent internals beyond `tui_gateway/server.py` in this phase.
- Do not touch unrelated Python files while another agent is active.
- Keep all new Go code inside `gormes/`.

---

### Task 1: Export Parity Fixtures and Bootstrap the Go Module

**Files:**
- Create: `gormes/go.mod`
- Create: `gormes/hack/export_surfaces.py`
- Create: `gormes/testdata/cli_surface.json`
- Create: `gormes/testdata/docs_surface.json`
- Test: `gormes/internal/cli/root_test.go`

- [ ] **Step 1: Write the failing test**

Create `gormes/internal/cli/root_test.go`:

```go
package cli

import "testing"

func TestLoadParityFixtures(t *testing.T) {
	cli, err := LoadCLISurface("../../testdata/cli_surface.json")
	if err != nil {
		t.Fatalf("load cli surface: %v", err)
	}
	if len(cli.Commands) == 0 {
		t.Fatal("expected at least one command in cli surface fixture")
	}

	docs, err := LoadDocsSurface("../../testdata/docs_surface.json")
	if err != nil {
		t.Fatalf("load docs surface: %v", err)
	}
	if len(docs.Paths) == 0 {
		t.Fatal("expected mirrored docs paths in docs surface fixture")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd gormes && go test ./internal/cli -run TestLoadParityFixtures -v
```

Expected: FAIL with file-not-found or undefined `LoadCLISurface`.

- [ ] **Step 3: Write minimal implementation**

Create `gormes/go.mod`:

```go
module github.com/XelHaku/golang-hermes-agent/gormes

go 1.22

require github.com/spf13/cobra v1.9.1
```

Create `gormes/hack/export_surfaces.py`:

```python
#!/usr/bin/env python3
import json
import pathlib
import re

ROOT = pathlib.Path(__file__).resolve().parents[2]
MAIN = ROOT / "hermes_cli" / "main.py"
DOCS = ROOT / "website" / "docs"
OUT = ROOT / "gormes" / "testdata"

cmd_pattern = re.compile(r'subparsers\\.add_parser\\("([^"]+)"')

commands = []
for line in MAIN.read_text(encoding="utf-8").splitlines():
    m = cmd_pattern.search(line)
    if m:
        commands.append(m.group(1))

paths = []
for path in DOCS.rglob("*.md"):
    rel = path.relative_to(DOCS).as_posix()
    paths.append(rel.removesuffix(".md"))

OUT.mkdir(parents=True, exist_ok=True)
(OUT / "cli_surface.json").write_text(json.dumps({"commands": sorted(set(commands))}, indent=2))
(OUT / "docs_surface.json").write_text(json.dumps({"paths": sorted(set(paths))}, indent=2))
```

Create `gormes/internal/cli/parity.go`:

```go
package cli

import (
	"encoding/json"
	"os"
)

type CLISurface struct {
	Commands []string `json:"commands"`
}

type DocsSurface struct {
	Paths []string `json:"paths"`
}

func LoadCLISurface(path string) (CLISurface, error) {
	var out CLISurface
	b, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(b, &out)
	return out, err
}

func LoadDocsSurface(path string) (DocsSurface, error) {
	var out DocsSurface
	b, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(b, &out)
	return out, err
}
```

Generate fixtures:

```bash
python3 gormes/hack/export_surfaces.py
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
cd gormes && go test ./internal/cli -run TestLoadParityFixtures -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add gormes/go.mod gormes/hack/export_surfaces.py gormes/testdata/cli_surface.json gormes/testdata/docs_surface.json gormes/internal/cli/parity.go gormes/internal/cli/root_test.go
git commit -m "feat(gormes): add parity fixtures and module scaffold"
```

### Task 2: Build the Cobra Root Command and Python Pass-Through Router

**Files:**
- Create: `gormes/cmd/gormes/main.go`
- Create: `gormes/internal/cli/root.go`
- Create: `gormes/internal/cli/chat.go`
- Create: `gormes/internal/cli/proxy.go`
- Modify: `gormes/internal/cli/root_test.go`

- [ ] **Step 1: Write the failing test**

Extend `gormes/internal/cli/root_test.go`:

```go
func TestRouteMode(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want RouteMode
	}{
		{"bare interactive", []string{}, RouteNativeDashboard},
		{"chat interactive", []string{"chat"}, RouteNativeDashboard},
		{"chat one shot", []string{"chat", "-q", "hello"}, RoutePythonProxy},
		{"gateway", []string{"gateway", "status"}, RoutePythonProxy},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetermineRoute(tt.args, true); got != tt.want {
				t.Fatalf("DetermineRoute(%q) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd gormes && go test ./internal/cli -run TestRouteMode -v
```

Expected: FAIL with undefined `RouteMode` or `DetermineRoute`.

- [ ] **Step 3: Write minimal implementation**

Create `gormes/internal/cli/root.go`:

```go
package cli

import "github.com/spf13/cobra"

type RouteMode int

const (
	RouteNativeDashboard RouteMode = iota
	RoutePythonProxy
)

func DetermineRoute(args []string, stdinIsTTY bool) RouteMode {
	if len(args) == 0 {
		return RouteNativeDashboard
	}
	if args[0] == "chat" && stdinIsTTY {
		for i, arg := range args {
			if arg == "-q" || arg == "--query" || arg == "-Q" || arg == "--quiet" {
				return RoutePythonProxy
			}
			if arg == "--query" && i+1 < len(args) {
				return RoutePythonProxy
			}
		}
		return RouteNativeDashboard
	}
	return RoutePythonProxy
}

func NewRootCommand(runNative func() error, runProxy func([]string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "gormes",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if DetermineRoute(args, true) == RouteNativeDashboard {
				return runNative()
			}
			return runProxy(args)
		},
	}
	cmd.AddCommand(&cobra.Command{Use: "chat", RunE: cmd.RunE})
	cmd.AddCommand(&cobra.Command{Use: "gateway", RunE: func(cmd *cobra.Command, args []string) error {
		return runProxy(append([]string{"gateway"}, args...))
	}})
	return cmd
}
```

Create `gormes/internal/cli/proxy.go`:

```go
package cli

import (
	"os"
	"os/exec"
)

func RunPythonProxy(args []string) error {
	c := exec.Command("hermes", args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
```

Create `gormes/cmd/gormes/main.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/XelHaku/golang-hermes-agent/gormes/internal/cli"
)

func main() {
	cmd := cli.NewRootCommand(
		func() error { return fmt.Errorf("native dashboard not wired yet") },
		cli.RunPythonProxy,
	)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
cd gormes && go test ./internal/cli -run TestRouteMode -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add gormes/cmd/gormes/main.go gormes/internal/cli/root.go gormes/internal/cli/proxy.go gormes/internal/cli/root_test.go
git commit -m "feat(gormes): add cobra root and proxy router"
```

### Task 3: Launch the Python Backend and Implement the JSON-RPC Client

**Files:**
- Create: `gormes/internal/backend/process.go`
- Create: `gormes/internal/backend/jsonrpc.go`
- Create: `gormes/internal/backend/types.go`
- Create: `gormes/internal/backend/client_test.go`
- Modify: `gormes/internal/cli/chat.go`

- [ ] **Step 1: Write the failing test**

Create `gormes/internal/backend/client_test.go`:

```go
package backend

import (
	"context"
	"testing"
	"time"
)

func TestClientWaitsForGatewayReady(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client := NewStubClient([]string{
		`{"jsonrpc":"2.0","method":"event","params":{"type":"gateway.ready","payload":{"skin":"default"}}}`,
	})

	if err := client.WaitReady(ctx); err != nil {
		t.Fatalf("WaitReady() error = %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd gormes && go test ./internal/backend -run TestClientWaitsForGatewayReady -v
```

Expected: FAIL with undefined `NewStubClient`.

- [ ] **Step 3: Write minimal implementation**

Create `gormes/internal/backend/types.go`:

```go
package backend

type GatewayEvent struct {
	Type      string         `json:"type"`
	SessionID string         `json:"session_id,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
}
```

Create `gormes/internal/backend/jsonrpc.go`:

```go
package backend

import (
	"context"
	"encoding/json"
	"errors"
)

type Client struct {
	events chan GatewayEvent
}

func NewStubClient(lines []string) *Client {
	c := &Client{events: make(chan GatewayEvent, len(lines))}
	for _, line := range lines {
		var msg struct {
			Method string       `json:"method"`
			Params GatewayEvent `json:"params"`
		}
		_ = json.Unmarshal([]byte(line), &msg)
		c.events <- msg.Params
	}
	close(c.events)
	return c
}

func (c *Client) WaitReady(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-c.events:
			if !ok {
				return errors.New("backend exited before gateway.ready")
			}
			if ev.Type == "gateway.ready" {
				return nil
			}
		}
	}
}
```

Create `gormes/internal/backend/process.go`:

```go
package backend

import (
	"os"
	"os/exec"
)

func StartPythonBackend() (*exec.Cmd, error) {
	cmd := exec.Command("python3", "-m", "tui_gateway.entry")
	cmd.Stdin = os.Stdin
	return cmd, nil
}
```

Create `gormes/internal/cli/chat.go`:

```go
package cli

import (
	"context"
	"time"

	"github.com/XelHaku/golang-hermes-agent/gormes/internal/backend"
)

func RunNativeChat() error {
	client := backend.NewStubClient([]string{
		`{"jsonrpc":"2.0","method":"event","params":{"type":"gateway.ready","payload":{"skin":"default"}}}`,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return client.WaitReady(ctx)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
cd gormes && go test ./internal/backend -run TestClientWaitsForGatewayReady -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add gormes/internal/backend/types.go gormes/internal/backend/jsonrpc.go gormes/internal/backend/process.go gormes/internal/backend/client_test.go gormes/internal/cli/chat.go
git commit -m "feat(gormes): add backend launcher and json-rpc client skeleton"
```

### Task 4: Render the Bubble Tea Dashboard and Stream Backend Events

**Files:**
- Create: `gormes/internal/tui/model.go`
- Create: `gormes/internal/tui/update.go`
- Create: `gormes/internal/tui/view.go`
- Create: `gormes/internal/tui/soul.go`
- Create: `gormes/internal/tui/model_test.go`
- Modify: `gormes/internal/cli/chat.go`

- [ ] **Step 1: Write the failing test**

Create `gormes/internal/tui/model_test.go`:

```go
package tui

import "testing"

func TestApplyMessageDeltaAppendsDraft(t *testing.T) {
	m := InitialModel()
	m.ApplyGatewayEvent("message.delta", map[string]any{"text": "hel"})
	m.ApplyGatewayEvent("message.delta", map[string]any{"text": "lo"})
	if m.Draft != "hello" {
		t.Fatalf("Draft = %q, want hello", m.Draft)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd gormes && go test ./internal/tui -run TestApplyMessageDeltaAppendsDraft -v
```

Expected: FAIL with undefined `InitialModel`.

- [ ] **Step 3: Write minimal implementation**

Create `gormes/internal/tui/model.go`:

```go
package tui

type SoulEntry struct {
	Text string
}

type Model struct {
	Draft string
	Soul  []SoulEntry
}

func InitialModel() Model {
	return Model{Soul: make([]SoulEntry, 0, 10)}
}
```

Create `gormes/internal/tui/update.go`:

```go
package tui

func (m *Model) ApplyGatewayEvent(kind string, payload map[string]any) {
	switch kind {
	case "message.delta":
		if text, ok := payload["text"].(string); ok {
			m.Draft += text
		}
	case "thinking.delta", "reasoning.delta", "status.update", "tool.start", "tool.complete", "telemetry.update":
		if text, ok := payload["text"].(string); ok && text != "" {
			m.pushSoul(text)
		}
	}
}

func (m *Model) pushSoul(text string) {
	m.Soul = append(m.Soul, SoulEntry{Text: text})
	if len(m.Soul) > 10 {
		m.Soul = m.Soul[len(m.Soul)-10:]
	}
}
```

Create `gormes/internal/tui/view.go`:

```go
package tui

func (m Model) View() string {
	return "Draft:\n" + m.Draft
}
```

Create `gormes/internal/tui/soul.go`:

```go
package tui

func SoulLabel(kind string) string {
	switch kind {
	case "tool.start":
		return "tool"
	case "reasoning.delta":
		return "reasoning"
	case "telemetry.update":
		return "telemetry"
	default:
		return "status"
	}
}
```

Modify `gormes/internal/cli/chat.go`:

```go
package cli

import "github.com/XelHaku/golang-hermes-agent/gormes/internal/tui"

func RunNativeChat() error {
	_ = tui.InitialModel()
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
cd gormes && go test ./internal/tui -run TestApplyMessageDeltaAppendsDraft -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add gormes/internal/tui/model.go gormes/internal/tui/update.go gormes/internal/tui/view.go gormes/internal/tui/soul.go gormes/internal/tui/model_test.go gormes/internal/cli/chat.go
git commit -m "feat(gormes): add bubble tea dashboard state skeleton"
```

### Task 5: Extend `tui_gateway` to Emit `telemetry.update`

**Files:**
- Modify: `tui_gateway/server.py`
- Create: `tests/tui_gateway/test_telemetry_update.py`

- [ ] **Step 1: Write the failing test**

Create `tests/tui_gateway/test_telemetry_update.py`:

```python
from tui_gateway import server


def test_emit_telemetry_update(monkeypatch):
    seen = []

    monkeypatch.setattr(server, "_emit", lambda event, sid, payload=None: seen.append((event, sid, payload)))

    class Agent:
        model = "hermes-agent"
        session_input_tokens = 12
        session_output_tokens = 3
        session_total_tokens = 15
        session_api_calls = 1
        context_compressor = None

    server._emit_telemetry("sid-1", Agent())

    assert seen == [
        (
            "telemetry.update",
            "sid-1",
            {
                "model": "hermes-agent",
                "input": 12,
                "output": 3,
                "total": 15,
                "calls": 1,
            },
        )
    ]
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
source venv/bin/activate && scripts/run_tests.sh tests/tui_gateway/test_telemetry_update.py -q
```

Expected: FAIL with missing `_emit_telemetry`.

- [ ] **Step 3: Write minimal implementation**

Modify `tui_gateway/server.py` near `_get_usage`:

```python
def _emit_telemetry(sid: str, agent) -> None:
    usage = _get_usage(agent)
    payload = {
        "model": usage.get("model", "") or "",
        "input": int(usage.get("input", 0) or 0),
        "output": int(usage.get("output", 0) or 0),
        "total": int(usage.get("total", 0) or 0),
        "calls": int(usage.get("calls", 0) or 0),
    }
    if "context_used" in usage:
        payload["context_used"] = int(usage["context_used"] or 0)
    if "context_max" in usage:
        payload["context_max"] = int(usage["context_max"] or 0)
    if "context_percent" in usage:
        payload["context_percent"] = int(usage["context_percent"] or 0)
    _emit("telemetry.update", sid, payload)
```

And inside `prompt.submit`:

```python
            _emit_telemetry(sid, agent)

            def _stream(delta):
                payload = {"text": delta}
                if streamer and (r := streamer.feed(delta)) is not None:
                    payload["rendered"] = r
                _emit("message.delta", sid, payload)
                _emit_telemetry(sid, agent)
```

And before final `message.complete`:

```python
            _emit_telemetry(sid, agent)
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
source venv/bin/activate && scripts/run_tests.sh tests/tui_gateway/test_telemetry_update.py -q
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add tui_gateway/server.py tests/tui_gateway/test_telemetry_update.py
git commit -m "feat(tui-gateway): emit telemetry update events for gormes"
```

### Task 6: Enforce CLI Parity and Proxy Behavior with Tests

**Files:**
- Modify: `gormes/internal/cli/root_test.go`
- Modify: `gormes/internal/cli/root.go`
- Modify: `gormes/internal/cli/proxy.go`

- [ ] **Step 1: Write the failing test**

Append to `gormes/internal/cli/root_test.go`:

```go
func TestAllTopLevelFixtureCommandsAreRegistered(t *testing.T) {
	fixture, err := LoadCLISurface("../../testdata/cli_surface.json")
	if err != nil {
		t.Fatalf("LoadCLISurface: %v", err)
	}
	registered := map[string]bool{}
	root := NewRootCommand(func() error { return nil }, func([]string) error { return nil })
	for _, cmd := range root.Commands() {
		registered[cmd.Name()] = true
	}
	for _, want := range fixture.Commands {
		if !registered[want] && want != "login" && want != "logout" {
			t.Fatalf("missing top-level command %q", want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd gormes && go test ./internal/cli -run TestAllTopLevelFixtureCommandsAreRegistered -v
```

Expected: FAIL listing missing commands.

- [ ] **Step 3: Write minimal implementation**

Expand `gormes/internal/cli/root.go` with explicit command registration:

```go
var topLevel = []string{
	"chat", "model", "gateway", "setup", "whatsapp", "auth", "login", "logout",
	"status", "cron", "webhook", "doctor", "dump", "debug", "backup", "import",
	"config", "pairing", "skills", "plugins", "memory", "tools", "mcp",
	"sessions", "insights", "claw", "version", "update", "uninstall", "acp",
	"profile", "completion", "dashboard", "logs",
}

func addProxyCommands(root *cobra.Command, runProxy func([]string) error) {
	for _, name := range topLevel {
		if name == "chat" {
			continue
		}
		commandName := name
		root.AddCommand(&cobra.Command{
			Use: commandName,
			RunE: func(cmd *cobra.Command, args []string) error {
				return runProxy(append([]string{commandName}, args...))
			},
		})
	}
}
```

And call `addProxyCommands` from `NewRootCommand`.

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
cd gormes && go test ./internal/cli -run TestAllTopLevelFixtureCommandsAreRegistered -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add gormes/internal/cli/root.go gormes/internal/cli/root_test.go
git commit -m "feat(gormes): register full top-level parity command tree"
```

### Task 7: Scaffold the Hugo Site and Mirror the Hermes Docs IA

**Files:**
- Create: `gormes/docs/hugo.toml`
- Create: `gormes/docs/content/_index.md`
- Create: `gormes/docs/content/getting-started/_index.md`
- Create: `gormes/docs/content/user-guide/_index.md`
- Create: `gormes/docs/content/user-guide/features/_index.md`
- Create: `gormes/docs/content/user-guide/messaging/_index.md`
- Create: `gormes/docs/content/user-guide/skills/_index.md`
- Create: `gormes/docs/content/guides/_index.md`
- Create: `gormes/docs/content/integrations/_index.md`
- Create: `gormes/docs/content/reference/_index.md`
- Create: `gormes/docs/content/developer-guide/_index.md`
- Create: `gormes/docs/content/reference/cli-commands.md`
- Create: `gormes/docs/content/getting-started/installation.md`
- Create: `gormes/docs/content/developer-guide/architecture.md`
- Create: `gormes/docs/content/guides/migrate-from-hermes.md`

- [ ] **Step 1: Write the failing test**

Create `gormes/docs/docs_test.go`:

```go
package docs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMirroredSectionRootsExist(t *testing.T) {
	root := "."
	required := []string{
		"content/getting-started",
		"content/user-guide",
		"content/user-guide/features",
		"content/user-guide/messaging",
		"content/user-guide/skills",
		"content/guides",
		"content/integrations",
		"content/reference",
		"content/developer-guide",
	}
	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("missing docs section %s: %v", rel, err)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd gormes && go test ./docs -run TestMirroredSectionRootsExist -v
```

Expected: FAIL because the Hugo content tree does not exist yet.

- [ ] **Step 3: Write minimal implementation**

Create `gormes/docs/hugo.toml`:

```toml
baseURL = "https://docs.gormes.io/"
title = "Gormes Documentation"
languageCode = "en-us"
theme = "hugo-book"

[markup.goldmark.renderer]
unsafe = false
```

Create `gormes/docs/content/_index.md`:

```md
+++
title = "Gormes Documentation"
+++

# Gormes Documentation

This site mirrors the Hermes documentation structure so existing Hermes users can navigate by muscle memory.
```

Create `gormes/docs/content/reference/cli-commands.md`:

```md
+++
title = "CLI Commands Reference"
+++

# CLI Commands Reference

`gormes` preserves the Hermes command tree. Native dashboard paths are implemented in Go; non-native commands proxy to Python Hermes unchanged in Phase 1.
```

Create section index files for each required directory with minimal headers:

```md
+++
title = "Getting Started"
+++
```

Repeat with appropriate titles for `user-guide`, `guides`, `integrations`, `reference`, and `developer-guide`.

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
cd gormes && go test ./docs -run TestMirroredSectionRootsExist -v
cd gormes/docs && hugo --panicOnWarning
```

Expected: PASS, then Hugo builds successfully.

- [ ] **Step 5: Commit**

```bash
git add gormes/docs/hugo.toml gormes/docs/content gormes/docs/docs_test.go
git commit -m "feat(gormes-docs): scaffold hugo site and mirrored section tree"
```

### Task 8: Add Goldmark Lint and Docs/CLI Migration Guardrails

**Files:**
- Modify: `gormes/docs/docs_test.go`
- Modify: `gormes/docs/content/reference/cli-commands.md`
- Modify: `gormes/docs/content/developer-guide/architecture.md`
- Modify: `gormes/docs/content/guides/migrate-from-hermes.md`

- [ ] **Step 1: Write the failing test**

Append to `gormes/docs/docs_test.go`:

```go
func TestGoldmarkUnsafePatternsAreRejected(t *testing.T) {
	badPatterns := []string{"<div style=", "{/*", ":::warning", "className=", "/docs/"}
	checkFile := func(path string) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(data)
		for _, bad := range badPatterns {
			if strings.Contains(text, bad) {
				t.Fatalf("%s contains unsupported pattern %q", path, bad)
			}
		}
	}
	checkFile("content/reference/cli-commands.md")
	checkFile("content/developer-guide/architecture.md")
	checkFile("content/guides/migrate-from-hermes.md")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd gormes && go test ./docs -run TestGoldmarkUnsafePatternsAreRejected -v
```

Expected: FAIL until the files are written without Docusaurus/MDX patterns and `strings` is imported.

- [ ] **Step 3: Write minimal implementation**

Update `gormes/docs/docs_test.go` imports:

```go
import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)
```

Update `gormes/docs/content/developer-guide/architecture.md`:

```md
+++
title = "Architecture"
+++

# Architecture

Phase 1 uses a Go dashboard over the Python `tui_gateway` backend. Go owns the cabin; Python still owns the brain, memory, and tools.
```

Update `gormes/docs/content/guides/migrate-from-hermes.md`:

```md
+++
title = "Migrate from Hermes"
+++

# Migrate from Hermes

Swap `hermes` for `gormes` when you want the Go dashboard. Commands not yet native in Go are forwarded to Python Hermes so your existing workflow keeps running.
```

Update `gormes/docs/content/reference/cli-commands.md` to avoid Docusaurus syntax and root-relative links:

```md
+++
title = "CLI Commands Reference"
+++

# CLI Commands Reference

## Global entrypoint

```bash
gormes [global-options] <command> [subcommand/options]
```

See `../guides/migrate-from-hermes.md` for the Phase 1 proxy model.
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
cd gormes && go test ./docs -v
cd gormes/docs && hugo --panicOnWarning
```

Expected: PASS, then Hugo builds with no warnings.

- [ ] **Step 5: Commit**

```bash
git add gormes/docs/docs_test.go gormes/docs/content/reference/cli-commands.md gormes/docs/content/developer-guide/architecture.md gormes/docs/content/guides/migrate-from-hermes.md
git commit -m "feat(gormes-docs): add goldmark lint and migration guardrails"
```

---

## Self-Review

### Spec coverage

- JSON-RPC dashboard seam: Tasks 2, 3, 4
- CLI parity and pass-through: Tasks 1, 2, 6
- Python telemetry seam: Task 5
- Hugo scaffolding: Task 7
- Goldmark-safe markdown: Task 8
- Docs IA mirror: Tasks 1, 7, 8

### Placeholder scan

- No deferred-work markers remain in the plan body.
- The only conditional choice is the Hugo theme fallback in the spec; the plan fixes the implementation choice to `hugo-book`.

### Type consistency

- `LoadCLISurface`, `LoadDocsSurface`, `DetermineRoute`, `RunNativeChat`, and `_emit_telemetry` use the same names across tasks.
- `telemetry.update` is the canonical event name in both spec and plan.
