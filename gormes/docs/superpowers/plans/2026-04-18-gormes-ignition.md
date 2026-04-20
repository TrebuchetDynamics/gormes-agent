# Gormes Ignition (M0 + M1) Implementation Plan

> **⚠️ DISCARDED — 2026-04-18**
>
> This plan implemented the superseded `2026-04-18-gormes-ignition-design.md`, which greenfield-rebuilt an OpenRouter client, SQLite schema, session layer, and prompt builder. Post-recon, all of that was found to already exist in the Python codebase and reachable via an OpenAI-compatible HTTP+SSE server. A new plan will be produced from `2026-04-18-gormes-frontend-adapter-design.md`.
>
> Do **not** execute the tasks below.

---

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a Go binary at `gormes/cmd/gormes` that boots a Bubble Tea Debug/Dashboard TUI, streams one LLM turn against OpenRouter, and persists the conversation in SQLite — the vertical slice described in spec `docs/superpowers/specs/2026-04-18-gormes-ignition-design.md`.

**Architecture:** One Go process, three actors communicating over channels (TUI ↔ Agent ↔ Provider). CGO-free. All code under `gormes/` — no upstream Python files are modified. `internal/pybridge` is reserved as an interface stub for the M4 Python-subprocess bridge.

**Tech Stack:** Go 1.22+, `charmbracelet/bubbletea`, `charmbracelet/bubbles`, `charmbracelet/lipgloss`, `modernc.org/sqlite`, `pelletier/go-toml/v2`, `spf13/pflag`, stdlib `log/slog` and `net/http`.

---

## Prerequisites

- Go 1.22 or newer (`go version`)
- `OPENROUTER_API_KEY` set for live integration tests (tests skip cleanly without it)
- Working directory: repository root. All new files land under `gormes/`
- No worktree required — the blast radius is entirely new files under `gormes/`

## File Structure Map

```
gormes/
├── cmd/gormes/main.go                    # entry point
├── internal/
│   ├── agent/
│   │   ├── agent.go
│   │   ├── default_prompt.go
│   │   └── agent_test.go
│   ├── tui/
│   │   ├── model.go
│   │   ├── view.go
│   │   ├── update.go
│   │   └── tui_test.go
│   ├── provider/
│   │   ├── provider.go
│   │   ├── errors.go
│   │   ├── openrouter.go
│   │   ├── openrouter_test.go
│   │   ├── mock.go
│   │   └── mock_test.go
│   ├── session/
│   │   ├── session.go
│   │   └── session_test.go
│   ├── db/
│   │   ├── db.go
│   │   ├── queries.go
│   │   ├── migrations/0001_initial.sql
│   │   └── db_test.go
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go
│   ├── telemetry/
│   │   ├── telemetry.go
│   │   └── telemetry_test.go
│   └── pybridge/
│       └── pybridge.go
├── pkg/gormes/types.go
├── docs/
│   ├── ARCH_PLAN.md
│   └── docs_test.go
├── go.mod
├── go.sum
├── .gitignore
├── README.md
└── Makefile
```

---

## Task 1: Bootstrap the Go Module

**Files:**
- Create: `gormes/go.mod`
- Create: `gormes/.gitignore`
- Create: `gormes/Makefile`
- Create: `gormes/README.md`

- [ ] **Step 1:** Create the `gormes/` directory and initialize the module.

```bash
cd gormes
go mod init github.com/TrebuchetDynamics/gormes-agent/gormes
```

Expected: `go.mod` is created with `module github.com/TrebuchetDynamics/gormes-agent/gormes` and `go 1.22` directive. Run `cat go.mod` to verify.

- [ ] **Step 2:** Edit `gormes/go.mod` to pin the Go version explicitly (if `go mod init` picked a different one).

```
module github.com/TrebuchetDynamics/gormes-agent/gormes

go 1.22
```

- [ ] **Step 3:** Create `gormes/.gitignore`:

```
bin/
*.test
*.out
coverage.out
crash-*.log
.gormes/
```

- [ ] **Step 4:** Create `gormes/Makefile`:

```makefile
.PHONY: build test test-live lint fmt clean

build:
	go build -o bin/gormes ./cmd/gormes

test:
	go test ./...

test-live:
	go test -tags=live ./...

lint:
	golangci-lint run

fmt:
	gofmt -w .
	goimports -w .

clean:
	rm -rf bin/ coverage.out
```

- [ ] **Step 5:** Create `gormes/README.md` (skeleton — the "Rosetta Stone" expansion lives in ARCH_PLAN.md):

```markdown
# Gormes

Go port of [Hermes Agent](../README.md) (the Python reference implementation in this repo's root).

**Status:** M0 + M1 "Ignition" — in progress.

## Install

```bash
cd gormes
make build
./bin/gormes
```

Requires Go 1.22+ and `OPENROUTER_API_KEY` in the environment.

## Architecture

See [`docs/ARCH_PLAN.md`](docs/ARCH_PLAN.md) for the executive roadmap and the "Motherboard" strategy (Go chassis + Python peripheral library).

## Relationship to the Python implementation

Neither replaces the other. The repository root is the **Reference Implementation** (Python, upstream `NousResearch/hermes-agent`). This `gormes/` directory is the **High-Performance Implementation** (Go). See `docs/ARCH_PLAN.md` for the full rationale.

## License

MIT — see `../LICENSE`.
```

- [ ] **Step 6:** Commit.

```bash
cd ..
git add gormes/go.mod gormes/.gitignore gormes/Makefile gormes/README.md
git commit -m "feat(gormes): bootstrap Go module skeleton

Adds go.mod (github.com/TrebuchetDynamics/gormes-agent/gormes, go 1.22),
Makefile targets, .gitignore, and README pointing at ARCH_PLAN.md."
```

---

## Task 2: Write ARCH_PLAN.md

**Files:**
- Create: `gormes/docs/ARCH_PLAN.md`

- [ ] **Step 1:** Create `gormes/docs/ARCH_PLAN.md` with all six required subsections from spec §21.2. Use exactly this content:

````markdown
# Gormes — Executive Roadmap (ARCH_PLAN)

**Public site:** https://gormes.io
**Source:** https://github.com/TrebuchetDynamics/gormes-agent
**Upstream reference:** https://github.com/NousResearch/hermes-agent

---

## 1. Rosetta Stone Declaration

The repository root is the **Reference Implementation** (Python, upstream `NousResearch/hermes-agent`). The `gormes/` directory is the **High-Performance Implementation** (Go). Neither replaces the other; they co-evolve as a translation pair.

The Python side remains the place where AI research moves fast and where heavyweight library ecosystems live. The Go side provides the always-on, low-footprint orchestrator and the typed interfaces that make the Python side safer to change.

---

## 2. Why Go — for a Python developer

Five concrete bullets, no hype:

1. **Binary portability.** One 15–30 MB static binary. No `uv`, `pip`, venv, or system Python on the target host. `scp`-and-run on a $5 VPS or Termux.
2. **Static types and compile-time contracts.** Tool schemas, Provider envelopes, and MCP payloads become typed structs. Schema drift is a compile error, not a silent agent-loop failure.
3. **True concurrency.** Goroutines over channels replace `asyncio`. The gateway scales to 10+ platform connections without event-loop starvation.
4. **Lower idle footprint.** Target ≈ 10 MB RSS at idle vs. ≈ 80+ MB for Python Hermes. Meaningful on always-on or low-spec hosts.
5. **Explicit trade-off.** The Python AI-library moat (`litellm`, `instructor`, heavyweight ML, research skills) stays in Python. M4's Python Bridge is the seam — not an afterthought.

---

## 3. Hybrid Manifesto — the Motherboard Strategy

Go is the **chassis**: orchestrator, state, persistence, platform I/O, and agent cognition.

Python is the **peripheral library**: research tools, legacy Hermes skills, and ML heavy lifting.

**M4's Python Bridge is the PCIe slot.** M1 ships the interface stub at `internal/pybridge`. Delegating agent cognition itself to subprocess RPC was considered and rejected — every turn would pay IPC latency, and the agent's identity would couple to disk I/O.

---

## 4. Milestone Status

| Milestone | Status | Deliverable |
|---|---|---|
| M0 — Scaffold | 🔨 in progress | Go module, interfaces, migrations, this ARCH_PLAN |
| M1 — TUI + LLM | 🔨 in progress | Bubble Tea Dashboard streaming OpenRouter turn |
| M2 — Ontological Memory | ⏳ planned | FTS5, fact-triples, semantic recall |
| M3 — Multi-Platform Gateway | ⏳ planned | Telegram / Discord / Slack concurrent adapters |
| M4 — Python Bridge | ⏳ planned | Subprocess RPC; skills and research tools plug in |

Legend: 🔨 in progress · ✅ complete · ⏳ planned · ⏸ deferred.

Status updates are committed changes to this file.

---

## 5. Project Boundaries

Hard rule: no Python file in this repository is modified. All Gormes work lives under `gormes/`. Upstream rebases against `NousResearch/hermes-agent` cannot conflict with Gormes because paths do not overlap.

A one-time "Go Implementation Status" addition to the repository-root `README.md` is explicitly deferred until after M1 ships.

---

## 6. Documentation

