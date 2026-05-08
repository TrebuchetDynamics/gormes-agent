package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func newUninstallCommand() *cobra.Command {
	var (
		dryRun         bool
		yes            bool
		keepConfig     bool
		keepCredential bool
		asJSON         bool
	)
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove Gormes artifacts from this system",
		Long:  "Enumerates every artifact Gormes wrote. Dry-run by default; use --yes to confirm.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUninstall(cmd, uninstallOptions{
				DryRun:         dryRun,
				Yes:            yes,
				KeepConfig:     keepConfig,
				KeepCredential: keepCredential,
				JSON:           asJSON,
			})
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "List artifacts without deleting")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm destructive removal")
	cmd.Flags().BoolVar(&keepConfig, "keep-config", false, "Preserve config files")
	cmd.Flags().BoolVar(&keepCredential, "keep-credentials", false, "Preserve credential pool")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: dry-run returns `{build, action: 'preview', dry_run, total, groups: [...]}`; `--yes --dry-run=false` returns `{build, action: 'uninstalled', dry_run: false, removed, failed, groups: [...]}`")
	return cmd
}

type uninstallOptions struct {
	DryRun         bool
	Yes            bool
	KeepConfig     bool
	KeepCredential bool
	JSON           bool
}

// uninstallReportJSON is the wire shape for `gormes uninstall --json`.
// Fleet automation parses this to audit cleanup across machines:
// `dry_run: true` distinguishes the preview from the apply outcome.
// `groups` lists what would be / was removed by category, so scripts
// can branch on missing categories without scraping bracketed prose.
type uninstallReportJSON struct {
	Build   buildProvenanceJSON      `json:"build"`
	Action  string                   `json:"action"`
	DryRun  bool                     `json:"dry_run"`
	Total   int                      `json:"total"`
	Removed int                      `json:"removed,omitempty"`
	Failed  int                      `json:"failed,omitempty"`
	Groups  []uninstallGroupJSON     `json:"groups"`
}

type uninstallGroupJSON struct {
	Name  string   `json:"name"`
	Paths []string `json:"paths"`
}

type artifactGroup struct {
	Name  string
	Paths []string
}

func runUninstall(cmd *cobra.Command, opts uninstallOptions) error {
	home := config.GormesHome()
	groups := collectArtifacts(home)
	if opts.KeepConfig {
		groups = removeGroup(groups, "config")
	}
	if opts.KeepCredential {
		groups = removeGroup(groups, "credentials")
	}
	if opts.DryRun || !opts.Yes {
		if opts.JSON {
			return printDryRunJSON(cmd.OutOrStdout(), groups)
		}
		return printDryRun(cmd.OutOrStdout(), groups)
	}
	if opts.JSON {
		return executeUninstallJSON(cmd.OutOrStdout(), cmd.ErrOrStderr(), groups)
	}
	return executeUninstall(cmd.OutOrStdout(), cmd.ErrOrStderr(), groups)
}

func printDryRunJSON(out io.Writer, groups []artifactGroup) error {
	total := 0
	report := uninstallReportJSON{
		Build:  newBuildProvenance(),
		Action: "preview",
		DryRun: true,
		Groups: make([]uninstallGroupJSON, 0, len(groups)),
	}
	for _, g := range groups {
		total += len(g.Paths)
		if len(g.Paths) == 0 {
			continue
		}
		report.Groups = append(report.Groups, uninstallGroupJSON{
			Name:  g.Name,
			Paths: append([]string{}, g.Paths...),
		})
	}
	report.Total = total
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(out, string(body))
	return nil
}

func executeUninstallJSON(out, errOut io.Writer, groups []artifactGroup) error {
	var removed, failed int
	report := uninstallReportJSON{
		Build:  newBuildProvenance(),
		Action: "uninstalled",
		DryRun: false,
		Groups: make([]uninstallGroupJSON, 0, len(groups)),
	}
	for _, g := range groups {
		if len(g.Paths) == 0 {
			continue
		}
		report.Groups = append(report.Groups, uninstallGroupJSON{
			Name:  g.Name,
			Paths: append([]string{}, g.Paths...),
		})
		for _, p := range g.Paths {
			clean := strings.TrimSuffix(p, "/")
			if err := os.RemoveAll(clean); err != nil {
				fmt.Fprintf(errOut, "warning: could not remove %s: %v\n", clean, err)
				failed++
				continue
			}
			removed++
		}
	}
	report.Total = removed + failed
	report.Removed = removed
	report.Failed = failed
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(out, string(body))
	if failed > 0 {
		return newExitCodeError(2, fmt.Errorf("uninstall: %d artifact(s) could not be removed (see warnings on stderr)", failed))
	}
	return nil
}

