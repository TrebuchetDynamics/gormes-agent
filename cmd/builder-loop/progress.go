package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
)

func runProgress(deps cliDeps, root string, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("%w\n%s", errParse, subUsage["progress"])
	}

	switch args[0] {
	case "validate":
		return validateProgress(deps, root)
	case "write":
		return writeProgress(deps, root)
	default:
		return fmt.Errorf("%w\n%s", errParse, subUsage["progress"])
	}
}

func validateProgress(deps cliDeps, root string) error {
	p, err := loadValidProgress(root)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(deps.stdout, "progress: validated %d phases\n", len(p.Phases))
	return err
}

// progressMarker describes one place where `progress write` regenerates a
// generated section of a tracked file. New markers are added by appending
// to the table in writeProgress; the loop that drives this is intentionally
// dumb so order, error handling, and the operator-facing log line for each
// marker stay consistent.
type progressMarker struct {
	pathOf func(progressPathSet) string
	kind   string
	label  string
	render func(*progress.Progress) string
}

func writeProgress(deps cliDeps, root string) error {
	p, err := loadValidProgress(root)
	if err != nil {
		return err
	}

	paths := progressPaths(root)

	markers := []progressMarker{
		{
			pathOf: func(s progressPathSet) string { return s.docsIndex },
			kind:   "docs-full-checklist",
			label:  "_index.md regenerated",
			render: progress.RenderDocsChecklist,
		},
		{
			pathOf: func(s progressPathSet) string { return s.readme },
			kind:   "readme-rollup",
			label:  "README.md regenerated",
			render: progress.RenderReadmeRollup,
		},
		{
			pathOf: func(s progressPathSet) string { return s.contractReadiness },
			kind:   "contract-readiness",
			label:  "contract readiness regenerated",
			render: progress.RenderContractReadiness,
		},
		{
			pathOf: func(s progressPathSet) string { return s.builderLoopHandoff },
			kind:   "builder-loop-handoff",
			label:  "builder-loop handoff regenerated",
			render: progress.RenderBuilderLoopHandoff,
		},
		{
			pathOf: func(s progressPathSet) string { return s.agentQueue },
			kind:   "agent-queue",
			label:  "agent queue regenerated",
			render: func(p *progress.Progress) string { return progress.RenderAgentQueue(p, 10) },
		},
		{
			pathOf: func(s progressPathSet) string { return s.nextSlices },
			kind:   "next-slices",
			label:  "next slices regenerated",
			render: func(p *progress.Progress) string { return progress.RenderNextSlices(p, 10) },
		},
		{
			pathOf: func(s progressPathSet) string { return s.blockedSlices },
			kind:   "blocked-slices",
			label:  "blocked slices regenerated",
			render: progress.RenderBlockedSlices,
		},
		{
			pathOf: func(s progressPathSet) string { return s.umbrellaCleanup },
			kind:   "umbrella-cleanup",
			label:  "umbrella cleanup regenerated",
			render: progress.RenderUmbrellaCleanup,
		},
		{
			pathOf: func(s progressPathSet) string { return s.progressSchema },
			kind:   "progress-schema",
			label:  "progress schema regenerated",
			render: func(*progress.Progress) string { return progress.RenderProgressSchema() },
		},
	}

	var errs []error
	for _, m := range markers {
		if err := rewriteProgressMarker(m.pathOf(paths), m.kind, m.render(p)); err != nil {
			errs = append(errs, err)
		} else {
			fmt.Fprintln(deps.stdout, "progress:", m.label)
		}
	}

	// Site progress is a raw file mirror, not a markered rewrite; kept
	// out of the marker table on purpose.
	if err := syncProgressFile(paths.progressJSON, paths.siteProgress); err != nil {
		errs = append(errs, err)
	} else {
		fmt.Fprintln(deps.stdout, "progress: site progress data refreshed")
	}
	return joinProgressErrors(deps, errs)
}

func loadValidProgress(root string) (*progress.Progress, error) {
	p, err := progress.Load(progressPaths(root).progressJSON)
	if err != nil {
		return nil, err
	}
	if err := progress.Validate(p); err != nil {
		return nil, err
	}
	return p, nil
}

type progressPathSet struct {
	progressJSON       string
	readme             string
	docsIndex          string
	contractReadiness  string
	builderLoopHandoff string
	agentQueue         string
	nextSlices         string
	blockedSlices      string
	umbrellaCleanup    string
	progressSchema     string
	siteProgress       string
}

func progressPaths(root string) progressPathSet {
	buildingGormes := filepath.Join(root, "docs", "content", "building-gormes")
	builderLoopDir := filepath.Join(buildingGormes, "builder-loop")
	return progressPathSet{
		progressJSON:       filepath.Join(buildingGormes, "architecture_plan", "progress.json"),
		readme:             filepath.Join(root, "README.md"),
		docsIndex:          filepath.Join(buildingGormes, "architecture_plan", "_index.md"),
		contractReadiness:  filepath.Join(buildingGormes, "contract-readiness.md"),
		builderLoopHandoff: filepath.Join(builderLoopDir, "builder-loop-handoff.md"),
		agentQueue:         filepath.Join(builderLoopDir, "agent-queue.md"),
		nextSlices:         filepath.Join(builderLoopDir, "next-slices.md"),
		blockedSlices:      filepath.Join(builderLoopDir, "blocked-slices.md"),
		umbrellaCleanup:    filepath.Join(builderLoopDir, "umbrella-cleanup.md"),
		progressSchema:     filepath.Join(builderLoopDir, "progress-schema.md"),
		siteProgress:       filepath.Join(root, "www.gormes.ai", "internal", "site", "data", "progress.json"),
	}
}

func rewriteProgressMarker(path, kind, body string) error {
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

func syncProgressFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(dst), err)
	}
	if err := os.WriteFile(dst, b, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}

func joinProgressErrors(deps cliDeps, errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	for _, err := range errs {
		fmt.Fprintln(deps.stdout, "progress:", err)
	}
	return errors.Join(errs...)
}
