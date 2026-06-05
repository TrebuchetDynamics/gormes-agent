package provenance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// InventoryState is the file/directory presence state used by the memory
// provenance inventory. It stays intentionally small so JSON consumers can
// branch without parsing free-form text.
type InventoryState string

const (
	InventoryStatePresent InventoryState = "present"
	InventoryStateMissing InventoryState = "missing"
	InventoryStateError   InventoryState = "error"
)

// InventoryOptions controls ReadInventory. Callers pass the active profile
// root explicitly so tests and profile-aware CLI invocations never inspect the
// developer's live home by accident.
type InventoryOptions struct {
	ProfileRoot             string
	CWD                     string
	MemoryDBPath            string
	SelectedPromptMemoryDir string
	ContextMemoryDirEnv     string
	DB                      *sql.DB
}

// Inventory is the provenance-explicit read model behind
// `gormes memory status`. Goncho counts are deliberately separate from
// durable markdown files and legacy Hermes memory files so "0 active items" can
// never be mistaken for "no memory exists".
type Inventory struct {
	SelectedPromptMemoryDir    string             `json:"selected_prompt_memory_dir"`
	SelectedPromptMemoryDirRel string             `json:"-"`
	LegacyImportNeeded         bool               `json:"legacy_import_needed"`
	Goncho                     InventoryGoncho    `json:"goncho"`
	DurableMarkdown            InventoryMemoryDir `json:"durable_markdown"`
	LegacyHermes               InventoryMemoryDir `json:"legacy_hermes"`
	ContextFiles               []InventoryFile    `json:"context_files"`
	SessionTranscripts         InventoryDir       `json:"session_transcripts"`
}

type InventoryGoncho struct {
	Database      InventoryFile `json:"database"`
	ActiveItems   int           `json:"active_items"`
	TurnsTotal    int           `json:"turns_total"`
	EvalArtifacts int           `json:"eval_artifacts"`
}

type InventoryMemoryDir struct {
	Directory InventoryDir  `json:"directory"`
	User      InventoryFile `json:"user"`
	Memory    InventoryFile `json:"memory"`
}

type InventoryDir struct {
	RelativePath string         `json:"relative_path"`
	State        InventoryState `json:"state"`
	Files        int            `json:"files"`
	Error        string         `json:"error,omitempty"`
}

type InventoryFile struct {
	RelativePath string         `json:"relative_path"`
	State        InventoryState `json:"state"`
	SizeBytes    int64          `json:"size_bytes"`
	Error        string         `json:"error,omitempty"`
}

// ReadInventory inspects memory provenance metadata for one profile root. It
// does not read memory contents.
func ReadInventory(ctx context.Context, opts InventoryOptions) (Inventory, error) {
	root := strings.TrimSpace(opts.ProfileRoot)
	if root == "" {
		return Inventory{}, errors.New("memory inventory: profile root is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Inventory{}, fmt.Errorf("memory inventory: resolve profile root: %w", err)
	}
	dbPath := strings.TrimSpace(opts.MemoryDBPath)
	if dbPath == "" {
		dbPath = filepath.Join(absRoot, "memory.db")
	}

	selected := strings.TrimSpace(opts.SelectedPromptMemoryDir)
	if selected == "" {
		selected = selectInventoryPromptMemoryDir(absRoot, opts.CWD, opts.ContextMemoryDirEnv)
	}
	selectedRel := inventoryDisplayPath(absRoot, selected)

	inv := Inventory{
		SelectedPromptMemoryDir:    selectedRel,
		SelectedPromptMemoryDirRel: selectedRel,
		Goncho: InventoryGoncho{
			Database: inspectInventoryFile(absRoot, dbPath),
		},
		DurableMarkdown: readInventoryMemoryDir(absRoot, "memory"),
		LegacyHermes:    readInventoryMemoryDir(absRoot, "memories"),
		ContextFiles:    readInventoryContextFiles(absRoot),
		SessionTranscripts: readInventoryDir(
			absRoot,
			filepath.Join(absRoot, "sessions"),
		),
	}
	inv.LegacyImportNeeded = inventoryMemoryDirHasAny(inv.LegacyHermes) && !inventoryMemoryDirHasAny(inv.DurableMarkdown)

	if opts.DB != nil {
		if err := readInventoryDBCounts(ctx, opts.DB, &inv.Goncho); err != nil {
			return Inventory{}, err
		}
	}

	return inv, nil
}

func readInventoryDBCounts(ctx context.Context, db *sql.DB, goncho *InventoryGoncho) error {
	active, err := inventoryCountTable(ctx, db, "goncho_memory_items", "active = 1")
	if err != nil {
		return err
	}
	turns, err := inventoryCountTable(ctx, db, "turns", "")
	if err != nil {
		return err
	}
	eval, err := inventoryCountTable(ctx, db, "goncho_memory_eval_artifacts", "")
	if err != nil {
		return err
	}
	goncho.ActiveItems = active
	goncho.TurnsTotal = turns
	goncho.EvalArtifacts = eval
	return nil
}

func inventoryCountTable(ctx context.Context, db *sql.DB, table, where string) (int, error) {
	ok, err := inventoryTableExists(ctx, db, table)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, nil
	}
	query := "SELECT COUNT(*) FROM " + table
	if strings.TrimSpace(where) != "" {
		query += " WHERE " + where
	}
	var count int
	if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("memory inventory: count %s: %w", table, err)
	}
	return count, nil
}