This `ARCH_PLAN.md` is the executive roadmap. Per-milestone specs live at `docs/superpowers/specs/YYYY-MM-DD-*.md`. Per-milestone implementation plans live at `docs/superpowers/plans/YYYY-MM-DD-*.md`.

Public-site (`gormes.io`) deployment is M1.5 work. The documentation is authored in CommonMark + GFM so every mainstream static-site generator (MkDocs Material, Hugo, Astro Starlight) can render it without rewrites.
````

- [ ] **Step 2:** Commit.

```bash
git add gormes/docs/ARCH_PLAN.md
git commit -m "docs(gormes): add ARCH_PLAN.md executive roadmap"
```

---

## Task 3: Markdown SSG-Portability Lint

**Files:**
- Create: `gormes/docs/docs_test.go`

- [ ] **Step 1:** Write the failing test. Create `gormes/docs/docs_test.go`:

```go
package docs_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Banned patterns from spec §21.3 avoidance list.
var banned = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{"github-admonition", regexp.MustCompile(`^> \[!(NOTE|WARNING|TIP|IMPORTANT|CAUTION)\]`)},
	{"root-relative-link", regexp.MustCompile(`\]\(/[^)]+\)`)},
	{"emoji-in-heading", regexp.MustCompile(`^#{1,6} .*[\x{1F300}-\x{1FAFF}\x{2600}-\x{27BF}]`)},
	// Raw HTML is allowed only for <br> inside table cells; a blanket block-level tag check:
	{"raw-html-block", regexp.MustCompile(`(?m)^<(div|span|details|summary|p)\b`)},
}

// Files that must pass the lint.
var targets = []string{
	"ARCH_PLAN.md",
	"superpowers/specs/2026-04-18-gormes-ignition-design.md",
	"superpowers/plans/2026-04-18-gormes-ignition.md",
}

func TestMarkdownIsSSGPortable(t *testing.T) {
	for _, rel := range targets {
		t.Run(rel, func(t *testing.T) {
			path := filepath.Join(".", rel)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			content := string(raw)
			for i, line := range strings.Split(content, "\n") {
				for _, b := range banned {
					if b.pattern.MatchString(line) {
						t.Errorf("%s:%d violates %s: %q", path, i+1, b.name, line)
					}
				}
			}
		})
	}
}
```

- [ ] **Step 2:** Run and verify it passes on the already-committed files.

```bash
cd gormes
go test ./docs/...
```

Expected: PASS for `ARCH_PLAN.md` and the existing spec; the plan-file target will not exist yet on its first run, so either skip this test until the plan is committed, or expect one non-fatal file-not-found in the failure list. After this plan is committed (at the end of the plan-writing phase), all three targets exist.

- [ ] **Step 3:** If the test errors on a file-not-found (plan), temporarily remove the plan path from `targets`, re-run, then restore it once the plan is committed. This is a one-time chicken-and-egg.

- [ ] **Step 4:** Commit.

```bash
cd ..
git add gormes/docs/docs_test.go
git commit -m "test(gormes/docs): add SSG-portability lint for ARCH_PLAN and specs"
```

---

## Task 4: Config Package (env + XDG + TOML precedence)

**Files:**
- Create: `gormes/internal/config/config.go`
- Create: `gormes/internal/config/config_test.go`

- [ ] **Step 1:** Add the TOML dependency.

```bash
cd gormes
go get github.com/pelletier/go-toml/v2@latest
go get github.com/spf13/pflag@latest
```

- [ ] **Step 2:** Write the failing test. Create `gormes/internal/config/config_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_BuiltinDefaults(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("GORMES_MODEL", "")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Provider.Name != "openrouter" {
		t.Errorf("default provider = %q, want openrouter", cfg.Provider.Name)
	}
	if cfg.Provider.Model == "" {
		t.Error("default model must be non-empty")
	}
	if cfg.Storage.DBPath == "" {
		t.Error("DBPath must resolve to an XDG default")
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	dir := filepath.Join(cfgHome, "gormes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(file, []byte(`[provider]
name = "openrouter"
model = "file-model"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GORMES_MODEL", "env-model")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Provider.Model != "env-model" {
		t.Errorf("model = %q, want env-model (env beats file)", cfg.Provider.Model)
	}
}

func TestLoad_FlagsOverrideEnv(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("GORMES_MODEL", "env-model")

	cfg, err := Load([]string{"--model", "flag-model"})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Provider.Model != "flag-model" {
		t.Errorf("model = %q, want flag-model (flag beats env)", cfg.Provider.Model)
	}
}
```

- [ ] **Step 3:** Run the test; expect it to fail (package doesn't compile).

```bash
go test ./internal/config/...
```

Expected: FAIL — `Load` undefined.

- [ ] **Step 4:** Implement `gormes/internal/config/config.go`:

```go
// Package config loads Gormes configuration from (in precedence order):
// CLI flags > environment variables > TOML file > built-in defaults.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/pflag"
)

type Config struct {
	Provider ProviderCfg `toml:"provider"`
	Storage  StorageCfg  `toml:"storage"`
	TUI      TUICfg      `toml:"tui"`
	Agent    AgentCfg    `toml:"agent"`
	// Secrets never live in the TOML file; env only.
	OpenRouterAPIKey string `toml:"-"`
}

type ProviderCfg struct {
	Name  string `toml:"name"`
	Model string `toml:"model"`
}

type StorageCfg struct {
	DBPath string `toml:"db_path"`
}

type TUICfg struct {
	Theme string `toml:"theme"`
}

type AgentCfg struct {
	SystemPrompt string `toml:"system_prompt"`
}

// Load resolves configuration. Pass os.Args[1:] (or nil to skip flag parsing).
func Load(args []string) (Config, error) {
	cfg := defaults()

	// File (lowest override over defaults).
	if err := loadFile(&cfg); err != nil {
		return cfg, err
	}
	// Env.
	loadEnv(&cfg)
	// Flags (highest).
	if err := loadFlags(&cfg, args); err != nil {
		return cfg, err
	}
	// Resolve empty paths to XDG defaults.
	if cfg.Storage.DBPath == "" {
		cfg.Storage.DBPath = defaultDBPath()
	}
	return cfg, nil
}

func defaults() Config {
	return Config{
		Provider: ProviderCfg{Name: "openrouter", Model: "anthropic/claude-opus-4-7"},
		TUI:      TUICfg{Theme: "dark"},
	}
}

func loadFile(cfg *Config) error {
	path := filepath.Join(xdgConfigHome(), "gormes", "config.toml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}
	if err := toml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}
	return nil
}

func loadEnv(cfg *Config) {
	if v := os.Getenv("GORMES_MODEL"); v != "" {
		cfg.Provider.Model = v
	}
	if v := os.Getenv("GORMES_DB"); v != "" {
		cfg.Storage.DBPath = v
	}
	if v := os.Getenv("GORMES_SYSTEM_PROMPT"); v != "" {
		cfg.Agent.SystemPrompt = v
	}
	cfg.OpenRouterAPIKey = os.Getenv("OPENROUTER_API_KEY")
}

func loadFlags(cfg *Config, args []string) error {
	if args == nil {
		return nil
	}
	fs := pflag.NewFlagSet("gormes", pflag.ContinueOnError)
	model := fs.String("model", "", "LLM model slug")
	provider := fs.String("provider", "", "provider name")
	db := fs.String("db-path", "", "SQLite DB path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *model != "" {
		cfg.Provider.Model = *model
	}
	if *provider != "" {
		cfg.Provider.Name = *provider
	}
	if *db != "" {
		cfg.Storage.DBPath = *db
	}
	return nil
}

func xdgConfigHome() string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config")
}

func xdgDataHome() string {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share")
}

func defaultDBPath() string {
	return filepath.Join(xdgDataHome(), "gormes", "gormes.db")
}

// CrashLogDir is the directory for TUI crash dumps.
func CrashLogDir() string {
	return filepath.Join(xdgDataHome(), "gormes")
}
```

- [ ] **Step 5:** Run the tests; expect PASS.

```bash
go test ./internal/config/... -v
```

Expected: three tests PASS.

- [ ] **Step 6:** Commit.

```bash
cd ..
git add gormes/internal/config/ gormes/go.mod gormes/go.sum
git commit -m "feat(gormes/config): add config loader with flag>env>toml precedence"
```

---

## Task 5: DB Package with Embedded Migrations

**Files:**
- Create: `gormes/internal/db/migrations/0001_initial.sql`
- Create: `gormes/internal/db/db.go`
- Create: `gormes/internal/db/queries.go`
- Create: `gormes/internal/db/db_test.go`

- [ ] **Step 1:** Add the SQLite driver.

```bash
cd gormes
go get modernc.org/sqlite@latest
```

- [ ] **Step 2:** Create `gormes/internal/db/migrations/0001_initial.sql`:

```sql
CREATE TABLE sessions (
    id          TEXT PRIMARY KEY,
    created_at  INTEGER NOT NULL,
    model       TEXT NOT NULL,
    title       TEXT
);

