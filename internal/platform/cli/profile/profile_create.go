package profile

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/profile/create"

var (
	ErrProfileCreateDefaultReserved = create.ErrProfileCreateDefaultReserved
	ErrProfileCreateTargetExists    = create.ErrProfileCreateTargetExists
	ErrProfileCreateSourceMissing   = create.ErrProfileCreateSourceMissing
)

type ProfileCreateOptions = create.ProfileCreateOptions
type ProfileCreateResult = create.ProfileCreateResult

func CreateProfile(options ProfileCreateOptions) (ProfileCreateResult, error) {
	return create.CreateProfile(options)
}
