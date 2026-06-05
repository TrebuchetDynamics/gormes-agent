package kanban

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var boardSlugRE = regexp.MustCompile(`^[a-z0-9]([a-z0-9_-]*[a-z0-9])?$|^[a-z0-9]$`)

const maxBoardSlugLen = 64

func ValidateBoardSlug(slug string) error {
	if slug == "" {
		return errors.New("board name must not be empty")
	}
	if len(slug) > maxBoardSlugLen {
		return fmt.Errorf("board name too long: %d > %d", len(slug), maxBoardSlugLen)
	}
	if !boardSlugRE.MatchString(slug) {
		return fmt.Errorf("invalid board name %q: must be lowercase letters, digits, hyphens, underscores; cannot start or end with hyphen/underscore", slug)
	}
	return nil
}

func NormalizeBoardSlug(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

type Board struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type BoardRegistry struct {
	root      string
	boardsDir string
	currentFn string
}

func NewBoardRegistry(root string) *BoardRegistry {
	return &BoardRegistry{
		root:      root,
		boardsDir: filepath.Join(root, "kanban", "boards"),
		currentFn: filepath.Join(root, "kanban", "current"),
	}
}

func (r *BoardRegistry) BoardPath(slug string) string {
	if slug == "default" || slug == "" {
		return filepath.Join(r.root, "kanban.db")
	}
	return filepath.Join(r.boardsDir, slug, "kanban.db")
}

func (r *BoardRegistry) Create(slug string) error {
	slug = NormalizeBoardSlug(slug)
	if err := ValidateBoardSlug(slug); err != nil {
		return err
	}
	if slug == "default" {
		return errors.New("cannot create board named 'default'")
	}

	boardDir := filepath.Join(r.boardsDir, slug)
	if _, err := os.Stat(boardDir); err == nil {
		return fmt.Errorf("board %q already exists", slug)
	}

	if err := os.MkdirAll(boardDir, 0o755); err != nil {
		return fmt.Errorf("create board dir: %w", err)
	}
	return nil
}

func (r *BoardRegistry) List() ([]Board, error) {
	entries, err := os.ReadDir(r.boardsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list boards: %w", err)
	}
	var boards []Board
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		slug := entry.Name()
		if slug == "" || strings.HasPrefix(slug, ".") || strings.HasPrefix(slug, "_") {
			continue
		}
		boards = append(boards, Board{
			Name: slug,
			Path: r.BoardPath(slug),
		})
	}
	return boards, nil
}

func (r *BoardRegistry) Switch(slug string) error {
	slug = NormalizeBoardSlug(slug)
	if slug != "default" {
		if err := ValidateBoardSlug(slug); err != nil {
			return err
		}
		boardDir := filepath.Join(r.boardsDir, slug)
		if _, err := os.Stat(boardDir); os.IsNotExist(err) {
			return fmt.Errorf("board %q does not exist", slug)
		}
	}

	dir := filepath.Dir(r.currentFn)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("prepare kanban dir: %w", err)
	}
	if err := os.WriteFile(r.currentFn, []byte(slug+"\n"), 0o644); err != nil {
		return fmt.Errorf("write current board: %w", err)
	}
	return nil
}

func (r *BoardRegistry) Current() (Board, error) {
	data, err := os.ReadFile(r.currentFn)
	if err != nil {
		if os.IsNotExist(err) {
			return Board{Name: "default", Path: r.BoardPath("default")}, nil
		}
		return Board{}, fmt.Errorf("read current board: %w", err)
	}
	slug := strings.TrimSpace(string(data))
	if slug == "" {
		return Board{Name: "default", Path: r.BoardPath("default")}, nil
	}
	return Board{Name: slug, Path: r.BoardPath(slug)}, nil
}

func (r *BoardRegistry) Rename(oldSlug, newSlug string) error {
	oldSlug = NormalizeBoardSlug(oldSlug)
	newSlug = NormalizeBoardSlug(newSlug)
	if oldSlug == "default" {
		return errors.New("cannot rename the default board")
	}
	if err := ValidateBoardSlug(newSlug); err != nil {
		return err
	}
	if newSlug == "default" {
		return errors.New("cannot rename to 'default'")
	}

	oldDir := filepath.Join(r.boardsDir, oldSlug)
	newDir := filepath.Join(r.boardsDir, newSlug)
	if _, err := os.Stat(oldDir); os.IsNotExist(err) {
		return fmt.Errorf("board %q does not exist", oldSlug)
	}
	if _, err := os.Stat(newDir); err == nil {
		return fmt.Errorf("board %q already exists", newSlug)
	}

	if err := os.Rename(oldDir, newDir); err != nil {
		return fmt.Errorf("rename board dir: %w", err)
	}

	cur, _ := r.Current()
	if cur.Name == oldSlug {
		if err := r.Switch(newSlug); err != nil {
			return fmt.Errorf("update current after rename: %w", err)
		}
	}
	return nil
}

func (r *BoardRegistry) Remove(slug string) error {
	slug = NormalizeBoardSlug(slug)
	if slug == "default" {
		return errors.New("cannot remove the default board")
	}
	if err := ValidateBoardSlug(slug); err != nil {
		return err
	}

	boardDir := filepath.Join(r.boardsDir, slug)
	if _, err := os.Stat(boardDir); os.IsNotExist(err) {
		return fmt.Errorf("board %q does not exist", slug)
	}

	if err := os.RemoveAll(boardDir); err != nil {
		return fmt.Errorf("remove board dir: %w", err)
	}

	cur, _ := r.Current()
	if cur.Name == slug {
		if err := r.writeCurrent(""); err != nil {
			return fmt.Errorf("clear current after remove: %w", err)
		}
	}
	return nil
}

func (r *BoardRegistry) writeCurrent(slug string) error {
	dir := filepath.Dir(r.currentFn)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("prepare kanban dir: %w", err)
	}
	return os.WriteFile(r.currentFn, []byte(slug+"\n"), 0o644)
}