CREATE TABLE turns (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id  TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    role        TEXT NOT NULL CHECK (role IN ('system','user','assistant')),
    content     TEXT NOT NULL,
    reasoning   TEXT,
    metadata    TEXT,
    tokens_in   INTEGER,
    tokens_out  INTEGER,
    latency_ms  INTEGER,
    status      TEXT NOT NULL DEFAULT 'complete'
                   CHECK (status IN ('complete','cancelled','error')),
    created_at  INTEGER NOT NULL
);

CREATE INDEX idx_turns_session ON turns(session_id, id);

CREATE TABLE schema_version (version INTEGER PRIMARY KEY);
INSERT INTO schema_version VALUES (1);
```

- [ ] **Step 3:** Write the failing test. Create `gormes/internal/db/db_test.go`:

```go
package db

import (
	"path/filepath"
	"testing"
)

func TestOpen_CreatesFileAndRunsMigrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	d, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	var v int
	if err := d.Handle().QueryRow(`SELECT version FROM schema_version`).Scan(&v); err != nil {
		t.Fatalf("schema_version query: %v", err)
	}
	if v != 1 {
		t.Errorf("schema_version = %d, want 1", v)
	}
}

func TestOpen_Idempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	d1, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	d1.Close()

	d2, err := Open(path)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer d2.Close()

	var v int
	if err := d2.Handle().QueryRow(`SELECT version FROM schema_version`).Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v != 1 {
		t.Errorf("schema_version after re-open = %d, want 1", v)
	}
}
```

- [ ] **Step 4:** Run; expect FAIL (no `Open`).

```bash
go test ./internal/db/...
```

- [ ] **Step 5:** Implement `gormes/internal/db/db.go`:

```go
// Package db opens the SQLite store and applies embedded migrations.
package db

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type DB struct {
	sql *sql.DB
}

// Open opens (or creates) the SQLite DB at path and applies any pending migrations.
func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	h, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	d := &DB{sql: h}
	if err := d.migrate(); err != nil {
		h.Close()
		return nil, err
	}
	return d, nil
}

func (d *DB) Handle() *sql.DB { return d.sql }

func (d *DB) Close() error { return d.sql.Close() }

func (d *DB) migrate() error {
	current, err := d.currentVersion()
	if err != nil {
		return err
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		version := parseVersion(name)
		if version <= current {
			continue
		}
		raw, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		tx, err := d.sql.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(raw)); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) currentVersion() (int, error) {
	var v int
	err := d.sql.QueryRow(`SELECT version FROM schema_version`).Scan(&v)
	if err != nil {
		// Table doesn't exist yet on first run; treat as version 0.
		if strings.Contains(err.Error(), "no such table") {
			return 0, nil
		}
		return 0, err
	}
	return v, nil
}

func parseVersion(name string) int {
	// "0001_initial.sql" -> 1
	parts := strings.SplitN(name, "_", 2)
	if len(parts) == 0 {
		return 0
	}
	var v int
	fmt.Sscanf(parts[0], "%d", &v)
	return v
}
```

- [ ] **Step 6:** Create `gormes/internal/db/queries.go` (query constants used by `session`):

```go
package db

// Query constants shared across packages that access the DB.
const (
	QInsertSession = `INSERT INTO sessions(id, created_at, model, title) VALUES (?, ?, ?, ?)`

	QSelectLatestSession = `SELECT id, created_at, model, COALESCE(title,'')
	                       FROM sessions ORDER BY created_at DESC LIMIT 1`

	QInsertTurn = `INSERT INTO turns(session_id, role, content, created_at, status)
	              VALUES (?, ?, ?, ?, 'complete')`

	QUpdateTurnStats = `UPDATE turns
	                   SET tokens_in = ?, tokens_out = ?, latency_ms = ?
	                   WHERE id = ?`

	QUpdateTurnMetadata = `UPDATE turns SET reasoning = ?, metadata = ? WHERE id = ?`

	QMarkTurnStatus = `UPDATE turns SET status = ? WHERE id = ?`

	QSelectHistory = `SELECT role, content FROM turns
	                  WHERE session_id = ? AND status = 'complete'
	                  ORDER BY id ASC LIMIT ?`
)
```

- [ ] **Step 7:** Run tests; expect PASS.

```bash
go test ./internal/db/... -v
```

- [ ] **Step 8:** Commit.

```bash
cd ..
git add gormes/internal/db/ gormes/go.mod gormes/go.sum
git commit -m "feat(gormes/db): SQLite store with embedded migrations"
```

---

## Task 6: Session Package

**Files:**
- Create: `gormes/internal/session/session.go`
- Create: `gormes/internal/session/session_test.go`

- [ ] **Step 1:** Write the failing test. Create `gormes/internal/session/session_test.go`:

```go
package session

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/db"
)