func inventoryTableExists(ctx context.Context, db *sql.DB, table string) (bool, error) {
	var name string
	err := db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("memory inventory: inspect table %s: %w", table, err)
	}
	return true, nil
}

func readInventoryMemoryDir(root, rel string) InventoryMemoryDir {
	dir := filepath.Join(root, rel)
	return InventoryMemoryDir{
		Directory: readInventoryDir(root, dir),
		User:      inspectInventoryFile(root, filepath.Join(dir, "USER.md")),
		Memory:    inspectInventoryFile(root, filepath.Join(dir, "MEMORY.md")),
	}
}

func readInventoryContextFiles(root string) []InventoryFile {
	names := []string{"SOUL.md", "AGENTS.md", "IDENTITY.md", "TOOLS.md", "USER.md", "MEMORY.md"}
	items := make([]InventoryFile, 0, len(names))
	for _, name := range names {
		items = append(items, inspectInventoryFile(root, filepath.Join(root, name)))
	}
	return items
}

func inspectInventoryFile(root, path string) InventoryFile {
	item := InventoryFile{
		RelativePath: inventoryDisplayPath(root, path),
		State:        InventoryStateMissing,
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return item
	}
	if err != nil {
		item.State = InventoryStateError
		item.Error = err.Error()
		return item
	}
	if info.IsDir() {
		return item
	}
	item.State = InventoryStatePresent
	item.SizeBytes = info.Size()
	return item
}

func readInventoryDir(root, path string) InventoryDir {
	item := InventoryDir{
		RelativePath: inventoryDisplayPath(root, path),
		State:        InventoryStateMissing,
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return item
	}
	if err != nil {
		item.State = InventoryStateError
		item.Error = err.Error()
		return item
	}
	if !info.IsDir() {
		return item
	}
	item.State = InventoryStatePresent
	err = filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			item.Files++
		}
		return nil
	})
	if err != nil {
		item.State = InventoryStateError
		item.Error = err.Error()
	}
	return item
}

func inventoryMemoryDirHasAny(item InventoryMemoryDir) bool {
	return item.User.State == InventoryStatePresent || item.Memory.State == InventoryStatePresent
}

func selectInventoryPromptMemoryDir(root, cwd, override string) string {
	if strings.TrimSpace(override) != "" {
		return strings.TrimSpace(override)
	}
	native := filepath.Join(root, "memory")
	if inventoryDirHasAnyFile(native, "USER.md", "MEMORY.md") {
		return native
	}
	legacy := filepath.Join(root, "memories")
	if inventoryDirHasAnyFile(legacy, "USER.md", "MEMORY.md") {
		return legacy
	}
	if dir := findInventoryAncestorWithAny(cwd, "USER.md", "MEMORY.md"); dir != "" {
		return dir
	}
	if dir := findInventoryAncestorSubdirWithAny(cwd, "memory", "USER.md", "MEMORY.md"); dir != "" {
		return dir
	}
	return native
}

func inventoryDirHasAnyFile(dir string, names ...string) bool {
	for _, name := range names {
		info, err := os.Stat(filepath.Join(dir, name))
		if err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

func findInventoryAncestorWithAny(start string, names ...string) string {
	start = strings.TrimSpace(start)
	if start == "" {
		return ""
	}
	dir, err := filepath.Abs(start)
	if err != nil {
		dir = start
	}
	for {
		if inventoryDirHasAnyFile(dir, names...) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func findInventoryAncestorSubdirWithAny(start, subdir string, names ...string) string {
	start = strings.TrimSpace(start)
	if start == "" {
		return ""
	}
	dir, err := filepath.Abs(start)
	if err != nil {
		dir = start
	}
	for {
		candidate := filepath.Join(dir, subdir)
		if inventoryDirHasAnyFile(candidate, names...) {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func inventoryDisplayPath(root, path string) string {
	root = strings.TrimSpace(root)
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if root != "" {
		if rel, err := filepath.Rel(root, path); err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
			return filepath.ToSlash(rel)
		}
		if rel, err := filepath.Rel(root, path); err == nil && rel == "." {
			return "."
		}
	}
	return filepath.ToSlash(path)
}
