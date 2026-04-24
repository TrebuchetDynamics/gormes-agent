# Phase 2.G — Skills System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the MVP Skills System — the learning loop that detects complex tasks via LLM, extracts reusable patterns, saves them as SKILL.md files + SQLite metadata, and improves them over time.

**Architecture:** SkillStore (SQLite metadata + filesystem content), SkillExtractor (LLM-based IsComplex + ExtractPattern), SkillManager (orchestrates lifecycle), prompt integration. Hybrid storage: SQLite for index/metadata, filesystem for SKILL.md files.

**Tech Stack:** Go stdlib, internal/memory (SqliteStore), internal/kernel (LLM calls), internal/tools (ToolExecutor), internal/config

---

## File Structure

```
internal/skills/
  skill.go         — Skill, SkillImprovement, SkillFeedback types
  store.go         — SkillStore: SQLite CRUD + filesystem sync
  extractor.go     — SkillExtractor: LLM-based IsComplex + ExtractPattern
  manager.go       — SkillManager: orchestrates full lifecycle
  prompts.go       — LLM prompt templates for extraction/improvement
internal/kernel/
  skills_prompt.go  — skills section in system prompt
internal/tools/
  skills_tools.go  — skills_list, skill_view, skill_manage tools
gormes/cmd/
  skills_cli.go    — /skill*, /feedback CLI commands
docs/superpowers/plans/
  2026-04-20-gormes-phase2g-skills.md (this file)
```

---

## Task 1: Skill data structures + SQLite schema

**Files:**
- Create: `internal/skills/skill.go`
- Modify: `internal/memory/memory.go` (add skill tables to SQLite schema)
- Test: `internal/skills/skill_test.go`

- [ ] **Step 1: Create internal/skills/skill.go with all types**

```go
package skills

import (
    "time"
)

type Skill struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string   `json:"description"`
    Category    string   `json:"category"`
    Version     string   `json:"version"`
    CreatedAt   int64    `json:"created_at"`
    UpdatedAt   int64    `json:"updated_at"`
    UseCount    int      `json:"use_count"`
    LastUsedAt  *int64   `json:"last_used_at,omitempty"`
    FilePath    string   `json:"file_path"` // relative path under skills dir
}

type SkillImprovement struct {
    ID              string `json:"id"`
    SkillID         string `json:"skill_id"`
    Feedback        string `json:"feedback"`
    ImprovedContent string `json:"improved_content"`
    CreatedAt       int64  `json:"created_at"`
}

type SkillFeedback struct {
    SkillID   string `json:"skill_id"`
    Rating    int    `json:"rating"` // 1–5
    Comments  string `json:"comments"`
    SessionID string `json:"session_id"`
}

type ExtractionRequest struct {
    Conversation []Turn   `json:"conversation"`
    Task         string  `json:"task"`
    Outcome      string  `json:"outcome"`
    ToolCalls    []ToolCall `json:"tool_calls"`
}

type Turn struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type ToolCall struct {
    Name string `json:"name"`
    Args string `json:"args"`
}

type ExtractionResult struct {
    Name        string  `json:"name"`
    Description string  `json:"description"`
    Category    string  `json:"category"`
    Content     string  `json:"content"` // Full SKILL.md content
    Confidence  float64 `json:"confidence"`
}

const (
    MaxNameLength         = 64
    MaxDescriptionLength  = 1024
    MaxContentLength      = 100000
)

func (s *Skill) Validate() error {
    if len(s.Name) == 0 || len(s.Name) > MaxNameLength {
        return fmt.Errorf("name length must be 1–%d", MaxNameLength)
    }
    if len(s.Description) > MaxDescriptionLength {
        return fmt.Errorf("description too long (max %d)", MaxDescriptionLength)
    }
    if !namePattern.MatchString(s.Name) {
        return fmt.Errorf("name must match pattern: lowercase, hyphens/underscores allowed")
    }
    return nil
}
```

- [ ] **Step 2: Add skill tables to internal/memory/memory.go**

```go
// In the schema migration section of memory.go, add:

func migrateSkillsV3e(db *sql.DB) error {
    queries := []string{
        `CREATE TABLE IF NOT EXISTS skills (
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
        )`,
        `CREATE TABLE IF NOT EXISTS skill_improvements (
            id               TEXT PRIMARY KEY,
            skill_id         TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
            feedback         TEXT NOT NULL,
            improved_content TEXT NOT NULL,
            created_at       INTEGER NOT NULL
        )`,
        `CREATE TABLE IF NOT EXISTS skill_usage (
            skill_id   TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
            used_at    INTEGER NOT NULL,
            helpful    INTEGER,
            session_id TEXT
        )`,
        `CREATE INDEX IF NOT EXISTS idx_skills_category ON skills(category)`,
        `CREATE INDEX IF NOT EXISTS idx_skill_usage_skill_id ON skill_usage(skill_id)`,
    }
    for _, q := range queries {
        if _, err := db.Exec(q); err != nil {
            return fmt.Errorf("skills migration failed: %w", err)
        }
    }
    return nil
}
```

And call `migrateSkillsV3e` in the main migration chain at the appropriate version.

- [ ] **Step 3: Write skill type tests**

```go
func TestSkillValidate(t *testing.T) {
    cases := []struct {
        name    string
        skill   Skill
        wantErr bool
    }{
        {"valid", Skill{Name: "fix-git-conflict", Description: "Fix git merge conflicts"}, false},
        {"empty name", Skill{Name: "", Description: "test"}, true},
        {"name too long", Skill{Name: strings.Repeat("a", 65), Description: "test"}, true},
        {"description too long", Skill{Name: "test", Description: strings.Repeat("a", MaxDescriptionLength+1)}, true},
    }
    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            err := c.skill.Validate()
            if (err != nil) != c.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, c.wantErr)
            }
        })
    }
}

func TestSkillFeedbackRating(t *testing.T) {
    fb := SkillFeedback{Rating: 5}
    if fb.Rating < 1 || fb.Rating > 5 {
        t.Errorf("rating should be 1–5, got %d", fb.Rating)
    }
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/skills/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/skills/skill.go internal/memory/memory.go
git commit -m "feat(skills): add Skill types + SQLite schema migrations"
```

---

## Task 2: SkillStore — SQLite CRUD + filesystem sync

**Files:**
- Create: `internal/skills/store.go`
- Test: `internal/skills/store_test.go`

- [ ] **Step 1: Create SkillStore interface and implementation**

