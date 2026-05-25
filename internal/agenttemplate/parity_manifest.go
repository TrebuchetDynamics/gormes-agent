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
	TemplatePairByteEquivalent TemplatePairStatus = "byte_equivalent"
	TemplatePairTransformed    TemplatePairStatus = "transformed"
	TemplatePairGormesOwned    TemplatePairStatus = "gormes_owned"
	TemplatePairNotApplicable  TemplatePairStatus = "not_applicable"
	TemplatePairBlocked        TemplatePairStatus = "blocked"

	TemplatePairCovered         = TemplatePairByteEquivalent
	TemplatePairOwnedDivergence = TemplatePairGormesOwned
)

// TemplatePair records the source-backed parity classification for one
// first-run agent context template. Covered entries track a direct Hermes
// source; owned divergences document Gormes templates that Hermes consumes or
// inspires but does not seed as matching files.
type TemplatePair struct {
	TemplateID      string
	Path            string
	Status          TemplatePairStatus
	HermesSources   []string
	GormesSources   []string
	TransformReason string
	TestGate        []string
	OwnerRow        string
	Contract        string
}

type TemplatePairValidationOptions struct {
	RepoRoot   string
	HermesRoot string
}

func TemplatePairManifest() []TemplatePair {
	return []TemplatePair{
		{
			TemplateID: "soul",
			Path:       "SOUL.md",
			Status:     TemplatePairTransformed,
			HermesSources: []string{
				"hermes_cli/default_soul.py",
				"hermes_cli/config.py",
			},
			GormesSources: []string{
				"internal/hermes/default_soul.go",
				"internal/agenttemplate/default_templates.go",
			},
			TransformReason: "Gormes replaces the upstream Hermes/Nous identity with the Gorm persona and gormes runtime, then appends Gormes-owned concise personality and boundary guidance while leaving workflow rules to AGENTS.md/TOOLS.md.",
			TestGate: []string{
				"go test ./internal/hermes -run TestDefaultSoulMD -count=1",
				"go test ./internal/agenttemplate -count=1",
			},
			OwnerRow: "Gormes agent template reset command",
			Contract: "Gormes seeds SOUL.md from the Hermes DEFAULT_SOUL_MD contract with the declared Gormes product-identity substitution, then appends a lean Gormes-owned personality/boundary block; workflow-specific instructions live in AGENTS.md, IDENTITY.md, or TOOLS.md.",
		},
		{
			TemplateID: "agents",
			Path:       "AGENTS.md",
			Status:     TemplatePairGormesOwned,
			HermesSources: []string{
				"agent/prompt_builder.py",
			},
			GormesSources: []string{
				"internal/hermes/context_files.go",
				"internal/agenttemplate/default_templates.go",
			},
			TransformReason: "Hermes consumes AGENTS.md as project context but does not seed this file; Gormes owns a starter workspace contract for clean installs.",
			TestGate: []string{
				"go test ./internal/agenttemplate -count=1",
				"go test ./cmd/gormes -run TestAgentResetCommand -count=1",
			},
			OwnerRow: "Gormes agent template reset command",
			Contract: "Hermes consumes AGENTS.md as project context but does not seed a default AGENTS.md; Gormes owns a starter workspace contract so clean installs have editable project instructions.",
		},
		{
			TemplateID: "identity",
			Path:       "IDENTITY.md",
			Status:     TemplatePairGormesOwned,
			HermesSources: []string{
				"agent/prompt_builder.py",
			},
			GormesSources: []string{
				"internal/hermes/context_files.go",
				"internal/agenttemplate/default_templates.go",
			},
			TransformReason: "Hermes has no matching seeded IDENTITY.md; Gormes owns this editable identity file for stable agent and workspace facts.",
			TestGate: []string{
				"go test ./internal/agenttemplate -count=1",
				"go test ./cmd/gormes -run TestAgentResetCommand -count=1",
			},
			OwnerRow: "Gormes agent template reset command",
			Contract: "Hermes has no matching seeded IDENTITY.md; Gormes owns this additive operational context file for stable agent and workspace identity facts.",
		},
		{
			TemplateID: "tools",
			Path:       "TOOLS.md",
			Status:     TemplatePairGormesOwned,
			HermesSources: []string{
				"agent/prompt_builder.py",
			},
			GormesSources: []string{
				"internal/hermes/context_files.go",
				"internal/agenttemplate/default_templates.go",
			},
			TransformReason: "Hermes exposes tool guidance through prompt assembly rather than a seeded TOOLS.md file; Gormes owns this editable tool and verification rules file.",
			TestGate: []string{
				"go test ./internal/agenttemplate -count=1",
				"go test ./cmd/gormes -run TestAgentResetCommand -count=1",
			},
			OwnerRow: "Gormes agent template reset command",
			Contract: "Hermes exposes tool guidance through prompt assembly rather than a seeded TOOLS.md file; Gormes owns this additive operational context file for workspace tool and verification rules.",
		},
		{
			TemplateID: "memory-user",
			Path:       "memory/USER.md",
			Status:     TemplatePairGormesOwned,
			HermesSources: []string{
				"agent/prompt_builder.py",
			},
			GormesSources: []string{
				"internal/hermes/durable_user_context.go",
				"internal/agenttemplate/default_templates.go",
			},
			TransformReason: "Hermes supports durable user context in prompt assembly but does not seed this exact memory/USER.md template; Gormes owns the editable durable-user starter file.",
			TestGate: []string{
				"go test ./internal/agenttemplate -count=1",
				"go test ./cmd/gormes -run TestAgentResetCommand -count=1",
			},
			OwnerRow: "Gormes agent template reset command",
			Contract: "Hermes supports durable user context in prompt assembly but does not seed this exact memory/USER.md template; Gormes owns the editable durable-user starter file.",
		},
		{
			TemplateID: "memory-memory",
			Path:       "memory/MEMORY.md",
			Status:     TemplatePairGormesOwned,
			HermesSources: []string{
				"agent/prompt_builder.py",
			},
			GormesSources: []string{
				"internal/hermes/durable_user_context.go",
				"internal/agenttemplate/default_templates.go",
			},
			TransformReason: "Hermes supports durable memory context in prompt assembly but does not seed this exact memory/MEMORY.md template; Gormes owns the editable durable-memory starter file.",
			TestGate: []string{
				"go test ./internal/agenttemplate -count=1",
				"go test ./cmd/gormes -run TestAgentResetCommand -count=1",
			},
			OwnerRow: "Gormes agent template reset command",
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
		if strings.TrimSpace(pair.TemplateID) == "" {
			errs = append(errs, fmt.Errorf("%s template ID is required", prefix))
		}
		if seen[pair.Path] {
			errs = append(errs, fmt.Errorf("%s duplicate template path", prefix))
		}
		seen[pair.Path] = true

		if !validTemplatePairStatus(pair.Status) {
			errs = append(errs, fmt.Errorf("%s unsupported status %q", prefix, pair.Status))
		}
		if pair.Status != TemplatePairByteEquivalent && strings.TrimSpace(pair.TransformReason) == "" {
			errs = append(errs, fmt.Errorf("%s transform reason is required for status %q", prefix, pair.Status))
		}
		if strings.TrimSpace(pair.OwnerRow) == "" {
			errs = append(errs, fmt.Errorf("%s owner row is required", prefix))
		}
		if len(pair.TestGate) == 0 {
			errs = append(errs, fmt.Errorf("%s test gate is required", prefix))
		}
		for _, gate := range pair.TestGate {
			if strings.TrimSpace(gate) == "" {
				errs = append(errs, fmt.Errorf("%s test gate entries must be non-empty", prefix))
			}
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

func validTemplatePairStatus(status TemplatePairStatus) bool {
	switch status {
	case TemplatePairByteEquivalent, TemplatePairTransformed, TemplatePairGormesOwned, TemplatePairNotApplicable, TemplatePairBlocked:
		return true
	default:
		return false
	}
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
