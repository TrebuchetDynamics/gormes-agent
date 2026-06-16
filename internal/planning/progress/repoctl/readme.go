package repoctl

import "github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/repoctl/readme"

type ReadmeOptions = readme.ReadmeOptions

func UpdateReadme(opts ReadmeOptions) error {
	return readme.UpdateReadme(opts)
}
