// Package workspace owns the filesystem seam for the Progress Control Plane.
// It hides whether the logical backlog is a monolithic progress.json file or
// a split progress.json directory and centralizes the generated artifact paths
// used by cmd/progress.
package workspace

import (
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
)

// Paths names the canonical backlog and generated progress artifacts for a
// repository root. The canonical backlog is always ProgressJSON; that path may
// be either a regular file or a split-layout directory.
type Paths struct {
	ProgressJSON       string
	Readme             string
	DocsIndex          string
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

// Workspace is the caller-facing filesystem interface for progress tooling.
type Workspace struct {
	Root  string
	Paths Paths
}

// New returns the workspace rooted at root.
func New(root string) Workspace {
	return Workspace{Root: root, Paths: PathsForRoot(root)}
}

// PathsForRoot returns the generated progress paths for root.
func PathsForRoot(root string) Paths {
	buildingGormes := filepath.Join(root, "webpages", "docs", "content", "building-gormes")
	builderLoopDir := filepath.Join(buildingGormes, "builder-loop")
	return Paths{
		ProgressJSON:       filepath.Join(buildingGormes, "architecture_plan", "progress.json"),
		Readme:             filepath.Join(root, "README.md"),
		DocsIndex:          filepath.Join(buildingGormes, "architecture_plan", "_index.md"),
		ContractReadiness:  filepath.Join(buildingGormes, "contract-readiness.md"),
		BuilderLoopHandoff: filepath.Join(builderLoopDir, "builder-loop-handoff.md"),
		AgentQueue:         filepath.Join(builderLoopDir, "agent-queue.md"),
		NextSlices:         filepath.Join(builderLoopDir, "next-slices.md"),
		BlockedSlices:      filepath.Join(builderLoopDir, "blocked-slices.md"),
		UmbrellaCleanup:    filepath.Join(builderLoopDir, "umbrella-cleanup.md"),
		ProgressSchema:     filepath.Join(builderLoopDir, "progress-schema.md"),
		ModuleRoadmapsDir:  filepath.Join(buildingGormes, "modules"),
		// Site mirrors: now empty. The active Astro landing page has no
		// progress.json consumer, and the retired Go renderer no longer
		// receives generated progress mirrors.
		SiteProgress: nil,
	}
}

// CanonicalSource returns the single logical backlog path. The returned path
// may be a monolithic file or a split directory; progress.Load handles either.
// The historical progress.split staging directory is intentionally ignored.
func (w Workspace) CanonicalSource() string {
	return w.Paths.ProgressJSON
}

// Load reads the logical backlog from the canonical source.
func (w Workspace) Load() (*progress.Progress, error) {
	return progress.Load(w.CanonicalSource())
}

// LoadValid reads and validates the logical backlog from the canonical source.
func (w Workspace) LoadValid() (*progress.Progress, error) {
	p, err := w.Load()
	if err != nil {
		return nil, err
	}
	if err := progress.Validate(p); err != nil {
		return nil, err
	}
	return p, nil
}

// Save writes the logical backlog back to the canonical source, preserving the
// current on-disk layout when the source is a split directory.
func (w Workspace) Save(p *progress.Progress) error {
	return progress.SaveProgress(w.CanonicalSource(), p)
}

// EmitBytes returns stable monolithic JSON bytes for the logical backlog. It
// centralizes the split-safe temp re-encode recipe used by CLI and tests.
func (w Workspace) EmitBytes() ([]byte, error) {
	p, err := w.LoadValid()
	if err != nil {
		return nil, err
	}
	tmpDir, err := os.MkdirTemp("", "progress-emit-")
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
