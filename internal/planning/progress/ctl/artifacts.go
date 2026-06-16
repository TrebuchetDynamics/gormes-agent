package progressctl

import (
	"io"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
	ctlartifacts "github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/ctl/artifacts"
)

// WriteOptions controls progress write execution.
type WriteOptions = ctlartifacts.Options

type artifact = ctlartifacts.Artifact

func planArtifacts(p *progress.Progress, paths pathSet) []artifact {
	return ctlartifacts.Plan(p, artifactPaths(paths))
}

func WriteWithOptions(stdout io.Writer, root string, opts WriteOptions) error {
	p, err := loadValidProgress(root)
	if err != nil {
		return err
	}
	return ctlartifacts.WriteWithOptions(stdout, p, artifactPaths(progressPaths(root)), opts)
}

func artifactPaths(paths pathSet) ctlartifacts.Paths {
	return ctlartifacts.Paths{
		DocsIndex:          paths.docsIndex,
		Readme:             paths.readme,
		ContractReadiness:  paths.contractReadiness,
		BuilderLoopHandoff: paths.builderLoopHandoff,
		AgentQueue:         paths.agentQueue,
		NextSlices:         paths.nextSlices,
		BlockedSlices:      paths.blockedSlices,
		UmbrellaCleanup:    paths.umbrellaCleanup,
		ProgressSchema:     paths.progressSchema,
		ModuleRoadmapsDir:  paths.moduleRoadmapsDir,
		SiteProgress:       paths.siteProgress,
	}
}
