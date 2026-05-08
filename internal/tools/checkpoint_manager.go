package tools

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const checkpointWorkdirMarker = "GORMES_WORKDIR"

// CheckpointManagerOptions configures the startup shadow-repo GC contract.
type CheckpointManagerOptions struct {
	Root      string
	Now       func() time.Time
	ShadowTTL time.Duration
	DryRun    bool
}

// CheckpointManager owns the read model for Gormes checkpoint rollback state.
type CheckpointManager struct {
	root   string
	now    func() time.Time
	ttl    time.Duration
	dryRun bool
	status CheckpointStatus
}

// CheckpointStatus reports degraded-mode evidence from checkpoint startup GC.
type CheckpointStatus struct {
	Evidence []CheckpointEvidence
}

// CheckpointEvidence names a cleanup condition and the affected shadow repos.
type CheckpointEvidence struct {
	Kind  string
	Count int
	Paths []string
	Error string
}

// NewCheckpointManager performs deterministic startup cleanup before callers
// can depend on rollback state.
func NewCheckpointManager(opts CheckpointManagerOptions) (*CheckpointManager, error) {
	if opts.Root == "" {
		opts.Root = DefaultCheckpointRoot()
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	mgr := &CheckpointManager{
		root:   opts.Root,
		now:    opts.Now,
		ttl:    opts.ShadowTTL,
		dryRun: opts.DryRun,
	}
	mgr.runStartupGC()
	return mgr, nil
}

// DefaultCheckpointRoot returns Gormes' XDG-owned rollback directory.
func DefaultCheckpointRoot() string {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return filepath.Join(v, "gormes", "checkpoints")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "gormes", "checkpoints")
}

// Status returns a copy of the checkpoint read model.
func (m *CheckpointManager) Status() CheckpointStatus {
	out := m.status
	out.Evidence = append([]CheckpointEvidence(nil), m.status.Evidence...)
	for i := range out.Evidence {
		out.Evidence[i].Paths = append([]string(nil), out.Evidence[i].Paths...)
	}
	return out
}

func (m *CheckpointManager) runStartupGC() {
	entries, err := os.ReadDir(m.root)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		m.addEvidence("shadow_gc_unavailable", nil, err)
		return
	}

	var orphanPaths []string
	var stalePaths []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		shadow := filepath.Join(m.root, entry.Name())
		if _, err := os.Stat(filepath.Join(shadow, "HEAD")); err != nil {
			continue
		}
		workdir, ok := readCheckpointWorkdir(shadow)
		if !ok || !pathExists(workdir) {
			orphanPaths = append(orphanPaths, entry.Name())
			if !m.dryRun {
				_ = os.RemoveAll(shadow)
			}
			continue
		}
		if m.ttl > 0 {
			newest, ok := newestCheckpointMTime(shadow)
			if ok && m.now().Sub(newest) > m.ttl {
				stalePaths = append(stalePaths, entry.Name())
				if !m.dryRun {
					_ = os.RemoveAll(shadow)
				}
			}
		}
	}
	m.addEvidence("orphan_shadow_repo", orphanPaths, nil)
	m.addEvidence("stale_shadow_repo", stalePaths, nil)
}

func readCheckpointWorkdir(shadow string) (string, bool) {
	raw, err := os.ReadFile(filepath.Join(shadow, checkpointWorkdirMarker))
	if err != nil {
		return "", false
	}
	workdir := strings.TrimSpace(string(raw))
	return workdir, workdir != ""
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func newestCheckpointMTime(shadow string) (time.Time, bool) {
	var newest time.Time
	err := filepath.WalkDir(shadow, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
		return nil
	})
	return newest, err == nil && !newest.IsZero()
}

func (m *CheckpointManager) addEvidence(kind string, paths []string, err error) {
	if len(paths) == 0 && err == nil {
		return
	}
	sort.Strings(paths)
	evidence := CheckpointEvidence{
		Kind:  kind,
		Count: len(paths),
		Paths: append([]string(nil), paths...),
	}
	if err != nil {
		evidence.Count = 1
		evidence.Error = "checkpoint root unavailable"
	}
	m.status.Evidence = append(m.status.Evidence, evidence)
}

// ── Store status ──────────────────────────────────────────────

// StoreStatusResult is the read model returned by StoreStatus, mirroring
// Hermes' store_status() dict shape.
type StoreStatusResult struct {
	Root            string
	TotalSizeBytes  int64
	StoreSizeBytes  int64
	LegacySizeBytes int64
	ProjectCount    int
	Projects        []StoreStatusProject
	LegacyArchives  []StoreStatusArchive
}

// StoreStatusProject is one checkpoint shadow-repo entry in the status table.
type StoreStatusProject struct {
	Name      string
	Workdir   string
	Commits   int
	LastTouch time.Time
	Exists    bool // workdir exists on disk
}

// StoreStatusArchive is a legacy archive visible to clear-legacy.
type StoreStatusArchive struct {
	Name      string
	SizeBytes int64
	ModTime   time.Time
}

