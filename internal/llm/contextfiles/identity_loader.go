package contextfiles

import (
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/guidance"
)

type IdentityLoaderOptions struct {
	ProfileDir string
	SkipSoul   bool
	MaxChars   int
}

type IdentityLoaderResult struct {
	Identity string
	Source   string
	Fallback bool
	Skipped  bool
	Evidence ContextFileEvidence
}

func LoadAgentIdentity(opts IdentityLoaderOptions) IdentityLoaderResult {
	if opts.SkipSoul {
		return IdentityLoaderResult{
			Identity: guidance.DefaultAgentIdentity,
			Source:   "default",
			Skipped:  true,
			Evidence: ContextFileEvidence{Source: "SOUL.md", Skipped: true},
		}
	}

	profileDir := opts.ProfileDir
	if profileDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			profileDir = filepath.Join(home, ".gormes")
		}
	}

	maxChars := opts.MaxChars
	if maxChars <= 0 {
		maxChars = contextFilesDefaultMaxChars
	}

	content, ev := loadSoulContext(profileDir, maxChars)
	if content != "" && !ev.Blocked && ev.Error == "" {
		return IdentityLoaderResult{
			Identity: content,
			Source:   ev.Source,
			Evidence: ev,
		}
	}

	return IdentityLoaderResult{
		Identity: guidance.DefaultAgentIdentity,
		Source:   "default",
		Fallback: true,
		Evidence: ev,
	}
}
