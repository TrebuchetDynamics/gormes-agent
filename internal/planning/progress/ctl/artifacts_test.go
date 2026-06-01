package progressctl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
)

func TestPlanArtifactsListsStableGeneratedOutputs(t *testing.T) {
	root := t.TempDir()
	paths := progressPaths(root)
	artifacts := planArtifacts(c2Fixture(), paths)

	for _, want := range []struct {
		kind string
		path string
	}{
		{kind: "marker:docs-full-checklist", path: paths.docsIndex},
		{kind: "marker:progress-schema", path: paths.progressSchema},
		{kind: "module-roadmap:index", path: filepath.Join(paths.moduleRoadmapsDir, "_index.md")},
		{kind: "module-roadmap:providers", path: filepath.Join(paths.moduleRoadmapsDir, progress.ModuleRoadmapRelPath(progress.ModuleProviders))},
		{kind: "site-progress-slim", path: paths.siteProgressSlim},
	} {
		if !artifactPlanContains(artifacts, want.kind, want.path) {
			t.Fatalf("artifact plan missing kind=%q path=%q\nplan=%s", want.kind, want.path, artifactPlanDebug(artifacts))
		}
	}
}

func TestWriteDryRunListsArtifactsAndDoesNotWrite(t *testing.T) {
	root := t.TempDir()
	seedMonolith(t, root, c2Fixture())
	seedWriteMarkerFiles(t, root)
	paths := progressPaths(root)
	beforeDocsIndex := mustReadFile(t, paths.docsIndex)
	moduleIndex := filepath.Join(paths.moduleRoadmapsDir, "_index.md")

	var out bytes.Buffer
	if err := WriteWithOptions(&out, root, WriteOptions{DryRun: true}); err != nil {
		t.Fatalf("WriteWithOptions dry-run: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"progress: dry-run",
		"marker:docs-full-checklist",
		paths.docsIndex,
		"module-roadmap:providers",
		filepath.Join(paths.moduleRoadmapsDir, progress.ModuleRoadmapRelPath(progress.ModuleProviders)),
		"site-progress-slim",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("dry-run output missing %q:\n%s", want, got)
		}
	}
	if after := mustReadFile(t, paths.docsIndex); after != beforeDocsIndex {
		t.Fatalf("dry-run rewrote docs index\nbefore=%q\nafter=%q", beforeDocsIndex, after)
	}
	if _, err := os.Stat(moduleIndex); !os.IsNotExist(err) {
		t.Fatalf("dry-run must not create module roadmap index, stat err=%v", err)
	}
}

func TestWriteExecutesArtifactPlanAndReportsArtifactFailures(t *testing.T) {
	root := t.TempDir()
	seedMonolith(t, root, c2Fixture())
	seedWriteMarkerFiles(t, root)
	paths := progressPaths(root)

	if err := Write(&bytes.Buffer{}, root); err != nil {
		t.Fatalf("Write: %v", err)
	}
	providers := mustReadFile(t, filepath.Join(paths.moduleRoadmapsDir, progress.ModuleRoadmapRelPath(progress.ModuleProviders)))
	if !strings.Contains(providers, "# Providers Module Roadmap") {
		t.Fatalf("providers module roadmap not written from artifact plan:\n%s", providers)
	}

	brokenRoot := t.TempDir()
	seedMonolith(t, brokenRoot, c2Fixture())
	seedWriteMarkerFiles(t, brokenRoot)
	brokenPaths := progressPaths(brokenRoot)
	if err := os.Remove(brokenPaths.nextSlices); err != nil {
		t.Fatalf("remove next-slices marker file: %v", err)
	}
	var out bytes.Buffer
	if err := Write(&out, brokenRoot); err == nil {
		t.Fatal("Write with missing next-slices marker must fail")
	} else if !strings.Contains(err.Error(), "marker:next-slices") || !strings.Contains(err.Error(), brokenPaths.nextSlices) {
		t.Fatalf("error must include artifact kind and path, got %v", err)
	}
}

func artifactPlanContains(artifacts []artifact, kind, path string) bool {
	for _, artifact := range artifacts {
		if artifact.Kind == kind && artifact.Path == path {
			return true
		}
	}
	return false
}

func artifactPlanDebug(artifacts []artifact) string {
	var b strings.Builder
	for _, artifact := range artifacts {
		b.WriteString(artifact.Kind)
		b.WriteByte(' ')
		b.WriteString(artifact.Path)
		b.WriteByte('\n')
	}
	return b.String()
}