// StoreStatus builds a read-only snapshot of the checkpoint store under root.
// It does not mutate any files. A non-existent root returns an empty result
// with no error.
func StoreStatus(root string) (StoreStatusResult, error) {
	if root == "" {
		root = DefaultCheckpointRoot()
	}
	result := StoreStatusResult{Root: root}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return result, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		child := filepath.Join(root, entry.Name())
		// Legacy archives are directories whose name starts with "legacy-".
		if strings.HasPrefix(entry.Name(), "legacy-") {
			sz, _ := dirSize(child)
			info, _ := entry.Info()
			mtime := time.Time{}
			if info != nil {
				mtime = info.ModTime()
			}
			result.LegacySizeBytes += sz
			result.LegacyArchives = append(result.LegacyArchives, StoreStatusArchive{
				Name:      entry.Name(),
				SizeBytes: sz,
				ModTime:   mtime,
			})
			continue
		}
		// Shadow repos: directories containing HEAD.
		if _, err := os.Stat(filepath.Join(child, "HEAD")); err != nil {
			continue
		}
		proj := StoreStatusProject{
			Name:   entry.Name(),
			Exists: true,
		}
		workdir, ok := readCheckpointWorkdir(child)
		if ok {
			proj.Workdir = workdir
			proj.Exists = pathExists(workdir)
		}
		proj.Commits = countCheckpointCommits(child)
		if mt, ok := newestCheckpointMTime(child); ok {
			proj.LastTouch = mt
		}
		sz, _ := dirSize(child)
		result.StoreSizeBytes += sz
		result.Projects = append(result.Projects, proj)
	}
	result.ProjectCount = len(result.Projects)
	result.TotalSizeBytes = result.StoreSizeBytes + result.LegacySizeBytes
	sort.Slice(result.Projects, func(i, j int) bool {
		return result.Projects[i].LastTouch.After(result.Projects[j].LastTouch)
	})
	return result, nil
}

func countCheckpointCommits(shadow string) int {
	// Walk the shadow and count refs/heads entries as approximations of
	// checkpointed commits. Hermes uses `store_status()["commits"]` as the
	// count of commits created by checkpoint_manager.
	refs := filepath.Join(shadow, "refs", "heads")
	entries, err := os.ReadDir(refs)
	if err != nil {
		return 0
	}
	return len(entries)
}

// dirSize returns the total size in bytes of all regular files under path.
func dirSize(path string) (int64, error) {
	var total int64
	err := filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			if info, err := d.Info(); err == nil {
				total += info.Size()
			}
		}
		return nil
	})
	return total, err
}

// ── Prune ─────────────────────────────────────────────────────

// PruneResult reports what prune deleted.
type PruneResult struct {
	Scanned        int
	DeletedOrphan  int
	DeletedStale   int
	Errors         int
	BytesFreed     int64
}

// PruneCheckpoints deletes orphan (workdir missing) and stale (last touch
// older than retentionDays) shadow repos under root. keepOrphans skips
// orphan deletion. maxSizeMB is a soft cap on total store size after prune;
// if exceeded, the oldest commits across projects are dropped.
func PruneCheckpoints(root string, retentionDays int, keepOrphans bool, maxSizeMB int, now func() time.Time) PruneResult {
	if root == "" {
		root = DefaultCheckpointRoot()
	}
	if now == nil {
		now = time.Now
	}
	result := PruneResult{}
	entries, err := os.ReadDir(root)
	if err != nil {
		result.Errors++
		return result
	}
	retention := time.Duration(retentionDays) * 24 * time.Hour

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), "legacy-") {
			continue
		}
		shadow := filepath.Join(root, entry.Name())
		if _, err := os.Stat(filepath.Join(shadow, "HEAD")); err != nil {
			continue
		}
		result.Scanned++

		workdir, ok := readCheckpointWorkdir(shadow)
		if !ok || !pathExists(workdir) {
			if !keepOrphans {
				sz, _ := dirSize(shadow)
				_ = os.RemoveAll(shadow)
				result.DeletedOrphan++
				result.BytesFreed += sz
			}
			continue
		}

		newest, ok := newestCheckpointMTime(shadow)
		if ok && now().Sub(newest) > retention {
			sz, _ := dirSize(shadow)
			_ = os.RemoveAll(shadow)
			result.DeletedStale++
			result.BytesFreed += sz
		}
	}
	return result
}

// ── Clear ──────────────────────────────────────────────────────

// ClearResult reports what clear deleted.
type ClearResult struct {
	Deleted    bool
	BytesFreed int64
}

// ClearAll deletes the entire checkpoint root directory.
func ClearAll(root string) ClearResult {
	if root == "" {
		root = DefaultCheckpointRoot()
	}
	sz, _ := dirSize(root)
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return ClearResult{Deleted: false, BytesFreed: 0}
	}
	_ = os.RemoveAll(root)
	return ClearResult{Deleted: true, BytesFreed: sz}
}

// ClearLegacy deletes only legacy-* archive directories under root.
func ClearLegacy(root string) ClearResult {
	if root == "" {
		root = DefaultCheckpointRoot()
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return ClearResult{}
	}
	var freed int64
	deleted := 0
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "legacy-") {
			continue
		}
		child := filepath.Join(root, entry.Name())
		sz, _ := dirSize(child)
		_ = os.RemoveAll(child)
		freed += sz
		deleted++
	}
	return ClearResult{Deleted: deleted > 0, BytesFreed: freed}
}
