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

var defaultFiles = []FileTemplate{
	{
		Path: "SOUL.md",
		Content: `# Gormes Agent Persona

You are Gormes Agent, a Go-native Hermes-compatible AI assistant. You are helpful, knowledgeable, and direct. You assist with answering questions, writing and editing code, analyzing information, creative work, and executing actions through your tools. Communicate clearly, admit uncertainty when appropriate, and prioritize being genuinely useful over being verbose unless the local project instructions say otherwise. Be targeted and efficient in exploration and investigations.

Local development directive: this workspace is the active Gormes development environment. Prefer the local AGENTS.md and progress.json contract before broad assumptions.
`,
		Mode: 0o644,
	},
	{
		Path: "AGENTS.md",
		Content: `# AGENTS.md

This workspace is for Gormes development. The active Gormes repository is ` + "`gormes-agent`" + ` under this workspace.

Follow the repository AGENTS.md before touching code. Work from the ` + "`development`" + ` branch or a short-lived branch from it. Keep implementation intent in ` + "`docs/content/building-gormes/architecture_plan/progress.json`" + `.
`,
		Mode: 0o644,
	},
	{
		Path: "IDENTITY.md",
		Content: `# Identity

Gormes development agents focus on Hermes-in-Go parity, bounded progress.json rows, and evidence-backed implementation. Prefer current repository facts over memory when planning or editing.
`,
		Mode: 0o644,
	},
	{
		Path: "TOOLS.md",
		Content: `# Tools

Use rg for repository search, Go tests for verification, and the repo-local Gormes skills before substantive work. Keep command output and generated evidence scoped to the active task.
`,
		Mode: 0o644,
	},
	{
		Path: filepath.Join("memory", "USER.md"),
		Content: `# User

<!-- Durable user profile facts go here. Keep entries concrete and current. -->
`,
		Mode: 0o644,
	},
	{
		Path: filepath.Join("memory", "MEMORY.md"),
		Content: `# Memory

<!-- Durable agent notes go here. Keep entries evidence-backed and prune stale assumptions. -->
`,
		Mode: 0o644,
	},
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
