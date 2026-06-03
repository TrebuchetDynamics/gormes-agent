package gormescli

import (
	"github.com/spf13/cobra"

	appsecrets "github.com/TrebuchetDynamics/gormes-agent/internal/app/secrets"
)

// NewSecretsCommand builds the secrets command tree while keeping cmd/gormes on the shared CLI facade.
func NewSecretsCommand(build func() BuildProvenance) *cobra.Command {
	return appsecrets.NewCommand(appsecrets.Options{
		BuildProvenance: func() appsecrets.BuildProvenance {
			if build == nil {
				return appsecrets.BuildProvenance{}
			}
			provenance := build()
			return appsecrets.BuildProvenance{
				Version:   provenance.Version,
				GitCommit: provenance.GitCommit,
			}
		},
	})
}
