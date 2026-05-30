package backend

import (
	"fmt"
	"os"
	"path/filepath"
)

func BuildCommand(backend, mode string) ([]string, error) {
	return BuildCommandWithRepoRoot(backend, mode, "")
}

func BuildCommandWithRepoRoot(backend, mode, repoRoot string) ([]string, error) {
	if backend == "" {
		backend = "codexu"
	}
	if mode == "" {
		mode = "safe"
	}

	sandbox, err := sandboxForMode(mode)
	if err != nil {
		return nil, err
	}

	backendBinary := backend
	switch backend {
	case "codexu":
		if shim := resolveRepoShim(repoRoot, "codexu"); shim != "" {
			backendBinary = shim
		}
		return []string{backendBinary, "exec", "--json", "--ephemeral", "-m", "gpt-5.5", "-c", "approval_policy=never", "--sandbox", sandbox}, nil
	case "claudeu":
		if shim := resolveRepoShim(repoRoot, "claudeu"); shim != "" {
			backendBinary = shim
		}
		return []string{backendBinary, "exec", "--json", "-m", "gpt-5.5", "-c", "approval_policy=never", "--sandbox", sandbox}, nil
	case "opencode":
		return []string{"opencode", "run"}, nil
	default:
		return nil, fmt.Errorf("invalid BACKEND %q: expected codexu, claudeu, or opencode", backend)
	}
}

func resolveRepoShim(repoRoot, name string) string {
	if repoRoot == "" {
		return ""
	}
	candidate := filepath.Join(repoRoot, "scripts", "orchestrator", name)
	info, err := os.Stat(candidate)
	if err != nil || info.IsDir() {
		return ""
	}
	if info.Mode()&0o111 == 0 {
		return ""
	}
	abs, err := filepath.Abs(candidate)
	if err != nil {
		return candidate
	}
	return abs
}

func sandboxForMode(mode string) (string, error) {
	switch mode {
	case "safe", "unattended":
		return "workspace-write", nil
	case "full":
		return "danger-full-access", nil
	default:
		return "", fmt.Errorf("invalid MODE %q: expected safe, unattended, or full", mode)
	}
}