func newTestSession(t *testing.T) *Session {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	s, err := New(d, "test-model")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAppendAndHistory(t *testing.T) {
	s := newTestSession(t)
	if _, err := s.AppendTurn("user", "hi"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendTurn("assistant", "hello"); err != nil {
		t.Fatal(err)
	}
	msgs, err := s.History(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("got %d messages, want 2", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "hi" {
		t.Errorf("msgs[0] = %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "hello" {
		t.Errorf("msgs[1] = %+v", msgs[1])
	}
}

func TestHistorySkipsCancelledAndError(t *testing.T) {
	s := newTestSession(t)
	id1, _ := s.AppendTurn("user", "q1")
	_ = s.MarkTurnStatus(id1, "cancelled")
	_, _ = s.AppendTurn("assistant", "a1")

	msgs, err := s.History(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("got %d, want 1 (cancelled turn skipped)", len(msgs))
	}
	if msgs[0].Content != "a1" {
		t.Errorf("unexpected msg: %+v", msgs[0])
	}
}

func TestUpdateTurnMetadata(t *testing.T) {
	s := newTestSession(t)
	id, _ := s.AppendTurn("assistant", "text")
	if err := s.UpdateTurnMetadata(id, "thinking...", []byte(`{"finish_reason":"stop"}`)); err != nil {
		t.Fatal(err)
	}
	var reasoning, metadata string
	err := s.db.Handle().QueryRow(
		`SELECT COALESCE(reasoning,''), COALESCE(metadata,'') FROM turns WHERE id = ?`, id,
	).Scan(&reasoning, &metadata)
	if err != nil {
		t.Fatal(err)
	}
	if reasoning != "thinking..." {
		t.Errorf("reasoning = %q", reasoning)
	}
	if metadata != `{"finish_reason":"stop"}` {
		t.Errorf("metadata = %q", metadata)
	}
}
```

- [ ] **Step 2:** Run; expect FAIL.

```bash
cd gormes
go test ./internal/session/...
```

- [ ] **Step 3:** Implement `gormes/internal/session/session.go`:

```go
// Package session owns a single conversation's history and persists it.
package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/db"
)

type Message struct {
	Role    string
	Content string
}

type Session struct {
	ID        string
	Model     string
	CreatedAt time.Time
	db        *db.DB
}

// New creates a fresh session row.
func New(d *db.DB, model string) (*Session, error) {
	id := newID()
	now := time.Now().UTC()
	if _, err := d.Handle().Exec(db.QInsertSession, id, now.Unix(), model, ""); err != nil {
		return nil, err
	}
	return &Session{ID: id, Model: model, CreatedAt: now, db: d}, nil
}

// AttachLatest returns the most-recent session in the DB, or creates a new one if none exists.
func AttachLatest(d *db.DB, defaultModel string) (*Session, error) {
	var (
		id        string
		createdAt int64
		model     string
		title     string
	)
	err := d.Handle().QueryRow(db.QSelectLatestSession).Scan(&id, &createdAt, &model, &title)
	if err != nil {
		return New(d, defaultModel)
	}
	return &Session{
		ID:        id,
		Model:     model,
		CreatedAt: time.Unix(createdAt, 0).UTC(),
		db:        d,
	}, nil
}

func (s *Session) AppendTurn(role, content string) (int64, error) {
	res, err := s.db.Handle().Exec(db.QInsertTurn, s.ID, role, content, time.Now().UTC().Unix())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// History returns up to `limit` most-recent turns in chronological (ascending) order.
// Only turns with status='complete' are returned; cancelled/error turns are skipped.
func (s *Session) History(ctx context.Context, limit int) ([]Message, error) {
	rows, err := s.db.Handle().QueryContext(ctx, db.QSelectHistory, s.ID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.Role, &m.Content); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Session) UpdateTurnStats(id int64, tokensIn, tokensOut, latencyMs int) error {
	_, err := s.db.Handle().Exec(db.QUpdateTurnStats, tokensIn, tokensOut, latencyMs, id)
	return err
}

func (s *Session) UpdateTurnMetadata(id int64, reasoning string, envelope json.RawMessage) error {
	var r any = reasoning
	if reasoning == "" {
		r = nil
	}
	var m any = string(envelope)
	if len(envelope) == 0 {
		m = nil
	}
	_, err := s.db.Handle().Exec(db.QUpdateTurnMetadata, r, m, id)
	return err
}

func (s *Session) MarkTurnStatus(id int64, status string) error {
	_, err := s.db.Handle().Exec(db.QMarkTurnStatus, status, id)
	return err
}

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
```

- [ ] **Step 4:** Run; expect PASS.

```bash
go test ./internal/session/... -v
```

- [ ] **Step 5:** Commit.

```bash
cd ..
git add gormes/internal/session/
git commit -m "feat(gormes/session): conversation history + metadata persistence"
```

---

## Task 7: Provider Interface, Types, and Error Classifier

**Files:**
- Create: `gormes/internal/provider/provider.go`
- Create: `gormes/internal/provider/errors.go`

- [ ] **Step 1:** Create `gormes/internal/provider/provider.go`:

```go
// Package provider defines the LLM provider interface and shared types.
package provider

import (
	"context"
	"encoding/json"
)

// Provider is implemented by every LLM backend.
type Provider interface {
	// Stream issues a completion request and returns a channel of Deltas.
	// The channel is closed by the Provider when streaming ends (normal or error).
	// The final Delta before close has Done=true and, if applicable, Err set.
	Stream(ctx context.Context, req Request) (<-chan Delta, error)
	Name() string
}

type Request struct {
	Model    string
	Messages []Message
	Params   Params
}

type Params struct {
	Temperature float64
	MaxTokens   int
}

type Message struct {
	Role    string // "system" | "user" | "assistant"
	Content string
}

type Delta struct {
	Token        string          // incremental assistant content
	Reasoning    string          // incremental thinking content (o1, <think>, Anthropic reasoning)
	TokensIn     int             // set on final delta only
	TokensOut    int             // running count on each delta; final on Done
	Done         bool
	FinishReason string          // set on Done: "stop" | "length" | "tool_calls" | "error" | ...
	RawEnvelope  json.RawMessage // final provider payload, stored verbatim in DB metadata
	Err          error           // non-nil → terminal error
}
```

- [ ] **Step 2:** Create `gormes/internal/provider/errors.go`:

```go
package provider

import (
	"errors"
	"net"
	"net/http"
	"strings"
)

type ErrorClass int

const (
	ClassUnknown ErrorClass = iota
	ClassRetryable           // 429, 500, 502, 503, 504, network timeouts
	ClassFatal               // 401, 403, context-length, malformed
)

// HTTPError wraps a non-2xx response so Classify can see the status.
type HTTPError struct {
	Status int
	Body   string
}

func (e *HTTPError) Error() string {
	return http.StatusText(e.Status) + ": " + e.Body
}

// Classify inspects an error returned anywhere in the provider pipeline.
func Classify(err error) ErrorClass {
	if err == nil {
		return ClassUnknown
	}
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		switch httpErr.Status {
		case 429, 500, 502, 503, 504:
			return ClassRetryable
		case 401, 403:
			return ClassFatal
		}
		if strings.Contains(strings.ToLower(httpErr.Body), "context length") {
			return ClassFatal
		}
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return ClassRetryable
	}
	return ClassUnknown
}
```

- [ ] **Step 3:** Write a tiny test to lock behaviour. Create `gormes/internal/provider/errors_test.go`:

```go
package provider

import (
	"errors"
	"testing"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want ErrorClass
	}{
		{"nil", nil, ClassUnknown},
		{"429", &HTTPError{Status: 429}, ClassRetryable},
		{"500", &HTTPError{Status: 500}, ClassRetryable},
		{"401", &HTTPError{Status: 401}, ClassFatal},
		{"context-length", &HTTPError{Status: 400, Body: "context length exceeded"}, ClassFatal},
		{"plain", errors.New("boom"), ClassUnknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Classify(c.err); got != c.want {
				t.Errorf("Classify = %v, want %v", got, c.want)
			}
		})
	}
}
```

- [ ] **Step 4:** Run; expect PASS.

```bash
cd gormes
go test ./internal/provider/...
```

- [ ] **Step 5:** Commit.

```bash
cd ..
git add gormes/internal/provider/provider.go gormes/internal/provider/errors.go gormes/internal/provider/errors_test.go
git commit -m "feat(gormes/provider): Provider interface, types, and error classifier"
```

---

## Task 8: MockProvider

**Files:**
- Create: `gormes/internal/provider/mock.go`
- Create: `gormes/internal/provider/mock_test.go`

- [ ] **Step 1:** Write the failing test. Create `gormes/internal/provider/mock_test.go`:

```go
package provider

import (
	"context"
	"testing"
)

func TestMockProvider_Script(t *testing.T) {
	m := NewMock("mock")
	m.Script([]Delta{
		{Token: "hel"},
		{Token: "lo"},
		{Done: true, TokensIn: 5, TokensOut: 2, FinishReason: "stop"},
	})

	ch, err := m.Stream(context.Background(), Request{Model: "x"})
	if err != nil {
		t.Fatal(err)
	}
	var got []Delta
	for d := range ch {
		got = append(got, d)
	}
	if len(got) != 3 {
		t.Fatalf("got %d deltas, want 3", len(got))
	}
	if got[2].FinishReason != "stop" {
		t.Errorf("final finish_reason = %q", got[2].FinishReason)
	}
}

func TestMockProvider_CancelCutsStream(t *testing.T) {
	m := NewMock("mock")
	m.Script([]Delta{
		{Token: "a"}, {Token: "b"}, {Token: "c"},
		{Done: true},
	})
	ctx, cancel := context.WithCancel(context.Background())
	ch, _ := m.Stream(ctx, Request{})
	<-ch // consume first delta
	cancel()
	// Drain until channel closes — must close eventually.
	for range ch {
	}
}
```

- [ ] **Step 2:** Run; expect FAIL.

```bash
cd gormes
go test ./internal/provider/...
```

- [ ] **Step 3:** Implement `gormes/internal/provider/mock.go`:

```go
package provider

import "context"

// MockProvider emits a scripted sequence of Deltas. Useful for agent tests.
type MockProvider struct {
	name    string
	scripts [][]Delta
}

func NewMock(name string) *MockProvider {
	return &MockProvider{name: name}
}

func (m *MockProvider) Name() string { return m.name }

// Script appends a delta sequence to be emitted by the next Stream call.
func (m *MockProvider) Script(deltas []Delta) {
	m.scripts = append(m.scripts, deltas)
}

func (m *MockProvider) Stream(ctx context.Context, _ Request) (<-chan Delta, error) {
	if len(m.scripts) == 0 {
		ch := make(chan Delta)
		close(ch)
		return ch, nil
	}
	script := m.scripts[0]
	m.scripts = m.scripts[1:]

	out := make(chan Delta, 1)
	go func() {
		defer close(out)
		for _, d := range script {
			select {
			case <-ctx.Done():
				return
			case out <- d:
			}
		}
	}()
	return out, nil
}
```

- [ ] **Step 4:** Run; expect PASS.

```bash
go test ./internal/provider/... -v
```

- [ ] **Step 5:** Commit.

```bash
cd ..
git add gormes/internal/provider/mock.go gormes/internal/provider/mock_test.go
git commit -m "feat(gormes/provider): MockProvider for agent tests"
```

---

## Task 9: OpenRouter Provider (SSE Streaming)

**Files:**
- Create: `gormes/internal/provider/openrouter.go`
- Create: `gormes/internal/provider/openrouter_test.go`

- [ ] **Step 1:** Write the failing test using a `httptest` fixture. Create `gormes/internal/provider/openrouter_test.go`:

```go
package provider

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const fixtureSSE = `data: {"id":"1","choices":[{"delta":{"content":"hel"}}]}

data: {"id":"1","choices":[{"delta":{"content":"lo"}}]}

data: {"id":"1","choices":[{"delta":{"content":"","reasoning":"thinking..."}}]}

data: {"id":"1","choices":[{"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":2}}

data: [DONE]

`

func TestOpenRouter_Stream_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, "unauthorized", 401)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		bw := bufio.NewWriter(w)
		fmt.Fprint(bw, fixtureSSE)
		bw.Flush()
	}))
	defer srv.Close()

	p := NewOpenRouter("test-key")
	p.baseURL = srv.URL

	ch, err := p.Stream(context.Background(), Request{
		Model:    "anthropic/claude-opus-4-7",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	var tokens, reasoning strings.Builder
	var finalDelta Delta
	for d := range ch {
		tokens.WriteString(d.Token)
		reasoning.WriteString(d.Reasoning)
		if d.Done {
			finalDelta = d
		}
	}
	if tokens.String() != "hello" {
		t.Errorf("tokens = %q, want %q", tokens.String(), "hello")
	}
	if reasoning.String() != "thinking..." {
		t.Errorf("reasoning = %q", reasoning.String())
	}
	if finalDelta.FinishReason != "stop" {
		t.Errorf("finish = %q, want stop", finalDelta.FinishReason)
	}
	if finalDelta.TokensIn != 5 || finalDelta.TokensOut != 2 {
		t.Errorf("usage = in:%d out:%d, want 5/2", finalDelta.TokensIn, finalDelta.TokensOut)
	}
}

func TestOpenRouter_HTTPErrorClassified(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", 429)
	}))
	defer srv.Close()

	p := NewOpenRouter("k")
	p.baseURL = srv.URL
	_, err := p.Stream(context.Background(), Request{Model: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	if Classify(err) != ClassRetryable {
		t.Errorf("Classify = %v, want ClassRetryable", Classify(err))
	}
}
```

- [ ] **Step 2:** Run; expect FAIL.

- [ ] **Step 3:** Implement `gormes/internal/provider/openrouter.go`:

```go
package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultOpenRouterURL = "https://openrouter.ai/api/v1/chat/completions"

type OpenRouter struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

func NewOpenRouter(apiKey string) *OpenRouter {
	return &OpenRouter{
		apiKey:  apiKey,
		baseURL: defaultOpenRouterURL,
		http:    &http.Client{Timeout: 10 * time.Minute},
	}
}

func (p *OpenRouter) Name() string { return "openrouter" }

type orMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type orRequest struct {
	Model       string      `json:"model"`
	Messages    []orMessage `json:"messages"`
	Stream      bool        `json:"stream"`
	Temperature float64     `json:"temperature,omitempty"`
	MaxTokens   int         `json:"max_tokens,omitempty"`
}

func (p *OpenRouter) Stream(ctx context.Context, req Request) (<-chan Delta, error) {
	msgs := make([]orMessage, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = orMessage{Role: m.Role, Content: m.Content}
	}
	body, err := json.Marshal(orRequest{
		Model:       req.Model,
		Messages:    msgs,
		Stream:      true,
		Temperature: req.Params.Temperature,
		MaxTokens:   req.Params.MaxTokens,
	})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		return nil, &HTTPError{Status: resp.StatusCode, Body: string(raw)}
	}

	out := make(chan Delta, 8)
	go p.readStream(resp.Body, out)
	return out, nil
}