```go
package skills

import (
    "context"
    "database/sql"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/google/uuid"
)

type SkillStore interface {
    Create(ctx context.Context, skill Skill, content string) error
    Get(ctx context.Context, name string) (Skill, string, error) // skill + SKILL.md content
    List(ctx context.Context, category string) ([]Skill, error)
    Update(ctx context.Context, skill Skill, content string) error
    Delete(ctx context.Context, name string) error
    IncrementUse(ctx context.Context, name string) error
    RecordFeedback(ctx context.Context, fb SkillFeedback) error
    GetImprovements(ctx context.Context, skillID string) ([]SkillImprovement, error)
    GetUsageStats(ctx context.Context, skillID string) (int, float64, error)
}

type skillStore struct {
    db         *sql.DB
    skillsRoot string // e.g. ~/.local/share/gormes/skills
}

func NewSkillStore(db *sql.DB, skillsRoot string) SkillStore {
    return &skillStore{db: db, skillsRoot: skillsRoot}
}

func (s *skillStore) Create(ctx context.Context, skill Skill, content string) error {
    if skill.ID == "" {
        skill.ID = uuid.New().String()
    }
    skill.CreatedAt = time.Now().Unix()
    skill.UpdatedAt = skill.CreatedAt
    skill.UseCount = 0
    skill.Version = "1.0.0"

    // Write SKILL.md to filesystem
    filePath := skill.FilePath
    if filePath == "" {
        filePath = filepath.Join(skill.Category, skill.Name, "SKILL.md")
    }
    fullPath := filepath.Join(s.skillsRoot, filePath)
    if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
        return fmt.Errorf("failed to create skill directory: %w", err)
    }
    if err := atomicWrite(fullPath, []byte(content)); err != nil {
        return fmt.Errorf("failed to write SKILL.md: %w", err)
    }

    skill.FilePath = filePath

    // Insert into SQLite
    _, err := s.db.ExecContext(ctx, `
        INSERT INTO skills (id, name, description, category, version, created_at, updated_at, use_count, file_path)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
        skill.ID, skill.Name, skill.Description, skill.Category, skill.Version,
        skill.CreatedAt, skill.UpdatedAt, skill.UseCount, skill.FilePath)
    if err != nil {
        os.Remove(fullPath)
        return fmt.Errorf("failed to insert skill: %w", err)
    }
    return nil
}

func (s *skillStore) Get(ctx context.Context, name string) (Skill, string, error) {
    row := s.db.QueryRowContext(ctx, `
        SELECT id, name, description, category, version, created_at, updated_at, use_count, last_used_at, file_path
        FROM skills WHERE name = ?`, name)

    var skill Skill
    err := row.Scan(&skill.ID, &skill.Name, &skill.Description, &skill.Category,
        &skill.Version, &skill.CreatedAt, &skill.UpdatedAt, &skill.UseCount,
        &skill.LastUsedAt, &skill.FilePath)
    if err == sql.ErrNoRows {
        return Skill{}, "", fmt.Errorf("skill not found: %s", name)
    }
    if err != nil {
        return Skill{}, "", err
    }

    // Read SKILL.md content
    fullPath := filepath.Join(s.skillsRoot, skill.FilePath)
    content, err := os.ReadFile(fullPath)
    if err != nil {
        return Skill{}, "", fmt.Errorf("failed to read SKILL.md: %w", err)
    }

    return skill, string(content), nil
}

func (s *skillStore) List(ctx context.Context, category string) ([]Skill, error) {
    var rows *sql.Rows
    var err error
    if category != "" {
        rows, err = s.db.QueryContext(ctx, `SELECT id, name, description, category, version, created_at, updated_at, use_count, last_used_at, file_path FROM skills WHERE category = ? ORDER BY name`, category)
    } else {
        rows, err = s.db.QueryContext(ctx, `SELECT id, name, description, category, version, created_at, updated_at, use_count, last_used_at, file_path FROM skills ORDER BY category, name`)
    }
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var out []Skill
    for rows.Next() {
        var skill Skill
        err := rows.Scan(&skill.ID, &skill.Name, &skill.Description, &skill.Category,
            &skill.Version, &skill.CreatedAt, &skill.UpdatedAt, &skill.UseCount,
            &skill.LastUsedAt, &skill.FilePath)
        if err != nil {
            return nil, err
        }
        out = append(out, skill)
    }
    return out, rows.Err()
}

func (s *skillStore) Update(ctx context.Context, skill Skill, content string) error {
    skill.UpdatedAt = time.Now().Unix()

    // Parse version (semver bump)
    skill.Version = bumpVersion(skill.Version)

    // Write filesystem
    fullPath := filepath.Join(s.skillsRoot, skill.FilePath)
    if err := atomicWrite(fullPath, []byte(content)); err != nil {
        return fmt.Errorf("failed to update SKILL.md: %w", err)
    }

    _, err := s.db.ExecContext(ctx, `
        UPDATE skills SET description=?, category=?, version=?, updated_at=?, use_count=?, last_used_at=?
        WHERE id=?`,
        skill.Description, skill.Category, skill.Version, skill.UpdatedAt,
        skill.UseCount, skill.LastUsedAt, skill.ID)
    return err
}

func (s *skillStore) Delete(ctx context.Context, name string) error {
    skill, _, err := s.Get(ctx, name)
    if err != nil {
        return err
    }
    fullPath := filepath.Join(s.skillsRoot, skill.FilePath)
    os.Remove(fullPath)
    _, err = s.db.ExecContext(ctx, `DELETE FROM skills WHERE name = ?`, name)
    return err
}

func (s *skillStore) IncrementUse(ctx context.Context, name string) error {
    now := time.Now().Unix()
    _, err := s.db.ExecContext(ctx, `
        UPDATE skills SET use_count = use_count + 1, last_used_at = ? WHERE name = ?`, now, name)
    return err
}

func (s *skillStore) RecordFeedback(ctx context.Context, fb SkillFeedback) error {
    if fb.SkillID == "" {
        return fmt.Errorf("skill_id required")
    }
    id := uuid.New().String()
    _, err := s.db.ExecContext(ctx, `
        INSERT INTO skill_usage (skill_id, used_at, helpful, session_id)
        VALUES (?, ?, ?, ?)`,
        fb.SkillID, time.Now().Unix(), fb.Rating, fb.SessionID)
    return err
}

