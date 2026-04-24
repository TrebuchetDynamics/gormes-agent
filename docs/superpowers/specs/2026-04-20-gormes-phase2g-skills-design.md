# Phase 2.G — Skills System Spec

**Date:** 2026-04-20
**Status:** Draft — pending implementation plan
**Priority:** P0
**Upstream:** `tools/skill_manager_tool.py`, `agent/skill_commands.py`, `agent/skill_utils.py`, `agent/prompt_builder.py`

---

## Goal

Implement the Gormes Skills System — the **learning loop** that detects complex tasks, extracts reusable patterns, saves them as versioned skills, and improves them over time. This is the primary differentiator. Without it, Gormes is a stateless chat interface.

---

## What Is a Skill

A skill is **procedural memory** — a reusable approach for a recurring task type. Narrow and actionable, not broad declarative memory like entities in the graph.

Example: a skill for "respond to a GitHub PR review" contains the exact steps, tone, and tool patterns the agent learned from doing it successfully three times.

Skills follow the **agentskills.io SKILL.md format** for compatibility with the broader ecosystem.

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  Kernel / Agent Loop                                     │
│  ┌───────────────┐  ┌──────────────┐  ┌─────────────┐ │
│  │ Complexity    │  │ Extraction   │  │ Improvement │ │
│  │ Detector      │→ │ Engine       │→ │ Engine      │ │
│  │ (LLM call)    │  │ (LLM call)   │  │ (LLM call)  │ │
│  └───────────────┘  └──────────────┘  └─────────────┘ │
│         │                 │                 │           │
│         ▼                 ▼                 ▼           │
│  ┌─────────────────────────────────────────────────────┐│
│  │  SkillStore (SQLite metadata + filesystem content)  ││
│  │  ~/.local/share/gormes/skills/                     ││
│  └─────────────────────────────────────────────────────┘│
│         ▲                                              │
│         │                                              │
│  ┌──────┴──────────────────────────────────────────┐   │
│  │  Prompt Builder → skills injected into context   │   │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

---

## Data Structures

### Skill (stored in SQLite)

```sql
CREATE TABLE skills (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL,
    category    TEXT NOT NULL DEFAULT 'general',
    version     TEXT NOT NULL DEFAULT '1.0.0',
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL,
    use_count   INTEGER NOT NULL DEFAULT 0,
    last_used_at INTEGER,
    file_path   TEXT NOT NULL  -- relative path under skills dir
);

CREATE TABLE skill_improvements (
    id              TEXT PRIMARY KEY,
    skill_id        TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    feedback        TEXT NOT NULL,
    improved_content TEXT NOT NULL,
    created_at      INTEGER NOT NULL
);

CREATE TABLE skill_usage (
    skill_id  TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    used_at   INTEGER NOT NULL,
    helpful   INTEGER,  -- 1, 0, or NULL (no rating)
    session_id TEXT
);
```

### SKILL.md File (filesystem)

Stored at `~/.local/share/gormes/skills/<category>/<name>/SKILL.md`

```yaml
---
name: skill-name
description: Brief description of what this skill does
version: 1.0.0
platforms: [macos, linux]
---

# Skill Title

Full instructions, examples, and context...
```

### ExtractionRequest / ExtractionResult

```go
type ExtractionRequest struct {
    Conversation  []Turn
    Task         string
    Outcome      string
    ToolCalls    []ToolCall
}

type ExtractionResult struct {
    Name        string
    Description string
    Category    string
    Content     string  // Full SKILL.md content
    Confidence  float64 // 0.0–1.0
}

type SkillFeedback struct {
    SkillID   string
    Rating    int        // 1–5
    Comments  string
    SessionID string
}
```

### ToolExecutor Interface

```go
// Defined in internal/tools/executor.go
type ToolExecutor interface {
    Execute(ctx context.Context, req ToolRequest) (<-chan ToolEvent, error)
}

type ToolRequest struct {
    AgentID   string
    ToolName  string
    Input     string
    Metadata  map[string]string
}

type ToolEvent struct {
    Type   string // "started" | "progress" | "output" | "completed" | "failed"
    Chunk  string
    Err    error
}

// In-process implementation (Phase 2.E MVP)
type InProcessToolExecutor struct {
    Registry *ToolRegistry
}
```

