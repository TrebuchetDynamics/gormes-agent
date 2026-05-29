package progressctl

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
)

// WriteOptions controls progress write execution.
type WriteOptions struct {
	DryRun bool
}

type artifact struct {
	Kind  string
	Path  string
	Label string
	Group string
	Write func() error
}

func planArtifacts(p *progress.Progress, paths pathSet) []artifact {
	markers := []struct {
		path   string
		kind   string
		label  string
		render func(*progress.Progress) string
	}{
		{path: paths.docsIndex, kind: "docs-full-checklist", label: "_index.md regenerated", render: progress.RenderDocsChecklist},
		{path: paths.readme, kind: "readme-rollup", label: "README.md regenerated", render: progress.RenderReadmeRollup},
		{path: paths.contractReadiness, kind: "contract-readiness", label: "contract readiness regenerated", render: progress.RenderContractReadiness},
		{path: paths.builderLoopHandoff, kind: "builder-loop-handoff", label: "builder-loop handoff regenerated", render: progress.RenderBuilderLoopHandoff},
		{path: paths.agentQueue, kind: "agent-queue", label: "agent queue regenerated", render: func(p *progress.Progress) string { return progress.RenderAgentQueue(p, 10) }},
		{path: paths.nextSlices, kind: "next-slices", label: "next slices regenerated", render: func(p *progress.Progress) string { return progress.RenderNextSlices(p, 10) }},
		{path: paths.blockedSlices, kind: "blocked-slices", label: "blocked slices regenerated", render: progress.RenderBlockedSlices},
		{path: paths.umbrellaCleanup, kind: "umbrella-cleanup", label: "umbrella cleanup regenerated", render: progress.RenderUmbrellaCleanup},
		{path: paths.progressSchema, kind: "progress-schema", label: "progress schema regenerated", render: func(*progress.Progress) string { return progress.RenderProgressSchema() }},
	}

	artifacts := make([]artifact, 0, len(markers)+len(progress.AllowedModules())+2+len(paths.siteProgress))
	for _, marker := range markers {
		marker := marker
		body := marker.render(p)
		artifacts = append(artifacts, artifact{
			Kind:  "marker:" + marker.kind,
			Path:  marker.path,
			Label: marker.label,
			Write: func() error { return rewriteMarker(marker.path, marker.kind, body) },
		})
	}

	moduleIndexPath := filepath.Join(paths.moduleRoadmapsDir, "_index.md")
	artifacts = append(artifacts, artifact{
		Kind:  "module-roadmap:index",
		Path:  moduleIndexPath,
		Group: "module-roadmaps",
		Write: func() error { return writeTextFile(moduleIndexPath, progress.RenderModuleRoadmapIndex(p)) },
	})
	for _, module := range progress.AllowedModules() {
		module := module
		path := filepath.Join(paths.moduleRoadmapsDir, module+".md")
		artifacts = append(artifacts, artifact{
			Kind:  "module-roadmap:" + module,
			Path:  path,
			Group: "module-roadmaps",
			Write: func() error { return writeTextFile(path, progress.RenderModuleRoadmapPage(p, module)) },
		})
	}

	for _, dst := range paths.siteProgress {
		dst := dst
		artifacts = append(artifacts, artifact{
			Kind:  "site-progress",
			Path:  dst,
			Group: "site-progress",
			Write: func() error {
				body, err := stableProgressBytes(p)
				if err != nil {
					return err
				}
				return writeBytesFile(dst, body)
			},
		})
	}
	if paths.siteProgressSlim != "" {
		dst := paths.siteProgressSlim
		artifacts = append(artifacts, artifact{
			Kind:  "site-progress-slim",
			Path:  dst,
			Group: "site-progress",
			Write: func() error { return writeSlimProgress(p, dst) },
		})
	}

	return artifacts
}

func WriteWithOptions(stdout io.Writer, root string, opts WriteOptions) error {
	p, err := loadValidProgress(root)
	if err != nil {
		return err
	}
	artifacts := planArtifacts(p, progressPaths(root))
	if opts.DryRun {
		if _, err := fmt.Fprintf(stdout, "progress: dry-run %d artifact(s)\n", len(artifacts)); err != nil {
			return err
		}
		for _, artifact := range artifacts {
			if _, err := fmt.Fprintf(stdout, "progress: dry-run %s %s\n", artifact.Kind, artifact.Path); err != nil {
				return err
			}
		}
		return nil
	}
	return executeArtifacts(stdout, artifacts)
}

func executeArtifacts(stdout io.Writer, artifacts []artifact) error {
	var errs []error
	moduleRoadmapsOK := true
	hasModuleRoadmaps := false
	hasSiteProgress := false

	for _, artifact := range artifacts {
		if artifact.Group == "module-roadmaps" {
			hasModuleRoadmaps = true
		}
		if artifact.Group == "site-progress" {
			hasSiteProgress = true
		}
		if err := artifact.Write(); err != nil {
			if artifact.Group == "module-roadmaps" {
				moduleRoadmapsOK = false
			}
			errs = append(errs, fmt.Errorf("%s %s: %w", artifact.Kind, artifact.Path, err))
			continue
		}
		if artifact.Label != "" {
			fmt.Fprintln(stdout, "progress:", artifact.Label)
		}
	}
	if hasModuleRoadmaps && moduleRoadmapsOK {
		fmt.Fprintln(stdout, "progress: module roadmaps regenerated")
	}
	if hasSiteProgress && len(errs) == 0 {
		fmt.Fprintln(stdout, "progress: site progress data refreshed")
	}
	return joinErrors(stdout, errs)
}

func writeTextFile(path, body string) error {
	return writeBytesFile(path, []byte(body))
}

func writeBytesFile(path string, body []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func stableProgressBytes(p *progress.Progress) ([]byte, error) {
	tmpDir, err := os.MkdirTemp("", "progress-artifact-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)
	tmp := filepath.Join(tmpDir, "progress.json")
	if err := progress.SaveProgress(tmp, p); err != nil {
		return nil, err
	}
	return os.ReadFile(tmp)
}
