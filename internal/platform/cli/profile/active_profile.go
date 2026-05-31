package profile

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/profile/active"

// ErrActiveProfileUnset is returned by ReadActiveProfile when the active
// profile file does not exist. Callers can use errors.Is to detect this
// degraded mode and fall back to the main profile without surfacing a
// hard error.
var ErrActiveProfileUnset = active.ErrActiveProfileUnset

// ReadActiveProfile reads the sticky active profile name from rootFile.
func ReadActiveProfile(rootFile string) (string, error) {
	return active.ReadActiveProfile(rootFile)
}

// WriteActiveProfile persists name as the active profile in rootFile.
func WriteActiveProfile(rootFile, name string) error {
	return active.WriteActiveProfile(rootFile, name)
}

// ClearActiveProfile removes rootFile so future reads return
// ErrActiveProfileUnset. The operation is idempotent: a missing file is not
// an error.
func ClearActiveProfile(rootFile string) error {
	return active.ClearActiveProfile(rootFile)
}