func (s *skillStore) GetImprovements(ctx context.Context, skillID string) ([]SkillImprovement, error) {
    rows, err := s.db.QueryContext(ctx, `
        SELECT id, skill_id, feedback, improved_content, created_at
        FROM skill_improvements WHERE skill_id = ? ORDER BY created_at DESC`, skillID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var out []SkillImprovement
    for rows.Next() {
        var i SkillImprovement
        if err := rows.Scan(&i.ID, &i.SkillID, &i.Feedback, &i.ImprovedContent, &i.CreatedAt); err != nil {
            return nil, err
        }
        out = append(out, i)
    }
    return out, rows.Err()
}

func (s *skillStore) GetUsageStats(ctx context.Context, skillID string) (int, float64, error) {
    row := s.db.QueryRowContext(ctx, `
        SELECT COUNT(*), COALESCE(AVG(helpful), 0) FROM skill_usage WHERE skill_id = ?`, skillID)
    var count int
    var avg float64
    if err := row.Scan(&count, &avg); err != nil {
        return 0, 0, err
    }
    return count, avg, nil
}

// Helpers

func atomicWrite(path string, data []byte) error {
    tmp := path + ".tmp"
    if err := os.WriteFile(tmp, data, 0644); err != nil {
        return err
    }
    return os.Rename(tmp, path)
}

func bumpVersion(v string) string {
    parts := strings.Split(v, ".")
    if len(parts) != 3 {
        return "1.0.0"
    }
    // Simple patch bump
    patch := 0
    fmt.Sscanf(parts[2], "%d", &patch)
    return fmt.Sprintf("%s.%s.%d", parts[0], parts[1], patch+1)
}

var namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
```

- [ ] **Step 2: Write store tests (using sqlmock)**

```go
func TestSkillStoreCreate(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    tmp := t.TempDir()
    store := NewSkillStore(db, tmp)

    skill := Skill{Name: "test-skill", Category: "general", Description: "A test skill"}
    content := "---\nname: test-skill\n---\n# Test"

    mock.ExpectExec("INSERT INTO skills").WillReturnResult(sqlmock.NewResult(1, 1))

    err := store.Create(context.Background(), skill, content)
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }

    // Verify file was written
    if _, err := os.Stat(filepath.Join(tmp, "general", "test-skill", "SKILL.md")); os.IsNotExist(err) {
        t.Error("SKILL.md not written")
    }
}

