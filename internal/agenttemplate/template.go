// Package agenttemplate writes first-party Gormes agent context templates.
package agenttemplate

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var ErrTargetRequired = errors.New("agent_template_target_required")

type Action string

const (
	ActionCreate         Action = "create"
	ActionOverwrite      Action = "overwrite"
	ActionSkip           Action = "skip"
	ActionWouldCreate    Action = "would_create"
	ActionWouldOverwrite Action = "would_overwrite"
	ActionWouldSkip      Action = "would_skip"
)

type FileTemplate struct {
	ID      string
	Path    string
	Content string
	Mode    fs.FileMode
}

type WriteOptions struct {
	TargetDir string
	Force     bool
	DryRun    bool
}

type FileResult struct {
	Path   string
	Action Action
}

type WriteResult struct {
	TargetDir string
	Files     []FileResult
}

func DefaultFiles() []FileTemplate {
	files := make([]FileTemplate, len(defaultFiles))
	copy(files, defaultFiles)
	return files
}

func ApplyDefaultTemplates(opts WriteOptions) (WriteResult, error) {
	target := strings.TrimSpace(opts.TargetDir)
	if target == "" {
		return WriteResult{}, ErrTargetRequired
	}
	target = filepath.Clean(target)
	result := WriteResult{
		TargetDir: target,
		Files:     make([]FileResult, 0, len(defaultFiles)),
	}

	for _, file := range defaultFiles {
		abs, err := templateTargetPath(target, file.Path)
		if err != nil {
			return result, err
		}
		exists, err := fileExists(abs)
		if err != nil {
			return result, fmt.Errorf("stat agent template %s: %w", displayPath(file.Path), err)
		}

		action := ActionCreate
		if exists {
			if opts.Force {
				action = ActionOverwrite
			} else {
				action = ActionSkip
			}
		}
		if opts.DryRun {
			result.Files = append(result.Files, FileResult{Path: displayPath(file.Path), Action: dryRunAction(action)})
			continue
		}
		if action == ActionSkip {
			result.Files = append(result.Files, FileResult{Path: displayPath(file.Path), Action: action})
			continue
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return result, fmt.Errorf("prepare agent template %s: %w", displayPath(file.Path), err)
		}
		mode := file.Mode
		if mode == 0 {
			mode = 0o644
		}
		if err := os.WriteFile(abs, []byte(file.Content), mode); err != nil {
			return result, fmt.Errorf("write agent template %s: %w", displayPath(file.Path), err)
		}
		result.Files = append(result.Files, FileResult{Path: displayPath(file.Path), Action: action})
	}
	return result, nil
}

func templateTargetPath(target, rel string) (string, error) {
	clean := filepath.Clean(rel)
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid agent template path %q", rel)
	}
	return filepath.Join(target, clean), nil
}

func fileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		return !info.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func dryRunAction(action Action) Action {
	switch action {
	case ActionCreate:
		return ActionWouldCreate
	case ActionOverwrite:
		return ActionWouldOverwrite
	case ActionSkip:
		return ActionWouldSkip
	default:
		return action
	}
}

func displayPath(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
}
