package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	Build       buildProvenanceJSON  `json:"build"`
	Action      string               `json:"action"`
	DryRun      bool                 `json:"dry_run"`
	RemovalMode string               `json:"removal_mode,omitempty"`
	Total       int                  `json:"total"`
	Removed     int                  `json:"removed,omitempty"`
	Failed      int                  `json:"failed,omitempty"`
	Groups      []uninstallGroupJSON `json:"groups"`
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
	mover := pickArtifactMover()
	report := uninstallReportJSON{
		Build:       newBuildProvenance(),
		Action:      "uninstalled",
		DryRun:      false,
		RemovalMode: mover.label,
		Groups:      make([]uninstallGroupJSON, 0, len(groups)),
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
			if err := mover.move(clean); err != nil {
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
		// "logs" enumerates explicit log files only. The home-dir
		// wildcard (which used to ride here because
		// config.CrashLogDir() returns GormesHome()) now lives in
		// the dedicated `gormes-home` group below — surfacing the
		// scope honestly to operators reading the preview.
		{Name: "logs", Paths: sortedExisting(config.LogPath())},
		{Name: "cron", Paths: sortedExisting(filepath.Join(home, "CRON.md"))},
		{Name: "mcp-oauth", Paths: sortedExisting(filepath.Join(home, "mcp_oauth.json"))},
		{Name: "legacy-xdg", Paths: sortedExisting(legacyXDGGormesDir())},
		{Name: "published-binary", Paths: collectPublishedBinaryPaths(home)},
		// `gormes-home` removes the entire managed home directory
		// tree — the catch-all for everything not named above
		// (skills/, subagents/, kanban.db, install.log.jsonl, the
		// managed binary at bin/gormes, crash-*.log files, etc.).
		// Listed last so the explicit per-feature groups appear
		// first in the preview; operators see the wildcard with a
		// truthful name and immediately understand the blast radius.
		{Name: "gormes-home", Paths: sortedExisting(home)},
	}
}

// collectPublishedBinaryPaths enumerates the install.sh-published PATH
// symlinks at `<bin_dir>/gormes` that point back into the gormes home.
// Without this, `gormes uninstall` removed the managed binary under
// `<home>/bin/gormes` but left the symlink at e.g. `~/.local/bin/gormes`
// dangling — the operator was greeted by a broken `gormes` command on
// the next login, and reinstalls had to step around the stale link.
//
// Discovery mirrors install.sh's pick_bin_dir() candidates:
//
//   - $GORMES_BIN_DIR (operator override; install.sh exports this)
//   - $GORMES_PREFIX/bin (compatibility prefix)
//   - $HOME/.local/bin (non-root default)
//   - /usr/local/bin (root linux default)
//
// Safety: only entries that are SYMLINKS whose target resolves into the
// gormes home are returned. A real binary at the same path (built from
// source, package-managed, manually placed) is never touched.
func collectPublishedBinaryPaths(home string) []string {
	if home == "" {
		return nil
	}
	candidates := publishedBinaryCandidates()
	homeAbs, _ := filepath.Abs(home)
	out := make([]string, 0, len(candidates))
	seen := make(map[string]bool)
	for _, path := range candidates {
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		target, err := os.Readlink(path)
		if err != nil {
			continue
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(path), target)
		}
		targetAbs, err := filepath.Abs(target)
		if err != nil {
			continue
		}
		if homeAbs != "" && (targetAbs == homeAbs || strings.HasPrefix(targetAbs, homeAbs+string(os.PathSeparator))) {
			out = append(out, path)
		}
	}
	sort.Strings(out)
	return out
}

func publishedBinaryCandidates() []string {
	const exe = "gormes"
	candidates := make([]string, 0, 4)
	if dir := strings.TrimSpace(os.Getenv("GORMES_BIN_DIR")); dir != "" {
		candidates = append(candidates, filepath.Join(dir, exe))
	}
	if prefix := strings.TrimSpace(os.Getenv("GORMES_PREFIX")); prefix != "" {
		candidates = append(candidates, filepath.Join(prefix, "bin", exe))
	}
	if userHome, err := os.UserHomeDir(); err == nil && userHome != "" {
		candidates = append(candidates, filepath.Join(userHome, ".local", "bin", exe))
	}
	candidates = append(candidates, filepath.Join("/usr", "local", "bin", exe))
	return candidates
}

// legacyXDGGormesDir returns the pre-Apr-29 runtime-state directory
// path. Commit 4cc864e80 ("fix(config): use gormes home for runtime
// state") moved memory.db, sessions/, sessions.db, gateway.pid,
// gateway_state.json, subagents/, and tools/ from
// `$XDG_DATA_HOME/gormes/` to `$GORMES_HOME`. Operators upgrading
// across that change otherwise keep the entire legacy tree forever
// because uninstall only enumerates the current home.
func legacyXDGGormesDir() string {
	xdg := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
	if xdg == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return ""
		}
		xdg = filepath.Join(home, ".local", "share")
	}
	candidate := filepath.Join(xdg, "gormes")
	// Don't double-enumerate when an operator has explicitly pointed
	// GORMES_HOME at the legacy path — that's their current home, and
	// the existing groups already cover it.
	if candidate == config.GormesHome() {
		return ""
	}
	return candidate
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
	mover := pickArtifactMover()
	if mover.label != "" {
		fmt.Fprintf(out, "removal mode: %s\n", mover.label)
	}
	for _, g := range groups {
		fmt.Fprintf(out, "removing [%s]...\n", g.Name)
		for _, p := range g.Paths {
			clean := strings.TrimSuffix(p, "/")
			if err := mover.move(clean); err != nil {
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

// artifactMover encapsulates how an uninstall removes a path. The default
// resolves to `gio trash` when available so artifacts move to the
// freedesktop trash and stay recoverable; permanent deletion via
// os.RemoveAll is the fallback for hosts that lack gio (containers, BSD,
// macOS without gvfs, etc.). Live regression 2026-05-10 made the
// recoverable default urgent: a sandbox uninstall accidentally targeting
// the operator's real ~/.gormes wiped .env (provider keys), memory.db
// (Goncho conversation history), and config.toml — recoverable from a
// May-2 trash backup only because the uninstall on that earlier date had
// used a trash-aware path. Today's permanent-delete behavior would have
// destroyed everything outright. Operators who want guaranteed permanent
// deletion can opt in via GORMES_UNINSTALL_FORCE_PURGE=1.
type artifactMover struct {
	label string
	move  func(string) error
}

func pickArtifactMover() artifactMover {
	if force := strings.TrimSpace(os.Getenv("GORMES_UNINSTALL_FORCE_PURGE")); force == "1" || strings.EqualFold(force, "true") {
		return artifactMover{
			label: "permanent delete (GORMES_UNINSTALL_FORCE_PURGE=1)",
			move:  os.RemoveAll,
		}
	}
	if gio, err := exec.LookPath("gio"); err == nil && gio != "" {
		return artifactMover{
			label: "move to freedesktop trash via gio (recoverable from your file manager's trash)",
			move:  func(p string) error { return exec.Command(gio, "trash", "--", p).Run() },
		}
	}
	return artifactMover{
		label: "permanent delete (gio not available; install glib2-tools for recoverable trash)",
		move:  os.RemoveAll,
	}
}
