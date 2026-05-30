package splitlayout

import "os"

// Manifest and member-directory names for the on-disk progress split layout.
const (
	IndexName  = "index.json"
	PhasesDir  = "phases"
	ModulesDir = "modules"
)

// Supported split-layout keying modes. The empty key is normalized to phase
// for backward compatibility with early split indexes.
const (
	KeyByPhase  = "phase"
	KeyByModule = "module"
)

// IsDir reports whether path is an existing directory that should be treated
// as a split-layout target by the progress loader/writer.
func IsDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

// NormalizeKeyBy returns the explicit keying mode for a manifest key_by value.
func NormalizeKeyBy(keyBy string) (string, bool) {
	switch keyBy {
	case "", KeyByPhase:
		return KeyByPhase, true
	case KeyByModule:
		return KeyByModule, true
	default:
		return "", false
	}
}
