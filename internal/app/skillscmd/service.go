package skillscmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	skillruntime "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	profilecli "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/profile"
	skillcmdruntime "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/skillscmd"
)

type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

type ProfileSyncSeams struct {
	BundledRoot func() string
	Profiles    func() ([]skillruntime.SkillProfileRoot, error)
	Sync        func(context.Context, skillruntime.BundledSkillProfileSyncRequest) (skillruntime.BundledSkillProfileSyncReport, error)
}

type SyncOptions struct {
	JSON  bool
	Build BuildProvenance
}

// skillsSyncReportJSON is the wire shape for `skills sync --json`.
// Fleet automation rolling skills across profiles parses this to
// audit per-profile counts. Build provenance leads — same convention
// as the rest of the `--json` arc.
type skillsSyncReportJSON struct {
	Build     BuildProvenance         `json:"build"`
	Summaries []skillsSyncSummaryJSON `json:"summaries"`
}

type skillsSyncSummaryJSON struct {
	Profile   string `json:"profile"`
	Added     int    `json:"added"`
	Unchanged int    `json:"unchanged"`
	Conflicts int    `json:"conflicts"`
	Failed    int    `json:"failed"`
}

func ListInstalledSkills(opts skillruntime.ListOptions, disabled map[string]struct{}) []skillruntime.SkillRow {
	cfg, err := config.Load(nil)
	if err != nil {
		return nil
	}
	externalRoots, _ := cfg.ExternalSkillsDirs()
	opts.ExternalRoots = externalRoots
	return skillruntime.ListInstalledSkillsFromRoots(cfg.SkillsRoot(), skillruntime.BundledRoot(), opts, disabled)
}

func URLInstallDeps() skillcmdruntime.SkillsURLInstallDeps {
	return skillcmdruntime.SkillsURLInstallDeps{
		Fetcher: httpSkillFetcher{client: &http.Client{Timeout: 30 * time.Second}},
		Store:   configSkillStore{},
	}
}

func CommandOptionsForConfig(cfg config.Config) gateway.SkillsCommandOptions {
	externalRoots, _ := cfg.ExternalSkillsDirs()
	return gateway.SkillsCommandOptions{
		SkillsRoot:   cfg.SkillsRoot(),
		BundledRoot:  skillruntime.BundledRoot(),
		ExternalDirs: externalRoots,
		URLInstall: skillruntime.URLInstallPolicy{
			Fetcher: httpSkillFetcher{client: &http.Client{Timeout: 30 * time.Second}},
			Store:   configSkillStore{root: cfg.SkillsRoot()},
		},
	}
}

func RunProfileSync(ctx context.Context, out io.Writer, seams ProfileSyncSeams, opts SyncOptions) error {
	seams = withDefaultProfileSyncSeams(seams)
	profiles, err := seams.Profiles()
	if err != nil {
		return err
	}
	report, err := seams.Sync(ctx, skillruntime.BundledSkillProfileSyncRequest{
		BundledRoot: seams.BundledRoot(),
		Profiles:    profiles,
	})
	if err != nil {
		return err
	}
	return WriteProfileSyncReport(out, report, opts.JSON, opts.Build)
}

func WriteProfileSyncReport(out io.Writer, report skillruntime.BundledSkillProfileSyncReport, jsonOut bool, build BuildProvenance) error {
	if out == nil {
		out = io.Discard
	}
	if jsonOut {
		wire := skillsSyncReportJSON{
			Build:     build,
			Summaries: make([]skillsSyncSummaryJSON, len(report.Summaries)),
		}
		for i, s := range report.Summaries {
			wire.Summaries[i] = skillsSyncSummaryJSON{
				Profile:   s.Profile,
				Added:     s.Added,
				Unchanged: s.Unchanged,
				Conflicts: s.Conflicts,
				Failed:    s.Failed,
			}
		}
		body, err := json.MarshalIndent(wire, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(body))
		return nil
	}
	for _, summary := range report.Summaries {
		fmt.Fprintf(out, "%s\tadded=%d unchanged=%d conflicts=%d failed=%d\n", summary.Profile, summary.Added, summary.Unchanged, summary.Conflicts, summary.Failed)
	}
	return nil
}

func withDefaultProfileSyncSeams(seams ProfileSyncSeams) ProfileSyncSeams {
	if seams.BundledRoot == nil {
		seams.BundledRoot = skillruntime.BundledRoot
	}
	if seams.Profiles == nil {
		seams.Profiles = DefaultProfileRoots
	}
	if seams.Sync == nil {
		seams.Sync = skillruntime.SyncBundledSkillsToProfiles
	}
	return seams
}

func DefaultProfileRoots() ([]skillruntime.SkillProfileRoot, error) {
	baseHome := config.GormesBaseHome()
	names := listKnownProfileNames(baseHome)
	activePath := filepath.Join(baseHome, "active_profile")
	if active, err := profilecli.ReadActiveProfile(activePath); err == nil {
		names = append(names, strings.TrimSpace(active))
	} else if !errors.Is(err, profilecli.ErrActiveProfileUnset) {
		return nil, err
	}

	seen := map[string]bool{}
	out := make([]skillruntime.SkillProfileRoot, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true

		root, err := profilecli.ResolveProfileRuntimeRoot(baseHome, name)
		if err != nil {
			return nil, err
		}
		out = append(out, skillruntime.SkillProfileRoot{Name: name, Root: root})
	}
	return out, nil
}

func listKnownProfileNames(baseHome string) []string {
	known := []string{config.DefaultProfileID}
	seen := map[string]struct{}{config.DefaultProfileID: {}}
	addName := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, dup := seen[name]; dup {
			return
		}
		if err := profilecli.ValidateProfileName(name); err != nil {
			return
		}
		seen[name] = struct{}{}
		known = append(known, name)
	}
	if cfg, err := config.Load(nil); err == nil {
		for name := range cfg.Profiles {
			addName(name)
		}
	}
	entries, err := os.ReadDir(filepath.Join(baseHome, "profiles"))
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				addName(entry.Name())
			}
		}
	}
	return known
}

type httpSkillFetcher struct {
	client *http.Client
}

func (f httpSkillFetcher) Fetch(ctx context.Context, rawURL string) ([]byte, error) {
	client := f.client
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch skill: HTTP %d", resp.StatusCode)
	}
	limited := io.LimitReader(resp.Body, int64(skillruntime.DefaultMaxDocumentBytes)+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(body) > skillruntime.DefaultMaxDocumentBytes {
		return nil, fmt.Errorf("skill document too large: %d > %d bytes", len(body), skillruntime.DefaultMaxDocumentBytes)
	}
	return body, nil
}

type configSkillStore struct {
	root string
}

func (s configSkillStore) ActiveDir() string {
	if strings.TrimSpace(s.root) != "" {
		return filepath.Join(s.root, "active")
	}
	cfg, err := config.Load(nil)
	if err != nil {
		return filepath.Join(config.GormesHome(), "skills", "active")
	}
	return filepath.Join(cfg.SkillsRoot(), "active")
}

func (s configSkillStore) WriteSkill(_ context.Context, dir string, file string, body []byte) (string, error) {
	if file != "SKILL.md" {
		return "", fmt.Errorf("unsupported skill file %q", file)
	}
	activeDir := s.ActiveDir()
	target := filepath.Join(dir, file)
	if !pathWithin(activeDir, target) {
		return "", fmt.Errorf("skill install target escapes active skills root")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(dir, ".SKILL.md-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	if err := os.Rename(tmpPath, target); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	return target, nil
}

func pathWithin(root, target string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}
