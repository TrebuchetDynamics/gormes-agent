package main

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

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
	"github.com/spf13/cobra"
)

// skillsSyncReportJSON is the wire shape for `skills sync --json`.
// Fleet automation rolling skills across profiles parses this to
// audit per-profile counts. Build provenance leads — same convention
// as the rest of the `--json` arc.
type skillsSyncReportJSON struct {
	Build     buildProvenanceJSON     `json:"build"`
	Summaries []skillsSyncSummaryJSON `json:"summaries"`
}

type skillsSyncSummaryJSON struct {
	Profile   string `json:"profile"`
	Added     int    `json:"added"`
	Unchanged int    `json:"unchanged"`
	Conflicts int    `json:"conflicts"`
	Failed    int    `json:"failed"`
}

type skillsProfileSyncSeams struct {
	BundledRoot func() string
	Profiles    func() ([]skills.SkillProfileRoot, error)
	Sync        func(context.Context, skills.BundledSkillProfileSyncRequest) (skills.BundledSkillProfileSyncReport, error)
}

func newSkillsCommand() *cobra.Command {
	return newSkillsCommandWithProfileSync(skillsProfileSyncSeams{})
}

func newSkillsCommandWithProfileSync(syncSeams skillsProfileSyncSeams) *cobra.Command {
	cmd := cli.NewSkillsCommand(cli.SkillsCommandDeps{
		ListInstalledSkills: func(opts skills.ListOptions, disabled map[string]struct{}) []skills.SkillRow {
			cfg, err := config.Load(nil)
			if err != nil {
				return nil
			}
			externalRoots, _ := cfg.ExternalSkillsDirs()
			opts.ExternalRoots = externalRoots
			return skills.ListInstalledSkillsFromRoots(cfg.SkillsRoot(), skills.BundledRoot(), opts, disabled)
		},
		DisabledSkills: func(string) map[string]struct{} { return nil },
		URLInstall: cli.SkillsURLInstallDeps{
			Fetcher: httpSkillFetcher{client: &http.Client{Timeout: 30 * time.Second}},
			Store:   configSkillStore{},
		},
		BuildProvenance: func() any { return newBuildProvenance() },
	})
	cmd.AddCommand(newSkillsSyncCommand(syncSeams))
	cmd.AddCommand(newSkillsRowBackedCommands()...)
	return cmd
}

func newSkillsRowBackedCommands() []*cobra.Command {
	return []*cobra.Command{
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "browse",
			Short: "Browse the Hermes skills hub",
			Row:   hermesSkillsRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "search <query>",
			Short: "Search the Hermes skills hub",
			Row:   hermesSkillsRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "inspect <name>",
			Short: "Inspect a skill manifest",
			Row:   hermesSkillsRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "check",
			Short: "Check installed skill health",
			Row:   hermesSkillsRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "update <name>",
			Short: "Update an installed skill",
			Row:   hermesSkillsRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "audit",
			Short: "Audit installed skills",
			Row:   hermesSkillsRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:         "uninstall <name>",
			Short:       "Uninstall a skill",
			Row:         hermesSkillsRow,
			Destructive: true,
			FlagSet:     hermesUnavailableYesFlag,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:         "reset",
			Short:       "Reset installed skills",
			Row:         hermesSkillsRow,
			Destructive: true,
			FlagSet:     hermesUnavailableYesFlag,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "publish <path>",
			Short: "Publish a skill",
			Row:   hermesSkillsRow,
		}),
		newHermesUnavailableParent(
			"snapshot",
			"Manage skill snapshots",
			newHermesUnavailableCommand(hermesUnavailableCommandSpec{
				Use:   "export",
				Short: "Export a skill snapshot",
				Row:   hermesSkillsRow,
			}),
			newHermesUnavailableCommand(hermesUnavailableCommandSpec{
				Use:   "import <path>",
				Short: "Import a skill snapshot",
				Row:   hermesSkillsRow,
			}),
		),
		newHermesUnavailableParent(
			"tap",
			"Manage skill taps",
			newHermesUnavailableCommand(hermesUnavailableCommandSpec{
				Use:     "list",
				Aliases: []string{"ls"},
				Short:   "List configured skill taps",
				Row:     hermesSkillsRow,
			}),
			newHermesUnavailableCommand(hermesUnavailableCommandSpec{
				Use:   "add <url>",
				Short: "Add a skill tap",
				Row:   hermesSkillsRow,
			}),
			newHermesUnavailableCommand(hermesUnavailableCommandSpec{
				Use:         "remove <name>",
				Aliases:     []string{"rm"},
				Short:       "Remove a skill tap",
				Row:         hermesSkillsRow,
				Destructive: true,
				FlagSet:     hermesUnavailableYesFlag,
			}),
		),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "config",
			Short: "Show skill hub configuration",
			Row:   hermesSkillsRow,
		}),
	}
}

