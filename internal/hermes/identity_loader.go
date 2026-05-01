package hermes

import (
	"os"
	"path/filepath"
)

const DefaultAgentIdentity = "You are Gormes Agent, an intelligent AI assistant. You are helpful, knowledgeable, and direct. You assist users with a wide range of tasks including answering questions, writing and editing code, analyzing information, creative work, and executing actions via your tools. You communicate clearly, admit uncertainty when appropriate, and prioritize being genuinely useful over being verbose unless otherwise directed. Be targeted and efficient in your exploration and investigations."

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
			Identity: DefaultAgentIdentity,
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
		Identity: DefaultAgentIdentity,
		Source:   "default",
		Fallback: true,
		Evidence: ev,
	}
}
