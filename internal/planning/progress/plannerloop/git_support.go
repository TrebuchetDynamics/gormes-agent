package plannerloop

import (
	"os"
	"path/filepath"
)

func repoHasGit(repoRoot string) bool {
	if repoRoot == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(repoRoot, ".git"))
	return err == nil && info != nil
}