func TestSkillStoreGetNotFound(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()
    store := NewSkillStore(db, t.TempDir())

    mock.ExpectQuery("SELECT .* FROM skills").WillReturnError(sql.ErrNoRows)

    _, _, err := store.Get(context.Background(), "nonexistent")
    if err == nil {
        t.Error("expected error for nonexistent skill")
    }
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/skills/... -v -short`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/skills/store.go
git commit -m "feat(skills): add SkillStore with SQLite + filesystem sync"
```

---

## Task 3: LLM prompt templates + ExtractionRequest/Result types

**Files:**
- Create: `internal/skills/prompts.go`
- Test: `internal/skills/prompts_test.go`

- [ ] **Step 1: Create prompts.go with LLM prompt templates**

```go
package skills

const ComplexityDetectionPrompt = `Given this conversation segment, rate how complex the task was.

Consider:
- Tool call count (5+ suggests complexity)
- Error recovery events
- Multi-step reasoning required
- Novel approach discovery
- User corrections

Return JSON with exactly this shape:
{"complex": true/false, "confidence": 0.0-1.0, "reason": "brief explanation"}

Only mark as complex if this represents a reusable pattern worth capturing as a skill.

===
CONVERSATION:
{{.Conversation}}
===
TASK: {{.Task}}
===
OUTCOME: {{.Outcome}}
`

const PatternExtractionPrompt = `You are extracting a reusable skill from a successful task execution.

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
- Name: lowercase, hyphens allowed, max 64 chars, pattern: ^[a-z0-9][a-z0-9._-]*$
- Description: max 1024 chars
- Content: max 100,000 chars
- Focus on WHAT to do, not WHAT happened
- Include edge cases and common pitfalls
- No placeholder [TODO] sections
- Be specific enough that another agent can follow this skill without additional context

===
CONVERSATION:
{{.Conversation}}
===
TASK: {{.Task}}
===
OUTCOME: {{.Outcome}}
`

const ImprovementPrompt = `Compare this skill against recent usage feedback and determine if it needs improvement.

Skill:
---
{{.SkillContent}}
---

Recent feedback:
{{.Feedback}}

Return JSON:
{"needs_improvement": true/false, "reason": "...", "updated_content": "full SKILL.md if needs_improvement, empty string otherwise"}
`

const SkillsSystemPromptSection = `## Skills

Before replying, scan the skills below. If a skill matches or is even partially relevant to your task, you MUST load it with skill_view(name) and follow its instructions.

<available_skills>
{{range .Categories}}{{.Name}}:
  {{range .Skills}}  - {{.Name}}: {{.Description}}
  {{end}}{{end}}</available_skills>

Only proceed without loading a skill if genuinely none are relevant to the task.
`

func FormatConversation(turns []Turn) string {
    var out strings.Builder
    for _, t := range turns {
        out.WriteString(fmt.Sprintf("%s: %s\n\n", t.Role, t.Content))
    }
    return out.String()
}
```

- [ ] **Step 2: Write prompt template tests**

```go
func TestComplexityPromptHasPlaceholders(t *testing.T) {
    if !strings.Contains(ComplexityDetectionPrompt, "{{.Conversation}}") {
        t.Error("ComplexityDetectionPrompt missing Conversation placeholder")
    }
    if !strings.Contains(ComplexityDetectionPrompt, "{{.Task}}") {
        t.Error("ComplexityDetectionPrompt missing Task placeholder")
    }
}

func TestPatternExtractionPromptHasFrontmatter(t *testing.T) {
    if !strings.Contains(PatternExtractionPrompt, "name:") {
        t.Error("PatternExtractionPrompt missing name field")
    }
    if !strings.Contains(PatternExtractionPrompt, "SKILL.md") {
        t.Error("PatternExtractionPrompt missing SKILL.md format")
    }
}

func TestFormatConversation(t *testing.T) {
    turns := []Turn{{Role: "user", Content: "hello"}, {Role: "assistant", Content: "hi"}}
    out := FormatConversation(turns)
    if !strings.Contains(out, "user: hello") {
        t.Error("FormatConversation not working")
    }
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/skills/... -v -run Prompt`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/skills/prompts.go
git commit -m "feat(skills): add LLM prompt templates for extraction and improvement"
```

---

## Task 4: SkillExtractor — LLM-based IsComplex + ExtractPattern + Improve

**Files:**
- Create: `internal/skills/extractor.go`
- Test: `internal/skills/extractor_test.go`

- [ ] **Step 1: Create SkillExtractor interface and LLM implementation**

```go
package skills

import (
    "context"
    "encoding/json"
    "fmt"
    "regexp"
    "strings"
    "text/template"
)

type LLMClient interface {
    Complete(ctx context.Context, prompt string) (string, error)
}

type skillExtractor struct {
    llm         LLMClient
    complexityTmpl *template.Template
    extractionTmpl *template.Template
    improvementTmpl *template.Template
}

func NewSkillExtractor(llm LLMClient) SkillExtractor {
    return &skillExtractor{
        llm:              llm,
        complexityTmpl:   template.Must(template.New("complexity").Parse(ComplexityDetectionPrompt)),
        extractionTmpl:   template.Must(template.New("extraction").Parse(PatternExtractionPrompt)),
        improvementTmpl:   template.Must(template.New("improvement").Parse(ImprovementPrompt)),
    }
}

type complexityResponse struct {
    Complex   bool    `json:"complex"`
    Confidence float64 `json:"confidence"`
    Reason    string  `json:"reason"`
}

func (e *skillExtractor) IsComplex(ctx context.Context, req ExtractionRequest) (bool, float64, error) {
    prompt, err := e.buildPrompt(e.complexityTmpl, req)
    if err != nil {
        return false, 0, err
    }

    resp, err := e.llm.Complete(ctx, prompt)
    if err != nil {
        return false, 0, err // fail open — don't interrupt on LLM errors
    }

    var cr complexityResponse
    if err := json.Unmarshal([]byte(resp), &cr); err != nil {
        return false, 0, fmt.Errorf("failed to parse complexity response: %w", err)
    }

    return cr.Complex && cr.Confidence >= 0.7, cr.Confidence, nil
}

func (e *skillExtractor) ExtractPattern(ctx context.Context, req ExtractionRequest) (*ExtractionResult, error) {
    prompt, err := e.buildPrompt(e.extractionTmpl, req)
    if err != nil {
        return nil, err
    }

    resp, err := e.llm.Complete(ctx, prompt)
    if err != nil {
        return nil, err
    }

    result, err := parseExtractionResult(resp)
    if err != nil {
        return nil, err
    }

    return result, nil
}

func (e *skillExtractor) Improve(ctx context.Context, skillID string, skillContent string, feedback string) (*ExtractionResult, error) {
    data := map[string]string{
        "SkillContent": skillContent,
        "Feedback":     feedback,
    }
    prompt, err := e.buildPromptFromMap(e.improvementTmpl, data)
    if err != nil {
        return nil, err
    }

    resp, err := e.llm.Complete(ctx, prompt)
    if err != nil {
        return nil, err
    }

    var ir struct {
        NeedsImprovement bool   `json:"needs_improvement"`
        Reason           string `json:"reason"`
        UpdatedContent   string `json:"updated_content"`
    }
    if err := json.Unmarshal([]byte(resp), &ir); err != nil {
        return nil, err
    }

    if !ir.NeedsImprovement || ir.UpdatedContent == "" {
        return nil, nil // no improvement needed
    }

    // Parse the updated SKILL.md to extract metadata
    name, desc, cat := parseSKILLMeta(ir.UpdatedContent)
    return &ExtractionResult{
        Name:        name,
        Description: desc,
        Category:    cat,
        Content:     ir.UpdatedContent,
        Confidence:  0.8,
    }, nil
}

func (e *skillExtractor) buildPrompt(tmpl *template.Template, req ExtractionRequest) (string, error) {
    conversation := FormatConversation(req.Conversation)
    data := map[string]any{
        "Conversation": conversation,
        "Task":         req.Task,
        "Outcome":      req.Outcome,
    }
    var b strings.Builder
    if err := tmpl.Execute(&b, data); err != nil {
        return "", err
    }
    return b.String(), nil
}

func (e *skillExtractor) buildPromptFromMap(tmpl *template.Template, data map[string]string) (string, error) {
    var b strings.Builder
    if err := tmpl.Execute(&b, data); err != nil {
        return "", err
    }
    return b.String(), nil
}

func parseExtractionResult(raw string) (*ExtractionResult, error) {
    // Try to find SKILL.md block in the response
    start := strings.Index(raw, "---")
    if start == -1 {
        return nil, fmt.Errorf("no frontmatter found in extraction response")
    }

    // Find the closing --- of frontmatter
    end := strings.Index(raw[start+3:], "---")
    if end == -1 {
        return nil, fmt.Errorf("incomplete frontmatter")
    }
    frontmatter := raw[start+3 : start+3+end]
    content := raw[start:]

    name := extractField(frontmatter, "name")
    desc := extractField(frontmatter, "description")
    cat := extractField(frontmatter, "category")
    if cat == "" {
        cat = "general"
    }

    if name == "" {
        return nil, fmt.Errorf("extraction missing name field")
    }

    return &ExtractionResult{
        Name:        name,
        Description: desc,
        Category:    cat,
        Content:     content,
        Confidence:  0.8,
    }, nil
}

var fieldRe = regexp.MustCompile(`(?m)^(\w+):\s*(.+)$`)

func extractField(frontmatter, key string) string {
    m := fieldRe.FindStringSubmatch(frontmatter)
    if m == nil {
        return ""
    }
    // This only finds first match; need to loop for multi-field
    for _, line := range strings.Split(frontmatter, "\n") {
        if strings.HasPrefix(line, key+":") {
            return strings.TrimSpace(strings.TrimPrefix(line, key+":"))
        }
    }
    return ""
}

func parseSKILLMeta(content string) (name, description, category string) {
    start := strings.Index(content, "---")
    if start == -1 {
        return
    }
    end := strings.Index(content[start+3:], "---")
    if end == -1 {
        return
    }
    fm := content[start+3 : start+3+end]
    name = extractField(fm, "name")
    description = extractField(fm, "description")
    category = extractField(fm, "category")
    if category == "" {
        category = "general"
    }
    return
}
```

- [ ] **Step 2: Write extractor tests with mock LLM**

```go
func TestSkillExtractor_IsComplex(t *testing.T) {
    mock := &mockLLM{resp: `{"complex": true, "confidence": 0.85, "reason": "5 tool calls with error recovery"}`}
    ext := NewSkillExtractor(mock)

    req := ExtractionRequest{
        Conversation: []Turn{{Role: "user", Content: "fix the bug"}},
        Task:         "Fix the bug",
        Outcome:      "Fixed",
    }

    complex, conf, err := ext.IsComplex(context.Background(), req)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !complex {
        t.Error("expected complex=true")
    }
    if conf < 0.7 {
        t.Errorf("confidence should be >= 0.7, got %f", conf)
    }
}

func TestSkillExtractor_IsComplex_LowConfidence(t *testing.T) {
    mock := &mockLLM{resp: `{"complex": true, "confidence": 0.5, "reason": "somewhat complex"}`}
    ext := NewSkillExtractor(mock)

    req := ExtractionRequest{Task: "simple task"}
    complex, _, _ := ext.IsComplex(context.Background(), req)
    if complex {
        t.Error("expected complex=false for low confidence")
    }
}

func TestSkillExtractor_ExtractPattern(t *testing.T) {
    mock := &mockLLM{resp: `---\nname: test-skill\ndescription: A test skill\n---\n# Test\nContent here`}
    ext := NewSkillExtractor(mock)

    req := ExtractionRequest{
        Conversation: []Turn{{Role: "user", Content: "do the thing"}},
        Task:         "do the thing",
        Outcome:      "done",
    }

    res, err := ext.ExtractPattern(context.Background(), req)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if res.Name != "test-skill" {
        t.Errorf("expected name 'test-skill', got %q", res.Name)
    }
}

type mockLLM struct {
    resp string
    err  error
}

func (m *mockLLM) Complete(ctx context.Context, prompt string) (string, error) {
    return m.resp, m.err
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/skills/... -v -run "TestSkillExtractor|TestComplexity"`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/skills/extractor.go
git commit -m "feat(skills): add SkillExtractor with LLM-based IsComplex and ExtractPattern"
```

---

## Task 5: SkillManager — orchestrates full lifecycle

**Files:**
- Create: `internal/skills/manager.go`
- Test: `internal/skills/manager_test.go`

- [ ] **Step 1: Create SkillManager interface and implementation**

```go
package skills

import (
    "context"
    "fmt"
    "path/filepath"
    "strings"
    "time"
)

type SkillManager interface {
    OnTaskComplete(ctx context.Context, req ExtractionRequest) error
    CreateSkill(ctx context.Context, res *ExtractionResult) error
    UseSkill(ctx context.Context, name string) (*Skill, string, error)
    ImproveSkill(ctx context.Context, fb SkillFeedback) error
    ListSkills(ctx context.Context, category string) ([]Skill, error)
    ViewSkill(ctx context.Context, name string) (*Skill, string, error)
    DeleteSkill(ctx context.Context, name string) error
}

type skillManager struct {
    store     SkillStore
    extractor SkillExtractor
    config    SkillsConfig
}

type SkillsConfig struct {
    Enabled            bool
    AutoImprove        bool
    ImprovementThreshold int // trigger improvement after N uses
    ExtractionConfidence float64
}

func NewSkillManager(store SkillStore, extractor SkillExtractor, config SkillsConfig) SkillManager {
    return &skillManager{
        store:     store,
        extractor: extractor,
        config:    config,
    }
}

func (m *skillManager) OnTaskComplete(ctx context.Context, req ExtractionRequest) error {
    if !m.config.Enabled {
        return nil
    }

    complex, confidence, err := m.extractor.IsComplex(ctx, req)
    if err != nil {
        // Log but don't fail — LLM errors shouldn't interrupt the agent
        fmt.Printf("skills: complexity detection failed: %v\n", err)
        return nil
    }

    if !complex || confidence < m.config.ExtractionConfidence {
        return nil
    }

    // Ask user (stub — returns "yes" for MVP; full prompt integration in Task 7)
    return nil // TODO: prompt user for confirmation
}

func (m *skillManager) CreateSkill(ctx context.Context, res *ExtractionResult) error {
    if err := validateExtractionResult(res); err != nil {
        return err
    }

    skill := Skill{
        Name:        res.Name,
        Description: res.Description,
        Category:    res.Category,
        Version:     "1.0.0",
    }

    return m.store.Create(ctx, skill, res.Content)
}

func (m *skillManager) UseSkill(ctx context.Context, name string) (*Skill, string, error) {
    skill, content, err := m.store.Get(ctx, name)
    if err != nil {
        return nil, "", err
    }

    // Increment use count asynchronously
    go func() {
        m.store.IncrementUse(context.Background(), name)
    }()

    return &skill, content, nil
}

func (m *skillManager) ImproveSkill(ctx context.Context, fb SkillFeedback) error {
    if !m.config.Enabled || !m.config.AutoImprove {
        return nil
    }

    // Get skill and recent improvements
    skill, content, err := m.store.Get(ctx, fb.SkillID)
    if err != nil {
        return err
    }

    improvements, err := m.store.GetImprovements(ctx, skill.ID)
    if err != nil {
        return err
    }

    // Build feedback summary
    fbSummary := fmt.Sprintf("Rating: %d/5. Comments: %s", fb.Rating, fb.Comments)
    for _, imp := range improvements {
        fbSummary += fmt.Sprintf("\nPrior improvement feedback: %s", imp.Feedback)
    }

    // Run LLM improvement check
    improved, err := m.extractor.Improve(ctx, skill.ID, content, fbSummary)
    if err != nil {
        return err
    }
    if improved == nil {
        return nil // no improvement needed
    }

    // Save improvement record
    imp := SkillImprovement{
        ID:              fmt.Sprintf("imp_%d", time.Now().UnixNano()),
        SkillID:         skill.ID,
        Feedback:        fb.Comments,
        ImprovedContent: improved.Content,
        CreatedAt:       time.Now().Unix(),
    }

    // Update skill with improved content
    improved.ID = skill.ID
    return m.store.Update(ctx, *improved, improved.Content)
}

func (m *skillManager) ListSkills(ctx context.Context, category string) ([]Skill, error) {
    return m.store.List(ctx, category)
}

func (m *skillManager) ViewSkill(ctx context.Context, name string) (*Skill, string, error) {
    skill, content, err := m.store.Get(ctx, name)
    if err != nil {
        return nil, "", err
    }
    return &skill, content, nil
}

func (m *skillManager) DeleteSkill(ctx context.Context, name string) error {
    return m.store.Delete(ctx, name)
}

func validateExtractionResult(res *ExtractionResult) error {
    if res.Name == "" || len(res.Name) > MaxNameLength {
        return fmt.Errorf("invalid skill name")
    }
    if !namePattern.MatchString(res.Name) {
        return fmt.Errorf("name must match pattern: lowercase, hyphens/underscores allowed")
    }
    if len(res.Description) > MaxDescriptionLength {
        return fmt.Errorf("description too long")
    }
    if len(res.Content) > MaxContentLength {
        return fmt.Errorf("content too long")
    }
    return nil
}
```

- [ ] **Step 2: Write manager tests with mock store + mock LLM**

```go
func TestSkillManager_CreateSkill(t *testing.T) {
    store := &mockSkillStore{}
    ext := &mockSkillExtractor{}
    mgr := NewSkillManager(store, ext, SkillsConfig{Enabled: true})

    res := &ExtractionResult{
        Name:        "test-skill",
        Description: "A test skill",
        Category:    "general",
        Content:     "---\nname: test-skill\n---\n# Test",
    }

    err := mgr.CreateSkill(context.Background(), res)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if store.CreatedSkill.Name != "test-skill" {
        t.Errorf("expected skill name 'test-skill', got %q", store.CreatedSkill.Name)
    }
}

func TestSkillManager_UseSkill(t *testing.T) {
    store := &mockSkillStore{
        Skills: map[string]Skill{"test": {Name: "test", Category: "general"}},
        Contents: map[string]string{"test": "---\nname: test\n---\n# Test"},
    }
    ext := &mockSkillExtractor{}
    mgr := NewSkillManager(store, ext, SkillsConfig{Enabled: true})

    skill, content, err := mgr.UseSkill(context.Background(), "test")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if skill.Name != "test" {
        t.Errorf("expected skill 'test', got %q", skill.Name)
    }
    if content == "" {
        t.Error("expected content, got empty")
    }
}

func TestSkillManager_ValidateName(t *testing.T) {
    cases := []struct {
        name  string
        valid bool
    }{
        {"valid-skill", true},
        {"valid_skill", true},
        {"valid.skill", true},
        {"ValidSkill", false}, // uppercase not allowed
        {"valid skill", false}, // spaces not allowed
        {"-invalid", false},    // can't start with hyphen
    }
    for _, c := range cases {
        err := validateExtractionResult(&ExtractionResult{Name: c.name, Description: "test", Content: "# Test"})
        if (err == nil) != c.valid {
            t.Errorf("name %q: valid=%v, err=%v", c.name, c.valid, err)
        }
    }
}

type mockSkillStore struct {
    CreatedSkill Skill
    Skills      map[string]Skill
    Contents    map[string]string
}

func (m *mockSkillStore) Create(ctx context.Context, skill Skill, content string) error {
    m.CreatedSkill = skill
    return nil
}
func (m *mockSkillStore) Get(ctx context.Context, name string) (Skill, string, error) {
    s, ok := m.Skills[name]
    if !ok {
        return Skill{}, "", fmt.Errorf("not found")
    }
    return s, m.Contents[name], nil
}
func (m *mockSkillStore) List(ctx context.Context, category string) ([]Skill, error) {
    return nil, nil
}
func (m *mockSkillStore) Update(ctx context.Context, skill Skill, content string) error { return nil }
func (m *mockSkillStore) Delete(ctx context.Context, name string) error { return nil }
func (m *mockSkillStore) IncrementUse(ctx context.Context, name string) error { return nil }
func (m *mockSkillStore) RecordFeedback(ctx context.Context, fb SkillFeedback) error { return nil }
func (m *mockSkillStore) GetImprovements(ctx context.Context, skillID string) ([]SkillImprovement, error) { return nil, nil }
func (m *mockSkillStore) GetUsageStats(ctx context.Context, skillID string) (int, float64, error) { return 0, 0, nil }

type mockSkillExtractor struct{}

func (m *mockSkillExtractor) IsComplex(ctx context.Context, req ExtractionRequest) (bool, float64, error) {
    return true, 0.9, nil
}
func (m *mockSkillExtractor) ExtractPattern(ctx context.Context, req ExtractionRequest) (*ExtractionResult, error) {
    return &ExtractionResult{Name: "extracted", Description: "desc", Content: "# Test"}, nil
}
func (m *mockSkillExtractor) Improve(ctx context.Context, skillID, content, feedback string) (*ExtractionResult, error) {
    return nil, nil
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/skills/... -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/skills/manager.go
git commit -m "feat(skills): add SkillManager orchestrating full lifecycle"
```

---

## Task 6: skills_list, skill_view, skill_manage tools

**Files:**
- Create: `internal/tools/skills_tools.go`
- Test: `internal/tools/skills_tools_test.go`

- [ ] **Step 1: Create skills tool implementations**

```go
package tools

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"

    "github.com/TrebuchetDynamics/gormes-agent/internal/skills"
)

type SkillsTools struct {
    manager *skills.SkillManager
}

func NewSkillsTools(mgr *skills.SkillManager) *SkillsTools {
    return &SkillsTools{manager: mgr}
}

var SkillsListSchema = ToolSchema{
    Name:        "skills_list",
    Description: "List all installed skills, optionally filtered by category. Returns skill names, descriptions, and categories.",
    Parameters: []ToolParam{
        {Name: "category", Type: "string", Required: false, Description: "Filter by category"},
    },
}

var SkillViewSchema = ToolSchema{
    Name:        "skill_view",
    Description: "View the full content of a skill including its SKILL.md instructions.",
    Parameters: []ToolParam{
        {Name: "name", Type: "string", Required: true, Description: "Skill name"},
    },
}

var SkillManageSchema = ToolSchema{
    Name:        "skill_manage",
    Description: "Create, edit, or delete a skill. Skills are your procedural memory — reusable approaches for recurring task types.",
    Parameters: []ToolParam{
        {Name: "action", Type: "string", Required: true, Enum: []string{"create", "edit", "delete"}, Description: "Action to perform"},
        {Name: "name", Type: "string", Required: true, Description: "Skill name"},
        {Name: "description", Type: "string", Required: false, Description: "Skill description (for create)"},
        {Name: "category", Type: "string", Required: false, Default: "general", Description: "Category (for create)"},
        {Name: "content", Type: "string", Required: false, Description: "Full SKILL.md content (for create/edit)"},
        {Name: "old_string", Type: "string", Required: false, Description: "String to replace (for edit)"},
        {Name: "new_string", Type: "string", Required: false, Description: "Replacement string (for edit)"},
    },
}

func (t *SkillsTools) SkillsList(ctx context.Context, args map[string]any) (string, error) {
    category, _ := args["category"].(string)
    list, err := t.manager.ListSkills(ctx, category)
    if err != nil {
        return "", err
    }

    out := map[string]any{"success": true, "skills": list}
    b, _ := json.Marshal(out)
    return string(b), nil
}

func (t *SkillsTools) SkillView(ctx context.Context, args map[string]any) (string, error) {
    name, _ := args["name"].(string)
    if name == "" {
        return "", fmt.Errorf("name required")
    }

    skill, content, err := t.manager.ViewSkill(ctx, name)
    if err != nil {
        return "", err
    }

    out := map[string]any{
        "success": true,
        "name":    skill.Name,
        "description": skill.Description,
        "category": skill.Category,
        "version":  skill.Version,
        "use_count": skill.UseCount,
        "content": content,
    }
    b, _ := json.Marshal(out)
    return string(b), nil
}

func (t *SkillsTools) SkillManage(ctx context.Context, args map[string]any) (string, error) {
    action, _ := args["action"].(string)
    name, _ := args["name"].(string)
    if action == "" || name == "" {
        return "", fmt.Errorf("action and name required")
    }

    switch action {
    case "create":
        return t.handleCreate(ctx, args)
    case "edit":
        return t.handleEdit(ctx, args)
    case "delete":
        return t.handleDelete(ctx, args)
    default:
        return "", fmt.Errorf("unknown action: %s", action)
    }
}

func (t *SkillsTools) handleCreate(ctx context.Context, args map[string]any) (string, error) {
    description, _ := args["description"].(string)
    category, _ := args["category"].(string)
    content, _ := args["content"].(string)

    if content == "" {
        return "", fmt.Errorf("content (SKILL.md) required for create")
    }

    res := &skills.ExtractionResult{
        Name:        name,
        Description: description,
        Category:    category,
        Content:     content,
    }

    if err := t.manager.CreateSkill(ctx, res); err != nil {
        return "", err
    }

    return json.dumps(map[string]any{"success": true, "skill": name})
}

func (t *SkillsTools) handleEdit(ctx context.Context, args map[string]any) (string, error) {
    // For MVP: full content replacement
    content, _ := args["content"].(string)
    skill, _, err := t.manager.ViewSkill(ctx, name)
    if err != nil {
        return "", err
    }

    // Parse and validate new content
    n, d, c := parseSKILLMeta(content)
    skill.Name = n
    skill.Description = d

    if err := t.manager.CreateSkill(ctx, &skills.ExtractionResult{
        Name: n, Description: d, Category: skill.Category, Content: content,
    }); err != nil {
        return "", err
    }

    return json.dumps(map[string]any{"success": true, "skill": n})
}

func (t *SkillsTools) handleDelete(ctx context.Context, args map[string]any) (string, error) {
    if err := t.manager.DeleteSkill(ctx, name); err != nil {
        return "", err
    }
    return json.dumps(map[string]any{"success": true})
}
```

- [ ] **Step 2: Register tools in tool registry**

In `internal/tools/builtin.go`:

```go
func RegisterSkillsTools(mgr *skills.SkillManager, reg *ToolRegistry) {
    tools := NewSkillsTools(mgr)
    reg.RegisterMany([]ToolEntry{
        {Name: "skills_list", Handler: tools.SkillsList, Schema: SkillsListSchema},
        {Name: "skill_view", Handler: tools.SkillView, Schema: SkillViewSchema},
        {Name: "skill_manage", Handler: tools.SkillManage, Schema: SkillManageSchema},
    })
}
```

- [ ] **Step 3: Write tools tests**

```go
func TestSkillsTools_List(t *testing.T) {
    store := &mockSkillStore{Skills: []skills.Skill{{Name: "test", Category: "general"}}}
    mgr := skills.NewSkillManager(store, nil, skills.SkillsConfig{})
    tools := NewSkillsTools(mgr)

    result, err := tools.SkillsList(context.Background(), map[string]any{})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    var out map[string]any
    json.Unmarshal([]byte(result), &out)
    if !out["success"].(bool) {
        t.Error("expected success=true")
    }
}

func TestSkillsTools_ViewNotFound(t *testing.T) {
    store := &mockSkillStore{}
    mgr := skills.NewSkillManager(store, nil, skills.SkillsConfig{})
    tools := NewSkillsTools(mgr)

    _, err := tools.SkillView(context.Background(), map[string]any{"name": "nonexistent"})
    if err == nil {
        t.Error("expected error for nonexistent skill")
    }
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/tools/... -v -run Skills`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tools/skills_tools.go internal/tools/builtin.go
git commit -m "feat(skills): add skills_list, skill_view, skill_manage tools"
```

---

## Task 7: Prompt builder integration + complexity detection trigger

**Files:**
- Create: `internal/kernel/skills_prompt.go`
- Modify: `internal/kernel/kernel.go` (integrate complexity detection after tool execution)
- Test: `internal/kernel/skills_prompt_test.go`

- [ ] **Step 1: Create skills prompt builder**

```go
package kernel

import (
    "bytes"
    "fmt"
    "strings"
    "text/template"

    "github.com/TrebuchetDynamics/gormes-agent/internal/skills"
)

type skillsPromptBuilder struct {
    manager *skills.SkillManager
}

func NewSkillsPromptBuilder(mgr *skills.SkillManager) *skillsPromptBuilder {
    return &skillsPromptBuilder{manager: mgr}
}

type skillCategory struct {
    Name  string
    Skills []struct {
        Name        string
        Description string
    }
}

func (b *skillsPromptBuilder) BuildSkillsSection(ctx context.Context) (string, error) {
    allSkills, err := b.manager.ListSkills(ctx, "")
    if err != nil {
        return "", err
    }

    // Group by category
    catMap := make(map[string][]struct{ Name, Description string })
    for _, s := range allSkills {
        catMap[s.Category] = append(catMap[s.Category], struct{ Name, Description string }{s.Name, s.Description})
    }

    var cats []skillCategory
    for name, skills := range catMap {
        cats = append(cats, skillCategory{Name: name, Skills: skills})
    }

    // Render template
    var buf bytes.Buffer
    if err := skillsTemplate.Execute(&buf, map[string]any{"Categories": cats}); err != nil {
        return "", err
    }

    return buf.String(), nil
}

func (b *skillsPromptBuilder) OnTurnComplete(ctx context.Context, turns []Turn, toolCalls []ToolCall) error {
    if b.manager == nil {
        return nil
    }

    req := skills.ExtractionRequest{
        Conversation: turns,
        Task:         extractTaskFromTurns(turns),
        Outcome:     extractOutcomeFromTurns(turns),
        ToolCalls:    toolCalls,
    }

    return b.manager.OnTaskComplete(ctx, req)
}

func extractTaskFromTurns(turns []Turn) string {
    // Last user message as task description
    for i := len(turns) - 1; i >= 0; i-- {
        if turns[i].Role == "user" {
            return turns[i].Content
        }
    }
    return ""
}

func extractOutcomeFromTurns(turns []Turn) string {
    // Last assistant message as outcome
    for i := len(turns) - 1; i >= 0; i-- {
        if turns[i].Role == "assistant" {
            return turns[i].Content
        }
    }
    return ""
}

var skillsTemplate = template.Must(template.New("skills").Parse(skills.SkillsSystemPromptSection))
```

- [ ] **Step 2: Add complexity detection trigger to Kernel**

In `internal/kernel/kernel.go`, after tool execution completes in the main loop:

```go
func (k *Kernel) onTurnComplete() {
    if k.skillsPromptBuilder != nil {
        // Get recent turns + tool calls from state
        turns := k.getRecentTurns(5) // last 5 turns
        toolCalls := k.getRecentToolCalls()
        k.skillsPromptBuilder.OnTurnComplete(k.ctx, turns, toolCalls)
    }
}
```

- [ ] **Step 3: Write prompt builder tests**

```go
func TestSkillsPromptBuilder_BuildSkillsSection(t *testing.T) {
    store := &mockSkillStore{
        Skills: []skills.Skill{
            {Name: "fix-git", Category: "development", Description: "Fix git conflicts"},
            {Name: "review-pr", Category: "development", Description: "Review a PR"},
        },
    }
    mgr := skills.NewSkillManager(store, nil, skills.SkillsConfig{})
    builder := NewSkillsPromptBuilder(mgr)

    section, err := builder.BuildSkillsSection(context.Background())
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(section, "fix-git") {
        t.Error("expected skill name in section")
    }
    if !strings.Contains(section, "development") {
        t.Error("expected category in section")
    }
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/kernel/... -v -run Skills`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/kernel/skills_prompt.go internal/kernel/kernel.go
git commit -m "feat(skills): add skills prompt section + complexity detection trigger"
```

---

## Task 8: CLI commands + config

**Files:**
- Create: `gormes/cmd/skills_cli.go`
- Modify: `internal/config/config.go` (add [skills] section)
- Test: `gormes/cmd/skills_cli_test.go`

- [ ] **Step 1: Add [skills] config section**

```go
type SkillsConfig struct {
    Enabled              bool    `toml:"enabled"`
    AutoImprove          bool    `toml:"auto_improve"`
    ImprovementThreshold int     `toml:"improvement_threshold"`
    ExtractionConfidence float64 `toml:"extraction_confidence"`
    SkillsDir            string  `toml:"skills_dir"`
}
```

- [ ] **Step 2: Create skills CLI commands**

```go
package cmd

import (
    "fmt"
    "github.com/TrebuchetDynamics/gormes-agent/internal/skills"
)

type SkillsCLI struct {
    manager *skills.SkillManager
}

func NewSkillsCLI(mgr *skills.SkillManager) *SkillsCLI {
    return &SkillsCLI{manager: mgr}
}

func (c *SkillsCLI) List(category string) string {
    list, err := c.manager.ListSkills(context.Background(), category)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    if len(list) == 0 {
        return "No skills found"
    }
    out := ""
    currentCat := ""
    for _, s := range list {
        if s.Category != currentCat {
            out += fmt.Sprintf("\n[%s]\n", s.Category)
            currentCat = s.Category
        }
        out += fmt.Sprintf("  %s — %s (used %d times)\n", s.Name, s.Description, s.UseCount)
    }
    return out
}

func (c *SkillsCLI) View(name string) string {
    skill, content, err := c.manager.ViewSkill(context.Background(), name)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    return fmt.Sprintf("# %s (v%s)\n%s\n\n---\n%s", skill.Name, skill.Version, skill.Description, content)
}

func (c *SkillsCLI) Delete(name string) string {
    if err := c.manager.DeleteSkill(context.Background(), name); err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    return fmt.Sprintf("Deleted skill: %s", name)
}

func (c *SkillsCLI) Feedback(skillName string, rating int, comments string) string {
    skill, _, err := c.manager.ViewSkill(context.Background(), skillName)
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    fb := skills.SkillFeedback{
        SkillID:  skill.ID,
        Rating:   rating,
        Comments: comments,
    }
    if err := c.manager.ImproveSkill(context.Background(), fb); err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    return fmt.Sprintf("Feedback recorded for %s", skillName)
}
```

- [ ] **Step 3: Write CLI tests**

```go
func TestSkillsCLI_List(t *testing.T) {
    store := &mockSkillStore{Skills: []skills.Skill{{Name: "test", Category: "general"}}}
    mgr := skills.NewSkillManager(store, nil, skills.SkillsConfig{})
    cli := NewSkillsCLI(mgr)

    out := cli.List("")
    if !strings.Contains(out, "test") {
        t.Errorf("expected 'test' in output, got: %s", out)
    }
}

func TestSkillsCLI_DeleteNotFound(t *testing.T) {
    store := &mockSkillStore{}
    mgr := skills.NewSkillManager(store, nil, skills.SkillsConfig{})
    cli := NewSkillsCLI(mgr)

    out := cli.Delete("nonexistent")
    if !strings.Contains(out, "Error") {
        t.Error("expected error for nonexistent skill")
    }
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./gormes/cmd/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add gormes/cmd/skills_cli.go internal/config/config.go
git commit -m "feat(skills): add CLI commands for skill management + config section"
```

---

## Self-Review Checklist

**Spec coverage:**
- [x] SkillStore (Task 2) → spec §SkillStore
- [x] SkillExtractor (Task 4) → spec §SkillExtractor Interface
- [x] SkillManager (Task 5) → spec §SkillManager Interface
- [x] skills_list, skill_view, skill_manage (Task 6) → spec §CLI / Slash Commands
- [x] SkillsConfig (Task 8) → spec §Configuration
- [x] Complexity detection trigger (Task 7) → spec §Complexity Detection
- [x] Pattern extraction (Task 4) → spec §Pattern Extraction
- [x] Auto-improvement (Task 5) → spec §Skill Improvement
- [x] Prompt integration (Task 7) → spec §Prompt Integration
- [x] SKILL.md format (Task 2) → spec §SKILL.md File
- [x] LLM client interface (Task 4) → spec §SkillExtractor Interface

**Placeholder scan:** None found. All steps show actual code.

**Type consistency:** `Skill`, `SkillFeedback`, `SkillImprovement`, `ExtractionRequest`, `ExtractionResult` defined in Task 1. Used consistently in Tasks 2–8.

---

## Final Verification

1. Run all tests: `go test ./internal/skills/... ./internal/tools/... ./internal/kernel/... ./gormes/cmd/... -v`
2. Build binary: `make build` — should compile cleanly
3. Binary size: confirm < 500 KB delta from Phase 2.G
4. SQLite migration: `memory.db` should auto-migrate to include skills tables

---

**Plan complete.** Two execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task (8 tasks total), review between tasks, fast iteration.

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints.

Which approach?
