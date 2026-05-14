package agenttemplate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type TemplatePairStatus string

const (
	TemplatePairCovered         TemplatePairStatus = "covered"
	TemplatePairOwnedDivergence TemplatePairStatus = "owned_divergence"
)

// TemplatePair records the source-backed parity classification for one
// first-run agent context template. Covered entries track a direct Hermes
// source; owned divergences document Gormes templates that Hermes consumes or
// inspires but does not seed as matching files.
type TemplatePair struct {
	Path          string
	Status        TemplatePairStatus
	HermesSources []string
	GormesSources []string
	Contract      string
}

type TemplatePairValidationOptions struct {
	RepoRoot   string
	HermesRoot string
}

func TemplatePairManifest() []TemplatePair {
	return []TemplatePair{
		{
			Path:   "SOUL.md",
			Status: TemplatePairCovered,
			HermesSources: []string{
				"hermes_cli/default_soul.py",
				"hermes_cli/config.py",
			},
			GormesSources: []string{
				"internal/hermes/default_soul.go",
				"internal/agenttemplate/default_templates.go",
			},
			Contract: "Gormes seeds SOUL.md from the Hermes DEFAULT_SOUL_MD contract with only the declared Gormes product-identity substitution, then appends Gormes-owned operating and boundary guidance.",
		},
		{
			Path:   "AGENTS.md",
			Status: TemplatePairOwnedDivergence,
			HermesSources: []string{
				"agent/prompt_builder.py",
			},
			GormesSources: []string{
				"internal/hermes/context_files.go",
				"internal/agenttemplate/default_templates.go",
			},
			Contract: "Hermes consumes AGENTS.md as project context but does not seed a default AGENTS.md; Gormes owns a starter workspace contract so clean installs have editable project instructions.",
		},
		{
			Path:   "IDENTITY.md",
			Status: TemplatePairOwnedDivergence,
			HermesSources: []string{
				"agent/prompt_builder.py",
			},
			GormesSources: []string{
				"internal/hermes/context_files.go",
				"internal/agenttemplate/default_templates.go",
			},
			Contract: "Hermes has no matching seeded IDENTITY.md; Gormes owns this additive operational context file for stable agent and workspace identity facts.",
		},
		{
			Path:   "TOOLS.md",
			Status: TemplatePairOwnedDivergence,
			HermesSources: []string{
				"agent/prompt_builder.py",
			},
			GormesSources: []string{
				"internal/hermes/context_files.go",
				"internal/agenttemplate/default_templates.go",
			},
			Contract: "Hermes exposes tool guidance through prompt assembly rather than a seeded TOOLS.md file; Gormes owns this additive operational context file for workspace tool and verification rules.",
		},
		{
			Path:   "memory/USER.md",
			Status: TemplatePairOwnedDivergence,
			HermesSources: []string{
				"agent/prompt_builder.py",
			},
			GormesSources: []string{
				"internal/hermes/durable_user_context.go",
				"internal/agenttemplate/default_templates.go",
			},
			Contract: "Hermes supports durable user context in prompt assembly but does not seed this exact memory/USER.md template; Gormes owns the editable durable-user starter file.",
		},
		{
			Path:   "memory/MEMORY.md",
			Status: TemplatePairOwnedDivergence,
			HermesSources: []string{
				"agent/prompt_builder.py",
			},
			GormesSources: []string{
				"internal/hermes/durable_user_context.go",
				"internal/agenttemplate/default_templates.go",
			},
			Contract: "Hermes supports durable memory context in prompt assembly but does not seed this exact memory/MEMORY.md template; Gormes owns the editable durable-memory starter file.",
		},
	}
}

func ValidateTemplatePairManifest(opts TemplatePairValidationOptions) error {
	return ValidateTemplatePairs(TemplatePairManifest(), opts)
}

func ValidateTemplatePairs(pairs []TemplatePair, opts TemplatePairValidationOptions) error {
	repoRoot := strings.TrimSpace(opts.RepoRoot)
	if repoRoot == "" {
		repoRoot = "."
	}
	hermesRoot := strings.TrimSpace(opts.HermesRoot)
	if hermesRoot == "" {
		hermesRoot = filepath.Join(repoRoot, "hermes-agent")
	}

	var errs []error
	seen := map[string]bool{}
	for i, pair := range pairs {
		prefix := fmt.Sprintf("template pair[%d] %s:", i, pair.Path)
		cleanPath, err := validateTemplatePairRelPath(pair.Path)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s invalid template path: %w", prefix, err))
			continue
		}
		if pair.Path != cleanPath {
			errs = append(errs, fmt.Errorf("%s template path must be slash-cleaned as %q", prefix, cleanPath))
		}
		if seen[pair.Path] {
			errs = append(errs, fmt.Errorf("%s duplicate template path", prefix))
		}
		seen[pair.Path] = true

		if pair.Status != TemplatePairCovered && pair.Status != TemplatePairOwnedDivergence {
			errs = append(errs, fmt.Errorf("%s unsupported status %q", prefix, pair.Status))
		}
		if strings.TrimSpace(pair.Contract) == "" {
			errs = append(errs, fmt.Errorf("%s contract is required", prefix))
		}
		if len(pair.HermesSources) == 0 {
			errs = append(errs, fmt.Errorf("%s Hermes source references are required", prefix))
		}
		if len(pair.GormesSources) == 0 {
			errs = append(errs, fmt.Errorf("%s Gormes source references are required", prefix))
		}

		for _, source := range pair.HermesSources {
			if err := validateTemplatePairSourceFile(hermesRoot, source); err != nil {
				errs = append(errs, fmt.Errorf("%s missing Hermes source %q: %w", prefix, source, err))
			}
		}
		for _, source := range pair.GormesSources {
			if err := validateTemplatePairSourceFile(repoRoot, source); err != nil {
				errs = append(errs, fmt.Errorf("%s missing Gormes source %q: %w", prefix, source, err))
			}
		}
	}

	for _, file := range DefaultFiles() {
		path := filepath.ToSlash(file.Path)
		if !seen[path] {
			errs = append(errs, fmt.Errorf("template pair manifest missing default template %q", path))
		}
	}
	return errors.Join(errs...)
}

func validateTemplatePairSourceFile(root, rel string) error {
	clean, err := validateTemplatePairRelPath(rel)
	if err != nil {
		return err
	}
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(clean)))
	if err != nil {
		return err
	}
	if info.IsDir() {
		return errors.New("is a directory")
	}
	return nil
}

func validateTemplatePairRelPath(rel string) (string, error) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", errors.New("empty path")
	}
	if strings.Contains(rel, "\\") {
		return "", errors.New("path must use slash separators")
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("unsafe relative path %q", rel)
	}
	return clean, nil
}
