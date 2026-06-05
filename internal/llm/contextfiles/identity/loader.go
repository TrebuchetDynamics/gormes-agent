package identity

import (
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/contextfiles/contextsource"
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
	Evidence contextsource.Evidence
}

func LoadAgentIdentity(opts IdentityLoaderOptions) IdentityLoaderResult {
	if opts.SkipSoul {
		return IdentityLoaderResult{
			Identity: guidance.DefaultAgentIdentity,
			Source:   "default",
			Skipped:  true,
			Evidence: contextsource.Evidence{Source: "SOUL.md", Skipped: true},
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
		maxChars = contextsource.DefaultMaxChars
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

func loadSoulContext(profileDir string, maxChars int) (string, contextsource.Evidence) {
	ev := contextsource.Evidence{Source: "SOUL.md"}
	if profileDir == "" {
		ev.Missing = true
		return "", ev
	}
	path := filepath.Join(profileDir, "SOUL.md")
	content, ok := contextsource.ReadFile(path, &ev)
	if !ok {
		return "", ev
	}
	content, ev = contextsource.ScanContent(content, "SOUL.md", ev)
	content, ev = contextsource.TruncateContent(content, "SOUL.md", maxChars, ev)
	return content, ev
}