---

## Core Interfaces

```go
// SkillStore manages skill persistence
type SkillStore interface {
    Create(ctx context.Context, skill Skill) error
    Get(ctx context.Context, name string) (Skill, error)
    List(ctx context.Context, category string) ([]Skill, error)
    Update(ctx context.Context, skill Skill) error
    Delete(ctx context.Context, name string) error
    IncrementUse(ctx context.Context, name string) error
    RecordFeedback(ctx context.Context, fb SkillFeedback) error
    GetImprovements(ctx context.Context, skillID string) ([]SkillImprovement, error)
}

// SkillExtractor runs complexity detection + pattern extraction via LLM
type SkillExtractor interface {
    IsComplex(ctx context.Context, req ExtractionRequest) (bool, float64, error)
    ExtractPattern(ctx context.Context, req ExtractionRequest) (*ExtractionResult, error)
    Improve(ctx context.Context, skillID string, feedback SkillFeedback) (*ExtractionResult, error)
}

// SkillManager orchestrates the full lifecycle
type SkillManager interface {
    OnTaskComplete(ctx context.Context, req ExtractionRequest) error
    CreateSkill(ctx context.Context, res *ExtractionResult) error
    UseSkill(ctx context.Context, name string) (*Skill, error)
    ImproveSkill(ctx context.Context, fb SkillFeedback) error
    ListSkills(ctx context.Context, category string) ([]Skill, error)
    ViewSkill(ctx context.Context, name string) (*Skill, string, error) // returns skill + SKILL.md content
    DeleteSkill(ctx context.Context, name string) error
}
```

---

## Complexity Detection (IsComplex)

**Trigger:** After every agent turn completes (tool execution finished, response sent).

**Logic:** LLM call with conversation summary:

```
System prompt:
"Given this conversation segment, rate how complex the task was.
Consider: tool call count, error recovery, multi-step reasoning,
novel approach discovery, or user correction.

Return JSON: {"complex": true/false, "confidence": 0.0-1.0, "reason": "..."}

Only mark as complex if this represents a reusable pattern worth
capturing as a skill."

User: [conversation summary]
```

**Thresholds:**
- `complex=true` AND `confidence >= 0.7` → trigger extraction
- `complex=true` AND `confidence < 0.7` → log, no extraction
- `complex=false` → log, no extraction

---

## Pattern Extraction (ExtractPattern)

**Trigger:** After `IsComplex` returns `true` with high confidence.

**Logic:** LLM call to extract reusable pattern:

```
System prompt:
"You are extracting a reusable skill from a successful task execution.
Write a SKILL.md that captures the approach, not the specific outcome.

SKILL.md format:
---
name: slug-name
description: One sentence describing what this skill does
version: 1.0.0
---

# Title

Detailed instructions, examples, and edge cases...
---

Rules:
- Name: lowercase, hyphens, max 64 chars
- Description: max 1024 chars
- Content: max 100,000 chars
- Focus on WHAT to do, not WHAT happened
- Include edge cases and common pitfalls
- No placeholder [TODO] sections"

User: [full conversation + task outcome]
```

---

## Skill Improvement (Improve)

**Trigger:** Two paths:
1. Explicit: user runs `/feedback <skill> <rating>` — `RecordFeedback` stores it
2. Auto (if `skills.auto_improve=true`): agent self-reviews after skill use

**Logic:**
- After `use_count` reaches 5, trigger improvement check
- OR after 3+ negative ratings (helpful < 3)
- LLM compares original skill against recent usage outcomes
- If gaps found, generates improved version

---

## Prompt Integration

Skills are injected into the agent's system prompt via the prompt builder.

**Skills section in system prompt:**
```
## Skills

Before replying, scan the skills below. If a skill matches or is
even partially relevant to your task, you MUST load it with
skill_view(name) and follow its instructions.

<available_skills>
  category-name:
    - skill-name: description
    - another-skill: description
</available_skills>

Only proceed without loading a skill if genuinely none are relevant.
```

**Caching:** Skills list cached; invalidated on skill create/update/delete.

---

