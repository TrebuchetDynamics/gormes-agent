package subprocess

import "strings"

// HomeResolver returns the profile-local HOME that should be used by local
// shell subprocesses.
type HomeResolver func() (string, bool)

// EnvWithHome overlays the resolved profile-local HOME onto env, preserving the
// input when the resolver is absent or empty.
func EnvWithHome(env []string, resolve HomeResolver) []string {
	out := append([]string(nil), env...)
	if resolve == nil {
		return out
	}
	home, ok := resolve()
	home = strings.TrimSpace(home)
	if !ok || home == "" {
		return out
	}

	replaced := false
	out = out[:0]
	for _, entry := range env {
		key, _, hasValue := strings.Cut(entry, "=")
		if hasValue && key == "HOME" {
			if !replaced {
				out = append(out, "HOME="+home)
				replaced = true
			}
			continue
		}
		out = append(out, entry)
	}
	if !replaced {
		out = append(out, "HOME="+home)
	}
	return out
}