func collectArtifacts(home string) []artifactGroup {
	return []artifactGroup{
		{Name: "config", Paths: sortedExisting(config.ConfigPath(), config.YAMLConfigPath(),
			filepath.Join(home, ".env"), filepath.Join(home, "config.toml"), filepath.Join(home, "config.yaml"))},
		{Name: "credentials", Paths: sortedExisting(filepath.Join(home, "auth.json"))},
		{Name: "sessions", Paths: sortedExisting(config.SessionDBPath(), config.SessionIndexMirrorPath())},
		{Name: "gateway-state", Paths: sortedExisting(filepath.Join(home, "gateway_state.json"),
			filepath.Join(home, "gateway-locks"), filepath.Join(home, "gateway.pid"),
			filepath.Join(home, "channel_directory_sources.json"))},
		{Name: "memory", Paths: sortedExisting(config.MemoryDBPath(), filepath.Join(home, "memory"))},
		{Name: "logs", Paths: sortedExisting(config.LogPath(), config.CrashLogDir())},
		{Name: "cron", Paths: sortedExisting(filepath.Join(home, "CRON.md"))},
		{Name: "mcp-oauth", Paths: sortedExisting(filepath.Join(home, "mcp_oauth.json"))},
	}
}

func sortedExisting(paths ...string) []string {
	existing := make([]string, 0, len(paths))
	seen := make(map[string]bool)
	for _, p := range paths {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		entry := p
		if info.IsDir() {
			entry += "/"
		}
		existing = append(existing, entry)
	}
	sort.Strings(existing)
	return existing
}

func removeGroup(groups []artifactGroup, name string) []artifactGroup {
	out := make([]artifactGroup, 0, len(groups))
	for _, g := range groups {
		if g.Name != name {
			out = append(out, g)
		}
	}
	return out
}

func printDryRun(out io.Writer, groups []artifactGroup) error {
	total := 0
	for _, g := range groups {
		total += len(g.Paths)
	}
	fmt.Fprintf(out, "uninstall dry-run: %d artifact(s)\n\n", total)
	// Skip empty groups so operators only see headers for groups with
	// actual artifacts. Without this, every dry-run shows a wall of
	// empty bracketed headers from the static group manifest, drowning
	// out the groups that actually have content.
	for _, g := range groups {
		if len(g.Paths) == 0 {
			continue
		}
		fmt.Fprintf(out, "[%s]\n", g.Name)
		for _, p := range g.Paths {
			fmt.Fprintf(out, "  %s\n", p)
		}
		fmt.Fprintln(out)
	}
	if total == 0 {
		fmt.Fprintln(out, "No Gormes artifacts found.")
	}
	return nil
}

func executeUninstall(out, errOut io.Writer, groups []artifactGroup) error {
	var removed, failed int
	for _, g := range groups {
		fmt.Fprintf(out, "removing [%s]...\n", g.Name)
		for _, p := range g.Paths {
			clean := strings.TrimSuffix(p, "/")
			if err := os.RemoveAll(clean); err != nil {
				fmt.Fprintf(errOut, "warning: could not remove %s: %v\n", clean, err)
				failed++
				continue
			}
			removed++
		}
	}
	// Always surface both counts so operators see "0 failed" explicitly
	// rather than inferring success from absence of warnings. Without
	// the explicit failed count, an unrelated terminal scrollback could
	// hide warnings and an operator could believe "uninstall complete"
	// meant the whole tree was cleared.
	fmt.Fprintf(out, "\nuninstall complete: %d removed, %d failed\n", removed, failed)
	if failed > 0 {
		// Return a non-nil error so cobra exits non-zero. Fleet
		// scripts running `gormes uninstall --yes && echo OK` need
		// the exit code as a machine signal — per-file stderr
		// warnings alone don't survive into shell control flow.
		return newExitCodeError(2, fmt.Errorf("uninstall: %d artifact(s) could not be removed (see warnings on stderr)", failed))
	}
	return nil
}