## CLI / Slash Commands

| Command | Action |
|---------|--------|
| `/skills` | List all skills (categories) |
| `/skill <name>` | View skill full content |
| `/skill-create <name> <category>` | Create skill interactively |
| `/skill-edit <name>` | Edit skill content |
| `/skill-delete <name>` | Delete skill |
| `/feedback <skill> <1-5>` | Record feedback on a skill |

---

## Configuration

```toml
[skills]
enabled = true
auto_improve = false      # Agent self-review after each use
improvement_threshold = 5 # Trigger improvement check after N uses
extraction_confidence = 0.7
max_skill_content_chars = 100000
skills_dir = "~/.local/share/gormes/skills"
```

---

## File Layout

```
~/.local/share/gormes/skills/
├── <category>/
│   └── <skill-name>/
│       └── SKILL.md
```

Example:
```
~/.local/share/gormes/skills/software-development/
├── fix-git-conflict/
│   └── SKILL.md
└── review-pr/
    └── SKILL.md
```

---

## Gormes SQLite

```
~/.local/share/gormes/memory/memory.db

-- Skills metadata
CREATE TABLE skills (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL,
    category    TEXT NOT NULL DEFAULT 'general',
    version     TEXT NOT NULL DEFAULT '1.0.0',
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL,
    use_count   INTEGER NOT NULL DEFAULT 0,
    last_used_at INTEGER,
    file_path   TEXT NOT NULL
);

CREATE TABLE skill_improvements (
    id               TEXT PRIMARY KEY,
    skill_id         TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    feedback         TEXT NOT NULL,
    improved_content TEXT NOT NULL,
    created_at       INTEGER NOT NULL
);

CREATE TABLE skill_usage (
    skill_id   TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    used_at    INTEGER NOT NULL,
    helpful    INTEGER,
    session_id TEXT
);

-- Indexes
CREATE INDEX idx_skills_category ON skills(category);
CREATE INDEX idx_skill_usage_skill_id ON skill_usage(skill_id);
```

---

## Error Handling

| Error | Behavior |
|-------|----------|
| LLM extraction fails | Log error, skip skill creation, do not interrupt agent |
| Skill file write fails | Return error to agent, retry with exponential backoff |
| Skill name collision | Return error with existing skill info |
| Invalid SKILL.md format | Validation error before save |
| LLM improvement fails | Log, keep original skill, retry next use |

---

## Acceptance Criteria

1. After a complex task (5+ tool calls or error recovery), agent prompts "Should I save this as a skill?"
2. If user approves, SKILL.md is written to filesystem and metadata stored in SQLite
3. `/skills` lists all skills grouped by category
4. `/skill <name>` shows full SKILL.md content
5. Skill is injected into context when name is mentioned or task matches
6. `/feedback <skill> <rating>` records feedback; after 5 uses with good rating, skill is considered "validated"
7. `skills.auto_improve=true` enables agent self-review after skill use
8. Binary size impact: < 500 KB (SQLite schema + interfaces + LLM prompt templates)

---

## Dependencies

- `internal/memory` — SqliteStore (existing)
- `internal/tools` — ToolRegistry, ToolExecutor interface
- `internal/kernel` — LLM calls (via pybridge or future native)
- `internal/config` — `[skills]` config section

---

## Phase 2.G Ledger

| Subphase | Status | Description |
|----------|--------|-------------|
| 2.G.1 — SkillStore (SQLite + filesystem) | ⏳ planned | CRUD, metadata indexing |
| 2.G.2 — Complexity Detector | ⏳ planned | LLM-based IsComplex |
| 2.G.3 — Pattern Extractor | ⏳ planned | LLM-based ExtractPattern |
| 2.G.4 — SkillManager | ⏳ planned | Orchestrates lifecycle |
| 2.G.5 — Prompt Integration | ⏳ planned | Skills in system prompt |
| 2.G.6 — CLI Commands | ⏳ planned | /skill*, /feedback |
| 2.G.7 — Auto-Improvement | ⏳ planned | LLM-based Improve |

**Ship criterion:** Agent detects complex task → offers to save skill → skill appears in `/skills` list → skill is injected into relevant future contexts → feedback loop closes.
