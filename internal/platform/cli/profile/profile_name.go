package profile

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/profile/identity"

// Sentinel errors returned by ValidateProfileName so callers can render uniform
// error messages without parsing free-form strings.
var (
	ErrProfileNameEmpty        = identity.ErrProfileNameEmpty
	ErrProfileNameTooLong      = identity.ErrProfileNameTooLong
	ErrProfileNameInvalidChars = identity.ErrProfileNameInvalidChars
	ErrProfileNameReserved     = identity.ErrProfileNameReserved
)

// ValidateProfileName reports whether name is a valid profile identifier.
//
// Names must match [a-z0-9][a-z0-9_-]{0,63} and must not collide with
// reserved CLI subcommand names or the retired built-in profile name
// "default".
func ValidateProfileName(name string) error {
	return identity.ValidateProfileName(name)
}
