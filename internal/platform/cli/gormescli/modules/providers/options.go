package providers

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"

// Options carries binary-owned values into the provider module without making
// importable command code depend on cmd/gormes.
type Options struct {
	BuildProvenance func() gormescli.BuildProvenance
}

func (o Options) buildProvenance() gormescli.BuildProvenance {
	if o.BuildProvenance == nil {
		return gormescli.BuildProvenance{}
	}
	return o.BuildProvenance()
}
