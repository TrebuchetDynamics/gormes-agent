package profile

import "errors"

// ErrProfileXDGRootRequired is returned when ResolveProfileRoot is called
// without a non-empty XDG config home; the helper refuses to invent a default
// so callers stay in charge of env resolution.
var ErrProfileXDGRootRequired = errors.New("gormes XDG config home is required")

// ResolveProfileRoot maps a profile name and a caller-supplied XDG config home
// to the physical directory that holds that profile's Gormes state. It is a
// pure string helper: it never reads the environment, never stats, and never
// creates directories. Every valid profile, including the built-in main
// profile, resolves under gormesXDGConfigHome+"/gormes/profiles/"+name.
func ResolveProfileRoot(name string, gormesXDGConfigHome string) (string, error) {
	if gormesXDGConfigHome == "" {
		return "", ErrProfileXDGRootRequired
	}
	if err := ValidateProfileName(name); err != nil {
		return "", err
	}
	return gormesXDGConfigHome + "/gormes/profiles/" + name, nil
}
