package environment

import "strings"

// SubprocessHomeResolver returns the profile-local HOME that should be used
// by local shell subprocesses. It is injected from config to keep tools free of
// config package dependencies and to make the behavior directly testable.
type SubprocessHomeResolver func() (string, bool)

func EnvWithSubprocessHome(env []string, resolve SubprocessHomeResolver) []string {
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