type orChunk struct {
	Choices []struct {
		Delta struct {
			Content   string `json:"content"`
			Reasoning string `json:"reasoning"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage,omitempty"`
}

func (p *OpenRouter) readStream(body io.ReadCloser, out chan<- Delta) {
	defer close(out)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	var tokensOut int
	var lastChunk []byte

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}
		var chunk orChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue // skip malformed frames rather than fail the whole stream
		}
		lastChunk = []byte(payload)

		if len(chunk.Choices) == 0 {
			continue
		}
		c := chunk.Choices[0]
		if c.Delta.Content != "" || c.Delta.Reasoning != "" {
			tokensOut++ // coarse counter; final usage overrides
			out <- Delta{
				Token:     c.Delta.Content,
				Reasoning: c.Delta.Reasoning,
				TokensOut: tokensOut,
			}
		}
		if c.FinishReason != "" {
			final := Delta{
				Done:         true,
				FinishReason: c.FinishReason,
				TokensOut:    tokensOut,
				RawEnvelope:  lastChunk,
			}
			if chunk.Usage != nil {
				final.TokensIn = chunk.Usage.PromptTokens
				final.TokensOut = chunk.Usage.CompletionTokens
			}
			out <- final
			return
		}
	}
	if err := scanner.Err(); err != nil {
		out <- Delta{Done: true, Err: fmt.Errorf("sse scan: %w", err)}
	}
}
```

- [ ] **Step 4:** Run; expect PASS.

```bash
cd gormes
go test ./internal/provider/... -v
```

- [ ] **Step 5:** Commit.

```bash
cd ..
git add gormes/internal/provider/openrouter.go gormes/internal/provider/openrouter_test.go
git commit -m "feat(gormes/provider): OpenRouter SSE streaming client"
```

---

## Task 10: Telemetry Package

**Files:**
- Create: `gormes/internal/telemetry/telemetry.go`
- Create: `gormes/internal/telemetry/telemetry_test.go`

- [ ] **Step 1:** Write the failing test. Create `gormes/internal/telemetry/telemetry_test.go`:

```go
package telemetry

import (
	"testing"
	"time"
)

func TestSnapshotAfterTicks(t *testing.T) {
	tm := New()
	tm.StartTurn()
	tm.Tick(1)
	tm.Tick(2)
	tm.FinishTurn(10 * time.Millisecond)
	s := tm.Snapshot()
	if s.TokensOutTotal != 2 {
		t.Errorf("tokens_out_total = %d, want 2", s.TokensOutTotal)
	}
	if s.LatencyMsLast != 10 {
		t.Errorf("latency_ms_last = %d, want 10", s.LatencyMsLast)
	}
}
```

- [ ] **Step 2:** Run; expect FAIL.

- [ ] **Step 3:** Implement `gormes/internal/telemetry/telemetry.go`:

```go
// Package telemetry tracks per-session counters emitted to the TUI.
package telemetry

import (
	"time"
)

type Snapshot struct {
	Model          string
	TokensInTotal  int
	TokensOutTotal int
	LatencyMsLast  int
	TokensPerSec   float64
}

type Telemetry struct {
	snap       Snapshot
	turnStart  time.Time
	turnTokens int
	ema        float64 // EMA of tokens/sec
}

func New() *Telemetry { return &Telemetry{} }

func (t *Telemetry) StartTurn() {
	t.turnStart = time.Now()
	t.turnTokens = 0
}

func (t *Telemetry) Tick(tokensOut int) {
	delta := tokensOut - t.turnTokens
	if delta < 0 {
		delta = 0
	}
	t.turnTokens = tokensOut
	t.snap.TokensOutTotal += delta

	elapsed := time.Since(t.turnStart).Seconds()
	if elapsed > 0 {
		tps := float64(t.turnTokens) / elapsed
		const alpha = 0.2
		t.ema = alpha*tps + (1-alpha)*t.ema
		t.snap.TokensPerSec = t.ema
	}
}

func (t *Telemetry) FinishTurn(latency time.Duration) {
	t.snap.LatencyMsLast = int(latency / time.Millisecond)
}

func (t *Telemetry) SetModel(m string) { t.snap.Model = m }

func (t *Telemetry) Snapshot() Snapshot { return t.snap }
```

- [ ] **Step 4:** Run; expect PASS.

```bash
cd gormes
go test ./internal/telemetry/...
```

- [ ] **Step 5:** Commit.

```bash
cd ..
git add gormes/internal/telemetry/
git commit -m "feat(gormes/telemetry): per-session counters with tokens/sec EMA"
```

---

## Task 11: Pybridge Stub

**Files:**
- Create: `gormes/internal/pybridge/pybridge.go`

- [ ] **Step 1:** Create `gormes/internal/pybridge/pybridge.go` (interface only; no implementation in M1):

```go
// Package pybridge reserves the M4 boundary for the Python-subprocess tool bridge.
// M1 ships only the Tool interface and a sentinel ErrNotImplemented; no subprocess
// runtime lives here until M4's brainstorm.
package pybridge

import (
	"context"
	"encoding/json"
	"errors"
)

// Tool is the boundary any tool must satisfy. M2 adds native Go tools; M4 adds a
// subprocess-backed implementation.
type Tool interface {
	Name() string
	Call(ctx context.Context, args json.RawMessage) (json.RawMessage, error)
}

// ErrNotImplemented is returned by placeholder Tool implementations until M2.
var ErrNotImplemented = errors.New("gormes/pybridge: tool calls not implemented until M2")
```

- [ ] **Step 2:** Commit.

```bash
cd ..
git add gormes/internal/pybridge/
git commit -m "feat(gormes/pybridge): M4-boundary stub — Tool interface only"
```

---

## Task 12: Agent Orchestrator — Happy-Path Turn

**Files:**
- Create: `gormes/internal/agent/default_prompt.go`
- Create: `gormes/internal/agent/agent.go`
- Create: `gormes/internal/agent/agent_test.go`

- [ ] **Step 1:** Create `gormes/internal/agent/default_prompt.go`:

```go
package agent

import "github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"

const defaultSystemPrompt = `You are Gormes, a Go-native AI agent in the "Ignition" vertical slice (M1).
You currently have no tools — you can converse but cannot call external functions.
Keep answers concise and direct. Reply in plain text without markdown decoration
unless the user asks otherwise.`

// BuildSystemPrompt returns the system prompt to prepend on every turn.
// Precedence: config.Agent.SystemPrompt overrides the built-in default.
func BuildSystemPrompt(cfg config.Config) string {
	if cfg.Agent.SystemPrompt != "" {
		return cfg.Agent.SystemPrompt
	}
	return defaultSystemPrompt
}
```

- [ ] **Step 2:** Write the failing test. Create `gormes/internal/agent/agent_test.go`:

```go
package agent

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/db"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/provider"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/session"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/telemetry"
)

func newAgentFixture(t *testing.T) (*Agent, *provider.MockProvider, *session.Session) {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "a.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	s, err := session.New(d, "test-model")
	if err != nil {
		t.Fatal(err)
	}
	p := provider.NewMock("mock")
	tm := telemetry.New()
	updates := make(chan UIUpdate, 64)
	cfg := config.Config{Provider: config.ProviderCfg{Model: "test-model"}}
	a := New(cfg, p, s, tm, updates)
	return a, p, s
}

