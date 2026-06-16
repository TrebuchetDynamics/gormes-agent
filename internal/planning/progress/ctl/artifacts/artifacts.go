// Package artifacts owns generated artifact planning and execution for
// progress write. It depends on progress renderers and filesystem writes, but
// must not load or validate the canonical backlog itself.
package artifacts

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
)

// Options controls progress write execution.
type Options struct {
	DryRun bool
}

// Paths names every generated destination touched by progress write.
type Paths struct {
	DocsIndex          string
	Readme             string
	ContractReadiness  string
	BuilderLoopHandoff string
	AgentQueue         string
	NextSlices         string
	BlockedSlices      string
	UmbrellaCleanup    string
	ProgressSchema     string
	ModuleRoadmapsDir  string
	SiteProgress       []string
}

// Artifact is one planned generated output.
type Artifact struct {
	Kind  string
	Path  string
	Label string
	Group string
	Write func() error
}

// Plan returns the stable list of generated artifacts for progress write.
func Plan(p *progress.Progress, paths Paths) []Artifact {
	markers := []struct {
		path   string
		kind   string
		label  string
		render func(*progress.Progress) string
	}{
		{path: paths.DocsIndex, kind: "docs-full-checklist", label: "_index.md regenerated", render: progress.RenderDocsChecklist},
		{path: paths.Readme, kind: "readme-rollup", label: "README.md regenerated", render: progress.RenderReadmeRollup},
		{path: paths.ContractReadiness, kind: "contract-readiness", label: "contract readiness regenerated", render: progress.RenderContractReadiness},
		{path: paths.BuilderLoopHandoff, kind: "builder-loop-handoff", label: "builder-loop handoff regenerated", render: progress.RenderBuilderLoopHandoff},
		{path: paths.AgentQueue, kind: "agent-queue", label: "agent queue regenerated", render: func(p *progress.Progress) string { return progress.RenderAgentQueue(p, 10) }},
		{path: paths.NextSlices, kind: "next-slices", label: "next slices regenerated", render: func(p *progress.Progress) string { return progress.RenderNextSlices(p, 10) }},
		{path: paths.BlockedSlices, kind: "blocked-slices", label: "blocked slices regenerated", render: progress.RenderBlockedSlices},
		{path: paths.UmbrellaCleanup, kind: "umbrella-cleanup", label: "umbrella cleanup regenerated", render: progress.RenderUmbrellaCleanup},
		{path: paths.ProgressSchema, kind: "progress-schema", label: "progress schema regenerated", render: func(*progress.Progress) string { return progress.RenderProgressSchema() }},
	}

	artifacts := make([]Artifact, 0, len(markers)+len(progress.AllowedModules())+2+len(paths.SiteProgress))
	for _, marker := range markers {
		marker := marker
		body := marker.render(p)
		artifacts = append(artifacts, Artifact{
			Kind:  "marker:" + marker.kind,
			Path:  marker.path,
			Label: marker.label,
			Write: func() error { return rewriteMarker(marker.path, marker.kind, body) },
		})
	}

	moduleIndexPath := filepath.Join(paths.ModuleRoadmapsDir, "_index.md")
	artifacts = append(artifacts, Artifact{
		Kind:  "module-roadmap:index",
		Path:  moduleIndexPath,
		Group: "module-roadmaps",
		Write: func() error { return writeTextFile(moduleIndexPath, progress.RenderModuleRoadmapIndex(p)) },
	})
	for _, module := range progress.AllowedModules() {
		module := module
		path := filepath.Join(paths.ModuleRoadmapsDir, progress.ModuleRoadmapRelPath(module))
		artifacts = append(artifacts, Artifact{
			Kind:  "module-roadmap:" + module,
			Path:  path,
			Group: "module-roadmaps",
			Write: func() error { return writeTextFile(path, progress.RenderModuleRoadmapPage(p, module)) },
		})
	}

	for _, dst := range paths.SiteProgress {
		dst := dst
		artifacts = append(artifacts, Artifact{
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
	return artifacts
}

// WriteWithOptions executes or dry-runs the generated artifact plan.
func WriteWithOptions(stdout io.Writer, p *progress.Progress, paths Paths, opts Options) error {
	artifacts := Plan(p, paths)
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
	return Execute(stdout, artifacts)
}

// Execute writes an artifact plan and reports grouped success/failure output.
func Execute(stdout io.Writer, artifacts []Artifact) error {
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

func rewriteMarker(path, kind, body string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	out, err := progress.ReplaceMarker(string(b), kind, body)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
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

func joinErrors(stdout io.Writer, errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	for _, err := range errs {
		fmt.Fprintln(stdout, "progress:", err)
	}
	return errors.Join(errs...)
}