func newSkillsSyncCommand(seams skillsProfileSyncSeams) *cobra.Command {
	if seams.BundledRoot == nil {
		seams.BundledRoot = skills.BundledRoot
	}
	if seams.Profiles == nil {
		seams.Profiles = defaultSkillSyncProfiles
	}
	if seams.Sync == nil {
		seams.Sync = skills.SyncBundledSkillsToProfiles
	}

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync bundled skills into all configured profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			profiles, err := seams.Profiles()
			if err != nil {
				return err
			}
			report, err := seams.Sync(cmd.Context(), skills.BundledSkillProfileSyncRequest{
				BundledRoot: seams.BundledRoot(),
				Profiles:    profiles,
			})
			if err != nil {
				return err
			}
			asJSON, _ := cmd.Flags().GetBool("json")
			if asJSON {
				wire := skillsSyncReportJSON{
					Build:     newBuildProvenance(),
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
				body, marshalErr := json.MarshalIndent(wire, "", "  ")
				if marshalErr != nil {
					return marshalErr
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(body))
				return nil
			}
			for _, summary := range report.Summaries {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\tadded=%d unchanged=%d conflicts=%d failed=%d\n", summary.Profile, summary.Added, summary.Unchanged, summary.Conflicts, summary.Failed)
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: `{build, summaries: [{profile, added, unchanged, conflicts, failed}]}`")
	return cmd
}

func skillsCommandOptionsForConfig(cfg config.Config) gateway.SkillsCommandOptions {
	externalRoots, _ := cfg.ExternalSkillsDirs()
	return gateway.SkillsCommandOptions{
		SkillsRoot:   cfg.SkillsRoot(),
		BundledRoot:  skills.BundledRoot(),
		ExternalDirs: externalRoots,
		URLInstall: skills.URLInstallPolicy{
			Fetcher: httpSkillFetcher{client: &http.Client{Timeout: 30 * time.Second}},
			Store:   configSkillStore{root: cfg.SkillsRoot()},
		},
	}
}

func defaultSkillSyncProfiles() ([]skills.SkillProfileRoot, error) {
	names, err := defaultListKnownProfiles()
	if err != nil {
		return nil, err
	}
	activePath := filepath.Join(config.GormesHome(), "active_profile")
	if active, err := cli.ReadActiveProfile(activePath); err == nil {
		names = append(names, strings.TrimSpace(active))
	} else if !errors.Is(err, cli.ErrActiveProfileUnset) {
		return nil, err
	}

	seen := map[string]bool{}
	out := make([]skills.SkillProfileRoot, 0, len(names))
	xdgRoot := filepath.Dir(config.GormesHome())
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		root, err := cli.ResolveProfileRoot(name, xdgRoot)
		if err != nil {
			return nil, err
		}
		out = append(out, skills.SkillProfileRoot{Name: name, Root: root})
	}
	return out, nil
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
	limited := io.LimitReader(resp.Body, int64(skills.DefaultMaxDocumentBytes)+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(body) > skills.DefaultMaxDocumentBytes {
		return nil, fmt.Errorf("skill document too large: %d > %d bytes", len(body), skills.DefaultMaxDocumentBytes)
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