func TestAgent_HappyPath(t *testing.T) {
	a, p, s := newAgentFixture(t)
	p.Script([]provider.Delta{
		{Token: "hel", TokensOut: 1},
		{Token: "lo", TokensOut: 2},
		{Done: true, FinishReason: "stop", TokensIn: 3, TokensOut: 2},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := a.HandleInput(ctx, "hi")
	if err != nil {
		t.Fatalf("HandleInput: %v", err)
	}

	msgs, err := s.History(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("history len = %d, want 2", len(msgs))
	}
	if msgs[1].Content != "hello" {
		t.Errorf("assistant content = %q, want hello", msgs[1].Content)
	}
}
```

- [ ] **Step 3:** Run; expect FAIL.

- [ ] **Step 4:** Implement `gormes/internal/agent/agent.go`:

```go
// Package agent orchestrates the turn lifecycle: build request, stream, persist.
package agent

import (
	"context"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/provider"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/session"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/telemetry"
)

// UIUpdate is what the TUI consumes from the agent.
type UIUpdate struct {
	Kind      UpdateKind
	Token     string
	Telemetry telemetry.Snapshot
	SoulEvent string
	Err       error
}

type UpdateKind int

const (
	UpdateToken UpdateKind = iota
	UpdateTurnStart
	UpdateTurnComplete
	UpdateTelemetry
	UpdateSoulEvent
	UpdateError
)

type Agent struct {
	cfg     config.Config
	prov    provider.Provider
	sess    *session.Session
	tm      *telemetry.Telemetry
	updates chan<- UIUpdate
}

func New(cfg config.Config, p provider.Provider, s *session.Session, tm *telemetry.Telemetry, u chan<- UIUpdate) *Agent {
	tm.SetModel(cfg.Provider.Model)
	return &Agent{cfg: cfg, prov: p, sess: s, tm: tm, updates: u}
}

// HandleInput processes one user turn end-to-end. Returns only on turn completion or fatal error.
func (a *Agent) HandleInput(ctx context.Context, text string) error {
	if _, err := a.sess.AppendTurn("user", text); err != nil {
		a.emit(UIUpdate{Kind: UpdateError, Err: err})
		return err
	}
	history, err := a.sess.History(ctx, 100)
	if err != nil {
		a.emit(UIUpdate{Kind: UpdateError, Err: err})
		return err
	}

	msgs := []provider.Message{{Role: "system", Content: BuildSystemPrompt(a.cfg)}}
	for _, m := range history {
		msgs = append(msgs, provider.Message{Role: m.Role, Content: m.Content})
	}

	a.emit(UIUpdate{Kind: UpdateSoulEvent, SoulEvent: "thinking"})
	a.emit(UIUpdate{Kind: UpdateTurnStart})
	a.tm.StartTurn()
	start := time.Now()

	deltaCh, err := a.prov.Stream(ctx, provider.Request{
		Model:    a.cfg.Provider.Model,
		Messages: msgs,
	})
	if err != nil {
		a.emit(UIUpdate{Kind: UpdateError, Err: err})
		return err
	}
	a.emit(UIUpdate{Kind: UpdateSoulEvent, SoulEvent: "streaming"})

	var buf, reasoning strings.Builder
	var finalDelta provider.Delta
	cancelled := false

streamLoop:
	for {
		select {
		case <-ctx.Done():
			cancelled = true
			break streamLoop
		case d, ok := <-deltaCh:
			if !ok {
				break streamLoop
			}
			if d.Err != nil {
				a.emit(UIUpdate{Kind: UpdateError, Err: d.Err})
				return d.Err
			}
			if d.Token != "" {
				buf.WriteString(d.Token)
				a.emit(UIUpdate{Kind: UpdateToken, Token: d.Token})
			}
			if d.Reasoning != "" {
				reasoning.WriteString(d.Reasoning)
			}
			a.tm.Tick(d.TokensOut)
			a.emit(UIUpdate{Kind: UpdateTelemetry, Telemetry: a.tm.Snapshot()})
			if d.Done {
				finalDelta = d
				break streamLoop
			}
		}
	}

	latency := time.Since(start)
	a.tm.FinishTurn(latency)

	turnID, err := a.sess.AppendTurn("assistant", buf.String())
	if err != nil {
		a.emit(UIUpdate{Kind: UpdateError, Err: err})
		return err
	}
	if cancelled {
		_ = a.sess.MarkTurnStatus(turnID, "cancelled")
	}
	_ = a.sess.UpdateTurnStats(turnID, finalDelta.TokensIn, finalDelta.TokensOut, int(latency/time.Millisecond))
	_ = a.sess.UpdateTurnMetadata(turnID, reasoning.String(), finalDelta.RawEnvelope)

	a.emit(UIUpdate{Kind: UpdateTurnComplete, Telemetry: a.tm.Snapshot()})
	a.emit(UIUpdate{Kind: UpdateSoulEvent, SoulEvent: "idle"})
	return nil
}

func (a *Agent) emit(u UIUpdate) {
	select {
	case a.updates <- u:
	default:
		// TUI is slow or gone — drop updates rather than block the agent.
	}
}
```

- [ ] **Step 5:** Run; expect PASS.

```bash
cd gormes
go test ./internal/agent/... -v
```

- [ ] **Step 6:** Commit.

```bash
cd ..
git add gormes/internal/agent/
git commit -m "feat(gormes/agent): turn orchestrator with streaming, telemetry, persistence"
```

---

## Task 13: Agent — Cancellation Coverage

**Files:**
- Modify: `gormes/internal/agent/agent_test.go`

- [ ] **Step 1:** Append a cancellation test to `gormes/internal/agent/agent_test.go`:

```go
func TestAgent_CancelMidStream(t *testing.T) {
	a, p, s := newAgentFixture(t)
	p.Script([]provider.Delta{
		{Token: "a", TokensOut: 1},
		{Token: "b", TokensOut: 2},
		{Token: "c", TokensOut: 3},
		{Done: true, FinishReason: "stop"},
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()
	_ = a.HandleInput(ctx, "hi")

	// Re-open a fresh *sql.DB-backed read to inspect status.
	rows, err := s.History(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	// History filters out cancelled turns — so it returns only the user turn.
	if len(rows) != 1 {
		t.Errorf("completed history = %d rows, want 1 (user only, cancelled assistant filtered)", len(rows))
	}
}
```

- [ ] **Step 2:** Run the new test.

```bash
cd gormes
go test ./internal/agent/... -run CancelMidStream -v
```

Expected: PASS (the cancel path already works in the implementation from Task 12).

- [ ] **Step 3:** Commit.

```bash
cd ..
git add gormes/internal/agent/agent_test.go
git commit -m "test(gormes/agent): add mid-stream cancellation coverage"
```

---

## Task 14: Public `pkg/gormes` Re-Exports

**Files:**
- Create: `gormes/pkg/gormes/types.go`

- [ ] **Step 1:** Create `gormes/pkg/gormes/types.go`:

```go
// Package gormes re-exports the public types that external consumers might import.
// Keep this surface small and stable — internal packages remain the canonical home.
package gormes

import (
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/agent"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/provider"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/pybridge"
)

type (
	Provider = provider.Provider
	Request  = provider.Request
	Delta    = provider.Delta
	Message  = provider.Message

	Tool = pybridge.Tool

	UIUpdate   = agent.UIUpdate
	UpdateKind = agent.UpdateKind
)
```

- [ ] **Step 2:** Verify compilation.

```bash
cd gormes
go build ./...
```

- [ ] **Step 3:** Commit.

```bash
cd ..
git add gormes/pkg/gormes/
git commit -m "feat(gormes/pkg): public type re-exports"
```

---

## Task 15: TUI Model + WindowSizeMsg

**Files:**
- Create: `gormes/internal/tui/model.go`

- [ ] **Step 1:** Add Bubble Tea dependencies.

```bash
cd gormes
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/charmbracelet/lipgloss@latest
```

- [ ] **Step 2:** Create `gormes/internal/tui/model.go`:

```go
// Package tui renders the Gormes Dashboard (main pane + telemetry + soul monitor).
package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/agent"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/telemetry"
)

const soulBufferSize = 10

// InputHandler is called when the user submits a line. The caller (main) wires
// this to a goroutine that invokes Agent.HandleInput.
type InputHandler func(ctx context.Context, text string)

type Model struct {
	width, height int

	viewport  viewport.Model
	editor    textarea.Model
	conv      []convLine
	telemetry telemetry.Snapshot
	soul      []soulEntry

	model       string
	inFlight    bool
	cancelTurn  context.CancelFunc
	submitQueue chan string
	updates     <-chan agent.UIUpdate
	input       InputHandler

	assistantBuf string
}

type convLine struct {
	role    string // "user" | "assistant" | "system" | "error"
	content string
}

type soulEntry struct {
	at   time.Time
	text string
}

func NewModel(model string, updates <-chan agent.UIUpdate, input InputHandler) Model {
	vp := viewport.New(80, 20)
	ta := textarea.New()
	ta.Placeholder = "Type a message and hit Enter…"
	ta.ShowLineNumbers = false
	ta.Focus()

	return Model{
		viewport:    vp,
		editor:      ta,
		model:       model,
		updates:     updates,
		input:       input,
		submitQueue: make(chan string, 8),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.listenUpdates())
}

// uiUpdateMsg wraps agent.UIUpdate so Bubble Tea delivers it to Update.
type uiUpdateMsg agent.UIUpdate

// listenUpdates returns a Cmd that blocks on the updates channel and emits
// one Bubble Tea message. The Update handler re-schedules it.
func (m Model) listenUpdates() tea.Cmd {
	return func() tea.Msg {
		u, ok := <-m.updates
		if !ok {
			return tea.Quit()
		}
		return uiUpdateMsg(u)
	}
}
```

- [ ] **Step 3:** Verify compile.

```bash
go build ./internal/tui/...
```

Expected: PASS (no tests yet; Update and View come in later tasks).

- [ ] **Step 4:** Commit.

```bash
cd ..
git add gormes/internal/tui/model.go gormes/go.mod gormes/go.sum
git commit -m "feat(gormes/tui): Bubble Tea Model scaffolding"
```

---

## Task 16: TUI View (lipgloss responsive layout)

**Files:**
- Create: `gormes/internal/tui/view.go`

- [ ] **Step 1:** Create `gormes/internal/tui/view.go`:

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	borderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	muted       = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	header      = lipgloss.NewStyle().Bold(true)
	userRole    = lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Bold(true)
	assistRole  = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	errorRole   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

func (m Model) View() string {
	if m.width < 20 || m.height < 10 {
		return "terminal too narrow — resize to at least 20×10"
	}

	mainW := m.width
	sidebarW := 0
	if m.width >= 100 {
		sidebarW = 28
	} else if m.width >= 80 {
		sidebarW = 24
	}
	mainW = m.width - sidebarW - 4 // borders + gap

	convBody := m.renderConversation(mainW)
	main := borderStyle.Width(mainW).Height(m.height - 6).Render(convBody)

	var sidebar string
	if sidebarW > 0 {
		sidebar = borderStyle.Width(sidebarW).Height(m.height - 6).Render(m.renderSidebar(sidebarW))
	}

	top := lipgloss.JoinHorizontal(lipgloss.Top, main, sidebar)
	editor := borderStyle.Width(m.width - 2).Render(m.editor.View())
	statusLine := muted.Render(fmt.Sprintf("model: %s · session live", m.model))
	return lipgloss.JoinVertical(lipgloss.Left, top, editor, statusLine)
}

func (m Model) renderConversation(width int) string {
	var lines []string
	for _, c := range m.conv {
		tag := roleTag(c.role)
		lines = append(lines, tag+" "+lipgloss.NewStyle().Width(width-4).Render(c.content))
	}
	if m.inFlight && m.assistantBuf != "" {
		lines = append(lines, assistRole.Render("assistant:")+" "+m.assistantBuf)
	}
	return strings.Join(lines, "\n\n")
}

func (m Model) renderSidebar(width int) string {
	var b strings.Builder
	b.WriteString(header.Render("Telemetry") + "\n")
	b.WriteString(fmt.Sprintf(" model: %s\n", m.telemetry.Model))
	b.WriteString(fmt.Sprintf(" tok/s: %.1f\n", m.telemetry.TokensPerSec))
	b.WriteString(fmt.Sprintf(" latency: %d ms\n", m.telemetry.LatencyMsLast))
	b.WriteString(fmt.Sprintf(" in/out: %d/%d\n", m.telemetry.TokensInTotal, m.telemetry.TokensOutTotal))
	b.WriteString(strings.Repeat("─", width-2) + "\n")
	b.WriteString(header.Render("Soul Monitor") + "\n")
	for _, s := range m.soul {
		b.WriteString(fmt.Sprintf(" [%s] %s\n", s.at.Format("15:04:05"), s.text))
	}
	return b.String()
}

func roleTag(role string) string {
	switch role {
	case "user":
		return userRole.Render("you:")
	case "assistant":
		return assistRole.Render("gormes:")
	case "error":
		return errorRole.Render("err:")
	default:
		return muted.Render(role + ":")
	}
}
```

- [ ] **Step 2:** Verify compile.

```bash
cd gormes
go build ./internal/tui/...
```

- [ ] **Step 3:** Commit.

```bash
cd ..
git add gormes/internal/tui/view.go
git commit -m "feat(gormes/tui): responsive lipgloss dashboard view"
```

---

## Task 17: TUI Update — keybindings + event routing

**Files:**
- Create: `gormes/internal/tui/update.go`

- [ ] **Step 1:** Create `gormes/internal/tui/update.go`:

```go
package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/agent"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 6
		m.editor.SetWidth(msg.Width - 4)

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			if m.inFlight && m.cancelTurn != nil {
				m.cancelTurn()
				return m, nil
			}
			return m, tea.Quit
		case tea.KeyCtrlD:
			return m, tea.Quit
		case tea.KeyCtrlL:
			m.conv = nil
		case tea.KeyEnter:
			if msg.Alt {
				// Shift+Enter (Alt under many terminals) = newline in editor.
				break
			}
			text := m.editor.Value()
			if text != "" && !m.inFlight {
				m.editor.Reset()
				m.conv = append(m.conv, convLine{role: "user", content: text})
				m.inFlight = true
				m.assistantBuf = ""
				ctx, cancel := context.WithCancel(context.Background())
				m.cancelTurn = cancel
				go m.input(ctx, text)
			}
			return m, textarea.Blink
		}

	case uiUpdateMsg:
		u := agent.UIUpdate(msg)
		m = m.applyUpdate(u)
		cmds = append(cmds, m.listenUpdates())
	}

	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	cmds = append(cmds, cmd)
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m Model) applyUpdate(u agent.UIUpdate) Model {
	switch u.Kind {
	case agent.UpdateToken:
		m.assistantBuf += u.Token
	case agent.UpdateTelemetry:
		m.telemetry = u.Telemetry
	case agent.UpdateSoulEvent:
		m.soul = append(m.soul, soulEntry{at: time.Now(), text: u.SoulEvent})
		if len(m.soul) > soulBufferSize {
			m.soul = m.soul[len(m.soul)-soulBufferSize:]
		}
	case agent.UpdateTurnComplete:
		if m.assistantBuf != "" {
			m.conv = append(m.conv, convLine{role: "assistant", content: m.assistantBuf})
		}
		m.assistantBuf = ""
		m.inFlight = false
		m.cancelTurn = nil
	case agent.UpdateError:
		m.inFlight = false
		m.cancelTurn = nil
		if u.Err != nil {
			m.conv = append(m.conv, convLine{role: "error", content: u.Err.Error()})
		}
	}
	return m
}
```

- [ ] **Step 2:** Verify compile.

```bash
cd gormes
go build ./internal/tui/...
```

- [ ] **Step 3:** Commit.

```bash
cd ..
git add gormes/internal/tui/update.go
git commit -m "feat(gormes/tui): keybindings and UIUpdate routing"
```

---

## Task 18: TUI teatest — type-send, cancel, resize

**Files:**
- Create: `gormes/internal/tui/tui_test.go`

- [ ] **Step 1:** Add the teatest dependency.

```bash
cd gormes
go get github.com/charmbracelet/x/exp/teatest@latest
```

- [ ] **Step 2:** Create `gormes/internal/tui/tui_test.go`:

```go
package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/agent"
)

func TestTypeSendRendersUserAndAssistant(t *testing.T) {
	updates := make(chan agent.UIUpdate, 16)
	handler := func(ctx context.Context, text string) {
		updates <- agent.UIUpdate{Kind: agent.UpdateSoulEvent, SoulEvent: "thinking"}
		updates <- agent.UIUpdate{Kind: agent.UpdateToken, Token: "hel"}
		updates <- agent.UIUpdate{Kind: agent.UpdateToken, Token: "lo"}
		updates <- agent.UIUpdate{Kind: agent.UpdateTurnComplete}
	}
	m := NewModel("test-model", updates, handler)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	tm.Type("hi")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFor(func(out []byte) bool { return contains(out, "hello") }, teatest.WithDuration(2*time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlD})
	tm.WaitFinished(t)
}

func TestResizeDoesNotCrash(t *testing.T) {
	updates := make(chan agent.UIUpdate, 1)
	m := NewModel("test-model", updates, func(ctx context.Context, s string) {})
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))
	for _, w := range []int{200, 80, 50, 10, 200} {
		tm.Send(tea.WindowSizeMsg{Width: w, Height: 24})
	}
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlD})
	tm.WaitFinished(t)
}

func contains(haystack []byte, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if string(haystack[i:i+len(needle)]) == needle {
			return true
		}
	}
	return false
}
```

- [ ] **Step 3:** Run; expect PASS.

```bash
go test ./internal/tui/... -v
```

Note: `teatest` API surface occasionally changes between Charm releases. If a method name differs (e.g., `teatest.NewTestModel` is renamed), adjust to the current API and keep the test shape: send a key, wait for an output substring, send Ctrl+D, wait for program exit.

- [ ] **Step 4:** Commit.

```bash
cd ..
git add gormes/internal/tui/tui_test.go gormes/go.mod gormes/go.sum
git commit -m "test(gormes/tui): teatest coverage for type-send and resize"
```

---

## Task 19: Wire Everything in `cmd/gormes/main.go`

**Files:**
- Create: `gormes/cmd/gormes/main.go`

- [ ] **Step 1:** Create `gormes/cmd/gormes/main.go`:

```go
// Command gormes is the Gormes binary entry point. It wires config → DB →
// session → provider → agent → TUI and handles signals + panic recovery.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/agent"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/db"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/provider"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/session"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/tui"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			dumpCrash(r)
			os.Exit(2)
		}
	}()

	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	if cfg.OpenRouterAPIKey == "" {
		fmt.Fprintln(os.Stderr, "OPENROUTER_API_KEY is not set")
		os.Exit(1)
	}

	d, err := db.Open(cfg.Storage.DBPath)
	if err != nil {
		slog.Error("db", "err", err)
		os.Exit(1)
	}

	rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	sess, err := session.AttachLatest(d, cfg.Provider.Model)
	if err != nil {
		slog.Error("session", "err", err)
		os.Exit(1)
	}

	prov := provider.NewOpenRouter(cfg.OpenRouterAPIKey)
	tm := telemetry.New()
	updates := make(chan agent.UIUpdate, 128)
	a := agent.New(cfg, prov, sess, tm, updates)

	handler := func(ctx context.Context, text string) {
		_ = a.HandleInput(ctx, text)
	}
	model := tui.NewModel(cfg.Provider.Model, updates, handler)

	prog := tea.NewProgram(model, tea.WithAltScreen())
	go func() {
		<-rootCtx.Done()
		// 2-second shutdown budget per spec §10.4.
		time.AfterFunc(2*time.Second, func() {
			slog.Error("shutdown budget exceeded")
			os.Exit(3)
		})
		prog.Quit()
	}()

	if _, err := prog.Run(); err != nil {
		slog.Error("tui", "err", err)
		os.Exit(1)
	}
	_ = d.Close()
}

func dumpCrash(r any) {
	dir := config.CrashLogDir()
	_ = os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, fmt.Sprintf("crash-%d.log", time.Now().Unix()))
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "panic:", r)
		fmt.Fprintln(os.Stderr, string(debug.Stack()))
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "panic: %v\n\n%s\n", r, debug.Stack())
	fmt.Fprintln(os.Stderr, "gormes crashed — log at "+path)
}
```

- [ ] **Step 2:** Build.

```bash
cd gormes
make build
```

Expected: `bin/gormes` exists. Do not run it interactively in this task — the manual smoke test lives in Task 21.

- [ ] **Step 3:** Commit.

```bash
cd ..
git add gormes/cmd/gormes/main.go
git commit -m "feat(gormes/cmd): wire config, DB, session, provider, agent, TUI"
```

---

## Task 20: Live Integration Test (build tag)

**Files:**
- Create: `gormes/internal/provider/live_test.go`

- [ ] **Step 1:** Create `gormes/internal/provider/live_test.go`:

```go
//go:build live

package provider

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestOpenRouter_LiveStream(t *testing.T) {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		t.Skip("OPENROUTER_API_KEY unset; skipping live test")
	}
	p := NewOpenRouter(key)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	ch, err := p.Stream(ctx, Request{
		Model: "anthropic/claude-opus-4-7",
		Messages: []Message{
			{Role: "system", Content: "Reply with exactly the word OK."},
			{Role: "user", Content: "go"},
		},
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	var out strings.Builder
	var got Delta
	for d := range ch {
		out.WriteString(d.Token)
		if d.Done {
			got = d
		}
	}
	if !strings.Contains(strings.ToUpper(out.String()), "OK") {
		t.Errorf("expected reply to contain OK, got %q", out.String())
	}
	if got.FinishReason == "" {
		t.Error("missing finish_reason on final delta")
	}
}
```

- [ ] **Step 2:** Verify the tagged test compiles.

```bash
cd gormes
go test -tags=live -run LiveStream ./internal/provider/... -v
```

Expected: PASS if `OPENROUTER_API_KEY` is set, SKIP otherwise. Either outcome is acceptable for the plan.

- [ ] **Step 3:** Commit.

```bash
cd ..
git add gormes/internal/provider/live_test.go
git commit -m "test(gormes/provider): live OpenRouter integration test behind -tags=live"
```

---

## Task 21: Success-Criteria Verification Checklist

**Files:**
- No file changes. This task verifies §18 of the spec.

- [ ] **Step 1:** `go build ./cmd/gormes` succeeds on the current host.

```bash
cd gormes
go build ./cmd/gormes
```

Expected: exit code 0, `gormes` binary produced.

- [ ] **Step 2:** `make test` passes with coverage ≥ 70 % on `internal/` (excluding `tui/`).

```bash
go test -cover ./internal/agent/... ./internal/config/... ./internal/db/... \
  ./internal/provider/... ./internal/session/... ./internal/telemetry/... \
  ./internal/pybridge/...
```

Expected: every package reports coverage ≥ 70 %. If one falls short, add a targeted test in that package and repeat.

- [ ] **Step 3:** `go test ./docs/...` passes the Markdown lint.

```bash
go test ./docs/...
```

Expected: PASS. If the plan filename is missing from the lint `targets` (Task 3, Step 3), add it back now.

- [ ] **Step 4:** Manual smoke test.

```bash
export OPENROUTER_API_KEY=<your key>
./bin/gormes
```

Then verify:
- The Debug/Dashboard TUI launches with no prior config beyond `OPENROUTER_API_KEY`.
- A typed prompt streams tokens live into the conversation pane.
- Soul Monitor shows `thinking → streaming → idle` during the turn.
- Telemetry pane displays non-zero `tok/s`, `latency`, and token counts.
- `Ctrl+C` mid-stream cancels cleanly; second `Ctrl+C` quits.
- Resizing the terminal while a turn is streaming does not panic the process.
- Relaunching the binary resumes the prior session.

- [ ] **Step 5:** Confirm no Python file has been modified.

```bash
cd ..
git log --oneline --name-only origin/main..HEAD | grep -vE '^(gormes/|$|[0-9a-f]+ )' || echo "no python changes — OK"
```

Expected: `no python changes — OK`.

- [ ] **Step 6:** Mark M0 and M1 as ✅ complete in `gormes/docs/ARCH_PLAN.md` section 4 (the milestone table) and commit.

```bash
# Edit gormes/docs/ARCH_PLAN.md — change:
#   | M0 — Scaffold | 🔨 in progress | ... |
#   | M1 — TUI + LLM | 🔨 in progress | ... |
# to:
#   | M0 — Scaffold | ✅ complete | ... |
#   | M1 — TUI + LLM | ✅ complete | ... |
git add gormes/docs/ARCH_PLAN.md
git commit -m "docs(gormes): mark M0+M1 milestones complete"
```

- [ ] **Step 7:** Rerun the doc lint to confirm the ARCH_PLAN changes keep portability.

```bash
cd gormes
go test ./docs/...
```

Expected: PASS.

---

## Appendix A: Self-Review Results

The author ran the plan self-review against spec sections. Summary:

**Spec coverage by section:**
- §3 (Architectural Principles) — embodied by Tasks 12, 15, 16, 19 (channels, ctx cancellation, CGO-free via `modernc.org/sqlite`).
- §4 (Process Model) — Task 12 (Agent) + Task 19 (main wiring).
- §5 (Directory Layout) — exactly mapped in the File Structure Map at plan top.
- §6 (Core Interfaces) — Tasks 7 (Provider), 11 (Tool stub), 6 (Session), 12 (UIUpdate).
- §6.1 (System Prompt) — Task 12 Step 1 (`default_prompt.go`).
- §7 (Data Flow) — Task 12 implementation.
- §8 (Persistence + schema including `reasoning`/`metadata`) — Task 5, Task 6.
- §9 (TUI Dashboard layout + responsive rules + SIGWINCH) — Tasks 15, 16, 17, 18 (resize test).
- §10 (Error Handling) — Task 7 (classifier), Task 12 (error path), Task 19 (panic recovery + 2-second shutdown budget).
- §11 (Configuration) — Task 4.
- §12 (Telemetry) — Task 10.
- §13 (Testing Strategy) — every task includes unit tests; Task 18 covers TUI teatest; Task 20 covers live integration.
- §14 (Build & Tooling) — Task 1 (Makefile, go.mod).
- §15 (Dependency Map) — satisfied via `go get` calls in the relevant tasks.
- §16 (No Python modified) — verified in Task 21 Step 5.
- §17 (Out-of-scope) — plan omits every deferred feature by construction.
- §18 (Success Criteria, 11 items) — enumerated in Task 21.
- §19 (Risks) — token-rate coalescer risk (16 ms) is NOT implemented in M1; the provider emits per-delta and the TUI renders each — acceptable at typical 50 tok/s, revisit only if the live smoke test shows jank.
- §21 (Documentation Strategy) — Tasks 2 (ARCH_PLAN) and 3 (Markdown lint).

**Placeholder scan:** No TBD / TODO / "similar to Task N" references remain. All code blocks contain concrete code.

**Type consistency:** Cross-task signatures checked — `Session.AppendTurn`, `UpdateTurnStats`, `UpdateTurnMetadata`, `MarkTurnStatus` match between Tasks 5/6/12. `agent.UIUpdate` / `agent.UpdateKind` match between Tasks 12 and 15/17/18. `provider.Delta` fields (`Token`, `Reasoning`, `RawEnvelope`, `FinishReason`) match spec §6 and all consumer tasks.

**One intentional simplification:** the 16 ms token coalescer mentioned in spec §19 as a jank mitigation is not implemented in M1. The provider streams per-delta and the TUI's `applyUpdate` appends immediately. If the live smoke test in Task 21 Step 4 reveals jank above ~100 tok/s, a follow-up commit adds a coalescer in `internal/agent` (batch `UpdateToken` emissions with a 16 ms ticker). This is recorded here so the deferral is explicit, not forgotten.
